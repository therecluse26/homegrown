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

> To be filled in as the project scaffolds. Check `Makefile` or `justfile` at project root.

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
