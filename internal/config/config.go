// internal/config/config.go
package config

import (
	"fmt"
	"os"
	"strconv"

	"finflow-wallet/pkg/db" // Import db package for its Config struct
)

// AppConfig holds all application-wide configurations.
type AppConfig struct {
	ServerPort string
	DB         db.Config
}

// LoadConfig loads configuration from environment variables.
// It returns an AppConfig instance or an error if any required variable is missing or invalid.
func LoadConfig() (*AppConfig, error) {
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort == "" {
		serverPort = "8080" // Default port
	}

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost" // Default to localhost for local development
	}
	dbPortStr := os.Getenv("DB_PORT")
	if dbPortStr == "" {
		dbPortStr = "5432" // Default PostgreSQL port
	}
	dbPort, err := strconv.Atoi(dbPortStr)
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %w", err)
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "user" // Default user for local development
	}
	dbPassword := os.Getenv("DB_PASSWORD")
	if dbPassword == "" {
		dbPassword = "password" // Default password for local development
	}
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "walletdb" // Default database name for local development
	}
	dbSSLMode := os.Getenv("DB_SSLMODE")
	if dbSSLMode == "" {
		dbSSLMode = "disable" // Default to disable for local development
	}

	return &AppConfig{
		ServerPort: serverPort,
		DB: db.Config{
			Host:     dbHost,
			Port:     dbPort,
			User:     dbUser,
			Password: dbPassword,
			DBName:   dbName,
			SSLMode:  dbSSLMode,
		},
	}, nil
}
