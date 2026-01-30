package handler

import (
	"archeryhub-api/models"
	"net/http"
	"time"

	"archeryhub-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// UpdatePasswordRequest represents the password update request
type UpdatePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// UpdatePassword allows users to set or change their password
func UpdatePassword(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var req UpdatePasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password baru harus minimal 6 karakter"})
			return
		}

		// Validate password length
		if len(req.NewPassword) < 6 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password baru harus minimal 6 karakter"})
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

		// If user has a password, verify the current password (plain text comparison)
		if user.HasPassword && user.Password != nil {
			if req.CurrentPassword == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Password saat ini diperlukan"})
				return
			}

			if *user.Password != req.CurrentPassword {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Password saat ini salah"})
				return
			}
		}

		// Update the password (store as plain text)
		updateQuery := "UPDATE " + table + " SET password = ?, updated_at = NOW() WHERE uuid = ?"
		_, err = db.Exec(updateQuery, req.NewPassword, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "Password berhasil diperbarui",
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
			UUID        string  `json:"uuid" db:"uuid"`
			Email       string  `json:"email" db:"email"`
			FullName    *string `json:"full_name" db:"full_name"`
			UserType    string  `json:"user_type" db:"user_type"`
			AvatarURL   *string `json:"avatar_url" db:"avatar_url"`
			LogoURL     *string `json:"logo_url" db:"logo_url"`
			HasPassword bool    `json:"has_password" db:"has_password"`
		}

		logoField := "NULL as logo_url"
		if userType == "club" || userType == "organization" {
			logoField = "avatar_url as logo_url"
		}

		roleField := "'" + userType.(string) + "'"

		query := `
			SELECT uuid, email, ` + nameField + ` as full_name, ` + roleField + ` as user_type, avatar_url, ` + logoField + `,
				CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password
			FROM ` + table + ` WHERE uuid = ?
		`
		err := db.Get(&user, query, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Mask URLs
		if user.AvatarURL != nil {
			masked := utils.MaskMediaURL(*user.AvatarURL)
			user.AvatarURL = &masked
		}
		if user.LogoURL != nil {
			masked := utils.MaskMediaURL(*user.LogoURL)
			user.LogoURL = &masked
		}

		c.JSON(http.StatusOK, user)
	}
}

// UpdateUserProfile handles profile updates for different user types
func UpdateUserProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType == "club" {
			var req models.UpdateClubRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
				return
			}

			query := "UPDATE clubs SET updated_at = NOW()"
			args := []interface{}{}

			if req.Name != nil {
				query += ", name = ?"
				args = append(args, *req.Name)
			}
			if req.Abbreviation != nil {
				query += ", abbreviation = ?"
				args = append(args, *req.Abbreviation)
			}
			if req.Description != nil {
				query += ", description = ?"
				args = append(args, *req.Description)
			}
			if req.Address != nil {
				query += ", address = ?"
				args = append(args, *req.Address)
			}
			if req.City != nil {
				query += ", city = ?"
				args = append(args, *req.City)
			}
			if req.Province != nil {
				query += ", province = ?"
				args = append(args, *req.Province)
			}
			if req.Phone != nil {
				query += ", phone = ?"
				args = append(args, *req.Phone)
			}
			if req.Email != nil {
				query += ", email = ?"
				args = append(args, *req.Email)
			}
			if req.Website != nil {
				query += ", website = ?"
				args = append(args, *req.Website)
			}
			if req.SocialFacebook != nil {
				query += ", social_facebook = ?"
				args = append(args, *req.SocialFacebook)
			}
			if req.SocialInstagram != nil {
				query += ", social_instagram = ?"
				args = append(args, *req.SocialInstagram)
			}
			if req.TrainingSchedule != nil {
				query += ", training_schedule = ?"
				args = append(args, *req.TrainingSchedule)
			}
			if req.HeadCoachName != nil {
				query += ", head_coach_name = ?"
				args = append(args, *req.HeadCoachName)
			}
			if req.HeadCoachPhone != nil {
				query += ", head_coach_phone = ?"
				args = append(args, *req.HeadCoachPhone)
			}
			if req.AvatarURL != nil {
				query += ", avatar_url = ?"
				args = append(args, utils.ExtractFilename(*req.AvatarURL))
			}
			if req.BannerURL != nil {
				query += ", banner_url = ?"
				args = append(args, utils.ExtractFilename(*req.BannerURL))
			}
			if req.EstablishedDate != nil && *req.EstablishedDate != "" {
				// Try to parse year from date string
				t, err := time.Parse("2006-01-02", *req.EstablishedDate)
				if err == nil {
					year := t.Year()
					query += ", established_year = ?"
					args = append(args, year)
				}
			}

			if len(args) == 0 {
				c.JSON(http.StatusOK, gin.H{"message": "No changes to save"})
				return
			}

			query += " WHERE uuid = ?"
			args = append(args, userID)

			_, err := db.Exec(query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update club profile: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Profil klub berhasil diperbarui"})
			return
		}

		if userType == "seller" {
			var req models.UpdateSellerRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
				return
			}

			query := "UPDATE sellers SET updated_at = NOW()"
			args := []interface{}{}

			if req.StoreName != nil {
				query += ", store_name = ?"
				args = append(args, *req.StoreName)
			}
			if req.Slug != nil {
				query += ", slug = ?"
				args = append(args, *req.Slug)
			}
			if req.Description != nil {
				query += ", description = ?"
				args = append(args, *req.Description)
			}
			if req.Phone != nil {
				query += ", phone = ?"
				args = append(args, *req.Phone)
			}
			if req.Email != nil {
				query += ", email = ?"
				args = append(args, *req.Email)
			}
			if req.Address != nil {
				query += ", address = ?"
				args = append(args, *req.Address)
			}
			if req.City != nil {
				query += ", city = ?"
				args = append(args, *req.City)
			}
			if req.Province != nil {
				query += ", province = ?"
				args = append(args, *req.Province)
			}
			if req.AvatarURL != nil {
				query += ", avatar_url = ?"
				args = append(args, utils.ExtractFilename(*req.AvatarURL))
			}
			if req.BannerURL != nil {
				query += ", banner_url = ?"
				args = append(args, utils.ExtractFilename(*req.BannerURL))
			}

			if len(args) == 0 {
				c.JSON(http.StatusOK, gin.H{"message": "No changes to save"})
				return
			}

			query += " WHERE uuid = ?"
			args = append(args, userID)

			_, err := db.Exec(query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update seller profile: " + err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "Profil toko berhasil diperbarui"})
			return
		}

		// Default to Archer update if not club (or implement others if needed)
		var req models.UpdateArcherRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		table := "archers"
		if userType == "organization" {
			table = "organizations"
		} else if userType == "seller" {
			table = "sellers"
		}

		query := "UPDATE " + table + " SET updated_at = NOW()"
		args := []interface{}{}

		if req.FullName != nil {
			field := "full_name"
			if userType == "organization" {
				field = "name"
			} else if userType == "seller" {
				field = "store_name"
			}
			query += ", " + field + " = ?"
			args = append(args, *req.FullName)
		}
		if req.Phone != nil {
			query += ", phone = ?"
			args = append(args, *req.Phone)
		}
		if req.Address != nil {
			query += ", address = ?"
			args = append(args, *req.Address)
		}
		if req.Bio != nil {
			query += ", bio = ?"
			args = append(args, *req.Bio)
		}
		if req.Achievements != nil {
			query += ", achievements = ?"
			args = append(args, *req.Achievements)
		}
		if req.ClubID != nil {
			query += ", club_id = ?"
			args = append(args, *req.ClubID)
		}
		if req.City != nil {
			query += ", city = ?"
			args = append(args, *req.City)
		}
		if req.School != nil {
			query += ", school = ?"
			args = append(args, *req.School)
		}
		if req.Province != nil {
			query += ", province = ?"
			args = append(args, *req.Province)
		}

		if len(args) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "No changes to save"})
			return
		}

		query += " WHERE uuid = ?"
		args = append(args, userID)

		_, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
	}
}
