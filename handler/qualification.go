package handler

import (
	"archeryhub-api/models"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
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
				COUNT(DISTINCT qta.archer_uuid) as participant_count
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

// UpdateQualificationScore updates end scores for an assignment
func UpdateQualificationScore(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")
		var req models.ScoreUpdateRequest

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get assignment details
		var sessionUUID, archerUUID string
		err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}
		err = db.Get(&archerUUID, `SELECT archer_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}

		// Calculate end total and counts
		total := 0
		xCount := 0
		tenCount := 0

		for _, arrow := range req.Arrows {
			val, x, ten := calculateArrowValue(arrow)
			total += val
			xCount += x
			tenCount += ten
		}

		// Check if end score already exists
		var existingUUID sql.NullString
		err = db.Get(&existingUUID, `
			SELECT uuid FROM qualification_end_scores 
			WHERE session_uuid = ? AND archer_uuid = ? AND end_number = ?`,
			sessionUUID, archerUUID, req.EndNumber)

		if existingUUID.Valid {
			// Update existing end score
			_, err = db.Exec(`
				UPDATE qualification_end_scores 
				SET total_score_end = ?, x_count_end = ?, ten_count_end = ?
				WHERE uuid = ?`,
				total, xCount, tenCount, existingUUID.String)
		} else {
			// Create new end score
			scoreUUID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO qualification_end_scores 
				(uuid, session_uuid, archer_uuid, end_number, total_score_end, x_count_end, ten_count_end)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				scoreUUID, sessionUUID, archerUUID, req.EndNumber, total, xCount, tenCount)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update score", "details": err.Error()})
			return
		}

		// Now handle individual arrow scores
		// First, delete existing arrow scores for this end
		if existingUUID.Valid {
			_, err = db.Exec(`DELETE FROM qualification_arrow_scores WHERE end_score_uuid = ?`, existingUUID.String)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete old arrow scores"})
				return
			}
		}

		// Insert new arrow scores
		var endScoreUUID string
		if existingUUID.Valid {
			endScoreUUID = existingUUID.String
		} else {
			err = db.Get(&endScoreUUID, `
				SELECT uuid FROM qualification_end_scores 
				WHERE session_uuid = ? AND archer_uuid = ? AND end_number = ?`,
				sessionUUID, archerUUID, req.EndNumber)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve end score"})
				return
			}
		}

		for i, arrow := range req.Arrows {
			arrowUUID := uuid.New().String()
			val, _, _ := calculateArrowValue(arrow)
			isX := 0
			if arrow == "X" {
				isX = 1
			}

			_, err = db.Exec(`
				INSERT INTO qualification_arrow_scores 
				(uuid, end_score_uuid, arrow_number, score, is_x)
				VALUES (?, ?, ?, ?, ?)`,
				arrowUUID, endScoreUUID, i+1, val, isX)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save arrow score"})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Score updated successfully", "total": total})
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

		// Get session and archer from assignment
		var sessionUUID, archerUUID string
		err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}
		err = db.Get(&archerUUID, `SELECT archer_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
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
			WHERE session_uuid = ? AND archer_uuid = ?
			ORDER BY end_number ASC
		`, sessionUUID, archerUUID)
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

		type Entry struct {
			ArcherName    string  `json:"archer_name" db:"archer_name"`
			ClubName      *string `json:"club_name" db:"club_name"`
			CategoryName  string  `json:"category_name" db:"category_name"`
			TotalScore    int     `json:"total_score" db:"total_score"`
			TotalTenX     int     `json:"total_10x" db:"total_10x"`
			TotalX        int     `json:"total_x" db:"total_x"`
			EndsCompleted int     `json:"ends_completed" db:"ends_completed"`
		}

		var leaderboard []Entry
		err := db.Select(&leaderboard, `
			SELECT 
				a.full_name as archer_name,
				cl.name as club_name,
				CONCAT(bt.name, ' ', ag.name) as category_name,
				COALESCE(SUM(s.total_score_end), 0) as total_score,
				COALESCE(SUM(s.ten_count_end), 0) as total_10x,
				COALESCE(SUM(s.x_count_end), 0) as total_x,
				COUNT(s.uuid) as ends_completed
			FROM event_participants ep
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			LEFT JOIN qualification_target_assignments qta ON qta.archer_uuid = a.uuid
			LEFT JOIN qualification_sessions qs ON qs.uuid = qta.session_uuid
			LEFT JOIN qualification_end_scores s ON s.session_uuid = qs.uuid AND s.archer_uuid = a.uuid
			WHERE ep.category_id = ?
			GROUP BY ep.uuid, a.full_name, cl.name, bt.name, ag.name
			ORDER BY total_score DESC, total_10x DESC, total_x DESC`,
			categoryID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
	}
}

// GetSessionAssignments returns all archer assignments for a qualification session
func GetSessionAssignments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")

		type Assignment struct {
			UUID           string  `json:"uuid" db:"uuid"`
			ArcherUUID     string  `json:"archer_uuid" db:"archer_uuid"`
			TargetUUID     string  `json:"target_uuid" db:"target_uuid"`
			TargetNumber   string  `json:"target_number" db:"target_number"`
			TargetName     string  `json:"target_name" db:"target_name"`
			TargetPosition string  `json:"target_position" db:"target_position"`
			ArcherName     string  `json:"archer_name" db:"archer_name"`
			ClubName       *string `json:"club_name" db:"club_name"`
		}

		var assignments []Assignment
		err := db.Select(&assignments, `
			SELECT 
				qta.uuid,
				qta.archer_uuid,
				qta.target_uuid,
				et.target_number,
				et.target_name,
				qta.target_position,
				a.full_name as archer_name,
				c.name as club_name
			FROM qualification_target_assignments qta
			LEFT JOIN event_targets et ON qta.target_uuid = et.uuid
			LEFT JOIN archers a ON qta.archer_uuid = a.uuid
			LEFT JOIN clubs c ON a.club_id = c.uuid
			WHERE qta.session_uuid = ?
			ORDER BY CAST(et.target_number AS UNSIGNED) ASC, et.target_name ASC, qta.target_position ASC
		`, sessionID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignments", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"assignments": assignments})
	}
}

// AutoAssignParticipants automatically assigns participants to targets
func AutoAssignParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := c.Param("sessionId")

		var req struct {
			CategoryID       string `json:"category_id" binding:"required"`
			StartTargetUUID  string `json:"start_target_uuid" binding:"required"`
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

		// Get available targets for this event starting from the specified target
		type Target struct {
			UUID         string `db:"uuid"`
			TargetNumber string `db:"target_number"`
		}
		var targets []Target
		err = db.Select(&targets, `
			SELECT uuid, target_number
			FROM event_targets
			WHERE event_uuid = ?
			ORDER BY CAST(target_number AS UNSIGNED) ASC, target_name ASC
		`, eventUUID)

		if err != nil || len(targets) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "No targets available"})
			return
		}

		// Get participants who are not yet assigned
		type Participant struct {
			ArcherUUID string `db:"archer_id"`
		}
		var participants []Participant
		err = db.Select(&participants, `
			SELECT ep.archer_id
			FROM event_participants ep
			WHERE ep.category_id = ?
			AND ep.status = 'Terdaftar'
			AND ep.archer_id NOT IN (
				SELECT archer_uuid 
				FROM qualification_target_assignments 
				WHERE session_uuid = ?
			)
			ORDER BY ep.created_at ASC
		`, req.CategoryID, sessionID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants", "details": err.Error()})
			return
		}

		// Assign participants to targets
		positions := []string{"A", "B", "C", "D"}
		targetIndex := 0
		positionIndex := 0

		for _, participant := range participants {
			if targetIndex >= len(targets) {
				break // No more targets available
			}

			assignmentUUID := uuid.New().String()
			_, err := db.Exec(`
				INSERT INTO qualification_target_assignments 
				(uuid, session_uuid, archer_uuid, target_uuid, target_position)
				VALUES (?, ?, ?, ?, ?)`,
				assignmentUUID, sessionID, participant.ArcherUUID, targets[targetIndex].UUID, positions[positionIndex])

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment", "details": err.Error()})
				return
			}

			positionIndex++
			if positionIndex >= req.ArchersPerTarget || positionIndex >= len(positions) {
				positionIndex = 0
				targetIndex++
			}
		}

		c.JSON(http.StatusOK, gin.H{"message": "Participants assigned successfully", "count": len(participants)})
	}
}

// DeleteQualificationAssignment deletes an archer assignment
func DeleteQualificationAssignment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")

		// Get session and archer UUIDs
		var sessionUUID, archerUUID string
		err := db.Get(&sessionUUID, `SELECT session_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}
		err = db.Get(&archerUUID, `SELECT archer_uuid FROM qualification_target_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Assignment not found"})
			return
		}

		// First delete all related arrow scores
		_, err = db.Exec(`
			DELETE FROM qualification_arrow_scores 
			WHERE end_score_uuid IN (
				SELECT uuid FROM qualification_end_scores 
				WHERE session_uuid = ? AND archer_uuid = ?
			)`, sessionUUID, archerUUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete arrow scores"})
			return
		}

		// Delete end scores
		_, err = db.Exec(`DELETE FROM qualification_end_scores WHERE session_uuid = ? AND archer_uuid = ?`, sessionUUID, archerUUID)
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
				ArcherUUID string `json:"archer_uuid" binding:"required"`
				TargetID   string `json:"target_id" binding:"required"`
			} `json:"assignments" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

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

			// Validate archer exists in participants for this category
			var archerParticipantID string
			err := tx.Get(&archerParticipantID, `
				SELECT uuid FROM event_participants 
				WHERE archer_id = ? AND category_id = ?
			`, assignment.ArcherUUID, req.CategoryID)
			if err != nil {
				errors = append(errors, map[string]interface{}{
					"archer_uuid": assignment.ArcherUUID,
					"error":       "Archer not found in this category",
				})
				continue
			}

			// Insert assignment with default position 'A'
			_, err = tx.Exec(`
				INSERT INTO qualification_target_assignments 
				(uuid, session_uuid, archer_uuid, target_uuid, target_position, created_at, updated_at)
				VALUES (?, ?, ?, ?, 'A', NOW(), NOW())
			`, assignmentUUID, sessionUUID, assignment.ArcherUUID, assignment.TargetID)
			if err != nil {
				errors = append(errors, map[string]interface{}{
					"archer_uuid": assignment.ArcherUUID,
					"target_id":   assignment.TargetID,
					"error":       "Failed to create assignment: " + err.Error(),
				})
				continue
			}

			successCount++
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
