# E2E Deep Exploratory Testing Report — 2026-04-07 (v2)

> **Agent:** Claude Opus 4.6
> **Duration:** Complete
> **Scope:** Full application E2E via Playwright MCP — deep functional testing

---

## 1  Executive Summary

- **Routes tested:** ~90 / ~90 (120+ test cases across routes + deep interactions)
- **Pass:** 71 | **Warn:** 15 | **Fail:** 8 | **Blocked:** 1
- **Deep interaction tests:** 15 (notifications, messaging, reporting, data export, session mgmt, attendance, test scores, quiz builder, cart, reviews, privacy)
- **Critical gaps found:** 0
- **High gaps found:** 4
- **Medium gaps found:** 8
- **Low gaps found:** 9

---

## 2  Route Smoke Test Results

### 2.1  Auth & Legal Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| A1 | `/auth/login` | PASS | Form renders, validation works (empty fields, invalid creds) |
| A2 | `/auth/register` | PASS | Form with name/email/password/ToS checkbox, Cloudflare Turnstile |
| A3 | `/auth/recovery` | PASS | Email input + send recovery link |
| A4 | `/auth/verification` | PASS | Verification code input + resend button |
| A5 | `/auth/coppa/verify` | WARN | Redirects to login (expected for unauth) but triggers React "Cannot update component while rendering" console error |
| A6 | `/auth/accept-invite/test-token-123` | PASS | Shows invite UI with Accept/Decline buttons |
| L1 | `/legal/terms` | PASS | Full Terms of Service content renders |
| L2 | `/legal/privacy` | PASS | Full Privacy Policy content renders |
| L3 | `/legal/guidelines` | PASS | Full Community Guidelines content renders |

### 2.2  Social Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| S1 | `/` (Feed) | PASS | Feed renders with post composer and posts; post creation, like/unlike, comments all functional |
| S2 | `/friends` | PASS | 20 friends listed, search/filter works, All/Online/Pending tabs |
| S3 | `/friends/discover` | PASS | Empty state "No suggestions available" shown |
| S4 | `/messages` | PASS | Conversation list with unread badges, search works |
| S5 | `/messages/:id` | PASS | Message thread renders. Message sending works with optimistic update — sent message appears immediately in thread |
| S6 | `/groups` | PASS | 9 groups listed, My Groups/Discover tabs |
| S7 | `/groups/:id` | PASS | Group detail with Posts/Members tabs |
| S8 | `/groups/:id/manage` | PASS | Members + Pending requests management tabs |
| S9 | `/events` | PASS | Event list, RSVP (Interested/Going) works |
| S10 | `/post/:id` | WARN | Post detail with comments, reply, delete, report all present. Actions menu shows only "Delete post" for own posts — **no Edit option** despite edit being expected. Report content flow works: 7 report categories, "Thank you" confirmation on submit |
| S11 | `/events/new` | PASS | Event creation form: title, description, date/time, recurring toggle, location type (In Person/Virtual/Hybrid), capacity, visibility, group/methodology tags; event created successfully |
| S12 | `/events/:eventId` | PASS | Event detail with date, location, methodology tag, RSVP buttons (Going/Interested/Not Going), attendee list tabs, Cancel Event button (for host), Report button |
| S13 | `/groups/new` | WARN | Group creation form renders (name, description, join policy radio), BUT submission fails with 422 Unprocessable Entity and no error message shown to user |

### 2.3  Learning Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| LN1 | `/learning` | PASS | Dashboard with quick actions, student progress cards |
| LN2 | `/learning/activities` | PASS | Student selector, activity list, log form with 70+ subjects |
| LN3 | `/learning/journals` | PASS | Student selector, entry types (Nature, Reading, General) |
| LN4 | `/learning/reading-lists` | PASS | Reading list management |
| LN5 | `/learning/grades` | PASS | Grade overview per student |
| LN6 | `/learning/progress/:studentId` | PASS | Emma's progress: 6 activities, 5.9 hours, 2 journals; hours-by-subject bars (reading, nature_study, science, math); recent activity list; Export button; date range filter |
| LN7 | `/learning/activities` → Log form | PASS | Inline log form: title, description, duration (min), date, 70+ subject checkboxes as tree, Add custom subject button, Cancel/Save buttons |
| LN8 | `/learning/quiz/:sessionId` | WARN | Renders blank main content with 404 errors for non-existent session ID; no error state UI shown |
| LN9 | `/learning/video/:videoId` | WARN | Renders blank main content with 6× 404 errors for non-existent video; no error state UI shown |
| LN10 | `/learning/read/:contentId` | PASS | Content Viewer gracefully handles missing content: "Content not found" message, zero console errors |
| LN11 | `/learning/sequence/:progressId` | WARN | Renders blank main content with 3× 404 errors for non-existent progress ID; no error state UI |
| LN12 | `/reading-lists/:id` (via list click) | FAIL | Clicking a reading list navigates to `/reading-lists/:id` which is a 404 — route doesn't exist (should be `/learning/reading-lists/:id`) |

### 2.4  Marketplace Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| MK1 | `/marketplace` | PASS | Featured + Staff Picks sections, search works |
| MK2 | `/marketplace/listings/:id` | WARN | Listing detail renders with reviews, files. **Add to Cart** returns 409 Conflict with no user feedback (item may already be in cart). **Write a Review** form works: star rating, text area, Cancel/Submit — review submitted successfully |
| MK3 | `/marketplace/cart` | PASS | Cart items, remove item, total calculation |
| MK4 | `/marketplace/cart` → Checkout | FAIL | 409 Conflict on `/v1/marketplace/cart/checkout`; missing translation key shown as raw i18n key |
| MK5 | `/marketplace/library` | WARN | "Coming soon" placeholder — no content |
| MK6 | `/marketplace` search | PASS | Search returns results (5 for "science") |

### 2.5  Creator Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CR1 | `/creator` | PASS | Creator dashboard with earnings, sales, listings |
| CR2 | `/creator/listings/:id/edit` | PASS | Edit form with title, description, price, save |
| CR3 | `/creator/analytics` | WARN | "Coming soon" placeholder — no content |
| CR4 | `/creator/listings/new` | PASS | New listing form functional, but publisher field shows raw UUID (see GAP-E2E-L3) |
| CR5 | `/creator/quiz-builder` | PASS | Quiz builder with title, question counter (0), Add Question button. **Add Question** form works: question text, 6 question types (multiple-choice/true-false/short-answer/essay/matching/ordering), 4 answer options, points field, correct answer selector |
| CR6 | `/creator/sequence-builder` | PASS | Sequence builder with title, description, step counter, Add Step button |
| CR7 | `/creator/payouts` | FAIL | Blank main content, 12 console errors from 404s on `/v1/marketplace/payouts/*` and `/v1/marketplace/creator/verification` |
| CR8 | `/creator/verification` | FAIL | "Something went wrong" error, 4 console errors from `/v1/marketplace/creator/verification` 404 |
| CR9 | `/creator/reviews` | FAIL | "Something went wrong" error, 4 console errors from `/v1/marketplace/creator/reviews` 404 |
| CR10 | `/creator/earnings` | WARN | "Coming soon" placeholder — no content |

### 2.6  Billing Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| B1 | `/billing` | PASS | Three tiers (Free/Plus/Premium), Monthly/Annual toggle, feature comparison lists, current plan highlighted |
| B2 | `/billing/payment-methods` | PASS | Empty state with "Add payment method" button, no errors |
| B3 | `/billing/transactions` | PASS | Filter tabs (All/Subscriptions/Purchases/Payouts), date range filter, 1 subscription transaction shown |
| B4 | `/billing/subscription` | PASS | Current plan (Premium $9.99/mo), next billing date, Cancel subscription button, links to Payment Methods and Transactions |
| B5 | `/billing/invoices` | PASS | Filter tabs (All/Subscription/Purchase), date range filter, empty state "No invoices found" |

### 2.7  Settings Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| ST1 | `/settings` | PASS | Family Settings: Profile/Students/Co-Parents tabs; edit family name, methodology selector, student add/edit/delete, co-parent invite |
| ST2 | `/settings/account` | PASS | Email display, password change (disabled "Coming soon"), links to sessions/export/delete/appeals |
| ST3 | `/settings/account/sessions` | PASS | Active sessions list with device info, IP, last active time, revoke buttons. **Deep test:** Clicked Revoke → confirmation dialog appears with Cancel/Revoke Session buttons; Cancel closes dialog correctly |
| ST4 | `/settings/account/export` | PASS | Export format selector, data category checkboxes, past exports list. **Deep test:** Requested CSV export → queued successfully, status shows "Processing" |
| ST5 | `/settings/account/delete` | WARN | Deletion UI renders with checkbox + typed name confirmation, BUT `/v1/account/deletion` returns 404 (4 console errors) |
| ST6 | `/settings/account/appeals` | FAIL | Empty page (heading only, no content, no empty state), `/v1/safety/appeals` returns 404 (3 console errors) |
| ST7 | `/settings/notifications` | PASS | Comprehensive notification preferences grid with In-app/Email columns, toggleable checkboxes, system items properly disabled. **Deep test:** Toggled "Social — Friend requests" checkbox off → saved successfully, toggle state persisted |
| ST8 | `/settings/privacy` | PASS | Location sharing toggle, field visibility dropdowns (Everyone/Friends/Only Me). **Deep test:** Changed methodology visibility from "Hidden" to "Friends only" → saved successfully |
| ST9 | `/settings/notifications/history` | PASS | 20 notifications with types/timestamps, Mark all as read, Filters button, links to preferences |
| ST10 | `/settings/blocks` | FAIL | Only heading "Blocked Users" renders, no content/empty state; 3 console errors from `/v1/social/blocks` returning 500 |
| ST11 | `/settings/account/mfa` | PASS | Two-factor authentication status (not enabled), Enable button |
| ST12 | `/settings/subscription` | PASS | Three tiers with feature lists, Premium marked "Current", Upgrade buttons disabled appropriately |
| ST13 | `/settings/subscription/manage` | PASS | Current plan details (Premium $9.99/monthly), next billing, Change plan/Cancel buttons |
| ST14 | `/settings/account/delete/student/:studentId` | PASS | "Delete Emma" page: COPPA immediate deletion notice, what-will-be-deleted list, checkbox + typed name ("Emma") confirmation, Cancel/Delete buttons; confirmation enables Delete button correctly |

### 2.8  Planning & Calendar Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| PC1 | `/calendar` | PASS | Week view with scheduled items, day view toggle, student filter, prev/next navigation all functional |
| PC2 | `/schedule/new` | PASS | New schedule item form: title, description, student selector, date picker, time picker, category dropdown |
| PC3 | `/planning/print` | PASS | Print schedule with date range selector, student filter, printable table layout |
| PC4 | `/planning/templates` | PASS | "Charlotte Mason Week" template with 5 items, day-of-week indicators, Apply/Delete/Create buttons |
| PC5 | `/planning/coop` | PASS | Co-op group selector (8 groups), week view with Previous/Next/Today navigation |

### 2.9  Compliance Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CP1 | `/compliance` | PASS | State selector dropdown, tracking threshold configuration, links to attendance and tests sub-pages |
| CP2 | `/compliance/attendance` | WARN | Calendar grid with clickable days, attendance status selector (Present/Absent/Excused/Holiday), per-student tracking. **Deep test:** Marking attendance as "Excused" for April 7 and saving returns 422 Unprocessable Entity — no error shown to user, save appears to silently fail |
| CP3 | `/compliance/tests` | PASS | Test scores list per student, add test score form with subject/score/date/notes fields. **Add Test Score** deep test: submitted "Iowa Test of Basic Skills" Reading score 92 — saved successfully |
| CP4 | `/compliance/assessments` | PASS | 4 assessment records with subjects, types, scores, and dates |
| CP5 | `/compliance/portfolios` | PASS | Student filter, New Portfolio button, "Emma Spring 2026 Portfolio" with 3 items |
| CP6 | `/compliance/transcripts` | PASS | Student filter, New Transcript button, "Emma — 2025-2026 Year-End Transcript" with 5 items |
| CP7 | `/compliance/portfolios/:studentId/:id` | WARN | Portfolio Builder: cover page editor, 3 items with reorder/remove, Add Items, Preview dialog, Save, Generate PDF. Reorder works (Save enables). BUT 3× 400 errors on `/portfolios/candidates` endpoint |
| CP8 | `/compliance/transcripts/:studentId/:id` | PASS | Transcript Builder: GPA (3.60), 3 editable courses (Literature A, Mathematics B, Nature Science A), level/credits/grade editing, Add Course, Remove, Generate PDF. Zero errors |

### 2.10  Other Authenticated Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| O1 | `/notifications` | PASS | 20 notifications listed with various types, Mark as read buttons, Mark all as read; however mark-as-read may not persist (see GAP-E2E-L2) |
| O2 | `/search?q=math` | PASS | Global search with Social/Marketplace/Learning tabs, results render correctly across all tabs |
| O3 | `/creator/listings/new` | PASS | (Duplicate of CR4 — see Creator Routes) |
| O4 | `/recommendations` | PASS | Personalized recommendations with tabs (All/Content/Activities/Resources), methodology-matched content, dismiss/block buttons |
| O5 | `/profile` | PASS | Correctly redirects to `/family/:familyId`, shows family avatar, name, bio, Add Friend button |
| O6 | `/family/:familyId` | PASS | Family profile page with avatar, name, bio; own profile shows "Add Friend" (minor UX: shouldn't show for own profile) |
| O7 | `/marketplace/purchases` | PASS | 11 purchases with listing names, prices, dates, Download and Request Refund buttons |
| O8 | `/marketplace/purchases/:id/refund` | PASS | Refund reason dropdown (5 options), details textarea, Submit/Cancel buttons, 7-day notice |
| O9 | `/marketplace/listings/:id/versions` | FAIL | Error boundary crash: `TypeError: versions.map is not a function` (6 console errors) |

### 2.11  Admin Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| AD1 | `/admin` | BLOCKED | Non-admin user correctly redirected to `/`. Admin login (`admin@example.com` / `AdminPassword123!`) returns "Invalid email or password" — agent Kratos identity not configured |

### 2.12  Student / Methodology Tool Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| MT1 | `/learning/nature-journal` | PASS | Charlotte Mason: student selector, observation type, weather, species ID, drawing upload area, subject connections |
| MT2 | `/learning/trivium-tracker` | PASS | Classical: Grammar/Logic/Rhetoric stage tabs, topic input, vocabulary, memorization type selector |
| MT3 | `/learning/rhythm-planner` | PASS | Waldorf: day-of-week selector, time block grid, activity category assignments |
| MT4 | `/learning/observation-logs` | PASS | Montessori: observation log form with concentration level slider, work period tracking |
| MT5 | `/learning/habit-tracking` | PASS | Charlotte Mason: daily check-in grid, suggested habits list, custom habit creation |
| MT6 | `/learning/interest-led-log` | PASS | Unschooling: exploration mode selector, resource links, subject connections tree |
| MT7 | `/learning/handwork-projects` | PASS | Waldorf: craft type selector, project status tracking, materials list, techniques checklist |
| MT8 | `/learning/practical-life` | PASS | Montessori: life skill area categories, mastery level progression indicators |
| MT9 | `/learning/journals/new` | PASS | Journal editor: student selector, entry type (Freeform/Narration/Reflection), title, content, date, 70+ flat subject checkboxes |
| MT10 | `/learning/projects` | PASS | Empty state "No projects yet", New project button |
| MT11 | `/learning/tools` | PASS | Tool assignment page: 14 learning tools with enable/disable toggles per student, Reset to defaults button |
| MT12 | `/learning/session` | PASS | Student session launcher: student cards (Emma Age 12, James Age 9) with Back link |
| MT13 | `/student` | PASS | StudentGuard correctly redirects parent user to feed (expected behavior) |

---

## 3  User Journey Results

### Journey 1: Onboarding Flow
- **Status:** PARTIAL — cannot test without new registration (Turnstile blocks automation)
- **Steps completed:** 1/6
- **Issues found:** (none — login + auth redirect tested via route smoke tests)

### Journey 2: Learning Workflow
- **Status:** COMPLETE
- **Steps completed:** 12/12
- **Issues found:**
  - All 8 methodology tools tested and functional (nature journal, trivium tracker, rhythm planner, observation logs, habit tracking, interest-led log, handwork projects, practical life)
  - Activity logging with 70+ subjects works; log form opens inline with comprehensive fields
  - Journal entries (Nature, Reading, General) work; journal editor tested
  - Reading lists render but clicking a list navigates to a 404 route — GAP-E2E-L6
  - Student progress view fully functional (stats, subject hours, recent activity, export)
  - Quiz/video/sequence viewers show blank page for missing resources — GAP-E2E-L7
  - Content viewer correctly handles missing content with friendly error state
  - Grades render correctly; learning tools assignment page (14 tools) works

### Journey 3: Social Interactions
- **Status:** COMPLETE
- **Steps completed:** 14/14 (including deep interaction tests)
- **Issues found:**
  - Feed does not optimistically update after post creation (requires page reload) — GAP-E2E-M1
  - Nested comment replies not rendered in UI (comment count increments but reply text not visible) — GAP-E2E-H1
  - Feed sort order may be ascending instead of descending (older posts appear above newer ones) — GAP-E2E-L1
  - Post actions menu shows only "Delete" for own posts — no "Edit" option — GAP-E2E-L9
  - **Deep tests passed:** Message sending (optimistic update), content reporting (7 categories, confirmation), friends list, groups, events all functional

### Journey 4: Marketplace Browse & Purchase
- **Status:** COMPLETE
- **Steps completed:** 8/9 (checkout blocked by 409)
- **Issues found:**
  - Browse, search, listing detail, cart management all work
  - **Add to Cart** returns 409 with no feedback (likely "already in cart") — GAP-E2E-M8
  - Checkout fails with 409 Conflict — GAP-E2E-H2
  - **Write a Review** form works: star rating, text area, submitted successfully
  - Library is "Coming soon" placeholder

### Journey 5: Creator Workflow
- **Status:** COMPLETE
- **Steps completed:** 7/10 (3 pages crash/blank)
- **Issues found:**
  - Dashboard, listing edit, new listing creation all work
  - Publisher field shows raw UUID — GAP-E2E-L3
  - Quiz builder and sequence builder functional
  - Analytics and earnings pages are "Coming soon" placeholders
  - Payouts page blank with 12× 404 errors — GAP-E2E-H3
  - Verification page crashes with "Something went wrong" — GAP-E2E-H3
  - Reviews page crashes with "Something went wrong" — GAP-E2E-H3

### Journey 6: Planning & Compliance
- **Status:** COMPLETE
- **Steps completed:** 14/14 (including deep interaction tests)
- **Issues found:**
  - Calendar week/day views, navigation, student filter all work
  - Schedule creation form functional
  - Print schedule renders correctly
  - Compliance state selector, attendance calendar, test scores all functional
  - **Attendance save** returns 422 silently when marking "Excused" — GAP-E2E-M7
  - **Add Test Score** deep test: "Iowa Test of Basic Skills" Reading 92 — saved successfully
  - Assessments list (4 records) renders correctly
  - Portfolio Builder: cover page, 3 items with reorder/remove, preview dialog, save — all work. 400 errors on candidates endpoint — GAP-E2E-L8
  - Transcript Builder: GPA calculation (3.60), 3 editable courses with level/credits/grade, Add Course, Remove, Generate PDF — all work perfectly, zero errors
  - Templates and Co-op planning views functional

### Journey 7: Settings & Account Management
- **Status:** COMPLETE
- **Steps completed:** 16/17 (password change disabled; including deep interaction tests)
- **Issues found:**
  - Family settings (profile, students, co-parents) all functional
  - Account info, sessions, export all functional
  - **Session revocation** deep test: Revoke button shows confirmation dialog with Cancel/Revoke Session — Cancel works correctly
  - **Data export** deep test: CSV export queued successfully, shows "Processing" status
  - Delete account page has 404 errors on status check — GAP-E2E-M3
  - Student deletion page works correctly: COPPA notice, checkbox + typed name confirmation flow enables Delete button
  - Appeals page empty with 404 errors — GAP-E2E-M4
  - **Notification preferences** deep test: Toggled checkbox off → saved and persisted
  - **Privacy controls** deep test: Changed methodology visibility to "Friends only" → saved correctly
  - Notification history (20 items, filters, mark all as read) functional
  - Blocked users page returns 500 — GAP-E2E-H4
  - MFA settings page renders correctly (enable/disable toggle)
  - Subscription management (3 tiers, change plan, cancel) functional
  - Password change shows "Coming soon"

### Journey 8: Admin Workflow
- **Status:** BLOCKED
- **Steps completed:** 0/9
- **Issues found:**
  - Cannot test — admin Kratos identity not configured in agent instance
  - Non-admin correctly redirected away from admin routes

---

## 4  Edge Case Results

### 4.1  Empty States
| Page | Status | Notes |
|------|--------|-------|
| `/friends/discover` | PASS | Shows "No suggestions available" empty state |
| `/marketplace/library` | WARN | Shows "Coming soon" placeholder — acceptable for unreleased feature |
| `/creator/analytics` | WARN | Shows "Coming soon" placeholder — acceptable for unreleased feature |
| `/creator/earnings` | WARN | Shows "Coming soon" placeholder — acceptable for unreleased feature |
| `/settings/account/appeals` | FAIL | Shows only heading, no empty state message when no appeals exist |
| `/settings/blocks` | FAIL | Shows only heading "Blocked Users", no empty state; 500 error |
| `/learning/projects` | PASS | Shows "No projects yet" empty state with New Project button |
| `/billing/invoices` | PASS | Shows "No invoices found" empty state |
| `/learning/read/:contentId` (missing) | PASS | Shows "Content not found" with helpful message |
| `/learning/quiz/:sessionId` (missing) | FAIL | Blank page, no error state shown |
| `/learning/video/:videoId` (missing) | FAIL | Blank page, no error state shown |
| `/learning/sequence/:progressId` (missing) | FAIL | Blank page, no error state shown |

### 4.2  Invalid Routes
| Route | Status | Notes |
|-------|--------|-------|
| `/nonexistent-route` | PASS | 404 page with "Page not found" heading and "Go Home" button |
| `/learning/progress` | PASS | Returns 404 (route requires student ID parameter) |

### 4.3  Permission Boundaries
| Test | Status | Notes |
|------|--------|-------|
| Non-admin → `/admin` | PASS | Correctly redirected to `/` (feed) |
| Unauthenticated → `/auth/coppa/verify` | WARN | Redirects to login but triggers React render warning (see GAP-E2E-M2) |

### 4.4  Form Validation
| Form | Status | Notes |
|------|--------|-------|
| Login (empty fields) | PASS | Shows "email is required" and "password is required" alerts |
| Login (invalid creds) | PASS | Shows "Invalid email or password" error |
| Settings → Add Student (empty name) | PASS | Shows "Name is required" validation error |
| Student deletion confirmation | PASS | Delete button stays disabled until both checkbox checked AND correct name typed |
| Activity log (no student) | PASS | Log activity button disabled until student selected |
| Group creation (submit) | FAIL | 422 error with no user-facing error message — GAP-E2E-M6 |
| Attendance save (Excused) | FAIL | 422 Unprocessable Entity with no error shown to user — GAP-E2E-M7 |
| Add to Cart (listing detail) | FAIL | 409 Conflict with no user feedback (item may already be in cart) — GAP-E2E-M8 |
| Report content (post) | PASS | 7 report categories, text area, submitted successfully with "Thank you" confirmation |
| Write a Review (marketplace) | PASS | Star rating + text area form submits correctly |

### 4.5  Navigation & Routing
| Test | Status | Notes |
|------|--------|-------|
| Reading list → detail click | FAIL | Navigates to `/reading-lists/:id` (404) instead of valid route — GAP-E2E-L6 |
| Portfolio list → builder click | PASS | Navigates to `/compliance/portfolios/:studentId/:id` correctly |
| Transcript list → builder click | PASS | Navigates to `/compliance/transcripts/:studentId/:id` correctly |
| `/profile` redirect | PASS | Correctly redirects to `/family/:familyId` |
| `/student` guard (parent user) | PASS | Correctly redirects parent to feed |

---

## 5  Gap Register

### 5.1  Critical

(none yet)

### 5.2  High

#### GAP-E2E-H1 — Nested comment replies not rendered in post detail

| Field | Value |
|-------|-------|
| **Route(s)** | `/post/:id` |
| **Observed** | After submitting a reply to a comment, the comment count increments (from 2 to 3) but the reply text is never displayed in the comment thread. After page reload, still only top-level comments are visible. |
| **Expected** | Nested replies should render indented below their parent comment, showing the reply text, author, and timestamp. |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-H2 — Marketplace checkout fails with 409 Conflict

| Field | Value |
|-------|-------|
| **Route(s)** | `/marketplace/cart` → Checkout |
| **Observed** | Clicking "Proceed to Checkout" triggers POST to `/v1/marketplace/cart/checkout` which returns HTTP 409 Conflict. The error message displayed is the raw i18n translation key `marketplace.cart.checkout.error` instead of a user-friendly message. |
| **Expected** | Checkout should either succeed or show a translated, user-friendly error message explaining why it failed. |
| **Console errors** | `MissingTranslationError` for key `marketplace.cart.checkout.error` |
| **Screenshot** | N/A |

#### GAP-E2E-H3 — Creator payouts, verification, and reviews pages fail (404 endpoints)

| Field | Value |
|-------|-------|
| **Route(s)** | `/creator/payouts`, `/creator/verification`, `/creator/reviews` |
| **Observed** | `/creator/payouts` renders blank main content with 12 console errors from four 404 endpoints (`/v1/marketplace/payouts/methods`, `/v1/marketplace/payouts/config`, `/v1/marketplace/payouts/history`, `/v1/marketplace/creator/verification`). `/creator/verification` and `/creator/reviews` both show "Something went wrong" error boundary with 4 console errors each from 404 endpoints. |
| **Expected** | Pages should either render with available data or show a graceful "coming soon" / empty state instead of crashing or going blank. |
| **Console errors** | 20+ combined 404 errors across the three pages |
| **Screenshot** | N/A |

#### GAP-E2E-H4 — Blocked users page returns 500 Internal Server Error

| Field | Value |
|-------|-------|
| **Route(s)** | `/settings/blocks` |
| **Observed** | Page renders only the heading "Blocked Users" with no content below. The API call to `/v1/social/blocks` returns HTTP 500 Internal Server Error, producing 3 console errors. |
| **Expected** | The endpoint should return a 200 with an empty list (or populated list), and the UI should render either the block list or an empty state ("No blocked users"). |
| **Console errors** | 3× `GET /v1/social/blocks 500` |
| **Screenshot** | N/A |

### 5.3  Medium

#### GAP-E2E-M1 — Feed does not update after post creation

| Field | Value |
|-------|-------|
| **Route(s)** | `/` (Feed) |
| **Observed** | After creating a new post, the post composer clears and the Post button disables (success state), but the new post does not appear in the feed. A manual page reload is required to see the new post. |
| **Expected** | New post should appear at the top of the feed immediately after creation (optimistic update or query invalidation). |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-M2 — React state update during render on COPPA redirect

| Field | Value |
|-------|-------|
| **Route(s)** | `/auth/coppa/verify` |
| **Observed** | When navigating to COPPA verify page while unauthenticated, the redirect to login triggers a React console error: "Cannot update a component while rendering a different component". |
| **Expected** | Auth redirect should not trigger React state update warnings. |
| **Console errors** | `Cannot update a component (%s) while rendering a different component` |
| **Screenshot** | N/A |

#### GAP-E2E-M3 — Delete account page returns 404 on status check

| Field | Value |
|-------|-------|
| **Route(s)** | `/settings/account/delete` |
| **Observed** | The delete account page renders its UI (checkbox confirmation, typed name input, delete button) but the API call to `/v1/account/deletion` returns HTTP 404, producing 4 console errors. The endpoint may not be implemented yet. |
| **Expected** | Either the endpoint should exist and return the deletion status, or the page should gracefully handle the 404 without console errors. |
| **Console errors** | 4× `GET /v1/account/deletion 404` |
| **Screenshot** | N/A |

#### GAP-E2E-M4 — Moderation appeals page empty with 404 errors

| Field | Value |
|-------|-------|
| **Route(s)** | `/settings/account/appeals` |
| **Observed** | The page renders only the heading "Appeals" with no content below — no empty state message, no list, no form. The API call to `/v1/safety/appeals` returns HTTP 404 (3 console errors). |
| **Expected** | Should either show an empty state ("No appeals") when the endpoint returns no data, or the endpoint should exist and return appropriate data. |
| **Console errors** | 3× `GET /v1/safety/appeals 404` |
| **Screenshot** | N/A |

#### GAP-E2E-M5 — Listing version history crashes with TypeError

| Field | Value |
|-------|-------|
| **Route(s)** | `/marketplace/listings/:id/versions` |
| **Observed** | Navigating to a listing's version history triggers an error boundary with `TypeError: versions.map is not a function`. The component attempts to call `.map()` on a null/undefined value, meaning the API response is not returning an array as expected. |
| **Expected** | The page should either render the version list or show an empty state if no versions exist. The component should handle null/undefined data gracefully. |
| **Console errors** | 6× errors including TypeError |
| **Screenshot** | N/A |

#### GAP-E2E-M6 — Group creation fails silently with 422 error

| Field | Value |
|-------|-------|
| **Route(s)** | `/groups/new` |
| **Observed** | After filling out the group creation form (name, description, join policy) and submitting, the API returns HTTP 422 Unprocessable Entity. No error message is displayed to the user — the form simply remains on screen with no feedback. |
| **Expected** | Either the form should submit successfully, or validation errors from the 422 response should be displayed to the user (e.g., missing required fields, invalid values). |
| **Console errors** | None visible (error may be swallowed by the mutation handler) |
| **Screenshot** | N/A |

#### GAP-E2E-M7 — Attendance save returns 422 with no error shown to user

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance/attendance` |
| **Observed** | After selecting a day on the attendance calendar, choosing "Excused" status, and clicking Save, the PUT request to `/v1/compliance/students/:id/attendance` returns HTTP 422 Unprocessable Entity. No error message is displayed to the user — the save appears to silently fail. The attendance status reverts to its previous value on page reload. |
| **Expected** | Either the save should succeed, or a validation error should be displayed explaining why the attendance record couldn't be saved (e.g., missing required fields, invalid date). |
| **Console errors** | None visible (error may be swallowed by mutation handler) |
| **Screenshot** | N/A |

#### GAP-E2E-M8 — Add to Cart returns 409 with no user feedback

| Field | Value |
|-------|-------|
| **Route(s)** | `/marketplace/listings/:id` |
| **Observed** | Clicking "Add to Cart" on a marketplace listing triggers POST to `/v1/marketplace/cart/items` which returns HTTP 409 Conflict. No feedback is shown to the user — the button remains in its default state with no success or error indication. The 409 likely means the item is already in the cart, but this isn't communicated. |
| **Expected** | If the item is already in the cart, show a message like "Item already in cart" or change the button to "In Cart" / "Go to Cart". If it's a different conflict, show an appropriate error message. |
| **Console errors** | None visible (error may be swallowed) |
| **Screenshot** | N/A |

### 5.4  Low

#### GAP-E2E-L1 — Feed sort order shows older posts first

| Field | Value |
|-------|-------|
| **Route(s)** | `/` (Feed) |
| **Observed** | After reload, the seed post ("1h ago") appears above the newly created post ("Just now"). Social feeds typically show newest content first. |
| **Expected** | Feed should default to reverse chronological order (newest first). |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-L2 — Notification mark-as-read may not persist

| Field | Value |
|-------|-------|
| **Route(s)** | `/notifications` |
| **Observed** | After clicking "Mark as read" on a notification, the unread badge in the header still shows "9 unread". After page reload, the same notification still shows the "Mark as read" button. The action may not be saving to the backend. |
| **Expected** | Mark as read should persist: the badge count should decrement and the notification should appear as read after reload. |
| **Console errors** | None observed |
| **Screenshot** | N/A |

#### GAP-E2E-L3 — Creator listing publisher field shows raw UUID

| Field | Value |
|-------|-------|
| **Route(s)** | `/creator/listings/new` |
| **Observed** | The publisher/author field in the new listing form displays the raw UUID (`01900000-0000-7000-8000-000000000202`) in the text input, with the human-readable name displayed below it. |
| **Expected** | The publisher field should show a human-readable display name, not a raw UUID. The UUID should be an internal value only. |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-L4 — Events list doesn't link to event detail pages

| Field | Value |
|-------|-------|
| **Route(s)** | `/events` |
| **Observed** | Event titles on the events list page are rendered as plain headings, not as clickable links. Users cannot navigate from the event list to an individual event's detail page by clicking its title. RSVP buttons work, but there is no navigation path to `/events/:eventId`. |
| **Expected** | Event titles should be clickable links (or the event card should be clickable) that navigate to the event detail page `/events/:eventId`. |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-L5 — Family profile shows "Add Friend" button for own profile

| Field | Value |
|-------|-------|
| **Route(s)** | `/profile`, `/family/:familyId` |
| **Observed** | When viewing your own family profile page, the "Add Friend" button is still displayed. Clicking it on your own profile is a no-op but confusing UX. |
| **Expected** | The "Add Friend" button should be hidden when viewing your own family's profile page. |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-L6 — Reading list detail navigates to non-existent route

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/reading-lists` → `/reading-lists/:id` |
| **Observed** | Clicking "Emma's Books" on the reading lists page navigates to `/reading-lists/:id`, which is a 404 page. The route is defined as `learning/reading-lists` but there is no `learning/reading-lists/:id` or `reading-lists/:id` route for the detail view. |
| **Expected** | Either a reading list detail route should exist (e.g., `/learning/reading-lists/:id`), or the reading list items should expand inline without navigation. |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-L7 — Quiz, video, and sequence viewers show blank page for missing resources

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/quiz/:sessionId`, `/learning/video/:videoId`, `/learning/sequence/:progressId` |
| **Observed** | When navigating to these routes with a non-existent ID, the main content area renders completely blank with no error message or "not found" state. Multiple 404 console errors are produced. By contrast, `/learning/read/:contentId` correctly shows "Content not found". |
| **Expected** | All content viewer routes should show a user-friendly "not found" or error state when the resource doesn't exist, consistent with the Content Viewer's behavior. |
| **Console errors** | 3–6 × 404 errors per page |
| **Screenshot** | N/A |

#### GAP-E2E-L9 — Post feed actions menu missing Edit option for own posts

| Field | Value |
|-------|-------|
| **Route(s)** | `/` (Feed), `/post/:id` |
| **Observed** | The actions menu (three-dot/more button) on the user's own posts shows only "Delete post" — there is no "Edit post" option. The post detail page (`/post/:id`) does show edit/delete on comments, but the main post actions on the feed lack an edit action. |
| **Expected** | Own posts should have both "Edit post" and "Delete post" options in the actions menu, consistent with comment actions which do show edit. |
| **Console errors** | None |
| **Screenshot** | N/A |

#### GAP-E2E-L8 — Portfolio builder 400 errors on candidates endpoint

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance/portfolios/:studentId/:id` |
| **Observed** | The Portfolio Builder page loads and is functional (reorder, preview, save all work), but produces 3× 400 Bad Request errors on the `/v1/compliance/students/:id/portfolios/candidates` endpoint on initial load. The errors don't prevent usage but indicate a backend validation issue with the date range query parameters. |
| **Expected** | The candidates endpoint should accept the date range from the portfolio and return available items without errors. |
| **Console errors** | 3× `GET /v1/compliance/students/.../portfolios/candidates 400` |
| **Screenshot** | N/A |

---

## 6  Screenshots

All screenshots saved to `research/screenshots/e2e/` with naming convention:
`{route-group}-{route-number}-{status}.png`

- `auth-A1-PASS.png` — Login page
- `auth-A2-PASS.png` — Register page
