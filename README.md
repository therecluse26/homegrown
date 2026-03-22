# Homegrown Academy

A hybrid social-media + learning subscription platform that helps homeschooling families track progress, discover methodology-aligned content, and build local & online community relationships.

## Prerequisites

- [Go](https://go.dev/dl/) 1.23+
- [Node.js](https://nodejs.org/) 20+
- [Docker](https://docs.docker.com/get-docker/) (for Postgres, Redis, Kratos)
- [Air](https://github.com/air-verse/air) (Go hot-reload)

## Quick Start

```bash
# 1. Clone the repo
git clone https://github.com/your-org/homegrown-academy.git
cd homegrown-academy

# 2. Copy env file and configure
cp .env.example .env

# 3. Start infrastructure (Postgres, Redis, Kratos)
make docker-up

# 4. Install frontend dependencies
cd frontend && npm install && cd ..

# 5. Install Air (one-time)
go install github.com/air-verse/air@latest

# 6. Start development servers (backend + frontend with hot-reload)
make dev
```

The backend runs on `localhost:3000` and the frontend on `localhost:5173`.

## Available Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start backend (Air hot-reload) + frontend (Vite HMR) |
| `make dev-api` | Start only the Go backend with hot-reload |
| `make dev-web` | Start only the Vite frontend dev server |
| `make docker-up` | Start infrastructure services |
| `make docker-down` | Stop infrastructure services |
| `make check` | Run all quality gates (lint + test + type-check) |
| `make lint` | Run Go linter |
| `make test` | Run Go tests |
| `make type-check` | Run TypeScript type checker |
| `make migrate` | Run pending database migrations |
| `make db-reset` | Drop, recreate, and re-migrate the database |
| `make full-generate` | Generate OpenAPI spec + TypeScript types |
| `make audit` | Run vulnerability check |

## Architecture

See [`specs/ARCHITECTURE.md`](specs/ARCHITECTURE.md) for the full system design and ADRs.
