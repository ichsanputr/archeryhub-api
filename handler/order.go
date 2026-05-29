package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func GetSellerOrders(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "meta": gin.H{"total": 0, "pages": 1, "page": 1}})
	}
}

func GetSellerStats(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": gin.H{
			"total_revenue": 0.0,
			"total_orders":  0,
			"products_sold": 0,
			"rating":       0.0,
		}})
	}
}

func UpdateOrderStatus(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur pesanan telah dinonaktifkan"})
	}
}

func ExportSellerOrders(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur ekspor pesanan telah dinonaktifkan"})
	}
}

func GetSellerOrderByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pesanan tidak ditemukan"})
	}
}
