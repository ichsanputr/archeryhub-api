package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"time"

	"archeryhub-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
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
		var exists bool
		query := "SELECT EXISTS(SELECT 1 FROM archers WHERE email = ? OR username = ?) " +
			"OR EXISTS(SELECT 1 FROM organizations WHERE email = ? OR username = ?) " +
			"OR EXISTS(SELECT 1 FROM clubs WHERE email = ? OR username = ?) " +
			"OR EXISTS(SELECT 1 FROM sellers WHERE email = ? OR username = ?)"
		err := db.Get(&exists, query, req.Email, req.Username, req.Email, req.Username, req.Email, req.Username, req.Email, req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "User with this email or username already exists"})
			return
		}

		// Hash password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}

		// Create entity
		userID := uuid.New().String()
		var nameField string
		if table == "archers" {
			nameField = "full_name"
		} else if table == "sellers" {
			nameField = "store_name"
		} else {
			nameField = "name"
		}

		insertQuery := `
			INSERT INTO ` + table + ` (id, username, email, password, ` + nameField + `, phone, role, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'active')
		`
		_, err = db.Exec(insertQuery, userID, req.Username, req.Email, string(hashedPassword), req.FullName, req.Phone, role)

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
				"username":   req.Username,
				"full_name":  req.FullName,
				"email":      req.Email,
				"avatar_url": avatar,
				"role":       role,
				"user_type":  req.UserType,
			},
		})
	}
}

// Login handles user authentication
func Login(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		type UserResult struct {
			UUID      string  `db:"uuid"`
			Username  string  `db:"username"`
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

		// Check archers
		err := db.Get(&user, "SELECT uuid, username, email, password, full_name, avatar_url, role, status FROM archers WHERE email = ?", req.Email)
		if err == nil {
			user.Type = "archer"
			found = true
		}

		// Check organizations
		if !found {
			err = db.Get(&user, "SELECT uuid, username, email, password, name as full_name, avatar_url, role, status FROM organizations WHERE email = ?", req.Email)
			if err == nil {
				user.Type = "organization"
				found = true
			}
		}

		// Check clubs
		if !found {
			err = db.Get(&user, "SELECT uuid, username, email, password, name as full_name, avatar_url, role, status FROM clubs WHERE email = ?", req.Email)
			if err == nil {
				user.Type = "club"
				found = true
			}
		}

		// Check sellers
		if !found {
			err = db.Get(&user, "SELECT uuid, username, email, password, store_name as full_name, avatar_url, role, status FROM sellers WHERE email = ?", req.Email)
			if err == nil {
				user.Type = "seller"
				found = true
			}
		}

		if !found {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		// Check if account is active
		if user.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
			return
		}

		// Verify password using bcrypt
		err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
		if err != nil {
			// Fallback check for plain text (for migrated/legacy users during transition)
			if user.Password != req.Password {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
				return
			}
		}

		// Generate JWT token
		avatar := ""
		if user.AvatarURL != nil {
			avatar = *user.AvatarURL
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
				"id":         user.UUID,
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
			ID        string  `db:"uuid" json:"id"`
			Username  string  `db:"username" json:"username"`
			Email     string  `db:"email" json:"email"`
			FullName  string  `db:"full_name" json:"full_name"`
			Role      string  `db:"role" json:"role"`
			AvatarURL *string `db:"avatar_url" json:"avatar_url"`
			UserType  string  `db:"-" json:"user_type"`
			Phone     *string `db:"phone" json:"phone"`
			Status    string  `db:"status" json:"status"`
			CreatedAt string  `db:"created_at" json:"created_at"`
		}

		query := `SELECT uuid, username, email, ` + nameField + ` as full_name, role, avatar_url, phone, status, created_at FROM ` + table + ` WHERE uuid = ?`
		err := db.Get(&user, query, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		user.UserType = userType.(string)

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
