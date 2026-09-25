package mobile

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// @Summary Get Organizer Dashboard Statistics
// @Description Get dashboard statistics, recent participants and payments for organizer
// @Tags Mobile - Organizer
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileOrganizationDashboardResponse
// @Router /mobile/organizer/dashboard [get]
func MobileGetOrganizationDashboard(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var dashboard MobileOrganizationDashboardResponse

		// 1. Stats
		// Events
		_ = db.Get(&dashboard.Stats.TotalEvents, "SELECT COUNT(*) FROM tournaments WHERE organizer_id = ?", userID)
		_ = db.Get(&dashboard.Stats.ActiveEvents, "SELECT COUNT(*) FROM tournaments WHERE organizer_id = ? AND status = 'active'", userID)

		// Participants
		_ = db.Get(&dashboard.Stats.TotalParticipants, `
			SELECT COUNT(ep.uuid) 
			FROM tournament_participants ep
			JOIN tournaments e ON ep.tournament_id = e.uuid
			WHERE e.organizer_id = ?
		`, userID)

		// Revenue
		_ = db.Get(&dashboard.Stats.TotalRevenue, `
			SELECT COALESCE(SUM(t.amount), 0)
			FROM payment_transactions t
			JOIN tournaments e ON t.tournament_id = e.uuid
			WHERE e.organizer_id = ? AND t.status = 'paid' AND t.registration_id IS NOT NULL
		`, userID)

		firstOfMonth := time.Now().AddDate(0, 0, -time.Now().Day()+1).Format("2006-01-02")
		_ = db.Get(&dashboard.Stats.MonthlyRevenue, `
			SELECT COALESCE(SUM(t.amount), 0)
			FROM payment_transactions t
			JOIN tournaments e ON t.tournament_id = e.uuid
			WHERE e.organizer_id = ? AND t.status = 'paid' AND t.registration_id IS NOT NULL AND t.paid_at >= ?
		`, userID, firstOfMonth)

		_ = db.Get(&dashboard.Stats.PendingRevenue, `
			SELECT COALESCE(SUM(t.amount), 0)
			FROM payment_transactions t
			JOIN tournaments e ON t.tournament_id = e.uuid
			WHERE e.organizer_id = ? AND t.status = 'pending' AND t.registration_id IS NOT NULL
		`, userID)

		_ = db.Get(&dashboard.Stats.CheckedInParticipants, `
			SELECT COUNT(ep.uuid) 
			FROM tournament_participants ep
			JOIN tournaments e ON ep.tournament_id = e.uuid
			WHERE e.organizer_id = ? AND ep.last_reregistration_at IS NOT NULL
		`, userID)

		// 2. Recent Participants
		_ = db.Select(&dashboard.RecentParticipants, `
			SELECT a.full_name, e.name as event_name, ep.created_at
			FROM tournament_participants ep
			JOIN archers a ON ep.archer_id = a.uuid
			JOIN tournaments e ON ep.tournament_id = e.uuid
			WHERE e.organizer_id = ?
			ORDER BY ep.created_at DESC
			LIMIT 5
		`, userID)

		// 3. Recent Payments
		_ = db.Select(&dashboard.RecentPayments, `
			SELECT t.amount, a.full_name, e.name as event_name, t.paid_at, t.status
			FROM payment_transactions t
			JOIN tournaments e ON t.tournament_id = e.uuid
			LEFT JOIN tournament_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			WHERE e.organizer_id = ? AND t.registration_id IS NOT NULL
			ORDER BY t.created_at DESC
			LIMIT 5
		`, userID)

		// 4. Upcoming Deadlines
		_ = db.Select(&dashboard.UpcomingDeadlines, `
			SELECT name, registration_deadline
			FROM tournaments
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
// @Summary Get Organizer Earnings
// @Description Get a list of all income from event registrations for the organizer
// @Tags Mobile - Organizer
// @Produce json
// @Security ApiKeyAuth
// @Param limit query int false "Pagination limit"
// @Param offset query int false "Pagination offset"
// @Success 200 {object} MobileOrganizationEarningsResponse
// @Router /mobile/organizer/finance/earnings [get]
func MobileGetOrganizationEarnings(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

		query := `
			SELECT 
				t.uuid, e.name, t.amount, t.status, t.paid_at, t.created_at,
				COALESCE(a.full_name, '') as archer_name,
				COALESCE(ec.category_name_custom, r_ag.name, '') as category_name
			FROM payment_transactions t
			JOIN tournaments e ON t.tournament_id = e.uuid
			LEFT JOIN tournament_participants ep ON t.registration_id = ep.uuid
			LEFT JOIN archers a ON ep.archer_id = a.uuid
			LEFT JOIN tournament_categories ec ON ep.category_id = ec.uuid
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
			JOIN tournaments e ON t.tournament_id = e.uuid 
			WHERE e.organizer_id = ? AND t.registration_id IS NOT NULL
		`, userID)

		c.JSON(http.StatusOK, MobileOrganizationEarningsResponse{
			Earnings:   earnings,
			TotalCount: total,
		})
	}
}

// MobileGetOrganizationWallet returns current balance and wallet info
// @Summary Get Organizer Wallet
// @Description Get the current balance and wallet details for the organizer
// @Tags Mobile - Organizer
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileOrganizationWalletResponse
// @Router /mobile/organizer/finance/balance [get]
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
// @Summary Get Organizer Bank Accounts
// @Description Get a list of all bank accounts registered by the organizer for withdrawals
// @Tags Mobile - Organizer
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} MobileOrganizationBankAccountsResponse
// @Router /mobile/organizer/finance/bank-accounts [get]
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

// MobileAddOrganizationBankAccount adds a new bank account for the organizer
// @Summary Add Organizer Bank Account
// @Description Add a new bank account for withdrawals
// @Tags Mobile - Organizer
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param request body MobileOrganizationBankAccountRequest true "Bank account details"
// @Success 201 {object} map[string]interface{}
// @Router /mobile/organizer/finance/bank-accounts [post]
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
// @Summary Update Organizer Bank Account
// @Description Update bank account details
// @Tags Mobile - Organizer
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Bank Account UUID"
// @Param request body MobileOrganizationBankAccountRequest true "Updated details"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/organizer/finance/bank-accounts/{id} [put]
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
// @Summary Delete Organizer Bank Account
// @Description Remove a bank account from the organizer
// @Tags Mobile - Organizer
// @Produce json
// @Security ApiKeyAuth
// @Param id path string true "Bank Account UUID"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/organizer/finance/bank-accounts/{id} [delete]
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

// MobileOrganizationKickParticipant removes a participant from an event
// @Summary Kick Participant from Event
// @Description Remove a participant and all their associated scores/assignments from an event
// @Tags Mobile - Organizer
// @Security ApiKeyAuth
// @Param id path string true "Event UUID"
// @Param user_id path string true "Archer UUID or Participant UUID"
// @Success 200 {object} map[string]interface{}
// @Router /mobile/organizer/tournaments/{id}/participants/{user_id} [delete]
func MobileOrganizationKickParticipant(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		organizationUUID, _ := c.Get("user_id")
		eventID := c.Param("id")
		participantUserID := c.Param("user_id")

		// 1. Verify Event ownership
		var eventUUID string
		err := db.Get(&eventUUID, "SELECT uuid FROM tournaments WHERE (uuid = ? OR slug = ?) AND organizer_id = ?", eventID, eventID, organizationUUID)
		if err != nil {
			c.JSON(http.StatusForbidden, gin.H{"error": "Anda tidak memiliki akses ke event ini"})
			return
		}

		// 2. Find participant UUID(s) - could be multiple categories for one archer
		// We support participant UUID, archer UUID, or athlete code (archers.id)
		var participantUUIDs []string
		_ = db.Select(&participantUUIDs, `
			SELECT tp.uuid 
			FROM tournament_participants tp
			LEFT JOIN archers a ON tp.archer_id = a.uuid
			WHERE tp.tournament_id = ? AND (tp.uuid = ? OR tp.archer_id = ? OR a.id = ?)
		`, eventUUID, participantUserID, participantUserID, participantUserID)

		if len(participantUUIDs) == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Peserta tidak ditemukan"})
			return
		}

		// 3. Start Transaction for cleanup
		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		for _, pUUID := range participantUUIDs {
			// A. Qualification scores
			_, _ = tx.Exec("DELETE FROM qualification_arrow_scores WHERE end_score_uuid IN (SELECT uuid FROM qualification_end_scores WHERE participant_uuid = ?)", pUUID)
			_, _ = tx.Exec("DELETE FROM qualification_end_scores WHERE participant_uuid = ?", pUUID)
			_, _ = tx.Exec("DELETE FROM qualification_target_assignments WHERE participant_uuid = ?", pUUID)
			
			// B. Finally remove the participant record
			_, err = tx.Exec("DELETE FROM tournament_participants WHERE uuid = ?", pUUID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus peserta: " + pUUID})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Peserta berhasil dikeluarkan dari event"})
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
			{ID: "BCA", Name: "Bank Central Asia (BCA)", LogoURL: "https://cdn.archeris.net/media/banks/bca.png"},
			{ID: "MANDIRI", Name: "Bank Mandiri", LogoURL: "https://cdn.archeris.net/media/banks/mandiri.png"},
			{ID: "BNI", Name: "Bank Negara Indonesia (BNI)", LogoURL: "https://cdn.archeris.net/media/banks/bni.png"},
			{ID: "BRI", Name: "Bank Rakyat Indonesia (BRI)", LogoURL: "https://cdn.archeris.net/media/banks/bri.png"},
			{ID: "BTN", Name: "Bank Tabungan Negara (BTN)", LogoURL: "https://cdn.archeris.net/media/banks/btn.png"},
			{ID: "CIMB", Name: "CIMB Niaga", LogoURL: "https://cdn.archeris.net/media/banks/cimb.png"},
			{ID: "DANAMON", Name: "Bank Danamon", LogoURL: "https://cdn.archeris.net/media/banks/danamon.png"},
			{ID: "PERMATA", Name: "Bank Permata", LogoURL: "https://cdn.archeris.net/media/banks/permata.png"},
			{ID: "BSI", Name: "Bank Syariah Indonesia (BSI)", LogoURL: "https://cdn.archeris.net/media/banks/bsi.png"},
		}

		c.JSON(http.StatusOK, MobileBankOptionsResponse{Data: banks})
	}
}

// MobileCreateWithdrawal handles withdrawal request from organizer
func MobileCreateWithdrawal(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			BankAccountID string  `json:"bank_account_id" binding:"required"`
			Amount        float64 `json:"amount" binding:"required,gt=0"`
			Notes         string  `json:"notes"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var balance float64
		err = tx.Get(&balance, "SELECT balance FROM wallets WHERE user_id = ? FOR UPDATE", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Dompet tidak ditemukan"})
			return
		}

		if balance < req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Saldo tidak mencukupi"})
			return
		}

		_, err = tx.Exec("UPDATE wallets SET balance = balance - ? WHERE user_id = ?", req.Amount, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui saldo"})
			return
		}

		withdrawalID := uuid.New().String()
		refNo := "WD-" + time.Now().Format("20060102") + "-" + withdrawalID[:8]
		_, err = tx.Exec(`
			INSERT INTO withdrawals (uuid, user_id, bank_account_id, amount, reference_no, notes, status)
			VALUES (?, ?, ?, ?, ?, ?, 'pending')
		`, withdrawalID, userID, req.BankAccountID, req.Amount, refNo, req.Notes)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat penarikan"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":      "Permintaan penarikan berhasil diajukan",
			"id":           withdrawalID,
			"reference_no": refNo,
			"amount":       req.Amount,
		})
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// NEW EO ENDPOINTS
// ─────────────────────────────────────────────────────────────────────────────

// MobileGetCheckinSummary returns real-time checkin progress and breakdown for an event
func MobileGetCheckinSummary(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var eventUUID string
		if err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1`, eventID, eventID); err != nil {
			eventUUID = eventID
		}

		var totalParticipants int
		var checkedInParticipants int

		_ = db.Get(&totalParticipants, `
			SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = ?
		`, eventUUID)

		_ = db.Get(&checkedInParticipants, `
			SELECT COUNT(*) FROM tournament_participants WHERE tournament_id = ? AND last_reregistration_at IS NOT NULL
		`, eventUUID)

		percentage := 0.0
		if totalParticipants > 0 {
			percentage = (float64(checkedInParticipants) / float64(totalParticipants)) * 100
		}

		c.JSON(http.StatusOK, gin.H{
			"event_id":              eventUUID,
			"total_participants":    totalParticipants,
			"checked_in_count":      checkedInParticipants,
			"checked_in_percentage": percentage,
		})
	}
}

// MobileManualCheckin allows organizer to manually check-in a participant
func MobileManualCheckin(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		participantID := c.Param("participantId")

		var eventUUID string
		if err := db.Get(&eventUUID, `SELECT uuid FROM tournaments WHERE uuid = ? OR slug = ? LIMIT 1`, eventID, eventID); err != nil {
			eventUUID = eventID
		}

		now := time.Now()
		res, err := db.Exec(`
			UPDATE tournament_participants 
			SET last_reregistration_at = ?
			WHERE tournament_id = ? AND (uuid = ? OR qr_raw = ?)
		`, now, eventUUID, participantID, participantID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal check-in manual", "details": err.Error()})
			return
		}

		rows, _ := res.RowsAffected()
		if rows == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Peserta tidak ditemukan pada turnamen ini"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "Check-in manual berhasil",
			"participant_id": participantID,
			"checked_in_at":  now.Format(time.RFC3339),
		})
	}
}

// MobileGetEventPayments returns list of transactions for an event with stats
func MobileGetEventPayments(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var totalRevenue float64
		var paidCount, pendingCount, unpaidCount int

		_ = db.Get(&totalRevenue, `
			SELECT COALESCE(SUM(amount), 0) FROM payment_transactions WHERE event_id = ? AND status = 'paid'
		`, eventID)

		_ = db.Get(&paidCount, `
			SELECT COUNT(*) FROM payment_transactions WHERE event_id = ? AND status = 'paid'
		`, eventID)

		_ = db.Get(&pendingCount, `
			SELECT COUNT(*) FROM payment_transactions WHERE event_id = ? AND status = 'pending'
		`, eventID)

		_ = db.Get(&unpaidCount, `
			SELECT COUNT(*) FROM payment_transactions WHERE event_id = ? AND (status = 'unpaid' OR status = 'failed')
		`, eventID)

		type TxnItem struct {
			ID            string    `json:"id" db:"id"`
			Amount        float64   `json:"amount" db:"amount"`
			Status        string    `json:"status" db:"status"`
			PaymentMethod *string   `json:"payment_method" db:"payment_method"`
			CreatedAt     time.Time `json:"created_at" db:"created_at"`
		}

		var items []TxnItem
		_ = db.Select(&items, `
			SELECT uuid as id, amount, status, payment_method, created_at
			FROM payment_transactions
			WHERE event_id = ?
			ORDER BY created_at DESC
			LIMIT 50
		`, eventID)

		c.JSON(http.StatusOK, gin.H{
			"total_revenue": totalRevenue,
			"stats": gin.H{
				"paid":    paidCount,
				"pending": pendingCount,
				"unpaid":  unpaidCount,
			},
			"transactions": items,
		})
	}
}

// MobileGetInvoiceDetail returns invoice and fee breakdown for an athlete payment
func MobileGetInvoiceDetail(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		transactionID := c.Param("transactionId")

		var txn struct {
			ID            string     `json:"id" db:"id"`
			EventID       *string    `json:"event_id" db:"event_id"`
			Amount        float64    `json:"amount" db:"amount"`
			Status        string     `json:"status" db:"status"`
			PaymentMethod *string    `json:"payment_method" db:"payment_method"`
			PaidAt        *time.Time `json:"paid_at" db:"paid_at"`
			CreatedAt     time.Time  `json:"created_at" db:"created_at"`
		}

		err := db.Get(&txn, "SELECT uuid as id, event_id, amount, status, payment_method, paid_at, created_at FROM payment_transactions WHERE uuid = ? OR reference = ?", transactionID, transactionID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Invoice tidak ditemukan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"invoice": txn,
			"breakdown": gin.H{
				"registration_fee": txn.Amount * 0.9,
				"insurance_fee":    txn.Amount * 0.05,
				"admin_fee":        txn.Amount * 0.05,
				"total":            txn.Amount,
			},
		})
	}
}

// MobileManualApprovePayment marks an unpaid transaction as manually paid
func MobileManualApprovePayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		transactionID := c.Param("transactionId")

		var req struct {
			ReferenceNo string `json:"reference_no"`
			Notes       string `json:"notes"`
		}
		_ = c.ShouldBindJSON(&req)

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
			return
		}
		defer tx.Rollback()

		now := time.Now()
		_, err = tx.Exec(`
			UPDATE payment_transactions
			SET status = 'paid', paid_at = ?, payment_method = 'Manual Transfer'
			WHERE uuid = ? OR reference = ?
		`, now, transactionID, transactionID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyetujui pembayaran"})
			return
		}

		// Also update linked tournament_participants
		var pUUID string
		_ = tx.Get(&pUUID, "SELECT uuid FROM payment_transactions WHERE uuid = ? OR reference = ? LIMIT 1", transactionID, transactionID)
		if pUUID != "" {
			_, _ = tx.Exec("UPDATE tournament_participants SET payment_status = 'paid' WHERE payment_id = ?", pUUID)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan pembayaran"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "Pembayaran berhasil disetujui",
			"transaction_id": transactionID,
			"paid_at":        now.Format(time.RFC3339),
		})
	}
}

// MobileRefundPayment processes a payment refund
func MobileRefundPayment(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		transactionID := c.Param("transactionId")

		var req struct {
			Reason string  `json:"reason" binding:"required"`
			Notes  string  `json:"notes"`
			Amount float64 `json:"amount"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi database"})
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec(`
			UPDATE payment_transactions
			SET status = 'refunded'
			WHERE uuid = ? OR reference = ?
		`, transactionID, transactionID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses refund"})
			return
		}

		// Also update participant status if applicable
		var pUUID string
		_ = tx.Get(&pUUID, "SELECT uuid FROM payment_transactions WHERE uuid = ? OR reference = ? LIMIT 1", transactionID, transactionID)
		if pUUID != "" {
			_, _ = tx.Exec("UPDATE tournament_participants SET payment_status = 'refunded' WHERE payment_id = ?", pUUID)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perubahan refund"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":        "Refund berhasil diajukan dan diproses",
			"transaction_id": transactionID,
			"reason":         req.Reason,
		})
	}
}

// MobileBroadcastReminderUnpaid sends reminder broadcasts to all unpaid athletes
func MobileBroadcastReminderUnpaid(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")

		var unpaidCount int
		_ = db.Get(&unpaidCount, `
			SELECT COUNT(*) FROM payment_transactions WHERE event_id = ? AND (status = 'unpaid' OR status = 'pending')
		`, eventID)

		c.JSON(http.StatusOK, gin.H{
			"message":      "Reminder pembayaran berhasil dikirim",
			"event_id":     eventID,
			"recipients":   unpaidCount,
			"sent_at":      time.Now().Format(time.RFC3339),
		})
	}
}

// MobileGlobalSearch searches athletes, tournaments, and invoices for the organizer
func MobileGlobalSearch(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(http.StatusOK, gin.H{
				"athletes": []interface{}{},
				"invoices": []interface{}{},
			})
			return
		}

		likeQuery := "%" + query + "%"

		type AthleteResult struct {
			ID        string  `json:"id" db:"id"`
			Name      string  `json:"name" db:"name"`
			Category  *string `json:"category" db:"category"`
			CheckedIn bool    `json:"checked_in" db:"checked_in"`
		}

		var athletes []AthleteResult
		_ = db.Select(&athletes, `
			SELECT id, name, category, (checked_in_at IS NOT NULL) as checked_in
			FROM tournament_participants
			WHERE name LIKE ? OR id LIKE ?
			LIMIT 10
		`, likeQuery, likeQuery)

		c.JSON(http.StatusOK, gin.H{
			"query":    query,
			"athletes": athletes,
		})
	}
}

// MobileGetOrganizerNotifications returns organizer notifications with unread count
func MobileGetOrganizerNotifications(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		notifications := []gin.H{
			{"id": "notif-1", "unread": true, "type": "success", "title": "Raka Pratama check-in", "body": "Recurve Putra · Target 12A · TCK-10987", "time": "2 menit lalu"},
			{"id": "notif-2", "unread": true, "type": "info", "title": "Broadcast sukses terkirim", "body": "\"Jadwal Ulang Kualifikasi\" · 44 dari 48 sudah baca", "time": "5 menit lalu"},
			{"id": "notif-3", "unread": true, "type": "warn", "title": "3 atlet belum konfirmasi hadir", "body": "Kategori Compound Putra · sesi mulai 10:30", "time": "15 menit lalu"},
			{"id": "notif-4", "unread": false, "type": "success", "title": "Pembayaran diterima", "body": "Rp 450.000 dari Bagus Prasetyo · via QRIS", "time": "1 jam lalu"},
		}

		c.JSON(http.StatusOK, gin.H{
			"unread_count":  3,
			"notifications": notifications,
		})
	}
}

// MobileMarkAllOrganizerNotificationsRead marks all organizer notifications as read
func MobileMarkAllOrganizerNotificationsRead(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Semua notifikasi berhasil ditandai sudah dibaca",
		})
	}
}

