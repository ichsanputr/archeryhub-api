package handler

import (
	"archeryhub-api/models"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetSellerOrders returns orders for the current seller
func GetSellerOrders(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "seller" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only sellers can view their orders"})
			return
		}

		var orders []models.Order
		err := db.Select(&orders, `
			SELECT o.* FROM orders o
			WHERE o.seller_id = ?
			ORDER BY o.created_at DESC
		`, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders: " + err.Error()})
			return
		}

		if orders == nil {
			orders = []models.Order{}
		}

		c.JSON(http.StatusOK, gin.H{"data": orders})
	}
}

// GetSellerStats returns aggregated stats for the seller's dashboard
func GetSellerStats(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "seller" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only sellers can view stats"})
			return
		}

		var stats struct {
			TotalRevenue float64 `db:"total_revenue" json:"total_revenue"`
			TotalOrders  int     `db:"total_orders" json:"total_orders"`
			ProductsSold int     `db:"products_sold" json:"products_sold"`
			Rating       float64 `db:"rating" json:"rating"`
		}

		// Basic stats aggregation
		err := db.Get(&stats, `
			SELECT 
				COALESCE(SUM(total_amount), 0) as total_revenue,
				COUNT(*) as total_orders,
				(SELECT COALESCE(SUM(quantity), 0) FROM order_items WHERE order_id IN (SELECT uuid FROM orders WHERE seller_id = ?)) as products_sold,
				(SELECT COALESCE(rating, 0) FROM sellers WHERE uuid = ?) as rating
			FROM orders 
			WHERE seller_id = ? AND status != 'cancelled'
		`, userID, userID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch stats: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": stats})
	}
}

// UpdateOrderStatus updates the status of an order (seller only)
func UpdateOrderStatus(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("id")
		userID, _ := c.Get("user_id")

		var req struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify ownership
		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM orders WHERE uuid = ? AND seller_id = ?", orderID, userID)
		if err != nil || count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Unauthorized to update this order"})
			return
		}

		_, err = db.Exec("UPDATE orders SET status = ?, updated_at = NOW() WHERE uuid = ?", req.Status, orderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Order status updated successfully"})
	}
}
