package handler

import (
	"archeryhub-api/models"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetProducts returns all products (public)
func GetProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var products []models.Product
		err := db.Select(&products, "SELECT * FROM products WHERE status = 'active' ORDER BY created_at DESC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
			return
		}

		if products == nil {
			products = []models.Product{}
		}

		c.JSON(http.StatusOK, gin.H{"data": products})
	}
}

// GetMyProducts returns products owned by the current user's organization/club
func GetMyProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var products []models.Product
		var err error

		if userType == "organization" {
			err = db.Select(&products, "SELECT * FROM products WHERE organization_id = ? ORDER BY created_at DESC", userID)
		} else if userType == "club" {
			err = db.Select(&products, "SELECT * FROM products WHERE club_id = ? ORDER BY created_at DESC", userID)
		} else {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to view products"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
			return
		}

		if products == nil {
			products = []models.Product{}
		}

		c.JSON(http.StatusOK, gin.H{"data": products})
	}
}

// GetProductByID returns a single product
func GetProductByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var product models.Product
		err := db.Get(&product, "SELECT * FROM products WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		// Increment views
		db.Exec("UPDATE products SET views = views + 1 WHERE uuid = ?", product.UUID)

		c.JSON(http.StatusOK, gin.H{"data": product})
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

		imagesJSON, _ := json.Marshal(req.Images)
		specJSON, _ := json.Marshal(req.Specifications)

		var orgID, clubID *string
		userIDStr := userID.(string)
		if userType == "organization" {
			orgID = &userIDStr
		} else if userType == "club" {
			clubID = &userIDStr
		}

		if req.Status == "" {
			req.Status = "draft"
		}

		_, err := db.Exec(`
			INSERT INTO products (uuid, organization_id, club_id, name, slug, description, price, sale_price, category, stock, status, image_url, images, specifications)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, productID, orgID, clubID, req.Name, slug, req.Description, req.Price, req.SalePrice, req.Category, req.Stock, req.Status, req.ImageURL, string(imagesJSON), string(specJSON))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create product: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Product created successfully",
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		isOwner := false
		if userType == "organization" && product.OrganizationID != nil && *product.OrganizationID == userID.(string) {
			isOwner = true
		} else if userType == "club" && product.ClubID != nil && *product.ClubID == userID.(string) {
			isOwner = true
		}

		if !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to update this product"})
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
		if req.Specifications != nil {
			specJSON, _ := json.Marshal(req.Specifications)
			query += ", specifications = ?"
			args = append(args, string(specJSON))
		}

		query += " WHERE uuid = ?"
		args = append(args, id)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}

		isOwner := false
		if userType == "organization" && product.OrganizationID != nil && *product.OrganizationID == userID.(string) {
			isOwner = true
		} else if userType == "club" && product.ClubID != nil && *product.ClubID == userID.(string) {
			isOwner = true
		}

		if !isOwner {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to delete this product"})
			return
		}

		_, err = db.Exec("DELETE FROM products WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
	}
}
