package handler

import (
	"Archeris-api/utils"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Wallet struct {
	UUID      string    `json:"id" db:"uuid"`
	UserID    string    `json:"user_id" db:"user_id"`
	Balance   float64   `json:"balance" db:"balance"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type Withdrawal struct {
	UUID          string    `json:"id" db:"uuid"`
	UserID        string    `json:"user_id" db:"user_id"`
	BankAccountID string    `json:"bank_account_id" db:"bank_account_id"`
	Amount        float64   `json:"amount" db:"amount"`
	Status        string    `json:"status" db:"status"`
	ReferenceNo   string    `json:"reference_no" db:"reference_no"`
	Notes         *string   `json:"notes" db:"notes"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

func GetMyWallet(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var wallet Wallet
		err := db.Get(&wallet, "SELECT * FROM wallets WHERE user_id = ?", userID)
		if err != nil {
			// If not exists, create one
			newID := uuid.New().String()
			_, err = db.Exec("INSERT INTO wallets (uuid, user_id, balance) VALUES (?, ?, 0)", newID, userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat dompet"})
				return
			}
			wallet = Wallet{
				UUID:    newID,
				UserID:  userID.(string),
				Balance: 0,
			}
		}

		// Calculate total earnings per currency from tournament paid payments
		type CurrencyTotal struct {
			Currency string  `db:"currency"`
			Total    float64 `db:"total"`
		}
		var earnings []CurrencyTotal
		_ = db.Select(&earnings, `
			SELECT 
				COALESCE(e.currency, 'IDR') as currency,
				COALESCE(SUM(pt.amount), 0) as total
			FROM tournaments e
			JOIN tournament_participants ep ON e.uuid = ep.tournament_id
			JOIN payment_transactions pt ON ep.uuid = pt.registration_id AND pt.status = 'paid'
			WHERE e.organizer_id = ?
			GROUP BY e.currency
		`, userID)

		// Calculate total withdrawals (currently IDR)
		var totalWithdrawnIDR float64
		_ = db.Get(&totalWithdrawnIDR, "SELECT COALESCE(SUM(amount), 0) FROM withdrawals WHERE user_id = ? AND status != 'failed'", userID)

		balances := map[string]float64{
			"IDR": 0,
			"USD": 0,
		}
		totalEarningsMap := map[string]float64{
			"IDR": 0,
			"USD": 0,
		}

		for _, item := range earnings {
			curr := strings.ToUpper(item.Currency)
			totalEarningsMap[curr] = item.Total
			if curr == "IDR" {
				net := item.Total - totalWithdrawnIDR
				if net < 0 {
					net = 0
				}
				balances[curr] = net
			} else {
				balances[curr] = item.Total
			}
		}

		// Sync IDR balance into wallets table
		idrBalance := balances["IDR"]
		wallet.Balance = idrBalance
		_, _ = db.Exec("UPDATE wallets SET balance = ?, updated_at = NOW() WHERE uuid = ?", idrBalance, wallet.UUID)

		c.JSON(http.StatusOK, gin.H{
			"id":              wallet.UUID,
			"user_id":         wallet.UserID,
			"balance":         wallet.Balance,
			"balances":        balances,
			"total_earnings":  totalEarningsMap,
			"total_withdrawn": map[string]float64{"IDR": totalWithdrawnIDR},
		})
	}
}

func GetWithdrawals(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		limit, offset, page := utils.GetPaginationParams(c)

		sortBy := c.DefaultQuery("sort_by", "created_at")
		order := strings.ToUpper(c.DefaultQuery("order", "DESC"))

		// Validate sortBy
		allowedSortFields := map[string]bool{
			"amount":     true,
			"status":     true,
			"created_at": true,
		}

		if !allowedSortFields[sortBy] {
			sortBy = "created_at"
		}

		if order != "ASC" && order != "DESC" {
			order = "DESC"
		}

		// Count total
		var totalCount int
		err := db.Get(&totalCount, "SELECT COUNT(*) FROM withdrawals WHERE user_id = ?", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data penarikan"})
			return
		}

		// Get data
		var withdrawals []Withdrawal
		query := fmt.Sprintf("SELECT * FROM withdrawals WHERE user_id = ? ORDER BY %s %s LIMIT ? OFFSET ?", sortBy, order)
		err = db.Select(&withdrawals, query, userID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penarikan"})
			return
		}

		if withdrawals == nil {
			withdrawals = []Withdrawal{}
		}

		meta := utils.CalculatePagination(totalCount, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{"data": withdrawals, "meta": meta})
	}
}

func CreateWithdrawal(db *sqlx.DB) gin.HandlerFunc {
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

		// Verification: ensure bank_account_id belongs to the logged-in user
		if req.BankAccountID != "" {
			var isOwnBank bool
			_ = db.Get(&isOwnBank, "SELECT EXISTS(SELECT 1 FROM bank_accounts WHERE uuid = ? AND user_id = ?)", req.BankAccountID, userID)
			if !isOwnBank {
				c.JSON(http.StatusForbidden, gin.H{"error": "Rekening bank yang dipilih bukan milik Anda"})
				return
			}
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		// Check balance and wallet uuid
		var walletInfo struct {
			UUID    string  `db:"uuid"`
			Balance float64 `db:"balance"`
		}
		err = tx.Get(&walletInfo, "SELECT uuid, balance FROM wallets WHERE user_id = ? FOR UPDATE", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil saldo"})
			return
		}

		balance := walletInfo.Balance
		if balance < req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Saldo tidak mencukupi"})
			return
		}

		// Deduct balance
		_, err = tx.Exec("UPDATE wallets SET balance = balance - ?, updated_at = NOW() WHERE user_id = ?", req.Amount, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui saldo"})
			return
		}

		// Create withdrawal record
		withdrawalID := uuid.New().String()
		refNo := "WD-" + time.Now().Format("20060102") + "-" + withdrawalID[:8]
		_, err = tx.Exec(`
			INSERT INTO withdrawals (uuid, user_id, bank_account_id, amount, reference_no, notes)
			VALUES (?, ?, ?, ?, ?, ?)
		`, withdrawalID, userID, req.BankAccountID, req.Amount, refNo, req.Notes)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat catatan penarikan"})
			return
		}

		// Record ledger mutation
		mutationID := uuid.New().String()
		_, _ = tx.Exec(`
			INSERT INTO wallet_mutations (uuid, wallet_id, user_id, mutation_type, amount, balance_before, balance_after, reference_type, reference_id, description, created_at)
			VALUES (?, ?, ?, 'debit', ?, ?, ?, 'withdrawal', ?, ?, NOW())
		`, mutationID, walletInfo.UUID, userID, req.Amount, balance, balance-req.Amount, refNo, "Penarikan saldo ("+refNo+")")

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Permintaan penarikan berhasil diajukan", "id": withdrawalID, "reference_no": refNo})
	}
}

type WalletMutation struct {
	UUID          string  `db:"uuid"           json:"uuid"`
	WalletID      string  `db:"wallet_id"      json:"wallet_id"`
	UserID        string  `db:"user_id"        json:"user_id"`
	MutationType  string  `db:"mutation_type"  json:"mutation_type"`
	Amount        float64 `db:"amount"         json:"amount"`
	BalanceBefore float64 `db:"balance_before" json:"balance_before"`
	BalanceAfter  float64 `db:"balance_after"  json:"balance_after"`
	ReferenceType string  `db:"reference_type" json:"reference_type"`
	ReferenceID   string  `db:"reference_id"   json:"reference_id"`
	Description   *string `db:"description"    json:"description"`
	CreatedAt     string  `db:"created_at"     json:"created_at"`
}

// GetWalletMutations returns paginated ledger mutation history for the logged-in user
func GetWalletMutations(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "15"))
		if page < 1 {
			page = 1
		}
		if limit < 1 || limit > 50 {
			limit = 15
		}
		offset := (page - 1) * limit

		var totalCount int
		_ = db.Get(&totalCount, "SELECT COUNT(*) FROM wallet_mutations WHERE user_id = ?", userID)

		var mutations []WalletMutation
		_ = db.Select(&mutations, `
			SELECT uuid, wallet_id, user_id, mutation_type, amount, balance_before, balance_after, reference_type, reference_id, description, created_at
			FROM wallet_mutations
			WHERE user_id = ?
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?
		`, userID, limit, offset)

		if mutations == nil {
			mutations = []WalletMutation{}
		}

		meta := utils.CalculatePagination(totalCount, limit, offset, page)
		c.JSON(http.StatusOK, gin.H{"data": mutations, "meta": meta})
	}
}

// RecordWalletCreditTx records a credit mutation into wallet_mutations and updates wallet balance inside an existing transaction
func RecordWalletCreditTx(tx *sqlx.Tx, organizerID string, amount float64, referenceNo string, referenceType string, description string) error {
	if organizerID == "" || amount <= 0 {
		return nil
	}
	if referenceType == "" {
		referenceType = "tournament_registration"
	}

	// 1. Check idempotency: if credit for referenceNo already exists, do nothing
	var exists bool
	_ = tx.Get(&exists, "SELECT EXISTS(SELECT 1 FROM wallet_mutations WHERE reference_id = ? AND mutation_type = 'credit')", referenceNo)
	if exists {
		return nil
	}

	// 2. Ensure wallet exists FOR UPDATE
	var walletInfo struct {
		UUID    string  `db:"uuid"`
		Balance float64 `db:"balance"`
	}
	err := tx.Get(&walletInfo, "SELECT uuid, balance FROM wallets WHERE user_id = ? FOR UPDATE", organizerID)
	if err != nil {
		// Create wallet if not exists
		walletUUID := uuid.New().String()
		_, err = tx.Exec("INSERT INTO wallets (uuid, user_id, balance) VALUES (?, ?, ?)", walletUUID, organizerID, amount)
		if err != nil {
			return err
		}
		walletInfo.UUID = walletUUID
		walletInfo.Balance = 0
	} else {
		// Update balance
		_, err = tx.Exec("UPDATE wallets SET balance = balance + ?, updated_at = NOW() WHERE uuid = ?", amount, walletInfo.UUID)
		if err != nil {
			return err
		}
	}

	balanceBefore := walletInfo.Balance
	balanceAfter := balanceBefore + amount

	// 3. Insert mutation record
	mutationID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO wallet_mutations (uuid, wallet_id, user_id, mutation_type, amount, balance_before, balance_after, reference_type, reference_id, description, created_at)
		VALUES (?, ?, ?, 'credit', ?, ?, ?, ?, ?, ?, NOW())
	`, mutationID, walletInfo.UUID, organizerID, amount, balanceBefore, balanceAfter, referenceType, referenceNo, description)

	return err
}

// RecordWalletCredit is a standalone wrapper around RecordWalletCreditTx
func RecordWalletCredit(db *sqlx.DB, organizerID string, amount float64, referenceNo string, referenceType string, description string) error {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := RecordWalletCreditTx(tx, organizerID, amount, referenceNo, referenceType, description); err != nil {
		return err
	}
	return tx.Commit()
}

