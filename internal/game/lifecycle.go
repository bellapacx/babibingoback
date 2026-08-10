package game

import (
	"babibingo/internal/models"
	"log"
	"math/rand"
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Run starts the game engine ticker
func (e *Engine) Run() {
	log.Println("🔄 GAME ENGINE RUN STARTED!")
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		e.tick()
	}
}

// tick handles the game tick
func (e *Engine) tick() {
	if e.currentGame == nil {
		log.Println("🟡 No active game, starting new one...")
		e.startNewGame()
		return
	}

	state := e.currentGame
	
	// ✅ Lock the mutex for reading the state
	state.mu.RLock()
	status := state.Game.Status
	timer := int(state.Timer.Seconds())
	called := len(state.CalledNums)
	players := len(state.UserCards)
	reserved := len(state.ReservedCards)
	state.mu.RUnlock()

	// ✅ Log every tick with game status
	log.Printf("🔄 TICK: Status=%s, Timer=%d, Called=%d/75, Players=%d, Reserved=%d", 
		status, timer, called, players, reserved)

	// ✅ Handle each state WITHOUT holding the lock
	switch status {
	case GameStatusWaiting:
		e.handleWaitingState(state)
	case GameStatusCalling:
		e.handleCallingState(state)
	case GameStatusFinished:
		log.Println("📌 Game is finished, waiting for cleanup...")
	default:
		log.Printf("⚠️ Unknown game status: %s", status)
	}
}

// handleWaitingState handles the waiting/lobby state
func (e *Engine) handleWaitingState(state *GameState) {
	// ✅ Lock just to update timer
	state.mu.Lock()
	state.Timer -= 1 * time.Second
	if state.Timer < 0 {
		state.Timer = 0
	}
	timerSeconds := int(state.Timer.Seconds())
	reservedCards := len(state.ReservedCards)
	userCards := len(state.UserCards)
	totalPool := state.Game.TotalPool
	gameID := state.Game.ID
	state.mu.Unlock()

	// ✅ Log timer countdown every 5 seconds
	if timerSeconds%5 == 0 {
		log.Printf("⏱️ WAITING: Timer=%d, Players=%d, Reserved=%d, Pool=%.2f", 
			timerSeconds, userCards, reservedCards, totalPool)
	}

	// ✅ Log when timer is close to 0
	if timerSeconds <= 3 && timerSeconds > 0 {
		log.Printf("⏰ Timer at %d seconds! Reserved cards: %d", 
			timerSeconds, reservedCards)
	}

	if timerSeconds <= 0 {
		log.Printf("🚀🚀🚀 TIMER REACHED 0! Reserved cards: %d", reservedCards)
		
		if reservedCards == 0 {
			log.Println("⚠️ No cards reserved, cancelling game...")
			e.endGame(state, nil)
			return
		}
		
		log.Println("🚀 STARTING GAME - calling startCalling()...")
		e.startCalling(state)
		log.Println("✅ startCalling() completed")
		return
	}

	grossPool := totalPool
	netPool, houseCut := GetPoolBreakdown(grossPool)

	e.broadcast(GameEvent{
		Type:       "timer.tick",
		GameID:     gameID.String(),
		Status:     GameStatusWaiting,
		Timer:      timerSeconds,
		Players:    e.getPlayerCount(gameID),
		BoardCount: e.getBoardCount(gameID),
		Pool:       netPool,
		GrossPool:  grossPool,
		HouseCut:   houseCut,
		Stake:      StakeAmount,
	})
}

// handleCallingState handles the active calling state
func (e *Engine) handleCallingState(state *GameState) {
	// ✅ Lock just to update timer
	state.mu.Lock()
	state.Timer -= time.Second
	if state.Timer < 0 {
		state.Timer = 0
	}
	timerSeconds := int(state.Timer.Seconds())
	calledCount := len(state.CalledNums)
	callIndex := state.CallIndex
	userCards := len(state.UserCards)
	totalPool := state.Game.TotalPool
	gameID := state.Game.ID
	state.mu.Unlock()

	// ✅ Log calling state every tick
	// ✅ Log calling state every tick
// To:
log.Printf("🔊 CALLING: Timer=%d, Called=%d/75, CallIndex=%d, Players=%d, Pool=%.2f, GameID=%s", 
    timerSeconds, calledCount, callIndex, userCards, totalPool, gameID.String())
	
	if timerSeconds <= 0 {
		log.Printf("⏰ Calling timer reached 0! CallIndex=%d, MaxCalls=%d", callIndex, MaxCalls)
		
		if callIndex >= MaxCalls {
			log.Println("🏁 Max calls reached, ending game...")
			e.endGame(state, nil)
			return
		}
		
		log.Println("📞 Calling next number...")
		e.callNextNumber(state)
		log.Println("✅ callNextNumber() completed")
	}
}

// startNewGame creates a new game
func (e *Engine) startNewGame() {
	log.Println("🆕 Creating new game...")
	
	// Stop any existing bot routine before creating new game
	if e.botManager != nil {
		log.Println("🛑 Stopping existing bot routine...")
		e.botManager.StopBotRoutine()
		e.botManager.ResetGameBots()
	}

	game := &models.Game{
		Status:            GameStatusWaiting,
		StakeAmount:       StakeAmount,
		MaxCardsPerPlayer: MaxCardsPerPlayer,
		MaxPlayers:        MaxPlayers,
		CalledNumbers:     pq.Int64Array{},
		TotalPool:         0,
	}

	if err := e.db.Create(game).Error; err != nil {
		log.Printf("🔴 Failed to create game: %v", err)
		return
	}
	log.Printf("✅ Game created in database: %s", game.ID.String())

	e.currentGame = &GameState{
		Game:          game,
		Timer:         LobbyDuration,
		CallIndex:     0,
		CalledNums:    []int{},
		ReservedCards: make(map[int]int64),
		UserCards:     make(map[int64][]int),
	}
	log.Printf("✅ Game state initialized with timer: %v", LobbyDuration)

	// Start bots in background
	if e.botManager != nil {
		log.Println("🤖 Starting bot manager...")
		go func() {
			time.Sleep(2 * time.Second)
			initialBots := rand.Intn(6) + 5 // 5-10 bots
			log.Printf("🤖 Adding %d initial bots...", initialBots)
			e.botManager.ReserveCardsForBots(initialBots)
		}()
		go e.botManager.StartBotRoutine()
		log.Println("✅ Bot manager started")
	}

	grossPool := 0.0
	netPool, houseCut := GetPoolBreakdown(grossPool)

	e.broadcast(GameEvent{
		Type:       "game.new",
		GameID:     game.ID.String(),
		Status:     GameStatusWaiting,
		Timer:      int(LobbyDuration.Seconds()),
		Players:    0,
		BoardCount: 0,
		Pool:       netPool,
		GrossPool:  grossPool,
		HouseCut:   houseCut,
		Stake:      StakeAmount,
	})

	log.Printf("🟢 New game started: %s (Timer: %v)", game.ID.String(), LobbyDuration)
}

// endGame ends the current game
func (e *Engine) endGame(state *GameState, winner *WinnerInfo) {
	log.Printf("🏁 Ending game - Winner: %v, GameID: %s", winner != nil, state.Game.ID.String())
	
	if e.botManager != nil {
		log.Println("🛑 Stopping bot routine...")
		e.botManager.StopBotRoutine()
	}
	
	state.mu.Lock()
	state.Game.Status = GameStatusFinished
	now := time.Now()
	state.Game.EndedAt = &now
	
	if winner != nil {
		state.Game.WinnerUserID = &winner.UserID
		state.Game.WinnerPrize = winner.Prize
	}
	gameID := state.Game.ID
	totalPool := state.Game.TotalPool
	state.mu.Unlock()

	if winner != nil {
		log.Printf("💰 Winner: UserID=%d, Prize=%.2f", winner.UserID, winner.Prize)

		// Update winner balance
		e.db.Model(&models.User{}).Where("id = ?", winner.UserID).
			UpdateColumn("balance", gorm.Expr("balance + ?", winner.Prize))

		// Create win transaction
		e.db.Create(&models.Transaction{
			UserID: winner.UserID,
			Type:   "win",
			Amount: winner.Prize,
			Status: "completed",
			Method: "system",
		})

		log.Printf("💰 Winner %d won %.2f ETB", winner.UserID, winner.Prize)
	} else {
		log.Println("📌 No winner - game ended without a winner")
	}

	state.mu.Lock()
	if err := e.db.Save(state.Game).Error; err != nil {
		log.Printf("⚠️ Failed to save game: %v", err)
	} else {
		log.Printf("✅ Game saved to database")
	}
	state.mu.Unlock()

	grossPool := totalPool
	netPool, houseCut := GetPoolBreakdown(grossPool)

	e.broadcast(GameEvent{
		Type:      "game.ended",
		GameID:    gameID.String(),
		Status:    GameStatusFinished,
		Winner:    winner,
		Pool:      netPool,
		GrossPool: grossPool,
		HouseCut:  houseCut,
	})

	// Reset after delay
	go func() {
		log.Println("⏳ Waiting 10 seconds before reset...")
		time.Sleep(10 * time.Second)
		e.currentGame = nil
		log.Println("🔄 Game reset complete - ready for new game")
	}()
}