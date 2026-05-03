package mobile

import (
	"archeryhub-api/models"
	"fmt"
	"archeryhub-api/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileListEvents handles listing events for mobile
// @Summary List Mobile Events
// @Description Get a list of active or past events optimized for mobile
// @Tags Mobile - Events
// @Produce json
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Param search query string false "Search by name or location"
// @Param history query bool false "Filter past events"
// @Success 200 {object} MobileEventsResponse
// @Router /mobile/events [get]
func MobileListEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := c.Query("search")

		whereClause := "WHERE t.status != 'draft'"
		
		// If path is /events/history or ?history=true is passed
		if c.Request.URL.Path == "/api/v1/mobile/events/history" || c.Query("history") == "true" {
			whereClause += " AND t.end_date < NOW()"
		}

		args := []interface{}{}

		if search != "" {
			whereClause += ` AND (t.name LIKE ? OR t.location LIKE ?)`
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm)
		}

		query := `
			SELECT 
				t.uuid, t.slug, t.name, COALESCE(t.location, '') as location, 
				COALESCE(t.start_date, '') as start_date, COALESCE(t.end_date, '') as end_date, 
				t.logo_url, t.banner_url,
				COALESCE(u.full_name, '') as organizer_name,
				u.avatar_url as organizer_avatar_url,
				COUNT(DISTINCT tp.archer_id) as participant_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, avatar_url FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, logo_url as avatar_url FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.uuid = tp.event_id
			` + whereClause + `
			GROUP BY t.uuid, t.slug, u.full_name, u.avatar_url
			ORDER BY t.start_date DESC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)

		var events []MobileEvent
		err := db.Select(&events, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event", "details": err.Error()})
			return
		}

		for i := range events {
			if events[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*events[i].LogoURL)
				events[i].LogoURL = &masked
			}
			if events[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*events[i].BannerURL)
				events[i].BannerURL = &masked
			}
			if events[i].OrganizerAvatarURL != nil {
				masked := utils.MaskMediaURL(*events[i].OrganizerAvatarURL)
				events[i].OrganizerAvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"events":      events,
			"total_count": len(events), // Simple count for now, could be improved with separate COUNT query
		})
	}
}

// MobileArcherGetEventDetail returns event detail with archer's registration status
// @Summary Get Event Detail (Archer)
// @Description Get event details including registration status for the authenticated archer
// @Tags Mobile - Archer
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event Slug or UUID"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/archer/events/{id}/detail [get]
func MobileArcherGetEventDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetString("user_id")

		// 1. Get Event Detail (same query as public detail)
		query := `
			SELECT 
				t.uuid, t.slug, t.name, t.venue, t.gmaps_link, t.location, t.address, t.city, t.location_type,
				t.start_date, t.end_date, t.registration_deadline,
				t.logo_url, t.banner_url, t.description, t.technical_guidebook_url,
				COALESCE(u.full_name, '') as organizer_name,
				COALESCE(u.avatar_url, '') as organizer_avatar_url,
				COALESCE(u.slug, '') as organizer_slug,
				COALESCE(u.phone, '') as organizer_phone,
				COALESCE(active_target_stats.participant_count, 0) as participant_count,
				t.organizer_id
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, avatar_url, slug, whatsapp_no as phone FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, logo_url as avatar_url, slug, phone FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN (
				SELECT event_id, COUNT(*) as participant_count
				FROM event_participants
				GROUP BY event_id
			) active_target_stats ON t.uuid = active_target_stats.event_id
			WHERE t.uuid = ? OR t.slug = ?
			LIMIT 1
		`

		var event MobileEventDetail
		err := db.Get(&event, query, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		if event.BannerURL != nil { *event.BannerURL = utils.MaskMediaURL(*event.BannerURL) }
		if event.LogoURL != nil { *event.LogoURL = utils.MaskMediaURL(*event.LogoURL) }
		if event.TechnicalGuidebookURL != nil { *event.TechnicalGuidebookURL = utils.MaskMediaURL(*event.TechnicalGuidebookURL) }
		if event.OrganizerAvatarURL != nil { *event.OrganizerAvatarURL = utils.MaskMediaURL(*event.OrganizerAvatarURL) }

		// Manual populate nested objects for mobile model consistency
		event.LocationDetail = models.EventLocationDetail{
			Venue:        event.Venue,
			Address:      event.Address,
			GmapLink:     event.GmapLink,
			Location:     event.Location,
			City:         event.City,
			LocationType: event.LocationType,
		}

		// 2. Get Archer Registration Status
		var registration struct {
			UUID          string  `db:"uuid" json:"id"`
			PaymentStatus string  `db:"payment_status" json:"payment_status"`
			TargetName    *string `db:"target_name" json:"target_name"`
			PaymentAmount float64 `db:"payment_amount" json:"payment_amount"`
		}
		
		isRegistered := false
		err = db.Get(&registration, `
			SELECT uuid, payment_status, target_name, payment_amount
			FROM event_participants
			WHERE event_id = ? AND archer_id = ?
			LIMIT 1
		`, event.UUID, userID)
		if err == nil {
			isRegistered = true
		}

		c.JSON(http.StatusOK, gin.H{
			"event":         event,
			"is_registered": isRegistered,
			"registration":  registration,
		})
	}
}

// MobileGetEventDetail returns core event information
// @Summary Get Mobile Event Detail
// @Description Get summary and location details for a specific event
// @Tags Mobile - Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventDetail
// @Failure 404 {object} map[string]interface{}
// @Router /mobile/events/{slug} [get]
func MobileGetEventDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")

		query := `
			SELECT
				t.uuid, t.slug, t.name, t.venue, t.gmaps_link, t.location, t.address, t.city, t.location_type,
				t.start_date, t.end_date, t.registration_deadline,
				t.logo_url, t.banner_url, t.description, t.technical_guidebook_url,
				COALESCE(u.full_name, '') as organizer_name,
				COALESCE(u.avatar_url, '') as organizer_avatar_url,
				COALESCE(u.slug, '') as organizer_slug,
				COALESCE(u.phone, '') as organizer_phone,
				COALESCE(active_target_stats.participant_count, 0) as participant_count,
				t.organizer_id
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, avatar_url, slug, whatsapp_no as phone FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, logo_url as avatar_url, slug, phone FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN (
				SELECT event_id, COUNT(*) as participant_count
				FROM event_participants
				GROUP BY event_id
			) active_target_stats ON t.uuid = active_target_stats.event_id
			WHERE t.uuid = ? OR t.slug = ?
			LIMIT 1
		`

		var event MobileEventDetail
		err := db.Get(&event, query, id, id)
		if err != nil {
			fmt.Printf("[MobileGetEventDetail] Database error: %v\n", err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan atau terjadi kesalahan data"})
			return
		}

		if event.BannerURL != nil { *event.BannerURL = utils.MaskMediaURL(*event.BannerURL) }
		if event.LogoURL != nil { *event.LogoURL = utils.MaskMediaURL(*event.LogoURL) }
		if event.TechnicalGuidebookURL != nil { *event.TechnicalGuidebookURL = utils.MaskMediaURL(*event.TechnicalGuidebookURL) }
		if event.OrganizerAvatarURL != nil { *event.OrganizerAvatarURL = utils.MaskMediaURL(*event.OrganizerAvatarURL) }

		// Manual populate nested objects for mobile model consistency
		event.LocationDetail = models.EventLocationDetail{
			Venue:        event.Venue,
			Address:      event.Address,
			GmapLink:     event.GmapLink,
			Location:     event.Location,
			City:         event.City,
			LocationType: event.LocationType,
		}

		c.JSON(http.StatusOK, event)
	}
}

// MobileGetEventParticipants returns only the participant list for an event
// @Summary Get Event Participants
// @Description Get the list of registered archers for an event
// @Tags Mobile - Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} map[string][]models.EventParticipantPreview
// @Router /mobile/events/{slug}/participants [get]
func MobileGetEventParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var eventID string
		_ = db.Get(&eventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ?", id, id)
		if eventID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		event := &models.EventWithDetails{Event: models.Event{UUID: eventID}}
		utils.PopulateEventDetailExtras(db, event)

		c.JSON(http.StatusOK, gin.H{
			"participants": event.Participants,
		})
	}
}

// MobileGetEventSchedule returns only the schedule for an event
// @Summary Get Event Schedule
// @Description Get the daily schedule and rundown for an event
// @Tags Mobile - Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} map[string][]models.EventSchedule
// @Router /mobile/events/{slug}/schedule [get]
func MobileGetEventSchedule(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var eventID string
		_ = db.Get(&eventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ?", id, id)
		if eventID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		event := &models.EventWithDetails{Event: models.Event{UUID: eventID}}
		utils.PopulateEventDetailExtras(db, event)

		c.JSON(http.StatusOK, gin.H{
			"schedules": event.Schedules,
		})
	}
}

// MobileGetEventCategories returns only the competition categories for an event
// @Summary Get Event Categories
// @Description Get list of divisions and age groups in this event
// @Tags Mobile - Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} map[string][]models.EventCompetitionCategory
// @Router /mobile/events/{slug}/categories [get]
func MobileGetEventCategories(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var eventID string
		_ = db.Get(&eventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ?", id, id)
		if eventID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		event := &models.EventWithDetails{Event: models.Event{UUID: eventID}}
		utils.PopulateEventDetailExtras(db, event)

		c.JSON(http.StatusOK, gin.H{
			"competition_categories": event.CompetitionCategories,
		})
	}
}

// MobileGetEventGallery returns only the gallery images for an event
// @Summary Get Event Gallery
// @Description Get event gallery and documentation images
// @Tags Mobile - Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} map[string][]models.EventImage
// @Router /mobile/events/{slug}/gallery [get]
func MobileGetEventGallery(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var eventID string
		_ = db.Get(&eventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ?", id, id)
		if eventID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		event := &models.EventWithDetails{Event: models.Event{UUID: eventID}}
		utils.PopulateEventDetailExtras(db, event)

		c.JSON(http.StatusOK, gin.H{
			"gallery": event.Gallery,
		})
	}
}


