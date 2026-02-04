package handler

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
			UUID           string  `json:"id" db:"uuid"`
			EventUUID      string  `json:"event_id" db:"event_uuid"`
			CategoryUUID   string  `json:"category_id" db:"category_uuid"`
			CategoryName   string  `json:"category_name" db:"category_name"`
			BracketType    string  `json:"bracket_type" db:"bracket_type"`
			Format         string  `json:"format" db:"format"`
			BracketSize    int     `json:"bracket_size" db:"bracket_size"`
			Status         string  `json:"status" db:"status"`
			GeneratedAt    *string `json:"generated_at" db:"generated_at"`
			CreatedAt      string  `json:"created_at" db:"created_at"`
			MatchCount     int     `json:"match_count" db:"match_count"`
		}

		query := `
			SELECT eb.uuid, eb.event_uuid, eb.category_uuid, 
				COALESCE(CONCAT(COALESCE(rbt.name, ''), ' ', COALESCE(rag.name, ''), ' ', COALESCE(rgd.name, '')), 'Unknown Category') as category_name,
				eb.bracket_type, eb.format, eb.bracket_size, 
				eb.status, eb.generated_at, eb.created_at,
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
			UUID           string  `json:"id" db:"uuid"`
			EventUUID      string  `json:"event_id" db:"event_uuid"`
			CategoryUUID   string  `json:"category_id" db:"category_uuid"`
			BracketType    string  `json:"bracket_type" db:"bracket_type"`
			Format         string  `json:"format" db:"format"`
			BracketSize    int     `json:"bracket_size" db:"bracket_size"`
			Status         string  `json:"status" db:"status"`
			GeneratedAt    *string `json:"generated_at" db:"generated_at"`
			CreatedAt      string  `json:"created_at" db:"created_at"`
		}

		var bracket Bracket
		err := db.Get(&bracket, `SELECT * FROM elimination_brackets WHERE uuid = ?`, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		// Get entries
		type Entry struct {
			UUID            string  `json:"id" db:"uuid"`
			ParticipantType string  `json:"participant_type" db:"participant_type"`
			ParticipantUUID string  `json:"participant_id" db:"participant_uuid"`
			ParticipantName string  `json:"participant_name" db:"participant_name"`
			Seed            int     `json:"seed" db:"seed"`
			QualTotalScore  *int    `json:"qual_total_score" db:"qual_total_score"`
			QualTotalX      *int    `json:"qual_total_x" db:"qual_total_x"`
			QualTotal10     *int    `json:"qual_total_10" db:"qual_total_10"`
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
		`, bracketID)

		if entries == nil {
			entries = []Entry{}
		}

		// Get matches grouped by round
		type Match struct {
			UUID            string  `json:"id" db:"uuid"`
			RoundNo         int     `json:"round_no" db:"round_no"`
			MatchNo         int     `json:"match_no" db:"match_no"`
			EntryAUUID      *string `json:"entry_a_id" db:"entry_a_uuid"`
			EntryBUUID      *string `json:"entry_b_id" db:"entry_b_uuid"`
			EntryAName      *string `json:"entry_a_name" db:"entry_a_name"`
			EntryBName      *string `json:"entry_b_name" db:"entry_b_name"`
			EntryASeed      *int    `json:"entry_a_seed" db:"entry_a_seed"`
			EntryBSeed      *int    `json:"entry_b_seed" db:"entry_b_seed"`
			WinnerEntryUUID *string `json:"winner_entry_id" db:"winner_entry_uuid"`
			IsBye           bool    `json:"is_bye" db:"is_bye"`
			ScheduledAt     *string `json:"scheduled_at" db:"scheduled_at"`
			Status          string  `json:"status" db:"status"`
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
				em.winner_entry_uuid, em.is_bye, em.scheduled_at, em.status
			FROM elimination_matches em
			LEFT JOIN elimination_entries eeA ON em.entry_a_uuid = eeA.uuid
			LEFT JOIN elimination_entries eeB ON em.entry_b_uuid = eeB.uuid
			LEFT JOIN archers aA ON eeA.participant_type = 'archer' AND eeA.participant_uuid = aA.uuid
			LEFT JOIN archers aB ON eeB.participant_type = 'archer' AND eeB.participant_uuid = aB.uuid
			LEFT JOIN teams tA ON eeA.participant_type = 'team' AND eeA.participant_uuid = tA.uuid
			LEFT JOIN teams tB ON eeB.participant_type = 'team' AND eeB.participant_uuid = tB.uuid
			WHERE em.bracket_uuid = ?
			ORDER BY em.round_no ASC, em.match_no ASC
		`, bracketID)

		if matches == nil {
			matches = []Match{}
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
			CategoryID     string `json:"category_id" binding:"required"`
			BracketType    string `json:"bracket_type" binding:"required"` // individual, team3, mixed2
			Format         string `json:"format" binding:"required"`       // recurve_set, compound_total
			BracketSize    int    `json:"bracket_size" binding:"required"` // 8, 16, 32, 64
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
			db.Get(&participantCount, `SELECT COUNT(*) FROM event_participants WHERE category_id = ? AND status = 'confirmed'`, req.CategoryID)
		} else {
			// Count teams for this tournament/category
			db.Get(&participantCount, `SELECT COUNT(*) FROM teams WHERE category_uuid = ?`, req.CategoryID)
		}

		if participantCount < req.BracketSize {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Jumlah peserta tidak mencukupi. Diperlukan minimal %d peserta, tersedia %d peserta.", req.BracketSize, participantCount),
				"participant_count": participantCount,
				"required": req.BracketSize,
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
		_, err = db.Exec(`
			INSERT INTO elimination_brackets (uuid, event_uuid, category_uuid, bracket_type, format, bracket_size, status)
			VALUES (?, ?, ?, ?, ?, ?, 'draft')
		`, bracketUUID, eventUUID, req.CategoryID, req.BracketType, req.Format, req.BracketSize)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bracket", "details": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Bracket created successfully",
			"id":      bracketUUID,
		})
	}
}

// GenerateBracket populates entries from qualification and creates match structure
func GenerateBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		// Get bracket info
		type Bracket struct {
			UUID           string `db:"uuid"`
			EventUUID      string `db:"event_uuid"`
			CategoryUUID   string `db:"category_uuid"`
			BracketType    string `db:"bracket_type"`
			BracketSize    int    `db:"bracket_size"`
			Status         string `db:"status"`
		}

		var bracket Bracket
		err := db.Get(&bracket, `SELECT uuid, event_uuid, category_uuid, bracket_type, bracket_size, status FROM elimination_brackets WHERE uuid = ?`, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

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
		tx.Exec(`DELETE FROM elimination_entries WHERE bracket_uuid = ?`, bracketID)
		tx.Exec(`DELETE FROM elimination_matches WHERE bracket_uuid = ?`, bracketID)

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
				LEFT JOIN qualification_end_scores qes ON qes.archer_uuid = a.uuid
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
				`, entryUUID, bracketID, participantType, entries[i].ParticipantUUID, i+1, entries[i].TotalScore, entries[i].TotalX, entries[i].Total10)
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
					INSERT INTO elimination_matches (uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid, is_bye, status)
					VALUES (?, ?, ?, ?, ?, ?, ?, 'scheduled')
				`, matchUUID, bracketID, roundNo, matchNo, entryAUUID, entryBUUID, isBye)
				
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create matches", "details": err.Error()})
					return
				}
			}
		}

		// Update bracket status
		now := time.Now().Format("2006-01-02 15:04:05")
		_, err = tx.Exec(`UPDATE elimination_brackets SET status = 'generated', generated_at = ? WHERE uuid = ?`, now, bracketID)
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

// DeleteBracket deletes a bracket and all related data
func DeleteBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		// Get bracket to check status
		var status string
		err := db.Get(&status, `SELECT status FROM elimination_brackets WHERE uuid = ?`, bracketID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bracket not found"})
			return
		}

		if status == "running" {
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
			WHERE em.bracket_uuid = ?`, bracketID)
		tx.Exec(`DELETE eme FROM elimination_match_ends eme 
			JOIN elimination_matches em ON eme.match_uuid = em.uuid
			WHERE em.bracket_uuid = ?`, bracketID)
		tx.Exec(`DELETE FROM elimination_matches WHERE bracket_uuid = ?`, bracketID)
		tx.Exec(`DELETE FROM elimination_entries WHERE bracket_uuid = ?`, bracketID)
		tx.Exec(`DELETE FROM elimination_brackets WHERE uuid = ?`, bracketID)

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
			UUID            string  `json:"id" db:"uuid"`
			BracketUUID     string  `json:"bracket_id" db:"bracket_uuid"`
			RoundNo         int     `json:"round_no" db:"round_no"`
			MatchNo         int     `json:"match_no" db:"match_no"`
			EntryAUUID      *string `json:"entry_a_id" db:"entry_a_uuid"`
			EntryBUUID      *string `json:"entry_b_id" db:"entry_b_uuid"`
			WinnerEntryUUID *string `json:"winner_entry_id" db:"winner_entry_uuid"`
			IsBye           bool    `json:"is_bye" db:"is_bye"`
			ScheduledAt     *string `json:"scheduled_at" db:"scheduled_at"`
			Status          string  `json:"status" db:"status"`
			Format          string  `json:"format" db:"format"`
		}

		var match Match
		err := db.Get(&match, `
			SELECT em.*, eb.format 
			FROM elimination_matches em
			JOIN elimination_brackets eb ON em.bracket_uuid = eb.uuid
			WHERE em.uuid = ?
		`, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		// Get participant names
		type Participant struct {
			EntryUUID  string  `json:"entry_id" db:"entry_uuid"`
			Name       string  `json:"name" db:"name"`
			Seed       int     `json:"seed" db:"seed"`
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
			UUID     string `json:"id" db:"uuid"`
			EndNo    int    `json:"end_no" db:"end_no"`
			Side     string `json:"side" db:"side"`
			EndTotal int    `json:"end_total" db:"end_total"`
			XCount   int    `json:"x_count" db:"x_count"`
			TenCount int    `json:"ten_count" db:"ten_count"`
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

		c.JSON(http.StatusOK, gin.H{
			"match":        match,
			"participant_a": participantA,
			"participant_b": participantB,
			"ends":         ends,
		})
	}
}

// UpdateMatchScore updates or creates end scores for a match
func UpdateMatchScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		matchID := c.Param("matchId")

		var req struct {
			EndNo    int `json:"end_no" binding:"required"`
			ScoreA   int `json:"score_a"`
			ScoreB   int `json:"score_b"`
			XCountA  int `json:"x_count_a"`
			XCountB  int `json:"x_count_b"`
			TenCountA int `json:"ten_count_a"`
			TenCountB int `json:"ten_count_b"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check match exists
		var matchStatus string
		err := db.Get(&matchStatus, `SELECT status FROM elimination_matches WHERE uuid = ?`, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
			return
		}

		// Upsert scores for side A
		_, err = db.Exec(`
			INSERT INTO elimination_match_ends (uuid, match_uuid, end_no, side, end_total, x_count, ten_count)
			VALUES (?, ?, ?, 'A', ?, ?, ?)
			ON DUPLICATE KEY UPDATE end_total = VALUES(end_total), x_count = VALUES(x_count), ten_count = VALUES(ten_count)
		`, uuid.New().String(), matchID, req.EndNo, req.ScoreA, req.XCountA, req.TenCountA)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update score A", "details": err.Error()})
			return
		}

		// Upsert scores for side B
		_, err = db.Exec(`
			INSERT INTO elimination_match_ends (uuid, match_uuid, end_no, side, end_total, x_count, ten_count)
			VALUES (?, ?, ?, 'B', ?, ?, ?)
			ON DUPLICATE KEY UPDATE end_total = VALUES(end_total), x_count = VALUES(x_count), ten_count = VALUES(ten_count)
		`, uuid.New().String(), matchID, req.EndNo, req.ScoreB, req.XCountB, req.TenCountB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update score B", "details": err.Error()})
			return
		}

		// Update match status to running if scheduled
		if matchStatus == "scheduled" {
			db.Exec(`UPDATE elimination_matches SET status = 'running' WHERE uuid = ?`, matchID)
		}

		c.JSON(http.StatusOK, gin.H{"message": "Score updated successfully"})
	}
}

// FinishMatch marks a match as finished and advances winner
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
			UUID        string `db:"uuid"`
			BracketUUID string `db:"bracket_uuid"`
			RoundNo     int    `db:"round_no"`
			MatchNo     int    `db:"match_no"`
			EntryAUUID  *string `db:"entry_a_uuid"`
			EntryBUUID  *string `db:"entry_b_uuid"`
		}

		var match MatchInfo
		err := db.Get(&match, `SELECT uuid, bracket_uuid, round_no, match_no, entry_a_uuid, entry_b_uuid FROM elimination_matches WHERE uuid = ?`, matchID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Match not found"})
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

		// Update match
		_, err = tx.Exec(`UPDATE elimination_matches SET winner_entry_uuid = ?, status = 'finished' WHERE uuid = ?`, req.WinnerEntryID, matchID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update match"})
			return
		}

		// Advance winner to next round
		// Find next round match
		var bracketSize int
		tx.Get(&bracketSize, `SELECT bracket_size FROM elimination_brackets WHERE uuid = ?`, match.BracketUUID)
		
		numRounds := int(math.Log2(float64(bracketSize)))
		if match.RoundNo < numRounds {
			// Calculate next match position
			nextMatchNo := (match.MatchNo + 1) / 2
			nextRound := match.RoundNo + 1
			
			// Determine if winner goes to slot A or B (odd match = A, even match = B)
			slot := "entry_a_uuid"
			if match.MatchNo%2 == 0 {
				slot = "entry_b_uuid"
			}
			
			_, err = tx.Exec(fmt.Sprintf(`UPDATE elimination_matches SET %s = ? WHERE bracket_uuid = ? AND round_no = ? AND match_no = ?`, slot),
				req.WinnerEntryID, match.BracketUUID, nextRound, nextMatchNo)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to advance winner"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Match finished and winner advanced"})
	}
}

// StartBracket changes bracket status to running
func StartBracket(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bracketID := c.Param("bracketId")

		result, err := db.Exec(`UPDATE elimination_brackets SET status = 'running' WHERE uuid = ? AND status = 'generated'`, bracketID)
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

		result, err := db.Exec(`UPDATE elimination_brackets SET status = 'closed' WHERE uuid = ? AND status = 'running'`, bracketID)
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
