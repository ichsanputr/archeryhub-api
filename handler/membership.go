package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// ─── Models ───────────────────────────────────────────────────────────────────

type MembershipPackage struct {
	UUID         string    `json:"uuid" db:"uuid"`
	ClubID       string    `json:"club_id" db:"club_id"`
	Name         string    `json:"name" db:"name"`
	Description  *string   `json:"description" db:"description"`
	Price        float64   `json:"price" db:"price"`
	DurationDays int       `json:"duration_days" db:"duration_days"`
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

type MemberSubscription struct {
	UUID                string     `json:"uuid" db:"uuid"`
	UserID              string     `json:"user_id" db:"user_id"`
	ClubUUID            string     `json:"club_uuid" db:"club_uuid"`
	PlanID              *int       `json:"plan_id" db:"plan_id"`
	MembershipPackageID *string    `json:"membership_package_id" db:"membership_package_id"`
	Amount              float64    `json:"amount" db:"amount"`
	PaymentMethod       *string    `json:"payment_method" db:"payment_method"`
	PaymentNote         *string    `json:"payment_note" db:"payment_note"`
	PaidAt              *time.Time `json:"paid_at" db:"paid_at"`
	CreatedBy           *string    `json:"created_by" db:"created_by"`
	Status              string     `json:"status" db:"status"`
	StartDate           *time.Time `json:"start_date" db:"start_date"`
	EndDate             *time.Time `json:"end_date" db:"end_date"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
	// Joined fields
	ArcherName  string  `json:"archer_name" db:"archer_name"`
	ArcherEmail string  `json:"archer_email" db:"archer_email"`
	PackageName *string `json:"package_name" db:"package_name"`
	AvatarURL   *string `json:"avatar_url" db:"avatar_url"`
}

type MembershipPayment struct {
	UUID             string    `json:"uuid" db:"uuid"`
	SubscriptionUUID string    `json:"subscription_uuid" db:"subscription_uuid"`
	ClubID           string    `json:"club_id" db:"club_id"`
	ArcherID         string    `json:"archer_id" db:"archer_id"`
	Amount           float64   `json:"amount" db:"amount"`
	PaymentMethod    string    `json:"payment_method" db:"payment_method"`
	PaymentNote      *string   `json:"payment_note" db:"payment_note"`
	ProofURL         *string   `json:"proof_url" db:"proof_url"`
	RecordedBy       string    `json:"recorded_by" db:"recorded_by"`
	PaidAt           time.Time `json:"paid_at" db:"paid_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	// Joined fields
	ArcherName  string  `json:"archer_name" db:"archer_name"`
	ArcherEmail string  `json:"archer_email" db:"archer_email"`
	AvatarURL   *string `json:"avatar_url" db:"avatar_url"`
	PackageName *string `json:"package_name" db:"package_name"`
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func getClubIDFromCtx(c *gin.Context) string {
	userID, _ := c.Get("user_id")
	return fmt.Sprintf("%v", userID)
}

// ─── Package Handlers ─────────────────────────────────────────────────────────

// GetMembershipPackages lists all packages for the club
func GetMembershipPackages(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)
		var packages []MembershipPackage
		err := db.Select(&packages, `
			SELECT * FROM club_membership_packages
			WHERE club_id = ?
			ORDER BY is_active DESC, price ASC
		`, clubID)
		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to list packages")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch packages"})
			return
		}
		if packages == nil {
			packages = []MembershipPackage{}
		}
		c.JSON(http.StatusOK, gin.H{"data": packages})
	}
}

// CreateMembershipPackage creates a new package for the club
func CreateMembershipPackage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)
		var req struct {
			Name         string  `json:"name" binding:"required"`
			Description  *string `json:"description"`
			Price        float64 `json:"price" binding:"required"`
			DurationDays int     `json:"duration_days" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		id := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO club_membership_packages (uuid, club_id, name, description, price, duration_days)
			VALUES (?, ?, ?, ?, ?, ?)
		`, id, clubID, req.Name, req.Description, req.Price, req.DurationDays)
		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to create package")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create package"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Package created"})
	}
}

// UpdateMembershipPackage updates a package
func UpdateMembershipPackage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)
		pkgID := c.Param("packageId")
		var req struct {
			Name         *string  `json:"name"`
			Description  *string  `json:"description"`
			Price        *float64 `json:"price"`
			DurationDays *int     `json:"duration_days"`
			IsActive     *bool    `json:"is_active"`
		}
		c.ShouldBindJSON(&req)
		_, err := db.Exec(`
			UPDATE club_membership_packages
			SET name          = COALESCE(?, name),
			    description   = COALESCE(?, description),
			    price         = COALESCE(?, price),
			    duration_days = COALESCE(?, duration_days),
			    is_active     = COALESCE(?, is_active),
			    updated_at    = NOW()
			WHERE uuid = ? AND club_id = ?
		`, req.Name, req.Description, req.Price, req.DurationDays, req.IsActive, pkgID, clubID)
		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to update package")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update package"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Package updated"})
	}
}

// DeleteMembershipPackage soft-deletes a package (marks inactive)
func DeleteMembershipPackage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)
		pkgID := c.Param("packageId")
		_, err := db.Exec(`
			UPDATE club_membership_packages SET is_active = 0, updated_at = NOW()
			WHERE uuid = ? AND club_id = ?
		`, pkgID, clubID)
		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to delete package")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete package"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Package deactivated"})
	}
}

// ─── Subscription Handlers ────────────────────────────────────────────────────

// GetMembershipSubscriptions lists all subscriptions for the club
func GetMembershipSubscriptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)

		// Auto-expire subscriptions that have passed end_date
		db.Exec(`
			UPDATE club_member_subscriptions
			SET status = 'expired'
			WHERE club_uuid = ? AND status = 'active' AND end_date < NOW()
		`, clubID)

		var subs []MemberSubscription
		err := db.Select(&subs, `
			SELECT
				cms.*,
				COALESCE(a.full_name, a.username, '') AS archer_name,
				COALESCE(a.email, '')                  AS archer_email,
				a.avatar_url,
				cmp.name                               AS package_name
			FROM club_member_subscriptions cms
			INNER JOIN (
				SELECT user_id, MAX(created_at) as latest_created
				FROM club_member_subscriptions
				WHERE club_uuid = ?
				GROUP BY user_id
			) latest ON cms.user_id = latest.user_id AND cms.created_at = latest.latest_created
			JOIN archers a ON a.uuid = cms.user_id
			LEFT JOIN club_membership_packages cmp ON cmp.uuid = cms.membership_package_id
			WHERE cms.club_uuid = ?
			ORDER BY cms.created_at DESC
		`, clubID, clubID)
		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to list subscriptions")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch subscriptions"})
			return
		}
		if subs == nil {
			subs = []MemberSubscription{}
		}
		c.JSON(http.StatusOK, gin.H{"data": subs})
	}
}

// AssignMembershipPackage assigns a membership package to an archer (admin action)
func AssignMembershipPackage(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)
		var req struct {
			ArcherID          string  `json:"archer_id" binding:"required"`
			MembershipPackageID string `json:"membership_package_id" binding:"required"`
			PaymentMethod     *string `json:"payment_method"`
			PaymentNote       *string `json:"payment_note"`
			ProofURL          *string `json:"proof_url"`
			StartDate         *string `json:"start_date"` // optional, defaults to now
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Verify archer is a club member
		var memberStatus string
		err := db.Get(&memberStatus, `
			SELECT status FROM club_members
			WHERE club_id = ? AND archer_id = ? AND status IN ('active', 'pending', 'invited')
		`, clubID, req.ArcherID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Archer is not an active member of this club"})
			return
		}

		// Get package duration
		var pkg MembershipPackage
		err = db.Get(&pkg, `
			SELECT * FROM club_membership_packages WHERE uuid = ? AND club_id = ? AND is_active = 1
		`, req.MembershipPackageID, clubID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Package not found or inactive"})
			return
		}

		// Calculate dates
		startDate := time.Now()
		if req.StartDate != nil && *req.StartDate != "" {
			if parsed, err := time.Parse("2006-01-02", *req.StartDate); err == nil {
				startDate = parsed
			}
		}
		endDate := startDate.AddDate(0, 0, pkg.DurationDays)

		// Expire any existing active subscription for this archer in this club
		db.Exec(`
			UPDATE club_member_subscriptions
			SET status = 'expired', updated_at = NOW()
			WHERE club_uuid = ? AND user_id = ? AND status = 'active'
		`, clubID, req.ArcherID)

		subID := uuid.New().String()
		isPaid := req.PaymentMethod != nil
		status := "pending"
		var paidAt interface{} = nil
		if isPaid {
			status = "active"
			now := time.Now()
			paidAt = now
		}

		_, err = db.Exec(`
			INSERT INTO club_member_subscriptions
			  (uuid, user_id, club_uuid, membership_package_id, plan_id, amount,
			   payment_method, payment_note, paid_at, created_by,
			   status, start_date, end_date)
			VALUES (?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?, ?)
		`, subID, req.ArcherID, clubID, req.MembershipPackageID, pkg.Price,
			req.PaymentMethod, req.PaymentNote, paidAt, clubID,
			status, startDate, endDate)
		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to assign package")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign package"})
			return
		}

		// Record payment if method provided
		if isPaid {
			payID := uuid.New().String()
			db.Exec(`
				INSERT INTO club_membership_payments
				  (uuid, subscription_uuid, club_id, archer_id, amount, payment_method, payment_note, proof_url, recorded_by)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, payID, subID, clubID, req.ArcherID, pkg.Price, *req.PaymentMethod, req.PaymentNote, req.ProofURL, clubID)
		}

		c.JSON(http.StatusCreated, gin.H{
			"id":       subID,
			"end_date": endDate.Format("2006-01-02"),
			"message":  "Package assigned successfully",
		})
	}
}

// RecordMembershipPayment records a manual payment for a subscription
func RecordMembershipPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)
		subID := c.Param("subscriptionId")

		var req struct {
			Amount        float64 `json:"amount" binding:"required"`
			PaymentMethod string  `json:"payment_method" binding:"required"`
			PaymentNote   *string `json:"payment_note"`
			ProofURL      *string `json:"proof_url"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Get subscription
		var sub struct {
			ArcherID string `db:"user_id"`
		}
		if err := db.Get(&sub, `SELECT user_id FROM club_member_subscriptions WHERE uuid = ? AND club_uuid = ?`, subID, clubID); err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Subscription not found"})
			return
		}

		// Mark subscription as active + record paid_at
		db.Exec(`
			UPDATE club_member_subscriptions
			SET status = 'active', paid_at = NOW(), payment_method = ?, payment_note = ?, updated_at = NOW()
			WHERE uuid = ? AND club_uuid = ?
		`, req.PaymentMethod, req.PaymentNote, subID, clubID)

		// Insert payment record
		payID := uuid.New().String()
		_, err := db.Exec(`
			INSERT INTO club_membership_payments
			  (uuid, subscription_uuid, club_id, archer_id, amount, payment_method, payment_note, proof_url, recorded_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, payID, subID, clubID, sub.ArcherID, req.Amount, req.PaymentMethod, req.PaymentNote, req.ProofURL, clubID)
		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to record payment")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record payment"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Payment recorded"})
	}
}

// GetMembershipStats returns dashboard stats for the club
func GetMembershipStats(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)

		var stats struct {
			TotalActive   int     `json:"total_active" db:"total_active"`
			TotalExpired  int     `json:"total_expired" db:"total_expired"`
			TotalPending  int     `json:"total_pending" db:"total_pending"`
			RevenueMonth  float64 `json:"revenue_month" db:"revenue_month"`
			ExpiringIn3   int     `json:"expiring_in_3_days" db:"expiring_in_3_days"`
		}

		db.QueryRowx(`
			SELECT
				COALESCE(SUM(CASE WHEN cms.status = 'active'  THEN 1 ELSE 0 END), 0) AS total_active,
				COALESCE(SUM(CASE WHEN cms.status = 'expired' THEN 1 ELSE 0 END), 0) AS total_expired,
				COALESCE(SUM(CASE WHEN cms.status = 'pending' THEN 1 ELSE 0 END), 0) AS total_pending,
				0 AS revenue_month, 0 AS expiring_in_3_days
			FROM club_member_subscriptions cms
			INNER JOIN (
				SELECT user_id, MAX(created_at) as latest_created
				FROM club_member_subscriptions
				WHERE club_uuid = ?
				GROUP BY user_id
			) latest ON cms.user_id = latest.user_id AND cms.created_at = latest.latest_created
			WHERE cms.club_uuid = ?
		`, clubID, clubID).StructScan(&stats)

		// Revenue this month
		db.QueryRowx(`
			SELECT COALESCE(SUM(amount), 0)
			FROM club_membership_payments
			WHERE club_id = ? AND MONTH(paid_at) = MONTH(NOW()) AND YEAR(paid_at) = YEAR(NOW())
		`, clubID).Scan(&stats.RevenueMonth)

		// Expiring in 3 days
		db.QueryRowx(`
			SELECT COUNT(*) FROM club_member_subscriptions cms
			INNER JOIN (
				SELECT user_id, MAX(created_at) as latest_created
				FROM club_member_subscriptions
				WHERE club_uuid = ?
				GROUP BY user_id
			) latest ON cms.user_id = latest.user_id AND cms.created_at = latest.latest_created
			WHERE cms.club_uuid = ? AND cms.status = 'active' AND cms.end_date BETWEEN NOW() AND DATE_ADD(NOW(), INTERVAL 3 DAY)
		`, clubID, clubID).Scan(&stats.ExpiringIn3)

		c.JSON(http.StatusOK, gin.H{"data": stats})
	}
}

// GetArcherSubscriptionHistory returns subscription history for a specific archer
func GetArcherSubscriptionHistory(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)
		archerID := c.Param("archerId")

		var subs []MemberSubscription
		db.Select(&subs, `
			SELECT cms.*, cmp.name AS package_name,
			       COALESCE(a.full_name, a.username, '') AS archer_name,
			       COALESCE(a.email, '') AS archer_email,
			       a.avatar_url
			FROM club_member_subscriptions cms
			JOIN archers a ON a.uuid = cms.user_id
			LEFT JOIN club_membership_packages cmp ON cmp.uuid = cms.membership_package_id
			WHERE cms.club_uuid = ? AND cms.user_id = ?
			ORDER BY cms.created_at DESC
		`, clubID, archerID)
		if subs == nil {
			subs = []MemberSubscription{}
		}

		var payments []MembershipPayment
		db.Select(&payments, `
			SELECT * FROM club_membership_payments
			WHERE club_id = ? AND archer_id = ?
			ORDER BY paid_at DESC
		`, clubID, archerID)
		if payments == nil {
			payments = []MembershipPayment{}
		}

		c.JSON(http.StatusOK, gin.H{
			"subscriptions": subs,
			"payments":      payments,
		})
	}
}

// GetMembershipPayments lists all membership payments for the club
func GetMembershipPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)

		var payments []MembershipPayment
		err := db.Select(&payments, `
			SELECT
				cmp.*,
				COALESCE(a.full_name, a.username, '') AS archer_name,
				COALESCE(a.email, '')                  AS archer_email,
				a.avatar_url,
				pk.name                                AS package_name
			FROM club_membership_payments cmp
			JOIN archers a ON a.uuid = cmp.archer_id
			LEFT JOIN club_member_subscriptions cms ON cms.uuid = cmp.subscription_uuid
			LEFT JOIN club_membership_packages pk ON pk.uuid = cms.membership_package_id
			WHERE cmp.club_id = ?
			ORDER BY cmp.paid_at DESC
		`, clubID)

		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to list payments")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
			return
		}

		if payments == nil {
			payments = []MembershipPayment{}
		}

		c.JSON(http.StatusOK, gin.H{"data": payments})
	}
}

// GetMembershipPaymentDetail returns details of a single membership payment
func GetMembershipPaymentDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		paymentID := c.Param("id")
		clubID := getClubIDFromCtx(c)

		var payment MembershipPayment
		err := db.Get(&payment, `
			SELECT
				cmp.*,
				COALESCE(a.full_name, a.username, '') AS archer_name,
				COALESCE(a.email, '')                  AS archer_email,
				a.avatar_url,
				pk.name                                AS package_name
			FROM club_membership_payments cmp
			JOIN archers a ON a.uuid = cmp.archer_id
			LEFT JOIN club_member_subscriptions cms ON cms.uuid = cmp.subscription_uuid
			LEFT JOIN club_membership_packages pk ON pk.uuid = cms.membership_package_id
			WHERE cmp.uuid = ? AND cmp.club_id = ?
		`, paymentID, clubID)

		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to fetch payment detail")
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"data": payment})
	}
}

// GetUnpaidSubscribers lists members who have pending (unpaid) subscriptions
func GetUnpaidSubscribers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		clubID := getClubIDFromCtx(c)

		var subs []MemberSubscription
		err := db.Select(&subs, `
			SELECT cms.*, COALESCE(a.full_name, a.username, '') AS archer_name,
			       COALESCE(a.email, '') AS archer_email, a.avatar_url,
				   pk.name AS package_name
			FROM club_member_subscriptions cms
			JOIN archers a ON a.uuid = cms.user_id
			LEFT JOIN club_membership_packages pk ON pk.uuid = cms.membership_package_id
			WHERE cms.club_uuid = ? AND cms.status = 'pending'
			ORDER BY cms.created_at DESC
		`, clubID)

		if err != nil {
			logrus.WithError(err).Error("[MEMBERSHIP] Failed to list unpaid subs")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch unpaid subscriptions"})
			return
		}

		if subs == nil {
			subs = []MemberSubscription{}
		}

		c.JSON(http.StatusOK, gin.H{"data": subs})
	}
}

