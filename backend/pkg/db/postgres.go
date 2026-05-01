// Package db provides PostgreSQL connection pool initialization.
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates and validates a pgx connection pool using the given DATABASE_URL.
// Returns an error if the pool cannot be created or if the initial ping fails.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("db.NewPool: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db.NewPool: ping: %w", err)
	}
	return pool, nil
}
