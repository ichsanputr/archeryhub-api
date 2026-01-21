package utils

import (
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// LogActivity inserts a record into the activity_logs table
func LogActivity(db *sqlx.DB, userID, eventID, action, entityType, entityID, description, ipAddress, userAgent string) {
	logID := uuid.New().String()
	query := `
		INSERT INTO activity_logs (id, user_id, event_id, action, entity_type, entity_id, description, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	// Use eventID if provided, otherwise it can be empty/NULL
	var eID interface{}
	eID = eventID
	if eventID == "" {
		eID = nil
	}

	db.Exec(query, logID, userID, eID, action, entityType, entityID, description, ipAddress, userAgent)
}
