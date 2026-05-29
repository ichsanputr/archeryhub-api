package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func GetCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}})
	}
}

func AddToCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur keranjang telah dinonaktifkan"})
	}
}

func UpdateCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur keranjang telah dinonaktifkan"})
	}
}

func DeleteCartItem(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur keranjang telah dinonaktifkan"})
	}
}

func CheckoutCart(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur checkout telah dinonaktifkan"})
	}
}
