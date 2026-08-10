package bot

import (
	"context"
	"log"
	"strings"

	"github.com/mymmrac/telego"
)

func (b *Bot) handleCallback(
	ctx context.Context,
	callback *telego.CallbackQuery,
) {

	if callback == nil {
		return
	}

	log.Printf(
		"callback received: %s",
		callback.Data,
	)

	chatID := callback.Message.GetChat().ID

	// Handle deposit bank selection
	if strings.HasPrefix(callback.Data, "deposit_") {
		bank := strings.TrimPrefix(callback.Data, "deposit_")
		b.handleDepositBankSelection(ctx, chatID, bank)
	}

	// Handle back to menu
	if callback.Data == "back_to_menu" {
		b.sendMainMenu(ctx, chatID)
	}

	// Telegram requires answering callback queries.
	err := b.api.AnswerCallbackQuery(
		ctx,
		&telego.AnswerCallbackQueryParams{
			CallbackQueryID: callback.ID,
		},
	)

	if err != nil {
		log.Printf(
			"failed answering callback: %v",
			err,
		)
	}
}

// handleDepositBankSelection sends deposit info based on bank selection
func (b *Bot) handleDepositBankSelection(
	ctx context.Context,
	chatID int64,
	bank string,
) {
	switch bank {
	case "telebirr":
		b.sendMarkdown(
			ctx,
			chatID,
			"📱 *Telebirr Deposit*\n\n"+
				"Account:  `0936033937` — Frezer Wudeneh\n\n"+
				"━━━━━━━━━━━━━━━\n"+
				"📱 *Telebirr Deposit Steps*\n\n"+
				"1️⃣ ከላይ ባለው የ Telebirr አካውንት ገንዘቡን ያስገቡ።\n"+
				"2️⃣ ክፍያ ካደረጉ በኋላ የ Telebirr የጹሁፍ መልክት (SMS) ይደርሳችኋል።\n"+
				"3️⃣ የደረሳችሁን SMS ሙሉ በሙሉ ኮፒ (copy) በማረግ በዚህ ቻት ፔስት (paste) አድርጉ።\n\n"+
				"💬 የክፍያ ችግር ካለ፣ @babibingosupport ያናግሩ።\n\n"+
				"━━━━━━━━━━━━━━━\n"+
				"📤 After sending payment, please paste the SMS confirmation below 👇",
		)

	case "cbebirr":
		b.sendMarkdown(
			ctx,
			chatID,
			"🏦 *CBEBirr Deposit*\n\n"+
				"Account: `0936033937` — Frezer Wudeneh\n\n"+
				"━━━━━━━━━━━━━━━\n"+
				"📱 *CBEBirr Deposit Steps*\n\n"+
				"1️⃣ ከላይ ባለው የ CBEBirr አካውንት ገንዘቡን ያስገቡ።\n"+
				"2️⃣ ክፍያ ካደረጉ በኋላ የ CBEBirr የጹሁፍ መልክት (SMS) ይደርሳችኋል።\n"+
				"3️⃣ የደረሳችሁን SMS ሙሉ በሙሉ ኮፒ (copy) በማረግ በዚህ ቻት ፔስት (paste) አድርጉ።\n\n"+
				"💬 የክፍያ ችግር ካለ፣ @babibingosupport ያናግሩ።\n\n"+
				"━━━━━━━━━━━━━━━\n"+
				"📤 After sending payment, please paste the SMS confirmation below 👇",
		)

	default:
		b.sendText(ctx, chatID, "❌ Invalid selection. Please try again.")
		b.handleDeposit(ctx, chatID)
	}
}