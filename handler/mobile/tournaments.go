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
		search := c.Query("search")

		whereClause := "WHERE t.status != 'draft'"
		
		// If path is /tournaments/history or ?history=true is passed
		if c.Request.URL.Path == "/api/v1/mobile/tournaments/history" || c.Query("history") == "true" {
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
			GROUP BY t.uuid, t.slug, u.full_name, u.avatar_url, cat_stats.cat_count, t.entry_fee
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
func MobileRegisterEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileRegisterEventRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		internalReq := mobileRegistrationInternal{
			EventID:          req.EventID,
			AthleteID:        req.AthleteID,
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
	PaymentMethod    string
	PaymentType      string
}

// processMobileRegistration contains the core logic for mobile event registration
func processMobileRegistration(c *gin.Context, db *sqlx.DB, req mobileRegistrationInternal) {
	// Re-use logic from RegisterParticipant but adapted for mobile responses and requirements
	// 1. Resolve Event
	var event struct {
		UUID        string  `db:"uuid"`
		OrganizerID string  `db:"organizer_id"`
		EntryFee    float64 `db:"entry_fee"`
	}
	err := db.Get(&event, `SELECT uuid, organizer_id, COALESCE(entry_fee, 0.0) as entry_fee FROM tournaments WHERE uuid = ? OR slug = ?`, req.EventID, req.EventID)
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
	paymentStatus := "unpaid"

	var firstRegID string
	registeredCats := []string{}

	for i, catID := range allCategoryIDs {
		// Check duplicate
		var exists bool
		_ = tx.Get(&exists, "SELECT EXISTS(SELECT 1 FROM tournament_participants WHERE tournament_id = ? AND archer_id = ? AND category_id = ? AND payment_status != 'cancelled')", event.UUID, archerUUID, catID)
		if exists { continue }

		// Check category quota capacity and fee
		var catInfo struct {
			Quota        int     `db:"quota"`
			Fee          float64 `db:"fee"`
			CurrentCount int     `db:"current_count"`
		}
		qErr := tx.Get(&catInfo, `
			SELECT 
				COALESCE(ec.quota, 0) as quota,
				COALESCE(ec.fee, 0.0) as fee,
				(SELECT COUNT(*) FROM tournament_participants WHERE category_id = ec.uuid AND payment_status != 'cancelled') as current_count
			FROM tournament_categories ec WHERE ec.uuid = ? FOR UPDATE
		`, catID)
		if qErr == nil && catInfo.Quota > 0 && catInfo.CurrentCount >= catInfo.Quota {
			c.JSON(http.StatusConflict, gin.H{"error": "Kuota pendaftaran untuk kategori ini telah penuh"})
			return
		}

		catFee := event.EntryFee
		if catInfo.Fee > 0 {
			catFee = catInfo.Fee
		}

		regUUID := uuid.New().String()
		if i == 0 { firstRegID = regUUID }

		_, err = tx.Exec(`
			INSERT INTO tournament_participants (
				uuid, tournament_id, archer_id, category_id, 
				registration_date, payment_status, payment_amount,
				registration_source
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, regUUID, event.UUID, archerUUID, catID, registrationDate, paymentStatus, catFee, "self_register")

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

	var checkoutURL *string
	var vaNumber *string
	var qrURL *string
	var gatewayReference *string

	totalAmount := float64(len(registeredCats)) * event.EntryFee

	if req.PaymentMethod != "" && totalAmount > 0 {
		var archer struct {
			FullName string  `db:"full_name"`
			Email    *string `db:"email"`
			Phone    *string `db:"phone"`
		}
		_ = db.Get(&archer, "SELECT full_name, email, phone FROM archers WHERE uuid = ?", archerUUID)

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
			if err == nil {
				orderID := orderResp.ID
				checkoutURLVal := approveURL
				expiredAt := time.Now().Add(24 * time.Hour)

				transaction := models.PaymentTransaction{
					UUID:             transactionID,
					Reference:        merchantRef,
					GatewayReference: &orderID,
					UserID:           archerUUID,
					EventID:          &event.UUID,
					RegistrationID:   &firstRegID,
					Amount:           totalAmount,
					FeeAmount:        0,
					TotalAmount:      totalAmount,
					PaymentMethod:    utils.StringPtr("paypal"),
					CheckoutURL:      &checkoutURLVal,
					Months:           1,
					Status:           "pending",
					ExpiredAt:        expiredAt,
				}

				query := `
					INSERT INTO payment_transactions (
						uuid, reference, gateway_reference, user_id, event_id, registration_id,
						amount, fee_amount, total_amount, payment_method,
						checkout_url, months, status, expired_at
					) VALUES (
						:uuid, :reference, :gateway_reference, :user_id, :event_id, :registration_id,
						:amount, :fee_amount, :total_amount, :payment_method,
						:checkout_url, :months, :status, :expired_at
					)
				`
				_, err = db.NamedExec(query, transaction)
				if err == nil {
					_, _ = db.Exec("UPDATE event_participants SET payment_id = ?, payment_status = 'pending', payment_method = 'paypal' WHERE event_id = ? AND archer_id = ?", transactionID, event.UUID, archerUUID)
					gatewayReference = &orderID
					checkoutURL = &checkoutURLVal
					paymentStatus = "pending"

					c.JSON(http.StatusOK, MobileRegisterEventResponse{
						Message:              "Pendaftaran berhasil",
						RegistrationID:       firstRegID,
						RegisteredCategories: registeredCats,
						PaymentStatus:        paymentStatus,
						TotalFee:             totalAmount,
						CheckoutURL:          checkoutURL,
						VANumber:             vaNumber,
						QRURL:                qrURL,
						GatewayReference:     gatewayReference,
					})
					return
				}
			}
		} else if req.PaymentMethod == "manual" {
			expiredAt := time.Now().Add(7 * 24 * time.Hour)
			transaction := models.PaymentTransaction{
				UUID:           transactionID,
				Reference:      merchantRef,
				UserID:         archerUUID,
				EventID:        &event.UUID,
				RegistrationID: &firstRegID,
				Amount:         totalAmount,
				FeeAmount:      0,
				TotalAmount:    totalAmount,
				PaymentMethod:  utils.StringPtr("manual"),
				Months:         1,
				Status:         "pending",
				ExpiredAt:      expiredAt,
			}

			query := `
				INSERT INTO payment_transactions (
					uuid, reference, user_id, event_id, registration_id,
					amount, fee_amount, total_amount, payment_method,
					months, status, expired_at
				) VALUES (
					:uuid, :reference, :user_id, :event_id, :registration_id,
					:amount, :fee_amount, :total_amount, :payment_method,
					:months, :status, :expired_at
				)
			`
			_, err = db.NamedExec(query, transaction)
			if err == nil {
				_, _ = db.Exec("UPDATE event_participants SET payment_id = ?, payment_status = 'pending', payment_method = 'manual' WHERE event_id = ? AND archer_id = ?", transactionID, event.UUID, archerUUID)
				paymentStatus = "pending"
				refVal := merchantRef
				gatewayReference = &refVal

				c.JSON(http.StatusOK, MobileRegisterEventResponse{
					Message:              "Pendaftaran berhasil",
					RegistrationID:       firstRegID,
					RegisteredCategories: registeredCats,
					PaymentStatus:        paymentStatus,
					TotalFee:             totalAmount,
					CheckoutURL:          nil,
					VANumber:             nil,
					QRURL:                nil,
					GatewayReference:     gatewayReference,
				})
				return
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

			mayarData, err := mayarClient.CreatePaymentRequest(paymentReq)
			if err == nil {
				checkoutURLVal := mayarData.Link
				mayarTxID := mayarData.TransactionID
				expiredAt := time.Now().Add(24 * time.Hour)

				transaction := models.PaymentTransaction{
					UUID:             transactionID,
					Reference:        merchantRef,
					GatewayReference: &mayarTxID,
					UserID:           archerUUID,
					EventID:          &event.UUID,
					RegistrationID:   &firstRegID,
					Amount:           totalAmount,
					FeeAmount:        0,
					TotalAmount:      totalAmount,
					PaymentMethod:    utils.StringPtr("mayar"),
					CheckoutURL:      &checkoutURLVal,
					Months:           1,
					Status:           "pending",
					ExpiredAt:        expiredAt,
				}

				query := `
					INSERT INTO payment_transactions (
						uuid, reference, gateway_reference, user_id, event_id, registration_id,
						amount, fee_amount, total_amount, payment_method,
						checkout_url, months, status, expired_at
					) VALUES (
						:uuid, :reference, :gateway_reference, :user_id, :event_id, :registration_id,
						:amount, :fee_amount, :total_amount, :payment_method,
						:checkout_url, :months, :status, :expired_at
					)
				`
				_, err = db.NamedExec(query, transaction)
				if err == nil {
					_, _ = db.Exec("UPDATE event_participants SET payment_id = ?, payment_status = 'pending', payment_method = ? WHERE event_id = ? AND archer_id = ?", transactionID, req.PaymentMethod, event.UUID, archerUUID)
					qrURL = transaction.QRURL
					gatewayReference = &mayarTxID
					checkoutURL = &checkoutURLVal
					paymentStatus = "pending"

					c.JSON(http.StatusOK, MobileRegisterEventResponse{
						Message:              "Pendaftaran berhasil",
						RegistrationID:       firstRegID,
						RegisteredCategories: registeredCats,
						PaymentStatus:        paymentStatus,
						TotalFee:             totalAmount,
						CheckoutURL:          checkoutURL,
						VANumber:             vaNumber,
						QRURL:                qrURL,
						GatewayReference:     gatewayReference,
					})
					return
				}
			}
		}
	}

	c.JSON(http.StatusOK, MobileRegisterEventResponse{
		Message:              "Pendaftaran berhasil",
		RegistrationID:       firstRegID,
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
