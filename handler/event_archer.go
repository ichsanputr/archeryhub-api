package handler

import (
	"archeryhub-api/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CreateEventArcher creates a new event-only archer for a specific event
func CreateEventArcher(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventIDParam := c.Param("id")

		var req models.CreateEventArcherRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		if req.FullName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nama lengkap wajib diisi"})
			return
		}

		// Resolve event UUID (slug or uuid)
		var actualEventID string
		if err := db.Get(&actualEventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ? LIMIT 1", eventIDParam, eventIDParam); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		id := uuid.New().String()
		now := time.Now()

		// Insert event-only archer
		_, err := db.Exec(`
			INSERT INTO event_archers (
				uuid, event_id, full_name, username, email, phone,
				date_of_birth, gender, bow_type, city, school,
				club, club_id, address, photo_url, notes, status, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?, ?)
		`,
			id, actualEventID, req.FullName, req.Username, req.Email, req.Phone,
			req.DateOfBirth, req.Gender, req.BowType, req.City, req.School,
			req.Club, req.ClubID, req.Address, req.PhotoURL, req.Notes, now, now,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create event archer", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":       id,
			"event_id": actualEventID,
		})
	}
}

// GetEventArchers returns all event-only archers for a specific event
func GetEventArchers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventIDParam := c.Param("id")
		search := c.Query("search")

		// Resolve event UUID (slug or uuid)
		var actualEventID string
		if err := db.Get(&actualEventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ? LIMIT 1", eventIDParam, eventIDParam); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		query := `
			SELECT 
				uuid, event_id, full_name, username, email, phone,
				date_of_birth, gender, bow_type, city, school,
				club, club_id, address, photo_url, notes, status, created_at, updated_at
			FROM event_archers
			WHERE event_id = ?
		`
		args := []interface{}{actualEventID}

		if search != "" {
			query += " AND (full_name LIKE ? OR COALESCE(email, '') LIKE ? OR COALESCE(city, '') LIKE ? OR COALESCE(school, '') LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
		}

		query += " ORDER BY full_name ASC"

		var archers []models.EventArcher
		if err := db.Select(&archers, query, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event archers", "details": err.Error()})
			return
		}

		if archers == nil {
			archers = []models.EventArcher{}
		}

		c.JSON(http.StatusOK, gin.H{
			"data": archers,
		})
	}
}

// UpdateEventArcher updates an event-only archer's information
func UpdateEventArcher(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventArcherID := c.Param("eventArcherId")
		eventID := c.Param("id")

		var req struct {
			models.CreateEventArcherRequest
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Resolve event UUID (slug or uuid)
		var actualEventID string
		if err := db.Get(&actualEventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		if req.Status == "" {
			req.Status = "active"
		}

		_, err := db.Exec(`
			UPDATE event_archers SET 
				full_name = ?, username = ?, email = ?, phone = ?,
				date_of_birth = ?, gender = ?, bow_type = ?, city = ?, school = ?,
				club = ?, club_id = ?, address = ?, photo_url = ?, notes = ?, status = ?,
				updated_at = NOW()
			WHERE uuid = ? AND event_id = ?
		`,
			req.FullName, req.Username, req.Email, req.Phone,
			req.DateOfBirth, req.Gender, req.BowType, req.City, req.School,
			req.Club, req.ClubID, req.Address, req.PhotoURL, req.Notes, req.Status,
			eventArcherID, actualEventID,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update event archer", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Event archer updated successfully"})
	}
}

// DeleteEventArcher removes an event-only archer
func DeleteEventArcher(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventArcherID := c.Param("eventArcherId")
		eventID := c.Param("id")

		// Resolve event UUID (slug or uuid)
		var actualEventID string
		if err := db.Get(&actualEventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		_, err := db.Exec("DELETE FROM event_archers WHERE uuid = ? AND event_id = ?", eventArcherID, actualEventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete event archer", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Event archer deleted successfully"})
	}
}

