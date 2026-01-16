package handler

import (
	"fmt"
	"net/http"

	"archeryhub-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Phase constants
const (
	PhaseQualification = "QUAL"
	PhaseR64           = "R64"
	PhaseR32           = "R32"
	PhaseR16           = "R16"
	PhaseQF            = "QF"
	PhaseSF            = "SF"
	PhaseBM            = "BM" // Bronze Medal
	PhaseGM            = "GM" // Gold Medal
)

// GetPhaseOrder returns the numeric order of a phase
func GetPhaseOrder(phase string) int {
	order := map[string]int{
		"QUAL": 0, "R64": 1, "R32": 2, "R16": 3,
		"R8": 4, "QF": 5, "SF": 6, "BM": 7, "GM": 8,
	}
	if o, ok := order[phase]; ok {
		return o
	}
	return -1
}

// GetNextPhase returns the next phase after the current one
func GetNextPhase(current string, numParticipants int) string {
	phases := []string{"R64", "R32", "R16", "R8", "QF", "SF", "GM"}
	
	// Determine starting phase based on participants
	startIdx := 0
	switch {
	case numParticipants <= 4:
		startIdx = 5 // SF
	case numParticipants <= 8:
		startIdx = 4 // QF
	case numParticipants <= 16:
		startIdx = 3 // R8
	case numParticipants <= 32:
		startIdx = 2 // R16
	case numParticipants <= 64:
		startIdx = 1 // R32
	}
	
	for i := startIdx; i < len(phases); i++ {
		if phases[i] == current && i+1 < len(phases) {
			return phases[i+1]
		}
	}
	return ""
}

// AdvanceToNextPhase advances winners from current phase to next phase
func AdvanceToNextPhase(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")
		userID, _ := c.Get("user_id")

		var req struct {
			CurrentPhase string `json:"current_phase" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get all completed matches in current phase
		type MatchResult struct {
			ID       string  `db:"id"`
			WinnerID *string `db:"winner_id"`
			LoserID  *string `db:"loser_id"`
		}

		var matches []MatchResult
		err := db.Select(&matches, `
			SELECT em.id, em.winner_id,
				CASE 
					WHEN em.winner_id = em.participant1_id THEN em.participant2_id
					ELSE em.participant1_id
				END as loser_id
			FROM elimination_matches em
			WHERE em.event_id = ? AND em.round = ? AND em.status = 'completed'
			ORDER BY em.match_number
		`, eventID, req.CurrentPhase)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get matches"})
			return
		}

		if len(matches) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No completed matches in this phase"})
			return
		}

		// Determine next phase
		nextPhase := GetNextPhase(req.CurrentPhase, len(matches)*2)
		if nextPhase == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Already at final phase"})
			return
		}

		// Get tournament ID
		var tournamentID string
		db.Get(&tournamentID, "SELECT tournament_id FROM elimination_matches WHERE event_id = ? LIMIT 1", eventID)

		// Create next round matches
		matchesCreated := 0
		for i := 0; i < len(matches)/2; i++ {
			matchID := uuid.New().String()
			p1 := matches[i*2].WinnerID
			p2 := matches[i*2+1].WinnerID

			_, err = db.Exec(`
				INSERT INTO elimination_matches 
				(id, tournament_id, event_id, round, match_number, participant1_id, participant2_id, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, 'pending')
			`, matchID, tournamentID, eventID, nextPhase, i+1, p1, p2)

			if err == nil {
				matchesCreated++
			}
		}

		// Handle bronze medal match if moving to GM from SF
		if nextPhase == "GM" && len(matches) >= 2 {
			bronzeID := uuid.New().String()
			loser1 := matches[0].LoserID
			loser2 := matches[1].LoserID

			db.Exec(`
				INSERT INTO elimination_matches 
				(id, tournament_id, event_id, round, match_number, participant1_id, participant2_id, status)
				VALUES (?, ?, ?, 'BM', 1, ?, ?, 'pending')
			`, bronzeID, tournamentID, eventID, loser1, loser2)
			matchesCreated++
		}

		utils.LogActivity(db, userID.(string), tournamentID, "phase_advanced", "elimination", eventID,
			fmt.Sprintf("Advanced from %s to %s", req.CurrentPhase, nextPhase), c.ClientIP(), c.Request.UserAgent())

		// Broadcast update
		BroadcastTournamentUpdate(tournamentID, gin.H{
			"type": "phase_advanced",
			"data": gin.H{"event_id": eventID, "new_phase": nextPhase},
		})

		c.JSON(http.StatusOK, gin.H{
			"message":         "Phase advanced successfully",
			"next_phase":      nextPhase,
			"matches_created": matchesCreated,
		})
	}
}

// GetFinalRankings returns the final rankings for an event (after all matches complete)
func GetFinalRankings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")

		type FinalRanking struct {
			Rank        int     `json:"rank" db:"rank"`
			AthleteID   string  `json:"athlete_id" db:"athlete_id"`
			AthleteName string  `json:"athlete_name" db:"athlete_name"`
			Country     *string `json:"country" db:"country"`
			Medal       string  `json:"medal" db:"medal"`
			FinalPhase  string  `json:"final_phase" db:"final_phase"`
		}

		var rankings []FinalRanking

		// Gold and Silver from GM match
		err := db.Select(&rankings, `
			SELECT 
				CASE WHEN em.winner_id = tp.id THEN 1 ELSE 2 END as rank,
				a.id as athlete_id,
				CONCAT(a.first_name, ' ', a.last_name) as athlete_name,
				a.country,
				CASE WHEN em.winner_id = tp.id THEN 'GOLD' ELSE 'SILVER' END as medal,
				'GM' as final_phase
			FROM elimination_matches em
			JOIN tournament_participants tp ON tp.id IN (em.participant1_id, em.participant2_id)
			JOIN athletes a ON tp.athlete_id = a.id
			WHERE em.event_id = ? AND em.round = 'GM' AND em.status = 'completed'
			ORDER BY rank
		`, eventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch GM results"})
			return
		}

		// Bronze from BM match
		var bronze []FinalRanking
		db.Select(&bronze, `
			SELECT 
				3 as rank,
				a.id as athlete_id,
				CONCAT(a.first_name, ' ', a.last_name) as athlete_name,
				a.country,
				'BRONZE' as medal,
				'BM' as final_phase
			FROM elimination_matches em
			JOIN tournament_participants tp ON em.winner_id = tp.id
			JOIN athletes a ON tp.athlete_id = a.id
			WHERE em.event_id = ? AND em.round = 'BM' AND em.status = 'completed'
		`, eventID)

		rankings = append(rankings, bronze...)

		// Add remaining participants based on elimination round
		type EliminatedAthlete struct {
			AthleteID   string  `db:"athlete_id"`
			AthleteName string  `db:"athlete_name"`
			Country     *string `db:"country"`
			Phase       string  `db:"phase"`
			QualRank    int     `db:"qual_rank"`
		}

		var eliminated []EliminatedAthlete
		db.Select(&eliminated, `
			SELECT DISTINCT
				a.id as athlete_id,
				CONCAT(a.first_name, ' ', a.last_name) as athlete_name,
				a.country,
				em.round as phase,
				COALESCE((SELECT SUM(qs.end_total) FROM qualification_scores qs WHERE qs.participant_id = tp.id), 0) as qual_rank
			FROM elimination_matches em
			JOIN tournament_participants tp ON 
				(em.participant1_id = tp.id AND em.winner_id != tp.id) OR
				(em.participant2_id = tp.id AND em.winner_id != tp.id)
			JOIN athletes a ON tp.athlete_id = a.id
			WHERE em.event_id = ? AND em.status = 'completed' AND em.round NOT IN ('GM', 'BM')
			ORDER BY 
				CASE em.round
					WHEN 'SF' THEN 1
					WHEN 'QF' THEN 2
					WHEN 'R8' THEN 3
					WHEN 'R16' THEN 4
					WHEN 'R32' THEN 5
					WHEN 'R64' THEN 6
				END,
				qual_rank DESC
		`, eventID)

		rank := len(rankings) + 1
		prevPhase := ""
		tieRank := rank

		for _, e := range eliminated {
			if e.Phase != prevPhase {
				tieRank = rank
				prevPhase = e.Phase
			}
			rankings = append(rankings, FinalRanking{
				Rank:        tieRank,
				AthleteID:   e.AthleteID,
				AthleteName: e.AthleteName,
				Country:     e.Country,
				Medal:       "",
				FinalPhase:  e.Phase,
			})
			rank++
		}

		c.JSON(http.StatusOK, gin.H{
			"rankings": rankings,
			"total":    len(rankings),
		})
	}
}

// GetMatchDetails returns detailed info for a specific match
func GetMatchDetails(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		type MatchDetail struct {
			ID               string  `db:"id" json:"id"`
			TournamentID     string  `db:"tournament_id" json:"tournament_id"`
			EventID          string  `db:"event_id" json:"event_id"`
			Round            string  `db:"round" json:"round"`
			MatchNumber      int     `db:"match_number" json:"match_number"`
			Participant1ID   *string `db:"participant1_id" json:"participant1_id"`
			Participant2ID   *string `db:"participant2_id" json:"participant2_id"`
			Score1           int     `db:"score1" json:"score1"`
			Score2           int     `db:"score2" json:"score2"`
			SetScore1        int     `db:"set_score1" json:"set_score1"`
			SetScore2        int     `db:"set_score2" json:"set_score2"`
			WinnerID         *string `db:"winner_id" json:"winner_id"`
			Status           string  `db:"status" json:"status"`
			ScheduledTime    *string `db:"scheduled_time" json:"scheduled_time"`
			ActualStartTime  *string `db:"actual_start_time" json:"actual_start_time"`
			ActualEndTime    *string `db:"actual_end_time" json:"actual_end_time"`
		}

		var match MatchDetail
		err := db.Get(&match, "SELECT * FROM elimination_matches WHERE id = ?", matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		// Get participant details
		type ParticipantInfo struct {
			ID          string  `db:"id" json:"id"`
			FirstName   string  `db:"first_name" json:"first_name"`
			LastName    string  `db:"last_name" json:"last_name"`
			Country     *string `db:"country" json:"country"`
			BackNumber  *string `db:"back_number" json:"back_number"`
			QualScore   int     `db:"qual_score" json:"qual_score"`
			QualRank    int     `db:"qual_rank" json:"qual_rank"`
		}

		var p1, p2 *ParticipantInfo

		if match.Participant1ID != nil {
			p1 = &ParticipantInfo{}
			db.Get(p1, `
				SELECT tp.id, a.first_name, a.last_name, a.country, tp.back_number,
					COALESCE(SUM(qs.end_total), 0) as qual_score,
					0 as qual_rank
				FROM tournament_participants tp
				JOIN athletes a ON tp.athlete_id = a.id
				LEFT JOIN qualification_scores qs ON qs.participant_id = tp.id
				WHERE tp.id = ?
				GROUP BY tp.id
			`, *match.Participant1ID)
		}

		if match.Participant2ID != nil {
			p2 = &ParticipantInfo{}
			db.Get(p2, `
				SELECT tp.id, a.first_name, a.last_name, a.country, tp.back_number,
					COALESCE(SUM(qs.end_total), 0) as qual_score,
					0 as qual_rank
				FROM tournament_participants tp
				JOIN athletes a ON tp.athlete_id = a.id
				LEFT JOIN qualification_scores qs ON qs.participant_id = tp.id
				WHERE tp.id = ?
				GROUP BY tp.id
			`, *match.Participant2ID)
		}

		c.JSON(http.StatusOK, gin.H{
			"match":        match,
			"participant1": p1,
			"participant2": p2,
		})
	}
}

// SetMatchSchedule sets the scheduled time for a match
func SetMatchSchedule(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")
		userID, _ := c.Get("user_id")

		var req struct {
			ScheduledTime string `json:"scheduled_time" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		result, err := db.Exec("UPDATE elimination_matches SET scheduled_time = ? WHERE id = ?", req.ScheduledTime, matchID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		var tournamentID string
		db.Get(&tournamentID, "SELECT tournament_id FROM elimination_matches WHERE id = ?", matchID)

		utils.LogActivity(db, userID.(string), tournamentID, "match_scheduled", "elimination_match", matchID,
			"Scheduled match for "+req.ScheduledTime, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Match scheduled successfully"})
	}
}

// StartMatch marks a match as ongoing
func StartMatch(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")
		userID, _ := c.Get("user_id")

		_, err := db.Exec(`
			UPDATE elimination_matches 
			SET status = 'ongoing', actual_start_time = NOW() 
			WHERE id = ? AND status = 'pending'
		`, matchID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start match"})
			return
		}

		var tournamentID string
		db.Get(&tournamentID, "SELECT tournament_id FROM elimination_matches WHERE id = ?", matchID)

		utils.LogActivity(db, userID.(string), tournamentID, "match_started", "elimination_match", matchID,
			"Match started", c.ClientIP(), c.Request.UserAgent())

		BroadcastTournamentUpdate(tournamentID, gin.H{
			"type": "match_started",
			"data": gin.H{"match_id": matchID},
		})

		c.JSON(http.StatusOK, gin.H{"message": "Match started"})
	}
}

// CompleteMatch marks a match as completed with winner
func CompleteMatch(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")
		userID, _ := c.Get("user_id")

		var req struct {
			WinnerID string `json:"winner_id" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE elimination_matches 
			SET status = 'completed', winner_id = ?, actual_end_time = NOW() 
			WHERE id = ?
		`, req.WinnerID, matchID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete match"})
			return
		}

		var tournamentID string
		db.Get(&tournamentID, "SELECT tournament_id FROM elimination_matches WHERE id = ?", matchID)

		utils.LogActivity(db, userID.(string), tournamentID, "match_completed", "elimination_match", matchID,
			"Match completed", c.ClientIP(), c.Request.UserAgent())

		BroadcastTournamentUpdate(tournamentID, gin.H{
			"type": "match_completed",
			"data": gin.H{"match_id": matchID, "winner_id": req.WinnerID},
		})

		c.JSON(http.StatusOK, gin.H{"message": "Match completed"})
	}
}
