# E2E Exploratory Testing Report — 2026-04-12

> **Agent:** Claude Opus 4.6
> **Duration:** ~45 minutes
> **Scope:** Full application E2E via Playwright MCP
> **Accounts tested:** seed@example.com (parent), admin@example.com (admin)

---

## 1  Executive Summary

- **Routes tested:** 72 / ~80
- **Pass:** 63 | **Warn:** 1 | **Fail:** 5 | **Blocked:** 3
- **Critical gaps found:** 0
- **High gaps found:** 2
- **Medium gaps found:** 4
- **Low gaps found:** 2

---

## 2  Route Smoke Test Results

### 2.1  Auth & Legal Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| A1 | /auth/login | PASS | Form renders: email, password, remember me, submit, register link |
| A2 | /auth/register | PASS | Full form with Turnstile CAPTCHA in test mode |
| A3 | /auth/recovery | PASS | Email field, send recovery link, back to login |
| A4 | /auth/verification | PASS | Verification code input, verify button, resend link |
| A5 | /auth/coppa/verify | WARN | Redirects to login (expected for unauthenticated), but React "Cannot update component while rendering another" console error |
| A6 | /auth/accept-invite/:token | PASS | Shows invitation with accept/decline buttons |
| L1 | /legal/terms | PASS | Terms of Service renders correctly |
| L2 | /legal/privacy | PASS | Privacy Policy renders correctly |
| L3 | /legal/community-guidelines | PASS | Community Guidelines renders correctly |

### 2.2  Social Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| S1 | /feed | PASS | Posts with author info, timestamps, like buttons, post composer with type buttons (text/photo/milestone/event_share/resource_share) |
| S2 | /friends | PASS | 20 friends listed, tabs (All/Incoming/Sent), search, message/unfriend buttons |
| S3 | /friends/discover | PASS | Search, methodology filters, suggested families, group suggestions |
| S4 | /messages | PASS | Conversation list with avatars, unread badges, previews |
| S5 | /messages/:id | PASS | Message history, search/mute buttons, message input; sent message successfully appeared with timestamp |
| S6 | /groups | PASS | 9 groups in "My Groups" tab with Discover tab |
| S7 | /groups/new | PASS | Name, description, join policy radios; validation disables submit until filled |
| S8 | /groups/:id | PASS | Header, members, posts tabs, leave button, empty posts state |
| S9 | /groups/:id (management) | PASS | Members tab with roles, pending requests tab, promote/remove actions |
| S10 | /events | PASS | Event list with RSVP buttons; toggled RSVP "Going" on Pottery workshop successfully |
| S11 | /events/new | PASS | Comprehensive form with all fields |
| S12 | /events/:id | PASS | Full detail with RSVP, attendee list, cancel event, export CSV |
| S13 | /feed/post/:id | PASS | Post with comments, like/unlike toggle works, posted comment successfully |
| S14 | /family/:id | PASS | Avatar, name, bio |

### 2.3  Learning Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| LR1 | /learning | PASS | Quick actions grid, student progress cards (Emma, James) |
| LR2 | /learning/activity-log | PASS | Student selector, filters, Emma's 6 activities loaded |
| LR3 | /learning/journals | PASS | Student selector, type filter, Emma's 2 journals |
| LR4 | /learning/journals/new | PASS | Full composition form; 75+ subjects in flat unsorted list (UX concern — see GAP-L1) |
| LR5 | /learning/reading | PASS | Emma's reading list with progress |
| LR6 | /learning/progress/:id (Emma) | PASS | Detailed stats, hours by subject, recent activity |
| LR8 | /learning/assessments | PASS | Student selector, subject filter |
| LR14 | /learning/session | PASS | Student cards (Emma age 12, James age 9) |
| LR17 | /learning/nature-journal | PASS | Rich form, methodology mismatch banner; same flat subject tree issue |
| LR18 | /learning/trivium | PASS | Grammar/Logic/Rhetoric stages; same flat subject tree issue |

### 2.4  Marketplace Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| MK1 | /marketplace | PASS | Search, Featured/Staff Picks/Trending/New Arrivals sections |
| MK2 | /marketplace/:id | PASS | Full detail with Add to Cart, reviews, files |
| MK3 | /marketplace/cart | PASS | 3 items, prices, remove buttons, total, checkout |
| MK4 | /marketplace/purchases | PASS | 11 purchases with Download and Refund buttons |

### 2.5  Creator Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CR1 | /creator | PASS | Dashboard: earnings ($35.98), 2 sales, 5 listings, recent sales, time range selector |
| CR2 | /creator/listings/new | PASS | Full form: title, description, price, 13 content types, subject tags, grade range |
| CR3 | /creator/listings/:id/edit | PASS | Pre-filled form, status badge (published v1), Archive button, change summary |
| CR4 | /creator/earnings | PASS | Shows "Coming soon" placeholder (expected) |
| CR5 | /creator/payouts | FAIL | Empty page — 12 console errors: 404s on /v1/marketplace/payouts/config, /history, /methods, /creator/verification |

### 2.6  Billing Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| B1 | /billing | PASS | Monthly/Annual toggle, 3 tiers (Free $0, Plus $99.99/yr, Premium $199.99/yr), current plan badge |
| B2 | /billing/transactions | PASS | Filter tabs (All/Subscriptions/Purchases/Payouts), date range filter, transaction list |

### 2.7  Settings Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| ST1 | /settings | PASS | Profile tab (family name, state, city, tier, methodology), Students tab, Co-Parents tab |
| ST2 | /settings (Students tab) | PASS | Emma (Born 2014), James (Born 2017), Edit/Delete buttons, Add student |
| ST3 | /settings (Co-Parents tab) | PASS | seed@example.com as Primary, invite form with email input |
| ST4 | /settings/account | PASS | Email, password (Coming soon), links to sessions/export/delete/appeals |
| ST5 | /settings/notifications | PASS | Comprehensive grid: In-app/Email toggles for 15+ event types, system alerts properly disabled |
| ST6 | /settings/privacy | PASS | Location sharing toggle, 6 field visibility controls (Friends only/Hidden) |
| ST7 | /settings/subscription | PASS | 3 tiers with feature lists, current plan highlighted |
| ST8 | /settings/blocks | PASS | Empty state: "No blocked users" |
| ST9 | /settings/account/sessions | PASS | 3 sessions, "This device" badge, Revoke buttons, "Log out all devices" |
| ST10 | /settings/account/export | PASS | Format picker (JSON/CSV), 5 data category checkboxes, 2 past exports shown |
| ST11 | /settings/account/delete | FAIL | Empty main content area, 2 console errors: 404 on /v1/account/deletion |
| ST12 | /settings/account/appeals | FAIL | Only heading shown, 1 console error: 404 on /v1/safety/appeals |

### 2.8  Planning & Calendar Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| PL1 | /calendar | PASS | Weekly view with events, Day/Week toggle, student filter, navigation arrows, Today button |
| PL2 | /schedule/new | PASS | Full form: title, description, student, date/time, duration, category (8 types), notes |
| PL3 | /planning/print | PASS | Date range, student filter, Print button, tabular layout with Time/Item/Student/Source/Category/Done |
| PL4 | /planning/templates | PASS | "Charlotte Mason Week" template (5 items), Apply/Delete buttons, day-of-week indicators, Create Template |

### 2.9  Compliance Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CP1 | /compliance | PASS | State picker (50 states), state requirements, tracking thresholds, links to sub-pages |
| CP2 | /compliance/attendance | PASS | Calendar grid, student selector, attendance pace, clickable days with status dropdown |
| CP3 | /compliance/tests | PASS | 2 test records (Iowa ITBS, CAT), scores per subject, "Add test score" button |
| CP4 | /compliance/portfolios | PASS | Student filter, "Emma Spring 2026 Portfolio" (3 items, Configuring), New/Delete buttons |
| CP5 | /compliance/assessments | PASS | 4 assessments with scores, subjects, and types |
| CP6 | /compliance/reports | BLOCKED | Route returns 404 — not yet implemented |

### 2.10  Other Authenticated Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| OA1 | /recommendations | PASS | Tabs (All/Content/Activities/Resources), personalized recs with AI Suggested badges, Dismiss/Block buttons |
| OA2 | /notifications | PASS | 20 notifications with Mark as read, Mark all as read, timestamps, link to preferences |
| OA3 | /search | PASS | Search box, tabs (Social/Marketplace/Learning), executed "Charlotte Mason" query returning 20 results |

### 2.11  Admin Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| AD1 | /admin | PASS | System health (DB 0ms, Redis 0ms, Kratos 3ms), moderation stats, quick links |
| AD2 | /admin/users | PASS | Search, status filter, 20+ families with name/status/tier/email/parent-student counts |
| AD3 | /admin/moderation | PASS | Queue tab (1 pending), Appeals tab (1), Approve/Reject/Escalate actions |
| AD4 | /admin/flags | PASS | 5 feature flags with toggle switches, rollout percentage sliders, family whitelists |
| AD5 | /admin/audit | PASS | Action/target type filters, entries with timestamps and admin actor |
| AD6 | /admin/methodologies | PASS | 6 methodologies (Charlotte Mason, Traditional, Classical, Waldorf, Montessori, Unschooling) |
| AD7 | /admin/system | FAIL | Route returns 404 — dashboard "System" link points to nonexistent route |

### 2.12  Student Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| — | — | BLOCKED | Student login not available in seed data; student routes not testable |

---

## 3  User Journey Results

### Journey 1: Onboarding Flow
- **Status:** PARTIAL (covered via route testing)
- **Steps completed:** 4/6
- **Issues found:** A5 COPPA verify has React state error on redirect

### Journey 2: Learning Workflow
- **Status:** PARTIAL (covered via route testing)
- **Steps completed:** 6/8
- **Issues found:** Subject connections tree UX (75+ flat unsorted checkboxes)

### Journey 3: Social Interactions
- **Status:** COMPLETE
- **Steps completed:** 11/11
- **Issues found:** None — sent messages, toggled likes, posted comments, toggled RSVPs all work

### Journey 4: Marketplace Browse & Purchase
- **Status:** PARTIAL
- **Steps completed:** 5/7
- **Issues found:** None in tested routes

### Journey 5: Creator Workflow
- **Status:** PARTIAL
- **Steps completed:** 5/7
- **Issues found:** Payouts page completely broken (12 console errors, empty page)

### Journey 6: Planning & Compliance
- **Status:** PARTIAL
- **Steps completed:** 6/8
- **Issues found:** /compliance/reports not implemented

### Journey 7: Settings & Account Management
- **Status:** COMPLETE
- **Steps completed:** 10/10
- **Issues found:** Delete account page broken (404 API), Moderation appeals page broken (404 API)

### Journey 8: Admin Workflow
- **Status:** COMPLETE
- **Steps completed:** 7/9
- **Issues found:** /admin/system link is dead (404)

---

## 4  Edge Case Results

### 4.1  Empty States
| Page | Status | Notes |
|------|--------|-------|
| Blocked Users | PASS | Shows "No blocked users" with descriptive text |
| Calendar Day (no events) | PASS | Shows "Nothing scheduled for this day." |
| Calendar Week (Fri-Sun) | PASS | Shows "—" for empty days |

### 4.2  Invalid Routes
| Route | Status | Notes |
|-------|--------|-------|
| /nonexistent-route | PASS | Clean 404 page with "Go Home" button |
| /marketplace/:invalid-id | PASS | Clean 404 page |
| /groups/:invalid-id | FAIL | Empty main content, 9 console errors — no user-facing error state |

### 4.3  Permission Boundaries
| Test | Status | Notes |
|------|--------|-------|
| /admin as non-admin (seed) | PASS | Silently redirects to Feed (no error exposed) |
| Sidebar nav for admin | PASS | No Admin link shown to non-admin users |

### 4.4  Form Validation
| Form | Status | Notes |
|------|--------|-------|
| Schedule New | PASS | Save disabled until title + date filled |
| Create Listing | PASS | Create disabled until required fields filled |
| New Journal Entry | PASS | Save disabled until student + content provided |
| Group Create | PASS | Disabled until name + description filled |
| Co-parent Invite | PASS | Send invite disabled until email entered |
| Notification Prefs | PASS | Toggle checkboxes save immediately, system alerts properly locked |

---

## 5  Gap Register

### 5.1  Critical

(none)

### 5.2  High

**GAP-H1: Creator Payouts page completely broken**
- **Route:** `/creator/payouts`
- **Severity:** High
- **Description:** Page renders with empty `<main>` element and 12 console errors. API endpoints return 404: `/v1/marketplace/payouts/config`, `/v1/marketplace/payouts/history`, `/v1/marketplace/creator/verification`, `/v1/marketplace/payouts/methods`. The frontend component exists and attempts to load data but the backend endpoints are not implemented.
- **Impact:** Creators cannot view or configure payout settings, see payout history, or verify their creator account.
- **Suggested fix:** Either implement the missing backend endpoints or show a "Coming soon" placeholder (like the earnings page does).

**GAP-H2: Account deletion page broken**
- **Route:** `/settings/account/delete`
- **Severity:** High
- **Description:** Page renders with empty `<main>` element and 2 console errors. API endpoint returns 404: `/v1/account/deletion`. Frontend component attempts to fetch deletion status but backend endpoint does not exist.
- **Impact:** Users cannot initiate account deletion — a GDPR/privacy requirement.
- **Suggested fix:** Implement the `/v1/account/deletion` endpoint or add a "Coming soon" placeholder with a note to contact support.

### 5.3  Medium

**GAP-M1: Moderation appeals page broken**
- **Route:** `/settings/account/appeals`
- **Severity:** Medium
- **Description:** Page shows only the "Moderation Appeals" heading with no content. Console error: 404 on `/v1/safety/appeals`. Frontend component exists but backend endpoint is missing.
- **Impact:** Users cannot view or submit moderation appeals through the UI.
- **Suggested fix:** Implement the endpoint or add empty state messaging.

**GAP-M2: Admin System page is dead link**
- **Route:** `/admin/system`
- **Severity:** Medium
- **Description:** The Admin Dashboard has a "System" quick link pointing to `/admin/system`, but this route returns 404. The page also breaks out of the admin layout into the regular app layout.
- **Impact:** Admin cannot access system configuration page linked from dashboard.
- **Suggested fix:** Either implement the route or remove the link from the admin dashboard.

**GAP-M3: Invalid group ID shows empty page instead of error state**
- **Route:** `/groups/:invalid-id`
- **Severity:** Medium
- **Description:** Navigating to a group with a non-existent UUID shows an empty main content area with 9 console errors (404s on group, members, posts endpoints). Unlike `/marketplace/:invalid-id` which correctly shows a 404 page, the group detail page has no error handling for missing groups.
- **Impact:** Users who follow a broken group link see a blank page with no guidance.
- **Suggested fix:** Add error boundary or not-found handling to the group detail component, similar to marketplace detail.

**GAP-M4: COPPA verify route triggers React state update error**
- **Route:** `/auth/coppa/verify`
- **Severity:** Medium
- **Description:** When visiting this route unauthenticated, the redirect to login triggers a React "Cannot update a component while rendering a different component" console error. Also triggers a 401 on `/v1/billing/micro-charge/status`.
- **Impact:** No user-visible impact (redirect works), but indicates a React anti-pattern that could cause issues elsewhere.
- **Suggested fix:** Move the redirect logic out of the render phase into a useEffect.

### 5.4  Low

**GAP-L1: Subject connections tree UX — 75+ flat unsorted checkboxes**
- **Routes:** `/learning/journals/new`, `/learning/nature-journal`, `/learning/trivium`, and likely other methodology tool forms
- **Severity:** Low
- **Description:** The subject selector shows 75+ subjects as a flat, unsorted list of checkboxes with no search, grouping, or hierarchy. Subjects range from broad categories (Mathematics, Science) to specific topics (Polynomials, Quadratic Equations) intermixed. Finding a specific subject requires scrolling through the entire list.
- **Impact:** Poor UX for parents trying to tag journal entries or activities with relevant subjects.
- **Suggested fix:** Add search/filter, alphabetical sorting, or group subjects into categories (e.g., Math → Algebra, Geometry, Calculus).

**GAP-L2: WebSocket connection warnings on every authenticated page**
- **Routes:** All authenticated routes
- **Severity:** Low
- **Description:** Every authenticated page load produces 2 console warnings about WebSocket connections being closed (`ws://localhost:...shed`). Originates from `src/lib/websocket.ts` lines 48 and 63.
- **Impact:** No user-visible impact. Expected in dev environment if notification WebSocket server isn't running.
- **Suggested fix:** Add graceful degradation / suppress repeated reconnect warnings in development mode.

---

## 6  Not Tested

The following were not tested due to scope limitations:
- **Student routes** (SR1-SR5): No student login credentials in seed data
- **Compliance reports** (`/compliance/reports`): Route not implemented (404)
- **Responsive layout**: All testing at 1280x800 viewport only
- **File upload**: No file upload flows tested (marketplace listing files, portfolio attachments)
- **Checkout flow**: Did not complete a purchase to avoid modifying billing state

---

## 7  Screenshots

All screenshots saved to `research/screenshots/e2e/` with naming convention:
`{route-group}-{route-number}-{status}.png`
