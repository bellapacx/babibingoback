package game

import (
	"babibingo/internal/models"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// startCalling starts the calling phase
func (e *Engine) startCalling(state *GameState) {
	log.Println("🔥🔥🔥 START CALLING FUNCTION ENTERED 🔥🔥🔥")
	log.Printf("📊 Game ID: %s, Players: %d, Reserved Cards: %d", 
		state.Game.ID.String(), len(state.UserCards), len(state.ReservedCards))
	
	// Collect stakes from all players
	log.Println("💰 Attempting to collect stakes...")
	if err := e.collectAllStakes(state); err != nil {
		log.Printf("❌ Failed to collect stakes: %v", err)
		e.broadcast(GameEvent{
			Type:    "game.error",
			GameID:  state.Game.ID.String(),
			Message: fmt.Sprintf("Failed to collect stakes: %v", err),
		})
		return
	}
	log.Println("✅ Stakes collected successfully!")

	log.Println("🔄 Transitioning game to CALLING state...")
	state.Game.Status = GameStatusCalling
	now := time.Now()
	state.Game.StartedAt = &now
	state.Timer = CallInterval
	log.Printf("⏱️ Timer set to: %v", CallInterval)

	if err := e.db.Save(state.Game).Error; err != nil {
		log.Printf("⚠️ Failed to save game: %v", err)
	} else {
		log.Printf("✅ Game saved with status: %s", state.Game.Status)
	}

	grossPool := state.Game.TotalPool
	netPool, houseCut := GetPoolBreakdown(grossPool)
	log.Printf("💰 Gross Pool: %.2f, Net Pool: %.2f, House Cut: %.2f", grossPool, netPool, houseCut)

	e.broadcast(GameEvent{
		Type:       "game.started",
		GameID:     state.Game.ID.String(),
		Status:     GameStatusCalling,
		Players:    e.getPlayerCount(state.Game.ID),
		BoardCount: e.getBoardCount(state.Game.ID),
		Pool:       netPool,
		GrossPool:  grossPool,
		HouseCut:   houseCut,
	})

	log.Printf("🚀 Game %s started calling! Pool: %.2f ETB", state.Game.ID.String(), grossPool)
	log.Println("🔥🔥🔥 START CALLING FUNCTION COMPLETED 🔥🔥🔥")
}

// callNextNumber calls the next random number and checks for winners
func (e *Engine) callNextNumber(state *GameState) {
	log.Printf("📞📞📞 CALL NEXT NUMBER CALLED - CallIndex: %d, Called: %d/75", state.CallIndex, len(state.CalledNums))
	
	available := e.getAvailableNumbers(state)
	log.Printf("📊 Available numbers: %d", len(available))
	
	if len(available) == 0 {
		log.Println("⚠️ No available numbers to call, ending game...")
		e.endGame(state, nil)
		return
	}

	num := available[rand.Intn(len(available))]
	log.Printf("🎯 Selected number: %d", num)
	
	state.CalledNums = append(state.CalledNums, num)
	state.CallIndex++
	log.Printf("📝 Added number %d to called list (Total: %d)", num, state.CallIndex)

	// Update database
	called := make([]int64, len(state.CalledNums))
	for i, n := range state.CalledNums {
		called[i] = int64(n)
	}
	state.Game.CalledNumbers = pq.Int64Array(called)
	state.Timer = CallInterval
	if err := e.db.Save(state.Game).Error; err != nil {
		log.Printf("⚠️ Failed to save game: %v", err)
	} else {
		log.Printf("✅ Game saved with called number %d", num)
	}

	// Broadcast the number
	letter := getBingoLetter(num)
	display := fmt.Sprintf("%s%d", letter, num)
	grossPool := state.Game.TotalPool
	netPool, houseCut := GetPoolBreakdown(grossPool)

	e.broadcast(GameEvent{
		Type:        "number.called",
		GameID:      state.Game.ID.String(),
		CallNumber:  num,
		CallDisplay: display,
		Called:      e.getCalledDisplays(state.CalledNums),
		Players:     e.getPlayerCount(state.Game.ID),
		Pool:        netPool,
		GrossPool:   grossPool,
		HouseCut:    houseCut,
	})

	log.Printf("🔢 Number called: %s (%d/%d) - Pool: %.2f ETB", display, state.CallIndex, MaxCalls, netPool)

	// Auto-mark cards
	log.Printf("🔄 Auto-marking cards for number %d...", num)
	e.autoMarkCards(state.Game.ID, num)

	// Check for winners after marking
	log.Println("🔍 Checking for winners...")
	winners := e.checkAllCardsForWinners(state.Game.ID, state)
	
	if len(winners) > 0 {
		log.Printf("🎉 Found %d winner(s)!", len(winners))
		e.handleWinners(state, winners)
	} else {
		log.Println("❌ No winners found for number", num)
	}
	
	log.Printf("✅ CALL NEXT NUMBER COMPLETED for number %d", num)
}

// getAvailableNumbers returns numbers that haven't been called
func (e *Engine) getAvailableNumbers(state *GameState) []int {
	available := make([]int, 0, 75-len(state.CalledNums))
	calledSet := make(map[int]bool)
	for _, n := range state.CalledNums {
		calledSet[n] = true
	}
	for i := 1; i <= 75; i++ {
		if !calledSet[i] {
			available = append(available, i)
		}
	}
	return available
}

// autoMarkCards automatically marks cards that have the called number
func (e *Engine) autoMarkCards(gameID uuid.UUID, number int) {
	log.Printf("🔄 Auto-marking cards for game %s, number %d", gameID.String(), number)
	
	var cards []models.Card
	if err := e.db.Where("game_id = ? AND status = ?", gameID, "active").Find(&cards).Error; err != nil {
		log.Printf("⚠️ Failed to get cards for auto-mark: %v", err)
		return
	}
	log.Printf("📊 Found %d active cards to check", len(cards))

	markedCount := 0
	for _, card := range cards {
		if containsNumber(card.CardData, number) {
			card.MarkedNumbers = append(card.MarkedNumbers, int64(number))
			if err := e.db.Save(&card).Error; err != nil {
				log.Printf("⚠️ Failed to save card %s: %v", card.ID, err)
			} else {
				markedCount++
			}
		}
	}

	if markedCount > 0 {
		log.Printf("✅ Auto-marked %d cards for number %d", markedCount, number)
	} else {
		log.Printf("ℹ️ No cards had number %d", number)
	}
}

// checkAllCardsForWinners checks all cards for winners
func (e *Engine) checkAllCardsForWinners(gameID uuid.UUID, state *GameState) []WinnerInfo {
	log.Printf("🔍🔍🔍 CHECKING FOR WINNERS - Game: %s", gameID.String())
	
	var cards []models.Card
	if err := e.db.Where("game_id = ? AND status = ? AND is_winner = ?", gameID, "active", false).Find(&cards).Error; err != nil {
		log.Printf("⚠️ Failed to get cards for winner check: %v", err)
		return nil
	}

	var winners []WinnerInfo
	log.Printf("📊 Checking %d active cards for winners", len(cards))

	for _, card := range cards {
		markedInts := int64SliceToInt(card.MarkedNumbers)
		
		// Skip cards with less than 5 marks (can't have a Bingo)
		if len(markedInts) < 5 {
			continue
		}

		// Check if card has a winning pattern
		pattern := checkWinPattern(card.CardData, markedInts)
		
		if pattern != "" {
			// Double-check with verification
			if !verifyWinDoubleCheck(card.CardData, markedInts, pattern) {
				log.Printf("❌ False positive detected for card #%d - ignoring", card.CardNumber)
				continue
			}

			// Get user details
			var user models.User
			if err := e.db.First(&user, card.UserID).Error; err != nil {
				log.Printf("⚠️ Failed to find user for card %s: %v", card.ID, err)
				continue
			}

			// Mark card as winner
			card.IsWinner = true
			if err := e.db.Save(&card).Error; err != nil {
				log.Printf("⚠️ Failed to mark card %s as winner: %v", card.ID, err)
			}

			var fullCard models.Card
			if err := e.db.Where("id = ?", card.ID).First(&fullCard).Error; err != nil {
				log.Printf("⚠️ Failed to get full card data: %v", err)
				fullCard = card
			}

			winner := WinnerInfo{
				UserID:     user.TelegramID,
				Name:       user.FirstName + " " + user.LastName,
				Phone:      maskPhone(user.PhoneNumber),
				CardNumber: card.CardNumber,
				Pattern:    pattern,
				Card:       &fullCard,
			}
			winners = append(winners, winner)

			log.Printf("🎯 Verified winner! User: %d, Card: #%d, Pattern: %s", 
				user.TelegramID, card.CardNumber, pattern)
		}
	}

	log.Printf("✅ Found %d verified winners", len(winners))
	return winners
}

// handleWinners handles single or multiple winners
func (e *Engine) handleWinners(state *GameState, winners []WinnerInfo) {
	log.Printf("🏆🏆🏆 HANDLING WINNERS - %d winners found", len(winners))
	
	if len(winners) == 0 {
		return
	}

	grossPool := state.Game.TotalPool
	totalPrize := CalculateNetPool(grossPool)
	
	// Split prize among all winners
	prizePerWinner := totalPrize / float64(len(winners))

	log.Printf("💰 Total Prize: %.2f ETB, Winners: %d, Each: %.2f ETB", 
		totalPrize, len(winners), prizePerWinner)

	// Update each winner's prize
	for i := range winners {
		winners[i].Prize = prizePerWinner
	}

	// Update game with winner info
	state.Game.Status = GameStatusFinished
	now := time.Now()
	state.Game.EndedAt = &now

	// Use first winner as the primary winner for game record
	firstWinner := &winners[0]
	state.Game.WinnerUserID = &firstWinner.UserID
	state.Game.WinnerPrize = prizePerWinner

	if err := e.db.Save(state.Game).Error; err != nil {
		log.Printf("⚠️ Failed to save game with winners: %v", err)
	} else {
		log.Printf("✅ Game saved with winner info")
	}

	// Collect all winning cards
	var winningCards []models.Card
	var winnerUserIDs []int64

	// Process each winner
	for _, winner := range winners {
		// Get user by Telegram ID
		var user models.User
		if err := e.db.Where("telegram_id = ?", winner.UserID).First(&user).Error; err != nil {
			log.Printf("⚠️ Failed to find user %d: %v", winner.UserID, err)
			continue
		}

		// Update winner balance
		if err := e.db.Model(&models.User{}).Where("id = ?", user.ID).
			UpdateColumn("balance", gorm.Expr("balance + ?", prizePerWinner)).Error; err != nil {
			log.Printf("⚠️ Failed to update balance for user %d: %v", user.ID, err)
		} else {
			log.Printf("✅ Balance updated for user %d: +%.2f ETB", user.ID, prizePerWinner)
		}

		// Create win transaction
		tx := models.Transaction{
			UserID:    user.ID,
			Type:      "win",
			Amount:    prizePerWinner,
			Status:    "completed",
			Method:    "system",
			CreatedAt: time.Now(),
		}
		if err := e.db.Create(&tx).Error; err != nil {
			log.Printf("⚠️ Failed to create transaction for user %d: %v", user.ID, err)
		} else {
			log.Printf("✅ Transaction created for user %d", user.ID)
		}

		log.Printf("💰 Winner %d (Telegram: %d) awarded %.2f ETB", 
			user.ID, winner.UserID, prizePerWinner)

		winnerUserIDs = append(winnerUserIDs, user.ID)
	}

	// Fetch all winning cards from database
	var cards []models.Card
	if err := e.db.Where("game_id = ? AND is_winner = ?", state.Game.ID, true).Find(&cards).Error; err != nil {
		log.Printf("⚠️ Failed to fetch winning cards: %v", err)
	} else {
		winningCards = cards
		log.Printf("✅ Found %d winning cards", len(winningCards))
	}

	// Broadcast winner info to all clients
	netPool, houseCut := GetPoolBreakdown(grossPool)
	
	// Send individual winner events with card data
	for i, winner := range winners {
		var cardData *models.Card
		if i < len(winningCards) {
			cardData = &winningCards[i]
		} else if winner.Card != nil {
			cardData = winner.Card
		}

		e.broadcast(GameEvent{
			Type: "game.winner",
			Winner: &WinnerInfo{
				UserID:     winner.UserID,
				Name:       winner.Name,
				Phone:      winner.Phone,
				Prize:      winner.Prize,
				CardNumber: winner.CardNumber,
				Pattern:    winner.Pattern,
				Card:       cardData,
			},
			Pool:      netPool,
			GrossPool: grossPool,
			HouseCut:  houseCut,
		})
		log.Printf("📨 Broadcast winner event for user %d", winner.UserID)
	}

	// Send a summary event with all winners and their cards
	var winnerInfos []WinnerInfo
	for i, winner := range winners {
		var cardData *models.Card
		if i < len(winningCards) {
			cardData = &winningCards[i]
		} else if winner.Card != nil {
			cardData = winner.Card
		}
		
		winnerInfos = append(winnerInfos, WinnerInfo{
			UserID:     winner.UserID,
			Name:       winner.Name,
			Phone:      winner.Phone,
			Prize:      winner.Prize,
			CardNumber: winner.CardNumber,
			Pattern:    winner.Pattern,
			Card:       cardData,
		})
	}

	e.broadcast(GameEvent{
		Type:         "game.winners_summary",
		Winners:      winnerInfos,
		WinningCards: winningCards,
		Pool:         netPool,
		GrossPool:    grossPool,
		HouseCut:     houseCut,
		Message:      fmt.Sprintf("🎉 %d winner(s)! Total Prize: %.2f ETB", len(winners), totalPrize),
	})
	log.Printf("📨 Broadcast winners summary with %d winners", len(winners))

	log.Printf("🏁 Game ended with %d winners", len(winners))
	log.Printf("🏆🏆🏆 WINNERS HANDLING COMPLETED 🏆🏆🏆")

	// Reset after delay
	go func() {
		log.Println("⏳ Waiting 10 seconds before reset...")
		time.Sleep(10 * time.Second)
		e.currentGame = nil
		log.Println("🔄 Game reset complete - ready for new game")
	}()
}

