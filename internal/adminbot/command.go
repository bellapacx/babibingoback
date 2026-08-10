package adminbot

import (
	"babibingo/internal/models"
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

    // ✅ Admin check
    if !b.checkAdminAccess(ctx, chatID, user.ID) {
        return
    }
    // ✅ Check if we're waiting for bot count input
	if state, ok := b.tempState.Load(chatID); ok && state == "awaiting_bot_count" {
		b.handleBotCountInput(ctx, chatID, text)
		return
	}
    // Log admin action
    log.Printf("📋 Admin %d (%s): %s", user.ID, user.Username, text)

    // Commands
    if strings.HasPrefix(text, "/") {
        b.handleCommand(ctx, chatID, user, text)
        return
    }

  

     b.handleUserTextInput(ctx, chatID, text)
    
    // Also check for agent text input
    b.handleAgentTextInput(ctx, chatID, text)

      b.sendAdminMenu(ctx, chatID)
    
}

func (b *Bot) handleCommand(ctx context.Context, chatID int64, user *telego.User, text string) {
    parts := strings.Split(strings.TrimPrefix(text, "/"), " ")
    command := parts[0]
    args := parts[1:]

    switch command {
    case "start":
        b.handleStart(ctx, chatID, user)
    case "help":
        b.handleHelp(ctx, chatID)
    case "agents":
        b.handleAgents(ctx, chatID, args)
    case "deposits":
        b.handleDeposits(ctx, chatID, args)
    case "withdrawals":
        b.handleWithdrawals(ctx, chatID, args)
    case "dashboard":  // ✅ Add this case
        b.showDashboard(ctx, chatID)
    case "games":
        b.handleGames(ctx, chatID, args)
    case "bots":
        b.handleBots(ctx, chatID, args)
    case "users":
        b.handleUsers(ctx, chatID, args)
    case "stats":
        b.handleStats(ctx, chatID, args)
    
    case "settings":
        b.handleSettings(ctx, chatID, args)
    default:
        b.sendText(ctx, chatID, "❌ Unknown command. Use /help for available commands.")
    }
}

func (b *Bot) handleStart(ctx context.Context, chatID int64, user *telego.User) {
    b.sendAdminMenu(ctx, chatID)
}
// command.go - Add this function

func (b *Bot) handleHelp(ctx context.Context, chatID int64) {
    b.sendMarkdown(
        ctx,
        chatID,
        "🔐 *Admin Bot Commands*\n\n"+
            "👥 *Agent Management:*\n"+
            "/agents - List all agents\n"+
            "/agents pending - View pending\n"+
            "/agents approve <id> - Approve\n"+
            "/agents reject <id> - Reject\n"+
            "/agents view <id> - View details\n"+
            "/agents revoke <id> - Revoke status\n"+
            "/agents commissions - View commissions\n\n"+
            "💳 *Deposit Management:*\n"+
            "/deposits - View pending\n"+
            "/deposits all - View all\n"+
            "/deposits approve <id> - Approve\n"+
            "/deposits reject <id> - Reject\n"+
            "/deposits search <query> - Search\n\n"+
            "🏧 *Withdrawal Management:*\n"+
            "/withdrawals - View pending\n"+
            "/withdrawals all - View all\n"+
            "/withdrawals approve <id> - Approve\n"+
            "/withdrawals reject <id> - Reject\n\n"+
            "🎱 *Game Monitoring:*\n"+
            "/games - View active\n"+
            "/games current - Current game\n"+
            "/games stats - Game stats\n"+
            "/games end <id> - Force end\n\n"+
            "🤖 *Bot Management:*\n"+
            "/bots - View status\n"+
            "/bots start - Start bots\n"+
            "/bots stop - Stop bots\n"+
            "/bots count - Bot count\n"+
            "/bots speed <n> - Set speed\n"+
            "/bots max <n> - Set max bots\n\n"+
            "👤 *User Management:*\n"+
            "/users - List users\n"+
            "/users search <query> - Search\n"+
            "/users view <id> - View\n"+
            "/users balance <id> <amt> - Adjust\n\n"+
            "📊 *Statistics:*\n"+
            "/stats - Daily stats\n"+
            "/stats weekly - Weekly\n"+
            "/stats revenue - Revenue\n"+
            "/stats agents - Agent report\n"+
            "/stats bots - Bot report\n\n"+
            "⚙️ *Settings:*\n"+
            "/settings - View settings\n"+
            "/settings admins - Manage admins\n"+
            "/settings autoapprove - Toggle\n"+
            "/settings notifications - Toggle",
    )
}
// command.go - Add this function

// ✅ showDashboard - Show admin dashboard with summary
func (b *Bot) showDashboard(ctx context.Context, chatID int64) {
	// Get pending counts
	var pendingAgents int64
	b.db.Model(&AgentRequest{}).Where("status = ?", "pending").Count(&pendingAgents)

	var pendingDeposits int64
	b.db.Model(&models.Transaction{}).Where("type = ? AND status = ?", "deposit", "pending").Count(&pendingDeposits)

	var pendingWithdrawals int64
	b.db.Model(&models.Transaction{}).Where("type = ? AND status = ?", "withdraw", "pending").Count(&pendingWithdrawals)

	var activeGames int64
	b.db.Model(&models.Game{}).Where("status IN (?)", []string{"waiting", "calling"}).Count(&activeGames)

	var totalUsers int64
	b.db.Model(&models.User{}).Count(&totalUsers)

	var totalAgents int64
	b.db.Model(&models.User{}).Where("is_agent = ?", true).Count(&totalAgents)

	// Get today's stats
	today := time.Now().Truncate(24 * time.Hour)
	var todayDeposits float64
	b.db.Model(&models.Transaction{}).
		Where("type = ? AND status = ? AND created_at >= ?", "deposit", "completed", today).
		Select("COALESCE(SUM(amount), 0)").Scan(&todayDeposits)

	var todayWithdrawals float64
	b.db.Model(&models.Transaction{}).
		Where("type = ? AND status = ? AND created_at >= ?", "withdraw", "completed", today).
		Select("COALESCE(SUM(amount), 0)").Scan(&todayWithdrawals)

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text: fmt.Sprintf(
			"📊 *Admin Dashboard*\n\n"+
				"🟡 *Pending Actions:*\n"+
				"• 👥 Agents: %d\n"+
				"• 💳 Deposits: %d\n"+
				"• 🏧 Withdrawals: %d\n\n"+
				"🎱 *Active Games:* %d\n\n"+
				"👥 *Users:* %d (🤝 Agents: %d)\n\n"+
				"💰 *Today's Activity:*\n"+
				"• Deposits: %.2f ETB\n"+
				"• Withdrawals: %.2f ETB\n\n"+
				"📋 Use /help for commands",
			pendingAgents,
			pendingDeposits,
			pendingWithdrawals,
			activeGames,
			totalUsers,
			totalAgents,
			todayDeposits,
			todayWithdrawals,
		),
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

// ============ MEMORY COMMAND ============


// ============ MONITOR COMMAND ============

