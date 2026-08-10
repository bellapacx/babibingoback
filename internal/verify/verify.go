package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type VerifyClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// ✅ Unified request format for all banks
type VerifyRequest struct {
	Bank              string `json:"bank"`
	TransactionNumber string `json:"transactionNumber,omitempty"`
	ReferenceNumber   string `json:"referenceNumber,omitempty"`
	ReceiptNumber     string `json:"receiptNumber,omitempty"`     // ✅ Added for CBE Birr
	AccountSuffix     string `json:"accountSuffix,omitempty"`
	PhoneNumber       string `json:"phoneNumber,omitempty"`
	SettlementAccount string `json:"settlementAccount,omitempty"`
}

// ✅ Unified response format
type VerifyResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
	Data      []struct {
		Bank              string  `json:"bank"`
		Status            string  `json:"status"`
		Verified          bool    `json:"verified"`
		Amount            float64 `json:"amount"`
		Currency          string  `json:"currency"`
		SenderName        string  `json:"senderName"`
		ReceiverName      string  `json:"receiverName"`
		ReceiverAccount   string  `json:"receiverAccount"`
		TransactionNumber string  `json:"transactionNumber"`
		ReferenceNumber   string  `json:"referenceNumber"`
		ReceiptNumber     string  `json:"receiptNumber"`
		AccountSuffix     string  `json:"accountSuffix"`
		Timestamp         string  `json:"timestamp"`
		SettlementAccountMatch struct {
			Matched          bool   `json:"matched"`
			MatchType        string `json:"matchType"`
			MatchConfidence  string `json:"matchConfidence"`
			ReceiverAccount  string `json:"receiverAccount"`
			MatchedSettlementAccount string `json:"matchedSettlementAccount"`
		} `json:"settlementAccountMatch"`
	} `json:"data"`
	Verification struct {
		RequestID        string `json:"requestId"`
		ProcessingStatus string `json:"processingStatus"`
		Status           string `json:"status"`
		Verified         bool   `json:"verified"`
	} `json:"verification"`
}

func NewVerifyClient(apiKey string) *VerifyClient {
	return &VerifyClient{
		BaseURL: "https://verify.et",
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ✅ VerifyTeleBirrTransaction - Verify Telebirr transaction
func (c *VerifyClient) VerifyTeleBirrTransaction(txnID string, amount float64, receiverPhone string) (*VerifyResponse, error) {
	reqBody := VerifyRequest{
		Bank:              "telebirr",
		TransactionNumber: txnID,
		SettlementAccount: receiverPhone,
	}
	return c.doVerify(reqBody, txnID)
}

// ✅ VerifyCBEBirrTransaction - Verify CBE Birr transaction
// Uses receiptNumber (which is the transaction ID) + phoneNumber
func (c *VerifyClient) VerifyCBEBirrTransaction(receiptNumber string, phoneNumber string, amount float64) (*VerifyResponse, error) {
	reqBody := VerifyRequest{
		Bank:          "cbebirr",
		ReceiptNumber: receiptNumber, // ✅ Use receiptNumber instead of referenceNumber
		PhoneNumber:   phoneNumber,
	}
	return c.doVerify(reqBody, receiptNumber)
}

// ✅ Unified verify method
func (c *VerifyClient) doVerify(reqBody VerifyRequest, idempotencyKey string) (*VerifyResponse, error) {
	url := fmt.Sprintf("%s/api/verify?waitMs=5000", c.BaseURL)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("Idempotency-Key", fmt.Sprintf("verify-%s-%d", idempotencyKey, time.Now().Unix()))

	if c.APIKey == "" {
		return nil, fmt.Errorf("verify.et API key is not configured")
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var verifyResp VerifyResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &verifyResp, nil
}