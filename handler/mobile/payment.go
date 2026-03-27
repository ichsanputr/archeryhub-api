package mobile

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MobileCreateParticipantPayment godoc
// @Summary      Create participant payment
// @Description  Create Tripay payment transaction for archer registration in an event
// @Tags         Archer
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        participantId  path  string  true  "Participant registration UUID"
// @Param        request        body  object{method=string}  true  "Payment method"
// @Success      200            {object}  MobilePaymentTransactionResponse
// @Failure      400            {object}  ErrorResponse
// @Failure      401            {object}  ErrorResponse
// @Failure      404            {object}  ErrorResponse
// @Router       /archer/participants/{participantId}/payment [post]
func MobileCreateParticipantPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		participantID := c.Param("participantId")
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		var req struct {
			Method string `json:"method" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
			SELECT ep.uuid, ep.event_id, ep.archer_id, ep.payment_amount,
			       a.full_name, a.email, a.phone
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			WHERE ep.uuid = ? AND a.uuid = ?
		`, participantID, fmt.Sprintf("%v", userID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registrasi tidak ditemukan"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi pembayaran: " + err.Error()})
			return
		}

		transactionID := uuid.New().String()
		tripayRef := tripayResult["reference"].(string)
		expiredAt := time.Now().Add(24 * time.Hour)
		if exp, ok := tripayResult["expiry_date"].(float64); ok {
			expiredAt = time.Unix(int64(exp), 0)
		}

		var instructionsJSON *string
		if inst, ok := tripayResult["instructions"]; ok {
			instBytes, _ := json.Marshal(inst)
			instStr := string(instBytes)
			instructionsJSON = &instStr
		}

		uid := fmt.Sprintf("%v", userID)
		transaction := models.PaymentTransaction{
			UUID:            transactionID,
			Reference:       merchantRef,
			TripayReference: &tripayRef,
			UserID:          uid,
			EventID:         &reg.EventID,
			RegistrationID:  &reg.UUID,
			Amount:          float64(amount),
			FeeAmount:       0,
			TotalAmount:     float64(amount),
			PaymentMethod:   utils.StringPtr(req.Method),
			VANumber:        utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			QRURL:           utils.InterfaceToStringPtr(tripayResult["qr_url"]),
			CheckoutURL:     utils.InterfaceToStringPtr(tripayResult["checkout_url"]),
			PayCode:         utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			Instructions:    instructionsJSON,
			Months:          1,
			Status:          "pending",
			ExpiredAt:       expiredAt,
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
		if _, err = db.NamedExec(query, transaction); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi: " + err.Error()})
			return
		}

		_, _ = db.Exec("UPDATE event_participants SET payment_id = ?, payment_status = 'pending' WHERE uuid = ?", transactionID, reg.UUID)

		c.JSON(http.StatusOK, transaction)
	}
}
