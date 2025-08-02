// internal/repository/user_repo.go
package repository

import (
	"context"

	"finflow-wallet/internal/domain"
)

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	// CreateUser adds a new user to the database using the provided DBExecutor.
	CreateUser(ctx context.Context, q DBExecutor, user *domain.User) error
	// GetUserByID retrieves a user by their ID using the provided DBExecutor.
	GetUserByID(ctx context.Context, q DBExecutor, id int64) (*domain.User, error)
	// GetUserByUsername retrieves a user by their username using the provided DBExecutor.
	GetUserByUsername(ctx context.Context, q DBExecutor, username string) (*domain.User, error)
}
