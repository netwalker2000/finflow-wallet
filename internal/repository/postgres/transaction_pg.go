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

// GetTransactionsByWalletID retrieves a paginated list of transactions for a specific wallet.
// It performs two queries: one for the data and one for the total count.
func (r *TransactionRepository) GetTransactionsByWalletID(ctx context.Context, q repository.DBExecutor, walletID int64, limit, offset int) ([]domain.Transaction, int64, error) {
	transactions := []domain.Transaction{}

	// Query 1: Get the paginated transactions
	// We need to check both from_wallet_id and to_wallet_id for transactions related to this wallet.
	query := `
		SELECT id, from_wallet_id, to_wallet_id, amount, currency, type, status, transaction_time, description, created_at
		FROM transactions
		WHERE from_wallet_id = $1 OR to_wallet_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`
	err := q.SelectContext(ctx, &transactions, query, walletID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch transactions for wallet %d: %w", walletID, err)
	}

	// Query 2: Get the total count of transactions for the wallet
	var totalCount int64
	countQuery := `
		SELECT COUNT(*)
		FROM transactions
		WHERE from_wallet_id = $1 OR to_wallet_id = $1`
	err = q.GetContext(ctx, &totalCount, countQuery, walletID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total transaction count for wallet %d: %w", walletID, err)
	}

	return transactions, totalCount, nil
}
