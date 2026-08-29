package handler

import (
	"fmt"
	"net/http"
	"time"

	"Archeris-api/models"
	"Archeris-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetCart fetches all items in cart for the logged-in user
func GetCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == nil || userID == "" {
			c.JSON(http.StatusOK, gin.H{"data": []models.CartItem{}})
			return
		}

		query := `
			SELECT 
				c.uuid as uuid,
				c.user_id as user_id,
				c.product_id as product_id,
				c.quantity as quantity,
				c.color as color,
				c.created_at as created_at,
				c.updated_at as updated_at,
				COALESCE(p.seller_id, '') as seller_id,
				p.name as product_name,
				p.price as product_price,
				p.sale_price as product_sale_price,
				COALESCE(NULLIF(p.image_url, ''), JSON_UNQUOTE(JSON_EXTRACT(p.images, '$[0]')), '') as product_image_url,
				p.stock as product_stock,
				COALESCE(s.store_name, 'Archeris Store') as seller_name
			FROM cart_items c
			JOIN products p ON c.product_id = p.uuid
			LEFT JOIN sellers s ON (p.seller_id = s.uuid OR p.seller_id = s.user_id)
			WHERE c.user_id = ?
			ORDER BY c.created_at DESC
		`
		var items []models.CartItem
		err := db.Select(&items, query, userID)
		if err != nil {
			fmt.Printf("[ERROR GetCart]: %v\n", err)
			c.JSON(http.StatusOK, gin.H{"data": []models.CartItem{}})
			return
		}

		for i := range items {
			if items[i].ProductImage != nil && *items[i].ProductImage != "" {
				masked := utils.MaskMediaURL(*items[i].ProductImage)
				items[i].ProductImage = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

// AddToCart adds a product to the user's cart
func AddToCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == nil || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Silakan login terlebih dahulu untuk menambah ke keranjang"})
			return
		}

		var req models.AddToCartRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid", "details": err.Error()})
			return
		}

		if req.Quantity <= 0 {
			req.Quantity = 1
		}

		// Check if product exists and check stock
		var productStock int
		err := db.Get(&productStock, "SELECT stock FROM products WHERE uuid = ?", req.ProductID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		if req.Quantity > productStock {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Jumlah permintaan (%d) melebihi stok yang tersedia (%d)", req.Quantity, productStock)})
			return
		}

		// Check if item already exists in cart for user
		var existingID string
		var existingQty int
		err = db.QueryRow("SELECT uuid, quantity FROM cart_items WHERE user_id = ? AND product_id = ? LIMIT 1", userID, req.ProductID).Scan(&existingID, &existingQty)

		if err == nil && existingID != "" {
			// Update quantity
			newQty := existingQty + req.Quantity
			_, err = db.Exec("UPDATE cart_items SET quantity = ?, updated_at = NOW() WHERE uuid = ?", newQty, existingID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui jumlah barang di keranjang"})
				return
			}
		} else {
			// Insert new cart item
			itemUUID := uuid.New().String()
			_, err = db.Exec("INSERT INTO cart_items (uuid, user_id, product_id, quantity, color) VALUES (?, ?, ?, ?, ?)", itemUUID, userID, req.ProductID, req.Quantity, req.Color)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan produk ke keranjang", "details": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Berhasil ditambahkan ke keranjang"})
	}
}

// UpdateCartItem updates quantity of an item in cart
func UpdateCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		itemUUID := c.Param("id")
		var req models.UpdateCartItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid"})
			return
		}

		if req.Quantity <= 0 {
			_, _ = db.Exec("DELETE FROM cart_items WHERE uuid = ? AND user_id = ?", itemUUID, userID)
		} else {
			_, _ = db.Exec("UPDATE cart_items SET quantity = ?, updated_at = NOW() WHERE uuid = ? AND user_id = ?", req.Quantity, itemUUID, userID)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Keranjang diperbarui"})
	}
}

// DeleteCartItem removes item from cart
func DeleteCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		itemUUID := c.Param("id")
		_, err := db.Exec("DELETE FROM cart_items WHERE uuid = ? AND user_id = ?", itemUUID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus barang dari keranjang"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Barang dihapus dari keranjang"})
	}
}

// CheckoutCart handles cart checkout with support for Wallet, Manual Transfer, and Payment Gateways
func CheckoutCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == nil || userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Silakan login terlebih dahulu"})
			return
		}

		var req models.CheckoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload checkout tidak valid", "details": err.Error()})
			return
		}

		userIDStr := userID.(string)

		// 1. Get cart items for user
		query := `
			SELECT 
				c.uuid as uuid, c.user_id as user_id, c.product_id as product_id,
				c.quantity as quantity, c.color as color,
				p.name as product_name, p.price as product_price, p.sale_price as product_sale_price
			FROM cart_items c
			JOIN products p ON c.product_id = p.uuid
			WHERE c.user_id = ?
		`
		var items []models.CartItem
		err := db.Select(&items, query, userIDStr)
		if err != nil || len(items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Keranjang belanja Anda kosong"})
			return
		}

		var totalAmount float64
		for _, item := range items {
			price := item.ProductPrice
			if item.ProductSale != nil && *item.ProductSale > 0 {
				price = *item.ProductSale
			}
			totalAmount += price * float64(item.Quantity)
		}

		// 2. Handle Wallet Payment
		if req.Method == "wallet" || req.Method == "WALLET" {
			var walletBalance float64
			err := db.Get(&walletBalance, "SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = ?", userIDStr)
			if err != nil || walletBalance < totalAmount {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Saldo dompet Anda tidak mencukupi (Rp %.0f). Total tagihan: Rp %.0f", walletBalance, totalAmount)})
				return
			}

			// Deduct wallet balance
			_, err = db.Exec("UPDATE wallets SET balance = balance - ?, updated_at = NOW() WHERE user_id = ?", totalAmount, userIDStr)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memotong saldo dompet"})
				return
			}

			// ── Credit seller wallet & deduct product stock for each item (95% seller, 5% fee) ────
			for _, item := range items {
				itemPrice := item.ProductPrice
				if item.ProductSale != nil && *item.ProductSale > 0 {
					itemPrice = *item.ProductSale
				}
				itemTotal := itemPrice * float64(item.Quantity)
				sellerAmount := itemTotal * 0.95

				// Deduct stock from product
				_, _ = db.Exec("UPDATE products SET stock = GREATEST(0, stock - ?) WHERE uuid = ?", item.Quantity, item.ProductID)

				// Find seller_id for this product
				var sellerID string
				_ = db.Get(&sellerID, "SELECT COALESCE(seller_id, '') FROM products WHERE uuid = ?", item.ProductID)
				if sellerID == "" {
					continue
				}
				// Upsert seller wallet and credit
				_, _ = db.Exec(`
					INSERT INTO wallets (uuid, user_id, balance) VALUES (UUID(), ?, ?)
					ON DUPLICATE KEY UPDATE balance = balance + ?
				`, sellerID, sellerAmount, sellerAmount)
			}

			// Record payment transaction as PAID
			txUUID := uuid.New().String()
			refNo := fmt.Sprintf("PAY-WLT-%d", time.Now().Unix())
			_, _ = db.Exec(`
				INSERT INTO payment_transactions 
				(uuid, user_id, payment_type, reference, merchant_ref, payment_method, amount, fee_amount, total_amount, status, paid_at, created_at)
				VALUES (?, ?, 'marketplace_product', ?, ?, 'WALLET', ?, 0, ?, 'PAID', NOW(), NOW())
			`, txUUID, userIDStr, refNo, refNo, totalAmount, totalAmount)

			// Clear user's cart
			_, _ = db.Exec("DELETE FROM cart_items WHERE user_id = ?", userIDStr)

			c.JSON(http.StatusOK, gin.H{
				"message": "Pembayaran berhasil menggunakan Saldo Dompet!",
				"status":  "PAID",
				"payment": gin.H{
					"reference": refNo,
					"status":    "PAID",
					"amount":    totalAmount,
					"method":    "WALLET",
				},
			})
			return
		}

		// 3. Handle Manual Bank Transfer Payment
		if req.Method == "manual" || req.Method == "MANUAL" {
			txUUID := uuid.New().String()
			refNo := fmt.Sprintf("PAY-MNL-%d", time.Now().Unix())
			_, _ = db.Exec(`
				INSERT INTO payment_transactions 
				(uuid, user_id, payment_type, reference, merchant_ref, payment_method, amount, fee_amount, total_amount, status, created_at)
				VALUES (?, ?, 'marketplace_product', ?, ?, 'MANUAL_TRANSFER', ?, 0, ?, 'UNPAID', NOW())
			`, txUUID, userIDStr, refNo, refNo, totalAmount, totalAmount)

			// Clear user's cart
			_, _ = db.Exec("DELETE FROM cart_items WHERE user_id = ?", userIDStr)

			c.JSON(http.StatusOK, gin.H{
				"message": "Pesanan berhasil dibuat. Silakan lakukan transfer bank manual.",
				"status": "UNPAID",
				"payment": gin.H{
					"reference": refNo,
					"status": "UNPAID",
					"amount": totalAmount,
					"method": "MANUAL_TRANSFER",
				},
			})
			return
		}

		// 4. Default / Payment Gateway
		txUUID := uuid.New().String()
		refNo := fmt.Sprintf("PAY-TP-%d", time.Now().Unix())
		_, _ = db.Exec(`
			INSERT INTO payment_transactions 
			(uuid, user_id, payment_type, reference, merchant_ref, payment_method, amount, fee_amount, total_amount, status, created_at)
			VALUES (?, ?, 'marketplace_product', ?, ?, ?, ?, 0, ?, 'UNPAID', NOW())
		`, txUUID, userIDStr, refNo, refNo, req.Method, totalAmount, totalAmount)

		// Clear user's cart
		_, _ = db.Exec("DELETE FROM cart_items WHERE user_id = ?", userIDStr)

		c.JSON(http.StatusOK, gin.H{
			"message": "Pesanan berhasil dibuat",
			"status": "UNPAID",
			"payment": gin.H{
				"reference": refNo,
				"status": "UNPAID",
				"amount": totalAmount,
				"method": req.Method,
			},
		})
	}
}
