package database

import (
	"os"
	"time"

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

	// Configure connection pool parameters for high-concurrency tournament traffic
	db.SetMaxOpenConns(50)                  // Max simultaneous open connections
	db.SetMaxIdleConns(25)                  // Idle connections preserved in pool for instant response
	db.SetConnMaxLifetime(5 * time.Minute)  // Connection maximum lifetime to avoid stale TCP connections
	db.SetConnMaxIdleTime(2 * time.Minute)  // Idle timeout to release unused connections

	// Ensure required tables exist
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS wallet_mutations (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			uuid VARCHAR(64) NOT NULL UNIQUE,
			wallet_id VARCHAR(64) NOT NULL,
			user_id VARCHAR(64) NOT NULL,
			mutation_type ENUM('credit', 'debit') NOT NULL,
			amount DECIMAL(15, 2) NOT NULL,
			balance_before DECIMAL(15, 2) NOT NULL,
			balance_after DECIMAL(15, 2) NOT NULL,
			reference_type VARCHAR(50) NOT NULL,
			reference_id VARCHAR(100) NULL,
			description TEXT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			INDEX idx_user_id (user_id),
			INDEX idx_wallet_id (wallet_id),
			INDEX idx_created_at (created_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)

	// Ensure is_locked columns exist for scoring lock features
	_, _ = db.Exec(`ALTER TABLE qualification_sessions ADD COLUMN is_locked TINYINT(1) DEFAULT 0`)
	_, _ = db.Exec(`ALTER TABLE elimination_brackets ADD COLUMN is_locked TINYINT(1) DEFAULT 0`)

	return db, nil
}

