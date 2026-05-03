package mobile

import (
	"archeryhub-api/handler"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type SellerDashboard struct {
	Stats struct {
		TotalRevenue   float64 `json:"total_revenue"`
		TotalOrders    int     `json:"total_orders"`
		ProductsSold   int     `json:"products_sold"`
		MonthlyRevenue float64 `json:"monthly_revenue"`
		PendingBalance float64 `json:"pending_balance"`
	} `json:"stats"`
	RecentOrders []RecentOrder `json:"recent_orders"`
}

type RecentOrder struct {
	OrderID     string    `json:"order_id" db:"uuid"`
	Customer    string    `json:"customer_name" db:"buyer_name"`
	Amount      float64   `json:"amount" db:"total_amount"`
	Status      string    `json:"status" db:"status"`
	Date        time.Time `json:"date" db:"created_at"`
}

// @Summary Get Seller Dashboard Statistics
// @Description Get dashboard statistics and recent orders for seller
// @Tags Mobile - Seller
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} SellerDashboard
// @Router /mobile/seller/dashboard [get]
func MobileGetSellerDashboard(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var dashboard SellerDashboard

		// 1. Stats
		_ = db.Get(&dashboard.Stats.TotalRevenue, "SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE seller_id = ? AND status != 'cancelled'", userID)
		_ = db.Get(&dashboard.Stats.TotalOrders, "SELECT COUNT(*) FROM orders WHERE seller_id = ?", userID)
		
		_ = db.Get(&dashboard.Stats.ProductsSold, `
			SELECT COALESCE(SUM(quantity), 0) 
			FROM order_items 
			WHERE order_id IN (SELECT uuid FROM orders WHERE seller_id = ?)
		`, userID)

		firstOfMonth := time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02")
		_ = db.Get(&dashboard.Stats.MonthlyRevenue, "SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE seller_id = ? AND status != 'cancelled' AND created_at >= ?", userID, firstOfMonth)

		// Pending balance
		_ = db.Get(&dashboard.Stats.PendingBalance, "SELECT COALESCE(SUM(total_amount), 0) FROM orders WHERE seller_id = ? AND status IN ('paid', 'shipped')", userID)

		// 2. Recent Orders
		_ = db.Select(&dashboard.RecentOrders, `
			SELECT o.uuid, a.full_name as buyer_name, o.total_amount, o.status, o.created_at
			FROM orders o
			LEFT JOIN archers a ON o.buyer_id = a.uuid
			WHERE o.seller_id = ?
			ORDER BY o.created_at DESC
			LIMIT 5
		`, userID)

		c.JSON(http.StatusOK, dashboard)
	}
}

// @Summary Get My Products (Seller)
// @Description Get list of products owned by the seller
// @Tags Mobile - Seller
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Router /mobile/seller/products [get]
func MobileGetSellerProducts(db *sqlx.DB) gin.HandlerFunc {
	return handler.GetMyProducts(db)
}

// @Summary Create Product (Seller)
// @Description Create a new product in the marketplace
// @Tags Mobile - Seller
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param product body interface{} true "Product object"
// @Success 201 {object} map[string]interface{}
// @Router /mobile/seller/products [post]
func MobileCreateProduct(db *sqlx.DB) gin.HandlerFunc {
	return handler.CreateProduct(db)
}

// @Summary Update Product (Seller)
// @Description Update an existing product
// @Tags Mobile - Seller
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Product UUID"
// @Param product body interface{} true "Product object"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/seller/products/{id} [put]
func MobileUpdateProduct(db *sqlx.DB) gin.HandlerFunc {
	return handler.UpdateProduct(db)
}

// @Summary Delete Product (Seller)
// @Description Delete a product
// @Tags Mobile - Seller
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Product UUID"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/seller/products/{id} [delete]
func MobileDeleteProduct(db *sqlx.DB) gin.HandlerFunc {
	return handler.DeleteProduct(db)
}
