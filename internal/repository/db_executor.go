// internal/repository/db_executor.go
package repository

import (
	"context"
	"database/sql"
	// No longer imports pkg/db
)

// DBExecutor defines the common database operations needed by repositories.
// Both *sqlx.DB and *sqlx.Tx implement these methods.
// This allows repositories to operate on either a direct DB connection or a transaction.
type DBExecutor interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}
