package utils

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PayPalClient provides methods to interact with PayPal REST API V2
type PayPalClient struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	WebhookID    string
	ExchangeRate float64
	HTTPClient   *http.Client

	mu          sync.RWMutex
	accessToken string
	tokenExpiry time.Time
}

// NewPayPalClient initializes a PayPalClient instance from environment variables
func NewPayPalClient() *PayPalClient {
	baseURL := os.Getenv("PAYPAL_BASE_URL")
	if baseURL == "" {
		if os.Getenv("PAYPAL_ENVIRONMENT") == "live" {
			baseURL = "https://api-m.paypal.com"
		} else {
			baseURL = "https://api-m.sandbox.paypal.com"
		}
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	clientID := os.Getenv("PAYPAL_CLIENT_ID")
	clientSecret := os.Getenv("PAYPAL_CLIENT_SECRET")
	webhookID := os.Getenv("PAYPAL_WEBHOOK_ID")

	exchangeRate := 16000.0
	if rateStr := os.Getenv("USD_IDR_EXCHANGE_RATE"); rateStr != "" {
		if rate, err := strconv.ParseFloat(rateStr, 64); err == nil && rate > 0 {
			exchangeRate = rate
		}
	}

	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	return &PayPalClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		BaseURL:      baseURL,
		WebhookID:    webhookID,
		ExchangeRate: exchangeRate,
		HTTPClient: &http.Client{
			Transport: transport,
			Timeout:   20 * time.Second,
		},
	}
}

// ConvertIDRToUSD converts IDR amount to USD string formatted with 2 decimal places (minimum $1.00)
func (c *PayPalClient) ConvertIDRToUSD(idrAmount float64) string {
	if idrAmount <= 0 {
		return "1.00"
	}
	usd := idrAmount / c.ExchangeRate
	// Round up to 2 decimal places
	rounded := math.Ceil(usd*100) / 100
	if rounded < 1.00 {
		rounded = 1.00
	}
	return fmt.Sprintf("%.2f", rounded)
}

// GetAccessToken retrieves or cached OAuth 2.0 bearer token from PayPal
func (c *PayPalClient) GetAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		token := c.accessToken
		c.mu.RUnlock()
		return token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double check after lock
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	if c.ClientID == "" || c.ClientSecret == "" {
		return "", errors.New("PayPal Client ID or Secret is not configured")
	}

	endpoint := fmt.Sprintf("%s/v1/oauth2/token", c.BaseURL)
	reqBody := strings.NewReader("grant_type=client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, reqBody)
	if err != nil {
		return "", err
	}

	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.ClientID, c.ClientSecret)))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request PayPal token: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("PayPal auth failed (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to decode PayPal token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	// Subtract 60 seconds for safety margin
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn-60) * time.Second)

	return c.accessToken, nil
}

// PayPalAmount represents money value and currency in PayPal V2
type PayPalAmount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

// PayPalPurchaseUnit represents an order line item in PayPal V2
type PayPalPurchaseUnit struct {
	ReferenceID string       `json:"reference_id,omitempty"`
	Description string       `json:"description,omitempty"`
	CustomID    string       `json:"custom_id,omitempty"`
	Amount      PayPalAmount `json:"amount"`
}

// PayPalAppContext represents experience context in PayPal V2
type PayPalAppContext struct {
	BrandName          string `json:"brand_name,omitempty"`
	LandingPage        string `json:"landing_page,omitempty"`
	UserAction         string `json:"user_action,omitempty"`
	ReturnURL          string `json:"return_url,omitempty"`
	CancelURL          string `json:"cancel_url,omitempty"`
	ShippingPreference string `json:"shipping_preference,omitempty"`
}

// PayPalCreateOrderReq payload for /v2/checkout/orders
type PayPalCreateOrderReq struct {
	Intent             string               `json:"intent"`
	PurchaseUnits      []PayPalPurchaseUnit `json:"purchase_units"`
	ApplicationContext PayPalAppContext     `json:"application_context"`
}

// PayPalLink represents HATEOAS link returned by PayPal
type PayPalLink struct {
	Href   string `json:"href"`
	Rel    string `json:"rel"`
	Method string `json:"method"`
}

// PayPalOrderResp response from /v2/checkout/orders
type PayPalOrderResp struct {
	ID     string       `json:"id"`
	Status string       `json:"status"`
	Links  []PayPalLink `json:"links"`
}

// CreateOrder creates a PayPal V2 checkout order and returns order ID & approve checkout URL
func (c *PayPalClient) CreateOrder(ctx context.Context, referenceID, description, usdAmount, returnURL, cancelURL string) (*PayPalOrderResp, string, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, "", err
	}

	payload := PayPalCreateOrderReq{
		Intent: "CAPTURE",
		PurchaseUnits: []PayPalPurchaseUnit{
			{
				ReferenceID: referenceID,
				Description: description,
				CustomID:    referenceID,
				Amount: PayPalAmount{
					CurrencyCode: "USD",
					Value:        usdAmount,
				},
			},
		},
		ApplicationContext: PayPalAppContext{
			BrandName:          "ArcheryHub",
			LandingPage:        "LOGIN",
			UserAction:         "PAY_NOW",
			ReturnURL:          returnURL,
			CancelURL:          cancelURL,
			ShippingPreference: "NO_SHIPPING",
		},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}

	endpoint := fmt.Sprintf("%s/v2/checkout/orders", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, "", err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("PayPal create order request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("PayPal create order returned error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var orderResp PayPalOrderResp
	if err := json.Unmarshal(bodyBytes, &orderResp); err != nil {
		return nil, "", fmt.Errorf("failed to parse PayPal create order response: %w", err)
	}

	var approveURL string
	for _, link := range orderResp.Links {
		if link.Rel == "approve" || link.Rel == "payer-action" {
			approveURL = link.Href
			break
		}
	}

	return &orderResp, approveURL, nil
}

// PayPalCaptureResp represents capture response from PayPal
type PayPalCaptureResp struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Payer  struct {
		EmailAddress string `json:"email_address"`
		PayerID      string `json:"payer_id"`
		Name         struct {
			GivenName string `json:"given_name"`
			Surname   string `json:"surname"`
		} `json:"name"`
	} `json:"payer"`
	PurchaseUnits []struct {
		ReferenceID string `json:"reference_id"`
		Payments    struct {
			Captures []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
				Amount struct {
					CurrencyCode string `json:"currency_code"`
					Value        string `json:"value"`
				} `json:"amount"`
			} `json:"captures"`
		} `json:"payments"`
	} `json:"purchase_units"`
}

// CaptureOrder captures payment for an approved PayPal order
func (c *PayPalClient) CaptureOrder(ctx context.Context, orderID string) (*PayPalCaptureResp, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/v2/checkout/orders/%s/capture", c.BaseURL, orderID)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("PayPal capture order request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("PayPal capture order returned error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var captureResp PayPalCaptureResp
	if err := json.Unmarshal(bodyBytes, &captureResp); err != nil {
		return nil, fmt.Errorf("failed to parse PayPal capture response: %w", err)
	}

	return &captureResp, nil
}

// VerifyWebhookSignature verifies incoming PayPal webhook signature
func (c *PayPalClient) VerifyWebhookSignature(ctx context.Context, authAlgo, certURL, transmissionID, transmissionSig, transmissionTime, webhookID string, rawBody []byte) (bool, error) {
	token, err := c.GetAccessToken(ctx)
	if err != nil {
		return false, err
	}

	targetWebhookID := webhookID
	if targetWebhookID == "" {
		targetWebhookID = c.WebhookID
	}

	var webhookEvent map[string]interface{}
	if err := json.Unmarshal(rawBody, &webhookEvent); err != nil {
		return false, fmt.Errorf("invalid json in webhook body: %w", err)
	}

	verifyPayload := map[string]interface{}{
		"auth_algo":         authAlgo,
		"cert_url":          certURL,
		"transmission_id":   transmissionID,
		"transmission_sig":  transmissionSig,
		"transmission_time": transmissionTime,
		"webhook_id":        targetWebhookID,
		"webhook_event":     webhookEvent,
	}

	jsonBytes, err := json.Marshal(verifyPayload)
	if err != nil {
		return false, err
	}

	endpoint := fmt.Sprintf("%s/v1/notifications/verify-webhook-signature", c.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("PayPal verify webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("PayPal verify webhook returned error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	var verifyResp struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := json.Unmarshal(bodyBytes, &verifyResp); err != nil {
		return false, fmt.Errorf("failed to parse verification response: %w", err)
	}

	return verifyResp.VerificationStatus == "SUCCESS", nil
}
