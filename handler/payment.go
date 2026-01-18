package handler

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RegisterTournament handles tournament registration
func RegisterTournament(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req models.RegisterTournamentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if tournament exists and get entry fee
		var tournament struct {
			ID       string  `db:"id"`
			EntryFee float64 `db:"entry_fee"` // Assuming there's a default entry fee
		}
		err := db.Get(&tournament, "SELECT id FROM tournaments WHERE id = ?", tournamentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tournament not found"})
			return
		}

		// Fixed entry fee for now or get from tournament events
		entryFee := 350000.0 // Default
		adminFee := 5000.0
		totalFee := entryFee + adminFee

		registrationID := uuid.New().String()
		regNumber := fmt.Sprintf("REG-%d-%s", time.Now().Unix(), registrationID[:8])

		registration := models.TournamentRegistration{
			ID:                 registrationID,
			TournamentID:       tournamentID,
			UserID:             userID.(string),
			AthleteName:        req.AthleteName,
			AthleteEmail:       req.AthleteEmail,
			AthletePhone:       req.AthletePhone,
			ClubName:           req.ClubName,
			Division:           req.Division,
			Category:           req.Category,
			BowType:            req.BowType,
			EntryFee:           entryFee,
			AdminFee:           adminFee,
			TotalFee:           totalFee,
			PaymentStatus:      "unpaid",
			RegistrationNumber: &regNumber,
			Status:             "pending",
		}

		query := `
			INSERT INTO tournament_registrations (
				id, tournament_id, user_id, athlete_name, athlete_email, athlete_phone, 
				club_name, division, category, bow_type, entry_fee, admin_fee, 
				total_fee, payment_status, registration_number, status
			) VALUES (
				:id, :tournament_id, :user_id, :athlete_name, :athlete_email, :athlete_phone, 
				:club_name, :division, :category, :bow_type, :entry_fee, :admin_fee, 
				:total_fee, :payment_status, :registration_number, :status
			)
		`
		_, err = db.NamedExec(query, registration)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, registration)
	}
}

// CreatePayment handles creating a Tripay transaction
func CreatePayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req models.CreatePaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get registration details
		var reg models.TournamentRegistration
		err := db.Get(&reg, "SELECT * FROM tournament_registrations WHERE id = ? AND user_id = ?", req.RegistrationID, userID.(string))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registration not found"})
			return
		}

		if reg.PaymentStatus == "paid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Already paid"})
			return
		}

		tripay := utils.NewTripayClient()
		merchantRef := fmt.Sprintf("PAY-%s", uuid.New().String()[:12])
		amount := int(reg.TotalFee)

		signature := tripay.GenerateSignature(merchantRef, amount)

		payload := gin.H{
			"method":         req.Method,
			"merchant_ref":   merchantRef,
			"amount":         amount,
			"customer_name":  reg.AthleteName,
			"customer_email": utils.StringValue(reg.AthleteEmail, "user@archeryhub.id"),
			"customer_phone": utils.StringValue(reg.AthletePhone, ""),
			"order_items": []gin.H{
				{
					"name":     fmt.Sprintf("Tournament Entry Fee - %s", reg.Division),
					"price":    int(reg.EntryFee),
					"quantity": 1,
				},
				{
					"name":     "Platform Admin Fee",
					"price":    int(reg.AdminFee),
					"quantity": 1,
				},
			},
			"signature": signature,
		}

		tripayResult, err := tripay.CreateTransaction(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Tripay transaction: " + err.Error()})
			return
		}

		// Save transaction to database
		transactionID := uuid.New().String()
		tripayRef := tripayResult["reference"].(string)
		expiredAt := time.Now().Add(24 * time.Hour) // Default 24h
		if exp, ok := tripayResult["expiry_date"].(float64); ok {
			expiredAt = time.Unix(int64(exp), 0)
		}

		transaction := models.PaymentTransaction{
			ID:              transactionID,
			Reference:       merchantRef,
			TripayReference: &tripayRef,
			UserID:          userID.(string),
			TournamentID:    &reg.TournamentID,
			RegistrationID:  &reg.ID,
			Amount:          reg.EntryFee,
			FeeAmount:       reg.AdminFee,
			TotalAmount:     reg.TotalFee,
			PaymentMethod:   utils.StringPtr(req.Method),
			VANumber:        utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			QRURL:           utils.InterfaceToStringPtr(tripayResult["qr_url"]),
			CheckoutURL:     utils.InterfaceToStringPtr(tripayResult["checkout_url"]),
			PayCode:         utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			Status:          "pending",
			ExpiredAt:       expiredAt,
		}

		query := `
			INSERT INTO payment_transactions (
				id, reference, tripay_reference, user_id, tournament_id, registration_id,
				amount, fee_amount, total_amount, payment_method, va_number, qr_url,
				checkout_url, pay_code, status, expired_at
			) VALUES (
				:id, :reference, :tripay_reference, :user_id, :tournament_id, :registration_id,
				:amount, :fee_amount, :total_amount, :payment_method, :va_number, :qr_url,
				:checkout_url, :pay_code, :status, :expired_at
			)
		`
		_, err = db.NamedExec(query, transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction: " + err.Error()})
			return
		}

		// Update registration with payment ID and status
		_, err = db.Exec("UPDATE tournament_registrations SET payment_id = ?, payment_status = ? WHERE id = ?", transactionID, "pending", reg.ID)
		if err != nil {
			// Log error but don't fail because transaction is created
			fmt.Printf("Error updating registration: %v\n", err)
		}

		c.JSON(http.StatusOK, transaction)
	}
}

// PaymentCallback handles Tripay webhook notifications
func PaymentCallback(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tripay := utils.NewTripayClient()
		
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read body"})
			return
		}

		signature := c.GetHeader("X-Callback-Signature")
		if !tripay.VerifyCallbackSignature(body, signature) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid signature"})
			return
		}

		var payload struct {
			Reference      string `json:"reference"`
			MerchantRef    string `json:"merchant_ref"`
			Status         string `json:"status"`
			IsClosedPayment int    `json:"is_closed_payment"`
		}

		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payload"})
			return
		}

		// Update transaction status
		var transactionID string
		var registrationID string
		err = db.QueryRow("SELECT id, registration_id FROM payment_transactions WHERE reference = ?", payload.MerchantRef).Scan(&transactionID, &registrationID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}

		status := "pending"
		regPaymentStatus := "pending"
		if payload.Status == "PAID" {
			status = "paid"
			regPaymentStatus = "paid"
		} else if payload.Status == "EXPIRED" {
			status = "expired"
			regPaymentStatus = "unpaid"
		} else if payload.Status == "FAILED" {
			status = "failed"
			regPaymentStatus = "unpaid"
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}

		_, err = tx.Exec("UPDATE payment_transactions SET status = ?, callback_data = ?, paid_at = ? WHERE id = ?", status, body, time.Now(), transactionID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
			return
		}

		_, err = tx.Exec("UPDATE tournament_registrations SET payment_status = ? WHERE id = ?", regPaymentStatus, registrationID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update registration"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// GetPaymentStatus returns the status of a payment transaction
func GetPaymentStatus(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reference := c.Param("reference")
		
		var transaction models.PaymentTransaction
		err := db.Get(&transaction, "SELECT * FROM payment_transactions WHERE reference = ?", reference)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}

		c.JSON(http.StatusOK, transaction)
	}
}

// GetPaymentChannels returns available Tripay payment channels
func GetPaymentChannels(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tripay := utils.NewTripayClient()
		channels, err := tripay.GetPaymentChannels()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get channels: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, channels)
	}
}
