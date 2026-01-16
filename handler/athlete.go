package handler

import (
	"archeryhub-api/models"
	"net/http"
	"time"

	"archeryhub/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetAthletes returns a list of athletes with optional filtering
func GetAthletes(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		search := c.Query("search") // search by name, code, or club
		country := c.Query("country")
		limit := c.DefaultQuery("limit", "50")
		offset := c.DefaultQuery("offset", "0")

		query := `
			SELECT 
				a.*,
				COUNT(DISTINCT tp.id) as total_events,
				COUNT(DISTINCT CASE WHEN t.status = 'completed' THEN tp.id END) as completed_events,
				MAX(t.end_date) as last_event_date
			FROM athletes a
			LEFT JOIN tournament_participants tp ON a.id = tp.athlete_id
			LEFT JOIN tournaments t ON tp.tournament_id = t.id
			WHERE 1=1
		`
		args := []interface{}{}

		if status != "" {
			query += " AND a.status = ?"
			args = append(args, status)
		}

		if search != "" {
			query += " AND (a.first_name LIKE ? OR a.last_name LIKE ? OR a.athlete_code LIKE ? OR a.club LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
		}

		if country != "" {
			query += " AND a.country = ?"
			args = append(args, country)
		}

		query += `
			GROUP BY a.id
			ORDER BY a.last_name, a.first_name
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)

		var athletes []models.AthleteWithStats
		err := db.Select(&athletes, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch athletes", "details": err.Error()})
			return
		}

		// Get total count
		countQuery := "SELECT COUNT(*) FROM athletes WHERE 1=1"
		countArgs := []interface{}{}

		if status != "" {
			countQuery += " AND status = ?"
			countArgs = append(countArgs, status)
		}

		var total int
		db.Get(&total, countQuery, countArgs...)

		c.JSON(http.StatusOK, gin.H{
			"athletes": athletes,
			"count":    len(athletes),
			"total":    total,
		})
	}
}

// GetAthleteByID returns a single athlete by ID
func GetAthleteByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		query := `
			SELECT 
				a.*,
				COUNT(DISTINCT tp.id) as total_events,
				COUNT(DISTINCT CASE WHEN t.status = 'completed' THEN tp.id END) as completed_events,
				MAX(t.end_date) as last_event_date
			FROM athletes a
			LEFT JOIN tournament_participants tp ON a.id = tp.athlete_id
			LEFT JOIN tournaments t ON tp.tournament_id = t.id
			WHERE a.id = ?
			GROUP BY a.id
		`

		var athlete models.AthleteWithStats
		err := db.Get(&athlete, query, id)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Athlete not found"})
			return
		}

		c.JSON(http.StatusOK, athlete)
	}
}

// CreateAthlete creates a new athlete
func CreateAthlete(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateAthleteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		athleteID := uuid.New().String()
		now := time.Now()

		// Generate athlete code if not provided
		athleteCode := req.AthleteCode
		if athleteCode == nil {
			code := generateAthleteCode(db)
			athleteCode = &code
		}

		query := `
			INSERT INTO athletes (
				id, athlete_code, first_name, last_name, date_of_birth, gender,
				country, club, email, phone, photo_url, address,
				emergency_contact, emergency_phone, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', ?, ?)
		`

		_, err := db.Exec(query,
			athleteID, athleteCode, req.FirstName, req.LastName, req.DateOfBirth,
			req.Gender, req.Country, req.Club, req.Email, req.Phone, req.PhotoURL,
			req.Address, req.EmergencyContact, req.EmergencyPhone, now, now,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create athlete", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), "", "athlete_created", "athlete", athleteID, "Created new athlete: "+req.FirstName+" "+req.LastName, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":      "Athlete created successfully",
			"athlete_id":   athleteID,
			"athlete_code": athleteCode,
		})
	}
}

// UpdateAthlete updates an existing athlete
func UpdateAthlete(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req models.UpdateAthleteRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Check if athlete exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM athletes WHERE id = ?)", id)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Athlete not found"})
			return
		}

		// Build dynamic update query
		query := "UPDATE athletes SET updated_at = NOW()"
		args := []interface{}{}

		if req.FirstName != nil {
			query += ", first_name = ?"
			args = append(args, *req.FirstName)
		}
		if req.LastName != nil {
			query += ", last_name = ?"
			args = append(args, *req.LastName)
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

		query += " WHERE id = ?"
		args = append(args, id)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update athlete", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), "", "athlete_updated", "athlete", id, "Updated athlete", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Athlete updated successfully"})
	}
}

// DeleteAthlete deletes an athlete
func DeleteAthlete(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		// Check if athlete has any tournament participations
		var participationCount int
		db.Get(&participationCount, "SELECT COUNT(*) FROM tournament_participants WHERE athlete_id = ?", id)

		if participationCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete athlete with tournament participations"})
			return
		}

		result, err := db.Exec("DELETE FROM athletes WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete athlete", "details": err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Athlete not found"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), "", "athlete_deleted", "athlete", id, "Deleted athlete", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Athlete deleted successfully"})
	}
}

// RegisterParticipant and GetTournamentParticipants are now in division_category.go to avoid duplication

// Helper function to generate unique athlete code
func generateAthleteCode(db *sqlx.DB) string {
	// Format: ATH-YYYY-NNN (e.g., ATH-2024-001)
	year := time.Now().Year()

	var maxCode string
	query := "SELECT athlete_code FROM athletes WHERE athlete_code LIKE ? ORDER BY athlete_code DESC LIMIT 1"
	err := db.Get(&maxCode, query, "ATH-"+string(year)+"-%")

	if err != nil || maxCode == "" {
		return "ATH-" + string(year) + "-001"
	}

	// Extract number and increment
	// This is a simplified version; in production, use proper parsing
	return "ATH-" + string(year) + "-" + string(time.Now().Unix()%1000)
}
