package handler

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// ============= BRACKET CRUD =============

// GetBrackets returns all brackets for an event
func GetBrackets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Resolve event UUID
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		categoryID := c.Query("category_id")

		type BracketInfo struct {
			BracketID     string  `json:"id" db:"bracket_id"`
			UUID          string  `json:"uuid" db:"uuid"`
			EventUUID     string  `json:"event_id" db:"event_uuid"`
			CategoryUUID  string  `json:"category_id" db:"category_uuid"`
			CategoryName  string  `json:"category_name" db:"category_name"`
			BracketType   string  `json:"bracket_type" db:"bracket_type"`
			Format        string  `json:"format" db:"format"`
			BracketSize   int     `json:"bracket_size" db:"bracket_size"`
			EndsPerMatch  int     `json:"ends_per_match" db:"ends_per_match"`
			ArrowsPerEnd  int     `json:"arrows_per_end" db:"arrows_per_end"`
			Status        string  `json:"status" db:"status"`
			GeneratedAt   *string `json:"generated_at" db:"generated_at"`
			CreatedAt     string  `json:"created_at" db:"created_at"`
			MatchCount    int     `json:"match_count" db:"match_count"`
		}

		query := `
			SELECT eb.bracket_id, eb.uuid, eb.event_uuid, eb.category_uuid, 
				COALESCE(CONCAT(COALESCE(rbt.name, ''), ' ', COALESCE(rag.name, ''), ' ', COALESCE(rgd.name, '')), 'Unknown Category') as category_name,
				eb.bracket_type, eb.format, eb.bracket_size, eb.status, eb.ends_per_match, eb.arrows_per_end,
				eb.generated_at, eb.created_at,
				(SELECT COUNT(*) FROM elimination_matches em WHERE em.bracket_uuid = eb.uuid) as match_count
			FROM elimination_brackets eb
			LEFT JOIN event_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE eb.event_uuid = ?
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch brackets", "details": err.Error()})
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
			EventUUID    string  `json:"event_id" db:"event_uuid"`
			CategoryUUID string  `json:"category_id" db:"category_uuid"`
			CategoryName string  `json:"category_name" db:"category_name"`
			BracketType  string  `json:"bracket_type" db:"bracket_type"`
			Format       string  `json:"format" db:"format"`
			BracketSize  int     `json:"bracket_size" db:"bracket_size"`
			Status       string  `json:"status" db:"status"`
			EndsPerMatch int     `json:"ends_per_match" db:"ends_per_match"`
			ArrowsPerEnd int     `json:"arrows_per_end" db:"arrows_per_end"`
			GeneratedAt  *string `json:"generated_at" db:"generated_at"`
			CreatedAt    string  `json:"created_at" db:"created_at"`
		}

		var bracket Bracket
		err := db.Get(&bracket, `
			SELECT eb.bracket_id, eb.uuid, eb.event_uuid, eb.category_uuid, eb.bracket_type, eb.status, eb.format, eb.bracket_size, eb.ends_per_match, eb.arrows_per_end, eb.generated_at, eb.created_at,
				COALESCE(CONCAT(COALESCE(rbt.name, ''), ' ', COALESCE(rag.name, ''), ' ', COALESCE(rgd.name, '')), 'Unknown Category') as category_name
			FROM elimination_brackets eb
			LEFT JOIN event_categories ec ON eb.category_uuid = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			LEFT JOIN ref_gender_divisions rgd ON ec.gender_division_uuid = rgd.uuid
			WHERE eb.bracket_id = ? OR eb.uuid = ?
		`, bracketID, bracketID)
		if err != nil {
			logrus.WithError(err).WithField("bracket_id", bracketID).Error("Failed to fetch bracket")
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
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
			LEFT JOIN archers a ON ee.participant_type = 'archer' AND ee.participant_uuid = a.uuid
			LEFT JOIN teams t ON ee.participant_type = 'team' AND ee.participant_uuid = t.uuid
			WHERE ee.bracket_uuid = ?
			ORDER BY ee.seed ASC
		`, bracketUUID)

		if entries == nil {
			entries = []Entry{}
		}

		// Get matches grouped by round
		type Match struct {
			UUID            string     `json:"id" db:"uuid"`
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
			TotalScoreA     int        `json:"total_score_a"`
			TotalScoreB     int        `json:"total_score_b"`
			TotalPointsA    int        `json:"total_points_a"`
			TotalPointsB    int        `json:"total_points_b"`
			ShootOffA       *string    `json:"shoot_off_a"`
			ShootOffB       *string    `json:"shoot_off_b"`
		}

		var matches []Match
		db.Select(&matches, `
			SELECT em.uuid, em.round_no, em.match_no, 
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
				em.target_uuid, et.target_name
			FROM elimination_matches em
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
			LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
			LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
			LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
			LEFT JOIN event_targets et ON em.target_uuid = et.uuid
			WHERE em.bracket_uuid = ?
			ORDER BY em.round_no ASC, em.match_no ASC
		`, bracketUUID)

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
			logrus.WithError(err).Error("Failed to fetch match ends for bracket")
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch ends", "details": err.Error()})
			return
		}

		if ends == nil {
			ends = []EndScore{}
		}

		// Fetch all arrows for these matches in one query
		type arrowScore struct {
			EndUUID  string `db:"match_end_uuid"`
			ArrowNo  int    `db:"arrow_no"`
			Score    int    `db:"score"`
			IsX      bool   `db:"is_x"`
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
			logrus.WithError(err).Error("Failed to fetch all arrow scores for bracket")
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
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		var req struct {
			CategoryID   string `json:"category_id" binding:"required"`
			BracketType  string `json:"bracket_type" binding:"required"`          // individual, team3, mixed2
			Format       string `json:"format" binding:"required"`                // recurve_set, compound_total
			BracketSize  int    `json:"bracket_size" binding:"required"`          // 8, 16, 32, 64
			EndsPerMatch int    `json:"ends_per_match" binding:"required,min=1"`  // default 5
			ArrowsPerEnd int    `json:"arrows_per_end" binding:"required,min=1"`  // default 3
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate bracket size (must be power of 2)
		validSizes := map[int]bool{4: true, 8: true, 16: true, 32: true, 64: true, 128: true}
		if !validSizes[req.BracketSize] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bracket size. Must be 4, 8, 16, 32, 64, or 128"})
			return
		}

		// Check participant count for the category
		var participantCount int
		if req.BracketType == "individual" {
			// Count archers in this category
			db.Get(&participantCount, `SELECT COUNT(*) FROM event_participants WHERE category_id = ? AND status IN ('confirmed', 'Terdaftar')`, req.CategoryID)
		} else {
			// Count teams for this tournament/category
			db.Get(&participantCount, `SELECT COUNT(*) FROM teams WHERE category_uuid = ?`, req.CategoryID)
		}

		if participantCount < req.BracketSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":             fmt.Sprintf("Jumlah peserta tidak mencukupi. Diperlukan minimal %d peserta, tersedia %d peserta.", req.BracketSize, participantCount),
				"participant_count": participantCount,
				"required":          req.BracketSize,
			})
			return
		}

		// Check if bracket already exists for this category
		var existingCount int
		db.Get(&existingCount, `SELECT COUNT(*) FROM elimination_brackets WHERE event_uuid = ? AND category_uuid = ?`, eventUUID, req.CategoryID)
		if existingCount > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "A bracket already exists for this category"})
			return
		}

		bracketUUID := uuid.New().String()
		bracketID := fmt.Sprintf("BR-%s-%s", time.Now().Format("20060102"), bracketUUID[:8])

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec(`
			INSERT INTO elimination_brackets (uuid, bracket_id, event_uuid, category_uuid, bracket_type, format, bracket_size, ends_per_match, arrows_per_end, status)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'generated')
		`, bracketUUID, bracketID, eventUUID, req.CategoryID, req.BracketType, req.Format, req.BracketSize, req.EndsPerMatch, req.ArrowsPerEnd)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bracket", "details": err.Error()})
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
				SELECT a.uuid as participant_uuid,
					COALESCE(SUM(qes.total_score_end), 0) as total_score,
					COALESCE(SUM(qes.x_count_end), 0) as total_x,
					COALESCE(SUM(qes.ten_count_end), 0) as total_10
				FROM event_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
				WHERE ep.category_id = ?
				GROUP BY a.uuid
				ORDER BY total_score DESC, total_x DESC, total_10 DESC
				LIMIT ?
			`, req.CategoryID, req.BracketSize)
		} else {
			err = tx.Select(&entries, `
				SELECT t.uuid as participant_uuid,
					COALESCE(t.total_score, 0) as total_score,
					COALESCE(t.total_x_count, 0) as total_x,
					0 as total_10
				FROM teams t
				WHERE t.event_id = ? AND t.tournament_id = ?
				ORDER BY total_score DESC, total_x DESC
				LIMIT ?
			`, req.CategoryID, eventUUID, req.BracketSize)
		}

		if err != nil {
			logrus.WithError(err).Error("Failed to fetch qualification results for auto-generation")
		}

		// Create entries
		participantType := "archer"
		if req.BracketType != "individual" {
			participantType = "team"
		}

		entryUUIDs := make([]string, req.BracketSize)
		for i := 0; i < req.BracketSize; i++ {
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
		numRounds := int(math.Log2(float64(req.BracketSize)))
		firstRoundMatchups := generateBracketSeeding(req.BracketSize)

		for roundNo := 1; roundNo <= numRounds; roundNo++ {
			matchesInRound := req.BracketSize / int(math.Pow(2, float64(roundNo)))
			for matchNo := 1; matchNo <= matchesInRound; matchNo++ {
				matchUUID := uuid.New().String()
				var entryAUUID, entryBUUID *string
				isBye := false

				if roundNo == 1 {
					matchIdx := matchNo - 1
					seedA := firstRoundMatchups[matchIdx*2]
					seedB := firstRoundMatchups[matchIdx*2+1]
					if seedA <= len(entries) { entryAUUID = &entryUUIDs[seedA-1] }
					if seedB <= len(entries) { entryBUUID = &entryUUIDs[seedB-1] }
					if (entryAUUID == nil || entryBUUID == nil) && !(entryAUUID == nil && entryBUUID == nil) {
						isBye = true
					}
				}

				tx.Exec(`
					INSERT INTO elimination_matches (uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, is_bye)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, matchUUID, bracketUUID, roundNo, matchNo, entryAUUID, entryBUUID, isBye)
			}
		}

		if req.BracketSize >= 4 {
			bronzeMatchUUID := uuid.New().String()
			tx.Exec(`
				INSERT INTO elimination_matches (uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, is_bye, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, 'pending')
			`, bronzeMatchUUID, bracketUUID, numRounds, 2, nil, nil, false)
		}

		// Update generated_at
		now := time.Now().Format("2006-01-02 15:04:05")
		tx.Exec(`UPDATE elimination_brackets SET generated_at = ? WHERE uuid = ?`, now, bracketUUID)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit bracket creation"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Bracket created and generated successfully",
			"bracket": gin.H{
				"id":   bracketID,
				"uuid": bracketUUID,
			},
		})
	}
}

// UpdateBracket updates an existing elimination bracket
func UpdateBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		var req struct {
			CategoryID   string `json:"category_id" binding:"required"`
			BracketType  string `json:"bracket_type" binding:"required"`
			Format       string `json:"format" binding:"required"`
			BracketSize  int    `json:"bracket_size" binding:"required"`
			EndsPerMatch int    `json:"ends_per_match" binding:"required,min=1"`
			ArrowsPerEnd int    `json:"arrows_per_end" binding:"required,min=1"`
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

		// Validate bracket size
		validSizes := map[int]bool{4: true, 8: true, 16: true, 32: true, 64: true, 128: true}
		if !validSizes[req.BracketSize] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bracket size"})
			return
		}

		_, err = db.Exec(`
			UPDATE elimination_brackets 
			SET category_uuid = ?, bracket_type = ?, format = ?, bracket_size = ?, ends_per_match = ?, arrows_per_end = ?
			WHERE bracket_id = ? OR uuid = ?
		`, req.CategoryID, req.BracketType, req.Format, req.BracketSize, req.EndsPerMatch, req.ArrowsPerEnd, bracketID, bracketID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bracket", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Bracket updated successfully"})
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
			SELECT uuid, event_uuid, category_uuid, bracket_type, bracket_size, status
			FROM elimination_brackets
			WHERE bracket_id = ? OR uuid = ?
		`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		bracketUUID := bracket.UUID

		if bracket.Status != "draft" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bracket has already been generated or is running"})
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
				SELECT a.uuid as participant_uuid,
					COALESCE(SUM(qes.total_score_end), 0) as total_score,
					COALESCE(SUM(qes.x_count_end), 0) as total_x,
					COALESCE(SUM(qes.ten_count_end), 0) as total_10
				FROM event_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid
				WHERE ep.category_id = ?
				GROUP BY a.uuid
				ORDER BY total_score DESC, total_x DESC, total_10 DESC
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
				WHERE t.event_id = ? AND t.tournament_id = ?
				ORDER BY total_score DESC, total_x DESC
				LIMIT ?
			`, bracket.CategoryUUID, bracket.EventUUID, bracket.BracketSize)
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

		for roundNo := 1; roundNo <= numRounds; roundNo++ {
			matchesInRound := bracket.BracketSize / int(math.Pow(2, float64(roundNo)))

			for matchNo := 1; matchNo <= matchesInRound; matchNo++ {
				matchUUID := uuid.New().String()

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

					// If one side has no participant, it's a BYE
					if (entryAUUID == nil || entryBUUID == nil) && !(entryAUUID == nil && entryBUUID == nil) {
						isBye = true
					}
				}
				// Later rounds will be filled as matches complete

				_, err = tx.Exec(`
					INSERT INTO elimination_matches (uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, is_bye)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`, matchUUID, bracketUUID, roundNo, matchNo, entryAUUID, entryBUUID, isBye)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create matches", "details": err.Error()})
					return
				}
			}
		}

		// Create 3rd Place Match (Bronze Match) if bracket size >= 4
		if bracket.BracketSize >= 4 {
			numRounds := int(math.Log2(float64(bracket.BracketSize)))
			bronzeMatchUUID := uuid.New().String()
			_, err = tx.Exec(`
				INSERT INTO elimination_matches (uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, is_bye, status)
				VALUES (?, ?, ?, ?, ?, ?, ?, 'pending')
			`, bronzeMatchUUID, bracketUUID, numRounds, 2, nil, nil, false)
			if err != nil {
				logrus.WithError(err).Error("Failed to create bronze match")
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "Bracket generated successfully",
			"entries_count": len(entries),
			"rounds":        numRounds,
		})
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

// UpdateMatchTargets assigns targets to matches in a bracket
func UpdateMatchTargets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		var req struct {
			Assignments []struct {
				MatchID  string `json:"match_id" binding:"required"`
				TargetID string `json:"target_id"`
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
		err := db.Get(&bracket, `SELECT uuid, event_uuid FROM elimination_brackets WHERE bracket_id = ? OR uuid = ?`, bracketID, bracketID)
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
			var targetUUID *string
			if assignment.TargetID != "" {
				// Validate target belongs to the event
				var exists int
				err = tx.Get(&exists, `SELECT COUNT(*) FROM event_targets WHERE uuid = ? AND event_uuid = ?`, assignment.TargetID, bracket.EventUUID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate target"})
					return
				}
				if exists == 0 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target for this event"})
					return
				}
				targetUUID = &assignment.TargetID
			}

			result, err := tx.Exec(`UPDATE elimination_matches SET target_uuid = ? WHERE uuid = ? AND bracket_uuid = ?`, targetUUID, assignment.MatchID, bracket.UUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update target assignment"})
				return
			}
			affected, _ := result.RowsAffected()
			if affected == 0 {
				c.JSON(http.StatusNotFound, gin.H{"error": "Match not found for this bracket"})
				return
			}
			updated += int(affected)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Targets updated successfully",
			"updated": updated,
		})
	}
}

// DeleteBracket deletes a bracket and all related data
func DeleteBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		// Get bracket to check status
		var bracket struct {
			UUID   string `db:"uuid"`
			Status string `db:"status"`
		}
		err := db.Get(&bracket, `SELECT uuid, status FROM elimination_brackets WHERE bracket_id = ? OR uuid = ?`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		if bracket.Status == "running" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete a running bracket"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// Delete match arrows, ends, matches, entries, then bracket
		tx.Exec(`DELETE emas FROM elimination_match_arrow_scores emas 
			JOIN elimination_match_ends eme ON emas.match_end_uuid = eme.uuid
			JOIN elimination_matches em ON eme.match_uuid = em.uuid
			WHERE em.bracket_uuid = ?`, bracket.UUID)
		tx.Exec(`DELETE eme FROM elimination_match_ends eme 
			JOIN elimination_matches em ON eme.match_uuid = em.uuid
			WHERE em.bracket_uuid = ?`, bracket.UUID)
		tx.Exec(`DELETE FROM elimination_matches WHERE bracket_uuid = ?`, bracket.UUID)
		tx.Exec(`DELETE FROM elimination_entries WHERE bracket_uuid = ?`, bracket.UUID)
		tx.Exec(`DELETE FROM elimination_brackets WHERE uuid = ?`, bracket.UUID)

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bracket"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Bracket deleted successfully"})
	}
}

// ============= MATCH SCORING =============

// GetMatch returns a match with its scoring details
func GetMatch(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		type Match struct {
			UUID            string     `json:"id" db:"uuid"`
			BracketUUID     string     `json:"bracket_id" db:"bracket_uuid"`
			RoundNo         int        `json:"round_no" db:"round_no"`
			MatchNo         int        `json:"match_no" db:"match_no"`
			EntryAUUID      *string    `json:"entry_a_id" db:"entry_a_uuid"`
			EntryBUUID      *string    `json:"entry_b_id" db:"entry_b_uuid"`
			WinnerEntryUUID *string    `json:"winner_entry_id" db:"winner_entry_uuid"`
			Status          string     `json:"status" db:"status"`
			IsBye           bool       `json:"is_bye" db:"is_bye"`
			ScheduledAt     *time.Time `json:"scheduled_at" db:"scheduled_at"`
			TargetUUID      *string    `json:"target_id" db:"target_uuid"`
			Format          string     `json:"format" db:"format"`
			TotalScoreA     int        `json:"total_score_a"`
			TotalScoreB     int        `json:"total_score_b"`
			TotalPointsA    int        `json:"total_points_a"`
			TotalPointsB    int        `json:"total_points_b"`
			ShootOffA       *string    `json:"shoot_off_a"`
			ShootOffB       *string    `json:"shoot_off_b"`
		}

		var match Match
		err := db.Unsafe().Get(&match, `
			SELECT em.*, eb.format 
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			WHERE em.uuid = ?
		`, matchID)
			if err != nil {
			logrus.WithError(err).WithField("match_id", matchID).Error("Failed to fetch match details")
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		// Get participant names
		type Participant struct {
			EntryUUID string `json:"entry_id" db:"entry_uuid"`
			Name      string `json:"name" db:"name"`
			Seed      int    `json:"seed" db:"seed"`
		}

		var participantA, participantB *Participant
		if match.EntryAUUID != nil {
			var p Participant
			db.Get(&p, `
				SELECT ee.uuid as entry_uuid, 
					CASE 
						WHEN ee.participant_type = 'archer' THEN a.full_name
						WHEN ee.participant_type = 'team' THEN t.team_name
					END as name,
					ee.seed
				FROM elimination_entries ee
				LEFT JOIN archers a ON ee.participant_type = 'archer' AND ee.participant_uuid = a.uuid
				LEFT JOIN teams t ON ee.participant_type = 'team' AND ee.participant_uuid = t.uuid
				WHERE ee.uuid = ?
			`, *match.EntryAUUID)
			participantA = &p
		}
		if match.EntryBUUID != nil {
			var p Participant
			db.Get(&p, `
				SELECT ee.uuid as entry_uuid, 
					CASE 
						WHEN ee.participant_type = 'archer' THEN a.full_name
						WHEN ee.participant_type = 'team' THEN t.team_name
					END as name,
					ee.seed
				FROM elimination_entries ee
				LEFT JOIN archers a ON ee.participant_type = 'archer' AND ee.participant_uuid = a.uuid
				LEFT JOIN teams t ON ee.participant_type = 'team' AND ee.participant_uuid = t.uuid
				WHERE ee.uuid = ?
			`, *match.EntryBUUID)
			participantB = &p
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
				} else {
					totalScoreA++
				}
			} else if scB > scA {
				if match.Format == "recurve_set" {
					totalPointsB++
				} else {
					totalScoreB++
				}
			}
		}

		match.TotalScoreA = totalScoreA
		match.TotalScoreB = totalScoreB
		match.TotalPointsA = totalPointsA
		match.TotalPointsB = totalPointsB

		c.JSON(http.StatusOK, gin.H{
			"match":         match,
			"participant_a": participantA,
			"participant_b": participantB,
			"ends":          ends,
		})
	}
}

// UpdateMatchScore updates or creates end scores for a match
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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update side A", "details": err.Error()})
			return
		}

		if err := processSide("B", req.ScoreB, req.ArrowsB); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update side B", "details": err.Error()})
			return
		}
		// Recalculate totals and update elimination_matches
		var m struct {
			Format string `db:"format"`
		}
		if err := tx.Get(&m, `SELECT format FROM elimination_matches WHERE uuid = ?`, matchID); err == nil {
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
				if en == 99 { continue }
				sA, sB := sides["A"], sides["B"]
				tSA += sA
				tSB += sB
				if m.Format == "recurve_set" {
					if sA > sB { tPA += 2 } else if sB > sA { tPB += 2 } else if sA == sB && sA > 0 { tPA += 1; tPB += 1 }
				}
			}

			// Shoot-off +1 logic
			soA, soB := -1, -1
			for _, a := range soArrows {
				val := a.Score
				if a.IsX { val = 11 }
				if a.Side == "A" { soA = val } else { soB = val }
			}
			if soA >= 0 && soB >= 0 {
				if soA > soB { if m.Format == "recurve_set" { tPA++ } else { tSA++ } } else if soB > soA { if m.Format == "recurve_set" { tPB++ } else { tSB++ } }
			}
			tx.Exec(`UPDATE elimination_matches SET total_score_a=?, total_score_b=?, total_points_a=?, total_points_b=? WHERE uuid=?`, tSA, tSB, tPA, tPB, matchID)
		}


		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Score updated successfully"})
	}
}

// FinishMatch marks a match as finished and advances winner to next round
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
		err := db.Get(&match, `SELECT uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, status FROM elimination_matches WHERE uuid = ?`, matchID)
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
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
			// Calculate next match position
			nextMatchNo := (match.MatchNo + 1) / 2
			nextRound := match.RoundNo + 1

			// Determine if winner goes to slot A or B (odd match = A, even match = B)
			slot := "entry_a_uuid"
			if match.MatchNo%2 == 0 {
				slot = "entry_b_uuid"
			}

			// Check if the next round match exists
			var nextMatchExists int
			tx.Get(&nextMatchExists, `SELECT COUNT(*) FROM elimination_matches WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, 
				match.BracketUUID, nextRound, nextMatchNo)

			if nextMatchExists > 0 {
				// Update existing next round match
				_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, slot),
					req.WinnerEntryID, match.BracketUUID, nextRound, nextMatchNo)
				if err != nil {
					logrus.WithError(err).Error("Failed to advance winner to existing match")
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to advance winner"})
					return
				}
			} else {
				// Find the pair match (if match_no is odd, pair is match_no+1; if even, pair is match_no-1)
				var pairMatchNo int
				if match.MatchNo%2 == 1 {
					pairMatchNo = match.MatchNo + 1
				} else {
					pairMatchNo = match.MatchNo - 1
				}

				// Check if pair match is finished
				var pairMatch struct {
					Status          string  `db:"status"`
					WinnerEntryUUID *string `db:"winner_entry_uuid"`
				}
				pairErr := tx.Get(&pairMatch, `SELECT status, winner_entry_uuid FROM elimination_matches WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`,
					match.BracketUUID, match.RoundNo, pairMatchNo)

				// If pair match exists and is finished, create the next round match
				if pairErr == nil && pairMatch.Status == "finished" && pairMatch.WinnerEntryUUID != nil {
					// Determine entry positions for next match
					var entryA, entryB string
					if match.MatchNo%2 == 1 {
						// Current match is odd, so current winner goes to A, pair winner to B
						entryA = req.WinnerEntryID
						entryB = *pairMatch.WinnerEntryUUID
					} else {
						// Current match is even, so pair winner goes to A, current winner to B
						entryA = *pairMatch.WinnerEntryUUID
						entryB = req.WinnerEntryID
					}

					// Create the next round match
					nextMatchUUID := uuid.New().String()
					_, err = tx.Exec(`INSERT INTO elimination_matches (uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, status) VALUES (?, ?, ?, ?, ?, ?, 'pending')`,
						nextMatchUUID, match.BracketUUID, nextRound, nextMatchNo, entryA, entryB)
					if err != nil {
						logrus.WithError(err).Error("Failed to create next round match")
						c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create next round match"})
						return
					}

					logrus.WithFields(logrus.Fields{
						"next_match_uuid": nextMatchUUID,
						"round":           nextRound,
						"match_no":        nextMatchNo,
						"entry_a":         entryA,
						"entry_b":         entryB,
					}).Info("Created next round match")
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
					if match.MatchNo%2 == 0 {
						bronzeSlot = "entry_b_uuid"
					}
					_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, bronzeSlot),
						*loserID, match.BracketUUID, numRounds, 2)
					if err != nil {
						logrus.WithError(err).Error("Failed to advance loser to bronze match")
					}
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":          "Match finished and winner advanced",
			"winner_entry_id":  req.WinnerEntryID,
			"match_status":     "finished",
		})
	}
}

// EndMatch calculates winner from scores and finishes the match
func EndMatch(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		var req struct {
			WinnerEntryID string `json:"winner_entry_id"`
		}
		c.ShouldBindJSON(&req)

		// Get match info including bracket format
		type MatchInfo struct {
			UUID        string  `db:"uuid"`
			BracketUUID string  `db:"bracket_uuid"`
			RoundNo     int     `db:"round_no"`
			MatchNo     int     `db:"match_no"`
			EntryAUUID  *string `db:"entry_a_uuid"`
			EntryBUUID  *string `db:"entry_b_uuid"`
			Status      string  `db:"status"`
			Format      string  `db:"format"`
		}

		var match MatchInfo
		err := db.Get(&match, `
			SELECT em.uuid, em.bracket_uuid, em.round_no, em.match_no, 
					em.entry_a_uuid, em.entry_b_uuid, em.status, eb.format 
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

		// Calculate scores from ends
		type EndScore struct {
			Side     string `db:"side"`
			TotalEnd int    `db:"total_end"`
		}
		var ends []EndScore
		err = db.Select(&ends, `
			SELECT side, SUM(end_total) as total_end 
			FROM elimination_match_ends 
			WHERE match_uuid = ? 
			GROUP BY side`, matchID)
		if err != nil {
			logrus.WithError(err).Error("Failed to fetch end scores")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate scores"})
			return
		}

		var totalA, totalB int
		for _, e := range ends {
			if e.Side == "A" {
				totalA = e.TotalEnd
			} else if e.Side == "B" {
				totalB = e.TotalEnd
			}
		}

		// For set system (recurve), calculate set points
		if match.Format == "recurve_set" {
			type SetEnd struct {
				EndNo  int `db:"end_no"`
				Side   string `db:"side"`
				Total  int    `db:"end_total"`
			}
			var setEnds []SetEnd
			db.Select(&setEnds, `
				SELECT end_no, side, end_total 
				FROM elimination_match_ends 
				WHERE match_uuid = ? 
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
			totalA, totalB = 0, 0
			for _, sides := range endTotals {
				scoreA := sides["A"]
				scoreB := sides["B"]
				if scoreA > scoreB {
					totalA += 2
				} else if scoreB > scoreA {
					totalB += 2
				} else {
					totalA += 1
					totalB += 1
				}
			}
		}

		// Determine winner
		var winnerID string
		if totalA > totalB {
			if match.EntryAUUID != nil {
				winnerID = *match.EntryAUUID
			}
		} else if totalB > totalA {
			if match.EntryBUUID != nil {
				winnerID = *match.EntryBUUID
			}
		} else {
			// Tie - Check for manual selection or tie-breaker scores
			// Check if there is an end 99 (Shoot-off)
			type shootOffEnd struct {
				Side     string `db:"side"`
				EndTotal int    `db:"end_total"`
			}
			var soEnds []shootOffEnd
			db.Select(&soEnds, `SELECT side, end_total FROM elimination_match_ends WHERE match_uuid = ? AND end_no = 99`, matchID)
			
			soA, soB := -1, -1
			for _, e := range soEnds {
				if e.Side == "A" { soA = e.EndTotal }
				if e.Side == "B" { soB = e.EndTotal }
			}

			if soA > soB {
				if match.EntryAUUID != nil { winnerID = *match.EntryAUUID }
			} else if soB > soA {
				if match.EntryBUUID != nil { winnerID = *match.EntryBUUID }
			} else if req.WinnerEntryID != "" {
				// Final manual override if provided in request
				winnerID = req.WinnerEntryID
			} else {
				// Tie - for now, we'll require higher score to win
				c.JSON(http.StatusBadRequest, gin.H{"error": "Pertandingan Seri. Silahkan lakukan Shoot-off atau pilih pemenang secara manual."})
				return
			}
		}

		if winnerID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot determine winner"})
			return
		}

		// Start transaction
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// Update match with winner and status
		_, err = tx.Exec(`UPDATE elimination_matches SET winner_entry_uuid = ?, status = 'finished' WHERE uuid = ?`, winnerID, matchID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update match"})
			return
		}

		// Advance winner to next round (reuse logic from FinishMatch)
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

			var nextMatchExists int
			tx.Get(&nextMatchExists, `SELECT COUNT(*) FROM elimination_matches WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`,
				match.BracketUUID, nextRound, nextMatchNo)

			if nextMatchExists > 0 {
				_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, slot),
					winnerID, match.BracketUUID, nextRound, nextMatchNo)
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
						nextMatchUUID, match.BracketUUID, nextRound, nextMatchNo, entryA, entryB)

					logrus.WithFields(logrus.Fields{
						"next_match_uuid": nextMatchUUID,
						"round":           nextRound,
						"match_no":        nextMatchNo,
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
					if match.MatchNo%2 == 0 {
						bronzeSlot = "entry_b_uuid"
					}
					_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, bronzeSlot),
						*loserID, match.BracketUUID, numRounds, 2)
					if err != nil {
						logrus.WithError(err).Error("Failed to advance loser to bronze match in EndMatch")
					}
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit"})
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

		c.JSON(http.StatusOK, gin.H{
			"message":         "Match ended",
			"winner_entry_id": winnerID,
			"winner_name":     winnerName,
			"score_a":         totalA,
			"score_b":         totalB,
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bracket not found or not in generated state"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Bracket started"})
	}
}

// CloseBracket changes bracket status to closed
func CloseBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		result, err := db.Exec(`
			UPDATE elimination_brackets
			SET status = 'closed'
			WHERE (bracket_id = ? OR uuid = ?) AND status = 'running'
		`, bracketID, bracketID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to close bracket"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bracket not found or not in running state"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Bracket closed"})
	}
}
