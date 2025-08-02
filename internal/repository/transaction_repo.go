// internal/repository/transaction_repo.go
package repository

import (
	"context"

	"finflow-wallet/internal/domain"
)

// TransactionRepository defines the interface for transaction data operations.
type TransactionRepository interface {
	// CreateTransaction adds a new transaction record to the database using the provided DBExecutor.
	CreateTransaction(ctx context.Context, q DBExecutor, transaction *domain.Transaction) error
	// GetTransactionsByWalletID retrieves transaction history for a specific wallet using the provided DBExecutor.
	GetTransactionsByWalletID(ctx context.Context, q DBExecutor, walletID int64, limit, offset int) ([]domain.Transaction, error)
}
