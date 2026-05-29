package mobile

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func MobileMarketplaceListProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "meta": gin.H{"total": 0, "pages": 1, "page": 1}})
	}
}

func MobileMarketplaceGetProductDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Produk tidak ditemukan"})
	}
}
