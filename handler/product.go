package handler

import (
	"archeryhub-api/models"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"archeryhub-api/utils"
)

// GetProducts returns all products (public)
func GetProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, offset, page := utils.GetPaginationParams(c)
		category := c.Query("category")
		search := c.Query("search")

		whereClause := "WHERE status = 'active'"
		args := []interface{}{}

		if category != "" && category != "all" {
			whereClause += " AND category = ?"
			args = append(args, category)
		}
		if search != "" {
			whereClause += " AND (name LIKE ? OR description LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm)
		}

		// Count total
		var totalCount int
		err := db.Get(&totalCount, "SELECT COUNT(*) FROM products "+whereClause, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data produk"})
			return
		}

		// Get data
		var products []models.Product
		query := "SELECT * FROM products " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
		queryArgs := append(args, limit, offset)

		err = db.Select(&products, query, queryArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk"})
			return
		}

		if products == nil {
			products = []models.Product{}
		}

		for i := range products {
			if products[i].ImageURL != nil && *products[i].ImageURL != "" {
				masked := utils.MaskMediaURL(*products[i].ImageURL)
				products[i].ImageURL = &masked
			}
			if products[i].Images != nil && *products[i].Images != "" {
				var images []string
				json.Unmarshal([]byte(*products[i].Images), &images)
				for j, img := range images {
					images[j] = utils.MaskMediaURL(img)
				}
				maskedJSON, _ := json.Marshal(images)
				maskedStr := string(maskedJSON)
				products[i].Images = &maskedStr
			}
		}

		meta := utils.CalculatePagination(totalCount, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{"data": products, "meta": meta})
	}
}

// GetMyProducts returns products owned by the current seller
func GetMyProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "seller" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya penjual yang bisa melihat produk mereka"})
			return
		}

		limit, offset, page := utils.GetPaginationParams(c)
		status := c.Query("status")
		category := c.Query("category")
		search := c.Query("search")

		whereClause := "WHERE seller_id = ?"
		args := []interface{}{userID}

		if status != "" && status != "all" {
			whereClause += " AND status = ?"
			args = append(args, status)
		}
		if category != "" && category != "all" {
			whereClause += " AND category = ?"
			args = append(args, category)
		}
		if search != "" {
			whereClause += " AND (name LIKE ? OR description LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm)
		}

		// Count total
		var totalCount int
		err := db.Get(&totalCount, "SELECT COUNT(*) FROM products "+whereClause, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data produk"})
			return
		}

		// Get data
		var products []models.Product
		query := "SELECT * FROM products " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
		queryArgs := append(args, limit, offset)

		err = db.Select(&products, query, queryArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk"})
			return
		}

		if products == nil {
			products = []models.Product{}
		}

		for i := range products {
			if products[i].ImageURL != nil && *products[i].ImageURL != "" {
				masked := utils.MaskMediaURL(*products[i].ImageURL)
				products[i].ImageURL = &masked
			}
			if products[i].Images != nil && *products[i].Images != "" {
				var images []string
				json.Unmarshal([]byte(*products[i].Images), &images)
				for j, img := range images {
					images[j] = utils.MaskMediaURL(img)
				}
				maskedJSON, _ := json.Marshal(images)
				maskedStr := string(maskedJSON)
				products[i].Images = &maskedStr
			}
		}

		meta := utils.CalculatePagination(totalCount, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{"data": products, "meta": meta})
	}
}

// GetProductByID returns a single product
func GetProductByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var product models.Product
		err := db.Get(&product, "SELECT * FROM products WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		// Increment views
		db.Exec("UPDATE products SET views = COALESCE(views, 0) + 1 WHERE uuid = ?", product.UUID)

		if product.ImageURL != nil && *product.ImageURL != "" {
			masked := utils.MaskMediaURL(*product.ImageURL)
			product.ImageURL = &masked
		}
		if product.Images != nil && *product.Images != "" {
			var images []string
			json.Unmarshal([]byte(*product.Images), &images)
			for j, img := range images {
				images[j] = utils.MaskMediaURL(img)
			}
			maskedJSON, _ := json.Marshal(images)
			maskedStr := string(maskedJSON)
			product.Images = &maskedStr
		}
		if product.Colors != nil && *product.Colors != "" {
			var colors []string
			json.Unmarshal([]byte(*product.Colors), &colors)
			maskedJSON, _ := json.Marshal(colors)
			maskedStr := string(maskedJSON)
			product.Colors = &maskedStr
		}

		// Enrich with seller info (explicit columns so DB schema matches struct)
		var enriched models.EnrichedProduct
		enriched.Product = product

		if product.SellerID != nil && *product.SellerID != "" {
			var seller models.Seller
			err = db.Get(&seller, `
				SELECT uuid, user_id,
				       COALESCE(slug, '') AS slug, COALESCE(email, '') AS email,
				       store_name, description, avatar_url, banner_url,
				       phone, address, city, province, COALESCE(is_verified, 0) AS is_verified,
				       COALESCE(rating, 0) AS rating, COALESCE(total_sales, 0) AS total_sales,
				       COALESCE(followers_count, 0) AS followers_count, COALESCE(product_count, 0) AS product_count,
				       COALESCE(chat_response_rate, '') AS chat_response_rate,
				       COALESCE(chat_response_time, '') AS chat_response_time,
				       last_active_at, COALESCE(status, '') AS status,
				       created_at, updated_at
				FROM sellers WHERE uuid = ?`, *product.SellerID)
			if err == nil {
				if seller.AvatarURL != nil && *seller.AvatarURL != "" {
					masked := utils.MaskMediaURL(*seller.AvatarURL)
					seller.AvatarURL = &masked
				}
				if seller.BannerURL != nil && *seller.BannerURL != "" {
					masked := utils.MaskMediaURL(*seller.BannerURL)
					seller.BannerURL = &masked
				}
				enriched.Seller = &seller
			}
		}

		c.JSON(http.StatusOK, gin.H{"data": enriched})
	}
}

// CreateProduct creates a new product
func CreateProduct(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var req models.CreateProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		productID := uuid.New().String()
		slug := strings.ToLower(req.Name)
		slug = strings.ReplaceAll(slug, " ", "-") + "-" + uuid.New().String()[:8]

		if userType != "seller" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya penjual yang bisa membuat produk"})
			return
		}

		imagesJSON, _ := json.Marshal(req.Images)
		colorsJSON, _ := json.Marshal(req.Colors)
		specJSON, _ := json.Marshal(req.Specifications)

		userIDStr := userID.(string)
		sellerID := &userIDStr

		if req.Status == "" {
			req.Status = "draft"
		}

		_, err := db.Exec(`
			INSERT INTO products (uuid, seller_id, name, slug, description, price, sale_price, category, stock, status, image_url, images, colors, specifications)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, productID, sellerID, req.Name, slug, req.Description, req.Price, req.SalePrice, req.Category, req.Stock, req.Status, req.ImageURL, string(imagesJSON), string(colorsJSON), string(specJSON))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat produk: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Produk berhasil dibuat",
			"id":      productID,
		})
	}
}

// UpdateProduct updates an existing product
func UpdateProduct(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var req models.UpdateProductRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify ownership
		var product models.Product
		err := db.Get(&product, "SELECT * FROM products WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		if userType != "seller" || product.SellerID == nil || *product.SellerID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak diizinkan untuk mengubah produk ini"})
			return
		}

		// Update fields if provided
		query := "UPDATE products SET updated_at = NOW()"
		args := []interface{}{}

		if req.Name != nil {
			query += ", name = ?"
			args = append(args, *req.Name)
		}
		if req.Description != nil {
			query += ", description = ?"
			args = append(args, *req.Description)
		}
		if req.Price != nil {
			query += ", price = ?"
			args = append(args, *req.Price)
		}
		if req.SalePrice != nil {
			query += ", sale_price = ?"
			args = append(args, *req.SalePrice)
		}
		if req.Category != nil {
			query += ", category = ?"
			args = append(args, *req.Category)
		}
		if req.Stock != nil {
			query += ", stock = ?"
			args = append(args, *req.Stock)
		}
		if req.Status != nil {
			query += ", status = ?"
			args = append(args, *req.Status)
		}
		if req.ImageURL != nil {
			query += ", image_url = ?"
			args = append(args, *req.ImageURL)
		}
		if req.Images != nil {
			imagesJSON, _ := json.Marshal(req.Images)
			query += ", images = ?"
			args = append(args, string(imagesJSON))
		}
		if req.Colors != nil {
			colorsJSON, _ := json.Marshal(req.Colors)
			query += ", colors = ?"
			args = append(args, string(colorsJSON))
		}
		if req.Specifications != nil {
			specJSON, _ := json.Marshal(req.Specifications)
			query += ", specifications = ?"
			args = append(args, string(specJSON))
		}

		query += " WHERE uuid = ?"
		args = append(args, id)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui produk"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Produk berhasil diperbarui"})
	}
}

// DeleteProduct deletes a product
func DeleteProduct(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		// Verify ownership
		var product models.Product
		err := db.Get(&product, "SELECT * FROM products WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		if userType != "seller" || product.SellerID == nil || *product.SellerID != userID.(string) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Tidak diizinkan untuk menghapus produk ini"})
			return
		}

		_, err = db.Exec("DELETE FROM products WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus produk"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Produk berhasil dihapus"})
	}
}

// IncrementProductViews increments the view count of a product
func IncrementProductViews(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		_, err := db.Exec("UPDATE products SET views = COALESCE(views, 0) + 1 WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data analitik"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Analitik diperbarui"})
	}
}
