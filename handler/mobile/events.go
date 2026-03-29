package mobile

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileListEvents godoc
func MobileListEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := c.Query("search")

		whereClause := "WHERE t.status != 'draft'"
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

// MobileArcherGetEventDetail godoc
func MobileArcherGetEventDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetString("user_id")

		// 1. Get Event Detail (same query as public detail)
		query := `
			SELECT 
				t.uuid, t.slug, t.name, COALESCE(t.location, '') as location, 
				COALESCE(t.venue, '') as venue, t.city, t.province,
				COALESCE(t.start_date, '') as start_date, COALESCE(t.end_date, '') as end_date, 
				t.logo_url, t.banner_url, t.description, t.rules, t.technical_guidebook_url,
				COALESCE(u.full_name, '') as organizer_name,
				COALESCE(u.avatar_url, '') as organizer_avatar_url,
				COALESCE(u.slug, '') as organizer_slug,
				COALESCE(active_target_stats.participant_count, 0) as participant_count,
				t.organizer_id, t.status, t.registration_deadline
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, avatar_url, slug FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, logo_url as avatar_url, slug FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN (
				SELECT event_id, COUNT(*) as participant_count
				FROM event_participants
				GROUP BY event_id
			) active_target_stats ON t.uuid = active_target_stats.event_id
			WHERE t.uuid = ? OR t.slug = ?
			LIMIT 1
		`

		var event models.EventWithDetails
		err := db.Get(&event, query, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		if event.BannerURL != nil { *event.BannerURL = utils.MaskMediaURL(*event.BannerURL) }
		if event.LogoURL != nil { *event.LogoURL = utils.MaskMediaURL(*event.LogoURL) }
		if event.TechnicalGuidebookURL != nil { *event.TechnicalGuidebookURL = utils.MaskMediaURL(*event.TechnicalGuidebookURL) }
		if event.OrganizerAvatarURL != nil { *event.OrganizerAvatarURL = utils.MaskMediaURL(*event.OrganizerAvatarURL) }

		utils.PopulateEventDetailExtras(db, &event)

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

// MobileGetEventParticipants returns only the participant list for an event
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

// MobileGetEventResults returns only the result list for an event
func MobileGetEventResults(db *sqlx.DB) gin.HandlerFunc {
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
			"results":       event.Results,
			"page_settings": event.PageSettings, // Includes the manual PDF links
		})
	}
}

// MobileGetEventSchedule returns only the schedule for an event
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


