package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

// GetScoringCards returns selectable "card target" options for scoring context.
// For now it supports qualification phase and returns cards across sessions for a given event category.
func GetScoringCards(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		phase := c.Query("phase")
		categoryID := c.Query("category_id")

		if phase == "" || categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phase and category_id are required"})
			return
		}

		// Qualification: list all target_numbers for all sessions in this event (derived from category_id),
		// and attach target_name if any.
		if phase == "qualification" {
			type Row struct {
				ID           string `db:"id" json:"id"`
				Label        string `db:"label" json:"label"`
				Phase        string `db:"phase" json:"phase"`
				SessionID    string `db:"session_id" json:"session_id"`
				SessionName  string `db:"session_name" json:"session_name"`
				SessionOrder int    `db:"session_order" json:"session_order"`
				TargetName   string `db:"target_name" json:"target_name"`
				CardName     string `db:"card_name" json:"card_name"`
			}

			var rows []Row
			err := db.Select(&rows, `
				SELECT
					CONCAT(qs.uuid, '-', et.uuid) as id,
					CONCAT(qs.name, ' - ', et.target_name, 
						COALESCE(CONCAT(' [', (
							SELECT GROUP_CONCAT(COALESCE(a2.full_name, '-') ORDER BY qta2.target_position SEPARATOR ', ')
							FROM qualification_target_assignments qta2
							JOIN event_participants ep2 ON qta2.participant_uuid = ep2.uuid
							JOIN archers a2 ON ep2.archer_id = a2.uuid
							WHERE qta2.session_uuid = qs.uuid AND qta2.target_uuid = et.uuid
						), ']'), ' (Kosong)')) as label,
					'qualification' as phase,
					qs.uuid as session_id,
					qs.name as session_name,
					0 as session_order,
					0 as session_order,
					et.target_name,
					et.target_name as card_name
				FROM qualification_sessions qs
				JOIN event_targets et ON et.event_uuid = qs.event_uuid
				WHERE qs.event_uuid = (SELECT event_id FROM event_categories WHERE uuid = ?)
				ORDER BY qs.created_at ASC, et.target_name ASC
			`, categoryID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scoring cards"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"cards": rows})
			return
		}

		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported phase"})
	}
}

// GetScoringTargets returns scoring progress for a selected target name in a session.
// Qualification-only for now.
func GetScoringTargets(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		phase := c.Query("phase")
		sessionID := c.Query("session_id")
		targetNameStr := c.Query("target_name")

		if phase == "" || sessionID == "" || targetNameStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "phase, session_id, and target_name are required"})
			return
		}

		targetName := targetNameStr

		if phase != "qualification" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported phase"})
			return
		}

		// Resolve target name from event_targets
		cardName := ""
		_ = db.Get(&cardName, `
			SELECT et.target_name
			FROM event_targets et
			JOIN qualification_sessions qs ON qs.event_uuid = et.event_uuid
			WHERE qs.uuid = ? AND et.target_name = ?
			LIMIT 1
		`, sessionID, targetName)
		if cardName == "" {
			cardName = "Target " + targetName
		}

		// Compute total ends for the event (fallback 12)
		totalEnds := 12
		var qualificationArrows *int
		_ = db.Get(&qualificationArrows, `
			SELECT e.qualification_arrows
			FROM qualification_sessions qs
			JOIN events e ON qs.event_uuid = e.uuid
			WHERE qs.uuid = ?
			LIMIT 1
		`, sessionID)
		if qualificationArrows != nil && *qualificationArrows > 0 {
			// arrows_per_end is fixed 6 in our scoring request
			totalEnds = (*qualificationArrows + 5) / 6
		}

		type ArcherRow struct {
			AssignmentID  string `db:"assignment_id" json:"assignment_id"`
			ParticipantID string `db:"participant_id" json:"participant_id"`
			Position      string `db:"position" json:"position"`
			Name          string `db:"name" json:"name"`
			Division      string `db:"division" json:"division"`
			CurrentScore  int    `db:"current_score" json:"current_score"`
			EndsCompleted int    `db:"ends_completed" json:"ends_completed"`
		}

		var archers []ArcherRow
		err := db.Select(&archers, `
			SELECT
				qta.uuid as assignment_id,
				qta.participant_uuid as participant_id,
				qta.target_position as position,
				COALESCE(a.full_name, '') as name,
				COALESCE(CONCAT(bt.name, ' ', ag.name), '') as division,
				COALESCE(SUM(qes.total_score_end), 0) as current_score,
				COUNT(qes.uuid) as ends_completed
			FROM qualification_target_assignments qta
			JOIN event_targets et ON qta.target_uuid = et.uuid
			JOIN event_participants ep ON qta.participant_uuid = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			LEFT JOIN qualification_end_scores qes ON qes.participant_uuid = ep.uuid AND qes.session_uuid = qta.session_uuid
			WHERE qta.session_uuid = ? AND et.target_name = ?
			GROUP BY qta.uuid, qta.participant_uuid, qta.target_position, a.full_name, bt.name, ag.name
			ORDER BY qta.target_position ASC
		`, sessionID, targetName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch scoring targets"})
			return
		}

		status := "pending"
		if len(archers) > 0 {
			status = "live"
		}
		completed := 0
		for _, a := range archers {
			if a.EndsCompleted > completed {
				completed = a.EndsCompleted
			}
		}
		if completed >= totalEnds && len(archers) > 0 {
			status = "completed"
		}

		c.JSON(http.StatusOK, gin.H{
			"targets": []gin.H{
				{
					"id":             sessionID + "-" + targetName,
					"target_name":    targetName,
					"display_name":   cardName,
					"status":         status,
					"completed_ends": completed,
					"total_ends":     totalEnds,
					"archers":        archers,
				},
			},
		})
	}
}
