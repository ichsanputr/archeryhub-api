package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"Archeris-api/models"
	"Archeris-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type SellerInfo struct {
	ID         string  `json:"id" db:"uuid"`
	StoreName  string  `json:"store_name" db:"store_name"`
	Slug       string  `json:"slug" db:"slug"`
	AvatarURL  *string `json:"avatar_url" db:"avatar_url"`
	City       *string `json:"city" db:"city"`
	Province   *string `json:"province" db:"province"`
	IsVerified bool    `json:"is_verified" db:"is_verified"`
	Rating     float64 `json:"rating" db:"rating"`
}

type EnrichedProductResponse struct {
	models.Product
	Seller *SellerInfo `json:"seller,omitempty"`
}

// GetProducts returns all active marketplace products
func GetProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Query("category")
		search := c.Query("search")

		query := `
			SELECT 
				uuid, seller_id, name, slug, description, price, sale_price,
				category, stock, status, COALESCE(image_url, '') as image_url,
				images, colors, specifications, COALESCE(allowed_payment_methods, '["automatic","wallet","manual"]') as allowed_payment_methods,
				views, created_at, updated_at
			FROM products 
			WHERE (status = 'active' OR status IS NULL OR status = '')
		`
		args := []interface{}{}

		if category != "" && category != "all" {
			query += " AND category = ?"
			args = append(args, category)
		}

		if search != "" {
			query += " AND (name LIKE ? OR description LIKE ?)"
			args = append(args, "%"+search+"%", "%"+search+"%")
		}

		query += " ORDER BY created_at DESC"

		var products []models.Product
		err := db.Select(&products, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk", "details": err.Error()})
			return
		}

		var enrichedList []EnrichedProductResponse
		for i := range products {
			if products[i].ImageURL != nil && *products[i].ImageURL != "" {
				masked := utils.MaskMediaURL(*products[i].ImageURL)
				products[i].ImageURL = &masked
			}

			item := EnrichedProductResponse{Product: products[i]}
			if products[i].SellerID != nil && *products[i].SellerID != "" {
				var s SellerInfo
				sErr := db.Get(&s, `
					SELECT uuid, store_name, slug, avatar_url, city, province, is_verified, rating 
					FROM sellers 
					WHERE uuid = ? OR user_id = ? OR google_id = ? 
					LIMIT 1
				`, *products[i].SellerID, *products[i].SellerID, *products[i].SellerID)
				if sErr == nil && s.StoreName != "" {
					if s.AvatarURL != nil && *s.AvatarURL != "" {
						maskedAvatar := utils.MaskMediaURL(*s.AvatarURL)
						s.AvatarURL = &maskedAvatar
					}
					item.Seller = &s
				}
			}
			enrichedList = append(enrichedList, item)
		}

		c.JSON(http.StatusOK, gin.H{
			"data": enrichedList,
			"meta": gin.H{
				"total": len(enrichedList),
				"pages": 1,
				"page":  1,
			},
		})
	}
}

// GetMyProducts returns products for the logged in seller
func GetMyProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		var products []models.Product
		query := `
			SELECT 
				uuid, seller_id, name, slug, description, price, sale_price,
				category, stock, status, COALESCE(image_url, '') as image_url,
				images, colors, specifications, COALESCE(allowed_payment_methods, '["automatic","wallet","manual"]') as allowed_payment_methods,
				views, created_at, updated_at
			FROM products 
			WHERE seller_id = ? OR seller_id IN (SELECT uuid FROM sellers WHERE user_id = ?)
			ORDER BY created_at DESC
		`
		err := db.Select(&products, query, userID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil produk saya", "details": err.Error()})
			return
		}

		for i := range products {
			if products[i].ImageURL != nil && *products[i].ImageURL != "" {
				masked := utils.MaskMediaURL(*products[i].ImageURL)
				products[i].ImageURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": products,
			"meta": gin.H{
				"total": len(products),
				"pages": 1,
				"page":  1,
			},
		})
	}
}

// GetProductByID fetches product by UUID or slug
func GetProductByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var product models.Product
		query := `
			SELECT 
				uuid, seller_id, name, slug, description, price, sale_price,
				category, stock, status, COALESCE(image_url, '') as image_url,
				images, colors, specifications, COALESCE(allowed_payment_methods, '["automatic","wallet","manual"]') as allowed_payment_methods,
				views, created_at, updated_at
			FROM products 
			WHERE uuid = ? OR slug = ?
			LIMIT 1
		`
		err := db.Get(&product, query, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		if product.ImageURL != nil && *product.ImageURL != "" {
			masked := utils.MaskMediaURL(*product.ImageURL)
			product.ImageURL = &masked
		}

		var seller SellerInfo
		if product.SellerID != nil && *product.SellerID != "" {
			sErr := db.Get(&seller, `
				SELECT uuid, store_name, slug, avatar_url, city, province, is_verified, rating 
				FROM sellers 
				WHERE uuid = ? OR user_id = ? OR google_id = ? 
				LIMIT 1
			`, *product.SellerID, *product.SellerID, *product.SellerID)
			if sErr == nil {
				if seller.AvatarURL != nil && *seller.AvatarURL != "" {
					maskedAvatar := utils.MaskMediaURL(*seller.AvatarURL)
					seller.AvatarURL = &maskedAvatar
				}
			}
		}

		resp := EnrichedProductResponse{
			Product: product,
		}
		if seller.ID != "" || seller.StoreName != "" {
			resp.Seller = &seller
		}

		c.JSON(http.StatusOK, gin.H{"data": resp})
	}
}

func CreateProduct(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists || userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Silakan login terlebih dahulu"})
			return
		}

		var req models.CreateProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid", "details": err.Error()})
			return
		}

		// Generate slug from product name
		slug := strings.ToLower(req.Name)
		slug = strings.ReplaceAll(slug, " ", "-")

		allowedJSON := `["automatic","wallet","manual"]`
		if len(req.AllowedPaymentMethods) > 0 {
			b, _ := json.Marshal(req.AllowedPaymentMethods)
			allowedJSON = string(b)
		}

		imagesJSON := "[]"
		if len(req.Images) > 0 {
			b, _ := json.Marshal(req.Images)
			imagesJSON = string(b)
		}

		colorsJSON := "[]"
		if len(req.Colors) > 0 {
			b, _ := json.Marshal(req.Colors)
			colorsJSON = string(b)
		}

		productUUID := uuid.New().String()
		sellerIDStr := userID.(string)

		if req.Status == "" {
			req.Status = "active"
		}

		_, err := db.Exec(`
			INSERT INTO products 
			(uuid, seller_id, name, slug, description, price, sale_price, category, stock, status, image_url, images, colors, allowed_payment_methods)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, productUUID, sellerIDStr, req.Name, slug, req.Description, req.Price, req.SalePrice, req.Category, req.Stock, req.Status, req.ImageURL, imagesJSON, colorsJSON, allowedJSON)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat produk", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Produk berhasil dibuat", "id": productUUID})
	}
}

func UpdateProduct(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		userRole, _ := c.Get("role")
		if !exists || userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Silakan login terlebih dahulu"})
			return
		}

		id := c.Param("id")
		userIDStr := fmt.Sprintf("%v", userID)

		// Verification: check if product exists and belongs to seller
		var productSellerID string
		err := db.Get(&productSellerID, "SELECT COALESCE(seller_id, '') FROM products WHERE uuid = ? OR slug = ?", id, id)
		if err != nil || productSellerID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		if userRole != "admin" && userRole != "root" && productSellerID != userIDStr {
			var isOwner bool
			_ = db.Get(&isOwner, "SELECT EXISTS(SELECT 1 FROM sellers WHERE (uuid = ? OR user_id = ?) AND (uuid = ? OR user_id = ?))", productSellerID, productSellerID, userIDStr, userIDStr)
			if !isOwner {
				c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki izin untuk mengedit produk ini"})
				return
			}
		}

		var req models.UpdateProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Payload tidak valid"})
			return
		}

		if len(req.AllowedPaymentMethods) > 0 {
			b, _ := json.Marshal(req.AllowedPaymentMethods)
			allowedJSON := string(b)
			_, _ = db.Exec("UPDATE products SET allowed_payment_methods = ? WHERE uuid = ? OR slug = ?", allowedJSON, id, id)
		}

		if req.Name != nil {
			slug := strings.ToLower(*req.Name)
			slug = strings.ReplaceAll(slug, " ", "-")
			_, _ = db.Exec("UPDATE products SET name = ?, slug = ? WHERE uuid = ? OR slug = ?", *req.Name, slug, id, id)
		}
		if req.Price != nil {
			_, _ = db.Exec("UPDATE products SET price = ? WHERE uuid = ? OR slug = ?", *req.Price, id, id)
		}
		if req.SalePrice != nil {
			_, _ = db.Exec("UPDATE products SET sale_price = ? WHERE uuid = ? OR slug = ?", *req.SalePrice, id, id)
		}
		if req.Stock != nil {
			_, _ = db.Exec("UPDATE products SET stock = ? WHERE uuid = ? OR slug = ?", *req.Stock, id, id)
		}
		if req.Status != nil {
			_, _ = db.Exec("UPDATE products SET status = ? WHERE uuid = ? OR slug = ?", *req.Status, id, id)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Produk berhasil diperbarui"})
	}
}

func DeleteProduct(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		userRole, _ := c.Get("role")
		if !exists || userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Silakan login terlebih dahulu"})
			return
		}

		id := c.Param("id")
		userIDStr := fmt.Sprintf("%v", userID)

		var productSellerID string
		err := db.Get(&productSellerID, "SELECT COALESCE(seller_id, '') FROM products WHERE uuid = ? OR slug = ?", id, id)
		if err != nil || productSellerID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		if userRole != "admin" && userRole != "root" && productSellerID != userIDStr {
			var isOwner bool
			_ = db.Get(&isOwner, "SELECT EXISTS(SELECT 1 FROM sellers WHERE (uuid = ? OR user_id = ?) AND (uuid = ? OR user_id = ?))", productSellerID, productSellerID, userIDStr, userIDStr)
			if !isOwner {
				c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki izin untuk menghapus produk ini"})
				return
			}
		}

		_, err = db.Exec("DELETE FROM products WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus produk"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Produk berhasil dihapus"})
	}
}

func IncrementProductViews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if id != "" {
			_, _ = db.Exec("UPDATE products SET views = views + 1 WHERE uuid = ? OR slug = ?", id, id)
		}
		c.JSON(http.StatusOK, gin.H{"message": "Analitik diperbarui"})
	}
}
