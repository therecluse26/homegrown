# Homegrown Academy — Coding Standards

## §1 Authority

This document is the authoritative rulebook for all implementation work on Homegrown Academy.
It is a companion to `specs/ARCHITECTURE.md`, which explains *why* decisions were made.
This document says *how* code MUST be written.

- **Do not read this document for rationale.** Read `specs/ARCHITECTURE.md` for that.
- **Violations of rules in this document are bugs**, not style preferences. They MUST be fixed
  before a feature is merged, regardless of whether they affect observable behavior.
- RFC 2119 keywords apply throughout: **MUST** / **MUST NOT** are absolute;
  **SHOULD** / **SHOULD NOT** are strong recommendations; **MAY** is discretionary.

Cross-references to ARCHITECTURE.md use `[ARCH §n]` notation.

---

## §2 Rust Backend Standards

### §2.1 Module Structure

Every domain MUST contain the **base files** listed below. Absent a base file means absent
functionality — do not create placeholder files. The **conditional files** MUST be created only
when the domain needs them; do not create them as placeholders.

**Base files (all domains):**

```
src/{domain}/
├── mod.rs              # Re-exports, domain-level doc comments
├── handlers.rs         # Axum route handlers (thin layer only)
├── service.rs          # Business logic
├── repository.rs       # Database access (all queries live here)
├── models.rs           # Request/response types, DTOs
├── ports.rs            # Service + repository trait definitions [§8.2]
└── entities/           # SeaORM-generated entity files (do not hand-edit)
```

**Conditional files (add when the domain needs them):**

```
src/{domain}/
├── events.rs           # Domain event types emitted by this domain (if any) [§8.4]
├── event_handlers.rs   # Handlers for events from other domains (if any) [§8.4]
├── adapters/           # External service wrappers (if domain calls external APIs) [§8.1]
│   ├── mod.rs
│   └── {service}.rs
└── domain/             # Aggregate roots + value objects (complex domains only) [§8.3]
    ├── mod.rs
    ├── {aggregate}.rs
    └── errors.rs
```

> `ports.rs` is **required** for all domains (§8.2). `events.rs`, `event_handlers.rs`,
> `adapters/`, and `domain/` are conditional — only create them when the domain needs them.

**Layer responsibilities** — violations are bugs:

| Layer | MUST do | MUST NOT do |
|-------|---------|-------------|
| `handlers.rs` | Extract inputs from request, call one service method, return response | Contain business logic, call repositories, contain conditional logic beyond input validation |
| `service.rs` | Orchestrate business rules, call repositories, enforce invariants | Execute raw SQL, call another domain's repository |
| `repository.rs` | Execute database queries, map SeaORM results to domain types | Contain business logic, call other repositories across domain boundaries |
| `models.rs` | Define request/response structs with serde/OpenAPI derives | Contain logic |
| `ports.rs` | Define `{Domain}Service` and `{Entity}Repository` traits (inbound and outbound ports) | Contain implementations |
| `adapters/*.rs` | Wrap external SDK calls; return domain types only | Contain business logic |
| `domain/*.rs` | Enforce aggregate invariants via methods; emit domain events | Access database directly |
| `events.rs` | Define domain event types emitted by this domain | Import other domains' services |

### §2.2 Error Handling

- MUST use `thiserror` for all error types. `[ARCH §3]`
- MUST use `?` for error propagation. Do not call `.map_err(|_| ...)` when `?` suffices.
- MUST NOT use `.unwrap()` or `.expect()` outside `#[cfg(test)]` blocks.
- MUST NOT use `panic!()` in handler, service, or repository code.
- Application errors MUST map to `AppError` before reaching the handler return type.
- Error messages returned to API clients MUST NOT expose internal details (stack traces,
  SQL errors, internal field names). `[ARCH §1.5]`

### §2.3 Type Safety

- All API request and response types MUST be defined in `models.rs` for their domain.
- All types used in API responses MUST derive `serde::Serialize`, `serde::Deserialize`,
  and the OpenAPI schema trait (`utoipa::ToSchema` or equivalent).
- MUST NOT use `serde_json::Value` in API-facing types, **except** for methodology JSONB
  configuration fields, which are inherently schema-free by design. `[ARCH §1.6]`
- MUST NOT write hand-authored TypeScript API types. All frontend types come from code
  generation. `[ARCH §1.3]`

### §2.4 Database Patterns

- ALL database queries MUST go through the domain's `repository.rs`. No exceptions.
- Repositories MUST accept a `FamilyScope` parameter on every user-data query.
  `FamilyScope` is defined in `src/shared/family_scope.rs`. `[ARCH §1.5]`
- MUST NOT write raw SQL strings outside of migration files. Use SeaORM query builder.
- MUST NOT call another domain's `repository.rs` directly. Call its `service.rs` instead.
- SeaORM entities MUST be generated from migrations before writing queries against new
  tables. The generated files in `entities/` MUST NOT be hand-edited.

### §2.5 Privacy Enforcement

- MUST filter by `family_id` in every query that touches user-generated data. `[ARCH §1.5]`
- MUST NOT store GPS coordinates, lat/long, or precise location data. Store city/region
  identifiers only. `[ARCH §1.5]`
- Endpoints that access student resources (portfolios, assessments, progress records)
  MUST verify that the requesting user is a parent of that student before returning data.
- Social content visibility MUST default to friends-only at the database level.
  Do not rely on application-layer filtering as the only privacy control.

### §2.6 API Handler Pattern

- Handlers MUST have the signature:
  ```rust
  async fn handler_name(...extractors...) -> Result<impl IntoResponse, AppError>
  ```
- Input extraction MUST use Axum extractors (`Json`, `Path`, `Query`, `State`, `Extension`).
  Do not parse `Request` manually.
- Input validation MUST use the `validator` crate on request structs. Do not write bespoke
  validation logic in handlers.
- HTTP verbs and status codes MUST follow REST conventions:
  - `GET` → 200 OK (list/retrieve)
  - `POST` → 201 Created (resource creation)
  - `PUT` / `PATCH` → 200 OK (update)
  - `DELETE` → 204 No Content
  - Validation errors → 422 Unprocessable Entity
  - Authorization failures → 403 Forbidden (not 404, unless intentional obscuring)

### §2.7 Forbidden Patterns

The following patterns are **never** acceptable in committed code:

| Pattern | Why it is forbidden |
|---------|---------------------|
| `.unwrap()` or `.expect()` outside `#[cfg(test)]` | Panics in production on None/Err |
| `panic!()` in handlers, services, or repositories | Kills the request thread |
| `unsafe { }` without a `// SAFETY:` comment and team review | Bypasses Rust's guarantees |
| `if methodology == "charlotte_mason"` (branching on methodology name) | Violates Methodology-as-Configuration `[ARCH §1.6]` |
| Calling another domain's `repository.rs` directly | Violates layering; bypasses domain invariants |
| `todo!()` or `unimplemented!()` in committed code | Panics on invocation |
| `serde_json::Value` in non-JSONB API types | Destroys type safety at the API boundary |
| Raw SQL strings in application code (outside migrations) | Bypasses SeaORM type safety |
| Raw SDK call in `service.rs` (e.g., `stripe.create_customer()`) | Bypasses adapter isolation; blocks vendor swaps and unit testing `[ARCH §4.3]` |
| Logging PII, tokens, or secrets | Privacy and security violation `[ARCH §1.5]` |

### §2.8 Testing

- Unit tests MUST live in `#[cfg(test)]` blocks within the source file they test.
- Integration tests MUST live in `tests/` and MUST use a real test database.
- MUST NOT mock the database in integration tests. Test against real PostgreSQL.
- Every new public API endpoint MUST have at least one integration test covering the
  happy path and at least one covering an authorization failure.
- Test database setup MUST use migrations to create schema (not hand-rolled DDL).

**Unit vs. integration test distinction:**

- Service-layer business logic SHOULD have unit tests in `#[cfg(test)]` blocks within
  `service.rs` that inject mock `impl {Entity}Repository` types. This is enabled by the
  port traits defined in `ports.rs` (§8.2).
- Mock repository implementations are `#[cfg(test)]`-only and MUST be defined inline in
  the test module — never in production code paths.
- **Unit tests** (in-file, mock repo) verify *business logic* in isolation.
  **Integration tests** (`tests/`, real `Pg*Repository`) verify *correctness with a real
  database*. Both are required for domains with non-trivial service logic.

---

## §3 TypeScript / React SPA Standards

> **Note**: The Astro public site and AWS CDK infrastructure are deferred. Standards in this
> section apply to the React SPA (`frontend/`).

### §3.1 Type Strictness

- MUST NOT use the `any` type anywhere in application code. `[ARCH §1.3]`
- MUST NOT use `as T` type assertions without an explanatory comment on the same line
  justifying why the assertion is safe.
- ALL types for API request and response shapes MUST come from `src/api/generated/`.
  These are generated from the OpenAPI spec — never hand-write them. `[ARCH §1.3]`
- TypeScript strict mode MUST be enabled (`"strict": true` in `tsconfig.json`).

### §3.2 Component Structure

- Feature components MUST live in `src/features/{domain}/` matching the backend domain name.
- Shared primitive UI components (buttons, inputs, modals) MUST live in `src/components/ui/`.
- MUST NOT co-locate API calls or data-fetching logic in component render functions.
  Data fetching belongs in custom hooks using TanStack Query.
- MUST use functional components. Class components are forbidden.

### §3.3 State Management

- Server state (data from the API) MUST use TanStack Query (`@tanstack/react-query`).
- Local UI state MUST use `useState` or `useContext`. No other client-state library is used.
- MUST NOT use Redux, Zustand, Jotai, or any external client-state library.
- MUST NOT call the TanStack Query client directly inside components. Wrap all queries and
  mutations in custom hooks (`use` prefix).

### §3.4 API Consumption

- MUST use the generated API client from `src/api/generated/`. Do not instantiate raw
  HTTP clients in components or hooks.
- MUST NOT use `fetch` or `axios` directly in components or hooks.
- MUST NOT hardcode API base URLs. Use `import.meta.env.VITE_API_BASE_URL` (or the
  configured environment variable) exclusively.

### §3.5 Naming Conventions

| Entity | Convention | Example |
|--------|-----------|---------|
| React components | PascalCase | `StudentProgressCard` |
| Custom hooks | `use` prefix, camelCase | `useStudentProgress` |
| Interfaces / types | PascalCase, no `I` prefix | `StudentProfile`, not `IStudentProfile` |
| Module-level constants | SCREAMING_SNAKE_CASE | `MAX_UPLOAD_SIZE_MB` |
| Files | kebab-case | `student-progress-card.tsx` |
| Feature directories | Match backend domain name | `src/features/learning/` |

### §3.6 Forbidden Patterns

| Pattern | Why it is forbidden |
|---------|---------------------|
| `any` type | Defeats TypeScript's type system |
| Inline styles (`style={{ }}`) | Bypasses design system; use Tailwind classes |
| Class components | Not supported; use functional components |
| Direct DOM manipulation (`document.getElementById`, etc.) | Use React refs |
| Hardcoded methodology name strings | Use `useMethodologyContext()` hook |
| Hand-written API types | Use generated types from `src/api/generated/` |
| `fetch` or `axios` in components/hooks | Use generated API client |
| `placeholder` as only label | Placeholders disappear on input; use visible `<label>` elements |
| `onClick` on non-interactive elements | Use `<button>` or `<a>` for click handlers; `<div onClick>` is not keyboard-accessible |
| `tabIndex > 0` | Positive tabindex disrupts natural tab order; use 0 or -1 only |

### §3.7 Accessibility `[ARCH §11.5]`

Accessibility is a first-class concern, not a post-launch enhancement. Violations are bugs. `[S§17.6]`

- All images MUST have `alt` attributes. Decorative images MUST use `alt=""`. Dynamic images
  (user uploads, marketplace covers) MUST have descriptive alt text derived from metadata.
- All form inputs MUST have an associated `<label>` element (via `htmlFor`). Placeholder text
  is NOT a substitute for a label.
- All interactive elements MUST be keyboard-operable. Click handlers MUST be on `<button>` or
  `<a>` elements — not `<div>` or `<span>`. Custom interactive components MUST handle `Enter`,
  `Space`, and `Escape` keys as appropriate per WAI-ARIA Authoring Practices.
- Focus management: Route transitions MUST move focus to the page's main heading (`<h1>`) or
  main content region. Modals MUST trap focus and return focus to the trigger element on close.
- ARIA: Prefer semantic HTML (`<nav>`, `<main>`, `<aside>`, `<dialog>`) over ARIA roles.
  When ARIA is needed, follow WAI-ARIA Authoring Practices exactly. MUST NOT use `role="button"`
  on a `<div>` when a `<button>` element would suffice.
- Dynamic content: Feed updates, notification toasts, and quiz feedback MUST use `aria-live`
  regions. Polite for background updates; assertive for error messages and critical alerts.
- Color: MUST NOT use color as the sole indicator of state. Progress bars, status badges, and
  validation feedback MUST include text or icons in addition to color.
- Skip links: Every page layout MUST include a visually hidden "Skip to main content" link as
  the first focusable element.
- Drag-and-drop: All drag-and-drop interfaces (schedule planning, sequence building) MUST
  provide an equivalent keyboard mechanism (e.g., move-up/move-down buttons, reorder modal).

### §3.8 Print Styles `[S§17.9]`

- Pages designated as printable (schedules, compliance reports, progress summaries) MUST include
  `@media print` stylesheets that hide navigation, sidebars, and interactive controls.
- Print layouts MUST reflow content to fit US Letter (8.5"×11") and A4 page widths.
- Print output MUST include a header with family name, date range, and generation timestamp.
- Color output MUST NOT be required for print readability — all meaning conveyed by color MUST
  also be conveyed by text, icons, or patterns that reproduce in grayscale.

---

## §4 Database Standards

### §4.1 Naming Conventions

- Table names: `{domain_prefix}_{plural_noun}` (e.g., `soc_posts`, `lrn_activities`,
  `iam_sessions`). Domain prefixes mirror the backend module name.
- Index names: `idx_{table_name}_{column_or_columns}` (e.g., `idx_soc_posts_family_id`).
- Foreign key names: `fk_{child_table}_{parent_table}` (e.g., `fk_soc_posts_iam_families`).
- Column names: `snake_case`.
- Enum type names: `{domain_prefix}_{noun}_enum` (e.g., `soc_visibility_enum`).

### §4.2 Migration Rules

- Migrations are **append-only**. MUST NOT edit or delete a committed migration file.
  If a schema change is needed, write a new migration.
- Every migration MUST include a reversible `down` migration. Irreversible migrations MUST
  include an explicit comment explaining why reversal is impossible.
- Migrations MUST be idempotent where possible (use `IF NOT EXISTS`, `IF EXISTS`).
- After running a new migration, MUST regenerate SeaORM entities before writing any query
  against the new or modified tables.
- Migration files MUST be named with a timestamp prefix: `YYYYMMDD_HHMMSS_description.rs`
  (following `sea-orm-migration` conventions).

### §4.3 Index Policy

- Every foreign key column MUST have an index.
- Every column used in a `WHERE` clause in a repository query MUST have an index (or be
  part of a composite index).
- Composite indexes MUST list the highest-cardinality column first.

---

## §5 Cross-Cutting Rules

### §5.1 Methodology-as-Configuration

`[ARCH §1.6]`

- MUST NOT branch on methodology name in application code. No string comparisons, no enums
  switching on methodology identity.
- Methodology-dependent behavior MUST be resolved by querying the methodology configuration
  record from the `method::` service (or its cached form).
- Methodology display labels and terminology MUST come from the methodology config record,
  not from hard-coded strings.
- Adding a new methodology MUST require only inserting database rows. If new code is
  required to support a new methodology, that is a violation of this principle.

### §5.2 Security Rules

- MUST NOT log personally identifiable information (names, emails, IP addresses in
  application logs), session tokens, or secrets.
- User-submitted HTML MUST be sanitized using the `ammonia` crate before storage or display.
  Do not sanitize HTML in the frontend only.
- File uploads MUST validate file magic bytes (not just extension or MIME type from the
  `Content-Type` header). Extension-only validation is insufficient and exploitable.
- MUST NOT expose internal error details (stack traces, SQL error messages, internal field
  names) in API error responses. Log internally; return a generic message externally.
- All endpoints that mutate state MUST enforce CSRF protection where applicable.
- Rate limiting MUST be applied to all public-facing endpoints.

### §5.3 Shared Utilities

Before writing a new utility, check `src/shared/` for an existing implementation:

| File | Purpose |
|------|---------|
| `src/shared/pagination.rs` | Cursor-based and offset pagination helpers |
| `src/shared/family_scope.rs` | `FamilyScope` type for privacy-enforcing queries |
| `src/shared/db.rs` | Database connection pool acquisition |
| `src/shared/redis.rs` | Redis connection and caching helpers |
| `src/shared/types.rs` | Common newtypes (e.g., `FamilyId`, `UserId`) |
| `src/shared/events.rs` | `EventBus` and `DomainEvent` trait (§8.4) |

MUST NOT duplicate functionality already present in `src/shared/`. Extend the shared
module instead.

---

## §6 Code Generation Protocol

Type safety flows from database → Rust → OpenAPI → TypeScript. Each step produces
artifacts that MUST be committed to version control.

### §6.1 OpenAPI Spec Generation

```bash
cargo run --bin openapi-gen
```

- Output: `openapi/spec.yaml`
- MUST run after any change to Rust API types in `models.rs`.
- MUST commit `openapi/spec.yaml` alongside the Rust changes in the same commit.
- MUST NOT generate the spec at runtime. It is a build artifact, committed statically.

### §6.2 TypeScript Type Generation

```bash
cd frontend && npm run generate-types
```

- Output: `frontend/src/api/generated/`
- MUST run after `openapi/spec.yaml` changes.
- MUST commit generated files alongside the spec change.
- MUST NOT hand-edit files in `src/api/generated/`. They will be overwritten on the next
  generation run.

### §6.3 SeaORM Entity Generation

```bash
sea-orm-cli generate entity -o src/{domain}/entities/
```

- MUST run after every database migration.
- Generated files in `entities/` MUST NOT be hand-edited.
- MUST commit generated entities alongside the migration that produced them.

### §6.4 Generation Order

When making a change that touches all layers:

1. Write and run database migration
2. Regenerate SeaORM entities (`§6.3`)
3. Write/update Rust types and handlers
4. Regenerate OpenAPI spec (`§6.1`)
5. Regenerate TypeScript types (`§6.2`)
6. Update frontend components/hooks to use new types

---

## §7 AI-Assisted Development Protocol

These rules apply when Claude (or any AI assistant) is generating or modifying code.

### §7.1 Before Writing Code

1. Read `specs/ARCHITECTURE.md` §1 (Principles), §4 (Internal Architecture Patterns), and the relevant domain section.
2. Read `specs/SPEC.md` for the requirements in the domain being worked on.
3. Check `src/shared/` for existing utilities before writing new ones.
4. Read the existing module files (if any) to understand current patterns before adding to them.

### §7.2 Quality Gates

Every code generation session MUST leave the codebase passing:

```bash
cargo clippy -- -D warnings    # Zero warnings
cargo test                     # All tests pass
npm run type-check             # Zero TypeScript errors (frontend/)
npx playwright test --project=a11y  # Zero critical/serious accessibility violations
```

The accessibility gate (`npx playwright test --project=a11y`) runs axe-core against all page
routes and MUST produce zero critical or serious violations. `[S§17.6.6, ARCH §11.5]`

Committing code that fails any of these gates is not acceptable, even as a "WIP" commit
on a feature branch.

### §7.3 Structural Rules for Generated Code

- Prefer editing existing files over creating new ones.
- MUST NOT add `TODO` comments in committed code. If something is unfinished, the commit
  should not include it.
- MUST NOT use `todo!()` or `unimplemented!()` as placeholders in committed code.
- MUST NOT add docstrings or comments to code that was not changed in the current session,
  unless the comment directly supports understanding the new code.
- MUST NOT refactor code that is not directly related to the current task.
- MUST NOT add features, fallback handling, or configuration options beyond what the
  current task requires (no speculative generality).

---

## §8 Architectural Pattern Rules

These rules are the enforcement layer for the pattern stack defined in `[ARCH §4]`. Do not
read this section for rationale — read `specs/ARCHITECTURE.md §4` for that. Every rule
here is an absolute enforcement imperative.

### §8.1 Bounded Context Rules

`[ARCH §4.2]`

- MUST NOT write to another domain's prefixed tables from outside that domain. Each domain
  owns its `{prefix}_*` tables exclusively.
- MUST NOT call another domain's `repository.rs` directly. Call its service trait instead.
  (Reinforces §2.4 with explicit bounded-context framing.)
- MUST wrap all external SDK calls (Hyperswitch, Kratos, R2, Thorn Safer, Postmark, Rekognition)
  in an `adapters/` file within the owning domain. No raw SDK calls in `service.rs`.
- MUST NOT add files to `src/shared/` without explicit justification that the utility is
  needed by three or more domains. Convenience refactors do not qualify.

**Forbidden patterns** (additions to the §2.7 table):

| Pattern | Why it is forbidden |
|---------|---------------------|
| Domain A writing to `domain_b_*` tables | Violates domain table ownership `[ARCH §4.2]` |
| Raw SDK call in `service.rs` | Bypasses adapter isolation, blocks vendor swaps `[ARCH §4.3]` |
| Adding to `src/shared/` for convenience | Grows the Shared Kernel; increases coupling `[ARCH §4.2]` |

### §8.2 Port Trait Rules (Inbound + Outbound)

`[ARCH §4.4]`

- MUST define a service trait before implementing it. The trait MUST be named `{Domain}Service`
  (e.g., `LearningService`). The implementation MUST be named `{Domain}ServiceImpl`.
- MUST define a repository trait before implementing it. The trait MUST be named
  `{Entity}Repository` (e.g., `ActivityRepository`). The concrete PostgreSQL implementation
  MUST be named `Pg{Entity}Repository`.
- Handlers MUST receive the service as `Arc<dyn DomainService>` via Axum `State`. MUST NOT
  receive the concrete `{Domain}ServiceImpl` type.
- Services MUST receive repositories as `Arc<dyn RepositoryTrait>`. MUST NOT construct or
  hold a concrete `Pg*Repository` directly.
- MUST NOT construct concrete service or repository implementations inside handlers or
  other services. Wiring happens exclusively in `app.rs` / `main.rs` at startup.
- Trait definitions MUST live in `ports.rs` within the domain directory. For simple domains
  where the trait is short (≤20 lines), it MAY be colocated at the top of `service.rs` or
  `repository.rs` — but the placement MUST be consistent across all files in the domain.

### §8.3 Domain Layer Rules (Complex Domains)

`[ARCH §4.5]`

Applies to: `learn/`, `social/`, `mkt/`, `safety/`, `comply/`, `method/`.

- MUST add a `domain/` subdirectory to any of the above domains before implementing
  state-machine logic or invariant enforcement.
- Aggregate root structs MUST have all fields `private` (no `pub` fields). State transitions
  happen exclusively via methods on the aggregate.
- MUST NOT modify aggregate state directly in `service.rs`. The service calls aggregate
  methods (which return `Result<DomainEvent, DomainError>`), then persists and publishes.
- Domain events returned from aggregate methods MUST be published via `EventBus` (§8.4).
  MUST NOT silently discard returned domain events.
- `DomainError` types MUST be defined in `domain/errors.rs` and MUST NOT be the same type
  as `AppError`. Conversion from `DomainError` to `AppError` happens in `service.rs`.

### §8.4 Event Bus Rules

`[ARCH §4.6]`

- MUST use `EventBus::publish` for all cross-domain reactions. MUST NOT import another
  domain's service to call it directly in response to a domain event.
- Event types MUST be defined in the *emitting* domain's `events.rs` file. The consuming
  domain imports the event type, never the emitting domain's service.
- Event handler implementations MUST live in the *consuming* domain, in a file named
  `{event_name}_handler.rs` or grouped in `event_handlers.rs`.
- Handlers that perform heavy work (image scanning, search indexing, email sending) MUST
  enqueue a background job rather than executing the work inline. MUST NOT block the
  request thread with expensive operations inside an event handler.
- `EventBus` subscriptions MUST be registered at application startup (in `app.rs` or
  `main.rs`). MUST NOT register subscriptions dynamically at runtime.

### §8.5 CQRS Rules (Applicable Domains)

`[ARCH §4.7]`

Applies to: `social/`, `mkt/`, `learn/`, `comply/`, `search/`, `ai/`.

- Command functions (writes) MUST return `Result<Id, AppError>` or `Result<(), AppError>`.
  MUST NOT return rich read models after a write ("return-what-you-created" pattern).
- Query functions (reads) MUST NOT have side effects. MUST NOT mutate state, emit events,
  or enqueue jobs.
- MUST NOT mix command and query logic in the same service method. A method that both
  writes and returns a rich read is a violation of this rule.
- Read-side optimization MUST follow the progressive ladder defined in `[ARCH §4.7]`.
  MUST NOT add Redis caching or materialized views for a query before measuring that the
  standard SeaORM query is actually insufficient.
