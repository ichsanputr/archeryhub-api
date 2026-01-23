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

// GetQualificationAssignmentScores returns all saved end scores for a single assignment.
func GetQualificationAssignmentScores(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		assignmentID := c.Param("assignmentId")
		if assignmentID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "assignmentId is required"})
			return
		}

		type EndScore struct {
			ID         string  `db:"uuid" json:"id"`
			EndNumber  int     `db:"end_number" json:"end_number"`
			Arrow1     *string `db:"arrow_1" json:"arrow_1"`
			Arrow2     *string `db:"arrow_2" json:"arrow_2"`
			Arrow3     *string `db:"arrow_3" json:"arrow_3"`
			Arrow4     *string `db:"arrow_4" json:"arrow_4"`
			Arrow5     *string `db:"arrow_5" json:"arrow_5"`
			Arrow6     *string `db:"arrow_6" json:"arrow_6"`
			EndTotal   int     `db:"end_total" json:"end_total"`
			EndXCount  int     `db:"end_x_count" json:"end_x_count"`
			End10Count int     `db:"end_10_count" json:"end_10_count"`
		}

		var scores []EndScore
		err := db.Select(&scores, `
			SELECT uuid, end_number, arrow_1, arrow_2, arrow_3, arrow_4, arrow_5, arrow_6, end_total, end_x_count, end_10_count
			FROM qualification_end_scores
			WHERE assignment_uuid = ?
			ORDER BY end_number ASC
		`, assignmentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch assignment scores"})
			return
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
		categoryID := c.Param("categoryId")

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
				COALESCE(SUM(s.end_total), 0) as total_score,
				COALESCE(SUM(s.end_10_count), 0) as total_10x,
				COALESCE(SUM(s.end_x_count), 0) as total_x,
				COUNT(s.uuid) as ends_completed
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN clubs cl ON a.club_id = cl.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_bow_types bt ON ec.division_uuid = bt.uuid
			LEFT JOIN ref_age_groups ag ON ec.category_uuid = ag.uuid
			LEFT JOIN qualification_assignments qa ON qa.participant_uuid = ep.uuid
			LEFT JOIN qualification_end_scores s ON s.assignment_uuid = qa.uuid
			WHERE ep.category_id = ?
			GROUP BY a.uuid, a.full_name, cl.name, bt.name, ag.name
			ORDER BY total_score DESC, total_10x DESC, total_x DESC`,
			categoryID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch leaderboard"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
	}
}
