package handler

import (
	"Archeris-api/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

// DevPayPalCheckToken tests OAuth 2.0 connection to PayPal Sandbox
func DevPayPalCheckToken(c *gin.Context) {
	client := utils.NewPayPalClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := client.GetAccessToken(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":     false,
			"error":       err.Error(),
			"environment": os.Getenv("PAYPAL_ENVIRONMENT"),
			"base_url":    client.BaseURL,
			"client_id":   maskString(client.ClientID),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Successfully authenticated with PayPal Sandbox OAuth 2.0",
		"environment":   os.Getenv("PAYPAL_ENVIRONMENT"),
		"base_url":      client.BaseURL,
		"client_id":     maskString(client.ClientID),
		"token_preview": maskString(token),
		"exchange_rate": client.ExchangeRate,
		"tested_at":     time.Now().Format(time.RFC3339),
	})
}

// DevPayPalCreateOrderRequest payload for creating test order
type DevPayPalCreateOrderRequest struct {
	Amount      string `json:"amount"`       // e.g. "15.00"
	Currency    string `json:"currency"`     // e.g. "USD"
	Description string `json:"description"`  // e.g. "Test Event Entry Fee"
	ReferenceID string `json:"reference_id"` // e.g. "TEST-PAY-001"
	ReturnURL   string `json:"return_url"`   // custom return URL
	CancelURL   string `json:"cancel_url"`   // custom cancel URL
}

// DevPayPalCreateOrder creates an order in PayPal Sandbox
func DevPayPalCreateOrder(c *gin.Context) {
	var req DevPayPalCreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.Amount == "" {
		req.Amount = "10.00"
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}
	if req.Description == "" {
		req.Description = "Archeris SaaS Sandbox Test Payment"
	}
	if req.ReferenceID == "" {
		req.ReferenceID = fmt.Sprintf("DEV-PP-%d", time.Now().Unix())
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:3003"
	}

	if req.ReturnURL == "" {
		req.ReturnURL = fmt.Sprintf("%s/dev/paypal?status=approved&ref=%s", appURL, req.ReferenceID)
	}
	if req.CancelURL == "" {
		req.CancelURL = fmt.Sprintf("%s/dev/paypal?status=cancelled&ref=%s", appURL, req.ReferenceID)
	}

	client := utils.NewPayPalClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	orderResp, approveURL, err := client.CreateOrder(ctx, req.ReferenceID, req.Description, req.Amount, req.ReturnURL, req.CancelURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to create PayPal order: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"order_id":     orderResp.ID,
		"status":       orderResp.Status,
		"approve_url":  approveURL,
		"reference_id": req.ReferenceID,
		"amount":       req.Amount,
		"currency":     req.Currency,
		"links":        orderResp.Links,
		"raw_response": orderResp,
	})
}

// DevPayPalCaptureOrderRequest payload for capturing test order
type DevPayPalCaptureOrderRequest struct {
	OrderID string `json:"order_id"`
}

// DevPayPalCaptureOrder captures payment for an approved PayPal Sandbox order
func DevPayPalCaptureOrder(c *gin.Context) {
	var req DevPayPalCaptureOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	if req.OrderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
		return
	}

	client := utils.NewPayPalClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	captureResp, err := client.CaptureOrder(ctx, req.OrderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to capture PayPal order: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"order_id":     captureResp.ID,
		"status":       captureResp.Status,
		"payer":        captureResp.Payer,
		"capture_data": captureResp.PurchaseUnits,
		"raw_response": captureResp,
	})
}

// DevPayPalGetOrder gets order details directly from PayPal API
func DevPayPalGetOrder(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id parameter is required"})
		return
	}

	client := utils.NewPayPalClient()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := client.GetAccessToken(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	endpoint := fmt.Sprintf("%s/v2/checkout/orders/%s", client.BaseURL, orderID)
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &result)

	c.JSON(resp.StatusCode, gin.H{
		"status_code": resp.StatusCode,
		"data":        result,
	})
}

// Helper to mask sensitive keys for display
func maskString(s string) string {
	if len(s) <= 8 {
		return "******"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
