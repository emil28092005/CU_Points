package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when the requested user does not exist in the database.
var ErrNotFound = errors.New("not found")

// UserRecord is the minimal user row fetched from the database during authentication.
type UserRecord struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
}

// UserRepository defines the database operations the auth service depends on.
// Defined as an interface so unit tests can inject a mock without a real database.
type UserRepository interface {
	// GetUserByEmail returns the user row for the given email address.
	// Returns ErrNotFound if no user exists with that email.
	GetUserByEmail(ctx context.Context, email string) (*UserRecord, error)

	// GetUserByID returns the user row for the given primary key.
	// Returns ErrNotFound if the user has been deleted since the token was issued.
	GetUserByID(ctx context.Context, id string) (*UserRecord, error)
}

// Repository is the PostgreSQL-backed implementation of UserRepository.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL-backed auth Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetUserByEmail fetches the user row needed for password verification.
// Returns ErrNotFound if no user exists with that email.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*UserRecord, error) {
	var u UserRecord
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role FROM users WHERE email = $1`,
		email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetUserByEmail: %w", err)
	}
	return &u, nil
}

// GetUserByID fetches the user row needed when refreshing a token.
// The role is re-read from the DB so that admin role changes take effect on the next refresh.
// Returns ErrNotFound if the user has been deleted since the token was issued.
func (r *Repository) GetUserByID(ctx context.Context, id string) (*UserRecord, error) {
	var u UserRecord
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, role FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetUserByID: %w", err)
	}
	return &u, nil
}
