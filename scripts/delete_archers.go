package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "root")
	dbPass := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "archeris")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&multiStatements=true",
		dbUser, dbPass, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	fmt.Println("✅ Connected to database")

	// Emails to delete
	emails := []string{
		"anggerraka2017@gmail.com",
		"anggerrakasanjayapamungkas@gmail.com",
	}

	fmt.Println("\n=== Checking users to delete ===")
	for _, email := range emails {
		var uuid, fullName string
		err := db.QueryRow("SELECT uuid, full_name FROM archers WHERE email = ?", email).Scan(&uuid, &fullName)
		if err == sql.ErrNoRows {
			fmt.Printf("❌ User not found: %s\n", email)
			continue
		} else if err != nil {
			fmt.Printf("❌ Error checking %s: %v\n", email, err)
			continue
		}
		fmt.Printf("✓ Found: %s (%s) - UUID: %s\n", email, fullName, uuid)
	}

	fmt.Println("\n=== Checking related data ===")
	for _, email := range emails {
		var uuid string
		err := db.QueryRow("SELECT uuid FROM archers WHERE email = ?", email).Scan(&uuid)
		if err != nil {
			continue
		}

		// Check event participants
		var participantCount int
		err = db.QueryRow("SELECT COUNT(*) FROM event_participants WHERE archer_id = ?", uuid).Scan(&participantCount)
		if err == nil && participantCount > 0 {
			fmt.Printf("⚠️  %s has %d event participant records\n", email, participantCount)
		}

		// Check payment transactions
		var paymentCount int
		err = db.QueryRow("SELECT COUNT(*) FROM payment_transactions WHERE archer_id = ?", uuid).Scan(&paymentCount)
		if err == nil && paymentCount > 0 {
			fmt.Printf("⚠️  %s has %d payment transaction records\n", email, paymentCount)
		}
	}

	fmt.Println("\n=== Deleting users ===")
	for _, email := range emails {
		var uuid string
		err := db.QueryRow("SELECT uuid FROM archers WHERE email = ?", email).Scan(&uuid)
		if err == sql.ErrNoRows {
			fmt.Printf("⏭️  Skipping %s (not found)\n", email)
			continue
		} else if err != nil {
			fmt.Printf("❌ Error checking %s: %v\n", email, err)
			continue
		}

		// Delete related data first
		fmt.Printf("\n🗑️  Deleting related data for %s...\n", email)

		// Delete event participants
		result, err := db.Exec("DELETE FROM event_participants WHERE archer_id = ?", uuid)
		if err != nil {
			fmt.Printf("❌ Error deleting event_participants: %v\n", err)
		} else {
			rows, _ := result.RowsAffected()
			if rows > 0 {
				fmt.Printf("   ✓ Deleted %d event_participant records\n", rows)
			}
		}

		// Delete payment transactions
		result, err = db.Exec("DELETE FROM payment_transactions WHERE archer_id = ?", uuid)
		if err != nil {
			fmt.Printf("❌ Error deleting payment_transactions: %v\n", err)
		} else {
			rows, _ := result.RowsAffected()
			if rows > 0 {
				fmt.Printf("   ✓ Deleted %d payment_transaction records\n", rows)
			}
		}

		// Delete the archer
		result, err = db.Exec("DELETE FROM archers WHERE uuid = ?", uuid)
		if err != nil {
			fmt.Printf("❌ Error deleting archer %s: %v\n", email, err)
		} else {
			rows, _ := result.RowsAffected()
			if rows > 0 {
				fmt.Printf("✅ Successfully deleted archer: %s\n", email)
			} else {
				fmt.Printf("⚠️  No rows affected for %s\n", email)
			}
		}
	}

	fmt.Println("\n=== Verification ===")
	for _, email := range emails {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM archers WHERE email = ?", email).Scan(&count)
		if err != nil {
			fmt.Printf("❌ Error verifying %s: %v\n", email, err)
		} else if count == 0 {
			fmt.Printf("✅ Confirmed deleted: %s\n", email)
		} else {
			fmt.Printf("⚠️  Still exists: %s\n", email)
		}
	}

	fmt.Println("\n✅ Done!")
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
