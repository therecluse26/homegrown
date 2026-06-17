# E2E Exploratory Testing — Agent Prompt

> **Purpose:** A self-contained prompt that an AI agent can execute to perform exhaustive
> exploratory end-to-end testing of the Homegrown Academy application using Playwright MCP
> tools. The goal is to systematically discover **functional gaps AND visual/design-system
> defects** that code review alone cannot reveal.
>
> **Output:** A structured gap report + coverage ledger (§6, §7).

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

### 1.3  Viewport Configuration (E6)

Every surface MUST be tested at **two mandatory viewports** (see §4.2 — E6):

| Label | Command | Dimensions |
|-------|---------|------------|
| Desktop | `mcp__pw__browser_resize → width: 1440, height: 900` | 1440×900 |
| Mobile | `mcp__pw__browser_resize → width: 390, height: 844` | 390×844 |

Do **not** use any other viewport as a substitute. `1280×800` is retired.

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
   - { name: "Password", type: "textbox", ref: <password-field-ref>, value: "SeedPassword123!" }
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

Testing proceeds through three phases. Every phase applies the **per-surface execution
pipeline** defined in §4.1–§4.6 before recording a verdict. Read §4.1–§4.6 completely
before beginning Phase 1.

---

### 4.1  E1 — Render-AND-Judge Gate (Verdict Rule)

> **Rule: A surface verdict MUST NOT be filed without (a) a rendered screenshot and
> (b) explicit citation of rubric items from §4.3 (E2) and §4.4 (E3). Rendering alone
> is NOT a pass. "Content present, no console errors" is NOT sufficient for PASS.**

For every surface, the mandatory pipeline is:

```
RENDER  →  JUDGE  →  FILE
```

Never:

```
RENDER  →  FILE          ← forbidden (no judgment)
RENDER  →  JUDGE  →  ...   ← forbidden if filed without rubric citation
```

**What "judge" requires:**
1. A `mcp__pw__browser_take_screenshot` captured (not just snapshot).
2. A visual quality score from E2 rubric (§4.3) stated explicitly: `VQ: 3/3`.
3. Any token/Hearth violations from E3 (§4.4) listed or confirmed absent: `E3: none`.
4. State coverage from E7 matrix (§4.5) noted: which states were exercised.

Only after these four items are recorded may a verdict (`PASS` / `WARN` / `FAIL` / `BLOCKED`) be filed.

**Verdict definitions (updated):**

| Verdict | Meaning |
|---------|---------|
| **PASS** | Renders, no console errors, VQ ≥ 2/3, no E3 violations |
| **WARN** | Renders but VQ = 1/3, or has console warnings, or minor E3 violations |
| **FAIL** | Page crashes / error boundary / JS errors, **or** VQ = 0/3, **or** critical E3 violations (off-token color, 1px sectioning border) |
| **BLOCKED** | Cannot reach (auth issue, redirect loop, token requirement) |

> **Important:** A surface with VQ = 0/3 or critical E3 violations MUST be `FAIL` even
> if the page renders and has no console errors. Raw/cramped/off-token is a defect.

---

### 4.2  E6 — Mandatory Viewports

Every surface in Phase 1 (smoke test) MUST be tested at both viewports. Journeys (Phase 2)
MUST be run at desktop; repeat the journey at mobile for any flows modified in recent work.

```
Desktop:  mcp__pw__browser_resize → width: 1440, height: 900
Mobile:   mcp__pw__browser_resize → width: 390, height: 844
```

Record the viewport in the verdict: `PASS/D` (desktop), `PASS/M` (mobile), `FAIL/M` (mobile fail).

If a surface passes desktop but fails mobile, the overall verdict is `WARN` at minimum.
If it crashes at mobile, the verdict is `FAIL`.

---

### 4.3  E2 — Visual-Quality Rubric

Score each surface 0–3 using the four criteria below. **All four must be assessed.** A
score of 3 means all four pass cleanly. Record as `VQ: N/3` in the surface row.

Partial credit (0.5) is allowed for borderline cases; round down when filing.

#### Criterion A — Hierarchy Visible

A stranger should identify primary, secondary, and tertiary content within two seconds of
viewing the screenshot. Primary actions (CTAs) must stand out; supporting text must recede.

- **3:** Clear 3-tier hierarchy. Primary: uses `text-primary` or `bg-primary` token.
  Secondary: `text-on-surface-variant`. Tertiary: clearly subordinate size or color.
- **2:** Two tiers visible but a third is ambiguous or missing.
- **1:** Only one visual weight used; everything competes equally.
- **0:** No hierarchy. Everything is the same size, color, and weight ("programmer default").

#### Criterion B — Spacing on Token Scale

All padding, margin, and gap values MUST come from Tailwind's spacing scale as registered in
`frontend/src/styles/tokens.css`. Reference values: `spacing-1` (0.35rem), `spacing-2`
(0.7rem), `spacing-4` (1.4rem), `spacing-6` (2.1rem), `spacing-8` (2.8rem), `spacing-12`
(4.2rem), `spacing-16` (5.6rem). No arbitrary values (`p-[13px]`, `gap-[22px]`).

- **3:** All gaps generous and from token scale. Breathing room evident; elements don't crowd edges.
- **2:** Mostly correct but one area is cramped or uses a one-off value.
- **1:** Visible crowding or elements touching edges; likely off-scale.
- **0:** Densely packed; default browser spacing; content runs to viewport edge.

#### Criterion C — Ruthless Alignment

Every element must align to a grid, a shared baseline, or a common edge. Misaligned text,
ragged card grids, and uneven column starts are defects.

- **3:** Perfect grid alignment. Cards, labels, and content share left/right edges. Text baselines match within rows.
- **2:** Mostly aligned with one visible exception.
- **1:** Visibly ragged layout; multiple elements misaligned.
- **0:** No alignment system apparent.

#### Criterion D — Type System Consistency

Headlines MUST use Plus Jakarta Sans (`font-display` token); body/utility MUST use Manrope
(`font-sans` token). Sizes MUST come from the type scale defined in `components.css`
(`display-lg`, `headline-md`, `title-sm`, `body-lg`, `label-md`, etc.). No ad-hoc font
sizes (`text-[17px]`).

- **3:** Consistent application of display/headline/title/body/label scale. No free-form sizes.
- **2:** Mostly correct; one component uses a mismatched size or weight.
- **1:** Multiple inconsistent sizes; visual hierarchy broken by type variation.
- **0:** Default browser font-size throughout; or multiple different fonts not from the token set.

---

### 4.4  E3 — Token & Curated-Hearth Conformance Scan

After capturing screenshots, scan for the following violations. Each violation must be cited
by its offending element, the incorrect value observed, and the correct token/rule that should
apply. Record findings as a list under `E3:` in the surface row. If no violations are found,
write `E3: clean`.

#### Violation Checklist

| ID | What to Flag | Correct Token / Rule |
|----|-------------|---------------------|
| **E3-A** | Any hex color value not in the token palette (check `tokens.css §2`) | Use `text-primary`, `bg-surface-container`, `text-error`, etc. from `DESIGN_TOKENS.md §2` |
| **E3-B** | 1px solid borders used for sectioning / dividing layout regions | **Forbidden.** Use background color shifts (`bg-surface-container-low` vs `bg-surface`) per DESIGN.md "No-Line Rule" |
| **E3-C** | Horizontal dividers (`<hr>`, `divide-y`, `border-b`) between list items | **Forbidden.** Use `spacing-4` (1.4rem) vertical gap or alternating `surface` / `surface-container-low` per DESIGN.md §5 |
| **E3-D** | Arbitrary z-index values (`z-[50]`, `z-[100]`, inline `z-index: 999`) | Use Tailwind's named z-scale (`z-0` through `z-50`) or token-registered z-values only |
| **E3-E** | Off-scale spacing (`p-[13px]`, `gap-[22px]`, inline `margin: 7px`) | Replace with nearest token step (`spacing-2` = 0.7rem, `spacing-4` = 1.4rem, etc.) |
| **E3-F** | Wrong typeface for context (e.g., Manrope used for `display-lg` headline) | Headlines: `font-display` (Plus Jakarta Sans). Body/utility: `font-sans` (Manrope) |
| **E3-G** | Flat drop shadows with full opacity or tight blur (< 8px) | Ambient only: 32px+ blur, 4% opacity using `on-surface` tint (`#1b1c1a`) per DESIGN.md §4 |
| **E3-H** | Cards without `xl` radius on student content, or without `lg` on parent data | Student cards: `rounded-xl` (1.5rem). Parent data cards: `rounded-lg` (1rem) per DESIGN.md §5 |
| **E3-I** | Primary buttons not using `bg-primary` (`#0c5252`) + `text-on-primary` (`#ffffff`) | Per DESIGN.md §5 Buttons spec |
| **E3-J** | Navigation using opaque background instead of `surface-container-low` at 80% opacity with `backdrop-blur` | Per DESIGN.md §2 "Glass & Gradient" Rule |

**Severity mapping for E3 violations:**

| Severity | Violations |
|----------|-----------|
| **High** (→ FAIL) | E3-A (off-token hex), E3-B (1px sectioning borders), E3-C (list dividers) |
| **Medium** (→ WARN) | E3-D (arbitrary z-index), E3-E (off-scale spacing), E3-F (wrong typeface), E3-G (wrong shadow) |
| **Low** (→ WARN or note) | E3-H (wrong radius), E3-I (button token), E3-J (nav glass) |

---

### 4.5  E7 — State-Coverage Matrix

For every surface, assess each interaction state below. Mark each cell as:
- **✓** — tested and judged (apply E1 pipeline)
- **N/A** — state is structurally impossible for this surface (document reason)
- **—** — not tested this run (must be listed in E9 coverage ledger as untested)

| State | Description | How to exercise |
|-------|-------------|-----------------|
| **Default** | Loaded, data present | Navigate to route normally |
| **Loading** | In-flight API state (skeleton/spinner) | Slow network, or inspect loading state during first render |
| **Empty** | No data in the collection | Navigate with no seed / filter to empty result |
| **Error** | API error, offline, or server 500 | Disconnect backend or use an invalid ID |
| **No-permission** | Unauthorized user accessing route | Use wrong account role (parent→admin route, etc.) |
| **Overflow/long-content** | Unusually long text, many items | Observe with large seed data or manually trigger |
| **Focus/hover/active** | Interactive element states | Tab through the page; hover over buttons/cards |

**Minimum state coverage per surface per viewport:**

- `Default` and `No-permission` (where applicable) are **required** in Phase 1.
- `Empty` and `Error` are **required** for list/collection views.
- `Focus/hover/active` is **required** for any surface with interactive controls.
- Remaining states are best-effort; note them as untested in E9 if skipped.

---

### 4.6  E9 — Coverage Ledger

The coverage ledger is a mandatory output artifact. It records every `surface × viewport ×
state` cell as **covered**, **not covered**, or **N/A**, with a reason when not covered.

**No silent truncation.** If a surface was skipped, it MUST appear in the ledger as
`not covered` with a reason. Omitting a row is a reporting defect.

The ledger template is in §6.3. Generate it as part of the output report.

---

### Phase 1: Route Smoke Test

**Goal:** Apply the E1 pipeline (§4.1) to every route at both viewports (E6, §4.2).
For each route, perform this sequence **at desktop then at mobile**:

```
1. mcp__pw__browser_resize → [Desktop: 1440×900] [Mobile: 390×844]
2. mcp__pw__browser_navigate → the URL
3. mcp__pw__browser_snapshot → accessibility tree (DOM structure check)
4. mcp__pw__browser_console_messages → level: "error" (zero tolerance)
5. mcp__pw__browser_take_screenshot → type: "png" (REQUIRED — judgment requires pixels)
6. Judge: score VQ (§4.3), scan E3 violations (§4.4), note states covered (§4.5)
7. File verdict: PASS / WARN / FAIL / BLOCKED with VQ score + E3 result
```

**Do not file a verdict until steps 5 and 6 are complete.**

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

Interactive workflows that exercise state transitions across multiple pages. Apply the E1
judgment pipeline at key screens within each journey (not every page — focus on state
transitions, final states, and error branches).

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

Test pages with no data to verify graceful empty-state rendering. **Apply E1 judgment:**
empty states must be visually designed — not raw "No items found." text on a blank page.
A styled, on-brand empty state scores VQ ≥ 2/3; a raw text fallback scores VQ = 0/3 → FAIL.

- Navigate to `/search` with no query → expect empty state or prompt
- Navigate to `/search?q=xyznonexistent` → expect "no results"
- Create a new group → verify empty member list displays correctly
- Check marketplace with filter that returns no results

#### 3.2  Invalid / Missing Data Routes

Navigate to routes with non-existent IDs. Expect graceful error handling (not crashes).
**Apply E1 judgment:** error states must render a designed error component, not a blank page.

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

**Expected:** Inline validation errors, form does NOT submit. Error text uses `text-error`
(`#ba1a1a`); field background shifts to `bg-error-container` (`#ffdad6`) per DESIGN.md §5.

#### 3.5  Methodology Tool Behavior

Methodology-specific tools should render based on the family's configured methodology. Test:

1. Verify which tools appear in navigation based on seed family's methodology
2. Navigate to each methodology tool (LR17–LR24) regardless of family config
3. Check if non-matching methodology tools show an appropriate message or are hidden

#### 3.6  Overflow / Long-Content States

Test how surfaces handle unusually long content:

- Feed with many posts — scroll to verify pagination or infinite scroll works
- Group with many members — verify overflow list handling
- Conversation with long message history — scroll behavior
- Long student name or family name in header elements — no text overflow clip

**Apply E1 judgment at the overflow state.** A surface that clips text without ellipsis or
wraps in a way that breaks layout scores VQ = 0/3 on Criterion C.

#### 3.7  Focus / Hover / Active States

Tab through key interactive surfaces to verify keyboard navigability:

- Tab through the main navigation — verify focus ring is visible
- Tab through a form — verify all fields are reachable
- Hover over buttons and cards — verify hover states are visible
- Check that focus ring uses `primary` color token (`#0c5252`) not browser default blue

**E3 note:** Missing focus ring or wrong focus ring color is an E3-A violation (off-token).

#### 3.8  Notification System

1. Navigate to `/notifications` → verify notification center
2. Check notification bell in header → verify unread badge
3. Click on a notification → verify it navigates to relevant content
4. Check notification preferences at `/settings/notifications`

#### 3.9  Search Functionality

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
| **High** | Feature non-functional **or** severe visual/design-system violation | Form won't submit; page crashes; off-token hex colors; 1px sectioning borders; VQ = 0/3 |
| **Medium** | Degraded experience but functional **or** moderate visual defect | Poor error messages; off-scale spacing; arbitrary z-index; VQ = 1/3; missing empty/loading states |
| **Low** | Minor polish / improvement opportunity | Minor hover state missing; radius one step off; VQ = 2/3 with one criterion missed |

> **Design-system and visual-polish defects are first-class defects at the same severity
> level as functional defects.** A screen with raw HTML, cramped spacing, or off-token
> colors is a High defect, not Low. "It renders" is not an excuse for "it looks broken."

### 5.2  Visual Polish & Design-System Conformance (First-Class Category)

This is an explicit top-level gap category in the gap register (§6.4). Report visual polish
and design-system gaps here, not buried under "Medium/Low miscellaneous."

**Severity mapping for visual defects:**

| Defect | Severity |
|--------|---------|
| VQ = 0/3 (no hierarchy, default browser styling, no spacing) | **High** |
| E3-A: off-token hex color | **High** |
| E3-B: 1px sectioning border | **High** |
| E3-C: list dividers | **High** |
| VQ = 1/3 (one criterion met) | **Medium** |
| E3-D through E3-G violations | **Medium** |
| Missing empty/loading/error state design | **Medium** |
| VQ = 2/3 (three criteria met, one missed) | **Low** |
| E3-H through E3-J violations | **Low** |

### 5.3  What Counts as a Gap

**IS a gap:**
- Page fails to render (JS error, blank page, error boundary)
- Console errors on page load
- Form submission fails silently (no error, no success)
- Navigation leads to unexpected destination
- Data from seed not appearing where expected
- Missing permission checks (unauthorized access succeeds)
- Broken responsive layout (content unusable at mobile viewport)
- Missing ARIA labels on interactive elements (accessibility)
- VQ = 0/3 or 1/3 for any surface
- Any E3-A through E3-C violation (High)
- Any E3-D through E3-J violation (Medium/Low)
- Missing designed empty/loading/error states (raw text fallbacks)

**Is NOT a gap:**
- Features documented as "not yet implemented" in gap reports
- External services not configured (Hyperswitch, Typesense, Thorn) — these are known
- Student routes blocked by token auth — mark as `BLOCKED`, not `FAIL`
- Placeholder content clearly marked as "coming soon" or similar

---

## 6  Output Report Format

Produce a structured gap report as a markdown file. Follow this template:

````markdown
# E2E Exploratory Testing Report — {DATE}

> **Agent:** {model name}
> **Duration:** {approximate time}
> **Scope:** Full application E2E via Playwright MCP
> **Viewports:** Desktop 1440×900 + Mobile 390×844

---

## 1  Executive Summary

- **Routes tested:** {N} / {total}
- **Pass:** {N} | **Warn:** {N} | **Fail:** {N} | **Blocked:** {N}
- **VQ Distribution:** 3/3: {N} | 2/3: {N} | 1/3: {N} | 0/3: {N}
- **E3 Violations:** High: {N} | Medium: {N} | Low: {N}
- **Critical gaps found:** {N}
- **High gaps found:** {N}  (functional: {N} / visual: {N})
- **Medium gaps found:** {N}  (functional: {N} / visual: {N})
- **Low gaps found:** {N}

---

## 2  Route Smoke Test Results

### 2.1  Auth & Legal Routes

| # | Route | Status/D | Status/M | VQ | E3 | Notes |
|---|-------|----------|----------|----|-----|-------|
| A1 | `/auth/login` | PASS/D | PASS/M | 3/3 | clean | Renders login form correctly |
| ... | ... | ... | ... | ... | ... | ... |

### 2.2  Social Routes
(same format)

### 2.3  Learning Routes
(same format)

(... continue for all route groups)

---

## 3  User Journey Results

### Journey 1: Onboarding Flow
- **Status:** PASS / PARTIAL / FAIL
- **Key screens judged:** {list of screens with VQ scores}
- **E3 findings:** {list or "none"}
- **Issues found:**
  - {description of issue}

(... continue for all journeys)

---

## 4  Edge Case Results

### 4.1  Empty States
| Page | Status | VQ | E3 | Notes |
|------|--------|----|-----|-------|
| Search (no query) | PASS | 2/3 | clean | Shows "Enter a search term" prompt |
| ... | ... | ... | ... | ... |

### 4.2  Invalid Routes
(same format)

### 4.3  Permission Boundaries
(same format)

### 4.4  Overflow & Focus States
(same format)

---

## 5  Gap Register

### 5.1  Critical

#### GAP-E2E-C{N} — {Title}

| Field | Value |
|-------|-------|
| **Route(s)** | {affected routes} |
| **Viewport(s)** | Desktop / Mobile / Both |
| **Observed** | {what happened} |
| **Expected** | {what should happen} |
| **Console errors** | {any JS errors from console} |
| **VQ Score** | {N}/3 — {which criteria failed} |
| **E3 Violations** | {list or "none"} |
| **Screenshot** | {filename} |

### 5.2  High (Functional)
(same format)

### 5.3  High (Visual Polish & Design-System Conformance)
(same format — cite offending element, incorrect value, correct token/rule)

### 5.4  Medium (Functional)
(same format)

### 5.5  Medium (Visual Polish & Design-System Conformance)
(same format)

### 5.6  Low
(same format)

---

## 6  Screenshots

All screenshots saved to `research/screenshots/e2e/` with naming convention:
`{route-group}-{route-number}-{viewport}-{status}.png`

Example: `social-S1-desktop-PASS.png`, `learning-LR9-mobile-FAIL.png`
````

---

### 6.3  Coverage Ledger Template (E9)

The coverage ledger MUST be included in the output report. Copy this table and fill every
row. **Do not omit rows.** Every surface in §4.1 must appear. Use one row per
surface × viewport combination. States that were not exercised MUST appear with reason.

```markdown
## 7  Coverage Ledger (E9)

| Surface | Route | Viewport | Default | Loading | Empty | Error | No-perm | Overflow | Focus/Hover | Notes |
|---------|-------|----------|---------|---------|-------|-------|---------|----------|-------------|-------|
| Login | /auth/login | Desktop | ✓ | N/A | N/A | N/A | N/A | N/A | ✓ | |
| Login | /auth/login | Mobile | ✓ | N/A | N/A | N/A | N/A | N/A | — | not tested on mobile |
| Feed | / | Desktop | ✓ | — | N/A | — | ✓ | ✓ | ✓ | error/loading skipped — time budget |
| Feed | / | Mobile | ✓ | — | N/A | — | N/A | — | — | partial mobile coverage |
| ... (one row per surface × viewport) | | | | | | | | | | |

**Legend:** ✓ = covered and judged | N/A = structurally impossible (reason in Notes) | — = not tested this run (reason in Notes)

**Untested cells summary:**
- {N} cells untested: {list reasons — time budget / session expiry / auth blocker / etc.}
- Silent truncation count: 0 (all untested cells are listed above)
```

> **Rule:** "Silent truncation count" MUST be 0. If you cannot complete a row, write it
> anyway with `—` and a reason. Never leave a surface out of the ledger.

---

## 7  Execution Notes

### 7.1  Recommended Order

1. Auth & Legal routes (unauthenticated) — desktop then mobile
2. Login as parent → Onboarding → Skip
3. All parent-accessible routes (Phase 1) — desktop then mobile
4. User journeys 1–7 (Phase 2)
5. Logout → Login as admin
6. Admin routes (Phase 1) → Journey 8
7. Edge cases (Phase 3)
8. Fill coverage ledger (E9) from results

### 7.2  Time Budget

This is an extensive test suite. If time is limited, prioritize:
1. **Phase 1 smoke test at desktop** — catches the most issues per minute
2. **Phase 1 smoke test at mobile** — repeat top 20 highest-traffic routes
3. **Journey 2 (Learning)** and **Journey 3 (Social)** — highest user traffic
4. **Edge case §3.2 (Invalid routes)** — catches missing error handling
5. **Edge case §3.7 (Focus states)** — catches accessibility gaps

When time is cut short, still fill all ledger rows — mark skipped cells `—` with reason.

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

Save all screenshots to `research/screenshots/e2e/`. Create the directory if it doesn't exist:

```bash
mkdir -p research/screenshots/e2e
```

Use descriptive filenames: `{group}-{number}-{viewport}-{status}.png`

Desktop: `social-S1-desktop-PASS.png`
Mobile: `social-S1-mobile-WARN.png`
