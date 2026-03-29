# Homegrown Academy — Claude Development Guide

Homegrown Academy is a privacy-first platform for homeschooling families, built with a
Go/Echo backend, React SPA frontend, and PostgreSQL database. It supports multiple
homeschooling methodologies (Charlotte Mason, Classical, Unschooling, etc.) as runtime
configuration — not code branches.

---

## Before Writing Any Code

1. **Read `specs/CODING_STANDARDS.md`** — the authoritative rulebook for this project.
   Violations are treated as bugs, not style preferences.
2. **Read the relevant section(s) of `specs/ARCHITECTURE.md`** for any domain you are
   working in. It explains *why* each decision was made.
3. **Read `specs/SPEC.md` §[n]** for the requirements in your domain before implementing.
4. **Check `internal/shared/`** for existing utilities before writing new ones.
5. **For frontend work**, read `specs/DESIGN_TOKENS.md` — the normative design token spec.
   All colors, typography, spacing, and component styles MUST use tokens defined there.

---

## Code Searching (Reflex MCP)

This project uses [Reflex](https://github.com/reflex-search/reflex) as an MCP tool for all
code searching. The Reflex MCP tools (`search_code`, `search_regex`, `search_ast`,
`list_locations`, `count_occurrences`) **MUST** be used instead of Grep/Glob for searching
code content. The `Read` tool is still used for reading full files.

- **Index maintenance:** Run `./node_modules/.bin/rfx index` if the index becomes stale.
- **Binary location:** `./node_modules/.bin/rfx`
- **Subagent delegation:** When launching Explore or Plan agents via the Task tool, the
  prompt MUST instruct the agent to use `mcp__reflex__*` tools (search_code, search_regex,
  list_locations, count_occurrences) instead of Grep/Glob for all code content searches.
  Glob is still acceptable for file-name-only pattern matching (e.g., finding `*.go` files).

---

## Database Inspection (Plenum MCP)

This project uses [Plenum](https://github.com/coredb-io/plenum) as a read-only database
introspection and query MCP server. It gives Claude direct access to PostgreSQL schema and
data without needing `psql` via Bash.

### Tools

| Tool | Purpose |
|------|---------|
| `mcp__plenum__connect` | Connect to a named database (`agent` or `dev`) |
| `mcp__plenum__introspect` | Inspect schema: tables, columns, types, constraints |
| `mcp__plenum__query` | Execute read-only SQL (`SELECT` only) |

### Named Connections

| Name | Database | Use case |
|------|----------|----------|
| `agent` | `homegrown_agent` | Agent testing — **use this by default** |
| `dev` | `homegrown` | Developer's working database |

Connection config lives in `.plenum/config.json` (committed to git).
Credentials: user `homegrown`, password `homegrown`, host `localhost`, port `5932`.

### When to Use

- **Schema inspection** — check table structure, column types, constraints after migrations
- **Debugging data issues** — verify seed data, check foreign key relationships
- **Verifying migrations** — confirm new tables/columns exist after `make migrate`
- **Query validation** — test SELECT queries before embedding them in Go code

### When NOT to Use

- **Write operations** — Plenum is strictly read-only (rejects INSERT/UPDATE/DELETE/DDL).
  Use `psql` via Bash for writes.
- **Performance testing** — use `psql` with `EXPLAIN ANALYZE` directly.

### Subagent Delegation

When launching Explore or Plan agents via the Task tool, the prompt MUST instruct the agent
to use `mcp__plenum__*` tools for database inspection instead of running `psql` via Bash.

---

## Coding Standards & Architecture

All implementation work MUST follow `specs/CODING_STANDARDS.md`. That document is the
authoritative source for:

- Package structure and layer responsibilities
- Error handling rules
- Type safety requirements
- Database patterns (family-scoped queries, migration rules)
- Privacy enforcement
- Naming conventions
- Forbidden patterns
- Code generation protocol (OpenAPI → TypeScript types)

`specs/ARCHITECTURE.md` explains *why* each decision was made and contains the full system
design. Consult it before making any architectural decision.

---

## Key Constraints (Non-Negotiable Summary)

| Area | Rule |
|------|------|
| **Go** | All errors MUST be checked (`if err != nil`). No ignored error returns in production code. Use custom error types with `errors.Is`/`errors.As`. |
| **TypeScript** | Strict mode enabled. Zero `any`. All API types from `src/api/generated/` only. |
| **Database** | Every user-data query MUST be family-scoped via `FamilyScope`. Migrations are append-only (goose). |
| **Methodology** | MUST NOT branch on methodology name in code. Use config lookup via `method` package. |
| **Privacy** | Never store GPS coordinates. Student resources require parent ownership check. Never log PII or tokens. |
| **File uploads** | Validate magic bytes, not just file extension. |
| **API errors** | Never expose internal error details in responses. Log internally; return generic message. |
| **Generated files** | `frontend/src/api/generated/` is generated — never hand-edit. |
| **Frontend styling** | All colors, spacing, radii, shadows, and z-index MUST use design token classes from `specs/DESIGN_TOKENS.md`. No hardcoded hex values. No arbitrary z-index. No `1px solid` borders for sectioning. |

---

## Quality Gates (Must Pass Before Every Commit)

```bash
golangci-lint run              # Zero warnings
go test ./...                  # All tests pass
npm run type-check             # Zero TypeScript errors (in frontend/)
```

Additionally, before every commit verify:
- [ ] Relevant `specs/` files updated if any architectural or design decision was made or changed

---

## Frontend Validation with Playwright (Non-Negotiable)

After implementing any frontend change, you MUST validate it visually using the Playwright
MCP server **before declaring the work complete**. TypeScript type-checking alone is not
sufficient — components must be verified to render and behave correctly in a browser.

### Setup

Start the Vite dev server yourself using the Bash tool with `run_in_background: true`.
Pick any free port (e.g. 5174 to avoid colliding with the user's own dev server):

```bash
# run_in_background: true
cd frontend && npm run dev -- --port 15173
```

Wait a few seconds for the server to be ready, then run your Playwright checks against
`http://localhost:15173`. Kill the background process when validation is complete.

### Required Validation Steps

For every frontend change, perform ALL of the following:

1. **Navigate to the affected page(s):**
   ```
   mcp__pw__browser_navigate → http://localhost:15173/{route}
   ```

2. **Capture an accessibility snapshot** to verify the DOM structure rendered correctly:
   ```
   mcp__pw__browser_snapshot
   ```

3. **Check for console errors and warnings:**
   ```
   mcp__pw__browser_console_messages (level: "warning")
   ```
   Zero errors are acceptable. Investigate and fix any that appear.

4. **Exercise the changed interaction(s).** If you added or modified a button, form,
   modal, or other interactive element — click it, fill it, or trigger it:
   ```
   mcp__pw__browser_click / mcp__pw__browser_type / mcp__pw__browser_fill_form
   ```

5. **Take a screenshot** of the final visual state and describe what you observe:
   ```
   mcp__pw__browser_take_screenshot (type: "png")
   ```

### Definition of Done (Frontend)

A frontend change is **not complete** until:
- [ ] `npm run type-check` passes (zero TypeScript errors)
- [ ] The page renders without console errors in Playwright
- [ ] The changed component/feature is visually correct per screenshot
- [ ] All interactions you implemented were exercised via Playwright and behaved as expected

---

## Spec Maintenance (Non-Negotiable)

When any implementation decision changes or extends the current specs, you MUST update the
relevant spec file(s) in `specs/` before considering the task complete:

| Change type | Update target |
|-------------|---------------|
| New or changed architectural decision | `specs/ARCHITECTURE.md` — add/update ADR |
| New or changed requirements | `specs/SPEC.md` — update relevant § |
| New coding patterns or rules | `specs/CODING_STANDARDS.md` |
| New design tokens or UI rules | `specs/DESIGN_TOKENS.md` |
| Domain-specific changes | `specs/domains/{nn}-{domain}.md` |

Do NOT wait until the end of a session. Update specs **as decisions are made**, inline with
the implementation work. If you discover during implementation that a spec is wrong or
incomplete, fix the spec first, then continue coding.

---

## Development Commands

| Command | Purpose |
|---------|---------|
| `make dev` | Start backend (air) + frontend (Vite) together |
| `make dev-api` | Start only the Go backend with hot-reload |
| `make dev-web` | Start only the Vite frontend dev server |
| `make check` | Run all quality gates (lint + test + type-check) |
| `make lint` | Run `golangci-lint` (zero warnings required) |
| `make test` | Run `go test ./...` |
| `make type-check` | Run TypeScript type checker in `frontend/` |
| `make migrate` | Run pending database migrations (goose) |
| `make db-reset` | Drop + recreate + migrate the database |
| `make seed` | Re-seed the agent database (idempotent; safe to rerun) |
| `make agent-db-reset` | Full agent DB reset: drop → recreate → migrate → seed |
| `make agent-kratos-reset` | Wipe + reinitialise the agent Kratos identity store |
| `make openapi` | Regenerate OpenAPI spec from Go swag annotations |
| `make generate-types` | Regenerate TypeScript types from OpenAPI spec |
| `make full-generate` | Run both `openapi` + `generate-types` in sequence |
| `make install-tools` | Install required build tools (swag, lefthook) |
| `make install-hooks` | Install lefthook git hooks (one-shot setup) |
| `make audit` | Run `govulncheck` vulnerability check |

---

## Agent Database

The project runs two isolated databases and two Kratos instances so that agent test
sessions never pollute the developer's working data.

### Two-Database Model

| Database | Purpose |
|----------|---------|
| `homegrown` | Developer's own working database; normally left alone |
| `homegrown_agent` | Isolated database used exclusively by AI agents for testing |

All seed/reset commands default to `homegrown_agent`. To target the developer database
explicitly: `make seed DB=homegrown`.

### Two Kratos Instances

| Instance | Public port | Admin port | Used by |
|----------|-------------|------------|---------|
| Dev Kratos | `localhost:4933` | `localhost:4934` | Developer login |
| Agent Kratos | `localhost:4935` | `localhost:4936` | Agent test identities (`seed@example.com`, `admin@example.com`) |

### When to Run Each Command

| Situation | Command |
|-----------|---------|
| Starting a new agent test session (data may be stale) | `make seed` |
| New migrations have been added since last reset | `make agent-db-reset` |
| Agent login / Kratos identity is broken or stale | `make agent-kratos-reset` |
| Full clean slate (schema + data + identities) | `make agent-db-reset && make agent-kratos-reset` |
| Seed the developer's own DB (rarely needed) | `make seed DB=homegrown` |

### Seeder Behaviour

- **Idempotent** — uses `ON CONFLICT … DO NOTHING` throughout; always safe to rerun
  without duplicating data.
- **Kratos connectivity** — tries agent Kratos admin API (`localhost:4936`) first, then
  falls back to dev Kratos (`localhost:4934`). If both are unreachable it falls back to
  deterministic UUID constants and logs a WARN. Seed data is still inserted correctly; the
  only difference is that Kratos identities won't exist for those UUIDs (agent login will
  fail until Kratos is running).

---

## Code Generation Protocol

The project generates TypeScript API types from Go source via a three-stage pipeline:

1. `make openapi` — runs `swag init` to produce `openapi/swagger.json` from Go annotations
2. `make generate-types` — converts swagger → OpenAPI 3 → `frontend/src/api/generated/schema.ts`
3. `make full-generate` — runs both steps in sequence (**use this by default**)

**When to regenerate:**

- **After changing any Go handler, request/response type, or swag annotation** — run
  `make full-generate` and stage the generated output alongside your Go changes.
- **Before starting any frontend feature work** — run `make full-generate` to ensure
  `schema.ts` reflects the latest backend state. Types may have changed in a prior session.
- **The lefthook pre-commit hook runs `make full-generate` automatically** when `.go` files
  are staged, but you should still regenerate proactively during development to catch type
  errors early.

**Never hand-edit** `frontend/src/api/generated/schema.ts` — it will be overwritten.

---

## Spec References

| Document | Location | Purpose |
|----------|----------|---------|
| Vision | `specs/VISION.md` | Product intent and user goals |
| Requirements | `specs/SPEC.md` | What the platform must do |
| Architecture | `specs/ARCHITECTURE.md` | Why each technical decision was made |
| **Coding Rules** | **`specs/CODING_STANDARDS.md`** | **How to write the code** |
| Design Vision | `specs/DESIGN.md` | Creative direction ("Curated Hearth" aesthetic) |
| **Design Tokens** | **`specs/DESIGN_TOKENS.md`** | **Implementable token values for all frontend styling** |

---

## Auto-Memory Policy

The specs in `specs/` are the single source of truth. Auto-memory (`memory/MEMORY.md`) is
only for information that **cannot be found** in specs or code. Before writing to memory:

1. **Do not duplicate specs.** Architecture, coding rules, domain requirements, design tokens
   — all live in `specs/`. Never copy them to memory.
2. **Do not store implementation details** that are discoverable by reading code (function
   names, test counts, migration numbers, wiring patterns).
3. **Only store:**
   - Current project status (which domains are implemented)
   - Dev environment quirks (tooling paths, OS-specific workarounds)
   - Recurring cross-domain patterns not yet promoted to `specs/CODING_STANDARDS.md`
4. **Keep it under 40 lines.** If adding a new entry would exceed this, remove or condense
   an existing entry first.
5. **When a memory entry gets promoted to a spec**, delete it from memory immediately.
