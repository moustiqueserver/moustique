package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LightningRuntime holds Lightning Network runtime configuration
type LightningRuntime struct {
	Enabled         bool
	Provider        string // "opennode" or "lnbits"
	APIKey          string
	WebhookSecret   string
	OpenNodeClient  *OpenNodeClient
	LNBitsClient    *LNBitsClient
	LNBitsURL       string // LNBits server URL
}

// OpenNodeClient handles OpenNode API interactions
type OpenNodeClient struct {
	apiKey    string
	baseURL   string
	client    *http.Client
}

// NewOpenNodeClient creates a new OpenNode API client
func NewOpenNodeClient(apiKey string) *OpenNodeClient {
	return &OpenNodeClient{
		apiKey:  apiKey,
		baseURL: "https://api.opennode.com/v1",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// CreateChargeRequest represents the request to create a charge
type CreateChargeRequest struct {
	Amount      int64  `json:"amount"`
	Currency    string `json:"currency"`
	Description string `json:"description"`
	OrderID     string `json:"order_id,omitempty"`
	CallbackURL string `json:"callback_url,omitempty"`
}

// CreateChargeResponse represents OpenNode's charge creation response
type CreateChargeResponse struct {
	Data struct {
		ID              string `json:"id"`
		Description     string `json:"description"`
		Amount          int64  `json:"amount"`
		Status          string `json:"status"`
		FiatValue       string `json:"fiat_value"`
		Currency        string `json:"currency"`
		LightningInvoice struct {
			ExpiresAt int64  `json:"expires_at"`
			Payreq    string `json:"payreq"`
		} `json:"lightning_invoice"`
		OrderID   string `json:"order_id"`
		URI       string `json:"uri"`
		HostedURL string `json:"hosted_checkout_url"`
	} `json:"data"`
}

// ChargeInfo represents charge status info
type ChargeInfo struct {
	Data struct {
		ID     string `json:"id"`
		Status string `json:"status"` // unpaid, processing, paid, expired
		Amount int64  `json:"amount"`
	} `json:"data"`
}

// WebhookPayload represents the OpenNode webhook payload
type WebhookPayload struct {
	ID          string `json:"id"`
	CallbackURL string `json:"callback_url"`
	SuccessURL  string `json:"success_url"`
	Status      string `json:"status"`
	OrderID     string `json:"order_id"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	FiatValue   string `json:"fiat_value"`
	Currency    string `json:"currency"`
	Source      string `json:"source"`
	PayReq      string `json:"payreq"`
	HodlInvoice bool   `json:"hodl_invoice"`
	CreatedAt   int64  `json:"created_at"`
	ExpiresAt   int64  `json:"expires_at"`
	MissingAmt  int64  `json:"missing_amt"`
	ChainInvoice struct {
		Address string `json:"address"`
	} `json:"chain_invoice"`
}

// CreateCharge creates a new Lightning charge/invoice
func (c *OpenNodeClient) CreateCharge(amountSats int64, description string, orderID string, callbackURL string) (*CreateChargeResponse, error) {
	reqBody := CreateChargeRequest{
		Amount:      amountSats,
		Currency:    "btc",
		Description: description,
		OrderID:     orderID,
		CallbackURL: callbackURL,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/charges", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("opennode API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chargeResp CreateChargeResponse
	if err := json.Unmarshal(body, &chargeResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chargeResp, nil
}

// GetCharge retrieves charge status
func (c *OpenNodeClient) GetCharge(chargeID string) (*ChargeInfo, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/charge/"+chargeID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opennode API error (status %d): %s", resp.StatusCode, string(body))
	}

	var chargeInfo ChargeInfo
	if err := json.Unmarshal(body, &chargeInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &chargeInfo, nil
}

// VerifyWebhookSignature verifies the HMAC signature from OpenNode webhook
func VerifyWebhookSignature(body []byte, signature string, secret string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

// ========== LNBits Client ==========

// LNBitsClient handles LNBits API interactions
type LNBitsClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewLNBitsClient creates a new LNBits API client
func NewLNBitsClient(apiKey string, baseURL string) *LNBitsClient {
	return &LNBitsClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// LNBitsInvoiceRequest represents the request to create an invoice
type LNBitsInvoiceRequest struct {
	Out    bool   `json:"out"`
	Amount int64  `json:"amount"` // amount in msats
	Memo   string `json:"memo"`
	Expiry int    `json:"expiry,omitempty"` // seconds
}

// LNBitsInvoiceResponse represents LNBits invoice creation response
type LNBitsInvoiceResponse struct {
	CheckingID     string `json:"checking_id"`
	PaymentHash    string `json:"payment_hash"`
	Bolt11         string `json:"bolt11"`
	PaymentRequest string `json:"payment_request"`
	Amount         int64  `json:"amount"` // in msats
	Memo           string `json:"memo"`
	Expiry         string `json:"expiry"` // ISO 8601 timestamp
	Status         string `json:"status"` // pending, paid, expired
}

// LNBitsPaymentStatus represents payment status check response
type LNBitsPaymentStatus struct {
	Paid bool `json:"paid"`
}

// CreateInvoice creates a new Lightning invoice via LNBits
func (c *LNBitsClient) CreateInvoice(amountSats int64, description string, expiry int) (*LNBitsInvoiceResponse, error) {
	reqBody := LNBitsInvoiceRequest{
		Out:    false, // false = incoming payment (invoice)
		Amount: amountSats * 1000, // convert sats to msats
		Memo:   description,
		Expiry: expiry,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/payments", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("lnbits API error (status %d): %s", resp.StatusCode, string(body))
	}

	var invoiceResp LNBitsInvoiceResponse
	if err := json.Unmarshal(body, &invoiceResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &invoiceResp, nil
}

// CheckPayment checks if an invoice has been paid
func (c *LNBitsClient) CheckPayment(paymentHash string) (bool, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/api/v1/payments/"+paymentHash, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Api-Key", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("lnbits API error (status %d): %s", resp.StatusCode, string(body))
	}

	var status LNBitsPaymentStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return status.Paid, nil
}
