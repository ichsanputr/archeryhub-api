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

		teamID := uuid.New().String()

		_, err := db.Exec(`
			INSERT INTO teams (uuid, tournament_id, event_id, team_name, status)
			VALUES (?, ?, ?, ?, 'active')
		`, teamID, eventID, req.CategoryID, req.TeamName)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat tim"})
			return
		}

		// 1. Get team_size based on event type
		var teamSize int
		err = db.Get(&teamSize, `
			SELECT CASE 
				WHEN ret.code = 'mixed_team' THEN 2 
				WHEN ret.code = 'team' THEN 3 
				ELSE 1 
			END as team_size
			FROM event_categories ec
			JOIN ref_event_types ret ON ec.event_type_uuid = ret.uuid
			WHERE ec.uuid = ?`, req.CategoryID)
		if err != nil {
			teamSize = 3 // Fallback
		}

		// 2. Add team members
		for i, participantID := range req.MemberIDs {
			memberID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO team_members (uuid, team_id, participant_id, member_order)
				VALUES (?, ?, ?, ?)
			`, memberID, teamID, participantID, i+1)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan anggota tim"})
				return
			}
		}

		utils.LogActivity(db, userID.(string), eventID, "team_created", "team", teamID,
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
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		query := `
			SELECT t.*, COUNT(tm.uuid) as member_count 
			FROM teams t
			LEFT JOIN team_members tm ON t.uuid = tm.team_id
			WHERE t.tournament_id = ?
		`
		args := []interface{}{eventUUID}

		if categoryID != "" {
			query += " AND t.event_id = ?"
			args = append(args, categoryID)
		}

		query += " GROUP BY t.uuid ORDER BY t.team_rank ASC, t.total_score DESC, t.total_x_count DESC"

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
					COALESCE(cl.name, 'Independen') as club_name,
					COALESCE(SUM(qes.total_score_end), 0) as total_score,
					COALESCE(SUM(qes.x_count_end), 0) as total_x,
					tm.member_order
				FROM team_members tm
				JOIN event_participants ep ON tm.participant_id = ep.uuid
				LEFT JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN clubs cl ON a.club_id = cl.uuid
				LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
				WHERE tm.team_id = ?
				GROUP BY tm.uuid, ep.uuid, a.full_name, cl.name, tm.member_order
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

// GetMyTeams returns all teams managed by the authenticated user's organization or club
func GetMyTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		userType, _ := c.Get("user_type")

		query := `
			SELECT t.*, e.name as event_name, c.name as category_name, COUNT(tm.id) as member_count 
			FROM teams t
			JOIN events e ON t.tournament_id = e.uuid
			JOIN event_categories c ON t.event_id = c.uuid
			LEFT JOIN team_members tm ON t.uuid = tm.team_id
			WHERE `

		if userType == "organization" {
			query += "e.organization_id = ?"
		} else if userType == "club" {
			query += "e.club_id = ?"
		} else {
			c.JSON(http.StatusOK, gin.H{"data": []interface{}{}, "total": 0})
			return
		}

		query += `
			GROUP BY t.uuid
			ORDER BY t.created_at DESC
		`

		var teams []struct {
			models.Team
			EventName    string `json:"event_name" db:"event_name"`
			CategoryName string `json:"category_name" db:"category_name"`
			MemberCount  int    `json:"member_count" db:"member_count"`
		}

		err := db.Select(&teams, query, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data tim Anda"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"data":  teams,
			"total": len(teams),
		})
	}
}

// GetTeam returns a single team with members
func GetTeam(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamID := c.Param("teamId")

		var team models.Team
		err := db.Get(&team, "SELECT * FROM teams WHERE uuid = ?", teamID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tim tidak ditemukan"})
			return
		}

		var members []models.TeamMemberWithDetails
		err = db.Select(&members, `
			SELECT tm.uuid, tm.team_id, tm.participant_id, tm.member_order, COALESCE(a.full_name, '') as full_name, tp.target_name as back_number, COALESCE(a.city, '') as city
			FROM team_members tm
			JOIN event_participants tp ON tm.participant_id = tp.uuid
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			WHERE tm.team_id = ?
			ORDER BY tm.member_order
		`, teamID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data anggota tim"})
			return
		}

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

		// Get previous running total
		var prevRunningTotal int
		db.Get(&prevRunningTotal, `
			SELECT COALESCE(MAX(running_total), 0) 
			FROM team_scores 
			WHERE team_id = ? AND session = ? AND distance_order = ? AND end_number < ?
		`, req.TeamID, req.Session, req.DistanceOrder, req.EndNumber)

		runningTotal := prevRunningTotal + endTotal

		// Store member scores as JSON
		memberScoresJSON, _ := json.Marshal(req.MemberScores)

		scoreID := uuid.New().String()
		_, err := db.Exec(`
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
		db.Exec(`
			UPDATE teams SET 
				total_score = (SELECT COALESCE(SUM(end_total), 0) FROM team_scores WHERE team_id = ?),
				total_x_count = (SELECT COALESCE(SUM(x_count), 0) FROM team_scores WHERE team_id = ?)
			WHERE id = ?
		`, req.TeamID, req.TeamID, req.TeamID)

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

		// Get team size for the category
		var teamSize int
		err := db.Get(&teamSize, `
			SELECT CASE 
				WHEN ret.code = 'mixed_team' THEN 2 
				WHEN ret.code = 'team' THEN 3 
				ELSE 1 
			END as team_size
			FROM event_categories ec
			JOIN ref_event_types ret ON ec.event_type_uuid = ret.uuid
			WHERE ec.uuid = ?`, categoryID)
		if err != nil {
			teamSize = 3 // Fallback
		}

		type TeamRankingEntry struct {
			Rank           int    `json:"rank"`
			ClubID         string `json:"club_id" db:"club_id"`
			ClubName       string `json:"club_name" db:"club_name"`
			TotalScore     int    `json:"total_score" db:"total_score"`
			TotalXCount    int    `json:"total_x" db:"total_x"`
			MemberCount    int    `json:"member_count" db:"member_count"`
			MemberNames    string `json:"member_names" db:"member_names"`
			ParticipantIDs string `json:"participant_ids" db:"participant_ids"`
		}

		// SQL for Multi-Team and Partial Team support
		query := `
			SELECT 
				ROW_NUMBER() OVER(ORDER BY SUM(individual_score) DESC, SUM(individual_x) DESC) as rank,
				club_id,
				club_name,
				SUM(individual_score) as total_score,
				SUM(individual_x) as total_x,
				COUNT(*) as member_count,
				GROUP_CONCAT(archer_name ORDER BY individual_score DESC SEPARATOR ', ') as member_names,
				GROUP_CONCAT(archer_id ORDER BY individual_score DESC SEPARATOR ',') as participant_ids
			FROM (
				SELECT 
					a.uuid as archer_id,
					a.full_name as archer_name,
					cl.uuid as club_id,
					COALESCE(cl.name, 'Independen') as club_name,
					COALESCE(SUM(s.total_score_end), 0) as individual_score,
					COALESCE(SUM(s.x_count_end), 0) as individual_x,
					ROW_NUMBER() OVER(PARTITION BY a.club_id ORDER BY SUM(s.total_score_end) DESC, SUM(s.ten_count_end) DESC, SUM(s.x_count_end) DESC) as club_rank
				FROM event_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN clubs cl ON a.club_id = cl.uuid
				LEFT JOIN qualification_end_scores s ON s.participant_uuid = ep.uuid
				WHERE ep.category_id = ?
				GROUP BY ep.uuid, cl.uuid, cl.name
			) ranked
			WHERE club_id IS NOT NULL
			GROUP BY club_id, club_name, CEIL(club_rank / ?)
			HAVING member_count >= 2
			ORDER BY total_score DESC, total_x DESC
		`

		var rankings []TeamRankingEntry
		err = db.Select(&rankings, query, categoryID, teamSize)
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
			FROM event_categories ec
			JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE ec.event_id = ? AND ec.division_uuid = ? AND ec.category_uuid = ?
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
						FROM event_participants ep
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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// 1. Delete existing teams for this category to allow "Regeneration"
		// First get team UUIDs to delete members
		var teamUUIDs []string
		err = tx.Select(&teamUUIDs, "SELECT uuid FROM teams WHERE tournament_id = ? AND event_id = ?", tournamentID, req.CategoryID)
		if err == nil && len(teamUUIDs) > 0 {
			query, args, _ := sqlx.In("DELETE FROM team_members WHERE team_id IN (?)", teamUUIDs)
			query = db.Rebind(query)
			_, _ = tx.Exec(query, args...)

			_, _ = tx.Exec("DELETE FROM teams WHERE tournament_id = ? AND event_id = ?", tournamentID, req.CategoryID)
		}

		// 2. Insert new teams
		for i, teamReq := range req.Teams {
			teamUUID := uuid.New().String()

			_, err = tx.Exec(`
				INSERT INTO teams (uuid, tournament_id, event_id, team_name, team_rank, total_score, total_x_count, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, 'active')
			`, teamUUID, tournamentID, req.CategoryID, teamReq.TeamName, i+1, teamReq.TotalScore, teamReq.TotalX)

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

		utils.LogActivity(db, userID.(string), tournamentID, "teams_regenerated", "event", tournamentID,
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
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
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

		// 1. Check category type (Standard vs Mixed)
		var catInfo struct {
			TypeID           string `db:"event_type_uuid"`
			TypeCode         string `db:"type_code"`
			DivisionID       string `db:"division_uuid"`
			AgeGroupID       string `db:"category_uuid"`
			GenderDivisionID string `db:"gender_division_uuid"`
			DivisionName     string `db:"division_name"`
			TeamSize         int    `db:"team_size"`
		}
		err = db.Get(&catInfo, `
			SELECT 
				ec.event_type_uuid, 
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
			FROM event_categories ec
			JOIN ref_event_types ret ON ec.event_type_uuid = ret.uuid
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// 2. Clear old teams
		var teamUUIDs []string
		err = tx.Select(&teamUUIDs, "SELECT uuid FROM teams WHERE tournament_id = ? AND event_id = ?", eventUUID, req.CategoryID)
		if err == nil && len(teamUUIDs) > 0 {
			query, args, _ := sqlx.In("DELETE FROM team_members WHERE team_id IN (?)", teamUUIDs)
			query = db.Rebind(query)
			_, _ = tx.Exec(query, args...)
			_, _ = tx.Exec("DELETE FROM teams WHERE tournament_id = ? AND event_id = ?", eventUUID, req.CategoryID)
		}

		syncCount := 0
		syncDetails := gin.H{
			"category_id": req.CategoryID,
			"team_size":   catInfo.TeamSize,
			"type_code":   catInfo.TypeCode,
			"is_mixed":    isMixed,
		}

		if isMixed {
			var maleParticipants int
			_ = tx.Get(&maleParticipants, `
				SELECT COUNT(DISTINCT ep.archer_id)
				FROM event_participants ep
				WHERE ep.category_id = ?
			`, req.CategoryID)

			var femaleParticipants int
			_ = tx.Get(&femaleParticipants, `
				SELECT COUNT(DISTINCT ep.archer_id)
				FROM event_participants ep
				WHERE ep.category_id = ?
			`, req.CategoryID)

			// Find male/female categories
			var catIDs []struct {
				UUID   string `db:"uuid"`
				Gender string `db:"gender_code"`
			}
			err = tx.Select(&catIDs, `
				SELECT ec.uuid, rgd.code as gender_code
				FROM event_categories ec
				JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
				JOIN ref_event_types ret ON ec.event_type_uuid = ret.uuid
				WHERE ec.event_id = ? AND ec.division_uuid = ? AND ec.category_uuid = ?
				  AND ret.code = 'individual'
			`, eventUUID, catInfo.DivisionID, catInfo.AgeGroupID)

			if err != nil || len(catIDs) < 2 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Kategori pasangan putra/putri (Individual) tidak ditemukan atau tidak lengkap",
					"details": gin.H{
						"male_participants":   maleParticipants,
						"female_participants": femaleParticipants,
						"found_categories":    len(catIDs),
						"reason":              "Mixed team membutuhkan kategori Individual Putra dan Individual Putri dengan divisi dan kelompok umur yang sama.",
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
				c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal mengidentifikasi kategori putra/putri"})
				return
			}

			// Calculate mixed rankings and insert
			var rankings []struct {
				ClubID      string `db:"club_id"`
				ClubName    string `db:"club_name"`
				MaleID      string `db:"male_id"`
				FemaleID    string `db:"female_id"`
				MaleScore   int    `db:"male_score"`
				FemaleScore int    `db:"female_score"`
				TotalScore  int    `db:"total_score"`
				TotalX      int    `db:"total_x"`
			}

			query := `
				SELECT club_id, club_name, male_id, female_id, male_score, female_score, (male_score + female_score) as total_score, (male_x + female_x) as total_x
				FROM (
					SELECT 
						cl.uuid as club_id, cl.name as club_name, ep_ranked.rank_in_club,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.participant_id ELSE '' END) as male_id,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.participant_id ELSE '' END) as female_id,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_score ELSE 0 END) as male_score,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_score ELSE 0 END) as female_score,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_x ELSE 0 END) as male_x,
						MAX(CASE WHEN ep_ranked.category_id = ? THEN ep_ranked.individual_x ELSE 0 END) as female_x
					FROM (
						SELECT ep_team.archer_id, ep_team.uuid as participant_id, a.club_id, ep_indiv.category_id, 
                               COALESCE(SUM(s.total_score_end), 0) as individual_score, 
                               COALESCE(SUM(s.x_count_end), 0) as individual_x,
							ROW_NUMBER() OVER(PARTITION BY a.club_id, ep_indiv.category_id ORDER BY SUM(s.total_score_end) DESC, SUM(s.x_count_end) DESC) as rank_in_club
						FROM event_participants ep_team
						JOIN archers a ON ep_team.archer_id = a.uuid
						JOIN event_participants ep_indiv ON a.uuid = ep_indiv.archer_id
						LEFT JOIN qualification_end_scores s ON s.participant_uuid = ep_indiv.uuid
						WHERE ep_team.category_id = ? AND ep_indiv.category_id IN (?, ?)
						GROUP BY ep_team.archer_id, ep_team.uuid, a.club_id, ep_indiv.category_id
					) ep_ranked
					JOIN clubs cl ON ep_ranked.club_id = cl.uuid
					GROUP BY cl.uuid, cl.name, ep_ranked.rank_in_club
					HAVING male_score > 0 AND female_score > 0
				) mixed ORDER BY total_score DESC, total_x DESC`

			err = tx.Select(&rankings, query, maleCatID, femaleCatID, maleCatID, femaleCatID, maleCatID, femaleCatID, req.CategoryID, maleCatID, femaleCatID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung peringkat mixed team", "details": err.Error()})
				return
			}

			syncDetails["male_participants"] = maleParticipants
			syncDetails["female_participants"] = femaleParticipants
			syncDetails["eligible_team_groups"] = len(rankings)

			clubCounter := make(map[string]int)
			letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
			for i, r := range rankings {
				clubCounter[r.ClubID]++
				suffix := ""
				if clubCounter[r.ClubID] <= len(letters) {
					suffix = " " + string(letters[clubCounter[r.ClubID]-1])
				}

				teamUUID := uuid.New().String()
				if _, err = tx.Exec(`INSERT INTO teams (uuid, tournament_id, event_id, team_name, team_rank, total_score, total_x_count) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					teamUUID, eventUUID, req.CategoryID, "Mixed "+r.ClubName+suffix, i+1, r.TotalScore, r.TotalX); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memasukkan data mixed team", "details": err.Error()})
					return
				}

				pIDs := []string{r.MaleID, r.FemaleID}
				for order, pID := range pIDs {
					if _, err = tx.Exec(`INSERT INTO team_members (uuid, team_id, participant_id, member_order) VALUES (?, ?, ?, ?)`,
						uuid.New().String(), teamUUID, pID, order+1); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memasukkan anggota mixed team", "details": err.Error()})
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
			participantCatID := req.CategoryID
			var indivCatID string
			if err2 := tx.Get(&indivCatID, `
				SELECT ec.uuid
				FROM event_categories ec
				JOIN ref_event_types ret ON ec.event_type_uuid = ret.uuid
				WHERE ec.event_id = ? AND ec.division_uuid = ? AND ec.category_uuid = ?
				  AND ec.gender_division_uuid = ? AND ret.code = 'individual'
			`, eventUUID, catInfo.DivisionID, catInfo.AgeGroupID, catInfo.GenderDivisionID); err2 == nil && indivCatID != "" {
				participantCatID = indivCatID
			}

			var totalParticipants int
			_ = tx.Get(&totalParticipants, `
				SELECT COUNT(DISTINCT ep.archer_id)
				FROM event_participants ep
				WHERE ep.category_id = ?
			`, req.CategoryID)

			var clubsWithParticipants int
			_ = tx.Get(&clubsWithParticipants, `
				SELECT COUNT(DISTINCT a.club_id)
				FROM event_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				WHERE ep.category_id = ? AND a.club_id IS NOT NULL AND a.club_id <> ''
			`, req.CategoryID)

			query := `
				SELECT club_id, club_name, SUM(individual_score) as total_score, SUM(individual_x) as total_x, GROUP_CONCAT(participant_id ORDER BY individual_score DESC SEPARATOR ',') as participant_ids, COUNT(*) as member_count
				FROM (
					SELECT ep_team.uuid as participant_id, cl.uuid as club_id, COALESCE(cl.name, 'Independen') as club_name, COALESCE(SUM(s.total_score_end), 0) as individual_score, COALESCE(SUM(s.x_count_end), 0) as individual_x,
						ROW_NUMBER() OVER(PARTITION BY a.club_id ORDER BY SUM(s.total_score_end) DESC, SUM(s.ten_count_end) DESC, SUM(s.x_count_end) DESC) as club_rank
					FROM event_participants ep_team
					JOIN archers a ON ep_team.archer_id = a.uuid
					JOIN event_participants ep_indiv ON a.uuid = ep_indiv.archer_id
					LEFT JOIN clubs cl ON a.club_id = cl.uuid
					LEFT JOIN qualification_end_scores s ON s.participant_uuid = ep_indiv.uuid
					WHERE ep_team.category_id = ? AND ep_indiv.category_id = ?
					GROUP BY ep_team.uuid, cl.uuid, cl.name
				) ranked
				WHERE club_id IS NOT NULL
				GROUP BY club_id, club_name, CEIL(club_rank / ?)
				HAVING member_count >= ?
				ORDER BY total_score DESC, total_x DESC`

			err = tx.Select(&rankings, query, req.CategoryID, participantCatID, teamSize, teamSize)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung peringkat tim", "details": err.Error()})
				return
			}

			syncDetails["total_participants"] = totalParticipants
			syncDetails["clubs_with_participants"] = clubsWithParticipants
			syncDetails["eligible_team_groups"] = len(rankings)

			clubCounter := make(map[string]int)
			letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
			for i, r := range rankings {
				clubCounter[r.ClubID]++
				suffix := ""
				if clubCounter[r.ClubID] <= len(letters) {
					suffix = " " + string(letters[clubCounter[r.ClubID]-1])
				}

				teamUUID := uuid.New().String()
				if _, err = tx.Exec(`INSERT INTO teams (uuid, tournament_id, event_id, team_name, team_rank, total_score, total_x_count) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					teamUUID, eventUUID, req.CategoryID, r.ClubName+suffix, i+1, r.TotalScore, r.TotalX); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memasukkan data tim", "details": err.Error()})
					return
				}

				pIDs := strings.Split(r.ParticipantIDs, ",")
				for order, pID := range pIDs {
					if _, err = tx.Exec(`INSERT INTO team_members (uuid, team_id, participant_id, member_order) VALUES (?, ?, ?, ?)`,
						uuid.New().String(), teamUUID, pID, order+1); err != nil {
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memasukkan anggota tim", "details": err.Error()})
						return
					}
				}
				syncCount++
			}
		}

		if syncCount == 0 {
			if isMixed {
				syncDetails["reason"] = "Belum ada kombinasi klub dengan 1 putra + 1 putri yang keduanya punya skor kualifikasi"
			} else {
				syncDetails["reason"] = fmt.Sprintf("Belum ada grup klub dengan minimal %d peserta berskor di kategori ini", catInfo.TeamSize)
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan hasil sinkronisasi"})
			return
		}

		utils.LogActivity(db, userID.(string), eventUUID, "teams_synced_directly", "event", eventUUID,
			fmt.Sprintf("Directly synced %d teams for category %s", syncCount, req.CategoryID), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Sinkronisasi selesai", "count": syncCount, "details": syncDetails})
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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// 1. Update team basic info
		_, err = tx.Exec(`
			UPDATE teams SET team_name = ?, event_id = ?
			WHERE uuid = ?
		`, req.TeamName, req.CategoryID, teamID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui info tim"})
			return
		}

		// 2. Delete existing members
		_, err = tx.Exec("DELETE FROM team_members WHERE team_id = ?", teamID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset anggota tim"})
			return
		}

		// 3. Get team_size based on event type
		var teamSize int
		err = tx.Get(&teamSize, `
			SELECT CASE 
				WHEN ret.code = 'mixed_team' THEN 2 
				WHEN ret.code = 'team' THEN 3 
				ELSE 1 
			END as team_size
			FROM event_categories ec
			JOIN ref_event_types ret ON ec.event_type_uuid = ret.uuid
			WHERE ec.uuid = ?`, req.CategoryID)
		if err != nil {
			teamSize = 3 // Fallback
		}

		// 4. Add new members
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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// 1. Delete members
		_, err = tx.Exec("DELETE FROM team_members WHERE team_id = ?", teamID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus anggota tim"})
			return
		}

		// 2. Delete team
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

