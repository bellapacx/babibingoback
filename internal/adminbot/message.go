package adminbot

import (
	"context"
	"log"
	"time"

	"github.com/mymmrac/telego"
)

func (b *Bot) sendMessage(ctx context.Context, params *telego.SendMessageParams) {
    if _, err := b.api.SendMessage(ctx, params); err != nil {
        log.Printf("Failed to send message: %v", err)
    }
}

func (b *Bot) sendText(ctx context.Context, chatID int64, text string) {
    b.sendMessage(ctx, &telego.SendMessageParams{
        ChatID: telego.ChatID{ID: chatID},
        Text:   text,
    })
}

func (b *Bot) sendMarkdown(ctx context.Context, chatID int64, text string) {
    b.sendMessage(ctx, &telego.SendMessageParams{
        ChatID:    telego.ChatID{ID: chatID},
        Text:      text,
        ParseMode: "Markdown",
    })
}

func (b *Bot) logAdminAction(ctx context.Context, adminID int64, action string, targetID int64, targetType string, details string) {
    log := AdminActionLog{
        AdminID:    adminID,
        Action:     action,
        TargetID:   targetID,
        TargetType: targetType,
        Details:    details,
        CreatedAt:  time.Now(),
    }
    b.db.Create(&log)
}