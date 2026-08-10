package game

import (
	"context"
	"fmt"
	"log"
	"time"

	"babibingo/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// getUserByTelegramID gets a user by their Telegram ID (with timeout)
func (e *Engine) getUserByTelegramID(telegramID int64) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return e.getUserByTelegramIDWithContext(ctx, telegramID)
}

// getUserByTelegramIDWithContext gets a user by their Telegram ID with context
func (e *Engine) getUserByTelegramIDWithContext(ctx context.Context, telegramID int64) (*models.User, error) {
	var user models.User
	if err := e.db.WithContext(ctx).Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with telegram_id %d not found", telegramID)
		}
		return nil, fmt.Errorf("failed to get user %d: %w", telegramID, err)
	}
	return &user, nil
}

// getUserByID gets a user by their database ID
func (e *Engine) getUserByID(userID uint) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var user models.User
	if err := e.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user with id %d not found", userID)
		}
		return nil, fmt.Errorf("failed to get user %d: %w", userID, err)
	}
	return &user, nil
}

// getUsersByTelegramIDs gets multiple users by their Telegram IDs
func (e *Engine) getUsersByTelegramIDs(telegramIDs []int64) ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var users []models.User
	if err := e.db.WithContext(ctx).Where("telegram_id IN ?", telegramIDs).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	return users, nil
}

// getPlayerCount returns the number of players in a game
func (e *Engine) getPlayerCount(gameID uuid.UUID) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	var count int64
	if err := e.db.WithContext(ctx).Model(&models.GamePlayer{}).Where("game_id = ?", gameID).Count(&count).Error; err != nil {
		log.Printf("⚠️ Failed to get player count for game %s: %v", gameID.String(), err)
		return 0
	}
	return int(count)
}

// getBoardCount returns the number of cards in a game
func (e *Engine) getBoardCount(gameID uuid.UUID) int {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	var count int64
	if err := e.db.WithContext(ctx).Model(&models.Card{}).Where("game_id = ?", gameID).Count(&count).Error; err != nil {
		log.Printf("⚠️ Failed to get board count for game %s: %v", gameID.String(), err)
		return 0
	}
	return int(count)
}

// getCalledDisplays returns formatted called numbers
func (e *Engine) getCalledDisplays(nums []int) []string {
	displays := make([]string, len(nums))
	for i, n := range nums {
		displays[i] = fmt.Sprintf("%s%d", getBingoLetter(n), n)
	}
	return displays
}

// getGameByID gets a game by its ID
func (e *Engine) getGameByID(gameID uuid.UUID) (*models.Game, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var game models.Game
	if err := e.db.WithContext(ctx).Where("id = ?", gameID).First(&game).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("game with id %s not found", gameID.String())
		}
		return nil, fmt.Errorf("failed to get game %s: %w", gameID.String(), err)
	}
	return &game, nil
}

// getActiveGames returns all active games
func (e *Engine) getActiveGames() ([]models.Game, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var games []models.Game
	if err := e.db.WithContext(ctx).Where("status IN ?", []string{GameStatusWaiting, GameStatusCalling}).Find(&games).Error; err != nil {
		return nil, fmt.Errorf("failed to get active games: %w", err)
	}
	return games, nil
}

// getFinishedGames returns finished games with optional limit
func (e *Engine) getFinishedGames(limit int) ([]models.Game, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var games []models.Game
	query := e.db.WithContext(ctx).Where("status = ?", GameStatusFinished).Order("created_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&games).Error; err != nil {
		return nil, fmt.Errorf("failed to get finished games: %w", err)
	}
	return games, nil
}

// getGamePlayers returns all players in a game
func (e *Engine) getGamePlayers(gameID uuid.UUID) ([]models.GamePlayer, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var players []models.GamePlayer
	if err := e.db.WithContext(ctx).Where("game_id = ?", gameID).Find(&players).Error; err != nil {
		return nil, fmt.Errorf("failed to get game players: %w", err)
	}
	return players, nil
}

// getGameCards returns all cards in a game
func (e *Engine) getGameCards(gameID uuid.UUID) ([]models.Card, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var cards []models.Card
	if err := e.db.WithContext(ctx).Where("game_id = ?", gameID).Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("failed to get game cards: %w", err)
	}
	return cards, nil
}

// getGameTransactions returns all transactions for a game
func (e *Engine) getGameTransactions(gameID uuid.UUID) ([]models.Transaction, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var transactions []models.Transaction
	// Join with game_players to get transactions for this game
	if err := e.db.WithContext(ctx).
		Joins("JOIN game_players ON game_players.user_id = transactions.user_id").
		Where("game_players.game_id = ?", gameID).
		Find(&transactions).Error; err != nil {
		return nil, fmt.Errorf("failed to get game transactions: %w", err)
	}
	return transactions, nil
}

// getUserCards returns all cards for a user in a game
func (e *Engine) getUserCards(gameID uuid.UUID, userID uint) ([]models.Card, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var cards []models.Card
	if err := e.db.WithContext(ctx).Where("game_id = ? AND user_id = ?", gameID, userID).Find(&cards).Error; err != nil {
		return nil, fmt.Errorf("failed to get user cards: %w", err)
	}
	return cards, nil
}

// getAgentByReferralCode finds an agent by referral code
func (e *Engine) getAgentByReferralCode(referralCode string) (*models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var user models.User
	if err := e.db.WithContext(ctx).Where("referral_code = ? AND is_agent = ?", referralCode, true).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("agent with referral code %s not found", referralCode)
		}
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}
	return &user, nil
}

// getReferredUsers returns all users referred by a specific user
func (e *Engine) getReferredUsers(userID uint) ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var users []models.User
	if err := e.db.WithContext(ctx).Where("referred_by = ?", userID).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to get referred users: %w", err)
	}
	return users, nil
}

// getReferralCount returns the number of referrals for a user
func (e *Engine) getReferralCount(userID uint) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	var count int64
	if err := e.db.WithContext(ctx).Model(&models.User{}).Where("referred_by = ?", userID).Count(&count).Error; err != nil {
		log.Printf("⚠️ Failed to get referral count for user %d: %v", userID, err)
		return 0
	}
	return count
}

// updateUserBalance updates a user's balance
func (e *Engine) updateUserBalance(userID uint, newBalance float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return e.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("balance", newBalance).Error
}

// updateUserAgentBalance updates a user's agent balance
func (e *Engine) updateUserAgentBalance(userID uint, newBalance float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return e.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Update("agent_balance", newBalance).Error
}

// getTotalStakedForGame returns total staked amount for a game
func (e *Engine) getTotalStakedForGame(gameID uuid.UUID) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	var total float64
	if err := e.db.WithContext(ctx).Model(&models.GamePlayer{}).Where("game_id = ?", gameID).Select("COALESCE(SUM(total_stake), 0)").Scan(&total).Error; err != nil {
		log.Printf("⚠️ Failed to get total staked for game %s: %v", gameID.String(), err)
		return 0
	}
	return total
}