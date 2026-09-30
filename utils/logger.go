package utils

import (
	"database/sql"
)

// Execer is an interface that matches both *sqlx.DB and *sqlx.Tx
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}

// LogScorekeeperAction inserts a record into the scorekeeper_logs table
func LogScorekeeperAction(db interface {
	Exec(query string, args ...any) (sql.Result, error)
}, skUUID, orgUUID, eventUUID, action, details, ipAddress, userAgent string) {
	query := `
		INSERT INTO scorekeeper_logs (scorekeeper_uuid, organization_uuid, tournament_uuid, action, details, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	var eID interface{} = eventUUID
	if eventUUID == "" {
		eID = nil
	}

	db.Exec(query, skUUID, orgUUID, eID, action, details, ipAddress, userAgent)
}
