# Homegrown Academy — Development Commands

# ─── Tool Detection ───────────────────────────────────────────────────
# Detect Go binary — supports Nix environments where go isn't in system PATH.
# Override with: make openapi GO=/path/to/go
GO ?= $(shell command -v go 2>/dev/null || ls /nix/store/*-go-1.*/bin/go 2>/dev/null | tail -1)
# Honor $GOBIN if set (e.g. Nix home-manager sets it to /scratch/cache/go/bin);
# fall back to $GOPATH/bin (the Go toolchain default).
GOBIN := $(or $(shell $(GO) env GOBIN),$(shell $(GO) env GOPATH)/bin)
SWAG := $(GOBIN)/swag

.PHONY: default dev dev-api dev-web docker-up docker-down check lint test type-check \
        migrate db-reset openapi generate-types full-generate audit install-tools install-hooks

# Default: run all quality gates
default: check

# ─── Development ─────────────────────────────────────────────────────

# Start backend (air hot-reload) + frontend (Vite HMR) together
dev: docker-up
	@trap 'kill 0' EXIT; \
	air & \
	cd frontend && npm install --silent && npm run dev & \
	wait

# Start only the Go backend with hot-reload
dev-api: docker-up
	air

# Start only the Vite frontend dev server
dev-web:
	cd frontend && npm install --silent && npm run dev

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
	$(GO) test ./...

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

# Install required build tools
install-tools:
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	$(GO) install github.com/evilmartians/lefthook@latest

# Install lefthook binary and register git hooks (one-shot dev environment setup)
install-hooks: install-tools
	$(GOBIN)/lefthook install

# Generate OpenAPI spec from Go annotations
openapi: install-tools
	$(SWAG) init -g cmd/server/main.go -o openapi/

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
