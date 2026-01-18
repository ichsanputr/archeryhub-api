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

		// Check if user already exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM users WHERE email = ? OR username = ?)", req.Email, req.Username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
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

		// Create user
		userID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO users (id, username, email, password, full_name, phone, role, status)
			VALUES (?, ?, ?, ?, ?, ?, 'athlete', 'active')
		`, userID, req.Username, req.Email, string(hashedPassword), req.FullName, req.Phone)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}

		// Generate JWT token
		token, err := generateJWT(userID, req.Email, "athlete")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Log activity
		utils.LogActivity(db, userID, "", "user_registered", "user", userID, "User registered: "+req.Username, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, AuthResponse{
			Token: token,
			User: gin.H{
				"id":        userID,
				"username":  req.Username,
				"email":     req.Email,
				"full_name": req.FullName,
				"role":      "athlete",
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

		var user struct {
			ID       string `db:"id"`
			Username string `db:"username"`
			Email    string `db:"email"`
			Password string `db:"password"`
			FullName string `db:"full_name"`
			Role     string `db:"role"`
			Status   string `db:"status"`
		}

		err := db.Get(&user, "SELECT id, username, email, password, full_name, role, status FROM users WHERE email = ?", req.Email)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		// Check if account is active
		if user.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active"})
			return
		}

		// Verify password (plain text comparison for development)
		if user.Password != req.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
			return
		}

		// Generate JWT token
		token, err := generateJWT(user.ID, user.Email, user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Log activity
		utils.LogActivity(db, user.ID, "", "user_logged_in", "user", user.ID, "User logged in: "+user.Username, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, AuthResponse{
			Token: token,
			User: gin.H{
				"id":        user.ID,
				"username":  user.Username,
				"email":     user.Email,
				"full_name": user.FullName,
				"role":      user.Role,
			},
		})
	}
}

// Logout handles user logout
func Logout() gin.HandlerFunc {
	return func(c *gin.Context) {
		// In a JWT-based system, logout is typically handled on the client-side
		// by removing the token. This endpoint exists for completeness.
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

		var user struct {
			ID        string  `db:"id" json:"id"`
			Username  string  `db:"username" json:"username"`
			Email     string  `db:"email" json:"email"`
			FullName  string  `db:"full_name" json:"full_name"`
			Role      string  `db:"role" json:"role"`
			AvatarURL *string `db:"avatar_url" json:"avatar_url"`
			Phone     *string `db:"phone" json:"phone"`
			Status    string  `db:"status" json:"status"`
			CreatedAt string  `db:"created_at" json:"created_at"`
		}

		err := db.Get(&user, `
			SELECT id, username, email, full_name, role, avatar_url, phone, status, created_at
			FROM users WHERE id = ?
		`, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		c.JSON(http.StatusOK, user)
	}
}

// generateJWT generates a JWT token for the user
func generateJWT(userID, email, role string) (string, error) {
	secret := []byte(os.Getenv("JWT_SECRET"))
	if len(secret) == 0 {
		// For production, you should return an error here or ensure the environment variable is set.
		// For development, a fallback is provided.
		secret = []byte("archeryhub-secret-key-change-in-production") // Fallback for dev only
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"email":   email,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 72).Unix(), // 3 days expiration
		"iat":     time.Now().Unix(),
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
