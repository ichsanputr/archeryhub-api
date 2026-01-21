package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
	)

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tables := []string{
		"ref_disciplines",
		"ref_bow_types",
		"ref_event_types",
		"ref_gender_divisions",
		"ref_age_groups",
	}

	for _, table := range tables {
		fmt.Printf("Migrating %s...\n", table)

		// 1. Add uuid column if it doesn't exist
		_, _ = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN uuid VARCHAR(36) AFTER id", table))

		// 2. Populate uuid
		rows, err := db.Query(fmt.Sprintf("SELECT id FROM %s", table))
		if err != nil {
			log.Fatalf("Failed to select from %s: %v", table, err)
		}

		for rows.Next() {
			var id int
			if err := rows.Scan(&id); err != nil {
				log.Fatal(err)
			}
			newUUID := uuid.New().String()
			_, err = db.Exec(fmt.Sprintf("UPDATE %s SET uuid = ? WHERE id = ?", table), newUUID, id)
			if err != nil {
				log.Fatalf("Failed to update %s id %d: %v", table, id, err)
			}
		}
		rows.Close()

		// 3. Make uuid non-null and unique
		_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY uuid VARCHAR(36) NOT NULL", table))
		if err != nil {
			log.Fatal(err)
		}
		_, _ = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD UNIQUE (uuid)", table))

		// 4. Drop old id and set uuid as primary key
		// We first need to drop the auto_increment but we can't just set it to primary key without removing the old one.
		_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s MODIFY id INT NOT NULL", table)) // Remove auto_increment
		if err != nil {
			log.Fatal(err)
		}
		_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s DROP PRIMARY KEY", table))
		if err != nil {
			log.Fatal(err)
		}
		_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s DROP COLUMN id", table))
		if err != nil {
			log.Fatal(err)
		}
		_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD PRIMARY KEY (uuid)", table))
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Successfully migrated %s to UUID PK\n", table)
	}

	fmt.Println("All reference tables migrated successfully.")
}
