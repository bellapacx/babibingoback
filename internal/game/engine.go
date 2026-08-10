package game

import (
	"babibingo/internal/models"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	GameStatusWaiting   = "waiting"
	GameStatusCalling   = "calling"
	GameStatusFinished  = "finished"
	GameStatusCancelled = "cancelled"

	LobbyDuration      = 60 * time.Second
	CallInterval       = 5 * time.Second
	MaxCalls           = 75
	StakeAmount        = 20.0
	MaxCardsPerPlayer  = 4
	MaxPlayers         = 400
	HouseCutPercent    = 0.20 // 10% house cut
)

// Engine is the main game engine
type Engine struct {
	db          *gorm.DB
	rdb         *redis.Client
	clients     map[string]*Client
	mu          sync.RWMutex
	currentGame *GameState
	botManager  *BotManager // ✅ NEW
}

// NewEngine creates a new game engine
func NewEngine(db *gorm.DB, rdb *redis.Client) *Engine {
	InitCardCache()
	
	if err := db.AutoMigrate(&models.RobotBotSettings{}); err != nil {
		log.Printf("Failed to migrate BotSettings: %v", err)
	}

	engine := &Engine{
		db:      db,
		rdb:     rdb,
		clients: make(map[string]*Client),
	}
	
	// ✅ Initialize bot manager
	engine.botManager = NewBotManager(engine)
	
	return engine
}
// GetBotManager returns the bot manager
func (e *Engine) GetBotManager() *BotManager {
	return e.botManager
}

// StartBots starts the bot system
func (e *Engine) StartBots() {
	e.botManager.StartBotRoutine()
}

// StopBots stops the bot system
func (e *Engine) StopBots() {
	e.botManager.StopBotRoutine()
}
// GetCurrentGame returns the current game state
func (e *Engine) GetCurrentGame() (*models.Game, int, int, float64, float64, float64, error) {
	state := e.GetCurrentGameState()

	if state == nil {
		return nil,0,0,0,0,0,fmt.Errorf("no active game")
	}

	
	state.mu.RLock()
	defer state.mu.RUnlock()

	players := e.getPlayerCount(state.Game.ID)
	boards := e.getBoardCount(state.Game.ID)

	grossPool := state.Game.TotalPool
	netPool, houseCut := GetPoolBreakdown(grossPool)

	return state.Game, players, boards, grossPool, netPool, houseCut, nil
}

// GetGameStatus returns the current game status
func (e *Engine) GetGameStatus() (string, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.currentGame == nil {
		return "", fmt.Errorf("no active game")
	}
	return e.currentGame.Game.Status, nil
}

// GetGameState returns the game state for a user
func (e *Engine) GetGameState(userID int64) (*GameStateResponse, error) {
	state := e.GetCurrentGameState()

if state == nil {
	return nil, fmt.Errorf("no active game")
}

	
	state.mu.RLock()
	defer state.mu.RUnlock()

	// Get user by Telegram ID
	user, err := e.getUserByTelegramID(userID)
	if err != nil {
		return nil, err
	}

	var myCards []models.Card
	e.db.Where("game_id = ? AND user_id = ?", state.Game.ID, user.ID).Find(&myCards)

	calledDisplays := make([]string, 0, len(state.CalledNums))
	for _, n := range state.CalledNums {
		calledDisplays = append(calledDisplays, fmt.Sprintf("%s%d", getBingoLetter(n), n))
	}

	reservedCards := make([]int, 0, len(state.ReservedCards))
	for card := range state.ReservedCards {
		reservedCards = append(reservedCards, card)
	}

	grossPool := state.Game.TotalPool
	netPool, houseCut := GetPoolBreakdown(grossPool)

	return &GameStateResponse{
		GameID:        state.Game.ID.String(),
		Status:        state.Game.Status,
		Stake:         StakeAmount,
		Timer:         int(state.Timer.Seconds()),
		Players:       e.getPlayerCount(state.Game.ID),
		BoardCount:    e.getBoardCount(state.Game.ID),
		Pool:          netPool,
		GrossPool:     grossPool,
		HouseCut:      houseCut,
		Called:        calledDisplays,
		MyCards:       myCards,
		MaxCards:      MaxCardsPerPlayer,
		ReservedCards: reservedCards,
		Balance:       user.Balance,
	}, nil 
}
// GetGameStats returns game statistics for admin dashboard
func (e *Engine) GetGameStats() map[string]interface{} {
    e.mu.RLock()
    defer e.mu.RUnlock()

    stats := make(map[string]interface{})
    
   state := e.GetCurrentGameState()

if state == nil {
	stats["has_active_game"] = false
	return stats
}

    
    stats["has_active_game"] = true
    stats["game_id"] = state.Game.ID.String()
    stats["status"] = state.Game.Status
    stats["total_pool"] = state.Game.TotalPool
    stats["players"] = len(state.UserCards)
    stats["reserved_cards"] = len(state.ReservedCards)
    stats["called_numbers"] = len(state.CalledNums)
    stats["timer"] = int(state.Timer.Seconds())
    stats["call_index"] = state.CallIndex

    // Get bot count in game
    botCount := 0
    for _, userID := range state.ReservedCards {
        var user models.User
        if err := e.db.Where("telegram_id = ?", userID).First(&user).Error; err == nil {
            if user.IsBot {
                botCount++
            }
        }
    }
    stats["bot_count"] = botCount

    return stats
}

// GetActiveGamesCount returns number of active games
func (e *Engine) GetActiveGamesCount() int64 {
    var count int64
    e.db.Model(&models.Game{}).Where("status IN (?)", []string{GameStatusWaiting, GameStatusCalling}).Count(&count)
    return count
}

// GetTotalGamesCount returns total games played
func (e *Engine) GetTotalGamesCount() int64 {
    var count int64
    e.db.Model(&models.Game{}).Count(&count)
    return count
}

// GetTotalPoolAllGames returns total pool from all finished games
func (e *Engine) GetTotalPoolAllGames() float64 {
    var total float64
    e.db.Model(&models.Game{}).Where("status = ?", GameStatusFinished).Select("COALESCE(SUM(total_pool), 0)").Scan(&total)
    return total
}
// In engine.go - Add this helper to send error events to frontend

// sendError sends an error event to the frontend
// GetCurrentGameState safely returns current game
func (e *Engine) GetCurrentGameState() *GameState {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.currentGame
}
// SetCurrentGame safely updates current game
func (e *Engine) SetCurrentGame(state *GameState) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.currentGame = state
}