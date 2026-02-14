package handler

import (
	"archeryhub-api/models"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetQualificationSessions returns all sessions for an event
func GetQualificationSessions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "eventId is required"})
			return
		}

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		type SessionWithCount struct {
			UUID             string  `db:"uuid" json:"uuid"`
			EventUUID        string  `db:"event_uuid" json:"event_uuid"`
			SessionCode      string  `db:"session_code" json:"session_code"`
			SessionDate      *string `db:"session_date" json:"session_date"`
			Name             string  `db:"name" json:"name"`
			StartTime        *string `db:"start_time" json:"start_time"`
			EndTime          *string `db:"end_time" json:"end_time"`
			TotalEnds        int     `db:"total_ends" json:"total_ends"`
			ArrowsPerEnd     int     `db:"arrows_per_end" json:"arrows_per_end"`
			CreatedAt        *string `db:"created_at" json:"created_at"`
			UpdatedAt        *string `db:"updated_at" json:"updated_at"`
			ParticipantCount int     `db:"participant_count" json:"participant_count"`
		}

		var sessions []SessionWithCount
		err = db.Select(&sessions, `
			SELECT 
				qs.uuid,
				qs.event_uuid,
				qs.session_code,
				qs.session_date,
				qs.name,
				qs.start_time,
				qs.end_time,
				qs.total_ends,
				qs.arrows_per_end,
				qs.created_at,
				qs.updated_at,
				COUNT(DISTINCT qta.participant_uuid) as participant_count
			FROM qualification_sessions qs
			LEFT JOIN qualification_target_assignments qta ON qs.uuid = qta.session_uuid
			WHERE qs.event_uuid = ?
			GROUP BY qs.uuid, qs.session_date, qs.name, qs.start_time, qs.end_time, qs.total_ends, qs.arrows_per_end, qs.created_at, qs.updated_at
			ORDER BY qs.session_date ASC, qs.start_time ASC, qs.created_at ASC
		`, eventUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sessions", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	}
}

// CreateQualificationSession creates a new scoring session
func CreateQualificationSession(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		var req struct {
			Name         string  `json:"name" binding:"required"`
			SessionDate  *string `json:"session_date"`
			StartTime    *string `json:"start_time"`
			EndTime      *string `json:"end_time"`
			TotalEnds    int     `json:"total_ends"`
			ArrowsPerEnd int     `json:"arrows_per_end"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Set defaults
		if req.TotalEnds == 0 {
			req.TotalEnds = 12
		}
		if req.ArrowsPerEnd == 0 {
			req.ArrowsPerEnd = 6
		}

		// Generate session code (e.g., QS-20260203-001)
		var sessionCount int
		_ = db.Get(&sessionCount, `SELECT COUNT(*) FROM qualification_sessions WHERE event_uuid = ?`, eventUUID)
		sessionCode := fmt.Sprintf("QS-%s-%03d", time.Now().Format("20060102"), sessionCount+1)

		// Handle StartTime and EndTime if they are just "HH:MM" and session_date is provided
		var finalStartTime, finalEndTime *string
		if req.SessionDate != nil && *req.SessionDate != "" {
			if req.StartTime != nil && *req.StartTime != "" {
				s := fmt.Sprintf("%s %s:00", *req.SessionDate, *req.StartTime)
				finalStartTime = &s
			}
			if req.EndTime != nil && *req.EndTime != "" {
				s := fmt.Sprintf("%s %s:00", *req.SessionDate, *req.EndTime)
				finalEndTime = &s
			}
		} else {
			finalStartTime = req.StartTime
			finalEndTime = req.EndTime
		}

		newUUID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO qualification_sessions (uuid, event_uuid, session_code, session_date, name, start_time, end_time, total_ends, arrows_per_end)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			newUUID, eventUUID, sessionCode, req.SessionDate, req.Name, finalStartTime, finalEndTime, req.TotalEnds, req.ArrowsPerEnd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "Session created successfully",
			"session_uuid": newUUID,
			"session_code": sessionCode,
		})
	}
}

// UpdateQualificationSession updates an existing session
func UpdateQualificationSession(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionUUID := c.Param("sessionId")

		var req struct {
			Name         string  `json:"name" binding:"required"`
			SessionDate  *string `json:"session_date"`
			StartTime    *string `json:"start_time"`
			EndTime      *string `json:"end_time"`
			TotalEnds    int     `json:"total_ends"`
			ArrowsPerEnd int     `json:"arrows_per_end"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Handle StartTime and EndTime merging
		var finalStartTime, finalEndTime *string
		if req.SessionDate != nil && *req.SessionDate != "" {
			if req.StartTime != nil && *req.StartTime != "" {
				s := fmt.Sprintf("%s %s:00", *req.SessionDate, *req.StartTime)
				finalStartTime = &s
			}
			if req.EndTime != nil && *req.EndTime != "" {
				s := fmt.Sprintf("%s %s:00", *req.SessionDate, *req.EndTime)
				finalEndTime = &s
			}
		} else {
			finalStartTime = req.StartTime
			finalEndTime = req.EndTime
		}

		_, err := db.Exec(`
			UPDATE qualification_sessions 
			SET name = ?, session_date = ?, start_time = ?, end_time = ?, total_ends = ?, arrows_per_end = ?, updated_at = NOW()
			WHERE uuid = ?`,
			req.Name, req.SessionDate, finalStartTime, finalEndTime, req.TotalEnds, req.ArrowsPerEnd, sessionUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Session updated successfully"})
	}
}

// DeleteQualificationSession removes a session and all its related data
func DeleteQualificationSession(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionUUID := c.Param("sessionId")

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// 1. Delete arrow scores
		_, err = tx.Exec(`
			DELETE FROM qualification_arrow_scores 
			WHERE end_score_uuid IN (SELECT uuid FROM qualification_end_scores WHERE session_uuid = ?)`,
			sessionUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete arrow scores"})
			return
		}

		// 2. Delete end scores
		_, err = tx.Exec(`DELETE FROM qualification_end_scores WHERE session_uuid = ?`, sessionUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete end scores"})
			return
		}

		// 3. Delete assignments
		_, err = tx.Exec(`DELETE FROM qualification_target_assignments WHERE session_uuid = ?`, sessionUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete assignments"})
			return
		}

		// 4. Delete the session itself
		_, err = tx.Exec(`DELETE FROM qualification_sessions WHERE uuid = ?`, sessionUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Session deleted successfully"})
	}
}

// UpdateQualificationScore updates end scores for an assignment (supports batch)
func UpdateQualificationScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")

		var sessionUUID, participantUUID string
		if err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}
		if err := db.Get(&participantUUID, `SELECT participant_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}

		var raw map[string]interface{}
		if err := c.ShouldBindJSON(&raw); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var ends []models.SingleEndScore
		if _, exists := raw["ends"]; exists {
			data, _ := json.Marshal(raw)
			var batchReq models.ScoreBatchUpdateRequest
			json.Unmarshal(data, &batchReq)
			ends = batchReq.Ends
		} else if _, exists := raw["end_number"]; exists {
			data, _ := json.Marshal(raw)
			var singleReq models.ScoreUpdateRequest
			json.Unmarshal(data, &singleReq)
			ends = []models.SingleEndScore{{EndNumber: singleReq.EndNumber, Arrows: singleReq.Arrows}}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format: 'ends' or 'end_number' required"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// 1. Fetch all existing end scores for this participant and session once
		type ExistingEnd struct {
			UUID      string `db:"uuid"`
			EndNumber int    `db:"end_number"`
		}
		var existingEnds []ExistingEnd
		err = tx.Select(&existingEnds, `
			SELECT uuid, end_number FROM qualification_end_scores 
			WHERE session_uuid = ? AND participant_uuid = ?`, sessionUUID, participantUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch existing scores"})
			return
		}

		existingMap := make(map[int]string)
		for _, ee := range existingEnds {
			existingMap[ee.EndNumber] = ee.UUID
		}

		var allEndScoreUUIDs []string
		var arrowValues []interface{}
		arrowCount := 0

		for _, end := range ends {
			total, xCount, tenCount := 0, 0, 0
			for _, arrow := range end.Arrows {
				val, x, ten := calculateArrowValue(arrow)
				total += val
				xCount += x
				tenCount += ten
			}

			currentEndScoreUUID, exists := existingMap[end.EndNumber]
			
			if exists {
				// Update end score
				_, err = tx.Exec(`UPDATE qualification_end_scores SET total_score_end = ?, x_count_end = ?, ten_count_end = ? WHERE uuid = ?`,
					total, xCount, tenCount, currentEndScoreUUID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update end score"})
					return
				}
			} else {
				// Insert end score
				currentEndScoreUUID = uuid.New().String()
				_, err = tx.Exec(`INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					currentEndScoreUUID, sessionUUID, participantUUID, end.EndNumber, total, xCount, tenCount)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new end score"})
					return
				}
			}

			allEndScoreUUIDs = append(allEndScoreUUIDs, currentEndScoreUUID)

			for i, arrow := range end.Arrows {
				val, _, _ := calculateArrowValue(arrow)
				isX := 0
				if arrow == "X" {
					isX = 1
				}
				arrowValues = append(arrowValues, uuid.New().String(), currentEndScoreUUID, i+1, val, isX)
				arrowCount++
			}
		}

		// 2. Clear old arrows for all affected ends in a single query
		if len(allEndScoreUUIDs) > 0 {
			query, args, err := sqlx.In(`DELETE FROM qualification_arrow_scores WHERE end_score_uuid IN (?)`, allEndScoreUUIDs)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare arrow cleanup"})
				return
			}
			query = tx.Rebind(query)
			_, err = tx.Exec(query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear old arrow scores"})
				return
			}
		}

		// 3. Bulk insert all new arrows in a single query
		if arrowCount > 0 {
			valueStrings := make([]string, 0, arrowCount)
			for i := 0; i < arrowCount; i++ {
				valueStrings = append(valueStrings, "(?, ?, ?, ?, ?)")
			}
			bulkQuery := fmt.Sprintf("INSERT INTO qualification_arrow_scores (uuid, end_score_uuid, arrow_number, score, is_x) VALUES %s", 
				strings.Join(valueStrings, ","))
			
			_, err = tx.Exec(bulkQuery, arrowValues...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save arrow scores (bulk)"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit scores"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Scores updated successfully"})
	}
}

// GetQualificationAssignmentScores returns all saved end scores for a single assignment
func GetQualificationAssignmentScores(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")
		if assignmentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assignmentId is required"})
			return
		}

		// Get session and participant from assignment
		var sessionUUID, participantUUID string
		err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}
		err = db.Get(&participantUUID, `SELECT participant_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}

		type ArrowScore struct {
			ArrowNumber int  `db:"arrow_number" json:"arrow_number"`
			Score       int  `db:"score" json:"score"`
			IsX         bool `db:"is_x" json:"is_x"`
		}

		type EndScore struct {
			ID            string       `db:"uuid" json:"id"`
			EndNumber     int          `db:"end_number" json:"end_number"`
			TotalScoreEnd int          `db:"total_score_end" json:"total_score_end"`
			XCountEnd     int          `db:"x_count_end" json:"x_count_end"`
			TenCountEnd   int          `db:"ten_count_end" json:"ten_count_end"`
			Arrows        []ArrowScore `json:"arrows"`
		}

		var scores []EndScore
		err = db.Select(&scores, `
			SELECT uuid, end_number, total_score_end, x_count_end, ten_count_end
			FROM qualification_end_scores
			WHERE session_uuid = ? AND participant_uuid = ?
			ORDER BY end_number ASC
		`, sessionUUID, participantUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch end scores"})
			return
		}

		// Get arrow scores for each end
		for i := range scores {
			var arrows []ArrowScore
			err = db.Select(&arrows, `
				SELECT arrow_number, score, is_x
				FROM qualification_arrow_scores
				WHERE end_score_uuid = ?
				ORDER BY arrow_number ASC
			`, scores[i].ID)
			if err == nil {
				scores[i].Arrows = arrows
			} else {
				scores[i].Arrows = []ArrowScore{}
			}
		}

		c.JSON(http.StatusOK, gin.H{"scores": scores})
	}
}

func calculateArrowValue(arrow string) (val int, x int, ten int) {
	switch arrow {
	case "X":
		return 10, 1, 1
	case "10":
		return 10, 0, 1
	case "M":
		return 0, 0, 0
	case "":
		return 0, 0, 0
	default:
		v, _ := strconv.Atoi(arrow)
		return v, 0, 0
	}
}

// GetQualificationLeaderboard returns ranked participants for a category
func GetQualificationLeaderboard(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID := c.Query("category_id")
		if categoryID == "" {
			categoryID = c.Param("categoryId")
		}
		if categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "categoryId is required"})
			return
		}

		type SessionScore struct {
			SessionCode string `json:"session_code"`
			SessionName string `json:"session_name"`
			EndScores   string `json:"end_scores"`
		}

		type Entry struct {
			ParticipantUUID string         `json:"participant_uuid"`
			ArcherName      string         `json:"archer_name"`
			AvatarURL       *string        `json:"avatar_url"`
			ClubName        *string        `json:"club_name"`
			CategoryName    string         `json:"category_name"`
			TotalScore      int            `json:"total_score"`
			TotalTenX       int            `json:"total_10x"`
			TotalX          int            `json:"total_x"`
			EndsCompleted   int            `json:"ends_completed"`
			Sessions        []SessionScore `json:"sessions"`
		}

		type dbEntry struct {
			ParticipantUUID string  `db:"participant_uuid"`
			ArcherName      string  `db:"archer_name"`
			AvatarURL       *string `db:"avatar_url"`
			ClubName        *string `db:"club_name"`
			CategoryName    string  `db:"category_name"`
			SessionName     *string `db:"session_name"`
			SessionCode     *string `db:"session_code"`
			TotalScore      int     `db:"total_score"`
			TotalTenX       int     `db:"total_10x"`
			TotalX          int     `db:"total_x"`
			EndsCompleted   int     `db:"ends_completed"`
			EndScores       *string `db:"end_scores"`
		}

		var dbEntries []dbEntry
		err := db.Select(&dbEntries, `
			SELECT 
				ep.uuid as participant_uuid,
				a.full_name as archer_name,
				a.avatar_url as avatar_url,
				cl.name as club_name,
				CONCAT(bt.name, ' ', ag.name) as category_name,
				qs.name as session_name,
				qs.session_code as session_code,
				COALESCE(score_summary.total_score, 0) as total_score,
				COALESCE(score_summary.total_10x, 0) as total_10x,
				COALESCE(score_summary.total_x, 0) as total_x,
				COALESCE(score_summary.ends_completed, 0) as ends_completed,
				score_summary.end_scores
			FROM event_participants ep
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			JOIN qualification_target_assignments qta ON qta.participant_uuid = ep.uuid
			JOIN qualification_sessions qs ON qs.uuid = qta.session_uuid
			LEFT JOIN (
				SELECT 
					participant_uuid, 
					session_uuid,
					SUM(total_score_end) as total_score,
					SUM(ten_count_end) as total_10x,
					SUM(x_count_end) as total_x,
					COUNT(uuid) as ends_completed,
					GROUP_CONCAT(COALESCE(total_score_end, 0) ORDER BY end_number ASC SEPARATOR ', ') as end_scores
				FROM qualification_end_scores
				GROUP BY participant_uuid, session_uuid
			) score_summary ON score_summary.participant_uuid = ep.uuid AND score_summary.session_uuid = qs.uuid
			WHERE ep.category_id = ?
			ORDER BY participant_uuid, qs.created_at ASC`,
			categoryID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard", "details": err.Error()})
			return
		}

		// Group by archer
		archerMap := make(map[string]*Entry)
		archerOrder := []string{}

		for _, de := range dbEntries {
			if _, ok := archerMap[de.ParticipantUUID]; !ok {
				archerMap[de.ParticipantUUID] = &Entry{
					ParticipantUUID: de.ParticipantUUID,
					ArcherName:      de.ArcherName,
					AvatarURL:       de.AvatarURL,
					ClubName:        de.ClubName,
					CategoryName:    de.CategoryName,
					Sessions:        []SessionScore{},
				}
				archerOrder = append(archerOrder, de.ParticipantUUID)
			}

			entry := archerMap[de.ParticipantUUID]
			entry.TotalScore += de.TotalScore
			entry.TotalTenX += de.TotalTenX
			entry.TotalX += de.TotalX
			entry.EndsCompleted += de.EndsCompleted

			if de.SessionCode != nil && de.EndScores != nil {
				entry.Sessions = append(entry.Sessions, SessionScore{
					SessionCode: *de.SessionCode,
					SessionName: *de.SessionName,
					EndScores:   *de.EndScores,
				})
			}
		}

		// Convert map to slice and sort by total score
		leaderboard := make([]*Entry, 0, len(archerOrder))
		for _, uuid := range archerOrder {
			leaderboard = append(leaderboard, archerMap[uuid])
		}

		sort.Slice(leaderboard, func(i, j int) bool {
			if leaderboard[i].TotalScore != leaderboard[j].TotalScore {
				return leaderboard[i].TotalScore > leaderboard[j].TotalScore
			}
			if leaderboard[i].TotalTenX != leaderboard[j].TotalTenX {
				return leaderboard[i].TotalTenX > leaderboard[j].TotalTenX
			}
			return leaderboard[i].TotalX > leaderboard[j].TotalX
		})

		c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
	}
}

// GetSessionScores returns all scores for all archers in a specific session
func GetSessionScores(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")
		categoryID := c.Query("category_id")

		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId is required"})
			return
		}

		type ArrowScore struct {
			EndScoreUUID string `db:"end_score_uuid" json:"-"`
			ArrowNumber  int    `db:"arrow_number" json:"arrow_number"`
			Score        int    `db:"score" json:"score"`
			IsX          bool   `db:"is_x" json:"is_x"`
		}

		type EndScore struct {
			UUID          string `db:"uuid"`
			ParticipantUUID string `db:"participant_uuid" json:"participant_uuid"`
			EndNumber     int    `db:"end_number" json:"end_number"`
			TotalScoreEnd int    `db:"total_score_end" json:"total_score_end"`
			XCountEnd     int    `db:"x_count_end" json:"x_count_end"`
			TenCountEnd   int    `db:"ten_count_end" json:"ten_count_end"`
		}

		// Query building
		query := `
			SELECT qes.uuid, qes.participant_uuid, qes.end_number, qes.total_score_end, qes.x_count_end, qes.ten_count_end
			FROM qualification_end_scores qes
		`
		args := []interface{}{}

		if categoryID != "" {
			query += " JOIN event_participants ep ON qes.participant_uuid = ep.uuid"
			query += " WHERE qes.session_uuid = ? AND ep.category_id = ?"
			args = append(args, sessionID, categoryID)
		} else {
			query += " WHERE qes.session_uuid = ?"
			args = append(args, sessionID)
		}

		query += " ORDER BY qes.participant_uuid, qes.end_number ASC"

		var endScores []EndScore
		err := db.Select(&endScores, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch end scores", "details": err.Error()})
			return
		}

		// Fetch arrows for these ends
		var arrows []ArrowScore
		arrowQuery := `
			SELECT arrow_number, score, is_x, end_score_uuid
			FROM qualification_arrow_scores
			WHERE end_score_uuid IN (
				SELECT qes.uuid 
				FROM qualification_end_scores qes
		`
		if categoryID != "" {
			arrowQuery += " JOIN event_participants ep ON qes.participant_uuid = ep.uuid"
			arrowQuery += " WHERE qes.session_uuid = ? AND ep.category_id = ?"
		} else {
			arrowQuery += " WHERE qes.session_uuid = ?"
		}
		arrowQuery += ")"
		arrowQuery += " ORDER BY end_score_uuid, arrow_number ASC"

		err = db.Select(&arrows, arrowQuery, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch arrow scores", "details": err.Error()})
			return
		}

		// Grouping
		arrowsByEnd := make(map[string][]ArrowScore)
		for _, a := range arrows {
			arrowsByEnd[a.EndScoreUUID] = append(arrowsByEnd[a.EndScoreUUID], a)
		}

		type EndWithArrows struct {
			EndNumber     int          `json:"end_number"`
			TotalScoreEnd int          `json:"total_score_end"`
			XCountEnd     int          `json:"x_count_end"`
			TenCountEnd   int          `json:"ten_count_end"`
			Arrows        []ArrowScore `json:"arrows"`
		}

		type ArcherScores struct {
			ParticipantUUID string          `json:"participant_uuid"`
			Ends            []EndWithArrows `json:"ends"`
		}

		scoresByArcher := make(map[string]*ArcherScores)
		archerOrder := []string{}

		for _, es := range endScores {
			if _, ok := scoresByArcher[es.ParticipantUUID]; !ok {
				scoresByArcher[es.ParticipantUUID] = &ArcherScores{
					ParticipantUUID: es.ParticipantUUID,
					Ends:       []EndWithArrows{},
				}
				archerOrder = append(archerOrder, es.ParticipantUUID)
			}

			endWithArrows := EndWithArrows{
				EndNumber:     es.EndNumber,
				TotalScoreEnd: es.TotalScoreEnd,
				XCountEnd:     es.XCountEnd,
				TenCountEnd:   es.TenCountEnd,
				Arrows:        arrowsByEnd[es.UUID],
			}
			if endWithArrows.Arrows == nil {
				endWithArrows.Arrows = []ArrowScore{}
			}
			scoresByArcher[es.ParticipantUUID].Ends = append(scoresByArcher[es.ParticipantUUID].Ends, endWithArrows)
		}

		result := make([]*ArcherScores, 0, len(archerOrder))
		for _, archerID := range archerOrder {
			result = append(result, scoresByArcher[archerID])
		}

		c.JSON(http.StatusOK, gin.H{"scores": result})
	}
}

// GetSessionAssignments returns all archer assignments for a qualification session
func GetSessionAssignments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")
		categoryID := c.Query("category_id")

		fmt.Printf("[DEBUG] GetSessionAssignments - SessionID: %s, CategoryID: %s\n", sessionID, categoryID)

		type Assignment struct {
			UUID            string  `json:"uuid" db:"uuid"`
			ParticipantUUID string  `json:"participant_id" db:"participant_uuid"`
			TargetUUID      string  `json:"target_id" db:"target_uuid"`
			TargetName      string  `json:"target_name" db:"target_name"`
			ArcherName      string  `json:"archer_name" db:"archer_name"`
			ClubName        *string `json:"club_name" db:"club_name"`
		}

		var assignments []Assignment
		query := `
			SELECT 
				qta.uuid,
				qta.participant_uuid,
				qta.target_uuid,
				et.target_name,
				a.full_name as archer_name,
				c.name as club_name
			FROM qualification_target_assignments qta
			LEFT JOIN event_targets et ON qta.target_uuid = et.uuid
			LEFT JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs c ON a.club_id = c.uuid
			WHERE qta.session_uuid = ?
		`
		args := []interface{}{sessionID}

		if categoryID != "" {
			query += " AND ep.category_id = ?"
			args = append(args, categoryID)
		}

		query += " ORDER BY et.target_name ASC"

		err := db.Select(&assignments, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignments", "details": err.Error()})
			return
		}

		fmt.Printf("[DEBUG] GetSessionAssignments - Found %d assignments\n", len(assignments))

		c.JSON(http.StatusOK, gin.H{"assignments": assignments})
	}
}

// AutoAssignParticipants automatically assigns participants to targets.
// Participants are randomized; slots are filled target-by-target so each target
// is full (archers_per_target) before moving to the next target.
func AutoAssignParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")

		var req struct {
			CategoryID       string `json:"category_id" binding:"required"`
			StartTargetName  string `json:"start_target"`
			ArchersPerTarget int    `json:"archers_per_target"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.ArchersPerTarget == 0 {
			req.ArchersPerTarget = 4
		}

		// Get session details
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT event_uuid FROM qualification_sessions WHERE uuid = ?`, sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
			return
		}

		type Target struct {
			UUID       string `db:"uuid"`
			TargetName string `db:"target_name"`
		}
		var allTargets []Target
		err = db.Select(&allTargets, `
			SELECT uuid, target_name
			FROM event_targets
			WHERE event_uuid = ?
			ORDER BY (target_name + 0) ASC, target_name ASC
		`, eventUUID)

		if err != nil || len(allTargets) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No targets available"})
			return
		}

		// 1. Natural sort targets: 1A, 1B, 1C, 1D, 2A, 2B... (target number first, then letter)
		sort.Slice(allTargets, func(i, j int) bool {
			ni, _ := strconv.Atoi(strings.TrimRight(allTargets[i].TargetName, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
			nj, _ := strconv.Atoi(strings.TrimRight(allTargets[j].TargetName, "ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
			if ni != nj {
				return ni < nj
			}
			return allTargets[i].TargetName < allTargets[j].TargetName
		})

		// 2. Build available slots: fill one target completely (all archers_per_target positions) before the next.
		// "Taken" is per category: one physical target can be used by one archer per event category, so when
		// auto-assigning category 2 we only consider targets already assigned to this category (not category 1).
		// That way each category starts from the beginning (1A, 1B, 1C, 1D, 2A, ...).
		var existing []string
		db.Select(&existing, `
			SELECT qta.target_uuid
			FROM qualification_target_assignments qta
			INNER JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			WHERE qta.session_uuid = ? AND ep.category_id = ?
		`, sessionID, req.CategoryID)
		isTaken := make(map[string]bool)
		for _, e := range existing {
			isTaken[e] = true
		}

		letterOrder := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
		availableSlots := []Target{}
		startFound := (req.StartTargetName == "")

		for _, t := range allTargets {
			if !startFound {
				if t.TargetName == req.StartTargetName {
					startFound = true
				} else {
					continue
				}
			}

			letter := ""
			if len(t.TargetName) > 0 {
				letter = string(t.TargetName[len(t.TargetName)-1])
			}
			letterIdx := -1
			for i, l := range letterOrder {
				if l == letter {
					letterIdx = i
					break
				}
			}
			if letterIdx >= req.ArchersPerTarget {
				continue
			}
			if isTaken[t.UUID] {
				continue
			}
			availableSlots = append(availableSlots, t)
		}

		// 3. Get unassigned participants (no club ordering; we randomize next)
		type ParticipantWithClub struct {
			ParticipationUUID string  `db:"uuid"`
			ClubName          *string `db:"club_name"`
		}
		var participants []ParticipantWithClub
		err = db.Select(&participants, `
			SELECT ep.uuid, c.name as club_name
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs c ON a.club_id = c.uuid
			WHERE ep.category_id = ?
			AND ep.status = 'Terdaftar'
			AND ep.uuid NOT IN (SELECT participant_uuid FROM qualification_target_assignments WHERE session_uuid = ?)
			ORDER BY ep.uuid
		`, req.CategoryID, sessionID)

		if err != nil || len(participants) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "No participants to assign", "count": 0})
			return
		}

		// 4. Randomize participants
		rand.Shuffle(len(participants), func(i, j int) {
			participants[i], participants[j] = participants[j], participants[i]
		})

		// 5. Assign in order: slot order is already target-full-first (1A..1D, 2A..2D, ...)
		assignedCount := 0
		for i, archer := range participants {
			if i >= len(availableSlots) {
				break
			}
			target := availableSlots[i]
			assignmentUUID := uuid.New().String()
			_, err := db.Exec(`
				INSERT INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid)
				VALUES (?, ?, ?, ?)`,
				assignmentUUID, sessionID, archer.ParticipationUUID, target.UUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment", "details": err.Error()})
				return
			}
			assignedCount++
		}

		c.JSON(http.StatusOK, gin.H{"message": "Participants assigned successfully", "count": assignedCount})
	}
}


// DeleteQualificationAssignment deletes an archer assignment
func DeleteQualificationAssignment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")

		// Get session and participant UUIDs
		var sessionUUID, participantUUID string
		err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}
		err = db.Get(&participantUUID, `SELECT participant_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}

		// First delete all related arrow scores
		_, err = db.Exec(`
			DELETE FROM qualification_arrow_scores 
			WHERE end_score_uuid IN (
				SELECT uuid FROM qualification_end_scores 
				WHERE session_uuid = ? AND participant_uuid = ?
			)`, sessionUUID, participantUUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete arrow scores"})
			return
		}

		// Delete end scores
		_, err = db.Exec(`DELETE FROM qualification_end_scores WHERE session_uuid = ? AND participant_uuid = ?`, sessionUUID, participantUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete end scores"})
			return
		}

		// Then delete the assignment
		result, err := db.Exec("DELETE FROM qualification_target_assignments WHERE uuid = ?", assignmentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete assignment"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Assignment deleted successfully"})
	}
}

// CreateBulkTargetAssignments creates multiple target assignments for a category
func CreateBulkTargetAssignments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		sessionID := c.Param("sessionId")
		if eventID == "" || sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "eventId and sessionId are required"})
			return
		}

		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		var req struct {
			CategoryID  string `json:"category_id" binding:"required"`
			Assignments []struct {
				ParticipantID string `json:"participant_id" binding:"required"`
				TargetID      string `json:"target_id" binding:"required"`
			} `json:"assignments" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		fmt.Printf("[DEBUG] CreateBulkTargetAssignments - Attempting to assign for CategoryID: %s\n", req.CategoryID)

		// Validate session belongs to event
		var sessionUUID string
		err = db.Get(&sessionUUID, `
			SELECT uuid FROM qualification_sessions 
			WHERE uuid = ? AND event_uuid = ?
			LIMIT 1
		`, sessionID, eventUUID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Qualification session not found for this event"})
			return
		}

		// Start transaction
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		successCount := 0
		errors := []map[string]interface{}{}

		for _, assignment := range req.Assignments {
			assignmentUUID := uuid.New().String()

			// Validate participation exists
			var count int
			err := tx.Get(&count, `
				SELECT COUNT(*) FROM event_participants 
				WHERE uuid = ? AND category_id = ?
			`, assignment.ParticipantID, req.CategoryID)
			if err != nil || count == 0 {
				errors = append(errors, map[string]interface{}{
					"participant_id": assignment.ParticipantID,
					"error":          "Participant not found in this category",
				})
				continue
			}

			// 1. Delete existing assignment for this participant in this session to ensure clean move
			_, err = tx.Exec(`
				DELETE FROM qualification_target_assignments 
				WHERE session_uuid = ? AND participant_uuid = ?
			`, sessionUUID, assignment.ParticipantID)
			if err != nil {
				errors = append(errors, map[string]interface{}{
					"participant_id": assignment.ParticipantID,
					"error":          "Failed to clear existing assignment: " + err.Error(),
				})
				continue
			}

			// 2. Delete existing assignment for this target in this session (evict current occupant if any)
			_, err = tx.Exec(`
				DELETE FROM qualification_target_assignments 
				WHERE session_uuid = ? AND target_uuid = ?
			`, sessionUUID, assignment.TargetID)
			if err != nil {
				errors = append(errors, map[string]interface{}{
					"participant_id": assignment.ParticipantID,
					"target_id":      assignment.TargetID,
					"error":          "Failed to clear target assignment: " + err.Error(),
				})
				continue
			}

			// 3. Insert new assignment
			_, err = tx.Exec(`
				INSERT INTO qualification_target_assignments 
				(uuid, session_uuid, participant_uuid, target_uuid, created_at, updated_at)
				VALUES (?, ?, ?, ?, NOW(), NOW())
			`, assignmentUUID, sessionUUID, assignment.ParticipantID, assignment.TargetID)
			if err != nil {
				errors = append(errors, map[string]interface{}{
					"participant_id": assignment.ParticipantID,
					"target_id":      assignment.TargetID,
					"error":          "Failed to create assignment: " + err.Error(),
				})
				continue
			}

			successCount++
			fmt.Printf("[DEBUG] Assignment successful for participant %s to target %s\n", assignment.ParticipantID, assignment.TargetID)
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		if successCount == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "No assignments created",
				"errors":  errors,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "Assignments created successfully",
			"success_count": successCount,
			"errors":        errors,
		})
	}
}

// ResetSessionAssignments removes all assignments and scores for a category in a session
func ResetSessionAssignments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")
		
		var req struct {
			CategoryID string `json:"category_id" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
			return
		}

		categoryID := req.CategoryID

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// 1. Delete arrow scores for this category and session
		_, err = tx.Exec(`
			DELETE FROM qualification_arrow_scores 
			WHERE end_score_uuid IN (
				SELECT qes.uuid 
				FROM qualification_end_scores qes
				JOIN event_participants ep ON qes.participant_uuid = ep.uuid
				WHERE qes.session_uuid = ? AND ep.category_id = ?
			)`, sessionID, categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete arrow scores", "details": err.Error()})
			return
		}

		// 2. Delete end scores
		_, err = tx.Exec(`
			DELETE qes FROM qualification_end_scores qes
			JOIN event_participants ep ON qes.participant_uuid = ep.uuid
			WHERE qes.session_uuid = ? AND ep.category_id = ?`,
			sessionID, categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete end scores", "details": err.Error()})
			return
		}

		// 3. Delete assignments
		_, err = tx.Exec(`
			DELETE qta FROM qualification_target_assignments qta
			JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			WHERE qta.session_uuid = ? AND ep.category_id = ?`,
			sessionID, categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete assignments", "details": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Assignments reset successfully"})
	}
}

// SwapTargetAssignments swaps targets between two participants in a session
func SwapTargetAssignments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")

		var req struct {
			ParticipantA string `json:"participant_a" binding:"required"`
			ParticipantB string `json:"participant_b" binding:"required"`
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

		// Get assignments for both to verify they exist and get their current targets
		var targetA, targetB string
		err = tx.Get(&targetA, "SELECT target_uuid FROM qualification_target_assignments WHERE session_uuid = ? AND participant_uuid = ?", sessionID, req.ParticipantA)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment for participant A not found"})
			return
		}

		err = tx.Get(&targetB, "SELECT target_uuid FROM qualification_target_assignments WHERE session_uuid = ? AND participant_uuid = ?", sessionID, req.ParticipantB)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment for participant B not found"})
			return
		}

		// 1. Delete Participant A's assignment to free up Target A in the unique index
		_, err = tx.Exec("DELETE FROM qualification_target_assignments WHERE session_uuid = ? AND participant_uuid = ?", sessionID, req.ParticipantA)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to detach participant A", "details": err.Error()})
			return
		}

		// 2. Move Participant B to Target A
		_, err = tx.Exec("UPDATE qualification_target_assignments SET target_uuid = ? WHERE session_uuid = ? AND participant_uuid = ?", targetA, sessionID, req.ParticipantB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move participant B", "details": err.Error()})
			return
		}

		// 3. Re-insert Participant A into Target B
		assignmentUUID := uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid, created_at, updated_at)
			VALUES (?, ?, ?, ?, NOW(), NOW())`,
			assignmentUUID, sessionID, req.ParticipantA, targetB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to re-attach participant A", "details": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Targets swapped successfully"})
	}
}
