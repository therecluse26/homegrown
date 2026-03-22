# Domain Spec 00 — Core Infrastructure

## §1 Overview

This document specifies the **shared infrastructure** that must exist before any domain module
can be implemented. It is not a domain — it produces no user-facing features. It produces a
compilable, runnable project skeleton with database connectivity, configuration, error handling,
middleware, and all shared types in place.

| Attribute | Value |
|-----------|-------|
| **Purpose** | Foundation for all 14 domain modules |
| **Produces** | A `go build ./...`-able binary, passing `golangci-lint run` and `go test ./...` |
| **Frontend** | Minimum shell passing `npm run type-check` and `npm run dev` |
| **Prerequisite for** | Every domain spec (01-iam through 17-planning) |

### What core infrastructure owns

- Project scaffolding (`go.mod`, directory skeleton, frontend `package.json`)
- Application entrypoint (`cmd/server/main.go`)
- Application wiring (`internal/app/app.go`) — `AppState`, router composition, middleware ordering
- Configuration (`internal/config/config.go`) — `AppConfig` struct, env var loading
- Shared kernel (`internal/shared/`) — error framework, types, family scope, DB/Redis helpers,
  event bus, pagination
- Middleware stack (`internal/middleware/`) — auth, rate limiting, role extractors
- Health endpoint (`GET /health`)
- OpenAPI generation command (`cmd/swag-gen/main.go`)
- Bootstrap database migration (PostgreSQL extensions)
- Local development environment (`docker-compose.yml`, Kratos dev config)
- Development commands (`Makefile`)
- Frontend shell (`frontend/`)

### What core infrastructure does NOT own

- Domain logic (each domain spec owns its own logic)
- Auth middleware *implementation* — the auth middleware signature and behavior are defined
  here; the actual Kratos integration is implemented by IAM (01-iam) via the `KratosAdapter`
  port
- CI/CD pipeline `[ARCH §2.15]`
- Dockerfile `[ARCH §2.11]`
- CDK infrastructure `[ARCH §2.17]`
- Production deployment procedures

---

## §2 Project Scaffolding

### §2.1 go.mod

Module definition with all Phase 1 dependencies.

```go
module github.com/homegrown-academy/homegrown-academy

go 1.23

require (
	// Web framework
	github.com/labstack/echo/v4 v4.12.0

	// Database
	gorm.io/gorm v1.25.0
	gorm.io/driver/postgres v1.5.0

	// Migrations
	github.com/pressly/goose/v3 v3.21.0

	// Validation
	github.com/go-playground/validator/v10 v10.22.0

	// Logging (slog is stdlib Go 1.21+)

	// OpenAPI
	github.com/swaggo/swag v1.16.0
	github.com/swaggo/echo-swagger v1.4.0

	// Types
	github.com/google/uuid v1.6.0

	// Redis
	github.com/redis/go-redis/v9 v9.5.0

	// Configuration
	github.com/joho/godotenv v1.5.0

	// HTTP client (stdlib net/http)

	// Crypto (stdlib crypto/hmac, crypto/sha256, encoding/base64)

	// HTML sanitization [CODING §5.2]
	github.com/microcosm-cc/bluemonday v1.0.27

	// Background jobs
	github.com/hibiken/asynq v0.24.0

	// WebSocket
	github.com/gorilla/websocket v1.5.3

	// Sentry (optional error tracking) [ARCH §2.14]
	github.com/getsentry/sentry-go v0.28.0
)
```

**Note**: Exact version numbers SHOULD be updated to latest stable at implementation time.
The versions above represent the minimum compatible release.

### §2.2 Directory Skeleton

Completes the structure outlined in `[ARCH §3.4]` with all files that ARCHITECTURE.md omits.
Files marked `(core)` are created by this spec. Files marked `(domain)` are created by the
owning domain spec.

```
homegrown-academy/
├── go.mod                                  (core)
├── go.sum                                  (core)
├── Makefile                                (core) §17
├── .env.example                            (core) §3
├── docker-compose.yml                      (core) §16
├── kratos/                                 (core) §16
│   ├── kratos.yml
│   └── identity.schema.json
├── openapi/
│   └── spec.yaml                           (generated)
├── migrations/
│   ├── 00000000000000_bootstrap.sql        (core) §9 — PG extensions
│   └── {timestamp}_*.sql                   (domain) — per-domain migrations (goose format)
├── cmd/
│   └── server/
│       └── main.go                         (core) §4
├── internal/
│   ├── app/
│   │   └── app.go                          (core) §5
│   ├── config/
│   │   └── config.go                       (core) §3
│   ├── shared/
│   │   ├── error.go                        (core) §6
│   │   ├── types.go                        (core) §7
│   │   ├── family_scope.go                 (core) §8
│   │   ├── db.go                           (core) §9
│   │   ├── redis.go                        (core) §10
│   │   ├── events.go                       (core) §11
│   │   └── pagination.go                   (core) §12
│   ├── middleware/
│   │   ├── auth.go                         (core) §13
│   │   ├── rate_limit.go                   (core) §13
│   │   └── extractors.go                   (core) §13
│   └── domains/                            (domain modules added incrementally)
│       ├── iam/                            (01-iam)
│       ├── method/                         (02-method)
│       ├── media/                          (03-media)
│       └── ...                             (04-14)
└── frontend/                               (core) §18
    ├── package.json
    ├── tsconfig.json
    ├── vite.config.ts
    ├── index.html
    └── src/
        ├── main.tsx
        ├── App.tsx
        ├── query-client.ts
        ├── api/
        │   ├── client.ts
        │   └── generated/              (generated — never hand-edit)
        │       └── .gitkeep
        ├── components/
        │   └── ui/
        │       └── .gitkeep
        └── features/
            └── .gitkeep
```

### §2.3 Frontend package.json

Key dependencies and scripts for the React SPA. `[ARCH §2.3]`

```json
{
  "name": "homegrown-academy",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "type-check": "tsc --noEmit",
    "generate-types": "openapi-typescript ../openapi/spec.yaml -o src/api/generated/schema.ts",
    "lint": "eslint ."
  },
  "dependencies": {
    "@tanstack/react-query": "^5",
    "react": "^19",
    "react-dom": "^19",
    "react-router": "^7"
  },
  "devDependencies": {
    "@types/react": "^19",
    "@types/react-dom": "^19",
    "@vitejs/plugin-react": "^4",
    "autoprefixer": "^10",
    "openapi-typescript": "^7",
    "postcss": "^8",
    "tailwindcss": "^4",
    "typescript": "^5",
    "vite": "^6"
  }
}
```

### §2.4 Frontend tsconfig.json

TypeScript strict mode. `[CODING §3.1]`

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2023", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "isolatedModules": true,
    "skipLibCheck": true,
    "outDir": "dist",
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  "include": ["src"]
}
```

### §2.5 Frontend vite.config.ts

Vite configuration with API proxy for local development. `[ARCH §2.3]`

```typescript
import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import path from "path";

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/v1": {
        target: "http://localhost:3000",
        changeOrigin: true,
      },
      "/hooks": {
        target: "http://localhost:3000",
        changeOrigin: true,
      },
      "/health": {
        target: "http://localhost:3000",
        changeOrigin: true,
      },
    },
  },
});
```

---

## §3 Configuration (`internal/config/config.go`)

### §3.1 AppConfig Struct

All environment variables typed and documented. Loaded at startup, immutable thereafter.

```go
package config

// Environment represents the runtime environment.
type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentStaging     Environment = "staging"
	EnvironmentProduction  Environment = "production"
)

// AppConfig holds all application configuration loaded from environment variables.
type AppConfig struct {
	// ─── Database ───────────────────────────────────────────────────
	// PostgreSQL connection string.
	// Example: "postgres://user:pass@localhost:5432/homegrown"
	DatabaseURL string

	// Maximum connections in the GORM pool. Default: 10.
	DatabaseMaxConnections int

	// ─── Redis ──────────────────────────────────────────────────────
	// Redis connection string.
	// Example: "redis://localhost:6379"
	RedisURL string

	// ─── Ory Kratos ─────────────────────────────────────────────────
	// Kratos Admin API URL (internal sidecar, never public).
	// Example: "http://kratos:4434"
	KratosAdminURL string

	// Kratos Public API URL (browser-facing, session validation).
	// Example: "http://kratos:4433"
	KratosPublicURL string

	// Shared secret for Kratos webhook signature validation.
	KratosWebhookSecret string

	// ─── CORS ───────────────────────────────────────────────────────
	// Comma-separated list of allowed origins.
	// Example: "http://localhost:5173,https://app.homegrown.academy"
	CORSAllowedOrigins []string

	// ─── Server ─────────────────────────────────────────────────────
	// Host to bind to. Default: "0.0.0.0".
	ServerHost string

	// Port to bind to. Default: 3000.
	ServerPort int

	// ─── Logging ────────────────────────────────────────────────────
	// slog log level. Default: "info".
	// Example: "debug"
	LogLevel string

	// ─── Observability ──────────────────────────────────────────────
	// Sentry DSN. Optional — omit to disable Sentry. [ARCH §2.14]
	SentryDSN *string

	// ─── Environment ────────────────────────────────────────────────
	// Runtime environment. Controls log format, debug features, etc.
	Environment Environment
}
```

### §3.2 Loading

`LoadConfig()` loads values from environment variables with `godotenv` fallback
for local development.

```go
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// LoadConfig loads configuration from environment variables.
func LoadConfig() (*AppConfig, error) {
	// Load .env file if it exists (dev only, not required)
	_ = godotenv.Load()

	databaseURL, err := requiredEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}

	redisURL, err := requiredEnv("REDIS_URL")
	if err != nil {
		return nil, err
	}

	kratosAdminURL, err := requiredEnv("KRATOS_ADMIN_URL")
	if err != nil {
		return nil, err
	}

	kratosPublicURL, err := requiredEnv("KRATOS_PUBLIC_URL")
	if err != nil {
		return nil, err
	}

	kratosWebhookSecret, err := requiredEnv("KRATOS_WEBHOOK_SECRET")
	if err != nil {
		return nil, err
	}

	corsOrigins, err := requiredEnv("CORS_ALLOWED_ORIGINS")
	if err != nil {
		return nil, err
	}

	maxConns := 10
	if v, ok := os.LookupEnv("DATABASE_MAX_CONNECTIONS"); ok {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid DATABASE_MAX_CONNECTIONS: %w", err)
		}
		maxConns = parsed
	}

	serverHost := envOrDefault("SERVER_HOST", "0.0.0.0")

	serverPort := 3000
	if v, ok := os.LookupEnv("SERVER_PORT"); ok {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return nil, fmt.Errorf("invalid SERVER_PORT: %w", err)
		}
		serverPort = parsed
	}

	logLevel := envOrDefault("LOG_LEVEL", "info")

	var sentryDSN *string
	if v, ok := os.LookupEnv("SENTRY_DSN"); ok {
		sentryDSN = &v
	}

	envStr := envOrDefault("ENVIRONMENT", "development")
	var env Environment
	switch envStr {
	case "production":
		env = EnvironmentProduction
	case "staging":
		env = EnvironmentStaging
	default:
		env = EnvironmentDevelopment
	}

	origins := strings.Split(corsOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	return &AppConfig{
		DatabaseURL:            databaseURL,
		DatabaseMaxConnections: maxConns,
		RedisURL:               redisURL,
		KratosAdminURL:         kratosAdminURL,
		KratosPublicURL:        kratosPublicURL,
		KratosWebhookSecret:    kratosWebhookSecret,
		CORSAllowedOrigins:     origins,
		ServerHost:             serverHost,
		ServerPort:             serverPort,
		LogLevel:               logLevel,
		SentryDSN:              sentryDSN,
		Environment:            env,
	}, nil
}

func requiredEnv(key string) (string, error) {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return val, nil
}

func envOrDefault(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}
```

`requiredEnv` returns an error if absent. `envOrDefault` returns the default if absent.

### §3.3 .env.example

Complete template for local development. Committed to version control. The `.env` file itself
is gitignored.

```env
# =============================================================================
# Homegrown Academy — Local Development Environment
# Copy to .env and fill in values
# =============================================================================

# Database
DATABASE_URL=postgres://homegrown:homegrown@localhost:5432/homegrown
DATABASE_MAX_CONNECTIONS=5

# Redis
REDIS_URL=redis://localhost:6379

# Ory Kratos (sidecar)
KRATOS_ADMIN_URL=http://localhost:4434
KRATOS_PUBLIC_URL=http://localhost:4433
KRATOS_WEBHOOK_SECRET=dev-webhook-secret-change-in-production

# CORS (comma-separated)
CORS_ALLOWED_ORIGINS=http://localhost:5173

# Server
SERVER_HOST=0.0.0.0
SERVER_PORT=3000

# Logging
LOG_LEVEL=debug

# Environment
ENVIRONMENT=development

# Sentry (optional — omit to disable)
# SENTRY_DSN=https://key@sentry.io/project
```

### §3.4 Secret Injection (Production)

In production, secrets (`DATABASE_URL`, `KRATOS_WEBHOOK_SECRET`, `SENTRY_DSN`) are injected
via ECS task definition secrets referencing AWS Secrets Manager or SSM Parameter Store.
`[ARCH §2.17]` — CDK stack handles the wiring. This spec does not define CDK resources; it
only documents the contract (env var names) that the deployment stack must satisfy.

---

## §4 Application Entrypoint (`cmd/server/main.go`)

### §4.1 Runtime

Go uses goroutines natively — no async runtime configuration needed. The `main` function
orchestrates the startup sequence.

```go
package main

import (
	"context"
	"log/slog"
	"os"
)

func main() {
	// Startup sequence — order matters
	// (1) Init slog logger
	// (2) Load config
	// (3) Create DB pool
	// (4) Run migrations (goose)
	// (5) Create Redis client
	// (6) Init EventBus + register subscriptions
	// (7) Wire AppState
	// (8) Build Echo app
	// (9) Start server
	// (10) Serve with graceful shutdown
}
```

### §4.2 Startup Sequence

The startup sequence MUST execute in the order below. Each step depends on the output of the
previous step.

| Step | Action | Depends On | Failure Behavior |
|------|--------|------------|------------------|
| 1 | `initLogger(cfg)` | None (uses defaults until config loaded) | Fatal — exit |
| 2 | `config.LoadConfig()` | — | Fatal — exit with missing-env message |
| 3 | `db.CreatePool(cfg)` | Config | Fatal — exit |
| 4 | `goose.Up(db, migrationsDir)` | DB pool | Fatal — exit with migration error |
| 5 | `redis.NewClient(cfg)` | Config | Fatal — exit |
| 6 | `events.NewEventBus()` + subscription registration | Services | Fatal — exit |
| 7 | `app.NewAppState(db, redis, eventBus, services...)` | All above | Infallible |
| 8 | `app.NewApp(state)` | AppState | Infallible |
| 9 | `e.Start(addr)` | Config | Fatal — exit |
| 10 | Graceful shutdown via `os.Signal` listener | All above | Runs until signal |

**Logger bootstrap**: Step 1 initializes with defaults first (so steps 2-5 can log). After
config is loaded, the slog handler is reconfigured with the config-specified log level.

### §4.3 Graceful Shutdown

The server listens for SIGTERM (ECS sends this on task stop) and SIGINT (Ctrl-C for local
development). On signal reception:

1. Stop accepting new connections
2. Wait up to 30 seconds for in-flight requests to complete
3. Close database pool connections
4. Close Redis connections
5. Exit with code 0

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func gracefulShutdown(ctx context.Context, e *echo.Echo, cleanup func()) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sig := <-quit
	slog.Info("received shutdown signal", "signal", sig.String())

	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	cleanup()
	slog.Info("server stopped")
}
```

---

## §5 Application Wiring (`internal/app/app.go`)

### §5.1 AppState

The central state struct passed to all Echo handlers via context. Contains shared
infrastructure and all domain service interfaces.

```go
package app

import (
	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// AppState holds shared infrastructure and domain service interfaces.
type AppState struct {
	// ─── Infrastructure ─────────────────────────────────────────────
	DB       *gorm.DB
	Redis    *redis.Client
	EventBus *shared.EventBus
	Config   *config.AppConfig

	// ─── Domain Services (added incrementally as domains are built) ─
	// IAM    IamService
	// Method MethodologyService
	// Social SocialService
	// ... etc.
}
```

Domain service fields are uncommented as each domain is implemented. The core skeleton
compiles with infrastructure fields only.

### §5.2 Router Composition

`NewApp(state) -> *echo.Echo` builds the Echo router with middleware layering and route groups.

```go
package app

import (
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/homegrown-academy/homegrown-academy/internal/middleware"
)

func NewApp(state *AppState) *echo.Echo {
	e := echo.New()

	// ─── Global Middleware (outermost applied first) ──────────────
	e.Use(echomw.RequestLoggerWithConfig(requestLoggerConfig()))
	e.Use(middleware.SecurityHeaders())
	e.Use(echomw.CORSWithConfig(corsConfig(state.Config)))

	// ─── Public Routes ───────────────────────────────────────────
	e.GET("/health", healthHandler(state))

	// ─── Webhook Routes ──────────────────────────────────────────
	// Domain webhooks added here (e.g., /hooks/kratos/*)
	// hooks := e.Group("/hooks")

	// ─── Authenticated Routes ────────────────────────────────────
	// Domain routes added here, all behind auth middleware
	auth := e.Group("")
	auth.Use(middleware.Auth(state))

	return e
}
```

### §5.3 Middleware Stack Ordering

Middleware is applied outermost-first. The request passes through them top-to-bottom:

| Order | Layer | Scope | Purpose |
|-------|-------|-------|---------|
| 1 | `RequestLogger` | All routes | Request/response logging with timing |
| 2 | Security headers | All routes | `X-Content-Type-Options`, `X-Frame-Options`, etc. |
| 3 | CORS | All routes | Enforces allowed origins `[ARCH §2.3]` |
| 4 | Rate limiting | All routes | Token bucket per IP (unauth) or per user (auth) |
| 5 | Auth middleware | Authenticated routes only | Cookie → Kratos → DB → `AuthContext` |

Rate limiting (layer 4) applies globally but with different tiers based on authentication
status. Auth middleware (layer 5) applies only to authenticated route groups.

### §5.4 Health Endpoint

```
GET /health
```

Returns 200 with:

```json
{
  "status": "ok",
  "version": "0.1.0"
}
```

The version string is set from a build-time `-ldflags` variable. This endpoint
is unauthenticated — used by ALB health checks and UptimeRobot. `[ARCH §2.14]`

No database connectivity check in the health endpoint. Database health is validated at startup
(step 3-4) and monitored via CloudWatch RDS metrics. A health endpoint that queries the
database on every ALB tick creates unnecessary load.

### §5.5 Naming Conflict Resolution: AppError

**Decision**: The project uses `AppError` (per CODING_STANDARDS.md §2.2), not `ApiError`.

ARCHITECTURE.md §6.2 uses `ApiError` in code samples. This was a naming inconsistency
introduced during documentation. CODING_STANDARDS.md is the authoritative source for naming
(`[CODING §1]`), and it specifies `AppError`. All code and documentation MUST use `AppError`.

All references to `ApiError` in ARCHITECTURE.md code samples should be read as `AppError`.
This does NOT require editing ARCHITECTURE.md — it is a documentation artifact; the code is
authoritative.

---

## §6 Error Framework (`internal/shared/error.go`)

### §6.1 AppError Type

The application-wide error type. All domain errors convert to `AppError` before reaching
the handler return type. `[CODING §2.2]`

```go
package shared

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError represents an application-level error with HTTP status mapping.
type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Err        error // wrapped internal error, never exposed to client
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Predefined error constructors

func ErrNotFound() *AppError {
	return &AppError{Code: "not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
}

func ErrUnauthorized() *AppError {
	return &AppError{Code: "unauthorized", Message: "Authentication required", StatusCode: http.StatusUnauthorized}
}

func ErrForbidden() *AppError {
	return &AppError{Code: "forbidden", Message: "Access denied", StatusCode: http.StatusForbidden}
}

func ErrPremiumRequired() *AppError {
	return &AppError{Code: "premium_required", Message: "Premium subscription required", StatusCode: http.StatusPaymentRequired}
}

func ErrCoppaConsentRequired() *AppError {
	return &AppError{Code: "coppa_consent_required", Message: "COPPA parental consent required", StatusCode: http.StatusForbidden}
}

func ErrValidation(msg string) *AppError {
	return &AppError{Code: "validation_error", Message: msg, StatusCode: http.StatusUnprocessableEntity}
}

func ErrConflict(msg string) *AppError {
	return &AppError{Code: "conflict", Message: msg, StatusCode: http.StatusConflict}
}

func ErrRateLimited() *AppError {
	return &AppError{Code: "rate_limited", Message: "Rate limit exceeded", StatusCode: http.StatusTooManyRequests}
}

func ErrBadRequest(msg string) *AppError {
	return &AppError{Code: "bad_request", Message: msg, StatusCode: http.StatusBadRequest}
}

func ErrAccountSuspended() *AppError {
	return &AppError{Code: "account_suspended", Message: "Your account has been temporarily suspended", StatusCode: http.StatusForbidden}
}

func ErrAccountBanned() *AppError {
	return &AppError{Code: "account_banned", Message: "Your account has been permanently restricted", StatusCode: http.StatusForbidden}
}

func ErrInternal(err error) *AppError {
	return &AppError{Code: "internal_error", Message: "An internal error occurred", StatusCode: http.StatusInternalServerError, Err: err}
}

func ErrDatabase(err error) *AppError {
	return &AppError{Code: "internal_error", Message: "An internal error occurred", StatusCode: http.StatusInternalServerError, Err: err}
}
```

### §6.2 HTTP Status Mapping

| Constructor | HTTP Status | JSON `code` |
|---------|-------------|-------------|
| `ErrNotFound()` | 404 Not Found | `not_found` |
| `ErrUnauthorized()` | 401 Unauthorized | `unauthorized` |
| `ErrForbidden()` | 403 Forbidden | `forbidden` |
| `ErrPremiumRequired()` | 402 Payment Required | `premium_required` |
| `ErrCoppaConsentRequired()` | 403 Forbidden | `coppa_consent_required` |
| `ErrValidation(msg)` | 422 Unprocessable Entity | `validation_error` |
| `ErrConflict(msg)` | 409 Conflict | `conflict` |
| `ErrRateLimited()` | 429 Too Many Requests | `rate_limited` |
| `ErrBadRequest(msg)` | 400 Bad Request | `bad_request` |
| `ErrAccountSuspended()` | 403 Forbidden | `account_suspended` |
| `ErrAccountBanned()` | 403 Forbidden | `account_banned` |
| `ErrInternal(err)` | 500 Internal Server Error | `internal_error` |
| `ErrDatabase(err)` | 500 Internal Server Error | `internal_error` |

### §6.3 Echo Error Handler

```go
package shared

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"
)

// ErrorResponse is the JSON structure returned for all errors.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains the machine-readable code and human-readable message.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPErrorHandler is a custom Echo error handler that maps AppError to JSON responses.
func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.Err != nil {
			slog.Error("internal server error", "error", appErr.Err)
		}
		_ = c.JSON(appErr.StatusCode, ErrorResponse{
			Error: ErrorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
			},
		})
		return
	}

	// Fallback for non-AppError errors
	slog.Error("unhandled error", "error", err)
	_ = c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorBody{
			Code:    "internal_error",
			Message: "An internal error occurred",
		},
	})
}
```

**Key behavior**:
- `ErrInternal` and `ErrDatabase` constructors set the wrapped `Err` which is logged via
  `slog.Error` but only `"An internal error occurred"` is returned to the client.
  `[CODING §2.2, §5.2]`
- Validation messages from `go-playground/validator` are user-facing and included in the response.
- The `code` field provides a machine-readable error identifier for the frontend.

### §6.4 Domain Error Conversion Pattern

Each domain defines its own error types and provides a function to convert to `AppError`.
This pattern keeps domain errors specific while converging to a single HTTP error type.

```go
// Example pattern — each domain implements this
func ToAppError(err error) *shared.AppError {
	var iamErr *IamError
	if errors.As(err, &iamErr) {
		switch iamErr.Kind {
		case IamErrFamilyNotFound, IamErrParentNotFound:
			return shared.ErrNotFound()
		case IamErrNotPrimaryParent:
			return shared.ErrForbidden()
		case IamErrPremiumRequired:
			return shared.ErrPremiumRequired()
		case IamErrCoppaConsentRequired:
			return shared.ErrCoppaConsentRequired()
		case IamErrInvalidConsentTransition:
			return shared.ErrValidation(iamErr.Error())
		case IamErrParentAlreadyInFamily:
			return shared.ErrConflict(iamErr.Error())
		case IamErrDatabaseError:
			return shared.ErrDatabase(iamErr.Err)
		case IamErrKratosError:
			return shared.ErrInternal(fmt.Errorf("Kratos error: %w", iamErr.Err))
		default:
			return shared.ErrInternal(iamErr)
		}
	}
	return shared.ErrInternal(err)
}
```

### §6.5 Validator Integration

```go
package shared

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationError converts go-playground/validator errors to an AppError.
func ValidationError(err error) *AppError {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		var msgs []string
		for _, fe := range ve {
			msgs = append(msgs, fmt.Sprintf("field '%s' failed on '%s' validation", fe.Field(), fe.Tag()))
		}
		return ErrValidation(strings.Join(msgs, "; "))
	}
	return ErrValidation(err.Error())
}
```

This allows handlers to call `validate.Struct(cmd)` and convert validation errors to
422 responses via `shared.ValidationError(err)`.

---

## §7 Shared Types (`internal/shared/types.go`)

### §7.1 Newtype Wrappers

Type-safe wrappers for UUID identifiers. Prevents accidentally passing a `FamilyID` where a
`ParentID` is expected. `[ARCH §4.2]`

```go
package shared

import (
	"encoding/json"

	"github.com/google/uuid"
)

// FamilyID is a type-safe wrapper for family UUIDs.
type FamilyID struct {
	uuid.UUID
}

// NewFamilyID creates a FamilyID from a UUID.
func NewFamilyID(id uuid.UUID) FamilyID {
	return FamilyID{UUID: id}
}

// MarshalJSON implements json.Marshaler.
func (id FamilyID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.UUID)
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *FamilyID) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &id.UUID)
}

// ParentID is a type-safe wrapper for parent UUIDs.
type ParentID struct {
	uuid.UUID
}

func NewParentID(id uuid.UUID) ParentID {
	return ParentID{UUID: id}
}

func (id ParentID) MarshalJSON() ([]byte, error)  { return json.Marshal(id.UUID) }
func (id *ParentID) UnmarshalJSON(data []byte) error { return json.Unmarshal(data, &id.UUID) }

// StudentID is a type-safe wrapper for student UUIDs.
type StudentID struct {
	uuid.UUID
}

func NewStudentID(id uuid.UUID) StudentID {
	return StudentID{UUID: id}
}

func (id StudentID) MarshalJSON() ([]byte, error)  { return json.Marshal(id.UUID) }
func (id *StudentID) UnmarshalJSON(data []byte) error { return json.Unmarshal(data, &id.UUID) }

// CreatorID is a type-safe wrapper for creator UUIDs.
type CreatorID struct {
	uuid.UUID
}

func NewCreatorID(id uuid.UUID) CreatorID {
	return CreatorID{UUID: id}
}

func (id CreatorID) MarshalJSON() ([]byte, error)  { return json.Marshal(id.UUID) }
func (id *CreatorID) UnmarshalJSON(data []byte) error { return json.Unmarshal(data, &id.UUID) }
```

### §7.2 AuthContext

The authoritative type definition for `AuthContext`. Consolidates `[ARCH §6.2]` and
`[IAM §11.1]`.

```go
// SubscriptionTier represents the family's subscription level.
type SubscriptionTier string

const (
	SubscriptionTierFree    SubscriptionTier = "free"
	SubscriptionTierPremium SubscriptionTier = "premium"
)

// ParseSubscriptionTier parses a string into a SubscriptionTier.
func ParseSubscriptionTier(s string) SubscriptionTier {
	switch s {
	case "premium":
		return SubscriptionTierPremium
	default:
		return SubscriptionTierFree
	}
}

// AuthContext represents the authenticated user context, stored in Echo's
// request context by auth middleware. Consumed by every authenticated handler.
type AuthContext struct {
	ParentID         uuid.UUID        `json:"parent_id"`
	FamilyID         uuid.UUID        `json:"family_id"`
	KratosIdentityID uuid.UUID        `json:"kratos_identity_id"`
	IsPrimaryParent  bool             `json:"is_primary_parent"`
	IsPlatformAdmin  bool             `json:"is_platform_admin"`  // [S§3.1.5, 11-safety §9]
	SubscriptionTier SubscriptionTier `json:"subscription_tier"`
	Email            string           `json:"-"` // NOT logged or serialized — PII [CODING §5.2]
}
```

**`AuthContext` extraction from Echo context**: The auth middleware stores `AuthContext` in
Echo's context; helper functions retrieve it.

```go
// contextKey is an unexported type to prevent key collisions.
type contextKey string

const authContextKey contextKey = "auth_context"

// SetAuthContext stores the AuthContext in the Echo context.
func SetAuthContext(c echo.Context, auth *AuthContext) {
	c.Set(string(authContextKey), auth)
}

// GetAuthContext retrieves the AuthContext from the Echo context.
// Returns an error if not present (handler is behind auth middleware).
func GetAuthContext(c echo.Context) (*AuthContext, error) {
	val := c.Get(string(authContextKey))
	if val == nil {
		return nil, ErrUnauthorized()
	}
	auth, ok := val.(*AuthContext)
	if !ok {
		return nil, ErrUnauthorized()
	}
	return auth, nil
}
```

### §7.3 SubscriptionTier

Used by `AuthContext` and permission extractors. Matches the `subscription_tier` column on
`iam_families`. (Defined inline with `AuthContext` above in §7.2.)

---

## §8 Family Scope (`internal/shared/family_scope.go`)

### §8.1 FamilyScope Type

`FamilyScope` wraps a `family_id` and enforces that every repository query includes a family
filter. The `familyID` field is unexported — callers can read it but cannot construct a
`FamilyScope` outside of the designated paths. `[ARCH §1.5, CODING §2.4]`

```go
package shared

import "github.com/google/uuid"

// FamilyScope wraps a family_id for privacy-enforcing database queries.
//
// The unexported field ensures FamilyScope can only be created from:
// 1. AuthContext (via NewFamilyScopeFromAuth) — the normal authenticated path
// 2. newFamilyScope — for auth middleware and registration flows (package-internal)
type FamilyScope struct {
	familyID uuid.UUID
}

// newFamilyScope creates a FamilyScope from a raw family_id.
// Package-internal use only (auth middleware, registration).
func newFamilyScope(familyID uuid.UUID) FamilyScope {
	return FamilyScope{familyID: familyID}
}

// FamilyID returns the wrapped family_id. The only public access to the value.
func (s FamilyScope) FamilyID() uuid.UUID {
	return s.familyID
}
```

### §8.2 From AuthContext

```go
// NewFamilyScopeFromAuth creates a FamilyScope from an AuthContext.
func NewFamilyScopeFromAuth(auth *AuthContext) FamilyScope {
	return newFamilyScope(auth.FamilyID)
}
```

### §8.3 Echo Context Helper

```go
// GetFamilyScope extracts a FamilyScope from the Echo context's AuthContext.
// Returns an error if AuthContext is not present.
func GetFamilyScope(c echo.Context) (FamilyScope, error) {
	auth, err := GetAuthContext(c)
	if err != nil {
		return FamilyScope{}, err
	}
	return NewFamilyScopeFromAuth(auth), nil
}
```

### §8.4 RLS Integration

`FamilyScope` connects to the database RLS layer via `ScopedTransaction()` (§9.2). When a
repository executes a query inside a scoped transaction, PostgreSQL's `app.current_family_id`
setting is automatically applied, and RLS policies enforce family isolation at the database
level as defense-in-depth.

---

## §9 Database Infrastructure (`internal/shared/db.go`)

### §9.1 Pool Management

```go
package shared

import (
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// CreatePool creates a GORM database connection pool.
func CreatePool(cfg *config.AppConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Use slog, not GORM's built-in logging
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.DatabaseMaxConnections)
	sqlDB.SetMaxIdleConns(1)

	return db, nil
}
```

### §9.2 Scoped Transactions

`ScopedTransaction` begins a database transaction, sets `SET LOCAL app.current_family_id`
for RLS enforcement, executes a callback, and commits. The `SET LOCAL` value is automatically
cleared on commit or rollback.

```go
package shared

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

// ScopedTransaction executes fn within a family-scoped transaction.
// Sets `app.current_family_id` for RLS enforcement.
func ScopedTransaction(ctx context.Context, db *gorm.DB, scope FamilyScope, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		setSQL := fmt.Sprintf("SET LOCAL app.current_family_id = '%s'", scope.FamilyID().String())
		if err := tx.Exec(setSQL).Error; err != nil {
			return fmt.Errorf("failed to set family scope: %w", err)
		}
		return fn(tx)
	})
}
```

### §9.3 Unscoped Transactions

For operations that intentionally bypass family scope. Each call site MUST document why.

```go
// UnscopedTransaction executes fn in a transaction WITHOUT family scope.
//
// ONLY for:
// - Auth middleware lookups (FamilyScope not yet constructed)
// - Registration webhooks (family does not exist yet)
// - Background cleanup jobs (cross-family by design)
//
// Every call site MUST have a comment explaining why unscoped access is required.
func UnscopedTransaction(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	})
}
```

### §9.4 Bootstrap Migration

The first migration (`00000000000000_bootstrap.sql`) installs PostgreSQL extensions required by
all domains. This was previously in 01-iam's first migration — it is moved here because
extensions are shared infrastructure, not IAM-specific.

```sql
-- +goose Up
-- Migration: 00000000000000_bootstrap.sql
-- Installs PostgreSQL extensions required by the platform.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";     -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";       -- Cryptographic functions
CREATE EXTENSION IF NOT EXISTS "postgis";        -- Spatial queries [ARCH §5.4]
CREATE EXTENSION IF NOT EXISTS "pg_trgm";        -- Trigram fuzzy matching [ARCH §2.6]

-- +goose Down
DROP EXTENSION IF EXISTS "pg_trgm";
DROP EXTENSION IF EXISTS "postgis";
DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
```

**Idempotent**: `IF NOT EXISTS` ensures this migration can be re-run safely.

### §9.5 Application vs. Migration Roles

Two PostgreSQL roles are used:

| Role | Purpose | RLS behavior |
|------|---------|--------------|
| Migration role | Runs goose migrations (CREATE TABLE, ALTER, etc.) | Bypasses RLS (superuser or table owner) |
| Application role | Used by the Go API at runtime | Subject to RLS policies |

The application role MUST NOT be a superuser. This ensures RLS cannot be accidentally
bypassed by application queries. The migration role creates tables and RLS policies; the
application role respects them.

In local development (docker-compose), a single `homegrown` user with superuser is acceptable.
In production (RDS), the application user is a non-superuser role granted `SELECT, INSERT,
UPDATE, DELETE` on all tables. `[ARCH §2.5, §5.2]`

---

## §10 Redis Infrastructure (`internal/shared/redis.go`)

### §10.1 Client Creation

```go
package shared

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/config"
	"github.com/redis/go-redis/v9"
)

// CreateRedisClient creates a Redis client and validates connectivity.
func CreateRedisClient(ctx context.Context, cfg *config.AppConfig) (*redis.Client, error) {
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("invalid redis URL: %w", err)
	}

	client := redis.NewClient(opts)

	// Validate connectivity with PING
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	return client, nil
}
```

### §10.2 Phase 1 Helpers

Generic get/set with TTL and delete. These are the only Redis operations needed in Phase 1
(rate limiting + methodology config caching).

```go
package shared

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisGet retrieves a JSON-serialized value from Redis.
func RedisGet[T any](ctx context.Context, client *redis.Client, key string) (*T, error) {
	val, err := client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, ErrInternal(fmt.Errorf("redis get: %w", err))
	}

	var result T
	if err := json.Unmarshal([]byte(val), &result); err != nil {
		return nil, ErrInternal(fmt.Errorf("redis unmarshal: %w", err))
	}
	return &result, nil
}

// RedisSet stores a JSON-serialized value in Redis with a TTL.
func RedisSet[T any](ctx context.Context, client *redis.Client, key string, value T, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return ErrInternal(fmt.Errorf("redis marshal: %w", err))
	}
	if err := client.Set(ctx, key, string(data), ttl).Err(); err != nil {
		return ErrInternal(fmt.Errorf("redis set: %w", err))
	}
	return nil
}

// RedisDelete removes a key from Redis.
func RedisDelete(ctx context.Context, client *redis.Client, key string) error {
	if err := client.Del(ctx, key).Err(); err != nil {
		return ErrInternal(fmt.Errorf("redis del: %w", err))
	}
	return nil
}
```

### §10.3 Rate Limit Counter

Atomic increment with expiry for rate limiting (§13.2).

```go
// RedisIncrementWithExpiry increments a counter and sets expiry if this is
// the first increment in the window. Returns the new counter value.
func RedisIncrementWithExpiry(ctx context.Context, client *redis.Client, key string, window time.Duration) (int64, error) {
	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return 0, ErrInternal(fmt.Errorf("redis incr: %w", err))
	}

	if count == 1 {
		// First request in window — set expiry
		if err := client.Expire(ctx, key, window).Err(); err != nil {
			return 0, ErrInternal(fmt.Errorf("redis expire: %w", err))
		}
	}

	return count, nil
}
```

### §10.4 Deferred Redis Features

The following Redis usage patterns are owned by their consuming domains and are NOT specified
here:

| Feature | Owning Domain | Redis Data Structure |
|---------|---------------|---------------------|
| Feed fan-out | `social::` | Sorted sets |
| Pub/sub (WebSocket) | `social::`, `notify::` | Pub/sub channels |
| Background job queues | `safety::`, `notify::`, `search::` | Lists (asynq) |
| Session caching | `iam::` | Strings with TTL |

---

## §11 Event Bus (`internal/shared/events.go`)

### §11.1 Core Interfaces

```go
package shared

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
)

// DomainEvent is the interface all domain events must implement.
// Events MUST be defined in the emitting domain's events.go. [CODING §8.4]
type DomainEvent interface {
	EventName() string
}

// DomainEventHandler handles a specific domain event type.
// Handlers MUST be defined in the consuming domain. [CODING §8.4]
type DomainEventHandler interface {
	Handle(ctx context.Context, event DomainEvent) error
}
```

### §11.2 EventBus Implementation

The event bus uses `reflect.Type`-based dispatch. Each event type maps to a list of handlers.

```go
// EventBus dispatches domain events to registered handlers.
type EventBus struct {
	mu       sync.RWMutex
	handlers map[reflect.Type][]DomainEventHandler
}

// NewEventBus creates a new EventBus.
func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[reflect.Type][]DomainEventHandler),
	}
}

// Subscribe registers a handler for a specific event type.
// MUST be called at startup only (in main.go). [CODING §8.4]
func (b *EventBus) Subscribe(eventType reflect.Type, handler DomainEventHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventType] = append(b.handlers[eventType], handler)
}

// Publish dispatches a domain event to all registered handlers.
//
// Phase 1: Synchronous dispatch. Handlers run inline within the
// caller's goroutine. Handler errors are logged but do NOT fail the
// publish — the publishing domain's operation has already succeeded.
func (b *EventBus) Publish(ctx context.Context, event DomainEvent) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	eventType := reflect.TypeOf(event)
	handlers, ok := b.handlers[eventType]
	if !ok {
		return nil
	}

	for _, handler := range handlers {
		if err := handler.Handle(ctx, event); err != nil {
			slog.Error("event handler failed",
				"event_type", event.EventName(),
				"handler", fmt.Sprintf("%T", handler),
				"error", err,
			)
			// Handler errors are logged, not propagated.
			// The domain operation that triggered the event has
			// already completed successfully.
		}
	}

	return nil
}
```

### §11.3 Phase 1 Behavior

- **Synchronous in-process dispatch**: `Publish()` calls handlers sequentially within the
  same goroutine. No message broker, no serialization overhead.
- **Error isolation**: Handler failures are logged but do not fail the publish call. The
  domain operation that triggered the event has already committed successfully — handler
  failure is a downstream concern, not a transactional rollback trigger.
- **No retry**: Failed handlers are logged and skipped. Phase 2 adds retry via background
  jobs for handlers that do heavy work.

### §11.4 Subscription Registration

All subscriptions MUST be registered in `cmd/server/main.go` during startup (step 6 of §4.2).
`[CODING §8.4]`

```go
// In cmd/server/main.go, step 6:
eventBus := shared.NewEventBus()

// IAM events → Social handler
eventBus.Subscribe(
	reflect.TypeOf(iam.FamilyCreated{}),
	social.NewOnFamilyCreatedHandler(socialService),
)

// IAM events → Onboarding handler
eventBus.Subscribe(
	reflect.TypeOf(iam.FamilyCreated{}),
	onboard.NewOnFamilyCreatedHandler(onboardService),
)

// ... more subscriptions added as domains are built
```

---

## §12 Pagination (`internal/shared/pagination.go`)

### §12.1 PaginationParams

Query parameters accepted by list endpoints. Cursor-based pagination is the primary
mechanism. `[ARCH §1.4]`

```go
package shared

// PaginationParams holds cursor-based pagination parameters.
type PaginationParams struct {
	// Opaque cursor from previous response's NextCursor.
	// Omit for the first page.
	Cursor *string `query:"cursor"`

	// Items per page. Default: 20. Max: 100.
	Limit *int `query:"limit"`
}

// EffectiveLimit returns the clamped limit value.
func (p PaginationParams) EffectiveLimit() int {
	if p.Limit == nil {
		return 20
	}
	limit := *p.Limit
	if limit > 100 {
		return 100
	}
	if limit < 1 {
		return 20
	}
	return limit
}
```

### §12.2 PaginatedResponse

```go
// PaginatedResponse wraps a page of results with cursor metadata.
type PaginatedResponse[T any] struct {
	Data       []T     `json:"data"`
	NextCursor *string `json:"next_cursor"` // nil if no more results
	HasMore    bool    `json:"has_more"`
}
```

### §12.3 Cursor Encoding

Cursors encode `(id, created_at)` as base64url to prevent clients from constructing or
manipulating cursors. The encoding is an implementation detail — clients treat cursors as
opaque strings.

```go
package shared

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EncodeCursor encodes an ID and timestamp into an opaque cursor string.
func EncodeCursor(id uuid.UUID, createdAt time.Time) string {
	raw := fmt.Sprintf("%s:%d", id.String(), createdAt.UnixMilli())
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes an opaque cursor string into an ID and timestamp.
func DecodeCursor(cursor string) (uuid.UUID, time.Time, error) {
	bytes, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return uuid.Nil, time.Time{}, ErrBadRequest("invalid cursor")
	}

	raw := string(bytes)
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return uuid.Nil, time.Time{}, ErrBadRequest("invalid cursor")
	}

	id, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, time.Time{}, ErrBadRequest("invalid cursor")
	}

	ts, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return uuid.Nil, time.Time{}, ErrBadRequest("invalid cursor")
	}

	createdAt := time.UnixMilli(ts)
	return id, createdAt, nil
}
```

---

## §13 Middleware Stack (`internal/middleware/`)

### §13.1 Auth Middleware (`auth.go`)

The auth middleware validates Kratos session cookies and populates `AuthContext` in the Echo
context.

**Signature**:

```go
package middleware

import (
	"github.com/labstack/echo/v4"
	"github.com/homegrown-academy/homegrown-academy/internal/app"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Auth returns an Echo middleware that validates Kratos sessions
// and populates AuthContext.
func Auth(state *app.AppState) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Extract session cookie
			// 2. Validate via Kratos
			// 3. Look up parent in local DB
			// 4. Look up family
			// 5. Build AuthContext
			// 5.5. Check account access (safety)
			// 6. Store AuthContext in Echo context
			// 7. Call next(c)
			return next(c)
		}
	}
}
```

**Behavior**:

1. Extract session cookie from the `Cookie` header
2. Call Kratos public API to validate the session (via `KratosAdapter` port from IAM)
3. Look up the parent in the local database by `kratos_identity_id` (unscoped — FamilyScope
   does not exist yet)
4. Look up the parent's family
5. Build `AuthContext` from parent + family data (including `is_platform_admin`)
5.5. **[safety:: integration]** Check account access via
   `SafetyService.CheckAccountAccess(familyID)`. Uses Redis cache (60s TTL). If the
   account is suspended or banned, return `AccountSuspended` or `AccountBanned` error
   immediately — do not proceed to the handler. `[11-safety §12.3]`
6. Store `AuthContext` in Echo context via `shared.SetAuthContext(c, auth)`
7. Call `next(c)`

**Error responses**:
- No cookie present → 401 Unauthorized
- Kratos session invalid/expired → 401 Unauthorized
- Parent not found in local DB (orphaned Kratos identity) → 401 Unauthorized

**Implementation ownership**: The auth middleware *function* lives in `internal/middleware/auth.go`
(core infrastructure). It calls `KratosAdapter.ValidateSession()` which is an interface defined
in `internal/iam/ports.go` and implemented in `internal/iam/adapters/kratos.go`. The middleware depends
on IAM's adapter at runtime but does not import IAM's service layer.

**Dependency note**: Because auth middleware depends on `KratosAdapter`, the IAM domain's
adapter must be available before the middleware can function. The `AppState` provides the
adapter as a `KratosAdapter` interface value.

### §13.2 Rate Limiting (`rate_limit.go`)

Token-bucket rate limiting via Redis. `[ARCH §3.3, S§2.3]`

**Tiers**:

| Tier | Scope | Limit | Window |
|------|-------|-------|--------|
| Default (unauthenticated) | Per IP | 100 requests | 60 seconds |
| Default (authenticated) | Per user ID | 100 requests | 60 seconds |
| Auth endpoints (`/hooks/*`) | Per IP | 10 requests | 60 seconds |
| Sensitive endpoints | Per user ID | 20 requests | 60 seconds |

**Behavior**:
- Uses `RedisIncrementWithExpiry` from `internal/shared/redis.go` (§10.3)
- On exceeded limit: return 429 Too Many Requests with `Retry-After` header (seconds until
  window expires)
- Rate limit key format: `rl:{scope}:{identifier}:{window_start}`

**Note**: Rate limit tiers can be overridden per-route using Echo's middleware system. Specific
domains may apply more restrictive limits to their endpoints.

### §13.3 Role Extractors (`extractors.go`)

Helper functions that extract `AuthContext` with additional permission checks. These are consumed
by many domains — they are shared infrastructure, not IAM-specific.

#### `RequirePremium`

```go
// RequirePremium extracts AuthContext and verifies the user has a premium subscription.
// Returns 402 Payment Required if the user is on the free tier. [S§3.2]
func RequirePremium(c echo.Context) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	if auth.SubscriptionTier != shared.SubscriptionTierPremium {
		return nil, shared.ErrPremiumRequired()
	}

	return auth, nil
}
```

**Consuming domains**: `learn::`, `comply::`, `recs::` `[IAM §13.1]`

#### `RequireCreator`

```go
// CreatorContext holds auth context plus verified creator ID.
type CreatorContext struct {
	Auth      *shared.AuthContext
	CreatorID uuid.UUID
}

// RequireCreator extracts AuthContext and verifies the user has a creator account.
// Returns 403 Forbidden if no creator account exists. [S§3.1.4]
//
// The creator_id is looked up from `mkt_creators` via the parent_id.
// This lookup crosses the IAM → Marketplace boundary via AppState,
// which holds a reference to the marketplace service or a lightweight
// creator-check query.
```

**Implementation note**: `RequireCreator` needs to check `mkt_creators` table. This crosses
domain boundaries. Two acceptable approaches:

1. **Preferred**: Add `CreatorID *uuid.UUID` to `AuthContext` (populated during auth
   middleware from a JOIN). No cross-domain call needed.
2. **Alternative**: Query `mkt::` service from the extractor via `AppState`.

The chosen approach is documented in the marketplace domain spec (when written).

**Consuming domains**: `mkt::` `[IAM §13.1]`

#### `RequireCoppaConsent`

```go
// RequireCoppaConsent extracts AuthContext and verifies the family has active COPPA consent.
// Returns 403 Forbidden with code `coppa_consent_required` if consent
// is not `Consented` or `ReVerified`. [ARCH §6.3]
func RequireCoppaConsent(c echo.Context) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	// Check coppa_consent_status from AuthContext
	// (populated by auth middleware from iam_families)

	return auth, nil
}
```

**Implementation note**: Requires looking up `iam_families.coppa_consent_status`. Two
approaches:

1. **Preferred**: Add `CoppaConsentStatus` to `AuthContext` (populated during auth
   middleware). No additional DB query per request.
2. **Alternative**: Query `iam::` service from the extractor.

Approach 1 is recommended — the auth middleware already queries `iam_families` for the
subscription tier, so adding the consent status is free.

**Consuming domains**: `learn::`, `social::` (student-facing features) `[IAM §13.1]`

#### `RequirePrimaryParent` (Phase 2)

```go
// RequirePrimaryParent extracts AuthContext and verifies the user is the
// family's primary parent.
// Returns 403 Forbidden if not. [S§3.4]
func RequirePrimaryParent(c echo.Context) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	if !auth.IsPrimaryParent {
		return nil, shared.ErrForbidden()
	}

	return auth, nil
}
```

**Phase 2**: Used by co-parent management, family deletion, and COPPA withdrawal endpoints.

#### `RequireAdmin`

```go
// RequireAdmin extracts AuthContext and verifies the user is a platform administrator.
// Returns 403 Forbidden if the user is not an admin. [S§3.1.5]
//
// Backed by iam_parents.is_platform_admin column (01-iam §3.1).
// Phase 1: single boolean. Phase 2: granular admin roles. [11-safety §9]
func RequireAdmin(c echo.Context) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	if !auth.IsPlatformAdmin {
		return nil, shared.ErrForbidden()
	}

	return auth, nil
}
```

**Consuming domains**: `safety::` (moderation dashboard, admin actions) `[11-safety §4.2]`

---

## §14 Logging & Observability

### §14.1 Initialization

```go
package main

import (
	"log/slog"
	"os"

	"github.com/getsentry/sentry-go"
	"github.com/homegrown-academy/homegrown-academy/internal/config"
)

func initLogger(cfg *config.AppConfig) {
	var handler slog.Handler

	level := parseLogLevel(cfg.LogLevel)

	if cfg.Environment == config.EnvironmentProduction {
		// JSON format for structured log aggregation (CloudWatch)
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		// Text format for local development
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	slog.SetDefault(slog.New(handler))

	// Optional Sentry integration [ARCH §2.14]
	if cfg.SentryDSN != nil {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              *cfg.SentryDSN,
			Release:          version, // set via -ldflags at build time
			TracesSampleRate: 0.1,
		})
		if err != nil {
			slog.Error("sentry initialization failed", "error", err)
		}
	}
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
```

### §14.2 Log Level Defaults

Development default: `debug`

Production default: `info`

### §14.3 PII Rules

`[CODING §5.2]` — These are absolute prohibitions:

| Data | Logging Rule |
|------|-------------|
| Email addresses | NEVER log |
| Display names | NEVER log |
| IP addresses | NEVER log (hash if needed for rate limiting) |
| Session tokens / cookies | NEVER log |
| Webhook secrets | NEVER log |
| Passwords / API keys | NEVER log |
| `family_id` | ALWAYS log (for debugging, not PII) |
| `parent_id` | ALWAYS log (for debugging, not PII) |
| HTTP method + path | ALWAYS log |
| HTTP status code | ALWAYS log |
| Request duration | ALWAYS log |

---

## §15 OpenAPI Generation

### §15.1 Generation Approach

OpenAPI spec generation uses `swaggo/swag`, which parses Go doc comments (annotations) on
handler functions and model structs to produce `openapi/spec.yaml`. `[CODING §6.1]`

### §15.2 Behavior

Annotations are added to handler functions and model structs using swaggo comment syntax:

```go
// @title Homegrown Academy API
// @version 0.1.0
// @description API for the Homegrown Academy homeschooling platform

package main
```

Handler annotations example:

```go
// HealthCheck godoc
// @Summary Health check
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func healthHandler(state *app.AppState) echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.JSON(http.StatusOK, HealthResponse{
			Status:  "ok",
			Version: version,
		})
	}
}
```

**Usage**: `swag init -g cmd/server/main.go -o openapi/`

**Key rules** `[CODING §6.1]`:
- MUST run after any change to Go API types in `models.go`
- MUST commit `openapi/spec.yaml` alongside the Go changes
- MUST NOT generate the spec at runtime — it is a build artifact

---

## §16 Local Development Environment

### §16.1 docker-compose.yml

```yaml
services:
  postgres:
    image: postgis/postgis:16-3.4
    environment:
      POSTGRES_USER: homegrown
      POSTGRES_PASSWORD: homegrown
      POSTGRES_DB: homegrown
    ports:
      - "5432:5432"
    volumes:
      - pg_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U homegrown"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

  kratos:
    image: oryd/kratos:v1.3
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DSN: postgres://homegrown:homegrown@postgres:5432/kratos?sslmode=disable
      SERVE_PUBLIC_BASE_URL: http://localhost:4433
      SERVE_ADMIN_BASE_URL: http://localhost:4434
    ports:
      - "4433:4433"   # Public API
      - "4434:4434"   # Admin API
    volumes:
      - ./kratos:/etc/kratos
    command: serve --config /etc/kratos/kratos.yml --dev --watch-courier
    healthcheck:
      test: ["CMD", "wget", "--spider", "--quiet", "http://localhost:4434/health/alive"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s

  kratos-migrate:
    image: oryd/kratos:v1.3
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DSN: postgres://homegrown:homegrown@postgres:5432/kratos?sslmode=disable
    volumes:
      - ./kratos:/etc/kratos
    command: migrate sql -e --yes --config /etc/kratos/kratos.yml

volumes:
  pg_data:
```

**Notes**:
- `postgis/postgis:16-3.4` includes PostgreSQL 16 with PostGIS pre-installed `[ARCH §2.5]`
- Kratos uses a separate `kratos` database on the same PostgreSQL instance
- `kratos-migrate` runs once at startup; `kratos` waits until migrations complete
- Health checks ensure dependent services are ready before startup

### §16.2 Kratos Dev Configuration

Minimal `kratos/kratos.yml` for local development. Full Kratos configuration (OIDC, MFA) is
documented in `[ARCH §6.1]` and `[IAM §7]`.

```yaml
version: v1.3.0

dsn: postgres://homegrown:homegrown@postgres:5432/kratos?sslmode=disable

serve:
  public:
    base_url: http://localhost:4433
    cors:
      enabled: true
      allowed_origins:
        - http://localhost:5173
      allowed_methods:
        - GET
        - POST
        - PUT
        - PATCH
        - DELETE
      allowed_headers:
        - Content-Type
        - Authorization
        - Cookie
      exposed_headers:
        - Set-Cookie
      allow_credentials: true
  admin:
    base_url: http://localhost:4434

selfservice:
  default_browser_return_url: http://localhost:5173/

  flows:
    registration:
      ui_url: http://localhost:5173/auth/registration
      after:
        password:
          hooks:
            - hook: web_hook
              config:
                url: http://host.docker.internal:3000/hooks/kratos/post-registration
                method: POST
                body: file:///etc/kratos/webhook-body.jsonnet
                auth:
                  type: api_key
                  config:
                    name: X-Webhook-Secret
                    value: dev-webhook-secret-change-in-production
                    in: header

    login:
      ui_url: http://localhost:5173/auth/login
      after:
        password:
          hooks:
            - hook: web_hook
              config:
                url: http://host.docker.internal:3000/hooks/kratos/post-login
                method: POST
                body: file:///etc/kratos/webhook-body.jsonnet
                auth:
                  type: api_key
                  config:
                    name: X-Webhook-Secret
                    value: dev-webhook-secret-change-in-production
                    in: header

    verification:
      enabled: true
      ui_url: http://localhost:5173/auth/verification

    recovery:
      enabled: true
      ui_url: http://localhost:5173/auth/recovery

  methods:
    password:
      enabled: true
      config:
        min_password_length: 10
        identifier_similarity_check_enabled: true

identity:
  default_schema_id: user
  schemas:
    - id: user
      url: file:///etc/kratos/identity.schema.json

session:
  lifespan: 720h
  cookie:
    same_site: Lax

courier:
  smtp:
    connection_uri: smtp://mailhog:1025/?skip_ssl_verify=true
```

### §16.3 Identity Schema

`kratos/identity.schema.json` — Defines the identity traits used by Kratos.

```json
{
  "$id": "https://homegrown.academy/identity.schema.json",
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Homegrown Academy User",
  "type": "object",
  "properties": {
    "traits": {
      "type": "object",
      "properties": {
        "email": {
          "type": "string",
          "format": "email",
          "title": "Email",
          "ory.sh/kratos": {
            "credentials": {
              "password": {
                "identifier": true
              }
            },
            "verification": {
              "via": "email"
            },
            "recovery": {
              "via": "email"
            }
          }
        },
        "name": {
          "type": "string",
          "title": "Name",
          "minLength": 1,
          "maxLength": 100
        }
      },
      "required": ["email", "name"],
      "additionalProperties": false
    }
  }
}
```

### §16.4 Dev Workflow

```bash
# 1. Start infrastructure
docker compose up -d

# 2. Wait for services (health checks ensure readiness)
docker compose ps  # verify all services are healthy

# 3. Start the Go API
go run ./cmd/server

# 4. Start the frontend dev server (in a separate terminal)
cd frontend && npm run dev

# 5. Access the app at http://localhost:5173
```

---

## §17 Dev Commands (`Makefile`)

A `Makefile` at project root provides all common development commands. `[CLAUDE.md —
"Development Commands" section]`

```makefile
# Homegrown Academy — Development Commands

.PHONY: default dev docker-up docker-down check lint test type-check \
        migrate db-reset openapi generate-types full-generate

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

# ─── Quality Gates ───────────────────────────────────────────────────

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

# ─── Database ────────────────────────────────────────────────────────

# Run pending database migrations (goose)
migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up

# Reset the database (drop + recreate + migrate)
db-reset:
	docker compose exec postgres psql -U homegrown -c "DROP DATABASE IF EXISTS homegrown;"
	docker compose exec postgres psql -U homegrown -c "CREATE DATABASE homegrown;"
	$(MAKE) migrate

# ─── Code Generation ────────────────────────────────────────────────

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

# ─── Security ────────────────────────────────────────────────────────

# Run vulnerability check
audit:
	govulncheck ./...
```

---

## §18 Frontend Shell

Minimum viable frontend files — just enough to pass `npm run type-check` and `npm run dev`.
Domain components are created by their domain specs.

### §18.1 index.html

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Homegrown Academy</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

### §18.2 main.tsx

```typescript
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { QueryClientProvider } from "@tanstack/react-query";
import { queryClient } from "./query-client";
import { App } from "./App";

const rootElement = document.getElementById("root");
if (!rootElement) throw new Error("Root element not found");

createRoot(rootElement).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>
);
```

### §18.3 App.tsx

```typescript
export function App() {
  return (
    <div>
      <h1>Homegrown Academy</h1>
      <p>App shell ready. Domain features will be added per domain spec.</p>
    </div>
  );
}
```

### §18.4 query-client.ts

```typescript
import { QueryClient } from "@tanstack/react-query";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60 * 5, // 5 minutes
      retry: 1,
    },
  },
});
```

### §18.5 api/client.ts

```typescript
const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "";

type RequestOptions = {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
};

export async function apiClient<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const { method = "GET", body, headers = {} } = options;

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
    credentials: "include", // Send cookies for Kratos session
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({
      error: { code: "unknown", message: "An error occurred" },
    }));
    throw error;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json() as Promise<T>;
}
```

**Note**: This is a minimal client. When domain specs add generated types from
`src/api/generated/`, hooks will use typed wrappers around `apiClient`. `[CODING §3.4]`

### §18.6 Student View Architecture `[S§8.6]`

#### Frontend Routing

The student view operates as a separate route tree, isolated from parent features:

```
/student                        — Student session root (redirect to /student/dashboard)
/student/dashboard              — Assignment list, due items, recent activity
/student/assignments            — Full assignment list with status filters
/student/quiz/:session_id       — Quiz-taking interface
/student/video/:video_def_id    — Video player interface
/student/read/:content_id       — Content viewer (PDF)
/student/sequence/:progress_id  — Sequence progression interface
```

The `/student` route tree uses a simplified layout:
- No navigation to social features, marketplace, messaging, groups, or events
- No access to account settings or family profile
- Shows only the current student's assigned content and progress
- Includes a "Return to Parent" button that ends the student session

#### Auth Context

The student view uses a `StudentSessionContext` alongside the existing `AuthContext`:

- `AuthContext` — populated from parent JWT/session (for parent routes)
- `StudentSessionContext` — populated from student session token (for `/student` routes)
  - Contains: `student_id`, `family_id`, `allowed_tool_slugs`, `expires_at`
  - Automatically redirects to parent login when session expires

Both contexts are provided at the app root level. Route guards check the appropriate context:
- `/student/*` routes require `StudentSessionContext`
- All other routes require `AuthContext`

#### Component Architecture

Student view components are a **constrained subset** of parent components:
- Quiz-taking component: reuses quiz rendering but omits admin/scoring UI
- Video player component: reuses player but omits creator/publisher controls
- Content viewer component: reuses PDF viewer with progress tracking
- Sequence view component: shows current position, completed items, and next item
- Assignment list component: simplified view of assigned content with status

Student components MUST NOT import or render:
- Social components (timeline, posts, comments, friends, groups, events)
- Marketplace components (browse, cart, checkout, reviews)
- Messaging components (DMs, conversations)
- Account/settings components
- Other students' data components

---

## §19 Implementation Checklist

Phase 1 items organized by dependency order. Each item maps to a section in this document.

### Phase 1 — Core Infrastructure (must complete before any domain)

#### Project Setup
- [ ] Initialize `go.mod` with all dependencies (§2.1)
- [ ] Create directory skeleton (§2.2)
- [ ] Create `frontend/package.json` and install dependencies (§2.3)
- [ ] Create `frontend/tsconfig.json` (§2.4)
- [ ] Create `frontend/vite.config.ts` (§2.5)
- [ ] Create `.env.example` (§3.3)
- [ ] Create `.gitignore` (include `.env`, `bin/`, `node_modules/`, etc.)

#### Configuration
- [ ] Implement `AppConfig` struct and `LoadConfig()` (§3.1, §3.2)
- [ ] Implement config error handling with `requiredEnv` / `envOrDefault` helpers
- [ ] Implement `Environment` type

#### Error Framework
- [ ] Implement `AppError` struct with all constructors (§6.1)
- [ ] Implement `HTTPErrorHandler` for Echo (§6.3)
- [ ] Implement `ValidationError` helper (§6.5)

#### Shared Types
- [ ] Implement newtype wrappers: `FamilyID`, `ParentID`, `StudentID`, `CreatorID` (§7.1)
- [ ] Implement `AuthContext` struct (§7.2)
- [ ] Implement `SubscriptionTier` type (§7.3)
- [ ] Implement `GetAuthContext` / `SetAuthContext` helpers (§7.2)

#### Family Scope
- [ ] Implement `FamilyScope` struct with unexported field (§8.1)
- [ ] Implement `NewFamilyScopeFromAuth` (§8.2)
- [ ] Implement `GetFamilyScope` helper (§8.3)

#### Database
- [ ] Implement `CreatePool()` (§9.1)
- [ ] Implement `ScopedTransaction()` (§9.2)
- [ ] Implement `UnscopedTransaction()` (§9.3)
- [ ] Create bootstrap migration with PostgreSQL extensions (§9.4)
- [ ] Set up goose migration runner in `cmd/server/main.go`

#### Redis
- [ ] Implement `CreateRedisClient()` (§10.1)
- [ ] Implement generic `RedisGet`, `RedisSet`, `RedisDelete` helpers (§10.2)
- [ ] Implement `RedisIncrementWithExpiry` (§10.3)

#### Event Bus
- [ ] Implement `DomainEvent` and `DomainEventHandler` interfaces (§11.1)
- [ ] Implement `EventBus` with `reflect.Type` dispatch (§11.2)
- [ ] Verify handler errors are logged but don't fail publish

#### Pagination
- [ ] Implement `PaginationParams` (§12.1)
- [ ] Implement `PaginatedResponse[T]` (§12.2)
- [ ] Implement cursor encode/decode (§12.3)

#### Middleware
- [ ] Implement auth middleware function (§13.1) — stub until IAM provides KratosAdapter
- [ ] Implement rate limiting middleware (§13.2)
- [ ] Implement `RequirePremium` extractor (§13.3)
- [ ] Implement `RequireCoppaConsent` extractor (§13.3)
- [ ] Stub `RequireCreator` extractor (depends on marketplace domain)

#### Application Wiring
- [ ] Implement `AppState` struct (§5.1)
- [ ] Implement `NewApp()` with router and middleware stack (§5.2)
- [ ] Implement health endpoint (§5.4)

#### Main Entrypoint
- [ ] Implement `main()` with startup sequence (§4.2)
- [ ] Implement graceful shutdown (§4.3)

#### Logging
- [ ] Implement `initLogger()` with JSON/text format switching (§14.1)
- [ ] Configure log level parsing (§14.2)

#### OpenAPI
- [ ] Add swaggo annotations to `cmd/server/main.go` (§15)
- [ ] Create empty `openapi/` directory

#### Local Dev
- [ ] Create `docker-compose.yml` (§16.1)
- [ ] Create Kratos dev config (§16.2)
- [ ] Create identity schema (§16.3)

#### Dev Commands
- [ ] Create `Makefile` with all commands (§17)
- [ ] Update `CLAUDE.md` "Development Commands" section to reference Makefile

#### Frontend Shell
- [ ] Create `index.html` (§18.1)
- [ ] Create `main.tsx` (§18.2)
- [ ] Create `App.tsx` (§18.3)
- [ ] Create `query-client.ts` (§18.4)
- [ ] Create `api/client.ts` (§18.5)
- [ ] Create empty `api/generated/`, `components/ui/`, `features/` directories

#### Quality Gates
- [ ] Verify `go build ./...` succeeds
- [ ] Verify `golangci-lint run` passes
- [ ] Verify `go test ./...` passes (health endpoint test)
- [ ] Verify `cd frontend && npm run type-check` passes
- [ ] Verify `cd frontend && npm run dev` starts without errors
- [ ] Verify `docker compose up -d` starts all services with healthy status

### Phase 2 — Deferred Items

These items are NOT specified in this document. They will be added when their consuming
domain requires them.

| Item | Owned By | When |
|------|----------|------|
| `RequirePrimaryParent` extractor | This spec (§13.3) | When IAM Phase 2 is implemented |
| Background job runner (asynq) | Consuming domain | When first domain needs async jobs |
| WebSocket infrastructure (gorilla/websocket) | `social::` | When messaging is implemented |
| Typesense integration | `search::` | When PG FTS is insufficient (§2.6) |
| Full CQRS read models | Consuming domain | When progressive optimization requires it |

---

## §20 01-iam.md Updates

The following changes MUST be made to `specs/domains/01-iam.md` to avoid duplication with
this document and establish clear ownership boundaries.

### §20.1 Remove PostgreSQL Extensions from IAM Migration

**Current** (IAM §3.1, lines 76-78):

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "postgis";
```

**Change**: Remove these three lines from the IAM migration. Add a comment referencing the
bootstrap migration:

```sql
-- PostgreSQL extensions (uuid-ossp, pgcrypto, postgis, pg_trgm) are installed
-- by the bootstrap migration (00-core §9.4). They are available to all domains.
```

### §20.2 Update IAM §1 Overview — Ownership Clarification

**Current** (IAM §1, line 21-22):

> **What IAM owns**: Family accounts, parent users, student profiles, COPPA consent tracking,
> co-parent invitations, role-based extractors (`AuthContext`, `FamilyScope`, `RequirePremium`,
> `RequireCreator`, `RequireCoppaConsent`, `RequirePrimaryParent`), Kratos webhook handlers.

**Change**: Update to clarify ownership split:

> **What IAM owns**: Family accounts, parent users, student profiles, COPPA consent tracking,
> co-parent invitations, Kratos webhook handlers, auth middleware *implementation*
> (calls `KratosAdapter`), `AuthContext` *population* logic.
>
> **Shared infrastructure** (defined in 00-core, consumed by IAM and all other domains):
> `AuthContext` type (00-core §7.2), `FamilyScope` type (00-core §8), `AppError` base
> variants (00-core §6), role extractors `RequirePremium`, `RequireCreator`,
> `RequireCoppaConsent`, `RequirePrimaryParent` (00-core §13.3).

### §20.3 Update IAM §11 Middleware & Extractors — Back-references

**Current** (IAM §11): Defines `AuthContext`, `FamilyScope`, `RequirePremium`,
`RequireCreator`, `RequireCoppaConsent`, `RequirePrimaryParent` with full code samples.

**Change**: Replace the implementation details with back-references. Keep IAM-specific
behavioral documentation (what IAM populates), remove shared type definitions:

> ### §11 Middleware & Extractors
>
> The shared types and extractors that IAM depends on and populates are defined in
> **00-core §7** (types), **00-core §8** (FamilyScope), and **00-core §13** (middleware
> and extractors). This section documents IAM-specific behavior only.
>
> #### §11.1 AuthContext Population
>
> IAM owns the *population* of `AuthContext` (type defined in 00-core §7.2). The auth
> middleware (`internal/middleware/auth.go`, defined in 00-core §13.1) calls IAM's
> `KratosAdapter.ValidateSession()` and queries IAM repositories to build the
> `AuthContext`. The flow is:
>
> 1. Auth middleware extracts session cookie
> 2. Calls `KratosAdapter.ValidateSession()` (IAM §7)
> 3. Calls `ParentRepository.FindByKratosID()` (IAM §6)
> 4. Calls `FamilyRepository.FindByID()` (IAM §6)
> 5. Constructs `AuthContext` from parent + family data
>
> #### §11.2 COPPA Consent Check
>
> The `RequireCoppaConsent` extractor (00-core §13.3) checks
> `AuthContext.CoppaConsentStatus` which IAM populates from `iam_families`.
>
> [Keep existing §11.3-§11.6 behavioral docs but remove code samples that duplicate 00-core]

### §20.4 Update IAM §12 Error Types — Back-reference

**Current** (IAM §12): Defines `IamError` type and its mapping to HTTP statuses.

**Change**: Add a reference to the base `AppError`:

> `IamError` maps to `AppError` (defined in 00-core §6) via the `ToAppError()` conversion
> function. See 00-core §6.4 for the conversion pattern.

This is an additive note — do NOT remove the `IamError` definition or its HTTP mapping table.
`IamError` is IAM-specific; only the `AppError` base type moves to core.

### §20.5 Update IAM §16 Implementation Checklist — Remove Shared Items

**Current** (IAM §16, "Shared Infrastructure" section, lines 1547-1553):

```
#### Shared Infrastructure
- [ ] Implement `FamilyScope` type in `internal/shared/family_scope.go`
- [ ] Implement `AuthContext` struct and auth middleware in `internal/middleware/auth.go`
- [ ] Implement `FamilyScope` extractor (from AuthContext)
- [ ] Implement `RequirePremium` extractor
- [ ] Implement `RequireCreator` extractor
- [ ] Implement `RequireCoppaConsent` middleware
- [ ] Implement `AppError` type with IAM variants in `internal/shared/error.go`
```

**Change**: Replace with a dependency reference:

```
#### Shared Infrastructure (prerequisite — see 00-core)
- [ ] Verify 00-core §19 checklist is complete (AppError, AuthContext, FamilyScope,
      extractors, middleware, DB pool, Redis pool, EventBus)
- [ ] Implement `ToAppError()` conversion for IamError (00-core §6.4 pattern)
```

### §20.6 Add CoppaConsentStatus to AuthContext Note

Add a note in IAM §11.1 or §9 that `CoppaConsentStatus` SHOULD be included in
`AuthContext` to enable the `RequireCoppaConsent` extractor to work without an extra DB query.
This means the auth middleware's family lookup (already happening) also reads
`coppa_consent_status` and includes it in the `AuthContext`.

This change requires adding `CoppaConsentStatus` to the `AuthContext`
struct in 00-core §7.2. However, since `CoppaConsentStatus` is an IAM-owned type, this
creates a dependency from shared types to IAM. Two resolution approaches:

1. **Preferred**: Store the consent status as a `string` in `AuthContext` and let the
   `RequireCoppaConsent` extractor match on known values. No cross-domain type dependency.
2. **Alternative**: Move `CoppaConsentStatus` to `internal/shared/types.go` since it's needed
   by shared infrastructure.

Decision: Use approach 1 (string). Add to `AuthContext`:

```go
CoppaConsentStatus string `json:"coppa_consent_status"` // "consented", "re_verified", etc.
```

---

## Appendix A: Cross-Reference Matrix

How this document relates to the existing spec hierarchy.

| This Section | ARCH Reference | CODING Reference | Fills Gap? |
|-------------|----------------|------------------|------------|
| §2 Scaffolding | §3.4 (partial) | — | Yes — ARCH lists dirs but omits files |
| §3 Config | — | — | Yes — completely unspecified |
| §4 Entrypoint | — | — | Yes — completely unspecified |
| §5 App Wiring | §3.3 (request flow) | §8.2 (wiring location) | Yes — ARCH shows flow, not wiring code |
| §6 Error Framework | §4.2 (shared kernel) | §2.2 (error rules) | Partial — rules existed, implementation didn't |
| §7 Shared Types | §6.2 (AuthContext) | §2.3 (type rules) | Partial — AuthContext was in ARCH, newtypes weren't |
| §8 Family Scope | §1.5 (privacy) | §2.4 (FamilyScope rule) | Partial — concept existed, implementation spec didn't |
| §9 Database | §2.5 (PostgreSQL) | §4.1-4.3 (DB rules) | Yes — pool creation, RLS helper unspecified |
| §10 Redis | §2.7 (Redis uses) | — | Yes — completely unspecified |
| §11 Event Bus | §4.6 (signatures) | §8.4 (event rules) | Partial — signatures existed, internals didn't |
| §12 Pagination | §4.2 (shared kernel) | — | Yes — mentioned in shared kernel list, never specified |
| §13 Middleware | §3.3 (request flow), §6.2-6.5 | §2.6 (handler pattern) | Partial — auth middleware in ARCH, rate limit/extractors weren't |
| §14 Logging | — | §5.2 (PII rules) | Yes — PII rules existed, logging setup didn't |
| §15 OpenAPI | — | §6.1 (generation rules) | Yes — rules existed, generation spec didn't |
| §16 Local Dev | — | — | Yes — completely unspecified |
| §17 Dev Commands | — | §7.2 (quality gates) | Yes — gates existed, commands didn't |
| §18 Frontend Shell | §2.3 (React stack) | §3 (TS rules) | Yes — stack chosen, files unspecified |

## Appendix B: Naming Conflict Resolution

| Item | ARCH Uses | CODING Uses | This Spec Uses | Rationale |
|------|-----------|-------------|----------------|-----------|
| Application error type | `ApiError` | `AppError` | `AppError` | CODING_STANDARDS is authoritative for naming (§1) |

All code and documentation produced after this document MUST use `AppError`. ARCHITECTURE.md
code samples using `ApiError` should be read as `AppError` — they are not normative.
