package handler

import (
	"archeryhub-api/models"
	"net/http"

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
				c.uuid, c.user_id, c.product_id, c.quantity, c.created_at, c.updated_at,
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cart items"})
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
			c.JSON(http.StatusForbidden, gin.H{"error": "Only archers can add items to cart"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		if product.Stock < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock"})
			return
		}

		// Check if item already in cart
		var existingID string
		err = db.Get(&existingID, "SELECT uuid FROM cart_items WHERE user_id = ? AND product_id = ?", userID, req.ProductID)

		if err == nil {
			// Update quantity
			_, err = db.Exec("UPDATE cart_items SET quantity = quantity + ? WHERE user_id = ? AND product_id = ?", req.Quantity, userID, req.ProductID)
		} else {
			// Insert new item
			cartID := uuid.New().String()
			_, err = db.Exec("INSERT INTO cart_items (uuid, user_id, product_id, quantity) VALUES (?, ?, ?, ?)", cartID, userID, req.ProductID, req.Quantity)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cart"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Product added to cart"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Cart item not found"})
			return
		}

		if stockCheck.Stock < req.Quantity {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient stock"})
			return
		}

		_, err = db.Exec("UPDATE cart_items SET quantity = ? WHERE uuid = ? AND user_id = ?", req.Quantity, id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cart item"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Cart updated"})
	}
}

// DeleteCartItem removes an item from the cart
func DeleteCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")

		res, err := db.Exec("DELETE FROM cart_items WHERE uuid = ? AND user_id = ?", id, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove item"})
			return
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Cart item not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Item removed from cart"})
	}
}
