# Homegrown Academy — Claude Development Guide

Homegrown Academy is a privacy-first platform for homeschooling families, built with a
Rust/Axum backend, React SPA frontend, and PostgreSQL database. It supports multiple
homeschooling methodologies (Charlotte Mason, Classical, Unschooling, etc.) as runtime
configuration — not code branches.

---

## Before Writing Any Code

1. **Read `specs/CODING_STANDARDS.md`** — the authoritative rulebook for this project.
   Violations are treated as bugs, not style preferences.
2. **Read the relevant section(s) of `specs/ARCHITECTURE.md`** for any domain you are
   working in. It explains *why* each decision was made.
3. **Read `specs/SPEC.md` §[n]** for the requirements in your domain before implementing.
4. **Check `src/shared/`** for existing utilities before writing new ones.

---

## Coding Standards & Architecture

All implementation work MUST follow `specs/CODING_STANDARDS.md`. That document is the
authoritative source for:

- Module structure and layer responsibilities
- Error handling rules
- Type safety requirements
- Database patterns (family-scoped queries, migration rules, entity generation)
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
| **Rust** | Zero `.unwrap()` / `.expect()` in production code. Use `?` and `thiserror`. |
| **TypeScript** | Strict mode enabled. Zero `any`. All API types from `src/api/generated/` only. |
| **Database** | Every user-data query MUST be family-scoped via `FamilyScope`. Migrations are append-only. Regenerate SeaORM entities after every migration before writing queries. |
| **Methodology** | MUST NOT branch on methodology name in code. Use config lookup via `method::` service. |
| **Privacy** | Never store GPS coordinates. Student resources require parent ownership check. Never log PII or tokens. |
| **File uploads** | Validate magic bytes, not just file extension. |
| **API errors** | Never expose internal error details in responses. Log internally; return generic message. |
| **Generated files** | `src/api/generated/` and `src/{domain}/entities/` are generated — never hand-edit. |

---

## Quality Gates (Must Pass Before Every Commit)

```bash
cargo clippy -- -D warnings    # Zero warnings
cargo test                     # All tests pass
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
