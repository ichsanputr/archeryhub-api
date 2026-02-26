package handler

import (
	"archeryhub-api/models"
	"crypto/rand"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type CreateScorekeeperRequest struct {
	Name string `json:"name" binding:"required"`
}

type UpdateScorekeeperRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// generateScorekeeperCode generates a random 5-character uppercase code
func generateScorekeeperCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // Exclude confusing chars like 0, O, I, 1
	b := make([]byte, 5)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

// CreateScorekeeper creates a new scorekeeper for an organization
func CreateScorekeeper(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgUUID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "organization" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only organizations can manage scorekeepers"})
			return
		}

		var req CreateScorekeeperRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Generate unique code
		var scorekeeperCode string
		for {
			scorekeeperCode = generateScorekeeperCode()
			var exists bool
			err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM scorekeepers WHERE code = ?)", scorekeeperCode)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
				return
			}
			if !exists {
				break
			}
		}

		scorekeeperUUID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO scorekeepers (uuid, organization_uuid, code, name, status)
			VALUES (?, ?, ?, ?, 'active')
		`, scorekeeperUUID, orgUUID, scorekeeperCode, req.Name)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create scorekeeper", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Scorekeeper created successfully",
			"uuid":    scorekeeperUUID,
			"code":    scorekeeperCode,
		})
	}
}

// GetOrganizationScorekeepers returns all scorekeepers for the authenticated organization
func GetOrganizationScorekeepers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgUUID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "organization" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only organizations can access this"})
			return
		}

		var scorekeepers []models.Scorekeeper
		err := db.Select(&scorekeepers, "SELECT uuid, organization_uuid, code, name, avatar_url, status, created_at FROM scorekeepers WHERE organization_uuid = ? ORDER BY created_at DESC", orgUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scorekeepers", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"scorekeepers": scorekeepers})
	}
}

// UpdateScorekeeper updates a scorekeeper's information
func UpdateScorekeeper(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgUUID, _ := c.Get("user_id")
		scorekeeperID := c.Param("id")

		var req UpdateScorekeeperRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify ownership
		var ownerOrg string
		err := db.Get(&ownerOrg, "SELECT organization_uuid FROM scorekeepers WHERE uuid = ?", scorekeeperID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scorekeeper not found"})
			return
		}
		if ownerOrg != orgUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to update this scorekeeper"})
			return
		}

		_, err = db.Exec("UPDATE scorekeepers SET name = ?, status = ?, updated_at = NOW() WHERE uuid = ?", 
			req.Name, req.Status, scorekeeperID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update scorekeeper", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Scorekeeper updated successfully"})
	}
}

// DeleteScorekeeper removes a scorekeeper account
func DeleteScorekeeper(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgUUID, _ := c.Get("user_id")
		scorekeeperID := c.Param("id")

		// Verify ownership
		var ownerOrg string
		err := db.Get(&ownerOrg, "SELECT organization_uuid FROM scorekeepers WHERE uuid = ?", scorekeeperID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scorekeeper not found"})
			return
		}
		if ownerOrg != orgUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to delete this scorekeeper"})
			return
		}

		_, err = db.Exec("DELETE FROM scorekeepers WHERE uuid = ?", scorekeeperID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete scorekeeper", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Scorekeeper deleted successfully"})
	}
}
