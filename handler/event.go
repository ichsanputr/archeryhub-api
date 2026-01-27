package handler

import (
	"archeryhub-api/models"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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
				COALESCE(d.name, t.type, '') as discipline_name,
				COUNT(DISTINCT tp.uuid) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, email FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.uuid = tp.event_id
			LEFT JOIN event_categories te ON t.uuid = te.event_id
			LEFT JOIN ref_disciplines d ON t.type = d.uuid OR t.type = d.code
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

		// Check if user is archer to filter events and include participant status
		userID, userExists := c.Get("user_id")
		userRole, roleExists := c.Get("role")

		if userExists && roleExists && userRole == "archer" {
			// Build archer-specific query with participant status
			query = `
			SELECT 
				t.*,
				u.full_name as organizer_name,
				u.email as organizer_email,
				COALESCE(d.name, t.type, '') as discipline_name,
				COUNT(DISTINCT tp2.uuid) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count,
				tp.accreditation_status,
				tp.payment_status,
				tp.uuid as participant_uuid
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, email FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.uuid = tp.event_id AND tp.archer_id = ?
			LEFT JOIN event_participants tp2 ON t.uuid = tp2.event_id
			LEFT JOIN event_categories te ON t.uuid = te.event_id
			LEFT JOIN ref_disciplines d ON t.type = d.uuid OR t.type = d.code
			WHERE t.uuid IN (SELECT event_id FROM event_participants WHERE archer_id = ?)
			`
			args = []interface{}{userID, userID}
			
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
			GROUP BY t.uuid, tp.accreditation_status, tp.payment_status, tp.uuid, u.full_name, u.email, d.name, t.type
			ORDER BY t.start_date DESC
			LIMIT ? OFFSET ?
			`
			args = append(args, limit, offset)
		} else {
			query += `
			GROUP BY t.uuid
			ORDER BY t.start_date DESC
			LIMIT ? OFFSET ?
			`
			args = append(args, limit, offset)
		}

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
				COALESCE(d.name, t.type, '') as discipline_name,
				COUNT(DISTINCT tp.uuid) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, email FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.uuid = tp.event_id
			LEFT JOIN event_categories te ON t.uuid = te.event_id
			LEFT JOIN ref_disciplines d ON t.type = d.uuid OR t.type = d.code
			WHERE t.uuid = ? OR t.slug = ?
			GROUP BY t.uuid
		`

		var Event models.EventWithDetails
		err := db.Get(&Event, query, id, id)

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
				uuid, code, name, short_name, venue, gmaps_link, location, country, 
				latitude, longitude, start_date, end_date, registration_deadline,
				description, banner_url, logo_url, type, num_distances, num_sessions, 
				entry_fee, max_participants, status, organizer_id, created_at, updated_at
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
			)
		`

		status := req.Status
		if status == "" {
			status = "draft"
		}

		_, err := db.Exec(query,
			EventID, req.Code, req.Name, req.ShortName, req.Venue, req.GmapLink,
			req.Location, req.Country, req.Latitude, req.Longitude,
			startDate, endDate, regDeadline,
			req.Description, req.BannerURL, req.LogoURL, req.Type, req.NumDistances, req.NumSessions,
			req.EntryFee, req.MaxParticipants,
			status, userID, now, now,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Event", "details": err.Error()})
			return
		}

		// Save categories if provided (simplified for now, expects list of category UUIDs or similar)
		// Note: The user requested single step creation, so we might skip this if the frontend doesn't send it yet.
		if len(req.Divisions) > 0 && len(req.Categories) > 0 {
			for _, divUUID := range req.Divisions {
				for _, catUUID := range req.Categories {
					catEventID := uuid.New().String()
					_, err = db.Exec(`
						INSERT INTO event_categories (
							uuid, event_id, division_uuid, category_uuid, 
							max_participants
						) VALUES (?, ?, ?, ?, ?)
					`, catEventID, EventID, divUUID, catUUID, req.MaxParticipants)
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
					INSERT INTO event_images (uuid, event_id, url, caption, alt_text, display_order, is_primary)
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
		if req.Address != nil {
			query += ", address = ?"
			args = append(args, *req.Address)
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
		if req.BannerURL != nil {
			query += ", banner_url = ?"
			args = append(args, *req.BannerURL)
		}
		if req.LogoURL != nil {
			query += ", logo_url = ?"
			args = append(args, *req.LogoURL)
		}
		if req.EntryFee != nil {
			query += ", entry_fee = ?"
			args = append(args, *req.EntryFee)
		}
		if req.MaxParticipants != nil {
			query += ", max_participants = ?"
			args = append(args, *req.MaxParticipants)
		}
		if req.RegistrationDeadline != nil {
			query += ", registration_deadline = ?"
			if (*req.RegistrationDeadline).IsZero() {
				args = append(args, nil)
			} else {
				args = append(args, (*req.RegistrationDeadline).Time)
			}
		}
		if req.Status != nil {
			query += ", status = ?"
			args = append(args, *req.Status)
		}

		query += " WHERE uuid = ?"
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

// GetEventEvents returns events for a specific event
func GetEventEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// First, resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `
			SELECT uuid FROM events WHERE uuid = ? OR slug = ?
		`, eventID, eventID)
		
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		type EventEvent struct {
			ID                  string `db:"id" json:"id"`
			EventID             string `db:"event_id" json:"event_id"`
			DivisionName        string `db:"division_name" json:"division_name"`
			DivisionID           string `db:"division_id" json:"division_id"`
			CategoryName         string `db:"category_name" json:"category_name"`
			CategoryID           string `db:"category_id" json:"category_id"`
			EventTypeName        string `db:"event_type_name" json:"event_type_name"`
			EventTypeID          string `db:"event_type_id" json:"event_type_id"`
			GenderDivisionName   string `db:"gender_division_name" json:"gender_division_name"`
			GenderDivisionID     string `db:"gender_division_id" json:"gender_division_id"`
			MaxParticipants      *int   `db:"max_participants" json:"max_participants"`
			Status               string `db:"status" json:"status"`
			CreatedAt            string `db:"created_at" json:"created_at"`
		}

		var events []EventEvent
		err = db.Select(&events, `
			SELECT 
				te.uuid as id, te.event_id, 
				te.max_participants, te.status, te.created_at,
				d.name as division_name, d.uuid as division_id,
				c.name as category_name, c.uuid as category_id,
				COALESCE(et.name, '') as event_type_name, COALESCE(te.event_type_uuid, '') as event_type_id,
				COALESCE(gd.name, '') as gender_division_name, COALESCE(te.gender_division_uuid, '') as gender_division_id
			FROM event_categories te
			JOIN ref_bow_types d ON te.division_uuid = d.uuid
			JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			WHERE te.event_id = ?
			ORDER BY d.name, c.name
		`, actualEventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event categories", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"total":  len(events),
		})
	}
}

// GetEventParticipants returns participants for a specific event with pagination
func GetEventParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		limitStr := c.DefaultQuery("limit", "10")
		offsetStr := c.DefaultQuery("offset", "0")

		limit, _ := strconv.Atoi(limitStr)
		offset, _ := strconv.Atoi(offsetStr)

		// Resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Get total count
		var total int
		err = db.Get(&total, "SELECT COUNT(*) FROM event_participants WHERE event_id = ?", actualEventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count participants"})
			return
		}

		type Participant struct {
			ID                  string  `db:"id" json:"id"`
			ArcherID            string  `db:"archer_id" json:"archer_id"`
			FullName            string  `db:"full_name" json:"full_name"`
			Email               string  `db:"email" json:"email"`
			ArcherCode          string  `db:"athlete_code" json:"archer_code"`
			Country             *string `db:"country" json:"country"`
			ClubID              *string `db:"club_id" json:"club_id"`
			ClubName            *string `db:"club_name" json:"club_name"`
			EventID             string  `db:"event_id" json:"event_id"`
			CategoryID          string  `db:"category_id" json:"category_id"`
			DivisionName        string  `db:"division_name" json:"division_name"`
			CategoryName        string  `db:"category_name" json:"category_name"`
			EventTypeName       *string `db:"event_type_name" json:"event_type_name"`
			GenderDivisionName  *string `db:"gender_division_name" json:"gender_division_name"`
			TargetNumber        *string `db:"target_number" json:"target_number"`
			Session             *int    `db:"session" json:"session"`
			Status              string  `db:"status" json:"status"`
			AccreditationStatus string  `db:"accreditation_status" json:"accreditation_status"`
			RegistrationDate    string  `db:"registration_date" json:"registration_date"`
		}

		var participants []Participant
		err = db.Select(&participants, `
			SELECT 
				tp.uuid as id, tp.archer_id, tp.event_id, tp.category_id, tp.target_number, tp.session,
				COALESCE(tp.status, 'Menunggu Acc') as status, tp.accreditation_status, tp.registration_date,
				a.full_name, COALESCE(a.email, '') as email, COALESCE(a.athlete_code, '') as athlete_code, a.country, a.club_id,
				COALESCE(cl.name, '') as club_name,
				COALESCE(d.name, '') as division_name, COALESCE(c.name, '') as category_name,
				COALESCE(et.name, '') as event_type_name, COALESCE(gd.name, '') as gender_division_name
			FROM event_participants tp
			JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			WHERE tp.event_id = ?
			ORDER BY d.name, c.name, a.full_name
			LIMIT ? OFFSET ?
		`, actualEventID, limit, offset)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"participants": participants,
			"total":        total,
			"limit":        limit,
			"offset":       offset,
		})
	}
}

// GetEventParticipant returns a single participant for an event
func GetEventParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")

		// Resolve event slug to UUID
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			fmt.Printf("[DEBUG] Event not found: %s\n", eventID)
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found", "event_id": eventID})
			return
		}

		fmt.Printf("[DEBUG] Fetching participant for event %s (ID: %s), participant %s\n", eventID, actualEventID, participantID)


		type Participant struct {
			ID                  string  `db:"id" json:"id"`
			ArcherID            string  `db:"archer_id" json:"archer_id"`
			FullName            string  `db:"full_name" json:"full_name"`
			Email               string  `db:"email" json:"email"`
			ArcherCode          string  `db:"athlete_code" json:"archer_code"`
			Country             *string `db:"country" json:"country"`
			ClubID              *string `db:"club_id" json:"club_id"`
			ClubName            *string `db:"club_name" json:"club_name"`
			EventID             string  `db:"event_id" json:"event_id"`
			CategoryID          string  `db:"category_id" json:"category_id"`
			DivisionName        string  `db:"division_name" json:"division_name"`
			CategoryName        string  `db:"category_name" json:"category_name"`
			EventTypeName       *string `db:"event_type_name" json:"event_type_name"`
			GenderDivisionName  *string `db:"gender_division_name" json:"gender_division_name"`
			TargetNumber        *string `db:"target_number" json:"target_number"`
			Session             *int    `db:"session" json:"session"`
			Status              string  `db:"status" json:"status"`
			AccreditationStatus string  `db:"accreditation_status" json:"accreditation_status"`
			PaymentAmount       float64 `db:"payment_amount" json:"payment_amount"`
			PaymentProofURLs    *string `db:"payment_proof_urls" json:"payment_proof_urls"`
			RegistrationDate    string  `db:"registration_date" json:"registration_date"`
		}

		var participant Participant
		err = db.Get(&participant, `
			SELECT 
				tp.uuid as id, tp.archer_id, tp.event_id, tp.category_id, tp.target_number, tp.session,
				tp.payment_amount, tp.payment_proof_urls,
				COALESCE(tp.status, 'Menunggu Acc') as status, tp.accreditation_status, tp.registration_date,
				a.full_name, COALESCE(a.email, '') as email, COALESCE(a.athlete_code, '') as athlete_code, a.country, a.club_id,
				COALESCE(cl.name, '') as club_name,
				COALESCE(d.name, '') as division_name, COALESCE(c.name, '') as category_name,
				COALESCE(et.name, '') as event_type_name, COALESCE(gd.name, '') as gender_division_name
			FROM event_participants tp
			JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			WHERE tp.event_id = ? AND (tp.uuid = ? OR LOWER(a.athlete_code) = LOWER(?))
			LIMIT 1
		`, actualEventID, participantID, participantID)

		if err != nil {
			fmt.Printf("[DEBUG] Participant not found error: %v\n", err)
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Participant not found", 
				"details": err.Error(), 
				"participant_id": participantID, 
				"event_id": actualEventID,
			})
			return
		}

		c.JSON(http.StatusOK, participant)
	}
}

// GetEventSchedule returns schedule items for an event
func GetEventSchedule(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var exists bool
		err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM events WHERE uuid = ? OR slug = ?)`, eventID, eventID)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		var schedules []models.EventSchedule
		err = db.Select(&schedules, `
			SELECT es.* 
			FROM event_schedule es
			JOIN events e ON es.event_id = e.uuid
			WHERE e.uuid = ? OR e.slug = ?
			ORDER BY 
				COALESCE(es.day_order, 0),
				COALESCE(es.sort_order, 0),
				es.start_time
		`, eventID, eventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event schedule", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"schedules": schedules,
			"count":     len(schedules),
		})
	}
}

// ListEventCategoryRefs returns reusable event category definitions
func ListEventCategoryRefs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var list []models.EventCategoryRef
		err := db.Select(&list, `
			SELECT 
				ecr.uuid,
				ecr.name,
				ecr.bow_type_id,
				bt.name as bow_name,
				ecr.age_group_id,
				ag.name as age_name,
				ecr.status
			FROM event_category_refs ecr
			JOIN ref_bow_types bt ON ecr.bow_type_id = bt.uuid
			JOIN ref_age_groups ag ON ecr.age_group_id = ag.uuid
			ORDER BY bt.name, ag.name, ecr.name
		`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event categories", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"categories": list,
			"total":      len(list),
		})
	}
}

// CreateEventCategoryRef creates a new reusable event category
func CreateEventCategoryRef(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name       string `json:"name" binding:"required"`
			BowTypeID  string `json:"bow_type_id" binding:"required"`
			AgeGroupID string `json:"age_group_id" binding:"required"`
			Status     string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		if req.Status == "" {
			req.Status = "active"
		}

		id := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO event_category_refs (uuid, name, bow_type_id, age_group_id, status)
			VALUES (?, ?, ?, ?, ?)
		`, id, req.Name, req.BowTypeID, req.AgeGroupID, req.Status)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

// UpdateEventCategoryRef updates an existing reusable event category
func UpdateEventCategoryRef(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var req struct {
			Name       *string `json:"name"`
			BowTypeID  *string `json:"bow_type_id"`
			AgeGroupID *string `json:"age_group_id"`
			Status     *string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		var exists bool
		if err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM event_category_refs WHERE uuid = ?)`, id); err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}

		query := "UPDATE event_category_refs SET updated_at = NOW()"
		args := []interface{}{}

		if req.Name != nil {
			query += ", name = ?"
			args = append(args, *req.Name)
		}
		if req.BowTypeID != nil {
			query += ", bow_type_id = ?"
			args = append(args, *req.BowTypeID)
		}
		if req.AgeGroupID != nil {
			query += ", age_group_id = ?"
			args = append(args, *req.AgeGroupID)
		}
		if req.Status != nil {
			query += ", status = ?"
			args = append(args, *req.Status)
		}

		query += " WHERE uuid = ?"
		args = append(args, id)

		if _, err := db.Exec(query, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Category updated"})
	}
}

// PublishEvent changes event status to published
func PublishEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		_, err := db.Exec("UPDATE events SET status = 'published' WHERE uuid = ?", eventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish event"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "event_published", "event", eventID, "Published event", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Event published successfully"})
	}
}

// RegisterParticipant registers a participant for a event
func RegisterParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			AthleteID       string   `json:"athlete_id" binding:"required"`
			EventCategoryID string   `json:"event_category_id" binding:"required"`
			PaymentAmount   float64  `json:"payment_amount"`
			PaymentProofURLs []string `json:"payment_proof_urls"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve event slug to UUID
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

	// Check if already registered
	var exists bool
	err = db.Get(&exists, `
		SELECT EXISTS(SELECT 1 FROM event_participants 
		WHERE event_id = ? AND archer_id = ? AND category_id = ?)
	`, actualEventID, req.AthleteID, req.EventCategoryID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
		return
	}

	if exists {
		c.JSON(http.StatusConflict, gin.H{"error": "Participant already registered for this event category"})
		return
	}

	// Join proof URLs with comma
	proofURLs := strings.Join(req.PaymentProofURLs, ",")

	// Insert participant
	participantID := uuid.New().String()
	_, err = db.Exec(`
		INSERT INTO event_participants 
		(uuid, event_id, archer_id, category_id, payment_amount, payment_proof_urls, status, accreditation_status)
		VALUES (?, ?, ?, ?, ?, ?, 'Menunggu Acc', 'pending')
	`, participantID, actualEventID, req.AthleteID, req.EventCategoryID, req.PaymentAmount, proofURLs)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register participant", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), actualEventID, "participant_registered", "event_participant", participantID, "Registered participant for event category: "+req.EventCategoryID, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      participantID,
			"message": "Participant registered successfully",
		})
	}
}

// CancelParticipantRegistration allows an archer to cancel their registration
func CancelParticipantRegistration(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		participantID := c.Param("participantId")
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Verify the participant belongs to the user
		var archerID string
		err := db.Get(&archerID, "SELECT archer_id FROM event_participants WHERE uuid = ?", participantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registration not found"})
			return
		}

		// Check if the participant belongs to the logged-in user
		var userArcherID string
		err = db.Get(&userArcherID, "SELECT uuid FROM archers WHERE uuid = ? OR user_id = ?", userID, userID)
		if err != nil || userArcherID != archerID {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can only cancel your own registration"})
			return
		}

		// Check if already approved - can't cancel approved registrations
		var accreditationStatus string
		err = db.Get(&accreditationStatus, "SELECT accreditation_status FROM event_participants WHERE uuid = ?", participantID)
		if err == nil && accreditationStatus == "approved" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel an approved registration. Please contact the organizer."})
			return
		}

		// Delete the participant registration
		_, err = db.Exec("DELETE FROM event_participants WHERE uuid = ?", participantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel registration"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Registration cancelled successfully"})
	}
}

// UpdateEventParticipant updates an existing event participant
func UpdateEventParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")

		var req struct {
			CategoryID          *string  `json:"category_id"`
			TargetNumber        *string  `json:"target_number"`
			Session             *int     `json:"session"`
			Status              *string  `json:"status"`
			PaymentAmount       *float64 `json:"payment_amount"`
			PaymentProofURLs    []string `json:"payment_proof_urls"`
			AccreditationStatus *string  `json:"accreditation_status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve event slug to UUID
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Check if participant exists and belongs to the event
		// support both UUID and archer code
		var actualParticipantID string
		err = db.Get(&actualParticipantID, `
			SELECT tp.uuid FROM event_participants tp
			JOIN archers a ON tp.archer_id = a.uuid
			WHERE tp.event_id = ? AND (tp.uuid = ? OR LOWER(a.athlete_code) = LOWER(?))
			LIMIT 1
		`, actualEventID, participantID, participantID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found"})
			return
		}

		// Build dynamic update query
		query := "UPDATE event_participants SET updated_at = NOW()"
		args := []interface{}{}

		if req.CategoryID != nil {
			// Verify category exists and belongs to event
			var categoryExists bool
			err = db.Get(&categoryExists, `
				SELECT EXISTS(SELECT 1 FROM event_categories 
				WHERE uuid = ? AND event_id = ?)
			`, *req.CategoryID, actualEventID)
			if err != nil || !categoryExists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid category for this event"})
				return
			}
			query += ", category_id = ?"
			args = append(args, *req.CategoryID)
		}
		if req.TargetNumber != nil {
			query += ", target_number = ?"
			args = append(args, *req.TargetNumber)
		}
		if req.Session != nil {
			query += ", session = ?"
			args = append(args, *req.Session)
		}
		if req.Status != nil {
			query += ", status = ?"
			args = append(args, *req.Status)
		}
		if req.PaymentAmount != nil {
			query += ", payment_amount = ?"
			args = append(args, *req.PaymentAmount)
		}
		if len(req.PaymentProofURLs) > 0 {
			proofURLs := strings.Join(req.PaymentProofURLs, ",")
			query += ", payment_proof_urls = ?"
			args = append(args, proofURLs)
		}
		if req.AccreditationStatus != nil {
			query += ", accreditation_status = ?"
			args = append(args, *req.AccreditationStatus)
		}


		if len(args) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "No changes to save"})
			return
		}

		query += " WHERE uuid = ? AND event_id = ?"
		args = append(args, actualParticipantID, actualEventID)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update participant", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), actualEventID, "participant_updated", "event_participant", actualParticipantID, "Updated participant", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Participant updated successfully"})
	}
}

// CreateEventCategories adds categories to an existing event in batch
func CreateEventCategories(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			Divisions []string `json:"divisions" binding:"required"`
			Categories []string `json:"categories" binding:"required"`
			MaxParticipants int    `json:"max_participants"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Check if event exists
		var exists bool
		err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM events WHERE uuid = ?)`, eventID)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		count := 0
		for _, divUUID := range req.Divisions {
			for _, catUUID := range req.Categories {
				// Check if combination already exists
				var catExists bool
				err = db.Get(&catExists, `
					SELECT EXISTS(SELECT 1 FROM event_categories 
					WHERE event_id = ? AND division_uuid = ? AND category_uuid = ?)
				`, eventID, divUUID, catUUID)
				
				if err != nil || catExists {
					continue
				}

				catEventID := uuid.New().String()
				_, err = db.Exec(`
					INSERT INTO event_categories (
						uuid, event_id, division_uuid, category_uuid, 
						max_participants
					) VALUES (?, ?, ?, ?, ?)
				`, catEventID, eventID, divUUID, catUUID, req.MaxParticipants)
				
				if err == nil {
					count++
				}
			}
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "categories_created", "event", eventID, fmt.Sprintf("Created %d categories in batch", count), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"message": fmt.Sprintf("Successfully created %d categories", count),
			"count":   count,
		})
	}
}

// CreateEventCategory creates a single event category
func CreateEventCategory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			DivisionUUID      string `json:"division_uuid" binding:"required"`
			CategoryUUID      string `json:"category_uuid" binding:"required"`
			EventTypeUUID     string `json:"event_type_uuid" binding:"required"`
			GenderDivisionUUID string `json:"gender_division_uuid" binding:"required"`
			MaxParticipants   *int   `json:"max_participants"`
			Status            string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Check if combination already exists
		var catExists bool
		err = db.Get(&catExists, `
			SELECT EXISTS(SELECT 1 FROM event_categories 
			WHERE event_id = ? AND division_uuid = ? AND category_uuid = ? AND event_type_uuid = ? AND gender_division_uuid = ?)
		`, actualEventID, req.DivisionUUID, req.CategoryUUID, req.EventTypeUUID, req.GenderDivisionUUID)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check category", "details": err.Error()})
			return
		}
		
		if catExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Category already exists for this event"})
			return
		}

		status := req.Status
		if status == "" {
			status = "active"
		}

		catEventID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO event_categories (
				uuid, event_id, division_uuid, category_uuid, event_type_uuid, gender_division_uuid,
				max_participants, status
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, catEventID, actualEventID, req.DivisionUUID, req.CategoryUUID, req.EventTypeUUID, req.GenderDivisionUUID, req.MaxParticipants, status)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "category_created", "event_category", catEventID, "Created event category", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      catEventID,
			"message": "Category created successfully",
		})
	}
}

// UpdateEventCategory updates a single event category
func UpdateEventCategory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Param("categoryId")

		var req struct {
			DivisionUUID       *string `json:"division_uuid"`
			CategoryUUID       *string `json:"category_uuid"`
			EventTypeUUID      *string `json:"event_type_uuid"`
			GenderDivisionUUID *string `json:"gender_division_uuid"`
			MaxParticipants    *int    `json:"max_participants"`
			Status             *string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Check if category exists and belongs to event
		var exists bool
		err = db.Get(&exists, `
			SELECT EXISTS(SELECT 1 FROM event_categories 
			WHERE uuid = ? AND event_id = ?)
		`, categoryID, actualEventID)
		
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}

		// Build dynamic update query
		query := "UPDATE event_categories SET updated_at = NOW()"
		args := []interface{}{}

		if req.DivisionUUID != nil {
			query += ", division_uuid = ?"
			args = append(args, *req.DivisionUUID)
		}
		if req.CategoryUUID != nil {
			query += ", category_uuid = ?"
			args = append(args, *req.CategoryUUID)
		}
		if req.EventTypeUUID != nil {
			query += ", event_type_uuid = ?"
			args = append(args, *req.EventTypeUUID)
		}
		if req.GenderDivisionUUID != nil {
			query += ", gender_division_uuid = ?"
			args = append(args, *req.GenderDivisionUUID)
		}
		if req.MaxParticipants != nil {
			query += ", max_participants = ?"
			args = append(args, *req.MaxParticipants)
		} else if req.MaxParticipants == nil {
			// Allow setting to NULL
			query += ", max_participants = NULL"
		}
		if req.Status != nil {
			query += ", status = ?"
			args = append(args, *req.Status)
		}

		query += " WHERE uuid = ? AND event_id = ?"
		args = append(args, categoryID, actualEventID)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update category", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "category_updated", "event_category", categoryID, "Updated event category", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Category updated successfully"})
	}
}

// DeleteEventCategory deletes a single event category
func DeleteEventCategory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Param("categoryId")

		// Resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Check if category exists and belongs to event
		var exists bool
		err = db.Get(&exists, `
			SELECT EXISTS(SELECT 1 FROM event_categories 
			WHERE uuid = ? AND event_id = ?)
		`, categoryID, actualEventID)
		
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
			return
		}

		// Check if category has participants
		var hasParticipants bool
		err = db.Get(&hasParticipants, `
			SELECT EXISTS(SELECT 1 FROM event_participants 
			WHERE category_id = ?)
		`, categoryID)
		
		if err == nil && hasParticipants {
			c.JSON(http.StatusConflict, gin.H{"error": "Cannot delete category with existing participants"})
			return
		}

		_, err = db.Exec("DELETE FROM event_categories WHERE uuid = ? AND event_id = ?", categoryID, actualEventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "category_deleted", "event_category", categoryID, "Deleted event category", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
	}
}

// GetEventImages returns all images for an event
func GetEventImages(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		type EventImage struct {
			UUID         string `db:"uuid" json:"id"`
			EventID      string `db:"event_id" json:"event_id"`
			URL          string `db:"url" json:"url"`
			Caption      *string `db:"caption" json:"caption"`
			AltText      *string `db:"alt_text" json:"alt_text"`
			DisplayOrder int    `db:"display_order" json:"display_order"`
			IsPrimary    bool   `db:"is_primary" json:"is_primary"`
			CreatedAt    string `db:"created_at" json:"created_at"`
		}

		var images []EventImage
		err := db.Select(&images, `
			SELECT uuid, event_id, url, caption, alt_text, display_order, is_primary, created_at
			FROM event_images
			WHERE event_id = ?
			ORDER BY display_order, created_at
		`, eventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event images", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"images": images,
			"count":  len(images),
		})
	}
}

// UpdateEventImages updates event images (replaces all)
func UpdateEventImages(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		var req struct {
			Images []struct {
				URL          string  `json:"url" binding:"required"`
				Caption      *string `json:"caption"`
				AltText      *string `json:"alt_text"`
				DisplayOrder int     `json:"display_order"`
				IsPrimary    bool    `json:"is_primary"`
			} `json:"images"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Delete existing images
		_, err := db.Exec("DELETE FROM event_images WHERE event_id = ?", eventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete existing images", "details": err.Error()})
			return
		}

		// Insert new images
		for i, img := range req.Images {
			imageID := uuid.New().String()
			displayOrder := img.DisplayOrder
			if displayOrder == 0 {
				displayOrder = i
			}
			_, err = db.Exec(`
				INSERT INTO event_images (uuid, event_id, url, caption, alt_text, display_order, is_primary)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, imageID, eventID, img.URL, img.Caption, img.AltText, displayOrder, img.IsPrimary)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save event image", "details": err.Error()})
				return
			}
		}

		// Log activity
		utils.LogActivity(db, userID.(string), eventID, "event_images_updated", "event", eventID, fmt.Sprintf("Updated %d event images", len(req.Images)), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{
			"message": "Event images updated successfully",
			"count":   len(req.Images),
		})
	}
}

// GetEventTeams returns teams for a specific event
func GetEventTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventIDParam := c.Param("id") // This can be UUID or slug

		// Resolve eventIDParam to actual event UUID
		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ?", eventIDParam, eventIDParam)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		categoryID := c.Query("category_id")

		query := `
			SELECT t.uuid, t.team_name, t.country_code, t.country_name, t.status, 
			       t.total_score, t.total_x_count, t.created_at,
			       COUNT(tm.uuid) as member_count 
			FROM teams t
			LEFT JOIN team_members tm ON t.uuid = tm.team_id
			WHERE t.event_id = ?
		`
		args := []interface{}{eventUUID}

		if categoryID != "" {
			query += " AND t.category_id = ?"
			args = append(args, categoryID)
		}

		query += " GROUP BY t.uuid ORDER BY t.total_score DESC, t.total_x_count DESC"

		type Team struct {
			ID          string  `db:"uuid" json:"id"`
			TeamName    string  `db:"team_name" json:"team_name"`
			CountryCode *string `db:"country_code" json:"country_code"`
			CountryName *string `db:"country_name" json:"country_name"`
			Status      string  `db:"status" json:"status"`
			TotalScore  *int    `db:"total_score" json:"total_score"`
			TotalXCount *int    `db:"total_x_count" json:"total_x_count"`
			MemberCount int     `db:"member_count" json:"member_count"`
			CreatedAt   string  `db:"created_at" json:"created_at"`
		}

		var teams []Team
		err = db.Select(&teams, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"teams": teams,
			"total": len(teams),
		})
	}
}
