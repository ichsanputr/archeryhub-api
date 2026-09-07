package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BankAccount struct {
	UUID          string     `json:"id" db:"uuid"`
	UserID        string     `json:"user_id" db:"user_id"`
	BankName      string     `json:"bank_name" db:"bank_name"`
	AccountNumber string     `json:"account_number" db:"account_number"`
	AccountName   string     `json:"account_name" db:"account_name"`
	IsPrimary     bool       `json:"is_primary" db:"is_primary"`
	Status        string     `json:"status" db:"status"`
	CustomName    *string    `json:"custom_name" db:"custom_name"`
	Type          string     `json:"type" db:"type"`
	Instructions  *string    `json:"instructions" db:"instructions"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

func GetBankAccounts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var accounts []BankAccount
		err := db.Select(&accounts, "SELECT * FROM bank_accounts WHERE user_id = ? ORDER BY is_primary DESC, created_at DESC", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data rekening bank"})
			return
		}

		c.JSON(http.StatusOK, accounts)
	}
}

func CreateBankAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			BankName      string  `json:"bank_name" binding:"required"`
			AccountNumber string  `json:"account_number" binding:"required"`
			AccountName   string  `json:"account_name" binding:"required"`
			IsPrimary     bool    `json:"is_primary"`
			CustomName    *string `json:"custom_name"`
			Type          string  `json:"type"`
			Instructions  *string `json:"instructions"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		accountID := uuid.New().String()

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var count int
		_ = tx.Get(&count, "SELECT COUNT(*) FROM bank_accounts WHERE user_id = ?", userID)
		if count == 0 {
			req.IsPrimary = true
		}

		// If this is primary, unset other primaries for this user
		if req.IsPrimary {
			_, err = tx.Exec("UPDATE bank_accounts SET is_primary = FALSE WHERE user_id = ?", userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset status utama"})
				return
			}
		}

		if req.Type == "" {
			req.Type = "bank"
		}

		_, err = tx.Exec(`
			INSERT INTO bank_accounts (uuid, user_id, bank_name, account_number, account_name, is_primary, custom_name, type, instructions)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, accountID, userID, req.BankName, req.AccountNumber, req.AccountName, req.IsPrimary, req.CustomName, req.Type, req.Instructions)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat rekening bank: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Rekening bank berhasil ditambahkan", "id": accountID})
	}
}

func UpdateBankAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		accountID := c.Param("id")

		var req struct {
			BankName      string  `json:"bank_name" binding:"required"`
			AccountNumber string  `json:"account_number" binding:"required"`
			AccountName   string  `json:"account_name" binding:"required"`
			IsPrimary     bool    `json:"is_primary"`
			CustomName    *string `json:"custom_name"`
			Type          string  `json:"type"`
			Instructions  *string `json:"instructions"`
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

		var otherCount int
		_ = tx.Get(&otherCount, "SELECT COUNT(*) FROM bank_accounts WHERE user_id = ? AND uuid != ?", userID, accountID)

		if otherCount == 0 {
			// If this is the only account, it must remain primary
			req.IsPrimary = true
		} else if req.IsPrimary {
			// If marked primary, unset any other primary
			_, err = tx.Exec("UPDATE bank_accounts SET is_primary = FALSE WHERE user_id = ? AND uuid != ?", userID, accountID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset status utama"})
				return
			}
		} else {
			// If unmarking primary, ensure at least one other account is primary
			var primaryCount int
			_ = tx.Get(&primaryCount, "SELECT COUNT(*) FROM bank_accounts WHERE user_id = ? AND uuid != ? AND is_primary = TRUE", userID, accountID)
			if primaryCount == 0 {
				_, _ = tx.Exec("UPDATE bank_accounts SET is_primary = TRUE WHERE user_id = ? AND uuid != ? ORDER BY created_at ASC LIMIT 1", userID, accountID)
			}
		}

		if req.Type == "" {
			req.Type = "bank"
		}

		_, err = tx.Exec(`
			UPDATE bank_accounts 
			SET bank_name = ?, account_number = ?, account_name = ?, is_primary = ?, custom_name = ?, type = ?, instructions = ?, updated_at = NOW()
			WHERE uuid = ? AND user_id = ?
		`, req.BankName, req.AccountNumber, req.AccountName, req.IsPrimary, req.CustomName, req.Type, req.Instructions, accountID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui rekening bank: " + err.Error()})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Rekening bank berhasil diperbarui"})
	}
}

func DeleteBankAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		accountID := c.Param("id")

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
			return
		}
		defer tx.Rollback()

		var isPrimary bool
		_ = tx.Get(&isPrimary, "SELECT is_primary FROM bank_accounts WHERE uuid = ? AND user_id = ?", accountID, userID)

		_, err = tx.Exec("DELETE FROM bank_accounts WHERE uuid = ? AND user_id = ?", accountID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus rekening bank"})
			return
		}

		// If the deleted account was primary, make the oldest remaining account primary
		if isPrimary {
			_, _ = tx.Exec("UPDATE bank_accounts SET is_primary = TRUE WHERE user_id = ? ORDER BY created_at ASC LIMIT 1", userID)
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Rekening bank berhasil dihapus"})
	}
}

func SyncBankAccounts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req []struct {
			UUID          string  `json:"id"`
			BankName      string  `json:"bank_name"`
			AccountNumber string  `json:"account_number"`
			AccountName   string  `json:"account_name"`
			IsPrimary     bool    `json:"is_primary"`
			CustomName    *string `json:"custom_name"`
			Type          string  `json:"type"`
			Instructions  *string `json:"instructions"`
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

		// Get existing accounts to determine which ones to delete
		var existing []string
		err = tx.Select(&existing, "SELECT uuid FROM bank_accounts WHERE user_id = ?", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memeriksa rekening bank lama"})
			return
		}

		existingMap := make(map[string]bool)
		for _, uuidVal := range existing {
			existingMap[uuidVal] = true
		}

		// Keep track of IDs we processed in the request
		processedIDs := make(map[string]bool)

		// Ensure only one is primary, or default first item to primary
		hasPrimary := false
		for i := range req {
			if req[i].IsPrimary {
				if !hasPrimary {
					hasPrimary = true
				} else {
					// Only the first encountered primary remains true
					req[i].IsPrimary = false
				}
			}
		}
		if !hasPrimary && len(req) > 0 {
			req[0].IsPrimary = true
			hasPrimary = true
		}

		if hasPrimary {
			_, err = tx.Exec("UPDATE bank_accounts SET is_primary = FALSE WHERE user_id = ?", userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mereset status utama"})
				return
			}
		}

		for _, item := range req {
			if item.BankName == "" || item.AccountNumber == "" || item.AccountName == "" {
				// Skip invalid/empty inputs
				continue
			}

			itemType := item.Type
			if itemType == "" {
				itemType = "bank"
			}

			// Check if this is an existing account
			if item.UUID != "" && len(item.UUID) == 36 && existingMap[item.UUID] {
				// Update
				_, err = tx.Exec(`
					UPDATE bank_accounts 
					SET bank_name = ?, account_number = ?, account_name = ?, is_primary = ?, custom_name = ?, type = ?, instructions = ?, updated_at = NOW()
					WHERE uuid = ? AND user_id = ?
				`, item.BankName, item.AccountNumber, item.AccountName, item.IsPrimary, item.CustomName, itemType, item.Instructions, item.UUID, userID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memperbarui rekening bank: " + err.Error()})
					return
				}
				processedIDs[item.UUID] = true
			} else {
				// Insert new
				newUUID := uuid.New().String()
				_, err = tx.Exec(`
					INSERT INTO bank_accounts (uuid, user_id, bank_name, account_number, account_name, is_primary, custom_name, type, instructions)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				`, newUUID, userID, item.BankName, item.AccountNumber, item.AccountName, item.IsPrimary, item.CustomName, itemType, item.Instructions)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat rekening bank: " + err.Error()})
					return
				}
			}
		}

		// Delete any existing accounts that were not in the request
		for _, uuidVal := range existing {
			if !processedIDs[uuidVal] {
				_, err = tx.Exec("DELETE FROM bank_accounts WHERE uuid = ? AND user_id = ?", uuidVal, userID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus rekening bank lama"})
					return
				}
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Rekening bank berhasil disinkronisasi"})
	}
}
