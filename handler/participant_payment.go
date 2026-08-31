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
		eventID := c.Param("id")
		participantID := c.Param("participantId")

		// Resolve event slug to UUID
		var actualEventID string
		_ = db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if actualEventID == "" {
			actualEventID = eventID
		}

		// Resolve participant ID / Archer ID
		var pInfo struct {
			UUID     string  `db:"uuid"`
			ArcherID *string `db:"archer_id"`
		}
		_ = db.Get(&pInfo, `
			SELECT tp.uuid, tp.archer_id FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			WHERE (tp.event_id = ? OR tp.event_id = ?) AND (
				tp.uuid = ? OR
				tp.archer_id = ? OR
				a.username = ? OR
				a.id = ? OR
				LOWER(REPLACE(a.full_name, ' ', '-')) = LOWER(?)
			)
			LIMIT 1
		`, eventID, actualEventID, participantID, participantID, participantID, participantID, participantID)

		regIDs := []string{participantID}
		if pInfo.UUID != "" {
			regIDs = append(regIDs, pInfo.UUID)
		}
		if pInfo.ArcherID != nil && *pInfo.ArcherID != "" {
			regIDs = append(regIDs, *pInfo.ArcherID)
		}

		query, args, err := sqlx.In(`
			SELECT uuid, reference, amount, total_amount, payment_method, status, created_at, instructions as note 
			FROM payment_transactions 
			WHERE registration_id IN (?) OR (event_id = ? AND user_id = ?)
			ORDER BY created_at DESC
		`, regIDs, actualEventID, pInfo.ArcherID)

		if err != nil {
			query = "SELECT uuid, reference, amount, total_amount, payment_method, status, created_at, instructions as note FROM payment_transactions WHERE registration_id = ? ORDER BY created_at DESC"
			args = []interface{}{participantID}
		} else {
			query = db.Rebind(query)
		}

		var payments []map[string]interface{}
		rows, err := db.Queryx(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments: " + err.Error()})
			return
		}
		defer rows.Close()

		for rows.Next() {
			row := make(map[string]interface{})
			err := rows.MapScan(row)
			if err == nil {
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

// AddParticipantPayment adds a manual payment, add-on, or refund with category management
func AddParticipantPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")
		userID, _ := c.Get("user_id")

		var req struct {
			Amount            float64  `json:"amount" binding:"required"`
			Type              string   `json:"type" binding:"required"` // "payment", "add_on", or "refund"
			Note              string   `json:"note"`
			PaymentMethod     string   `json:"payment_method"`
			AddCategoryIDs    []string `json:"add_category_ids"`
			RemoveCategoryIDs []string `json:"remove_category_ids"`
			UpdateCategories  bool     `json:"update_categories"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve event slug to UUID
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Lookup participant info
		var pInfo struct {
			UUID          string   `db:"uuid"`
			ArcherID      *string  `db:"archer_id"`
			PaymentAmount float64  `db:"payment_amount"`
			PaymentStatus string   `db:"payment_status"`
			TargetName    *string  `db:"target_name"`
			BackNumber    *string  `db:"back_number"`
			QRRaw         *string  `db:"qr_raw"`
		}
		err = db.Get(&pInfo, `
			SELECT tp.uuid, tp.archer_id, tp.payment_amount, tp.payment_status, tp.target_name, tp.back_number, tp.qr_raw
			FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			WHERE tp.event_id = ? AND (
				tp.uuid = ? OR
				tp.archer_id = ? OR
				a.username = ? OR
				a.id = ? OR
				LOWER(REPLACE(a.full_name, ' ', '-')) = LOWER(?)
			)
			LIMIT 1
		`, actualEventID, participantID, participantID, participantID, participantID, participantID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Peserta tidak ditemukan pada event ini"})
			return
		}

		actualParticipantID := pInfo.UUID
		txAmount := req.Amount
		refPrefix := "MANUAL"
		if req.Type == "refund" {
			txAmount = -req.Amount
			refPrefix = "REFUND"
		} else if req.Type == "add_on" {
			refPrefix = "ADDON"
		}

		paymentUUID := uuid.New().String()
		refID := actualParticipantID
		if len(refID) > 8 {
			refID = refID[:8]
		}
		reference := fmt.Sprintf("%s-%s-%d", refPrefix, strings.ToUpper(refID), time.Now().Unix())

		if req.PaymentMethod == "" {
			req.PaymentMethod = "Manual Transfer"
		}

		// Begin DB Transaction
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
			return
		}
		defer tx.Rollback()

		// 1. Insert Payment Transaction Record
		_, err = tx.Exec(`
			INSERT INTO payment_transactions (
				uuid, reference, user_id, event_id, registration_id, 
				amount, total_amount, payment_method, status, instructions, 
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'paid', ?, NOW(), NOW())
		`, paymentUUID, reference, userID, actualEventID, actualParticipantID, txAmount, txAmount, req.PaymentMethod, req.Note)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat riwayat transaksi: " + err.Error()})
			return
		}

		// 2. Handle Category Adjustments & Participant Payment Amount
		if pInfo.ArcherID != nil && *pInfo.ArcherID != "" {
			archerUUID := *pInfo.ArcherID

			if req.UpdateCategories {
				// Handle Category Removal for Refund
				if req.Type == "refund" && len(req.RemoveCategoryIDs) > 0 {
					for _, catID := range req.RemoveCategoryIDs {
						if catID != "" {
							_, _ = tx.Exec(`
								DELETE FROM event_participants 
								WHERE event_id = ? AND archer_id = ? AND category_id = ?
							`, actualEventID, archerUUID, catID)
						}
					}
				}

				// Handle Category Addition for Add-on
				if (req.Type == "add_on" || req.Type == "payment") && len(req.AddCategoryIDs) > 0 {
					for _, catID := range req.AddCategoryIDs {
						if catID != "" {
							var exists bool
							_ = tx.Get(&exists, `
								SELECT EXISTS(SELECT 1 FROM event_participants WHERE event_id = ? AND archer_id = ? AND category_id = ?)
							`, actualEventID, archerUUID, catID)

							if !exists {
								newParticipantUUID := uuid.New().String()
								_, err = tx.Exec(`
									INSERT INTO event_participants (
										uuid, event_id, archer_id, category_id, payment_amount, 
										payment_status, target_name, back_number, qr_raw, 
										registration_source, registration_date, created_at, updated_at
									) VALUES (?, ?, ?, ?, ?, 'paid', ?, ?, ?, 'organizer_added', NOW(), NOW(), NOW())
								`, newParticipantUUID, actualEventID, archerUUID, catID, 0, pInfo.TargetName, pInfo.BackNumber, pInfo.QRRaw)

								if err != nil {
									c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan kategori peserta: " + err.Error()})
									return
								}
							}
						}
					}
				}
			}

			// Update payment_amount across remaining participant records for this archer in this event
			if req.Type == "refund" {
				_, _ = tx.Exec(`
					UPDATE event_participants 
					SET payment_amount = GREATEST(0, payment_amount - ?), updated_at = NOW() 
					WHERE event_id = ? AND archer_id = ?
				`, req.Amount, actualEventID, archerUUID)
			} else {
				_, _ = tx.Exec(`
					UPDATE event_participants 
					SET payment_amount = payment_amount + ?, payment_status = 'paid', updated_at = NOW() 
					WHERE event_id = ? AND archer_id = ?
				`, req.Amount, actualEventID, archerUUID)
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan transaksi: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Transaksi berhasil dicatat",
			"id":        paymentUUID,
			"reference": reference,
			"amount":    txAmount,
			"type":      req.Type,
		})
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
