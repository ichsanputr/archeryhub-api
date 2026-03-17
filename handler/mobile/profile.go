package mobile

import (
	"archeryhub-api/handler"
	"archeryhub-api/models"
	"archeryhub-api/utils"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

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

// MobileGetSellerMe godoc
// @Summary      Get mobile seller profile
// @Description  Get the authenticated seller profile for mobile pages
// @Tags         Mobile - Seller
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MobileSellerProfileResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /mobile/seller/me [get]
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

// MobileUpdateSellerMe godoc
// @Summary      Update mobile seller profile
// @Description  Update the authenticated seller basic profile for mobile pages
// @Tags         Mobile - Seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      MobileUpdateSellerProfileBasicRequest  true  "Seller profile payload"
// @Success      200      {object}  MessageResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Router       /mobile/seller/me [put]
func MobileUpdateSellerMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "seller") {
			return
		}
		handler.UpdateSellerProfileBasic(db)(c)
	}
}

// MobileUpdateSellerPage godoc
// @Summary      Update mobile seller page settings
// @Description  Update the authenticated seller page settings for mobile pages
// @Tags         Mobile - Seller
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      MobileUpdateSellerPageRequest  true  "Seller page settings payload"
// @Success      200      {object}  MessageResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Router       /mobile/seller/me/page [put]
func MobileUpdateSellerPage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "seller") {
			return
		}
		handler.UpdateSellerProfile(db)(c)
	}
}

// MobileGetSellerProducts godoc
// @Summary      List seller products for mobile
// @Description  Get products owned by the authenticated seller for mobile pages
// @Tags         Mobile - Seller
// @Produce      json
// @Security     BearerAuth
// @Param        limit     query     int     false  "Limit"   default(20)
// @Param        offset    query     int     false  "Offset"  default(0)
// @Param        status    query     string  false  "Product status filter"
// @Param        category  query     string  false  "Product category filter"
// @Param        search    query     string  false  "Search by product name or description"
// @Success      200       {object}  MobileSellerProductsResponse
// @Failure      403       {object}  ErrorResponse
// @Failure      404       {object}  ErrorResponse
// @Router       /mobile/seller/products [get]
func MobileGetSellerProducts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requireMobileUserType(c, "seller") {
			return
		}

		userID := c.GetString("user_id")
		var sellerUUID string
		if err := db.Get(&sellerUUID, `SELECT uuid FROM sellers WHERE uuid = ? OR user_id = ? LIMIT 1`, userID, userID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penjual tidak ditemukan"})
			return
		}

		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		status := strings.TrimSpace(c.Query("status"))
		category := strings.TrimSpace(c.Query("category"))
		search := strings.TrimSpace(c.Query("search"))

		whereClause := "WHERE seller_id = ?"
		args := []interface{}{sellerUUID}
		if status != "" && status != "all" {
			whereClause += " AND status = ?"
			args = append(args, status)
		}
		if category != "" && category != "all" {
			whereClause += " AND category = ?"
			args = append(args, category)
		}
		if search != "" {
			searchTerm := "%" + search + "%"
			whereClause += " AND (name LIKE ? OR description LIKE ?)"
			args = append(args, searchTerm, searchTerm)
		}

		var total int
		if err := db.Get(&total, "SELECT COUNT(*) FROM products "+whereClause, args...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung produk", "details": err.Error()})
			return
		}

		products := []models.Product{}
		query := "SELECT * FROM products " + whereClause + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
		queryArgs := append(args, limit, offset)
		if err := db.Select(&products, query, queryArgs...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data produk", "details": err.Error()})
			return
		}

		if products == nil {
			products = []models.Product{}
		}
		for i := range products {
			maskProductMedia(&products[i])
		}

		c.JSON(http.StatusOK, MobileSellerProductsResponse{
			Products: products,
			Total:    total,
			Limit:    limit,
			Offset:   offset,
		})
	}
}

// MobileGetOrganizationMe godoc
// @Summary      Get mobile organization profile
// @Description  Get the authenticated organization profile for mobile pages
// @Tags         Mobile - Organization
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MobileOrganizationProfileResponse
// @Failure      403  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Router       /mobile/organization/me [get]
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
			VerificationStatus    *string `db:"verification_status"`
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
			       verification_status, status, created_at, updated_at,
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
				VerificationStatus:    org.VerificationStatus,
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

// MobileUpdateOrganizationMe godoc
// @Summary      Update mobile organization profile
// @Description  Update the authenticated organization profile for mobile pages
// @Tags         Mobile - Organization
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request  body      MobileUpdateOrganizationProfileRequest  true  "Organization profile payload"
// @Success      200      {object}  MessageResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Router       /mobile/organization/me [put]
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

// MobileGetOrganizationEvents godoc
// @Summary      List organization events for mobile
// @Description  Get events owned by the authenticated organization for mobile pages
// @Tags         Mobile - Organization
// @Produce      json
// @Security     BearerAuth
// @Param        limit   query     int     false  "Limit"   default(20)
// @Param        offset  query     int     false  "Offset"  default(0)
// @Param        status  query     string  false  "Event status filter"
// @Param        search  query     string  false  "Search by event name, code, or location"
// @Success      200     {object}  MobileOrganizationEventsResponse
// @Failure      403     {object}  ErrorResponse
// @Failure      404     {object}  ErrorResponse
// @Router       /mobile/organization/events [get]
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
					SUM(CASE WHEN payment_status = 'lunas' THEN 1 ELSE 0 END) as verified_count,
					SUM(CASE WHEN payment_status = 'menunggu acc' THEN 1 ELSE 0 END) as pending_count
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

// MobileGetOrganizationEventParticipants godoc
// @Summary      List event participants with QR code for mobile organization
// @Description  Get participants for an organization-owned event including QR code data URLs
// @Tags         Mobile - Organization
// @Produce      json
// @Security     BearerAuth
// @Param        id              path      string  true   "Event UUID or slug"
// @Param        limit           query     int     false  "Limit"   default(20)
// @Param        offset          query     int     false  "Offset"  default(0)
// @Param        search          query     string  false  "Search by participant or club name"
// @Param        payment_status  query     string  false  "Payment status filter"
// @Param        category_id     query     string  false  "Category UUID filter"
// @Success      200             {object}  MobileOrganizationEventParticipantsResponse
// @Failure      403             {object}  ErrorResponse
// @Failure      404             {object}  ErrorResponse
// @Router       /mobile/organization/events/{id}/participants [get]
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
			whereClause += " AND tp.payment_status = ?"
			args = append(args, paymentStatus)
			countArgs = append(countArgs, paymentStatus)
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
				COALESCE(tp.payment_status, 'menunggu acc') as payment_status
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
		_ = db.Get(&verifiedCount, "SELECT COUNT(*) FROM event_participants "+statusWhere+" AND payment_status = 'lunas'", statusArgs...)
		_ = db.Get(&pendingCount, "SELECT COUNT(*) FROM event_participants "+statusWhere+" AND payment_status = 'menunggu acc'", statusArgs...)

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
