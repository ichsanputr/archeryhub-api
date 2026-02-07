package handler

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetTargets returns all targets for a given context (qualification session)
func GetTargets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		phase := c.Query("phase") // "qualification" only
		sessionID := c.Query("session_id")

		if phase == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phase is required"})
			return
		}

		if phase == "qualification" {
			if sessionID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required for qualification phase"})
				return
			}

			// Get all unique target names with their assignments and full archer details
			type ArcherInfo struct {
				ID            string `json:"id" db:"assignment_uuid"`
				ParticipantID string `json:"participant_id" db:"participant_uuid"`
				Name          string `json:"name" db:"archer_name"`
				Division      string `json:"division" db:"division_name"`
				Position      string `json:"position" db:"target_position"`
			}

			type TargetInfo struct {
				TargetName string       `json:"target_name" db:"target_name"`
				Archers    []ArcherInfo `json:"archers"`
			}

			// First, get all assignments with archer details
			type AssignmentRow struct {
				TargetName      string `db:"target_name"`
				AssignmentUUID  string `db:"assignment_uuid"`
				ParticipantUUID string `db:"participant_uuid"`
				ArcherName      string `db:"archer_name"`
				DivisionName    string `db:"division_name"`
				TargetPosition  string `db:"target_position"`
			}

			var assignments []AssignmentRow
			err := db.Select(&assignments, `
				SELECT 
				et.target_name,
				qta.uuid as assignment_uuid,
				qta.archer_uuid as participant_uuid,
				COALESCE(a.full_name, '') as archer_name,
				COALESCE(CONCAT(bt.name, ' ', ag.name), '') as division_name,
				qta.target_position
			FROM qualification_target_assignments qta
			JOIN event_targets et ON qta.target_uuid = et.uuid
			JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN archers a ON qta.archer_uuid = a.uuid
			LEFT JOIN event_participants ep ON ep.archer_id = a.uuid AND ep.event_id = qs.event_uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			WHERE qta.session_uuid = ?
			ORDER BY et.target_name ASC, qta.target_position ASC`,
				sessionID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch targets"})
				return
			}

			targetMap := make(map[string][]ArcherInfo)
			for _, a := range assignments {
				archer := ArcherInfo{
					ID:            a.AssignmentUUID,
					ParticipantID: a.ParticipantUUID,
					Name:          a.ArcherName,
					Division:      a.DivisionName,
					Position:      a.TargetPosition,
				}
				targetMap[a.TargetName] = append(targetMap[a.TargetName], archer)
			}

			type TargetRow struct {
				TargetName string `db:"target_name"`
			}
			var eventTargets []TargetRow
			var eventUUID string
			db.Get(&eventUUID, `SELECT event_uuid FROM qualification_sessions WHERE uuid = ?`, sessionID)

			if eventUUID != "" {
				db.Select(&eventTargets, `
					SELECT target_name
					FROM event_targets
					WHERE event_uuid = ? AND status = 'active'
					ORDER BY target_name ASC
				`, eventUUID)
			}

			var targets []TargetInfo
			for _, et := range eventTargets {
				archers := targetMap[et.TargetName]
				if archers == nil {
					archers = []ArcherInfo{}
				}

				targets = append(targets, TargetInfo{
					TargetName: et.TargetName,
					Archers:    archers,
				})
			}

			c.JSON(http.StatusOK, gin.H{"targets": targets})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phase"})
	}
}

// GetTargetNames returns all target names (contexts) for an event
func GetTargetNames(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Verify event exists
		var eventExists bool
		err := db.Get(&eventExists, `SELECT EXISTS(SELECT 1 FROM events WHERE uuid = ? OR slug = ?)`, eventID, eventID)
		if err != nil || !eventExists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		type TargetName struct {
			ID           string `json:"id" db:"id"`
			Name         string `json:"name" db:"name"`
			Phase        string `json:"phase" db:"phase"`
			CategoryID   string `json:"category_id" db:"category_id"`
			SessionID    string `json:"session_id,omitempty" db:"session_id"`
			SessionOrder int    `json:"session_order,omitempty" db:"session_order"`
			RoundName    string `json:"round_name,omitempty" db:"round_name"`
		}

		var targetNames []TargetName

		// Get qualification target names (event-level sessions)
		qualificationNames := []TargetName{}
		err = db.Select(&qualificationNames, `
			SELECT 
				CONCAT('qualification-', qs.event_uuid, '-sesi-', qs.uuid) as id,
				CONCAT('Kualifikasi (', qs.name, ')') as name,
				'qualification' as phase,
				'' as category_id,
				qs.uuid as session_id,
				0 as session_order
			FROM qualification_sessions qs
			WHERE qs.event_uuid = ? OR qs.event_uuid = (SELECT uuid FROM events WHERE slug = ?)
			ORDER BY qs.created_at ASC
		`, eventID, eventID)

		if err == nil {
			targetNames = append(targetNames, qualificationNames...)
		}

		c.JSON(http.StatusOK, gin.H{
			"target_names": targetNames,
			"count":        len(targetNames),
		})
	}
}

// UpdateQualificationAssignment updates or creates a qualification assignment
func UpdateQualificationAssignment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SessionUUID     string  `json:"session_id" binding:"required"`
			ParticipantUUID string  `json:"participant_id" binding:"required"`
			TargetName      string  `json:"target_name" binding:"required"`
			TargetPosition  string  `json:"target_position" binding:"required"` // A, B, C, D
			AssignmentUUID  *string `json:"assignment_id,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate target position
		if req.TargetPosition != "A" && req.TargetPosition != "B" && req.TargetPosition != "C" && req.TargetPosition != "D" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target_position must be A, B, C, or D"})
			return
		}

		var targetUUID string
		err := db.Get(&targetUUID, `
			SELECT et.uuid FROM event_targets et
			JOIN qualification_sessions qs ON qs.event_uuid = et.event_uuid
			WHERE qs.uuid = ? AND et.target_name = ?
		`, req.SessionUUID, req.TargetName)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid target name for this session"})
			return
		}

		// Check if position is already taken by another archer
		var existingAssignment string
		err = db.Get(&existingAssignment, `
			SELECT uuid FROM qualification_target_assignments 
			WHERE session_uuid = ? AND target_uuid = ? AND target_position = ? 
			AND uuid != COALESCE(?, '')
		`, req.SessionUUID, targetUUID, req.TargetPosition, req.AssignmentUUID)

		if err == nil && existingAssignment != "" {
			c.JSON(http.StatusConflict, gin.H{"error": "Target position already assigned to another archer"})
			return
		}

		// Check if archer already has an assignment in this session
		var archerUUID string
		err = db.Get(&archerUUID, `SELECT archer_id FROM event_participants WHERE uuid = ?`, req.ParticipantUUID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Participant not found"})
			return
		}

		var existingParticipantAssignment string
		err = db.Get(&existingParticipantAssignment, `
			SELECT uuid FROM qualification_target_assignments 
			WHERE session_uuid = ? AND archer_uuid = ? AND uuid != COALESCE(?, '')
		`, req.SessionUUID, archerUUID, req.AssignmentUUID)

		if err == nil && existingParticipantAssignment != "" {
			// Update existing assignment for this archer
			_, err = db.Exec(`
					UPDATE qualification_target_assignments 
					SET target_uuid = ?, target_position = ?, updated_at = NOW()
					WHERE uuid = ? AND session_uuid = ?
				`, targetUUID, req.TargetPosition, existingParticipantAssignment, req.SessionUUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message":       "Assignment updated successfully",
				"assignment_id": existingParticipantAssignment,
			})
			return
		}

		if req.AssignmentUUID != nil && *req.AssignmentUUID != "" {
			// Update existing assignment
			_, err = db.Exec(`
				UPDATE qualification_target_assignments 
				SET target_uuid = ?, target_position = ?, updated_at = NOW()
				WHERE uuid = ? AND session_uuid = ?
			`, targetUUID, req.TargetPosition, *req.AssignmentUUID, req.SessionUUID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message":       "Assignment updated successfully",
				"assignment_id": *req.AssignmentUUID,
			})
		} else {
			// Create new assignment
			newUUID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO qualification_target_assignments (uuid, session_uuid, archer_uuid, target_uuid, target_position)
				VALUES (?, ?, ?, ?, ?)
			`, newUUID, req.SessionUUID, archerUUID, targetUUID, req.TargetPosition)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment"})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"message":       "Assignment created successfully",
				"assignment_id": newUUID,
			})
		}
	}
}

// GetEventTargets returns all targets for an event - Data Master with pagination
func GetEventTargets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		page := c.DefaultQuery("page", "1")
		limit := c.DefaultQuery("limit", "10")
		orderBy := c.DefaultQuery("order_by", "created_at")
		orderDir := c.DefaultQuery("order_dir", "DESC")

		// Validate order_by to prevent SQL injection
		allowedSortFields := map[string]bool{
			"created_at":  true,
			"target_name": true,
			"updated_at":  true,
		}
		if !allowedSortFields[orderBy] {
			orderBy = "created_at"
		}

		// Validate order_dir
		orderDir = strings.ToUpper(orderDir)
		if orderDir != "ASC" && orderDir != "DESC" {
			orderDir = "DESC"
		}

		// Verify event exists
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		type Target struct {
			Number    string    `json:"target_number" db:"target_number"`
			Letters   string    `json:"letters" db:"letters"`
			TargetIDs string    `json:"target_ids" db:"target_ids"` // comma separated UUIDs
			Status    string    `json:"status" db:"status"`
			CreatedAt time.Time `json:"created_at" db:"created_at"`
		}

		// Calculate pagination based on unique target numbers
		var total int
		err = db.Get(&total, `SELECT COUNT(DISTINCT REGEXP_REPLACE(target_name, '[^0-9]', '')) FROM event_targets WHERE event_uuid = ?`, eventUUID)
		if err != nil {
			// Fallback if REGEXP_REPLACE is not available (older MariaDB)
			db.Get(&total, `SELECT COUNT(DISTINCT LEFT(target_name, 1)) FROM event_targets WHERE event_uuid = ?`, eventUUID)
		}

		offset := 0
		limitInt := 10
		if l, err := strconv.Atoi(limit); err == nil && l > 0 {
			limitInt = l
		}
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			offset = (p - 1) * limitInt
		}

		var targets []Target
		// Sort by the numeric part
		err = db.Select(&targets, fmt.Sprintf(`
			SELECT 
				REGEXP_REPLACE(target_name, '[^0-9]', '') as target_number,
				GROUP_CONCAT(REGEXP_REPLACE(target_name, '[0-9]', '') ORDER BY target_name ASC SEPARATOR ', ') as letters,
				GROUP_CONCAT(uuid SEPARATOR ',') as target_ids,
				MAX(status) as status,
				MIN(created_at) as created_at
			FROM event_targets
			WHERE event_uuid = ?
			GROUP BY target_number
			ORDER BY CAST(target_number AS UNSIGNED) %s
			LIMIT %d OFFSET %d
		`, orderDir, limitInt, offset), eventUUID)

		if err != nil {
			// Fallback for systems without REGEXP_REPLACE
			err = db.Select(&targets, fmt.Sprintf(`
				SELECT 
					LEFT(target_name, 1) as target_number,
					GROUP_CONCAT(SUBSTRING(target_name, 2) ORDER BY target_name ASC SEPARATOR ', ') as letters,
					GROUP_CONCAT(uuid SEPARATOR ',') as target_ids,
					MAX(status) as status,
					MIN(created_at) as created_at
				FROM event_targets
				WHERE event_uuid = ?
				GROUP BY target_number
				ORDER BY CAST(target_number AS UNSIGNED) %s
				LIMIT %d OFFSET %d
			`, orderDir, limitInt, offset), eventUUID)
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch targets", "details": err.Error()})
			return
		}

		if targets == nil {
			targets = []Target{}
		}

		c.JSON(http.StatusOK, gin.H{
			"targets": targets,
			"total":   total,
			"page":    page,
			"limit":   limit,
		})
	}
}

// CreateEventTarget creates a new target for an event
func CreateEventTarget(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			TargetNumbers []string `json:"target_numbers"`
			TargetName    string   `json:"target_name" binding:"required"`
			Description   string   `json:"description"`
			VenueArea     string   `json:"venue_area"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify event exists
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Normalize target numbers
		numbers := req.TargetNumbers
		clean := []string{}
		seen := map[string]bool{}
		for _, n := range numbers {
			val := strings.TrimSpace(n)
			if val == "" || seen[val] {
				continue
			}
			seen[val] = true
			clean = append(clean, val)
		}
		if len(clean) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target_numbers is required"})
			return
		}

		// Check duplicates in DB
		dup := []string{}
		for _, letter := range clean {
			fullName := fmt.Sprintf("%s%s", req.TargetName, letter)
			var existingTarget string
			err = db.Get(&existingTarget, `
				SELECT uuid FROM event_targets 
				WHERE event_uuid = ? AND target_name = ?
			`, eventUUID, fullName)
			if err == nil && existingTarget != "" {
				dup = append(dup, fullName)
			}
		}
		if len(dup) > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Target already exists", "duplicates": dup})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		createdIDs := []string{}
		for _, letter := range clean {
			newUUID := uuid.New().String()
			fullName := fmt.Sprintf("%s%s", req.TargetName, letter)
			_, err = tx.Exec(`
				INSERT INTO event_targets (
					uuid, event_uuid, target_name, 
					description, venue_area, status, 
					created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, 'active', NOW(), NOW())
			`, newUUID, eventUUID, fullName,
				req.Description, req.VenueArea)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create target", "details": err.Error()})
				return
			}
			createdIDs = append(createdIDs, newUUID)
		}
		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":        "Targets created successfully",
			"created_count":  len(createdIDs),
			"target_numbers": clean,
		})
	}
}

// UpdateEventTarget updates an existing target
func UpdateEventTarget(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetID := c.Param("target_id")

		var req struct {
			TargetName  *string `json:"target_name"`
			Description *string `json:"description"`
			VenueArea   *string `json:"venue_area"`
			Status      *string `json:"status"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify target exists
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT event_uuid FROM event_targets WHERE uuid = ?`, targetID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Target not found"})
			return
		}

		// Check if new target name conflicts
		if req.TargetName != nil {
			var existingTarget string
			err = db.Get(&existingTarget, `
				SELECT uuid FROM event_targets 
				WHERE event_uuid = ? AND target_name = ? AND uuid != ?
			`, eventUUID, *req.TargetName, targetID)

			if err == nil && existingTarget != "" {
				c.JSON(http.StatusConflict, gin.H{"error": "Target name already exists"})
				return
			}
		}

		// Build update query dynamically
		updateFields := []string{}
		updateValues := []interface{}{}

		if req.TargetName != nil {
			updateFields = append(updateFields, "target_name = ?")
			updateValues = append(updateValues, *req.TargetName)
		}
		if req.Description != nil {
			updateFields = append(updateFields, "description = ?")
			updateValues = append(updateValues, *req.Description)
		}
		if req.VenueArea != nil {
			updateFields = append(updateFields, "venue_area = ?")
			updateValues = append(updateValues, *req.VenueArea)
		}
		if req.Status != nil {
			updateFields = append(updateFields, "status = ?")
			updateValues = append(updateValues, *req.Status)
		}

		if len(updateFields) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
			return
		}

		updateFields = append(updateFields, "updated_at = NOW()")
		updateValues = append(updateValues, targetID)

		query := fmt.Sprintf("UPDATE event_targets SET %s WHERE uuid = ?",
			joinStrings(updateFields, ", "))

		_, err = db.Exec(query, updateValues...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update target", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Target updated successfully"})
	}
}

// DeleteEventTarget deletes a target
func DeleteEventTarget(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetID := c.Param("target_id")

		// Check if target has any assignments
		var assignmentCount int
		err := db.Get(&assignmentCount, `
			SELECT COUNT(*) FROM qualification_target_assignments
			WHERE target_uuid = ?
		`, targetID)

		if err == nil && assignmentCount > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error":          "Cannot delete target with existing archer assignments",
				"assigned_count": assignmentCount,
			})
			return
		}

		_, err = db.Exec(`DELETE FROM event_targets WHERE uuid = ?`, targetID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete target"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Target deleted successfully"})
	}
}

// GetTargetDetails returns detailed information about a specific target
func GetTargetDetails(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetID := c.Param("target_id")

		type TargetDetail struct {
			UUID        string    `json:"id" db:"uuid"`
			TargetName  string    `json:"target_name" db:"target_name"`
			Description string    `json:"description" db:"description"`
			VenueArea   string    `json:"venue_area" db:"venue_area"`
			Status      string    `json:"status" db:"status"`
			CreatedAt   time.Time `json:"created_at" db:"created_at"`
			UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
		}

		var target TargetDetail
		err := db.Get(&target, `
			SELECT 
				uuid,
				CONCAT(target_number, target_name) as target_name,
				COALESCE(description, '') as description,
				COALESCE(venue_area, '') as venue_area,
				status,
				created_at,
				updated_at
			FROM event_targets
			WHERE uuid = ?
		`, targetID)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Target not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch target details"})
			}
			return
		}

		// Get assigned archers
		type AssignedArcher struct {
			Name     string `json:"name" db:"name"`
			Position string `json:"position" db:"position"`
			Session  string `json:"session" db:"session"`
		}

		var archers []AssignedArcher
		db.Select(&archers, `
			SELECT 
				COALESCE(a.full_name, '') as name,
				qta.target_position as position,
				qs.name as session
			FROM qualification_target_assignments qta
			JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN archers a ON qta.archer_uuid = a.uuid
			WHERE qta.target_uuid = ?
			ORDER BY qs.name, qta.target_position
		`, targetID)

		if archers == nil {
			archers = []AssignedArcher{}
		}

		c.JSON(http.StatusOK, gin.H{
			"target":  target,
			"archers": archers,
		})
	}
}

// GetTargetOptions returns target options for an event session
func GetTargetOptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		type TargetOption struct {
			ID    string `json:"id" db:"uuid"`
			Value string `json:"value" db:"target_name"`
			Name  string `json:"name" db:"target_name"`
		}

		var options []TargetOption
		err := db.Select(&options, `
			SELECT 
				uuid,
				CONCAT(target_number, target_name) as target_name
			FROM event_targets
			WHERE (event_uuid = ? OR event_uuid = (SELECT uuid FROM events WHERE slug = ?))
			AND status = 'active'
			ORDER BY target_name ASC
		`, eventID, eventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch target options"})
			return
		}

		if options == nil {
			options = []TargetOption{}
		}

		c.JSON(http.StatusOK, gin.H{
			"options": options,
			"count":   len(options),
		})
	}
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
