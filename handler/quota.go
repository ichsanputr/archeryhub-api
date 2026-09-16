package handler

import (
	"context"
	"Archeris-api/utils"
	"fmt"
	"net/http"
	"os"
	"strconv"
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
			QuotaFree     int    `db:"quota_free"`
			QuotaStandard int    `db:"quota_standard"`
			QuotaElite    int    `db:"quota_elite"`
		}
		if err := db.Get(&org, "SELECT uuid, COALESCE(quota_free, 20) as quota_free, quota_standard, quota_elite FROM organizers WHERE uuid = ? OR email = ?", userID, userID); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"quota_free":             0,
				"quota_standard":         0,
				"quota_elite":            0,
				"total_quota":            0,
				"published_events_count": 0,
			})
			return
		}

		var publishedCount int
		db.Get(&publishedCount, "SELECT COUNT(*) FROM tournaments WHERE organizer_id = ? AND status = 'published' AND quota_type IS NOT NULL", org.UUID)

		c.JSON(http.StatusOK, gin.H{
			"quota_free":             org.QuotaFree,
			"quota_standard":         org.QuotaStandard,
			"quota_elite":            org.QuotaElite,
			"total_quota":            org.QuotaFree + org.QuotaStandard + org.QuotaElite,
			"published_events_count": publishedCount,
		})
	}
}

// GetQuotaHistory GET /organizers/me/quota/history?limit=10&page=1
func GetQuotaHistory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			return
		}

		var orgUUID string
		if err := db.Get(&orgUUID, "SELECT uuid FROM organizers WHERE uuid = ? OR email = ?", userID, userID); err != nil {
			c.JSON(http.StatusOK, gin.H{
				"data":        []interface{}{},
				"total":       0,
				"page":        1,
				"limit":       10,
				"total_pages": 0,
			})
			return
		}

		limitStr := c.DefaultQuery("limit", "10")
		pageStr := c.DefaultQuery("page", "1")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			limit = 10
		}
		if limit > 100 {
			limit = 100
		}

		page, err := strconv.Atoi(pageStr)
		if err != nil || page <= 0 {
			page = 1
		}
		offset := (page - 1) * limit

		type QuotaHistoryItem struct {
			UUID             string    `db:"uuid" json:"id"`
			Quantity         int       `db:"quantity" json:"quantity"`
			Amount           float64   `db:"total_amount" json:"amount"`
			PaymentStatus    string    `db:"payment_status" json:"payment_status"`
			PaymentMethod    string    `db:"payment_method" json:"payment_method"`
			PaymentReference string    `db:"payment_reference" json:"payment_reference"`
			CheckoutURL      *string   `db:"checkout_url" json:"checkout_url"`
			PurchasedAt      time.Time `db:"purchased_at" json:"purchased_at"`
			PlanName         string    `db:"plan_name" json:"plan_name"`
		}

		var total int
		_ = db.Get(&total, "SELECT COUNT(*) FROM quota_purchases WHERE organizer_id = ?", orgUUID)

		var history []QuotaHistoryItem
		query := `SELECT q.uuid, q.quantity, q.total_amount, q.payment_status, COALESCE(q.payment_method, 'Mayar') as payment_method, COALESCE(q.payment_reference, '') as payment_reference, q.checkout_url, q.purchased_at, COALESCE(p.name, 'Paket Kuota Event') as plan_name 
				  FROM quota_purchases q 
				  LEFT JOIN subscription_plans p ON q.plan_id = p.id 
				  WHERE q.organizer_id = ? 
				  ORDER BY q.purchased_at DESC LIMIT ? OFFSET ?`
		
		if err := db.Select(&history, query, orgUUID, limit, offset); err != nil {
			history = []QuotaHistoryItem{}
		}

		if history == nil {
			history = []QuotaHistoryItem{}
		}

		totalPages := (total + limit - 1) / limit
		if totalPages == 0 && total > 0 {
			totalPages = 1
		}

		c.JSON(http.StatusOK, gin.H{
			"data":        history,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		})
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
			Channel       string `json:"channel"`
			Currency      string `json:"currency"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
			return
		}

		if req.Quantity < 1 {
			req.Quantity = 1
		}
		if req.Currency == "" {
			req.Currency = "IDR"
		}

		var org struct {
			UUID       string `db:"uuid"`
			Name       string `db:"name"`
			Email      string `db:"email"`
			WhatsappNo string `db:"whatsapp_no"`
		}
		orgQuery := "SELECT uuid, name, email, COALESCE(whatsapp_no, '') as whatsapp_no FROM organizers WHERE uuid = ? OR email = ?"
		if err := db.Get(&org, orgQuery, userID, userID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get organizer: " + err.Error()})
			return
		}

		var plan struct {
			Price     float64 `db:"price"`
			Type      string  `db:"type"`
			Name      string  `db:"name"`
			QuotaType string  `db:"quota_type"`
		}
		if err := db.Get(&plan, "SELECT price, type, name, COALESCE(quota_type, 'standard') as quota_type FROM subscription_plans WHERE id = ?", req.PlanID); err != nil || plan.Type != "quota" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plan"})
			return
		}

		discountPct := 0.0
		switch {
		case req.Quantity >= 10:
			discountPct = 0.20
		case req.Quantity >= 5:
			discountPct = 0.12
		case req.Quantity >= 3:
			discountPct = 0.07
		}

		promoPrice := plan.Price * 0.50
		totalAmount := (promoPrice * float64(req.Quantity)) * (1.0 - discountPct)
		refID := fmt.Sprintf("QUOTA-%s-%d", strings.ToUpper(uuid.New().String()[:8]), time.Now().Unix())
		purchaseUUID := uuid.New().String()

		appURL := os.Getenv("APP_URL")
		if appURL == "" {
			appURL = "http://localhost:3003"
		}

		var checkoutUrl string
		var externalTxID string
		paymentMethod := "mayar"
		if req.PaymentMethod == "paypal" {
			paymentMethod = "paypal"
		}

		if paymentMethod == "paypal" {
			paypalClient := utils.NewPayPalClient()
			usdAmount := paypalClient.ConvertIDRToUSD(totalAmount)
			returnURL := fmt.Sprintf("%s/dashboard/organizer/package/detail?trx_id=%s&provider=paypal", strings.TrimSuffix(appURL, "/"), refID)
			cancelURL := fmt.Sprintf("%s/dashboard/organizer/package/detail?trx_id=%s&cancelled=true", strings.TrimSuffix(appURL, "/"), refID)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			orderResp, approveURL, err := paypalClient.CreateOrder(ctx, refID, fmt.Sprintf("Pembelian %d Paket %s ArcheryHub", req.Quantity, plan.Name), usdAmount, returnURL, cancelURL)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi pembayaran PayPal: " + err.Error()})
				return
			}
			checkoutUrl = approveURL
			externalTxID = orderResp.ID
		} else {
			mayarClient := utils.NewMayarClient()
			redirectURL := fmt.Sprintf("%s/dashboard/organizer/package/detail?trx_id=%s", strings.TrimSuffix(appURL, "/"), refID)

			paymentReq := utils.MayarPaymentReq{
				Name:        fmt.Sprintf("Quota %s x%d - %s", plan.Name, req.Quantity, org.Name),
				Amount:      int(totalAmount),
				Email:       org.Email,
				Mobile:      org.WhatsappNo,
				Description: fmt.Sprintf("Pembelian %d Paket %s ArcheryHub", req.Quantity, plan.Name),
				RedirectURL: redirectURL,
			}

			mayarData, err := mayarClient.CreatePaymentRequest(paymentReq)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat transaksi pembayaran Mayar: " + err.Error()})
				return
			}

			checkoutUrl = mayarData.Link
			externalTxID = mayarData.TransactionID
		}

		insertSQL := `INSERT INTO quota_purchases (uuid, organizer_id, plan_id, quota_type, quantity, unit_price, total_amount, currency, payment_method, payment_reference, checkout_url, gateway_reference, payment_status, purchased_at) 
					  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', NOW())`
		_, err := db.Exec(insertSQL, purchaseUUID, org.UUID, req.PlanID, plan.QuotaType, req.Quantity, promoPrice, totalAmount, req.Currency, paymentMethod, refID, checkoutUrl, externalTxID)
		
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save purchase: " + err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"purchase_id":    refID,
			"checkout_url":   checkoutUrl,
			"transaction_id": externalTxID,
			"total_amount":   totalAmount,
			"currency":       req.Currency,
		})
	}
}

// GetPublicQuotaPlans GET /public/quota/plans
func GetPublicQuotaPlans(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var plans []struct {
			ID              int     `db:"id" json:"id"`
			Name            string  `db:"name" json:"name"`
			Price           float64 `db:"price" json:"price"`
			QuotaType       string  `db:"quota_type" json:"quota_type"`
			MaxParticipants *int    `db:"max_participants" json:"max_participants"`
			MaxCategories   *int    `db:"max_categories" json:"max_categories"`
			MaxScorekeepers *int    `db:"max_scorekeepers" json:"max_scorekeepers"`
			MaxMediaMB      *int    `db:"max_media_mb" json:"max_media_mb"`
		}
		
		query := `SELECT id, name, price, COALESCE(quota_type, 'standard') as quota_type, max_participants, max_categories, max_scorekeepers, max_media_mb 
				  FROM subscription_plans 
				  WHERE type = 'quota' AND target_type = 'organization' 
				  ORDER BY id`
				  
		if err := db.Select(&plans, query); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get plans"})
			return
		}
		
		if plans == nil {
			plans = []struct {
				ID              int     `db:"id" json:"id"`
				Name            string  `db:"name" json:"name"`
				Price           float64 `db:"price" json:"price"`
				QuotaType       string  `db:"quota_type" json:"quota_type"`
				MaxParticipants *int    `db:"max_participants" json:"max_participants"`
				MaxCategories   *int    `db:"max_categories" json:"max_categories"`
				MaxScorekeepers *int    `db:"max_scorekeepers" json:"max_scorekeepers"`
				MaxMediaMB      *int    `db:"max_media_mb" json:"max_media_mb"`
			}{}
		}

		c.JSON(http.StatusOK, plans)
	}
}
