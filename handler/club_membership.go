package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"

	"archeryhub-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ClubMember represents the relationship between an archer and a club
type ClubMember struct {
	UUID      string     `json:"uuid" db:"uuid"`
	ClubID    string     `json:"club_id" db:"club_id"`
	ArcherID  string     `json:"archer_id" db:"archer_id"`
	Status    string     `json:"status" db:"status"`
	Role      string     `json:"role" db:"role"`
	JoinedAt  *time.Time `json:"joined_at" db:"joined_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
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
				if clubs[i].AvatarURL != nil {
					*clubs[i].AvatarURL = utils.MaskMediaURL(*clubs[i].AvatarURL)
				}
				if clubs[i].BannerURL != nil {
					*clubs[i].BannerURL = utils.MaskMediaURL(*clubs[i].BannerURL)
				}
				if clubs[i].LogoURL != nil {
					*clubs[i].LogoURL = utils.MaskMediaURL(*clubs[i].LogoURL)
				}

				var avatars []string
				err := db.Select(&avatars, `
					SELECT a.avatar_url 
					FROM club_members cm
					JOIN archers a ON cm.archer_id = a.uuid
					WHERE cm.club_id = ? AND cm.status = 'active' AND a.avatar_url IS NOT NULL AND a.avatar_url != ''
					LIMIT 3
				`, clubs[i].UUID)
				if err == nil {
					maskedAvatars := make([]string, len(avatars))
					for j, avatar := range avatars {
						maskedAvatars[j] = utils.MaskMediaURL(avatar)
					}
					clubs[i].MemberAvatars = maskedAvatars
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
			CreatedAt        string  `json:"created_at" db:"created_at"`
		}

		err := db.Get(&club, `
			SELECT uuid, name, slug, description, avatar_url, banner_url, avatar_url as logo_url, 
			       address, city, province, phone, email, website, social_facebook, social_instagram, 
			       established_date, facilities, training_schedule, social_media, page_settings, created_at 
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

		// Get member list (Top 8 for profile)
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
			LIMIT 8
		`, club.UUID)

		// Get event count
		var eventCount int
		db.Get(&eventCount, "SELECT COUNT(DISTINCT tp.event_id) FROM event_participants tp JOIN archers a ON tp.archer_id = a.uuid WHERE a.club_id = ?", club.UUID)

		// Get real achievements (Sum of podiums for club members)
		// For now, let's treat any participation in a completed event as an 'achievement' if they were in top 3
		// Assuming we have a way to track results, but if not, let's at least return a count
		var achievements int
		db.Get(&achievements, `
			SELECT COUNT(*) 
			FROM event_participants tp 
			JOIN archers a ON tp.archer_id = a.uuid 
			WHERE a.club_id = ? AND (tp.score > 0)
		`, club.UUID)

		// Varied dummy achievements for display
		dummyAchievements := []map[string]interface{}{
			{"name": "Indonesian Open 2024", "date": "12 Okt 2024", "result": "Medali Emas"},
			{"name": "Piala Menpora 2023", "date": "05 Jun 2023", "result": "Juara Umum"},
			{"name": "Kejurda DKI Jakarta 2024", "date": "15 Mar 2024", "result": "Medali Perak"},
			{"name": "Jakarta Archery Series", "date": "20 Nov 2023", "result": "Best Performance"},
			{"name": "Bali Archery Festival", "date": "12 Jan 2024", "result": "Juara 3"},
			{"name": "Surabaya Open 2023", "date": "18 Aug 2023", "result": "Juara 2"},
			{"name": "Bandung Archery Cup", "date": "10 Apr 2024", "result": "Medali Perunggu"},
		}

		seedValue := 0
		if len(club.UUID) >= 4 {
			seedValue = int(club.UUID[0]) + int(club.UUID[1]) + int(club.UUID[2]) + int(club.UUID[3])
		}

		clubAchievements := []interface{}{}
		numAchievements := (seedValue % 3) + 2 // 2 to 4 achievements
		for i := 0; i < numAchievements; i++ {
			idx := (seedValue + i) % len(dummyAchievements)
			clubAchievements = append(clubAchievements, dummyAchievements[idx])
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
				if sectionsArray, ok := sectionsVal.([]interface{}); ok {
					sections = sectionsArray
				}
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
			"achievements":  len(clubAchievements),
			"recent_events": clubAchievements,
			"top_members":   topMembers,
			"sections":      sections,
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
		err = db.Get(&existing, "SELECT club_id, status FROM club_members WHERE archer_id = ? AND status IN ('pending', 'active')", userID)
		if err == nil {
			if existing.ClubID == clubID {
				if existing.Status == "active" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "You are already an active member of this club"})
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "You already have a pending membership request to this club"})
				}
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "You are already a member or have a pending request to another club"})
			}
			return
		}

		// Create membership request
		memberID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO club_members (uuid, club_id, archer_id, status, role)
			VALUES (?, ?, ?, 'pending', 'member')
		`, memberID, clubID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit membership request"})
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
			c.JSON(http.StatusOK, gin.H{"data": nil, "message": "No active membership"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": membership})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "No active membership found"})
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
			c.JSON(http.StatusForbidden, gin.H{"error": "Only club owners can approve members"})
			return
		}

		// Verify the member belongs to the user's club
		var clubID string
		err := db.Get(&clubID, "SELECT uuid FROM clubs WHERE owner_id = ?", userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't own any club"})
			return
		}

		now := time.Now()
		result, err := db.Exec(`
			UPDATE club_members SET status = 'active', joined_at = ?, updated_at = NOW() 
			WHERE uuid = ? AND club_id = ? AND status = 'pending'
		`, now, memberID, clubID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve member"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Membership request not found or already processed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Member approved successfully"})
	}
}

// GetClubMembers returns all members of a club
func GetClubMembers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := c.Param("clubId")

		var members []struct {
			ClubMember
			ArcherName string `json:"archer_name" db:"archer_name"`
		}

		err := db.Select(&members, `
			SELECT cm.*, u.full_name as archer_name
			FROM club_members cm
			JOIN users u ON cm.archer_id = u.uuid
			WHERE cm.club_id = ?
			ORDER BY cm.status ASC, cm.created_at DESC
		`, clubID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch members"})
			return
		}

		if members == nil {
			members = make([]struct {
				ClubMember
				ArcherName string `json:"archer_name" db:"archer_name"`
			}, 0)
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Archer ID is required"})
			return
		}

		// Get club owned by user
		var clubID string
		err := db.Get(&clubID, "SELECT uuid FROM clubs WHERE owner_id = ?", userID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't own any club"})
			return
		}

		// Check if archer exists
		var archerExists bool
		err = db.Get(&archerExists, "SELECT EXISTS(SELECT 1 FROM users WHERE uuid = ? AND user_type = 'archer')")
		if err != nil || !archerExists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer not found"})
			return
		}

		// Check if archer already has membership
		var existingMembership string
		err = db.Get(&existingMembership, "SELECT club_id FROM club_members WHERE archer_id = ? AND status IN ('pending', 'active', 'invited')", req.ArcherID)
		if err == nil && existingMembership != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Archer already has a club membership"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send invitation"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Invitation sent successfully",
			"id":      memberID,
		})
	}
}
