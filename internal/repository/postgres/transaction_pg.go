// internal/repository/postgres/transaction_pg.go
package postgres

import (
	"context"
	"fmt"

	"finflow-wallet/internal/domain"
	"finflow-wallet/internal/repository"

	"github.com/jmoiron/sqlx"
)

// TransactionRepository implements repository.TransactionRepository for PostgreSQL.
type TransactionRepository struct {
	// No longer holds *sqlx.DB as methods receive DBExecutor directly
}

// NewTransactionRepository creates a new TransactionRepository.
func NewTransactionRepository(db *sqlx.DB) repository.TransactionRepository {
	return &TransactionRepository{}
}

// CreateTransaction inserts a new transaction record into the database using the provided DBExecutor.
func (r *TransactionRepository) CreateTransaction(ctx context.Context, q repository.DBExecutor, transaction *domain.Transaction) error {
	query := `INSERT INTO transactions (from_wallet_id, to_wallet_id, amount, currency, type, status, transaction_time, description, created_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	err := q.QueryRowContext(ctx, query,
		transaction.FromWalletID,
		transaction.ToWalletID,
		transaction.Amount,
		transaction.Currency,
		transaction.Type,
		transaction.Status,
		transaction.TransactionTime,
		transaction.Description,
		transaction.CreatedAt,
	).Scan(&transaction.ID)

	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}
	return nil
}

// GetTransactionsByWalletID retrieves transaction history for a specific wallet using the provided DBExecutor.
func (r *TransactionRepository) GetTransactionsByWalletID(ctx context.Context, q repository.DBExecutor, walletID int64, limit, offset int) ([]domain.Transaction, error) {
	var transactions []domain.Transaction
	query := `SELECT id, from_wallet_id, to_wallet_id, amount, currency, type, status, transaction_time, description, created_at
              FROM transactions
              WHERE from_wallet_id = $1 OR to_wallet_id = $1
              ORDER BY transaction_time DESC
              LIMIT $2 OFFSET $3`
	err := q.SelectContext(ctx, &transactions, query, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for wallet ID %d: %w", walletID, err)
	}
	return transactions, nil
}
