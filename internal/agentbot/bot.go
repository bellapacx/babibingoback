package agentbot

import (
	"context"
	"log"

	"github.com/mymmrac/telego"
	"gorm.io/gorm"
)

type Bot struct {
	api *telego.Bot
	me  *telego.User
	db  *gorm.DB
}

func New(token string, db *gorm.DB) (*Bot, error) {
	api, err := telego.NewBot(token)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	me, err := api.GetMe(ctx)
	if err != nil {
		return nil, err
	}

	// Auto-migrate the requests table
	if err := db.AutoMigrate(&AgentRequest{}); err != nil {
		log.Printf("Failed to migrate agent requests: %v", err)
	}

	b := &Bot{
		api: api,
		me:  me,
		db:  db,
	}

	// Set commands
	if err := b.setupCommands(ctx); err != nil {
		log.Printf("Failed to set commands: %v", err)
	}

	log.Printf("Agent Bot started: @%s", me.Username)

	return b, nil
}

func (b *Bot) setupCommands(ctx context.Context) error {
	commands := []telego.BotCommand{
		{Command: "start", Description: "Start the bot"},
		{Command: "apply", Description: "Apply to become an agent"},
		{Command: "status", Description: "Check your application status"},
	}

	err := b.api.SetMyCommands(ctx, &telego.SetMyCommandsParams{
		Commands: commands,
	})
	return err
}

func (b *Bot) Start(ctx context.Context) error {
	log.Printf("Starting agent bot @%s...", b.me.Username)

	// ✅ Correct way to start long polling
	updates, err := b.api.UpdatesViaLongPolling(ctx, nil)
	if err != nil {
		return err
	}

	for update := range updates {
		if update.Message != nil {
			b.handleMessage(ctx, update.Message)
		}
	}

	return nil
}