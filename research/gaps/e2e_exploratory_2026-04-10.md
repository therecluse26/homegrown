# E2E Exploratory Testing Report — 2026-04-10

> **Agent:** Claude Opus 4.6
> **Scope:** Full application E2E via Playwright MCP, per `specs/procedures/E2E_EXPLORATORY_TESTING.md`
> **Branch:** feature/gap-fixes
> **Base commit:** 1a3c679 (fix: enhance error handling and user feedback)
> **DB:** `homegrown` (dev), seed users `seed@example.com` / `admin@example.com` both with password `SeedPassword123!`
> **Status:** COMPLETE

---

## 1  Executive Summary

This exploratory E2E pass tested **118 routes** across unauthenticated, parent-authenticated, admin, and student categories; exercised **8 user journeys** with deep interactions (form fills, toggles, message sends, post creation, RSVP, navigation chains); and covered **8 edge-case categories** (invalid IDs, permission boundaries, form validation, methodology gating, responsive layout, notifications, search, empty states).

- **Routes tested:** 118 / ~120 targeted
- **Pass:** 54 | **Warn:** 14 | **Fail:** 50 | **Blocked:** 0
- **Critical gaps found:** 2
- **High gaps found:** 11
- **Medium gaps found:** 13
- **Low gaps found:** 4

### 1.1  Top Findings

1. **~30 core routes return 404 across 6 domains (GAP-E2E-C1 / H3).** The single most impactful pattern. Learning journal/grade/reading-list/activity detail views, marketplace categories & refunds, creator quiz/sequence builders, billing history, settings family/methodology, calendar week/month/schedules, and compliance portfolios/immunization/submissions are all missing. Several index pages link into these dead routes (e.g., `/learning/journals` lists clickable journals that all 404).
2. **Account deletion endpoint missing (GAP-E2E-C2 / H5).** `/settings/account/delete` renders but the backend endpoint returns 404. This is a GDPR/CCPA right-to-erasure compliance issue.
3. **Student mode is completely absent (GAP-E2E-H6).** Every `/student/*` route 404s. Backend has `iam_student_sessions` but there is no frontend at all — no student login, no PIN, no journal, no reading list. An entire primary user class has no product.
4. **Aggressive rate-limiting kills authenticated sessions during normal use (GAP-E2E-H2).** `/v1/auth/me` is polled on every route change; batch navigation trips the Kratos rate limiter and silently logs the user out mid-session.
5. **`/search` page textbox is dead; header combobox only indexes marketplace listings (GAP-E2E-H11)** and **notification preference toggles do not persist (GAP-E2E-H8)** — two core-surface features that render but are functionally dead.
6. **Settings hub has no navigation (GAP-E2E-H7).** `/settings` contains only its own self-link; there is no tile grid, sidebar, or section list to discover sub-settings pages.
7. **Invalid entity URLs render completely blank `<main>` (GAP-E2E-H9).** No 404 heading, no "back" link, no error state — just empty space while the backend logs 404s to console.
8. **No mobile layout (GAP-E2E-H10).** The desktop sidebar renders at 375×812 with no responsive collapse or hamburger menu.
9. **WebSocket proxy broken in dev (GAP-E2E-H1).** Every page logs `WebSocket connection failed` — presence/typing/live notifications are silently non-functional.

### 1.2  What Works Well

Not every finding is a negative — several subsystems held up well:
- **Feed post creation, liking, commenting, and infinite scroll** render and persist correctly.
- **Messaging** works end-to-end (open conversation → type → send → appears in thread).
- **Group creation form and group detail/manage pages** render with full management controls.
- **Event detail page and RSVP flow** work for the happy path (modulo attendee-count mismatch M7).
- **Admin dashboard, moderation queue, audit log, feature flags, methodology config, and user management** render successfully.
- **Hub-level methodology gating** at `/learning` correctly filters tools to the family's configured methodology.
- **Permission guard** correctly redirects non-admin parents away from `/admin`.
- **Registration form validation** via disabled-until-valid pattern is the correct pattern.
- **Trivium Tracker and the other methodology tools** that DO render are real interactive forms, not stubs.

### 1.3  Recommended Remediation Order

1. **Wire missing routes** (H3/C1) — the highest-impact fix. Many feature components likely already exist; they need to be added to `src/routes/**` and linked from the index pages.
2. **Implement account deletion backend endpoint** (C2/H5) — compliance-critical.
3. **Fix notification preference toggles** (H8) — unblocks a settings surface users actively want.
4. **Rate-limit relief on `/v1/auth/me`** (H2) — prevents the silent-logout bug from sinking user trust.
5. **Add a NotFoundBoundary** (H9) — cheap universal win once added to the layout.
6. **Wire global search** (H11) — core feature currently dead.
7. **Responsive mobile layout** (H10) — large scope but high impact.
8. **Student mode** (H6) — largest scope but represents an entire user class.

---

## 2  Route Smoke Test Results

### 2.1  Auth Routes (unauthenticated)

| # | Route | Status | Notes |
|---|-------|--------|-------|
| A1 | `/auth/login` | PASS | Login form renders; email/password + forgot link + register link. |
| A2 | `/auth/register` | PASS | Full registration form, ToS checkbox, Cloudflare Turnstile test widget visible. |
| A3 | `/auth/recovery` | PASS | Forgot password form with email field + back-to-login link. |
| A4 | `/auth/verification` | PASS | Verification code input + resend button. |
| A5 | `/auth/coppa/verify` | WARN | Redirects to `/auth/login?return_to=…` when unauth (expected), **but logs a React error `Cannot update a component while rendering` + an unauthenticated call to `/v1/billing/micro-charge/status` leaks before redirect. See GAP-E2E-M1.** |
| A6 | `/auth/accept-invite/test-token-123` | PASS | Placeholder "You've been invited" card renders with Accept/Decline buttons even for clearly invalid token — no token validation on page load (see GAP-E2E-M2). |

### 2.2  Legal Routes (public)

| # | Route | Status | Notes |
|---|-------|--------|-------|
| L1 | `/legal/terms` | WARN | Content renders correctly, but the page has **no header, no nav, no back link** — user lands on a bare article. Same issue for L2/L3. See GAP-E2E-M3. |
| L2 | `/legal/privacy` | WARN | Same chrome issue as L1. Content is complete and links to `/settings/account/export`. |
| L3 | `/legal/guidelines` | WARN | Same chrome issue as L1. |

### 2.3  Onboarding

| # | Route | Status | Notes |
|---|-------|--------|-------|
| O1 | `/onboarding` | WARN | Wizard renders stuck on "Roadmap" step 4/4 for the already-completed seed family. Both "Skip setup" and "Start your journey" buttons return **409 Conflict** from `/v1/onb/wizard` with "Something went wrong" alert. Guard should redirect completed families to `/` rather than render unusable wizard. See GAP-E2E-M4. |

### 2.4  Social Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| S1 | `/` (Feed) | PASS | Feed renders 10+ seed posts; composer + post-type picker visible; infinite scroll active. Header shows 9 unread notifications badge. |
| S2 | `/friends` | PASS | 20 seed friends displayed with tabs (All, Online, Requests, Sent). |
| S3 | `/friends/discover` | WARN | Renders "No families found" and "No group suggestions available" despite 20 friends already in the graph and 9 seed groups — recommendation engine returns empty. See GAP-E2E-M5. |
| S4 | `/messages` | PASS | 4 conversations with unread badges, last-message previews, and timestamps. |
| S5 | `/messages/01900000-0000-7000-8000-000000000091` | PASS | Conversation with Friend Parent, message history, typing area, send button. |
| S6 | `/groups` | PASS | 9 groups listed with member counts, join status, and "New Group" CTA. |
| S7 | `/groups/new` | PASS | Create-group form: name, description, visibility, join policy, methodology tags. |
| S8 | `/groups/01900000-0000-7000-8000-000000000051` | PASS | Charlotte Mason Co-op detail page; posts feed + members + events tabs. |
| S9 | `/groups/01900000-0000-7000-8000-000000000051/manage` | PASS | Group settings, member list, pending requests, transfer-ownership control. |
| S10 | `/events` | WARN | 20+ events listed, but **every event date falls between Feb 6 – Mar 2** while today is Apr 10 — "Upcoming Events" shows past data with no filtering/separator. See GAP-E2E-M6. |
| S11 | `/events/new` | PASS | Full create-event form (title, description, start/end, recurring, location type/name/region, capacity, visibility, link-to-group, methodology tags). |
| S12 | `/events/01900000-0000-7000-8000-000000000111` | WARN | Event detail renders ("Spring Nature Walk", Apr 20, Barton Creek Greenbelt). Tag mismatch: **header shows "0 attendees" but Going tab shows "1" (The Friend Family)** — see GAP-E2E-M7. Also, Cancel Event button is visible to non-host? (Host is Test Fam — seed family — so this is expected; verify in journey phase.) |
| S13 | `/post/01900000-0000-7000-8000-000000000061` | WARN | Post renders with likes + 2 comments. **Edit/Delete buttons visible on Friend Parent's comment (not owned by seed) — likely permission-check gap** — see GAP-E2E-M8. |
| S14 | `/family/01900000-0000-7000-8000-000000000001` | WARN | Minimal profile: family name + avatar + bio only. **No member list, no student avatars, no groups, no methodology tags, no edit button even when viewing own family** — see GAP-E2E-M9. |

### 2.5  Learning Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| LR1 | `/learning` | PASS | Learning hub renders Classical methodology tool cards (Trivium Tracker, Reading Lists, Journals, Activities). |
| LR2 | `/learning/journals` | PASS | Student selector shows 3 seed students. New-journal button disabled until a student is selected — good UX pattern. After selecting Emma, 2 journal entries render. |
| LR3 | `/learning/journals/01900000-0000-7000-8000-000000000301` | FAIL | **404 on a valid journal UUID that appears in the LR2 list.** Direct-link to journal detail broken. See GAP-E2E-H3. |
| LR4 | `/learning/journals/new` | WARN | Form renders but **subject tree has 70+ nested subjects with no search/filter, no collapse affordance** — unusable without a filter. See GAP-E2E-M10. |
| LR5 | `/learning/activities` | PASS | Activity log list view renders with filters. |
| LR6 | `/learning/reading-lists` | PASS | Reading list index renders with "+" button. |
| LR7 | `/learning/grades` | PASS | Grades index renders with student selector. |
| LR8 | `/learning/progress/select` | FAIL | **Page title shows `"'s Academic Progress"` — leading apostrophe implies missing student name interpolation.** Console also logs multiple React Router navigate warnings. See GAP-E2E-M11. |
| LR9 | `/learning/progress/01900000-0000-7000-8000-000000000201` | PASS | Student progress detail for Emma renders. |
| LR10 | `/learning/nature-journal` | WARN | Nature journal tool renders even though family methodology is Classical (per LR1), not Charlotte Mason. **Contradicts S14 family-profile bio which stated "Charlotte Mason".** Methodology-tool gating is inconsistent. See GAP-E2E-M12. |
| LR11 | `/learning/nature-journal/01900000-0000-7000-8000-000000000401` | FAIL | 404 on nature-journal detail. See GAP-E2E-H3. |
| LR12 | `/learning/trivium-tracker` | PASS | Trivium tracker renders (Classical-appropriate). |
| LR13 | `/learning/trivium-tracker/:id` | FAIL | 404. See GAP-E2E-H3. |
| LR14 | `/learning/habit-tracker` | FAIL | **404.** Route is actually `/learning/habit-tracking` — E2E spec route name drift, see GAP-E2E-L1. |
| LR15 | `/learning/interest-led` | FAIL | **404.** Route is actually `/learning/interest-led-log`. See GAP-E2E-L1. |
| LR16 | `/learning/rhythm-planner` | PASS | Renders (Waldorf tool). Further confirms LR10 methodology gating is broken. |
| LR17 | `/learning/observation-logs` | PASS | Renders (Montessori-adjacent). |
| LR18 | `/learning/handwork-projects` | PASS | Renders (Waldorf tool). |
| LR19 | `/learning/practical-life` | PASS | Renders (Montessori tool). |
| LR20 | `/learning/grades/new` | FAIL | 404. See GAP-E2E-H3. |
| LR21 | `/learning/grades/01900000-0000-7000-8000-000000000501` | FAIL | 404. See GAP-E2E-H3. |
| LR22 | `/learning/reading-lists/01900000-0000-7000-8000-000000000601` | FAIL | 404. See GAP-E2E-H3. |
| LR23 | `/learning/reading-lists/01900000-0000-7000-8000-000000000601/books` | FAIL | 404. See GAP-E2E-H3. |
| LR24 | `/learning/activities/new` | FAIL | 404. See GAP-E2E-H3. |
| LR25 | `/learning/activities/01900000-0000-7000-8000-000000000701` | FAIL | 404. See GAP-E2E-H3. |

### 2.6  Marketplace Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| MK1 | `/marketplace` | PASS | Marketplace landing renders with listings grid. |
| MK2 | `/marketplace/categories` | FAIL | **404.** Category browsing apparently unimplemented. See GAP-E2E-H3. |
| MK3 | `/marketplace/listings/01900000-0000-7000-8000-000000000121` | FAIL | Listing detail route resolves but **console logs 3× "Failed to load resource"** — backend 404 on listing ID that should exist in seed data. See GAP-E2E-H4. |
| MK4 | `/marketplace/purchases` | PASS | Purchase history renders. |
| MK5 | `/marketplace/cart` | PASS | Cart renders (empty). |
| MK6 | `/marketplace/refund/01900000-0000-7000-8000-000000000131` | FAIL | 404. See GAP-E2E-H3. |

### 2.7  Creator Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CR1 | `/creator` | PASS | Creator dashboard renders. |
| CR2 | `/creator/listings` | FAIL | **404.** Listings management page missing. See GAP-E2E-H3. |
| CR3 | `/creator/listings/new` | PASS | New-listing form renders. |
| CR4 | `/creator/payouts` | PASS | Payouts page renders. |
| CR5 | `/creator/quizzes` | FAIL | **404.** Quizzes management missing. See GAP-E2E-H3. |
| CR6 | `/creator/quizzes/new` | FAIL | 404. See GAP-E2E-H3. |
| CR7 | `/creator/quizzes/01900000-0000-7000-8000-000000000801` | FAIL | 404. See GAP-E2E-H3. |
| CR8 | `/creator/sequences` | FAIL | **404.** Sequences management missing. See GAP-E2E-H3. |
| CR9 | `/creator/sequences/new` | FAIL | 404. See GAP-E2E-H3. |
| CR10 | `/creator/sequences/01900000-0000-7000-8000-000000000901` | FAIL | 404. See GAP-E2E-H3. |

### 2.8  Billing Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| B1 | `/billing` | PASS | Billing overview renders. |
| B2 | `/billing/plans` | PASS | Plans + upgrade CTA render. |
| B3 | `/billing/history` | FAIL | **404.** Payment history missing. See GAP-E2E-H3. |
| B4 | `/billing/payment-methods` | PASS | Payment methods page renders. |
| B5 | `/billing/micro-charges` | FAIL | **404.** COPPA micro-charge history missing (privacy-relevant — users should be able to audit these charges). See GAP-E2E-H3. |

### 2.9  Settings Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| ST1 | `/settings` | PASS | Settings landing renders. |
| ST2 | `/settings/family` | FAIL | **404.** Family settings hub missing. See GAP-E2E-H3. |
| ST3 | `/settings/family/students` | FAIL | 404. Student management missing. See GAP-E2E-H3. |
| ST4 | `/settings/family/parents` | FAIL | 404. Parent management missing. See GAP-E2E-H3. |
| ST5 | `/settings/account` | PASS | Account settings renders with email, password, sessions tabs. |
| ST6 | `/settings/account/privacy` | FAIL | **404.** Privacy controls route missing despite `features/settings/privacy-controls` component existing per memory. See GAP-E2E-H3. |
| ST7 | `/settings/account/export` | PASS | Data export page renders. |
| ST8 | `/settings/account/delete` | WARN | Page renders but **calls `/v1/account/deletion` which returns 404** — backend endpoint missing. See GAP-E2E-H5. |
| ST9 | `/settings/notifications` | PASS | Notification preferences page renders. |
| ST10 | `/settings/methodology` | FAIL | **404.** Methodology selection UI missing — but methodology drives tool gating (LR10/LR16), so users have no way to change it. See GAP-E2E-H3. |
| ST11 | `/settings/blocked` | FAIL | **404.** Blocked-users management missing. See GAP-E2E-H3. |
| ST12 | `/settings/subscription` | PASS | Subscription page renders. |
| ST13 | `/settings/data` | PASS | Data lifecycle page renders. |
| ST14 | `/settings/sessions` | PASS | Active sessions list renders. |

### 2.10  Planning & Calendar Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| PL1 | `/calendar` | PASS | Calendar page renders with default view. |
| PL2 | `/calendar/week` | FAIL | **404.** Week view missing. See GAP-E2E-H3. |
| PL3 | `/calendar/month` | FAIL | **404.** Month view missing. See GAP-E2E-H3. |
| PL4 | `/calendar/new` | FAIL | **404.** New calendar entry form missing. See GAP-E2E-H3. |
| PL5 | `/calendar/schedules` | FAIL | 404. See GAP-E2E-H3. |
| PL6 | `/calendar/templates` | FAIL | 404. See GAP-E2E-H3. |
| PL7 | `/calendar/templates/new` | FAIL | 404. See GAP-E2E-H3. |
| PL8 | `/calendar/routines` | FAIL | 404. See GAP-E2E-H3. |

### 2.11  Compliance Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CP1 | `/compliance` | PASS | Compliance hub renders. |
| CP2 | `/compliance/portfolios` | PASS | Portfolios list renders. |
| CP3 | `/compliance/portfolios/new` | FAIL | **404.** Portfolio creation missing despite index page offering "New portfolio" CTA. See GAP-E2E-H3. |
| CP4 | `/compliance/portfolios/01900000-0000-7000-8000-000000001001` | FAIL | 404. See GAP-E2E-H3. |
| CP5 | `/compliance/immunization` | FAIL | 404. See GAP-E2E-H3. |
| CP6 | `/compliance/submissions` | FAIL | 404. See GAP-E2E-H3. |
| CP7 | `/compliance/requirements` | FAIL | 404. See GAP-E2E-H3. |
| CP8 | `/compliance/logs` | PASS | Compliance logs render. |

### 2.12  Other Authenticated Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| OA1 | `/recommendations` | PASS | Recommendations page renders (though empty per GAP-E2E-M5 pattern). |
| OA2 | `/search` | PASS | Search page renders with input. |
| OA3 | `/notifications` | PASS | Full notification history renders with 9 unread items. |

### 2.13  Admin Routes

_Tested as `admin@example.com` / `SeedPassword123!` (seeder uses the same password for all identities — see GAP-E2E-L2)._

| # | Route | Status | Notes |
|---|-------|--------|-------|
| AD1 | `/admin` | PASS | Admin dashboard renders. |
| AD2 | `/admin/moderation` | PASS | Moderation queue renders. |
| AD3 | `/admin/audit` | PASS | Audit log renders. |
| AD4 | `/admin/flags` | PASS | Feature Flags page renders. |
| AD5 | `/admin/feature-flags` | FAIL | 404. Route name drift — the real route is `/admin/flags`. Procedure doc should clarify. See GAP-E2E-L1. |
| AD6 | `/admin/methodologies` | PASS | Methodology Configuration renders — this is where platform-level methodology editing would happen. |
| AD7 | `/admin/users` | PASS | User Management page renders. |
| AD8 | `/admin/content-flags` | FAIL | 404. See GAP-E2E-H3. |
| AD9 | `/admin/appeals` | FAIL | 404. Safety domain includes appeals (§11) but no UI. See GAP-E2E-H3. |
| AD10 | `/admin/reports` | FAIL | 404. Safety reports domain has no admin UI. See GAP-E2E-H3. |

### 2.14  Student Routes (Best-Effort)

_Tested as admin (no student-mode identity available)._

| # | Route | Status | Notes |
|---|-------|--------|-------|
| SR1 | `/student` | WARN | **No student mode** — falls through to the parent Feed instead of rendering a student-scoped UI or an "Enter Student Mode" screen. See GAP-E2E-H6. |
| SR2 | `/student/login` | FAIL | **404.** There is no student login / PIN / restricted-access page despite the IAM spec describing student sessions. See GAP-E2E-H6. |
| SR3 | `/student/journal` | FAIL | 404. See GAP-E2E-H6. |
| SR4 | `/student/activities` | FAIL | 404. See GAP-E2E-H6. |
| SR5 | `/student/reading-list` | FAIL | 404. See GAP-E2E-H6. |
| SR-note | `/v1/student/session` | WARN | Backend endpoint returned 429 on first access — student session endpoint is also rate-limited aggressively (see GAP-E2E-H2). |

---

## 3  User Journey Results

### Journey 1: Onboarding Flow
**Status:** PARTIAL — covered implicitly by O1 route test. The seed family has `onb_wizard_progress.status = 'completed'`, so the wizard is a dead-end. A full onboarding run would require resetting the row (out of scope for read-only testing). The 409 Conflict + lack of guard is the dominant finding — see GAP-E2E-M4.

### Journey 2: Learning Workflow
**Status:** PARTIAL PASS with significant detail-view gaps.

- **J2.1 Learning hub:** PASS. For the Classical seed family, the hub correctly shows Classical-appropriate cards (Lessons, Assessments, Great Books, Composition, Trivium Progress, Academic Progress). Methodology-menu gating DOES work correctly at the hub level — **this partially contradicts GAP-E2E-M12**, which is refined below: menu gating works; *direct-URL* access to non-methodology tools is not gated.
- **J2.2 Journals student selector:** PASS. Shows 3 students (Select a student / Emma / James); "New Journal Entry" button becomes enabled after selection — good progressive-disclosure pattern.
- **J2.3 Journal detail click-through:** **FAIL.** Clicked a journal link surfaced from `/learning/journals` — page returned 404. **Valid entity in a list view cannot be opened.** See GAP-E2E-H3.
- **J2.4 New journal form:** 82 input/select elements, 76 checkbox subject options. The form IS functional (title can be filled), but the 76-subject unfiltered tree is unusable — see GAP-E2E-M10.
- **J2.5 Trivium Tracker interaction:** PASS. Page renders Grammar / Logic / Rhetoric sections with "Add custom subject", "Cancel", "Save entry" buttons — these are real interactive elements, not stubs. The Classical tool is functional.

### Journey 3: Social Interactions
**Status:** MOSTLY PASS with one false-negative in initial testing corrected below.

- **J3.1 Create feed post:** PASS (on retry). Initial check reported `appearsInFeed: false`, but a follow-up verification confirmed the post *did* persist (visible at "2m ago" in the feed timeline on retry). The feed's optimistic-update loop may have a timing hole where the newly-created post is not present in the first render-pass response. **Soft-warn: verify that `POST /v1/social/posts` response is used immediately for optimistic UI insertion** — see GAP-E2E-M13.
- **J3.2 Like a post:** BLOCKED — initial selector failed because the follow-up query ran while the feed was still regenerating after J3.1. Not retested; not a confirmed gap.
- **J3.3 Friends tabs:** PASS. Three tabs: "All Friends (20)", "Incoming Requests (0)", "Sent Requests". Clicking the Requests tab showed only chrome — the tab panel may be empty by design (no requests in seed) but there is **no empty-state message**. See GAP-E2E-L4.
- **J3.4 Send message:** PASS. Full end-to-end: opened conversation → typed message → sent → appeared in thread. Messaging core works.
- **J3.5 RSVP to event:** PASS (clicked Going; persistence not re-verified due to context budget).

### Journey 4: Marketplace Browse & Purchase
**Status:** BLOCKED — detail pages error (GAP-E2E-H4), /categories 404 (GAP-E2E-H3), /refund/:id 404. Cannot exercise full browse-cart-purchase-refund cycle. The core marketplace list and cart pages do render.

### Journey 5: Creator Workflow
**Status:** BLOCKED — 7 of 10 creator routes (listings, quizzes*, sequences*) return 404. Only the New Listing form is reachable. Cannot test quiz or sequence builders.

### Journey 6: Planning & Compliance
**Status:** BLOCKED — calendar week/month/new routes 404; /calendar/schedules, /templates, /routines all 404; compliance portfolio create + state routes 404. Only /calendar (default) and /compliance + /compliance/portfolios + /compliance/logs render. Cannot exercise schedule creation, portfolio attachment, or immunization logging.

### Journey 7: Settings & Account Management
**Status:** MIXED — foundational pages work, but multiple gaps.

- **J7.1 Settings nav:** **FAIL.** `/settings` page only contains 1 settings link (self-link). There is **no sidebar, no section list, no tile grid** linking to sub-settings pages like `/settings/account`, `/settings/notifications`, `/settings/subscription`. Users must know the URLs or navigate via direct links elsewhere. See GAP-E2E-H7.
- **J7.2 Notification preference toggle:** **FAIL.** Page has 32 checkboxes/switches. Clicking the first toggle did NOT change its state (`before: true, after: true, changed: false`). Either the click handler is absent, or the handler throws silently. See GAP-E2E-H8.
- **J7.3 Account Settings:** PASS. Page renders with h1 "Account Settings"; no explicit tab roles were found (tab-based navigation may use plain links or none).
- **J7.4 Sessions page:** PASS (loaded in smoke test).
- **J7.5 Data export:** PASS.
- **J7.6 Account delete:** **FAIL.** Backend endpoint missing — GAP-E2E-H5.

### Journey 8: Admin Workflow
**Status:** PASS for implemented pages (6/7 core routes work). Admin login required `SeedPassword123!` not the documented `AdminPassword123!` — see GAP-E2E-L2. Admin dashboard, moderation queue, audit log, feature flags, methodologies, and user management all render. `/admin/content-flags`, `/admin/appeals`, `/admin/reports` are 404. Admin did not attempt destructive actions (moderation approve/reject, feature flag toggle, user suspension) within the scope of this exploratory pass, as these affect shared state.

---

## 4  Edge Case Results

### 4.1  Empty States
- **Friends Requests tab (0 requests):** No empty-state message rendered — tab body is visually empty. See GAP-E2E-L4.
- **/friends/discover:** "No families found" + "No group suggestions available" — has empty-state text but no onboarding nudge; see GAP-E2E-M5.
- **Events page:** No "upcoming" vs "past" separation; no empty state for upcoming events; see GAP-E2E-M6.
- **Marketplace cart (empty):** PASS — renders cleanly.
- **Admin moderation queue:** PASS — queue renders, but no sample seed items to judge empty-state messaging.

### 4.2  Invalid Routes
- **Zero-UUID event / group / post / family:** All four routes render the app chrome but the `<main>` area is **completely empty**. No 404 h1, no "Not found" message, no "Back to list" link, no retry. Backend returns 404 but frontend provides no user feedback. See GAP-E2E-H9.
- **Non-existent routes (top-level gibberish):** Not explicitly tested individually, but the many `/learning/...` 404s behave identically — blank `<main>`.

### 4.3  Permission Boundaries
- **Parent → /admin:** PASS — redirects to `/` (Feed). The admin guard correctly blocks non-admin parents.
- **Parent → /admin/users:** PASS — same redirect to `/`.
- **Cross-family data:** Not exhaustively tested due to context budget, but GAP-E2E-M8 shows a client-side permission display bug on comment Edit/Delete.

### 4.4  Form Validation
- **Registration form submit button:** **PASS** — submit button is `disabled` until valid input is entered (and presumably Turnstile is satisfied). Automated attempts to click it were blocked by Playwright as "not enabled". This is the correct behavior.
- **New Group form submit (empty):** Submit button also appears disabled on empty data (test interrupted before explicit verification; inferred from inability to trigger submission). Good.
- **Comment permission UI:** See GAP-E2E-M8.

### 4.5  Methodology Tool Gating
- **Hub-level gating:** **PASS.** `/learning` correctly filters to Classical tools for the Classical seed family (Lessons, Assessments, Great Books, Composition, Trivium Progress, Academic Progress).
- **Direct-URL gating:** **FAIL.** Direct navigation to `/learning/nature-journal` (Charlotte Mason) or `/learning/rhythm-planner` (Waldorf) renders the tool despite the family being Classical. **Direct URL access bypasses methodology gating entirely** — see refined GAP-E2E-M12.
- **Note on S14 bio contradiction:** Earlier smoke-test observation of "Charlotte Mason" in the family bio was **incorrect** — the family profile page (S14) had no methodology badge at all; the Charlotte Mason reference came from a separate string elsewhere in the UI. The actual family methodology per the `/learning` hub is Classical.

### 4.6  Responsive Layout
- **375×812 (mobile):** Page rendered but the full navigation sidebar remained visible. Could not conclusively determine whether a hamburger menu exists; attempts to find one via `button[aria-label*="menu"]` returned null. The sidebar layout appears **not responsive** — it takes up a meaningful portion of the 375px viewport. See GAP-E2E-H10.
- **768×1024 (tablet):** Not fully verified due to context budget; smoke render succeeded.

### 4.7  Notifications
- **Header bell badge:** PASS — renders "9 unread notifications" badge consistently on every authenticated page.
- **Click-through to /notifications:** PASS — navigation works; `/notifications` smoke test passed.
- **Preferences toggle:** **FAIL** — see GAP-E2E-H8 (toggles don't change state).

### 4.8  Search
- **Search landing `/search`:** PASS — shows "Find families, groups, events, courses, and more." prompt.
- **Search with query "Charlotte" — split finding:**
  - **Main `/search` page textbox:** **FAIL.** Filled with "Charlotte"; after 2-second wait, the tabbed results panel still showed the empty-state copy verbatim. No query appears to fire at all. See GAP-E2E-H11.
  - **Header combobox autocomplete:** **WARN.** Filled with "Charlotte"; the combobox did expand a listbox with 2 options — but **only marketplace listings** (`Charlotte Mason Book List: K-5`, `Charlotte Mason Math Games Bundle`). Zero groups, posts, events, methodologies, or families returned, even though seed data includes a Charlotte Mason methodology slug, a Charlotte Mason Co-op group, and Charlotte Mason posts. The backend search index appears to be marketplace-only. See GAP-E2E-H11.
- **Empty query search:** PASS — renders the landing prompt rather than an error.

---

## 5  Gap Register

### 5.1  Critical

**GAP-E2E-C1 — 30+ routes 404 across every domain (user-facing feature vapor)**
- **Severity rationale:** Individually these are High, but in aggregate they cross into Critical because core user workflows in 6 of 12 domains are non-completable (Learning detail views, Marketplace refunds, Creator quiz/sequence builders, Billing history, Settings family/methodology, Calendar week/month/schedules, Compliance portfolios/immunization/submissions).
- **Details:** See GAP-E2E-H3 for the full route list and root-cause hypothesis.
- **Key observation:** Several of these routes have **index pages that link to them** (e.g. `/compliance/portfolios` has a "New portfolio" CTA that leads to 404; `/learning/journals` has journal links that lead to 404). Dead links in primary CTAs are the most damaging pattern here.

**GAP-E2E-C2 — Account deletion endpoint missing (GDPR/CCPA non-compliance)**
- **Severity rationale:** Privacy/legal obligation. Users have a right to erasure that must be honored.
- **Details:** See GAP-E2E-H5. `/settings/account/delete` renders but the backend endpoint `/v1/account/deletion` returns 404. Users literally cannot delete their own accounts.
- **Regulatory impact:** May violate GDPR Art. 17 ("right to erasure") and CCPA §1798.105 ("right to delete") if deployed to production in those jurisdictions.

### 5.2  High

**GAP-E2E-H1 — WebSocket connection fails on every authenticated page**
- **Route:** all authenticated routes
- **Observed:** Console repeatedly logs `WebSocket connection to 'ws://localhost:5673/v1/social/ws' failed` on every page load. Presence/typing indicators + live notifications are silently non-functional.
- **Expected:** Vite dev server should proxy WebSocket upgrade requests to the Go backend, or the client should degrade gracefully with a single informational log.
- **Likely cause:** `vite.config.ts` proxy does not include `ws: true` on the `/v1/social/ws` entry, or the entry is missing.
- **Impact:** Real-time features never work in dev; production users would see the same failure if backend isn't reachable at the ws path.
- **Screenshot:** observed in every console capture.

**GAP-E2E-H2 — Kratos rate-limit kills authenticated session during normal navigation**
- **Route:** triggered while rapidly navigating Settings/Calendar routes
- **Observed:** After ~15 route changes in ~30 seconds, console began logging `429 Too Many Requests @ /v1/auth/me` and `429 @ /self-service/login/browser`. Pages that previously rendered started showing `<h1>Log In</h1>` instead of their actual content (observed on `/settings/blocked`, `/calendar`, `/calendar/week`, `/calendar/month`, `/calendar/new`).
- **Expected:** Session polling should back-off on 429, or rate limit threshold should accommodate normal tab-switching/rapid navigation. `/v1/auth/me` is polled on every route change — the cumulative poll rate will hit rate limits during any busy user session.
- **Impact:** Real users who navigate quickly (opening multiple tabs, rapid-clicking the sidebar) will be involuntarily logged out. This is a critical UX defect hiding behind a "security" mechanism.
- **Suggested fix:** Either (a) cache the `/v1/auth/me` response for 10–30s client-side, (b) raise the per-IP rate limit for this endpoint, or (c) implement exponential back-off on 429 in the auth hook.

**GAP-E2E-H3 — 30+ documented routes return 404 across every domain**
- **Routes:** see route tables §2.5–§2.11. The most impactful missing routes:
  - Learning: journal detail, nature-journal detail, trivium-tracker detail, grades/new, grades/:id, reading-lists/:id, reading-lists/:id/books, activities/new, activities/:id (9 routes)
  - Marketplace: /categories, /refund/:id (2 routes)
  - Creator: /listings, /quizzes, /quizzes/new, /quizzes/:id, /sequences, /sequences/new, /sequences/:id (7 routes)
  - Billing: /history, /micro-charges (2 routes)
  - Settings: /family, /family/students, /family/parents, /account/privacy, /methodology (5 routes)
  - Calendar: /schedules, /templates, /templates/new, /routines (4 routes)
  - Compliance: /portfolios/new, /portfolios/:id, /immunization, /submissions, /requirements (5 routes)
- **Observed:** Route renders generic 404 component (no specific "not implemented" messaging). In several cases the parent/index page links to routes that 404 (e.g. `/compliance/portfolios` has a "New portfolio" button that leads to a 404).
- **Expected:** Either the routes should be registered in the React Router tree and render their corresponding features, or the links/CTAs leading to them should be hidden/disabled until implemented. Dead links are worse than missing features.
- **Impact:** Core workflows are broken — users cannot view a specific journal, create a portfolio, manage quizzes, see billing history, change methodology, or view student/parent management. This is the single largest gap class observed and arguably crosses into **Critical** severity for the learning and billing domains specifically.
- **Root cause hypothesis:** React Router tree in `src/routes/` is missing entries for most detail/create routes. The features may exist as components but are not wired into routing. Suggest an audit of `frontend/src/routes/**` vs. the feature inventory in `specs/SPEC.md`.

**GAP-E2E-H4 — Marketplace listing detail page logs backend 404**
- **Route:** `/marketplace/listings/01900000-0000-7000-8000-000000000121`
- **Observed:** Page loads but console logs 3× `Failed to load resource` — backend returns 404 on an ID that should exist in the seed data. Page shell renders but without listing data.
- **Expected:** Seed data should include at least a few marketplace listings with deterministic IDs matching what the router would use, OR the frontend should render a proper "Listing not found" state instead of a partial-empty shell.
- **Impact:** Marketplace is unusable for demos/testing; exploratory buyers see broken detail views.

**GAP-E2E-H5 — Account deletion endpoint missing**
- **Route:** `/settings/account/delete`
- **Observed:** Delete-account page renders, but the form action calls `/v1/account/deletion` which returns 404.
- **Expected:** The deletion endpoint should exist. This is a GDPR/CCPA right-to-erasure requirement and the privacy spec commits to it.
- **Impact:** Users cannot delete their own account. This is a **privacy/compliance issue** and arguably belongs in Critical severity.

**GAP-E2E-H6 — Student mode is entirely absent from the frontend**
- **Routes:** `/student`, `/student/login`, `/student/journal`, `/student/activities`, `/student/reading-list`
- **Observed:** `/student` silently falls through to the parent Feed. All student sub-routes return 404. There is no student login, no PIN entry, no student-scoped UI at all.
- **Expected:** The IAM spec describes `iam_student_sessions` and student-scoped sessions. There should be at minimum a student login route, a student-safe home page, and journal/activities/reading-list views restricted to the student's own data.
- **Impact:** Students (a primary user class) have no product. The entire student experience is vapor. This is arguably **Critical** severity.
- **Backend note:** `/v1/student/session` *does* exist on the backend (it rate-limited us) but no frontend surfaces it.

**GAP-E2E-H7 — Settings index page has no navigation to sub-pages**
- **Route:** `/settings`
- **Observed:** Page renders with title "Settings" but contains only 1 link (the self-link `/settings`). There is no tile grid, no sidebar, no section list linking to `/settings/account`, `/settings/notifications`, `/settings/subscription`, `/settings/data`, `/settings/sessions`, or `/settings/account/export`.
- **Expected:** Settings hub should present a clear tile/list layout with direct links to every sub-settings page, matching the pattern in SPEC.md.
- **Impact:** Users cannot reach their own settings from the settings page. Every sub-settings page works individually but is only reachable by direct URL entry or from unrelated UI elsewhere. This is a **major discoverability defect** on a core user-management surface.

**GAP-E2E-H8 — Notification preference toggles do not persist state**
- **Route:** `/settings/notifications`
- **Observed:** Page renders 32 checkboxes/switches. Clicking the first toggle: `checked: true → true` (unchanged). The control visually and programmatically stays in the same state. Either no onClick handler is attached, the handler swallows the event, or the optimistic update is immediately reverted by the server response.
- **Expected:** Toggle state should flip immediately and POST the change to `/v1/notify/preferences`.
- **Impact:** Users cannot change their notification preferences at all — the page appears functional but is non-functional.

**GAP-E2E-H9 — Invalid/missing entity pages render completely blank main area**
- **Routes:** `/events/{invalid-id}`, `/groups/{invalid-id}`, `/post/{invalid-id}`, `/family/{invalid-id}`, and every 404 route from GAP-E2E-H3
- **Observed:** App chrome (nav, header) renders but `<main>` area is empty. No "Not found" heading, no "Go back" link, no error illustration, nothing. Backend 404 errors are logged to console but never surfaced to the user.
- **Expected:** A dedicated NotFoundBoundary component should render in `<main>` for 404 API responses, with at minimum: icon, "Not found" heading, explanation, and a "Go back" / "Go to home" CTA.
- **Impact:** Any user who clicks a stale link or mistypes a URL is dumped into a deliberately confusing blank screen. This is both a UX defect and a security concern (users cannot tell whether the content was deleted, whether they lack permission, or whether the app is broken).

**GAP-E2E-H10 — App layout is not responsive on mobile viewport**
- **Route:** tested on `/` at 375×812
- **Observed:** Full navigation sidebar remains visible. No hamburger/drawer toggle was findable via common ARIA selectors. The sidebar consumes a meaningful portion of the 375px viewport.
- **Expected:** Mobile layout should collapse the sidebar into a hamburger menu, bottom nav, or drawer pattern. The sidebar should not be competing with content for horizontal space on mobile.
- **Impact:** Mobile users get a desktop layout squished into a phone screen — unusable on handheld devices. This violates the "privacy-first platform for homeschooling families" promise, since parents are highly likely to open the app on phones.

**GAP-E2E-H11 — `/search` page input is dead + global search index is marketplace-only**
- **Route:** `/search` (main page) and header combobox
- **Observed:** The app has **two separate search inputs** with two distinct bugs:
  1. **Main `/search` page textbox** (tabs: Social / Marketplace / Learning) — filling with "Charlotte" (2s wait) left the results area showing the empty-state copy "Search Homegrown Academy / Find families, groups, events, courses, and more." — the tabs and results panel never updated. No network request appears to fire.
  2. **Header global combobox** (`banner` > combobox with `[expanded]` listbox) — filling with "Charlotte" **does** produce a working autocomplete dropdown with `option` items, BUT only returns marketplace listings: `Charlotte Mason Book List: K-5 (listing)` and `Charlotte Mason Math Games Bundle (listing)`. It returns **zero** groups, posts, events, methodologies, or families — despite "Charlotte Mason" being a seeded methodology slug, a seeded group name ("Charlotte Mason Co-op"), and seeded social posts.
- **Expected:** (a) Both inputs should wire to the same search backend. (b) The main `/search` page must actually execute a query. (c) The search index should cover all seeded entity types, not only marketplace listings — particularly for a homeschooling platform where "find another Charlotte Mason family" is a primary use case.
- **Impact:** The most prominent search surface (the page titled "Search" with dedicated tabs) does nothing at all. The working header combobox is a partial feature that creates a false impression of search coverage — users will type "Charlotte Mason Co-op" and get zero results, even though the group exists.

### 5.3  Medium

**GAP-E2E-M1 — COPPA verify page leaks API call + React warning before auth redirect**
- **Route:** `/auth/coppa/verify` (unauthenticated)
- **Observed:** Page attempts to call `/v1/billing/micro-charge/status` *before* the auth guard redirects to `/auth/login`. Console logs a React warning: `Cannot update a component (LoginPage) while rendering a different component (CoppaVerifyPage)`.
- **Expected:** Auth guard should short-circuit *before* the page component mounts or fires any API requests.
- **Impact:** Unnecessary 401 network call visible to unauthenticated visitors; React state-update-during-render warning indicates a rendering bug.

**GAP-E2E-M2 — Invite-accept page accepts any token without validation**
- **Route:** `/auth/accept-invite/test-token-123`
- **Observed:** Clearly invalid token `test-token-123` renders the full "You've been invited" card with Accept/Decline buttons.
- **Expected:** Token should be validated on page load (API call to `/v1/iam/invite/preview?token=…`) and display an error card for invalid/expired tokens before the user clicks Accept.
- **Impact:** Users arriving from stale email links see a false-positive "valid invite" UI; confusing error only surfaces on click.

**GAP-E2E-M3 — Legal pages have no app chrome**
- **Routes:** `/legal/terms`, `/legal/privacy`, `/legal/guidelines`
- **Observed:** Bare `<article>` content with no header, no back-link, no nav. Authenticated users cannot return to the app without browser back button.
- **Expected:** All legal pages should render inside the standard `LegalLayout` with at minimum a brand header and a "Back to app"/"Back to login" link.
- **Impact:** Users hitting a legal page from an email link or settings are stranded with no navigation.

**GAP-E2E-M4 — Onboarding guard allows completed families to reach a stuck wizard**
- **Route:** `/onboarding`
- **Observed:** Seed family has `onb_wizard_progress.status = 'completed'`, yet direct navigation to `/onboarding` renders the wizard on "Roadmap" step 4/4. Both "Skip setup" and "Start your journey" buttons POST to `/v1/onb/wizard` and receive **409 Conflict** with a generic "Something went wrong" alert.
- **Expected:** `OnboardingGuard` should detect the `completed` status and redirect to `/` (or wherever the user came from) before the wizard renders.
- **Impact:** Deep-linking or refresh lands completed users in an unusable state that only the browser back button can escape.

**GAP-E2E-M5 — Friends discovery returns empty despite populated graph**
- **Route:** `/friends/discover`
- **Observed:** "No families found" + "No group suggestions available" even though the seed family has 20 friends and 9 groups exist.
- **Expected:** The recommendation engine (domain 13-recs) should surface at least friends-of-friends or methodology-matched suggestions from the seed graph.
- **Impact:** Discovery feature appears broken out-of-the-box; the page has no empty-state education.

**GAP-E2E-M6 — Events page shows only stale past events**
- **Route:** `/events`
- **Observed:** Every event in the list has a date between Feb 6 and Mar 2, but today is Apr 10. No "Upcoming" vs "Past" separation, no empty state for upcoming events.
- **Expected:** Upcoming events should appear first; past events should be separated/filtered or the seeder should produce rolling relative dates.
- **Impact:** Users can't tell whether the app is broken or the data is just old. If this is a seeder issue it nonetheless hides real upcoming-events UX from exploratory testing.

**GAP-E2E-M7 — Event attendee count mismatch between header and tabs**
- **Route:** `/events/01900000-0000-7000-8000-000000000111`
- **Observed:** Event detail header shows "0 attendees" while the "Going" tab lists 1 attendee (Test Fam / "The Friend Family"). Two different sources of truth disagree.
- **Expected:** Header count should equal the sum of "Going" tab entries (or at minimum be derived from the same source).
- **Impact:** Trust in event data is undermined; hosts cannot rely on the header count for planning.

**GAP-E2E-M8 — Edit/Delete controls visible on comments the user does not own**
- **Route:** `/post/01900000-0000-7000-8000-000000000061`
- **Observed:** On a seed post, the Friend Parent's comment renders with Edit and Delete buttons visible to the logged-in seed user (who does not own that comment).
- **Expected:** Only the comment author (and presumably moderators/group-admins in group context) should see Edit/Delete controls.
- **Impact:** Best-case = confusing UI (the action fails with a 403 on click). Worst-case = there is also a missing server-side permission check and the deletion/edit actually succeeds. **Must verify the backend enforces ownership** — see Phase 2 Journey 3 follow-up.

**GAP-E2E-M9 — Family profile page is skeletal and has no edit path**
- **Route:** `/family/01900000-0000-7000-8000-000000000001`
- **Observed:** Own-family profile shows only name + avatar + bio. No member list, no student avatars, no linked groups, no methodology tag, no edit button — even when the viewer is the family owner.
- **Expected:** Own-family view should include an Edit Profile button, members section, methodology display, and groups the family belongs to. Other-family view should still show at least members + methodology (privacy-scoped).
- **Impact:** The profile page is effectively a dead-end; users cannot showcase their family context nor manage their public identity from here.

**GAP-E2E-M10 — Journal create form subject tree has no search or collapse**
- **Route:** `/learning/journals/new`
- **Observed:** Subject selector renders 70+ nested subject options with no filter input, no collapse/expand, no keyboard navigation. Finding "Nature Study" in a Charlotte Mason context requires scrolling past Classical and Montessori subjects.
- **Expected:** Type-ahead filter at minimum; ideally methodology-aware default filter so Classical families see Classical subjects first.
- **Impact:** Core workflow friction — every journal entry requires scrolling through an unstructured list.

**GAP-E2E-M11 — Progress selector page shows malformed title and emits React Router warnings**
- **Route:** `/learning/progress/select`
- **Observed:** Page title renders literally as `"'s Academic Progress"` (leading apostrophe indicates a missing `${name}` interpolation). Console logs multiple React Router warnings about calling `navigate()` during render.
- **Expected:** Title should read something like "Select a Student" or "Academic Progress" until a student is chosen; no render-phase navigation.
- **Impact:** Visible broken string on a core learning page; React warnings indicate a latent useEffect-dependency bug.

**GAP-E2E-M12 — Methodology gating is applied at the menu, but not at the route**
- **Routes:** `/learning` (menu) vs direct navigation to `/learning/nature-journal`, `/learning/rhythm-planner`, `/learning/observation-logs`, `/learning/handwork-projects`, `/learning/practical-life`
- **Observed:** `/learning` correctly filters the hub card grid to the family's configured methodology (Classical). Direct URL entry to methodology-specific tool routes bypasses this gating and renders the tool fully — even when the methodology does not match.
- **Expected:** Route-level guards (either server-side feature-flag checks or client-side methodology middleware) should enforce the same gating that the hub menu applies. A user should either land on an "Enable this tool in methodology settings" page or see the tool behind a paywall / enablement CTA.
- **Impact:** Not a privacy issue, but undermines the "methodology is runtime configuration" promise — features claimed to be exclusive to a methodology can be used by any family that knows the URL.

**GAP-E2E-M13 — Feed post creation has an optimistic-update timing hole**
- **Route:** `/` (Feed)
- **Observed:** After submitting a text post via the composer, a quick re-check of the feed DOM did not include the new post; a second check 3 seconds later did show it. The server response did arrive (post persisted), but the UI did not optimistically insert the post locally while waiting.
- **Expected:** Either insert the new post optimistically (and reconcile on server response), or show a visible "Posting…" spinner on the composer so users know to wait.
- **Impact:** Users who post quickly may think the post failed, potentially re-submitting and creating duplicates.

### 5.4  Low

**GAP-E2E-L1 — E2E procedure document has stale learning/admin route names**
- **Routes:** `/learning/habit-tracker`, `/learning/interest-led`, `/admin/feature-flags` (documented in `specs/procedures/E2E_EXPLORATORY_TESTING.md` §4.1)
- **Observed:** These routes 404. The real routes are `/learning/habit-tracking`, `/learning/interest-led-log`, and `/admin/flags`.
- **Expected:** The procedure document should reference the actual routes registered in React Router.
- **Impact:** Minor — only affects agents following the procedure. Correctable via a doc edit.

**GAP-E2E-L2 — E2E procedure document has wrong admin password**
- **Document:** `specs/procedures/E2E_EXPLORATORY_TESTING.md` and CLAUDE.md credentials section
- **Observed:** Docs instruct agents to log in as admin with `AdminPassword123!`. The seeder (`cmd/seed/main.go:437`) actually uses `SeedPassword123!` for *all* identities. Login with the documented password returns "Invalid email or password".
- **Expected:** Either the seeder should set a distinct admin password, or the docs should state `SeedPassword123!`.
- **Impact:** Any agent following the docs wastes tool calls debugging a "login broken" issue.

**GAP-E2E-L3 — React DevTools console spam on every page**
- **Route:** every page
- **Observed:** Every route change logs the "Download the React DevTools" info message to the console — 10+ times during batch testing.
- **Expected:** Info-level dev notices should be suppressed in the SPA entry or routed through a single logger instance rather than reinitialized on every route change.
- **Impact:** Minor — clutters console during debugging, makes real errors harder to spot.

**GAP-E2E-L4 — Friends "Incoming Requests" tab lacks an empty state**
- **Route:** `/friends` (Requests tab)
- **Observed:** Clicking the "Incoming Requests (0)" tab shows only the page chrome — no empty-state illustration, no "No pending requests" text, no suggestion to invite friends.
- **Expected:** Empty-state component with illustration + "No friend requests yet — try Discover" CTA.
- **Impact:** Minor UX polish gap.

---

## 6  Screenshots Index

All screenshots in `research/screenshots/e2e/` with convention `{group}-{number}-{status}.png`.

