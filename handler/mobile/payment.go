package mobile

import (
	"Archeris-api/models"
	"Archeris-api/utils"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileGetPaymentDetail returns status of a payment transaction
// @Summary Get Payment Detail
// @Description Get status and details of a specific payment by reference
// @Tags Mobile - Archer
// @Produce json
// @Security ApiKeyAuth
// @Param reference path string true "Merchant Reference (ORD-... or PAY-...)"
// @Success 200 {object} MobilePaymentTransactionResponse
// @Router /mobile/archer/payments/{reference} [get]
func MobileGetPaymentDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reference := c.Param("reference")
		userID, _ := c.Get("user_id")

		var transaction models.PaymentTransaction
		err := db.Get(&transaction, `SELECT * FROM payment_transactions WHERE reference = ? AND user_id = ?`, reference, fmt.Sprintf("%v", userID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, MobilePaymentTransactionResponse{
			ID:              transaction.UUID,
			Reference:       transaction.Reference,
			TripayReference: transaction.TripayReference,
			Amount:          transaction.Amount,
			VANumber:        transaction.VANumber,
			PayCode:         transaction.PayCode,
			PaymentMethod:   transaction.PaymentMethod,
			CheckoutURL:     transaction.CheckoutURL,
			QRURL:           transaction.QRURL,
			Instructions:    transaction.Instructions,
			Status:          transaction.Status,
		})
	}
}

// MobileGetPaymentInstructions returns ONLY the instructions for a payment
// @Summary Get Payment Instructions
// @Description Get step-by-step instructions for a specific payment
// @Tags Mobile - Events
// @Produce json
// @Security ApiKeyAuth
// @Param reference path string true "Merchant Reference"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/events/payments/{reference}/instructions [get]
func MobileGetPaymentInstructions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		reference := c.Param("reference")

		var result struct {
			Instructions  *string `db:"instructions"`
			PaymentMethod *string `db:"payment_method"`
		}
		err := db.Get(&result, `SELECT instructions, payment_method FROM payment_transactions WHERE reference = ?`, reference)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaksi tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"reference":      reference,
			"payment_method": result.PaymentMethod,
			"instructions":   result.Instructions,
		})
	}
}

// MobileArcherGetEventPayments returns payment history related to event registrations
// @Summary List Event Payments
// @Description Get a list of all payment transactions made for event registrations by the authenticated archer
// @Tags Mobile - Archer
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Success 200 {object} MobileArcherEventPaymentsResponse
// @Router /mobile/archer/events/payments [get]
func MobileArcherGetEventPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		// Query payments where registration_id is not null (event registrations)
		// Or where event_id is present (could be platform fees, but for archer it's registration)
		query := `
			SELECT 
				pt.uuid, pt.reference, pt.tripay_reference, pt.amount, pt.total_amount,
				pt.payment_method, pt.status, pt.va_number, pt.checkout_url,
				pt.created_at, pt.paid_at, pt.expired_at,
				e.name as event_name,
				e.slug as event_slug,
				e.logo_url as event_logo_url
			FROM payment_transactions pt
			JOIN events e ON pt.event_id = e.uuid
			WHERE pt.user_id = ? AND pt.registration_id IS NOT NULL
			ORDER BY pt.created_at DESC
			LIMIT ? OFFSET ?
		`

		var payments []MobileArcherEventPaymentItem
		err := db.Select(&payments, query, fmt.Sprintf("%v", userID), limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pembayaran", "details": err.Error()})
			return
		}

		for i := range payments {
			if payments[i].EventLogoURL != nil {
				masked := utils.MaskMediaURL(*payments[i].EventLogoURL)
				payments[i].EventLogoURL = &masked
			}
		}

		if payments == nil {
			payments = []MobileArcherEventPaymentItem{}
		}

		// Count total
		var total int
		_ = db.Get(&total, "SELECT COUNT(*) FROM payment_transactions WHERE user_id = ? AND registration_id IS NOT NULL", userID)

		c.JSON(http.StatusOK, gin.H{
			"payments":    payments,
			"total_count": total,
		})
	}
}

// MobileArcherGetEventPaymentsByEvent returns payment history for a specific event
// @Summary Get Event Payments by Slug
// @Description Get a list of payment transactions made for a specific event registration
// @Tags Mobile - Archer
// @Produce json
// @Security ApiKeyAuth
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileArcherEventPaymentsResponse
// @Router /mobile/archer/events/payments/{slug} [get]
func MobileArcherGetEventPaymentsByEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		slug := c.Param("slug")

		// Resolve event UUID first
		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ?", slug, slug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT 
				pt.uuid, pt.reference, pt.tripay_reference, pt.amount, pt.total_amount,
				pt.payment_method, pt.status, pt.va_number, pt.checkout_url,
				pt.created_at, pt.paid_at, pt.expired_at,
				e.name as event_name,
				e.slug as event_slug,
				e.logo_url as event_logo_url
			FROM payment_transactions pt
			JOIN events e ON pt.event_id = e.uuid
			WHERE pt.user_id = ? AND pt.event_id = ? AND pt.registration_id IS NOT NULL
			ORDER BY pt.created_at DESC
		`

		var payments []MobileArcherEventPaymentItem
		err = db.Select(&payments, query, fmt.Sprintf("%v", userID), eventUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pembayaran", "details": err.Error()})
			return
		}

		for i := range payments {
			if payments[i].EventLogoURL != nil {
				masked := utils.MaskMediaURL(*payments[i].EventLogoURL)
				payments[i].EventLogoURL = &masked
			}
		}

		if payments == nil {
			payments = []MobileArcherEventPaymentItem{}
		}

		c.JSON(http.StatusOK, gin.H{
			"payments":    payments,
			"total_count": len(payments),
		})
	}
}

