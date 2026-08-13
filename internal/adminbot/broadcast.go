package adminbot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"babibingo/internal/models"

	"github.com/mymmrac/telego"
)

// handleBroadcast - Start broadcast flow
func (b *Bot) handleBroadcast(ctx context.Context, chatID int64, args []string) {
	if len(args) > 0 && args[0] == "cancel" {
		b.tempState.Delete(chatID)
		b.sendText(ctx, chatID, "❌ Broadcast cancelled.")
		return
	}

	// Check if there's a pending broadcast
	if _, ok := b.tempState.Load(chatID); ok {
		b.sendText(ctx, chatID, "⚠️ You already have a pending broadcast.\nType `/broadcast cancel` to cancel it.")
		return
	}

	// Ask for the message
	b.sendMarkdown(ctx, chatID,
		"📢 *Send Broadcast*\n\n"+
			"Please type the message you want to send to ALL users.\n\n"+
			"📌 You can use emojis and formatting.\n\n"+
			"⌨️ Type your message below:\n"+
			"• Type `/broadcast cancel` to cancel")
	
	b.tempState.Store(chatID, "awaiting_broadcast_message")
}

// handleBroadcastMessageInput - Handle the message input from admin
func (b *Bot) handleBroadcastMessageInput(ctx context.Context, chatID int64, text string) {
	// Check if it's a command
	if strings.HasPrefix(text, "/") {
		if text == "/broadcast cancel" {
			b.tempState.Delete(chatID)
			b.sendText(ctx, chatID, "❌ Broadcast cancelled.")
			return
		}
		b.sendText(ctx, chatID, "❌ Invalid input. Please type a message or `/broadcast cancel`.")
		return
	}

	// Store the message
	message := text
	b.tempState.Store(chatID, fmt.Sprintf("broadcast_pending_%s", message))

	// Show preview with buttons
	b.showBroadcastPreview(ctx, chatID, message)
}

// showBroadcastPreview - Show broadcast preview with buttons
func (b *Bot) showBroadcastPreview(ctx context.Context, chatID int64, message string) {
	totalUsers := b.getTotalUsers()
	
	text := fmt.Sprintf(
		"📢 *Broadcast Preview*\n\n"+
			"📝 Message:\n%s\n\n"+
			"📊 Total Users: %d\n\n"+
			"⚠️ This will be sent to ALL main bot users.\n\n"+
			"Choose an action below:",
		escapeMarkdown(message),
		totalUsers,
	)

	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "✅ Confirm & Send", CallbackData: "broadcast_confirm"},
			{Text: "🧪 Send Test", CallbackData: "broadcast_test"},
		},
		{
			{Text: "✏️ Edit Message", CallbackData: "broadcast_edit"},
			{Text: "❌ Cancel", CallbackData: "broadcast_cancel"},
		},
	}

	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// handleBroadcastConfirm - Handle broadcast confirmation
func (b *Bot) handleBroadcastConfirm(ctx context.Context, chatID int64) {
	state, ok := b.tempState.Load(chatID)
	if !ok {
		b.sendText(ctx, chatID, "❌ No pending broadcast found. Use `/broadcast` first.")
		return
	}

	stateStr := state.(string)
	if !strings.HasPrefix(stateStr, "broadcast_pending_") {
		b.sendText(ctx, chatID, "❌ Invalid state. Use `/broadcast` first.")
		return
	}

	message := strings.TrimPrefix(stateStr, "broadcast_pending_")
	totalUsers := b.getTotalUsers()

	text := fmt.Sprintf(
		"📢 *Final Confirmation*\n\n"+
			"⚠️ You are about to send a broadcast to **%d** users.\n\n"+
			"📝 Message:\n%s\n\n"+
			"Are you sure you want to send this broadcast?",
		totalUsers,
		escapeMarkdown(message),
	)

	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "🚀 Send Now", CallbackData: "broadcast_send"},
			{Text: "❌ Cancel", CallbackData: "broadcast_cancel"},
		},
	}

	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// handleBroadcastSend - Send the broadcast
func (b *Bot) handleBroadcastSend(ctx context.Context, chatID int64) {
	state, ok := b.tempState.Load(chatID)
	if !ok {
		b.sendText(ctx, chatID, "❌ No broadcast ready. Use `/broadcast` first.")
		return
	}

	stateStr := state.(string)
	if !strings.HasPrefix(stateStr, "broadcast_ready_") && !strings.HasPrefix(stateStr, "broadcast_pending_") {
		b.sendText(ctx, chatID, "❌ Invalid state. Use `/broadcast` first.")
		return
	}

	message := strings.TrimPrefix(stateStr, "broadcast_ready_")
	if message == stateStr {
		message = strings.TrimPrefix(stateStr, "broadcast_pending_")
	}

	// Start broadcast
	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"📢 *Broadcast Started*\n\n"+
			"⏳ Sending message to all users...\n\n"+
			"📝 Message:\n%s",
		escapeMarkdown(message),
	))

	// Send broadcast in background
	go b.executeBroadcastThroughMainBot(ctx, chatID, message)

	b.tempState.Delete(chatID)
}

// handleBroadcastTest - Send test broadcast
func (b *Bot) handleBroadcastTest(ctx context.Context, chatID int64) {
	state, ok := b.tempState.Load(chatID)
	if !ok {
		b.sendText(ctx, chatID, "❌ No pending broadcast found. Use `/broadcast` first.")
		return
	}

	stateStr := state.(string)
	if !strings.HasPrefix(stateStr, "broadcast_pending_") {
		b.sendText(ctx, chatID, "❌ Invalid state. Use `/broadcast` first.")
		return
	}

	message := strings.TrimPrefix(stateStr, "broadcast_pending_")
	b.sendTestBroadcast(ctx, chatID, message)
}

// handleBroadcastEdit - Edit the broadcast message
func (b *Bot) handleBroadcastEdit(ctx context.Context, chatID int64) {
	b.sendMarkdown(ctx, chatID,
		"✏️ *Edit Broadcast Message*\n\n"+
			"Please type the new message:\n\n"+
			"📌 Type `/broadcast cancel` to cancel")
	
	b.tempState.Store(chatID, "awaiting_broadcast_edit")
}

// handleBroadcastEditInput - Handle edited message input
func (b *Bot) handleBroadcastEditInput(ctx context.Context, chatID int64, text string) {
	if strings.HasPrefix(text, "/") {
		if text == "/broadcast cancel" {
			b.tempState.Delete(chatID)
			b.sendText(ctx, chatID, "❌ Broadcast cancelled.")
			return
		}
		b.sendText(ctx, chatID, "❌ Invalid input. Please type a message or `/broadcast cancel`.")
		return
	}

	// Update the message
	message := text
	b.tempState.Store(chatID, fmt.Sprintf("broadcast_pending_%s", message))

	// Show updated preview
	b.showBroadcastPreview(ctx, chatID, message)
}

// handleBroadcastCancel - Cancel broadcast
func (b *Bot) handleBroadcastCancel(ctx context.Context, chatID int64) {
	b.tempState.Delete(chatID)
	b.sendText(ctx, chatID, "❌ Broadcast cancelled.")
}

// sendTestBroadcast - Send test broadcast to admin only
func (b *Bot) sendTestBroadcast(ctx context.Context, chatID int64, message string) {
	mainBotAPI, err := telego.NewBot(b.cfg.BotToken)
	if err != nil {
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Failed to initialize main bot: %v", err))
		return
	}

	mainBot, err := mainBotAPI.GetMe(ctx)
	if err != nil {
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Failed to get main bot info: %v", err))
		return
	}

	_, err = mainBotAPI.SendMessage(ctx, &telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text: fmt.Sprintf("🧪 TEST BROADCAST\n\n%s\n\n— @%s Admin", 
			message, 
			mainBot.Username),
	})

	if err != nil {
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Test broadcast failed: %v", err))
		return
	}

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"✅ *Test Broadcast Sent*\n\n"+
			"📝 Message sent to you only:\n%s\n\n"+
			"📊 If the test looks good, click the confirm button above.",
		escapeMarkdown(message),
	))
}

// getTotalUsers - Get total number of users
func (b *Bot) getTotalUsers() int64 {
	var count int64
	b.db.Model(&models.User{}).Where("is_bot = ?", false).Count(&count)
	return count
}

// executeBroadcastThroughMainBot - Send broadcast through main bot
func (b *Bot) executeBroadcastThroughMainBot(ctx context.Context, adminChatID int64, message string) {
	var users []models.User
	if err := b.db.Where("is_bot = ?", false).Find(&users).Error; err != nil {
		b.sendText(ctx, adminChatID, fmt.Sprintf("❌ Failed to fetch users: %v", err))
		return
	}

	totalUsers := len(users)
	if totalUsers == 0 {
		b.sendText(ctx, adminChatID, "❌ No users found to broadcast to.")
		return
	}

	mainBotAPI, err := telego.NewBot(b.cfg.BotToken)
	if err != nil {
		b.sendText(ctx, adminChatID, fmt.Sprintf("❌ Failed to initialize main bot: %v", err))
		return
	}

	

	progressMsg := b.sendMarkdownReturn(ctx, adminChatID, fmt.Sprintf(
		"📢 *Broadcast in Progress*\n\n"+
			"👥 Total Users: %d\n"+
			"📤 Sent: 0\n"+
			"❌ Failed: 0\n\n"+
			"⏳ Please wait...",
		totalUsers,
	))

	successCount := 0
	failCount := 0
	var failedUsers []int64

	for i, user := range users {
		// NEW (Keep this):
_, err := mainBotAPI.SendMessage(ctx, &telego.SendMessageParams{
    ChatID: telego.ChatID{ID: user.TelegramID},
    Text: message,
})

		if err != nil {
			failCount++
			failedUsers = append(failedUsers, user.TelegramID)
			log.Printf("Failed to send broadcast to user %d: %v", user.TelegramID, err)
		} else {
			successCount++
		}

		if i%10 == 0 && i > 0 {
			b.editMarkdown(ctx, adminChatID, progressMsg, fmt.Sprintf(
				"📢 *Broadcast in Progress*\n\n"+
					"👥 Total Users: %d\n"+
					"📤 Sent: %d\n"+
					"❌ Failed: %d\n\n"+
					"⏳ Progress: %.1f%%",
				totalUsers,
				successCount,
				failCount,
				float64(successCount+failCount)/float64(totalUsers)*100,
			))
		}

		if i%10 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	report := fmt.Sprintf(
		"📢 *Broadcast Complete*\n\n"+
			"📊 *Summary:*\n"+
			"• 👥 Total Users: %d\n"+
			"• ✅ Sent: %d\n"+
			"• ❌ Failed: %d\n"+
			"• 📊 Success Rate: %.1f%%\n\n",
		totalUsers,
		successCount,
		failCount,
		float64(successCount)/float64(totalUsers)*100,
	)

	if len(failedUsers) > 0 && len(failedUsers) <= 20 {
		report += "❌ *Failed Users:*\n"
		for _, id := range failedUsers {
			report += fmt.Sprintf("• `%d`\n", id)
		}
	} else if len(failedUsers) > 20 {
		report += fmt.Sprintf("❌ *Failed Users:* %d users (list truncated)\n", len(failedUsers))
	}

	b.editMarkdown(ctx, adminChatID, progressMsg, report)

	b.logAdminAction(ctx, adminChatID, "broadcast", 0, "system",
		fmt.Sprintf("Sent broadcast to %d users (%d failed)", successCount, failCount))
}

// sendMarkdownReturn - Send markdown and return the message
func (b *Bot) sendMarkdownReturn(ctx context.Context, chatID int64, text string) *telego.Message {
	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: "Markdown",
	}

	msg, err := b.api.SendMessage(ctx, params)
	if err != nil {
		log.Printf("Failed to send markdown message to %d: %v", chatID, err)
		return nil
	}
	return msg
}

// editMarkdown - Edit a message with markdown
func (b *Bot) editMarkdown(ctx context.Context, chatID int64, msg *telego.Message, text string) {
	if msg == nil {
		b.sendMarkdown(ctx, chatID, text)
		return
	}

	params := &telego.EditMessageTextParams{
		ChatID:    telego.ChatID{ID: chatID},
		MessageID: msg.MessageID,
		Text:      text,
		ParseMode: "Markdown",
	}

	_, err := b.api.EditMessageText(ctx, params)
	if err != nil {
		log.Printf("Failed to edit message: %v", err)
	}
}

