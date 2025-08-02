// internal/repository/transaction_repo.go
package repository

import (
	"context"

	"finflow-wallet/internal/domain"

	"github.com/jmoiron/sqlx"
)

// TransactionRepository defines the interface for transaction data operations.
type TransactionRepository interface {
	// CreateTransaction adds a new transaction record to the database.
	// It takes an optional sqlx.Tx for transactional operations.
	CreateTransaction(ctx context.Context, q sqlx.ExtContext, transaction *domain.Transaction) error
	// GetTransactionsByWalletID retrieves transaction history for a specific wallet.
	GetTransactionsByWalletID(ctx context.Context, walletID int64, limit, offset int) ([]domain.Transaction, error)
}
