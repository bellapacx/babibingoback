package adminbot

import (
	"context"
	"fmt"
	"log"

	"github.com/mymmrac/telego"
)

func (b *Bot) sendAdminMenu(ctx context.Context, chatID int64) {
    msg := telego.SendMessageParams{
        ChatID: telego.ChatID{ID: chatID},
        Text: "🔐 *Admin Dashboard*\n\n" +
            "Welcome to the BabiBingo Admin Bot!\n\n" +
            "📋 *Commands:* /help for full list",
        ParseMode: "Markdown",
        ReplyMarkup: &telego.InlineKeyboardMarkup{
            InlineKeyboard: [][]telego.InlineKeyboardButton{
                {
                    {Text: "👥 Agents", CallbackData: "menu_agents"},
                    {Text: "💳 Deposits", CallbackData: "menu_deposits"},
                },
                {
                    {Text: "🏧 Withdrawals", CallbackData: "menu_withdrawals"},
                    {Text: "🎱 Games", CallbackData: "menu_games"},
                },
                {
                    {Text: "🤖 Bots", CallbackData: "menu_bots"},
                    {Text: "👤 Users", CallbackData: "menu_users"},
                },
                {
                    {Text: "📊 Stats", CallbackData: "menu_stats"},
                    {Text: "⚙️ Settings", CallbackData: "menu_settings"},
                },
            },
        },
    }

    b.sendMessage(ctx, &msg)
}

func (b *Bot) requestActionKeyboard(requestID uint) *telego.InlineKeyboardMarkup {
    return &telego.InlineKeyboardMarkup{
        InlineKeyboard: [][]telego.InlineKeyboardButton{
            {
                {
                    Text:         "✅ Approve",
                    CallbackData: fmt.Sprintf("approve_%d", requestID),
                },
                {
                    Text:         "❌ Reject",
                    CallbackData: fmt.Sprintf("reject_%d", requestID),
                },
            },
            {
                {
                    Text:         "📋 View Details",
                    CallbackData: fmt.Sprintf("view_%d", requestID),
                },
            },
        },
    }
}

func (b *Bot) transactionActionKeyboard(txID string, txType string) *telego.InlineKeyboardMarkup {
    return &telego.InlineKeyboardMarkup{
        InlineKeyboard: [][]telego.InlineKeyboardButton{
            {
                {
                    Text:         "✅ Approve",
                    CallbackData: fmt.Sprintf("tx_approve_%s_%s", txType, txID),
                },
                {
                    Text:         "❌ Reject",
                    CallbackData: fmt.Sprintf("tx_reject_%s_%s", txType, txID),
                },
            },
        },
    }
}
// sendMarkdownKeyboard - Send a markdown message with inline keyboard
func (b *Bot) sendMarkdownKeyboard(ctx context.Context, chatID int64, text string, keyboard [][]telego.InlineKeyboardButton) {
    params := &telego.SendMessageParams{
        ChatID:    telego.ChatID{ID: chatID},
        Text:      text,
        ParseMode: "Markdown",
    }
    
    if len(keyboard) > 0 {
        params.ReplyMarkup = &telego.InlineKeyboardMarkup{
            InlineKeyboard: keyboard,
        }
    }
    
    _, err := b.api.SendMessage(ctx, params)
    if err != nil {
        log.Printf("Failed to send markdown keyboard message to %d: %v", chatID, err)
    }
}