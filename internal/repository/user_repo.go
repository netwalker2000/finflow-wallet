// internal/repository/user_repo.go
package repository

import (
	"context"

	"finflow-wallet/internal/domain"
)

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	// CreateUser adds a new user to the database.
	CreateUser(ctx context.Context, user *domain.User) error
	// GetUserByID retrieves a user by their ID.
	GetUserByID(ctx context.Context, id int64) (*domain.User, error)
	// GetUserByUsername retrieves a user by their username.
	GetUserByUsername(ctx context.Context, username string) (*domain.User, error)
}
