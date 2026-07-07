package mobile

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CartItem struct {
	UUID       string  `db:"uuid" json:"id"`
	ProductID  string  `db:"product_id" json:"product_id"`
	Name       string  `db:"name" json:"name"`
	Price      float64 `db:"price" json:"price"`
	ImageURL   *string `db:"image_url" json:"image_url"`
	Quantity   int     `db:"quantity" json:"quantity"`
	Color      *string `db:"color" json:"color"`
	Stock      int     `db:"stock" json:"stock"`
}

func MobileArcherGetCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var items []CartItem
		err := db.Select(&items, `
			SELECT c.uuid, c.product_id, p.name, p.price, p.image_url, c.quantity, c.color, p.stock
			FROM cart_items c
			JOIN products p ON c.product_id = p.uuid
			WHERE c.user_id = ?
			ORDER BY c.created_at DESC
		`, userID)
		if err != nil {
			items = []CartItem{}
		}
		if items == nil {
			items = []CartItem{}
		}

		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

func MobileArcherAddToCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			ProductID string `json:"product_id" binding:"required"`
			Quantity  int    `json:"quantity"`
			Color     string `json:"color"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "product_id wajib diisi"})
			return
		}
		if req.Quantity <= 0 {
			req.Quantity = 1
		}

		// Check if already in cart
		var existingID string
		err := db.Get(&existingID, "SELECT uuid FROM cart_items WHERE user_id = ? AND product_id = ?", userID, req.ProductID)
		if err == nil && existingID != "" {
			_, _ = db.Exec("UPDATE cart_items SET quantity = quantity + ? WHERE uuid = ?", req.Quantity, existingID)
			c.JSON(http.StatusOK, gin.H{"message": "Produk sudah ada di keranjang, jumlah ditambahkan"})
			return
		}

		id := uuid.New().String()
		_, err = db.Exec("INSERT INTO cart_items (uuid, user_id, product_id, quantity, color) VALUES (?, ?, ?, ?, ?)",
			id, userID, req.ProductID, req.Quantity, req.Color)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan ke keranjang"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Berhasil ditambahkan ke keranjang", "id": id})
	}
}

func MobileArcherUpdateCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		itemID := c.Param("id")

		var req struct {
			Quantity int `json:"quantity" binding:"required,gt=0"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "quantity wajib diisi"})
			return
		}

		result, err := db.Exec("UPDATE cart_items SET quantity = ? WHERE uuid = ? AND user_id = ?", req.Quantity, itemID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui keranjang"})
			return
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Keranjang diperbarui"})
	}
}

func MobileArcherRemoveFromCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		itemID := c.Param("id")

		result, err := db.Exec("DELETE FROM cart_items WHERE uuid = ? AND user_id = ?", itemID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus item"})
			return
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Item dihapus dari keranjang"})
	}
}

func MobileArcherClearCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		_, _ = db.Exec("DELETE FROM cart_items WHERE user_id = ?", userID)
		c.JSON(http.StatusOK, gin.H{"message": "Keranjang dibersihkan"})
	}
}

func MobileArcherCheckoutCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var items []CartItem
		err = tx.Select(&items, `
			SELECT c.uuid, c.product_id, p.name, p.price, p.image_url, c.quantity, c.color, p.stock, p.seller_id
			FROM cart_items c
			JOIN products p ON c.product_id = p.uuid
			WHERE c.user_id = ?
		`, userID)
		if err != nil || len(items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Keranjang kosong"})
			return
		}

		// Group by seller
		sellerMap := make(map[string][]CartItem)
		var sellerID string
		for _, item := range items {
			if item.Stock < item.Quantity {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Stok " + item.Name + " tidak mencukupi"})
				return
			}
			// Reduce stock
			_, _ = tx.Exec("UPDATE products SET stock = stock - ? WHERE uuid = ?", item.Quantity, item.ProductID)
			sellerID = ""
			_ = tx.Get(&sellerID, "SELECT seller_id FROM products WHERE uuid = ?", item.ProductID)
			sellerMap[sellerID] = append(sellerMap[sellerID], item)
		}

		type OrderResult struct {
			OrderID string  `json:"order_id"`
			Total   float64 `json:"total"`
		}
		var results []OrderResult

		for sID, sellerItems := range sellerMap {
			orderID := uuid.New().String()
			var total float64
			for _, item := range sellerItems {
				price := item.Price
				_ = tx.Get(&price, "SELECT price FROM products WHERE uuid = ?", item.ProductID)
				total += price * float64(item.Quantity)
			}

			_, err = tx.Exec(`
				INSERT INTO orders (uuid, seller_id, buyer_id, total_amount, status, payment_status, created_at, updated_at)
				VALUES (?, ?, ?, ?, 'pending', 'unpaid', NOW(), NOW())
			`, orderID, sID, userIDStr, total)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat pesanan"})
				return
			}

			for _, item := range sellerItems {
				var price float64
				_ = tx.Get(&price, "SELECT price FROM products WHERE uuid = ?", item.ProductID)
				_, err = tx.Exec(`
					INSERT INTO order_items (uuid, order_id, product_id, quantity, price, created_at)
					VALUES (?, ?, ?, ?, ?, NOW())
				`, uuid.New().String(), orderID, item.ProductID, item.Quantity, price)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat item pesanan"})
					return
				}
			}

			results = append(results, OrderResult{OrderID: orderID, Total: total})
		}

		// Clear cart
		_, _ = tx.Exec("DELETE FROM cart_items WHERE user_id = ?", userID)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Checkout berhasil",
			"orders":  results,
		})
	}
}

func MobileArcherGetOrderHistory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)

		var orders []struct {
			UUID         string  `db:"uuid" json:"id"`
			TotalAmount  float64 `db:"total_amount" json:"total_amount"`
			Status       string  `db:"status" json:"status"`
			PaymentStatus string `db:"payment_status" json:"payment_status"`
			CreatedAt    string  `db:"created_at" json:"date"`
			ProductName  string  `db:"product_name" json:"product"`
			ItemCount    int     `db:"item_count" json:"item_count"`
		}

		err := db.Select(&orders, `
			SELECT o.uuid, o.total_amount, o.status, o.payment_status, o.created_at,
				GROUP_CONCAT(DISTINCT p.name SEPARATOR ', ') as product_name,
				COUNT(oi.uuid) as item_count
			FROM orders o
			LEFT JOIN order_items oi ON o.uuid = oi.order_id
			LEFT JOIN products p ON oi.product_id = p.uuid
			WHERE o.buyer_id = ?
			GROUP BY o.uuid
			ORDER BY o.created_at DESC
		`, userIDStr)
		if err != nil {
			orders = []struct {
				UUID         string  `db:"uuid" json:"id"`
				TotalAmount  float64 `db:"total_amount" json:"total_amount"`
				Status       string  `db:"status" json:"status"`
				PaymentStatus string `db:"payment_status" json:"payment_status"`
				CreatedAt    string  `db:"created_at" json:"date"`
				ProductName  string  `db:"product_name" json:"product"`
				ItemCount    int     `db:"item_count" json:"item_count"`
			}{}
		}

		c.JSON(http.StatusOK, gin.H{"data": orders})
	}
}
