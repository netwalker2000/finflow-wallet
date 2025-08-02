// internal/util/errors.go
package util

import "errors"

// Common application-specific errors.
var (
	ErrNotFound           = errors.New("resource not found")
	ErrInvalidInput       = errors.New("invalid input provided")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrSameWalletTransfer = errors.New("cannot transfer to the same wallet")
	ErrWalletNotFound     = errors.New("wallet not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrDuplicateEntry     = errors.New("duplicate entry") // For cases like creating a user with existing username
	// Add more specific errors as needed
)
