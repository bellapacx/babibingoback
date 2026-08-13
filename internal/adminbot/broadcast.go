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

// handleBroadcast - Send broadcast to all main bot users
func (b *Bot) handleBroadcast(ctx context.Context, chatID int64, args []string) {
	if len(args) == 0 {
		b.sendMarkdown(ctx, chatID,
			"📢 *Broadcast System*\n\n"+
				"Send a message to ALL main bot users.\n\n"+
				"Usage: `/broadcast <message>`\n\n"+
				"📌 Examples:\n"+
				"• `/broadcast 🎉 Special promotion!`\n"+
				"• `/broadcast 📢 System maintenance at 2 AM`\n\n"+
				"🧪 *Test Mode:*\n"+
				"• `/broadcast test <message>` - Send test to yourself only\n\n"+
				"⚠️ *Important:*\n"+
				"• Message will be sent to ALL users\n"+
				"• You will receive a confirmation report\n"+
				"• Type `/broadcast cancel` to cancel")
		return
	}

	// Check for cancel
	if args[0] == "cancel" {
		b.tempState.Delete(chatID)
		b.sendText(ctx, chatID, "❌ Broadcast cancelled.")
		return
	}

	// ✅ Check for test mode
	if args[0] == "test" {
		if len(args) < 2 {
			b.sendText(ctx, chatID, "❌ Please provide a test message.\n\nExample: `/broadcast test Hello!`")
			return
		}
		message := strings.Join(args[1:], " ")
		b.sendTestBroadcast(ctx, chatID, message)
		return
	}

	message := strings.Join(args, " ")

	// Show preview and ask for confirmation
	previewText := fmt.Sprintf(
		"📢 *Broadcast Preview*\n\n"+
			"📝 Message:\n%s\n\n"+
			"⚠️ This will be sent to ALL main bot users.\n"+
			"Type `/broadcast confirm` to send or `/broadcast cancel` to cancel.\n\n"+
			"📊 Total Users: %d\n\n"+
			"🧪 To test first: `/broadcast test %s`",
		message,
		b.getTotalUsers(),
		message,
	)

	b.sendMarkdown(ctx, chatID, previewText)
	b.tempState.Store(chatID, fmt.Sprintf("broadcast_pending_%s", message))
}

// sendTestBroadcast - Send test broadcast to the admin only
func (b *Bot) sendTestBroadcast(ctx context.Context, chatID int64, message string) {
	// Create main bot API using main bot token
	mainBotAPI, err := telego.NewBot(b.cfg.BotToken)
	if err != nil {
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Failed to initialize main bot: %v", err))
		return
	}

	// Get main bot info
	mainBot, err := mainBotAPI.GetMe(ctx)
	if err != nil {
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Failed to get main bot info: %v", err))
		return
	}

	// Send test message to the admin only
	_, err = mainBotAPI.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      fmt.Sprintf("🧪 *TEST BROADCAST*\n\n%s\n\n— @%s Admin", message, mainBot.Username),
		ParseMode: "Markdown",
	})

	if err != nil {
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Test broadcast failed: %v", err))
		return
	}

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"✅ *Test Broadcast Sent*\n\n"+
			"📝 Message sent to you only:\n%s\n\n"+
			"📊 If the test looks good, send the broadcast to all users:\n"+
			"`/broadcast confirm`",
		message,
	))
}

// handleBroadcastConfirm - Confirm and send broadcast
func (b *Bot) handleBroadcastConfirm(ctx context.Context, chatID int64) {
	// Get pending broadcast
	state, ok := b.tempState.Load(chatID)
	if !ok {
		b.sendText(ctx, chatID, "❌ No pending broadcast found. Use `/broadcast <message>` first.")
		return
	}

	stateStr := state.(string)
	if !strings.HasPrefix(stateStr, "broadcast_pending_") {
		b.sendText(ctx, chatID, "❌ Invalid state. Use `/broadcast <message>` first.")
		return
	}

	message := strings.TrimPrefix(stateStr, "broadcast_pending_")

	// Ask for final confirmation with number of users
	totalUsers := b.getTotalUsers()
	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"📢 *Final Confirmation*\n\n"+
			"⚠️ You are about to send a broadcast to **%d** users.\n\n"+
			"📝 Message:\n%s\n\n"+
			"Type `/broadcast send` to confirm and send, or `/broadcast cancel` to cancel.",
		totalUsers,
		message,
	))
	b.tempState.Store(chatID, fmt.Sprintf("broadcast_ready_%s", message))
}

// handleBroadcastSend - Send the broadcast
func (b *Bot) handleBroadcastSend(ctx context.Context, chatID int64) {
	// Get pending broadcast
	state, ok := b.tempState.Load(chatID)
	if !ok {
		b.sendText(ctx, chatID, "❌ No broadcast ready. Use `/broadcast <message>` first.")
		return
	}

	stateStr := state.(string)
	if !strings.HasPrefix(stateStr, "broadcast_ready_") {
		b.sendText(ctx, chatID, "❌ Invalid state. Use `/broadcast <message>` first.")
		return
	}

	message := strings.TrimPrefix(stateStr, "broadcast_ready_")

	// Start broadcast
	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"📢 *Broadcast Started*\n\n"+
			"⏳ Sending message to all users...\n\n"+
			"📝 Message:\n%s",
		message,
	))

	// Send broadcast in background
	go b.executeBroadcastThroughMainBot(ctx, chatID, message)

	b.tempState.Delete(chatID)
}

// getTotalUsers - Get total number of users in main bot
func (b *Bot) getTotalUsers() int64 {
	var count int64
	b.db.Model(&models.User{}).Where("is_bot = ?", false).Count(&count)
	return count
}

// executeBroadcastThroughMainBot - Send broadcast through main bot
func (b *Bot) executeBroadcastThroughMainBot(ctx context.Context, adminChatID int64, message string) {
	// Get all users from database (excluding bots)
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

	// Create main bot API using main bot token
	mainBotAPI, err := telego.NewBot(b.cfg.BotToken)
	if err != nil {
		b.sendText(ctx, adminChatID, fmt.Sprintf("❌ Failed to initialize main bot: %v", err))
		return
	}

	// Get main bot info
	mainBot, err := mainBotAPI.GetMe(ctx)
	if err != nil {
		b.sendText(ctx, adminChatID, fmt.Sprintf("❌ Failed to get main bot info: %v", err))
		return
	}

	// Send initial progress
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

	// Send to each user through main bot
	for i, user := range users {
		// Send through main bot
		_, err := mainBotAPI.SendMessage(ctx, &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: user.TelegramID},
			Text:      fmt.Sprintf("📢 *Announcement*\n\n%s\n\n— @%s Admin", message, mainBot.Username),
			ParseMode: "Markdown",
		})

		if err != nil {
			failCount++
			failedUsers = append(failedUsers, user.TelegramID)
			log.Printf("Failed to send broadcast to user %d: %v", user.TelegramID, err)
		} else {
			successCount++
		}

		// Update progress every 10 users
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

		// Rate limit - send 10 messages per second
		if i%10 == 0 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Send completion report
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

	// Log broadcast action
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