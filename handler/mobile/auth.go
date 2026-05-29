package mobile

import (
	"Archeris-api/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type MobileScorekeeperLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

type mobileEmailPasswordRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type MobileOAuthLoginRequest struct {
	IDToken string `json:"idToken" binding:"required"`
}

type mobileLoginUser struct {
	UUID         string  `db:"uuid"`
	ID           string  `db:"id"`
	Username     string  `db:"username"`
	Email        string  `db:"email"`
	Password     string  `db:"password"`
	FullName     string  `db:"full_name"`
	AvatarURL    *string `db:"avatar_url"`
	Status       string  `db:"status"`
	TokenVersion int     `db:"token_version"`
}

func handleMobileEmailPasswordLogin(c *gin.Context, db *sqlx.DB, query string, req mobileEmailPasswordRequest, role string, userType string) {
	var user mobileLoginUser
	err := db.Get(&user, query, req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau kata sandi tidak valid", "code": "invalid_credentials"})
		return
	}

	if user.Status != "active" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akun tidak aktif", "code": "account_inactive"})
		return
	}
	if user.Password == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Akun ini menggunakan Google sign-in. Silakan masuk menggunakan Google.", "code": "use_google_signin"})
		return
	}
	if user.Password != req.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau kata sandi tidak valid", "code": "invalid_credentials"})
		return
	}

	avatar := ""
	if user.AvatarURL != nil {
		avatar = utils.MaskMediaURL(*user.AvatarURL)
	}

	organizationUUID := ""
	if userType == "organizer" {
		organizationUUID = user.UUID
	}

	token, err := generateJWT(user.UUID, user.Email, role, userType, user.FullName, avatar, organizationUUID, user.TokenVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
		return
	}

	utils.LogActivity(db, user.UUID, "", "mobile_login", userType, user.UUID, "User logged in via mobile", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, MobileLoginResponse{
		Token:     token,
		IsNewUser: false,
		User: MobileUser{
			UUID:      user.UUID,
			ID:        user.ID,
			Username:  user.Username,
			FullName:  user.FullName,
			Email:     user.Email,
			AvatarURL: avatar,
			Role:      role,
			UserType:  userType,
		},
	})
}

// MobileScorekeeperLogin godoc
// MobileScorekeeperLogin handles scorekeeper login
// @Summary Scorekeeper Login
// @Description Login using a numeric code assigned by organizer
// @Tags Mobile - Scorekeeper
// @Accept json
// @Produce json
// @Param request body MobileScorekeeperLoginRequest true "Scorekeeper Code"
// @Success 200 {object} MobileLoginResponse
// @Failure 401 {object} map[string]interface{}
// @Router /mobile/auth/scorekeeper/login [post]
func MobileScorekeeperLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileScorekeeperLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode wajib diisi"})
			return
		}

		var sk struct {
			UUID             string  `db:"uuid"`
			OrganizationUUID string  `db:"organization_uuid"`
			Code             string  `db:"code"`
			Name             string  `db:"name"`
			Email            string  `db:"email"`
			AvatarURL        *string `db:"avatar_url"`
			Status           string  `db:"status"`
			OrgSubStatus     *string `db:"org_sub_status"`
			TokenVersion     int     `db:"token_version"`
		}

		err := db.Get(&sk, `
			SELECT sk.uuid, sk.organization_uuid, sk.code, sk.name, IFNULL(sk.email, '') as email, sk.avatar_url, COALESCE(sk.status, '') as status,
                   o.subscription_status as org_sub_status, sk.token_version
			FROM scorekeepers sk 
            JOIN organizers o ON sk.organization_uuid = o.uuid
            WHERE sk.code = ?`, req.Code)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode scorekeeper tidak valid", "code": "invalid_code"})
			return
		}

		if sk.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun scorekeeper tidak aktif", "code": "account_inactive"})
			return
		}

		// Check Organizer Subscription
		orgSub := "trial"
		if sk.OrgSubStatus != nil {
			orgSub = *sk.OrgSubStatus
		}

		if orgSub != "active" && orgSub != "trial" {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":   "Langganan organisasi telah berakhir",
				"code":    "subscription_expired",
				"message": "Layanan scoring dihentikan sementara karena masa berlaku langganan organisasi telah berakhir. Silakan hubungi admin organisasi Anda.",
			})
			return
		}

		avatar := ""
		if sk.AvatarURL != nil {
			avatar = utils.MaskMediaURL(*sk.AvatarURL)
		}

		token, err := generateJWT(sk.UUID, sk.Email, "scorekeeper", "scorekeeper", sk.Name, avatar, sk.OrganizationUUID, sk.TokenVersion)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
			return
		}

		utils.LogActivity(db, sk.UUID, "", "mobile_login", "scorekeeper", sk.UUID, "Scorekeeper logged in via mobile", c.ClientIP(), c.Request.UserAgent())
		utils.LogScorekeeperAction(db, sk.UUID, sk.OrganizationUUID, "", "mobile_login", "Logged in via mobile app", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, MobileLoginResponse{
			Token:     token,
			IsNewUser: false,
			User: MobileUser{
				UUID:      sk.UUID,
				ID:        sk.UUID,
				Username:  sk.Code,
				FullName:  sk.Name,
				Email:     sk.Email,
				AvatarURL: avatar,
				Role:      "scorekeeper",
				UserType:  "scorekeeper",
			},
		})
	}
}

// MobileListEvents godoc

// ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ Archer Auth ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡ÃŽâ€œÃƒÂ¶Ãƒâ€¡

// MobileArcherLogin godoc
// MobileArcherLogin handles archer login for mobile
// @Summary Archer Login
// @Description Login using email and password for archers
// @Tags Mobile - Archer
// @Accept json
// @Produce json
// @Param request body mobileEmailPasswordRequest true "Login Credentials"
// @Success 200 {object} MobileLoginResponse
// @Failure 401 {object} map[string]interface{}
// @Router /mobile/auth/archer/login [post]
func MobileArcherLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req mobileEmailPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		handleMobileEmailPasswordLogin(
			c,
			db,
			`SELECT uuid, id, username, email, COALESCE(password,'') as password, full_name, avatar_url, COALESCE(status,'') as status, token_version FROM archers WHERE email = ?`,
			req,
			"archer",
			"archer",
		)
	}
}

// MobileOrganizationLogin godoc
// MobileOrganizationLogin handles organizer login for mobile
// @Summary Organizer Login
// @Description Login for organizer accounts
// @Tags Mobile - Organizer
// @Accept json
// @Produce json
// @Param request body mobileEmailPasswordRequest true "Login Credentials"
// @Success 200 {object} MobileLoginResponse
// @Failure 401 {object} map[string]interface{}
// @Router /mobile/auth/organizer/login [post]
func MobileOrganizationLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req mobileEmailPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		handleMobileEmailPasswordLogin(
			c,
			db,
			`SELECT uuid, uuid as id, slug as username, email, COALESCE(password,'') as password, name as full_name, avatar_url, COALESCE(status,'') as status, token_version FROM organizers WHERE email = ?`,
			req,
			"organizer",
			"organizer",
		)
	}
}

// MobileSellerLogin godoc
// MobileSellerLogin handles seller login for mobile
// @Summary Seller Login
// @Description Login for seller accounts
// @Tags Mobile - Seller
// @Accept json
// @Produce json
// @Param request body mobileEmailPasswordRequest true "Login Credentials"
// @Success 200 {object} MobileLoginResponse
// @Failure 401 {object} map[string]interface{}
// @Router /mobile/auth/seller/login [post]
func MobileSellerLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req mobileEmailPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		handleMobileEmailPasswordLogin(
			c,
			db,
			`SELECT uuid, uuid as id, slug as username, email, COALESCE(password,'') as password, store_name as full_name, avatar_url, COALESCE(status,'') as status, token_version FROM sellers WHERE email = ?`,
			req,
			"seller",
			"seller",
		)
	}
}

// MobileArcherRegister godoc
// MobileArcherRegister handles archer registration for mobile
// @Summary Archer Registration
// @Description Register a new archer account
// @Tags Mobile - Archer
// @Accept json
// @Produce json
// @Param request body MobileArcherRegisterRequest true "Registration Details"
// @Success 201 {object} MobileLoginResponse
// @Failure 400 {object} map[string]interface{}
// @Router /mobile/auth/archer/register [post]
func MobileArcherRegister(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileArcherRegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var exists bool
		db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM archers WHERE email = ?)`, req.Email)
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Email sudah terdaftar", "code": "email_exists"})
			return
		}

		userID := uuid.New().String()

		var lastID string
		_ = db.Get(&lastID, "SELECT id FROM archers WHERE id LIKE 'ARC-%' ORDER BY id DESC LIMIT 1")
		nextIDNum := 1
		if lastID != "" {
			parts := strings.Split(lastID, "-")
			if len(parts) == 2 {
				fmt.Sscanf(parts[1], "%d", &nextIDNum)
				nextIDNum++
			}
		}
		athleteID := fmt.Sprintf("ARC-%04d", nextIDNum)

		username := utils.CleanUsername(req.FullName)
		if username == "" {
			username = "archer"
		}
		username = username + "-" + userID[:8]

		_, err := db.Exec(`
			INSERT INTO archers (uuid, id, username, email, password, full_name, phone, status, is_verified, gender, date_of_birth, city, bow_type)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'active', 1, ?, ?, ?, ?)
		`, userID, athleteID, username, req.Email, req.Password, req.FullName, req.Phone, req.Gender, req.DateOfBirth, req.City, req.BowType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat akun: " + err.Error()})
			return
		}

		token, err := generateJWT(userID, req.Email, "archer", "archer", req.FullName, "", "", 1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
			return
		}

		c.JSON(http.StatusCreated, MobileLoginResponse{
			Token:     token,
			IsNewUser: true,
			User: MobileUser{
				UUID:      userID,
				ID:        athleteID,
				Username:  username,
				FullName:  req.FullName,
				Email:     req.Email,
				Role:      "archer",
				UserType:  "archer",
			},
		})
	}
}

// MobileSellerRegister handles seller registration for mobile
// @Summary Seller Registration
// @Description Register a new seller/merchant
// @Tags Mobile - Seller
// @Accept json
// @Produce json
// @Param request body MobileSellerRegisterRequest true "Seller Registration Data"
// @Success 201 {object} MobileResponse
// @Router /mobile/auth/seller/register [post]
func MobileSellerRegister(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileSellerRegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var exists bool
		db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM sellers WHERE email = ?)`, req.Email)
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Email sudah terdaftar", "code": "email_exists"})
			return
		}

		sellerUUID := uuid.New().String()
		slug := utils.CleanUsername(req.StoreName)
		if slug == "" {
			slug = "seller"
		}
		slug = slug + "-" + sellerUUID[:8]

		_, err := db.Exec(`
			INSERT INTO sellers (uuid, slug, store_name, email, password, phone, city, province, address, status, is_verified, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', 1, NOW(), NOW())
		`, sellerUUID, slug, req.StoreName, req.Email, req.Password, req.Phone, req.City, req.Province, req.Address)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat akun toko: " + err.Error()})
			return
		}

		token, err := generateJWT(sellerUUID, req.Email, "seller", "seller", req.StoreName, "", "", 1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
			return
		}

		c.JSON(http.StatusCreated, MobileLoginResponse{
			Token:     token,
			IsNewUser: true,
			User: MobileUser{
				UUID:      sellerUUID,
				ID:        sellerUUID,
				Username:  slug,
				FullName:  req.StoreName,
				Email:     req.Email,
				Role:      "seller",
				UserType:  "seller",
			},
		})
	}
}

type googleInfo struct {
	Email         string `json:"email"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Sub           string `json:"sub"` // Google ID
	EmailVerified string `json:"email_verified"`
}

func verifyGoogleToken(idToken string) (*googleInfo, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid status code: %d", resp.StatusCode)
	}

	var info googleInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// MobileGoogleLogin handles Google Sign-In for mobile using idToken
// @Summary Google Login
// @Description Authenticate or register using Google ID Token
// @Tags Mobile - Auth
// @Accept json
// @Produce json
// @Param request body MobileOAuthLoginRequest true "Google ID Token"
// @Success 200 {object} MobileLoginResponse
// @Failure 401 {object} map[string]interface{}
// @Router /mobile/auth/google/login [post]
func MobileGoogleLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileOAuthLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID Token wajib diisi"})
			return
		}

		// Verify ID Token with Google
		googleInfo, err := verifyGoogleToken(req.IDToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Google ID Token tidak valid", "details": err.Error()})
			return
		}

		if googleInfo.Email == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email tidak ditemukan di Google token"})
			return
		}

		// Search for existing user (Archer only for now for mobile app)
		var user mobileLoginUser
		err = db.Get(&user, `SELECT uuid, id, username, email, COALESCE(password,'') as password, full_name, avatar_url, COALESCE(status,'') as status, token_version FROM archers WHERE email = ? OR google_id = ?`, googleInfo.Email, googleInfo.Sub)

		isNewUser := false
		if err != nil {
			// User not found, register new archer
			isNewUser = true
			userID := uuid.New().String()

			// Generate athlete ID (ARC-XXXX)
			var lastID string
			_ = db.Get(&lastID, "SELECT id FROM archers WHERE id LIKE 'ARC-%' ORDER BY id DESC LIMIT 1")
			nextIDNum := 1
			if lastID != "" {
				parts := strings.Split(lastID, "-")
				if len(parts) == 2 {
					fmt.Sscanf(parts[1], "%d", &nextIDNum)
					nextIDNum++
				}
			}
			athleteID := fmt.Sprintf("ARC-%04d", nextIDNum)

			// Generate username
			username := utils.CleanUsername(googleInfo.Name)
			if username == "" {
				username = "archer"
			}
			username = username + "-" + userID[:8]

			_, err = db.Exec(`
				INSERT INTO archers (uuid, id, username, email, google_id, full_name, avatar_url, status, is_verified, created_at, updated_at, token_version)
				VALUES (?, ?, ?, ?, ?, ?, ?, 'active', 1, NOW(), NOW(), 1)
			`, userID, athleteID, username, googleInfo.Email, googleInfo.Sub, googleInfo.Name, googleInfo.Picture)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan user baru: " + err.Error()})
				return
			}

			// Assign values to user struct for token generation
			user.UUID = userID
			user.ID = athleteID
			user.Username = username
			user.Email = googleInfo.Email
			user.FullName = googleInfo.Name
			user.AvatarURL = &googleInfo.Picture
			user.Status = "active"
			user.TokenVersion = 1

			utils.LogActivity(db, userID, "", "mobile_register_google", "archer", userID, "User registered via Google on mobile", c.ClientIP(), c.Request.UserAgent())
		} else {
			// Update existing user with Google ID and Avatar if needed
			_, _ = db.Exec(`UPDATE archers SET google_id = ?, avatar_url = ?, updated_at = NOW() WHERE uuid = ?`, googleInfo.Sub, googleInfo.Picture, user.UUID)
		}

		if user.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun tidak aktif", "code": "account_inactive"})
			return
		}

		avatar := ""
		if user.AvatarURL != nil {
			avatar = *user.AvatarURL
		}

		token, err := generateJWT(user.UUID, user.Email, "archer", "archer", user.FullName, avatar, "", user.TokenVersion)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
			return
		}

		utils.LogActivity(db, user.UUID, "", "mobile_login_google", "archer", user.UUID, "User logged in via Google mobile", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, MobileLoginResponse{
			Token:     token,
			IsNewUser: isNewUser,
			User: MobileUser{
				UUID:      user.UUID,
				ID:        user.ID,
				Username:  user.Username,
				FullName:  user.FullName,
				Email:     user.Email,
				AvatarURL: avatar,
				Role:      "archer",
				UserType:  "archer",
			},
		})
	}
}

// MobileForgotPassword initiates password reset
// @Summary Forgot Password
// @Description Send a password reset OTP to user's email
// @Tags Mobile - Auth
// @Accept json
// @Produce json
// @Param request body MobileForgotPasswordRequest true "Email Address"
// @Success 200 {object} MobileResponse
// @Router /mobile/auth/forgot-password [post]
func MobileForgotPassword(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileForgotPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email tidak valid"})
			return
		}

		var userData struct {
			UUID     string `db:"uuid"`
			FullName string `db:"full_name"`
			UserType string `db:"user_type"`
		}

		// Try archers first
		err := db.Get(&userData, "SELECT uuid, full_name, 'archer' as user_type FROM archers WHERE email = ? LIMIT 1", req.Email)
		if err != nil {
			// Try organizers
			err = db.Get(&userData, "SELECT uuid, name as full_name, 'organizer' as user_type FROM organizers WHERE email = ? LIMIT 1", req.Email)
			if err != nil {
				// Try sellers
				err = db.Get(&userData, "SELECT uuid, store_name as full_name, 'seller' as user_type FROM sellers WHERE email = ? LIMIT 1", req.Email)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": "Email tidak terdaftar"})
					return
				}
			}
		}

		otp := utils.GenerateOTP()
		expiresAt := time.Now().Add(15 * time.Minute)

		_, err = db.Exec(`
			INSERT INTO password_resets (uuid, email, user_id, user_type, otp_code, expires_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, uuid.New().String(), req.Email, userData.UUID, userData.UserType, otp, expiresAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan", "details": err.Error()})
			return
		}

		emailBody := fmt.Sprintf(`
			<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto; padding: 20px; border: 1px solid #eee; border-radius: 10px;">
				<h3 style="color: #333;">Pemulihan Kata Sandi Archeris</h3>
				<p>Halo <strong>%s</strong>,</p>
				<p>Anda telah meminta pemulihan kata sandi. Gunakan kode OTP berikut untuk melanjutkan:</p>
				<div style="background-color: #f9f9f9; padding: 20px; text-align: center; border-radius: 5px; margin: 20px 0;">
					<h2 style="letter-spacing: 12px; color: #C1121F; margin: 0; font-size: 32px;">%s</h2>
				</div>
				<p style="color: #666; font-size: 14px;">Kode ini akan kadaluwarsa dalam 15 menit.</p>
				<p style="color: #999; font-size: 12px; margin-top: 30px;">Jika Anda tidak merasa meminta ini, silakan abaikan email ini.</p>
			</div>
		`, userData.FullName, otp)

		_ = utils.SendEmail(req.Email, "Kode OTP Pemulihan Kata Sandi - Archeris", emailBody)

		c.JSON(http.StatusOK, gin.H{"message": "Kode OTP telah dikirim ke email Anda"})
	}
}

// MobileVerifyOTP verifies the reset token/OTP
// @Summary Verify OTP
// @Description Verify the 6-digit OTP sent to user's email
// @Tags Mobile - Auth
// @Accept json
// @Produce json
// @Param request body MobileVerifyOTPRequest true "Email and OTP"
// @Success 200 {object} MobileResponse
// @Router /mobile/auth/verify-otp [post]
func MobileVerifyOTP(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileVerifyOTPRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak lengkap"})
			return
		}

		var resetID string
		err := db.Get(&resetID, `
			SELECT uuid FROM password_resets 
			WHERE email = ? AND otp_code = ? AND is_used = 0 AND expires_at > NOW()
			ORDER BY created_at DESC LIMIT 1
		`, req.Email, req.OTP)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode OTP tidak valid atau sudah kadaluwarsa"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "OTP berhasil diverifikasi"})
	}
}

// MobileResetPassword resets the password using OTP
// @Summary Reset Password
// @Description Reset password using verified OTP and email
// @Tags Mobile - Auth
// @Accept json
// @Produce json
// @Param request body MobileResetPasswordRequest true "Email, OTP, and New Password"
// @Success 200 {object} MobileResponse
// @Router /mobile/auth/reset-password [post]
func MobileResetPassword(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileResetPasswordRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data tidak valid atau password terlalu pendek"})
			return
		}

		var reset struct {
			UUID     string `db:"uuid"`
			UserID   string `db:"user_id"`
			UserType string `db:"user_type"`
		}
		err := db.Get(&reset, `
			SELECT uuid, user_id, user_type FROM password_resets 
			WHERE email = ? AND otp_code = ? AND is_used = 0 AND expires_at > NOW()
			ORDER BY created_at DESC LIMIT 1
		`, req.Email, req.OTP)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid atau sudah kadaluwarsa"})
			return
		}

		tableName := "archers"
		if reset.UserType == "organizer" {
			tableName = "organizers"
		} else if reset.UserType == "seller" {
			tableName = "sellers"
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan"})
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec(fmt.Sprintf("UPDATE %s SET password = ?, token_version = token_version + 1 WHERE uuid = ?", tableName), req.NewPassword, reset.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset kata sandi"})
			return
		}

		_, err = tx.Exec("UPDATE password_resets SET is_used = 1 WHERE uuid = ?", reset.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses sesi"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Kata sandi berhasil diperbarui"})
	}
}

// @Summary Mobile Logout
// @Description Logout from mobile app and log activity
// @Tags Mobile - Auth
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Router /mobile/auth/logout [post]
func MobileLogout(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if exists && userID != nil {
			utils.LogActivity(db, userID.(string), "", "mobile_logout", userType.(string), userID.(string), "User logged out via mobile", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Berhasil logout",
		})
	}
}

// @Summary Bind Google Account
// @Description Link Google account to an existing email/password account
// @Tags Mobile - Auth
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileOAuthLoginRequest true "Google ID Token"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/auth/google/bind [post]
func MobileGoogleBind(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req MobileOAuthLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID Token wajib diisi"})
			return
		}

		// Verify Google Token
		googleInfo, err := verifyGoogleToken(req.IDToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Google ID Token tidak valid", "details": err.Error()})
			return
		}

		// Link Google ID based on user type
		tableName := "archers"
		if userType == "organizer" {
			tableName = "organizers"
		} else if userType == "seller" {
			tableName = "sellers"
		}

		// Check if this Google ID is already used by anyone else
		var existingCount int
		_ = db.Get(&existingCount, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE google_id = ?", tableName), googleInfo.Sub)
		if existingCount > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Akun Google ini sudah terhubung dengan akun lain", "code": "google_already_linked"})
			return
		}

		// Update user with google_id
		_, err = db.Exec(fmt.Sprintf("UPDATE %s SET google_id = ?, avatar_url = COALESCE(avatar_url, ?), updated_at = NOW() WHERE uuid = ?", tableName), googleInfo.Sub, googleInfo.Picture, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghubungkan akun Google: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "Berhasil menghubungkan akun Google",
		})
	}
}

