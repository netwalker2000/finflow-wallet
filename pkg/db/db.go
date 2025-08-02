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
