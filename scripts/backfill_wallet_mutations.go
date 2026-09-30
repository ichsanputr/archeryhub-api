package main

import (
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

type BackfillTx struct {
	UUID           string    `db:"uuid"`
	Reference      string    `db:"reference"`
	Amount         float64   `db:"amount"`
	PaidAt         time.Time `db:"paid_at"`
	CreatedAt      time.Time `db:"created_at"`
	OrganizerID    string    `db:"organizer_id"`
	TournamentName string    `db:"tournament_name"`
	Currency       string    `db:"currency"`
}

type BackfillWithdrawal struct {
	UUID        string    `db:"uuid"`
	UserID      string    `db:"user_id"`
	Amount      float64   `db:"amount"`
	Status      string    `db:"status"`
	ReferenceNo string    `db:"reference_no"`
	Notes       *string   `db:"notes"`
	CreatedAt   time.Time `db:"created_at"`
}

func main() {
	_ = godotenv.Load("../.env")
	_ = godotenv.Load(".env")

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "3306"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "root"
	}
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "archeris"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer db.Close()

	fmt.Println("=== Starting Wallet Mutations Historical Backfill ===")

	// 1. Fetch all paid transactions
	var transactions []BackfillTx
	query := `
		SELECT 
			pt.uuid, pt.reference, pt.amount, 
			COALESCE(pt.paid_at, pt.created_at) as paid_at,
			pt.created_at,
			t.organizer_id,
			COALESCE(t.name, 'Turnamen') as tournament_name,
			COALESCE(t.currency, 'IDR') as currency
		FROM payment_transactions pt
		JOIN tournaments t ON pt.tournament_id = t.uuid
		WHERE pt.status = 'paid' AND pt.amount > 0 AND pt.registration_id IS NOT NULL 
		  AND (pt.payment_method IS NULL OR pt.payment_method NOT IN ('manual', 'manual_transfer'))
		ORDER BY pt.created_at ASC
	`
	err = db.Select(&transactions, query)
	if err != nil {
		log.Fatalf("Failed to fetch transactions: %v", err)
	}
	fmt.Printf("Found %d paid transactions to backfill.\n", len(transactions))

	// Group transactions by organizer
	orgTransactions := make(map[string][]BackfillTx)
	for _, tx := range transactions {
		if tx.OrganizerID != "" {
			orgTransactions[tx.OrganizerID] = append(orgTransactions[tx.OrganizerID], tx)
		}
	}

	for orgID, txList := range orgTransactions {
		fmt.Printf("\nProcessing Organizer ID: %s (%d transactions)\n", orgID, len(txList))

		// Ensure wallet exists
		var wallet struct {
			UUID    string  `db:"uuid"`
			Balance float64 `db:"balance"`
		}
		err = db.Get(&wallet, "SELECT uuid, balance FROM wallets WHERE user_id = ?", orgID)
		if err != nil {
			wallet.UUID = uuid.New().String()
			wallet.Balance = 0
			_, err = db.Exec("INSERT INTO wallets (uuid, user_id, balance) VALUES (?, ?, 0)", wallet.UUID, orgID)
			if err != nil {
				log.Printf("Error creating wallet for %s: %v", orgID, err)
				continue
			}
		}

		var runningBalance float64 = 0

		// Insert credit mutations
		for _, tx := range txList {
			// Check if mutation already exists
			var exists bool
			_ = db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM wallet_mutations WHERE reference_id = ? AND mutation_type = 'credit')", tx.Reference)
			if exists {
				continue
			}

			balanceBefore := runningBalance
			runningBalance += tx.Amount
			balanceAfter := runningBalance

			mutationUUID := uuid.New().String()
			desc := fmt.Sprintf("Pendaftaran tiket turnamen %s (%s)", tx.TournamentName, tx.Reference)

			_, err = db.Exec(`
				INSERT INTO wallet_mutations (uuid, wallet_id, user_id, mutation_type, amount, balance_before, balance_after, reference_type, reference_id, description, created_at)
				VALUES (?, ?, ?, 'credit', ?, ?, ?, 'tournament_registration', ?, ?, ?)
			`, mutationUUID, wallet.UUID, orgID, tx.Amount, balanceBefore, balanceAfter, tx.Reference, desc, tx.PaidAt)

			if err != nil {
				log.Printf("Failed to insert mutation for %s: %v", tx.Reference, err)
			} else {
				fmt.Printf("  + [CREDIT] %s: +%.2f (Balance: %.2f)\n", tx.Reference, tx.Amount, balanceAfter)
			}
		}

		// Process withdrawals for this organizer
		var withdrawals []BackfillWithdrawal
		_ = db.Select(&withdrawals, "SELECT uuid, user_id, amount, status, reference_no, notes, created_at FROM withdrawals WHERE user_id = ? AND status != 'failed' ORDER BY created_at ASC", orgID)

		for _, wd := range withdrawals {
			var exists bool
			_ = db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM wallet_mutations WHERE reference_id = ? AND mutation_type = 'debit')", wd.ReferenceNo)
			if exists {
				continue
			}

			balanceBefore := runningBalance
			runningBalance -= wd.Amount
			if runningBalance < 0 {
				runningBalance = 0
			}
			balanceAfter := runningBalance

			mutationUUID := uuid.New().String()
			desc := fmt.Sprintf("Penarikan saldo (%s)", wd.ReferenceNo)

			_, err = db.Exec(`
				INSERT INTO wallet_mutations (uuid, wallet_id, user_id, mutation_type, amount, balance_before, balance_after, reference_type, reference_id, description, created_at)
				VALUES (?, ?, ?, 'debit', ?, ?, ?, 'withdrawal', ?, ?, ?)
			`, mutationUUID, wallet.UUID, orgID, wd.Amount, balanceBefore, balanceAfter, wd.ReferenceNo, desc, wd.CreatedAt)

			if err != nil {
				log.Printf("Failed to insert debit mutation for %s: %v", wd.ReferenceNo, err)
			} else {
				fmt.Printf("  - [DEBIT]  %s: -%.2f (Balance: %.2f)\n", wd.ReferenceNo, wd.Amount, balanceAfter)
			}
		}

		// Update final wallet balance
		_, _ = db.Exec("UPDATE wallets SET balance = ?, updated_at = NOW() WHERE uuid = ?", runningBalance, wallet.UUID)
		fmt.Printf("  -> Final synced balance for %s: %.2f\n", orgID, runningBalance)
	}

	fmt.Println("\n=== Wallet Mutations Historical Backfill Finished Successfully ===")
}
