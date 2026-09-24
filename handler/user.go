package handler

import (
	"Archeris-api/models"
	"Archeris-api/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
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

		if userType == "root" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Password root tidak dapat diubah dari panel ini"})
			return
		}

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
		case "organizer":
			table = "organizers"
		}

		// Get current user data
		var user struct {
			Password    *string `db:"password"`
			HasPassword bool    `db:"has_password"`
		}

		query := "SELECT password, CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password FROM " + table + " WHERE uuid = ?"
		err := db.Get(&user, query, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan di " + table})
			return
		}

		// Hash the new password securely with bcrypt
		hashedBytes, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses password"})
			return
		}

		// Update password hash and increment token_version to invalidate old sessions, ensuring account is active & verified
		updateQuery := "UPDATE " + table + " SET password = ?, status = 'active', is_verified = 1, token_version = token_version + 1, updated_at = NOW() WHERE uuid = ?"
		_, err = db.Exec(updateQuery, string(hashedBytes), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui password"})
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

		if userType == "root" {
			c.JSON(http.StatusOK, gin.H{
				"uuid":         "00000000-0000-0000-0000-000000000000",
				"email":        "root",
				"full_name":    "Super Administrator",
				"username":     "root",
				"user_type":    "root",
				"avatar_url":   nil,
				"logo_url":     nil,
				"has_password": true,
				"google_id":    nil,
				"club_id":      nil,
				"phone":        nil,
				"city":         nil,
				"address":      nil,
				"bio":          "Pusat Administrasi Sistem Archeris.id",
				"country":      nil,
			})
			return
		}

		table := "archers"
		switch userType {
		case "organizer":
			table = "organizers"
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
			Country         *string `json:"country" db:"country"`
			SocialInstagram *string `json:"social_instagram" db:"social_instagram"`
			SocialTiktok    *string `json:"social_tiktok" db:"social_tiktok"`
			SocialWhatsapp  *string `json:"social_whatsapp" db:"social_whatsapp"`
			SocialYoutube   *string `json:"social_youtube" db:"social_youtube"`
			SocialSpotify   *string `json:"social_spotify" db:"social_spotify"`
			SocialWebsite   *string `json:"social_website" db:"social_website"`
			SocialPinterest *string `json:"social_pinterest" db:"social_pinterest"`
			SocialLinkedin  *string `json:"social_linkedin" db:"social_linkedin"`
		}
		
		var selectFields string
		if userType == "archer" {
			selectFields = `uuid, email, full_name, username, 'archer' as user_type, avatar_url, NULL as logo_url,
				CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password,
				google_id, club_id, phone, NULL as city, address, bio, country, social_instagram, social_tiktok, social_whatsapp,
				social_youtube, social_spotify, social_website, social_pinterest, social_linkedin`
		} else if userType == "organizer" {
			selectFields = `uuid, email, name as full_name, slug as username, 'organizer' as user_type, avatar_url, avatar_url as logo_url,
				CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password,
				google_id, NULL as club_id, whatsapp_no as phone, city, address, description as bio, country, NULL as social_instagram, NULL as social_tiktok, NULL as social_whatsapp,
				NULL as social_youtube, NULL as social_spotify, NULL as social_website, NULL as social_pinterest, NULL as social_linkedin`
		} else {
			// seller
			selectFields = `uuid, email, store_name as full_name, slug as username, 'seller' as user_type, avatar_url, NULL as logo_url,
				CASE WHEN password IS NOT NULL AND password != '' THEN true ELSE false END as has_password,
				google_id, NULL as club_id, phone, city, address, description as bio, NULL as country, NULL as social_instagram, NULL as social_tiktok, NULL as social_whatsapp,
				NULL as social_youtube, NULL as social_spotify, NULL as social_website, NULL as social_pinterest, NULL as social_linkedin`
		}

		query := `SELECT ` + selectFields + ` FROM ` + table + ` WHERE uuid = ?`
		err := db.Get(&user, query, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
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

		if userType == "root" {
			c.JSON(http.StatusOK, gin.H{"message": "Profil root adalah statis dan tidak dapat diubah"})
			return
		}



		// Default to Archer update if not club (or implement others if needed)
		var req models.UpdateArcherRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid", "details": err.Error()})
			return
		}

		table := "archers"
		if userType == "organizer" {
			table = "organizers"
		}

		query := "UPDATE " + table + " SET updated_at = NOW()"
		args := []interface{}{}

		if req.FullName != nil {
			field := "full_name"
			if userType == "organizer" {
				field = "name"
			}
			query += ", " + field + " = ?"
			args = append(args, *req.FullName)
		}
		if req.Username != nil {
			un := utils.CleanUsername(*req.Username)
			if un != "" {
				// check if username is taken in archers or organizers
				// excluding the current user UUID
				var exists bool
				
				// Check in archers
				err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM archers WHERE username = ? AND uuid != ?)", un, userID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan database: " + err.Error()})
					return
				}
				if exists {
					c.JSON(http.StatusConflict, gin.H{"error": "Username sudah digunakan oleh pemanah lain"})
					return
				}

				// Check in organizers
				err = db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM organizers WHERE slug = ? AND uuid != ?)", un, userID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan database: " + err.Error()})
					return
				}
				if exists {
					c.JSON(http.StatusConflict, gin.H{"error": "Username sudah digunakan oleh organisasi lain"})
					return
				}

				field := "username"
				if userType == "organizer" {
					field = "slug"
				}
				query += ", " + field + " = ?"
				args = append(args, un)
			}
		}
		if req.Phone != nil {
			field := "phone"
			if userType == "organizer" {
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
			if userType == "organizer" {
				field = "description"
			}
			query += ", " + field + " = ?"
			args = append(args, *req.Bio)
		}
		if req.ClubID != nil && userType == "archer" {
			query += ", club_id = ?"
			args = append(args, *req.ClubID)
		}
		if req.DateOfBirth != nil {
			query += ", date_of_birth = ?"
			args = append(args, *req.DateOfBirth)
		}
		if req.Gender != nil {
			query += ", gender = ?"
			args = append(args, *req.Gender)
		}
		if req.HandDominance != nil {
			query += ", hand_dominance = ?"
			args = append(args, *req.HandDominance)
		}
		if req.EmergencyContactName != nil {
			query += ", emergency_contact_name = ?"
			args = append(args, *req.EmergencyContactName)
		}
		if req.City != nil {
			query += ", city = ?"
			args = append(args, *req.City)
		}
		if req.BowType != nil {
			query += ", bow_type = ?"
			args = append(args, *req.BowType)
		}
		if req.Country != nil {
			query += ", country = ?"
			args = append(args, *req.Country)
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
		if req.SocialFacebook != nil {
			query += ", social_facebook = ?"
			args = append(args, *req.SocialFacebook)
		}
		if req.SocialTwitter != nil {
			query += ", social_twitter = ?"
			args = append(args, *req.SocialTwitter)
		}
		if req.SocialYoutube != nil {
			query += ", social_youtube = ?"
			args = append(args, *req.SocialYoutube)
		}
		if req.SocialSpotify != nil {
			query += ", social_spotify = ?"
			args = append(args, *req.SocialSpotify)
		}
		if req.SocialWebsite != nil {
			query += ", social_website = ?"
			args = append(args, *req.SocialWebsite)
		}
		if req.SocialPinterest != nil {
			query += ", social_pinterest = ?"
			args = append(args, *req.SocialPinterest)
		}
		if req.SocialLinkedin != nil {
			query += ", social_linkedin = ?"
			args = append(args, *req.SocialLinkedin)
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
			c.JSON(http.StatusOK, gin.H{"message": "Tidak ada perubahan untuk disimpan"})
			return
		}

		query += " WHERE uuid = ?"
		args = append(args, userID)

		_, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil: " + err.Error()})
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

		// Check if new email already exists in any table (archers, organizers)
		var exists bool
		tables := []string{"archers", "organizers"}
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
		case "organizer": table = "organizers"
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
		err = utils.SendEmailChangeOTPEmail(req.NewEmail, req.NewEmail, otp, 15)
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan database"})
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
		case "organizer": table = "organizers"
		}

		_, err = tx.Exec("UPDATE "+table+" SET email = ?, updated_at = NOW() WHERE uuid = ?", req.NewEmail, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui email"})
			return
		}

		err = tx.Commit()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		// Log activity
		utils.LogActivity(db, userID.(string), "", "email_changed", otpRecord.UserType, userID.(string), "User changed email to: "+req.NewEmail, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Email berhasil diperbarui. Silakan gunakan email baru untuk login berikutnya."})
	}
}


