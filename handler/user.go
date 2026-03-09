package handler

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		case "seller":
			table = "sellers"
			nameField = "store_name"
		}

		var user struct {
			UUID            string  `json:"uuid" db:"uuid"`
			Email           string  `json:"email" db:"email"`
			FullName        *string `json:"full_name" db:"full_name"`
			Username        *string `json:"username" db:"username"`
			UserType        string  `json:"user_type" db:"user_type"`
			AvatarURL       *string `json:"avatar_url" db:"avatar_url"`
			LogoURL         *string `json:"logo_url" db:"logo_url"`
			HasPassword     bool    `json:"has_password" db:"has_password"`
			GoogleID        *string `json:"google_id" db:"google_id"`
			ClubID          *string `json:"club_id" db:"club_id"`
			Phone           *string `json:"phone" db:"phone"`
			City            *string `json:"city" db:"city"`
			Address         *string `json:"address" db:"address"`
			Bio             *string `json:"bio" db:"bio"`
			School          *string `json:"school" db:"school"`
			SocialInstagram *string `json:"social_instagram" db:"social_instagram"`
			SocialTiktok    *string `json:"social_tiktok" db:"social_tiktok"`
			SocialWhatsapp  *string `json:"social_whatsapp" db:"social_whatsapp"`
		}

		nameField = nameField // Already set above
		
		var selectFields string
		if userType == "archer" {
			selectFields = `uuid, email, full_name, username, 'archer' as user_type, avatar_url, NULL as logo_url,
				CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password,
				google_id, club_id, phone, city, address, bio, school, social_instagram, social_tiktok, social_whatsapp`
		} else if userType == "organization" {
			selectFields = `uuid, email, name as full_name, slug as username, 'organization' as user_type, avatar_url, avatar_url as logo_url,
				CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password,
				google_id, NULL as club_id, whatsapp_no as phone, city, address, description as bio, NULL as school, NULL as social_instagram, NULL as social_tiktok, NULL as social_whatsapp`
		} else {
			// seller
			selectFields = `uuid, email, store_name as full_name, slug as username, 'seller' as user_type, avatar_url, NULL as logo_url,
				CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password,
				google_id, NULL as club_id, phone, city, address, description as bio, NULL as school, NULL as social_instagram, NULL as social_tiktok, NULL as social_whatsapp`
		}

		query := `SELECT ` + selectFields + ` FROM ` + table + ` WHERE uuid = ?`
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
				un := utils.CleanUsername(*req.Slug)
				if un != "" {
					query += ", slug = ?"
					args = append(args, un)
				}
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
		if req.Username != nil {
			un := utils.CleanUsername(*req.Username)
			if un != "" {
				field := "username"
				if userType == "organization" || userType == "seller" {
					field = "slug"
				}
				query += ", " + field + " = ?"
				args = append(args, un)
			}
		}
		if req.Phone != nil {
			field := "phone"
			if userType == "organization" {
				field = "whatsapp_no"
			}
			query += ", " + field + " = ?"
			args = append(args, *req.Phone)
		}
		if req.Address != nil {
			query += ", address = ?"
			args = append(args, *req.Address)
		}
		if req.Bio != nil {
			field := "bio"
			if userType == "organization" || userType == "seller" {
				field = "description"
			}
			query += ", " + field + " = ?"
			args = append(args, *req.Bio)
		}
		if req.ClubID != nil && userType == "archer" {
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
		if req.SocialInstagram != nil {
			query += ", social_instagram = ?"
			args = append(args, *req.SocialInstagram)
		}
		if req.SocialTiktok != nil {
			query += ", social_tiktok = ?"
			args = append(args, *req.SocialTiktok)
		}
		if req.SocialWhatsapp != nil {
			query += ", social_whatsapp = ?"
			args = append(args, *req.SocialWhatsapp)
		}
		if req.Achievements != nil {
			query += ", achievements = ?"
			args = append(args, *req.Achievements)
		}
		if req.Equipment != nil {
			query += ", equipment = ?"
			args = append(args, *req.Equipment)
		}
		if req.AvatarURL != nil {
			query += ", avatar_url = ?"
			args = append(args, utils.ExtractFilename(*req.AvatarURL))
		}
		if req.BannerURL != nil {
			query += ", banner_url = ?"
			args = append(args, utils.ExtractFilename(*req.BannerURL))
		}
		if req.PageSettings != nil {
			query += ", page_settings = ?"
			args = append(args, *req.PageSettings)
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

// RequestEmailChange sends an OTP to the new email address
func RequestEmailChange(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var req struct {
			NewEmail string `json:"new_email" binding:"required,email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email tidak valid"})
			return
		}

		// Check if new email already exists in any table (archers, organizations, clubs, sellers)
		var exists bool
		tables := []string{"archers", "organizations", "sellers"}
		for _, t := range tables {
			err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM "+t+" WHERE email = ?)", req.NewEmail)
			if err == nil && exists {
				c.JSON(http.StatusConflict, gin.H{"error": "Email sudah digunakan oleh akun lain"})
				return
			}
		}

		// Get old email
		var oldEmail string
		table := "archers"
		switch userType {
		case "organization": table = "organizations"
		case "seller": table = "sellers"
		}
		
		err := db.Get(&oldEmail, "SELECT email FROM "+table+" WHERE uuid = ?", userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
			return
		}

		// Generate OTP
		otp := utils.GenerateOTP()
		otpUUID := uuid.New().String()
		expiresAt := time.Now().Add(15 * time.Minute)

		// Save to database
		_, err = db.Exec(`
			INSERT INTO email_otps (uuid, user_id, user_type, old_email, new_email, otp_code, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, otpUUID, userID, userType, oldEmail, req.NewEmail, otp, expiresAt)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan OTP: " + err.Error()})
			return
		}

		// Send Email
		subject := "Kode OTP Verifikasi Perubahan Email - ArcheryHub"
		body := fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #eee; border-radius: 10px;">
				<h2 style="color: #0ea5e9; text-align: center;">Verifikasi Email Baru Anda</h2>
				<p>Halo,</p>
				<p>Kami menerima permintaan untuk mengubah alamat email akun ArcheryHub Anda ke <strong>%s</strong>.</p>
				<p>Silakan gunakan kode OTP di bawah ini untuk memverifikasi perubahan ini:</p>
				<div style="background: #f0f9ff; padding: 20px; text-align: center; border-radius: 8px; margin: 20px 0;">
					<span style="font-size: 32px; font-weight: 900; letter-spacing: 5px; color: #0369a1;">%s</span>
				</div>
				<p style="color: #666; font-size: 14px;">Kode ini akan kedaluwarsa dalam 15 menit. Jika Anda tidak merasa melakukan permintaan ini, silakan abaikan email ini.</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;">
				<p style="text-align: center; color: #999; font-size: 12px;">&copy; 2026 ArcheryHub.id - Platform Panahan Indonesia</p>
			</div>
		`, req.NewEmail, otp)

		err = utils.SendEmail(req.NewEmail, subject, body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim email verifikasi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Kode OTP telah dikirim ke email baru Anda"})
	}
}

// VerifyEmailChange verifies the OTP and updates the email
func VerifyEmailChange(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		
		var req struct {
			NewEmail string `json:"new_email" binding:"required,email"`
			OTP      string `json:"otp" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid"})
			return
		}

		// Find valid OTP
		var otpRecord struct {
			UUID     string `db:"uuid"`
			UserType string `db:"user_type"`
		}
		
		err := db.Get(&otpRecord, `
			SELECT uuid, user_type FROM email_otps 
			WHERE user_id = ? AND new_email = ? AND otp_code = ? AND is_used = false AND expires_at > NOW()
			ORDER BY created_at DESC LIMIT 1
		`, userID, req.NewEmail, req.OTP)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode OTP salah atau sudah kedaluwarsa"})
			return
		}

		// Begin transaction
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
		defer tx.Rollback()

		// Mark OTP as used
		_, err = tx.Exec("UPDATE email_otps SET is_used = true WHERE uuid = ?", otpRecord.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status OTP"})
			return
		}

		// Update Email in relevant table
		table := "archers"
		switch otpRecord.UserType {
		case "organization": table = "organizations"
		case "seller": table = "sellers"
		}

		_, err = tx.Exec("UPDATE "+table+" SET email = ?, updated_at = NOW() WHERE uuid = ?", req.NewEmail, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui email"})
			return
		}

		err = tx.Commit()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		// Log activity
		utils.LogActivity(db, userID.(string), "", "email_changed", otpRecord.UserType, userID.(string), "User changed email to: "+req.NewEmail, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Email berhasil diperbarui. Silakan gunakan email baru untuk login berikutnya."})
	}
}
