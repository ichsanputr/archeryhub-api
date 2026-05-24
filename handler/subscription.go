package handler
import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

func GetMySubscription(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
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
			MediaUsage struct {
				Current int64 `json:"current"`
				Limit   int64 `json:"limit"`
			} `json:"media_usage"`
		}
		
		// Set default media limit (1 GB in bytes)
		subscription.MediaUsage.Limit = 1024 * 1024 * 1024

		var err error
		// Get actual media usage
		db.Get(&subscription.MediaUsage.Current, "SELECT COALESCE(SUM(size), 0) FROM media WHERE user_id = ?", userID)

		if userType == "organization" {
			err = db.Get(&subscription, `
				SELECT o.subscription_plan_id, COALESCE(o.subscription_status, 'active') as subscription_status,
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
			c.JSON(http.StatusBadRequest, gin.H{"error": "Layanan berlangganan hanya tersedia untuk Organisasi"})
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
		
		targetType := "organization"

		db.Select(&plans, "SELECT id, name, price, type, features FROM subscription_plans WHERE target_type = ?", targetType)
		
		// Auto-expire pending transactions older than 1 day
		db.Exec("UPDATE payment_transactions SET status = 'expired' WHERE status = 'pending' AND created_at < DATE_SUB(NOW(), INTERVAL 1 DAY)")

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
				CASE 
					WHEN payment_method = 'paddle' THEN CONCAT('$', FORMAT(amount, 0, 'en_US'))
					ELSE CONCAT('Rp ', FORMAT(amount, 0, 'id_ID'))
				END as amount,
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
// ExportInvoicesCSV exports transaction history to CSV
func ExportInvoicesCSV(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak diizinkan"})
			return
		}

		type Invoice struct {
			Date        string `db:"date"`
			Description string `db:"description"`
			Amount      int    `db:"amount"`
			Status      string `db:"status"`
			Method      string `db:"payment_method"`
			Reference   string `db:"reference"`
		}

		var invoices []Invoice
		err := db.Select(&invoices, `
			SELECT 
				DATE_FORMAT(created_at, '%Y-%m-%d %H:%i') as date,
				CASE 
					WHEN subscription_plan_id IS NOT NULL THEN 'Pembayaran Langganan'
					WHEN event_id IS NOT NULL THEN 'Pembayaran Layanan Event'
					ELSE 'Transaksi Lainnya'
				END as description,
				amount,
				status,
				COALESCE(payment_method, '-') as payment_method,
				reference
			FROM payment_transactions 
			WHERE user_id = ? 
			ORDER BY created_at DESC`, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data invoice", "details": err.Error()})
			return
		}

		// Set response headers
		fileName := fmt.Sprintf("billing-history-%s.csv", time.Now().Format("20060102"))
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		c.Header("Content-Type", "text/csv")

		writer := csv.NewWriter(c.Writer)
		defer writer.Flush()

		// Write header
		writer.Write([]string{"No", "Tanggal", "Deskripsi", "Jumlah", "Status", "Metode Pembayaran", "Referensi"})

		for i, v := range invoices {
			writer.Write([]string{
				strconv.Itoa(i + 1),
				v.Date,
				v.Description,
				strconv.Itoa(v.Amount),
				v.Status,
				v.Method,
				v.Reference,
			})
		}
	}
}

// GetSubscriptionComparison returns the comparison matrix for subscriptions
func GetSubscriptionComparison() gin.HandlerFunc {
	type FeatureRow struct {
		FeatureKey  string      `json:"feature_key"`
		FeatureName string      `json:"feature_name"`
		Free        interface{} `json:"free"`
		Standar     interface{} `json:"standar"`
		Elite       interface{} `json:"elite"`
	}

	comparisonData := []FeatureRow{
		{
			FeatureKey:  "events_categories",
			FeatureName: "Turnamen & Kategori",
			Free:        "1 Event",
			Standar:     "5 Event",
			Elite:       "unlimited",
		},
		{
			FeatureKey:  "online_reg",
			FeatureName: "Pendaftaran Peserta Online",
			Free:        "manual",
			Standar:     "auto_local",
			Elite:       "auto_global",
		},
		{
			FeatureKey:  "participants_limit",
			FeatureName: "Batas Peserta per Event",
			Free:        "10 / Event",
			Standar:     "50 / Event",
			Elite:       "unlimited",
		},
		{
			FeatureKey:  "referees_limit",
			FeatureName: "Wasit & Pencatat Skor",
			Free:        "referee_1",
			Standar:     "referee_5",
			Elite:       "unlimited",
		},
		{
			FeatureKey:  "scoring_methods",
			FeatureName: "Sistem Scoring Turnamen",
			Free:        "scoring_basic",
			Standar:     "scoring_elimination",
			Elite:       "scoring_full",
		},
		{
			FeatureKey:  "elimination_finals",
			FeatureName: "Babak Eliminasi Match Finals",
			Free:        false,
			Standar:     false,
			Elite:       true,
		},
		{
			FeatureKey:  "team_club_mgmt",
			FeatureName: "Manajemen Tim & Klub",
			Free:        false,
			Standar:     "standard_team",
			Elite:       "mixed_teams",
		},
		{
			FeatureKey:  "news_publishing",
			FeatureName: "Berita & Pengumuman",
			Free:        false,
			Standar:     true,
			Elite:       true,
		},
		{
			FeatureKey:  "exports_reports",
			FeatureName: "Laporan & Hasil Turnamen",
			Free:        "export_basic",
			Standar:     "export_standard",
			Elite:       "export_elite",
		},
		{
			FeatureKey:  "media_storage",
			FeatureName: "Penyimpanan Media",
			Free:        "250 MB",
			Standar:     "1 GB",
			Elite:       "5 GB",
		},
	}

	return func(c *gin.Context) {
		c.JSON(http.StatusOK, comparisonData)
	}
}
