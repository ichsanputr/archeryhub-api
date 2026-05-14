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
		expiry := time.Now().Add(5 * time.Minute)

		_, err := db.Exec(`
			INSERT INTO password_resets (uuid, email, user_id, user_type, otp_code, expires_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, resetID, req.Email, found.UUID, found.UserType, otp, expiry)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan"})
			return
		}

		// Send OTP email
		subject := "Kode Reset Password â€“ archeris.net"
		body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Lexend:wght@500;700;800;900&display=swap" rel="stylesheet">
	<style>
		body { margin: 0; padding: 0; font-family: 'Inter', sans-serif; background: #020617; }
		.wrapper { width: 100%%; padding: 32px 12px; box-sizing: border-box; }
		.container { width: 100%%; max-width: 560px; margin: 0 auto; background: #ffffff; border-radius: 24px; overflow: hidden; box-shadow: 0 22px 60px rgba(2,6,23,0.45); }
		.hero { background: linear-gradient(170deg, #0f172a 0%%, #111827 60%%, #052e16 100%%); padding: 34px 28px 28px; }
		.brand { margin: 0 0 18px; font-family: 'Lexend', sans-serif; color: #ffffff; font-size: 30px; font-weight: 900; letter-spacing: -0.6px; }
		.brand-accent { color: #84cc16; }
		.hero-title { margin: 0; color: #e2e8f0; font-family: 'Lexend', sans-serif; font-size: 20px; font-weight: 800; line-height: 1.35; }
		.hero-sub { margin: 10px 0 0; color: #94a3b8; font-size: 13px; line-height: 1.7; }
		.content { padding: 30px 28px 10px; }
		.badge { display: inline-block; margin-bottom: 14px; padding: 6px 12px; border-radius: 999px; background: #f0fdf4; border: 1px solid #dcfce7; color: #4d7c0f; font-size: 11px; font-weight: 800; letter-spacing: 0.8px; text-transform: uppercase; }
		.title { margin: 0 0 12px; color: #0f172a; font-family: 'Lexend', sans-serif; font-size: 26px; font-weight: 900; line-height: 1.2; }
		.text { margin: 0 0 22px; color: #475569; font-size: 15px; line-height: 1.75; }
		.otp-card { background: linear-gradient(180deg, #f7fee7 0%%, #ecfccb 100%%); border: 1px solid #d9f99d; border-radius: 18px; padding: 22px 14px; text-align: center; margin-bottom: 20px; }
		.otp-label { color: #4d7c0f; font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: 2.4px; margin-bottom: 10px; }
		.otp-code { margin: 0; color: #365314; font-family: 'Lexend', sans-serif; font-size: 46px; font-weight: 900; letter-spacing: 10px; line-height: 1; padding-left: 10px; }
		.tip { margin: 0 0 26px; color: #64748b; font-size: 13px; line-height: 1.7; }
		.tip b { color: #0f172a; }
		.footer { padding: 0 28px 28px; }
		.footer-box { border-top: 1px solid #e2e8f0; padding-top: 18px; }
		.footer-text { margin: 0 0 10px; color: #94a3b8; font-size: 12px; line-height: 1.7; text-align: center; }
		.copyright { margin: 0; color: #cbd5e1; font-size: 11px; font-weight: 600; text-align: center; }
		@media only screen and (max-width: 640px) {
			.wrapper { padding: 16px 8px; }
			.hero { padding: 28px 20px 24px; }
			.brand { font-size: 26px; }
			.hero-title { font-size: 18px; }
			.content { padding: 24px 20px 8px; }
			.title { font-size: 23px; }
			.otp-code { font-size: 38px; letter-spacing: 7px; padding-left: 7px; }
			.footer { padding: 0 20px 24px; }
		}
	</style>
</head>
<body>
	<div class="wrapper">
		<div class="container">
			<div class="hero">
				<h2 class="brand">Archeris<span class="brand-accent">.id</span></h2>
				<h3 class="hero-title">Atur Ulang Akses Akun Kamu</h3>
				<p class="hero-sub">Kami menjaga akunmu tetap aman dengan verifikasi OTP sekali pakai.</p>
			</div>

			<div class="content">
				<span class="badge">Reset Password</span>
				<h1 class="title">Hai, %s!</h1>
				<p class="text">Kami menerima permintaan untuk mereset password akun kamu. Masukkan kode OTP berikut ke halaman reset password archeris.net.</p>

				<div class="otp-card">
					<div class="otp-label">Kode OTP Kamu</div>
					<p class="otp-code">%s</p>
				</div>

				<p class="tip"><b>Penting:</b> kode ini berlaku selama 5 menit dan hanya bisa dipakai satu kali. Jika kamu tidak merasa melakukan permintaan ini, abaikan email ini.</p>
			</div>

			<div class="footer">
				<div class="footer-box">
					<p class="footer-text">Email ini dikirim otomatis oleh sistem archeris.net untuk keamanan akun kamu.</p>
					<p class="copyright">&copy; 2026 archeris.net &bull; Platform Panahan No. 1 Indonesia</p>
				</div>
			</div>
		</div>
	</div>
</body>
</html>`, found.FullName, otp)

		// Fire-and-forget (don't fail request if email fails)
		go utils.SendEmail(req.Email, subject, body)

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

// ResetPassword â€” Step 3: set the new password using the verified reset token
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
		case "organization":
			table = "organizations"
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

		_, execErr := tx.Exec(fmt.Sprintf("UPDATE %s SET password = ?, updated_at = NOW() WHERE uuid = ?", table), req.NewPassword, row.UserID)
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


