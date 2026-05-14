package mobile

import (
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MobileListEvents handles listing events for mobile
// @Summary List Mobile Events
// @Description Get a list of active or past events optimized for mobile
// @Tags         Events
// @Produce json
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Param search query string false "Search by name or location"
// @Param history query bool false "Filter past events"
// @Success 200 {object} MobileEventsResponse
// @Router       /events [get]
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
// @Tags         Archer
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event Slug or UUID"
// @Success 200 {object} MobileArcherEventDetailResponse
// @Router       /archer/events/{id}/detail [get]
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
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventDetail
// @Failure 404 {object} map[string]interface{}
// @Router       /events/{slug} [get]
// MobileGetEventDetail returns core event information (slim)
// @Summary Get Mobile Event Detail (Slim)
// @Description Get summary details for a specific event without location, FAQ, or other granular info
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventDetailSlim
// @Failure 404 {object} map[string]interface{}
// @Router       /events/{slug} [get]
func MobileGetEventDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")

		query := `
			SELECT
				t.uuid, t.slug, t.name,
				t.start_date, t.end_date,
				t.logo_url, t.banner_url, t.description,
				COALESCE(u.full_name, '') as organizer_name,
				COALESCE(u.avatar_url, '') as organizer_avatar_url,
				COALESCE(u.slug, '') as organizer_slug,
				COALESCE(u.phone, '') as organizer_phone,
				COALESCE(active_target_stats.participant_count, 0) as participant_count
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

		var event MobileEventDetailSlim
		err := db.Get(&event, query, id, id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		if event.BannerURL != nil { *event.BannerURL = utils.MaskMediaURL(*event.BannerURL) }
		if event.LogoURL != nil { *event.LogoURL = utils.MaskMediaURL(*event.LogoURL) }
		if event.OrganizerAvatarURL != nil { *event.OrganizerAvatarURL = utils.MaskMediaURL(*event.OrganizerAvatarURL) }

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
// @Router       /events/{slug}/faq [get]
func MobileGetEventFAQ(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var faqRaw *string
		err := db.Get(&faqRaw, "SELECT faq FROM events WHERE uuid = ? OR slug = ?", id, id)
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
// @Router       /events/{slug}/registration-fee [get]
func MobileGetEventRegistrationFees(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var pageSettingsRaw *string
		err := db.Get(&pageSettingsRaw, "SELECT page_settings FROM events WHERE uuid = ? OR slug = ?", id, id)
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
// @Router       /events/{slug}/rewards [get]
func MobileGetEventRewards(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")
		var pageSettingsRaw *string
		err := db.Get(&pageSettingsRaw, "SELECT page_settings FROM events WHERE uuid = ? OR slug = ?", id, id)
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
// @Router       /events/{slug}/location [get]
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
			FROM events 
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
// @Router       /events/{slug}/participants [get]
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
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventScheduleResponse
// @Router       /events/{slug}/schedule [get]
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
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventCategoriesResponse
// @Router       /events/{slug}/categories [get]
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
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventGalleryResponse
// @Router       /events/{slug}/gallery [get]
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



// MobileRegisterEvent handles unified archer registration from mobile app
// @Summary Register for Event (Unified)
// @Description Register the authenticated archer for an event (handles both manual and gateway)
// @Tags Mobile - Archer
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileRegisterEventRequest true "Registration Details"
// @Success 200 {object} MobileRegisterEventResponse
// @Router /mobile/archer/events/register [post]
func MobileRegisterEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileRegisterEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		paymentType := "manual"
		if req.PaymentType == "online" || req.PaymentType == "gateway" {
			paymentType = "gateway"
		}

		internalReq := mobileRegistrationInternal{
			EventID:          req.EventID,
			AthleteID:        req.AthleteID,
			EventCategoryID:  req.EventCategoryID,
			EventCategoryIDs: req.EventCategoryIDs,
			PaymentMethod:    req.PaymentMethod,
			PaymentType:      paymentType,
		}

		processMobileRegistration(c, db, internalReq)
	}
}

// MobileRegisterEventManual handles archer registration with manual transfer
// @Summary Register for Event (Manual)
// @Description Register the authenticated archer for an event using manual transfer
// @Tags         Archer
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event UUID or Slug"
// @Param request body MobileEventManualRegistrationRequest true "Registration Details"
// @Success 200 {object} MobileRegisterEventResponse
// @Router       /archer/events/{id}/register/manual [post]
func MobileRegisterEventManual(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req MobileEventManualRegistrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		internalReq := mobileRegistrationInternal{
			EventID:          id,
			EventCategoryID:  req.EventCategoryID,
			EventCategoryIDs: req.EventCategoryIDs,
			PaymentAmount:    req.PaymentAmount,
			PaymentProofURLs: req.PaymentProofURLs,
			PaymentType:      "manual",
		}
		
		processMobileRegistration(c, db, internalReq)
	}
}

// MobileRegisterEventGateway handles archer registration with payment gateway
// @Summary Register for Event (Payment Gateway)
// @Description Register the authenticated archer for an event using online payment gateway
// @Tags         Archer
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event UUID or Slug"
// @Param request body MobileEventGatewayRegistrationRequest true "Registration Details"
// @Success 200 {object} MobileRegisterEventResponse
// @Router       /archer/events/{id}/register/payment-gateway [post]
func MobileRegisterEventGateway(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req MobileEventGatewayRegistrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		internalReq := mobileRegistrationInternal{
			EventID:          id,
			EventCategoryID:  req.EventCategoryID,
			EventCategoryIDs: req.EventCategoryIDs,
			PaymentMethod:    req.PaymentMethod,
			PaymentType:      "gateway",
		}
		
		processMobileRegistration(c, db, internalReq)
	}
}

type mobileRegistrationInternal struct {
	EventID          string
	AthleteID        string
	EventCategoryID  string
	EventCategoryIDs []string
	PaymentAmount    float64
	PaymentProofURLs []string
	PaymentMethod    string
	PaymentType      string
}

// processMobileRegistration contains the core logic for mobile event registration
func processMobileRegistration(c *gin.Context, db *sqlx.DB, req mobileRegistrationInternal) {
	// Re-use logic from RegisterParticipant but adapted for mobile responses and requirements
	// 1. Resolve Event
	var event struct {
		UUID        string `db:"uuid"`
		OrganizerID string `db:"organizer_id"`
	}
	err := db.Get(&event, `SELECT uuid, organizer_id FROM events WHERE uuid = ? OR slug = ?`, req.EventID, req.EventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
		return
	}

	// 2. Resolve Archer
	userID := c.GetString("user_id")
	var archerUUID string
	if req.AthleteID != "" {
		err = db.Get(&archerUUID, "SELECT uuid FROM archers WHERE uuid = ? OR id = ?", req.AthleteID, req.AthleteID)
	} else {
		err = db.Get(&archerUUID, "SELECT uuid FROM archers WHERE uuid = ?", userID)
	}
	
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Profil pemanah tidak ditemukan atau tidak valid"})
		return
	}

	// 3. Combine Categories
	allCategoryIDs := []string{}
	if req.EventCategoryID != "" {
		allCategoryIDs = append(allCategoryIDs, req.EventCategoryID)
	}
	for _, catID := range req.EventCategoryIDs {
		if catID != "" {
			exists := false
			for _, e := range allCategoryIDs {
				if e == catID { exists = true; break }
			}
			if !exists { allCategoryIDs = append(allCategoryIDs, catID) }
		}
	}

	if len(allCategoryIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pilih setidaknya satu kategori"})
		return
	}

	// 4. Transaction
	tx, err := db.Beginx()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}
	defer tx.Rollback()

	registrationDate := time.Now()
	paymentStatus := "menunggu acc"
	if req.PaymentType == "gateway" {
		paymentStatus = "unpaid"
	}

	var firstRegID string
	registeredCats := []string{}

	for i, catID := range allCategoryIDs {
		// Check duplicate
		var exists bool
		_ = tx.Get(&exists, "SELECT EXISTS(SELECT 1 FROM event_participants WHERE event_id = ? AND archer_id = ? AND category_id = ?)", event.UUID, archerUUID, catID)
		if exists { continue }

		regUUID := uuid.New().String()
		if i == 0 { firstRegID = regUUID }

		proofs := ""
		if len(req.PaymentProofURLs) > 0 {
			proofs = strings.Join(req.PaymentProofURLs, ",")
		}

		_, err = tx.Exec(`
			INSERT INTO event_participants (
				uuid, event_id, archer_id, category_id, 
				registration_date, payment_status, payment_amount, payment_proof_urls,
				registration_source
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, regUUID, event.UUID, archerUUID, catID, registrationDate, paymentStatus, req.PaymentAmount, proofs, "mobile_app")

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mendaftarkan ke kategori: " + catID})
			return
		}
		registeredCats = append(registeredCats, catID)
	}

	if len(registeredCats) == 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Anda sudah terdaftar di semua kategori pilihan"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pendaftaran"})
		return
	}

	c.JSON(http.StatusOK, MobileRegisterEventResponse{
		Message:              "Pendaftaran berhasil",
		RegistrationID:       firstRegID,
		RegisteredCategories: registeredCats,
		PaymentStatus:        paymentStatus,
	})
}

// MobileGetEventPaymentMethods returns available payment methods for an event
// @Summary Get Event Payment Methods
// @Description Get a list of available payment methods (manual bank transfer and online gateway) for an event
// @Tags         Events
// @Produce json
// @Param slug path string true "Event Slug or UUID"
// @Success 200 {object} MobileEventPaymentMethodsResponse
// @Router       /events/{slug}/payment-method [get]
func MobileGetEventPaymentMethods(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("slug")

		// 1. Get Event and Organizer ID
		var event struct {
			UUID        string `db:"uuid"`
			OrganizerID string `db:"organizer_id"`
		}
		err := db.Get(&event, "SELECT uuid, organizer_id FROM events WHERE uuid = ? OR slug = ? LIMIT 1", id, id)
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
		}
		_ = db.Select(&bankAccounts, `
			SELECT uuid, bank_name, account_number, account_name 
			FROM bank_accounts 
			WHERE user_id = ? AND status = 'verified'
			ORDER BY is_primary DESC
		`, event.OrganizerID)

		for _, b := range bankAccounts {
			accName := b.AccountName
			accNum := b.AccountNumber
			methods = append(methods, MobileEventPaymentMethodItem{
				Type:          "manual",
				ID:            b.UUID,
				BankName:      b.BankName,
				AccountName:   &accName,
				AccountNumber: &accNum,
			})
		}

		// 3. Fetch Gateway Payment Methods (Tripay)
		tripay := utils.NewTripayClient()
		channels, err := tripay.GetPaymentChannels()
		if err == nil {
			for _, ch := range channels {
				code := ch.Code
				icon := ch.IconURL
				methods = append(methods, MobileEventPaymentMethodItem{
					Type:     "gateway",
					ID:       ch.Code,
					BankName: ch.Name,
					Code:     &code,
					IconURL:  &icon,
				})
			}
		}

		c.JSON(http.StatusOK, MobileEventPaymentMethodsResponse{
			Methods: methods,
		})
	}
}
