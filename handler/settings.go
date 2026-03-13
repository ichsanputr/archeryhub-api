package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type UserSettings struct {
	UserUUID       string `json:"user_uuid" db:"user_uuid"`
	DashboardTheme string `json:"dashboard_theme" db:"dashboard_theme"`
}

func GetUserSettings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var settings UserSettings
		err := db.Get(&settings, "SELECT user_uuid, dashboard_theme FROM user_settings WHERE user_uuid = ?", userID)
		if err != nil {
			if err == sql.ErrNoRows {
				// Return default if not found
				c.JSON(http.StatusOK, UserSettings{
					UserUUID:       userID.(string),
					DashboardTheme: "default",
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil pengaturan"})
			return
		}

		c.JSON(http.StatusOK, settings)
	}
}

func UpdateUserSettings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			DashboardTheme string `json:"dashboard_theme" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid"})
			return
		}

		_, err := db.Exec(`
			INSERT INTO user_settings (user_uuid, dashboard_theme) 
			VALUES (?, ?) 
			ON DUPLICATE KEY UPDATE dashboard_theme = ?, updated_at = NOW()
		`, userID, req.DashboardTheme, req.DashboardTheme)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui pengaturan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Pengaturan berhasil diperbarui", "dashboard_theme": req.DashboardTheme})
	}
}
