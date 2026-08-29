package handler

import (
	"Archeris-api/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// GetMyQuota GET /organizers/me/quota
func GetMyQuota(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		var org struct {
			UUID          string `db:"uuid"`
			QuotaStandard int    `db:"quota_standard"`
			QuotaElite    int    `db:"quota_elite"`
		}
		if err := db.Get(&org, "SELECT uuid, quota_standard, quota_elite FROM organizers WHERE user_id = ?", userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get organizer"})
			return
		}

		var publishedCount int
		db.Get(&publishedCount, "SELECT COUNT(*) FROM events WHERE organizer_id = ? AND status = 'published' AND quota_type IS NOT NULL", org.UUID)

		c.JSON(http.StatusOK, gin.H{
			"quota_standard":         org.QuotaStandard,
			"quota_elite":            org.QuotaElite,
			"total_quota":            org.QuotaStandard + org.QuotaElite,
			"published_events_count": publishedCount,
		})
	}
}

// GetQuotaHistory GET /organizers/me/quota/history
func GetQuotaHistory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		var orgUUID string
		if err := db.Get(&orgUUID, "SELECT uuid FROM organizers WHERE user_id = ?", userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get organizer"})
			return
		}

		var history []struct {
			ID            int       `db:"id" json:"id"`
			Quantity      int       `db:"quantity" json:"quantity"`
			Amount        float64   `db:"amount" json:"amount"`
			PaymentStatus string    `db:"payment_status" json:"payment_status"`
			PurchasedAt   time.Time `db:"purchased_at" json:"purchased_at"`
			PlanName      string    `db:"plan_name" json:"plan_name"`
		}
		
		query := `SELECT q.id, q.quantity, q.amount, q.payment_status, q.purchased_at, p.name as plan_name 
				  FROM quota_purchases q 
				  JOIN subscription_plans p ON q.plan_id = p.id 
				  WHERE q.organizer_id = ? 
				  ORDER BY q.purchased_at DESC LIMIT 50`
		
		if err := db.Select(&history, query, orgUUID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get history"})
			return
		}

		// Ensure we don't return nil
		if history == nil {
			history = []struct {
				ID            int       `db:"id" json:"id"`
				Quantity      int       `db:"quantity" json:"quantity"`
				Amount        float64   `db:"amount" json:"amount"`
				PaymentStatus string    `db:"payment_status" json:"payment_status"`
				PurchasedAt   time.Time `db:"purchased_at" json:"purchased_at"`
				PlanName      string    `db:"plan_name" json:"plan_name"`
			}{}
		}

		c.JSON(http.StatusOK, history)
	}
}

// PurchaseQuota POST /organizers/me/quota/purchase
func PurchaseQuota(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		var req struct {
			PlanID        int    `json:"plan_id"`
			Quantity      int    `json:"quantity"`
			PaymentMethod string `json:"payment_method"`
			Currency      string `json:"currency"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		var org struct {
			UUID  string `db:"uuid"`
			Name  string `db:"name"`
			Email string `db:"email"`
			Phone string `db:"phone"`
		}
		if err := db.Get(&org, "SELECT uuid, name, email, phone FROM organizers WHERE user_id = ?", userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get organizer"})
			return
		}
		
		if org.Email == "" {
			db.Get(&org.Email, "SELECT email FROM users WHERE uuid = ?", userID)
		}

		var plan struct {
			Price float64 `db:"price"`
			Type  string  `db:"type"`
			Name  string  `db:"name"`
		}
		if err := db.Get(&plan, "SELECT price, type, name FROM subscription_plans WHERE id = ?", req.PlanID); err != nil || plan.Type != "quota" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan"})
			return
		}

		totalAmount := plan.Price * float64(req.Quantity)
		refID := fmt.Sprintf("QUOTA-%s-%d", strings.ToUpper(uuid.New().String()[:8]), time.Now().Unix())

		if req.PaymentMethod == "tripay" {
			tripay := utils.NewTripayClient()
			signature := tripay.GenerateSignature(refID, int(totalAmount))
			expiredTime := time.Now().Add(24 * time.Hour).Unix()
			
			orderItems := []map[string]interface{}{
				{
					"sku":         fmt.Sprintf("PLAN-%d", req.PlanID),
					"name":        fmt.Sprintf("Quota: %s x%d", plan.Name, req.Quantity),
					"price":       int(plan.Price),
					"quantity":    req.Quantity,
					"product_url": "",
					"image_url":   "",
				},
			}

			payload := gin.H{
				"method":         "MYBCAVA",
				"merchant_ref":   refID,
				"amount":         int(totalAmount),
				"customer_name":  org.Name,
				"customer_email": org.Email,
				"customer_phone": org.Phone,
				"order_items":    orderItems,
				"signature":      signature,
				"expired_time":   expiredTime,
			}
			
			res, err := tripay.CreateTransaction(payload)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment: " + err.Error()})
				return
			}
			
			var checkoutUrl string
			if url, ok := res["checkout_url"].(string); ok {
				checkoutUrl = url
			} else if data, ok := res["data"].(map[string]interface{}); ok {
				if url, ok := data["checkout_url"].(string); ok {
					checkoutUrl = url
				}
			}

			_, err = db.Exec(`INSERT INTO quota_purchases (organizer_id, plan_id, quantity, amount, currency, payment_method, payment_reference, payment_status, purchased_at) 
							  VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', NOW())`,
				org.UUID, req.PlanID, req.Quantity, totalAmount, req.Currency, req.PaymentMethod, refID)
			
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save purchase"})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"purchase_id":  refID,
				"checkout_url": checkoutUrl,
				"total_amount": totalAmount,
				"currency":     req.Currency,
			})
		} else {
			_, err := db.Exec(`INSERT INTO quota_purchases (organizer_id, plan_id, quantity, amount, currency, payment_method, payment_reference, payment_status, purchased_at) 
							  VALUES (?, ?, ?, ?, ?, ?, ?, 'pending', NOW())`,
				org.UUID, req.PlanID, req.Quantity, totalAmount, req.Currency, req.PaymentMethod, refID)
			
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save purchase"})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"purchase_id":  refID,
				"total_amount": totalAmount,
				"currency":     req.Currency,
				"instructions": "Use paddle flow",
			})
		}
	}
}

// QuotaTripayCallback POST /payment/quota/tripay/callback
func QuotaTripayCallback(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		bodyBytes, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Failed to read body"})
			return
		}

		tripay := utils.NewTripayClient()
		signature := c.GetHeader("X-Callback-Signature")
		if !tripay.VerifyCallbackSignature(bodyBytes, signature) {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid signature"})
			return
		}

		var payload struct {
			Reference string `json:"reference"`
			Status    string `json:"status"`
		}
		if err := json.Unmarshal(bodyBytes, &payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Invalid payload"})
			return
		}

		if payload.Status == "PAID" {
			var purchase struct {
				ID          int    `db:"id"`
				OrganizerID string `db:"organizer_id"`
				PlanID      int    `db:"plan_id"`
				Quantity    int    `db:"quantity"`
			}
			err := db.Get(&purchase, "SELECT id, organizer_id, plan_id, quantity FROM quota_purchases WHERE payment_reference = ?", payload.Reference)
			if err == nil {
				db.Exec("UPDATE quota_purchases SET payment_status = 'paid' WHERE id = ?", purchase.ID)
				
				var plan struct {
					QuotaType string `db:"quota_type"`
				}
				db.Get(&plan, "SELECT quota_type FROM subscription_plans WHERE id = ?", purchase.PlanID)
				
				if plan.QuotaType == "standard" {
					db.Exec("UPDATE organizers SET quota_standard = quota_standard + ? WHERE uuid = ?", purchase.Quantity, purchase.OrganizerID)
				} else if plan.QuotaType == "elite" {
					db.Exec("UPDATE organizers SET quota_elite = quota_elite + ? WHERE uuid = ?", purchase.Quantity, purchase.OrganizerID)
				}

				var archerEmail string
				db.Get(&archerEmail, "SELECT email FROM users WHERE uuid = (SELECT user_id FROM organizers WHERE uuid = ?)", purchase.OrganizerID)
				
				if archerEmail != "" {
					utils.SendEmail(archerEmail, "Quota Berhasil Ditambahkan", fmt.Sprintf("Anda telah berhasil membeli quota event sebanyak %d", purchase.Quantity))
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// GetPublicQuotaPlans GET /public/quota/plans
func GetPublicQuotaPlans(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var plans []struct {
			ID               int     `db:"id" json:"id"`
			Name             string  `db:"name" json:"name"`
			Price            float64 `db:"price" json:"price"`
			QuotaType        string  `db:"quota_type" json:"quota_type"`
			MaxParticipants  *int    `db:"max_participants" json:"max_participants"`
			MaxCategories    *int    `db:"max_categories" json:"max_categories"`
			MaxScorekeepers  *int    `db:"max_scorekeepers" json:"max_scorekeepers"`
			MaxMediaMB       *int    `db:"max_media_mb" json:"max_media_mb"`
		}
		
		query := `SELECT id, name, price, quota_type, max_participants, max_categories, max_scorekeepers, max_media_mb 
				  FROM subscription_plans 
				  WHERE type = 'quota' AND target_type = 'organization' 
				  ORDER BY id`
				  
		if err := db.Select(&plans, query); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get plans"})
			return
		}
		
		if plans == nil {
			plans = []struct {
				ID               int     `db:"id" json:"id"`
				Name             string  `db:"name" json:"name"`
				Price            float64 `db:"price" json:"price"`
				QuotaType        string  `db:"quota_type" json:"quota_type"`
				MaxParticipants  *int    `db:"max_participants" json:"max_participants"`
				MaxCategories    *int    `db:"max_categories" json:"max_categories"`
				MaxScorekeepers  *int    `db:"max_scorekeepers" json:"max_scorekeepers"`
				MaxMediaMB       *int    `db:"max_media_mb" json:"max_media_mb"`
			}{}
		}

		c.JSON(http.StatusOK, plans)
	}
}
