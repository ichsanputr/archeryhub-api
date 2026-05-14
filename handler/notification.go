package handler

import (
	"Archeris-api/models"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"net/http"
	"strconv"
)

// GetNotifications returns notifications for the current user
func GetNotifications(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		claims := user.(map[string]interface{})
		userID := claims["user_id"].(string)
		userRole := claims["role"].(string)

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		unreadOnly := c.DefaultQuery("unread_only", "false") == "true"

		if limit > 500 {
			limit = 500
		}

		query := `SELECT id, user_id, user_role, type, title, message, link, is_read, created_at, updated_at
				  FROM notifications
				  WHERE user_id = ? AND user_role = ?`
		
		if unreadOnly {
			query += " AND is_read = FALSE"
		}
		
		query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"

		var notifications []models.Notification
		err := db.Select(&notifications, query, userID, userRole, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil notifikasi"})
			return
		}

		var unreadCount int
		_ = db.Get(&unreadCount, "SELECT COUNT(*) FROM notifications WHERE user_id = ? AND user_role = ? AND is_read = FALSE", userID, userRole)

		var total int
		_ = db.Get(&total, "SELECT COUNT(*) FROM notifications WHERE user_id = ? AND user_role = ?", userID, userRole)

		c.JSON(http.StatusOK, models.NotificationListResponse{
			Notifications: notifications,
			UnreadCount:   unreadCount,
			Total:         total,
		})
	}
}

// MarkNotificationAsRead marks a specific notification as read
func MarkNotificationAsRead(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		claims := user.(map[string]interface{})
		userID := claims["user_id"].(string)
		userRole := claims["role"].(string)
		notificationID := c.Param("id")

		result, err := db.Exec(
			"UPDATE notifications SET is_read = TRUE WHERE id = ? AND user_id = ? AND user_role = ?",
			notificationID, userID, userRole,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui notifikasi"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notifikasi tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Notifikasi ditandai sebagai sudah dibaca"})
	}
}

// MarkAllNotificationsAsRead marks all user notifications as read
func MarkAllNotificationsAsRead(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		claims := user.(map[string]interface{})
		userID := claims["user_id"].(string)
		userRole := claims["role"].(string)

		result, err := db.Exec(
			"UPDATE notifications SET is_read = TRUE WHERE user_id = ? AND user_role = ? AND is_read = FALSE",
			userID, userRole,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui notifikasi"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		c.JSON(http.StatusOK, gin.H{
			"message": "Semua notifikasi ditandai sebagai sudah dibaca",
			"count":   rowsAffected,
		})
	}
}

// CreateNotification creates a new notification (admin only)
func CreateNotification(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user, exists := c.Get("user")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		claims := user.(map[string]interface{})
		role := claims["role"].(string)

		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya admin yang dapat membuat notifikasi"})
			return
		}

		var req models.CreateNotificationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.Type == "" {
			req.Type = "default"
		}

		_, err := db.Exec(
			`INSERT INTO notifications (user_id, user_role, type, title, message, link)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			req.UserID, req.UserRole, req.Type, req.Title, req.Message, req.Link,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat notifikasi"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Notifikasi dibuat"})
	}
}

