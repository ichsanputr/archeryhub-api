package handler

import (
	"archeryhub-api/models"
	"net/http"
	"time"

	"archeryhub-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetArchers returns a list of archers with optional filtering
func GetArchers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		search := c.Query("search") // search by name, code, or club
		country := c.Query("country")
		limit := c.DefaultQuery("limit", "50")
		offset := c.DefaultQuery("offset", "0")

		query := `
			SELECT 
				a.*,
				COUNT(DISTINCT tp.uuid) as total_events,
				COUNT(DISTINCT CASE WHEN t.status = 'completed' THEN tp.uuid END) as completed_events,
				MAX(t.end_date) as last_event_date
			FROM archers a
			LEFT JOIN event_participants tp ON a.uuid = tp.archer_id
			LEFT JOIN events t ON tp.event_id = t.uuid
			WHERE 1=1
		`
		args := []interface{}{}

		if status != "" {
			query += " AND a.status = ?"
			args = append(args, status)
		}

		if search != "" {
			query += " AND (a.full_name LIKE ? OR a.athlete_code LIKE ? OR a.club_id LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		if country != "" {
			query += " AND a.country = ?"
			args = append(args, country)
		}

		query += `
			GROUP BY a.uuid
			ORDER BY a.full_name
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)

		var archers []models.ArcherWithStats
		err := db.Select(&archers, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch archers", "details": err.Error()})
			return
		}

		// Get total count
		countQuery := "SELECT COUNT(*) FROM archers WHERE 1=1"
		countArgs := []interface{}{}

		if status != "" {
			countQuery += " AND status = ?"
			countArgs = append(countArgs, status)
		}

		var total int
		db.Get(&total, countQuery, countArgs...)

		c.JSON(http.StatusOK, gin.H{
			"archers": archers,
			"count":   len(archers),
			"total":   total,
		})
	}
}

// GetArcherByID returns a single archer by ID
func GetArcherByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		query := `
			SELECT 
				a.*,
				COUNT(DISTINCT tp.uuid) as total_events,
				COUNT(DISTINCT CASE WHEN t.status = 'completed' THEN tp.uuid END) as completed_events,
				MAX(t.end_date) as last_event_date
			FROM archers a
			LEFT JOIN event_participants tp ON a.uuid = tp.archer_id
			LEFT JOIN events t ON tp.event_id = t.uuid
			WHERE a.uuid = ?
			GROUP BY a.uuid
		`

		var archer models.ArcherWithStats
		err := db.Get(&archer, query, id)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer not found"})
			return
		}

		c.JSON(http.StatusOK, archer)
	}
}

// CreateArcher creates a new archer
func CreateArcher(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateArcherRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		archerID := uuid.New().String()
		now := time.Now()

		// Generate archer code if not provided
		archerCode := req.ArcherCode
		if archerCode == nil {
			code := generateArcherCode(db)
			archerCode = &code
		}

		query := `
			INSERT INTO archers (
				uuid, athlete_code, full_name, date_of_birth, gender,
				country, email, phone, address,
				emergency_contact_name, emergency_contact_phone, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)
		`

		_, err := db.Exec(query,
			archerID, archerCode, req.FullName, req.DateOfBirth,
			req.Gender, req.Country, req.Email, req.Phone,
			req.Address, req.EmergencyContact, req.EmergencyPhone, now, now,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create archer", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), "", "archer_created", "archer", archerID, "Created new archer: "+req.FullName, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":     "Archer created successfully",
			"archer_id":   archerID,
			"archer_code": archerCode,
		})
	}
}

// UpdateArcher updates an existing archer
func UpdateArcher(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req models.UpdateArcherRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Check if archer exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM archers WHERE uuid = ?)", id)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer not found"})
			return
		}

		// Build dynamic update query
		query := "UPDATE archers SET updated_at = NOW()"
		args := []interface{}{}

		if req.FullName != nil {
			query += ", full_name = ?"
			args = append(args, *req.FullName)
		}
		if req.DateOfBirth != nil {
			query += ", date_of_birth = ?"
			args = append(args, *req.DateOfBirth)
		}
		if req.Gender != nil {
			query += ", gender = ?"
			args = append(args, *req.Gender)
		}
		if req.Country != nil {
			query += ", country = ?"
			args = append(args, *req.Country)
		}
		if req.Club != nil {
			query += ", club = ?"
			args = append(args, *req.Club)
		}
		if req.Email != nil {
			query += ", email = ?"
			args = append(args, *req.Email)
		}
		if req.Phone != nil {
			query += ", phone = ?"
			args = append(args, *req.Phone)
		}
		if req.PhotoURL != nil {
			query += ", photo_url = ?"
			args = append(args, *req.PhotoURL)
		}
		if req.Status != nil {
			query += ", status = ?"
			args = append(args, *req.Status)
		}

		query += " WHERE uuid = ?"
		args = append(args, id)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update archer", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), "", "archer_updated", "archer", id, "Updated archer", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Archer updated successfully"})
	}
}

// DeleteArcher deletes an archer
func DeleteArcher(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Check if archer has any event participations
		var participationCount int
		db.Get(&participationCount, "SELECT COUNT(*) FROM event_participants WHERE archer_id = ?", id)

		if participationCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete archer with event participations"})
			return
		}

		result, err := db.Exec("DELETE FROM archers WHERE uuid = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete archer", "details": err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer not found"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), "", "archer_deleted", "archer", id, "Deleted archer", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Archer deleted successfully"})
	}
}

// Helper function to generate unique archer code
func generateArcherCode(db *sqlx.DB) string {
	// Format: ARC-YYYY-NNN (e.g., ARC-2024-001)
	year := time.Now().Year()

	var maxCode string
	query := "SELECT athlete_code FROM archers WHERE athlete_code LIKE ? ORDER BY athlete_code DESC LIMIT 1"
	err := db.Get(&maxCode, query, "ARC-"+string(rune(year))+"%")

	if err != nil || maxCode == "" {
		return "ARC-" + string(rune(year)) + "-001"
	}

	// Extract number and increment
	// This is a simplified version; in production, use proper parsing
	return "ARC-" + string(rune(year)) + "-" + string(rune(time.Now().Unix()%1000))
}
