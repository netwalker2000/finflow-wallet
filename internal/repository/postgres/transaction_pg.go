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
// Renamed from PostgresTransactionRepository to TransactionRepository to avoid stuttering.
type TransactionRepository struct {
	db *sqlx.DB
}

// NewTransactionRepository creates a new TransactionRepository.
// Renamed from NewPostgresTransactionRepository to NewTransactionRepository.
func NewTransactionRepository(db *sqlx.DB) repository.TransactionRepository {
	return &TransactionRepository{db: db}
}

// CreateTransaction inserts a new transaction record into the database.
// It takes an optional sqlx.ExtContext (either *sqlx.DB or *sqlx.Tx) for transactional operations.
func (r *TransactionRepository) CreateTransaction(ctx context.Context, q sqlx.ExtContext, transaction *domain.Transaction) error {
	query := `INSERT INTO transactions (from_wallet_id, to_wallet_id, amount, currency, type, status, transaction_time, description, created_at)
              VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`

	err := q.QueryRowxContext(ctx, query,
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

// GetTransactionsByWalletID retrieves transaction history for a specific wallet.
func (r *TransactionRepository) GetTransactionsByWalletID(ctx context.Context, walletID int64, limit, offset int) ([]domain.Transaction, error) {
	var transactions []domain.Transaction
	query := `SELECT id, from_wallet_id, to_wallet_id, amount, currency, type, status, transaction_time, description, created_at
              FROM transactions
              WHERE from_wallet_id = $1 OR to_wallet_id = $1
              ORDER BY transaction_time DESC
              LIMIT $2 OFFSET $3`
	err := r.db.SelectContext(ctx, &transactions, query, walletID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for wallet ID %d: %w", walletID, err)
	}
	return transactions, nil
}
