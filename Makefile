# Homegrown Academy — Development Commands

.PHONY: default dev docker-up docker-down check lint test type-check \
        migrate db-reset openapi generate-types full-generate audit

# Default: run all quality gates
default: check

# ─── Development ─────────────────────────────────────────────────────

# Start the Go API server
dev:
	go run ./cmd/server

# Start all infrastructure services
docker-up:
	docker compose up -d

# Stop all infrastructure services
docker-down:
	docker compose down

# ─── Quality Gates ────────────────────────────────────────────────────

# Run all quality gates (must pass before every commit)
check: lint test type-check

# Run Go linter (zero warnings required)
lint:
	golangci-lint run

# Run Go tests
test:
	go test ./...

# Run TypeScript type checker (zero errors required)
type-check:
	cd frontend && npm run type-check

# ─── Database ─────────────────────────────────────────────────────────

# Run pending database migrations (goose)
migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up

# Reset the database (drop + recreate + migrate)
db-reset:
	docker compose exec postgres psql -U homegrown -c "DROP DATABASE IF EXISTS homegrown;"
	docker compose exec postgres psql -U homegrown -c "CREATE DATABASE homegrown;"
	$(MAKE) migrate

# ─── Code Generation ─────────────────────────────────────────────────

# Generate OpenAPI spec from Go annotations
openapi:
	swag init -g cmd/server/main.go -o openapi/

# Generate TypeScript types from OpenAPI spec
generate-types:
	cd frontend && npm run generate-types

# Full generation pipeline: openapi → TS types
full-generate:
	$(MAKE) openapi
	$(MAKE) generate-types

# ─── Security ─────────────────────────────────────────────────────────

# Run vulnerability check
audit:
	govulncheck ./...
