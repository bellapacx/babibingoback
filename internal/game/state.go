package game

import (
	"sync"
	"time"

	"babibingo/internal/models"
)

// GameState represents the current state of a game
type GameState struct {
	Game          *models.Game
	Timer         time.Duration
	CallIndex     int
	CalledNums    []int
	ReservedCards map[int]int64   // card_number -> telegram_id
	UserCards     map[int64][]int // telegram_id -> []card_numbers
	mu            sync.RWMutex
}

// GetReservedCardCount returns the number of reserved cards
func (gs *GameState) GetReservedCardCount() int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return len(gs.ReservedCards)
}

// GetPlayerCount returns the number of players
func (gs *GameState) GetPlayerCount() int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return len(gs.UserCards)
}

// GetUserCards returns cards for a specific user
func (gs *GameState) GetUserCards(telegramID int64) []int {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.UserCards[telegramID]
}

// IsCardReserved checks if a card is reserved
func (gs *GameState) IsCardReserved(cardNumber int) (int64, bool) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	userID, ok := gs.ReservedCards[cardNumber]
	return userID, ok
}