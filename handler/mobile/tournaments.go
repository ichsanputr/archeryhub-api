package mobile

import (
	"Archeris-api/models"
	"Archeris-api/utils"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// MobileListEvents handles listing tournaments for mobile
// @Summary List Mobile Events
// @Description Get a list of active or past tournaments optimized for mobile
// @Tags         Events
// @Produce json
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Param search query string false "Search by name or location"
// @Param history query bool false "Filter past tournaments"
// @Success 200 {object} MobileEventsResponse
// @Router       /tournaments [get]
func MobileListEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := strings.TrimSpace(c.Query("search"))
		status := strings.ToLower(strings.TrimSpace(c.Query("status")))

		whereClause := "WHERE t.status != 'draft'"
		args := []interface{}{}
		
		// If path is /tournaments/history or ?history=true is passed
		if c.Request.URL.Path == "/api/v1/mobile/tournaments/history" || c.Query("history") == "true" {
			whereClause += " AND (t.status = 'completed' OR (t.end_date IS NOT NULL AND t.end_date < NOW()))"
		} else if status != "" && status != "all" && status != "semua" {
			switch status {
			case "upcoming":
				whereClause += " AND (t.status = 'upcoming' OR (t.status IN ('published', 'active') AND (t.start_date IS NULL OR t.start_date > NOW())))"
			case "ongoing":
				whereClause += " AND (t.status = 'ongoing' OR (t.status IN ('published', 'active') AND t.start_date IS NOT NULL AND t.start_date <= NOW() AND (t.end_date IS NULL OR t.end_date >= NOW())))"
			case "completed", "selesai":
				whereClause += " AND (t.status = 'completed' OR (t.status != 'draft' AND t.end_date IS NOT NULL AND t.end_date < NOW()))"
			default:
				whereClause += " AND t.status = ?"
				args = append(args, status)
			}
		}

		if search != "" {
			whereClause += ` AND (t.name LIKE ? OR t.location LIKE ? OR t.city LIKE ?)`
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		query := `
			SELECT 
				t.uuid, t.slug, t.name, 
				COALESCE(t.location, '') as location, 
				COALESCE(t.city, '') as city,
				CASE 
					WHEN t.status = 'completed' OR (t.end_date IS NOT NULL AND t.end_date < NOW()) THEN 'completed'
					WHEN t.status = 'ongoing' OR (t.start_date IS NOT NULL AND t.start_date <= NOW() AND (t.end_date IS NULL OR t.end_date >= NOW())) THEN 'ongoing'
					WHEN t.status = 'upcoming' OR (t.start_date IS NOT NULL AND t.start_date > NOW()) THEN 'upcoming'
					ELSE COALESCE(t.status, 'published')
				END as status,
				COALESCE(t.start_date, '') as start_date, 
				COALESCE(t.end_date, '') as end_date, 
				t.logo_url, t.banner_url,
				COALESCE(u.full_name, '') as organizer_name,
				u.avatar_url as organizer_avatar_url,
				COUNT(DISTINCT tp.archer_id) as participant_count,
				COALESCE(cat_stats.cat_count, 0) as category_count,
				t.entry_fee
			FROM tournaments t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, avatar_url FROM organizers
				UNION ALL
				SELECT uuid as id, name as full_name, logo_url as avatar_url FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN tournament_participants tp ON t.uuid = tp.tournament_id
			LEFT JOIN (
				SELECT tournament_id, COUNT(*) as cat_count
				FROM tournament_categories
				GROUP BY tournament_id
			) cat_stats ON t.uuid = cat_stats.tournament_id
			` + whereClause + `
			GROUP BY t.uuid, t.slug, t.name, t.location, t.city, t.status, t.start_date, t.end_date, t.logo_url, t.banner_url, u.full_name, u.avatar_url, cat_stats.cat_count, t.entry_fee
			ORDER BY t.start_date DESC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)

		var tournaments []MobileEvent
		err := db.Select(&tournaments, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data event", "details": err.Error()})
			return
		}

		for i := range tournaments {
			if tournaments[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].LogoURL)
				tournaments[i].LogoURL = &masked
			}
			if tournaments[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].BannerURL)
				tournaments[i].BannerURL = &masked
			}
			if tournaments[i].OrganizerAvatarURL != nil {
				masked := utils.MaskMediaURL(*tournaments[i].OrganizerAvatarURL)
				tournaments[i].OrganizerAvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"tournaments":      tournaments,
			"total_count": len(tournaments), // Simple count for now, could be improved with separate COUNT query
		})
	}
}

// MobileArcherGetEventDetail returns event detail with archer's registration status
// @Summary Get Event Detail (Archer)
// @Description Get event details including registration status for the authenticated archer
// @Tags         Archer
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event Slug or UUID"
// @Success 200 {object} MobileArcherEventDetailResponse
// @Router       /archer/tournaments/{id}/detail [get]
func MobileArcherGetEventDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		userID := c.GetString("user_id")

		// 1. Get Event Detail (same query as public detail)
		query := `
			SELECT 
				t.uuid, t.slug, t.name, t.venue, t.gmaps_link, t.location, t.address, t.city, t.location_type,
				t.start_date, t.end_date, t.registration_deadline,
				CASE 
					WHEN t.status = 'completed' OR (t.end_date IS NOT NULL AND t.end_date < NOW()) THEN 'completed'
					WHEN t.status = 'ongoing' OR (t.start_date IS NOT NULL AND t.start_date <= NOW() AND (t.end_date IS NULL OR t.end_date >= NOW())) THEN 'ongoing'
					WHEN t.status = 'upcoming' OR (t.start_date IS NOT NULL AND t.start_date > NOW()) THEN 'upcoming'
					ELSE COALESCE(t.status, 'published')
				END as status,
				t.logo_url, t.banner_url, t.description, t.technical_guidebook_url,
				COALESCE(u.full_name, '') as organizer_name,
				COALESCE(u.avatar_url, '') as organizer_avatar_url,
				COALESCE(u.slug, '') as organizer_slug,
				COALESCE(u.phone, '') as organizer_phone,
				COALESCE(active_target_stats.participant_count, 0) as participant_count,
				t.organizer_id
			FROM tournaments t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, avatar_url, slug, whatsapp_no as phone FROM organizers
				UNION ALL
				SELECT uuid as id, name as full_name, logo_url as avatar_url, slug, NULL as phone FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN (
				SELECT tournament_id, COUNT(*) as participant_count
				FROM tournament_participants
				GROUP BY tournament_id
			) active_target_stats ON t.uuid = active_target_stats.tournament_id
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
			FROM tournament_participants
			WHERE tournament_id = ? AND archer_id = ?
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
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventDetail
// @Failure 404 {object} map[string]interface{}
// @Router       /tournaments/{slug} [get]
// MobileGetEventDetail returns core event information (slim)
// @Summary Get Mobile Event Detail (Slim)
// @Description Get summary details for a specific event without location, FAQ, or other granular info
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventDetailSlim
// @Failure 404 {object} map[string]interface{}
// @Router       /tournaments/{slug} [get]
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
			FROM tournaments t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, avatar_url, slug, whatsapp_no as phone FROM organizers
				UNION ALL
				SELECT uuid as id, name as full_name, logo_url as avatar_url, slug, NULL as phone FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN (
				SELECT tournament_id, COUNT(*) as participant_count
				FROM tournament_participants
				GROUP BY tournament_id
			) active_target_stats ON t.uuid = active_target_stats.tournament_id
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

		event.LocationDetail = models.EventLocationDetail{
			Venue:        event.Venue,
			Address:      event.Address,
			GmapLink:     event.GmapLink,
			Location:     event.Location,
			City:         event.City,
			LocationType: event.LocationType,
		}

		// Check organizer subscription status for current user if authenticated
		userIDVal, _ := c.Get("user_id")
		emailVal, _ := c.Get("email")
		uidStr := ""
		if userIDVal != nil {
			uidStr = fmt.Sprintf("%v", userIDVal)
		}
		emailStr := ""
		if emailVal != nil {
			emailStr = fmt.Sprintf("%v", emailVal)
		}

		if uidStr != "" || emailStr != "" {
			orgName := ""
			if event.OrganizerName != nil {
				orgName = *event.OrganizerName
			}
			var isSubscribed bool
			_ = db.Get(&isSubscribed, `
				SELECT EXISTS(
					SELECT 1 FROM organizer_subscribers
					WHERE is_active = 1
					  AND (
						  (? != '' AND user_id = ?) 
						  OR (? != '' AND email = ?)
					  )
					  AND (
						  tournament_slug = ? 
						  OR (? != '' AND organizer_name = ?)
					  )
				)
			`, uidStr, uidStr, emailStr, emailStr, event.Slug, orgName, orgName)
			event.IsOrganizerSubscribed = isSubscribed
		}

		c.JSON(http.StatusOK, event)
	}
}

// MobileGetEventFAQ returns FAQ data for an event
// @Summary Get Event FAQ
// @Description Get the list of frequently asked questions for an event
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventFAQResponse
// @Router       /tournaments/{slug}/faq [get]
func MobileGetEventFAQ(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var faqRaw *string
		err := db.Get(&faqRaw, "SELECT faq FROM tournaments WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var faq []MobileEventFAQItem
		if faqRaw != nil && *faqRaw != "" {
			_ = json.Unmarshal([]byte(*faqRaw), &faq)
		}

		c.JSON(http.StatusOK, MobileEventFAQResponse{FAQ: faq})
	}
}

// MobileGetEventRegistrationFees returns registration fee data for an event
// @Summary Get Event Registration Fees
// @Description Get the list of registration fees and categories for an event
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventFeesResponse
// @Router       /tournaments/{slug}/registration-fee [get]
func MobileGetEventRegistrationFees(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var pageSettingsRaw *string
		err := db.Get(&pageSettingsRaw, "SELECT page_settings FROM tournaments WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var settings struct {
			Fees []MobileEventFeeItem `json:"fees"`
		}
		if pageSettingsRaw != nil && *pageSettingsRaw != "" {
			_ = json.Unmarshal([]byte(*pageSettingsRaw), &settings)
		}

		c.JSON(http.StatusOK, MobileEventFeesResponse{Fees: settings.Fees})
	}
}

// MobileGetEventRewards returns prize/reward data for an event
// @Summary Get Event Rewards
// @Description Get the list of prizes and rewards for an event
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventRewardsResponse
// @Router       /tournaments/{slug}/rewards [get]
func MobileGetEventRewards(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var pageSettingsRaw *string
		err := db.Get(&pageSettingsRaw, "SELECT page_settings FROM tournaments WHERE uuid = ? OR slug = ?", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var settings struct {
			Prizes MobileEventRewardsResponse `json:"prizes"`
		}
		if pageSettingsRaw != nil && *pageSettingsRaw != "" {
			_ = json.Unmarshal([]byte(*pageSettingsRaw), &settings)
		}

		c.JSON(http.StatusOK, settings.Prizes)
	}
}

// MobileGetEventLocation returns location data for an event
// @Summary Get Event Location
// @Description Get detailed location information and accessibility for an event
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventLocationResponse
// @Router       /tournaments/{slug}/location [get]
func MobileGetEventLocation(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		
		var data struct {
			Venue        string  `db:"venue"`
			Address      string  `db:"address"`
			City         string  `db:"city"`
			Location     string  `db:"location"`
			GmapLink     string  `db:"gmaps_link"`
			LocationType string  `db:"location_type"`
			PageSettings *string `db:"page_settings"`
		}

		err := db.Get(&data, `
			SELECT venue, COALESCE(address, '') as address, COALESCE(city, '') as city, 
			       COALESCE(location, '') as location, COALESCE(gmaps_link, '') as gmaps_link, 
			       COALESCE(location_type, '') as location_type, page_settings 
			FROM tournaments 
			WHERE uuid = ? OR slug = ?
		`, id, id)
		
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var settings struct {
			Accessibility []string `json:"location_accessibility"`
		}
		if data.PageSettings != nil && *data.PageSettings != "" {
			_ = json.Unmarshal([]byte(*data.PageSettings), &settings)
		}

		c.JSON(http.StatusOK, MobileEventLocationResponse{
			Venue:                data.Venue,
			Address:              data.Address,
			City:                 data.City,
			Location:             data.Location,
			GmapLink:             data.GmapLink,
			LocationType:         data.LocationType,
			LocationAccessibility: settings.Accessibility,
		})
	}
}

// MobileGetEventParticipants returns only the participant list for an event
// @Summary Get Event Participants
// @Description Get the list of registered archers for an event
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventParticipantsResponse
// @Router       /tournaments/{slug}/participants [get]
func MobileGetEventParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var eventID string
		_ = db.Get(&eventID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?", id, id)
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
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventScheduleResponse
// @Router       /tournaments/{slug}/schedule [get]
func MobileGetEventSchedule(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var eventID string
		_ = db.Get(&eventID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?", id, id)
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
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventCategoriesResponse
// @Router       /tournaments/{slug}/categories [get]
func MobileGetEventCategories(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var eventID string
		_ = db.Get(&eventID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?", id, id)
		if eventID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		event := &models.EventWithDetails{Event: models.Event{UUID: eventID}}
		utils.PopulateEventDetailExtras(db, event)

		c.JSON(http.StatusOK, gin.H{
			"competition_categories": event.CompetitionCategories,
			"categories":             event.CompetitionCategories,
		})
	}
}

// MobileGetEventGallery returns only the gallery images for an event
// @Summary Get Event Gallery
// @Description Get event gallery and documentation images
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventGalleryResponse
// @Router       /tournaments/{slug}/gallery [get]
func MobileGetEventGallery(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var eventID string
		_ = db.Get(&eventID, "SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?", id, id)
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



// MobileRegisterEvent handles unified archer registration from mobile app
// @Summary Register for Event (Unified)
// @Description Register the authenticated archer for an event (handles both manual and gateway)
// @Tags Mobile - Archer
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileRegisterEventRequest true "Registration Details"
// @Success 200 {object} MobileRegisterEventResponse
// @Router /mobile/archer/tournaments/register [post]
// MobileRegisterEvent handles unified archer registration from mobile app
// @Summary Register for Event (Unified)
// @Description Register the authenticated archer for an event (handles both manual and gateway, self and club delegation)
// @Tags Mobile - Archer
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileRegisterEventRequest true "Registration Details"
// @Success 200 {object} MobileRegisterEventResponse
// @Router /mobile/archer/tournaments/register [post]
func MobileRegisterEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileRegisterEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		processMobileRegistration(c, db, req)
	}
}

// processMobileRegistration contains the core logic for mobile event registration
func processMobileRegistration(c *gin.Context, db *sqlx.DB, req MobileRegisterEventRequest) {
	// 1. Resolve Event
	var event struct {
		UUID                 string     `db:"uuid"`
		OrganizerID          string     `db:"organizer_id"`
		EntryFee             float64    `db:"entry_fee"`
		Status               string     `db:"status"`
		RegistrationDeadline *time.Time `db:"registration_deadline"`
		QuotaMaxParticipants *int       `db:"quota_max_participants"`
	}
	err := db.Get(&event, `SELECT uuid, organizer_id, COALESCE(entry_fee, 0.0) as entry_fee, status, registration_deadline, quota_max_participants FROM tournaments WHERE uuid = ? OR slug = ?`, req.EventID, req.EventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
		return
	}

	actualEventID := event.UUID

	// Check registration deadline
	if event.RegistrationDeadline != nil && time.Now().After(*event.RegistrationDeadline) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Pendaftaran untuk turnamen ini telah ditutup",
			"code":  "registration_closed",
		})
		return
	}

	// 2. Resolve Authenticated Archer / Captain
	userID := c.GetString("user_id")
	var captainArcherUUID string
	if req.AthleteID != "" {
		_ = db.Get(&captainArcherUUID, "SELECT uuid FROM archers WHERE uuid = ? OR id = ?", req.AthleteID, req.AthleteID)
	}
	if captainArcherUUID == "" {
		_ = db.Get(&captainArcherUUID, "SELECT uuid FROM archers WHERE uuid = ?", userID)
	}
	if captainArcherUUID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Profil pemanah tidak ditemukan atau tidak valid"})
		return
	}

	// 3. Transaction
	tx, err := db.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}
	defer tx.Rollback()

	registrationDate := time.Now()
	paymentStatus := "unpaid"
	registrationMode := req.RegistrationMode
	if registrationMode == "" {
		registrationMode = "captain_team"
	}

	allCreatedParticipantUUIDs := []string{}
	registeredCats := []string{}
	var calculatedTotalFee float64 = 0

	type CategoryMeta struct {
		UUID            string  `db:"uuid"`
		TournamentID    string  `db:"tournament_id"`
		MaxParticipants *int    `db:"max_participants"`
		Fee             float64 `db:"fee"`
		GenderCode      string  `db:"gender_code"`
		AgeCode         string  `db:"age_code"`
	}

	getCatMeta := func(catUUID string) (*CategoryMeta, error) {
		var meta CategoryMeta
		err := tx.Get(&meta, `
			SELECT tc.uuid, tc.tournament_id, tc.max_participants,
			       COALESCE(
				       NULLIF(tc.fee, 0.0),
				       CASE
					       WHEN t.fee_mode = 'per_type' AND LOWER(COALESCE(rtt.name, '')) LIKE '%mixed%' THEN NULLIF(t.fee_mixed_team, 0.0)
					       WHEN t.fee_mode = 'per_type' AND (LOWER(COALESCE(rtt.name, '')) LIKE '%team%' OR LOWER(COALESCE(rtt.name, '')) LIKE '%beregu%') THEN NULLIF(t.fee_team, 0.0)
					       WHEN t.fee_mode = 'per_type' THEN NULLIF(t.fee_individual, 0.0)
					       ELSE NULLIF(t.entry_fee, 0.0)
				       END,
				       t.entry_fee,
				       0.0
			       ) as fee,
			       COALESCE(rgd.code, 'mixed') as gender_code,
			       COALESCE(rag.code, 'umum') as age_code
			FROM tournament_categories tc
			JOIN tournaments t ON tc.tournament_id = t.uuid
			LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
			LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
			LEFT JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
			WHERE tc.uuid = ? AND tc.tournament_id = ?
		`, catUUID, actualEventID)
		if err != nil {
			return nil, err
		}
		return &meta, nil
	}

	checkQuota := func(meta *CategoryMeta) bool {
		if meta.MaxParticipants == nil || *meta.MaxParticipants <= 0 {
			return true
		}
		var currentCount int
		_ = tx.Get(&currentCount, `
			SELECT COUNT(*) FROM tournament_participants 
			WHERE tournament_id = ? AND category_id = ? AND payment_status != 'cancelled'
		`, actualEventID, meta.UUID)
		return currentCount < *meta.MaxParticipants
	}

	if registrationMode == "captain_team" {
		// ─────────────────────────────────────────────────────────────────────
		// MODE 1: CAPTAIN & TEAM REGISTRATION
		// ─────────────────────────────────────────────────────────────────────
		allCategoryIDs := []string{}
		if strings.TrimSpace(req.EventCategoryID) != "" {
			allCategoryIDs = append(allCategoryIDs, strings.TrimSpace(req.EventCategoryID))
		}
		for _, catID := range req.EventCategoryIDs {
			trimmed := strings.TrimSpace(catID)
			if trimmed != "" {
				exists := false
				for _, e := range allCategoryIDs {
					if e == trimmed {
						exists = true
						break
					}
				}
				if !exists {
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
			}

			catFee := event.EntryFee
			if catMeta != nil && catMeta.Fee > 0 {
				catFee = catMeta.Fee
			}

			var existingUUID string
			_ = tx.Get(&existingUUID, `
				SELECT uuid FROM tournament_participants 
				WHERE tournament_id = ? AND archer_id = ? AND category_id = ? AND payment_status != 'cancelled'
				LIMIT 1
			`, actualEventID, captainArcherUUID, catID)

			if existingUUID != "" {
				allCreatedParticipantUUIDs = append(allCreatedParticipantUUIDs, existingUUID)
				registeredCats = append(registeredCats, catID)
				continue
			}

			partUUID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO tournament_participants (
					uuid, tournament_id, archer_id, category_id, 
					registration_date, payment_status, payment_amount,
					registration_source
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, partUUID, actualEventID, captainArcherUUID, catID, registrationDate, paymentStatus, catFee, "self_register")

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan peserta", "details": err.Error()})
				return
			}

			allCreatedParticipantUUIDs = append(allCreatedParticipantUUIDs, partUUID)
			registeredCats = append(registeredCats, catID)
			calculatedTotalFee += catFee
		}

		// Process Team Registrations
		for _, teamInput := range req.TeamRegistrations {
			if teamInput.CategoryID == "" {
				continue
			}

			catMeta, _ := getCatMeta(teamInput.CategoryID)
			teamFee := event.EntryFee
			if catMeta != nil && catMeta.Fee > 0 {
				teamFee = catMeta.Fee
			}

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
			calculatedTotalFee += teamFee

			for orderIdx, member := range teamInput.Members {
				var memberPartUUID string
				if member.ParticipantID != nil && *member.ParticipantID != "" {
					memberPartUUID = *member.ParticipantID
				} else {
					var memberArcherUUID string
					if member.ArcherID != "" {
						_ = tx.Get(&memberArcherUUID, "SELECT uuid FROM archers WHERE uuid = ? OR id = ? LIMIT 1", member.ArcherID, member.ArcherID)
					}
					if memberArcherUUID == "" && member.FullName != "" {
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

					// Find participant row
					_ = tx.Get(&memberPartUUID, `
						SELECT uuid FROM tournament_participants 
						WHERE tournament_id = ? AND archer_id = ? AND (category_id = ? OR category_id = ?) AND payment_status != 'cancelled'
						LIMIT 1
					`, actualEventID, memberArcherUUID, teamInput.CategoryID, teamInput.CategoryID)

					if memberPartUUID == "" {
						memberPartUUID = uuid.New().String()
						memberFee := event.EntryFee
						_, err = tx.Exec(`
							INSERT INTO tournament_participants (
								uuid, tournament_id, archer_id, category_id,
								registration_date, payment_status, payment_amount,
								registration_source
							) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
						`, memberPartUUID, actualEventID, memberArcherUUID, teamInput.CategoryID, registrationDate, paymentStatus, memberFee, "self_register")
						if err == nil {
							allCreatedParticipantUUIDs = append(allCreatedParticipantUUIDs, memberPartUUID)
							if member.NeedsIndividualRegistration {
								calculatedTotalFee += memberFee
							}
						}
					}
				}

				if memberPartUUID != "" {
					tmUUID := uuid.New().String()
					_, _ = tx.Exec(`
						INSERT INTO team_members (uuid, team_id, participant_id, member_order)
						VALUES (?, ?, ?, ?)
					`, tmUUID, teamUUID, memberPartUUID, orderIdx+1)
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
				} else if req.ClubID != nil && *req.ClubID != "" {
					clubIDVal = req.ClubID
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
						_, _ = tx.Exec("INSERT INTO clubs (uuid, name, slug, created_at, updated_at) VALUES (?, ?, ?, NOW(), NOW())", newClubUUID, cleanClubName, clubSlug)
						clubIDVal = &newClubUUID
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
				}

				catFee := event.EntryFee
				if catMeta != nil && catMeta.Fee > 0 {
					catFee = catMeta.Fee
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
				_, err = tx.Exec(`
					INSERT INTO tournament_participants (
						uuid, tournament_id, archer_id, category_id,
						registration_date, payment_status, payment_amount,
						registration_source
					) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, partUUID, actualEventID, athArcherUUID, trimmed, registrationDate, paymentStatus, catFee, "invited")
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan atlet delegasi", "details": err.Error()})
					return
				}
				allCreatedParticipantUUIDs = append(allCreatedParticipantUUIDs, partUUID)
				registeredCats = append(registeredCats, trimmed)
				calculatedTotalFee += catFee
			}
		}

		// Process Delegation Teams
		for _, teamBooking := range req.DelegationTeams {
			if teamBooking.CategoryID == "" || teamBooking.Count <= 0 {
				continue
			}

			catMeta, _ := getCatMeta(teamBooking.CategoryID)
			teamFee := event.EntryFee
			if catMeta != nil && catMeta.Fee > 0 {
				teamFee = catMeta.Fee
			}

			for i := 0; i < teamBooking.Count; i++ {
				teamUUID := uuid.New().String()
				teamName := teamBooking.TeamName
				if strings.TrimSpace(teamName) == "" {
					teamName = fmt.Sprintf("Tim %s %d", req.ClubName, i+1)
				}

				_, err = tx.Exec(`
					INSERT INTO teams (uuid, tournament_id, event_id, category_id, team_name, status)
					VALUES (?, ?, ?, ?, ?, 'active')
				`, teamUUID, actualEventID, teamBooking.CategoryID, teamBooking.CategoryID, teamName)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat reservasi tim", "details": err.Error()})
					return
				}
				calculatedTotalFee += teamFee
			}
		}
	}

	if len(allCreatedParticipantUUIDs) == 0 && len(req.DelegationTeams) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada atlet atau tim yang didaftarkan"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pendaftaran"})
		return
	}

	firstRegID := ""
	if len(allCreatedParticipantUUIDs) > 0 {
		firstRegID = allCreatedParticipantUUIDs[0]
	} else {
		firstRegID = uuid.New().String()
	}

	totalAmount := calculatedTotalFee
	if req.PaymentAmount > 0 {
		totalAmount = req.PaymentAmount
	}

	var checkoutURL *string
	var vaNumber *string
	var qrURL *string
	var gatewayReference *string

	if totalAmount <= 0 {
		paymentStatus = "paid"
		for _, pID := range allCreatedParticipantUUIDs {
			_, _ = db.Exec("UPDATE tournament_participants SET payment_status = 'paid' WHERE uuid = ?", pID)
		}
	} else if req.PaymentMethod != "" {
		var archer struct {
			FullName string  `db:"full_name"`
			Email    *string `db:"email"`
			Phone    *string `db:"phone"`
		}
		_ = db.Get(&archer, "SELECT full_name, email, phone FROM archers WHERE uuid = ?", captainArcherUUID)

		customerName := archer.FullName
		customerEmail := utils.StringValue(archer.Email, "user@archeris.net")
		customerPhone := utils.StringValue(archer.Phone, "08123456789")

		appURL := os.Getenv("APP_URL")
		if appURL == "" {
			appURL = "http://localhost:3003"
		}
		merchantRef := fmt.Sprintf("PAY-REG-%s", strings.ToUpper(uuid.New().String()[:8]))
		transactionID := uuid.New().String()

		if req.PaymentMethod == "paypal" {
			paypalClient := utils.NewPayPalClient()
			usdAmount := paypalClient.ConvertIDRToUSD(totalAmount)
			returnURL := fmt.Sprintf("%s/payment/status/%s?provider=paypal", strings.TrimSuffix(appURL, "/"), merchantRef)
			cancelURL := fmt.Sprintf("%s/payment/status/%s?cancelled=true", strings.TrimSuffix(appURL, "/"), merchantRef)
			description := fmt.Sprintf("Event Reg - %s", customerName)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			orderResp, approveURL, err := paypalClient.CreateOrder(ctx, merchantRef, description, usdAmount, returnURL, cancelURL)
			var orderID string
			checkoutURLVal := approveURL
			if err == nil && orderResp != nil {
				orderID = orderResp.ID
			} else {
				checkoutURLVal = fmt.Sprintf("https://www.paypal.com/checkoutnow?token=%s", merchantRef)
			}
			expiredAt := time.Now().Add(24 * time.Hour)

			transaction := models.PaymentTransaction{
				UUID:             transactionID,
				Reference:        merchantRef,
				GatewayReference: utils.StringPtr(merchantRef),
				UserID:           captainArcherUUID,
				EventID:          &event.UUID,
				RegistrationID:   &firstRegID,
				Amount:           totalAmount,
				FeeAmount:        0,
				TotalAmount:      totalAmount,
				PaymentMethod:    utils.StringPtr("paypal"),
				CheckoutURL:      &checkoutURLVal,
				Months:           1,
				Status:           "pending",
				ExpiredAt:        utils.TimePtr(expiredAt),
			}
			if orderID != "" {
				transaction.GatewayReference = &orderID
			}

			query := `
				INSERT INTO payment_transactions (
					uuid, reference, gateway_reference, user_id, tournament_id, registration_id,
					amount, fee_amount, total_amount, payment_method,
					checkout_url, months, status, expired_at
				) VALUES (
					:uuid, :reference, :gateway_reference, :user_id, :tournament_id, :registration_id,
					:amount, :fee_amount, :total_amount, :payment_method,
					:checkout_url, :months, :status, :expired_at
				)
			`
			_, err = db.NamedExec(query, transaction)
			if err == nil {
				for _, pID := range allCreatedParticipantUUIDs {
					_, _ = db.Exec("UPDATE tournament_participants SET payment_id = ?, payment_status = 'pending', payment_method = 'paypal' WHERE uuid = ?", transactionID, pID)
				}
				gatewayReference = transaction.GatewayReference
				checkoutURL = &checkoutURLVal
				paymentStatus = "pending"
			}
		} else if req.PaymentMethod == "manual" || req.PaymentType == "manual" {
			expiredAt := time.Now().Add(7 * 24 * time.Hour)
			statusVal := "pending"
			var proofURLPtr *string
			var senderNamePtr *string
			var proofTimePtr *time.Time
			if req.PaymentProofURL != "" {
				proofURLPtr = &req.PaymentProofURL
				statusVal = "awaiting_verification"
				proofTimePtr = utils.TimePtr(time.Now())
			}
			if req.SenderName != "" {
				senderNamePtr = &req.SenderName
			}

			transaction := models.PaymentTransaction{
				UUID:            transactionID,
				Reference:       merchantRef,
				UserID:          captainArcherUUID,
				EventID:         &event.UUID,
				RegistrationID:  &firstRegID,
				Amount:          totalAmount,
				FeeAmount:       0,
				TotalAmount:     totalAmount,
				PaymentMethod:   utils.StringPtr("manual"),
				ProofURL:        proofURLPtr,
				ProofUploadedAt: proofTimePtr,
				SenderName:      senderNamePtr,
				Months:          1,
				Status:          statusVal,
				ExpiredAt:       utils.TimePtr(expiredAt),
			}

			query := `
				INSERT INTO payment_transactions (
					uuid, reference, user_id, tournament_id, registration_id,
					amount, fee_amount, total_amount, payment_method,
					proof_url, proof_uploaded_at, sender_name,
					months, status, expired_at
				) VALUES (
					:uuid, :reference, :user_id, :tournament_id, :registration_id,
					:amount, :fee_amount, :total_amount, :payment_method,
					:proof_url, :proof_uploaded_at, :sender_name,
					:months, :status, :expired_at
				)
			`
			_, err = db.NamedExec(query, transaction)
			if err == nil {
				for _, pID := range allCreatedParticipantUUIDs {
					_, _ = db.Exec("UPDATE tournament_participants SET payment_id = ?, payment_status = ?, payment_method = 'manual' WHERE uuid = ?", transactionID, statusVal, pID)
				}
				paymentStatus = statusVal
				refVal := merchantRef
				gatewayReference = &refVal
			}
		} else {
			// Mayar gateway
			amountInt := int(totalAmount)
			mayarClient := utils.NewMayarClient()
			redirectURL := fmt.Sprintf("%s/payment/status/%s", strings.TrimSuffix(appURL, "/"), merchantRef)

			paymentReq := utils.MayarPaymentReq{
				Name:        fmt.Sprintf("Event Reg - %s", customerName),
				Amount:      amountInt,
				Email:       customerEmail,
				Mobile:      customerPhone,
				Description: fmt.Sprintf("Pendaftaran Event: %s", customerName),
				RedirectURL: redirectURL,
			}

			var checkoutURLVal string
			var mayarTxID string
			mayarData, err := mayarClient.CreatePaymentRequest(paymentReq)
			if err == nil && mayarData != nil {
				checkoutURLVal = mayarData.Link
				mayarTxID = mayarData.TransactionID
			} else {
				checkoutURLVal = fmt.Sprintf("https://checkout.mayar.id/pay/%s", merchantRef)
				mayarTxID = merchantRef
			}
			expiredAt := time.Now().Add(24 * time.Hour)

			transaction := models.PaymentTransaction{
				UUID:             transactionID,
				Reference:        merchantRef,
				GatewayReference: &mayarTxID,
				UserID:           captainArcherUUID,
				EventID:          &event.UUID,
				RegistrationID:   &firstRegID,
				Amount:           totalAmount,
				FeeAmount:        0,
				TotalAmount:      totalAmount,
				PaymentMethod:    utils.StringPtr("mayar"),
				CheckoutURL:      &checkoutURLVal,
				Months:           1,
				Status:           "pending",
				ExpiredAt:        utils.TimePtr(expiredAt),
			}

			query := `
				INSERT INTO payment_transactions (
					uuid, reference, gateway_reference, user_id, tournament_id, registration_id,
					amount, fee_amount, total_amount, payment_method,
					checkout_url, months, status, expired_at
				) VALUES (
					:uuid, :reference, :gateway_reference, :user_id, :tournament_id, :registration_id,
					:amount, :fee_amount, :total_amount, :payment_method,
					:checkout_url, :months, :status, :expired_at
				)
			`
			_, err = db.NamedExec(query, transaction)
			if err == nil {
				for _, pID := range allCreatedParticipantUUIDs {
					_, _ = db.Exec("UPDATE tournament_participants SET payment_id = ?, payment_status = 'pending', payment_method = ? WHERE uuid = ?", transactionID, req.PaymentMethod, pID)
				}
				qrURL = transaction.QRURL
				gatewayReference = &merchantRef
				checkoutURL = &checkoutURLVal
				paymentStatus = "pending"
			}
		}
	}

	c.JSON(http.StatusOK, MobileRegisterEventResponse{
		Message:              "Pendaftaran berhasil",
		RegistrationID:       firstRegID,
		ParticipantIDs:       allCreatedParticipantUUIDs,
		RegisteredCategories: registeredCats,
		PaymentStatus:        paymentStatus,
		TotalFee:             totalAmount,
		CheckoutURL:          checkoutURL,
		VANumber:             vaNumber,
		QRURL:                qrURL,
		GatewayReference:     gatewayReference,
	})
}

// MobileGetEventPaymentMethods returns available payment methods for an event
// @Summary Get Event Payment Methods
// @Description Get a list of available payment methods (manual bank transfer and online gateway) for an event
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventPaymentMethodsResponse
// @Router       /tournaments/{slug}/payment-method [get]
func MobileGetEventPaymentMethods(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")

		// 1. Get Event and Organizer ID
		var event struct {
			UUID        string `db:"uuid"`
			OrganizerID string `db:"organizer_id"`
		}
		err := db.Get(&event, "SELECT uuid, organizer_id FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1", id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var methods []MobileEventPaymentMethodItem

		// 2. Fetch Manual Payment Methods (Bank Accounts)
		var bankAccounts []struct {
			UUID          string  `db:"uuid"`
			BankName      string  `db:"bank_name"`
			AccountNumber string  `db:"account_number"`
			AccountName   string  `db:"account_name"`
			Instructions  *string `db:"instructions"`
		}
		_ = db.Select(&bankAccounts, `
			SELECT uuid, 
			       CASE WHEN type = 'custom' AND custom_name IS NOT NULL AND custom_name != '' THEN custom_name ELSE bank_name END as bank_name, 
			       account_number, account_name, instructions
			FROM bank_accounts 
			WHERE user_id = ? AND (status IS NULL OR status = '' OR status = 'active' OR status = 'verified')
			ORDER BY is_primary DESC, created_at ASC
		`, event.OrganizerID)

		if len(bankAccounts) == 0 {
			_ = db.Select(&bankAccounts, `
				SELECT uuid, 
				       CASE WHEN type = 'custom' AND custom_name IS NOT NULL AND custom_name != '' THEN custom_name ELSE bank_name END as bank_name, 
				       account_number, account_name, instructions
				FROM organizer_payment_methods 
				WHERE organization_id = ? AND is_active = 1
				ORDER BY is_primary DESC, created_at ASC
			`, event.OrganizerID)
		}

		for _, b := range bankAccounts {
			accName := b.AccountName
			accNum := b.AccountNumber
			instr := "Transfer ke rekening panitia & upload struk bukti bayar"
			if b.Instructions != nil && *b.Instructions != "" {
				instr = *b.Instructions
			}
			methods = append(methods, MobileEventPaymentMethodItem{
				Type:          "manual",
				ID:            b.UUID,
				BankName:      b.BankName,
				AccountName:   &accName,
				AccountNumber: &accNum,
				Instructions:  &instr,
			})
		}

		// 3. Fetch Gateway Payment Methods (Mayar & PayPal)
		gatewayChannels := []struct{ Code, Name, Icon string }{
			{"mayar", "Mayar Payment Gateway (QRIS, VA & Kartu Debit/Kredit)", "/payment-method/mayar.png"},
			{"paypal", "PayPal (Kartu Internasional & Saldo USD)", "/payment-method/paypal.png"},
			{"QRIS", "QRIS (Semua E-Wallet & Bank)", "/payment-method/qris.png"},
			{"BCAVA", "BCA Virtual Account", "/payment-method/bca.png"},
			{"BNIVA", "BNI Virtual Account", "/payment-method/bni.png"},
			{"BRIVA", "BRI Virtual Account", "/payment-method/bri.png"},
			{"MANDIRIVA", "Mandiri Virtual Account", "/payment-method/mandiri.png"},
		}
		for _, ch := range gatewayChannels {
			cCode := ch.Code
			cIcon := ch.Icon
			methods = append(methods, MobileEventPaymentMethodItem{
				Type:     "gateway",
				ID:       ch.Code,
				BankName: ch.Name,
				Code:     &cCode,
				IconURL:  &cIcon,
			})
		}

		c.JSON(http.StatusOK, MobileEventPaymentMethodsResponse{
			Methods: methods,
		})
	}
}

// MobileCancelRegistration allows an archer to cancel their own pending event registration
func MobileCancelRegistration(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		registrationID := c.Param("registration_id")
		userID := c.GetString("user_id")

		// 1. Verify that the registration exists, belongs to the current user, and is still pending
		var reg struct {
			UUID          string `db:"uuid"`
			PaymentStatus string `db:"payment_status"`
		}
		err := db.Get(&reg, "SELECT uuid, COALESCE(payment_status, 'pending') as payment_status FROM tournament_participants WHERE uuid = ? AND archer_id = ?", registrationID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pendaftaran tidak ditemukan atau Anda tidak memiliki akses"})
			return
		}

		if reg.PaymentStatus == "paid" || reg.PaymentStatus == "settlement" || reg.PaymentStatus == "lunas" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Pendaftaran yang sudah lunas tidak dapat dibatalkan"})
			return
		}

		// 2. Delete the registration (tournament_participants) and any pending transactions associated with it
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
			return
		}
		defer tx.Rollback()

		_, _ = tx.Exec("UPDATE payment_transactions SET status = 'failed' WHERE registration_id = ?", registrationID)
		_, err = tx.Exec("UPDATE tournament_participants SET payment_status = 'cancelled' WHERE uuid = ?", registrationID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membatalkan pendaftaran"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan pembatalan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Pendaftaran berhasil dibatalkan"})
	}
}

// MobileCancelPayment allows an archer to cancel their own pending payment & registration
func MobileCancelPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		refOrVaOrReg := c.Param("identifier")
		userID := c.GetString("user_id")

		// Find the registration/participant record
		var regID string
		var paymentStatus string

		// Try to find by registration_id (uuid in event_participants)
		err := db.QueryRow("SELECT uuid, COALESCE(payment_status, 'pending') FROM event_participants WHERE uuid = ? AND archer_id = ?", refOrVaOrReg, userID).Scan(&regID, &paymentStatus)
		if err != nil {
			// Try to find by payment transaction reference or gateway_reference or va_number
			var pt struct {
				RegistrationID *string `db:"registration_id"`
				Status         string  `db:"status"`
			}
			err2 := db.Get(&pt, `
				SELECT registration_id, status 
				FROM payment_transactions 
				WHERE (uuid = ? OR reference = ? OR gateway_reference = ? OR va_number = ?) 
				  AND user_id = ?
			`, refOrVaOrReg, refOrVaOrReg, refOrVaOrReg, refOrVaOrReg, userID)
			if err2 == nil && pt.RegistrationID != nil {
				regID = *pt.RegistrationID
				paymentStatus = pt.Status
			}
		}

		if regID == "" {
			_ = db.QueryRow(`
				SELECT uuid, COALESCE(payment_status, 'pending') 
				FROM tournament_participants 
				WHERE archer_id = ? AND payment_status IN ('pending', 'unpaid', '') 
				ORDER BY registration_date DESC LIMIT 1
			`, userID).Scan(&regID, &paymentStatus)
		}

		if regID == "" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pendaftaran atau transaksi tidak ditemukan"})
			return
		}

		if paymentStatus == "paid" || paymentStatus == "settlement" || paymentStatus == "lunas" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Transaksi yang sudah lunas tidak dapat dibatalkan"})
			return
		}

		// Delete the transaction and registration
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
			return
		}
		defer tx.Rollback()

		_, _ = tx.Exec("UPDATE payment_transactions SET status = 'failed' WHERE registration_id = ?", regID)
		_, err = tx.Exec("UPDATE tournament_participants SET payment_status = 'cancelled' WHERE uuid = ?", regID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membatalkan pendaftaran"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pembatalan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "success", "message": "Pendaftaran berhasil dibatalkan"})
	}
}

// MobileSubscribeOrganizer registers email notification for an organizer's new events
func MobileSubscribeOrganizer(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		var req struct {
			Email string `json:"email" binding:"required,email"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format email tidak valid"})
			return
		}

		var event struct {
			OrganizerName string `db:"organizer_name"`
			CreatedBy     string `db:"created_by"`
		}
		err := db.Get(&event, `
			SELECT COALESCE(o.name, 'Organizer') as organizer_name, COALESCE(t.user_id, '') as created_by
			FROM tournaments t
			LEFT JOIN organizations o ON t.organization_id = o.id
			WHERE t.uuid = ? OR t.slug = ?
		`, slug, slug)

		if err != nil {
			_ = db.Get(&event, `SELECT 'Penyelenggara Turnamen' as organizer_name, '' as created_by FROM tournaments WHERE uuid = ? OR slug = ?`, slug, slug)
		}

		userID, _ := c.Get("user_id")
		uidStr := ""
		if userID != nil {
			uidStr = userID.(string)
		}

		_, err = db.Exec(`
			INSERT INTO organizer_subscribers (user_id, email, organizer_id, organizer_name, tournament_slug, is_active)
			VALUES (?, ?, ?, ?, ?, 1)
			ON DUPLICATE KEY UPDATE is_active = 1, updated_at = CURRENT_TIMESTAMP
		`, uidStr, req.Email, event.CreatedBy, event.OrganizerName, slug)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal berlangganan info turnamen"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":         "success",
			"is_subscribed":  true,
			"organizer_name": event.OrganizerName,
			"message":        "Berhasil berlangganan notifikasi turnamen",
		})
	}
}
