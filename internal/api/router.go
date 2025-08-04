// internal/api/router.go
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"finflow-wallet/internal/api/handler"
)

// NewRouter sets up and returns a new HTTP router.
func NewRouter(walletHandler *handler.WalletHandler, logger *slog.Logger) http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.RequestID)                       // Add a request ID to the context
	r.Use(middleware.RealIP)                          // Use the real IP address
	r.Use(middleware.Logger)                          // Log HTTP requests
	r.Use(middleware.Recoverer)                       // Recover from panics and return 500
	r.Use(middleware.Timeout(handler.DefaultTimeout)) // Set a default timeout for requests (define DefaultTimeout in handler)

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Wallet API routes
	r.Route("/wallets", func(r chi.Router) {
		r.Post("/{walletID}/deposit", walletHandler.Deposit)
		r.Post("/{walletID}/withdraw", walletHandler.Withdraw)
		r.Get("/{walletID}/balance", walletHandler.GetWalletBalance)
		r.Get("/{walletID}/transactions", walletHandler.GetTransactionHistory)
	})

	// Transfer is a separate top-level endpoint as it involves two wallets
	r.Post("/transfers", walletHandler.Transfer)

	return r
}
