package handler

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func GetMySubscription(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		userType, _ := c.Get("user_type")

		var subscription struct {
			PlanID      *int    `json:"plan_id" db:"subscription_plan_id"`
			Status      string  `json:"status" db:"subscription_status"`
			PlanName    *string `json:"plan_name" db:"plan_name"`
			PlanPrice   *float64 `json:"plan_price" db:"plan_price"`
			BillingType     *string `json:"billing_type" db:"billing_type"`
			NextBillingDate *string `json:"next_billing_date" db:"next_billing_date"`
			Usage       struct {
				Label string `json:"label"`
				Current int `json:"current"`
				Limit   int `json:"limit"`
			} `json:"usage"`
		}

		var err error
		if userType == "club" {
			err = db.Get(&subscription, `
				SELECT c.subscription_plan_id, COALESCE(c.subscription_status, 'trial') as subscription_status,
				       p.name as plan_name, p.price as plan_price, p.type as billing_type,
				       DATE_FORMAT(c.subscription_expires_at, '%d %b %Y') as next_billing_date
				FROM clubs c
				LEFT JOIN subscription_plans p ON c.subscription_plan_id = p.id
				WHERE c.user_id = ?`, userID)
			
			if err == nil {
				db.Get(&subscription.Usage.Current, "SELECT COUNT(*) FROM club_members WHERE club_id = (SELECT uuid FROM clubs WHERE user_id = ?) AND status = 'active'", userID)
				subscription.Usage.Label = "Anggota Aktif"
				subscription.Usage.Limit = 1000 // Default limit for display
				if subscription.PlanID != nil && *subscription.PlanID == 1 { subscription.Usage.Limit = 15 }
			}
		} else if userType == "organization" {
			err = db.Get(&subscription, `
				SELECT o.subscription_plan_id, COALESCE(o.subscription_status, 'trial') as subscription_status,
				       p.name as plan_name, p.price as plan_price, p.type as billing_type,
				       DATE_FORMAT(o.subscription_expires_at, '%d %b %Y') as next_billing_date
				FROM organizations o
				LEFT JOIN subscription_plans p ON o.subscription_plan_id = p.id
				WHERE o.user_id = ?`, userID)
			
			if err == nil {
				db.Get(&subscription.Usage.Current, "SELECT COUNT(*) FROM event_participants WHERE event_id IN (SELECT uuid FROM events WHERE organization_id = (SELECT uuid FROM organizations WHERE user_id = ?))", userID)
				subscription.Usage.Label = "Total Atlet"
				subscription.Usage.Limit = 5000
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Layanan berlangganan hanya tersedia untuk Klub dan Organisasi"})
			return
		}

		if err != nil && err != sql.ErrNoRows {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Get all available plans for this type
		var plans []struct {
			ID       int     `json:"id"`
			Name     string  `json:"name"`
			Price    float64 `json:"price"`
			Type     string  `json:"type"`
			Features string  `json:"features"`
		}
		
		targetType := "club"
		if userType == "organization" {
			targetType = "organization"
		}

		db.Select(&plans, "SELECT id, name, price, type, features FROM subscription_plans WHERE target_type = ?", targetType)

		// Get transaction history (invoices)
		var invoices []struct {
			Date          string  `json:"date" db:"date"`
			Description   string  `json:"description" db:"description"`
			Amount        string  `json:"amount" db:"amount"`
			Status        string  `json:"status" db:"status"`
			Method        string  `json:"method" db:"payment_method"`
			Reference     string  `json:"reference" db:"reference"`
			CheckoutURL   *string `json:"checkout_url" db:"checkout_url"`
			Instructions  *string `json:"instructions" db:"instructions"`
		}

		db.Select(&invoices, `
			SELECT 
				DATE_FORMAT(created_at, '%d %b %Y') as date,
				CASE 
					WHEN subscription_plan_id IS NOT NULL THEN 'Pembayaran Langganan'
					WHEN event_id IS NOT NULL THEN 'Pembayaran Layanan Event'
					ELSE 'Transaksi Lainnya'
				END as description,
				CONCAT('Rp ', FORMAT(amount, 0, 'id_ID')) as amount,
				status,
				COALESCE(payment_method, '-') as payment_method,
				reference,
				checkout_url,
				instructions
			FROM payment_transactions 
			WHERE user_id = ? 
			ORDER BY created_at DESC 
			LIMIT 10`, userID)

		c.JSON(http.StatusOK, gin.H{
			"current":  subscription,
			"plans":    plans,
			"invoices": invoices,
		})
	}
}
