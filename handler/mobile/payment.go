package mobile

import (
	"archeryhub-api/models"
	"fmt"
	"net/http"

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
			CheckoutURL:     transaction.CheckoutURL,
			Status:          transaction.Status,
		})
	}
}
