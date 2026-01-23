package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// EventPaymentMethod represents a payment method for an event
type EventPaymentMethod struct {
	UUID           string  `json:"uuid" db:"uuid"`
	EventID        string  `json:"event_id" db:"event_id"`
	PaymentMethod  string  `json:"payment_method" db:"payment_method"`
	AccountName    *string `json:"account_name" db:"account_name"`
	AccountNumber  *string `json:"account_number" db:"account_number"`
	Instructions   *string `json:"instructions" db:"instructions"`
	IsActive       bool    `json:"is_active" db:"is_active"`
	DisplayOrder   int     `json:"display_order" db:"display_order"`
	CreatedAt      string  `json:"created_at" db:"created_at"`
	UpdatedAt      string  `json:"updated_at" db:"updated_at"`
}

// GetEventPaymentMethods returns all payment methods for an event
func GetEventPaymentMethods(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		// Check if it's a slug or UUID, get event UUID if it's a slug
		var actualEventID string
		err := db.Get(&actualEventID, "SELECT uuid FROM events WHERE uuid = ? OR slug = ? LIMIT 1", eventID, eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}

		var methods []EventPaymentMethod
		err = db.Select(&methods, `
			SELECT uuid, event_id, payment_method, account_name, account_number, 
			       instructions, is_active, display_order, created_at, updated_at
			FROM event_payment_methods
			WHERE event_id = ? AND is_active = TRUE
			ORDER BY display_order ASC, created_at ASC
		`, actualEventID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment methods"})
			return
		}

		if methods == nil {
			methods = []EventPaymentMethod{}
		}

		c.JSON(http.StatusOK, gin.H{"data": methods})
	}
}

// CreateEventPaymentMethod creates a new payment method for an event
func CreateEventPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var req struct {
			PaymentMethod string  `json:"payment_method" binding:"required"`
			AccountName   *string `json:"account_name"`
			AccountNumber *string `json:"account_number"`
			Instructions  *string `json:"instructions"`
			DisplayOrder  int     `json:"display_order"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		methodID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO event_payment_methods 
			(uuid, event_id, payment_method, account_name, account_number, instructions, display_order)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, methodID, eventID, req.PaymentMethod, req.AccountName, req.AccountNumber, req.Instructions, req.DisplayOrder)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment method"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"uuid":    methodID,
			"message": "Payment method created successfully",
		})
	}
}

// UpdateEventPaymentMethod updates a payment method
func UpdateEventPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		methodID := c.Param("methodId")

		var req struct {
			PaymentMethod string  `json:"payment_method"`
			AccountName   *string `json:"account_name"`
			AccountNumber *string `json:"account_number"`
			Instructions  *string `json:"instructions"`
			IsActive      *bool   `json:"is_active"`
			DisplayOrder  *int    `json:"display_order"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		_, err := db.Exec(`
			UPDATE event_payment_methods 
			SET payment_method = COALESCE(?, payment_method),
			    account_name = ?,
			    account_number = ?,
			    instructions = ?,
			    is_active = COALESCE(?, is_active),
			    display_order = COALESCE(?, display_order),
			    updated_at = NOW()
			WHERE uuid = ?
		`, req.PaymentMethod, req.AccountName, req.AccountNumber, req.Instructions, req.IsActive, req.DisplayOrder, methodID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment method"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Payment method updated successfully"})
	}
}

// DeleteEventPaymentMethod deletes a payment method
func DeleteEventPaymentMethod(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		methodID := c.Param("methodId")

		_, err := db.Exec("DELETE FROM event_payment_methods WHERE uuid = ?", methodID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment method"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Payment method deleted successfully"})
	}
}

// GetArcherProfile returns the authenticated archer's profile
func GetArcherProfile(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var archer struct {
			UUID                  string  `json:"uuid" db:"uuid"`
			Username              *string `json:"username" db:"username"`
			Slug                  *string `json:"slug" db:"slug"`
			Email                 *string `json:"email" db:"email"`
			AthleteCode           *string `json:"athlete_code" db:"athlete_code"`
			AvatarURL             *string `json:"avatar_url" db:"avatar_url"`
			FullName              string  `json:"full_name" db:"full_name"`
			Nickname              *string `json:"nickname" db:"nickname"`
			DateOfBirth           *string `json:"date_of_birth" db:"date_of_birth"`
			Gender                string  `json:"gender" db:"gender"`
			Country               *string `json:"country" db:"country"`
			Phone                 *string `json:"phone" db:"phone"`
			Address               *string `json:"address" db:"address"`
			City                  *string `json:"city" db:"city"`
			Province              *string `json:"province" db:"province"`
			PostalCode            *string `json:"postal_code" db:"postal_code"`
			NationalID            *string `json:"national_id" db:"national_id"`
			BowType               string  `json:"bow_type" db:"bow_type"`
			DominantHand          string  `json:"dominant_hand" db:"dominant_hand"`
			ExperienceYears       int     `json:"experience_years" db:"experience_years"`
			ClubID                *string `json:"club_id" db:"club_id"`
			ClubName              *string `json:"club_name" db:"club_name"`
			CurrentRanking        *int    `json:"current_ranking" db:"current_ranking"`
			BestScore             *int    `json:"best_score" db:"best_score"`
			EmergencyContactName  *string `json:"emergency_contact_name" db:"emergency_contact_name"`
			EmergencyContactPhone *string `json:"emergency_contact_phone" db:"emergency_contact_phone"`
			MedicalConditions     *string `json:"medical_conditions" db:"medical_conditions"`
			Achievements          *string `json:"achievements" db:"achievements"`
			Status                string  `json:"status" db:"status"`
		}

	err := db.Get(&archer, `
		SELECT a.uuid, a.username, a.slug, a.email, a.athlete_code, a.avatar_url, 
		       a.full_name, a.nickname, a.date_of_birth, 
		       COALESCE(a.gender, 'male') as gender,
		       a.country, a.phone, a.address, a.city, a.province, a.postal_code, 
		       a.national_id, 
		       COALESCE(a.bow_type, 'recurve') as bow_type,
		       COALESCE(a.dominant_hand, 'right') as dominant_hand,
		       COALESCE(a.experience_years, 0) as experience_years,
		       a.club_id, c.name as club_name,
		       a.current_ranking, a.best_score,
		       a.emergency_contact_name, a.emergency_contact_phone,
		       a.medical_conditions, a.achievements,
		       COALESCE(a.status, 'active') as status
		FROM archers a
		LEFT JOIN clubs c ON a.club_id = c.uuid
		WHERE a.uuid = ? OR a.user_id = ? OR a.email = (SELECT email FROM archers WHERE uuid = ? LIMIT 1)
	`, userID, userID, userID)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer profile not found"})
			return
		}

		c.JSON(http.StatusOK, archer)
	}
}
