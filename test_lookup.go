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
	godotenv.Load()
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, pass, host, port, name)
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	eventSlug := "kalimantan-open-archery-2025"
	participantID := "jokowi-dod"

	var actualEventID string
	err = db.Get(&actualEventID, `SELECT uuid FROM events WHERE uuid = ? OR slug = ?`, eventSlug, eventSlug)
	if err != nil {
		fmt.Printf("Event not found: %v\n", err)
		return
	}
	fmt.Printf("Actual Event ID: %s\n", actualEventID)

	var foundID string
	err = db.Get(&foundID, `
		SELECT tp.uuid FROM event_participants tp
		LEFT JOIN archers a ON tp.archer_id = a.uuid
		LEFT JOIN event_archers ea ON tp.event_archer_id = ea.uuid
		WHERE tp.event_id = ? AND (
			tp.uuid = ? OR 
			a.username = ? OR 
			ea.username = ? OR 
			LOWER(REPLACE(ea.full_name, ' ', '-')) = LOWER(?)
		)
		LIMIT 1
	`, actualEventID, participantID, participantID, participantID, participantID)

	if err != nil {
		fmt.Printf("Participant not found: %v\n", err)
		
		// Additional debug: check event_archers directly
		var username string
		err = db.Get(&username, "SELECT username FROM event_archers WHERE event_id = ? AND username = ?", actualEventID, participantID)
		if err != nil {
			fmt.Printf("DEBUG: Direct event_archers lookup failed: %v\n", err)
		} else {
			fmt.Printf("DEBUG: Found in event_archers: %s\n", username)
		}
	} else {
		fmt.Printf("Found Participant UUID: %s\n", foundID)
	}
}
