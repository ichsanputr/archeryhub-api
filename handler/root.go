package handler

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// RootLogin handles login specifically for the root user
func RootLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var root struct {
			UUID     string `db:"uuid"`
			Email    string `db:"email"`
			Password string `db:"password"`
			Name     string `db:"name"`
		}

		err := db.Get(&root, "SELECT uuid, email, password, name FROM roots WHERE email = ?", req.Email)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid root credentials"})
			return
		}

		// Simple password verify (as per existing logic)
		if root.Password != req.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid root credentials"})
			return
		}

		// Generate JWT token
		token, err := generateJWT(root.UUID, root.Email, "root", "root", root.Name, "", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		setAuthCookie(c, token, 60*60*24) // 24 hours

		c.JSON(http.StatusOK, AuthResponse{
			Token: token,
			User: gin.H{
				"id":        root.UUID,
				"email":     root.Email,
				"full_name": root.Name,
				"role":      "root",
				"user_type": "root",
			},
		})
	}
}

// GetAllUsers lists all users from all tables
func GetAllUsers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type SimpleUser struct {
			UUID      string  `json:"uuid" db:"uuid"`
			Email     string  `json:"email" db:"email"`
			Name      string  `json:"name" db:"name"`
			Type      string  `json:"type" db:"type"`
			Status    string  `json:"status" db:"status"`
			CreatedAt string  `json:"created_at" db:"created_at"`
		}

		var users []SimpleUser

		// Combine all user tables
		query := `
			SELECT uuid, email, full_name as name, 'archer' as type, status, created_at FROM archers
			UNION ALL
			SELECT uuid, email, name, 'club' as type, status, created_at FROM clubs
			UNION ALL
			SELECT uuid, email, name, 'organization' as type, status, created_at FROM organizations
			UNION ALL
			SELECT uuid, email, store_name as name, 'seller' as type, status, created_at FROM sellers
			ORDER BY created_at DESC
		`

		err := db.Select(&users, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, users)
	}
}
