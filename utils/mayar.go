package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

// MayarClient provides methods to interact with Mayar Headless API V2
type MayarClient struct {
	APIKey        string
	BaseURL       string
	WebhookSecret string
	HTTPClient    *http.Client
}

// NewMayarClient initializes a MayarClient instance from environment variables
func NewMayarClient() *MayarClient {
	baseURL := os.Getenv("MAYAR_API_URL")
	if baseURL == "" {
		baseURL = "https://api.mayar.id/hl/v2"
	}
	baseURL = strings.TrimSuffix(baseURL, "/")
	if strings.HasSuffix(baseURL, "/hl/v1") {
		baseURL = strings.TrimSuffix(baseURL, "/hl/v1") + "/hl/v2"
	}

	apiKey := os.Getenv("MAYAR_API_KEY")
	webhookSecret := os.Getenv("MAYAR_WEBHOOK_SECRET")

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &MayarClient{
		APIKey:        apiKey,
		BaseURL:       baseURL,
		WebhookSecret: webhookSecret,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}
}

// MayarPaymentReq payload for single payment request (/payments/create)
type MayarPaymentReq struct {
	Name        string `json:"name"`
	Amount      int    `json:"amount"`
	Email       string `json:"email,omitempty"`
	Mobile      string `json:"mobile,omitempty"`
	Description string `json:"description,omitempty"`
	ExpiredAt   string `json:"expiredAt,omitempty"`
	RedirectURL string `json:"redirectUrl,omitempty"`
}

// MayarInvoiceItem represents an item line in invoice
type MayarInvoiceItem struct {
	Quantity    int    `json:"quantity"`
	Rate        int    `json:"rate"`
	Description string `json:"description"`
}

// MayarInvoiceReq payload for invoice (/invoices/create)
type MayarInvoiceReq struct {
	Name          string                 `json:"name"`
	Email         string                 `json:"email"`
	Mobile        string                 `json:"mobile"`
	Description   string                 `json:"description,omitempty"`
	ExpiredAt     string                 `json:"expiredAt,omitempty"`
	Items         []MayarInvoiceItem     `json:"items"`
	ExtraData     map[string]interface{} `json:"extraData,omitempty"`
	PaymentMethod string                 `json:"paymentMethod,omitempty"`
	RedirectURL   string                 `json:"redirectUrl,omitempty"`
}

// MayarCustomer sub-struct
type MayarCustomer struct {
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Email  string `json:"email,omitempty"`
	Mobile string `json:"mobile,omitempty"`
}

// MayarPaymentLink sub-struct
type MayarPaymentLink struct {
	ID        string      `json:"id,omitempty"`
	Link      string      `json:"link,omitempty"`
	Amount    int         `json:"amount,omitempty"`
	Status    string      `json:"status,omitempty"`
	ExpiredAt interface{} `json:"expiredAt,omitempty"`
	Type      string      `json:"type,omitempty"`
}

// MayarTransactionData represents the transaction/invoice payload returned by Mayar V2
type MayarTransactionData struct {
	ID            string            `json:"id"`
	TransactionID string            `json:"transactionId"`
	PaymentLinkId string            `json:"paymentLinkId"`
	Link          string            `json:"link"`
	Amount        int               `json:"amount"`
	Description   string            `json:"description"`
	RedirectURL   string            `json:"redirectUrl"`
	Status        string            `json:"status"`
	InvoiceCode   string            `json:"invoiceCode"`
	PaymentMethod string            `json:"paymentMethod"`
	Customer      *MayarCustomer    `json:"customer,omitempty"`
	PaymentLink   *MayarPaymentLink `json:"paymentLink,omitempty"`
	CreatedAt     interface{}       `json:"createdAt,omitempty"`
	ExpiredAt     interface{}       `json:"expiredAt,omitempty"`
	ExtraData     interface{}       `json:"extraData,omitempty"`
}

// MayarEnvelope wraps standard Mayar V2 response
type MayarEnvelope struct {
	StatusCode int                   `json:"statusCode"`
	Messages   string                `json:"messages"`
	Data       *MayarTransactionData `json:"data"`
}

// CreatePaymentRequest sends request to POST /hl/v2/payments/create
func (m *MayarClient) CreatePaymentRequest(req MayarPaymentReq) (*MayarTransactionData, error) {
	if m.APIKey == "" {
		return nil, errors.New("MAYAR_API_KEY is not configured")
	}

	url := fmt.Sprintf("%s/payments/create", m.BaseURL)
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payment request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+m.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "ArcheryHub/1.0")

	resp, err := m.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("network error calling Mayar: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Mayar response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Mayar API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var env MayarEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("failed to parse Mayar response: %w", err)
	}

	if env.Data == nil {
		return nil, fmt.Errorf("empty data in Mayar response: %s", string(respBody))
	}

	if env.Data.TransactionID == "" && env.Data.ID != "" {
		env.Data.TransactionID = env.Data.ID
	}

	return env.Data, nil
}

// CreateInvoice sends request to POST /hl/v2/invoices/create
func (m *MayarClient) CreateInvoice(req MayarInvoiceReq) (*MayarTransactionData, error) {
	if m.APIKey == "" {
		return nil, errors.New("MAYAR_API_KEY is not configured")
	}

	url := fmt.Sprintf("%s/invoices/create", m.BaseURL)
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invoice request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+m.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "ArcheryHub/1.0")

	resp, err := m.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("network error calling Mayar invoice API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Mayar response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Mayar API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var env MayarEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("failed to parse Mayar invoice response: %w", err)
	}

	if env.Data == nil {
		return nil, fmt.Errorf("empty data in Mayar response: %s", string(respBody))
	}

	if env.Data.TransactionID == "" && env.Data.ID != "" {
		env.Data.TransactionID = env.Data.ID
	}

	return env.Data, nil
}

// GetTransactionDetail fetches transaction detail from GET /hl/v2/transactions/{id}
func (m *MayarClient) GetTransactionDetail(transactionID string) (*MayarTransactionData, error) {
	if m.APIKey == "" {
		return nil, errors.New("MAYAR_API_KEY is not configured")
	}

	url := fmt.Sprintf("%s/transactions/%s", m.BaseURL, transactionID)
	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+m.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "ArcheryHub/1.0")

	resp, err := m.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("network error querying Mayar transaction: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Mayar transaction response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Mayar API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var env MayarEnvelope
	if err := json.Unmarshal(respBody, &env); err != nil {
		return nil, fmt.Errorf("failed to parse Mayar transaction response: %w", err)
	}

	return env.Data, nil
}

// VerifyWebhookSecret validates incoming webhook token against configured MAYAR_WEBHOOK_SECRET
func (m *MayarClient) VerifyWebhookSecret(headerToken string) bool {
	if m.WebhookSecret == "" {
		return true
	}
	return strings.TrimSpace(headerToken) == strings.TrimSpace(m.WebhookSecret)
}

// Helpers
func StringValue(s *string, defaultValue string) string {
	if s == nil {
		return defaultValue
	}
	return *s
}

func StringPtr(s string) *string {
	return &s
}

func InterfaceToStringPtr(i interface{}) *string {
	if i == nil {
		return nil
	}
	s := fmt.Sprintf("%v", i)
	return &s
}

