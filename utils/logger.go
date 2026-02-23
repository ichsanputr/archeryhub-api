package utils

import (
	"database/sql"
	"github.com/google/uuid"
)

// Execer is an interface that matches both *sqlx.DB and *sqlx.Tx
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// LogActivity inserts a record into the activity_logs table
// It takes an interface to support both *sqlx.DB and *sqlx.Tx
func LogActivity(db interface {
	Exec(query string, args ...any) (sql.Result, error)
}, userID, eventID, action, entityType, entityID, description, ipAddress, userAgent string) {
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

// LogScorekeeperAction inserts a record into the scorekeeper_logs table
func LogScorekeeperAction(db interface {
	Exec(query string, args ...any) (sql.Result, error)
}, skUUID, orgUUID, eventUUID, action, details, ipAddress, userAgent string) {
	query := `
		INSERT INTO scorekeeper_logs (scorekeeper_uuid, organization_uuid, event_uuid, action, details, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	var eID interface{} = eventUUID
	if eventUUID == "" {
		eID = nil
	}

	db.Exec(query, skUUID, orgUUID, eID, action, details, ipAddress, userAgent)
}
