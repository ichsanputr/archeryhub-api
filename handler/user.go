package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// UpdatePasswordRequest represents the password update request
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// UpdatePassword allows users to set or change their password
func UpdatePassword(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var req UpdatePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password baru harus minimal 8 karakter"})
			return
		}

		// Determine target table
		table := "archers"
		switch userType {
		case "organization":
			table = "organizations"
		case "club":
			table = "clubs"
		case "seller":
			table = "sellers"
		}

		// Get current user data
		var user struct {
			Password    *string `db:"password"`
			HasPassword bool    `db:"has_password"`
		}
		
		query := "SELECT password, CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password FROM " + table + " WHERE uuid = ?"
		err := db.Get(&user, query, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found in " + table})
			return
		}

		// If user has a password, verify the current password
		if user.HasPassword && user.Password != nil {
			if req.CurrentPassword == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Password saat ini diperlukan"})
				return
			}
			
			err = bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(req.CurrentPassword))
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Password saat ini salah"})
				return
			}
		}

		// Hash the new password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
			return
		}

		// Update the password
		updateQuery := "UPDATE " + table + " SET password = ?, updated_at = NOW() WHERE uuid = ?"
		_, err = db.Exec(updateQuery, string(hashedPassword), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Password berhasil diperbarui",
			"has_password": true,
		})
	}
}

// GetUserProfile returns the current user's profile with has_password flag
func GetUserProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		table := "archers"
		nameField := "full_name"
		switch userType {
		case "organization":
			table = "organizations"
			nameField = "name"
		case "club":
			table = "clubs"
			nameField = "name"
		case "seller":
			table = "sellers"
			nameField = "store_name"
		}

		var user struct {
			UUID         string  `json:"uuid" db:"uuid"`
			Email        string  `json:"email" db:"email"`
			FullName     *string `json:"full_name" db:"full_name"`
			UserType     string  `json:"user_type" db:"user_type"`
			AvatarURL    *string `json:"avatar_url" db:"avatar_url"`
			HasPassword  bool    `json:"has_password" db:"has_password"`
		}

		query := `
			SELECT uuid, email, ` + nameField + ` as full_name, role as user_type, avatar_url,
				CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password
			FROM ` + table + ` WHERE uuid = ?
		`
		err := db.Get(&user, query, userID)
		
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}
