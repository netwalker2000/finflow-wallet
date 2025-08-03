// cmd/api/main.go
package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	app "finflow-wallet/internal"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create and initialize the application
	application := app.NewApplication()
	if err := application.Initialize(ctx); err != nil {
		application.Logger.Error("Failed to initialize application", "error", err)
		os.Exit(1)
	}

	// Start HTTP server
	server := &http.Server{
		Addr:         ":" + application.Config.ServerPort,
		Handler:      application.HTTPHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Run server in a goroutine
	go func() {
		application.Logger.Info("Starting HTTP server", "port", application.Config.ServerPort)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			application.Logger.Error("HTTP server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received

	application.Logger.Info("Shutting down HTTP server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		application.Logger.Error("HTTP server shutdown failed", "error", err)
		os.Exit(1)
	}

	// Perform application-level shutdown (e.g., close DB connections)
	if err := application.Shutdown(shutdownCtx); err != nil {
		application.Logger.Error("Application shutdown failed", "error", err)
		os.Exit(1)
	}

	application.Logger.Info("Application gracefully stopped.")
}
