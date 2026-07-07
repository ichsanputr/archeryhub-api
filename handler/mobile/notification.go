package mobile

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type Notification struct {
	ID        int       `db:"id" json:"id"`
	UserID    string    `db:"user_id" json:"user_id"`
	UserRole  string    `db:"user_role" json:"user_role"`
	Type      string    `db:"type" json:"type"`
	Title     string    `db:"title" json:"title"`
	Message   string    `db:"message" json:"message"`
	Link      *string   `db:"link" json:"link"`
	IsRead    bool      `db:"is_read" json:"is_read"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}

func MobileGetNotifications(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		limit := c.DefaultQuery("limit", "20")
		offset := c.DefaultQuery("offset", "0")

		var notifications []Notification
		err := db.Select(&notifications, `
			SELECT id, user_id, user_role, type, title, message, link, is_read, created_at
			FROM notifications
			WHERE user_id = ? AND user_role = ?
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`, userID, userType, limit, offset)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat notifikasi"})
			return
		}

		if notifications == nil {
			notifications = []Notification{}
		}

		c.JSON(http.StatusOK, gin.H{
			"notifications": notifications,
		})
	}
}

func MobileMarkNotificationRead(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		notificationID := c.Param("id")

		result, err := db.Exec(`
			UPDATE notifications SET is_read = 1, updated_at = NOW()
			WHERE id = ? AND user_id = ?
		`, notificationID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui notifikasi"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notifikasi tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Notifikasi ditandai dibaca"})
	}
}

func MobileMarkAllNotificationsRead(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		_, err := db.Exec(`
			UPDATE notifications SET is_read = 1, updated_at = NOW()
			WHERE user_id = ? AND user_role = ? AND is_read = 0
		`, userID, userType)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menandai semua notifikasi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Semua notifikasi ditandai dibaca"})
	}
}

func MobileGetUnreadCount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var count int
		err := db.Get(&count, `
			SELECT COUNT(*) FROM notifications
			WHERE user_id = ? AND user_role = ? AND is_read = 0
		`, userID, userType)

		if err != nil {
			count = 0
		}

		c.JSON(http.StatusOK, UnreadCountResponse{Count: count})
	}
}
