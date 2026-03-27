package mobile

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileListEvents godoc
// @Summary      List mobile events
// @Description  Get published events for mobile with pagination and search
// @Tags         Mobile - Events
// @Produce      json
// @Param        limit   query     int     false  "Limit"   default(20)
// @Param        offset  query     int     false  "Offset"  default(0)
// @Param        search  query     string  false  "Search term (event name/location)"
// @Success      200     {object}  MobileEventsResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /mobile/events [get]
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
				t.uuid, t.name, COALESCE(t.location, '') as location, 
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
			GROUP BY t.uuid, u.full_name, u.avatar_url
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

// MobileGetEventDetail godoc
// @Summary      Get mobile event detail
// @Description  Get event detail (by slug or UUID) with the same response body as /events/:id
// @Tags         Mobile - Events
// @Produce      json
// @Param        slug  path      string  true  "Event slug or UUID"
// @Success      200   {object}  models.EventWithDetails
// @Failure      404   {object}  ErrorResponse
// @Router       /mobile/events/{slug} [get]
func MobileGetEventDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")

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

		var event models.EventWithDetails
		err := db.Get(&event, query, id, id)
		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan", "id": id})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event", "details": err.Error()})
			}
			return
		}

		if event.Status == "draft" {
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
				c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
				return
			}
		}

		if event.BannerURL != nil {
			masked := utils.MaskMediaURL(*event.BannerURL)
			event.BannerURL = &masked
		}
		if event.LogoURL != nil {
			masked := utils.MaskMediaURL(*event.LogoURL)
			event.LogoURL = &masked
		}
		if event.TechnicalGuidebookURL != nil {
			masked := utils.MaskMediaURL(*event.TechnicalGuidebookURL)
			event.TechnicalGuidebookURL = &masked
		}
		if event.OrganizerAvatarURL != nil {
			masked := utils.MaskMediaURL(*event.OrganizerAvatarURL)
			event.OrganizerAvatarURL = &masked
		}

		utils.PopulateEventDetailExtras(db, &event)

		c.JSON(http.StatusOK, event)
	}
}

// Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡ Archer Account (requires auth) Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡Î“Ã¶Ã‡

// MobileRegisterForEvent godoc
// @Summary      Register archer for event
// @Description  Authenticated archer self-registers for one or more event categories
// @Tags         Mobile - Archer
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "Event UUID or slug"
// @Param        request  body  object  true  "Category IDs and payment type"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
