# Homegrown Academy — Technical Architecture

## 1. Architecture Principles

Six principles govern every technical decision. They are ordered by priority — when principles conflict, higher-ranked principles win.

### 1.1 AI-First Development

The codebase is primarily generated and maintained by AI (Claude). This inverts traditional technology selection criteria:

- **Learning curve is irrelevant** — AI generates idiomatic Go, TypeScript, SQL, and configuration equally well. The "difficulty" of a language is not a cost.
- **Static analysis is a free safety net** — Go's compiler, `golangci-lint`, and the race detector catch entire categories of bugs before production. For AI-generated code, this is pure upside: the toolchain reviews every line before it runs.
- **Explicitness over magic** — AI generates better code when patterns are explicit and consistent. Convention-over-configuration frameworks (Rails, Laravel) rely on implicit knowledge that AI may misapply. Explicit configuration and typed contracts produce more reliable generated code.
- **Strong type systems reduce review burden** — When the compiler and linter enforce correctness, the human developer (solo) can focus review on business logic and architecture rather than null checks and type mismatches.

### 1.2 Monolith-First

A single deployable unit until proven otherwise. `[S§17.4]`

- **17 spec domains → Go packages** within one binary, not 17 services.
- **One PostgreSQL database** with schema prefixes per domain.
- **One deployment target** — a single Docker container on a single server.
- **Microservice extraction is a scaling decision**, not an architecture decision. Extract only when a domain has demonstrably different scaling needs (e.g., media processing) that cannot be solved by adding capacity.

**Revision trigger**: Extract a domain when its resource consumption exceeds 40% of total system resources, or when its deployment cadence fundamentally conflicts with the rest of the system.

### 1.3 Type-Safety-Everywhere

Types flow from database schema to API response to React component. `[S§17.1]`

- **Go structs** define API request/response shapes via struct tags (`json:"field"`).
- **GORM models** are hand-written in `models.go` — the schema is the source of truth.
- **OpenAPI 3.1** spec is generated from Go types via swaggo/swag annotations.
- **TypeScript client types** are generated from OpenAPI spec.
- Zero `any` in TypeScript. All errors MUST be checked in production Go (`if err != nil`).

### 1.4 Progressive Complexity

Start simple. Add complexity only when measured load demands it. `[S§17.4]`

- PostgreSQL full-text search before Typesense.
- Single server before load balancer.
- In-process background tasks before distributed job queues.
- Reverse-chronological feed before algorithmic ranking.

Every "simple first" choice includes a documented **revision trigger** — the specific metric or threshold that justifies upgrading.

### 1.5 Privacy-by-Architecture

Privacy is enforced structurally, not by policy. `[S§17.2]`

- **Family-scoped queries** — a Go interface enforces that every database query includes a `family_id` filter. Cross-family data access is structurally impossible without explicit opt-in (social sharing).
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

### 2.1 Backend: Go + Echo

**Satisfies**: `[S§17.3]` performance targets, `[S§17.1]` security, `[S§17.4]` scalability

| Attribute | Value |
|-----------|-------|
| **Language** | Go (latest stable, 1.23+) |
| **Web framework** | Echo v4 |
| **Concurrency** | goroutines (built-in) |
| **Expected throughput** | 10,000-30,000 requests/sec with DB (single server) |
| **Memory footprint** | 20-60MB per instance |

**Rationale**: AI generates Go code equally well; Go's compiler, `golangci-lint`, and the built-in race detector catch categories of bugs before production. A solo developer cannot manually review every line — static analysis and the type system serve as the second reviewer. Go's simplicity (one way to do things, minimal abstraction) produces highly readable AI-generated code. Performance means Phase 1-2 runs comfortably on a ~$110/mo AWS setup (§2.11), keeping infrastructure costs minimal. Go's garbage collector eliminates manual memory management, and the race detector catches data races in tests.

**Rejected alternatives**:
- **TypeScript (Node.js)**: 3-5x lower throughput, no compile-time safety beyond types, `null`/`undefined` hazards, need for runtime validation libraries. Reasonable choice, but Go's compile-time guarantees and goroutine concurrency model are more valuable.
- **Rust**: Stronger compile-time guarantees but significantly higher complexity (borrow checker, lifetimes). The additional safety margins do not justify the cognitive overhead for a solo developer who values simplicity and fast iteration.
- **Ruby on Rails**: No type safety, 10-20x slower than Go, magic conventions conflict with AI-first development principle.

**Revision trigger**: Never — Go is the foundational language choice. Individual dependencies may be replaced, but the language stays.

### 2.2 ORM: GORM v2

**Satisfies**: `[S§16]` data architecture, `[S§17.3]` performance

| Attribute | Value |
|-----------|-------|
| **Library** | GORM v2 (latest stable) |
| **Migration tool** | goose |
| **Query style** | Chainable, struct-tag-based mapping |

**Rationale**: Most mature and widely-used Go ORM. Supports PostgreSQL JSONB and array types natively (needed for methodology config `[S§4.1]`). Goose handles schema evolution with plain SQL migrations. GORM models are hand-written Go structs with struct tags, keeping the schema close to the code.

**Rejected alternatives**:
- **sqlc**: SQL-first with generated Go code. Good type safety but lacks the relationship management and query-building ergonomics that a 17-domain app needs.
- **Ent (entgo.io)**: Schema-as-code approach with code generation. More opinionated than needed, and the generated code is harder for AI to reason about.
- **sqlx (jmoiron/sqlx)**: Not an ORM — thin wrapper over `database/sql`. Good for raw SQL but lacks relationship handling and migration tooling.

**Revision trigger**: If GORM's query performance becomes a bottleneck, supplement specific hot-path queries with raw `database/sql`.

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

**Rationale**: The platform has 17 domains with extensive interactivity — messaging, social feeds, learning tool interactions, marketplace browse/purchase, quiz flows. React's component ecosystem (rich text editors, file uploaders, drag-and-drop, data visualization) is unmatched. SPA architecture keeps real-time features (messaging, notifications) simple — no SSR hydration complexity.

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

**Rationale**: Discovery content `[S§5]` must be SEO-indexable and fast. Astro generates static HTML with perfect Lighthouse scores. Cloudflare Pages hosts static sites for free with global CDN. This completely eliminates SSR as a backend concern — the Go API serves JSON only.

**Rejected alternatives**:
- **SSR from Go backend**: Adds template rendering complexity to the API server. Mixes concerns (API + HTML rendering). Go's template ecosystem (`html/template`) is functional but unnecessary.
- **Next.js static export**: Heavier runtime than Astro for content-focused pages. Astro's island architecture is better suited for mostly-static content with optional interactive components.

**Revision trigger**: None. Static site generation is the correct pattern for SEO content.

### 2.5 Database: PostgreSQL 16

**Satisfies**: `[S§16]` data architecture, `[S§14]` search, `[S§7.8]` location, `[S§4.1]` methodology config

| Attribute | Value |
|-----------|-------|
| **Version** | PostgreSQL 16+ |
| **Extensions** | `pg_trgm`, `PostGIS`, `pgcrypto`, `uuid-ossp` |
| **Connection pooling** | RDS manages server-side connections; GORM's built-in pool (via `database/sql`) handles app-side pooling |

**Capabilities used**:
- **JSONB**: Methodology configuration, tool registry, quiz scoring weights `[S§4.1]`
- **PostGIS**: Location-based discovery with coarse-grained geometry `[S§7.8]`
- **Full-text search**: `tsvector` + `pg_trgm` for Phase 1 search `[S§14]`
- **Arrays**: Multi-tag storage (methodology tags, subject tags) `[S§9.2]`
- **Row-level security (RLS)**: Defense-in-depth for family data isolation `[S§16.2]`

**Rejected alternatives**:
- **MySQL**: No JSONB, no PostGIS, weaker full-text search. PostgreSQL is strictly superior for this use case.
- **MongoDB**: Loses relational integrity across 17 interconnected domains. The data model `[S§16.1]` is fundamentally relational.

**Revision trigger**: Never for the primary database. Add read replicas when write throughput exceeds single-server capacity (~Phase 3).

### 2.6 Search: PostgreSQL FTS → Typesense

**Satisfies**: `[S§14]` search, `[S§9.3]` marketplace discovery

**Phase 1**: PostgreSQL `tsvector` + `pg_trgm` indexes per search scope.
- Social search (users, groups, events)
- Marketplace search (listings by title, description, tags)
- Learning search (family-scoped: activities, journals, reading lists)

**Phase 2+**: Typesense for marketplace and social search.
- Typo-tolerant, faceted filtering `[S§9.3]`, instant autocomplete `[S§14.2]`
- PostgreSQL FTS retained for family-scoped learning search (smaller dataset, privacy-sensitive)

**Revision trigger**: Migrate to Typesense when marketplace exceeds ~100K listings or search latency exceeds 500ms p95 `[S§17.3]`.

### 2.7 Background Jobs: Redis + asynq

**Satisfies**: `[S§13]` notifications, `[S§12]` moderation pipeline, `[S§14]` search indexing

| Attribute | Value |
|-----------|-------|
| **Queue backend** | Redis 7+ |
| **Job processor** | hibiken/asynq |
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

**Rationale**: Building auth from scratch in Go means implementing password hashing, session management, OIDC flows, MFA, account recovery, and email verification. Kratos handles all of this as a battle-tested, self-hosted identity service. Custom COPPA consent flow `[S§17.2]` is built on top of Kratos hooks.

**Rejected alternatives**:
- **Auth0 / Firebase Auth**: SaaS dependency, per-MAU pricing becomes expensive at scale. At 100K+ families, Auth0 costs $1,000+/mo vs. self-hosted Kratos at $0.
- **Custom Go auth**: Months of development for a solved problem. Security-critical code that should not be custom-written by a solo developer.

**Revision trigger**: None. Kratos is open-source and self-hosted — no vendor lock-in.

### 2.9 Payments: Hyperswitch + Stripe

**Satisfies**: `[S§15]` billing, `[S§9.6]` marketplace payouts

| Attribute | Value |
|-----------|-------|
| **Orchestration** | Hyperswitch (self-hosted Docker sidecar) |
| **Payment processor** | Stripe (configured as a Hyperswitch connector) |
| **Marketplace payments** | Hyperswitch split payments / sub-merchant accounts `[S§9.6]` |
| **Subscriptions** | Hyperswitch (native subscription engine) — resolved in `10-billing.md §7` `[S§15.3]` |
| **Sales tax** | Stripe Tax (via Hyperswitch connector) `[S§15.4]` |
| **1099-K** | Stripe handles for sub-merchant accounts `[S§9.6]` |

**Rationale**: Hyperswitch provides processor-agnostic payment orchestration. The platform talks to Hyperswitch (self-hosted), which talks to Stripe as the underlying processor. This gives us the compliance benefits of Stripe (creator KYC, identity verification, 1099-K filing, sales tax) while adding the flexibility to swap or add processors without application code changes. Marketplace payments (split payments, creator sub-merchants, payouts) are managed through Hyperswitch's payment orchestration layer. See `specs/domains/07-mkt.md §7` for full marketplace payment details and `§18.5` for the Hyperswitch deployment architecture.

**Note**: COPPA micro-charge verification `[S§1.4]` uses the `billing` adapter, which wraps Hyperswitch for one-time payments. See `10-billing.md §13` for the full COPPA flow.

**Revision trigger**: Swap Stripe connector for another processor if pricing or features change — only Hyperswitch connector configuration changes, no application code impact.

### 2.10 File Storage: Cloudflare R2

**Satisfies**: `[S§14]` content & media, `[S§9.2]` marketplace files

| Attribute | Value |
|-----------|-------|
| **Storage** | Cloudflare R2 (S3-compatible) |
| **CDN** | Cloudflare CDN (automatic with R2) |
| **Egress cost** | $0 (R2's key differentiator) |

**Rationale**: A media-heavy platform (profile photos, learning journals with images, marketplace content files, nature study photos) generates significant egress. R2's zero egress pricing saves potentially thousands per month at scale vs. AWS S3.

**Revision trigger**: None. S3-compatible API means migration is straightforward if needed.

### 2.11 Hosting: AWS (ECS + Managed Services)

**Satisfies**: `[S§17.3]` performance, `[S§17.5]` availability, `[S§17.1]` security

| Attribute | Value |
|-----------|-------|
| **Compute** | ECS on EC2 (Graviton): t4g.small (2 vCPU, 2 GB ARM), ECS-optimized AMI |
| **Sidecar** | Ory Kratos as second container in same ECS Task Definition |
| **Database** | RDS PostgreSQL 16: db.t4g.medium, Single-AZ, 20 GB gp3 |
| **Cache/Queue** | ElastiCache Redis 7: cache.t4g.micro, Single-AZ |
| **Load balancer** | ALB with TLS via ACM (free, auto-renewing certificates) |
| **Container registry** | ECR with lifecycle policy retaining last 10 images |
| **Region** | us-east-1 (N. Virginia) |
| **Est. cost** | ~$100-120/mo |

**Rationale**: A solo developer building a COPPA-regulated platform handling children's data cannot afford the operational risk of self-managing PostgreSQL (backups, upgrades, replication), Redis, TLS certificates, firewall rules, and OS patching. AWS managed services (RDS, ElastiCache, ACM, ALB) shift this operational burden to AWS. Go's efficient resource profile (20-60MB memory, steady CPU) keeps managed service costs predictable. ECS on EC2 with Graviton provides the best price-performance: t4g.small at ~$12/mo vs. Fargate's ~$25-35/mo for equivalent compute.

**Rejected alternatives**:
- **Hetzner dedicated server ($60/mo)**: ~50% cheaper, but requires self-managing PostgreSQL backups/PITR, Redis persistence, TLS certificates (Let's Encrypt), firewall rules (UFW), and OS patching. Operational burden and risk outweigh cost savings for a solo developer with COPPA obligations. (See ADR-010.)
- **EKS**: Control plane alone is $73/mo. Kubernetes is overkill for a single-binary monolith with a sidecar.
- **Lightsail**: Limited networking, no ECS integration, restrictive scaling options.

**Scaling path**:
- **Phase 1**: 1 EC2 instance, Single-AZ RDS + Redis (~$110/mo)
- **Phase 2**: Multiple EC2 instances or migrate to Fargate launch type, Multi-AZ RDS, ECS auto-scaling (~$250/mo)
- **Phase 3+**: Aurora PostgreSQL + read replicas, ElastiCache cluster, Typesense on separate ECS task (~$500-800/mo)

**Upgrade path — Fargate**: If managing the EC2 instance (AMI patching, capacity planning) becomes burdensome, migrate ECS tasks from EC2 to Fargate launch type. Same task definitions, same service — only the launch type changes.

**Revision trigger**: Consider Fargate when instance management overhead exceeds 2 hours/month. Consider dedicated servers only with a dedicated DevOps hire.

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

### 2.14 Monitoring & Alerting: Sentry + UptimeRobot + CloudWatch

**Satisfies**: `[S§17.5]` availability

| Component | Service | Purpose |
|-----------|---------|---------|
| **Error tracking** | Sentry | Go + React error capture, performance monitoring, release health |
| **Uptime** | UptimeRobot | External availability monitoring, alerting (5-min interval) |
| **Infrastructure** | CloudWatch | ECS, RDS, ElastiCache metrics, alarms, and log aggregation |

**Service Level Objectives (SLOs)** — Phase 1 targets:

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Availability** | 99.5% monthly | UptimeRobot external checks |
| **API latency (p50)** | < 50ms | Sentry performance monitoring |
| **API latency (p99)** | < 500ms | Sentry performance monitoring |
| **Error rate** | < 1% of requests | Sentry error tracking |
| **CSAM report latency** | < 5 minutes from detection to NCMEC filing | Application logs |

**CloudWatch alarm thresholds**:

| Alarm | Threshold | Action |
|-------|-----------|--------|
| ECS CPU | > 80% for 5 min | SNS → email alert |
| ECS Memory | > 85% for 5 min | SNS → email alert |
| RDS CPU | > 75% for 10 min | SNS → email alert |
| RDS free storage | < 5 GB | SNS → email alert |
| RDS connections | > 80% of max | SNS → email alert |
| ElastiCache memory | > 80% | SNS → email alert |
| ALB 5xx rate | > 5% for 5 min | SNS → email alert |
| ALB target response time | > 2s (p99) for 10 min | SNS → email alert |
| Dead-letter queue depth | > 0 for 15 min | SNS → email alert |

**Application-level metrics** (Phase 2 — via Sentry custom metrics or CloudWatch custom metrics):

- Quiz completion rate and average score
- Marketplace conversion rate (view → purchase)
- Search latency by scope (social, marketplace, learning)
- Background job queue depth and processing time per queue tier

**Revision trigger**: CloudWatch provides sufficient infrastructure metrics for Phase 1-2. Add Grafana + Prometheus when custom application metrics or cross-service dashboards are needed (Phase 3+).

### 2.15 CI/CD: GitHub Actions

**Satisfies**: `[S§17.1]` security (dependency scanning)

Pipeline stages:
1. `golangci-lint run` — lint
2. `go test ./...` — unit + integration tests
3. `govulncheck ./...` — dependency vulnerability scan
4. `npm audit` — frontend dependency scan
5. Docker multi-stage build
6. Push image to Amazon ECR, update ECS service (rolling deployment)

**Revision trigger**: None. GitHub Actions is sufficient for any scale of this project.

### 2.16 Real-Time: WebSockets via gorilla/websocket

**Satisfies**: `[S§7.5]` direct messaging, `[S§13]` notifications

| Attribute | Value |
|-----------|-------|
| **Protocol** | WebSocket (RFC 6455) |
| **Server** | gorilla/websocket integrated with Echo |
| **Pub/sub** | Redis pub/sub for multi-connection distribution |

**Rationale**: gorilla/websocket is the de facto Go WebSocket library with a clean API for upgrading HTTP connections. Each WebSocket connection runs in its own goroutine. Redis pub/sub distributes messages across WebSocket connections (and across servers when scaling horizontally).

**Revision trigger**: None for the WebSocket layer. Redis pub/sub scales to the connection counts needed through Phase 3.

### 2.17 Infrastructure as Code: AWS CDK (TypeScript)

**Satisfies**: `[S§17.5]` availability, `[S§17.1]` security (reproducible, auditable infrastructure)

| Attribute | Value |
|-----------|-------|
| **Language** | TypeScript |
| **Framework** | AWS CDK v2 |
| **Stack count** | 1 |
| **Constructs** | 9 |

**Rationale**: TypeScript is already in the stack (React frontend), so CDK adds no new language. CDK provides type-safe, high-level constructs that reduce boilerplate vs raw CloudFormation — a VPC with subnets is ~5 lines instead of ~50. A single `cdk deploy` provisions the entire Phase 1 infrastructure. `cdk synth` validates the template without touching AWS, catching misconfigurations before deployment.

**Rejected alternatives**:
- **Terraform / OpenTofu (HCL)**: HCL is another language to learn and maintain. CDK's L2 constructs are more concise for this AWS-only setup. Terraform's multi-cloud abstraction is unnecessary overhead when the project is committed to AWS (ADR-010).
- **Raw CloudFormation (YAML/JSON)**: Verbose, no type safety, no abstractions. A single CDK construct replaces hundreds of lines of CloudFormation.
- **Pulumi**: Smaller ecosystem, unnecessary multi-cloud abstraction layer, and less mature AWS construct library compared to CDK.

**Revision trigger**: If the project moves multi-cloud, evaluate Terraform. CDK's construct library is AWS-only.

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
                                     │   Go API        │
                                     │   (Echo)        │
                                     │                 │
                                     │  ┌───────────┐  │
                                     │  │  Domains  │  │
                                     │  │           │  │
                                     │  │ iam       │  │
                                     │  │ social    │  │
                                     │  │ learn     │  │
                                     │  │ mkt       │  │
                                     │  │ method    │  │
                                     │  │ discover  │  │
                                     │  │ onboard   │  │
                                     │  │ billing   │  │
                                     │  │ notify    │  │
                                     │  │ search    │  │
                                     │  │ comply    │  │
                                     │  │ safety    │  │
                                     │  │ recs      │  │
                                     │  │ media     │  │
                                     │  │ lifecycle │  │
                                     │  │ admin     │  │
                                     │  │ plan      │  │
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

Each spec domain `[S§2.1]` maps to a Go package within the monolith:

| Spec Domain | Go Package | Spec Reference | Key Responsibilities |
|-------------|------------|----------------|---------------------|
| Identity & Access | `iam` | `[S§3]` | Users, families, roles, permissions, sessions |
| Methodology | `method` | `[S§4]` | Definitions, tool registry, config propagation |
| Discovery | `discover` | `[S§5]` | Quiz engine, explorer content, state guides |
| Onboarding | `onboard` | `[S§6]` | Account setup wizard, roadmaps, recommendations |
| Social | `social` | `[S§7]` | Profiles, feed, friends, messaging, groups, events |
| Learning | `learn` | `[S§8]` | Tools, activities, journals, progress tracking, interactive assessments, lesson sequences, student assignments |
| Marketplace | `mkt` | `[S§9]` | Listings, purchases, reviews, creator dashboard |
| Recommendations & Signals | `recs` | `[S§10]` | Recommendation engine, content suggestions |
| Compliance & Reporting | `comply` | `[S§11]` | Attendance, assessments, portfolios, transcripts |
| Trust & Safety | `safety` | `[S§12]` | CSAM, moderation, reporting, bot prevention |
| Billing & Subscriptions | `billing` | `[S§15]` | Subscriptions, transactions, payouts |
| Notifications | `notify` | `[S§13]` | In-app, email, digests, preferences |
| Search | `search` | `[S§14]` | Full-text search, autocomplete, faceted filtering |
| Content & Media | `media` | `[S§2.1]` | Upload, processing, storage, delivery |
| Data Lifecycle | `lifecycle` | `[S§16.3]` | Data export, account deletion, retention, recovery |
| Administration | `admin` | `[S§3.1.5]` | Admin dashboard, user management, feature flags, system health |
| Planning & Scheduling | `plan` | `[S§18.9]` | Calendar views, schedule items, recurring plans |

### 3.3 Request Flow

A typical authenticated API request flows through these layers:

```
HTTP Request
    │
    ▼
┌─────────────────────┐
│ Echo Router         │  Route matching
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
│ Domain Handler      │  Business logic (e.g., social.CreatePost)
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ Service Layer       │  Orchestrates DB + cache + external services
└─────────┬───────────┘
          ▼
┌─────────────────────┐
│ Repository Layer    │  GORM queries (always family-scoped)
└─────────────────────┘
```

### 3.4 Module Structure

Each domain module follows a consistent internal structure:

```
cmd/
└── server/
    └── main.go             # Entry point
internal/
├── app/
│   └── app.go              # Echo app builder, router composition
├── config/
│   └── config.go           # Environment configuration
├── middleware/
│   ├── auth.go             # Kratos session validation
│   ├── family_scope.go     # Family context extraction
│   └── rate_limit.go       # Rate limiting
├── iam/
│   ├── handler.go          # Echo handlers (HTTP layer)
│   ├── service.go          # Business logic
│   ├── repository.go       # Database queries
│   ├── models.go           # GORM models + request/response types
│   └── ports.go            # Service + repository interfaces
├── social/
│   ├── handler.go
│   ├── service.go
│   ├── repository.go
│   ├── models.go
│   └── ports.go
├── learn/
│   └── ...
├── mkt/
│   └── ...
├── ... (all 17 domains)
└── shared/
    ├── db.go               # Database pool, transaction helpers
    ├── redis.go            # Redis connection pool
    ├── pagination.go       # Cursor-based pagination
    ├── family_scope.go     # FamilyScope type
    ├── errors.go           # AppError and error-to-HTTP mapping
    └── types.go            # Shared types (type aliases, newtypes)
migrations/
└── *.sql                   # goose SQL migrations
```

---

## 4. Hexagonal Architecture (Ports & Adapters)

Every domain in the monolith follows **hexagonal architecture**: the service interface
(`ports.go`) is the core, handlers are inbound adapters, and repositories / `adapters/*.go`
are outbound adapters.

> **Dependency Direction Rule** — All dependencies point inward toward `service.go`.
> Handlers and repositories depend on the service interface, never the reverse.

This is the strategic enabler for microservice extraction (§1.2): when a domain is extracted
to its own service, only the adapters change — the service interface is the seam. This
section bridges "we use Go + Echo + GORM" (§2) with "here are the file naming rules"
(CODING_STANDARDS.md §2.1).

Hexagonal architecture in Go is simple — an interface in `ports.go`, a struct behind it, DI
via constructor. Do not add adapter registries, port resolution frameworks, or abstraction
beyond the §4.3/§4.4 examples.

### 4.1 Pattern Stack Overview

The six patterns below are facets of the hexagonal architecture — domain isolation, port
definition, adapter implementation, invariant enforcement, cross-domain communication, and
read/write separation. Each is adopted because of specific project characteristics, not
generic "best practices."

| Layer | Pattern | Scope | Primary Benefit |
|-------|---------|-------|-----------------|
| Strategic | Modular Monolith with Bounded Contexts | Entire system | Domain isolation with extraction path |
| Intra-domain | Hexagonal Core (handler → service ← repository) | Every domain | Testable, swappable adapters; extraction-ready seams |
| Outbound ports | Repository Trait Pattern | Every domain | Service unit tests without DB |
| Inbound ports | Service Interface Traits | Every domain | Extraction-ready seams, handler decoupling |
| Complex domains | Domain Model Layer (Aggregates) | 6 domains | State machine invariant enforcement |
| Cross-domain | In-Process Domain Event Bus | System-wide | Cross-domain decoupling, no circular imports |
| Read-heavy | Lightweight CQRS | 6 domains | Separation of write/read path optimization |

### 4.2 Bounded Contexts (the 17 Modules)

The 17 Go packages (`iam`, `social`, `learn`, etc.) are **Bounded Contexts** in
Domain-Driven Design terminology. This is already committed to — this section names it
explicitly so the intent is clear to all contributors (human and AI).

**Why bounded contexts fit this project**: 17 distinct problem domains with their own
vocabulary, rules, and data ownership. All share infrastructure (one PostgreSQL database,
one Redis instance) but must not share implementation. The extraction path to microservices
(§1.2) is possible only if domain boundaries are respected from the start.

**Shared Kernel** — `internal/shared/` contains the minimal cross-cutting types all domains
need. Every addition to `shared/` is a deliberate decision, not a convenience refactor.
The shared kernel is strictly limited to:

| File | Contents |
|------|----------|
| `shared/family_scope.go` | `FamilyScope` type for privacy-enforcing queries |
| `shared/types.go` | Type aliases (`FamilyID`, `UserID`, `StudentID`, etc.) |
| `shared/db.go` | Database pool and transaction helpers |
| `shared/cache.go` | `Cache` port interface + generic `CacheGet[T]`/`CacheSet[T]` helpers |
| `shared/redis.go` | Redis-backed `Cache` implementation (`redisCache`; factory: `CreateCache`) |
| `shared/auth.go` | `SessionValidator` port — auth provider abstraction (Kratos wired in 01-iam) |
| `shared/error_reporter.go` | `ErrorReporter` port + `NoopErrorReporter` |
| `shared/jobs.go` | `JobEnqueuer` port — background job abstraction (asynq-backed; factory: `CreateJobEnqueuer`) |
| `shared/pagination.go` | Cursor-based and offset pagination |
| `shared/events.go` | `EventBus` and `DomainEvent` interface (§4.6) |
| `shared/errors.go` | `AppError` and error-to-HTTP mapping |

**Anti-Corruption Layers (ACLs)** — External services (Hyperswitch, Ory Kratos, S3/R2,
Thorn Safer, Postmark) are wrapped in Adapter modules within the domain that owns the
interaction. No raw SDK calls exist outside these adapters:

| External Service | Owning Domain | Interface (`ports.go`) | Adapter Location |
|-----------------|---------------|------------------------|-----------------|
| Hyperswitch (payments) | `mkt` | `PaymentProcessor` | `internal/mkt/adapters/payment.go` |
| Hyperswitch (billing) | `billing` | `PaymentProcessor` | `internal/billing/adapters/payment.go` |
| Ory Kratos (auth) | `iam` | `shared.SessionValidator` | `internal/iam/adapters/kratos.go` |
| Cloudflare R2 | `media` | `ObjectStorage` | `internal/media/adapters/r2.go` |
| Thorn Safer (CSAM) | `safety` | `CsamDetector` | `internal/safety/adapters/thorn.go` |
| AWS Rekognition | `safety` | `ContentModerator` | `internal/safety/adapters/rekognition.go` |
| AWS Comprehend (Phase 2) | `safety` | `TextScanner` | `internal/safety/adapters/comprehend.go` |
| Postmark (email) | `notify` | `Mailer` | `internal/notify/adapters/postmark.go` |
| Typesense (Phase 2) | `search` | `SearchEngine` | `internal/search/adapters/typesense.go` |

> **Note**: Both `mkt` and `billing` payment adapters talk to the same self-hosted
> Hyperswitch instance but use different Hyperswitch profiles (marketplace vs. subscription
> billing). Stripe is the underlying processor configured as a Hyperswitch connector.

> **Shared-level ACLs**: For vendor isolation at the infrastructure level (Redis/`Cache`,
> Sentry/`ErrorReporter`, asynq/`JobEnqueuer`), the port interface lives in
> `internal/shared/` rather than a domain `ports.go` — see the Shared Kernel table above.

**What bounded contexts rule out**:
- Domain A writing directly to domain B's prefixed tables
- Domain A calling domain B's `repository.go` directly
- Raw SDK calls scattered through `service.go` files
- Utility modules that don't belong to a domain or `shared/`

### 4.3 The Hexagonal Core (Handler → Service ← Repository)

The `handler.go / service.go / repository.go` file split is the hexagonal core of each
domain. The service interface (in `ports.go`) is the port. Handlers are inbound adapters;
repositories and `adapters/*.go` are outbound adapters:

```
HTTP Request  →  handler.go       (Inbound Adapter)
                      ↓
               service.go         (Application / Use-Case Layer — the "port")
                      ↓
               repository.go      (Outbound Adapter — database)
               adapters/*.go      (Outbound Adapter — third-party APIs)
```

**External service adapter pattern** — Integrations with third-party APIs MUST go through
dedicated `adapters/` files, not inline in `service.go`:

```
internal/billing/
├── handler.go
├── service.go
├── repository.go
├── models.go
├── ports.go
└── adapters/
    └── payment.go    ← wraps Hyperswitch SDK, returns domain types only
```

Services call the billing adapter's `CreateSubscription(...)` method which returns
`(SubscriptionID, error)`. Hyperswitch already provides processor-agnostic
flexibility — swapping the underlying payment processor (e.g., Stripe → Adyen) requires
only a Hyperswitch connector configuration change, no application code impact. The adapter
layer adds a second level of isolation: if Hyperswitch itself were ever replaced, only
`adapters/payment.go` changes.

**Adapter type boundary rule** — Adapters MUST:
1. Accept only domain types (or Go primitives) as input parameters. MUST NOT accept vendor SDK types.
2. Return only domain types (or Go primitives), or `error`. MUST NOT return vendor SDK types.
3. Convert vendor SDK errors to `shared.AppError` (or domain `DomainError`) before returning.
   Vendor error types (e.g., `*stripe.CardError`, `*redis.Error`) MUST NOT propagate to callers.
4. MUST NOT embed vendor SDK types in any struct used outside the adapter file itself.

The adapter file is the only place in the codebase where a vendor import is permitted for
that service. One import site = one change site on vendor swap.

**Why this fits**: The existing split is 95% of the way there. Making external adapters
explicit prevents the common pattern of SDK calls proliferating through service methods,
which makes testing and future vendor swaps painful.

### 4.4 Port Definitions (ports.go — All Interfaces)

Every domain service and every repository MUST be defined as a Go interface in `ports.go`
before the implementation. These interfaces are the hexagonal ports — the contracts that
make microservice extraction (§1.2) possible without changing callers.

#### Inbound Ports — Service Interfaces

Handlers receive service interfaces via dependency injection, never the concrete type:

```go
// internal/learn/ports.go
type LearningService interface {
    LogActivity(ctx context.Context, cmd LogActivityCommand, scope shared.FamilyScope) (uuid.UUID, error)
    GetProgressSummary(ctx context.Context, query ProgressSummaryQuery, scope shared.FamilyScope) (*ProgressSummary, error)
    // ... all use cases exposed to other layers
}

// internal/learn/service.go
type learningService struct {
    activities ActivityRepository
    events     *shared.EventBus
}

func NewLearningService(activities ActivityRepository, events *shared.EventBus) LearningService {
    return &learningService{activities: activities, events: events}
}

func (s *learningService) LogActivity(ctx context.Context, cmd LogActivityCommand, scope shared.FamilyScope) (uuid.UUID, error) {
    // ...
}
```

App setup wires the concrete type behind the interface:

```go
// In app setup (internal/app/app.go or cmd/server/main.go)
appState := &AppState{
    Learning: learn.NewLearningService(activityRepo, eventBus),
    // ...
}
```

#### Outbound Ports — Repository Interfaces

Services receive repository interfaces, not the concrete `pg*Repository`:

```go
// internal/learn/ports.go
type ActivityRepository interface {
    Create(ctx context.Context, cmd CreateActivity, scope shared.FamilyScope) (*Activity, error)
    List(ctx context.Context, query ActivityQuery, scope shared.FamilyScope) ([]Activity, error)
    GetProgressSummary(ctx context.Context, query ProgressQuery, scope shared.FamilyScope) (*ProgressSummary, error)
}

// internal/learn/repository.go
type pgActivityRepository struct {
    db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
    return &pgActivityRepository{db: db}
}

func (r *pgActivityRepository) Create(ctx context.Context, cmd CreateActivity, scope shared.FamilyScope) (*Activity, error) {
    // ...
}
```

**Naming conventions**:
- Interface: `{Domain}Service` / `{Entity}Repository` (e.g., `LearningService`, `ActivityRepository`)
- Implementation: unexported struct `{domain}Service` / `pg{Entity}Repository` (e.g., `learningService`, `pgActivityRepository`)
- Constructor: `New{Domain}Service(...)` / `New{Entity}Repository(...)` returns the interface type
- Interface file: `ports.go` within the domain package

**Why all domains, not just complex ones**:

1. **Extraction path** — When a domain is extracted as a microservice, only the adapter
   changes: swap `learningService` for `learningServiceHTTPClient`. Handlers are untouched.
   This seam must exist *before* extraction, not added during it.

2. **Consistent rule** — AI-generated code benefits from one rule ("all domains have a
   service interface") rather than a conditional rule ("complex domains get interfaces"). Conditional
   rules create ambiguity that leads to inconsistency.

3. **Minimal overhead** — Go interfaces are implicitly satisfied (no `implements` keyword).
   A service interface is ~10-15 lines per domain. No boilerplate.

4. **Authoritative contract** — The interface is the type-checked documentation of a
   domain's public use cases. Without it, there is no single place to see what a domain
   exposes to handlers or other domains.

**Phase 1 → Phase 2 extraction path (no handler changes required)**:

```
Phase 1 (monolith):
  appState.Learning = learn.NewLearningService(activityRepo, eventBus)

Phase 2 (extracted service):
  appState.Learning = learn.NewLearningServiceHTTPClient(baseURL)
  // Handlers are identical — they only see learn.LearningService interface
```

### 4.5 Domain Model Layer (Complex Domains Only)

For simple domains (Discovery, Notifications, Content/Media, Onboarding, AI, Search,
Billing), the service + repository split is sufficient — there is no complex business logic
that needs structural enforcement.

For complex domains, a `domain/` subdirectory adds **Aggregate Roots** and **Value Objects**
that enforce invariants structurally (not by convention):

| Domain | Complex? | Reason |
|--------|----------|--------|
| `learn/` | **Yes** | Activity logging invariants, tool lifecycles, multi-student assignment, progress state, quiz session lifecycle (not_started→in_progress→submitted→scored), lesson sequence progression, student assignment state machine |
| `social/` | **Yes** | Friend system invariants, visibility rules, blocking logic |
| `mkt/` | **Yes** | Listing lifecycle state machine (Draft → Review → Published → Archived), purchase invariants |
| `safety/` | **Yes** | Moderation state machine, CSAM handling pipeline |
| `comply/` | **Yes** | Attendance thresholds, GPA calculation, state config rules |
| `method/` | **Yes** | Tool activation rules, multi-methodology union logic |
| `discover/` | No | Read-only public content, simple queries |
| `notify/` | No | Event-triggered dispatch, no complex invariants |
| `media/` | No | Upload/process/store/serve, infrastructure domain |
| `search/` | No | Indexing and retrieval, no business invariants |
| `recs/` | No | Recommendation queries, no invariants to enforce |
| `onboard/` | No | Workflow steps, IAM interactions |
| `billing/` | No | Hyperswitch delegation; subscription state machine lives in the payment processor |
| `iam/` | No | Identity data, permissions; session state is Kratos' responsibility |

**Module structure for complex domains**:

```
internal/mkt/
├── handler.go
├── service.go
├── repository.go
├── models.go
├── ports.go
├── adapters/
│   └── payment.go
└── domain/
    ├── listing.go       ← Aggregate Root: MarketplaceListing
    └── value_objects.go
```

**Aggregate Root pattern** (Go-specific — unexported fields, state changes via methods only):

```go
// internal/mkt/domain/listing.go
package domain

type ListingState string

const (
    ListingStateDraft       ListingState = "draft"
    ListingStateUnderReview ListingState = "under_review"
    ListingStatePublished   ListingState = "published"
    ListingStateArchived    ListingState = "archived"
)

type MarketplaceListing struct {
    id        uuid.UUID     // unexported — state only changes via methods below
    state     ListingState
    creatorID uuid.UUID
    title     string
    priceCents uint32
}

func (l *MarketplaceListing) SubmitForReview() (*ListingSubmittedEvent, error) {
    if l.state != ListingStateDraft {
        return nil, &DomainError{
            From:   string(l.state),
            Action: "submit_for_review",
        }
    }
    l.state = ListingStateUnderReview
    return &ListingSubmittedEvent{ListingID: l.id, CreatorID: l.creatorID}, nil
}

func (l *MarketplaceListing) Publish(reviewerID uuid.UUID) (*ListingPublishedEvent, error) {
    if l.state != ListingStateUnderReview {
        return nil, &DomainError{
            From:   string(l.state),
            Action: "publish",
        }
    }
    l.state = ListingStatePublished
    return &ListingPublishedEvent{ListingID: l.id, ReviewerID: reviewerID}, nil
}
```

The service layer loads the aggregate from the repository, calls methods (which enforce
invariants and return domain events), then persists the updated aggregate and publishes events:

```go
// internal/mkt/service.go
func (s *marketplaceService) SubmitListingForReview(ctx context.Context, cmd SubmitListingCommand, scope shared.FamilyScope) error {
    listing, err := s.listings.Get(ctx, cmd.ListingID, scope)
    if err != nil {
        return err
    }
    event, err := listing.SubmitForReview()        // invariant enforced here
    if err != nil {
        return err
    }
    if err := s.listings.Save(ctx, listing, scope); err != nil {
        return err
    }
    return s.events.Publish(event)                 // domain event emitted
}
```

**Why it fits**: State machines (listing lifecycle, moderation pipeline, attendance
thresholds) MUST be enforced somewhere. An aggregate root with unexported fields and
method-only state transitions makes invalid transitions impossible from outside the package.
A fat service method could be bypassed. A Go aggregate with unexported fields cannot.

### 4.6 Domain Event Bus

When domain A completes an operation that domain B needs to react to, domain A MUST NOT
import domain B's service or call it directly. Instead, domain A publishes a domain event
and domain B subscribes to it.

**Implementation** — a lightweight in-process event bus, wired at application startup:

```go
// internal/shared/events.go

// DomainEvent is a marker interface for all domain events.
type DomainEvent interface {
    EventName() string
}

// DomainEventHandler handles a specific type of domain event.
type DomainEventHandler interface {
    Handle(ctx context.Context, event DomainEvent) error
}

// EventBus dispatches domain events to registered handlers.
type EventBus struct {
    // Internal dispatch map: reflect.Type → []DomainEventHandler
    // Implementation detail; callers use only Publish() and Subscribe()
    handlers map[reflect.Type][]DomainEventHandler
    mu       sync.RWMutex
}

func (b *EventBus) Publish(ctx context.Context, event DomainEvent) error { /* ... */ }
func (b *EventBus) Subscribe(eventType DomainEvent, handler DomainEventHandler) { /* ... */ }
```

**Cross-domain event flows**:

| Event (defined in) | Subscribing Domains | Effect |
|---|---|---|
| `ActivityLogged` (`learn/events`) | `comply`, `recs`, `notify` | Attendance tracking, recommendation signal, streak milestone check |
| `PostCreated` (`social/events`) | `safety`, `search` | Content scan, search index update |
| `PurchaseCompleted` (`mkt/events`) | `learn`, `billing`, `notify` | Tool access grant, creator earnings credit, receipt email |
| `ContentFlagged` (`safety/events`) | `notify` | Moderation queue alert |
| `MethodologyConfigUpdated` (`method/events`) | All domains | Config cache invalidation |
| `MilestoneAchieved` (`learn/events`) | `notify`, `social` | In-app + email notification, optional milestone post |
| `QuizCompleted` (`learn/events`) | `notify`, `recs` | Score notification, recommendation signal |
| `SequenceAdvanced` (`learn/events`) | `recs` | Recommendation signal for sequence engagement |
| `SequenceCompleted` (`learn/events`) | `notify`, `recs` | Completion notification, recommendation signal |
| `AssignmentCompleted` (`learn/events`) | `notify` | Notify parent of assignment completion |

**Event ownership rule** — Event types are defined in the *emitting* domain's `events.go`
file. The consuming domain imports the event type, never the emitting domain's service:

```go
// internal/learn/events.go  — defined here, consumed elsewhere
package learn

type ActivityLogged struct {
    FamilyID        uuid.UUID
    StudentID       uuid.UUID
    ActivityID      uuid.UUID
    Subject         string
    DurationMinutes uint32
}

func (e ActivityLogged) EventName() string { return "learn.activity_logged" }

// internal/comply/event_handlers.go  — subscribes here
package comply

type ActivityLoggedHandler struct {
    complyService ComplianceService
}

func (h *ActivityLoggedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
    e := event.(learn.ActivityLogged)
    return h.complyService.RecordAttendance(ctx, &e)
}
```

**Phase 1 → Phase 2 transition**:

- **Phase 1** (current): Synchronous dispatch within the same request. `EventBus.Publish`
  calls handlers inline before the request returns. Simple, no message broker needed,
  consistent (event handlers participate in the request's error context).

- **Phase 2** (when async is needed): Handlers that do heavy work (CSAM scanning, search
  indexing, email sending) enqueue a background job instead of executing inline.
  The event handler's `Handle()` method enqueues a job to Redis; the actual work runs
  in an asynq worker. The event bus is unchanged; only the handler implementation
  changes.

**What the event bus rules out**: Domain A importing domain B's service to call it directly
in response to a domain A event. That pattern creates coupling that prevents independent
extraction and creates circular dependency risks.

### 4.7 Lightweight CQRS

This is NOT full CQRS (no separate event store, no separate read model database). Instead:
separate **command functions** (writes with side effects) from **query functions** (reads
optimized for the UI) within the same service and repository.

**Applies to these domains** (identified from spec analysis — clear read/write asymmetry):

| Domain | Write (command) side | Read (query) side |
|--------|---------------------|-------------------|
| `social/` | Post creation, friend request | Feed aggregation, friend timeline |
| `mkt/` | Listing CRUD, purchase flow | Faceted marketplace browse |
| `learn/` | Activity log, book completion | Progress trends, subject balance |
| `comply/` | Attendance mark, assessment record | Attendance summary, threshold check |
| `search/` | Index document, update index | Full-text search, autocomplete |
| `recs/` | Record learning signal | Fetch pre-computed recommendations |

**Implementation pattern** — command and query functions coexist in the same service interface
but are clearly separated by naming and return type conventions:

```go
// internal/learn/ports.go
type LearningService interface {
    // --- Command side (write, has side effects) ---
    // Return only IDs or nothing — never rich reads after write (no "return what you created")
    LogActivity(ctx context.Context, cmd LogActivityCommand, scope shared.FamilyScope) (uuid.UUID, error)
    CompleteBook(ctx context.Context, cmd CompleteBookCommand, scope shared.FamilyScope) error

    // --- Query side (read, no side effects) ---
    GetProgressSummary(ctx context.Context, query ProgressSummaryQuery, scope shared.FamilyScope) (*ProgressSummary, error)
    GetFeed(ctx context.Context, query FeedQuery, scope shared.FamilyScope) (*FeedPage, error)
}
```

Read-side query optimization is progressive (never add the next level until the previous
is measured as insufficient):

| Level | Mechanism | When to use |
|-------|-----------|-------------|
| 0 | Standard GORM query | Always start here |
| 1 | PostgreSQL aggregate / window functions | Complex analytics (progress trends, subject balance) |
| 2 | Materialized views | Expensive pre-computations refreshed on schedule |
| 3 | Redis sorted sets / caches | Feed data, frequently accessed aggregates |
| 4 | Read replica | Write throughput exceeds single-server capacity (~Phase 3) |

**Why it fits**: The spec identifies clear read/write asymmetries — social feed reads vastly
exceed post writes; marketplace browse dwarfs listing updates; progress dashboard queries
are more complex than activity log writes. Without separating these, the repository becomes
a mixed bag of CRUD and complex analytical queries. Separating command and query *functions*
(not separate stores) creates the seam needed to later add caching, materialized views, or
a read replica without restructuring the service interface.

### 4.8 Patterns NOT Used and Why

| Pattern | Decision | Rationale |
|---------|----------|-----------|
| **Full Event Sourcing** | Rejected | Massive operational complexity (append-only log, snapshot management, replay logic). No consistency requirement that demands it. Domain events (§4.6) provide cross-domain decoupling without the overhead. |
| **Saga Orchestration** | Deferred | Current cross-domain flows (purchase → access + earnings + notification) are simple enough for the event bus. Add sagas only if distributed transactions become a problem after service extraction. |
| **Actor Model** | Rejected | Over-complex for this use case. Goroutines + Echo handle concurrency well. The actor model adds message-passing overhead without meaningful benefit for request/response workloads. |
| **Anemic Domain Model** | Rejected | Domains with rich invariants (Marketplace, Trust & Safety, Compliance) have state machines that MUST be enforced structurally, not by convention. Anemic models push invariant enforcement into service methods that can be bypassed. |
| **Command Bus (MediatR style)** | Rejected | Too much indirection for Go. Direct method calls on interfaces (§4.4) are idiomatic, explicit, and type-checked. MediatR-style patterns obscure data flow in ways that make AI-generated code harder to review. |
| **Separate Read Model (full CQRS)** | Deferred | Start with query/command separation within the same repository (§4.7). Add a separate read store (e.g., denormalized Redis hashes) only when the progressive optimization ladder (§4.7) is insufficient. |

---

## 5. Data Architecture

### 5.1 Schema Organization

Tables are prefixed by domain to avoid collision and provide clear ownership. `[S§16]`

| Domain | Prefix | Key Tables |
|--------|--------|------------|
| Identity & Access | `iam_` | `iam_families`, `iam_parents`, `iam_students`, `iam_roles` |
| Methodology | `method_` | `method_definitions`, `method_tool_registry`, `method_philosophy_modules` |
| Discovery | `disc_` | `disc_quiz_definitions`, `disc_quiz_results`, `disc_state_guides` |
| Onboarding | `onb_` | `onb_roadmaps`, `onb_wizard_progress` |
| Social | `soc_` | `soc_profiles`, `soc_posts`, `soc_comments`, `soc_friendships`, `soc_messages`, `soc_groups`, `soc_events` |
| Learning | `learn_` | `learn_activities`, `learn_journals`, `learn_reading_lists`, `learn_progress`, `learn_questions`, `learn_quiz_defs`, `learn_quiz_sessions`, `learn_sequence_defs`, `learn_sequence_progress`, `learn_student_assignments` |
| Marketplace | `mkt_` | `mkt_creators`, `mkt_listings`, `mkt_purchases`, `mkt_reviews`, `mkt_files` |
| Recommendations & Signals | `recs_` | `recs_signals`, `recs_recommendations` |
| Compliance | `comply_` | `comply_attendance`, `comply_state_configs`, `comply_portfolios` |
| Trust & Safety | `safety_` | `safety_reports`, `safety_mod_actions`, `safety_content_flags` |
| Billing | `bill_` | `bill_subscriptions`, `bill_transactions`, `bill_payouts` |
| Notifications | `notify_` | `notify_notifications`, `notify_preferences`, `notify_digests` |
| Search | `search_` | (uses FTS indexes on domain tables directly) |
| Content & Media | `media_` | `media_uploads`, `media_processing_jobs` |

### 5.2 Core Schema Design

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

-- Interactive Assessment Engine [S§8.1.9]
-- Layer 1: Publisher-scoped content definitions (no RLS)
CREATE TABLE learn_questions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id    UUID NOT NULL,                  -- references mkt_publishers
    question_type   TEXT NOT NULL CHECK (question_type IN (
                        'multiple_choice', 'fill_in_blank', 'true_false',
                        'matching', 'ordering', 'short_answer'
                    )),
    content         TEXT NOT NULL,                  -- question text (markdown)
    answer_data     JSONB NOT NULL,                 -- type-specific: choices[], correct_answer, match_pairs[], etc.
    subject_tags    TEXT[] NOT NULL DEFAULT '{}',
    methodology_id  UUID REFERENCES method_definitions(id),
    difficulty_level TEXT CHECK (difficulty_level IN ('beginner', 'intermediate', 'advanced')),
    auto_scorable   BOOLEAN NOT NULL DEFAULT true,
    points          SMALLINT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE learn_quiz_defs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id    UUID NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT,
    subject_tags    TEXT[] NOT NULL DEFAULT '{}',
    methodology_id  UUID REFERENCES method_definitions(id),
    time_limit_minutes SMALLINT,                    -- NULL = no time limit
    passing_score_percent SMALLINT NOT NULL DEFAULT 70,
    shuffle_questions BOOLEAN NOT NULL DEFAULT false,
    show_correct_after BOOLEAN NOT NULL DEFAULT true,
    question_count  SMALLINT NOT NULL DEFAULT 0,    -- denormalized
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Layer 3: Family-scoped quiz taking (RLS-protected)
CREATE TABLE learn_quiz_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    student_id      UUID NOT NULL REFERENCES iam_students(id),
    quiz_def_id     UUID NOT NULL REFERENCES learn_quiz_defs(id),
    status          TEXT NOT NULL DEFAULT 'not_started'
                    CHECK (status IN ('not_started', 'in_progress', 'submitted', 'scored')),
    started_at      TIMESTAMPTZ,
    submitted_at    TIMESTAMPTZ,
    scored_at       TIMESTAMPTZ,
    score           SMALLINT,
    max_score       SMALLINT,
    passed          BOOLEAN,
    answers         JSONB DEFAULT '[]',             -- [{question_id, response, is_correct, points_awarded}]
    scored_by       UUID REFERENCES iam_parents(id) -- NULL = auto-scored
);

CREATE INDEX idx_quiz_sessions_family ON learn_quiz_sessions(family_id, student_id);

-- Lesson Sequences [S§8.1.12]
CREATE TABLE learn_sequence_defs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id    UUID NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT,
    subject_tags    TEXT[] NOT NULL DEFAULT '{}',
    methodology_id  UUID REFERENCES method_definitions(id),
    is_linear       BOOLEAN NOT NULL DEFAULT true,  -- must complete in order
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE learn_sequence_progress (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    student_id      UUID NOT NULL REFERENCES iam_students(id),
    sequence_def_id UUID NOT NULL REFERENCES learn_sequence_defs(id),
    current_item_index SMALLINT NOT NULL DEFAULT 0,
    status          TEXT NOT NULL DEFAULT 'not_started'
                    CHECK (status IN ('not_started', 'in_progress', 'completed')),
    item_completions JSONB DEFAULT '[]',
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    UNIQUE (family_id, student_id, sequence_def_id)
);

CREATE INDEX idx_sequence_progress_family ON learn_sequence_progress(family_id, student_id);

-- Student Assignments [S§8.6]
CREATE TABLE learn_student_assignments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    student_id      UUID NOT NULL REFERENCES iam_students(id),
    assigned_by     UUID NOT NULL REFERENCES iam_parents(id),
    content_type    TEXT NOT NULL,                   -- quiz_def, sequence_def, video_def, etc.
    content_id      UUID NOT NULL,
    due_date        DATE,
    status          TEXT NOT NULL DEFAULT 'assigned'
                    CHECK (status IN ('assigned', 'in_progress', 'completed', 'skipped')),
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_assignments_family ON learn_student_assignments(family_id, student_id);
```

#### Marketplace `[S§9]`

> **Refinement note**: This baseline schema has been superseded by `specs/domains/07-mkt.md`,
> which introduces the following changes:
> - `stripe_account_id` → `payment_account_id` (processor-agnostic via Hyperswitch)
> - `stripe_payment_id` → `payment_id`
> - Adds tables: `mkt_publishers`, `mkt_listing_files`, `mkt_listing_versions`,
>   `mkt_cart_items`, `mkt_curated_sections`
>
> The SQL below is preserved as the original architectural baseline. Use 07-mkt.md for the
> current schema definition.

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

### 5.3 JSONB Usage Patterns

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

### 5.4 PostGIS for Location Discovery `[S§7.8]`

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

### 5.5 Entity Relationship Summary

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
               ├── 1:N ── learn_quiz_sessions ──── N:1 ── learn_quiz_defs
               │                                              │
               ├── 1:N ── learn_sequence_progress ── N:1 ── learn_sequence_defs
               │
               ├── 1:N ── learn_student_assignments
               │
               ├── 1:N ── iam_student_sessions     -- supervised student views [S§8.6]
               │
               └── N:M ── soc_groups (via soc_group_members)

learn_quiz_defs ── N:M ── learn_questions (via learn_quiz_questions)
learn_sequence_defs ── 1:N ── learn_sequence_items

method_definitions ── N:M ── method_tools (via method_tool_activations)
```

---

## 6. Authentication & Authorization

### 6.1 Ory Kratos Configuration

Kratos runs as a sidecar container alongside the Go API, managing the full authentication lifecycle. `[S§3]`

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

### 6.2 Auth Middleware (Go)

```go
// internal/middleware/auth.go
package middleware

import (
    "net/http"

    "github.com/google/uuid"
    "github.com/labstack/echo/v4"
    "gorm.io/gorm"
)

// AuthContext holds the authenticated user context extracted from the auth provider session.
type AuthContext struct {
    ParentID         uuid.UUID
    FamilyID         uuid.UUID
    IdentityID       uuid.UUID
    IsPrimaryParent  bool
    SubscriptionTier SubscriptionTier
    Email            string
}

type SubscriptionTier string

const (
    SubscriptionTierFree    SubscriptionTier = "free"
    SubscriptionTierPremium SubscriptionTier = "premium"
)

// AuthMiddleware validates Kratos session cookie and builds AuthContext.
func AuthMiddleware(kratosClient *KratosClient, db *gorm.DB) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            // Extract session cookie
            sessionCookie := c.Request().Header.Get("Cookie")
            if sessionCookie == "" {
                return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
            }

            // Validate with Kratos
            kratosSession, err := kratosClient.ToSession(sessionCookie)
            if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
            }

            kratosIdentityID, err := uuid.Parse(kratosSession.Identity.ID)
            if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
            }

            // Look up parent + family from our DB
            parent, err := findParentByKratosID(db, kratosIdentityID)
            if err != nil || parent == nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
            }

            family, err := findFamily(db, parent.FamilyID)
            if err != nil || family == nil {
                return echo.NewHTTPError(http.StatusUnauthorized, "authentication required")
            }

            tier := SubscriptionTierFree
            if family.SubscriptionTier == "premium" {
                tier = SubscriptionTierPremium
            }

            authCtx := AuthContext{
                ParentID:         parent.ID,
                FamilyID:         family.ID,
                IdentityID:       kratosIdentityID,
                IsPrimaryParent:  parent.IsPrimary,
                SubscriptionTier: tier,
                Email:            parent.Email,
            }

            // Insert into Echo context for handlers to extract
            c.Set("auth", authCtx)
            return next(c)
        }
    }
}
```

### 6.3 COPPA Consent State Machine `[S§17.2]`

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

```go
// internal/iam/models.go
type CoppaConsentStatus string

const (
    CoppaRegistered CoppaConsentStatus = "registered"
    CoppaNoticed    CoppaConsentStatus = "noticed"
    CoppaConsented  CoppaConsentStatus = "consented"
    CoppaReVerified CoppaConsentStatus = "re_verified"
    CoppaWithdrawn  CoppaConsentStatus = "withdrawn"
)

// RequireCoppaConsent is Echo middleware that ensures COPPA consent before accessing student data.
func RequireCoppaConsent(db *gorm.DB) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            authCtx := c.Get("auth").(AuthContext)

            family, err := findFamily(db, authCtx.FamilyID)
            if err != nil || family == nil {
                return echo.NewHTTPError(http.StatusNotFound, "resource not found")
            }

            switch family.CoppaConsentStatus {
            case CoppaConsented, CoppaReVerified:
                return next(c)
            default:
                return echo.NewHTTPError(http.StatusForbidden, "parental consent required")
            }
        }
    }
}
```

### 6.4 Family Account Model `[S§3.1]`

```go
// internal/iam/handler.go

// HandlePostRegistration is a Kratos post-registration webhook that creates family + parent atomically. [S§6.1]
func HandlePostRegistration(db *gorm.DB) echo.HandlerFunc {
    return func(c echo.Context) error {
        var payload KratosWebhookPayload
        if err := c.Bind(&payload); err != nil {
            return echo.NewHTTPError(http.StatusBadRequest, "invalid payload")
        }

        tx := db.Begin()
        if tx.Error != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
        }
        defer tx.Rollback()

        // Create family with default methodology
        defaultMethodology, err := findMethodologyBySlug(tx, "traditional")
        if err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
        }

        family := &Family{
            DisplayName:          payload.Traits.Name,
            PrimaryMethodologyID: defaultMethodology.ID,
            // Methodology will be properly set during onboarding wizard [S§6.3]
        }
        if err := tx.Create(family).Error; err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
        }

        // Create parent linked to Kratos identity
        parent := &Parent{
            FamilyID:         family.ID,
            IdentityID:       payload.IdentityID,
            DisplayName:      payload.Traits.Name,
            Email:            payload.Traits.Email,
            IsPrimary:        true,
        }
        if err := tx.Create(parent).Error; err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
        }

        // Set primary parent reference
        if err := tx.Model(family).Update("primary_parent_id", parent.ID).Error; err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
        }

        // Create social profile
        profile := &SocialProfile{FamilyID: family.ID}
        if err := tx.Create(profile).Error; err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
        }

        if err := tx.Commit().Error; err != nil {
            return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
        }

        return c.NoContent(http.StatusOK)
    }
}
```

### 6.5 Role-Based Access Control `[S§3.2]`

Permission checks are implemented as Echo middleware and helper functions:

```go
// internal/middleware/permissions.go

// RequirePremium is Echo middleware that requires premium subscription. [S§15.2]
func RequirePremium() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            authCtx := c.Get("auth").(AuthContext)
            if authCtx.SubscriptionTier != SubscriptionTierPremium {
                return echo.NewHTTPError(http.StatusPaymentRequired, "premium subscription required")
            }
            return next(c)
        }
    }
}

// RequireCreator is Echo middleware that requires creator role. [S§3.1.4]
func RequireCreator(db *gorm.DB) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            authCtx := c.Get("auth").(AuthContext)
            creator, err := findCreatorByParentID(db, authCtx.ParentID)
            if err != nil || creator == nil {
                return echo.NewHTTPError(http.StatusForbidden, "creator role required")
            }
            c.Set("creator_id", creator.ID)
            return next(c)
        }
    }
}

// Usage in handlers:
// Route: e.GET("/v1/comply/portfolios/:student_id", GeneratePortfolio, RequirePremium())
func GeneratePortfolio(c echo.Context) error {
    authCtx := c.Get("auth").(AuthContext)
    studentID, err := uuid.Parse(c.Param("student_id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
    }

    // Verify student belongs to this family (family-scoped)
    student, err := findStudentByIDAndFamily(db, studentID, authCtx.FamilyID)
    if err != nil || student == nil {
        return echo.NewHTTPError(http.StatusNotFound, "resource not found")
    }

    // Generate portfolio...
    return c.JSON(http.StatusOK, portfolio)
}
```

---

## 7. Methodology System Implementation

### 7.1 Config-Driven Architecture `[S§4.1]`

The methodology system is the platform's most distinctive architectural pattern. Every methodology-dependent behavior resolves through configuration lookup — never through conditionals.

```go
// internal/method/service.go

// ResolveFamilyTools resolves the active tool set for a family based on methodology selections. [S§4.2]
func ResolveFamilyTools(ctx context.Context, db *gorm.DB, familyID uuid.UUID) ([]ActiveTool, error) {
    // Get family's methodology selections
    var family Family
    if err := db.WithContext(ctx).First(&family, "id = ?", familyID).Error; err != nil {
        return nil, fmt.Errorf("family not found: %w", err)
    }

    // Collect all methodology IDs (primary + secondary) [S§4.3]
    methodologyIDs := []uuid.UUID{family.PrimaryMethodologyID}
    methodologyIDs = append(methodologyIDs, family.SecondaryMethodologyIDs...)

    // Union of all activated tools across selected methodologies [S§4.2]
    var activations []MethodToolActivation
    if err := db.WithContext(ctx).
        Preload("Tool").
        Where("methodology_id IN ?", methodologyIDs).
        Find(&activations).Error; err != nil {
        return nil, fmt.Errorf("failed to load tool activations: %w", err)
    }

    // Deduplicate (a tool activated by multiple methodologies appears once)
    seen := make(map[uuid.UUID]bool)
    var activeTools []ActiveTool
    for _, activation := range activations {
        if activation.Tool == nil || seen[activation.Tool.ID] {
            continue
        }
        seen[activation.Tool.ID] = true
        activeTools = append(activeTools, ActiveTool{
            ToolID:          activation.Tool.ID,
            Slug:            activation.Tool.Slug,
            DisplayName:     activation.Tool.DisplayName,
            Tier:            activation.Tool.Tier,
            ConfigOverrides: activation.ConfigOverrides,
        })
    }

    return activeTools, nil
}

// ResolveStudentTools resolves tools for a specific student, considering per-student overrides. [S§4.6]
func ResolveStudentTools(ctx context.Context, db *gorm.DB, familyID, studentID uuid.UUID) ([]ActiveTool, error) {
    var student Student
    if err := db.WithContext(ctx).
        Where("id = ? AND family_id = ?", studentID, familyID).
        First(&student).Error; err != nil {
        return nil, fmt.Errorf("student not found: %w", err)
    }

    if student.MethodologyOverrideID != nil {
        // Student has override — use their personal methodology [S§4.6]
        var activations []MethodToolActivation
        if err := db.WithContext(ctx).
            Preload("Tool").
            Where("methodology_id = ?", *student.MethodologyOverrideID).
            Find(&activations).Error; err != nil {
            return nil, fmt.Errorf("failed to load tool activations: %w", err)
        }

        var tools []ActiveTool
        for _, a := range activations {
            if a.Tool == nil {
                continue
            }
            tools = append(tools, ActiveTool{
                ToolID:          a.Tool.ID,
                Slug:            a.Tool.Slug,
                DisplayName:     a.Tool.DisplayName,
                Tier:            a.Tool.Tier,
                ConfigOverrides: a.ConfigOverrides,
            })
        }
        return tools, nil
    }

    // No override — use family-level tools
    return ResolveFamilyTools(ctx, db, familyID)
}
```

### 7.2 Methodology-Aware API Responses

API responses include methodology context so the frontend renders appropriately:

```go
// internal/method/models.go

// DashboardResponse is shaped by methodology. [S§4.4]
type DashboardResponse struct {
    Family            FamilySummary      `json:"family"`
    Students          []StudentSummary   `json:"students"`
    ActiveTools       []ActiveTool       `json:"active_tools"`
    MethodologyContext MethodologyContext `json:"methodology_context"`
    RoadmapProgress   *RoadmapProgress   `json:"roadmap_progress,omitempty"` // [S§6.4]
}

type MethodologyContext struct {
    Primary     MethodologySummary   `json:"primary"`
    Secondary   []MethodologySummary `json:"secondary"`
    // Methodology-specific terminology overrides
    // e.g., Charlotte Mason calls activities "lessons", Unschooling calls them "explorations"
    Terminology json.RawMessage      `json:"terminology"`
    // Current mastery path level [S§4.1]
    MasteryLevel *string             `json:"mastery_level,omitempty"`
}
```

### 7.3 Adding a New Methodology

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

## 8. File & Media Architecture

### 8.1 Upload Pipeline `[S§14 Content & Media]`

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

```go
// internal/media/handler.go

// RequestUpload handles upload request.
func RequestUpload(c echo.Context) error {
    authCtx := c.Get("auth").(AuthContext)

    var req UploadRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid request")
    }

    // Validate file type and size limits
    var maxSize int64
    switch req.Context {
    case UploadContextProfilePhoto:
        maxSize = 5 * 1024 * 1024 // 5MB
    case UploadContextJournalImage:
        maxSize = 10 * 1024 * 1024 // 10MB
    case UploadContextMarketplaceFile:
        maxSize = 500 * 1024 * 1024 // 500MB
    default:
        maxSize = 10 * 1024 * 1024
    }

    allowedTypes := getAllowedTypes(req.Context)
    if !contains(allowedTypes, req.ContentType) {
        return echo.NewHTTPError(http.StatusBadRequest, "file type not allowed")
    }

    // Generate upload record
    upload, err := createUpload(db, CreateUpload{
        FamilyID:    authCtx.FamilyID,
        Filename:    req.Filename,
        ContentType: req.ContentType,
        SizeBytes:   req.SizeBytes,
        Context:     req.Context,
        Status:      UploadStatusPending,
    })
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
    }

    // Generate presigned PUT URL for direct upload to R2
    presignedURL, err := r2Client.PresignedPut(upload.StorageKey(), maxSize, req.ContentType)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
    }

    return c.JSON(http.StatusOK, UploadResponse{
        UploadID:         upload.ID,
        PresignedURL:     presignedURL,
        ExpiresInSeconds: 3600,
    })
}
```

### 8.2 Image Processing

After upload confirmation, a background job handles image processing:

```go
// internal/media/jobs.go

// ProcessImage is a background job that processes an uploaded image.
func ProcessImage(ctx context.Context, uploadID uuid.UUID, state *AppState) error {
    upload, err := findUpload(state.DB, uploadID)
    if err != nil {
        return fmt.Errorf("upload not found: %w", err)
    }

    // 1. CSAM scan (all images) [S§12.1]
    if strings.HasPrefix(upload.ContentType, "image/") {
        scanResult, err := state.ThornClient.Scan(ctx, upload.StorageKey())
        if err != nil {
            return fmt.Errorf("CSAM scan failed: %w", err)
        }
        if scanResult.IsCSAM {
            // Quarantine immediately, report to NCMEC [S§12.1]
            if err := quarantineUpload(state.DB, uploadID); err != nil {
                return err
            }
            return reportCSAM(ctx, state, uploadID, scanResult)
        }
    }

    // 2. Generate thumbnails for images
    if strings.HasPrefix(upload.ContentType, "image/") {
        variants := []ImageVariant{
            {Suffix: "thumb", MaxWidth: 200, MaxHeight: 200},
            {Suffix: "medium", MaxWidth: 800, MaxHeight: 800},
        }
        for _, variant := range variants {
            resized, err := resizeImage(upload, variant)
            if err != nil {
                return fmt.Errorf("resize failed: %w", err)
            }
            if err := state.R2Client.Put(ctx, variantKey(upload, variant), resized); err != nil {
                return fmt.Errorf("upload variant failed: %w", err)
            }
        }
    }

    // 3. General content moderation [S§12.2]
    if strings.HasPrefix(upload.ContentType, "image/") {
        moderation, err := state.RekognitionClient.DetectModerationLabels(ctx, upload)
        if err != nil {
            return fmt.Errorf("moderation check failed: %w", err)
        }
        if moderation.HasViolations() {
            return flagUpload(state.DB, uploadID, moderation)
        }
    }

    // 4. Mark as published
    return publishUpload(state.DB, uploadID)
}
```

### 8.3 Video Transcoding Pipeline `[S§8.1.11]`

Creator-uploaded videos are processed into HLS adaptive bitrate streams for in-platform playback:

```
Creator Upload                  media                           R2 Storage
  │                               │                                │
  │  1. Upload raw video          │                                │
  │  (via presigned PUT)          │                                │
  │  ──────────────────────────────────────────────────────────────►│
  │                               │                                │
  │  2. Confirm upload            │                                │
  │  ──────────────────────────►  │                                │
  │                               │  3. Enqueue TranscodeVideoJob  │
  │                               │  ──────────────────────────►   │
  │                               │                                │
  │                               │  4. FFmpeg: generate HLS       │
  │                               │     480p / 720p / 1080p        │
  │                               │     + master.m3u8 playlist     │
  │                               │  ──────────────────────────►   │
  │                               │                                │
  │                               │  5. Store segments + playlist  │
  │                               │  ──────────────────────────►   │
```

**Key details**:
- Raw video stored in R2 (up to 5 GB); transcoded to HLS segments at 480p/720p/1080p
- `media_transcode_jobs` table tracks job status, input/output keys, resolutions, duration
- Delivery via signed HLS URLs (4-hour expiry), CDN-served segments
- External videos (YouTube/Vimeo) stored as metadata only — JS API integration for progress tracking
- See `specs/domains/09-media.md §10.8` for full pipeline specification

### 8.4 Marketplace File Delivery `[S§9.4]`

Purchased marketplace files are delivered via time-limited signed URLs:

```go
// internal/mkt/handler.go

// DownloadPurchasedFile handles downloading purchased marketplace content.
func DownloadPurchasedFile(c echo.Context) error {
    authCtx := c.Get("auth").(AuthContext)

    listingID, err := uuid.Parse(c.Param("listing_id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid listing ID")
    }
    fileID, err := uuid.Parse(c.Param("file_id"))
    if err != nil {
        return echo.NewHTTPError(http.StatusBadRequest, "invalid file ID")
    }

    // Verify purchase exists [S§9.4]
    _, err = findPurchase(db, authCtx.FamilyID, listingID)
    if err != nil {
        return echo.NewHTTPError(http.StatusForbidden, "content not purchased")
    }

    file, err := findListingFile(db, listingID, fileID)
    if err != nil {
        return echo.NewHTTPError(http.StatusNotFound, "resource not found")
    }

    // Generate time-limited signed download URL (1 hour)
    signedURL, err := r2Client.PresignedGet(file.StorageKey, 3600)
    if err != nil {
        return echo.NewHTTPError(http.StatusInternalServerError, "an internal error occurred")
    }

    return c.JSON(http.StatusOK, DownloadResponse{DownloadURL: signedURL})
}
```

---

## 9. Search Architecture

### 9.1 Phase 1: PostgreSQL Full-Text Search `[S§14]`

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

### 9.2 Phase 2+: Typesense Migration Path

When marketplace exceeds ~100K listings or search latency exceeds 500ms p95:

```go
// internal/search/service.go

// SearchBackend represents the search implementation to use.
type SearchBackend interface {
    SearchMarketplace(ctx context.Context, query *MarketplaceSearchQuery) (SearchResults, error)
}

// PostgresFTSBackend uses PostgreSQL full-text search.
type PostgresFTSBackend struct {
    db *gorm.DB
}

func (b *PostgresFTSBackend) SearchMarketplace(ctx context.Context, query *MarketplaceSearchQuery) (SearchResults, error) {
    // Use PostgreSQL FTS queries from §9.1
    return postgresMarketplaceSearch(ctx, b.db, query)
}

// TypesenseBackend uses Typesense for typo tolerance, faceted filtering,
// and instant search at any scale. [S§14.2]
// Built-in Raft HA — no enterprise license required.
type TypesenseBackend struct {
    adapter TypesenseAdapter
}

func (b *TypesenseBackend) SearchMarketplace(ctx context.Context, query *MarketplaceSearchQuery) (SearchResults, error) {
    tsQuery := buildTypesenseMarketplaceQuery(query)
    results, err := b.adapter.Search(ctx, "marketplace_listings", tsQuery)
    if err != nil {
        return SearchResults{}, fmt.Errorf("typesense query failed: %w", err)
    }
    return convertTypesenseToSearchResponse(results), nil
}
```

**Migration strategy**: Zero-downtime switchover.
1. Index existing PostgreSQL data into Typesense.
2. Run both backends in parallel, comparing results (shadow mode).
3. Switch reads to Typesense.
4. Maintain PostgreSQL FTS indexes as fallback.
5. Remove PostgreSQL FTS indexes only after Typesense has proven stable for 30+ days.

---

## 10. API Design

### 10.1 RESTful JSON API `[S§17.3]`

The API follows REST conventions with consistent patterns across all 17 domains:

```
Base URL: https://api.homegrown.academy/v1

Authentication: Kratos session cookie (browser)
Content-Type: application/json
Rate Limiting: Token bucket per user (100 req/min default) [S§2.3]
```

### 10.2 URL Structure

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

# Interactive Learning [S§8.1.9-8.1.12]
POST   /v1/learning/questions                   # Create question (publisher)
GET    /v1/learning/questions                   # List/filter questions
POST   /v1/learning/quizzes                     # Create quiz definition (publisher)
GET    /v1/learning/quizzes/:id                 # Get quiz definition
POST   /v1/learning/students/:id/quiz-sessions  # Start quiz session (family-scoped)
PATCH  /v1/learning/students/:id/quiz-sessions/:sid  # Save progress / submit
POST   /v1/learning/sequences                   # Create sequence (publisher)
GET    /v1/learning/sequences/:id               # Get sequence with items
POST   /v1/learning/students/:id/sequence-progress  # Start sequence
PATCH  /v1/learning/students/:id/sequence-progress/:pid  # Advance/skip
POST   /v1/learning/students/:id/assignments    # Assign content (parent)
GET    /v1/learning/students/:id/assignments    # List assignments

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

# Planning [S§17]
GET    /v1/planning/calendar                    # Get calendar view (date range, aggregated)
POST   /v1/planning/schedule-items              # Create schedule item
GET    /v1/planning/schedule-items              # List schedule items (filterable)
PATCH  /v1/planning/schedule-items/:id          # Update schedule item
DELETE /v1/planning/schedule-items/:id          # Delete schedule item

# Data Lifecycle [S§16.3]
POST   /v1/account/export                       # Request data export
GET    /v1/account/export/:id                   # Check export status / download
POST   /v1/account/deletion                     # Request account deletion
DELETE /v1/account/deletion                     # Cancel pending deletion (during grace period)
GET    /v1/account/sessions                     # List active sessions
DELETE /v1/account/sessions/:id                 # Revoke a specific session
DELETE /v1/account/sessions                     # Revoke all sessions ("sign out everywhere")

# Administration (RequireAdmin)
GET    /v1/admin/users                          # Search/list users
GET    /v1/admin/users/:id                      # Get user details
POST   /v1/admin/users/:id/suspend              # Suspend account
GET    /v1/admin/system/health                  # System health overview
GET    /v1/admin/system/jobs                    # Background job status & dead-letter queue
GET    /v1/admin/flags                          # List feature flags
PATCH  /v1/admin/flags/:key                     # Toggle feature flag

# Notifications [S§13]
GET    /v1/notifications                        # List notifications
PATCH  /v1/notifications/:id/read               # Mark as read
GET    /v1/notifications/preferences            # Get preferences
PATCH  /v1/notifications/preferences            # Update preferences
```

### 10.3 Pagination `[S§17.3]`

All list endpoints use cursor-based pagination for consistent performance:

```go
// internal/shared/pagination.go

type PaginationParams struct {
    Cursor *string `query:"cursor"` // opaque cursor (base64-encoded ID + timestamp)
    Limit  *int    `query:"limit"`  // default 20, max 100
}

type PaginatedResponse[T any] struct {
    Data       []T     `json:"data"`
    NextCursor *string `json:"next_cursor,omitempty"`
    HasMore    bool    `json:"has_more"`
}

// EncodeCursor encodes the last item's sort key for stable pagination.
func EncodeCursor(id uuid.UUID, createdAt time.Time) string {
    raw := fmt.Sprintf("%s:%d", id.String(), createdAt.UnixMilli())
    return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor decodes a pagination cursor.
func DecodeCursor(cursor string) (uuid.UUID, time.Time, error) {
    raw, err := base64.RawURLEncoding.DecodeString(cursor)
    if err != nil {
        return uuid.Nil, time.Time{}, fmt.Errorf("invalid cursor: %w", err)
    }
    parts := strings.SplitN(string(raw), ":", 2)
    if len(parts) != 2 {
        return uuid.Nil, time.Time{}, fmt.Errorf("invalid cursor format")
    }
    id, err := uuid.Parse(parts[0])
    if err != nil {
        return uuid.Nil, time.Time{}, fmt.Errorf("invalid cursor ID: %w", err)
    }
    ts, err := strconv.ParseInt(parts[1], 10, 64)
    if err != nil {
        return uuid.Nil, time.Time{}, fmt.Errorf("invalid cursor timestamp: %w", err)
    }
    return id, time.UnixMilli(ts), nil
}
```

### 10.4 OpenAPI & TypeScript Client Generation

```go
// internal/learn/models.go
// Using swaggo/swag for OpenAPI spec generation from Go types

// CreateActivityRequest represents a request to create a learning activity.
// @Summary Create activity
// @Tags learning
type CreateActivityRequest struct {
    Title           string    `json:"title" validate:"required"`
    Description     *string   `json:"description"`
    StudentID       uuid.UUID `json:"student_id" validate:"required"`
    SubjectTags     []string  `json:"subject_tags" validate:"required"`
    ActivityDate    string    `json:"activity_date" validate:"required"`
    DurationMinutes *int16    `json:"duration_minutes"`
}

type ActivityResponse struct {
    ID              uuid.UUID         `json:"id"`
    Title           string            `json:"title"`
    Description     *string           `json:"description"`
    StudentID       uuid.UUID         `json:"student_id"`
    StudentName     string            `json:"student_name"`
    SubjectTags     []string          `json:"subject_tags"`
    ActivityDate    string            `json:"activity_date"`
    DurationMinutes *int16            `json:"duration_minutes"`
    Attachments     []MediaAttachment `json:"attachments"`
    CreatedAt       time.Time         `json:"created_at"`
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

### 10.5 Error Response Format

```go
// internal/error/error.go

type ErrorResponse struct {
    Error ErrorBody `json:"error"`
}

type ErrorBody struct {
    Code    string      `json:"code"`              // machine-readable: "PREMIUM_REQUIRED"
    Message string      `json:"message"`           // human-readable: "This feature requires a premium subscription"
    Details interface{} `json:"details,omitempty"`  // optional structured details
}

// HTTPErrorHandler is a custom Echo error handler that maps domain errors to HTTP responses.
func HTTPErrorHandler(err error, c echo.Context) {
    var appErr *AppError
    if errors.As(err, &appErr) {
        status, code, message := appErr.HTTPMapping()
        _ = c.JSON(status, ErrorResponse{
            Error: ErrorBody{Code: code, Message: message},
        })
        return
    }

    var echoErr *echo.HTTPError
    if errors.As(err, &echoErr) {
        msg, _ := echoErr.Message.(string)
        _ = c.JSON(echoErr.Code, ErrorResponse{
            Error: ErrorBody{Code: "ERROR", Message: msg},
        })
        return
    }

    slog.Error("Internal error", "error", err)
    _ = c.JSON(http.StatusInternalServerError, ErrorResponse{
        Error: ErrorBody{Code: "INTERNAL", Message: "an internal error occurred"},
    })
}
```

### 10.6 Rate Limiting `[S§2.3]`

```go
// internal/middleware/rate_limit.go

type RateLimitConfig struct {
    Default   RateLimit // 100 req/min
    Auth      RateLimit // 10 req/min (login attempts)
    Upload    RateLimit // 20 req/min
    Search    RateLimit // 60 req/min
    Messaging RateLimit // 30 req/min
}

type RateLimit struct {
    Requests int
    Window   time.Duration
}
```

### 10.7 API Evolution & Versioning `[S§17.11]`

All endpoints are under `/v1`. The versioning strategy is designed for a monolith that will eventually serve both the first-party SPA and a mobile app (Phase 3+).

**Additive (non-breaking) changes** — deploy freely to `/v1`:
- New optional fields in response bodies
- New optional query parameters
- New endpoints
- New enum variants in response fields (consumers MUST handle unknown variants gracefully)

**Breaking changes** — require a new version prefix (`/v2`):
- Removing or renaming a response field
- Changing a field type
- Changing a field from optional to required
- Changing the semantics of an existing value

**Deprecation workflow**:

```go
// internal/middleware/deprecation.go

// DeprecationHeaders is Echo middleware that adds deprecation headers to sunset endpoints.
func DeprecationHeaders(sunsetDate string, successorURL string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            err := next(c)
            // Add Sunset header per RFC 8594
            c.Response().Header().Set("Deprecation", "true")
            c.Response().Header().Set("Sunset", sunsetDate)
            c.Response().Header().Set("Link", fmt.Sprintf("<%s>; rel=\"successor-version\"", successorURL))
            return err
        }
    }
}
```

**Revision trigger**: Implement `/v2` prefix routing when the first breaking change is needed (likely Phase 3 with mobile app launch).

### 10.8 Idempotency `[S§17.10]`

Critical state-changing operations support idempotency to ensure safe retries:

```go
// internal/middleware/idempotency.go

// IdempotencyLayer stores and replays responses for idempotent endpoints.
// Client sends `Idempotency-Key: <uuid>` header.
// Server stores the response and returns the cached response on replay.
type IdempotencyLayer struct {
    cache shared.Cache
    ttl   time.Duration // 24 hours default
}

// Idempotency-protected endpoints:
// - POST /v1/marketplace/checkout        (payment)
// - POST /v1/billing/subscriptions       (subscription creation)
// - POST /v1/marketplace/payouts         (creator payout)
// - POST /v1/learning/students/:id/quiz-sessions  (quiz start)
```

The idempotency key is stored in Redis with a 24-hour TTL. If a request arrives with a key that already has a stored response, the stored response is returned without re-executing the handler. If a request is in-flight (key exists but no response yet), the server returns `409 Conflict` to prevent concurrent duplicate execution.

---

## 11. Frontend Architecture

This section is unchanged from the original — the frontend is React/TypeScript and is not affected by the backend language migration. See the full frontend architecture (§11.1 through §11.6) in the spec for: React SPA structure, state management (TanStack Query), routing (React Router v7), WebSocket integration, accessibility (WCAG 2.1 AA), and internationalization readiness.

### 11.1 React SPA Structure

The frontend mirrors the backend's domain structure for clear ownership and navigation:

```
frontend/
├── src/
│   ├── main.tsx
│   ├── App.tsx                    # Root layout + router
│   ├── api/
│   │   ├── client.ts              # Fetch wrapper with auth cookie
│   │   └── generated/             # Auto-generated from OpenAPI [§10.4]
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
│   │   │   ├── ProgressView.tsx
│   │   │   ├── QuizPlayer.tsx         # Interactive quiz taking [S§8.1.9]
│   │   │   ├── VideoPlayer.tsx        # HLS + external video [S§8.1.11]
│   │   │   ├── ContentViewer.tsx      # In-platform PDF/document viewer [S§8.1.10]
│   │   │   └── SequenceView.tsx       # Lesson sequence progression [S§8.1.12]
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
│   │   ├── search/                # Global search [S§14]
│   │   │   └── SearchResults.tsx
│   │   └── student/               # Supervised student views [S§8.6]
│   │       ├── StudentDashboard.tsx
│   │       ├── StudentQuiz.tsx
│   │       ├── StudentVideo.tsx
│   │       ├── StudentReader.tsx
│   │       └── StudentSequence.tsx
│   ├── hooks/
│   │   ├── useAuth.ts             # AuthContext consumer
│   │   ├── useFamily.ts           # Family data + methodology context
│   │   ├── useWebSocket.ts        # Real-time connection
│   │   ├── useMethodologyTools.ts # Active tools for current family
│   │   └── useStudentSession.ts  # Student session context [S§8.6]
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

### 11.2 State Management

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

### 11.3 Routing

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
      { path: 'learning/quiz/:sessionId', element: <QuizPlayer /> },
      { path: 'learning/video/:videoId', element: <VideoPlayer /> },
      { path: 'learning/read/:contentId', element: <ContentViewer /> },
      { path: 'learning/sequence/:progressId', element: <SequenceView /> },

      // Marketplace [S§9]
      { path: 'marketplace', element: <MarketplaceBrowse /> },
      { path: 'marketplace/listings/:id', element: <ListingDetail /> },
      { path: 'marketplace/cart', element: <Cart /> },
      { path: 'marketplace/purchases', element: <PurchaseHistory /> },

      // Creator [S§9.1]
      { path: 'creator', element: <CreatorDashboard /> },
      { path: 'creator/listings/new', element: <CreateListing /> },
      { path: 'creator/listings/:id/edit', element: <EditListing /> },
      { path: 'creator/quiz-builder', element: <QuizBuilder /> },
      { path: 'creator/quiz-builder/:id', element: <QuizBuilder /> },
      { path: 'creator/sequence-builder', element: <SequenceBuilder /> },
      { path: 'creator/sequence-builder/:id', element: <SequenceBuilder /> },

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

  // Supervised Student Views [S§8.6]
  // Simplified layout — no social, marketplace, or messaging access
  {
    path: '/student',
    element: <StudentShell />,
    children: [
      { index: true, element: <StudentDashboard /> },
      { path: 'quiz/:sessionId', element: <StudentQuiz /> },
      { path: 'video/:videoId', element: <StudentVideo /> },
      { path: 'read/:contentId', element: <StudentReader /> },
      { path: 'sequence/:progressId', element: <StudentSequence /> },
    ],
  },
]);
```

### 11.4 WebSocket Integration `[S§7.5, S§13]`

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

### 11.5 Accessibility `[S§17.6]`

WCAG 2.1 Level AA compliance is enforced through design, implementation, and CI:

**Design patterns**:

- **Semantic HTML** — All interactive elements use proper ARIA roles and labels. Prefer native HTML elements (`<button>`, `<select>`, `<dialog>`) over custom ARIA widgets.
- **Keyboard navigation** — Every interactive element is reachable via Tab, operable via Enter/Space. Custom components (date pickers, dropdowns, drag-and-drop) follow WAI-ARIA Authoring Practices.
- **Focus management** — Route transitions move focus to the main content heading. Modals trap focus until dismissed via Escape. Toast notifications do NOT steal focus.
- **Color contrast** — Tailwind config enforces minimum 4.5:1 contrast ratio for text. Design tokens include a `--color-focus-ring` variable for consistent, high-visibility focus indicators.
- **Screen reader support** — All images have alt text; decorative images use `alt=""`. Dynamic content (feed updates, quiz feedback, notification toasts) uses `aria-live` regions with appropriate politeness levels.
- **Skip navigation** — Every page includes a visually hidden "Skip to main content" link as the first focusable element.
- **Touch targets** — All interactive elements have a minimum tap target of 44x44 CSS pixels on viewports under 768px.

**CI enforcement**:

```yaml
# In GitHub Actions CI pipeline
- name: Accessibility audit
  run: |
    # axe-core integration tests (zero critical/serious violations)
    npx playwright test --project=a11y
    # HTML validation (no duplicate IDs, correct ARIA usage)
    npx html-validate dist/**/*.html
```

```typescript
// tests/a11y/global-a11y.spec.ts
// Every page route is tested for axe violations
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const routes = ['/dashboard', '/learning', '/social', '/marketplace', '/settings'];

for (const route of routes) {
  test(`${route} has no a11y violations`, async ({ page }) => {
    await page.goto(route);
    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21aa'])
      .analyze();
    expect(results.violations).toEqual([]);
  });
}
```

**Screen reader test matrix** (manual, before major releases):

| Screen Reader | Browser | Platform |
|---------------|---------|----------|
| NVDA | Chrome | Windows |
| VoiceOver | Safari | macOS |
| VoiceOver | Safari | iOS |
| TalkBack | Chrome | Android |

### 11.6 Internationalization Readiness `[S§17.7]`

All user-facing strings are externalized from day one, even though Phase 1 is US-only:

```typescript
// i18n setup using react-intl (or similar)
// All strings are keys, not hardcoded English
<FormattedMessage id="learning.activity.create" defaultMessage="Log Activity" />
<FormattedMessage id="social.friends.request" defaultMessage="Send Friend Request" />
```

Date, time, and number formatting use `Intl` APIs with locale-aware formatting.

---

## 12. Background Processing

### 12.1 Job Queue Architecture `[S§13, S§12]`

Redis-backed job queue using `hibiken/asynq` with three priority tiers:

```go
// internal/shared/jobs.go

// Job priority queues (asynq)
const (
    QueueCritical = "critical" // Safety-critical: CSAM reports, account suspensions, security alerts. Target: process within 30 seconds
    QueueDefault  = "default"  // Standard: email delivery, notification dispatch, search indexing. Target: process within 5 minutes
    QueueLow      = "low"      // Bulk/deferrable: digest compilation, analytics aggregation, re-scans. Target: process within 1 hour
)
```

### 12.2 Key Jobs by Domain

| Domain | Job | Queue | Description |
|--------|-----|-------|-------------|
| **Trust & Safety** | `CsamReportJob` | Critical | Report CSAM to NCMEC, suspend account `[S§12.1]` |
| **Trust & Safety** | `ContentModerationJob` | Critical | Process flagged content `[S§12.2]` |
| **Notifications** | `SendEmailJob` | Default | Deliver transactional email via Postmark `[S§13.2]` |
| **Notifications** | `PushNotificationJob` | Default | Deliver in-app notification `[S§13.1]` |
| **Social** | `FanOutPostJob` | Default | Fan-out new post to friends' feeds `[S§7.2]` |
| **Media** | `ProcessImageJob` | Default | Resize images, generate thumbnails `[§8.2]` |
| **Media** | `TranscodeVideoJob` | Default | Convert raw video to HLS adaptive bitrate (480p/720p/1080p) `[S§8.1.11]` |
| **Media** | `CsamScanJob` | Default | Scan uploaded media via Thorn Safer `[S§12.1]` |
| **Search** | `IndexContentJob` | Default | Update search indexes on content change `[S§14]` |
| **Marketplace** | `ProcessPayoutJob` | Default | Calculate and initiate creator payouts `[S§9.6]` |
| **Notifications** | `CompileDigestJob` | Low | Build daily/weekly email digests `[S§13.3]` |
| **Trust & Safety** | `CheckCsamHashUpdateJob` | Low | Check for new CSAM hash databases; trigger rescan if updated `[S§12.1, 11-safety §10.7]` |
| **Learning** | `ProgressAggregationJob` | Low | Aggregate progress metrics per student `[S§8.1.7]` |
| **Billing** | `SubscriptionRenewalCheckJob` | Low | Check upcoming renewals, send reminders `[S§15.3]` |

### 12.3 Recurring Schedule

```go
// internal/shared/scheduler.go

// RegisterRecurringJobs sets up cron-style recurring jobs with asynq scheduler.
func RegisterRecurringJobs(scheduler *asynq.Scheduler) {
    // Daily at 6:00 AM UTC — compile and send daily digests [S§13.3]
    scheduler.Register("0 6 * * *", asynq.NewTask("compile_digest", mustMarshal(DigestParams{Type: "daily"})))

    // Weekly on Mondays at 6:00 AM UTC — weekly digests [S§13.3]
    scheduler.Register("0 6 * * 1", asynq.NewTask("compile_digest", mustMarshal(DigestParams{Type: "weekly"})))

    // Daily at 3:00 AM UTC — check for CSAM hash database updates [S§12.1, 11-safety §10.7]
    // Triggers CsamRescanJob only when new hashes are available (event-driven, not blanket rescan)
    scheduler.Register("0 3 * * *", asynq.NewTask("check_csam_hash_update", nil))

    // Hourly — aggregate progress metrics [S§8.1.7]
    scheduler.Register("0 * * * *", asynq.NewTask("progress_aggregation", nil))

    // Daily at 2:00 AM UTC — check subscription renewals [S§15.3]
    scheduler.Register("0 2 * * *", asynq.NewTask("subscription_renewal_check", nil))
}
```

### 12.4 Social Feed Fan-Out `[S§7.2]`

The feed uses a **fan-out-on-write** pattern via Redis sorted sets:

```go
// internal/social/jobs.go

// FanOutPost fans out a new post to all friends' feeds.
func FanOutPost(ctx context.Context, cache shared.Cache, db *gorm.DB, post *Post) error {
    // Get all accepted friends of the post author [S§7.4]
    friendFamilyIDs, err := getFriendIDs(ctx, db, post.FamilyID)
    if err != nil {
        return fmt.Errorf("failed to get friends: %w", err)
    }

    // Add post to each friend's feed (sorted set, scored by timestamp)
    // NOTE: The social domain defines a FeedStore port for sorted set operations.
    // This example is illustrative — actual implementation uses the domain port.
    score := float64(post.CreatedAt.UnixMilli())
    member := post.ID.String()

    for _, friendID := range friendFamilyIDs {
        feedKey := fmt.Sprintf("feed:%s", friendID)
        if err := cache.ZAdd(ctx, feedKey, score, member); err != nil {
            return fmt.Errorf("feed zadd failed: %w", err)
        }
        // Trim feed to last 1000 items to bound memory
        if err := cache.ZRemRangeByRank(ctx, feedKey, 0, -1001); err != nil {
            return fmt.Errorf("feed trim failed: %w", err)
        }
    }

    // If post is in a group, also add to group feed [S§7.6]
    if post.GroupID != nil {
        groupFeedKey := fmt.Sprintf("feed:group:%s", *post.GroupID)
        if err := cache.ZAdd(ctx, groupFeedKey, score, member); err != nil {
            return fmt.Errorf("group feed zadd failed: %w", err)
        }
        if err := cache.ZRemRangeByRank(ctx, groupFeedKey, 0, -1001); err != nil {
            return fmt.Errorf("group feed trim failed: %w", err)
        }
    }

    return nil
}

// GetFeed reads a user's feed. [S§7.2.3]
func GetFeed(ctx context.Context, cache shared.Cache, db *gorm.DB, familyID uuid.UUID, cursor *float64, limit int64) ([]Post, error) {
    feedKey := fmt.Sprintf("feed:%s", familyID)

    maxScore := "+inf"
    if cursor != nil {
        maxScore = fmt.Sprintf("%f", *cursor)
    }

    // Get post IDs from cache sorted set (reverse chronological) [S§7.2.3]
    postIDs, err := cache.ZRevRangeByScore(ctx, feedKey, "0", maxScore, limit)
    if err != nil {
        return nil, fmt.Errorf("feed query failed: %w", err)
    }

    // Hydrate post data from PostgreSQL
    postUUIDs := make([]uuid.UUID, 0, len(postIDs))
    for _, id := range postIDs {
        uid, err := uuid.Parse(id)
        if err != nil {
            continue
        }
        postUUIDs = append(postUUIDs, uid)
    }

    return findPostsByIDs(ctx, db, postUUIDs)
}
```

### 12.5 Error Recovery & Resilience `[S§17.10]`

**Job retry policy**:

| Queue | Max Retries | Backoff | Dead-Letter |
|-------|-------------|---------|-------------|
| Critical (CSAM, suspensions) | Unlimited (alert after 5) | 30s, 60s, 120s, then every 5min | Never — manual escalation after alert |
| Default (email, indexing) | 3 | 60s, 300s, 900s (exponential) | Yes — `dead_letter:{queue}` Redis list |
| Low (digests, aggregation) | 3 | 300s, 900s, 3600s | Yes — `dead_letter:{queue}` Redis list |

Dead-letter jobs are surfaced in the admin dashboard (§16-admin) for manual inspection and replay.

**Circuit breaker for external services**:

```go
// internal/shared/circuit_breaker.go

// CircuitBreaker tracks failure count and transitions between states.
type CircuitBreaker struct {
    failureThreshold uint32        // trips after N consecutive failures
    recoveryTimeout  time.Duration // how long to stay open before half-open
    state            CircuitState
    mu               sync.Mutex
}

type CircuitState int

const (
    CircuitClosed   CircuitState = iota // normal operation, requests pass through
    CircuitOpen                         // circuit tripped, requests fail fast with fallback
    CircuitHalfOpen                     // testing recovery, one request allowed through
)

// Circuit breaker configuration per external service:
// Thorn Safer:     threshold=5, recovery=60s, fallback=queue for manual review
// Rekognition:     threshold=5, recovery=60s, fallback=queue for manual review
// Hyperswitch:     threshold=3, recovery=30s, fallback=return payment error to user
// Postmark:        threshold=5, recovery=120s, fallback=queue email for retry
// Kratos:          threshold=3, recovery=10s, fallback=503 maintenance mode
```

**Webhook idempotency** — all incoming webhook handlers deduplicate by event ID:

```go
// internal/shared/webhook.go

// IsDuplicateWebhook checks if a webhook event has already been processed via cache SET with TTL.
func IsDuplicateWebhook(ctx context.Context, cache shared.Cache, provider, eventID string) (bool, error) {
    key := fmt.Sprintf("webhook:seen:%s:%s", provider, eventID)
    // SetNX returns true only if the key was newly created
    isNew, err := cache.SetNX(ctx, key, "1", 7*24*time.Hour)
    if err != nil {
        return false, fmt.Errorf("cache setnx failed: %w", err)
    }
    return !isNew, nil
}
```

---

## 13. Deployment & Infrastructure

### 13.1 Docker Multi-Stage Build

```dockerfile
# Stage 1: Build Go binary
FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o homegrown-academy ./cmd/server

# Stage 2: Minimal runtime
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/homegrown-academy /usr/local/bin/
COPY --from=builder /app/migrations/ /app/migrations/
EXPOSE 3000
CMD ["homegrown-academy"]
```

### 13.2 AWS Infrastructure (Phase 1) `[S§17.5]`

```
AWS VPC (10.0.0.0/16) — us-east-1
│
├── Public Subnets
│   ├── us-east-1a (10.0.1.0/24)
│   │   ├── ALB (TLS termination via ACM)
│   │   └── NAT Gateway
│   └── us-east-1b (10.0.2.0/24)
│       └── ALB (second AZ, required by ALB)
│
├── Private Subnets
│   ├── us-east-1a (10.0.10.0/24)
│   │   ├── EC2 t4g.small (ECS-optimized AMI)
│   │   │   └── ECS Task
│   │   │       ├── homegrown-api    (Go binary, port 3000)
│   │   │       └── ory-kratos       (sidecar, port 4433/4434)
│   │   ├── RDS PostgreSQL 16       (db.t4g.medium, 20 GB gp3)
│   │   └── ElastiCache Redis 7     (cache.t4g.micro)
│   └── us-east-1b (10.0.11.0/24)
│       └── (reserved for Multi-AZ in Phase 2)
│
├── Security Groups
│   ├── sg-alb:       ingress 443 from 0.0.0.0/0
│   ├── sg-ecs:       ingress 3000 from sg-alb only
│   ├── sg-rds:       ingress 5432 from sg-ecs only
│   ├── sg-redis:     ingress 6379 from sg-ecs only
│   └── sg-ssh:       ingress 22 from admin IPs only
│
├── ECR Repository
│   └── homegrown-api (lifecycle: retain last 10 images)
│
└── Monitoring
    ├── CloudWatch (ECS, RDS, ElastiCache metrics + alarms)
    ├── Sentry SDK (in-process error tracking)
    └── UptimeRobot (external availability)
```

### 13.3 ALB Configuration + Security Headers

ALB replaces Caddy for TLS termination and request routing. Security headers move into Echo middleware since ALB does not natively inject response headers.

**ALB configuration**:

| Setting | Value |
|---------|-------|
| **Listener** | HTTPS :443, TLS certificate from ACM (auto-renewing) |
| **Target group** | ECS tasks on port 3000, HTTP health check `GET /health` |
| **Health check** | Interval 30s, healthy threshold 2, unhealthy threshold 3 |
| **Idle timeout** | 300s (supports long-lived WebSocket connections) |
| **Stickiness** | Disabled (application is stateless; sessions stored in Redis) |
| **HTTP->HTTPS** | Redirect rule on port 80 |

**Security headers** — Echo middleware `[S§17.1]`:

```go
// internal/middleware/security_headers.go

func SecurityHeaders() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            h := c.Response().Header()
            h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
            h.Set("X-Content-Type-Options", "nosniff")
            h.Set("X-Frame-Options", "DENY")
            h.Set("Content-Security-Policy",
                "default-src 'self'; "+
                    "img-src 'self' https://*.r2.cloudflarestorage.com; "+
                    "connect-src 'self' wss://api.homegrown.academy")
            h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
            return next(c)
        }
    }
}
```

### 13.4 Scaling Path

```
Phase 1 (MVP) ~$110/mo          Phase 2 ~$250/mo                  Phase 3 ~$500-800/mo
──────────────────────           ────────────────                  ────────────────────
Single-AZ, 1 instance           Multi-AZ, auto-scaling            Managed + dedicated

┌──────────────┐                ┌──────────────┐                  ┌──────────────┐
│  ALB         │                │  ALB         │                  │  ALB         │
├──────────────┤                ├──────────────┤                  ├──────────────┤
│  1x EC2      │                │  ECS Service │                  │  ECS Fargate │
│  t4g.small   │       →        │  2-4 tasks   │        →         │  auto-scale  │
│  (ECS task:  │                │  (EC2 or     │                  ├──────────────┤
│   API +      │                │   Fargate)   │                  │  Aurora PG   │
│   Kratos)    │                ├──────────────┤                  │  + replicas  │
├──────────────┤                │  RDS Multi-AZ│                  ├──────────────┤
│  RDS Single  │                │  (auto       │                  │  ElastiCache │
│  -AZ         │                │   failover)  │                  │  cluster     │
├──────────────┤                ├──────────────┤                  ├──────────────┤
│  ElastiCache │                │  ElastiCache │                  │  Typesense   │
│  single node │                │  single node │                  │  (ECS task)  │
└──────────────┘                └──────────────┘                  └──────────────┘
```

### 13.5 Backup & Disaster Recovery Strategy `[S§17.5]`

#### Recovery Objectives

| Metric | Phase 1 Target | Phase 2+ Target |
|--------|---------------|-----------------|
| **RPO** (Recovery Point Objective) | <= 5 minutes (RDS continuous WAL) | <= 1 minute (cross-region replica) |
| **RTO** (Recovery Time Objective) | <= 2 hours (manual restore) | <= 30 minutes (automated failover) |

RPO defines the maximum acceptable data loss window. RTO defines the maximum acceptable downtime. Phase 1 targets are achievable with single-AZ RDS and manual intervention. Phase 2 targets require cross-region replication and automated failover.

#### Backup Components

| Component | Method | Retention | Recovery |
|-----------|--------|-----------|----------|
| **RDS PostgreSQL** | Automated daily snapshots + continuous WAL archiving | 30 days | Point-in-Time Recovery (PITR) to any second within retention window |
| **Off-site backup** | Weekly `pg_dump` via ECS Scheduled Task -> Cloudflare R2 | 12 weekly, 12 monthly | Full database restore from portable dump file |
| **ElastiCache Redis** | Automatic daily snapshots | 7 days | Restore to new cache node |
| **Redis data** | Ephemeral by design (cache, feeds, rate limits) | N/A | Feeds rebuild from PostgreSQL; cache repopulates on demand |
| **Cloudflare R2** (media files) | Cloudflare-managed replication (11 nines durability) | Indefinite (lifecycle rules for temp uploads) | Self-healing; R2 is multi-region by default |

**Off-site backup task** (ECS Scheduled Task, weekly):

```bash
# Runs as a one-off ECS task on a cron schedule
# Connects to RDS, dumps to R2 for provider-independent backup
pg_dump -Fc -h $RDS_ENDPOINT -U $DB_USER homegrown_academy | \
    aws s3 cp - s3://homegrown-backups/postgresql/pg_$(date +%Y%m%d).dump \
    --endpoint-url https://${R2_ACCOUNT_ID}.r2.cloudflarestorage.com

# R2 lifecycle rules handle retention cleanup
```

#### Disaster Recovery Plan

**Phase 1 (single-AZ)**:

- **Database failure**: RDS automated failover restores from latest snapshot + WAL replay. Manual verification required.
- **Application failure**: ECS redeploys from the latest container image. Stateless application — no data loss.
- **AZ failure**: RDS PITR to a new AZ. ECS service recreated via CDK in the surviving AZ. Expected downtime: 1-2 hours.
- **Region failure**: Restore from R2 off-site backup to a new region. Expected downtime: 4-8 hours (manual process). Acceptable risk for Phase 1 (pre-revenue, no paying customers).

**Phase 2+ (multi-AZ, cross-region read replica)**:

- **Database failure**: Multi-AZ RDS automated failover (< 2 min).
- **AZ failure**: ECS runs across multiple AZs; automatic rebalancing. Zero downtime.
- **Region failure**: Promote cross-region read replica to primary. Update DNS. Expected downtime: 15-30 minutes.

**Backup verification**: The off-site backup task MUST include a `pg_restore --list` verification step that confirms the dump is valid. A failed verification MUST trigger an ErrorReporter alert.

**Revision trigger**: Add cross-region RDS read replica and multi-AZ ECS deployment when the platform handles paying customers (Phase 2).

### 13.6 CI/CD Pipeline (GitHub Actions) `[S§17.1]`

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

      - name: Go setup
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: latest

      - name: Test
        run: go test ./...

      - name: Security audit
        run: go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...

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
    permissions:
      id-token: write   # OIDC token for AWS auth
      contents: read
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials (OIDC)
        uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: ${{ secrets.AWS_DEPLOY_ROLE_ARN }}
          aws-region: us-east-1

      - name: Login to Amazon ECR
        id: ecr-login
        uses: aws-actions/amazon-ecr-login@v2

      - name: Build and push Docker image
        env:
          ECR_REGISTRY: ${{ steps.ecr-login.outputs.registry }}
          IMAGE_TAG: ${{ github.sha }}
        run: |
          docker build -t $ECR_REGISTRY/homegrown-api:$IMAGE_TAG .
          docker push $ECR_REGISTRY/homegrown-api:$IMAGE_TAG

      - name: Run database migrations
        env:
          IMAGE_TAG: ${{ github.sha }}
        run: |
          # Run migrations as a one-off ECS task before deploying
          aws ecs run-task \
            --cluster homegrown \
            --task-definition homegrown-migrate \
            --overrides '{"containerOverrides":[{"name":"migrate","command":["goose","-dir","/app/migrations","up"]}]}' \
            --network-configuration "${{ secrets.ECS_NETWORK_CONFIG }}" \
            --launch-type EC2

      - name: Update ECS service (rolling deploy)
        run: |
          aws ecs update-service \
            --cluster homegrown \
            --service homegrown-api \
            --force-new-deployment
          aws ecs wait services-stable \
            --cluster homegrown \
            --services homegrown-api

      - name: Build and deploy frontend
        run: |
          cd frontend && npm ci && npm run build
          npx wrangler pages deploy dist --project-name=homegrown-app
```

### 13.7 Infrastructure as Code (AWS CDK)

All AWS resources described in §13.2 are provisioned via a single AWS CDK (TypeScript) stack. See §2.17 for the technology selection rationale.

**Project structure**:

```
infra/
├── bin/infra.ts                    # CDK app entry point
├── lib/
│   ├── homegrown-stack.ts          # Main stack — composes all constructs
│   ├── config.ts                   # Typed config interface + defaults
│   └── constructs/
│       ├── networking.ts           # VPC, subnets, NAT gateway, security groups
│       ├── database.ts             # RDS PostgreSQL, Secrets Manager
│       ├── cache.ts                # ElastiCache Redis
│       ├── container-registry.ts   # ECR + lifecycle policy
│       ├── load-balancer.ts        # ALB, ACM certificate, target group
│       ├── compute.ts              # ECS cluster, EC2 capacity, task def, service
│       ├── monitoring.ts           # CloudWatch log groups + alarms
│       ├── ci-cd.ts                # GitHub Actions OIDC provider + deploy role
│       └── backup.ts              # ECS scheduled task for weekly pg_dump -> R2
├── cdk.json
├── tsconfig.json
└── package.json
```

**Stack organization** — single stack with 9 CDK Constructs:

| Construct | File | Provisions | Key Outputs |
|-----------|------|------------|-------------|
| Networking | `networking.ts` | VPC (10.0.0.0/16), 2 public + 2 private subnets, NAT gateway, 5 security groups (§13.2) | `vpc`, `sgAlb`, `sgEcs`, `sgRds`, `sgRedis`, `sgSsh` |
| Database | `database.ts` | RDS PostgreSQL 16 (db.t4g.medium, 20GB gp3, Single-AZ), auto-generated credentials in Secrets Manager | `dbInstance`, `dbSecret`, `dbEndpoint` |
| Cache | `cache.ts` | ElastiCache Redis 7 (cache.t4g.micro, 7-day snapshots) | `redisEndpoint` |
| ContainerRegistry | `container-registry.ts` | ECR repository, lifecycle policy (retain last 10 images) | `repository` |
| LoadBalancer | `load-balancer.ts` | ALB, ACM certificate, HTTPS listener (TLS 1.3), HTTP->HTTPS redirect, target group with health checks | `alb`, `targetGroup` |
| Compute | `compute.ts` | ECS cluster, EC2 ASG (t4g.small, ECS-optimized ARM AMI), capacity provider, task definition (API + Kratos sidecar), ECS service, IAM task roles | `cluster`, `service` |
| Monitoring | `monitoring.ts` | CloudWatch log groups (API, Kratos, backup), alarms (CPU, memory, RDS, Redis, ALB 5xx) | log group refs |
| CiCd | `ci-cd.ts` | GitHub Actions OIDC provider, IAM deploy role scoped to `repo:org/repo:ref:refs/heads/main` | `deployRoleArn` |
| Backup | `backup.ts` | ECS scheduled task (weekly `pg_dump` -> R2), EventBridge cron rule | -- |

**Dependency graph** (composition order in `homegrown-stack.ts`):

```
Networking ──┬──→ Database
             ├──→ Cache
             ├──→ LoadBalancer ──┐
             │                   ├──→ Compute ──┬──→ CiCd
ContainerRegistry ───────────────┘              └──→ Backup
Monitoring ──────────────────────────────────────┘
```

**Configuration** — typed `HomegrownConfig` interface in `config.ts` centralizing all tunable parameters:

| Parameter | Default | Purpose |
|-----------|---------|---------|
| `projectName` | `"homegrown"` | Resource naming prefix |
| `region` | `"us-east-1"` | AWS region |
| `vpcCidr` | `"10.0.0.0/16"` | VPC CIDR block |
| `adminSshCidrs` | `[]` | IP ranges for SSH access |
| `domainName` | `"api.homegrown.academy"` | ACM certificate domain |
| `certificateArn` | `undefined` | Skip ACM creation if pre-existing cert |
| `ec2InstanceType` | `"t4g.small"` | ECS EC2 instance type |
| `rdsInstanceType` | `"db.t4g.medium"` | RDS instance type |
| `redisNodeType` | `"cache.t4g.micro"` | ElastiCache node type |
| `githubOrg` / `githubRepo` | -- | OIDC trust scope |
| `placeholderMode` | `true` | Use nginx:alpine until first real image push |

Overrides provided via CDK context (`cdk.json` or `--context` flag).

**Secrets management**:

| Secret | Storage | Injection |
|--------|---------|-----------|
| RDS credentials | Secrets Manager (auto-generated by CDK) | `ecs.Secret.fromSecretsManager()` |
| Kratos cookie/cipher | Secrets Manager (values set manually post-deploy) | `ecs.Secret.fromSecretsManager()` |
| OIDC client secrets | Secrets Manager (values set manually post-deploy) | Kratos container env |
| R2 credentials | Secrets Manager (values set manually post-deploy) | Backup task container env |

**Deploy workflow**:

```
1. npm install              # Install CDK dependencies
2. cdk bootstrap            # One-time: create CDKToolkit stack in AWS account
3. cdk synth                # Validate — generates CloudFormation template without deploying
4. cdk deploy               # Provision all resources (~15-20 min first deploy)
5. Add ACM CNAME to DNS     # Manual: copy CNAME from deploy output -> Cloudflare DNS
6. Note stack outputs       # ALB DNS name, ECR URI, deploy role ARN, RDS/Redis endpoints
7. Configure GitHub secrets # AWS_DEPLOY_ROLE_ARN from stack output
```

**Stack outputs** (CloudFormation):

| Output | Used By |
|--------|---------|
| `AlbDnsName` | DNS CNAME target for `api.homegrown.academy` |
| `EcrRepositoryUri` | CI/CD pipeline `docker push` target |
| `GithubDeployRoleArn` | GitHub Actions `AWS_DEPLOY_ROLE_ARN` secret |
| `RdsEndpoint` | Reference / debugging |
| `RedisEndpoint` | Reference / debugging |

**Key patterns**:
- Stateful resources (RDS, ECR) use `removalPolicy: RETAIN` + `deletionProtection: true` — `cdk destroy` won't delete data
- All container secrets injected via Secrets Manager, never plain environment variables
- ECS service uses `minHealthyPercent: 0` for single-instance rolling deploy (brief downtime acceptable in Phase 1)
- Kratos configured via environment variables (all YAML keys map to `SCREAMING_SNAKE_CASE` env vars)
- ElastiCache uses L1 constructs (`CfnCacheCluster`) since no L2 exists in CDK

**Revision trigger**: Split into multiple stacks if independent deployment cadences are needed (e.g., update ALB rules without touching database). Current single-stack approach is sufficient through Phase 2.

---

## 14. Security Architecture

### 14.1 Network Security `[S§17.1]`

| Layer | Control |
|-------|---------|
| **VPC isolation** | Compute (ECS) and data (RDS, ElastiCache) run in private subnets with no direct internet access. Only ALB is in public subnets |
| **Security Groups** | Least-privilege ingress: ALB accepts 443 from internet -> ECS accepts 3000 from ALB only -> RDS/Redis accept connections from ECS only (§13.2) |
| **TLS** | ALB terminates TLS 1.3 via ACM-managed certificates (auto-renewing, no manual cert management) `[S§17.1]` |
| **CSP** | Content Security Policy set in Echo middleware (§13.3) — restricts script sources, image sources, connection targets |
| **CORS** | Strict origin allowlist: `app.homegrown.academy`, `homegrown.academy` |
| **SSH** | Restricted to admin IPs via Security Group. Key-only authentication on EC2 instances |
| **Audit** | CloudTrail logs all AWS API calls for security audit trail |

### 14.2 Application Security (OWASP Top 10) `[S§17.1]`

| OWASP Risk | Go/Echo Mitigation |
|------------|---------------------|
| **A01: Broken Access Control** | Family-scoped queries via `FamilyScope` helper (§6.2). Every handler receives `AuthContext` with verified `family_id`. Permission middleware enforces role checks. |
| **A02: Cryptographic Failures** | Passwords handled by Ory Kratos (Argon2id). Sensitive data encrypted at rest via PostgreSQL `pgcrypto`. All transit encrypted via TLS. |
| **A03: Injection** | GORM parameterized queries prevent SQL injection. Go's type system prevents most injection vectors. User input validated via struct binding with `validator` tags. |
| **A04: Insecure Design** | Privacy-by-architecture (§1.5). COPPA consent state machine (§6.3). No public user content by design. |
| **A05: Security Misconfiguration** | Docker containers run as non-root. Minimal base images (distroless). Security headers set in Echo middleware (§13.3). ECS-optimized AMI managed by AWS. |
| **A06: Vulnerable Components** | `govulncheck` + `npm audit` in CI pipeline. Dependabot for automated dependency updates. |
| **A07: Auth Failures** | Ory Kratos handles auth — battle-tested implementation. MFA encouraged. Rate-limited login attempts (10/min). Session timeout (30 days, revocable). |
| **A08: Data Integrity** | All API inputs validated via typed Go structs with `validator` tags. CSRF protection via SameSite cookies. Webhook signatures verified (Stripe, Kratos). |
| **A09: Logging Failures** | Security events logged via `slog`. Audit logging for admin actions `[S§2.3]`. Logs shipped to Sentry. |
| **A10: SSRF** | No user-controlled URL fetching in backend. R2 presigned URLs are generated server-side with controlled parameters. |

### 14.3 Go's Safety Advantages

Go eliminates or mitigates several categories of vulnerabilities:

- **Garbage collection** eliminates use-after-free and most memory leaks.
- **Built-in race detector** (`go test -race`) catches data races at test time.
- **No pointer arithmetic** — pointer operations are bounded and type-safe.
- **Bounds-checked slice access** — out-of-range access panics rather than corrupting memory.
- **`nil` handling is explicit** — check before dereference; linters enforce nil checks.
- **Strong typing** prevents type confusion vulnerabilities.

These properties are valuable for a platform handling children's data `[S§17.2]` — memory safety vulnerabilities are among the most exploited attack vectors, and Go's garbage collector and type system make the most dangerous classes structurally impossible.

### 14.4 Data Encryption `[S§17.1]`

| Data | At Rest | In Transit |
|------|---------|------------|
| **Credentials** | Argon2id (Ory Kratos) | TLS 1.3 |
| **Payment data** | Never stored (Stripe handles) | TLS 1.3 to Stripe |
| **PII** | PostgreSQL `pgcrypto` for sensitive fields | TLS 1.3 |
| **Session tokens** | Kratos encrypted sessions | HTTPS-only SameSite cookies |
| **Media files** | R2 server-side encryption | TLS 1.3 to/from R2 |
| **Backups** | Encrypted R2 storage | TLS 1.3 during transfer |

### 14.5 COPPA Security Controls `[S§17.2]`

| Control | Implementation |
|---------|---------------|
| **Data minimization** | Student profiles collect only name, birth year, grade. No email, no credentials, no social profile. `[S§3.1.3]` |
| **Parental consent** | COPPA consent state machine (§6.3) required before creating student profiles |
| **Data access** | Parents have complete visibility into all student data `[S§3.3]` |
| **Data deletion** | Parents can request deletion of child data; processed within COPPA's required timeframe `[S§16.3]` |
| **Third-party sharing** | Student data never shared with third parties except CSAM reporting (legal requirement) |
| **No direct contact** | Platform prohibits unmediated adult-student communication `[S§3.3]` |

### 14.6 Dependency Security

```bash
# Automated via CI/CD (§13.6)
govulncheck ./...              # Go dependency vulnerabilities
npm audit --production         # Node.js dependency vulnerabilities

# GitHub Dependabot configuration
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: gomod
    directory: "/"
    schedule:
      interval: weekly
  - package-ecosystem: npm
    directory: "/frontend"
    schedule:
      interval: weekly
```

---

## 15. Phase 1 (MVP) Scope `[S§19]`

This section maps each Phase 1 domain from `[S§19]` to concrete implementation artifacts. The content is identical to the original spec — domain scopes, component tables, API endpoint counts, and React component lists are unchanged. The backend implementation language changes from Rust to Go but the functional scope remains the same.

Refer to the original Phase 1 scope for full details on: Identity & Access (§15.1), Methodology (§15.2), Discovery (§15.3), Onboarding (§15.4), Social (§15.5), Learning (§15.6), Marketplace (§15.7), Trust & Safety (§15.8), Billing (§15.9), Notifications (§15.10), Search (§15.11), Content & Media (§15.12), and Privacy/Compliance (§15.13).

Key note for §15.6 Learning: Video transcoding is a background job in the existing asynq queue. All other Phase 1 scope details are unchanged.

---

## 16. Architecture Decision Records

### ADR-001: Go over Rust for Backend

**Status**: Accepted (supersedes original)

**Context**: The backend must serve a 17-domain application with sub-300ms API responses `[S§17.3]`, scale to 100K concurrent users `[S§17.3]`, handle sensitive children's data `[S§17.2]`, and be primarily written by AI. The project is built by a solo developer + Claude across 17 domains, 80-100 tables, and ~150 endpoints.

**Decision**: Use Go (Echo + GORM) instead of Rust (Axum + SeaORM).

**Consequences**:
- **Positive**: Fast iteration — Go compiles in 1-3 seconds vs. Rust's 30-90 seconds. Simple concurrency model (goroutines). Large ecosystem of production-grade libraries. GORM provides automatic CRUD for 80-100+ tables. Go's simplicity reduces cognitive overhead for AI-generated code. Sufficient performance (20-40K req/sec with DB).
- **Negative**: No borrow checker (use race detector instead via `go test -race`). Slightly lower raw throughput than Rust. No compile-time null safety (use linting and explicit nil checks).
- **Mitigation**: AI generates idiomatic Go code. Race detector catches concurrency bugs in tests. `golangci-lint` enforces code quality. Performance is more than sufficient for the target scale.

**Revision trigger**: Never. Go is the long-term choice.

### ADR-002: Monolith Architecture

**Status**: Accepted

**Context**: 17 spec domains `[S§2.1]` could be individual microservices. However, a solo developer maintains the system, all domains share the same database, and inter-domain communication is frequent `[S§18]`.

**Decision**: Single Go binary with domain packages. No microservices.

**Consequences**:
- **Positive**: Single deployment unit. No inter-service networking complexity. Shared database with referential integrity. Simpler debugging and tracing. Lower infrastructure cost.
- **Negative**: All domains scale together (cannot independently scale Search vs. Social). Single point of failure. Larger binary size.
- **Mitigation**: Go's efficiency makes "scale together" viable to 100K+ concurrent users on modest hardware. Health checks and graceful degradation `[S§17.5]` mitigate single-point-of-failure risk.

**Revision trigger**: Extract a domain as a separate service when it accounts for >40% of system resources or has fundamentally different deployment cadence needs.

### ADR-003: PostgreSQL as Infrastructure

**Status**: Accepted

**Context**: The platform needs: relational data (families, friendships, purchases), document storage (methodology config), location queries (nearby families), full-text search, and background job metadata. Using separate services for each would multiply operational complexity.

**Decision**: Use PostgreSQL as the primary datastore for relational data, JSONB documents, PostGIS location queries, and full-text search (Phase 1). Redis supplements for caching, job queues, and feed fan-out.

**Consequences**:
- **Positive**: Single database to operate, back up, and monitor. PostgreSQL's JSONB is performant enough for methodology config. PostGIS eliminates a separate geo service. FTS eliminates a search service for Phase 1. Strong consistency guarantees.
- **Negative**: PostgreSQL FTS lacks typo tolerance and instant search capabilities that Typesense provides. JSONB queries are less optimized than purpose-built document stores.
- **Mitigation**: Typesense migration path defined (§9.2) with specific trigger (100K listings or 500ms p95 latency). JSONB data volume is small (6 methodologies x ~10KB each).

**Revision trigger**: Add Typesense when search latency exceeds 500ms p95 or marketplace exceeds 100K listings.

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

**Context**: The application has 17 domains with extensive interactivity — messaging, social feeds, learning tools, marketplace, quizzes. SEO is handled separately by Astro for public content `[S§5]`. The authenticated app does not need SEO.

**Decision**: React SPA (Vite) for the authenticated application. Astro SSG for public content.

**Consequences**:
- **Positive**: Simpler architecture — no SSR hydration complexity. Real-time features (messaging, notifications) are natural in SPA. React's ecosystem provides rich components for every domain need. Clear separation: Astro = public/SEO, React = app/interactive.
- **Negative**: Initial page load requires JavaScript bundle download. No SSR means no server-side data fetching.
- **Mitigation**: Vite code splitting keeps initial bundle small. TanStack Query prefetching can begin data loading immediately. The app is behind auth — users expect a login flow before seeing content.

**Revision trigger**: None anticipated. The two-site architecture (Astro + React) cleanly separates concerns.

### ADR-006: Hetzner over Cloud Providers

**Status**: Superseded by ADR-010

**Context**: Comparable AWS EC2 instances cost $500-2,000/mo. Go's efficiency means a single dedicated server handles Phase 1-2 workloads comfortably.

**Decision**: Hetzner dedicated server (~$60/mo) for Phase 1-2, with documented scaling path.

**Consequences**:
- **Positive**: ~90% cost savings vs. AWS/GCP for equivalent hardware. Predictable pricing. No vendor lock-in (standard Linux + Docker). Sufficient for 10K-50K concurrent users with Go.
- **Negative**: No managed services (must operate PostgreSQL, Redis yourself). No auto-scaling. Single datacenter (no multi-region).
- **Mitigation**: Automated backups (§13.5). Monitoring (§2.14). Scaling path documented (§13.4). Multi-region is a Phase 4 concern.

**Revision trigger**: ~~Move to managed services (AWS RDS, ElastiCache) when operational complexity of self-managing databases exceeds a solo developer's capacity, or when multi-region is required.~~ Triggered before initial deployment — operational risk of self-managing databases for COPPA-regulated children's data deemed unacceptable for a solo developer. See ADR-010.

### ADR-007: Stripe Connect for Marketplace Payments

**Status**: Superseded by Hyperswitch (see `specs/domains/07-mkt.md §7`)

**Context**: The marketplace requires creator identity verification, payouts, sales tax handling, and 1099-K filing `[S§9.6, S§15.4]`. Building payment infrastructure from scratch is prohibitively complex for a solo developer.

**Decision**: Use Stripe Connect (Standard accounts) for marketplace payments and Stripe Billing for subscriptions.

> **Superseded**: Hyperswitch (self-hosted) replaces direct Stripe Connect integration as
> the payment orchestration layer. Stripe remains the underlying payment processor,
> configured as a connector in Hyperswitch. Creator KYC, 1099-K filing, and sales tax are
> still ultimately handled by Stripe — but the platform interacts with Stripe through
> Hyperswitch rather than directly. This provides processor-agnostic flexibility without
> sacrificing Stripe's compliance features. See `specs/domains/07-mkt.md §7` for the
> marketplace payment architecture and `§18.5` for Hyperswitch deployment details.

**Consequences** *(original, preserved for historical context)*:
- **Positive**: Stripe handles creator KYC/identity verification. 1099-K filing offloaded to Stripe. Sales tax automated via Stripe Tax. Stripe Connect Standard means creators manage their own Stripe accounts — reduced platform liability. Subscription management with upgrade/downgrade built in.
- **Negative**: Stripe fees (~2.9% + 30c per transaction + Connect fees). Platform doesn't control the payout experience directly.
- **Mitigation**: Stripe fees are industry standard and competitive. The alternative (custom payment infrastructure) is not viable for a solo developer.

**Revision trigger**: ~~None. Stripe is the industry standard for this exact use case.~~ Triggered — Hyperswitch provides an orchestration layer that preserves all Stripe benefits while adding processor-agnostic flexibility. See §2.9.

### ADR-008: Family-Scoped Data Isolation

**Status**: Accepted

**Context**: The platform handles children's data subject to COPPA `[S§17.2]`. Cross-family data leaks would be both a privacy violation and a regulatory violation. The spec mandates family data ownership `[S§16.2]`.

**Decision**: Enforce family-scoped data isolation at the architecture level through a Go interface/helper that requires `family_id` on all data-access queries, plus PostgreSQL Row-Level Security as defense-in-depth.

**Consequences**:
- **Positive**: Cross-family data access is structurally impossible without explicit opt-in. Go tooling catches missing family_id filters via interface enforcement. RLS provides database-level enforcement even if application logic has a bug. COPPA compliance is enforced by architecture, not by developer discipline.
- **Negative**: Every query includes a `family_id` filter, even when not logically necessary (e.g., platform-wide analytics). Social features (friends, groups) require explicit cross-family data paths.
- **Mitigation**: Social cross-family queries use separate, explicitly-defined repository functions that document why cross-family access is needed. Analytics queries use separate read-only database roles that bypass RLS.

```go
// internal/shared/family_scope.go

// FamilyScope enforces family-scoped database access.
type FamilyScope struct {
    FamilyID uuid.UUID
}

// ApplyScope adds a family_id WHERE clause to a GORM query.
func (s FamilyScope) ApplyScope(db *gorm.DB) *gorm.DB {
    return db.Where("family_id = ?", s.FamilyID)
}

// FamilyScopedRepository defines the interface for family-scoped data access.
type FamilyScopedRepository[T any] interface {
    FindByFamily(ctx context.Context, scope FamilyScope) ([]T, error)
    FindByIDAndFamily(ctx context.Context, id uuid.UUID, scope FamilyScope) (*T, error)
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

### ADR-010: AWS over Hetzner Dedicated Server

**Status**: Accepted (supersedes ADR-006)

**Context**: ADR-006 selected Hetzner dedicated server (~$60/mo) for cost efficiency. However, this requires the solo developer to self-manage PostgreSQL (backups, upgrades, PITR, replication), Redis persistence, TLS certificates (Let's Encrypt via Caddy), firewall rules (UFW), and OS patching. For a COPPA-regulated platform handling children's learning data, the operational burden and risk of data loss from self-managed infrastructure outweighs the cost savings. The ADR-006 revision trigger — "when operational complexity exceeds a solo developer's capacity" — was triggered before initial deployment.

**Decision**: AWS managed services — ECS on EC2 (Graviton) for compute, RDS PostgreSQL 16 for database, ElastiCache Redis 7 for cache/queue, ALB with ACM for TLS termination. Estimated ~$100-120/mo (Phase 1).

**Consequences**:
- **Positive**: RDS automated backups with 30-day PITR. Auto-renewing TLS via ACM (zero certificate management). VPC network isolation (private subnets for compute and data). ECS-optimized AMI managed by AWS. Clear scaling path from Single-AZ to Multi-AZ to Aurora. No manual database operations.
- **Negative**: ~2x cost ($110 vs $60/mo). Partial vendor lock-in to AWS managed service APIs. NAT Gateway fixed cost (~$32/mo). Upfront VPC and IAM configuration complexity.
- **Mitigation**: Application code is cloud-agnostic — standard PostgreSQL wire protocol, standard Redis protocol, standard HTTP. Only infrastructure configuration (Terraform/CDK) is AWS-specific. Migration to another provider requires only infrastructure re-provisioning, not application changes.

**Revision trigger**: Consider Fargate if EC2 instance management becomes a burden. Consider dedicated servers only with a dedicated DevOps hire.

### ADR-011: AWS CDK over Terraform

**Status**: Accepted

**Context**: Infrastructure needs to be provisioned reproducibly. Two leading IaC options: AWS CDK (TypeScript) and Terraform (HCL). The project is AWS-only (ADR-010) and already uses TypeScript for the React frontend.

**Decision**: AWS CDK v2 (TypeScript). Single stack with 9 constructs in `infra/`. See §2.17 for technology selection and §13.7 for full construct breakdown.

**Consequences**:
- **Positive**: Same language as frontend (TypeScript) — no new language to introduce. Type-safe constructs catch configuration errors at compile time. L2 constructs reduce boilerplate (VPC in ~5 lines vs ~50 in Terraform). `cdk synth` validates without deploying. Single `cdk deploy` provisions everything.
- **Negative**: AWS-only (no multi-cloud). CloudFormation under the hood (slower deploys than Terraform apply). Debugging sometimes requires reading CloudFormation events.
- **Mitigation**: Application code is cloud-agnostic (ADR-010) — standard PostgreSQL wire protocol, standard Redis protocol, standard HTTP. Only IaC is AWS-specific. If multi-cloud is needed, Terraform migration is straightforward — construct boundaries map cleanly to Terraform modules.

**Revision trigger**: Evaluate Terraform if the project moves multi-cloud or if a DevOps hire prefers HCL.

### ADR-012: Interactive Learning Engine in Phase 1

**Status**: Accepted

**Context**: The original plan deferred interactive student-facing features (quizzes, in-platform content viewing, video playback, lesson sequences) to Phase 2. However, without these features the platform is a parent-operated tracking/logging system — parents record what happened offline. The core value proposition ("functionality modules scoped by methodology") requires students to engage with content directly on the platform. Deferring interactive learning to Phase 2 leaves Phase 1 without its primary differentiator.

**Decision**: Move the full interactive learning engine to Phase 1:
- **Assessment Engine** — Question bank with 6 question types, quiz builder, auto-scoring for objective types, parent-scored short answers `[S§8.1.9]`
- **In-Platform Content Viewer** — PDF rendering with page tracking `[S§8.1.10]`
- **Video Player** — Self-hosted HLS adaptive bitrate streaming + YouTube/Vimeo embed integration `[S§8.1.11]`
- **Lesson Sequences** — Ordered content paths (reading -> quiz -> video -> activity) with progression tracking `[S§8.1.12]`
- **Supervised Student Views** — Students 10+ get a simplified, parent-controlled interface `[S§8.6]`
- **Creator Authoring Tools** — Quiz builder and sequence builder in the marketplace creator dashboard `[S§9.1]`

**Consequences**:
- **Positive**: Phase 1 delivers the platform's core value from launch — methodology-scoped interactive learning, not just logging. Marketplace content is consumable in-platform, not just downloadable. Supervised student views create a genuine student-facing product.
- **Negative**: Phase 1 scope increases significantly (~72 learn endpoints vs ~12 previously). Video transcoding pipeline requires FFmpeg integration in the media domain. Student sessions add an auth pathway (scoped JWT) alongside the existing Kratos parent auth.
- **Mitigation**: New tables follow the existing three-layer pattern (Layer 1 publisher-scoped, Layer 3 family-scoped with RLS). Assessment engine reuses the methodology-as-configuration pattern — quiz availability is driven by `method_tool_activations`, not code branches. Video transcoding is a background job in the existing `asynq` queue. Student sessions are parent-initiated and COPPA-compliant within the existing consent framework.

**Revision trigger**: None. This is a scope decision that aligns Phase 1 with the product's core value proposition.

### ADR-013: Planning & Scheduling Domain

**Status**: Accepted

**Context**: Homeschool families plan their days and weeks around learning activities, co-op days, and social events. The existing spec has Learning (activity logging), Compliance (attendance tracking), and Social (events) as separate domains — but no unified calendar or scheduling view. Competing homeschool platforms (Homeschool Planet, Homeschool Manager) are essentially calendar-first applications. Without scheduling, the platform is a record-keeping tool, not a planning tool.

**Decision**: Add a Planning domain (`plan`, `specs/domains/17-planning.md`) that provides:
- Unified calendar view synthesizing learning activities, compliance attendance, and social events
- Weekly/daily schedule creation with recurring patterns
- Co-op day coordination (link schedules to group events)
- Phase 1 scope is read-only calendar + basic schedule creation; full scheduling (recurring templates, sharing) deferred to Phase 2

**Consequences**:
- **Positive**: Families can see their homeschool week in one view — the primary workflow for daily planning. Calendar is the highest-demand feature in homeschool forums. Integrating existing domain data (rather than duplicating) keeps the domain lightweight.
- **Negative**: Cross-domain data aggregation requires reading from Learning, Compliance, and Social — slightly complex read model. Calendar UI components add frontend complexity.
- **Mitigation**: Planning is a read-heavy, write-light domain. CQRS applies — the calendar read model aggregates data from other domains via their service interfaces. The domain's own write model is limited to schedule items (recurring patterns, custom entries). No new database tables for existing domain data.

**Revision trigger**: None. Planning is a core homeschool workflow.

### ADR-014: Data Lifecycle as an Explicit Domain

**Status**: Accepted

**Context**: Data export and account deletion are mentioned in SPEC.md §16.3 and required by COPPA §17.2, but no domain owns the orchestration of cross-domain data export, retention enforcement, or account deletion workflows. In practice, "delete my family account" requires coordinated action across IAM, Learning, Social, Marketplace, Compliance, Media, Notifications, and Planning — no single existing domain should own this cross-cutting workflow.

**Decision**: Add a Data Lifecycle domain (`lifecycle`, `specs/domains/15-data-lifecycle.md`) that orchestrates:
- Family data export (all domains contribute export handlers)
- Account deletion (grace period, confirmation, cross-domain deletion)
- COPPA-specific deletion (regulatory timelines)
- Data retention policy enforcement
- Account recovery (identity verification, support escalation)

**Consequences**:
- **Positive**: Clear ownership of cross-domain deletion and export orchestration. Regulatory compliance (COPPA, state privacy laws) has a single enforcement point. Prevents "deletion missed domain X" bugs.
- **Negative**: Lifecycle must understand what data exists in each domain — tight coupling on the data catalog. Background job orchestration adds complexity.
- **Mitigation**: Each domain implements an `ExportHandler` and `DeletionHandler` interface. Lifecycle orchestrates but does not access other domains' data directly — it calls their service interfaces. Registration is at startup (like event bus handlers), so missing domains are caught immediately.

**Revision trigger**: None. COPPA and privacy regulation make this a legal requirement.

---

*This architecture document translates the Homegrown Academy specification (`specs/SPEC.md`) into concrete, opinionated technology decisions. Every choice traces back to spec requirements via `[S§n]` references. The document is designed to be practical enough that development can start directly from it — the Go code examples, SQL schemas, and React patterns are intended as starting templates, not pseudocode.*

*Architecture decisions are not permanent. Each ADR includes a revision trigger — the specific condition under which the decision should be revisited. The goal is to build the simplest system that satisfies Phase 1 requirements while ensuring a clear, documented path to Phase 2-4 scale.*
