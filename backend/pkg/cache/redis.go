// Package cache provides Redis client initialization.
package cache

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// NewClient creates and validates a Redis client using the given REDIS_URL.
// Returns an error if the URL cannot be parsed or if the initial PING fails.
func NewClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("cache.NewClient: parse URL: %w", err)
	}
	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("cache.NewClient: ping: %w", err)
	}
	return client, nil
}
