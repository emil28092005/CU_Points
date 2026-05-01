package points_test

import (
	"testing"
	"time"

	"github.com/cu-points/backend/internal/points"
	"github.com/redis/go-redis/v9"
)

// TestNewRepository_Constructor exercises the factory function without a real DB.
// Method calls on the returned value would panic (nil pool), so we only test creation.
func TestNewRepository_Constructor(t *testing.T) {
	repo := points.NewRepository(nil)
	if repo == nil {
		t.Error("NewRepository(nil) returned nil")
	}
}

// TestNewRedisCache_Constructor exercises the cache factory without a real Redis.
func TestNewRedisCache_Constructor(t *testing.T) {
	// Use a client with an unreachable address; we only test that construction succeeds.
	client := redis.NewClient(&redis.Options{
		Addr:        "localhost:0",
		DialTimeout: time.Millisecond,
	})
	cache := points.NewRedisCache(client)
	if cache == nil {
		t.Error("NewRedisCache returned nil")
	}
}
