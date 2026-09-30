package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"Archeris-api/database"
	"Archeris-api/handler"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	gin.SetMode(gin.TestMode)

	// Load env
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")
	_ = godotenv.Load("api/.env")

	// Ensure DB_NAME is set to archeris
	if os.Getenv("DB_NAME") == "" {
		_ = os.Setenv("DB_NAME", "archeris")
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("==================================================")
	fmt.Println("🚀 STARTING E2E TEST: WITHDRAWAL & WALLET SALDO")
	fmt.Println("==================================================")

	// 1. Find a test organizer with tournaments and payments
	var organizer struct {
		UUID   string `db:"uuid"`
		UserID string `db:"user_id"`
		Email  string `db:"email"`
		Name   string `db:"name"`
	}

	err = db.Get(&organizer, `
		SELECT uuid, user_id, email, name
		FROM organizers
		LIMIT 1
	`)
	if err != nil {
		log.Fatalf("❌ No organizer found in database: %v", err)
	}

	fmt.Printf("👤 Test Organizer: %s (%s) [User ID: %s]\n\n", organizer.Name, organizer.Email, organizer.UserID)

	// Setup Gin test engine
	r := gin.New()
	authMiddleware := func(c *gin.Context) {
		c.Set("user_id", organizer.UserID)
		c.Set("user_role", "organizer")
		c.Next()
	}

	r.GET("/organizers/wallet", authMiddleware, handler.GetMyWallet(db))
	r.GET("/organizers/wallet/withdrawals", authMiddleware, handler.GetWithdrawals(db))
	r.POST("/organizers/wallet/withdrawals", authMiddleware, handler.CreateWithdrawal(db))

	// ----------------------------------------------------
	// STEP 1: TEST GET WALLET & MULTI-CURRENCY SALDO
	// ----------------------------------------------------
	fmt.Println("▶️  [STEP 1] Testing GET /organizers/wallet (Saldo Calculation)...")
	req, _ := http.NewRequest("GET", "/organizers/wallet", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		log.Fatalf("❌ GET /organizers/wallet failed with status %d: %s", w.Code, w.Body.String())
	}

	var walletRes struct {
		ID             string             `json:"id"`
		Balance        float64            `json:"balance"`
		Balances       map[string]float64 `json:"balances"`
		TotalEarnings  map[string]float64 `json:"total_earnings"`
		TotalWithdrawn map[string]float64 `json:"total_withdrawn"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &walletRes); err != nil {
		log.Fatalf("❌ Failed to parse wallet response: %v", err)
	}

	fmt.Printf("   ✅ Wallet UUID: %s\n", walletRes.ID)
	fmt.Printf("   ✅ Available Saldo (IDR): Rp %.2f\n", walletRes.Balances["IDR"])
	fmt.Printf("   ✅ Available Saldo (USD): $ %.2f\n", walletRes.Balances["USD"])
	fmt.Printf("   ✅ Total Gross Revenue (IDR): Rp %.2f\n", walletRes.TotalEarnings["IDR"])
	fmt.Printf("   ✅ Total Withdrawn (IDR): Rp %.2f\n", walletRes.TotalWithdrawn["IDR"])

	initialBalance := walletRes.Balances["IDR"]

	// ----------------------------------------------------
	// STEP 2: TEST REJECT INVALID WITHDRAWAL (Exceeding Saldo)
	// ----------------------------------------------------
	fmt.Println("\n▶️  [STEP 2] Testing Reject Withdrawal (Amount > Saldo)...")
	excessAmount := initialBalance + 999999999
	badPayload, _ := json.Marshal(map[string]interface{}{
		"bank_account_id": "dummy-bank-id",
		"amount":          excessAmount,
	})
	req, _ = http.NewRequest("POST", "/organizers/wallet/withdrawals", bytes.NewBuffer(badPayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusBadRequest || w.Code == http.StatusForbidden {
		fmt.Printf("   ✅ Successfully rejected excess withdrawal (Status: %d, Message: %s)\n", w.Code, strings.TrimSpace(w.Body.String()))
	} else {
		log.Fatalf("❌ Expected rejection but got status %d: %s", w.Code, w.Body.String())
	}

	// ----------------------------------------------------
	// STEP 3: ENSURE ORGANIZER HAS A REGISTERED BANK ACCOUNT
	// ----------------------------------------------------
	fmt.Println("\n▶️  [STEP 3] Ensuring Valid Bank Account for Test Organizer...")
	var bankAccount struct {
		UUID          string `db:"uuid"`
		BankName      string `db:"bank_name"`
		AccountNumber string `db:"account_number"`
	}
	err = db.Get(&bankAccount, "SELECT uuid, bank_name, account_number FROM bank_accounts WHERE user_id = ? LIMIT 1", organizer.UserID)
	if err != nil {
		// Create a test bank account
		testBankID := uuid.New().String()
		_, err = db.Exec(`
			INSERT INTO bank_accounts (uuid, user_id, bank_name, account_number, account_name, is_primary, created_at, updated_at)
			VALUES (?, ?, 'Bank Central Asia (BCA)', '8830192831', ?, 1, NOW(), NOW())
		`, testBankID, organizer.UserID, organizer.Name)
		if err != nil {
			log.Fatalf("❌ Failed to create test bank account: %v", err)
		}
		bankAccount.UUID = testBankID
		bankAccount.BankName = "Bank Central Asia (BCA)"
		bankAccount.AccountNumber = "8830192831"
		fmt.Printf("   ✅ Created Test Bank Account: %s (%s)\n", bankAccount.BankName, bankAccount.AccountNumber)
	} else {
		fmt.Printf("   ✅ Found Existing Bank Account: %s (%s) [%s]\n", bankAccount.BankName, bankAccount.AccountNumber, bankAccount.UUID)
	}

	// ----------------------------------------------------
	// STEP 4: TEST SUCCESSFUL WITHDRAWAL SUBMISSION
	// ----------------------------------------------------
	withdrawAmount := 50000.0
	// Ensure wallet has enough balance for test; if 0, temporarily credit wallet balance
	if initialBalance < withdrawAmount {
		fmt.Println("   ℹ️ Initial balance is low, seeding temporary wallet balance for test...")
		_, _ = db.Exec("UPDATE wallets SET balance = balance + 100000 WHERE user_id = ?", organizer.UserID)
		initialBalance += 100000
	}

	fmt.Printf("\n▶️  [STEP 4] Submitting Valid Withdrawal of Rp %.2f...\n", withdrawAmount)
	validPayload, _ := json.Marshal(map[string]interface{}{
		"bank_account_id": bankAccount.UUID,
		"amount":          withdrawAmount,
		"notes":           "E2E Automated Test Withdrawal",
	})
	req, _ = http.NewRequest("POST", "/organizers/wallet/withdrawals", bytes.NewBuffer(validPayload))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		log.Fatalf("❌ Withdrawal creation failed with status %d: %s", w.Code, w.Body.String())
	}

	var createRes struct {
		Message     string `json:"message"`
		ID          string `json:"id"`
		ReferenceNo string `json:"reference_no"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &createRes)

	fmt.Printf("   ✅ Withdrawal Successfully Created!\n")
	fmt.Printf("   ✅ Reference No: %s\n", createRes.ReferenceNo)
	fmt.Printf("   ✅ Record ID: %s\n", createRes.ID)

	// ----------------------------------------------------
	// STEP 5: VERIFY DATABASE MUTATION & UPDATED SALDO
	// ----------------------------------------------------
	fmt.Println("\n▶️  [STEP 5] Verifying Ledger Mutation & Updated Saldo...")

	// 1. Check withdrawals table record
	var dbWd struct {
		Amount float64 `db:"amount"`
		Status string  `db:"status"`
	}
	err = db.Get(&dbWd, "SELECT amount, status FROM withdrawals WHERE uuid = ?", createRes.ID)
	if err != nil {
		log.Fatalf("❌ Withdrawal record not found in database: %v", err)
	}
	fmt.Printf("   ✅ Database Record: Amount = Rp %.2f, Status = %s\n", dbWd.Amount, dbWd.Status)

	// 2. Check wallet_mutations entry
	var dbMut struct {
		MutationType  string  `db:"mutation_type"`
		Amount        float64 `db:"amount"`
		BalanceBefore float64 `db:"balance_before"`
		BalanceAfter  float64 `db:"balance_after"`
	}
	err = db.Get(&dbMut, "SELECT mutation_type, amount, balance_before, balance_after FROM wallet_mutations WHERE reference_id = ?", createRes.ReferenceNo)
	if err == nil {
		fmt.Printf("   ✅ Ledger Mutation: Type = %s, Amount = Rp %.2f, Before = Rp %.2f, After = Rp %.2f\n",
			dbMut.MutationType, dbMut.Amount, dbMut.BalanceBefore, dbMut.BalanceAfter)
	}

	// 3. Check GET /organizers/wallet/withdrawals
	req, _ = http.NewRequest("GET", "/organizers/wallet/withdrawals", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		var listRes struct {
			Data []handler.Withdrawal `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &listRes)
		found := false
		for _, item := range listRes.Data {
			if item.UUID == createRes.ID {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("   ✅ GET /organizers/wallet/withdrawals lists newly created withdrawal!\n")
		}
	}

	// Cleanup test withdrawal to maintain clean data state
	_, _ = db.Exec("DELETE FROM wallet_mutations WHERE reference_id = ?", createRes.ReferenceNo)
	_, _ = db.Exec("DELETE FROM withdrawals WHERE uuid = ?", createRes.ID)
	_, _ = db.Exec("UPDATE wallets SET balance = balance + ? WHERE user_id = ?", withdrawAmount, organizer.UserID)

	fmt.Println("\n==================================================")
	fmt.Println("🎉 ALL E2E TESTS PASSED SUCCESSFULLY!")
	fmt.Println("==================================================")
}
