package handler

import (
	"Archeris-api/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// RootLoginRequest is a separate struct to avoid the strict email-format validation of LoginRequest.
type RootLoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RootLogin handles login specifically for the root user
func RootLogin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req RootLoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email dan password wajib diisi"})
			return
		}

		var root struct {
			UUID     string `db:"uuid"`
			Email    string `db:"email"`
			Password string `db:"password"`
			Name     string `db:"name"`
		}

		err := db.Get(&root, "SELECT uuid, email, password, name FROM roots WHERE email = ?", req.Email)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kredensial root tidak valid"})
			return
		}

		if root.Password != req.Password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Kredensial root tidak valid"})
			return
		}

		// Generate JWT token
		token, err := generateJWT(root.UUID, root.Email, "root", "root", root.Name, "", "")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat sesi"})
			return
		}

		setAuthCookie(c, token, 60*60*24) // 24 hours

		c.JSON(http.StatusOK, AuthResponse{
			Token: token,
			User: gin.H{
				"id":        root.UUID,
				"email":     root.Email,
				"full_name": root.Name,
				"role":      "root",
				"user_type": "root",
			},
		})
	}
}

// GetAllUsers lists all users from all tables
func GetAllUsers(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type SimpleUser struct {
			UUID      string `json:"uuid" db:"uuid"`
			Email     string `json:"email" db:"email"`
			Name      string `json:"name" db:"name"`
			Type      string `json:"type" db:"type"`
			Status    string `json:"status" db:"status"`
			AvatarURL string `json:"avatar_url" db:"avatar_url"`
			CreatedAt string `json:"created_at" db:"created_at"`
		}

		var users []SimpleUser

		// COALESCE on every string column to avoid NULL scan errors
		query := `
			SELECT
				COALESCE(uuid,'')      as uuid,
				COALESCE(email,'')     as email,
				COALESCE(full_name,'') as name,
				'archer'               as type,
				COALESCE(status,'')   as status,
				COALESCE(avatar_url,'') as avatar_url,
				COALESCE(created_at,'1970-01-01') as created_at
			FROM archers
			UNION ALL
			SELECT
				COALESCE(uuid,''),
				COALESCE(email,''),
				COALESCE(name,''),
				'club',
				COALESCE(status,''),
				COALESCE(logo_url, avatar_url, ''),
				COALESCE(created_at,'1970-01-01')
			FROM clubs
			UNION ALL
			SELECT
				COALESCE(uuid,''),
				COALESCE(email,''),
				COALESCE(name,''),
				'organization',
				COALESCE(status,''),
				COALESCE(avatar_url,''),
				COALESCE(created_at,'1970-01-01')
			FROM organizations
			UNION ALL
			SELECT
				COALESCE(uuid,''),
				COALESCE(email,''),
				COALESCE(store_name,''),
				'seller',
				COALESCE(status,''),
				COALESCE(avatar_url,''),
				COALESCE(created_at,'1970-01-01')
			FROM sellers
			ORDER BY created_at DESC
		`

		err := db.Select(&users, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data pengguna: " + err.Error()})
			return
		}

		if users == nil {
			users = []SimpleUser{}
		}
		c.JSON(http.StatusOK, users)
	}
}

// GetAllSubscriptions returns all club and organization subscriptions for root management
func GetAllSubscriptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		typeFilter := c.Query("type") // "club", "organization", ""
		statusFilter := c.Query("status") // "active", "trial", "expired", ""

		type SubRow struct {
			UUID               string  `json:"uuid" db:"uuid"`
			Name               string  `json:"name" db:"name"`
			Email              string  `json:"email" db:"email"`
			UserType           string  `json:"user_type" db:"user_type"`
			SubscriptionStatus string  `json:"subscription_status" db:"subscription_status"`
			PlanID             *int    `json:"plan_id" db:"plan_id"`
			PlanName           *string `json:"plan_name" db:"plan_name"`
			ExpiresAt          *string `json:"expires_at" db:"expires_at"`
			AvatarURL          string  `json:"avatar_url" db:"avatar_url"`
			CreatedAt          string  `json:"created_at" db:"created_at"`
		}

		var rows []SubRow

		// COALESCE all string columns to avoid NULL scan panics
		clubQuery := `
			SELECT
				COALESCE(c.uuid,'')   as uuid,
				COALESCE(c.name,'')   as name,
				COALESCE(c.email,'')  as email,
				'club'                 as user_type,
				COALESCE(c.subscription_status,'trial') as subscription_status,
				c.subscription_plan_id as plan_id,
				sp.name                as plan_name,
				DATE_FORMAT(c.subscription_expires_at,'%Y-%m-%d %H:%i:%s') as expires_at,
				COALESCE(c.logo_url, c.avatar_url, '') as avatar_url,
				COALESCE(c.created_at,'1970-01-01')                        as created_at
			FROM clubs c
			LEFT JOIN subscription_plans sp ON sp.id = c.subscription_plan_id
		`
		orgQuery := `
			SELECT
				COALESCE(o.uuid,'')   as uuid,
				COALESCE(o.name,'')   as name,
				COALESCE(o.email,'')  as email,
				'organization'         as user_type,
				COALESCE(o.subscription_status,'trial') as subscription_status,
				o.subscription_plan_id as plan_id,
				sp.name                as plan_name,
				DATE_FORMAT(o.subscription_expires_at,'%Y-%m-%d %H:%i:%s') as expires_at,
				COALESCE(o.avatar_url,'') as avatar_url,
				COALESCE(o.created_at,'1970-01-01')                        as created_at
			FROM organizations o
			LEFT JOIN subscription_plans sp ON sp.id = o.subscription_plan_id
		`

		var clubRows, orgRows []SubRow

		if typeFilter == "" || typeFilter == "club" {
			if err := db.Select(&clubRows, clubQuery); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Club query error: " + err.Error()})
				return
			}
			rows = append(rows, clubRows...)
		}
		if typeFilter == "" || typeFilter == "organization" {
			if err := db.Select(&orgRows, orgQuery); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Org query error: " + err.Error()})
				return
			}
			rows = append(rows, orgRows...)
		}

		// Filter by status if requested
		if statusFilter != "" {
			filtered := []SubRow{}
			for _, r := range rows {
				if r.SubscriptionStatus == statusFilter {
					filtered = append(filtered, r)
				}
			}
			rows = filtered
		}

		if rows == nil {
			rows = []SubRow{}
		}

		// Sort all rows combined by created_at DESC
		// (Optional, since we are doing UNION manually we might need to sort in Go or use a better query)
		// For now let's just return them.

		c.JSON(http.StatusOK, gin.H{"subscriptions": rows, "total": len(rows)})
	}
}

// UpdateUserSubscription allows root to update a club or org subscription
func UpdateUserSubscription(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID := c.Param("uuid")
		userType := c.Param("type") // "club" or "organization"

		var req struct {
			Status    string  `json:"status"`     // active, trial, expired, canceled
			PlanID    *int    `json:"plan_id"`
			ExpiresAt *string `json:"expires_at"` // "YYYY-MM-DD"
			ExtendDays *int   `json:"extend_days"` // extend current expiry by N days
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Request tidak valid"})
			return
		}

		table := ""
		switch userType {
		case "club":
			table = "clubs"
		case "organization":
			table = "organizations"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe user harus 'club' atau 'organization'"})
			return
		}

		setParts := []string{}
		args := []interface{}{}

		if req.Status != "" {
			setParts = append(setParts, "subscription_status = ?")
			args = append(args, req.Status)
		}
		if req.PlanID != nil {
			setParts = append(setParts, "subscription_plan_id = ?")
			args = append(args, *req.PlanID)
		}
		if req.ExpiresAt != nil {
			setParts = append(setParts, "subscription_expires_at = ?")
			args = append(args, *req.ExpiresAt)
		}
		if req.ExtendDays != nil {
			setParts = append(setParts, fmt.Sprintf("subscription_expires_at = DATE_ADD(GREATEST(COALESCE(subscription_expires_at, NOW()), NOW()), INTERVAL %d DAY)", *req.ExtendDays))
			// If extending and current status is expired, set to active
			if req.Status == "" {
				setParts = append(setParts, "subscription_status = 'active'")
			}
		}

		if len(setParts) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada perubahan yang diminta"})
			return
		}

		setParts = append(setParts, "updated_at = NOW()")
		args = append(args, userUUID)

		query := "UPDATE " + table + " SET "
		for i, part := range setParts {
			if i > 0 {
				query += ", "
			}
			query += part
		}
		query += " WHERE uuid = ?"

		result, err := db.Exec(query, args...)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui subscription: " + err.Error()})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Subscription berhasil diperbarui",
			"uuid":    userUUID,
			"type":    userType,
		})
	}
}

// AddSubscriptionAddon allows root to add extra days or upgrade plan
func AddSubscriptionAddon(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID := c.Param("uuid")
		userType := c.Param("type")

		var req struct {
			AddonDays int    `json:"addon_days"` // Add N days to current subscription
			Note      string `json:"note"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.AddonDays <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "addon_days harus lebih dari 0"})
			return
		}

		table := ""
		switch userType {
		case "club":
			table = "clubs"
		case "organization":
			table = "organizations"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe user tidak valid"})
			return
		}

		// Extend subscription while also activating if expired
		query := fmt.Sprintf(`
			UPDATE %s SET
				subscription_expires_at = DATE_ADD(GREATEST(COALESCE(subscription_expires_at, NOW()), NOW()), INTERVAL ? DAY),
				subscription_status = CASE WHEN subscription_status = 'expired' THEN 'active' ELSE subscription_status END,
				updated_at = NOW()
			WHERE uuid = ?
		`, table)

		result, err := db.Exec(query, req.AddonDays, userUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan addon: " + err.Error()})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
			return
		}

		// Record in subscription invoices if table exists
		db.Exec(`INSERT INTO subscription_invoices (user_uuid, user_type, action, days_added, note, created_at)
			VALUES (?, ?, 'addon', ?, ?, NOW())
			ON DUPLICATE KEY UPDATE created_at = NOW()`,
			userUUID, userType, req.AddonDays, req.Note)

		c.JSON(http.StatusOK, gin.H{
			"message":    fmt.Sprintf("Berhasil menambahkan %d hari ke subscription", req.AddonDays),
			"addon_days": req.AddonDays,
			"note":       req.Note,
			"applied_at": time.Now().Format("2006-01-02 15:04:05"),
		})
	}
}

// TerminateUser suspends (or re-activates) a user account by root
func TerminateUser(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userUUID := c.Param("uuid")
		userType := c.Param("type") // "archer", "club", "organization", "seller"

		var req struct {
			Action string `json:"action" binding:"required"` // "suspend" or "activate"
			Reason string `json:"reason"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "action wajib diisi (suspend / activate)"})
			return
		}

		table := ""
		switch userType {
		case "archer":
			table = "archers"
		case "club":
			table = "clubs"
		case "organization":
			table = "organizations"
		case "seller":
			table = "sellers"
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipe user tidak valid"})
			return
		}

		newStatus := "suspended"
		if req.Action == "activate" {
			newStatus = "active"
		}

		result, err := db.Exec("UPDATE "+table+" SET status = ?, updated_at = NOW() WHERE uuid = ?", newStatus, userUUID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui status: " + err.Error()})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": fmt.Sprintf("Akun berhasil di-%s", req.Action),
			"uuid":    userUUID,
			"status":  newStatus,
		})
	}
}

// RootCreateAccount creates a club, organization, or seller account by root
func RootCreateAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			UserType    string `json:"user_type" binding:"required"` // "club", "organization", "seller"
			Name        string `json:"name" binding:"required"`
			Email       string `json:"email" binding:"required"`
			Password    string `json:"password" binding:"required"`
			Phone       string `json:"phone"`
			City        string `json:"city"`
			Address     string `json:"address"`
			WhatsAppNo  string `json:"whatsapp_no"`
			Acronym     string `json:"acronym"`
			TrialDays   int    `json:"trial_days"` // 0 = use default 90 days
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		trialDays := req.TrialDays
		if trialDays <= 0 {
			trialDays = 90
		}

		// Use proper UUID standard to fix "Data too long" error (varchar 36)
		newUUID := uuid.New().String()

		// Use utils helper for consistent slug generation
		slug := utils.CleanUsername(req.Name)
		if slug == "" {
			slug = req.UserType + "-" + newUUID[:8]
		}

		var err error
		switch req.UserType {
		case "club":
			_, err = db.Exec(`
				INSERT INTO clubs (uuid, user_id, slug, email, password, name, status, subscription_plan_id, subscription_status, subscription_expires_at)
				VALUES (?, ?, ?, ?, ?, ?, 'active', 3, 'trial', DATE_ADD(NOW(), INTERVAL ? DAY))
			`, newUUID, newUUID, slug, req.Email, req.Password, req.Name, trialDays)
		case "organization":
			whatsApp := req.WhatsAppNo
			if whatsApp == "" {
				whatsApp = req.Phone
			}
			_, err = db.Exec(`
				INSERT INTO organizations (uuid, user_id, slug, email, password, name, acronym, whatsapp_no, city, address, status, subscription_plan_id, subscription_status, subscription_expires_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', 5, 'trial', DATE_ADD(NOW(), INTERVAL ? DAY))
			`, newUUID, newUUID, slug, req.Email, req.Password, req.Name, req.Acronym, whatsApp, req.City, req.Address, trialDays)
		case "seller":
			_, err = db.Exec(`
				INSERT INTO sellers (uuid, user_id, slug, email, password, store_name, status)
				VALUES (?, ?, ?, ?, ?, ?, 'active')
			`, newUUID, newUUID, slug, req.Email, req.Password, req.Name)
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_type harus club, organization, atau seller"})
			return
		}

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat akun: " + err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":   "Akun berhasil dibuat",
			"uuid":      newUUID,
			"user_type": req.UserType,
			"email":     req.Email,
			"name":      req.Name,
		})
	}
}

// GetSubscriptionPlans returns all available subscription plans
func GetSubscriptionPlans(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		type Plan struct {
			ID          int     `json:"id" db:"id"`
			Name        string  `json:"name" db:"name"`
			Price       float64 `json:"price" db:"price"`
			BillingType string  `json:"billing_type" db:"type"`
			UserType    *string `json:"user_type" db:"target_type"`
			Features    *string `json:"features" db:"features"`
			IsActive    bool    `json:"is_active"`
		}

		var plans []Plan
		// Simplified Grouping: exactly 4 unique plans (Standard/Elite for Club/Org)
		query := `
			SELECT MAX(id) as id, name, MAX(price) as price, MAX(COALESCE(type, 'monthly')) as type, target_type, MAX(features) as features 
			FROM subscription_plans 
			WHERE target_type IN ('club', 'organization')
			GROUP BY name, target_type
			ORDER BY target_type DESC, price ASC
		`
		err := db.Select(&plans, query)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data paket: " + err.Error()})
			return
		}

		if plans == nil {
			plans = []Plan{}
		} else {
			for i := range plans {
				plans[i].IsActive = true
			}
		}
		c.JSON(http.StatusOK, gin.H{"plans": plans, "total": len(plans)})
	}
}

