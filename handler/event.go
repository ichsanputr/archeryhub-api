package handler

import (
	"archeryhub-api/models"
	"fmt"
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

		// Handle dates: if zero time, use nil (NULL in DB)
		var startDate, endDate, regDeadline interface{}
		if !req.StartDate.IsZero() {
			startDate = req.StartDate.Time
		}
		if !req.EndDate.IsZero() {
			endDate = req.EndDate.Time
		}
		if !req.RegistrationDeadline.IsZero() {
			regDeadline = req.RegistrationDeadline.Time
		}

		query := `
			INSERT INTO events (
				id, code, name, short_name, venue, gmaps_link, location, country, 
				latitude, longitude, start_date, end_date, registration_deadline,
				description, banner_url, logo_url, type, num_sessions, 
				entry_fee, max_participants, status, organizer_id, created_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft', ?, ?, ?
			)
		`

		_, err := db.Exec(query,
			EventID, req.Code, req.Name, req.ShortName, req.Venue, req.GmapLink,
			req.Location, req.Country, req.Latitude, req.Longitude,
			startDate, endDate, regDeadline,
			req.Description, req.BannerURL, req.LogoURL, req.Type, req.NumSessions,
			req.EntryFee, req.MaxParticipants,
			userID, now, now,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Event", "details": err.Error()})
			return
		}

		// Save divisions and categories if provided
		if len(req.Divisions) > 0 && len(req.Categories) > 0 {
			for _, divID := range req.Divisions {
				for _, catID := range req.Categories {
					catEventID := uuid.New().String()
					_, err = db.Exec(`
						INSERT INTO event_categories (
							id, event_id, division_id, category_id, 
							max_participants, qualification_arrows, elimination_format, team_event
						) VALUES (?, ?, ?, ?, ?, ?, 'single', false)
					`, catEventID, EventID, divID, catID, req.MaxParticipants, 72)
					if err != nil {
						fmt.Printf("Error: Failed to save event category: %v\n", err)
					}
				}
			}
		}

		// Save event images if provided
		if len(req.Images) > 0 {
			for i, img := range req.Images {
				imageID := uuid.New().String()
				isPrimary := img.IsPrimary || i == 0 // First image is primary by default
				_, err = db.Exec(`
					INSERT INTO event_images (id, event_id, url, caption, alt_text, display_order, is_primary)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, imageID, EventID, img.URL, img.Caption, img.AltText, i, isPrimary)
				if err != nil {
					fmt.Printf("Error: Failed to save event image: %v\n", err)
				}
			}
		}

		// Log activity
		userID, _ = c.Get("user_id")
		utils.LogActivity(db, userID.(string), EventID, "Event_created", "Event", EventID, "Created new Event: "+req.Name, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"message": "Event created successfully",
			"id":      EventID,
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
		err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM events WHERE uuid = ?)`, id)
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
		if req.GmapLink != nil {
			query += ", gmaps_link = ?"
			args = append(args, *req.GmapLink)
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
			if (*req.StartDate).IsZero() {
				args = append(args, nil)
			} else {
				args = append(args, (*req.StartDate).Time)
			}
		}
		if req.EndDate != nil {
			query += ", end_date = ?"
			if (*req.EndDate).IsZero() {
				args = append(args, nil)
			} else {
				args = append(args, (*req.EndDate).Time)
			}
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

		result, err := db.Exec("DELETE FROM events WHERE uuid = ?", id)
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
