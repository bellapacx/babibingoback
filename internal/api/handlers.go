package api

import (
	"babibingo/internal/config"
	"babibingo/internal/game"
	"babibingo/internal/models"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db     *gorm.DB
	rdb    *redis.Client
	engine *game.Engine
	cfg    *config.Config
}

func NewHandler(db *gorm.DB, rdb *redis.Client, engine *game.Engine, cfg *config.Config) *Handler {
	return &Handler{db: db, rdb: rdb, engine: engine, cfg: cfg}
}
func GenerateJWT(userID int64, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString([]byte(secret))
}

func (h *Handler) AuthTelegram(c *gin.Context) {
	initData := c.GetHeader("X-Telegram-Init-Data")
	if initData == "" {
		c.JSON(400, gin.H{"error": "missing init data"})
		return
	}

	// Validate and extract user
	// For demo, we'll create/get user by initData
	// In production, properly parse the Telegram user object

	var req struct {
		TelegramID int64  `json:"telegram_id"`
		FirstName  string `json:"first_name"`
		Username   string `json:"username"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	result := h.db.Where("telegram_id = ?", req.TelegramID).First(&user)

	if result.Error != nil {
		// Create new user
		user = models.User{
			TelegramID: req.TelegramID,
			FirstName:  req.FirstName,
			Username:   req.Username,
			Balance:    0,
		}
		h.db.Create(&user)
	}

	token, err := GenerateJWT(user.ID, h.cfg.JWTSecret)
	if err != nil {
		c.JSON(500, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(200, gin.H{
		"token":   token,
		"user":    user,
		"balance": user.Balance,
	})
}

func (h *Handler) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		c.JSON(404, gin.H{"error": "user not found"})
		return
	}

	c.JSON(200, user)
}

func (h *Handler) GetCurrentGame(c *gin.Context) {
	game, players, boards, pool,grossPool, houseCut, err := h.engine.GetCurrentGame()
	if err != nil {
		c.JSON(200, gin.H{
			"status":  "waiting",
			"message": "No active game, waiting for next round",
		})
		return
	}

	c.JSON(200, gin.H{
		"id":          game.ID,
		"status":      game.Status,
		"stake":       game.StakeAmount,
		"players":     players,
		"board_count": boards,
		"pool":        pool,
		"max_players": game.MaxPlayers,
		"max_cards":   game.MaxCardsPerPlayer,
		"house_cut" : houseCut,
		"groos_pool" : grossPool,
	})
}

func (h *Handler) GetGameState(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	state, err := h.engine.GetGameState(userID.(int64))
	if err != nil {
		c.JSON(200, gin.H{
			"status":  "waiting",
			"message": "Waiting for next game",
		})
		return
	}

	c.JSON(200, state)
}

func (h *Handler) JoinGame(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		CardNumbers []int `json:"card_numbers"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	game, cards, err := h.engine.JoinGame(userID.(int64), req.CardNumbers)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, gin.H{
		"game":  game,
		"cards": cards,
	})
}

func (h *Handler) GetCards(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}
	_ = userID

	cards := game.GetAllCards() // if you have this function in cards.go
	c.JSON(200, cards)
}

func (h *Handler) GetTransactions(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	var transactions []models.Transaction
	h.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(50).Find(&transactions)

	c.JSON(200, transactions)
}

func (h *Handler) CreateDeposit(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Amount    float64 `json:"amount"`
		Method    string  `json:"method"`
		Reference string  `json:"reference"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	transaction := models.Transaction{
		UserID:    userID.(int64),
		Type:      "deposit",
		Amount:    req.Amount,
		Method:    req.Method,
		Reference: req.Reference,
		Status:    "pending",
	}

	h.db.Create(&transaction)

	c.JSON(201, transaction)
}

func (h *Handler) CreateWithdrawal(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(401, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Amount float64 `json:"amount"`
		Method string  `json:"method"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Check balance
	var user models.User
	h.db.First(&user, userID)
	if user.Balance < req.Amount {
		c.JSON(400, gin.H{"error": "insufficient balance"})
		return
	}

	transaction := models.Transaction{
		UserID: userID.(int64),
		Type:   "withdraw",
		Amount: req.Amount,
		Method: req.Method,
		Status: "pending",
	}

	h.db.Create(&transaction)

	c.JSON(201, transaction)
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
// internal/api/handlers.go

// GetBotStats returns bot statistics
func (h *Handler) GetBotStats(c *gin.Context) {
	stats := h.engine.GetBotManager().GetBotStats()
	c.JSON(http.StatusOK, stats)
}

// AddBots manually adds bots
func (h *Handler) AddBots(c *gin.Context) {
	var req struct {
		Count int `json:"count"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if req.Count <= 0 || req.Count > 50 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "count must be between 1 and 50"})
		return
	}
	
	h.engine.GetBotManager().ReserveCardsForBots(req.Count)
	
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Added %d bots", req.Count),
	})
}

// ToggleBots turns bot system on/off
func (h *Handler) ToggleBots(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	if req.Enabled {
		h.engine.StartBots()
	} else {
		h.engine.StopBots()
	}
	
	c.JSON(http.StatusOK, gin.H{
		"enabled": req.Enabled,
	})
}
// internal/api/handlers.go

// GetUserBalance returns the user's balance
func (h *Handler) GetUserBalance(c *gin.Context) {
	telegramIDStr := c.Query("telegram_id")
	if telegramIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "telegram_id is required"})
		return
	}

	telegramID, err := strconv.ParseInt(telegramIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid telegram_id"})
		return
	}

	var user models.User
	if err := h.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balance": user.Balance,
		"telegram_id": user.TelegramID,
	})
}