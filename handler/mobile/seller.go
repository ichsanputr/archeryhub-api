package mobile

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func MobileGetSellerDashboard(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"total_revenue": 0.0,
			"total_orders":  0,
			"products_sold": 0,
			"rating":       0.0,
		}})
	}
}

func MobileGetSellerProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "meta": gin.H{"total": 0, "pages": 1, "page": 1}})
	}
}

func MobileCreateProduct(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur produk telah dinonaktifkan"})
	}
}

func MobileUpdateProduct(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur produk telah dinonaktifkan"})
	}
}

func MobileDeleteProduct(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur produk telah dinonaktifkan"})
	}
}
