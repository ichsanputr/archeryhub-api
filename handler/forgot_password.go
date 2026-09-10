package handler

import (
	"Archeris-api/utils"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// ForgotPassword â€” Step 1: user submits email, we find their account and send OTP
func ForgotPassword(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email tidak valid"})
			return
		}

		// Locate the user across all tables
		type Row struct {
			UUID     string `db:"uuid"`
			FullName string `db:"full_name"`
			UserType string
		}
		var found *Row

		tables := []struct {
			table    string
			nameCol  string
			userType string
		}{
			{"archers", "full_name", "archer"},
			{"organizers", "name", "organizer"},
			{"clubs", "name", "club"},
			{"sellers", "store_name", "seller"},
		}

		for _, t := range tables {
			var r Row
			err := db.Get(&r, fmt.Sprintf(
				"SELECT uuid, %s as full_name FROM %s WHERE email = ? LIMIT 1",
				t.nameCol, t.table,
			), req.Email)
			if err == nil {
				r.UserType = t.userType
				found = &r
				break
			}
		}

		// Always return 200 to avoid email enumeration
		if found == nil {
			c.JSON(http.StatusOK, gin.H{"message": "Jika email terdaftar, kode OTP telah dikirimkan"})
			return
		}

		// Invalidate all previous unused OTPs for this email
		db.Exec(`UPDATE password_resets SET is_used = 1 WHERE email = ? AND is_used = 0`, req.Email)

		// Generate OTP and persist
		otp := utils.GenerateOTP()
		resetID := uuid.New().String()
		expiry := time.Now().Add(5 * time.Minute)

		_, err := db.Exec(`
			INSERT INTO password_resets (uuid, email, user_id, user_type, otp_code, expires_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, resetID, req.Email, found.UUID, found.UserType, otp, expiry)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan"})
			return
		}

		// Send OTP email with Archeris design system (navy + neon yellow)
		go utils.SendOTPEmail(req.Email, found.FullName, otp, 5)

		c.JSON(http.StatusOK, gin.H{"message": "Jika email terdaftar, kode OTP telah dikirimkan"})
	}
}

// VerifyResetOTP â€” Step 2: validate OTP only (returns a short-lived token)
func VerifyResetOTP(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
			OTP   string `json:"otp"   binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak lengkap"})
			return
		}

		type ResetRow struct {
			UUID      string    `db:"uuid"`
			UserID    string    `db:"user_id"`
			UserType  string    `db:"user_type"`
			IsUsed    bool      `db:"is_used"`
			ExpiresAt time.Time `db:"expires_at"`
		}
		var row ResetRow
		err := db.Get(&row, `
			SELECT uuid, user_id, user_type, is_used, expires_at
			FROM password_resets
			WHERE email = ? AND otp_code = ?
			ORDER BY created_at DESC LIMIT 1
		`, req.Email, req.OTP)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode OTP tidak valid"})
			return
		}
		if row.IsUsed {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode OTP sudah digunakan"})
			return
		}
		if time.Now().After(row.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode OTP sudah kedaluwarsa"})
			return
		}

		// Generate a one-time reset token valid for 10 minutes
		resetToken, _ := generateRandomToken(24)
		tokenExpiry := time.Now().Add(10 * time.Minute)
		db.Exec(`UPDATE password_resets SET otp_code = ? , expires_at = ? WHERE uuid = ?`,
			"VERIFIED:"+resetToken, tokenExpiry, row.UUID)

		c.JSON(http.StatusOK, gin.H{
			"message":     "OTP valid",
			"reset_token": resetToken,
			"user_type":   row.UserType,
		})
	}
}

// ResetPassword — Step 3: set the new password using the verified reset token
func ResetPassword(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email       string `json:"email"        binding:"required,email"`
			ResetToken  string `json:"reset_token"  binding:"required"`
			NewPassword string `json:"new_password" binding:"required,min=6"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak lengkap atau password terlalu pendek (minimal 6 karakter)"})
			return
		}

		tx, txErr := db.Beginx()
		if txErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses reset password"})
			return
		}
		defer tx.Rollback()

		type ResetRow struct {
			UUID      string    `db:"uuid"`
			UserID    string    `db:"user_id"`
			UserType  string    `db:"user_type"`
			IsUsed    bool      `db:"is_used"`
			ExpiresAt time.Time `db:"expires_at"`
		}
		var row ResetRow
		err := tx.Get(&row, `
			SELECT uuid, user_id, user_type, is_used, expires_at
			FROM password_resets
			WHERE email = ? AND otp_code = ?
			ORDER BY created_at DESC LIMIT 1 FOR UPDATE
		`, req.Email, "VERIFIED:"+req.ResetToken)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi reset tidak valid atau sudah kedaluwarsa"})
			return
		}
		if row.IsUsed {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token reset sudah digunakan"})
			return
		}
		if time.Now().After(row.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi reset sudah kedaluwarsa, silakan mulai ulang"})
			return
		}

		// Map user_type to table
		table := "archers"
		switch row.UserType {
		case "organizer":
			table = "organizers"
		case "club":
			table = "clubs"
		case "seller":
			table = "sellers"
		}

		// Update password and increment token_version to invalidate other sessions
		_, err = tx.Exec(fmt.Sprintf(
			"UPDATE %s SET password = ?, token_version = token_version + 1, updated_at = NOW() WHERE uuid = ?",
			table,
		), req.NewPassword, row.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan password baru"})
			return
		}

		// Mark token as used
		_, err = tx.Exec(`UPDATE password_resets SET is_used = 1 WHERE uuid = ?`, row.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan proses reset password"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan proses reset password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password berhasil direset. Silakan masuk dengan password baru."})
	}
}

// ChangePasswordWithOTP â€” public endpoint to set new password directly using email + OTP
func ChangePasswordWithOTP(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			OTP         string `json:"otp" binding:"required"`
			NewPassword string `json:"new_password" binding:"required,min=6"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak lengkap atau tidak valid"})
			return
		}

		matched, _ := regexp.MatchString(`^\d{6}$`, req.OTP)
		if !matched {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format OTP tidak valid"})
			return
		}

		type ResetRow struct {
			UUID      string    `db:"uuid"`
			UserID    string    `db:"user_id"`
			UserType  string    `db:"user_type"`
			IsUsed    bool      `db:"is_used"`
			ExpiresAt time.Time `db:"expires_at"`
		}

		var row ResetRow
		err := db.Get(&row, `
			SELECT uuid, user_id, user_type, is_used, expires_at
			FROM password_resets
			WHERE email = ? AND otp_code = ?
			ORDER BY created_at DESC
			LIMIT 1
		`, req.Email, req.OTP)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "OTP tidak valid atau sudah kedaluwarsa"})
			return
		}

		if row.IsUsed || time.Now().After(row.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "OTP tidak valid atau sudah kedaluwarsa"})
			return
		}

		table := "archers"
		switch row.UserType {
		case "organizer":
			table = "organizers"
		case "club":
			table = "clubs"
		case "seller":
			table = "sellers"
		}

		tx, txErr := db.Beginx()
		if txErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses reset password"})
			return
		}

		// Hash the new password with bcrypt
		hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses kata sandi"})
			return
		}

		_, execErr := tx.Exec(fmt.Sprintf("UPDATE %s SET password = ?, token_version = token_version + 1, updated_at = NOW() WHERE uuid = ?", table), string(hashedPassword), row.UserID)
		if execErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan password baru"})
			return
		}

		_, execErr = tx.Exec(`UPDATE password_resets SET is_used = 1 WHERE email = ? AND is_used = 0`, req.Email)
		if execErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan proses reset password"})
			return
		}

		if commitErr := tx.Commit(); commitErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan proses reset password"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password berhasil direset. Silakan masuk dengan password baru."})
	}
}


