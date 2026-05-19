package handler

import (
	"Archeris-api/models"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Archeris-api/utils"

	"encoding/csv"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetEvents returns a list of events
func GetEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		search := c.Query("search")
		limit, offset, page := utils.GetPaginationParams(c)
		organizerID := c.Query("organizer_id")

		// Check if user is archer to filter events and include participant status
		userID, userExists := c.Get("user_id")
		userRole, roleExists := c.Get("role")

		whereClause := "WHERE 1=1"
		args := []interface{}{}

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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung jumlah event", "details": err.Error()})
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
				u.phone as organizer_phone,
				COUNT(DISTINCT tp2.archer_id) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count,
				tp.payment_status,
				tp.uuid as participant_uuid
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url, whatsapp_no as phone FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, NULL as email, slug, logo_url as avatar_url, phone FROM clubs
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
				u.phone as organizer_phone,
				COUNT(DISTINCT tp.archer_id) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url, whatsapp_no as phone FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, NULL as email, slug, logo_url as avatar_url, phone FROM clubs
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event", "details": err.Error()})
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

		meta := utils.CalculatePagination(total, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{
			"data":   events,
			"events": events,
			"total":  total,
			"meta":   meta,
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
				u.phone as organizer_phone,
				COALESCE(participant_stats.participant_count, 0) as participant_count,
				COALESCE(category_stats.event_count, 0) as event_count,
				COALESCE(target_stats.target_count, 0) as target_count,
				COALESCE(active_target_stats.active_target_count, 0) as active_target_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, avatar_url, slug, whatsapp_no as phone FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, NULL as email, logo_url as avatar_url, slug, phone FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN (
				SELECT event_id, COUNT(DISTINCT archer_id) as participant_count
				FROM event_participants
				GROUP BY event_id
			) participant_stats ON t.uuid = participant_stats.event_id
			LEFT JOIN (
				SELECT event_id, COUNT(DISTINCT uuid) as event_count
				FROM event_categories
				GROUP BY event_id
			) category_stats ON t.uuid = category_stats.event_id
			LEFT JOIN (
				SELECT event_uuid, COUNT(*) as target_count
				FROM event_targets
				GROUP BY event_uuid
			) target_stats ON t.uuid = target_stats.event_uuid
			LEFT JOIN (
				SELECT event_id, COUNT(DISTINCT target_uuid) as active_target_count
				FROM (
					SELECT qs.event_uuid as event_id, qta.target_uuid
					FROM qualification_target_assignments qta
					JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
					UNION ALL
					SELECT eb.event_uuid as event_id, em.target_uuid
					FROM elimination_matches em
					JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
					WHERE em.target_uuid IS NOT NULL
				) combined
				GROUP BY event_id
			) active_target_stats ON t.uuid = active_target_stats.event_id
			WHERE t.uuid = ? OR t.slug = ?
			LIMIT 1
		`

		var Event models.EventWithDetails
		err := db.Get(&Event, query, id, id)
		if err != nil {
			// Log the error for debugging
			fmt.Printf("[GetEventByID] Error fetching event with id/slug '%s': %v\n", id, err)
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan", "id": id})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event", "details": err.Error()})
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
				c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
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

		utils.PopulateEventDetailExtras(db, &Event)

		c.JSON(http.StatusOK, Event)
	}
}

// CreateEvent creates a new Event
func CreateEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid", "details": err.Error()})
			return
		}

		// Get user ID from context (set by auth middleware)
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
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

		// Generate slug from user input (if provided) or fallback to name.
		// Keep it clean and readable without random suffix by default.
		baseSlugSource := req.Slug
		if strings.TrimSpace(baseSlugSource) == "" {
			baseSlugSource = req.Name
		}
		baseSlug := strings.ToLower(strings.TrimSpace(baseSlugSource))
		baseSlug = strings.ReplaceAll(baseSlug, " ", "-")
		var cleanSlug strings.Builder
		for _, r := range baseSlug {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
				cleanSlug.WriteRune(r)
			}
		}
		finalSlug := strings.Trim(cleanSlug.String(), "-")
		finalSlug = strings.Join(strings.FieldsFunc(finalSlug, func(r rune) bool { return r == '-' }), "-")
		if finalSlug == "" {
			finalSlug = "event"
		}

		// Ensure uniqueness with deterministic numeric suffix, not random text.
		originalSlug := finalSlug
		suffix := 2
		var err error
		for {
			var exists int
			err = db.Get(&exists, `SELECT COUNT(1) FROM events WHERE slug = ?`, finalSlug)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi keunikan slug", "details": err.Error()})
				return
			}
			if exists == 0 {
				break
			}
			finalSlug = fmt.Sprintf("%s-%d", originalSlug, suffix)
			suffix++
		}

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
		} else if status == "published" {
			status = "active"
		}

		// Use location_type if provided, otherwise fallback to type for backward compatibility
		locationType := req.LocationType
		if locationType == nil && req.Type != nil {
			locationType = req.Type
		}

		_, err = db.Exec(query,
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat event", "details": err.Error()})
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
			"message": "Event berhasil dibuat",
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid", "details": err.Error()})
			return
		}

		// Resolve slug to UUID if needed
		var actualID string
		err := db.Get(&actualID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data event", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), id, "Event_updated", "Event", id, "Updated Event", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Data event berhasil diperbarui"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		result, err := db.Exec("DELETE FROM events WHERE uuid = ?", actualID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data event", "details": err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "Event_deleted", "Event", id, "Deleted Event", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Data event berhasil dihapus"})
	}
}

// These functions are now in division_category.go to avoid duplication

// GetEventEvents returns events for a specific event
func GetEventEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		limit, offset, page := utils.GetPaginationParams(c)
		bowTypeFilter := c.Query("bow_type")
		eventTypeFilter := c.Query("event_type")

		// First, resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `
			SELECT uuid FROM events WHERE uuid = ? OR slug = ?
		`, eventID, eventID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type EventEvent struct {
			ID                 string  `db:"id" json:"id"`
			EventID            string  `db:"event_id" json:"event_id"`
			DivisionName       string  `db:"division_name" json:"division_name"`
			DivisionID         string  `db:"division_id" json:"division_id"`
			CategoryName       string  `db:"category_name" json:"category_name"`
			CategoryNameCustom *string `db:"category_name_custom" json:"category_name_custom"`
			CategoryID         string  `db:"category_id" json:"category_id"`
			EventTypeName      string  `db:"event_type_name" json:"event_type_name"`
			EventTypeID        string  `db:"event_type_id" json:"event_type_id"`
			GenderDivisionName string  `db:"gender_division_name" json:"gender_division_name"`
			GenderDivisionID   string  `db:"gender_division_id" json:"gender_division_id"`
			MaxParticipants    *int    `db:"max_participants" json:"max_participants"`
			TeamSize           int     `db:"team_size" json:"team_size"`
			ParticipantCount   int     `db:"participant_count" json:"participant_count"`
			TeamCount          int     `db:"team_count" json:"team_count"`
			Status             string  `db:"status" json:"status"`
			CreatedAt          string  `db:"created_at" json:"created_at"`
		}

		whereClause := "WHERE te.event_id = ?"
		args := []interface{}{actualEventID}

		if bowTypeFilter != "" && bowTypeFilter != "all" {
			whereClause += " AND d.code = ?"
			args = append(args, bowTypeFilter)
		}

		if eventTypeFilter != "" && eventTypeFilter != "all" {
			whereClause += " AND te.event_type_uuid = ?"
			args = append(args, eventTypeFilter)
		}

		// Get total count
		var total int
		err = db.Get(&total, `
			SELECT COUNT(*) 
			FROM event_categories te
			JOIN ref_bow_types d ON te.division_uuid = d.uuid
			`+whereClause, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung kategori event", "details": err.Error()})
			return
		}

		var events []EventEvent
		query := `
			SELECT 
				te.uuid as id, te.event_id, 
				te.max_participants, te.status, te.created_at, te.category_name_custom,
				CASE 
					WHEN et.code = 'mixed_team' THEN 2 
					WHEN et.code = 'team' THEN 3 
					ELSE 1 
				END as team_size,
				d.name as division_name, d.uuid as division_id,
				COALESCE(te.category_name_custom, c.name) as category_name, c.uuid as category_id,
				COALESCE(et.name, '') as event_type_name, COALESCE(te.event_type_uuid, '') as event_type_id,
				COALESCE(gd.name, '') as gender_division_name, COALESCE(te.gender_division_uuid, '') as gender_division_id,
				COALESCE(p.p_count, 0) as participant_count,
				COALESCE(t.t_count, 0) as team_count
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
			LEFT JOIN (
				SELECT event_id as category_id, COUNT(*) as t_count 
				FROM teams 
				GROUP BY event_id
			) t ON te.uuid = t.category_id
			` + whereClause + `
			ORDER BY d.name, c.name, et.name, gd.name
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)
		err = db.Select(&events, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil kategori event", "details": err.Error()})
			return
		}

		meta := utils.CalculatePagination(total, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{
			"data":   events,
			"events": events,
			"total":  total,
			"meta":   meta,
		})
	}
}

// GetEventParticipants returns participants for a specific event with pagination
func GetEventParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		limit, offset, page := utils.GetPaginationParams(c)
		categoryFilter := c.Query("category")
		categoryIDFilter := c.Query("category_id")
		categoryIDsFilter := c.Query("category_ids")
		searchQuery := c.Query("search")
		groupBy := c.Query("group_by")

		// Resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Collect requested category IDs from category_id(s) and category_ids (CSV or repeated query params).
		rawCategoryIDs := []string{}
		for _, q := range c.QueryArray("category_id") {
			for _, part := range strings.Split(q, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					rawCategoryIDs = append(rawCategoryIDs, part)
				}
			}
		}
		if len(rawCategoryIDs) == 0 && categoryIDFilter != "" {
			for _, part := range strings.Split(categoryIDFilter, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					rawCategoryIDs = append(rawCategoryIDs, part)
				}
			}
		}
		for _, q := range c.QueryArray("category_ids") {
			for _, part := range strings.Split(q, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					rawCategoryIDs = append(rawCategoryIDs, part)
				}
			}
		}
		if categoryIDsFilter != "" {
			for _, part := range strings.Split(categoryIDsFilter, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					rawCategoryIDs = append(rawCategoryIDs, part)
				}
			}
		}

		resolveParticipantCategoryIDs := func(input []string) []string {
			resolved := []string{}
			seen := map[string]bool{}
			for _, categoryID := range input {
				categoryID = strings.TrimSpace(categoryID)
				if categoryID == "" || seen[categoryID] {
					continue
				}

				var indivIDs []string
				_ = db.Select(&indivIDs, `
					SELECT i.uuid
					FROM event_categories t
					JOIN ref_event_types tt ON t.event_type_uuid = tt.uuid
					JOIN event_categories i ON i.event_id = t.event_id
						AND i.division_uuid = t.division_uuid
						AND i.category_uuid = t.category_uuid
					JOIN ref_event_types it ON i.event_type_uuid = it.uuid
					WHERE t.uuid = ?
					  AND tt.code IN ('team', 'mixed_team')
					  AND it.code = 'individual'
				`, categoryID)

				if len(indivIDs) > 0 {
					for _, indivID := range indivIDs {
						if indivID == "" || seen[indivID] {
							continue
						}
						resolved = append(resolved, indivID)
						seen[indivID] = true
					}
					continue
				}

				resolved = append(resolved, categoryID)
				seen[categoryID] = true
			}
			return resolved
		}

		resolvedCategoryIDs := resolveParticipantCategoryIDs(rawCategoryIDs)

		if groupBy == "archer" {
			// Grouped by archer logic
			whereClause := "WHERE tp.event_id = ?"
			args := []interface{}{actualEventID}

			if len(resolvedCategoryIDs) == 1 {
				whereClause += " AND tp.category_id = ?"
				args = append(args, resolvedCategoryIDs[0])
			} else if len(resolvedCategoryIDs) > 1 {
				placeholders := strings.Repeat(",?", len(resolvedCategoryIDs))[1:]
				whereClause += " AND tp.category_id IN (" + placeholders + ")"
				for _, id := range resolvedCategoryIDs {
					args = append(args, id)
				}
			}

			if searchQuery != "" {
				searchTerm := "%" + searchQuery + "%"
				whereClause += " AND (a.full_name LIKE ? OR a.email LIKE ? OR cl.name LIKE ?)"
				args = append(args, searchTerm, searchTerm, searchTerm)
			}

			if paymentStatus := c.Query("payment_status"); paymentStatus != "" && paymentStatus != "Semua" {
				whereClause += " AND tp.payment_status = ?"
				args = append(args, paymentStatus)
			}

			// Count unique archers
			var total int
			countQuery := "SELECT COUNT(DISTINCT archer_id) FROM event_participants tp LEFT JOIN archers a ON tp.archer_id = a.uuid LEFT JOIN clubs cl ON a.club_id = cl.uuid " + whereClause
			err = db.Get(&total, countQuery, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung jumlah atlet", "details": err.Error()})
				return
			}

			type GroupedParticipant struct {
				ArcherID      string        `db:"archer_id" json:"archer_id"`
				AthleteCode   string        `db:"athlete_code" json:"athlete_code"`
				FullName      string        `db:"full_name" json:"full_name"`
				Email         string        `db:"email" json:"email"`
				AvatarURL     *string       `db:"avatar_url" json:"avatar_url"`
				ClubName      *string       `db:"club_name" json:"club_name"`
				City          *string       `db:"city" json:"city"`
				PaymentStatus string        `db:"payment_status" json:"payment_status"`
				Categories    string        `db:"categories" json:"-"`
				CategoryList  []interface{} `json:"categories"`
			}

			var participants []GroupedParticipant
			query := `
				SELECT 
					a.uuid as archer_id,
					a.id as athlete_code,
					a.full_name,
					COALESCE(a.email, '') as email,
					a.avatar_url,
					COALESCE(cl.name, '') as club_name,
					a.city as city,
					MAX(tp.payment_status) as payment_status,
					JSON_ARRAYAGG(JSON_OBJECT(
						'participant_id', tp.uuid,
						'category_id', tp.category_id,
						'division_name', COALESCE(d.name, ''),
						'category_name', COALESCE(te.category_name_custom, c.name, ''),
						'event_type_name', COALESCE(et.name, ''),
						'gender_division_name', COALESCE(gd.name, ''),
						'payment_status', COALESCE(tp.payment_status, 'pending'),
						'registration_source', COALESCE(tp.registration_source, 'self_register'),
						'qr_raw', tp.qr_raw,
						'registration_date', tp.registration_date,
						'last_reregistration_at', tp.last_reregistration_at
					)) as categories
				FROM event_participants tp
				JOIN archers a ON tp.archer_id = a.uuid
				LEFT JOIN clubs cl ON a.club_id = cl.uuid
				LEFT JOIN event_categories te ON tp.category_id = te.uuid
				LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
				LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
				LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
				LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
				` + whereClause + `
				GROUP BY a.uuid, cl.name
				ORDER BY a.full_name ASC
				LIMIT ? OFFSET ?
			`
			fetchArgs := append(args, limit, offset)
			err = db.Select(&participants, query, fetchArgs...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data peserta berkelompok", "details": err.Error()})
				return
			}

			// Post-process categories and mask URLs
			for i := range participants {
				if participants[i].Categories != "" {
					var cats []interface{}
					if err := json.Unmarshal([]byte(participants[i].Categories), &cats); err == nil {
						participants[i].CategoryList = cats
					}
				}
				if participants[i].AvatarURL != nil {
					masked := utils.MaskMediaURL(*participants[i].AvatarURL)
					participants[i].AvatarURL = &masked
				}
			}

			// Get paid/pending counters aligned with active category filter.
			statusWhere := "WHERE tp.event_id = ?"
			statusArgs := []interface{}{actualEventID}
			if len(resolvedCategoryIDs) == 1 {
				statusWhere += " AND tp.category_id = ?"
				statusArgs = append(statusArgs, resolvedCategoryIDs[0])
			} else if len(resolvedCategoryIDs) > 1 {
				placeholders := strings.Repeat(",?", len(resolvedCategoryIDs))[1:]
				statusWhere += " AND tp.category_id IN (" + placeholders + ")"
				for _, id := range resolvedCategoryIDs {
					statusArgs = append(statusArgs, id)
				}
			}

			var verifiedCount, pendingCount int
			verifiedQuery := "SELECT COUNT(DISTINCT tp.archer_id) FROM event_participants tp " + statusWhere + " AND tp.payment_status IN ('paid', 'lunas')"
			pendingQuery := "SELECT COUNT(DISTINCT tp.archer_id) FROM event_participants tp " + statusWhere + " AND tp.payment_status IN ('pending', 'menunggu_acc', 'menunggu acc')"
			_ = db.Get(&verifiedCount, verifiedQuery, statusArgs...)
			_ = db.Get(&pendingCount, pendingQuery, statusArgs...)

			c.JSON(http.StatusOK, gin.H{
				"participants":   participants,
				"total":          total,
				"verified_count": verifiedCount,
				"pending_count":  pendingCount,
				"limit":          limit,
				"offset":         offset,
			})
			return
		}

		// Standard logic (existing)
		whereClause := "WHERE tp.event_id = ?"
		args := []interface{}{actualEventID}
		countArgs := []interface{}{actualEventID}

		// Filter by one or many category IDs.
		if len(resolvedCategoryIDs) == 1 {
			whereClause += " AND tp.category_id = ?"
			args = append(args, resolvedCategoryIDs[0])
			countArgs = append(countArgs, resolvedCategoryIDs[0])
		} else if len(resolvedCategoryIDs) > 1 {
			placeholders := strings.Repeat(",?", len(resolvedCategoryIDs))[1:]
			whereClause += " AND tp.category_id IN (" + placeholders + ")"
			for _, id := range resolvedCategoryIDs {
				args = append(args, id)
				countArgs = append(countArgs, id)
			}
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

		if paymentStatus := c.Query("payment_status"); paymentStatus != "" && paymentStatus != "Semua" {
			whereClause += " AND tp.payment_status = ?"
			args = append(args, paymentStatus)
			countArgs = append(countArgs, paymentStatus)
		}

		// Get total count with filters
		countQuery := "SELECT COUNT(*) FROM event_participants tp LEFT JOIN archers a ON tp.archer_id = a.uuid LEFT JOIN clubs cl ON a.club_id = cl.uuid LEFT JOIN event_categories te ON tp.category_id = te.uuid LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid " + whereClause
		var total int
		err = db.Get(&total, countQuery, countArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung jumlah peserta", "details": err.Error()})
			return
		}

		type Participant struct {
			ID                   string  `db:"id" json:"id"`
			ArcherID             *string `db:"archer_id" json:"archer_id"`
			AthleteCode          *string `db:"athlete_code" json:"athlete_code"`
			Username             *string `db:"username" json:"username"`
			FullName             string  `db:"full_name" json:"full_name"`
			Email                string  `db:"email" json:"email"`
			City                 *string `db:"city" json:"city"`
			ClubID               *string `db:"club_id" json:"club_id"`
			ClubName             *string `db:"club_name" json:"club_name"`
			EventID              string  `db:"event_id" json:"event_id"`
			CategoryID           string  `db:"category_id" json:"category_id"`
			DivisionName         string  `db:"division_name" json:"division_name"`
			CategoryName         string  `db:"category_name" json:"category_name"`
			EventTypeName        *string `db:"event_type_name" json:"event_type_name"`
			GenderDivisionName   *string `db:"gender_division_name" json:"gender_division_name"`
			TargetName           *string `db:"target_name" json:"target_name"`
			QRRaw                *string `db:"qr_raw" json:"qr_raw"`
			AvatarURL            *string `db:"avatar_url" json:"avatar_url"`
			RegistrationDate     string  `db:"registration_date" json:"registration_date"`
			LastReregistrationAt *string `db:"last_reregistration_at" json:"last_reregistration_at"`
			TotalScore           int     `db:"total_score" json:"total_score"`
			TotalX               int     `db:"total_x" json:"total_x"`
			RegistrationSource   string  `db:"registration_source" json:"registration_source"`
			PaymentStatus        string  `db:"payment_status" json:"payment_status"`
		}

		var participants []Participant
		query := `
			SELECT 
				tp.uuid as id, tp.archer_id, tp.event_id, tp.category_id, tp.target_name, tp.qr_raw,
				tp.payment_status, tp.registration_date, tp.last_reregistration_at,
				COALESCE(tp.registration_source, 'self_register') as registration_source,
				a.id as athlete_code,
				a.username as username,
				a.full_name as full_name,
				COALESCE(a.email, '') as email,
				a.city as city,
				a.club_id as club_id,
				a.avatar_url as avatar_url,
				COALESCE(cl.name, '') as club_name,
				COALESCE(d.name, '') as division_name, COALESCE(te.category_name_custom, c.name, '') as category_name,
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
				SELECT participant_uuid, SUM(total_score_end) as total_score, SUM(x_count_end) as total_x
				FROM qualification_end_scores
				GROUP BY participant_uuid
			) scores ON tp.uuid = scores.participant_uuid
			` + whereClause + `
			GROUP BY tp.uuid, a.uuid, cl.uuid, te.uuid, d.uuid, c.uuid, et.uuid, gd.uuid, a.id, a.username, a.full_name, a.email, a.city, a.club_id, a.avatar_url, cl.name, d.name, c.name, te.category_name_custom, et.name, gd.name, scores.total_score, scores.total_x, tp.payment_status
			ORDER BY total_score DESC, total_x DESC, a.full_name ASC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)
		err = db.Select(&participants, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants", "details": err.Error()})
			return
		}

		// Get verified (paid) and pending counts
		var verifiedCount, pendingCount int
		db.Get(&verifiedCount, "SELECT COUNT(*) FROM event_participants WHERE event_id = ? AND payment_status IN ('paid', 'lunas')", actualEventID)
		db.Get(&pendingCount, "SELECT COUNT(*) FROM event_participants WHERE event_id = ? AND payment_status IN ('pending', 'menunggu_acc', 'menunggu acc')", actualEventID)

		// Mask avatar URLs
		for i := range participants {
			if participants[i].AvatarURL != nil {
				masked := utils.MaskMediaURL(*participants[i].AvatarURL)
				participants[i].AvatarURL = &masked
			}
		}

		meta := utils.CalculatePagination(total, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{
			"data":           participants,
			"participants":   participants,
			"total":          total,
			"verified_count": verifiedCount,
			"pending_count":  pendingCount,
			"meta":           meta,
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
			ID                          string        `db:"id" json:"id"`
			AthleteCode                 *string       `db:"athlete_code" json:"athlete_code"`
			ArcherID                    *string       `db:"archer_id" json:"archer_id"`
			FullName                    string        `db:"full_name" json:"full_name"`
			Username                    *string       `db:"username" json:"username"`
			Email                       string        `db:"email" json:"email"`
			City                        *string       `db:"city" json:"city"`
			ClubID                      *string       `db:"club_id" json:"club_id"`
			ClubName                    *string       `db:"club_name" json:"club_name"`
			EventID                     string        `db:"event_id" json:"event_id"`
			CategoryID                  string        `db:"category_id" json:"category_id"`
			CategoriesRaw               *string       `db:"categories" json:"-"`
			Categories                  []interface{} `json:"categories"`
			DivisionName                string        `db:"division_name" json:"division_name"`
			CategoryName                string        `db:"category_name" json:"category_name"`
			EventTypeName               *string       `db:"event_type_name" json:"event_type_name"`
			GenderDivisionName          *string       `db:"gender_division_name" json:"gender_division_name"`
			TargetName                  *string       `db:"target_name" json:"target_name"`
			QRRaw                       *string       `db:"qr_raw" json:"qr_raw"`
			PaymentStatus               string        `db:"payment_status" json:"payment_status"`
			AvatarURL                   *string       `db:"avatar_url" json:"avatar_url"`
			PaymentAmount               float64       `db:"payment_amount" json:"payment_amount"`
			PaymentProofURLs            []string      `json:"payment_proof_urls"`
			RegistrationDate            string        `db:"registration_date" json:"registration_date"`
			IsVerified                  bool          `db:"is_verified" json:"is_verified"`
			RegistrationSource          string        `db:"registration_source" json:"registration_source"`
			QualificationAssignmentUUID *string       `db:"qualification_assignment_uuid" json:"qualification_assignment_uuid"`
			InElimination               bool          `db:"in_elimination" json:"in_elimination"`
		}

		var participant Participant
		err = db.Get(&participant, `
			SELECT 
				tp.uuid as id, tp.archer_id, tp.event_id, tp.category_id, tp.target_name, tp.qr_raw,
				tp.payment_amount, tp.payment_status,
				COALESCE(tp.registration_source, 'self_register') as registration_source,
				tp.registration_date,
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
				COALESCE(a.is_verified, 0) as is_verified,
				(SELECT uuid FROM qualification_target_assignments WHERE participant_uuid = tp.uuid LIMIT 1) as qualification_assignment_uuid,
				EXISTS(
					SELECT 1 FROM elimination_matches 
					WHERE entry_a_uuid IN (SELECT uuid FROM elimination_entries WHERE participant_uuid = tp.uuid)
					   OR entry_b_uuid IN (SELECT uuid FROM elimination_entries WHERE participant_uuid = tp.uuid)
				) as in_elimination,
				(
					SELECT JSON_ARRAYAGG(JSON_OBJECT(
						'participant_id', tp2.uuid,
						'category_id', tp2.category_id,
						'division_name', COALESCE(d2.name, ''),
						'category_name', COALESCE(te2.category_name_custom, c2.name, ''),
						'event_type_name', COALESCE(et2.name, ''),
						'gender_division_name', COALESCE(gd2.name, ''),
						'payment_status', COALESCE(tp2.payment_status, 'pending'),
						'registration_date', tp2.registration_date
					))
					FROM event_participants tp2
					LEFT JOIN event_categories te2 ON tp2.category_id = te2.uuid
					LEFT JOIN ref_bow_types d2 ON te2.division_uuid = d2.uuid
					LEFT JOIN ref_age_groups c2 ON te2.category_uuid = c2.uuid
					LEFT JOIN ref_event_types et2 ON te2.event_type_uuid = et2.uuid
					LEFT JOIN ref_gender_divisions gd2 ON te2.gender_division_uuid = gd2.uuid
					WHERE tp2.event_id = tp.event_id AND tp2.archer_id = tp.archer_id
				) as categories
			FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			WHERE tp.event_id = ? AND a.uuid = (
				SELECT archer_id FROM event_participants tp_sub
				LEFT JOIN archers a_sub ON tp_sub.archer_id = a_sub.uuid
				WHERE tp_sub.event_id = ? AND (
					tp_sub.uuid = ? OR 
					tp_sub.archer_id = ? OR
					a_sub.username = ? OR 
					a_sub.id = ? OR
					LOWER(REPLACE(a_sub.full_name, ' ', '-')) = LOWER(?)
				)
				LIMIT 1
			)
			GROUP BY a.uuid
			ORDER BY tp.qr_raw IS NULL ASC, tp.created_at ASC
		LIMIT 1
		`, actualEventID, actualEventID, participantID, participantID, participantID, participantID, participantID)

		if err != nil {
			fmt.Printf("[DEBUG] Participant not found in DB for Event: %s, ID: %s. Error: %v\n", actualEventID, participantID, err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":          "Peserta tidak ditemukan",
				"details":        err.Error(),
				"participant_id": participantID,
				"event_id":       actualEventID,
				"hint":           "Make sure the participant exists for this event and the ID/Username is correct.",
			})
			return
		}

		fmt.Printf("[DEBUG] Found participant: %s (UUID: %s)\n", participant.FullName, participant.ID)

		// Parse payment proof URLs
		participant.PaymentProofURLs = []string{}

		// Parse categories if raw exists
		if participant.CategoriesRaw != nil && *participant.CategoriesRaw != "" {
			var cats []interface{}
			if err := json.Unmarshal([]byte(*participant.CategoriesRaw), &cats); err == nil {
				participant.Categories = cats
			}
		}

		// Mask avatar URL
		if participant.AvatarURL != nil {
			masked := utils.MaskMediaURL(*participant.AvatarURL)
			participant.AvatarURL = &masked
		}

		c.JSON(http.StatusOK, participant)
	}
}

// GetMyEventRegistration returns the current logged-in archer's registration for a specific event
func GetMyEventRegistration(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Resolve event slug to UUID
		var actualEventID string
		if err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Get archer UUID for this user (if any)
		var archerID string
		if err := db.Get(&archerID, `SELECT uuid FROM archers WHERE uuid = ? LIMIT 1`, userID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profil pemanah tidak ditemukan untuk pengguna ini"})
			return
		}

		type MyRegistrationCategory struct {
			ID                 string  `json:"id"`
			CategoryUUID       string  `json:"category_id"`
			DivisionName       string  `json:"division_name"`
			CategoryName       string  `json:"category_name"`
			EventTypeName      *string `json:"event_type_name"`
			GenderDivisionName *string `json:"gender_division_name"`
			TargetName         *string `json:"target_name"`
			PaymentStatus      string  `json:"payment_status"`
			PaymentAmount      float64 `json:"payment_amount"`
			RegistrationDate   string  `json:"registration_date"`
		}

		type MyRegistrationResponse struct {
			ArcherID            string                     `json:"archer_id"`
			AthleteCode         *string                    `json:"athlete_code"`
			FullName            string                     `json:"full_name"`
			Email               string                     `json:"email"`
			ClubName            *string                    `json:"club_name"`
			City                *string                    `json:"city"`
			AvatarURL           *string                    `json:"avatar_url"`
			PaymentStatus       string                     `json:"payment_status"` // Combined or latest
			PaymentAmount       float64                    `json:"payment_amount"` // Total
			PaymentProofURLs    []string                   `json:"payment_proof_urls"`
			Categories          []MyRegistrationCategory   `json:"categories"`
			Transaction         *models.PaymentTransaction `json:"transaction"`
			PaymentMethodManual *string                    `json:"payment_method_manual"`
		}

		type Row struct {
			ID                  string  `db:"id"`
			TargetName          *string `db:"target_name"`
			PaymentStatus       string  `db:"payment_status"`
			PaymentAmount       float64 `db:"payment_amount"`
			RegistrationDate    string  `db:"registration_date"`
			DivisionName        string  `db:"division_name"`
			CategoryUUID        string  `db:"category_id"`
			CategoryName        string  `db:"category_name"`
			EventTypeName       *string `db:"event_type_name"`
			GenderDivisionName  *string `db:"gender_division_name"`
			AthleteCode         *string `db:"athlete_code"`
			FullName            string  `db:"full_name"`
			Email               string  `db:"email"`
			City                *string `db:"city"`
			AvatarURL           *string `db:"avatar_url"`
			ClubName            *string `db:"club_name"`
		}

		var rows []Row
		err := db.Select(&rows, `
			SELECT 
				tp.uuid as id, tp.target_name, tp.category_id,
				tp.payment_status, tp.payment_amount, tp.registration_date,
				a.id as athlete_code, a.full_name, COALESCE(a.email, '') as email,
				a.city as city, a.avatar_url, COALESCE(cl.name, '') as club_name,
				COALESCE(d.name, '') as division_name, COALESCE(c.name, '') as category_name,
				COALESCE(et.name, '') as event_type_name, COALESCE(gd.name, '') as gender_division_name
			FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			WHERE tp.event_id = ? AND tp.archer_id = ?
			ORDER BY tp.created_at ASC
		`, actualEventID, archerID)

		if err != nil || len(rows) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registrasi tidak ditemukan untuk event ini"})
			return
		}

		firstRow := rows[0]
		resp := MyRegistrationResponse{
			ArcherID:    archerID,
			AthleteCode: firstRow.AthleteCode,
			FullName:    firstRow.FullName,
			Email:       firstRow.Email,
			ClubName:    firstRow.ClubName,
			City:        firstRow.City,
			AvatarURL:   firstRow.AvatarURL,
			Categories:  []MyRegistrationCategory{},
		}

		// Combined status logic: if any is "lunas", show lunas?
		// Or if any is "menunggu acc", show that?
		// Let's just use the first one for the global chip if we have to,
		// but we'll show per category anyway.
		resp.PaymentStatus = firstRow.PaymentStatus

		for _, row := range rows {
			resp.Categories = append(resp.Categories, MyRegistrationCategory{
				ID:                 row.ID,
				CategoryUUID:       row.CategoryUUID,
				DivisionName:       row.DivisionName,
				CategoryName:       row.CategoryName,
				EventTypeName:      row.EventTypeName,
				GenderDivisionName: row.GenderDivisionName,
				TargetName:         row.TargetName,
				PaymentStatus:      row.PaymentStatus,
				PaymentAmount:      row.PaymentAmount,
				RegistrationDate:   row.RegistrationDate,
			})
			resp.PaymentAmount += row.PaymentAmount
		}

		resp.PaymentProofURLs = []string{}

		// Fetch payment transaction for the first registration row (most common)
		var transaction models.PaymentTransaction
		errTx := db.Get(&transaction, `SELECT * FROM payment_transactions WHERE registration_id = ? ORDER BY created_at DESC LIMIT 1`, firstRow.ID)
		if errTx == nil {
			resp.Transaction = &transaction
		}

		c.JSON(http.StatusOK, resp)
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil jadwal event", "details": err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus jadwal lama", "details": err.Error()})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan jadwal", "details": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Jadwal event berhasil diperbarui",
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil kategori event", "details": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid", "details": err.Error()})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid", "details": err.Error()})
			return
		}

		var exists bool
		if err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM event_category_refs WHERE uuid = ?)`, id); err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
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

		c.JSON(http.StatusOK, gin.H{"message": "Kategori berhasil diperbarui"})
	}
}

// PublishEvent changes event status to published
func PublishEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		_, err := db.Exec("UPDATE events SET status = 'published' WHERE uuid = ?", eventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mempublikasikan event"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "event_published", "event", eventID, "Published event", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Event berhasil dipublikasikan"})
	}
}

// RegisterParticipant registers a participant for a event
func RegisterParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			AthleteID          string   `json:"athlete_id" binding:"required"`
			EventCategoryID    string   `json:"event_category_id"`
			EventCategoryIDs   []string `json:"event_category_ids"`
			PaymentAmount      float64  `json:"payment_amount"`
			PaymentStatus      string   `json:"payment_status"`
			RegistrationSource string   `json:"registration_source"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			msg := err.Error()
			if strings.Contains(msg, "AthleteID") || strings.Contains(msg, "athlete_id") {
				msg = "athlete_id wajib diisi"
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": msg, "details": err.Error()})
			return
		}
		req.AthleteID = strings.TrimSpace(req.AthleteID)

		// Combine single ID and multi IDs
		allCategoryIDs := []string{}
		if strings.TrimSpace(req.EventCategoryID) != "" {
			allCategoryIDs = append(allCategoryIDs, strings.TrimSpace(req.EventCategoryID))
		}
		for _, id := range req.EventCategoryIDs {
			trimmed := strings.TrimSpace(id)
			if trimmed != "" {
				// Avoid duplicates
				duplicate := false
				for _, existing := range allCategoryIDs {
					if existing == trimmed {
						duplicate = true
						break
					}
				}
				if !duplicate {
					allCategoryIDs = append(allCategoryIDs, trimmed)
				}
			}
		}

		if req.AthleteID == "" || len(allCategoryIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data yang diperlukan tidak lengkap", "details": "athlete_id dan setidaknya satu event_category_id diperlukan"})
			return
		}

		// Resolve event slug to UUID and get organizer ID
		var event struct {
			UUID        string `db:"uuid"`
			OrganizerID string `db:"organizer_id"`
		}
		err := db.Get(&event, `SELECT uuid, organizer_id FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		actualEventID := event.UUID
		organizerID := event.OrganizerID

		// Verification sub status organizer (Organization or Club)
		var orgStatus string
		db.Get(&orgStatus, `
			SELECT COALESCE(s, 'trial') FROM (
				SELECT subscription_status as s FROM organizations WHERE uuid = ?
				UNION ALL
				SELECT 'trial' as s FROM clubs WHERE uuid = ?
			) combined LIMIT 1`, organizerID, organizerID)

		if orgStatus != "" && orgStatus != "active" && orgStatus != "trial" {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":   "Pendaftaran ditutup sementara",
				"code":    "organizer_subscription_expired",
				"message": "Pendaftaran peserta untuk event ini ditutup sementara oleh sistem karena masa berlaku layanan penyelenggara telah berakhir.",
			})
			return
		}

		var archerUUID string
		err = db.Get(&archerUUID, "SELECT uuid FROM archers WHERE uuid = ? OR id = ?", req.AthleteID, req.AthleteID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Archer tidak ditemukan"})
			return
		}

		// Use transaction to ensure all registrations succeed or none
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		registrationDate := time.Now()

		// Determine payment status
		paymentStatus := "unpaid"
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("role")
		orgID, _ := c.Get("org_id")

		// isPrivileged: admin always; organizer if their org_id matches the event's organizer_id
		isPrivileged := userRole == "admin" || (userRole == "organizer" && orgID != nil && fmt.Sprintf("%v", orgID) == event.OrganizerID)

		// Only allow admin or the event organizer to set status directly
		if req.PaymentStatus != "" && isPrivileged {
			paymentStatus = req.PaymentStatus
		}

		// Determine registration source
		registrationSource := "self_register"
		if isPrivileged {
			if req.RegistrationSource != "" {
				registrationSource = req.RegistrationSource
			} else {
				registrationSource = "admin_created"
			}
		}

		// Prepare QR code if status is lunas
		var qrRaw *string
		if paymentStatus == "lunas" || paymentStatus == "paid" {
			// Check if archer already has a QR for this event
			var existingQR sql.NullString
			err = tx.Get(&existingQR, "SELECT qr_raw FROM event_participants WHERE event_id = ? AND archer_id = ? AND qr_raw IS NOT NULL LIMIT 1", actualEventID, archerUUID)
			if err == nil && existingQR.Valid {
				qrRaw = &existingQR.String
			} else {
				randomQR := uuid.New().String()
				qrRaw = &randomQR
			}
		}

		var firstParticipantUUID string
		registeredCategoryIDs := []string{}
		for i, catID := range allCategoryIDs {
			// Check if already registered for THIS category
			var exists bool
			err = tx.Get(&exists, `
				SELECT EXISTS(SELECT 1 FROM event_participants 
				WHERE event_id = ? AND archer_id = ? AND category_id = ?)
			`, actualEventID, archerUUID, catID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengecek status pendaftaran", "details": err.Error()})
				return
			}

			if exists {
				continue // Skip if already registered for this category
			}

			participantUUID := uuid.New().String()
			if i == 0 || firstParticipantUUID == "" {
				firstParticipantUUID = participantUUID
			}

			_, err = tx.Exec(`
				INSERT INTO event_participants (
					uuid, event_id, archer_id, category_id, 
					registration_date, payment_status, payment_amount, qr_raw,
					registration_source
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, participantUUID, actualEventID, archerUUID, catID, registrationDate, paymentStatus, req.PaymentAmount, qrRaw, registrationSource)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan peserta", "details": err.Error()})
				return
			}

			// Log activity
			utils.LogActivity(tx, fmt.Sprintf("%v", userID), actualEventID, "participant_registered", "event_participant", participantUUID, "Registered participant for event category: "+catID, c.ClientIP(), c.Request.UserAgent())

			registeredCategoryIDs = append(registeredCategoryIDs, catID)
		}

		if len(registeredCategoryIDs) == 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Pemanah sudah terdaftar di semua kategori pilihan pada event ini"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pendaftaran"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":         "Pendaftaran berhasil",
			"registration_id": firstParticipantUUID,
			"category_ids":    registeredCategoryIDs,
		})
	}
}

// BatchRegisterParticipants registers multiple archers for an event in a single transaction
func BatchRegisterParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			AthleteIDs         []string `json:"athlete_ids" binding:"required"`
			EventCategoryIDs   []string `json:"event_category_ids" binding:"required"`
			PaymentAmount      float64  `json:"payment_amount"`
			PaymentStatus      string   `json:"payment_status"`
			RegistrationSource string   `json:"registration_source"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if len(req.AthleteIDs) == 0 || len(req.EventCategoryIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "athlete_ids and event_category_ids are required"})
			return
		}

		// Resolve event
		var event struct {
			UUID        string `db:"uuid"`
			OrganizerID string `db:"organizer_id"`
		}
		if err := db.Get(&event, `SELECT uuid, organizer_id FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}
		actualEventID := event.UUID

		// Determine privileges
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("role")
		orgID, _ := c.Get("org_id")

		isPrivileged := userRole == "admin" || (userRole == "organizer" && orgID != nil && fmt.Sprintf("%v", orgID) == event.OrganizerID)

		paymentStatus := "unpaid"

		registrationSource := "self_register"
		if isPrivileged {
			if req.RegistrationSource != "" {
				registrationSource = req.RegistrationSource
			} else {
				registrationSource = "admin_created"
			}
		}


		// Resolve all archer UUIDs in one query
		cleanedIDs := make([]string, 0, len(req.AthleteIDs))
		for _, id := range req.AthleteIDs {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				cleanedIDs = append(cleanedIDs, trimmed)
			}
		}

		type archerRow struct {
			UUID string `db:"uuid"`
			ID   string `db:"id"`
		}
		query, args, err := sqlx.In(`SELECT uuid, id FROM archers WHERE uuid IN (?) OR id IN (?)`, cleanedIDs, cleanedIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build archer query"})
			return
		}
		query = db.Rebind(query)
		var archerRows []archerRow
		if err := db.Select(&archerRows, query, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve archers"})
			return
		}

		// Map input IDs â†’ resolved UUIDs (deduplicate)
		seenUUIDs := map[string]bool{}
		archerUUIDs := []string{}
		for _, row := range archerRows {
			if !seenUUIDs[row.UUID] {
				seenUUIDs[row.UUID] = true
				archerUUIDs = append(archerUUIDs, row.UUID)
			}
		}

		if len(archerUUIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada data pemanah yang valid"})
			return
		}

		// Deduplicate category IDs
		seenCats := map[string]bool{}
		allCategoryIDs := []string{}
		for _, id := range req.EventCategoryIDs {
			if trimmed := strings.TrimSpace(id); trimmed != "" && !seenCats[trimmed] {
				seenCats[trimmed] = true
				allCategoryIDs = append(allCategoryIDs, trimmed)
			}
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		registrationDate := time.Now()
		registeredCount := 0
		skippedCount := 0

		for _, archerUUID := range archerUUIDs {
			// Get or generate a shared QR for this archerÃ—event if status is lunas
			var qrRaw *string
			if paymentStatus == "lunas" || paymentStatus == "paid" {
				var existingQR sql.NullString
				_ = tx.Get(&existingQR, "SELECT qr_raw FROM event_participants WHERE event_id = ? AND archer_id = ? AND qr_raw IS NOT NULL LIMIT 1", actualEventID, archerUUID)
				if existingQR.Valid {
					qrRaw = &existingQR.String
				} else {
					randomQR := uuid.New().String()
					qrRaw = &randomQR
				}
			}

			for _, catID := range allCategoryIDs {
				var exists bool
				if err := tx.Get(&exists, `SELECT EXISTS(SELECT 1 FROM event_participants WHERE event_id = ? AND archer_id = ? AND category_id = ?)`, actualEventID, archerUUID, catID); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check registration status"})
					return
				}
				if exists {
					skippedCount++
					continue
				}

				participantUUID := uuid.New().String()
				_, err = tx.Exec(`
						INSERT INTO event_participants (
						uuid, event_id, archer_id, category_id,
						registration_date, payment_status, payment_amount, qr_raw,
						registration_source
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, participantUUID, actualEventID, archerUUID, catID, registrationDate, paymentStatus, req.PaymentAmount, qrRaw, registrationSource)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan peserta", "details": err.Error()})
					return
				}

				utils.LogActivity(tx, fmt.Sprintf("%v", userID), actualEventID, "participant_registered", "event_participant", participantUUID, "Batch registered participant for event category: "+catID, c.ClientIP(), c.Request.UserAgent())
				registeredCount++
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit batch registration"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":    "Pendaftaran massal selesai",
			"registered": registeredCount,
			"skipped":    skippedCount,
		})
	}
}

// UnregisterFromEvent allows an archer to cancel ALL their registrations from an event by event ID.
// Uses DELETE /events/:id/participants/me
func UnregisterFromEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		uid := userID.(string)

		// Resolve event slug/UUID
		var actualEventID string
		if err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Resolve archer UUID for this user
		var archerID string
		if err := db.Get(&archerID, `SELECT uuid FROM archers WHERE uuid = ? LIMIT 1`, uid); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profil pemanah tidak ditemukan"})
			return
		}

		// Find all registrations for this archer in this event
		type RegInfo struct {
			UUID          string `db:"uuid"`
			PaymentStatus string `db:"payment_status"`
		}
		var regs []RegInfo
		if err := db.Select(&regs, `SELECT uuid, payment_status FROM event_participants WHERE event_id = ? AND archer_id = ?`, actualEventID, archerID); err != nil || len(regs) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registrasi tidak ditemukan untuk event ini"})
			return
		}

		// Block cancellation if any category is already paid
		for _, r := range regs {
			if r.PaymentStatus == "lunas" || r.PaymentStatus == "paid" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak dapat membatalkan pendaftaran yang sudah lunas. Hubungi panitia."})
				return
			}
		}

		// Delete all registrations for this archer in this event
		if _, err := db.Exec(`DELETE FROM event_participants WHERE event_id = ? AND archer_id = ?`, actualEventID, archerID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membatalkan pendaftaran"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Pendaftaran berhasil dibatalkan"})
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
		err = db.Get(&userArcherID, "SELECT uuid FROM archers WHERE uuid = ? LIMIT 1", userID)
		if err != nil || userArcherID != archerID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat membatalkan pendaftaran sendiri"})
			return
		}

		// Check if already approved - can't cancel approved registrations
		var status string
		err = db.Get(&status, "SELECT status FROM event_participants WHERE uuid = ?", participantID)
		if err == nil && (status == "registered" || status == "Terdaftar") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot cancel an approved registration. Please contact the organizer."})
			return
		}

		// Delete the participant registration
		_, err = db.Exec("DELETE FROM event_participants WHERE uuid = ?", participantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel registration"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Pendaftaran berhasil dibatalkan"})
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

		// Resolve participant (support UUID, Username, or Athlete Code)
		var pInfo struct {
			UUID     string `db:"uuid"`
			ArcherID string `db:"archer_id"`
			FullName string `db:"full_name"`
		}

		err = db.Get(&pInfo, `
			SELECT tp.uuid, tp.archer_id, a.full_name FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			WHERE tp.event_id = ? AND (tp.uuid = ? OR a.username = ? OR a.id = ?)
			LIMIT 1
		`, actualEventID, participantID, participantID, participantID)

		if err != nil {
			fmt.Printf("[DEBUG] Delete lookup failed for Event: %s, ID: %s. Error: %v\n", actualEventID, participantID, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found", "details": err.Error()})
			return
		}

		archerID := pInfo.ArcherID

		fmt.Printf("[DEBUG] Found participant to delete: %s (Archer: %s) - removing ALL registrations for this archer in event\n", pInfo.FullName, archerID)

		// Check if this archer (any of their participant rows) is in any elimination match
		var inMatch bool
		err = db.Get(&inMatch, `
			SELECT EXISTS(
				SELECT 1 FROM elimination_matches em
				JOIN elimination_entries ee ON (em.entry_a_uuid = ee.uuid OR em.entry_b_uuid = ee.uuid)
				WHERE ee.participant_uuid IN (SELECT uuid FROM event_participants WHERE archer_id = ? AND event_id = ?)
			)
		`, archerID, actualEventID)

		if err == nil && inMatch {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Tidak dapat mengeluarkan peserta: Peserta sudah terdaftar dalam babak eliminasi. Silakan hapus mereka dari bracket eliminasi terlebih dahulu.",
			})
			return
		}

		// Start transaction for cleanup
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// 1. Cleanup qualification data for ALL participant rows of this archer in this event
		// Arrows first (child table)
		tx.Exec(`
			DELETE FROM qualification_arrow_scores 
			WHERE end_score_uuid IN (
				SELECT uuid FROM qualification_end_scores 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE archer_id = ? AND event_id = ?)
			)
		`, archerID, actualEventID)

		// End scores
		tx.Exec(`
			DELETE FROM qualification_end_scores 
			WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE archer_id = ? AND event_id = ?)
		`, archerID, actualEventID)

		// 2. Delete qualification target assignments
		tx.Exec(`
			DELETE FROM qualification_target_assignments 
			WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE archer_id = ? AND event_id = ?)
		`, archerID, actualEventID)

		// 3. Cleanup elimination entries if any
		tx.Exec(`
			DELETE FROM elimination_entries 
			WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE archer_id = ? AND event_id = ?)
		`, archerID, actualEventID)

		// 4. Delete ALL event_participants for this archer in this event (handles multi-category registrations)
		_, err = tx.Exec("DELETE FROM event_participants WHERE archer_id = ? AND event_id = ?", archerID, actualEventID)
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
			utils.LogActivity(db, userID.(string), actualEventID, "participant_kicked", "event", actualEventID, "Kicked participant: "+archerID, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Peserta berhasil dikeluarkan dari event"})
	}
}

// UpdateEventParticipant updates an existing event participant
func UpdateEventParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")

		var req struct {
			CategoryID          *string   `json:"category_id"`
			CategoryIDs         []string  `json:"category_ids"`
			TargetName          *string   `json:"target_name"`
			BackNumber          *string   `json:"back_number"`
			PaymentStatus       *string   `json:"payment_status"`
			PaymentAmount       *float64  `json:"payment_amount"`
			PaymentProofURLs    *[]string `json:"payment_proof_urls"`
			AccreditationStatus *string   `json:"accreditation_status"`
			IsVerified          *bool     `json:"is_verified"`
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

		if req.CategoryID != nil || len(req.CategoryIDs) > 0 {
			archerID := pInfo.ArcherID
			if archerID == nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Pemanah tidak ditemukan untuk peserta ini"})
				return
			}

			// Handle multi-category sync if CategoryIDs is provided
			if len(req.CategoryIDs) > 0 {
				// Get current category IDs for this archer in this event
				var currentIDs []string
				err = db.Select(&currentIDs, "SELECT category_id FROM event_participants WHERE archer_id = ? AND event_id = ?", *archerID, actualEventID)

				// Identify to add
				toAdd := []string{}
				for _, id := range req.CategoryIDs {
					found := false
					for _, curr := range currentIDs {
						if curr == id {
							found = true
							break
						}
					}
					if !found {
						toAdd = append(toAdd, id)
					}
				}

				// Identify to remove
				toRemove := []string{}
				for _, curr := range currentIDs {
					found := false
					for _, id := range req.CategoryIDs {
						if id == curr {
							found = true
							break
						}
					}
					if !found {
						toRemove = append(toRemove, curr)
					}
				}

				// Start transaction for sync
				tx, err := db.Beginx()
				if err == nil {
					defer tx.Rollback()

					// Add new registrations
					// First, look up any existing qr_raw for this archer in this event
					var existingQrRaw *string
					tx.Get(&existingQrRaw, `SELECT qr_raw FROM event_participants WHERE archer_id = ? AND event_id = ? AND qr_raw IS NOT NULL LIMIT 1`, *archerID, actualEventID)

					for _, catID := range toAdd {
						// Verify category
						var catExists bool
						tx.Get(&catExists, "SELECT EXISTS(SELECT 1 FROM event_categories WHERE uuid = ? AND event_id = ?)", catID, actualEventID)
						if catExists {
							newUUID := uuid.New().String()
							tx.Exec(`INSERT INTO event_participants (uuid, event_id, archer_id, category_id, registration_date, payment_status, payment_amount, qr_raw, registration_source) 
							VALUES (?, ?, ?, ?, NOW(), ?, ?, ?, 'admin_created')`,
								newUUID, actualEventID, *archerID, catID,
								models.FromPtr(req.PaymentStatus), models.FromPtrFloat(req.PaymentAmount), existingQrRaw)
						}
					}

					// Remove registrations
					for _, catID := range toRemove {
						tx.Exec("DELETE FROM event_participants WHERE archer_id = ? AND event_id = ? AND category_id = ?", *archerID, actualEventID, catID)
					}

					tx.Commit()
				}
			} else if req.CategoryID != nil {
				// Old behavior: single category update
				var categoryExists bool
				err = db.Get(&categoryExists, `
					SELECT EXISTS(SELECT 1 FROM event_categories 
					WHERE uuid = ? AND event_id = ?)
				`, *req.CategoryID, actualEventID)
				if err == nil && categoryExists {
					query += ", category_id = ?"
					args = append(args, *req.CategoryID)
				}
			}
		}
		if req.TargetName != nil {
			query += ", target_name = ?"
			args = append(args, *req.TargetName)
		}
		if req.BackNumber != nil {
			query += ", back_number = ?"
			args = append(args, *req.BackNumber)
		}


		if req.PaymentStatus != nil {
			query += ", payment_status = ?"
			args = append(args, *req.PaymentStatus)

			if *req.PaymentStatus == "lunas" || *req.PaymentStatus == "paid" {
				// Generate QR raw string when payment is lunas (paid) for all entries if missing
				var currentQR sql.NullString
				err = db.Get(&currentQR, "SELECT qr_raw FROM event_participants WHERE event_id = ? AND archer_id = ? AND qr_raw IS NOT NULL LIMIT 1", actualEventID, *pInfo.ArcherID)
				if err != nil || !currentQR.Valid {
					// Generate random QR string using uuid
					qrRaw := uuid.New().String()
					query += ", qr_raw = ?"
					args = append(args, qrRaw)
				}
			}
		}

		// Remove status from direct updates - it's now managed by payment_status
		// if req.Status != nil { ... } - REMOVED
		if req.PaymentAmount != nil {
			query += ", payment_amount = ?"
			args = append(args, *req.PaymentAmount)
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

		query += " WHERE event_id = ? AND archer_id = ?"
		args = append(args, actualEventID, *pInfo.ArcherID)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data peserta", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), actualEventID, "participant_updated", "event_participant", actualParticipantID, "Updated participant", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Data peserta berhasil diperbarui"})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe event tidak valid"})
			return
		}

		if eventTypeCode == "mixed_team" {
			if req.GenderDivisionUUID == "" {
				var mixedUUID string
				err = db.Get(&mixedUUID, "SELECT uuid FROM ref_gender_divisions WHERE code = 'mixed'")
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Divisi gender mixed tidak ditemukan dalam sistem"})
					return
				}
				req.GenderDivisionUUID = mixedUUID
			}
		}

		// Check if event exists
		var eventExists bool
		err = db.Get(&eventExists, `SELECT EXISTS(SELECT 1 FROM events WHERE uuid = ?)`, eventID)
		if err != nil || !eventExists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
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
			"message": fmt.Sprintf("Berhasil membuat %d kategori", count),
			"count":   count,
		})
	}
}

// CreateEventCategory creates a single event category
func CreateEventCategory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			DivisionUUID       string  `json:"division_uuid" binding:"required"`
			CategoryUUID       string  `json:"category_uuid" binding:"required"`
			CategoryNameCustom *string `json:"category_name_custom"`
			EventTypeUUID      string  `json:"event_type_uuid" binding:"required"`
			GenderDivisionUUID string  `json:"gender_division_uuid"`
			MaxParticipants    *int    `json:"max_participants"`
			Status             string  `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Resolve event type code to enforce team size
		var eventTypeCode string
		err := db.Get(&eventTypeCode, "SELECT code FROM ref_event_types WHERE uuid = ?", req.EventTypeUUID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe event tidak valid"})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Divisi gender mixed tidak ditemukan"})
				return
			}
		} else {

			// Individual or Team must have a specific gender (Men/Women)
			if req.GenderDivisionUUID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Divisi gender wajib diisi untuk tipe kategori ini"})
				return
			}

			// Ensure it's not "mixed"
			var genderCode string
			db.Get(&genderCode, "SELECT code FROM ref_gender_divisions WHERE uuid = ?", req.GenderDivisionUUID)
			if genderCode == "mixed" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Hanya mixed team yang dapat menggunakan divisi gender 'Mixed'"})
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
			c.JSON(http.StatusConflict, gin.H{"error": "Kategori sudah ada di event ini"})
			return
		}

		status := req.Status
		if status == "" {
			status = "active"
		}

		catEventID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO event_categories (
				uuid, event_id, division_uuid, category_uuid, category_name_custom, event_type_uuid, gender_division_uuid,
				max_participants, status
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, catEventID, actualEventID, req.DivisionUUID, req.CategoryUUID, req.CategoryNameCustom, req.EventTypeUUID, req.GenderDivisionUUID, req.MaxParticipants, status)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create category", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "category_created", "event_category", catEventID, "Created event category", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      catEventID,
			"message": "Kategori berhasil dibuat",
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
			CategoryNameCustom *string `json:"category_name_custom"`
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
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
		if req.CategoryNameCustom != nil {
			query += ", category_name_custom = ?"
			args = append(args, req.CategoryNameCustom)
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

		c.JSON(http.StatusOK, gin.H{"message": "Kategori berhasil diperbarui"})
	}
}

// GetEventCategoryDetails returns detailed info about a category's usage across the system
func GetEventCategoryDetails(db *sqlx.DB) gin.HandlerFunc {
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

		type CategoryInfo struct {
			UUID               string  `db:"uuid" json:"id"`
			DivisionName       string  `db:"division_name" json:"division_name"`
			CategoryName       string  `db:"category_name" json:"category_name"`
			CategoryNameCustom *string `db:"category_name_custom" json:"category_name_custom"`
			GenderDivisionName string  `db:"gender_division_name" json:"gender_division_name"`
			EventTypeName      string  `db:"event_type_name" json:"event_type_name"`
		}

		var info CategoryInfo
		err = db.Get(&info, `
			SELECT ec.uuid, rbt.name as division_name, rag.name as category_name, 
			       ec.category_name_custom, rgd.name as gender_division_name, ret.name as event_type_name
			FROM event_categories ec
			JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			JOIN ref_event_types ret ON ec.event_type_uuid = ret.uuid
			WHERE ec.uuid = ? AND ec.event_id = ?
		`, categoryID, actualEventID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
			return
		}

		// Count participants
		var participantCount int
		db.Get(&participantCount, "SELECT COUNT(*) FROM event_participants WHERE category_id = ?", categoryID)

		// Count qualification sessions linked
		var sessionCount int
		db.Get(&sessionCount, "SELECT COUNT(*) FROM qualification_session_categories WHERE category_uuid = ?", categoryID)

		// Count qualification scores
		var scoreCount int
		db.Get(&scoreCount, `
			SELECT COUNT(*) FROM qualification_end_scores 
			WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE category_id = ?)
		`, categoryID)

		// Count elimination brackets
		var bracketCount int
		db.Get(&bracketCount, "SELECT COUNT(*) FROM elimination_brackets WHERE category_uuid = ?", categoryID)

		// Count teams
		var teamCount int
		db.Get(&teamCount, "SELECT COUNT(*) FROM teams WHERE category_uuid = ?", categoryID)

		c.JSON(http.StatusOK, gin.H{
			"category":          info,
			"participant_count": participantCount,
			"session_count":     sessionCount,
			"score_count":       scoreCount,
			"bracket_count":     bracketCount,
			"team_count":        teamCount,
			"is_deletable":      participantCount == 0 && sessionCount == 0 && bracketCount == 0 && teamCount == 0,
		})
	}
}

// DeleteEventCategory deletes a single event category and all its related data (Destructive)
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

		// 1. Check if category exists
		var exists bool
		err = db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM event_categories WHERE uuid = ? AND event_id = ?)", categoryID, actualEventID)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
			return
		}

		// 2. Perform Cascading Deletion
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// A. Cleanup Qualification Scores & Assignments
		// Arrows must be deleted before their parent ends
		tx.Exec("DELETE FROM qualification_arrow_scores WHERE end_score_uuid IN (SELECT uuid FROM qualification_end_scores WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE category_id = ?))", categoryID)
		tx.Exec("DELETE FROM qualification_end_scores WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE category_id = ?)", categoryID)
		tx.Exec("DELETE FROM qualification_target_assignments WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE category_id = ?)", categoryID)
		tx.Exec("DELETE FROM qualification_session_categories WHERE category_uuid = ?", categoryID)

		// B. Cleanup Elimination Data (Deep Cleanup)
		// Delete arrow scores first (lowest level)
		tx.Exec(`
			DELETE FROM elimination_match_arrow_scores 
			WHERE match_end_uuid IN (
				SELECT uuid FROM elimination_match_ends 
				WHERE match_uuid IN (
					SELECT uuid FROM elimination_matches 
					WHERE bracket_uuid IN (
						SELECT uuid FROM elimination_brackets WHERE category_uuid = ?
					)
				)
			)
		`, categoryID)

		// Delete ends
		tx.Exec(`
			DELETE FROM elimination_match_ends 
			WHERE match_uuid IN (
				SELECT uuid FROM elimination_matches 
				WHERE bracket_uuid IN (
					SELECT uuid FROM elimination_brackets WHERE category_uuid = ?
				)
			)
		`, categoryID)

		// Delete matches
		tx.Exec("DELETE FROM elimination_matches WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE category_uuid = ?)", categoryID)

		// Delete entries and brackets
		tx.Exec("DELETE FROM elimination_entries WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE category_uuid = ?)", categoryID)
		tx.Exec("DELETE FROM elimination_brackets WHERE category_uuid = ?", categoryID)

		// C. Cleanup Team Data
		tx.Exec("DELETE FROM team_members WHERE team_id IN (SELECT uuid FROM teams WHERE event_id = ?)", categoryID)
		tx.Exec("DELETE FROM teams WHERE event_id = ?", categoryID)

		// D. Cleanup Board & Verification Data
		tx.Exec("DELETE FROM target_board_qualification WHERE category_uuid = ?", categoryID)
		tx.Exec("DELETE FROM target_board_elimination WHERE category_uuid = ?", categoryID)

		// E. Cleanup Participants
		tx.Exec("DELETE FROM event_participants WHERE category_id = ?", categoryID)

		// F. Delete the Category
		_, err = tx.Exec("DELETE FROM event_categories WHERE uuid = ? AND event_id = ?", categoryID, actualEventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete category", "details": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit destructive deletion"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "category_deleted_destructive", "event_category", categoryID, "Permanently deleted category and all related data", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Kategori dan seluruh data terkait berhasil dihapus secara permanen"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil gambar event", "details": err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus gambar lama", "details": err.Error()})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan gambar event", "details": err.Error()})
				return
			}
		}

		// Log activity
		utils.LogActivity(db, userID.(string), eventID, "event_images_updated", "event", eventID, fmt.Sprintf("Updated %d event images", len(req.Images)), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{
			"message": "Gambar event berhasil diperbarui",
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data tim", "details": err.Error()})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		sortBy := c.DefaultQuery("sort_by", "created_at")
		order := strings.ToUpper(c.DefaultQuery("order", "DESC"))
		status := c.Query("status")
		search := c.Query("search")
		limit, offset, page := utils.GetPaginationParams(c)

		// Validate sortBy to prevent SQL injection
		allowedSortFields := map[string]string{
			"name":              "t.name",
			"start_date":        "t.start_date",
			"venue":             "t.venue",
			"status":            "t.status",
			"created_at":        "t.created_at",
			"participant_count": "participant_count",
			"event_count":       "event_count",
		}

		dbSortField, ok := allowedSortFields[sortBy]
		if !ok {
			dbSortField = "t.created_at"
		}

		if order != "ASC" && order != "DESC" {
			order = "DESC"
		}

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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung jumlah event", "details": err.Error()})
			return
		}

		query := fmt.Sprintf(`
			SELECT 
				t.*,
				u.full_name as organizer_name,
				u.email as organizer_email,
				u.slug as organizer_slug,
				u.avatar_url as organizer_avatar_url,
				COUNT(DISTINCT tp.archer_id) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, NULL as email, slug, logo_url as avatar_url FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.uuid = tp.event_id
			LEFT JOIN event_categories te ON t.uuid = te.event_id
			%s
			GROUP BY t.uuid
			ORDER BY %s %s
			LIMIT ? OFFSET ?
		`, whereClause, dbSortField, order)
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

		meta := utils.CalculatePagination(total, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{
			"data":   events,
			"events": events,
			"total":  total,
			"meta":   meta,
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "QR code wajib diisi"})
			return
		}

		// Find participant by qr_raw
		type ParticipantInfo struct {
			UUID          string  `db:"uuid"`
			FullName      string  `db:"full_name"`
			Email         string  `db:"email"`
			ClubName      *string `db:"club_name"`
			DivisionName  string  `db:"division_name"`
			CategoryName  string  `db:"category_name"`
			EventName     string  `db:"event_name"`
			PaymentStatus string  `db:"payment_status"`
		}

		var participant ParticipantInfo
		err := db.Get(&participant, `
			SELECT 
				ep.uuid,
				a.full_name,
				a.email,
				c.name as club_name,
				COALESCE(d.name, '') as division_name,
				COALESCE(ec.category_name_custom, ag.name, '') as category_name,
				e.name as event_name,
				COALESCE(ep.payment_status, 'pending') as payment_status
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kesalahan database", "details": err.Error()})
			return
		}

		// Check if participant is registered (payment_status = "lunas" or "paid")
		if participant.PaymentStatus != "lunas" && participant.PaymentStatus != "paid" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Peserta belum disetujui atau belum lunas. Status: " + participant.PaymentStatus,
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui pendaftaran", "details": err.Error()})
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

// ExportParticipantsCSV exports all participants of an event to a CSV file
func ExportParticipantsCSV(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Resolve event
		var event struct {
			UUID string `db:"uuid"`
			Name string `db:"name"`
		}
		err := db.Get(&event, `SELECT uuid, name FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		type Participant struct {
			AthleteCode        string `db:"athlete_code"`
			FullName           string `db:"full_name"`
			Email              string `db:"email"`
			ClubName           string `db:"club_name"`
			City               string `db:"city"`
			PaymentStatus      string `db:"payment_status"`
			RegistrationSource string `db:"registration_source"`
			RegistrationDate   string `db:"registration_date"`
			TargetNames        string `db:"target_names"`
			Categories         string `db:"categories"`
			TotalScore         int    `db:"total_score"`
			TotalX             int    `db:"total_x"`
		}

		var participants []Participant
		query := `
			SELECT 
				a.id as athlete_code,
				a.full_name,
				COALESCE(a.email, '') as email,
				COALESCE(cl.name, '') as club_name,
				COALESCE(a.city, '') as city,
				COALESCE(MAX(tp.payment_status), 'pending') as payment_status,
				COALESCE(MAX(tp.registration_source), 'self_register') as registration_source,
				COALESCE(DATE_FORMAT(MIN(tp.registration_date), '%Y-%m-%d %H:%i:%s'), '') as registration_date,
				GROUP_CONCAT(DISTINCT COALESCE(tp.target_name, '') ORDER BY tp.target_name SEPARATOR ', ') as target_names,
				GROUP_CONCAT(
					DISTINCT TRIM(CONCAT_WS(' - ',
						NULLIF(COALESCE(d.name, ''), ''),
						NULLIF(COALESCE(te.category_name_custom, c.name, ''), ''),
						NULLIF(COALESCE(et.name, ''), ''),
						NULLIF(COALESCE(gd.name, ''), '')
					))
					SEPARATOR '; '
				) as categories,
				COALESCE(SUM(scores.total_score), 0) as total_score,
				COALESCE(SUM(scores.total_x), 0) as total_x
			FROM event_participants tp
			JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_event_types et ON te.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			LEFT JOIN (
				SELECT participant_uuid, SUM(total_score_end) as total_score, SUM(x_count_end) as total_x
				FROM qualification_end_scores
				GROUP BY participant_uuid
			) scores ON tp.uuid = scores.participant_uuid
			WHERE tp.event_id = ?
			GROUP BY a.uuid, a.id, a.full_name, a.email, cl.name, a.city
			ORDER BY a.full_name ASC
		`
		err = db.Select(&participants, query, event.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants", "details": err.Error()})
			return
		}

		// Set response headers
		fileName := fmt.Sprintf("participants-%s-%s.csv", strings.ReplaceAll(strings.ToLower(event.Name), " ", "-"), time.Now().Format("20060102"))
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		c.Header("Content-Type", "text/csv")

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		capitalizeWords := func(s string) string {
			s = strings.TrimSpace(strings.ToLower(s))
			if s == "" {
				return ""
			}
			parts := strings.Fields(s)
			for i, p := range parts {
				if len(p) == 0 {
					continue
				}
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
			return strings.Join(parts, " ")
		}

		formatPaymentStatus := func(s string) string {
			s = strings.ToLower(strings.TrimSpace(s))
			switch s {
			case "lunas", "paid":
				return "Paid"
			case "menunggu", "menunggu acc", "pending":
				return "Pending"
			case "unpaid", "belum_lunas":
				return "Unpaid"
			case "expired":
				return "Expired"
			default:
				return capitalizeWords(s)
			}
		}

		formatRegistrationSource := func(s string) string {
			s = strings.ToLower(strings.TrimSpace(s))
			switch s {
			case "self_register":
				return "Self Registered"
			case "admin_created":
				return "Added by Admin"
			case "invited":
				return "Invited"
			default:
				return capitalizeWords(strings.ReplaceAll(s, "_", " "))
			}
		}

		// Write header
		writer.Write([]string{
			"No",
			"Kode Atlet",
			"Nama Peserta",
			"Email",
			"Klub",
			"Kota",
			"Status Pembayaran",
			"Sumber Registrasi",
			"Tanggal Pendaftaran",
			"Target",
			"Kategori",
			"Total Skor",
			"Total X",
		})

		for i, p := range participants {
			writer.Write([]string{
				strconv.Itoa(i + 1),
				p.AthleteCode,
				p.FullName,
				p.Email,
				capitalizeWords(p.ClubName),
				capitalizeWords(p.City),
				formatPaymentStatus(p.PaymentStatus),
				formatRegistrationSource(p.RegistrationSource),
				p.RegistrationDate,
				p.TargetNames,
				capitalizeWords(p.Categories),
				strconv.Itoa(p.TotalScore),
				strconv.Itoa(p.TotalX),
			})
		}
	}
}

func GetEventParticipantList(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		listType := c.Query("type") // alphabetical or by-club
		autoprint := c.Query("autoprint")

		// Internal PHP Printout Service URL
		printoutURL := fmt.Sprintf("http://localhost:8002/api/v1/events/%s/participants/printout?type=%s", eventID, listType)
		if autoprint != "" {
			printoutURL += "&autoprint=" + autoprint
		}

		// Forward the request to the PHP service
		resp, err := http.Get(printoutURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghubungi layanan cetak internal", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		// Set headers from the PHP service response
		for k, v := range resp.Header {
			for _, val := range v {
				c.Header(k, val)
			}
		}
		c.Status(resp.StatusCode)

		// Stream the PDF response
		c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
	}
}

// ResetEventData allows organizers or admins to reset specific data of an event
func ResetEventData(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req struct {
			Target      string `json:"target" binding:"required"`
			ConfirmText string `json:"confirm_text" binding:"required"`
			Code        string `json:"code" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid", "details": err.Error()})
			return
		}

		if strings.ToUpper(req.ConfirmText) != "RESET" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Teks konfirmasi harus 'RESET'"})
			return
		}

		// Resolve event slug to UUID
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Verify the verification code
		var savedCode struct {
			Code      string    `db:"code"`
			ExpiresAt time.Time `db:"expires_at"`
		}
		err = db.Get(&savedCode, `
			SELECT code, expires_at FROM event_reset_codes 
			WHERE event_id = ? AND user_id = ? AND code = ?
			LIMIT 1
		`, eventID, userID.(string), req.Code)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode verifikasi salah atau tidak ditemukan"})
			return
		}

		if time.Now().After(savedCode.ExpiresAt) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Kode verifikasi telah kadaluarsa"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		switch req.Target {
		case "qualification":
			// 1. Delete arrows
			_, err = tx.Exec(`
				DELETE FROM qualification_arrow_scores 
				WHERE end_score_uuid IN (
					SELECT uuid FROM qualification_end_scores 
					WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
				)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus skor anak panah", "details": err.Error()})
				return
			}

			// 2. Delete end scores
			_, err = tx.Exec(`
				DELETE FROM qualification_end_scores 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus skor kualifikasi", "details": err.Error()})
				return
			}

			// 3. Delete target assignments
			_, err = tx.Exec(`
				DELETE FROM qualification_target_assignments 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus penugasan target kualifikasi", "details": err.Error()})
				return
			}

			// 4. Reset target names and back numbers in event_participants
			_, err = tx.Exec(`
				UPDATE event_participants 
				SET target_name = NULL, back_number = NULL 
				WHERE event_id = ?
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset nomor bantalan", "details": err.Error()})
				return
			}

			// 5. Delete qualification session categories
			_, err = tx.Exec(`
				DELETE FROM qualification_session_categories 
				WHERE session_uuid IN (SELECT uuid FROM qualification_sessions WHERE event_uuid = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kategori sesi kualifikasi", "details": err.Error()})
				return
			}

			// 6. Delete target board qualification verification codes
			_, err = tx.Exec(`
				DELETE FROM target_board_qualification 
				WHERE session_uuid IN (SELECT uuid FROM qualification_sessions WHERE event_uuid = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kode verifikasi papan target kualifikasi", "details": err.Error()})
				return
			}

			// 7. Delete qualification sessions
			_, err = tx.Exec(`
				DELETE FROM qualification_sessions 
				WHERE event_uuid = ?
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus sesi kualifikasi", "details": err.Error()})
				return
			}

		case "elimination":
			// 1. Delete elimination matches
			_, err = tx.Exec(`
				DELETE FROM elimination_matches 
				WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE event_uuid = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus match eliminasi", "details": err.Error()})
				return
			}

			// 2. Delete elimination entries
			_, err = tx.Exec(`
				DELETE FROM elimination_entries 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus entri eliminasi", "details": err.Error()})
				return
			}

			// 3. Reset brackets to draft status
			tx.Exec(`
				UPDATE elimination_brackets 
				SET status = 'draft', generated_at = NULL 
				WHERE event_uuid = ?
			`, actualEventID)

		case "participants":
			// Wipe qualification first due to foreign keys
			tx.Exec(`
				DELETE FROM qualification_arrow_scores 
				WHERE end_score_uuid IN (
					SELECT uuid FROM qualification_end_scores 
					WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
				)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_end_scores 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_target_assignments 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
			`, actualEventID)

			// Wipe elimination due to foreign keys
			tx.Exec(`
				DELETE FROM elimination_matches 
				WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE event_uuid = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM elimination_entries 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
			`, actualEventID)
			tx.Exec(`
				UPDATE elimination_brackets 
				SET status = 'draft', generated_at = NULL 
				WHERE event_uuid = ?
			`, actualEventID)

			// Delete all event participants
			_, err = tx.Exec(`DELETE FROM event_participants WHERE event_id = ?`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data peserta", "details": err.Error()})
				return
			}

		case "all":
			// Wipe qualification
			tx.Exec(`
				DELETE FROM qualification_arrow_scores 
				WHERE end_score_uuid IN (
					SELECT uuid FROM qualification_end_scores 
					WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
				)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_end_scores 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_target_assignments 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
			`, actualEventID)

			// Wipe qualification sessions & associated links
			tx.Exec(`
				DELETE FROM qualification_session_categories 
				WHERE session_uuid IN (SELECT uuid FROM qualification_sessions WHERE event_uuid = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM target_board_qualification 
				WHERE session_uuid IN (SELECT uuid FROM qualification_sessions WHERE event_uuid = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_sessions 
				WHERE event_uuid = ?
			`, actualEventID)

			// Wipe elimination
			tx.Exec(`
				DELETE FROM elimination_matches 
				WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE event_uuid = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM elimination_entries 
				WHERE participant_uuid IN (SELECT uuid FROM event_participants WHERE event_id = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM elimination_brackets 
				WHERE event_uuid = ?
			`, actualEventID)

			// Wipe participants
			_, err = tx.Exec(`DELETE FROM event_participants WHERE event_id = ?`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal melakukan factory reset", "details": err.Error()})
				return
			}

		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Target reset tidak valid"})
			return
		}

		// Instead of deleting the used verification code, set its expires_at to 5 minutes from now
		// so the user can perform other resets within a 5-minute window without requesting a new code.
		newExpiry := time.Now().Add(5 * time.Minute)
		tx.Exec("UPDATE event_reset_codes SET expires_at = ? WHERE event_id = ? AND user_id = ?", newExpiry, eventID, userID.(string))

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan reset"})
			return
		}

		// Log activity
		if userID != nil {
			utils.LogActivity(db, userID.(string), actualEventID, "event_reset", "event", actualEventID, "Reset target data: "+req.Target, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Data event berhasil direset!"})
	}
}

// RequestResetCode sends a 6-digit verification code to the organizer's email for event reset operations
func RequestResetCode(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		if userID == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Resolve user email
		var userEmail string
		err := db.Get(&userEmail, `SELECT email FROM users WHERE uuid = ?`, userID.(string))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data user", "details": err.Error()})
			return
		}

		// Generate code
		otpCode := utils.GenerateOTP()

		// Save to event_reset_codes (delete any existing codes first)
		_, _ = db.Exec("DELETE FROM event_reset_codes WHERE event_id = ? AND user_id = ?", eventID, userID.(string))

		expiresAt := time.Now().Add(15 * time.Minute)
		_, err = db.Exec(`
			INSERT INTO event_reset_codes (uuid, event_id, user_id, code, expires_at)
			VALUES (?, ?, ?, ?, ?)
		`, uuid.New().String(), eventID, userID.(string), otpCode, expiresAt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat kode verifikasi", "details": err.Error()})
			return
		}

		// Send email
		subject := "Kode Verifikasi Reset Event - ArcheryHub"
		body := fmt.Sprintf(`
			<div style="font-family: sans-serif; padding: 20px; border: 1px solid #eee; border-radius: 10px; max-width: 500px;">
				<h2 style="color: #1e293b; margin-bottom: 20px;">Permintaan Reset Data Event</h2>
				<p style="color: #64748b; font-size: 14px; line-height: 1.5;">
					Anda menerima email ini karena ada permintaan untuk mereset data event di akun Anda.
					Gunakan kode verifikasi berikut untuk mengonfirmasi tindakan ini:
				</p>
				<div style="background-color: #f1f5f9; padding: 15px; text-align: center; border-radius: 8px; margin: 25px 0;">
					<span style="font-size: 32px; font-weight: bold; letter-spacing: 5px; color: #dc2626;">%s</span>
				</div>
				<p style="color: #ef4444; font-size: 12px; font-weight: bold;">
					Peringatan: Reset data bersifat permanen dan tidak dapat dibatalkan. Jangan bagikan kode ini kepada siapapun.
				</p>
				<p style="color: #94a3b8; font-size: 11px; margin-top: 30px;">
					Kode verifikasi ini akan kadaluarsa dalam 15 menit. Jika Anda tidak merasa mengajukan tindakan ini, silakan abaikan email ini.
				</p>
			</div>
		`, otpCode)

		err = utils.SendEmail(userEmail, subject, body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim email verifikasi", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Kode verifikasi telah dikirim ke email Anda"})
	}
}

