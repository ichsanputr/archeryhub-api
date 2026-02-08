package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetPublicQualificationResults returns qualification results for a specific category
func GetPublicQualificationResults(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Query("category_id")
		
		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
			return
		}
		
		if categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
			return
		}

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Get total ends from first session
		var totalEnds int
		err = db.Get(&totalEnds, `
			SELECT COALESCE(MAX(total_ends), 12) 
			FROM qualification_sessions 
			WHERE event_uuid = ?
		`, eventUUID)
		if err != nil {
			totalEnds = 12
		}

		// Get results with end scores
		type ResultEntry struct {
			Rank          int             `json:"rank"`
			ParticipantUUID string          `json:"participant_id" db:"participant_uuid"`
			ArcherName    string          `json:"archer_name" db:"archer_name"`
			ClubName      *string         `json:"club_name" db:"club_name"`
			TotalScore    int             `json:"total_score" db:"total_score"`
			TotalTenX     int             `json:"total_10x" db:"total_10x"`
			TotalX        int             `json:"total_x" db:"total_x"`
			EndsCompleted int             `json:"ends_completed" db:"ends_completed"`
			EndScoresJSON sql.NullString  `db:"end_scores_json"`
			EndScores     []int           `json:"end_scores"`
		}

		var entries []ResultEntry
		err = db.Select(&entries, `
			SELECT 
				ep.uuid as participant_uuid,
				a.full_name as archer_name,
				cl.name as club_name,
				COALESCE(SUM(s.total_score_end), 0) as total_score,
				COALESCE(SUM(s.ten_count_end), 0) as total_10x,
				COALESCE(SUM(s.x_count_end), 0) as total_x,
				COUNT(DISTINCT s.end_number) as ends_completed,
				(
					SELECT CONCAT('[', GROUP_CONCAT(total_score_end ORDER BY end_number SEPARATOR ','), ']')
					FROM qualification_end_scores s2
					JOIN qualification_sessions qs2 ON s2.session_uuid = qs2.uuid
					WHERE s2.participant_uuid = ep.uuid AND qs2.event_uuid = ?
				) as end_scores_json
			FROM event_participants ep
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN qualification_target_assignments qta ON qta.participant_uuid = ep.uuid
			LEFT JOIN qualification_sessions qs ON qs.uuid = qta.session_uuid AND qs.event_uuid = ?
			LEFT JOIN qualification_end_scores s ON s.session_uuid = qs.uuid AND s.participant_uuid = ep.uuid
			WHERE ep.category_id = ? AND ep.status = 'confirmed'
			GROUP BY ep.uuid, ep.archer_id, a.full_name, cl.name
			HAVING total_score > 0
			ORDER BY total_score DESC, total_10x DESC, total_x DESC
		`, eventUUID, eventUUID, categoryID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch results", "details": err.Error()})
			return
		}

		// Parse end scores and add ranking
		for i := range entries {
			entries[i].Rank = i + 1
			
			if entries[i].EndScoresJSON.Valid && entries[i].EndScoresJSON.String != "" {
				var scores []int
				if err := json.Unmarshal([]byte(entries[i].EndScoresJSON.String), &scores); err == nil {
					entries[i].EndScores = scores
				} else {
					entries[i].EndScores = []int{}
				}
			} else {
				entries[i].EndScores = []int{}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"results": entries,
			"total_ends": totalEnds,
		})
	}
}

// GetPublicEliminationResults returns elimination bracket for a specific category
func GetPublicEliminationResults(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Query("category_id")
		
		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
			return
		}
		
		if categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
			return
		}

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Get bracket for this category
		type Bracket struct {
			UUID         string  `db:"uuid" json:"uuid"`
			BracketID    string  `db:"bracket_id" json:"bracket_id"`
			CategoryUUID string  `db:"category_uuid" json:"category_uuid"`
			BracketType  string  `db:"bracket_type" json:"bracket_type"`
			Format       string  `db:"format" json:"format"`
			BracketSize  int     `db:"bracket_size" json:"bracket_size"`
			GeneratedAt  *string `db:"generated_at" json:"generated_at"`
		}

		var bracket Bracket
		err = db.Get(&bracket, `
			SELECT 
				eb.uuid,
				eb.bracket_id,
				eb.category_uuid,
				eb.bracket_type,
				eb.format,
				eb.bracket_size,
				eb.generated_at
			FROM elimination_brackets eb
			WHERE eb.event_uuid = ? AND eb.category_uuid = ? AND eb.generated_at IS NOT NULL
			LIMIT 1
		`, eventUUID, categoryID)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusOK, gin.H{"bracket": nil})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bracket", "details": err.Error()})
			return
		}

		// Get all matches for this bracket
		type Match struct {
			UUID             string  `db:"uuid" json:"uuid"`
			RoundNo          int     `db:"round_no" json:"round_no"`
			MatchNo          int     `db:"match_no" json:"match_no"`
			EntryAUUID       *string `db:"entry_a_uuid" json:"entry_a_uuid"`
			EntryAName       *string `db:"entry_a_name" json:"entry_a_name"`
			EntryASeed       *int    `db:"entry_a_seed" json:"entry_a_seed"`
			EntryBUUID       *string `db:"entry_b_uuid" json:"entry_b_uuid"`
			EntryBName       *string `db:"entry_b_name" json:"entry_b_name"`
			EntryBSeed       *int    `db:"entry_b_seed" json:"entry_b_seed"`
			WinnerEntryUUID  *string `db:"winner_entry_uuid" json:"winner_entry_uuid"`
			Status           string  `db:"status" json:"status"`
			IsBye            bool    `db:"is_bye" json:"is_bye"`
			TotalScoreA      *int    `db:"total_score_a" json:"total_score_a"`
			TotalScoreB      *int    `db:"total_score_b" json:"total_score_b"`
			SetPointsA       *int    `db:"set_points_a" json:"set_points_a"`
			SetPointsB       *int    `db:"set_points_b" json:"set_points_b"`
		}

		var matches []Match
		err = db.Select(&matches, `
			SELECT 
				em.uuid,
				em.round_no,
				em.match_no,
				em.entry_a_uuid,
				COALESCE(a1.full_name, t1.team_name) as entry_a_name,
				ee1.seed as entry_a_seed,
				em.entry_b_uuid,
				COALESCE(a2.full_name, t2.team_name) as entry_b_name,
				ee2.seed as entry_b_seed,
				em.winner_entry_uuid,
				em.status,
				em.is_bye,
				(SELECT SUM(end_total) FROM elimination_match_ends WHERE match_uuid = em.uuid AND side = 'A') as total_score_a,
				(SELECT SUM(end_total) FROM elimination_match_ends WHERE match_uuid = em.uuid AND side = 'B') as total_score_b,
				0 as set_points_a,
				0 as set_points_b
			FROM elimination_matches em
			LEFT JOIN elimination_entries ee1 ON em.entry_a_uuid = ee1.uuid
			LEFT JOIN elimination_entries ee2 ON em.entry_b_uuid = ee2.uuid
			LEFT JOIN archers a1 ON ee1.participant_uuid = a1.uuid AND ee1.participant_type = 'archer'
			LEFT JOIN archers a2 ON ee2.participant_uuid = a2.uuid AND ee2.participant_type = 'archer'
			LEFT JOIN teams t1 ON ee1.participant_uuid = t1.uuid AND ee1.participant_type = 'team'
			LEFT JOIN teams t2 ON ee2.participant_uuid = t2.uuid AND ee2.participant_type = 'team'
			WHERE em.bracket_uuid = ?
			ORDER BY em.round_no ASC, em.match_no ASC
		`, bracket.UUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch matches", "details": err.Error()})
			return
		}

		// Group matches by round
		matchesByRound := make(map[int][]Match)
		for _, match := range matches {
			matchesByRound[match.RoundNo] = append(matchesByRound[match.RoundNo], match)
		}

		bracket_result := gin.H{
			"uuid":         bracket.UUID,
			"bracket_id":   bracket.BracketID,
			"bracket_type": bracket.BracketType,
			"format":       bracket.Format,
			"bracket_size": bracket.BracketSize,
			"generated_at": bracket.GeneratedAt,
			"matches":      matchesByRound,
		}

		c.JSON(http.StatusOK, gin.H{"bracket": bracket_result})
	}
}
