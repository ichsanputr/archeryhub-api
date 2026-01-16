package handler

import (
	"fmt"
	"net/http"

	"archeryhub-api/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Award represents a medal/award given to an athlete or team
type Award struct {
	ID           string  `json:"id" db:"id"`
	TournamentID string  `json:"tournament_id" db:"tournament_id"`
	EventID      string  `json:"event_id" db:"event_id"`
	RecipientID  string  `json:"recipient_id" db:"recipient_id"` // participant_id or team_id
	RecipientType string `json:"recipient_type" db:"recipient_type"` // individual, team
	AwardType    string  `json:"award_type" db:"award_type"` // gold, silver, bronze, participation
	Rank         int     `json:"rank" db:"rank"`
	AwardedAt    string  `json:"awarded_at" db:"awarded_at"`
	AwardedBy    *string `json:"awarded_by" db:"awarded_by"`
	Notes        *string `json:"notes" db:"notes"`
}

// AwardWithDetails includes recipient info
type AwardWithDetails struct {
	Award
	RecipientName string  `json:"recipient_name" db:"recipient_name"`
	Country       *string `json:"country" db:"country"`
	EventName     string  `json:"event_name" db:"event_name"`
}

// CreateAward creates a new award entry
func CreateAward(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		userID, _ := c.Get("user_id")

		var req struct {
			EventID       string  `json:"event_id" binding:"required"`
			RecipientID   string  `json:"recipient_id" binding:"required"`
			RecipientType string  `json:"recipient_type" binding:"required,oneof=individual team"`
			AwardType     string  `json:"award_type" binding:"required,oneof=gold silver bronze participation"`
			Rank          int     `json:"rank" binding:"required"`
			Notes         *string `json:"notes"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		awardID := uuid.New().String()

		_, err := db.Exec(`
			INSERT INTO awards 
			(id, tournament_id, event_id, recipient_id, recipient_type, award_type, rank, awarded_by, notes)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, awardID, tournamentID, req.EventID, req.RecipientID, req.RecipientType, req.AwardType, req.Rank, userID.(string), req.Notes)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create award"})
			return
		}

		utils.LogActivity(db, userID.(string), tournamentID, "award_created", "award", awardID,
			fmt.Sprintf("Awarded %s medal", req.AwardType), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      awardID,
			"message": "Award created successfully",
		})
	}
}

// GetAwards returns all awards for a tournament
func GetAwards(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")
		eventID := c.Query("event_id")

		query := `
			SELECT 
				a.*,
				CASE 
					WHEN a.recipient_type = 'individual' THEN CONCAT(ath.first_name, ' ', ath.last_name)
					ELSE t.team_name
				END as recipient_name,
				CASE 
					WHEN a.recipient_type = 'individual' THEN ath.country
					ELSE t.country_code
				END as country,
				CONCAT(d.name, ' - ', cat.name) as event_name
			FROM awards a
			LEFT JOIN tournament_participants tp ON a.recipient_id = tp.id AND a.recipient_type = 'individual'
			LEFT JOIN athletes ath ON tp.athlete_id = ath.id
			LEFT JOIN teams t ON a.recipient_id = t.id AND a.recipient_type = 'team'
			LEFT JOIN tournament_events te ON a.event_id = te.id
			LEFT JOIN divisions d ON te.division_id = d.id
			LEFT JOIN categories cat ON te.category_id = cat.id
			WHERE a.tournament_id = ?
		`
		args := []interface{}{tournamentID}

		if eventID != "" {
			query += " AND a.event_id = ?"
			args = append(args, eventID)
		}

		query += " ORDER BY a.rank ASC"

		var awards []AwardWithDetails
		err := db.Select(&awards, query, args...)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch awards"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"awards": awards,
			"total":  len(awards),
		})
	}
}

// GetMedalTable returns a summary of medals by country
func GetMedalTable(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("tournamentId")

		type MedalCount struct {
			Country string `json:"country" db:"country"`
			Gold    int    `json:"gold" db:"gold"`
			Silver  int    `json:"silver" db:"silver"`
			Bronze  int    `json:"bronze" db:"bronze"`
			Total   int    `json:"total" db:"total"`
		}

		var medals []MedalCount
		err := db.Select(&medals, `
			SELECT 
				COALESCE(
					CASE 
						WHEN a.recipient_type = 'individual' THEN ath.country
						ELSE t.country_code
					END, 'Unknown'
				) as country,
				SUM(CASE WHEN a.award_type = 'gold' THEN 1 ELSE 0 END) as gold,
				SUM(CASE WHEN a.award_type = 'silver' THEN 1 ELSE 0 END) as silver,
				SUM(CASE WHEN a.award_type = 'bronze' THEN 1 ELSE 0 END) as bronze,
				COUNT(*) as total
			FROM awards a
			LEFT JOIN tournament_participants tp ON a.recipient_id = tp.id AND a.recipient_type = 'individual'
			LEFT JOIN athletes ath ON tp.athlete_id = ath.id
			LEFT JOIN teams t ON a.recipient_id = t.id AND a.recipient_type = 'team'
			WHERE a.tournament_id = ? AND a.award_type IN ('gold', 'silver', 'bronze')
			GROUP BY country
			ORDER BY gold DESC, silver DESC, bronze DESC
		`, tournamentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch medal table"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"medals":    medals,
			"countries": len(medals),
		})
	}
}

// AutoAwardMedals automatically creates awards based on final rankings
func AutoAwardMedals(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("eventId")
		userID, _ := c.Get("user_id")

		// Get tournament ID
		var tournamentID string
		err := db.Get(&tournamentID, "SELECT tournament_id FROM tournament_events WHERE id = ?", eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		// Get gold medal winner (GM match winner)
		var goldWinner *string
		db.Get(&goldWinner, `
			SELECT winner_id FROM elimination_matches 
			WHERE event_id = ? AND round = 'GM' AND status = 'completed'
		`, eventID)

		// Get silver medal (GM match loser)
		var silverWinner *string
		db.Get(&silverWinner, `
			SELECT CASE 
				WHEN winner_id = participant1_id THEN participant2_id 
				ELSE participant1_id 
			END FROM elimination_matches 
			WHERE event_id = ? AND round = 'GM' AND status = 'completed'
		`, eventID)

		// Get bronze medal winners (BM match winner)
		var bronzeWinner *string
		db.Get(&bronzeWinner, `
			SELECT winner_id FROM elimination_matches 
			WHERE event_id = ? AND round = 'BM' AND status = 'completed'
		`, eventID)

		awardsCreated := 0

		// Create awards
		if goldWinner != nil {
			id := uuid.New().String()
			db.Exec(`INSERT INTO awards (id, tournament_id, event_id, recipient_id, recipient_type, award_type, rank, awarded_by) 
				VALUES (?, ?, ?, ?, 'individual', 'gold', 1, ?)`, id, tournamentID, eventID, *goldWinner, userID.(string))
			awardsCreated++
		}

		if silverWinner != nil {
			id := uuid.New().String()
			db.Exec(`INSERT INTO awards (id, tournament_id, event_id, recipient_id, recipient_type, award_type, rank, awarded_by) 
				VALUES (?, ?, ?, ?, 'individual', 'silver', 2, ?)`, id, tournamentID, eventID, *silverWinner, userID.(string))
			awardsCreated++
		}

		if bronzeWinner != nil {
			id := uuid.New().String()
			db.Exec(`INSERT INTO awards (id, tournament_id, event_id, recipient_id, recipient_type, award_type, rank, awarded_by) 
				VALUES (?, ?, ?, ?, 'individual', 'bronze', 3, ?)`, id, tournamentID, eventID, *bronzeWinner, userID.(string))
			awardsCreated++
		}

		utils.LogActivity(db, userID.(string), tournamentID, "medals_awarded", "award", eventID,
			fmt.Sprintf("Auto-awarded %d medals", awardsCreated), c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{
			"message":        "Medals awarded successfully",
			"awards_created": awardsCreated,
		})
	}
}
