package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"Archeris-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetParticipantPayments returns all payment transactions for a participant
func GetParticipantPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		participantID := c.Param("participantId")

		var payments []map[string]interface{}
		rows, err := db.Queryx("SELECT uuid, reference, amount, total_amount, payment_method, status, created_at, instructions as note FROM payment_transactions WHERE registration_id = ? ORDER BY created_at DESC", participantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments: " + err.Error()})
			return
		}
		defer rows.Close()

		for rows.Next() {
			row := make(map[string]interface{})
			err := rows.MapScan(row)
			if err == nil {
				// Format byte arrays to strings if needed
				for k, v := range row {
					if b, ok := v.([]byte); ok {
						row[k] = string(b)
					}
				}
				payments = append(payments, row)
			}
		}

		if payments == nil {
			payments = []map[string]interface{}{}
		}

		c.JSON(http.StatusOK, gin.H{"data": payments})
	}
}

// AddParticipantPayment adds a manual payment or refund
func AddParticipantPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")
		userID, _ := c.Get("user_id")

		var req struct {
			Amount        float64 `json:"amount" binding:"required"`
			Type          string  `json:"type" binding:"required"` // "payment" or "refund"
			Note          string  `json:"note"`
			PaymentMethod string  `json:"payment_method"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		amount := req.Amount
		if req.Type == "refund" {
			amount = -amount
		}

		paymentUUID := uuid.New().String()
		reference := fmt.Sprintf("MANUAL-%s-%d", strings.ToUpper(participantID[:8]), time.Now().Unix())

		_, err := db.Exec(`
			INSERT INTO payment_transactions (
				uuid, reference, user_id, event_id, registration_id, 
				amount, total_amount, payment_method, status, instructions, 
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'paid', ?, NOW(), NOW())
		`, paymentUUID, reference, userID, eventID, participantID, amount, amount, req.PaymentMethod, req.Note)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan pembayaran: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Pembayaran berhasil ditambahkan", "id": paymentUUID})
	}
}

// ApproveParticipantPayment updates payment_status to paid and sends email
func ApproveParticipantPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		participantID := c.Param("participantId")

		// First get the participant details for the email
		var pInfo struct {
			ArcherID   *string `db:"archer_id"`
			EventID    string  `db:"event_id"`
			Amount     float64 `db:"payment_amount"`
			ArcherName string  `db:"archer_name"`
			ArcherMail string  `db:"email"`
			EventName  string  `db:"event_name"`
		}

		err := db.Get(&pInfo, `
			SELECT 
				ep.archer_id, ep.event_id, ep.payment_amount,
				a.full_name as archer_name, a.email as email,
				e.name as event_name
			FROM event_participants ep
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN events e ON ep.event_id = e.uuid
			WHERE ep.uuid = ?
		`, participantID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Peserta tidak ditemukan"})
			return
		}

		// Update status
		_, err = db.Exec("UPDATE event_participants SET payment_status = 'paid', updated_at = NOW() WHERE uuid = ?", participantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate status: " + err.Error()})
			return
		}

		// Update related payment transaction if exists and pending
		db.Exec("UPDATE payment_transactions SET status = 'paid', updated_at = NOW() WHERE registration_id = ? AND status IN ('pending', 'awaiting_verification')", participantID)

		// Get categories
		var categories []string
		db.Select(&categories, `
			SELECT ec.name 
			FROM event_participant_categories epc
			JOIN event_categories ec ON epc.event_category_id = ec.uuid
			WHERE epc.participant_id = ?
		`, participantID)

		// Send email async
		if pInfo.ArcherMail != "" {
			go func() {
				_ = utils.SendPaymentApprovedEmail(pInfo.ArcherMail, pInfo.ArcherName, pInfo.EventName, pInfo.Amount, categories)
			}()
		}

		c.JSON(http.StatusOK, gin.H{"message": "Pembayaran berhasil disetujui"})
	}
}
