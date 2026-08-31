package mobile

import (
	"net/http"
	"strconv"

	"Archeris-api/models"
	"Archeris-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type MobileSellerSummary struct {
	ID         string  `json:"id" db:"uuid"`
	StoreName  string  `json:"store_name" db:"store_name"`
	Slug       string  `json:"slug" db:"slug"`
	AvatarURL  *string `json:"avatar_url" db:"avatar_url"`
	City       *string `json:"city" db:"city"`
	Province   *string `json:"province" db:"province"`
	IsVerified bool    `json:"is_verified" db:"is_verified"`
	Rating     float64 `json:"rating" db:"rating"`
}

type MobileEnrichedProduct struct {
	models.Product
	Seller *MobileSellerSummary `json:"seller,omitempty"`
}

// MobileMarketplaceListProducts returns all active marketplace products for mobile
// @Summary List Marketplace Products
// @Description Get a list of active products with optional search and category filters
// @Tags Mobile - Marketplace
// @Produce json
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Param category query string false "Filter by category"
// @Param search query string false "Search query"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/marketplace/products [get]
func MobileMarketplaceListProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
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

		if category != "" && category != "all" && category != "Semua" {
			query += " AND category = ?"
			args = append(args, category)
		}

		if search != "" {
			query += " AND (name LIKE ? OR description LIKE ?)"
			term := "%" + search + "%"
			args = append(args, term, term)
		}

		query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
		args = append(args, limit, offset)

		var products []models.Product
		err := db.Select(&products, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk", "details": err.Error()})
			return
		}

		var enrichedList []MobileEnrichedProduct
		for i := range products {
			if products[i].ImageURL != nil && *products[i].ImageURL != "" {
				masked := utils.MaskMediaURL(*products[i].ImageURL)
				products[i].ImageURL = &masked
			}

			item := MobileEnrichedProduct{Product: products[i]}
			if products[i].SellerID != nil && *products[i].SellerID != "" {
				var s MobileSellerSummary
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
			"data":     enrichedList,
			"products": enrichedList,
			"meta": gin.H{
				"total": len(enrichedList),
				"limit": limit,
				"page":  (offset / limit) + 1,
			},
		})
	}
}

// MobileMarketplaceGetProductDetail returns single product detail by UUID or slug
// @Summary Get Product Detail
// @Description Get detailed info of a product for mobile app
// @Tags Mobile - Marketplace
// @Produce json
// @Param id path string true "Product ID or Slug"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/marketplace/products/{id} [get]
func MobileMarketplaceGetProductDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idOrSlug := c.Param("id")

		var p models.Product
		err := db.Get(&p, `
			SELECT 
				uuid, seller_id, name, slug, description, price, sale_price,
				category, stock, status, COALESCE(image_url, '') as image_url,
				images, colors, specifications, COALESCE(allowed_payment_methods, '["automatic","wallet","manual"]') as allowed_payment_methods,
				views, created_at, updated_at
			FROM products 
			WHERE uuid = ? OR slug = ? 
			LIMIT 1
		`, idOrSlug, idOrSlug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
			return
		}

		// Increment view count asynchronously
		go func(productUUID string) {
			_, _ = db.Exec("UPDATE products SET views = views + 1 WHERE uuid = ?", productUUID)
		}(p.UUID)

		if p.ImageURL != nil && *p.ImageURL != "" {
			masked := utils.MaskMediaURL(*p.ImageURL)
			p.ImageURL = &masked
		}

		var seller *MobileSellerSummary
		if p.SellerID != nil && *p.SellerID != "" {
			var s MobileSellerSummary
			sErr := db.Get(&s, `
				SELECT uuid, store_name, slug, avatar_url, city, province, is_verified, rating 
				FROM sellers 
				WHERE uuid = ? OR user_id = ? OR google_id = ? 
				LIMIT 1
			`, *p.SellerID, *p.SellerID, *p.SellerID)
			if sErr == nil && s.StoreName != "" {
				if s.AvatarURL != nil && *s.AvatarURL != "" {
					maskedAvatar := utils.MaskMediaURL(*s.AvatarURL)
					s.AvatarURL = &maskedAvatar
				}
				seller = &s
			}
		}

		enriched := MobileEnrichedProduct{
			Product: p,
			Seller:  seller,
		}

		c.JSON(http.StatusOK, gin.H{
			"data":    enriched,
			"product": enriched,
		})
	}
}
