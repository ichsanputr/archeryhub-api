package mobile

import (
	"archeryhub-api/utils"
	"fmt"
	"net/http"
	"strings"

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

type mobileLoginUser struct {
	UUID      string  `db:"uuid"`
	ID        string  `db:"id"`
	Username  string  `db:"username"`
	Email     string  `db:"email"`
	Password  string  `db:"password"`
	FullName  string  `db:"full_name"`
	AvatarURL *string `db:"avatar_url"`
	Status    string  `db:"status"`
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
	if userType == "organization" {
		organizationUUID = user.UUID
	}

	token, err := generateJWT(user.UUID, user.Email, role, userType, user.FullName, avatar, organizationUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
		return
	}

	utils.LogActivity(db, user.UUID, "", "mobile_login", userType, user.UUID, "User logged in via mobile", c.ClientIP(), c.Request.UserAgent())

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"uuid":       user.UUID,
			"id":         user.ID,
			"username":   user.Username,
			"full_name":  user.FullName,
			"email":      user.Email,
			"avatar_url": avatar,
			"role":       role,
			"user_type":  userType,
		},
	})
}

// MobileScorekeeperLogin godoc
// @Summary      Scorekeeper login
// @Description  Login for scorekeepers using their unique code
// @Tags         Mobile - Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      MobileScorekeeperLoginRequest  true  "Scorekeeper login request"
// @Success      200      {object}  MobileLoginResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Router       /mobile/auth/scorekeeper/login [post]
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
		}

		err := db.Get(&sk, `
			SELECT sk.uuid, sk.organization_uuid, sk.code, sk.name, IFNULL(sk.email, '') as email, sk.avatar_url, COALESCE(sk.status, '') as status,
                   o.subscription_status as org_sub_status
			FROM scorekeepers sk 
            JOIN organizations o ON sk.organization_uuid = o.uuid
            WHERE sk.code = ?`, req.Code)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kode scorekeeper tidak valid", "code": "invalid_code"})
			return
		}

		if sk.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun scorekeeper tidak aktif", "code": "account_inactive"})
			return
		}

		// Check Organization Subscription
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

		token, err := generateJWT(sk.UUID, sk.Email, "scorekeeper", "scorekeeper", sk.Name, avatar, sk.OrganizationUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
			return
		}

		utils.LogActivity(db, sk.UUID, "", "mobile_login", "scorekeeper", sk.UUID, "Scorekeeper logged in via mobile", c.ClientIP(), c.Request.UserAgent())
		utils.LogScorekeeperAction(db, sk.UUID, sk.OrganizationUUID, "", "mobile_login", "Logged in via mobile app", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"uuid":       sk.UUID,
				"id":         sk.UUID,
				"username":   sk.Code,
				"full_name":  sk.Name,
				"email":      sk.Email,
				"avatar_url": avatar,
				"role":       "scorekeeper",
				"user_type":  "scorekeeper",
			},
		})
	}
}

// MobileListEvents godoc
// @Summary      List mobile events
// @Description  Get events optimized for mobile view
// @Tags         Mobile - Events
// @Produce      json
// @Param        limit   query     int     false  "Limit"
// @Param        offset  query     int     false  "Offset"
// @Param        search  query     string  false  "Search term"
// @Success      200     {object}  MobileEventsResponse
// @Failure      500     {object}  ErrorResponse

// Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡ Archer Auth Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡

// MobileArcherLogin godoc
// @Summary      Archer login
// @Description  Login for archers using email and password
// @Tags         Mobile - Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{email=string,password=string}  true  "Login credentials"
// @Success      200      {object}  MobileLoginResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Router       /mobile/auth/login [post]
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
			`SELECT uuid, id, username, email, COALESCE(password,'') as password, full_name, avatar_url, COALESCE(status,'') as status FROM archers WHERE email = ?`,
			req,
			"archer",
			"archer",
		)
	}
}

// MobileOrganizationLogin godoc
// @Summary      Organization login
// @Description  Login for organizations using email and password
// @Tags         Mobile - Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{email=string,password=string}  true  "Login credentials"
// @Success      200      {object}  MobileLoginResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Router       /mobile/auth/organization/login [post]
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
			`SELECT uuid, uuid as id, slug as username, email, COALESCE(password,'') as password, name as full_name, avatar_url, COALESCE(status,'') as status FROM organizations WHERE email = ?`,
			req,
			"organization",
			"organization",
		)
	}
}

// MobileSellerLogin godoc
// @Summary      Seller login
// @Description  Login for sellers using email and password
// @Tags         Mobile - Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{email=string,password=string}  true  "Login credentials"
// @Success      200      {object}  MobileLoginResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Router       /mobile/auth/seller/login [post]
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
			`SELECT uuid, uuid as id, slug as username, email, COALESCE(password,'') as password, store_name as full_name, avatar_url, COALESCE(status,'') as status FROM sellers WHERE email = ?`,
			req,
			"seller",
			"seller",
		)
	}
}

// MobileArcherRegister godoc
// @Summary      Archer registration
// @Description  Register a new archer account
// @Tags         Mobile - Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object  true  "Registration info"
// @Success      201      {object}  MobileLoginResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse
// @Router       /mobile/auth/register [post]
func MobileArcherRegister(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=6"`
			FullName    string `json:"full_name" binding:"required"`
			Phone       string `json:"phone"`
			Gender      string `json:"gender"`
			DateOfBirth string `json:"date_of_birth"`
			City        string `json:"city"`
			BowType     string `json:"bow_type"`
		}
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

		token, err := generateJWT(userID, req.Email, "archer", "archer", req.FullName, "", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"token": token,
			"user": gin.H{
				"uuid":      userID,
				"id":        athleteID,
				"username":  username,
				"full_name": req.FullName,
				"email":     req.Email,
				"role":      "archer",
				"user_type": "archer",
			},
		})
	}
}
