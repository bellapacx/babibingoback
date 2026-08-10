package main

import (
	"babibingo/internal/adminbot"
	"babibingo/internal/agentbot" // ✅ Import agentbot
	"babibingo/internal/api"
	"babibingo/internal/bot"
	"babibingo/internal/config"
	"babibingo/internal/game"
	"babibingo/internal/repository"
	"context"
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	cfg := config.Load()

	if err := game.LoadCardsFromJSON("data/cards.json"); err != nil {
		log.Fatalf("Failed to load cards: %v", err)
	}

	// Initialize database
	db, err := repository.InitDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize Redis
	rdb := repository.InitRedis(cfg.RedisURL)

	// Initialize game engine
	engine := game.NewEngine(db, rdb)
	go engine.Run()

	// ✅ Initialize Agent Bot
	if cfg.AgentBotToken != "" {
		agentBot, err := agentbot.New(cfg.AgentBotToken, db)
		if err != nil {
			log.Printf("Failed to initialize agent bot: %v", err)
		} else {
			go func() {
				ctx := context.Background()
				if err := agentBot.Start(ctx); err != nil {
					log.Printf("Agent bot stopped: %v", err)
				}
			}()
			log.Println("✅ Agent Bot started successfully")
		}
	} else {
		log.Println("⚠️ AGENT_BOT_TOKEN not set, agent bot disabled")
	}

	// Initialize Telegram bot (Main Bot)
	if cfg.BotToken != "" {
		telegramBot, err := bot.New(
			cfg.BotToken,
			cfg.WebAppURL,
			db,
			rdb,
			cfg,
		)
		if err != nil {
			log.Fatalf("Failed to initialize telegram bot: %v", err)
		}

		go func() {
			if err := telegramBot.Start(context.Background()); err != nil {
				log.Printf("Telegram bot stopped: %v", err)
			}
		}()
		log.Println("✅ Main Bot started successfully")
	} else {
		log.Println("⚠️ TELEGRAM_BOT_TOKEN not set, main bot disabled")
	}
	// ✅ Initialize Admin Bot
    if cfg.AdminBotToken != "" && len(cfg.GetAdminIDs()) > 0 {
        adminBot, err := adminbot.New(cfg.AdminBotToken, db, cfg, engine)
        if err != nil {
            log.Printf("Failed to initialize admin bot: %v", err)
        } else {
            go func() {
                ctx := context.Background()
                if err := adminBot.Start(ctx); err != nil {
                    log.Printf("Admin bot stopped: %v", err)
                }
            }()
            log.Printf("✅ Admin Bot started with %d admins", len(cfg.GetAdminIDs()))
        }
    } else {
        if cfg.AdminBotToken == "" {
            log.Println("⚠️ ADMIN_BOT_TOKEN not set, admin bot disabled")
        }
        if len(cfg.GetAdminIDs()) == 0 {
            log.Println("⚠️ ADMIN_IDS not set, admin bot disabled")
        }
    }

	// Setup HTTP server
	router := gin.Default()

	// CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Telegram-Init-Data"},
		AllowCredentials: true,
	}))

	// Register routes
	api.RegisterRoutes(router, db, rdb, engine)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 Server starting on port %s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}