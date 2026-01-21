package handler

import (
	"archeryhub-api/models"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"github.com/google/uuid"
	"strconv"
)

// GetQualificationSessions returns all sessions for a category
func GetQualificationSessions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		categoryID := c.Query("category_id")
		if categoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
			return
		}

		var sessions []models.QualificationSession
		err := db.Select(&sessions, "SELECT * FROM qualification_sessions WHERE event_category_uuid = ? ORDER BY session_order ASC", categoryID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sessions"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"sessions": sessions})
	}
}

// CreateQualificationSession creates a new scoring session
func CreateQualificationSession(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			EventCategoryUUID string `json:"event_category_id" binding:"required"`
			SessionName       string `json:"session_name" binding:"required"`
			SessionOrder      int    `json:"session_order"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		newUUID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO qualification_sessions (uuid, event_category_uuid, session_name, session_order, status)
			VALUES (?, ?, ?, ?, 'draft')`,
			newUUID, req.EventCategoryUUID, req.SessionName, req.SessionOrder)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": newUUID, "message": "Session created successfully"})
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

		// Prepare scores for storage (ensure 6 arrows even if less provided)
		arrows := make([]interface{}, 6)
		for i := 0; i < 6; i++ {
			if i < len(req.Arrows) {
				arrows[i] = req.Arrows[i]
			} else {
				arrows[i] = nil
			}
		}

		scoreUUID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO qualification_end_scores 
			(uuid, assignment_uuid, end_number, arrow_1, arrow_2, arrow_3, arrow_4, arrow_5, arrow_6, end_total, end_x_count, end_10_count)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON DUPLICATE KEY UPDATE 
			arrow_1=VALUES(arrow_1), arrow_2=VALUES(arrow_2), arrow_3=VALUES(arrow_3), 
			arrow_4=VALUES(arrow_4), arrow_5=VALUES(arrow_5), arrow_6=VALUES(arrow_6), 
			end_total=VALUES(end_total), end_x_count=VALUES(end_x_count), end_10_count=VALUES(end_10_count)`,
			scoreUUID, assignmentID, req.EndNumber, arrows[0], arrows[1], arrows[2], arrows[3], arrows[4], arrows[5], total, xCount, tenCount)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update score"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Score updated successfully", "total": total})
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
		categoryID := c.Param("categoryId")

		type Entry struct {
			ArcherName    string `json:"archer_name" db:"name"`
			TotalScore    int    `json:"total_score" db:"total_score"`
			TotalTenX     int    `json:"total_10x" db:"total_10x"`
			TotalX        int    `json:"total_x" db:"total_x"`
			EndsCompleted int    `json:"ends_completed" db:"ends_completed"`
		}

		var leaderboard []Entry
		err := db.Select(&leaderboard, `
			SELECT 
				a.name,
				COALESCE(SUM(s.end_total), 0) as total_score,
				COALESCE(SUM(s.end_10_count), 0) as total_10x,
				COALESCE(SUM(s.end_x_count), 0) as total_x,
				COUNT(s.uuid) as ends_completed
			FROM archers a
			JOIN event_participants ep ON ep.archer_uuid = a.uuid
			JOIN qualification_assignments qa ON qa.participant_uuid = ep.uuid
			LEFT JOIN qualification_end_scores s ON s.assignment_uuid = qa.uuid
			WHERE ep.event_category_uuid = ?
			GROUP BY a.uuid, a.name
			ORDER BY total_score DESC, total_10x DESC, total_x DESC`,
			categoryID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
	}
}
