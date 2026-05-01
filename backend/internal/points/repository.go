package points

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Repository defines the database operations needed by the points service.
// Using an interface allows the service to be unit-tested with a mock implementation.
type Repository interface {
	// GetBalance fetches the current balance for the given user.
	GetBalance(ctx context.Context, userID string) (int, error)
	// EarnAtomic credits amount to the user's balance and inserts a transaction row
	// in a single DB transaction. txType must be "earn" or "admin_grant".
	EarnAtomic(ctx context.Context, userID string, amount int, txType, description string) error
	// SpendAtomic debits amount from user balance and inserts a spend transaction
	// in a single database transaction. Returns the new balance on success.
	// The DB CHECK (balance >= 0) is the authoritative guard; the service also
	// pre-checks to return ErrInsufficientBalance early.
	SpendAtomic(ctx context.Context, userID, partnerID string, amount int) (int, error)
}

// CacheClient defines the Redis operations needed by the points service.
type CacheClient interface {
	// IsQRUsed returns true if the given jti has already been redeemed.
	IsQRUsed(ctx context.Context, jti string) (bool, error)
	// MarkQRUsed records the jti as used with a TTL of 5 minutes.
	MarkQRUsed(ctx context.Context, jti string) error
}

// pgRepository is the PostgreSQL-backed implementation of Repository.
type pgRepository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new PostgreSQL-backed points Repository.
func NewRepository(db *pgxpool.Pool) Repository {
	return &pgRepository{db: db}
}

// GetBalance returns the current point balance for the given user.
func (r *pgRepository) GetBalance(ctx context.Context, userID string) (int, error) {
	var balance int
	err := r.db.QueryRow(ctx,
		`SELECT balance FROM users WHERE id = $1`,
		userID,
	).Scan(&balance)
	if err != nil {
		return 0, fmt.Errorf("repository.GetBalance: %w", err)
	}
	return balance, nil
}

// EarnAtomic credits amount to the user's balance and records a transaction,
// all within a single DB transaction.
// txType must be a value accepted by the transactions.type CHECK constraint ("earn" or "admin_grant").
func (r *pgRepository) EarnAtomic(ctx context.Context, userID string, amount int, txType, description string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("repository.EarnAtomic: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx,
		`UPDATE users SET balance = balance + $1 WHERE id = $2`,
		amount, userID,
	)
	if err != nil {
		return fmt.Errorf("repository.EarnAtomic: update balance: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (user_id, amount, type, description)
		 VALUES ($1, $2, $3, $4)`,
		userID, amount, txType, description,
	)
	if err != nil {
		return fmt.Errorf("repository.EarnAtomic: insert transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("repository.EarnAtomic: commit: %w", err)
	}
	return nil
}

// SpendAtomic deducts amount from the student's balance and records a spend
// transaction, all within a single DB transaction.
// The negative amount stored in transactions follows the ledger convention:
// positive = earn, negative = spend.
func (r *pgRepository) SpendAtomic(ctx context.Context, userID, partnerID string, amount int) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("repository.SpendAtomic: begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var newBalance int
	err = tx.QueryRow(ctx,
		`UPDATE users SET balance = balance - $1 WHERE id = $2 RETURNING balance`,
		amount, userID,
	).Scan(&newBalance)
	if err != nil {
		return 0, fmt.Errorf("repository.SpendAtomic: update balance: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO transactions (user_id, partner_id, amount, type)
		 VALUES ($1, $2, $3, 'spend')`,
		userID, partnerID, -amount,
	)
	if err != nil {
		return 0, fmt.Errorf("repository.SpendAtomic: insert transaction: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("repository.SpendAtomic: commit: %w", err)
	}
	return newBalance, nil
}

// redisCache is the Redis-backed implementation of CacheClient.
type redisCache struct {
	client *redis.Client
}

// NewRedisCache creates a Redis-backed CacheClient for QR token one-time-use tracking.
func NewRedisCache(client *redis.Client) CacheClient {
	return &redisCache{client: client}
}

// IsQRUsed returns true if the given jti key already exists in Redis.
func (c *redisCache) IsQRUsed(ctx context.Context, jti string) (bool, error) {
	count, err := c.client.Exists(ctx, "used_qr:"+jti).Result()
	return count > 0, err
}

// MarkQRUsed sets used_qr:<jti> = "1" with a 5-minute TTL.
// TTL matches the QR token expiry so the key is automatically cleaned up.
func (c *redisCache) MarkQRUsed(ctx context.Context, jti string) error {
	return c.client.Set(ctx, "used_qr:"+jti, "1", 5*time.Minute).Err()
}
