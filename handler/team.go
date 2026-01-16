package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"archeryhub/models"
	"archeryhub/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// CreateTeam creates a new team for a tournament event
func CreateTeam(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		userID, _ := c.Get("user_id")

		var req models.CreateTeamRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		teamID := uuid.New().String()

		// Create team
		_, err := db.Exec(`
			INSERT INTO teams (id, tournament_id, event_id, team_name, country_code, country_name, status)
			VALUES (?, ?, ?, ?, ?, ?, 'active')
		`, teamID, tournamentID, req.EventID, req.TeamName, req.CountryCode, req.CountryName)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create team"})
			return
		}

		// Add team members
		for i, participantID := range req.MemberIDs {
			memberID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO team_members (id, team_id, participant_id, member_order, is_substitute)
				VALUES (?, ?, ?, ?, ?)
			`, memberID, teamID, participantID, i+1, i >= 3) // 4th member is substitute

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add team member"})
				return
			}
		}

		utils.LogActivity(db, userID.(string), tournamentID, "team_created", "team", teamID, 
			fmt.Sprintf("Created team: %s", req.TeamName), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      teamID,
			"message": "Team created successfully",
		})
	}
}

// GetTeams returns all teams for a tournament
func GetTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		eventID := c.Query("event_id")

		query := `
			SELECT t.*, COUNT(tm.id) as member_count 
			FROM teams t
			LEFT JOIN team_members tm ON t.id = tm.team_id
			WHERE t.tournament_id = ?
		`
		args := []interface{}{tournamentID}

		if eventID != "" {
			query += " AND t.event_id = ?"
			args = append(args, eventID)
		}

		query += " GROUP BY t.id ORDER BY t.total_score DESC, t.total_x_count DESC"

		var teams []models.Team
		err := db.Select(&teams, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch teams"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"teams": teams,
			"total": len(teams),
		})
	}
}

// GetTeam returns a single team with members
func GetTeam(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		teamID := c.Param("teamId")

		var team models.Team
		err := db.Get(&team, "SELECT * FROM teams WHERE id = ?", teamID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
			return
		}

		var members []models.TeamMemberWithDetails
		err = db.Select(&members, `
			SELECT tm.*, a.first_name, a.last_name, tp.back_number, a.country
			FROM team_members tm
			JOIN tournament_participants tp ON tm.participant_id = tp.id
			JOIN athletes a ON tp.athlete_id = a.id
			WHERE tm.team_id = ?
			ORDER BY tm.member_order
		`, teamID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch team members"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"team":    team,
			"members": members,
		})
	}
}

// MakeTeams automatically generates teams from qualification rankings (by country)
func MakeTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		userID, _ := c.Get("user_id")

		var req struct {
			EventID     string `json:"event_id" binding:"required"`
			TeamSize    int    `json:"team_size"` // Usually 3
			TopN        int    `json:"top_n"`     // Top N athletes per country
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.TeamSize == 0 {
			req.TeamSize = 3
		}
		if req.TopN == 0 {
			req.TopN = 3
		}

		// Get ranked participants grouped by country
		type RankedParticipant struct {
			ParticipantID string  `db:"participant_id"`
			Country       string  `db:"country"`
			TotalScore    int     `db:"total_score"`
			TotalXCount   int     `db:"total_x_count"`
			RowNum        int     `db:"row_num"`
		}

		var ranked []RankedParticipant
		err := db.Select(&ranked, `
			SELECT 
				tp.id as participant_id,
				COALESCE(a.country, 'UNK') as country,
				COALESCE(SUM(qs.end_total), 0) as total_score,
				COALESCE(SUM(qs.x_count), 0) as total_x_count,
				ROW_NUMBER() OVER (PARTITION BY a.country ORDER BY SUM(qs.end_total) DESC, SUM(qs.x_count) DESC) as row_num
			FROM tournament_participants tp
			JOIN athletes a ON tp.athlete_id = a.id
			LEFT JOIN qualification_scores qs ON qs.participant_id = tp.id
			WHERE tp.tournament_id = ? AND tp.event_id = ?
			GROUP BY tp.id, a.country
			HAVING row_num <= ?
			ORDER BY country, row_num
		`, tournamentID, req.EventID, req.TopN)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rankings"})
			return
		}

		// Group by country and create teams
		countryParticipants := make(map[string][]string)
		countryScores := make(map[string]int)
		countryXCounts := make(map[string]int)

		for _, r := range ranked {
			if r.RowNum <= req.TeamSize {
				countryParticipants[r.Country] = append(countryParticipants[r.Country], r.ParticipantID)
				countryScores[r.Country] += r.TotalScore
				countryXCounts[r.Country] += r.TotalXCount
			}
		}

		teamsCreated := 0
		for country, participants := range countryParticipants {
			if len(participants) >= req.TeamSize {
				teamID := uuid.New().String()
				
				_, err = db.Exec(`
					INSERT INTO teams (id, tournament_id, event_id, team_name, country_code, total_score, total_x_count, status)
					VALUES (?, ?, ?, ?, ?, ?, ?, 'active')
				`, teamID, tournamentID, req.EventID, country, country, countryScores[country], countryXCounts[country])

				if err != nil {
					continue
				}

				// Add members
				for i, pid := range participants[:req.TeamSize] {
					memberID := uuid.New().String()
					db.Exec(`
						INSERT INTO team_members (id, team_id, participant_id, member_order, is_substitute)
						VALUES (?, ?, ?, ?, false)
					`, memberID, teamID, pid, i+1)
				}

				teamsCreated++
			}
		}

		utils.LogActivity(db, userID.(string), tournamentID, "teams_generated", "team", "", 
			fmt.Sprintf("Auto-generated %d teams", teamsCreated), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{
			"message":       "Teams generated successfully",
			"teams_created": teamsCreated,
		})
	}
}

// SubmitTeamScore submits a score for a team end
func SubmitTeamScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			TeamID        string  `json:"team_id" binding:"required"`
			TournamentID  string  `json:"tournament_id" binding:"required"`
			Session       int     `json:"session" binding:"required"`
			DistanceOrder int     `json:"distance_order" binding:"required"`
			EndNumber     int     `json:"end_number" binding:"required"`
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
		`, scoreID, req.TeamID, req.TournamentID, req.Session, req.DistanceOrder, req.EndNumber,
			string(memberScoresJSON), endTotal, xCount, runningTotal, userID.(string))

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit team score"})
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
		BroadcastTournamentUpdate(req.TournamentID, gin.H{
			"type": "team_score_update",
			"data": gin.H{"team_id": req.TeamID, "end_total": endTotal, "running_total": runningTotal},
		})

		c.JSON(http.StatusCreated, gin.H{
			"id":            scoreID,
			"end_total":     endTotal,
			"running_total": runningTotal,
			"message":       "Team score submitted successfully",
		})
	}
}

// GetTeamRankings returns team qualification rankings
func GetTeamRankings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		eventID := c.Query("event_id")

		query := `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY t.total_score DESC, t.total_x_count DESC) as rank,
				t.id as team_id,
				t.team_name,
				t.country_code,
				t.total_score,
				t.total_x_count
			FROM teams t
			WHERE t.tournament_id = ?
		`
		args := []interface{}{tournamentID}

		if eventID != "" {
			query += " AND t.event_id = ?"
			args = append(args, eventID)
		}

		query += " ORDER BY t.total_score DESC, t.total_x_count DESC"

		var rankings []models.TeamRanking
		err := db.Select(&rankings, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch team rankings"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"rankings": rankings,
			"total":    len(rankings),
		})
	}
}
