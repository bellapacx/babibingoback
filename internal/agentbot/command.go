package agentbot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mymmrac/telego"
)

func (b *Bot) handleMessage(ctx context.Context, msg *telego.Message) {
	if msg.From == nil {
		return
	}

	chatID := msg.Chat.ID
	user := msg.From
	text := msg.Text

	// Command handling
	if strings.HasPrefix(text, "/") {
		b.handleCommand(ctx, chatID, user, text)
		return
	}

	// Default response
	b.sendMarkdown(
		ctx,
		chatID,
		"🤖 *BabiBingo Agent Bot*\n\n"+
			"Use /apply to submit your agent application.\n"+
			"Use /status to check your application status.",
	)
}

func (b *Bot) handleCommand(ctx context.Context, chatID int64, user *telego.User, text string) {
	parts := strings.Split(strings.TrimPrefix(text, "/"), " ")
	command := parts[0]

	switch command {
	case "start":
		b.handleStart(ctx, chatID, user)

	case "apply":
		b.handleApply(ctx, chatID, user)

	case "status":
		b.handleStatus(ctx, chatID, user)

	default:
		b.sendText(ctx, chatID, "❌ Unknown command. Use /apply to apply.")
	}
}

func (b *Bot) handleStart(ctx context.Context, chatID int64, user *telego.User) {
	b.sendMarkdown(
		ctx,
		chatID,
		"🤖 *BabiBingo Agent Bot*\n\n"+
			"Welcome! This bot handles agent applications for BabiBingo.\n\n"+
			"📝 *Commands:*\n"+
			"/apply - Submit agent application\n"+
			"/status - Check application status\n\n"+
			"💰 *Benefits:*\n"+
			"• 1 ETB commission per card played by referrals\n"+
			"• Access to agent dashboard\n"+
			"• Exclusive agent support",
	)
}

func (b *Bot) handleApply(ctx context.Context, chatID int64, user *telego.User) {
	// Check if user already applied
	var existing AgentRequest
	err := b.db.Where("user_id = ?", user.ID).First(&existing).Error

	if err == nil {
		if existing.Status == "pending" {
			b.sendMarkdown(
				ctx,
				chatID,
				"⏳ *Application Pending*\n\n"+
					"You already have a pending application.\n"+
					"Please wait for admin review.\n\n"+
					"Status: /status",
			)
			return
		}
		if existing.Status == "approved" {
			b.sendMarkdown(
				ctx,
				chatID,
				"✅ *You are already an agent!*\n\n"+
					"You have been approved as a BabiBingo agent.\n"+
					"Check the main bot for your dashboard.",
			)
			return
		}
		if existing.Status == "rejected" {
			b.sendMarkdown(
				ctx,
				chatID,
				"❌ *Application Rejected*\n\n"+
					"Your previous application was rejected.\n"+
					"Contact support for more information.",
			)
			return
		}
	}

	// ✅ Create new application
	request := AgentRequest{
		UserID:      user.ID,
		Username:    user.Username,
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		PhoneNumber: "", // Will be added later
		Status:      "pending",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := b.db.Create(&request).Error; err != nil {
		log.Printf("Failed to create agent request: %v", err)
		b.sendText(ctx, chatID, "❌ Failed to submit application. Please try again.")
		return
	}

	// ✅ Send confirmation to user
	b.sendMarkdown(
		ctx,
		chatID,
		"✅ *Request Submitted!*\n\n"+
			"Your agent application has been sent to the admin.\n\n"+
			"You will receive a notification here once it's approved. 🎉\n\n"+
			"📊 Check status: /status",
	)

	// ✅ Send to admin (your admin bot will handle this)
	// We'll just log it here
	log.Printf("📋 New agent application from user %d (@%s)", user.ID, user.Username)
}

func (b *Bot) handleStatus(ctx context.Context, chatID int64, user *telego.User) {
	var request AgentRequest
	err := b.db.Where("user_id = ?", user.ID).First(&request).Error

	if err != nil {
		b.sendMarkdown(
			ctx,
			chatID,
			"❌ *No Application Found*\n\n"+
				"You haven't submitted an agent application yet.\n\n"+
				"Submit one now: /apply",
		)
		return
	}

	statusEmoji := "⏳"
	statusText := "Pending Review"
	switch request.Status {
	case "pending":
		statusEmoji = "⏳"
		statusText = "Pending Review"
	case "approved":
		statusEmoji = "✅"
		statusText = "Approved! 🎉"
	case "rejected":
		statusEmoji = "❌"
		statusText = "Rejected"
	}

	b.sendMarkdown(
		ctx,
		chatID,
		fmt.Sprintf(
			"📊 *Application Status*\n\n"+
				"%s Status: %s\n"+
				"📅 Submitted: %s\n\n"+
				"Once approved, you'll get access to the agent dashboard.",
			statusEmoji,
			statusText,
			request.CreatedAt.Format("Jan 2, 2006 15:04"),
		),
	)
}