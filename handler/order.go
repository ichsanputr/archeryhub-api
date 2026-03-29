package handler

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetSellerOrders returns orders for the current seller
func GetSellerOrders(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "seller" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya penjual yang bisa melihat pesanan mereka"})
			return
		}

		limit, offset, page := utils.GetPaginationParams(c)
		status := c.Query("status")
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")

		type DetailedOrder struct {
			models.Order
			BuyerName  string `json:"customer_name" db:"buyer_name"`
			BuyerEmail string `json:"customer_email" db:"buyer_email"`
			TotalItems int    `json:"total_items" db:"total_items"`
		}

		whereClause := "WHERE o.seller_id = ?"
		args := []interface{}{userID}

		if status != "" && status != "all" {
			whereClause += " AND o.status = ?"
			args = append(args, status)
		}
		if startDate != "" {
			whereClause += " AND o.created_at >= ?"
			args = append(args, startDate)
		}
		if endDate != "" {
			whereClause += " AND o.created_at <= ?"
			args = append(args, endDate)
		}

		sortBy := c.DefaultQuery("sort_by", "created_at")
		order := strings.ToUpper(c.DefaultQuery("order", "DESC"))

		// Validate sortBy
		allowedSortFields := map[string]string{
			"total_amount": "o.total_amount",
			"status":       "o.status",
			"created_at":   "o.created_at",
			"buyer_name":   "buyer_name",
		}

		dbSortField, ok := allowedSortFields[sortBy]
		if !ok {
			dbSortField = "o.created_at"
		}

		if order != "ASC" && order != "DESC" {
			order = "DESC"
		}

		// Count total
		var totalCount int
		err := db.Get(&totalCount, "SELECT COUNT(*) FROM orders o "+whereClause, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data pesanan"})
			return
		}

		// Get data
		var orders []DetailedOrder
		query := fmt.Sprintf(`
			SELECT 
				o.*, 
				a.full_name as buyer_name, 
				a.email as buyer_email,
				(SELECT SUM(quantity) FROM order_items WHERE order_id = o.uuid) as total_items
			FROM orders o
			LEFT JOIN archers a ON o.buyer_id = a.uuid
			%s
			ORDER BY %s %s
			LIMIT ? OFFSET ?
		`, whereClause, dbSortField, order)
		queryArgs := append(args, limit, offset)

		err = db.Select(&orders, query, queryArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pesanan: " + err.Error()})
			return
		}

		if orders == nil {
			orders = []DetailedOrder{}
		}

		meta := utils.CalculatePagination(totalCount, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{"data": orders, "meta": meta})
	}
}

// GetSellerStats returns aggregated stats for the seller's dashboard
func GetSellerStats(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "seller" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya penjual yang bisa melihat statistik"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data statistik: " + err.Error()})
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
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak diizinkan untuk mengubah pesanan ini"})
			return
		}

		_, err = db.Exec("UPDATE orders SET status = ?, updated_at = NOW() WHERE uuid = ?", req.Status, orderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status pesanan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Status pesanan berhasil diperbarui"})
	}
}

// ExportSellerOrders exports seller orders as CSV
func ExportSellerOrders(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "seller" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak diizinkan"})
			return
		}

		type DetailedOrder struct {
			models.Order
			BuyerName  string `db:"buyer_name"`
			BuyerEmail string `db:"buyer_email"`
			TotalItems int    `db:"total_items"`
		}

		status := c.Query("status")
		startDate := c.Query("start_date")
		endDate := c.Query("end_date")

		query := `
			SELECT 
				o.*, 
				a.full_name as buyer_name, 
				a.email as buyer_email,
				(SELECT SUM(quantity) FROM order_items WHERE order_id = o.uuid) as total_items
			FROM orders o
			LEFT JOIN archers a ON o.buyer_id = a.uuid
			WHERE o.seller_id = ?
		`
		args := []interface{}{userID}

		if status != "" && status != "all" {
			query += " AND o.status = ?"
			args = append(args, status)
		}
		if startDate != "" {
			query += " AND o.created_at >= ?"
			args = append(args, startDate)
		}
		if endDate != "" {
			query += " AND o.created_at <= ?"
			args = append(args, endDate)
		}
		query += " ORDER BY o.created_at DESC"

		var orders []DetailedOrder
		err := db.Select(&orders, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data untuk ekspor: " + err.Error()})
			return
		}

		// Set headers for CSV download
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", "attachment; filename=orders_report.csv")
		c.Header("Content-Type", "text/csv")

		// Create CSV writer
		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Write header
		writer.Write([]string{"Order ID", "Tanggal", "Pelanggan", "Email", "Status", "Total Item", "Total Harga"})

		for _, o := range orders {
			writer.Write([]string{
				o.UUID,
				o.CreatedAt.Format("2006-01-02 15:04:05"),
				o.BuyerName,
				o.BuyerEmail,
				o.Status,
				fmt.Sprintf("%d", o.TotalItems),
				fmt.Sprintf("%.2f", o.TotalAmount),
			})
		}
	}
}
// GetSellerOrderByID returns detailed information for a single order
func GetSellerOrderByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("id")
		userID, _ := c.Get("user_id")

		type OrderDetail struct {
			models.Order
			BuyerName       string  `json:"customer_name" db:"buyer_name"`
			BuyerEmail      string  `json:"customer_email" db:"buyer_email"`
			BuyerPhone      string  `json:"customer_phone" db:"buyer_phone"`
			ShippingAddress string  `json:"shipping_address" db:"shipping_address"`
			Items           []struct {
				ID          string  `json:"id" db:"uuid"`
				ProductName string  `json:"product_name" db:"product_name"`
				ProductImg  string  `json:"product_image" db:"product_image"`
				Quantity    int     `json:"quantity" db:"quantity"`
				Price       float64 `json:"price" db:"price"`
			} `json:"items"`
		}

		var order OrderDetail
		err := db.Get(&order, `
			SELECT 
				o.*, 
				a.full_name as buyer_name, 
				a.email as buyer_email,
				a.phone as buyer_phone,
				COALESCE(o.shipping_address, '') as shipping_address
			FROM orders o
			LEFT JOIN archers a ON o.buyer_id = a.uuid
			WHERE o.uuid = ? AND o.seller_id = ?
		`, orderID, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pesanan tidak ditemukan"})
			return
		}

		err = db.Select(&order.Items, `
			SELECT 
				oi.uuid, 
				p.name as product_name, 
				p.image_url as product_image,
				oi.quantity, 
				oi.price
			FROM order_items oi
			JOIN products p ON oi.product_id = p.uuid
			WHERE oi.order_id = ?
		`, orderID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil item pesanan"})
			return
		}

		if order.Items == nil {
			order.Items = make([]struct {
				ID          string  `json:"id" db:"uuid"`
				ProductName string  `json:"product_name" db:"product_name"`
				ProductImg  string  `json:"product_image" db:"product_image"`
				Quantity    int     `json:"quantity" db:"quantity"`
				Price       float64 `json:"price" db:"price"`
			}, 0)
		}

		c.JSON(http.StatusOK, gin.H{"data": order})
	}
}
