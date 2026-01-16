package handler

import (
	"net/http"

	"archeryhub/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// === QUALIFICATION SCORING ===

type QualificationScore struct {
	ID            string  `db:"id" json:"id"`
	TournamentID  string  `db:"tournament_id" json:"tournament_id"`
	ParticipantID string  `db:"participant_id" json:"participant_id"`
	Session       int     `db:"session" json:"session"`
	DistanceOrder int     `db:"distance_order" json:"distance_order"`
	EndNumber     int     `db:"end_number" json:"end_number"`
	Arrow1        *int    `db:"arrow_1" json:"arrow_1"`
	Arrow2        *int    `db:"arrow_2" json:"arrow_2"`
	Arrow3        *int    `db:"arrow_3" json:"arrow_3"`
	Arrow4        *int    `db:"arrow_4" json:"arrow_4"`
	Arrow5        *int    `db:"arrow_5" json:"arrow_5"`
	Arrow6        *int    `db:"arrow_6" json:"arrow_6"`
	EndTotal      int     `db:"end_total" json:"end_total"`
	RunningTotal  int     `db:"running_total" json:"running_total"`
	XCount        int     `db:"x_count" json:"x_count"`
	TenCount      int     `db:"ten_count" json:"ten_count"`
	Verified      bool    `db:"verified" json:"verified"`
	EnteredBy     *string `db:"entered_by" json:"entered_by"`
	EnteredAt     string  `db:"entered_at" json:"entered_at"`
}

// SubmitQualificationScore submits scores for a qualification end
func SubmitQualificationScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req QualificationScore
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Calculate end total and X/10 counts
		arrows := []*int{req.Arrow1, req.Arrow2, req.Arrow3, req.Arrow4, req.Arrow5, req.Arrow6}
		endTotal := 0
		xCount := 0
		tenCount := 0

		for _, arrow := range arrows {
			if arrow != nil {
				endTotal += *arrow
				if *arrow == 10 {
					tenCount++
				} else if *arrow == 11 { // X score represented as 11
					xCount++
					tenCount++                    // X also counts as 10
					endTotal = endTotal - 11 + 10 // Adjust to count X as 10
				}
			}
		}

		req.EndTotal = endTotal
		req.XCount = xCount
		req.TenCount = tenCount

		// Calculate running total
		var prevRunningTotal int
		err := db.Get(&prevRunningTotal, `
			SELECT COALESCE(MAX(running_total), 0) 
			FROM qualification_scores 
			WHERE tournament_id = ? AND participant_id = ? AND session = ? AND distance_order = ? AND end_number < ?
		`, req.TournamentID, req.ParticipantID, req.Session, req.DistanceOrder, req.EndNumber)

		if err == nil {
			req.RunningTotal = prevRunningTotal + endTotal
		} else {
			req.RunningTotal = endTotal
		}

		scoreID := uuid.New().String()
		userIDStr := userID.(string)

		_, err = db.Exec(`
			INSERT INTO qualification_scores 
			(id, tournament_id, participant_id, session, distance_order, end_number,
			 arrow_1, arrow_2, arrow_3, arrow_4, arrow_5, arrow_6,
			 end_total, running_total, x_count, ten_count, verified, entered_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE
			 arrow_1 = VALUES(arrow_1), arrow_2 = VALUES(arrow_2), arrow_3 = VALUES(arrow_3),
			 arrow_4 = VALUES(arrow_4), arrow_5 = VALUES(arrow_5), arrow_6 = VALUES(arrow_6),
			 end_total = VALUES(end_total), running_total = VALUES(running_total),
			 x_count = VALUES(x_count), ten_count = VALUES(ten_count), entered_by = VALUES(entered_by)
		`, scoreID, req.TournamentID, req.ParticipantID, req.Session, req.DistanceOrder, req.EndNumber,
			req.Arrow1, req.Arrow2, req.Arrow3, req.Arrow4, req.Arrow5, req.Arrow6,
			req.EndTotal, req.RunningTotal, req.XCount, req.TenCount, req.Verified, userIDStr)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit score"})
			return
		}

		// Log activity
		utils.LogActivity(db, userIDStr, req.TournamentID, "score_submitted", "qualification_score", scoreID, fmt.Sprintf("Submitted score for participant %s, end %d", req.ParticipantID, req.EndNumber), c.ClientIP(), c.Request.UserAgent())

		// Broadcast update via WebSocket
		BroadcastTournamentUpdate(req.TournamentID, gin.H{
			"type": "score_update",
			"data": req,
		})

		c.JSON(http.StatusCreated, gin.H{
			"id":            scoreID,
			"message":       "Score submitted successfully",
			"end_total":     endTotal,
			"running_total": req.RunningTotal,
		})
	}
}

// GetQualificationScores returns scores for a participant
func GetQualificationScores(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		participantID := c.Param("participantId")

		var scores []QualificationScore
		err := db.Select(&scores, `
			SELECT * FROM qualification_scores 
			WHERE participant_id = ?
			ORDER BY session ASC, distance_order ASC, end_number ASC
		`, participantID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scores"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"scores": scores,
			"total":  len(scores),
		})
	}
}

// GetQualificationRankings returns the qualification rankings for a tournament/event
func GetQualificationRankings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		eventID := c.Query("event_id")

		type Ranking struct {
			Rank          int     `db:"rank" json:"rank"`
			AthleteID     string  `db:"athlete_id" json:"athlete_id"`
			AthleteName   string  `db:"athlete_name" json:"athlete_name"`
			Country       *string `db:"country" json:"country"`
			BackNumber    *string `db:"back_number" json:"back_number"`
			TotalScore    int     `db:"total_score" json:"total_score"`
			TotalXCount   int     `db:"total_x_count" json:"total_x_count"`
			TotalTenCount int     `db:"total_ten_count" json:"total_ten_count"`
		}

		query := `
			SELECT 
				ROW_NUMBER() OVER (ORDER BY SUM(qs.end_total) DESC, SUM(qs.x_count) DESC, SUM(qs.ten_count) DESC) as rank,
				a.id as athlete_id,
				CONCAT(a.first_name, ' ', a.last_name) as athlete_name,
				a.country,
				tp.back_number,
				SUM(qs.end_total) as total_score,
				SUM(qs.x_count) as total_x_count,
				SUM(qs.ten_count) as total_ten_count
			FROM qualification_scores qs
			JOIN tournament_participants tp ON qs.participant_id = tp.id
			JOIN athletes a ON tp.athlete_id = a.id
			WHERE qs.tournament_id = ?
		`

		args := []interface{}{tournamentID}

		if eventID != "" {
			query += " AND tp.event_id = ?"
			args = append(args, eventID)
		}

		query += `
			GROUP BY a.id, athlete_name, a.country, tp.back_number
			ORDER BY total_score DESC, total_x_count DESC, total_ten_count DESC
		`

		var rankings []Ranking
		err := db.Select(&rankings, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rankings"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"rankings": rankings,
			"total":    len(rankings),
		})
	}
}

// === ELIMINATION MATCHES ===

type EliminationMatch struct {
	ID              string  `db:"id" json:"id"`
	TournamentID    string  `db:"tournament_id" json:"tournament_id"`
	EventID         string  `db:"event_id" json:"event_id"`
	Round           string  `db:"round" json:"round"`
	MatchNumber     int     `db:"match_number" json:"match_number"`
	Participant1ID  *string `db:"participant1_id" json:"participant1_id"`
	Participant2ID  *string `db:"participant2_id" json:"participant2_id"`
	Score1          int     `db:"score1" json:"score1"`
	Score2          int     `db:"score2" json:"score2"`
	SetScore1       int     `db:"set_score1" json:"set_score1"`
	SetScore2       int     `db:"set_score2" json:"set_score2"`
	WinnerID        *string `db:"winner_id" json:"winner_id"`
	Status          string  `db:"status" json:"status"`
	ScheduledTime   *string `db:"scheduled_time" json:"scheduled_time"`
	ActualStartTime *string `db:"actual_start_time" json:"actual_start_time"`
	ActualEndTime   *string `db:"actual_end_time" json:"actual_end_time"`
}

// CreateEliminationBracket generates the elimination bracket
func CreateEliminationBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			TournamentID string `json:"tournament_id" binding:"required"`
			EventID      string `json:"event_id" binding:"required"`
			NumArchers   int    `json:"num_archers" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Determine round name based on number of archers
		startRound := "R64"

		switch {
		case req.NumArchers <= 4:
			startRound = "SF"
		case req.NumArchers <= 8:
			startRound = "QF"
		case req.NumArchers <= 16:
			startRound = "R8"
		case req.NumArchers <= 32:
			startRound = "R16"
		case req.NumArchers <= 64:
			startRound = "R32"
		}

		// Create matches
		matchesCreated := 0
		for i := 1; i <= req.NumArchers/2; i++ {
			matchID := uuid.New().String()
			_, err := db.Exec(`
				INSERT INTO elimination_matches 
				(id, tournament_id, event_id, round, match_number, status)
				VALUES (?, ?, ?, ?, ?, 'pending')
			`, matchID, req.TournamentID, req.EventID, startRound, i)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bracket"})
				return
			}
			matchesCreated++
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), req.TournamentID, "bracket_created", "elimination_match", req.EventID, "Created elimination bracket", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"message":         "Elimination bracket created successfully",
			"matches_created": matchesCreated,
			"start_round":     startRound,
		})
	}
}

// GetEliminationBracket returns the bracket for an event
func GetEliminationBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")

		var matches []EliminationMatch
		err := db.Select(&matches, `
			SELECT * FROM elimination_matches 
			WHERE event_id = ?
			ORDER BY 
				CASE round
					WHEN 'R64' THEN 1
					WHEN 'R32' THEN 2
					WHEN 'R16' THEN 3
					WHEN 'R8' THEN 4
					WHEN 'QF' THEN 5
					WHEN 'SF' THEN 6
					WHEN 'BM' THEN 7
					WHEN 'GM' THEN 8
				END,
				match_number ASC
		`, eventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bracket"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"matches": matches,
			"total":   len(matches),
		})
	}
}

// UpdateMatchScore updates the score for an elimination match
func UpdateMatchScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		var req struct {
			Score1    int    `json:"score1"`
			Score2    int    `json:"score2"`
			SetScore1 int    `json:"set_score1"`
			SetScore2 int    `json:"set_score2"`
			WinnerID  string `json:"winner_id"`
			Status    string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE elimination_matches 
			SET score1 = ?, score2 = ?, set_score1 = ?, set_score2 = ?, winner_id = ?, status = ?
			WHERE id = ?
		`, req.Score1, req.Score2, req.SetScore1, req.SetScore2, req.WinnerID, req.Status, matchID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update match score"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), "", "match_updated", "elimination_match", matchID, "Updated match score", c.ClientIP(), c.Request.UserAgent())

		// We need tournamentID for broadcast, let's fetch it if not available in context
		var tournamentID string
		db.Get(&tournamentID, "SELECT tournament_id FROM elimination_matches WHERE id = ?", matchID)
		if tournamentID != "" {
			BroadcastTournamentUpdate(tournamentID, gin.H{
				"type": "match_update",
				"data": req,
			})
		}

		c.JSON(http.StatusOK, gin.H{"message": "Match score updated successfully"})
	}
}
