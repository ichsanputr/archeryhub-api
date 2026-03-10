package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type BankAccount struct {
	UUID          string    `json:"id" db:"uuid"`
	UserID        string    `json:"user_id" db:"user_id"`
	BankName      string    `json:"bank_name" db:"bank_name"`
	AccountNumber string    `json:"account_number" db:"account_number"`
	AccountName   string    `json:"account_name" db:"account_name"`
	IsPrimary     bool      `json:"is_primary" db:"is_primary"`
	Status        string    `json:"status" db:"status"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

func GetBankAccounts(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var accounts []BankAccount
		err := db.Select(&accounts, "SELECT * FROM bank_accounts WHERE user_id = ? ORDER BY is_primary DESC, created_at DESC", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch bank accounts"})
			return
		}

		c.JSON(http.StatusOK, accounts)
	}
}

func CreateBankAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var req struct {
			BankName      string `json:"bank_name" binding:"required"`
			AccountNumber string `json:"account_number" binding:"required"`
			AccountName   string `json:"account_name" binding:"required"`
			IsPrimary     bool   `json:"is_primary"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		accountID := uuid.New().String()

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// If this is primary, unset other primaries for this user
		if req.IsPrimary {
			_, err = tx.Exec("UPDATE bank_accounts SET is_primary = FALSE WHERE user_id = ?", userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset primary status"})
				return
			}
		}

		_, err = tx.Exec(`
			INSERT INTO bank_accounts (uuid, user_id, bank_name, account_number, account_name, is_primary)
			VALUES (?, ?, ?, ?, ?, ?)
		`, accountID, userID, req.BankName, req.AccountNumber, req.AccountName, req.IsPrimary)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bank account"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Bank account added successfully", "id": accountID})
	}
}

func UpdateBankAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		accountID := c.Param("id")

		var req struct {
			BankName      string `json:"bank_name" binding:"required"`
			AccountNumber string `json:"account_number" binding:"required"`
			AccountName   string `json:"account_name" binding:"required"`
			IsPrimary     bool   `json:"is_primary"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		if req.IsPrimary {
			_, err = tx.Exec("UPDATE bank_accounts SET is_primary = FALSE WHERE user_id = ?", userID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset primary status"})
				return
			}
		}

		_, err = tx.Exec(`
			UPDATE bank_accounts 
			SET bank_name = ?, account_number = ?, account_name = ?, is_primary = ?, updated_at = NOW()
			WHERE uuid = ? AND user_id = ?
		`, req.BankName, req.AccountNumber, req.AccountName, req.IsPrimary, accountID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bank account"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Bank account updated successfully"})
	}
}

func DeleteBankAccount(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		accountID := c.Param("id")

		_, err := db.Exec("DELETE FROM bank_accounts WHERE uuid = ? AND user_id = ?", accountID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bank account"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Bank account deleted successfully"})
	}
}
