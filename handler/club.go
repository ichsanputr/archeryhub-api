package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"archeryhub-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// CheckSlugAvailability checks if a club slug is available
func CheckSlugAvailability(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Query("slug")
		if slug == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "slug is required"})
			return
		}

		// Get current user's club UUID to exclude from check
		userID, _ := c.Get("user_id")
		var currentClubUUID string
		db.Get(&currentClubUUID, "SELECT uuid FROM clubs WHERE uuid = ?", userID)

		var count int
		query := "SELECT COUNT(*) FROM clubs WHERE slug = ?"
		args := []interface{}{slug}

		if currentClubUUID != "" {
			query += " AND uuid != ?"
			args = append(args, currentClubUUID)
		}

		err := db.Get(&count, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check slug"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"available": count == 0, "slug": slug})
	}
}

// GetClubMe returns the club profile for the authenticated user
func GetClubMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var club struct {
			UUID             string  `json:"uuid" db:"uuid"`
			Name             string  `json:"name" db:"name"`
			Slug             string  `json:"slug" db:"slug"`
			SlugChanged      bool    `json:"slug_changed" db:"slug_changed"`
			Description      *string `json:"description" db:"description"`
			AvatarURL        *string `json:"avatar_url" db:"avatar_url"`
			BannerURL        *string `json:"banner_url" db:"banner_url"`
			LogoURL          *string `json:"logo_url" db:"logo_url"`
			Address          *string `json:"address" db:"address"`
			City             *string `json:"city" db:"city"`
			Province         *string `json:"province" db:"province"`
			Phone            *string `json:"phone" db:"phone"`
			Email            *string `json:"email" db:"email"`
			Website          *string `json:"website" db:"website"`
			Facebook         *string `json:"facebook" db:"social_facebook"`
			Instagram        *string `json:"instagram" db:"social_instagram"`
			WhatsApp         *string `json:"whatsapp" db:"-"`
			EstablishedDate  *string `json:"established" db:"established_date"`
			Facilities       *string `json:"facilities" db:"facilities"`
			TrainingSchedule *string `json:"schedules" db:"training_schedule"`
			SocialMedia      *string `json:"social_media" db:"social_media"`
			PageSettings     *string `json:"page_settings" db:"page_settings"`
		}

		err := db.Get(&club, `
			SELECT uuid, name, slug, COALESCE(slug_changed, 0) as slug_changed, description, avatar_url, banner_url, logo_url, address, city, province, phone, email, website, social_facebook, social_instagram, established_date, facilities, training_schedule, social_media, page_settings 
			FROM clubs 
			WHERE uuid = ?`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error or club not found: " + err.Error()})
			return
		}

		// Mask URLs
		if club.AvatarURL != nil {
			masked := utils.MaskMediaURL(*club.AvatarURL)
			club.AvatarURL = &masked
		}
		if club.BannerURL != nil {
			masked := utils.MaskMediaURL(*club.BannerURL)
			club.BannerURL = &masked
		}
		if club.LogoURL != nil {
			masked := utils.MaskMediaURL(*club.LogoURL)
			club.LogoURL = &masked
		}

		c.JSON(http.StatusOK, gin.H{
			"uuid":          club.UUID,
			"name":          club.Name,
			"slug":          club.Slug,
			"slug_changed":  club.SlugChanged,
			"description":   club.Description,
			"avatar_url":    club.AvatarURL,
			"banner_url":    club.BannerURL,
			"logo_url":      club.LogoURL,
			"address":       club.Address,
			"city":          club.City,
			"province":      club.Province,
			"phone":         club.Phone,
			"email":         club.Email,
			"website":       club.Website,
			"facebook":      club.Facebook,
			"instagram":     club.Instagram,
			"whatsapp":      club.WhatsApp,
			"established":   club.EstablishedDate,
			"facilities":    club.Facilities,
			"schedules":     club.TrainingSchedule,
			"social_media":  club.SocialMedia,
			"page_settings": club.PageSettings,
		})
	}
}

// UpdateClubMe updates the club profile for the authenticated user
func UpdateClubMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			Name         string        `json:"name"`
			Slug         string        `json:"slug"`
			Description  string        `json:"description"`
			BannerURL    string        `json:"banner_url"`
			LogoURL      string        `json:"logo_url"`
			City         string        `json:"city"`
			Province     string        `json:"province"`
			Established  string        `json:"established"`
			Phone        string        `json:"phone"`
			WhatsApp     string        `json:"whatsapp"`
			Email        string        `json:"email"`
			Instagram    string        `json:"instagram"`
			Facebook     string        `json:"facebook"`
			Website      string        `json:"website"`
			Address      string        `json:"address"`
			Facilities   []string      `json:"facilities"`
			Schedules    []interface{} `json:"schedules"`
			SocialMedia  []interface{} `json:"social_media"`
			PageSettings interface{}   `json:"page_settings"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if slug has already been changed
		var currentSlug string
		var slugChanged bool
		err := db.Get(&currentSlug, "SELECT COALESCE(slug, '') FROM clubs WHERE uuid = ?", userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}
		db.Get(&slugChanged, "SELECT COALESCE(slug_changed, 0) FROM clubs WHERE uuid = ?", userID)

		// Determine if we should update the slug
		newSlug := currentSlug
		newSlugChanged := slugChanged
		if req.Slug != "" && req.Slug != currentSlug {
			if slugChanged {
				// Slug already changed once, keep old slug
				newSlug = currentSlug
			} else {
				// Check if new slug is available
				var count int
				err := db.Get(&count, "SELECT COUNT(*) FROM clubs WHERE slug = ? AND uuid != ?", req.Slug, userID)
				if err == nil && count == 0 {
					newSlug = req.Slug
					newSlugChanged = true
				}
			}
		}

		facilitiesJSON, _ := json.Marshal(req.Facilities)
		schedulesJSON, _ := json.Marshal(req.Schedules)
		socialMediaJSON, _ := json.Marshal(req.SocialMedia)
		pageSettingsJSON, _ := json.Marshal(req.PageSettings)

		// Parse established date from ISO format to MySQL date format
		var establishedDate interface{}
		if req.Established != "" {
			// Try to parse ISO format (2020-01-15T00:00:00Z or 2020-01-15T00:00:00+00:00)
			dateStr := strings.TrimSpace(req.Established)
			// Remove timezone and time if present
			if strings.Contains(dateStr, "T") {
				dateStr = strings.Split(dateStr, "T")[0]
			}
			// Validate it's a valid date format (YYYY-MM-DD)
			if parsedDate, err := time.Parse("2006-01-02", dateStr); err == nil {
				establishedDate = parsedDate.Format("2006-01-02")
			} else {
				establishedDate = nil
			}
		} else {
			establishedDate = nil
		}

		_, err = db.Exec(`
			UPDATE clubs SET 
				name = ?, slug = ?, slug_changed = ?, description = ?, banner_url = ?, logo_url = ?, avatar_url = ?, 
				city = ?, province = ?, established_date = ?, phone = ?, email = ?, 
				social_facebook = ?, social_instagram = ?, website = ?, address = ?,
				facilities = ?, training_schedule = ?, social_media = ?, page_settings = ?, updated_at = NOW()
			WHERE uuid = ?`,
			req.Name, newSlug, newSlugChanged, req.Description, utils.ExtractFilename(req.BannerURL), utils.ExtractFilename(req.LogoURL), utils.ExtractFilename(req.LogoURL),
			req.City, req.Province, establishedDate, req.Phone, req.Email,
			req.Facebook, req.Instagram, req.Website, req.Address,
			string(facilitiesJSON), string(schedulesJSON), string(socialMediaJSON), string(pageSettingsJSON), userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update club: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Club profile updated successfully"})
	}
}

// GetClubProfile returns dynamic sections for a club
func GetClubProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		var pageSettings *string
		err := db.Get(&pageSettings, `
			SELECT page_settings FROM clubs
			WHERE slug = ? OR uuid = ?`, slug, slug)

		if err != nil || pageSettings == nil || *pageSettings == "" {
			// Return default empty sections if not found
			c.JSON(http.StatusOK, gin.H{"sections": []interface{}{}})
			return
		}

		var pageSettingsMap map[string]interface{}
		json.Unmarshal([]byte(*pageSettings), &pageSettingsMap)

		var sections interface{}
		if sectionsVal, ok := pageSettingsMap["sections"]; ok {
			sections = sectionsVal
		} else {
			sections = []interface{}{}
		}

		c.JSON(http.StatusOK, gin.H{"sections": sections})
	}
}

// UpdateMyClubProfile updates dynamic sections for the authenticated club owner
func UpdateMyClubProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			Sections interface{} `json:"sections" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get current page_settings
		var currentPageSettings *string
		err := db.Get(&currentPageSettings, "SELECT page_settings FROM clubs WHERE uuid = ?", userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}

		// Parse and update page_settings
		var pageSettingsMap map[string]interface{}
		if currentPageSettings != nil && *currentPageSettings != "" {
			json.Unmarshal([]byte(*currentPageSettings), &pageSettingsMap)
		}
		if pageSettingsMap == nil {
			pageSettingsMap = make(map[string]interface{})
		}
		pageSettingsMap["sections"] = req.Sections

		pageSettingsJSON, _ := json.Marshal(pageSettingsMap)

		// Update page_settings in clubs table
		_, err = db.Exec(`
			UPDATE clubs SET page_settings = ?, updated_at = NOW()
			WHERE uuid = ?`,
			string(pageSettingsJSON), userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sections: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Club sections updated successfully"})
	}
}

// GetClubDashboardStats returns real-time statistics for the club dashboard
func GetClubDashboardStats(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		// Get club UUID
		var clubID string
		err := db.Get(&clubID, "SELECT uuid FROM clubs WHERE uuid = ?", userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}

		var stats struct {
			TotalMembers   int `json:"totalMembers"`
			ActiveArchers  int `json:"activeArchers"`
			UpcomingEvents int `json:"upcomingEvents"`
			TotalAwards    int `json:"totalAwards"`
		}

		// Total Members
		db.Get(&stats.TotalMembers, "SELECT COUNT(*) FROM club_members WHERE club_id = ?", clubID)

		// Active Archers
		db.Get(&stats.ActiveArchers, "SELECT COUNT(*) FROM club_members WHERE club_id = ? AND status = 'active'", clubID)

		// Upcoming Events (General upcoming events as a fallback)
		db.Get(&stats.UpcomingEvents, "SELECT COUNT(*) FROM events WHERE status IN ('published', 'ongoing') AND start_date >= NOW()")

		// Total Awards (Generic count for now)
		stats.TotalAwards = 0

		// Recent Members
		var recentMembers []struct {
			Name     string `json:"name" db:"name"`
			JoinDate string `json:"joinDate" db:"joinDate"`
			Status   string `json:"status" db:"status"`
		}
		db.Select(&recentMembers, `
			SELECT u.full_name as name, DATE_FORMAT(cm.created_at, '%d %b %Y') as joinDate, cm.status
			FROM club_members cm
			JOIN archers u ON cm.archer_id = u.uuid
			WHERE cm.club_id = ?
			ORDER BY cm.created_at DESC
			LIMIT 5
		`, clubID)

		// Upcoming Tournaments
		var upcomingTournaments []struct {
			ID     string `json:"id" db:"id"`
			Name   string `json:"name" db:"name"`
			Date   string `json:"date" db:"date"`
			Status string `json:"status" db:"status"`
		}
		db.Select(&upcomingTournaments, `
			SELECT uuid as id, name, DATE_FORMAT(start_date, '%d %b %Y') as date, status
			FROM events
			WHERE status IN ('published', 'ongoing') AND start_date >= NOW()
			ORDER BY start_date ASC
			LIMIT 3
		`)

		c.JSON(http.StatusOK, gin.H{
			"stats":               stats,
			"recentMembers":       recentMembers,
			"upcomingTournaments": upcomingTournaments,
		})
	}
}
