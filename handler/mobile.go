package handler

import (
	"archeryhub-api/utils"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

type MobileScorekeeperLoginRequest struct {
	Code string `json:"code" binding:"required"`
}

// MobileScorekeeperLogin godoc
// @Summary      Scorekeeper login
// @Description  Login for scorekeepers using their unique code
// @Tags         Mobile - Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      MobileScorekeeperLoginRequest  true  "Scorekeeper login request"
// @Success      200      {object}  MobileLoginResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Failure      403      {object}  ErrorResponse
// @Router       /mobile/auth/scorekeeper/login [post]
func MobileScorekeeperLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req MobileScorekeeperLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Code is required"})
			return
		}

		var sk struct {
			UUID             string  `db:"uuid"`
			OrganizationUUID string  `db:"organization_uuid"`
			Code             string  `db:"code"`
			Name             string  `db:"name"`
			Email            string  `db:"email"`
			AvatarURL        *string `db:"avatar_url"`
			Status           string  `db:"status"`
			OrgSubStatus     *string `db:"org_sub_status"`
		}

		err := db.Get(&sk, `
			SELECT sk.uuid, sk.organization_uuid, sk.code, sk.name, IFNULL(sk.email, '') as email, sk.avatar_url, COALESCE(sk.status, '') as status,
                   o.subscription_status as org_sub_status
			FROM scorekeepers sk 
            JOIN organizations o ON sk.organization_uuid = o.uuid
            WHERE sk.code = ?`, req.Code)

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid scorekeeper code", "code": "invalid_code"})
			return
		}

		if sk.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Scorekeeper account is not active", "code": "account_inactive"})
			return
		}

		// Check Organization Subscription
		orgSub := "trial"
		if sk.OrgSubStatus != nil {
			orgSub = *sk.OrgSubStatus
		}

		if orgSub != "active" && orgSub != "trial" {
			c.JSON(http.StatusPaymentRequired, gin.H{
				"error":   "Organization subscription expired",
				"code":    "subscription_expired",
				"message": "Layanan scoring dihentikan sementara karena masa berlaku langganan organisasi telah berakhir. Silakan hubungi admin organisasi Anda.",
			})
			return
		}

		avatar := ""
		if sk.AvatarURL != nil {
			avatar = utils.MaskMediaURL(*sk.AvatarURL)
		}

		token, err := generateJWT(sk.UUID, sk.Email, "scorekeeper", "scorekeeper", sk.Name, avatar, sk.OrganizationUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		utils.LogActivity(db, sk.UUID, "", "mobile_login", "scorekeeper", sk.UUID, "Scorekeeper logged in via mobile", c.ClientIP(), c.Request.UserAgent())
		utils.LogScorekeeperAction(db, sk.UUID, sk.OrganizationUUID, "", "mobile_login", "Logged in via mobile app", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"uuid":       sk.UUID,
				"id":         sk.UUID,
				"username":   sk.Code,
				"full_name":  sk.Name,
				"email":      sk.Email,
				"avatar_url": avatar,
				"role":       "scorekeeper",
				"user_type":  "scorekeeper",
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
			"events":      events,
			"total_count": len(events), // Simple count for now, could be improved with separate COUNT query
		})
	}
}

// MobileScanTarget looks up a qualification target board by its QR/barcode code.
// Returns board info, session info, and all archers assigned to that board with their
// current score summary — used by the "Scan Barcode Bantalan" flow.
//
// MobileScanTarget godoc
// @Summary      Scan target barcode
// @Description  Look up a target board by its barcode code and return archer list with scores
// @Tags         Mobile - Scoring
// @Produce      json
// @Param        code  query     string  true  "Barcode / QR code on the target board"
// @Success      200   {object}  map[string]interface{}
// @Failure      400   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Router       /mobile/scan [get]
func MobileScanTarget(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		code := c.Query("code")
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "code is required"})
			return
		}

		type BoardInfo struct {
			UUID         string `db:"uuid"`
			SessionUUID  string `db:"session_uuid"`
			CategoryUUID string `db:"category_uuid"`
			BoardNumber  int    `db:"board_number"`
			Code         string `db:"code"`
			SessionName  string `db:"session_name"`
			EventUUID    string `db:"event_uuid"`
			EventName    string `db:"event_name"`
		}

		var board BoardInfo
		err := db.Get(&board, `
			SELECT 
				tbq.uuid, tbq.session_uuid, tbq.category_uuid, tbq.board_number, tbq.code,
				qs.name as session_name, qs.event_uuid,
				e.name as event_name
			FROM target_board_qualification tbq
			JOIN qualification_sessions qs ON tbq.session_uuid = qs.uuid
			JOIN events e ON qs.event_uuid = e.uuid
			WHERE tbq.code = ?
			LIMIT 1
		`, code)

		if err == nil {
			// --- Handle Qualification Board ---
			type ArcherInfo struct {
				AssignmentUUID  string  `db:"assignment_uuid" json:"assignment_uuid"`
				ParticipantUUID string  `db:"participant_uuid" json:"participant_uuid"`
				Position        string  `db:"position" json:"position"`       // e.g. "A"
				TargetName      string  `db:"target_name" json:"target_name"` // e.g. "003A"
				Name            string  `db:"name" json:"name"`
				Division        string  `db:"division" json:"division"`
				AvatarURL       *string `db:"avatar_url" json:"avatar_url"`
				CurrentScore    int     `db:"current_score" json:"current_score"`
				EndsCompleted   int     `db:"ends_completed" json:"ends_completed"`
				TotalEnds       int     `db:"total_ends" json:"total_ends"`
			}

			var archers []ArcherInfo
			err = db.Select(&archers, `
				SELECT
					qta.uuid as assignment_uuid,
				qta.participant_uuid,
				RIGHT(et.target_name, 1) as position,
				et.target_name as target_name,
				COALESCE(a.full_name, '') as name,
				COALESCE(CONCAT(bt.name, ' ', ag.name), '') as division,
					a.avatar_url,
					COALESCE(SUM(qes.total_score_end), 0) as current_score,
					COUNT(qes.uuid) as ends_completed,
					qs.total_ends
				FROM qualification_target_assignments qta
				JOIN event_targets et ON qta.target_uuid = et.uuid
				JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
				JOIN event_participants ep ON qta.participant_uuid = ep.uuid
				LEFT JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
				LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
				LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
				LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid AND qes.session_uuid = qta.session_uuid
				WHERE qta.target_board_id = ? AND qta.session_uuid = ?
				GROUP BY qta.uuid, qta.participant_uuid, et.target_name, a.full_name, bt.name, ag.name, qs.total_ends, a.avatar_url
				ORDER BY et.target_name ASC
			`, board.UUID, board.SessionUUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch archers", "details": err.Error()})
				return
			}

			for i := range archers {
				if archers[i].AvatarURL != nil {
					masked := utils.MaskMediaURL(*archers[i].AvatarURL)
					archers[i].AvatarURL = &masked
				}
			}

			c.JSON(http.StatusOK, gin.H{
				"type": "qualification",
				"board": gin.H{
					"uuid":          board.UUID,
					"board_number":  board.BoardNumber,
					"code":          board.Code,
					"session_uuid":  board.SessionUUID,
					"session_name":  board.SessionName,
					"event_uuid":    board.EventUUID,
					"event_name":    board.EventName,
					"category_uuid": board.CategoryUUID,
				},
				"archers": archers,
			})
			return
		}

		// --- If not found in Qual, try Elimination ---
		type ElimBoardInfo struct {
			UUID         string `db:"uuid"`
			BracketUUID  string `db:"bracket_uuid"`
			CategoryUUID string `db:"category_uuid"`
			BoardNumber  int    `db:"board_number"`
			Code         string `db:"code"`
			CategoryName string `db:"category_name"`
			EventUUID    string `db:"event_uuid"`
			EventName    string `db:"event_name"`
		}
		var eb ElimBoardInfo
		err = db.Get(&eb, `
			SELECT 
				tbe.uuid, tbe.bracket_uuid, tbe.category_uuid, tbe.board_number, tbe.code,
				COALESCE(ec.category_name_custom, '') as category_name,
				e.uuid as event_uuid, e.name as event_name
			FROM target_board_elimination tbe
			JOIN elimination_brackets eb ON tbe.bracket_uuid = eb.uuid
			JOIN events e ON eb.event_uuid = e.uuid
			LEFT JOIN event_categories ec ON tbe.category_uuid = ec.uuid
			WHERE tbe.code = ?
			LIMIT 1
		`, code)

		if err == nil {
			// Fetch matches on this elimination board
			type ElimArcherInfo struct {
				MatchUUID  string  `json:"match_uuid"`
				MatchID    string  `json:"match_id"`
				Side       string  `json:"side"` // A or B
				TargetName string  `json:"target_name"`
				Name       string  `json:"name"`
				Club       string  `json:"club"`
				AvatarURL  *string `json:"avatar_url"`
				Score      int     `json:"score"`
				Status     string  `json:"status"`
			}

			type MatchRow struct {
				MatchUUID  string         `db:"match_uuid"`
				MatchID    sql.NullString `db:"match_id"`
				TargetName string         `db:"target_name"`
				RoundNo    int            `db:"round_no"`
				MatchNo    int            `db:"match_no"`
				Status     string         `db:"status"`
				NameA      sql.NullString `db:"name_a"`
				AvatarA    sql.NullString `db:"avatar_a"`
				ClubA      sql.NullString `db:"club_a"`
				ScoreA     int            `db:"score_a"`
				NameB      sql.NullString `db:"name_b"`
				AvatarB    sql.NullString `db:"avatar_b"`
				ClubB      sql.NullString `db:"club_b"`
				ScoreB     int            `db:"score_b"`
			}
			var rows []MatchRow
			err = db.Select(&rows, `
				SELECT 
					em.uuid AS match_uuid, em.match_id, et.target_name, em.round_no, em.match_no, em.status,
					COALESCE(aA.full_name, tA.team_name, '') as name_a, aA.avatar_url as avatar_a, COALESCE(cA.name, '') as club_a,
					COALESCE(em.total_score_a, em.total_points_a, 0) as score_a,
					COALESCE(aB.full_name, tB.team_name, '') as name_b, aB.avatar_url as avatar_b, COALESCE(cB.name, '') as club_b,
					COALESCE(em.total_score_b, em.total_points_b, 0) as score_b
				FROM elimination_matches em
				JOIN event_targets et ON em.target_uuid = et.uuid
				LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
				LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
				LEFT JOIN clubs cA ON aA.club_id = cA.uuid
				LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
				LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
				LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
				LEFT JOIN clubs cB ON aB.club_id = cB.uuid
				LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
				WHERE em.bracket_uuid = ? AND et.board_number = ?
				ORDER BY em.round_no DESC, em.match_no ASC
			`, eb.BracketUUID, eb.BoardNumber)

			archers := []ElimArcherInfo{}
			for _, r := range rows {
				// Side A
				a := ElimArcherInfo{
					MatchUUID:  r.MatchUUID,
					MatchID:    r.MatchID.String,
					Side:       "A",
					TargetName: r.TargetName,
					Name:       r.NameA.String,
					Club:       r.ClubA.String,
					Score:      r.ScoreA,
					Status:     r.Status,
				}
				if r.AvatarA.Valid {
					masked := utils.MaskMediaURL(r.AvatarA.String)
					a.AvatarURL = &masked
				}
				archers = append(archers, a)

				// Side B
				b := ElimArcherInfo{
					MatchUUID:  r.MatchUUID,
					MatchID:    r.MatchID.String,
					Side:       "B",
					TargetName: r.TargetName,
					Name:       r.NameB.String,
					Club:       r.ClubB.String,
					Score:      r.ScoreB,
					Status:     r.Status,
				}
				if r.AvatarB.Valid {
					masked := utils.MaskMediaURL(r.AvatarB.String)
					b.AvatarURL = &masked
				}
				archers = append(archers, b)
			}

			c.JSON(http.StatusOK, gin.H{
				"type": "elimination",
				"board": gin.H{
					"uuid":          eb.UUID,
					"board_number":  eb.BoardNumber,
					"code":          eb.Code,
					"bracket_uuid":  eb.BracketUUID,
					"category_name": eb.CategoryName,
					"event_uuid":    eb.EventUUID,
					"event_name":    eb.EventName,
				},
				"archers": archers,
				"matches": rows,
			})
			return
		}

		c.JSON(http.StatusNotFound, gin.H{"error": "Code not found", "code": "not_found"})
	}
}

// MobileGetSessionBoards returns all target boards in a qualification session
// with each archer's current score summary — powers the "List Targets" leaderboard screen.
//
// MobileGetSessionBoards godoc
// @Summary      List all target boards in a session
// @Description  Get all boards and their archers with score summaries for a qualification session
// @Tags         Mobile - Scoring
// @Produce      json
// @Param        session_id  query  string  true  "Qualification session UUID"
// @Success      200         {object}  map[string]interface{}
// @Failure      400         {object}  ErrorResponse
// @Failure      404         {object}  ErrorResponse
// @Router       /mobile/sessions/boards [get]
func MobileGetSessionBoards(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Query("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required"})
			return
		}

		type SessionInfo struct {
			UUID         string `db:"uuid"`
			Name         string `db:"name"`
			EventUUID    string `db:"event_uuid"`
			EventName    string `db:"event_name"`
			TotalEnds    int    `db:"total_ends"`
			ArrowsPerEnd int    `db:"arrows_per_end"`
		}

		var session SessionInfo
		err := db.Get(&session, `
			SELECT qs.uuid, qs.name, qs.event_uuid, e.name as event_name, qs.total_ends, qs.arrows_per_end
			FROM qualification_sessions qs
			JOIN events e ON qs.event_uuid = e.uuid
			WHERE qs.uuid = ?
		`, sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}

		type ArcherAtBoard struct {
			AssignmentUUID  string  `db:"assignment_uuid" json:"assignment_uuid"`
			ParticipantUUID string  `db:"participant_uuid" json:"participant_uuid"`
			TargetName      string  `db:"target_name" json:"target_name"`
			BoardNumber     int     `db:"board_number" json:"board_number"`
			Position        string  `db:"position" json:"position"`
			Name            string  `db:"name" json:"name"`
			Division        string  `db:"division" json:"division"`
			AvatarURL       *string `db:"avatar_url" json:"avatar_url"`
			CurrentScore    int     `db:"current_score" json:"current_score"`
			EndsCompleted   int     `db:"ends_completed" json:"ends_completed"`
			LastEndScore    int     `db:"last_end_score" json:"last_end_score"`
			Rank            int     `json:"rank,omitempty"`
		}

		var archers []ArcherAtBoard
		err = db.Select(&archers, `
			SELECT
				qta.uuid as assignment_uuid,
				qta.participant_uuid,
				et.target_name,
				et.board_number,
				RIGHT(et.target_name, 1) as position,
				COALESCE(a.full_name, '') as name,
				COALESCE(CONCAT(bt.name, ' ', ag.name), '') as division,
				a.avatar_url,
				COALESCE(SUM(qes.total_score_end), 0) as current_score,
				COUNT(qes.uuid) as ends_completed,
				COALESCE((
					SELECT qes2.total_score_end
					FROM qualification_end_scores qes2
					WHERE qes2.participant_uuid = ep.uuid AND qes2.session_uuid = qta.session_uuid
					ORDER BY qes2.end_number DESC
					LIMIT 1
				), 0) as last_end_score
			FROM qualification_target_assignments qta
			JOIN event_targets et ON qta.target_uuid = et.uuid
			JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid AND qes.session_uuid = qta.session_uuid
			WHERE qta.session_uuid = ?
			GROUP BY qta.uuid, qta.participant_uuid, et.target_name, et.board_number, a.full_name, bt.name, ag.name, a.avatar_url, ep.uuid
			ORDER BY et.board_number ASC, et.target_name ASC
		`, sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch boards", "details": err.Error()})
			return
		}

		// Assign rank by current_score desc
		for i := range archers {
			if archers[i].AvatarURL != nil {
				masked := utils.MaskMediaURL(*archers[i].AvatarURL)
				archers[i].AvatarURL = &masked
			}
			archers[i].Rank = i + 1
		}

		c.JSON(http.StatusOK, gin.H{
			"session": gin.H{
				"uuid":           session.UUID,
				"name":           session.Name,
				"event_uuid":     session.EventUUID,
				"event_name":     session.EventName,
				"total_ends":     session.TotalEnds,
				"arrows_per_end": session.ArrowsPerEnd,
			},
			"archers": archers,
		})
	}
}

// MobileGetAssignmentScoreDetail returns the full arrow-by-arrow score history for a
// participant's target assignment — powers the "Detail Score Target" screen.
//
// MobileGetAssignmentScoreDetail godoc
// @Summary      Get full score detail for an assignment
// @Description  Returns all ends and arrow scores for a participant's qualification assignment
// @Tags         Mobile - Scoring
// @Produce      json
// @Param        assignmentId  path   string  true  "Target assignment UUID"
// @Success      200           {object}  map[string]interface{}
// @Failure      400           {object}  ErrorResponse
// @Failure      404           {object}  ErrorResponse
// @Router       /mobile/assignments/{assignmentId}/detail [get]
func MobileGetAssignmentScoreDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")
		if assignmentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assignmentId is required"})
			return
		}

		type AssignmentMeta struct {
			AssignmentUUID  string  `db:"assignment_uuid"`
			ParticipantUUID string  `db:"participant_uuid"`
			SessionUUID     string  `db:"session_uuid"`
			SessionName     string  `db:"session_name"`
			EventName       string  `db:"event_name"`
			TargetName      string  `db:"target_name"`
			ArcherName      string  `db:"archer_name"`
			Division        string  `db:"division"`
			AvatarURL       *string `db:"avatar_url"`
			TotalEnds       int     `db:"total_ends"`
			ArrowsPerEnd    int     `db:"arrows_per_end"`
		}

		var meta AssignmentMeta
		err := db.Get(&meta, `
			SELECT
				qta.uuid as assignment_uuid,
				qta.participant_uuid,
				qta.session_uuid,
				qs.name as session_name,
				e.name as event_name,
				et.target_name,
				COALESCE(a.full_name, '') as archer_name,
				COALESCE(CONCAT(bt.name, ' ', ag.name), '') as division,
				a.avatar_url,
				qs.total_ends,
				qs.arrows_per_end
			FROM qualification_target_assignments qta
			JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			JOIN events e ON qs.event_uuid = e.uuid
			JOIN event_targets et ON qta.target_uuid = et.uuid
			JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			WHERE qta.uuid = ?
			LIMIT 1
		`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}

		if meta.AvatarURL != nil {
			masked := utils.MaskMediaURL(*meta.AvatarURL)
			meta.AvatarURL = &masked
		}

		type ArrowScore struct {
			ArrowNumber int  `db:"arrow_number" json:"arrow_number"`
			Score       int  `db:"score" json:"score"`
			IsX         bool `db:"is_x" json:"is_x"`
		}

		type EndScore struct {
			EndNumber    int          `db:"end_number" json:"end_number"`
			EndScoreUUID string       `db:"end_score_uuid" json:"end_score_uuid"`
			EndTotal     int          `db:"end_total" json:"end_total"`
			XCount       int          `db:"x_count" json:"x_count"`
			TenCount     int          `db:"ten_count" json:"ten_count"`
			CumTotal     int          `json:"cumulative_total"`
			Arrows       []ArrowScore `json:"arrows"`
		}

		var ends []EndScore
		err = db.Select(&ends, `
			SELECT 
				qes.end_number,
				qes.uuid as end_score_uuid,
				qes.total_score_end as end_total,
				qes.x_count_end as x_count,
				qes.ten_count_end as ten_count
			FROM qualification_end_scores qes
			WHERE qes.participant_uuid = ? AND qes.session_uuid = ?
			ORDER BY qes.end_number ASC
		`, meta.ParticipantUUID, meta.SessionUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch end scores"})
			return
		}

		// For each end, fetch arrow scores and compute cumulative total
		cumTotal := 0
		totalXCount := 0
		totalTenPlusXCount := 0

		for i := range ends {
			cumTotal += ends[i].EndTotal
			ends[i].CumTotal = cumTotal
			totalXCount += ends[i].XCount
			totalTenPlusXCount += ends[i].TenCount + ends[i].XCount

			var arrows []ArrowScore
			_ = db.Select(&arrows, `
				SELECT arrow_number, score, is_x
				FROM qualification_arrow_scores
				WHERE end_score_uuid = ?
				ORDER BY arrow_number ASC
			`, ends[i].EndScoreUUID)
			if arrows == nil {
				arrows = []ArrowScore{}
			}
			ends[i].Arrows = arrows
		}

		c.JSON(http.StatusOK, gin.H{
			"assignment": gin.H{
				"uuid":           meta.AssignmentUUID,
				"session_uuid":   meta.SessionUUID,
				"session_name":   meta.SessionName,
				"event_name":     meta.EventName,
				"target_name":    meta.TargetName,
				"archer_name":    meta.ArcherName,
				"division":       meta.Division,
				"avatar_url":     meta.AvatarURL,
				"total_ends":     meta.TotalEnds,
				"arrows_per_end": meta.ArrowsPerEnd,
			},
			"summary": gin.H{
				"total_score":      cumTotal,
				"total_x":          totalXCount,
				"total_ten_plus_x": totalTenPlusXCount,
				"ends_completed":   len(ends),
			},
			"ends": ends,
		})
	}
}

// ─── Archer Auth ──────────────────────────────────────────────────────────────

// MobileArcherLogin godoc
// @Summary      Archer login
// @Description  Login for archers using email and password
// @Tags         Mobile - Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object{email=string,password=string}  true  "Login credentials"
// @Success      200      {object}  MobileLoginResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      401      {object}  ErrorResponse
// @Router       /mobile/auth/login [post]
func MobileArcherLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var archer struct {
			UUID      string  `db:"uuid"`
			ID        string  `db:"id"`
			Username  string  `db:"username"`
			Email     string  `db:"email"`
			Password  string  `db:"password"`
			FullName  string  `db:"full_name"`
			AvatarURL *string `db:"avatar_url"`
			Status    string  `db:"status"`
		}
		err := db.Get(&archer, `SELECT uuid, id, username, email, COALESCE(password,'') as password, full_name, avatar_url, COALESCE(status,'') as status FROM archers WHERE email = ?`, req.Email)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password", "code": "invalid_credentials"})
			return
		}

		if archer.Status != "active" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is not active", "code": "account_inactive"})
			return
		}
		if archer.Password == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "This account uses Google sign-in. Please sign in with Google.", "code": "use_google_signin"})
			return
		}
		if archer.Password != req.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password", "code": "invalid_credentials"})
			return
		}

		avatar := ""
		if archer.AvatarURL != nil {
			avatar = utils.MaskMediaURL(*archer.AvatarURL)
		}
		token, err := generateJWT(archer.UUID, archer.Email, "archer", "archer", archer.FullName, avatar, "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"uuid":       archer.UUID,
				"id":         archer.ID,
				"username":   archer.Username,
				"full_name":  archer.FullName,
				"email":      archer.Email,
				"avatar_url": avatar,
				"role":       "archer",
				"user_type":  "archer",
			},
		})
	}
}

// MobileArcherRegister godoc
// @Summary      Archer registration
// @Description  Register a new archer account
// @Tags         Mobile - Authentication
// @Accept       json
// @Produce      json
// @Param        request  body      object  true  "Registration info"
// @Success      201      {object}  MobileLoginResponse
// @Failure      400      {object}  ErrorResponse
// @Failure      409      {object}  ErrorResponse
// @Router       /mobile/auth/register [post]
func MobileArcherRegister(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Email       string `json:"email" binding:"required,email"`
			Password    string `json:"password" binding:"required,min=6"`
			FullName    string `json:"full_name" binding:"required"`
			Phone       string `json:"phone"`
			Gender      string `json:"gender"`
			DateOfBirth string `json:"date_of_birth"`
			City        string `json:"city"`
			BowType     string `json:"bow_type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var exists bool
		db.Get(&exists, `SELECT EXISTS(SELECT 1 FROM archers WHERE email = ?)`, req.Email)
		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Email already registered", "code": "email_exists"})
			return
		}

		userID := uuid.New().String()

		var lastID string
		_ = db.Get(&lastID, "SELECT id FROM archers WHERE id LIKE 'ARC-%' ORDER BY id DESC LIMIT 1")
		nextIDNum := 1
		if lastID != "" {
			parts := strings.Split(lastID, "-")
			if len(parts) == 2 {
				fmt.Sscanf(parts[1], "%d", &nextIDNum)
				nextIDNum++
			}
		}
		athleteID := fmt.Sprintf("ARC-%04d", nextIDNum)

		username := utils.CleanUsername(req.FullName)
		if username == "" {
			username = "archer"
		}
		username = username + "-" + userID[:8]

		_, err := db.Exec(`
			INSERT INTO archers (uuid, id, username, email, password, full_name, phone, status, is_verified, gender, date_of_birth, city, bow_type)
			VALUES (?, ?, ?, ?, ?, ?, ?, 'active', 1, ?, ?, ?, ?)
		`, userID, athleteID, username, req.Email, req.Password, req.FullName, req.Phone, req.Gender, req.DateOfBirth, req.City, req.BowType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account: " + err.Error()})
			return
		}

		token, err := generateJWT(userID, req.Email, "archer", "archer", req.FullName, "", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"token": token,
			"user": gin.H{
				"uuid":      userID,
				"id":        athleteID,
				"username":  username,
				"full_name": req.FullName,
				"email":     req.Email,
				"role":      "archer",
				"user_type": "archer",
			},
		})
	}
}

// ─── Events (public) ──────────────────────────────────────────────────────────

// MobileGetEventDetail godoc
// @Summary      Event detail by slug
// @Description  Get full event detail including categories and registration status
// @Tags         Mobile - Events
// @Produce      json
// @Param        slug  path   string  true  "Event slug or UUID"
// @Success      200   {object}  map[string]interface{}
// @Failure      404   {object}  ErrorResponse
// @Router       /mobile/events/{slug} [get]
func MobileGetEventDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")

		var event struct {
			UUID             string  `db:"uuid" json:"uuid"`
			Name             string  `db:"name" json:"name"`
			Slug             string  `db:"slug" json:"slug"`
			Location         string  `db:"location" json:"location"`
			StartDate        string  `db:"start_date" json:"start_date"`
			EndDate          string  `db:"end_date" json:"end_date"`
			RegStartDate     *string `db:"registration_start" json:"registration_start"`
			RegEndDate       *string `db:"registration_deadline" json:"registration_deadline"`
			LogoURL          *string `db:"logo_url" json:"logo_url"`
			BannerURL        *string `db:"banner_url" json:"banner_url"`
			Description      *string `db:"description" json:"description"`
			Status           string  `db:"status" json:"status"`
			OrganizerName    string  `db:"organizer_name" json:"organizer_name"`
			OrganizerAvatar  *string `db:"organizer_avatar" json:"organizer_avatar"`
			ParticipantCount int     `db:"participant_count" json:"participant_count"`
		}
		err := db.Get(&event, `
			SELECT e.uuid, e.name, e.slug, e.location, e.start_date, e.end_date,
				e.registration_start, e.registration_deadline,
				e.logo_url, e.banner_url, e.description, e.status,
				COALESCE(o.full_name, '') as organizer_name,
				o.avatar_url as organizer_avatar,
				COUNT(DISTINCT ep.archer_id) as participant_count
			FROM events e
			LEFT JOIN (
				SELECT uuid as id, name as full_name, avatar_url FROM organizations
				UNION ALL
				SELECT uuid as id, name as full_name, avatar_url FROM clubs
			) o ON e.organizer_id = o.id
			LEFT JOIN event_participants ep ON e.uuid = ep.event_id
			WHERE e.slug = ? OR e.uuid = ?
			GROUP BY e.uuid
		`, slug, slug)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		if event.LogoURL != nil {
			masked := utils.MaskMediaURL(*event.LogoURL)
			event.LogoURL = &masked
		}
		if event.BannerURL != nil {
			masked := utils.MaskMediaURL(*event.BannerURL)
			event.BannerURL = &masked
		}
		if event.OrganizerAvatar != nil {
			masked := utils.MaskMediaURL(*event.OrganizerAvatar)
			event.OrganizerAvatar = &masked
		}

		type Category struct {
			UUID             string  `db:"uuid" json:"uuid"`
			Name             string  `db:"name" json:"name"`
			RegistrationFee  float64 `db:"registration_fee" json:"registration_fee"`
			MaxParticipants  *int    `db:"max_participants" json:"max_participants"`
			ParticipantCount int     `db:"participant_count" json:"participant_count"`
		}
		var categories []Category
		db.Select(&categories, `
			SELECT ec.uuid,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name,''), ' ', COALESCE(rag.name,''), ' ', COALESCE(rgd.name,''))) as name,
				COALESCE(ec.registration_fee, 0) as registration_fee,
				ec.max_participants,
				COUNT(DISTINCT ep.archer_id) as participant_count
			FROM event_categories ec
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN event_participants ep ON ep.event_category_id = ec.uuid
			WHERE ec.event_id = ?
			GROUP BY ec.uuid
			ORDER BY ec.sort_order ASC, ec.created_at ASC
		`, event.UUID)
		if categories == nil {
			categories = []Category{}
		}

		// Check if authenticated archer is already registered
		isRegistered := false
		registrationStatus := ""
		if userID, ok := c.Get("user_id"); ok && userID != nil {
			var reg struct {
				Status string `db:"payment_status"`
			}
			if regErr := db.Get(&reg, `SELECT payment_status FROM event_participants WHERE event_id = ? AND archer_id = ? LIMIT 1`, event.UUID, fmt.Sprintf("%v", userID)); regErr == nil {
				isRegistered = true
				registrationStatus = reg.Status
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"event":               event,
			"categories":          categories,
			"is_registered":       isRegistered,
			"registration_status": registrationStatus,
		})
	}
}

// ─── Archer Account (requires auth) ──────────────────────────────────────────

// MobileRegisterForEvent godoc
// @Summary      Register archer for event
// @Description  Authenticated archer self-registers for one or more event categories
// @Tags         Mobile - Archer
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id       path  string  true  "Event UUID or slug"
// @Param        request  body  object  true  "Category IDs and payment type"
// @Success      201      {object}  map[string]interface{}
// @Failure      400      {object}  ErrorResponse
// @Failure      404      {object}  ErrorResponse
// @Router       /mobile/archer/events/{id}/register [post]
func MobileRegisterForEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		var req struct {
			EventCategoryIDs []string `json:"event_category_ids" binding:"required,min=1"`
			PaymentType      string   `json:"payment_type"` // "manual" or "gateway"
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var archerUUID string
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ?`, fmt.Sprintf("%v", userID)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Archer account not found"})
			return
		}

		var event struct {
			UUID        string `db:"uuid"`
			OrganizerID string `db:"organizer_id"`
		}
		if err := db.Get(&event, `SELECT uuid, organizer_id FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		var orgStatus string
		db.Get(&orgStatus, `
			SELECT COALESCE(s, 'trial') FROM (
				SELECT subscription_status as s FROM organizations WHERE uuid = ?
				UNION ALL
				SELECT 'trial' as s FROM clubs WHERE uuid = ?
			) combined LIMIT 1`, event.OrganizerID, event.OrganizerID)
		if orgStatus != "" && orgStatus != "active" && orgStatus != "trial" {
			c.JSON(http.StatusPaymentRequired, gin.H{"error": "Pendaftaran ditutup sementara oleh sistem", "code": "organizer_subscription_expired"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		paymentStatus := "menunggu acc"
		if req.PaymentType == "gateway" {
			paymentStatus = "unpaid"
		}

		var registeredCategories []string
		for _, catID := range req.EventCategoryIDs {
			var existingID string
			if err := tx.Get(&existingID, `SELECT uuid FROM event_participants WHERE event_id = ? AND archer_id = ? AND event_category_id = ?`, event.UUID, archerUUID, catID); err == nil {
				continue
			}
			participantID := uuid.New().String()
			if _, err = tx.Exec(`
				INSERT INTO event_participants (uuid, event_id, archer_id, event_category_id, payment_status, registration_source, registration_date)
				VALUES (?, ?, ?, ?, ?, 'mobile_app', NOW())
			`, participantID, event.UUID, archerUUID, catID, paymentStatus); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register: " + err.Error()})
				return
			}
			registeredCategories = append(registeredCategories, catID)
		}

		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit"})
			return
		}

		if registeredCategories == nil {
			registeredCategories = []string{}
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":               "Pendaftaran berhasil",
			"registered_categories": registeredCategories,
			"payment_status":        paymentStatus,
		})
	}
}

// MobileGetMyRegistration godoc
// @Summary      Get archer's registration for an event
// @Description  Returns registration details including payment status and payment instructions
// @Tags         Mobile - Archer
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Event UUID or slug"
// @Success      200 {object}  map[string]interface{}
// @Failure      404 {object}  ErrorResponse
// @Router       /mobile/archer/events/{id}/registration [get]
func MobileGetMyRegistration(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")

		var archerUUID string
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ?`, fmt.Sprintf("%v", userID)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Archer not found"})
			return
		}

		var eventUUID string
		if err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		type Registration struct {
			ParticipantID    string  `db:"uuid" json:"id"`
			CategoryName     string  `db:"category_name" json:"category_name"`
			PaymentStatus    string  `db:"payment_status" json:"payment_status"`
			PaymentAmount    float64 `db:"payment_amount" json:"payment_amount"`
			PaymentMethod    *string `db:"payment_method" json:"payment_method"`
			TripayRef        *string `db:"tripay_reference" json:"tripay_reference"`
			CheckoutURL      *string `db:"checkout_url" json:"checkout_url"`
			Instructions     *string `db:"instructions" json:"instructions"`
			VANumber         *string `db:"va_number" json:"va_number"`
			PayCode          *string `db:"pay_code" json:"pay_code"`
			QRURL            *string `db:"qr_url" json:"qr_url"`
			RegistrationDate string  `db:"registration_date" json:"registration_date"`
		}
		var registrations []Registration
		err := db.Select(&registrations, `
			SELECT
				ep.uuid,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name,''), ' ', COALESCE(rag.name,''), ' ', COALESCE(rgd.name,''))) as category_name,
				ep.payment_status,
				COALESCE(ep.payment_amount, 0) as payment_amount,
				pt.payment_method,
				pt.tripay_reference,
				pt.checkout_url,
				pt.instructions,
				pt.va_number,
				pt.pay_code,
				pt.qr_url,
				ep.registration_date
			FROM event_participants ep
			LEFT JOIN event_categories ec ON ep.event_category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			LEFT JOIN payment_transactions pt ON pt.participant_id = ep.uuid AND pt.status = 'pending'
			WHERE ep.event_id = ? AND ep.archer_id = ?
			ORDER BY ep.registration_date DESC
		`, eventUUID, archerUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch registration"})
			return
		}
		if registrations == nil {
			registrations = []Registration{}
		}

		c.JSON(http.StatusOK, gin.H{
			"event_id":      eventUUID,
			"registrations": registrations,
		})
	}
}

// MobileGetMyEvents godoc
// @Summary      List archer's registered events
// @Description  Returns all events the authenticated archer has registered for
// @Tags         Mobile - Archer
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  map[string]interface{}
// @Router       /mobile/archer/events [get]
func MobileGetMyEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var archerUUID string
		if err := db.Get(&archerUUID, `SELECT uuid FROM archers WHERE uuid = ?`, fmt.Sprintf("%v", userID)); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Archer not found"})
			return
		}

		type MyEvent struct {
			EventUUID        string  `db:"event_uuid" json:"event_uuid"`
			EventName        string  `db:"event_name" json:"event_name"`
			EventSlug        string  `db:"event_slug" json:"event_slug"`
			Location         string  `db:"location" json:"location"`
			StartDate        string  `db:"start_date" json:"start_date"`
			EndDate          string  `db:"end_date" json:"end_date"`
			LogoURL          *string `db:"logo_url" json:"logo_url"`
			CategoryName     string  `db:"category_name" json:"category_name"`
			PaymentStatus    string  `db:"payment_status" json:"payment_status"`
			RegistrationDate string  `db:"registration_date" json:"registration_date"`
		}
		var events []MyEvent
		err := db.Select(&events, `
			SELECT
				e.uuid as event_uuid, e.name as event_name, e.slug as event_slug,
				e.location, e.start_date, e.end_date, e.logo_url,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name,''), ' ', COALESCE(rag.name,''), ' ', COALESCE(rgd.name,''))) as category_name,
				ep.payment_status,
				ep.registration_date
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			LEFT JOIN event_categories ec ON ep.event_category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE ep.archer_id = ?
			ORDER BY e.start_date DESC
		`, archerUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
			return
		}
		if events == nil {
			events = []MyEvent{}
		}

		for i := range events {
			if events[i].LogoURL != nil {
				masked := utils.MaskMediaURL(*events[i].LogoURL)
				events[i].LogoURL = &masked
			}
		}

		c.JSON(http.StatusOK, gin.H{"events": events, "total": len(events)})
	}
}
