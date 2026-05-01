# Load .env if present — exports all variables to sub-processes
-include .env
export

# Ensure Go and user Go binaries are always on PATH (needed on servers where
# /etc/profile.d is only sourced for login shells, not by make).
export PATH := /usr/local/go/bin:$(HOME)/go/bin:$(PATH)

.PHONY: install build \
        docker-up docker-down migrate-up migrate-down \
        run-backend run-frontend \
        dev-backend dev-frontend \
        test test-coverage lint seed

## ── Setup ────────────────────────────────────────────────────────────────────

# Install all dependencies: Go CLI tools, Go modules, Node packages.
install:
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/air-verse/air@latest
	cd backend && go mod download
	cd frontend && npm ci

# Build the Go binary and the Next.js production bundle.
build:
	mkdir -p backend/bin
	cd backend && go build -o bin/api ./cmd/api
	cd frontend && npm run build

## ── Infrastructure ───────────────────────────────────────────────────────────

docker-up:
	docker compose up -d postgres redis

docker-down:
	docker compose down

migrate-up:
	goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	goose -dir migrations postgres "$(DATABASE_URL)" down

## ── Production servers (require `make build` first) ─────────────────────────

# Run the compiled Go binary.
run-backend:
	cd backend && ./bin/api

# Run the Next.js production server on port 3001.
run-frontend:
	cd frontend && env -u PORT npx next start -p 3001

## ── Development servers (hot reload) ────────────────────────────────────────

dev-backend:
	cd backend && air

dev-frontend:
	cd frontend && env -u PORT npm run dev

## ── Testing ──────────────────────────────────────────────────────────────────

test:
	cd backend && go test ./... && cd ../frontend && npm test -- --passWithNoTests

test-coverage:
	cd backend && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: backend/coverage.html"

## ── Dev seed (local DB only — never run against production) ──────────────────

seed:
	cd backend && go run ./cmd/seed

## ── Linting ──────────────────────────────────────────────────────────────────

lint:
	cd backend && golangci-lint run ./...
	cd frontend && npx tsc --noEmit && npx eslint .
