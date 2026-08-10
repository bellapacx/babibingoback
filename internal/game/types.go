package game

import (
	"babibingo/internal/models"
)

// GameEvent represents a WebSocket event
type GameEvent struct {
	Type        string      `json:"type"`
	GameID      string      `json:"game_id,omitempty"`
	Status      string      `json:"status,omitempty"`
	CallNumber  int         `json:"call_number,omitempty"`
	CallDisplay string      `json:"call_display,omitempty"`
	Called      []string    `json:"called,omitempty"`
	Players     int         `json:"players,omitempty"`
	BoardCount  int         `json:"board_count,omitempty"`
	Timer       int         `json:"timer,omitempty"`
	Winner      *WinnerInfo `json:"winner,omitempty"`
	Winners     []WinnerInfo `json:"winners,omitempty"` 
	Pool        float64     `json:"pool,omitempty"`
	GrossPool   float64     `json:"gross_pool,omitempty"`
	HouseCut    float64     `json:"house_cut,omitempty"`
	Stake       float64     `json:"stake,omitempty"`
	Message     string      `json:"message,omitempty"`
	CardNumber  int         `json:"card_number,omitempty"`
	WinningCards []models.Card  `json:"winning_cards,omitempty"`
	UserID      int64       `json:"user_id,omitempty"`
	Card        *models.Card `json:"card,omitempty"`
	Balance       float64        `json:"balance"`
}

// WinnerInfo represents winner information
type WinnerInfo struct {
	UserID      int64   `json:"user_id"`
	Name        string  `json:"name"`
	Phone       string  `json:"phone"`
	Prize       float64 `json:"prize"`
	CardNumber  int     `json:"card_number"`
	Pattern     string  `json:"pattern"`
	Card       *models.Card   `json:"card,omitempty"`
}

// Client represents a WebSocket client
type Client struct {
	ID     string
	UserID int64
	Conn   interface{} // WebSocket connection
	Send   chan []byte
}

// GameStateResponse represents the response for game state
type GameStateResponse struct {
	GameID        string         `json:"game_id"`
	Status        string         `json:"status"`
	Stake         float64        `json:"stake"`
	Timer         int            `json:"timer"`
	Players       int            `json:"players"`
	BoardCount    int            `json:"board_count"`
	Pool          float64        `json:"pool"`
	GrossPool     float64        `json:"gross_pool"`
	HouseCut      float64        `json:"house_cut"`
	Called        []string       `json:"called"`
	MyCards       []models.Card  `json:"my_cards"`
	MaxCards      int            `json:"max_cards"`
	ReservedCards []int          `json:"reserved_cards"`
	Balance       float64        `json:"balance"`
}
// models/game_player.go
