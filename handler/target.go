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
			c.JSON(http.StatusBadRequest, gin.H{"error": "phase wajib diisi"})
			return
		}

		if phase == "qualification" {
			if sessionID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "session_id wajib diisi untuk fase kualifikasi"})
				return
			}

			// Get all unique target names with their assignments and full archer details
			type ArcherInfo struct {
				ID            string `json:"id" db:"assignment_uuid"`
				ParticipantID string `json:"participant_id" db:"participant_uuid"`
				Name          string `json:"name" db:"archer_name"`
				Division      string `json:"division" db:"division_name"`
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
			}

			var assignments []AssignmentRow
			err := db.Select(&assignments, `
				SELECT 
				et.target_name,
				qta.uuid as assignment_uuid,
				qta.participant_uuid,
				COALESCE(a.full_name, '') as archer_name,
				COALESCE(ec.category_name_custom, CONCAT(bt.name, ' ', ag.name), '') as division_name
			FROM qualification_target_assignments qta
			JOIN event_targets et ON qta.target_uuid = et.uuid
			JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			WHERE qta.session_uuid = ?
			ORDER BY et.board_number ASC, et.target_name ASC`,
				sessionID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data target"})
				return
			}

			targetMap := make(map[string][]ArcherInfo)
			for _, a := range assignments {
				archer := ArcherInfo{
					ID:            a.AssignmentUUID,
					ParticipantID: a.ParticipantUUID,
					Name:          a.ArcherName,
					Division:      a.DivisionName,
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
					ORDER BY board_number ASC, target_name ASC
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

		c.JSON(http.StatusBadRequest, gin.H{"error": "Fase tidak valid"})
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
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
			TargetUUID      string  `json:"target_id" binding:"required"`
			AssignmentUUID  *string `json:"assignment_id,omitempty"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if target is already taken by another archer in this session
		var existingAssignment string
		err := db.Get(&existingAssignment, `
			SELECT uuid FROM qualification_target_assignments 
			WHERE session_uuid = ? AND target_uuid = ? 
			AND uuid != COALESCE(?, '')
		`, req.SessionUUID, req.TargetUUID, req.AssignmentUUID)

		if err == nil && existingAssignment != "" {
			c.JSON(http.StatusConflict, gin.H{"error": "Posisi target sudah diberikan ke pemanah lain"})
			return
		}

		// Check if participant already has an assignment in this session
		var existingParticipantAssignment string
		err = db.Get(&existingParticipantAssignment, `
			SELECT uuid FROM qualification_target_assignments 
			WHERE session_uuid = ? AND participant_uuid = ? AND uuid != COALESCE(?, '')
		`, req.SessionUUID, req.ParticipantUUID, req.AssignmentUUID)

		if err == nil && existingParticipantAssignment != "" {
			// Update existing assignment for this participant
			var boardNumber int
			db.Get(&boardNumber, "SELECT board_number FROM event_targets WHERE uuid = ?", req.TargetUUID)

			var categoryID string
			db.Get(&categoryID, "SELECT category_id FROM event_participants WHERE uuid = ?", req.ParticipantUUID)

			var targetBoardUUID sql.NullString
			db.Get(&targetBoardUUID, "SELECT uuid FROM target_board_qualification WHERE session_uuid = ? AND category_uuid = ? AND board_number = ?", 
				req.SessionUUID, categoryID, boardNumber)

			_, err = db.Exec(`
					UPDATE qualification_target_assignments 
					SET target_uuid = ?, target_board_id = ?, updated_at = NOW()
					WHERE uuid = ? AND session_uuid = ?
				`, req.TargetUUID, targetBoardUUID, existingParticipantAssignment, req.SessionUUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui penempatan"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message":       "Penempatan berhasil diperbarui",
				"assignment_id": existingParticipantAssignment,
			})
			return
		}

		if req.AssignmentUUID != nil && *req.AssignmentUUID != "" {
			// Update existing assignment
			var boardNumber int
			db.Get(&boardNumber, "SELECT board_number FROM event_targets WHERE uuid = ?", req.TargetUUID)

			var categoryID string
			db.Get(&categoryID, "SELECT category_id FROM event_participants WHERE uuid = ?", req.ParticipantUUID)

			var targetBoardUUID sql.NullString
			db.Get(&targetBoardUUID, "SELECT uuid FROM target_board_qualification WHERE session_uuid = ? AND category_uuid = ? AND board_number = ?", 
				req.SessionUUID, categoryID, boardNumber)

			_, err = db.Exec(`
				UPDATE qualification_target_assignments 
				SET target_uuid = ?, target_board_id = ?, updated_at = NOW()
				WHERE uuid = ? AND session_uuid = ?
			`, req.TargetUUID, targetBoardUUID, *req.AssignmentUUID, req.SessionUUID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui penempatan"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message":       "Penempatan berhasil diperbarui",
				"assignment_id": *req.AssignmentUUID,
			})
		} else {
			// Create new assignment
			var boardNumber int
			db.Get(&boardNumber, "SELECT board_number FROM event_targets WHERE uuid = ?", req.TargetUUID)

			var categoryID string
			db.Get(&categoryID, "SELECT category_id FROM event_participants WHERE uuid = ?", req.ParticipantUUID)

			var targetBoardUUID sql.NullString
			db.Get(&targetBoardUUID, "SELECT uuid FROM target_board_qualification WHERE session_uuid = ? AND category_uuid = ? AND board_number = ?", 
				req.SessionUUID, categoryID, boardNumber)

			newUUID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO qualification_target_assignments (uuid, session_uuid, participant_uuid, target_uuid, target_board_id)
				VALUES (?, ?, ?, ?, ?)
			`, newUUID, req.SessionUUID, req.ParticipantUUID, req.TargetUUID, targetBoardUUID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat penempatan"})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"message":       "Penempatan berhasil dibuat",
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
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		type Target struct {
			Number    string    `json:"target_number" db:"target_number"`
			Letters   string    `json:"letters" db:"letters"`
			TargetIDs string    `json:"target_ids" db:"target_ids"` // comma separated UUIDs
			CreatedAt time.Time `json:"created_at" db:"created_at"`
		}

		// Calculate pagination based on unique target numbers
		var total int
		err = db.Get(&total, `SELECT COUNT(DISTINCT board_number) FROM event_targets WHERE event_uuid = ?`, eventUUID)

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
				board_number as target_number,
				GROUP_CONCAT(REGEXP_REPLACE(target_name, '[0-9]', '') ORDER BY target_name ASC SEPARATOR ', ') as letters,
				GROUP_CONCAT(uuid ORDER BY target_name ASC SEPARATOR ',') as target_ids,
				MIN(created_at) as created_at
			FROM event_targets
			WHERE event_uuid = ?
			GROUP BY board_number
			ORDER BY board_number %s
			LIMIT %d OFFSET %d
		`, orderDir, limitInt, offset), eventUUID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data target", "details": err.Error()})
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
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify event exists
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		// Normalize target numbers and validate A-D
		numbers := req.TargetNumbers
		clean := []string{}
		seen := map[string]bool{}
		allowedLetters := map[string]bool{"A": true, "B": true, "C": true, "D": true}
		
		for _, n := range numbers {
			val := strings.TrimSpace(n)
			val = strings.ToUpper(val)
			if val == "" || seen[val] {
				continue
			}
			if !allowedLetters[val] {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Huruf target tidak valid: %s. Hanya A, B, C, D yang diizinkan.", val)})
				return
			}
			seen[val] = true
			clean = append(clean, val)
		}
		if len(clean) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target_numbers wajib diisi"})
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
			c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Nama target sudah ada: %s", strings.Join(dup, ", ")), "duplicates": dup})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		createdIDs := []string{}
		for _, letter := range clean {
			newUUID := uuid.New().String()
			fullName := fmt.Sprintf("%s%s", req.TargetName, letter)
			
			// Extract board number from target name
			boardNumStr := ""
			for _, char := range req.TargetName {
				if char >= '0' && char <= '9' {
					boardNumStr += string(char)
				}
			}
			boardNum, _ := strconv.Atoi(boardNumStr)

			_, err = tx.Exec(`
				INSERT INTO event_targets (
					uuid, event_uuid, target_name, board_number,
					created_at, updated_at
				) VALUES (?, ?, ?, ?, NOW(), NOW())
			`, newUUID, eventUUID, fullName, boardNum)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat target", "details": err.Error()})
				return
			}
			createdIDs = append(createdIDs, newUUID)
		}
		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":        "Target berhasil dibuat",
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
			TargetName *string `json:"target_name"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify target exists
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT event_uuid FROM event_targets WHERE uuid = ?`, targetID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Target tidak ditemukan"})
			return
		}

		// Check if new target name conflicts and validate A-D
		if req.TargetName != nil {
			name := *req.TargetName
			if len(name) < 2 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Format nama target tidak valid"})
				return
			}
			letter := strings.ToUpper(name[len(name)-1:])
			allowedLetters := map[string]bool{"A": true, "B": true, "C": true, "D": true}
			if !allowedLetters[letter] {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Huruf target tidak valid. Hanya A, B, C, D yang diizinkan."})
				return
			}

			var existingTarget string
			err = db.Get(&existingTarget, `
				SELECT uuid FROM event_targets 
				WHERE event_uuid = ? AND target_name = ? AND uuid != ?
			`, eventUUID, *req.TargetName, targetID)

			if err == nil && existingTarget != "" {
				c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Nama target '%s' sudah ada di event ini", *req.TargetName)})
				return
			}
		}

		// Build update query dynamically
		updateFields := []string{}
		updateValues := []interface{}{}

		if req.TargetName != nil {
			updateFields = append(updateFields, "target_name = ?")
			updateValues = append(updateValues, *req.TargetName)

			// Extract board number from target name
			boardNumStr := ""
			for _, char := range *req.TargetName {
				if char >= '0' && char <= '9' {
					boardNumStr += string(char)
				}
			}
			boardNum, _ := strconv.Atoi(boardNumStr)
			updateFields = append(updateFields, "board_number = ?")
			updateValues = append(updateValues, boardNum)
		}

		if len(updateFields) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada bidang untuk diperbarui"})
			return
		}

		updateFields = append(updateFields, "updated_at = NOW()")
		updateValues = append(updateValues, targetID)

		query := fmt.Sprintf("UPDATE event_targets SET %s WHERE uuid = ?",
			joinStrings(updateFields, ", "))

		_, err = db.Exec(query, updateValues...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui target", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Target berhasil diperbarui"})
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
				"error":          "Tidak dapat menghapus target yang sudah memiliki penempatan pemanah",
				"assigned_count": assignmentCount,
			})
			return
		}

		_, err = db.Exec(`DELETE FROM event_targets WHERE uuid = ?`, targetID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus target"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Target berhasil dihapus"})
	}
}

// GetTargetDetails returns detailed information about a specific target
func GetTargetDetails(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		targetID := c.Param("target_id")

		type TargetDetail struct {
			UUID       string    `json:"id" db:"uuid"`
			TargetName string    `json:"target_name" db:"target_name"`
			CreatedAt  time.Time `json:"created_at" db:"created_at"`
			UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
		}

		var target TargetDetail
		err := db.Get(&target, `
			SELECT 
				uuid,
				target_name,
				created_at,
				updated_at
			FROM event_targets
			WHERE uuid = ?
		`, targetID)

		if err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "Target tidak ditemukan"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail target"})
			}
			return
		}

		// Get assigned archers
		type AssignedArcher struct {
			Name     string `json:"name" db:"name"`
			Session  string `json:"session" db:"session"`
		}

		var archers []AssignedArcher
		db.Select(&archers, `
			SELECT 
				COALESCE(a.full_name, '') as name,
				qs.name as session
			FROM qualification_target_assignments qta
			JOIN qualification_sessions qs ON qta.session_uuid = qs.uuid
			LEFT JOIN archers a ON qta.archer_uuid = a.uuid
			WHERE qta.target_uuid = ?
			ORDER BY qs.name
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

// BatchUpdateTargets updates multiple targets at once within a transaction
func BatchUpdateTargets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			Updates []struct {
				UUID       string `json:"uuid" binding:"required"`
				TargetName string `json:"target_name" binding:"required"`
			} `json:"updates" binding:"required"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify event exists
		var eventUUID string
		err := db.Get(&eventUUID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event tidak ditemukan"})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// 1. Rename all targets in the request to a temporary name to avoid transient conflicts
		for _, update := range req.Updates {
			tempName := fmt.Sprintf("__tmp_%s", update.UUID)
			_, err = tx.Exec(`UPDATE event_targets SET target_name = ?, updated_at = NOW() WHERE uuid = ? AND event_uuid = ?`, tempName, update.UUID, eventUUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengatur nama sementara", "details": err.Error()})
				return
			}
		}

		// 2. Perform actual updates and check for conflicts
		for _, update := range req.Updates {
			// Check if new target name conflicts and validate A-D
			name := update.TargetName
			if len(name) < 2 {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Format nama target tidak valid: %s", name)})
				return
			}
			letter := strings.ToUpper(name[len(name)-1:])
			allowedLetters := map[string]bool{"A": true, "B": true, "C": true, "D": true}
			if !allowedLetters[letter] {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Huruf target tidak valid pada '%s'. Hanya A, B, C, D yang diizinkan.", name)})
				return
			}

			// Check if name is taken by a target NOT in our update list
			// (Since all in update list are now __tmp_..., we just check against any target)
			var existingTarget string
			err = tx.Get(&existingTarget, `
				SELECT uuid FROM event_targets 
				WHERE event_uuid = ? AND target_name = ? AND uuid != ?
			`, eventUUID, update.TargetName, update.UUID)

			if err == nil && existingTarget != "" {
				c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("Nama target '%s' sudah ada di event ini", update.TargetName)})
				return
			}

			// Extract board number from target name
			boardNumStr := ""
			for _, char := range update.TargetName {
				if char >= '0' && char <= '9' {
					boardNumStr += string(char)
				}
			}
			boardNum, _ := strconv.Atoi(boardNumStr)

			_, err = tx.Exec(`UPDATE event_targets SET target_name = ?, board_number = ?, updated_at = NOW() WHERE uuid = ? AND event_uuid = ?`, update.TargetName, boardNum, update.UUID, eventUUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui target", "details": err.Error()})
				return
			}
		}

		if err = tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Targets updated successfully"})
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
				target_name
			FROM event_targets
			WHERE (event_uuid = ? OR event_uuid = (SELECT uuid FROM events WHERE slug = ?))
			ORDER BY board_number ASC, target_name ASC
		`, eventID, eventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil opsi target"})
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
