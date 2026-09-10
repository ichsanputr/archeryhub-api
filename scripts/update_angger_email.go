package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	dsn := "ichsan:12345@tcp(151.243.222.93:30036)/archeris?parseTime=true"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("DB error:", err)
		os.Exit(1)
	}
	defer db.Close()

	// Cek data sebelum update
	var uuid, email, fullName, username string
	err = db.QueryRow(
		"SELECT uuid, email, full_name, username FROM archers WHERE username = 'angger-1e24f553' LIMIT 1",
	).Scan(&uuid, &email, &fullName, &username)
	if err != nil {
		fmt.Println("Archiver 'angger' not found:", err)
		os.Exit(1)
	}
	fmt.Println("=== Before ===")
	fmt.Printf("  UUID:     %s\n", uuid)
	fmt.Printf("  Email:    %s\n", email)
	fmt.Printf("  Name:     %s\n", fullName)
	fmt.Printf("  Username: %s\n\n", username)

	// Update email
	result, err := db.Exec(
		"UPDATE archers SET email = 'stewie4king@gmail.com' WHERE username = 'angger-1e24f553'",
	)
	if err != nil {
		fmt.Println("Update error:", err)
		os.Exit(1)
	}
	rows, _ := result.RowsAffected()
	fmt.Printf("Updated %d row(s)\n\n", rows)

	// Verify
	var newEmail string
	db.QueryRow("SELECT email FROM archers WHERE username = 'angger-1e24f553'").Scan(&newEmail)
	fmt.Println("=== After ===")
	fmt.Printf("  Email: %s\n", newEmail)
	fmt.Println("\n✅ Done!")
}