package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// EventPaymentMethod represents a payment method for an event
type EventPaymentMethod struct {
	UUID          string  `json:"uuid" db:"uuid"`
	EventID       string  `json:"event_id" db:"event_id"`
	PaymentMethod string  `json:"payment_method" db:"payment_method"`
	AccountName   *string `json:"account_name" db:"account_name"`
	AccountNumber *string `json:"account_number" db:"account_number"`
	Instructions  *string `json:"instructions" db:"instructions"`
	IsActive      bool    `json:"is_active" db:"is_active"`
	DisplayOrder  int     `json:"display_order" db:"display_order"`
	CreatedAt     string  `json:"created_at" db:"created_at"`
	UpdatedAt     string  `json:"updated_at" db:"updated_at"`
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
			UUID         string  `json:"uuid" db:"uuid"`
			Slug         *string `json:"slug" db:"slug"`
			Email        *string `json:"email" db:"email"`
			AvatarURL    *string `json:"avatar_url" db:"avatar_url"`
			FullName     string  `json:"full_name" db:"full_name"`
			Nickname     *string `json:"nickname" db:"nickname"`
			DateOfBirth  *string `json:"date_of_birth" db:"date_of_birth"`
			Gender       string  `json:"gender" db:"gender"`
			Phone        *string `json:"phone" db:"phone"`
			Address      *string `json:"address" db:"address"`
			City         *string `json:"city" db:"city"`
			Province     *string `json:"province" db:"province"`
			BowType      string  `json:"bow_type" db:"bow_type"`
			ClubID       *string `json:"club_id" db:"club_id"`
			ClubName     *string `json:"club_name" db:"club_name"`
			Achievements *string `json:"achievements" db:"achievements"`
			Status       string  `json:"status" db:"status"`
		}

		var pageSettings *string
		err := db.Get(&archer, `
		SELECT a.uuid, a.username, a.email, a.avatar_url, 
		       a.full_name, a.nickname, a.date_of_birth, 
		       COALESCE(a.gender, 'male') as gender,
		       a.phone, a.address, a.city, a.province, 
		       COALESCE(a.bow_type, 'recurve') as bow_type,
		       a.club_id, c.name as club_name,
		       a.achievements,
		       COALESCE(a.status, 'active') as status
		FROM archers a
		LEFT JOIN clubs c ON a.club_id = c.uuid
		WHERE a.uuid = ? OR a.user_id = ? OR a.email = (SELECT email FROM archers WHERE uuid = ? LIMIT 1)
	`, userID, userID, userID)

		if err == nil {
			db.Get(&pageSettings, "SELECT page_settings FROM archers WHERE uuid = ? OR user_id = ?", userID, userID)
		}

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Archer profile not found"})
			return
		}

		response := gin.H{
			"uuid":          archer.UUID,
			"username":      archer.Username,
			"email":         archer.Email,
			"avatar_url":    archer.AvatarURL,
			"full_name":     archer.FullName,
			"nickname":      archer.Nickname,
			"date_of_birth": archer.DateOfBirth,
			"gender":        archer.Gender,
			"phone":         archer.Phone,
			"address":       archer.Address,
			"city":          archer.City,
			"province":      archer.Province,
			"bow_type":      archer.BowType,
			"club_id":       archer.ClubID,
			"club_name":     archer.ClubName,
			"achievements":  archer.Achievements,
			"status":        archer.Status,
		}

		if pageSettings != nil {
			response["page_settings"] = pageSettings
		}

		c.JSON(http.StatusOK, response)
	}
}
