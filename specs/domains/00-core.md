# Domain Spec 00 — Core Infrastructure

## §1 Overview

This document specifies the **shared infrastructure** that must exist before any domain module
can be implemented. It is not a domain — it produces no user-facing features. It produces a
compilable, runnable project skeleton with database connectivity, configuration, error handling,
middleware, and all shared types in place.

| Attribute | Value |
|-----------|-------|
| **Purpose** | Foundation for all 14 domain modules |
| **Produces** | A `cargo build`-able binary, passing `cargo clippy -- -D warnings` and `cargo test` |
| **Frontend** | Minimum shell passing `npm run type-check` and `npm run dev` |
| **Prerequisite for** | Every domain spec (01-iam through 14-ai) |

### What core infrastructure owns

- Project scaffolding (`Cargo.toml`, directory skeleton, frontend `package.json`)
- Application entrypoint (`src/main.rs`)
- Application wiring (`src/app.rs`) — `AppState`, router composition, middleware ordering
- Configuration (`src/config.rs`) — `AppConfig` struct, env var loading
- Shared kernel (`src/shared/`) — error framework, types, family scope, DB/Redis helpers,
  event bus, pagination
- Middleware stack (`src/middleware/`) — auth, rate limiting, role extractors
- Health endpoint (`GET /health`)
- OpenAPI generation binary (`src/bin/openapi_gen.rs`)
- Bootstrap database migration (PostgreSQL extensions)
- Local development environment (`docker-compose.yml`, Kratos dev config)
- Development commands (`justfile`)
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

### §2.1 Cargo.toml

Workspace-level configuration with all Phase 1 dependencies. Two binary targets.

```toml
[package]
name = "homegrown-academy"
version = "0.1.0"
edition = "2021"
rust-version = "1.80"

[[bin]]
name = "homegrown-academy"
path = "src/main.rs"

[[bin]]
name = "openapi-gen"
path = "src/bin/openapi_gen.rs"

[dependencies]
# Web framework
axum = { version = "0.8", features = ["macros", "ws"] }
axum-extra = { version = "0.10", features = ["typed-header", "cookie"] }
tokio = { version = "1", features = ["full"] }
tower = { version = "0.5", features = ["util", "timeout"] }
tower-http = { version = "0.6", features = ["cors", "trace", "set-header"] }
hyper = { version = "1", features = ["full"] }

# Database
sea-orm = { version = "1", features = [
    "sqlx-postgres",
    "runtime-tokio-rustls",
    "macros",
    "with-uuid",
    "with-chrono",
    "with-json",
] }
sea-orm-migration = { version = "1" }

# Serialization & validation
serde = { version = "1", features = ["derive"] }
serde_json = "1"
validator = { version = "0.19", features = ["derive"] }

# Error handling
thiserror = "2"
anyhow = "1"

# Logging & tracing
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter", "json"] }
tracing-appender = "0.2"

# OpenAPI
utoipa = { version = "5", features = ["axum_extras", "uuid", "chrono"] }
utoipa-axum = "0.2"

# Types
uuid = { version = "1", features = ["v4", "serde"] }
chrono = { version = "0.4", features = ["serde"] }

# Redis
redis = { version = "0.27", features = ["tokio-comp", "connection-manager"] }

# Configuration
dotenvy = "0.15"

# HTTP client (for Kratos adapter)
reqwest = { version = "0.12", features = ["json", "cookies"] }

# Crypto (for invite tokens, webhook signatures)
base64 = "0.22"
hmac = "0.12"
sha2 = "0.10"
rand = "0.8"

# Async trait support
async-trait = "0.1"

# HTML sanitization [CODING §5.2]
ammonia = "4"

# Sentry (optional error tracking) [ARCH §2.14]
sentry = { version = "0.35", features = ["tower", "tracing"] }

[dev-dependencies]
tower = { version = "0.5", features = ["util"] }
axum-test = "16"
```

**Note**: Exact version numbers SHOULD be updated to latest stable at implementation time.
The versions above represent the minimum compatible release.

### §2.2 Directory Skeleton

Completes the structure outlined in `[ARCH §3.4]` with all files that ARCHITECTURE.md omits.
Files marked `(core)` are created by this spec. Files marked `(domain)` are created by the
owning domain spec.

```
homegrown-academy/
├── Cargo.toml                          (core)
├── justfile                            (core) §17
├── .env.example                        (core) §3
├── docker-compose.yml                  (core) §16
├── kratos/                             (core) §16
│   ├── kratos.yml
│   └── identity.schema.json
├── openapi/
│   └── spec.yaml                       (generated)
├── migration/
│   └── src/
│       ├── lib.rs                      (core) — migration registry
│       ├── m000000_000000_bootstrap.rs (core) §9 — PG extensions
│       └── m{timestamp}_*.rs           (domain) — per-domain migrations
├── src/
│   ├── main.rs                         (core) §4
│   ├── app.rs                          (core) §5
│   ├── config.rs                       (core) §3
│   ├── lib.rs                          (core) — re-exports for integration tests
│   ├── bin/
│   │   └── openapi_gen.rs              (core) §15
│   ├── shared/
│   │   ├── mod.rs                      (core)
│   │   ├── error.rs                    (core) §6
│   │   ├── types.rs                    (core) §7
│   │   ├── family_scope.rs             (core) §8
│   │   ├── db.rs                       (core) §9
│   │   ├── redis.rs                    (core) §10
│   │   ├── events.rs                   (core) §11
│   │   └── pagination.rs              (core) §12
│   ├── middleware/
│   │   ├── mod.rs                      (core)
│   │   ├── auth.rs                     (core) §13
│   │   ├── rate_limit.rs               (core) §13
│   │   └── extractors.rs              (core) §13
│   └── domains/                        (domain modules added incrementally)
│       ├── mod.rs                      (core) — empty initially
│       ├── iam/                        (01-iam)
│       ├── method/                     (02-method)
│       ├── media/                      (03-media)
│       └── ...                         (04-14)
└── frontend/                           (core) §18
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

## §3 Configuration (`src/config.rs`)

### §3.1 AppConfig Struct

All environment variables typed and documented. Loaded at startup, immutable thereafter.

```rust
use std::net::SocketAddr;

#[derive(Debug, Clone)]
pub struct AppConfig {
    // ─── Database ───────────────────────────────────────────────────
    /// PostgreSQL connection string.
    /// Example: `postgres://user:pass@localhost:5432/homegrown`
    pub database_url: String,

    /// Maximum connections in the SeaORM pool. Default: 10.
    pub database_max_connections: u32,

    // ─── Redis ──────────────────────────────────────────────────────
    /// Redis connection string.
    /// Example: `redis://localhost:6379`
    pub redis_url: String,

    // ─── Ory Kratos ─────────────────────────────────────────────────
    /// Kratos Admin API URL (internal sidecar, never public).
    /// Example: `http://kratos:4434`
    pub kratos_admin_url: String,

    /// Kratos Public API URL (browser-facing, session validation).
    /// Example: `http://kratos:4433`
    pub kratos_public_url: String,

    /// Shared secret for Kratos webhook signature validation.
    pub kratos_webhook_secret: String,

    // ─── CORS ───────────────────────────────────────────────────────
    /// Comma-separated list of allowed origins.
    /// Example: `http://localhost:5173,https://app.homegrown.academy`
    pub cors_allowed_origins: Vec<String>,

    // ─── Server ─────────────────────────────────────────────────────
    /// Host to bind to. Default: `0.0.0.0`.
    pub server_host: String,

    /// Port to bind to. Default: `3000`.
    pub server_port: u16,

    // ─── Logging ────────────────────────────────────────────────────
    /// Tracing filter directive. Default: `info`.
    /// Example: `homegrown_academy=debug,tower_http=debug,sea_orm=warn`
    pub log_level: String,

    // ─── Observability ──────────────────────────────────────────────
    /// Sentry DSN. Optional — omit to disable Sentry. [ARCH §2.14]
    pub sentry_dsn: Option<String>,

    // ─── Environment ────────────────────────────────────────────────
    /// Runtime environment. Controls log format, debug features, etc.
    pub environment: Environment,
}

#[derive(Debug, Clone, PartialEq)]
pub enum Environment {
    Development,
    Staging,
    Production,
}
```

### §3.2 Loading

`AppConfig::from_env()` loads values from environment variables with `dotenvy` fallback
for local development.

```rust
impl AppConfig {
    pub fn from_env() -> Result<Self, AppConfigError> {
        // Load .env file if it exists (dev only, not required)
        dotenvy::dotenv().ok();

        Ok(Self {
            database_url: required_env("DATABASE_URL")?,
            database_max_connections: optional_env("DATABASE_MAX_CONNECTIONS")?.unwrap_or(10),
            redis_url: required_env("REDIS_URL")?,
            kratos_admin_url: required_env("KRATOS_ADMIN_URL")?,
            kratos_public_url: required_env("KRATOS_PUBLIC_URL")?,
            kratos_webhook_secret: required_env("KRATOS_WEBHOOK_SECRET")?,
            cors_allowed_origins: required_env("CORS_ALLOWED_ORIGINS")?
                .split(',')
                .map(|s| s.trim().to_string())
                .collect(),
            server_host: optional_env("SERVER_HOST")?.unwrap_or_else(|| "0.0.0.0".into()),
            server_port: optional_env("SERVER_PORT")?.unwrap_or(3000),
            log_level: optional_env("LOG_LEVEL")?.unwrap_or_else(|| "info".into()),
            sentry_dsn: optional_env("SENTRY_DSN")?,
            environment: match optional_env::<String>("ENVIRONMENT")?
                .unwrap_or_else(|| "development".into())
                .as_str()
            {
                "production" => Environment::Production,
                "staging" => Environment::Staging,
                _ => Environment::Development,
            },
        })
    }
}
```

`required_env` returns `AppConfigError::Missing` if absent. `optional_env` returns `None` if
absent, `AppConfigError::InvalidValue` if present but unparseable.

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
LOG_LEVEL=homegrown_academy=debug,tower_http=debug,sea_orm=warn

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

## §4 Application Entrypoint (`src/main.rs`)

### §4.1 Runtime

Tokio multi-threaded runtime with default worker count (one per logical CPU core).

```rust
#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Startup sequence — order matters
    // (1) Init tracing
    // (2) Load config
    // (3) Create DB pool
    // (4) Run migrations
    // (5) Create Redis pool
    // (6) Init EventBus + register subscriptions
    // (7) Wire AppState
    // (8) Build Axum app
    // (9) Bind listener
    // (10) Serve with graceful shutdown
}
```

### §4.2 Startup Sequence

The startup sequence MUST execute in the order below. Each step depends on the output of the
previous step.

| Step | Action | Depends On | Failure Behavior |
|------|--------|------------|------------------|
| 1 | `init_tracing(&config)` | None (uses env defaults until config loaded) | Fatal — exit |
| 2 | `AppConfig::from_env()` | — | Fatal — exit with missing-env message |
| 3 | `create_db_pool(&config)` | Config | Fatal — exit |
| 4 | `Migrator::up(&db, None)` | DB pool | Fatal — exit with migration error |
| 5 | `create_redis_pool(&config)` | Config | Fatal — exit |
| 6 | `EventBus::new()` + subscription registration | Services | Fatal — exit |
| 7 | `AppState::new(db, redis, event_bus, services...)` | All above | Infallible |
| 8 | `create_app(state)` | AppState | Infallible |
| 9 | `TcpListener::bind(addr)` | Config | Fatal — exit |
| 10 | `axum::serve(listener, app).with_graceful_shutdown(signal)` | All above | Runs until signal |

**Tracing bootstrap**: Step 1 initializes with defaults first (so steps 2-5 can log). After
config is loaded, the tracing subscriber is reconfigured with the config-specified log level.

### §4.3 Graceful Shutdown

The server listens for SIGTERM (ECS sends this on task stop) and SIGINT (Ctrl-C for local
development). On signal reception:

1. Stop accepting new connections
2. Wait up to 30 seconds for in-flight requests to complete
3. Close database pool connections
4. Close Redis connections
5. Exit with code 0

```rust
async fn shutdown_signal() {
    let ctrl_c = tokio::signal::ctrl_c();
    let mut sigterm = tokio::signal::unix::signal(
        tokio::signal::unix::SignalKind::terminate(),
    ).expect("failed to install SIGTERM handler");

    tokio::select! {
        _ = ctrl_c => { tracing::info!("received Ctrl-C, shutting down"); }
        _ = sigterm.recv() => { tracing::info!("received SIGTERM, shutting down"); }
    }
}
```

---

## §5 Application Wiring (`src/app.rs`)

### §5.1 AppState

The central state struct passed to all Axum handlers via `State<AppState>`. Contains shared
infrastructure and all domain service trait objects.

```rust
use sea_orm::DatabaseConnection;
use std::sync::Arc;

#[derive(Clone)]
pub struct AppState {
    // ─── Infrastructure ─────────────────────────────────────────────
    pub db: DatabaseConnection,
    pub redis: RedisPool,
    pub event_bus: Arc<EventBus>,
    pub config: Arc<AppConfig>,

    // ─── Domain Services (added incrementally as domains are built) ─
    // pub iam: Arc<dyn IamService>,
    // pub method: Arc<dyn MethodologyService>,
    // pub social: Arc<dyn SocialService>,
    // ... etc.
}
```

Domain service fields are uncommented as each domain is implemented. The core skeleton
compiles with infrastructure fields only.

### §5.2 Router Composition

`create_app(state) -> Router` builds the Axum router with middleware layering and route groups.

```rust
pub fn create_app(state: AppState) -> Router {
    let public_routes = Router::new()
        .route("/health", get(health_handler));

    let webhook_routes = Router::new();
        // Domain webhooks added here (e.g., /hooks/kratos/*)

    let authenticated_routes = Router::new();
        // Domain routes added here, all behind auth middleware

    Router::new()
        .merge(public_routes)
        .nest("/hooks", webhook_routes)
        .merge(
            authenticated_routes
                .layer(middleware::from_fn_with_state(
                    state.clone(),
                    auth_middleware,
                ))
        )
        // ─── Middleware stack (outermost applied first) ──────────────
        .layer(TraceLayer::new_for_http())
        .layer(security_headers_layer())
        .layer(cors_layer(&state.config))
        .with_state(state)
}
```

### §5.3 Middleware Stack Ordering

Layers are applied outermost-first. The request passes through them top-to-bottom:

| Order | Layer | Scope | Purpose |
|-------|-------|-------|---------|
| 1 | `TraceLayer` | All routes | Request/response logging with timing |
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

The version string is set from `env!("CARGO_PKG_VERSION")` at compile time. This endpoint
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

## §6 Error Framework (`src/shared/error.rs`)

### §6.1 AppError Enum

The application-wide error type. All domain errors convert to `AppError` before reaching
the handler return type. `[CODING §2.2]`

```rust
use axum::http::StatusCode;
use axum::response::{IntoResponse, Response};

#[derive(Debug, thiserror::Error)]
pub enum AppError {
    #[error("not found")]
    NotFound,

    #[error("unauthorized")]
    Unauthorized,

    #[error("forbidden")]
    Forbidden,

    #[error("premium subscription required")]
    PremiumRequired,

    #[error("COPPA consent required")]
    CoppaConsentRequired,

    #[error("validation error: {0}")]
    Validation(String),

    #[error("conflict: {0}")]
    Conflict(String),

    #[error("rate limited")]
    RateLimited,

    #[error("bad request: {0}")]
    BadRequest(String),

    #[error("internal error")]
    Internal(#[from] anyhow::Error),

    #[error("database error")]
    Database(#[from] sea_orm::DbErr),
}
```

### §6.2 HTTP Status Mapping

| Variant | HTTP Status | JSON `code` |
|---------|-------------|-------------|
| `NotFound` | 404 Not Found | `not_found` |
| `Unauthorized` | 401 Unauthorized | `unauthorized` |
| `Forbidden` | 403 Forbidden | `forbidden` |
| `PremiumRequired` | 402 Payment Required | `premium_required` |
| `CoppaConsentRequired` | 403 Forbidden | `coppa_consent_required` |
| `Validation(msg)` | 422 Unprocessable Entity | `validation_error` |
| `Conflict(msg)` | 409 Conflict | `conflict` |
| `RateLimited` | 429 Too Many Requests | `rate_limited` |
| `BadRequest(msg)` | 400 Bad Request | `bad_request` |
| `Internal(err)` | 500 Internal Server Error | `internal_error` |
| `Database(err)` | 500 Internal Server Error | `internal_error` |

### §6.3 IntoResponse Implementation

```rust
impl IntoResponse for AppError {
    fn into_response(self) -> Response {
        let (status, code, message) = match &self {
            AppError::NotFound => (StatusCode::NOT_FOUND, "not_found", "Resource not found"),
            AppError::Unauthorized => (StatusCode::UNAUTHORIZED, "unauthorized", "Authentication required"),
            AppError::Forbidden => (StatusCode::FORBIDDEN, "forbidden", "Access denied"),
            AppError::PremiumRequired => (StatusCode::PAYMENT_REQUIRED, "premium_required", "Premium subscription required"),
            AppError::CoppaConsentRequired => (StatusCode::FORBIDDEN, "coppa_consent_required", "COPPA parental consent required"),
            AppError::Validation(msg) => (StatusCode::UNPROCESSABLE_ENTITY, "validation_error", msg.as_str()),
            AppError::Conflict(msg) => (StatusCode::CONFLICT, "conflict", msg.as_str()),
            AppError::RateLimited => (StatusCode::TOO_MANY_REQUESTS, "rate_limited", "Rate limit exceeded"),
            AppError::BadRequest(msg) => (StatusCode::BAD_REQUEST, "bad_request", msg.as_str()),
            AppError::Internal(err) => {
                tracing::error!(error = %err, "internal server error");
                (StatusCode::INTERNAL_SERVER_ERROR, "internal_error", "An internal error occurred")
            }
            AppError::Database(err) => {
                tracing::error!(error = %err, "database error");
                (StatusCode::INTERNAL_SERVER_ERROR, "internal_error", "An internal error occurred")
            }
        };

        let body = serde_json::json!({
            "error": {
                "code": code,
                "message": message,
            }
        });

        (status, axum::Json(body)).into_response()
    }
}
```

**Key behavior**:
- `Internal` and `Database` variants log the actual error via `tracing::error!` but return
  only `"An internal error occurred"` to the client. `[CODING §2.2, §5.2]`
- Validation messages from `validator` are user-facing and included in the response.
- The `code` field provides a machine-readable error identifier for the frontend.

### §6.4 Domain Error Conversion Pattern

Each domain defines its own error enum (e.g., `IamError`) and implements
`From<DomainError> for AppError`. This pattern keeps domain errors specific while converging
to a single HTTP error type.

```rust
// Example pattern — each domain implements this
impl From<IamError> for AppError {
    fn from(err: IamError) -> Self {
        match err {
            IamError::FamilyNotFound => AppError::NotFound,
            IamError::ParentNotFound => AppError::NotFound,
            IamError::NotPrimaryParent => AppError::Forbidden,
            IamError::PremiumRequired => AppError::PremiumRequired,
            IamError::CoppaConsentRequired => AppError::CoppaConsentRequired,
            IamError::InvalidConsentTransition { .. } => {
                AppError::Validation(err.to_string())
            }
            IamError::ParentAlreadyInFamily => {
                AppError::Conflict(err.to_string())
            }
            IamError::DatabaseError(db_err) => AppError::Database(db_err),
            IamError::KratosError => AppError::Internal(anyhow::anyhow!("Kratos error")),
            // ... other variants
        }
    }
}
```

### §6.5 Validator Integration

```rust
impl From<validator::ValidationErrors> for AppError {
    fn from(errors: validator::ValidationErrors) -> Self {
        AppError::Validation(errors.to_string())
    }
}
```

This allows handlers to use `cmd.validate()?` and have validation errors automatically
map to 422 responses.

---

## §7 Shared Types (`src/shared/types.rs`)

### §7.1 Newtype Wrappers

Type-safe wrappers for UUID identifiers. Prevents accidentally passing a `FamilyId` where a
`ParentId` is expected. `[ARCH §4.2]`

```rust
use serde::{Deserialize, Serialize};
use std::fmt;
use std::ops::Deref;
use utoipa::ToSchema;
use uuid::Uuid;

macro_rules! uuid_newtype {
    ($name:ident) => {
        #[derive(Debug, Clone, Copy, PartialEq, Eq, Hash, Serialize, Deserialize, ToSchema)]
        #[serde(transparent)]
        pub struct $name(Uuid);

        impl $name {
            pub fn new(id: Uuid) -> Self {
                Self(id)
            }
        }

        impl Deref for $name {
            type Target = Uuid;
            fn deref(&self) -> &Uuid {
                &self.0
            }
        }

        impl fmt::Display for $name {
            fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
                self.0.fmt(f)
            }
        }

        impl From<Uuid> for $name {
            fn from(id: Uuid) -> Self {
                Self(id)
            }
        }

        impl From<$name> for Uuid {
            fn from(id: $name) -> Self {
                id.0
            }
        }
    };
}

uuid_newtype!(FamilyId);
uuid_newtype!(ParentId);
uuid_newtype!(StudentId);
uuid_newtype!(CreatorId);
```

### §7.2 AuthContext

The authoritative type definition for `AuthContext`. Consolidates `[ARCH §6.2]` and
`[IAM §11.1]`.

```rust
/// Authenticated user context, inserted into request extensions by auth middleware.
/// Consumed by every authenticated handler.
#[derive(Clone, Debug)]
pub struct AuthContext {
    pub parent_id: Uuid,
    pub family_id: Uuid,
    pub kratos_identity_id: Uuid,
    pub is_primary_parent: bool,
    pub subscription_tier: SubscriptionTier,
    pub email: String,  // NOT logged — PII [CODING §5.2]
}

#[derive(Clone, Debug, PartialEq)]
pub enum SubscriptionTier {
    Free,
    Premium,
}
```

**`AuthContext` FromRequestParts impl**: Extracts `AuthContext` from Axum request extensions.
The auth middleware inserts it; this extractor retrieves it.

```rust
#[axum::async_trait]
impl<S> axum::extract::FromRequestParts<S> for AuthContext
where
    S: Send + Sync,
{
    type Rejection = AppError;

    async fn from_request_parts(
        parts: &mut axum::http::request::Parts,
        _state: &S,
    ) -> Result<Self, Self::Rejection> {
        parts
            .extensions
            .get::<AuthContext>()
            .cloned()
            .ok_or(AppError::Unauthorized)
    }
}
```

### §7.3 SubscriptionTier

Used by `AuthContext` and permission extractors. Matches the `subscription_tier` column on
`iam_families`.

```rust
impl SubscriptionTier {
    pub fn from_str(s: &str) -> Self {
        match s {
            "premium" => SubscriptionTier::Premium,
            _ => SubscriptionTier::Free,
        }
    }
}
```

---

## §8 Family Scope (`src/shared/family_scope.rs`)

### §8.1 FamilyScope Type

`FamilyScope` wraps a `family_id` and enforces that every repository query includes a family
filter. The `family_id` field is private — callers can read it but cannot construct a
`FamilyScope` outside of the designated paths. `[ARCH §1.5, CODING §2.4]`

```rust
use uuid::Uuid;

/// Wraps a family_id for privacy-enforcing database queries.
///
/// Private constructor ensures FamilyScope can only be created from:
/// 1. AuthContext (via From impl) — the normal authenticated path
/// 2. pub(crate) constructor — for auth middleware and registration flows
#[derive(Clone, Debug)]
pub struct FamilyScope {
    family_id: Uuid,
}

impl FamilyScope {
    /// Create a FamilyScope from a raw family_id.
    /// Restricted to crate-internal use (auth middleware, registration).
    pub(crate) fn new(family_id: Uuid) -> Self {
        Self { family_id }
    }

    /// Read the family_id. The only public access to the wrapped value.
    pub fn family_id(&self) -> Uuid {
        self.family_id
    }
}
```

### §8.2 From<&AuthContext>

```rust
impl From<&AuthContext> for FamilyScope {
    fn from(auth: &AuthContext) -> Self {
        FamilyScope::new(auth.family_id)
    }
}
```

### §8.3 FromRequestParts

```rust
#[axum::async_trait]
impl<S> axum::extract::FromRequestParts<S> for FamilyScope
where
    S: Send + Sync,
{
    type Rejection = AppError;

    async fn from_request_parts(
        parts: &mut axum::http::request::Parts,
        _state: &S,
    ) -> Result<Self, Self::Rejection> {
        let auth = parts
            .extensions
            .get::<AuthContext>()
            .ok_or(AppError::Unauthorized)?;
        Ok(FamilyScope::from(auth))
    }
}
```

### §8.4 RLS Integration

`FamilyScope` connects to the database RLS layer via `scoped_transaction()` (§9.2). When a
repository executes a query inside a scoped transaction, PostgreSQL's `app.current_family_id`
setting is automatically applied, and RLS policies enforce family isolation at the database
level as defense-in-depth.

---

## §9 Database Infrastructure (`src/shared/db.rs`)

### §9.1 Pool Management

```rust
use sea_orm::{ConnectOptions, Database, DatabaseConnection};

pub type DbPool = DatabaseConnection;

pub async fn create_pool(config: &AppConfig) -> Result<DbPool, sea_orm::DbErr> {
    let mut opts = ConnectOptions::new(&config.database_url);
    opts.max_connections(config.database_max_connections)
        .min_connections(1)
        .sqlx_logging(false);  // Use tracing, not sqlx's built-in logging

    Database::connect(opts).await
}
```

### §9.2 Scoped Transactions

`scoped_transaction` begins a database transaction, sets `SET LOCAL app.current_family_id`
for RLS enforcement, executes a closure, and commits. The `SET LOCAL` value is automatically
cleared on commit or rollback.

```rust
use sea_orm::{DatabaseConnection, TransactionTrait, ConnectionTrait, Statement};

/// Execute a closure within a family-scoped transaction.
/// Sets `app.current_family_id` for RLS enforcement.
pub async fn scoped_transaction<F, Fut, T>(
    db: &DatabaseConnection,
    scope: &FamilyScope,
    f: F,
) -> Result<T, AppError>
where
    F: FnOnce(&sea_orm::DatabaseTransaction) -> Fut,
    Fut: std::future::Future<Output = Result<T, AppError>>,
{
    let txn = db.begin().await?;

    txn.execute(Statement::from_string(
        sea_orm::DatabaseBackend::Postgres,
        format!(
            "SET LOCAL app.current_family_id = '{}'",
            scope.family_id()
        ),
    ))
    .await?;

    let result = f(&txn).await?;
    txn.commit().await?;
    Ok(result)
}
```

### §9.3 Unscoped Transactions

For operations that intentionally bypass family scope. Each call site MUST document why.

```rust
/// Execute a closure in a transaction WITHOUT family scope.
///
/// ONLY for:
/// - Auth middleware lookups (FamilyScope not yet constructed)
/// - Registration webhooks (family does not exist yet)
/// - Background cleanup jobs (cross-family by design)
///
/// Every call site MUST have a comment explaining why unscoped access is required.
pub async fn unscoped_transaction<F, Fut, T>(
    db: &DatabaseConnection,
    f: F,
) -> Result<T, AppError>
where
    F: FnOnce(&sea_orm::DatabaseTransaction) -> Fut,
    Fut: std::future::Future<Output = Result<T, AppError>>,
{
    let txn = db.begin().await?;
    let result = f(&txn).await?;
    txn.commit().await?;
    Ok(result)
}
```

### §9.4 Bootstrap Migration

The first migration (`m000000_000000_bootstrap`) installs PostgreSQL extensions required by
all domains. This was previously in 01-iam's first migration — it is moved here because
extensions are shared infrastructure, not IAM-specific.

```sql
-- Migration: m000000_000000_bootstrap.rs
-- Installs PostgreSQL extensions required by the platform.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";     -- UUID generation
CREATE EXTENSION IF NOT EXISTS "pgcrypto";       -- Cryptographic functions
CREATE EXTENSION IF NOT EXISTS "postgis";        -- Spatial queries [ARCH §5.4]
CREATE EXTENSION IF NOT EXISTS "pg_trgm";        -- Trigram fuzzy matching [ARCH §2.6]
```

**Idempotent**: `IF NOT EXISTS` ensures this migration can be re-run safely.

### §9.5 Application vs. Migration Roles

Two PostgreSQL roles are used:

| Role | Purpose | RLS behavior |
|------|---------|--------------|
| Migration role | Runs `sea-orm-migration` (CREATE TABLE, ALTER, etc.) | Bypasses RLS (superuser or table owner) |
| Application role | Used by the Rust API at runtime | Subject to RLS policies |

The application role MUST NOT be a superuser. This ensures RLS cannot be accidentally
bypassed by application queries. The migration role creates tables and RLS policies; the
application role respects them.

In local development (docker-compose), a single `homegrown` user with superuser is acceptable.
In production (RDS), the application user is a non-superuser role granted `SELECT, INSERT,
UPDATE, DELETE` on all tables. `[ARCH §2.5, §5.2]`

---

## §10 Redis Infrastructure (`src/shared/redis.rs`)

### §10.1 Pool Creation

```rust
use redis::aio::ConnectionManager;

pub type RedisPool = ConnectionManager;

pub async fn create_redis_pool(config: &AppConfig) -> Result<RedisPool, redis::RedisError> {
    let client = redis::Client::open(config.redis_url.as_str())?;
    let manager = ConnectionManager::new(client).await?;

    // Validate connectivity with PING
    let mut conn = manager.clone();
    redis::cmd("PING").query_async::<String>(&mut conn).await?;

    Ok(manager)
}
```

### §10.2 Phase 1 Helpers

Generic get/set with TTL and delete. These are the only Redis operations needed in Phase 1
(rate limiting + methodology config caching).

```rust
use redis::AsyncCommands;
use serde::{de::DeserializeOwned, Serialize};

/// Get a JSON-serialized value from Redis.
pub async fn get<T: DeserializeOwned>(
    pool: &mut RedisPool,
    key: &str,
) -> Result<Option<T>, AppError> {
    let value: Option<String> = pool.get(key).await
        .map_err(|e| AppError::Internal(e.into()))?;
    match value {
        Some(s) => Ok(Some(serde_json::from_str(&s)
            .map_err(|e| AppError::Internal(e.into()))?)),
        None => Ok(None),
    }
}

/// Set a JSON-serialized value in Redis with a TTL.
pub async fn set<T: Serialize>(
    pool: &mut RedisPool,
    key: &str,
    value: &T,
    ttl_seconds: u64,
) -> Result<(), AppError> {
    let serialized = serde_json::to_string(value)
        .map_err(|e| AppError::Internal(e.into()))?;
    pool.set_ex(key, serialized, ttl_seconds).await
        .map_err(|e| AppError::Internal(e.into()))
}

/// Delete a key from Redis.
pub async fn delete(pool: &mut RedisPool, key: &str) -> Result<(), AppError> {
    pool.del(key).await
        .map_err(|e| AppError::Internal(e.into()))
}
```

### §10.3 Rate Limit Counter

Atomic increment with expiry for rate limiting (§13.2).

```rust
/// Increment a counter and set expiry if this is the first increment in the window.
/// Returns the new counter value.
pub async fn increment_with_expiry(
    pool: &mut RedisPool,
    key: &str,
    window_seconds: u64,
) -> Result<i64, AppError> {
    let count: i64 = redis::cmd("INCR")
        .arg(key)
        .query_async(pool)
        .await
        .map_err(|e| AppError::Internal(e.into()))?;

    if count == 1 {
        // First request in window — set expiry
        pool.expire(key, window_seconds as i64).await
            .map_err(|e| AppError::Internal(e.into()))?;
    }

    Ok(count)
}
```

### §10.4 Deferred Redis Features

The following Redis usage patterns are owned by their consuming domains and are NOT specified
here:

| Feature | Owning Domain | Redis Data Structure |
|---------|---------------|---------------------|
| Feed fan-out | `social::` | Sorted sets |
| Pub/sub (WebSocket) | `social::`, `notify::` | Pub/sub channels |
| Background job queues | `safety::`, `notify::`, `search::` | Lists (sidekiq-rs) |
| Session caching | `iam::` | Strings with TTL |

---

## §11 Event Bus (`src/shared/events.rs`)

### §11.1 Core Traits

```rust
use std::any::TypeId;
use std::collections::HashMap;
use std::sync::Arc;
use async_trait::async_trait;

/// Marker trait for domain events.
/// Events MUST be defined in the emitting domain's `events.rs`. [CODING §8.4]
pub trait DomainEvent: Send + Sync + 'static {}

/// Handler for a specific domain event type.
/// Handlers MUST be defined in the consuming domain. [CODING §8.4]
#[async_trait]
pub trait DomainEventHandler<E: DomainEvent>: Send + Sync {
    async fn handle(&self, event: &E) -> Result<(), AppError>;
}
```

### §11.2 EventBus Implementation

The event bus uses `TypeId`-based dispatch. Each event type maps to a list of type-erased
handlers.

```rust
/// Type-erased handler wrapper.
#[async_trait]
trait ErasedHandler: Send + Sync {
    async fn handle_erased(&self, event: &dyn std::any::Any) -> Result<(), AppError>;
}

/// Adapter: wraps a typed handler as an erased handler.
struct HandlerAdapter<E: DomainEvent, H: DomainEventHandler<E>> {
    handler: Arc<H>,
    _phantom: std::marker::PhantomData<E>,
}

#[async_trait]
impl<E: DomainEvent + 'static, H: DomainEventHandler<E>> ErasedHandler
    for HandlerAdapter<E, H>
{
    async fn handle_erased(&self, event: &dyn std::any::Any) -> Result<(), AppError> {
        let event = event.downcast_ref::<E>()
            .expect("event type mismatch — this is a bug in EventBus");
        self.handler.handle(event).await
    }
}

pub struct EventBus {
    handlers: HashMap<TypeId, Vec<Box<dyn ErasedHandler>>>,
}

impl EventBus {
    pub fn new() -> Self {
        Self {
            handlers: HashMap::new(),
        }
    }

    /// Register a handler for a specific event type.
    /// MUST be called at startup only (in main.rs). [CODING §8.4]
    pub fn subscribe<E, H>(&mut self, handler: Arc<H>)
    where
        E: DomainEvent + 'static,
        H: DomainEventHandler<E> + 'static,
    {
        let adapter = Box::new(HandlerAdapter {
            handler,
            _phantom: std::marker::PhantomData::<E>,
        });
        self.handlers
            .entry(TypeId::of::<E>())
            .or_default()
            .push(adapter);
    }

    /// Publish a domain event. Dispatches to all registered handlers.
    ///
    /// Phase 1: Synchronous dispatch. Handlers run inline within the
    /// caller's context. Handler errors are logged but do NOT fail the
    /// publish — the publishing domain's operation has already succeeded.
    pub async fn publish<E: DomainEvent + 'static>(&self, event: E) -> Result<(), AppError> {
        if let Some(handlers) = self.handlers.get(&TypeId::of::<E>()) {
            for handler in handlers {
                if let Err(err) = handler.handle_erased(&event).await {
                    tracing::error!(
                        event_type = std::any::type_name::<E>(),
                        error = %err,
                        "event handler failed"
                    );
                    // Handler errors are logged, not propagated.
                    // The domain operation that triggered the event has
                    // already completed successfully.
                }
            }
        }
        Ok(())
    }
}
```

### §11.3 Phase 1 Behavior

- **Synchronous in-process dispatch**: `publish()` calls handlers sequentially within the
  same async task. No message broker, no serialization overhead.
- **Error isolation**: Handler failures are logged but do not fail the publish call. The
  domain operation that triggered the event has already committed successfully — handler
  failure is a downstream concern, not a transactional rollback trigger.
- **No retry**: Failed handlers are logged and skipped. Phase 2 adds retry via background
  jobs for handlers that do heavy work.

### §11.4 Subscription Registration

All subscriptions MUST be registered in `main.rs` during startup (step 6 of §4.2).
`[CODING §8.4]`

```rust
// In main.rs, step 6:
let mut event_bus = EventBus::new();

// IAM events → Social handler
event_bus.subscribe::<FamilyCreated, _>(
    Arc::new(social::event_handlers::OnFamilyCreated::new(social_service.clone()))
);

// IAM events → Onboarding handler
event_bus.subscribe::<FamilyCreated, _>(
    Arc::new(onboard::event_handlers::OnFamilyCreated::new(onboard_service.clone()))
);

// ... more subscriptions added as domains are built
let event_bus = Arc::new(event_bus);
```

---

## §12 Pagination (`src/shared/pagination.rs`)

### §12.1 PaginationParams

Query parameters accepted by list endpoints. Cursor-based pagination is the primary
mechanism. `[ARCH §1.4]`

```rust
use serde::Deserialize;
use utoipa::IntoParams;

#[derive(Debug, Deserialize, IntoParams)]
pub struct PaginationParams {
    /// Opaque cursor from previous response's `next_cursor`.
    /// Omit for the first page.
    pub cursor: Option<String>,

    /// Items per page. Default: 20. Max: 100.
    pub limit: Option<u32>,
}

impl PaginationParams {
    pub fn effective_limit(&self) -> u32 {
        self.limit.unwrap_or(20).min(100)
    }
}
```

### §12.2 PaginatedResponse

```rust
use serde::Serialize;
use utoipa::ToSchema;

#[derive(Debug, Serialize, ToSchema)]
pub struct PaginatedResponse<T: Serialize> {
    pub data: Vec<T>,
    /// Opaque cursor for the next page. `null` if no more results.
    pub next_cursor: Option<String>,
    /// Whether more results exist beyond this page.
    pub has_more: bool,
}
```

### §12.3 Cursor Encoding

Cursors encode `(id, created_at)` as base64url to prevent clients from constructing or
manipulating cursors. The encoding is an implementation detail — clients treat cursors as
opaque strings.

```rust
use base64::{engine::general_purpose::URL_SAFE_NO_PAD, Engine};
use chrono::{DateTime, Utc};
use uuid::Uuid;

pub fn encode_cursor(id: Uuid, created_at: DateTime<Utc>) -> String {
    let raw = format!("{}:{}", id, created_at.timestamp_millis());
    URL_SAFE_NO_PAD.encode(raw.as_bytes())
}

pub fn decode_cursor(cursor: &str) -> Result<(Uuid, DateTime<Utc>), AppError> {
    let bytes = URL_SAFE_NO_PAD
        .decode(cursor)
        .map_err(|_| AppError::BadRequest("invalid cursor".into()))?;
    let raw = String::from_utf8(bytes)
        .map_err(|_| AppError::BadRequest("invalid cursor".into()))?;

    let parts: Vec<&str> = raw.splitn(2, ':').collect();
    if parts.len() != 2 {
        return Err(AppError::BadRequest("invalid cursor".into()));
    }

    let id = Uuid::parse_str(parts[0])
        .map_err(|_| AppError::BadRequest("invalid cursor".into()))?;
    let ts = parts[1]
        .parse::<i64>()
        .map_err(|_| AppError::BadRequest("invalid cursor".into()))?;
    let created_at = DateTime::from_timestamp_millis(ts)
        .ok_or_else(|| AppError::BadRequest("invalid cursor".into()))?;

    Ok((id, created_at))
}
```

---

## §13 Middleware Stack (`src/middleware/`)

### §13.1 Auth Middleware (`auth.rs`)

The auth middleware validates Kratos session cookies and populates `AuthContext` in request
extensions.

**Signature**:

```rust
pub async fn auth_middleware(
    State(state): State<AppState>,
    mut req: Request<axum::body::Body>,
    next: Next,
) -> Result<Response, AppError>
```

**Behavior**:

1. Extract session cookie from the `Cookie` header
2. Call Kratos public API to validate the session (via `KratosAdapter` port from IAM)
3. Look up the parent in the local database by `kratos_identity_id` (unscoped — FamilyScope
   does not exist yet)
4. Look up the parent's family
5. Build `AuthContext` from parent + family data
6. Insert `AuthContext` into request extensions
7. Call `next.run(req)`

**Error responses**:
- No cookie present → 401 Unauthorized
- Kratos session invalid/expired → 401 Unauthorized
- Parent not found in local DB (orphaned Kratos identity) → 401 Unauthorized

**Implementation ownership**: The auth middleware *function* lives in `src/middleware/auth.rs`
(core infrastructure). It calls `KratosAdapter::validate_session()` which is a port defined
in `src/iam/ports.rs` and implemented in `src/iam/adapters/kratos.rs`. The middleware depends
on IAM's adapter at runtime but does not import IAM's service layer.

**Dependency note**: Because auth middleware depends on `KratosAdapter`, the IAM domain's
adapter must be available before the middleware can function. The `AppState` provides the
adapter as `Arc<dyn KratosAdapter>`.

### §13.2 Rate Limiting (`rate_limit.rs`)

Token-bucket rate limiting via Redis. `[ARCH §3.3, S§2.3]`

**Tiers**:

| Tier | Scope | Limit | Window |
|------|-------|-------|--------|
| Default (unauthenticated) | Per IP | 100 requests | 60 seconds |
| Default (authenticated) | Per user ID | 100 requests | 60 seconds |
| Auth endpoints (`/hooks/*`) | Per IP | 10 requests | 60 seconds |
| Sensitive endpoints | Per user ID | 20 requests | 60 seconds |

**Behavior**:
- Uses `increment_with_expiry` from `src/shared/redis.rs` (§10.3)
- On exceeded limit: return 429 Too Many Requests with `Retry-After` header (seconds until
  window expires)
- Rate limit key format: `rl:{scope}:{identifier}:{window_start}`

**Note**: Rate limit tiers can be overridden per-route using Axum's layer system. Specific
domains may apply more restrictive limits to their endpoints.

### §13.3 Role Extractors (`extractors.rs`)

Axum extractors that wrap `AuthContext` with additional permission checks. These are consumed
by many domains — they are shared infrastructure, not IAM-specific.

#### `RequirePremium`

```rust
/// Extracts AuthContext and verifies the user has a premium subscription.
/// Returns 402 Payment Required if the user is on the free tier. [S§3.2]
pub struct RequirePremium(pub AuthContext);

#[axum::async_trait]
impl<S> FromRequestParts<S> for RequirePremium
where
    S: Send + Sync,
{
    type Rejection = AppError;

    async fn from_request_parts(
        parts: &mut Parts,
        _state: &S,
    ) -> Result<Self, Self::Rejection> {
        let auth = parts
            .extensions
            .get::<AuthContext>()
            .cloned()
            .ok_or(AppError::Unauthorized)?;

        if auth.subscription_tier != SubscriptionTier::Premium {
            return Err(AppError::PremiumRequired);
        }

        Ok(RequirePremium(auth))
    }
}
```

**Consuming domains**: `learn::`, `comply::`, `ai::` `[IAM §13.1]`

#### `RequireCreator`

```rust
/// Extracts AuthContext and verifies the user has a creator account.
/// Returns 403 Forbidden if no creator account exists. [S§3.1.4]
///
/// The creator_id is looked up from `mkt_creators` via the parent_id.
/// This lookup crosses the IAM → Marketplace boundary via AppState,
/// which holds a reference to the marketplace service or a lightweight
/// creator-check query.
pub struct RequireCreator {
    pub auth: AuthContext,
    pub creator_id: Uuid,
}
```

**Implementation note**: `RequireCreator` needs to check `mkt_creators` table. This crosses
domain boundaries. Two acceptable approaches:

1. **Preferred**: Add `creator_id: Option<Uuid>` to `AuthContext` (populated during auth
   middleware from a JOIN). No cross-domain call needed.
2. **Alternative**: Query `mkt::` service from the extractor via `AppState`.

The chosen approach is documented in the marketplace domain spec (when written).

**Consuming domains**: `mkt::` `[IAM §13.1]`

#### `RequireCoppaConsent`

```rust
/// Extracts AuthContext and verifies the family has active COPPA consent.
/// Returns 403 Forbidden with code `coppa_consent_required` if consent
/// is not `Consented` or `ReVerified`. [ARCH §6.3]
pub struct RequireCoppaConsent(pub AuthContext);
```

**Implementation note**: Requires looking up `iam_families.coppa_consent_status`. Two
approaches:

1. **Preferred**: Add `coppa_consent_status` to `AuthContext` (populated during auth
   middleware). No additional DB query per request.
2. **Alternative**: Query `iam::` service from the extractor.

Approach 1 is recommended — the auth middleware already queries `iam_families` for the
subscription tier, so adding the consent status is free.

**Consuming domains**: `learn::`, `social::` (student-facing features) `[IAM §13.1]`

#### `RequirePrimaryParent` (Phase 2)

```rust
/// Extracts AuthContext and verifies the user is the family's primary parent.
/// Returns 403 Forbidden if not. [S§3.4]
pub struct RequirePrimaryParent(pub AuthContext);

#[axum::async_trait]
impl<S> FromRequestParts<S> for RequirePrimaryParent
where
    S: Send + Sync,
{
    type Rejection = AppError;

    async fn from_request_parts(
        parts: &mut Parts,
        _state: &S,
    ) -> Result<Self, Self::Rejection> {
        let auth = parts
            .extensions
            .get::<AuthContext>()
            .cloned()
            .ok_or(AppError::Unauthorized)?;

        if !auth.is_primary_parent {
            return Err(AppError::Forbidden);
        }

        Ok(RequirePrimaryParent(auth))
    }
}
```

**Phase 2**: Used by co-parent management, family deletion, and COPPA withdrawal endpoints.

---

## §14 Logging & Tracing

### §14.1 Initialization

```rust
use tracing_subscriber::{fmt, EnvFilter, layer::SubscriberExt, util::SubscriberInitExt};

pub fn init_tracing(config: &AppConfig) {
    let filter = EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| EnvFilter::new(&config.log_level));

    let format_layer = if config.environment == Environment::Production {
        // JSON format for structured log aggregation (CloudWatch)
        fmt::layer().json().flatten_event(true).boxed()
    } else {
        // Pretty format for local development
        fmt::layer().pretty().boxed()
    };

    let registry = tracing_subscriber::registry()
        .with(filter)
        .with(format_layer);

    // Optional Sentry integration [ARCH §2.14]
    if let Some(dsn) = &config.sentry_dsn {
        let _guard = sentry::init((
            dsn.as_str(),
            sentry::ClientOptions {
                release: Some(env!("CARGO_PKG_VERSION").into()),
                traces_sample_rate: 0.1,
                ..Default::default()
            },
        ));
        registry
            .with(sentry::integrations::tracing::layer())
            .init();
    } else {
        registry.init();
    }
}
```

### §14.2 Log Level Filters

Suppress noisy dependencies by default:

```
homegrown_academy=debug,tower_http=debug,sea_orm=warn,hyper=warn,rustls=warn
```

Production default: `homegrown_academy=info,tower_http=info`

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

## §15 OpenAPI Generation Binary

### §15.1 Binary Location

`src/bin/openapi_gen.rs` — a standalone binary that generates the OpenAPI spec.
`[CODING §6.1]`

### §15.2 Behavior

```rust
use utoipa::OpenApi;

#[derive(OpenApi)]
#[openapi(
    info(
        title = "Homegrown Academy API",
        version = "0.1.0",
        description = "API for the Homegrown Academy homeschooling platform",
    ),
    paths(
        // Domain paths are added here as domains are implemented
        // crate::domains::iam::handlers::get_current_user,
        // crate::domains::iam::handlers::get_family_profile,
        // ...
    ),
    components(schemas(
        // Domain schemas are added here as domains are implemented
        // crate::domains::iam::models::CurrentUserResponse,
        // crate::domains::iam::models::FamilyProfileResponse,
        // ...
    )),
    tags(
        (name = "health", description = "Health check"),
        // Domain tags added as domains are implemented
    )
)]
struct ApiDoc;

fn main() {
    let spec = ApiDoc::openapi()
        .to_yaml()
        .expect("failed to serialize OpenAPI spec");

    std::fs::write("openapi/spec.yaml", spec)
        .expect("failed to write openapi/spec.yaml");

    println!("OpenAPI spec written to openapi/spec.yaml");
}
```

**Usage**: `cargo run --bin openapi-gen`

**Key rules** `[CODING §6.1]`:
- MUST run after any change to Rust API types in `models.rs`
- MUST commit `openapi/spec.yaml` alongside the Rust changes
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

# 3. Start the Rust API
cargo run

# 4. Start the frontend dev server (in a separate terminal)
cd frontend && npm run dev

# 5. Access the app at http://localhost:5173
```

---

## §17 Dev Commands (`justfile`)

A `justfile` at project root provides all common development commands. `[CLAUDE.md —
"Development Commands" section]`

```justfile
# Homegrown Academy — Development Commands

# Default recipe: run all quality gates
default: check

# ─── Development ─────────────────────────────────────────────────────

# Start the Rust API server
dev:
    cargo run

# Start all infrastructure services
docker-up:
    docker compose up -d

# Stop all infrastructure services
docker-down:
    docker compose down

# ─── Quality Gates ───────────────────────────────────────────────────

# Run all quality gates (must pass before every commit)
check: lint test type-check

# Run Rust linter (zero warnings required)
lint:
    cargo clippy -- -D warnings

# Run Rust tests
test:
    cargo test

# Run TypeScript type checker (zero errors required)
type-check:
    cd frontend && npm run type-check

# ─── Database ────────────────────────────────────────────────────────

# Run pending database migrations
migrate:
    cargo run -- migrate

# Regenerate SeaORM entities for a domain
entities DOMAIN:
    sea-orm-cli generate entity \
        --with-serde both \
        --output-dir src/domains/{{DOMAIN}}/entities/ \
        --tables "$(echo {{DOMAIN}} | head -c 3)_%"

# Reset the database (drop + recreate + migrate)
db-reset:
    docker compose exec postgres psql -U homegrown -c "DROP DATABASE IF EXISTS homegrown;"
    docker compose exec postgres psql -U homegrown -c "CREATE DATABASE homegrown;"
    just migrate

# ─── Code Generation ────────────────────────────────────────────────

# Generate OpenAPI spec from Rust types
openapi:
    cargo run --bin openapi-gen

# Generate TypeScript types from OpenAPI spec
generate-types:
    cd frontend && npm run generate-types

# Full generation pipeline: entities → openapi → TS types
full-generate DOMAIN:
    just entities {{DOMAIN}}
    just openapi
    just generate-types
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

---

## §19 Implementation Checklist

Phase 1 items organized by dependency order. Each item maps to a section in this document.

### Phase 1 — Core Infrastructure (must complete before any domain)

#### Project Setup
- [ ] Initialize `Cargo.toml` with all dependencies (§2.1)
- [ ] Create directory skeleton (§2.2)
- [ ] Create `frontend/package.json` and install dependencies (§2.3)
- [ ] Create `frontend/tsconfig.json` (§2.4)
- [ ] Create `frontend/vite.config.ts` (§2.5)
- [ ] Create `.env.example` (§3.3)
- [ ] Create `.gitignore` (include `.env`, `target/`, `node_modules/`, etc.)

#### Configuration
- [ ] Implement `AppConfig` struct and `from_env()` (§3.1, §3.2)
- [ ] Implement `AppConfigError` with `required_env` / `optional_env` helpers
- [ ] Implement `Environment` enum

#### Error Framework
- [ ] Implement `AppError` enum with all variants (§6.1)
- [ ] Implement `IntoResponse for AppError` (§6.3)
- [ ] Implement `From<validator::ValidationErrors> for AppError` (§6.5)
- [ ] Implement `From<sea_orm::DbErr> for AppError`

#### Shared Types
- [ ] Implement newtype wrappers: `FamilyId`, `ParentId`, `StudentId`, `CreatorId` (§7.1)
- [ ] Implement `AuthContext` struct (§7.2)
- [ ] Implement `SubscriptionTier` enum (§7.3)
- [ ] Implement `FromRequestParts for AuthContext` (§7.2)

#### Family Scope
- [ ] Implement `FamilyScope` struct with private field (§8.1)
- [ ] Implement `From<&AuthContext> for FamilyScope` (§8.2)
- [ ] Implement `FromRequestParts for FamilyScope` (§8.3)

#### Database
- [ ] Implement `create_pool()` (§9.1)
- [ ] Implement `scoped_transaction()` (§9.2)
- [ ] Implement `unscoped_transaction()` (§9.3)
- [ ] Create bootstrap migration with PostgreSQL extensions (§9.4)
- [ ] Set up migration runner in `main.rs`

#### Redis
- [ ] Implement `create_redis_pool()` (§10.1)
- [ ] Implement generic `get<T>`, `set<T>`, `delete` helpers (§10.2)
- [ ] Implement `increment_with_expiry` (§10.3)

#### Event Bus
- [ ] Implement `DomainEvent` and `DomainEventHandler` traits (§11.1)
- [ ] Implement `EventBus` with TypeId dispatch (§11.2)
- [ ] Verify handler errors are logged but don't fail publish

#### Pagination
- [ ] Implement `PaginationParams` (§12.1)
- [ ] Implement `PaginatedResponse<T>` (§12.2)
- [ ] Implement cursor encode/decode (§12.3)

#### Middleware
- [ ] Implement auth middleware signature (§13.1) — stub until IAM provides KratosAdapter
- [ ] Implement rate limiting middleware (§13.2)
- [ ] Implement `RequirePremium` extractor (§13.3)
- [ ] Implement `RequireCoppaConsent` extractor (§13.3)
- [ ] Stub `RequireCreator` extractor (depends on marketplace domain)

#### Application Wiring
- [ ] Implement `AppState` struct (§5.1)
- [ ] Implement `create_app()` with router and middleware stack (§5.2)
- [ ] Implement health endpoint (§5.4)

#### Main Entrypoint
- [ ] Implement `main()` with startup sequence (§4.2)
- [ ] Implement graceful shutdown (§4.3)

#### Logging
- [ ] Implement `init_tracing()` with JSON/pretty format switching (§14.1)
- [ ] Configure log level filters (§14.2)

#### OpenAPI
- [ ] Create `src/bin/openapi_gen.rs` skeleton (§15)
- [ ] Create empty `openapi/` directory

#### Local Dev
- [ ] Create `docker-compose.yml` (§16.1)
- [ ] Create Kratos dev config (§16.2)
- [ ] Create identity schema (§16.3)

#### Dev Commands
- [ ] Create `justfile` with all commands (§17)
- [ ] Update `CLAUDE.md` "Development Commands" section to reference justfile

#### Frontend Shell
- [ ] Create `index.html` (§18.1)
- [ ] Create `main.tsx` (§18.2)
- [ ] Create `App.tsx` (§18.3)
- [ ] Create `query-client.ts` (§18.4)
- [ ] Create `api/client.ts` (§18.5)
- [ ] Create empty `api/generated/`, `components/ui/`, `features/` directories

#### Quality Gates
- [ ] Verify `cargo build` succeeds
- [ ] Verify `cargo clippy -- -D warnings` passes
- [ ] Verify `cargo test` passes (health endpoint test)
- [ ] Verify `cd frontend && npm run type-check` passes
- [ ] Verify `cd frontend && npm run dev` starts without errors
- [ ] Verify `docker compose up -d` starts all services with healthy status

### Phase 2 — Deferred Items

These items are NOT specified in this document. They will be added when their consuming
domain requires them.

| Item | Owned By | When |
|------|----------|------|
| `RequirePrimaryParent` extractor | This spec (§13.3) | When IAM Phase 2 is implemented |
| Background job runner (`sidekiq-rs`) | Consuming domain | When first domain needs async jobs |
| WebSocket infrastructure | `social::` | When messaging is implemented |
| Meilisearch integration | `search::` | When PG FTS is insufficient (§2.6) |
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
> middleware (`src/middleware/auth.rs`, defined in 00-core §13.1) calls IAM's
> `KratosAdapter::validate_session()` and queries IAM repositories to build the
> `AuthContext`. The flow is:
>
> 1. Auth middleware extracts session cookie
> 2. Calls `KratosAdapter::validate_session()` (IAM §7)
> 3. Calls `ParentRepository::find_by_kratos_id()` (IAM §6)
> 4. Calls `FamilyRepository::find_by_id()` (IAM §6)
> 5. Constructs `AuthContext` from parent + family data
>
> #### §11.2 COPPA Consent Check
>
> The `RequireCoppaConsent` extractor (00-core §13.3) checks
> `AuthContext.coppa_consent_status` which IAM populates from `iam_families`.
>
> [Keep existing §11.3-§11.6 behavioral docs but remove code samples that duplicate 00-core]

### §20.4 Update IAM §12 Error Types — Back-reference

**Current** (IAM §12): Defines `IamError` enum and its mapping to HTTP statuses.

**Change**: Add a reference to the base `AppError`:

> `IamError` maps to `AppError` (defined in 00-core §6) via `From<IamError> for AppError`.
> See 00-core §6.4 for the conversion pattern.

This is an additive note — do NOT remove the `IamError` definition or its HTTP mapping table.
`IamError` is IAM-specific; only the `AppError` base type moves to core.

### §20.5 Update IAM §16 Implementation Checklist — Remove Shared Items

**Current** (IAM §16, "Shared Infrastructure" section, lines 1547-1553):

```
#### Shared Infrastructure
- [ ] Implement `FamilyScope` type in `src/shared/family_scope.rs`
- [ ] Implement `AuthContext` struct and auth middleware in `src/middleware/auth.rs`
- [ ] Implement `FamilyScope` extractor (from AuthContext)
- [ ] Implement `RequirePremium` extractor
- [ ] Implement `RequireCreator` extractor
- [ ] Implement `RequireCoppaConsent` middleware
- [ ] Implement `AppError` enum with IAM variants in `src/shared/error.rs`
```

**Change**: Replace with a dependency reference:

```
#### Shared Infrastructure (prerequisite — see 00-core)
- [ ] Verify 00-core §19 checklist is complete (AppError, AuthContext, FamilyScope,
      extractors, middleware, DB pool, Redis pool, EventBus)
- [ ] Implement `From<IamError> for AppError` conversion (00-core §6.4 pattern)
```

### §20.6 Add coppa_consent_status to AuthContext Note

Add a note in IAM §11.1 or §9 that `coppa_consent_status` SHOULD be included in
`AuthContext` to enable the `RequireCoppaConsent` extractor to work without an extra DB query.
This means the auth middleware's family lookup (already happening) also reads
`coppa_consent_status` and includes it in the `AuthContext`.

This change requires adding `coppa_consent_status: CoppaConsentStatus` to the `AuthContext`
struct in 00-core §7.2. However, since `CoppaConsentStatus` is an IAM-owned enum, this
creates a dependency from shared types to IAM. Two resolution approaches:

1. **Preferred**: Store the consent status as a `String` in `AuthContext` and let the
   `RequireCoppaConsent` extractor match on known values. No cross-domain type dependency.
2. **Alternative**: Move `CoppaConsentStatus` to `src/shared/types.rs` since it's needed
   by shared infrastructure.

Decision: Use approach 1 (String). Add to `AuthContext`:

```rust
pub coppa_consent_status: String,  // "consented", "re_verified", etc.
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
| §14 Logging | — | §5.2 (PII rules) | Yes — PII rules existed, tracing setup didn't |
| §15 OpenAPI | — | §6.1 (generation rules) | Yes — rules existed, binary spec didn't |
| §16 Local Dev | — | — | Yes — completely unspecified |
| §17 Dev Commands | — | §7.2 (quality gates) | Yes — gates existed, commands didn't |
| §18 Frontend Shell | §2.3 (React stack) | §3 (TS rules) | Yes — stack chosen, files unspecified |

## Appendix B: Naming Conflict Resolution

| Item | ARCH Uses | CODING Uses | This Spec Uses | Rationale |
|------|-----------|-------------|----------------|-----------|
| Application error type | `ApiError` | `AppError` | `AppError` | CODING_STANDARDS is authoritative for naming (§1) |

All code and documentation produced after this document MUST use `AppError`. ARCHITECTURE.md
code samples using `ApiError` should be read as `AppError` — they are not normative.
