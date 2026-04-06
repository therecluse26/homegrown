# E2E Exploratory Testing Report — 2026-04-06

> **Agent:** Claude Opus 4.6
> **Scope:** Full application E2E via Playwright MCP
> **Procedure:** `specs/procedures/E2E_EXPLORATORY_TESTING.md`

---

## 1  Executive Summary

- **Test steps executed:** 133 (Phase 1: route smoke tests, Phase 2: user journeys, Phase 3: edge cases)
- **Pass:** 80 | **Warn:** 22 | **Fail:** 26 | **Blocked:** 5
- **Critical gaps found:** 0
- **High gaps found:** 7 (i18n missing on billing/recommendations, content players blank, search API broken, compliance/admin endpoints fail, projects 404)
- **Medium gaps found:** 19 (self-friending, email-as-name, builder edit mode, creator/payout endpoints, version history, payment methods, family profile, streaks 404, snake_case subjects, tools disabled, admin password mismatch, schedule templates, MFA blank, settings endpoints, invalid entity ID handling, add-to-cart UX, checkout endpoint, schedule item creation)
- **Low gaps found:** 5 (missing favicon, content viewer seed data, notification header contradiction, calendar day view, compliance spinbuttons)

**Key patterns:**
1. **Missing i18n:** billing, recommendations namespaces entirely untranslated (~70+ MissingTranslationErrors)
2. **Unimplemented backend endpoints:** search, compliance, admin user management, MFA, schedule creation, checkout
3. **Missing error states:** entity detail pages show empty `<main>` for invalid IDs instead of "Not Found"
4. **Working well:** Feed, social (like/comment/post), marketplace browse/cart, creator dashboard, calendar, notification system, settings, responsive layout

---

## 2  Route Smoke Test Results

### 2.1  Auth & Legal Routes (Unauthenticated)

| # | Route | Status | Notes |
|---|-------|--------|-------|
| A1 | `/auth/login` | PASS | Login form renders correctly with email/password fields |
| A2 | `/auth/register` | PASS | Registration form with Turnstile captcha, ToS/Privacy links |
| A3 | `/auth/recovery` | PASS | Password recovery form renders correctly |
| A4 | `/auth/verification` | PASS | Email verification form with code input and resend button |
| A5 | `/auth/coppa/verify` | WARN | Page renders but fires repeated 401 errors on `/v1/billing/micro-charge/status` — page accessible unauthenticated but makes authenticated API calls |
| A6 | `/auth/accept-invite/test-token-123` | PASS | Invitation acceptance page renders with Accept/Decline buttons |
| L1 | `/legal/terms` | PASS | Full Terms of Service content renders |
| L2 | `/legal/privacy` | PASS | Full Privacy Policy content renders |
| L3 | `/legal/guidelines` | PASS | Full Community Guidelines content renders |

### 2.2  Onboarding

| # | Route | Status | Notes |
|---|-------|--------|-------|

### 2.3  Home & Social Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| S1 | `/` (Feed) | PASS | Feed renders with posts, post composer (5 types), pagination. User shows as `seed@example.com` instead of display name. WebSocket warnings (non-blocking). |
| S2 | `/friends` | PASS | 20 friends listed with search, tabs (All/Incoming/Sent), Message/Unfriend actions. All family avatars show "TF" initials — likely avatar generation bug. |
| S3 | `/friends/discover` | PASS | Renders with search and filter buttons. "No families found" — expected since all are already friends. |
| S4 | `/messages` | PASS | 6 conversations listed with previews, unread badge counts, timestamps. |
| S5 | `/messages/{conversationID}` | PASS | Message history with send box, search and mute buttons. |
| S6 | `/groups` | PASS | 9 groups with member counts, methodology tags, My Groups/Discover tabs. |
| S7 | `/groups/new` | PASS | Create Group form with name, description, join policy radio buttons. |
| S8 | `/groups/{groupID}` | PASS | Group detail with header, description, Posts/Members tabs, Leave Group button. |
| S9 | `/groups/{groupID}/manage` | PASS | Group management with members list, roles, promote/remove actions, pending requests tab. |
| S10 | `/events` | PASS | Events list with RSVP buttons, attendee counts, capacity indicators. All events in Feb/Mar (past dates from seed). |
| S11 | `/events/new` | PASS | Comprehensive event creation form: title, description, dates, recurring, location type, capacity, visibility, group link, methodology tag. |
| S12 | `/events/{eventID}` | PASS | Event detail with RSVP (Going/Interested/Not Going), attendee list with tabs, Export CSV, Cancel Event. |
| S13 | `/post/{post1ID}` | PASS | Post detail with comments (reply/edit/delete/report). Title shows "seed@example.com's post" instead of display name. |
| S14 | `/family/{seedFamilyID}` | WARN | Profile renders but shows "Add Friend" button for own family — should not allow self-friending. Minimal profile content (no posts/members shown). |

### 2.4  Learning Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| LR1 | `/learning` | WARN | Dashboard renders with student progress cards (Emma/James), quick actions. 8 console errors on `/students/{id}/streak` endpoints (404). |
| LR2 | `/learning/activities` | PASS | Activity log with student selector, type/subject filters, seed entries. |
| LR3 | `/learning/journals` | PASS | Journal list with student/type selectors. |
| LR4 | `/learning/journals/new` | PASS | Rich journal form with 70+ subject tree (flat, unsorted). |
| LR5 | `/learning/reading-lists` | PASS | Emma's Books list (0/3 read). |
| LR6 | `/learning/progress/{emmaStudentID}` | PASS | Emma's stats, hours by subject, recent activity. Subject names in snake_case (e.g., "nature_study"). |
| LR7 | `/learning/progress/{jamesStudentID}` | PASS | James's stats, hours by subject, recent activity. Same snake_case subject names. |
| LR8 | `/learning/grades` | PASS | Assessments page renders. |
| LR9 | `/learning/quiz/{quizDef1ID}` | FAIL | Blank page — quiz-sessions API returns errors. No quiz content rendered. |
| LR10 | `/learning/video/{videoDef1ID}` | FAIL | Blank page — video-progress API returns errors. No video player rendered. |
| LR11 | `/learning/read/{activityDef1ID}` | WARN | Content Viewer renders but shows "Content not found" — seed activity may not have readable content type. No console errors. |
| LR12 | `/learning/sequence/{sequenceDef1ID}` | FAIL | Blank page — sequence-progress API returns errors. |
| LR13 | `/learning/session-log/{studentSession1ID}` | PASS | Session activity log renders (empty state). |
| LR14 | `/learning/session` | PASS | Session launcher with student selector (Emma age 12, James age 9). |
| LR15 | `/learning/projects` | FAIL | Blank page — 404 on `/v1/learning/projects` endpoint. Route not implemented in backend. |
| LR16 | `/learning/tools` | WARN | All tools show "Disabled" status. Duplicate "Assessments" entry in tools list. |
| LR17 | `/learning/nature-journal` | PASS | Charlotte Mason nature journal form with weather, season, location, specimen, subject tree. |
| LR18 | `/learning/trivium-tracker` | PASS | Classical trivium with Grammar/Logic/Rhetoric stage selector, rich form. |
| LR19 | `/learning/rhythm-planner` | PASS | Waldorf rhythm planner with day-of-week, time blocks, activity categories, cross-methodology note. |
| LR20 | `/learning/observation-logs` | PASS | Montessori observation log with concentration levels, materials, work chosen by child, 70+ subjects. |
| LR21 | `/learning/habit-tracking` | PASS | Charlotte Mason habit tracker with check-in buttons (Yes/Partial/Not today), preset habits. |
| LR22 | `/learning/interest-led-log` | PASS | Unschooling log with exploration method buttons, spark description, resources, subjects. |
| LR23 | `/learning/handwork-projects` | PASS | Waldorf handwork with craft types (13 options), project status, materials, photo upload. |
| LR24 | `/learning/practical-life` | PASS | Montessori practical life with 5 skill areas, 4 mastery levels, examples per area. |
| LR25 | `/learning/quiz/{quizDef1ID}/score` | FAIL | Blank page — same quiz-sessions API errors as LR9. |

### 2.5  Marketplace Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| MK1 | `/marketplace` | PASS | Browse page with Featured, Staff Picks, Trending, New Arrivals sections. 20+ listings with prices, ratings, publisher names. Search bar with filter button. |
| MK2 | `/marketplace/listings/{listing1ID}` | PASS | Listing detail with title, price ($29.99), Add to Cart, description, subject tags (snake_case: `language_arts`), included files, reviews with star ratings. |
| MK3 | `/marketplace/cart` | PASS | Shopping cart with 1 item (Living Books Video Series $19.99), Remove button, total, Proceed to Checkout. |
| MK4 | `/marketplace/purchases` | PASS | Purchase history with 10 purchases, each with Download and Request Refund buttons, prices, dates. |
| MK5 | `/marketplace/purchases/{purchase1ID}/refund` | PASS | Refund request form with reason dropdown (5 options), details textarea, disabled submit until reason selected. |
| MK6 | `/marketplace/listings/{listing1ID}/versions` | FAIL | "Something went wrong" — 4 console errors on `/versions` endpoint. Backend route may not be implemented. |

### 2.6  Creator Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CR1 | `/creator` | PASS | Dashboard with earnings ($35.98), sales (2), avg rating, pending payout ($0), recent sales, My Listings (5 published). Time range filter. |
| CR2 | `/creator/listings/new` | PASS | Create listing form with title, description, price, content type (13 types), publisher, subject tags, grade range. |
| CR3 | `/creator/listings/{listing1ID}/edit` | PASS | Edit form pre-filled with data, status (published v1), Archive button, change summary. |
| CR4 | `/creator/quiz-builder` | PASS | New quiz builder with title field, question/point count, Add Question button. |
| CR5 | `/creator/quiz-builder/{quizDef1ID}` | WARN | Quiz builder in edit mode shows empty state — title blank, 0 questions/0 points. Existing quiz data not loaded from API. |
| CR6 | `/creator/sequence-builder` | PASS | New sequence builder with title, description, step count, Add Step button. |
| CR7 | `/creator/sequence-builder/{sequenceDef1ID}` | WARN | Same as CR5: edit mode shows empty state — existing sequence data not loaded. |
| CR8 | `/creator/payouts` | WARN | Payout Setup renders "Creator Verification Required" with link, but fires 16 console errors on payouts/history, payouts/config, creator/verification, payouts/methods. |
| CR9 | `/creator/verification` | FAIL | "Something went wrong" — 4 errors on `/v1/marketplace/creator/verification` endpoint. |
| CR10 | `/creator/reviews` | FAIL | "Something went wrong" — 4 errors on `/v1/marketplace/creator/reviews` endpoint. |

### 2.7  Billing Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| B1 | `/billing` | PASS | Pricing page with 3 tiers (Free/$0, Plus/$99.99/yr, Premium/$199.99/yr), feature lists, Monthly/Annual toggle (save 17%). Current plan: Premium. |
| B2 | `/billing/payment-methods` | FAIL | "Something went wrong" — 4 errors on `/v1/billing/payment-methods` endpoint. |
| B3 | `/billing/transactions` | WARN | Transaction history renders with data and filters, but 4 `MissingTranslationError`s — type/status badges show raw i18n keys (`billing.transactions.type.subscription_payment`). |
| B4 | `/billing/subscription` | FAIL | Page structure renders but ALL text shows raw i18n keys (`billing.subscription.title`, `billing.subscription.cancel`, etc.). 24 `MissingTranslationError` console errors. |
| B5 | `/billing/invoices` | FAIL | Same as B4 — completely untranslated. All text shows raw i18n keys. 18 `MissingTranslationError` errors. |

### 2.8  Settings Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| ST1 | `/settings` | PASS | Family Settings with 3 tabs (Profile/Students/Co-Parents). Shows family name "Test Fam", state "Arizona", methodology "Classical". Edit/Change buttons. |
| ST2 | `/settings/notifications` | PASS | Full notification preferences grid with In-app/Email columns, 7 notification categories, toggle switches. |
| ST3 | `/settings/notifications/history` | WARN | 20 notifications listed with titles, timestamps, and read/unread status. Header says "No notifications" despite items showing below — contradictory UI. |
| ST4 | `/settings/subscription` | PASS | 3 pricing tiers (Free/Plus/Premium), feature lists, Monthly/Annual toggle. Current plan: Premium. |
| ST5 | `/settings/account` | PASS | Account settings with email display, password "Coming soon", links to sessions/export/delete. |
| ST6 | `/settings/account/sessions` | PASS | 2 active sessions with browser info, IP, last active. Revoke button on non-current session. |
| ST7 | `/settings/account/export` | PASS | Data export with JSON/CSV format, data category checkboxes, past exports list. |
| ST8 | `/settings/account/delete` | WARN | Account deletion page renders with deletion info, confirmation checkbox, family name confirmation field. 4 console errors on `/v1/account/deletion` endpoint (404). |
| ST9 | `/settings/account/delete/student/{emmaStudentID}` | PASS | Student deletion for Emma with COPPA immediate-deletion notice, data deletion list, confirmation checkbox, name confirmation field. |
| ST10 | `/settings/account/appeals` | WARN | "Moderation Appeals" heading renders but no content below. 3 errors on `/v1/safety/appeals` (404). Backend endpoint likely not implemented. |
| ST11 | `/settings/blocks` | WARN | "Blocked Users" heading renders but no content. 4 errors on `/v1/social/blocks` (500 Server Error). |
| ST12 | `/settings/privacy` | PASS | Privacy Controls with location sharing toggle, 6 field visibility dropdowns (family name, location, methodology, parent names, children's names, children's ages), all "Friends only". No errors. |
| ST13 | `/settings/account/mfa` | FAIL | Page title "Two-Factor Authentication" but `<main>` is completely empty. 3 errors on `/v1/auth/mfa/status` (404). Backend endpoint not implemented. |
| ST14 | `/settings/subscription/manage` | PASS | Subscription Management with current plan (premium, $9.99/monthly), next billing date (May 5, 2026), Change plan / Cancel subscription buttons, links to Payment Methods and Transaction History. |

### 2.9  Planning & Calendar Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| PL1 | `/calendar` | PASS | Full week calendar (Apr 6-12, 2026), student filter (All/Emma/James), Add Item/Export/Print, color-coded items (Learning/Events/Attendance/Schedule). |
| PL2 | `/calendar/day/2026-04-06` | WARN | Day view URL renders the same week view — URL param doesn't switch to single-day layout. Heading shows "April 6 – 12, 2026". |
| PL3 | `/calendar/week/2026-04-06` | PASS | Week view renders correctly (same as PL1 default). |
| PL4 | `/schedule/new` | PASS | New schedule item form with title, description, student selector, date (pre-filled today), start/end time, duration, category (8 types), notes. |
| PL5 | `/schedule/{schedItem1ID}/edit` | PASS | Edit form pre-filled with "Read Aloud: Charlotte's Web", student Emma, date 2026-04-05, time 09:00–09:45, category Reading. Delete/Save buttons. |
| PL6 | `/planning/templates` | FAIL | "Schedule Templates" heading and "Create Template" button render but shows "Something went wrong". 4 errors on `/v1/planning/schedule-templates`. |
| PL7 | `/planning/print` | PASS | Print schedule with date range, student filter, Print button, full tabular schedule by day with Time/Item/Student/Source/Category/Done columns. |
| PL8 | `/planning/coop` | PASS | Co-op Coordination with group selector (9 groups), "Charlotte Mason Co-op" selected showing 2 members, weekly navigation. |

### 2.10  Compliance Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CP1 | `/compliance` | WARN | Compliance Setup with 50 states (Texas selected), state requirements, days/hours thresholds. 4 warnings — spinbutton values are "undefined" instead of numbers. |
| CP2 | `/compliance/attendance` | WARN | Attendance Tracker renders with student selector, April 2026 calendar, legend. 6 errors on `/v1/compliance/attendance/summary` and student attendance endpoints. Calendar days missing. |
| CP3 | `/compliance/assessments` | FAIL | "Assessment Records" heading + "Something went wrong". 4 errors on `/v1/compliance/assessments`. |
| CP4 | `/compliance/tests` | FAIL | "Standardized Tests" heading + "Something went wrong". 4 errors on `/v1/compliance/tests` (404). |
| CP5 | `/compliance/portfolios` | WARN | "Portfolios" heading with student filter and "New Portfolio" button, but no portfolio list. 3 errors on `/v1/compliance/portfolios`. |
| CP6 | `/compliance/portfolios/{complyPortfolio1ID}` | WARN | "Portfolio not found" with graceful error handling (Back to Portfolios link). 4 errors — seed portfolio ID not in DB or API not implemented. |
| CP7 | `/compliance/transcripts` | WARN | "Transcripts" heading with student filter and "New Transcript" button, but no transcript list. 3 errors on `/v1/compliance/transcripts`. |
| CP8 | `/compliance/transcripts/{complyTranscript1ID}` | WARN | "Transcript not found" with graceful error handling (Back to Transcripts link). 4 errors — seed transcript ID not in DB or API not implemented. |

### 2.11  Other Authenticated Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| OA1 | `/recommendations` | FAIL | Page renders recommendation data (Charlotte Mason curriculum, Nature Journal, Charlotte's Web) but ALL UI text shows raw i18n keys (`recommendations.title`, `recommendations.filter.all`, `recommendations.dismiss`, etc.). 50 `MissingTranslationError` console errors. |
| OA2 | `/search` | PASS | Search page with search box, 3 tabs (Social/Marketplace/Learning), empty state prompt. |
| OA3 | `/notifications` | PASS | 20 notifications with titles, descriptions, timestamps, "Mark all as read", individual "Mark as read" buttons, link to manage preferences. |

### 2.12  Admin Routes

**Login:** `admin@example.com` / `SeedPassword123!` (procedure doc says `AdminPassword123!` which is wrong). Onboarding skip fails (404 on `/v1/onboarding/skip`), but direct nav to `/admin` works.

| # | Route | Status | Notes |
|---|-------|--------|-------|
| AD1 | `/admin` | PASS | Dashboard with System Health (DB 0ms, Redis 0ms, Kratos 1ms), moderation stats (0 Pending Reports, 1 Pending Appeal, 1 Unreviewed Flag), quick links (User Mgmt, Moderation, Audit, System). |
| AD2 | `/admin/users` | WARN | User Management renders search box and status filter (All/Active/Suspended/Banned) but shows "No users found". 4 errors on `/v1/admin/users` (500 Server Error). |
| AD3 | `/admin/users/{seedParentID}` | FAIL | Empty `<main>`. 4 errors on `/v1/admin/users/{id}` (500 Server Error). |
| AD4 | `/admin/moderation` | PASS | Moderation Queue with tabs (Queue 1, Appeals 1). 1 pending spam report on a post with Approve/Reject/Escalate buttons. |
| AD5 | `/admin/flags` | FAIL | "Feature Flags" heading + "Something went wrong". 4 errors on `/v1/admin/feature-flags` (404). |
| AD6 | `/admin/audit` | PASS | Audit Log with action/target filters, 3 entries (flag_create, user_suspend, content_remove) by admin@example.com with timestamps. |
| AD7 | `/admin/methodologies` | FAIL | Empty `<main>`. 3 errors on `/v1/admin/methodology-configs`. Backend endpoint missing. |

### 2.13  Student Routes

Student routes require token-based session auth (not Kratos browser login). Marked BLOCKED per procedure §4.1.14.

| # | Route | Status | Notes |
|---|-------|--------|-------|
| SR1 | `/student` | BLOCKED | Requires student session token — cannot authenticate via browser login |
| SR2 | `/student/quiz/{quizDef1ID}` | BLOCKED | Same — requires student token |
| SR3 | `/student/video/{videoDef1ID}` | BLOCKED | Same — requires student token |
| SR4 | `/student/read/{activityDef1ID}` | BLOCKED | Same — requires student token |
| SR5 | `/student/sequence/{sequenceDef1ID}` | BLOCKED | Same — requires student token |

---

## 3  User Journey Results

### Journey 1 — Onboarding (Partial)

Admin onboarding tested: after login as `admin@example.com`, onboarding guard redirects to `/onboarding`. "Skip setup" button returns 404 on `/v1/onboarding/skip`. Direct nav to `/admin` bypasses. See GAP-E2E-M11.

### Journey 2 — Learning Workflow

| Step | Action | Result |
|------|--------|--------|
| 1 | Navigate to `/learning` | PASS — dashboard renders with Emma/James progress cards |
| 2 | Click "View details" for Emma | PASS — navigates to `/learning/progress/{emmaID}` with stats |
| 3 | Navigate to Activity Log | PASS — shows activities with student/type filters |
| 4 | Navigate to Session Launcher | PASS — student selector shows Emma (12) and James (9) |

### Journey 3 — Social Interactions

| Step | Action | Result |
|------|--------|--------|
| 1 | Navigate to Feed (`/`) | PASS — 20+ posts, post composer visible |
| 2 | Click Like on first post | PASS — button toggles to `[active]`/`[pressed]` state |
| 3 | Navigate to post detail | PASS — like count shows "1" (persisted), author/timestamp correct |
| 4 | Write a comment | PASS — comment posted with Reply/Edit/Delete/Report buttons |
| 5 | Unlike post | PASS — button shows "Unlike" with count "1" on mobile view |

### Journey 4 — Marketplace Browse & Purchase

| Step | Action | Result |
|------|--------|--------|
| 1 | Navigate to `/marketplace` | PASS — Featured, Staff Picks, 20+ listings |
| 2 | Click listing detail | PASS — title, price, Add to Cart, reviews, files section |
| 3 | Click "Add to Cart" | WARN — item added but **no visual feedback** (no toast, badge, or button state change) |
| 4 | Navigate to `/marketplace/cart` | PASS — 2 items, $49.98 total, Proceed to Checkout |
| 5 | Click "Proceed to Checkout" | FAIL — error on `/v1/marketplace/cart/checkout` (endpoint missing) |

### Journey 5 — Creator Workflow

| Step | Action | Result |
|------|--------|--------|
| 1 | Navigate to `/creator` | PASS — dashboard with earnings, sales, listings |
| 2 | Click "New Listing" | PASS — create form with all fields, 13 content types |
| 3 | Fill form and review | PASS — form validates required fields, Save enables when filled |

### Journey 6 — Planning & Compliance

| Step | Action | Result |
|------|--------|--------|
| 1 | Navigate to `/schedule/new` | PASS — form with student selector, categories |
| 2 | Fill and submit schedule item | FAIL — error on `/v1/planning/schedule-items` (API error) |
| 3 | Navigate to `/calendar` | PASS — week view with color-coded items |

### Journey 7 — Settings & Account

| Step | Action | Result |
|------|--------|--------|
| 1 | Navigate to `/settings` | PASS — Profile tab with family info |
| 2 | Click Students tab | PASS — Emma (2014), James (2017) with Edit/Delete/Add |
| 3 | Navigate to Notification Preferences | PASS — 15+ notification types, In-app/Email grid, critical types disabled |

### Journey 8 — Admin Workflow

Covered in Phase 1 route testing (AD1-AD7). Dashboard, moderation queue, and audit log work. User management, flags, and methodology config fail (see GAP-E2E-H5).

---

## 4  Edge Case Results

### 4.1  404 / Not Found Handling

| Test | Result | Notes |
|------|--------|-------|
| Non-existent route (`/totally-nonexistent-route`) | PASS | Clean "Page not found" with description and "Go Home" button. No errors. |
| Invalid post UUID (`/post/00000000-...`) | FAIL | Empty `<main>`, 4 errors. No "Post not found" message. See GAP-E2E-M12. |
| Non-UUID post ID (`/post/not-a-uuid`) | FAIL | Same — empty `<main>`, 4 errors. No validation of UUID format. |
| Invalid listing UUID (`/marketplace/listings/00000000-...`) | FAIL | Empty `<main>`, 3 errors. No "Listing not found" message. |

### 4.2  Search Functionality

| Test | Result | Notes |
|------|--------|-------|
| Global search (`/search?q=Charlotte+Mason`) — Social tab | FAIL | API error on `/v1/search/search?q=...&scope=social`. See GAP-E2E-H6. |
| Global search — Marketplace tab | FAIL | Same API error with `scope=marketplace` |
| Global search — Learning tab | FAIL | Same pattern — all scopes fail |
| Header search bar | PASS | Redirects correctly to `/search?q=...` but then hits same broken API |
| Marketplace built-in search | PASS | Client-side filtering works perfectly (17 results for "nature") |

### 4.3  Form Validation

| Test | Result | Notes |
|------|--------|-------|
| Post composer — empty submit | PASS | Post button correctly disabled when textbox is empty |
| Comment — empty submit | PASS | Submit button correctly disabled when textbox is empty |
| Create Listing — required fields | PASS | Create Listing button disabled until required fields filled |
| Schedule item — required fields | PASS | Save Item button disabled until Title filled |

### 4.4  Responsive Layout (375×812 iPhone SE)

| Test | Result | Notes |
|------|--------|-------|
| Feed page | PASS | Navigation switches to bottom tab bar, header simplifies, posts readable |
| Post composer | PASS | Full-width, post type buttons visible |
| Like/comment buttons | PASS | Accessible and functional |
| Overall layout | PASS | No horizontal overflow, proper spacing |

### 4.5  Add-to-Cart UX

| Test | Result | Notes |
|------|--------|-------|
| Add item to cart | WARN | Item added successfully but **zero visual feedback** — no toast, no badge increment, no button state change. See GAP-E2E-M13. |

### 4.6  Checkout Flow

| Test | Result | Notes |
|------|--------|-------|
| Proceed to Checkout from cart | FAIL | Error on `/v1/marketplace/cart/checkout`. See GAP-E2E-M14. |

---

## 5  Gap Register

### 5.1  Critical

_None found yet_

### 5.2  High

#### GAP-E2E-H0 — Billing pages missing all i18n translations

| Field | Value |
|-------|-------|
| **Route(s)** | `/billing/subscription`, `/billing/invoices`, `/billing/transactions` |
| **Observed** | Subscription management and invoice pages display raw i18n message keys instead of translated text (e.g., `billing.subscription.title`, `billing.invoice.title`, `billing.transactions.type.subscription_payment`). 24+ `MissingTranslationError` errors on subscription page, 18+ on invoices. Transaction history partially affected (4 errors on type/status badges). |
| **Expected** | All billing pages should display human-readable text. Translation message catalog is missing or not loaded for the `billing.*` namespace. |
| **Console errors** | `MissingTranslationError` × 50+ across billing routes |
| **Screenshot** | N/A |

#### GAP-E2E-H1 — Content player routes (Quiz, Video, Sequence) all render blank

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/quiz/{id}`, `/learning/video/{id}`, `/learning/sequence/{id}`, `/learning/quiz/{id}/score` |
| **Observed** | All four content player pages render completely blank (empty `<main>`) with console errors on progress-tracking API endpoints (`quiz-sessions`, `video-progress`, `sequence-progress`). These endpoints appear to return 404 or 500. |
| **Expected** | Content players should render the quiz questions, video player, or sequence steps. If no active session exists, should show a "Start" button or informational message rather than a blank page. |
| **Console errors** | `Failed to load resource` on `/v1/learning/quiz-sessions/{id}`, `/v1/learning/video-progress/{id}`, `/v1/learning/sequence-progress/{id}` |
| **Screenshot** | `learning-LR9-FAIL.png` |

#### GAP-E2E-H3 — Recommendations page missing all i18n translations

| Field | Value |
|-------|-------|
| **Route(s)** | `/recommendations` |
| **Observed** | All UI text shows raw i18n keys (`recommendations.title`, `recommendations.filter.all`, `recommendations.filter.content`, `recommendations.dismiss`, `recommendations.blockCategory`, `recommendations.badge.ai`, `recommendations.type.content`, `recommendations.type.resource`). Recommendation data loads correctly but all chrome is untranslated. **50 `MissingTranslationError` console errors.** |
| **Expected** | All UI text should be translated. Translation message catalog is missing for the `recommendations.*` namespace. |
| **Console errors** | `MissingTranslationError` × 50 |
| **Screenshot** | N/A |

#### GAP-E2E-H5 — Admin endpoints (users, user detail, feature flags, methodologies) fail

| Field | Value |
|-------|-------|
| **Route(s)** | `/admin/users`, `/admin/users/{id}`, `/admin/flags`, `/admin/methodologies` |
| **Observed** | User Management returns 500 Server Error (shows "No users found"). User detail returns 500 (empty main). Feature Flags returns 404 ("Something went wrong"). Methodology Config returns errors (empty main). Only Dashboard, Moderation Queue, and Audit Log work correctly. |
| **Expected** | Admin endpoints should return data. User list should show seeded users. Feature flags should list flags. Methodology config should show available methodologies. |
| **Console errors** | `Failed to load resource: 500/404` across `/v1/admin/users`, `/v1/admin/feature-flags`, `/v1/admin/methodology-configs` |
| **Screenshot** | N/A |

#### GAP-E2E-H4 — Compliance endpoints (assessments, tests, attendance, portfolios, transcripts) all fail

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance/assessments`, `/compliance/tests`, `/compliance/attendance`, `/compliance/portfolios`, `/compliance/portfolios/{id}`, `/compliance/transcripts`, `/compliance/transcripts/{id}` |
| **Observed** | Assessment Records and Standardized Tests show "Something went wrong" (404). Attendance Tracker renders calendar structure but fails to load data (6 errors). Portfolio/Transcript list pages render headers and filters but no data (3 errors each). Individual portfolio/transcript pages show "not found" (4 errors each). |
| **Expected** | Compliance API endpoints should return data or graceful empty states. Seed data should include compliance records. |
| **Console errors** | Multiple 404/500 errors across `/v1/compliance/*` endpoints |
| **Screenshot** | N/A |

#### GAP-E2E-H6 — Global search API completely broken

| Field | Value |
|-------|-------|
| **Route(s)** | `/search?q=...&scope=social`, `/search?q=...&scope=marketplace`, `/search?q=...&scope=learning` |
| **Observed** | All search API calls to `/v1/search/search` fail with server errors across all three scopes (Social, Marketplace, Learning). The search page renders correctly with tabs and search input, but no results are ever returned. Header search bar redirects correctly but hits the same broken endpoint. |
| **Expected** | Search API should return matching results. Note: marketplace's own built-in search (client-side filtering on `/marketplace`) works perfectly. |
| **Console errors** | `Failed to load resource` × 2 per search attempt on `/v1/search/search?q=...&scope=...` |
| **Screenshot** | N/A |

#### GAP-E2E-H2 — Projects page returns 404 — backend route not implemented

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/projects` |
| **Observed** | Page renders blank; network request to `/v1/learning/projects` returns 404. The frontend route exists but the backend endpoint is missing. |
| **Expected** | Either implement the backend route or remove/hide the frontend route |
| **Console errors** | `Failed to load resource: 404 (Not Found)` on `/v1/learning/projects` |
| **Screenshot** | N/A |

### 5.3  Medium

#### GAP-E2E-M1 — Family profile shows "Add Friend" for own family

| Field | Value |
|-------|-------|
| **Route(s)** | `/family/{seedFamilyID}` |
| **Observed** | Viewing your own family profile displays an "Add Friend" button — implies you can friend yourself |
| **Expected** | Own family profile should not show "Add Friend" button; should show "This is your family" or edit link |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-M2 — User displays as email address instead of display name

| Field | Value |
|-------|-------|
| **Route(s)** | `/` (Feed), `/post/{id}`, top-right header, and throughout the app |
| **Observed** | Seed user shows as "seed@example.com" in post author, comments, page titles, and header avatar area |
| **Expected** | Should display the user's display name (e.g., from Kratos identity traits) — email fallback should only be used if no name is set |
| **Console errors** | None |
| **Screenshot** | `social-S1-PASS.png` |

#### GAP-E2E-M2b — Creator builders (Quiz/Sequence) don't load existing data in edit mode

| Field | Value |
|-------|-------|
| **Route(s)** | `/creator/quiz-builder/{quizDef1ID}`, `/creator/sequence-builder/{sequenceDef1ID}` |
| **Observed** | Both builders in edit mode show empty state (blank title, 0 questions/steps) — existing quiz/sequence data is not fetched from the API |
| **Expected** | Edit mode should pre-populate with the existing quiz questions or sequence steps |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-M2c — Creator verification and reviews endpoints return errors

| Field | Value |
|-------|-------|
| **Route(s)** | `/creator/verification`, `/creator/reviews` |
| **Observed** | Both pages show "Something went wrong" with 4 console errors each on their respective API endpoints. Payout Setup also fires 16 errors on related endpoints. |
| **Expected** | Pages should render verification form and review list respectively, or show meaningful empty states |
| **Console errors** | `Failed to load resource` on `/v1/marketplace/creator/verification`, `/v1/marketplace/creator/reviews`, `/v1/marketplace/payouts/*` |
| **Screenshot** | N/A |

#### GAP-E2E-M2d — Listing version history endpoint fails

| Field | Value |
|-------|-------|
| **Route(s)** | `/marketplace/listings/{id}/versions` |
| **Observed** | "Something went wrong" with 4 errors on the `/versions` endpoint |
| **Expected** | Should show version history for the listing or a clean empty state |
| **Console errors** | `Failed to load resource` on `/v1/marketplace/listings/{id}/versions` |
| **Screenshot** | N/A |

#### GAP-E2E-M2e — Payment methods endpoint fails

| Field | Value |
|-------|-------|
| **Route(s)** | `/billing/payment-methods` |
| **Observed** | "Something went wrong" with 4 errors on `/v1/billing/payment-methods` |
| **Expected** | Should show payment methods list with add/remove functionality |
| **Console errors** | `Failed to load resource` on `/v1/billing/payment-methods` |
| **Screenshot** | N/A |

#### GAP-E2E-M3 — Family profile page has minimal content

| Field | Value |
|-------|-------|
| **Route(s)** | `/family/{seedFamilyID}` |
| **Observed** | Profile page only shows family name, description, and avatar. No posts, members, methodology, or other profile details are displayed |
| **Expected** | Profile should show family members, methodology, location (if public), recent posts, and groups |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-M5 — Learning dashboard fires 404s on student streak endpoints

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning` |
| **Observed** | Dashboard renders correctly but fires 8 console errors: repeated 404s on `/v1/learning/students/{emmaID}/streak` and `/v1/learning/students/{jamesID}/streak`. Streak feature appears unimplemented. |
| **Expected** | Either implement the streak API or suppress the calls if the feature isn't ready |
| **Console errors** | `Failed to load resource: 404` × 8 on `/v1/learning/students/{id}/streak` |
| **Screenshot** | N/A |

#### GAP-E2E-M6 — Subject names display in snake_case instead of human-readable format

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/progress/{studentID}` (both Emma and James) |
| **Observed** | "Hours by subject" section displays subjects as `nature_study`, `mathematics`, `science`, `reading` instead of "Nature Study", "Mathematics", etc. |
| **Expected** | Subject names should be formatted as Title Case for display |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-M7 — Learning tools page shows all tools as "Disabled" with duplicate entry

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/tools` |
| **Observed** | Every tool in the list shows a "Disabled" badge. Also "Assessments" appears twice in the tools list. |
| **Expected** | Tools matching the family's methodology should be enabled by default. No duplicate entries. |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-M11 — Admin password mismatch in testing procedure + onboarding skip fails for admin

| Field | Value |
|-------|-------|
| **Route(s)** | `/auth/login`, `/onboarding` |
| **Observed** | E2E procedure doc (§2.2) says admin password is `AdminPassword123!` but actual password is `SeedPassword123!`. After admin login, onboarding guard redirects to `/onboarding` and "Skip setup" button fails with 404 on `/v1/onboarding/skip`. Direct navigation to `/admin` bypasses onboarding. |
| **Expected** | Procedure doc should match actual credentials. Admin onboarding skip should work or admin should bypass onboarding entirely. |
| **Console errors** | 404 on `/v1/onboarding/skip`, 404 on `/v1/onboarding/progress` |
| **Screenshot** | N/A |

#### GAP-E2E-M10 — Schedule Templates endpoint fails

| Field | Value |
|-------|-------|
| **Route(s)** | `/planning/templates` |
| **Observed** | "Schedule Templates" heading and "Create Template" button render but shows "Something went wrong". 4 errors on `/v1/planning/schedule-templates`. |
| **Expected** | Should list existing templates or show empty state |
| **Console errors** | `Failed to load resource` × 4 on `/v1/planning/schedule-templates` |
| **Screenshot** | N/A |

#### GAP-E2E-L4 — Calendar day view URL doesn't switch to day layout

| Field | Value |
|-------|-------|
| **Route(s)** | `/calendar/day/2026-04-06` |
| **Observed** | Navigating to day view URL renders the same week view (heading "April 6 – 12, 2026"). The URL param is ignored. |
| **Expected** | Day view should show a single-day detailed schedule |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-L5 — Compliance setup spinbuttons show "undefined" values

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance` |
| **Observed** | Days required per year and Hours required per year spinbuttons have "undefined" as their values, causing 4 React warnings about invalid input values. |
| **Expected** | Spinbuttons should show 0 or the configured threshold values |
| **Console errors** | 4 warnings: "The specified value 'undefined' cannot be parsed" |
| **Screenshot** | N/A |

#### GAP-E2E-M8 — MFA setup page blank — backend endpoint missing

| Field | Value |
|-------|-------|
| **Route(s)** | `/settings/account/mfa` |
| **Observed** | Page title renders "Two-Factor Authentication" but `<main>` element is completely empty. 3 console errors on `/v1/auth/mfa/status` (404). |
| **Expected** | MFA setup page should show TOTP enrollment flow or "MFA not yet available" message |
| **Console errors** | `Failed to load resource: 404` × 3 on `/v1/auth/mfa/status` |
| **Screenshot** | N/A |

#### GAP-E2E-M9 — Settings sub-pages have missing backend endpoints (appeals, blocks, deletion)

| Field | Value |
|-------|-------|
| **Route(s)** | `/settings/account/appeals`, `/settings/blocks`, `/settings/account/delete` |
| **Observed** | Appeals: heading only, 3 errors on `/v1/safety/appeals` (404). Blocks: heading only, 4 errors on `/v1/social/blocks` (500). Account deletion: renders form but 4 errors on `/v1/account/deletion` (404). |
| **Expected** | Pages should either show empty states gracefully or suppress API calls for unimplemented endpoints |
| **Console errors** | Multiple 404/500 errors across these three endpoints |
| **Screenshot** | N/A |

#### GAP-E2E-L3 — Notification history header contradicts content

| Field | Value |
|-------|-------|
| **Route(s)** | `/settings/notifications/history` |
| **Observed** | Page header says "No notifications" while 20 notification items are displayed below it |
| **Expected** | Header should say "Notification History" or reflect the actual count |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-M12 — Invalid entity IDs show empty page instead of "Not Found" state

| Field | Value |
|-------|-------|
| **Route(s)** | `/post/{invalidUUID}`, `/marketplace/listings/{invalidUUID}`, and likely all entity detail routes |
| **Observed** | Navigating to a detail page with a non-existent UUID (zero UUID) or non-UUID string shows an empty `<main>` with 3-4 console errors. No "Post not found" or "Listing not found" message is shown. The 404 page (`/totally-nonexistent-route`) works correctly — this is specifically about parameterized routes with invalid IDs. |
| **Expected** | Entity detail pages should show a user-friendly "Not found" message with navigation back, similar to how compliance portfolio/transcript pages handle it ("Portfolio not found" with a Back link). |
| **Console errors** | `Failed to load resource` × 3-4 on the API endpoints |
| **Screenshot** | N/A |

#### GAP-E2E-M13 — No visual feedback when adding item to cart

| Field | Value |
|-------|-------|
| **Route(s)** | `/marketplace/listings/{id}` |
| **Observed** | Clicking "Add to Cart" successfully adds the item (confirmed by navigating to `/marketplace/cart`) but provides zero visual feedback: no toast notification, no cart badge increment, no button state change (e.g., "Added!" or "In Cart"). |
| **Expected** | Should show a toast/snackbar ("Added to cart!"), change button text to "In Cart" or "Added", and/or show a cart item count badge somewhere in the header/nav. |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-M14 — Checkout endpoint not implemented

| Field | Value |
|-------|-------|
| **Route(s)** | `/marketplace/cart` → Proceed to Checkout |
| **Observed** | Clicking "Proceed to Checkout" fires an error on `/v1/marketplace/cart/checkout`. The cart stays on the same page with no feedback. |
| **Expected** | Should navigate to a checkout page or show an error message if checkout isn't available yet. |
| **Console errors** | `Failed to load resource` on `/v1/marketplace/cart/checkout` |
| **Screenshot** | N/A |

#### GAP-E2E-M15 — Schedule item creation fails — API error

| Field | Value |
|-------|-------|
| **Route(s)** | `/schedule/new` |
| **Observed** | Filling out the schedule item form (title, description, student, start time) and clicking "Save Item" produces an error on `/v1/planning/schedule-items`. The form stays on the page with no user-facing error message. |
| **Expected** | Schedule item should be created and redirect to calendar, or show a validation/API error message. |
| **Console errors** | `Failed to load resource` on `/v1/planning/schedule-items` |
| **Screenshot** | N/A |

#### GAP-E2E-M4 — COPPA verification page accessible unauthenticated with API errors

| Field | Value |
|-------|-------|
| **Route(s)** | `/auth/coppa/verify` |
| **Observed** | Page is accessible without authentication and makes repeated calls to `/v1/billing/micro-charge/status` that return 401, producing 4+ console errors |
| **Expected** | Either redirect to login (if auth required) or suppress API calls until authenticated |
| **Console errors** | `Failed to load resource: 401 (Unauthorized)` × 4+ on `/v1/billing/micro-charge/status` |
| **Screenshot** | `auth-A5-FAIL.png` |

### 5.4  Low

#### GAP-E2E-L1 — Missing favicon.ico

| Field | Value |
|-------|-------|
| **Route(s)** | All routes |
| **Observed** | 404 on `/favicon.ico` on every page load |
| **Expected** | Favicon should be present or the HTML should not reference it |
| **Console errors** | `Failed to load resource: 404 (Not Found)` on `/favicon.ico` |
| **Screenshot** | N/A |

#### GAP-E2E-L2 — Content Viewer shows "Content not found" for seed activity

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/read/{activityDef1ID}` |
| **Observed** | Page renders but shows "Content not found — This content may have been removed or is no longer available." Seed activity definition may not have a readable content type. |
| **Expected** | Seed data should include at least one viewable content item, or the UI should show a more specific error |
| **Console errors** | None |
| **Screenshot** | N/A |

---

## 6  Screenshots

All screenshots saved to `research/screenshots/e2e/` with naming convention:
`{route-group}-{route-number}-{status}.png`
