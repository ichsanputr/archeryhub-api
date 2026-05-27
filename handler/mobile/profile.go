package mobile

import (
	"Archeris-api/handler"
	"Archeris-api/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func requireMobileUserType(c *gin.Context, expected string) bool {
	if c.GetString("user_type") != expected {
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak untuk tipe pengguna ini"})
		return false
	}
	return true
}

func parseJSONObject(raw *string) map[string]interface{} {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(*raw), &data); err != nil {
		return nil
	}
	return data
}

func parseJSONArray(raw *string) []interface{} {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var data []interface{}
	if err := json.Unmarshal([]byte(*raw), &data); err != nil {
		return nil
	}
	return data
}

func parseJSONValue(raw *string) interface{} {
	if raw == nil || strings.TrimSpace(*raw) == "" {
		return nil
	}
	var data interface{}
	if err := json.Unmarshal([]byte(*raw), &data); err != nil {
		return *raw
	}
	return data
}

func stringPointerFromMap(data map[string]interface{}, key string) *string {
	if data == nil {
		return nil
	}
	value, ok := data[key]
	if !ok {
		return nil
	}
	stringValue, ok := value.(string)
	if !ok || strings.TrimSpace(stringValue) == "" {
		return nil
	}
	return &stringValue
}

func mapFromMap(data map[string]interface{}, key string) map[string]interface{} {
	if data == nil {
		return nil
	}
	value, ok := data[key]
	if !ok {
		return nil
	}
	mapValue, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	return mapValue
}

// MobileGetSellerMe returns current seller profile
// @Summary Get Seller Profile
// @Description Get profile information for the authenticated seller
// @Tags Mobile - Seller
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileSellerProfileResponse
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /mobile/seller/me [get]
func MobileGetSellerMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "seller") {
			return
		}

		userID := c.GetString("user_id")
		var seller struct {
			UUID         string  `db:"uuid"`
			StoreName    string  `db:"store_name"`
			Slug         *string `db:"slug"`
			Description  *string `db:"description"`
			AvatarURL    *string `db:"avatar_url"`
			BannerURL    *string `db:"banner_url"`
			Phone        *string `db:"phone"`
			Email        *string `db:"email"`
			Address      *string `db:"address"`
			City         *string `db:"city"`
			Province     *string `db:"province"`
			PageSettings *string `db:"page_settings"`
		}

		err := db.Get(&seller, `
			SELECT uuid, store_name, slug, description, avatar_url, banner_url,
			       phone, email, address, city, province, page_settings
			FROM sellers
			WHERE uuid = ? OR user_id = ?
		`, userID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penjual tidak ditemukan"})
			return
		}

		if seller.AvatarURL != nil {
			masked := utils.MaskMediaURL(*seller.AvatarURL)
			seller.AvatarURL = &masked
		}
		if seller.BannerURL != nil {
			masked := utils.MaskMediaURL(*seller.BannerURL)
			seller.BannerURL = &masked
		}

		pageSettings := parseJSONObject(seller.PageSettings)
		response := MobileSellerProfileResponse{
			Data: MobileSellerProfileData{
				ID:            seller.UUID,
				UUID:          seller.UUID,
				StoreName:     seller.StoreName,
				Slug:          seller.Slug,
				StoreSlug:     seller.Slug,
				Name:          seller.StoreName,
				Username:      seller.Slug,
				Description:   seller.Description,
				AvatarURL:     seller.AvatarURL,
				BannerURL:     seller.BannerURL,
				Logo:          seller.AvatarURL,
				Banner:        seller.BannerURL,
				Phone:         seller.Phone,
				Email:         seller.Email,
				Address:       seller.Address,
				City:          seller.City,
				Province:      seller.Province,
				Sections:      mapFromMap(pageSettings, "sections"),
				CatalogConfig: mapFromMap(pageSettings, "catalog_config"),
				ThemeColor:    stringPointerFromMap(pageSettings, "theme_color"),
				BannerText:    stringPointerFromMap(pageSettings, "banner_text"),
				PageSettings:  pageSettings,
				UserType:      "seller",
			},
		}

		c.JSON(http.StatusOK, response)
	}
}

// MobileUpdateSellerMe updates basic seller profile
// @Summary Update Seller Profile
// @Description Update basic profile info (name, store_name, phone, etc.)
// @Tags Mobile - Seller
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileUpdateSellerProfileBasicRequest true "Profile Data"
// @Success 200 {object} MessageResponse
// @Router /mobile/seller/me [patch]
func MobileUpdateSellerMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "seller") {
			return
		}
		handler.UpdateSellerProfileBasic(db)(c)
	}
}

// MobileUpdateSellerPage godoc
func MobileUpdateSellerPage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "seller") {
			return
		}
		handler.UpdateSellerProfile(db)(c)
	}
}

// MobileGetOrganizationMe returns current organization profile
// @Summary Get Organization Profile
// @Description Get profile information for the authenticated organization
// @Tags Mobile - Organization
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileOrganizationProfileResponse
// @Router /mobile/organization/me [get]
func MobileGetOrganizationMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "organization") {
			return
		}

		userID := c.GetString("user_id")
		var org struct {
			UUID                  string  `db:"uuid"`
			Slug                  *string `db:"slug"`
			Name                  string  `db:"name"`
			Acronym               *string `db:"acronym"`
			Description           *string `db:"description"`
			Website               *string `db:"website"`
			Email                 string  `db:"email"`
			WhatsAppNo            *string `db:"whatsapp_no"`
			AvatarURL             *string `db:"avatar_url"`
			BannerURL             *string `db:"banner_url"`
			Address               *string `db:"address"`
			City                  *string `db:"city"`
			Country               *string `db:"country"`
			RegistrationNumber    *string `db:"registration_number"`
			EstablishedDate       *string `db:"established_date"`
			ContactPersonName     *string `db:"contact_person_name"`
			ContactPersonEmail    *string `db:"contact_person_email"`
			ContactPersonPhone    *string `db:"contact_person_phone"`
			SocialFacebook        *string `db:"social_facebook"`
			SocialInstagram       *string `db:"social_instagram"`
			SocialTwitter         *string `db:"social_twitter"`
			SocialMedia           *string `db:"social_media"`
			Vision                *string `db:"vision"`
			Mission               *string `db:"mission"`
			History               *string `db:"history"`
			FAQ                   *string `db:"faq"`
			Status                *string `db:"status"`
			CreatedAt             string  `db:"created_at"`
			UpdatedAt             string  `db:"updated_at"`
			SubscriptionStatus    string  `db:"subscription_status"`
			SubscriptionExpiresAt *string `db:"subscription_expires_at"`
			PageSettings          *string `db:"page_settings"`
		}

		err := db.Get(&org, `
			SELECT uuid, slug, name, acronym, description, website, email, whatsapp_no,
			       avatar_url, banner_url, address, city, country,
			       registration_number, established_date, contact_person_name,
			       contact_person_email, contact_person_phone,
			       social_facebook, social_instagram, social_twitter, social_media,
			       status, created_at, updated_at,
			       vision, mission, history, faq,
			       COALESCE(subscription_status, 'trial') as subscription_status,
			       subscription_expires_at, page_settings
			FROM organizations
			WHERE uuid = ?
		`, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organisasi tidak ditemukan"})
			return
		}

		if org.AvatarURL != nil {
			masked := utils.MaskMediaURL(*org.AvatarURL)
			org.AvatarURL = &masked
		}
		if org.BannerURL != nil {
			masked := utils.MaskMediaURL(*org.BannerURL)
			org.BannerURL = &masked
		}

		verified := "verified"
		response := MobileOrganizationProfileResponse{
			Data: MobileOrganizationProfileData{
				ID:                    org.UUID,
				UUID:                  org.UUID,
				Slug:                  org.Slug,
				Name:                  org.Name,
				Acronym:               org.Acronym,
				Description:           org.Description,
				Website:               org.Website,
				Email:                 org.Email,
				WhatsAppNo:            org.WhatsAppNo,
				AvatarURL:             org.AvatarURL,
				BannerURL:             org.BannerURL,
				Address:               org.Address,
				City:                  org.City,
				Country:               org.Country,
				RegistrationNumber:    org.RegistrationNumber,
				EstablishedDate:       org.EstablishedDate,
				ContactPersonName:     org.ContactPersonName,
				ContactPersonEmail:    org.ContactPersonEmail,
				ContactPersonPhone:    org.ContactPersonPhone,
				SocialFacebook:        org.SocialFacebook,
				SocialInstagram:       org.SocialInstagram,
				SocialTwitter:         org.SocialTwitter,
				SocialMedia:           parseJSONValue(org.SocialMedia),
				Vision:                org.Vision,
				Mission:               org.Mission,
				History:               org.History,
				FAQ:                   parseJSONArray(org.FAQ),
				VerificationStatus:    &verified,
				Status:                org.Status,
				CreatedAt:             org.CreatedAt,
				UpdatedAt:             org.UpdatedAt,
				SubscriptionStatus:    org.SubscriptionStatus,
				SubscriptionExpiresAt: org.SubscriptionExpiresAt,
				PageSettings:          parseJSONObject(org.PageSettings),
				UserType:              "organization",
			},
		}

		c.JSON(http.StatusOK, response)
	}
}

// MobileUpdateOrganizationMe updates organization profile
// @Summary Update Organization Profile
// @Description Update profile info for the authenticated organization
// @Tags Mobile - Organization
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileUpdateOrganizationProfileRequest true "Profile Data"
// @Success 200 {object} MessageResponse
// @Router /mobile/organization/me [patch]
func MobileUpdateOrganizationMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "organization") {
			return
		}
		handler.UpdateOrganizationProfile(db)(c)
	}
}

func getMobileOrganizationUUID(c *gin.Context, db *sqlx.DB) (string, bool) {
	userID := c.GetString("user_id")
	var orgUUID string
	if err := db.Get(&orgUUID, `SELECT uuid FROM organizations WHERE uuid = ?`, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Organisasi tidak ditemukan"})
		return "", false
	}
	return orgUUID, true
}

func getOwnedEventUUID(c *gin.Context, db *sqlx.DB, organizationUUID, eventID string) (string, bool) {
	var eventUUID string
	err := db.Get(&eventUUID, `
		SELECT uuid
		FROM events
		WHERE organizer_id = ? AND (uuid = ? OR slug = ?)
		LIMIT 1
	`, organizationUUID, eventID, eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Event organisasi tidak ditemukan"})
		return "", false
	}
	return eventUUID, true
}

// MobileGetOrganizationEvents returns events owned by organization
// @Summary Get Organization Events
// @Description Get list of events organized by the authenticated organization
// @Tags Mobile - Organization
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Param status query string false "Filter by status"
// @Param search query string false "Search events"
// @Success 200 {object} MobileOrganizationEventsResponse
// @Router /mobile/organization/events [get]
func MobileGetOrganizationEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "organization") {
			return
		}

		organizationUUID, ok := getMobileOrganizationUUID(c, db)
		if !ok {
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		status := strings.TrimSpace(c.Query("status"))
		search := strings.TrimSpace(c.Query("search"))

		whereClause := "WHERE e.organizer_id = ?"
		args := []interface{}{organizationUUID}
		if status != "" {
			whereClause += " AND e.status = ?"
			args = append(args, status)
		}
		if search != "" {
			searchTerm := "%" + search + "%"
			whereClause += " AND (e.name LIKE ? OR e.code LIKE ? OR e.location LIKE ? OR e.venue LIKE ?)"
			args = append(args, searchTerm, searchTerm, searchTerm, searchTerm)
		}

		var total int
		if err := db.Get(&total, `SELECT COUNT(*) FROM events e `+whereClause, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung event organisasi", "details": err.Error()})
			return
		}

		var events []MobileOrganizationEventItem
		query := `
			SELECT
				e.uuid,
				e.name,
				e.slug,
				e.location,
				e.venue,
				e.start_date,
				e.end_date,
				COALESCE(e.status, '') as status,
				e.logo_url,
				e.banner_url,
				COALESCE(ps.participant_count, 0) as participant_count,
				COALESCE(ps.verified_count, 0) as verified_count,
				COALESCE(ps.pending_count, 0) as pending_count,
				CASE
					WHEN e.registration_deadline IS NOT NULL AND e.registration_deadline < NOW() THEN TRUE
					ELSE FALSE
				END as registration_closed
			FROM events e
			LEFT JOIN (
				SELECT
					event_id,
					COUNT(*) as participant_count,
					SUM(CASE WHEN payment_status IN ('paid', 'lunas') THEN 1 ELSE 0 END) as verified_count,
					SUM(CASE WHEN payment_status IN ('pending', 'menunggu_acc', 'menunggu acc') THEN 1 ELSE 0 END) as pending_count
				FROM event_participants
				GROUP BY event_id
			) ps ON ps.event_id = e.uuid
			` + whereClause + `
			ORDER BY e.start_date DESC, e.created_at DESC
			LIMIT ? OFFSET ?
		`
		queryArgs := append(args, limit, offset)
		if err := db.Select(&events, query, queryArgs...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil event organisasi", "details": err.Error()})
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
		}

		c.JSON(http.StatusOK, MobileOrganizationEventsResponse{
			Events: events,
			Total:  total,
			Limit:  limit,
			Offset: offset,
		})
	}
}

// MobileGetOrganizationEventParticipants returns participants for an event
// @Summary Get Event Participants (Org)
// @Description Get list of participants for an event owned by the organization
// @Tags Mobile - Organization
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event Slug or UUID"
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Param search query string false "Search participants"
// @Param payment_status query string false "Filter by payment status"
// @Param category_id query string false "Filter by category"
// @Success 200 {object} MobileOrganizationEventParticipantsResponse
// @Router /mobile/organization/events/{id}/participants [get]
func MobileGetOrganizationEventParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "organization") {
			return
		}

		organizationUUID, ok := getMobileOrganizationUUID(c, db)
		if !ok {
			return
		}
		eventUUID, ok := getOwnedEventUUID(c, db, organizationUUID, c.Param("id"))
		if !ok {
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := strings.TrimSpace(c.Query("search"))
		paymentStatus := strings.TrimSpace(c.Query("payment_status"))
		categoryID := strings.TrimSpace(c.Query("category_id"))
		reregistered := strings.TrimSpace(c.Query("reregistered")) // "true", "false", or ""

		whereClause := "WHERE tp.event_id = ?"
		args := []interface{}{eventUUID}
		countArgs := []interface{}{eventUUID}
		if categoryID != "" {
			whereClause += " AND tp.category_id = ?"
			args = append(args, categoryID)
			countArgs = append(countArgs, categoryID)
		}
		if search != "" {
			searchTerm := "%" + search + "%"
			whereClause += " AND (a.full_name LIKE ? OR cl.name LIKE ? OR a.id LIKE ?)"
			args = append(args, searchTerm, searchTerm, searchTerm)
			countArgs = append(countArgs, searchTerm, searchTerm, searchTerm)
		}
		if paymentStatus != "" && paymentStatus != "Semua" {
			if paymentStatus == "pending" || paymentStatus == "unpaid" {
				whereClause += " AND tp.payment_status IN ('pending', 'unpaid', 'belum_lunas', 'menunggu_acc', 'menunggu acc')"
			} else if paymentStatus == "paid" {
				whereClause += " AND tp.payment_status IN ('paid', 'lunas')"
			} else {
				whereClause += " AND tp.payment_status = ?"
				args = append(args, paymentStatus)
				countArgs = append(countArgs, paymentStatus)
			}
		}
		if reregistered == "true" {
			whereClause += " AND tp.last_reregistration_at IS NOT NULL"
		} else if reregistered == "false" {
			whereClause += " AND tp.last_reregistration_at IS NULL"
		}

		var total int
		countQuery := "SELECT COUNT(*) FROM event_participants tp LEFT JOIN archers a ON tp.archer_id = a.uuid LEFT JOIN clubs cl ON a.club_id = cl.uuid " + whereClause
		if err := db.Get(&total, countQuery, countArgs...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung jumlah peserta", "details": err.Error()})
			return
		}

		participants := []MobileOrganizationParticipantItem{}
		query := `
			SELECT
				tp.uuid as id,
				tp.archer_id,
				a.id as athlete_code,
				a.username,
				COALESCE(a.full_name, '') as full_name,
				COALESCE(a.email, '') as email,
				a.city,
				a.club_id,
				NULLIF(COALESCE(cl.name, ''), '') as club_name,
				tp.event_id,
				tp.category_id,
				COALESCE(bt.name, '') as division_name,
				COALESCE(ec.category_name_custom, ag.name, '') as category_name,
				NULLIF(COALESCE(et.name, ''), '') as event_type_name,
				NULLIF(COALESCE(gd.name, ''), '') as gender_division_name,
				tp.target_name,
				tp.qr_raw,
				a.avatar_url,
				tp.registration_date,
				COALESCE(tp.payment_status, 'pending') as payment_status
			FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories ec ON tp.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			LEFT JOIN ref_event_types et ON ec.event_type_uuid = et.uuid
			LEFT JOIN ref_gender_divisions gd ON ec.gender_division_uuid = gd.uuid
			` + whereClause + `
			ORDER BY tp.registration_date DESC, a.full_name ASC
			LIMIT ? OFFSET ?
		`
		queryArgs := append(args, limit, offset)
		if err := db.Select(&participants, query, queryArgs...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data peserta", "details": err.Error()})
			return
		}

		for i := range participants {
			if participants[i].AvatarURL != nil {
				masked := utils.MaskMediaURL(*participants[i].AvatarURL)
				participants[i].AvatarURL = &masked
			}
			participants[i].QRCodeDataURL = buildMobileQRCodeDataURL(participants[i].QRRaw)
		}

		statusWhere := "WHERE event_id = ?"
		statusArgs := []interface{}{eventUUID}
		if categoryID != "" {
			statusWhere += " AND category_id = ?"
			statusArgs = append(statusArgs, categoryID)
		}
		var verifiedCount, pendingCount int
		_ = db.Get(&verifiedCount, "SELECT COUNT(*) FROM event_participants "+statusWhere+" AND payment_status IN ('paid', 'lunas')", statusArgs...)
		_ = db.Get(&pendingCount, "SELECT COUNT(*) FROM event_participants "+statusWhere+" AND payment_status IN ('pending', 'menunggu_acc', 'menunggu acc')", statusArgs...)

		c.JSON(http.StatusOK, MobileOrganizationEventParticipantsResponse{
			Participants:  participants,
			Total:         total,
			VerifiedCount: verifiedCount,
			PendingCount:  pendingCount,
			Limit:         limit,
			Offset:        offset,
		})
	}
}

// MobileGetArcherMe returns current archer profile
// @Summary Get Archer Profile
// @Description Get profile information for the authenticated archer
// @Tags Mobile - Archer
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileArcherProfileResponse
// @Router /mobile/archer/me [get]
func MobileGetArcherMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "archer") {
			return
		}

		userID := c.GetString("user_id")
		var archer struct {
			UUID        string  `db:"uuid"`
			ID          string  `db:"id"`
			Username    *string `db:"username"`
			FullName    string  `db:"full_name"`
			Email       *string `db:"email"`
			AvatarURL   *string `db:"avatar_url"`
			Phone       *string `db:"phone"`
			Gender      *string `db:"gender"`
			DateOfBirth *string `db:"date_of_birth"`
			City        *string `db:"city"`
			Address     *string `db:"address"`
			BowType     *string `db:"bow_type"`
			ClubID      *string `db:"club_id"`
			ClubName    *string `db:"club_name"`
		}

		query := `
			SELECT 
				a.uuid, a.id, a.username, a.full_name, a.email, a.avatar_url,
				a.phone, a.gender, CAST(a.date_of_birth AS CHAR) as date_of_birth,
				a.city, a.address, a.bow_type,
				a.club_id, c.name as club_name
			FROM archers a
			LEFT JOIN clubs c ON a.club_id = c.uuid
			WHERE a.uuid = ?
		`
		err := db.Get(&archer, query, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Pemanah tidak ditemukan"})
			return
		}

		if archer.AvatarURL != nil {
			masked := utils.MaskMediaURL(*archer.AvatarURL)
			archer.AvatarURL = &masked
		}

		c.JSON(http.StatusOK, MobileArcherProfileResponse{
			Data: MobileArcherProfileData{
				ID:          archer.ID,
				UUID:        archer.UUID,
				Username:    archer.Username,
				FullName:    archer.FullName,
				Email:       archer.Email,
				AvatarURL:   archer.AvatarURL,
				Phone:       archer.Phone,
				Gender:      archer.Gender,
				DateOfBirth: archer.DateOfBirth,
				City:        archer.City,
				Address:     archer.Address,
				BowType:     archer.BowType,
				ClubID:      archer.ClubID,
				ClubName:    archer.ClubName,
				UserType:    "archer",
			},
		})
	}
}

// MobileUpdateArcherMe updates archer profile
// @Summary Update Archer Profile
// @Description Update profile info for the authenticated archer
// @Tags Mobile - Archer
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileUpdateArcherProfileRequest true "Profile Data"
// @Success 200 {object} MessageResponse
// @Router /mobile/archer/me [patch]
func MobileUpdateArcherMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "archer") {
			return
		}

		userID := c.GetString("user_id")
		var req MobileUpdateArcherProfileRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid", "details": err.Error()})
			return
		}

		query := "UPDATE archers SET updated_at = NOW()"
		args := []interface{}{}

		if req.FullName != nil { query += ", full_name = ?"; args = append(args, *req.FullName) }
		if req.Phone != nil { query += ", phone = ?"; args = append(args, *req.Phone) }
		if req.Gender != nil { query += ", gender = ?"; args = append(args, *req.Gender) }
		if req.DateOfBirth != nil { query += ", date_of_birth = ?"; args = append(args, *req.DateOfBirth) }
		if req.City != nil { query += ", city = ?"; args = append(args, *req.City) }
		if req.Address != nil { query += ", address = ?"; args = append(args, *req.Address) }
		if req.BowType != nil { query += ", bow_type = ?"; args = append(args, *req.BowType) }
		if req.AvatarURL != nil { query += ", avatar_url = ?"; args = append(args, *req.AvatarURL) }

		if len(args) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "Tidak ada data yang diperbarui"})
			return
		}

		query += " WHERE uuid = ?"
		args = append(args, userID)

		_, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui profil", "details": err.Error()})
			return
		}

		utils.LogActivity(db, userID, "", "mobile_profile_updated", "archer", userID, "Updated profile via mobile", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Profil berhasil diperbarui"})
	}
}

// MobileOrganizationScanRegistration handles QR scan for check-in
// @Summary Scan Registration QR
// @Description Scan and verify a participant's QR code for event check-in/reregistration
// @Tags Mobile - Organization
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileOrganizationScanRegistrationRequest true "QR Code"
// @Success 200 {object} MobileOrganizationScanRegistrationResponse
// @Router /mobile/organization/scan [post]
func MobileOrganizationScanRegistration(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "organization") {
			return
		}

		organizationUUID, ok := getMobileOrganizationUUID(c, db)
		if !ok {
			return
		}

		var req MobileOrganizationScanRegistrationRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Permintaan tidak valid", "details": err.Error()})
			return
		}

		var resp MobileOrganizationScanRegistrationResponse
		query := `
			SELECT 
				ep.uuid as participant_uuid,
				COALESCE(a.full_name, '') as full_name,
				COALESCE(a.id, '') as athlete_code,
				e.name as event_name,
				COALESCE(ec.category_name_custom, r_ag.name, '') as category_name,
				cl.name as club_name,
				COALESCE(ep.payment_status, 'unpaid') as payment_status,
				ep.last_reregistration_at
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups r_ag ON ec.category_uuid = r_ag.uuid
			WHERE ep.qr_raw = ? AND e.organizer_id = ?
			LIMIT 1
		`
		err := db.Get(&resp, query, req.Code, organizationUUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data pendaftaran tidak ditemukan atau Anda tidak memiliki akses ke event ini"})
			return
		}

		// Update reregistration time
		_, err = db.Exec(`UPDATE event_participants SET last_reregistration_at = NOW() WHERE uuid = ?`, resp.ParticipantUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data pendaftaran", "details": err.Error()})
			return
		}

		// Update response timestamp to "now" for immediate feedback
		now := time.Now().Format("2006-01-02T15:04:05Z")
		resp.LastReregistrationAt = &now

		utils.LogActivity(db, organizationUUID, "", "mobile_reregistration_scan", "organization", organizationUUID, "Scanned QR for reregistration: "+resp.ParticipantUUID, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, resp)
	}
}

// MobileGetOrganizationParticipantDetail returns detailed information about a participant
// @Summary Get Organization Participant Detail
// @Description Get full details of a specific participant for an organization event
// @Tags Mobile - Organization
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Event UUID"
// @Param user_id path string true "Archer UUID or Participant UUID"
// @Success 200 {object} MobileOrganizationParticipantDetail
// @Router /mobile/organization/events/{id}/participants/{user_id} [get]
func MobileGetOrganizationParticipantDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userIDParam := c.Param("user_id")

		query := `
			SELECT 
				tp.uuid as participant_uuid, a.uuid as archer_uuid, a.full_name, a.gender, a.birth_date, 
				a.email, a.phone, a.avatar_url, cl.name as club_name,
				ec.uuid as category_uuid, COALESCE(ec.category_name_custom, r_ag.name, '') as category_name,
				tp.target_name, tp.back_number, tp.payment_status, tp.payment_amount,
				tp.registration_date, tp.last_reregistration_at
			FROM event_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories ec ON tp.category_id = ec.uuid
			LEFT JOIN ref_age_groups r_ag ON ec.category_uuid = r_ag.uuid
			WHERE tp.event_id = ? AND (tp.uuid = ? OR tp.archer_id = ?)
			LIMIT 1
		`

		var raw struct {
			ParticipantUUID      string         `db:"participant_uuid"`
			ArcherUUID           string         `db:"archer_uuid"`
			FullName             string         `db:"full_name"`
			Gender               *string        `db:"gender"`
			BirthDate            *string        `db:"birth_date"`
			Email                *string        `db:"email"`
			Phone                *string        `db:"phone"`
			AvatarURL            *string        `db:"avatar_url"`
			ClubName             *string        `db:"club_name"`
			CategoryUUID         string         `db:"category_uuid"`
			CategoryName         string         `db:"category_name"`
			TargetName           *string        `db:"target_name"`
			BackNumber           *string        `db:"back_number"`
			PaymentStatus        string         `db:"payment_status"`
			PaymentAmount        float64        `db:"payment_amount"`
			RegistrationDate     time.Time      `db:"registration_date"`
			LastReregistrationAt *time.Time     `db:"last_reregistration_at"`
		}

		err := db.Get(&raw, query, eventID, userIDParam, userIDParam)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Peserta tidak ditemukan"})
			return
		}

		proofs := []string{}

		var lastRereg *string
		if raw.LastReregistrationAt != nil {
			s := raw.LastReregistrationAt.Format(time.RFC3339)
			lastRereg = &s
		}

		if raw.AvatarURL != nil {
			masked := utils.MaskMediaURL(*raw.AvatarURL)
			raw.AvatarURL = &masked
		}

		detail := MobileOrganizationParticipantDetail{
			ParticipantUUID:      raw.ParticipantUUID,
			ArcherUUID:           raw.ArcherUUID,
			FullName:             raw.FullName,
			Gender:               raw.Gender,
			BirthDate:            raw.BirthDate,
			Email:                raw.Email,
			Phone:                raw.Phone,
			AvatarURL:            raw.AvatarURL,
			ClubName:             raw.ClubName,
			CategoryUUID:         raw.CategoryUUID,
			CategoryName:         raw.CategoryName,
			TargetName:           raw.TargetName,
			BackNumber:           raw.BackNumber,
			PaymentStatus:        raw.PaymentStatus,
			PaymentAmount:        raw.PaymentAmount,
			PaymentProofURLs:     proofs,
			RegistrationDate:     raw.RegistrationDate.Format(time.RFC3339),
			LastReregistrationAt: lastRereg,
			CheckInStatus:        raw.LastReregistrationAt != nil,
		}

		c.JSON(http.StatusOK, detail)
	}
}

