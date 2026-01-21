package handler

import (
	"net/http"

	"archeryhub-api/utils"
	"github.com/gin-gonic/gin"
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

// GetEventEvents returns events for a specific event
func GetEventEvents(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		type EventEvent struct {
			ID                  string `db:"id" json:"id"`
			EventID        string `db:"event_id" json:"event_id"`
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

		var events []EventEvent
		err := db.Select(&events, `
			SELECT 
				te.id, te.event_id, te.division_id, te.category_id,
				te.max_participants, te.qualification_arrows, te.elimination_format, te.team_event,
				te.created_at,
				d.name as division_name, d.code as division_code,
				c.name as category_name, c.code as category_code
			FROM event_categories te
			JOIN divisions d ON te.division_id = d.id
			JOIN categories c ON te.category_id = c.id
			WHERE te.event_id = ?
			ORDER BY d.display_order, c.display_order
		`, eventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch event events"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"events": events,
			"total":  len(events),
		})
	}
}

// GetEventParticipants returns participants for a specific event
func GetEventParticipants(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		type Participant struct {
			ID                  string  `db:"id" json:"id"`
			AthleteID           string  `db:"athlete_id" json:"athlete_id"`
			FullName            string  `db:"full_name" json:"full_name"`
			AthleteCode         string  `db:"athlete_code" json:"athlete_code"`
			Country             *string `db:"country" json:"country"`
			ClubID              *string `db:"club_id" json:"club_id"`
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
				a.full_name, a.athlete_code, a.country, a.club_id,
				d.name as division_name, c.name as category_name
			FROM event_participants tp
			JOIN archers a ON tp.athlete_id = a.id
			JOIN event_categories te ON tp.event_id = te.id
			JOIN divisions d ON te.division_id = d.id
			JOIN categories c ON te.category_id = c.id
			WHERE tp.event_id = ?
			ORDER BY d.display_order, c.display_order, a.full_name
		`, eventID)

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

// PublishEvent changes event status to published
func PublishEvent(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		_, err := db.Exec("UPDATE events SET status = 'published' WHERE id = ?", eventID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish event"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "event_published", "event", eventID, "Published event", c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusOK, gin.H{"message": "Event published successfully"})
	}
}

// RegisterParticipant registers a participant for a event
func RegisterParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

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
			SELECT EXISTS(SELECT 1 FROM event_participants 
			WHERE event_id = ? AND athlete_id = ? AND event_id = ?)
		`, eventID, req.AthleteID, req.EventID)

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
			INSERT INTO event_participants 
			(id, event_id, athlete_id, event_id, payment_amount, payment_status, accreditation_status)
			VALUES (?, ?, ?, ?, ?, 'pending', 'pending')
		`, participantID, eventID, req.AthleteID, req.EventID, req.PaymentAmount)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register participant"})
			return
		}

		// Log activity
		userID, _ := c.Get("user_id")
		utils.LogActivity(db, userID.(string), eventID, "participant_registered", "event_participant", participantID, "Registered participant for event: "+req.EventID, c.ClientIP(), c.Request.UserAgent())

		c.JSON(http.StatusCreated, gin.H{
			"id":      participantID,
			"message": "Participant registered successfully",
		})
	}
}
