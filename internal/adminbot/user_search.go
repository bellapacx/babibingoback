package adminbot

import (
	"context"
	"log"
	"regexp"
	"strings"

	"babibingo/internal/models"
)

// ============ SEARCH UTILITIES ============

// findUsersByQuery searches for users by phone number, username, or name
func (b *Bot) findUsersByQuery(ctx context.Context, query string) []models.User {
	log.Printf("🔵 findUsersByQuery called with query: '%s'", query)
	
	var users []models.User
	
	// Format the query
	formattedQuery := b.formatSearchQuery(query)
	log.Printf("🔵 Formatted query: '%s'", formattedQuery)
	
	// Try exact phone number match (both with and without +)
	if formattedQuery != "" {
		log.Printf("🔵 Trying exact phone number match with: '%s'", formattedQuery)
		
		// Try with + prefix
		b.db.Where("phone_number = ?", formattedQuery).
			Where("is_bot = ?", false).
			Find(&users)
		if len(users) > 0 {
			log.Printf("✅ Found %d user(s) with exact match: '%s'", len(users), formattedQuery)
			return users
		}
		
		// Try without + prefix (for database entries without +)
		phoneWithoutPlus := strings.TrimPrefix(formattedQuery, "+")
		log.Printf("🔵 Trying without + prefix: '%s'", phoneWithoutPlus)
		b.db.Where("phone_number = ?", phoneWithoutPlus).
			Where("is_bot = ?", false).
			Find(&users)
		if len(users) > 0 {
			log.Printf("✅ Found %d user(s) without + prefix: '%s'", len(users), phoneWithoutPlus)
			return users
		}
		
		// Try with + prefix if the database has it
		if !strings.HasPrefix(formattedQuery, "+") {
			phoneWithPlus := "+" + formattedQuery
			log.Printf("🔵 Trying with + prefix: '%s'", phoneWithPlus)
			b.db.Where("phone_number = ?", phoneWithPlus).
				Where("is_bot = ?", false).
				Find(&users)
			if len(users) > 0 {
				log.Printf("✅ Found %d user(s) with + prefix: '%s'", len(users), phoneWithPlus)
				return users
			}
		}
	}
	
	// Try partial phone match (search for the number without +)
	if formattedQuery != "" {
		searchPhone := strings.TrimPrefix(formattedQuery, "+")
		log.Printf("🔵 Trying partial phone match with: '%%%s%%'", searchPhone)
		b.db.Where("phone_number ILIKE ?", "%"+searchPhone+"%").
			Where("is_bot = ?", false).
			Find(&users)
		if len(users) > 0 {
			log.Printf("✅ Found %d user(s) with partial phone match: '%s'", len(users), searchPhone)
			return users
		}
	}
	
	// Try username or name search
	searchPattern := "%" + strings.TrimPrefix(query, "@") + "%"
	log.Printf("🔵 Trying username/name search with pattern: '%s'", searchPattern)
	b.db.Where("username ILIKE ? OR first_name ILIKE ? OR last_name ILIKE ?",
		searchPattern, searchPattern, searchPattern).
		Where("is_bot = ?", false).
		Order("created_at DESC").
		Limit(20).
		Find(&users)
	
	log.Printf("🔵 Username/name search returned %d users", len(users))
	return users
}

// formatSearchQuery formats a search query
func (b *Bot) formatSearchQuery(query string) string {
	log.Printf("🔵 formatSearchQuery called with: '%s'", query)
	
	query = strings.TrimSpace(query)
	query = strings.TrimPrefix(query, "@")

	if isPhoneNumberLike(query) {
		formatted := formatPhoneNumber(query)
		log.Printf("🔵 formatSearchQuery: phone number detected, formatted to: '%s'", formatted)
		return formatted
	}

	log.Printf("🔵 formatSearchQuery: returning as-is: '%s'", query)
	return query
}

// isPhoneNumberLike checks if a string looks like a phone number
func isPhoneNumberLike(text string) bool {
	clean := regexp.MustCompile(`[^0-9+]`).ReplaceAllString(text, "")

	digitCount := 0
	for _, char := range clean {
		if char >= '0' && char <= '9' {
			digitCount++
		}
	}

	return digitCount >= 8
}