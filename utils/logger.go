package utils

import (
	"database/sql"
)

// Execer is an interface that matches both *sqlx.DB and *sqlx.Tx
type Execer interface {
	Exec(query string, args ...any) (sql.Result, error)
}
