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

// GetTournaments returns a list of tournaments
func GetTournaments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		search := c.Query("search")
		limit := c.DefaultQuery("limit", "50")
		offset := c.DefaultQuery("offset", "0")

		query := `
			SELECT 
				t.*,
				u.full_name as organizer_name,
				u.email as organizer_email,
				COUNT(DISTINCT tp.id) as participant_count,
				COUNT(DISTINCT te.id) as event_count
			FROM tournaments t
			LEFT JOIN users u ON t.organizer_id = u.id
			LEFT JOIN tournament_participants tp ON t.id = tp.tournament_id
			LEFT JOIN tournament_events te ON t.id = te.tournament_id
			WHERE 1=1
		`
		args := []interface{}{}

		if status != "" {
			query += ` AND t.status = ?`
			args = append(args, status)
		}

		if search != "" {
			query += ` AND (t.name LIKE ? OR t.code LIKE ? OR t.location LIKE ?)`
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		query += `
			GROUP BY t.id
			ORDER BY t.start_date DESC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)

		var tournaments []models.TournamentWithDetails
		err := db.Select(&tournaments, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tournaments", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"tournaments": tournaments,
			"count":       len(tournaments),
		})
	}
}

// GetTournamentByID returns a single tournament by ID
func GetTournamentByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		query := `
			SELECT 
				t.*,
				u.full_name as organizer_name,
				u.email as organizer_email,
				COUNT(DISTINCT tp.id) as participant_count,
				COUNT(DISTINCT te.id) as event_count
			FROM tournaments t
			LEFT JOIN users u ON t.organizer_id = u.id
			LEFT JOIN tournament_participants tp ON t.id = tp.tournament_id
			LEFT JOIN tournament_events te ON t.id = te.tournament_id
			WHERE t.id = ?
			GROUP BY t.id
		`

		var tournament models.TournamentWithDetails
		err := db.Get(&tournament, query, id)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tournament not found"})
			return
		}

		c.JSON(http.StatusOK, tournament)
	}
}

// CreateTournament creates a new tournament
func CreateTournament(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateTournamentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Get user ID from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		tournamentID := uuid.New().String()
		now := time.Now()

		query := `
			INSERT INTO tournaments (
				id, code, name, short_name, venue, location, country, 
				latitude, longitude, start_date, end_date, description, 
				banner_url, logo_url, type, num_distances, num_sessions, 
				status, organizer_id, created_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft', ?, ?, ?
			)
		`

		_, err := db.Exec(query,
			tournamentID, req.Code, req.Name, req.ShortName, req.Venue,
			req.Location, req.Country, req.Latitude, req.Longitude,
			req.StartDate, req.EndDate, req.Description, req.BannerURL,
			req.LogoURL, req.Type, req.NumDistances, req.NumSessions,
			userID, now, now,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tournament", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ = c.Get("user_id")
		utils.LogActivity(db, userID.(string), tournamentID, "tournament_created", "tournament", tournamentID, "Created new tournament: "+req.Name, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"message":       "Tournament created successfully",
			"tournament_id": tournamentID,
		})
	}
}

// UpdateTournament updates an existing tournament
func UpdateTournament(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req models.UpdateTournamentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Check if tournament exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM tournaments WHERE id = ?)", id)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tournament not found"})
			return
		}

		// Build dynamic update query
		query := "UPDATE tournaments SET updated_at = NOW()"
		args := []interface{}{}

		if req.Name != nil {
			query += ", name = ?"
			args = append(args, *req.Name)
		}
		if req.ShortName != nil {
			query += ", short_name = ?"
			args = append(args, *req.ShortName)
		}
		if req.Venue != nil {
			query += ", venue = ?"
			args = append(args, *req.Venue)
		}
		if req.Location != nil {
			query += ", location = ?"
			args = append(args, *req.Location)
		}
		if req.Country != nil {
			query += ", country = ?"
			args = append(args, *req.Country)
		}
		if req.StartDate != nil {
			query += ", start_date = ?"
			args = append(args, *req.StartDate)
		}
		if req.EndDate != nil {
			query += ", end_date = ?"
			args = append(args, *req.EndDate)
		}
		if req.Description != nil {
			query += ", description = ?"
			args = append(args, *req.Description)
		}
		if req.Status != nil {
			query += ", status = ?"
			args = append(args, *req.Status)
		}

		query += " WHERE id = ?"
		args = append(args, id)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update tournament", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), id, "tournament_updated", "tournament", id, "Updated tournament", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Tournament updated successfully"})
	}
}

// DeleteTournament deletes a tournament
func DeleteTournament(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		result, err := db.Exec("DELETE FROM tournaments WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete tournament", "details": err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tournament not found"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "tournament_deleted", "tournament", id, "Deleted tournament", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Tournament deleted successfully"})
	}
}

// These functions are now in division_category.go to avoid duplication
