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

// ForgotPassword — Step 1: user submits email, we find their account and send OTP
func ForgotPassword(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email address", "code": "invalid_email"})
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
			c.JSON(http.StatusOK, gin.H{"message": "If this email is registered, a verification code has been sent"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request", "code": "server_error"})
			return
		}

		// Send OTP email with Archeris design system
		go utils.SendPasswordResetOTPEmail(req.Email, found.FullName, otp, 5)

		c.JSON(http.StatusOK, gin.H{"message": "If this email is registered, a verification code has been sent"})
	}
}

// VerifyResetOTP — Step 2: validate OTP only (returns a short-lived token)
func VerifyResetOTP(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email string `json:"email" binding:"required,email"`
			OTP   string `json:"otp"   binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Incomplete request data", "code": "invalid_input"})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Verification code is invalid or incorrect. Please check again.", "code": "otp_invalid"})
			return
		}
		if row.IsUsed {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Verification code has already been used. Please request a new code.", "code": "otp_already_used"})
			return
		}
		if time.Now().After(row.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Verification code has expired. Please request a new code.", "code": "otp_expired"})
			return
		}

		// Generate a one-time reset token valid for 10 minutes
		resetToken, _ := generateRandomToken(24)
		tokenExpiry := time.Now().Add(10 * time.Minute)
		db.Exec(`UPDATE password_resets SET otp_code = ? , expires_at = ? WHERE uuid = ?`,
			"VERIFIED:"+resetToken, tokenExpiry, row.UUID)

		c.JSON(http.StatusOK, gin.H{
			"message":     "OTP verified successfully",
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Password must be at least 6 characters", "code": "invalid_password"})
			return
		}

		tx, txErr := db.Beginx()
		if txErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password reset", "code": "server_error"})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Reset session is invalid or has expired", "code": "session_expired"})
			return
		}
		if row.IsUsed {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Reset token has already been used", "code": "token_used"})
			return
		}
		if time.Now().After(row.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Reset session has expired. Please start over.", "code": "session_expired"})
			return
		}

		// Map user_type to table
		table := "archers"
		switch row.UserType {
		case "organizer":
			table = "organizers"
		case "club":
			table = "clubs"
		}

		// Update password and increment token_version to invalidate other sessions
		_, err = tx.Exec(fmt.Sprintf(
			"UPDATE %s SET password = ?, token_version = token_version + 1, updated_at = NOW() WHERE uuid = ?",
			table,
		), req.NewPassword, row.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password", "code": "server_error"})
			return
		}

		// Mark token as used
		_, err = tx.Exec(`UPDATE password_resets SET is_used = 1 WHERE uuid = ?`, row.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize password reset", "code": "server_error"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize password reset", "code": "server_error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully. Please log in with your new password."})
	}
}

// ChangePasswordWithOTP — public endpoint to set new password directly using email + OTP
func ChangePasswordWithOTP(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			OTP         string `json:"otp" binding:"required"`
			NewPassword string `json:"new_password" binding:"required,min=6"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Incomplete or invalid request data", "code": "invalid_input"})
			return
		}

		matched, _ := regexp.MatchString(`^\d{6}$`, req.OTP)
		if !matched {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Verification code must be 6 digits", "code": "invalid_format"})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Verification code is invalid or incorrect. Please check again.", "code": "otp_invalid"})
			return
		}

		if row.IsUsed {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Verification code has already been used. Please request a new code.", "code": "otp_already_used"})
			return
		}

		if time.Now().After(row.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Verification code has expired. Please request a new code.", "code": "otp_expired"})
			return
		}

		table := "archers"
		switch row.UserType {
		case "organizer":
			table = "organizers"
		case "club":
			table = "clubs"
		}

		tx, txErr := db.Beginx()
		if txErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password reset", "code": "server_error"})
			return
		}

		// Hash the new password with bcrypt
		hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password", "code": "server_error"})
			return
		}

		_, execErr := tx.Exec(fmt.Sprintf("UPDATE %s SET password = ?, token_version = token_version + 1, updated_at = NOW() WHERE uuid = ?", table), string(hashedPassword), row.UserID)
		if execErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save new password", "code": "server_error"})
			return
		}

		_, execErr = tx.Exec(`UPDATE password_resets SET is_used = 1 WHERE email = ? AND is_used = 0`, req.Email)
		if execErr != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize password reset", "code": "server_error"})
			return
		}

		if commitErr := tx.Commit(); commitErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize password reset", "code": "server_error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully. Please log in with your new password."})
	}
}


