package handler

import (
	"context"
	"Archeris-api/models"
	"Archeris-api/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
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
		err := db.Get(&event, "SELECT uuid FROM tournaments WHERE uuid = ?", eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Dynamic entry fee and quota from event categories or event default
		var cat struct {
			UUID                string  `db:"uuid"`
			Fee                 float64 `db:"fee"`
			Quota               int     `db:"quota"`
			CurrentParticipants int     `db:"current_participants"`
		}
		catErr := db.Get(&cat, `
			SELECT uuid, COALESCE(fee, 0) as fee, COALESCE(quota, 0) as quota, COALESCE(current_participants, 0) as current_participants 
			FROM tournament_categories 
			WHERE (event_id = ? OR event_id = ?) AND (category_name = ? OR name = ? OR CONCAT(division_name, ' ', category_name) = ?)
			LIMIT 1
		`, eventID, event.UUID, req.Category, req.Category, req.Category)

		entryFee := event.EntryFee
		if catErr == nil && cat.Fee > 0 {
			entryFee = cat.Fee
		}
		if catErr == nil && cat.Quota > 0 && cat.CurrentParticipants >= cat.Quota {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kuota untuk kategori ini sudah penuh"})
			return
		}

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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftar: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, registration)
	}
}

// CreatePayment handles creating a payment transaction (Mayar, PayPal, or Manual)
func CreatePayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		var req models.CreatePaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Method == "" {
			if req.PaymentMethod != "" {
				req.Method = req.PaymentMethod
			} else if req.Gateway != "" {
				req.Method = req.Gateway
			}
		}
		if req.Method == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Metode pembayaran (method / payment_method) wajib diisi"})
			return
		}

		var amount int
		var floatAmount float64
		var customerName, customerEmail, customerPhone string
		var registrationID *string
		var allRegIDs []string
		var eventID *string
		if req.EventID != "" {
			eventID = &req.EventID
			var psStr *string
			_ = db.Get(&psStr, "SELECT page_settings FROM tournaments WHERE uuid = ? OR slug = ?", req.EventID, req.EventID)
			if psStr != nil && *psStr != "" {
				var ps struct {
					Currency string `json:"currency"`
				}
				if errJson := json.Unmarshal([]byte(*psStr), &ps); errJson == nil && ps.Currency != "" {
					if strings.ToUpper(ps.Currency) == "IDR" && strings.ToLower(req.Method) == "paypal" {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "Turnamen dengan mata uang IDR tidak mendukung pembayaran via PayPal",
							"code":  "gateway_currency_mismatch",
						})
						return
					}
					if strings.ToUpper(ps.Currency) == "USD" && strings.ToLower(req.Method) != "paypal" {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "Turnamen internasional dengan mata uang USD hanya mendukung pembayaran via PayPal",
							"code":  "gateway_currency_mismatch",
						})
						return
					}
				}
			}
		}

		if req.Type == "platform_fee" {
			// Get event details
			var event models.Event
			err := db.Get(&event, "SELECT * FROM tournaments WHERE uuid = ? AND organizer_id = ?", req.EventID, userID.(string))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan atau tidak diizinkan"})
				return
			}

			// Check if already has a pending platform fee for this event
			var existingPending int
			err = db.Get(&existingPending, "SELECT COUNT(*) FROM payment_transactions WHERE tournament_id = ? AND subscription_plan_id IS NULL AND registration_id IS NULL AND status = 'pending' AND expired_at > NOW()", req.EventID)
			if err == nil && existingPending > 0 {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Anda memiliki pembayaran biaya platform untuk turnamen ini yang masih tertunda. Silakan selesaikan di riwayat transaksi.",
					"code":  "pending_platform_fee_exists",
				})
				return
			}

			amount = 50000
			if envFee := os.Getenv("PLATFORM_FEE_AMOUNT"); envFee != "" {
				if feeVal, err := strconv.Atoi(envFee); err == nil && feeVal > 0 {
					amount = feeVal
				}
			}

			// Get user details for customer info
			emailCtx, _ := c.Get("email")
			customerEmail = emailCtx.(string)
			customerName = "Organizer"
			customerPhone = "08123456789" // Fallback

			userType, _ := c.Get("user_type")
			if userType == "organizer" {
				db.Get(&customerName, "SELECT name FROM organizers WHERE uuid = ?", userID.(string))
				db.Get(&customerPhone, "SELECT phone FROM organizers WHERE uuid = ?", userID.(string))
			} else if userType == "club" {
				db.Get(&customerName, "SELECT name FROM clubs WHERE uuid = ?", userID.(string))
				db.Get(&customerPhone, "SELECT phone FROM clubs WHERE uuid = ?", userID.(string))
			}

			if customerPhone == "" {
				customerPhone = "08123456789"
			}
		} else if req.Type == "subscription" {
			if req.PlanID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "PlanID wajib diisi untuk tipe langganan"})
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
				c.JSON(http.StatusNotFound, gin.H{"error": "Paket tidak ditemukan"})
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
			amount = totalPrice

			// Get user details for customer info
			emailCtx, _ := c.Get("email")
			customerEmail = emailCtx.(string)
			customerName = "User"

			userType, _ := c.Get("user_type")
			if userType == "organizer" {
				db.Get(&customerName, "SELECT name FROM organizers WHERE uuid = ?", userID.(string))
				db.Get(&customerPhone, "SELECT phone FROM organizers WHERE uuid = ?", userID.(string))
			} else if userType == "club" {
				db.Get(&customerName, "SELECT name FROM clubs WHERE uuid = ?", userID.(string))
				db.Get(&customerPhone, "SELECT phone FROM clubs WHERE uuid = ?", userID.(string))
			}

			if customerPhone == "" {
				customerPhone = "08123456789"
			}
		} else {
			allRegIDs = []string{}
			if req.RegistrationID != nil && *req.RegistrationID != "" {
				allRegIDs = append(allRegIDs, *req.RegistrationID)
			}
			for _, id := range req.RegistrationIDs {
				if trimmed := strings.TrimSpace(id); trimmed != "" {
					allRegIDs = append(allRegIDs, trimmed)
				}
			}
			for _, id := range req.ParticipantIDs {
				if trimmed := strings.TrimSpace(id); trimmed != "" {
					allRegIDs = append(allRegIDs, trimmed)
				}
			}

			if len(allRegIDs) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "RegistrationID atau ParticipantIDs wajib diisi untuk tipe registrasi"})
				return
			}

			type ParticipantReg struct {
				UUID          string  `db:"uuid"`
				EventID       string  `db:"event_id"`
				ArcherID      string  `db:"archer_id"`
				PaymentAmount float64 `db:"payment_amount"`
				CatFee        float64 `db:"cat_fee"`
				FeeMode       string  `db:"fee_mode"`
				FeeIndividual float64 `db:"fee_individual"`
				FeeTeam       float64 `db:"fee_team"`
				FeeMixedTeam  float64 `db:"fee_mixed_team"`
				EntryFee      float64 `db:"entry_fee"`
				TypeCode      string  `db:"type_code"`
				FullName      string  `db:"full_name"`
				Email         *string `db:"email"`
				Phone         *string `db:"phone"`
			}
			queryReg, argsReg, errIn := sqlx.In(`
				SELECT ep.uuid, ep.tournament_id as event_id, ep.archer_id, ep.payment_amount,
				       COALESCE(tc.fee, 0) as cat_fee,
				       COALESCE(t.fee_mode, 'per_type') as fee_mode,
				       COALESCE(t.fee_individual, 0) as fee_individual,
				       COALESCE(t.fee_team, 0) as fee_team,
				       COALESCE(t.fee_mixed_team, 0) as fee_mixed_team,
				       COALESCE(t.entry_fee, 0) as entry_fee,
				       COALESCE(rtt.code, 'individual') as type_code,
				       COALESCE(a.full_name, 'Peserta Panahan') as full_name, a.email, a.phone
				FROM tournament_participants ep
				LEFT JOIN archers a ON ep.archer_id = a.uuid OR ep.archer_id = a.id
				LEFT JOIN tournament_categories tc ON ep.category_id = tc.uuid
				LEFT JOIN tournaments t ON ep.tournament_id = t.uuid
				LEFT JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
				WHERE ep.uuid IN (?)
			`, allRegIDs)
			if errIn != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyusun query registrasi", "details": errIn.Error()})
				return
			}
			queryReg = db.Rebind(queryReg)
			var regs []ParticipantReg
			err := db.Select(&regs, queryReg, argsReg...)
			if err != nil || len(regs) == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "Registrasi tidak ditemukan", "reg_ids": allRegIDs})
				return
			}

			totalParticipantAmount := 0.0
			for _, r := range regs {
				itemFee := r.PaymentAmount
				if itemFee <= 0 {
					if r.FeeMode == "per_category" {
						if r.CatFee > 0 {
							itemFee = r.CatFee
						} else if r.EntryFee > 0 {
							itemFee = r.EntryFee
						} else if r.FeeIndividual > 0 {
							itemFee = r.FeeIndividual
						}
					} else {
						switch r.TypeCode {
						case "team":
							if r.FeeTeam > 0 {
								itemFee = r.FeeTeam
							}
						case "mixed_team":
							if r.FeeMixedTeam > 0 {
								itemFee = r.FeeMixedTeam
							}
						default:
							if r.FeeIndividual > 0 {
								itemFee = r.FeeIndividual
							}
						}
						if itemFee <= 0 && r.CatFee > 0 {
							itemFee = r.CatFee
						}
						if itemFee <= 0 && r.EntryFee > 0 {
							itemFee = r.EntryFee
						}
					}
					if itemFee > 0 {
						_, _ = db.Exec("UPDATE tournament_participants SET payment_amount = ? WHERE uuid = ?", itemFee, r.UUID)
					}
				}
				totalParticipantAmount += itemFee
			}

			firstReg := regs[0]
			amount = int(totalParticipantAmount)
			floatAmount = totalParticipantAmount
			if floatAmount <= 0 && req.Amount > 0 {
				floatAmount = req.Amount
				amount = int(req.Amount)
			}
			customerName = firstReg.FullName
			if len(regs) > 1 {
				customerName = fmt.Sprintf("%s (+%d lainnya)", firstReg.FullName, len(regs)-1)
			}
			customerEmail = utils.StringValue(firstReg.Email, "user@archeris.net")
			customerPhone = utils.StringValue(firstReg.Phone, "08123456789")
			registrationID = &firstReg.UUID
			if eventID == nil || *eventID == "" {
				eventID = &firstReg.EventID
			}
		}

		if amount <= 0 && floatAmount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Jumlah pembayaran tidak valid atau bernilai nol"})
			return
		}

		var transaction models.PaymentTransaction
		transactionID := uuid.New().String()
		merchantRef := fmt.Sprintf("PAY-SUB-%s", strings.ToUpper(uuid.New().String()[:8]))

		if req.Method == "manual" {
			// Handle manual payment - redirect to CreateManualPayment logic
			merchantRef = fmt.Sprintf("PAY-MANUAL-%s", strings.ToUpper(uuid.New().String()[:8]))

			transaction = models.PaymentTransaction{
				UUID:               transactionID,
				Reference:          merchantRef,
				UserID:             userID.(string),
				EventID:            eventID,
				RegistrationID:     registrationID,
				SubscriptionPlanID: req.PlanID,
				Amount:             float64(amount),
				FeeAmount:          0,
				TotalAmount:        float64(amount),
				PaymentMethod:      utils.StringPtr("manual"),
				Months:             req.Months,
				Status:             "pending",
				ExpiredAt:          utils.TimePtr(time.Now().Add(7 * 24 * time.Hour)), // 7 days for manual payment
			}
		} else {
			appURL := os.Getenv("APP_URL")
			if appURL == "" {
				appURL = "http://localhost:3003"
			}

			description := "Pembayaran Transaksi ArcheryHub"
			if req.Type == "subscription" {
				description = fmt.Sprintf("Langganan Paket ArcheryHub %d Bulan", req.Months)
			} else if registrationID != nil {
				description = fmt.Sprintf("Registrasi Event: %s", customerName)
			}

			// Validate currency against gateway for tournaments
			eventCurrency := "IDR"
			if eventID != nil && *eventID != "" {
				var psStr *string
				_ = db.Get(&psStr, "SELECT page_settings FROM tournaments WHERE uuid = ? OR slug = ?", *eventID, *eventID)
				if psStr != nil && *psStr != "" {
					var ps struct {
						Currency string `json:"currency"`
					}
					if errJson := json.Unmarshal([]byte(*psStr), &ps); errJson == nil && ps.Currency != "" {
						eventCurrency = strings.ToUpper(ps.Currency)
					}
				}

				if eventCurrency == "IDR" && req.Method == "paypal" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "Turnamen dengan mata uang IDR tidak mendukung pembayaran via PayPal",
						"code":  "gateway_currency_mismatch",
					})
					return
				}
				if eventCurrency == "USD" && req.Method != "paypal" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "Turnamen internasional dengan mata uang USD hanya mendukung pembayaran via PayPal",
						"code":  "gateway_currency_mismatch",
					})
					return
				}
			}

			if req.Method == "paypal" {
				paypalClient := utils.NewPayPalClient()
				var usdAmount string
				if eventCurrency == "USD" {
					usdAmount = fmt.Sprintf("%.2f", floatAmount)
				} else {
					usdAmount = paypalClient.ConvertIDRToUSD(floatAmount)
				}
				returnURL := fmt.Sprintf("%s/payment/status/%s?provider=paypal", strings.TrimSuffix(appURL, "/"), merchantRef)
				cancelURL := fmt.Sprintf("%s/payment/status/%s?cancelled=true", strings.TrimSuffix(appURL, "/"), merchantRef)

				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				orderResp, approveURL, err := paypalClient.CreateOrder(ctx, merchantRef, description, usdAmount, returnURL, cancelURL)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi pembayaran PayPal: " + err.Error()})
					return
				}

				orderID := orderResp.ID
				checkoutURL := approveURL

				transaction = models.PaymentTransaction{
					UUID:               transactionID,
					Reference:          merchantRef,
					GatewayReference:   &orderID,
					UserID:             userID.(string),
					EventID:            eventID,
					RegistrationID:     registrationID,
					SubscriptionPlanID: req.PlanID,
					Amount:             floatAmount,
					FeeAmount:          0,
					TotalAmount:        floatAmount,
					PaymentMethod:      utils.StringPtr("paypal"),
					CheckoutURL:        &checkoutURL,
					Months:             req.Months,
					Status:             "pending",
					ExpiredAt:          utils.TimePtr(time.Now().Add(24 * time.Hour)),
				}
			} else {
				mayarClient := utils.NewMayarClient()
				redirectURL := fmt.Sprintf("%s/payment/status/%s", strings.TrimSuffix(appURL, "/"), merchantRef)

				paymentReq := utils.MayarPaymentReq{
					Name:        fmt.Sprintf("Payment %s", merchantRef),
					Amount:      amount,
					Email:       customerEmail,
					Mobile:      customerPhone,
					Description: description,
					RedirectURL: redirectURL,
				}

				mayarData, err := mayarClient.CreatePaymentRequest(paymentReq)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi pembayaran Mayar: " + err.Error()})
					return
				}

				checkoutURL := mayarData.Link
				mayarTxID := mayarData.TransactionID

				transaction = models.PaymentTransaction{
					UUID:               transactionID,
					Reference:          merchantRef,
					GatewayReference:   &mayarTxID,
					UserID:             userID.(string),
					EventID:            eventID,
					RegistrationID:     registrationID,
					SubscriptionPlanID: req.PlanID,
					Amount:             float64(amount),
					FeeAmount:          0,
					TotalAmount:        float64(amount),
					PaymentMethod:      utils.StringPtr("mayar"),
					CheckoutURL:        &checkoutURL,
					Months:             req.Months,
					Status:             "pending",
					ExpiredAt:          utils.TimePtr(time.Now().Add(24 * time.Hour)),
				}
			}
		}
	// Set default months if not subscription it should be 1
		if transaction.Months <= 0 {
			transaction.Months = 1
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database: " + err.Error()})
			return
		}
		defer tx.Rollback()

		query := `
			INSERT INTO payment_transactions (
				uuid, reference, gateway_reference, user_id, tournament_id, registration_id, subscription_plan_id,
				amount, fee_amount, total_amount, payment_method, va_number, qr_url,
				checkout_url, pay_code, instructions, months, status, expired_at
			) VALUES (
				:uuid, :reference, :gateway_reference, :user_id, :tournament_id, :registration_id, :subscription_plan_id,
				:amount, :fee_amount, :total_amount, :payment_method, :va_number, :qr_url,
				:checkout_url, :pay_code, :instructions, :months, :status, :expired_at
			)
		`
		_, err = tx.NamedExec(query, transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi: " + err.Error()})
			return
		}

		// Update participant registration with payment_id
		if len(allRegIDs) > 0 {
			qUp, argsUp, errIn := sqlx.In("UPDATE tournament_participants SET payment_id = ?, payment_status = 'pending' WHERE uuid IN (?)", transactionID, allRegIDs)
			if errIn == nil {
				qUp = tx.Rebind(qUp)
				_, err = tx.Exec(qUp, argsUp...)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghubungkan transaksi ke peserta: " + err.Error()})
					return
				}
			}
		} else if registrationID != nil {
			_, err := tx.Exec("UPDATE tournament_participants SET payment_id = ?, payment_status = 'pending' WHERE uuid = ?", transactionID, *registrationID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghubungkan transaksi ke peserta: " + err.Error()})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan transaksi: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, transaction)
	}
}

// MayarWebhookCallback handles Mayar webhook notifications (tournaments: payment.received, invoice.paid, payment.status, payment.failed, payment.expired)
func MayarWebhookCallback(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Gagal membaca body webhook"})
			return
		}

		_ = os.MkdirAll("logs", 0755)
		f, errLog := os.OpenFile("logs/mayar-callback.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if errLog == nil {
			defer f.Close()
			logEntry := fmt.Sprintf("[%s] Body: %s\n", time.Now().Format("2006-01-02 15:04:05"), string(bodyBytes))
			f.WriteString(logEntry)
		}

		mayarClient := utils.NewMayarClient()
		authHeader := c.GetHeader("Authorization")
		xSig := c.GetHeader("x-mayar-signature")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			token = xSig
		}
		if token != "" && !mayarClient.VerifyWebhookSecret(token) {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Invalid webhook authorization"})
			return
		}

		var payload struct {
			Event string `json:"event"`
			Data  struct {
				ID                string                 `json:"id"`
				TransactionID     string                 `json:"transactionId"`
				Status            string                 `json:"status"`
				TransactionStatus string                 `json:"transactionStatus"`
				Amount            float64                `json:"amount"`
				CustomerEmail     string                 `json:"customerEmail"`
				CustomerName      string                 `json:"customerName"`
				PaymentMethod     string                 `json:"paymentMethod"`
				PaymentLinkId     string                 `json:"paymentLinkId"`
				InvoiceID         string                 `json:"invoiceId"`
				ExtraData         map[string]interface{} `json:"extraData"`
			} `json:"data"`
		}

		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Payload JSON tidak valid"})
			return
		}

		txID := payload.Data.TransactionID
		if txID == "" {
			txID = payload.Data.ID
		}
		if txID == "" {
			txID = payload.Data.PaymentLinkId
		}
		if txID == "" {
			txID = payload.Data.InvoiceID
		}

		extraRef := ""
		if payload.Data.ExtraData != nil {
			if r, ok := payload.Data.ExtraData["reference"].(string); ok && r != "" {
				extraRef = r
			} else if r, ok := payload.Data.ExtraData["merchant_ref"].(string); ok && r != "" {
				extraRef = r
			}
		}

		isPaid := strings.EqualFold(payload.Data.Status, "SUCCESS") ||
			strings.EqualFold(payload.Data.TransactionStatus, "paid") ||
			payload.Event == "payment.received" ||
			payload.Event == "invoice.paid"

		isFailed := strings.EqualFold(payload.Data.Status, "FAILED") ||
			strings.EqualFold(payload.Data.Status, "EXPIRED") ||
			strings.EqualFold(payload.Data.TransactionStatus, "failed") ||
			strings.EqualFold(payload.Data.TransactionStatus, "expired") ||
			payload.Event == "payment.failed" ||
			payload.Event == "payment.expired"

		if isFailed {
			// Mark as expired / failed in quota_purchases & payment_transactions
			db.Exec("UPDATE quota_purchases SET payment_status = 'expired' WHERE gateway_reference = ? OR payment_reference = ? OR uuid = ? OR (? != '' AND (payment_reference = ? OR gateway_reference = ?))", txID, txID, txID, extraRef, extraRef, extraRef)
			db.Exec("UPDATE payment_transactions SET status = 'expired' WHERE gateway_reference = ? OR reference = ? OR uuid = ? OR (? != '' AND (reference = ? OR gateway_reference = ?))", txID, txID, txID, extraRef, extraRef, extraRef)
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Transaction marked as expired/failed"})
			return
		}

		if !isPaid {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Event ignored"})
			return
		}

		// 1. Check if matching a Quota Purchase
		var qPurchase struct {
			UUID        string `db:"uuid"`
			OrganizerID string `db:"organizer_id"`
			QuotaType   string `db:"quota_type"`
			Quantity    int    `db:"quantity"`
			Status      string `db:"payment_status"`
		}
		errQ := db.Get(&qPurchase, "SELECT uuid, organizer_id, quota_type, quantity, payment_status FROM quota_purchases WHERE gateway_reference = ? OR payment_reference = ? OR uuid = ? OR (? != '' AND (payment_reference = ? OR gateway_reference = ?))", txID, txID, txID, extraRef, extraRef, extraRef)
		if errQ == nil {
			if qPurchase.Status != "paid" {
				tx, errTx := db.Beginx()
				if errTx == nil {
					_, _ = tx.Exec("UPDATE quota_purchases SET payment_status = 'paid', payment_method = 'mayar' WHERE uuid = ?", qPurchase.UUID)
					quotaCol := "quota_standard"
					if qPurchase.QuotaType == "elite" {
						quotaCol = "quota_elite"
					}
					_, _ = tx.Exec("UPDATE organizers SET "+quotaCol+" = "+quotaCol+" + ? WHERE uuid = ?", qPurchase.Quantity, qPurchase.OrganizerID)
					_ = tx.Commit()
				}
			}
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Quota payment processed"})
			return
		}

		// 2. Check payment_transactions
		var transaction struct {
			UUID               string  `db:"uuid"`
			UserID             string  `db:"user_id"`
			EventID            *string `db:"tournament_id"`
			RegistrationID     *string `db:"registration_id"`
			SubscriptionPlanID *int    `db:"subscription_plan_id"`
			Months             int     `db:"months"`
			Status             string  `db:"status"`
		}
		err = db.Get(&transaction, "SELECT uuid, user_id, tournament_id, registration_id, subscription_plan_id, months, status FROM payment_transactions WHERE gateway_reference = ? OR reference = ? OR uuid = ? OR (? != '' AND (reference = ? OR gateway_reference = ?))", txID, txID, txID, extraRef, extraRef, extraRef)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Transaction not found locally, ignored"})
			return
		}

		if transaction.Status == "paid" {
			c.JSON(http.StatusOK, gin.H{"success": true, "message": "Already processed"})
			return
		}

		tx, errTx := db.Beginx()
		if errTx != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Gagal memulai transaksi database"})
			return
		}
		defer tx.Rollback()

		now := time.Now()
		_, _ = tx.Exec("UPDATE payment_transactions SET status = 'paid', paid_at = ?, callback_data = ? WHERE uuid = ?", now, bodyBytes, transaction.UUID)

		if transaction.RegistrationID != nil && *transaction.RegistrationID != "" {
			_, _ = tx.Exec("UPDATE tournament_participants SET payment_status = 'paid' WHERE uuid = ? OR payment_id = ?", *transaction.RegistrationID, transaction.UUID)
		}

		if transaction.RegistrationID == nil && transaction.EventID != nil && transaction.SubscriptionPlanID == nil {
			_, _ = tx.Exec("UPDATE tournaments SET status = 'published' WHERE uuid = ?", *transaction.EventID)
		}

		if transaction.SubscriptionPlanID != nil {
			var plan struct {
				Type       string `db:"type"`
				TargetType string `db:"target_type"`
			}
			err = db.Get(&plan, "SELECT type, target_type FROM subscription_plans WHERE id = ?", *transaction.SubscriptionPlanID)
			if err == nil {
				table := "clubs"
				if plan.TargetType == "organizer" {
					table = "organizers"
				}
				effectiveMonths := transaction.Months
				if effectiveMonths <= 0 {
					effectiveMonths = 1
				}
				newExpiry := now.AddDate(0, effectiveMonths, 0)
				_, _ = tx.Exec("UPDATE "+table+" SET subscription_plan_id = ?, subscription_status = 'active', subscription_expires_at = ? WHERE user_id = ?",
					*transaction.SubscriptionPlanID, newExpiry, transaction.UserID)
			}
		}

		_ = tx.Commit()
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// GetPaymentStatus returns the status of a payment transaction with enriched details
func GetPaymentStatus(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reference := c.Param("reference")

		type EnrichedTransaction struct {
			models.PaymentTransaction
			Description *string `json:"description" db:"description"`
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
					WHEN t.registration_id IS NOT NULL THEN CONCAT('Registrasi: ', COALESCE(a.full_name, 'Peserta'))
					WHEN t.tournament_id IS NOT NULL THEN CONCAT('Platform Fee: ', COALESCE(e.name, 'Turnamen'))
					ELSE 'Transaksi Archeris'
				END as description,
				p.name as plan_name,
				e.name as event_name,
				a.full_name as athlete_name,
				rbt.name as division,
				COALESCE(ec.category_name_custom, rag.name) as category
			FROM payment_transactions t
			LEFT JOIN subscription_plans p ON t.subscription_plan_id = p.id
			LEFT JOIN tournament_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN tournaments e ON t.tournament_id = e.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			WHERE t.reference = ? OR t.gateway_reference = ? OR t.uuid = ?
		`
		err := db.Get(&transaction, query, reference, reference, reference)
		if err != nil {
			fmt.Printf("[DEBUG GetPaymentStatus] reference=%s, err=%v\n", reference, err)
			// Fallback: Check in quota_purchases table
			var qPurchase struct {
				UUID             string    `db:"uuid"`
				OrganizerID      string    `db:"organizer_id"`
				PlanID           int       `db:"plan_id"`
				QuotaType        string    `db:"quota_type"`
				Quantity         int       `db:"quantity"`
				UnitPrice        float64   `db:"unit_price"`
				TotalAmount      float64   `db:"total_amount"`
				Currency         string    `db:"currency"`
				PaymentStatus    string    `db:"payment_status"`
				PaymentMethod    string    `db:"payment_method"`
				PaymentReference string    `db:"payment_reference"`
				PayCode          *string   `db:"pay_code"`
				QRURL            *string   `db:"qr_url"`
				CheckoutURL      *string   `db:"checkout_url"`
				GatewayReference *string   `db:"gateway_reference"`
				Instructions     *string   `db:"instructions"`
				PurchasedAt      time.Time `db:"purchased_at"`
				PlanName         string    `db:"plan_name"`
			}
			qQuery := `
				SELECT q.uuid, q.organizer_id, q.plan_id, q.quota_type, q.quantity, q.unit_price, 
					   q.total_amount, q.currency, q.payment_status, COALESCE(q.payment_method, 'mayar') as payment_method, 
					   q.payment_reference, q.pay_code, q.qr_url, q.checkout_url, q.gateway_reference, q.instructions, q.purchased_at, 
					   COALESCE(p.name, 'Paket Kuota Event') as plan_name
				FROM quota_purchases q
				LEFT JOIN subscription_plans p ON q.plan_id = p.id
				WHERE q.payment_reference = ? OR q.uuid = ? OR q.gateway_reference = ?
				LIMIT 1
			`
			if errQ := db.Get(&qPurchase, qQuery, reference, reference, reference); errQ == nil {
				payCode := ""
				if qPurchase.PayCode != nil && *qPurchase.PayCode != "" && *qPurchase.PayCode != "88300" {
					payCode = *qPurchase.PayCode
				}

				qrURL := ""
				if qPurchase.QRURL != nil && *qPurchase.QRURL != "" {
					qrURL = *qPurchase.QRURL
				}

				checkoutURL := ""
				if qPurchase.CheckoutURL != nil && *qPurchase.CheckoutURL != "" {
					checkoutURL = *qPurchase.CheckoutURL
				}

				instStr := ""
				if qPurchase.Instructions != nil && *qPurchase.Instructions != "" {
					instStr = *qPurchase.Instructions
				} else if checkoutURL != "" {
					instructions := []map[string]interface{}{
						{
							"title": "Pembayaran Online via Mayar (QRIS, VA Bank, E-Wallet)",
							"steps": []string{
								"Klik tombol 'Bayar Sekarang via Mayar' untuk membuka halaman pembayaran resmi.",
								"Pilih metode pembayaran yang Anda inginkan (QRIS, Virtual Account BCA/Mandiri/BRI/BNI/Permata, atau E-Wallet).",
								"Selesaikan pembayaran sesuai petunjuk di halaman Mayar.",
								"Setelah pembayaran berhasil, kuota event Anda akan otomatis aktif secara instan.",
							},
						},
					}
					instBytes, _ := json.Marshal(instructions)
					instStr = string(instBytes)
				}

				c.JSON(http.StatusOK, gin.H{
					"uuid":             qPurchase.UUID,
					"reference":        qPurchase.PaymentReference,
					"status":           qPurchase.PaymentStatus,
					"payment_status":   qPurchase.PaymentStatus,
					"total_amount":     qPurchase.TotalAmount,
					"amount":           qPurchase.TotalAmount,
					"currency":         qPurchase.Currency,
					"payment_method":   qPurchase.PaymentMethod,
					"plan_name":        qPurchase.PlanName,
					"description":      fmt.Sprintf("%s (%d Event)", qPurchase.PlanName, qPurchase.Quantity),
					"quantity":         qPurchase.Quantity,
					"pay_code":         payCode,
					"va_number":        payCode,
					"qr_url":           qrURL,
					"checkout_url":     checkoutURL,
					"instructions":     instStr,
					"created_at":       qPurchase.PurchasedAt,
					"purchased_at":     qPurchase.PurchasedAt,
					"expiry_date":      qPurchase.PurchasedAt.Add(24 * time.Hour).Unix(),
					"participants":     []interface{}{},
					"participant_count": 0,
				})
				return
			}

			c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
			return
		}

		type ParticipantItem struct {
			UUID               string     `json:"uuid" db:"uuid"`
			ArcherID           string     `json:"archer_id" db:"archer_id"`
			ArcherName         string     `json:"archer_name" db:"athlete_name"`
			AthleteName        string     `json:"athlete_name" db:"athlete_name"`
			Gender             *string    `json:"gender" db:"gender"`
			ClubName           *string    `json:"club_name" db:"club_name"`
			CategoryID         *string    `json:"category_id" db:"category_id"`
			CategoryName       *string    `json:"category_name" db:"category_name"`
			DivisionName       *string    `json:"division_name" db:"division_name"`
			AgeGroupName       *string    `json:"age_group_name" db:"age_group_name"`
			PaymentAmount      float64    `json:"payment_amount" db:"payment_amount"`
			PaymentStatus      string     `json:"payment_status" db:"payment_status"`
			LastReregistration *time.Time `json:"last_reregistration_at" db:"last_reregistration_at"`
			QRRaw              *string    `json:"qr_raw" db:"qr_raw"`
		}

		var participants []ParticipantItem
		partQuery := `
			SELECT 
				tp.uuid,
				tp.archer_id,
				COALESCE(a.full_name, 'Peserta') as athlete_name,
				a.gender,
				COALESCE(c.name, '') as club_name,
				tp.category_id,
				COALESCE(tc.category_name_custom, CONCAT_WS(' ', rbt.name, rag.name, rgd.name), '') as category_name,
				COALESCE(rbt.name, '') as division_name,
				COALESCE(rag.name, '') as age_group_name,
				tp.payment_amount,
				tp.payment_status,
				tp.last_reregistration_at,
				tp.qr_raw
			FROM tournament_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs c ON a.club_id = c.uuid
			LEFT JOIN tournament_categories tc ON tp.category_id = tc.uuid
			LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON tc.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
			WHERE (tp.payment_id = ? AND tp.payment_id != '') 
			   OR (tp.payment_id = ? AND tp.payment_id != '') 
			   OR (tp.uuid = ? AND tp.uuid != '')
			ORDER BY tp.created_at ASC
		`
		_ = db.Select(&participants, partQuery, transaction.UUID, transaction.Reference, transaction.RegistrationID)
		if len(participants) == 0 && transaction.EventID != nil && transaction.UserID != "" {
			fallbackPartQuery := `
				SELECT 
					tp.uuid,
					tp.archer_id,
					COALESCE(a.full_name, 'Peserta') as athlete_name,
					a.gender,
					COALESCE(c.name, '') as club_name,
					tp.category_id,
					COALESCE(tc.category_name_custom, CONCAT_WS(' ', rbt.name, rag.name, rgd.name), '') as category_name,
					COALESCE(rbt.name, '') as division_name,
					COALESCE(rag.name, '') as age_group_name,
					tp.payment_amount,
					tp.payment_status,
					tp.last_reregistration_at,
					tp.qr_raw
				FROM tournament_participants tp
				LEFT JOIN archers a ON tp.archer_id = a.uuid
				LEFT JOIN clubs c ON a.club_id = c.uuid
				LEFT JOIN tournament_categories tc ON tp.category_id = tc.uuid
				LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
				LEFT JOIN ref_bow_types rbt ON tc.division_uuid = rbt.uuid
				LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
				WHERE tp.tournament_id = ? AND tp.archer_id = ?
				ORDER BY tp.created_at ASC
			`
			_ = db.Select(&participants, fallbackPartQuery, *transaction.EventID, transaction.UserID)
		}
		if participants == nil {
			participants = []ParticipantItem{}
		}

		var regUser struct {
			FullName string `db:"full_name"`
			Email    string `db:"email"`
		}
		if errU := db.Get(&regUser, `SELECT full_name, COALESCE(email, '') as email FROM archers WHERE uuid = ? OR id = ? LIMIT 1`, transaction.UserID, transaction.UserID); errU == nil {
			transaction.RegisteredByName = &regUser.FullName
			transaction.RegisteredByEmail = &regUser.Email
			if transaction.PayerName == nil || *transaction.PayerName == "" {
				transaction.PayerName = &regUser.FullName
			}
			if transaction.PayerEmail == nil || *transaction.PayerEmail == "" {
				transaction.PayerEmail = &regUser.Email
			}
		}

		type TeamItem struct {
			UUID         string  `json:"uuid" db:"uuid"`
			TeamName     string  `json:"team_name" db:"team_name"`
			CategoryID   string  `json:"category_id" db:"category_id"`
			CategoryName string  `json:"category_name" db:"category_name"`
			DivisionName string  `json:"division_name" db:"division_name"`
			Fee          float64 `json:"fee" db:"fee"`
			Status       string  `json:"status" db:"status"`
			MemberCount  int     `json:"member_count" db:"member_count"`
		}

		var teams []TeamItem
		if transaction.EventID != nil {
			teamQuery := `
				SELECT 
					t.uuid,
					t.team_name,
					t.category_id,
					COALESCE(tc.category_name_custom, CONCAT_WS(' ', rbt.name, rag.name, rgd.name), '') as category_name,
					COALESCE(rbt.name, '') as division_name,
					COALESCE(tc.team_fee, 0) as fee,
					t.status,
					(SELECT COUNT(*) FROM team_members tm WHERE tm.team_id = t.uuid) as member_count
				FROM teams t
				LEFT JOIN tournament_categories tc ON t.category_id = tc.uuid
				LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
				LEFT JOIN ref_bow_types rbt ON tc.division_uuid = rbt.uuid
				LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
				WHERE t.tournament_id = ? AND (
					t.uuid IN (SELECT DISTINCT team_id FROM team_members tm WHERE tm.archer_id IN (SELECT archer_id FROM tournament_participants WHERE payment_id = ? OR uuid = ?))
				)
				ORDER BY t.created_at ASC
			`
			_ = db.Select(&teams, teamQuery, *transaction.EventID, transaction.UUID, transaction.RegistrationID)
		}
		if teams == nil {
			teams = []TeamItem{}
		}

		res := gin.H{
			"uuid":                 transaction.UUID,
			"reference":            transaction.Reference,
			"gateway_reference":    transaction.GatewayReference,
			"user_id":              transaction.UserID,
			"tournament_id":        transaction.EventID,
			"event_id":             transaction.EventID,
			"registration_id":      transaction.RegistrationID,
			"subscription_plan_id": transaction.SubscriptionPlanID,
			"amount":               transaction.Amount,
			"fee_amount":           transaction.FeeAmount,
			"total_amount":         transaction.TotalAmount,
			"payment_method":       transaction.PaymentMethod,
			"payment_channel":      transaction.PaymentChannel,
			"va_number":            transaction.VANumber,
			"qr_url":               transaction.QRURL,
			"checkout_url":         transaction.CheckoutURL,
			"pay_code":             transaction.PayCode,
			"instructions":         transaction.Instructions,
			"status":               transaction.Status,
			"paid_at":              transaction.PaidAt,
			"created_at":           transaction.CreatedAt,
			"updated_at":           transaction.UpdatedAt,
			"expired_at":           transaction.ExpiredAt,
			"proof_url":            transaction.ProofURL,
			"proof_uploaded_at":    transaction.ProofUploadedAt,
			"verified_at":          transaction.VerifiedAt,
			"sender_name":          transaction.SenderName,
			"rejection_reason":     transaction.RejectionReason,
			"registered_by_name":   transaction.RegisteredByName,
			"registered_by_email":  transaction.RegisteredByEmail,
			"payer_name":           transaction.PayerName,
			"payer_email":          transaction.PayerEmail,
			"description":          transaction.Description,
			"plan_name":            transaction.PlanName,
			"event_name":           transaction.EventName,
			"athlete_name":         transaction.AthleteName,
			"division":             transaction.Division,
			"category":             transaction.Category,
			"category_name":        transaction.Category,
			"participants":         participants,
			"participant_count":    len(participants),
			"teams":                teams,
			"team_count":           len(teams),
		}

		c.JSON(http.StatusOK, res)
	}
}

// GetEventPayments returns all paid transactions for a specific event
func GetEventPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var actualEventID string
		err := db.Get(&actualEventID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type PaymentItem struct {
			Reference   string     `json:"reference" db:"reference"`
			PayMethod   *string    `json:"payment_method" db:"payment_method"`
			Amount      float64    `json:"amount" db:"amount"`
			Fee         float64    `json:"fee_amount" db:"fee_amount"`
			Total       float64    `json:"total_amount" db:"total_amount"`
			Status      string     `json:"status" db:"status"`
			PaidAt      *time.Time `json:"paid_at" db:"paid_at"`
			CreatedAt   time.Time  `json:"created_at" db:"created_at"`
			AthleteName *string    `json:"athlete_name" db:"athlete_name"`
		}

		var payments []PaymentItem
		query := `
			SELECT 
				t.reference, t.payment_method, t.amount, t.fee_amount, t.total_amount, 
				t.status, t.paid_at, t.created_at,
				a.full_name as athlete_name
			FROM payment_transactions t
			LEFT JOIN tournament_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE t.tournament_id = ? AND t.status = 'paid'
			ORDER BY t.paid_at DESC
		`
		err = db.Select(&payments, query, actualEventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pembayaran turnamen"})
			return
		}

		c.JSON(http.StatusOK, payments)
	}
}

// GetOrganizationEarningsSummary returns aggregated earnings per tournament for an organizer
func GetOrganizationEarningsSummary(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		type EventSummary struct {
			ID           string  `json:"id" db:"id"`
			UUID         string  `json:"uuid" db:"uuid"`
			Slug         *string `json:"slug" db:"slug"`
			EventName    string  `json:"eventName" db:"name"`
			Category     string  `json:"category" db:"category_label"`
			EndDate      string  `json:"date" db:"end_date"`
			Participants int     `json:"participants" db:"participant_count"`
			TotalAmount  float64 `json:"amount" db:"total_amount"`
		}

		summaries := []EventSummary{}
		query := `
			SELECT 
				COALESCE(NULLIF(e.slug, ''), e.uuid) as id,
				e.uuid, e.slug, e.name, COALESCE(e.location_type, 'Tournament') as category_label, 
				e.end_date,
				COUNT(DISTINCT ep.uuid) as participant_count,
				COALESCE(SUM(t.amount), 0) as total_amount
			FROM tournaments e
			LEFT JOIN tournament_participants ep ON e.uuid = ep.tournament_id
			LEFT JOIN payment_transactions t ON ep.uuid = t.registration_id AND t.status = 'paid'
			WHERE e.organizer_id = ?
			GROUP BY e.uuid, e.slug, e.name, e.location_type, e.end_date
			HAVING COALESCE(SUM(t.amount), 0) > 0
			ORDER BY e.created_at DESC
		`
		err := db.Select(&summaries, query, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil ringkasan pendapatan", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, summaries)
	}
}

// GetOrganizationEarningsDetail returns detailed payments for a specific event
func GetOrganizationEarningsDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		eventIdentifier := c.Param("id")

		// Verify event belongs to organizer (lookup by slug or uuid)
		var event struct {
			UUID string `db:"uuid"`
			Name string `db:"name"`
		}
		err := db.Get(&event, "SELECT uuid, name FROM tournaments WHERE (uuid = ? OR slug = ?) AND organizer_id = ?", eventIdentifier, eventIdentifier, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan atau tidak diizinkan"})
			return
		}

		type PaymentDetail struct {
			UUID          string    `json:"id" db:"uuid"`
			ArcherName    string    `json:"archerName" db:"full_name"`
			ArcherEmail   string    `json:"archerEmail" db:"email"`
			Amount        float64   `json:"amount" db:"amount"`
			Status        string    `json:"status" db:"status"`
			CreatedAt     time.Time `json:"createdAt" db:"created_at"`
			PaymentMethod string    `json:"method" db:"payment_method"`
			Reference     string    `json:"reference" db:"reference"`
		}

		var details []PaymentDetail
		query := `
			SELECT 
				pt.uuid, a.full_name, COALESCE(a.email, '-') as email, pt.amount, pt.status, pt.created_at, 
				COALESCE(pt.payment_method, '-') as payment_method, pt.reference
			FROM payment_transactions pt
			JOIN tournament_participants ep ON pt.registration_id = ep.uuid
			JOIN archers a ON ep.archer_id = a.uuid
			WHERE ep.tournament_id = ? AND pt.status = 'paid'
			ORDER BY pt.created_at DESC
		`
		err = db.Select(&details, query, event.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail pembayaran", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"eventName": event.Name,
			"payments":  details,
		})
	}
}

// GetPaymentChannels returns available Mayar payment channels
func GetPaymentChannels(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		channels := []models.PaymentChannel{
			{
				Code:    "QRIS",
				Name:    "QRIS (All E-Wallet & Mobile Banking)",
				Group:   "QRIS",
				Type:    "direct",
				IconURL: "/payment-method/qris.png",
				Active:  true,
			},
			{
				Code:    "BCAVA",
				Name:    "BCA Virtual Account",
				Group:   "Virtual Account",
				Type:    "direct",
				IconURL: "/payment-method/bca.png",
				Active:  true,
			},
			{
				Code:    "BNIVA",
				Name:    "BNI Virtual Account",
				Group:   "Virtual Account",
				Type:    "direct",
				IconURL: "/payment-method/bni.png",
				Active:  true,
			},
			{
				Code:    "BRIVA",
				Name:    "BRI Virtual Account",
				Group:   "Virtual Account",
				Type:    "direct",
				IconURL: "/payment-method/bri.png",
				Active:  true,
			},
			{
				Code:    "MANDIRIVA",
				Name:    "Mandiri Virtual Account",
				Group:   "Virtual Account",
				Type:    "direct",
				IconURL: "/payment-method/mandiri.png",
				Active:  true,
			},
			{
				Code:    "PERMATAVA",
				Name:    "Permata Virtual Account",
				Group:   "Virtual Account",
				Type:    "direct",
				IconURL: "/payment-method/permata.png",
				Active:  true,
			},
			{
				Code:    "BSIVA",
				Name:    "BSI Virtual Account",
				Group:   "Virtual Account",
				Type:    "direct",
				IconURL: "/payment-method/bsi.png",
				Active:  true,
			},
			{
				Code:    "PAYPAL",
				Name:    "PayPal (International - USD)",
				Group:   "International",
				Type:    "redirect",
				IconURL: "https://www.paypalobjects.com/webstatic/mktg/logo/pp_cc_mark_37x23.jpg",
				Active:  true,
			},
		}
		c.JSON(http.StatusOK, gin.H{"data": channels})
	}
}

// CreateParticipantPayment handles creating a payment for a specific participant registration
func CreateParticipantPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		participantID := c.Param("participantId")
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

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
			SELECT ep.uuid, ep.tournament_id as event_id, ep.archer_id, ep.payment_amount,
				   COALESCE(a.full_name, 'Peserta') as full_name, a.email, a.phone
			FROM tournament_participants ep
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE ep.uuid = ? AND (a.uuid = ? OR EXISTS(SELECT 1 FROM tournaments e WHERE e.uuid = ep.tournament_id AND e.organizer_id = ?))
		`, participantID, userID.(string), userID.(string))

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registrasi tidak ditemukan"})
			return
		}

		amount := int(reg.PaymentAmount)
		customerName := reg.FullName
		customerEmail := utils.StringValue(reg.Email, "user@archeris.net")
		customerPhone := utils.StringValue(reg.Phone, "08123456789")

		var currentStatus string
		_ = db.Get(&currentStatus, "SELECT payment_status FROM tournament_participants WHERE uuid = ?", participantID)
		if currentStatus == "paid" {
			c.JSON(http.StatusConflict, gin.H{"error": "Registrasi ini sudah lunas. Tidak perlu bayar lagi."})
			return
		}

		var reqBody struct {
			Method string `json:"method"`
		}
		_ = c.ShouldBindJSON(&reqBody)

		appURL := os.Getenv("APP_URL")
		if appURL == "" {
			appURL = "http://localhost:3003"
		}
		merchantRef := fmt.Sprintf("PAY-REG-%s", strings.ToUpper(uuid.New().String()[:8]))

		var checkoutURL string
		var externalTxID string
		paymentMethodStr := "mayar"
		if reqBody.Method == "paypal" {
			paymentMethodStr = "paypal"
		}

		// Validate currency against gateway for tournaments
		eventCurrency := "IDR"
		var psStr *string
		_ = db.Get(&psStr, "SELECT page_settings FROM tournaments WHERE uuid = ?", reg.EventID)
		if psStr != nil && *psStr != "" {
			var ps struct {
				Currency string `json:"currency"`
			}
			if errJson := json.Unmarshal([]byte(*psStr), &ps); errJson == nil && ps.Currency != "" {
				eventCurrency = strings.ToUpper(ps.Currency)
			}
		}

		if eventCurrency == "IDR" && paymentMethodStr == "paypal" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Turnamen dengan mata uang IDR tidak mendukung pembayaran via PayPal",
				"code":  "gateway_currency_mismatch",
			})
			return
		}
		if eventCurrency == "USD" && paymentMethodStr != "paypal" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Turnamen internasional dengan mata uang USD hanya mendukung pembayaran via PayPal",
				"code":  "gateway_currency_mismatch",
			})
			return
		}

		if paymentMethodStr == "paypal" {
			paypalClient := utils.NewPayPalClient()
			var usdAmount string
			if eventCurrency == "USD" {
				usdAmount = fmt.Sprintf("%.2f", float64(amount))
			} else {
				usdAmount = paypalClient.ConvertIDRToUSD(float64(amount))
			}
			returnURL := fmt.Sprintf("%s/payment/status/%s?provider=paypal", strings.TrimSuffix(appURL, "/"), merchantRef)
			cancelURL := fmt.Sprintf("%s/payment/status/%s?cancelled=true", strings.TrimSuffix(appURL, "/"), merchantRef)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			orderResp, approveURL, err := paypalClient.CreateOrder(ctx, merchantRef, fmt.Sprintf("Pendaftaran Turnamen: %s", customerName), usdAmount, returnURL, cancelURL)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi pembayaran PayPal: " + err.Error()})
				return
			}
			checkoutURL = approveURL
			externalTxID = orderResp.ID
		} else {
			mayarClient := utils.NewMayarClient()
			redirectURL := fmt.Sprintf("%s/payment/status/%s", strings.TrimSuffix(appURL, "/"), merchantRef)

			paymentReq := utils.MayarPaymentReq{
				Name:        fmt.Sprintf("Tournament Reg - %s", customerName),
				Amount:      amount,
				Email:       customerEmail,
				Mobile:      customerPhone,
				Description: fmt.Sprintf("Pendaftaran Turnamen: %s", customerName),
				RedirectURL: redirectURL,
			}

			mayarData, err := mayarClient.CreatePaymentRequest(paymentReq)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi pembayaran Mayar: " + err.Error()})
				return
			}
			checkoutURL = mayarData.Link
			externalTxID = mayarData.TransactionID
		}

		transactionID := uuid.New().String()

		transaction := models.PaymentTransaction{
			UUID:             transactionID,
			Reference:        merchantRef,
			GatewayReference: &externalTxID,
			UserID:           userID.(string),
			EventID:          &reg.EventID,
			RegistrationID:   &participantID,
			Amount:           float64(amount),
			FeeAmount:        0,
			TotalAmount:      float64(amount),
			PaymentMethod:    utils.StringPtr(paymentMethodStr),
			CheckoutURL:      &checkoutURL,
			Status:           "pending",
			ExpiredAt:        utils.TimePtr(time.Now().Add(24 * time.Hour)),
		}

		query := `
			INSERT INTO payment_transactions (
				uuid, reference, gateway_reference, user_id, tournament_id, registration_id,
				amount, fee_amount, total_amount, payment_method,
				checkout_url, status, expired_at
			) VALUES (
				:uuid, :reference, :gateway_reference, :user_id, :tournament_id, :registration_id,
				:amount, :fee_amount, :total_amount, :payment_method,
				:checkout_url, :status, :expired_at
			)
		`
		_, err = db.NamedExec(query, transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi: " + err.Error()})
			return
		}

		_, _ = db.Exec("UPDATE event_participants SET payment_id = ?, payment_status = 'pending' WHERE uuid = ?", transactionID, reg.UUID)

		c.JSON(http.StatusOK, transaction)
	}
}



// CreateManualPayment creates a manual payment transaction for bank transfer
func CreateManualPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		var req models.CreatePaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate that method is "manual"
		if req.Method != "manual" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Method harus 'manual' untuk pembayaran manual"})
			return
		}

		var amount int
		var registrationID *string
		var eventID *string
		var allRegIDs []string

		if req.Type == "platform_fee" {
			// Get event details
			var event models.Event
			err := db.Get(&event, "SELECT * FROM tournaments WHERE uuid = ? AND organizer_id = ?", req.EventID, userID.(string))
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan atau tidak diizinkan"})
				return
			}

			// Check if already has a pending platform fee for this event
			var existingPending int
			err = db.Get(&existingPending, "SELECT COUNT(*) FROM payment_transactions WHERE tournament_id = ? AND subscription_plan_id IS NULL AND registration_id IS NULL AND status IN ('pending', 'awaiting_verification') AND expired_at > NOW()", req.EventID)
			if err == nil && existingPending > 0 {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Anda memiliki pembayaran biaya platform untuk turnamen ini yang masih tertunda",
					"code":  "pending_platform_fee_exists",
				})
				return
			}

			amount = 50000
			if req.EventID != "" {
				eventID = &req.EventID
			}
		} else if req.Type == "subscription" {
			if req.PlanID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "PlanID wajib diisi untuk tipe langganan"})
				return
			}

			// Check if user already has a pending subscription payment
			var existingPending int
			err := db.Get(&existingPending, "SELECT COUNT(*) FROM payment_transactions WHERE user_id = ? AND subscription_plan_id IS NOT NULL AND status IN ('pending', 'awaiting_verification') AND expired_at > NOW()", userID.(string))
			if err == nil && existingPending > 0 {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Anda memiliki pembayaran langganan yang masih tertunda",
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
				c.JSON(http.StatusNotFound, gin.H{"error": "Paket tidak ditemukan"})
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
			amount = totalPrice
		} else {
			// Default to registration
			if req.RegistrationID != nil && *req.RegistrationID != "" {
				allRegIDs = append(allRegIDs, *req.RegistrationID)
			}
			for _, id := range req.RegistrationIDs {
				if trimmed := strings.TrimSpace(id); trimmed != "" {
					allRegIDs = append(allRegIDs, trimmed)
				}
			}
			for _, id := range req.ParticipantIDs {
				if trimmed := strings.TrimSpace(id); trimmed != "" {
					allRegIDs = append(allRegIDs, trimmed)
				}
			}

			if len(allRegIDs) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "RegistrationID atau ParticipantIDs wajib diisi untuk tipe registrasi"})
				return
			}

			type ParticipantReg struct {
				UUID          string  `db:"uuid"`
				EventID       string  `db:"event_id"`
				ArcherID      string  `db:"archer_id"`
				PaymentAmount float64 `db:"payment_amount"`
				CatFee        float64 `db:"cat_fee"`
				FeeMode       string  `db:"fee_mode"`
				FeeIndividual float64 `db:"fee_individual"`
				FeeTeam       float64 `db:"fee_team"`
				FeeMixedTeam  float64 `db:"fee_mixed_team"`
				EntryFee      float64 `db:"entry_fee"`
				TypeCode      string  `db:"type_code"`
			}
			var regs []ParticipantReg
			qIn, argsIn, errIn := sqlx.In(`
				SELECT ep.uuid, ep.tournament_id as event_id, ep.archer_id, ep.payment_amount,
				       COALESCE(tc.fee, 0) as cat_fee,
				       COALESCE(t.fee_mode, 'per_type') as fee_mode,
				       COALESCE(t.fee_individual, 0) as fee_individual,
				       COALESCE(t.fee_team, 0) as fee_team,
				       COALESCE(t.fee_mixed_team, 0) as fee_mixed_team,
				       COALESCE(t.entry_fee, 0) as entry_fee,
				       COALESCE(rtt.code, 'individual') as type_code
				FROM tournament_participants ep
				LEFT JOIN tournament_categories tc ON ep.category_id = tc.uuid
				LEFT JOIN tournaments t ON ep.tournament_id = t.uuid
				LEFT JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
				WHERE ep.uuid IN (?)
			`, allRegIDs)
			if errIn == nil {
				qIn = db.Rebind(qIn)
				_ = db.Select(&regs, qIn, argsIn...)
			}

			if len(regs) > 0 {
				totalParticipantAmount := 0.0
				for _, r := range regs {
					itemFee := r.PaymentAmount
					if itemFee <= 0 {
						if r.FeeMode == "per_category" {
							if r.CatFee > 0 {
								itemFee = r.CatFee
							} else if r.EntryFee > 0 {
								itemFee = r.EntryFee
							} else if r.FeeIndividual > 0 {
								itemFee = r.FeeIndividual
							}
						} else {
							switch r.TypeCode {
							case "team":
								if r.FeeTeam > 0 {
									itemFee = r.FeeTeam
								}
							case "mixed_team":
								if r.FeeMixedTeam > 0 {
									itemFee = r.FeeMixedTeam
								}
							default:
								if r.FeeIndividual > 0 {
									itemFee = r.FeeIndividual
								}
							}
							if itemFee <= 0 && r.CatFee > 0 {
								itemFee = r.CatFee
							}
							if itemFee <= 0 && r.EntryFee > 0 {
								itemFee = r.EntryFee
							}
						}
						if itemFee > 0 {
							_, _ = db.Exec("UPDATE tournament_participants SET payment_amount = ? WHERE uuid = ?", itemFee, r.UUID)
						}
					}
					totalParticipantAmount += itemFee
				}
				if totalParticipantAmount <= 0 && req.Amount > 0 {
					totalParticipantAmount = req.Amount
				}
				amount = int(totalParticipantAmount)
				registrationID = &regs[0].UUID
				eventID = &regs[0].EventID
			} else {
				// Check if it's a team booking (e.g. from delegation_teams)
				var team struct {
					UUID         string `db:"uuid"`
					TournamentID string `db:"tournament_id"`
				}
				errTeam := db.Get(&team, "SELECT uuid, tournament_id FROM teams WHERE uuid = ? LIMIT 1", allRegIDs[0])
				if errTeam == nil {
					var fee float64
					_ = db.Get(&fee, "SELECT entry_fee FROM tournaments WHERE uuid = ?", team.TournamentID)
					amount = int(fee)
					if amount <= 0 {
						amount = 75000
					}
					if len(allRegIDs) > 1 {
						amount = amount * len(allRegIDs)
					}
					registrationID = &team.UUID
					eventID = &team.TournamentID
				} else {
					c.JSON(http.StatusNotFound, gin.H{"error": "Registrasi tidak ditemukan", "reg_ids": allRegIDs})
					return
				}
			}
		}

		if amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Jumlah pembayaran tidak valid atau bernilai nol"})
			return
		}

		transactionID := uuid.New().String()
		merchantRef := fmt.Sprintf("PAY-MANUAL-%s", strings.ToUpper(uuid.New().String()[:8]))

		transaction := models.PaymentTransaction{
			UUID:               transactionID,
			Reference:          merchantRef,
			UserID:             userID.(string),
			EventID:            eventID,
			RegistrationID:     registrationID,
			SubscriptionPlanID: req.PlanID,
			Amount:             float64(amount),
			FeeAmount:          0,
			TotalAmount:        float64(amount),
			PaymentMethod:      utils.StringPtr("manual"),
			Months:             req.Months,
			Status:             "pending",                                         // Will change to awaiting_verification after proof upload
			ExpiredAt:          utils.TimePtr(time.Now().Add(7 * 24 * time.Hour)), // 7 days for manual payment
		}

		if transaction.Months <= 0 {
			transaction.Months = 1
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database: " + err.Error()})
			return
		}
		defer tx.Rollback()

		query := `
			INSERT INTO payment_transactions (
				uuid, reference, user_id, tournament_id, registration_id, subscription_plan_id,
				amount, fee_amount, total_amount, payment_method, months, status, expired_at
			) VALUES (
				:uuid, :reference, :user_id, :tournament_id, :registration_id, :subscription_plan_id,
				:amount, :fee_amount, :total_amount, :payment_method, :months, :status, :expired_at
			)
		`
		_, err = tx.NamedExec(query, transaction)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi: " + err.Error()})
			return
		}

		// Update participant registrations with payment_id
		if len(allRegIDs) > 0 {
			qUp, argsUp, errIn := sqlx.In("UPDATE tournament_participants SET payment_id = ?, payment_status = 'pending', updated_at = NOW() WHERE uuid IN (?)", transactionID, allRegIDs)
			if errIn == nil {
				qUp = tx.Rebind(qUp)
				_, _ = tx.Exec(qUp, argsUp...)
			}
		} else if req.RegistrationID != nil && *req.RegistrationID != "" {
			_, _ = tx.Exec("UPDATE tournament_participants SET payment_id = ?, payment_status = 'pending', updated_at = NOW() WHERE uuid = ?", transactionID, *req.RegistrationID)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan transaksi: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, transaction)
	}
}

// UploadPaymentProof allows users to upload proof of manual payment
func UploadPaymentProof(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		reference := c.Param("reference")

		var req models.UploadPaymentProofRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get transaction
		var transaction struct {
			UUID           string  `db:"uuid"`
			UserID         string  `db:"user_id"`
			RegistrationID *string `db:"registration_id"`
			PaymentMethod  *string `db:"payment_method"`
			Status         string  `db:"status"`
		}
		err := db.Get(&transaction, "SELECT uuid, user_id, registration_id, payment_method, status FROM payment_transactions WHERE reference = ?", reference)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
			return
		}

		// Verify ownership
		if transaction.UserID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke transaksi ini"})
			return
		}

		// Verify payment method is manual
		method := ""
		if transaction.PaymentMethod != nil {
			method = strings.ToLower(*transaction.PaymentMethod)
		}
		if method != "" && !strings.Contains(method, "manual") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Upload bukti hanya untuk pembayaran manual"})
			return
		}

		// Verify status is pending or rejected
		if transaction.Status != "pending" && transaction.Status != "rejected" && transaction.Status != "awaiting_verification" && transaction.Status != "cancelled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Transaksi tidak dalam status yang dapat diupload bukti"})
			return
		}

		// Update transaction with proof URL and change status to awaiting_verification
		now := time.Now()
		_, err = db.Exec(`
			UPDATE payment_transactions 
			SET proof_url = ?, proof_uploaded_at = ?, sender_name = ?, status = 'awaiting_verification', rejection_reason = NULL, updated_at = ?
			WHERE uuid = ?
		`, req.ProofURL, now, req.SenderName, now, transaction.UUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan bukti pembayaran: " + err.Error()})
			return
		}

		regID := ""
		if transaction.RegistrationID != nil {
			regID = *transaction.RegistrationID
		}
		_, _ = db.Exec(`
			UPDATE tournament_participants 
			SET payment_id = ?, payment_status = 'pending', updated_at = ?
			WHERE payment_id = ? OR (uuid = ? AND ? != '')
		`, transaction.UUID, now, transaction.UUID, regID, regID)

		c.JSON(http.StatusOK, gin.H{
			"message":     "Bukti pembayaran berhasil diupload",
			"status":      "awaiting_verification",
			"proof_url":   req.ProofURL,
			"uploaded_at": now,
		})
	}
}

// VerifyManualPayment allows organizers to approve or reject manual payments
func VerifyManualPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		reference := c.Param("reference")

		var req models.VerifyManualPaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate action
		if req.Action != "approve" && req.Action != "reject" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Action harus 'approve' atau 'reject'"})
			return
		}

		// Validate rejection reason if rejecting
		if req.Action == "reject" && (req.RejectionReason == nil || *req.RejectionReason == "") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Alasan penolakan wajib diisi"})
			return
		}

		// Get transaction with event details
		var transaction struct {
			UUID               string  `db:"uuid"`
			UserID             string  `db:"user_id"`
			EventID            *string `db:"event_id"`
			RegistrationID     *string `db:"registration_id"`
			SubscriptionPlanID *int    `db:"subscription_plan_id"`
			PaymentMethod      *string `db:"payment_method"`
			Status             string  `db:"status"`
			Months             int     `db:"months"`
			OrganizerID        *string `db:"organizer_id"`
		}
		query := `
			SELECT 
				pt.uuid, pt.user_id, pt.tournament_id as event_id, pt.registration_id, pt.subscription_plan_id,
				pt.payment_method, pt.status, pt.months, e.organizer_id
			FROM payment_transactions pt
			LEFT JOIN tournaments e ON pt.tournament_id = e.uuid
			WHERE pt.reference = ?
		`
		err := db.Get(&transaction, query, reference)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
			return
		}

		// Verify payment method is manual
		if transaction.PaymentMethod == nil || (*transaction.PaymentMethod != "manual" && *transaction.PaymentMethod != "manual_transfer") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Verifikasi hanya untuk pembayaran manual"})
			return
		}

		// Verify status is awaiting_verification
		if transaction.Status != "awaiting_verification" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Transaksi tidak dalam status menunggu verifikasi"})
			return
		}

		// Verify user is the organizer of the event
		if transaction.OrganizerID == nil || *transaction.OrganizerID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses untuk memverifikasi pembayaran ini"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
			return
		}

		now := time.Now()

		if req.Action == "approve" {
			// Update transaction status to paid
			_, err = tx.Exec(`
				UPDATE payment_transactions 
				SET status = 'paid', paid_at = ?, verified_by = ?, verified_at = ?, updated_at = ?
				WHERE uuid = ?
			`, now, userID.(string), now, now, transaction.UUID)

			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status transaksi"})
				return
			}

			// Update linked participants
			regID := ""
			if transaction.RegistrationID != nil {
				regID = *transaction.RegistrationID
			}

			// Update all tournament participants linked by payment_id OR registration_id, and assign qr_raw if missing
			_, err = tx.Exec(`
				UPDATE tournament_participants 
				SET payment_status = 'paid',
				    qr_raw = COALESCE(NULLIF(qr_raw, ''), UUID()),
				    updated_at = ?
				WHERE payment_id = ? OR (uuid = ? AND ? != '')
			`, now, transaction.UUID, regID, regID)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui registrasi peserta"})
				return
			}

			// Also update teams if registration was for a team
			if regID != "" {
				_, _ = tx.Exec("UPDATE teams SET status = 'active' WHERE uuid = ?", regID)
			}

			// Update event status if platform fee is paid
			if transaction.RegistrationID == nil && transaction.EventID != nil && transaction.SubscriptionPlanID == nil {
				_, err = tx.Exec("UPDATE tournaments SET status = 'published' WHERE uuid = ?", *transaction.EventID)
				if err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status event"})
					return
				}
			}

			// Update subscription if applicable
			if transaction.SubscriptionPlanID != nil {
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
					if plan.TargetType == "organizer" {
						table = "organizers"
					}

					var currentExpires *time.Time
					_ = db.Get(&currentExpires, "SELECT subscription_expires_at FROM "+table+" WHERE user_id = ?", transaction.UserID)

					effectiveMonths := transaction.Months
					if effectiveMonths <= 0 {
						effectiveMonths = months
					}

					baseTime := now
					if currentExpires != nil && currentExpires.After(now) {
						baseTime = *currentExpires
					}

					newExpiry := baseTime.AddDate(0, effectiveMonths, 0)

					_, err = tx.Exec("UPDATE "+table+" SET subscription_plan_id = ?, subscription_status = 'active', subscription_expires_at = ? WHERE user_id = ?",
						*transaction.SubscriptionPlanID, newExpiry, transaction.UserID)
					if err != nil {
						tx.Rollback()
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui langganan"})
						return
					}
				}
			}

			// Update orders if applicable
			_, err = tx.Exec("UPDATE orders SET payment_status = 'paid' WHERE payment_id = ?", transaction.UUID)
			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status pesanan"})
				return
			}

		} else {
			// Reject payment
			_, err = tx.Exec(`
				UPDATE payment_transactions 
				SET status = 'rejected', rejection_reason = ?, verified_by = ?, verified_at = ?, updated_at = ?
				WHERE uuid = ?
			`, *req.RejectionReason, userID.(string), now, now, transaction.UUID)

			if err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menolak pembayaran"})
				return
			}

			// Update all linked participants to rejected
			regID := ""
			if transaction.RegistrationID != nil {
				regID = *transaction.RegistrationID
			}
			_, _ = tx.Exec(`
				UPDATE tournament_participants 
				SET payment_status = 'rejected', updated_at = ? 
				WHERE payment_id = ? OR (uuid = ? AND ? != '')
			`, now, transaction.UUID, regID, regID)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan"})
			return
		}

		if req.Action == "approve" {
			// Asynchronously send approval email to the athlete/payer
			go func(txUUID string) {
				var emailData struct {
					Email     string  `db:"email"`
					FullName  string  `db:"full_name"`
					EventName string  `db:"event_name"`
					Amount    float64 `db:"total_amount"`
				}
				errEmail := db.Get(&emailData, `
					SELECT 
						COALESCE(NULLIF(a.email, ''), NULLIF(u.email, ''), o.email, '') as email,
						COALESCE(NULLIF(a.full_name, ''), NULLIF(u.full_name, ''), o.name, 'Peserta') as full_name,
						COALESCE(e.name, 'Turnamen Archeris') as event_name,
						pt.total_amount
					FROM payment_transactions pt
					LEFT JOIN tournaments e ON pt.tournament_id = e.uuid
					LEFT JOIN tournament_participants ep ON pt.registration_id = ep.uuid
					LEFT JOIN archers a ON ep.archer_id = a.uuid
					LEFT JOIN archers u ON pt.user_id = u.uuid
					LEFT JOIN organizers o ON pt.user_id = o.uuid
					WHERE pt.uuid = ? LIMIT 1
				`, txUUID)
				if errEmail == nil && emailData.Email != "" {
					var catNames []string
					_ = db.Select(&catNames, `
						SELECT DISTINCT COALESCE(tc.category_name_custom, rag.name, 'General')
						FROM tournament_participants tp
						LEFT JOIN tournament_categories tc ON tp.category_id = tc.uuid
						LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
						WHERE tp.payment_id = ?
					`, txUUID)
					_ = utils.SendPaymentApprovedEmail(emailData.Email, emailData.FullName, emailData.EventName, emailData.Amount, catNames)
				}
			}(transaction.UUID)
		}

		if req.Action == "reject" && req.RejectionReason != nil {
			// Asynchronously send email notification to the athlete/user
			go func(txRef, rejReason string) {
				var emailData struct {
					Email     string  `db:"email"`
					FullName  string  `db:"full_name"`
					EventName *string `db:"event_name"`
				}
				errEmail := db.Get(&emailData, `
					SELECT 
						COALESCE(a.email, u.email, '') as email,
						COALESCE(a.full_name, u.full_name, 'Peserta') as full_name,
						e.name as event_name
					FROM payment_transactions pt
					LEFT JOIN tournaments e ON pt.tournament_id = e.uuid
					LEFT JOIN tournament_participants ep ON pt.registration_id = ep.uuid
					LEFT JOIN archers a ON ep.archer_id = a.uuid
					LEFT JOIN archers u ON pt.user_id = u.uuid
					WHERE pt.reference = ? LIMIT 1
				`, txRef)
				if errEmail == nil && emailData.Email != "" {
					eventName := "Turnamen"
					if emailData.EventName != nil && *emailData.EventName != "" {
						eventName = *emailData.EventName
					}
					subject := fmt.Sprintf("[Archeris] Tindak Lanjut Bukti Pembayaran: %s", eventName)
					contentHTML := fmt.Sprintf(`
						<p>Halo <strong>%s</strong>,</p>
						<p>Bukti pembayaran yang Anda unggah untuk transaksi turnamen <strong>%s</strong> (Kode Referensi: <code>%s</code>) <strong>belum dapat diverifikasi</strong> oleh pihak penyelenggara.</p>
						<div style="background-color:#fef2f2;border-left:4px solid #ef4444;padding:12px 16px;margin:16px 0;border-radius:4px;">
							<strong style="color:#b91c1c;">Alasan Penolakan:</strong>
							<p style="margin:4px 0 0;color:#7f1d1d;">%s</p>
						</div>
						<p>Mohon untuk mengunggah ulang foto atau screenshot bukti transfer yang jelas dan sesuai nominal melalui tautan berikut:</p>
						<p style="margin:24px 0;">
							<a href="%s/dashboard/archer/payments/%s" style="background-color:#0d9488;color:#ffffff;padding:12px 24px;border-radius:8px;text-decoration:none;font-weight:bold;display:inline-block;">Upload Ulang Bukti Pembayaran</a>
						</p>
						<p style="color:#64748b;font-size:13px;">Jika Anda merasa ini adalah kekeliruan, silakan hubungi kontak panitia penyelenggara turnamen.</p>
					`, emailData.FullName, eventName, txRef, rejReason, utils.GetAppURL(), txRef)
					emailBody := utils.BuildCleanCardEmail(subject, contentHTML)
					_ = utils.SendEmail(emailData.Email, subject, emailBody)
				}
			}(reference, *req.RejectionReason)
		}

		message := "Pembayaran berhasil disetujui"
		if req.Action == "reject" {
			message = "Pembayaran ditolak"
		}

		c.JSON(http.StatusOK, gin.H{
			"message":     message,
			"status":      req.Action,
			"verified_at": now,
		})
	}
}

// GetPendingManualPayments returns all manual payments awaiting verification for organizer's tournaments
func GetPendingManualPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		eventID := c.Query("event_id")

		type PendingPayment struct {
			UUID            string     `json:"id" db:"uuid"`
			Reference       string     `json:"reference" db:"reference"`
			Amount          float64    `json:"amount" db:"amount"`
			ProofURL        *string    `json:"proof_url" db:"proof_url"`
			ProofUploadedAt *time.Time `json:"proof_uploaded_at" db:"proof_uploaded_at"`
			Status          string     `json:"status" db:"status"`
			CreatedAt       time.Time  `json:"created_at" db:"created_at"`
			EventName       *string    `json:"event_name" db:"event_name"`
			ArcherName      *string    `json:"archer_name" db:"archer_name"`
			ArcherEmail     *string    `json:"archer_email" db:"archer_email"`
			SenderName      *string    `json:"sender_name" db:"sender_name"`
		}

		var payments []PendingPayment
		query := `
			SELECT 
				pt.uuid, pt.reference, pt.amount, pt.proof_url, pt.proof_uploaded_at,
				pt.status, pt.created_at, pt.sender_name, e.name as event_name,
				a.full_name as archer_name, a.email as archer_email
			FROM payment_transactions pt
			LEFT JOIN tournaments e ON pt.tournament_id = e.uuid
			LEFT JOIN tournament_participants ep ON pt.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE pt.payment_method = 'manual' 
			AND pt.status = 'awaiting_verification'
			AND e.organizer_id = ?
		`

		args := []interface{}{userID.(string)}

		if eventID != "" {
			query += " AND pt.tournament_id = ?"
			args = append(args, eventID)
		}

		query += " ORDER BY pt.proof_uploaded_at DESC"

		err := db.Select(&payments, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pembayaran: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"payments": payments,
			"count":    len(payments),
		})
	}
}

// GetEventManualPayments returns all manual payments/invoices for a tournament with participant breakdowns
func GetEventManualPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		eventID := c.Param("id")
		var actualEventID string
		var organizerID string
		err := db.QueryRow("SELECT uuid, organizer_id FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID).Scan(&actualEventID, &organizerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		userRole, _ := c.Get("user_role")
		if organizerID != userID.(string) && userRole != "root" && userRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya penyelenggara turnamen yang dapat mengakses daftar pembayaran ini"})
			return
		}

		type ParticipantDetail struct {
			UUID               string     `json:"id" db:"uuid"`
			PaymentID          *string    `json:"payment_id" db:"payment_id"`
			PaymentStatus      string     `json:"payment_status" db:"payment_status"`
			PaymentAmount      float64    `json:"payment_amount" db:"payment_amount"`
			LastReregistration *time.Time `json:"last_reregistration_at" db:"last_reregistration_at"`
			ArcherName         string     `json:"archer_name" db:"archer_name"`
			Gender             *string    `json:"gender" db:"gender"`
			ClubName           *string    `json:"club_name" db:"club_name"`
			CategoryName       *string    `json:"category_name" db:"category_name"`
		}

		type ManualPaymentInvoice struct {
			UUID             string              `json:"id" db:"uuid"`
			Reference        string              `json:"reference" db:"reference"`
			PaymentMethod    *string             `json:"payment_method" db:"payment_method"`
			Amount           float64             `json:"amount" db:"amount"`
			FeeAmount        float64             `json:"fee_amount" db:"fee_amount"`
			TotalAmount      float64             `json:"total_amount" db:"total_amount"`
			Status           string              `json:"status" db:"status"`
			ProofURL         *string             `json:"proof_url" db:"proof_url"`
			ProofUploadedAt  *time.Time          `json:"proof_uploaded_at" db:"proof_uploaded_at"`
			SenderName       *string             `json:"sender_name" db:"sender_name"`
			RejectionReason  *string             `json:"rejection_reason" db:"rejection_reason"`
			CreatedAt        time.Time           `json:"created_at" db:"created_at"`
			PaidAt           *time.Time          `json:"paid_at" db:"paid_at"`
			VerifiedAt       *time.Time          `json:"verified_at" db:"verified_at"`
			PayerName        *string             `json:"payer_name" db:"payer_name"`
			PayerEmail       *string             `json:"payer_email" db:"payer_email"`
			ClubName         *string             `json:"club_name" db:"club_name"`
			RegistrationID   *string             `json:"registration_id" db:"registration_id"`
			Participants     []ParticipantDetail `json:"participants"`
			ParticipantCount int                 `json:"participant_count"`
		}

		var invoices []ManualPaymentInvoice
		txQuery := `
			SELECT 
				pt.uuid, pt.reference, pt.payment_method, pt.amount, pt.fee_amount, pt.total_amount,
				pt.status, pt.proof_url, pt.proof_uploaded_at, pt.sender_name, pt.rejection_reason,
				pt.created_at, pt.paid_at, pt.verified_at, pt.registration_id,
				COALESCE(a.full_name, o.name, 'Pendaftar') as payer_name,
				COALESCE(a.email, o.email, '') as payer_email,
				c.name as club_name
			FROM payment_transactions pt
			LEFT JOIN archers a ON pt.user_id = a.uuid
			LEFT JOIN organizers o ON pt.user_id = o.uuid
			LEFT JOIN clubs c ON a.club_id = c.uuid
			WHERE pt.tournament_id = ?
			ORDER BY pt.created_at DESC
		`
		err = db.Select(&invoices, txQuery, actualEventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data tagihan: " + err.Error()})
			return
		}

		var allParticipants []ParticipantDetail
		partQuery := `
			SELECT 
				tp.uuid, tp.payment_id, COALESCE(tp.payment_status, 'unpaid') as payment_status,
				tp.payment_amount, tp.last_reregistration_at,
				a.full_name as archer_name, a.gender,
				COALESCE(c.name, '') as club_name,
				COALESCE(tc.category_name_custom, ag.name, '') as category_name
			FROM tournament_participants tp
			INNER JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs c ON a.club_id = c.uuid
			LEFT JOIN tournament_categories tc ON tp.category_id = tc.uuid
			LEFT JOIN ref_age_groups ag ON tc.category_uuid = ag.uuid
			WHERE tp.tournament_id = ?
		`
		_ = db.Select(&allParticipants, partQuery, actualEventID)

		// Map participants to invoices
		for i := range invoices {
			invoices[i].Participants = []ParticipantDetail{}
			for _, p := range allParticipants {
				if (p.PaymentID != nil && *p.PaymentID == invoices[i].UUID) ||
					(invoices[i].RegistrationID != nil && *invoices[i].RegistrationID == p.UUID) {
					invoices[i].Participants = append(invoices[i].Participants, p)
				}
			}
			invoices[i].ParticipantCount = len(invoices[i].Participants)
		}

		c.JSON(http.StatusOK, gin.H{
			"payments": invoices,
			"count":    len(invoices),
		})
	}
}

// CancelPaymentTransaction allows a user or organizer to cancel a pending/awaiting payment transaction
func CancelPaymentTransaction(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		reference := c.Param("reference")

		var tx struct {
			UUID           string  `db:"uuid"`
			UserID         string  `db:"user_id"`
			Status         string  `db:"status"`
			PaymentMethod  string  `db:"payment_method"`
			ProofURL       *string `db:"proof_url"`
			TournamentID   *string `db:"tournament_id"`
			RegistrationID *string `db:"registration_id"`
			OrganizerID    *string `db:"organizer_id"`
		}

		query := `
			SELECT 
				pt.uuid, pt.user_id, pt.status, COALESCE(pt.payment_method, '') AS payment_method, pt.proof_url, pt.tournament_id, pt.registration_id,
				e.organizer_id
			FROM payment_transactions pt
			LEFT JOIN tournaments e ON pt.tournament_id = e.uuid
			WHERE pt.reference = ? OR pt.uuid = ?
			LIMIT 1
		`
		err := db.Get(&tx, query, reference, reference)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
			return
		}

		userRole, _ := c.Get("user_role")
		isOwner := tx.UserID == userID.(string)
		isOrganizer := tx.OrganizerID != nil && *tx.OrganizerID == userID.(string)
		isAdmin := userRole == "root" || userRole == "admin"

		if !isOwner && !isOrganizer && !isAdmin {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki wewenang untuk membatalkan transaksi ini"})
			return
		}

		if isOwner && !isOrganizer && !isAdmin && (tx.PaymentMethod == "manual" || (tx.ProofURL != nil && *tx.ProofURL != "")) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tagihan transfer manual tidak dapat dibatalkan. Silakan hubungi penyelenggara event jika ingin melakukan perubahan."})
			return
		}

		if tx.Status == "paid" || tx.Status == "settlement" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Transaksi yang sudah lunas tidak dapat dibatalkan"})
			return
		}

		if tx.Status == "cancelled" {
			c.JSON(http.StatusOK, gin.H{"message": "Transaksi sudah dibatalkan sebelumnya", "status": "cancelled"})
			return
		}

		dbTx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
			return
		}
		defer dbTx.Rollback()

		now := time.Now()
		_, err = dbTx.Exec(`
			UPDATE payment_transactions 
			SET status = 'cancelled', updated_at = ? 
			WHERE uuid = ?
		`, now, tx.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status transaksi: " + err.Error()})
			return
		}

		// Reset linked participants payment_status = 'unpaid', payment_id = NULL
		regID := ""
		if tx.RegistrationID != nil {
			regID = *tx.RegistrationID
		}
		_, err = dbTx.Exec(`
			UPDATE tournament_participants 
			SET payment_status = 'cancelled', payment_id = NULL, updated_at = ? 
			WHERE (payment_id = ? OR uuid = ?) AND payment_status NOT IN ('paid', 'lunas')
		`, now, tx.UUID, regID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset status peserta: " + err.Error()})
			return
		}

		if err := dbTx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Transaksi berhasil dibatalkan. Tagihan peserta telah direset.",
			"status":  "cancelled",
		})
	}
}

// CleanupExpiredPayments auto-expires stale pending payment transactions and releases slots.
func CleanupExpiredPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		count, err := PerformPaymentCleanup(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membersihkan transaksi kedaluwarsa", "details": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Pembersihan transaksi kedaluwarsa selesai", "expired_count": count})
	}
}

func PerformPaymentCleanup(db *sqlx.DB) (int, error) {
	tx, err := db.Beginx()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var expiredTx []struct {
		UUID           string  `db:"uuid"`
		RegistrationID *string `db:"registration_id"`
	}
	err = tx.Select(&expiredTx, `
		SELECT uuid, registration_id
		FROM payment_transactions
		WHERE (status = 'pending' OR status = 'rejected') AND expired_at IS NOT NULL AND expired_at < NOW()
	`)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	if len(expiredTx) == 0 {
		return 0, nil
	}

	_, err = tx.Exec(`
		UPDATE payment_transactions
		SET status = 'EXPIRED', updated_at = NOW()
		WHERE (status = 'pending' OR status = 'rejected') AND expired_at IS NOT NULL AND expired_at < NOW()
	`)
	if err != nil {
		return 0, err
	}

	for _, t := range expiredTx {
		regID := ""
		if t.RegistrationID != nil {
			regID = *t.RegistrationID
		}
		_, _ = tx.Exec(`
			UPDATE tournament_participants 
			SET payment_status = 'cancelled', updated_at = NOW() 
			WHERE (payment_id = ? OR (uuid = ? AND ? != '')) 
			  AND payment_status IN ('pending', 'rejected', 'unpaid')
		`, t.UUID, regID, regID)
		if regID != "" {
			_, _ = tx.Exec(`DELETE FROM qualification_target_assignments WHERE participant_id = ?`, regID)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(expiredTx), nil
}

// GetPaymentInstruction returns payment instructions for given channel
func GetPaymentInstruction(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		instructions := []map[string]interface{}{
			{
				"title": "Pembayaran Online (" + code + ")",
				"steps": []string{
					"Buka tautan checkout Mayar yang tersedia pada detail transaksi.",
					"Pilih metode pembayaran (QRIS, Virtual Account Bank, atau E-Wallet).",
					"Selesaikan pembayaran sesuai instruksi di halaman Mayar.",
					"Status transaksi akan otomatis terupdate setelah pembayaran berhasil.",
				},
			},
		}
		c.JSON(http.StatusOK, gin.H{"data": instructions})
	}
}

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

// GetEventPaymentMethods returns all payment methods configured for an event
func GetEventPaymentMethods(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Check if it's a slug or UUID, get event organizer_id and page_settings
		var event struct {
			OrganizerID  string  `db:"organizer_id"`
			PageSettings *string `db:"page_settings"`
		}
		err := db.Get(&event, "SELECT organizer_id, page_settings FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}
		organizerID := event.OrganizerID

		// Fetch all active bank accounts / payment methods for this organizer
		var bankAccounts []EventPaymentMethod
		err = db.Select(&bankAccounts, `
			SELECT uuid, user_id as event_id, 
			       CASE WHEN type = 'custom' AND custom_name IS NOT NULL AND custom_name != '' THEN custom_name ELSE bank_name END as payment_method, 
			       account_name, account_number, COALESCE(instructions, '') as instructions, 1 as is_active, 0 as display_order, created_at, updated_at
			FROM bank_accounts
			WHERE user_id = ?
			ORDER BY is_primary DESC, created_at ASC
		`, organizerID)

		if err != nil || len(bankAccounts) == 0 {
			// Fallback to organizer_payment_methods for backward compatibility
			_ = db.Select(&bankAccounts, `
				SELECT uuid, organization_id as event_id, 
				       CASE WHEN type = 'custom' AND custom_name IS NOT NULL AND custom_name != '' THEN custom_name ELSE bank_name END as payment_method, 
				       account_name, account_number, COALESCE(instructions, '') as instructions, 1 as is_active, 0 as display_order, created_at, updated_at
				FROM organizer_payment_methods
				WHERE organization_id = ?
				ORDER BY is_primary DESC, created_at ASC
			`, organizerID)
		}

		// Filter payment methods based on event page_settings selection if present
		if event.PageSettings != nil && *event.PageSettings != "" {
			var pageSettings struct {
				PaymentMethods []string `json:"payment_methods"`
			}
			if errJson := json.Unmarshal([]byte(*event.PageSettings), &pageSettings); errJson == nil {
				// If payment_methods array is defined and non-empty, we filter the active bank accounts.
				// If empty or not present, we return all active bank accounts for backward compatibility.
				if len(pageSettings.PaymentMethods) > 0 {
					enabledMap := make(map[string]bool)
					for _, uuid := range pageSettings.PaymentMethods {
						enabledMap[uuid] = true
					}
					var enabledMethods []EventPaymentMethod
					for _, ba := range bankAccounts {
						if enabledMap[ba.UUID] {
							enabledMethods = append(enabledMethods, ba)
						}
					}
					if enabledMethods == nil {
						enabledMethods = []EventPaymentMethod{}
					}
					c.JSON(http.StatusOK, gin.H{"data": enabledMethods})
					return
				}
			}
		}

		if bankAccounts == nil {
			bankAccounts = []EventPaymentMethod{}
		}

		c.JSON(http.StatusOK, gin.H{"data": bankAccounts})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat metode pembayaran"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"uuid":    methodID,
			"message": "Metode pembayaran berhasil dibuat",
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui metode pembayaran"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Metode pembayaran berhasil diperbarui"})
	}
}

// DeleteEventPaymentMethod deletes a payment method
func DeleteEventPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		methodID := c.Param("methodId")

		_, err := db.Exec("DELETE FROM event_payment_methods WHERE uuid = ?", methodID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus metode pembayaran"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Metode pembayaran berhasil dihapus"})
	}
}

// SimulatePaymentSuccess simulates a successful payment for testing
func SimulatePaymentSuccess(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reference := c.Param("reference")

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var transaction models.PaymentTransaction
		err = tx.Get(&transaction, "SELECT * FROM payment_transactions WHERE reference = ?", reference)
		if err != nil {
			// Fallback: Check in quota_purchases
			var qPurchase struct {
				UUID        string `db:"uuid"`
				OrganizerID string `db:"organizer_id"`
				PlanID      int    `db:"plan_id"`
				Quantity    int    `db:"quantity"`
				Status      string `db:"payment_status"`
			}
			errQ := tx.Get(&qPurchase, "SELECT uuid, organizer_id, plan_id, quantity, payment_status FROM quota_purchases WHERE payment_reference = ? OR uuid = ?", reference, reference)
			if errQ == nil {
				if qPurchase.Status == "paid" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Sudah dibayar"})
					return
				}
				_, _ = tx.Exec("UPDATE quota_purchases SET payment_status = 'paid' WHERE uuid = ?", qPurchase.UUID)
				var plan struct {
					QuotaType string `db:"quota_type"`
				}
				_ = tx.Get(&plan, "SELECT COALESCE(quota_type, 'standard') as quota_type FROM subscription_plans WHERE id = ?", qPurchase.PlanID)
				if plan.QuotaType == "standard" {
					_, _ = tx.Exec("UPDATE organizers SET quota_standard = quota_standard + ? WHERE uuid = ?", qPurchase.Quantity, qPurchase.OrganizerID)
				} else if plan.QuotaType == "elite" {
					_, _ = tx.Exec("UPDATE organizers SET quota_elite = quota_elite + ? WHERE uuid = ?", qPurchase.Quantity, qPurchase.OrganizerID)
				}
				if errCommit := tx.Commit(); errCommit != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"message":         "Simulasi pembayaran kuota berhasil",
					"reference":       reference,
					"is_subscription": false,
				})
				return
			}

			c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
			return
		}

		if transaction.Status == "paid" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Sudah dibayar"})
			return
		}

		now := time.Now()
		_, err = tx.Exec("UPDATE payment_transactions SET status = 'paid', paid_at = ? WHERE reference = ?", now, reference)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui transaksi"})
			return
		}

		if transaction.RegistrationID != nil {
			_, err = tx.Exec(
				"UPDATE tournament_participants SET payment_status = 'paid' WHERE uuid = ? OR payment_id = ?",
				*transaction.RegistrationID, transaction.UUID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status peserta"})
				return
			}
		}

		if transaction.SubscriptionPlanID != nil {
			var plan struct {
				Type       string `db:"type"`
				TargetType string `db:"target_type"`
			}
			errPlan := db.Get(&plan, "SELECT type, target_type FROM subscription_plans WHERE id = ?", *transaction.SubscriptionPlanID)
			if errPlan == nil {
				months := 1
				if plan.Type == "yearly" {
					months = 12
				}

				table := "clubs"
				if plan.TargetType == "organizer" {
					table = "organizers"
				}

				// Fetch current expiry date
				var currentExpires *time.Time
				_ = db.Get(&currentExpires, "SELECT subscription_expires_at FROM "+table+" WHERE user_id = ? OR uuid = ? OR id = ? LIMIT 1", transaction.UserID, transaction.UserID, transaction.UserID)

				effectiveMonths := transaction.Months
				if effectiveMonths <= 0 {
					effectiveMonths = months
				}

				now := time.Now()
				baseTime := now
				if currentExpires != nil && currentExpires.After(now) {
					baseTime = *currentExpires
				}

				newExpiry := baseTime.AddDate(0, effectiveMonths, 0)

				_, _ = tx.Exec("UPDATE "+table+" SET subscription_plan_id = ?, subscription_status = 'active', subscription_expires_at = ? WHERE user_id = ? OR uuid = ? OR id = ?",
					*transaction.SubscriptionPlanID, newExpiry, transaction.UserID, transaction.UserID, transaction.UserID)
			}
		}

		// Update orders table if matching payment exists
		_, err = tx.Exec("UPDATE orders SET payment_status = 'paid' WHERE payment_id = ?", transaction.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status pesanan"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		isSubscription := transaction.SubscriptionPlanID != nil
		c.JSON(http.StatusOK, gin.H{
			"message":         "Simulasi pembayaran berhasil",
			"reference":       reference,
			"is_subscription": isSubscription,
		})
	}
}

// GetMyPayments returns the authenticated user's payment history
func GetMyPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		limit := c.DefaultQuery("limit", "10")
		offset := c.DefaultQuery("offset", "0")

		limitInt, _ := strconv.Atoi(limit)
		offsetInt, _ := strconv.Atoi(offset)

		// Build query that matches direct user_id, archer id/uuid, and archer email
		query := `
			SELECT 
				pt.*,
				COALESCE(e.name, (SELECT name FROM tournaments WHERE uuid = ep.tournament_id LIMIT 1), (SELECT name FROM tournaments WHERE uuid = pt.tournament_id LIMIT 1)) as event_name,
				COALESCE(CONCAT(rbt.name, ' ', rag.name, ' ', rgd.name), ec.category_name_custom, '') as category_name,
				COALESCE((SELECT COUNT(*) FROM tournament_participants tp WHERE (tp.payment_id = pt.uuid AND tp.payment_id != '') OR (tp.uuid = pt.registration_id AND tp.uuid != '')), 1) as participant_count,
				sp.name as plan_name,
				CASE 
					WHEN pt.subscription_plan_id IS NOT NULL THEN 'Langganan Organisasi / Klub'
					WHEN pt.registration_id IS NOT NULL OR pt.tournament_id IS NOT NULL THEN 'Registrasi Turnamen Panahan'
					ELSE 'Transaksi Pembayaran'
				END as purpose
			FROM payment_transactions pt
			LEFT JOIN tournaments e ON pt.tournament_id = e.uuid
			LEFT JOIN tournament_participants ep ON pt.registration_id = ep.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN subscription_plans sp ON pt.subscription_plan_id = sp.id
			WHERE (
				pt.user_id = ?
				OR pt.user_id IN (SELECT uuid FROM archers WHERE id = ? OR email = (SELECT email FROM archers WHERE id = ? OR uuid = ? LIMIT 1))
				OR pt.user_id IN (SELECT id FROM archers WHERE uuid = ? OR email = (SELECT email FROM archers WHERE id = ? OR uuid = ? LIMIT 1))
			)
			ORDER BY pt.created_at DESC
			LIMIT ? OFFSET ?
		`

		type PaymentWithExtra struct {
			models.PaymentTransaction
			EventName        *string `json:"event_name" db:"event_name"`
			CategoryName     *string `json:"category_name" db:"category_name"`
			ParticipantCount int     `json:"participant_count" db:"participant_count"`
			PlanName         *string `json:"plan_name" db:"plan_name"`
			Purpose          *string `json:"purpose" db:"purpose"`
		}

		uid := userID.(string)
		payments := []PaymentWithExtra{}
		err := db.Select(&payments, query, uid, uid, uid, uid, uid, uid, uid, limitInt, offsetInt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pembayaran", "details": err.Error()})
			return
		}

		// Count total for pagination
		var total int
		countQuery := `
			SELECT COUNT(*) FROM payment_transactions pt
			WHERE (
				pt.user_id = ?
				OR pt.user_id IN (SELECT uuid FROM archers WHERE id = ? OR email = (SELECT email FROM archers WHERE id = ? OR uuid = ? LIMIT 1))
				OR pt.user_id IN (SELECT id FROM archers WHERE uuid = ? OR email = (SELECT email FROM archers WHERE id = ? OR uuid = ? LIMIT 1))
			)
		`
		db.Get(&total, countQuery, uid, uid, uid, uid, uid, uid, uid)

		c.JSON(http.StatusOK, gin.H{
			"payments": payments,
			"total":    total,
		})
	}
}

