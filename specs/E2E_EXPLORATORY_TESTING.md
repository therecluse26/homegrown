# E2E Exploratory Testing — Agent Prompt

> **Purpose:** A self-contained prompt that an AI agent can execute to perform exhaustive
> exploratory end-to-end testing of the Homegrown Academy application using Playwright MCP
> tools. The goal is to systematically discover functional gaps and bugs that code review
> alone cannot reveal.
>
> **Output:** A structured gap report following the format in `specs/gaps_04-05_26.md`.

---

## 1  Environment Setup

### 1.1  Infrastructure

The application runs on the **dev infrastructure**:

| Component | URL / Port | Notes |
|-----------|-----------|-------|
| Frontend (Vite) | `http://localhost:5673` | SPA, proxies API to backend |
| Backend (Go/Echo) | `http://localhost:3500` | REST API at `/v1/*` |
| Dev Kratos (public) | `http://localhost:4933` | Identity provider |
| Dev Kratos (admin) | `http://localhost:4934` | Admin API |
| PostgreSQL | `localhost:5932` | Database `homegrown` |

### 1.2  Startup Procedure

```
# 1. Ensure seed data is fresh (idempotent, safe to rerun)
make seed DB=homegrown

# 2. Start both backend and frontend
make dev
```

Wait for both servers to be ready before proceeding. The frontend dev server prints
`Local: http://localhost:5673/` when ready.

### 1.3  Browser Configuration

Use Playwright MCP in **Chromium** mode. Set viewport to `1280×800` for desktop testing:

```
mcp__pw__browser_resize → width: 1280, height: 800
```

---

## 2  Authentication Procedures

### 2.1  Parent Login (seed@example.com)

This is the primary test account with full seed data across all domains.

```
1. mcp__pw__browser_navigate → http://localhost:5673/auth/login
2. mcp__pw__browser_snapshot → verify login form is visible
3. mcp__pw__browser_fill_form → fields:
   - { name: "Email", type: "textbox", ref: <email-field-ref>, value: "seed@example.com" }
   - { name: "Password", type: "textbox", ref: <password-field-ref>, value: "SeedPassword123!" }
4. mcp__pw__browser_click → Submit button
5. mcp__pw__browser_wait_for → text: "Feed" or "/" route content
6. mcp__pw__browser_snapshot → verify authenticated state (nav bar, user menu)
```

**Onboarding gate:** After login, the `OnboardingGuard` checks onboarding status. If the
seed account has `status = completed` or `status = skipped`, you proceed to the app. If
redirected to `/onboarding`, click the **"Skip all"** button to bypass, then continue.

### 2.2  Admin Login (admin@example.com)

Used for admin-only routes (`/admin/*`).

```
1. mcp__pw__browser_navigate → http://localhost:5673/auth/login
2. mcp__pw__browser_fill_form → fields:
   - { name: "Email", type: "textbox", ref: <email-field-ref>, value: "admin@example.com" }
   - { name: "Password", type: "textbox", ref: <password-field-ref>, value: "AdminPassword123!" }
3. mcp__pw__browser_click → Submit button
4. mcp__pw__browser_wait_for → text: "Admin" or redirect to admin dashboard
5. Handle onboarding gate if needed (same as §2.1)
```

### 2.3  Logout

```
1. mcp__pw__browser_click → User menu / avatar in top nav
2. mcp__pw__browser_click → "Log out" or "Sign out" option
3. mcp__pw__browser_wait_for → redirect to /auth/login
```

---

## 3  Seed Data Reference

All seed entity UUIDs for parameterized routes. Source: `cmd/seed/main.go`.

### 3.1  Core Entities

| Entity | Variable | UUID |
|--------|----------|------|
| Seed family | `seedFamilyID` | `01900000-0000-7000-8000-000000000001` |
| Friend family | `friendFamilyID` | `01900000-0000-7000-8000-000000000002` |
| Platform family | `platformFamilyID` | `01900000-0000-7000-8000-000000000003` |
| Seed parent | `seedParentID` | `01900000-0000-7000-8000-000000000011` |
| Friend parent | `friendParentID` | `01900000-0000-7000-8000-000000000012` |
| Admin parent | `adminParentID` | `01900000-0000-7000-8000-000000000013` |
| Emma (student) | `emmaStudentID` | `01900000-0000-7000-8000-000000000021` |
| James (student) | `jamesStudentID` | `01900000-0000-7000-8000-000000000022` |

### 3.2  Social

| Entity | Variable | UUID |
|--------|----------|------|
| Group | `groupID` | `01900000-0000-7000-8000-000000000051` |
| Post 1 | `post1ID` | `01900000-0000-7000-8000-000000000061` |
| Post 2 | `post2ID` | `01900000-0000-7000-8000-000000000062` |
| Post 3 | `post3ID` | `01900000-0000-7000-8000-000000000063` |
| Conversation | `conversationID` | `01900000-0000-7000-8000-000000000091` |
| Event | `eventID` | `01900000-0000-7000-8000-000000000111` |

### 3.3  Marketplace

| Entity | Variable | UUID |
|--------|----------|------|
| Listing 1 | `listing1ID` | `01900000-0000-7000-8000-000000000211` |
| Listing 2 | `listing2ID` | `01900000-0000-7000-8000-000000000212` |
| Listing 3 | `listing3ID` | `01900000-0000-7000-8000-000000000213` |
| Listing 4 | `listing4ID` | `01900000-0000-7000-8000-000000000214` |
| Listing 5 | `listing5ID` | `01900000-0000-7000-8000-000000000215` |
| Purchase 1 | `purchase1ID` | `01900000-0000-7000-8000-000000000221` |
| Purchase 2 | `purchase2ID` | `01900000-0000-7000-8000-000000000222` |

### 3.4  Learning

| Entity | Variable | UUID |
|--------|----------|------|
| Emma (student) | `emmaStudentID` | `01900000-0000-7000-8000-000000000021` |
| James (student) | `jamesStudentID` | `01900000-0000-7000-8000-000000000022` |
| Video def | `videoDef1ID` | `01900000-0000-7000-8000-000000000306` |
| Quiz def | `quizDef1ID` | `01900000-0000-7000-8000-000000000381` |
| Sequence def | `sequenceDef1ID` | `01900000-0000-7000-8000-000000000385` |
| Journal 1 | `journal1ID` | `01900000-0000-7000-8000-000000000351` |
| Student session | `studentSession1ID` | `01900000-0000-7000-8000-000000000033` |
| Student assign 1 | `studentAssign1ID` | `01900000-0000-7000-8000-000000000394` |

### 3.5  Compliance

| Entity | Variable | UUID |
|--------|----------|------|
| Schedule item 1 | `complySchedule1ID` | `01900000-0000-7000-8000-000000000800` |
| Portfolio 1 | `complyPortfolio1ID` | `01900000-0000-7000-8000-000000000850` |
| Transcript 1 | `complyTranscript1ID` | `01900000-0000-7000-8000-000000000840` |

### 3.6  Planning

| Entity | Variable | UUID |
|--------|----------|------|
| Schedule item 1 | `schedItem1ID` | `01900000-0000-7000-8000-000000000801` |
| Schedule item 2 | `schedItem2ID` | `01900000-0000-7000-8000-000000000802` |

---

## 4  Testing Strategy

Testing proceeds in three phases: breadth-first smoke testing, depth-focused user journeys,
then creative edge case exploration.

### Phase 1: Route Smoke Test

**Goal:** Visit every route in the application. For each route:

1. `mcp__pw__browser_navigate` → the URL
2. `mcp__pw__browser_snapshot` → capture accessibility tree
3. `mcp__pw__browser_console_messages` → check for errors (level: "error")
4. `mcp__pw__browser_take_screenshot` → visual record

**Classification per route:**
- **PASS** — Page renders meaningful content, no console errors
- **WARN** — Page renders but has console warnings or missing data
- **FAIL** — Page crashes, shows error boundary, or has JS errors
- **BLOCKED** — Cannot reach (auth issue, redirect loop, etc.)

#### 4.1.1  Auth Routes (unauthenticated)

Test these **before** logging in:

| # | Route | URL |
|---|-------|-----|
| A1 | Login | `/auth/login` |
| A2 | Register | `/auth/register` |
| A3 | Account Recovery | `/auth/recovery` |
| A4 | Email Verification | `/auth/verification` |
| A5 | COPPA Micro-Charge | `/auth/coppa/verify` |
| A6 | Accept Invitation | `/auth/accept-invite/test-token-123` |

#### 4.1.2  Legal Routes (public, no auth)

| # | Route | URL |
|---|-------|-----|
| L1 | Terms of Service | `/legal/terms` |
| L2 | Privacy Policy | `/legal/privacy` |
| L3 | Community Guidelines | `/legal/guidelines` |

#### 4.1.3  Onboarding (after login, before skip)

| # | Route | URL |
|---|-------|-----|
| O1 | Onboarding Wizard | `/onboarding` |

After testing onboarding, **skip it** (click "Skip all") to unlock the main app routes.

#### 4.1.4  Home & Social Routes

Log in as `seed@example.com` first.

| # | Route | URL | Notes |
|---|-------|-----|-------|
| S1 | Feed (home) | `/` | Index route |
| S2 | Friends List | `/friends` | |
| S3 | Friend Discovery | `/friends/discover` | |
| S4 | Direct Messages | `/messages` | |
| S5 | Conversation | `/messages/01900000-0000-7000-8000-000000000091` | `conversationID` |
| S6 | Groups List | `/groups` | |
| S7 | Create Group | `/groups/new` | |
| S8 | Group Detail | `/groups/01900000-0000-7000-8000-000000000051` | `groupID` |
| S9 | Group Management | `/groups/01900000-0000-7000-8000-000000000051/manage` | `groupID` |
| S10 | Events List | `/events` | |
| S11 | Create Event | `/events/new` | |
| S12 | Event Detail | `/events/01900000-0000-7000-8000-000000000111` | `eventID` |
| S13 | Post Detail | `/post/01900000-0000-7000-8000-000000000061` | `post1ID` |
| S14 | Family Profile | `/family/01900000-0000-7000-8000-000000000001` | `seedFamilyID` |

#### 4.1.5  Learning Routes

| # | Route | URL | Notes |
|---|-------|-----|-------|
| LR1 | Learning Dashboard | `/learning` | |
| LR2 | Activity Log | `/learning/activities` | |
| LR3 | Journal List | `/learning/journals` | |
| LR4 | New Journal | `/learning/journals/new` | |
| LR5 | Reading Lists | `/learning/reading-lists` | |
| LR6 | Progress (Emma) | `/learning/progress/01900000-0000-7000-8000-000000000021` | `emmaStudentID` |
| LR7 | Progress (James) | `/learning/progress/01900000-0000-7000-8000-000000000022` | `jamesStudentID` |
| LR8 | Tests & Grades | `/learning/grades` | |
| LR9 | Quiz Player | `/learning/quiz/01900000-0000-7000-8000-000000000381` | `quizDef1ID` — may need active session |
| LR10 | Video Player | `/learning/video/01900000-0000-7000-8000-000000000306` | `videoDef1ID` |
| LR11 | Content Viewer | `/learning/read/01900000-0000-7000-8000-000000000301` | `activityDef1ID` |
| LR12 | Sequence View | `/learning/sequence/01900000-0000-7000-8000-000000000385` | `sequenceDef1ID` |
| LR13 | Session Activity Log | `/learning/session-log/01900000-0000-7000-8000-000000000033` | `studentSession1ID` |
| LR14 | Session Launcher | `/learning/session` | |
| LR15 | Projects | `/learning/projects` | |
| LR16 | Tool Assignment | `/learning/tools` | |
| LR17 | Nature Journal | `/learning/nature-journal` | Charlotte Mason tool |
| LR18 | Trivium Tracker | `/learning/trivium-tracker` | Classical tool |
| LR19 | Rhythm Planner | `/learning/rhythm-planner` | Waldorf tool |
| LR20 | Observation Logs | `/learning/observation-logs` | Montessori tool |
| LR21 | Habit Tracking | `/learning/habit-tracking` | Charlotte Mason tool |
| LR22 | Interest-Led Log | `/learning/interest-led-log` | Unschooling tool |
| LR23 | Handwork Projects | `/learning/handwork-projects` | Waldorf tool |
| LR24 | Practical Life | `/learning/practical-life` | Montessori tool |
| LR25 | Parent Quiz Scoring | `/learning/quiz/01900000-0000-7000-8000-000000000381/score` | `quizDef1ID` |

#### 4.1.6  Marketplace Routes

| # | Route | URL | Notes |
|---|-------|-----|-------|
| MK1 | Browse | `/marketplace` | |
| MK2 | Listing Detail | `/marketplace/listings/01900000-0000-7000-8000-000000000211` | `listing1ID` |
| MK3 | Cart | `/marketplace/cart` | |
| MK4 | Purchase History | `/marketplace/purchases` | |
| MK5 | Refund Request | `/marketplace/purchases/01900000-0000-7000-8000-000000000221/refund` | `purchase1ID` |
| MK6 | Listing Versions | `/marketplace/listings/01900000-0000-7000-8000-000000000211/versions` | `listing1ID` |

#### 4.1.7  Creator Routes

| # | Route | URL | Notes |
|---|-------|-----|-------|
| CR1 | Creator Dashboard | `/creator` | |
| CR2 | Create Listing | `/creator/listings/new` | |
| CR3 | Edit Listing | `/creator/listings/01900000-0000-7000-8000-000000000211/edit` | `listing1ID` |
| CR4 | Quiz Builder (new) | `/creator/quiz-builder` | |
| CR5 | Quiz Builder (edit) | `/creator/quiz-builder/01900000-0000-7000-8000-000000000381` | `quizDef1ID` |
| CR6 | Sequence Builder (new) | `/creator/sequence-builder` | |
| CR7 | Sequence Builder (edit) | `/creator/sequence-builder/01900000-0000-7000-8000-000000000385` | `sequenceDef1ID` |
| CR8 | Payout Setup | `/creator/payouts` | |
| CR9 | Creator Verification | `/creator/verification` | |
| CR10 | Creator Reviews | `/creator/reviews` | |

#### 4.1.8  Billing Routes

| # | Route | URL |
|---|-------|-----|
| B1 | Pricing Page | `/billing` |
| B2 | Payment Methods | `/billing/payment-methods` |
| B3 | Transaction History | `/billing/transactions` |
| B4 | Subscription Mgmt | `/billing/subscription` |
| B5 | Invoice History | `/billing/invoices` |

#### 4.1.9  Settings Routes

| # | Route | URL | Notes |
|---|-------|-----|-------|
| ST1 | Family Settings | `/settings` | 3-tab layout |
| ST2 | Notification Prefs | `/settings/notifications` | |
| ST3 | Notification History | `/settings/notifications/history` | |
| ST4 | Subscription Upgrade | `/settings/subscription` | |
| ST5 | Account Settings | `/settings/account` | |
| ST6 | Session Management | `/settings/account/sessions` | |
| ST7 | Data Export | `/settings/account/export` | |
| ST8 | Account Deletion | `/settings/account/delete` | |
| ST9 | Student Deletion (Emma) | `/settings/account/delete/student/01900000-0000-7000-8000-000000000021` | `emmaStudentID` |
| ST10 | Moderation Appeals | `/settings/account/appeals` | |
| ST11 | Block Management | `/settings/blocks` | |
| ST12 | Privacy Controls | `/settings/privacy` | |
| ST13 | MFA Setup | `/settings/account/mfa` | |
| ST14 | Subscription Manager | `/settings/subscription/manage` | |

#### 4.1.10  Planning & Calendar Routes

| # | Route | URL | Notes |
|---|-------|-----|-------|
| PL1 | Calendar View | `/calendar` | |
| PL2 | Calendar Day | `/calendar/day/2026-04-05` | Today's date |
| PL3 | Calendar Week | `/calendar/week/2026-04-05` | Today's date |
| PL4 | New Schedule Item | `/schedule/new` | |
| PL5 | Edit Schedule Item | `/schedule/01900000-0000-7000-8000-000000000801/edit` | `schedItem1ID` |
| PL6 | Schedule Templates | `/planning/templates` | |
| PL7 | Print Schedule | `/planning/print` | |
| PL8 | Co-op Coordination | `/planning/coop` | |

#### 4.1.11  Compliance Routes

| # | Route | URL | Notes |
|---|-------|-----|-------|
| CP1 | Compliance Setup | `/compliance` | Index route |
| CP2 | Attendance Tracker | `/compliance/attendance` | |
| CP3 | Assessment Records | `/compliance/assessments` | |
| CP4 | Standardized Tests | `/compliance/tests` | |
| CP5 | Portfolio List | `/compliance/portfolios` | |
| CP6 | Portfolio Builder | `/compliance/portfolios/01900000-0000-7000-8000-000000000850` | `complyPortfolio1ID` |
| CP7 | Transcript List | `/compliance/transcripts` | |
| CP8 | Transcript Builder | `/compliance/transcripts/01900000-0000-7000-8000-000000000840` | `complyTranscript1ID` |

#### 4.1.12  Other Authenticated Routes

| # | Route | URL |
|---|-------|-----|
| OA1 | Recommendations | `/recommendations` |
| OA2 | Search Results | `/search` |
| OA3 | Notification Center | `/notifications` |

#### 4.1.13  Admin Routes

**Log out** from the parent account, then **log in as `admin@example.com`** (see §2.2).

| # | Route | URL | Notes |
|---|-------|-----|-------|
| AD1 | Admin Dashboard | `/admin` | |
| AD2 | User Management | `/admin/users` | |
| AD3 | User Detail | `/admin/users/01900000-0000-7000-8000-000000000011` | `seedParentID` |
| AD4 | Moderation Queue | `/admin/moderation` | |
| AD5 | Feature Flags | `/admin/flags` | |
| AD6 | Audit Log | `/admin/audit` | |
| AD7 | Methodology Config | `/admin/methodologies` | |

#### 4.1.14  Student Routes (Best-Effort)

Student routes require token-based session auth (not Kratos browser login). These are
**best-effort** — test them if possible, but mark as `BLOCKED` if auth cannot be established.

| # | Route | URL | Notes |
|---|-------|-----|-------|
| SR1 | Student Dashboard | `/student` | Needs student token |
| SR2 | Student Quiz | `/student/quiz/01900000-0000-7000-8000-000000000381` | `quizDef1ID` |
| SR3 | Student Video | `/student/video/01900000-0000-7000-8000-000000000306` | `videoDef1ID` |
| SR4 | Student Reader | `/student/read/01900000-0000-7000-8000-000000000301` | `activityDef1ID` |
| SR5 | Student Sequence | `/student/sequence/01900000-0000-7000-8000-000000000385` | `sequenceDef1ID` |

---

### Phase 2: User Journey Tests

Interactive workflows that exercise state transitions across multiple pages. For each
journey, follow the steps using Playwright MCP tools. Document any unexpected behavior.

#### Journey 1: Onboarding Flow

**Prerequisite:** Fresh login (no onboarding skip yet), or reset onboarding status.

1. Login as `seed@example.com`
2. Expect redirect to `/onboarding`
3. Snapshot the onboarding wizard — verify all steps are visible
4. Attempt to complete each step:
   - Family profile step (name, methodology selection)
   - Student addition step
   - Preferences / interests step
5. Verify completion redirects to the main app (`/`)
6. Navigate back to `/onboarding` — verify it does NOT redirect back (guard should pass)

**What to look for:**
- Step transitions work without errors
- Skip button works at any step
- Form validation messages display properly
- Methodology selection persists

#### Journey 2: Learning Workflow

1. Navigate to `/learning`
2. Click through to a student's progress view
3. Navigate to Activity Log → verify entries from seed data
4. Open a journal entry
5. Create a new journal entry → fill form → save
6. Navigate to Reading Lists → verify seed reading items
7. Open the Session Launcher → verify student selector works
8. Open a methodology-specific tool (e.g., Nature Journal)

**What to look for:**
- Student selector shows Emma and James
- Activity entries display with correct dates and types
- Journal creation form saves and appears in list
- Methodology tools render without errors

#### Journey 3: Social Interactions

1. Navigate to `/` (Feed)
2. Verify seed posts appear in the feed
3. Click on a post → verify post detail page
4. Try liking/unliking a post
5. Navigate to Friends → verify friend family appears
6. Navigate to Groups → verify seed group
7. Open group detail → verify members list
8. Navigate to Messages → verify seed conversation
9. Open conversation → verify message history
10. Navigate to Events → verify seed event
11. Open event detail → verify RSVP functionality

**What to look for:**
- Feed renders posts with author info and timestamps
- Like/unlike toggles correctly
- Group membership displays correctly
- Message history shows chronological order
- Event RSVP buttons work

#### Journey 4: Marketplace Browse & Purchase

1. Navigate to `/marketplace`
2. Verify seed listings appear
3. Click on a listing → verify detail page with price, description
4. Add to cart → navigate to `/marketplace/cart`
5. Verify cart shows the item
6. Navigate to Purchase History → verify seed purchases
7. Click on a purchase → try refund request flow

**What to look for:**
- Listing cards display correctly (title, price, methodology tag)
- Cart add/remove works
- Purchase history shows correct items and dates
- Refund form renders with purchase details

#### Journey 5: Creator Workflow

1. Navigate to `/creator`
2. Verify creator dashboard shows seed listings
3. Click "Create Listing" → fill the listing form
4. Navigate to Quiz Builder → create a simple quiz
5. Navigate to Sequence Builder → create a sequence
6. Navigate to Creator Reviews → verify seed reviews
7. Check Payout Setup page renders

**What to look for:**
- Dashboard metrics display (views, sales, revenue)
- Listing form has all required fields
- Quiz builder allows adding questions
- Sequence builder allows ordering items

#### Journey 6: Planning & Compliance

1. Navigate to `/calendar`
2. Switch between day/week views
3. Create a new schedule item → fill form → save
4. Navigate to Compliance Setup → verify state selector
5. Open Attendance Tracker → verify seed attendance records
6. Open Portfolio List → click into portfolio builder
7. Open Transcript List → click into transcript builder
8. Check Assessment Records page

**What to look for:**
- Calendar renders with correct date context
- Schedule creation form works
- Compliance setup shows state requirements
- Attendance records display with correct dates
- Portfolio and transcript builders render item lists

#### Journey 7: Settings & Account Management

1. Navigate to `/settings` (Family Settings)
2. Switch between tabs (Profile, Students, Preferences)
3. Navigate to Account Settings → verify profile info
4. Navigate to Session Management → verify active sessions
5. Navigate to Privacy Controls → verify toggle states
6. Navigate to Notification Prefs → verify channel options
7. Navigate to Notification History
8. Navigate to Block Management
9. Navigate to MFA Setup
10. Navigate to Subscription Manager

**What to look for:**
- Tab switching works without page reload
- Form fields are pre-populated with seed data
- Toggle switches respond to clicks
- Session list shows at least one active session
- Navigation between settings sub-pages works

#### Journey 8: Admin Workflow

1. Log in as `admin@example.com`
2. Navigate to `/admin`
3. Verify dashboard shows system metrics
4. Navigate to User Management → verify user list
5. Click on a user → verify detail page shows family info
6. Navigate to Moderation Queue → verify seed moderation items
7. Navigate to Audit Log → verify seed audit entries
8. Navigate to Feature Flags → verify flag list
9. Navigate to Methodology Config → verify methodology list

**What to look for:**
- Admin guard correctly gates access
- User list pagination works
- User detail shows roles, family, students
- Moderation actions (approve/reject) are available
- Audit log entries have timestamps and actor info
- Feature flag toggles respond

---

### Phase 3: Edge Cases & Exploratory Testing

#### 3.1  Empty States

Test pages with no data to verify graceful empty-state rendering:

- Navigate to `/search` with no query → expect empty state or prompt
- Navigate to `/search?q=xyznonexistent` → expect "no results"
- Create a new group → verify empty member list displays correctly
- Check marketplace with filter that returns no results

#### 3.2  Invalid / Missing Data Routes

Navigate to routes with non-existent IDs. Expect graceful error handling (not crashes):

```
/messages/00000000-0000-0000-0000-000000000000
/groups/00000000-0000-0000-0000-000000000000
/events/00000000-0000-0000-0000-000000000000
/post/00000000-0000-0000-0000-000000000000
/marketplace/listings/00000000-0000-0000-0000-000000000000
/learning/progress/00000000-0000-0000-0000-000000000000
/learning/quiz/00000000-0000-0000-0000-000000000000
/compliance/portfolios/00000000-0000-0000-0000-000000000000
/compliance/transcripts/00000000-0000-0000-0000-000000000000
/admin/users/00000000-0000-0000-0000-000000000000
/schedule/00000000-0000-0000-0000-000000000000/edit
/family/00000000-0000-0000-0000-000000000000
```

**Expected:** Error boundary or "not found" message — NOT a blank page or JS crash.

#### 3.3  Permission Boundaries

- **Parent accessing admin:** Navigate to `/admin` while logged in as `seed@example.com`
  → expect redirect or "access denied"
- **Admin accessing parent data:** Navigate to parent-specific routes as admin →
  verify behavior (should work if admin has a family, or show appropriate empty state)
- **Unauthenticated access:** Navigate to `/learning` without logging in →
  expect redirect to `/auth/login`
- **Cross-family isolation:** Verify friend family's data is not visible in seed account's
  views (check `/family/01900000-0000-7000-8000-000000000002` for appropriate display)

#### 3.4  Form Validation

Test form inputs with invalid data:

- **Register form:** Try submitting with empty fields, invalid email, weak password
- **Journal editor:** Submit with empty title/body
- **Event creation:** Submit with past date, missing required fields
- **Group creation:** Submit with empty name
- **Schedule editor:** Submit with conflicting times or missing student
- **Listing creation:** Submit with empty price or missing required fields

**Expected:** Inline validation errors, form does NOT submit.

#### 3.5  Methodology Tool Behavior

Methodology-specific tools should render based on the family's configured methodology. Test:

1. Verify which tools appear in navigation based on seed family's methodology
2. Navigate to each methodology tool (LR17–LR24) regardless of family config
3. Check if non-matching methodology tools show an appropriate message or are hidden

#### 3.6  Responsive Layout Checks

Resize the browser to test mobile breakpoints:

```
mcp__pw__browser_resize → width: 375, height: 812    # iPhone
mcp__pw__browser_resize → width: 768, height: 1024   # iPad
```

Key pages to check at mobile size:
- Feed (`/`)
- Learning Dashboard (`/learning`)
- Settings (`/settings`)
- Calendar (`/calendar`)
- Navigation (hamburger menu)

**Expected:** Content is usable, no horizontal overflow, navigation collapses to mobile menu.

#### 3.7  Notification System

1. Navigate to `/notifications` → verify notification center
2. Check notification bell in header → verify unread badge
3. Click on a notification → verify it navigates to relevant content
4. Check notification preferences at `/settings/notifications`

#### 3.8  Search Functionality

1. Navigate to `/search`
2. Try searching for a known entity (e.g., seed listing title, student name)
3. Verify search results link to the correct detail pages
4. Test search with special characters
5. Test empty search query

---

## 5  Gap Classification

### 5.1  Severity Levels

| Level | Definition | Examples |
|-------|-----------|----------|
| **Critical** | Blocks core user workflow or violates privacy/legal | Auth broken, data leak, COPPA violation |
| **High** | Feature non-functional but workaround exists | Form won't submit, page crashes on valid data |
| **Medium** | Degraded experience but functional | Missing empty states, poor error messages, layout issues |
| **Low** | Polish / improvement opportunity | Visual glitches, minor UX friction, missing loading states |

### 5.2  What Counts as a Gap

**IS a gap:**
- Page fails to render (JS error, blank page, error boundary)
- Console errors on page load
- Form submission fails silently (no error, no success)
- Navigation leads to unexpected destination
- Data from seed not appearing where expected
- Missing permission checks (unauthorized access succeeds)
- Broken responsive layout (content unusable)
- Missing ARIA labels on interactive elements (accessibility)

**Is NOT a gap:**
- Features documented as "not yet implemented" in gap reports
- External services not configured (Hyperswitch, Typesense, Thorn) — these are known
- Student routes blocked by token auth — mark as `BLOCKED`, not `FAIL`
- Placeholder content clearly marked as "coming soon" or similar

---

## 6  Output Report Format

Produce a structured gap report as a markdown file. Follow this template:

```markdown
# E2E Exploratory Testing Report — {DATE}

> **Agent:** {model name}
> **Duration:** {approximate time}
> **Scope:** Full application E2E via Playwright MCP

---

## 1  Executive Summary

- **Routes tested:** {N} / {total}
- **Pass:** {N} | **Warn:** {N} | **Fail:** {N} | **Blocked:** {N}
- **Critical gaps found:** {N}
- **High gaps found:** {N}
- **Medium gaps found:** {N}
- **Low gaps found:** {N}

---

## 2  Route Smoke Test Results

### 2.1  Auth & Legal Routes

| # | Route | Status | Notes |
|---|-------|--------|-------|
| A1 | `/auth/login` | PASS | Renders login form correctly |
| ... | ... | ... | ... |

### 2.2  Social Routes
(same format)

### 2.3  Learning Routes
(same format)

(... continue for all route groups)

---

## 3  User Journey Results

### Journey 1: Onboarding Flow
- **Status:** PASS / PARTIAL / FAIL
- **Steps completed:** {N}/{total}
- **Issues found:**
  - {description of issue}

(... continue for all journeys)

---

## 4  Edge Case Results

### 4.1  Empty States
| Page | Status | Notes |
|------|--------|-------|
| Search (no query) | PASS | Shows "Enter a search term" prompt |
| ... | ... | ... |

### 4.2  Invalid Routes
(same format)

### 4.3  Permission Boundaries
(same format)

---

## 5  Gap Register

### 5.1  Critical

#### GAP-E2E-C{N} — {Title}

| Field | Value |
|-------|-------|
| **Route(s)** | {affected routes} |
| **Observed** | {what happened} |
| **Expected** | {what should happen} |
| **Console errors** | {any JS errors from console} |
| **Screenshot** | {filename} |

### 5.2  High
(same format)

### 5.3  Medium
(same format)

### 5.4  Low
(same format)

---

## 6  Screenshots

All screenshots saved to `specs/screenshots/e2e/` with naming convention:
`{route-group}-{route-number}-{status}.png`

Example: `social-S1-PASS.png`, `learning-LR9-FAIL.png`
```

---

## 7  Execution Notes

### 7.1  Recommended Order

1. Auth & Legal routes (unauthenticated)
2. Login as parent → Onboarding → Skip
3. All parent-accessible routes (Phase 1)
4. User journeys 1–7 (Phase 2)
5. Logout → Login as admin
6. Admin routes (Phase 1) → Journey 8
7. Edge cases (Phase 3)

### 7.2  Time Budget

This is an extensive test suite. If time is limited, prioritize:
1. **Phase 1 smoke test** — catches the most issues per minute
2. **Journey 2 (Learning)** and **Journey 3 (Social)** — highest user traffic
3. **Edge case §3.2 (Invalid routes)** — catches missing error handling

### 7.3  Error Recovery

If a page crash or redirect loop occurs:
1. Take a screenshot and note the console errors
2. Navigate directly to the next route in sequence — do NOT try to debug
3. Record the issue in the gap register and continue

If authentication is lost (session expired):
1. Navigate to `/auth/login`
2. Re-authenticate using the appropriate credentials (§2.1 or §2.2)
3. Continue from where you left off

### 7.4  Screenshot Storage

Save all screenshots to `specs/screenshots/e2e/`. Create the directory if it doesn't exist:

```bash
mkdir -p specs/screenshots/e2e
```

Use descriptive filenames: `{group}-{number}-{status}.png`
