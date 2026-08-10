package api

import (
	"babibingo/internal/config"
	"babibingo/internal/game"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, rdb *redis.Client, engine *game.Engine) {
	cfg := config.Load()
	h := NewHandler(db, rdb, engine, cfg)

	// WebSocket
	wsHub := game.NewWSHub()
	go wsHub.Run()
	go game.SubscribeToRedis(wsHub, engine)

	r.GET("/ws", game.HandleWebSocket(wsHub, engine))

	// Public routes
	r.GET("/health", h.Health)
	r.POST("/auth/telegram", h.AuthTelegram)
	r.GET("/game/current", h.GetCurrentGame)
    
	r.GET("/user/balance", h.GetUserBalance)
	// Protected routes
	api := r.Group("/api")
	api.Use(JWTMiddleware(cfg))
	{
		api.GET("/me", h.GetMe)
		api.GET("/game/state", h.GetGameState)
		api.POST("/game/join", h.JoinGame)
		api.GET("/cards", h.GetCards)
		api.GET("/transactions", h.GetTransactions)
		api.POST("/deposits", h.CreateDeposit)
		api.POST("/withdrawals", h.CreateWithdrawal)
	}
}
