# Load .env if present — exports all variables to sub-processes
-include .env
export

.PHONY: docker-up docker-down migrate-up migrate-down \
        run-backend run-frontend test test-coverage lint seed

## Infrastructure
docker-up:
	docker compose up -d postgres redis

docker-down:
	docker compose down

## Migrations (requires goose: go install github.com/pressly/goose/v3/cmd/goose@latest)
migrate-up:
	PATH=$$HOME/go/bin:$$PATH goose -dir migrations postgres "$(DATABASE_URL)" up

migrate-down:
	PATH=$$HOME/go/bin:$$PATH goose -dir migrations postgres "$(DATABASE_URL)" down

## Development servers
run-backend:
	cd backend && PATH=$$HOME/go/bin:$$PATH air

# PORT is exported from .env (backend uses it); unset it so Next.js defaults to 3000.
run-frontend:
	cd frontend && env -u PORT npm run dev

## Testing
test:
	cd backend && go test ./... && cd ../frontend && npm test -- --passWithNoTests

test-coverage:
	cd backend && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: backend/coverage.html"

## Dev seed (local DB only — never run against production)
seed:
	cd backend && go run ./cmd/seed

## Linting
lint:
	cd backend && golangci-lint run ./...
	cd frontend && npx tsc --noEmit && npx eslint .
