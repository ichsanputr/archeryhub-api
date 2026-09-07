package handler

import (
	"Archeris-api/models"
	"Archeris-api/utils"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetQualificationSessions returns all sessions for an event
func GetQualificationSessions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "eventId wajib diisi"})
			return
		}

		// Resolve event UUID (allow slug)
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type SessionWithCount struct {
			UUID             string   `db:"uuid" json:"uuid"`
			EventUUID        string   `db:"event_uuid" json:"event_uuid"`
			SessionCode      string   `db:"session_code" json:"session_code"`
			SessionDate      *string  `db:"session_date" json:"session_date"`
			Name             string   `db:"name" json:"name"`
			StartTime        *string  `db:"start_time" json:"start_time"`
			EndTime          *string  `db:"end_time" json:"end_time"`
			TotalEnds        int      `db:"total_ends" json:"total_ends"`
			ArrowsPerEnd     int      `db:"arrows_per_end" json:"arrows_per_end"`
			CreatedAt        *string  `db:"created_at" json:"created_at"`
			UpdatedAt        *string  `db:"updated_at" json:"updated_at"`
			ParticipantCount int      `db:"participant_count" json:"participant_count"`
			CategoryIDs      string   `db:"category_ids" json:"-"`
			CategoryList     []string `json:"category_ids"`
			IsLocked         bool     `db:"is_locked" json:"is_locked"`
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
				COALESCE(qs.is_locked, 0) as is_locked,
				COUNT(DISTINCT qta.participant_uuid) as participant_count,
				COALESCE(GROUP_CONCAT(DISTINCT qsc.category_uuid), '') as category_ids
			FROM qualification_sessions qs
			LEFT JOIN qualification_target_assignments qta ON qs.uuid = qta.session_uuid
			LEFT JOIN qualification_session_categories qsc ON qs.uuid = qsc.session_uuid
			WHERE qs.event_uuid = ?
			GROUP BY qs.uuid, qs.event_uuid, qs.session_code, qs.session_date, qs.name, qs.start_time, qs.end_time, qs.total_ends, qs.arrows_per_end, qs.created_at, qs.updated_at, qs.is_locked
			ORDER BY qs.session_date ASC, qs.start_time ASC, qs.created_at ASC
		`, eventUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data sesi", "details": err.Error()})
			return
		}

		for i := range sessions {
			if sessions[i].CategoryIDs != "" {
				sessions[i].CategoryList = strings.Split(sessions[i].CategoryIDs, ",")
			} else {
				sessions[i].CategoryList = []string{}
			}
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data sesi", "details": err.Error()})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		var req struct {
			Name         string   `json:"name" binding:"required"`
			SessionDate  *string  `json:"session_date"`
			StartTime    *string  `json:"start_time"`
			EndTime      *string  `json:"end_time"`
			TotalEnds    int      `json:"total_ends"`
			ArrowsPerEnd int      `json:"arrows_per_end"`
			CategoryIDs  []string `json:"category_ids"`
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

		// Generate clean short session code (e.g., QS-7K9M2)
		var sessionCode string
		for {
			sessionCode = utils.GenerateShortCode("QS", 5)
			var count int
			err := db.Get(&count, "SELECT COUNT(*) FROM qualification_sessions WHERE session_code = ?", sessionCode)
			if err == nil && count == 0 {
				break
			}
		}

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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		newUUID := uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO qualification_sessions (uuid, event_uuid, session_code, session_date, name, start_time, end_time, total_ends, arrows_per_end)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			newUUID, eventUUID, sessionCode, req.SessionDate, req.Name, finalStartTime, finalEndTime, req.TotalEnds, req.ArrowsPerEnd)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat data sesi", "details": err.Error()})
			return
		}

		// Insert categories
		for _, catID := range req.CategoryIDs {
			_, err = tx.Exec(`INSERT INTO qualification_session_categories (session_uuid, category_uuid) VALUES (?, ?)`, newUUID, catID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan kategori ke sesi", "details": err.Error()})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":      "Sesi berhasil dibuat",
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
			Name         string   `json:"name" binding:"required"`
			SessionDate  *string  `json:"session_date"`
			StartTime    *string  `json:"start_time"`
			EndTime      *string  `json:"end_time"`
			TotalEnds    int      `json:"total_ends"`
			ArrowsPerEnd int      `json:"arrows_per_end"`
			CategoryIDs  []string `json:"category_ids"`
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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec(`
			UPDATE qualification_sessions 
			SET name = ?, session_date = ?, start_time = ?, end_time = ?, total_ends = ?, arrows_per_end = ?, updated_at = NOW()
			WHERE uuid = ?`,
			req.Name, req.SessionDate, finalStartTime, finalEndTime, req.TotalEnds, req.ArrowsPerEnd, sessionUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui data sesi", "details": err.Error()})
			return
		}

		// Update categories: Delete old and insert new
		_, err = tx.Exec(`DELETE FROM qualification_session_categories WHERE session_uuid = ?`, sessionUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui kategori", "details": err.Error()})
			return
		}

		for _, catID := range req.CategoryIDs {
			_, err = tx.Exec(`INSERT INTO qualification_session_categories (session_uuid, category_uuid) VALUES (?, ?)`, sessionUUID, catID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan kembali kategori", "details": err.Error()})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Sesi berhasil diperbarui"})
	}
}

// ToggleLockQualificationSession locks or unlocks scoring for a qualification session
func ToggleLockQualificationSession(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId wajib diisi"})
			return
		}

		var isLocked bool
		err := db.Get(&isLocked, `SELECT COALESCE(is_locked, 0) FROM qualification_sessions WHERE uuid = ? OR session_code = ?`, sessionID, sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sesi kualifikasi tidak ditemukan"})
			return
		}

		newLockState := !isLocked
		_, err = db.Exec(`UPDATE qualification_sessions SET is_locked = ? WHERE uuid = ? OR session_code = ?`, newLockState, sessionID, sessionID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengubah status kunci sesi"})
			return
		}

		msg := "Sesi kualifikasi berhasil dikunci"
		if !newLockState {
			msg = "Sesi kualifikasi berhasil dibuka kembali (unlocked)"
		}

		c.JSON(http.StatusOK, gin.H{
			"message":   msg,
			"is_locked": newLockState,
		})
	}
}

// UpdateQualificationScore updates end scores for an assignment (supports batch)
// @Summary Update Qualification Score
// @Description Submit or update scores for a specific qualification assignment (end-by-end)
// @Tags Mobile - Scorekeeper
// @Accept json
// @Produce json
// @Param assignmentId path string true "Assignment UUID"
// @Param request body object true "Score Update Payload (models.ScoreUpdateRequest or models.ScoreBatchUpdateRequest)"
// @Success 200 {object} MessageResponse
// @Router /mobile/qualification/scoring/scores/{assignmentId} [post]
func UpdateQualificationScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")

		var sessionUUID, participantUUID string
		if err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}
		if err := db.Get(&participantUUID, `SELECT participant_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}

		var isLocked bool
		_ = db.Get(&isLocked, `SELECT COALESCE(is_locked, 0) FROM qualification_sessions WHERE uuid = ?`, sessionUUID)
		if isLocked {
			c.JSON(http.StatusForbidden, gin.H{"error": "Sesi kualifikasi telah dikunci oleh panitia/wasit. Perubahan skor tidak diperbolehkan."})
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format permintaan tidak valid: 'ends' atau 'end_number' wajib diisi"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data skor lama"})
			return
		}

		existingMap := make(map[int]string)
		for _, ee := range existingEnds {
			existingMap[ee.EndNumber] = ee.UUID
		}

		if len(ends) == 0 {
			// Clear all existing end scores and arrows for this participant in this session
			var oldUUIDs []string
			for _, ee := range existingEnds {
				oldUUIDs = append(oldUUIDs, ee.UUID)
			}
			if len(oldUUIDs) > 0 {
				query, args, inErr := sqlx.In(`DELETE FROM qualification_arrow_scores WHERE end_score_uuid IN (?)`, oldUUIDs)
				if inErr == nil {
					query = tx.Rebind(query)
					_, _ = tx.Exec(query, args...)
				}
				_, _ = tx.Exec(`DELETE FROM qualification_end_scores WHERE session_uuid = ? AND participant_uuid = ?`, sessionUUID, participantUUID)
			}
		}

		var allEndScoreUUIDs []string
		var arrowValues []interface{}
		arrowCount := 0

		for _, end := range ends {
			if len(end.Arrows) == 0 {
				// If this end is sent as empty, delete it if it existed
				if currentEndScoreUUID, exists := existingMap[end.EndNumber]; exists {
					_, _ = tx.Exec(`DELETE FROM qualification_arrow_scores WHERE end_score_uuid = ?`, currentEndScoreUUID)
					_, _ = tx.Exec(`DELETE FROM qualification_end_scores WHERE uuid = ?`, currentEndScoreUUID)
				}
				continue
			}

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
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui skor babak"})
					return
				}
			} else {
				// Insert end score
				currentEndScoreUUID = uuid.New().String()
				_, err = tx.Exec(`INSERT INTO qualification_end_scores (uuid, session_uuid, participant_uuid, end_number, total_score_end, x_count_end, ten_count_end) VALUES (?, ?, ?, ?, ?, ?, ?)`,
					currentEndScoreUUID, sessionUUID, participantUUID, end.EndNumber, total, xCount, tenCount)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat skor babak baru"})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyiapkan pembersihan data anak panah"})
				return
			}
			query = tx.Rebind(query)
			_, err = tx.Exec(query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data anak panah lama"})
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan skor anak panah"})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan skor"})
			return
		}

		// Log for scorekeeper audit
		userTypeContext, _ := c.Get("user_type")
		if userTypeContext == "scorekeeper" {
			userID, _ := c.Get("user_id")
			orgID, _ := c.Get("org_id")
			details, _ := json.Marshal(raw)

			var eventUUID string
			_ = db.Get(&eventUUID, "SELECT event_uuid FROM qualification_sessions WHERE uuid = ?", sessionUUID)

			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "update_qualification_score", string(details), c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Skor berhasil diperbarui"})
	}
}

// GetQualificationAssignmentScores returns all saved end scores for a single assignment
func GetQualificationAssignmentScores(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")
		if assignmentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assignmentId wajib diisi"})
			return
		}

		// Get session and participant from assignment
		var sessionUUID, participantUUID string
		err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}
		err = db.Get(&participantUUID, `SELECT participant_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data skor babak"})
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
		v, err := strconv.Atoi(arrow)
		// Valid integer arrow values are 1-9; reject anything out of range
		if err != nil || v < 0 || v > 9 {
			return 0, 0, 0
		}
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "categoryId wajib diisi"})
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
				COALESCE(ec.category_name_custom, CONCAT(bt.name, ' ', ag.name)) as category_name,
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
			JOIN (
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
			WHERE ep.category_id = ? AND score_summary.ends_completed > 0
			ORDER BY participant_uuid, qs.created_at ASC`,
			categoryID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data klasemen", "details": err.Error()})
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

		// Convert map to slice (only include archers with completed ends / recorded scores) and sort by total score
		leaderboard := make([]*Entry, 0, len(archerOrder))
		for _, uuid := range archerOrder {
			if archerMap[uuid].EndsCompleted > 0 {
				leaderboard = append(leaderboard, archerMap[uuid])
			}
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId wajib diisi"})
			return
		}

		type ArrowScore struct {
			EndScoreUUID string `db:"end_score_uuid" json:"-"`
			ArrowNumber  int    `db:"arrow_number" json:"arrow_number"`
			Score        int    `db:"score" json:"score"`
			IsX          bool   `db:"is_x" json:"is_x"`
		}

		type EndScore struct {
			UUID            string `db:"uuid"`
			ParticipantUUID string `db:"participant_uuid" json:"participant_uuid"`
			EndNumber       int    `db:"end_number" json:"end_number"`
			TotalScoreEnd   int    `db:"total_score_end" json:"total_score_end"`
			XCountEnd       int    `db:"x_count_end" json:"x_count_end"`
			TenCountEnd     int    `db:"ten_count_end" json:"ten_count_end"`
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data skor babak", "details": err.Error()})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil skor anak panah", "details": err.Error()})
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
					Ends:            []EndWithArrows{},
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penempatan", "details": err.Error()})
			return
		}

		fmt.Printf("[DEBUG] GetSessionAssignments - Found %d assignments\n", len(assignments))

		c.JSON(http.StatusOK, gin.H{"assignments": assignments})
	}
}

// GetMyEventTarget returns the archer's own target assignments for all sessions in an event.
// Called by archer from dashboard: GET /events/:id/my-target
func GetMyEventTarget(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		userID, _ := c.Get("user_id")
		userIDStr := fmt.Sprintf("%v", userID)
		userEmailVal, _ := c.Get("email")
		userEmail := fmt.Sprintf("%v", userEmailVal)

		// Get archer UUID for this user (if any)
		var archerID string
		_ = db.Get(&archerID, `SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?) LIMIT 1`, userIDStr, userIDStr, userEmail)
		if archerID == "" {
			archerID = userIDStr
		}

		type SessionTarget struct {
			SessionID    string  `json:"session_id" db:"session_id"`
			SessionName  string  `json:"session_name" db:"session_name"`
			SessionOrder string  `json:"session_order" db:"session_order"`
			StartTime    *string `json:"start_time" db:"start_time"`
			EndTime      *string `json:"end_time" db:"end_time"`
			TargetName   string  `json:"target_name" db:"target_name"`
			TargetBoard  *string `json:"target_board" db:"target_board"`
			CategoryName string  `json:"category_name" db:"category_name"`
			AssignmentID string  `json:"assignment_id" db:"assignment_id"`
		}

		var targets []SessionTarget
		query := `
			SELECT
				qs.uuid        AS session_id,
				qs.name        AS session_name,
				COALESCE(qs.session_code, 'S1') AS session_order,
				DATE_FORMAT(qs.start_time, '%H:%i') AS start_time,
				DATE_FORMAT(qs.end_time, '%H:%i') AS end_time,
				COALESCE(NULLIF(et.target_name, ''), NULLIF(ep.target_name, ''), CONCAT('Target ', COALESCE(NULLIF(qta.target_board_id, ''), '01'))) AS target_name,
				COALESCE(qta.target_board_id, ep.back_number, 'A') AS target_board,
				COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' - ', COALESCE(rag.name, ''))) AS category_name,
				qta.uuid AS assignment_id
			FROM qualification_target_assignments qta
			JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			JOIN events e ON (e.uuid = ep.event_id OR e.slug = ep.event_id)
			LEFT JOIN event_targets et ON qta.target_uuid = et.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
			LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
			WHERE (e.uuid = ? OR e.slug = ?)
			  AND (
					ep.archer_id = ? 
					OR ep.archer_id = ?
					OR ep.archer_id IN (SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?))
					OR ep.archer_id IN (SELECT id FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?))
					OR ep.uuid = ?
					OR ep.archer_id IN (SELECT uuid FROM users WHERE id = ? OR uuid = ? OR email = ?)
			  )
			ORDER BY qs.start_time ASC
		`
		err := db.Select(&targets, query, eventID, eventID, archerID, userIDStr, userIDStr, userIDStr, userEmail, userIDStr, userIDStr, userEmail, userIDStr, userIDStr, userIDStr, userEmail)
		if err != nil || len(targets) == 0 {
			// Fallback: check if participant has direct target_name assigned or qualification sessions
			fallbackQuery := `
				SELECT
					COALESCE(qs.uuid, ep.uuid) AS session_id,
					COALESCE(qs.name, 'Sesi Kualifikasi') AS session_name,
					COALESCE(qs.session_code, 'S1') AS session_order,
					DATE_FORMAT(qs.start_time, '%H:%i') AS start_time,
					DATE_FORMAT(qs.end_time, '%H:%i') AS end_time,
					COALESCE(NULLIF(ep.target_name, ''), 'Target 01') AS target_name,
					COALESCE(NULLIF(ep.back_number, ''), 'A') AS target_board,
					COALESCE(ec.category_name_custom, CONCAT(COALESCE(rbt.name, ''), ' - ', COALESCE(rag.name, ''))) AS category_name,
					ep.uuid AS assignment_id
				FROM event_participants ep
				JOIN events e ON (e.uuid = ep.event_id OR e.slug = ep.event_id)
				LEFT JOIN qualification_sessions qs ON qs.event_uuid = e.uuid
				LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
				LEFT JOIN ref_bow_types rbt ON ec.division_uuid = rbt.uuid
				LEFT JOIN ref_age_groups rag ON ec.category_uuid = rag.uuid
				WHERE (e.uuid = ? OR e.slug = ?)
				  AND (
						ep.archer_id = ? 
						OR ep.archer_id = ?
						OR ep.archer_id IN (SELECT uuid FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?))
						OR ep.archer_id IN (SELECT id FROM archers WHERE uuid = ? OR id = ? OR (email != '' AND email = ?))
						OR ep.uuid = ?
						OR ep.archer_id IN (SELECT uuid FROM users WHERE id = ? OR uuid = ? OR email = ?)
				  )
				LIMIT 1
			`
			_ = db.Select(&targets, fallbackQuery, eventID, eventID, archerID, userIDStr, userIDStr, userIDStr, userEmail, userIDStr, userIDStr, userEmail, userIDStr, userIDStr, userIDStr, userEmail)
		}

		if targets == nil {
			targets = []SessionTarget{}
		}

		c.JSON(http.StatusOK, gin.H{"targets": targets})
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
			DrawType         string `json:"draw_type"`   // "standard" or "field"
			AssignMode       string `json:"assign_mode"` // "unassigned_only" (default) or "all"
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if req.ArchersPerTarget == 0 {
			req.ArchersPerTarget = 4
		}
		if req.DrawType == "" {
			req.DrawType = "standard"
		}
		if req.AssignMode == "" {
			req.AssignMode = "unassigned_only"
		}

		// Get session details
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT event_uuid FROM qualification_sessions WHERE uuid = ?`, sessionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sesi tidak ditemukan"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Tidak ada target yang tersedia"})
			return
		}

		// Check if scoring has already started for this category+session
		var endsCompleted int
		db.Get(&endsCompleted, `
			SELECT COUNT(*) FROM qualification_end_scores
			WHERE session_uuid = ?
			  AND participant_uuid IN (
			    SELECT uuid FROM event_participants WHERE category_id = ?
			  )
		`, sessionID, req.CategoryID)

		// If scores already exist and user asked for "all", force "unassigned_only" mode to protect active archers
		if endsCompleted > 0 && req.AssignMode == "all" {
			req.AssignMode = "unassigned_only"
		}

		// 1. If mode is "all" and no scores exist, clear existing assignments for this category in this session
		if req.AssignMode == "all" && endsCompleted == 0 {
			if _, err = db.Exec(`
				DELETE FROM qualification_target_assignments
				WHERE session_uuid = ?
				  AND participant_uuid IN (
				    SELECT uuid FROM event_participants WHERE category_id = ?
				  )
			`, sessionID, req.CategoryID); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data penempatan sebelumnya", "details": err.Error()})
				return
			}
		}

		// 2. Build map of targets grouped by number
		type TargetGroup struct {
			Number int
			Slots  map[string]Target
		}
		targetGroupsMap := make(map[int]*TargetGroup)
		for _, t := range allTargets {
			numPart := strings.TrimRight(t.TargetName, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
			num, _ := strconv.Atoi(numPart)
			letter := strings.TrimPrefix(t.TargetName, numPart)

			if targetGroupsMap[num] == nil {
				targetGroupsMap[num] = &TargetGroup{
					Number: num,
					Slots:  make(map[string]Target),
				}
			}
			targetGroupsMap[num].Slots[letter] = t
		}

		// Get sorted target numbers
		var targetNumbers []int
		for num := range targetGroupsMap {
			targetNumbers = append(targetNumbers, num)
		}
		sort.Ints(targetNumbers)

		// 3. Build available slots based on Draw Type
		var existing []string
		if req.AssignMode == "unassigned_only" {
			// In unassigned_only mode, all currently assigned targets in this session are taken
			db.Select(&existing, `
				SELECT qta.target_uuid
				FROM qualification_target_assignments qta
				WHERE qta.session_uuid = ?
			`, sessionID)
		} else {
			// In "all" mode (0 scores), targets taken by other categories are preserved
			db.Select(&existing, `
				SELECT qta.target_uuid
				FROM qualification_target_assignments qta
				JOIN event_participants ep ON qta.participant_uuid = ep.uuid
				WHERE qta.session_uuid = ? AND ep.category_id != ?
			`, sessionID, req.CategoryID)
		}

		isTaken := make(map[string]bool)
		for _, e := range existing {
			isTaken[e] = true
		}

		// Sequence for Standard Draw (A-C-B-D style)
		letterSequence := []string{"A", "C", "B", "D", "E", "F", "G", "H"}
		if req.ArchersPerTarget <= 2 {
			letterSequence = []string{"A", "B"}
		}

		availableSlots := []Target{}
		startTargetNum := 0
		if req.StartTargetName != "" {
			numPart := strings.TrimRight(req.StartTargetName, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
			startTargetNum, _ = strconv.Atoi(numPart)
		}

		step := 1
		if req.DrawType == "field" {
			step = 2
		}

		for i := 0; i < len(targetNumbers); i += step {
			num := targetNumbers[i]
			if num < startTargetNum {
				continue
			}

			group := targetGroupsMap[num]
			// Fill in specific letter sequence
			for _, letter := range letterSequence {
				target, exists := group.Slots[letter]
				if !exists {
					continue
				}

				// Ensure index of letter < ArchersPerTarget
				idxInAlphabet := int(letter[0] - 'A')
				if idxInAlphabet >= req.ArchersPerTarget {
					continue
				}

				if !isTaken[target.UUID] {
					availableSlots = append(availableSlots, target)
				}
			}
		}

		// 4. Get participants for this category
		type ParticipantWithClub struct {
			ParticipationUUID string  `db:"uuid"`
			ClubName          *string `db:"club_name"`
		}
		var participants []ParticipantWithClub

		if req.AssignMode == "unassigned_only" {
			// Fetch only participants who are NOT yet assigned in this session
			err = db.Select(&participants, `
				SELECT ep.uuid, c.name as club_name
				FROM event_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN clubs c ON a.club_id = c.uuid
				WHERE ep.category_id = ?
				  AND ep.uuid NOT IN (
				    SELECT participant_uuid FROM qualification_target_assignments WHERE session_uuid = ?
				  )
				ORDER BY ep.uuid
			`, req.CategoryID, sessionID)
		} else {
			// Fetch ALL participants for this category
			err = db.Select(&participants, `
				SELECT ep.uuid, c.name as club_name
				FROM event_participants ep
				JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN clubs c ON a.club_id = c.uuid
				WHERE ep.category_id = ?
				ORDER BY ep.uuid
			`, req.CategoryID)
		}

		if err != nil || len(participants) == 0 {
			c.JSON(http.StatusOK, gin.H{"message": "Tidak ada peserta untuk ditempatkan", "count": 0})
			return
		}

		// Pre-fetch all board numbers for targets in this event
		type TargetBoardInfo struct {
			UUID        string `db:"uuid"`
			BoardNumber int    `db:"board_number"`
		}
		var targetsInfo []TargetBoardInfo
		err = db.Select(&targetsInfo, `SELECT uuid, board_number FROM event_targets WHERE event_uuid = ?`, eventUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch targets info", "details": err.Error()})
			return
		}
		targetUUIDToBoardNumber := make(map[string]int)
		for _, info := range targetsInfo {
			targetUUIDToBoardNumber[info.UUID] = info.BoardNumber
		}

		// Pre-fetch target boards for this session and category
		type TargetBoardQual struct {
			UUID        string `db:"uuid"`
			BoardNumber int    `db:"board_number"`
		}
		var boardsQual []TargetBoardQual
		err = db.Select(&boardsQual, `SELECT uuid, board_number FROM target_board_qualification WHERE session_uuid = ? AND category_uuid = ?`, sessionID, req.CategoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch qualification boards", "details": err.Error()})
			return
		}
		boardNumberToBoardUUID := make(map[int]string)
		for _, b := range boardsQual {
			boardNumberToBoardUUID[b.BoardNumber] = b.UUID
		}

		// 5. Randomize participants with Club Separation (Tournament Fairness / World Archery Rule)
		// Group participants by club and interleave them so same-club archers are placed on different target boards
		clubMap := make(map[string][]ParticipantWithClub)
		var clubNames []string
		for _, p := range participants {
			cName := "Independent"
			if p.ClubName != nil && strings.TrimSpace(*p.ClubName) != "" {
				cName = strings.TrimSpace(*p.ClubName)
			}
			if len(clubMap[cName]) == 0 {
				clubNames = append(clubNames, cName)
			}
			clubMap[cName] = append(clubMap[cName], p)
		}

		// Shuffle club order and members within each club
		rand.Shuffle(len(clubNames), func(i, j int) {
			clubNames[i], clubNames[j] = clubNames[j], clubNames[i]
		})
		for _, cName := range clubNames {
			group := clubMap[cName]
			rand.Shuffle(len(group), func(i, j int) {
				group[i], group[j] = group[j], group[i]
			})
			clubMap[cName] = group
		}

		// Round-Robin across clubs to interleave archers
		var separatedParticipants []ParticipantWithClub
		maxInAnyClub := 0
		for _, group := range clubMap {
			if len(group) > maxInAnyClub {
				maxInAnyClub = len(group)
			}
		}

		for round := 0; round < maxInAnyClub; round++ {
			for _, cName := range clubNames {
				group := clubMap[cName]
				if round < len(group) {
					separatedParticipants = append(separatedParticipants, group[round])
				}
			}
		}
		participants = separatedParticipants

		// Start Transaction
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction", "details": err.Error()})
			return
		}
		defer tx.Rollback()

		// 6. Assign in order: slot order is already target-full-first (1A..1D, 2A..2D, ...)
		assignedCount := 0
		for i, archer := range participants {
			if i >= len(availableSlots) {
				break
			}
			target := availableSlots[i]

			boardNumber := targetUUIDToBoardNumber[target.UUID]
			
			var targetBoardUUID sql.NullString
			if uuidVal, ok := boardNumberToBoardUUID[boardNumber]; ok {
				targetBoardUUID.String = uuidVal
				targetBoardUUID.Valid = true
			}

			assignmentUUID := uuid.New().String()
			_, err := tx.Exec(`
				INSERT INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid, target_board_id)
				VALUES (?, ?, ?, ?, ?)`,
				assignmentUUID, sessionID, archer.ParticipationUUID, target.UUID, targetBoardUUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment", "details": err.Error()})
				return
			}
			assignedCount++
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Peserta berhasil ditempatkan", "count": assignedCount})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}
		err = db.Get(&participantUUID, `SELECT participant_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}

		// Delete only the assignment, keeping recorded scores safe attached to the participant
		result, err := db.Exec("DELETE FROM qualification_target_assignments WHERE uuid = ?", assignmentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete assignment"})
			return
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan target tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Penempatan berhasil dihapus"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
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
			var boardNumber int
			tx.Get(&boardNumber, "SELECT board_number FROM event_targets WHERE uuid = ?", assignment.TargetID)

			var targetBoardUUID sql.NullString
			tx.Get(&targetBoardUUID, "SELECT uuid FROM target_board_qualification WHERE session_uuid = ? AND category_uuid = ? AND board_number = ?",
				sessionUUID, req.CategoryID, boardNumber)

			_, err = tx.Exec(`
				INSERT INTO qualification_target_assignments 
				(uuid, session_uuid, participant_uuid, target_uuid, target_board_id, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, NOW(), NOW())
			`, assignmentUUID, sessionUUID, assignment.ParticipantID, assignment.TargetID, targetBoardUUID)
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		if successCount == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"message": "No assignments created",
				"errors":  errors,
			})
			return
		}

		// Log for scorekeeper audit
		userTypeContext, _ := c.Get("user_type")
		if userTypeContext == "scorekeeper" {
			userID, _ := c.Get("user_id")
			orgID, _ := c.Get("org_id")

			var eventUUID string
			_ = db.Get(&eventUUID, "SELECT event_uuid FROM qualification_sessions WHERE uuid = ?", sessionUUID)

			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "auto_assign_participants", "Auto-assigned participants in session: "+sessionID, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{
			"message":       "Penempatan berhasil dibuat",
			"success_count": successCount,
			"errors":        errors,
		})
	}
}

// ResetSessionAssignments removes assignments for unscored participants for a category in a session
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

		// Count participants with scores in this session + category
		var scoredCount int
		db.Get(&scoredCount, `
			SELECT COUNT(DISTINCT qes.participant_uuid)
			FROM qualification_end_scores qes
			JOIN event_participants ep ON qes.participant_uuid = ep.uuid
			WHERE qes.session_uuid = ? AND ep.category_id = ?
		`, sessionID, categoryID)

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// ONLY delete assignments for participants who DO NOT have scores yet
		var res sql.Result
		if scoredCount > 0 {
			res, err = tx.Exec(`
				DELETE qta FROM qualification_target_assignments qta
				JOIN event_participants ep ON qta.participant_uuid = ep.uuid
				WHERE qta.session_uuid = ? AND ep.category_id = ?
				  AND ep.uuid NOT IN (
				    SELECT DISTINCT participant_uuid FROM qualification_end_scores WHERE session_uuid = ?
				  )
			`, sessionID, categoryID, sessionID)
		} else {
			res, err = tx.Exec(`
				DELETE qta FROM qualification_target_assignments qta
				JOIN event_participants ep ON qta.participant_uuid = ep.uuid
				WHERE qta.session_uuid = ? AND ep.category_id = ?`,
				sessionID, categoryID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset assignments", "details": err.Error()})
			return
		}

		rowsAffected, _ := res.RowsAffected()

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		// Log for scorekeeper audit
		userTypeContext, _ := c.Get("user_type")
		if userTypeContext == "scorekeeper" {
			userID, _ := c.Get("user_id")
			orgID, _ := c.Get("org_id")

			var eventUUID string
			_ = db.Get(&eventUUID, "SELECT event_uuid FROM qualification_sessions WHERE uuid = ?", sessionID)

			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "reset_session_assignments", "Resetting assignments for session: "+sessionID, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "Penempatan berhasil direset",
			"reset_count":    rowsAffected,
			"retained_count": scoredCount,
		})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// Get assignments for both to verify they exist and get their current targets
		var targetA, targetB string
		err = tx.Get(&targetA, "SELECT target_uuid FROM qualification_target_assignments WHERE session_uuid = ? AND participant_uuid = ?", sessionID, req.ParticipantA)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan untuk peserta A tidak ditemukan"})
			return
		}

		err = tx.Get(&targetB, "SELECT target_uuid FROM qualification_target_assignments WHERE session_uuid = ? AND participant_uuid = ?", sessionID, req.ParticipantB)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Penempatan untuk peserta B tidak ditemukan"})
			return
		}

		// 1. Delete Participant A's assignment to free up Target A in the unique index
		_, err = tx.Exec("DELETE FROM qualification_target_assignments WHERE session_uuid = ? AND participant_uuid = ?", sessionID, req.ParticipantA)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal melepas peserta A", "details": err.Error()})
			return
		}

		// 2. Move Participant B to Target A
		var boardNumberA int
		tx.Get(&boardNumberA, "SELECT board_number FROM event_targets WHERE uuid = ?", targetA)

		var categoryIDB string
		tx.Get(&categoryIDB, "SELECT ep.category_id FROM event_participants ep WHERE ep.uuid = ?", req.ParticipantB)

		var targetBoardUUIDB sql.NullString
		tx.Get(&targetBoardUUIDB, "SELECT uuid FROM target_board_qualification WHERE session_uuid = ? AND category_uuid = ? AND board_number = ?",
			sessionID, categoryIDB, boardNumberA)

		_, err = tx.Exec("UPDATE qualification_target_assignments SET target_uuid = ?, target_board_id = ? WHERE session_uuid = ? AND participant_uuid = ?",
			targetA, targetBoardUUIDB, sessionID, req.ParticipantB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memindahkan peserta B", "details": err.Error()})
			return
		}

		// 3. Re-insert Participant A into Target B
		var boardNumberB int
		tx.Get(&boardNumberB, "SELECT board_number FROM event_targets WHERE uuid = ?", targetB)

		var categoryID string
		tx.Get(&categoryID, "SELECT ep.category_id FROM event_participants ep WHERE ep.uuid = ?", req.ParticipantA)

		var targetBoardUUIDA sql.NullString
		tx.Get(&targetBoardUUIDA, "SELECT uuid FROM target_board_qualification WHERE session_uuid = ? AND category_uuid = ? AND board_number = ?",
			sessionID, categoryID, boardNumberB)

		assignmentUUID := uuid.New().String()
		_, err = tx.Exec(`
			INSERT INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid, target_board_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, NOW(), NOW())`,
			assignmentUUID, sessionID, req.ParticipantA, targetB, targetBoardUUIDA)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memasang kembali peserta A", "details": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		// Log for scorekeeper audit
		userTypeContext, _ := c.Get("user_type")
		if userTypeContext == "scorekeeper" {
			userID, _ := c.Get("user_id")
			orgID, _ := c.Get("org_id")

			var eventUUID string
			_ = db.Get(&eventUUID, "SELECT event_uuid FROM qualification_sessions WHERE uuid = ?", sessionID)

			utils.LogScorekeeperAction(db, userID.(string), orgID.(string), eventUUID, "swap_assignments", "Swapped targets in session: "+sessionID, c.ClientIP(), c.Request.UserAgent())
		}

		c.JSON(http.StatusOK, gin.H{"message": "Target berhasil ditukar"})
	}
}

// GetBoardCodes returns all generated codes for target boards in a session for a specific category
func GetBoardCodes(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")
		categoryID := c.Query("category_id")
		if sessionID == "" || categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId dan category_id wajib diisi"})
			return
		}

		// Identify all unique board numbers (e.g. 1, 2, 13) currently in use for this category
		var boardNumbers []int
		err := db.Select(&boardNumbers, `
			SELECT DISTINCT et.board_number 
			FROM qualification_target_assignments qta 
			JOIN event_targets et ON qta.target_uuid = et.uuid 
			JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			WHERE qta.session_uuid = ? AND ep.category_id = ? AND et.board_number > 0
			ORDER BY et.board_number ASC
		`, sessionID, categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengidentifikasi papan target aktif", "details": err.Error()})
			return
		}

		// Get or generate the "event part" suffix (3 letters) for this session
		var eventUUID string
		db.Get(&eventUUID, "SELECT event_uuid FROM qualification_sessions WHERE uuid = ?", sessionID)

		var suffix string
		db.Get(&suffix, `
			SELECT RIGHT(code, 3) 
			FROM target_board_qualification tbq
			JOIN qualification_sessions qs ON tbq.session_uuid = qs.uuid
			WHERE qs.event_uuid = ? LIMIT 1
		`, eventUUID)

		if suffix == "" {
			const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ"
			b := make([]byte, 3)
			for j := range b {
				b[j] = charset[rand.Intn(len(charset))]
			}
			suffix = string(b)
		}

		// Ensure codes exist for all these board numbers for this category
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction failed", "details": err.Error()})
			return
		}
		defer tx.Rollback()

		for _, bn := range boardNumbers {
			var exists bool
			err := tx.Get(&exists, "SELECT EXISTS(SELECT 1 FROM `target_board_qualification` WHERE `session_uuid` = ? AND `category_uuid` = ? AND `board_number` = ?)", sessionID, categoryID, bn)
			if err != nil {
				continue
			}
			if !exists {
				code := fmt.Sprintf("%03d%s", bn, suffix)
				_, err = tx.Exec("INSERT INTO `target_board_qualification` (`uuid`, `session_uuid`, `category_uuid`, `board_number`, `code`) VALUES (?, ?, ?, ?, ?)",
					uuid.New().String(), sessionID, categoryID, bn, code)
				if err != nil {
					continue
				}
			}
		}
		tx.Commit()

		// Fetch all codes for this category in this session
		var codes []models.TargetBoardQualification
		err = db.Select(&codes, "SELECT `uuid`, `session_uuid`, `category_uuid`, `board_number`, `code`, `created_at` FROM `target_board_qualification` WHERE `session_uuid` = ? AND `category_uuid` = ?", sessionID, categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil kode papan", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"board_codes": codes})
	}
}

func generateUniqueBoardCodeWithTx(tx *sqlx.Tx, sessionUUID string) (string, error) {
	// Fallback/Legacy generator - now mostly handled inline in GetBoardCodes for specific format
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	for i := 0; i < 100; i++ {
		b := make([]byte, 6)
		for j := range b {
			b[j] = charset[rand.Intn(len(charset))]
		}
		code := string(b)
		var exists bool
		err := tx.Get(&exists, "SELECT EXISTS(SELECT 1 FROM `target_board_qualification` WHERE `session_uuid` = ? AND `code` = ?)", sessionUUID, code)
		if err != nil {
			return "", err
		}
		if !exists {
			return code, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique code")
}

// â”€â”€â”€ Scoresheet Data Structures â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type ScoresheetPosition struct {
	TargetName string
	ArcherName string
	ClubName   string
	Category   string
	Code       string
	ArrowRange []int
	EndRange   []int
	Empty      bool
}

type ScoresheetRow struct {
	Left  *ScoresheetPosition
	Right *ScoresheetPosition
}

type ScoresheetBoard struct {
	BoardNumber  int
	Code         string
	QRCodeBase64 string
	Rows         []ScoresheetRow
}

type ScoresheetData struct {
	EventName      string
	EventOrg       string
	Location       string
	EventDates     string
	SessionName    string
	SessionCode    string
	SessionDate    string
	SessionDayName string
	TotalEnds      int
	ArrowsPerEnd   int
	PrintDate      string
	Boards         []ScoresheetBoard
}

// makeRange returns []int{start, start+1, ..., end}
func makeRange(start, end int) []int {
	result := make([]int, end-start+1)
	for i := range result {
		result[i] = start + i
	}
	return result
}


