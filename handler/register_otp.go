package handler

import (
	"Archeris-api/utils"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type RegisterEmailRequest struct {
	UserType         string `json:"user_type" binding:"required"` // archer, organizer
	FullName         string `json:"full_name"`
	OrganizationName string `json:"organization_name"`
	Email            string `json:"email" binding:"required,email"`
	Password         string `json:"password" binding:"required,min=6"`
	Phone            string `json:"phone"`
	WhatsAppNo       string `json:"whatsapp_no"`
	Country          string `json:"country"`
	ClubID           string `json:"club_id"`
	NewClubName      string `json:"new_club_name"`
	NewClubAcronym   string `json:"new_club_acronym"`
}

type VerifyRegisterOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required"`
}

type ResendRegisterOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RegisterWithEmail initiates email + password signup and sends 6-digit OTP
func RegisterWithEmail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterEmailRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data registrasi tidak lengkap atau format tidak valid: " + err.Error()})
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		if req.UserType != "archer" && req.UserType != "organizer" {
			req.UserType = "archer"
		}

		name := strings.TrimSpace(req.FullName)
		if req.UserType == "organizer" {
			if req.OrganizationName != "" {
				name = strings.TrimSpace(req.OrganizationName)
			}
			if name == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Nama organisasi wajib diisi"})
				return
			}
		} else {
			if name == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Nama lengkap wajib diisi"})
				return
			}
		}

		phone := strings.TrimSpace(req.Phone)
		if phone == "" {
			phone = strings.TrimSpace(req.WhatsAppNo)
		}

		// Check if user already exists in archers or organizers
		type UserStub struct {
			UUID       string `db:"uuid"`
			Source     string `db:"source"`
			IsVerified bool   `db:"is_verified"`
		}
		var existingUser UserStub
		found := false

		err := db.Get(&existingUser, `SELECT uuid, 'archer' as source, is_verified FROM archers WHERE email = ? LIMIT 1`, req.Email)
		if err == nil {
			found = true
		} else {
			err = db.Get(&existingUser, `SELECT uuid, 'organizer' as source, is_verified FROM organizers WHERE email = ? LIMIT 1`, req.Email)
			if err == nil {
				found = true
			}
		}

		// Hash password securely with bcrypt
		hashedBytes, hErr := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if hErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengenkripsi password"})
			return
		}
		hashedPassword := string(hashedBytes)

		userID := ""
		isUpdate := false

		if found {
			if existingUser.IsVerified {
				c.JSON(http.StatusConflict, gin.H{"error": "Email sudah terdaftar. Silakan langsung masuk ke akun Anda."})
				return
			}
			// Unverified existing account -> update credentials & reuse
			userID = existingUser.UUID
			isUpdate = true
		} else {
			userID = uuid.New().String()
		}

		if req.UserType == "organizer" {
			cleanUsername := utils.CleanUsername(name)
			if cleanUsername == "" {
				cleanUsername = "org-" + userID[:8]
			}

			if isUpdate {
				_, err = db.Exec(`
					UPDATE organizers
					SET password = ?, name = ?, whatsapp_no = ?, status = 'inactive', is_verified = false, updated_at = NOW()
					WHERE uuid = ?
				`, hashedPassword, name, phone, userID)
			} else {
				insertQuery := `
					INSERT INTO organizers (uuid, user_id, slug, email, password, name, acronym, whatsapp_no, city, address, status, is_verified, quota_free, quota_standard, quota_elite, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, '', 'inactive', false, 20, 0, 0, NOW(), NOW())
				`
				_, err = db.Exec(insertQuery, userID, userID, cleanUsername, req.Email, hashedPassword, name, "", phone, "")
			}
		} else {
			// Archer
			var maxID sql.NullInt64
			_ = db.Get(&maxID, "SELECT MAX(CAST(SUBSTRING(id, 5) AS UNSIGNED)) FROM archers WHERE id REGEXP '^ARC-[0-9]+$'")
			nextIDNum := 1
			if maxID.Valid && maxID.Int64 > 0 {
				nextIDNum = int(maxID.Int64) + 1
			}
			athleteID := fmt.Sprintf("ARC-%04d", nextIDNum)

			username := utils.CleanUsername(name)
			if username == "" {
				username = "archer"
			}
			username = username + "-" + userID[:8]

			// Club ID handling (optional)
			var clubIDVal *string
			clubID := strings.TrimSpace(req.ClubID)
			if req.NewClubName != "" {
				newClubUUID := uuid.New().String()
				clubSlug := utils.CleanUsername(req.NewClubName)
				if clubSlug == "" {
					clubSlug = "club-" + newClubUUID[:8]
				}
				_, _ = db.Exec(`
					INSERT INTO clubs (uuid, slug, name, created_at, updated_at)
					VALUES (?, ?, ?, NOW(), NOW())
				`, newClubUUID, clubSlug, req.NewClubName)
				clubID = newClubUUID
			}
			if clubID != "" {
				clubIDVal = &clubID
			}

			if isUpdate {
				_, err = db.Exec(`
					UPDATE archers
					SET password = ?, full_name = ?, phone = ?, club_id = ?, status = 'inactive', is_verified = false, updated_at = NOW()
					WHERE uuid = ?
				`, hashedPassword, name, phone, clubIDVal, userID)
			} else {
				insertQuery := `
					INSERT INTO archers (uuid, id, username, email, password, full_name, phone, status, is_verified, gender, date_of_birth, bow_type, club_id, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, ?, ?, 'inactive', false, NULL, NULL, NULL, ?, NOW(), NOW())
				`
				_, err = db.Exec(insertQuery, userID, athleteID, username, req.Email, hashedPassword, name, phone, clubIDVal)
			}
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan akun: " + err.Error()})
			return
		}

		// Invalidate old OTPs
		_, _ = db.Exec(`UPDATE email_verifications SET is_used = 1 WHERE email = ? AND is_used = 0`, req.Email)

		// Generate OTP
		otp := utils.GenerateOTP()
		verifID := uuid.New().String()
		expiry := time.Now().Add(10 * time.Minute)

		_, err = db.Exec(`
			INSERT INTO email_verifications (uuid, email, otp_code, user_id, user_type, is_used, expires_at, created_at)
			VALUES (?, ?, ?, ?, ?, 0, ?, NOW())
		`, verifID, req.Email, otp, userID, req.UserType, expiry)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat kode verifikasi"})
			return
		}

		// Dispatch email
		go utils.SendOTPEmail(req.Email, name, otp, 10)

		c.JSON(http.StatusOK, gin.H{
			"status":  "pending_verification",
			"email":   req.Email,
			"message": "Kode OTP 6-digit telah dikirimkan ke email Anda",
		})
	}
}

// VerifyRegisterOTP validates the 6-digit OTP, activates the user, and logs them in
func VerifyRegisterOTP(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req VerifyRegisterOTPRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email dan kode OTP wajib diisi"})
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))
		req.OTP = strings.TrimSpace(req.OTP)

		matched, _ := regexp.MatchString(`^\d{6}$`, req.OTP)
		if !matched {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format kode OTP harus 6 digit angka"})
			return
		}

		type VerifRow struct {
			UUID      string    `db:"uuid"`
			UserID    string    `db:"user_id"`
			UserType  string    `db:"user_type"`
			ExpiresAt time.Time `db:"expires_at"`
			IsUsed    bool      `db:"is_used"`
		}
		var record VerifRow

		err := db.Get(&record, `
			SELECT uuid, user_id, user_type, expires_at, is_used
			FROM email_verifications
			WHERE email = ? AND otp_code = ?
			ORDER BY created_at DESC
			LIMIT 1
		`, req.Email, req.OTP)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode OTP tidak valid atau salah"})
			return
		}

		if record.IsUsed {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode OTP sudah pernah digunakan. Silakan minta kode baru."})
			return
		}

		if time.Now().After(record.ExpiresAt) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode OTP sudah kedaluwarsa. Silakan minta kode baru."})
			return
		}

		// Mark OTP as used
		_, _ = db.Exec(`UPDATE email_verifications SET is_used = 1 WHERE uuid = ?`, record.UUID)

		// Activate user
		role := record.UserType
		userName := ""
		if record.UserType == "organizer" {
			_, err = db.Exec(`UPDATE organizers SET is_verified = true, status = 'active', updated_at = NOW() WHERE uuid = ?`, record.UserID)
			var org struct {
				Name string `db:"name"`
			}
			_ = db.Get(&org, `SELECT name FROM organizers WHERE uuid = ?`, record.UserID)
			userName = org.Name
		} else {
			_, err = db.Exec(`UPDATE archers SET is_verified = true, status = 'active', updated_at = NOW() WHERE uuid = ?`, record.UserID)
			var arch struct {
				FullName string `db:"full_name"`
			}
			_ = db.Get(&arch, `SELECT full_name FROM archers WHERE uuid = ?`, record.UserID)
			userName = arch.FullName
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengaktifkan akun"})
			return
		}

		// Generate JWT token
		orgID := ""
		if record.UserType == "organizer" {
			orgID = record.UserID
		}
		token, err := generateJWT(record.UserID, req.Email, role, record.UserType, userName, "", orgID, 1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat sesi login"})
			return
		}

		setAuthCookie(c, token, 60*60*24*60) // 60 days

		utils.LogActivity(db, record.UserID, "", "user_verified_otp", record.UserType, record.UserID, "User verified via OTP: "+req.Email, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{
			"message":      "Akun berhasil diverifikasi",
			"token":        token,
			"redirect_url": "/dashboard",
			"user": gin.H{
				"id":         record.UserID,
				"email":      req.Email,
				"full_name":  userName,
				"role":       role,
				"user_type":  record.UserType,
				"is_verified": true,
			},
		})
	}
}

// ResendRegisterOTP sends a new 6-digit OTP code to the given email
func ResendRegisterOTP(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ResendRegisterOTPRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email wajib diisi"})
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))

		// Check rate limit: 1 request per 60 seconds
		var count int
		_ = db.Get(&count, `
			SELECT COUNT(1) FROM email_verifications 
			WHERE email = ? AND created_at > DATE_SUB(NOW(), INTERVAL 60 SECOND)
		`, req.Email)

		if count > 0 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Mohon tunggu 60 detik sebelum meminta kode OTP baru"})
			return
		}

		// Check user in database
		type Row struct {
			UUID     string `db:"uuid"`
			Name     string `db:"name"`
			UserType string
		}
		var user Row
		found := false

		err := db.Get(&user, `SELECT uuid, full_name as name FROM archers WHERE email = ? LIMIT 1`, req.Email)
		if err == nil {
			user.UserType = "archer"
			found = true
		} else {
			err = db.Get(&user, `SELECT uuid, name FROM organizers WHERE email = ? LIMIT 1`, req.Email)
			if err == nil {
				user.UserType = "organizer"
				found = true
			}
		}

		if !found {
			c.JSON(http.StatusNotFound, gin.H{"error": "Email pendaftaran tidak ditemukan"})
			return
		}

		// Invalidate old OTPs
		_, _ = db.Exec(`UPDATE email_verifications SET is_used = 1 WHERE email = ? AND is_used = 0`, req.Email)

		// Generate new OTP
		otp := utils.GenerateOTP()
		verifID := uuid.New().String()
		expiry := time.Now().Add(10 * time.Minute)

		_, err = db.Exec(`
			INSERT INTO email_verifications (uuid, email, otp_code, user_id, user_type, is_used, expires_at, created_at)
			VALUES (?, ?, ?, ?, ?, 0, ?, NOW())
		`, verifID, req.Email, otp, user.UUID, user.UserType, expiry)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat kode OTP baru"})
			return
		}

		go utils.SendOTPEmail(req.Email, user.Name, otp, 10)

		c.JSON(http.StatusOK, gin.H{"message": "Kode OTP baru telah dikirimkan ke email Anda"})
	}
}
