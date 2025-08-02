// internal/domain/user.go
package domain

import "time"

// User represents a user in the wallet system.
type User struct {
	ID        int64     `db:"id" json:"id"`                 // Primary key, BIGSERIAL in DB
	Username  string    `db:"username" json:"username"`     // Unique username
	CreatedAt time.Time `db:"created_at" json:"created_at"` // Timestamp of creation
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"` // Timestamp of last update
}

// NewUser creates a new User instance.
func NewUser(username string) *User {
	now := time.Now().UTC()
	return &User{
		Username:  username,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
