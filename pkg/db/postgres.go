// pkg/db/postgres.go
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// Config holds database connection configuration.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// NewPostgresDB initializes and returns a new PostgreSQL database connection.
// It uses sqlx for enhanced database operations.
func NewPostgresDB(cfg Config) (*sqlx.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)                 // Max number of open connections
	db.SetMaxIdleConns(10)                 // Max number of idle connections
	db.SetConnMaxLifetime(5 * time.Minute) // Max lifetime of a connection

	// Ping the database to verify the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return db, nil
}

// BeginTx starts a new transaction.
// It's a helper function to ensure all repository operations can be performed within a transaction.
func BeginTx(ctx context.Context, db *sqlx.DB) (*sqlx.Tx, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return tx, nil
}

// RollbackTx is a helper function to rollback a transaction.
func RollbackTx(tx *sqlx.Tx) {
	if err := tx.Rollback(); err != nil {
		// Log the error, but don't return it as it's typically called in a defer
		// and the original error from the transaction operation is more important.
		fmt.Printf("Error rolling back transaction: %v\n", err)
	}
}

// CommitTx is a helper function to commit a transaction.
func CommitTx(tx *sqlx.Tx) error {
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
