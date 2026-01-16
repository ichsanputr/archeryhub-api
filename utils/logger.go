package utils

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// LogActivity inserts a record into the activity_logs table
func LogActivity(db *sqlx.DB, userID, tournamentID, action, entityType, entityID, description, ipAddress, userAgent string) {
	logID := uuid.New().String()
	query := `
		INSERT INTO activity_logs (id, user_id, tournament_id, action, entity_type, entity_id, description, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	// Use tournamentID if provided, otherwise it can be empty/NULL
	var tID interface{}
	tID = tournamentID
	if tournamentID == "" {
		tID = nil
	}

	db.Exec(query, logID, userID, tID, action, entityType, entityID, description, ipAddress, userAgent)
}
