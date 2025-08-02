// internal/domain/transaction.go
package domain

import (
	"time"

	"github.com/shopspring/decimal" // For precise monetary calculations
)

// TransactionType defines the type of a financial transaction.
type TransactionType string

const (
	TransactionTypeDeposit    TransactionType = "DEPOSIT"
	TransactionTypeWithdrawal TransactionType = "WITHDRAWAL"
	TransactionTypeTransfer   TransactionType = "TRANSFER"
)

// TransactionStatus defines the status of a financial transaction.
type TransactionStatus string

const (
	TransactionStatusCompleted TransactionStatus = "COMPLETED"
	TransactionStatusPending   TransactionStatus = "PENDING"
	TransactionStatusFailed    TransactionStatus = "FAILED"
)

// Transaction represents a financial transaction record.
type Transaction struct {
	ID              int64             `db:"id" json:"id"`                             // Primary key, BIGSERIAL in DB
	FromWalletID    *int64            `db:"from_wallet_id" json:"from_wallet_id"`     // Source wallet ID (nullable for deposits)
	ToWalletID      *int64            `db:"to_wallet_id" json:"to_wallet_id"`         // Destination wallet ID (nullable for withdrawals)
	Amount          decimal.Decimal   `db:"amount" json:"amount"`                     // Transaction amount, NUMERIC(20, 4) in DB
	Currency        string            `db:"currency" json:"currency"`                 // Currency of the transaction
	Type            TransactionType   `db:"type" json:"type"`                         // Type of transaction (DEPOSIT, WITHDRAWAL, TRANSFER)
	Status          TransactionStatus `db:"status" json:"status"`                     // Status of the transaction (COMPLETED, PENDING, FAILED)
	TransactionTime time.Time         `db:"transaction_time" json:"transaction_time"` // Actual time of the transaction
	Description     *string           `db:"description" json:"description"`           // Optional description
	CreatedAt       time.Time         `db:"created_at" json:"created_at"`             // Timestamp of record creation
}

// NewTransaction creates a new Transaction instance.
func NewTransaction(
	fromWalletID *int64,
	toWalletID *int64,
	amount decimal.Decimal,
	currency string,
	txType TransactionType,
	description *string,
) *Transaction {
	now := time.Now().UTC()
	return &Transaction{
		FromWalletID:    fromWalletID,
		ToWalletID:      toWalletID,
		Amount:          amount,
		Currency:        currency,
		Type:            txType,
		Status:          TransactionStatusCompleted, // Default to completed for simplicity in this assignment
		TransactionTime: now,
		Description:     description,
		CreatedAt:       now,
	}
}
