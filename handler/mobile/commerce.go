package mobile

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func ensureArcherUser(c *gin.Context) bool {
	userType, _ := c.Get("user_type")
	if fmt.Sprintf("%v", userType) != "archer" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya pemanah yang dapat mengakses endpoint ini"})
		return false
	}
	return true
}

// MobileArcherGetCart returns all items in the archer's cart
// @Summary Get Cart
// @Description Get current list of items in the shopping cart for the authenticated archer
// @Tags Mobile - Commerce
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileCartResponse
// @Router /mobile/archer/cart [get]
func MobileArcherGetCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureArcherUser(c) {
			return
		}

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
		err := db.Select(&items, query, fmt.Sprintf("%v", userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil isi keranjang"})
			return
		}

		for i := range items {
			if items[i].ProductImage != nil && *items[i].ProductImage != "" {
				masked := utils.MaskMediaURL(*items[i].ProductImage)
				items[i].ProductImage = &masked
			}
		}

		if items == nil {
			items = []models.CartItem{}
		}

		c.JSON(http.StatusOK, MobileCartResponse{Data: items})
	}
}

// MobileArcherAddToCart (POST /archer/cart)
// MobileArcherAddToCart adds a product to the cart
// @Summary Add to Cart
// @Description Add a product with specific quantity and color to the shopping cart
// @Tags Mobile - Commerce
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.AddToCartRequest true "Add to Cart Payload"
// @Success 200 {object} MessageResponse
// @Router /mobile/archer/cart [post]
func MobileArcherAddToCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureArcherUser(c) {
			return
		}

		userID, _ := c.Get("user_id")
		var req models.AddToCartRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid: " + err.Error()})
			return
		}

		// Check if product exists and is active
		var product struct {
			UUID  string `db:"uuid"`
			Stock int    `db:"stock"`
		}
		err := db.Get(&product, "SELECT uuid, stock FROM products WHERE uuid = ? AND status = 'active'", req.ProductID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan atau tidak aktif"})
			return
		}

		if req.Quantity > product.Stock {
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Stok tidak mencukupi, sisa stok: %d", product.Stock)})
			return
		}

		// Check if already in cart
		var existing models.CartItem
		err = db.Get(&existing, "SELECT uuid, quantity FROM cart_items WHERE user_id = ? AND product_id = ?", fmt.Sprintf("%v", userID), req.ProductID)
		if err == nil {
			// Update quantity
			newQty := existing.Quantity + req.Quantity
			if newQty > product.Stock {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Total kuantitas di keranjang melebihi stok"})
				return
			}
			_, err = db.Exec("UPDATE cart_items SET quantity = ?, updated_at = NOW() WHERE uuid = ?", newQty, existing.UUID)
		} else {
			// Insert new
			colorParam := interface{}(nil)
			if req.Color != "" {
				colorParam = req.Color
			}
			_, err = db.Exec(`
				INSERT INTO cart_items (uuid, user_id, product_id, quantity, color)
				VALUES (?, ?, ?, ?, ?)
			`, uuid.New().String(), fmt.Sprintf("%v", userID), req.ProductID, req.Quantity, colorParam)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses keranjang: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Produk berhasil ditambahkan ke keranjang"})
	}
}

// MobileArcherUpdateCartItem (PUT /archer/cart/:id)
// MobileArcherUpdateCartItem updates quantity of a cart item
// @Summary Update Cart Item
// @Description Update the quantity of a specific item in the cart
// @Tags Mobile - Commerce
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Cart Item UUID"
// @Param request body models.UpdateCartItemRequest true "Update Quantity"
// @Success 200 {object} MessageResponse
// @Router /mobile/archer/cart/{id} [put]
func MobileArcherUpdateCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureArcherUser(c) {
			return
		}

		itemID := c.Param("id")
		userID, _ := c.Get("user_id")

		var req models.UpdateCartItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}

		// Verify ownership and check stock
		var info struct {
			Stock int `db:"product_stock"`
		}
		query := `
			SELECT p.stock as product_stock 
			FROM cart_items c 
			JOIN products p ON c.product_id = p.uuid 
			WHERE c.uuid = ? AND c.user_id = ?
		`
		err := db.Get(&info, query, itemID, fmt.Sprintf("%v", userID))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item keranjang tidak ditemukan"})
			return
		}

		if req.Quantity > info.Stock {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kuantitas melebihi stok yang tersedia"})
			return
		}

		_, err = db.Exec("UPDATE cart_items SET quantity = ?, updated_at = NOW() WHERE uuid = ? AND user_id = ?", req.Quantity, itemID, fmt.Sprintf("%v", userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui keranjang"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Kuantitas keranjang berhasil diperbarui"})
	}
}

// MobileArcherRemoveFromCart (DELETE /archer/cart/:id)
// MobileArcherRemoveFromCart removes an item from the cart
// @Summary Remove from Cart
// @Description Delete a specific item from the shopping cart
// @Tags Mobile - Commerce
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Cart Item UUID"
// @Success 200 {object} MessageResponse
// @Router /mobile/archer/cart/{id} [delete]
func MobileArcherRemoveFromCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureArcherUser(c) {
			return
		}

		itemID := c.Param("id")
		userID, _ := c.Get("user_id")

		result, err := db.Exec("DELETE FROM cart_items WHERE uuid = ? AND user_id = ?", itemID, fmt.Sprintf("%v", userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus item"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Item berhasil dihapus dari keranjang"})
	}
}

// MobileArcherClearCart (DELETE /archer/cart)
// MobileArcherClearCart removes all items from the cart
// @Summary Clear Cart
// @Description Delete all items from the shopping cart for the current archer
// @Tags Mobile - Commerce
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MessageResponse
// @Router /mobile/archer/cart [delete]
func MobileArcherClearCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureArcherUser(c) {
			return
		}

		userID, _ := c.Get("user_id")
		_, err := db.Exec("DELETE FROM cart_items WHERE user_id = ?", fmt.Sprintf("%v", userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengosongkan keranjang"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Keranjang berhasil dikosongkan"})
	}
}

// MobileArcherCheckoutCart processes the checkout and creates orders
// @Summary Checkout Cart
// @Description Create orders from cart items and initiate payment transaction
// @Tags Mobile - Commerce
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body models.CheckoutRequest true "Checkout Details"
// @Success 200 {object} MobileCheckoutResponse
// @Router /mobile/archer/cart/checkout [post]
func MobileArcherCheckoutCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureArcherUser(c) {
			return
		}

		userID, _ := c.Get("user_id")
		userEmail, _ := c.Get("email")

		var req models.CheckoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var items []struct {
			models.CartItem
			SellerID  string   `db:"seller_id"`
			Price     float64  `db:"product_price"`
			SalePrice *float64 `db:"product_sale_price"`
			Stock     int      `db:"product_stock"`
		}

		query := `
			SELECT
				c.uuid, c.user_id, c.product_id, c.quantity, c.color,
				p.name as product_name, p.seller_id, p.price as product_price, p.sale_price as product_sale_price, p.stock as product_stock
			FROM cart_items c
			JOIN products p ON c.product_id = p.uuid
			WHERE c.user_id = ?
		`
		err := db.Select(&items, query, fmt.Sprintf("%v", userID))
		if err != nil || len(items) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Keranjang kosong atau tidak ditemukan"})
			return
		}

		totalAmount := 0.0
		type SellerItem struct {
			ProductID string
			Quantity  int
			Price     float64
			Color     *string
		}
		type SellerGroup struct {
			Items []SellerItem
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

			groups[item.SellerID].Items = append(groups[item.SellerID].Items, SellerItem{item.ProductID, item.Quantity, price, item.Color})
			itemTotal := price * float64(item.Quantity)
			groups[item.SellerID].Total += itemTotal
			totalAmount += itemTotal
		}

		tripay := utils.NewTripayClient()
		merchantRef := fmt.Sprintf("ORD-%s", uuid.New().String()[:12])

		var archer struct {
			FullName string `db:"full_name"`
			Phone    string `db:"phone"`
		}
		err = db.Get(&archer, "SELECT full_name, COALESCE(phone, '') as phone FROM archers WHERE uuid = ?", fmt.Sprintf("%v", userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data profil"})
			return
		}

		customerEmail := fmt.Sprintf("%v", userEmail)
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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses pesanan"})
			return
		}
		defer tx.Rollback()

		transactionID := uuid.New().String()
		tripayRef := tripayResult["reference"].(string)

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
		`, transactionID, merchantRef, tripayRef, fmt.Sprintf("%v", userID), totalAmount, totalAmount,
			req.Method, utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			utils.InterfaceToStringPtr(tripayResult["qr_url"]),
			utils.InterfaceToStringPtr(tripayResult["checkout_url"]),
			utils.InterfaceToStringPtr(tripayResult["pay_code"]),
			instructionsJSON, expiredTime)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi: " + err.Error()})
			return
		}

		for sellerID, group := range groups {
			orderID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO orders (uuid, seller_id, buyer_id, total_amount, status, payment_status, payment_id, shipping_address)
				VALUES (?, ?, ?, ?, 'pending', 'unpaid', ?, ?)
			`, orderID, sellerID, fmt.Sprintf("%v", userID), group.Total, transactionID, req.ShippingAddress)
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

				_, err = tx.Exec("UPDATE products SET stock = stock - ? WHERE uuid = ?", item.Quantity, item.ProductID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui stok produk"})
					return
				}
			}
		}

		_, err = tx.Exec("DELETE FROM cart_items WHERE user_id = ?", fmt.Sprintf("%v", userID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membersihkan keranjang"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pesanan"})
			return
		}

		c.JSON(http.StatusOK, MobileCheckoutResponse{
			Message:   "Pesanan berhasil dibuat",
			Reference: merchantRef,
			Payment:   tripayResult,
		})
	}
}

// MobileArcherGetOrderHistory returns order history for the archer
// @Summary Get Order History
// @Description Get a list of all past orders for the authenticated archer
// @Tags Mobile - Commerce
// @Produce json
// @Security ApiKeyAuth
// @Param status query string false "Filter by order status"
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Success 200 {object} MobileArcherOrdersResponse
// @Router /mobile/archer/orders [get]
func MobileArcherGetOrderHistory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !ensureArcherUser(c) {
			return
		}

		userID, _ := c.Get("user_id")
		status := c.Query("status")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		query := `
			SELECT
				o.uuid,
				o.seller_id,
				COALESCE(s.store_name, '') as seller_name,
				o.total_amount,
				o.status,
				o.payment_status,
				COALESCE((SELECT SUM(quantity) FROM order_items oi WHERE oi.order_id = o.uuid), 0) as total_items,
				pt.reference,
				pt.checkout_url,
				pt.payment_method,
				o.created_at
			FROM orders o
			LEFT JOIN sellers s ON o.seller_id = s.uuid
			LEFT JOIN payment_transactions pt ON o.payment_id = pt.uuid
			WHERE o.buyer_id = ?
		`
		args := []interface{}{fmt.Sprintf("%v", userID)}
		if status != "" && status != "all" {
			query += " AND o.status = ?"
			args = append(args, status)
		}
		query += " ORDER BY o.created_at DESC LIMIT ? OFFSET ?"
		args = append(args, limit, offset)

		var orders []MobileOrderHistoryItem
		if err := db.Select(&orders, query, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat pesanan"})
			return
		}
		if orders == nil {
			orders = []MobileOrderHistoryItem{}
		}

		var total int
		countQuery := "SELECT COUNT(*) FROM orders WHERE buyer_id = ?"
		countArgs := []interface{}{fmt.Sprintf("%v", userID)}
		if status != "" && status != "all" {
			countQuery += " AND status = ?"
			countArgs = append(countArgs, status)
		}
		_ = db.Get(&total, countQuery, countArgs...)

		c.JSON(http.StatusOK, MobileArcherOrdersResponse{
			Orders: orders,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		})
	}
}
