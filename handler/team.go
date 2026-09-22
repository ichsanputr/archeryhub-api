package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"Archeris-api/models"
	"Archeris-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CreateTeam creates a new team for an event category
func CreateTeam(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")
		userID, _ := c.Get("user_id")

		var req models.CreateTeamRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// 1. Check if an elimination bracket already exists/active for this category
		var bracketCount int
		_ = db.Get(&bracketCount, `
			SELECT COUNT(*) FROM elimination_brackets 
			WHERE category_uuid = ? AND tournament_uuid = ? AND status != 'draft'
		`, req.CategoryID, eventUUID)
		if bracketCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Bagan eliminasi untuk kategori ini sudah dibuat. Hapus atau reset bagan eliminasi terlebih dahulu untuk mengubah susunan tim.",
			})
			return
		}

		// 2. Get category info and required team size
		var catInfo struct {
			TypeCode   string `db:"type_code"`
			GenderCode string `db:"gender_code"`
			TeamSize   int    `db:"team_size"`
		}
		err = db.Get(&catInfo, `
			SELECT 
				ret.code as type_code, 
				COALESCE(rgd.code, '') as gender_code,
				CASE 
					WHEN ret.code = 'mixed_team' THEN 2 
					WHEN ret.code = 'team' THEN 3 
					ELSE 1 
				END as team_size
			FROM tournament_categories ec
			JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE ec.uuid = ?`, req.CategoryID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
			return
		}

		if len(req.MemberIDs) != catInfo.TeamSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Jumlah anggota harus tepat %d orang untuk kategori ini", catInfo.TeamSize),
			})
			return
		}

		// 3. Validate member genders if mixed team
		if catInfo.TypeCode == "mixed_team" || strings.Contains(strings.ToLower(catInfo.TypeCode), "mixed") {
			type MemberGender struct {
				Gender string `db:"gender"`
			}
			query, args, _ := sqlx.In(`
				SELECT COALESCE(a.gender, '') as gender
				FROM tournament_participants tp
				JOIN archers a ON tp.archer_id = a.uuid
				WHERE tp.uuid IN (?)
			`, req.MemberIDs)
			query = db.Rebind(query)
			var genders []MemberGender
			if err := db.Select(&genders, query, args...); err != nil || len(genders) != 2 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal memverifikasi data anggota tim"})
				return
			}
			hasMale := false
			hasFemale := false
			for _, g := range genders {
				if g.Gender == "male" || g.Gender == "men" {
					hasMale = true
				}
				if g.Gender == "female" || g.Gender == "women" {
					hasFemale = true
				}
			}
			if !hasMale || !hasFemale {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Mixed team wajib terdiri dari 1 pemanah putra dan 1 pemanah putri"})
				return
			}
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		teamID := uuid.New().String()

		_, err = tx.Exec(`
			INSERT INTO teams (uuid, tournament_id, event_id, category_id, team_name, status)
			VALUES (?, ?, ?, ?, ?, 'active')
		`, teamID, eventUUID, req.CategoryID, req.CategoryID, req.TeamName)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat tim", "details": err.Error()})
			return
		}

		// 4. Add team members
		for i, participantID := range req.MemberIDs {
			memberID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO team_members (uuid, team_id, participant_id, member_order)
				VALUES (?, ?, ?, ?)
			`, memberID, teamID, participantID, i+1)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan anggota tim"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data tim"})
			return
		}

		utils.LogActivity(db, userID.(string), eventUUID, "team_created", "team", teamID,
			fmt.Sprintf("Created team: %s", req.TeamName), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      teamID,
			"message": "Tim berhasil dibuat",
		})
	}
}

// GetTeams returns all teams for an event with their members and scores
func GetTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")
		categoryID := c.Query("category_id")

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT 
				t.uuid, 
				t.tournament_id, 
				t.event_id, 
				COALESCE(t.category_id, t.event_id) as category_id,
				t.team_name, 
				t.team_rank, 
				t.total_score, 
				t.total_x_count, 
				t.status, 
				t.created_at, 
				t.updated_at, 
				COUNT(tm.uuid) as member_count 
			FROM teams t
			LEFT JOIN team_members tm ON t.uuid = tm.team_id
			WHERE t.tournament_id = ?
		`
		args := []interface{}{eventUUID}

		if categoryID != "" {
			query += " AND (t.event_id = ? OR t.category_id = ?)"
			args = append(args, categoryID, categoryID)
		}

		query += " GROUP BY t.uuid, t.tournament_id, t.event_id, t.category_id, t.team_name, t.team_rank, t.total_score, t.total_x_count, t.status, t.created_at, t.updated_at ORDER BY t.team_rank ASC, t.total_score DESC, t.total_x_count DESC"

		var teams []models.Team
		err = db.Select(&teams, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Gagal mengambil data tim",
				"details": err.Error(),
			})
			return
		}

		// Build response with members for each team
		type MemberInfo struct {
			UUID          string `json:"id" db:"uuid"`
			ParticipantID string `json:"participant_id" db:"participant_id"`
			FullName      string `json:"full_name" db:"full_name"`
			Gender        string `json:"gender" db:"gender"`
			ClubName      string `json:"club_name" db:"club_name"`
			TotalScore    int    `json:"total_score" db:"total_score"`
			TotalX        int    `json:"total_x" db:"total_x"`
			MemberOrder   int    `json:"member_order" db:"member_order"`
		}

		type TeamWithMembers struct {
			models.Team
			Members []MemberInfo `json:"members"`
		}

		var result []TeamWithMembers
		for _, team := range teams {
			var members []MemberInfo
			db.Select(&members, `
				SELECT 
					tm.uuid,
					ep.uuid as participant_id,
					COALESCE(a.full_name, '') as full_name,
					COALESCE(a.gender, '') as gender,
					COALESCE(cl.name, 'Independen') as club_name,
					COALESCE(SUM(qes.total_score_end), 0) as total_score,
					COALESCE(SUM(qes.x_count_end), 0) as total_x,
					tm.member_order
				FROM team_members tm
				JOIN tournament_participants ep ON tm.participant_id = ep.uuid
				LEFT JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN clubs cl ON a.club_id = cl.uuid
				LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
				WHERE tm.team_id = ?
				GROUP BY tm.uuid, ep.uuid, a.full_name, a.gender, cl.name, tm.member_order
				ORDER BY tm.member_order ASC
			`, team.UUID)

			if members == nil {
				members = []MemberInfo{}
			}

			result = append(result, TeamWithMembers{
				Team:    team,
				Members: members,
			})
		}

		if result == nil {
			result = []TeamWithMembers{}
		}

		c.JSON(http.StatusOK, gin.H{
			"teams": result,
			"total": len(result),
		})
	}
}

// GetMyTeams returns all teams managed by the authenticated user's organizer, club, or archer
func GetMyTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		userType, _ := c.Get("user_type")

		var teams []struct {
			models.Team
			EventName    string `json:"event_name" db:"event_name"`
			CategoryName string `json:"category_name" db:"category_name"`
			MemberCount  int    `json:"member_count" db:"member_count"`
			IsCaptain    bool   `json:"is_captain" db:"is_captain"`
		}

		if userType == "organizer" {
			query := `
				SELECT t.*, e.name as event_name, COALESCE(c.category_name_custom, '') as category_name, 
				       COUNT(DISTINCT tm.uuid) as member_count, 0 as is_captain
				FROM teams t
				JOIN tournaments e ON t.tournament_id = e.uuid
				LEFT JOIN tournament_categories c ON (t.category_id = c.uuid OR t.event_id = c.uuid)
				LEFT JOIN team_members tm ON t.uuid = tm.team_id
				WHERE e.organizer_id = ?
				GROUP BY t.uuid, t.tournament_id, t.event_id, t.category_id, t.team_name, t.team_rank, t.total_score, t.total_x_count, t.status, t.created_at, t.updated_at, e.name, c.category_name_custom
				ORDER BY t.created_at DESC
			`
			_ = db.Select(&teams, query, userID)
		} else if userType == "club" {
			query := `
				SELECT t.*, e.name as event_name, COALESCE(c.category_name_custom, '') as category_name, 
				       COUNT(DISTINCT tm.uuid) as member_count, 0 as is_captain
				FROM teams t
				JOIN tournaments e ON t.tournament_id = e.uuid
				LEFT JOIN tournament_categories c ON (t.category_id = c.uuid OR t.event_id = c.uuid)
				LEFT JOIN team_members tm ON t.uuid = tm.team_id
				WHERE e.organizer_id = ?
				GROUP BY t.uuid, t.tournament_id, t.event_id, t.category_id, t.team_name, t.team_rank, t.total_score, t.total_x_count, t.status, t.created_at, t.updated_at, e.name, c.category_name_custom
				ORDER BY t.created_at DESC
			`
			_ = db.Select(&teams, query, userID)
		} else {
			// Archer user type
			query := `
				SELECT t.*, e.name as event_name, COALESCE(c.category_name_custom, '') as category_name, 
				       COUNT(DISTINCT tm.uuid) as member_count,
				       CASE WHEN my_tm.member_order = 1 THEN 1 ELSE 0 END as is_captain
				FROM teams t
				JOIN tournaments e ON t.tournament_id = e.uuid
				LEFT JOIN tournament_categories c ON (t.category_id = c.uuid OR t.event_id = c.uuid)
				JOIN team_members my_tm ON t.uuid = my_tm.team_id
				JOIN tournament_participants tp ON my_tm.participant_id = tp.uuid
				LEFT JOIN team_members tm ON t.uuid = tm.team_id
				WHERE tp.archer_id = ?
				GROUP BY t.uuid, t.tournament_id, t.event_id, t.category_id, t.team_name, t.team_rank, t.total_score, t.total_x_count, t.status, t.created_at, t.updated_at, e.name, c.category_name_custom, my_tm.member_order
				ORDER BY t.created_at DESC
			`
			_ = db.Select(&teams, query, userID)
		}

		if teams == nil {
			teams = []struct {
				models.Team
				EventName    string `json:"event_name" db:"event_name"`
				CategoryName string `json:"category_name" db:"category_name"`
				MemberCount  int    `json:"member_count" db:"member_count"`
				IsCaptain    bool   `json:"is_captain" db:"is_captain"`
			}{}
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  teams,
			"total": len(teams),
		})
	}
}

// GetEligiblePartners returns archers eligible to join a team in a category
func GetEligiblePartners(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("slug")
		if eventID == "" {
			eventID = c.Param("id")
		}
		categoryID := c.Param("categoryId")
		search := strings.TrimSpace(c.Query("search"))
		genderFilter := strings.TrimSpace(strings.ToLower(c.Query("gender")))
		clubID := strings.TrimSpace(c.Query("club_id"))
		excludeArcherID := strings.TrimSpace(c.Query("exclude_archer_id"))

		// 1. Resolve Tournament UUID & entry fee
		var tour struct {
			UUID     string  `db:"uuid"`
			EntryFee float64 `db:"entry_fee"`
		}
		err := db.Get(&tour, `SELECT uuid, entry_fee FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Turnamen tidak ditemukan"})
			return
		}

		// 2. Resolve Category Info
		var catInfo struct {
			UUID               string  `db:"uuid"`
			DivisionUUID       string  `db:"division_uuid"`
			CategoryUUID       string  `db:"category_uuid"`
			TournamentTypeUUID string  `db:"tournament_type_uuid"`
			GenderDivisionUUID *string `db:"gender_division_uuid"`
			TypeCode           string  `db:"type_code"`
			GenderCode         string  `db:"gender_code"`
		}
		err = db.Get(&catInfo, `
			SELECT 
				tc.uuid,
				COALESCE(tc.division_uuid, '') as division_uuid,
				COALESCE(tc.category_uuid, '') as category_uuid,
				COALESCE(tc.tournament_type_uuid, '') as tournament_type_uuid,
				tc.gender_division_uuid,
				COALESCE(rtt.code, 'team') as type_code,
				COALESCE(rgd.code, '') as gender_code
			FROM tournament_categories tc
			LEFT JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
			LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
			WHERE tc.uuid = ? AND tc.tournament_id = ?
		`, categoryID, tour.UUID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
			return
		}

		// Required gender
		requiredGender := genderFilter
		if requiredGender == "" && catInfo.GenderCode != "" && catInfo.GenderCode != "mixed" {
			requiredGender = catInfo.GenderCode
		}

		// 3. Find matching individual category ID for this division & age group
		var indivCatID string
		_ = db.Get(&indivCatID, `
			SELECT tc.uuid
			FROM tournament_categories tc
			JOIN ref_tournament_types rtt ON tc.tournament_type_uuid = rtt.uuid
			WHERE tc.tournament_id = ? 
			  AND tc.division_uuid = ? 
			  AND tc.category_uuid = ? 
			  AND rtt.code = 'individual'
			  AND (? = '' OR tc.gender_division_uuid = (SELECT uuid FROM ref_gender_divisions WHERE code = ? LIMIT 1))
			LIMIT 1
		`, tour.UUID, catInfo.DivisionUUID, catInfo.CategoryUUID, requiredGender, requiredGender)

		// 4. Query Registered Participants (already in tournament)
		queryRegistered := `
			SELECT 
				a.uuid as archer_id,
				tp.uuid as participant_id,
				a.full_name,
				COALESCE(a.gender, 'male') as gender,
				COALESCE(cl.name, 'Independen') as club_name,
				a.avatar_url,
				tp.payment_status,
				CASE WHEN tp.payment_status IN ('paid', 'lunas') THEN 1 ELSE 0 END as is_already_registered_individual,
				CASE WHEN tp.payment_status IN ('paid', 'lunas') THEN 0.00 ELSE ? END as individual_fee
			FROM tournament_participants tp
			JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			WHERE tp.tournament_id = ?
			  AND tp.payment_status != 'cancelled'
			  AND (? = '' OR tp.category_id = ? OR tp.category_id = ?)
			  AND (? = '' OR a.gender = ?)
			  AND (? = '' OR a.uuid != ?)
			  AND (? = '' OR a.full_name LIKE ? OR a.username LIKE ? OR a.id LIKE ?)
			  AND tp.uuid NOT IN (
				  SELECT tm.participant_id 
				  FROM team_members tm 
				  JOIN teams t ON tm.team_id = t.uuid 
				  WHERE (t.category_id = ? OR t.event_id = ?) AND t.status != 'eliminated'
			  )
			GROUP BY a.uuid, tp.uuid, a.full_name, a.gender, cl.name, a.avatar_url, tp.payment_status
			ORDER BY is_already_registered_individual DESC, a.full_name ASC
			LIMIT 25
		`
		searchPattern := "%" + search + "%"
		var registeredPartners []models.EligiblePartner
		_ = db.Select(&registeredPartners, queryRegistered,
			tour.EntryFee,
			tour.UUID,
			indivCatID, indivCatID, categoryID,
			requiredGender, requiredGender,
			excludeArcherID, excludeArcherID,
			search, searchPattern, searchPattern, searchPattern,
			categoryID, categoryID,
		)

		// 5. Query non-registered archers if searching or club provided
		var nonRegisteredPartners []models.EligiblePartner
		if search != "" || clubID != "" {
			queryArchers := `
				SELECT 
					a.uuid as archer_id,
					NULL as participant_id,
					a.full_name,
					COALESCE(a.gender, 'male') as gender,
					COALESCE(cl.name, 'Independen') as club_name,
					a.avatar_url,
					'unregistered' as payment_status,
					0 as is_already_registered_individual,
					? as individual_fee
				FROM archers a
				LEFT JOIN clubs cl ON a.club_id = cl.uuid
				WHERE (? = '' OR a.gender = ?)
				  AND (? = '' OR a.uuid != ?)
				  AND (? = '' OR a.club_id = ?)
				  AND (? = '' OR a.full_name LIKE ? OR a.username LIKE ? OR a.id LIKE ?)
				  AND a.uuid NOT IN (
					  SELECT DISTINCT archer_id FROM tournament_participants 
					  WHERE tournament_id = ? AND payment_status != 'cancelled'
				  )
				ORDER BY a.full_name ASC
				LIMIT 25
			`
			_ = db.Select(&nonRegisteredPartners, queryArchers,
				tour.EntryFee,
				requiredGender, requiredGender,
				excludeArcherID, excludeArcherID,
				clubID, clubID,
				search, searchPattern, searchPattern, searchPattern,
				tour.UUID,
			)
		}

		// Deduplicate and combine
		seenArchers := map[string]bool{}
		finalResults := []models.EligiblePartner{}

		for _, p := range registeredPartners {
			if !seenArchers[p.ArcherID] {
				seenArchers[p.ArcherID] = true
				finalResults = append(finalResults, p)
			}
		}
		for _, p := range nonRegisteredPartners {
			if !seenArchers[p.ArcherID] {
				seenArchers[p.ArcherID] = true
				finalResults = append(finalResults, p)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"data":   finalResults,
			"meta": gin.H{
				"tournament_id": tour.UUID,
				"category_id":   catInfo.UUID,
				"team_type":     catInfo.TypeCode,
				"gender":        requiredGender,
				"total":         len(finalResults),
			},
		})
	}
}


// GetTeam returns a single team with members
func GetTeam(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamID := c.Param("teamId")

		type MemberDetail struct {
			UUID          string  `json:"id" db:"uuid"`
			TeamID        string  `json:"team_id" db:"team_id"`
			ParticipantID string  `json:"participant_id" db:"participant_id"`
			MemberOrder   int     `json:"member_order" db:"member_order"`
			FullName      string  `json:"full_name" db:"full_name"`
			ArcherName    string  `json:"archer_name" db:"archer_name"`
			Gender        string  `json:"gender" db:"gender"`
			ClubName      string  `json:"club_name" db:"club_name"`
			AthleteCode   string  `json:"athlete_code" db:"athlete_code"`
			BackNumber    *string `json:"back_number" db:"back_number"`
			Score         int     `json:"score" db:"score"`
			TotalScore    int     `json:"total_score" db:"total_score"`
			TotalX        int     `json:"total_x" db:"total_x"`
		}

		type TeamDetail struct {
			models.Team
			Name         string         `json:"name" db:"name"`
			CategoryName string         `json:"category_name" db:"category_name"`
			ClubName     string         `json:"club_name" db:"club_name"`
			Rank         *int           `json:"rank" db:"rank"`
			XCount       int            `json:"x_count" db:"x_count"`
			Members      []MemberDetail `json:"members"`
		}

		var team TeamDetail
		err := db.Get(&team, `
			SELECT 
				t.*,
				t.team_name as name,
				t.team_rank as rank,
				t.total_x_count as x_count,
				COALESCE(tc.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' ', COALESCE(rag.name, ''), ' ', COALESCE(rgd.name, ''))) as category_name,
				COALESCE(
					(SELECT cl.name FROM team_members tm2 
					 JOIN tournament_participants tp2 ON tm2.participant_id = tp2.uuid 
					 JOIN archers a2 ON tp2.archer_id = a2.uuid 
					 JOIN clubs cl ON a2.club_id = cl.uuid 
					 WHERE tm2.team_id = t.uuid LIMIT 1), 'Independen'
				) as club_name
			FROM teams t
			LEFT JOIN tournament_categories tc ON (t.category_id = tc.uuid OR t.event_id = tc.uuid)
			LEFT JOIN ref_bow_types rbt ON tc.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON tc.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON tc.gender_division_uuid = rgd.uuid
			WHERE t.uuid = ?
		`, teamID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tim tidak ditemukan"})
			return
		}

		var members []MemberDetail
		err = db.Select(&members, `
			SELECT 
				tm.uuid, tm.team_id, tm.participant_id, tm.member_order,
				COALESCE(a.full_name, '') as full_name,
				COALESCE(a.full_name, '') as archer_name,
				COALESCE(a.gender, '') as gender,
				COALESCE(cl.name, 'Independen') as club_name,
				COALESCE(a.id, tm.participant_id) as athlete_code,
				tp.target_name as back_number,
				COALESCE(SUM(qes.total_score_end), 0) as score,
				COALESCE(SUM(qes.total_score_end), 0) as total_score,
				COALESCE(SUM(qes.x_count_end), 0) as total_x
			FROM team_members tm
			JOIN tournament_participants tp ON tm.participant_id = tp.uuid
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = tp.uuid
			WHERE tm.team_id = ?
			GROUP BY tm.uuid, tm.team_id, tm.participant_id, tm.member_order, a.full_name, a.gender, cl.name, a.id, tp.target_name
			ORDER BY tm.member_order ASC
		`, teamID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data anggota tim"})
			return
		}

		if members == nil {
			members = []MemberDetail{}
		}
		team.Members = members

		c.JSON(http.StatusOK, gin.H{
			"team":    team,
			"members": members,
		})
	}
}

// SubmitTeamScore submits a score for a team end
func SubmitTeamScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			TeamID        string `json:"team_id" binding:"required"`
			EventID       string `json:"event_id" binding:"required"`
			Session       int    `json:"session" binding:"required"`
			DistanceOrder int    `json:"distance_order" binding:"required"`
			EndNumber     int    `json:"end_number" binding:"required"`
			MemberScores  []struct {
				ParticipantID string `json:"participant_id"`
				Arrow1        *int   `json:"arrow_1"`
				Arrow2        *int   `json:"arrow_2"`
				Arrow3        *int   `json:"arrow_3"`
				Arrow4        *int   `json:"arrow_4"`
				Arrow5        *int   `json:"arrow_5"`
				Arrow6        *int   `json:"arrow_6"`
			} `json:"member_scores"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Calculate team total
		endTotal := 0
		xCount := 0
		for _, ms := range req.MemberScores {
			arrows := []*int{ms.Arrow1, ms.Arrow2, ms.Arrow3, ms.Arrow4, ms.Arrow5, ms.Arrow6}
			for _, arrow := range arrows {
				if arrow != nil {
					val := *arrow
					if val == 11 { // X
						xCount++
						endTotal += 10
					} else {
						endTotal += val
					}
				}
			}
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// Get previous running total
		var prevRunningTotal int
		_ = tx.Get(&prevRunningTotal, `
			SELECT COALESCE(MAX(running_total), 0) 
			FROM team_scores 
			WHERE team_id = ? AND session = ? AND distance_order = ? AND end_number < ?
		`, req.TeamID, req.Session, req.DistanceOrder, req.EndNumber)

		runningTotal := prevRunningTotal + endTotal

		// Store member scores as JSON
		memberScoresJSON, _ := json.Marshal(req.MemberScores)

		scoreID := uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO team_scores 
			(id, team_id, tournament_id, session, distance_order, end_number, member_scores, end_total, x_count, running_total, entered_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
			member_scores = VALUES(member_scores), end_total = VALUES(end_total), x_count = VALUES(x_count), 
			running_total = VALUES(running_total), entered_by = VALUES(entered_by)
		`, scoreID, req.TeamID, req.EventID, req.Session, req.DistanceOrder, req.EndNumber,
			string(memberScoresJSON), endTotal, xCount, runningTotal, userID.(string))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim skor tim"})
			return
		}

		// Update team total
		_, err = tx.Exec(`
			UPDATE teams SET 
				total_score = (SELECT COALESCE(SUM(end_total), 0) FROM team_scores WHERE team_id = ?),
				total_x_count = (SELECT COALESCE(SUM(x_count), 0) FROM team_scores WHERE team_id = ?)
			WHERE id = ? OR uuid = ?
		`, req.TeamID, req.TeamID, req.TeamID, req.TeamID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui total skor tim"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan skor"})
			return
		}

		// Broadcast update
		// BroadcastEventUpdate(req.EventID, gin.H{
		// 	"type": "team_score_update",
		// 	"data": gin.H{"team_id": req.TeamID, "end_total": endTotal, "running_total": runningTotal},
		// })

		c.JSON(http.StatusCreated, gin.H{
			"id":            scoreID,
			"end_total":     endTotal,
			"running_total": runningTotal,
			"message":       "Skor tim berhasil dikirim",
		})
	}
}

// GetTeamRankings returns team qualification rankings from the teams table
func GetTeamRankings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")
		categoryID := c.Query("category_id")

		query := `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY t.total_score DESC, t.total_x_count DESC) as rank,
				t.uuid as team_id,
				t.team_name,
				t.total_score,
				t.total_x_count
			FROM teams t
			WHERE t.tournament_id = ?
		`
		args := []interface{}{eventID}

		if categoryID != "" {
			query += " AND t.event_id = ?" // Note: event_id column in teams table stores category UUID
			args = append(args, categoryID)
		}

		query += " ORDER BY t.total_score DESC, t.total_x_count DESC"

		var rankings []models.TeamRanking
		err := db.Select(&rankings, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data peringkat tim", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"rankings": rankings,
			"total":    len(rankings),
		})
	}
}

// GetTeamQualificationRankings calculates rankings by taking top 3 archers from each club in a category
func GetTeamQualificationRankings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID := c.Query("category_id")
		if categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_id wajib diisi"})
			return
		}

		// Get team size and category info
		var catInfo struct {
			TournamentID     string  `db:"tournament_id"`
			DivisionID       string  `db:"division_uuid"`
			AgeGroupID       string  `db:"category_uuid"`
			GenderDivisionID *string `db:"gender_division_uuid"`
			TeamSize         int     `db:"team_size"`
		}
		err := db.Get(&catInfo, `
			SELECT 
				ec.tournament_id,
				ec.division_uuid,
				ec.category_uuid,
				ec.gender_division_uuid,
				CASE 
					WHEN ret.code = 'mixed_team' THEN 2 
					WHEN ret.code = 'team' THEN 3 
					ELSE 1 
				END as team_size
			FROM tournament_categories ec
			JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
			WHERE ec.uuid = ?`, categoryID)
		if err != nil {
			catInfo.TeamSize = 3 // Fallback
		}

		teamSize := catInfo.TeamSize
		if teamSize <= 0 {
			teamSize = 3
		}

		participantCatIDs := []string{categoryID}
		var indivCatID string
		if catInfo.GenderDivisionID != nil && catInfo.TournamentID != "" {
			if err2 := db.Get(&indivCatID, `
				SELECT ec.uuid
				FROM tournament_categories ec
				JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
				WHERE ec.tournament_id = ? AND ec.division_uuid = ? AND ec.category_uuid = ?
				  AND ec.gender_division_uuid = ? AND ret.code = 'individual'
			`, catInfo.TournamentID, catInfo.DivisionID, catInfo.AgeGroupID, *catInfo.GenderDivisionID); err2 == nil && indivCatID != "" {
				participantCatIDs = append(participantCatIDs, indivCatID)
			}
		}

		type TeamRankingEntry struct {
			Rank           int    `json:"rank"`
			ClubID         string `json:"club_id" db:"club_id"`
			ClubName       string `json:"club_name" db:"club_name"`
			TotalScore     int    `json:"total_score" db:"total_score"`
			Total10Count   int    `json:"total_10" db:"total_10"`
			TotalXCount    int    `json:"total_x" db:"total_x"`
			MemberCount    int    `json:"member_count" db:"member_count"`
			MemberNames    string `json:"member_names" db:"member_names"`
			ParticipantIDs string `json:"participant_ids" db:"participant_ids"`
		}

		// SQL for Multi-Team and Partial Team support
		query := `
			SELECT 
				ROW_NUMBER() OVER(ORDER BY SUM(individual_score) DESC, SUM(individual_10) DESC, SUM(individual_x) DESC) as rank,
				club_id,
				club_name,
				SUM(individual_score) as total_score,
				SUM(individual_10) as total_10,
				SUM(individual_x) as total_x,
				COUNT(*) as member_count,
				GROUP_CONCAT(archer_name ORDER BY individual_score DESC, individual_10 DESC, individual_x DESC SEPARATOR ', ') as member_names,
				GROUP_CONCAT(participant_id ORDER BY individual_score DESC, individual_10 DESC, individual_x DESC SEPARATOR ',') as participant_ids
			FROM (
				SELECT 
					ep.uuid as participant_id,
					a.uuid as archer_id,
					a.full_name as archer_name,
					cl.uuid as club_id,
					COALESCE(cl.name, 'Independen') as club_name,
					COALESCE(SUM(s.total_score_end), 0) as individual_score,
					COALESCE(SUM(s.ten_count_end), 0) as individual_10,
					COALESCE(SUM(s.x_count_end), 0) as individual_x,
					ROW_NUMBER() OVER(PARTITION BY a.club_id ORDER BY SUM(s.total_score_end) DESC, SUM(s.ten_count_end) DESC, SUM(s.x_count_end) DESC) as club_rank
				FROM tournament_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN clubs cl ON a.club_id = cl.uuid
				LEFT JOIN qualification_end_scores s ON s.participant_uuid = ep.uuid
				WHERE ep.category_id IN (?)
				GROUP BY ep.uuid, a.uuid, a.full_name, cl.uuid, cl.name
			) ranked
			WHERE club_id IS NOT NULL
			GROUP BY club_id, club_name, CEIL(club_rank / ?)
			HAVING member_count >= 2
			ORDER BY total_score DESC, total_10 DESC, total_x DESC
		`

		fullQuery, args, _ := sqlx.In(query, participantCatIDs, teamSize)
		fullQuery = db.Rebind(fullQuery)

		var rankings []TeamRankingEntry
		err = db.Select(&rankings, fullQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung peringkat tim", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"rankings": rankings,
			"total":    len(rankings),
		})
	}
}

// GetMixedTeamQualificationRankings calculates rankings by taking top 1 male and top 1 female from each club
func GetMixedTeamQualificationRankings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")
		divisionID := c.Query("division_id")  // standard, recurve, etc.
		ageGroupID := c.Query("age_group_id") // u15, senior, u18, etc.

		if divisionID == "" || ageGroupID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "division_id dan age_group_id wajib diisi"})
			return
		}

		// 1. Get the Male and Female categories for this division + age group
		var catIDs []struct {
			UUID   string `db:"uuid"`
			Gender string `db:"gender_code"`
		}
		err := db.Select(&catIDs, `
			SELECT ec.uuid, rgd.code as gender_code
			FROM tournament_categories ec
			JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE ec.tournament_id = ? AND ec.division_uuid = ? AND ec.category_uuid = ?
		`, eventID, divisionID, ageGroupID)

		if err != nil || len(catIDs) < 2 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori putra dan putri tidak ditemukan untuk divisi/kelompok umur ini"})
			return
		}

		maleCatID := ""
		femaleCatID := ""
		for _, cat := range catIDs {
			if cat.Gender == "men" || cat.Gender == "male" {
				maleCatID = cat.UUID
			} else if cat.Gender == "women" || cat.Gender == "female" {
				femaleCatID = cat.UUID
			}
		}

		if maleCatID == "" || femaleCatID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal mengidentifikasi kategori putra dan putri"})
			return
		}

		type MixedTeamEntry struct {
			Rank        int    `json:"rank"`
			ClubID      string `json:"club_id" db:"club_id"`
			ClubName    string `json:"club_name" db:"club_name"`
			MaleName    string `json:"male_name" db:"male_name"`
			FemaleName  string `json:"female_name" db:"female_name"`
			MaleID      string `json:"male_id" db:"male_id"`
			FemaleID    string `json:"female_id" db:"female_id"`
			MaleScore   int    `json:"male_score" db:"male_score"`
			FemaleScore int    `json:"female_score" db:"female_score"`
			TotalScore  int    `json:"total_score" db:"total_score"`
			TotalX      int    `json:"total_x" db:"total_x"`
		}

		// SQL to get all potential M and F pairs per club
		query := `
			SELECT 
				ROW_NUMBER() OVER(ORDER BY (male_score + female_score) DESC, (male_x + female_x) DESC) as rank,
				club_id,
				club_name,
				male_name,
				female_name,
				male_id,
				female_id,
				male_score,
				female_score,
				(male_score + female_score) as total_score,
				(male_x + female_x) as total_x
			FROM (
				SELECT 
					cl.uuid as club_id,
					cl.name as club_name,
					ep.rank_in_club,
					MAX(CASE WHEN ep.category_id = ? THEN archer_name ELSE '' END) as male_name,
					MAX(CASE WHEN ep.category_id = ? THEN archer_name ELSE '' END) as female_name,
					MAX(CASE WHEN ep.category_id = ? THEN archer_id ELSE '' END) as male_id,
					MAX(CASE WHEN ep.category_id = ? THEN archer_id ELSE '' END) as female_id,
					MAX(CASE WHEN ep.category_id = ? THEN individual_score ELSE 0 END) as male_score,
					MAX(CASE WHEN ep.category_id = ? THEN individual_score ELSE 0 END) as female_score,
					MAX(CASE WHEN ep.category_id = ? THEN individual_x ELSE 0 END) as male_x,
					MAX(CASE WHEN ep.category_id = ? THEN individual_x ELSE 0 END) as female_x
				FROM (
					SELECT 
						a.archer_id,
						a.archer_name,
						a.club_id,
						a.category_id,
						individual_score,
						individual_x,
						ROW_NUMBER() OVER(PARTITION BY a.club_id, a.category_id ORDER BY individual_score DESC, individual_x DESC) as rank_in_club
					FROM (
						SELECT 
							ep.archer_id,
							a.full_name as archer_name,
							a.club_id,
							ep.category_id,
							COALESCE(SUM(s.total_score_end), 0) as individual_score,
							COALESCE(SUM(s.x_count_end), 0) as individual_x
						FROM tournament_participants ep
						JOIN archers a ON ep.archer_id = a.uuid
						LEFT JOIN qualification_end_scores s ON s.participant_uuid = ep.uuid
						WHERE ep.category_id IN (?, ?)
						GROUP BY ep.uuid, a.club_id, ep.category_id
					) a
				) ep
				JOIN clubs cl ON ep.club_id = cl.uuid
				GROUP BY cl.uuid, cl.name, ep.rank_in_club
				HAVING male_score > 0 AND female_score > 0
			) mixed
			ORDER BY total_score DESC, total_x DESC
		`

		var rankings []MixedTeamEntry
		err = db.Select(&rankings, query, maleCatID, femaleCatID, maleCatID, femaleCatID, maleCatID, femaleCatID, maleCatID, femaleCatID, maleCatID, femaleCatID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung peringkat mixed team", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"rankings": rankings,
			"total":    len(rankings),
		})
	}
}

// AutoCreateTeams creates team records from calculated qualification rankings
func AutoCreateTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("eventId") // The main event UUID
		userID, _ := c.Get("user_id")

		var req struct {
			CategoryID string `json:"category_id" binding:"required"`
			Teams      []struct {
				ClubID         string   `json:"club_id"`
				TeamName       string   `json:"team_name"`
				TotalScore     int      `json:"total_score"`
				TotalX         int      `json:"total_x"`
				ParticipantIDs []string `json:"participant_ids"`
			} `json:"teams" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, tournamentID, tournamentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Check if elimination bracket exists
		var bracketCount int
		_ = db.Get(&bracketCount, `
			SELECT COUNT(*) FROM elimination_brackets 
			WHERE category_uuid = ? AND tournament_uuid = ? AND status != 'draft'
		`, req.CategoryID, eventUUID)
		if bracketCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Bagan eliminasi untuk kategori ini sudah dibuat. Hapus atau reset bagan eliminasi terlebih dahulu untuk mengubah susunan tim.",
			})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// 1. Delete existing teams for this category to allow "Regeneration"
		var teamUUIDs []string
		err = tx.Select(&teamUUIDs, "SELECT uuid FROM teams WHERE tournament_id = ? AND (event_id = ? OR category_id = ?)", eventUUID, req.CategoryID, req.CategoryID)
		if err == nil && len(teamUUIDs) > 0 {
			query, args, _ := sqlx.In("DELETE FROM team_members WHERE team_id IN (?)", teamUUIDs)
			query = db.Rebind(query)
			_, _ = tx.Exec(query, args...)

			_, _ = tx.Exec("DELETE FROM teams WHERE tournament_id = ? AND (event_id = ? OR category_id = ?)", eventUUID, req.CategoryID, req.CategoryID)
		}

		// 2. Insert new teams
		for i, teamReq := range req.Teams {
			teamUUID := uuid.New().String()

			_, err = tx.Exec(`
				INSERT INTO teams (uuid, tournament_id, event_id, category_id, team_name, team_rank, total_score, total_x_count, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'active')
			`, teamUUID, eventUUID, req.CategoryID, req.CategoryID, teamReq.TeamName, i+1, teamReq.TotalScore, teamReq.TotalX)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat data tim", "details": err.Error(), "team": teamReq.TeamName})
				return
			}

			for order, pID := range teamReq.ParticipantIDs {
				if pID == "" {
					continue
				}
				memberUUID := uuid.New().String()
				_, err = tx.Exec(`
					INSERT INTO team_members (uuid, team_id, participant_id, member_order)
					VALUES (?, ?, ?, ?)
				`, memberUUID, teamUUID, pID, order+1)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan anggota tim", "details": err.Error()})
					return
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data tim"})
			return
		}

		utils.LogActivity(db, userID.(string), eventUUID, "teams_regenerated", "event", eventUUID,
			fmt.Sprintf("Regenerated %d teams for category %s", len(req.Teams), req.CategoryID), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Tim berhasil disinkronisasi", "count": len(req.Teams)})
	}
}

// SyncTeams calculates rankings and creates team records in one step on the server
func SyncTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")
		userID, _ := c.Get("user_id")

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var req struct {
			CategoryID string `json:"category_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 1. Check if elimination bracket exists
		var bracketCount int
		_ = db.Get(&bracketCount, `
			SELECT COUNT(*) FROM elimination_brackets 
			WHERE category_uuid = ? AND tournament_uuid = ? AND status != 'draft'
		`, req.CategoryID, eventUUID)
		if bracketCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Bagan eliminasi untuk kategori ini sudah dibuat. Hapus atau reset bagan eliminasi terlebih dahulu sebelum melakukan sinkronisasi ulang tim.",
			})
			return
		}

		// 2. Check category type (Standard vs Mixed)
		var catInfo struct {
			TypeID           string  `db:"tournament_type_uuid"`
			TypeCode         string  `db:"type_code"`
			DivisionID       string  `db:"division_uuid"`
			AgeGroupID       string  `db:"category_uuid"`
			GenderDivisionID *string `db:"gender_division_uuid"`
			DivisionName     string  `db:"division_name"`
			TeamSize         int     `db:"team_size"`
		}
		err = db.Get(&catInfo, `
			SELECT 
				ec.tournament_type_uuid, 
				ret.code as type_code, 
				ec.division_uuid, 
				ec.category_uuid, 
				ec.gender_division_uuid,
				rbt.name as division_name, 
				CASE 
					WHEN ret.code = 'mixed_team' THEN 2 
					WHEN ret.code = 'team' THEN 3 
					ELSE 1 
				END as team_size
			FROM tournament_categories ec
			JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
			JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			WHERE ec.uuid = ?
		`, req.CategoryID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Kategori tidak ditemukan",
				"details": err.Error(),
			})
			return
		}

		isMixed := catInfo.TypeCode == "mixed_team" || strings.Contains(strings.ToLower(catInfo.TypeCode), "mixed")

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start database transaction"})
			return
		}
		defer tx.Rollback()

		// 3. Clear old teams
		var teamUUIDs []string
		err = tx.Select(&teamUUIDs, "SELECT uuid FROM teams WHERE tournament_id = ? AND (event_id = ? OR category_id = ?)", eventUUID, req.CategoryID, req.CategoryID)
		if err == nil && len(teamUUIDs) > 0 {
			query, args, _ := sqlx.In("DELETE FROM team_members WHERE team_id IN (?)", teamUUIDs)
			query = db.Rebind(query)
			_, _ = tx.Exec(query, args...)
			_, _ = tx.Exec("DELETE FROM teams WHERE tournament_id = ? AND (event_id = ? OR category_id = ?)", eventUUID, req.CategoryID, req.CategoryID)
		}

		syncCount := 0
		syncDetails := gin.H{
			"category_id": req.CategoryID,
			"team_size":   catInfo.TeamSize,
			"type_code":   catInfo.TypeCode,
			"is_mixed":    isMixed,
		}

		if isMixed {
			// Find male/female individual categories
			var catIDs []struct {
				UUID   string `db:"uuid"`
				Gender string `db:"gender_code"`
			}
			err = tx.Select(&catIDs, `
				SELECT ec.uuid, rgd.code as gender_code
				FROM tournament_categories ec
				JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
				JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
				WHERE ec.tournament_id = ? AND ec.division_uuid = ? AND ec.category_uuid = ?
				  AND ret.code = 'individual'
			`, eventUUID, catInfo.DivisionID, catInfo.AgeGroupID)

			if err != nil || len(catIDs) < 2 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Kategori individu putra dan putri tidak lengkap untuk divisi/kelompok umur ini",
					"details": gin.H{
						"reason_code": "missing_individual_categories",
						"reason":      "Mixed team membutuhkan kategori Individu Putra dan Individu Putri dalam divisi dan kelompok umur yang sama.",
					},
				})
				return
			}

			maleCatID, femaleCatID := "", ""
			for _, cat := range catIDs {
				if cat.Gender == "men" || cat.Gender == "male" {
					maleCatID = cat.UUID
				}
				if cat.Gender == "women" || cat.Gender == "female" {
					femaleCatID = cat.UUID
				}
			}

			if maleCatID == "" || femaleCatID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal mengidentifikasi kategori putra dan putri"})
				return
			}

			// Count participants with scores
			var maleParticipants, femaleParticipants int
			_ = tx.Get(&maleParticipants, `SELECT COUNT(DISTINCT archer_id) FROM tournament_participants WHERE category_id = ? AND payment_status IN ('paid', 'lunas')`, maleCatID)
			_ = tx.Get(&femaleParticipants, `SELECT COUNT(DISTINCT archer_id) FROM tournament_participants WHERE category_id = ? AND payment_status IN ('paid', 'lunas')`, femaleCatID)

			// Calculate mixed rankings and insert
			var rankings []struct {
				ClubID      string `db:"club_id"`
				ClubName    string `db:"club_name"`
				MaleID      string `db:"male_id"`
				FemaleID    string `db:"female_id"`
				MaleScore   int    `db:"male_score"`
				FemaleScore int    `db:"female_score"`
				Male10      int    `db:"male_10"`
				Female10    int    `db:"female_10"`
				MaleX       int    `db:"male_x"`
				FemaleX     int    `db:"female_x"`
				TotalScore  int    `db:"total_score"`
				Total10     int    `db:"total_10"`
				TotalX      int    `db:"total_x"`
			}

			query := `
				SELECT 
					club_id, club_name, male_id, female_id, 
					male_score, female_score, (male_score + female_score) as total_score, 
					male_10, female_10, (male_10 + female_10) as total_10,
					male_x, female_x, (male_x + female_x) as total_x
				FROM (
					SELECT 
						cl.uuid as club_id, cl.name as club_name, ep_ranked.rank_in_club,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.participant_id ELSE '' END) as male_id,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.participant_id ELSE '' END) as female_id,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_score ELSE 0 END) as male_score,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_score ELSE 0 END) as female_score,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_10 ELSE 0 END) as male_10,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_10 ELSE 0 END) as female_10,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_x ELSE 0 END) as male_x,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_x ELSE 0 END) as female_x
					FROM (
						SELECT 
							ep.archer_id, ep.uuid as participant_id, a.club_id, ep.category_id, 
							COALESCE(SUM(s.total_score_end), 0) as individual_score, 
							COALESCE(SUM(s.ten_count_end), 0) as individual_10, 
							COALESCE(SUM(s.x_count_end), 0) as individual_x,
							ROW_NUMBER() OVER(PARTITION BY a.club_id, ep.category_id ORDER BY SUM(s.total_score_end) DESC, SUM(s.ten_count_end) DESC, SUM(s.x_count_end) DESC) as rank_in_club
						FROM tournament_participants ep
						JOIN archers a ON ep.archer_id = a.uuid
						LEFT JOIN qualification_end_scores s ON s.participant_uuid = ep.uuid
						WHERE ep.category_id IN (?, ?) AND ep.payment_status IN ('paid', 'lunas')
						GROUP BY ep.archer_id, ep.uuid, a.club_id, ep.category_id
					) ep_ranked
					JOIN clubs cl ON ep_ranked.club_id = cl.uuid
					GROUP BY cl.uuid, cl.name, ep_ranked.rank_in_club
					HAVING male_id != '' AND female_id != ''
				) mixed 
				ORDER BY total_score DESC, total_10 DESC, total_x DESC`

			err = tx.Select(&rankings, query, maleCatID, femaleCatID, maleCatID, femaleCatID, maleCatID, femaleCatID, maleCatID, femaleCatID, maleCatID, femaleCatID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate mixed team rankings", "details": err.Error()})
				return
			}

			syncDetails["male_participants"] = maleParticipants
			syncDetails["female_participants"] = femaleParticipants
			syncDetails["eligible_team_groups"] = len(rankings)

			// Count total teams per club to format names
			clubTotalTeams := make(map[string]int)
			for _, r := range rankings {
				clubTotalTeams[r.ClubID]++
			}
			clubCurrentIndex := make(map[string]int)
			letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

			for i, r := range rankings {
				clubCurrentIndex[r.ClubID]++
				suffix := ""
				if clubTotalTeams[r.ClubID] > 1 {
					idx := clubCurrentIndex[r.ClubID]
					if idx <= len(letters) {
						suffix = " " + string(letters[idx-1])
					} else {
						suffix = fmt.Sprintf(" %d", idx)
					}
				}

				teamUUID := uuid.New().String()
				teamName := "Mixed " + r.ClubName + suffix
				if _, err = tx.Exec(`
					INSERT INTO teams (uuid, tournament_id, event_id, category_id, team_name, team_rank, total_score, total_x_count, status) 
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'active')
				`, teamUUID, eventUUID, req.CategoryID, req.CategoryID, teamName, i+1, r.TotalScore, r.TotalX); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert mixed team data", "details": err.Error()})
					return
				}

				pIDs := []string{r.MaleID, r.FemaleID}
				for order, pID := range pIDs {
					if _, err = tx.Exec(`INSERT INTO team_members (uuid, team_id, participant_id, member_order) VALUES (?, ?, ?, ?)`,
						uuid.New().String(), teamUUID, pID, order+1); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert mixed team member", "details": err.Error()})
						return
					}
				}
				syncCount++
			}
		} else {
			// Standard Team calculation and insert
			var rankings []struct {
				ClubID         string `db:"club_id"`
				ClubName       string `db:"club_name"`
				TotalScore     int    `db:"total_score"`
				Total10        int    `db:"total_10"`
				TotalX         int    `db:"total_x"`
				ParticipantIDs string `db:"participant_ids"`
				MemberCount    int    `db:"member_count"`
			}
			teamSize := catInfo.TeamSize
			if teamSize <= 0 {
				teamSize = 3 // Standard default
			}

			// Participants register under the individual category, not the team category.
			// Resolve the matching individual category (same event/division/age/gender).
			participantCatIDs := []string{req.CategoryID}
			var indivCatID string
			if catInfo.GenderDivisionID != nil {
				if err2 := tx.Get(&indivCatID, `
					SELECT ec.uuid
					FROM tournament_categories ec
					JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
					WHERE ec.tournament_id = ? AND ec.division_uuid = ? AND ec.category_uuid = ?
					  AND ec.gender_division_uuid = ? AND ret.code = 'individual'
				`, eventUUID, catInfo.DivisionID, catInfo.AgeGroupID, *catInfo.GenderDivisionID); err2 == nil && indivCatID != "" {
					participantCatIDs = append(participantCatIDs, indivCatID)
				}
			}

			var totalParticipants int
			pQuery, pArgs, _ := sqlx.In(`SELECT COUNT(DISTINCT archer_id) FROM tournament_participants WHERE category_id IN (?) AND payment_status IN ('paid', 'lunas')`, participantCatIDs)
			pQuery = tx.Rebind(pQuery)
			_ = tx.Get(&totalParticipants, pQuery, pArgs...)

			var clubsWithParticipants int
			cQuery, cArgs, _ := sqlx.In(`
				SELECT COUNT(DISTINCT a.club_id)
				FROM tournament_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				WHERE ep.category_id IN (?) AND ep.payment_status IN ('paid', 'lunas') AND a.club_id IS NOT NULL AND a.club_id <> ''
			`, participantCatIDs)
			cQuery = tx.Rebind(cQuery)
			_ = tx.Get(&clubsWithParticipants, cQuery, cArgs...)

			query := `
				SELECT 
					club_id, club_name, 
					SUM(individual_score) as total_score, 
					SUM(individual_10) as total_10, 
					SUM(individual_x) as total_x, 
					GROUP_CONCAT(participant_id ORDER BY individual_score DESC, individual_10 DESC, individual_x DESC SEPARATOR ',') as participant_ids, 
					COUNT(*) as member_count
				FROM (
					SELECT 
						ep.uuid as participant_id, cl.uuid as club_id, COALESCE(cl.name, 'Independen') as club_name, 
						COALESCE(SUM(s.total_score_end), 0) as individual_score, 
						COALESCE(SUM(s.ten_count_end), 0) as individual_10, 
						COALESCE(SUM(s.x_count_end), 0) as individual_x,
						ROW_NUMBER() OVER(PARTITION BY a.club_id ORDER BY SUM(s.total_score_end) DESC, SUM(s.ten_count_end) DESC, SUM(s.x_count_end) DESC) as club_rank
					FROM tournament_participants ep
					JOIN archers a ON ep.archer_id = a.uuid
					LEFT JOIN clubs cl ON a.club_id = cl.uuid
					LEFT JOIN qualification_end_scores s ON s.participant_uuid = ep.uuid
					WHERE ep.category_id IN (?) AND ep.payment_status IN ('paid', 'lunas')
					GROUP BY ep.uuid, a.uuid, a.club_id, cl.uuid, cl.name
				) ranked
				WHERE club_id IS NOT NULL
				GROUP BY club_id, club_name, CEIL(club_rank / ?)
				HAVING member_count >= ?
				ORDER BY total_score DESC, total_10 DESC, total_x DESC`

			fullQuery, args, _ := sqlx.In(query, participantCatIDs, teamSize, teamSize)
			fullQuery = tx.Rebind(fullQuery)
			err = tx.Select(&rankings, fullQuery, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate team rankings", "details": err.Error()})
				return
			}

			syncDetails["total_participants"] = totalParticipants
			syncDetails["clubs_with_participants"] = clubsWithParticipants
			syncDetails["eligible_team_groups"] = len(rankings)

			// Count total teams per club to format names
			clubTotalTeams := make(map[string]int)
			for _, r := range rankings {
				clubTotalTeams[r.ClubID]++
			}
			clubCurrentIndex := make(map[string]int)
			letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

			for i, r := range rankings {
				clubCurrentIndex[r.ClubID]++
				suffix := ""
				if clubTotalTeams[r.ClubID] > 1 {
					idx := clubCurrentIndex[r.ClubID]
					if idx <= len(letters) {
						suffix = " " + string(letters[idx-1])
					} else {
						suffix = fmt.Sprintf(" %d", idx)
					}
				}

				teamUUID := uuid.New().String()
				teamName := r.ClubName + suffix
				if _, err = tx.Exec(`
					INSERT INTO teams (uuid, tournament_id, event_id, category_id, team_name, team_rank, total_score, total_x_count, status) 
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'active')
				`, teamUUID, eventUUID, req.CategoryID, req.CategoryID, teamName, i+1, r.TotalScore, r.TotalX); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert team data", "details": err.Error()})
					return
				}

				pIDs := strings.Split(r.ParticipantIDs, ",")
				for order, pID := range pIDs {
					if _, err = tx.Exec(`INSERT INTO team_members (uuid, team_id, participant_id, member_order) VALUES (?, ?, ?, ?)`,
						uuid.New().String(), teamUUID, pID, order+1); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert team member", "details": err.Error()})
						return
					}
				}
				syncCount++
			}
		}

		if syncCount == 0 {
			teamSize := catInfo.TeamSize
			if teamSize <= 0 {
				teamSize = 3
			}
			if isMixed {
				syncDetails["reason_code"] = "no_mixed_pairs"
				syncDetails["reason"] = "No club combinations found with 1 male and 1 female archer who both have qualification scores"
			} else {
				syncDetails["reason_code"] = "no_eligible_groups"
				syncDetails["reason"] = fmt.Sprintf("No club groups found with at least %d scored archers in this category", teamSize)
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		utils.LogActivity(db, userID.(string), eventUUID, "teams_synced_directly", "event", eventUUID,
			fmt.Sprintf("Directly synced %d teams for category %s", syncCount, req.CategoryID), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Teams synchronized successfully", "count": syncCount, "details": syncDetails})
	}
}

// UpdateTeam updates a team's details and members
func UpdateTeam(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamID := c.Param("teamId")
		userID, _ := c.Get("user_id")

		var req models.CreateTeamRequest // Reuse CreateTeamRequest for update
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 1. Get existing team info
		var currentTeam struct {
			TournamentID string `db:"tournament_id"`
			EventID      string `db:"event_id"`
		}
		err := db.Get(&currentTeam, "SELECT tournament_id, event_id FROM teams WHERE uuid = ?", teamID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tim tidak ditemukan"})
			return
		}

		// Check elimination bracket
		var bracketCount int
		_ = db.Get(&bracketCount, `
			SELECT COUNT(*) FROM elimination_brackets 
			WHERE category_uuid = ? AND tournament_uuid = ? AND status != 'draft'
		`, req.CategoryID, currentTeam.TournamentID)
		if bracketCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Bagan eliminasi untuk kategori ini sudah dibuat. Hapus/reset bagan eliminasi terlebih dahulu untuk mengubah susunan tim.",
			})
			return
		}

		// 2. Get category info and team_size
		var catInfo struct {
			TypeCode string `db:"type_code"`
			TeamSize int    `db:"team_size"`
		}
		err = db.Get(&catInfo, `
			SELECT 
				ret.code as type_code, 
				CASE 
					WHEN ret.code = 'mixed_team' THEN 2 
					WHEN ret.code = 'team' THEN 3 
					ELSE 1 
				END as team_size
			FROM tournament_categories ec
			JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
			WHERE ec.uuid = ?`, req.CategoryID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kategori tidak ditemukan"})
			return
		}

		if len(req.MemberIDs) != catInfo.TeamSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Jumlah anggota harus tepat %d orang untuk kategori ini", catInfo.TeamSize),
			})
			return
		}

		// Validate mixed team genders
		if catInfo.TypeCode == "mixed_team" || strings.Contains(strings.ToLower(catInfo.TypeCode), "mixed") {
			type MemberGender struct {
				Gender string `db:"gender"`
			}
			query, args, _ := sqlx.In(`
				SELECT COALESCE(a.gender, '') as gender
				FROM tournament_participants tp
				JOIN archers a ON tp.archer_id = a.uuid
				WHERE tp.uuid IN (?)
			`, req.MemberIDs)
			query = db.Rebind(query)
			var genders []MemberGender
			if err := db.Select(&genders, query, args...); err != nil || len(genders) != 2 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal memverifikasi data anggota tim"})
				return
			}
			hasMale := false
			hasFemale := false
			for _, g := range genders {
				if g.Gender == "male" || g.Gender == "men" {
					hasMale = true
				}
				if g.Gender == "female" || g.Gender == "women" {
					hasFemale = true
				}
			}
			if !hasMale || !hasFemale {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Mixed team wajib terdiri dari 1 pemanah putra dan 1 pemanah putri"})
				return
			}
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// Update team basic info
		_, err = tx.Exec(`
			UPDATE teams SET team_name = ?, event_id = ?, category_id = ?
			WHERE uuid = ?
		`, req.TeamName, req.CategoryID, req.CategoryID, teamID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui info tim"})
			return
		}

		// Delete existing members
		_, err = tx.Exec("DELETE FROM team_members WHERE team_id = ?", teamID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset anggota tim"})
			return
		}

		// Add new members
		for i, participantID := range req.MemberIDs {
			memberID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO team_members (uuid, team_id, participant_id, member_order)
				VALUES (?, ?, ?, ?)
			`, memberID, teamID, participantID, i+1)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan anggota tim"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan"})
			return
		}

		utils.LogActivity(db, userID.(string), "", "team_updated", "team", teamID,
			fmt.Sprintf("Memperbarui tim: %s", req.TeamName), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Tim berhasil diperbarui"})
	}
}

// DeleteTeam deletes a team and its members
func DeleteTeam(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamID := c.Param("teamId")
		userID, _ := c.Get("user_id")

		// 1. Check if team is part of an elimination bracket match
		var matchCount int
		_ = db.Get(&matchCount, `
			SELECT COUNT(*) FROM elimination_entries ee 
			WHERE ee.participant_uuid = ? AND ee.participant_type = 'team'
		`, teamID)
		if matchCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Tim tidak dapat dihapus karena sudah terdaftar dalam bagan eliminasi. Hapus atau reset bagan eliminasi terlebih dahulu.",
			})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// 2. Delete members
		_, err = tx.Exec("DELETE FROM team_members WHERE team_id = ?", teamID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus anggota tim"})
			return
		}

		// 3. Delete team
		_, err = tx.Exec("DELETE FROM teams WHERE uuid = ?", teamID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus tim"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan penghapusan"})
			return
		}

		utils.LogActivity(db, userID.(string), "", "team_deleted", "team", teamID,
			"Menghapus tim", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Tim berhasil dihapus"})
	}
}

// GetMyEventTeam returns the team info for the logged-in archer in a specific event
func GetMyEventTeam(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		if eventID == "" {
			eventID = c.Param("eventId")
		}
		
		var archerID string
		aID, _ := c.Get("archer_id")
		if aID != nil && aID != "" {
			archerID = fmt.Sprintf("%v", aID)
		}
		if archerID == "" {
			userID, _ := c.Get("user_id")
			if userID != nil && userID != "" {
				userEmailVal, _ := c.Get("email")
				userEmail := fmt.Sprintf("%v", userEmailVal)
				_ = db.Get(&archerID, `SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?) LIMIT 1`, userID, userID, userEmail)
			}
		}

		if archerID == "" {
			c.JSON(http.StatusOK, gin.H{"team": nil})
			return
		}

		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"team": nil})
			return
		}

		type TeamResult struct {
			UUID         string `db:"uuid" json:"uuid"`
			TeamName     string `db:"team_name" json:"name"`
			TotalScore   int    `db:"total_score" json:"total_score"`
			TotalXCount  int    `db:"total_x_count" json:"x_count"`
			TeamRank     *int   `db:"team_rank" json:"rank"`
			Status       string `db:"status" json:"status"`
			ClubName     string `db:"club_name" json:"club_name"`
			CategoryName string `db:"category_name" json:"category_name"`
		}

		var teamRes TeamResult
		query := `
			SELECT 
				t.uuid,
				t.team_name,
				COALESCE(t.total_score, 0) as total_score,
				COALESCE(t.total_x_count, 0) as total_x_count,
				t.team_rank,
				COALESCE(t.status, 'registered') as status,
				COALESCE(cl.name, 'Independen') as club_name,
				COALESCE(ec.category_name_custom, 'Recurve Team') as category_name
			FROM teams t
			JOIN team_members tm ON t.uuid = tm.team_id
			JOIN tournament_participants ep ON tm.participant_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid OR ep.archer_id = a.id
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN tournament_categories ec ON COALESCE(t.category_id, t.event_id) = ec.uuid
			WHERE t.tournament_id = ? AND (
				ep.archer_id = ? 
				OR ep.archer_id IN (SELECT uuid FROM archers WHERE email = (SELECT email FROM archers WHERE uuid = ? OR id = ?))
				OR ep.archer_id IN (SELECT id FROM archers WHERE uuid = ? OR id = ?)
			)
			LIMIT 1
		`
		err = db.Get(&teamRes, query, eventUUID, archerID, archerID, archerID, archerID, archerID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"team": nil})
			return
		}

		type MemberResult struct {
			UUID        string `db:"uuid" json:"uuid"`
			ArcherName  string `db:"archer_name" json:"archer_name"`
			ClubName    string `db:"club_name" json:"club_name"`
			MemberOrder int    `db:"member_order" json:"member_order"`
			Score       int    `db:"score" json:"score"`
		}
		var members []MemberResult
		memberQuery := `
			SELECT 
				tm.uuid,
				COALESCE(a.full_name, 'Pemanah') as archer_name,
				COALESCE(cl.name, 'Independen') as club_name,
				tm.member_order,
				COALESCE(ep.qual_score, tm.total_score, 0) as score
			FROM team_members tm
			JOIN tournament_participants ep ON tm.participant_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid OR ep.archer_id = a.id
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			WHERE tm.team_id = ?
			ORDER BY tm.member_order ASC
		`
		_ = db.Select(&members, memberQuery, teamRes.UUID)

		c.JSON(http.StatusOK, gin.H{
			"team": gin.H{
				"uuid":          teamRes.UUID,
				"name":          teamRes.TeamName,
				"total_score":   teamRes.TotalScore,
				"x_count":       teamRes.TotalXCount,
				"rank":          teamRes.TeamRank,
				"status":        teamRes.Status,
				"club_name":     teamRes.ClubName,
				"category_name": teamRes.CategoryName,
				"members":       members,
			},
		})
	}
}


