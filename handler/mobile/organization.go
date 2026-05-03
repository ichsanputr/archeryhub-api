package mobile

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// @Summary Get Organization Dashboard Statistics
// @Description Get dashboard statistics, recent participants and payments for organization
// @Tags Mobile - Organization
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileOrganizationDashboardResponse
// @Router /mobile/organization/dashboard [get]
func MobileGetOrganizationDashboard(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var dashboard MobileOrganizationDashboardResponse

		// 1. Stats
		// Events
		_ = db.Get(&dashboard.Stats.TotalEvents, "SELECT COUNT(*) FROM events WHERE organizer_id = ?", userID)
		_ = db.Get(&dashboard.Stats.ActiveEvents, "SELECT COUNT(*) FROM events WHERE organizer_id = ? AND status = 'active'", userID)

		// Participants
		_ = db.Get(&dashboard.Stats.TotalParticipants, `
			SELECT COUNT(ep.uuid) 
			FROM event_participants ep
			JOIN events e ON ep.event_id = e.uuid
			WHERE e.organizer_id = ?
		`, userID)

		// Revenue
		_ = db.Get(&dashboard.Stats.TotalRevenue, `
			SELECT COALESCE(SUM(t.amount), 0)
			FROM payment_transactions t
			JOIN events e ON t.event_id = e.uuid
			WHERE e.organizer_id = ? AND t.status = 'paid' AND t.registration_id IS NOT NULL
		`, userID)

		firstOfMonth := time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02")
		_ = db.Get(&dashboard.Stats.MonthlyRevenue, `
			SELECT COALESCE(SUM(t.amount), 0)
			FROM payment_transactions t
			JOIN events e ON t.event_id = e.uuid
			WHERE e.organizer_id = ? AND t.status = 'paid' AND t.registration_id IS NOT NULL AND t.paid_at >= ?
		`, userID, firstOfMonth)

		_ = db.Get(&dashboard.Stats.PendingRevenue, `
			SELECT COALESCE(SUM(t.amount), 0)
			FROM payment_transactions t
			JOIN events e ON t.event_id = e.uuid
			WHERE e.organizer_id = ? AND t.status = 'pending' AND t.registration_id IS NOT NULL
		`, userID)

		// 2. Recent Participants
		_ = db.Select(&dashboard.RecentParticipants, `
			SELECT a.full_name, e.name as event_name, ep.created_at
			FROM event_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			JOIN events e ON ep.event_id = e.uuid
			WHERE e.organizer_id = ?
			ORDER BY ep.created_at DESC
			LIMIT 5
		`, userID)

		// 3. Recent Payments
		_ = db.Select(&dashboard.RecentPayments, `
			SELECT t.amount, a.full_name, e.name as event_name, t.paid_at, t.status
			FROM payment_transactions t
			JOIN events e ON t.event_id = e.uuid
			LEFT JOIN event_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE e.organizer_id = ? AND t.registration_id IS NOT NULL
			ORDER BY t.created_at DESC
			LIMIT 5
		`, userID)

		// 4. Upcoming Deadlines
		_ = db.Select(&dashboard.UpcomingDeadlines, `
			SELECT name, registration_deadline
			FROM events
			WHERE organizer_id = ? AND registration_deadline > NOW() AND status = 'active'
			ORDER BY registration_deadline ASC
			LIMIT 3
		`, userID)

		// Calculate days left
		for i := range dashboard.UpcomingDeadlines {
			dashboard.UpcomingDeadlines[i].DaysLeft = int(time.Until(dashboard.UpcomingDeadlines[i].Deadline).Hours() / 24)
		}

		c.JSON(http.StatusOK, dashboard)
	}
}

// MobileGetOrganizationEarnings returns detailed earnings from event registrations
// @Summary Get Organization Earnings
// @Description Get a list of all income from event registrations for the organization
// @Tags Mobile - Organization
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Success 200 {object} MobileOrganizationEarningsResponse
// @Router /mobile/organization/finance/earnings [get]
func MobileGetOrganizationEarnings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		query := `
			SELECT 
				t.uuid, e.name, t.amount, t.status, t.paid_at, t.created_at,
				a.full_name as archer_name,
				COALESCE(ec.category_name_custom, r_ag.name, '') as category_name
			FROM payment_transactions t
			JOIN events e ON t.event_id = e.uuid
			LEFT JOIN event_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN event_categories ec ON ep.category_id = ec.uuid
			LEFT JOIN ref_age_groups r_ag ON ec.category_uuid = r_ag.uuid
			WHERE e.organizer_id = ? AND t.registration_id IS NOT NULL
			ORDER BY t.created_at DESC
			LIMIT ? OFFSET ?
		`

		var earnings []MobileOrganizationEarningItem
		err := db.Select(&earnings, query, userID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penghasilan", "details": err.Error()})
			return
		}

		if earnings == nil {
			earnings = []MobileOrganizationEarningItem{}
		}

		var total int
		_ = db.Get(&total, `
			SELECT COUNT(*) 
			FROM payment_transactions t 
			JOIN events e ON t.event_id = e.uuid 
			WHERE e.organizer_id = ? AND t.registration_id IS NOT NULL
		`, userID)

		c.JSON(http.StatusOK, MobileOrganizationEarningsResponse{
			Earnings:   earnings,
			TotalCount: total,
		})
	}
}

// MobileGetOrganizationWallet returns current balance and wallet info
// @Summary Get Organization Wallet
// @Description Get the current balance and wallet details for the organization
// @Tags Mobile - Organization
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileOrganizationWalletResponse
// @Router /mobile/organization/finance/balance [get]
func MobileGetOrganizationWallet(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var wallet MobileOrganizationWalletResponse
		err := db.Get(&wallet, "SELECT uuid as wallet_id, balance FROM wallets WHERE user_id = ?", userID)
		if err != nil {
			// If wallet doesn't exist, return zero balance
			c.JSON(http.StatusOK, MobileOrganizationWalletResponse{
				Balance:    0,
				WalletUUID: "",
			})
			return
		}

		c.JSON(http.StatusOK, wallet)
	}
}

// MobileGetOrganizationBankAccounts returns list of registered bank accounts
// @Summary Get Organization Bank Accounts
// @Description Get a list of all bank accounts registered by the organization for withdrawals
// @Tags Mobile - Organization
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileOrganizationBankAccountsResponse
// @Router /mobile/organization/finance/bank-accounts [get]
func MobileGetOrganizationBankAccounts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var accounts []MobileOrganizationBankAccount
		err := db.Select(&accounts, "SELECT uuid, bank_name, account_number, account_name, is_primary, status FROM bank_accounts WHERE user_id = ? ORDER BY is_primary DESC", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data rekening bank", "details": err.Error()})
			return
		}

		if accounts == nil {
			accounts = []MobileOrganizationBankAccount{}
		}

		c.JSON(http.StatusOK, MobileOrganizationBankAccountsResponse{
			BankAccounts: accounts,
		})
	}
}

// MobileAddOrganizationBankAccount adds a new bank account for the organization
// @Summary Add Organization Bank Account
// @Description Add a new bank account for withdrawals
// @Tags Mobile - Organization
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileOrganizationBankAccountRequest true "Bank account details"
// @Success 201 {object} map[string]interface{}
// @Router /mobile/organization/finance/bank-accounts [post]
func MobileAddOrganizationBankAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		var req MobileOrganizationBankAccountRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid", "details": err.Error()})
			return
		}

		accountID := uuid.New().String()
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		if req.IsPrimary {
			_, _ = tx.Exec("UPDATE bank_accounts SET is_primary = FALSE WHERE user_id = ?", userID)
		}

		_, err = tx.Exec(`
			INSERT INTO bank_accounts (uuid, user_id, bank_name, account_number, account_name, is_primary, status)
			VALUES (?, ?, ?, ?, ?, ?, 'verified')
		`, accountID, userID, req.BankName, req.AccountNumber, req.AccountName, req.IsPrimary)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan rekening bank", "details": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Rekening bank berhasil ditambahkan", "id": accountID})
	}
}

// MobileUpdateOrganizationBankAccount updates an existing bank account
// @Summary Update Organization Bank Account
// @Description Update bank account details
// @Tags Mobile - Organization
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Bank Account UUID"
// @Param request body MobileOrganizationBankAccountRequest true "Updated details"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/organization/finance/bank-accounts/{id} [put]
func MobileUpdateOrganizationBankAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		accountID := c.Param("id")
		var req MobileOrganizationBankAccountRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format data tidak valid", "details": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		if req.IsPrimary {
			_, _ = tx.Exec("UPDATE bank_accounts SET is_primary = FALSE WHERE user_id = ?", userID)
		}

		_, err = tx.Exec(`
			UPDATE bank_accounts 
			SET bank_name = ?, account_number = ?, account_name = ?, is_primary = ?, updated_at = NOW()
			WHERE uuid = ? AND user_id = ?
		`, req.BankName, req.AccountNumber, req.AccountName, req.IsPrimary, accountID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui rekening bank", "details": err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Rekening bank berhasil diperbarui"})
	}
}

// MobileDeleteOrganizationBankAccount deletes a bank account
// @Summary Delete Organization Bank Account
// @Description Remove a bank account from the organization
// @Tags Mobile - Organization
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Bank Account UUID"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/organization/finance/bank-accounts/{id} [delete]
func MobileDeleteOrganizationBankAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		accountID := c.Param("id")

		_, err := db.Exec("DELETE FROM bank_accounts WHERE uuid = ? AND user_id = ?", accountID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus rekening bank", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Rekening bank berhasil dihapus"})
	}
}

// MobileGetBankOptions returns a list of supported banks with logos
// @Summary Get Bank Options
// @Description Get a list of supported Indonesian banks with their logo URLs
// @Tags Mobile - Options
// @Produce json
// @Success 200 {object} MobileBankOptionsResponse
// @Router /mobile/options/banks [get]
func MobileGetBankOptions(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		banks := []MobileBankOption{
			{ID: "BCA", Name: "Bank Central Asia (BCA)", LogoURL: "https://cdn.archeryhub.id/media/banks/bca.png"},
			{ID: "MANDIRI", Name: "Bank Mandiri", LogoURL: "https://cdn.archeryhub.id/media/banks/mandiri.png"},
			{ID: "BNI", Name: "Bank Negara Indonesia (BNI)", LogoURL: "https://cdn.archeryhub.id/media/banks/bni.png"},
			{ID: "BRI", Name: "Bank Rakyat Indonesia (BRI)", LogoURL: "https://cdn.archeryhub.id/media/banks/bri.png"},
			{ID: "BTN", Name: "Bank Tabungan Negara (BTN)", LogoURL: "https://cdn.archeryhub.id/media/banks/btn.png"},
			{ID: "CIMB", Name: "CIMB Niaga", LogoURL: "https://cdn.archeryhub.id/media/banks/cimb.png"},
			{ID: "DANAMON", Name: "Bank Danamon", LogoURL: "https://cdn.archeryhub.id/media/banks/danamon.png"},
			{ID: "PERMATA", Name: "Bank Permata", LogoURL: "https://cdn.archeryhub.id/media/banks/permata.png"},
			{ID: "BSI", Name: "Bank Syariah Indonesia (BSI)", LogoURL: "https://cdn.archeryhub.id/media/banks/bsi.png"},
		}

		c.JSON(http.StatusOK, MobileBankOptionsResponse{Data: banks})
	}
}
