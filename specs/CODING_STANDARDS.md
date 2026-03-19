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

Every domain MUST contain the following files. Absent a file means absent functionality — do
not create placeholder files.

```
src/{domain}/
├── mod.rs          # Re-exports, domain-level doc comments
├── handlers.rs     # Axum route handlers (thin layer only)
├── service.rs      # Business logic
├── repository.rs   # Database access (all queries live here)
├── models.rs       # Request/response types, DTOs
└── entities/       # SeaORM-generated entity files (do not hand-edit)
```

**Layer responsibilities** — violations are bugs:

| Layer | MUST do | MUST NOT do |
|-------|---------|-------------|
| `handlers.rs` | Extract inputs from request, call one service method, return response | Contain business logic, call repositories, contain conditional logic beyond input validation |
| `service.rs` | Orchestrate business rules, call repositories, enforce invariants | Execute raw SQL, call another domain's repository |
| `repository.rs` | Execute database queries, map SeaORM results to domain types | Contain business logic, call other repositories across domain boundaries |
| `models.rs` | Define request/response structs with serde/OpenAPI derives | Contain logic |

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
| Logging PII, tokens, or secrets | Privacy and security violation `[ARCH §1.5]` |

### §2.8 Testing

- Unit tests MUST live in `#[cfg(test)]` blocks within the source file they test.
- Integration tests MUST live in `tests/` and MUST use a real test database.
- MUST NOT mock the database in integration tests. Test against real PostgreSQL.
- Every new public API endpoint MUST have at least one integration test covering the
  happy path and at least one covering an authorization failure.
- Test database setup MUST use migrations to create schema (not hand-rolled DDL).

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

1. Read `specs/ARCHITECTURE.md` §1 (Principles) and the relevant domain section.
2. Read `specs/SPEC.md` for the requirements in the domain being worked on.
3. Check `src/shared/` for existing utilities before writing new ones.
4. Read the existing module files (if any) to understand current patterns before adding to them.

### §7.2 Quality Gates

Every code generation session MUST leave the codebase passing:

```bash
cargo clippy -- -D warnings    # Zero warnings
cargo test                     # All tests pass
npm run type-check             # Zero TypeScript errors (frontend/)
```

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
