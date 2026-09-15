package handler

import (
	"Archeris-api/utils"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// setAuthCookie sets the authentication cookie with appropriate security flags
func setAuthCookie(c *gin.Context, token string, maxAge int) {
	isProduction := os.Getenv("ENV") == "production"
	host := c.Request.Host
	
	// If we are on localhost, dev, or 127.0.0.1, never use domain/secure (unless explicitly needed)
	isLocal := strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") || strings.HasPrefix(host, "0.0.0.0")
	
	domain := ""
	secure := false
	
	if isProduction && !isLocal {
		domain = ".archeris.net"
		secure = true
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("auth_token", token, maxAge, "/", domain, secure, true)
}

type RegisterRequest struct {
	Username       string `json:"username" binding:"required"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	FullName       string `json:"full_name"`
	Phone          string `json:"phone"`
	UserType       string `json:"user_type" binding:"required"` // archer, organizer, seller
	Gender         string `json:"gender"`
	DateOfBirth    string `json:"date_of_birth"`
	City           string `json:"city"`
	BowType        string `json:"bow_type"`
	Acronym        string `json:"acronym"`
	Address        string `json:"address"`
	WhatsAppNo     string `json:"whatsapp_no"`
	ClubID         string `json:"club_id"`
	NewClubName    string `json:"new_club_name"`
	NewClubAcronym string `json:"new_club_acronym"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

// Register handles user registration
func Register(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Determine target table
		table := ""
		role := ""
		switch req.UserType {
		case "archer":
			table = "archers"
			role = "archer"
		case "organizer":
			table = "organizers"
			role = "organizer"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe user tidak valid"})
			return
		}

		// Check if user already exists in any table
		type UserStub struct {
			UUID       string `db:"uuid"`
			Source     string `db:"source"`
			IsVerified bool   `db:"is_verified"`
		}
		var existingUser UserStub

		found := false
		// Check archers first
		err := db.Get(&existingUser, `SELECT uuid, 'archer' as source, is_verified FROM archers WHERE email = ? LIMIT 1`, req.Email)
		if err == nil {
			found = true
		} else {
			// Check organizers
			err = db.Get(&existingUser, `SELECT uuid, 'organizer' as source, true as is_verified FROM organizers WHERE email = ? LIMIT 1`, req.Email)
			if err == nil {
				found = true
			} else {
				err = db.Get(&existingUser, `SELECT uuid, 'club' as source, true as is_verified FROM clubs WHERE email = ? LIMIT 1`, req.Email)
				if err == nil {
					found = true
				}
			}
		}

		userID := ""
		isUpdate := false

		// Hash password securely with bcrypt
		hashedPassword := req.Password
		if req.Password != "" {
			hBytes, hErr := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
			if hErr == nil {
				hashedPassword = string(hBytes)
			}
		}

		if found {
			// If it's an unverified archer and we're registering as an archer, allow verification
			if existingUser.Source == "archer" && !existingUser.IsVerified && req.UserType == "archer" {
				userID = existingUser.UUID
				isUpdate = true
			} else {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Email atau username sudah digunakan",
					"type":  existingUser.Source,
				})
				return
			}
		}

		if userID == "" {
			userID = uuid.New().String()
		}

		var nameField string
		if table == "archers" {
			nameField = "full_name"
		} else if table == "sellers" {
			nameField = "store_name"
		} else {
			nameField = "name"
		}

		if isUpdate {
			updateQuery := `
				UPDATE ` + table + ` 
				SET password = ?, full_name = ?, phone = ?, status = 'active', is_verified = true, updated_at = NOW()
				WHERE uuid = ?
			`
			_, err = db.Exec(updateQuery, req.Password, req.FullName, req.Phone, userID)
		} else {
			isVerified := true
			if table == "organizers" {
				whatsappNo := req.WhatsAppNo
				if whatsappNo == "" {
					whatsappNo = req.Phone
				}
				cleanUsername := utils.CleanUsername(req.Username)
				if cleanUsername == "" {
					cleanUsername = "org-" + userID[:8]
				}

				insertQuery := `
					INSERT INTO organizers (uuid, user_id, slug, email, password, name, acronym, whatsapp_no, city, address, status, subscription_plan_id, subscription_status, subscription_expires_at, quota_free, quota_standard, quota_elite)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', NULL, 'active', NULL, 20, 0, 0)
				`
				_, err = db.Exec(insertQuery, userID, userID, cleanUsername, req.Email, req.Password, req.FullName, req.Acronym, whatsappNo, req.City, req.Address)

			} else if table != "archers" {
				// For other tables (sellers, organizers, clubs)
				// Clean the username/slug
				cleanUsername := utils.CleanUsername(req.Username)
				if cleanUsername == "" {
					cleanUsername = "user-" + userID[:8]
				}

				columnName := "slug"
				insertQuery := `
					INSERT INTO ` + table + ` (uuid, user_id, ` + columnName + `, email, password, ` + nameField + `, status)
					VALUES (?, ?, ?, ?, ?, ?, 'active')
				`
				_, err = db.Exec(insertQuery, userID, userID, cleanUsername, req.Email, req.Password, req.FullName)
			} else {
				// For archers, include id and is_verified
				// Generate id (ARC-XXXX)
				var maxID sql.NullInt64
				_ = db.Get(&maxID, "SELECT MAX(CAST(SUBSTRING(id, 5) AS UNSIGNED)) FROM archers WHERE id REGEXP '^ARC-[0-9]+$'")
				nextIDNum := 1
				if maxID.Valid && maxID.Int64 > 0 {
					nextIDNum = int(maxID.Int64) + 1
				}
				athleteID := fmt.Sprintf("ARC-%04d", nextIDNum)

				// Generate username from full name
				username := utils.CleanUsername(req.FullName)
				if username == "" {
					username = "archer"
				}
				username = username + "-" + userID[:8]

				// Handle club logic
				clubID := req.ClubID
				if req.NewClubName != "" {
					newClubUUID := uuid.New().String()
					clubSlug := utils.CleanUsername(req.NewClubName)
					if clubSlug == "" {
						clubSlug = "club-" + newClubUUID[:8]
					}
					_, err = db.Exec(`
						INSERT INTO clubs (uuid, slug, name, created_at, updated_at)
						VALUES (?, ?, ?, NOW(), NOW())
					`, newClubUUID, clubSlug, req.NewClubName)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat klub baru: " + err.Error()})
						return
					}
					clubID = newClubUUID
				}

				insertQuery := `
					INSERT INTO archers (uuid, id, username, email, password, full_name, phone, status, is_verified, gender, date_of_birth, bow_type, club_id)
					VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?, ?, ?, ?, ?)
				`
				_, err = db.Exec(insertQuery, userID, athleteID, username, req.Email, hashedPassword, req.FullName, req.Phone, isVerified, req.Gender, req.DateOfBirth, req.BowType, clubID)
			}
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat akun: " + err.Error()})
			return
		}

		// Send welcome email (async, don't block registration)
		if req.UserType == "archer" && !isUpdate {
			go func() {
				_ = utils.SendArcherWelcomeEmail(req.Email, req.FullName, req.Email, req.Password)
			}()
		}

		// Generate JWT token
		name := req.FullName
		avatar := "" // New registration has no avatar yet
		orgID := ""
		if req.UserType == "organizer" {
			orgID = userID
		}
		token, err := generateJWT(userID, req.Email, role, req.UserType, name, avatar, orgID, 1)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token akses"})
			return
		}

		// Set Cookie using helper
		setAuthCookie(c, token, 60*60*24*60) // 60 days

		// Log activity (silently fail if log table doesn't exist yet)
		utils.LogActivity(db, userID, "", "user_registered", req.UserType, userID, "User registered: "+req.Username, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, AuthResponse{
			Token: token,
			User: gin.H{
				"id":         userID,
				"username":   req.FullName, // Use FullName as identifier in response if username is gone
				"full_name":  req.FullName,
				"email":      req.Email,
				"avatar_url": avatar,
				"role":       role,
				"user_type":  req.UserType,
			},
		})
	}
}

// CheckNameExists checks if a name already exists in the database for a specific user type
func CheckNameExists(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Query("name")
		userType := c.Query("type")

		if name == "" || userType == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama dan tipe wajib diisi"})
			return
		}

		table := ""
		column := ""

		switch userType {
		case "archer":
			table = "archers"
			column = "full_name"
		case "organizer":
			table = "organizers"
			column = "name"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe user tidak valid"})
			return
		}

		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM " + table + " WHERE " + column + " = ?)"
		err := db.Get(&exists, query, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan database: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"exists": exists})
	}
}

// CheckUsernameExists checks if a username/slug is already taken
func CheckUsernameExists(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Query("username")
		excludeUUID := c.Query("exclude_uuid")

		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username is required"})
			return
		}

		username = utils.CleanUsername(username)
		if username == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Username is invalid"})
			return
		}

		// check archers, organizers, sellers, clubs
		var exists bool
		var query string
		var err error

		// 1. Check archers
		if excludeUUID != "" {
			query = "SELECT EXISTS(SELECT 1 FROM archers WHERE username = ? AND uuid != ?)"
			err = db.Get(&exists, query, username, excludeUUID)
		} else {
			query = "SELECT EXISTS(SELECT 1 FROM archers WHERE username = ?)"
			err = db.Get(&exists, query, username)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}
		if exists {
			c.JSON(http.StatusOK, gin.H{"exists": true, "source": "archer"})
			return
		}

		// 2. Check organizers
		if excludeUUID != "" {
			query = "SELECT EXISTS(SELECT 1 FROM organizers WHERE slug = ? AND uuid != ?)"
			err = db.Get(&exists, query, username, excludeUUID)
		} else {
			query = "SELECT EXISTS(SELECT 1 FROM organizers WHERE slug = ?)"
			err = db.Get(&exists, query, username)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}
		if exists {
			c.JSON(http.StatusOK, gin.H{"exists": true, "source": "organizer"})
			return
		}

		// 3. Check clubs
		query = "SELECT EXISTS(SELECT 1 FROM clubs WHERE slug = ?)"
		err = db.Get(&exists, query, username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}
		if exists {
			c.JSON(http.StatusOK, gin.H{"exists": true, "source": "club"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"exists": false})
	}
}

// Login handles user authentication
func Login(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("[auth] login bind error: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if os.Getenv("ENV") == "development" {
			log.Printf("[auth] login attempt email=%q password_len=%d", req.Email, len(req.Password))
		}

		type UserResult struct {
			UUID         string  `db:"uuid"`
			ID           string  `db:"id"`
			Username     string  `db:"slug"` // Use slug for frontend username field
			Email        string  `db:"email"`
			Password     string  `db:"password"`
			FullName     string  `db:"full_name"`
			AvatarURL    *string `db:"avatar_url"`
			Role         string  `db:"role"`
			Status       string  `db:"status"`
			Type         string
			OrgUUID      string  `db:"organization_uuid"`
			TokenVersion int     `db:"token_version"`
		}

		var user UserResult
		found := false

		// COALESCE(password,'') so NULL (e.g. Google-created org/club/seller) is handled as empty
		// Check archers
		err := db.Get(&user, "SELECT uuid, COALESCE(id, uuid) as id, username as slug, email, COALESCE(password,'') as password, full_name, avatar_url, 'archer' as role, COALESCE(status,'') as status, '' as organization_uuid, token_version FROM archers WHERE email = ?", req.Email)
		if err == nil {
			user.Type = "archer"
			found = true
		} else if os.Getenv("ENV") == "development" {
			log.Printf("[auth] archers lookup failed for %q: %v", req.Email, err)
		}

		// Check organizers (Google sign-up does not set password; only Register does)
		// Use column alias "slug" so result matches UserResult (db:"slug" for Username)
		if !found {
			err = db.Get(&user, "SELECT uuid, uuid as id, slug, email, COALESCE(password,'') as password, name as full_name, avatar_url, 'organizer' as role, COALESCE(status,'') as status, uuid as organization_uuid, token_version FROM organizers WHERE email = ?", req.Email)
			if err == nil {
				user.Type = "organizer"
				found = true
			} else if os.Getenv("ENV") == "development" {
				log.Printf("[auth] organizers lookup failed for %q: %v", req.Email, err)
			}
		}

		// Check clubs (use slug so result matches UserResult)
		if !found {
			err = db.Get(&user, "SELECT uuid, uuid as id, slug, email, COALESCE(password,'') as password, name as full_name, avatar_url, 'club' as role, 'active' as status, '' as organization_uuid, token_version FROM clubs WHERE email = ?", req.Email)
			if err == nil {
				user.Type = "club"
				found = true
			}
		}

		// Scorekeepers only allowed via mobile login now

		if !found {
			if os.Getenv("ENV") == "development" {
				log.Printf("[auth] login user not found email=%q", req.Email)
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah", "code": "invalid_credentials"})
			return
		}

		// Check if account is active (NULL or empty status treated as inactive)
		if user.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Akun tidak aktif", "code": "account_inactive"})
			return
		}

		// Account created via Google has no password; tell user to use Google sign-in
		if user.Password == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Akun ini terdaftar melalui Google. Silakan login dengan Google.",
				"code":  "use_google_signin",
			})
			return
		}

		// Verify password (supports bcrypt hash and fallback plain text for legacy accounts)
		isBcryptMatch := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)) == nil
		isPlainTextMatch := user.Password == req.Password

		if !isBcryptMatch && !isPlainTextMatch {
			if os.Getenv("ENV") == "development" {
				log.Printf("[auth] login password mismatch email=%q (db_len=%d req_len=%d)", req.Email, len(user.Password), len(req.Password))
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Email atau password salah", "code": "invalid_credentials"})
			return
		}

		// Generate JWT token
		avatar := ""
		if user.AvatarURL != nil {
			avatar = utils.MaskMediaURL(*user.AvatarURL)
		}
		token, err := generateJWT(user.UUID, user.Email, user.Role, user.Type, user.FullName, avatar, user.OrgUUID, user.TokenVersion)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat token akses"})
			return
		}

		// Set Cookie using helper
		setAuthCookie(c, token, 60*60*24*60) // 60 days

		// Log activity
		utils.LogActivity(db, user.UUID, "", "user_logged_in", user.Type, user.UUID, "User logged in: "+user.Username, c.ClientIP(), c.Request.UserAgent())
		if user.Type == "scorekeeper" {
			utils.LogScorekeeperAction(db, user.UUID, user.OrgUUID, "", "web_login", "Logged in via web", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, AuthResponse{
			Token: token,
			User: gin.H{
				"uuid":       user.UUID,
				"id":         user.ID,
				"username":   user.Username,
				"full_name":  user.FullName,
				"email":      user.Email,
				"avatar_url": avatar,
				"role":       user.Role,
				"user_type":  user.Type,
			},
		})
	}
}

// Logout handles user logout by clearing the auth cookie
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Clear cookie using helper (-1 maxAge means delete)
		setAuthCookie(c, "", -1)

		c.JSON(http.StatusOK, gin.H{"message": "Berhasil logout"})
	}
}

// GetCurrentUser returns the currently authenticated user
func GetCurrentUser(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		userType, _ := c.Get("user_type")
		table := "archers"
		nameField := "full_name"

		switch userType {
		case "organizer":
			table = "organizers"
			nameField = "name"
		case "club":
			table = "clubs"
			nameField = "name"
		case "scorekeeper":
			table = "scorekeepers"
			nameField = "name"
		}

		var user struct {
			UUID         string  `db:"uuid" json:"uuid"`
			ID           string  `db:"id" json:"id"`
			Username     string  `db:"slug" json:"username"` // Use slug for username field
			Email        string  `db:"email" json:"email"`
			Slug         string  `db:"slug" json:"slug"`
			FullName     string  `db:"full_name" json:"full_name"`
			Role         string  `db:"role" json:"role"`
			AvatarURL    *string `db:"avatar_url" json:"avatar_url"`
			UserType     string  `db:"-" json:"user_type"`
			Phone        *string `db:"phone" json:"phone"`
			Bio          *string `db:"bio" json:"bio"`
			Gender       *string `db:"gender" json:"gender"`
			DateOfBirth  *string `db:"date_of_birth" json:"date_of_birth"`
			BowType      *string `db:"bow_type" json:"bow_type"`
			City         *string `db:"city" json:"city"`
			Province     *string `db:"province" json:"province"`
			ClubID       *string `db:"club_id" json:"club_id"`
			Description  *string `db:"description" json:"description"`
			StoreName    *string `db:"store_name" json:"store_name"`
			BannerURL    *string `db:"banner_url" json:"banner_url"`
			Status       string  `db:"status" json:"status"`
			CreatedAt    string  `db:"created_at" json:"created_at"`
		}

		roleSelect := "'" + userType.(string) + "' as role"
		
		// Base query parts
		idExpr := "id"
		if table == "organizers" || table == "clubs" || table == "scorekeepers" {
			idExpr = "uuid as id"
		}

		slugExpr := "slug"
		emailExpr := "email"
		if table == "scorekeepers" {
			slugExpr = "code as slug"
			emailExpr = "COALESCE(email, '') as email"
		}

		phoneExpr := "phone"
		if table == "scorekeepers" {
			phoneExpr = "NULL as phone"
		}

		statusExpr := "status"
		if table == "clubs" {
			statusExpr = "'active' as status"
		}

		query := fmt.Sprintf(`SELECT uuid, %s, %s as username, %s, %s as slug, %s as full_name, %s, avatar_url, %s, %s, created_at`, 
			idExpr, slugExpr, emailExpr, slugExpr, nameField, roleSelect, phoneExpr, statusExpr)

		if table == "archers" {
			query += ", bio, gender, date_of_birth, bow_type, city, province, club_id"
		} else if table == "sellers" {
			query += ", store_name, slug, description, banner_url"
		}
		query += " FROM " + table + " WHERE uuid = ?"
		err := db.Get(&user, query, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
			return
		}

		user.UserType = userType.(string)

		// Mask media URLs
		if user.AvatarURL != nil {
			masked := utils.MaskMediaURL(*user.AvatarURL)
			user.AvatarURL = &masked
		}
		if user.BannerURL != nil {
			masked := utils.MaskMediaURL(*user.BannerURL)
			user.BannerURL = &masked
		}

		c.JSON(http.StatusOK, user)
	}
}

// setRefreshTokenCookie sets the refresh token cookie
func setRefreshTokenCookie(c *gin.Context, token string, maxAge int) {
	isProduction := os.Getenv("ENV") == "production"
	host := c.Request.Host

	isLocal := strings.HasPrefix(host, "localhost") || strings.HasPrefix(host, "127.0.0.1") || strings.HasPrefix(host, "0.0.0.0")

	domain := ""
	secure := false

	if isProduction && !isLocal {
		domain = ".archeris.net"
		secure = true
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("refresh_token", token, maxAge, "/", domain, secure, true)
}

// generateAccessToken generates a short-lived access token (1 hour)
func generateAccessToken(userID, email, role, userType, name, avatar, orgUUID string, tokenVersion int) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("Archeris-secret-key-change-in-production")
	}

	claims := jwt.MapClaims{
		"user_id":       userID,
		"email":         email,
		"name":          name,
		"avatar":        avatar,
		"role":          role,
		"user_type":     userType,
		"org_id":        orgUUID,
		"token_version": tokenVersion,
		"token_type":    "access",
		"exp":           time.Now().Add(time.Hour * 1).Unix(), // 1 hour
		"iat":           time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// generateRefreshToken generates a long-lived refresh token (60 days)
func generateRefreshToken(userID, email, role, userType string, tokenVersion int) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("Archeris-secret-key-change-in-production")
	}

	claims := jwt.MapClaims{
		"user_id":       userID,
		"email":         email,
		"role":          role,
		"user_type":     userType,
		"token_version": tokenVersion,
		"token_type":    "refresh",
		"exp":           time.Now().Add(time.Hour * 24 * 60).Unix(), // 60 days
		"iat":           time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// generateJWT generates a JWT token for the user (backwards-compatible)
func generateJWT(userID, email, role, userType, name, avatar, orgUUID string, tokenVersion int) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("Archeris-secret-key-change-in-production")
	}

	claims := jwt.MapClaims{
		"user_id":       userID,
		"email":         email,
		"name":          name,
		"avatar":        avatar,
		"role":          role,
		"user_type":     userType,
		"org_id":        orgUUID,
		"token_version": tokenVersion,
		"exp":           time.Now().Add(time.Hour * 24 * 60).Unix(), // 60 days
		"iat":           time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// RefreshToken handles token refresh via refresh_token cookie or body
func RefreshToken(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""
		// 1. Try cookie
		if cookie, err := c.Cookie("refresh_token"); err == nil && cookie != "" {
			tokenString = cookie
		}

		// 2. Try JSON body
		if tokenString == "" {
			var bodyReq struct {
				RefreshToken string `json:"refresh_token"`
			}
			if err := c.ShouldBindJSON(&bodyReq); err == nil && bodyReq.RefreshToken != "" {
				tokenString = bodyReq.RefreshToken
			}
		}

		// 3. Try Authorization header
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			utils.Error(c, http.StatusUnauthorized, "REFRESH_TOKEN_REQUIRED", "Refresh token diperlukan")
			return
		}

		secret := []byte(os.Getenv("JWT_SECRET"))
		if len(secret) == 0 {
			secret = []byte("Archeris-secret-key-change-in-production")
		}

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return secret, nil
		})

		if err != nil || !token.Valid {
			utils.Error(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "Refresh token tidak valid atau telah kadaluarsa")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.Error(c, http.StatusUnauthorized, "INVALID_CLAIMS", "Klaim token tidak valid")
			return
		}

		userID, _ := claims["user_id"].(string)
		userType, _ := claims["user_type"].(string)
		if userType == "" {
			userType, _ = claims["role"].(string)
		}
		if userID == "" || userType == "" {
			utils.Error(c, http.StatusUnauthorized, "USER_NOT_FOUND", "Informasi user pada token tidak lengkap")
			return
		}

		// Verify user exists and is active in DB
		table := "archers"
		nameField := "full_name"
		switch userType {
		case "organizer":
			table = "organizers"
			nameField = "name"
		case "club":
			table = "clubs"
			nameField = "name"
		case "scorekeeper":
			table = "scorekeepers"
			nameField = "name"
		}

		var userInfo struct {
			UUID      string  `db:"uuid"`
			Email     string  `db:"email"`
			Name      string  `db:"name"`
			AvatarURL *string `db:"avatar_url"`
			Status    string  `db:"status"`
		}

		err = db.Get(&userInfo, fmt.Sprintf("SELECT uuid, COALESCE(email, '') as email, %s as name, avatar_url, COALESCE(status, 'active') as status FROM %s WHERE uuid = ? LIMIT 1", nameField, table), userID)
		if err != nil {
			utils.Error(c, http.StatusUnauthorized, "USER_NOT_FOUND", "Akun user tidak ditemukan")
			return
		}

		if userInfo.Status == "banned" || userInfo.Status == "suspended" || userInfo.Status == "inactive" {
			utils.Error(c, http.StatusForbidden, "ACCOUNT_SUSPENDED", "Akun Anda telah dinonaktifkan")
			return
		}

		orgID := ""
		if userType == "organizer" {
			orgID = userID
		}

		avatar := ""
		if userInfo.AvatarURL != nil {
			avatar = *userInfo.AvatarURL
		}

		newAccessToken, err := generateAccessToken(userID, userInfo.Email, userType, userType, userInfo.Name, avatar, orgID, 1)
		if err != nil {
			utils.Error(c, http.StatusInternalServerError, "TOKEN_GEN_ERROR", "Gagal menghasilkan access token baru")
			return
		}

		newRefreshToken, err := generateRefreshToken(userID, userInfo.Email, userType, userType, 1)
		if err != nil {
			utils.Error(c, http.StatusInternalServerError, "TOKEN_GEN_ERROR", "Gagal menghasilkan refresh token baru")
			return
		}

		// Set new cookies
		setAuthCookie(c, newAccessToken, 60*60*24*60)
		setRefreshTokenCookie(c, newRefreshToken, 60*60*24*60)

		c.JSON(http.StatusOK, gin.H{
			"success":       true,
			"token":         newAccessToken,
			"access_token":  newAccessToken,
			"refresh_token": newRefreshToken,
			"expires_in":    3600,
			"user": gin.H{
				"id":         userInfo.UUID,
				"email":      userInfo.Email,
				"name":       userInfo.Name,
				"full_name":  userInfo.Name,
				"avatar_url": avatar,
				"role":       userType,
				"user_type":  userType,
			},
		})
	}
}

// generateRandomToken generates a random token for various purposes
func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

