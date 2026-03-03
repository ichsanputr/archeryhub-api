package handler

import (
	"archeryhub-api/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ForgotPassword — Step 1: user submits email, we find their account and send OTP
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
			{"organizations", "name", "organization"},
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
		expiry := time.Now().Add(15 * time.Minute)

		_, err := db.Exec(`
			INSERT INTO password_resets (uuid, email, user_id, user_type, otp_code, expires_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, resetID, req.Email, found.UUID, found.UserType, otp, expiry)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan"})
			return
		}

		// Send OTP email
		subject := "Kode Reset Password – Archeryhub.id"
		body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;700;900&family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
    <style>
        body { margin: 0; padding: 0; font-family: 'Inter', sans-serif; background-color: #f7f9fc; }
        .container { width: 100%%; max-width: 500px; margin: 40px auto; background: #ffffff; border-radius: 24px; overflow: hidden; box-shadow: 0 10px 40px rgba(0,0,0,0.05); }
        .header { background-color: #0f172a; padding: 40px 20px; text-align: center; }
        .logo { font-family: 'Outfit', sans-serif; color: #ffffff; font-size: 28px; font-weight: 900; letter-spacing: -1px; }
        .logo-id { color: #f9d006; }
        .content { padding: 48px 40px; }
        .label { font-size: 11px; color: #64748b; font-weight: 800; text-transform: uppercase; letter-spacing: 2px; margin-bottom: 12px; display: block; }
        .title { font-family: 'Outfit', sans-serif; font-size: 26px; color: #0f172a; font-weight: 900; line-height: 1.2; margin: 0 0 16px; }
        .text { font-size: 15px; color: #475569; line-height: 1.6; margin: 0 0 32px; }
        .otp-container { background-color: #fffbeb; border: 2px solid #fef3c7; border-radius: 20px; padding: 32px 20px; text-align: center; margin-bottom: 32px; }
        .otp-label { font-size: 10px; color: #b45309; font-weight: 800; text-transform: uppercase; letter-spacing: 3px; margin-bottom: 8px; }
        .otp-code { font-family: 'Outfit', sans-serif; font-size: 56px; color: #f9d006; font-weight: 900; letter-spacing: 12px; margin: 0; line-height: 1; margin-left: 12px; }
        .footer { padding: 0 40px 40px; text-align: center; border-top: 1px solid #f1f5f9; padding-top: 32px; }
        .footer-text { font-size: 12px; color: #94a3b8; line-height: 1.6; margin-bottom: 16px; }
        .copyright { font-size: 11px; color: #cbd5e1; font-weight: 600; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <div class="logo">Archeryhub<span class="logo-id">.id</span></div>
        </div>
        <div class="content">
            <span class="label">Reset Password</span>
            <h1 class="title">Hai, %s!</h1>
            <p class="text">
                Kami menerima permintaan untuk mereset password akun kamu. 
                Gunakan kode OTP di bawah ini untuk melanjutkan ke langkah berikutnya.
            </p>
            <div class="otp-container">
                <div class="otp-label">Kode OTP Anda</div>
                <div class="otp-code">%s</div>
            </div>
            <p class="text" style="font-size: 13px; color: #94a3b8; font-style: italic; margin-bottom: 0;">
                Kode ini berlaku selama 15 menit. Jika kamu tidak merasa melakukan permintaan ini, silakan abaikan email ini.
            </p>
        </div>
        <div class="footer">
            <p class="footer-text">Email ini dikirim secara otomatis oleh sistem Archeryhub.id untuk melindungi keamanan akun kamu.</p>
            <div class="copyright">&copy; 2025 Archeryhub.id &bull; Platform Panahan No. 1 Indonesia</div>
        </div>
    </div>
</body>
</html>`, found.FullName, otp)

		// Fire-and-forget (don't fail request if email fails)
		go utils.SendEmail(req.Email, subject, body)

		c.JSON(http.StatusOK, gin.H{"message": "Jika email terdaftar, kode OTP telah dikirimkan"})
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
			"message":      "OTP valid",
			"reset_token":  resetToken,
			"user_type":    row.UserType,
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
		case "organization":
			table = "organizations"
		case "club":
			table = "clubs"
		case "seller":
			table = "sellers"
		}

		// Update password (plain text, matching existing auth pattern)
		_, err = db.Exec(fmt.Sprintf(
			"UPDATE %s SET password = ?, updated_at = NOW() WHERE uuid = ?",
			table,
		), req.NewPassword, row.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan password baru"})
			return
		}

		// Mark token as used
		db.Exec(`UPDATE password_resets SET is_used = 1 WHERE uuid = ?`, row.UUID)

		c.JSON(http.StatusOK, gin.H{"message": "Password berhasil direset. Silakan masuk dengan password baru."})
	}
}
