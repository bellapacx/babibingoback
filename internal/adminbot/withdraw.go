package adminbot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"babibingo/internal/models"

	"github.com/mymmrac/telego"
	"gorm.io/gorm"
)

// handleWithdrawals - Main withdrawal handler
func (b *Bot) handleWithdrawals(ctx context.Context, chatID int64, args []string) {
    if len(args) == 0 {
        b.showPendingWithdrawals(ctx, chatID)
        return
    }

    switch args[0] {
    case "pending":
        b.showPendingWithdrawals(ctx, chatID)
    case "all":
        b.showAllWithdrawals(ctx, chatID)
    case "approve":
        if len(args) > 1 {
            id := args[1]
            b.approveWithdraw(ctx, chatID, id)
        } else {
            b.sendText(ctx, chatID, "❌ Usage: /withdrawals approve <transaction_id>")
        }
    case "reject":
        if len(args) > 1 {
            id := args[1]
            b.rejectWithdraw(ctx, chatID, id)
        } else {
            b.sendText(ctx, chatID, "❌ Usage: /withdrawals reject <transaction_id>")
        }
    case "search":
        if len(args) > 1 {
            query := strings.Join(args[1:], " ")
            b.searchWithdrawals(ctx, chatID, query)
        } else {
            b.sendText(ctx, chatID, "❌ Usage: /withdrawals search <user>")
        }
    case "stats":
        b.showWithdrawalStats(ctx, chatID)
    default:
        b.sendText(ctx, chatID, "❌ Usage: /withdrawals [pending|all|approve <id>|reject <id>|search <query>|stats]")
    }
}

// showPendingWithdrawals - Show all pending withdrawals
func (b *Bot) showPendingWithdrawals(ctx context.Context, chatID int64) {
    var withdrawals []models.Transaction
    b.db.Where("type = ? AND status = ?", "withdraw", "pending").
        Order("created_at ASC").
        Find(&withdrawals)

    if len(withdrawals) == 0 {
        b.sendText(ctx, chatID, "📋 No pending withdrawals.")
        return
    }

    b.sendMarkdown(ctx, chatID, fmt.Sprintf(
        "🏧 *Pending Withdrawals*\n\nTotal: %d", len(withdrawals),
    ))

    for _, withdrawal := range withdrawals {
        b.sendWithdrawalCard(ctx, chatID, withdrawal)
    }
}

// showAllWithdrawals - Show all withdrawals (paginated)
func (b *Bot) showAllWithdrawals(ctx context.Context, chatID int64) {
    var withdrawals []models.Transaction
    b.db.Where("type = ?", "withdraw").
        Order("created_at DESC").
        Limit(20).
        Find(&withdrawals)

    if len(withdrawals) == 0 {
        b.sendText(ctx, chatID, "📋 No withdrawals found.")
        return
    }

    b.sendMarkdown(ctx, chatID, fmt.Sprintf(
        "🏧 *Recent Withdrawals (Last 20)*\n\n"+
            "🟡 Pending | ✅ Completed | ❌ Failed",
    ))

    for _, withdrawal := range withdrawals {
        statusEmoji := "🟡"
        if withdrawal.Status == "completed" {
            statusEmoji = "✅"
        } else if withdrawal.Status == "failed" {
            statusEmoji = "❌"
        }

        var user models.User
        b.db.First(&user, withdrawal.UserID)

        b.sendText(
            ctx,
            chatID,
            fmt.Sprintf(
                "%s %.2f ETB | @%s | 📱%s | %s | Ref: %s",
                statusEmoji,
                withdrawal.Amount,
                user.Username,
                user.PhoneNumber,
                withdrawal.CreatedAt.Format("Jan 2, 15:04"),
                withdrawal.Reference,
            ),
        )
    }
}

// approveWithdraw - Approve a pending withdrawal
func (b *Bot) approveWithdraw(ctx context.Context, chatID int64, transactionID string) {
    var withdrawal models.Transaction
    err := b.db.Where("id = ? AND type = ? AND status = ?", transactionID, "withdraw", "pending").
        First(&withdrawal).Error

    if err != nil {
        if err == gorm.ErrRecordNotFound {
            b.sendText(ctx, chatID, "❌ Withdrawal not found or already processed.")
        } else {
            b.sendText(ctx, chatID, fmt.Sprintf("❌ Error: %v", err))
        }
        return
    }

    // Get user
    var user models.User
    if err := b.db.First(&user, withdrawal.UserID).Error; err != nil {
        b.sendText(ctx, chatID, "❌ User not found.")
        return
    }

    // Check if user has enough balance
    if user.Balance < withdrawal.Amount {
        b.sendMarkdown(ctx, chatID, fmt.Sprintf(
            "❌ *Insufficient Balance*\n\n"+
                "User @%s has insufficient balance.\n"+
                "Requested: %.2f ETB\n"+
                "Balance: %.2f ETB",
            user.Username,
            withdrawal.Amount,
            user.Balance,
        ))
        return
    }

    // Start transaction
    tx := b.db.Begin()

    // Update withdrawal status
    withdrawal.Status = "completed"
    if err := tx.Save(&withdrawal).Error; err != nil {
        tx.Rollback()
        b.sendText(ctx, chatID, "❌ Failed to update withdrawal status.")
        return
    }

    // Deduct from user balance
    user.Balance -= withdrawal.Amount
    if err := tx.Save(&user).Error; err != nil {
        tx.Rollback()
        b.sendText(ctx, chatID, "❌ Failed to update user balance.")
        return
    }

    // Commit transaction
    if err := tx.Commit().Error; err != nil {
        b.sendText(ctx, chatID, "❌ Failed to commit transaction.")
        return
    }

    // ✅ Notify the user
    b.sendMarkdown(
        ctx,
        user.TelegramID,
        fmt.Sprintf(
            "✅ *Withdrawal Approved!*\n\n"+
                "💰 Amount: %.2f ETB\n"+
                "🆔 Reference: `%s`\n"+
                "💳 New Balance: %.2f ETB\n\n"+
                "🏦 Funds will be sent to your account (%s) shortly.",
            withdrawal.Amount,
            withdrawal.Reference,
            user.Balance,
            user.PhoneNumber,
        ),
    )

    // Log admin action
    b.logAdminAction(ctx, chatID, "approve_withdraw", withdrawal.UserID, "withdraw",
        fmt.Sprintf("Approved withdrawal %.2f ETB for user %d", withdrawal.Amount, withdrawal.UserID))

    // Confirm to admin with phone number
    b.sendMarkdown(ctx, chatID, fmt.Sprintf(
        "✅ *Withdrawal Approved*\n\n"+
            "💰 Amount: %.2f ETB\n"+
            "👤 User: @%s\n"+
            "📱 Phone: %s\n"+
            "🆔 Reference: `%s`\n"+
            "💳 New Balance: %.2f ETB",
        withdrawal.Amount,
        user.Username,
        user.PhoneNumber,
        withdrawal.Reference,
        user.Balance,
    ))
}

// rejectWithdraw - Reject a pending withdrawal
func (b *Bot) rejectWithdraw(ctx context.Context, chatID int64, transactionID string) {
    var withdrawal models.Transaction
    err := b.db.Where("id = ? AND type = ? AND status = ?", transactionID, "withdraw", "pending").
        First(&withdrawal).Error

    if err != nil {
        if err == gorm.ErrRecordNotFound {
            b.sendText(ctx, chatID, "❌ Withdrawal not found or already processed.")
        } else {
            b.sendText(ctx, chatID, fmt.Sprintf("❌ Error: %v", err))
        }
        return
    }

    // Get user
    var user models.User
    if err := b.db.First(&user, withdrawal.UserID).Error; err != nil {
        b.sendText(ctx, chatID, "❌ User not found.")
        return
    }

    // Update withdrawal status
    withdrawal.Status = "failed"
    if err := b.db.Save(&withdrawal).Error; err != nil {
        b.sendText(ctx, chatID, "❌ Failed to update withdrawal status.")
        return
    }

    // ✅ Notify the user
    b.sendMarkdown(
        ctx,
        user.TelegramID,
        fmt.Sprintf(
            "❌ *Withdrawal Rejected*\n\n"+
                "💰 Amount: %.2f ETB\n"+
                "🆔 Reference: `%s`\n\n"+
                "Please contact support for more information.",
            withdrawal.Amount,
            withdrawal.Reference,
        ),
    )

    // Log admin action
    b.logAdminAction(ctx, chatID, "reject_withdraw", withdrawal.UserID, "withdraw",
        fmt.Sprintf("Rejected withdrawal %.2f ETB for user %d", withdrawal.Amount, withdrawal.UserID))

    // Confirm to admin with phone number
    b.sendMarkdown(ctx, chatID, fmt.Sprintf(
        "❌ *Withdrawal Rejected*\n\n"+
            "💰 Amount: %.2f ETB\n"+
            "👤 User: @%s\n"+
            "📱 Phone: %s\n"+
            "🆔 Reference: `%s`",
        withdrawal.Amount,
        user.Username,
        user.PhoneNumber,
        withdrawal.Reference,
    ))
}

// searchWithdrawals - Search withdrawals by user
func (b *Bot) searchWithdrawals(ctx context.Context, chatID int64, query string) {
    var withdrawals []models.Transaction

    // Try to find user by username, phone number, or telegram_id
    var user models.User
    userFound := false
    
    // Search by @username
    if strings.HasPrefix(query, "@") {
        username := strings.TrimPrefix(query, "@")
        if err := b.db.Where("username = ?", username).First(&user).Error; err == nil {
            userFound = true
        }
    } else if strings.HasPrefix(query, "09") && len(query) >= 10 {
        // Search by phone number
        if err := b.db.Where("phone_number = ?", query).First(&user).Error; err == nil {
            userFound = true
        }
    } else if id, err := strconv.ParseInt(query, 10, 64); err == nil {
        // Search by Telegram ID
        if err := b.db.Where("telegram_id = ?", id).First(&user).Error; err == nil {
            userFound = true
        }
    }

    if userFound {
        b.db.Where("type = ? AND user_id = ?", "withdraw", user.ID).
            Order("created_at DESC").
            Limit(20).
            Find(&withdrawals)
    } else {
        // Search by reference
        b.db.Where("type = ? AND reference ILIKE ?", "withdraw", "%"+query+"%").
            Order("created_at DESC").
            Limit(20).
            Find(&withdrawals)
    }

    if len(withdrawals) == 0 {
        b.sendText(ctx, chatID, fmt.Sprintf("📋 No withdrawals found for: %s", query))
        return
    }

    b.sendMarkdown(ctx, chatID, fmt.Sprintf(
        "🔍 *Search Results for '%s'*\n\nFound: %d withdrawals",
        query, len(withdrawals),
    ))

    for _, withdrawal := range withdrawals {
        var u models.User
        b.db.First(&u, withdrawal.UserID)

        statusEmoji := "🟡"
        if withdrawal.Status == "completed" {
            statusEmoji = "✅"
        } else if withdrawal.Status == "failed" {
            statusEmoji = "❌"
        }

        b.sendText(
            ctx,
            chatID,
            fmt.Sprintf(
                "%s %.2f ETB | @%s | 📱%s | %s | Ref: %s",
                statusEmoji,
                withdrawal.Amount,
                u.Username,
                u.PhoneNumber,
                withdrawal.CreatedAt.Format("Jan 2, 15:04"),
                withdrawal.Reference,
            ),
        )
    }
}

// showWithdrawalStats - Show withdrawal statistics
func (b *Bot) showWithdrawalStats(ctx context.Context, chatID int64) {
    var totalWithdrawals float64
    var totalCount int64
    var pendingWithdrawals float64
    var pendingCount int64
    var completedWithdrawals float64
    var completedCount int64

    b.db.Model(&models.Transaction{}).
        Where("type = ?", "withdraw").
        Select("COALESCE(SUM(amount), 0)").Scan(&totalWithdrawals)
    b.db.Model(&models.Transaction{}).
        Where("type = ?", "withdraw").
        Count(&totalCount)

    b.db.Model(&models.Transaction{}).
        Where("type = ? AND status = ?", "withdraw", "pending").
        Select("COALESCE(SUM(amount), 0)").Scan(&pendingWithdrawals)
    b.db.Model(&models.Transaction{}).
        Where("type = ? AND status = ?", "withdraw", "pending").
        Count(&pendingCount)

    b.db.Model(&models.Transaction{}).
        Where("type = ? AND status = ?", "withdraw", "completed").
        Select("COALESCE(SUM(amount), 0)").Scan(&completedWithdrawals)
    b.db.Model(&models.Transaction{}).
        Where("type = ? AND status = ?", "withdraw", "completed").
        Count(&completedCount)

    // Get today's withdrawals
    today := time.Now().Truncate(24 * time.Hour)
    var todayWithdrawals float64
    var todayCount int64
    b.db.Model(&models.Transaction{}).
        Where("type = ? AND status = ? AND created_at >= ?", "withdraw", "completed", today).
        Select("COALESCE(SUM(amount), 0)").Scan(&todayWithdrawals)
    b.db.Model(&models.Transaction{}).
        Where("type = ? AND status = ? AND created_at >= ?", "withdraw", "completed", today).
        Count(&todayCount)

    b.sendMarkdown(
        ctx,
        chatID,
        fmt.Sprintf(
            "📊 *Withdrawal Statistics*\n\n"+
                "💰 *Total Withdrawals:*\n"+
                "• Amount: %.2f ETB\n"+
                "• Count: %d\n\n"+
                "🟡 *Pending Withdrawals:*\n"+
                "• Amount: %.2f ETB\n"+
                "• Count: %d\n\n"+
                "✅ *Completed Withdrawals:*\n"+
                "• Amount: %.2f ETB\n"+
                "• Count: %d\n\n"+
                "📅 *Today's Withdrawals:*\n"+
                "• Amount: %.2f ETB\n"+
                "• Count: %d",
            totalWithdrawals,
            totalCount,
            pendingWithdrawals,
            pendingCount,
            completedWithdrawals,
            completedCount,
            todayWithdrawals,
            todayCount,
        ),
    )
}

// sendWithdrawalCard - Send a detailed withdrawal card with phone number
func (b *Bot) sendWithdrawalCard(ctx context.Context, chatID int64, withdrawal models.Transaction) {
    var user models.User
    b.db.First(&user, withdrawal.UserID)

    statusEmoji := "🟡"
    statusText := "Pending"
    if withdrawal.Status == "completed" {
        statusEmoji = "✅"
        statusText = "Completed"
    } else if withdrawal.Status == "failed" {
        statusEmoji = "❌"
        statusText = "Failed"
    }

    text := fmt.Sprintf(
        "🏧 *Withdrawal #%s*\n\n"+
            "👤 User: @%s\n"+
            "🆔 ID: %d\n"+
            "📱 Phone: %s\n"+
            "💰 Amount: %.2f ETB\n"+
            "📱 Method: %s\n"+
            "🆔 Reference: `%s`\n"+
            "📅 Date: %s\n"+
            "📊 Status: %s %s",
        withdrawal.ID.String()[:8],
        user.Username,
        user.TelegramID,
        user.PhoneNumber,
        withdrawal.Amount,
        withdrawal.Method,
        withdrawal.Reference,
        withdrawal.CreatedAt.Format("Jan 2, 2006 15:04"),
        statusEmoji,
        statusText,
    )

    if withdrawal.Status == "pending" {
        msg := telego.SendMessageParams{
            ChatID:      telego.ChatID{ID: chatID},
            Text:        text,
            ParseMode:   "Markdown",
            ReplyMarkup: b.transactionActionKeyboard(withdrawal.ID.String(), "withdraw"),
        }
        b.sendMessage(ctx, &msg)
    } else {
        // ✅ Include phone number in completed/failed cards too
        b.sendMarkdown(ctx, chatID, text)
    }
}