package handler

import (
	"net/http"
	"strings"

	"archeryhub-api/utils"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// Organization represents an organization entity
type Organization struct {
	UUID               string  `db:"uuid" json:"id"`
	Username           *string `db:"username" json:"username"`
	Name               string  `db:"name" json:"name"`
	Acronym            *string `db:"acronym" json:"acronym"`
	Description        *string `db:"description" json:"description"`
	Website            *string `db:"website" json:"website"`
	Email              string  `db:"email" json:"email"`
	Phone              *string `db:"phone" json:"phone"`
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
	VerificationStatus *string `db:"verification_status" json:"verification_status"`
	Status             *string `db:"status" json:"status"`
	CreatedAt          string  `db:"created_at" json:"created_at"`
	UpdatedAt          string  `db:"updated_at" json:"updated_at"`
}

// GetOrganizations returns all organizations (public)
func GetOrganizations(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		search := c.Query("search")
		status := c.DefaultQuery("status", "active")

		query := `
			SELECT uuid, username, name, acronym, description, website, email, phone,
				   avatar_url, banner_url, address, city, country,
				   verification_status, status, created_at
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
			if orgs[i].BannerURL != nil {
				masked := utils.MaskMediaURL(*orgs[i].BannerURL)
				orgs[i].BannerURL = &masked
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

		var org Organization
		err := db.Get(&org, `
			SELECT uuid, username, name, acronym, description, website, email, phone,
				   avatar_url, banner_url, address, city, country,
				   registration_number, established_date, contact_person_name,
				   contact_person_email, contact_person_phone,
				   social_facebook, social_instagram, social_twitter,
				   verification_status, status, created_at, updated_at
			FROM organizations
			WHERE (username = ? OR uuid = ?) AND status = 'active'
		`, slug, slug)

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

		// Get events organized by this organization
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
			WHERE organization_id = ? AND status IN ('published', 'live', 'completed')
			ORDER BY start_date DESC
			LIMIT 10
		`, org.UUID)

		for i := range events {
			if events[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*events[i].LogoURL)
				events[i].LogoURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"organization": org,
			"events":       events,
		})
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

		var org Organization
		err := db.Get(&org, `
			SELECT uuid, username, name, acronym, description, website, email, phone,
				   avatar_url, banner_url, address, city, country,
				   registration_number, established_date, contact_person_name,
				   contact_person_email, contact_person_phone,
				   social_facebook, social_instagram, social_twitter,
				   verification_status, status, created_at, updated_at
			FROM organizations
			WHERE uuid = ?
		`, userID)

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

		c.JSON(http.StatusOK, gin.H{"organization": org})
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
			Username           *string `json:"username"`
			Name               *string `json:"name"`
			Acronym            *string `json:"acronym"`
			Description        *string `json:"description"`
			Website            *string `json:"website"`
			Phone              *string `json:"phone"`
			AvatarURL          *string `json:"avatar_url"`
			BannerURL          *string `json:"banner_url"`
			Address            *string `json:"address"`
			City               *string `json:"city"`
			Country            *string `json:"country"`
			RegistrationNumber *string `json:"registration_number"`
			EstablishedDate    *string `json:"established_date"`
			ContactPersonName  *string `json:"contact_person_name"`
			ContactPersonEmail *string `json:"contact_person_email"`
			ContactPersonPhone *string `json:"contact_person_phone"`
			SocialFacebook     *string `json:"social_facebook"`
			SocialInstagram    *string `json:"social_instagram"`
			SocialTwitter      *string `json:"social_twitter"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
			return
		}

		// Build dynamic update query
		query := "UPDATE organizations SET updated_at = NOW()"
		args := []interface{}{}

		if req.Username != nil {
			// Check if username is already taken
			var exists bool
			db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM organizations WHERE username = ? AND uuid != ?)", *req.Username, userID)
			if exists {
				c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
				return
			}
			query += ", username = ?"
			args = append(args, strings.ToLower(*req.Username))
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
		if req.Phone != nil {
			query += ", phone = ?"
			args = append(args, *req.Phone)
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
