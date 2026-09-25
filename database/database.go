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

	// Ensure blog and newsletter tables exist
	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS news_subscribers (
			id INT AUTO_INCREMENT PRIMARY KEY,
			email VARCHAR(255) NOT NULL UNIQUE,
			is_active TINYINT(1) DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_email (email)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)

	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS blog_articles (
			id INT AUTO_INCREMENT PRIMARY KEY,
			uuid VARCHAR(36) NOT NULL UNIQUE,
			slug VARCHAR(255) NOT NULL UNIQUE,
			title VARCHAR(500) NOT NULL,
			excerpt TEXT,
			content LONGTEXT,
			category VARCHAR(100) NOT NULL,
			tags TEXT,
			image_url VARCHAR(500),
			author_name VARCHAR(255) DEFAULT 'Archeris Admin',
			author_role VARCHAR(255) DEFAULT 'Editorial Team & Archery Scoring Specialists',
			author_avatar VARCHAR(500) DEFAULT 'https://api.dicebear.com/7.x/bottts/svg?seed=ArcherisAdmin',
			read_time INT DEFAULT 5,
			views INT DEFAULT 0,
			status ENUM('draft', 'published') DEFAULT 'published',
			published_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_slug (slug),
			INDEX idx_category (category),
			INDEX idx_status_published (status, published_at)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)

	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS blog_comments (
			uuid VARCHAR(36) NOT NULL PRIMARY KEY,
			article_slug VARCHAR(255) NOT NULL,
			parent_id VARCHAR(36) NULL,
			user_id VARCHAR(36) NULL,
			user_type ENUM('archer', 'organization', 'seller', 'guest') DEFAULT 'guest',
			guest_name VARCHAR(100) NULL,
			content TEXT NOT NULL,
			status ENUM('pending', 'approved', 'spam', 'deleted') DEFAULT 'approved',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_article_slug (article_slug),
			INDEX idx_parent_id (parent_id),
			INDEX idx_status (status)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)

	_, _ = db.Exec(`
		CREATE TABLE IF NOT EXISTS organizer_subscribers (
			id INT AUTO_INCREMENT PRIMARY KEY,
			user_id VARCHAR(64) NULL,
			email VARCHAR(255) NOT NULL,
			organizer_id VARCHAR(64) NULL,
			organizer_name VARCHAR(255) NULL,
			tournament_slug VARCHAR(255) NULL,
			is_active TINYINT(1) DEFAULT 1,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			UNIQUE KEY uq_email_organizer (email, organizer_name),
			INDEX idx_email (email),
			INDEX idx_organizer (organizer_name)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)

	return db, nil
}

