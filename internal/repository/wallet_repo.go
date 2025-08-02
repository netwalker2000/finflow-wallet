// internal/repository/wallet_repo.go
package repository

import (
	"context"

	"finflow-wallet/internal/domain"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// WalletRepository defines the interface for wallet data operations.
type WalletRepository interface {
	// CreateWallet adds a new wallet to the database.
	CreateWallet(ctx context.Context, wallet *domain.Wallet) error
	// GetWalletByID retrieves a wallet by its ID.
	GetWalletByID(ctx context.Context, id int64) (*domain.Wallet, error)
	// GetWalletByUserIDAndCurrency retrieves a wallet by user ID and currency.
	GetWalletByUserIDAndCurrency(ctx context.Context, userID int64, currency string) (*domain.Wallet, error)
	// UpdateWalletBalance updates the balance of a specific wallet.
	// It takes an optional sqlx.Tx for transactional operations.
	UpdateWalletBalance(ctx context.Context, q sqlx.ExtContext, walletID int64, amount decimal.Decimal) error
}
