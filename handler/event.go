package handler

import (
	"archeryhub-api/models"
	"database/sql"
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
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		organizerID := c.Query("organizer_id")

		// Check if user is archer to filter events and include participant status
		userID, userExists := c.Get("user_id")
		userRole, roleExists := c.Get("role")

		whereClause := "WHERE 1=1"
		args := []interface{}{}

		if userExists && roleExists && userRole == "archer" && organizerID == "" {
			// Special case for archers viewing THEIR events is handled by another endpoint 
			// usually /archers/my/events, but if they hit /events, we filter by their participation
			whereClause += " AND t.uuid IN (SELECT event_id FROM event_participants WHERE archer_id = ?)"
			args = append(args, userID)
		}

		if organizerID != "" {
			whereClause += ` AND t.organizer_id = ?`
			args = append(args, organizerID)
		}

		if status != "" {
			whereClause += ` AND t.status = ?`
			args = append(args, status)
		} else if organizerID == "" {
			whereClause += ` AND t.status != 'draft'`
		}

		if search != "" {
			whereClause += ` AND (t.name LIKE ? OR t.code LIKE ? OR t.location LIKE ?)`
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		// Get total count
		var total int
		countQuery := `SELECT COUNT(*) FROM events t ` + whereClause
		err := db.Get(&total, countQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count events", "details": err.Error()})
			return
		}

		var query string
		if userExists && roleExists && userRole == "archer" && organizerID == "" {
			query = `
			SELECT 
				t.*,
				u.full_name as organizer_name,
				u.email as organizer_email,
				u.slug as organizer_slug,
				u.avatar_url as organizer_avatar_url,
				COUNT(DISTINCT tp2.uuid) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count,
				tp.payment_status,
				tp.uuid as participant_uuid
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.uuid = tp.event_id AND tp.archer_id = ?
			LEFT JOIN event_participants tp2 ON t.uuid = tp2.event_id
			LEFT JOIN event_categories te ON t.uuid = te.event_id
			` + whereClause + `
			GROUP BY t.uuid, tp.payment_status, tp.uuid, u.full_name, u.email, u.slug, u.avatar_url
			ORDER BY t.start_date DESC
			LIMIT ? OFFSET ?
			`
			// Prepend userID for the LEFT JOIN tp
			newArgs := []interface{}{userID}
			newArgs = append(newArgs, args...)
			newArgs = append(newArgs, limit, offset)
			args = newArgs
		} else {
			query = `
			SELECT 
				t.*,
				u.full_name as organizer_name,
				u.email as organizer_email,
				u.slug as organizer_slug,
				u.avatar_url as organizer_avatar_url,
				COUNT(DISTINCT tp.uuid) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.uuid = tp.event_id
			LEFT JOIN event_categories te ON t.uuid = te.event_id
			` + whereClause + `
			GROUP BY t.uuid, u.full_name, u.email, u.slug, u.avatar_url
			ORDER BY t.start_date DESC
			LIMIT ? OFFSET ?
			`
			args = append(args, limit, offset)
		}

		var events []models.EventWithDetails
		err = db.Select(&events, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events", "details": err.Error()})
			return
		}

		// Mask URLs
		for i := range events {
			if events[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*events[i].BannerURL)
				events[i].BannerURL = &masked
			}
			if events[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*events[i].LogoURL)
				events[i].LogoURL = &masked
			}
			if events[i].TechnicalGuidebookURL != nil {
				masked := utils.MaskMediaURL(*events[i].TechnicalGuidebookURL)
				events[i].TechnicalGuidebookURL = &masked
			}
			if events[i].OrganizerAvatarURL != nil {
				masked := utils.MaskMediaURL(*events[i].OrganizerAvatarURL)
				events[i].OrganizerAvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"total":  total,
			"limit":  limit,
			"offset": offset,
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
				u.avatar_url as organizer_avatar_url,
				u.slug as organizer_slug,
				COALESCE(participant_stats.participant_count, 0) as participant_count,
				COALESCE(category_stats.event_count, 0) as event_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, avatar_url, slug FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, email, avatar_url, slug FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN (
				SELECT event_id, COUNT(DISTINCT uuid) as participant_count
				FROM event_participants
				GROUP BY event_id
			) participant_stats ON t.uuid = participant_stats.event_id
			LEFT JOIN (
				SELECT event_id, COUNT(DISTINCT uuid) as event_count
				FROM event_categories
				GROUP BY event_id
			) category_stats ON t.uuid = category_stats.event_id
			WHERE t.uuid = ? OR t.slug = ?
			LIMIT 1
		`

		var Event models.EventWithDetails
		err := db.Get(&Event, query, id, id)
		if err != nil {
			// Log the error for debugging
			fmt.Printf("[GetEventByID] Error fetching event with id/slug '%s': %v\n", id, err)
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Event not found", "id": id})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event", "details": err.Error()})
			}
			return
		}

		// Check visibility
		if Event.Status == "draft" {
			// Check if user is organizer
			userID, exists := c.Get("user_id")
			isAuthorized := false
			if exists {
				// Check if userID matches organizerID
				if Event.OrganizerID != nil && *Event.OrganizerID == userID.(string) {
					isAuthorized = true
				}
				// Allow admins too
				role, _ := c.Get("role")
				if role == "admin" {
					isAuthorized = true
				}
			}

			if !isAuthorized {
				c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
				return
			}
		}

		// Mask URLs
		if Event.BannerURL != nil {
			masked := utils.MaskMediaURL(*Event.BannerURL)
			Event.BannerURL = &masked
		}
		if Event.LogoURL != nil {
			masked := utils.MaskMediaURL(*Event.LogoURL)
			Event.LogoURL = &masked
		}
		if Event.TechnicalGuidebookURL != nil {
			masked := utils.MaskMediaURL(*Event.TechnicalGuidebookURL)
			Event.TechnicalGuidebookURL = &masked
		}
		if Event.OrganizerAvatarURL != nil {
			masked := utils.MaskMediaURL(*Event.OrganizerAvatarURL)
			Event.OrganizerAvatarURL = &masked
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

		// Generate code if not provided
		if req.Code == "" {
			var lastCode string
			_ = db.Get(&lastCode, "SELECT code FROM events WHERE code LIKE 'EVT-%' ORDER BY code DESC LIMIT 1")
			nextNum := 1
			if lastCode != "" {
				// Extract number from EVT-XXXX
				parts := strings.Split(lastCode, "-")
				if len(parts) == 2 {
					fmt.Sscanf(parts[1], "%d", &nextNum)
					nextNum++
				}
			}
			req.Code = fmt.Sprintf("EVT-%04d", nextNum)
		}

		eventUUID := uuid.New().String()
		now := time.Now()

		// Generate slug from name
		slug := strings.ToLower(req.Name)
		slug = strings.ReplaceAll(slug, " ", "-")
		// Remove non-alphanumeric
		var cleanSlug strings.Builder
		for _, r := range slug {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				cleanSlug.WriteRune(r)
			}
		}
		finalSlug := cleanSlug.String() + "-" + eventUUID[:8]

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
				uuid, code, name, short_name, slug, venue, gmaps_link, location, city, 
				start_date, end_date, registration_deadline,
				description, banner_url, logo_url, location_type, num_distances, num_sessions, 
				entry_fee, status, organizer_id, created_at, updated_at,
				total_prize, technical_guidebook_url, page_settings, faq
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
			)
		`

		status := req.Status
		if status == "" {
			status = "draft"
		}

		// Use location_type if provided, otherwise fallback to type for backward compatibility
		locationType := req.LocationType
		if locationType == nil && req.Type != nil {
			locationType = req.Type
		}

		_, err := db.Exec(query,
			eventUUID, req.Code, req.Name, req.ShortName, finalSlug, req.Venue, req.GmapLink,
			req.Location, req.City,
			startDate, endDate, regDeadline,
			req.Description, utils.ExtractFilename(models.FromPtr(req.BannerURL)), utils.ExtractFilename(models.FromPtr(req.LogoURL)), locationType, req.NumDistances, req.NumSessions,
			req.EntryFee,
			status, userID, now, now,
			req.TotalPrize, utils.ExtractFilename(models.FromPtr(req.TechnicalGuidebookURL)), req.PageSettings,
			models.ToJSON(req.FAQ),
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
						) VALUES (?, ?, ?, ?, NULL)
					`, catEventID, eventUUID, divUUID, catUUID)
					if err != nil {
						// fmt.Printf("Error: Failed to save event category: %v\n", err) // Removed fmt import
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
				`, imageID, eventUUID, utils.ExtractFilename(img.URL), img.Caption, img.AltText, i, isPrimary)
				if err != nil {
					// fmt.Printf("Error: Failed to save event image: %v\n", err) // Removed fmt import
				}
			}
		}

		// Log activity
		userID, _ = c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventUUID, "Event_created", "Event", eventUUID, "Created new Event: "+req.Name, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"message": "Event created successfully",
			"id":      eventUUID,
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

		// Resolve slug to UUID if needed
		var actualID string
		err := db.Get(&actualID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}
		id = actualID

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
		if req.City != nil {
			query += ", city = ?"
			args = append(args, *req.City)
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
			args = append(args, utils.ExtractFilename(*req.BannerURL))
		}
		if req.LogoURL != nil {
			query += ", logo_url = ?"
			args = append(args, utils.ExtractFilename(*req.LogoURL))
		}
		if req.EntryFee != nil {
			query += ", entry_fee = ?"
			args = append(args, *req.EntryFee)
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
		if req.TotalPrize != nil {
			query += ", total_prize = ?"
			args = append(args, *req.TotalPrize)
		}
		if req.TechnicalGuidebookURL != nil {
			query += ", technical_guidebook_url = ?"
			args = append(args, utils.ExtractFilename(*req.TechnicalGuidebookURL))
		}
		if req.PageSettings != nil {
			query += ", page_settings = ?"
			args = append(args, *req.PageSettings)
		}
		if req.FAQ != nil {
			query += ", faq = ?"
			args = append(args, models.ToJSON(req.FAQ))
		}
		if req.LocationType != nil {
			query += ", location_type = ?"
			args = append(args, *req.LocationType)
		} else if req.Type != nil {
			// Backward compatibility: if location_type not provided but type is, use type
			query += ", location_type = ?"
			args = append(args, *req.Type)
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

		// Resolve slug to UUID if needed
		var actualID string
		err := db.Get(&actualID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		result, err := db.Exec("DELETE FROM events WHERE uuid = ?", actualID)
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
		limitStr := c.DefaultQuery("limit", "10")
		offsetStr := c.DefaultQuery("offset", "0")
		bowTypeFilter := c.Query("bow_type")

		limit, _ := strconv.Atoi(limitStr)
		offset, _ := strconv.Atoi(offsetStr)

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
			ID                 string `db:"id" json:"id"`
			EventID            string `db:"event_id" json:"event_id"`
			DivisionName       string `db:"division_name" json:"division_name"`
			DivisionID         string `db:"division_id" json:"division_id"`
			CategoryName       string `db:"category_name" json:"category_name"`
			CategoryID         string `db:"category_id" json:"category_id"`
			EventTypeName      string `db:"event_type_name" json:"event_type_name"`
			EventTypeID        string `db:"event_type_id" json:"event_type_id"`
			GenderDivisionName string `db:"gender_division_name" json:"gender_division_name"`
			GenderDivisionID   string `db:"gender_division_id" json:"gender_division_id"`
			MaxParticipants    *int   `db:"max_participants" json:"max_participants"`
			TeamSize           int    `db:"team_size" json:"team_size"`
			ParticipantCount   int    `db:"participant_count" json:"participant_count"`
			Status             string `db:"status" json:"status"`
			CreatedAt          string `db:"created_at" json:"created_at"`
		}

		whereClause := "WHERE te.event_id = ?"
		args := []interface{}{actualEventID}

		if bowTypeFilter != "" && bowTypeFilter != "all" {
			whereClause += " AND d.code = ?"
			args = append(args, bowTypeFilter)
		}

		// Get total count
		var total int
		err = db.Get(&total, `
			SELECT COUNT(*) 
			FROM event_categories te
			JOIN ref_bow_types d ON te.division_uuid = d.uuid
			`+whereClause, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count event categories", "details": err.Error()})
			return
		}

		var events []EventEvent
		query := `
			SELECT 
				te.uuid as id, te.event_id, 
				te.max_participants, te.status, te.created_at,
				CASE 
					WHEN et.code = 'mixed_team' THEN 2 
					WHEN et.code = 'team' THEN 3 
					ELSE 1 
				END as team_size,
				d.name as division_name, d.uuid as division_id,
				c.name as category_name, c.uuid as category_id,
				COALESCE(et.name, '') as event_type_name, COALESCE(te.event_type_uuid, '') as event_type_id,
				COALESCE(gd.name, '') as gender_division_name, COALESCE(te.gender_division_uuid, '') as gender_division_id,
				COALESCE(p.p_count, 0) as participant_count
			FROM event_categories te
			JOIN ref_bow_types d ON te.division_uuid = d.uuid
			JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			LEFT JOIN (
				SELECT category_id, COUNT(*) as p_count 
				FROM event_participants 
				GROUP BY category_id
			) p ON te.uuid = p.category_id
			` + whereClause + `
			ORDER BY d.name, c.name, et.name, gd.name
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)
		err = db.Select(&events, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event categories", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
	}
}

// GetEventParticipants returns participants for a specific event with pagination
func GetEventParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		limitStr := c.DefaultQuery("limit", "10")
		offsetStr := c.DefaultQuery("offset", "0")
		categoryFilter := c.Query("category")
		categoryIDFilter := c.Query("category_id")
		searchQuery := c.Query("search")

		limit, _ := strconv.Atoi(limitStr)
		offset, _ := strconv.Atoi(offsetStr)

		// Resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Build WHERE clause for filtering
		whereClause := "WHERE tp.event_id = ?"
		args := []interface{}{actualEventID}
		countArgs := []interface{}{actualEventID}

		// Filter by category_id
		if categoryIDFilter != "" {
			whereClause += " AND tp.category_id = ?"
			args = append(args, categoryIDFilter)
			countArgs = append(countArgs, categoryIDFilter)
		} else if categoryFilter != "" && categoryFilter != "Semua" {
			// Filter by category name (Compatibility)
			parts := strings.Fields(categoryFilter)
			if len(parts) >= 2 {
				divisionName := parts[0]
				genderName := parts[1]
				whereClause += " AND d.name = ? AND gd.name = ?"
				args = append(args, divisionName, genderName)
				countArgs = append(countArgs, divisionName, genderName)
			} else if len(parts) == 1 {
				// Only division filter
				whereClause += " AND d.name = ?"
				args = append(args, parts[0])
				countArgs = append(countArgs, parts[0])
			}
		}

		// Filter by search query
		if searchQuery != "" {
			searchTerm := "%" + searchQuery + "%"
			whereClause += " AND (a.full_name LIKE ? OR cl.name LIKE ?)"
			args = append(args, searchTerm, searchTerm)
			countArgs = append(countArgs, searchTerm, searchTerm)
		}

		// Get total count with filters
		countQuery := "SELECT COUNT(*) FROM event_participants tp LEFT JOIN archers a ON tp.archer_id = a.uuid LEFT JOIN clubs cl ON a.club_id = cl.uuid LEFT JOIN event_categories te ON tp.category_id = te.uuid LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid " + whereClause
		var total int
		err = db.Get(&total, countQuery, countArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count participants", "details": err.Error()})
			return
		}

		type Participant struct {
			ID                 string  `db:"id" json:"id"`
			ArcherID           *string `db:"archer_id" json:"archer_id"`
			AthleteCode        *string `db:"athlete_code" json:"athlete_code"`
			Username           *string `db:"username" json:"username"`
			FullName           string  `db:"full_name" json:"full_name"`
			Email              string  `db:"email" json:"email"`
			City               *string `db:"city" json:"city"`
			ClubID             *string `db:"club_id" json:"club_id"`
			ClubName           *string `db:"club_name" json:"club_name"`
			EventID            string  `db:"event_id" json:"event_id"`
			CategoryID         string  `db:"category_id" json:"category_id"`
			DivisionName       string  `db:"division_name" json:"division_name"`
			CategoryName       string  `db:"category_name" json:"category_name"`
			EventTypeName      *string `db:"event_type_name" json:"event_type_name"`
			GenderDivisionName *string `db:"gender_division_name" json:"gender_division_name"`
			TargetName         *string `db:"target_name" json:"target_name"`
			QRRaw              *string `db:"qr_raw" json:"qr_raw"`
			Status             string  `db:"status" json:"status"`
			AvatarURL            *string `db:"avatar_url" json:"avatar_url"`
			RegistrationDate     string  `db:"registration_date" json:"registration_date"`
			LastReregistrationAt *string `db:"last_reregistration_at" json:"last_reregistration_at"`
			TotalScore           int     `db:"total_score" json:"total_score"`
			TotalX               int     `db:"total_x" json:"total_x"`
		}

		var participants []Participant
		query := `
			SELECT 
				tp.uuid as id, tp.archer_id, tp.event_id, tp.category_id, tp.target_name, tp.qr_raw,
				COALESCE(tp.status, 'Menunggu Acc') as status, tp.registration_date, tp.last_reregistration_at,
				a.id as athlete_code,
				a.username as username,
				a.full_name as full_name,
				COALESCE(a.email, '') as email,
				a.city as city,
				a.club_id as club_id,
				a.avatar_url as avatar_url,
				COALESCE(cl.name, '') as club_name,
				COALESCE(d.name, '') as division_name, COALESCE(c.name, '') as category_name,
				COALESCE(et.name, '') as event_type_name, COALESCE(gd.name, '') as gender_division_name,
				COALESCE(scores.total_score, 0) as total_score,
				COALESCE(scores.total_x, 0) as total_x
			FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			LEFT JOIN (
				SELECT archer_uuid, SUM(total_score_end) as total_score, SUM(x_count_end) as total_x
				FROM qualification_end_scores
				GROUP BY archer_uuid
			) scores ON a.uuid = scores.archer_uuid
			` + whereClause + `
			GROUP BY tp.uuid, a.uuid, cl.uuid, te.uuid, d.uuid, c.uuid, et.uuid, gd.uuid, a.id, a.username, a.full_name, a.email, a.city, a.club_id, a.avatar_url, cl.name, d.name, c.name, et.name, gd.name, scores.total_score, scores.total_x
			ORDER BY total_score DESC, total_x DESC, a.full_name ASC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)
		err = db.Select(&participants, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants", "details": err.Error()})
			return
		}

		// Get verified and pending counts
		var verifiedCount, pendingCount int
		db.Get(&verifiedCount, "SELECT COUNT(*) FROM event_participants WHERE event_id = ? AND status = 'Terdaftar'", actualEventID)
		db.Get(&pendingCount, "SELECT COUNT(*) FROM event_participants WHERE event_id = ? AND status = 'Menunggu Acc'", actualEventID)

		// Mask avatar URLs
		for i := range participants {
			if participants[i].AvatarURL != nil {
				masked := utils.MaskMediaURL(*participants[i].AvatarURL)
				participants[i].AvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"participants":   participants,
			"total":          total,
			"verified_count": verifiedCount,
			"pending_count":  pendingCount,
			"limit":          limit,
			"offset":         offset,
		})
	}
}

// GetEventParticipant returns a single participant for an event
func GetEventParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")

		// Resolve event slug to UUID and get details for visibility check
		var event struct {
			UUID        string  `db:"uuid"`
			Status      string  `db:"status"`
			OrganizerID *string `db:"organizer_id"`
		}
		err := db.Get(&event, `SELECT uuid, status, organizer_id FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}
		actualEventID := event.UUID

		// Check visibility
		if event.Status == "draft" {
			// Check if user is organizer
			userID, exists := c.Get("user_id")
			isAuthorized := false
			if exists {
				if event.OrganizerID != nil && *event.OrganizerID == userID.(string) {
					isAuthorized = true
				}
				role, _ := c.Get("role")
				if role == "admin" {
					isAuthorized = true
				}
			}

			if !isAuthorized {
				fmt.Printf("[DEBUG] Unauthorized draft access. EventID: %s, UserID: %v, OrganizerID: %v, Exists: %v\n", event.UUID, userID, event.OrganizerID, exists)
				c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
				return
			}
		}

		fmt.Printf("[DEBUG] Fetching participant for event %s (ID: %s), participant %s\n", eventID, actualEventID, participantID)

		type Participant struct {
			ID                 string  `db:"id" json:"id"`
			AthleteCode        *string `db:"athlete_code" json:"athlete_code"`
			ArcherID           *string `db:"archer_id" json:"archer_id"`
			FullName           string  `db:"full_name" json:"full_name"`
			Username           *string `db:"username" json:"username"`
			Email              string  `db:"email" json:"email"`
			City               *string `db:"city" json:"city"`
			ClubID             *string `db:"club_id" json:"club_id"`
			ClubName           *string `db:"club_name" json:"club_name"`
			EventID            string  `db:"event_id" json:"event_id"`
			CategoryID         string  `db:"category_id" json:"category_id"`
			DivisionName       string  `db:"division_name" json:"division_name"`
			CategoryName       string  `db:"category_name" json:"category_name"`
			EventTypeName      *string `db:"event_type_name" json:"event_type_name"`
			GenderDivisionName *string `db:"gender_division_name" json:"gender_division_name"`
			TargetName         *string `db:"target_name" json:"target_name"`
			QRRaw              *string `db:"qr_raw" json:"qr_raw"`
			Status             string  `db:"status" json:"status"`
			AvatarURL          *string `db:"avatar_url" json:"avatar_url"`
			PaymentAmount      float64 `db:"payment_amount" json:"payment_amount"`
			PaymentProofURLs   *string `db:"payment_proof_urls" json:"payment_proof_urls"`
			RegistrationDate   string  `db:"registration_date" json:"registration_date"`
			IsVerified         bool    `db:"is_verified" json:"is_verified"`
		}

		var participant Participant
		err = db.Get(&participant, `
			SELECT 
				tp.uuid as id, tp.archer_id, tp.event_id, tp.category_id, tp.target_name, tp.qr_raw,
				tp.payment_amount, tp.payment_proof_urls,
				COALESCE(tp.status, 'Menunggu Acc') as status, tp.registration_date,
				a.id as athlete_code,
				a.username as username,
				a.full_name as full_name,
				COALESCE(a.email, '') as email,
				a.city as city,
				a.club_id as club_id,
				a.avatar_url as avatar_url,
				COALESCE(cl.name, '') as club_name,
				COALESCE(d.name, '') as division_name, COALESCE(c.name, '') as category_name,
				COALESCE(et.name, '') as event_type_name, COALESCE(gd.name, '') as gender_division_name,
				COALESCE(a.is_verified, 0) as is_verified
			FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			WHERE tp.event_id = ? AND (
				tp.uuid = ? OR 
				a.username = ? OR 
				a.id = ? OR
				LOWER(REPLACE(a.full_name, ' ', '-')) = LOWER(?)
			)
			ORDER BY tp.created_at DESC
			LIMIT 1
		`, actualEventID, participantID, participantID, participantID, participantID)

		if err != nil {
			fmt.Printf("[DEBUG] Participant not found in DB for Event: %s, ID: %s. Error: %v\n", actualEventID, participantID, err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":          "Participant not found",
				"details":        err.Error(),
				"participant_id": participantID,
				"event_id":       actualEventID,
				"hint":           "Make sure the participant exists for this event and the ID/Username is correct.",
			})
			return
		}

		fmt.Printf("[DEBUG] Found participant: %s (UUID: %s)\n", participant.FullName, participant.ID)

		// Mask avatar URL
		if participant.AvatarURL != nil {
			masked := utils.MaskMediaURL(*participant.AvatarURL)
			participant.AvatarURL = &masked
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

// UpdateEventSchedule updates event schedules (replaces all)
func UpdateEventSchedule(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		var req struct {
			Schedules []struct {
				ID          *string `json:"id"`
				Title       string  `json:"title" binding:"required"`
				Description *string `json:"description"`
				StartTime   string  `json:"start_time" binding:"required"`
				EndTime     *string `json:"end_time"`
				DayOrder    *int    `json:"day_order"`
				SortOrder   *int    `json:"sort_order"`
				Location    *string `json:"location"`
			} `json:"schedules" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Delete existing schedules
		_, err = db.Exec("DELETE FROM event_schedule WHERE event_id = ?", actualEventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete existing schedules", "details": err.Error()})
			return
		}

		// Insert new schedules
		for _, s := range req.Schedules {
			scheduleID := uuid.New().String()
			if s.ID != nil && *s.ID != "" {
				scheduleID = *s.ID
			}

			// Parse StartTime RFC3339
			parsedStartTime, err := time.Parse(time.RFC3339, s.StartTime)
			if err != nil {
				// Try parsing without timezone if RFC3339 fails, or just use as is if compatible
				// For now, logging error but attempting to use string might still fail if format is wrong
				fmt.Printf("Error parsing start_time: %v\n", err)
			}
			formattedStartTime := parsedStartTime.Format("2006-01-02 15:04:05")

			var formattedEndTime interface{}
			if s.EndTime != nil && *s.EndTime != "" {
				parsedEndTime, err := time.Parse(time.RFC3339, *s.EndTime)
				if err == nil {
					formattedEndTime = parsedEndTime.Format("2006-01-02 15:04:05")
				} else {
					formattedEndTime = *s.EndTime // Fallback
				}
			}

			dayOrder := 1
			if s.DayOrder != nil {
				dayOrder = *s.DayOrder
			}

			sortOrder := 1
			if s.SortOrder != nil {
				sortOrder = *s.SortOrder
			}

			_, err = db.Exec(`
				INSERT INTO event_schedule (uuid, event_id, title, description, start_time, end_time, day_order, sort_order, location)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, scheduleID, actualEventID, s.Title, s.Description, formattedStartTime, formattedEndTime, dayOrder, sortOrder, s.Location)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save schedule", "details": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Event schedules updated successfully",
			"count":   len(req.Schedules),
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
			AthleteID        string   `json:"athlete_id" binding:"required"`
			EventCategoryID  string   `json:"event_category_id" binding:"required"`
			PaymentAmount    float64  `json:"payment_amount"`
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
		var archerUUID string
		err = db.Get(&archerUUID, "SELECT uuid FROM archers WHERE uuid = ? OR id = ?", req.AthleteID, req.AthleteID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Archer tidak ditemukan"})
			return
		}

		err = db.Get(&exists, `
			SELECT EXISTS(SELECT 1 FROM event_participants 
			WHERE event_id = ? AND archer_id = ?)
		`, actualEventID, archerUUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check registration status", "details": err.Error()})
			return
		}

		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Pemanah sudah terdaftar di event ini"})
			return
		}

		// Insert participant
		participantUUID := uuid.New().String()
		registrationDate := time.Now()
		proofURLs := ""
		if len(req.PaymentProofURLs) > 0 {
			proofURLs = strings.Join(req.PaymentProofURLs, ",")
		}

		_, err = db.Exec(`
			INSERT INTO event_participants (
				uuid, event_id, archer_id, category_id, 
				registration_date, payment_status, payment_amount, payment_proof_urls, status
			) VALUES (?, ?, ?, ?, ?, 'menunggu_acc', ?, ?, 'Menunggu Acc')
		`, participantUUID, actualEventID, archerUUID, req.EventCategoryID, registrationDate, req.PaymentAmount, proofURLs)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register participant", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), actualEventID, "participant_registered", "event_participant", participantUUID, "Registered participant for event category: "+req.EventCategoryID, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      participantUUID,
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

// DeleteEventParticipant allows an admin to remove a participant from an event
func DeleteEventParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")

		// Resolve event slug to UUID
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Resolve participant (support UUID or Username)
		var actualParticipantID string
		err = db.Get(&actualParticipantID, `
			SELECT tp.uuid FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			WHERE tp.event_id = ? AND (tp.uuid = ? OR a.username = ?)
			LIMIT 1
		`, actualEventID, participantID, participantID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found"})
			return
		}

		// Start transaction for cleanup
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// 1. Get archer UUID from participant
		var archerUUID string
		err = tx.Get(&archerUUID, "SELECT archer_id FROM event_participants WHERE uuid = ?", actualParticipantID)
		if err == nil {
			// Delete arrow scores first
			_, _ = tx.Exec(`
				DELETE FROM qualification_arrow_scores 
				WHERE end_score_uuid IN (
					SELECT uuid FROM qualification_end_scores WHERE archer_uuid = ?
				)
			`, archerUUID)

			// Delete end scores
			_, _ = tx.Exec("DELETE FROM qualification_end_scores WHERE archer_uuid = ?", archerUUID)
		}

		// 2. Delete qualification target assignments
		_, err = tx.Exec("DELETE FROM qualification_target_assignments WHERE archer_uuid = ?", archerUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete target assignments", "details": err.Error()})
			return
		}

		// 3. Delete from event_participants
		_, err = tx.Exec("DELETE FROM event_participants WHERE uuid = ? AND event_id = ?", actualParticipantID, actualEventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete participant", "details": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit deletion"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), actualEventID, "participant_kicked", "event", actualEventID, "Kicked participant: "+actualParticipantID, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Participant removed from event successfully"})
	}
}

// UpdateEventParticipant updates an existing event participant
func UpdateEventParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")

		var req struct {
			CategoryID          *string  `json:"category_id"`
			TargetName          *string  `json:"target_name"`
			BackNumber          *string  `json:"back_number"`
			Status              *string  `json:"status"`
			PaymentStatus       *string  `json:"payment_status"`
			PaymentAmount       *float64 `json:"payment_amount"`
			PaymentProofURLs    []string `json:"payment_proof_urls"`
			AccreditationStatus *string  `json:"accreditation_status"`
			IsVerified          *bool    `json:"is_verified"`
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

		// Check if participant exists and belongs to the event (support UUID, Username, or Slugified Name)
		var pInfo struct {
			UUID     string  `db:"uuid"`
			ArcherID *string `db:"archer_id"`
		}
		err = db.Get(&pInfo, `
			SELECT tp.uuid, tp.archer_id FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			WHERE tp.event_id = ? AND (
				tp.uuid = ? OR 
				a.username = ? OR 
				a.id = ? OR
				LOWER(REPLACE(a.full_name, ' ', '-')) = LOWER(?)
			)
			LIMIT 1
		`, actualEventID, participantID, participantID, participantID, participantID)

		if err != nil {
			fmt.Printf("[DEBUG] Update lookup failed for Event: %s, ID: %s. Error: %v\n", actualEventID, participantID, err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":          "Participant not found",
				"details":        err.Error(),
				"participant_id": participantID,
				"event_id":       actualEventID,
				"hint":           "Make sure the participant exists for this event and the ID/Username is correct.",
			})
			return
		}

		actualParticipantID := pInfo.UUID
		fmt.Printf("[DEBUG] Updating participant UUID: %s for input: %s\n", actualParticipantID, participantID)

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
		if req.TargetName != nil {
			query += ", target_name = ?"
			args = append(args, *req.TargetName)
		}
		if req.BackNumber != nil {
			query += ", back_number = ?"
			args = append(args, *req.BackNumber)
		}

		// Payment status drives the participant status
		if req.PaymentStatus != nil {
			query += ", payment_status = ?"
			args = append(args, *req.PaymentStatus)

			// Auto-update status based on payment_status
			var newStatus string
			switch *req.PaymentStatus {
			case "lunas":
				newStatus = "Terdaftar"
				// Generate QR raw string when payment is lunas (paid)
				var currentQR sql.NullString
				err = db.Get(&currentQR, "SELECT qr_raw FROM event_participants WHERE uuid = ?", actualParticipantID)
				if err == nil && !currentQR.Valid {
					// Generate random QR string using uuid
					qrRaw := uuid.New().String()
					query += ", qr_raw = ?"
					args = append(args, qrRaw)
				}
			case "belum_lunas", "menunggu_acc":
				newStatus = "Menunggu Acc"
			default:
				newStatus = "Menunggu Acc"
			}
			query += ", status = ?"
			args = append(args, newStatus)
		}

		// Remove status from direct updates - it's now managed by payment_status
		// if req.Status != nil { ... } - REMOVED
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

		// Handle IsVerified
		if req.IsVerified != nil {
			if pInfo.ArcherID != nil {
				// Update existing archer verified status
				_, err = db.Exec("UPDATE archers SET is_verified = ? WHERE uuid = ?", *req.IsVerified, *pInfo.ArcherID)
				if err != nil {
					fmt.Printf("[ERROR] Failed to update archer verification: %v\n", err)
				}
			}
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
			Divisions          []string `json:"divisions" binding:"required"`
			Categories         []string `json:"categories" binding:"required"`
			EventTypeUUID      string   `json:"event_type_uuid" binding:"required"`
			GenderDivisionUUID string   `json:"gender_division_uuid"`
			MaxParticipants    int      `json:"max_participants"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Resolve event type code to enforce team size
		var eventTypeCode string
		err := db.Get(&eventTypeCode, "SELECT code FROM ref_event_types WHERE uuid = ?", req.EventTypeUUID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event type"})
			return
		}

		if eventTypeCode == "mixed_team" {
			if req.GenderDivisionUUID == "" {
				var mixedUUID string
				db.Get(&mixedUUID, "SELECT uuid FROM ref_gender_divisions WHERE code = 'mixed'")
				req.GenderDivisionUUID = mixedUUID
			}
		}

		// Check if event exists
		var eventExists bool
		err = db.Get(&eventExists, `SELECT EXISTS(SELECT 1 FROM events WHERE uuid = ?)`, eventID)
		if err != nil || !eventExists {
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
					WHERE event_id = ? AND division_uuid = ? AND category_uuid = ? AND event_type_uuid = ? AND gender_division_uuid = ?)
				`, eventID, divUUID, catUUID, req.EventTypeUUID, req.GenderDivisionUUID)

				if err != nil || catExists {
					continue
				}

				catEventID := uuid.New().String()
				_, err = db.Exec(`
					INSERT INTO event_categories (
						uuid, event_id, division_uuid, category_uuid, event_type_uuid, gender_division_uuid,
						max_participants
					) VALUES (?, ?, ?, ?, ?, ?, ?)
				`, catEventID, eventID, divUUID, catUUID, req.EventTypeUUID, req.GenderDivisionUUID, req.MaxParticipants)

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
			DivisionUUID       string `json:"division_uuid" binding:"required"`
			CategoryUUID       string `json:"category_uuid" binding:"required"`
			EventTypeUUID      string `json:"event_type_uuid" binding:"required"`
			GenderDivisionUUID string `json:"gender_division_uuid"`
			MaxParticipants    *int   `json:"max_participants"`
			Status             string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Resolve event type code to enforce team size
		var eventTypeCode string
		err := db.Get(&eventTypeCode, "SELECT code FROM ref_event_types WHERE uuid = ?", req.EventTypeUUID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid event type"})
			return
		}

		// Enforce requirements
		if eventTypeCode == "mixed_team" {
			// For mixed team, force mixed gender
			var mixedUUID string
			err = db.Get(&mixedUUID, "SELECT uuid FROM ref_gender_divisions WHERE code = 'mixed'")
			if err == nil {
				req.GenderDivisionUUID = mixedUUID
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Mixed gender division not found in system"})
				return
			}
		} else {

			// Individual or Team must have a specific gender (Men/Women)
			if req.GenderDivisionUUID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Gender division is required for this category type"})
				return
			}

			// Ensure it's not "mixed"
			var genderCode string
			db.Get(&genderCode, "SELECT code FROM ref_gender_divisions WHERE uuid = ?", req.GenderDivisionUUID)
			if genderCode == "mixed" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Only mixed teams can use the 'Mixed' gender division"})
				return
			}
		}

		// Resolve slug to UUID if needed
		var actualEventID string
		err = db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
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

		// Enforce logic if event type is being updated
		if req.EventTypeUUID != nil {
			var eventTypeCode string
			err := db.Get(&eventTypeCode, "SELECT code FROM ref_event_types WHERE uuid = ?", *req.EventTypeUUID)
			if err == nil {
				if eventTypeCode == "mixed_team" {
					if req.GenderDivisionUUID == nil || *req.GenderDivisionUUID == "" {
						var mixedUUID string
						db.Get(&mixedUUID, "SELECT uuid FROM ref_gender_divisions WHERE code = 'mixed'")
						if mixedUUID != "" {
							req.GenderDivisionUUID = &mixedUUID
						}
					}
				}
			}
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
			UUID         string  `db:"uuid" json:"id"`
			EventID      string  `db:"event_id" json:"event_id"`
			URL          string  `db:"url" json:"url"`
			Caption      *string `db:"caption" json:"caption"`
			AltText      *string `db:"alt_text" json:"alt_text"`
			DisplayOrder int     `db:"display_order" json:"display_order"`
			IsPrimary    bool    `db:"is_primary" json:"is_primary"`
			CreatedAt    string  `db:"created_at" json:"created_at"`
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
			       COUNT(tm.uuid) as member_count,
				   GROUP_CONCAT(COALESCE(a.full_name, 'Unknown') ORDER BY tm.member_order SEPARATOR ', ') as member_names,
				   GROUP_CONCAT(COALESCE(tm.total_score, 0) ORDER BY tm.member_order SEPARATOR ', ') as member_scores
			FROM teams t
			LEFT JOIN team_members tm ON t.uuid = tm.team_id
			LEFT JOIN event_participants ep ON tm.participant_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE t.event_id = ?
		`
		args := []interface{}{eventUUID}

		if categoryID != "" {
			query += " AND t.category_id = ?"
			args = append(args, categoryID)
		}

		query += " GROUP BY t.uuid, t.team_name, t.country_code, t.country_name, t.status, t.total_score, t.total_x_count, t.created_at ORDER BY t.total_score DESC, t.total_x_count DESC"

		type Team struct {
			ID           string  `db:"uuid" json:"id"`
			TeamName     string  `db:"team_name" json:"team_name"`
			CountryCode  *string `db:"country_code" json:"country_code"`
			CountryName  *string `db:"country_name" json:"country_name"`
			Status       string  `db:"status" json:"status"`
			TotalScore   *int    `db:"total_score" json:"total_score"`
			TotalXCount  *int    `db:"total_x_count" json:"total_x_count"`
			MemberCount  int     `db:"member_count" json:"member_count"`
			MemberNames  *string `db:"member_names" json:"member_names"`
			MemberScores *string `db:"member_scores" json:"member_scores"`
			CreatedAt    string  `db:"created_at" json:"created_at"`
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

// GetMyEvents returns events managed by the authenticated user
func GetMyEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		status := c.Query("status")
		search := c.Query("search")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		// Base query: get events where organizer_id is the current user
		whereClause := "WHERE t.organizer_id = ?"
		args := []interface{}{userID}

		if status != "" {
			whereClause += ` AND t.status = ?`
			args = append(args, status)
		}

		if search != "" {
			whereClause += ` AND (t.name LIKE ? OR t.code LIKE ? OR t.location LIKE ?)`
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		// Get total count
		var total int
		err := db.Get(&total, `SELECT COUNT(*) FROM events t `+whereClause, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count events", "details": err.Error()})
			return
		}

		query := `
			SELECT 
				t.*,
				u.full_name as organizer_name,
				u.email as organizer_email,
				u.slug as organizer_slug,
				u.avatar_url as organizer_avatar_url,
				COUNT(DISTINCT tp.uuid) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.uuid = tp.event_id
			LEFT JOIN event_categories te ON t.uuid = te.event_id
			` + whereClause + `
			GROUP BY t.uuid
			ORDER BY t.created_at DESC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)

		var events []models.EventWithDetails
		err = db.Select(&events, query, args...)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"events": []interface{}{},
				"total":  0,
			})
			return
		}

		// Mask URLs
		for i := range events {
			if events[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*events[i].BannerURL)
				events[i].BannerURL = &masked
			}
			if events[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*events[i].LogoURL)
				events[i].LogoURL = &masked
			}
			if events[i].TechnicalGuidebookURL != nil {
				masked := utils.MaskMediaURL(*events[i].TechnicalGuidebookURL)
				events[i].TechnicalGuidebookURL = &masked
			}
			if events[i].OrganizerAvatarURL != nil {
				masked := utils.MaskMediaURL(*events[i].OrganizerAvatarURL)
				events[i].OrganizerAvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
	}
}

// ReregisterParticipant handles QR code scanning for participant re-registration
func ReregisterParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			QRRaw string `json:"qr_raw" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "QR code is required"})
			return
		}

		// Find participant by qr_raw
		type ParticipantInfo struct {
			UUID         string  `db:"uuid"`
			FullName     string  `db:"full_name"`
			Email        string  `db:"email"`
			ClubName     *string `db:"club_name"`
			DivisionName string  `db:"division_name"`
			CategoryName string  `db:"category_name"`
			EventName    string  `db:"event_name"`
			Status       string  `db:"status"`
		}

		var participant ParticipantInfo
		err := db.Get(&participant, `
			SELECT 
				ep.uuid,
				a.full_name,
				a.email,
				c.name as club_name,
				COALESCE(d.name, '') as division_name,
				COALESCE(ag.name, '') as category_name,
				e.name as event_name,
				COALESCE(ep.status, 'Menunggu Acc') as status
			FROM event_participants ep
			INNER JOIN archers a ON ep.archer_id = a.uuid
			INNER JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN clubs c ON a.club_id = c.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types d ON ec.division_uuid = d.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			WHERE ep.qr_raw = ?
			LIMIT 1
		`, req.QRRaw)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Peserta tidak ditemukan. QR Code tidak valid."})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error", "details": err.Error()})
			return
		}

		// Check if participant is registered (status = "Terdaftar")
		if participant.Status != "Terdaftar" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Peserta belum disetujui. Status: " + participant.Status,
			})
			return
		}

		// Update last_reregistration_at
		_, err = db.Exec(`
			UPDATE event_participants 
			SET last_reregistration_at = NOW()
			WHERE uuid = ?
		`, participant.UUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update registration", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Registrasi ulang berhasil",
			"participant": gin.H{
				"uuid":          participant.UUID,
				"full_name":     participant.FullName,
				"email":         participant.Email,
				"club_name":     participant.ClubName,
				"division_name": participant.DivisionName,
				"category_name": participant.CategoryName,
				"event_name":    participant.EventName,
			},
		})
	}
}
