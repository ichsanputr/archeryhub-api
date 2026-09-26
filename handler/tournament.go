package handler

import (
	"Archeris-api/models"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"Archeris-api/utils"

	"encoding/csv"
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// GetEvents returns a list of tournaments
func GetEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		search := c.Query("search")
		limit, offset, page := utils.GetPaginationParams(c)
		organizerID := c.Query("organizer_id")

		// Check if user is archer to filter tournaments and include participant status
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

		// Public listing only shows external tournaments (internal tournaments are hidden from public list)
		if organizerID == "" {
			whereClause += ` AND (t.visibility = 'external' OR t.visibility IS NULL OR t.visibility = '')`
		}

		if search != "" {
			whereClause += ` AND (t.name LIKE ? OR t.code LIKE ? OR t.location LIKE ?)`
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		// Get total count
		var total int
		countQuery := `SELECT COUNT(*) FROM tournaments t ` + whereClause
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
				u.country as organizer_country,
				COUNT(DISTINCT tp2.archer_id) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count,
				COALESCE(MAX(tp.payment_status), '') as payment_status,
				COALESCE(MAX(tp.uuid), '') as participant_uuid
			FROM tournaments t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url, whatsapp_no as phone, country FROM organizers
				UNION ALL
				SELECT uuid as id, name as full_name, NULL as email, slug, logo_url as avatar_url, NULL as phone, 'Indonesia' as country FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN tournament_participants tp ON t.uuid = tp.tournament_id AND tp.archer_id = ?
			LEFT JOIN tournament_participants tp2 ON t.uuid = tp2.tournament_id
			LEFT JOIN tournament_categories te ON t.uuid = te.tournament_id
			` + whereClause + `
			GROUP BY t.uuid, u.full_name, u.email, u.slug, u.avatar_url, u.phone, u.country
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
				u.country as organizer_country,
				COUNT(DISTINCT tp.archer_id) as participant_count,
				COUNT(DISTINCT te.uuid) as event_count
			FROM tournaments t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url, whatsapp_no as phone, country FROM organizers
				UNION ALL
				SELECT uuid as id, name as full_name, NULL as email, slug, logo_url as avatar_url, NULL as phone, 'Indonesia' as country FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN tournament_participants tp ON t.uuid = tp.tournament_id
			LEFT JOIN tournament_categories te ON t.uuid = te.tournament_id
			` + whereClause + `
			GROUP BY t.uuid, u.full_name, u.email, u.slug, u.avatar_url, u.phone, u.country
			ORDER BY t.start_date DESC
			LIMIT ? OFFSET ?
			`
			args = append(args, limit, offset)
		}

		var tournaments []models.EventWithDetails
		err = db.Select(&tournaments, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event", "details": err.Error()})
			return
		}

		// Mask URLs
		for i := range tournaments {
			if tournaments[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].BannerURL)
				tournaments[i].BannerURL = &masked
			}
			if tournaments[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].LogoURL)
				tournaments[i].LogoURL = &masked
			}
			if tournaments[i].TechnicalGuidebookURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].TechnicalGuidebookURL)
				tournaments[i].TechnicalGuidebookURL = &masked
			}
			if tournaments[i].OrganizerAvatarURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].OrganizerAvatarURL)
				tournaments[i].OrganizerAvatarURL = &masked
			}
		}

		meta := utils.CalculatePagination(total, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{
			"data":   tournaments,
			"tournaments": tournaments,
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
				u.country as organizer_country,
				COALESCE(participant_stats.participant_count, 0) as participant_count,
				COALESCE(category_stats.event_count, 0) as event_count,
				COALESCE(target_stats.target_count, 0) as target_count,
				COALESCE(active_target_stats.active_target_count, 0) as active_target_count
			FROM tournaments t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, avatar_url, slug, whatsapp_no as phone, country FROM organizers
				UNION ALL
				SELECT uuid as id, name as full_name, NULL as email, logo_url as avatar_url, slug, NULL as phone, 'Indonesia' as country FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN (
				SELECT tournament_id, COUNT(DISTINCT archer_id) as participant_count
				FROM tournament_participants
				GROUP BY tournament_id
			) participant_stats ON t.uuid = participant_stats.tournament_id
			LEFT JOIN (
				SELECT tournament_id, COUNT(DISTINCT uuid) as event_count
				FROM tournament_categories
				GROUP BY tournament_id
			) category_stats ON t.uuid = category_stats.tournament_id
			LEFT JOIN (
				SELECT tournament_uuid, COUNT(*) as target_count
				FROM tournament_targets
				GROUP BY tournament_uuid
			) target_stats ON t.uuid = target_stats.tournament_uuid
			LEFT JOIN (
				SELECT tournament_id, COUNT(DISTINCT target_uuid) as active_target_count
				FROM (
					SELECT qs.tournament_uuid as tournament_id, qta.target_uuid
					FROM qualification_target_assignments qta
					JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
					UNION ALL
					SELECT eb.tournament_uuid as tournament_id, em.target_uuid
					FROM elimination_matches em
					JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
					WHERE em.target_uuid IS NOT NULL
				) combined
				GROUP BY tournament_id
			) active_target_stats ON t.uuid = active_target_stats.tournament_id
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

		// Get user ID and user type from context
		userID, exists := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		if userType == "archer" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Hanya organizer atau klub yang diizinkan untuk membuat event panahan.",
				"code":  "only_organizer_allowed",
			})
			return
		}

		// Start database transaction
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database", "details": err.Error()})
			return
		}
		defer tx.Rollback()

		// Generate code if not provided
		if req.Code == "" {
			var lastCode string
			_ = tx.Get(&lastCode, "SELECT code FROM tournaments WHERE code LIKE 'EVT-%' ORDER BY code DESC LIMIT 1 FOR UPDATE")
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
		for {
			var existsCount int
			err = tx.Get(&existsCount, `SELECT COUNT(1) FROM tournaments WHERE slug = ?`, finalSlug)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi keunikan slug", "details": err.Error()})
				return
			}
			if existsCount == 0 {
				break
			}
			finalSlug = fmt.Sprintf("%s-%d", originalSlug, suffix)
			suffix++
		}

		// Handle dates: if zero time, use nil (NULL in DB)
		var startDate, endDate, regDeadline, regStart interface{}
		if !req.StartDate.IsZero() {
			startDate = req.StartDate.Time
		}
		if !req.EndDate.IsZero() {
			endDate = req.EndDate.Time
		}
		if !req.RegistrationDeadline.IsZero() {
			regDeadline = req.RegistrationDeadline.Time
		}
		if req.RegistrationStart != nil && !req.RegistrationStart.IsZero() {
			regStart = req.RegistrationStart.Time
		} else {
			regStart = now
		}

		// Process Quota Type & Limits inside transaction
		quotaType := "free"
		if req.QuotaType != nil && *req.QuotaType != "" {
			quotaType = strings.ToLower(*req.QuotaType)
		}
		if quotaType != "free" && quotaType != "standard" && quotaType != "elite" {
			quotaType = "free"
		}

		var maxParticipants, maxCategories, maxScorekeepers, maxMediaMB *int
		if quotaType == "standard" {
			var qStandard int
			err = tx.Get(&qStandard, "SELECT quota_standard FROM organizers WHERE (uuid = ? OR user_id = ?) FOR UPDATE", userID, userID)
			if err != nil || qStandard <= 0 {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Quota Standard tidak mencukupi. Silakan beli kuota terlebih dahulu.",
					"code":  "quota_insufficient",
				})
				return
			}
			res, err := tx.Exec("UPDATE organizers SET quota_standard = quota_standard - 1 WHERE (uuid = ? OR user_id = ?) AND quota_standard > 0", userID, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses kuota", "details": err.Error()})
				return
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Quota Standard tidak mencukupi. Silakan beli kuota terlebih dahulu.",
					"code":  "quota_insufficient",
				})
				return
			}
			mp, mm := 200, 500
			maxParticipants, maxMediaMB = &mp, &mm
			// maxCategories & maxScorekeepers are nil (Unlimited)
		} else if quotaType == "elite" {
			var qElite int
			err = tx.Get(&qElite, "SELECT quota_elite FROM organizers WHERE (uuid = ? OR user_id = ?) FOR UPDATE", userID, userID)
			if err != nil || qElite <= 0 {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Quota Elite tidak mencukupi. Silakan beli kuota terlebih dahulu.",
					"code":  "quota_insufficient",
				})
				return
			}
			res, err := tx.Exec("UPDATE organizers SET quota_elite = quota_elite - 1 WHERE (uuid = ? OR user_id = ?) AND quota_elite > 0", userID, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses kuota", "details": err.Error()})
				return
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Quota Elite tidak mencukupi. Silakan beli kuota terlebih dahulu.",
					"code":  "quota_insufficient",
				})
				return
			}
			mm := 5120
			maxMediaMB = &mm
			// maxParticipants, maxCategories, maxScorekeepers are nil (Unlimited)
		} else {
			// Free tier (initial 20 welcome bonus)
			var qFree int
			err = tx.Get(&qFree, "SELECT COALESCE(quota_free, 20) FROM organizers WHERE (uuid = ? OR user_id = ?) FOR UPDATE", userID, userID)
			if err != nil || qFree <= 0 {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Kuota Free Tier Anda telah habis (0 tersisa). Silakan gunakan paket Standard atau Elite.",
					"code":  "quota_free_exhausted",
				})
				return
			}
			res, err := tx.Exec("UPDATE organizers SET quota_free = quota_free - 1 WHERE (uuid = ? OR user_id = ?) AND quota_free > 0", userID, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses kuota", "details": err.Error()})
				return
			}
			if rows, _ := res.RowsAffected(); rows == 0 {
				c.JSON(http.StatusPaymentRequired, gin.H{
					"error": "Kuota Free Tier Anda telah habis (0 tersisa). Silakan gunakan paket Standard atau Elite.",
					"code":  "quota_free_exhausted",
				})
				return
			}
			mp, mm := 50, 100
			maxParticipants, maxMediaMB = &mp, &mm
			// maxCategories & maxScorekeepers are nil (Unlimited)
		}

		visibility := "external"
		if req.Visibility != nil && *req.Visibility != "" {
			if *req.Visibility == "internal" {
				visibility = "internal"
			}
		}

		query := `
			INSERT INTO tournaments (
				uuid, code, name, short_name, slug, venue, gmaps_link, location, city, 
				start_date, end_date, registration_start, registration_deadline,
				description, banner_url, logo_url, location_type, num_distances, num_sessions, 
				entry_fee, status, organizer_id, created_at, updated_at,
				total_prize, technical_guidebook_url, page_settings, faq,
				quota_type, quota_max_participants, quota_max_categories, quota_max_scorekeepers, quota_max_media_mb,
				visibility
			) VALUES (
				?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
				?, ?, ?, ?, ?, ?, ?
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

		gmapsLinkClean := utils.NormalizeGmapsEmbed(req.GmapLink)

		_, err = tx.Exec(query,
			eventUUID, req.Code, req.Name, req.ShortName, finalSlug, req.Venue, gmapsLinkClean,
			req.Location, req.City,
			startDate, endDate, regStart, regDeadline,
			req.Description, utils.ExtractFilename(models.FromPtr(req.BannerURL)), utils.ExtractFilename(models.FromPtr(req.LogoURL)), locationType, req.NumDistances, req.NumSessions,
			req.EntryFee,
			status, userID, now, now,
			req.TotalPrize, utils.ExtractFilename(models.FromPtr(req.TechnicalGuidebookURL)), req.PageSettings,
			models.ToJSON(req.FAQ),
			quotaType, maxParticipants, maxCategories, maxScorekeepers, maxMediaMB,
			visibility,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat event", "details": err.Error()})
			return
		}

		// Save categories if provided
		if len(req.Divisions) > 0 && len(req.Categories) > 0 {
			for _, divUUID := range req.Divisions {
				for _, catUUID := range req.Categories {
					catEventID := uuid.New().String()
					_, err = tx.Exec(`
						INSERT INTO tournament_categories (
							uuid, tournament_id, division_uuid, category_uuid, 
							max_participants
						) VALUES (?, ?, ?, ?, NULL)
					`, catEventID, eventUUID, divUUID, catUUID)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan kategori event", "details": err.Error()})
						return
					}
				}
			}
		}

		// Save event images if provided
		if len(req.Images) > 0 {
			for i, img := range req.Images {
				imageID := uuid.New().String()
				isPrimary := img.IsPrimary || i == 0 // First image is primary by default
				_, err = tx.Exec(`
					INSERT INTO tournament_images (uuid, tournament_id, url, caption, alt_text, display_order, is_primary)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, imageID, eventUUID, utils.ExtractFilename(img.URL), img.Caption, img.AltText, i, isPrimary)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan gambar event", "details": err.Error()})
					return
				}
			}
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan transaksi event", "details": err.Error()})
			return
		}

		// Link uploaded media files to tournament
		if req.BannerURL != nil && *req.BannerURL != "" {
			cleanName := utils.ExtractFilename(*req.BannerURL)
			_, _ = db.Exec("UPDATE media SET tournament_id = ? WHERE (url = ? OR url LIKE ?) AND tournament_id IS NULL", eventUUID, cleanName, "%"+cleanName)
		}
		if req.LogoURL != nil && *req.LogoURL != "" {
			cleanName := utils.ExtractFilename(*req.LogoURL)
			_, _ = db.Exec("UPDATE media SET tournament_id = ? WHERE (url = ? OR url LIKE ?) AND tournament_id IS NULL", eventUUID, cleanName, "%"+cleanName)
		}
		if req.TechnicalGuidebookURL != nil && *req.TechnicalGuidebookURL != "" {
			cleanName := utils.ExtractFilename(*req.TechnicalGuidebookURL)
			_, _ = db.Exec("UPDATE media SET tournament_id = ? WHERE (url = ? OR url LIKE ?) AND tournament_id IS NULL", eventUUID, cleanName, "%"+cleanName)
		}

		// Log activity (after successful commit)
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
		err := db.Get(&actualID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}
		id = actualID

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
		if req.GmapLink != nil {
			cleanGmap := utils.NormalizeGmapsEmbed(req.GmapLink)
			query += ", gmaps_link = ?"
			args = append(args, cleanGmap)
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
		if req.Visibility != nil {
			vis := "external"
			if *req.Visibility == "internal" {
				vis = "internal"
			}
			query += ", visibility = ?"
			args = append(args, vis)
		}

		query += " WHERE uuid = ?"
		args = append(args, id)

		_, err = db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data event", "details": err.Error()})
			return
		}

		// Normalize and persist relational fields from page_settings
		if req.PageSettings != nil && *req.PageSettings != "" {
			var ps struct {
				FeeMode               string                 `json:"fee_mode"`
				FeePerType            map[string]float64     `json:"fee_per_type"`
				FeePerCategory        map[string]float64     `json:"fee_per_category"`
				Currency              string                 `json:"currency"`
				CountryCode           string                 `json:"country_code"`
				EnableManualPayment   *bool                  `json:"enable_manual_payment"`
				ResultsType           string                 `json:"results_type"`
				RegistrationStart     *string                `json:"registration_start"`
				Sections              map[string]bool        `json:"sections"`
				PaymentMethods        []string               `json:"payment_methods"`
				LocationAccessibility []string               `json:"location_accessibility"`
				Prizes                map[string]interface{} `json:"prizes"`
				TechnicalGuidebooks   []struct {
					Title string `json:"title"`
					URL   string `json:"url"`
				} `json:"technical_guidebooks"`
				Results []struct {
					Title string `json:"title"`
					URL   string `json:"url"`
				} `json:"results"`
			}
			if errJson := json.Unmarshal([]byte(*req.PageSettings), &ps); errJson == nil {
				// 1. Update tournaments relational columns
				updateTournQuery := "UPDATE tournaments SET updated_at = NOW()"
				var updateTournArgs []interface{}

				if ps.FeeMode != "" {
					updateTournQuery += ", fee_mode = ?"
					updateTournArgs = append(updateTournArgs, ps.FeeMode)
				}
				if ps.FeePerType != nil {
					if indVal, ok := ps.FeePerType["individual"]; ok {
						updateTournQuery += ", fee_individual = ?"
						updateTournArgs = append(updateTournArgs, indVal)
					}
					if teamVal, ok := ps.FeePerType["team"]; ok {
						updateTournQuery += ", fee_team = ?"
						updateTournArgs = append(updateTournArgs, teamVal)
					}
					if mixVal, ok := ps.FeePerType["mixed_team"]; ok {
						updateTournQuery += ", fee_mixed_team = ?"
						updateTournArgs = append(updateTournArgs, mixVal)
					}
				}
				if ps.Currency != "" {
					updateTournQuery += ", currency = ?"
					updateTournArgs = append(updateTournArgs, ps.Currency)
				}
				if ps.CountryCode != "" {
					updateTournQuery += ", country_code = ?"
					updateTournArgs = append(updateTournArgs, ps.CountryCode)
				}
				if ps.EnableManualPayment != nil {
					updateTournQuery += ", enable_manual_payment = ?"
					updateTournArgs = append(updateTournArgs, *ps.EnableManualPayment)
				}
				if ps.ResultsType != "" {
					updateTournQuery += ", results_type = ?"
					updateTournArgs = append(updateTournArgs, ps.ResultsType)
				}
				if ps.RegistrationStart != nil && *ps.RegistrationStart != "" {
					parsedStart, errParse := time.Parse(time.RFC3339, *ps.RegistrationStart)
					if errParse == nil {
						updateTournQuery += ", registration_start = ?"
						updateTournArgs = append(updateTournArgs, parsedStart)
					}
				}
				updateTournQuery += " WHERE uuid = ?"
				updateTournArgs = append(updateTournArgs, id)
				_, _ = db.Exec(updateTournQuery, updateTournArgs...)

				// 2. Update category fees if per_category
				if ps.FeePerCategory != nil {
					for catUUID, feeVal := range ps.FeePerCategory {
						_, _ = db.Exec("UPDATE tournament_categories SET fee = ? WHERE uuid = ? AND tournament_id = ?", feeVal, catUUID, id)
					}
				}

				// 3. Upsert tournament_page_sections
				if ps.Sections != nil {
					showAbout := ps.Sections["about"]
					showDivisions := ps.Sections["divisions"]
					showFees := ps.Sections["fees"]
					showPaymentMethods := ps.Sections["payment_methods"]
					showPrizes := ps.Sections["prizes"]
					showSchedule := ps.Sections["schedule"]
					showLocation := ps.Sections["location"]
					showFAQ := ps.Sections["faq"]

					_, _ = db.Exec(`
						INSERT INTO tournament_page_sections (
							tournament_id, show_about, show_divisions, show_fees, show_payment_methods,
							show_prizes, show_schedule, show_location, show_faq
						) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
						ON DUPLICATE KEY UPDATE
							show_about = VALUES(show_about),
							show_divisions = VALUES(show_divisions),
							show_fees = VALUES(show_fees),
							show_payment_methods = VALUES(show_payment_methods),
							show_prizes = VALUES(show_prizes),
							show_schedule = VALUES(show_schedule),
							show_location = VALUES(show_location),
							show_faq = VALUES(show_faq)
					`, id, showAbout, showDivisions, showFees, showPaymentMethods, showPrizes, showSchedule, showLocation, showFAQ)
				}

				// 4. Update tournament_selected_payment_methods
				if ps.PaymentMethods != nil {
					_, _ = db.Exec("DELETE FROM tournament_selected_payment_methods WHERE tournament_id = ?", id)
					for _, pmID := range ps.PaymentMethods {
						if strings.TrimSpace(pmID) != "" {
							_, _ = db.Exec("INSERT INTO tournament_selected_payment_methods (uuid, tournament_id, payment_method_id) VALUES (UUID(), ?, ?)", id, strings.TrimSpace(pmID))
						}
					}
				}

				// 5. Update tournament_facilities
				if ps.LocationAccessibility != nil {
					_, _ = db.Exec("DELETE FROM tournament_facilities WHERE tournament_id = ?", id)
					for _, fac := range ps.LocationAccessibility {
						if strings.TrimSpace(fac) != "" {
							_, _ = db.Exec("INSERT INTO tournament_facilities (uuid, tournament_id, facility_name) VALUES (UUID(), ?, ?)", id, strings.TrimSpace(fac))
						}
					}
				}

				// 6. Update tournament_documents
				_, _ = db.Exec("DELETE FROM tournament_documents WHERE tournament_id = ?", id)
				for _, gb := range ps.TechnicalGuidebooks {
					if gb.URL != "" {
						title := gb.Title
						if title == "" {
							title = "Petunjuk Teknis"
						}
						_, _ = db.Exec("INSERT INTO tournament_documents (uuid, tournament_id, doc_type, title, file_url) VALUES (UUID(), ?, 'guidebook', ?, ?)", id, title, gb.URL)
					}
				}
				for _, res := range ps.Results {
					if res.URL != "" {
						title := res.Title
						if title == "" {
							title = "Hasil Pertandingan"
						}
						_, _ = db.Exec("INSERT INTO tournament_documents (uuid, tournament_id, doc_type, title, file_url) VALUES (UUID(), ?, 'result', ?, ?)", id, title, res.URL)
					}
				}
			}
		}

		// Link uploaded media files to tournament
		if req.BannerURL != nil && *req.BannerURL != "" {
			cleanName := utils.ExtractFilename(*req.BannerURL)
			_, _ = db.Exec("UPDATE media SET tournament_id = ? WHERE (url = ? OR url LIKE ?) AND tournament_id IS NULL", id, cleanName, "%"+cleanName)
		}
		if req.LogoURL != nil && *req.LogoURL != "" {
			cleanName := utils.ExtractFilename(*req.LogoURL)
			_, _ = db.Exec("UPDATE media SET tournament_id = ? WHERE (url = ? OR url LIKE ?) AND tournament_id IS NULL", id, cleanName, "%"+cleanName)
		}
		if req.TechnicalGuidebookURL != nil && *req.TechnicalGuidebookURL != "" {
			cleanName := utils.ExtractFilename(*req.TechnicalGuidebookURL)
			_, _ = db.Exec("UPDATE media SET tournament_id = ? WHERE (url = ? OR url LIKE ?) AND tournament_id IS NULL", id, cleanName, "%"+cleanName)
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
		err := db.Get(&actualID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
			return
		}
		defer tx.Rollback()

		// Refund quota if event was published
		var eventInfo struct {
			Status      string  `db:"status"`
			QuotaType   *string `db:"quota_type"`
			OrganizerID string  `db:"organizer_id"`
		}
		if err := tx.Get(&eventInfo, "SELECT status, quota_type, organizer_id FROM tournaments WHERE uuid = ? FOR UPDATE", actualID); err == nil {
			if eventInfo.Status == "published" && eventInfo.QuotaType != nil {
				if *eventInfo.QuotaType == "standard" {
					_, _ = tx.Exec("UPDATE organizers SET quota_standard = quota_standard + 1 WHERE uuid = ?", eventInfo.OrganizerID)
				} else if *eventInfo.QuotaType == "elite" {
					_, _ = tx.Exec("UPDATE organizers SET quota_elite = quota_elite + 1 WHERE uuid = ?", eventInfo.OrganizerID)
				}
			}
		}

		result, err := tx.Exec("DELETE FROM tournaments WHERE uuid = ?", actualID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data event: " + err.Error()})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyelesaikan transaksi"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "Event_deleted", "Event", id, "Deleted Event", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Data event berhasil dihapus"})
	}
}

// These functions are now in division_category.go to avoid duplication

// GetEventEvents returns tournaments for a specific event
func GetEventEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		limit, offset, page := utils.GetPaginationParams(c)
		bowTypeFilter := c.Query("bow_type")
		eventTypeFilter := c.Query("event_type")

		// First, resolve slug to UUID if needed
		var actualEventID string
		err := db.Get(&actualEventID, `
			SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?
		`, eventID, eventID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type EventEvent struct {
			ID                 string  `db:"id" json:"id"`
			EventID            string  `db:"event_id" json:"event_id"`
			TournamentID       *string `db:"tournament_id" json:"tournament_id"`
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

		whereClause := "WHERE te.tournament_id = ?"
		args := []interface{}{actualEventID}

		if bowTypeFilter != "" && bowTypeFilter != "all" {
			whereClause += " AND d.code = ?"
			args = append(args, bowTypeFilter)
		}

		if eventTypeFilter != "" && eventTypeFilter != "all" {
			whereClause += " AND te.tournament_type_uuid = ?"
			args = append(args, eventTypeFilter)
		}

		// Get total count
		var total int
		err = db.Get(&total, `
			SELECT COUNT(*) 
			FROM tournament_categories te
			JOIN ref_bow_types d ON te.division_uuid = d.uuid
			`+whereClause, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung kategori event", "details": err.Error()})
			return
		}

		var tournaments []EventEvent
		query := `
			SELECT 
				te.uuid as id, te.tournament_id as event_id, te.tournament_id,
				te.max_participants, te.status, te.created_at, te.category_name_custom,
				CASE 
					WHEN et.code = 'mixed_team' THEN 2 
					WHEN et.code = 'team' THEN 3 
					ELSE 1 
				END as team_size,
				d.name as division_name, d.uuid as division_id,
				COALESCE(te.category_name_custom, c.name) as category_name, c.uuid as category_id,
				COALESCE(et.name, '') as event_type_name, COALESCE(te.tournament_type_uuid, '') as event_type_id,
				COALESCE(gd.name, '') as gender_division_name, COALESCE(te.gender_division_uuid, '') as gender_division_id,
				COALESCE(p.p_count, 0) as participant_count,
				COALESCE(t.t_count, 0) as team_count
			FROM tournament_categories te
			JOIN ref_bow_types d ON te.division_uuid = d.uuid
			JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_tournament_types et ON te.tournament_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			LEFT JOIN (
				SELECT category_id, COUNT(*) as p_count 
				FROM tournament_participants 
				GROUP BY category_id
			) p ON te.uuid = p.category_id
			LEFT JOIN (
				SELECT event_id as category_id, COUNT(*) as t_count 
				FROM teams 
				GROUP BY tournament_id
			) t ON te.uuid = t.category_id
			` + whereClause + `
			ORDER BY d.name, c.name, et.name, gd.name
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)
		err = db.Select(&tournaments, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil kategori event", "details": err.Error()})
			return
		}

		meta := utils.CalculatePagination(total, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{
			"data":        tournaments,
			"tournaments": tournaments,
			"categories":  tournaments,
			"events":       tournaments,
			"total":       total,
			"meta":        meta,
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
		err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
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
					FROM tournament_categories t
					JOIN ref_tournament_types tt ON t.tournament_type_uuid = tt.uuid
					JOIN tournament_categories i ON i.tournament_id = t.tournament_id
						AND i.division_uuid = t.division_uuid
						AND i.category_uuid = t.category_uuid
						AND (
							tt.code = 'mixed_team' 
							OR (tt.code = 'team' AND i.gender_division_uuid = t.gender_division_uuid)
						)
					JOIN ref_tournament_types it ON i.tournament_type_uuid = it.uuid
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
			whereClause := "WHERE tp.tournament_id = ?"
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
				pLower := strings.ToLower(paymentStatus)
				if pLower == "terbayar" || pLower == "paid" || pLower == "lunas" || pLower == "verified" {
					whereClause += " AND tp.payment_status IN ('paid', 'lunas')"
				} else if pLower == "pending" || pLower == "menunggu" || pLower == "menunggu_acc" {
					whereClause += " AND tp.payment_status IN ('pending', 'menunggu_acc', 'menunggu acc', 'unpaid', 'awaiting_verification')"
				} else if pLower == "unpaid" || pLower == "belum_bayar" {
					whereClause += " AND tp.payment_status IN ('unpaid', 'pending', 'menunggu_acc')"
				} else {
					whereClause += " AND tp.payment_status = ?"
					args = append(args, paymentStatus)
				}
			}

			if reregStatus := c.Query("reregistration_status"); reregStatus != "" && reregStatus != "Semua" {
				if reregStatus == "reregistered" || reregStatus == "sudah" {
					whereClause += " AND tp.last_reregistration_at IS NOT NULL"
				} else if reregStatus == "not_reregistered" || reregStatus == "belum" || reregStatus == "pending" {
					whereClause += " AND tp.last_reregistration_at IS NULL"
				}
			}

			if divFilter := c.Query("division"); divFilter != "" && divFilter != "Semua" {
				divList := strings.Split(divFilter, ",")
				if len(divList) > 0 {
					divHolders := strings.Repeat(",?", len(divList))[1:]
					whereClause += " AND (d.name IN (" + divHolders + ") OR d.uuid IN (" + divHolders + "))"
					for _, dVal := range divList {
						args = append(args, strings.TrimSpace(dVal))
					}
					for _, dVal := range divList {
						args = append(args, strings.TrimSpace(dVal))
					}
				}
			}

			if ageFilter := c.Query("age_group"); ageFilter != "" && ageFilter != "Semua" {
				ageList := strings.Split(ageFilter, ",")
				if len(ageList) > 0 {
					ageHolders := strings.Repeat(",?", len(ageList))[1:]
					whereClause += " AND (c.name IN (" + ageHolders + ") OR c.uuid IN (" + ageHolders + "))"
					for _, aVal := range ageList {
						args = append(args, strings.TrimSpace(aVal))
					}
					for _, aVal := range ageList {
						args = append(args, strings.TrimSpace(aVal))
					}
				}
			}

			if genderFilter := c.Query("gender"); genderFilter != "" && genderFilter != "Semua" {
				if strings.EqualFold(genderFilter, "male") || strings.EqualFold(genderFilter, "men") || strings.EqualFold(genderFilter, "putra") {
					whereClause += " AND (gd.code = 'men' OR gd.name LIKE '%Putra%' OR gd.name LIKE '%Men%')"
				} else if strings.EqualFold(genderFilter, "female") || strings.EqualFold(genderFilter, "women") || strings.EqualFold(genderFilter, "putri") {
					whereClause += " AND (gd.code = 'women' OR gd.name LIKE '%Putri%' OR gd.name LIKE '%Women%')"
				} else {
					whereClause += " AND (gd.uuid = ? OR gd.name = ?)"
					args = append(args, genderFilter, genderFilter)
				}
			}

			if clubFilter := c.Query("club_id"); clubFilter != "" && clubFilter != "Semua" {
				whereClause += " AND (cl.uuid = ? OR cl.name LIKE ?)"
				args = append(args, clubFilter, "%"+clubFilter+"%")
			}

			// Count unique archers
			var total int
			countQuery := `
				SELECT COUNT(DISTINCT tp.archer_id)
				FROM tournament_participants tp
				JOIN archers a ON tp.archer_id = a.uuid
				LEFT JOIN clubs cl ON a.club_id = cl.uuid
				LEFT JOIN tournament_categories te ON tp.category_id = te.uuid
				LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
				LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
				LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			` + whereClause
			err = db.Get(&total, countQuery, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung jumlah atlet", "details": err.Error()})
				return
			}

			type FlatParticipantRow struct {
				ArcherID             string     `db:"archer_id" json:"archer_id"`
				AthleteCode          *string    `db:"athlete_code" json:"athlete_code"`
				FullName             string     `db:"full_name" json:"full_name"`
				Email                string     `db:"email" json:"email"`
				AvatarURL            *string    `db:"avatar_url" json:"avatar_url"`
				ClubName             *string    `db:"club_name" json:"club_name"`
				City                 *string    `db:"city" json:"city"`
				ParticipantID        string     `db:"participant_id" json:"participant_id"`
				CategoryID           string     `db:"category_id" json:"category_id"`
				DivisionName         string     `db:"division_name" json:"division_name"`
				CategoryName         string     `db:"category_name" json:"category_name"`
				EventTypeName        string     `db:"event_type_name" json:"event_type_name"`
				GenderDivisionName   string     `db:"gender_division_name" json:"gender_division_name"`
				PaymentStatus        *string    `db:"payment_status" json:"payment_status"`
				RegistrationSource   *string    `db:"registration_source" json:"registration_source"`
				QrRaw                *string    `db:"qr_raw" json:"qr_raw"`
				RegistrationDate     *time.Time `db:"registration_date" json:"registration_date"`
				LastReregistrationAt *time.Time `db:"last_reregistration_at" json:"last_reregistration_at"`
			}

			var rows []FlatParticipantRow
			query := `
				SELECT 
					a.uuid as archer_id,
					a.id as athlete_code,
					a.full_name,
					COALESCE(a.email, '') as email,
					a.avatar_url,
					COALESCE(cl.name, '') as club_name,
					'' as city,
					tp.uuid as participant_id,
					tp.category_id,
					COALESCE(d.name, '') as division_name,
					COALESCE(te.category_name_custom, c.name, '') as category_name,
					COALESCE(et.name, '') as event_type_name,
					COALESCE(gd.name, '') as gender_division_name,
					COALESCE(tp.payment_status, 'pending') as payment_status,
					COALESCE(tp.registration_source, 'self_register') as registration_source,
					tp.qr_raw,
					tp.registration_date,
					tp.last_reregistration_at
				FROM tournament_participants tp
				JOIN archers a ON tp.archer_id = a.uuid
				LEFT JOIN clubs cl ON a.club_id = cl.uuid
				LEFT JOIN tournament_categories te ON tp.category_id = te.uuid
				LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
				LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
				LEFT JOIN ref_tournament_types et ON te.tournament_type_uuid = et.uuid
				LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
				` + whereClause + `
				ORDER BY a.full_name ASC
			`
			err = db.Select(&rows, query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data peserta berkelompok", "details": err.Error()})
				return
			}

			// Group in Go
			type GroupedParticipant struct {
				ArcherID             string                   `json:"archer_id"`
				AthleteCode          *string                  `json:"athlete_code"`
				FullName             string                   `json:"full_name"`
				Email                string                   `json:"email"`
				AvatarURL            *string                  `json:"avatar_url"`
				ClubName             *string                  `json:"club_name"`
				City                 *string                  `json:"city"`
				PaymentStatus        *string                  `json:"payment_status"`
				LastReregistrationAt *time.Time               `json:"last_reregistration_at"`
				CategoryList         []map[string]interface{} `json:"categories"`
			}

			groupedMap := make(map[string]*GroupedParticipant)
			var orderedKeys []string

			for _, r := range rows {
				g, exists := groupedMap[r.ArcherID]
				if !exists {
					var avatar *string
					if r.AvatarURL != nil {
						masked := utils.MaskMediaURL(*r.AvatarURL)
						avatar = &masked
					}
					g = &GroupedParticipant{
						ArcherID:             r.ArcherID,
						AthleteCode:          r.AthleteCode,
						FullName:             r.FullName,
						Email:                r.Email,
						AvatarURL:            avatar,
						ClubName:             r.ClubName,
						City:                 r.City,
						PaymentStatus:        r.PaymentStatus,
						LastReregistrationAt: r.LastReregistrationAt,
						CategoryList:         []map[string]interface{}{},
					}
					groupedMap[r.ArcherID] = g
					orderedKeys = append(orderedKeys, r.ArcherID)
				} else if r.LastReregistrationAt != nil && g.LastReregistrationAt == nil {
					g.LastReregistrationAt = r.LastReregistrationAt
				}

				catItem := map[string]interface{}{
					"participant_id":         r.ParticipantID,
					"category_id":            r.CategoryID,
					"division_name":          r.DivisionName,
					"category_name":          r.CategoryName,
					"event_type_name":        r.EventTypeName,
					"gender_division_name":   r.GenderDivisionName,
					"payment_status":         r.PaymentStatus,
					"registration_source":    r.RegistrationSource,
					"qr_raw":                 r.QrRaw,
					"registration_date":      r.RegistrationDate,
					"last_reregistration_at": r.LastReregistrationAt,
				}

				alreadyInList := false
				for _, existingCat := range g.CategoryList {
					if existingCat["category_id"] == r.CategoryID {
						alreadyInList = true
						break
					}
				}
				if !alreadyInList {
					g.CategoryList = append(g.CategoryList, catItem)
				}
			}

			sortBy := strings.ToLower(c.DefaultQuery("sort_by", "name"))
			sortDir := strings.ToLower(c.DefaultQuery("order", c.DefaultQuery("sort_dir", "asc")))

			// Sort orderedKeys
			sort.SliceStable(orderedKeys, func(i, j int) bool {
				a := groupedMap[orderedKeys[i]]
				b := groupedMap[orderedKeys[j]]
				if a == nil || b == nil {
					return false
				}
				var cmp int
				switch sortBy {
				case "club":
					clubA := ""
					if a.ClubName != nil {
						clubA = *a.ClubName
					}
					clubB := ""
					if b.ClubName != nil {
						clubB = *b.ClubName
					}
					cmp = strings.Compare(strings.ToLower(clubA), strings.ToLower(clubB))
				case "status":
					statA := ""
					if a.PaymentStatus != nil {
						statA = *a.PaymentStatus
					}
					statB := ""
					if b.PaymentStatus != nil {
						statB = *b.PaymentStatus
					}
					cmp = strings.Compare(strings.ToLower(statA), strings.ToLower(statB))
				case "reregistration":
					var tA, tB int64
					if a.LastReregistrationAt != nil {
						tA = a.LastReregistrationAt.Unix()
					}
					if b.LastReregistrationAt != nil {
						tB = b.LastReregistrationAt.Unix()
					}
					if tA < tB {
						cmp = -1
					} else if tA > tB {
						cmp = 1
					} else {
						cmp = 0
					}
				default: // "name"
					cmp = strings.Compare(strings.ToLower(a.FullName), strings.ToLower(b.FullName))
				}
				if cmp == 0 {
					cmp = strings.Compare(strings.ToLower(a.FullName), strings.ToLower(b.FullName))
				}
				if sortDir == "desc" {
					return cmp > 0
				}
				return cmp < 0
			})

			// Apply pagination slice
			total = len(orderedKeys)
			start := offset
			if start > total {
				start = total
			}
			end := start + limit
			if end > total {
				end = total
			}

			var paginatedResult []*GroupedParticipant
			if start < end {
				for _, key := range orderedKeys[start:end] {
					paginatedResult = append(paginatedResult, groupedMap[key])
				}
			} else {
				paginatedResult = []*GroupedParticipant{}
			}

			statusWhere := "WHERE tp.tournament_id = ?"
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
			verifiedQuery := "SELECT COUNT(DISTINCT tp.archer_id) FROM tournament_participants tp " + statusWhere + " AND tp.payment_status IN ('paid', 'lunas')"
			pendingQuery := "SELECT COUNT(DISTINCT tp.archer_id) FROM tournament_participants tp " + statusWhere + " AND tp.payment_status IN ('pending', 'menunggu_acc', 'menunggu acc', 'unpaid', 'awaiting_verification')"
			_ = db.Get(&verifiedCount, verifiedQuery, statusArgs...)
			_ = db.Get(&pendingCount, pendingQuery, statusArgs...)

			lastPage := (total + limit - 1) / limit
			if lastPage < 1 {
				lastPage = 1
			}

			c.JSON(http.StatusOK, gin.H{
				"participants":   paginatedResult,
				"total":          total,
				"verified_count": verifiedCount,
				"pending_count":  pendingCount,
				"meta": utils.PaginationMeta{
					TotalCount:  total,
					Limit:       limit,
					Offset:      offset,
					CurrentPage: page,
					LastPage:    lastPage,
				},
			})
			return
		}

		// Standard logic (existing)
		whereClause := "WHERE tp.tournament_id = ?"
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
			whereClause += " AND (a.full_name LIKE ? OR cl.name LIKE ? OR a.email LIKE ?)"
			args = append(args, searchTerm, searchTerm, searchTerm)
			countArgs = append(countArgs, searchTerm, searchTerm, searchTerm)
		}

		if paymentStatus := c.Query("payment_status"); paymentStatus != "" && paymentStatus != "Semua" {
			pLower := strings.ToLower(paymentStatus)
			if pLower == "terbayar" || pLower == "paid" || pLower == "lunas" || pLower == "verified" {
				whereClause += " AND tp.payment_status IN ('paid', 'lunas')"
			} else if pLower == "pending" || pLower == "menunggu" || pLower == "menunggu_acc" {
				whereClause += " AND tp.payment_status IN ('pending', 'menunggu_acc', 'menunggu acc', 'unpaid', 'awaiting_verification')"
			} else if pLower == "unpaid" || pLower == "belum_bayar" {
				whereClause += " AND tp.payment_status IN ('unpaid', 'pending', 'menunggu_acc')"
			} else {
				whereClause += " AND tp.payment_status = ?"
				args = append(args, paymentStatus)
				countArgs = append(countArgs, paymentStatus)
			}
		}

		// Get total count with filters
		countQuery := "SELECT COUNT(*) FROM tournament_participants tp LEFT JOIN archers a ON tp.archer_id = a.uuid LEFT JOIN clubs cl ON a.club_id = cl.uuid LEFT JOIN tournament_categories te ON tp.category_id = te.uuid LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid " + whereClause
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
			TournamentID         *string `db:"tournament_id" json:"tournament_id"`
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
				tp.uuid as id, tp.archer_id, tp.tournament_id as event_id, tp.tournament_id, tp.category_id, tp.target_name, tp.qr_raw,
				tp.payment_status, tp.registration_date, tp.last_reregistration_at,
				COALESCE(tp.registration_source, 'self_register') as registration_source,
				a.id as athlete_code,
				a.username as username,
				a.full_name as full_name,
				COALESCE(a.email, '') as email,
				'' as city,
				a.club_id as club_id,
				a.avatar_url as avatar_url,
				COALESCE(cl.name, '') as club_name,
				COALESCE(d.name, '') as division_name, COALESCE(te.category_name_custom, c.name, '') as category_name,
				COALESCE(et.name, '') as event_type_name, COALESCE(gd.name, '') as gender_division_name,
				COALESCE(scores.total_score, 0) as total_score,
				COALESCE(scores.total_x, 0) as total_x
			FROM tournament_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_tournament_types et ON te.tournament_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			LEFT JOIN (
				SELECT participant_uuid, SUM(total_score_end) as total_score, SUM(x_count_end) as total_x
				FROM qualification_end_scores
				GROUP BY participant_uuid
			) scores ON tp.uuid = scores.participant_uuid
			` + whereClause + `
			GROUP BY tp.uuid, a.uuid, cl.uuid, te.uuid, d.uuid, c.uuid, et.uuid, gd.uuid, a.id, a.username, a.full_name, a.email, a.club_id, a.avatar_url, cl.name, d.name, c.name, te.category_name_custom, et.name, gd.name, scores.total_score, scores.total_x, tp.payment_status
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
		db.Get(&verifiedCount, "SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = ? AND payment_status IN ('paid', 'lunas')", actualEventID)
		db.Get(&pendingCount, "SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = ? AND payment_status IN ('pending', 'menunggu_acc', 'menunggu acc')", actualEventID)

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

func enrichPaymentTransaction(db *sqlx.DB, tx *models.PaymentTransaction) {
	if tx == nil {
		return
	}
	var regUser struct {
		FullName string `db:"full_name"`
		Email    string `db:"email"`
	}
	if errU := db.Get(&regUser, `SELECT full_name, COALESCE(email, '') as email FROM archers WHERE uuid = ? OR id = ? LIMIT 1`, tx.UserID, tx.UserID); errU == nil {
		tx.RegisteredByName = &regUser.FullName
		tx.RegisteredByEmail = &regUser.Email
		if tx.PayerName == nil || *tx.PayerName == "" {
			tx.PayerName = &regUser.FullName
		}
		if tx.PayerEmail == nil || *tx.PayerEmail == "" {
			tx.PayerEmail = &regUser.Email
		}
	}
	var delCount int
	_ = db.Get(&delCount, `SELECT COUNT(DISTINCT archer_id) FROM tournament_participants WHERE payment_id = ? OR uuid = ?`, tx.UUID, tx.RegistrationID)
	tx.DelegationCount = delCount
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
		err := db.Get(&event, `SELECT uuid, status, organizer_id FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
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

		type ParticipantRow struct {
			ID                          string     `db:"id" json:"id"`
			AthleteCode                 *string    `db:"athlete_code" json:"athlete_code"`
			ArcherID                    *string    `db:"archer_id" json:"archer_id"`
			FullName                    string     `db:"full_name" json:"full_name"`
			Username                    *string    `db:"username" json:"username"`
			Email                       string     `db:"email" json:"email"`
			Phone                       *string    `db:"phone" json:"phone"`
			Gender                      *string    `db:"gender" json:"gender"`
			BirthDate                   *time.Time `db:"birth_date" json:"birth_date"`
			DateOfBirth                 *time.Time `db:"date_of_birth" json:"date_of_birth"`
			BowType                     *string    `db:"bow_type" json:"bow_type"`
			HandDominance               *string    `db:"hand_dominance" json:"hand_dominance"`
			Address                     *string    `db:"address" json:"address"`
			EmergencyContactName        *string    `db:"emergency_contact_name" json:"emergency_contact_name"`
			City                        *string    `db:"city" json:"city"`
			ClubID                      *string    `db:"club_id" json:"club_id"`
			ClubName                    *string    `db:"club_name" json:"club_name"`
			EventID                     string     `db:"event_id" json:"event_id"`
			TournamentID                *string    `db:"tournament_id" json:"tournament_id"`
			CategoryID                  string     `db:"category_id" json:"category_id"`
			DivisionName                string     `db:"division_name" json:"division_name"`
			CategoryName                string     `db:"category_name" json:"category_name"`
			EventTypeName               *string    `db:"event_type_name" json:"event_type_name"`
			GenderDivisionName          *string    `db:"gender_division_name" json:"gender_division_name"`
			TargetName                  *string    `db:"target_name" json:"target_name"`
			QRRaw                       *string    `db:"qr_raw" json:"qr_raw"`
			PaymentStatus               string     `db:"payment_status" json:"payment_status"`
			AvatarURL                   *string    `db:"avatar_url" json:"avatar_url"`
			PaymentAmount               float64    `db:"payment_amount" json:"payment_amount"`
			RegistrationDate            string     `db:"registration_date" json:"registration_date"`
			LastReregistrationAt        *time.Time `db:"last_reregistration_at" json:"last_reregistration_at"`
			Reregistered                bool       `db:"reregistered" json:"reregistered"`
			IsVerified                  bool       `db:"is_verified" json:"is_verified"`
			RegistrationSource          string     `db:"registration_source" json:"registration_source"`
			QualificationAssignmentUUID *string    `db:"qualification_assignment_uuid" json:"qualification_assignment_uuid"`
			HasScores                   bool       `db:"has_scores" json:"has_scores"`
			InElimination               bool       `db:"in_elimination" json:"in_elimination"`
		}

		type ParticipantResponse struct {
			ParticipantRow
			Categories       []map[string]interface{}    `json:"categories"`
			PaymentProofURLs []string                    `json:"payment_proof_urls"`
			Transaction      *models.PaymentTransaction  `json:"transaction"`
			Transactions     []models.PaymentTransaction `json:"transactions"`
		}

		var rows []ParticipantRow
		err = db.Select(&rows, `
			SELECT 
				tp.uuid as id, tp.archer_id, tp.tournament_id as event_id, tp.tournament_id, tp.category_id, tp.target_name, tp.qr_raw,
				tp.payment_amount, tp.payment_status,
				COALESCE(tp.registration_source, 'self_register') as registration_source,
				tp.registration_date,
				tp.last_reregistration_at,
				(tp.last_reregistration_at IS NOT NULL) as reregistered,
				a.id as athlete_code,
				a.username as username,
				a.full_name as full_name,
				COALESCE(a.email, '') as email,
				COALESCE(a.phone, '') as phone,
				a.gender as gender,
				COALESCE(a.birth_date, a.date_of_birth) as birth_date,
				a.date_of_birth as date_of_birth,
				a.bow_type as bow_type,
				a.hand_dominance as hand_dominance,
				a.address as address,
				a.emergency_contact_name as emergency_contact_name,
				COALESCE(a.city, '') as city,
				a.club_id as club_id,
				a.avatar_url as avatar_url,
				COALESCE(cl.name, '') as club_name,
				COALESCE(d.name, '') as division_name, COALESCE(c.name, '') as category_name,
				COALESCE(et.name, '') as event_type_name, COALESCE(gd.name, '') as gender_division_name,
				COALESCE(a.is_verified, 0) as is_verified,
				(SELECT uuid FROM qualification_target_assignments WHERE participant_uuid = tp.uuid LIMIT 1) as qualification_assignment_uuid,
				(
					EXISTS(SELECT 1 FROM qualification_end_scores qes WHERE qes.participant_uuid = tp.uuid)
					OR (tp.qual_score IS NOT NULL AND tp.qual_score > 0)
				) as has_scores,
				EXISTS(
					SELECT 1 FROM elimination_matches 
					WHERE entry_a_uuid IN (SELECT uuid FROM elimination_entries WHERE participant_uuid = tp.uuid)
					   OR entry_b_uuid IN (SELECT uuid FROM elimination_entries WHERE participant_uuid = tp.uuid)
				) as in_elimination
			FROM tournament_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_tournament_types et ON te.tournament_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			WHERE tp.tournament_id = ? AND a.uuid = (
				SELECT archer_id FROM tournament_participants tp_sub
				LEFT JOIN archers a_sub ON tp_sub.archer_id = a_sub.uuid
				WHERE tp_sub.tournament_id = ? AND (
					tp_sub.uuid = ? OR 
					tp_sub.archer_id = ? OR
					a_sub.username = ? OR 
					a_sub.id = ? OR
					LOWER(REPLACE(a_sub.full_name, ' ', '-')) = LOWER(?)
				)
				LIMIT 1
			)
			ORDER BY 
				(tp.payment_status = 'paid') DESC,
				(tp.payment_status IN ('pending', 'menunggu_acc', 'unpaid')) DESC,
				tp.qr_raw IS NULL ASC,
				tp.created_at DESC
		`, actualEventID, actualEventID, participantID, participantID, participantID, participantID, participantID)

		if err != nil || len(rows) == 0 {
			fmt.Printf("[DEBUG] Participant not found in DB for Event: %s, ID: %s. Error: %v\n", actualEventID, participantID, err)
			c.JSON(http.StatusNotFound, gin.H{
				"error":          "Peserta tidak ditemukan",
				"details":        fmt.Sprintf("%v", err),
				"participant_id": participantID,
				"event_id":       actualEventID,
			})
			return
		}

		first := rows[0]
		if first.AvatarURL != nil {
			masked := utils.MaskMediaURL(*first.AvatarURL)
			first.AvatarURL = &masked
		}

		for _, r := range rows {
			if r.LastReregistrationAt != nil {
				first.LastReregistrationAt = r.LastReregistrationAt
				first.Reregistered = true
				break
			}
		}

		resp := ParticipantResponse{
			ParticipantRow:   first,
			Categories:       []map[string]interface{}{},
			PaymentProofURLs: []string{},
			Transactions:     []models.PaymentTransaction{},
		}

		seenCategories := make(map[string]bool)
		for _, r := range rows {
			if seenCategories[r.CategoryID] {
				continue
			}
			seenCategories[r.CategoryID] = true
			isLocked := r.HasScores || r.InElimination
			catItem := map[string]interface{}{
				"participant_id":       r.ID,
				"category_id":          r.CategoryID,
				"division_name":        r.DivisionName,
				"category_name":        r.CategoryName,
				"event_type_name":      r.EventTypeName,
				"gender_division_name": r.GenderDivisionName,
				"payment_status":       r.PaymentStatus,
				"payment_amount":       r.PaymentAmount,
				"fee":                  r.PaymentAmount,
				"registration_date":    r.RegistrationDate,
				"has_scores":           r.HasScores,
				"in_elimination":       r.InElimination,
				"is_locked":            isLocked,
			}
			resp.Categories = append(resp.Categories, catItem)
		}

		// Fetch all payment transactions and audit ledger records for this participant (including delegation invoices)
		var transactions []models.PaymentTransaction
		errTxList := db.Select(&transactions, `
			SELECT * FROM payment_transactions 
			WHERE (tournament_id = ? AND (
					user_id = ? 
					OR registration_id IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ? AND archer_id = ?)
					OR uuid IN (SELECT payment_id FROM tournament_participants WHERE tournament_id = ? AND archer_id = ? AND payment_id IS NOT NULL)
				  ))
			   OR registration_id = ?
			   OR uuid IN (SELECT payment_id FROM tournament_participants WHERE uuid = ? AND payment_id IS NOT NULL)
			ORDER BY created_at DESC
		`, actualEventID, *first.ArcherID, actualEventID, *first.ArcherID, actualEventID, *first.ArcherID, first.ID, first.ID)

		if errTxList == nil && len(transactions) > 0 {
			for i := range transactions {
				enrichPaymentTransaction(db, &transactions[i])
			}
			resp.Transactions = transactions
			resp.Transaction = &transactions[0]
			for _, txItem := range transactions {
				if txItem.ProofURL != nil && *txItem.ProofURL != "" {
					resp.PaymentProofURLs = append(resp.PaymentProofURLs, *txItem.ProofURL)
				}
			}
		} else {
			var singleTx models.PaymentTransaction
			if errTx := db.Get(&singleTx, `SELECT * FROM payment_transactions WHERE registration_id = ? ORDER BY created_at DESC LIMIT 1`, first.ID); errTx == nil {
				enrichPaymentTransaction(db, &singleTx)
				resp.Transaction = &singleTx
				resp.Transactions = append(resp.Transactions, singleTx)
				if singleTx.ProofURL != nil && *singleTx.ProofURL != "" {
					resp.PaymentProofURLs = append(resp.PaymentProofURLs, *singleTx.ProofURL)
				}
			}
		}

		c.JSON(http.StatusOK, resp)
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
		if err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		userEmailVal, _ := c.Get("email")
		userEmail := fmt.Sprintf("%v", userEmailVal)

		// Get archer UUID for this user (if any)
		var archerID string
		_ = db.Get(&archerID, `SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?) LIMIT 1`, userID, userID, userEmail)
		if archerID == "" {
			archerID = fmt.Sprintf("%v", userID)
		}

		type MyRegistrationCategory struct {
			ID                 string  `json:"id"`
			CategoryUUID       string  `json:"category_id"`
			DivisionName       string  `json:"division_name"`
			CategoryName       string  `json:"category_name"`
			EventTypeName      *string `json:"event_type_name"`
			GenderDivisionName *string `json:"gender_division_name"`
			TargetName         *string `json:"target_name"`
			BackNumber         *string `json:"back_number"`
			PaymentStatus      string  `json:"payment_status"`
			PaymentAmount      float64 `json:"payment_amount"`
			RegistrationDate   string  `json:"registration_date"`
			LastReregisteredAt *string `json:"last_reregistration_at"`
		}

		type TargetAssignmentInfo struct {
			ID             string  `json:"id" db:"id"`
			Stage          string  `json:"stage" db:"stage"`                   // "qualification" | "elimination" | "final"
			StageLabel     string  `json:"stage_label" db:"stage_label"`       // "Babak Kualifikasi", "Babak Eliminasi 1/8", etc.
			TargetNumber   string  `json:"target_number" db:"target_number"`   // "1", "6", "12"
			TargetPosition string  `json:"target_position" db:"target_position"` // "A", "B", "C", "D"
			TargetName     string  `json:"target_name" db:"target_name"`       // "1A", "6B", "12C"
			SessionName    string  `json:"session_name" db:"session_name"`     // "Kualifikasi Sesi 1 (Pagi)"
			SessionDate    *string `json:"session_date" db:"session_date"`
			StartTime      *string `json:"start_time" db:"start_time"`
			EndTime        *string `json:"end_time" db:"end_time"`
			CategoryID     string  `json:"category_id" db:"category_id"`
			CategoryName   string  `json:"category_name" db:"category_name"`
			DivisionName   string  `json:"division_name" db:"division_name"`
			Distance       *string `json:"distance" db:"distance"`
			TargetFace     *string `json:"target_face" db:"target_face"`
			ParticipantID  string  `json:"participant_id" db:"participant_id"`
		}

		type MyRegistrationResponse struct {
			ArcherID            string                     `json:"archer_id"`
			AthleteCode         *string                    `json:"athlete_code"`
			FullName            string                     `json:"full_name"`
			Email               string                     `json:"email"`
			ClubName            *string                    `json:"club_name"`
			City                *string                    `json:"city"`
			AvatarURL           *string                    `json:"avatar_url"`
			QRRaw               *string                    `json:"qr_raw"`
			PaymentStatus       string                     `json:"payment_status"` // Combined or latest
			PaymentAmount       float64                    `json:"payment_amount"` // Total
			PaymentProofURLs    []string                   `json:"payment_proof_urls"`
			Categories          []MyRegistrationCategory   `json:"categories"`
			Teams               any                        `json:"teams"`
			Targets             []TargetAssignmentInfo     `json:"targets"`
			Transaction         *models.PaymentTransaction `json:"transaction"`
			PaymentMethodManual *string                    `json:"payment_method_manual"`
		}

		type Row struct {
			ID                  string  `db:"id"`
			TargetName          *string `db:"target_name"`
			BackNumber          *string `db:"back_number"`
			PaymentStatus       string  `db:"payment_status"`
			PaymentAmount       float64 `db:"payment_amount"`
			RegistrationDate    string  `db:"registration_date"`
			LastReregisteredAt  *string `db:"last_reregistration_at"`
			QRRaw               *string `db:"qr_raw"`
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
				tp.uuid as id, tp.target_name, tp.back_number, tp.category_id,
				tp.payment_status, tp.payment_amount, tp.registration_date,
				tp.last_reregistration_at,
				COALESCE(tp.qr_raw, CONCAT('ARCHERIS-CHECKIN:', tp.uuid)) as qr_raw,
				a.id as athlete_code, a.full_name, COALESCE(a.email, '') as email,
				'' as city, a.avatar_url, COALESCE(cl.name, '') as club_name,
				COALESCE(d.name, '') as division_name, COALESCE(te.category_name_custom, c.name, '') as category_name,
				COALESCE(et.name, '') as event_type_name, COALESCE(gd.name, '') as gender_division_name
			FROM tournament_participants tp
			LEFT JOIN archers a ON (tp.archer_id = a.uuid OR tp.archer_id = a.id)
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_tournament_types et ON te.tournament_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			WHERE (tp.tournament_id = ? OR tp.tournament_id IN (SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?)) AND (
				tp.archer_id = ? 
				OR tp.archer_id = ?
				OR tp.archer_id IN (SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?))
				OR tp.archer_id IN (SELECT id FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?))
			)
			ORDER BY tp.created_at ASC
		`, actualEventID, actualEventID, actualEventID, archerID, userID, userID, userID, userEmail, userID, userID, userEmail)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil registrasi", "details": err.Error()})
			return
		}
		if len(rows) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registrasi tidak ditemukan untuk event ini"})
			return
		}

		firstRow := rows[0]
		avatarURL := firstRow.AvatarURL
		if avatarURL != nil && *avatarURL != "" {
			masked := utils.MaskMediaURL(*avatarURL)
			avatarURL = &masked
		}

		resp := MyRegistrationResponse{
			ArcherID:    archerID,
			AthleteCode: firstRow.AthleteCode,
			FullName:    firstRow.FullName,
			Email:       firstRow.Email,
			ClubName:    firstRow.ClubName,
			City:        firstRow.City,
			AvatarURL:   avatarURL,
			QRRaw:       firstRow.QRRaw,
			Categories:  []MyRegistrationCategory{},
			Targets:     []TargetAssignmentInfo{},
		}

		// Combined status logic
		resp.PaymentStatus = firstRow.PaymentStatus

		participantIDs := make([]string, 0, len(rows))
		for _, row := range rows {
			participantIDs = append(participantIDs, row.ID)
			resp.Categories = append(resp.Categories, MyRegistrationCategory{
				ID:                 row.ID,
				CategoryUUID:       row.CategoryUUID,
				DivisionName:       row.DivisionName,
				CategoryName:       row.CategoryName,
				EventTypeName:      row.EventTypeName,
				GenderDivisionName: row.GenderDivisionName,
				TargetName:         row.TargetName,
				BackNumber:         row.BackNumber,
				PaymentStatus:      row.PaymentStatus,
				PaymentAmount:      row.PaymentAmount,
				RegistrationDate:   row.RegistrationDate,
				LastReregisteredAt: row.LastReregisteredAt,
			})
			resp.PaymentAmount += row.PaymentAmount
		}

		seenTargets := make(map[string]bool)

		// Fetch qualification target assignments
		if len(participantIDs) > 0 {
			queryQTA, argsQTA, errIn := sqlx.In(`
				SELECT 
					qta.uuid as id,
					'qualification' as stage,
					'Babak Kualifikasi' as stage_label,
					COALESCE(NULLIF(CONVERT(tt.board_number, CHAR), '0'), REGEXP_SUBSTR(tt.target_name, '^[0-9]+'), '') as target_number,
					COALESCE(REGEXP_SUBSTR(tt.target_name, '[A-Za-z]+$'), '') as target_position,
					COALESCE(tt.target_name, tp.target_name, '-') as target_name,
					COALESCE(qs.name, 'Sesi Kualifikasi') as session_name,
					DATE_FORMAT(qs.session_date, '%Y-%m-%d') as session_date,
					DATE_FORMAT(qs.start_time, '%H:%i') as start_time,
					DATE_FORMAT(qs.end_time, '%H:%i') as end_time,
					tp.category_id,
					COALESCE(te.category_name_custom, CONCAT(COALESCE(d.name, ''), ' ', COALESCE(c.name, ''))) as category_name,
					COALESCE(d.name, '') as division_name,
					NULL as distance,
					NULL as target_face,
					tp.uuid as participant_id
				FROM qualification_target_assignments qta
				JOIN tournament_participants tp ON qta.participant_uuid = tp.uuid
				LEFT JOIN tournament_targets tt ON qta.target_uuid = tt.uuid
				LEFT JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
				LEFT JOIN tournament_categories te ON tp.category_id = te.uuid
				LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
				LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
				WHERE qta.participant_uuid IN (?)
				ORDER BY qs.session_date ASC, qs.start_time ASC, tt.board_number ASC
			`, participantIDs)
			if errIn == nil {
				queryQTA = db.Rebind(queryQTA)
				var qtaList []TargetAssignmentInfo
				if errQTA := db.Select(&qtaList, queryQTA, argsQTA...); errQTA == nil {
					for _, item := range qtaList {
						key := item.ParticipantID + "-" + item.Stage + "-" + item.TargetName + "-" + item.SessionName
						if !seenTargets[key] {
							seenTargets[key] = true
							resp.Targets = append(resp.Targets, item)
						}
					}
				}
			}

			// Query elimination matches target assignments
			queryElim, argsElim, errElimIn := sqlx.In(`
				SELECT 
					em.uuid as id,
					'elimination' as stage,
					CASE 
						WHEN em.round_no = 1 THEN 'Babak Final (Medali)'
						WHEN em.round_no = 2 THEN 'Babak Semifinal'
						WHEN em.round_no = 4 THEN 'Babak Perempat Final'
						WHEN em.round_no = 8 THEN 'Babak 1/8 Eliminasi'
						WHEN em.round_no = 16 THEN 'Babak 1/16 Eliminasi'
						ELSE CONCAT('Babak Eliminasi R', em.round_no)
					END as stage_label,
					COALESCE(NULLIF(CONVERT(tt.board_number, CHAR), '0'), REGEXP_SUBSTR(tt.target_name, '^[0-9]+'), '') as target_number,
					COALESCE(REGEXP_SUBSTR(tt.target_name, '[A-Za-z]+$'), '') as target_position,
					COALESCE(tt.target_name, '-') as target_name,
					CONCAT('Match Eliminasi #', em.match_no) as session_name,
					DATE_FORMAT(em.scheduled_at, '%Y-%m-%d') as session_date,
					DATE_FORMAT(em.scheduled_at, '%H:%i') as start_time,
					NULL as end_time,
					eb.category_uuid as category_id,
					COALESCE(te.category_name_custom, CONCAT(COALESCE(d.name, ''), ' ', COALESCE(c.name, ''))) as category_name,
					COALESCE(d.name, '') as division_name,
					NULL as distance,
					NULL as target_face,
					ee.participant_uuid as participant_id
				FROM elimination_matches em
				JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
				JOIN elimination_entries ee ON (em.entry_a_uuid = ee.uuid OR em.entry_b_uuid = ee.uuid)
				LEFT JOIN tournament_targets tt ON em.target_uuid = tt.uuid
				LEFT JOIN tournament_categories te ON eb.category_uuid = te.uuid
				LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
				LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
				WHERE ee.participant_uuid IN (?)
				ORDER BY em.scheduled_at ASC, em.round_no DESC, em.match_no ASC
			`, participantIDs)
			if errElimIn == nil {
				queryElim = db.Rebind(queryElim)
				var elimList []TargetAssignmentInfo
				if errElim := db.Select(&elimList, queryElim, argsElim...); errElim == nil {
					for _, item := range elimList {
						key := item.ParticipantID + "-" + item.Stage + "-" + item.TargetName + "-" + item.SessionName
						if !seenTargets[key] {
							seenTargets[key] = true
							resp.Targets = append(resp.Targets, item)
						}
					}
				}
			}
		}

		// Fallback for categories that have tp.target_name but no qta/elim record
		for _, row := range rows {
			if row.TargetName != nil && *row.TargetName != "" {
				key := row.ID + "-qualification-" + *row.TargetName + "-Sesi Utama"
				if !seenTargets[key] {
					seenTargets[key] = true
					resp.Targets = append(resp.Targets, TargetAssignmentInfo{
						ID:             row.ID,
						Stage:          "qualification",
						StageLabel:     "Babak Kualifikasi",
						TargetNumber:   *row.TargetName,
						TargetPosition: "",
						TargetName:     *row.TargetName,
						SessionName:    "Sesi Kualifikasi Utama",
						CategoryID:     row.CategoryUUID,
						CategoryName:   row.CategoryName,
						DivisionName:   row.DivisionName,
						ParticipantID:  row.ID,
					})
				}
			}
		}

		// Fetch team registrations for this archer in this event
		type MyRegistrationTeamMember struct {
			ParticipantID string  `json:"participant_id" db:"participant_id"`
			ArcherID      string  `json:"archer_id" db:"archer_id"`
			FullName      string  `json:"full_name" db:"full_name"`
			Gender        string  `json:"gender" db:"gender"`
			ClubName      string  `json:"club_name" db:"club_name"`
			MemberOrder   int     `json:"member_order" db:"member_order"`
			IsCaptain     bool    `json:"is_captain" db:"is_captain"`
		}

		type MyRegistrationTeam struct {
			TeamID       string                     `json:"team_id" db:"team_id"`
			TeamName     string                     `json:"team_name" db:"team_name"`
			CategoryID   string                     `json:"category_id" db:"category_id"`
			CategoryName string                     `json:"category_name" db:"category_name"`
			Status       string                     `json:"status" db:"status"`
			IsCaptain    bool                       `json:"is_captain" db:"is_captain"`
			Members      []MyRegistrationTeamMember `json:"members"`
		}

		var myTeams []MyRegistrationTeam
		errTeams := db.Select(&myTeams, `
			SELECT 
				t.uuid as team_id,
				t.team_name,
				COALESCE(t.category_id, t.event_id) as category_id,
				COALESCE(te.category_name_custom, CONCAT(COALESCE(d.name, ''), ' ', COALESCE(c.name, ''))) as category_name,
				t.status,
				CASE WHEN my_tm.member_order = 1 THEN 1 ELSE 0 END as is_captain
			FROM teams t
			JOIN team_members my_tm ON t.uuid = my_tm.team_id
			JOIN tournament_participants my_tp ON my_tm.participant_id = my_tp.uuid
			LEFT JOIN tournament_categories te ON (t.category_id = te.uuid OR t.event_id = te.uuid)
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			WHERE t.tournament_id = ? AND my_tp.archer_id = ?
			GROUP BY t.uuid, t.team_name, t.category_id, t.event_id, te.category_name_custom, d.name, c.name, t.status, my_tm.member_order
			ORDER BY t.created_at ASC
		`, actualEventID, archerID)

		if errTeams == nil {
			for idx := range myTeams {
				var members []MyRegistrationTeamMember
				_ = db.Select(&members, `
					SELECT 
						tm.participant_id,
						COALESCE(tp.archer_id, '') as archer_id,
						COALESCE(a.full_name, 'Pemanah') as full_name,
						COALESCE(a.gender, 'male') as gender,
						COALESCE(cl.name, 'Independen') as club_name,
						tm.member_order,
						CASE WHEN tm.member_order = 1 THEN 1 ELSE 0 END as is_captain
					FROM team_members tm
					JOIN tournament_participants tp ON tm.participant_id = tp.uuid
					LEFT JOIN archers a ON tp.archer_id = a.uuid
					LEFT JOIN clubs cl ON a.club_id = cl.uuid
					WHERE tm.team_id = ?
					ORDER BY tm.member_order ASC
				`, myTeams[idx].TeamID)
				if members == nil {
					members = []MyRegistrationTeamMember{}
				}
				myTeams[idx].Members = members
			}
		}
		if myTeams == nil {
			myTeams = []MyRegistrationTeam{}
		}
		resp.Teams = myTeams

		// Fetch payment transaction for this participant / archer / event
		var transaction models.PaymentTransaction
		errTx := db.Get(&transaction, `
			SELECT * FROM payment_transactions 
			WHERE (
				uuid IN (SELECT payment_id FROM tournament_participants WHERE tournament_id = ? AND archer_id = ? AND payment_id IS NOT NULL)
				OR registration_id IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ? AND archer_id = ?)
				OR (tournament_id = ? AND user_id = ?)
				OR registration_id = ?
			)
			ORDER BY created_at DESC LIMIT 1
		`, actualEventID, archerID, actualEventID, archerID, actualEventID, userID, firstRow.ID)
		if errTx == nil {
			enrichPaymentTransaction(db, &transaction)
			resp.Transaction = &transaction
			if transaction.ProofURL != nil && *transaction.ProofURL != "" {
				resp.PaymentProofURLs = []string{*transaction.ProofURL}
			}
		}

		c.JSON(http.StatusOK, resp)
	}
}

// GetEventSchedule returns schedule items for an event
func GetEventSchedule(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var tournament struct {
			UUID string `db:"uuid"`
			Slug string `db:"slug"`
		}
		err := db.Get(&tournament, `SELECT uuid, slug FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		type ScheduleItemResult struct {
			UUID            string         `json:"uuid" db:"uuid"`
			TournamentID    string         `json:"tournament_id" db:"tournament_id"`
			ItemType        string         `json:"item_type" db:"item_type"`
			StartTime       string         `json:"start_time" db:"start_time"`
			EndTime         string         `json:"end_time" db:"end_time"`
			DurationMinutes int            `json:"duration_minutes" db:"duration_minutes"`
			DelayMinutes    int            `json:"delay_minutes" db:"delay_minutes"`
			Title           string         `json:"title" db:"title"`
			Subtitle        sql.NullString `json:"subtitle" db:"subtitle"`
			Description     sql.NullString `json:"description" db:"description"`
			Location        sql.NullString `json:"location" db:"location"`
			SessionCode     sql.NullString `json:"session_code" db:"session_code"`
			CategoryUUIDs   sql.NullString `json:"category_uuids" db:"category_uuids"`
			BracketUUID     sql.NullString `json:"bracket_uuid" db:"bracket_uuid"`
			ElimRound       sql.NullInt64  `json:"elim_round" db:"elim_round"`
			TargetStart     sql.NullInt64  `json:"target_start" db:"target_start"`
			TargetEnd       sql.NullInt64  `json:"target_end" db:"target_end"`
			SortOrder       int            `json:"sort_order" db:"sort_order"`
			DayOrder        int            `json:"day_order" db:"day_number"`
			DayNumber       int            `json:"day_number" db:"day_number"`
			ScheduleDate    string         `json:"schedule_date" db:"schedule_date"`
		}

		var items []ScheduleItemResult
		err = db.Select(&items, `
			SELECT 
				uuid, tournament_id, item_type,
				CAST(start_time AS CHAR) AS start_time,
				CAST(end_time AS CHAR) AS end_time,
				duration_minutes, delay_minutes, title, subtitle, description, location,
				session_code, category_uuids, bracket_uuid, elim_round,
				target_start, target_end, sort_order, day_number,
				COALESCE(CAST(schedule_date AS CHAR), '') AS schedule_date
			FROM tournament_schedule_items
			WHERE tournament_id = ?
			ORDER BY day_number ASC, start_time ASC, sort_order ASC
		`, tournament.UUID)

		if err == nil && len(items) > 0 {
			c.JSON(http.StatusOK, gin.H{
				"schedules": items,
				"count":     len(items),
			})
			return
		}

		// Fallback to legacy tournament_schedules table
		var schedules []models.EventSchedule
		err = db.Select(&schedules, `
			SELECT es.* 
			FROM tournament_schedules es
			WHERE es.tournament_id = ?
			ORDER BY 
				COALESCE(es.day_order, 0),
				COALESCE(es.sort_order, 0),
				es.start_time
		`, tournament.UUID)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"schedules": []models.EventSchedule{},
				"count":     0,
			})
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
		err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database", "details": err.Error()})
			return
		}
		defer tx.Rollback()

		// Delete existing schedules
		_, err = tx.Exec("DELETE FROM tournament_schedules WHERE tournament_id = ?", actualEventID)
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

			_, err = tx.Exec(`
				INSERT INTO tournament_schedules (uuid, tournament_id, title, description, start_time, end_time, day_order, sort_order, location)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, scheduleID, actualEventID, s.Title, s.Description, formattedStartTime, formattedEndTime, dayOrder, sortOrder, s.Location)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan jadwal", "details": err.Error()})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan jadwal event", "details": err.Error()})
			return
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
			FROM tournament_category_refs ecr
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
			INSERT INTO tournament_category_refs (uuid, name, bow_type_id, age_group_id, status)
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
		if err := db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM tournament_category_refs WHERE uuid = ?)`, id); err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
			return
		}

		query := "UPDATE tournament_category_refs SET updated_at = NOW()"
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
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		// Check quota_type from request, default 'standard'
		quotaType := c.DefaultQuery("quota_type", "")
		if quotaType == "" {
			// Try JSON body
			var body struct { QuotaType string `json:"quota_type"` }
			if err := c.ShouldBindJSON(&body); err == nil && body.QuotaType != "" {
				quotaType = body.QuotaType
			}
		}
		if quotaType != "standard" && quotaType != "elite" {
			quotaType = "standard"
		}

		// Get organizer uuid
		var orgUUID string
		db.Get(&orgUUID, "SELECT uuid FROM organizers WHERE user_id = ?", userID)

		// Get plan limits
		var plan struct {
			QuotaStandard int `db:"quota_standard"`
			QuotaElite    int `db:"quota_elite"`
		}
		db.Get(&plan, "SELECT quota_standard, quota_elite FROM organizers WHERE uuid = ?", orgUUID)

		// Check quota availability
		if quotaType == "standard" && plan.QuotaStandard <= 0 {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Quota Standard EO tidak tersedia. Silakan beli quota terlebih dahulu.",
				"code": "quota_insufficient",
				"quota_type": "standard",
			})
			return
		}
		if quotaType == "elite" && plan.QuotaElite <= 0 {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error": "Quota Elite EO tidak tersedia. Silakan beli quota terlebih dahulu.",
				"code": "quota_insufficient",
				"quota_type": "elite",
			})
			return
		}

		// Get plan limits from subscription_plans
		var limits struct {
			MaxParticipants  *int `db:"max_participants"`
			MaxCategories    *int `db:"max_categories"`
			MaxScorekeepers  *int `db:"max_scorekeepers"`
			MaxMediaMB       *int `db:"max_media_mb"`
		}
		db.Get(&limits, "SELECT max_participants, max_categories, max_scorekeepers, max_media_mb FROM subscription_plans WHERE type='quota' AND target_type='organization' AND quota_type=? ORDER BY id DESC LIMIT 1", quotaType)

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database: " + err.Error()})
			return
		}
		defer tx.Rollback()

		// Deduct quota
		if quotaType == "standard" {
			_, err = tx.Exec("UPDATE organizers SET quota_standard = quota_standard - 1 WHERE uuid = ? AND quota_standard > 0", orgUUID)
		} else {
			_, err = tx.Exec("UPDATE organizers SET quota_elite = quota_elite - 1 WHERE uuid = ? AND quota_elite > 0", orgUUID)
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memotong kuota turnamen"})
			return
		}

		// Set event quota metadata
		_, err = tx.Exec("UPDATE tournaments SET quota_type = ?, quota_max_participants = ?, quota_max_categories = ?, quota_max_scorekeepers = ?, quota_max_media_mb = ? WHERE uuid = ?",
			quotaType, limits.MaxParticipants, limits.MaxCategories, limits.MaxScorekeepers, limits.MaxMediaMB, eventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui metadata kuota turnamen"})
			return
		}

		_, err = tx.Exec("UPDATE tournaments SET status = 'published' WHERE uuid = ?", eventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mempublikasikan event"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan status publikasi"})
			return
		}

		// Log activity
		utils.LogActivity(db, userID.(string), eventID, "event_published", "event", eventID, "Published event", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Event berhasil dipublikasikan"})
	}
}

// RegisterParticipant registers a participant or team for an event
func RegisterParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			AthleteID          string                               `json:"athlete_id"`
			EventCategoryID    string                               `json:"event_category_id"`
			EventCategoryIDs   []string                             `json:"event_category_ids"`
			PaymentAmount      float64                              `json:"payment_amount"`
			PaymentStatus      string                               `json:"payment_status"`
			RegistrationSource string                               `json:"registration_source"`
			RegistrationMode   string                               `json:"registration_mode"` // "individual" | "captain_team" | "club_delegation"
			TeamRegistration   *models.TeamRegistrationInput        `json:"team_registration"`
			TeamRegistrations  []models.TeamRegistrationInput       `json:"team_registrations"`
			DelegationAthletes []models.DelegationAthleteInput      `json:"delegation_athletes"`
			DelegationTeams    []models.DelegationTeamBookingInput  `json:"delegation_teams"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Data registrasi tidak valid", "details": err.Error()})
			return
		}
		req.AthleteID = strings.TrimSpace(req.AthleteID)

		// Resolve event slug to UUID and get organizer ID
		var event struct {
			UUID                 string     `db:"uuid"`
			OrganizerID          string     `db:"organizer_id"`
			EntryFee             float64    `db:"entry_fee"`
			FeeMode              string     `db:"fee_mode"`
			FeeIndividual        float64    `db:"fee_individual"`
			FeeTeam              float64    `db:"fee_team"`
			FeeMixedTeam         float64    `db:"fee_mixed_team"`
			Status               string     `db:"status"`
			RegistrationDeadline *time.Time `db:"registration_deadline"`
			StartDate            *time.Time `db:"start_date"`
			QuotaMaxParticipants *int       `db:"quota_max_participants"`
		}
		err := db.Get(&event, `SELECT uuid, organizer_id, entry_fee, fee_mode, fee_individual, fee_team, fee_mixed_team, status, registration_deadline, start_date, quota_max_participants FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		actualEventID := event.UUID
		organizerID := event.OrganizerID

		// Verification sub status organizer
		var orgStatus string
		db.Get(&orgStatus, `
			SELECT COALESCE(s, 'active') FROM (
				SELECT subscription_status as s FROM organizers WHERE uuid = ?
				UNION ALL
				SELECT 'active' as s FROM clubs WHERE uuid = ?
			) combined LIMIT 1`, organizerID, organizerID)

		if orgStatus != "" && orgStatus != "active" {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":   "Pendaftaran ditutup sementara",
				"code":    "organizer_subscription_expired",
				"message": "Pendaftaran peserta untuk event ini ditutup sementara oleh sistem karena masa berlaku layanan penyelenggara telah berakhir.",
			})
			return
		}

		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("role")
		orgID, _ := c.Get("org_id")

		isPrivileged := userRole == "admin" || (userRole == "organizer" && orgID != nil && fmt.Sprintf("%v", orgID) == event.OrganizerID)

		// Check registration deadline
		if !isPrivileged && event.RegistrationDeadline != nil && time.Now().After(*event.RegistrationDeadline) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Pendaftaran untuk turnamen ini telah ditutup",
				"code":  "registration_closed",
			})
			return
		}

		// Check overall event quota
		if !isPrivileged && event.QuotaMaxParticipants != nil && *event.QuotaMaxParticipants > 0 {
			var currentTotal int
			_ = db.Get(&currentTotal, "SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = ? AND payment_status != 'cancelled'", event.UUID)
			if currentTotal >= *event.QuotaMaxParticipants {
				c.JSON(http.StatusConflict, gin.H{
					"error": "Kuota keseluruhan turnamen ini telah penuh",
					"code":  "event_quota_exceeded",
				})
				return
			}
		}

		// Determine payment status
		paymentStatus := "unpaid"
		if req.PaymentStatus != "" && isPrivileged {
			paymentStatus = req.PaymentStatus
		}

		registrationSource := "self_register"
		if isPrivileged {
			if req.RegistrationSource != "" {
				registrationSource = req.RegistrationSource
			} else {
				registrationSource = "organizer_added"
			}
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// Category metadata and eligibility helpers
		type CategoryMeta struct {
			UUID            string  `db:"uuid"`
			TournamentID    string  `db:"tournament_id"`
			MaxParticipants *int    `db:"max_participants"`
			Fee             float64 `db:"fee"`
			TypeCode        string  `db:"type_code"`
			GenderCode      string  `db:"gender_code"`
			AgeCode         string  `db:"age_code"`
		}

		getCatMeta := func(catUUID string) (*CategoryMeta, error) {
			var meta CategoryMeta
			err := tx.Get(&meta, `
				SELECT tc.uuid, tc.tournament_id, tc.max_participants,
				       COALESCE(tc.fee, 0) as fee,
				       COALESCE(rtt.code, 'individual') as type_code,
				       COALESCE(rgd.code, 'mixed') as gender_code,
				       COALESCE(rag.code, 'umum') as age_code
				FROM tournament_categories tc
				LEFT JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
				LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
				LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
				WHERE tc.uuid = ? AND tc.tournament_id = ?
			`, catUUID, actualEventID)
			if err != nil {
				return nil, err
			}
			return &meta, nil
		}

		calculateCategoryFee := func(catMeta *CategoryMeta) float64 {
			if catMeta == nil {
				if event.EntryFee > 0 {
					return event.EntryFee
				}
				if event.FeeIndividual > 0 {
					return event.FeeIndividual
				}
				return 0
			}

			if event.FeeMode == "per_category" {
				if catMeta.Fee > 0 {
					return catMeta.Fee
				}
				if event.EntryFee > 0 {
					return event.EntryFee
				}
				if event.FeeIndividual > 0 {
					return event.FeeIndividual
				}
				return 0
			}

			// Default: fee_mode == "per_type"
			switch catMeta.TypeCode {
			case "team":
				if event.FeeTeam > 0 {
					return event.FeeTeam
				}
			case "mixed_team":
				if event.FeeMixedTeam > 0 {
					return event.FeeMixedTeam
				}
			default: // individual
				if event.FeeIndividual > 0 {
					return event.FeeIndividual
				}
			}

			if catMeta.Fee > 0 {
				return catMeta.Fee
			}
			if event.EntryFee > 0 {
				return event.EntryFee
			}
			return 0
		}

		validateEligibility := func(catMeta *CategoryMeta, archerGender string, archerDOB *time.Time) (string, string) {
			gender := strings.ToLower(strings.TrimSpace(archerGender))
			catGender := strings.ToLower(strings.TrimSpace(catMeta.GenderCode))
			if catGender == "men" && gender != "" && gender != "male" && gender != "men" {
				return "Atlet perempuan tidak dapat mendaftar di kategori putra (Men)", "gender_ineligible"
			}
			if catGender == "women" && gender != "" && gender != "female" && gender != "women" {
				return "Atlet laki-laki tidak dapat mendaftar di kategori putri (Women)", "gender_ineligible"
			}

			return "", ""
		}

		checkQuota := func(catMeta *CategoryMeta) bool {
			if isPrivileged || catMeta.MaxParticipants == nil || *catMeta.MaxParticipants <= 0 {
				return true
			}
			var currentCount int
			_ = tx.Get(&currentCount, `
				SELECT COUNT(*) FROM tournament_participants 
				WHERE category_id = ? AND payment_status != 'cancelled'
			`, catMeta.UUID)
			return currentCount < *catMeta.MaxParticipants
		}

		registrationDate := time.Now()
		var allCreatedParticipantUUIDs []string
		var createdTeamUUIDs []string
		registeredCategoryIDs := []string{}
		totalCalculatedRegistrationFee := 0.0

		// ─────────────────────────────────────────────────────────────────────────
		// MODE 1: INDIVIDUAL OR CAPTAIN REGISTRATION
		// ─────────────────────────────────────────────────────────────────────────
		if req.RegistrationMode != "club_delegation" {
			athleteIDToFind := req.AthleteID
			if athleteIDToFind == "" && userID != nil {
				athleteIDToFind = fmt.Sprintf("%v", userID)
			}
			var captainData struct {
				UUID   string     `db:"uuid"`
				Gender string     `db:"gender"`
				DOB    *time.Time `db:"date_of_birth"`
			}
			_ = tx.Get(&captainData, "SELECT uuid, COALESCE(gender, '') as gender, date_of_birth FROM archers WHERE uuid = ? OR id = ? OR username = ? LIMIT 1", athleteIDToFind, athleteIDToFind, athleteIDToFind)
			captainArcherUUID := captainData.UUID
			if captainArcherUUID == "" && userID != nil {
				captainArcherUUID = fmt.Sprintf("%v", userID)
			}

			// Combine individual category IDs
			allCategoryIDs := []string{}
			if strings.TrimSpace(req.EventCategoryID) != "" {
				allCategoryIDs = append(allCategoryIDs, strings.TrimSpace(req.EventCategoryID))
			}
			for _, id := range req.EventCategoryIDs {
				trimmed := strings.TrimSpace(id)
				if trimmed != "" {
					dup := false
					for _, existing := range allCategoryIDs {
						if existing == trimmed {
							dup = true
							break
						}
					}
					if !dup {
						allCategoryIDs = append(allCategoryIDs, trimmed)
					}
				}
			}

			// Register captain for individual categories
			for _, catID := range allCategoryIDs {
				catMeta, errCat := getCatMeta(catID)
				if errCat == nil && catMeta != nil {
					if !checkQuota(catMeta) {
						c.JSON(http.StatusConflict, gin.H{
							"error":       "Kuota untuk kategori ini sudah penuh",
							"code":        "quota_exceeded",
							"category_id": catMeta.UUID,
						})
						return
					}
					errMsg, errCode := validateEligibility(catMeta, captainData.Gender, captainData.DOB)
					if errMsg != "" {
						c.JSON(http.StatusBadRequest, gin.H{
							"error": errMsg,
							"code":  errCode,
						})
						return
					}
				}

				var existingUUID string
				_ = tx.Get(&existingUUID, `
					SELECT uuid FROM tournament_participants 
					WHERE tournament_id = ? AND archer_id = ? AND category_id = ? AND payment_status != 'cancelled'
					LIMIT 1
				`, actualEventID, captainArcherUUID, catID)

				if existingUUID != "" {
					allCreatedParticipantUUIDs = append(allCreatedParticipantUUIDs, existingUUID)
					registeredCategoryIDs = append(registeredCategoryIDs, catID)
					continue
				}

				partUUID := uuid.New().String()
				partFee := calculateCategoryFee(catMeta)
				totalCalculatedRegistrationFee += partFee

				_, err = tx.Exec(`
					INSERT INTO tournament_participants (
						uuid, tournament_id, archer_id, category_id, 
						registration_date, payment_status, payment_amount,
						registration_source
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, partUUID, actualEventID, captainArcherUUID, catID, registrationDate, paymentStatus, partFee, registrationSource)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan peserta", "details": err.Error()})
					return
				}

				allCreatedParticipantUUIDs = append(allCreatedParticipantUUIDs, partUUID)
				registeredCategoryIDs = append(registeredCategoryIDs, catID)
			}

			// Handle Team Registrations for Captain Mode
			allTeams := req.TeamRegistrations
			if req.TeamRegistration != nil && req.TeamRegistration.CategoryID != "" {
				allTeams = append(allTeams, *req.TeamRegistration)
			}

			for _, teamInput := range allTeams {
				if teamInput.CategoryID == "" {
					continue
				}

				// Find or resolve team category
				var teamCat struct {
					UUID         string `db:"uuid"`
					DivisionUUID string `db:"division_uuid"`
					CategoryUUID string `db:"category_uuid"`
				}
				_ = tx.Get(&teamCat, `SELECT uuid, division_uuid, category_uuid FROM tournament_categories WHERE uuid = ? AND tournament_id = ?`, teamInput.CategoryID, actualEventID)

				// Create Team record
				teamUUID := uuid.New().String()
				teamName := teamInput.TeamName
				if strings.TrimSpace(teamName) == "" {
					teamName = "Tim " + captainArcherUUID[:6]
				}

				_, err = tx.Exec(`
					INSERT INTO teams (uuid, tournament_id, event_id, category_id, team_name, status)
					VALUES (?, ?, ?, ?, ?, 'active')
				`, teamUUID, actualEventID, teamInput.CategoryID, teamInput.CategoryID, teamName)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat tim", "details": err.Error()})
					return
				}
				createdTeamUUIDs = append(createdTeamUUIDs, teamUUID)

				teamCatMeta, _ := getCatMeta(teamInput.CategoryID)
				teamCatFee := calculateCategoryFee(teamCatMeta)
				totalCalculatedRegistrationFee += teamCatFee

				// Process Team Members
				for orderIdx, member := range teamInput.Members {
					var memberPartUUID string

					if member.ParticipantID != nil && *member.ParticipantID != "" {
						memberPartUUID = *member.ParticipantID
					} else {
						// Resolve or auto-create archer
						var memberArcherUUID string
						if member.ArcherID != "" {
							_ = tx.Get(&memberArcherUUID, "SELECT uuid FROM archers WHERE uuid = ? OR id = ? LIMIT 1", member.ArcherID, member.ArcherID)
						}
						if memberArcherUUID == "" && member.FullName != "" {
							// Create quick archer profile
							memberArcherUUID = uuid.New().String()
							gender := member.Gender
							if gender == "" {
								gender = "male"
							}
							_, _ = tx.Exec(`
								INSERT INTO archers (uuid, full_name, gender, status, is_verified)
								VALUES (?, ?, ?, 'active', 0)
							`, memberArcherUUID, member.FullName, gender)
						}
						if memberArcherUUID == "" {
							memberArcherUUID = captainArcherUUID
						}

						// Find matching individual category ID for this member's gender & team division/age group
						var indivCatID string
						_ = tx.Get(&indivCatID, `
							SELECT tc.uuid
							FROM tournament_categories tc
							JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
							WHERE tc.tournament_id = ? 
							  AND tc.division_uuid = ? 
							  AND tc.category_uuid = ? 
							  AND rtt.code = 'individual'
							LIMIT 1
						`, actualEventID, teamCat.DivisionUUID, teamCat.CategoryUUID)
						if indivCatID == "" {
							indivCatID = teamInput.CategoryID
						}

						// Check if member already has a participant row
						_ = tx.Get(&memberPartUUID, `
							SELECT uuid FROM tournament_participants 
							WHERE tournament_id = ? AND archer_id = ? AND (category_id = ? OR category_id = ?) AND payment_status != 'cancelled'
							LIMIT 1
						`, actualEventID, memberArcherUUID, indivCatID, teamInput.CategoryID)

						if memberPartUUID == "" {
							// Create participant row for this member
							memberPartUUID = uuid.New().String()
							memberCatMeta, _ := getCatMeta(indivCatID)
							memberPartFee := calculateCategoryFee(memberCatMeta)
							if !member.PayIndividualFee {
								memberPartFee = 0
							} else {
								totalCalculatedRegistrationFee += memberPartFee
							}

							_, err = tx.Exec(`
								INSERT INTO tournament_participants (
									uuid, tournament_id, archer_id, category_id,
									registration_date, payment_status, payment_amount,
									registration_source
								) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
							`, memberPartUUID, actualEventID, memberArcherUUID, indivCatID, registrationDate, paymentStatus, memberPartFee, "self_register")
							if err != nil {
								c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan anggota tim", "details": err.Error()})
								return
							}
							allCreatedParticipantUUIDs = append(allCreatedParticipantUUIDs, memberPartUUID)
						}
					}

					// Verify member is not already on another team in this category
					var alreadyInTeam bool
					_ = tx.Get(&alreadyInTeam, `
						SELECT EXISTS(
							SELECT 1 FROM team_members tm
							JOIN teams t ON tm.team_id = t.uuid
							WHERE t.category_id = ? AND tm.participant_id = ? AND t.status != 'cancelled'
						)
					`, teamInput.CategoryID, memberPartUUID)
					if alreadyInTeam {
						c.JSON(http.StatusConflict, gin.H{
							"error": fmt.Sprintf("Atlet %s sudah terdaftar pada tim lain di kategori ini", member.FullName),
							"code":  "duplicate_team_member",
						})
						return
					}

					// Link to team_members
					tmUUID := uuid.New().String()
					_, err = tx.Exec(`
						INSERT INTO team_members (uuid, team_id, participant_id, member_order)
						VALUES (?, ?, ?, ?)
					`, tmUUID, teamUUID, memberPartUUID, orderIdx+1)
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menautkan anggota ke tim", "details": err.Error()})
						return
					}
				}
			}

		} else {
			// ─────────────────────────────────────────────────────────────────────
			// MODE 2: CLUB DELEGATION REGISTRATION
			// ─────────────────────────────────────────────────────────────────────
			for _, ath := range req.DelegationAthletes {
				var athArcherUUID string
				if ath.ArcherID != "" {
					_ = tx.Get(&athArcherUUID, "SELECT uuid FROM archers WHERE uuid = ? OR id = ? LIMIT 1", ath.ArcherID, ath.ArcherID)
				}
				if athArcherUUID == "" && strings.TrimSpace(ath.Email) != "" {
					_ = tx.Get(&athArcherUUID, "SELECT uuid FROM archers WHERE email = ? LIMIT 1", strings.ToLower(strings.TrimSpace(ath.Email)))
				}
				if athArcherUUID == "" && ath.FullName != "" {
					athArcherUUID = uuid.New().String()
					gender := ath.Gender
					if gender == "" {
						gender = "male"
					}

					defaultPass := "Archeris123!"
					hashedPass, _ := bcrypt.GenerateFromPassword([]byte(defaultPass), bcrypt.DefaultCost)
					username := utils.CleanUsername(ath.FullName)
					if username == "" {
						username = "archer"
					}
					var uExists bool
					_ = tx.Get(&uExists, "SELECT EXISTS(SELECT 1 FROM archers WHERE username = ?)", username)
					if uExists {
						username = fmt.Sprintf("%s-%s", username, uuid.New().String()[:6])
					}

					var emailVal *string
					if strings.TrimSpace(ath.Email) != "" {
						e := strings.ToLower(strings.TrimSpace(ath.Email))
						emailVal = &e
					}
					var phoneVal *string
					if strings.TrimSpace(ath.Phone) != "" {
						p := strings.TrimSpace(ath.Phone)
						phoneVal = &p
					}
					var dobVal *string
					if strings.TrimSpace(ath.DateOfBirth) != "" {
						d := strings.TrimSpace(ath.DateOfBirth)
						dobVal = &d
					}

					var clubIDVal *string
					if ath.ClubID != nil && *ath.ClubID != "" {
						clubIDVal = ath.ClubID
					} else if ath.ClubName != nil && strings.TrimSpace(*ath.ClubName) != "" {
						cleanClubName := strings.TrimSpace(*ath.ClubName)
						var existingClubID string
						if err := tx.Get(&existingClubID, "SELECT uuid FROM clubs WHERE LOWER(TRIM(name)) = LOWER(TRIM(?)) LIMIT 1", cleanClubName); err == nil {
							clubIDVal = &existingClubID
						} else {
							newClubUUID := uuid.New().String()
							clubSlug := utils.CleanSlug(cleanClubName)
							if clubSlug == "" {
								clubSlug = "club-" + uuid.New().String()[:6]
							}
							_, err := tx.Exec("INSERT INTO clubs (uuid, name, slug, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())", newClubUUID, cleanClubName, clubSlug)
							if err == nil {
								clubIDVal = &newClubUUID
							}
						}
					}

					_, _ = tx.Exec(`
						INSERT INTO archers (
							uuid, username, email, phone, date_of_birth, password, 
							full_name, gender, club_id, status, is_verified, created_at, updated_at
						) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', 1, NOW(), NOW())
					`, athArcherUUID, username, emailVal, phoneVal, dobVal, string(hashedPass), ath.FullName, gender, clubIDVal)
				}
				if athArcherUUID == "" {
					continue
				}

				for _, catID := range ath.CategoryIDs {
					trimmed := strings.TrimSpace(catID)
					if trimmed == "" {
						continue
					}

					catMeta, errCat := getCatMeta(trimmed)
					if errCat == nil && catMeta != nil {
						if !checkQuota(catMeta) {
							c.JSON(http.StatusConflict, gin.H{
								"error":       fmt.Sprintf("Kuota untuk kategori pilihan atlet %s sudah penuh", ath.FullName),
								"code":        "quota_exceeded",
								"category_id": catMeta.UUID,
							})
							return
						}
						var athDOB *time.Time
						if ath.DateOfBirth != "" {
							if t, err := time.Parse("2006-01-02", strings.TrimSpace(ath.DateOfBirth)); err == nil {
								athDOB = &t
							}
						}
						errMsg, errCode := validateEligibility(catMeta, ath.Gender, athDOB)
						if errMsg != "" {
							c.JSON(http.StatusBadRequest, gin.H{
								"error": fmt.Sprintf("Atlet %s: %s", ath.FullName, errMsg),
								"code":  errCode,
							})
							return
						}
					}

					var existingUUID string
					_ = tx.Get(&existingUUID, `
						SELECT uuid FROM tournament_participants 
						WHERE tournament_id = ? AND archer_id = ? AND category_id = ? AND payment_status != 'cancelled'
						LIMIT 1
					`, actualEventID, athArcherUUID, trimmed)

					if existingUUID != "" {
						allCreatedParticipantUUIDs = append(allCreatedParticipantUUIDs, existingUUID)
						continue
					}

					partUUID := uuid.New().String()
					partFee := calculateCategoryFee(catMeta)
					totalCalculatedRegistrationFee += partFee

					_, err = tx.Exec(`
						INSERT INTO tournament_participants (
							uuid, tournament_id, archer_id, category_id,
							registration_date, payment_status, payment_amount,
							registration_source
						) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
					`, partUUID, actualEventID, athArcherUUID, trimmed, registrationDate, paymentStatus, partFee, "invited")
					if err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan atlet delegasi", "details": err.Error()})
						return
					}
					allCreatedParticipantUUIDs = append(allCreatedParticipantUUIDs, partUUID)
					registeredCategoryIDs = append(registeredCategoryIDs, trimmed)
				}
			}

			// Book reserved team slots for delegation
			for _, tBooking := range req.DelegationTeams {
				count := tBooking.Count
				if count <= 0 {
					count = 1
				}
				teamCatMeta, _ := getCatMeta(tBooking.CategoryID)
				teamFee := calculateCategoryFee(teamCatMeta)

				for k := 0; k < count; k++ {
					teamUUID := uuid.New().String()
					tName := tBooking.TeamName
					if count > 1 {
						tName = fmt.Sprintf("%s %d", tBooking.TeamName, k+1)
					}
					if strings.TrimSpace(tName) == "" {
						tName = "Tim Klub"
					}
					_, err = tx.Exec(`
						INSERT INTO teams (uuid, tournament_id, event_id, category_id, team_name, status)
						VALUES (?, ?, ?, ?, ?, 'active')
					`, teamUUID, actualEventID, tBooking.CategoryID, tBooking.CategoryID, tName)
					if err == nil {
						createdTeamUUIDs = append(createdTeamUUIDs, teamUUID)
						totalCalculatedRegistrationFee += teamFee
					}
				}
			}
		}

		if len(allCreatedParticipantUUIDs) == 0 && len(createdTeamUUIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada peserta atau tim yang valid untuk didaftarkan"})
			return
		}

		firstUUID := ""
		if len(allCreatedParticipantUUIDs) > 0 {
			firstUUID = allCreatedParticipantUUIDs[0]
		} else if len(createdTeamUUIDs) > 0 {
			firstUUID = createdTeamUUIDs[0]
		}

		var freeTxReference string
		var freeTxUUID string
		isFreeEvent := totalCalculatedRegistrationFee <= 0 && req.PaymentAmount <= 0 && event.EntryFee <= 0 && event.FeeIndividual <= 0
		if isFreeEvent {
			freeTxUUID = uuid.New().String()
			freeTxReference = fmt.Sprintf("PAY-FREE-%s", strings.ToUpper(uuid.New().String()[:8]))
			payingUserID := ""
			if userID != nil {
				payingUserID = fmt.Sprintf("%v", userID)
			}
			if payingUserID == "" || payingUserID == "<nil>" {
				payingUserID = req.AthleteID
			}

			_, err = tx.Exec(`
				INSERT INTO payment_transactions (
					uuid, reference, gateway_reference, user_id, tournament_id, registration_id,
					amount, fee_amount, total_amount, payment_method, payment_channel,
					status, paid_at, created_at, updated_at
				) VALUES (
					?, ?, ?, ?, ?, ?,
					0, 0, 0, 'free', 'free',
					'paid', NOW(), NOW(), NOW()
				)
			`, freeTxUUID, freeTxReference, freeTxReference, payingUserID, actualEventID, firstUUID)

			if err == nil && len(allCreatedParticipantUUIDs) > 0 {
				qUp, argsUp, errIn := sqlx.In("UPDATE tournament_participants SET payment_id = ?, payment_status = 'paid' WHERE uuid IN (?)", freeTxUUID, allCreatedParticipantUUIDs)
				if errIn == nil {
					qUp = tx.Rebind(qUp)
					_, _ = tx.Exec(qUp, argsUp...)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pendaftaran"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":           "Pendaftaran berhasil",
			"registration_id":   firstUUID,
			"participant_ids":   allCreatedParticipantUUIDs,
			"team_ids":          createdTeamUUIDs,
			"category_ids":      registeredCategoryIDs,
			"registration_mode": req.RegistrationMode,
			"reference_code":    freeTxReference,
			"transaction_id":    freeTxUUID,
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
			UUID                 string     `db:"uuid"`
			OrganizerID          string     `db:"organizer_id"`
			StartDate            *time.Time `db:"start_date"`
			QuotaMaxParticipants *int       `db:"quota_max_participants"`
		}
		if err := db.Get(&event, `SELECT uuid, organizer_id, start_date, quota_max_participants FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}
		actualEventID := event.UUID

		// Determine privileges
		userID, _ := c.Get("user_id")
		userRole, _ := c.Get("role")
		orgID, _ := c.Get("org_id")

		isPrivileged := userRole == "admin" || userRole == "superadmin" || (userRole == "organizer" && (orgID == nil || fmt.Sprintf("%v", orgID) == event.OrganizerID || event.OrganizerID == ""))

		paymentStatus := "unpaid"
		registrationSource := "self_register"

		if isPrivileged {
			if req.PaymentStatus != "" {
				paymentStatus = req.PaymentStatus
			} else {
				paymentStatus = "paid"
			}

			if req.RegistrationSource != "" {
				registrationSource = req.RegistrationSource
			} else {
				registrationSource = "invited"
			}
		}

		// Resolve all archer UUIDs in one query
		cleanedIDs := make([]string, 0, len(req.AthleteIDs))
		for _, id := range req.AthleteIDs {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				cleanedIDs = append(cleanedIDs, trimmed)
			}
		}

		if len(cleanedIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pilih minimal satu pemanah"})
			return
		}

		type archerRow struct {
			UUID      string     `db:"uuid"`
			ID        *string    `db:"id"`
			FullName  string     `db:"full_name"`
			Gender    *string    `db:"gender"`
			BirthDate *time.Time `db:"birth_date"`
		}
		query, args, err := sqlx.In(`SELECT uuid, id, full_name, gender, birth_date FROM archers WHERE uuid IN (?) OR id IN (?)`, cleanedIDs, cleanedIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build archer query", "details": err.Error()})
			return
		}
		query = db.Rebind(query)
		var archerRows []archerRow
		if err := db.Select(&archerRows, query, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve archers", "details": err.Error()})
			return
		}

		// Map input IDs -> resolved UUIDs (deduplicate)
		seenUUIDs := map[string]bool{}
		archerUUIDs := []string{}
		archerMap := map[string]archerRow{}
		for _, row := range archerRows {
			if !seenUUIDs[row.UUID] {
				seenUUIDs[row.UUID] = true
				archerUUIDs = append(archerUUIDs, row.UUID)
				archerMap[row.UUID] = row
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

		// Pre-fetch category metadata for eligibility & quota checking
		type catMetaRow struct {
			UUID            string  `db:"uuid"`
			MaxParticipants *int    `db:"max_participants"`
			GenderCode      *string `db:"gender_code"`
			AgeCode         *string `db:"age_code"`
			CategoryName    string  `db:"category_name"`
		}
		var catMetaRows []catMetaRow
		catQuery, catArgs, catErr := sqlx.In(`
			SELECT tc.uuid, tc.max_participants,
			       COALESCE(rgd.code, 'mixed') as gender_code,
			       COALESCE(rag.code, 'umum') as age_code,
			       COALESCE(tc.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' ', COALESCE(rag.name, ''), ' ', COALESCE(rgd.name, ''))) as category_name
			FROM tournament_categories tc
			LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
			LEFT JOIN ref_bow_types rbt ON tc.division_uuid = rbt.uuid
			WHERE tc.uuid IN (?) AND tc.tournament_id = ?
		`, allCategoryIDs, actualEventID)
		if catErr == nil {
			catQuery = db.Rebind(catQuery)
			_ = db.Select(&catMetaRows, catQuery, catArgs...)
		}
		catMap := make(map[string]catMetaRow)
		for _, cm := range catMetaRows {
			catMap[cm.UUID] = cm
		}

		// Pre-fetch current counts for quotas
		var currentTotalParticipants int
		_ = db.Get(&currentTotalParticipants, `SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = ? AND payment_status != 'cancelled'`, actualEventID)

		type catCountRow struct {
			CategoryID string `db:"category_id"`
			Count      int    `db:"c"`
		}
		var catCounts []catCountRow
		_ = db.Select(&catCounts, `SELECT category_id, COUNT(*) as c FROM tournament_participants WHERE tournament_id = ? AND payment_status != 'cancelled' GROUP BY category_id`, actualEventID)
		catCountMap := make(map[string]int)
		for _, cc := range catCounts {
			catCountMap[cc.CategoryID] = cc.Count
		}

		type RejectedEntry struct {
			ArcherUUID   string `json:"archer_id"`
			ArcherName   string `json:"archer_name"`
			CategoryID   string `json:"category_id"`
			CategoryName string `json:"category_name"`
			Reason       string `json:"reason"`
		}
		rejectedEntries := make([]RejectedEntry, 0)

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
			archer := archerMap[archerUUID]

			// Get or generate a shared QR for this archer x event if status is lunas
			var qrRaw *string
			if paymentStatus == "lunas" || paymentStatus == "paid" {
				var existingQR sql.NullString
				_ = tx.Get(&existingQR, "SELECT qr_raw FROM tournament_participants WHERE tournament_id = ? AND archer_id = ? AND qr_raw IS NOT NULL LIMIT 1", actualEventID, archerUUID)
				if existingQR.Valid {
					qrRaw = &existingQR.String
				} else {
					randomQR := uuid.New().String()
					qrRaw = &randomQR
				}
			}

			for _, catID := range allCategoryIDs {
				cat, catExists := catMap[catID]
				catName := catID
				if catExists && cat.CategoryName != "" {
					catName = cat.CategoryName
				}

				// Check if already registered
				var exists bool
				if err := tx.Get(&exists, `SELECT EXISTS(SELECT 1 FROM tournament_participants WHERE tournament_id = ? AND archer_id = ? AND category_id = ?)`, actualEventID, archerUUID, catID); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check registration status"})
					return
				}
				if exists {
					skippedCount++
					continue
				}

				// 1. Check overall tournament quota
				if event.QuotaMaxParticipants != nil && *event.QuotaMaxParticipants > 0 {
					if currentTotalParticipants >= *event.QuotaMaxParticipants {
						rejectedEntries = append(rejectedEntries, RejectedEntry{
							ArcherUUID:   archerUUID,
							ArcherName:   archer.FullName,
							CategoryID:   catID,
							CategoryName: catName,
							Reason:       "Kuota keseluruhan turnamen telah penuh",
						})
						continue
					}
				}

				// 2. Check category quota
				if catExists && cat.MaxParticipants != nil && *cat.MaxParticipants > 0 {
					if catCountMap[catID] >= *cat.MaxParticipants {
						rejectedEntries = append(rejectedEntries, RejectedEntry{
							ArcherUUID:   archerUUID,
							ArcherName:   archer.FullName,
							CategoryID:   catID,
							CategoryName: catName,
							Reason:       fmt.Sprintf("Kuota kategori '%s' telah penuh (maks. %d peserta)", catName, *cat.MaxParticipants),
						})
						continue
					}
				}

				// 3. Gender eligibility check
				if catExists && cat.GenderCode != nil {
					catGender := strings.ToLower(strings.TrimSpace(*cat.GenderCode))
					archerGender := ""
					if archer.Gender != nil {
						archerGender = strings.ToLower(strings.TrimSpace(*archer.Gender))
					}
					if catGender == "men" && archerGender != "" && archerGender != "male" && archerGender != "men" {
						rejectedEntries = append(rejectedEntries, RejectedEntry{
							ArcherUUID:   archerUUID,
							ArcherName:   archer.FullName,
							CategoryID:   catID,
							CategoryName: catName,
							Reason:       "Atlet perempuan tidak dapat didaftarkan di kategori putra (Men)",
						})
						continue
					}
					if catGender == "women" && archerGender != "" && archerGender != "female" && archerGender != "women" {
						rejectedEntries = append(rejectedEntries, RejectedEntry{
							ArcherUUID:   archerUUID,
							ArcherName:   archer.FullName,
							CategoryID:   catID,
							CategoryName: catName,
							Reason:       "Atlet laki-laki tidak dapat didaftarkan di kategori putri (Women)",
						})
						continue
					}
				}

				// 4. World Archery Age Calculation: eventYear - birthYear
				if catExists && cat.AgeCode != nil && archer.BirthDate != nil {
					refYear := time.Now().Year()
					if event.StartDate != nil {
						refYear = event.StartDate.Year()
					}
					waAge := refYear - archer.BirthDate.Year()
					ageCode := strings.ToLower(strings.TrimSpace(*cat.AgeCode))

					ageRejected := false
					ageReason := ""
					switch ageCode {
					case "u12", "u-12":
						if waAge > 12 {
							ageRejected = true
							ageReason = fmt.Sprintf("Usia atlet (%d tahun berdasarkan tahun kompetisi WA) melebihi batas kategori U-12", waAge)
						}
					case "u13", "u-13":
						if waAge > 13 {
							ageRejected = true
							ageReason = fmt.Sprintf("Usia atlet (%d tahun berdasarkan tahun kompetisi WA) melebihi batas kategori U-13", waAge)
						}
					case "u15", "u-15":
						if waAge > 15 {
							ageRejected = true
							ageReason = fmt.Sprintf("Usia atlet (%d tahun berdasarkan tahun kompetisi WA) melebihi batas kategori U-15", waAge)
						}
					case "u18", "u-18":
						if waAge > 18 {
							ageRejected = true
							ageReason = fmt.Sprintf("Usia atlet (%d tahun berdasarkan tahun kompetisi WA) melebihi batas kategori U-18", waAge)
						}
					case "master":
						if waAge < 50 {
							ageRejected = true
							ageReason = fmt.Sprintf("Usia atlet (%d tahun berdasarkan tahun kompetisi WA) belum memenuhi batas minimal kategori Master (50+)", waAge)
						}
					}

					if ageRejected {
						rejectedEntries = append(rejectedEntries, RejectedEntry{
							ArcherUUID:   archerUUID,
							ArcherName:   archer.FullName,
							CategoryID:   catID,
							CategoryName: catName,
							Reason:       ageReason,
						})
						continue
					}
				}

				participantUUID := uuid.New().String()
				_, err = tx.Exec(`
						INSERT INTO tournament_participants (
						uuid, tournament_id, archer_id, category_id,
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
				currentTotalParticipants++
				catCountMap[catID]++
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
			"rejected":   rejectedEntries,
		})
	}
}

// UnregisterFromEvent allows an archer to cancel ALL their registrations from an event by event ID.
// Uses DELETE /tournaments/:id/participants/me
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
		if err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		userEmailVal, _ := c.Get("email")
		userEmail := fmt.Sprintf("%v", userEmailVal)

		// Resolve archer UUID for this user
		var archerID string
		_ = db.Get(&archerID, `SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?) LIMIT 1`, uid, uid, userEmail)
		if archerID == "" {
			archerID = uid
		}

		// Find all registrations for this archer in this event
		type RegInfo struct {
			UUID          string `db:"uuid"`
			PaymentStatus string `db:"payment_status"`
		}
		var regs []RegInfo
		if err := db.Select(&regs, `SELECT uuid, payment_status FROM tournament_participants WHERE (tournament_id = ? OR tournament_id IN (SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?)) AND (archer_id = ? OR archer_id = ? OR archer_id IN (SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?)))`, actualEventID, actualEventID, actualEventID, archerID, uid, uid, uid, userEmail); err != nil || len(regs) == 0 {
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

		// Soft cancel all registrations for this archer in this event
		if _, err := db.Exec(`
			UPDATE tournament_participants 
			SET payment_status = 'cancelled', updated_at = NOW() 
			WHERE (tournament_id = ? OR tournament_id IN (SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?)) 
			  AND (archer_id = ? OR archer_id = ? OR archer_id IN (SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?)))
			  AND payment_status NOT IN ('paid', 'lunas')
		`, actualEventID, actualEventID, actualEventID, archerID, uid, uid, uid, userEmail); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membatalkan pendaftaran"})
			return
		}

		// Also cancel pending payment transactions
		for _, r := range regs {
			_, _ = db.Exec(`
				UPDATE payment_transactions 
				SET status = 'cancelled', updated_at = NOW() 
				WHERE registration_id = ? AND status IN ('pending', 'awaiting_verification')
			`, r.UUID)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Pendaftaran berhasil dibatalkan"})
	}
}

// CancelParticipantRegistration allows an archer or organizer to cancel their registration
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
		err := db.Get(&archerID, "SELECT archer_id FROM tournament_participants WHERE uuid = ?", participantID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Registration not found"})
			return
		}

		// Check if the participant belongs to the logged-in user or caller is organizer/admin
		var userArcherID string
		err = db.Get(&userArcherID, "SELECT uuid FROM archers WHERE uuid = ? LIMIT 1", userID)
		if err != nil || userArcherID != archerID {
			userRole, _ := c.Get("user_role")
			var orgID string
			_ = db.Get(&orgID, "SELECT e.organizer_id FROM tournaments e JOIN tournament_participants tp ON tp.tournament_id = e.uuid WHERE tp.uuid = ?", participantID)
			if orgID != userID.(string) && userRole != "root" && userRole != "admin" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Anda hanya dapat membatalkan pendaftaran sendiri"})
				return
			}
		}

		// Check if manual payment is already approved or gateway already paid
		var payStatus string
		_ = db.Get(&payStatus, "SELECT COALESCE(payment_status,'unpaid') FROM tournament_participants WHERE uuid = ?", participantID)
		if payStatus == "paid" || payStatus == "lunas" {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Pembayaran sudah dikonfirmasi. Tidak dapat membatalkan pendaftaran.",
				"code": "payment_already_confirmed",
			})
			return
		}

		// Soft cancel the participant registration
		_, err = db.Exec("UPDATE tournament_participants SET payment_status = 'cancelled', updated_at = NOW() WHERE uuid = ?", participantID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel registration"})
			return
		}

		// Also cancel pending payment transactions if any
		_, _ = db.Exec(`
			UPDATE payment_transactions 
			SET status = 'cancelled', updated_at = NOW() 
			WHERE (registration_id = ? OR uuid IN (SELECT payment_id FROM tournament_participants WHERE uuid = ?))
			  AND status IN ('pending', 'awaiting_verification')
		`, participantID, participantID)

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
		err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
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
			SELECT tp.uuid, tp.archer_id, a.full_name FROM tournament_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			WHERE tp.tournament_id = ? AND (tp.uuid = ? OR a.username = ? OR a.id = ?)
			LIMIT 1
		`, actualEventID, participantID, participantID, participantID)

		if err != nil {
			fmt.Printf("[DEBUG] Delete lookup failed for Event: %s, ID: %s. Error: %v\n", actualEventID, participantID, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Participant not found", "details": err.Error()})
			return
		}

		archerID := pInfo.ArcherID

		// Check if participant can be kicked
		var participantCheck struct {
			PaymentStatus string `db:"payment_status"`
			ReregisteredAt *time.Time `db:"last_reregistration_at"`
		}
		err = db.Get(&participantCheck, "SELECT COALESCE(payment_status,'unpaid') as payment_status, last_reregistration_at FROM tournament_participants WHERE uuid = ?", pInfo.UUID)
		if err == nil {
			if participantCheck.PaymentStatus == "paid" || participantCheck.PaymentStatus == "lunas" {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Peserta tidak dapat dikeluarkan karena pembayaran sudah lunas. Tambahkan refund terlebih dahulu.",
					"code": "participant_paid",
				})
				return
			}
			if participantCheck.ReregisteredAt != nil {
				c.JSON(http.StatusForbidden, gin.H{
					"error": "Peserta tidak dapat dikeluarkan karena sudah melakukan registrasi ulang.",
					"code": "participant_reregistered",
				})
				return
			}
		}

		fmt.Printf("[DEBUG] Found participant to delete: %s (Archer: %s) - removing ALL registrations for this archer in event\n", pInfo.FullName, archerID)

		// Check if this archer (any of their participant rows) is in any elimination match
		var inMatch bool
		err = db.Get(&inMatch, `
			SELECT EXISTS(
				SELECT 1 FROM elimination_matches em
				JOIN elimination_entries ee ON (em.entry_a_uuid = ee.uuid OR em.entry_b_uuid = ee.uuid)
				WHERE ee.participant_uuid IN (SELECT uuid FROM tournament_participants WHERE archer_id = ? AND tournament_id = ?)
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
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE archer_id = ? AND tournament_id = ?)
			)
		`, archerID, actualEventID)

		// End scores
		tx.Exec(`
			DELETE FROM qualification_end_scores 
			WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE archer_id = ? AND tournament_id = ?)
		`, archerID, actualEventID)

		// 2. Delete qualification target assignments
		tx.Exec(`
			DELETE FROM qualification_target_assignments 
			WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE archer_id = ? AND tournament_id = ?)
		`, archerID, actualEventID)

		// 3. Cleanup elimination entries if any
		tx.Exec(`
			DELETE FROM elimination_entries 
			WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE archer_id = ? AND tournament_id = ?)
		`, archerID, actualEventID)

		// 4. Delete ALL tournament_participants for this archer in this event (handles multi-category registrations)
		_, err = tx.Exec("DELETE FROM tournament_participants WHERE archer_id = ? AND tournament_id = ?", archerID, actualEventID)
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
			FullName          *string   `json:"full_name"`
			ClubID            *string   `json:"club_id"`
			CategoryID        *string   `json:"category_id"`
			CategoryIDs       []string  `json:"category_ids"`
			TargetName        *string   `json:"target_name"`
			BackNumber        *string   `json:"back_number"`
			PaymentStatus     *string   `json:"payment_status"`
			PaymentAmount     *float64  `json:"payment_amount"`
			PaymentProofURLs  *[]string `json:"payment_proof_urls"`
			IsVerified        *bool     `json:"is_verified"`
			Reregistered      *bool     `json:"reregistered"`
			Notes             *string   `json:"notes"`
			PaymentMethod     *string   `json:"payment_method"`
			RecordTransaction *bool     `json:"record_transaction"`
			TransactionNotes  *string   `json:"transaction_notes"`
			TransactionAmount *float64  `json:"transaction_amount"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve event slug to UUID
		var actualEventID string
		err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
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
			SELECT tp.uuid, tp.archer_id FROM tournament_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			WHERE tp.tournament_id = ? AND (
				tp.uuid = ? OR 
				tp.archer_id = ? OR
				a.username = ? OR 
				a.id = ? OR
				LOWER(REPLACE(a.full_name, ' ', '-')) = LOWER(?)
			)
			LIMIT 1
		`, actualEventID, participantID, participantID, participantID, participantID, participantID)

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
		query := "UPDATE tournament_participants SET updated_at = NOW()"
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
				err = db.Select(&currentIDs, "SELECT category_id FROM tournament_participants WHERE archer_id = ? AND tournament_id = ?", *archerID, actualEventID)

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
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi perubahan kategori"})
					return
				}
				defer tx.Rollback()

				// First, check score protection for all categories to be removed
				for _, catID := range toRemove {
					var hasScores bool
					errScore := tx.Get(&hasScores, `
						SELECT (
							EXISTS(
								SELECT 1 FROM qualification_end_scores qes
								JOIN tournament_participants tp ON qes.participant_uuid = tp.uuid
								WHERE tp.archer_id = ? AND tp.tournament_id = ? AND tp.category_id = ?
							) OR EXISTS(
								SELECT 1 FROM tournament_participants tp
								WHERE tp.archer_id = ? AND tp.tournament_id = ? AND tp.category_id = ? AND tp.qual_score IS NOT NULL AND tp.qual_score > 0
							) OR EXISTS(
								SELECT 1 FROM elimination_entries ee
								JOIN tournament_participants tp ON ee.participant_uuid = tp.uuid
								WHERE tp.archer_id = ? AND tp.tournament_id = ? AND tp.category_id = ?
							)
						)
					`, *archerID, actualEventID, catID, *archerID, actualEventID, catID, *archerID, actualEventID, catID)

					if errScore == nil && hasScores {
						tx.Rollback()
						c.JSON(http.StatusBadRequest, gin.H{
							"error": "Kategori tidak dapat dihapus karena atlet telah memiliki catatan skor pertandingan resmi atau terdaftar dalam bagan eliminasi.",
							"code":  "CATEGORY_LOCKED_SCORED",
						})
						return
					}

					// Auto-release target allocation & target board if unscored
					_, _ = tx.Exec(`DELETE FROM qualification_target_assignments WHERE participant_uuid IN (
						SELECT uuid FROM tournament_participants WHERE archer_id = ? AND tournament_id = ? AND category_id = ?
					)`, *archerID, actualEventID, catID)

					_, _ = tx.Exec(`DELETE FROM target_board_qualification WHERE participant_id IN (
						SELECT uuid FROM tournament_participants WHERE archer_id = ? AND tournament_id = ? AND category_id = ?
					)`, *archerID, actualEventID, catID)

					// Delete category registration
					_, _ = tx.Exec("DELETE FROM tournament_participants WHERE archer_id = ? AND tournament_id = ? AND category_id = ?", *archerID, actualEventID, catID)
				}

				// Look up any existing qr_raw for this archer in this event
				var existingQrRaw *string
				_ = tx.Get(&existingQrRaw, `SELECT qr_raw FROM tournament_participants WHERE archer_id = ? AND tournament_id = ? AND qr_raw IS NOT NULL LIMIT 1`, *archerID, actualEventID)

				// Add new registrations
				for _, catID := range toAdd {
					var catExists bool
					_ = tx.Get(&catExists, "SELECT EXISTS(SELECT 1 FROM tournament_categories WHERE uuid = ? AND tournament_id = ?)", catID, actualEventID)
					if catExists {
						newUUID := uuid.New().String()
						_, _ = tx.Exec(`INSERT INTO tournament_participants (uuid, tournament_id, archer_id, category_id, registration_date, payment_status, payment_amount, qr_raw, registration_source) 
						VALUES (?, ?, ?, ?, NOW(), ?, ?, ?, 'admin_created')`,
							newUUID, actualEventID, *archerID, catID,
							models.FromPtr(req.PaymentStatus), models.FromPtrFloat(req.PaymentAmount), existingQrRaw)
					}
				}

				if errCommit := tx.Commit(); errCommit != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan sinkronisasi kategori"})
					return
				}
			} else if req.CategoryID != nil {
				// Old behavior: single category update
				var categoryExists bool
				err = db.Get(&categoryExists, `
					SELECT EXISTS(SELECT 1 FROM tournament_categories 
					WHERE uuid = ? AND tournament_id = ?)
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
				err = db.Get(&currentQR, "SELECT qr_raw FROM tournament_participants WHERE tournament_id = ? AND archer_id = ? AND qr_raw IS NOT NULL LIMIT 1", actualEventID, *pInfo.ArcherID)
				if err != nil || !currentQR.Valid {
					qrRaw := uuid.New().String()
					query += ", qr_raw = ?"
					args = append(args, qrRaw)
				}
			}
		}

		if req.PaymentAmount != nil {
			query += ", payment_amount = ?"
			args = append(args, *req.PaymentAmount)
		}
		if req.Reregistered != nil {
			query += ", last_reregistration_at = ?"
			if *req.Reregistered {
				now := time.Now()
				args = append(args, now)
			} else {
				args = append(args, nil)
			}
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database", "details": err.Error()})
			return
		}
		defer tx.Rollback()

		// Handle Archer Profile Updates (Name & Club)
		if pInfo.ArcherID != nil {
			if req.FullName != nil && strings.TrimSpace(*req.FullName) != "" {
				_, err = tx.Exec("UPDATE archers SET full_name = ? WHERE uuid = ?", strings.TrimSpace(*req.FullName), *pInfo.ArcherID)
				if err != nil {
					fmt.Printf("[ERROR] Failed to update archer full_name: %v\n", err)
				}
			}
			if req.ClubID != nil {
				var clubVal interface{}
				if *req.ClubID == "" || *req.ClubID == "independent" {
					clubVal = nil
				} else {
					clubVal = *req.ClubID
				}
				_, err = tx.Exec("UPDATE archers SET club_id = ? WHERE uuid = ?", clubVal, *pInfo.ArcherID)
				if err != nil {
					fmt.Printf("[ERROR] Failed to update archer club_id: %v\n", err)
				}
			}
		}

		// Handle IsVerified
		if req.IsVerified != nil {
			if pInfo.ArcherID != nil {
				_, err = tx.Exec("UPDATE archers SET is_verified = ? WHERE uuid = ?", *req.IsVerified, *pInfo.ArcherID)
				if err != nil {
					fmt.Printf("[ERROR] Failed to update archer verification: %v\n", err)
				}
			}
		}

		if len(args) > 0 {
			query += " WHERE tournament_id = ? AND archer_id = ?"
			args = append(args, actualEventID, *pInfo.ArcherID)

			_, err = tx.Exec(query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data peserta", "details": err.Error()})
				return
			}
		}

		// Synchronize payment_transactions status if payment_status was updated
		if req.PaymentStatus != nil {
			statusVal := strings.ToLower(*req.PaymentStatus)
			if statusVal == "paid" || statusVal == "lunas" {
				_, _ = tx.Exec(`
					UPDATE payment_transactions 
					SET status = 'paid', updated_at = NOW(), paid_at = COALESCE(paid_at, NOW()) 
					WHERE ((tournament_id = ? AND user_id = ?) 
					   OR registration_id = ? 
					   OR uuid IN (SELECT payment_id FROM tournament_participants WHERE tournament_id = ? AND archer_id = ? AND payment_id IS NOT NULL))
					  AND status IN ('pending', 'awaiting_verification', 'unpaid')
				`, actualEventID, *pInfo.ArcherID, actualParticipantID, actualEventID, *pInfo.ArcherID)
			} else if statusVal == "pending" || statusVal == "unpaid" {
				_, _ = tx.Exec(`
					UPDATE payment_transactions 
					SET status = 'pending', updated_at = NOW() 
					WHERE ((tournament_id = ? AND user_id = ?) 
					   OR registration_id = ? 
					   OR uuid IN (SELECT payment_id FROM tournament_participants WHERE tournament_id = ? AND archer_id = ? AND payment_id IS NOT NULL))
					  AND status NOT IN ('paid', 'success', 'settlement', 'completed')
				`, actualEventID, *pInfo.ArcherID, actualParticipantID, actualEventID, *pInfo.ArcherID)
			}
		}

		// Financial Ledger Recording: Record adjustment / cash on desk / refund in payment_transactions
		if req.RecordTransaction != nil && *req.RecordTransaction {
			trxAmount := 0.0
			if req.TransactionAmount != nil && *req.TransactionAmount > 0 {
				trxAmount = *req.TransactionAmount
			} else if req.PaymentAmount != nil {
				trxAmount = *req.PaymentAmount
			}

			if trxAmount > 0 {
				method := "cash_on_desk"
				if req.PaymentMethod != nil && *req.PaymentMethod != "" {
					method = *req.PaymentMethod
				}

				status := "paid"
				if strings.ToLower(method) == "refund" || (req.TransactionNotes != nil && strings.Contains(strings.ToLower(*req.TransactionNotes), "refund")) {
					status = "refunded"
				}

				adminUserID := ""
				if uid, exists := c.Get("user_id"); exists && uid != nil {
					adminUserID = uid.(string)
				}

				trxRef := fmt.Sprintf("TRX-ADJ-%d", time.Now().UnixNano()/1000000)
				notes := "Penyesuaian kategori / pembayaran oleh panitia"
				if req.TransactionNotes != nil && strings.TrimSpace(*req.TransactionNotes) != "" {
					notes = strings.TrimSpace(*req.TransactionNotes)
				}

				_, errTrx := tx.Exec(`
					INSERT INTO payment_transactions (
						uuid, reference, user_id, tournament_id, registration_id,
						amount, fee_amount, total_amount, payment_method, status,
						paid_at, verified_by, verified_at, rejection_reason, created_at, updated_at
					) VALUES (
						UUID(), ?, ?, ?, ?,
						?, 0, ?, ?, ?,
						NOW(), ?, NOW(), ?, NOW(), NOW()
					)
				`, trxRef, *pInfo.ArcherID, actualEventID, actualParticipantID,
					trxAmount, trxAmount, method, status,
					adminUserID, notes)

				if errTrx != nil {
					fmt.Printf("[ERROR] Failed to record payment adjustment transaction: %v\n", errTrx)
				} else {
					fmt.Printf("[AUDIT] Recorded payment adjustment: %s, amount: %.2f, method: %s, status: %s\n", trxRef, trxAmount, method, status)
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan data peserta", "details": err.Error()})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		if userID != nil {
			utils.LogActivity(db, userID.(string), actualEventID, "participant_updated", "event_participant", actualParticipantID, "Updated participant", c.ClientIP(), c.Request.UserAgent())
		}

		// If payment_status was updated to paid/lunas, send confirmation email asynchronously
		if req.PaymentStatus != nil {
			sVal := strings.ToLower(*req.PaymentStatus)
			if sVal == "paid" || sVal == "lunas" {
				go func(archUUID, evUUID string) {
					var pEmailInfo struct {
						Email     string  `db:"email"`
						FullName  string  `db:"full_name"`
						EventName string  `db:"event_name"`
						Amount    float64 `db:"payment_amount"`
					}
					errP := db.Get(&pEmailInfo, `
						SELECT 
							COALESCE(a.email, '') as email,
							COALESCE(a.full_name, 'Peserta') as full_name,
							COALESCE(e.name, 'Turnamen Archeris') as event_name,
							COALESCE(SUM(tp.payment_amount), 0) as payment_amount
						FROM tournament_participants tp
						JOIN archers a ON tp.archer_id = a.uuid
						JOIN tournaments e ON tp.tournament_id = e.uuid
						WHERE tp.tournament_id = ? AND tp.archer_id = ?
						GROUP BY a.email, a.full_name, e.name
						LIMIT 1
					`, evUUID, archUUID)
					if errP == nil && pEmailInfo.Email != "" {
						var cats []string
						_ = db.Select(&cats, `
							SELECT DISTINCT COALESCE(tc.category_name_custom, rag.name, 'General')
							FROM tournament_participants tp
							LEFT JOIN tournament_categories tc ON tp.category_id = tc.uuid
							LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
							WHERE tp.tournament_id = ? AND tp.archer_id = ?
						`, evUUID, archUUID)
						_ = utils.SendPaymentApprovedEmail(pEmailInfo.Email, pEmailInfo.FullName, pEmailInfo.EventName, pEmailInfo.Amount, cats)
					}
				}(*pInfo.ArcherID, actualEventID)
			}
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
		err := db.Get(&eventTypeCode, "SELECT code FROM ref_tournament_types WHERE uuid = ?", req.EventTypeUUID)
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
		err = db.Get(&eventExists, `SELECT EXISTS(SELECT 1 FROM tournaments WHERE uuid = ?)`, eventID)
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
					SELECT EXISTS(SELECT 1 FROM tournament_categories 
					WHERE tournament_id = ? AND division_uuid = ? AND category_uuid = ? AND tournament_type_uuid = ? AND gender_division_uuid = ?)
				`, eventID, divUUID, catUUID, req.EventTypeUUID, req.GenderDivisionUUID)

				if err != nil || catExists {
					continue
				}

				catEventID := uuid.New().String()
				_, err = db.Exec(`
					INSERT INTO tournament_categories (
						uuid, tournament_id, division_uuid, category_uuid, tournament_type_uuid, gender_division_uuid,
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
		err := db.Get(&eventTypeCode, "SELECT code FROM ref_tournament_types WHERE uuid = ?", req.EventTypeUUID)
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

		// If a custom age group name is provided, persist it to ref_age_groups if not already present
		// so other event organizers can pick it directly in the future.
		if req.CategoryNameCustom != nil && strings.TrimSpace(*req.CategoryNameCustom) != "" {
			trimmedCustom := strings.TrimSpace(*req.CategoryNameCustom)
			cleanCode := strings.ToLower(trimmedCustom)
			cleanCode = strings.ReplaceAll(cleanCode, " ", "-")
			cleanCode = strings.ReplaceAll(cleanCode, "_", "-")
			cleanCode = strings.ReplaceAll(cleanCode, "/", "-")

			var existingAgeUUID string
			err := db.Get(&existingAgeUUID, `SELECT uuid FROM ref_age_groups WHERE LOWER(TRIM(name)) = LOWER(?) OR code = ? LIMIT 1`, trimmedCustom, cleanCode)
			if err == nil && existingAgeUUID != "" {
				req.CategoryUUID = existingAgeUUID
			} else {
				newAgeUUID := uuid.New().String()
				_, err = db.Exec(`INSERT INTO ref_age_groups (uuid, code, name) VALUES (?, ?, ?)`, newAgeUUID, cleanCode, trimmedCustom)
				if err == nil {
					req.CategoryUUID = newAgeUUID
				}
			}
		}

		// Resolve slug to UUID if needed
		var actualEventID string
		err = db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Check if combination already exists
		var catExists bool
		err = db.Get(&catExists, `
			SELECT EXISTS(SELECT 1 FROM tournament_categories 
			WHERE tournament_id = ? AND division_uuid = ? AND category_uuid = ? AND tournament_type_uuid = ? AND gender_division_uuid = ?)
		`, actualEventID, req.DivisionUUID, req.CategoryUUID, req.EventTypeUUID, req.GenderDivisionUUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check category", "details": err.Error()})
			return
		}

		if catExists {
			c.JSON(http.StatusConflict, gin.H{"error": "Kategori sudah ada di event ini"})
			return
		}

		// Gate checking: Team & Mixed Team require corresponding Individual category(ies)
		if eventTypeCode == "team" {
			var indivCount int
			_ = db.Get(&indivCount, `
				SELECT COUNT(*) 
				FROM tournament_categories tc
				JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
				WHERE tc.tournament_id = ? 
				  AND tc.division_uuid = ? 
				  AND tc.category_uuid = ? 
				  AND tc.gender_division_uuid = ? 
				  AND rtt.code = 'individual'
			`, actualEventID, req.DivisionUUID, req.CategoryUUID, req.GenderDivisionUUID)
			if indivCount == 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Kategori Beregu memerlukan kategori Individual yang sesuai terlebih dahulu untuk pencatatan skor kualifikasi.",
				})
				return
			}
		} else if eventTypeCode == "mixed_team" {
			var indivGenderCount int
			_ = db.Get(&indivGenderCount, `
				SELECT COUNT(DISTINCT tc.gender_division_uuid) 
				FROM tournament_categories tc
				JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
				JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
				WHERE tc.tournament_id = ? 
				  AND tc.division_uuid = ? 
				  AND tc.category_uuid = ? 
				  AND rtt.code = 'individual'
				  AND rgd.code IN ('men', 'women', 'male', 'female')
			`, actualEventID, req.DivisionUUID, req.CategoryUUID)
			if indivGenderCount < 2 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Kategori Beregu Campuran (Mixed Team) memerlukan kedua kategori Individual (Putra dan Putri) terlebih dahulu untuk pencatatan skor kualifikasi.",
				})
				return
			}
		}

		status := req.Status
		if status == "" {
			status = "active"
		}

		catEventID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO tournament_categories (
				uuid, tournament_id, division_uuid, category_uuid, category_name_custom, tournament_type_uuid, gender_division_uuid,
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
			err := db.Get(&eventTypeCode, "SELECT code FROM ref_tournament_types WHERE uuid = ?", *req.EventTypeUUID)
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
		err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Check if category exists and belongs to event
		var exists bool
		err = db.Get(&exists, `
			SELECT EXISTS(SELECT 1 FROM tournament_categories 
			WHERE uuid = ? AND tournament_id = ?)
		`, categoryID, actualEventID)

		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
			return
		}

		// Build dynamic update query
		query := "UPDATE tournament_categories SET updated_at = NOW()"
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
			query += ", tournament_type_uuid = ?"
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

		query += " WHERE uuid = ? AND tournament_id = ?"
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
		err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
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
			FROM tournament_categories ec
			JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
			WHERE ec.uuid = ? AND ec.tournament_id = ?
		`, categoryID, actualEventID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
			return
		}

		// Count participants
		var participantCount int
		db.Get(&participantCount, "SELECT COUNT(*) FROM tournament_participants WHERE category_id = ?", categoryID)

		// Count qualification sessions linked
		var sessionCount int
		db.Get(&sessionCount, "SELECT COUNT(*) FROM qualification_session_categories WHERE category_uuid = ?", categoryID)

		// Count qualification scores
		var scoreCount int
		db.Get(&scoreCount, `
			SELECT COUNT(*) FROM qualification_end_scores 
			WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE category_id = ?)
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
		err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// 1. Check if category exists
		var exists bool
		err = db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM tournament_categories WHERE uuid = ? AND tournament_id = ?)", categoryID, actualEventID)
		if err != nil || !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
			return
		}

		// Gate Check 1: Block deletion if there are paid/confirmed participants
		var paidCount int
		_ = db.Get(&paidCount, `
			SELECT COUNT(*) 
			FROM tournament_participants 
			WHERE category_id = ? 
			  AND payment_status IN ('paid', 'lunas')
		`, categoryID)
		if paidCount > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error":      "Kategori tidak dapat dihapus karena telah memiliki peserta dengan status pembayaran lunas. Nonaktifkan status kategori jika tidak lagi menerima pendaftaran.",
				"error_code": "category_has_paid_participants",
				"paid_count": paidCount,
			})
			return
		}

		// Gate Check 2: Block deletion of Individual category if dependent Team / Mixed Team categories exist
		var catType string
		var divUUID, catAgeUUID string
		_ = db.QueryRow(`
			SELECT COALESCE(rtt.code, 'individual'), tc.division_uuid, tc.category_uuid
			FROM tournament_categories tc
			LEFT JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
			WHERE tc.uuid = ?
		`, categoryID).Scan(&catType, &divUUID, &catAgeUUID)

		if catType == "individual" {
			var depTeamCount int
			_ = db.Get(&depTeamCount, `
				SELECT COUNT(*)
				FROM tournament_categories tc
				JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
				WHERE tc.tournament_id = ?
				  AND tc.division_uuid = ?
				  AND tc.category_uuid = ?
				  AND rtt.code IN ('team', 'mixed_team')
				  AND tc.uuid != ?
			`, actualEventID, divUUID, catAgeUUID, categoryID)
			if depTeamCount > 0 {
				c.JSON(http.StatusConflict, gin.H{
					"error":           "Kategori Individual ini tidak dapat dihapus karena masih digunakan sebagai basis skoring oleh kategori Beregu / Beregu Campuran pada event ini.",
					"error_code":      "individual_required_by_team",
					"dependent_teams": depTeamCount,
				})
				return
			}
		}

		// Gate Check 3: Block deletion if qualification scores already exist
		var scoreCount int
		_ = db.Get(&scoreCount, `
			SELECT COUNT(*) 
			FROM qualification_end_scores qes
			JOIN tournament_participants tp ON qes.participant_uuid = tp.uuid
			WHERE tp.category_id = ?
		`, categoryID)
		if scoreCount > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error":       "Kategori tidak dapat dihapus karena pertandingan kualifikasi telah memiliki rekaman skor anak panah.",
				"error_code":  "category_has_scores",
				"score_count": scoreCount,
			})
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
		tx.Exec("DELETE FROM qualification_arrow_scores WHERE end_score_uuid IN (SELECT uuid FROM qualification_end_scores WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE category_id = ?))", categoryID)
		tx.Exec("DELETE FROM qualification_end_scores WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE category_id = ?)", categoryID)
		tx.Exec("DELETE FROM qualification_target_assignments WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE category_id = ?)", categoryID)
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
		tx.Exec("DELETE FROM tournament_participants WHERE category_id = ?", categoryID)

		// F. Delete the Category
		_, err = tx.Exec("DELETE FROM tournament_categories WHERE uuid = ? AND tournament_id = ?", categoryID, actualEventID)
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

		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

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
		err = db.Select(&images, `
			SELECT uuid, tournament_id as event_id, url, caption, alt_text, display_order, is_primary, created_at
			FROM tournament_images
			WHERE tournament_id = ?
			ORDER BY display_order, created_at
		`, eventUUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil gambar event", "details": err.Error()})
			return
		}

		for i := range images {
			images[i].URL = utils.MaskMediaURL(images[i].URL)
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

		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database", "details": err.Error()})
			return
		}
		defer tx.Rollback()

		// Delete existing images
		_, err = tx.Exec("DELETE FROM tournament_images WHERE tournament_id = ?", eventUUID)
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
			cleanURL := utils.ExtractFilename(img.URL)
			_, err = tx.Exec(`
				INSERT INTO tournament_images (uuid, tournament_id, url, caption, alt_text, display_order, is_primary)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, imageID, eventUUID, cleanURL, img.Caption, img.AltText, displayOrder, img.IsPrimary)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan gambar event", "details": err.Error()})
				return
			}

			// Link media record if exists
			_, _ = tx.Exec("UPDATE media SET tournament_id = ? WHERE (url = ? OR url LIKE ?) AND tournament_id IS NULL", eventUUID, cleanURL, "%"+cleanURL)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan gambar event", "details": err.Error()})
			return
		}

		// Log activity
		utils.LogActivity(db, userID.(string), eventUUID, "event_images_updated", "event", eventUUID, fmt.Sprintf("Updated %d event images", len(req.Images)), c.ClientIP(), c.Request.UserAgent())

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
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?", eventIDParam, eventIDParam)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		categoryID := c.Query("category_id")

		query := `
			SELECT t.uuid, t.team_name, '' as country_code, '' as country_name, t.status, 
			       COALESCE(t.total_score, 0) as total_score, COALESCE(t.total_x_count, 0) as total_x_count, t.created_at,
			       COUNT(tm.uuid) as member_count,
				   GROUP_CONCAT(COALESCE(a.full_name, 'Unknown') ORDER BY tm.member_order SEPARATOR ', ') as member_names,
				   GROUP_CONCAT(COALESCE(tm.total_score, 0) ORDER BY tm.member_order SEPARATOR ', ') as member_scores
			FROM teams t
			LEFT JOIN team_members tm ON t.uuid = tm.team_id
			LEFT JOIN tournament_participants ep ON tm.participant_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE t.tournament_id = ?
		`
		args := []interface{}{eventUUID}

		if categoryID != "" {
			query += " AND (t.category_id = ? OR t.event_id = ?)"
			args = append(args, categoryID, categoryID)
		}

		query += " GROUP BY t.uuid, t.team_name, t.status, t.total_score, t.total_x_count, t.created_at ORDER BY t.team_rank ASC, t.total_score DESC, t.total_x_count DESC"

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

// GetMyEvents returns tournaments managed by the authenticated user
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

		// Base query: get tournaments where organizer_id is the current user
		whereClause := "WHERE t.organizer_id = ?"
		args := []interface{}{userID}

		if status != "" && status != "all" {
			if status == "published" {
				whereClause += ` AND (t.status = 'published' OR t.status = 'active')`
			} else if status == "completed" {
				whereClause += ` AND (t.status = 'completed' OR t.status = 'finished')`
			} else {
				whereClause += ` AND t.status = ?`
				args = append(args, status)
			}
		}

		if search != "" {
			whereClause += ` AND (t.name LIKE ? OR t.code LIKE ? OR t.location LIKE ?)`
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		// Get total count
		var total int
		err := db.Get(&total, `SELECT COUNT(*) FROM tournaments t `+whereClause, args...)
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
			FROM tournaments t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, email, slug, avatar_url FROM organizers
				UNION ALL
				SELECT uuid as id, name as full_name, NULL as email, slug, logo_url as avatar_url FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN tournament_participants tp ON t.uuid = tp.tournament_id
			LEFT JOIN tournament_categories te ON t.uuid = te.tournament_id
			%s
			GROUP BY t.uuid
			ORDER BY %s %s
			LIMIT ? OFFSET ?
		`, whereClause, dbSortField, order)
		args = append(args, limit, offset)

		var tournaments []models.EventWithDetails
		err = db.Select(&tournaments, query, args...)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"tournaments": []interface{}{},
				"total":  0,
			})
			return
		}

		// Mask URLs
		for i := range tournaments {
			if tournaments[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].BannerURL)
				tournaments[i].BannerURL = &masked
			}
			if tournaments[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].LogoURL)
				tournaments[i].LogoURL = &masked
			}
			if tournaments[i].TechnicalGuidebookURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].TechnicalGuidebookURL)
				tournaments[i].TechnicalGuidebookURL = &masked
			}
			if tournaments[i].OrganizerAvatarURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].OrganizerAvatarURL)
				tournaments[i].OrganizerAvatarURL = &masked
			}
		}

		meta := utils.CalculatePagination(total, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{
			"data":   tournaments,
			"tournaments": tournaments,
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
			EventID       string  `db:"tournament_id"`
			ArcherID      string  `db:"archer_id"`
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
				ep.tournament_id,
				ep.archer_id,
				a.full_name,
				a.email,
				c.name as club_name,
				COALESCE(d.name, '') as division_name,
				COALESCE(ec.category_name_custom, ag.name, '') as category_name,
				e.name as event_name,
				COALESCE(ep.payment_status, 'pending') as payment_status
			FROM tournament_participants ep
			INNER JOIN archers a ON ep.archer_id = a.uuid
			INNER JOIN tournaments e ON ep.tournament_id = e.uuid
			LEFT JOIN clubs c ON a.club_id = c.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
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

		// Verify caller access: must be organizer or root/admin
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		userRole, _ := c.Get("user_role")
		var organizerID string
		_ = db.Get(&organizerID, "SELECT organizer_id FROM tournaments WHERE uuid = ?", participant.EventID)
		if organizerID != userID.(string) && userRole != "root" && userRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya penyelenggara turnamen ini yang dapat melakukan check-in"})
			return
		}

		// Update last_reregistration_at for all registrations of this archer in this event
		_, err = db.Exec(`
			UPDATE tournament_participants 
			SET last_reregistration_at = NOW()
			WHERE tournament_id = ? AND archer_id = ?
		`, participant.EventID, participant.ArcherID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui pendaftaran", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Check-in kehadiran atlet berhasil dicatat",
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

// BatchCheckinParticipants allows organizer to check in multiple participants at once
func BatchCheckinParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		eventID := c.Param("id")
		var actualEventID string
		var organizerID string
		err := db.QueryRow("SELECT uuid, organizer_id FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID).Scan(&actualEventID, &organizerID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		userRole, _ := c.Get("user_role")
		if organizerID != userID.(string) && userRole != "root" && userRole != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Hanya penyelenggara turnamen yang dapat melakukan check-in peserta"})
			return
		}

		var req struct {
			ParticipantIDs []string `json:"participant_ids"`
			ClubID         *string  `json:"club_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid"})
			return
		}

		var targetIDs []string
		if len(req.ParticipantIDs) > 0 {
			targetIDs = req.ParticipantIDs
		} else if req.ClubID != nil && *req.ClubID != "" {
			err = db.Select(&targetIDs, `
				SELECT tp.uuid 
				FROM tournament_participants tp
				INNER JOIN archers a ON tp.archer_id = a.uuid
				WHERE tp.tournament_id = ? AND a.club_id = ? AND tp.payment_status IN ('paid', 'lunas')
			`, actualEventID, *req.ClubID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data peserta klub"})
				return
			}
		}

		if len(targetIDs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada peserta yang dipilih atau memenuhi syarat check-in"})
			return
		}

		qUp, argsUp, errIn := sqlx.In(`
			UPDATE tournament_participants 
			SET last_reregistration_at = NOW(), updated_at = NOW()
			WHERE tournament_id = ? 
			  AND payment_status IN ('paid', 'lunas')
			  AND uuid IN (?)
		`, actualEventID, targetIDs)
		if errIn != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyusun query check-in: " + errIn.Error()})
			return
		}
		qUp = db.Rebind(qUp)
		res, err := db.Exec(qUp, argsUp...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal melakukan check-in peserta: " + err.Error()})
			return
		}

		rowsAffected, _ := res.RowsAffected()

		c.JSON(http.StatusOK, gin.H{
			"message":       fmt.Sprintf("Check-in kehadiran berhasil untuk %d peserta", rowsAffected),
			"checked_count": rowsAffected,
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
		err := db.Get(&event, `SELECT uuid, name FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		type Participant struct {
			AthleteCode        *string `db:"athlete_code"`
			FullName           string  `db:"full_name"`
			Email              string  `db:"email"`
			ClubName           string  `db:"club_name"`
			City               string  `db:"city"`
			PaymentStatus      string  `db:"payment_status"`
			RegistrationSource string  `db:"registration_source"`
			RegistrationDate   string  `db:"registration_date"`
			TargetNames        string  `db:"target_names"`
			Categories         string  `db:"categories"`
			TotalScore         int     `db:"total_score"`
			TotalX             int     `db:"total_x"`
		}

		var participants []Participant
		query := `
			SELECT 
				a.id as athlete_code,
				a.full_name,
				COALESCE(a.email, '') as email,
				COALESCE(cl.name, '') as club_name,
				'' as city,
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
			FROM tournament_participants tp
			JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories te ON tp.category_id = te.uuid
			LEFT JOIN ref_bow_types d ON te.division_uuid = d.uuid
			LEFT JOIN ref_age_groups c ON te.category_uuid = c.uuid
			LEFT JOIN ref_tournament_types et ON te.tournament_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON te.gender_division_uuid = gd.uuid
			LEFT JOIN (
				SELECT participant_uuid, SUM(total_score_end) as total_score, SUM(x_count_end) as total_x
				FROM qualification_end_scores
				GROUP BY participant_uuid
			) scores ON tp.uuid = scores.participant_uuid
			WHERE tp.tournament_id = ?
			GROUP BY a.uuid, a.id, a.full_name, a.email, cl.name
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
				func(s *string) string {
					if s == nil {
						return ""
					}
					return *s
				}(p.AthleteCode),
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
		err := db.Get(&actualEventID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Ownership check: only the organizer who owns this event may reset it
		var ownerCount int
		db.Get(&ownerCount, `SELECT COUNT(*) FROM tournaments WHERE uuid = ? AND organizer_id = ?`, actualEventID, userID.(string))
		if ownerCount == 0 {
			// Also allow root admin
			userTypeCtx, _ := c.Get("user_type")
			if userTypeCtx != "root" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Hanya penyelenggara event ini yang dapat mereset data"})
				return
			}
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
					WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
				)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus skor anak panah", "details": err.Error()})
				return
			}

			// 2. Delete end scores
			_, err = tx.Exec(`
				DELETE FROM qualification_end_scores 
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus skor kualifikasi", "details": err.Error()})
				return
			}

			// 3. Delete target assignments
			_, err = tx.Exec(`
				DELETE FROM qualification_target_assignments 
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus penugasan target kualifikasi", "details": err.Error()})
				return
			}

			// 4. Reset target names and back numbers in tournament_participants
			_, err = tx.Exec(`
				UPDATE tournament_participants 
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
				WHERE session_uuid IN (SELECT uuid FROM qualification_sessions WHERE tournament_uuid = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kategori sesi kualifikasi", "details": err.Error()})
				return
			}

			// 6. Delete target board qualification verification codes
			_, err = tx.Exec(`
				DELETE FROM target_board_qualification 
				WHERE session_uuid IN (SELECT uuid FROM qualification_sessions WHERE tournament_uuid = ?)
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
				WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE tournament_uuid = ?)
			`, actualEventID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus match eliminasi", "details": err.Error()})
				return
			}

			// 2. Delete elimination entries
			_, err = tx.Exec(`
				DELETE FROM elimination_entries 
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
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
					WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
				)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_end_scores 
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_target_assignments 
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			`, actualEventID)

			// Wipe elimination due to foreign keys
			tx.Exec(`
				DELETE FROM elimination_matches 
				WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE tournament_uuid = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM elimination_entries 
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			`, actualEventID)
			tx.Exec(`
				UPDATE elimination_brackets 
				SET status = 'draft', generated_at = NULL 
				WHERE event_uuid = ?
			`, actualEventID)

			// Delete all event participants
			_, err = tx.Exec(`DELETE FROM tournament_participants WHERE tournament_id = ?`, actualEventID)
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
					WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
				)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_end_scores 
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_target_assignments 
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			`, actualEventID)

			// Wipe qualification sessions & associated links
			tx.Exec(`
				DELETE FROM qualification_session_categories 
				WHERE session_uuid IN (SELECT uuid FROM qualification_sessions WHERE tournament_uuid = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM target_board_qualification 
				WHERE session_uuid IN (SELECT uuid FROM qualification_sessions WHERE tournament_uuid = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM qualification_sessions 
				WHERE event_uuid = ?
			`, actualEventID)

			// Wipe elimination
			tx.Exec(`
				DELETE FROM elimination_matches 
				WHERE bracket_uuid IN (SELECT uuid FROM elimination_brackets WHERE tournament_uuid = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM elimination_entries 
				WHERE participant_uuid IN (SELECT uuid FROM tournament_participants WHERE tournament_id = ?)
			`, actualEventID)
			tx.Exec(`
				DELETE FROM elimination_brackets 
				WHERE event_uuid = ?
			`, actualEventID)

			// Wipe participants
			_, err = tx.Exec(`DELETE FROM tournament_participants WHERE tournament_id = ?`, actualEventID)
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
		err := db.Get(&userEmail, `SELECT email FROM organizers WHERE uuid = ?`, userID.(string))
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
		err = utils.SendEventResetOTPEmail(userEmail, "Penyelenggara Event", otpCode, 15)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim email verifikasi", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Kode verifikasi telah dikirim ke email Anda"})
	}
}

// ImportParticipantsCSV imports participants into an event from a CSV file
func ImportParticipantsCSV(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID event tidak boleh kosong"})
			return
		}

		file, _, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca file CSV. Pastikan file dikirim dalam field 'file'"})
			return
		}
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format file CSV tidak valid", "details": err.Error()})
			return
		}

		if len(records) <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File CSV kosong atau hanya berisi baris header"})
			return
		}

		// Read header
		header := records[0]
		colIdx := make(map[string]int)
		for i, h := range header {
			colIdx[strings.TrimSpace(strings.ToLower(h))] = i
		}

		// Required columns check
		requiredCols := []string{"full_name"}
		for _, col := range requiredCols {
			if _, exists := colIdx[col]; !exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Kolom wajib '%s' tidak ditemukan di CSV", col)})
				return
			}
		}

		// Fetch existing categories for mapping
		type EventCat struct {
			UUID               string  `db:"uuid"`
			CustomName         *string `db:"category_name_custom"`
			DivisionName       string  `db:"division_name"`
			CategoryName       string  `db:"category_name"`
			GenderDivisionName string  `db:"gender_division_name"`
			EventType          string  `db:"event_type_name"`
		}
		var categories []EventCat
		err = db.Select(&categories, `
			SELECT 
				ec.uuid,
				ec.category_name_custom,
				COALESCE(d.name, '') as division_name,
				COALESCE(ag.name, '') as category_name,
				COALESCE(gd.name, '') as gender_division_name,
				COALESCE(et.name, '') as event_type_name
			FROM tournament_categories ec
			LEFT JOIN ref_bow_types d ON ec.division_uuid = d.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			LEFT JOIN ref_gender_divisions gd ON ec.gender_division_uuid = gd.uuid
			LEFT JOIN ref_tournament_types et ON ec.tournament_type_uuid = et.uuid
			WHERE ec.tournament_id = ?
		`, eventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kategori event", "details": err.Error()})
			return
		}

		catMap := make(map[string]string)
		for _, cat := range categories {
			if cat.CustomName != nil && strings.TrimSpace(*cat.CustomName) != "" {
				catMap[strings.ToLower(strings.TrimSpace(*cat.CustomName))] = cat.UUID
			}
			composed := strings.ToLower(fmt.Sprintf("%s - %s %s %s", cat.DivisionName, cat.CategoryName, cat.EventType, cat.GenderDivisionName))
			composed = strings.Join(strings.Fields(composed), " ")
			if composed != "" {
				catMap[composed] = cat.UUID
			}
		}

		// Check Tournament Level Participant Quota (Anti-Bypass Protection)
		var tourQuota struct {
			UUID                  string  `db:"uuid"`
			QuotaType             *string `db:"quota_type"`
			QuotaMaxParticipants *int    `db:"quota_max_participants"`
		}
		if err := db.Get(&tourQuota, `SELECT uuid, quota_type, quota_max_participants FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID); err == nil {
			var maxTourParticipants *int = tourQuota.QuotaMaxParticipants
			if maxTourParticipants == nil && tourQuota.QuotaType != nil {
				switch strings.ToLower(*tourQuota.QuotaType) {
				case "free":
					fifty := 50
					maxTourParticipants = &fifty
				case "standard":
					twoHundred := 200
					maxTourParticipants = &twoHundred
				case "elite":
					maxTourParticipants = nil
				}
			}

			if maxTourParticipants != nil && *maxTourParticipants > 0 {
				var currentUniqueAthletes int
				_ = db.Get(&currentUniqueAthletes, `SELECT COUNT(DISTINCT archer_id) FROM tournament_participants WHERE tournament_id = ? AND status != 'cancelled'`, tourQuota.UUID)

				remainingQuota := *maxTourParticipants - currentUniqueAthletes
				if remainingQuota <= 0 {
					c.JSON(http.StatusForbidden, gin.H{
						"error": fmt.Sprintf("Batas kapasitas peserta turnamen ini telah mencapai batas paket (%d peserta).", *maxTourParticipants),
						"code":  "QUOTA_PARTICIPANT_EXCEEDED",
					})
					return
				}
			}
		}

		var importedCount int
		var skippedCount int
		var errorsList []string

		for rowIdx, record := range records[1:] {
			rowNum := rowIdx + 2
			getValue := func(col string) string {
				if idx, ok := colIdx[col]; ok && idx < len(record) {
					return strings.TrimSpace(record[idx])
				}
				return ""
			}

			fullName := getValue("full_name")
			if fullName == "" {
				errorsList = append(errorsList, fmt.Sprintf("Baris %d: Nama lengkap kosong", rowNum))
				skippedCount++
				continue
			}

			email := getValue("email")
			phone := getValue("phone")
			gender := strings.ToUpper(getValue("gender"))
			if gender != "M" && gender != "F" {
				if strings.HasPrefix(strings.ToLower(gender), "l") || strings.HasPrefix(strings.ToLower(gender), "m") {
					gender = "M"
				} else if strings.HasPrefix(strings.ToLower(gender), "p") || strings.HasPrefix(strings.ToLower(gender), "f") {
					gender = "F"
				} else {
					gender = "M"
				}
			}

			bowType := strings.ToLower(getValue("bow_type"))
			if bowType == "" {
				bowType = "recurve"
			}
			clubName := getValue("club_name")
			categoryInput := strings.ToLower(getValue("category_name"))

			// Determine Category ID
			var targetCatID string
			if categoryInput != "" {
				if catID, ok := catMap[categoryInput]; ok {
					targetCatID = catID
				} else {
					// Fallback search substring match
					for k, v := range catMap {
						if strings.Contains(k, categoryInput) || strings.Contains(categoryInput, k) {
							targetCatID = v
							break
						}
					}
				}
			}

			if targetCatID == "" && len(categories) > 0 {
				targetCatID = categories[0].UUID
			}

			if targetCatID == "" {
				errorsList = append(errorsList, fmt.Sprintf("Baris %d: Kategori '%s' tidak ditemukan", rowNum, getValue("category_name")))
				skippedCount++
				continue
			}

			// Find or create Archer
			var archerID string
			if email != "" {
				_ = db.Get(&archerID, "SELECT uuid FROM archers WHERE email = ?", email)
			}
			if archerID == "" && phone != "" {
				_ = db.Get(&archerID, "SELECT uuid FROM archers WHERE phone = ?", phone)
			}
			if archerID == "" {
				_ = db.Get(&archerID, "SELECT uuid FROM archers WHERE LOWER(full_name) = LOWER(?)", fullName)
			}

			if archerID == "" {
				// Create new archer
				newUUID := uuid.New().String()
				var clubUUID sql.NullString
				if clubName != "" {
					cleanClubName := strings.TrimSpace(clubName)
					var cID string
					if err := db.Get(&cID, "SELECT uuid FROM clubs WHERE LOWER(TRIM(name)) = LOWER(TRIM(?))", cleanClubName); err == nil {
						clubUUID = sql.NullString{String: cID, Valid: true}
					} else {
						newClubUUID := uuid.New().String()
						clubSlug := utils.CleanSlug(cleanClubName)
						if clubSlug == "" {
							clubSlug = "club-" + uuid.New().String()[:6]
						}
						_, err := db.Exec("INSERT INTO clubs (uuid, name, slug, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())", newClubUUID, cleanClubName, clubSlug)
						if err == nil {
							clubUUID = sql.NullString{String: newClubUUID, Valid: true}
						}
					}
				}

				username := strings.ToLower(strings.ReplaceAll(fullName, " ", "-"))
				username = strings.Map(func(r rune) rune {
					if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
						return r
					}
					return -1
				}, username)

				_, err := db.Exec(`
					INSERT INTO archers (uuid, full_name, username, email, phone, gender, bow_type, club_id, created_at, updated_at)
					VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, NOW(), NOW())
				`, newUUID, fullName, username, email, phone, gender, bowType, clubUUID)

				if err != nil {
					// Try without email/phone if constraint error
					_, err = db.Exec(`
						INSERT INTO archers (uuid, full_name, username, gender, bow_type, club_id, created_at, updated_at)
						VALUES (?, ?, ?, ?, ?, ?, NOW(), NOW())
					`, newUUID, fullName, fmt.Sprintf("%s-%d", username, time.Now().Unix()%10000), gender, bowType, clubUUID)
				}

				if err == nil {
					archerID = newUUID
				} else {
					errorsList = append(errorsList, fmt.Sprintf("Baris %d: Gagal membuat data pemanah (%s)", rowNum, err.Error()))
					skippedCount++
					continue
				}
			}

			// Check if already registered for this event
			var existingParticipant string
			err = db.Get(&existingParticipant, "SELECT uuid FROM tournament_participants WHERE tournament_id = ? AND archer_id = ?", eventID, archerID)
			var participantUUID string
			if err == sql.ErrNoRows {
				participantUUID = uuid.New().String()
				paymentStatus := strings.ToLower(getValue("payment_status"))
				if paymentStatus == "" {
					paymentStatus = "paid"
				}
				amountStr := getValue("payment_amount")
				amount, _ := strconv.ParseFloat(amountStr, 64)

				_, err = db.Exec(`
					INSERT INTO tournament_participants (uuid, tournament_id, archer_id, payment_status, payment_amount, registration_source, created_at, updated_at)
					VALUES (?, ?, ?, ?, ?, 'organizer_added', NOW(), NOW())
				`, participantUUID, eventID, archerID, paymentStatus, amount)

				if err != nil {
					errorsList = append(errorsList, fmt.Sprintf("Baris %d: Gagal mendaftarkan peserta (%s)", rowNum, err.Error()))
					skippedCount++
					continue
				}
			} else if err == nil {
				participantUUID = existingParticipant
			} else {
				errorsList = append(errorsList, fmt.Sprintf("Baris %d: Gagal mengecek data peserta (%s)", rowNum, err.Error()))
				skippedCount++
				continue
			}

			// Assign category to participant
			var catCount int
			_ = db.Get(&catCount, "SELECT COUNT(*) FROM event_participant_categories WHERE participant_id = ? AND event_category_id = ?", participantUUID, targetCatID)
			if catCount == 0 {
				_, _ = db.Exec(`
					INSERT INTO event_participant_categories (uuid, participant_id, event_category_id, created_at)
					VALUES (?, ?, ?, NOW())
				`, uuid.New().String(), participantUUID, targetCatID)
			}

			importedCount++
		}

		c.JSON(http.StatusOK, gin.H{
			"message":  fmt.Sprintf("Berhasil mengimpor %d peserta (%d terlewati)", importedCount, skippedCount),
			"imported": importedCount,
			"skipped":  skippedCount,
			"errors":   errorsList,
		})
	}
}


