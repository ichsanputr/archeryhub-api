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
			QuotaFree     int `json:"quota_free" db:"quota_free"`
			QuotaStandard int `json:"quota_standard" db:"quota_standard"`
			QuotaElite    int `json:"quota_elite" db:"quota_elite"`
			TotalQuota    int `json:"total_quota"`
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

		if userType == "organizer" {
			err = db.Get(&subscription, `
				SELECT o.subscription_plan_id, COALESCE(o.subscription_status, 'active') as subscription_status,
				       p.name as plan_name, p.price as plan_price, p.type as billing_type,
				       DATE_FORMAT(o.subscription_expires_at, '%d %b %Y') as next_billing_date,
				       COALESCE(o.quota_free, 20) as quota_free, o.quota_standard, o.quota_elite
				FROM organizers o
				LEFT JOIN subscription_plans p ON o.subscription_plan_id = p.id
				WHERE o.user_id = ?`, userID)

			if err == nil {
				subscription.TotalQuota = subscription.QuotaFree + subscription.QuotaStandard + subscription.QuotaElite
				db.Get(&subscription.Usage.Current, "SELECT COUNT(*) FROM tournament_participants WHERE tournament_id IN (SELECT uuid FROM tournaments WHERE organization_id = (SELECT uuid FROM organizers WHERE user_id = ?))", userID)
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
		
		targetType := "organizer"

		db.Select(&plans, "SELECT id, name, price, type, features FROM subscription_plans WHERE target_type = ?", targetType)
		
		// Auto-expire check is handled asynchronously or via background jobs

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
					WHEN tournament_id IS NOT NULL THEN 'Pembayaran Layanan Event'
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
					WHEN tournament_id IS NOT NULL THEN 'Pembayaran Layanan Event'
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
	return func(c *gin.Context) {
		pricing := buildUnifiedPricing()
		c.JSON(http.StatusOK, pricing.ComparisonMatrix)
	}
}

type UnifiedPlanItem struct {
	ID                int      `json:"id"`
	TierKey           string   `json:"tier_key"`
	Name              string   `json:"name"`
	Badge             string   `json:"badge"`
	Description       string   `json:"description"`
	Period            string   `json:"period"`
	PriceIDR          float64  `json:"price_idr"`
	PromoPriceIDR     float64  `json:"promo_price_idr"`
	PriceUSD          float64  `json:"price_usd"`
	PromoPriceUSD     float64  `json:"promo_price_usd"`
	DiscountPct       int      `json:"discount_pct"`
	IsPopular         bool     `json:"is_popular"`
	MaxParticipants   *int     `json:"max_participants"`
	MaxCategories     *int     `json:"max_categories"`
	MaxScorekeepers   *int     `json:"max_scorekeepers"`
	HighlightFeatures []string `json:"highlight_features"`
	CTAText           string   `json:"cta_text"`
	CTALink           string   `json:"cta_link"`
}

type UnifiedComparisonRow struct {
	Category string      `json:"category"`
	Feature  string      `json:"feature"`
	Free     interface{} `json:"free"`
	Standard interface{} `json:"standard"`
	Elite    interface{} `json:"elite"`
}

type UnifiedBundleRule struct {
	MinQty      int    `json:"min_qty"`
	DiscountPct int    `json:"discount_pct"`
	Label       string `json:"label"`
}

type UnifiedFAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type UnifiedPricingResponse struct {
	Plans            []UnifiedPlanItem      `json:"plans"`
	ComparisonMatrix []UnifiedComparisonRow `json:"comparison_matrix"`
	BundleDiscounts  []UnifiedBundleRule    `json:"bundle_discounts"`
	FAQs             []UnifiedFAQItem       `json:"faqs"`
}

func buildUnifiedPricing() UnifiedPricingResponse {
	maxPartFree := 50
	maxPartStd := 200

	plans := []UnifiedPlanItem{
		{
			ID:                0,
			TierKey:           "free",
			Name:              "Free Starter",
			Badge:             "Starter",
			Description:       "Try core tournament scoring features with 1 free tournament quota.",
			Period:            "/tournament",
			PriceIDR:          0,
			PromoPriceIDR:     0,
			PriceUSD:          0,
			PromoPriceUSD:     0,
			DiscountPct:       0,
			IsPopular:         false,
			MaxParticipants:   &maxPartFree,
			MaxCategories:     nil,
			MaxScorekeepers:   nil,
			HighlightFeatures: []string{
				"Up to 50 Participants / Tournament",
				"200 MB Media Storage",
				"Access All Features",
			},
			CTAText: "Get Started Free",
			CTALink: "/auth/register",
		},
		{
			ID:                7,
			TierKey:           "standard",
			Name:              "Standard EO",
			Badge:             "Most Popular",
			Description:       "Complete tournament scoring solution for clubs, regional circuits, and open tournaments.",
			Period:            "/tournament",
			PriceIDR:          49999,
			PromoPriceIDR:     24999,
			PriceUSD:          3.00,
			PromoPriceUSD:     1.50,
			DiscountPct:       50,
			IsPopular:         true,
			MaxParticipants:   &maxPartStd,
			MaxCategories:     nil,
			MaxScorekeepers:   nil,
			HighlightFeatures: []string{
				"Up to 200 Participants / Tournament",
				"3 GB Media Storage",
				"Access All Features",
			},
			CTAText: "Claim Standard Promo",
			CTALink: "/package",
		},
		{
			ID:                8,
			TierKey:           "elite",
			Name:              "Elite EO",
			Badge:             "Professional Tier",
			Description:       "Unlimited capabilities for professional championships, multi-field live streaming, and national tournaments.",
			Period:            "/tournament",
			PriceIDR:          79999,
			PromoPriceIDR:     39999,
			PriceUSD:          7.00,
			PromoPriceUSD:     3.50,
			DiscountPct:       50,
			IsPopular:         false,
			MaxParticipants:   nil,
			MaxCategories:     nil,
			MaxScorekeepers:   nil,
			HighlightFeatures: []string{
				"Unlimited Participants",
				"10 GB Media Storage",
				"Access All Features",
			},
			CTAText: "Choose Elite",
			CTALink: "/package",
		},
	}

	comparison := []UnifiedComparisonRow{
		{
			Category: "Capacity & Limits",
			Feature:  "Participant limit per tournament",
			Free:     "50 Archers",
			Standard: "200 Archers",
			Elite:    "Unlimited",
		},
		{
			Category: "Capacity & Limits",
			Feature:  "Media storage",
			Free:     "200 MB",
			Standard: "3 GB",
			Elite:    "10 GB",
		},
		{
			Category: "Capacity & Limits",
			Feature:  "Competition categories",
			Free:     "Unlimited",
			Standard: "Unlimited",
			Elite:    "Unlimited",
		},
		{
			Category: "Capacity & Limits",
			Feature:  "Scorekeeper accounts",
			Free:     "Unlimited",
			Standard: "Unlimited",
			Elite:    "Unlimited",
		},
		{
			Category: "Capacity & Limits",
			Feature:  "Access period",
			Free:     "Lifetime",
			Standard: "Lifetime",
			Elite:    "Lifetime",
		},
		{
			Category: "Tournament Setup",
			Feature:  "Tournament setup (Categories, fees, lane setup & schedule)",
			Free:     true,
			Standard: true,
			Elite:    true,
		},
		{
			Category: "Registration",
			Feature:  "Participant registration (Online, manual entry & payment modes)",
			Free:     true,
			Standard: true,
			Elite:    true,
		},
		{
			Category: "Scoring & Match Play",
			Feature:  "Scoring & match play (Live qualification, brackets & scorekeeper)",
			Free:     true,
			Standard: true,
			Elite:    true,
		},
		{
			Category: "Printouts & Certificates",
			Feature:  "Printouts & certificates (8 official PDF formats with QR code)",
			Free:     true,
			Standard: true,
			Elite:    true,
		},
		{
			Category: "Data Export",
			Feature:  "Data export (Participants, payments & stats in Excel & CSV)",
			Free:     true,
			Standard: true,
			Elite:    true,
		},
		{
			Category: "Support",
			Feature:  "Customer support",
			Free:     "24/7 Support",
			Standard: "24/7 Support",
			Elite:    "24/7 Support",
		},
	}

	bundleRules := []UnifiedBundleRule{
		{MinQty: 1, DiscountPct: 0, Label: "Single Package"},
		{MinQty: 3, DiscountPct: 7, Label: "Save 7%"},
		{MinQty: 5, DiscountPct: 12, Label: "Save 12%"},
		{MinQty: 10, DiscountPct: 20, Label: "Save 20%"},
	}

	faqs := []UnifiedFAQItem{
		{
			Question: "How does the tournament quota system work?",
			Answer:   "Organizers can purchase tournament quotas individually or in discounted bundles. Each quota allows you to publish and manage one complete tournament with all platform features.",
		},
		{
			Question: "What is the difference between Standard EO and Elite EO?",
			Answer:   "Standard EO supports up to 200 participants and 3 GB media storage per tournament. Elite EO offers unlimited participants and 10 GB media storage. All scoring, certificates, live leaderboards, and management features are fully unlocked on both tiers.",
		},
		{
			Question: "Do purchased tournament quotas have an expiration date?",
			Answer:   "No. All tournament quotas stored in your organizer account never expire. You can keep them and use them whenever your tournament schedule is set.",
		},
		{
			Question: "When is tournament quota deducted from my account balance?",
			Answer:   "Tournament quota is only deducted when you publish a tournament (changing status from Draft to Public). While setting up categories, brackets, and rules in Draft mode, no quota is consumed.",
		},
		{
			Question: "Can I purchase custom numbers of tournament slots?",
			Answer:   "Yes! In your organizer dashboard, you can enter any custom number of tournament slots you require to plan out your organization's yearly calendar with automatic volume bundle discounts.",
		},
	}

	return UnifiedPricingResponse{
		Plans:            plans,
		ComparisonMatrix: comparison,
		BundleDiscounts:  bundleRules,
		FAQs:             faqs,
	}
}

// GetUnifiedPricingPlans returns the unified single source of truth for pricing across the entire platform
func GetUnifiedPricingPlans(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		pricing := buildUnifiedPricing()
		c.JSON(http.StatusOK, pricing)
	}
}

