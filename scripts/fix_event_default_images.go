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

	// Update logo_url if null or empty
	_, err = db.Exec(`
		UPDATE events 
		SET logo_url = CONCAT('https://picsum.photos/seed/', uuid, '/400/400')
		WHERE logo_url IS NULL OR logo_url = ''
	`)
	if err != nil {
		fmt.Printf("Error updating logo_url: %v\n", err)
	}

	// Update banner_url if null or empty
	_, err = db.Exec(`
		UPDATE events 
		SET banner_url = CONCAT('https://picsum.photos/seed/', uuid, '_banner/1200/400')
		WHERE banner_url IS NULL OR banner_url = ''
	`)
	if err != nil {
		fmt.Printf("Error updating banner_url: %v\n", err)
	}

	fmt.Println("Successfully updated events with default images to Picsum images.")
}
