// internal/repository/transaction_repo.go
package repository

import (
	"context"

	"finflow-wallet/internal/domain"
)

// TransactionRepository defines the interface for transaction data operations.
type TransactionRepository interface {
	CreateTransaction(ctx context.Context, q DBExecutor, tx *domain.Transaction) error
	// Modified: GetTransactionsByWalletID now returns total count
	GetTransactionsByWalletID(ctx context.Context, q DBExecutor, walletID int64, limit, offset int) ([]domain.Transaction, int64, error)
}
