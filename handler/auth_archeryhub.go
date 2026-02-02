package handler

import (
	"archeryhub-api/utils"
	"crypto/rand"
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
)

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
	UserType string `json:"user_type" binding:"required"` // archer, organization, club
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
		case "organization":
			table = "organizations"
			role = "organization"
		case "club":
			table = "clubs"
			role = "club"
		case "seller":
			table = "sellers"
			role = "seller"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user type"})
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
			// Check organizations
			err = db.Get(&existingUser, `SELECT uuid, 'organization' as source, true as is_verified FROM organizations WHERE email = ? LIMIT 1`, req.Email)
			if err == nil {
				found = true
			} else {
				// Check clubs (uses slug)
				err = db.Get(&existingUser, `SELECT uuid, 'club' as source, true as is_verified FROM clubs WHERE email = ? LIMIT 1`, req.Email)
				if err == nil {
					found = true
				}
			}
		}

		userID := ""
		isUpdate := false

		if found {
			// If it's an unverified archer and we're registering as an archer, allow verification
			if existingUser.Source == "archer" && !existingUser.IsVerified && req.UserType == "archer" {
				userID = existingUser.UUID
				isUpdate = true
			} else {
				c.JSON(http.StatusConflict, gin.H{
					"error": "User with this email or username already exists",
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
			if table != "archers" {
				// For non-archers, we don't have is_verified column yet in some tables,
				// but the user only specified archer verification logic.
				columnName := "slug"
				if table == "organizations" {
					columnName = "slug"
				}
				insertQuery := `
					INSERT INTO ` + table + ` (uuid, ` + columnName + `, email, password, ` + nameField + `, phone, status)
					VALUES (?, ?, ?, ?, ?, ?, 'active')
				`
				_, err = db.Exec(insertQuery, userID, req.Username, req.Email, req.Password, req.FullName, req.Phone)
			} else {
				// For archers, include id and is_verified
				// Generate id (ARC-XXXX)
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

				// Generate username from full name
				username := strings.ToLower(req.FullName)
				username = strings.ReplaceAll(username, " ", "-")
				username = strings.ReplaceAll(username, "'", "")
				username = strings.ReplaceAll(username, ".", "")
				username = strings.ReplaceAll(username, ",", "")
				var cleaned strings.Builder
				for _, r := range username {
					if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
						cleaned.WriteRune(r)
					}
				}
				username = cleaned.String()
				username = username + "-" + userID[:8]

				insertQuery := `
					INSERT INTO archers (uuid, id, username, email, password, full_name, phone, status, is_verified)
					VALUES (?, ?, ?, ?, ?, ?, ?, 'active', ?)
				`
				_, err = db.Exec(insertQuery, userID, athleteID, username, req.Email, req.Password, req.FullName, req.Phone, isVerified)
			}
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user: " + err.Error()})
			return
		}

		// Generate JWT token
		name := req.FullName
		avatar := "" // New registration has no avatar yet
		token, err := generateJWT(userID, req.Email, role, req.UserType, name, avatar)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Name and type are required"})
			return
		}

		table := ""
		column := ""

		switch userType {
		case "archer":
			table = "archers"
			column = "full_name"
		case "organization":
			table = "organizations"
			column = "name"
		case "club":
			table = "clubs"
			column = "name"
		case "seller":
			table = "sellers"
			column = "store_name"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user type"})
			return
		}

		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM " + table + " WHERE " + column + " = ?)"
		err := db.Get(&exists, query, name)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"exists": exists})
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
			UUID      string  `db:"uuid"`
			ID        string  `db:"id"`
			Username  string  `db:"slug"` // Use slug for frontend username field
			Email     string  `db:"email"`
			Password  string  `db:"password"`
			FullName  string  `db:"full_name"`
			AvatarURL *string `db:"avatar_url"`
			Role      string  `db:"role"`
			Status    string  `db:"status"`
			Type      string
		}

		var user UserResult
		found := false

		// COALESCE(password,'') so NULL (e.g. Google-created org/club/seller) is handled as empty
		// Check archers
		err := db.Get(&user, "SELECT uuid, id, username as slug, email, COALESCE(password,'') as password, full_name, avatar_url, 'archer' as role, COALESCE(status,'') as status FROM archers WHERE email = ?", req.Email)
		if err == nil {
			user.Type = "archer"
			found = true
		}

		// Check organizations (Google sign-up does not set password; only Register does)
		// Use column alias "slug" so result matches UserResult (db:"slug" for Username)
		if !found {
			err = db.Get(&user, "SELECT uuid, uuid as id, slug, email, COALESCE(password,'') as password, name as full_name, avatar_url, 'organization' as role, COALESCE(status,'') as status FROM organizations WHERE email = ?", req.Email)
			if err == nil {
				user.Type = "organization"
				found = true
			} else if os.Getenv("ENV") == "development" {
				log.Printf("[auth] organizations lookup failed for %q: %v", req.Email, err)
			}
		}

		// Check clubs (use slug so result matches UserResult)
		if !found {
			err = db.Get(&user, "SELECT uuid, uuid as id, slug, email, COALESCE(password,'') as password, name as full_name, avatar_url, 'club' as role, COALESCE(status,'') as status FROM clubs WHERE email = ?", req.Email)
			if err == nil {
				user.Type = "club"
				found = true
			}
		}

		// Check sellers
		if !found {
			err = db.Get(&user, "SELECT uuid, uuid as id, slug, email, COALESCE(password,'') as password, store_name as full_name, avatar_url, 'seller' as role, COALESCE(status,'') as status FROM sellers WHERE email = ?", req.Email)
			if err == nil {
				user.Type = "seller"
				found = true
			}
		}

		if !found {
			if os.Getenv("ENV") == "development" {
				log.Printf("[auth] login user not found email=%q", req.Email)
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password", "code": "invalid_credentials"})
			return
		}

		// Check if account is active (NULL or empty status treated as inactive)
		if user.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active", "code": "account_inactive"})
			return
		}

		// Account created via Google has no password; tell user to use Google sign-in
		if user.Password == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "This account uses Google sign-in. Please sign in with Google.",
				"code":  "use_google_signin",
			})
			return
		}

		// Verify password (plain text comparison)
		if user.Password != req.Password {
			if os.Getenv("ENV") == "development" {
				log.Printf("[auth] login password mismatch email=%q (db_len=%d req_len=%d)", req.Email, len(user.Password), len(req.Password))
			}
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password", "code": "invalid_credentials"})
			return
		}

		// Generate JWT token
		avatar := ""
		if user.AvatarURL != nil {
			avatar = utils.MaskMediaURL(*user.AvatarURL)
		}
		token, err := generateJWT(user.UUID, user.Email, user.Role, user.Type, user.FullName, avatar)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Log activity
		utils.LogActivity(db, user.UUID, "", "user_logged_in", user.Type, user.UUID, "User logged in: "+user.Username, c.ClientIP(), c.Request.UserAgent())

		// Set cookie
		isProduction := os.Getenv("ENV") == "production"
		domain := ""
		if isProduction {
			domain = ".archeryhub.id"
		}

		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", token, 60*60*24*7, "/", domain, isProduction, true)

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
		// Get environment settings
		isProduction := os.Getenv("ENV") == "production"
		domain := ""
		if isProduction {
			domain = ".archeryhub.id"
		}

		// Clear the auth cookie by setting it to empty with expired time
		c.SetSameSite(http.SameSiteLaxMode)
		c.SetCookie("auth_token", "", -1, "/", domain, isProduction, true)

		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
	}
}

// GetCurrentUser returns the currently authenticated user
func GetCurrentUser(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		userType, _ := c.Get("user_type")
		table := "archers"
		nameField := "full_name"

		switch userType {
		case "organization":
			table = "organizations"
			nameField = "name"
		case "club":
			table = "clubs"
			nameField = "name"
		case "seller":
			table = "sellers"
			nameField = "store_name"
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

		query := `SELECT uuid, id, slug as username, email, slug, ` + nameField + ` as full_name, ` + roleSelect + `, avatar_url, phone, status, created_at`
		if table == "archers" {
			query += ", bio, gender, date_of_birth, bow_type, city, province, club_id"
		} else if table == "sellers" {
			query += ", store_name, slug, description, banner_url"
		} else {
			// organizations, clubs don't have id column yet, so use uuid as id
			query = strings.Replace(query, "id,", "uuid as id,", 1)
		}
		query += " FROM " + table + " WHERE uuid = ?"
		err := db.Get(&user, query, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
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

// generateJWT generates a JWT token for the user
func generateJWT(userID, email, role, userType, name, avatar string) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		secret = []byte("archeryhub-secret-key-change-in-production")
	}

	claims := jwt.MapClaims{
		"user_id":   userID,
		"email":     email,
		"name":      name,
		"avatar":    avatar,
		"role":      role,
		"user_type": userType,
		"exp":       time.Now().Add(time.Hour * 72).Unix(),
		"iat":       time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

// generateRandomToken generates a random token for various purposes
func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
