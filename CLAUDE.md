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

---

## Quality Gates (Must Pass Before Every Commit)

```bash
golangci-lint run              # Zero warnings
go test ./...                  # All tests pass
npm run type-check             # Zero TypeScript errors (in frontend/)
```

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
