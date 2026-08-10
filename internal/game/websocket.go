package game

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WSClient struct {
	ID     string
	UserID int64
	Conn   *websocket.Conn
	Send   chan []byte
	Hub    *WSHub
}

type WSHub struct {
	clients    map[*WSClient]bool
	register   chan *WSClient
	unregister chan *WSClient
	broadcast  chan []byte
	mu         sync.RWMutex
}

func NewWSHub() *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		broadcast:  make(chan []byte, 256),
	}
}

func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("Client registered: %s (user %d)", client.ID, client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("Client unregistered: %s", client.ID)

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					// Client not responding, remove them
					go func(c *WSClient) {
						h.unregister <- c
					}(client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *WSHub) BroadcastToUser(userID int64, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	for client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- data:
			default:
				// Client not responding, remove them
				go func(c *WSClient) {
					h.unregister <- c
				}(client)
			}
		}
	}
}

func (h *WSHub) BroadcastAll(data []byte) {
	select {
	case h.broadcast <- data:
	default:
		log.Printf("Broadcast channel full, dropping message")
	}
}

func HandleWebSocket(hub *WSHub, engine *Engine) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.Query("user_id")
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "invalid user_id"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		client := &WSClient{
			ID:     userIDStr + "_" + strconv.FormatInt(time.Now().UnixNano(), 10),
			UserID: userID,
			Conn:   conn,
			Send:   make(chan []byte, 256),
			Hub:    hub,
		}

		hub.register <- client

		// Send initial game state with recovery
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[WS] Panic in sendInitialState: %v", r)
				}
			}()
			sendInitialState(client, engine)
		}()

		go client.writePump()
		go client.readPump(hub, engine)
	}
}

func (c *WSClient) readPump(hub *WSHub, engine *Engine) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[WS] Panic in readPump: %v", r)
		}
		hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var req WSRequest
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		handleWSMessage(c, engine, req)
	}
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		if r := recover(); r != nil {
			log.Printf("[WS] Panic in writePump: %v", r)
		}
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[WS] Failed to write message: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("[WS] Failed to send ping: %v", err)
				return
			}
		}
	}
}

type WSRequest struct {
	Type        string `json:"type"`
	CardNumbers []int  `json:"card_numbers,omitempty"`
	CardNumber  int    `json:"card_number,omitempty"`
	CardID      string `json:"card_id,omitempty"`
}

func handleWSMessage(client *WSClient, engine *Engine, req WSRequest) {
	log.Printf("[WS] Handling message type: %s from user %d", req.Type, client.UserID)

	switch req.Type {
	case "card.select":
		game, cards, err := engine.JoinGame(client.UserID, req.CardNumbers)
		if err != nil {
			sendToClient(client, WSResponse{Type: "error", Message: err.Error()})
			return
		}
		sendToClient(client, WSResponse{
			Type:    "cards.selected",
			GameID:  game.ID.String(),
			Cards:   cards,
			Message: "Cards purchased successfully",
		})

	case "bingo.claim":
		sendToClient(client, WSResponse{Type: "info", Message: "Bingo claim received"})

	case "card.reserve":
		err := engine.ReserveCard(client.UserID, req.CardNumber)
		if err != nil {
			sendToClient(client, WSResponse{
				Type:    "error",
				Message: err.Error(),
			})
			return
		}

	case "card.cancel":
		log.Printf("[WS] User %d cancelling card %d", client.UserID, req.CardNumber)
		err := engine.CancelReservation(client.UserID, req.CardNumber)
		if err != nil {
			sendToClient(client, WSResponse{
				Type:    "error",
				Message: err.Error(),
			})
			return
		}

	case "game.state":
		state, err := engine.GetGameState(client.UserID)
		if err != nil {
			sendToClient(client, WSResponse{Type: "error", Message: err.Error()})
			return
		}
		sendToClient(client, WSResponse{
			Type:  "game.state",
			State: state,
		})

	default:
		log.Printf("[WS] Unknown message type: %s", req.Type)
		sendToClient(client, WSResponse{
			Type:    "error",
			Message: "Unknown message type",
		})
	}
}

type WSResponse struct {
	Type    string      `json:"type"`
	GameID  string      `json:"game_id,omitempty"`
	Cards   interface{} `json:"cards,omitempty"`
	State   interface{} `json:"state,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ✅ Fixed sendToClient with recovery
func sendToClient(client *WSClient, resp WSResponse) {
	if client == nil {
		return
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("[WS] Failed to marshal response: %v", err)
		return
	}

	// ✅ Use defer recover to prevent panic on closed channel
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[WS] Recovered from panic sending to client %s: %v", client.ID, r)
		}
	}()

	select {
	case client.Send <- data:
		// Success
	default:
		// Channel is full or closed
		log.Printf("[WS] Client %s send buffer full or channel closed", client.ID)
	}
}

// ✅ Fixed sendInitialState with recovery
func sendInitialState(client *WSClient, engine *Engine) {
	if client == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[WS] Panic in sendInitialState for client %s: %v", client.ID, r)
		}
	}()

	state, err := engine.GetGameState(client.UserID)
	if err != nil {
		// No active game
		sendToClient(client, WSResponse{
			Type: "game.state",
			State: map[string]interface{}{
				"status": "idle",
			},
			Message: "Waiting for next game",
		})
		return
	}

	sendToClient(client, WSResponse{
		Type:  "game.state",
		State: state,
	})
}

// SubscribeToRedis subscribes to Redis pub/sub for multi-instance support
func SubscribeToRedis(hub *WSHub, engine *Engine) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[Redis] Panic in SubscribeToRedis: %v", r)
		}
	}()

	pubsub := engine.SubscribeEvents()
	if pubsub == nil {
		log.Printf("[Redis] Failed to subscribe to events")
		return
	}
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		if msg != nil && len(msg.Payload) > 0 {
			hub.BroadcastAll([]byte(msg.Payload))
		}
	}
}