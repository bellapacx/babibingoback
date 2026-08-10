package game

import (
	"babibingo/internal/models"
	"fmt"

	"github.com/google/uuid"
)

// ClaimBingo handles a manual bingo claim (if you still want to allow this)
func (e *Engine) ClaimBingo(telegramID int64, cardID uuid.UUID) (*GameEvent, error) {
	if e.currentGame == nil || e.currentGame.Game.Status != GameStatusCalling {
		return nil, fmt.Errorf("no active game")
	}

	state := e.currentGame

	user, err := e.getUserByTelegramID(telegramID)
	if err != nil {
		return nil, err
	}

	var card models.Card
	if err := e.db.Where("id = ? AND user_id = ? AND game_id = ?",
		cardID, user.ID, state.Game.ID).First(&card).Error; err != nil {
		return nil, fmt.Errorf("card not found")
	}

	// Check if card already won
	if card.IsWinner {
		return nil, fmt.Errorf("this card already won")
	}

	pattern := checkWinPattern(card.CardData, int64SliceToInt(card.MarkedNumbers))
	if pattern == "" {
		return nil, fmt.Errorf("no winning pattern")
	}

	// ✅ Mark card as winner
	card.IsWinner = true
	e.db.Save(&card)

	// Get all winners (including this one)
	winners := e.checkAllCardsForWinners(state.Game.ID, state)
	
	if len(winners) > 0 {
		e.handleWinners(state, winners)
	}

	return &GameEvent{
		Type:      "game.winner",
		Winner: &WinnerInfo{
			UserID:     telegramID,
			Name:       user.FirstName + " " + user.LastName,
			Phone:      maskPhone(user.PhoneNumber),
			Prize:      CalculateNetPool(state.Game.TotalPool),
			CardNumber: card.CardNumber,
			Pattern:    pattern,
		},
		Pool:      CalculateNetPool(state.Game.TotalPool),
		GrossPool: state.Game.TotalPool,
		HouseCut:  CalculateHouseCut(state.Game.TotalPool),
	}, nil
}

// int64SliceToInt converts []int64 to []int
func int64SliceToInt(input []int64) []int {
	result := make([]int, len(input))
	for i, v := range input {
		result[i] = int(v)
	}
	return result
}

// JoinGame allows a user to join a game with multiple cards
func (e *Engine) JoinGame(userID int64, cardNumbers []int) (*models.Game, []models.Card, error) {
	if e.currentGame == nil {
		return nil, nil, fmt.Errorf("no active game")
	}

	state := e.currentGame
	state.mu.RLock()
	defer state.mu.RUnlock()

	if state.Game.Status != GameStatusWaiting {
		return nil, nil, fmt.Errorf("game already started")
	}

	// Check user balance
	var user models.User
	if err := e.db.First(&user, userID).Error; err != nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	totalStake := float64(len(cardNumbers)) * StakeAmount
	if user.Balance < totalStake {
		return nil, nil, fmt.Errorf("insufficient balance")
	}

	// Check if cards are available
	var existingCards []models.Card
	e.db.Where("game_id = ? AND card_number IN ?", state.Game.ID, cardNumbers).Find(&existingCards)
	if len(existingCards) > 0 {
		return nil, nil, fmt.Errorf("some cards already taken")
	}

	// Check max cards per player
	var playerCards int64
	e.db.Model(&models.Card{}).Where("game_id = ? AND user_id = ?", state.Game.ID, userID).Count(&playerCards)
	if int(playerCards)+len(cardNumbers) > MaxCardsPerPlayer {
		return nil, nil, fmt.Errorf("max %d cards per player", MaxCardsPerPlayer)
	}

	// Deduct balance
	user.Balance -= totalStake
	e.db.Save(&user)

	// Create stake transaction
	e.db.Create(&models.Transaction{
		UserID: userID,
		Type:   "stake",
		Amount: totalStake,
		Status: "completed",
		Method: "system",
	})

	// Create or update game player
	var gamePlayer models.GamePlayer
	result := e.db.Where("game_id = ? AND user_id = ?", state.Game.ID, userID).First(&gamePlayer)
	if result.Error != nil {
		gamePlayer = models.GamePlayer{
			GameID:     state.Game.ID,
			UserID:     userID,
			CardsCount: len(cardNumbers),
			TotalStake: totalStake,
		}
		e.db.Create(&gamePlayer)
	} else {
		gamePlayer.CardsCount += len(cardNumbers)
		gamePlayer.TotalStake += totalStake
		e.db.Save(&gamePlayer)
	}

	// Create cards with predefined data
	var createdCards []models.Card
	for _, cardNum := range cardNumbers {
		cardData, found := GetCardByID(cardNum)
		if !found {
			cardData = generateRandomCard(cardNum)
		}

		card := models.Card{
			GameID:     state.Game.ID,
			UserID:     userID,
			CardNumber: cardNum,
			CardData:   cardData,
		}
		e.db.Create(&card)
		createdCards = append(createdCards, card)
	}

	// Update pool
	state.Game.TotalPool += totalStake
	e.db.Save(state.Game)

	return state.Game, createdCards, nil
}