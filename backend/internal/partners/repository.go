package partners

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when the requested partner does not exist.
var ErrNotFound = errors.New("partner not found")

// Repository handles database access for the partners domain.
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new partners Repository.
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ListActive fetches all partners where is_active = true, ordered alphabetically.
func (r *Repository) ListActive(ctx context.Context) ([]Partner, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, address, max_spend_pct
		 FROM partners
		 WHERE is_active = true
		 ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("repository.ListActive: %w", err)
	}
	defer rows.Close()

	var ps []Partner
	for rows.Next() {
		var p Partner
		if err := rows.Scan(&p.ID, &p.Name, &p.Address, &p.MaxSpendPct); err != nil {
			return nil, fmt.Errorf("repository.ListActive: scan: %w", err)
		}
		ps = append(ps, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository.ListActive: rows: %w", err)
	}
	return ps, nil
}

// GetByID fetches a single partner by its primary key.
// Returns ErrNotFound if no partner exists with that ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*Partner, error) {
	var p Partner
	err := r.db.QueryRow(ctx,
		`SELECT id, name, address, max_spend_pct
		 FROM partners WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Name, &p.Address, &p.MaxSpendPct)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByID: %w", err)
	}
	return &p, nil
}

// GetByUserID fetches the partner record associated with the given cashier user account.
// Returns ErrNotFound if no partner is linked to that user.
func (r *Repository) GetByUserID(ctx context.Context, userID string) (*Partner, error) {
	var p Partner
	err := r.db.QueryRow(ctx,
		`SELECT id, name, address, max_spend_pct
		 FROM partners WHERE user_id = $1`,
		userID,
	).Scan(&p.ID, &p.Name, &p.Address, &p.MaxSpendPct)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("repository.GetByUserID: %w", err)
	}
	return &p, nil
}
