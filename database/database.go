package database

import (
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func InitDB() (*sqlx.DB, error) {
	// Use environment variables with sensible local defaults
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "3306")
	user := getEnv("DB_USER", "root")
	password := getEnv("DB_PASSWORD", "")
	dbname := getEnv("DB_NAME", "Archeris")

	// loc=Local keeps the driver's interpretation in sync with MySQL's
	// SYSTEM session time_zone (both follow the host OS timezone). Without it,
	// MySQL wall-clock values (+07:00) are parsed as UTC, shifting every
	// TIMESTAMP/DATETIME ~7 hours into the future.
	dsn := user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname + "?parseTime=true&loc=Local"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}

