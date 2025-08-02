// internal/domain/wallet.go
package domain

import (
	"time"

	"github.com/shopspring/decimal" // For precise monetary calculations
)

// Wallet represents a user's wallet.
type Wallet struct {
	ID        int64           `db:"id" json:"id"`                 // Primary key, BIGSERIAL in DB
	UserID    int64           `db:"user_id" json:"user_id"`       // Foreign key to User
	Currency  string          `db:"currency" json:"currency"`     // e.g., "USD", "FIAT"
	Balance   decimal.Decimal `db:"balance" json:"balance"`       // Current balance, NUMERIC(20, 4) in DB
	CreatedAt time.Time       `db:"created_at" json:"created_at"` // Timestamp of creation
	UpdatedAt time.Time       `db:"updated_at" json:"updated_at"` // Timestamp of last update
}

// NewWallet creates a new Wallet instance.
func NewWallet(userID int64, currency string) *Wallet {
	now := time.Now().UTC()
	return &Wallet{
		UserID:    userID,
		Currency:  currency,
		Balance:   decimal.Zero, // Initialize balance to 0
		CreatedAt: now,
		UpdatedAt: now,
	}
}
