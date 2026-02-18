package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SubmitContactMessage handles the submission of contact messages
func SubmitContactMessage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name    string `json:"name" binding:"required"`
			Email   string `json:"email" binding:"required,email"`
			Subject string `json:"subject" binding:"required"`
			Message string `json:"message" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
			return
		}

		newUUID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO contact_messages (uuid, name, email, subject, message, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, newUUID, req.Name, req.Email, req.Subject, req.Message, time.Now())

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save message", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":      newUUID,
			"message": "Pesan Anda telah berhasil dikirim. Terima kasih!",
		})
	}
}
