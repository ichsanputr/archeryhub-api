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

// GetEvents returns a list of events
func GetEvents(db *sqlx.DB) gin.HandlerFunc {
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
			FROM events t
			LEFT JOIN (
				SELECT id, name as full_name, email FROM organizations
				UNION ALL
				SELECT id, name as full_name, email FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.id = tp.event_id
			LEFT JOIN event_categories te ON t.id = te.event_id
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

		var events []models.EventWithDetails
		err := db.Select(&events, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"count":       len(events),
		})
	}
}

// GetEventByID returns a single Event by ID
func GetEventByID(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		query := `
			SELECT 
				t.*,
				u.full_name as organizer_name,
				u.email as organizer_email,
				COUNT(DISTINCT tp.id) as participant_count,
				COUNT(DISTINCT te.id) as event_count
			FROM events t
			LEFT JOIN (
				SELECT id, name as full_name, email FROM organizations
				UNION ALL
				SELECT id, name as full_name, email FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.id = tp.event_id
			LEFT JOIN event_categories te ON t.id = te.event_id
			WHERE t.id = ?
			GROUP BY t.id
		`

		var Event models.EventWithDetails
		err := db.Get(&Event, query, id)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		c.JSON(http.StatusOK, Event)
	}
}

// CreateEvent creates a new Event
func CreateEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateEventRequest
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

		EventID := uuid.New().String()
		now := time.Now()

		query := `
			INSERT INTO events (
				id, code, name, short_name, venue, location, country, 
				latitude, longitude, start_date, end_date, description, 
				banner_url, logo_url, type, num_distances, num_sessions, 
				status, organizer_id, created_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft', ?, ?, ?
			)
		`

		_, err := db.Exec(query,
			EventID, req.Code, req.Name, req.ShortName, req.Venue,
			req.Location, req.Country, req.Latitude, req.Longitude,
			req.StartDate, req.EndDate, req.Description, req.BannerURL,
			req.LogoURL, req.Type, req.NumDistances, req.NumSessions,
			userID, now, now,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Event", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ = c.Get("user_id")
		utils.LogActivity(db, userID.(string), EventID, "Event_created", "Event", EventID, "Created new Event: "+req.Name, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"message":       "Event created successfully",
			"event_id": EventID,
		})
	}
}

// UpdateEvent updates an existing Event
func UpdateEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req models.UpdateEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Check if Event exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM events WHERE id = ?)", id)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Build dynamic update query
		query := "UPDATE events SET updated_at = NOW()"
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Event", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), id, "Event_updated", "Event", id, "Updated Event", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Event updated successfully"})
	}
}

// DeleteEvent deletes a Event
func DeleteEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		result, err := db.Exec("DELETE FROM events WHERE id = ?", id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete Event", "details": err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "Event_deleted", "Event", id, "Deleted Event", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Event deleted successfully"})
	}
}

// These functions are now in division_category.go to avoid duplication
