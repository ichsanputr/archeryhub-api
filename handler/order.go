package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

type SellerOrderItem struct {
	ID          string  `json:"id"           db:"id"`
	ProductID   string  `json:"product_id"   db:"product_id"`
	ProductName string  `json:"product_name" db:"product_name"`
	Price       float64 `json:"price"        db:"price"`
	Quantity    int     `json:"quantity"     db:"quantity"`
	ImageURL    *string `json:"image_url"    db:"image_url"`
}

type SellerOrder struct {
	UUID            string            `json:"uuid"             db:"uuid"`
	ID              string            `json:"id"               db:"id"`
	Reference       string            `json:"reference"        db:"reference"`
	CustomerName    string            `json:"customer_name"    db:"customer_name"`
	BuyerName       string            `json:"buyer_name"       db:"buyer_name"`
	CustomerEmail   string            `json:"customer_email"   db:"customer_email"`
	BuyerEmail      string            `json:"buyer_email"      db:"buyer_email"`
	BuyerPhone      *string           `json:"buyer_phone"      db:"buyer_phone"`
	TotalAmount     float64           `json:"total_amount"     db:"total_amount"`
	Amount          float64           `json:"amount"           db:"amount"`
	SellerAmount    float64           `json:"seller_amount"    db:"seller_amount"`
	Status          string            `json:"status"           db:"status"`
	PaymentStatus   string            `json:"payment_status"   db:"payment_status"`
	PaymentMethod   string            `json:"payment_method"   db:"payment_method"`
	ShippingAddress *string           `json:"shipping_address" db:"shipping_address"`
	CreatedAt       string            `json:"created_at"       db:"created_at"`
	TotalItems      int               `json:"total_items"      db:"total_items"`
	Items           []SellerOrderItem `json:"items"`
}

// CreateMarketplaceOrder creates a new order in orders & order_items table
// POST /orders
func CreateMarketplaceOrder(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)

		var req struct {
			SellerID        string `json:"seller_id"`
			ShippingAddress string `json:"shipping_address"`
			Notes           string `json:"notes"`
			Items           []struct {
				ProductID string  `json:"product_id" binding:"required"`
				Quantity  int     `json:"quantity"   binding:"required"`
				Price     float64 `json:"price"      binding:"required"`
				Color     string  `json:"color"`
			} `json:"items" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data pesanan tidak valid: " + err.Error()})
			return
		}

		if len(req.Items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Item pesanan tidak boleh kosong"})
			return
		}

		// Resolve seller UUID
		var sellerUUID string
		if req.SellerID != "" {
			_ = db.Get(&sellerUUID, "SELECT uuid FROM sellers WHERE uuid = ? OR user_id = ? LIMIT 1", req.SellerID, req.SellerID)
		}
		if sellerUUID == "" && len(req.Items) > 0 {
			_ = db.Get(&sellerUUID, "SELECT s.uuid FROM products p JOIN sellers s ON (p.seller_id = s.uuid OR p.seller_id = s.user_id) WHERE p.uuid = ? LIMIT 1", req.Items[0].ProductID)
			if sellerUUID == "" {
				_ = db.Get(&sellerUUID, "SELECT seller_id FROM products WHERE uuid = ? AND seller_id IS NOT NULL LIMIT 1", req.Items[0].ProductID)
			}
		}
		if sellerUUID == "" {
			_ = db.Get(&sellerUUID, "SELECT uuid FROM sellers ORDER BY created_at ASC LIMIT 1")
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi pesanan"})
			return
		}
		defer tx.Rollback()

		var totalAmount float64
		for _, it := range req.Items {
			// Check and deduct stock atomically
			var prod struct {
				Stock *int   `db:"stock"`
				Name  string `db:"name"`
			}
			if err := tx.Get(&prod, "SELECT stock, name FROM products WHERE uuid = ? FOR UPDATE", it.ProductID); err == nil {
				if prod.Stock != nil && *prod.Stock < it.Quantity {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": fmt.Sprintf("Stok produk '%s' tidak mencukupi (sisa: %d)", prod.Name, *prod.Stock),
					})
					return
				}
				if prod.Stock != nil {
					_, _ = tx.Exec("UPDATE products SET stock = stock - ?, updated_at = NOW() WHERE uuid = ? AND stock >= ?", it.Quantity, it.ProductID, it.Quantity)
				}
			}

			totalAmount += it.Price * float64(it.Quantity)
		}

		orderUUID := uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO orders (uuid, seller_id, buyer_id, total_amount, status, payment_status, shipping_address, created_at, updated_at)
			VALUES (?, ?, ?, ?, 'pending', 'unpaid', ?, NOW(), NOW())
		`, orderUUID, sellerUUID, userIDStr, totalAmount, req.ShippingAddress)
		if err != nil {
			logrus.WithError(err).Error("[CreateMarketplaceOrder] INSERT orders failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat pesanan: " + err.Error()})
			return
		}

		for _, it := range req.Items {
			itemUUID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO order_items (uuid, order_id, product_id, quantity, price, created_at)
				VALUES (?, ?, ?, ?, ?, NOW())
			`, itemUUID, orderUUID, it.ProductID, it.Quantity, it.Price)
			if err != nil {
				logrus.WithError(err).Error("[CreateMarketplaceOrder] INSERT order_items failed")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan rincian item pesanan"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan transaksi pesanan"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":  "Pesanan berhasil dibuat",
			"order_id": orderUUID,
			"uuid":     orderUUID,
		})
	}
}

// GetSellerOrders returns all marketplace orders for the logged-in seller.
func GetSellerOrders(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)
		status := c.Query("status")
		page := 1
		limit := 50

		// Resolve seller UUID
		var sellerUUID string
		_ = db.Get(&sellerUUID, "SELECT uuid FROM sellers WHERE uuid = ? OR user_id = ? LIMIT 1", userIDStr, userIDStr)
		if sellerUUID == "" {
			sellerUUID = userIDStr
		}

		query := `
			SELECT
				o.uuid,
				o.uuid AS id,
				o.uuid AS reference,
				COALESCE(a.full_name, 'Pelanggan') AS customer_name,
				COALESCE(a.full_name, 'Pelanggan') AS buyer_name,
				COALESCE(a.email, '') AS customer_email,
				COALESCE(a.email, '') AS buyer_email,
				COALESCE(a.phone, '') AS buyer_phone,
				o.total_amount,
				o.total_amount AS amount,
				o.total_amount * 0.95 AS seller_amount,
				COALESCE(o.status, 'pending') AS status,
				COALESCE(o.payment_status, 'unpaid') AS payment_status,
				'Chat / Manual' AS payment_method,
				COALESCE(o.shipping_address, '') AS shipping_address,
				DATE_FORMAT(o.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
				COALESCE((SELECT COUNT(*) FROM order_items oi WHERE oi.order_id = o.uuid), 1) AS total_items
			FROM orders o
			LEFT JOIN archers a ON o.buyer_id = a.uuid
			WHERE o.seller_id = ? OR o.seller_id = ?
		`
		args := []interface{}{sellerUUID, userIDStr}
		if status != "" && strings.ToLower(status) != "all" {
			query += " AND o.status = ?"
			args = append(args, strings.ToLower(status))
		}
		query += " ORDER BY o.created_at DESC LIMIT ? OFFSET ?"
		args = append(args, limit, (page-1)*limit)

		var orders []SellerOrder
		if err := db.Select(&orders, query, args...); err != nil {
			logrus.WithError(err).Error("[GetSellerOrders] db.Select orders failed")
			c.JSON(http.StatusOK, gin.H{"data": []SellerOrder{}, "orders": []SellerOrder{}, "meta": gin.H{"total": 0, "page": page, "pages": 1}})
			return
		}
		if orders == nil {
			orders = []SellerOrder{}
		}
		c.JSON(http.StatusOK, gin.H{"data": orders, "orders": orders, "meta": gin.H{"total": len(orders), "page": page, "pages": 1}})
	}
}

// GetSellerStats returns revenue, order count, and products sold for the seller.
func GetSellerStats(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)

		var sellerUUID string
		_ = db.Get(&sellerUUID, "SELECT uuid FROM sellers WHERE uuid = ? OR user_id = ? LIMIT 1", userIDStr, userIDStr)
		if sellerUUID == "" {
			sellerUUID = userIDStr
		}

		var stats struct {
			TotalRevenue float64 `db:"total_revenue"`
			TotalOrders  int     `db:"total_orders"`
		}
		_ = db.Get(&stats, `
			SELECT
				COALESCE(SUM(total_amount), 0) AS total_revenue,
				COUNT(*) AS total_orders
			FROM orders
			WHERE seller_id = ? OR seller_id = ?
		`, sellerUUID, userIDStr)

		var productsSold int
		_ = db.Get(&productsSold, `
			SELECT COALESCE(SUM(oi.quantity), 0)
			FROM order_items oi
			JOIN orders o ON oi.order_id = o.uuid
			WHERE o.seller_id = ? OR o.seller_id = ?
		`, sellerUUID, userIDStr)

		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"total_revenue": stats.TotalRevenue,
			"total_orders":  stats.TotalOrders,
			"products_sold": productsSold,
			"rating":        0.0,
		}})
	}
}

// UpdateOrderStatus lets seller update an order status (e.g., processing -> shipped).
func UpdateOrderStatus(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)
		orderID := c.Param("id")

		var sellerUUID string
		_ = db.Get(&sellerUUID, "SELECT uuid FROM sellers WHERE uuid = ? OR user_id = ? LIMIT 1", userIDStr, userIDStr)
		if sellerUUID == "" {
			sellerUUID = userIDStr
		}

		var req struct {
			Status string `json:"status" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Status tidak valid"})
			return
		}

		res, err := db.Exec(`
			UPDATE orders
			SET status = ?, updated_at = NOW()
			WHERE (uuid = ?) AND (seller_id = ? OR seller_id = ?)
		`, strings.ToLower(req.Status), orderID, sellerUUID, userIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status pesanan: " + err.Error()})
			return
		}
		rows, _ := res.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pesanan tidak ditemukan atau bukan milik Anda"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Status pesanan diperbarui"})
	}
}

// ExportSellerOrders returns orders as CSV for download.
func ExportSellerOrders(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)

		var sellerUUID string
		_ = db.Get(&sellerUUID, "SELECT uuid FROM sellers WHERE uuid = ? OR user_id = ? LIMIT 1", userIDStr, userIDStr)
		if sellerUUID == "" {
			sellerUUID = userIDStr
		}

		var orders []SellerOrder
		_ = db.Select(&orders, `
			SELECT
				o.uuid,
				o.uuid AS id,
				o.uuid AS reference,
				COALESCE(a.full_name, 'Pelanggan') AS customer_name,
				COALESCE(a.full_name, 'Pelanggan') AS buyer_name,
				COALESCE(a.email, '') AS customer_email,
				COALESCE(a.email, '') AS buyer_email,
				COALESCE(a.phone, '') AS buyer_phone,
				o.total_amount,
				o.total_amount AS amount,
				o.total_amount * 0.95 AS seller_amount,
				COALESCE(o.status, 'pending') AS status,
				COALESCE(o.payment_status, 'unpaid') AS payment_status,
				'Chat / Manual' AS payment_method,
				COALESCE(o.shipping_address, '') AS shipping_address,
				DATE_FORMAT(o.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
				COALESCE((SELECT COUNT(*) FROM order_items oi WHERE oi.order_id = o.uuid), 1) AS total_items
			FROM orders o
			LEFT JOIN archers a ON o.buyer_id = a.uuid
			WHERE o.seller_id = ? OR o.seller_id = ?
			ORDER BY o.created_at DESC
		`, sellerUUID, userIDStr)

		filename := fmt.Sprintf("orders-%s.csv", time.Now().Format("20060102"))
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Header("Content-Type", "text/csv")

		w := csv.NewWriter(c.Writer)
		_ = w.Write([]string{"No Order", "Nama Pembeli", "Email Pembeli", "Total Tagihan", "Status", "Metode Bayar", "Tanggal"})
		for _, o := range orders {
			_ = w.Write([]string{
				o.Reference, o.BuyerName, o.BuyerEmail,
				fmt.Sprintf("%.0f", o.TotalAmount),
				o.Status, o.PaymentMethod, o.CreatedAt,
			})
		}
		w.Flush()
	}
}

// GetSellerOrderByID returns a single order detail for seller.
func GetSellerOrderByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)
		orderID := c.Param("id")

		var sellerUUID string
		_ = db.Get(&sellerUUID, "SELECT uuid FROM sellers WHERE uuid = ? OR user_id = ? LIMIT 1", userIDStr, userIDStr)
		if sellerUUID == "" {
			sellerUUID = userIDStr
		}

		var order SellerOrder
		err := db.Get(&order, `
			SELECT
				o.uuid,
				o.uuid AS id,
				o.uuid AS reference,
				COALESCE(a.full_name, 'Pelanggan') AS customer_name,
				COALESCE(a.full_name, 'Pelanggan') AS buyer_name,
				COALESCE(a.email, '') AS customer_email,
				COALESCE(a.email, '') AS buyer_email,
				COALESCE(a.phone, '') AS buyer_phone,
				o.total_amount,
				o.total_amount AS amount,
				o.total_amount * 0.95 AS seller_amount,
				COALESCE(o.status, 'pending') AS status,
				COALESCE(o.payment_status, 'unpaid') AS payment_status,
				'Chat / Manual' AS payment_method,
				COALESCE(o.shipping_address, '') AS shipping_address,
				DATE_FORMAT(o.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
				1 AS total_items
			FROM orders o
			LEFT JOIN archers a ON o.buyer_id = a.uuid
			WHERE (o.uuid = ?) AND (o.seller_id = ? OR o.seller_id = ?)
		`, orderID, sellerUUID, userIDStr)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pesanan tidak ditemukan"})
			return
		}

		var items []SellerOrderItem
		_ = db.Select(&items, `
			SELECT
				oi.uuid AS id,
				oi.product_id,
				COALESCE(p.name, 'Produk') AS product_name,
				oi.price,
				oi.quantity,
				COALESCE(NULLIF(p.image_url, ''), JSON_UNQUOTE(JSON_EXTRACT(p.images, '$[0]')), '') AS image_url
			FROM order_items oi
			LEFT JOIN products p ON oi.product_id = p.uuid
			WHERE oi.order_id = ?
		`, order.UUID)
		if items == nil {
			items = []SellerOrderItem{}
		}
		order.Items = items
		order.TotalItems = len(items)

		c.JSON(http.StatusOK, gin.H{"data": order, "order": order})
	}
}

// ApproveSellerOrderPayment lets seller mark order as paid / done
func ApproveSellerOrderPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)
		orderID := c.Param("id")

		var sellerUUID string
		_ = db.Get(&sellerUUID, "SELECT uuid FROM sellers WHERE uuid = ? OR user_id = ? LIMIT 1", userIDStr, userIDStr)
		if sellerUUID == "" {
			sellerUUID = userIDStr
		}

		_, err := db.Exec(`
			UPDATE orders 
			SET status = 'done', payment_status = 'paid', updated_at = NOW()
			WHERE (uuid = ?) AND (seller_id = ? OR seller_id = ?)
		`, orderID, sellerUUID, userIDStr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyetujui pesanan: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Pesanan berhasil diselesaikan",
			"status":  "done",
		})
	}
}

// UploadOrderProof allows buyer to upload payment receipt
func UploadOrderProof(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderID := c.Param("id")

		var req struct {
			ProofURL   string `json:"proof_url" binding:"required"`
			SenderName string `json:"sender_name"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bukti transfer tidak valid"})
			return
		}

		_, err := db.Exec(`
			UPDATE orders 
			SET status = 'processing', updated_at = NOW()
			WHERE uuid = ?
		`, orderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan status: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   "Bukti transfer berhasil dikonfirmasi.",
			"status":    "processing",
			"proof_url": req.ProofURL,
		})
	}
}
