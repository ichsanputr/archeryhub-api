package mobile

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileMarketplaceListProducts godoc
// @Summary      List marketplace products for mobile
// @Description  Returns active products for mobile marketplace read-only pages
// @Tags         Mobile - Marketplace
// @Produce      json
// @Param        limit     query     int     false  "Limit"
// @Param        offset    query     int     false  "Offset"
// @Param        search    query     string  false  "Search term"
// @Param        category  query     string  false  "Product category"
// @Success      200       {object}  MobileMarketplaceProductsResponse
// @Failure      500       {object}  ErrorResponse
// @Router       /mobile/marketplace/products [get]
func MobileMarketplaceListProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
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

		query := "SELECT * FROM products " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
		args = append(args, limit, offset)

		var products []models.Product
		if err := db.Select(&products, query, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk", "details": err.Error()})
			return
		}

		if products == nil {
			products = []models.Product{}
		}

		for i := range products {
			maskProductMedia(&products[i])
		}

		c.JSON(http.StatusOK, gin.H{
			"products":    products,
			"total_count": len(products),
		})
	}
}

// MobileMarketplaceGetProductDetail godoc
// @Summary      Get marketplace product detail for mobile
// @Description  Returns one product by UUID or slug for mobile marketplace read-only pages
// @Tags         Mobile - Marketplace
// @Produce      json
// @Param        id  path      string  true  "Product UUID or slug"
// @Success      200 {object}  MobileMarketplaceProductResponse
// @Failure      404 {object}  ErrorResponse
// @Router       /mobile/marketplace/products/{id} [get]
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

		c.JSON(http.StatusOK, gin.H{"product": product})
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
