# Homegrown Academy — Development Commands

# ─── Tool Detection ───────────────────────────────────────────────────
# Detect Go binary — supports Nix environments where go isn't in system PATH.
# Override with: make openapi GO=/path/to/go
GO ?= $(shell command -v go 2>/dev/null || ls /nix/store/*-go-1.*/bin/go 2>/dev/null | tail -1)
# Honor $GOBIN if set (e.g. Nix home-manager sets it to /scratch/cache/go/bin);
# fall back to $GOPATH/bin (the Go toolchain default).
GOBIN := $(or $(shell $(GO) env GOBIN),$(shell $(GO) env GOPATH)/bin)
SWAG  := $(GOBIN)/swag
GOOSE := $(GOBIN)/goose

DATABASE_URL ?= postgres://homegrown:homegrown@localhost:5932/homegrown

.PHONY: default dev dev-api dev-web docker-up docker-down check lint test type-check \
        migrate db-reset seed seed-full agent-db-reset agent-kratos-reset agent-server \
        openapi generate-types full-generate audit install-tools install-hooks

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
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" up

# Reset the database (drop + recreate + migrate)
db-reset:
	docker compose exec postgres psql -U homegrown -d postgres -c "DROP DATABASE IF EXISTS homegrown;"
	docker compose exec postgres psql -U homegrown -d postgres -c "CREATE DATABASE homegrown;"
	$(MAKE) migrate

# ─── Code Generation ─────────────────────────────────────────────────

# Install required build tools
install-tools:
	$(GO) install github.com/swaggo/swag/cmd/swag@latest
	$(GO) install github.com/evilmartians/lefthook@latest
	$(GO) install github.com/pressly/goose/v3/cmd/goose@latest

# Install lefthook binary and register git hooks (one-shot dev environment setup)
install-hooks: install-tools
	npm install
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

# ─── Agent Database ───────────────────────────────────────────────────────────

# Seed the agent database (creates it if absent, runs migrations, seeds data).
# Override target DB: make seed DB=homegrown  (to seed your dev database)
DB ?= homegrown_agent
seed:
	$(GO) run ./cmd/seed/ --db $(DB)

# Full-scale seed: basic seed + 997 additional families with rich data across all domains.
# Creates ~1000 families, social interactions, learning content, marketplace, compliance, etc.
seed-full: seed
	$(GO) run ./cmd/seed-full/ --db $(DB)

# Full agent database reset: drop → recreate → migrate → seed
agent-db-reset:
	docker compose exec postgres psql -U homegrown -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'homegrown_agent' AND pid <> pg_backend_pid();"
	docker compose exec postgres psql -U homegrown -d postgres -c "DROP DATABASE IF EXISTS homegrown_agent;"
	docker compose exec postgres psql -U homegrown -d postgres -c "CREATE DATABASE homegrown_agent;"
	$(MAKE) seed

# Wipe and reinitialise only the agent Kratos identity store.
agent-kratos-reset:
	docker compose exec postgres psql -U homegrown -d postgres -c "DROP DATABASE IF EXISTS kratos_agent;"
	docker compose exec postgres psql -U homegrown -d postgres -c "CREATE DATABASE kratos_agent;"
	docker compose restart kratos_agent

# Start the API server on port 15180 pointed at the agent database.
agent-server:
	DATABASE_URL=postgres://homegrown:homegrown@localhost:5932/homegrown_agent \
	SERVER_PORT=15180 \
	$(GO) run ./cmd/server/

# ─── Security ─────────────────────────────────────────────────────────

# Run vulnerability check
audit:
	govulncheck ./...
