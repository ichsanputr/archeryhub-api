package handler

import (
	"Archeris-api/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetNotifications returns notifications for the current user
func GetNotifications(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists || userIDVal == nil || userIDVal.(string) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}
		userID := userIDVal.(string)
		orgIDVal, _ := c.Get("org_id")
		orgID, _ := orgIDVal.(string)
		if orgID == "" {
			orgID = userID
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		unreadOnly := c.DefaultQuery("unread_only", "false") == "true"

		if limit > 500 {
			limit = 500
		}

		query := `SELECT id, user_id, user_role, type, title, message, link, is_read, created_at, updated_at
				  FROM notifications
				  WHERE (user_id = ? OR user_id = ?)`

		if unreadOnly {
			query += " AND is_read = FALSE"
		}

		query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"

		var notifications []models.Notification
		err := db.Select(&notifications, query, userID, orgID, limit, offset)
		if err != nil {
			notifications = []models.Notification{}
		}

		var unreadCount int
		_ = db.Get(&unreadCount, "SELECT COUNT(*) FROM notifications WHERE (user_id = ? OR user_id = ?) AND is_read = FALSE", userID, orgID)

		var total int
		_ = db.Get(&total, "SELECT COUNT(*) FROM notifications WHERE (user_id = ? OR user_id = ?)", userID, orgID)

		c.JSON(http.StatusOK, models.NotificationListResponse{
			Notifications: notifications,
			UnreadCount:   unreadCount,
			Total:         total,
		})
	}
}

// GetUnreadNotificationCount returns the count of unread notifications for current user
func GetUnreadNotificationCount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists || userIDVal == nil || userIDVal.(string) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}
		userID := userIDVal.(string)
		orgIDVal, _ := c.Get("org_id")
		orgID, _ := orgIDVal.(string)
		if orgID == "" {
			orgID = userID
		}

		var count int
		err := db.Get(&count, "SELECT COUNT(*) FROM notifications WHERE (user_id = ? OR user_id = ?) AND is_read = FALSE", userID, orgID)
		if err != nil {
			count = 0
		}

		c.JSON(http.StatusOK, gin.H{
			"count":        count,
			"unread_count": count,
		})
	}
}

// MarkNotificationAsRead marks a specific notification as read
func MarkNotificationAsRead(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists || userIDVal == nil || userIDVal.(string) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}
		userID := userIDVal.(string)
		orgIDVal, _ := c.Get("org_id")
		orgID, _ := orgIDVal.(string)
		if orgID == "" {
			orgID = userID
		}
		notificationID := c.Param("id")

		result, err := db.Exec(
			"UPDATE notifications SET is_read = TRUE WHERE id = ? AND (user_id = ? OR user_id = ?)",
			notificationID, userID, orgID,
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
		userIDVal, exists := c.Get("user_id")
		if !exists || userIDVal == nil || userIDVal.(string) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}
		userID := userIDVal.(string)
		orgIDVal, _ := c.Get("org_id")
		orgID, _ := orgIDVal.(string)
		if orgID == "" {
			orgID = userID
		}

		result, err := db.Exec(
			"UPDATE notifications SET is_read = TRUE WHERE (user_id = ? OR user_id = ?) AND is_read = FALSE",
			userID, orgID,
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

// DeleteNotification deletes a specific notification
func DeleteNotification(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists || userIDVal == nil || userIDVal.(string) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}
		userID := userIDVal.(string)
		orgIDVal, _ := c.Get("org_id")
		orgID, _ := orgIDVal.(string)
		if orgID == "" {
			orgID = userID
		}
		notificationID := c.Param("id")

		result, err := db.Exec(
			"DELETE FROM notifications WHERE id = ? AND (user_id = ? OR user_id = ?)",
			notificationID, userID, orgID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus notifikasi"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Notifikasi tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Notifikasi berhasil dihapus"})
	}
}

// DeleteAllNotifications deletes all notifications for current user
func DeleteAllNotifications(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDVal, exists := c.Get("user_id")
		if !exists || userIDVal == nil || userIDVal.(string) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}
		userID := userIDVal.(string)
		orgIDVal, _ := c.Get("org_id")
		orgID, _ := orgIDVal.(string)
		if orgID == "" {
			orgID = userID
		}

		result, err := db.Exec(
			"DELETE FROM notifications WHERE (user_id = ? OR user_id = ?)",
			userID, orgID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus notifikasi"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		c.JSON(http.StatusOK, gin.H{
			"message": "Semua notifikasi berhasil dihapus",
			"count":   rowsAffected,
		})
	}
}

// CreateNotification creates a new notification
func CreateNotification(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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

