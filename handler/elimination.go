package handler

import (
	"Archeris-api/models"
	"Archeris-api/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/jung-kurt/gofpdf"
	"github.com/sirupsen/logrus"
)

// MatchScoreRequest represents the request to update a match score
type MatchScoreRequest struct {
	EndNo   int      `json:"end_no" binding:"required"`
	ScoreA  int      `json:"score_a"`
	ScoreB  int      `json:"score_b"`
	ArrowsA []string `json:"arrows_a"`
	ArrowsB []string `json:"arrows_b"`
}

// FinishMatchRequest represents the request to finish a match
type FinishMatchRequest struct {
	WinnerEntryID string `json:"winner_entry_id" binding:"required"`
}

// ============= BRACKET CRUD =============

// GetBrackets returns all brackets for an event
func GetBrackets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Resolve event UUID
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		categoryID := c.Query("category_id")

		type BracketInfo struct {
			BracketID    string  `json:"id" db:"bracket_id"`
			UUID         string  `json:"uuid" db:"uuid"`
			EventUUID    string  `json:"event_id" db:"tournament_uuid"`
			CategoryUUID string  `json:"category_id" db:"category_uuid"`
			CategoryName string  `json:"category_name" db:"category_name"`
			BracketType  string  `json:"bracket_type" db:"bracket_type"`
			Format       string  `json:"format" db:"format"`
			BracketSize  int     `json:"bracket_size" db:"bracket_size"`
			EndsPerMatch int     `json:"ends_per_match" db:"ends_per_match"`
			ArrowsPerEnd int     `json:"arrows_per_end" db:"arrows_per_end"`
			StartTime    *string `json:"start_time" db:"start_time"`
			EndTime      *string `json:"end_time" db:"end_time"`
			Status       string  `json:"status" db:"status"`
			GeneratedAt  *string `json:"generated_at" db:"generated_at"`
			CreatedAt    string  `json:"created_at" db:"created_at"`
			MatchCount   int     `json:"match_count" db:"match_count"`
			IsLocked     bool    `json:"is_locked" db:"is_locked"`
		}

		query := `
			SELECT COALESCE(eb.bracket_id, eb.uuid) as bracket_id, eb.uuid, eb.tournament_uuid, eb.category_uuid, 
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' ', COALESCE(rag.name, ''), ' ', COALESCE(rgd.name, ''))) as category_name,
				eb.bracket_type, eb.format, eb.bracket_size, eb.status, eb.ends_per_match, eb.arrows_per_end,
				eb.start_time, eb.end_time, eb.generated_at, eb.created_at,
				COALESCE(eb.is_locked, 0) as is_locked,
				(SELECT COUNT(*) FROM elimination_matches em WHERE em.bracket_uuid = eb.uuid) as match_count
			FROM elimination_brackets eb
			LEFT JOIN tournament_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE eb.tournament_uuid = ?
		`
		args := []interface{}{eventUUID}

		if categoryID != "" {
			query += " AND eb.category_uuid = ?"
			args = append(args, categoryID)
		}

		query += " ORDER BY eb.created_at DESC"

		var brackets []BracketInfo
		err = db.Select(&brackets, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve brackets", "details": err.Error()})
			return
		}

		if brackets == nil {
			brackets = []BracketInfo{}
		}

		c.JSON(http.StatusOK, gin.H{"brackets": brackets, "total": len(brackets)})
	}
}

// GetBracket returns a single bracket with its entries and matches
func GetBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		type Bracket struct {
			BracketID    string  `json:"id" db:"bracket_id"`
			UUID         string  `json:"uuid" db:"uuid"`
			EventUUID    string  `json:"event_id" db:"tournament_uuid"`
			CategoryUUID string  `json:"category_id" db:"category_uuid"`
			CategoryName string  `json:"category_name" db:"category_name"`
			BracketType  string  `json:"bracket_type" db:"bracket_type"`
			Format       string  `json:"format" db:"format"`
			BracketSize  int     `json:"bracket_size" db:"bracket_size"`
			Status       string  `json:"status" db:"status"`
			EndsPerMatch int     `json:"ends_per_match" db:"ends_per_match"`
			ArrowsPerEnd int     `json:"arrows_per_end" db:"arrows_per_end"`
			StartTime    *string `json:"start_time" db:"start_time"`
			EndTime      *string `json:"end_time" db:"end_time"`
			GeneratedAt  *string `json:"generated_at" db:"generated_at"`
			CreatedAt    string  `json:"created_at" db:"created_at"`
		}

		var bracket Bracket
		err := db.Get(&bracket, `
			SELECT eb.bracket_id, eb.uuid, eb.tournament_uuid, eb.category_uuid, eb.bracket_type, eb.status, eb.format, eb.bracket_size, eb.ends_per_match, eb.arrows_per_end, eb.start_time, eb.end_time, eb.generated_at, eb.created_at,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' ', COALESCE(rag.name, ''), ' ', COALESCE(rgd.name, ''))) as category_name
			FROM elimination_brackets eb
			LEFT JOIN tournament_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE eb.bracket_id = ? OR eb.uuid = ?
		`, bracketID, bracketID)
		if err != nil {
			logrus.WithError(err).WithField("bracket_id", bracketID).Error("Failed to fetch bracket")
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket tidak ditemukan"})
			return
		}

		bracketUUID := bracket.UUID

		// Get entries
		type Entry struct {
			UUID            string `json:"id" db:"uuid"`
			ParticipantType string `json:"participant_type" db:"participant_type"`
			ParticipantUUID string `json:"participant_id" db:"participant_uuid"`
			ParticipantName string `json:"participant_name" db:"participant_name"`
			Seed            int    `json:"seed" db:"seed"`
			QualTotalScore  *int   `json:"qual_total_score" db:"qual_total_score"`
			QualTotalX      *int   `json:"qual_total_x" db:"qual_total_x"`
			QualTotal10     *int   `json:"qual_total_10" db:"qual_total_10"`
		}

		var entries []Entry
		db.Select(&entries, `
			SELECT ee.uuid, ee.participant_type, ee.participant_uuid, 
				CASE 
					WHEN ee.participant_type = 'archer' THEN COALESCE(a.full_name, 'Unknown')
					WHEN ee.participant_type = 'team' THEN COALESCE(t.team_name, 'Unknown Team')
				END as participant_name,
				ee.seed, ee.qual_total_score, ee.qual_total_x, ee.qual_total_10
			FROM elimination_entries ee
			LEFT JOIN tournament_participants ep ON ee.participant_type = 'archer' AND ee.participant_uuid = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN teams t ON ee.participant_type = 'team' AND ee.participant_uuid = t.uuid
			WHERE ee.bracket_uuid = ?
			ORDER BY ee.seed ASC
		`, bracketUUID)

		if entries == nil {
			entries = []Entry{}
		}

		// Get matches grouped by round
		type Match struct {
			UUID            string     `json:"uuid" db:"uuid"`
			MatchID         string     `json:"id" db:"match_id"`
			RoundNo         int        `json:"round_no" db:"round_no"`
			MatchNo         int        `json:"match_no" db:"match_no"`
			EntryAUUID      *string    `json:"entry_a_id" db:"entry_a_uuid"`
			EntryBUUID      *string    `json:"entry_b_id" db:"entry_b_uuid"`
			EntryAName      *string    `json:"entry_a_name" db:"entry_a_name"`
			EntryBName      *string    `json:"entry_b_name" db:"entry_b_name"`
			EntryASeed      *int       `json:"entry_a_seed" db:"entry_a_seed"`
			EntryBSeed      *int       `json:"entry_b_seed" db:"entry_b_seed"`
			WinnerEntryUUID *string    `json:"winner_entry_id" db:"winner_entry_uuid"`
			Status          string     `json:"status" db:"status"`
			IsBye           bool       `json:"is_bye" db:"is_bye"`
			ScheduledAt     *time.Time `json:"scheduled_at" db:"scheduled_at"`
			TargetUUID      *string    `json:"target_id" db:"target_uuid"`
			TargetName      *string    `json:"target_name" db:"target_name"`
			BoardCode       *string    `json:"board_code" db:"board_code"`
			TotalScoreA     int        `json:"total_score_a" db:"total_score_a"`
			TotalScoreB     int        `json:"total_score_b" db:"total_score_b"`
			TotalPointsA    int        `json:"total_points_a" db:"total_points_a"`
			TotalPointsB    int        `json:"total_points_b" db:"total_points_b"`
			ShootOffA       *string    `json:"shoot_off_a"`
			ShootOffB       *string    `json:"shoot_off_b"`
		}

		var matches []Match
		err = db.Select(&matches, `
			SELECT em.uuid, em.match_id, em.round_no, em.match_no, 
				em.entry_a_uuid, em.entry_b_uuid,
				CASE 
					WHEN eeA.participant_type = 'archer' THEN aA.full_name
					WHEN eeA.participant_type = 'team' THEN tA.team_name
				END as entry_a_name,
				CASE 
					WHEN eeB.participant_type = 'archer' THEN aB.full_name
					WHEN eeB.participant_type = 'team' THEN tB.team_name
				END as entry_b_name,
				eeA.seed as entry_a_seed,
				eeB.seed as entry_b_seed,
				em.winner_entry_uuid, em.status, em.is_bye, em.scheduled_at,
				em.target_uuid, COALESCE(et.target_name, '') as target_name,
				COALESCE(tbe.code, '') as board_code,
				COALESCE(em.total_score_a, 0) as total_score_a,
				COALESCE(em.total_score_b, 0) as total_score_b,
				COALESCE(em.total_points_a, 0) as total_points_a,
				COALESCE(em.total_points_b, 0) as total_points_b
			FROM elimination_matches em
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN tournament_participants epA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = epA.uuid
			LEFT JOIN tournament_participants epB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = epB.uuid
			LEFT JOIN archers aA ON epA.archer_id = aA.uuid
			LEFT JOIN archers aB ON epB.archer_id = aB.uuid
			LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
			LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
			LEFT JOIN tournament_targets et ON em.target_uuid = et.uuid
			LEFT JOIN target_board_elimination tbe ON em.bracket_uuid = tbe.bracket_uuid AND et.board_number = tbe.board_number
			WHERE em.bracket_uuid = ?
			ORDER BY em.round_no ASC, em.match_no ASC
		`, bracketUUID)
		if err != nil {
			logrus.WithError(err).WithField("bracket_uuid", bracketUUID).Error("Gagal mengambil data pertandingan bracket")
		}

		if matches == nil {
			matches = []Match{}
		}

		// Fetch all ends for all matches to calculate total scores and set points
		type matchEnd struct {
			MatchUUID string `db:"match_uuid"`
			EndNo     int    `db:"end_no"`
			Side      string `db:"side"`
			EndTotal  int    `db:"end_total"`
		}
		var allEnds []matchEnd
		err = db.Select(&allEnds, `
			SELECT match_uuid, end_no, side, end_total
			FROM elimination_match_ends
			WHERE match_uuid IN (SELECT uuid FROM elimination_matches WHERE bracket_uuid = ?)
		`, bracketUUID)
		if err != nil {
			logrus.WithError(err).Error("Gagal mengambil data nilai babak bracket")
		}

		// Map to store ends per match: match_uuid -> end_no -> side -> total
		endsByMatch := make(map[string]map[int]map[string]int)
		for _, e := range allEnds {
			if endsByMatch[e.MatchUUID] == nil {
				endsByMatch[e.MatchUUID] = make(map[int]map[string]int)
			}
			if endsByMatch[e.MatchUUID][e.EndNo] == nil {
				endsByMatch[e.MatchUUID][e.EndNo] = make(map[string]int)
			}
			endsByMatch[e.MatchUUID][e.EndNo][e.Side] = e.EndTotal
		}

		// Fetch shoot-off arrows for all matches to show in summary
		type shootOffArrow struct {
			MatchUUID string `db:"match_uuid"`
			Side      string `db:"side"`
			Score     int    `db:"score"`
			IsX       bool   `db:"is_x"`
		}
		var allSoArrows []shootOffArrow
		db.Select(&allSoArrows, `
			SELECT eme.match_uuid, eme.side, emas.score, emas.is_x
			FROM elimination_match_arrow_scores emas
			JOIN elimination_match_ends eme ON emas.match_end_uuid = eme.uuid
			WHERE eme.match_uuid IN (SELECT uuid FROM elimination_matches WHERE bracket_uuid = ?)
			  AND eme.end_no = 99
		`, bracketUUID)

		soArrowsMap := make(map[string]map[string]string)
		for _, a := range allSoArrows {
			if soArrowsMap[a.MatchUUID] == nil {
				soArrowsMap[a.MatchUUID] = make(map[string]string)
			}
			val := fmt.Sprintf("%d", a.Score)
			if a.IsX {
				val = "X"
			} else if a.Score == 0 {
				val = "M"
			}
			soArrowsMap[a.MatchUUID][a.Side] = val
		}

		// Calculate scores and points for each match
		for i := range matches {
			mID := matches[i].UUID
			matchEnds := endsByMatch[mID]

			totalScoreA := 0
			totalScoreB := 0
			totalPointsA := 0
			totalPointsB := 0

			// Sort end numbers to process them in order for set points
			var endNos []int
			for en := range matchEnds {
				if en != 99 {
					endNos = append(endNos, en)
				}
			}

			for _, en := range endNos {
				scoreA := matchEnds[en]["A"]
				scoreB := matchEnds[en]["B"]
				totalScoreA += scoreA
				totalScoreB += scoreB

				if bracket.Format == "recurve_set" {
					if scoreA > scoreB {
						totalPointsA += 2
					} else if scoreB > scoreA {
						totalPointsB += 2
					} else if scoreA == scoreB && scoreA > 0 { // at least some arrows shot
						totalPointsA += 1
						totalPointsB += 1
					}
				}
			}

			// Add shoot-off info
			if soMap, ok := soArrowsMap[mID]; ok {
				var vA, vB string
				if val, ok := soMap["A"]; ok {
					matches[i].ShootOffA = &val
					vA = val
				}
				if val, ok := soMap["B"]; ok {
					matches[i].ShootOffB = &val
					vB = val
				}

				// Determiner winner of shoot-off and add 1 point
				if vA != "" && vB != "" {
					getV := func(v string) int {
						if v == "X" {
							return 11
						}
						if v == "M" {
							return 0
						}
						var sc int
						fmt.Sscanf(v, "%d", &sc)
						return sc
					}
					scA := getV(vA)
					scB := getV(vB)
					if scA > scB {
						if bracket.Format == "recurve_set" {
							totalPointsA++
						} else {
							totalScoreA++
						}
					} else if scB > scA {
						if bracket.Format == "recurve_set" {
							totalPointsB++
						} else {
							totalScoreB++
						}
					} else {
						// Tie in shoot-off - Award +1 to the recorded winner
						if matches[i].WinnerEntryUUID != nil && matches[i].Status == "finished" {
							if matches[i].EntryAUUID != nil && *matches[i].WinnerEntryUUID == *matches[i].EntryAUUID {
								if bracket.Format == "recurve_set" {
									totalPointsA++
								} else {
									totalScoreA++
								}
							} else if matches[i].EntryBUUID != nil && *matches[i].WinnerEntryUUID == *matches[i].EntryBUUID {
								if bracket.Format == "recurve_set" {
									totalPointsB++
								} else {
									totalScoreB++
								}
							}
						}
					}
				}
			}

			matches[i].TotalScoreA = totalScoreA
			matches[i].TotalScoreB = totalScoreB
			matches[i].TotalPointsA = totalPointsA
			matches[i].TotalPointsB = totalPointsB
		}

		// Group matches by round
		roundsMap := make(map[int][]Match)
		for _, m := range matches {
			roundsMap[m.RoundNo] = append(roundsMap[m.RoundNo], m)
		}

		c.JSON(http.StatusOK, gin.H{
			"bracket": bracket,
			"entries": entries,
			"matches": matches,
			"rounds":  roundsMap,
		})
	}
}

// GetBracketScores returns all match scores for a bracket
func GetBracketScores(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		// Resolve bracket UUID
		var bracket struct {
			UUID         string `db:"uuid"`
			EndsPerMatch int    `db:"ends_per_match"`
			ArrowsPerEnd int    `db:"arrows_per_end"`
		}
		err := db.Get(&bracket, `SELECT uuid, ends_per_match, arrows_per_end FROM elimination_brackets WHERE bracket_id = ? OR uuid = ?`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket tidak ditemukan"})
			return
		}

		type EndScore struct {
			MatchUUID string   `json:"match_id" db:"match_uuid"`
			UUID      string   `json:"id" db:"uuid"`
			EndNo     int      `json:"end_no" db:"end_no"`
			Side      string   `json:"side" db:"side"`
			EndTotal  int      `json:"end_total" db:"end_total"`
			XCount    int      `json:"x_count" db:"x_count"`
			TenCount  int      `json:"ten_count" db:"ten_count"`
			Arrows    []string `json:"arrows"`
		}

		var ends []EndScore
		err = db.Select(&ends, `
			SELECT uuid, match_uuid, end_no, side, end_total, x_count, ten_count
			FROM elimination_match_ends
			WHERE match_uuid IN (SELECT uuid FROM elimination_matches WHERE bracket_uuid = ?)
			ORDER BY match_uuid, end_no ASC, side ASC
		`, bracket.UUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve end data", "details": err.Error()})
			return
		}

		if ends == nil {
			ends = []EndScore{}
		}

		// Fetch all arrows for these matches in one query
		type arrowScore struct {
			EndUUID string `db:"match_end_uuid"`
			ArrowNo int    `db:"arrow_no"`
			Score   int    `db:"score"`
			IsX     bool   `db:"is_x"`
		}
		var allArrows []arrowScore
		err = db.Select(&allArrows, `
			SELECT emas.match_end_uuid, emas.arrow_no, emas.score, emas.is_x
			FROM elimination_match_arrow_scores emas
			JOIN elimination_match_ends eme ON emas.match_end_uuid = eme.uuid
			WHERE eme.match_uuid IN (SELECT uuid FROM elimination_matches WHERE bracket_uuid = ?)
			ORDER BY emas.match_end_uuid, emas.arrow_no ASC
		`, bracket.UUID)

		if err != nil {
			logrus.WithError(err).Error("Failed to fetch bracket arrow scores")
		}

		// Map arrows to ends
		arrowsMap := make(map[string][]string)
		for _, a := range allArrows {
			val := fmt.Sprintf("%d", a.Score)
			if a.IsX {
				val = "X"
			} else if a.Score == 0 {
				val = "M"
			}
			arrowsMap[a.EndUUID] = append(arrowsMap[a.EndUUID], val)
		}

		// Attach arrows to ends and ensure they are padded
		for i := range ends {
			sideArrows := arrowsMap[ends[i].UUID]
			if sideArrows == nil {
				sideArrows = []string{}
			}
			// Pad to arrows_per_end
			for len(sideArrows) < bracket.ArrowsPerEnd {
				sideArrows = append(sideArrows, "")
			}
			ends[i].Arrows = sideArrows
		}

		c.JSON(http.StatusOK, gin.H{
			"ends": ends,
		})
	}
}

// CreateBracket creates a new elimination bracket
func CreateBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Resolve event UUID
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		var req struct {
			CategoryID   string  `json:"category_id" binding:"required"`
			BracketType  string  `json:"bracket_type" binding:"required"`         // individual, team3, mixed2
			Format       string  `json:"format" binding:"required"`               // recurve_set, compound_total
			BracketSize  int     `json:"bracket_size" binding:"required,min=4"`   // chosen by admin; must be power of 2 and ≤ max
			EndsPerMatch int     `json:"ends_per_match" binding:"required,min=1"` // default 5
			ArrowsPerEnd int     `json:"arrows_per_end" binding:"required,min=1"` // default 3
			StartTime    *string `json:"start_time"`                              // ISO datetime, optional
			EndTime      *string `json:"end_time"`                                // ISO datetime, optional
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Count participants with qualification scores and auto-calculate bracket size (smallest power of 2 >= count)
		var participantCount int
		if req.BracketType == "individual" {
			db.Get(&participantCount, `
				SELECT COUNT(*) FROM (
					SELECT ep.uuid
					FROM tournament_participants ep
					JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
					WHERE ep.category_id = ? AND ep.payment_status IN ('paid', 'lunas')
					GROUP BY ep.uuid
					HAVING SUM(qes.total_score_end) > 0 OR COUNT(qes.uuid) > 0
				) scored_archers
			`, req.CategoryID)
		} else {
			// SyncTeams stores: tournament_id = eventUUID, event_id/category_id = categoryID
			db.Get(&participantCount, `SELECT COUNT(*) FROM teams WHERE (event_id = ? OR category_id = ?) AND tournament_id = ?`, req.CategoryID, req.CategoryID, eventUUID)
			if participantCount == 0 {
				c.JSON(http.StatusBadRequest, gin.H{
					"error":             "No teams found for this category. Please run Team Synchronization first.",
					"participant_count": 0,
				})
				return
			}
		}

		if participantCount < 2 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             "Not enough participants to create an elimination bracket (minimum 2 participants).",
				"participant_count": participantCount,
			})
			return
		}

		// Validate chosen bracket size: must be power of 2 and ≤ calcBracketSize(participantCount)
		maxBracketSize := calcBracketSize(participantCount)
		// Check power of 2
		if req.BracketSize < 4 || (req.BracketSize&(req.BracketSize-1)) != 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bracket size must be a power of 2 (4, 8, 16, 32, 64, ...)"})
			return
		}
		if req.BracketSize > maxBracketSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             fmt.Sprintf("Bracket size is too large. Maximum is %d based on %d participants.", maxBracketSize, participantCount),
				"max_bracket_size":  maxBracketSize,
				"participant_count": participantCount,
			})
			return
		}

		bracketSize := req.BracketSize

		bracketUUID := uuid.New().String()
		var bracketID string
		for {
			bracketID = utils.GenerateShortCode("BR", 5)
			var count int
			err := db.Get(&count, "SELECT COUNT(*) FROM elimination_brackets WHERE bracket_id = ?", bracketID)
			if err == nil && count == 0 {
				break
			}
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec(`
			INSERT INTO elimination_brackets (uuid, bracket_id, tournament_uuid, category_uuid, bracket_type, format, bracket_size, ends_per_match, arrows_per_end, start_time, end_time, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'generated')
		`, bracketUUID, bracketID, eventUUID, req.CategoryID, req.BracketType, req.Format, bracketSize, req.EndsPerMatch, req.ArrowsPerEnd, req.StartTime, req.EndTime)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bracket data", "details": err.Error()})
			return
		}

		// --- AUTO-GENERATE MATCHES & ENTRIES ---
		var entries []struct {
			ParticipantUUID string `db:"participant_uuid"`
			TotalScore      int    `db:"total_score"`
			TotalX          int    `db:"total_x"`
			Total10         int    `db:"total_10"`
		}

		if req.BracketType == "individual" {
			err = tx.Select(&entries, `
				SELECT ep.uuid as participant_uuid,
					COALESCE(SUM(qes.total_score_end), 0) as total_score,
					COALESCE(SUM(qes.x_count_end), 0) as total_x,
					COALESCE(SUM(qes.ten_count_end), 0) as total_10
				FROM tournament_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
				WHERE ep.category_id = ? AND ep.payment_status IN ('paid', 'lunas')
				GROUP BY ep.uuid
				HAVING SUM(qes.total_score_end) > 0 OR COUNT(qes.uuid) > 0
				ORDER BY total_score DESC, total_10 DESC, total_x DESC
				LIMIT ?
			`, req.CategoryID, bracketSize)
		} else {
			err = tx.Select(&entries, `
				SELECT t.uuid as participant_uuid,
					COALESCE(t.total_score, 0) as total_score,
					COALESCE(t.total_x_count, 0) as total_x,
					0 as total_10
				FROM teams t
				WHERE (t.event_id = ? OR t.category_id = ?) AND t.tournament_id = ?
				ORDER BY total_score DESC, total_x DESC
				LIMIT ?
			`, req.CategoryID, req.CategoryID, eventUUID, bracketSize)
		}

		if err != nil {
			logrus.WithError(err).Error("Gagal mengambil hasil kualifikasi untuk generate otomatis")
		}

		// Create entries
		participantType := "archer"
		if req.BracketType != "individual" {
			participantType = "team"
		}

		entryUUIDs := make([]string, bracketSize)
		for i := 0; i < bracketSize; i++ {
			entryUUID := uuid.New().String()
			entryUUIDs[i] = entryUUID
			if i < len(entries) {
				tx.Exec(`
					INSERT INTO elimination_entries (uuid, bracket_uuid, participant_type, participant_uuid, seed, qual_total_score, qual_total_x, qual_total_10)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, entryUUID, bracketUUID, participantType, entries[i].ParticipantUUID, i+1, entries[i].TotalScore, entries[i].TotalX, entries[i].Total10)
			}
		}

		// Generate matches
		numRounds := int(math.Log2(float64(bracketSize)))
		firstRoundMatchups := generateBracketSeeding(bracketSize)

		// globalMatchCounter ensures matches are numbered 1 to N
		globalMatchCounter := 1
		for roundNo := 1; roundNo <= numRounds; roundNo++ {
			matchesInRound := bracketSize / int(math.Pow(2, float64(roundNo)))
			for matchNo := 1; matchNo <= matchesInRound; matchNo++ {
				matchUUID := uuid.New().String()
				var entryAUUID, entryBUUID *string
				isBye := false

				if roundNo == 1 {
					matchIdx := matchNo - 1
					seedA := firstRoundMatchups[matchIdx*2]
					seedB := firstRoundMatchups[matchIdx*2+1]
					if seedA <= len(entries) {
						entryAUUID = &entryUUIDs[seedA-1]
					}
					if seedB <= len(entries) {
						entryBUUID = &entryUUIDs[seedB-1]
					}
					if (entryAUUID == nil || entryBUUID == nil) && !(entryAUUID == nil && entryBUUID == nil) {
						isBye = true
					}
				}

				matchID := strings.ToUpper(uuid.New().String()[:5])

				tx.Exec(`
					INSERT INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, is_bye)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, matchUUID, matchID, bracketUUID, roundNo, globalMatchCounter, entryAUUID, entryBUUID, isBye)
				globalMatchCounter++
			}
		}
		if bracketSize >= 4 {
			bronzeMatchUUID := uuid.New().String()
			bronzeMatchID := strings.ToUpper(uuid.New().String()[:5])
			tx.Exec(`
				INSERT INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, is_bye, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending')
			`, bronzeMatchUUID, bronzeMatchID, bracketUUID, numRounds, globalMatchCounter, nil, nil, false)
		}

		// Auto-advance BYE matches: players who get a BYE should automatically advance to Round 2
		byeMatches := []struct {
			UUID       string  `db:"uuid"`
			MatchNo    int     `db:"match_no"`
			EntryAUUID *string `db:"entry_a_uuid"`
		}{}
		tx.Select(&byeMatches, `
			SELECT uuid, match_no, entry_a_uuid
			FROM elimination_matches
			WHERE bracket_uuid = ? AND round_no = 1 AND is_bye = 1 AND entry_a_uuid IS NOT NULL
		`, bracketUUID)

		if bracketSize >= 4 {
			nextRoundOffset := getMatchNoOffset(bracketSize, 2)
			for _, byeMatch := range byeMatches {
				if byeMatch.EntryAUUID == nil {
					continue
				}
				// Mark this BYE match as finished with entry_a as winner
				tx.Exec(`
					UPDATE elimination_matches
					SET status = 'finished', winner_entry_uuid = ?
					WHERE uuid = ?
				`, *byeMatch.EntryAUUID, byeMatch.UUID)

				// Advance winner to Round 2 with global match offset
				relativeNextMatchNo := (byeMatch.MatchNo + 1) / 2
				globalNextMatchNo := nextRoundOffset + relativeNextMatchNo
				slot := "entry_b_uuid"
				if byeMatch.MatchNo%2 != 0 {
					slot = "entry_a_uuid"
				}
				tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = 2 AND match_no = ?`, slot),
					*byeMatch.EntryAUUID, bracketUUID, globalNextMatchNo)
			}
		} else {
			for _, byeMatch := range byeMatches {
				if byeMatch.EntryAUUID == nil {
					continue
				}
				tx.Exec(`
					UPDATE elimination_matches
					SET status = 'finished', winner_entry_uuid = ?
					WHERE uuid = ?
				`, *byeMatch.EntryAUUID, byeMatch.UUID)
			}
		}

		// Update generated_at
		now := time.Now().Format("2006-01-02 15:04:05")
		tx.Exec(`UPDATE elimination_brackets SET generated_at = ? WHERE uuid = ?`, now, bracketUUID)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save bracket creation"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Bracket created and generated successfully",
			"bracket": gin.H{
				"id":                bracketID,
				"uuid":              bracketUUID,
				"bracket_size":      bracketSize,
				"participant_count": participantCount,
				"byes":              bracketSize - participantCount,
			},
		})
	}
}

// UpdateBracket updates an existing elimination bracket
func UpdateBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		var req struct {
			CategoryID   string  `json:"category_id" binding:"required"`
			BracketType  string  `json:"bracket_type" binding:"required"`
			Format       string  `json:"format" binding:"required"`
			EndsPerMatch int     `json:"ends_per_match" binding:"required,min=1"`
			ArrowsPerEnd int     `json:"arrows_per_end" binding:"required,min=1"`
			StartTime    *string `json:"start_time"`
			EndTime      *string `json:"end_time"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if bracket exists
		var exists int
		err := db.Get(&exists, `SELECT 1 FROM elimination_brackets WHERE bracket_id = ? OR uuid = ?`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		// bracket_size is intentionally NOT updated — it is fixed at creation based on participant count
		_, err = db.Exec(`
			UPDATE elimination_brackets 
			SET category_uuid = ?, bracket_type = ?, format = ?, ends_per_match = ?, arrows_per_end = ?, start_time = ?, end_time = ?
			WHERE bracket_id = ? OR uuid = ?
		`, req.CategoryID, req.BracketType, req.Format, req.EndsPerMatch, req.ArrowsPerEnd, req.StartTime, req.EndTime, bracketID, bracketID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bracket", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Bracket updated successfully"})
	}
}

// ToggleLockEliminationBracket toggles lock status of an elimination bracket
func ToggleLockEliminationBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")
		if bracketID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bracketId is required"})
			return
		}

		var isLocked bool
		err := db.Get(&isLocked, `SELECT COALESCE(is_locked, 0) FROM elimination_brackets WHERE uuid = ? OR bracket_id = ?`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Elimination bracket not found"})
			return
		}

		newLockState := !isLocked
		_, err = db.Exec(`UPDATE elimination_brackets SET is_locked = ? WHERE uuid = ? OR bracket_id = ?`, newLockState, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bracket lock status"})
			return
		}

		msg := "Elimination bracket locked successfully"
		if !newLockState {
			msg = "Elimination bracket unlocked successfully"
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   msg,
			"is_locked": newLockState,
		})
	}
}

// GenerateBracket populates entries from qualification and creates match structure
func GenerateBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		// Get bracket info
		type Bracket struct {
			UUID         string `db:"uuid"`
			EventUUID    string `db:"event_uuid"`
			CategoryUUID string `db:"category_uuid"`
			BracketType  string `db:"bracket_type"`
			BracketSize  int    `db:"bracket_size"`
			Status       string `db:"status"`
		}

		var bracket Bracket
		err := db.Get(&bracket, `
			SELECT uuid, tournament_uuid as event_uuid, category_uuid, bracket_type, bracket_size, status
			FROM elimination_brackets
			WHERE bracket_id = ? OR uuid = ?
		`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		bracketUUID := bracket.UUID

		if bracket.Status == "running" || bracket.Status == "completed" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bracket has already been completed or is currently running"})
			return
		}

		var activeMatchesCount int
		_ = db.Get(&activeMatchesCount, `
			SELECT COUNT(*) 
			FROM elimination_matches 
			WHERE bracket_uuid = ? AND is_bye = 0 AND (status IN ('ongoing', 'finished', 'completed') OR COALESCE(total_score_a, 0) > 0 OR COALESCE(total_score_b, 0) > 0 OR COALESCE(total_points_a, 0) > 0 OR COALESCE(total_points_b, 0) > 0)
		`, bracketUUID)
		if activeMatchesCount > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Bracket cannot be regenerated because elimination matches have already started or finished. Please reset matches first.",
				"code":  "matches_already_started",
			})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// Clear existing entries and matches
		tx.Exec(`DELETE FROM elimination_entries WHERE bracket_uuid = ?`, bracketUUID)
		tx.Exec(`DELETE FROM elimination_matches WHERE bracket_uuid = ?`, bracketUUID)

		// Get qualified participants based on bracket type
		var entries []struct {
			ParticipantUUID string `db:"participant_uuid"`
			TotalScore      int    `db:"total_score"`
			TotalX          int    `db:"total_x"`
			Total10         int    `db:"total_10"`
		}

		if bracket.BracketType == "individual" {
			// Get top archers from qualification scores
			err = tx.Select(&entries, `
				SELECT ep.uuid as participant_uuid,
					COALESCE(SUM(qes.total_score_end), 0) as total_score,
					COALESCE(SUM(qes.x_count_end), 0) as total_x,
					COALESCE(SUM(qes.ten_count_end), 0) as total_10
				FROM tournament_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
				WHERE ep.category_id = ? AND ep.payment_status IN ('paid', 'lunas')
				GROUP BY ep.uuid
				HAVING SUM(qes.total_score_end) > 0 OR COUNT(qes.uuid) > 0
				ORDER BY total_score DESC, total_10 DESC, total_x DESC
				LIMIT ?
			`, bracket.CategoryUUID, bracket.BracketSize)
		} else {
			// Get teams
			err = tx.Select(&entries, `
				SELECT t.uuid as participant_uuid,
					COALESCE(t.total_score, 0) as total_score,
					COALESCE(t.total_x_count, 0) as total_x,
					0 as total_10
				FROM teams t
				WHERE (t.event_id = ? OR t.category_id = ?) AND t.tournament_id = ? AND (t.status = 'active' OR t.status IS NULL)
				ORDER BY total_score DESC, total_x DESC
				LIMIT ?
			`, bracket.CategoryUUID, bracket.CategoryUUID, bracket.EventUUID, bracket.BracketSize)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch qualification results", "details": err.Error()})
			return
		}

		if len(entries) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No qualified participants found for this category"})
			return
		}

		// Create entries with seeds
		participantType := "archer"
		if bracket.BracketType != "individual" {
			participantType = "team"
		}

		entryUUIDs := make([]string, bracket.BracketSize)
		for i := 0; i < bracket.BracketSize; i++ {
			entryUUID := uuid.New().String()
			entryUUIDs[i] = entryUUID

			if i < len(entries) {
				_, err = tx.Exec(`
					INSERT INTO elimination_entries (uuid, bracket_uuid, participant_type, participant_uuid, seed, qual_total_score, qual_total_x, qual_total_10)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, entryUUID, bracketUUID, participantType, entries[i].ParticipantUUID, i+1, entries[i].TotalScore, entries[i].TotalX, entries[i].Total10)
			}
		}

		// Generate bracket matches using standard seeding
		// For bracket size N, we have log2(N) rounds
		numRounds := int(math.Log2(float64(bracket.BracketSize)))

		// Generate first round matches using proper bracket seeding
		firstRoundMatchups := generateBracketSeeding(bracket.BracketSize)

		// globalMatchCounter will ensure matches are numbered 1 to N-1
		globalMatchCounter := 1

		for roundNo := 1; roundNo <= numRounds; roundNo++ {
			matchesInRound := bracket.BracketSize / int(math.Pow(2, float64(roundNo)))

			for matchNo := 1; matchNo <= matchesInRound; matchNo++ {
				matchUUID := uuid.New().String()
				matchID := strings.ToUpper(uuid.New().String()[:5])

				var entryAUUID, entryBUUID *string
				isBye := false

				if roundNo == 1 {
					// First round: use seeding
					matchIdx := matchNo - 1
					seedA := firstRoundMatchups[matchIdx*2]
					seedB := firstRoundMatchups[matchIdx*2+1]

					if seedA <= len(entries) {
						entryAUUID = &entryUUIDs[seedA-1]
					}
					if seedB <= len(entries) {
						entryBUUID = &entryUUIDs[seedB-1]
					}

					if (entryAUUID == nil || entryBUUID == nil) && !(entryAUUID == nil && entryBUUID == nil) {
						isBye = true
					}
				}

				_, err = tx.Exec(`
					INSERT INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, is_bye)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, matchUUID, matchID, bracketUUID, roundNo, globalMatchCounter, entryAUUID, entryBUUID, isBye)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create match data", "details": err.Error()})
					return
				}
				globalMatchCounter++
			}
		}

		// Create 3rd Place Match (Bronze Match)
		if bracket.BracketSize >= 4 {
			bronzeMatchUUID := uuid.New().String()
			bronzeMatchID := strings.ToUpper(uuid.New().String()[:5])
			_, err = tx.Exec(`
				INSERT INTO elimination_matches (uuid, match_id, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, is_bye, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'pending')
			`, bronzeMatchUUID, bronzeMatchID, bracketUUID, numRounds, globalMatchCounter, nil, nil, false)
			if err != nil {
				logrus.WithError(err).Error("Failed to create bronze match")
			}
		}

		// Auto-advance BYE matches: players who get a BYE should automatically advance to Round 2
		// Without this, scorekeepers have to manually finish each BYE match
		byeMatches := []struct {
			UUID       string  `db:"uuid"`
			MatchNo    int     `db:"match_no"`
			EntryAUUID *string `db:"entry_a_uuid"`
		}{}
		tx.Select(&byeMatches, `
			SELECT uuid, match_no, entry_a_uuid
			FROM elimination_matches
			WHERE bracket_uuid = ? AND round_no = 1 AND is_bye = 1 AND entry_a_uuid IS NOT NULL
		`, bracketUUID)

		if bracket.BracketSize >= 4 {
			nextRoundOffset := getMatchNoOffset(bracket.BracketSize, 2)
			for _, byeMatch := range byeMatches {
				if byeMatch.EntryAUUID == nil {
					continue
				}
				// Mark this BYE match as finished with entry_a as winner
				tx.Exec(`
					UPDATE elimination_matches
					SET status = 'finished', winner_entry_uuid = ?
					WHERE uuid = ?
				`, *byeMatch.EntryAUUID, byeMatch.UUID)

				// Advance winner to Round 2 with global match offset
				relativeNextMatchNo := (byeMatch.MatchNo + 1) / 2
				globalNextMatchNo := nextRoundOffset + relativeNextMatchNo
				slot := "entry_b_uuid"
				if byeMatch.MatchNo%2 != 0 {
					slot = "entry_a_uuid"
				}
				tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = 2 AND match_no = ?`, slot),
					*byeMatch.EntryAUUID, bracketUUID, globalNextMatchNo)
			}
		} else {
			for _, byeMatch := range byeMatches {
				if byeMatch.EntryAUUID == nil {
					continue
				}
				tx.Exec(`
					UPDATE elimination_matches
					SET status = 'finished', winner_entry_uuid = ?
					WHERE uuid = ?
				`, *byeMatch.EntryAUUID, byeMatch.UUID)
			}
		}

		// Update bracket status
		now := time.Now().Format("2006-01-02 15:04:05")
		_, err = tx.Exec(`UPDATE elimination_brackets SET status = 'generated', generated_at = ? WHERE uuid = ?`, now, bracketUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bracket status"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "Bracket created successfully",
			"entries_count": len(entries),
			"rounds":        numRounds,
		})
	}
}

// calcBracketSize returns the smallest power of 2 (min 4) that is >= n (per docs: Bracket Size = 2^⌈log₂(N)⌉)
func calcBracketSize(n int) int {
	if n < 2 {
		return 0
	}
	size := 4
	for size < n {
		size *= 2
	}
	return size
}

// GetBracketSizeRecommendation returns the auto-calculated bracket size and bye count for a category.
// For team brackets it calculates from BOTH synced teams (already in DB) and possible teams from
// qualification results (same logic as SyncTeams), so admins can see the bracket size even before
// SyncTeams has been run.
func GetBracketSizeRecommendation(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		categoryID := c.Query("category_id")
		bracketType := c.Query("bracket_type")

		if categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_id wajib diisi"})
			return
		}
		if bracketType == "" {
			bracketType = "individual"
		}

		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		effectiveCount := 0
		syncedTeams := 0
		possibleTeams := 0
		teamSize := 1

		if bracketType == "individual" {
			db.Get(&effectiveCount, `
				SELECT COUNT(*) FROM (
					SELECT ep.uuid
					FROM tournament_participants ep
					JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
					WHERE ep.category_id = ? AND ep.payment_status IN ('paid', 'lunas')
					GROUP BY ep.uuid
					HAVING SUM(qes.total_score_end) > 0 OR COUNT(qes.uuid) > 0
				) scored_archers
			`, categoryID)
		} else {
			// Fetch category type info (same as SyncTeams does)
			var catInfo struct {
				TypeCode   string `db:"type_code"`
				TeamSize   int    `db:"team_size"`
				DivUUID    string `db:"division_uuid"`
				AgeUUID    string `db:"age_uuid"`
				GenderUUID string `db:"gender_division_uuid"`
			}
			db.Get(&catInfo, `
				SELECT ret.code as type_code,
					CASE WHEN ret.code = 'mixed_team' THEN 2 WHEN ret.code = 'team' THEN 3 ELSE 1 END as team_size,
					ec.division_uuid, ec.category_uuid as age_uuid, ec.gender_division_uuid
				FROM tournament_categories ec
				JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
				WHERE ec.uuid = ?
			`, categoryID)
			teamSize = catInfo.TeamSize
			if teamSize <= 0 {
				teamSize = 3
			}

			// 1. Synced teams already in DB (SyncTeams stores event_id=categoryID, tournament_id=eventUUID)
			db.Get(&syncedTeams, `SELECT COUNT(*) FROM teams WHERE event_id = ? AND tournament_id = ?`, categoryID, eventUUID)

			isMixed := strings.Contains(catInfo.TypeCode, "mixed")

			// 2. Possible teams from qualification (preview before SyncTeams is run)
			if !isMixed {
				// Participants register under the individual category, not the team category.
				// Resolve the matching individual category (same event/division/age/gender).
				participantCatID := categoryID
				var indivCatID string
				if err2 := db.Get(&indivCatID, `
					SELECT ec.uuid
					FROM tournament_categories ec
					JOIN ref_tournament_types ret ON ec.tournament_type_uuid = ret.uuid
					WHERE ec.tournament_id = ? AND ec.division_uuid = ? AND ec.category_uuid = ?
					  AND ec.gender_division_uuid = ? AND ret.code = 'individual'
				`, eventUUID, catInfo.DivUUID, catInfo.AgeUUID, catInfo.GenderUUID); err2 == nil && indivCatID != "" {
					participantCatID = indivCatID
				}

				// Standard team: count eligible groups using the same grouping SyncTeams uses.
				// One club can form multiple teams: groups are CEIL(rank / teamSize).
				db.Get(&possibleTeams, `
					SELECT COUNT(*) FROM (
						SELECT club_id, COUNT(*) as mc
						FROM (
							SELECT a.uuid as archer_id, cl.uuid as club_id,
								ROW_NUMBER() OVER(PARTITION BY a.club_id ORDER BY COALESCE(SUM(s.total_score_end), 0) DESC) as club_rank
							FROM tournament_participants ep
							JOIN archers a ON ep.archer_id = a.uuid
							LEFT JOIN clubs cl ON a.club_id = cl.uuid
							LEFT JOIN qualification_end_scores s ON s.participant_uuid = ep.uuid
							WHERE ep.category_id = ?
							GROUP BY ep.uuid, cl.uuid
						) ranked
						WHERE club_id IS NOT NULL
						GROUP BY club_id, CEIL(club_rank / ?)
						HAVING mc >= ?
					) eg
				`, participantCatID, teamSize, teamSize)
			} else {
				// Mixed team: count clubs that have participants in BOTH male and female paired categories
				// (same division + age group, same event)
				db.Get(&possibleTeams, `
					SELECT COUNT(*) FROM (
						SELECT a.club_id
						FROM tournament_participants ep
						JOIN archers a ON ep.archer_id = a.uuid
						JOIN tournament_categories ec ON ep.category_id = ec.uuid
						JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
						WHERE ec.tournament_id = ?
						  AND ec.division_uuid = ?
						  AND ec.category_uuid = ?
						  AND a.club_id IS NOT NULL
						GROUP BY a.club_id
						HAVING COUNT(DISTINCT rgd.code) >= 2
					) eligible
				`, eventUUID, catInfo.DivUUID, catInfo.AgeUUID)
			}

			// Use synced teams if available; fall back to possible teams for preview
			effectiveCount = syncedTeams
			if effectiveCount == 0 {
				effectiveCount = possibleTeams
			}
		}

		bracketSize := calcBracketSize(effectiveCount)
		byes := 0
		if bracketSize > 0 {
			byes = bracketSize - effectiveCount
		}

		resp := gin.H{
			"participant_count": effectiveCount,
			"max_bracket_size":  bracketSize,
			"byes":              byes,
		}
		if bracketType != "individual" {
			resp["synced_teams"] = syncedTeams
			resp["possible_teams"] = possibleTeams
			resp["team_size"] = teamSize
		}
		c.JSON(http.StatusOK, resp)
	}
}

// generateBracketSeeding returns the proper seeding order for first round
// For size 8: [1,8,4,5,2,7,3,6] means match1: 1v8, match2: 4v5, match3: 2v7, match4: 3v6
func generateBracketSeeding(size int) []int {
	if size == 2 {
		return []int{1, 2}
	}

	prev := generateBracketSeeding(size / 2)
	result := make([]int, size)

	for i, seed := range prev {
		result[i*2] = seed
		result[i*2+1] = size + 1 - seed
	}

	return result
}

// getMatchNoOffset returns the total number of matches in all rounds prior to 'targetRound'
func getMatchNoOffset(size int, targetRound int) int {
	offset := 0
	for r := 1; r < targetRound; r++ {
		offset += size / int(math.Pow(2, float64(r)))
	}
	return offset
}

// UpdateMatchTargets assigns targets to matches in a bracket
func UpdateMatchTargets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		var req struct {
			Assignments []struct {
				MatchID     string `json:"match_id"`
				BoardNumber int    `json:"board_number"`
				TargetID    string `json:"target_id"`
				Code        string `json:"code"`
			} `json:"assignments" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve bracket UUID and event UUID
		var bracket struct {
			UUID      string `db:"uuid"`
			EventUUID string `db:"event_uuid"`
		}
		err := db.Get(&bracket, `SELECT uuid, tournament_uuid as event_uuid FROM elimination_brackets WHERE bracket_id = ? OR uuid = ?`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		updated := 0
		for _, assignment := range req.Assignments {
			if assignment.MatchID != "" {
				// Update by Match UUID or Human-Readable Match ID
				res, err := tx.Exec(`UPDATE elimination_matches SET target_uuid = ?, updated_at = NOW() WHERE (uuid = ? OR match_id = ?) AND bracket_uuid = ?`,
					assignment.TargetID, assignment.MatchID, assignment.MatchID, bracket.UUID)
				if err != nil {
					logrus.WithError(err).WithField("match_id", assignment.MatchID).Error("Failed to update match target")
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update match target"})
					return
				}
				count, _ := res.RowsAffected()
				updated += int(count)
			} else if assignment.BoardNumber > 0 {
				_, err := tx.Exec(`REPLACE INTO tournament_target_boards (uuid, tournament_uuid, bracket_uuid, board_number, target_uuid, code, is_active, updated_at)
					VALUES (UUID(), ?, ?, ?, ?, ?, 1, NOW())`,
					bracket.EventUUID, bracket.UUID, assignment.BoardNumber, assignment.TargetID, assignment.Code)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign target to board"})
					return
				}
				updated++
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Successfully updated %d targets", updated),
			"updated": updated,
		})
	}
}

// GetBracketTeamMembers returns all team members for every entry in a bracket,
// grouped by elimination entry UUID. Used by the scoring UI to display archer names.
func GetBracketTeamMembers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		var bracketUUID string
		if err := db.Get(&bracketUUID, `SELECT uuid FROM elimination_brackets WHERE bracket_id = ? OR uuid = ?`, bracketID, bracketID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		type MemberRow struct {
			EntryUUID   string  `db:"entry_uuid"`
			FullName    string  `db:"full_name"`
			AvatarURL   *string `db:"avatar_url"`
			MemberOrder int     `db:"member_order"`
		}
		var rows []MemberRow
		db.Select(&rows, `
			SELECT
				ee.uuid as entry_uuid,
				COALESCE(a.full_name, '') as full_name,
				a.avatar_url,
				tm.member_order
			FROM elimination_entries ee
			JOIN teams t ON ee.participant_uuid = t.uuid AND ee.participant_type = 'team'
			JOIN team_members tm ON tm.team_id = t.uuid
			JOIN tournament_participants ep ON tm.participant_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE ee.bracket_uuid = ?
			ORDER BY ee.uuid, tm.member_order ASC
		`, bracketUUID)

		result := make(map[string][]gin.H)
		for _, row := range rows {
			member := gin.H{
				"full_name":    row.FullName,
				"member_order": row.MemberOrder,
				"avatar_url":   nil,
			}
			if row.AvatarURL != nil {
				masked := utils.MaskMediaURL(*row.AvatarURL)
				member["avatar_url"] = masked
			}
			result[row.EntryUUID] = append(result[row.EntryUUID], member)
		}

		c.JSON(http.StatusOK, gin.H{"members": result})
	}
}

// AutoAssignMatchTargets automatically assigns available targets to matches in a specific round
func AutoAssignMatchTargets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")
		roundNoStr := c.Query("round")
		if roundNoStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "round parameter is required"})
			return
		}

		roundNo, err := strconv.Atoi(roundNoStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid round number"})
			return
		}

		// Resolve bracket UUID and event UUID
		var bracket struct {
			UUID      string `db:"uuid"`
			EventUUID string `db:"event_uuid"`
		}
		err = db.Get(&bracket, `SELECT uuid, tournament_uuid as event_uuid FROM elimination_brackets WHERE bracket_id = ? OR uuid = ?`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		// Get active matches in this round that are not BYEs
		type MatchInfo struct {
			UUID    string `db:"uuid"`
			MatchNo int    `db:"match_no"`
		}
		var matches []MatchInfo
		err = db.Select(&matches, `
			SELECT uuid, match_no 
			FROM elimination_matches 
			WHERE bracket_uuid = ? AND round_no = ? AND is_bye = 0 
			ORDER BY match_no ASC`,
			bracket.UUID, roundNo)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch match data", "details": err.Error()})
			return
		}

		if len(matches) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"message": fmt.Sprintf("No active (non-bye) matches found in round %d", roundNo),
				"updated": 0,
			})
			return
		}

		// Get available targets for the event
		type TargetInfo struct {
			UUID string `db:"uuid"`
		}
		var targets []TargetInfo
		err = db.Select(&targets, `
			SELECT uuid 
			FROM tournament_targets 
			WHERE tournament_uuid = ? 
			ORDER BY board_number ASC, target_name ASC`,
			bracket.EventUUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch targets data", "details": err.Error()})
			return
		}

		if len(targets) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No target butts available for this event. Please create targets in the Targets menu first."})
			return
		}

		// Update matches with targets
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		updated := 0
		for i, match := range matches {
			if i >= len(targets) {
				break // Not enough targets
			}

			res, err := tx.Exec(`UPDATE elimination_matches SET target_uuid = ?, updated_at = NOW() WHERE uuid = ?`,
				targets[i].UUID, match.UUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update match target", "details": err.Error()})
				return
			}
			count, _ := res.RowsAffected()
			updated += int(count)
		}

		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
			return
		}

		logrus.Infof("AutoAssignMatchTargets: bracket=%s, round=%d, matches_found=%d, targets_found=%d, updated=%d", bracket.UUID, roundNo, len(matches), len(targets), updated)

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Successfully auto-assigned %d matches out of %d matches in round %d", updated, len(matches), roundNo),
			"updated": updated,
			"total":   len(matches),
			"targets": len(targets),
		})
	}
}

// ============= MATCH SCORING =============

// GetMatch returns a match with its scoring details
func GetMatch(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		type Match struct {
			UUID            string     `json:"uuid" db:"uuid"`
			MatchID         string     `json:"id" db:"match_id"`
			BracketUUID     string     `json:"bracket_id" db:"bracket_uuid"`
			RoundNo         int        `json:"round_no" db:"round_no"`
			MatchNo         int        `json:"match_no" db:"match_no"`
			EntryAUUID      *string    `json:"entry_a_id" db:"entry_a_uuid"`
			EntryBUUID      *string    `json:"entry_b_id" db:"entry_b_uuid"`
			EntryAName      *string    `json:"entry_a_name" db:"entry_a_name"`
			EntryBName      *string    `json:"entry_b_name" db:"entry_b_name"`
			EntryASeed      *int       `json:"entry_a_seed" db:"entry_a_seed"`
			EntryBSeed      *int       `json:"entry_b_seed" db:"entry_b_seed"`
			WinnerEntryUUID *string    `json:"winner_entry_id" db:"winner_entry_uuid"`
			Status          string     `json:"status" db:"status"`
			IsBye           bool       `json:"is_bye" db:"is_bye"`
			ScheduledAt     *time.Time `json:"scheduled_at" db:"scheduled_at"`
			TargetUUID      *string    `json:"target_id" db:"target_uuid"`
			Format          string     `json:"format" db:"format"`
			ArrowsPerEnd    int        `json:"arrows_per_end" db:"arrows_per_end"`
			EndsPerMatch    int        `json:"ends_per_match" db:"ends_per_match"`
			TotalScoreA     int        `json:"total_score_a" db:"total_score_a"`
			TotalScoreB     int        `json:"total_score_b" db:"total_score_b"`
			TotalPointsA    int        `json:"total_points_a" db:"total_points_a"`
			TotalPointsB    int        `json:"total_points_b" db:"total_points_b"`
			ShootOffA       *string    `json:"shoot_off_a"`
			ShootOffB       *string    `json:"shoot_off_b"`
			CreatedAt       *time.Time `json:"created_at" db:"created_at"`
		}

		var match Match
		err := db.Unsafe().Get(&match, `
			SELECT 
				em.uuid,
				COALESCE(NULLIF(em.match_id, ''), em.uuid) as match_id,
				em.bracket_uuid,
				em.round_no,
				em.match_no,
				em.entry_a_uuid,
				em.entry_b_uuid,
				em.winner_entry_uuid,
				COALESCE(em.status, '') as status,
				COALESCE(em.is_bye, false) as is_bye,
				em.scheduled_at,
				em.target_uuid,
				COALESCE(em.total_score_a, 0) as total_score_a,
				COALESCE(em.total_score_b, 0) as total_score_b,
				COALESCE(em.total_points_a, 0) as total_points_a,
				COALESCE(em.total_points_b, 0) as total_points_b,
				em.created_at,
				eb.format, 
				eb.arrows_per_end, 
				eb.ends_per_match,
				COALESCE(aA.full_name, tA.team_name, '') as entry_a_name,
				COALESCE(aB.full_name, tB.team_name, '') as entry_b_name,
				CAST(COALESCE(eeA.seed, 0) AS SIGNED) as entry_a_seed,
				CAST(COALESCE(eeB.seed, 0) AS SIGNED) as entry_b_seed
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN tournament_participants epA ON (eeA.participant_uuid = epA.uuid OR em.entry_a_uuid = epA.uuid)
			LEFT JOIN tournament_participants epB ON (eeB.participant_uuid = epB.uuid OR em.entry_b_uuid = epB.uuid)
			LEFT JOIN archers aA ON epA.archer_id = aA.uuid
			LEFT JOIN archers aB ON epB.archer_id = aB.uuid
			LEFT JOIN teams tA ON eeA.participant_uuid = tA.uuid
			LEFT JOIN teams tB ON eeB.participant_uuid = tB.uuid
			WHERE em.uuid = ? OR em.match_id = ?
		`, matchID, matchID)
		if err != nil {
			logrus.WithError(err).WithField("match_id", matchID).Error("Failed to fetch match details")
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}
		// Use the actual UUID for subsequent queries
		matchID = match.UUID

		// Default arrows_per_end if not set
		arrowsPerEnd := match.ArrowsPerEnd
		if arrowsPerEnd == 0 {
			arrowsPerEnd = 3
		}

		// Map to participant objects for frontend legacy compatibility
		type Participant struct {
			EntryUUID string `json:"entry_id"`
			Name      string `json:"name"`
			Seed      int    `json:"seed"`
		}

		var participantA, participantB *Participant
		if match.EntryAUUID != nil {
			name := "TBD"
			if match.EntryAName != nil {
				name = *match.EntryAName
			}
			seed := 0
			if match.EntryASeed != nil {
				seed = *match.EntryASeed
			}
			participantA = &Participant{
				EntryUUID: *match.EntryAUUID,
				Name:      name,
				Seed:      seed,
			}
		}
		if match.EntryBUUID != nil {
			name := "TBD"
			if match.EntryBName != nil {
				name = *match.EntryBName
			}
			seed := 0
			if match.EntryBSeed != nil {
				seed = *match.EntryBSeed
			}
			participantB = &Participant{
				EntryUUID: *match.EntryBUUID,
				Name:      name,
				Seed:      seed,
			}
		}

		// Get scoring ends
		type EndScore struct {
			UUID     string   `json:"id" db:"uuid"`
			EndNo    int      `json:"end_no" db:"end_no"`
			Side     string   `json:"side" db:"side"`
			EndTotal int      `json:"end_total" db:"end_total"`
			XCount   int      `json:"x_count" db:"x_count"`
			TenCount int      `json:"ten_count" db:"ten_count"`
			Arrows   []string `json:"arrows"`
		}

		var ends []EndScore
		db.Select(&ends, `
			SELECT uuid, end_no, side, end_total, x_count, ten_count
			FROM elimination_match_ends
			WHERE match_uuid = ?
			ORDER BY end_no ASC, side ASC
		`, matchID)

		if ends == nil {
			ends = []EndScore{}
		}

		// Fetch arrows for each end
		for i := range ends {
			var arrowScores []struct {
				Score int  `db:"score"`
				IsX   bool `db:"is_x"`
			}
			db.Select(&arrowScores, `SELECT score, is_x FROM elimination_match_arrow_scores WHERE match_end_uuid = ? ORDER BY arrow_no ASC`, ends[i].UUID)

			if len(arrowScores) > 0 {
				// We have real per-arrow data
				ends[i].Arrows = make([]string, len(arrowScores))
				for j, as := range arrowScores {
					if as.IsX {
						ends[i].Arrows[j] = "X"
					} else if as.Score == 0 {
						ends[i].Arrows[j] = "M"
					} else {
						ends[i].Arrows[j] = fmt.Sprintf("%d", as.Score)
					}
				}
			} else {
				// No per-arrow records: fill with empty placeholders so the
				// frontend knows how many arrow slots to render.
				slotCount := arrowsPerEnd
				if ends[i].EndNo == 99 {
					slotCount = 1 // shoot-off is always 1 arrow
				}
				ends[i].Arrows = make([]string, slotCount)
				for j := range ends[i].Arrows {
					ends[i].Arrows[j] = "" // empty = no data
				}
			}
		}

		// Calculate total scores and points
		var totalScoreA, totalScoreB, totalPointsA, totalPointsB int

		// Map ends by number to calculate set points
		endByNo := make(map[int]map[string]int)
		for _, e := range ends {
			if endByNo[e.EndNo] == nil {
				endByNo[e.EndNo] = make(map[string]int)
			}
			endByNo[e.EndNo][e.Side] = e.EndTotal

			if e.EndNo != 99 { // Don't include shoot-off in totals
				if e.Side == "A" {
					totalScoreA += e.EndTotal
				} else {
					totalScoreB += e.EndTotal
				}
			}
		}

		if match.Format == "recurve_set" {
			for endNo, sides := range endByNo {
				if endNo == 99 {
					continue
				}
				sA := sides["A"]
				sB := sides["B"]
				if sA > sB {
					totalPointsA += 2
				} else if sB > sA {
					totalPointsB += 2
				} else if sA == sB && sA > 0 {
					totalPointsA += 1
					totalPointsB += 1
				}
			}
		}

		// Process shoot-off arrows for struct fields and extra point
		soArrowsA := ""
		soArrowsB := ""
		for _, e := range ends {
			if e.EndNo == 99 {
				if len(e.Arrows) > 0 {
					if e.Side == "A" {
						match.ShootOffA = &e.Arrows[0]
						soArrowsA = e.Arrows[0]
					} else {
						match.ShootOffB = &e.Arrows[0]
						soArrowsB = e.Arrows[0]
					}
				}
			}
		}

		if soArrowsA != "" && soArrowsB != "" {
			getV := func(v string) int {
				if v == "X" {
					return 11
				}
				if v == "M" {
					return 0
				}
				var sc int
				fmt.Sscanf(v, "%d", &sc)
				return sc
			}
			scA := getV(soArrowsA)
			scB := getV(soArrowsB)
			if scA > scB {
				if match.Format == "recurve_set" {
					totalPointsA++
				}
			} else if scB > scA {
				if match.Format == "recurve_set" {
					totalPointsB++
				}
			} else {
				// Tie in shoot-off - Award +1 to the recorded winner for recurve only
				if match.WinnerEntryUUID != nil && match.Status == "finished" {
					if match.EntryAUUID != nil && *match.WinnerEntryUUID == *match.EntryAUUID {
						if match.Format == "recurve_set" {
							totalPointsA++
						}
					} else if match.EntryBUUID != nil && *match.WinnerEntryUUID == *match.EntryBUUID {
						if match.Format == "recurve_set" {
							totalPointsB++
						}
					}
				}
			}
		}

		match.TotalScoreA = totalScoreA
		match.TotalScoreB = totalScoreB
		match.TotalPointsA = totalPointsA
		match.TotalPointsB = totalPointsB

		// Fetch tournament and category details for match context
		type TournamentInfo struct {
			UUID         string     `json:"id" db:"uuid"`
			Name         string     `json:"name" db:"name"`
			Slug         string     `json:"slug" db:"slug"`
			Venue        *string    `json:"venue" db:"venue"`
			Location     *string    `json:"location" db:"location"`
			City         *string    `json:"city" db:"city"`
			StartDate    *time.Time `json:"start_date" db:"start_date"`
			EndDate      *time.Time `json:"end_date" db:"end_date"`
			LogoURL      *string    `json:"logo_url" db:"logo_url"`
			BannerURL    *string    `json:"banner_url" db:"banner_url"`
			CategoryName *string    `json:"category_name" db:"category_name"`
			BracketType  *string    `json:"bracket_type" db:"bracket_type"`
		}

		var eventInfo TournamentInfo
		_ = db.Unsafe().Get(&eventInfo, `
			SELECT 
				t.uuid,
				COALESCE(t.name, '') as name,
				COALESCE(t.slug, '') as slug,
				t.venue,
				t.location,
				t.city,
				t.start_date,
				t.end_date,
				t.logo_url,
				t.banner_url,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' ', COALESCE(rag.name, ''), ' ', COALESCE(rgd.name, ''))) as category_name,
				eb.bracket_type
			FROM elimination_brackets eb
			JOIN tournaments t ON eb.tournament_uuid = t.uuid
			LEFT JOIN tournament_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE eb.uuid = ?
			LIMIT 1
		`, match.BracketUUID)

		c.JSON(http.StatusOK, gin.H{
			"match":         match,
			"event":         eventInfo,
			"participant_a": participantA,
			"participant_b": participantB,
			"ends":          ends,
		})
	}
}

// UpdateMatchScore updates or creates end scores for a match
// UpdateMatchScore godoc
// UpdateMatchScore updates score for a specific end in a match
// @Summary Update Match Score
// @Description Submit or update scores for a specific end (set/end) in an elimination match
// @Tags Mobile - Scorekeeper
// @Accept json
// @Produce json
// @Param matchId path string true "Match UUID"
// @Param request body MatchScoreRequest true "Match Score Payload"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/elimination/scoring/matches/{matchId}/score [post]
func UpdateMatchScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		var req struct {
			EndNo   int      `json:"end_no" binding:"required"`
			ScoreA  int      `json:"score_a"`
			ScoreB  int      `json:"score_b"`
			ArrowsA []string `json:"arrows_a"`
			ArrowsB []string `json:"arrows_b"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve match UUID
		var actualMatchUUID string
		err := db.Get(&actualMatchUUID, `SELECT uuid FROM elimination_matches WHERE uuid = ? OR match_id = ?`, matchID, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}
		matchID = actualMatchUUID

		// Check if bracket is locked
		var isLocked bool
		_ = db.Get(&isLocked, `
			SELECT COALESCE(eb.is_locked, 0) 
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			WHERE em.uuid = ?
		`, matchID)
		if isLocked {
			c.JSON(http.StatusForbidden, gin.H{"error": "Braket eliminasi telah dikunci oleh panitia/wasit. Perubahan skor tidak diperbolehkan."})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// Helper to process side
		processSide := func(side string, total int, arrows []string) error {
			xCount := 0
			tenCount := 0
			if len(arrows) > 0 {
				total = 0
				for _, a := range arrows {
					if a == "X" {
						xCount++
						tenCount++
						total += 10
					} else if a == "10" {
						tenCount++
						total += 10
					} else if a == "M" {
						total += 0
					} else {
						val := 0
						fmt.Sscanf(a, "%d", &val)
						total += val
					}
				}
			}

			// Upsert end
			_, err := tx.Exec(`
				INSERT INTO elimination_match_ends (uuid, match_uuid, end_no, side, end_total, x_count, ten_count)
				VALUES (?, ?, ?, ?, ?, ?, ?)
				ON DUPLICATE KEY UPDATE end_total = VALUES(end_total), x_count = VALUES(x_count), ten_count = VALUES(ten_count)
			`, uuid.New().String(), matchID, req.EndNo, side, total, xCount, tenCount)
			if err != nil {
				return err
			}

			// Get UUID of the end (new or existing)
			var endUUID string
			err = tx.Get(&endUUID, `SELECT uuid FROM elimination_match_ends WHERE match_uuid = ? AND end_no = ? AND side = ?`, matchID, req.EndNo, side)
			if err != nil {
				return err
			}

			// Upsert arrows if provided
			if len(arrows) > 0 {
				tx.Exec(`DELETE FROM elimination_match_arrow_scores WHERE match_end_uuid = ?`, endUUID)
				for i, a := range arrows {
					scoreVal := 0
					isX := false
					if a == "X" {
						scoreVal = 10
						isX = true
					} else if a == "M" {
						scoreVal = 0
					} else {
						fmt.Sscanf(a, "%d", &scoreVal)
					}
					_, err = tx.Exec(`
						INSERT INTO elimination_match_arrow_scores (uuid, match_end_uuid, arrow_no, score, is_x)
						VALUES (?, ?, ?, ?, ?)
					`, uuid.New().String(), endUUID, i+1, scoreVal, isX)
					if err != nil {
						return err
					}
				}
			}
			return nil
		}

		if err := processSide("A", req.ScoreA, req.ArrowsA); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui sisi A", "details": err.Error()})
			return
		}

		if err := processSide("B", req.ScoreB, req.ArrowsB); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui sisi B", "details": err.Error()})
			return
		}
		// Recalculate totals and update elimination_matches
		var m struct {
			Format string `db:"format"`
		}
		if err := tx.Get(&m, `SELECT eb.format FROM elimination_matches em JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid WHERE em.uuid = ?`, matchID); err == nil {
			var allEnds []struct {
				EndNo    int    `db:"end_no"`
				Side     string `db:"side"`
				EndTotal int    `db:"end_total"`
			}
			tx.Select(&allEnds, `SELECT end_no, side, end_total FROM elimination_match_ends WHERE match_uuid = ?`, matchID)

			type arrowScore struct {
				Side  string `db:"side"`
				Score int    `db:"score"`
				IsX   bool   `db:"is_x"`
			}
			var soArrows []arrowScore
			tx.Select(&soArrows, `
				SELECT eme.side, emas.score, emas.is_x
				FROM elimination_match_arrow_scores emas
				JOIN elimination_match_ends eme ON emas.match_end_uuid = eme.uuid
				WHERE eme.match_uuid = ? AND eme.end_no = 99
			`, matchID)

			mEnds := make(map[int]map[string]int)
			for _, e := range allEnds {
				if mEnds[e.EndNo] == nil {
					mEnds[e.EndNo] = make(map[string]int)
				}
				mEnds[e.EndNo][e.Side] = e.EndTotal
			}

			tSA, tSB, tPA, tPB := 0, 0, 0, 0
			for en, sides := range mEnds {
				if en == 99 {
					continue
				}
				sA, sB := sides["A"], sides["B"]
				tSA += sA
				tSB += sB
				if m.Format == "recurve_set" {
					if sA > sB {
						tPA += 2
					} else if sB > sA {
						tPB += 2
					} else if sA == sB && sA > 0 {
						tPA += 1
						tPB += 1
					}
				}
			}

			// Shoot-off +1 logic
			soA, soB := -1, -1
			for _, a := range soArrows {
				val := a.Score
				if a.IsX {
					val = 11
				}
				if a.Side == "A" {
					soA = val
				} else {
					soB = val
				}
			}
			if soA >= 0 && soB >= 0 {
				if soA > soB {
					if m.Format == "recurve_set" {
						tPA++
					}
				} else if soB > soA {
					if m.Format == "recurve_set" {
						tPB++
					}
				}
			}
			_, err = tx.Exec(`UPDATE elimination_matches SET total_score_a=?, total_score_b=?, total_points_a=?, total_points_b=? WHERE uuid=?`, tSA, tSB, tPA, tPB, matchID)
			if err != nil {
				logrus.WithError(err).Error("Gagal memperbarui ringkasan skor pertandingan")
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		// Log for scorekeeper audit
		userTypeContext, _ := c.Get("user_type")
		if userTypeContext == "scorekeeper" {
			userID, _ := c.Get("user_id")
			orgID, _ := c.Get("org_id")
			details, _ := json.Marshal(req)

			var eventUUID string
			_ = db.Get(&eventUUID, "SELECT eb.tournament_uuid FROM elimination_matches em JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid WHERE em.uuid = ?", matchID)

			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "update_match_score", string(details), c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Skor berhasil diperbarui"})
	}
}

// FinishMatch marks a match as finished and advances winner to next round
// FinishMatch godoc
// FinishMatch marks a match as finished with a winner
// @Summary Finish Match
// @Description Finish an elimination match and declare the winner
// @Tags Mobile - Scorekeeper
// @Accept json
// @Produce json
// @Param matchId path string true "Match UUID"
// @Param request body FinishMatchRequest true "Finish Match Payload"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/elimination/scoring/matches/{matchId}/finish [post]
func FinishMatch(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		var req struct {
			WinnerEntryID string `json:"winner_entry_id" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve match UUID
		var actualMatchUUID string
		err := db.Get(&actualMatchUUID, `SELECT uuid FROM elimination_matches WHERE uuid = ? OR match_id = ?`, matchID, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}
		matchID = actualMatchUUID

		// Get match info
		type MatchInfo struct {
			UUID        string  `db:"uuid"`
			BracketUUID string  `db:"bracket_uuid"`
			RoundNo     int     `db:"round_no"`
			MatchNo     int     `db:"match_no"`
			EntryAUUID  *string `db:"entry_a_uuid"`
			EntryBUUID  *string `db:"entry_b_uuid"`
			Status      string  `db:"status"`
		}

		var match MatchInfo
		err = db.Get(&match, `SELECT uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, status FROM elimination_matches WHERE uuid = ?`, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		// Check if match is already finished
		if match.Status == "finished" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Match is already finished"})
			return
		}

		// Validate winner is one of the participants
		if (match.EntryAUUID == nil || *match.EntryAUUID != req.WinnerEntryID) &&
			(match.EntryBUUID == nil || *match.EntryBUUID != req.WinnerEntryID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Winner must be one of the match participants"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// Update match with winner and status
		_, err = tx.Exec(`UPDATE elimination_matches SET winner_entry_uuid = ?, status = 'finished' WHERE uuid = ?`, req.WinnerEntryID, matchID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update match"})
			return
		}

		// Get bracket info for round calculation
		var bracketSize int
		tx.Get(&bracketSize, `SELECT bracket_size FROM elimination_brackets WHERE uuid = ?`, match.BracketUUID)

		numRounds := int(math.Log2(float64(bracketSize)))

		// Only advance if not the final round
		if match.RoundNo < numRounds {
			// Calculate relative match number in CURRENT round
			offset := getMatchNoOffset(bracketSize, match.RoundNo)
			relativeMatchNo := match.MatchNo - offset

			// Calculate next match position
			relativeNextMatchNo := (relativeMatchNo + 1) / 2
			nextRound := match.RoundNo + 1
			nextOffset := getMatchNoOffset(bracketSize, nextRound)
			globalNextMatchNo := nextOffset + relativeNextMatchNo

			// Determine if winner goes to slot A or B (odd relative match = A, even relative match = B)
			slot := "entry_a_uuid"
			if relativeMatchNo%2 == 0 {
				slot = "entry_b_uuid"
			}

			// Check if the next round match exists
			var nextMatchExists int
			tx.Get(&nextMatchExists, `SELECT COUNT(*) FROM elimination_matches WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`,
				match.BracketUUID, nextRound, globalNextMatchNo)

			if nextMatchExists > 0 {
				// Update existing next round match
				_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, slot),
					req.WinnerEntryID, match.BracketUUID, nextRound, globalNextMatchNo)
				if err != nil {
					logrus.WithError(err).Error("Failed to advance winner to existing match")
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to advance winner"})
					return
				}
			}
		}

		// SPECIAL CASE: Advance Losers to Bronze Match if it's the Semifinals
		if match.RoundNo == numRounds-1 {
			loserID := match.EntryAUUID
			if loserID != nil && *loserID == req.WinnerEntryID {
				loserID = match.EntryBUUID
			}

			if loserID != nil {
				bronzeSlot := "entry_a_uuid"
				offset := getMatchNoOffset(bracketSize, match.RoundNo)
				relativeMatchNo := match.MatchNo - offset
				if relativeMatchNo%2 == 0 {
					bronzeSlot = "entry_b_uuid"
				}
				// Bronze match is always the second match of the final tournament sequence (Finals is offset+1, Bronze is offset+2)
				finalRoundOffset := getMatchNoOffset(bracketSize, numRounds)
				globalBronzeMatchNo := finalRoundOffset + 2

				_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, bronzeSlot),
					*loserID, match.BracketUUID, numRounds, globalBronzeMatchNo)
				if err != nil {
					logrus.WithError(err).Error("Failed to advance loser to bronze match")
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		// Log for scorekeeper audit
		userTypeContext, _ := c.Get("user_type")
		if userTypeContext == "scorekeeper" {
			userID, _ := c.Get("user_id")
			orgID, _ := c.Get("org_id")
			details, _ := json.Marshal(req)

			var eventUUID string
			_ = db.Get(&eventUUID, "SELECT eb.tournament_uuid FROM elimination_matches em JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid WHERE em.uuid = ?", matchID)

			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "finish_match", string(details), c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{
			"message":         "Pertandingan selesai dan pemenang lanjut ke babak berikutnya",
			"winner_entry_id": req.WinnerEntryID,
			"match_status":    "finished",
		})
	}
}

// EndMatch calculates winner from scores and finishes the match
// EndMatch godoc
// EndMatch ends the match and auto-determines the winner
// @Summary End Match
// @Description End the match, calculate final scores/points, and advance the winner
// @Tags Mobile - Scorekeeper
// @Accept json
// @Produce json
// @Param matchId path string true "Match UUID"
// @Param request body object false "Optional Winner Override"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/elimination/scoring/matches/{matchId}/end [post]
func EndMatch(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		var req struct {
			WinnerEntryID string `json:"winner_entry_id"`
		}
		c.ShouldBindJSON(&req)

		// Resolve match UUID
		var actualMatchUUID string
		err := db.Get(&actualMatchUUID, `SELECT uuid FROM elimination_matches WHERE uuid = ? OR match_id = ?`, matchID, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}
		matchID = actualMatchUUID

		// Get match info including bracket format
		type MatchInfo struct {
			UUID         string  `db:"uuid"`
			BracketUUID  string  `db:"bracket_uuid"`
			RoundNo      int     `db:"round_no"`
			MatchNo      int     `db:"match_no"`
			EntryAUUID   *string `db:"entry_a_uuid"`
			EntryBUUID   *string `db:"entry_b_uuid"`
			Status       string  `db:"status"`
			Format       string  `db:"format"`
			TotalScoreA  int     `db:"total_score_a"`
			TotalScoreB  int     `db:"total_score_b"`
			TotalPointsA int     `db:"total_points_a"`
			TotalPointsB int     `db:"total_points_b"`
		}

		var match MatchInfo
		err = db.Get(&match, `
			SELECT em.uuid, em.bracket_uuid, em.round_no, em.match_no, 
					em.entry_a_uuid, em.entry_b_uuid, em.status, eb.format,
					COALESCE(em.total_score_a, 0) as total_score_a,
					COALESCE(em.total_score_b, 0) as total_score_b,
					COALESCE(em.total_points_a, 0) as total_points_a,
					COALESCE(em.total_points_b, 0) as total_points_b
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			WHERE em.uuid = ?`, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		if match.Status == "finished" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Match is already finished"})
			return
		}

		// Calculate scores from regular ends (excluding shoot-off end 99)
		type EndScore struct {
			Side     string `db:"side"`
			TotalEnd int    `db:"total_end"`
		}
		var ends []EndScore
		err = db.Select(&ends, `
			SELECT side, COALESCE(SUM(end_total), 0) as total_end 
			FROM elimination_match_ends 
			WHERE match_uuid = ? AND end_no != 99
			GROUP BY side`, matchID)
		if err != nil {
			logrus.WithError(err).Error("Failed to fetch end scores")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung skor"})
			return
		}

		var totalScoreA, totalScoreB int
		for _, e := range ends {
			if e.Side == "A" {
				totalScoreA = e.TotalEnd
			} else if e.Side == "B" {
				totalScoreB = e.TotalEnd
			}
		}

		// For set system (recurve), calculate set points
		var totalPointsA, totalPointsB int
		if match.Format == "recurve_set" {
			type SetEnd struct {
				EndNo int    `db:"end_no"`
				Side  string `db:"side"`
				Total int    `db:"end_total"`
			}
			var setEnds []SetEnd
			db.Select(&setEnds, `
				SELECT end_no, side, end_total 
				FROM elimination_match_ends 
				WHERE match_uuid = ? AND end_no != 99
				ORDER BY end_no, side`, matchID)

			// Group by end_no
			endTotals := make(map[int]map[string]int)
			for _, se := range setEnds {
				if endTotals[se.EndNo] == nil {
					endTotals[se.EndNo] = make(map[string]int)
				}
				endTotals[se.EndNo][se.Side] = se.Total
			}

			// Calculate set points
			for _, sides := range endTotals {
				scoreA := sides["A"]
				scoreB := sides["B"]
				if scoreA > scoreB {
					totalPointsA += 2
				} else if scoreB > scoreA {
					totalPointsB += 2
				} else if scoreA == scoreB && scoreA > 0 {
					totalPointsA += 1
					totalPointsB += 1
				}
			}
		}

		// Determine winner
		comparisonA := totalScoreA
		comparisonB := totalScoreB
		if match.Format == "recurve_set" {
			comparisonA = totalPointsA
			comparisonB = totalPointsB
		}

		var winnerID string
		if comparisonA > comparisonB {
			if match.EntryAUUID != nil {
				winnerID = *match.EntryAUUID
			}
		} else if comparisonB > comparisonA {
			if match.EntryBUUID != nil {
				winnerID = *match.EntryBUUID
			}
		} else {
			// Tie - Check for Shoot-off arrows in arrow_scores or ends
			type arrowScore struct {
				Side  string `db:"side"`
				Score int    `db:"score"`
				IsX   bool   `db:"is_x"`
			}
			var soArrows []arrowScore
			db.Select(&soArrows, `
				SELECT eme.side, emas.score, emas.is_x
				FROM elimination_match_arrow_scores emas
				JOIN elimination_match_ends eme ON emas.match_end_uuid = eme.uuid
				WHERE eme.match_uuid = ? AND eme.end_no = 99
			`, matchID)

			soA, soB := -1, -1
			for _, a := range soArrows {
				val := a.Score
				if a.IsX {
					val = 11
				}
				if a.Side == "A" {
					soA = val
				} else {
					soB = val
				}
			}

			if soA < 0 || soB < 0 {
				// Fallback to end_total if arrow scores not found
				type shootOffEnd struct {
					Side     string `db:"side"`
					EndTotal int    `db:"end_total"`
				}
				var soEnds []shootOffEnd
				db.Select(&soEnds, `SELECT side, end_total FROM elimination_match_ends WHERE match_uuid = ? AND end_no = 99`, matchID)
				for _, e := range soEnds {
					if e.Side == "A" {
						soA = e.EndTotal
					} else if e.Side == "B" {
						soB = e.EndTotal
					}
				}
			}

			if soA > soB {
				if match.EntryAUUID != nil {
					winnerID = *match.EntryAUUID
					if match.Format == "recurve_set" {
						totalPointsA++
					}
				}
			} else if soB > soA {
				if match.EntryBUUID != nil {
					winnerID = *match.EntryBUUID
					if match.Format == "recurve_set" {
						totalPointsB++
					}
				}
			} else if req.WinnerEntryID != "" {
				// Final manual override if provided in request
				winnerID = req.WinnerEntryID
				if match.Format == "recurve_set" {
					if match.EntryAUUID != nil && winnerID == *match.EntryAUUID {
						totalPointsA++
					} else if match.EntryBUUID != nil && winnerID == *match.EntryBUUID {
						totalPointsB++
					}
				}
			} else {
				// Tie - require shoot-off or manual winner
				c.JSON(http.StatusBadRequest, gin.H{"error": "Match is tied. Please proceed to Shoot-off or select a winner manually."})
				return
			}
		}

		if winnerID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot determine match winner"})
			return
		}

		// Start transaction
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// Update match with winner, status AND final scores
		_, err = tx.Exec(`UPDATE elimination_matches SET winner_entry_uuid = ?, status = 'finished', total_score_a = ?, total_score_b = ?, total_points_a = ?, total_points_b = ? WHERE uuid = ?`,
			winnerID, totalScoreA, totalScoreB, totalPointsA, totalPointsB, matchID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update match result"})
			return
		}

		// Advance winner to next round (reuse logic from FinishMatch)
		var bracketSize int
		tx.Get(&bracketSize, `SELECT bracket_size FROM elimination_brackets WHERE uuid = ?`, match.BracketUUID)

		numRounds := int(math.Log2(float64(bracketSize)))
		if match.RoundNo < numRounds {
			offset := getMatchNoOffset(bracketSize, match.RoundNo)
			relativeMatchNo := match.MatchNo - offset

			relativeNextMatchNo := (relativeMatchNo + 1) / 2
			nextRound := match.RoundNo + 1
			nextOffset := getMatchNoOffset(bracketSize, nextRound)
			globalNextMatchNo := nextOffset + relativeNextMatchNo

			slot := "entry_a_uuid"
			if relativeMatchNo%2 == 0 {
				slot = "entry_b_uuid"
			}

			var nextMatchExists int
			tx.Get(&nextMatchExists, `SELECT COUNT(*) FROM elimination_matches WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`,
				match.BracketUUID, nextRound, globalNextMatchNo)

			if nextMatchExists > 0 {
				_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, slot),
					winnerID, match.BracketUUID, nextRound, globalNextMatchNo)
				if err != nil {
					logrus.WithError(err).Error("Failed to advance winner")
				}
			} else {
				// Check pair match
				var pairMatchNo int
				if match.MatchNo%2 == 1 {
					pairMatchNo = match.MatchNo + 1
				} else {
					pairMatchNo = match.MatchNo - 1
				}

				var pairMatch struct {
					Status          string  `db:"status"`
					WinnerEntryUUID *string `db:"winner_entry_uuid"`
				}
				pairErr := tx.Get(&pairMatch, `SELECT status, winner_entry_uuid FROM elimination_matches WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`,
					match.BracketUUID, match.RoundNo, pairMatchNo)

				if pairErr == nil && pairMatch.Status == "finished" && pairMatch.WinnerEntryUUID != nil {
					var entryA, entryB string
					if match.MatchNo%2 == 1 {
						entryA = winnerID
						entryB = *pairMatch.WinnerEntryUUID
					} else {
						entryA = *pairMatch.WinnerEntryUUID
						entryB = winnerID
					}

					nextMatchUUID := uuid.New().String()
					tx.Exec(`INSERT INTO elimination_matches (uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, status) VALUES (?, ?, ?, ?, ?, ?, 'pending')`,
						nextMatchUUID, match.BracketUUID, nextRound, globalNextMatchNo, entryA, entryB)

					logrus.WithFields(logrus.Fields{
						"next_match_uuid": nextMatchUUID,
						"round":           nextRound,
						"match_no":        globalNextMatchNo,
					}).Info("Created next round match from EndMatch")
				}
			}

			// SPECIAL CASE: Advance Losers to Bronze Match if it's the Semifinals
			if match.RoundNo == numRounds-1 {
				loserID := match.EntryAUUID
				if loserID != nil && *loserID == winnerID {
					loserID = match.EntryBUUID
				}

				if loserID != nil {
					bronzeSlot := "entry_a_uuid"
					offset := getMatchNoOffset(bracketSize, match.RoundNo)
					relativeMatchNo := match.MatchNo - offset
					if relativeMatchNo%2 == 0 {
						bronzeSlot = "entry_b_uuid"
					}
					finalRoundOffset := getMatchNoOffset(bracketSize, numRounds)
					globalBronzeMatchNo := finalRoundOffset + 2
					_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, bronzeSlot),
						*loserID, match.BracketUUID, numRounds, globalBronzeMatchNo)
					if err != nil {
						logrus.WithError(err).Error("Failed to advance loser to bronze match in EndMatch")
					}
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save data"})
			return
		}

		// Get winner name for response
		var winnerName string
		db.Get(&winnerName, `
			SELECT COALESCE(a.full_name, t.team_name, 'Unknown') 
			FROM elimination_entries ee
			LEFT JOIN archers a ON ee.archer_uuid = a.uuid
			LEFT JOIN teams t ON ee.team_uuid = t.uuid
			WHERE ee.uuid = ?`, winnerID)

		// Log for scorekeeper audit
		userTypeContext, _ := c.Get("user_type")
		if userTypeContext == "scorekeeper" {
			userID, _ := c.Get("user_id")
			orgID, _ := c.Get("org_id")
			details, _ := json.Marshal(req)

			var eventUUID string
			_ = db.Get(&eventUUID, "SELECT eb.tournament_uuid FROM elimination_matches em JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid WHERE em.uuid = ?", matchID)

			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "end_match", string(details), c.ClientIP(), c.Request.UserAgent())
		}

		respScoreA := totalScoreA
		respScoreB := totalScoreB
		if match.Format == "recurve_set" {
			respScoreA = totalPointsA
			respScoreB = totalPointsB
		}

		c.JSON(http.StatusOK, gin.H{
			"message":         "Match finished",
			"winner_entry_id": winnerID,
			"winner_name":     winnerName,
			"score_a":         respScoreA,
			"score_b":         respScoreB,
			"match_status":    "finished",
		})
	}
}

// StartBracket changes bracket status to running
func StartBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		result, err := db.Exec(`
			UPDATE elimination_brackets
			SET status = 'running'
			WHERE (bracket_id = ? OR uuid = ?) AND status = 'generated'
		`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start bracket"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bracket not found or not yet generated"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Bracket started"})
	}
}

// ResetMatch resets a finished match back to live/pending
// @Summary Reset Match
// @Description Reset a finished match status and clear the winner from next round
// @Tags Mobile - Scorekeeper
// @Produce json
// @Param matchId path string true "Match UUID"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/elimination/scoring/matches/{matchId}/reset [post]
func ResetMatch(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		// Resolve match UUID
		var actualMatchUUID string
		err := db.Get(&actualMatchUUID, `SELECT uuid FROM elimination_matches WHERE uuid = ? OR match_id = ?`, matchID, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}
		matchID = actualMatchUUID

		// Get match info
		type MatchInfo struct {
			UUID            string  `db:"uuid"`
			BracketUUID     string  `db:"bracket_uuid"`
			RoundNo         int     `db:"round_no"`
			MatchNo         int     `db:"match_no"`
			WinnerEntryUUID *string `db:"winner_entry_uuid"`
			Status          string  `db:"status"`
			EntryAUUID      *string `db:"entry_a_uuid"`
			EntryBUUID      *string `db:"entry_b_uuid"`
		}

		var match MatchInfo
		err = db.Get(&match, `SELECT uuid, bracket_uuid, round_no, match_no, winner_entry_uuid, status, entry_a_uuid, entry_b_uuid FROM elimination_matches WHERE uuid = ?`, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// 1. Reset current match
		_, err = tx.Exec(`UPDATE elimination_matches SET winner_entry_uuid = NULL, status = 'in_progress' WHERE uuid = ?`, matchID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset match status"})
			return
		}

		// 2. Remove entries from next round match
		var bracketSize int
		tx.Get(&bracketSize, `SELECT bracket_size FROM elimination_brackets WHERE uuid = ?`, match.BracketUUID)
		numRounds := int(math.Log2(float64(bracketSize)))

		if match.RoundNo < numRounds {
			nextMatchNo := (match.MatchNo + 1) / 2
			nextRound := match.RoundNo + 1

			slot := "entry_a_uuid"
			if match.MatchNo%2 == 0 {
				slot = "entry_b_uuid"
			}

			// Clear the slot in the next round match
			_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = NULL WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, slot),
				match.BracketUUID, nextRound, nextMatchNo)
			if err != nil {
				logrus.WithError(err).Error("Failed to clear winner from next round match")
			}

			// SPECIAL CASE: Remove loser from Bronze Match if it's the Semifinals
			if match.RoundNo == numRounds-1 {
				bronzeSlot := "entry_a_uuid"
				if match.MatchNo%2 == 0 {
					bronzeSlot = "entry_b_uuid"
				}
				_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = NULL WHERE bracket_uuid = ? AND round_no = ? AND match_no = 2`, bronzeSlot),
					match.BracketUUID, numRounds)
				if err != nil {
					logrus.WithError(err).Error("Failed to clear loser from bronze match")
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		// Log for scorekeeper audit
		userTypeContext, _ := c.Get("user_type")
		if userTypeContext == "scorekeeper" {
			userID, _ := c.Get("user_id")
			orgID, _ := c.Get("org_id")

			var eventUUID string
			_ = db.Get(&eventUUID, "SELECT eb.tournament_uuid FROM elimination_matches em JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid WHERE em.uuid = ?", matchID)

			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "reset_match", "Match reset to in_progress", c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Match status has been reset. You can now edit the score.",
			"status":  "in_progress",
		})
	}
}

// GetEliminationBoardCodes returns all generated codes for target boards in an elimination bracket
func GetEliminationBoardCodes(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")
		if bracketID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bracketId is required"})
			return
		}

		// Resolve bracket UUID and category UUID
		var bracket struct {
			UUID         string `db:"uuid"`
			EventUUID    string `db:"event_uuid"`
			CategoryUUID string `db:"category_uuid"`
		}
		err := db.Get(&bracket, `SELECT uuid, tournament_uuid as event_uuid, category_uuid FROM elimination_brackets WHERE bracket_id = ? OR uuid = ?`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		// Identify all unique board numbers currently in use for this bracket
		var boardNumbers []int
		err = db.Select(&boardNumbers, `
			SELECT DISTINCT et.board_number 
			FROM elimination_matches em 
			JOIN tournament_targets et ON em.target_uuid = et.uuid 
			WHERE em.bracket_uuid = ? AND et.board_number > 0
			ORDER BY et.board_number ASC
		`, bracket.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to identify active target boards", "details": err.Error()})
			return
		}

		// Get or generate the "event part" suffix (3 letters) for this tournament
		var suffix string
		db.Get(&suffix, `
			SELECT RIGHT(code, 3) 
			FROM target_board_elimination tbe
			JOIN elimination_brackets eb ON tbe.bracket_uuid = eb.uuid
			WHERE eb.tournament_uuid = ? LIMIT 1
		`, bracket.EventUUID)

		if suffix == "" {
			const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ"
			b := make([]byte, 3)
			for j := range b {
				b[j] = charset[rand.Intn(len(charset))]
			}
			suffix = string(b)
		}

		// Ensure codes exist for all these board numbers for this bracket
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed", "details": err.Error()})
			return
		}
		defer tx.Rollback()

		for _, bn := range boardNumbers {
			var exists bool
			err := tx.Get(&exists, "SELECT EXISTS(SELECT 1 FROM `target_board_elimination` WHERE `bracket_uuid` = ? AND `board_number` = ?)", bracket.UUID, bn)
			if err != nil {
				continue
			}
			if !exists {
				code := fmt.Sprintf("%03d%s", bn, suffix)
				_, err = tx.Exec("INSERT INTO `target_board_elimination` (`uuid`, `bracket_uuid`, `category_uuid`, `board_number`, `code`) VALUES (?, ?, ?, ?, ?)",
					uuid.New().String(), bracket.UUID, bracket.CategoryUUID, bn, code)
				if err != nil {
					continue
				}
			}
		}
		tx.Commit()

		// Fetch all codes for this bracket
		var codes []models.TargetBoardElimination
		err = db.Select(&codes, "SELECT `uuid`, `bracket_uuid`, `category_uuid`, `board_number`, `code`, `created_at` FROM `target_board_elimination` WHERE `bracket_uuid` = ?", bracket.UUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve board codes", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"board_codes": codes})
	}
}

// â”€â”€â”€ Elimination Scoresheet â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

// ElimMatchCard holds data for one match panel on the scoresheet
type ElimMatchCard struct {
	RoundNo    int
	MatchNo    int
	RoundLabel string
	TargetName string
	BoardCode  string
	QRDataURI  string // base64â€‘encoded PNG data URI for QR code
	NameA      string
	ClubA      string
	SeedA      int
	NameB      string
	ClubB      string
	SeedB      int
	ArrowRange []int
	EndRange   []int
	IsSet      bool // true = recurve_set (show set columns)
}

// ElimScoresheetPage holds up to 2 match cards per printed page
type ElimScoresheetPage struct {
	Upper *ElimMatchCard
	Lower *ElimMatchCard
}

// ElimScoresheetData is the top-level template data for the elimination scoresheet
type ElimScoresheetData struct {
	EventName    string
	EventOrg     string
	Location     string
	EventDates   string
	CategoryName string
	Format       string
	BracketType  string
	EndsPerMatch int
	ArrowsPerEnd int
	PrintDate    string
	Matches      []ElimMatchCard // one entry per match, ordered by target board number
	Pages        []ElimScoresheetPage
}

// GetEliminationScoresheet generates a printable HTML scoresheet for an elimination bracket.
// Route: GET /api/v1/tournaments/:id/elimination/brackets/:bracketId/scoresheet
func GetEliminationScoresheet(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		ev, err := fetchPrintEvent(db, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		bracketID := c.Param("bracketId")
		if bracketID == "" {
			bracketID = c.Param("bracket_id")
		}
		if bracketID == "" {
			bracketID = c.Query("bracket_id")
		}
		if bracketID == "" {
			bracketID = c.Query("bracketId")
		}
		if bracketID == "" {
			catID := c.Query("category_id")
			if catID != "" {
				_ = db.Get(&bracketID, `SELECT uuid FROM elimination_brackets WHERE tournament_uuid = ? AND category_uuid = ? LIMIT 1`, ev.UUID, catID)
			}
			if bracketID == "" {
				_ = db.Get(&bracketID, `SELECT uuid FROM elimination_brackets WHERE tournament_uuid = ? ORDER BY created_at ASC LIMIT 1`, ev.UUID)
			}
		}

		// Fetch bracket
		type BracketInfo struct {
			UUID         string `db:"uuid"`
			BracketID    string `db:"bracket_id"`
			CategoryUUID string `db:"category_uuid"`
			Format       string `db:"format"`
			BracketType  string `db:"bracket_type"`
			BracketSize  int    `db:"bracket_size"`
			EndsPerMatch int    `db:"ends_per_match"`
			ArrowsPerEnd int    `db:"arrows_per_end"`
			CategoryName string `db:"category_name"`
		}
		var bracket BracketInfo
		if bracketID == "" && c.Query("blank") == "1" {
			bracket = BracketInfo{
				UUID:         "blank",
				BracketID:    "BLANK",
				Format:       "recurve_set",
				BracketType:  "individual",
				BracketSize:  16,
				EndsPerMatch: 5,
				ArrowsPerEnd: 3,
				CategoryName: "Individual Recurve",
			}
		} else {
			err = db.Get(&bracket, `
				SELECT eb.uuid, COALESCE(eb.bracket_id, eb.uuid) AS bracket_id, eb.category_uuid, eb.format, eb.bracket_type, eb.bracket_size,
				       COALESCE(eb.ends_per_match, 5) AS ends_per_match,
				       COALESCE(eb.arrows_per_end, 3) AS arrows_per_end,
				       COALESCE(
				           NULLIF(ec.category_name_custom,''),
				           CONCAT_WS(' ', rbt.name, rag.name, rgd.name)
				       ) AS category_name
				FROM elimination_brackets eb
				LEFT JOIN tournament_categories ec ON eb.category_uuid = ec.uuid
				LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
				LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
				LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
				WHERE eb.tournament_uuid = ? AND (eb.uuid = ? OR eb.bracket_id = ?)`,
				ev.UUID, bracketID, bracketID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": "Bracket tidak ditemukan"})
				return
			}
		}

		// If bracket is a team/mixed-team bracket, delegate to team scoresheet handler
		if bracket.BracketType != "individual" {
			GetTeamEliminationScoresheetPrintout(db)(c)
			return
		}

		loc := ev.Venue.String
		if ev.City.Valid && ev.City.String != "" {
			if loc != "" {
				loc += ", "
			}
			loc += ev.City.String
		}
		dateStr := formatPrintDateRange(ev.StartDate, ev.EndDate)

		// Auto-generate board codes
		{
			var suffix string
			_ = db.Get(&suffix, `SELECT RIGHT(code, 3) FROM target_board_elimination WHERE bracket_uuid = ? LIMIT 1`, bracket.UUID)
			if suffix == "" {
				const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ"
				buf := make([]byte, 3)
				for j := range buf {
					buf[j] = charset[rand.Intn(len(charset))]
				}
				suffix = string(buf)
			}
			var boardNums []int
			_ = db.Select(&boardNums, `
				SELECT DISTINCT et.board_number
				FROM elimination_matches em
				JOIN tournament_targets et ON em.target_uuid = et.uuid
				WHERE em.bracket_uuid = ? AND et.board_number > 0
				ORDER BY et.board_number ASC`, bracket.UUID)
			for _, bn := range boardNums {
				var exists bool
				_ = db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM target_board_elimination WHERE bracket_uuid = ? AND board_number = ?)", bracket.UUID, bn)
				if !exists {
					code := fmt.Sprintf("%03d%s", bn, suffix)
					_, _ = db.Exec("INSERT INTO target_board_elimination (uuid, bracket_uuid, category_uuid, board_number, code) VALUES (?, ?, ?, ?, ?)",
						uuid.New().String(), bracket.UUID, bracket.CategoryUUID, bn, code)
				}
			}
		}

		// Fetch matches
		type MatchRow struct {
			MatchUUID   string         `db:"match_uuid"`
			RoundNo     int            `db:"round_no"`
			MatchNo     int            `db:"match_no"`
			ScheduledAt sql.NullTime   `db:"scheduled_at"`
			TargetName  sql.NullString `db:"target_name"`
			BoardCode   sql.NullString `db:"board_code"`
			TypeA       sql.NullString `db:"type_a"`
			SeedA       sql.NullInt64  `db:"seed_a"`
			NameA       sql.NullString `db:"name_a"`
			ClubA       sql.NullString `db:"club_a"`
			TypeB       sql.NullString `db:"type_b"`
			SeedB       sql.NullInt64  `db:"seed_b"`
			NameB       sql.NullString `db:"name_b"`
			ClubB       sql.NullString `db:"club_b"`
		}
		var matchRows []MatchRow
		_ = db.Select(&matchRows, `
			SELECT
			    em.uuid AS match_uuid,
			    em.round_no, em.match_no, em.scheduled_at,
			    COALESCE(et.target_name, '') AS target_name,
			    COALESCE(tbe.code, '')       AS board_code,
			    eeA.participant_type AS type_a,
			    COALESCE(eeA.seed, 0)        AS seed_a,
			    CASE
			        WHEN eeA.participant_type = 'archer' THEN aA.full_name
			        WHEN eeA.participant_type = 'team'   THEN tA.team_name
			        ELSE ''
			    END AS name_a,
			    COALESCE(cA.name, 'Individu') AS club_a,
			    eeB.participant_type AS type_b,
			    COALESCE(eeB.seed, 0)        AS seed_b,
			    CASE
			        WHEN eeB.participant_type = 'archer' THEN aB.full_name
			        WHEN eeB.participant_type = 'team'   THEN tB.team_name
			        ELSE ''
			    END AS name_b,
			    COALESCE(cB.name, 'Individu') AS club_b
			FROM elimination_matches em
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN archers aA       ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
			LEFT JOIN clubs   cA       ON aA.club_id = cA.uuid
			LEFT JOIN teams   tA       ON eeA.participant_type = 'team'   AND eeA.participant_uuid = tA.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN archers aB       ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
			LEFT JOIN clubs   cB       ON aB.club_id = cB.uuid
			LEFT JOIN teams   tB       ON eeB.participant_type = 'team'   AND eeB.participant_uuid = tB.uuid
			LEFT JOIN tournament_targets et ON em.target_uuid = et.uuid
			LEFT JOIN target_board_elimination tbe ON tbe.bracket_uuid = em.bracket_uuid AND tbe.board_number = et.board_number
			WHERE em.bracket_uuid = ? AND (em.is_bye = 0 OR em.is_bye IS NULL)
			ORDER BY COALESCE(et.board_number, 9999) ASC, em.round_no ASC, em.match_no ASC`, bracket.UUID)

		blankMode := c.Query("blank") == "1"
		if len(matchRows) == 0 || blankMode {
			matchRows = []MatchRow{{
				RoundNo: 1,
				MatchNo: 1,
			}}
		}

		isSetSystem := bracket.Format == "recurve_set" || strings.Contains(strings.ToLower(bracket.Format), "set")
		numEnds := bracket.EndsPerMatch
		if numEnds <= 0 {
			numEnds = 5
		}

		pdf := gofpdf.New("P", "mm", "A4", "")
		pdf.SetAutoPageBreak(false, 0)

		renderIndividualEliminationHalf := func(y float64, m MatchRow, copyBadge string) {
			// 1. Header (Tournament & Document Info)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.2)

			pdf.SetFont("Arial", "B", 9.5)
			pdf.SetXY(10, y)
			pdf.CellFormat(120, 4.5, ev.Name, "", 0, "L", false, 0, "")

			pdf.SetFont("Arial", "B", 9.5)
			pdf.SetXY(130, y)
			pdf.CellFormat(70, 4.5, "Individual Elimination Match", "", 1, "R", false, 0, "")

			pdf.SetFont("Arial", "", 6.8)
			pdf.SetTextColor(80, 80, 80)
			pdf.SetXY(10, y+4.5)
			pdf.CellFormat(120, 3.5, fmt.Sprintf("%s | %s", loc, dateStr), "", 0, "L", false, 0, "")

			pdf.SetXY(130, y+4.5)
			rightSub := "World Archery / IanSeo Official Match Record [C75A]"
			if copyBadge != "" {
				rightSub = fmt.Sprintf("WA / IanSeo Match Record  [%s]", copyBadge)
			}
			pdf.CellFormat(70, 3.5, rightSub, "", 1, "R", false, 0, "")

			pdf.SetDrawColor(180, 180, 180)
			pdf.SetLineWidth(0.2)
			pdf.Line(10, y+8.5, 200, y+8.5)

			// 2. Round & Match Info Ribbon
			roundName := getPrintElimRoundLabel(bracket.BracketSize, m.RoundNo, m.MatchNo)
			pdf.SetFillColor(240, 240, 240)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.2)
			pdf.SetFont("Arial", "B", 8)
			pdf.SetXY(10, y+10)

			targetInfo := "Target: -"
			if m.TargetName.Valid && m.TargetName.String != "" {
				targetInfo = fmt.Sprintf("Target: %s", m.TargetName.String)
			}
			schedInfo := ""
			if m.ScheduledAt.Valid {
				schedInfo = fmt.Sprintf(" | Time: %s", m.ScheduledAt.Time.Format("02 Jan 15:04"))
			}

			catName := bracket.CategoryName
			if blankMode || catName == "" {
				catName = "Individual Elimination"
			}
			infoText := fmt.Sprintf(" %s - %s | %s%s", catName, roundName, targetInfo, schedInfo)
			pdf.CellFormat(190, 5.5, infoText, "1", 1, "L", true, 0, "")

			// 3. Archer A Card (Left) & Archer B Card (Right)
			cardW := 92.0
			cardH := 18.0
			yCards := y + 17.0

			// Archer A Box
			pdf.SetXY(10, yCards)
			pdf.Rect(10, yCards, cardW, cardH, "D")

			seedATxt := "-"
			if m.SeedA.Valid && m.SeedA.Int64 > 0 {
				seedATxt = fmt.Sprintf("#%d", m.SeedA.Int64)
			}
			pdf.SetFillColor(240, 240, 240)
			pdf.Rect(11.5, yCards+1.5, 9, 4.5, "FD")
			pdf.SetFont("Arial", "B", 7)
			pdf.SetXY(11.5, yCards+1.5)
			pdf.CellFormat(9, 4.5, seedATxt, "", 0, "C", false, 0, "")

			pdf.SetFont("Arial", "B", 8.5)
			pdf.SetXY(22, yCards+1.5)
			nameA := strings.ToUpper(m.NameA.String)
			if blankMode || nameA == "" {
				nameA = "ARCHER A: ...................................."
			}
			pdf.CellFormat(78, 4.5, nameA, "", 1, "L", false, 0, "")

			pdf.SetTextColor(60, 60, 60)
			pdf.SetFont("Arial", "", 6.8)
			pdf.SetXY(12, yCards+6.5)
			clubAName := m.ClubA.String
			if blankMode || clubAName == "" {
				clubAName = "-"
			}
			nocA := generateNocCode(clubAName)
			pdf.CellFormat(88, 3.2, fmt.Sprintf("Club / Contingent: [%s] %s", nocA, clubAName), "", 1, "L", false, 0, "")

			if m.BoardCode.Valid && m.BoardCode.String != "" && !blankMode {
				pdf.SetFont("Arial", "B", 6.5)
				pdf.SetXY(12, yCards+10.5)
				pdf.CellFormat(88, 3.2, fmt.Sprintf("Board Code: %s", m.BoardCode.String), "", 1, "L", false, 0, "")
			}

			// Archer B Box
			xCardB := 108.0
			pdf.SetTextColor(0, 0, 0)
			pdf.SetXY(xCardB, yCards)
			pdf.Rect(xCardB, yCards, cardW, cardH, "D")

			seedBTxt := "-"
			if m.SeedB.Valid && m.SeedB.Int64 > 0 {
				seedBTxt = fmt.Sprintf("#%d", m.SeedB.Int64)
			}
			pdf.SetFillColor(240, 240, 240)
			pdf.Rect(xCardB+1.5, yCards+1.5, 9, 4.5, "FD")
			pdf.SetFont("Arial", "B", 7)
			pdf.SetXY(xCardB+1.5, yCards+1.5)
			pdf.CellFormat(9, 4.5, seedBTxt, "", 0, "C", false, 0, "")

			pdf.SetFont("Arial", "B", 8.5)
			pdf.SetXY(xCardB+12, yCards+1.5)
			nameB := strings.ToUpper(m.NameB.String)
			if blankMode || nameB == "" {
				nameB = "ARCHER B: ...................................."
			}
			pdf.CellFormat(78, 4.5, nameB, "", 1, "L", false, 0, "")

			pdf.SetTextColor(60, 60, 60)
			pdf.SetFont("Arial", "", 6.8)
			pdf.SetXY(xCardB+2, yCards+6.5)
			clubBName := m.ClubB.String
			if blankMode || clubBName == "" {
				clubBName = "-"
			}
			nocB := generateNocCode(clubBName)
			pdf.CellFormat(88, 3.2, fmt.Sprintf("Club / Contingent: [%s] %s", nocB, clubBName), "", 1, "L", false, 0, "")

			if m.BoardCode.Valid && m.BoardCode.String != "" && !blankMode {
				pdf.SetFont("Arial", "B", 6.5)
				pdf.SetXY(xCardB+2, yCards+10.5)
				pdf.CellFormat(88, 3.2, fmt.Sprintf("Board Code: %s", m.BoardCode.String), "", 1, "L", false, 0, "")
			}

			// 4. Scoresheet Table (World Archery / IanSeo Standard)
			yTable := yCards + cardH + 2.5
			pdf.SetY(yTable)
			pdf.SetX(10)
			pdf.SetFillColor(232, 232, 232)
			pdf.SetTextColor(0, 0, 0)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.2)
			pdf.SetFont("Arial", "B", 6.8)

			setColTitle := "Set Pts"
			runningColTitle := "Total Set"
			if !isSetSystem {
				setColTitle = "Total"
				runningColTitle = "Running"
			}

			arrowColW := 9.0
			wEnd := 7.5
			wTot := 13.5
			wSet := 14.5
			wRun := 15.0
			wCenter := 14.0

			pdf.CellFormat(wEnd, 4.2, "Set", "1", 0, "C", true, 0, "")
			pdf.CellFormat(arrowColW, 4.2, "1", "1", 0, "C", true, 0, "")
			pdf.CellFormat(arrowColW, 4.2, "2", "1", 0, "C", true, 0, "")
			pdf.CellFormat(arrowColW, 4.2, "3", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wTot, 4.2, "Total", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wSet, 4.2, setColTitle, "1", 0, "C", true, 0, "")
			pdf.CellFormat(wRun, 4.2, runningColTitle, "1", 0, "C", true, 0, "")

			// Center VS
			pdf.CellFormat(wCenter, 4.2, "VS", "1", 0, "C", true, 0, "")

			// Archer B side
			pdf.CellFormat(wRun, 4.2, runningColTitle, "1", 0, "C", true, 0, "")
			pdf.CellFormat(wSet, 4.2, setColTitle, "1", 0, "C", true, 0, "")
			pdf.CellFormat(wTot, 4.2, "Total", "1", 0, "C", true, 0, "")
			pdf.CellFormat(arrowColW, 4.2, "1", "1", 0, "C", true, 0, "")
			pdf.CellFormat(arrowColW, 4.2, "2", "1", 0, "C", true, 0, "")
			pdf.CellFormat(arrowColW, 4.2, "3", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wEnd, 4.2, "Set", "1", 1, "C", true, 0, "")

			// Sets / Ends Rows
			rowH := 5.2
			for setNum := 1; setNum <= numEnds; setNum++ {
				pdf.SetX(10)
				pdf.SetFont("Arial", "B", 7.5)
				pdf.SetFillColor(240, 240, 240)
				pdf.CellFormat(wEnd, rowH, fmt.Sprintf("%d", setNum), "1", 0, "C", true, 0, "")

				for a := 1; a <= 3; a++ {
					pdf.CellFormat(arrowColW, rowH, "", "1", 0, "C", false, 0, "")
				}
				pdf.CellFormat(wTot, rowH, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(wSet, rowH, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(wRun, rowH, "", "1", 0, "C", true, 0, "")

				pdf.CellFormat(wCenter, rowH, fmt.Sprintf("S%d", setNum), "1", 0, "C", true, 0, "")

				pdf.CellFormat(wRun, rowH, "", "1", 0, "C", true, 0, "")
				pdf.CellFormat(wSet, rowH, "", "1", 0, "C", false, 0, "")
				pdf.CellFormat(wTot, rowH, "", "1", 0, "C", true, 0, "")
				for a := 1; a <= 3; a++ {
					pdf.CellFormat(arrowColW, rowH, "", "1", 0, "C", false, 0, "")
				}
				pdf.CellFormat(wEnd, rowH, fmt.Sprintf("%d", setNum), "1", 1, "C", true, 0, "")
			}

			// Shoot-Off Row (1 arrow each)
			soH := 5.2
			pdf.SetX(10)
			pdf.SetFont("Arial", "B", 7)
			pdf.SetFillColor(240, 240, 240)
			pdf.CellFormat(wEnd, soH, "S.O.", "1", 0, "C", true, 0, "")
			pdf.CellFormat(arrowColW, soH, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(arrowColW*2+wTot, soH, "Closest to Center: [   ]", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wSet, soH, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(wRun, soH, "", "1", 0, "C", true, 0, "")

			pdf.CellFormat(wCenter, soH, "TIE", "1", 0, "C", true, 0, "")

			pdf.CellFormat(wRun, soH, "", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wSet, soH, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(arrowColW*2+wTot, soH, "Closest to Center: [   ]", "1", 0, "C", true, 0, "")
			pdf.CellFormat(arrowColW, soH, "", "1", 0, "C", false, 0, "")
			pdf.CellFormat(wEnd, soH, "S.O.", "1", 1, "C", true, 0, "")

			// 5. Final Result Box
			pdf.SetX(10)
			pdf.SetFillColor(232, 232, 232)
			pdf.SetFont("Arial", "B", 7)
			pdf.CellFormat(wEnd+arrowColW*3+wTot, 5.5, "TOTAL SET POINTS / SCORE", "1", 0, "R", true, 0, "")
			pdf.CellFormat(wSet+wRun, 5.5, "", "1", 0, "C", true, 0, "")

			pdf.CellFormat(wCenter, 5.5, "WINNER", "1", 0, "C", true, 0, "")

			pdf.CellFormat(wSet+wRun, 5.5, "", "1", 0, "C", true, 0, "")
			pdf.CellFormat(wEnd+arrowColW*3+wTot, 5.5, "TOTAL SET POINTS / SCORE", "1", 1, "L", true, 0, "")

			// 6. Signatures Section
			ySig := y + 84.0
			pdf.SetY(ySig)
			pdf.SetDrawColor(0, 0, 0)
			pdf.SetLineWidth(0.15)
			pdf.SetFont("Arial", "", 6.5)

			pdf.Line(15, ySig+9, 65, ySig+9)
			pdf.SetXY(15, ySig+9.5)
			pdf.CellFormat(50, 3.2, "Archer A Signature", "", 0, "C", false, 0, "")

			pdf.Line(82.5, ySig+9, 132.5, ySig+9)
			pdf.SetXY(82.5, ySig+9.5)
			pdf.CellFormat(50, 3.2, "Judge / Scorekeeper Signature", "", 0, "C", false, 0, "")

			pdf.Line(150, ySig+9, 200, ySig+9)
			pdf.SetXY(150, ySig+9.5)
			pdf.CellFormat(50, 3.2, "Archer B Signature", "", 1, "C", false, 0, "")

			pdf.SetFont("Arial", "I", 5.2)
			pdf.SetTextColor(110, 110, 110)
			pdf.SetXY(10, ySig+13.5)
			pdf.CellFormat(190, 2.5, fmt.Sprintf("%s | %s | %s | WA Rules Match Record", ev.Name, catName, dateStr), "", 1, "C", false, 0, "")
		}

		for _, m := range matchRows {
			pdf.AddPage()

			// Top Half (Scorer / Judge Copy)
			renderIndividualEliminationHalf(8.0, m, "Scorer Copy")

			// Middle Cut Line (Dotted guide at Y = 148.5mm)
			pdf.SetDrawColor(160, 160, 160)
			pdf.SetLineWidth(0.15)
			pdf.SetFont("Arial", "", 6)
			pdf.SetTextColor(120, 120, 120)
			pdf.SetXY(6, 147.0)
			pdf.CellFormat(198, 3, "✂ - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - -", "", 1, "C", false, 0, "")

			// Bottom Half (Athlete Copy)
			renderIndividualEliminationHalf(154.0, m, "Athlete Copy")
		}

		setPdfHeaders(c, fmt.Sprintf("EliminationScoresheet-%s.pdf", ev.Slug))
		err = pdf.Output(c.Writer)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to output elimination scoresheet PDF"})
		}
	}
}


