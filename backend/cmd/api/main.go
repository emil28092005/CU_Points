package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/cu-points/backend/internal/admin"
	"github.com/cu-points/backend/internal/auth"
	"github.com/cu-points/backend/internal/config"
	"github.com/cu-points/backend/internal/middleware"
	"github.com/cu-points/backend/internal/partners"
	"github.com/cu-points/backend/internal/points"
	"github.com/cu-points/backend/internal/users"
	cachepkg "github.com/cu-points/backend/pkg/cache"
	dbpkg "github.com/cu-points/backend/pkg/db"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := dbpkg.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	redisClient, err := cachepkg.NewClient(ctx, cfg.RedisURL)
	if err != nil {
		slog.Error("redis connect failed", "err", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	slog.Info("infrastructure connected", "postgres", cfg.DatabaseURL, "redis", cfg.RedisURL)

	// ── Auth ──────────────────────────────────────────────────────────────────
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, jwtManager)
	authHandler := auth.NewHandler(authSvc)

	// ── Users ─────────────────────────────────────────────────────────────────
	usersRepo := users.NewRepository(db)
	usersSvc := users.NewService(usersRepo)
	usersHandler := users.NewHandler(usersSvc)

	// ── Partners ──────────────────────────────────────────────────────────────
	partnersRepo := partners.NewRepository(db)
	partnersSvc := partners.NewService(partnersRepo)
	partnersHandler := partners.NewHandler(partnersSvc)

	// ── Points ────────────────────────────────────────────────────────────────
	pointsRepo := points.NewRepository(db)
	pointsCache := points.NewRedisCache(redisClient)
	// pointsSvc is also passed to adminSvc so GrantPoints reuses EarnPoints logic.
	pointsSvc := points.NewService(pointsRepo, pointsCache, cfg.JWTSecret)
	pointsHandler := points.NewHandler(pointsSvc)

	// ── Admin ─────────────────────────────────────────────────────────────────
	adminSvc := admin.NewService(db, pointsSvc)
	adminHandler := admin.NewHandler(adminSvc)

	// ── Router ────────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(middleware.CORS)
	r.Use(middleware.Logger)

	r.Route("/api/v1", func(r chi.Router) {
		// Public — no authentication required.
		r.Post("/auth/login", authHandler.Login)
		r.Post("/auth/refresh", authHandler.Refresh)
		r.Get("/partners", partnersHandler.List)

		// Any authenticated user — profile endpoint used by the login flow for all roles.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))

			r.Get("/me", usersHandler.Me)
		})

		// Student-only endpoints.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Use(middleware.RequireRole("student"))

			r.Get("/me/transactions", usersHandler.Transactions)
			r.Get("/me/qr", pointsHandler.GenerateQR)
		})

		// Partner-only endpoints.
		// Rate-limited to 10 spend requests per minute per partner to prevent abuse.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Use(middleware.RequireRole("partner"))
			r.Use(middleware.SpendRateLimit(redisClient, 10))

			r.Post("/partner/spend", pointsHandler.Spend)
		})

		// Admin-only endpoints.
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Use(middleware.RequireRole("admin"))

			r.Post("/admin/points/grant", adminHandler.GrantPoints)
			r.Get("/admin/transactions", adminHandler.ListTransactions)
			r.Get("/admin/users", adminHandler.ListUsers)
			r.Get("/admin/stats", adminHandler.Stats)
		})
	})

	// Health check — no auth required.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	addr := ":" + cfg.Port
	slog.Info("server starting", "addr", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
