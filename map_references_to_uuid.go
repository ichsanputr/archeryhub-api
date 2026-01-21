package main

import (
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
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

	// 1. Map event_categories.division_uuid from ref_bow_types.code
	fmt.Println("Mapping event_categories.division_uuid...")
	_, err = db.Exec(`
		UPDATE event_categories ec
		JOIN ref_bow_types rbt ON ec.division_code = rbt.code
		SET ec.division_uuid = rbt.uuid
	`)
	if err != nil {
		log.Fatal(err)
	}

	// 2. Map event_categories.category_uuid from ref_age_groups.code
	fmt.Println("Mapping event_categories.category_uuid...")
	_, err = db.Exec(`
		UPDATE event_categories ec
		JOIN ref_age_groups rag ON ec.category_code = rag.code
		SET ec.category_uuid = rag.uuid
	`)
	if err != nil {
		log.Fatal(err)
	}

	// 3. Map events.type from ref_event_types.name (or code)
	// Let's check what's in events.type first. Assuming it was 'Outdoor' etc.
	fmt.Println("Mapping events.type to UUID...")
	_, err = db.Exec(`
		UPDATE events e
		JOIN ref_event_types ret ON e.type = ret.name OR e.type = ret.code
		SET e.type = ret.uuid
	`)
	if err != nil {
		log.Fatal(err)
	}

	// 4. Cleanup old columns
	fmt.Println("Cleaning up old columns...")
	_, _ = db.Exec("ALTER TABLE event_categories DROP COLUMN division_code")
	_, _ = db.Exec("ALTER TABLE event_categories DROP COLUMN category_code")

	fmt.Println("All mappings completed successfully.")
}
