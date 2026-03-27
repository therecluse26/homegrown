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
| 01-iam | Phase 1 complete | 8 co-parent/deletion endpoints + 4 student session endpoints not started; COPPA micro-charge stub accepts any token |
| 02-method | Phase 1 complete | 2 Phase 2 endpoints missing (methodology context + student methodology patch) |
| 03-discover | Phase 1 complete | Content detail endpoint missing (`GET /v1/discovery/content/:slug`) |
| 04-onboard | Phase 1 complete | 2 Phase 2 endpoints missing (complete roadmap item, restart onboarding) |
| 05-social | Phase 1 complete | Blocked by 3 deferred event subscriptions |
| 06-learn | Phase 1 complete | Progress snapshot job missing; Phase 2 tables exist but no models/handlers |
| 07-mkt | Phase 1 complete | **Zero test files**; creator API inaccessible (`RequireCreator` stub); background job + event wiring missing |
| 08-notify | Phase 1 complete | 4 billing event handlers completely missing |
| 09-media | Phase 1 complete | 3 endpoints missing (delete, list, reprocess) |
| 10-billing | Phase 1 complete | No handler tests; pause/resume subscription missing |
| 11-safety | Phase 1 complete | **NCMEC noop adapter — federal legal obligation**; Phase 2 not started |
| 12-search | Phase 1 complete | Typesense adapter missing; `/suggestions` returns 501 |
| 13-recs | Phase 1+2 complete | Phase 3 notification dispatch not implemented |
| 14-comply | Phase 1 complete | learn adapter stub + media adapter stub return errors |
| 15-lifecycle | **NOT WIRED** | **No handler, no repository, no migration, not in AppState — ~4K lines of dead code** |
| 16-admin | Phase 1+2 complete | All 7 cross-domain adapters are stubs (several wireable today) |
| 17-plan | Phase 1+2 complete | IAM stub bypasses family ownership; calendar shows only schedule items |

**Domains with zero test files:** 07-mkt
**Domains with no handler tests:** 07-mkt, 10-billing, 13-recs, 14-comply, 16-admin, 17-plan, 05-social

---

## 1 · Critical Issues (Runtime-Broken / Security / Legal)

### CRIT-1 · 15-lifecycle: Entire domain unreachable

`internal/lifecycle/` contains `errors.go`, `events.go`, `models.go`, `ports.go`,
`service.go`, `service_test.go`, and `mock_test.go` — but **no `handler.go`** and **no
`repository.go`**. The domain is not instantiated in `main.go`; the only reference is
through `adminLifecycleStub{}`.

Additionally, there is **no database migration** for the lifecycle domain. The migration
sequence jumps from migration 24 (17-plan) directly with nothing for lifecycle. Even after
writing `handler.go` and `repository.go`, every database operation will fail until a
migration creates the required tables (e.g., `data_export_requests`, `deletion_requests`,
`recovery_requests`). See also CRIT-6.

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

### CRIT-6 · 15-lifecycle: No database migration

The lifecycle domain has no migration file. The sequence goes: migration 24 (17-plan tables)
→ nothing. Even if `handler.go` and `repository.go` are implemented (CRIT-1), they will
immediately fail on any database operation because the backing tables don't exist.

Required tables (at minimum): `data_export_requests`, `deletion_requests`,
`recovery_requests`. The migration must also add foreign key relationships to the `families`
table and appropriate indexes for status/expiry lookups.

**Impact:** Blocks any functional test or deployment of the lifecycle domain.

---

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

### CRIT-7 · `UNSUBSCRIBE_SECRET` defaults to known public string

`internal/config/config.go:237` — `envOrDefault("UNSUBSCRIBE_SECRET", "notify-dev-secret")`

This secret signs email unsubscribe tokens via HMAC. If `UNSUBSCRIBE_SECRET` is not set in
production, the signing key is `"notify-dev-secret"` — a string visible in this repository.
Anyone who reads the source can forge valid unsubscribe tokens for any user, silently
removing them from all notification lists without their knowledge.

**Impact:** Forged unsubscribe tokens allow an attacker to suppress email notifications for
any user account — including password-reset and safety-alert emails. **Priority: P1.**

---

### CRIT-8 · WebSocket CSWSH — `CheckOrigin` unconditionally returns `true`

`internal/social/websocket.go:22-26` — `CheckOrigin` always returns `true` with the comment
"Origin check delegated to CORS middleware." However, for WebSocket upgrade requests (HTTP
`GET` with `Upgrade: websocket`), browsers do not send CORS preflights. CORS headers are
informational only and do not block the WebSocket upgrade itself. A malicious page can open
a WebSocket connection to the server, and the browser will automatically attach the victim's
session cookie. The server accepts it unconditionally.

**Impact:** Cross-Site WebSocket Hijacking (CSWSH). An attacker's page can open an
authenticated WebSocket as the victim, inject real-time messages, or extract conversation
data without the victim's knowledge. **Priority: P1.**

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

**COPPA credit card verification stub:**
`internal/iam/service.go:299-301` — `SubmitCoppaConsent` validates that `method` and
`token` are non-empty but performs no actual Stripe micro-charge. A `TODO(billing::)` marks
the unimplemented call. Any non-empty string passes the verification check — a fabricated
token is indistinguishable from a real one. Credit card micro-charge is an FTC-recognized
COPPA parental consent method; the current stub accepts any input.

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
- `PATCH /v1/onboarding/roadmap/:item_id/complete` — mark a specific roadmap item as
  completed (distinct from the existing `/complete` route)
- `POST /v1/onboarding/restart` — restart the onboarding flow (distinct from the existing
  `/skip` route)

Note: `POST /v1/onboarding/complete` (completeWizard) and `POST /v1/onboarding/skip`
(skipWizard) **are implemented**. The two missing endpoints are for per-item progress and
flow restart — different operations with different semantics.

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

**`RequireCreator` middleware always returns 403:**
`internal/middleware/extractors.go:63-72` — `RequireCreator` always returns `ErrForbidden`
(`return nil, shared.ErrForbidden()`) with no mkt_creators table lookup wired. Every
creator-facing mkt endpoint (upload curriculum resources, publish listings, etc.) returns
403 for all users. The creator API is entirely inaccessible at runtime.

**Response contract deviations:**
- Response uses `onboarding_url` key where spec says `url`
- Some endpoints return 201 where spec says 200

---

### 08-notify (Notifications)

**Missing event handlers (4):**
- `SubscriptionCreatedHandler` — not implemented (deferred in main.go at `billing.SubscriptionCreated`)
- `SubscriptionChangedHandler` — not implemented
- `SubscriptionCancelledHandler` — not implemented (deferred in main.go at `billing.SubscriptionCancelled`)
- `PayoutCompletedHandler` — not implemented

All four billing-event handler *functions* are absent from `event_handlers.go`. The
deferred subscriptions in `main.go` reference `billing.SubscriptionCreated` and
`billing.SubscriptionCancelled` by name but there are no implementing handler functions for
any of these four events.

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
- `learnForComply` adapter is a stub — portfolio item data loading from learn domain returns
  errors
- `mediaForComply` adapter is a stub — PDF upload for portfolios/transcripts returns errors

**Portfolio and transcript PDFs contain literal placeholder bytes:**
- `internal/comply/jobs.go:217` — `pdfBytes := []byte("placeholder PDF — gofpdf integration pending")`
- `internal/comply/jobs.go:333` — same for transcript PDF jobs

Both jobs call `mediaSvc.RequestUpload` with a literal ASCII string as the "PDF" content.
Families who download their portfolio or transcript receive a corrupt/garbage file. The
comply domain's primary deliverable — compliance documentation — is broken at runtime, not
merely feature-incomplete.

Note: `RequirePremium` is enforced via `middleware.RequirePremium(c)` in every endpoint
handler in `internal/comply/handler.go` (43 call sites). The premium check IS present at
the HTTP layer.

---

### 15-lifecycle (Data Lifecycle / GDPR)

**Critical:** See CRIT-1 and CRIT-6 above. Entire domain unreachable.

Files present: `errors.go`, `events.go`, `models.go`, `ports.go`, `service.go`,
`service_test.go`, `mock_test.go`

Missing: `handler.go`, `repository.go`, **database migration** (no lifecycle tables exist),
AppState registration, route group, event subscriptions.

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
- `planLearnStub.LogActivity` **returns an error** (`fmt.Errorf("not yet implemented")`) —
  `POST /planning/schedule-items/:id/log` is visibly broken (always 500), not merely empty
- `planLearnStub` returns empty for calendar queries — unified calendar omits learning activities
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

**Additional runtime wiring gaps:**

- **PostmarkEmailAdapter not wired** (`main.go` line ~676 has a `// TODO: Wire
  PostmarkEmailAdapter when cfg.PostmarkServerToken != ""`). The notify domain runs with a
  noop/stub email sender. All notification handlers fire and templates render, but **no
  emails are ever delivered**. This is a silent data-loss issue affecting password reset,
  onboarding complete, milestone achieved, purchase confirmed, and every other notification.
  **Priority: P1.**

- **RevokeSessions not wired for safety** (`main.go` line ~815 has `// FUTURE: wire real
  IamServiceForSafety when iam:: exposes RevokeSessions`). When an account is suspended or
  banned via the safety moderation flow, active sessions are **not revoked**. The suspended
  user remains authenticated until their session naturally expires. **Priority: P1.**

### 3.2 · Test Coverage Gaps

| Domain | Service Tests | Handler Tests | Notes |
|--------|:------------:|:-------------:|-------|
| 07-mkt | None | None | **Zero test files** |
| 10-billing | Yes | None | Missing `handler_test.go` |
| 13-recs | Yes | None | Missing `handler_test.go` |
| 14-comply | Yes (6 files) | None | Missing `handler_test.go` |
| 16-admin | Yes | None | Missing `handler_test.go` |
| 17-plan | Yes | None | Missing `handler_test.go` |
| 05-social | Yes (5 files) | None | Missing `handler_test.go` |
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

### 3.5 · Cross-Domain Interface Violations

**`GetStudentName` not in any published IAM interface:**

`internal/comply/ports.go` declares `IamServiceForComply.GetStudentName(...)`. However,
`internal/iam/ports.go` has no `GetStudentName` method on any exported interface. Instead,
`cmd/server/main.go` wires an anonymous closure that performs a raw DB lookup via
`BypassRLSTransaction`.

This is:
- Not part of any published IAM service contract
- Not covered by any test
- A maintenance trap — if the IAM schema changes (e.g., student name column renamed), this
  anonymous closure silently breaks with no compile-time error

The correct fix is to add `GetStudentName` to the IAM service interface, implement it in
the IAM service, and update the wiring.

---

### 3.6 · Config / Deployment Safety

**`OBJECT_STORAGE_PUBLIC_URL` defaults to `"https://media.localhost"`** (`internal/config/config.go:233`).
If omitted in a production environment, every generated media URL (images, attachments,
upload links) resolves to a non-existent hostname. All media references silently break with
no runtime error or warning.

**`kratos/kratos.yml` hardcodes webhook secrets** at lines 46 and 63
(`value: dev-webhook-secret-change-in-production`). These are the Kratos-side auth headers
for post-registration and post-login webhooks. The application validates against the
`AUTH_WEBHOOK_SECRET` env var, but the Kratos config must be updated to match at deploy
time. There is no documented mechanism (template variable, env substitution) for
externalizing these values. If the dev config is used in production unchanged, Kratos
webhooks can be spoofed from any origin.

**`kratos/kratos.yml:90` sets `same_site: Lax`** for session cookies. For a platform
storing children's educational records, `Strict` is required to prevent CSRF via
cross-origin navigation. `Lax` permits session cookies to be sent on top-level cross-site
GET navigations, broadening the CSRF attack surface.

---

## 4 · Priority-Ranked Remediation

### P0 — Legal / Compliance (do before any public beta)

1. **Wire 15-lifecycle domain** — implement `handler.go` + `repository.go`, **write
   database migration** (CRIT-6, tables: `data_export_requests`, `deletion_requests`,
   `recovery_requests`), register in `main.go`, connect event subscriptions. GDPR/COPPA
   deletion and export must be functional.
2. **Replace NoopThornAdapter** with real Thorn/PhotoDNA integration (or at minimum, a
   logging adapter that queues manual review). NCMEC reporting under 18 U.S.C. § 2258A is
   a federal obligation.
3. **Fix planIamStub** — wire real IAM service's `StudentBelongsToFamily` to prevent
   cross-family schedule manipulation.

### P1 — Runtime Correctness

4. **Wire PostmarkEmailAdapter** — replace noop email sender so notifications are actually
   delivered. All user-facing notification flows (password reset, onboarding complete, etc.)
   are silently broken today.
5. **Wire RevokeSessions for safety suspensions** — expose `RevokeSessions` on the IAM
   service interface and wire it into the safety domain so suspended users lose access
   immediately.
6. **Rotate `UNSUBSCRIBE_SECRET`** — change the default from the hardcoded
   `"notify-dev-secret"` to a required env var with no fallback; email unsubscribe tokens
   are forgeable with the current default (CRIT-7).
7. **Fix WebSocket CSWSH** — replace unconditional `CheckOrigin: true` in
   `internal/social/websocket.go` with origin validation against an explicit allowed-origin
   list derived from `cfg.AppURL` (CRIT-8).
8. **Fix planLearnStub.LogActivity** — the stub returns `fmt.Errorf("not yet implemented")`,
   making `POST /planning/schedule-items/:id/log` always return a 500 error.
9. **Fix adminBillingStub** — return a zero-value struct instead of `nil, nil` to prevent
   nil pointer panics.
10. **Wire adminHealthStub** to real health checks (database ping, Redis ping, R2
    connectivity, Kratos readiness).
11. **Wire the 8 non-blocked deferred event subscriptions** — especially
    `FamilyDeletionScheduled` for social/mkt/learn/billing (data retention issue).
12. **Wire admin cross-domain adapters** that have real implementations ready: adminIamStub,
    adminSafetyStub, adminMethodStub.
13. **Fix GetStudentName cross-domain wiring** — add `GetStudentName` to IAM service
    interface and replace the anonymous DB-closure in `main.go` (§3.5).

### P2 — Feature Completeness

14. **Fix `RequireCreator` middleware** — wire mkt_creators table lookup in
    `internal/middleware/extractors.go` so the creator-facing mkt API is accessible (07-mkt,
    §2).
15. **Implement gofpdf rendering** in comply PDF jobs — replace the literal placeholder bytes
    with real PDF generation for portfolio and transcript downloads
    (`internal/comply/jobs.go:217`, `internal/comply/jobs.go:333`, §2 14-comply).
16. **Wire COPPA credit card micro-charge** — implement the Stripe micro-charge call in
    `SubmitCoppaConsent` so credit card verification is genuine and not bypassed by any
    non-empty string (`internal/iam/service.go:299-301`, §2 01-iam).
17. **Harden production config defaults** — require `OBJECT_STORAGE_PUBLIC_URL` to be set
    explicitly; externalize kratos.yml webhook secrets via env substitution; set session
    `same_site: Strict` in kratos.yml (§3.6).
18. **07-mkt: Add test files** — only domain with zero tests.
19. **Add handler tests** for 10-billing, 13-recs, 14-comply, 16-admin, 17-plan, 05-social
    (all missing `handler_test.go`).
20. **Implement missing Phase 2 endpoints** per domain (01-iam co-parent flow is highest
    value).
21. **Implement all 4 missing event handlers** in 08-notify (SubscriptionCreatedHandler,
    SubscriptionChangedHandler, SubscriptionCancelledHandler, PayoutCompletedHandler).
22. **Define missing event types** (iam.CoParentRemoved, iam.PrimaryParentTransferred,
    safety.ContentFlagged) to unblock 3 deferred subscriptions.

### P3 — Polish / Phase 2+

23. Fix spec-vs-code inconsistencies (§3.4 above).
24. Implement 09-media missing endpoints (delete, list, reprocess).
25. Implement 12-search Typesense adapter and suggestions endpoint.
26. Implement 06-learn progress snapshot background job.
27. Begin frontend feature development (`features/` directory).
28. Implement 17-plan calendar PDF export.
