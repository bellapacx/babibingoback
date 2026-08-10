package adminbot

import (
	"regexp"
	"strings"
)

// ============ PHONE NUMBER UTILITIES ============

// formatPhoneNumber formats a phone number to a standard format
func formatPhoneNumber(phone string) string {
	if phone == "" {
		return "Not set"
	}

	re := regexp.MustCompile(`[^0-9+]`)
	phone = re.ReplaceAllString(phone, "")

	if strings.HasPrefix(phone, "0") {
		phone = phone[1:]
	}

	phone = strings.TrimPrefix(phone, "+")

	if strings.HasPrefix(phone, "251") {
		return "+" + phone
	}

	if strings.HasPrefix(phone, "9") && len(phone) == 9 {
		return "+251" + phone
	}

	if len(phone) == 10 && strings.HasPrefix(phone, "9") {
		return "+251" + phone
	}

	if len(phone) == 9 {
		return "+251" + phone
	}

	if !strings.HasPrefix(phone, "251") && len(phone) > 0 {
		return "+251" + phone
	}

	return "+" + phone
}

// getPhoneVariations returns various phone number formats
func (b *Bot) getPhoneVariations(phone string) []string {
	clean := regexp.MustCompile(`[^0-9]`).ReplaceAllString(phone, "")
	variations := []string{}

	if strings.HasPrefix(clean, "0") {
		clean = clean[1:]
	}

	if strings.HasPrefix(clean, "251") {
		variations = append(variations, "+"+clean)
		variations = append(variations, clean)
		variations = append(variations, "0"+clean[3:])
	} else if strings.HasPrefix(clean, "9") && len(clean) == 9 {
		variations = append(variations, "+251"+clean)
		variations = append(variations, "251"+clean)
		variations = append(variations, "0"+clean)
	} else {
		variations = append(variations, "+251"+clean)
		variations = append(variations, "251"+clean)
		variations = append(variations, "0"+clean)
	}

	return variations
}