package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"archeryhub-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
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
			RegistrationConfig *string `json:"registration_config" db:"registration_config"`
		}

		err := db.Get(&club, `
			SELECT uuid, name, slug, COALESCE(slug_changed, 0) as slug_changed, description, avatar_url, banner_url, logo_url, address, city, province, phone, email, website, social_facebook, social_instagram, established_date, facilities, training_schedule, social_media, page_settings, registration_config 
			FROM clubs 
			WHERE uuid = ?`, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
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

		data := gin.H{
			"id":            club.UUID,
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
			"registration_config": club.RegistrationConfig,
			"user_type":     "club",
		}

		c.JSON(http.StatusOK, gin.H{"data": data})
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
			RegistrationConfig interface{} `json:"registration_config"`
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
		registrationConfigJSON, _ := json.Marshal(req.RegistrationConfig)

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
				facilities = ?, training_schedule = ?, social_media = ?, page_settings = ?, registration_config = ?, updated_at = NOW()
			WHERE uuid = ?`,
			req.Name, newSlug, newSlugChanged, req.Description, utils.ExtractFilename(req.BannerURL), utils.ExtractFilename(req.LogoURL), utils.ExtractFilename(req.LogoURL),
			req.City, req.Province, establishedDate, req.Phone, req.Email,
			req.Facebook, req.Instagram, req.Website, req.Address,
			string(facilitiesJSON), string(schedulesJSON), string(socialMediaJSON), string(pageSettingsJSON), string(registrationConfigJSON), userID)

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

// === CLUB MEMBERSHIP FUNCTIONS ===

// ClubMember represents the relationship between an archer and a club
type ClubMember struct {
	UUID      string     `json:"uuid" db:"uuid"`
	ClubID    string     `json:"club_id" db:"club_id"`
	ArcherID  string     `json:"archer_id" db:"archer_id"`
	Status    string     `json:"status" db:"status"`
	Role      string     `json:"role" db:"role"`
	JoinedAt         *time.Time `json:"joined_at" db:"joined_at"`
	RegistrationData *string    `json:"registration_data" db:"registration_data"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
	CoachNotes       *string    `json:"coach_notes" db:"coach_notes"`
}

// GetClubs returns all clubs (public) with pagination and filtering
func GetClubs(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
		search := c.Query("q")
		province := c.Query("province")
		city := c.Query("city")

		if page < 1 {
			page = 1
		}
		offset := (page - 1) * limit

		baseQuery := `
			FROM clubs c 
			WHERE c.status = 'active'
		`
		args := []interface{}{}

		if search != "" {
			baseQuery += " AND (c.name LIKE ? OR c.description LIKE ?)"
			args = append(args, "%"+search+"%", "%"+search+"%")
		}

		if province != "" {
			baseQuery += " AND c.province = ?"
			args = append(args, province)
		}

		if city != "" {
			baseQuery += " AND c.city LIKE ?"
			args = append(args, "%"+city+"%")
		}

		// Count total items
		var totalItems int
		err := db.Get(&totalItems, "SELECT COUNT(*) "+baseQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count clubs: " + err.Error()})
			return
		}

		// Fetch data
		query := `
			SELECT c.uuid, c.name, c.slug, c.avatar_url, c.banner_url, c.logo_url, c.city, c.province, c.phone, c.social_instagram,
				   (SELECT COUNT(*) FROM club_members WHERE club_id = c.uuid AND status = 'active') as member_count
		` + baseQuery + ` ORDER BY c.name ASC LIMIT ? OFFSET ?`

		fetchArgs := append(args, limit, offset)

		type ClubResponse struct {
			UUID            string   `json:"uuid" db:"uuid"`
			Name            string   `json:"name" db:"name"`
			Slug            string   `json:"slug" db:"slug"`
			AvatarURL       *string  `json:"avatar_url" db:"avatar_url"`
			BannerURL       *string  `json:"banner_url" db:"banner_url"`
			LogoURL         *string  `json:"logo_url" db:"logo_url"`
			City            *string  `json:"city" db:"city"`
			Province        *string  `json:"province" db:"province"`
			Phone           *string  `json:"phone" db:"phone"`
			SocialInstagram *string  `json:"social_instagram" db:"social_instagram"`
			MemberCount     int      `json:"member_count" db:"member_count"`
			MemberAvatars   []string `json:"member_avatars" db:"-"`
		}

		var clubs []ClubResponse

		err = db.Select(&clubs, query, fetchArgs...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch clubs: " + err.Error()})
			return
		}

		if clubs == nil {
			clubs = []ClubResponse{}
		} else {
			for i := range clubs {
				// Mask URLs
				if clubs[i].AvatarURL != nil {
					masked := utils.MaskMediaURL(*clubs[i].AvatarURL)
					clubs[i].AvatarURL = &masked
				}
				if clubs[i].BannerURL != nil {
					masked := utils.MaskMediaURL(*clubs[i].BannerURL)
					clubs[i].BannerURL = &masked
				}
				if clubs[i].LogoURL != nil {
					masked := utils.MaskMediaURL(*clubs[i].LogoURL)
					clubs[i].LogoURL = &masked
				}

				// Get member avatars
				var memberAvatars []string
				db.Select(&memberAvatars, `
					SELECT a.avatar_url 
					FROM club_members cm 
					JOIN archers a ON cm.archer_id = a.uuid 
					WHERE cm.club_id = ? AND cm.status = 'active' AND a.avatar_url IS NOT NULL 
					LIMIT 3
				`, clubs[i].UUID)
				if memberAvatars != nil {
					clubs[i].MemberAvatars = memberAvatars
				} else {
					clubs[i].MemberAvatars = []string{}
				}
			}
		}

		totalPages := int(math.Ceil(float64(totalItems) / float64(limit)))

		c.JSON(http.StatusOK, gin.H{
			"data": clubs,
			"meta": gin.H{
				"current_page": page,
				"limit":        limit,
				"total_items":  totalItems,
				"total_pages":  totalPages,
			},
		})
	}
}

// GetClubBySlug returns a single club by slug
func GetClubBySlug(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		var club struct {
			UUID             string  `json:"uuid" db:"uuid"`
			Name             string  `json:"name" db:"name"`
			Slug             string  `json:"slug" db:"slug"`
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
			WhatsApp         *string `json:"whatsapp" db:"phone"`
			EstablishedDate  *string `json:"established" db:"established_date"`
			Facilities       *string `json:"facilities" db:"facilities"`
			TrainingSchedule *string `json:"training_schedule" db:"training_schedule"`
			SocialMedia      *string `json:"social_media" db:"social_media"`
			PageSettings     *string `json:"page_settings" db:"page_settings"`
			RegistrationConfig *string `json:"registration_config" db:"registration_config"`
			SubscriptionPlanID *int    `json:"subscription_plan_id" db:"subscription_plan_id"`
			SubscriptionStatus string  `json:"subscription_status" db:"subscription_status"`
			CreatedAt        string  `json:"created_at" db:"created_at"`
		}

		err := db.Get(&club, `
			SELECT uuid, name, slug, description, avatar_url, banner_url, avatar_url as logo_url, 
			       address, city, province, phone, email, website, social_facebook, social_instagram, 
			       established_date, facilities, training_schedule, social_media, page_settings, registration_config, 
			       COALESCE(subscription_status, 'trial') as subscription_status, subscription_plan_id, created_at 
			FROM clubs 
			WHERE slug = ? OR uuid = ?`, slug, slug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}

		club.WhatsApp = club.Phone

		// Get member count
		var memberCount int
		db.Get(&memberCount, "SELECT COUNT(*) FROM club_members WHERE club_id = ? AND status = 'active'", club.UUID)

		// Get member list
		memberLimit, _ := strconv.Atoi(c.DefaultQuery("member_limit", "8"))
		memberPage, _ := strconv.Atoi(c.DefaultQuery("member_page", "1"))
		if memberPage < 1 {
			memberPage = 1
		}
		memberOffset := (memberPage - 1) * memberLimit

		var topMembers []struct {
			ID       string  `json:"id" db:"uuid"`
			Name     string  `json:"name" db:"full_name"`
			Avatar   *string `json:"avatar" db:"avatar_url"`
			Division *string `json:"division" db:"bow_type"`
		}
		db.Select(&topMembers, `
			SELECT a.uuid, a.full_name, a.avatar_url, a.bow_type
			FROM club_members cm
			JOIN archers a ON cm.archer_id = a.uuid
			WHERE cm.club_id = ? AND cm.status = 'active'
			LIMIT ? OFFSET ?
		`, club.UUID, memberLimit, memberOffset)

		// Get event count
		var eventCount int
		db.Get(&eventCount, "SELECT COUNT(DISTINCT tp.event_id) FROM event_participants tp JOIN archers a ON tp.archer_id = a.uuid WHERE a.club_id = ?", club.UUID)

		// Get real achievements
		var achievements int
		db.Get(&achievements, `
			SELECT COUNT(*) 
			FROM event_participants tp 
			JOIN archers a ON tp.archer_id = a.uuid 
			WHERE a.club_id = ? AND (tp.score > 0)
		`, club.UUID)

		// Get real news for the club
		type NewsItem struct {
			UUID        string    `json:"uuid" db:"uuid"`
			Title       string    `json:"title" db:"title"`
			Slug        string    `json:"slug" db:"slug"`
			Excerpt     *string   `json:"excerpt" db:"excerpt"`
			ImageURL    *string   `json:"image_url" db:"image_url"`
			Category    string    `json:"category" db:"category"`
			PublishedAt *time.Time `json:"published_at" db:"published_at"`
		}

		var clubNews []NewsItem
		db.Select(&clubNews, `
			SELECT uuid, title, slug, excerpt, image_url, category, published_at 
			FROM news 
			WHERE club_id = ? AND status = 'published'
			ORDER BY published_at DESC 
			LIMIT 10
		`, club.UUID)

		if clubNews == nil {
			clubNews = []NewsItem{}
		}

		// Mask news images
		for i := range clubNews {
			if clubNews[i].ImageURL != nil {
				masked := utils.MaskMediaURL(*clubNews[i].ImageURL)
				clubNews[i].ImageURL = &masked
			}
		}

		// Split news into achievements and general news for backward compatibility or split use
		var achievementsItems []NewsItem
		var regularNewsItems []NewsItem
		for _, n := range clubNews {
			if n.Category == "prestasi" {
				achievementsItems = append(achievementsItems, n)
			} else {
				regularNewsItems = append(regularNewsItems, n)
			}
		}

		// Mask URLs
		var avatarURL, logoURL, bannerURL string
		if club.AvatarURL != nil {
			avatarURL = utils.MaskMediaURL(*club.AvatarURL)
		}
		if club.LogoURL != nil {
			logoURL = utils.MaskMediaURL(*club.LogoURL)
		}
		if club.BannerURL != nil {
			bannerURL = utils.MaskMediaURL(*club.BannerURL)
		}

		for i := range topMembers {
			if topMembers[i].Avatar != nil {
				masked := utils.MaskMediaURL(*topMembers[i].Avatar)
				topMembers[i].Avatar = &masked
			}
		}

		// Parse sections from page_settings
		var sections []interface{}
		if club.PageSettings != nil && *club.PageSettings != "" {
			var pageSettingsMap map[string]interface{}
			json.Unmarshal([]byte(*club.PageSettings), &pageSettingsMap)
			if sectionsVal, ok := pageSettingsMap["sections"]; ok {
				sections = sectionsVal.([]interface{})
			}
		}

		// Return data in expected format
		response := gin.H{
			"id":            club.UUID,
			"uuid":          club.UUID,
			"name":          club.Name,
			"slug":          club.Slug,
			"description":   club.Description,
			"avatar_url":    avatarURL,
			"logo_url":      logoURL,
			"banner_url":    bannerURL,
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
			"member_count":  memberCount,
			"members":       memberCount,
			"event_count":   eventCount,
			"events":        eventCount,
			"achievements":  achievementsItems,
			"news":          regularNewsItems,
			"recent_events": achievementsItems, // Keep for backward compatibility if needed
			"top_members":   topMembers,
			"sections":      sections,
			"registration_config": club.RegistrationConfig,
			"subscription_status":  club.SubscriptionStatus,
			"subscription_plan_id": club.SubscriptionPlanID,
		}

		// Parse social media
		if club.SocialMedia != nil && *club.SocialMedia != "" {
			var parsedSocialMedia interface{}
			json.Unmarshal([]byte(*club.SocialMedia), &parsedSocialMedia)
			response["social_media"] = parsedSocialMedia
		} else {
			response["social_media"] = []interface{}{}
		}

		c.JSON(http.StatusOK, response)
	}
}

// JoinClub allows an archer to request membership
func JoinClub(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := c.Param("clubId")
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		// Only archers can join clubs
		if userType != "archer" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only archers can join clubs"})
			return
		}

		// Check if club exists
		var clubExists bool
		err := db.Get(&clubExists, "SELECT EXISTS(SELECT 1 FROM clubs WHERE uuid = ?)", clubID)
		if err != nil || !clubExists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}

		// Check if already a member of any club
		var existing struct {
			ClubID string `db:"club_id"`
			Status string `db:"status"`
		}
		err = db.Get(&existing, "SELECT club_id, status FROM club_members WHERE archer_id = ? AND status IN ('pending', 'active', 'invited')", userID)
		if err == nil {
			if existing.ClubID == clubID {
				c.JSON(http.StatusConflict, gin.H{"error": "You already have a membership request for this club"})
			} else {
				c.JSON(http.StatusConflict, gin.H{"error": "You are already a member of another club"})
			}
			return
		}

		// Check for registration data
		var req struct {
			RegistrationData interface{} `json:"registration_data"`
		}
		c.ShouldBindJSON(&req)
		registrationDataJSON, _ := json.Marshal(req.RegistrationData)

		// Create membership request
		memberID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO club_members (uuid, club_id, archer_id, status, role, registration_data)
			VALUES (?, ?, ?, 'pending', 'member', ?)
		`, memberID, clubID, userID, string(registrationDataJSON))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create membership request: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Membership request submitted successfully",
			"id":      memberID,
		})
	}
}

// GetMyClubMembership returns the current user's club membership status
func GetMyClubMembership(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var membership struct {
			ClubMember
			ClubName string `json:"club_name" db:"club_name"`
		}

		err := db.Get(&membership, `
			SELECT cm.*, c.name as club_name 
			FROM club_members cm 
			JOIN clubs c ON cm.club_id = c.uuid 
			WHERE cm.archer_id = ? AND cm.status IN ('pending', 'active')
		`, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "No active membership found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": membership})
	}
}

// GetMyClubInvitations returns the current user's club invitations
func GetMyClubInvitations(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var invitations []struct {
			MemberUUID string  `json:"uuid" db:"uuid"`
			ClubID     string  `json:"club_id" db:"club_id"`
			ClubName   string  `json:"club_name" db:"club_name"`
			ClubLogo   *string `json:"club_logo" db:"club_logo"`
			Status     string  `json:"status" db:"status"`
			Role       string  `json:"role" db:"role"`
			CreatedAt  string  `json:"created_at" db:"created_at"`
		}

		err := db.Select(&invitations, `
			SELECT cm.uuid, cm.club_id, c.name as club_name, c.logo_url as club_logo, cm.status, cm.role, cm.created_at
			FROM club_members cm 
			JOIN clubs c ON cm.club_id = c.uuid 
			WHERE cm.archer_id = ? AND cm.status = 'invited'
		`, userID)

		if err != nil {
			logrus.WithError(err).Error("[INVITE] Failed to get invitations")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch invitations"})
			return
		}

		if invitations == nil {
			invitations = []struct {
				MemberUUID string  `json:"uuid" db:"uuid"`
				ClubID     string  `json:"club_id" db:"club_id"`
				ClubName   string  `json:"club_name" db:"club_name"`
				ClubLogo   *string `json:"club_logo" db:"club_logo"`
				Status     string  `json:"status" db:"status"`
				Role       string  `json:"role" db:"role"`
				CreatedAt  string  `json:"created_at" db:"created_at"`
			}{}
		}

		c.JSON(http.StatusOK, gin.H{"data": invitations})
	}
}

// RespondToInvitation allows an archer to accept or reject a club invitation
func RespondToInvitation(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")
		memberID := c.Param("memberId")

		if userType != "archer" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only archers can respond to invitations"})
			return
		}

		var req struct {
			Action string `json:"action" binding:"required"` // 'accept' or 'reject'
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		status := "rejected"
		if req.Action == "accept" {
			status = "active"
		} else if req.Action != "reject" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action, must be 'accept' or 'reject'"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// Verify invitation exists for this user
		var clubID string
		err = tx.Get(&clubID, `SELECT club_id FROM club_members WHERE uuid = ? AND archer_id = ? AND status = 'invited'`, memberID, userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found or no longer valid"})
			return
		}

		// Update membership status
		_, err = tx.Exec(`
			UPDATE club_members SET status = ?, updated_at = NOW() 
			WHERE uuid = ? AND archer_id = ?
		`, status, memberID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update membership status"})
			return
		}

		// If accepted, sync club_id in archers table
		if status == "active" {
			_, err = tx.Exec(`UPDATE archers SET club_id = ?, updated_at = NOW() WHERE uuid = ?`, clubID, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to sync archer record"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Successfully " + status + " the invitation"})
	}
}

// LeaveClub allows an archer to leave their club
func LeaveClub(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		result, err := db.Exec(`
			UPDATE club_members SET status = 'left', updated_at = NOW() 
			WHERE archer_id = ? AND status = 'active'
		`, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to leave club"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No active membership found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Successfully left the club"})
	}
}

// ApproveClubMember allows club admin to approve a membership request
func ApproveClubMember(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		memberID := c.Param("memberId")
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		// Only club admins can approve
		if userType != "club" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only club admins can approve members"})
			return
		}

		logrus.Infof("[APPROVE] Attempting to approve member %s. UserID: %v, UserType: %v", memberID, userID, userType)
		
		// In our system, for a club user, the userID in the token is their club's UUID
		clubID := fmt.Sprintf("%v", userID)
		
		// Verify this club actually exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM clubs WHERE uuid = ?)", clubID)
		if err != nil || !exists {
			logrus.Errorf("[APPROVE] Club verification failed for ID: %s. Error: %v", clubID, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}

		now := time.Now()
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// Update membership status to active
		var archerID string
		err = tx.Get(&archerID, "SELECT archer_id FROM club_members WHERE uuid = ? AND club_id = ? AND status = 'pending'", memberID, clubID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Membership request not found"})
			return
		}

		_, err = tx.Exec(`
			UPDATE club_members SET status = 'active', joined_at = ?, updated_at = NOW() 
			WHERE uuid = ? AND club_id = ? AND status = 'pending'
		`, now, memberID, clubID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve member record"})
			return
		}

		// Update archer's club_id
		_, err = tx.Exec(`
			UPDATE archers SET club_id = ? 
			WHERE uuid = ?
		`, clubID, archerID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update archer record"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Member approved successfully"})
	}
}

// MemberWithDetails includes all membership and archer profile information
type MemberWithDetails struct {
	ClubMember
	ArcherName  string     `json:"full_name" db:"archer_name"`
	BowType     *string    `json:"bow_type" db:"bow_type"`
	Gender      *string    `json:"gender" db:"gender"`
	DateOfBirth *time.Time `json:"date_of_birth" db:"date_of_birth"`
	City        *string    `json:"city" db:"city"`
	ID          string     `json:"id" db:"archer_human_id"`
	AvatarURL   *string    `json:"avatar_url" db:"avatar_url"`
}

// GetClubMembers returns all members of a club
func GetClubMembers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := c.Param("clubId")

		var members []MemberWithDetails

		err := db.Select(&members, `
			SELECT 
				cm.*, 
				u.full_name as archer_name,
				u.bow_type,
				u.gender,
				u.date_of_birth,
				u.city,
				u.id as archer_human_id,
				u.avatar_url
			FROM club_members cm
			JOIN archers u ON cm.archer_id = u.uuid
			WHERE cm.club_id = ? AND cm.status IN ('active', 'pending', 'invited')
			ORDER BY cm.status ASC, cm.created_at DESC
		`, clubID)

		if err != nil {
			logrus.WithError(err).Error("Failed to fetch members from database")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch members"})
			return
		}

		if err == nil {
			logrus.Infof("[MEMBERS] Found %d active/pending members for club %s", len(members), clubID)
		}

		if members == nil {
			members = []MemberWithDetails{}
		}

		c.JSON(http.StatusOK, gin.H{"data": members})
	}
}

// InviteToClub allows club admin to invite an archer
func InviteToClub(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		// Only club owners can invite
		if userType != "club" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only club owners can invite members"})
			return
		}

		var req struct {
			ArcherID string `json:"archer_id" binding:"required"`
			Role     string `json:"role"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		logrus.Infof("[INVITE] Attempting to invite archer %s. UserID: %v, UserType: %v", req.ArcherID, userID, userType)
		
		// In our system, for a club user, the userID in the token is their club's UUID
		clubID := fmt.Sprintf("%v", userID)
		
		// Verify this club actually exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM clubs WHERE uuid = ?)", clubID)
		if err != nil || !exists {
			logrus.Errorf("[INVITE] Club verification failed for ID: %s. Error: %v", clubID, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}

		// Check if archer exists
		var archerExists bool
		err = db.Get(&archerExists, "SELECT EXISTS(SELECT 1 FROM archers WHERE uuid = ?)", req.ArcherID)
		if err != nil || !archerExists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer not found"})
			return
		}

		// Check if archer already has membership
		var existingMembership string
		err = db.Get(&existingMembership, "SELECT club_id FROM club_members WHERE archer_id = ? AND status IN ('pending', 'active', 'invited')", req.ArcherID)
		if err == nil && existingMembership != "" {
			c.JSON(http.StatusConflict, gin.H{"error": "Archer already has an active membership"})
			return
		}

		if req.Role == "" {
			req.Role = "member"
		}

		// Create invitation
		memberID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO club_members (uuid, club_id, archer_id, status, role)
			VALUES (?, ?, ?, 'invited', ?)
		`, memberID, clubID, req.ArcherID, req.Role)

		if err != nil {
			logrus.WithError(err).Errorf("[INVITE] Failed to insert club_members record. clubID=%s archerID=%s", clubID, req.ArcherID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send invitation"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Invitation sent successfully",
			"id":      memberID,
		})
	}
}

// KickClubMember allows club admin to remove an archer from their club
func KickClubMember(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")
		archerID := c.Param("archerId")

		// Only club owners can kick
		if userType != "club" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only club owners can remove members"})
			return
		}

		logrus.Infof("[KICK] Attempting to kick archer %s. UserID: %v, UserType: %v", archerID, userID, userType)
		
		// In our system, for a club user, the userID in the token is their club's UUID
		clubID := fmt.Sprintf("%v", userID)
		
		// Verify this club actually exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM clubs WHERE uuid = ?)", clubID)
		if err != nil || !exists {
			logrus.Errorf("[KICK] Club verification failed for ID: %s. Error: %v", clubID, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}

		// Update membership status to left
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		result, err := tx.Exec(`
			UPDATE club_members SET status = 'left', updated_at = NOW() 
			WHERE archer_id = ? AND club_id = ? AND status IN ('active', 'pending', 'invited')
		`, archerID, clubID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member record"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "No active membership found for this archer in your club"})
			return
		}

		// Also clear club_id in archers table if it matches this club
		_, err = tx.Exec(`
			UPDATE archers SET club_id = NULL 
			WHERE uuid = ? AND club_id = ?
		`, archerID, clubID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update archer record"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Member successfully removed from the club"})
	}
}

// UpdateMemberNotes allows club owner to update notes for a specific member
func UpdateMemberNotes(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")
		archerID := c.Param("archerId")

		if userType != "club" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only club owners can update notes"})
			return
		}

		var req struct {
			Notes string `json:"notes"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		logrus.Infof("[NOTES] Updating notes for archer %s. UserID: %v, UserType: %v", archerID, userID, userType)
		
		// In our system, for a club user, the userID in the token is their club's UUID
		clubID := fmt.Sprintf("%v", userID)
		
		// Verify this club actually exists
		var exists bool
		err := db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM clubs WHERE uuid = ?)", clubID)
		if err != nil || !exists {
			logrus.Errorf("[NOTES] Club verification failed for ID: %s. Error: %v", clubID, err)
			c.JSON(http.StatusNotFound, gin.H{"error": "Club not found"})
			return
		}

		logrus.Infof("Updating notes for archer %s in club %s", archerID, clubID)

		// Update notes in club_members table
		result, err := db.Exec(`
			UPDATE club_members 
			SET coach_notes = ?, updated_at = NOW() 
			WHERE archer_id = ? AND club_id = ?
		`, req.Notes, archerID, clubID)

		if err != nil {
			logrus.WithError(err).Error("Failed to update notes in database")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update notes"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			logrus.Warnf("No rows updated for archer %s in club %s. Member might not belong to this club.", archerID, clubID)
			c.JSON(http.StatusNotFound, gin.H{"error": "Member not found in your club"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Notes updated successfully"})
	}
}
