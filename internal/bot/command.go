package bot

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"babibingo/internal/models"

	"github.com/mymmrac/telego"
)

// internal/bot/command.go - handleCommand

func (b *Bot) handleCommand(
    ctx context.Context,
    chatID int64,
    user *telego.User,
    text string,
) {
    // Parse command and arguments
    parts := strings.Split(strings.TrimPrefix(text, "/"), " ")
    command := parts[0]
    
    // ✅ Extract referral code if present (e.g., /start ABC123)
    referralCode := ""
    if len(parts) > 1 {
        referralCode = parts[1]
    }

    switch command {
    case "start":
        b.handleStart(ctx, chatID, user, referralCode) // ✅ Pass referral code

    case "play":
        b.handlePlay(ctx, chatID)

    case "balance":
        b.handleBalance(ctx, chatID, user)

    case "deposit":
        b.handleDeposit(ctx, chatID)

    case "withdraw":
        b.handleWithdraw(ctx, chatID,user)

    case "agent":
        b.handleAgent(ctx, chatID, user)

    case "invite":
        b.handleInvite(ctx, chatID, user)

    case "support":
        b.handleSupport(ctx, chatID)

    default:
        b.sendText(ctx, chatID, "Unknown command. Use the menu below.")
        b.sendMainMenu(ctx, chatID)
    }
}
// internal/bot/command.go - handleStart

func (b *Bot) handleStart(
    ctx context.Context,
    chatID int64,
    user *telego.User,
    referralCode string, // ✅ Add referral code parameter
) {
    var existing models.User
    err := b.db.Where("telegram_id = ?", user.ID).First(&existing).Error

    if err == nil {
        b.sendMainMenu(ctx, chatID)
        return
    }

    // ✅ Store referral code temporarily (will be used after phone registration)
    // We'll store in context or a map
    if referralCode != "" {
        b.tempReferralCache.Store(chatID, referralCode)
    }

    // Ask for phone number
    msg := telego.SendMessageParams{
        ChatID: telego.ChatID{
            ID: chatID,
        },
        Text: "📋 *Registration Process*\n\n" +
            "Please share your phone number to register automatically.",
        ParseMode: "Markdown",
        ReplyMarkup: b.contactKeyboard(),
    }

    b.sendMessage(ctx, &msg)
}
func (b *Bot) handlePlay(
	ctx context.Context,
	chatID int64,
) {

	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{
			ID: chatID,
		},

		Text:
		"🎱 *BabiBingo*\n\n" +
			"Click below to open the game.",

		ParseMode: "Markdown",

		ReplyMarkup: b.playKeyboard(),
	}


	b.sendMessage(
		ctx,
		&msg,
	)
}
func (b *Bot) handleBalance(
	ctx context.Context,
	chatID int64,
	user *telego.User,
) {

	var u models.User


	err := b.db.
		Where(
			"telegram_id = ?",
			user.ID,
		).
		First(&u).
		Error


	if err != nil {

		b.sendText(
			ctx,
			chatID,
			"❌ Please register first with /start",
		)

		return
	}


	b.sendMarkdown(
		ctx,
		chatID,
		fmt.Sprintf(
			"💰 *Your Balance*\n\n%.2f ETB",
			u.Balance,
		),
	)
}
// handleDeposit handles deposit requests with bank selection
func (b *Bot) handleDeposit(
	ctx context.Context,
	chatID int64,
) {
	// ✅ First ask user to choose their bank
	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{
			ID: chatID,
		},
		Text: "💳 *Select Deposit Method*\n\n" +
			"Choose your preferred payment method:",
		ParseMode: "Markdown",
		ReplyMarkup: b.bankSelectionKeyboard(),
	}

	b.sendMessage(ctx, &msg)
}

// ✅ New: Bank selection keyboard
func (b *Bot) bankSelectionKeyboard() *telego.InlineKeyboardMarkup {
	return &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{
					Text:         "📱 Telebirr",
					CallbackData: "deposit_telebirr",
				},
				{
					Text:         "🏦 CBE Birr",
					CallbackData: "deposit_cbebirr",
				},
			},
			{
				{
					Text:         "🔙 Back",
					CallbackData: "back_to_menu",
				},
			},
		},
	}
}



// ✅ Telebirr deposit info
func (b *Bot) sendTelebirrDepositInfo(
	ctx context.Context,
	chatID int64,
) {
	b.sendMarkdown(
		ctx,
		chatID,
		"📱 *Telebirr Deposit*\n\n"+
			"1️⃣ Open Telebirr app\n"+
			"2️⃣ Send money to:\n"+
			"   `0940072277` — BabiBingo\n"+
			"3️⃣ Copy the confirmation code\n"+
			"4️⃣ Submit code in WebApp\n\n"+
			"💰 *Minimum Deposit:* 20 ETB\n"+
			"⏱️ *Processing Time:* 1-2 minutes\n\n"+
			"⚠️ Include your Telegram username in the reference.",
	)
}

// ✅ CBE Birr deposit info
func (b *Bot) sendCBEBirrDepositInfo(
	ctx context.Context,
	chatID int64,
) {
	b.sendMarkdown(
		ctx,
		chatID,
		"🏦 *CBE Birr Deposit*\n\n"+
			"1️⃣ Open CBE Birr app\n"+
			"2️⃣ Send money to:\n"+
			"   Account: `1000123456789`\n"+
			"   Name: BabiBingo\n"+
			"   Bank: Commercial Bank of Ethiopia\n"+
			"3️⃣ Copy the transaction reference\n"+
			"4️⃣ Submit reference in WebApp\n\n"+
			"💰 *Minimum Deposit:* 20 ETB\n"+
			"⏱️ *Processing Time:* 1-5 minutes\n\n"+
			"⚠️ Include your Telegram username in the reference.",
	)
}
// internal/bot/command.go - handleWithdraw

func (b *Bot) handleWithdraw(
	ctx context.Context,
	chatID int64,
	user *telego.User,
) {
	// ✅ Get user from database
	var dbUser models.User
	if err := b.db.Where("telegram_id = ?", user.ID).First(&dbUser).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Please register first with /start")
		return
	}

	// ✅ Check minimum balance
	minWithdraw := 100.0
	if dbUser.Balance < minWithdraw {
		b.sendMarkdown(
			ctx,
			chatID,
			fmt.Sprintf(
				"❌ *Insufficient Balance*\n\n"+
					"Your current balance: %.2f ETB\n"+
					"Minimum withdrawal: %.2f ETB\n\n"+
					"💡 Please play more or deposit to reach the minimum withdrawal amount.",
				dbUser.Balance,
				minWithdraw,
			),
		)
		return
	}

	// ✅ Prompt for amount
	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{
			ID: chatID,
		},
		Text: fmt.Sprintf(
			"🏧 *Withdrawal Request*\n\n"+
				"💰 Your balance: %.2f ETB\n"+
				"📉 Minimum withdrawal: %.2f ETB\n"+
				"📈 Maximum withdrawal: %.2f ETB\n\n"+
				"📝 Please enter the amount you want to withdraw.\n\n"+
				"Example: `100`",
			dbUser.Balance,
			minWithdraw,
			dbUser.Balance,
		),
		ParseMode: "Markdown",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
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

	// ✅ Store the user's state to expect amount input
	b.tempState.Store(chatID, "awaiting_withdraw_amount")
}

// ✅ Handle the withdrawal amount input
func (b *Bot) handleWithdrawAmount(
	ctx context.Context,
	chatID int64,
	user *telego.User,
	text string,
) {
	// Parse amount
	amount, err := strconv.ParseFloat(text, 64)
	if err != nil || amount <= 0 {
		b.sendText(ctx, chatID, "❌ Invalid amount. Please enter a valid number.\n\nExample: `100`")
		return
	}

	// ✅ Get user from database
	var dbUser models.User
	if err := b.db.Where("telegram_id = ?", user.ID).First(&dbUser).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Please register first with /start")
		return
	}

	// ✅ Validate amount
	minWithdraw := 100.0
	if amount < minWithdraw {
		b.sendMarkdown(
			ctx,
			chatID,
			fmt.Sprintf(
				"❌ *Amount Too Low*\n\n"+
					"Amount: %.2f ETB\n"+
					"Minimum withdrawal: %.2f ETB\n\n"+
					"Please enter a higher amount.",
				amount,
				minWithdraw,
			),
		)
		return
	}

	if amount > dbUser.Balance {
		b.sendMarkdown(
			ctx,
			chatID,
			fmt.Sprintf(
				"❌ *Insufficient Balance*\n\n"+
					"Your balance: %.2f ETB\n"+
					"Requested: %.2f ETB\n\n"+
					"Please enter a lower amount.",
				dbUser.Balance,
				amount,
			),
		)
		return
	}

	// ✅ Create pending withdrawal transaction
	transaction := models.Transaction{
		UserID:      dbUser.ID,
		Type:        "withdraw",
		Amount:      amount,
		Status:      "pending",
		Method:      "telebirr",
		Reference:   fmt.Sprintf("WTH-%d-%d", dbUser.ID, time.Now().UnixNano()),
		Description: fmt.Sprintf("Withdrawal request for %.2f ETB", amount),
		CreatedAt:   time.Now(),
	}

	if err := b.db.Create(&transaction).Error; err != nil {
		log.Printf("Failed to create withdrawal transaction: %v", err)
		b.sendText(ctx, chatID, "❌ Failed to process withdrawal request. Please try again.")
		return
	}

	// ✅ Send confirmation to user
	b.sendMarkdown(
		ctx,
		chatID,
		fmt.Sprintf(
			"✅ *Withdrawal Request Submitted*\n\n"+
				"💰 Amount: %.2f ETB\n"+
				"🆔 Reference: `%s`\n"+
				"📊 Status: Pending\n\n"+
				"⏳ Your request is being processed by our team.\n"+
				"You will be notified once it's approved.\n\n"+
				"📌 If you have any questions, please contact support.",
			amount,
			transaction.Reference,
		),
	)

	// ✅ Send notification to admin (1929724270)
	go b.sendWithdrawalNotificationToAdmin(ctx, dbUser, transaction)

	// ✅ Clear the user's state
	b.tempState.Delete(chatID)
}

// ✅ Send withdrawal notification to admin
// sendWithdrawalNotificationToAdmin - Send withdrawal notification to admin
func (b *Bot) sendWithdrawalNotificationToAdmin(ctx context.Context, user models.User, transaction models.Transaction) {
	adminID := int64(1929724270) // Admin Telegram ID
	
	// Format user details
	username := user.Username
	if username == "" {
		username = "No username"
	}
	
	phone := user.PhoneNumber
	if phone == "" {
		phone = "Not set"
	}
	
	// Create notification message
	message := fmt.Sprintf(
		"💰 *New Withdrawal Request*\n\n"+
		"📋 *Transaction Details:*\n"+
		"• 🆔 Reference: `%s`\n"+
		"• 💰 Amount: %.2f ETB\n"+
		"• 📊 Status: Pending\n"+
		"• 📅 Time: %s\n\n"+
		"👤 *User Details:*\n"+
		"• 🆔 Telegram ID: `%d`\n"+
		"• 👤 Username: @%s\n"+
		"• 📱 Phone: %s\n"+
		"• 💳 Current Balance: %.2f ETB",
		transaction.Reference,
		transaction.Amount,
		transaction.CreatedAt.Format("2006-01-02 15:04:05"),
		user.TelegramID,
		username,
		phone,
		user.Balance,
	)
	
	// Send to admin
	b.sendMarkdown(ctx, adminID, message)
}

// ✅ Alternative: Send notification to all admins

// handleAgent handles agent-related actions
// internal/bot/command.go - handleAgent

func (b *Bot) handleAgent(
	ctx context.Context,
	chatID int64,
	user *telego.User,
) {
	var u models.User

	err := b.db.
		Where("telegram_id = ?", user.ID).
		First(&u).
		Error

	if err != nil {
		b.sendText(
			ctx,
			chatID,
			"❌ Please register first with /start",
		)
		return
	}

	// If user is already an agent, show dashboard
	if u.IsAgent {
		var referralCount int64
		b.db.Model(&models.User{}).Where("referred_by = ?", u.ID).Count(&referralCount)

		var totalCommission float64
		b.db.Model(&models.Transaction{}).
			Where("user_id = ? AND type = ?", u.ID, "agent_commission").
			Select("COALESCE(SUM(amount), 0)").
			Scan(&totalCommission)

		b.sendMarkdown(
			ctx,
			chatID,
			fmt.Sprintf(
				"🤝 *Agent Dashboard*\n\n"+
					"💰 Agent Balance: %.2f ETB\n"+
					"📊 Total Earned: %.2f ETB\n"+
					"👥 Referrals: %d\n"+
					"🔑 Referral Code: `%s`\n\n"+
					"📤 Share your referral link to earn commissions!\n"+
					"1 ETB per card played by your invited users.\n\n"+
					"🔗 https://t.me/babibingo\\_bot?start=ref\\_%d",
				u.AgentBalance,
				totalCommission,
				referralCount,
				u.ReferralCode,
				u.TelegramID,
			),
		)
		return
	}

	// ✅ Fixed: Clean markdown without special characters issues
	b.sendMarkdown(
		ctx,
		chatID,
		"💼 *Become a BabiBingo Agent*\n\n"+
			"Earn commissions every time your invited players play!\n\n"+
			"💰 *Commission:* 1 ETB per card played by your invited users\n\n"+
			"📝 *To become an agent:*\n"+
			"1️⃣ Click the button below to open the Agent Bot\n"+
			"2️⃣ Submit your request\n"+
			"3️⃣ Wait for admin approval\n"+
			"4️⃣ Once approved, you'll get access to your agent dashboard\n\n"+
			"🤖 *Apply here:* @babibingoagent_bot",
	)

	// Send with inline button
	msg := telego.SendMessageParams{
		ChatID: telego.ChatID{
			ID: chatID,
		},
		Text: "👇 Click below to apply as an agent:",
		ReplyMarkup: &telego.InlineKeyboardMarkup{
			InlineKeyboard: [][]telego.InlineKeyboardButton{
				{
					{
						Text: "🤝 Apply as Agent",
						URL:  "https://t.me/babibingoagent_bot",
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
// internal/bot/command.go - handleInvite

// internal/bot/command.go - handleInvite

// internal/bot/command.go - handleInvite

func (b *Bot) handleInvite(
	ctx context.Context,
	chatID int64,
	user *telego.User,
) {
	var u models.User
	if err := b.db.Where("telegram_id = ?", user.ID).First(&u).Error; err != nil {
		b.sendText(ctx, chatID, "❌ Please register first with /start")
		return
	}

	// ✅ Escape the underscore with backslash
	link := fmt.Sprintf(
		"https://t.me/babibingo\\_bot?start=ref\\_%d",
		user.ID,
	)

	b.sendMarkdown(
		ctx,
		chatID,
		fmt.Sprintf(
			"🔗 *Your Invitation Link*\n\n"+
				"Share this link with your friends!\n\n"+
				"%s",
			link,
		),
	)
}
func (b *Bot) handleSupport(
	ctx context.Context,
	chatID int64,
) {

	b.sendMarkdown(
		ctx,
		chatID,
		"🆘 *Support*\n\nContact: @babibingo_support",
	)
}