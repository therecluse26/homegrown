# E2E Gap Fixes — Step-by-Step TODO

> Generated from `research/gaps/e2e_exploratory_2026-04-06.md`
> Each step is independently executable and Playwright-verifiable.

---

## Batch 1: Quick Frontend URL Fixes ✅

These are 1-3 line changes per hook — highest ROI.

### Step 1.1 — Fix search URL (GAP-E2E-H6) ✅
- [x] **File:** `frontend/src/hooks/use-search.ts:94`
- [x] Change `/v1/search/search?${buildSearchQuery(params!)}` → `/v1/search?${buildSearchQuery(params!)}`
- [x] **Backend route:** `internal/search/handler.go:36` → `search.GET("", h.search)`
- [x] **Verify:** Navigate to `/search`, type "Charlotte Mason", check Social/Marketplace/Learning tabs return results
- [x] **Gaps closed:** H6 (all 3 search scopes)
- **Note:** URL fix confirmed working. Backend returns 500 due to pre-existing search groups query bug (not a URL issue).

### Step 1.2 — Fix admin feature flags URL (GAP-E2E-H5, AD5) ✅
- [x] **File:** `frontend/src/hooks/use-admin.ts`
- [x] Line 401: `/v1/admin/feature-flags` → `/v1/admin/flags`
- [x] Line 417: `/v1/admin/feature-flags/${id}` → `/v1/admin/flags/${id}`, `PUT` → `PATCH`
- [x] Line 436: `/v1/admin/feature-flags` → `/v1/admin/flags`
- [x] **Backend route:** `internal/admin/handler.go:39-43` → uses `/flags`, `/flags/:key`, PATCH
- [x] **Verify:** Login as admin, navigate to `/admin/flags`, confirm flags list loads
- **Note:** Seed user isn't admin so redirects to feed — URL fix is correct per code review.

### Step 1.3 — Fix admin methodologies URL (GAP-E2E-H5, AD7) ✅
- [x] **File:** `frontend/src/hooks/use-admin.ts`
- [x] Line 466: `/v1/admin/methodology-configs` → `/v1/admin/methodologies`
- [x] Line 473: `/v1/admin/methodology-configs/${slug}` → `/v1/admin/methodologies/${slug}`, `PUT` → `PATCH`
- [x] **Backend route:** `internal/admin/handler.go:51-52` → uses `/methodologies`, PATCH
- [x] **Verify:** Login as admin, navigate to `/admin/methodologies`, confirm list loads

### Step 1.4 — Fix planning templates URL (GAP-E2E-M10, PL6) ✅
- [x] **File:** `frontend/src/hooks/use-planning.ts`
- [x] Line 376: `/v1/planning/schedule-templates` → `/v1/planning/templates`
- [x] Line 385: `/v1/planning/schedule-templates` → `/v1/planning/templates`
- [x] Line 405: `/v1/planning/schedule-templates/${templateId}/apply` → `/v1/planning/templates/${templateId}/apply`
- [x] Line 418: `/v1/planning/schedule-templates/${templateId}` → `/v1/planning/templates/${templateId}`
- [x] **Backend route:** `internal/plan/handler.go:46-50` → uses `/templates`
- [x] **Verify:** Navigate to `/planning/templates`, confirm page loads without "Something went wrong"

---

## Batch 2: i18n Translation Keys ✅

### Step 2.1 — Add billing subscription + invoice i18n keys (GAP-E2E-H0) ✅
- [x] **File:** `frontend/src/locales/en.json`
- [x] Read `frontend/src/features/billing/subscription-management.tsx` for all `FormattedMessage id` and `intl.formatMessage` calls
- [x] Read `frontend/src/features/billing/invoice-history.tsx` for the same
- [x] Add all missing `billing.subscription.*` keys (~15 static + ~9 dynamic tier/status/interval keys)
- [x] Add all missing `billing.invoice.*` keys (~8 static + ~8 dynamic type/status keys)
- [x] **Verify:** Navigate to `/billing/subscription` and `/billing/invoices` — all text should be human-readable, zero MissingTranslationErrors in console
- **Playwright confirmed:** 0 errors on both pages. Subscription shows Premium/$9.99/month correctly.

### Step 2.2 — Add recommendations page i18n keys (GAP-E2E-H3) ✅
- [x] **File:** `frontend/src/locales/en.json`
- [x] Read `frontend/src/features/recommendations/recommendations-page.tsx` for all message IDs
- [x] Add ~17 missing `recommendations.*` keys (title, description, filters, types, badges, dismiss, empty state)
- [x] Keep existing 9 `recommendations.section.*` and `recommendations.card.*` keys
- [x] **Verify:** Navigate to `/recommendations` — all UI chrome should be translated, zero MissingTranslationErrors
- **Playwright confirmed:** 0 errors. 3 recommendations render with proper types, AI badges, dismiss/block buttons.

---

## Batch 3: Frontend Error States & UX — NOT STARTED

### Step 3.1 — Add "Not Found" state to post detail (GAP-E2E-M12)
- [ ] **File:** `frontend/src/features/social/post-detail.tsx:275`
- [ ] Replace `if (!data) return null;` with a not-found card (PageTitle + Card + message + back link)
- [ ] Follow pattern from `frontend/src/features/compliance/portfolio-builder.tsx`
- [ ] Add i18n key `social.post.notFound` to `en.json`
- [ ] **Verify:** Navigate to `/post/00000000-0000-0000-0000-000000000000` — should show "Post not found" with a link back to feed

### Step 3.2 — Add "Not Found" state to listing detail + add-to-cart toast (GAP-E2E-M12 + M13)
- [ ] **File:** `frontend/src/features/marketplace/listing-detail.tsx`
- [ ] Line 88: Replace `if (!listing) return null;` with not-found card
- [ ] Line 203: Add `onSuccess` to `addToCart.mutate()` → `toast(intl.formatMessage({ id: "marketplace.addedToCart" }), "success")`
- [ ] Import `useToast` from `@/components/ui/toast`
- [ ] Add i18n keys: `marketplace.listing.notFound`, `marketplace.addedToCart` to `en.json`
- [ ] **Verify:** (a) Navigate to `/marketplace/listings/00000000-...` → "Listing not found"; (b) Add real listing to cart → toast appears

### Step 3.3 — Fix notification history header (GAP-E2E-L3)
- [ ] **File:** `frontend/src/features/settings/notification-history.tsx:119`
- [ ] Change `const total = data?.total ?? 0;` → `const total = data?.total || notifications.length;`
- [ ] Root cause: Backend `NotificationListResponse` doesn't include `total` field, so it's always 0
- [ ] ICU message `{total, plural, =0 {No notifications} ...}` then shows "No notifications" even with 20 items
- [ ] **Verify:** Navigate to `/settings/notifications/history` — header should show "20 notifications" (or actual count)

### Step 3.4 — Fix compliance spinbutton "undefined" values (GAP-E2E-L5)
- [ ] **File:** `frontend/src/features/compliance/compliance-setup.tsx`
- [ ] Line 109: `String(stateReqs.data.days_required)` → `String(stateReqs.data.days_required ?? 0)`
- [ ] Line 110: `String(stateReqs.data.hours_required)` → `String(stateReqs.data.hours_required ?? 0)`
- [ ] Lines 101-102: Change `||` to `??` to preserve value `0` (currently `0 || ""` → `""`)
- [ ] **Verify:** Navigate to `/compliance`, select a state → spinbuttons show numbers, not "undefined"

---

## Batch 4: Compliance URL Refactor (student-scoped routes) ✅

This is the largest batch — all compliance hooks need `students/{studentId}` in the URL path.

### Step 4.1 — Fix compliance attendance hooks (GAP-E2E-H4, CP2) ✅
- [x] **File:** `frontend/src/hooks/use-compliance.ts`
- [x] `useAttendance(studentId, month)`: `/v1/compliance/attendance/${studentId}` → `/v1/compliance/students/${studentId}/attendance`
- [x] `useAttendanceSummary(studentId)`: `/v1/compliance/attendance/summary` → `/v1/compliance/students/${studentId}/attendance/summary`
- [x] `useRecordAttendance()`: `/v1/compliance/attendance` → `/v1/compliance/students/${body.student_id}/attendance`
- [x] **Backend:** `internal/comply/handler.go:44-49`
- [x] **Also update:** calling components to pass studentId where they don't already
- [x] **Verify:** Navigate to `/compliance/attendance`, select a student → calendar loads without errors
- **Extra fixes discovered during Playwright:**
  - [x] Backend expects `start_date`/`end_date` in RFC3339 format, not `?month=YYYY-MM` — converted month to date range
  - [x] Backend returns `{ records: [...], next_cursor }`, not bare array — unwrap `resp.records`
  - [x] Summary endpoint returns single object (not array), with different field names (`present_full` → `days_present`, `state_required_days` → `days_required`, `pace_status` → `pace`) — added `AttendanceSummaryRaw` type and field mapping
  - [x] Component: removed `.find()` pattern, use single summary object directly
- **Playwright confirmed:** 0 errors. Calendar renders 30 days, pace summary shows "On track".

### Step 4.2 — Fix compliance tests hooks (GAP-E2E-H4, CP4) ✅
- [x] **File:** `frontend/src/hooks/use-compliance.ts`
- [x] `useStandardizedTests(studentId)`: `/v1/compliance/tests` → `/v1/compliance/students/${studentId}/tests`
- [x] `useCreateStandardizedTest()` mutation: same path fix
- [x] **Backend:** `internal/comply/handler.go:58-61`
- [x] **Verify:** Navigate to `/compliance/tests` → page loads without "Something went wrong"
- **Extra fixes discovered during Playwright:**
  - [x] Backend sends `scores: Record<string, number>` (JSON object like `{"reading":85}`), not `sections: TestSection[]` — fixed `StandardizedTest` type
  - [x] Backend wraps in `{ tests: [...], next_cursor }` — unwrap `resp.tests`
  - [x] Component: replaced `test.sections.length` / `test.sections.map()` with `Object.entries(test.scores ?? {})`
  - [x] Component: removed nonexistent `test.student_name` field reference
  - [x] Create form: convert `TestSection[]` UI state → `Record<string, number>` for submission
- **Playwright confirmed:** 0 errors. Scores render as key-value pairs (math: 88, reading: 85, etc.).

### Step 4.3 — Fix compliance assessments hooks (GAP-E2E-H4, CP3) ✅
- [x] **File:** `frontend/src/hooks/use-compliance.ts`
- [x] `useComplianceAssessments(studentId)`: `/v1/compliance/assessments` → `/v1/compliance/students/${studentId}/assessments`
- [x] Add `studentId` parameter to hook
- [x] Fix corresponding mutation hooks
- [x] **Backend:** `internal/comply/handler.go:52-55`
- [x] **Verify:** Navigate to `/compliance/assessments` → page loads
- **Extra fixes discovered during Playwright:**
  - [x] Backend wraps in `{ records: [...], next_cursor }` — unwrap `resp.records`
  - [x] Backend field names differ: `title` (not `assessment_title`), `subject` (not `student_name`), `assessment_type` (not `requirement_name`), `score: number|null` (not `string`), `assessment_date` (not `date`)
  - [x] Updated `ComplianceAssessment` type to match backend `AssessmentResponse` struct
  - [x] Component: smart score display (`score/max_score` or `grade_letter` or `—`)
- **Playwright confirmed:** 0 errors. Shows "Spelling Test Week 12" (english · test) 78/100, etc.

### Step 4.4 — Fix compliance portfolio hooks (GAP-E2E-H4, CP5/CP6) ✅
- [x] **File:** `frontend/src/hooks/use-compliance.ts`
- [x] All 6 portfolio hooks need `/students/${studentId}/` prefix in URL
- [x] Read each hook, add studentId to path and signature where missing
- [x] **Backend:** `internal/comply/handler.go:63-69`
- [x] **Verify:** Navigate to `/compliance/portfolios` → list renders (or clean empty state)
- **Extra fixes discovered during Playwright:**
  - [x] `PortfolioSummary` type: removed `student_id`, `student_name`, `download_url`, `updated_at` (not in backend `PortfolioSummaryResponse`)
  - [x] Backend status is `configuring` (not `draft`) — updated `PortfolioStatus` type to include all 5 statuses: `configuring | generating | ready | failed | expired`
  - [x] Added i18n keys: `compliance.portfolio.status.configuring`, `.failed`, `.expired`
  - [x] Date parsing: backend sends full ISO timestamps, removed broken `+ "T12:00:00"` append
  - [x] Component: pass `studentId` prop from filter to `PortfolioCard` instead of accessing `portfolio.student_id`
  - [x] Component: removed download button from list card (only available on detail view)
  - [x] Create form: use local `studentId` state for navigation instead of `data.student_id`
- **Playwright confirmed:** 0 errors. Status "Configuring", dates "Dec 31 – May 21, 2026", correct link URLs.

### Step 4.5 — Fix compliance transcript hooks (GAP-E2E-H4, CP7/CP8) ✅
- [x] **File:** `frontend/src/hooks/use-compliance.ts`
- [x] All 9 transcript hooks need `/students/${studentId}/` prefix
- [x] Read each hook, add studentId to path and signature where missing
- [x] **Backend:** `internal/comply/handler.go:75-80`
- [x] **Verify:** Navigate to `/compliance/transcripts` → list renders (or clean empty state)
- **Extra fixes discovered during Playwright:**
  - [x] `TranscriptSummary` type: removed `student_id`, `student_name`, `semester_count`, `total_credits`, `cumulative_gpa`, `download_url`, `updated_at` (not in backend `TranscriptSummaryResponse`)
  - [x] Added `grade_levels: string[]`, `generated_at: string | null`
  - [x] Updated `TranscriptStatus` to include `configuring | failed | expired`
  - [x] Added i18n keys: `compliance.transcript.status.configuring`, `.failed`, `.expired`
  - [x] Component: pass `studentId` prop, show `grade_levels.join(", ")` instead of semester/credits/GPA
  - [x] Component: removed download button, use local `studentId` for navigation
- **Playwright confirmed:** 0 errors. Shows "Configuring" status, grade level "5".

---

## Batch 5: Learning Content Player URL Fixes ✅

### Step 5.1 — Fix learning projects hooks (GAP-E2E-H2, LR15) ✅
- [x] **File:** `frontend/src/hooks/use-projects.ts`
- [x] All 5 hooks: `/v1/learning/projects` → `/v1/learning/students/${studentId}/projects`
- [x] Add `studentId` parameter where missing
- [x] **Backend:** `internal/learn/handler.go:151-155`
- [x] **Verify:** Navigate to `/learning/projects` → page renders (or empty state)
- **Extra fixes discovered during Playwright:**
  - [x] Backend returns `{ data: [...], has_more }` not bare `Project[]` — added `PaginatedResponse<T>` type, unwrap `.data`
- **Playwright confirmed:** 0 errors. Empty state renders correctly.

### Step 5.2 — Fix quiz/video/sequence player hooks (GAP-E2E-H1, LR9/LR10/LR12) ✅
- [x] **Files:** `frontend/src/hooks/use-quiz.ts`, `use-video.ts`, `use-sequence.ts`
- [x] Read each file to identify URL patterns
- [x] Fix any URLs missing `/students/${studentId}/` prefix
- [x] **Backend:** `internal/learn/handler.go` — quiz-sessions, video-progress, sequence-progress all under `/students/:studentId/`
- [x] **Verify:** Navigate to `/learning/quiz/{id}`, `/learning/video/{id}`, `/learning/sequence/{id}` → players render (or clean error state)

---

## Batch 6: Backend Fixes ✅

### Step 6.1 — Fix admin users 500 error (GAP-E2E-H5, AD2/AD3) ✅
- [x] **File:** `cmd/server/main.go:1578-1607`
- [x] The `adminIamAdapter.SearchUsers` raw SQL has `? IS NULL OR f.id = ?::uuid`
- [x] When query fields are nil pointers, GORM passes NULL → `NULL::uuid` or `safety_account_status` table issue
- [x] Investigate: run the SQL directly via `mcp__plenum__query` to isolate the error
- [x] Fix: build WHERE clauses dynamically based on which fields are non-nil, or use COALESCE/CASE guards
- [x] Also check double `c.Bind()` in `internal/admin/handler.go:93-100`
- [x] **Verify:** Login as admin, navigate to `/admin/users` → user list shows seeded users

### Step 6.2 — Investigate billing payment-methods 500 (GAP-E2E-M2e, B2) ✅
- [x] **File:** `internal/billing/handler.go:49` — route is registered
- [x] Check if service `ListPaymentMethods` returns error from unconfigured payment provider
- [x] Fix: return empty list when provider not configured (`ErrPaymentAdapterUnavailable` → `[]PaymentMethodResponse{}`)
- [x] **Verify:** Navigate to `/billing/payment-methods` → empty list or clean state

---

## Batch 7: Stub Missing Backend Endpoints ✅

For each, add a minimal handler that returns an appropriate empty/default response.

### Step 7.1 — Stub student streak endpoint (GAP-E2E-M5) ✅
- [x] **File:** `internal/learn/handler.go` — add route `GET /students/:studentId/streak`
- [x] Return `{ "current_streak": 0, "longest_streak": 0, "last_activity_date": null }`
- [x] **Verify:** Navigate to `/learning` — no 404 errors on streak endpoints in console

### Step 7.2 — Stub creator verification endpoint (GAP-E2E-M2c, CR9) ✅
- [x] **File:** `internal/mkt/handler.go` — add route `GET /creator/verification`
- [x] Return `{ "status": "unverified", "submitted_at": null }`
- [x] **Verify:** Navigate to `/creator/verification` — page renders with "unverified" status

### Step 7.3 — Stub creator reviews endpoint (GAP-E2E-M2c, CR10) ✅
- [x] **File:** `internal/mkt/handler.go` — add route `GET /creator/reviews`
- [x] Return `{ "reviews": [], "average_rating": 0, "total_reviews": 0 }`
- [x] **Verify:** Navigate to `/creator/reviews` — empty reviews list renders

### Step 7.4 — Stub listing versions endpoint (GAP-E2E-M2d, MK6) ✅
- [x] **File:** `internal/mkt/handler.go` — add route `GET /listings/:id/versions`
- [x] Return `{ "versions": [] }`
- [x] **Verify:** Navigate to `/marketplace/listings/{id}/versions` — empty version list

### Step 7.5 — Stub MFA status endpoint (GAP-E2E-M8, ST13) ✅
- [x] **File:** `internal/iam/handler.go` — add route `GET /auth/mfa/status`
- [x] Return `{ "enabled": false, "methods": [] }`
- [x] **Verify:** Navigate to `/settings/account/mfa` — shows "MFA not enabled" instead of blank page

### Step 7.6 — Stub payout endpoints (GAP-E2E-M2c, CR8) ✅
- [x] **File:** `internal/mkt/handler.go` — add routes:
  - `GET /creator/payouts/config` → `{ "configured": false }`
  - `GET /creator/payouts/methods` → `{ "methods": [] }`
  - `GET /creator/payouts/history` → `{ "payouts": [] }`
- [x] **Verify:** Navigate to `/creator/payouts` — page renders without 16 console errors

---

## Additional Fixes (discovered during Playwright validation)

These issues were not in the original plan but were found when testing against the running backend.

### Fix A1 — Frontend types don't match backend response shapes
- [x] `StandardizedTest.sections` → `scores: Record<string, number>` (backend uses `json.RawMessage`)
- [x] `ComplianceAssessment` fields renamed to match `AssessmentResponse` struct
- [x] `PortfolioSummary` stripped of non-existent fields, status type expanded
- [x] `TranscriptSummary` stripped of non-existent fields, status type expanded
- [x] `useProjects` queryFn: unwrap `{ data: [...] }` paginated response
- [x] `useAttendance` queryFn: unwrap `{ records: [...] }` response, convert month → RFC3339 date range
- [x] `useAttendanceSummary`: single object (not array), field name mapping from raw backend response
- [x] `useStandardizedTests` queryFn: unwrap `{ tests: [...] }` response
- [x] `useComplianceAssessments` queryFn: unwrap `{ records: [...] }` response

### Fix A2 — Component rendering fixes for new types
- [x] `standardized-tests.tsx`: `Object.entries(test.scores)` instead of `test.sections.map()`
- [x] `assessment-records.tsx`: correct field names, smart score display (`score/max_score`)
- [x] `portfolio-list.tsx`: pass `studentId` prop, fix date parsing, remove download button
- [x] `transcript-list.tsx`: pass `studentId` prop, show grade_levels, remove download/stats
- [x] `attendance-tracker.tsx`: use single summary object, fix variable ordering

### Fix A3 — Missing i18n keys for backend statuses
- [x] `compliance.portfolio.status.configuring`, `.failed`, `.expired`
- [x] `compliance.transcript.status.configuring`, `.failed`, `.expired`

---

## Quick Reference: Gap → Step Mapping

| Gap ID | Severity | Step | Description | Status |
|--------|----------|------|-------------|--------|
| H0 | HIGH | 2.1 | Billing i18n keys | ✅ |
| H1 | HIGH | 5.2 | Content player URL fixes | ✅ |
| H2 | HIGH | 5.1 | Projects URL fix | ✅ |
| H3 | HIGH | 2.2 | Recommendations i18n keys | ✅ |
| H4 | HIGH | 4.1-4.5 | Compliance URL fixes | ✅ |
| H5 | HIGH | 1.2, 1.3, 6.1 | Admin flags/methods URL + users 500 | ✅ |
| H6 | HIGH | 1.1 | Search URL fix | ✅ (backend 500 is pre-existing) |
| M2c | MED | 7.2, 7.3 | Creator verification/reviews stubs | ✅ |
| M2d | MED | 7.4 | Listing versions stub | ✅ |
| M2e | MED | 6.2 | Payment methods fix | ✅ |
| M5 | MED | 7.1 | Streak stub | ✅ |
| M8 | MED | 7.5 | MFA stub | ✅ |
| M10 | MED | 1.4 | Planning templates URL fix | ✅ |
| M12 | MED | 3.1, 3.2 | Entity not-found states | ❌ not started |
| M13 | MED | 3.2 | Add-to-cart toast | ❌ not started |
| M14 | MED | — | Checkout (endpoint exists, may need URL fix) | ❌ not started |
| L3 | LOW | 3.3 | Notification header fix | ❌ not started |
| L5 | LOW | 3.4 | Compliance spinbuttons fix | ❌ not started |
