package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPass := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT uuid, name FROM events")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	type Event struct {
		UUID string
		Name string
	}
	var events []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.UUID, &e.Name); err != nil {
			log.Fatal(err)
		}
		events = append(events, e)
	}

	rand.Seed(time.Now().UnixNano())

	for _, event := range events {
		fmt.Printf("Seeding gallery for event: %s\n", event.Name)
		
		// Add 5-10 random images
		numImages := rand.Intn(6) + 5 // 5 to 10
		
		for i := 0; i < numImages; i++ {
			imageUUID := uuid.New().String()
			// Use random seed for picsum to get different images
			imageURL := fmt.Sprintf("https://picsum.photos/seed/%s/800/600", imageUUID)
			caption := fmt.Sprintf("Gallery image %d for %s", i+1, event.Name)
			
			_, err = db.Exec(`
				INSERT INTO event_images (uuid, event_id, url, caption, alt_text, display_order, is_primary)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, imageUUID, event.UUID, imageURL, caption, event.Name, i+1, 0)
			
			if err != nil {
				fmt.Printf("Error inserting image: %v\n", err)
			}
		}
	}

	fmt.Println("Successfully seeded gallery images for all events.")
}
