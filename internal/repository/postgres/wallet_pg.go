// internal/repository/postgres/wallet_pg.go
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"finflow-wallet/internal/domain"
	"finflow-wallet/internal/repository"
	"finflow-wallet/internal/util"

	"github.com/jmoiron/sqlx"
	"github.com/shopspring/decimal"
)

// WalletRepository implements repository.WalletRepository for PostgreSQL.
type WalletRepository struct {
	// No longer holds *sqlx.DB as methods receive DBExecutor directly
}

// NewWalletRepository creates a new WalletRepository.
func NewWalletRepository(db *sqlx.DB) repository.WalletRepository {
	return &WalletRepository{}
}

// CreateWallet inserts a new wallet into the database using the provided DBExecutor.
func (r *WalletRepository) CreateWallet(ctx context.Context, q repository.DBExecutor, wallet *domain.Wallet) error {
	query := `INSERT INTO wallets (user_id, currency, balance, created_at, updated_at)
              VALUES ($1, $2, $3, $4, $5) RETURNING id`
	err := q.QueryRowContext(ctx, query, wallet.UserID, wallet.Currency, wallet.Balance, wallet.CreatedAt, wallet.UpdatedAt).Scan(&wallet.ID)
	if err != nil {
		return fmt.Errorf("failed to create wallet: %w", err)
	}
	return nil
}

// GetWalletByID retrieves a wallet by its ID using the provided DBExecutor.
func (r *WalletRepository) GetWalletByID(ctx context.Context, q repository.DBExecutor, id int64) (*domain.Wallet, error) {
	var wallet domain.Wallet
	query := `SELECT id, user_id, currency, balance, created_at, updated_at FROM wallets WHERE id = $1`
	err := q.GetContext(ctx, &wallet, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get wallet by ID %d: %w", id, err)
	}
	return &wallet, nil
}

// GetWalletByUserIDAndCurrency retrieves a wallet by user ID and currency using the provided DBExecutor.
func (r *WalletRepository) GetWalletByUserIDAndCurrency(ctx context.Context, q repository.DBExecutor, userID int64, currency string) (*domain.Wallet, error) {
	var wallet domain.Wallet
	query := `SELECT id, user_id, currency, balance, created_at, updated_at FROM wallets WHERE user_id = $1 AND currency = $2`
	err := q.GetContext(ctx, &wallet, query, userID, currency)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get wallet by user ID %d and currency %s: %w", userID, currency, err)
	}
	return &wallet, nil
}

// UpdateWalletBalance updates the balance of a specific wallet using the provided DBExecutor.
func (r *WalletRepository) UpdateWalletBalance(ctx context.Context, q repository.DBExecutor, walletID int64, amount decimal.Decimal) error {
	query := `UPDATE wallets SET balance = balance + $1, updated_at = $2 WHERE id = $3`
	result, err := q.ExecContext(ctx, query, amount, time.Now().UTC(), walletID)
	if err != nil {
		return fmt.Errorf("failed to update wallet balance for ID %d: %w", walletID, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected after updating wallet balance for ID %d: %w", walletID, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no rows affected when updating wallet balance for ID %d, wallet might not exist", walletID)
	}
	return nil
}
