// internal/util/logger.go
package util

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

// InitLogger initializes the global structured logger.
// It sets up a JSON handler for production-like logs.
func InitLogger() {
	// You can customize the handler based on environment (e.g., TextHandler for dev, JSONHandler for prod)
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,           // Add file and line number to logs
		Level:     slog.LevelInfo, // Set default log level
	})
	logger = slog.New(handler)
	slog.SetDefault(logger) // Set as default logger for convenience
}

// GetLogger returns the initialized global logger.
func GetLogger() *slog.Logger {
	if logger == nil {
		InitLogger() // Initialize if not already initialized (should be called explicitly at app start)
	}
	return logger
}
