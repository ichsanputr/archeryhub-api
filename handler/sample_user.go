package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetSampleUser returns a sample user for development auto-fill
func GetSampleUser(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var email string
		err := db.Get(&email, "SELECT email FROM archers LIMIT 1")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sample user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"email":    email,
			"password": "password123", // Default for dev seeding
		})
	}
}
