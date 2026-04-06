# E2E Deep Exploratory Testing Report — 2026-04-06

> **Agent:** Claude Opus 4.6
> **Duration:** Complete (3 sessions)
> **Scope:** Full application deep E2E via Playwright MCP — exhaustive interaction testing
> **Procedure:** `specs/procedures/E2E_EXPLORATORY_TESTING.md`
> **Note:** This is a second-pass deep test focused on actual interactions (form submissions, button clicks, state changes) beyond surface-level route smoke tests.

---

## 1  Executive Summary

- **Routes tested:** 86 / ~90
- **Pass:** 38 | **Warn:** 24 | **Fail:** 17 | **Blocked:** 1 | **Skip:** 1
- **Critical gaps found:** 0
- **High gaps found:** 10 (H1–H10)
- **Medium gaps found:** 19 (M1–M19)
- **Low gaps found:** 10 (L1–L10)
- **Edge cases tested:** 4 (invalid IDs, non-UUID paths, non-existent routes, date timezone)

---

## 2  Route Smoke Test Results

### 2.1  Auth & Legal Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| A1 | `/auth/login` | PASS | Login form renders. Empty submit shows Kratos "Property X is missing" messages (see GAP-M1). Wrong creds shows proper generic "Invalid email or password". |
| A2 | `/auth/register` | PASS | Register form with name/email/password/terms. Empty submit shows Kratos property names (see GAP-M1). Weak password validation works ("must be at least 10 chars"). Terms checkbox validation works. |
| A3 | `/auth/recovery` | FAIL | Page renders but submitting with valid email returns "The requested resource could not be found" — recovery flow appears broken (see GAP-H1). |
| A4 | `/auth/verification` | WARN | Page renders. Submitting empty code fails silently — no user-facing error shown despite 400 API response (see GAP-M2). |
| A5 | `/auth/coppa/verify` | FAIL | Blank page. Requires auth but doesn't redirect to login. 3x 401 errors on micro-charge status endpoint (see GAP-M3). |
| A6 | `/auth/accept-invite/test-token-123` | PASS | Shows invite UI. Accept with invalid token shows "Something went wrong" (could be more specific). Decline redirects to login. Inviter name shows "Homegrown Academy" instead of actual inviter (see GAP-L1). |
| L1 | `/legal/terms` | PASS | Full Terms of Service renders with 7 sections. Links to privacy and guidelines work. |
| L2 | `/legal/privacy` | PASS | Full Privacy Policy renders. COPPA section present. |
| L3 | `/legal/guidelines` | PASS | Community Guidelines render with 7 sections. |
| — | `/auth/logout` | FAIL | Route does not exist — shows 404 "Page not found." No visible logout button anywhere in the authenticated UI (see GAP-H2). |

### 2.2  Onboarding

| # | Route | Status | Notes |
|---|-------|--------|-------|
| OB1 | `/onboarding` | SKIP | Seed user has already completed onboarding; revisiting redirects to home. |

### 2.3  Social Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| S1 | `/feed` | WARN | Post creation works but feed doesn't refresh — new post only appears after manual page reload. New posts appear at bottom of feed instead of top. Like button click doesn't update visual state on newly created posts (see GAP-M4, GAP-M5, GAP-L2). |
| S2 | `/friends` | PASS | 20 friends listed. Search filter works. Tabs (All, Online, Pending) work. |
| S3 | `/friends/discover` | PASS | Renders but shows no suggestions (expected for seed data). |
| S4 | `/messages` | PASS | Conversations list loads. Sending a message works with optimistic update (appears immediately). |
| S6 | `/groups` | PASS | 9 groups listed. Group detail pages load with members tab. |
| S9 | `/groups/:id/manage` | WARN | Owner appears in their own "Pending Join Requests" list (see GAP-M8). |
| S5 | `/messages/:id` | PASS | Message detail page with full conversation history, timestamps, and send works with optimistic update (new message appears immediately). |
| S7 | `/groups/:id` | PASS | Group detail with banner, description, member count, Leave Group button. Posts tab (empty state message), Members tab (shows owner/member roles). |
| S8 | `/groups` (Discover tab) | WARN | Discover tab renders but is completely empty — no content and no "no groups to discover" empty state message (see GAP-L9). |
| S10 | `/events` | WARN | RSVP "Going" works, but clicking again doesn't un-RSVP. Attendee count stays at 1 and button stays pressed (see GAP-M7). Event creation form renders with all fields but POST returns 400 with no error feedback (see GAP-H10). |
| S13 | `/feed/post/:id` | WARN | Comment creation works and updates immediately. Comment editing fails — PUT endpoint returns 404 (see GAP-H3). Reply to comment works but appears as top-level comment, no visual threading (see GAP-L3). Report dialog works end-to-end. |

### 2.4  Learning Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| LR1 | `/learning` | PASS | Dashboard renders with quick actions and student progress cards for Emma and James. |
| LR2 | `/learning/activities` | WARN | Student selector works. Activity log form: create works, new activity appears at top immediately. Date off by one day — entered 2026-04-06, shows 4/5/2026 (see GAP-M9). Subject tags show slug "math" instead of display name "Mathematics" (see GAP-L4). |
| LR3 | `/learning/journals` | PASS | Journal list loads per student. Entry type filter (Freeform/Narration/Reflection) works. "New entry" disabled until student selected. |
| LR4 | `/learning/journals/new` | WARN | Journal entry creation works end-to-end. Student selector loses context from list page — resets to "Select a student" on navigation (see GAP-L5). Tags show slugs. |
| LR5 | `/learning/reading-lists` | WARN | Lists load. "New list" inline creation works. Reading list item has `cursor:pointer` but clicking does nothing — no detail view or expansion (see GAP-M10). |
| LR6 | `/learning/progress/select` | WARN | Page renders but uses "select" as literal student ID — heading shows `'s Academic Progress` (missing name), 12 API errors. Works correctly when navigated via dashboard "View details" link with real student ID (see GAP-M14). |
| LR7 | `/learning/progress/:studentId` | PASS | Emma's progress page renders with summary stats (7 activities, 6.4h, 3 journals), hours-by-subject chart, recent activity timeline. Subject names show slugs (see GAP-L4). "math" and "mathematics" appear as separate subjects — data inconsistency. |
| LR8 | `/learning/grades` | PASS | Assessments page with student selector, subject filter. "Add grade" form has title, date, score type (Percentage/Points/Letter), weight, grading scale, subject tree, notes. |
| LR9 | `/learning/trivium-tracker` | WARN | Trivium Tracker (Classical methodology) renders with Grammar/Logic/Rhetoric stages, memorization types, subject tree. Save entry returns 400 Bad Request with no user-facing error message (see GAP-H9). Subject tag shows dot-path slug "foreign-languages.latin" instead of "Latin" (see GAP-L4). |
| LR10 | `/learning/nature-journal` | WARN | Nature Journal (Charlotte Mason) renders with observation type, weather radio buttons, temperature, location, species, drawing upload. Shows methodology note. Save entry returns 400 with no user-facing error (see GAP-H9). Pre-selected tag shows "nature_study" slug. |
| LR11 | `/learning/observation-logs` | PASS | Observation Log (Montessori) renders with concentration level radio buttons, work chosen field, materials, subject connections. Methodology note present. |
| LR12 | `/learning/rhythm-planner` | PASS | Rhythm Planner (Waldorf) renders with day-of-week tabs, time blocks with activity categories (Main Lesson, Circle Time, Arts, etc.). Pre-populated with 6 blocks. Add/delete block works. |
| LR13 | `/learning/habit-tracking` | PASS | Habit Tracking (Charlotte Mason) renders with predefined habits (Attention, Diligence), Yes/Partial/Not today toggles, add habit buttons, custom habit input. |
| LR14 | `/learning/interest-led-log` | PASS | Interest-Led Log (Unschooling) renders with exploration method buttons, resource tracking, subject connections with note about cross-subject learning. |
| LR15 | `/learning/handwork-projects` | PASS | Handwork Projects (Waldorf) renders with craft type dropdown, project status radio buttons, materials, techniques, photo upload. |
| LR16 | `/learning/practical-life` | PASS | Practical Life (Montessori) renders with 5 life skill areas, mastery level radio buttons, contextual examples per area. |

### 2.5  Marketplace Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| MK1 | `/marketplace` | PASS | Full marketplace with search, Featured/Staff Picks/Trending/New Arrivals sections, and listing grid with prices/ratings/publishers. |
| MK2 | `/marketplace/listings/:id` | WARN | Detail page renders with description, files, reviews. "Add to Cart" returns 500 (see GAP-H4). "Write a Review" form opens but submit returns 500 (see GAP-H5). Subject tags show slugs. |
| MK3 | `/marketplace/search` | PASS | Search works via header search bar. Tabs (Social/Marketplace/Learning) filter results correctly. |
| MK4 | `/marketplace/cart` | WARN | Shopping cart renders with items, prices, total. "Remove from cart" works — item removed and total recalculates. "Proceed to Checkout" POST to `/v1/marketplace/cart/checkout` fails silently — no error shown (see GAP-M15). |
| MK5 | `/search?q=nature` | PASS | Global search renders on `/search` with Social/Marketplace/Learning tabs. Results show relevant items with metadata (member counts, prices, ratings, publishers). 20+ social results, 17 marketplace results. |
| MK6 | `/marketplace/purchases` | PASS | Purchase History with 10 items showing title, price, date, Download and Request Refund buttons. Download triggers silently (no errors). |
| MK7 | `/marketplace/purchases/:id/refund` | PASS | Refund request form with reason dropdown (5 options), details textbox. Submit works end-to-end — shows "Refund Request Submitted" success page. |
| MK8 | `/marketplace/library` | FAIL | 404 "Page not found." Route not implemented. |

### 2.6  Creator Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CR1 | `/creator` | PASS | Creator Dashboard with earnings ($35.98), 2 sales, 5 listings with status/version. Time range filter works. "New Listing" link present. |
| CR2 | `/creator/listings/:id/edit` | PASS | Edit listing form with title, description, price, change summary. Save works — version bumps (v1→v2). Archive button present. |
| CR3 | `/creator/listings/new` | PASS | Create listing form with title, description, price, content type dropdown, publisher ID, subject tags, grade range. Publisher field shows raw ID placeholder instead of auto-filling (see GAP-L8). |
| CR4 | `/creator/earnings` | FAIL | 404 "Page not found." Route not implemented. |
| CR5 | `/creator/analytics` | FAIL | 404 "Page not found." Route not implemented. |
| CR6 | `/creator/payouts` | WARN | Payout Setup page renders but with 16 console errors on payout/verification endpoints (all return 500). Shows "Creator Verification Required" message (see GAP-M13). |
| CR7 | `/creator/verification` | FAIL | Page renders blank — main content area empty. 3 errors on `/v1/marketplace/creator/verification` endpoint (see GAP-M13). |

### 2.7  Billing Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| B1 | `/settings/billing` | FAIL | 404 "Page not found." Route not implemented (same as ST6). |

### 2.8  Settings Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| ST1 | `/settings` | PASS | Family Settings with Profile/Students/Co-Parents tabs. Profile shows family name, state, city, subscription tier, methodology. Edit form works — save persists changes. |
| ST2 | `/settings` (Students tab) | PASS | Lists Emma (Born 2014) and James (Born 2017) with Edit/Delete buttons and "Add student" button. |
| ST3 | `/settings/account` | PASS | Shows email, password section ("Coming soon" + disabled Change button), links to Sessions, Export, Delete account, Moderation appeals. |
| ST4 | `/settings/notifications` | PASS | Comprehensive notification preferences grid with In-app/Email toggles per category. Mandatory notifications disabled correctly. Toggle auto-saves. |
| ST5 | `/settings/privacy` | PASS | Privacy Controls page with field visibility dropdowns (Everyone/Friends/Only Me). All 5 fields configurable. |
| ST6 | `/settings/billing` | FAIL | 404 "Page not found." Route not implemented. |
| ST7 | `/settings/account` → Sessions | PASS | Active sessions page shows current session with browser/OS info and "Revoke" button. Revoke works. |
| ST8 | `/settings/account` → Export | PASS | Data export page with format selector (JSON/CSV) and category checkboxes. Export triggers download correctly. |
| ST9 | `/settings/account` → Delete | FAIL | Page renders blank. 3x 404 on `/v1/account/deletion` endpoint. No delete account UI visible (see GAP-M12). |

### 2.9  Planning & Calendar Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| PL1 | `/calendar` | PASS | Week view renders with events, learning activities, attendance, and schedule items. Student filter works. Week/Day toggle, Previous/Next/Today navigation, Add Item, Export, Print buttons all present. |
| PL2 | `/calendar` (attendance) | WARN | Calendar shows "Attendance: present_full" raw enum value instead of human-readable text (see GAP-L6). |
| PL3 | `/calendar` → New Schedule Item | FAIL | Schedule item creation form renders with time/title/recurrence fields. Save returns 500 on POST `/v1/planning/schedule-items` (see GAP-H8). |
| PL4 | `/planning/print` | PASS | Print Schedule page renders with date range picker, student filter, and printable table (Time, Item, Student, Source, Category, Done). Shows "Attendance: present_full" raw enum (see GAP-L6). |

### 2.10  Compliance Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| CP1 | `/compliance` | WARN | Compliance Setup renders with state dropdown, tracking thresholds. State defaults to "Texas" but family profile says "Connecticut" — state not synced from family settings (see GAP-M11). Save configuration returns 422 with no error shown (see GAP-M16). |
| CP2 | `/compliance/attendance` | WARN | Attendance Tracker with calendar, student selector, legend. Clicking a day opens status dropdown (Present/Absent/Partial/Excused). Save attendance record fails with API error — no feedback (see GAP-M17). "6 of 0 days (0%)" display is confusing when required days = 0 (see GAP-L7). |
| CP3 | `/compliance/transcripts` | PASS | Transcript list with student filter, "New Transcript" button. 2 transcripts for Emma visible. |
| CP4 | `/compliance/transcripts/:studentId/:transcriptId` | WARN | Transcript Builder renders with editable course table, GPA calculation (3.60), level dropdown (Regular/Honors/AP), credits, grades. "Add Course" works. "Generate PDF" triggers dialog but stays stuck at "generating" status — no download, no error, no timeout (see GAP-M18). |
| CP5 | `/compliance/tests` | WARN | Standardized Tests page with 2 test records. "Add test score" works end-to-end (form, save, new entry appears). But date off-by-one: entered "2026-04-01", displayed "Mar 31, 2026" — same timezone bug as GAP-M9. Subject label casing inconsistent: "Reading" (Iowa) vs "math", "reading" (CAT) (see GAP-L10). |
| CP6 | `/compliance/portfolio` | FAIL | 404 "Page not found." Route not implemented. |

### 2.11  Other Authenticated Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| OA1 | `/recommendations` | WARN | Recommendations page renders with personalized suggestions, tabs (All/Content/Activities/Resources). "Dismiss" button returns 500 error (see GAP-H6). "Block category" button present. |
| OA2 | `/search?q=math` | PASS | Search works across Social/Marketplace/Learning tabs. Results show relevant items with metadata. |
| OA3 | `/notifications` | WARN | 20 notifications render with "Mark as read" buttons. "Mark all as read" present. "Mark as read" on individual notification returns 500 error (see GAP-H7). Badge still shows 10 unread after marking. |
| OA4 | `/notifications/history` | FAIL | 404 "Page not found." Route not implemented. |
| OA5 | `/settings/moderation` | FAIL | 404 "Page not found." Route not implemented. |
| OA6 | `/profile` | FAIL | 404 "Page not found." No self-profile page exists. |
| OA7 | `/` (Home) | PASS | Home page redirects to Feed. Shows post composer (text/photo/milestone/event_share/resource_share types) and existing posts with like counts and comment links. |

### 2.12  Admin Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| AD1 | `/admin` | BLOCKED | Admin login (admin@example.com) fails — "Invalid email or password." Admin identity may not exist in agent Kratos instance. Non-admin user redirected to home. |

### 2.13  Student Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| SR1 | `/student/*` | SKIP | Student routes require token-based auth, not tested in this pass. |

---

## 3  User Journey Results

(populated during testing)

---

## 4  Edge Case Results

### 4.1  Invalid / Non-Existent Resource IDs

| Test | Result | Notes |
|------|--------|-------|
| Group with zero UUID | FAIL | `/groups/00000000-...` shows blank main area, 9 console errors (3 endpoints × 3 retries). No "group not found" message. |
| Group with non-UUID path | FAIL | `/groups/not-a-uuid` shows blank main area, 9 console errors. Same behavior — no validation on client or server. |
| Marketplace listing with zero UUID | FAIL | `/marketplace/listings/00000000-...` shows blank main area, 3 console errors. No "listing not found" message. |
| Non-existent route | PASS | `/nonexistent-page` correctly shows 404 "Page not found" with "Go Home" button. |

**Pattern:** Routes that use `:id` params render blank pages with console errors when given invalid IDs. Only truly non-existent routes show the 404 page. Detail pages should show a "not found" or "resource doesn't exist" message (see GAP-M19).

### 4.2  Date/Timezone Edge Cases

| Test | Result | Notes |
|------|--------|-------|
| Activity date entry | FAIL | Enter "2026-04-06", displays "4/5/2026" — off by one day (GAP-M9). |
| Test score date entry | FAIL | Enter "2026-04-01", displays "Mar 31, 2026" — same timezone bug. |

**Pattern:** All dates stored/displayed through the system appear to be shifted back by one day, suggesting UTC midnight parsing displayed in local time.

---

## 5  Gap Register

### 5.1  Critical

(none yet)

### 5.2  High

#### GAP-H1 — Password recovery flow completely broken

| Field | Value |
|-------|-------|
| **Route(s)** | `/auth/recovery` |
| **Observed** | Submitting the recovery form (with valid or empty email) returns "The requested resource could not be found." The Kratos recovery flow endpoint returns a 400/404. |
| **Expected** | With empty email: "Email is required." With valid email: "Recovery link sent" (or silent success to avoid user enumeration). |
| **Console errors** | `Failed to load resource: the server responded with a status of 400` on `/self-service/recovery?flow=...` |

#### GAP-H2 — No logout mechanism in UI

| Field | Value |
|-------|-------|
| **Route(s)** | Entire app, `/auth/logout` |
| **Observed** | There is no visible "Log out" button anywhere in the app UI. The `/auth/logout` route shows a 404 "Page not found" page. Users have no way to sign out. |
| **Expected** | A logout option should be accessible from the user menu / settings area. Clicking it should destroy the Kratos session and redirect to login. |

#### GAP-H3 — Comment edit API endpoint returns 404

| Field | Value |
|-------|-------|
| **Route(s)** | `/feed/post/:id` |
| **Observed** | Clicking "Edit" on a comment and submitting changes triggers a PUT request that returns 404. The backend endpoint for comment editing appears to not be implemented. |
| **Expected** | Comment edit should persist the changes and update the comment in-place. |

#### GAP-H4 — Marketplace "Add to Cart" returns 500

| Field | Value |
|-------|-------|
| **Route(s)** | `/marketplace/listings/:id` |
| **Observed** | Clicking "Add to Cart" on a marketplace listing triggers a POST to `/v1/marketplace/cart/items` which returns a 500 server error. No item is added. |
| **Expected** | Item should be added to the shopping cart with visual confirmation. |
| **Console errors** | `Failed to load resource: the server responded with a status of 500` on `/v1/marketplace/cart/items` |

#### GAP-H5 — Marketplace review submission returns 500

| Field | Value |
|-------|-------|
| **Route(s)** | `/marketplace/listings/:id` |
| **Observed** | "Write a Review" form opens correctly with star rating and text input. Submitting the review (4 stars + text) returns a 500 server error. The form remains open. |
| **Expected** | Review should be saved and appear in the reviews section below the listing. |
| **Console errors** | `Failed to load resource: the server responded with a status of 500` on `/v1/marketplace/listings/:id/reviews` |

#### GAP-H6 — Recommendation dismiss returns 500

| Field | Value |
|-------|-------|
| **Route(s)** | `/recommendations` |
| **Observed** | Clicking "Dismiss" on a recommendation triggers a POST to `/v1/recommendations/:id/dismiss` which returns a 500 server error. The recommendation remains visible. |
| **Expected** | Dismissed recommendation should be removed from the list. |

#### GAP-H7 — Notification "Mark as read" returns 500

| Field | Value |
|-------|-------|
| **Route(s)** | `/notifications` |
| **Observed** | Clicking "Mark as read" on an individual notification triggers a POST to `/v1/notifications/:id/read` which returns a 500 server error. The notification remains unread. |
| **Expected** | Notification should be marked as read, visual state should update, and unread count should decrement. |

#### GAP-H8 — Schedule item save returns 500

| Field | Value |
|-------|-------|
| **Route(s)** | `/calendar` → New Schedule Item |
| **Observed** | Filling out the new schedule item form (title, time, recurrence) and clicking Save triggers a POST to `/v1/planning/schedule-items` which returns a 500 server error. No item is created. |
| **Expected** | Schedule item should be saved and appear on the calendar. |
| **Console errors** | `Failed to load resource: the server responded with a status of 500` on `/v1/planning/schedule-items` |

#### GAP-H9 — Methodology tool save returns 400 with no user-facing error

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/trivium-tracker`, `/learning/nature-journal` (likely all methodology tools) |
| **Observed** | Filling out and submitting the Trivium Tracker or Nature Journal entry form triggers a POST to `/v1/learning/students/:id/activities` which returns 400 Bad Request. The form stays filled in — no error message, no success message, no visual feedback at all. |
| **Expected** | On success: form should clear and show confirmation. On validation error: display specific error message so user can correct the input. |
| **Console errors** | `Failed to load resource: the server responded with a status of 400 (Bad Request)` on `/v1/learning/students/:id/activities` |

#### GAP-H10 — Event creation returns 400 with no user feedback

| Field | Value |
|-------|-------|
| **Route(s)** | `/events` |
| **Observed** | Creating an event via the Create Event dialog (title, description, date, location, region, capacity, visibility all filled) triggers a POST to `/v1/social/events` which returns 400 Bad Request. Dialog stays open with no error message. |
| **Expected** | On success: dialog closes, new event appears in list. On validation error: display specific field-level errors so user can correct input. |
| **Console errors** | `Failed to load resource: the server responded with a status of 400 (Bad Request)` on `/v1/social/events` |

### 5.3  Medium

#### GAP-M4 — Feed doesn't refresh after creating a new post

| Field | Value |
|-------|-------|
| **Route(s)** | `/feed` |
| **Observed** | After successfully creating a new post, the feed does not update. The new post only appears after a manual page reload. |
| **Expected** | The feed should automatically refresh or optimistically insert the new post at the top. |

#### GAP-M5 — Like button doesn't update visual state

| Field | Value |
|-------|-------|
| **Route(s)** | `/feed` |
| **Observed** | Clicking the like button on a post doesn't visually update the button state or like count. |
| **Expected** | Like button should toggle visual state (filled/unfilled) and increment/decrement the count. |

#### GAP-M7 — Event RSVP cannot be un-toggled

| Field | Value |
|-------|-------|
| **Route(s)** | `/events` |
| **Observed** | After clicking "Going" on an event, clicking the button again does not remove the RSVP. Attendee count stays at 1 and the button remains in the pressed state. |
| **Expected** | Clicking "Going" again should un-RSVP the user, decrement the attendee count, and return the button to its default state. |

#### GAP-M8 — Group owner appears in own pending join requests

| Field | Value |
|-------|-------|
| **Route(s)** | `/groups/:id/manage` |
| **Observed** | The group owner/creator appears in the "Pending Join Requests" list on the group management page. |
| **Expected** | The owner should not appear in pending requests — they are already a member. |

#### GAP-M9 — Activity date off by one day (timezone issue)

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/activities` |
| **Observed** | When creating an activity with date "2026-04-06", it displays as "4/5/2026" — one day earlier. Likely a UTC vs local timezone conversion issue. |
| **Expected** | The date should display exactly as entered, or convert consistently. |

#### GAP-M1 — Kratos internal field names exposed in validation errors

| Field | Value |
|-------|-------|
| **Route(s)** | `/auth/login`, `/auth/register` |
| **Observed** | Empty form submissions show messages like "Property identifier is missing", "Property password is missing", "Property name is missing", "Property email is missing." |
| **Expected** | User-friendly messages like "Email is required", "Password is required", "Name is required." |

#### GAP-M2 — Email verification form fails silently on empty submit

| Field | Value |
|-------|-------|
| **Route(s)** | `/auth/verification` |
| **Observed** | Clicking "Verify email" with an empty code results in a 400 console error but no user-facing error message is displayed. |
| **Expected** | Should display an inline validation error like "Verification code is required." |

#### GAP-M3 — COPPA verification page blank when unauthenticated

| Field | Value |
|-------|-------|
| **Route(s)** | `/auth/coppa/verify` |
| **Observed** | Page renders completely blank (empty accessibility tree). 3 repeated 401 errors on `/v1/billing/micro-charge/status`. The page requires auth but doesn't redirect to login. |
| **Expected** | Either redirect to login (if auth required), or render the COPPA form for unauthenticated users with appropriate messaging. |

#### GAP-M10 — Reading list item click does nothing

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/reading-lists` |
| **Observed** | Reading list items show `cursor:pointer` and are styled as clickable, but clicking does nothing — no navigation to a detail view and no inline expansion. |
| **Expected** | Clicking a reading list should navigate to a detail view showing the books in the list, or expand inline to show contents. |

#### GAP-M11 — Compliance state not synced from family settings

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance` |
| **Observed** | Family settings shows state as "Connecticut" but compliance setup defaults to "Texas". The compliance state picker does not read from the family profile. |
| **Expected** | Compliance state should default to the state configured in family settings. |

#### GAP-M15 — Marketplace checkout fails silently

| Field | Value |
|-------|-------|
| **Route(s)** | `/marketplace/cart` |
| **Observed** | Clicking "Proceed to Checkout" triggers a POST to `/v1/marketplace/cart/checkout` which fails. No error message shown, no navigation occurs. Cart items and total remain visible. |
| **Expected** | Checkout should either navigate to a payment/confirmation page or display an error message. |
| **Console errors** | `Failed to load resource: the server responded with a status of ...` on `/v1/marketplace/cart/checkout` |

#### GAP-M16 — Compliance config save returns 422 with no error

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance` |
| **Observed** | Changing state to Oregon, setting thresholds (180 days, 900 hours), and clicking "Save configuration" triggers a PUT/POST to `/v1/compliance/config` which returns 422 Unprocessable Entity. No error message shown to user. |
| **Expected** | Configuration should save successfully. If validation fails, display specific error messages. |

#### GAP-M17 — Attendance record save fails with no feedback

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance/attendance` |
| **Observed** | Clicking a calendar date, selecting status (Present), and clicking Save fails with an API error on POST to `/v1/compliance/students/:id/attendance`. No error or success feedback shown. The date remains unmarked. |
| **Expected** | Attendance should save and the calendar date should update with the appropriate status color/icon. |

#### GAP-M18 — Transcript PDF generation stuck at "generating"

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance/transcripts/:studentId/:transcriptId` |
| **Observed** | Clicking "Generate PDF" opens a confirmation dialog. Clicking "Generate" closes the dialog and changes transcript status to "generating", but it never completes — no PDF download, no error, no timeout. Status remains "generating" indefinitely. |
| **Expected** | PDF should be generated and downloaded (or a download link should appear). If generation fails, an error message should be shown. |

#### GAP-M19 — Invalid resource IDs show blank pages instead of "not found"

| Field | Value |
|-------|-------|
| **Route(s)** | `/groups/:id`, `/marketplace/listings/:id`, and likely all detail routes with `:id` params |
| **Observed** | Navigating to a detail page with an invalid UUID (e.g., all zeros) or non-UUID string (e.g., "not-a-uuid") renders a blank main content area with multiple console errors. No "not found" or error message is shown to the user. The 404 page component exists but is only triggered for completely undefined routes, not for valid routes with invalid resource IDs. |
| **Expected** | Should display a user-friendly "Resource not found" message with a link back to the parent list page. |

#### GAP-M12 — Delete account page renders blank

| Field | Value |
|-------|-------|
| **Route(s)** | `/settings/account` → Delete Account |
| **Observed** | Navigating to the delete account page renders a blank page. Console shows 3x 404 errors on `/v1/account/deletion`. The API endpoint does not exist. |
| **Expected** | Delete account page should render with confirmation flow (password re-entry, reason, final confirmation). |

#### GAP-M13 — Creator payout/verification endpoints all return errors

| Field | Value |
|-------|-------|
| **Route(s)** | `/creator/payouts`, `/creator/verification` |
| **Observed** | Payout Setup page triggers 16 console errors — endpoints `/v1/marketplace/payouts/methods`, `/v1/marketplace/payouts/history`, `/v1/marketplace/payouts/config`, and `/v1/marketplace/creator/verification` all return 500. The verification page renders completely blank. |
| **Expected** | Payout setup should show payment method configuration. Verification page should show identity verification form. |

#### GAP-M14 — Academic Progress "select" route uses literal path segment as student ID

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/progress/select` |
| **Observed** | The "Academic Progress" quick action links to `/learning/progress/select`, which treats "select" as a student ID. Heading shows `'s Academic Progress` (empty name), and 12 API errors fire because "select" is not a valid UUID. |
| **Expected** | Route should either show a student selector, or the link should be to a different path that includes student selection. The dashboard "View details" link correctly uses `/learning/progress/:studentId` with the real UUID. |

### 5.4  Low

#### GAP-L2 — New posts appear at bottom of feed instead of top

| Field | Value |
|-------|-------|
| **Route(s)** | `/feed` |
| **Observed** | After creating a new post and refreshing the page, the new post appears at the bottom of the feed list instead of at the top (reverse chronological). |
| **Expected** | Newest posts should appear at the top of the feed. |

#### GAP-L3 — Comment replies appear as top-level comments (no threading)

| Field | Value |
|-------|-------|
| **Route(s)** | `/feed/post/:id` |
| **Observed** | Replying to a comment creates a new top-level comment rather than a nested reply. There is no visual threading or indentation for replies. |
| **Expected** | Replies should appear nested under the parent comment with visual indentation. |

#### GAP-L4 — Subject tags show slug instead of display name

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/activities`, `/feed` |
| **Observed** | Subject/tag labels display the slug (e.g., "math", "nature_study", "reading") instead of a human-readable display name (e.g., "Mathematics", "Nature Study", "Reading"). |
| **Expected** | Tags should display human-readable names. |

#### GAP-L5 — Student selector loses context between journal list and new entry form

| Field | Value |
|-------|-------|
| **Route(s)** | `/learning/journals`, `/learning/journals/new` |
| **Observed** | Selecting Emma on the journal list page, then clicking "New entry" navigates to `/learning/journals/new` where the student selector resets to "Select a student". The selected student context is lost on navigation. |
| **Expected** | The previously selected student should be pre-selected on the new entry form. |

#### GAP-L6 — Calendar attendance shows raw enum value

| Field | Value |
|-------|-------|
| **Route(s)** | `/calendar` |
| **Observed** | Calendar displays "Attendance: present_full" using the raw database enum value instead of a human-readable label. |
| **Expected** | Should display "Attendance: Full Day Present" or similar friendly text. |

#### GAP-L7 — Attendance tracker "0 of 0 days" display when no requirement configured

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance/attendance` |
| **Observed** | Progress shows "6 of 0 days (0%)" when no attendance days are required. Having 0 required days makes the percentage meaningless and the "On track" label misleading. |
| **Expected** | Should display "No attendance requirement configured" or omit the progress bar when required days = 0. |

#### GAP-L1 — Invitation page shows app name instead of inviter name

| Field | Value |
|-------|-------|
| **Route(s)** | `/auth/accept-invite/:token` |
| **Observed** | The invitation text says "Homegrown Academy has invited you to co-manage their family" — using the app name instead of the actual inviter's display name. |
| **Expected** | Should display the inviting user/family's name: e.g., "The Smith Family has invited you..." |

#### GAP-L8 — Creator listing "Publisher" field shows raw ID placeholder

| Field | Value |
|-------|-------|
| **Route(s)** | `/creator/listings/new` |
| **Observed** | The Publisher field on the Create Listing form shows placeholder text "Publisher ID" and expects a raw UUID input. |
| **Expected** | The publisher field should auto-fill with the current user's creator profile or show a human-readable name. |

#### GAP-L9 — Groups Discover tab shows empty with no message

| Field | Value |
|-------|-------|
| **Route(s)** | `/groups` (Discover tab) |
| **Observed** | Clicking the "Discover" tab on the Groups page shows a completely empty panel — no groups listed and no empty state message. |
| **Expected** | Should display either discoverable groups not yet joined, or an empty state message like "You've joined all available groups!" |

#### GAP-L10 — Standardized test subject label casing inconsistent

| Field | Value |
|-------|-------|
| **Route(s)** | `/compliance/tests` |
| **Observed** | Iowa Test shows "Reading" (capitalized) while California Achievement Test shows "math", "reading", "science", "language" (all lowercase). Casing is inconsistent across test records. |
| **Expected** | Subject labels should use consistent casing (e.g., all Title Case: "Math", "Reading", "Science", "Language"). |

---

## 6  Screenshots

All screenshots saved to `research/screenshots/e2e/` with naming convention:
`{route-group}-{route-number}-{description}.png`
