package handler

import (
	"archeryhub-api/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// MobileUser represents user information in mobile login response
type MobileUser struct {
	UUID      string `json:"uuid"`
	ID        string `json:"id"`
	Username  string `json:"username"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Role      string `json:"role"`
	UserType  string `json:"user_type"`
}

// MobileLoginResponse represents the response body for mobile login
type MobileLoginResponse struct {
	Token string     `json:"token"`
	User  MobileUser `json:"user"`
}

// MobileEvent represents event information optimized for mobile
type MobileEvent struct {
	UUID               string  `db:"uuid" json:"uuid"`
	Name               string  `db:"name" json:"name"`
	Location           string  `db:"location" json:"location"`
	StartDate          string  `db:"start_date" json:"start_date"`
	EndDate            string  `db:"end_date" json:"end_date"`
	LogoURL            *string `db:"logo_url" json:"logo_url"`
	BannerURL          *string `db:"banner_url" json:"banner_url"`
	OrganizerName      string  `db:"organizer_name" json:"organizer_name"`
	OrganizerAvatarURL *string `db:"organizer_avatar_url" json:"organizer_avatar_url"`
	ParticipantCount   int     `db:"participant_count" json:"participant_count"`
}

// MobileEventsResponse represents the list of events for mobile
type MobileEventsResponse struct {
	Events     []MobileEvent `json:"events"`
	TotalCount int           `json:"total_count"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// MessageResponse represents a standard success message response
type MessageResponse struct {
	Message string `json:"message"`
}

// MobileHello godoc
func MobileHello() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Welcome to ArcheryHub Mobile API",
			"status":  "active",
		})
	}
}

type MobileLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

// MobileLogin godoc
// @Summary      Mobile login
// @Description  Specialized login for mobile app
// @Tags         Mobile - Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      MobileLoginRequest  true  "Login request"
// @Success      200      {object}  MobileLoginResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Router       /mobile/auth/login [post]
func MobileLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		type UserResult struct {
			UUID      string  `db:"uuid"`
			ID        string  `db:"id"`
			Username  string  `db:"slug"`
			Email     string  `db:"email"`
			Password  string  `db:"password"`
			FullName  string  `db:"full_name"`
			AvatarURL *string `db:"avatar_url"`
			Role      string  `db:"role"`
			Status    string  `db:"status"`
			Type      string
			OrgUUID   string  `db:"organization_uuid"`
		}

		var user UserResult
		found := false

		// 1. Check if it's a scorekeeper login by Code
		if req.Code != "" || (len(req.Email) == 5 && req.Password == "") {
			code := req.Code
			if code == "" {
				code = req.Email
			}
			err := db.Get(&user, "SELECT uuid, uuid as id, code as slug, IFNULL(email, '') as email, '' as password, name as full_name, avatar_url, 'scorekeeper' as role, COALESCE(status,'') as status, organization_uuid FROM scorekeepers WHERE code = ?", code)
			if err == nil {
				user.Type = "scorekeeper"
				found = true
				// For scorekeepers by code, we skip the password check later
				req.Password = "" 
				user.Password = ""
			}
		}

		// 2. Standard login via Email/Password
		if !found && req.Email != "" && req.Password != "" {
			// Check archers
			err := db.Get(&user, "SELECT uuid, id, username as slug, email, COALESCE(password,'') as password, full_name, avatar_url, 'archer' as role, COALESCE(status,'') as status, '' as organization_uuid FROM archers WHERE email = ?", req.Email)
			if err == nil {
				user.Type = "archer"
				found = true
			}

			// Check organizations
			if !found {
				err = db.Get(&user, "SELECT uuid, uuid as id, slug, email, COALESCE(password,'') as password, name as full_name, avatar_url, 'organization' as role, COALESCE(status,'') as status, uuid as organization_uuid FROM organizations WHERE email = ?", req.Email)
				if err == nil {
					user.Type = "organization"
					found = true
				}
			}

			// Check clubs
			if !found {
				err = db.Get(&user, "SELECT uuid, uuid as id, slug, email, COALESCE(password,'') as password, name as full_name, avatar_url, 'club' as role, COALESCE(status,'') as status, '' as organization_uuid FROM clubs WHERE email = ?", req.Email)
				if err == nil {
					user.Type = "club"
					found = true
				}
			}
		}

		if !found {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials", "code": "invalid_credentials"})
			return
		}

		if user.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active", "code": "account_inactive"})
			return
		}

		// Skip password checks for scorekeeper code-based login
		if user.Type != "scorekeeper" {
			if user.Password == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "This account uses Google sign-in. Please sign in with Google.",
					"code":  "use_google_signin",
				})
				return
			}

			if user.Password != req.Password {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password", "code": "invalid_credentials"})
				return
			}
		}

		avatar := ""
		if user.AvatarURL != nil {
			avatar = utils.MaskMediaURL(*user.AvatarURL)
		}
		
		token, err := generateJWT(user.UUID, user.Email, user.Role, user.Type, user.FullName, avatar, user.OrgUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		// Update activity
		utils.LogActivity(db, user.UUID, "", "mobile_login", user.Type, user.UUID, "User logged in via mobile", c.ClientIP(), c.Request.UserAgent())
		if user.Type == "scorekeeper" {
			utils.LogScorekeeperAction(db, user.UUID, user.OrgUUID, "", "mobile_login", "Logged in via mobile app", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"uuid":       user.UUID,
				"id":         user.ID,
				"username":   user.Username,
				"full_name":  user.FullName,
				"email":      user.Email,
				"avatar_url": avatar,
				"role":       user.Role,
				"user_type":  user.Type,
			},
		})
	}
}

// MobileListEvents godoc
// @Summary      List mobile events
// @Description  Get events optimized for mobile view
// @Tags         Mobile - Events
// @Produce      json
// @Param        limit   query     int     false  "Limit"
// @Param        offset  query     int     false  "Offset"
// @Param        search  query     string  false  "Search term"
// @Success      200     {object}  MobileEventsResponse
// @Failure      500     {object}  ErrorResponse
// @Router       /mobile/events [get]
// MobileGetScorekeeperMe returns current scorekeeper profile
func MobileGetScorekeeperMe(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		if userType != "scorekeeper" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only scorekeepers can access this"})
			return
		}

		var sk struct {
			UUID             string  `db:"uuid" json:"uuid"`
			OrganizationUUID string  `db:"organization_uuid" json:"organization_uuid"`
			Code             string  `db:"code" json:"code"`
			Name             string  `db:"name" json:"name"`
			Email            *string `db:"email" json:"email"`
			AvatarURL        *string `db:"avatar_url" json:"avatar_url"`
			Status           string  `db:"status" json:"status"`
			OrgName          string  `db:"org_name" json:"organization_name"`
		}

		err := db.Get(&sk, `
			SELECT sk.uuid, sk.organization_uuid, sk.code, sk.name, sk.email, sk.avatar_url, sk.status, o.name as org_name 
			FROM scorekeepers sk 
			JOIN organizations o ON sk.organization_uuid = o.uuid 
			WHERE sk.uuid = ?`, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Scorekeeper not found"})
			return
		}

		if sk.AvatarURL != nil {
			masked := utils.MaskMediaURL(*sk.AvatarURL)
			sk.AvatarURL = &masked
		}

		c.JSON(http.StatusOK, sk)
	}
}

// MobileGetScorekeeperEvents returns events for scorekeeper's organization
func MobileGetScorekeeperEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgID, _ := c.Get("org_id")
		userType, _ := c.Get("user_type")

		if userType != "scorekeeper" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Only scorekeepers can access this"})
			return
		}

		var events []MobileEvent
		err := db.Select(&events, `
			SELECT 
				t.uuid, t.name, t.location, t.start_date, t.end_date, t.logo_url, t.banner_url,
				o.name as organizer_name,
				o.avatar_url as organizer_avatar_url,
				(SELECT COUNT(DISTINCT archer_id) FROM event_participants WHERE event_id = t.uuid) as participant_count
			FROM events t
			JOIN organizations o ON t.organizer_id = o.uuid
			WHERE t.organizer_id = ?
			ORDER BY t.start_date DESC`, orgID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events", "details": err.Error()})
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
			"total_count": len(events),
		})
	}
}

func MobileListEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		search := c.Query("search")

		whereClause := "WHERE t.status = 'published'"
		args := []interface{}{}

		if search != "" {
			whereClause += ` AND (t.name LIKE ? OR t.location LIKE ?)`
			searchTerm := "%" + search + "%"
			args = append(args, searchTerm, searchTerm)
		}

		query := `
			SELECT 
				t.uuid, t.name, t.location, t.start_date, t.end_date, t.logo_url, t.banner_url,
				u.full_name as organizer_name,
				u.avatar_url as organizer_avatar_url,
				COUNT(DISTINCT tp.archer_id) as participant_count
			FROM events t
			LEFT JOIN (
				SELECT uuid as id, name as full_name, avatar_url FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, avatar_url FROM clubs
			) u ON t.organizer_id = u.id
			LEFT JOIN event_participants tp ON t.uuid = tp.event_id
			` + whereClause + `
			GROUP BY t.uuid, u.full_name, u.avatar_url
			ORDER BY t.start_date DESC
			LIMIT ? OFFSET ?
		`
		args = append(args, limit, offset)

		var events []MobileEvent
		err := db.Select(&events, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events", "details": err.Error()})
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
			"events": events,
			"total_count": len(events), // Simple count for now, could be improved with separate COUNT query
		})
	}
}
