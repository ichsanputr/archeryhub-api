package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var (
	contactAttempts sync.Map
	contactRateLimit  = 5
	contactRateWindow = 10 * time.Minute
)

type contactTracker struct {
	count     int
	firstTime time.Time
}

func checkContactRateLimit(ip string) bool {
	now := time.Now()
	val, loaded := contactAttempts.Load(ip)
	if !loaded {
		contactAttempts.Store(ip, &contactTracker{count: 1, firstTime: now})
		return true
	}
	tr := val.(*contactTracker)
	if now.Sub(tr.firstTime) > contactRateWindow {
		tr.count = 1
		tr.firstTime = now
		return true
	}
	if tr.count >= contactRateLimit {
		return false
	}
	tr.count++
	return true
}

// SubmitContactMessage handles the submission of contact messages
func SubmitContactMessage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !checkContactRateLimit(c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Terlalu banyak pesan dikirim. Silakan coba lagi dalam 10 menit.",
			})
			return
		}

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
