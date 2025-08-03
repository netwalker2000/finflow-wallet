// internal/app.go
package app

import (
	"context"
	router "finflow-wallet/internal/api"
	"finflow-wallet/internal/api/handler"
	"finflow-wallet/internal/repository"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jmoiron/sqlx"

	"finflow-wallet/internal/config"
	"finflow-wallet/internal/repository/postgres"
	"finflow-wallet/internal/service"
	"finflow-wallet/internal/util"
	"finflow-wallet/pkg/db"
)

// Application holds all the initialized components of the application.
type Application struct {
	Config *config.AppConfig
	Logger *slog.Logger
	DB     *sqlx.DB

	// Repositories
	UserRepository        repository.UserRepository
	WalletRepository      repository.WalletRepository
	TransactionRepository repository.TransactionRepository

	// Services
	WalletService service.WalletService

	// HTTP API
	HTTPHandler http.Handler
}

// NewApplication creates a new Application instance.
func NewApplication() *Application {
	return &Application{}
}

// Initialize initializes all application components.
func (app *Application) Initialize(ctx context.Context) error {
	// 1. Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}
	app.Config = cfg

	// 2. Initialize Logger
	util.InitLogger()
	app.Logger = util.GetLogger()
	app.Logger.Info("Application configuration loaded successfully.")

	// 3. Connect to Database
	database, err := db.NewPostgresDB(app.Config.DB)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	app.DB = database
	app.Logger.Info("Database connection established.")

	// 4. Initialize Repositories
	app.UserRepository = postgres.NewUserRepository(app.DB)
	app.WalletRepository = postgres.NewWalletRepository(app.DB)
	app.TransactionRepository = postgres.NewTransactionRepository(app.DB)
	app.Logger.Info("Repositories initialized.")

	// 5. Initialize Services
	// Pass the concrete db.BeginTx, db.CommitTx, db.RollbackTx functions from pkg/db
	app.WalletService = service.NewWalletService(
		app.DB, // This is the DBTxBeginner
		app.DB, // This is the DBExecutor
		app.UserRepository,
		app.WalletRepository,
		app.TransactionRepository,
		db.BeginTx,
		db.CommitTx,
		db.RollbackTx,
	)
	app.Logger.Info("Services initialized.")

	// 6. Initialize HTTP Handlers and Router
	walletHandler := handler.NewWalletHandler(app.WalletService, app.Logger)
	app.HTTPHandler = router.NewRouter(walletHandler, app.Logger)
	app.Logger.Info("HTTP router and handlers initialized.")

	return nil
}

// Shutdown gracefully shuts down application resources.
func (app *Application) Shutdown(ctx context.Context) error {
	app.Logger.Info("Shutting down application...")
	if app.DB != nil {
		if err := app.DB.Close(); err != nil {
			app.Logger.Error("Failed to close database connection", "error", err)
			return fmt.Errorf("failed to close database connection: %w", err)
		}
		app.Logger.Info("Database connection closed.")
	}
	app.Logger.Info("Application shut down gracefully.")
	return nil
}
