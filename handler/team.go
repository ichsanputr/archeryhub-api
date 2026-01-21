package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"archeryhub-api/models"
	"archeryhub-api/utils"

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
			INSERT INTO teams (uuid, tournament_id, event_id, team_name, country_code, country_name, status)
			VALUES (?, ?, ?, ?, ?, ?, 'active')
		`, teamID, eventID, req.CategoryID, req.TeamName, req.CountryCode, req.CountryName)

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

		utils.LogActivity(db, userID.(string), eventID, "team_created", "team", teamID, 
			fmt.Sprintf("Created team: %s", req.TeamName), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      teamID,
			"message": "Team created successfully",
		})
	}
}

// GetTeams returns all teams for an event
func GetTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")
		categoryID := c.Query("category_id")

		query := `
			SELECT t.*, COUNT(tm.id) as member_count 
			FROM teams t
			LEFT JOIN team_members tm ON t.id = tm.team_id
			WHERE t.event_id = ?
		`
		args := []interface{}{eventID}

		if categoryID != "" {
			query += " AND t.category_id = ?"
			args = append(args, categoryID)
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

// GetMyTeams returns all teams managed by the authenticated user's organization or club
func GetMyTeams(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		query := `
			SELECT t.*, e.name as event_name, c.name as category_name, COUNT(tm.id) as member_count 
			FROM teams t
			JOIN events e ON t.tournament_id = e.uuid
			JOIN event_categories c ON t.event_id = c.uuid
			LEFT JOIN team_members tm ON t.uuid = tm.team_id
			WHERE e.organization_id = (SELECT organization_id FROM users WHERE uuid = ?)
			   OR e.club_id = (SELECT club_id FROM users WHERE uuid = ?)
			GROUP BY t.uuid
			ORDER BY t.created_at DESC
		`
		
		var teams []struct {
			models.Team
			EventName    string `json:"event_name" db:"event_name"`
			CategoryName string `json:"category_name" db:"category_name"`
			MemberCount  int    `json:"member_count" db:"member_count"`
		}
		
		err := db.Select(&teams, query, userID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch your teams"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Team not found"})
			return
		}

		var members []models.TeamMemberWithDetails
		err = db.Select(&members, `
			SELECT tm.*, a.full_name, tp.back_number, a.country
			FROM team_members tm
			JOIN event_participants tp ON tm.participant_id = tp.uuid
			JOIN archers a ON tp.athlete_id = a.uuid
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


// SubmitTeamScore submits a score for a team end
func SubmitTeamScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			TeamID        string  `json:"team_id" binding:"required"`
			EventID       string  `json:"event_id" binding:"required"`
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
		`, scoreID, req.TeamID, req.EventID, req.Session, req.DistanceOrder, req.EndNumber,
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
		BroadcastEventUpdate(req.EventID, gin.H{
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
		eventID := c.Param("eventId")
		categoryID := c.Query("category_id")

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
		args := []interface{}{eventID}

		if categoryID != "" {
			query += " AND t.event_id = ?"
			args = append(args, categoryID)
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
