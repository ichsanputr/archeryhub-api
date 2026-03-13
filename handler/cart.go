package handler

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetCart returns the current user's cart items
func GetCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var items []models.CartItem
		query := `
			SELECT 
				c.uuid, c.user_id, c.product_id, c.quantity, c.color, c.created_at, c.updated_at,
				p.name as product_name, 
				p.price as product_price, 
				p.sale_price as product_sale_price, 
				p.image_url as product_image_url,
				p.stock as product_stock,
				s.store_name as seller_name
			FROM cart_items c
			JOIN products p ON c.product_id = p.uuid
			JOIN sellers s ON p.seller_id = s.uuid
			WHERE c.user_id = ?
			ORDER BY c.created_at DESC
		`
		err := db.Select(&items, query, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil isi keranjang"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": items})
	}
}

// AddToCart adds a product to the user's cart
func AddToCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "archer" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya pemanah yang dapat menambah barang ke keranjang"})
			return
		}

		var req models.AddToCartRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if product exists and has enough stock
		var product models.Product
		err := db.Get(&product, "SELECT uuid, stock FROM products WHERE uuid = ?", req.ProductID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		if product.Stock < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok tidak mencukupi"})
			return
		}

		// Check if item already in cart with SAME color
		var existingID string
		err = db.Get(&existingID, "SELECT uuid FROM cart_items WHERE user_id = ? AND product_id = ? AND (color = ? OR (? = '' AND color IS NULL))", userID, req.ProductID, req.Color, req.Color)

		if err == nil {
			// Update quantity
			_, err = db.Exec("UPDATE cart_items SET quantity = quantity + ? WHERE user_id = ? AND product_id = ? AND (color = ? OR (? = '' AND color IS NULL))", req.Quantity, userID, req.ProductID, req.Color, req.Color)
		} else {
			// Insert new item
			cartID := uuid.New().String()
			colorPtr := &req.Color
			if req.Color == "" {
				colorPtr = nil
			}
			_, err = db.Exec("INSERT INTO cart_items (uuid, user_id, product_id, quantity, color) VALUES (?, ?, ?, ?, ?)", cartID, userID, req.ProductID, req.Quantity, colorPtr)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui keranjang"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Produk berhasil ditambah ke keranjang"})
	}
}

// UpdateCartItem updates the quantity of a cart item
func UpdateCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")

		var req models.UpdateCartItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify ownership and check stock
		var stockCheck struct {
			Stock int `db:"stock"`
		}
		err := db.Get(&stockCheck, `
			SELECT p.stock 
			FROM cart_items c 
			JOIN products p ON c.product_id = p.uuid 
			WHERE c.uuid = ? AND c.user_id = ?
		`, id, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Barang di keranjang tidak ditemukan"})
			return
		}

		if stockCheck.Stock < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok tidak mencukupi"})
			return
		}

		_, err = db.Exec("UPDATE cart_items SET quantity = ? WHERE uuid = ? AND user_id = ?", req.Quantity, id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui barang di keranjang"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Keranjang diperbarui"})
	}
}

// DeleteCartItem removes an item from the cart
func DeleteCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")

		res, err := db.Exec("DELETE FROM cart_items WHERE uuid = ? AND user_id = ?", id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus barang"})
			return
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Barang di keranjang tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Barang dihapus dari keranjang"})
	}
}

// CheckoutCart creates orders and a payment transaction from cart items
func CheckoutCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userEmail, _ := c.Get("email")

		var req models.CheckoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 1. Get cart items with product details
		var items []struct {
			models.CartItem
			SellerID      string   `db:"seller_id"`
			Price         float64  `db:"product_price"`
			SalePrice     *float64 `db:"product_sale_price"`
			Stock         int      `db:"product_stock"`
		}

		query := `
			SELECT 
				c.uuid, c.user_id, c.product_id, c.quantity, c.color,
				p.name as product_name, p.seller_id, p.price as product_price, p.sale_price as product_sale_price, p.stock as product_stock
			FROM cart_items c
			JOIN products p ON c.product_id = p.uuid
			WHERE c.user_id = ?
		`
		err := db.Select(&items, query, userID)
		if err != nil || len(items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Keranjang kosong atau tidak ditemukan"})
			return
		}

		// 2. Validate stock and calculate totals grouped by seller
		totalAmount := 0.0
		
		// Map for grouping items by seller
		type SellerGroup struct {
			Items []struct {
				ProductID string
				Quantity  int
				Price     float64
				Color     *string
			}
			Total float64
		}
		groups := make(map[string]*SellerGroup)

		for _, item := range items {
			if item.Quantity > item.Stock {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Stok produk %s tidak mencukupi", item.ProductName)})
				return
			}

			price := item.Price
			if item.SalePrice != nil {
				price = *item.SalePrice
			}

			if _, ok := groups[item.SellerID]; !ok {
				groups[item.SellerID] = &SellerGroup{}
			}
			
			groups[item.SellerID].Items = append(groups[item.SellerID].Items, struct {
				ProductID string
				Quantity  int
				Price     float64
				Color     *string
			}{item.ProductID, item.Quantity, price, item.Color})
			
			itemTotal := price * float64(item.Quantity)
			groups[item.SellerID].Total += itemTotal
			totalAmount += itemTotal
		}

		// 3. Create Tripay transaction
		tripay := utils.NewTripayClient()
		merchantRef := fmt.Sprintf("ORD-%s", uuid.New().String()[:12])
		
		// Get buyer info (archer)
		var archer struct {
			FullName string `db:"full_name"`
			Phone    string `db:"phone"`
		}
		err = db.Get(&archer, "SELECT full_name, COALESCE(phone, '') as phone FROM archers WHERE uuid = ?", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data profil"})
			return
		}

		customerEmail := userEmail.(string)
		customerName := archer.FullName
		customerPhone := archer.Phone
		if customerPhone == "" {
			customerPhone = "08123456789"
		}

		var tripayOrderItems []gin.H
		for _, item := range items {
			price := item.Price
			if item.SalePrice != nil {
				price = *item.SalePrice
			}
			tripayOrderItems = append(tripayOrderItems, gin.H{
				"sku":      item.ProductID,
				"name":     item.ProductName,
				"price":    int(price),
				"quantity": item.Quantity,
			})
		}

		signature := tripay.GenerateSignature(merchantRef, int(totalAmount))
		expiredTime := time.Now().Add(24 * time.Hour).Unix()

		payload := gin.H{
			"method":         req.Method,
			"merchant_ref":   merchantRef,
			"amount":         int(totalAmount),
			"customer_name":  customerName,
			"customer_email": customerEmail,
			"customer_phone": customerPhone,
			"order_items":    tripayOrderItems,
			"signature":      signature,
			"expired_time":   expiredTime,
		}

		tripayResult, err := tripay.CreateTransaction(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi pembayaran: " + err.Error()})
			return
		}

		// 4. Create database records (Transaction)
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses pesanan"})
			return
		}
		defer tx.Rollback()

		// 4a. Create Payment Transaction
		transactionID := uuid.New().String()
		tripayRef := tripayResult["reference"].(string)
		
		// Instructions
		var instructionsJSON *string
		if inst, ok := tripayResult["instructions"]; ok {
			instBytes, _ := json.Marshal(inst)
			instStr := string(instBytes)
			instructionsJSON = &instStr
		}

		_, err = tx.Exec(`
			INSERT INTO payment_transactions (
				uuid, reference, tripay_reference, user_id, amount, total_amount, 
				payment_method, va_number, qr_url, checkout_url, pay_code, 
				instructions, status, expired_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', FROM_UNIXTIME(?))
		`, transactionID, merchantRef, tripayRef, userID, totalAmount, totalAmount,
			req.Method, utils.InterfaceToStringPtr(tripayResult["pay_code"]), 
			utils.InterfaceToStringPtr(tripayResult["qr_url"]),
			utils.InterfaceToStringPtr(tripayResult["checkout_url"]),
			utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			instructionsJSON, expiredTime)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi: " + err.Error()})
			return
		}

		// 4b. Create Orders and Order Items
		for sellerID, group := range groups {
			orderID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO orders (uuid, seller_id, buyer_id, total_amount, status, payment_status, payment_id, shipping_address)
				VALUES (?, ?, ?, ?, 'pending', 'unpaid', ?, ?)
			`, orderID, sellerID, userID, group.Total, transactionID, req.ShippingAddress)
			
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat data pesanan: " + err.Error()})
				return
			}

			for _, item := range group.Items {
				_, err = tx.Exec(`
					INSERT INTO order_items (uuid, order_id, product_id, quantity, price)
					VALUES (?, ?, ?, ?, ?)
				`, uuid.New().String(), orderID, item.ProductID, item.Quantity, item.Price)
				
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan item pesanan"})
					return
				}
				
				// Optional: Update stock
				_, err = tx.Exec("UPDATE products SET stock = stock - ? WHERE uuid = ?", item.Quantity, item.ProductID)
			}
		}

		// 4c. Clear cart
		_, err = tx.Exec("DELETE FROM cart_items WHERE user_id = ?", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membersihkan keranjang"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pesanan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Pesanan berhasil dibuat",
			"reference": merchantRef,
			"payment": tripayResult,
		})
	}
}
