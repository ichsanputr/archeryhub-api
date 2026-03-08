package handler

import (
	"net/http"
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
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create wallet"})
				return
			}
			wallet = Wallet{
				UUID:    newID,
				UserID:  userID.(string),
				Balance: 0,
			}
		}

		c.JSON(http.StatusOK, wallet)
	}
}

func GetWithdrawals(db *sqlx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, _ := c.Get("user_id")

		var withdrawals []Withdrawal
		err := db.Select(&withdrawals, "SELECT * FROM withdrawals WHERE user_id = ? ORDER BY created_at DESC", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch withdrawals"})
			return
		}

		c.JSON(http.StatusOK, withdrawals)
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

		tx, err := db.Beginx()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()

		// Check balance
		var balance float64
		err = tx.Get(&balance, "SELECT balance FROM wallets WHERE user_id = ? FOR UPDATE", userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balance"})
			return
		}

		if balance < req.Amount {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
			return
		}

		// Deduct balance
		_, err = tx.Exec("UPDATE wallets SET balance = balance - ? WHERE user_id = ?", req.Amount, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update balance"})
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
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create withdrawal record"})
			return
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"message": "Withdrawal request submitted", "id": withdrawalID, "reference_no": refNo})
	}
}
