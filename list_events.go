package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// Read .env file
	b, err := os.ReadFile(".env")
	if err != nil {
		log.Fatal(err)
	}
	lines := strings.Split(string(b), "\n")
	config := map[string]string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "=", 2)
			config[parts[0]] = parts[1]
		}
	}
	host := config["DB_HOST"]
	port := config["DB_PORT"]
	user := config["DB_USER"]
	password := config["DB_PASSWORD"]
	dbname := config["DB_NAME"]

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", user, password, host, port, dbname)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Query events
	rows, err := db.Query(`SELECT uuid, code, name, short_name, venue, city, start_date, end_date, status FROM events ORDER BY start_date DESC LIMIT 100`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	fmt.Println("UUID | Code | Name | Short Name | Venue | City | Start Date | End Date | Status")
	fmt.Println("-----|------|------|------------|-------|------|------------|----------|-------")
	for rows.Next() {
		var uuid, code, name, status string
		var shortName, venue, city sql.NullString
		var startDate, endDate sql.NullTime
		if err := rows.Scan(&uuid, &code, &name, &shortName, &venue, &city, &startDate, &endDate, &status); err != nil {
			log.Fatal(err)
		}
		shortStr := ""
		if shortName.Valid {
			shortStr = shortName.String
		}
		venueStr := ""
		if venue.Valid {
			venueStr = venue.String
		}
		cityStr := ""
		if city.Valid {
			cityStr = city.String
		}
		startStr := "nil"
		if startDate.Valid {
			startStr = startDate.Time.Format("2006-01-02")
		}
		endStr := "nil"
		if endDate.Valid {
			endStr = endDate.Time.Format("2006-01-02")
		}
		fmt.Printf("%s | %s | %s | %s | %s | %s | %s | %s | %s\n",
			uuid, code, name, shortStr, venueStr, cityStr, startStr, endStr, status)
		count++
	}
	fmt.Printf("\nTotal events: %d\n", count)
}