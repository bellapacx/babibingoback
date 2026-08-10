package adminbot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"babibingo/internal/models"

	"github.com/mymmrac/telego"
	"gorm.io/gorm"
)

const (
	pendingPageSize = 10
)

// ============ AGENT MANAGEMENT HANDLER ============

func (b *Bot) handleAgents(ctx context.Context, chatID int64, args []string) {
	if len(args) == 0 {
		b.showAgentsMenu(ctx, chatID)
		return
	}

	switch args[0] {
	case "menu":
		b.showAgentsMenu(ctx, chatID)
		
	case "pending":
		b.showPendingAgents(ctx, chatID, 1) // Start from page 1
		
	case "all":
		b.showAllAgents(ctx, chatID,1)
		
	case "add":
		b.handleAddAgentFlow(ctx, chatID)
		
	case "remove":
		if len(args) > 1 {
			query := strings.Join(args[1:], " ")
			b.handleRemoveAgentSearch(ctx, chatID, query)
		} else {
			b.handleRemoveAgentFlow(ctx, chatID)
		}
		
	case "search":
		if len(args) > 1 {
			query := strings.Join(args[1:], " ")
			b.searchAgents(ctx, chatID, query)
		} else {
			b.sendText(ctx, chatID, "❌ Usage: /agents search <username or phone>")
		}
		
	case "approve":
		if len(args) > 1 {
			id, _ := strconv.Atoi(args[1])
			b.approveAgent(ctx, chatID, uint(id))
		} else {
			b.sendText(ctx, chatID, "❌ Usage: /agents approve <request_id>")
		}
		
	case "reject":
		if len(args) > 1 {
			id, _ := strconv.Atoi(args[1])
			b.rejectAgent(ctx, chatID, uint(id))
		} else {
			b.sendText(ctx, chatID, "❌ Usage: /agents reject <request_id>")
		}
		
	case "view":
		if len(args) > 1 {
			id, _ := strconv.Atoi(args[1])
			b.viewAgent(ctx, chatID, uint(id))
		} else {
			b.sendText(ctx, chatID, "❌ Usage: /agents view <request_id>")
		}
		
	case "revoke":
		if len(args) > 1 {
			id, _ := strconv.ParseInt(args[1], 10, 64)
			b.revokeAgent(ctx, chatID, id)
		} else {
			b.sendText(ctx, chatID, "❌ Usage: /agents revoke <user_id>")
		}
		
	case "commissions":
		b.showAgentCommissions(ctx, chatID)
		
	default:
		b.showAgentsMenu(ctx, chatID)
	}
}

// ============ SHOW AGENTS MENU ============

func (b *Bot) showAgentsMenu(ctx context.Context, chatID int64) {
	// Get counts
	var pendingCount int64
	b.db.Model(&AgentRequest{}).Where("status = ?", "pending").Count(&pendingCount)
	
	var totalAgents int64
	b.db.Model(&models.User{}).Where("is_agent = ?", true).Count(&totalAgents)
	
	text := fmt.Sprintf(
		"🤝 *Agent Management*\n\n"+
		"📊 Overview:\n"+
		"• Total Agents: %d\n"+
		"• Pending Requests: %d\n\n"+
		"Select an action below:",
		totalAgents,
		pendingCount,
	)
	
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "➕ Add Agent", CallbackData: "agents_add"},
			{Text: "➖ Remove Agent", CallbackData: "agents_remove"},
		},
		{
			{Text: fmt.Sprintf("📋 Pending Requests (%d)", pendingCount), CallbackData: "agents_pending_1"},
			{Text: "👥 All Agents", CallbackData: "agents_all_1"},
		},
		{
			{Text: "🔍 Search Agents", CallbackData: "agents_search"},
			{Text: "💰 Commissions", CallbackData: "agents_commissions"},
		},
		{
			{Text: "🔙 Back to Menu", CallbackData: "back_to_main"},
		},
	}
	
	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}
// ============ ADD AGENT FLOW ============

func (b *Bot) handleAddAgentFlow(ctx context.Context, chatID int64) {
	b.sendMarkdown(ctx, chatID,
		"🤝 *Add Agent*\n\n"+
		"Please enter the phone number or username of the user to make an agent:\n\n"+
		"📱 Examples:\n"+
		"• `09847488474` (will auto-format to +2519847488474)\n"+
		"• `@username`\n\n"+
		"⌨️ Type the phone number or username:")
	b.tempState.Store(chatID, "awaiting_add_agent_search")
}

// ============ REMOVE AGENT FLOW ============

func (b *Bot) handleRemoveAgentFlow(ctx context.Context, chatID int64) {
	b.sendMarkdown(ctx, chatID,
		"🤝 *Remove Agent*\n\n"+
		"Please enter the phone number or username of the agent to remove:\n\n"+
		"📱 Examples:\n"+
		"• `09847488474` (will auto-format to +2519847488474)\n"+
		"• `@username`\n\n"+
		"⌨️ Type the phone number or username:")
	b.tempState.Store(chatID, "awaiting_remove_agent_search")
}

// ============ HANDLE SEARCH AND ADD AGENT ============

func (b *Bot) handleSearchAndAddAgent(ctx context.Context, chatID int64, query string) {
	users := b.findUsersByQuery(ctx, query)
	
	if len(users) == 0 {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"❌ No user found for: `%s`\n\n"+
			"Please try again with a different phone number or username.",
			query,
		))
		b.tempState.Delete(chatID)
		return
	}
	
	if len(users) > 1 {
		b.showUserSelectionForAddAgent(ctx, chatID, users)
		return
	}
	
	// Single user found
	user := users[0]
	
	// Check if already an agent
	if user.IsAgent {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"⚠️ *User is Already an Agent*\n\n"+
			"👤 User: @%s\n"+
			"🆔 `%d`\n"+
			"📱 %s\n"+
			"💰 Agent Balance: %.2f ETB\n\n"+
			"This user is already an agent. Use `/agents remove` to remove agent status.",
			user.Username,
			user.TelegramID,
			formatPhoneNumber(user.PhoneNumber),
			user.AgentBalance,
		))
		b.tempState.Delete(chatID)
		return
	}
	
	// Confirm add agent
	b.confirmAddAgent(ctx, chatID, user)
}

// ============ CONFIRM ADD AGENT ============

func (b *Bot) confirmAddAgent(ctx context.Context, chatID int64, user models.User) {
	text := fmt.Sprintf(
		"🤝 *Add Agent Confirmation*\n\n"+
		"👤 User: @%s\n"+
		"🆔 `%d`\n"+
		"📱 %s\n"+
		"💰 Balance: %.2f ETB\n\n"+
		"⚠️ This user will become an agent and will be able to earn commissions.\n\n"+
		"Are you sure you want to add this user as an agent?",
		user.Username,
		user.TelegramID,
		formatPhoneNumber(user.PhoneNumber),
		user.Balance,
	)
	
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "✅ Confirm Add", CallbackData: fmt.Sprintf("agents_confirm_add_%d", user.TelegramID)},
			{Text: "❌ Cancel", CallbackData: "agents_menu"},
		},
	}
	
	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
	b.tempState.Delete(chatID)
}

// ============ HANDLE SEARCH AND REMOVE AGENT ============

func (b *Bot) handleRemoveAgentSearch(ctx context.Context, chatID int64, query string) {
	users := b.findUsersByQuery(ctx, query)
	
	if len(users) == 0 {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"❌ No user found for: `%s`\n\n"+
			"Please try again with a different phone number or username.",
			query,
		))
		b.tempState.Delete(chatID)
		return
	}
	
	if len(users) > 1 {
		b.showUserSelectionForRemoveAgent(ctx, chatID, users)
		return
	}
	
	user := users[0]
	
	// Check if user is an agent
	if !user.IsAgent {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"⚠️ *User is Not an Agent*\n\n"+
			"👤 User: @%s\n"+
			"🆔 `%d`\n"+
			"📱 %s\n\n"+
			"This user is not an agent. Use `/agents add` to add as agent.",
			user.Username,
			user.TelegramID,
			formatPhoneNumber(user.PhoneNumber),
		))
		b.tempState.Delete(chatID)
		return
	}
	
	// Confirm remove agent
	b.confirmRemoveAgent(ctx, chatID, user)
}

// ============ CONFIRM REMOVE AGENT ============

func (b *Bot) confirmRemoveAgent(ctx context.Context, chatID int64, user models.User) {
	var referralCount int64
	b.db.Model(&models.User{}).Where("referred_by = ?", user.ID).Count(&referralCount)
	
	text := fmt.Sprintf(
		"🤝 *Remove Agent Confirmation*\n\n"+
		"👤 User: @%s\n"+
		"🆔 `%d`\n"+
		"📱 %s\n"+
		"💰 Agent Balance: %.2f ETB\n"+
		"👥 Referrals: %d\n\n"+
		"⚠️ This will remove agent status from this user.\n"+
		"⚠️ The user will no longer earn commissions.\n\n"+
		"Are you sure you want to remove agent status?",
		user.Username,
		user.TelegramID,
		formatPhoneNumber(user.PhoneNumber),
		user.AgentBalance,
		referralCount,
	)
	
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "✅ Confirm Remove", CallbackData: fmt.Sprintf("agents_confirm_remove_%d", user.TelegramID)},
			{Text: "❌ Cancel", CallbackData: "agents_menu"},
		},
	}
	
	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
	b.tempState.Delete(chatID)
}

// ============ SHOW USER SELECTION FOR ADD AGENT ============

func (b *Bot) showUserSelectionForAddAgent(ctx context.Context, chatID int64, users []models.User) {
	text := "🔍 *Multiple users found*\n\nPlease select a user to add as agent:\n\n"
	
	for i, user := range users {
		if i >= 10 {
			break
		}
		status := ""
		if user.IsAgent {
			status = " ✅ Already Agent"
		}
		text += fmt.Sprintf(
			"%d. @%s | 📱 %s | 💰 %.2f ETB%s\n🆔 `%d`\n\n",
			i+1,
			user.Username,
			formatPhoneNumber(user.PhoneNumber),
			user.Balance,
			status,
			user.TelegramID,
		)
	}
	
	text += "\n💡 *Tip:* Type the exact Telegram ID to select a user."
	
	b.sendMarkdown(ctx, chatID, text)
	b.tempState.Store(chatID, "awaiting_add_agent_selection")
}

// ============ SHOW USER SELECTION FOR REMOVE AGENT ============

func (b *Bot) showUserSelectionForRemoveAgent(ctx context.Context, chatID int64, users []models.User) {
	text := "🔍 *Multiple users found*\n\nPlease select a user to remove as agent:\n\n"
	
	foundAgents := false
	for i, user := range users {
		if i >= 10 {
			break
		}
		if user.IsAgent {
			foundAgents = true
			text += fmt.Sprintf(
				"%d. @%s | 📱 %s | 💰 %.2f ETB\n🆔 `%d`\n\n",
				i+1,
				user.Username,
				formatPhoneNumber(user.PhoneNumber),
				user.AgentBalance,
				user.TelegramID,
			)
		}
	}
	
	if !foundAgents {
		text = "🔍 *No agents found*\n\nNone of the users found are agents."
		
		keyboard := [][]telego.InlineKeyboardButton{
			{{Text: "🔙 Back to Agents", CallbackData: "agents_menu"}},
		}
		b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
		b.tempState.Delete(chatID)
		return
	}
	
	text += "\n💡 *Tip:* Type the exact Telegram ID to select a user."
	
	b.sendMarkdown(ctx, chatID, text)
	b.tempState.Store(chatID, "awaiting_remove_agent_selection")
}

// ============ EXECUTE ADD AGENT ============

func (b *Bot) executeAddAgent(ctx context.Context, chatID int64, telegramID int64) {
	var user models.User
	if err := b.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		b.sendText(ctx, chatID, "❌ User not found.")
		return
	}
	
	if user.IsAgent {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"⚠️ User @%s is already an agent.",
			user.Username,
		))
		return
	}
	
	// Update user to agent
	user.IsAgent = true
	user.AgentBalance = 0
	if err := b.db.Save(&user).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Failed to add agent: "+err.Error())
		return
	}
	
	// Create agent request record if not exists
	var request AgentRequest
	result := b.db.Where("user_id = ?", user.TelegramID).First(&request)
	if result.Error == gorm.ErrRecordNotFound {
		request = AgentRequest{
			UserID:      user.TelegramID,
			Username:    user.Username,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
			PhoneNumber: user.PhoneNumber,
			Status:      "approved",
			ReviewedBy:  &chatID,
		}
		b.db.Create(&request)
	} else {
		request.Status = "approved"
		b.db.Save(&request)
	}
	
	// Log admin action
	b.logAdminAction(ctx, chatID, "add_agent", user.TelegramID, "agent",
		fmt.Sprintf("Added user @%s as agent", user.Username))
	
	// Notify user
	go func() {
		b.sendMarkdown(
			context.Background(),
			user.TelegramID,
			"🎉 *Agent Status Added*\n\n"+
			"You have been added as a BabiBingo agent by an administrator!\n\n"+
			"💰 You can now earn commissions on referrals.\n"+
			"📊 Use the main bot to view your agent dashboard.",
		)
	}()
	
	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"✅ *Agent Added Successfully*\n\n"+
		"👤 User: @%s\n"+
		"🆔 `%d`\n"+
		"📱 %s\n"+
		"💰 Balance: %.2f ETB",
		user.Username,
		user.TelegramID,
		formatPhoneNumber(user.PhoneNumber),
		user.Balance,
	))
}

// ============ EXECUTE REMOVE AGENT ============

func (b *Bot) executeRemoveAgent(ctx context.Context, chatID int64, telegramID int64) {
	var user models.User
	if err := b.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		b.sendText(ctx, chatID, "❌ User not found.")
		return
	}
	
	if !user.IsAgent {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"⚠️ User @%s is not an agent.",
			user.Username,
		))
		return
	}
	
	// Update user to remove agent status
	user.IsAgent = false
	if err := b.db.Save(&user).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Failed to remove agent: "+err.Error())
		return
	}
	
	// Update agent request
	var request AgentRequest
	if err := b.db.Where("user_id = ?", user.TelegramID).First(&request).Error; err == nil {
		request.Status = "rejected"
		b.db.Save(&request)
	}
	
	// Log admin action
	b.logAdminAction(ctx, chatID, "remove_agent", user.TelegramID, "agent",
		fmt.Sprintf("Removed agent status from user @%s", user.Username))
	
	// Notify user
	go func() {
		b.sendMarkdown(
			context.Background(),
			user.TelegramID,
			"❌ *Agent Status Removed*\n\n"+
			"Your agent status has been removed by an administrator.\n\n"+
			"💰 Your remaining agent balance has been preserved.\n"+
			"📊 You can reapply for agent status if needed.",
		)
	}()
	
	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"✅ *Agent Removed Successfully*\n\n"+
		"👤 User: @%s\n"+
		"🆔 `%d`\n"+
		"📱 %s\n"+
		"💰 Agent Balance Preserved: %.2f ETB",
		user.Username,
		user.TelegramID,
		formatPhoneNumber(user.PhoneNumber),
		user.AgentBalance,
	))
}

// ============ SEARCH AGENTS ============

func (b *Bot) searchAgents(ctx context.Context, chatID int64, query string) {
	users := b.findUsersByQuery(ctx, query)
	
	var agents []models.User
	for _, user := range users {
		if user.IsAgent {
			agents = append(agents, user)
		}
	}
	
	if len(agents) == 0 {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"🔍 *No agents found for:* `%s`\n\n"+
			"💡 Try searching with a different phone number or username.",
			query,
		))
		return
	}
	
	text := fmt.Sprintf("🔍 *Agent Search Results*\n\nFound: %d agent(s)\n\n", len(agents))
	
	for i, agent := range agents {
		var referralCount int64
		b.db.Model(&models.User{}).Where("referred_by = ?", agent.ID).Count(&referralCount)
		
		text += fmt.Sprintf(
			"%d. @%s\n"+
			"   💰 Balance: %.2f ETB | Agent: %.2f ETB\n"+
			"   👥 Referrals: %d | 📱 %s\n"+
			"   🆔 `%d`\n\n",
			i+1,
			agent.Username,
			agent.Balance,
			agent.AgentBalance,
			referralCount,
			formatPhoneNumber(agent.PhoneNumber),
			agent.TelegramID,
		)
	}
	
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "🔙 Back to Agents", CallbackData: "agents_menu"},
		},
	}
	
	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ PENDING AGENTS WITH PAGINATION ============

func (b *Bot) showPendingAgents(ctx context.Context, chatID int64, page int) {
	if page < 1 {
		page = 1
	}

	var totalRequests int64
	b.db.Model(&AgentRequest{}).Where("status = ?", "pending").Count(&totalRequests)

	if totalRequests == 0 {
		text := "📋 *No Pending Applications*\n\n" +
			"All agent applications have been reviewed."
		
		keyboard := [][]telego.InlineKeyboardButton{
			{{Text: "🔙 Back to Agents", CallbackData: "agents_menu"}},
		}
		
		b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
		return
	}

	// Calculate total pages
	totalPages := int((totalRequests + int64(pendingPageSize) - 1) / int64(pendingPageSize))
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * pendingPageSize

	var requests []AgentRequest
	b.db.Where("status = ?", "pending").
		Order("created_at DESC").
		Limit(pendingPageSize).
		Offset(offset).
		Find(&requests)

	// Send header message with pagination
	headerText := fmt.Sprintf("📋 *Pending Applications*\n\n"+
		"📊 Total: %d | Page %d/%d\n\n"+
		"Click on each request below to approve or reject:",
		totalRequests, page, totalPages)

	headerKeyboard := [][]telego.InlineKeyboardButton{}
	
	// Navigation row
	var navRow []telego.InlineKeyboardButton
	
	if page > 1 {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "⬅️ Previous",
			CallbackData: fmt.Sprintf("agents_pending_%d", page-1),
		})
	}
	
	if page < totalPages {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "Next ➡️",
			CallbackData: fmt.Sprintf("agents_pending_%d", page+1),
		})
	}
	
	if len(navRow) > 0 {
		headerKeyboard = append(headerKeyboard, navRow)
	}
	
	// Add page indicator and back button
	headerKeyboard = append(headerKeyboard, []telego.InlineKeyboardButton{
		{Text: fmt.Sprintf("📄 Page %d/%d", page, totalPages), CallbackData: "agents_current"},
	})
	
	headerKeyboard = append(headerKeyboard, []telego.InlineKeyboardButton{
		{Text: "🔙 Back to Agents", CallbackData: "agents_menu"},
	})
	
	// Send header
	b.sendMarkdownKeyboard(ctx, chatID, headerText, headerKeyboard)

	// Send each request as a separate message with its own buttons
	// With a small delay between messages to avoid rate limiting
	for i, req := range requests {
		// Escape special characters for Markdown
		username := escapeMarkdown(req.Username)
		phone := escapeMarkdown(formatPhoneNumber(req.PhoneNumber))
		
		// Build agent info text
		text := fmt.Sprintf(
			"📌 *Request #%d*\n"+
			"👤 @%s\n"+
			"🆔 `%d`\n"+
			"📱 %s\n"+
			"📅 %s",
			req.ID,
			username,
			req.UserID,
			phone,
			req.CreatedAt.Format("Jan 2, 2006 15:04"),
		)

		// Build buttons for this agent
		keyboard := [][]telego.InlineKeyboardButton{
			{
				{
					Text:         "✅ Approve",
					CallbackData: fmt.Sprintf("agents_approve_%d", req.ID),
				},
				{
					Text:         "❌ Reject",
					CallbackData: fmt.Sprintf("agents_reject_%d", req.ID),
				},
				{
					Text:         "👤 View",
					CallbackData: fmt.Sprintf("agents_view_%d", req.ID),
				},
			},
		}
		
		// Send each agent as a separate message
		b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
		
		// Small delay between messages to avoid rate limiting
		// Telegram allows ~30 messages per second
		if (i+1)%10 == 0 {
			time.Sleep(200 * time.Millisecond)
		}
	}
}
// ✅ escapeMarkdown escapes special characters for Telegram Markdown
func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"`", "\\`",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// ============ ALL AGENTS ============

func (b *Bot) showAllAgents(ctx context.Context, chatID int64, page int) {
	if page < 1 {
		page = 1
	}

	var totalAgents int64
	b.db.Model(&models.User{}).Where("is_agent = ?", true).Count(&totalAgents)

	if totalAgents == 0 {
		text := "📋 *No Agents Found*\n\n" +
			"There are currently no agents in the system."
		
		keyboard := [][]telego.InlineKeyboardButton{
			{{Text: "🔙 Back to Agents", CallbackData: "agents_menu"}},
		}
		
		b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
		return
	}

	// Calculate total pages
	pageSize := 10
	totalPages := int((totalAgents + int64(pageSize) - 1) / int64(pageSize))
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * pageSize

	var agents []models.User
	b.db.Where("is_agent = ?", true).
		Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&agents)

	// Send header with pagination
	text := fmt.Sprintf("👥 *All Agents*\n\n"+
		"📊 Total: %d | Page %d/%d\n\n",
		totalAgents, page, totalPages)
	
	for i, agent := range agents {
		var referralCount int64
		b.db.Model(&models.User{}).Where("referred_by = ?", agent.ID).Count(&referralCount)
		
		// ✅ Escape special characters for Markdown
		username := escapeMarkdown(agent.Username)
		phone := escapeMarkdown(formatPhoneNumber(agent.PhoneNumber))
		
		text += fmt.Sprintf(
			"%d. @%s\n"+
			"   💰 Agent Balance: %.2f ETB\n"+
			"   👥 Referrals: %d\n"+
			"   📱 %s\n"+
			"   🆔 `%d`\n\n",
			offset+i+1,
			username,
			agent.AgentBalance,
			referralCount,
			phone,
			agent.TelegramID,
		)
	}
	
	// Build keyboard with pagination
	var keyboard [][]telego.InlineKeyboardButton
	
	// Navigation row
	var navRow []telego.InlineKeyboardButton
	
	if page > 1 {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "⬅️ Previous",
			CallbackData: fmt.Sprintf("agents_all_%d", page-1),
		})
	}
	
	if page < totalPages {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "Next ➡️",
			CallbackData: fmt.Sprintf("agents_all_%d", page+1),
		})
	}
	
	if len(navRow) > 0 {
		keyboard = append(keyboard, navRow)
	}
	
	// Add page indicator
	keyboard = append(keyboard, []telego.InlineKeyboardButton{
		{Text: fmt.Sprintf("📄 Page %d/%d", page, totalPages), CallbackData: "agents_current"},
	})
	
	// Action buttons
	keyboard = append(keyboard, []telego.InlineKeyboardButton{
		{Text: "➕ Add Agent", CallbackData: "agents_add"},
		{Text: "➖ Remove Agent", CallbackData: "agents_remove"},
	})
	
	keyboard = append(keyboard, []telego.InlineKeyboardButton{
		{Text: "🔙 Back to Menu", CallbackData: "agents_menu"},
	})
	
	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ APPROVE AGENT ============

func (b *Bot) approveAgent(ctx context.Context, chatID int64, requestID uint) {
	var request AgentRequest
	if err := b.db.First(&request, requestID).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Request not found.")
		return
	}

	if request.Status != "pending" {
		b.sendText(ctx, chatID, fmt.Sprintf("⚠️ This request is already %s.", request.Status))
		return
	}

	request.Status = "approved"
	now := time.Now()
	request.ReviewedAt = &now
	reviewedBy := chatID
	request.ReviewedBy = &reviewedBy
	b.db.Save(&request)

	// Update user in main database
	if err := b.db.Model(&models.User{}).Where("telegram_id = ?", request.UserID).Update("is_agent", true).Error; err != nil {
		b.sendText(ctx, chatID, "⚠️ Approved but failed to update user status.")
		return
	}

	// Notify user
	go func() {
		b.sendMarkdown(
			context.Background(),
			request.UserID,
			"🎉 *Congratulations!*\n\n"+
			"Your agent application has been *approved*! 🎉\n\n"+
			"💰 You are now a BabiBingo agent!\n"+
			"• Earn commissions on referrals\n"+
			"• Use the main bot to view your agent dashboard",
		)
	}()

	b.logAdminAction(ctx, chatID, "approve_agent", request.UserID, "agent", 
		fmt.Sprintf("Approved agent request #%d", requestID))

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"✅ *Agent #%d Approved*\n\nUser @%s is now an agent.",
		request.ID, request.Username,
	))
}

// ============ REJECT AGENT ============

func (b *Bot) rejectAgent(ctx context.Context, chatID int64, requestID uint) {
	var request AgentRequest
	if err := b.db.First(&request, requestID).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Request not found.")
		return
	}

	if request.Status != "pending" {
		b.sendText(ctx, chatID, fmt.Sprintf("⚠️ This request is already %s.", request.Status))
		return
	}

	request.Status = "rejected"
	now := time.Now()
	request.ReviewedAt = &now
	reviewedBy := chatID
	request.ReviewedBy = &reviewedBy
	b.db.Save(&request)

	// Notify user
	go func() {
		b.sendMarkdown(
			context.Background(),
			request.UserID,
			"❌ *Application Rejected*\n\n"+
			"Your agent application has been rejected.\n\n"+
			"Contact support for more information.",
		)
	}()

	b.logAdminAction(ctx, chatID, "reject_agent", request.UserID, "agent", 
		fmt.Sprintf("Rejected agent request #%d", requestID))

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"❌ *Agent #%d Rejected*\n\nUser @%s has been rejected.",
		request.ID, request.Username,
	))
}

// ============ VIEW AGENT ============

func (b *Bot) viewAgent(ctx context.Context, chatID int64, requestID uint) {
	var request AgentRequest
	if err := b.db.First(&request, requestID).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Request not found.")
		return
	}

	var user models.User
	b.db.Where("telegram_id = ?", request.UserID).First(&user)

	var referralCount int64
	b.db.Model(&models.User{}).Where("referred_by = ?", user.ID).Count(&referralCount)

	var totalCommission float64
	b.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ?", user.ID, "agent_commission").
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalCommission)

	statusEmoji := "🟡"
	statusText := "Pending"
	if request.Status == "approved" {
		statusEmoji = "✅"
		statusText = "Approved"
	} else if request.Status == "rejected" {
		statusEmoji = "❌"
		statusText = "Rejected"
	}

	text := fmt.Sprintf(
		"📋 *Agent Details*\n\n"+
		"👤 Name: %s %s\n"+
		"🆔 Telegram: @%s\n"+
		"📱 Phone: %s\n"+
		"📊 Status: %s %s\n"+
		"📅 Joined: %s\n\n"+
		"💰 *Agent Stats:*\n"+
		"• Agent Balance: %.2f ETB\n"+
		"• Total Earned: %.2f ETB\n"+
		"• Referrals: %d",
		request.FirstName,
		request.LastName,
		request.Username,
		formatPhoneNumber(request.PhoneNumber),
		statusEmoji,
		statusText,
		user.CreatedAt.Format("Jan 2, 2006"),
		user.AgentBalance,
		totalCommission,
		referralCount,
	)
	
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "🔙 Back to Pending", CallbackData: "agents_pending_1"},
			{Text: "🔙 Back to Menu", CallbackData: "agents_menu"},
		},
	}
	
	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ REVOKE AGENT ============

func (b *Bot) revokeAgent(ctx context.Context, chatID int64, userID int64) {
	var user models.User
	if err := b.db.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
		b.sendText(ctx, chatID, "❌ User not found.")
		return
	}

	if !user.IsAgent {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"⚠️ User @%s is not an agent.",
			user.Username,
		))
		return
	}

	user.IsAgent = false
	if err := b.db.Save(&user).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Failed to revoke agent status.")
		return
	}

	// Update agent request
	b.db.Model(&AgentRequest{}).Where("user_id = ?", userID).Update("status", "rejected")

	// Notify user
	go func() {
		b.sendMarkdown(
			context.Background(),
			userID,
			"❌ *Agent Status Revoked*\n\n"+
			"Your agent status has been revoked by an administrator.",
		)
	}()

	b.logAdminAction(ctx, chatID, "revoke_agent", userID, "agent", 
		fmt.Sprintf("Revoked agent status for user %d", userID))

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"❌ *Agent Revoked*\n\nUser @%s is no longer an agent.",
		user.Username,
	))
}

// ============ AGENT COMMISSIONS ============

func (b *Bot) showAgentCommissions(ctx context.Context, chatID int64) {
	var agents []models.User
	b.db.Where("is_agent = ?", true).Order("agent_balance DESC").Limit(20).Find(&agents)

	if len(agents) == 0 {
		text := "📊 *No Agents Found*\n\n" +
			"There are currently no agents with commissions."
		
		keyboard := [][]telego.InlineKeyboardButton{
			{{Text: "🔙 Back to Agents", CallbackData: "agents_menu"}},
		}
		
		b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
		return
	}

	text := "📊 *Top Agents by Commission*\n\n"
	
	totalCommissions := 0.0
	for i, agent := range agents {
		if i >= 10 {
			break
		}
		var referralCount int64
		b.db.Model(&models.User{}).Where("referred_by = ?", agent.ID).Count(&referralCount)
		
		// ✅ Escape special characters for Markdown
		username := escapeMarkdown(agent.Username)
		
		text += fmt.Sprintf(
			"%d. @%s\n"+
			"   💰 Commission: %.2f ETB\n"+
			"   👥 Referrals: %d\n\n",
			i+1,
			username,
			agent.AgentBalance,
			referralCount,
		)
		
		totalCommissions += agent.AgentBalance
	}
	
	text += fmt.Sprintf("📊 *Total Commissions:* %.2f ETB", totalCommissions)
	
	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "🔙 Back to Agents", CallbackData: "agents_menu"},
		},
	}
	
	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ CALLBACK HANDLERS FOR AGENTS ============

func (b *Bot) handleAgentCallbacks(ctx context.Context, query *telego.CallbackQuery) {
	data := query.Data
	chatID := query.Message.GetChat().ID

	// ✅ Add debug log
	log.Printf("🔵 Agent callback received: %s from chat %d", data, chatID)

	b.api.AnswerCallbackQuery(ctx, &telego.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		log.Printf("⚠️ Invalid callback data format: %s", data)
		return
	}

	log.Printf("🔍 Parsed callback: prefix=%s, action=%s, parts=%v", parts[0], parts[1], parts)

	switch parts[0] {
	case "agents":
		switch parts[1] {
		case "menu":
			log.Println("📋 Showing agents menu")
			b.showAgentsMenu(ctx, chatID)
			
		case "add":
			log.Println("➕ Add agent flow")
			b.handleAddAgentFlow(ctx, chatID)
			
		case "remove":
			log.Println("➖ Remove agent flow")
			b.handleRemoveAgentFlow(ctx, chatID)
			
		case "pending":
			page := 1
			if len(parts) > 2 {
				if p, err := strconv.Atoi(parts[2]); err == nil {
					page = p
				}
			}
			log.Printf("📋 Showing pending agents page %d", page)
			b.showPendingAgents(ctx, chatID, page)
			
		// In handleAgentCallbacks function
case "all":
    page := 1
    if len(parts) > 2 {
        if p, err := strconv.Atoi(parts[2]); err == nil {
            page = p
        }
    }
    b.showAllAgents(ctx, chatID, page)
			
		case "search":
			log.Println("🔍 Search agents flow")
			b.sendMarkdown(ctx, chatID,
				"🔍 *Search Agents*\n\n"+
				"📱 Enter phone number or username to search for agents:")
			b.tempState.Store(chatID, "awaiting_agent_search")
			
		case "commissions":
			log.Println("💰 Showing agent commissions")
			b.showAgentCommissions(ctx, chatID)
			
		case "approve":
			if len(parts) > 2 {
				id, _ := strconv.Atoi(parts[2])
				log.Printf("✅ Approving agent request #%d", id)
				b.approveAgent(ctx, chatID, uint(id))
				// Refresh pending list after action
				b.showPendingAgents(ctx, chatID, 1)
			}
			
		case "reject":
			if len(parts) > 2 {
				id, _ := strconv.Atoi(parts[2])
				log.Printf("❌ Rejecting agent request #%d", id)
				b.rejectAgent(ctx, chatID, uint(id))
				// Refresh pending list after action
				b.showPendingAgents(ctx, chatID, 1)
			}
			
		case "view":
			if len(parts) > 2 {
				id, _ := strconv.Atoi(parts[2])
				log.Printf("👤 Viewing agent request #%d", id)
				b.viewAgent(ctx, chatID, uint(id))
			}
			
		case "confirm_add":
			if len(parts) > 2 {
				userID, _ := strconv.ParseInt(parts[2], 10, 64)
				log.Printf("✅ Confirming add agent for user %d", userID)
				b.executeAddAgent(ctx, chatID, userID)
			}
			
		case "confirm_remove":
			if len(parts) > 2 {
				userID, _ := strconv.ParseInt(parts[2], 10, 64)
				log.Printf("✅ Confirming remove agent for user %d", userID)
				b.executeRemoveAgent(ctx, chatID, userID)
			}
			
		default:
			log.Printf("⚠️ Unknown agent action: %s", parts[1])
			b.sendText(ctx, chatID, "❌ Unknown agent action.")
		}
	default:
		log.Printf("⚠️ Unexpected callback prefix: %s", parts[0])
		b.sendText(ctx, chatID, "❌ Unknown callback.")
	}
}

// ============ HANDLE AGENT TEXT INPUT ============

func (b *Bot) handleAgentTextInput(ctx context.Context, chatID int64, text string) {
	if state, ok := b.tempState.Load(chatID); ok {
		stateStr := state.(string)
		
		// Add agent search
		if stateStr == "awaiting_add_agent_search" {
			b.tempState.Delete(chatID)
			b.handleSearchAndAddAgent(ctx, chatID, text)
			return
		}
		
		// Remove agent search
		if stateStr == "awaiting_remove_agent_search" {
			b.tempState.Delete(chatID)
			b.handleRemoveAgentSearch(ctx, chatID, text)
			return
		}
		
		// Add agent selection (user typed an ID)
		if stateStr == "awaiting_add_agent_selection" {
			b.tempState.Delete(chatID)
			if id, err := strconv.ParseInt(text, 10, 64); err == nil {
				b.confirmAddAgent(ctx, chatID, models.User{TelegramID: id})
			} else {
				b.sendText(ctx, chatID, "❌ Invalid user ID. Please type a valid Telegram ID.")
			}
			return
		}
		
		// Remove agent selection (user typed an ID)
		if stateStr == "awaiting_remove_agent_selection" {
			b.tempState.Delete(chatID)
			if id, err := strconv.ParseInt(text, 10, 64); err == nil {
				b.confirmRemoveAgent(ctx, chatID, models.User{TelegramID: id})
			} else {
				b.sendText(ctx, chatID, "❌ Invalid user ID. Please type a valid Telegram ID.")
			}
			return
		}
		
		// Agent search from button
		if stateStr == "awaiting_agent_search" {
			b.tempState.Delete(chatID)
			b.searchAgents(ctx, chatID, text)
			return
		}
	}
}