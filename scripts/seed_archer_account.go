package main

import (
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := "root:@tcp(localhost:3306)/archeris?parseTime=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect DB: %v", err)
	}
	defer db.Close()

	password := "password123"
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Bcrypt error: %v", err)
	}
	hashedPassword := string(hashedBytes)

	// Create/Update main archer: archer@gmail.com / rizky.pratama@gmail.com
	_, err = db.Exec(`
		UPDATE archers SET 
			password = ?,
			is_verified = 1,
			status = 'active'
		WHERE uuid = 'arc-rizky-001';
	`, hashedPassword)
	if err != nil {
		log.Fatalf("Failed to update archer: %v", err)
	}

	// Also insert a dedicated simple demo archer: archer@archeris.net / archer@gmail.com
	_, err = db.Exec(`
		INSERT INTO archers (
			uuid, id, username, email, full_name, password, bow_type, club_id, gender, status, is_verified, avatar_url
		) VALUES (
			'arc-demo-user', 'ARC-DEMO', 'archer.demo', 'archer@gmail.com', 'Rizky Pratama (Archer)',
			?, 'recurve', 'club-sac-001', 'male', 'active', 1,
			'https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=200&q=80'
		)
		ON DUPLICATE KEY UPDATE 
			password = VALUES(password),
			status = 'active',
			is_verified = 1;
	`, hashedPassword)
	if err != nil {
		log.Fatalf("Failed to insert demo archer: %v", err)
	}

	// Update all seeded archers with password123 as well
	_, err = db.Exec(`
		UPDATE archers SET password = ?, is_verified = 1 WHERE password IS NULL;
	`, hashedPassword)
	if err != nil {
		log.Fatalf("Failed to update other archers: %v", err)
	}

	fmt.Printf("? Archer Account Ready:\nEmail: archer@gmail.com (atau rizky.pratama@example.com)\nPassword: %s\n", password)
}
