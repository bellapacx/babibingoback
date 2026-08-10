package game

import (
	"babibingo/internal/models"
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ReserveCard reserves a card for a user
func (e *Engine) ReserveCard(telegramID int64, cardNumber int) error {
	log.Printf("🔵 ReserveCard: telegram_id=%d, card=%d", telegramID, cardNumber)

	if e.currentGame == nil {
		err := fmt.Errorf("no active game")
		e.sendError(telegramID, err.Error())
		return err
	}

	state := e.currentGame
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.Game.Status != GameStatusWaiting {
		err := fmt.Errorf("game already started")
		e.sendError(telegramID, err.Error())
		return err
	}

	// Get user by Telegram ID FIRST
	user, err := e.getUserByTelegramID(telegramID)
	if err != nil {
		errMsg := fmt.Sprintf("user not found: %v", err)
		e.sendError(telegramID, errMsg)
		return fmt.Errorf("user not found: %w", err)
	}

	// ✅ CHECK BALANCE FIRST - before any reservation
	// Calculate total stake needed (current cards + 1 new card)
	currentCards := len(state.UserCards[telegramID])
	if currentCards >= MaxCardsPerPlayer {
		err := fmt.Errorf("maximum %d cards allowed per player", MaxCardsPerPlayer)
		e.sendError(telegramID, err.Error())
		return err
	}

	totalStakeNeeded := float64(currentCards+1) * StakeAmount
	
	if user.Balance < totalStakeNeeded {
		err := fmt.Errorf("insufficient balance: need %.2f ETB for %d cards, have %.2f ETB", 
			totalStakeNeeded, currentCards+1, user.Balance)
		e.sendError(telegramID, err.Error())
		return err
	}

	// Check reservation
	if reservedBy, ok := state.ReservedCards[cardNumber]; ok {
		if reservedBy == telegramID {
			err := fmt.Errorf("card already reserved by you")
			e.sendError(telegramID, err.Error())
			return err
		}
		err := fmt.Errorf("card already reserved by another player")
		e.sendError(telegramID, err.Error())
		return err
	}

	// Reserve in memory (ONLY after all checks pass)
	state.ReservedCards[cardNumber] = telegramID
	state.UserCards[telegramID] = append(state.UserCards[telegramID], cardNumber)
	e.UpdatePool(state)

	// Get card data
	cardData, found := GetCardByID(cardNumber)
	if !found {
		// Rollback if card data not found
		e.rollbackReservationLocked(state, telegramID, cardNumber)
		err := fmt.Errorf("card data not found")
		e.sendError(telegramID, err.Error())
		return err
	}

	// Create card record
	card := models.Card{
		ID:            uuid.New(),
		GameID:        state.Game.ID,
		UserID:        user.ID,
		CardNumber:    cardNumber,
		CardData:      cardData,
		MarkedNumbers: pq.Int64Array{},
		IsWinner:      false,
		Status:        "reserved",
	}

	if err := e.db.Create(&card).Error; err != nil {
		// Rollback if database save fails
		e.rollbackReservationLocked(state, telegramID, cardNumber)
		errMsg := fmt.Sprintf("failed saving card: %v", err)
		e.sendError(telegramID, errMsg)
		return fmt.Errorf("failed saving card: %w", err)
	}

	// Broadcast success
	grossPool := state.Game.TotalPool
	netPool, houseCut := GetPoolBreakdown(grossPool)

	e.broadcast(GameEvent{
		Type:       "card.reserved",
		GameID:     state.Game.ID.String(),
		CardNumber: cardNumber,
		UserID:     telegramID,
		Card:       &card,
		Players:    len(state.UserCards),
		Pool:       netPool,
		GrossPool:  grossPool,
		HouseCut:   houseCut,
		Stake:      StakeAmount,
		Message:    fmt.Sprintf("Card #%d reserved! Prize Pool: %.2f ETB", cardNumber, netPool),
	})

	// ✅ Send balance update to the user (balance hasn't changed yet, but confirms available balance)
	e.sendBalanceUpdate(telegramID, user.Balance)

	log.Printf("🟢 Card %d reserved for user %d", cardNumber, telegramID)
	return nil
}

// ✅ rollbackReservationLocked - Internal rollback (assumes lock is already held)
func (e *Engine) rollbackReservationLocked(state *GameState, telegramID int64, cardNumber int) {
	log.Printf("🔴 Rolling back reservation for user %d, card %d", telegramID, cardNumber)
	delete(state.ReservedCards, cardNumber)
	userCards := state.UserCards[telegramID]
	for i, num := range userCards {
		if num == cardNumber {
			state.UserCards[telegramID] = append(userCards[:i], userCards[i+1:]...)
			break
		}
	}
	e.UpdatePool(state)
}

// CancelReservation cancels a card reservation
func (e *Engine) CancelReservation(telegramID int64, cardNumber int) error {
	if e.currentGame == nil {
		return fmt.Errorf("no active game")
	}

	state := e.currentGame
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.Game.Status != GameStatusWaiting {
		return fmt.Errorf("game already started - cannot cancel")
	}

	reservedBy, ok := state.ReservedCards[cardNumber]
	if !ok || reservedBy != telegramID {
		return fmt.Errorf("card not reserved by you")
	}

	user, err := e.getUserByTelegramID(telegramID)
	if err != nil {
		return err
	}

	// Remove from memory
	delete(state.ReservedCards, cardNumber)
	userCards := state.UserCards[telegramID]
	for i, num := range userCards {
		if num == cardNumber {
			state.UserCards[telegramID] = append(userCards[:i], userCards[i+1:]...)
			break
		}
	}

	// Delete from database
	if err := e.db.Where("game_id = ? AND card_number = ? AND user_id = ?",
		state.Game.ID, cardNumber, user.ID).Delete(&models.Card{}).Error; err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}

	e.UpdatePool(state)
	grossPool := state.Game.TotalPool
	netPool, houseCut := GetPoolBreakdown(grossPool)

	e.broadcast(GameEvent{
		Type:       "card.cancelled",
		GameID:     state.Game.ID.String(),
		CardNumber: cardNumber,
		UserID:     telegramID,
		Pool:       netPool,
		GrossPool:  grossPool,
		HouseCut:   houseCut,
		Message:    fmt.Sprintf("Card #%d cancelled. Prize Pool: %.2f ETB", cardNumber, netPool),
	})

	// ✅ Send balance update to the user
	e.sendBalanceUpdate(telegramID, user.Balance)

	return nil
}

// ✅ sendError - Send error event to specific user (no deadlock)
func (e *Engine) sendError(telegramID int64, message string) {
	log.Printf("🔴 Sending error to user %d: %s", telegramID, message)

	// Create the error event
	event := GameEvent{
		Type:    "error",
		Message: message,
		UserID:  telegramID,
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("⚠️ Failed to marshal error event: %v", err)
		return
	}

	// Send to specific user
	e.sendToUser(telegramID, data)
}

// ✅ sendToUser - Send data to a specific user (no deadlock)
func (e *Engine) sendToUser(telegramID int64, data []byte) {
	// ✅ Use the engine's client map with read lock
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, client := range e.clients {
		if client.UserID == telegramID {
			select {
			case client.Send <- data:
				log.Printf("✅ Message sent directly to user %d", telegramID)
			default:
				log.Printf("⚠️ Client send buffer full for user %d", telegramID)
			}
			return
		}
	}

	// Fallback: broadcast if user not found in clients
	log.Printf("⚠️ User %d not found in clients, broadcasting as fallback", telegramID)
	
	// Since we have the data, broadcast it
	var event GameEvent
	if err := json.Unmarshal(data, &event); err == nil {
		e.broadcast(event)
	}
}

// ✅ sendBalanceUpdate - Send balance update to a specific user
func (e *Engine) sendBalanceUpdate(telegramID int64, balance float64) {
	event := GameEvent{
		Type:    "balance.update",
		UserID:  telegramID,
		Balance: balance,
	}

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("⚠️ Failed to marshal balance update: %v", err)
		return
	}

	e.sendToUser(telegramID, data)
}