package agentbot

import (
	"context"
	"log"

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