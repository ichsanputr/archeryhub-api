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
	stage := getEnv("STAGE", "development")

	var host, port, user, password, dbname string

	if stage == "production" {
		// Production: use hardcoded default values
		host = "151.243.222.93"
		port = "30036"
		user = "ichsan"
		password = "12345"
		dbname = "archeryhub"
	} else {
		// Development: use environment variables
		host = getEnv("DB_HOST", "151.243.222.93")
		port = getEnv("DB_PORT", "30036")
		user = getEnv("DB_USER", "ichsan")
		password = getEnv("DB_PASSWORD", "12345")
		dbname = getEnv("DB_NAME", "archeryhub")
	}

	dsn := user + ":" + password + "@tcp(" + host + ":" + port + ")/" + dbname + "?parseTime=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, err
	}

	return db, nil
}
