package bot

import (
	"babibingo/internal/config"
	"context"
	"log"
	"sync"

	"github.com/mymmrac/telego"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Bot struct {
	api                *telego.Bot
	me                 *telego.User
	db                 *gorm.DB
	rdb                *redis.Client
	cfg                *config.Config
	webAppURL          string
	tempReferralCache  sync.Map // ✅ Add this field
	tempState          sync.Map
}

// New creates a new Bot instance
func New(
	token string,
	webAppURL string,
	db *gorm.DB,
	redisClient *redis.Client,
	cfg *config.Config,
) (*Bot, error) {

	api, err := telego.NewBot(token)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	me, err := api.GetMe(ctx)
	if err != nil {
		return nil, err
	}

	b := &Bot{
		api:               api,
		me:                me,
		db:                db,
		rdb:               redisClient,
		cfg:               cfg,
		webAppURL:         webAppURL,
		tempReferralCache: sync.Map{}, // ✅ Initialize the sync.Map
		tempState:         sync.Map{},
	}

	if err := b.setupCommands(ctx); err != nil {
		log.Printf("failed to set bot commands: %v", err)
	}

	log.Printf("Telegram bot started: @%s", me.Username)

	return b, nil
}