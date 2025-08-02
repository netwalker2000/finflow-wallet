// internal/repository/wallet_repo.go
package repository

import (
	"context"

	"finflow-wallet/internal/domain"

	"github.com/shopspring/decimal"
)

// WalletRepository defines the interface for wallet data operations.
type WalletRepository interface {
	// CreateWallet adds a new wallet to the database using the provided DBExecutor.
	CreateWallet(ctx context.Context, q DBExecutor, wallet *domain.Wallet) error
	// GetWalletByID retrieves a wallet by its ID using the provided DBExecutor.
	GetWalletByID(ctx context.Context, q DBExecutor, id int64) (*domain.Wallet, error)
	// GetWalletByUserIDAndCurrency retrieves a wallet by user ID and currency using the provided DBExecutor.
	GetWalletByUserIDAndCurrency(ctx context.Context, q DBExecutor, userID int64, currency string) (*domain.Wallet, error)
	// UpdateWalletBalance updates the balance of a specific wallet using the provided DBExecutor.
	UpdateWalletBalance(ctx context.Context, q DBExecutor, walletID int64, amount decimal.Decimal) error
}
