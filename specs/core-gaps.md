# Core Gap Analysis — Homegrown Academy

> **Generated:** 2026-03-26 · **Branch:** `feature/core-code` · **Commit:** `b866261`
>
> Comprehensive audit of all 18 domain specs (00-core through 17-planning) versus
> actual implementation. This document catalogs every gap, broken stub, missing phase,
> and quality issue discovered.

---

## Executive Summary

| Domain | Status | Key Gap |
|--------|--------|---------|
| 00-core | Phase 1 complete | Shared utilities only; no gaps |
| 01-iam | Phase 1 complete | 8 co-parent/deletion endpoints + 4 student session endpoints not started |
| 02-method | Phase 1 complete | 2 Phase 2 endpoints missing (methodology context + student methodology patch) |
| 03-discover | Phase 1 complete | Content detail endpoint missing (`GET /v1/discovery/content/:slug`) |
| 04-onboard | Phase 1 complete | 2 Phase 2 endpoints missing (complete roadmap item, restart onboarding) |
| 05-social | Phase 1 complete | Blocked by 3 deferred event subscriptions |
| 06-learn | Phase 1 complete | Progress snapshot job missing; Phase 2 tables exist but no models/handlers |
| 07-mkt | Phase 1 complete | **Zero test files**; background job + event wiring missing |
| 08-notify | Phase 1 complete | 2 billing event handlers completely missing |
| 09-media | Phase 1 complete | 3 endpoints missing (delete, list, reprocess) |
| 10-billing | Phase 1 complete | No handler tests; pause/resume subscription missing |
| 11-safety | Phase 1 complete | **NCMEC noop adapter — federal legal obligation**; Phase 2 not started |
| 12-search | Phase 1 complete | Typesense adapter missing; `/suggestions` returns 501 |
| 13-recs | Phase 1+2 complete | Phase 3 notification dispatch not implemented |
| 14-comply | Phase 1 complete | `RequirePremium` not enforced at HTTP layer; learn adapter stub |
| 15-lifecycle | **NOT WIRED** | **No handler, no repository, not in AppState — ~4K lines of dead code** |
| 16-admin | Phase 1+2 complete | All 7 cross-domain adapters are stubs (several wireable today) |
| 17-plan | Phase 1+2 complete | IAM stub bypasses family ownership; calendar shows only schedule items |

**Domains with zero test files:** 07-mkt
**Domains with no handler tests:** 10-billing

---

## 1 · Critical Issues (Runtime-Broken / Security / Legal)

### CRIT-1 · 15-lifecycle: Entire domain unreachable

`internal/lifecycle/` contains `errors.go`, `events.go`, `models.go`, `ports.go`,
`service.go`, `service_test.go`, and `mock_test.go` — but **no `handler.go`** and **no
`repository.go`**. The domain is not instantiated in `main.go`; the only reference is
through `adminLifecycleStub{}`.

**Impact:** GDPR Article 17 (right to erasure), COPPA §312.10 (parental deletion rights),
and data export (GDPR Article 20) are entirely unreachable. Families cannot request
deletion or export of their data.

### CRIT-2 · 11-safety: NoopThornAdapter — no CSAM detection or NCMEC reporting

`internal/safety/noop_adapters.go` defines `NoopThornAdapter{}`:

- `ScanCsam` always returns `&CsamScanResult{IsCSAM: false}` — no content is ever flagged.
- `SubmitNcmecReport` returns `nil, nil` — a nil pointer dereference risk if callers don't
  check, and no report is ever filed.

**Impact:** Federal legal obligation under 18 U.S.C. § 2258A requires electronic service
providers to report CSAM to NCMEC. A noop adapter means the platform cannot comply.

### CRIT-3 · 17-plan: planIamStub bypasses family ownership

```go
func (planIamStub) StudentBelongsToFamily(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
    return true, nil // ALWAYS TRUE
}
```

**Impact:** Any authenticated parent can add any student to their schedule, regardless of
family membership. This is a horizontal privilege escalation — a parent in Family A could
create schedule items for a student in Family B.

### CRIT-4 · 16-admin: adminHealthStub lies about system health

```go
func (adminHealthStub) CheckAll(_ context.Context) []admin.ComponentHealth {
    return []admin.ComponentHealth{
        {Name: "database", Status: "healthy"},
        {Name: "redis", Status: "healthy"},
        {Name: "r2", Status: "healthy"},
        {Name: "kratos", Status: "healthy"},
    }
}
```

**Impact:** Admin health dashboard permanently shows all-green. Operators cannot detect
outages, degraded services, or connection failures through the admin interface.

### CRIT-5 · adminBillingStub returns nil pointer

```go
func (adminBillingStub) GetSubscriptionInfo(_ context.Context, _ uuid.UUID) (*admin.AdminSubscriptionInfo, error) {
    return nil, nil
}
```

**Impact:** If admin code dereferences the return value without a nil check, this will
panic at runtime. The `nil, nil` return violates the Go convention of returning an error
when the result is unusable.

---

## 2 · Per-Domain Detailed Findings

### 00-core (Shared Utilities)

No gaps. Shared packages (`family_scope`, `events`, `middleware`, etc.) are stable and used
across all domains.

---

### 01-iam (Identity & Access Management)

**Phase 2 endpoints not started (12 total):**

Co-parent management (8):
- `POST /v1/families/co-parents/invite` — InviteCoParent
- `DELETE /v1/families/co-parents/invite/:id` — CancelInvite
- `POST /v1/families/co-parents/invite/:id/accept` — AcceptInvite
- `DELETE /v1/families/co-parents/:id` — RemoveCoParent
- `POST /v1/families/co-parents/transfer-primary` — TransferPrimaryParent
- `POST /v1/families/coppa/withdraw` — WithdrawCoppaConsent
- `POST /v1/families/deletion` — RequestFamilyDeletion
- `DELETE /v1/families/deletion` — CancelFamilyDeletion

Student sessions (4):
- `POST /v1/families/students/:id/sessions`
- `GET /v1/families/students/:id/sessions`
- `DELETE /v1/families/students/:id/sessions/:sid`
- `GET /v1/session/me` (student-scoped)

**Missing event types (3):**
- `CoParentRemoved` — blocks social domain event subscription
- `PrimaryParentTransferred` — blocks billing domain event subscription
- `CoParentAdded`

---

### 02-method (Methodology)

**Phase 2 endpoints missing (2):**
- `GET /v1/families/methodology-context` — service method exists, HTTP route not wired
- `PATCH /v1/families/students/:id/methodology` — model exists, no handler or service method

---

### 03-discover (Discovery / Curriculum Catalog)

**Phase 2 endpoints missing (1):**
- `GET /v1/discovery/content/:slug` — discovery_content table is seeded, but no API layer
  (handler, service method, or repository query) exists to serve individual content items.

---

### 04-onboard (Onboarding)

**Phase 2 endpoints missing (2):**
- `PATCH /v1/onboarding/roadmap/:item_id/complete` — mark a roadmap item as completed
- `POST /v1/onboarding/restart` — restart the onboarding flow

---

### 05-social (Social / Community)

**Blocked by deferred events (3):**
- `iam.CoParentRemoved` → source event type doesn't exist
- `learn.MilestoneAchieved` → handler exists, subscription deferred
- `iam.FamilyDeletionScheduled` → wired for comply/recs/search but deferred for social

No additional endpoint gaps identified.

---

### 06-learn (Learning / Curriculum)

**Phase 2 gaps:**
- Progress snapshot background job — table and repository exist, but no periodic task or
  scheduler invocation
- Phase 2 tables DDL exists but no models/handlers: assessments, projects, grading scales

**Spec inconsistencies:**
- §6 uses `ReplaceQuestions` / `ReplaceItems` but code uses `SetQuestions` / `SetItems`
- §7 missing `GetStudentName` method documentation
- `AssignmentModel` vs `StudentAssignmentModel` naming inconsistency in repository

---

### 07-mkt (Marketplace)

**Quality: Zero test files** — the only domain with no `*_test.go` files at all.

**Missing implementations:**
- `RefreshAutoSection` background job — completely unimplemented
- `mkt.PurchaseCompleted → learn` event not wired — purchased marketplace content doesn't
  unlock in the learn domain

**Response contract deviations:**
- Response uses `onboarding_url` key where spec says `url`
- Some endpoints return 201 where spec says 200

---

### 08-notify (Notifications)

**Missing event handlers (2):**
- `SubscriptionChangedHandler` — not present in `event_handlers.go`
- `PayoutCompletedHandler` — not present in `event_handlers.go`

These handlers are referenced in deferred comments in `main.go` (as
`billing.SubscriptionCreated` and `billing.SubscriptionCancelled` subscriptions) but the
handler implementations don't exist.

**Spec gap:** `email_status` table is used in implementation but undocumented in the
08-notify spec.

---

### 09-media (Media / File Upload)

**Missing endpoints (3):**
- `DELETE /v1/media/uploads/:id` — service method exists, no handler route
- `GET /v1/media/uploads` — absent from service interface and handler
- `POST /v1/media/uploads/:id/reprocess` — absent entirely

---

### 10-billing (Billing / Subscriptions)

**Quality: No `handler_test.go`** — HTTP handler layer is untested.

**Missing features:**
- Pause/resume subscription — adapter methods exist, no service or handler
- `iam.PrimaryParentTransferred` event handling — event type doesn't exist yet (see 01-iam)

---

### 11-safety (Safety / Content Moderation)

**Critical:** See CRIT-2 above (NoopThornAdapter).

**Phase 2 not started:**
- ML grooming detection
- Parental controls
- Granular admin roles
- `ExpireSuspensionsJob` — not implemented

**Missing event type:** `safety.ContentFlagged` event doesn't exist, blocking mkt content
moderation wiring.

---

### 12-search (Search)

**Phase 2 not started:**
- No Typesense adapter (Phase 2 search backend)
- No `search_index_state` migration
- `GET /v1/search/suggestions` returns 501 Not Implemented

---

### 13-recs (Recommendations)

Phase 1+2 complete (23 unit tests, migration 20).

**Phase 3:** Notification dispatch for new recommendations not implemented.

---

### 14-comply (Compliance / Portfolio)

**Gaps:**
- `RequirePremium` enforcement not visible at HTTP middleware layer — premium-gated
  endpoints may not actually check subscription status
- `learnForComply` adapter is a stub — portfolio item data loading from learn domain returns
  errors
- `mediaForComply` adapter is a stub — PDF upload for portfolios/transcripts returns errors

---

### 15-lifecycle (Data Lifecycle / GDPR)

**Critical:** See CRIT-1 above. Entire domain unreachable.

Files present: `errors.go`, `events.go`, `models.go`, `ports.go`, `service.go`,
`service_test.go`, `mock_test.go`

Missing: `handler.go`, `repository.go`, AppState registration, route group, event
subscriptions.

---

### 16-admin (Administration)

Phase 1+2 complete (53 unit tests, migration 23).

**All 7 cross-domain adapters are stubs:**

| Stub | Methods | Wireable Today? |
|------|---------|-----------------|
| `adminIamStub` | 4 | Yes — iam service could bridge |
| `adminSafetyStub` | 7 | Yes — safety service fully ready |
| `adminBillingStub` | 1 | Partially — returns `nil, nil` (CRIT-5) |
| `adminMethodStub` | 2 | Yes — method service ready |
| `adminLifecycleStub` | 3 | No — blocked on lifecycle domain wiring (CRIT-1) |
| `adminHealthStub` | 1 | No — needs real health check implementation (CRIT-4) |
| `adminJobInspectorStub` | 3 | No — needs asynq inspection API |

---

### 17-plan (Planning / Scheduling)

Phase 1+2 complete (47 unit tests, migration 24).

**Security:** See CRIT-3 above (planIamStub).

**Stub-limited features:**
- `planLearnStub` returns empty — unified calendar omits learning activities
- `planComplyStub` returns empty — unified calendar omits compliance deadlines
- `planSocialStub` returns empty — unified calendar omits social events
- `GET /planning/calendar/pdf` — not implemented

---

## 3 · Cross-Cutting Issues

### 3.1 · Deferred Event Subscriptions (11 total in main.go)

**Blocked by missing source event types (3):**

| Deferred Subscription | Missing Event |
|-----------------------|---------------|
| `iam.CoParentRemoved → social` | `iam.CoParentRemoved` not defined |
| `iam.PrimaryParentTransferred → billing` | `iam.PrimaryParentTransferred` not defined |
| `safety.ContentFlagged → mkt` | `safety.ContentFlagged` not defined |

**No technical blocker — wiring deferred (8):**

| Source Event | Target Domain | Handler |
|-------------|---------------|---------|
| `learn.MilestoneAchieved` | social | `NewMilestoneAchievedHandler` |
| `iam.FamilyDeletionScheduled` | social | `NewFamilyDeletionScheduledHandler` |
| `iam.FamilyDeletionScheduled` | mkt | `NewFamilyDeletionScheduledHandler` |
| `iam.FamilyDeletionScheduled` | learn | `NewFamilyDeletionScheduledHandler` |
| `iam.FamilyDeletionScheduled` | billing | `NewFamilyDeletionScheduledHandler` |
| `mkt.PurchaseCompleted` | learn | `NewPurchaseCompletedHandler` |
| `billing.SubscriptionCreated` | notify | `NewSubscriptionCreatedHandler` |
| `billing.SubscriptionCancelled` | notify | `NewSubscriptionCancelledHandler` |

Note: `iam.FamilyDeletionScheduled` is already wired for comply, recs, and search — but
deferred for social, mkt, learn, and billing.

### 3.2 · Test Coverage Gaps

| Domain | Service Tests | Handler Tests | Notes |
|--------|:------------:|:-------------:|-------|
| 07-mkt | None | None | **Zero test files** |
| 10-billing | Yes | None | Missing `handler_test.go` |
| 15-lifecycle | Yes | N/A | No handler exists to test |

### 3.3 · Frontend

- `frontend/src/features/` contains only `.gitkeep` — no feature UI code exists
- `frontend/src/components/ui/` has design system shell only
- `frontend/src/api/generated/` and `api/client.ts` exist (API client ready)
- No pages, routes, or feature modules have been implemented

### 3.4 · Spec-vs-Code Inconsistencies

| Location | Spec Says | Code Uses |
|----------|-----------|-----------|
| 06-learn §6 | `ReplaceQuestions` / `ReplaceItems` | `SetQuestions` / `SetItems` |
| 06-learn §7 | (not documented) | `GetStudentName` method exists |
| 07-mkt responses | `url` key | `onboarding_url` key |
| 07-mkt responses | 200 status | 201 status (some endpoints) |
| 08-notify | (not documented) | `email_status` table in use |
| 06-learn repo | `AssignmentModel` | `StudentAssignmentModel` |

---

## 4 · Priority-Ranked Remediation

### P0 — Legal / Compliance (do before any public beta)

1. **Wire 15-lifecycle domain** — implement `handler.go` + `repository.go`, register in
   `main.go`, connect event subscriptions. GDPR/COPPA deletion and export must be
   functional.
2. **Replace NoopThornAdapter** with real Thorn/PhotoDNA integration (or at minimum, a
   logging adapter that queues manual review). NCMEC reporting under 18 U.S.C. § 2258A is
   a federal obligation.
3. **Fix planIamStub** — wire real IAM service's `StudentBelongsToFamily` to prevent
   cross-family schedule manipulation.

### P1 — Runtime Correctness

4. **Fix adminBillingStub** — return a zero-value struct instead of `nil, nil` to prevent
   nil pointer panics.
5. **Wire adminHealthStub** to real health checks (database ping, Redis ping, R2
   connectivity, Kratos readiness).
6. **Wire the 8 non-blocked deferred event subscriptions** — especially
   `FamilyDeletionScheduled` for social/mkt/learn/billing (data retention issue).
7. **Wire admin cross-domain adapters** that have real implementations ready: adminIamStub,
   adminSafetyStub, adminMethodStub.

### P2 — Feature Completeness

8. **07-mkt: Add test files** — only domain with zero tests.
9. **10-billing: Add handler tests.**
10. **Implement missing Phase 2 endpoints** per domain (01-iam co-parent flow is highest
    value).
11. **Implement missing event handlers** in 08-notify (SubscriptionChangedHandler,
    PayoutCompletedHandler).
12. **Define missing event types** (iam.CoParentRemoved, iam.PrimaryParentTransferred,
    safety.ContentFlagged) to unblock 3 deferred subscriptions.

### P3 — Polish / Phase 2+

13. Fix spec-vs-code inconsistencies (§3.4 above).
14. Implement 09-media missing endpoints (delete, list, reprocess).
15. Implement 12-search Typesense adapter and suggestions endpoint.
16. Implement 06-learn progress snapshot background job.
17. Begin frontend feature development (`features/` directory).
18. Implement 17-plan calendar PDF export.
