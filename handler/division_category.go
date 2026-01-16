package handler

import (
	"net/http"

	"archeryhub/utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// generateUUID generates a new UUID
func generateUUID() string {
	return uuid.New().String()
}

type Division struct {
	ID           string `db:"id" json:"id"`
	Name         string `db:"name" json:"name"`
	Code         string `db:"code" json:"code"`
	Description  string `db:"description" json:"description"`
	DisplayOrder int    `db:"display_order" json:"display_order"`
	CreatedAt    string `db:"created_at" json:"created_at"`
}

type Category struct {
	ID           string `db:"id" json:"id"`
	Name         string `db:"name" json:"name"`
	Code         string `db:"code" json:"code"`
	AgeFrom      *int   `db:"age_from" json:"age_from"`
	AgeTo        *int   `db:"age_to" json:"age_to"`
	Gender       string `db:"gender" json:"gender"`
	DisplayOrder int    `db:"display_order" json:"display_order"`
	CreatedAt    string `db:"created_at" json:"created_at"`
}

// GetDivisions returns all available divisions
func GetDivisions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var divisions []Division
		err := db.Select(&divisions, "SELECT * FROM divisions ORDER BY display_order ASC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch divisions"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"divisions": divisions,
			"total":     len(divisions),
		})
	}
}

// GetCategories returns all available categories
func GetCategories(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		gender := c.Query("gender") // Optional filter by gender

		query := "SELECT * FROM categories"
		args := []interface{}{}

		if gender != "" {
			query += " WHERE gender = ?"
			args = append(args, gender)
		}

		query += " ORDER BY display_order ASC"

		var categories []Category
		err := db.Select(&categories, query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch categories"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"categories": categories,
			"total":      len(categories),
		})
	}
}

// GetTournamentEvents returns events for a specific tournament
func GetTournamentEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")

		type TournamentEvent struct {
			ID                  string `db:"id" json:"id"`
			TournamentID        string `db:"tournament_id" json:"tournament_id"`
			DivisionID          string `db:"division_id" json:"division_id"`
			DivisionName        string `db:"division_name" json:"division_name"`
			DivisionCode        string `db:"division_code" json:"division_code"`
			CategoryID          string `db:"category_id" json:"category_id"`
			CategoryName        string `db:"category_name" json:"category_name"`
			CategoryCode        string `db:"category_code" json:"category_code"`
			MaxParticipants     int    `db:"max_participants" json:"max_participants"`
			QualificationArrows int    `db:"qualification_arrows" json:"qualification_arrows"`
			EliminationFormat   string `db:"elimination_format" json:"elimination_format"`
			TeamEvent           bool   `db:"team_event" json:"team_event"`
			CreatedAt           string `db:"created_at" json:"created_at"`
		}

		var events []TournamentEvent
		err := db.Select(&events, `
			SELECT 
				te.id, te.tournament_id, te.division_id, te.category_id,
				te.max_participants, te.qualification_arrows, te.elimination_format, te.team_event,
				te.created_at,
				d.name as division_name, d.code as division_code,
				c.name as category_name, c.code as category_code
			FROM tournament_events te
			JOIN divisions d ON te.division_id = d.id
			JOIN categories c ON te.category_id = c.id
			WHERE te.tournament_id = ?
			ORDER BY d.display_order, c.display_order
		`, tournamentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tournament events"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"total":  len(events),
		})
	}
}

// GetTournamentParticipants returns participants for a specific tournament
func GetTournamentParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")

		type Participant struct {
			ID                  string  `db:"id" json:"id"`
			AthleteID           string  `db:"athlete_id" json:"athlete_id"`
			FirstName           string  `db:"first_name" json:"first_name"`
			LastName            string  `db:"last_name" json:"last_name"`
			AthleteCode         string  `db:"athlete_code" json:"athlete_code"`
			Country             *string `db:"country" json:"country"`
			Club                *string `db:"club" json:"club"`
			EventID             string  `db:"event_id" json:"event_id"`
			DivisionName        string  `db:"division_name" json:"division_name"`
			CategoryName        string  `db:"category_name" json:"category_name"`
			BackNumber          *string `db:"back_number" json:"back_number"`
			TargetNumber        *string `db:"target_number" json:"target_number"`
			Session             *int    `db:"session" json:"session"`
			PaymentStatus       string  `db:"payment_status" json:"payment_status"`
			AccreditationStatus string  `db:"accreditation_status" json:"accreditation_status"`
			RegistrationDate    string  `db:"registration_date" json:"registration_date"`
		}

		var participants []Participant
		err := db.Select(&participants, `
			SELECT 
				tp.id, tp.athlete_id, tp.event_id, tp.back_number, tp.target_number, tp.session,
				tp.payment_status, tp.accreditation_status, tp.registration_date,
				a.first_name, a.last_name, a.athlete_code, a.country, a.club,
				d.name as division_name, c.name as category_name
			FROM tournament_participants tp
			JOIN athletes a ON tp.athlete_id = a.id
			JOIN tournament_events te ON tp.event_id = te.id
			JOIN divisions d ON te.division_id = d.id
			JOIN categories c ON te.category_id = c.id
			WHERE tp.tournament_id = ?
			ORDER BY d.display_order, c.display_order, a.last_name, a.first_name
		`, tournamentID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch participants"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"participants": participants,
			"total":        len(participants),
		})
	}
}

// PublishTournament changes tournament status to published
func PublishTournament(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")

		_, err := db.Exec("UPDATE tournaments SET status = 'published' WHERE id = ?", tournamentID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish tournament"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), id, "tournament_published", "tournament", id, "Published tournament", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Tournament published successfully"})
	}
}

// RegisterParticipant registers a participant for a tournament
func RegisterParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tournamentID := c.Param("id")

		var req struct {
			AthleteID     string  `json:"athlete_id" binding:"required"`
			EventID       string  `json:"event_id" binding:"required"`
			PaymentAmount float64 `json:"payment_amount"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Check if already registered
		var exists bool
		err := db.Get(&exists, `
			SELECT EXISTS(SELECT 1 FROM tournament_participants 
			WHERE tournament_id = ? AND athlete_id = ? AND event_id = ?)
		`, tournamentID, req.AthleteID, req.EventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}

		if exists {
			c.JSON(http.StatusConflict, gin.H{"error": "Participant already registered for this event"})
			return
		}

		// Insert participant
		participantID := generateUUID()
		_, err = db.Exec(`
			INSERT INTO tournament_participants 
			(id, tournament_id, athlete_id, event_id, payment_amount, payment_status, accreditation_status)
			VALUES (?, ?, ?, ?, ?, 'pending', 'pending')
		`, participantID, tournamentID, req.AthleteID, req.EventID, req.PaymentAmount)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register participant"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), req.TournamentID, "participant_registered", "tournament_participant", participantID, "Registered participant for event: "+req.EventID, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      participantID,
			"message": "Participant registered successfully",
		})
	}
}
