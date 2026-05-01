// Package config loads all runtime configuration from environment variables.
// All other packages must read settings through this package — no magic strings elsewhere.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all application settings read from environment variables.
type Config struct {
	DatabaseURL   string
	RedisURL      string
	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration
	Port          string
	Env           string
}

// Load reads environment variables and returns a populated Config.
// Returns an error if any required variable is missing or cannot be parsed.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		Port:        envOrDefault("PORT", "8080"),
		Env:         envOrDefault("ENV", "development"),
	}

	for _, req := range []struct{ name, val string }{
		{"DATABASE_URL", cfg.DatabaseURL},
		{"REDIS_URL", cfg.RedisURL},
		{"JWT_SECRET", cfg.JWTSecret},
	} {
		if req.val == "" {
			return nil, fmt.Errorf("config.Load: required env var %s is not set", req.name)
		}
	}

	var err error
	cfg.JWTAccessTTL, err = time.ParseDuration(envOrDefault("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("config.Load: invalid JWT_ACCESS_TTL: %w", err)
	}
	cfg.JWTRefreshTTL, err = time.ParseDuration(envOrDefault("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("config.Load: invalid JWT_REFRESH_TTL: %w", err)
	}

	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
