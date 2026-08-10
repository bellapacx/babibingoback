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

// internal/bot/user.go - handlePhoneShare

func (b *Bot) handlePhoneShare(
	ctx context.Context,
	chatID int64,
	tgUser *telego.User,
	contact *telego.Contact,
) {
	var existing models.User

	err := b.db.
		Where("telegram_id = ?", tgUser.ID).
		First(&existing).
		Error

	if err == nil {
		b.sendMainMenu(ctx, chatID)
		return
	}

	// ✅ Generate referral code for this user
	referralCode := generateReferralCode()

	// ✅ Check if there's a referral code from the start command
	var referredBy *int64
	if cachedCode, ok := b.tempReferralCache.Load(chatID); ok {
		referrerCode := cachedCode.(string)
		var referrer models.User
		
		// ✅ Handle both "ref_" prefix and plain referral codes
		if strings.HasPrefix(referrerCode, "ref_") {
			// Format: ref_7762372471
			refStr := strings.TrimPrefix(referrerCode, "ref_")
			if referrerTelegramID, err := strconv.ParseInt(refStr, 10, 64); err == nil {
				// Find by Telegram ID
				if err := b.db.Where("telegram_id = ?", referrerTelegramID).First(&referrer).Error; err == nil {
					referredBy = &referrer.ID
					log.Printf("✅ User %d referred by agent %d (from ref_ format)", tgUser.ID, referrer.ID)
				}
			}
		} else {
			// Legacy: find by referral code (e.g., "ABC123XYZ")
			if err := b.db.Where("referral_code = ?", referrerCode).First(&referrer).Error; err == nil {
				referredBy = &referrer.ID
				log.Printf("✅ User %d referred by agent %d (from referral_code format)", tgUser.ID, referrer.ID)
			}
		}
		
		b.tempReferralCache.Delete(chatID) // Clean up
	}

	// ✅ Create user with referral info
	user := models.User{
		TelegramID:   int64(tgUser.ID),
		PhoneNumber:  contact.PhoneNumber,
		Username:     tgUser.Username,
		FirstName:    tgUser.FirstName,
		LastName:     tgUser.LastName,
		ReferralCode: referralCode,
		ReferredBy:   referredBy, // ✅ Store who referred this user
		Balance:      0,
		AgentBalance: 0,
		IsAgent:      false,
		LastActive:   time.Now(),
		CreatedAt:    time.Now(),
	}

	if err := b.db.Create(&user).Error; err != nil {
		log.Printf("failed creating user: %v", err)
		b.sendText(ctx, chatID, "❌ Registration failed. Please try again.")
		return
	}

	// ✅ If user was referred, notify the referrer
	if referredBy != nil {
		var referrer models.User
		if err := b.db.First(&referrer, *referredBy).Error; err == nil {
			// ✅ Only notify if the referrer is an agent
			if referrer.IsAgent {
				b.sendMarkdown(
					ctx,
					referrer.TelegramID, // Send to agent
					fmt.Sprintf(
						"🎉 *New Agent Referral!*\n\n"+
							"User @%s has registered using your invitation link!\n\n"+
							"💰 You will earn 1 ETB commission for every card they play.",
						tgUser.Username,
					),
				)
			} else {
				log.Printf("📨 User %d referred by non-agent %d (no commission)", tgUser.ID, *referredBy)
			}
		}
	}

	// ✅ Send welcome message with referral info
	b.sendMarkdown(
		ctx,
		chatID,
		fmt.Sprintf(
			"✅ *Registration successful!*\n\n"+
				"🎱 Welcome to BabiBingo!\n",
		),
	)

	b.sendMainMenu(ctx, chatID)
}