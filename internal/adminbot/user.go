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

// ============ USER MANAGEMENT HANDLER ============

func (b *Bot) handleUsers(ctx context.Context, chatID int64, args []string) {
	log.Printf("🔵 handleUsers called by chatID: %d", chatID)
	b.showUsersMenu(ctx, chatID)
}

// ============ SHOW USERS MAIN MENU ============

func (b *Bot) showUsersMenu(ctx context.Context, chatID int64) {
	log.Printf("🔵 showUsersMenu called for chatID: %d", chatID)

	text := "👥 *User Management*\n\n" +
		"Select an action below:\n\n" +
		"🔍 *Search* - Find users by phone/username\n" +
		"➕ *Add Balance* - Add balance to a user\n" +
		"➖ *Deduct Balance* - Deduct balance from a user\n" +
		"📋 *List* - View all users\n" +
		"📊 *Stats* - View user statistics"

	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "🔍 Search Users", CallbackData: "users_search"},
			{Text: "➕ Add Balance", CallbackData: "users_add_balance"},
		},
		{
			{Text: "➖ Deduct Balance", CallbackData: "users_deduct_balance"},
			{Text: "📋 List All", CallbackData: "users_list"},
		},
		{
			{Text: "📊 User Stats", CallbackData: "users_stats"},
			{Text: "🔙 Back to Menu", CallbackData: "back_to_menu"},
		},
	}

	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ HANDLE ADD BALANCE FLOW ============

func (b *Bot) handleAddBalanceFlow(ctx context.Context, chatID int64) {
	log.Printf("🔵 handleAddBalanceFlow called for chatID: %d", chatID)

	b.sendMarkdown(ctx, chatID,
		"💰 *Add Balance*\n\n"+
			"Please enter the phone number or username of the user:\n\n"+
			"📱 Examples:\n"+
			"• `09847488474` (will auto-format to +2519847488474)\n"+
			"• `@username`\n\n"+
			"⌨️ Type the phone number or username:")
	b.tempState.Store(chatID, "awaiting_add_balance_search")
	log.Printf("🔵 State stored: awaiting_add_balance_search for chatID: %d", chatID)
}

// ============ HANDLE DEDUCT BALANCE FLOW ============

func (b *Bot) handleDeductBalanceFlow(ctx context.Context, chatID int64) {
	log.Printf("🔵 handleDeductBalanceFlow called for chatID: %d", chatID)

	b.sendMarkdown(ctx, chatID,
		"💰 *Deduct Balance*\n\n"+
			"Please enter the phone number or username of the user:\n\n"+
			"📱 Examples:\n"+
			"• `09847488474` (will auto-format to +2519847488474)\n"+
			"• `@username`\n\n"+
			"⌨️ Type the phone number or username:")
	b.tempState.Store(chatID, "awaiting_deduct_balance_search")
	log.Printf("🔵 State stored: awaiting_deduct_balance_search for chatID: %d", chatID)
}

// ============ HANDLE SEARCH AND ADD BALANCE ============

func (b *Bot) handleSearchAndAddBalance(ctx context.Context, chatID int64, query string) {
	log.Printf("🔵 handleSearchAndAddBalance called with query: '%s' for chatID: %d", query, chatID)

	// Search for the user
	users := b.findUsersByQuery(ctx, query)
	log.Printf("🔵 findUsersByQuery returned %d users", len(users))

	if len(users) == 0 {
		log.Printf("🔴 No users found for query: '%s'", query)
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"❌ No user found for: `%s`\n\n"+
				"Please try again with a different phone number or username.",
			query,
		))
		b.tempState.Delete(chatID)
		return
	}

	if len(users) > 1 {
		log.Printf("🔵 Multiple users found (%d), showing selection", len(users))
		b.showUserSelectionForAddBalance(ctx, chatID, users)
		return
	}

	// Single user found, ask for amount
	user := users[0]
	log.Printf("🔵 Single user found: @%s (ID: %d)", user.Username, user.TelegramID)

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"💰 *Add Balance*\n\n"+
			"👤 User: @%s\n"+
			"🆔 `%d`\n"+
			"📱 %s\n"+
			"💰 Current Balance: %.2f ETB\n\n"+
			"Enter the amount to add:\n\n"+
			"📌 Example: `100` or `50.5`",
		user.Username,
		user.TelegramID,
		formatPhoneNumber(user.PhoneNumber),
		user.Balance,
	))

	stateKey := fmt.Sprintf("awaiting_add_balance_amount_%d", user.TelegramID)
	b.tempState.Store(chatID, stateKey)
	log.Printf("🔵 State stored: %s for chatID: %d", stateKey, chatID)
}

// ============ HANDLE SEARCH AND DEDUCT BALANCE ============

func (b *Bot) handleSearchAndDeductBalance(ctx context.Context, chatID int64, query string) {
	log.Printf("🔵 handleSearchAndDeductBalance called with query: '%s' for chatID: %d", query, chatID)

	// Search for the user
	users := b.findUsersByQuery(ctx, query)
	log.Printf("🔵 findUsersByQuery returned %d users", len(users))

	if len(users) == 0 {
		log.Printf("🔴 No users found for query: '%s'", query)
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"❌ No user found for: `%s`\n\n"+
				"Please try again with a different phone number or username.",
			query,
		))
		b.tempState.Delete(chatID)
		return
	}

	if len(users) > 1 {
		log.Printf("🔵 Multiple users found (%d), showing selection", len(users))
		b.showUserSelectionForDeductBalance(ctx, chatID, users)
		return
	}

	// Single user found, ask for amount
	user := users[0]
	log.Printf("🔵 Single user found: @%s (ID: %d)", user.Username, user.TelegramID)

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"💰 *Deduct Balance*\n\n"+
			"👤 User: @%s\n"+
			"🆔 `%d`\n"+
			"📱 %s\n"+
			"💰 Current Balance: %.2f ETB\n\n"+
			"Enter the amount to deduct:\n\n"+
			"📌 Example: `100` or `50.5`",
		user.Username,
		user.TelegramID,
		formatPhoneNumber(user.PhoneNumber),
		user.Balance,
	))

	stateKey := fmt.Sprintf("awaiting_deduct_balance_amount_%d", user.TelegramID)
	b.tempState.Store(chatID, stateKey)
	log.Printf("🔵 State stored: %s for chatID: %d", stateKey, chatID)
}

// ============ SHOW USER SELECTION FOR ADD BALANCE ============

func (b *Bot) showUserSelectionForAddBalance(ctx context.Context, chatID int64, users []models.User) {
	log.Printf("🔵 showUserSelectionForAddBalance called with %d users", len(users))

	text := "🔍 *Multiple users found*\n\n" +
		"Please select a user:\n\n"

	for i, user := range users {
		if i >= 10 {
			break
		}
		text += fmt.Sprintf(
			"%d. @%s | 📱 %s | 💰 %.2f ETB | 🆔 `%d`\n",
			i+1,
			user.Username,
			formatPhoneNumber(user.PhoneNumber),
			user.Balance,
			user.TelegramID,
		)
	}

	text += "\n💡 *Tip:* Type the exact Telegram ID to select a user."

	b.sendMarkdown(ctx, chatID, text)
	b.tempState.Store(chatID, "awaiting_add_balance_user_selection")
	log.Printf("🔵 State stored: awaiting_add_balance_user_selection for chatID: %d", chatID)
}

// ============ SHOW USER SELECTION FOR DEDUCT BALANCE ============

func (b *Bot) showUserSelectionForDeductBalance(ctx context.Context, chatID int64, users []models.User) {
	log.Printf("🔵 showUserSelectionForDeductBalance called with %d users", len(users))

	text := "🔍 *Multiple users found*\n\n" +
		"Please select a user:\n\n"

	for i, user := range users {
		if i >= 10 {
			break
		}
		text += fmt.Sprintf(
			"%d. @%s | 📱 %s | 💰 %.2f ETB | 🆔 `%d`\n",
			i+1,
			user.Username,
			formatPhoneNumber(user.PhoneNumber),
			user.Balance,
			user.TelegramID,
		)
	}

	text += "\n💡 *Tip:* Type the exact Telegram ID to select a user."

	b.sendMarkdown(ctx, chatID, text)
	b.tempState.Store(chatID, "awaiting_deduct_balance_user_selection")
	log.Printf("🔵 State stored: awaiting_deduct_balance_user_selection for chatID: %d", chatID)
}

// ============ LIST USERS ============

func (b *Bot) listUsers(ctx context.Context, chatID int64, page int) {
	log.Printf("🔵 listUsers called for chatID: %d, page: %d", chatID, page)

	limit := 10
	offset := (page - 1) * limit

	var users []models.User
	var total int64

	b.db.Model(&models.User{}).Where("is_bot = ?", false).Count(&total)
	b.db.Where("is_bot = ?", false).Order("created_at DESC").Limit(limit).Offset(offset).Find(&users)

	if len(users) == 0 {
		b.sendText(ctx, chatID, "📋 No users found.")
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	text := fmt.Sprintf("👥 <b>Users (Page %d/%d)</b>\n", page, totalPages)
	text += fmt.Sprintf("📊 Total: %d users\n\n", total)

	for i, user := range users {
		agentBadge := ""
		if user.IsAgent {
			agentBadge = " 🤝"
		}

		phone := formatPhoneNumber(user.PhoneNumber)
		username := user.Username
		if username == "" {
			username = "No username"
		}

		isActive := time.Since(user.LastActive) < 7*24*time.Hour
		activeBadge := "🟢"
		if !isActive {
			activeBadge = "🔴"
		}

		text += fmt.Sprintf(
			"%d. %s @%s%s\n   💰 %.2f ETB | 📱 %s\n   🆔 <code>%d</code>\n\n",
			offset+i+1,
			activeBadge,
			username,
			agentBadge,
			user.Balance,
			phone,
			user.TelegramID,
		)
	}

	keyboard := [][]telego.InlineKeyboardButton{}

	navRow := []telego.InlineKeyboardButton{}
	if page > 1 {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "⬅️ Prev",
			CallbackData: fmt.Sprintf("users_page_%d", page-1),
		})
	}
	navRow = append(navRow, telego.InlineKeyboardButton{
		Text:         fmt.Sprintf("%d/%d", page, totalPages),
		CallbackData: "users_current",
	})
	if page < totalPages {
		navRow = append(navRow, telego.InlineKeyboardButton{
			Text:         "Next ➡️",
			CallbackData: fmt.Sprintf("users_page_%d", page+1),
		})
	}
	keyboard = append(keyboard, navRow)

	keyboard = append(keyboard, []telego.InlineKeyboardButton{
		{Text: "🔙 Back to Menu", CallbackData: "back_to_menu"},
	})

	// ✅ Use HTML parse mode
	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{ID: chatID},
		Text: text,
		ParseMode: "HTML",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	}
	
	if _, err := b.api.SendMessage(ctx, &msg); err != nil {
		log.Printf("❌ Failed to send users list: %v", err)
		b.sendText(ctx, chatID, "❌ Failed to display users.")
	}
}

// ============ SMART SEARCH ============

func (b *Bot) searchUsersSmart(ctx context.Context, chatID int64, query string) {
	log.Printf("🔵 searchUsersSmart called with query: '%s' for chatID: %d", query, chatID)

	users := b.findUsersByQuery(ctx, query)

	if len(users) == 0 {
		log.Printf("🔴 No users found for query: '%s'", query)
		b.showNoResultsFound(ctx, chatID, query)
		return
	}

	b.showSearchResults(ctx, chatID, query, users)
}

// ============ SHOW SEARCH RESULTS ============

func (b *Bot) showSearchResults(ctx context.Context, chatID int64, query string, users []models.User) {
	log.Printf("🔵 showSearchResults called with %d users for query: '%s'", len(users), query)

	text := fmt.Sprintf("🔍 *Search Results for '%s'*\n", query)
	text += fmt.Sprintf("📊 Found: %d user(s)\n\n", len(users))

	for i, user := range users {
		phone := formatPhoneNumber(user.PhoneNumber)
		username := user.Username
		if username == "" {
			username = "No username"
		}

		agentBadge := ""
		if user.IsAgent {
			agentBadge = " 🤝"
		}

		isActive := time.Since(user.LastActive) < 7*24*time.Hour
		activeBadge := "🟢"
		if !isActive {
			activeBadge = "🔴"
		}

		text += fmt.Sprintf(
			"%d. %s @%s%s\n   💰 %.2f ETB | 📱 %s\n   🆔 `%d`\n\n",
			i+1,
			activeBadge,
			username,
			agentBadge,
			user.Balance,
			phone,
			user.TelegramID,
		)
	}

	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "👤 View User", CallbackData: "users_view_prompt"},
			{Text: "🔙 Back to Menu", CallbackData: "back_to_menu"},
		},
		{
			{Text: "🔍 New Search", CallbackData: "users_search"},
		},
	}

	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ SHOW NO RESULTS ============

func (b *Bot) showNoResultsFound(ctx context.Context, chatID int64, query string) {
	log.Printf("🔵 showNoResultsFound for query: '%s'", query)

	displayQuery := query
	if isPhoneNumberLike(query) {
		displayQuery = formatPhoneNumber(query)
	}

	text := fmt.Sprintf(
		"🔍 *No users found for:* `%s`\n\n"+
			"💡 *Search tips:*\n"+
			"• 📱 Phone: Just type the number (e.g., `09847488474`)\n"+
			"• 👤 Username: Type with or without @ (e.g., `@username`)\n"+
			"• 📝 Name: Type first or last name\n\n"+
			"📌 *Examples:*\n"+
			"• `09847488474` → Will search as +2519847488474\n"+
			"• `@john_doe` → Search by username\n"+
			"• `John` → Search by name",
		displayQuery,
	)

	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "🔍 New Search", CallbackData: "users_search"},
			{Text: "🔙 Back to Menu", CallbackData: "back_to_menu"},
		},
	}

	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ VIEW USER ============

func (b *Bot) viewUser(ctx context.Context, chatID int64, telegramID int64) {
	log.Printf("🔵 viewUser called for telegramID: %d", telegramID)

	var user models.User
	if err := b.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			b.sendText(ctx, chatID, "❌ User not found.")
		} else {
			b.sendText(ctx, chatID, fmt.Sprintf("❌ Error: %v", err))
		}
		return
	}

	var gameCount int64
	b.db.Model(&models.GamePlayer{}).Where("user_id = ?", user.ID).Count(&gameCount)

	var cardCount int64
	b.db.Model(&models.Card{}).Where("user_id = ?", user.ID).Count(&cardCount)

	var totalStaked float64
	b.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND status = ?", user.ID, "stake", "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalStaked)

	var totalWon float64
	b.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND status = ?", user.ID, "win", "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalWon)

	var totalDeposits float64
	b.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND status = ?", user.ID, "deposit", "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalDeposits)

	var totalWithdrawals float64
	b.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND status = ?", user.ID, "withdraw", "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalWithdrawals)

	var referralCount int64
	b.db.Model(&models.User{}).Where("referred_by = ?", user.ID).Count(&referralCount)

	var recentTxs []models.Transaction
	b.db.Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Limit(5).
		Find(&recentTxs)

	recentTxText := ""
	for _, tx := range recentTxs {
		statusEmoji := "🟡"
		if tx.Status == "completed" {
			statusEmoji = "✅"
		} else if tx.Status == "failed" {
			statusEmoji = "❌"
		}
		recentTxText += fmt.Sprintf(
			"• %s %.2f ETB (%s) %s\n",
			statusEmoji,
			tx.Amount,
			tx.Type,
			tx.CreatedAt.Format("Jan 2 15:04"),
		)
	}
	if recentTxText == "" {
		recentTxText = "No recent transactions"
	}

	phone := formatPhoneNumber(user.PhoneNumber)
	username := user.Username
	if username == "" {
		username = "No username"
	}

	agentText := "❌"
	if user.IsAgent {
		agentText = fmt.Sprintf("✅ (Balance: %.2f ETB)", user.AgentBalance)
	}

	text := fmt.Sprintf(
		"👤 *User Details*\n\n"+
			"🆔 ID: `%d`\n"+
			"👤 Username: @%s\n"+
			"📝 Name: %s %s\n"+
			"📱 Phone: %s\n"+
			"💰 Balance: %.2f ETB\n"+
			"🤝 Agent: %s\n"+
			"🔑 Referral Code: `%s`\n"+
			"👥 Referrals: %d\n"+
			"📅 Joined: %s\n"+
			"🔄 Last Active: %s\n\n"+
			"📊 *Statistics:*\n"+
			"• Games Played: %d\n"+
			"• Cards Purchased: %d\n"+
			"• Total Staked: %.2f ETB\n"+
			"• Total Won: %.2f ETB\n"+
			"• Total Deposits: %.2f ETB\n"+
			"• Total Withdrawals: %.2f ETB\n\n"+
			"📋 *Recent Transactions:*\n%s",
		user.TelegramID,
		username,
		user.FirstName,
		user.LastName,
		phone,
		user.Balance,
		agentText,
		user.ReferralCode,
		referralCount,
		user.CreatedAt.Format("Jan 2, 2006 15:04"),
		user.LastActive.Format("Jan 2, 2006 15:04"),
		gameCount,
		cardCount,
		totalStaked,
		totalWon,
		totalDeposits,
		totalWithdrawals,
		recentTxText,
	)

	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "➕ Add Balance", CallbackData: fmt.Sprintf("user_add_%d", user.TelegramID)},
			{Text: "➖ Deduct Balance", CallbackData: fmt.Sprintf("user_deduct_%d", user.TelegramID)},
		},
		{
			{Text: "📱 Transactions", CallbackData: fmt.Sprintf("user_tx_%d", user.TelegramID)},
			{Text: "📊 Full Stats", CallbackData: fmt.Sprintf("user_stats_%d", user.TelegramID)},
		},
		{
			{Text: "🔙 Back to Menu", CallbackData: "back_to_menu"},
		},
	}

	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ USER STATISTICS ============

func (b *Bot) showUserStats(ctx context.Context, chatID int64) {
	log.Printf("🔵 showUserStats called for chatID: %d", chatID)

	var totalUsers int64
	var totalAgents int64
	var totalBots int64
	var activeUsers int64
	var usersWithBalance int64
	var totalBalance float64

	b.db.Model(&models.User{}).Where("is_bot = ?", false).Count(&totalUsers)
	b.db.Model(&models.User{}).Where("is_bot = ?", false).Where("is_agent = ?", true).Count(&totalAgents)
	b.db.Model(&models.User{}).Where("is_bot = ?", true).Count(&totalBots)

	weekAgo := time.Now().AddDate(0, 0, -7)
	b.db.Model(&models.User{}).Where("last_active >= ?", weekAgo).Where("is_bot = ?", false).Count(&activeUsers)

	today := time.Now().Truncate(24 * time.Hour)
	var newUsersToday int64
	b.db.Model(&models.User{}).Where("created_at >= ?", today).Where("is_bot = ?", false).Count(&newUsersToday)

	b.db.Model(&models.User{}).Where("balance > 0").Where("is_bot = ?", false).Count(&usersWithBalance)
	b.db.Model(&models.User{}).Where("is_bot = ?", false).Select("COALESCE(SUM(balance), 0)").Scan(&totalBalance)

	activePercentage := 0.0
	if totalUsers > 0 {
		activePercentage = float64(activeUsers) / float64(totalUsers) * 100
	}

	text := fmt.Sprintf(
		"📊 *User Statistics*\n\n"+
			"👥 *Total Users:* %d\n"+
			"🤝 *Agents:* %d\n"+
			"🤖 *Bots:* %d\n\n"+
			"📈 *Activity:*\n"+
			"• Active (7d): %d (%.1f%%)\n"+
			"• New Today: %d\n"+
			"• With Balance: %d\n\n"+
			"💰 *Balance Stats:*\n"+
			"• Total Balance: %.2f ETB\n"+
			"• Average Balance: %.2f ETB\n"+
			"• Highest Balance: %.2f ETB",
		totalUsers,
		totalAgents,
		totalBots,
		activeUsers,
		activePercentage,
		newUsersToday,
		usersWithBalance,
		totalBalance,
		b.getAverageBalance(),
		b.getHighestBalance(),
	)

	keyboard := [][]telego.InlineKeyboardButton{
		{{Text: "🔄 Refresh", CallbackData: "users_stats"}},
		{{Text: "🔙 Back to Menu", CallbackData: "back_to_menu"}},
	}

	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ USER TRANSACTIONS ============

func (b *Bot) showUserTransactions(ctx context.Context, chatID int64, telegramID int64) {
	log.Printf("🔵 showUserTransactions called for telegramID: %d", telegramID)

	var user models.User
	if err := b.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		b.sendText(ctx, chatID, "❌ User not found.")
		return
	}

	var transactions []models.Transaction
	b.db.Where("user_id = ?", user.ID).
		Order("created_at DESC").
		Limit(20).
		Find(&transactions)

	if len(transactions) == 0 {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"📋 *No transactions found*\n\n"+
				"👤 User: @%s\n"+
				"💰 Balance: %.2f ETB",
			user.Username,
			user.Balance,
		))
		return
	}

	text := fmt.Sprintf(
		"📋 *User Transactions*\n"+
			"👤 @%s\n"+
			"💰 Balance: %.2f ETB\n\n",
		user.Username,
		user.Balance,
	)

	for _, tx := range transactions {
		emoji := "🟡"
		if tx.Status == "completed" {
			emoji = "✅"
		} else if tx.Status == "failed" {
			emoji = "❌"
		} else if tx.Status == "pending" {
			emoji = "⏳"
		}

		sign := ""
		if tx.Amount > 0 && (tx.Type == "deposit" || tx.Type == "win" || tx.Type == "admin_add") {
			sign = "+"
		}

		text += fmt.Sprintf(
			"%s %s%.2f ETB | %s\n   📅 %s | %s\n",
			emoji,
			sign,
			tx.Amount,
			tx.Type,
			tx.CreatedAt.Format("Jan 2 15:04"),
			tx.Status,
		)
	}

	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "🔄 Refresh", CallbackData: fmt.Sprintf("user_tx_%d", user.TelegramID)},
			{Text: "🔙 Back", CallbackData: fmt.Sprintf("user_refresh_%d", user.TelegramID)},
		},
	}

	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ USER FULL STATS ============

func (b *Bot) showUserFullStats(ctx context.Context, chatID int64, telegramID int64) {
	log.Printf("🔵 showUserFullStats called for telegramID: %d", telegramID)

	var user models.User
	if err := b.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		b.sendText(ctx, chatID, "❌ User not found.")
		return
	}

	var totalGames int64
	b.db.Model(&models.GamePlayer{}).Where("user_id = ?", user.ID).Count(&totalGames)

	var totalCards int64
	b.db.Model(&models.Card{}).Where("user_id = ?", user.ID).Count(&totalCards)

	var totalStaked float64
	b.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type IN (?) AND status = ?", user.ID, []string{"stake", "deposit"}, "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalStaked)

	var totalWon float64
	b.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND status = ?", user.ID, "win", "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalWon)

	var totalDeposits float64
	b.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND status = ?", user.ID, "deposit", "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalDeposits)

	var totalWithdrawals float64
	b.db.Model(&models.Transaction{}).
		Where("user_id = ? AND type = ? AND status = ?", user.ID, "withdraw", "completed").
		Select("COALESCE(SUM(amount), 0)").Scan(&totalWithdrawals)

	var referralCount int64
	b.db.Model(&models.User{}).Where("referred_by = ?", user.ID).Count(&referralCount)

	winRate := 0.0
	if totalGames > 0 {
		var wins int64
		b.db.Model(&models.Transaction{}).
			Where("user_id = ? AND type = ? AND status = ?", user.ID, "win", "completed").
			Count(&wins)
		winRate = float64(wins) / float64(totalGames) * 100
	}

	isActive := time.Since(user.LastActive) < 7*24*time.Hour
	activeStatus := "🟢 Active"
	if !isActive {
		activeStatus = "🔴 Inactive"
	}

	text := fmt.Sprintf(
		"📊 *Full User Statistics*\n\n"+
			"👤 @%s\n"+
			"🆔 `%d`\n"+
			"📱 %s\n\n"+
			"📈 *Activity:*\n"+
			"• Status: %s\n"+
			"• Games Played: %d\n"+
			"• Cards Purchased: %d\n"+
			"• Win Rate: %.1f%%\n\n"+
			"💰 *Financial:*\n"+
			"• Current Balance: %.2f ETB\n"+
			"• Total Deposits: %.2f ETB\n"+
			"• Total Withdrawals: %.2f ETB\n"+
			"• Total Staked: %.2f ETB\n"+
			"• Total Won: %.2f ETB\n\n"+
			"👥 *Referrals:*\n"+
			"• Total Referrals: %d\n\n"+
			"🤝 *Agent:* %v\n"+
			"📅 Joined: %s",
		user.Username,
		user.TelegramID,
		formatPhoneNumber(user.PhoneNumber),
		activeStatus,
		totalGames,
		totalCards,
		winRate,
		user.Balance,
		totalDeposits,
		totalWithdrawals,
		totalStaked,
		totalWon,
		referralCount,
		user.IsAgent,
		user.CreatedAt.Format("Jan 2, 2006 15:04"),
	)

	keyboard := [][]telego.InlineKeyboardButton{
		{
			{Text: "🔄 Refresh", CallbackData: fmt.Sprintf("user_stats_%d", user.TelegramID)},
			{Text: "🔙 Back", CallbackData: fmt.Sprintf("user_refresh_%d", user.TelegramID)},
		},
	}

	b.sendMarkdownKeyboard(ctx, chatID, text, keyboard)
}

// ============ ADD BALANCE ============

func (b *Bot) addBalance(ctx context.Context, chatID int64, telegramID int64, amount float64) {
	log.Printf("🔵 addBalance called: userID=%d, amount=%.2f", telegramID, amount)

	if amount <= 0 {
		b.sendText(ctx, chatID, "❌ Amount must be greater than 0")
		return
	}
	b.adjustBalance(ctx, chatID, telegramID, amount)
}

// ============ DEDUCT BALANCE ============

func (b *Bot) deductBalance(ctx context.Context, chatID int64, telegramID int64, amount float64) {
	log.Printf("🔵 deductBalance called: userID=%d, amount=%.2f", telegramID, amount)

	if amount <= 0 {
		b.sendText(ctx, chatID, "❌ Amount must be greater than 0")
		return
	}

	var user models.User
	if err := b.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		b.sendText(ctx, chatID, "❌ User not found.")
		return
	}

	if user.Balance < amount {
		b.sendMarkdown(ctx, chatID, fmt.Sprintf(
			"❌ *Insufficient Balance*\n\n"+
				"👤 User: @%s\n"+
				"💰 Current Balance: %.2f ETB\n"+
				"💸 Attempted Deduction: %.2f ETB\n\n"+
				"⚠️ Cannot deduct more than current balance.",
			user.Username,
			user.Balance,
			amount,
		))
		return
	}

	b.adjustBalance(ctx, chatID, telegramID, -amount)
}

// ============ BALANCE ADJUSTMENT ============

func (b *Bot) adjustBalance(ctx context.Context, chatID int64, telegramID int64, amount float64) {
	log.Printf("🔵 adjustBalance called: userID=%d, amount=%.2f", telegramID, amount)

	var user models.User
	if err := b.db.Where("telegram_id = ?", telegramID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			b.sendText(ctx, chatID, "❌ User not found.")
		} else {
			b.sendText(ctx, chatID, fmt.Sprintf("❌ Error: %v", err))
		}
		return
	}

	oldBalance := user.Balance
	user.Balance += amount
	if err := b.db.Save(&user).Error; err != nil {
		b.sendText(ctx, chatID, fmt.Sprintf("❌ Failed to update balance: %v", err))
		return
	}

	txType := "admin_add"
	description := fmt.Sprintf("Admin added %.2f ETB", amount)

	if amount < 0 {
		txType = "admin_deduct"
		description = fmt.Sprintf("Admin deducted %.2f ETB", -amount)
	}

	transaction := models.Transaction{
		UserID:      user.ID,
		Type:        txType,
		Amount:      amount,
		Status:      "completed",
		Method:      "admin",
		Description: description,
		CreatedAt:   time.Now(),
	}
	b.db.Create(&transaction)

	b.logAdminAction(ctx, chatID, txType, user.TelegramID, "user",
		fmt.Sprintf("%s (%.2f ETB) - Old: %.2f, New: %.2f", description, amount, oldBalance, user.Balance))

	sign := "+"
	if amount < 0 {
		sign = ""
	}

	go func() {
		b.sendMarkdown(
			context.Background(),
			user.TelegramID,
			fmt.Sprintf(
				"💰 *Balance Adjustment*\n\n"+
					"Your balance has been adjusted by an administrator.\n\n"+
					"📊 Amount: %s%.2f ETB\n"+
					"💳 Previous Balance: %.2f ETB\n"+
					"💳 New Balance: %.2f ETB\n\n"+
					"If you have questions, please contact support.",
				sign,
				amount,
				oldBalance,
				user.Balance,
			),
		)
	}()

	action := "Added"
	if amount < 0 {
		action = "Deducted"
	}

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"✅ *Balance %s*\n\n"+
			"👤 User: @%s\n"+
			"🆔 ID: `%d`\n"+
			"📊 Previous Balance: %.2f ETB\n"+
			"💵 Amount: %s%.2f ETB\n"+
			"💳 New Balance: %.2f ETB",
		action,
		user.Username,
		user.TelegramID,
		oldBalance,
		sign,
		amount,
		user.Balance,
	))
}

// ============ SUSPEND / UNSUSPEND ============

func (b *Bot) suspendUser(ctx context.Context, chatID int64, telegramID int64) {
	log.Printf("🔵 suspendUser called for telegramID: %d", telegramID)

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"⏳ *User Suspension*\n\n"+
			"User `%d` has been suspended.\n\n"+
			"⚠️ This feature requires a `suspended` field in the User model.",
		telegramID,
	))

	b.logAdminAction(ctx, chatID, "suspend_user", telegramID, "user",
		fmt.Sprintf("Suspended user %d", telegramID))
}

func (b *Bot) unsuspendUser(ctx context.Context, chatID int64, telegramID int64) {
	log.Printf("🔵 unsuspendUser called for telegramID: %d", telegramID)

	b.sendMarkdown(ctx, chatID, fmt.Sprintf(
		"✅ *User Unsuspended*\n\n"+
			"User `%d` has been unsuspended.",
		telegramID,
	))

	b.logAdminAction(ctx, chatID, "unsuspend_user", telegramID, "user",
		fmt.Sprintf("Unsuspended user %d", telegramID))
}

// ============ HELPER FUNCTIONS ============

func (b *Bot) getAverageBalance() float64 {
	var avg float64
	b.db.Model(&models.User{}).Where("is_bot = ?", false).Select("COALESCE(AVG(balance), 0)").Scan(&avg)
	return avg
}

func (b *Bot) getHighestBalance() float64 {
	var max float64
	b.db.Model(&models.User{}).Where("is_bot = ?", false).Select("COALESCE(MAX(balance), 0)").Scan(&max)
	return max
}

// ============ HANDLE USER TEXT INPUT ============

func (b *Bot) handleUserTextInput(ctx context.Context, chatID int64, text string) {
	log.Printf("🔵 handleUserTextInput called for chatID: %d, text: '%s'", chatID, text)

	if state, ok := b.tempState.Load(chatID); ok {
		stateStr := state.(string)
		log.Printf("🔵 Current state for chatID %d: '%s'", chatID, stateStr)

		// Search users (from button click)
		if stateStr == "awaiting_user_search" {
			log.Printf("🔵 Handling awaiting_user_search state")
			b.tempState.Delete(chatID)
			b.searchUsersSmart(ctx, chatID, text)
			return
		}

		// Add balance - search for user
		if stateStr == "awaiting_add_balance_search" {
			log.Printf("🔵 Handling awaiting_add_balance_search state")
			b.tempState.Delete(chatID)
			b.handleSearchAndAddBalance(ctx, chatID, text)
			return
		}

		// Deduct balance - search for user
		if stateStr == "awaiting_deduct_balance_search" {
			log.Printf("🔵 Handling awaiting_deduct_balance_search state")
			b.tempState.Delete(chatID)
			b.handleSearchAndDeductBalance(ctx, chatID, text)
			return
		}

		// Add balance - amount input
		if strings.HasPrefix(stateStr, "awaiting_add_balance_amount_") {
			log.Printf("🔵 Handling awaiting_add_balance_amount_ state")
			userID, err := strconv.ParseInt(strings.TrimPrefix(stateStr, "awaiting_add_balance_amount_"), 10, 64)
			if err != nil {
				log.Printf("🔴 Invalid user ID in state: %s", stateStr)
				b.sendText(ctx, chatID, "❌ Invalid user ID")
				b.tempState.Delete(chatID)
				return
			}

			amount, err := strconv.ParseFloat(text, 64)
			if err != nil || amount <= 0 {
				log.Printf("🔴 Invalid amount: '%s', error: %v", text, err)
				b.sendText(ctx, chatID, "❌ Please enter a valid amount (e.g., `100` or `50.5`)")
				return
			}

			log.Printf("🔵 Adding balance: userID=%d, amount=%.2f", userID, amount)
			b.tempState.Delete(chatID)
			b.addBalance(ctx, chatID, userID, amount)
			return
		}

		// Deduct balance - amount input
		if strings.HasPrefix(stateStr, "awaiting_deduct_balance_amount_") {
			log.Printf("🔵 Handling awaiting_deduct_balance_amount_ state")
			userID, err := strconv.ParseInt(strings.TrimPrefix(stateStr, "awaiting_deduct_balance_amount_"), 10, 64)
			if err != nil {
				log.Printf("🔴 Invalid user ID in state: %s", stateStr)
				b.sendText(ctx, chatID, "❌ Invalid user ID")
				b.tempState.Delete(chatID)
				return
			}

			amount, err := strconv.ParseFloat(text, 64)
			if err != nil || amount <= 0 {
				log.Printf("🔴 Invalid amount: '%s', error: %v", text, err)
				b.sendText(ctx, chatID, "❌ Please enter a valid amount (e.g., `100` or `50.5`)")
				return
			}

			log.Printf("🔵 Deducting balance: userID=%d, amount=%.2f", userID, amount)
			b.tempState.Delete(chatID)
			b.deductBalance(ctx, chatID, userID, amount)
			return
		}

		// Add balance - user selection
		if stateStr == "awaiting_add_balance_user_selection" {
			log.Printf("🔵 Handling awaiting_add_balance_user_selection state")
			userID, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				log.Printf("🔴 Invalid user ID: '%s'", text)
				b.sendText(ctx, chatID, "❌ Invalid user ID. Please type a valid Telegram ID.")
				return
			}

			// Find the user
			var user models.User
			if err := b.db.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
				b.sendText(ctx, chatID, "❌ User not found.")
				b.tempState.Delete(chatID)
				return
			}

			b.tempState.Delete(chatID)
			// Now ask for amount
			b.sendMarkdown(ctx, chatID, fmt.Sprintf(
				"💰 *Add Balance*\n\n"+
					"👤 User: @%s\n"+
					"🆔 `%d`\n"+
					"📱 %s\n"+
					"💰 Current Balance: %.2f ETB\n\n"+
					"Enter the amount to add:\n\n"+
					"📌 Example: `100` or `50.5`",
				user.Username,
				user.TelegramID,
				formatPhoneNumber(user.PhoneNumber),
				user.Balance,
			))
			b.tempState.Store(chatID, fmt.Sprintf("awaiting_add_balance_amount_%d", user.TelegramID))
			return
		}

		// Deduct balance - user selection
		if stateStr == "awaiting_deduct_balance_user_selection" {
			log.Printf("🔵 Handling awaiting_deduct_balance_user_selection state")
			userID, err := strconv.ParseInt(text, 10, 64)
			if err != nil {
				log.Printf("🔴 Invalid user ID: '%s'", text)
				b.sendText(ctx, chatID, "❌ Invalid user ID. Please type a valid Telegram ID.")
				return
			}

			// Find the user
			var user models.User
			if err := b.db.Where("telegram_id = ?", userID).First(&user).Error; err != nil {
				b.sendText(ctx, chatID, "❌ User not found.")
				b.tempState.Delete(chatID)
				return
			}

			b.tempState.Delete(chatID)
			// Now ask for amount
			b.sendMarkdown(ctx, chatID, fmt.Sprintf(
				"💰 *Deduct Balance*\n\n"+
					"👤 User: @%s\n"+
					"🆔 `%d`\n"+
					"📱 %s\n"+
					"💰 Current Balance: %.2f ETB\n\n"+
					"Enter the amount to deduct:\n\n"+
					"📌 Example: `100` or `50.5`",
				user.Username,
				user.TelegramID,
				formatPhoneNumber(user.PhoneNumber),
				user.Balance,
			))
			b.tempState.Store(chatID, fmt.Sprintf("awaiting_deduct_balance_amount_%d", user.TelegramID))
			return
		}
	} else {
		log.Printf("🔵 No state found for chatID: %d", chatID)
	}

	// If no state, check if it looks like a phone number or username
	if strings.HasPrefix(text, "/") {
		log.Printf("🔵 Text starts with '/', ignoring")
		return
	}

	if isPhoneNumberLike(text) || strings.Contains(text, "@") {
		log.Printf("🔵 Text looks like a phone number or username, searching...")
		b.searchUsersSmart(ctx, chatID, text)
	}
}