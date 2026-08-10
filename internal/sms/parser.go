package sms

import (
	"regexp"
	"strconv"
	"strings"
)

// TransactionInfo represents parsed transaction details
type TransactionInfo struct {
	Amount          float64
	TransactionID   string
	Sender          string
	Receiver        string
	ReceiverPhone   string
	Date            string
	Balance         float64
	Fee             float64
	VAT             float64
	IsValid         bool
}

// ParseTelebirrSMS parses Telebirr SMS messages (supports both English and Amharic)
func ParseTelebirrSMS(smsText string) *TransactionInfo {
	result := &TransactionInfo{
		IsValid: false,
	}

	// Try English patterns first
	result = parseEnglishTelebirr(smsText)
	if result.IsValid {
		return result
	}

	// Try Amharic patterns
	result = parseAmharicTelebirr(smsText)
	if result.IsValid {
		return result
	}

	return result
}

// parseEnglishTelebirr parses English Telebirr SMS
func parseEnglishTelebirr(smsText string) *TransactionInfo {
	result := &TransactionInfo{
		IsValid: false,
	}

	// Extract transaction number
	txnRegex := regexp.MustCompile(`transaction number is ([A-Z0-9]+)`)
	if matches := txnRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.TransactionID = matches[1]
	}

	// Extract amount
	amountRegex := regexp.MustCompile(`transferred ETB ([0-9.]+) to`)
	if matches := amountRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if amount, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Amount = amount
		}
	}

	// Extract sender
	senderRegex := regexp.MustCompile(`Dear ([A-Za-z ]+)`)
	if matches := senderRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.Sender = strings.TrimSpace(matches[1])
	}

	// Extract receiver
	receiverRegex := regexp.MustCompile(`to ([A-Za-z ]+) \([0-9]+\*+[0-9]+\)`)
	if matches := receiverRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.Receiver = strings.TrimSpace(matches[1])
	}

	// Extract balance
	balanceRegex := regexp.MustCompile(`balance is ETB ([0-9.]+)`)
	if matches := balanceRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if balance, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Balance = balance
		}
	}

	// Extract fee
	feeRegex := regexp.MustCompile(`service charge ([0-9.]+)`)
	if matches := feeRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if fee, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Fee = fee
		}
	}

	// Check if we have enough info
	if result.TransactionID != "" && result.Amount > 0 {
		result.IsValid = true
	}

	return result
}

// parseAmharicTelebirr parses Amharic Telebirr SMS
func parseAmharicTelebirr(smsText string) *TransactionInfo {
	result := &TransactionInfo{
		IsValid: false,
	}

	// የሂሳብ እንቅስቃሴ ቁጥርዎ [CODE] ነዉ።
	// Extract transaction number (Amharic)
	txnRegex := regexp.MustCompile(`ቁጥርዎ\s+([A-Z0-9]+)\s+ነዉ`)
	if matches := txnRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.TransactionID = matches[1]
	}

	// ወደ [Receiver] ([Phone]) [Amount] ብር በ [Date] ልከዋል።
	// Extract receiver, phone, amount, date (Amharic)
	
	// Extract receiver name and phone
	receiverRegex := regexp.MustCompile(`ወደ\s+([\p{L} ]+)\s*\(([0-9]+\*+[0-9]+)\)`)
	if matches := receiverRegex.FindStringSubmatch(smsText); len(matches) > 2 {
		result.Receiver = strings.TrimSpace(matches[1])
		result.ReceiverPhone = matches[2]
	}

	// Extract amount (Amharic) - matches "50.00 ብር"
	amountRegex := regexp.MustCompile(`([0-9.]+)\s*ብር`)
	if matches := amountRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if amount, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Amount = amount
		}
	}

	// Extract date (Amharic) - matches "03/08/2026 16:52:28"
	dateRegex := regexp.MustCompile(`(\d{2}/\d{2}/\d{4}\s+\d{2}:\d{2}:\d{2})`)
	if matches := dateRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.Date = matches[1]
	}

	// Extract balance (Amharic) - matches "ቀሪ ሂሳብ [Amount] ብር"
	balanceRegex := regexp.MustCompile(`ቀሪ ሂሳብ\s+([0-9.]+)\s*ብር`)
	if matches := balanceRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if balance, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Balance = balance
		}
	}

	// Extract fee (Amharic) - matches "ክፍያው [Amount] ብር"
	feeRegex := regexp.MustCompile(`አገልግሎት ክፍያው\s+([0-9.]+)\s*ብር`)
	if matches := feeRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if fee, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Fee = fee
		}
	}

	// Extract VAT (Amharic) - matches "15% VAT [Amount] ብር"
	vatRegex := regexp.MustCompile(`15%\s*VAT\s+([0-9.]+)\s*ብር`)
	if matches := vatRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if vat, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.VAT = vat
		}
	}

	// Try to extract sender from "ውድ [Sender]" at the beginning
	senderRegex := regexp.MustCompile(`^ውድ\s+([\p{L} ]+)`)
	if matches := senderRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.Sender = strings.TrimSpace(matches[1])
	}

	// If we have transaction ID and amount, it's valid
	if result.TransactionID != "" && result.Amount > 0 {
		result.IsValid = true
	}

	return result
}

// IsTelebirrSMS checks if the SMS is from Telebirr (supports both English and Amharic)
func IsTelebirrSMS(smsText string) bool {
	// English keywords
	englishKeywords := []string{
		"telebirr",
		"Ethio telecom",
		"transferred ETB",
		"transaction number is",
	}
	
	// Amharic keywords
	amharicKeywords := []string{
		"ቴሌብር",
		"ኢትዮ ቴሌኮም",
		"ብር",
		"የሂሳብ እንቅስቃሴ ቁጥር",
		"ልከዋል",
		"ቀሪ ሂሳብ",
	}
	
	lowerText := strings.ToLower(smsText)
	
	// Check English keywords
	for _, keyword := range englishKeywords {
		if strings.Contains(lowerText, strings.ToLower(keyword)) {
			return true
		}
	}
	
	// Check Amharic keywords (no lowercasing needed for Amharic)
	for _, keyword := range amharicKeywords {
		if strings.Contains(smsText, keyword) {
			return true
		}
	}
	
	return false
}

// GetTelebirrType determines if the SMS is English or Amharic
func GetTelebirrType(smsText string) string {
	if strings.Contains(smsText, "ቴሌብር") || strings.Contains(smsText, "ብር") {
		return "amharic"
	}
	if strings.Contains(strings.ToLower(smsText), "telebirr") {
		return "english"
	}
	return "unknown"
}