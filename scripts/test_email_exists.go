package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := "ichsan:12345@tcp(151.243.222.93:30036)/archeris?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	email := "stewie4king@gmail.com"
	if len(os.Args) > 1 {
		email = os.Args[1]
	}

	fmt.Printf("=== Checking user: %s ===\n\n", email)

	// Check archer
	var uuid, fullName, password string
	var tv sql.NullInt64
	err = db.QueryRow("SELECT uuid, full_name, password, token_version FROM archers WHERE email = ?", email).Scan(&uuid, &fullName, &password, &tv)
	if err == sql.ErrNoRows {
		fmt.Println("❌ Not found in archers table")
	} else if err != nil {
		fmt.Println("DB error:", err)
	} else {
		fmt.Println("--- ARCHER ---")
		fmt.Printf("  UUID: %s\n", uuid)
		fmt.Printf("  Name: %s\n", fullName)
		fmt.Printf("  Password prefix: %s...\n", password[:20])
		fmt.Printf("  Is bcrypt: %v\n", len(password) > 4 && (password[:4] == "$2a$" || password[:4] == "$2b$"))
		fmt.Printf("  Token version: %v\n", tv.Int64)

		// Test bcrypt verify with common passwords
		commonPasswords := []string{"Archeris123!", "123456", "password", "archeris", "Archeris123", "angger123"}
		fmt.Println("\n  Testing common passwords:")
		for _, pwd := range commonPasswords {
			err := bcrypt.CompareHashAndPassword([]byte(password), []byte(pwd))
			if err == nil {
				fmt.Printf("    ✅ MATCH: \"%s\"\n", pwd)
			} else {
				fmt.Printf("    ❌ no match: \"%s\"\n", pwd)
			}
		}
	}

	// Check seller
	err = db.QueryRow("SELECT uuid, store_name, password, token_version FROM sellers WHERE email = ?", email).Scan(&uuid, &fullName, &password, &tv)
	if err == nil {
		fmt.Println("\n--- SELLER ---")
		fmt.Printf("  UUID: %s\n", uuid)
		fmt.Printf("  Store: %s\n", fullName)
		fmt.Printf("  Password prefix: %s...\n", password[:20])
		fmt.Printf("  Is bcrypt: %v\n", len(password) > 4 && (password[:4] == "$2a$" || password[:4] == "$2b$"))
		fmt.Printf("  Token version: %v\n", tv.Int64)
	}

	// Check organizer
	err = db.QueryRow("SELECT uuid, name, password, token_version FROM organizers WHERE email = ?", email).Scan(&uuid, &fullName, &password, &tv)
	if err == nil {
		fmt.Println("\n--- ORGANIZER ---")
		fmt.Printf("  UUID: %s\n", uuid)
		fmt.Printf("  Name: %s\n", fullName)
		fmt.Printf("  Password prefix: %s...\n", password[:20])
		fmt.Printf("  Is bcrypt: %v\n", len(password) > 4 && (password[:4] == "$2a$" || password[:4] == "$2b$"))
		fmt.Printf("  Token version: %v\n", tv.Int64)
	}
}