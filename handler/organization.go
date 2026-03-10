package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"archeryhub-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Organization represents an organization entity
type Organization struct {
	UUID               string  `db:"uuid" json:"id"`
	Slug               *string `db:"slug" json:"slug"`
	Name               string  `db:"name" json:"name"`
	Acronym            *string `db:"acronym" json:"acronym"`
	Description        *string `db:"description" json:"description"`
	Vision             *string `db:"vision" json:"vision"`
	Mission            *string `db:"mission" json:"mission"`
	History            *string `db:"history" json:"history"`
	Website            *string `db:"website" json:"website"`
	Email              string  `db:"email" json:"email"`
	WhatsAppNo         *string `db:"whatsapp_no" json:"whatsapp_no"`
	AvatarURL          *string `db:"avatar_url" json:"avatar_url"`
	BannerURL          *string `db:"banner_url" json:"banner_url"`
	Address            *string `db:"address" json:"address"`
	City               *string `db:"city" json:"city"`
	Country            *string `db:"country" json:"country"`
	RegistrationNumber *string `db:"registration_number" json:"registration_number"`
	EstablishedDate    *string `db:"established_date" json:"established_date"`
	ContactPersonName  *string `db:"contact_person_name" json:"contact_person_name"`
	ContactPersonEmail *string `db:"contact_person_email" json:"contact_person_email"`
	ContactPersonPhone *string `db:"contact_person_phone" json:"contact_person_phone"`
	SocialFacebook     *string `db:"social_facebook" json:"social_facebook"`
	SocialInstagram    *string `db:"social_instagram" json:"social_instagram"`
	SocialTwitter      *string `db:"social_twitter" json:"social_twitter"`
	SocialMedia        *string `db:"social_media" json:"social_media"`
	VerificationStatus *string `db:"verification_status" json:"verification_status"`
	Status             *string `db:"status" json:"status"`
	CreatedAt          string  `db:"created_at" json:"created_at"`
	UpdatedAt          string  `db:"updated_at" json:"updated_at"`
	FAQ                *string `db:"faq" json:"faq"`
}

// GetOrganizations returns all organizations (public)
func GetOrganizations(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		search := c.Query("search")
		status := c.DefaultQuery("status", "active")

		query := `
			SELECT uuid, slug, name, acronym, description, vision, mission, history, website, email, whatsapp_no,
				   avatar_url, banner_url, address, city, country,
				   verification_status, status, created_at, social_media
			FROM organizations
			WHERE status = ?
		`
		args := []interface{}{status}

		if search != "" {
			query += " AND (name LIKE ? OR acronym LIKE ? OR city LIKE ?)"
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm, searchTerm)
		}

		query += " ORDER BY name ASC"

		var orgs []Organization
		err := db.Select(&orgs, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch organizations", "details": err.Error()})
			return
		}

		for i := range orgs {
			if orgs[i].AvatarURL != nil {
				masked := utils.MaskMediaURL(*orgs[i].AvatarURL)
				orgs[i].AvatarURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"organizations": orgs,
			"total":         len(orgs),
		})
	}
}

// GetOrganizationBySlug returns a single organization by username/slug (public)
func GetOrganizationBySlug(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		// Struct with page_settings
		var orgData struct {
			Organization
			PageSettings *string `db:"page_settings" json:"page_settings"`
		}

		err := db.Get(&orgData, `
			SELECT uuid, slug, name, acronym, description, website, email, whatsapp_no,
				   avatar_url, banner_url, address, city, country,
				   registration_number, established_date, contact_person_name,
				   contact_person_email, contact_person_phone,
				   social_facebook, social_instagram, social_twitter, social_media,
				   verification_status, status, created_at, updated_at, page_settings,
				   vision, mission, history, faq
			FROM organizations
			WHERE (slug = ? OR uuid = ?) AND status = 'active'
		`, slug, slug)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found", "details": err.Error()})
			return
		}

		org := orgData.Organization

		// Mask URLs
		if org.AvatarURL != nil {
			masked := utils.MaskMediaURL(*org.AvatarURL)
			org.AvatarURL = &masked
		}
		if org.BannerURL != nil {
			masked := utils.MaskMediaURL(*org.BannerURL)
			org.BannerURL = &masked
		}

		// Get events organized by this organization with pagination
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
		if limit < 1 {
			limit = 5
		}
		offset := (page - 1) * limit

		var totalEvents int
		db.Get(&totalEvents, "SELECT COUNT(*) FROM events WHERE organizer_id = ? AND status IN ('published', 'ongoing', 'completed')", org.UUID)

		var events []struct {
			UUID      string  `db:"uuid" json:"id"`
			Name      string  `db:"name" json:"name"`
			Slug      string  `db:"slug" json:"slug"`
			StartDate *string `db:"start_date" json:"start_date"`
			EndDate   *string `db:"end_date" json:"end_date"`
			Venue     *string `db:"venue" json:"venue"`
			Status    *string `db:"status" json:"status"`
			LogoURL   *string `db:"logo_url" json:"logo_url"`
		}
		db.Select(&events, `
			SELECT uuid, name, slug, start_date, end_date, venue, status, logo_url
			FROM events
			WHERE organizer_id = ? AND status IN ('published', 'ongoing', 'completed')
			ORDER BY start_date DESC
			LIMIT ? OFFSET ?
		`, org.UUID, limit, offset)

		for i := range events {
			if events[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*events[i].LogoURL)
				events[i].LogoURL = &masked
			}
		}

		// Get affiliated clubs
		var clubs []struct {
			UUID    string  `db:"uuid" json:"id"`
			Name    string  `db:"name" json:"name"`
			Slug    string  `db:"slug" json:"slug"`
			City    *string `db:"city" json:"city"`
			LogoURL *string `db:"logo_url" json:"logo_url"`
			MemberCount int `db:"member_count" json:"member_count"`
		}
		db.Select(&clubs, `
			SELECT c.uuid, c.name, c.slug, c.city, c.logo_url,
				   (SELECT COUNT(*) FROM club_members WHERE club_id = c.uuid AND status = 'active') as member_count
			FROM clubs c
			WHERE c.organization_id = ? AND c.status = 'active'
			ORDER BY c.name ASC
		`, org.UUID)

		for i := range clubs {
			if clubs[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*clubs[i].LogoURL)
				clubs[i].LogoURL = &masked
			}
		}

		// Build response with page_settings
		response := gin.H{
			"organization": gin.H{
				"id":                   org.UUID,
				"slug":                 org.Slug,
				"name":                 org.Name,
				"acronym":              org.Acronym,
				"description":          org.Description,
				"website":              org.Website,
				"email":                org.Email,
				"whatsapp_no":          org.WhatsAppNo,
				"avatar_url":           org.AvatarURL,
				"banner_url":           org.BannerURL,
				"address":              org.Address,
				"city":                 org.City,
				"country":              org.Country,
				"registration_number":  org.RegistrationNumber,
				"established_date":     org.EstablishedDate,
				"contact_person_name":  org.ContactPersonName,
				"contact_person_email": org.ContactPersonEmail,
				"contact_person_phone": org.ContactPersonPhone,
				"social_facebook":      org.SocialFacebook,
				"social_instagram":     org.SocialInstagram,
				"social_twitter":       org.SocialTwitter,
				"social_media":         org.SocialMedia,
				"vision":               org.Vision,
				"mission":              org.Mission,
				"history":               org.History,
				"verification_status":  org.VerificationStatus,
				"status":               org.Status,
				"created_at":           org.CreatedAt,
				"updated_at":           org.UpdatedAt,
			},
			"events":       events,
			"total_events": totalEvents,
			"clubs":        clubs,
		}

		// Add FAQ if exists
		if orgData.FAQ != nil && *orgData.FAQ != "" {
			var faq []interface{}
			if err := json.Unmarshal([]byte(*orgData.FAQ), &faq); err == nil {
				response["organization"].(gin.H)["faq"] = faq
			}
		}

		// Add page_settings if exists
		if orgData.PageSettings != nil && *orgData.PageSettings != "" {
			var pageSettings map[string]interface{}
			if err := json.Unmarshal([]byte(*orgData.PageSettings), &pageSettings); err == nil {
				response["organization"].(gin.H)["page_settings"] = pageSettings
			}
		}

		c.JSON(http.StatusOK, response)
	}
}

// GetOrganizationProfile returns the current user's organization profile (protected)
func GetOrganizationProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var org struct {
			Organization
			SubscriptionStatus    string  `db:"subscription_status" json:"subscription_status"`
			SubscriptionExpiresAt *string `db:"subscription_expires_at" json:"subscription_expires_at"`
		}
		var pageSettings *string
		err := db.Get(&org, `
			SELECT uuid, slug, name, acronym, description, website, email, whatsapp_no,
				   avatar_url, banner_url, address, city, country,
				   registration_number, established_date, contact_person_name,
				   contact_person_email, contact_person_phone,
				   social_facebook, social_instagram, social_twitter, social_media,
				   verification_status, status, created_at, updated_at,
				   vision, mission, history, faq,
				   COALESCE(subscription_status, 'trial') as subscription_status,
				   subscription_expires_at
			FROM organizations
			WHERE uuid = ?
		`, userID)

		if err == nil {
			db.Get(&pageSettings, "SELECT page_settings FROM organizations WHERE uuid = ?", userID)
		}

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Organization not found"})
			return
		}

		// Mask URLs
		if org.AvatarURL != nil {
			masked := utils.MaskMediaURL(*org.AvatarURL)
			org.AvatarURL = &masked
		}
		if org.BannerURL != nil {
			masked := utils.MaskMediaURL(*org.BannerURL)
			org.BannerURL = &masked
		}

		// Prepare response
		data := gin.H{
			"id":                  org.UUID,
			"uuid":                org.UUID,
			"slug":                org.Slug,
			"name":                org.Name,
			"acronym":             org.Acronym,
			"description":         org.Description,
			"website":             org.Website,
			"email":               org.Email,
			"whatsapp_no":         org.WhatsAppNo,
			"avatar_url":          org.AvatarURL,
			"banner_url":          org.BannerURL,
			"address":             org.Address,
			"city":                org.City,
			"country":             org.Country,
			"registration_number": org.RegistrationNumber,
			"established_date":    org.EstablishedDate,
			"contact_person_name": org.ContactPersonName,
			"contact_person_email": org.ContactPersonEmail,
			"contact_person_phone": org.ContactPersonPhone,
			"social_facebook":     org.SocialFacebook,
			"social_instagram":    org.SocialInstagram,
			"social_twitter":      org.SocialTwitter,
			"social_media":        org.SocialMedia,
			"vision":              org.Vision,
			"mission":             org.Mission,
			"history":             org.History,
			"faq":                 org.FAQ,
			"verification_status": org.VerificationStatus,
			"status":                org.Status,
			"created_at":            org.CreatedAt,
			"updated_at":            org.UpdatedAt,
			"subscription_status":    org.SubscriptionStatus,
			"subscription_expires_at": org.SubscriptionExpiresAt,
			"user_type":             "organization",
		}

		if pageSettings != nil {
			data["page_settings"] = pageSettings
		}

		c.JSON(http.StatusOK, gin.H{"data": data})
	}
}

// UpdateOrganizationProfile updates the current user's organization profile (protected)
func UpdateOrganizationProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var req struct {
			Slug               *string     `json:"slug"`
			Name               *string     `json:"name"`
			Acronym            *string     `json:"acronym"`
			Description        *string     `json:"description"`
			Website            *string     `json:"website"`
			WhatsAppNo         *string     `json:"whatsapp_no"`
			AvatarURL          *string     `json:"avatar_url"`
			BannerURL          *string     `json:"banner_url"`
			Address            *string     `json:"address"`
			City               *string     `json:"city"`
			Country            *string     `json:"country"`
			RegistrationNumber *string     `json:"registration_number"`
			EstablishedDate    *string     `json:"established_date"`
			ContactPersonName  *string     `json:"contact_person_name"`
			ContactPersonEmail *string     `json:"contact_person_email"`
			ContactPersonPhone *string     `json:"contact_person_phone"`
			SocialFacebook     *string     `json:"social_facebook"`
			SocialInstagram    *string     `json:"social_instagram"`
			SocialTwitter      *string     `json:"social_twitter"`
			SocialMedia        interface{} `json:"social_media"`
			Vision             *string     `json:"vision"`
			Mission            *string     `json:"mission"`
			History            *string     `json:"history"`
			FAQ                interface{} `json:"faq"`
			PageSettings       interface{} `json:"page_settings"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Build dynamic update query
		query := "UPDATE organizations SET updated_at = NOW()"
		args := []interface{}{}

		if req.Slug != nil {
			// Check if slug is already taken
			var exists bool
			db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM organizations WHERE slug = ? AND uuid != ?)", *req.Slug, userID)
			if exists {
				c.JSON(http.StatusConflict, gin.H{"error": "Slug already taken"})
				return
			}
			query += ", slug = ?"
			args = append(args, strings.ToLower(*req.Slug))
		}
		if req.Name != nil {
			query += ", name = ?"
			args = append(args, *req.Name)
		}
		if req.Acronym != nil {
			query += ", acronym = ?"
			args = append(args, *req.Acronym)
		}
		if req.Description != nil {
			query += ", description = ?"
			args = append(args, *req.Description)
		}
		if req.Website != nil {
			query += ", website = ?"
			args = append(args, *req.Website)
		}
		if req.WhatsAppNo != nil {
			query += ", whatsapp_no = ?"
			args = append(args, *req.WhatsAppNo)
		}
		if req.AvatarURL != nil {
			query += ", avatar_url = ?"
			args = append(args, utils.ExtractFilename(*req.AvatarURL))
		}
		if req.BannerURL != nil {
			query += ", banner_url = ?"
			args = append(args, utils.ExtractFilename(*req.BannerURL))
		}
		if req.Address != nil {
			query += ", address = ?"
			args = append(args, *req.Address)
		}
		if req.City != nil {
			query += ", city = ?"
			args = append(args, *req.City)
		}
		if req.Country != nil {
			query += ", country = ?"
			args = append(args, *req.Country)
		}
		if req.RegistrationNumber != nil {
			query += ", registration_number = ?"
			args = append(args, *req.RegistrationNumber)
		}
		if req.EstablishedDate != nil {
			query += ", established_date = ?"
			args = append(args, *req.EstablishedDate)
		}
		if req.ContactPersonName != nil {
			query += ", contact_person_name = ?"
			args = append(args, *req.ContactPersonName)
		}
		if req.ContactPersonEmail != nil {
			query += ", contact_person_email = ?"
			args = append(args, *req.ContactPersonEmail)
		}
		if req.ContactPersonPhone != nil {
			query += ", contact_person_phone = ?"
			args = append(args, *req.ContactPersonPhone)
		}
		if req.SocialFacebook != nil {
			query += ", social_facebook = ?"
			args = append(args, *req.SocialFacebook)
		}
		if req.SocialInstagram != nil {
			query += ", social_instagram = ?"
			args = append(args, *req.SocialInstagram)
		}
		if req.SocialTwitter != nil {
			query += ", social_twitter = ?"
			args = append(args, *req.SocialTwitter)
		}
		if req.Vision != nil {
			query += ", vision = ?"
			args = append(args, *req.Vision)
		}
		mission := req.Mission
		if mission != nil {
			query += ", mission = ?"
			args = append(args, *mission)
		}
		if req.History != nil {
			query += ", history = ?"
			args = append(args, *req.History)
		}

		// Handle social_media JSON
		if req.SocialMedia != nil {
			socialMediaJSON, _ := json.Marshal(req.SocialMedia)
			query += ", social_media = ?"
			args = append(args, string(socialMediaJSON))
		}

		// Handle faq JSON
		if req.FAQ != nil {
			faqJSON, _ := json.Marshal(req.FAQ)
			query += ", faq = ?"
			args = append(args, string(faqJSON))
		}

		// Handle page_settings JSON
		if req.PageSettings != nil {
			var pageSettingsMap map[string]interface{}
			if pageSettingsStr, ok := req.PageSettings.(string); ok {
				json.Unmarshal([]byte(pageSettingsStr), &pageSettingsMap)
			} else {
				pageSettingsBytes, _ := json.Marshal(req.PageSettings)
				json.Unmarshal(pageSettingsBytes, &pageSettingsMap)
			}
			pageSettingsJSON, _ := json.Marshal(pageSettingsMap)
			query += ", page_settings = ?"
			args = append(args, string(pageSettingsJSON))
		}

		if len(args) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "No changes to save"})
			return
		}

		query += " WHERE uuid = ?"
		args = append(args, userID)

		_, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update organization", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Organization updated successfully"})
	}
}

// GetOrganizationDashboardStats returns aggregated statistics for the organization's dashboard
func GetOrganizationDashboardStats(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var stats struct {
			TotalArchers       int     `json:"totalArchers"`
			ActiveTargets      int     `json:"activeTargets"`
			ActiveTotalTargets int     `json:"activeTotalTargets"`
			CompletionRate     float64 `json:"completionRate"`
			TimeLeft           string  `json:"timeLeft"`
			RecentActiveEvent  *string `json:"recentActiveEvent"`
		}

		// 1. Total Unique Archers managed by this organization
		// These are archers who have participated in any event organized by this organization
		_ = db.Get(&stats.TotalArchers, `
			SELECT COUNT(DISTINCT ep.archer_id) 
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			WHERE e.organizer_id = ?
		`, userID)

		// 2. Active Stats (from ongoing events)
		var ongoingEventID string
		err := db.Get(&ongoingEventID, `
			SELECT uuid FROM events 
			WHERE organizer_id = ? AND status = 'ongoing' 
			ORDER BY updated_at DESC LIMIT 1
		`, userID)

		if err == nil && ongoingEventID != "" {
			stats.RecentActiveEvent = &ongoingEventID

			// Total targets in this event
			_ = db.Get(&stats.ActiveTotalTargets, "SELECT COUNT(*) FROM event_targets WHERE event_uuid = ?", ongoingEventID)

			// Occupied targets (targets with assignments)
			_ = db.Get(&stats.ActiveTargets, `
				SELECT COUNT(DISTINCT et.uuid) 
				FROM event_targets et
				JOIN qualification_target_assignments qta ON et.uuid = qta.target_uuid
				WHERE et.event_uuid = ?
			`, ongoingEventID)

			// Completion calculation (Qualification ends)
			var totalParticipants int
			_ = db.Get(&totalParticipants, "SELECT COUNT(*) FROM event_participants WHERE event_id = ?", ongoingEventID)

			var totalEndsForEvent int = 12 // Default
			var qualificationArrows int
			_ = db.Get(&qualificationArrows, "SELECT qualification_arrows FROM events WHERE uuid = ?", ongoingEventID)
			if qualificationArrows > 0 {
				totalEndsForEvent = (qualificationArrows + 5) / 6
			}

			if totalParticipants > 0 {
				var completedEnds int
				_ = db.Get(&completedEnds, `
					SELECT COUNT(*) FROM qualification_end_scores qes
					JOIN event_participants ep ON qes.participant_uuid = ep.uuid
					WHERE ep.event_id = ?
				`, ongoingEventID)

				totalExpectedEnds := totalParticipants * totalEndsForEvent
				if totalExpectedEnds > 0 {
					stats.CompletionRate = (float64(completedEnds) / float64(totalExpectedEnds)) * 100
				}
			}

			// Estimated Time Left (Stub for now, or based on ends)
			stats.TimeLeft = "Ongoing"
		} else {
			// If no ongoing event, return 0s
			stats.ActiveTargets = 0
			stats.ActiveTotalTargets = 0
			stats.CompletionRate = 0
			stats.TimeLeft = "-"
		}

		c.JSON(http.StatusOK, stats)
	}
}
