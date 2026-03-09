package handler

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RegisterEvent handles event registration
func RegisterEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req models.RegisterEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if event exists and get entry fee
		var event struct {
			UUID     string  `db:"uuid"`
			EntryFee float64 `db:"entry_fee"` // Assuming there's a default entry fee
		}
		err := db.Get(&event, "SELECT uuid FROM events WHERE uuid = ?", eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Fixed entry fee for now or get from event categories
		entryFee := 350000.0 // Default
		adminFee := 5000.0
		totalFee := entryFee + adminFee

		registrationID := uuid.New().String()
		regNumber := fmt.Sprintf("REG-%d-%s", time.Now().Unix(), registrationID[:8])

		registration := models.EventRegistration{
			UUID:               registrationID,
			EventID:            eventID,
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
			INSERT INTO event_registrations (
				id, event_id, user_id, athlete_name, athlete_email, athlete_phone, 
				club_name, division, category, bow_type, entry_fee, admin_fee, 
				total_fee, payment_status, registration_number, status
			) VALUES (
				:id, :event_id, :user_id, :athlete_name, :athlete_email, :athlete_phone, 
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

		var amount int
		var customerName, customerEmail, customerPhone string
		var registrationID *string
		var orderItems []gin.H

		if req.Type == "platform_fee" {
			// Get event details
			var event models.Event
			err := db.Get(&event, "SELECT * FROM events WHERE uuid = ? AND organizer_id = ?", req.EventID, userID.(string))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Event not found or unauthorized"})
				return
			}

			// Check if already has a pending platform fee for this event
			var existingPending int
			err = db.Get(&existingPending, "SELECT COUNT(*) FROM payment_transactions WHERE event_id = ? AND subscription_plan_id IS NULL AND registration_id IS NULL AND status = 'pending' AND expired_at > NOW()", req.EventID)
			if err == nil && existingPending > 0 {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Anda memiliki pembayaran biaya platform untuk event ini yang masih tertunda. Silakan selesaikan di riwayat transaksi.",
					"code":  "pending_platform_fee_exists",
				})
				return
			}

			amount = 50000 // For now hardcoded as per frontend

			// Get user details for customer info
			emailCtx, _ := c.Get("email")
			customerEmail = emailCtx.(string)
			customerName = "Organizer"
			customerPhone = "08123456789" // Fallback

			userType, _ := c.Get("user_type")
			if userType == "organization" {
				db.Get(&customerName, "SELECT name FROM organizations WHERE uuid = ?", userID.(string))
				db.Get(&customerPhone, "SELECT phone FROM organizations WHERE uuid = ?", userID.(string))
			} else if userType == "club" {
				db.Get(&customerName, "SELECT name FROM clubs WHERE uuid = ?", userID.(string))
				db.Get(&customerPhone, "SELECT phone FROM clubs WHERE uuid = ?", userID.(string))
			}

			if customerPhone == "" {
				customerPhone = "08123456789"
			}

			orderItems = []gin.H{
				{
					"sku":      "PLATFORM-FEE",
					"name":     fmt.Sprintf("Platform Fee - %s", event.Name),
					"price":    amount,
					"quantity": 1,
				},
			}
		} else if req.Type == "subscription" {
			if req.PlanID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "PlanID is required for subscription type"})
				return
			}

			// Check if user already has a pending subscription payment
			var existingPending int
			err := db.Get(&existingPending, "SELECT COUNT(*) FROM payment_transactions WHERE user_id = ? AND subscription_plan_id IS NOT NULL AND status = 'pending' AND expired_at > NOW()", userID.(string))
			if err == nil && existingPending > 0 {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Anda memiliki pembayaran langganan yang masih tertunda. Silakan selesaikan pembayaran tersebut atau tunggu hingga kedaluwarsa.",
					"code":  "pending_subscription_exists",
				})
				return
			}

			var plan struct {
				ID    int     `db:"id"`
				Name  string  `db:"name"`
				Price float64 `db:"price"`
			}
			err = db.Get(&plan, "SELECT id, name, price FROM subscription_plans WHERE id = ?", *req.PlanID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Plan not found"})
				return
			}

			amount = int(plan.Price)
			months := req.Months
			if months <= 0 {
				months = 1
			}
			if months > 12 {
				months = 12
			}
			totalPrice := amount * months
			amount = totalPrice // Use total for Tripay payload

			// Get user details for customer info
			emailCtx, _ := c.Get("email")
			customerEmail = emailCtx.(string)
			customerName = "User"

			userType, _ := c.Get("user_type")
			if userType == "organization" {
				db.Get(&customerName, "SELECT name FROM organizations WHERE uuid = ?", userID.(string))
				db.Get(&customerPhone, "SELECT phone FROM organizations WHERE uuid = ?", userID.(string))
			} else if userType == "club" {
				db.Get(&customerName, "SELECT name FROM clubs WHERE uuid = ?", userID.(string))
				db.Get(&customerPhone, "SELECT phone FROM clubs WHERE uuid = ?", userID.(string))
			}

			if customerPhone == "" {
				customerPhone = "08123456789"
			}

			orderItems = []gin.H{
				{
					"sku":      fmt.Sprintf("SUB-%d", plan.ID),
					"name":     fmt.Sprintf("Langganan - %s", plan.Name),
					"price":    int(plan.Price),
					"quantity": months,
				},
			}
		} else {
			// Default to registration
			if req.RegistrationID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "RegistrationID is required for registration type"})
				return
			}

			// Get participant info. Note: multiple records might exist for the same archer+event.
			// We take one to get the archer info and total amount.
			type ParticipantReg struct {
				UUID          string  `db:"uuid"`
				EventID       string  `db:"event_id"`
				ArcherID      string  `db:"archer_id"`
				PaymentAmount float64 `db:"payment_amount"`
				FullName      string  `db:"full_name"`
				Email         *string `db:"email"`
				Phone         *string `db:"phone"`
			}
			var reg ParticipantReg
			err := db.Get(&reg, `
				SELECT ep.uuid, ep.event_id, ep.archer_id, ep.payment_amount,
				       a.full_name, a.email, a.phone
				FROM event_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
			WHERE ep.uuid = ? AND (a.uuid = ? OR EXISTS(SELECT 1 FROM events e WHERE e.uuid = ep.event_id AND e.organizer_id = ?))
			`, *req.RegistrationID, userID.(string), userID.(string))

			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Registration not found"})
				return
			}

			amount = int(reg.PaymentAmount)
			customerName = reg.FullName
			customerEmail = utils.StringValue(reg.Email, "user@archeryhub.id")
			customerPhone = utils.StringValue(reg.Phone, "")
			registrationID = req.RegistrationID

			orderItems = []gin.H{
				{
					"sku":      "EVENT-REG",
					"name":     fmt.Sprintf("Event Registration - %s", reg.FullName),
					"price":    amount,
					"quantity": 1,
				},
			}
		}

		tripay := utils.NewTripayClient()
		merchantRef := fmt.Sprintf("PAY-%s", uuid.New().String()[:12])

		signature := tripay.GenerateSignature(merchantRef, amount)

		expiredTime := time.Now().Add(24 * time.Hour).Unix()

		payload := gin.H{
			"method":         req.Method,
			"merchant_ref":   merchantRef,
			"amount":         amount,
			"customer_name":  customerName,
			"customer_email": customerEmail,
			"customer_phone": customerPhone,
			"order_items":    orderItems,
			"signature":      signature,
			"expired_time":   expiredTime,
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

		var eventID *string
		if req.EventID != "" {
			eventID = &req.EventID
		}

		// Extract instructions if available (Tripay returns them as a slice of maps/structs)
		var instructionsJSON *string
		if inst, ok := tripayResult["instructions"]; ok {
			instBytes, _ := json.Marshal(inst)
			instStr := string(instBytes)
			instructionsJSON = &instStr
		}

		transaction := models.PaymentTransaction{
			UUID:            transactionID,
			Reference:       merchantRef,
			TripayReference: &tripayRef,
			UserID:             userID.(string),
			EventID:            eventID,
			RegistrationID:     registrationID,
			SubscriptionPlanID: req.PlanID,
			Amount:             float64(amount),
			FeeAmount:          0, // We'll calculate this better later if needed
			TotalAmount:        float64(amount),
			PaymentMethod:      utils.StringPtr(req.Method),
			VANumber:           utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			QRURL:              utils.InterfaceToStringPtr(tripayResult["qr_url"]),
			CheckoutURL:        utils.InterfaceToStringPtr(tripayResult["checkout_url"]),
			PayCode:            utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			Instructions:       instructionsJSON,
			Months:             req.Months,
			Status:             "pending",
			ExpiredAt:          expiredAt,
		}
		// Set default months if not subscription it should be 1
		if transaction.Months <= 0 {
			transaction.Months = 1
		}

		query := `
			INSERT INTO payment_transactions (
				uuid, reference, tripay_reference, user_id, event_id, registration_id, subscription_plan_id,
				amount, fee_amount, total_amount, payment_method, va_number, qr_url,
				checkout_url, pay_code, instructions, months, status, expired_at
			) VALUES (
				:uuid, :reference, :tripay_reference, :user_id, :event_id, :registration_id, :subscription_plan_id,
				:amount, :fee_amount, :total_amount, :payment_method, :va_number, :qr_url,
				:checkout_url, :pay_code, :instructions, :months, :status, :expired_at
			)
		`
		_, err = db.NamedExec(query, transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction: " + err.Error()})
			return
		}

		// Update participant registration with payment_id
		if registrationID != nil {
			_, err = db.Exec("UPDATE event_participants SET payment_id = ?, payment_status = 'pending' WHERE uuid = ?", transactionID, *registrationID)
			if err != nil {
				fmt.Printf("Warning: Failed to update participant: %v\n", err)
			}
		}

		c.JSON(http.StatusOK, transaction)
	}
}

// PaymentCallback handles Tripay webhook notifications
func PaymentCallback(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tripay := utils.NewTripayClient()

		// 1. Verify Callback Event
		event := c.GetHeader("X-Callback-Event")
		if event != "payment_status" {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "Unrecognized callback event: " + event})
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Failed to read body"})
			return
		}

		// 2. Verify Signature
		signature := c.GetHeader("X-Callback-Signature")
		if !tripay.VerifyCallbackSignature(body, signature) {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "Invalid signature"})
			return
		}

		var payload struct {
			Reference       string `json:"reference"`
			MerchantRef     string `json:"merchant_ref"`
			Status          string `json:"status"`
			IsClosedPayment int    `json:"is_closed_payment"`
		}

		if err := json.Unmarshal(body, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid payload"})
			return
		}

		// 3. Update transaction status
		var transaction struct {
			UUID               string  `db:"uuid"`
			UserID             string  `db:"user_id"`
			EventID            *string `db:"event_id"`
			RegistrationID     *string `db:"registration_id"`
			SubscriptionPlanID *int    `db:"subscription_plan_id"`
			Months             int     `db:"months"`
			Status             string  `db:"status"`
		}
		err = db.Get(&transaction, "SELECT uuid, user_id, event_id, registration_id, subscription_plan_id, months, status FROM payment_transactions WHERE reference = ?", payload.MerchantRef)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "Transaction not found: " + payload.MerchantRef})
			return
		}

		// Idempotency: skip if already paid
		if transaction.Status == "paid" {
			c.JSON(http.StatusOK, gin.H{"success": true})
			return
		}

		transactionID := transaction.UUID
		registrationID := transaction.RegistrationID
		eventID := transaction.EventID

		status := "pending"
		if payload.Status == "PAID" {
			status = "paid"
		} else if payload.Status == "EXPIRED" {
			status = "expired"
		} else if payload.Status == "FAILED" {
			status = "failed"
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to start transaction"})
			return
		}

		_, err = tx.Exec("UPDATE payment_transactions SET status = ?, callback_data = ?, paid_at = ? WHERE uuid = ?", status, body, time.Now(), transactionID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update transaction"})
			return
		}

		// Update registration if applicable
		if registrationID != nil {
			statusMap := map[string]string{
				"paid":    "lunas",
				"expired": "expired",
				"failed":  "failed",
			}
			regStatus := statusMap[status]
			if regStatus == "" {
				regStatus = "menunggu acc" // fallback or keep as is
			}

			// Update event_participants using payment_id or uuid
			_, err = tx.Exec("UPDATE event_participants SET payment_status = ? WHERE payment_id = ? OR uuid = ?", regStatus, transactionID, *registrationID)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update participant registration"})
				return
			}

			// If paid, maybe generate QR codes here too?
			// The RegisterParticipant handler already has logic for QR if status is Lunas.
			// We should probably trigger it or just ensure it happens.
		}

		// Update event status if platform fee is paid
		if status == "paid" && registrationID == nil && eventID != nil && transaction.SubscriptionPlanID == nil {
			_, err = tx.Exec("UPDATE events SET status = 'published' WHERE uuid = ?", *eventID)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update event status"})
				return
			}
		}

		// Update subscription if applicable
		if status == "paid" && transaction.SubscriptionPlanID != nil {
			var plan struct {
				Type       string `db:"type"`
				TargetType string `db:"target_type"`
			}
			err = db.Get(&plan, "SELECT type, target_type FROM subscription_plans WHERE id = ?", *transaction.SubscriptionPlanID)
			if err == nil {
				months := 1
				if plan.Type == "yearly" {
					months = 12
				}

				table := "clubs"
				if plan.TargetType == "organization" {
					table = "organizations"
				}

				// Calculate next expiration
				var currentExpires *time.Time
				_ = db.Get(&currentExpires, "SELECT subscription_expires_at FROM "+table+" WHERE user_id = ?", transaction.UserID)

				effectiveMonths := transaction.Months
				if effectiveMonths <= 0 {
					effectiveMonths = months // Fallback to plan default
				}

				// The user wants to preserve remaining days ("upgrading or downgrading package doesnt reset previous days")
				// Example: If user has 14 days of Standard and buys Elite, they get Elite for 30 + 14 = 44 days.
				// Example: If user has 14 days of Elite and buy Standard, they get Standard for 30 + 14 = 44 days.
				// This ensures the user never loses the time they've already paid for.
				now := time.Now()
				baseTime := now
				if currentExpires != nil && currentExpires.After(now) {
					baseTime = *currentExpires
				}

				// Add months to base time. We use AddDate for calendar month precision.
				newExpiry := baseTime.AddDate(0, effectiveMonths, 0)

				_, err = tx.Exec("UPDATE "+table+" SET subscription_plan_id = ?, subscription_status = 'active', subscription_expires_at = ? WHERE user_id = ?", *transaction.SubscriptionPlanID, newExpiry, transaction.UserID)
				if err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to update subscription"})
					return
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}


// GetPaymentStatus returns the status of a payment transaction with enriched details
func GetPaymentStatus(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reference := c.Param("reference")

		type EnrichedTransaction struct {
			models.PaymentTransaction
			Description string  `json:"description" db:"description"`
			PlanName    *string `json:"plan_name" db:"plan_name"`
			EventName   *string `json:"event_name" db:"event_name"`
			AthleteName *string `json:"athlete_name" db:"athlete_name"`
			Division    *string `json:"division" db:"division"`
			Category    *string `json:"category" db:"category"`
		}

		var transaction EnrichedTransaction
		query := `
			SELECT 
				t.*,
				CASE 
					WHEN t.subscription_plan_id IS NOT NULL THEN p.name
					WHEN t.registration_id IS NOT NULL THEN CONCAT('Registrasi: ', a.full_name)
					WHEN t.event_id IS NOT NULL THEN CONCAT('Platform Fee: ', e.name)
					ELSE 'Transaksi Archeryhub'
				END as description,
				p.name as plan_name,
				e.name as event_name,
				a.full_name as athlete_name,
				rbt.name as division,
				COALESCE(ec.category_name_custom, rag.name) as category
			FROM payment_transactions t
			LEFT JOIN subscription_plans p ON t.subscription_plan_id = p.id
			LEFT JOIN event_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN events e ON t.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			WHERE t.reference = ?
		`
		err := db.Get(&transaction, query, reference)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}

		c.JSON(http.StatusOK, transaction)
	}
}

// GetEventPayments returns all paid transactions for a specific event
func GetEventPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var actualEventID string
		err := db.Get(&actualEventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		type PaymentItem struct {
			Reference     string    `json:"reference" db:"reference"`
			PayMethod     *string   `json:"payment_method" db:"payment_method"`
			Amount        float64   `json:"amount" db:"amount"`
			Fee           float64   `json:"fee_amount" db:"fee_amount"`
			Total         float64   `json:"total_amount" db:"total_amount"`
			Status        string    `json:"status" db:"status"`
			PaidAt        *time.Time `json:"paid_at" db:"paid_at"`
			CreatedAt     time.Time `json:"created_at" db:"created_at"`
			AthleteName   *string   `json:"athlete_name" db:"athlete_name"`
		}

		var payments []PaymentItem
		query := `
			SELECT 
				t.reference, t.payment_method, t.amount, t.fee_amount, t.total_amount, 
				t.status, t.paid_at, t.created_at,
				a.full_name as athlete_name
			FROM payment_transactions t
			LEFT JOIN event_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE t.event_id = ? AND t.status = 'paid'
			ORDER BY t.paid_at DESC
		`
		err = db.Select(&payments, query, actualEventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event payments"})
			return
		}

		c.JSON(http.StatusOK, payments)
	}
}

// GetOrganizationEarningsSummary returns aggregated earnings per event for an organization
func GetOrganizationEarningsSummary(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		type EventSummary struct {
			UUID         string  `json:"id" db:"uuid"`
			EventName    string  `json:"eventName" db:"name"`
			Category     string  `json:"category" db:"category_label"`
			EndDate      string  `json:"date" db:"end_date"`
			Participants int     `json:"participants" db:"participant_count"`
			TotalAmount  float64 `json:"amount" db:"total_amount"`
		}

		var summaries []EventSummary
		query := `
			SELECT 
				e.uuid, e.name, COALESCE(e.location_type, 'Event') as category_label, 
				e.end_date,
				COUNT(DISTINCT ep.uuid) as participant_count,
				COALESCE(SUM(t.amount), 0) as total_amount
			FROM events e
			LEFT JOIN event_participants ep ON e.uuid = ep.event_id
			LEFT JOIN payment_transactions t ON ep.uuid = t.registration_id AND t.status = 'paid'
			WHERE e.organizer_id = ?
			GROUP BY e.uuid
			ORDER BY e.created_at DESC
		`
		err := db.Select(&summaries, query, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch earnings summary", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, summaries)
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

// EventPaymentMethod represents a payment method for an event
type EventPaymentMethod struct {
	UUID          string  `json:"uuid" db:"uuid"`
	EventID       string  `json:"event_id" db:"event_id"`
	PaymentMethod string  `json:"payment_method" db:"payment_method"`
	AccountName   *string `json:"account_name" db:"account_name"`
	AccountNumber *string `json:"account_number" db:"account_number"`
	Instructions  *string `json:"instructions" db:"instructions"`
	IsActive      bool    `json:"is_active" db:"is_active"`
	DisplayOrder  int     `json:"display_order" db:"display_order"`
	CreatedAt     string  `json:"created_at" db:"created_at"`
	UpdatedAt     string  `json:"updated_at" db:"updated_at"`
}

// GetEventPaymentMethods returns all payment methods for an event
func GetEventPaymentMethods(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Check if it's a slug or UUID, get event organizer_id
		var organizerID string
		err := db.Get(&organizerID, "SELECT organizer_id FROM events WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		var methods []EventPaymentMethod
		err = db.Select(&methods, `
			SELECT uuid, user_id as event_id, bank_name as payment_method, account_name, account_number, 
			       '' as instructions, 1 as is_active, 0 as display_order, created_at, updated_at
			FROM bank_accounts
			WHERE user_id = ? AND status = 'verified'
			ORDER BY is_primary DESC, created_at ASC
		`, organizerID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment methods", "details": err.Error()})
			return
		}

		if len(methods) == 0 {
			// If no verified bank accounts, fetch all bank accounts (fallback/for testing)
			err = db.Select(&methods, `
				SELECT uuid, user_id as event_id, bank_name as payment_method, account_name, account_number, 
				       '' as instructions, 1 as is_active, 0 as display_order, created_at, updated_at
				FROM bank_accounts
				WHERE user_id = ?
				ORDER BY is_primary DESC, created_at ASC
			`, organizerID)
		}

		if methods == nil {
			methods = []EventPaymentMethod{}
		}

		c.JSON(http.StatusOK, gin.H{"data": methods})
	}
}

// CreateEventPaymentMethod creates a new payment method for an event
func CreateEventPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			PaymentMethod string  `json:"payment_method" binding:"required"`
			AccountName   *string `json:"account_name"`
			AccountNumber *string `json:"account_number"`
			Instructions  *string `json:"instructions"`
			DisplayOrder  int     `json:"display_order"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		methodID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO event_payment_methods 
			(uuid, event_id, payment_method, account_name, account_number, instructions, display_order)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, methodID, eventID, req.PaymentMethod, req.AccountName, req.AccountNumber, req.Instructions, req.DisplayOrder)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment method"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"uuid":    methodID,
			"message": "Payment method created successfully",
		})
	}
}

// UpdateEventPaymentMethod updates a payment method
func UpdateEventPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		methodID := c.Param("methodId")

		var req struct {
			PaymentMethod string  `json:"payment_method"`
			AccountName   *string `json:"account_name"`
			AccountNumber *string `json:"account_number"`
			Instructions  *string `json:"instructions"`
			IsActive      *bool   `json:"is_active"`
			DisplayOrder  *int    `json:"display_order"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE event_payment_methods 
			SET payment_method = COALESCE(?, payment_method),
			    account_name = ?,
			    account_number = ?,
			    instructions = ?,
			    is_active = COALESCE(?, is_active),
			    display_order = COALESCE(?, display_order),
			    updated_at = NOW()
			WHERE uuid = ?
		`, req.PaymentMethod, req.AccountName, req.AccountNumber, req.Instructions, req.IsActive, req.DisplayOrder, methodID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment method"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Payment method updated successfully"})
	}
}

// DeleteEventPaymentMethod deletes a payment method
func DeleteEventPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		methodID := c.Param("methodId")

		_, err := db.Exec("DELETE FROM event_payment_methods WHERE uuid = ?", methodID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment method"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Payment method deleted successfully"})
	}
}
// SimulatePaymentSuccess simulates a successful payment for testing
func SimulatePaymentSuccess(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reference := c.Param("reference")

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		var transaction models.PaymentTransaction
		err = tx.Get(&transaction, "SELECT * FROM payment_transactions WHERE reference = ?", reference)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
			return
		}

		if transaction.Status == "paid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Already paid"})
			return
		}

		now := time.Now()
		_, err = tx.Exec("UPDATE payment_transactions SET status = 'paid', paid_at = ? WHERE reference = ?", now, reference)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update transaction"})
			return
		}

		if transaction.RegistrationID != nil {
			_, err = tx.Exec("UPDATE event_participants SET payment_status = 'lunas' WHERE payment_id = ? OR uuid = ?", transaction.UUID, *transaction.RegistrationID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update participant status"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Payment simulation successful", "reference": reference})
	}
}
// GetMyPayments returns the authenticated user's payment history
func GetMyPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		limit := c.DefaultQuery("limit", "10")
		offset := c.DefaultQuery("offset", "0")

		limitInt, _ := strconv.Atoi(limit)
		offsetInt, _ := strconv.Atoi(offset)

		query := `
			SELECT 
				pt.*,
				e.name as event_name,
				sp.name as plan_name
			FROM payment_transactions pt
			LEFT JOIN events e ON pt.event_id = e.uuid
			LEFT JOIN subscription_plans sp ON pt.subscription_plan_id = sp.id
			WHERE pt.user_id = ?
			ORDER BY pt.created_at DESC
			LIMIT ? OFFSET ?
		`

		type PaymentWithExtra struct {
			models.PaymentTransaction
			EventName *string `json:"event_name" db:"event_name"`
			PlanName  *string `json:"plan_name" db:"plan_name"`
		}

		var payments []PaymentWithExtra
		err := db.Select(&payments, query, userID, limitInt, offsetInt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments", "details": err.Error()})
			return
		}

		// Count total for pagination
		var total int
		db.Get(&total, "SELECT COUNT(*) FROM payment_transactions WHERE user_id = ?", userID)

		c.JSON(http.StatusOK, gin.H{
			"payments": payments,
			"total":    total,
		})
	}
}
// CreateParticipantPayment handles creating a payment for a specific participant registration
func CreateParticipantPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		participantID := c.Param("participantId")
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req struct {
			Method string `json:"method" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Fetch participant and archer info
		type ParticipantReg struct {
			UUID          string  `db:"uuid"`
			EventID       string  `db:"event_id"`
			ArcherID      string  `db:"archer_id"`
			PaymentAmount float64 `db:"payment_amount"`
			FullName      string  `db:"full_name"`
			Email         *string `db:"email"`
			Phone         *string `db:"phone"`
		}
		var reg ParticipantReg
		err := db.Get(&reg, `
			SELECT ep.uuid, ep.event_id, ep.archer_id, ep.payment_amount,
				   a.full_name, a.email, a.phone
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			WHERE ep.uuid = ? AND (a.uuid = ? OR EXISTS(SELECT 1 FROM events e WHERE e.uuid = ep.event_id AND e.organizer_id = ?))
		`, participantID, userID.(string), userID.(string))

		if err != nil {
			fmt.Printf("[DEBUG] CreateParticipantPayment error: %v, ID: %s, User: %s\n", err, participantID, userID)
			c.JSON(http.StatusNotFound, gin.H{"error": "Registration not found"})
			return
		}

		amount := int(reg.PaymentAmount)
		customerName := reg.FullName
		customerEmail := utils.StringValue(reg.Email, "user@archeryhub.id")
		customerPhone := utils.StringValue(reg.Phone, "08123456789")

		tripay := utils.NewTripayClient()
		merchantRef := fmt.Sprintf("PAY-REG-%s", uuid.New().String()[:12])
		signature := tripay.GenerateSignature(merchantRef, amount)
		expiredTime := time.Now().Add(24 * time.Hour).Unix()

		orderItems := []gin.H{
			{
				"sku":      "EVENT-REG",
				"name":     fmt.Sprintf("Event Registration - %s", reg.FullName),
				"price":    amount,
				"quantity": 1,
			},
		}

		payload := gin.H{
			"method":         req.Method,
			"merchant_ref":   merchantRef,
			"amount":         amount,
			"customer_name":  customerName,
			"customer_email": customerEmail,
			"customer_phone": customerPhone,
			"order_items":    orderItems,
			"signature":      signature,
			"expired_time":   expiredTime,
		}

		tripayResult, err := tripay.CreateTransaction(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Tripay transaction: " + err.Error()})
			return
		}

		// Save transaction
		transactionID := uuid.New().String()
		tripayRef := tripayResult["reference"].(string)
		expiredAt := time.Now().Add(24 * time.Hour)
		if exp, ok := tripayResult["expiry_date"].(float64); ok {
			expiredAt = time.Unix(int64(exp), 0)
		}

		// Instructions
		var instructionsJSON *string
		if inst, ok := tripayResult["instructions"]; ok {
			instBytes, _ := json.Marshal(inst)
			instStr := string(instBytes)
			instructionsJSON = &instStr
		}

		transaction := models.PaymentTransaction{
			UUID:            transactionID,
			Reference:       merchantRef,
			TripayReference: &tripayRef,
			UserID:             userID.(string),
			EventID:            &reg.EventID,
			RegistrationID:     &reg.UUID,
			Amount:             float64(amount),
			FeeAmount:          0,
			TotalAmount:        float64(amount),
			PaymentMethod:      utils.StringPtr(req.Method),
			VANumber:           utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			QRURL:              utils.InterfaceToStringPtr(tripayResult["qr_url"]),
			CheckoutURL:        utils.InterfaceToStringPtr(tripayResult["checkout_url"]),
			PayCode:            utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			Instructions:       instructionsJSON,
			Months:             1,
			Status:             "pending",
			ExpiredAt:          expiredAt,
		}

		query := `
			INSERT INTO payment_transactions (
				uuid, reference, tripay_reference, user_id, event_id, registration_id,
				amount, fee_amount, total_amount, payment_method, va_number, qr_url,
				checkout_url, pay_code, instructions, months, status, expired_at
			) VALUES (
				:uuid, :reference, :tripay_reference, :user_id, :event_id, :registration_id,
				:amount, :fee_amount, :total_amount, :payment_method, :va_number, :qr_url,
				:checkout_url, :pay_code, :instructions, :months, :status, :expired_at
			)
		`
		_, err = db.NamedExec(query, transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save transaction: " + err.Error()})
			return
		}

		// Update participant registration
		_, err = db.Exec("UPDATE event_participants SET payment_id = ?, payment_status = 'pending' WHERE uuid = ?", transactionID, reg.UUID)
		if err != nil {
			fmt.Printf("Warning: Failed to update participant: %v\n", err)
		}

		c.JSON(http.StatusOK, transaction)
	}
}
