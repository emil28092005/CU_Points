package users

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when the requested user does not exist.
var ErrNotFound = errors.New("not found")

// Repository handles all database access for the users domain.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new users Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetByID fetches a user's profile by primary key.
// Returns ErrNotFound if no user exists with that ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Profile, error) {
	var p Profile
	err := r.db.QueryRow(ctx,
		`SELECT id, email, name, COALESCE(student_id, ''), balance
		 FROM users WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Email, &p.Name, &p.StudentID, &p.Balance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}
	return &p, nil
}

// UpdateBalance adds delta to the user's balance within an existing pgx transaction.
// delta is positive when earning points, negative when spending.
// Returns the new balance after the update via RETURNING, so the service can include
// it in the API response without a second query.
// The database CHECK (balance >= 0) acts as the last line of defense against overdrafts;
// this function will return an error if the constraint fires.
func (r *Repository) UpdateBalance(ctx context.Context, tx pgx.Tx, id string, delta int) (int, error) {
	var newBalance int
	err := tx.QueryRow(ctx,
		`UPDATE users SET balance = balance + $1 WHERE id = $2 RETURNING balance`,
		delta, id,
	).Scan(&newBalance)
	if err != nil {
		return 0, fmt.Errorf("repository.UpdateBalance: %w", err)
	}
	return newBalance, nil
}

// CountTransactions returns the total number of transactions for the given user.
// Used alongside ListTransactions to populate pagination metadata.
func (r *Repository) CountTransactions(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM transactions WHERE user_id = $1`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("repository.CountTransactions: %w", err)
	}
	return count, nil
}

// ListTransactions returns paginated transactions for the given user, ordered newest first.
func (r *Repository) ListTransactions(ctx context.Context, userID string, limit, offset int) ([]Transaction, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id,
		       amount,
		       type,
		       COALESCE(description, ''),
		       COALESCE(partner_id::text, ''),
		       created_at
		FROM transactions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.ListTransactions: %w", err)
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.Amount, &t.Type, &t.Description, &t.PartnerID, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("repository.ListTransactions: scan: %w", err)
		}
		txs = append(txs, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.ListTransactions: rows: %w", err)
	}
	return txs, nil
}
