// internal/repository/postgres/user_pg.go
package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"finflow-wallet/internal/domain"
	"finflow-wallet/internal/repository"
	"finflow-wallet/internal/util"

	"github.com/jmoiron/sqlx"
)

// UserRepository implements repository.UserRepository for PostgreSQL.
// Renamed from PostgresUserRepository to UserRepository to avoid stuttering.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository.
// Renamed from NewPostgresUserRepository to NewUserRepository.
func NewUserRepository(db *sqlx.DB) repository.UserRepository {
	return &UserRepository{db: db}
}

// CreateUser inserts a new user into the database.
func (r *UserRepository) CreateUser(ctx context.Context, user *domain.User) error {
	query := `INSERT INTO users (username, created_at, updated_at)
              VALUES ($1, $2, $3) RETURNING id`
	err := r.db.QueryRowContext(ctx, query, user.Username, user.CreatedAt, user.UpdatedAt).Scan(&user.ID)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

// GetUserByID retrieves a user by their ID.
func (r *UserRepository) GetUserByID(ctx context.Context, id int64) (*domain.User, error) {
	var user domain.User
	query := `SELECT id, username, created_at, updated_at FROM users WHERE id = $1`
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by ID %d: %w", id, err)
	}
	return &user, nil
}

// GetUserByUsername retrieves a user by their username.
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User
	query := `SELECT id, username, created_at, updated_at FROM users WHERE username = $1`
	err := r.db.GetContext(ctx, &user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, util.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by username '%s': %w", username, err)
	}
	return &user, nil
}
