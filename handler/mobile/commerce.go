package mobile

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func MobileArcherGetCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
	}
}

func MobileArcherAddToCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur keranjang telah dinonaktifkan"})
	}
}

func MobileArcherUpdateCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur keranjang telah dinonaktifkan"})
	}
}

func MobileArcherRemoveFromCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur keranjang telah dinonaktifkan"})
	}
}

func MobileArcherClearCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Keranjang dibersihkan"})
	}
}

func MobileArcherCheckoutCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur checkout telah dinonaktifkan"})
	}
}

func MobileArcherGetOrderHistory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
	}
}
