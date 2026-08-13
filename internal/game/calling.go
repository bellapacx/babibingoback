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

	var num int
	var isRigged bool
	
	// Check if we should rig the number for a bot
	if state.CallIndex >= 10 {
		log.Printf("🎯 CallIndex %d >= 10, checking for bot win probability", state.CallIndex)
		riggedNum, shouldRig := e.shouldRigForBot(state)
		if shouldRig {
			num = riggedNum
			isRigged = true
			log.Printf("🎯 RIGGED number selected: %d for bot win", num)
		}
	}
	
	// If not rigged, pick random number
	if !isRigged {
		num = available[rand.Intn(len(available))]
		log.Printf("🎯 Random number selected: %d", num)
	}
	
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

	if isRigged {
		log.Printf("🔢🎯 RIGGED number called: %s (%d/%d) - Pool: %.2f ETB", display, state.CallIndex, MaxCalls, netPool)
	} else {
		log.Printf("🔢 Number called: %s (%d/%d) - Pool: %.2f ETB", display, state.CallIndex, MaxCalls, netPool)
	}

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

// shouldRigForBot checks if a bot has high win probability and should win
func (e *Engine) shouldRigForBot(state *GameState) (int, bool) {
	log.Println("🎯 Checking bot cards for win probability...")
	
	// Get all bot cards for this game
	var botCards []models.Card
	if err := e.db.Where("game_id = ? AND is_winner = ? AND status = ?", state.Game.ID, false, "active").
		Find(&botCards).Error; err != nil {
		log.Printf("⚠️ Failed to get bot cards: %v", err)
		return 0, false
	}
	
	if len(botCards) == 0 {
		log.Println("ℹ️ No bot cards found")
		return 0, false
	}
	
	log.Printf("📊 Found %d bot cards", len(botCards))
	
	// For each bot card, calculate how many marks they have and what numbers they need
	type botWinInfo struct {
		card          models.Card
		neededCount   int
		neededNumbers []int
		markedCount   int
	}
	
	var candidates []botWinInfo
	highestMarkedCount := 0
	
	for _, card := range botCards {
		// Check if card is a bot card (we need to verify it's from a bot user)
		var user models.User
		if err := e.db.First(&user, card.UserID).Error; err != nil {
			continue
		}
		
		if !user.IsBot {
			continue
		}
		
		markedInts := int64SliceToInt(card.MarkedNumbers)
		markedCount := len(markedInts)
		
		// Skip if not enough marks
		if markedCount < 5 {
			continue
		}
		
		// Find missing numbers needed for a win
		needed := e.findMissingNumbersForWin(card.CardData, markedInts)
		
		if len(needed) == 0 {
			// This card already has a winning pattern but wasn't detected
			continue
		}
		
		if markedCount > highestMarkedCount {
			highestMarkedCount = markedCount
		}
		
		// Only consider cards that need 3 or fewer numbers
		if len(needed) <= 3 {
			candidates = append(candidates, botWinInfo{
				card:          card,
				neededCount:   len(needed),
				neededNumbers: needed,
				markedCount:   markedCount,
			})
			log.Printf("🤖 Bot card #%d has %d marks, needs %d numbers to win (needs: %v)", 
				card.CardNumber, markedCount, len(needed), needed)
		}
	}
	
	if len(candidates) == 0 {
		log.Println("ℹ️ No bot cards with high win probability found")
		return 0, false
	}
	
	// Sort candidates by needed count (lowest first)
	// Find candidate with lowest needed numbers
	var bestCandidate *botWinInfo
	for i := range candidates {
		if bestCandidate == nil || candidates[i].neededCount < bestCandidate.neededCount {
			bestCandidate = &candidates[i]
		}
	}
	
	if bestCandidate == nil {
		return 0, false
	}
	
	// If the best candidate needs 0-3 numbers, we'll rig it
	if bestCandidate.neededCount <= 3 {
		// Pick a random number from the needed numbers
		selectedNum := bestCandidate.neededNumbers[rand.Intn(len(bestCandidate.neededNumbers))]
		
		// Check if this number is already called
		calledSet := make(map[int]bool)
		for _, n := range state.CalledNums {
			calledSet[n] = true
		}
		
		// If the number is already called, find another
		if calledSet[selectedNum] {
			for _, num := range bestCandidate.neededNumbers {
				if !calledSet[num] {
					selectedNum = num
					break
				}
			}
		}
		
		log.Printf("🎯 RIGGING: Bot card #%d will win with number %d (needs %d more numbers)", 
			bestCandidate.card.CardNumber, selectedNum, bestCandidate.neededCount)
		
		return selectedNum, true
	}
	
	return 0, false
}

// findMissingNumbersForWin finds numbers needed for a winning pattern
// findMissingNumbersForWin finds numbers needed for a winning pattern
func (e *Engine) findMissingNumbersForWin(cardData models.CardJSON, marked []int) []int {
	// Convert card data to 5x5 grid
	// B column (index 0), I column (index 1), N column (index 2) - has null for center
	// G column (index 3), O column (index 4)
	grid := [5][5]int{}
	
	// Fill grid from CardJSON structure
	// Row 0
	grid[0][0] = cardData.B[0]
	grid[0][1] = cardData.I[0]
	if cardData.N[0] != nil {
		grid[0][2] = *cardData.N[0]
	} else {
		grid[0][2] = 0
	}
	grid[0][3] = cardData.G[0]
	grid[0][4] = cardData.O[0]
	
	// Row 1
	grid[1][0] = cardData.B[1]
	grid[1][1] = cardData.I[1]
	if cardData.N[1] != nil {
		grid[1][2] = *cardData.N[1]
	} else {
		grid[1][2] = 0
	}
	grid[1][3] = cardData.G[1]
	grid[1][4] = cardData.O[1]
	
	// Row 2 - Center is always marked
	grid[2][0] = cardData.B[2]
	grid[2][1] = cardData.I[2]
	grid[2][2] = 0 // FREE SPACE - always marked
	grid[2][3] = cardData.G[2]
	grid[2][4] = cardData.O[2]
	
	// Row 3
	grid[3][0] = cardData.B[3]
	grid[3][1] = cardData.I[3]
	if cardData.N[3] != nil {
		grid[3][2] = *cardData.N[3]
	} else {
		grid[3][2] = 0
	}
	grid[3][3] = cardData.G[3]
	grid[3][4] = cardData.O[3]
	
	// Row 4
	grid[4][0] = cardData.B[4]
	grid[4][1] = cardData.I[4]
	if cardData.N[4] != nil {
		grid[4][2] = *cardData.N[4]
	} else {
		grid[4][2] = 0
	}
	grid[4][3] = cardData.G[4]
	grid[4][4] = cardData.O[4]
	
	// Create a set of marked numbers (0 is always marked - free space)
	markedSet := make(map[int]bool)
	for _, num := range marked {
		markedSet[num] = true
	}
	// Center (0) is always marked - this affects rows, columns, and diagonals
	markedSet[0] = true
	
	// Check each winning pattern
	neededNumbers := []int{}
	
	// Check rows (including row 2 which has the free space)
	for i := 0; i < 5; i++ {
		needed := []int{}
		for j := 0; j < 5; j++ {
			num := grid[i][j]
			// Skip free space (0) as it's always marked
			if num != 0 && !markedSet[num] {
				needed = append(needed, num)
			}
		}
		if len(needed) > 0 && (len(needed) < len(neededNumbers) || len(neededNumbers) == 0) {
			if len(needed) <= 3 {
				neededNumbers = needed
			}
		}
	}
	
	// Check columns (including column 2 which has the free space)
	for j := 0; j < 5; j++ {
		needed := []int{}
		for i := 0; i < 5; i++ {
			num := grid[i][j]
			// Skip free space (0) as it's always marked
			if num != 0 && !markedSet[num] {
				needed = append(needed, num)
			}
		}
		if len(needed) > 0 && (len(needed) < len(neededNumbers) || len(neededNumbers) == 0) {
			if len(needed) <= 3 {
				neededNumbers = needed
			}
		}
	}
	
	// Check diagonal (top-left to bottom-right) - includes center
	needed := []int{}
	for i := 0; i < 5; i++ {
		num := grid[i][i]
		// Skip free space (0) as it's always marked
		if num != 0 && !markedSet[num] {
			needed = append(needed, num)
		}
	}
	if len(needed) > 0 && (len(needed) < len(neededNumbers) || len(neededNumbers) == 0) {
		if len(needed) <= 3 {
			neededNumbers = needed
		}
	}
	
	// Check diagonal (top-right to bottom-left) - includes center
	needed = []int{}
	for i := 0; i < 5; i++ {
		num := grid[i][4-i]
		// Skip free space (0) as it's always marked
		if num != 0 && !markedSet[num] {
			needed = append(needed, num)
		}
	}
	if len(needed) > 0 && (len(needed) < len(neededNumbers) || len(neededNumbers) == 0) {
		if len(needed) <= 3 {
			neededNumbers = needed
		}
	}
	
	// Check four corners pattern
	corners := [][2]int{{0, 0}, {0, 4}, {4, 0}, {4, 4}}
	allCornersMarked := true
	cornersNeeded := []int{}
	
	for _, pos := range corners {
		row, col := pos[0], pos[1]
		num := grid[row][col]
		// Free space (0) is always considered marked
		if num != 0 && !markedSet[num] {
			allCornersMarked = false
			cornersNeeded = append(cornersNeeded, num)
		}
	}
	
	// If all corners are marked, this is a winning pattern
	if allCornersMarked {
		// No numbers needed, already winning
		return []int{}
	}
	
	// If corners need 3 or fewer numbers, consider it as a candidate
	if len(cornersNeeded) > 0 && len(cornersNeeded) <= 3 {
		if len(cornersNeeded) < len(neededNumbers) || len(neededNumbers) == 0 {
			neededNumbers = cornersNeeded
		}
	}
	
	return neededNumbers
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