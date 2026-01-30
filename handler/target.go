package handler

import (
	"fmt"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/google/uuid"
)

// CreateTarget creates a target for qualification or elimination phase
func CreateTarget(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Phase        string `json:"phase" binding:"required"` // "qualification" or "elimination"
			CategoryID   string `json:"category_id" binding:"required"`
			SessionID    string `json:"session_id"`               // Required for qualification
			RoundName    string `json:"round_name"`               // Required for elimination
			TargetNumber *int   `json:"target_number"`            // Optional - can be assigned later
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate phase-specific requirements
		if req.Phase == "qualification" && req.SessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required for qualification phase"})
			return
		}

		if req.Phase == "elimination" && req.RoundName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "round_name is required for elimination phase"})
			return
		}

		if req.Phase == "qualification" {
			// For qualification: Ensure session exists, target is ready for assignments
			var sessionExists bool
			err := db.Get(&sessionExists, `
				SELECT COUNT(*) > 0 
				FROM qualification_sessions 
				WHERE uuid = ? AND event_category_uuid = ?`,
				req.SessionID, req.CategoryID)
			
			if err != nil || !sessionExists {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session or category mismatch"})
				return
			}

			// Target context is ready (session exists)
			// Return success with target context info
			c.JSON(http.StatusCreated, gin.H{
				"message": "Target context created successfully",
				"target": gin.H{
					"phase":       req.Phase,
					"session_id":  req.SessionID,
					"category_id": req.CategoryID,
				},
			})
			return
		}

		// For elimination: Ensure match exists or create it
		if req.Phase == "elimination" {
			// Check if match exists for this round and category
			var matchUUID string
			err := db.Get(&matchUUID, `
				SELECT uuid 
				FROM matches 
				WHERE event_category_uuid = ? AND round_name = ? 
				LIMIT 1`,
				req.CategoryID, req.RoundName)

			if err != nil {
				// Match doesn't exist, create one
				matchUUID = uuid.New().String()
				_, err = db.Exec(`
					INSERT INTO matches (uuid, event_category_uuid, round_name, match_order, status)
					VALUES (?, ?, ?, 1, 'scheduled')`,
					matchUUID, req.CategoryID, req.RoundName)

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create match"})
					return
				}
			}

			// Target context is ready (match exists or created)
			c.JSON(http.StatusCreated, gin.H{
				"message": "Target context created successfully",
				"target": gin.H{
					"phase":       req.Phase,
					"match_id":    matchUUID,
					"round_name":  req.RoundName,
					"category_id": req.CategoryID,
				},
			})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phase"})
	}
}

// GetTargets returns all targets for a given context (qualification session or elimination round)
func GetTargets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		phase := c.Query("phase") // "qualification" or "elimination"
		sessionID := c.Query("session_id")
		roundName := c.Query("round_name")
		categoryID := c.Query("category_id")

		if phase == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phase is required"})
			return
		}

		if phase == "qualification" {
			if sessionID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required for qualification phase"})
				return
			}

			// Get all unique target numbers with their assignments and full archer details
			type ArcherInfo struct {
				ID           string `json:"id" db:"assignment_uuid"`
				ParticipantID string `json:"participant_id" db:"participant_uuid"`
				Name         string `json:"name" db:"archer_name"`
				Division     string `json:"division" db:"division_name"`
				Position     string `json:"position" db:"target_position"`
			}

			type TargetInfo struct {
				TargetNumber int         `json:"target_number" db:"target_number"`
				CardName     string      `json:"card_name,omitempty"`
				Archers      []ArcherInfo `json:"archers"`
			}

			// First, get all assignments with archer details
			type AssignmentRow struct {
				TargetNumber   int    `db:"target_number"`
				AssignmentUUID string `db:"assignment_uuid"`
				ParticipantUUID string `db:"participant_uuid"`
				ArcherName     string `db:"archer_name"`
				DivisionName   string `db:"division_name"`
				TargetPosition string `db:"target_position"`
			}

			var assignments []AssignmentRow
			err := db.Select(&assignments, `
				SELECT 
					qa.target_number,
					qa.uuid as assignment_uuid,
					qa.participant_uuid,
					COALESCE(a.full_name, ea.full_name, '') as archer_name,
					COALESCE(CONCAT(bt.name, ' ', ag.name), '') as division_name,
					qa.target_position
				FROM qualification_assignments qa
				LEFT JOIN event_participants ep ON qa.participant_uuid = ep.uuid
				LEFT JOIN archers a ON ep.archer_id = a.uuid
				LEFT JOIN event_archers ea ON ep.event_archer_id = ea.uuid
				LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
				LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
				LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
				WHERE qa.session_uuid = ?
				ORDER BY qa.target_number ASC, qa.target_position ASC`,
				sessionID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch targets"})
				return
			}

			// Group by target number
			targetMap := make(map[int][]ArcherInfo)
			for _, a := range assignments {
				archer := ArcherInfo{
					ID:           a.AssignmentUUID,
					ParticipantID: a.ParticipantUUID,
					Name:         a.ArcherName,
					Division:     a.DivisionName,
					Position:     a.TargetPosition,
				}
				targetMap[a.TargetNumber] = append(targetMap[a.TargetNumber], archer)
			}

			// Get target card names
			type TargetCardRow struct {
				TargetNumber int    `db:"target_number"`
				CardName     string `db:"card_name"`
			}
			var targetCards []TargetCardRow
			db.Select(&targetCards, `
				SELECT target_number, card_name
				FROM target_cards
				WHERE session_uuid = ? AND phase = 'qualification'
			`, sessionID)

			cardNameMap := make(map[int]string)
			for _, card := range targetCards {
				cardNameMap[card.TargetNumber] = card.CardName
			}

			// Convert to array - include targets with assignments
			var targets []TargetInfo
			for targetNum, archers := range targetMap {
				cardName := cardNameMap[targetNum]
				if cardName == "" {
					cardName = fmt.Sprintf("Target %d", targetNum)
				}
				targets = append(targets, TargetInfo{
					TargetNumber: targetNum,
					CardName:     cardName,
					Archers:      archers,
				})
			}

			// Also include target cards that don't have any assignments yet
			for _, card := range targetCards {
				if _, exists := targetMap[card.TargetNumber]; !exists {
					targets = append(targets, TargetInfo{
						TargetNumber: card.TargetNumber,
						CardName:     card.CardName,
						Archers:      []ArcherInfo{},
					})
				}
			}

			c.JSON(http.StatusOK, gin.H{"targets": targets})
			return
		}

		if phase == "elimination" {
			if roundName == "" || categoryID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "round_name and category_id are required for elimination phase"})
				return
			}

			// Get matches for this round and their target assignments
			type EliminationTarget struct {
				MatchUUID      string `json:"match_id" db:"match_uuid"`
				MatchOrder     int    `json:"match_order" db:"match_order"`
				TargetNumber   int    `json:"target_number" db:"target_number"`
				TargetPosition string `json:"target_position,omitempty" db:"target_position"`
				Status         string `json:"status" db:"status"`
			}

			var targets []EliminationTarget
			err := db.Select(&targets, `
				SELECT 
					m.uuid as match_uuid,
					m.match_order,
					mta.target_number,
					mta.target_position,
					m.status
				FROM matches m
				LEFT JOIN match_target_assignments mta ON m.uuid = mta.match_uuid
				WHERE m.event_category_uuid = ? AND m.round_name = ?
				ORDER BY m.match_order ASC, mta.target_number ASC`,
				categoryID, roundName)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch targets"})
				return
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

		// Get qualification target names (from sessions)
		qualificationNames := []TargetName{}
		err = db.Select(&qualificationNames, `
			SELECT 
				CONCAT('qualification-', ec.uuid, '-sesi-', qs.session_order) as id,
				CONCAT(bt.name, ' - ', ag.name, ' - ', et.name, ' - ', gd.name, ' (', qs.session_name, ')') as name,
				'qualification' as phase,
				ec.uuid as category_id,
				qs.uuid as session_id,
				qs.session_order
			FROM qualification_sessions qs
			JOIN event_categories ec ON qs.event_category_uuid = ec.uuid
			JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			JOIN ref_event_types et ON ec.event_type_uuid = et.uuid
			JOIN ref_gender_divisions gd ON ec.gender_division_uuid = gd.uuid
			WHERE ec.event_id = ? OR ec.event_id = (SELECT uuid FROM events WHERE slug = ?)
			ORDER BY qs.session_order ASC, bt.name ASC, ag.name ASC
		`, eventID, eventID)

		if err == nil {
			targetNames = append(targetNames, qualificationNames...)
		}

		// Get elimination target names (from matches/rounds)
		eliminationNames := []TargetName{}
		err = db.Select(&eliminationNames, `
			SELECT DISTINCT
				CONCAT('elimination-', ec.uuid, '-', m.round_name) as id,
				CONCAT(bt.name, ' - ', ag.name, ' - ', et.name, ' - ', gd.name, ' (', m.round_name, ')') as name,
				'elimination' as phase,
				ec.uuid as category_id,
				'' as session_id,
				0 as session_order,
				m.round_name
			FROM matches m
			JOIN event_categories ec ON m.event_category_uuid = ec.uuid
			JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			JOIN ref_event_types et ON ec.event_type_uuid = et.uuid
			JOIN ref_gender_divisions gd ON ec.gender_division_uuid = gd.uuid
			WHERE ec.event_id = ? OR ec.event_id = (SELECT uuid FROM events WHERE slug = ?)
			ORDER BY 
				CASE m.round_name
					WHEN '1/32' THEN 1
					WHEN '1/16' THEN 2
					WHEN '1/8' THEN 3
					WHEN '1/4' THEN 4
					WHEN 'Semi-Final' THEN 5
					WHEN 'Final' THEN 6
					ELSE 7
				END,
				bt.name ASC, ag.name ASC
		`, eventID, eventID)

		if err == nil {
			targetNames = append(targetNames, eliminationNames...)
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
			SessionUUID    string `json:"session_id" binding:"required"`
			ParticipantUUID string `json:"participant_id" binding:"required"`
			TargetNumber   int    `json:"target_number" binding:"required"`
			TargetPosition string `json:"target_position" binding:"required"` // A, B, C, D
			AssignmentUUID *string `json:"assignment_id,omitempty"` // If provided, update existing; otherwise create new
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

		// Check if position is already taken by another participant
		var existingAssignment string
		err := db.Get(&existingAssignment, `
			SELECT uuid FROM qualification_assignments 
			WHERE session_uuid = ? AND target_number = ? AND target_position = ? 
			AND (uuid != COALESCE(?, '') OR participant_uuid != ?)
		`, req.SessionUUID, req.TargetNumber, req.TargetPosition, req.AssignmentUUID, req.ParticipantUUID)

		if err == nil && existingAssignment != "" {
			c.JSON(http.StatusConflict, gin.H{"error": "Target position already assigned to another archer"})
			return
		}

		// Check if participant already has an assignment in this session
		var existingParticipantAssignment string
		err = db.Get(&existingParticipantAssignment, `
			SELECT uuid FROM qualification_assignments 
			WHERE session_uuid = ? AND participant_uuid = ? AND uuid != COALESCE(?, '')
		`, req.SessionUUID, req.ParticipantUUID, req.AssignmentUUID)

		if err == nil && existingParticipantAssignment != "" {
			// Update existing assignment for this participant
			_, err = db.Exec(`
				UPDATE qualification_assignments 
				SET target_number = ?, target_position = ?, updated_at = NOW()
				WHERE uuid = ? AND session_uuid = ?
			`, req.TargetNumber, req.TargetPosition, existingParticipantAssignment, req.SessionUUID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Assignment updated successfully",
				"assignment_id": existingParticipantAssignment,
			})
			return
		}

		if req.AssignmentUUID != nil && *req.AssignmentUUID != "" {
			// Update existing assignment
			_, err = db.Exec(`
				UPDATE qualification_assignments 
				SET target_number = ?, target_position = ?, updated_at = NOW()
				WHERE uuid = ? AND session_uuid = ?
			`, req.TargetNumber, req.TargetPosition, *req.AssignmentUUID, req.SessionUUID)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update assignment"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Assignment updated successfully",
				"assignment_id": *req.AssignmentUUID,
			})
		} else {
			// Create new assignment
			newUUID := uuid.New().String()
			_, err = db.Exec(`
				INSERT INTO qualification_assignments (uuid, session_uuid, participant_uuid, target_number, target_position)
				VALUES (?, ?, ?, ?, ?)
			`, newUUID, req.SessionUUID, req.ParticipantUUID, req.TargetNumber, req.TargetPosition)

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create assignment"})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"message": "Assignment created successfully",
				"assignment_id": newUUID,
			})
		}
	}
}

// DeleteQualificationAssignment removes an assignment
func DeleteQualificationAssignment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("id")

		_, err := db.Exec(`DELETE FROM qualification_assignments WHERE uuid = ?`, assignmentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete assignment"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Assignment deleted successfully"})
	}
}

// CreateTargetCard creates a new target card
func CreateTargetCard(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			SessionUUID  *string `json:"session_id"`   // For qualification
			MatchUUID    *string `json:"match_id"`     // For elimination
			TargetNumber int     `json:"target_number" binding:"required"`
			CardName     string  `json:"card_name" binding:"required"`
			Phase        string  `json:"phase" binding:"required"` // "qualification" or "elimination"
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Validate phase-specific requirements
		if req.Phase == "qualification" && (req.SessionUUID == nil || *req.SessionUUID == "") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id is required for qualification phase"})
			return
		}

		if req.Phase == "elimination" && (req.MatchUUID == nil || *req.MatchUUID == "") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "match_id is required for elimination phase"})
			return
		}

		// Check if target card already exists
		var existingCard string
		var checkErr error
		if req.Phase == "qualification" {
			checkErr = db.Get(&existingCard, `
				SELECT uuid FROM target_cards 
				WHERE session_uuid = ? AND target_number = ?
			`, *req.SessionUUID, req.TargetNumber)
		} else {
			checkErr = db.Get(&existingCard, `
				SELECT uuid FROM target_cards 
				WHERE match_uuid = ? AND target_number = ?
			`, *req.MatchUUID, req.TargetNumber)
		}

		if checkErr == nil && existingCard != "" {
			c.JSON(http.StatusConflict, gin.H{"error": "Target card already exists for this target number"})
			return
		}

		// Create new target card
		newUUID := uuid.New().String()
		var execErr error
		if req.Phase == "qualification" {
			_, execErr = db.Exec(`
				INSERT INTO target_cards (uuid, session_uuid, target_number, card_name, phase, status)
				VALUES (?, ?, ?, ?, ?, 'active')
			`, newUUID, *req.SessionUUID, req.TargetNumber, req.CardName, req.Phase)
		} else {
			_, execErr = db.Exec(`
				INSERT INTO target_cards (uuid, match_uuid, target_number, card_name, phase, status)
				VALUES (?, ?, ?, ?, ?, 'active')
			`, newUUID, *req.MatchUUID, req.TargetNumber, req.CardName, req.Phase)
		}

		if execErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create target card"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message": "Target card created successfully",
			"card": gin.H{
				"id":           newUUID,
				"target_number": req.TargetNumber,
				"card_name":    req.CardName,
				"phase":        req.Phase,
			},
		})
	}
}

// GetTargetCards returns all target cards for a given context
func GetTargetCards(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		phase := c.Query("phase")
		sessionID := c.Query("session_id")
		matchID := c.Query("match_id")

		type TargetCard struct {
			UUID         string `json:"id" db:"uuid"`
			TargetNumber int    `json:"target_number" db:"target_number"`
			CardName     string `json:"card_name" db:"card_name"`
			Phase        string `json:"phase" db:"phase"`
			Status       string `json:"status" db:"status"`
		}

		var cards []TargetCard
		var err error

		if phase == "qualification" && sessionID != "" {
			err = db.Select(&cards, `
				SELECT uuid, target_number, card_name, phase, status
				FROM target_cards
				WHERE session_uuid = ? AND phase = 'qualification'
				ORDER BY target_number ASC
			`, sessionID)
		} else if phase == "elimination" && matchID != "" {
			err = db.Select(&cards, `
				SELECT uuid, target_number, card_name, phase, status
				FROM target_cards
				WHERE match_uuid = ? AND phase = 'elimination'
				ORDER BY target_number ASC
			`, matchID)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid parameters"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch target cards"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"cards": cards})
	}
}
