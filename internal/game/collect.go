package game

import (
	"context"
	"fmt"
	"log"
	"time"

	"babibingo/internal/models"
)

// collectAllStakes collects stakes from all players with reservations (including bots)
// OPTIMIZED: Batch operations, reduced queries, and parallel processing
// BOTS: No balance deduction, but their stakes contribute to the pool
// HOUSE CUT: Applied to the total pool
// INSUFFICIENT BALANCE: Players with insufficient balance are removed from the game
func (e *Engine) collectAllStakes(state *GameState) error {
	log.Println("💰💰💰 COLLECT ALL STAKES STARTED 💰💰💰")
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Lock state and copy data
	state.mu.RLock()
	telegramIDs := make(map[int64]bool)
	// ✅ ReservedCards is map[int]int64, need to iterate correctly
	for cardIndex, telegramID := range state.ReservedCards {
		telegramIDs[telegramID] = true
		_ = cardIndex // Mark as used
	}
	userCardsCopy := make(map[int64][]int)
	for k, v := range state.UserCards {
		userCardsCopy[k] = v
	}
	state.mu.RUnlock()

	if len(telegramIDs) == 0 {
		log.Println("⚠️ No players with reservations, cancelling game...")
		return fmt.Errorf("no players to collect stakes from")
	}

	log.Printf("📊 Collecting stakes from %d players", len(telegramIDs))

	// Batch fetch users
	var telegramIDList []int64
	for id := range telegramIDs {
		telegramIDList = append(telegramIDList, id)
	}

	var users []models.User
	if err := e.db.WithContext(ctx).Where("telegram_id IN ?", telegramIDList).Find(&users).Error; err != nil {
		return fmt.Errorf("failed to fetch users: %w", err)
	}

	userMap := make(map[int64]*models.User)
	for i := range users {
		userMap[users[i].TelegramID] = &users[i]
	}

	// Start transaction
	tx := e.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to begin transaction: %w", tx.Error)
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("🔴 Panic recovered in collectAllStakes: %v", r)
		}
	}()

	type PlayerData struct {
		User       *models.User
		CardCount  int
		TotalStake float64
		IsBot      bool
		TelegramID int64
	}

	var realPlayers []PlayerData
	var botPlayers []PlayerData
	var realPool float64
	var botPool float64
	totalPool := 0.0
	commissionEarned := make(map[int64]float64)
	
	// ✅ Track removed players
	var removedPlayers []int64
	var removedCardCount int

	// Process players - check balances first
	for telegramID := range telegramIDs {
		user, exists := userMap[telegramID]
		if !exists {
			log.Printf("⚠️ User %d not found in database, skipping", telegramID)
			removedPlayers = append(removedPlayers, telegramID)
			removedCardCount += len(userCardsCopy[telegramID])
			continue
		}
		
		cardCount := len(userCardsCopy[telegramID])
		if cardCount == 0 {
			continue
		}
		totalStake := float64(cardCount) * StakeAmount

		playerData := PlayerData{
			User:       user,
			CardCount:  cardCount,
			TotalStake: totalStake,
			IsBot:      user.IsBot,
			TelegramID: telegramID,
		}

		if user.IsBot {
			botPlayers = append(botPlayers, playerData)
			botPool += totalStake
		} else {
			// ✅ Check balance for real players
			if user.Balance < totalStake {
				log.Printf("⚠️ User %d has insufficient balance (needs %.2f, has %.2f), removing from game", 
					telegramID, totalStake, user.Balance)
				removedPlayers = append(removedPlayers, telegramID)
				removedCardCount += cardCount
				continue
			}
			realPlayers = append(realPlayers, playerData)
			realPool += totalStake
		}
	}

	// ✅ Remove players with insufficient balance from reserved cards
	if len(removedPlayers) > 0 {
		log.Printf("🗑️ Removing %d players with insufficient balance (%d cards total)", 
			len(removedPlayers), removedCardCount)
		
		state.mu.Lock()
		// ✅ Create new ReservedCards map without removed players
		newReservedCards := make(map[int]int64)
		for cardIndex, telegramID := range state.ReservedCards {
			removed := false
			for _, removedID := range removedPlayers {
				if telegramID == removedID {
					removed = true
					break
				}
			}
			if !removed {
				newReservedCards[cardIndex] = telegramID
			}
		}
		state.ReservedCards = newReservedCards
		
		// Remove from UserCards
		for _, telegramID := range removedPlayers {
			delete(state.UserCards, telegramID)
		}
		state.mu.Unlock()
		
		// ✅ Broadcast removal to players
		for _, telegramID := range removedPlayers {
			e.broadcast(GameEvent{
				Type:      "player.removed",
				GameID:    state.Game.ID.String(),
				UserID:    telegramID,
				Message:   fmt.Sprintf("⚠️ Removed from game due to insufficient balance"),
			})
		}
	}

	// Check if we have any real players left
	if len(realPlayers) == 0 && len(botPlayers) == 0 {
		tx.Rollback()
		return fmt.Errorf("no valid players remaining after removing insufficient balances")
	}

	log.Printf("📊 Found %d real players and %d bot players (removed %d players)", 
		len(realPlayers), len(botPlayers), len(removedPlayers))

	// ✅ Process bots - NO BALANCE DEDUCTION, but they contribute to the pool
	if len(botPlayers) > 0 {
		log.Printf("🤖 Processing %d bot players (no balance deduction, pool contribution: %.2f ETB)...", 
			len(botPlayers), botPool)
		
		var botTransactions []models.Transaction
		var botGamePlayers []models.GamePlayer
		var botUserIDs []uint

		for _, bp := range botPlayers {
			totalPool += bp.TotalStake
			botUserIDs = append(botUserIDs, uint(bp.User.ID))

			// Prepare bot transactions (amount is 0 since no balance deduction)
			for _, cardNumber := range userCardsCopy[bp.TelegramID] {
				_ = cardNumber
				reference := fmt.Sprintf("bot_stake_%s_%d_%d",
					state.Game.ID.String()[:8],
					bp.User.ID,
					time.Now().UnixNano(),
				)
				botTransactions = append(botTransactions, models.Transaction{
					UserID:      bp.User.ID,
					Type:        "stake",
					Amount:      0,
					Status:      "completed",
					Method:      "system",
					Reference:   reference,
					Description: fmt.Sprintf("Bot card for game %s (free)", state.Game.ID.String()[:8]),
					CreatedAt:   time.Now(),
				})
			}

			// Prepare bot game players
			botGamePlayers = append(botGamePlayers, models.GamePlayer{
				GameID:     state.Game.ID,
				UserID:     bp.User.ID,
				CardsCount: bp.CardCount,
				TotalStake: bp.TotalStake,
				IsBot:      true,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			})
		}

		// Bulk insert bot transactions
		if len(botTransactions) > 0 {
			if err := tx.CreateInBatches(botTransactions, 500).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to create bot transactions: %w", err)
			}
			log.Printf("  ✅ Created %d bot transactions (amount: 0)", len(botTransactions))
		}

		// Bulk upsert bot game players
		if len(botGamePlayers) > 0 {
			for _, gp := range botGamePlayers {
				if err := tx.Save(&gp).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to save bot game player: %w", err)
				}
			}
			log.Printf("  ✅ Created %d bot game players", len(botGamePlayers))
		}

		// Bulk update bot card statuses
		if len(botUserIDs) > 0 {
			if err := tx.Model(&models.Card{}).
				Where("game_id = ? AND user_id IN ?", state.Game.ID, botUserIDs).
				Update("status", "active").Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to update bot card status: %w", err)
			}
			log.Printf("  ✅ Updated card status for %d bot players", len(botUserIDs))
		}
	}

	// ✅ Process real players - DEDUCT BALANCE
	if len(realPlayers) > 0 {
		log.Printf("👤 Processing %d real players (deducting balance)...", len(realPlayers))

		var userUpdates []models.User
		var transactions []models.Transaction
		var gamePlayers []models.GamePlayer
		var commissionTransactions []models.Transaction
		var realUserIDs []uint

		for _, rp := range realPlayers {
			totalPool += rp.TotalStake
			realUserIDs = append(realUserIDs, uint(rp.User.ID))

			// Deduct balance from real players only
			rp.User.Balance -= rp.TotalStake
			userUpdates = append(userUpdates, *rp.User)

			// Prepare transactions
			for _, cardNumber := range userCardsCopy[rp.TelegramID] {
				_ = cardNumber
				reference := fmt.Sprintf("stake_%s_%d_%d",
					state.Game.ID.String()[:8],
					rp.User.ID,
					time.Now().UnixNano(),
				)
				transactions = append(transactions, models.Transaction{
					UserID:      rp.User.ID,
					Type:        "stake",
					Amount:      StakeAmount,
					Status:      "completed",
					Method:      "system",
					Reference:   reference,
					Description: fmt.Sprintf("Card for game %s", state.Game.ID.String()[:8]),
					CreatedAt:   time.Now(),
				})
			}

			// Prepare game players
			gamePlayers = append(gamePlayers, models.GamePlayer{
				GameID:     state.Game.ID,
				UserID:     rp.User.ID,
				CardsCount: rp.CardCount,
				TotalStake: rp.TotalStake,
				IsBot:      false,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			})

			// Agent Commission (only for real players)
			if rp.User.ReferredBy != nil {
				var agent models.User
				if err := tx.Where("id = ? AND is_agent = ?", *rp.User.ReferredBy, true).First(&agent).Error; err == nil {
					if !agent.IsBot {
						commission := float64(rp.CardCount) * 1.0
						agent.AgentBalance += commission
						agent.Balance += commission
						commissionEarned[agent.TelegramID] = commissionEarned[agent.TelegramID] + commission
						
						if err := tx.Save(&agent).Error; err != nil {
							log.Printf("⚠️ Failed to update agent balance: %v", err)
						} else {
							commissionTransactions = append(commissionTransactions, models.Transaction{
								UserID:      agent.ID,
								Type:        "agent_commission",
								Amount:      commission,
								Status:      "completed",
								Method:      "system",
								Reference:   fmt.Sprintf("comm_%d_%d_%d", agent.ID, rp.User.ID, time.Now().UnixNano()),
								Description: fmt.Sprintf("Commission from user %d", rp.User.ID),
								CreatedAt:   time.Now(),
							})
						}
					}
				}
			}
		}

		// Bulk update user balances
		if len(userUpdates) > 0 {
			for _, user := range userUpdates {
				if err := tx.Save(&user).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to update user balance: %w", err)
				}
			}
			log.Printf("  ✅ Updated balances for %d real users", len(userUpdates))
		}

		// Bulk insert transactions
		if len(transactions) > 0 {
			if err := tx.CreateInBatches(transactions, 500).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to create transactions: %w", err)
			}
			log.Printf("  ✅ Created %d transactions", len(transactions))
		}

		// Bulk upsert game players
		if len(gamePlayers) > 0 {
			for _, gp := range gamePlayers {
				if err := tx.Save(&gp).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("failed to save game player: %w", err)
				}
			}
			log.Printf("  ✅ Created %d game players", len(gamePlayers))
		}

		// Bulk insert commission transactions
		if len(commissionTransactions) > 0 {
			if err := tx.CreateInBatches(commissionTransactions, 500).Error; err != nil {
				log.Printf("⚠️ Failed to create commission transactions: %v", err)
			}
			log.Printf("  ✅ Created %d commission transactions", len(commissionTransactions))
		}

		// Bulk update real player card statuses
		if len(realUserIDs) > 0 {
			if err := tx.Model(&models.Card{}).
				Where("game_id = ? AND user_id IN ?", state.Game.ID, realUserIDs).
				Update("status", "active").Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to update card status: %w", err)
			}
			log.Printf("  ✅ Updated card status for %d real players", len(realUserIDs))
		}
	}

	// ✅ Calculate final pool breakdown
	grossPool := totalPool
	netPool, houseCut := GetPoolBreakdown(grossPool)

	// ✅ Calculate total players (real + bots)
	totalPlayers := len(realPlayers) + len(botPlayers)

	log.Printf("📊 Game has %d real players and %d bot players (removed %d)", 
		len(realPlayers), len(botPlayers), len(removedPlayers))
	log.Printf("💰 Gross Pool: %.2f ETB (Real: %.2f, Bots: %.2f,TotalP: %2.d)", grossPool, realPool, botPool, totalPlayers)
	log.Printf("🏠 House Cut: %.2f ETB (%.0f%%)", houseCut, HouseCutPercent*100)
	log.Printf("🎯 Net Pool (for winners): %.2f ETB", netPool)

	// Update game with gross pool
	state.mu.Lock()
	state.Game.TotalPool = grossPool
	// ✅ Remove PlayerCount if it doesn't exist in the model
	// state.Game.PlayerCount = totalPlayers // Commented out
	state.mu.Unlock()

	if err := tx.Save(state.Game).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update game pool: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Broadcast pool update with house cut info
	e.broadcast(GameEvent{
		Type:      "pool.update",
		GameID:    state.Game.ID.String(),
		Pool:      netPool,
		GrossPool: grossPool,
		HouseCut:  houseCut,
		Message:   fmt.Sprintf("💰 Total pool: %.2f ETB (House: %.2f ETB)", netPool, houseCut),
	})

	log.Printf("✅ Collected total pool: Gross: %.2f ETB, Net: %.2f ETB, House: %.2f ETB", 
		grossPool, netPool, houseCut)
	log.Println("💰💰💰 COLLECT ALL STAKES COMPLETED SUCCESSFULLY 💰💰💰")
	return nil
}