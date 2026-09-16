package handler

import (
	"Archeris-api/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// PayPalWebhookCallback handles incoming server-to-server webhook notifications from PayPal
func PayPalWebhookCallback(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}

		// Log raw webhook
		_ = os.MkdirAll("logs", 0755)
		f, _ := os.OpenFile("logs/paypal-callback.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if f != nil {
			defer f.Close()
			timestamp := time.Now().Format("2006-01-02 15:04:05")
			_, _ = fmt.Fprintf(f, "\n[%s] === PAYPAL WEBHOOK RECEIVED ===\n%s\n", timestamp, string(bodyBytes))
		}

		paypalClient := utils.NewPayPalClient()
		webhookID := os.Getenv("PAYPAL_WEBHOOK_ID")

		// Verify signature if Webhook ID is configured
		if webhookID != "" {
			authAlgo := c.GetHeader("PAYPAL-AUTH-ALGO")
			certURL := c.GetHeader("PAYPAL-CERT-URL")
			transmissionID := c.GetHeader("PAYPAL-TRANSMISSION-ID")
			transmissionSig := c.GetHeader("PAYPAL-TRANSMISSION-SIG")
			transmissionTime := c.GetHeader("PAYPAL-TRANSMISSION-TIME")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			isValid, err := paypalClient.VerifyWebhookSignature(ctx, authAlgo, certURL, transmissionID, transmissionSig, transmissionTime, webhookID, bodyBytes)
			if err != nil || !isValid {
				fmt.Printf("[PayPal Webhook] Signature verification failed or error: %v\n", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid PayPal webhook signature"})
				return
			}
		}

		var event struct {
			ID           string                 `json:"id"`
			EventType    string                 `json:"event_type"`
			Summary      string                 `json:"summary"`
			CreateTime   string                 `json:"create_time"`
			Resource     map[string]interface{} `json:"resource"`
			ResourceType string                 `json:"resource_type"`
		}

		if err := json.Unmarshal(bodyBytes, &event); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON format"})
			return
		}

		fmt.Printf("[PayPal Webhook] Processing event: %s (ID: %s)\n", event.EventType, event.ID)

		switch event.EventType {
		case "PAYMENT.CAPTURE.COMPLETED":
			handlePayPalCaptureCompleted(db, event.Resource, bodyBytes)
		case "CHECKOUT.ORDER.APPROVED":
			handlePayPalOrderApproved(db, event.Resource)
		default:
			fmt.Printf("[PayPal Webhook] Unhandled event type: %s\n", event.EventType)
		}

		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	}
}

// handlePayPalCaptureCompleted marks transaction as paid in database
func handlePayPalCaptureCompleted(db *sqlx.DB, resource map[string]interface{}, rawBytes []byte) {
	customID, _ := resource["custom_id"].(string)
	captureID, _ := resource["id"].(string)

	if customID == "" {
		if supp, ok := resource["supplementary_data"].(map[string]interface{}); ok {
			if rID, ok := supp["related_ids"].(map[string]interface{}); ok {
				customID, _ = rID["order_id"].(string)
			}
		}
	}

	fmt.Printf("[PayPal Webhook] Capture completed for custom_id: %s, capture_id: %s\n", customID, captureID)
	if customID == "" {
		return
	}

	tx, err := db.Beginx()
	if err != nil {
		fmt.Printf("Failed to begin db tx: %v\n", err)
		return
	}
	defer tx.Rollback()

	// 1. Quota Purchase
	if strings.HasPrefix(customID, "QUOTA-") {
		var q struct {
			UUID          string `db:"uuid"`
			OrganizerID   string `db:"organizer_id"`
			QuotaType     string `db:"quota_type"`
			Quantity      int    `db:"quantity"`
			PaymentStatus string `db:"payment_status"`
		}
		err := tx.Get(&q, "SELECT uuid, organizer_id, quota_type, quantity, payment_status FROM quota_purchases WHERE payment_reference = ? OR gateway_reference = ?", customID, customID)
		if err == nil && q.PaymentStatus != "paid" {
			_, _ = tx.Exec("UPDATE quota_purchases SET payment_status = 'paid', payment_method = 'paypal', callback_data = ? WHERE uuid = ?", string(rawBytes), q.UUID)
			if q.QuotaType == "elite" {
				_, _ = tx.Exec("UPDATE organizers SET quota_elite = quota_elite + ? WHERE uuid = ?", q.Quantity, q.OrganizerID)
			} else {
				_, _ = tx.Exec("UPDATE organizers SET quota_standard = quota_standard + ? WHERE uuid = ?", q.Quantity, q.OrganizerID)
			}
			_ = tx.Commit()
			fmt.Printf("[PayPal Webhook] Quota purchase %s successfully fulfilled!\n", customID)
			return
		}
	}

	// 2. Regular Payment Transaction (Event Registration / Subscription)
	var payment struct {
		UUID           string  `db:"uuid"`
		EventID        *string `db:"event_id"`
		RegistrationID *string `db:"registration_id"`
		Status         string  `db:"status"`
	}
	err = tx.Get(&payment, "SELECT uuid, event_id, registration_id, status FROM payment_transactions WHERE reference = ? OR gateway_reference = ?", customID, customID)
	if err == nil {
		if payment.Status != "paid" {
			_, _ = tx.Exec("UPDATE payment_transactions SET status = 'paid', payment_method = 'paypal', paid_at = NOW(), callback_data = ? WHERE uuid = ?", string(rawBytes), payment.UUID)
			if payment.RegistrationID != nil {
				_, _ = tx.Exec("UPDATE event_participants SET payment_status = 'paid' WHERE uuid = ?", *payment.RegistrationID)
			}
			_ = tx.Commit()
			fmt.Printf("[PayPal Webhook] Payment transaction %s marked as paid!\n", customID)
		}
		return
	}

	_ = tx.Commit()
}

// handlePayPalOrderApproved triggers auto-capture as safety net if client missed return
func handlePayPalOrderApproved(db *sqlx.DB, resource map[string]interface{}) {
	orderID, _ := resource["id"].(string)
	if orderID == "" {
		return
	}

	paypalClient := utils.NewPayPalClient()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	captureResp, err := paypalClient.CaptureOrder(ctx, orderID)
	if err != nil {
		fmt.Printf("[PayPal Webhook Auto-Capture] Error or already captured for order %s: %v\n", orderID, err)
		return
	}

	fmt.Printf("[PayPal Webhook Auto-Capture] Successfully auto-captured order %s (Status: %s)\n", orderID, captureResp.Status)
}

// CapturePayPalPayment POST /payment/paypal/capture
func CapturePayPalPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			OrderID   string `json:"order_id" binding:"required"`
			Reference string `json:"reference"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "order_id is required"})
			return
		}

		paypalClient := utils.NewPayPalClient()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		captureResp, err := paypalClient.CaptureOrder(ctx, req.OrderID)
		if err != nil {
			fmt.Printf("Capture error (might already be captured): %v\n", err)
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}
		defer tx.Rollback()

		// 1. Quota purchase check
		if strings.HasPrefix(req.Reference, "QUOTA-") {
			var q struct {
				UUID          string `db:"uuid"`
				OrganizerID   string `db:"organizer_id"`
				QuotaType     string `db:"quota_type"`
				Quantity      int    `db:"quantity"`
				PaymentStatus string `db:"payment_status"`
			}
			err := tx.Get(&q, "SELECT uuid, organizer_id, quota_type, quantity, payment_status FROM quota_purchases WHERE payment_reference = ? OR gateway_reference = ?", req.Reference, req.OrderID)
			if err == nil {
				if q.PaymentStatus != "paid" {
					_, _ = tx.Exec("UPDATE quota_purchases SET payment_status = 'paid', payment_method = 'paypal' WHERE uuid = ?", q.UUID)
					if q.QuotaType == "elite" {
						_, _ = tx.Exec("UPDATE organizers SET quota_elite = quota_elite + ? WHERE uuid = ?", q.Quantity, q.OrganizerID)
					} else {
						_, _ = tx.Exec("UPDATE organizers SET quota_standard = quota_standard + ? WHERE uuid = ?", q.Quantity, q.OrganizerID)
					}
				}
				_ = tx.Commit()
				c.JSON(http.StatusOK, gin.H{
					"success": true,
					"status":  "paid",
					"type":    "quota",
					"message": "Pembayaran kuota via PayPal berhasil",
				})
				return
			}
		}

		// 2. Regular payment check
		var payment struct {
			UUID           string  `db:"uuid"`
			RegistrationID *string `db:"registration_id"`
			Status         string  `db:"status"`
		}
		err = tx.Get(&payment, "SELECT uuid, registration_id, status FROM payment_transactions WHERE reference = ? OR gateway_reference = ?", req.Reference, req.OrderID)
		if err == nil {
			if payment.Status != "paid" {
				_, _ = tx.Exec("UPDATE payment_transactions SET status = 'paid', payment_method = 'paypal', paid_at = NOW() WHERE uuid = ?", payment.UUID)
				if payment.RegistrationID != nil {
					_, _ = tx.Exec("UPDATE event_participants SET payment_status = 'paid' WHERE uuid = ?", *payment.RegistrationID)
				}
			}
			_ = tx.Commit()
			c.JSON(http.StatusOK, gin.H{
				"success":      true,
				"status":       "paid",
				"type":         "registration",
				"capture_data": captureResp,
				"message":      "Pembayaran registrasi event via PayPal berhasil",
			})
			return
		}

		_ = tx.Commit()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"status":  "paid",
			"message": "Capture PayPal diproses",
		})
	}
}
