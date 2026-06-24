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

.PHONY: default dev dev-api dev-web docker-up docker-down check check-full lint test type-check a11y \
        migrate db-reset seed seed-full agent-seed-full agent-db-reset hearth-bootstrap hearth-reset agent-server \
        openapi generate-types full-generate audit install-tools install-hooks \
        backup restore-drill hs-setup

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

# Run all quality gates plus a11y tests (requires Playwright browsers)
check-full: check a11y

# Run Playwright accessibility tests (requires Vite dev server running)
a11y:
	cd frontend && npx playwright test e2e/a11y.spec.ts

# ─── Database ─────────────────────────────────────────────────────────

# Run pending database migrations (goose)
migrate:
	$(GOOSE) -dir migrations postgres "$(DATABASE_URL)" up

# Reset the DEV database (homegrown). For the agent DB, use agent-db-reset.
db-reset:
	docker compose exec postgres psql -U homegrown -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'homegrown' AND pid <> pg_backend_pid();"
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

# Full-scale seed against the DEV database by default (homegrown).
# Override: make seed-full FULL_DB=homegrown_agent
# Creates ~1000 families, social interactions, learning content, marketplace, compliance, etc.
FULL_DB ?= homegrown
seed-full:
	$(MAKE) seed DB=$(FULL_DB)
	$(GO) run ./cmd/seed-full/ --db $(FULL_DB)

# Full-scale seed for the agent database: runs basic seed first, then seed-full against homegrown_agent.
agent-seed-full:
	$(MAKE) seed
	$(GO) run ./cmd/seed-full/ --db $(DB)

# Full agent reset: drop → recreate app DB → reset Hearth identity store → seed
agent-db-reset:
	docker compose exec postgres psql -U homegrown -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'homegrown_agent' AND pid <> pg_backend_pid();"
	docker compose exec postgres psql -U homegrown -d postgres -c "DROP DATABASE IF EXISTS homegrown_agent;"
	docker compose exec postgres psql -U homegrown -d postgres -c "CREATE DATABASE homegrown_agent;"
	$(MAKE) hearth-reset
	$(MAKE) seed

# Bootstrap the Hearth dev realm via the admin API (idempotent; safe to re-run).
# Hearth's --dev flag auto-bootstraps on first start, but this target lets you
# explicitly (re-)seed the realm config after a hearth-reset.
hearth-bootstrap:
	@echo "Waiting for Hearth admin API..."
	@until wget --spider --quiet http://localhost:4434/health 2>/dev/null; do sleep 2; done
	curl -sf -X POST http://localhost:4434/admin/bootstrap \
	  -H "Content-Type: application/json" \
	  -d '{"realm": "homegrown", "dev": true}'
	@echo ""
	@echo "Hearth realm 'homegrown' bootstrapped."

# Wipe and reinitialise the Hearth identity store (clears embedded DB, re-bootstraps realm).
# Equivalent to the old agent-kratos-reset but for Hearth's embedded storage.
hearth-reset:
	docker compose stop hearth
	docker compose rm -f hearth
	docker volume rm homegrown-academy_hearth_data 2>/dev/null || true
	docker compose up -d hearth
	$(MAKE) hearth-bootstrap

# Start the API server on port 15180 pointed at the agent database.
agent-server:
	DATABASE_URL=postgres://homegrown:homegrown@localhost:5932/homegrown_agent \
	SERVER_PORT=15180 \
	$(GO) run ./cmd/server/

# ─── Hyperswitch Local Setup ─────────────────────────────────────────────────

# Bootstrap Hyperswitch after `docker compose up hyperswitch-server`.
# Creates merchant account, billing + marketplace profiles, dummy connector,
# and a sub-merchant for the seed creator. Prints .env values to set.
# Override URL: make hs-setup HS_URL=http://localhost:8080
hs-setup:
	HS_URL=$${HS_URL:-http://localhost:8080} bash scripts/hs-setup.sh

# ─── Backup & Restore ────────────────────────────────────────────────────────

# Run a database backup now (requires .env.backup or env vars to be set).
# Usage: make backup  (sources scripts/backup/.env.backup if present)
backup:
	@if [ -f scripts/backup/.env.backup ]; then \
	  set -a && . scripts/backup/.env.backup && set +a; \
	fi; \
	bash scripts/backup/backup.sh

# Run a restore drill: restores latest backup into a temp DB and integrity-checks it.
# Usage: make restore-drill
restore-drill:
	@if [ -f scripts/backup/.env.backup ]; then \
	  set -a && . scripts/backup/.env.backup && set +a; \
	fi; \
	bash scripts/backup/restore-drill.sh

# ─── Security ─────────────────────────────────────────────────────────

# Run vulnerability check
audit:
	govulncheck ./...
