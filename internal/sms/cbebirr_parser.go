// internal/sms/cbebirr_parser.go

package sms

import (
	"regexp"
	"strconv"
	"strings"
)

// CBEBirrTransaction represents parsed CBE Birr transaction details
type CBEBirrTransaction struct {
	Amount          float64
	TransactionID   string // Txn ID
	ReferenceNumber string // Reference number (same as Txn ID)
	Sender          string
	Receiver        string
	ReceiverPhone   string
	Date            string
	PhoneNumber     string
	ReceiptURL      string
	Balance         float64
	Fee             float64
	VAT             float64
	IsValid         bool
}

// ParseCBEBirrSMS parses CBE Birr SMS messages (supports both English and Amharic)
func ParseCBEBirrSMS(smsText string) *CBEBirrTransaction {
	result := &CBEBirrTransaction{
		IsValid: false,
	}

	// Try English patterns first
	result = parseEnglishCBEBirr(smsText)
	if result.IsValid {
		return result
	}

	// Try Amharic patterns (same as Telebirr Amharic format)
	result = parseAmharicCBEBirr(smsText)
	if result.IsValid {
		return result
	}

	return result
}

// parseEnglishCBEBirr parses English CBE Birr SMS
func parseEnglishCBEBirr(smsText string) *CBEBirrTransaction {
	result := &CBEBirrTransaction{
		IsValid: false,
	}

	// Extract transaction ID / Reference Number
	txnRegex := regexp.MustCompile(`(?:Txn ID|TID[=\s]+)([A-Z0-9]+)`)
	if matches := txnRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.TransactionID = matches[1]
		result.ReferenceNumber = matches[1]
	}

	// Extract amount
	amountRegex := regexp.MustCompile(`sent\s+([0-9.]+)\s*Br`)
	if matches := amountRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if amount, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Amount = amount
		}
	}

	// Extract sender
	senderRegex := regexp.MustCompile(`Dear\s+([A-Za-z ]+)`)
	if matches := senderRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.Sender = strings.TrimSpace(matches[1])
	}

	// Extract receiver
	receiverRegex := regexp.MustCompile(`to\s+([A-Za-z ]+)`)
	if matches := receiverRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.Receiver = strings.TrimSpace(matches[1])
	}

	// Extract receiver phone
	phoneRegex := regexp.MustCompile(`to\s+[A-Za-z ]+\s+\(([0-9]+)\)`)
	if matches := phoneRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.ReceiverPhone = matches[1]
	}

	// Extract date
	dateRegex := regexp.MustCompile(`(\d{2}/\d{2}/\d{2}\s+\d{2}:\d{2})`)
	if matches := dateRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.Date = matches[1]
	}

	// Extract phone number from URL
	phoneURLRegex := regexp.MustCompile(`PH=([0-9]+)`)
	if matches := phoneURLRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.PhoneNumber = matches[1]
	}

	// Extract receipt URL
	urlRegex := regexp.MustCompile(`https://cbepay1\.cbe\.com\.et/aureceipt\?TID=[A-Z0-9]+&PH=[0-9]+`)
	if matches := urlRegex.FindString(smsText); matches != "" {
		result.ReceiptURL = matches
	}

	// Extract balance
	balanceRegex := regexp.MustCompile(`balance is ETB ([0-9.]+)`)
	if matches := balanceRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if balance, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Balance = balance
		}
	}

	// Validate
	if result.TransactionID != "" && result.Amount > 0 {
		result.IsValid = true
	}

	return result
}

// parseAmharicCBEBirr parses Amharic CBE Birr SMS (same format as Telebirr Amharic)
func parseAmharicCBEBirr(smsText string) *CBEBirrTransaction {
	result := &CBEBirrTransaction{
		IsValid: false,
	}

	// የሂሳብ እንቅስቃሴ ቁጥርዎ [CODE] ነዉ።
	// Extract transaction number (Amharic)
	txnRegex := regexp.MustCompile(`ቁጥርዎ\s+([A-Z0-9]+)\s+ነዉ`)
	if matches := txnRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.TransactionID = matches[1]
		result.ReferenceNumber = matches[1]
	}

	// ወደ [Receiver] ([Phone]) [Amount] ብር በ [Date] ልከዋል።
	// Extract receiver name and phone
	receiverRegex := regexp.MustCompile(`ወደ\s+([\p{L} ]+)\s*\(([0-9]+\*+[0-9]+)\)`)
	if matches := receiverRegex.FindStringSubmatch(smsText); len(matches) > 2 {
		result.Receiver = strings.TrimSpace(matches[1])
		result.ReceiverPhone = matches[2]
	}

	// Extract amount (Amharic) - matches "20.00 ብር"
	amountRegex := regexp.MustCompile(`([0-9.]+)\s*ብር`)
	if matches := amountRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		if amount, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Amount = amount
		}
	}

	// Extract date (Amharic) - matches "01/08/2026 17:27:46"
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

	// Extract fee (Amharic) - matches "አገልግሎት ክፍያው [Amount] ብር"
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

	// Extract sender from "ውድ[Sender]" at the beginning
	senderRegex := regexp.MustCompile(`^ውድ\s*([\p{L} ]+)`)
	if matches := senderRegex.FindStringSubmatch(smsText); len(matches) > 1 {
		result.Sender = strings.TrimSpace(matches[1])
	}

	// Extract receipt URL
	urlRegex := regexp.MustCompile(`https://transactioninfo\.ethiotelecom\.et/receipt/([A-Z0-9]+)`)
	if matches := urlRegex.FindString(smsText); matches != "" {
		result.ReceiptURL = matches
	}

	// Extract phone number from URL or from receiver phone
	if result.ReceiverPhone != "" {
		// Clean phone number (remove * and keep only digits)
		phone := strings.ReplaceAll(result.ReceiverPhone, "*", "")
		if len(phone) > 4 {
			result.PhoneNumber = phone
		}
	}

	// If we have transaction ID and amount, it's valid
	if result.TransactionID != "" && result.Amount > 0 {
		result.IsValid = true
	}

	return result
}

// IsCBEBirrSMS checks if the SMS is from CBE Birr (supports both English and Amharic)
func IsCBEBirrSMS(smsText string) bool {
	// English keywords
	englishKeywords := []string{
		"CBE Birr",
		"cbepay1.cbe.com.et",
		"sent.*Br",
		"Txn ID",
	}

	lowerText := strings.ToLower(smsText)
	for _, keyword := range englishKeywords {
		if strings.Contains(lowerText, strings.ToLower(keyword)) {
			return true
		}
	}

	// Amharic keywords (same as Telebirr since they use same format)
	amharicKeywords := []string{
		"ብር",
		"የሂሳብ እንቅስቃሴ ቁጥር",
		"ልከዋል",
		"ቀሪ ሂሳብ",
		"ኢትዮ ቴሌኮም",
	}

	for _, keyword := range amharicKeywords {
		if strings.Contains(smsText, keyword) {
			return true
		}
	}

	return false
}

// GetCBEBirrType determines if the SMS is English or Amharic
func GetCBEBirrType(smsText string) string {
	if strings.Contains(smsText, "ብር") || strings.Contains(smsText, "ልከዋል") {
		return "amharic"
	}
	if strings.Contains(strings.ToLower(smsText), "cbe birr") {
		return "english"
	}
	return "unknown"
}