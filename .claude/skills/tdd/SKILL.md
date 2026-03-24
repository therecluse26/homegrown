---
name: tdd
description: "Red/Green/Refactor TDD workflow against spec requirements"
argument-hint: "§12.3 or 'requirement in plain English'"
user-invocable: true
---

# TDD Workflow — Red / Green / Refactor

You are now operating in strict TDD mode. You MUST follow the red/green/refactor cycle
exactly as described below. Violations of this workflow are bugs.

## Argument Handling

The user invoked `/tdd` with an argument: `$ARGUMENTS`

- If the argument starts with `§`, it is a spec section reference. Read that section from
  `specs/SPEC.md` (or the relevant `specs/domains/*.md` file).
- If the argument is a quoted string, treat it as a plain-English requirement description.
  Search `specs/SPEC.md` and `specs/domains/*.md` for the matching requirement.
- If no argument was provided, ask the user which spec section or requirement to target.

---

## Phase 0 — Orientation (MUST complete before writing any code)

1. **Read the spec.** Load the relevant section from `specs/SPEC.md` or `specs/domains/`.
2. **Read the coding standards.** Load `specs/CODING_STANDARDS.md` §2.8 (Testing).
3. **Read existing domain code.** For the target domain (`internal/{domain}/`), read:
   - `ports.go` — interfaces you will test against
   - `service.go` — current implementation (to understand what exists)
   - `domain/errors.go` — sentinel errors and custom error types
   - `*_test.go` — existing test patterns and mocks
4. **Extract testable requirements.** From the spec section, produce a numbered checklist of
   every independently testable behavior. Each item should be a single assertion-worthy fact.
   Format:
   ```
   Requirements from [S§X.Y]:
   [ ] 1. <behavior description>
   [ ] 2. <behavior description>
   ...
   ```
5. **Present the checklist and STOP.** Wait for the user to confirm, reorder, or prune the
   list before writing any code.

---

## Phase 1 — RED (One Failing Test)

### Rules
- Write exactly ONE test function: `Test{Function}_{Behavior}`
- The test MUST target the PUBLIC interface (service method or domain function from `ports.go`)
- The test MUST assert on SPECIFIC outcomes:
  - Error types via `errors.Is` / `errors.As` (see `internal/method/service_test.go` for examples)
  - Return values with concrete expected data
  - State changes observable through the public interface
- The test MUST include a spec reference comment: `// [S§X.Y]`
- Use hand-written function-pointer stubs matching `ports.go` interfaces (the pattern from
  `internal/learn/mock_test.go`):
  ```go
  type stubRepo struct {
      createFn func(ctx context.Context, ...) (Thing, error)
  }
  func (s *stubRepo) Create(ctx context.Context, ...) (Thing, error) {
      if s.createFn != nil {
          return s.createFn(ctx, ...)
      }
      panic("Create not stubbed")
  }
  ```
- For family-scoped operations, create test scopes using the pattern from
  `internal/onboard/service_test.go`:
  ```go
  scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.Must(uuid.NewV7())})
  ```

### Forbidden During RED
- MUST NOT write or modify any production code (no `service.go`, `ports.go`, `domain/` changes)
- MUST NOT write more than one test function
- MUST NOT add test helpers that mask the assertion (keep the test readable)

### After Writing the Test
1. Run ONLY the new test: `go test -run TestXxx ./internal/{domain}/`
2. Show the failure output to the user
3. **STOP and wait for user confirmation before proceeding to GREEN**

---

## Phase 2 — GREEN (Minimum Implementation)

### Rules
- Write the MINIMUM production code needed to make the failing test pass
- "Minimum" means: if you can hard-code a return value and the test passes, that IS the
  correct GREEN implementation. The next RED test will force generalization.
- Changes go in the appropriate production files (`service.go`, `domain/errors.go`,
  `ports.go`, `domain/types.go`, etc.)

### Forbidden During GREEN
- MUST NOT modify any test code (no touching `*_test.go`)
- MUST NOT add functionality beyond what the single failing test requires
- MUST NOT add error handling for cases not yet tested
- MUST NOT refactor — that's the next phase

### After Implementation
1. Run the single test: `go test -run TestXxx ./internal/{domain}/`
2. Run the full domain test suite: `go test ./internal/{domain}/...`
3. Show output to the user
4. **STOP and wait for user confirmation before proceeding to REFACTOR**

---

## Phase 3 — REFACTOR (Clean Up)

### Rules
- Structural improvements only: extract helpers, improve naming, remove duplication,
  reorder code for readability
- You MAY add edge-case tests for the SAME requirement just tested (e.g., boundary values,
  nil inputs) — but these are refinements, not new requirements
- MUST NOT change observable behavior (all existing tests must still pass)
- MUST NOT implement new requirements from the checklist

### After Refactoring
1. Run the full domain test suite: `go test ./internal/{domain}/...`
2. Run the linter: `golangci-lint run ./internal/{domain}/...`
3. Show output to the user
4. **STOP and wait for user confirmation**
5. Announce the NEXT unchecked requirement from the checklist

---

## Cycle Control

After completing one full RED/GREEN/REFACTOR cycle:
- Mark the requirement as done: `[x]` in the checklist
- Announce the next requirement and wait for user confirmation

### Pace Modes
- **Default (step):** Stop after each phase (RED, GREEN, REFACTOR) and wait for user
- **`auto`:** If the user says "auto", complete one full R/G/R cycle without stopping,
  but MUST stop at requirement boundaries (between checklist items)
- **`auto N`:** Complete N full cycles, stopping after the Nth refactor

The user can say "stop" or "pause" at any time to return to step mode.

---

## Output Format

At each stop point, display:

```
═══ TDD [{PHASE}] ═════════════════════════════════
Requirement {N}/{total}: {description}
Status: {PASS|FAIL} — {test count} tests, {duration}
═══════════════════════════════════════════════════
```

Then wait for user input.

---

## Reference: Project Test Patterns

| Pattern | Example File | Notes |
|---------|-------------|-------|
| Function-pointer stubs | `internal/learn/mock_test.go` | Struct with `*Fn` fields, nil-check + panic |
| Error assertions | `internal/method/service_test.go` | `errors.As` for custom types, `errors.Is` for sentinels |
| FamilyScope in tests | `internal/onboard/service_test.go` | `shared.NewFamilyScopeFromAuth(&shared.AuthContext{...})` |
| Domain errors | `internal/{domain}/domain/errors.go` | Sentinel vars (`ErrNotFound`) + struct types (`*MethodError`) |
| Coding standards | `specs/CODING_STANDARDS.md` §2.8 | Table-driven tests preferred, unit vs integration distinction |
