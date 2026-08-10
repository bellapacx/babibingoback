package bot

import (
	"context"
	"log"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/mymmrac/telego"
)

func generateReferralCode() string {

	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	rand.Seed(time.Now().UnixNano())

	code := make([]byte, 6)

	for i := range code {
		code[i] = letters[rand.Intn(len(letters))]
	}

	return string(code)
}


func cleanCommand(text string) string {

	text = strings.TrimSpace(text)

	text = strings.TrimPrefix(
		text,
		"/",
	)

	if index := strings.Index(
		text,
		" ",
	); index != -1 {

		text = text[:index]
	}

	return text
}
// formatPhoneNumber - Formats phone number with +251 prefix
func formatPhoneNumber(phone string) string {
	if phone == "" {
		return "Not set"
	}

	// Remove all non-digit characters except +
	re := regexp.MustCompile(`[^0-9+]`)
	phone = re.ReplaceAllString(phone, "")

	// Remove leading 0 if present
	if strings.HasPrefix(phone, "0") {
		phone = phone[1:]
	}

	// Remove leading + if present (we'll add it back)
	phone = strings.TrimPrefix(phone, "+")

	// If starts with 251, format as +251...
	if strings.HasPrefix(phone, "251") {
		return "+" + phone
	}

	// If starts with 9, add +251
	if strings.HasPrefix(phone, "9") && len(phone) == 9 {
		return "+251" + phone
	}

	// If length is 10 (like 098xxxxxxx), remove first 0 and add +251
	if len(phone) == 10 && strings.HasPrefix(phone, "9") {
		return "+251" + phone
	}

	// If length is 9, add +251
	if len(phone) == 9 {
		return "+251" + phone
	}

	// Default: try to add +251
	if !strings.HasPrefix(phone, "251") && len(phone) > 0 {
		return "+251" + phone
	}

	return "+" + phone
}
// sendMarkdownKeyboard - Send a markdown message with inline keyboard
func (b *Bot) sendMarkdownKeyboard(ctx context.Context, chatID int64, text string, keyboard [][]telego.InlineKeyboardButton) {
	params := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: chatID},
		Text:      text,
		ParseMode: "Markdown",
	}

	if len(keyboard) > 0 {
		params.ReplyMarkup = &telego.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		}
	}

	_, err := b.api.SendMessage(ctx, params)
	if err != nil {
		log.Printf("Failed to send markdown keyboard message to %d: %v", chatID, err)
	}
}