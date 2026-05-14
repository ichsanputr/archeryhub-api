package mobile

import (
	"Archeris-api/models"
	"Archeris-api/utils"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileMarketplaceListProducts returns marketplace products
// @Summary List Marketplace Products
// @Description Get a list of active products in the marketplace
// @Tags Mobile - Marketplace
// @Produce json
// @Param limit query int false "Pagination limit"
// @Param page query int false "Pagination page"
// @Param search query string false "Search by name or description"
// @Param category query string false "Filter by category"
// @Success 200 {object} MobileMarketplaceProductsResponse
// @Router /mobile/marketplace/products [get]
func MobileMarketplaceListProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		if page < 1 {
			page = 1
		}
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", strconv.Itoa((page-1)*limit)))
		search := c.Query("search")
		category := c.Query("category")

		whereClause := "WHERE status = 'active'"
		args := []interface{}{}

		if search != "" {
			whereClause += " AND (name LIKE ? OR description LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm)
		}
		if category != "" {
			whereClause += " AND category = ?"
			args = append(args, category)
		}

		// Count total for pagination
		var totalCount int
		countQuery := "SELECT COUNT(*) FROM products " + whereClause
		if err := db.Get(&totalCount, countQuery, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data produk", "details": err.Error()})
			return
		}

		query := "SELECT * FROM products " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
		queryArgs := append(args, limit, offset)

		var products []models.Product
		if err := db.Select(&products, query, queryArgs...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk", "details": err.Error()})
			return
		}

		if products == nil {
			products = []models.Product{}
		}

		for i := range products {
			maskProductMedia(&products[i])
		}

		lastPage := (totalCount + limit - 1) / limit
		if limit <= 0 {
			lastPage = 1
		}

		c.JSON(http.StatusOK, MobileMarketplaceProductsResponse{
			Products:    products,
			TotalCount:  totalCount,
			Limit:       limit,
			Offset:      offset,
			CurrentPage: page,
			LastPage:    lastPage,
		})
	}
}

// MobileMarketplaceGetProductDetail returns product details
// @Summary Get Product Detail
// @Description Get detailed information about a specific product
// @Tags Mobile - Marketplace
// @Produce json
// @Param id path string true "Product Slug or UUID"
// @Success 200 {object} MobileMarketplaceProductResponse
// @Failure 404 {object} map[string]interface{}
// @Router /mobile/marketplace/products/{id} [get]
func MobileMarketplaceGetProductDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var product models.Product
		err := db.Get(&product, "SELECT * FROM products WHERE status = 'active' AND (uuid = ? OR slug = ?) LIMIT 1", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		_, _ = db.Exec("UPDATE products SET views = COALESCE(views, 0) + 1 WHERE uuid = ?", product.UUID)

		maskProductMedia(&product)

		c.JSON(http.StatusOK, MobileMarketplaceProductResponse{Product: product})
	}
}

func maskProductMedia(product *models.Product) {
	if product.ImageURL != nil && *product.ImageURL != "" {
		masked := utils.MaskMediaURL(*product.ImageURL)
		product.ImageURL = &masked
	}

	if product.Images != nil && *product.Images != "" {
		var images []string
		_ = json.Unmarshal([]byte(*product.Images), &images)
		for i, image := range images {
			images[i] = utils.MaskMediaURL(image)
		}
		maskedJSON, _ := json.Marshal(images)
		maskedStr := string(maskedJSON)
		product.Images = &maskedStr
	}
}

