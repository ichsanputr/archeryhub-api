package main

import (
	"Archeris-api/database"
	"fmt"
	"log"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	schema := `
	CREATE TABLE IF NOT EXISTS docs_comments (
		uuid VARCHAR(36) PRIMARY KEY,
		doc_slug VARCHAR(191) NOT NULL,
		user_id VARCHAR(36) NULL,
		user_type VARCHAR(50) NOT NULL,
		guest_name VARCHAR(100) NULL,
		content TEXT NOT NULL,
		status VARCHAR(20) DEFAULT 'approved',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_doc_slug (doc_slug),
		INDEX idx_created_at (created_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
	`

	_, err = db.Exec(schema)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	fmt.Println("docs_comments table created successfully!")
}
