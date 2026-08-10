package adminbot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"babibingo/internal/models"

	"github.com/mymmrac/telego"
)

// ✅ RobotSettings - Simple robot settings
type RobotSettings struct {
	DesiredCount int
}

var defaultRobotSettings = RobotSettings{
	DesiredCount: 20,
}

// handleBots - Main bot handler
func (b *Bot) handleBots(ctx context.Context, chatID int64, args []string) {
	// ✅ Check if we're waiting for input first
	if state, ok := b.tempState.Load(chatID); ok && state == "awaiting_bot_count" {
		b.handleBotCountInput(ctx, chatID, strings.Join(args, " "))
		return
	}

	if len(args) == 0 {
		b.showBotStatus(ctx, chatID)
		return
	}

	switch args[0] {
	case "set":
		// ✅ Always prompt for count (no inline args)
		b.sendMarkdown(
			ctx,
			chatID,
			"📝 *Set Bot Count*\n\n"+
				"Please enter the desired number of bots (1-200).\n\n"+
				"Example: `30`",
		)
		b.tempState.Store(chatID, "awaiting_bot_count")
		
	case "status":
		b.showBotStatus(ctx, chatID)
		
	case "disable":
		// ✅ Disable all bots
		if b.engine == nil {
			b.sendText(ctx, chatID, "❌ Game engine not available.")
			return
		}
		
		botManager := b.engine.GetBotManager()
		if botManager == nil {
			b.sendText(ctx, chatID, "❌ Bot manager not available.")
			return
		}
		
		// Confirm before disabling
		keyboard := [][]telego.InlineKeyboardButton{
			{
				{Text: "✅ Yes, Disable All Bots", CallbackData: "bots_disable_confirm"},
				{Text: "❌ Cancel", CallbackData: "bots_back"},
			},
		}
		
		b.sendMarkdownKeyboard(ctx, chatID, 
			"⚠️ *Disable All Bots*\n\n"+
			"This will:\n"+
			"• Set bot count to 0\n"+
			"• Stop the bot routine\n"+
			"• Clear all bots from memory\n\n"+
			"Are you sure you want to continue?", 
			keyboard)
		
	case "enable":
		// ✅ Enable bots with a count
		if b.engine == nil {
			b.sendText(ctx, chatID, "❌ Game engine not available.")
			return
		}
		
		botManager := b.engine.GetBotManager()
		if botManager == nil {
			b.sendText(ctx, chatID, "❌ Bot manager not available.")
			return
		}
		
		// Parse count
		count := 20
		if len(args) > 1 {
			if c, err := strconv.Atoi(args[1]); err == nil && c > 0 && c <= 200 {
				count = c
			} else {
				b.sendText(ctx, chatID, "❌ Invalid count. Please use a number between 1-200.\n\nExample: `/bots enable 30`")
				return
			}
		}
		
		// Enable bots
		botManager.EnableBots(count)
		
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"🚀 *Bots Enabled*\n\n"+
			"Bots have been enabled with count: %d\n\n"+
			"• Desired count: %d\n"+
			"• Bot routine started\n"+
			"• Bots will join next game", count, count))
		
	default:
		b.sendText(ctx, chatID, "❌ Unknown command.\n\nAvailable:\n"+
			"/bots status - Show bot status\n"+
			"/bots set - Set bot count (1-200)\n"+
			"/bots disable - Disable all bots\n"+
			"/bots enable <count> - Enable bots with count")
	}
}

// ✅ showBotStatus - With interactive buttons
func (b *Bot) showBotStatus(ctx context.Context, chatID int64) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("🔴 Panic in showBotStatus: %v", r)
			b.sendText(ctx, chatID, "❌ Error loading status. Please try again.")
		}
	}()

	// Get total bots
	var totalBots int64
	if err := b.db.Model(&models.User{}).Where("is_bot = ?", true).Count(&totalBots).Error; err != nil {
		log.Printf("DB Error: %v", err)
		b.sendText(ctx, chatID, "❌ Database error")
		return
	}

	desiredCount := b.getDesiredBotCount()

	// Check if bot manager is running
	isRunning := false
	if b.engine != nil && b.engine.GetBotManager() != nil {
		isRunning = true
	}

	statusEmoji := "✅"
	statusText := "Running"
	if !isRunning {
		statusEmoji = "⏹️"
		statusText = "Stopped"
	}

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text: fmt.Sprintf(
			"🤖 *Bot Manager*\n\n"+
				"📊 *Status:* %s %s\n"+
				"👥 *Current Bots:* %d\n"+
				"🎯 *Target Count:* %d\n\n"+
				"Select an option below:",
			statusEmoji,
			statusText,
			totalBots,
			desiredCount,
		),
		ParseMode: "Markdown",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text:         "➕ Add 5 Bots",
						CallbackData: "bots_add_5",
					},
					{
						Text:         "➖ Remove 5 Bots",
						CallbackData: "bots_remove_5",
					},
				},
				{
					{
						Text:         "➕ Add 10 Bots",
						CallbackData: "bots_add_10",
					},
					{
						Text:         "➖ Remove 10 Bots",
						CallbackData: "bots_remove_10",
					},
				},
				{
					{
						Text:         "🎯 Set Count",
						CallbackData: "bots_set_prompt",
					},
				},
				{
					{
						Text:         "🔄 Refresh",
						CallbackData: "bots_status",
					},
				},
				{
					{
						Text:         "🔙 Back",
						CallbackData: "back_to_menu",
					},
				},
			},
		},
	}

	b.sendMessage(ctx, &msg)
}

// ✅ setBotCount - Set desired number of bots (UPDATED: max 200)
func (b *Bot) setBotCount(ctx context.Context, chatID int64, count int) {
	if b.engine == nil {
		b.sendText(ctx, chatID, "❌ Game engine not available.")
		return
	}

	botManager := b.engine.GetBotManager()
	if botManager == nil {
		b.sendText(ctx, chatID, "❌ Bot manager not available.")
		return
	}

	// ✅ Updated validation: 1-200
	if count < 1 {
		count = 1
	}
	if count > 200 {
		count = 200
	}

	botManager.SetDesiredCount(count)

	b.logAdminAction(ctx, chatID, "set_bot_count", 0, "bots", fmt.Sprintf("Set desired bot count to %d", count))

	b.sendMarkdown(
		ctx,
		chatID,
		fmt.Sprintf(
			"✅ *Bot Count Updated*\n\n"+
				"🎯 Target bot count set to: *%d*\n\n"+
				"📊 Current bots: %d\n"+
				"⚠️ Bots will automatically adjust to reach this target.",
			count,
			b.getCurrentBotCount(),
		),
	)
}

// ✅ addBots - Add a specific number of bots (UPDATED: max 200)
func (b *Bot) addBots(ctx context.Context, chatID int64, count int) {
	if b.engine == nil {
		b.sendText(ctx, chatID, "❌ Game engine not available.")
		return
	}

	botManager := b.engine.GetBotManager()
	if botManager == nil {
		b.sendText(ctx, chatID, "❌ Bot manager not available.")
		return
	}

	currentCount := b.getCurrentBotCount()
	newCount := currentCount + count
	
	// ✅ Updated max to 200
	if newCount > 200 {
		newCount = 200
		b.sendText(ctx, chatID, fmt.Sprintf("⚠️ Max bots is 200. Setting to 200."))
	}

	botManager.SetDesiredCount(newCount)

	b.logAdminAction(ctx, chatID, "add_bots", 0, "bots", fmt.Sprintf("Added %d bots, new count: %d", count, newCount))

	b.sendMarkdown(
		ctx,
		chatID,
		fmt.Sprintf(
			"➕ *Bots Added*\n\n"+
				"Added *%d* bots.\n\n"+
				"🎯 New target: *%d* bots\n"+
				"📊 Current bots: %d",
			count,
			newCount,
			b.getCurrentBotCount(),
		),
	)
}

// ✅ removeBots - Remove a specific number of bots
func (b *Bot) removeBots(ctx context.Context, chatID int64, count int) {
	if b.engine == nil {
		b.sendText(ctx, chatID, "❌ Game engine not available.")
		return
	}

	botManager := b.engine.GetBotManager()
	if botManager == nil {
		b.sendText(ctx, chatID, "❌ Bot manager not available.")
		return
	}

	currentCount := b.getCurrentBotCount()
	newCount := currentCount - count
	if newCount < 1 {
		newCount = 1
		b.sendText(ctx, chatID, fmt.Sprintf("⚠️ Minimum bots is 1. Setting to 1."))
	}

	botManager.SetDesiredCount(newCount)

	b.logAdminAction(ctx, chatID, "remove_bots", 0, "bots", fmt.Sprintf("Removed %d bots, new count: %d", count, newCount))

	b.sendMarkdown(
		ctx,
		chatID,
		fmt.Sprintf(
			"➖ *Bots Removed*\n\n"+
				"Removed *%d* bots.\n\n"+
				"🎯 New target: *%d* bots\n"+
				"📊 Current bots: %d",
			count,
			newCount,
			b.getCurrentBotCount(),
		),
	)
}

// ✅ getCurrentBotCount - Get current bot count
func (b *Bot) getCurrentBotCount() int {
	var count int64
	b.db.Model(&models.User{}).Where("is_bot = ?", true).Count(&count)
	return int(count)
}

// ✅ getDesiredBotCount - Get desired bot count
func (b *Bot) getDesiredBotCount() int {
	if b.engine != nil && b.engine.GetBotManager() != nil {
		return b.engine.GetBotManager().GetDesiredCount()
	}
	return defaultRobotSettings.DesiredCount
}

// ✅ handleBotCountInput - Handle user input for bot count (UPDATED: max 200)
func (b *Bot) handleBotCountInput(ctx context.Context, chatID int64, text string) {
	// Trim whitespace
	text = strings.TrimSpace(text)
	
	// Parse the number
	count, err := strconv.Atoi(text)
	if err != nil || count < 1 || count > 200 {
		b.sendText(ctx, chatID, "❌ Invalid count. Please enter a number between 1 and 200.\n\nExample: `30`")
		return
	}

	// Set the count using the admin bot package
	b.setBotCount(ctx, chatID, count)
	
	// Clear the state
	b.tempState.Delete(chatID)
}