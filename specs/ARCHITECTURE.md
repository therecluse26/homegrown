# Homegrown Academy — Technical Architecture

## 1. Architecture Principles

Six principles govern every technical decision. They are ordered by priority — when principles conflict, higher-ranked principles win.

### 1.1 AI-First Development

The codebase is primarily generated and maintained by AI (Claude). This inverts traditional technology selection criteria:

- **Learning curve is irrelevant** — AI generates idiomatic Rust, TypeScript, SQL, and configuration equally well. The "difficulty" of a language is not a cost.
- **Compiler strictness is a free safety net** — Rust's borrow checker, lifetime system, and type system catch entire categories of bugs at compile time. For AI-generated code, this is pure upside: the compiler reviews every line before it runs.
- **Explicitness over magic** — AI generates better code when patterns are explicit and consistent. Convention-over-configuration frameworks (Rails, Laravel) rely on implicit knowledge that AI may misapply. Explicit configuration and typed contracts produce more reliable generated code.
- **Strong type systems reduce review burden** — When the compiler enforces correctness, the human developer (solo) can focus review on business logic and architecture rather than null checks and type mismatches.

### 1.2 Monolith-First

A single deployable unit until proven otherwise. `[S§17.4]`

- **14 spec domains → Rust modules** within one binary, not 14 services.
- **One PostgreSQL database** with schema prefixes per domain.
- **One deployment target** — a single Docker container on a single server.
- **Microservice extraction is a scaling decision**, not an architecture decision. Extract only when a domain has demonstrably different scaling needs (e.g., media processing) that cannot be solved by adding capacity.

**Revision trigger**: Extract a domain when its resource consumption exceeds 40% of total system resources, or when its deployment cadence fundamentally conflicts with the rest of the system.

### 1.3 Type-Safety-Everywhere

Types flow from database schema to API response to React component. `[S§17.1]`

- **Rust structs** define API request/response shapes.
- **SeaORM entities** are generated from database migrations — the schema is the source of truth.
- **OpenAPI 3.1** spec is generated from Rust types.
- **TypeScript client types** are generated from OpenAPI spec.
- Zero `any` in TypeScript. Zero `.unwrap()` in production Rust (use `?` or explicit error handling).

### 1.4 Progressive Complexity

Start simple. Add complexity only when measured load demands it. `[S§17.4]`

- PostgreSQL full-text search before Meilisearch.
- Single server before load balancer.
- In-process background tasks before distributed job queues.
- Reverse-chronological feed before algorithmic ranking.

Every "simple first" choice includes a documented **revision trigger** — the specific metric or threshold that justifies upgrading.

### 1.5 Privacy-by-Architecture

Privacy is enforced structurally, not by policy. `[S§17.2]`

- **Family-scoped queries** — a Rust trait enforces that every database query includes a `family_id` filter. Cross-family data access is structurally impossible without explicit opt-in (social sharing).
- **Default-deny visibility** — social content defaults to friends-only at the database level. There is no "public" visibility enum variant for user-generated content. `[S§7.2.2]`
- **Coarse location only** — the database stores city/region identifiers, never coordinates. `[S§7.8]`
- **COPPA by construction** — student profiles have no credentials, no independent sessions, and no direct messaging capability. The data model makes COPPA violations structurally difficult. `[S§17.2]`

### 1.6 Methodology-as-Configuration

Methodologies are data, not code. `[S§4.1]`

- Methodology definitions live in database rows with JSONB configuration.
- The tool registry, onboarding flows, and UI terminology are all driven by methodology config records.
- Adding a new methodology requires inserting rows, not writing code or deploying.
- No `if methodology == "charlotte_mason"` branches in application code — all methodology-dependent behavior is resolved through configuration lookup.

---

## 2. Technology Selections

Every selection references the spec requirement it satisfies, names rejected alternatives, and defines a revision trigger.

### 2.1 Backend: Rust + Axum + Tokio

**Satisfies**: `[S§17.3]` performance targets, `[S§17.1]` security, `[S§17.4]` scalability

| Attribute | Value |
|-----------|-------|
| **Language** | Rust (latest stable) |
| **Web framework** | Axum 0.8+ |
| **Async runtime** | Tokio (multi-threaded) |
| **Expected throughput** | 32,000-52,000 requests/sec with DB (single server) |
| **Memory footprint** | 8-20MB per instance |

**Rationale**: AI generates Rust code; the borrow checker catches memory safety, data races, and null pointer errors at compile time — for free. A solo developer cannot manually review every line. Rust's compiler is the second reviewer. Performance means Phase 1-2 runs comfortably on a single $60/mo server, saving thousands in infrastructure costs annually.

**Rejected alternatives**:
- **TypeScript (Node.js)**: 3-5x lower throughput, no compile-time safety beyond types, `null`/`undefined` hazards, need for runtime validation libraries. Reasonable choice, but Rust's compile-time guarantees are more valuable for AI-generated code.
- **Go**: Strong performance and simplicity, but the user finds the language unpleasant to work in daily. Developer satisfaction matters for a multi-year solo project.
- **Ruby on Rails**: No type safety, 10-20x slower than Rust, magic conventions conflict with AI-first development principle.

**Revision trigger**: Never — Rust is the "never rewrite" choice. Individual crates may be replaced, but the language stays.

### 2.2 ORM: SeaORM

**Satisfies**: `[S§16]` data architecture, `[S§17.3]` performance

| Attribute | Value |
|-----------|-------|
| **Crate** | `sea-orm` (latest stable) |
| **Migration tool** | `sea-orm-migration` |
| **Query style** | Compile-time checked, async-native |

**Rationale**: Most mature async Rust ORM. Supports PostgreSQL JSONB and array types natively (needed for methodology config `[S§4.1]`). Migration system handles schema evolution. Compile-time query checking catches SQL errors before runtime.

**Rejected alternatives**:
- **Diesel**: Synchronous, poor async story. Mature but doesn't fit Tokio ecosystem.
- **SQLx**: Not an ORM — raw SQL with compile-time checking. Good for complex queries but lacks migration system and entity relationships that a 14-domain app needs.

**Revision trigger**: If SeaORM's async performance becomes a bottleneck (unlikely), supplement specific hot-path queries with raw SQLx.

### 2.3 Application Frontend: React (Vite SPA)

**Satisfies**: `[S§17.8]` browser support, `[S§17.6]` accessibility, `[S§7]` social layer interactivity

| Attribute | Value |
|-----------|-------|
| **Library** | React 19+ |
| **Build tool** | Vite |
| **Routing** | React Router v7 |
| **Server state** | TanStack Query (React Query) |
| **Styling** | Tailwind CSS v4 |
| **Type checking** | TypeScript (strict mode) |

**Rationale**: The platform has 14 domains with extensive interactivity — messaging, social feeds, learning tool interactions, marketplace browse/purchase, quiz flows. React's component ecosystem (rich text editors, file uploaders, drag-and-drop, data visualization) is unmatched. SPA architecture keeps real-time features (messaging, notifications) simple — no SSR hydration complexity.

**Rejected alternatives**:
- **SvelteKit**: Smaller ecosystem. Fewer production-grade component libraries for the breadth needed (messaging, rich text, charts, file upload).
- **Next.js (SSR)**: SSR adds complexity without benefit — the app is behind auth, so SEO is irrelevant. Public content is handled by Astro (§2.4).
- **HTMX + server templates**: Inadequate for the interactivity level required by messaging, real-time feeds, and learning tools.

**Revision trigger**: None anticipated. React is the stable choice for complex SPAs.

### 2.4 Public Site: Astro + Cloudflare Pages

**Satisfies**: `[S§5.5]` SEO, `[S§5]` public discovery content

| Attribute | Value |
|-----------|-------|
| **Framework** | Astro 5+ |
| **Hosting** | Cloudflare Pages |
| **Content** | Methodology explorer, state guides, Homeschooling 101, blog |
| **Rendering** | Static Site Generation (SSG) |

**Rationale**: Discovery content `[S§5]` must be SEO-indexable and fast. Astro generates static HTML with perfect Lighthouse scores. Cloudflare Pages hosts static sites for free with global CDN. This completely eliminates SSR as a backend concern — the Rust API serves JSON only.

**Rejected alternatives**:
- **SSR from Rust backend**: Adds template rendering complexity to the API server. Mixes concerns (API + HTML rendering). Rust's template ecosystem (Askama, Tera) is functional but unnecessary.
- **Next.js static export**: Heavier runtime than Astro for content-focused pages. Astro's island architecture is better suited for mostly-static content with optional interactive components.

**Revision trigger**: None. Static site generation is the correct pattern for SEO content.

### 2.5 Database: PostgreSQL 16

**Satisfies**: `[S§16]` data architecture, `[S§14]` search, `[S§7.8]` location, `[S§4.1]` methodology config

| Attribute | Value |
|-----------|-------|
| **Version** | PostgreSQL 16+ |
| **Extensions** | `pg_trgm`, `PostGIS`, `pgcrypto`, `uuid-ossp` |
| **Connection pooling** | PgBouncer (transaction mode) |

**Capabilities used**:
- **JSONB**: Methodology configuration, tool registry, quiz scoring weights `[S§4.1]`
- **PostGIS**: Location-based discovery with coarse-grained geometry `[S§7.8]`
- **Full-text search**: `tsvector` + `pg_trgm` for Phase 1 search `[S§14]`
- **Arrays**: Multi-tag storage (methodology tags, subject tags) `[S§9.2]`
- **Row-level security (RLS)**: Defense-in-depth for family data isolation `[S§16.2]`

**Rejected alternatives**:
- **MySQL**: No JSONB, no PostGIS, weaker full-text search. PostgreSQL is strictly superior for this use case.
- **MongoDB**: Loses relational integrity across 14 interconnected domains. The data model `[S§16.1]` is fundamentally relational.

**Revision trigger**: Never for the primary database. Add read replicas when write throughput exceeds single-server capacity (~Phase 3).

### 2.6 Search: PostgreSQL FTS → Meilisearch

**Satisfies**: `[S§14]` search, `[S§9.3]` marketplace discovery

**Phase 1**: PostgreSQL `tsvector` + `pg_trgm` indexes per search scope.
- Social search (users, groups, events)
- Marketplace search (listings by title, description, tags)
- Learning search (family-scoped: activities, journals, reading lists)

**Phase 2+**: Meilisearch for marketplace and social search.
- Typo-tolerant, faceted filtering `[S§9.3]`, instant autocomplete `[S§14.2]`
- PostgreSQL FTS retained for family-scoped learning search (smaller dataset, privacy-sensitive)

**Revision trigger**: Migrate to Meilisearch when marketplace exceeds ~100K listings or search latency exceeds 500ms p95 `[S§17.3]`.

### 2.7 Background Jobs: Redis + sidekiq-rs

**Satisfies**: `[S§13]` notifications, `[S§12]` moderation pipeline, `[S§14]` search indexing

| Attribute | Value |
|-----------|-------|
| **Queue backend** | Redis 7+ |
| **Job processor** | `sidekiq-rs` (Rust port of Sidekiq) |
| **Priority tiers** | Critical, Default, Low |

Redis also serves as:
- **Cache layer** — methodology config, session data, rate limiting counters
- **Pub/sub** — WebSocket message distribution for real-time features `[S§7.5]`
- **Sorted sets** — Social feed fan-out `[S§7.2]`

**Revision trigger**: If job throughput exceeds Redis single-instance capacity (~100K jobs/sec), add Redis Cluster. This is unlikely before Phase 4.

### 2.8 Authentication: Ory Kratos

**Satisfies**: `[S§3]` accounts & permissions, `[S§17.1]` security, `[S§17.2]` COPPA

| Attribute | Value |
|-----------|-------|
| **Service** | Ory Kratos (self-hosted, sidecar) |
| **OIDC providers** | Google, Facebook, Apple `[S§6.1]` |
| **MFA** | TOTP + WebAuthn `[S§17.1]` |
| **Session management** | Kratos cookie-based sessions |

**Rationale**: Building auth from scratch in Rust means implementing password hashing, session management, OIDC flows, MFA, account recovery, and email verification. Kratos handles all of this as a battle-tested, self-hosted identity service. Custom COPPA consent flow `[S§17.2]` is built on top of Kratos hooks.

**Rejected alternatives**:
- **Auth0 / Firebase Auth**: SaaS dependency, per-MAU pricing becomes expensive at scale. At 100K+ families, Auth0 costs $1,000+/mo vs. self-hosted Kratos at $0.
- **Custom Rust auth**: Months of development for a solved problem. Security-critical code that should not be custom-written by a solo developer.

**Revision trigger**: None. Kratos is open-source and self-hosted — no vendor lock-in.

### 2.9 Payments: Stripe + Stripe Connect

**Satisfies**: `[S§15]` billing, `[S§9.6]` marketplace payouts

| Attribute | Value |
|-----------|-------|
| **Subscriptions** | Stripe Billing `[S§15.3]` |
| **Marketplace** | Stripe Connect (Standard accounts) `[S§9.6]` |
| **Sales tax** | Stripe Tax `[S§15.4]` |
| **1099-K** | Stripe handles for Connected accounts `[S§9.6]` |

**Rationale**: Stripe Connect Standard accounts offload creator KYC, identity verification, and 1099-K filing to Stripe. The platform never handles sensitive financial data for creators. This dramatically reduces compliance burden for a solo developer.

**Revision trigger**: None. Stripe is the industry standard for this pattern.

### 2.10 File Storage: Cloudflare R2

**Satisfies**: `[S§14]` content & media, `[S§9.2]` marketplace files

| Attribute | Value |
|-----------|-------|
| **Storage** | Cloudflare R2 (S3-compatible) |
| **CDN** | Cloudflare CDN (automatic with R2) |
| **Egress cost** | $0 (R2's key differentiator) |

**Rationale**: A media-heavy platform (profile photos, learning journals with images, marketplace content files, nature study photos) generates significant egress. R2's zero egress pricing saves potentially thousands per month at scale vs. AWS S3.

**Revision trigger**: None. S3-compatible API means migration is straightforward if needed.

### 2.11 Hosting: Hetzner Dedicated Server

**Satisfies**: `[S§17.3]` performance, `[S§17.5]` availability

| Attribute | Value |
|-----------|-------|
| **Server** | Hetzner AX52 (or equivalent): 8-core AMD, 64GB RAM, 2x1TB NVMe |
| **Cost** | ~$60/mo |
| **Location** | US East (Ashburn) or EU (Falkenstein) |
| **OS** | Ubuntu 24.04 LTS |

**Rationale**: Rust's efficiency means a single Hetzner dedicated server handles Phase 1-2 comfortably. At $60/mo vs. $500-2,000/mo for equivalent AWS/GCP instances, infrastructure costs stay minimal while the platform grows.

**Scaling path**:
- **Phase 1-2**: Single server (this spec)
- **Phase 2-3**: Add a second server behind a load balancer (Hetzner Load Balancer or Caddy reverse proxy)
- **Phase 3+**: PostgreSQL read replicas, dedicated Redis server, media processing worker

**Revision trigger**: Scale when sustained CPU utilization exceeds 70% or memory utilization exceeds 80% during peak hours.

### 2.12 Email: Postmark

**Satisfies**: `[S§13]` notifications

| Attribute | Value |
|-----------|-------|
| **Service** | Postmark |
| **Transactional** | Account verification, password reset, purchase receipts, social notifications |
| **Broadcast** | Digest emails, platform updates (separate Postmark stream) |

**Rationale**: Best transactional email deliverability in the industry. Separate broadcast stream keeps transactional deliverability unaffected by marketing sends. CAN-SPAM compliance built in `[S§13.3]`.

**Revision trigger**: None anticipated. Postmark scales to millions of emails/month.

### 2.13 CSAM Detection: Thorn Safer + AWS Rekognition

**Satisfies**: `[S§12.1]` CSAM detection, `[S§12.2]` content moderation

| Component | Service | Purpose |
|-----------|---------|---------|
| **Hash matching** | Thorn Safer | PhotoDNA hash matching against NCMEC database, automated NCMEC reporting `[S§12.1]` |
| **Visual moderation** | AWS Rekognition | General content moderation (explicit content, violence) `[S§12.2]` |

**Rationale**: Thorn Safer is a purpose-built nonprofit tool specifically designed for CSAM detection and NCMEC reporting — it handles the legal reporting workflow. AWS Rekognition provides general image moderation for non-CSAM policy violations.

**Revision trigger**: None. CSAM detection is a legal requirement `[S§12.1]`.

### 2.14 Monitoring: Sentry + UptimeRobot

**Satisfies**: `[S§17.5]` availability

| Component | Service | Purpose |
|-----------|---------|---------|
| **Error tracking** | Sentry | Rust + React error capture, performance monitoring |
| **Uptime** | UptimeRobot | External availability monitoring, alerting |

**Revision trigger**: Add Grafana + Prometheus when operating multiple servers (Phase 3+).

### 2.15 CI/CD: GitHub Actions

**Satisfies**: `[S§17.1]` security (dependency scanning)

Pipeline stages:
1. `cargo clippy` — lint
2. `cargo test` — unit + integration tests
3. `cargo audit` — dependency vulnerability scan
4. `npm audit` — frontend dependency scan
5. Docker multi-stage build
6. Deploy to Hetzner via SSH + Docker

**Revision trigger**: None. GitHub Actions is sufficient for any scale of this project.

### 2.16 Real-Time: WebSockets via Axum

**Satisfies**: `[S§7.5]` direct messaging, `[S§13]` notifications

| Attribute | Value |
|-----------|-------|
| **Protocol** | WebSocket (RFC 6455) |
| **Server** | Axum's built-in WebSocket support |
| **Pub/sub** | Redis pub/sub for multi-connection distribution |

**Rationale**: Axum has native async WebSocket support via `tokio-tungstenite`. Redis pub/sub distributes messages across WebSocket connections (and across servers when scaling horizontally).

**Revision trigger**: None for the WebSocket layer. Redis pub/sub scales to the connection counts needed through Phase 3.

---

## 3. System Architecture

### 3.1 High-Level Architecture Diagram

```
                                    ┌─────────────────────────────┐
                                    │      Cloudflare CDN         │
                                    │  (R2 media + Pages static)  │
                                    └──────────┬──────────────────┘
                                               │
                          ┌────────────────────┼────────────────────┐
                          │                    │                    │
                 ┌────────▼────────┐  ┌───────▼────────┐  ┌───────▼────────┐
                 │   Astro Site    │  │   React SPA    │  │  Mobile Apps   │
                 │ (Cloudflare     │  │   (Vite)       │  │  (Phase 3+)    │
                 │  Pages - SSG)   │  │                │  │                │
                 │                 │  │  ┌──────────┐  │  │                │
                 │ • Method. quiz  │  │  │ TanStack │  │  │                │
                 │ • State guides  │  │  │  Query   │  │  │                │
                 │ • Explorer      │  │  └────┬─────┘  │  │                │
                 │ • 101 content   │  │       │        │  │                │
                 │ • Blog          │  │       │ JSON   │  │                │
                 └────────┬────────┘  └───────┼────────┘  └───────┬────────┘
                          │ (API calls        │                    │
                          │  for quiz data)   │                    │
                          └───────────────────┼────────────────────┘
                                              │
                                     ┌────────▼────────┐
                                     │   Rust API      │
                                     │   (Axum)        │
                                     │                 │
                                     │  ┌───────────┐  │
                                     │  │  Domains  │  │
                                     │  │           │  │
                                     │  │ iam::     │  │
                                     │  │ social::  │  │
                                     │  │ learn::   │  │
                                     │  │ mkt::     │  │
                                     │  │ method::  │  │
                                     │  │ discover::│  │
                                     │  │ onboard:: │  │
                                     │  │ billing:: │  │
                                     │  │ notify::  │  │
                                     │  │ search::  │  │
                                     │  │ comply::  │  │
                                     │  │ safety::  │  │
                                     │  │ ai::      │  │
                                     │  │ media::   │  │
                                     │  └───────────┘  │
                                     │                 │
                                     │  WebSocket ─────┤
                                     └──┬───┬───┬───┬──┘
                                        │   │   │   │
                     ┌──────────────────┘   │   │   └──────────────────┐
                     │              ┌───────┘   └───────┐              │
              ┌──────▼──────┐ ┌────▼─────┐  ┌──────────▼──┐  ┌───────▼───────┐
              │ PostgreSQL  │ │  Redis   │  │ Ory Kratos  │  │  External     │
              │    16       │ │   7+     │  │  (sidecar)  │  │  Services     │
              │             │ │          │  │             │  │               │
              │ • Domains   │ │ • Cache  │  │ • Auth      │  │ • Stripe      │
              │ • JSONB     │ │ • Jobs   │  │ • OIDC      │  │ • Postmark    │
              │ • PostGIS   │ │ • Pub/sub│  │ • MFA       │  │ • Thorn Safer │
              │ • FTS       │ │ • Feed   │  │ • Sessions  │  │ • Rekognition │
              │ • RLS       │ │          │  │             │  │ • R2          │
              └─────────────┘ └──────────┘  └─────────────┘  └───────────────┘
```

### 3.2 Domain-to-Module Mapping

Each spec domain `[S§2.1]` maps to a Rust module within the monolith:

| Spec Domain | Rust Module | Spec Reference | Key Responsibilities |
|-------------|-------------|----------------|---------------------|
| Identity & Access | `iam::` | `[S§3]` | Users, families, roles, permissions, sessions |
| Methodology | `method::` | `[S§4]` | Definitions, tool registry, config propagation |
| Discovery | `discover::` | `[S§5]` | Quiz engine, explorer content, state guides |
| Onboarding | `onboard::` | `[S§6]` | Account setup wizard, roadmaps, recommendations |
| Social | `social::` | `[S§7]` | Profiles, feed, friends, messaging, groups, events |
| Learning | `learn::` | `[S§8]` | Tools, activities, journals, progress tracking |
| Marketplace | `mkt::` | `[S§9]` | Listings, purchases, reviews, creator dashboard |
| AI & Recommendations | `ai::` | `[S§10]` | Recommendation engine, content suggestions |
| Compliance & Reporting | `comply::` | `[S§11]` | Attendance, assessments, portfolios, transcripts |
| Trust & Safety | `safety::` | `[S§12]` | CSAM, moderation, reporting, bot prevention |
| Billing & Subscriptions | `billing::` | `[S§15]` | Subscriptions, transactions, payouts |
| Notifications | `notify::` | `[S§13]` | In-app, email, digests, preferences |
| Search | `search::` | `[S§14]` | Full-text search, autocomplete, faceted filtering |
| Content & Media | `media::` | `[S§2.1]` | Upload, processing, storage, delivery |

### 3.3 Request Flow

A typical authenticated API request flows through these layers:

```
HTTP Request
    │
    ▼
┌─────────────────────┐
│ Axum Router         │  Route matching
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ Rate Limit Layer    │  Token bucket per IP/user [S§2.3]
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ Auth Middleware      │  Validate Kratos session → extract AuthContext
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ Family Scope Layer  │  Extract family_id from AuthContext
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ Domain Handler      │  Business logic (e.g., social::create_post)
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ Service Layer       │  Orchestrates DB + cache + external services
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ Repository Layer    │  SeaORM queries (always family-scoped)
└─────────────────────┘
```

### 3.4 Module Structure

Each domain module follows a consistent internal structure:

```
src/
├── main.rs
├── app.rs              # Axum app builder, router composition
├── config.rs           # Environment configuration
├── error.rs            # Application error types
├── middleware/
│   ├── auth.rs         # Kratos session validation
│   ├── family_scope.rs # Family context extraction
│   └── rate_limit.rs   # Rate limiting
├── domains/
│   ├── iam/
│   │   ├── mod.rs
│   │   ├── handlers.rs     # Axum handlers (HTTP layer)
│   │   ├── service.rs      # Business logic
│   │   ├── repository.rs   # Database queries
│   │   ├── models.rs       # Request/response types
│   │   └── entities/       # SeaORM entities
│   ├── social/
│   │   ├── mod.rs
│   │   ├── handlers.rs
│   │   ├── service.rs
│   │   ├── repository.rs
│   │   ├── models.rs
│   │   └── entities/
│   ├── learn/
│   │   └── ...
│   ├── mkt/
│   │   └── ...
│   └── ... (all 14 domains)
└── shared/
    ├── db.rs           # Database pool, transaction helpers
    ├── redis.rs        # Redis connection pool
    ├── pagination.rs   # Cursor-based pagination
    ├── family_scope.rs # FamilyScoped trait
    └── types.rs        # Shared types (Uuid, DateTime, etc.)
```

---

## 4. Data Architecture

### 4.1 Schema Organization

Tables are prefixed by domain to avoid collision and provide clear ownership. `[S§16]`

| Domain | Prefix | Key Tables |
|--------|--------|------------|
| Identity & Access | `iam_` | `iam_families`, `iam_parents`, `iam_students`, `iam_roles` |
| Methodology | `method_` | `method_definitions`, `method_tool_registry`, `method_philosophy_modules` |
| Discovery | `disc_` | `disc_quiz_definitions`, `disc_quiz_results`, `disc_state_guides` |
| Onboarding | `onb_` | `onb_roadmaps`, `onb_wizard_progress` |
| Social | `soc_` | `soc_profiles`, `soc_posts`, `soc_comments`, `soc_friendships`, `soc_messages`, `soc_groups`, `soc_events` |
| Learning | `learn_` | `learn_activities`, `learn_assessments`, `learn_journals`, `learn_projects`, `learn_reading_lists`, `learn_progress` |
| Marketplace | `mkt_` | `mkt_creators`, `mkt_listings`, `mkt_purchases`, `mkt_reviews`, `mkt_files` |
| AI & Recs | `ai_` | `ai_signals`, `ai_recommendations` |
| Compliance | `comply_` | `comply_attendance`, `comply_state_configs`, `comply_portfolios` |
| Trust & Safety | `safety_` | `safety_reports`, `safety_mod_actions`, `safety_content_flags` |
| Billing | `bill_` | `bill_subscriptions`, `bill_transactions`, `bill_payouts` |
| Notifications | `notify_` | `notify_notifications`, `notify_preferences`, `notify_digests` |
| Search | `search_` | (uses FTS indexes on domain tables directly) |
| Content & Media | `media_` | `media_uploads`, `media_processing_jobs` |

### 4.2 Core Schema Design

#### Family & Identity `[S§3.1]`

```sql
-- Top-level family entity [S§3.1.1]
CREATE TABLE iam_families (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name    TEXT NOT NULL,
    state_code      CHAR(2),                    -- for compliance [S§6.2]
    location_region TEXT,                        -- coarse location [S§7.8]
    primary_parent_id UUID,                      -- set after first parent created
    primary_methodology_id UUID NOT NULL REFERENCES method_definitions(id),
    secondary_methodology_ids UUID[] DEFAULT '{}', -- array of methodology IDs [S§4.3]
    subscription_tier TEXT NOT NULL DEFAULT 'free' CHECK (subscription_tier IN ('free', 'premium')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Parent users [S§3.1.2]
CREATE TABLE iam_parents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    kratos_identity_id UUID NOT NULL UNIQUE,     -- links to Ory Kratos
    display_name    TEXT NOT NULL,
    email           TEXT NOT NULL,
    is_primary      BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_parents_family ON iam_parents(family_id);

-- Student profiles [S§3.1.3]
CREATE TABLE iam_students (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    display_name    TEXT NOT NULL,
    birth_year      SMALLINT,
    grade_level     TEXT,
    methodology_override_id UUID REFERENCES method_definitions(id), -- [S§4.6]
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_students_family ON iam_students(family_id);
```

#### Methodology System `[S§4.1]`

```sql
-- Platform-defined methodologies [S§4.1, S§4.5]
CREATE TABLE method_definitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,        -- e.g., 'charlotte-mason'
    display_name    TEXT NOT NULL,
    short_desc      TEXT NOT NULL,
    icon_url        TEXT,
    philosophy      JSONB NOT NULL DEFAULT '{}', -- philosophy module content
    onboarding_config JSONB NOT NULL DEFAULT '{}', -- roadmaps, starter recs [S§6.4]
    community_config JSONB NOT NULL DEFAULT '{}',  -- group IDs, mentor criteria [S§6.6]
    mastery_paths   JSONB NOT NULL DEFAULT '{}', -- beginner/intermediate/advanced [S§4.1]
    display_order   SMALLINT NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Learning tool definitions [S§4.2]
CREATE TABLE method_tools (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug            TEXT NOT NULL UNIQUE,        -- e.g., 'reading-lists'
    display_name    TEXT NOT NULL,
    description     TEXT,
    config_schema   JSONB NOT NULL DEFAULT '{}', -- JSON Schema for tool config
    tier            TEXT NOT NULL DEFAULT 'free' CHECK (tier IN ('free', 'premium')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Many-to-many: which tools are active per methodology [S§4.2]
CREATE TABLE method_tool_activations (
    methodology_id  UUID NOT NULL REFERENCES method_definitions(id),
    tool_id         UUID NOT NULL REFERENCES method_tools(id),
    config_overrides JSONB NOT NULL DEFAULT '{}', -- methodology-specific labels, guidance
    PRIMARY KEY (methodology_id, tool_id)
);
```

#### Social Layer `[S§7]`

```sql
-- Social profiles [S§7.1]
CREATE TABLE soc_profiles (
    family_id       UUID PRIMARY KEY REFERENCES iam_families(id),
    bio             TEXT,
    profile_photo_url TEXT,
    privacy_settings JSONB NOT NULL DEFAULT '{}', -- per-field visibility [S§7.1]
    location_visible BOOLEAN NOT NULL DEFAULT false, -- opt-in [S§7.8]
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Friendships (bidirectional) [S§7.4]
CREATE TABLE soc_friendships (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_family_id UUID NOT NULL REFERENCES iam_families(id),
    accepter_family_id  UUID NOT NULL REFERENCES iam_families(id),
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'accepted', 'blocked')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (requester_family_id, accepter_family_id)
);

CREATE INDEX idx_friendships_requester ON soc_friendships(requester_family_id, status);
CREATE INDEX idx_friendships_accepter ON soc_friendships(accepter_family_id, status);

-- Posts [S§7.2]
CREATE TABLE soc_posts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    author_parent_id UUID NOT NULL REFERENCES iam_parents(id),
    post_type       TEXT NOT NULL CHECK (post_type IN (
                        'text', 'photo', 'milestone', 'event_share',
                        'marketplace_review', 'resource_share'
                    )),
    content         TEXT,
    attachments     JSONB DEFAULT '[]',          -- array of media URLs
    group_id        UUID REFERENCES soc_groups(id), -- NULL = personal feed
    visibility      TEXT NOT NULL DEFAULT 'friends'
                    CHECK (visibility IN ('friends', 'group')), -- no 'public' [S§7.2.2]
    likes_count     INTEGER NOT NULL DEFAULT 0,
    comments_count  INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_posts_family ON soc_posts(family_id, created_at DESC);
CREATE INDEX idx_posts_group ON soc_posts(group_id, created_at DESC) WHERE group_id IS NOT NULL;

-- Full-text search index on posts [S§14.1]
ALTER TABLE soc_posts ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (to_tsvector('english', coalesce(content, ''))) STORED;
CREATE INDEX idx_posts_search ON soc_posts USING GIN(search_vector);

-- Direct messages [S§7.5]
CREATE TABLE soc_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL,               -- groups two users' messages
    sender_parent_id UUID NOT NULL REFERENCES iam_parents(id),
    recipient_parent_id UUID NOT NULL REFERENCES iam_parents(id),
    content         TEXT NOT NULL,
    attachments     JSONB DEFAULT '[]',
    read_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_messages_conversation ON soc_messages(conversation_id, created_at);
```

#### Learning Layer `[S§8]`

```sql
-- Activities [S§8.1.1]
CREATE TABLE learn_activities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    student_id      UUID NOT NULL REFERENCES iam_students(id),
    title           TEXT NOT NULL,
    description     TEXT,
    subject_tags    TEXT[] NOT NULL DEFAULT '{}',  -- from taxonomy [S§8.3]
    methodology_id  UUID REFERENCES method_definitions(id),
    tool_id         UUID REFERENCES method_tools(id),
    duration_minutes SMALLINT,
    attachments     JSONB DEFAULT '[]',
    activity_date   DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_activities_family_student ON learn_activities(family_id, student_id, activity_date DESC);

-- Full-text search on learning data (family-scoped) [S§14.1]
ALTER TABLE learn_activities ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_activities_search ON learn_activities USING GIN(search_vector);

-- Journal entries [S§8.1.4]
CREATE TABLE learn_journals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    student_id      UUID NOT NULL REFERENCES iam_students(id),
    entry_type      TEXT NOT NULL CHECK (entry_type IN ('freeform', 'narration', 'reflection')),
    content         TEXT NOT NULL,
    subject_tags    TEXT[] DEFAULT '{}',
    attachments     JSONB DEFAULT '[]',
    entry_date      DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_journals_family_student ON learn_journals(family_id, student_id, entry_date DESC);

-- Reading lists [S§8.1.3]
CREATE TABLE learn_reading_lists (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    name            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE learn_reading_list_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    list_id         UUID NOT NULL REFERENCES learn_reading_lists(id) ON DELETE CASCADE,
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    title           TEXT NOT NULL,
    author          TEXT,
    isbn            TEXT,
    subject_tags    TEXT[] DEFAULT '{}',
    status          TEXT NOT NULL DEFAULT 'to_read'
                    CHECK (status IN ('to_read', 'in_progress', 'completed')),
    student_id      UUID REFERENCES iam_students(id),
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reading_items_list ON learn_reading_list_items(list_id);
CREATE INDEX idx_reading_items_family ON learn_reading_list_items(family_id);
```

#### Marketplace `[S§9]`

```sql
-- Creator accounts [S§9.1]
CREATE TABLE mkt_creators (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id       UUID REFERENCES iam_parents(id), -- may be NULL for standalone
    stripe_account_id TEXT,                           -- Stripe Connect [S§15.4]
    store_name      TEXT NOT NULL,
    store_bio       TEXT,
    store_logo_url  TEXT,
    verified        BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Content listings [S§9.2]
CREATE TABLE mkt_listings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id      UUID NOT NULL REFERENCES mkt_creators(id),
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    price_cents     INTEGER NOT NULL CHECK (price_cents >= 0), -- 0 = free
    methodology_tags UUID[] NOT NULL,             -- references method_definitions
    subject_tags    TEXT[] NOT NULL,
    grade_min       SMALLINT,
    grade_max       SMALLINT,
    content_type    TEXT NOT NULL CHECK (content_type IN (
                        'curriculum', 'worksheet', 'unit_study',
                        'video', 'book_list', 'assessment'
                    )),
    worldview_tags  TEXT[] DEFAULT '{}',           -- [S§9.2.1]
    preview_url     TEXT,
    thumbnail_url   TEXT,
    status          TEXT NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'submitted', 'published', 'archived')),
    rating_avg      NUMERIC(3,2) DEFAULT 0,
    rating_count    INTEGER DEFAULT 0,
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Full-text search on listings [S§9.3]
ALTER TABLE mkt_listings ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_listings_search ON mkt_listings USING GIN(search_vector);
CREATE INDEX idx_listings_methodology ON mkt_listings USING GIN(methodology_tags);
CREATE INDEX idx_listings_status ON mkt_listings(status) WHERE status = 'published';

-- Purchases [S§9.4]
CREATE TABLE mkt_purchases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    listing_id      UUID NOT NULL REFERENCES mkt_listings(id),
    stripe_payment_id TEXT,
    amount_cents    INTEGER NOT NULL,
    platform_fee_cents INTEGER NOT NULL,
    creator_payout_cents INTEGER NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (family_id, listing_id)                -- one purchase per listing per family
);

-- Reviews (verified purchaser only) [S§9.5]
CREATE TABLE mkt_reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id      UUID NOT NULL REFERENCES mkt_listings(id),
    purchase_id     UUID NOT NULL REFERENCES mkt_purchases(id),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    rating          SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    review_text     TEXT,
    creator_response TEXT,                        -- [S§9.5]
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (purchase_id)                          -- one review per purchase
);
```

### 4.3 JSONB Usage Patterns

Methodology configuration uses JSONB for flexible, schema-less data that varies per methodology. `[S§4.1]`

```sql
-- Example: Charlotte Mason methodology config
INSERT INTO method_definitions (slug, display_name, short_desc, philosophy, onboarding_config)
VALUES (
    'charlotte-mason',
    'Charlotte Mason',
    'Living books, narration, nature study, short lessons, and habit formation',
    '{
        "principles": ["Living books over textbooks", "Short lessons", "Narration as assessment"],
        "history": "Developed by Charlotte Mason (1842-1923)...",
        "typical_day": "Morning: short math lesson (20min), narration from living book..."
    }',
    '{
        "roadmap_steps": [
            {"order": 1, "title": "Choose your first living books", "tool_link": "reading-lists"},
            {"order": 2, "title": "Set up nature journal", "tool_link": "nature-journals"},
            {"order": 3, "title": "Plan short lessons", "description": "Keep lessons to 15-20 minutes"}
        ],
        "starter_curriculum_tags": ["charlotte-mason", "living-books", "nature-study"]
    }'
);
```

### 4.4 PostGIS for Location Discovery `[S§7.8]`

```sql
-- Add PostGIS geometry for coarse location
ALTER TABLE iam_families ADD COLUMN location_point GEOMETRY(Point, 4326);

-- Only set from city/region centroid — never from precise coordinates [S§7.8]
-- Example: set family location to city centroid
UPDATE iam_families
SET location_point = ST_SetSRID(ST_MakePoint(-73.935242, 40.730610), 4326)
WHERE id = '...';

-- Find families within 50km radius (opt-in only)
SELECT f.id, f.display_name,
       ST_Distance(f.location_point::geography, target.location_point::geography) / 1000 as km
FROM iam_families f
JOIN soc_profiles sp ON sp.family_id = f.id AND sp.location_visible = true
CROSS JOIN (SELECT location_point FROM iam_families WHERE id = $1) target
WHERE ST_DWithin(f.location_point::geography, target.location_point::geography, 50000)
  AND f.id != $1
ORDER BY km;
```

### 4.5 Entity Relationship Summary

```
iam_families ──┬── 1:N ── iam_parents ──── 0:1 ── mkt_creators
               │                                        │
               ├── 0:N ── iam_students                  └── 1:N ── mkt_listings
               │                                                       │
               ├── 1:1 ── soc_profiles                                 │
               │                                                       │
               ├── N:M ── soc_friendships (self-referential)           │
               │                                                       │
               ├── 1:N ── soc_posts                    mkt_purchases ──┤
               │                                       (family ↔ listing)
               ├── 1:N ── learn_activities                             │
               │                                       mkt_reviews ────┘
               ├── 1:N ── learn_journals               (verified purchaser)
               │
               ├── 1:N ── learn_reading_lists
               │
               └── N:M ── soc_groups (via soc_group_members)

method_definitions ── N:M ── method_tools (via method_tool_activations)
```

---

## 5. Authentication & Authorization

### 5.1 Ory Kratos Configuration

Kratos runs as a sidecar container alongside the Rust API, managing the full authentication lifecycle. `[S§3]`

```yaml
# kratos.yml (key configuration)
selfservice:
  default_browser_return_url: https://app.homegrown.academy/

  flows:
    registration:
      after:
        password:
          hooks:
            - hook: web_hook
              config:
                url: http://api:3000/hooks/kratos/post-registration
                method: POST
                # Triggers family account creation [S§6.1]

    login:
      after:
        password:
          hooks:
            - hook: web_hook
              config:
                url: http://api:3000/hooks/kratos/post-login
                method: POST

    verification:
      enabled: true
      # Email verification required [S§6.1]

    recovery:
      enabled: true
      # Password reset flow

  methods:
    password:
      enabled: true
      config:
        min_password_length: 10
        identifier_similarity_check_enabled: true

    oidc:
      enabled: true
      config:
        providers:
          - id: google
            provider: google
            client_id: ${GOOGLE_CLIENT_ID}
            client_secret: ${GOOGLE_CLIENT_SECRET}
            mapper_url: file:///etc/kratos/oidc/google.jsonnet
            scope:
              - email
              - profile
          - id: facebook
            provider: facebook
            client_id: ${FACEBOOK_CLIENT_ID}
            client_secret: ${FACEBOOK_CLIENT_SECRET}
          - id: apple
            provider: apple
            client_id: ${APPLE_CLIENT_ID}
            client_secret: ${APPLE_CLIENT_SECRET}

    totp:
      enabled: true
      # MFA via authenticator app [S§17.1]

    webauthn:
      enabled: true
      # MFA via hardware key / passkey [S§17.1]

session:
  lifespan: 720h  # 30 days
  cookie:
    same_site: Lax
```

### 5.2 Auth Middleware (Rust)

```rust
use axum::{extract::State, http::Request, middleware::Next, response::Response};
use uuid::Uuid;

/// Authenticated user context extracted from Kratos session
#[derive(Clone, Debug)]
pub struct AuthContext {
    pub parent_id: Uuid,
    pub family_id: Uuid,
    pub kratos_identity_id: Uuid,
    pub is_primary_parent: bool,
    pub subscription_tier: SubscriptionTier,
    pub email: String,
}

#[derive(Clone, Debug, PartialEq)]
pub enum SubscriptionTier {
    Free,
    Premium,
}

/// Middleware: validates Kratos session cookie and builds AuthContext
pub async fn auth_middleware(
    State(state): State<AppState>,
    mut req: Request<axum::body::Body>,
    next: Next,
) -> Result<Response, ApiError> {
    // Extract session cookie
    let session_cookie = req
        .headers()
        .get("cookie")
        .and_then(|v| v.to_str().ok())
        .ok_or(ApiError::Unauthorized)?;

    // Validate with Kratos
    let kratos_session = state
        .kratos_client
        .to_session(Some(session_cookie), None)
        .await
        .map_err(|_| ApiError::Unauthorized)?;

    let kratos_identity_id: Uuid = kratos_session
        .identity
        .id
        .parse()
        .map_err(|_| ApiError::Unauthorized)?;

    // Look up parent + family from our DB
    let parent = state
        .db
        .find_parent_by_kratos_id(kratos_identity_id)
        .await?
        .ok_or(ApiError::Unauthorized)?;

    let family = state
        .db
        .find_family(parent.family_id)
        .await?
        .ok_or(ApiError::Unauthorized)?;

    let auth_context = AuthContext {
        parent_id: parent.id,
        family_id: family.id,
        kratos_identity_id,
        is_primary_parent: parent.is_primary,
        subscription_tier: match family.subscription_tier.as_str() {
            "premium" => SubscriptionTier::Premium,
            _ => SubscriptionTier::Free,
        },
        email: parent.email,
    };

    // Insert into request extensions for handlers to extract
    req.extensions_mut().insert(auth_context);
    Ok(next.run(req).await)
}
```

### 5.3 COPPA Consent State Machine `[S§17.2]`

COPPA consent is tracked per family, built on top of Kratos's registration flow:

```
                    ┌─────────────┐
                    │  REGISTERED │
                    │ (no consent)│
                    └──────┬──────┘
                           │
                    Parent acknowledges
                    COPPA notice [S§6.1]
                           │
                    ┌──────▼──────┐
                    │   NOTICED   │
                    │             │
                    └──────┬──────┘
                           │
                    Parent provides
                    verifiable consent
                           │
                    ┌──────▼──────┐
                    │  CONSENTED  │◄──── Can add student profiles
                    │             │      Can use learning tools
                    └──────┬──────┘
                           │
                    Parent requests    Parent withdraws
                    re-verification    consent
                           │                │
                    ┌──────▼──────┐   ┌─────▼──────┐
                    │ RE-VERIFIED │   │  WITHDRAWN  │
                    │             │   │             │
                    └─────────────┘   └─────────────┘
                                      Student data exported
                                      then deleted [S§16.3]
```

```rust
#[derive(Debug, Clone, PartialEq, sqlx::Type)]
#[sqlx(type_name = "text")]
pub enum CoppaConsentStatus {
    Registered,
    Noticed,
    Consented,
    ReVerified,
    Withdrawn,
}

/// Middleware: ensures COPPA consent before accessing student data
pub async fn require_coppa_consent(
    auth: AuthContext,
    State(state): State<AppState>,
    req: Request<axum::body::Body>,
    next: Next,
) -> Result<Response, ApiError> {
    let family = state.db.find_family(auth.family_id).await?
        .ok_or(ApiError::NotFound)?;

    match family.coppa_consent_status {
        CoppaConsentStatus::Consented | CoppaConsentStatus::ReVerified => {
            Ok(next.run(req).await)
        }
        _ => Err(ApiError::CoppaConsentRequired),
    }
}
```

### 5.4 Family Account Model `[S§3.1]`

```rust
/// Post-registration hook: creates family + parent atomically [S§6.1]
pub async fn handle_post_registration(
    State(state): State<AppState>,
    Json(payload): Json<KratosWebhookPayload>,
) -> Result<Json<()>, ApiError> {
    let txn = state.db.begin().await?;

    // Create family with default methodology
    let default_methodology = method::repository::find_by_slug(&txn, "traditional").await?;

    let family = iam::repository::create_family(
        &txn,
        CreateFamily {
            display_name: payload.traits.name.clone(),
            primary_methodology_id: default_methodology.id,
            // Methodology will be properly set during onboarding wizard [S§6.3]
        },
    ).await?;

    // Create parent linked to Kratos identity
    let parent = iam::repository::create_parent(
        &txn,
        CreateParent {
            family_id: family.id,
            kratos_identity_id: payload.identity_id,
            display_name: payload.traits.name,
            email: payload.traits.email,
            is_primary: true,
        },
    ).await?;

    // Set primary parent reference
    iam::repository::set_primary_parent(&txn, family.id, parent.id).await?;

    // Create social profile
    social::repository::create_profile(&txn, family.id).await?;

    txn.commit().await?;
    Ok(Json(()))
}
```

### 5.5 Role-Based Access Control `[S§3.2]`

Permission checks are implemented as Axum extractors:

```rust
/// Extractor: requires premium subscription [S§15.2]
pub struct RequirePremium(pub AuthContext);

#[axum::async_trait]
impl<S> FromRequestParts<S> for RequirePremium
where
    S: Send + Sync,
{
    type Rejection = ApiError;

    async fn from_request_parts(
        parts: &mut Parts,
        _state: &S,
    ) -> Result<Self, Self::Rejection> {
        let auth = parts
            .extensions
            .get::<AuthContext>()
            .cloned()
            .ok_or(ApiError::Unauthorized)?;

        if auth.subscription_tier != SubscriptionTier::Premium {
            return Err(ApiError::PremiumRequired);
        }

        Ok(RequirePremium(auth))
    }
}

/// Extractor: requires creator role [S§3.1.4]
pub struct RequireCreator {
    pub auth: AuthContext,
    pub creator_id: Uuid,
}

/// Usage in handlers:
async fn generate_portfolio(
    RequirePremium(auth): RequirePremium,  // enforces premium [S§11.4]
    State(state): State<AppState>,
    Path(student_id): Path<Uuid>,
) -> Result<Json<Portfolio>, ApiError> {
    // Verify student belongs to this family (family-scoped)
    let student = learn::repository::find_student(&state.db, auth.family_id, student_id)
        .await?
        .ok_or(ApiError::NotFound)?;

    // Generate portfolio...
    todo!()
}
```

---

## 6. Methodology System Implementation

### 6.1 Config-Driven Architecture `[S§4.1]`

The methodology system is the platform's most distinctive architectural pattern. Every methodology-dependent behavior resolves through configuration lookup — never through conditionals.

```rust
/// Resolves the active tool set for a family based on methodology selections [S§4.2]
pub async fn resolve_family_tools(
    db: &DatabaseConnection,
    family_id: Uuid,
) -> Result<Vec<ActiveTool>, DbErr> {
    // Get family's methodology selections
    let family = iam::repository::find_family(db, family_id).await?
        .ok_or(DbErr::RecordNotFound("family".into()))?;

    // Collect all methodology IDs (primary + secondary) [S§4.3]
    let mut methodology_ids = vec![family.primary_methodology_id];
    methodology_ids.extend(family.secondary_methodology_ids.iter());

    // Union of all activated tools across selected methodologies [S§4.2]
    let tools = method_tool_activations::Entity::find()
        .filter(method_tool_activations::Column::MethodologyId.is_in(methodology_ids))
        .find_also_related(method_tools::Entity)
        .all(db)
        .await?;

    // Deduplicate (a tool activated by multiple methodologies appears once)
    let mut seen = std::collections::HashSet::new();
    let active_tools: Vec<ActiveTool> = tools
        .into_iter()
        .filter_map(|(activation, tool)| {
            let tool = tool?;
            if seen.insert(tool.id) {
                Some(ActiveTool {
                    tool_id: tool.id,
                    slug: tool.slug,
                    display_name: tool.display_name,
                    tier: tool.tier,
                    config_overrides: activation.config_overrides,
                })
            } else {
                None
            }
        })
        .collect();

    Ok(active_tools)
}

/// Resolves tools for a specific student, considering per-student overrides [S§4.6]
pub async fn resolve_student_tools(
    db: &DatabaseConnection,
    family_id: Uuid,
    student_id: Uuid,
) -> Result<Vec<ActiveTool>, DbErr> {
    let student = iam::repository::find_student(db, family_id, student_id).await?
        .ok_or(DbErr::RecordNotFound("student".into()))?;

    match student.methodology_override_id {
        // Student has override — use their personal methodology [S§4.6]
        Some(override_id) => {
            let tools = method_tool_activations::Entity::find()
                .filter(method_tool_activations::Column::MethodologyId.eq(override_id))
                .find_also_related(method_tools::Entity)
                .all(db)
                .await?;

            Ok(tools.into_iter().filter_map(|(a, t)| {
                t.map(|tool| ActiveTool {
                    tool_id: tool.id,
                    slug: tool.slug,
                    display_name: tool.display_name,
                    tier: tool.tier,
                    config_overrides: a.config_overrides,
                })
            }).collect())
        }
        // No override — use family-level tools
        None => resolve_family_tools(db, family_id).await,
    }
}
```

### 6.2 Methodology-Aware API Responses

API responses include methodology context so the frontend renders appropriately:

```rust
/// Dashboard response shaped by methodology [S§4.4]
#[derive(Serialize)]
pub struct DashboardResponse {
    pub family: FamilySummary,
    pub students: Vec<StudentSummary>,
    pub active_tools: Vec<ActiveTool>,
    pub methodology_context: MethodologyContext,
    pub roadmap_progress: Option<RoadmapProgress>,  // [S§6.4]
}

#[derive(Serialize)]
pub struct MethodologyContext {
    pub primary: MethodologySummary,
    pub secondary: Vec<MethodologySummary>,
    /// Methodology-specific terminology overrides
    /// e.g., Charlotte Mason calls activities "lessons", Unschooling calls them "explorations"
    pub terminology: serde_json::Value,
    /// Current mastery path level [S§4.1]
    pub mastery_level: Option<String>,
}
```

### 6.3 Adding a New Methodology

Adding a new methodology (e.g., Reggio Emilia) requires zero code changes `[S§4.5]`:

```sql
-- 1. Insert methodology definition
INSERT INTO method_definitions (slug, display_name, short_desc, philosophy, onboarding_config)
VALUES (
    'reggio-emilia',
    'Reggio Emilia',
    'Child-led, project-based learning through art, exploration, and community',
    '{"principles": ["The child as protagonist", "The environment as third teacher", ...]}',
    '{"roadmap_steps": [...]}'
);

-- 2. Activate existing tools for this methodology
INSERT INTO method_tool_activations (methodology_id, tool_id, config_overrides)
VALUES
    -- Projects tool with Reggio-specific labels
    ((SELECT id FROM method_definitions WHERE slug = 'reggio-emilia'),
     (SELECT id FROM method_tools WHERE slug = 'projects'),
     '{"label": "Investigations", "guidance": "In Reggio Emilia, projects emerge from children''s interests..."}'),
    -- Journaling with documentation focus
    ((SELECT id FROM method_definitions WHERE slug = 'reggio-emilia'),
     (SELECT id FROM method_tools WHERE slug = 'journaling'),
     '{"label": "Documentation", "entry_types": ["observation", "documentation", "reflection"]}'),
    -- Activities
    ((SELECT id FROM method_definitions WHERE slug = 'reggio-emilia'),
     (SELECT id FROM method_tools WHERE slug = 'activities'),
     '{}');

-- 3. Create platform-managed group [S§7.6]
INSERT INTO soc_groups (name, description, group_type, methodology_id)
VALUES (
    'Reggio Emilia Community',
    'Connect with families using the Reggio Emilia approach',
    'platform_managed',
    (SELECT id FROM method_definitions WHERE slug = 'reggio-emilia')
);

-- Done. No deployment needed. The new methodology appears in:
-- - Methodology selection wizard [S§6.3]
-- - Methodology explorer (with Astro rebuild for static pages) [S§5.2]
-- - Tool resolution for families that select it
-- - Group and community discovery [S§6.6]
```

---

## 7. File & Media Architecture

### 7.1 Upload Pipeline `[S§14 Content & Media]`

All user file uploads follow the same pipeline, regardless of context (profile photos, journal images, marketplace content files):

```
Client                   API                      R2                 Workers
  │                       │                        │                    │
  │  1. Request upload    │                        │                    │
  │  ──────────────────►  │                        │                    │
  │                       │  2. Generate presigned  │                    │
  │                       │     PUT URL             │                    │
  │                       │  ─────────────────────► │                    │
  │  3. Presigned URL     │                        │                    │
  │  ◄──────────────────  │                        │                    │
  │                       │                        │                    │
  │  4. Upload directly   │                        │                    │
  │     to R2             │                        │                    │
  │  ─────────────────────────────────────────────►│                    │
  │                       │                        │                    │
  │  5. Confirm upload    │                        │                    │
  │  ──────────────────►  │                        │                    │
  │                       │  6. Enqueue processing  │                    │
  │                       │  ──────────────────────────────────────────►│
  │                       │                        │                    │
  │                       │                        │  7. CSAM scan      │
  │                       │                        │  ◄────────────────│
  │                       │                        │     (Thorn Safer)  │
  │                       │                        │                    │
  │                       │                        │  8. Image resize   │
  │                       │                        │  ◄────────────────│
  │                       │                        │     (thumbnails)   │
  │                       │                        │                    │
  │                       │  9. Mark published     │                    │
  │                       │  ◄─────────────────────────────────────────│
  │                       │     or quarantined     │                    │
```

```rust
/// Upload request handler
pub async fn request_upload(
    auth: AuthContext,
    State(state): State<AppState>,
    Json(req): Json<UploadRequest>,
) -> Result<Json<UploadResponse>, ApiError> {
    // Validate file type and size limits
    let max_size = match req.context {
        UploadContext::ProfilePhoto => 5 * 1024 * 1024,      // 5MB
        UploadContext::JournalImage => 10 * 1024 * 1024,     // 10MB
        UploadContext::MarketplaceFile => 500 * 1024 * 1024, // 500MB
        _ => 10 * 1024 * 1024,
    };

    let allowed_types = match req.context {
        UploadContext::ProfilePhoto => vec!["image/jpeg", "image/png", "image/webp"],
        UploadContext::JournalImage => vec!["image/jpeg", "image/png", "image/webp", "image/gif"],
        UploadContext::MarketplaceFile => vec!["application/pdf", "video/mp4", "image/jpeg", "image/png"],
        _ => vec!["image/jpeg", "image/png", "application/pdf"],
    };

    if !allowed_types.contains(&req.content_type.as_str()) {
        return Err(ApiError::InvalidFileType);
    }

    // Generate upload record
    let upload = media::repository::create_upload(
        &state.db,
        CreateUpload {
            family_id: auth.family_id,
            filename: req.filename,
            content_type: req.content_type.clone(),
            size_bytes: req.size_bytes,
            context: req.context,
            status: UploadStatus::Pending,
        },
    ).await?;

    // Generate presigned PUT URL for direct upload to R2
    let presigned_url = state.r2_client
        .presigned_put(&upload.storage_key(), max_size, &req.content_type)
        .await?;

    Ok(Json(UploadResponse {
        upload_id: upload.id,
        presigned_url,
        expires_in_seconds: 3600,
    }))
}
```

### 7.2 Image Processing

After upload confirmation, a background job handles image processing:

```rust
/// Background job: process uploaded image
pub async fn process_image(upload_id: Uuid, state: &AppState) -> Result<(), JobError> {
    let upload = media::repository::find_upload(&state.db, upload_id).await?
        .ok_or(JobError::NotFound)?;

    // 1. CSAM scan (all images) [S§12.1]
    if upload.content_type.starts_with("image/") {
        let scan_result = state.thorn_client.scan(&upload.storage_key()).await?;
        if scan_result.is_csam {
            // Quarantine immediately, report to NCMEC [S§12.1]
            media::repository::quarantine_upload(&state.db, upload_id).await?;
            safety::service::report_csam(&state, upload_id, scan_result).await?;
            return Ok(());
        }
    }

    // 2. Generate thumbnails for images
    if upload.content_type.starts_with("image/") {
        let variants = vec![
            ImageVariant { suffix: "thumb", max_width: 200, max_height: 200 },
            ImageVariant { suffix: "medium", max_width: 800, max_height: 800 },
        ];
        for variant in variants {
            let resized = resize_image(&upload, &variant).await?;
            state.r2_client.put(&variant_key(&upload, &variant), resized).await?;
        }
    }

    // 3. General content moderation [S§12.2]
    if upload.content_type.starts_with("image/") {
        let moderation = state.rekognition_client.detect_moderation_labels(&upload).await?;
        if moderation.has_violations() {
            media::repository::flag_upload(&state.db, upload_id, &moderation).await?;
            return Ok(());
        }
    }

    // 4. Mark as published
    media::repository::publish_upload(&state.db, upload_id).await?;
    Ok(())
}
```

### 7.3 Marketplace File Delivery `[S§9.4]`

Purchased marketplace files are delivered via time-limited signed URLs:

```rust
/// Download purchased marketplace content
pub async fn download_purchased_file(
    auth: AuthContext,
    State(state): State<AppState>,
    Path((listing_id, file_id)): Path<(Uuid, Uuid)>,
) -> Result<Json<DownloadResponse>, ApiError> {
    // Verify purchase exists [S§9.4]
    let _purchase = mkt::repository::find_purchase(&state.db, auth.family_id, listing_id)
        .await?
        .ok_or(ApiError::NotPurchased)?;

    let file = mkt::repository::find_listing_file(&state.db, listing_id, file_id)
        .await?
        .ok_or(ApiError::NotFound)?;

    // Generate time-limited signed download URL (1 hour)
    let signed_url = state.r2_client
        .presigned_get(&file.storage_key, 3600)
        .await?;

    Ok(Json(DownloadResponse { download_url: signed_url }))
}
```

---

## 8. Search Architecture

### 8.1 Phase 1: PostgreSQL Full-Text Search `[S§14]`

Phase 1 uses PostgreSQL's built-in full-text search capabilities. Three search scopes are implemented as defined in `[S§14.1]`:

#### Social Search

```sql
-- Users/families (respects privacy — friends + discoverable only) [S§14.2]
CREATE OR REPLACE FUNCTION search_families(
    searcher_family_id UUID,
    query TEXT,
    result_limit INT DEFAULT 20
) RETURNS TABLE(family_id UUID, display_name TEXT, rank REAL) AS $$
BEGIN
    RETURN QUERY
    SELECT f.id, f.display_name, ts_rank(to_tsvector('english', f.display_name), plainto_tsquery('english', query)) as rank
    FROM iam_families f
    JOIN soc_profiles sp ON sp.family_id = f.id
    WHERE (
        -- Friends [S§14.2]
        EXISTS (
            SELECT 1 FROM soc_friendships sf
            WHERE sf.status = 'accepted'
            AND (
                (sf.requester_family_id = searcher_family_id AND sf.accepter_family_id = f.id)
                OR (sf.accepter_family_id = searcher_family_id AND sf.requester_family_id = f.id)
            )
        )
        -- Or discoverable (opted into location-based discovery) [S§7.8]
        OR sp.location_visible = true
    )
    AND f.display_name ILIKE '%' || query || '%'
    ORDER BY rank DESC
    LIMIT result_limit;
END;
$$ LANGUAGE plpgsql;

-- Groups search [S§14.1]
CREATE INDEX idx_groups_search ON soc_groups
    USING GIN(to_tsvector('english', coalesce(name, '') || ' ' || coalesce(description, '')));

-- Events search [S§14.1]
CREATE INDEX idx_events_search ON soc_events
    USING GIN(to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, '')));
```

#### Marketplace Search `[S§9.3]`

```sql
-- Marketplace search with faceted filtering [S§9.3]
SELECT
    l.id, l.title, l.description, l.price_cents,
    l.rating_avg, l.rating_count,
    ts_rank(l.search_vector, websearch_to_tsquery('english', $1)) as relevance
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
  -- Faceted filters [S§9.3]
  AND ($2::uuid[] IS NULL OR l.methodology_tags && $2)   -- methodology filter
  AND ($3::text[] IS NULL OR l.subject_tags && $3)        -- subject filter
  AND ($4::int IS NULL OR l.grade_min <= $4)              -- grade range
  AND ($5::int IS NULL OR l.grade_max >= $5)
  AND ($6::int IS NULL OR l.price_cents <= $6)            -- max price
  AND ($7::text IS NULL OR l.content_type = $7)           -- content type
  AND ($8::text[] IS NULL OR l.worldview_tags && $8)      -- worldview filter
ORDER BY relevance DESC
LIMIT $9 OFFSET $10;
```

#### Learning Search (Family-Scoped) `[S§14.1]`

```sql
-- Learning data search — always scoped to family [S§14.2]
SELECT id, 'activity' as source, title, description, activity_date as date
FROM learn_activities
WHERE family_id = $1  -- ALWAYS family-scoped
  AND search_vector @@ websearch_to_tsquery('english', $2)

UNION ALL

SELECT id, 'journal' as source, content as title, NULL as description, entry_date as date
FROM learn_journals
WHERE family_id = $1
  AND to_tsvector('english', content) @@ websearch_to_tsquery('english', $2)

ORDER BY date DESC
LIMIT $3;
```

#### Autocomplete with pg_trgm `[S§14.2]`

```sql
-- Trigram index for fuzzy autocomplete
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_listings_title_trgm ON mkt_listings USING GIN(title gin_trgm_ops);

-- Autocomplete query
SELECT DISTINCT title, similarity(title, $1) as sim
FROM mkt_listings
WHERE status = 'published'
  AND title % $1  -- trigram similarity match
ORDER BY sim DESC
LIMIT 10;
```

### 8.2 Phase 2+: Meilisearch Migration Path

When marketplace exceeds ~100K listings or search latency exceeds 500ms p95:

```rust
/// Search service with dual-backend support
pub enum SearchBackend {
    PostgresFts,
    Meilisearch(MeilisearchClient),
}

/// Search service routes to appropriate backend
pub async fn search_marketplace(
    backend: &SearchBackend,
    query: &MarketplaceSearchQuery,
) -> Result<SearchResults, SearchError> {
    match backend {
        SearchBackend::PostgresFts => {
            // Use PostgreSQL FTS queries from §8.1
            postgres_marketplace_search(query).await
        }
        SearchBackend::Meilisearch(client) => {
            // Meilisearch provides typo tolerance, faceted filtering,
            // and instant search at any scale [S§14.2]
            let results = client
                .index("marketplace_listings")
                .search()
                .with_query(&query.text)
                .with_filter(&build_meilisearch_filter(query))
                .with_facets(&["methodology_tags", "subject_tags", "content_type", "worldview_tags"])
                .execute::<MarketplaceListing>()
                .await?;
            Ok(results.into())
        }
    }
}
```

**Migration strategy**: Zero-downtime switchover.
1. Index existing PostgreSQL data into Meilisearch.
2. Run both backends in parallel, comparing results (shadow mode).
3. Switch reads to Meilisearch.
4. Maintain PostgreSQL FTS indexes as fallback.
5. Remove PostgreSQL FTS indexes only after Meilisearch has proven stable for 30+ days.

---

## 9. API Design

### 9.1 RESTful JSON API `[S§17.3]`

The API follows REST conventions with consistent patterns across all 14 domains:

```
Base URL: https://api.homegrown.academy/v1

Authentication: Kratos session cookie (browser)
Content-Type: application/json
Rate Limiting: Token bucket per user (100 req/min default) [S§2.3]
```

### 9.2 URL Structure

```
# Identity & Access [S§3]
POST   /v1/families/students                    # Create student profile
GET    /v1/families/students                    # List students in family
PATCH  /v1/families/students/:id                # Update student
DELETE /v1/families/students/:id                # Remove student

# Methodology [S§4]
GET    /v1/methodologies                        # List all methodologies
GET    /v1/methodologies/:slug                  # Get methodology details
GET    /v1/families/tools                       # Get family's active tools

# Discovery [S§5] (mostly unauthenticated)
GET    /v1/discovery/quiz                       # Get quiz questions
POST   /v1/discovery/quiz/results               # Submit quiz answers
GET    /v1/discovery/quiz/results/:id           # Get quiz result
GET    /v1/discovery/methodologies/:slug         # Methodology explorer page
GET    /v1/discovery/state-guides/:state         # State legal guide

# Social [S§7]
GET    /v1/feed                                 # Get timeline feed
POST   /v1/posts                                # Create post
GET    /v1/posts/:id/comments                   # Get comments
POST   /v1/posts/:id/comments                   # Add comment
GET    /v1/friends                              # List friends
POST   /v1/friends/requests                     # Send friend request
PATCH  /v1/friends/requests/:id                 # Accept/reject
GET    /v1/messages/conversations               # List conversations
GET    /v1/messages/conversations/:id           # Get messages in conversation
POST   /v1/messages/conversations/:id           # Send message
GET    /v1/groups                               # List joined groups
GET    /v1/events                               # List events

# Learning [S§8]
POST   /v1/learning/activities                  # Log activity
GET    /v1/learning/activities                  # List activities (filterable)
POST   /v1/learning/journals                    # Create journal entry
GET    /v1/learning/journals                    # List journal entries
GET    /v1/learning/reading-lists               # List reading lists
POST   /v1/learning/reading-lists               # Create reading list
GET    /v1/learning/progress/:student_id        # Get progress summary

# Marketplace [S§9]
GET    /v1/marketplace/listings                 # Search/browse listings
GET    /v1/marketplace/listings/:id             # Get listing details
POST   /v1/marketplace/cart                     # Add to cart
POST   /v1/marketplace/checkout                 # Process purchase
POST   /v1/marketplace/listings/:id/reviews     # Leave review (verified purchaser)
GET    /v1/marketplace/purchases                # List purchases

# Search [S§14]
GET    /v1/search?scope=social&q=...            # Search families/groups/events
GET    /v1/search?scope=marketplace&q=...       # Search listings
GET    /v1/search?scope=learning&q=...          # Search own learning data
GET    /v1/search/autocomplete?q=...            # Type-ahead suggestions

# Notifications [S§13]
GET    /v1/notifications                        # List notifications
PATCH  /v1/notifications/:id/read               # Mark as read
GET    /v1/notifications/preferences            # Get preferences
PATCH  /v1/notifications/preferences            # Update preferences
```

### 9.3 Pagination `[S§17.3]`

All list endpoints use cursor-based pagination for consistent performance:

```rust
#[derive(Deserialize)]
pub struct PaginationParams {
    pub cursor: Option<String>,  // opaque cursor (base64-encoded ID + timestamp)
    pub limit: Option<u32>,      // default 20, max 100
}

#[derive(Serialize)]
pub struct PaginatedResponse<T: Serialize> {
    pub data: Vec<T>,
    pub next_cursor: Option<String>,
    pub has_more: bool,
}

/// Cursor encodes the last item's sort key for stable pagination
fn encode_cursor(id: Uuid, created_at: DateTime<Utc>) -> String {
    let raw = format!("{}:{}", id, created_at.timestamp_millis());
    base64::engine::general_purpose::URL_SAFE_NO_PAD.encode(raw)
}

fn decode_cursor(cursor: &str) -> Result<(Uuid, DateTime<Utc>), ApiError> {
    let raw = base64::engine::general_purpose::URL_SAFE_NO_PAD
        .decode(cursor)
        .map_err(|_| ApiError::InvalidCursor)?;
    let raw = String::from_utf8(raw).map_err(|_| ApiError::InvalidCursor)?;
    let parts: Vec<&str> = raw.splitn(2, ':').collect();
    let id = Uuid::parse_str(parts[0]).map_err(|_| ApiError::InvalidCursor)?;
    let ts = parts[1].parse::<i64>().map_err(|_| ApiError::InvalidCursor)?;
    let created_at = DateTime::from_timestamp_millis(ts).ok_or(ApiError::InvalidCursor)?;
    Ok((id, created_at))
}
```

### 9.4 OpenAPI & TypeScript Client Generation

```rust
// Using utoipa for OpenAPI spec generation from Rust types
use utoipa::ToSchema;

#[derive(Serialize, Deserialize, ToSchema)]
pub struct CreateActivityRequest {
    /// Activity title
    pub title: String,
    /// Optional description
    pub description: Option<String>,
    /// Student this activity is for
    pub student_id: Uuid,
    /// Subject tags from taxonomy [S§8.3]
    pub subject_tags: Vec<String>,
    /// Activity date
    pub activity_date: chrono::NaiveDate,
    /// Duration in minutes
    pub duration_minutes: Option<i16>,
}

#[derive(Serialize, Deserialize, ToSchema)]
pub struct ActivityResponse {
    pub id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub student_id: Uuid,
    pub student_name: String,
    pub subject_tags: Vec<String>,
    pub activity_date: chrono::NaiveDate,
    pub duration_minutes: Option<i16>,
    pub attachments: Vec<MediaAttachment>,
    pub created_at: DateTime<Utc>,
}
```

OpenAPI spec is generated at build time and used to auto-generate the TypeScript client:

```bash
# CI/CD pipeline step: generate TypeScript client from OpenAPI spec
npx openapi-typescript-codegen \
    --input ./openapi.json \
    --output ../frontend/src/api/generated \
    --client fetch
```

### 9.5 Error Response Format

```rust
#[derive(Serialize)]
pub struct ErrorResponse {
    pub error: ErrorBody,
}

#[derive(Serialize)]
pub struct ErrorBody {
    pub code: String,           // machine-readable: "PREMIUM_REQUIRED"
    pub message: String,        // human-readable: "This feature requires a premium subscription"
    pub details: Option<serde_json::Value>,  // optional structured details
}

/// Maps domain errors to HTTP responses
impl IntoResponse for ApiError {
    fn into_response(self) -> Response {
        let (status, code, message) = match self {
            ApiError::NotFound => (StatusCode::NOT_FOUND, "NOT_FOUND", "Resource not found"),
            ApiError::Unauthorized => (StatusCode::UNAUTHORIZED, "UNAUTHORIZED", "Authentication required"),
            ApiError::Forbidden => (StatusCode::FORBIDDEN, "FORBIDDEN", "Access denied"),
            ApiError::PremiumRequired => (StatusCode::PAYMENT_REQUIRED, "PREMIUM_REQUIRED", "Premium subscription required"),
            ApiError::CoppaConsentRequired => (StatusCode::FORBIDDEN, "COPPA_CONSENT_REQUIRED", "Parental consent required"),
            ApiError::InvalidFileType => (StatusCode::BAD_REQUEST, "INVALID_FILE_TYPE", "File type not allowed"),
            ApiError::NotPurchased => (StatusCode::FORBIDDEN, "NOT_PURCHASED", "Content not purchased"),
            ApiError::InvalidCursor => (StatusCode::BAD_REQUEST, "INVALID_CURSOR", "Invalid pagination cursor"),
            ApiError::RateLimited => (StatusCode::TOO_MANY_REQUESTS, "RATE_LIMITED", "Too many requests"),
            ApiError::Internal(e) => {
                tracing::error!("Internal error: {e}");
                (StatusCode::INTERNAL_SERVER_ERROR, "INTERNAL", "An internal error occurred")
            }
        };

        (status, Json(ErrorResponse {
            error: ErrorBody { code: code.into(), message: message.into(), details: None },
        })).into_response()
    }
}
```

### 9.6 Rate Limiting `[S§2.3]`

```rust
/// Rate limit configuration per endpoint category
pub struct RateLimitConfig {
    pub default: (u32, Duration),          // 100 req/min
    pub auth: (u32, Duration),             // 10 req/min (login attempts)
    pub upload: (u32, Duration),           // 20 req/min
    pub search: (u32, Duration),           // 60 req/min
    pub messaging: (u32, Duration),        // 30 req/min
}
```

---

## 10. Frontend Architecture

### 10.1 React SPA Structure

The frontend mirrors the backend's domain structure for clear ownership and navigation:

```
frontend/
├── src/
│   ├── main.tsx
│   ├── App.tsx                    # Root layout + router
│   ├── api/
│   │   ├── client.ts              # Fetch wrapper with auth cookie
│   │   └── generated/             # Auto-generated from OpenAPI [§9.4]
│   ├── components/
│   │   ├── ui/                    # Shared UI primitives (Button, Input, Modal, etc.)
│   │   ├── layout/                # Shell, Sidebar, Header
│   │   └── common/                # Shared domain components (UserAvatar, MethodologyBadge)
│   ├── features/
│   │   ├── auth/                  # Login, registration (Kratos UI)
│   │   ├── onboarding/            # Wizard, methodology selection [S§6]
│   │   │   ├── MethodologyWizard.tsx
│   │   │   ├── FamilySetup.tsx
│   │   │   └── GettingStartedRoadmap.tsx
│   │   ├── social/                # Feed, posts, friends, messaging [S§7]
│   │   │   ├── Feed.tsx
│   │   │   ├── PostComposer.tsx
│   │   │   ├── FriendsList.tsx
│   │   │   ├── DirectMessages.tsx
│   │   │   ├── Groups.tsx
│   │   │   └── Events.tsx
│   │   ├── learning/              # Tools, activities, journals [S§8]
│   │   │   ├── Dashboard.tsx
│   │   │   ├── ActivityLog.tsx
│   │   │   ├── JournalEditor.tsx
│   │   │   ├── ReadingList.tsx
│   │   │   └── ProgressView.tsx
│   │   ├── marketplace/           # Browse, listings, cart, purchases [S§9]
│   │   │   ├── Browse.tsx
│   │   │   ├── ListingDetail.tsx
│   │   │   ├── Cart.tsx
│   │   │   ├── ReviewForm.tsx
│   │   │   └── CreatorDashboard.tsx
│   │   ├── settings/              # Family, notifications, billing
│   │   │   ├── FamilySettings.tsx
│   │   │   ├── NotificationPrefs.tsx
│   │   │   └── Subscription.tsx
│   │   └── search/                # Global search [S§14]
│   │       └── SearchResults.tsx
│   ├── hooks/
│   │   ├── useAuth.ts             # AuthContext consumer
│   │   ├── useFamily.ts           # Family data + methodology context
│   │   ├── useWebSocket.ts        # Real-time connection
│   │   └── useMethodologyTools.ts # Active tools for current family
│   ├── lib/
│   │   ├── queryClient.ts         # TanStack Query configuration
│   │   └── websocket.ts           # WebSocket connection manager
│   ├── routes.tsx                 # React Router route definitions
│   └── types/
│       └── index.ts               # Shared frontend types (supplement generated types)
├── public/
├── index.html
├── tailwind.config.ts
├── vite.config.ts
├── tsconfig.json
└── package.json
```

### 10.2 State Management

**Server state** is managed entirely by TanStack Query (React Query). No Redux, no Zustand — server cache is the source of truth.

```typescript
// Example: learning activities with TanStack Query
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ActivitiesApi, CreateActivityRequest } from '../api/generated';

export function useActivities(studentId: string, dateRange?: DateRange) {
  return useQuery({
    queryKey: ['activities', studentId, dateRange],
    queryFn: () => ActivitiesApi.listActivities({
      studentId,
      startDate: dateRange?.start,
      endDate: dateRange?.end,
    }),
  });
}

export function useCreateActivity() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateActivityRequest) =>
      ActivitiesApi.createActivity(data),
    onSuccess: (_, variables) => {
      // Invalidate activity list to refetch
      queryClient.invalidateQueries({
        queryKey: ['activities', variables.studentId],
      });
      // Also invalidate progress (new activity affects progress)
      queryClient.invalidateQueries({
        queryKey: ['progress', variables.studentId],
      });
    },
  });
}
```

**Client state** (UI state that doesn't come from the server) is handled by React's built-in `useState` and `useContext`:

```typescript
// Methodology context — drives tool visibility and UI terminology
import { createContext, useContext } from 'react';

interface MethodologyContextValue {
  primary: Methodology;
  secondary: Methodology[];
  activeTools: ActiveTool[];
  terminology: Record<string, string>; // e.g., { "activity": "lesson" } for Charlotte Mason
}

const MethodologyContext = createContext<MethodologyContextValue | null>(null);

export function useMethodology() {
  const ctx = useContext(MethodologyContext);
  if (!ctx) throw new Error('useMethodology must be used within MethodologyProvider');
  return ctx;
}

// Usage in components — tools render based on methodology
function LearningDashboard() {
  const { activeTools, terminology } = useMethodology();

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {activeTools.map((tool) => (
        <ToolCard
          key={tool.toolId}
          tool={tool}
          label={terminology[tool.slug] ?? tool.displayName}
        />
      ))}
    </div>
  );
}
```

### 10.3 Routing

```typescript
// routes.tsx — React Router v7
import { createBrowserRouter } from 'react-router-dom';

export const router = createBrowserRouter([
  {
    path: '/',
    element: <AppShell />,
    children: [
      // Social [S§7]
      { index: true, element: <Feed /> },
      { path: 'friends', element: <FriendsList /> },
      { path: 'messages', element: <DirectMessages /> },
      { path: 'messages/:conversationId', element: <Conversation /> },
      { path: 'groups', element: <GroupsList /> },
      { path: 'groups/:groupId', element: <GroupDetail /> },
      { path: 'events', element: <EventsList /> },

      // Learning [S§8]
      { path: 'learning', element: <LearningDashboard /> },
      { path: 'learning/activities', element: <ActivityLog /> },
      { path: 'learning/journals', element: <JournalList /> },
      { path: 'learning/journals/new', element: <JournalEditor /> },
      { path: 'learning/reading-lists', element: <ReadingLists /> },
      { path: 'learning/progress/:studentId', element: <ProgressView /> },

      // Marketplace [S§9]
      { path: 'marketplace', element: <MarketplaceBrowse /> },
      { path: 'marketplace/listings/:id', element: <ListingDetail /> },
      { path: 'marketplace/cart', element: <Cart /> },
      { path: 'marketplace/purchases', element: <PurchaseHistory /> },

      // Creator [S§9.1]
      { path: 'creator', element: <CreatorDashboard /> },
      { path: 'creator/listings/new', element: <CreateListing /> },
      { path: 'creator/listings/:id/edit', element: <EditListing /> },

      // Settings
      { path: 'settings', element: <FamilySettings /> },
      { path: 'settings/notifications', element: <NotificationPrefs /> },
      { path: 'settings/subscription', element: <SubscriptionManager /> },

      // Search [S§14]
      { path: 'search', element: <SearchResults /> },

      // Profiles
      { path: 'family/:familyId', element: <FamilyProfile /> },
    ],
  },
  // Auth (Kratos UI)
  { path: '/auth/login', element: <Login /> },
  { path: '/auth/register', element: <Register /> },
  { path: '/auth/recovery', element: <AccountRecovery /> },
  { path: '/auth/verification', element: <EmailVerification /> },

  // Onboarding [S§6]
  { path: '/onboarding', element: <OnboardingWizard /> },
]);
```

### 10.4 WebSocket Integration `[S§7.5, S§13]`

```typescript
// Real-time connection for messaging + notifications
function useWebSocket() {
  const [socket, setSocket] = useState<WebSocket | null>(null);
  const queryClient = useQueryClient();

  useEffect(() => {
    const ws = new WebSocket(`wss://api.homegrown.academy/ws`);

    ws.onmessage = (event) => {
      const msg = JSON.parse(event.data);

      switch (msg.type) {
        case 'new_message':
          // Invalidate conversation to show new message
          queryClient.invalidateQueries({
            queryKey: ['messages', msg.conversationId],
          });
          break;
        case 'notification':
          // Add to notification cache
          queryClient.invalidateQueries({ queryKey: ['notifications'] });
          break;
        case 'friend_request':
          queryClient.invalidateQueries({ queryKey: ['friend-requests'] });
          break;
      }
    };

    setSocket(ws);
    return () => ws.close();
  }, [queryClient]);

  return socket;
}
```

### 10.5 Accessibility `[S§17.6]`

WCAG 2.1 Level AA compliance is enforced through:

- **Semantic HTML** — All interactive elements use proper ARIA roles and labels.
- **Keyboard navigation** — Every interactive element is reachable via Tab, operable via Enter/Space.
- **Focus management** — Route transitions and modals manage focus programmatically.
- **Color contrast** — Tailwind config enforces minimum 4.5:1 contrast ratio for text.
- **Screen reader support** — All images have alt text, dynamic content uses `aria-live` regions.
- **Testing** — `@axe-core/react` integration for automated accessibility audits in development.

### 10.6 Internationalization Readiness `[S§17.7]`

All user-facing strings are externalized from day one, even though Phase 1 is US-only:

```typescript
// i18n setup using react-intl (or similar)
// All strings are keys, not hardcoded English
<FormattedMessage id="learning.activity.create" defaultMessage="Log Activity" />
<FormattedMessage id="social.friends.request" defaultMessage="Send Friend Request" />
```

Date, time, and number formatting use `Intl` APIs with locale-aware formatting.

---

## 11. Background Processing

### 11.1 Job Queue Architecture `[S§13, S§12]`

Redis-backed job queue using `sidekiq-rs` with three priority tiers:

```rust
/// Job priority tiers
pub enum JobQueue {
    /// Safety-critical: CSAM reports, account suspensions, security alerts
    /// Target: process within 30 seconds
    Critical,
    /// Standard: email delivery, notification dispatch, search indexing
    /// Target: process within 5 minutes
    Default,
    /// Bulk/deferrable: digest compilation, analytics aggregation, re-scans
    /// Target: process within 1 hour
    Low,
}
```

### 11.2 Key Jobs by Domain

| Domain | Job | Queue | Description |
|--------|-----|-------|-------------|
| **Trust & Safety** | `CsamReportJob` | Critical | Report CSAM to NCMEC, suspend account `[S§12.1]` |
| **Trust & Safety** | `ContentModerationJob` | Critical | Process flagged content `[S§12.2]` |
| **Notifications** | `SendEmailJob` | Default | Deliver transactional email via Postmark `[S§13.2]` |
| **Notifications** | `PushNotificationJob` | Default | Deliver in-app notification `[S§13.1]` |
| **Social** | `FanOutPostJob` | Default | Fan-out new post to friends' feeds `[S§7.2]` |
| **Media** | `ProcessImageJob` | Default | Resize images, generate thumbnails `[§7.2]` |
| **Media** | `CsamScanJob` | Default | Scan uploaded media via Thorn Safer `[S§12.1]` |
| **Search** | `IndexContentJob` | Default | Update search indexes on content change `[S§14]` |
| **Marketplace** | `ProcessPayoutJob` | Default | Calculate and initiate creator payouts `[S§9.6]` |
| **Notifications** | `CompileDigestJob` | Low | Build daily/weekly email digests `[S§13.3]` |
| **Trust & Safety** | `PeriodicCsamRescanJob` | Low | Re-scan media against updated hash databases `[S§12.1]` |
| **Learning** | `ProgressAggregationJob` | Low | Aggregate progress metrics per student `[S§8.1.7]` |
| **Billing** | `SubscriptionRenewalCheckJob` | Low | Check upcoming renewals, send reminders `[S§15.3]` |

### 11.3 Recurring Schedule

```rust
/// Recurring jobs (cron-style)
pub fn register_recurring_jobs(scheduler: &mut Scheduler) {
    // Daily at 6:00 AM UTC — compile and send daily digests [S§13.3]
    scheduler.add("0 6 * * *", CompileDigestJob { digest_type: DigestType::Daily });

    // Weekly on Mondays at 6:00 AM UTC — weekly digests [S§13.3]
    scheduler.add("0 6 * * 1", CompileDigestJob { digest_type: DigestType::Weekly });

    // Daily at 3:00 AM UTC — CSAM hash database re-scan [S§12.1]
    scheduler.add("0 3 * * *", PeriodicCsamRescanJob);

    // Hourly — aggregate progress metrics [S§8.1.7]
    scheduler.add("0 * * * *", ProgressAggregationJob);

    // Daily at 2:00 AM UTC — check subscription renewals [S§15.3]
    scheduler.add("0 2 * * *", SubscriptionRenewalCheckJob);
}
```

### 11.4 Social Feed Fan-Out `[S§7.2]`

The feed uses a **fan-out-on-write** pattern via Redis sorted sets:

```rust
/// When a user creates a post, fan it out to all friends' feeds
pub async fn fan_out_post(
    redis: &RedisPool,
    db: &DatabaseConnection,
    post: &Post,
) -> Result<(), JobError> {
    // Get all accepted friends of the post author [S§7.4]
    let friend_family_ids = social::repository::get_friend_ids(db, post.family_id).await?;

    // Add post to each friend's feed (Redis sorted set, scored by timestamp)
    let score = post.created_at.timestamp_millis() as f64;
    let member = post.id.to_string();

    for friend_id in friend_family_ids {
        let feed_key = format!("feed:{}", friend_id);
        redis.zadd(&feed_key, &member, score).await?;

        // Trim feed to last 1000 items to bound memory
        redis.zremrangebyrank(&feed_key, 0, -1001).await?;
    }

    // If post is in a group, also add to group feed [S§7.6]
    if let Some(group_id) = post.group_id {
        let group_feed_key = format!("feed:group:{}", group_id);
        redis.zadd(&group_feed_key, &member, score).await?;
        redis.zremrangebyrank(&group_feed_key, 0, -1001).await?;
    }

    Ok(())
}

/// Read a user's feed [S§7.2.3]
pub async fn get_feed(
    redis: &RedisPool,
    db: &DatabaseConnection,
    family_id: Uuid,
    cursor: Option<f64>,
    limit: usize,
) -> Result<Vec<Post>, ApiError> {
    let feed_key = format!("feed:{}", family_id);

    // Get post IDs from Redis sorted set (reverse chronological) [S§7.2.3]
    let max_score = cursor.unwrap_or(f64::MAX);
    let post_ids: Vec<String> = redis
        .zrevrangebyscore_limit(&feed_key, max_score, 0.0, limit)
        .await?;

    // Hydrate post data from PostgreSQL
    let post_uuids: Vec<Uuid> = post_ids.iter()
        .filter_map(|id| Uuid::parse_str(id).ok())
        .collect();

    social::repository::find_posts_by_ids(db, &post_uuids).await
}
```

---

## 12. Deployment & Infrastructure

### 12.1 Docker Multi-Stage Build

```dockerfile
# Stage 1: Build Rust binary
FROM rust:1.82 AS builder
WORKDIR /app

# Cache dependencies
COPY Cargo.toml Cargo.lock ./
RUN mkdir src && echo "fn main(){}" > src/main.rs
RUN cargo build --release && rm -rf src target/release/homegrown-academy*

# Build application
COPY src/ src/
COPY migrations/ migrations/
RUN cargo build --release

# Stage 2: Minimal runtime image
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/target/release/homegrown-academy /usr/local/bin/
COPY --from=builder /app/migrations/ /app/migrations/

EXPOSE 3000
CMD ["homegrown-academy"]
```

### 12.2 Server Configuration (Phase 1-2) `[S§17.5]`

```
Hetzner AX52 (~$60/mo)
├── Docker Engine
│   ├── homegrown-api       (Rust binary, port 3000)
│   ├── ory-kratos          (sidecar, port 4433/4434)
│   ├── postgresql-16       (port 5432, data on NVMe)
│   ├── redis-7             (port 6379)
│   ├── pgbouncer           (port 6432, connection pooling)
│   └── caddy               (reverse proxy, TLS termination, port 443)
├── Data volumes
│   ├── /data/postgresql     (NVMe, 500GB)
│   ├── /data/redis          (NVMe, 10GB)
│   └── /data/backups        (NVMe, 200GB)
└── Monitoring
    ├── Sentry SDK (in-process)
    └── UptimeRobot (external)
```

### 12.3 Caddy Reverse Proxy

```
# Caddyfile
api.homegrown.academy {
    reverse_proxy localhost:3000

    # WebSocket upgrade for /ws path
    @websocket {
        header Connection *Upgrade*
        header Upgrade websocket
    }
    reverse_proxy @websocket localhost:3000

    # Security headers [S§17.1]
    header {
        Strict-Transport-Security "max-age=31536000; includeSubDomains"
        X-Content-Type-Options "nosniff"
        X-Frame-Options "DENY"
        Content-Security-Policy "default-src 'self'; img-src 'self' https://*.r2.cloudflarestorage.com; connect-src 'self' wss://api.homegrown.academy"
        Referrer-Policy "strict-origin-when-cross-origin"
    }
}

# Kratos public API (proxied through main API, not exposed directly)
```

### 12.4 Scaling Path

```
Phase 1 (MVP)                    Phase 2-3                         Phase 3+
─────────────                    ─────────                         ─────────
Single Hetzner server            Add second server                 Multi-server

┌──────────────┐                ┌──────────────┐                  ┌──────────────┐
│  App + DB +  │                │  Load        │                  │  LB (Caddy)  │
│  Redis +     │       →        │  Balancer    │        →         ├──────────────┤
│  Kratos      │                ├──────────────┤                  │  App 1       │
│              │                │  App Server  │                  │  App 2       │
└──────────────┘                │  + Redis +   │                  │  App N       │
                                │  Kratos      │                  ├──────────────┤
                                ├──────────────┤                  │  PG Primary  │
                                │  DB Server   │                  │  PG Replica  │
                                │  (PostgreSQL)│                  ├──────────────┤
                                └──────────────┘                  │  Redis       │
                                                                  │  Meilisearch │
                                                                  │  Kratos      │
                                                                  └──────────────┘
```

### 12.5 Backup Strategy

```bash
# PostgreSQL backups — daily full + continuous WAL archiving
# Full backup daily at 1:00 AM UTC
pg_dump -Fc homegrown_academy > /data/backups/pg_$(date +%Y%m%d).dump

# Upload to R2 for off-site storage
aws s3 cp /data/backups/pg_$(date +%Y%m%d).dump \
    s3://homegrown-backups/postgresql/pg_$(date +%Y%m%d).dump \
    --endpoint-url https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com

# Retention: 30 daily, 12 monthly, 2 yearly
# R2 lifecycle rules handle cleanup

# Redis: RDB snapshots (default Redis config, persisted to NVMe)
# Redis data is ephemeral (cache + feeds) — loss is tolerable, feeds rebuild from PostgreSQL
```

### 12.6 CI/CD Pipeline (GitHub Actions) `[S§17.1]`

```yaml
# .github/workflows/deploy.yml
name: Build and Deploy

on:
  push:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Rust toolchain
        uses: dtolnay/rust-toolchain@stable
        with:
          components: clippy

      - name: Lint
        run: cargo clippy -- -D warnings

      - name: Test
        run: cargo test

      - name: Security audit
        run: cargo install cargo-audit && cargo audit

      - name: Frontend install
        run: cd frontend && npm ci

      - name: Frontend lint
        run: cd frontend && npm run lint

      - name: Frontend type check
        run: cd frontend && npx tsc --noEmit

      - name: Frontend audit
        run: cd frontend && npm audit --production

  deploy:
    needs: test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build Docker image
        run: docker build -t homegrown-api:${{ github.sha }} .

      - name: Build frontend
        run: cd frontend && npm ci && npm run build

      - name: Deploy API to Hetzner
        run: |
          # Push image to registry, SSH to server, pull and restart
          docker save homegrown-api:${{ github.sha }} | \
            ssh deploy@${{ secrets.SERVER_IP }} 'docker load && \
            docker compose up -d api'

      - name: Deploy frontend to Cloudflare Pages
        run: npx wrangler pages deploy frontend/dist --project-name=homegrown-app

      - name: Run migrations
        run: |
          ssh deploy@${{ secrets.SERVER_IP }} \
            'docker exec homegrown-api sea-orm-cli migrate up'
```

---

## 13. Security Architecture

### 13.1 Network Security `[S§17.1]`

| Layer | Control |
|-------|---------|
| **TLS** | All traffic encrypted via TLS 1.3 (Caddy auto-manages Let's Encrypt certificates) `[S§17.1]` |
| **Firewall** | UFW: allow 443 (HTTPS), 22 (SSH from admin IPs only). All other ports blocked externally |
| **CSP** | Content Security Policy prevents XSS — restricts script sources, image sources, connection targets |
| **CORS** | Strict origin allowlist: `app.homegrown.academy`, `homegrown.academy` |
| **SSH** | Key-only authentication, no password login. Fail2ban for brute-force protection |

### 13.2 Application Security (OWASP Top 10) `[S§17.1]`

| OWASP Risk | Rust/Axum Mitigation |
|------------|---------------------|
| **A01: Broken Access Control** | Family-scoped queries via `FamilyScoped` trait (§5.2). Every handler receives `AuthContext` with verified `family_id`. Permission extractors enforce role checks at compile time. |
| **A02: Cryptographic Failures** | Passwords handled by Ory Kratos (Argon2id). Sensitive data encrypted at rest via PostgreSQL `pgcrypto`. All transit encrypted via TLS. |
| **A03: Injection** | SeaORM parameterized queries prevent SQL injection. Rust's type system prevents most injection vectors. User input validated via `serde` deserialization with strict types. |
| **A04: Insecure Design** | Privacy-by-architecture (§1.5). COPPA consent state machine (§5.3). No public user content by design. |
| **A05: Security Misconfiguration** | Docker containers run as non-root. Minimal base images (Debian slim). Security headers set in Caddy. |
| **A06: Vulnerable Components** | `cargo audit` + `npm audit` in CI pipeline. Dependabot for automated dependency updates. |
| **A07: Auth Failures** | Ory Kratos handles auth — battle-tested implementation. MFA encouraged. Rate-limited login attempts (10/min). Session timeout (30 days, revocable). |
| **A08: Data Integrity** | All API inputs validated via typed `serde` structs. CSRF protection via SameSite cookies. Webhook signatures verified (Stripe, Kratos). |
| **A09: Logging Failures** | Security events logged via `tracing` crate. Audit logging for admin actions `[S§2.3]`. Logs shipped to Sentry. |
| **A10: SSRF** | No user-controlled URL fetching in backend. R2 presigned URLs are generated server-side with controlled parameters. |

### 13.3 Rust's Memory Safety Advantages

Rust eliminates entire categories of vulnerabilities that plague C/C++ and even garbage-collected languages:

- **No buffer overflows** — Bounds-checked array access.
- **No use-after-free** — Ownership system prevents dangling references.
- **No data races** — The borrow checker prevents concurrent mutable access at compile time.
- **No null pointer dereferences** — `Option<T>` replaces null; must be explicitly handled.
- **No uninitialized memory** — All variables must be initialized before use.

These guarantees are especially valuable for a platform handling children's data `[S§17.2]` — memory safety vulnerabilities are among the most exploited attack vectors, and Rust makes them structurally impossible.

### 13.4 Data Encryption `[S§17.1]`

| Data | At Rest | In Transit |
|------|---------|------------|
| **Credentials** | Argon2id (Ory Kratos) | TLS 1.3 |
| **Payment data** | Never stored (Stripe handles) | TLS 1.3 to Stripe |
| **PII** | PostgreSQL `pgcrypto` for sensitive fields | TLS 1.3 |
| **Session tokens** | Kratos encrypted sessions | HTTPS-only SameSite cookies |
| **Media files** | R2 server-side encryption | TLS 1.3 to/from R2 |
| **Backups** | Encrypted R2 storage | TLS 1.3 during transfer |

### 13.5 COPPA Security Controls `[S§17.2]`

| Control | Implementation |
|---------|---------------|
| **Data minimization** | Student profiles collect only name, birth year, grade. No email, no credentials, no social profile. `[S§3.1.3]` |
| **Parental consent** | COPPA consent state machine (§5.3) required before creating student profiles |
| **Data access** | Parents have complete visibility into all student data `[S§3.3]` |
| **Data deletion** | Parents can request deletion of child data; processed within COPPA's required timeframe `[S§16.3]` |
| **Third-party sharing** | Student data never shared with third parties except CSAM reporting (legal requirement) |
| **No direct contact** | Platform prohibits unmediated adult-student communication `[S§3.3]` |

### 13.6 Dependency Security

```bash
# Automated via CI/CD (§12.6)
cargo audit                    # Rust dependency vulnerabilities
npm audit --production         # Node.js dependency vulnerabilities

# GitHub Dependabot configuration
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: cargo
    directory: "/"
    schedule:
      interval: weekly
  - package-ecosystem: npm
    directory: "/frontend"
    schedule:
      interval: weekly
```

---

## 14. Phase 1 (MVP) Scope `[S§19]`

This section maps each Phase 1 domain from `[S§19]` to concrete implementation artifacts.

### 14.1 Identity & Access `[S§3]`

**Scope**: Family accounts, parent users, student profiles, email + OAuth authentication.

| Component | Implementation |
|-----------|---------------|
| **Auth** | Ory Kratos: email/password + Google/Facebook/Apple OIDC `[S§6.1]` |
| **Family accounts** | `iam_families` table, atomic creation with first parent `[S§3.1.1]` |
| **Parent users** | `iam_parents` table, linked to Kratos identity `[S§3.1.2]` |
| **Student profiles** | `iam_students` table, parent-mediated `[S§3.1.3]` |
| **COPPA consent** | Consent state machine in `iam_families.coppa_consent_status` `[S§17.2]` |

**API Endpoints**: 8 endpoints (family CRUD, student CRUD, profile management).

**React Components**: `Login`, `Register`, `FamilySettings`, `StudentManager`.

### 14.2 Methodology `[S§4]`

**Scope**: 6 methodologies, tool registry, philosophy modules.

| Component | Implementation |
|-----------|---------------|
| **Definitions** | 6 rows in `method_definitions` (Charlotte Mason, Traditional, Classical, Waldorf, Montessori, Unschooling) `[S§4.5]` |
| **Tool registry** | `method_tools` + `method_tool_activations` tables `[S§4.2]` |
| **Philosophy** | JSONB content in `method_definitions.philosophy` `[S§4.1]` |
| **Tool resolver** | `resolve_family_tools()` + `resolve_student_tools()` functions `[S§4.2, S§4.6]` |

**API Endpoints**: 3 endpoints (list methodologies, get methodology, get family tools).

**React Components**: `MethodologyBadge`, `ToolCard`.

### 14.3 Discovery `[S§5]`

**Scope**: Methodology quiz, methodology explorer pages, state legal guides (all 51), Homeschooling 101, advocacy content.

| Component | Implementation |
|-----------|---------------|
| **Quiz engine** | `disc_quiz_definitions` + `disc_quiz_results` tables `[S§5.1]` |
| **Quiz API** | Rust endpoints serving quiz data to Astro site `[S§5.1]` |
| **Explorer pages** | Astro static pages generated from methodology data `[S§5.2]` |
| **State guides** | 51 structured documents in `disc_state_guides` table `[S§5.3]` |
| **101 content** | Astro static pages `[S§5.4]` |
| **SEO** | Structured data markup, server-rendered HTML via Astro `[S§5.5]` |

**API Endpoints**: 5 endpoints (quiz CRUD, state guides, methodology explorer data).

**Astro Pages**: Methodology explorer (6 pages), state guides (51 pages), 101 content (~10 pages), quiz page.

### 14.4 Onboarding `[S§6]`

**Scope**: Full onboarding flow — account creation, family setup, methodology wizard, getting-started roadmaps, starter recommendations, community connections.

| Component | Implementation |
|-----------|---------------|
| **Account creation** | Kratos registration flow + post-registration webhook `[S§6.1]` |
| **Family setup** | Multi-step wizard collecting family info `[S§6.2]` |
| **Methodology wizard** | Three-path selector (quiz-informed, explore, skip) `[S§6.3]` |
| **Roadmaps** | Methodology-specific checklists from `method_definitions.onboarding_config` `[S§6.4]` |
| **Starter recs** | Marketplace query filtered by methodology + age `[S§6.5]` |
| **Community** | Methodology-matched group suggestions `[S§6.6]` |

**API Endpoints**: 4 endpoints (wizard state, roadmap, recommendations, community suggestions).

**React Components**: `OnboardingWizard` (multi-step), `MethodologyWizard`, `FamilySetup`, `GettingStartedRoadmap`.

### 14.5 Social `[S§7]`

**Scope**: Profiles, timeline/feed (reverse-chronological), comments (threaded), friend system, direct messaging (friends-only), platform-managed methodology groups, events (basic), location-based discovery (opt-in).

| Component | Implementation |
|-----------|---------------|
| **Profiles** | `soc_profiles` table with per-field privacy `[S§7.1]` |
| **Feed** | Redis sorted sets + fan-out-on-write + PostgreSQL posts `[S§7.2]` |
| **Comments** | `soc_comments` table with one-level threading `[S§7.3]` |
| **Friends** | `soc_friendships` table with request/accept/block flow `[S§7.4]` |
| **Messaging** | `soc_messages` table + WebSocket real-time delivery `[S§7.5]` |
| **Groups** | 6 platform-managed methodology groups (user-created groups are Phase 2) `[S§7.6]` |
| **Events** | `soc_events` table with RSVP (basic, no recurring — full events in Phase 2) `[S§7.7]` |
| **Location** | PostGIS-based discovery, opt-in, coarse-grained `[S§7.8]` |

**API Endpoints**: ~20 endpoints (feed, posts, comments, friends, messages, groups, events, discovery).

**React Components**: `Feed`, `PostComposer`, `FriendsList`, `DirectMessages`, `GroupDetail`, `EventsList`, `NearbyFamilies`.

### 14.6 Learning `[S§8]`

**Scope**: Activities, Reading Lists, Journaling & Narration, basic Progress Tracking.

| Component | Implementation |
|-----------|---------------|
| **Activities** | `learn_activities` table `[S§8.1.1]` |
| **Reading Lists** | `learn_reading_lists` + `learn_reading_list_items` tables `[S§8.1.3]` |
| **Journals** | `learn_journals` table (freeform, narration, reflection types) `[S§8.1.4]` |
| **Progress** | Basic tracking: activity counts, reading completion, hours per subject `[S§8.1.7]` |

**Not in Phase 1**: Tests & Grades, Projects, methodology-specific tools (Nature Journals, Trivium Tracker, etc.), advanced analytics.

**API Endpoints**: ~12 endpoints (CRUD for each tool + progress summary).

**React Components**: `LearningDashboard`, `ActivityLog`, `JournalEditor`, `ReadingList`, `ProgressView`.

### 14.7 Marketplace `[S§9]`

**Scope**: Creator onboarding, content listings, browse/search/filter, purchase flow, ratings & reviews (verified-purchaser), basic creator dashboard.

| Component | Implementation |
|-----------|---------------|
| **Creator onboarding** | `mkt_creators` table + Stripe Connect Standard onboarding `[S§9.1]` |
| **Listings** | `mkt_listings` table with full lifecycle `[S§9.2]` |
| **Search** | PostgreSQL FTS with faceted filtering `[S§9.3]` |
| **Purchases** | Stripe checkout → `mkt_purchases` table `[S§9.4]` |
| **Reviews** | `mkt_reviews` table, verified-purchaser only `[S§9.5]` |
| **Creator dashboard** | Basic: sales history, earnings, listing management `[S§9.6]` |

**Not in Phase 1**: Creator payouts (1099-K processing), advanced creator analytics.

**API Endpoints**: ~15 endpoints (listings CRUD, search, purchase, reviews, creator dashboard).

**React Components**: `MarketplaceBrowse`, `ListingDetail`, `Cart`, `Checkout`, `ReviewForm`, `CreatorDashboard`.

### 14.8 Trust & Safety `[S§12]`

**Scope**: CSAM detection, automated content screening, user reporting, bot prevention, community guidelines, basic moderation dashboard.

| Component | Implementation |
|-----------|---------------|
| **CSAM** | Thorn Safer integration for all image uploads `[S§12.1]` |
| **Screening** | AWS Rekognition for content moderation `[S§12.2]` |
| **Reporting** | `safety_reports` table with categorized reports `[S§12.3]` |
| **Bot prevention** | CAPTCHA on registration + rate limiting `[S§12.4]` |
| **Moderation** | Basic admin dashboard for content review `[S§12.7]` |

**API Endpoints**: ~6 endpoints (report content, moderation queue, mod actions).

### 14.9 Billing `[S§15]`

**Scope**: Free tier, marketplace transactions, payment processing, sales tax.

| Component | Implementation |
|-----------|---------------|
| **Free tier** | Default `subscription_tier = 'free'` on all families `[S§15.1]` |
| **Marketplace payments** | Stripe Checkout for marketplace purchases `[S§15.4]` |
| **Sales tax** | Stripe Tax for automatic calculation and collection `[S§15.4]` |

**Not in Phase 1**: Premium subscriptions, upgrade/downgrade flows.

**API Endpoints**: ~4 endpoints (create checkout session, webhook handler, purchase history).

### 14.10 Notifications `[S§13]`

**Scope**: In-app notification center, transactional email.

| Component | Implementation |
|-----------|---------------|
| **In-app** | `notify_notifications` table + WebSocket push `[S§13.1]` |
| **Email** | Postmark transactional: verification, password reset, purchase receipts, social notifications `[S§13.2]` |
| **Preferences** | `notify_preferences` table with per-type, per-channel settings `[S§13.3]` |

**Not in Phase 1**: Digest emails, push notifications.

**API Endpoints**: ~4 endpoints (list, mark read, preferences CRUD).

### 14.11 Search `[S§14]`

**Scope**: Full-text search (social + marketplace), basic autocomplete.

| Component | Implementation |
|-----------|---------------|
| **Social search** | PostgreSQL FTS on families, groups, events `[S§14.1]` |
| **Marketplace search** | PostgreSQL FTS with faceted filtering on listings `[S§14.1]` |
| **Learning search** | PostgreSQL FTS, family-scoped `[S§14.1]` |
| **Autocomplete** | `pg_trgm` trigram index on listing titles `[S§14.2]` |

**API Endpoints**: 2 endpoints (search, autocomplete).

### 14.12 Content & Media

**Scope**: Image/file upload, storage, delivery.

| Component | Implementation |
|-----------|---------------|
| **Upload** | Presigned URL upload to R2 `[§7.1]` |
| **Processing** | Background image resize, CSAM scan `[§7.2]` |
| **Delivery** | Cloudflare CDN via R2 `[§7.3]` |

**API Endpoints**: 3 endpoints (request upload, confirm upload, download).

### 14.13 Privacy/Compliance `[S§17.2]`

**Scope**: COPPA compliance, privacy policy, terms of service, data export.

| Component | Implementation |
|-----------|---------------|
| **COPPA** | Consent state machine, student data minimization `[S§17.2]` |
| **Data export** | JSON/CSV export of all family data `[S§8.5]` |
| **Privacy policy** | Static page on Astro public site |
| **Terms of service** | Static page on Astro public site |

**API Endpoints**: 2 endpoints (request data export, download export).

---

## 15. Architecture Decision Records

### ADR-001: Rust over TypeScript for Backend

**Status**: Accepted

**Context**: The backend must serve a 14-domain application with sub-300ms API responses `[S§17.3]`, scale to 100K concurrent users `[S§17.3]`, handle sensitive children's data `[S§17.2]`, and be primarily written by AI.

**Decision**: Use Rust (Axum + Tokio) instead of TypeScript (Node.js).

**Consequences**:
- **Positive**: Compile-time safety catches bugs before runtime. 3-5x better throughput reduces infrastructure costs. Memory safety eliminates entire vulnerability classes. Single binary deployment. "Never rewrite" — Rust codebases don't hit performance walls.
- **Negative**: Longer compile times (~2-5 min for full build). Smaller ecosystem of web libraries compared to Node.js. Fewer Rust developers available for future hiring.
- **Mitigation**: AI generates code (negating learning curve). Incremental compilation + `cargo check` for development. Most needed libraries exist (Axum, SeaORM, Serde, Tokio are mature).

**Revision trigger**: Never. This is a foundational decision.

### ADR-002: Monolith Architecture

**Status**: Accepted

**Context**: 14 spec domains `[S§2.1]` could be individual microservices. However, a solo developer maintains the system, all domains share the same database, and inter-domain communication is frequent `[S§18]`.

**Decision**: Single Rust binary with domain modules. No microservices.

**Consequences**:
- **Positive**: Single deployment unit. No inter-service networking complexity. Shared database with referential integrity. Simpler debugging and tracing. Lower infrastructure cost.
- **Negative**: All domains scale together (cannot independently scale Search vs. Social). Single point of failure. Larger binary size.
- **Mitigation**: Rust's efficiency makes "scale together" viable to 100K+ concurrent users on modest hardware. Health checks and graceful degradation `[S§17.5]` mitigate single-point-of-failure risk.

**Revision trigger**: Extract a domain as a separate service when it accounts for >40% of system resources or has fundamentally different deployment cadence needs.

### ADR-003: PostgreSQL as Infrastructure

**Status**: Accepted

**Context**: The platform needs: relational data (families, friendships, purchases), document storage (methodology config), location queries (nearby families), full-text search, and background job metadata. Using separate services for each would multiply operational complexity.

**Decision**: Use PostgreSQL as the primary datastore for relational data, JSONB documents, PostGIS location queries, and full-text search (Phase 1). Redis supplements for caching, job queues, and feed fan-out.

**Consequences**:
- **Positive**: Single database to operate, back up, and monitor. PostgreSQL's JSONB is performant enough for methodology config. PostGIS eliminates a separate geo service. FTS eliminates a search service for Phase 1. Strong consistency guarantees.
- **Negative**: PostgreSQL FTS lacks typo tolerance and instant search capabilities that Meilisearch provides. JSONB queries are less optimized than purpose-built document stores.
- **Mitigation**: Meilisearch migration path defined (§8.2) with specific trigger (100K listings or 500ms p95 latency). JSONB data volume is small (6 methodologies × ~10KB each).

**Revision trigger**: Add Meilisearch when search latency exceeds 500ms p95 or marketplace exceeds 100K listings.

### ADR-004: Ory Kratos for Authentication

**Status**: Accepted

**Context**: Building authentication from scratch requires implementing password hashing, session management, OIDC flows (Google, Facebook, Apple), MFA, account recovery, and email verification. These are security-critical features where custom implementations frequently have vulnerabilities.

**Decision**: Use Ory Kratos as a self-hosted authentication sidecar.

**Consequences**:
- **Positive**: Battle-tested auth implementation. OIDC support out of the box. MFA (TOTP + WebAuthn) built in. No per-MAU pricing (self-hosted). Custom COPPA consent layer built on Kratos hooks. Session management with remote revocation.
- **Negative**: Additional container to operate. Learning Kratos configuration. Custom UI required (Kratos provides API, not UI).
- **Mitigation**: Kratos is lightweight (<50MB memory). React handles UI. Kratos configuration is well-documented.

**Revision trigger**: None. Open-source, self-hosted, no vendor lock-in.

### ADR-005: React SPA over SSR

**Status**: Accepted

**Context**: The application has 14 domains with extensive interactivity — messaging, social feeds, learning tools, marketplace, quizzes. SEO is handled separately by Astro for public content `[S§5]`. The authenticated app does not need SEO.

**Decision**: React SPA (Vite) for the authenticated application. Astro SSG for public content.

**Consequences**:
- **Positive**: Simpler architecture — no SSR hydration complexity. Real-time features (messaging, notifications) are natural in SPA. React's ecosystem provides rich components for every domain need. Clear separation: Astro = public/SEO, React = app/interactive.
- **Negative**: Initial page load requires JavaScript bundle download. No SSR means no server-side data fetching.
- **Mitigation**: Vite code splitting keeps initial bundle small. TanStack Query prefetching can begin data loading immediately. The app is behind auth — users expect a login flow before seeing content.

**Revision trigger**: None anticipated. The two-site architecture (Astro + React) cleanly separates concerns.

### ADR-006: Hetzner over Cloud Providers

**Status**: Accepted

**Context**: Comparable AWS EC2 instances cost $500-2,000/mo. Rust's efficiency means a single dedicated server handles Phase 1-2 workloads comfortably.

**Decision**: Hetzner dedicated server (~$60/mo) for Phase 1-2, with documented scaling path.

**Consequences**:
- **Positive**: ~90% cost savings vs. AWS/GCP for equivalent hardware. Predictable pricing. No vendor lock-in (standard Linux + Docker). Sufficient for 10K-50K concurrent users with Rust.
- **Negative**: No managed services (must operate PostgreSQL, Redis yourself). No auto-scaling. Single datacenter (no multi-region).
- **Mitigation**: Automated backups (§12.5). Monitoring (§2.14). Scaling path documented (§12.4). Multi-region is a Phase 4 concern.

**Revision trigger**: Move to managed services (AWS RDS, ElastiCache) when operational complexity of self-managing databases exceeds a solo developer's capacity, or when multi-region is required.

### ADR-007: Stripe Connect for Marketplace Payments

**Status**: Accepted

**Context**: The marketplace requires creator identity verification, payouts, sales tax handling, and 1099-K filing `[S§9.6, S§15.4]`. Building payment infrastructure from scratch is prohibitively complex for a solo developer.

**Decision**: Use Stripe Connect (Standard accounts) for marketplace payments and Stripe Billing for subscriptions.

**Consequences**:
- **Positive**: Stripe handles creator KYC/identity verification. 1099-K filing offloaded to Stripe. Sales tax automated via Stripe Tax. Stripe Connect Standard means creators manage their own Stripe accounts — reduced platform liability. Subscription management with upgrade/downgrade built in.
- **Negative**: Stripe fees (~2.9% + 30¢ per transaction + Connect fees). Platform doesn't control the payout experience directly.
- **Mitigation**: Stripe fees are industry standard and competitive. The alternative (custom payment infrastructure) is not viable for a solo developer.

**Revision trigger**: None. Stripe is the industry standard for this exact use case.

### ADR-008: Family-Scoped Data Isolation

**Status**: Accepted

**Context**: The platform handles children's data subject to COPPA `[S§17.2]`. Cross-family data leaks would be both a privacy violation and a regulatory violation. The spec mandates family data ownership `[S§16.2]`.

**Decision**: Enforce family-scoped data isolation at the architecture level through a Rust trait that requires `family_id` on all data-access queries, plus PostgreSQL Row-Level Security as defense-in-depth.

**Consequences**:
- **Positive**: Cross-family data access is structurally impossible without explicit opt-in. The Rust compiler catches missing family_id filters. RLS provides database-level enforcement even if application logic has a bug. COPPA compliance is enforced by architecture, not by developer discipline.
- **Negative**: Every query includes a `family_id` filter, even when not logically necessary (e.g., platform-wide analytics). Social features (friends, groups) require explicit cross-family data paths.
- **Mitigation**: Social cross-family queries use separate, explicitly-defined repository functions that document why cross-family access is needed. Analytics queries use separate read-only database roles that bypass RLS.

```rust
/// Trait enforcing family-scoped database access
pub trait FamilyScoped {
    /// Returns all records belonging to a family
    fn find_by_family(db: &DatabaseConnection, family_id: Uuid)
        -> impl Future<Output = Result<Vec<Self>, DbErr>> + Send
    where Self: Sized;

    /// Returns a single record if it belongs to the specified family
    fn find_by_id_and_family(db: &DatabaseConnection, id: Uuid, family_id: Uuid)
        -> impl Future<Output = Result<Option<Self>, DbErr>> + Send
    where Self: Sized;
}
```

**Revision trigger**: Never. This is a core privacy principle.

### ADR-009: Relational Social Graph (PostgreSQL + Apache AGE)

**Status**: Accepted

**Context**: The social layer `[S§7]` requires friendships, group memberships, event RSVPs, likes, follows, and location-based discovery. Phase 3+ will need friend-of-friend recommendations and social graph traversal for community suggestions `[S§10]`.

**Decision**: Store social relationships as indexed relational join tables in PostgreSQL for Phase 1-2. Use Apache AGE (PostgreSQL graph extension) for Phase 3+ graph queries (recommendations, friend-of-friend discovery) rather than adding a separate graph database.

**Consequences**:
- **Positive**: Social relationships are simple join tables with clear indexes — no additional infrastructure for Phase 1-2. Fan-out-on-write feed pattern via Redis sorted sets handles feed generation efficiently. Apache AGE runs inside PostgreSQL — no separate graph database to operate. Graph queries (when needed) use familiar SQL + Cypher syntax.
- **Negative**: Complex graph traversals (e.g., "friends of friends who use Charlotte Mason within 50km") require multiple joins in Phase 1-2. Apache AGE adds a PostgreSQL extension dependency.
- **Mitigation**: Phase 1-2 social features (friends, groups, events) are well-served by indexed join tables. Apache AGE is a PostgreSQL extension, not a separate service — operational complexity stays low.

```sql
-- Phase 1-2: relational queries for social features
-- Get friends of friends (2-hop) for recommendations
SELECT DISTINCT f2.accepter_family_id as suggested_family_id
FROM soc_friendships f1
JOIN soc_friendships f2 ON f1.accepter_family_id = f2.requester_family_id
WHERE f1.requester_family_id = $1  -- current user
  AND f1.status = 'accepted'
  AND f2.status = 'accepted'
  AND f2.accepter_family_id != $1  -- not self
  AND NOT EXISTS (                 -- not already friends
      SELECT 1 FROM soc_friendships existing
      WHERE existing.requester_family_id = $1
        AND existing.accepter_family_id = f2.accepter_family_id
        AND existing.status = 'accepted'
  );

-- Phase 3+: Apache AGE Cypher query for the same (cleaner, faster at scale)
-- SELECT * FROM cypher('social', $$
--     MATCH (me:Family {id: $1})-[:FRIEND]->(f)-[:FRIEND]->(suggested)
--     WHERE NOT (me)-[:FRIEND]->(suggested) AND suggested.id <> $1
--     RETURN DISTINCT suggested.id
-- $$) AS (suggested_id UUID);
```

**Revision trigger**: Install Apache AGE when implementing Phase 3 recommendation features or when friend-of-friend queries exceed 100ms with relational approach.

---

*This architecture document translates the Homegrown Academy specification (`specs/SPEC.md`) into concrete, opinionated technology decisions. Every choice traces back to spec requirements via `[S§n]` references. The document is designed to be practical enough that development can start directly from it — the Rust code examples, SQL schemas, and React patterns are intended as starting templates, not pseudocode.*

*Architecture decisions are not permanent. Each ADR includes a revision trigger — the specific condition under which the decision should be revisited. The goal is to build the simplest system that satisfies Phase 1 requirements while ensuring a clear, documented path to Phase 2-4 scale.*
