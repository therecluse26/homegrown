# E2E Gap Fixes — Step-by-Step TODO

> Generated from `research/gaps/e2e_exploratory_2026-04-06.md`
> Each step is independently executable and Playwright-verifiable.

---

## Batch 1: Quick Frontend URL Fixes

These are 1-3 line changes per hook — highest ROI.

### Step 1.1 — Fix search URL (GAP-E2E-H6)
- [ ] **File:** `frontend/src/hooks/use-search.ts:94`
- [ ] Change `/v1/search/search?${buildSearchQuery(params!)}` → `/v1/search?${buildSearchQuery(params!)}`
- [ ] **Backend route:** `internal/search/handler.go:36` → `search.GET("", h.search)`
- [ ] **Verify:** Navigate to `/search`, type "Charlotte Mason", check Social/Marketplace/Learning tabs return results
- [ ] **Gaps closed:** H6 (all 3 search scopes)

### Step 1.2 — Fix admin feature flags URL (GAP-E2E-H5, AD5)
- [ ] **File:** `frontend/src/hooks/use-admin.ts`
- [ ] Line 401: `/v1/admin/feature-flags` → `/v1/admin/flags`
- [ ] Line 417: `/v1/admin/feature-flags/${id}` → `/v1/admin/flags/${id}`, `PUT` → `PATCH`
- [ ] Line 436: `/v1/admin/feature-flags` → `/v1/admin/flags`
- [ ] **Backend route:** `internal/admin/handler.go:39-43` → uses `/flags`, `/flags/:key`, PATCH
- [ ] **Verify:** Login as admin, navigate to `/admin/flags`, confirm flags list loads

### Step 1.3 — Fix admin methodologies URL (GAP-E2E-H5, AD7)
- [ ] **File:** `frontend/src/hooks/use-admin.ts`
- [ ] Line 466: `/v1/admin/methodology-configs` → `/v1/admin/methodologies`
- [ ] Line 473: `/v1/admin/methodology-configs/${slug}` → `/v1/admin/methodologies/${slug}`, `PUT` → `PATCH`
- [ ] **Backend route:** `internal/admin/handler.go:51-52` → uses `/methodologies`, PATCH
- [ ] **Verify:** Login as admin, navigate to `/admin/methodologies`, confirm list loads

### Step 1.4 — Fix planning templates URL (GAP-E2E-M10, PL6)
- [ ] **File:** `frontend/src/hooks/use-planning.ts`
- [ ] Line 376: `/v1/planning/schedule-templates` → `/v1/planning/templates`
- [ ] Line 385: `/v1/planning/schedule-templates` → `/v1/planning/templates`
- [ ] Line 405: `/v1/planning/schedule-templates/${templateId}/apply` → `/v1/planning/templates/${templateId}/apply`
- [ ] Line 418: `/v1/planning/schedule-templates/${templateId}` → `/v1/planning/templates/${templateId}`
- [ ] **Backend route:** `internal/plan/handler.go:46-50` → uses `/templates`
- [ ] **Verify:** Navigate to `/planning/templates`, confirm page loads without "Something went wrong"

---

## Batch 2: i18n Translation Keys

### Step 2.1 — Add billing subscription + invoice i18n keys (GAP-E2E-H0)
- [ ] **File:** `frontend/src/locales/en.json`
- [ ] Read `frontend/src/features/billing/subscription-management.tsx` for all `FormattedMessage id` and `intl.formatMessage` calls
- [ ] Read `frontend/src/features/billing/invoice-history.tsx` for the same
- [ ] Add all missing `billing.subscription.*` keys (~15 static + ~9 dynamic tier/status/interval keys)
- [ ] Add all missing `billing.invoice.*` keys (~8 static + ~8 dynamic type/status keys)
- [ ] **Verify:** Navigate to `/billing/subscription` and `/billing/invoices` — all text should be human-readable, zero MissingTranslationErrors in console

### Step 2.2 — Add recommendations page i18n keys (GAP-E2E-H3)
- [ ] **File:** `frontend/src/locales/en.json`
- [ ] Read `frontend/src/features/recommendations/recommendations-page.tsx` for all message IDs
- [ ] Add ~17 missing `recommendations.*` keys (title, description, filters, types, badges, dismiss, empty state)
- [ ] Keep existing 9 `recommendations.section.*` and `recommendations.card.*` keys
- [ ] **Verify:** Navigate to `/recommendations` — all UI chrome should be translated, zero MissingTranslationErrors

---

## Batch 3: Frontend Error States & UX

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

## Batch 4: Compliance URL Refactor (student-scoped routes)

This is the largest batch — all compliance hooks need `students/{studentId}` in the URL path.

### Step 4.1 — Fix compliance attendance hooks (GAP-E2E-H4, CP2)
- [ ] **File:** `frontend/src/hooks/use-compliance.ts`
- [ ] `useAttendance(studentId, month)` line 276: `/v1/compliance/attendance/${studentId}` → `/v1/compliance/students/${studentId}/attendance`
- [ ] `useAttendanceSummary()` line 288: `/v1/compliance/attendance/summary` → needs `studentId` param added: `/v1/compliance/students/${studentId}/attendance/summary`
- [ ] `useRecordAttendance()` line 346: `/v1/compliance/attendance` → `/v1/compliance/students/${studentId}/attendance`
- [ ] **Backend:** `internal/comply/handler.go:44-49`
- [ ] **Also update:** calling components to pass studentId where they don't already
- [ ] **Verify:** Navigate to `/compliance/attendance`, select a student → calendar loads without errors

### Step 4.2 — Fix compliance tests hooks (GAP-E2E-H4, CP4)
- [ ] **File:** `frontend/src/hooks/use-compliance.ts`
- [ ] `useStandardizedTests(studentId)` line 300: `/v1/compliance/tests` → `/v1/compliance/students/${studentId}/tests`
- [ ] `useCreateStandardizedTest()` mutation: same path fix
- [ ] **Backend:** `internal/comply/handler.go:58-61`
- [ ] **Verify:** Navigate to `/compliance/tests` → page loads without "Something went wrong"

### Step 4.3 — Fix compliance assessments hooks (GAP-E2E-H4, CP3)
- [ ] **File:** `frontend/src/hooks/use-compliance.ts`
- [ ] `useComplianceAssessments()` line 312: `/v1/compliance/assessments` → `/v1/compliance/students/${studentId}/assessments`
- [ ] Add `studentId` parameter to hook
- [ ] Fix corresponding mutation hooks
- [ ] **Backend:** `internal/comply/handler.go:52-55`
- [ ] **Verify:** Navigate to `/compliance/assessments` → page loads

### Step 4.4 — Fix compliance portfolio hooks (GAP-E2E-H4, CP5/CP6)
- [ ] **File:** `frontend/src/hooks/use-compliance.ts`
- [ ] All 6 portfolio hooks need `/students/${studentId}/` prefix in URL
- [ ] Read each hook, add studentId to path and signature where missing
- [ ] **Backend:** `internal/comply/handler.go:63-69`
- [ ] **Verify:** Navigate to `/compliance/portfolios` → list renders (or clean empty state)

### Step 4.5 — Fix compliance transcript hooks (GAP-E2E-H4, CP7/CP8)
- [ ] **File:** `frontend/src/hooks/use-compliance.ts`
- [ ] All 9 transcript hooks need `/students/${studentId}/` prefix
- [ ] Read each hook, add studentId to path and signature where missing
- [ ] **Backend:** `internal/comply/handler.go:75-80`
- [ ] **Verify:** Navigate to `/compliance/transcripts` → list renders (or clean empty state)

---

## Batch 5: Learning Content Player URL Fixes

### Step 5.1 — Fix learning projects hooks (GAP-E2E-H2, LR15)
- [ ] **File:** `frontend/src/hooks/use-projects.ts`
- [ ] All 5 hooks: `/v1/learning/projects` → `/v1/learning/students/${studentId}/projects`
- [ ] Add `studentId` parameter where missing
- [ ] **Backend:** `internal/learn/handler.go:151-155`
- [ ] **Verify:** Navigate to `/learning/projects` → page renders (or empty state)

### Step 5.2 — Fix quiz/video/sequence player hooks (GAP-E2E-H1, LR9/LR10/LR12)
- [ ] **Files:** `frontend/src/hooks/use-quiz.ts`, `use-video.ts`, `use-sequence.ts`
- [ ] Read each file to identify URL patterns
- [ ] Fix any URLs missing `/students/${studentId}/` prefix
- [ ] **Backend:** `internal/learn/handler.go` — quiz-sessions, video-progress, sequence-progress all under `/students/:studentId/`
- [ ] **Verify:** Navigate to `/learning/quiz/{id}`, `/learning/video/{id}`, `/learning/sequence/{id}` → players render (or clean error state)

---

## Batch 6: Backend Fixes

### Step 6.1 — Fix admin users 500 error (GAP-E2E-H5, AD2/AD3)
- [ ] **File:** `cmd/server/main.go:1578-1607`
- [ ] The `adminIamAdapter.SearchUsers` raw SQL has `? IS NULL OR f.id = ?::uuid`
- [ ] When query fields are nil pointers, GORM passes NULL → `NULL::uuid` or `safety_account_status` table issue
- [ ] Investigate: run the SQL directly via `mcp__plenum__query` to isolate the error
- [ ] Fix: build WHERE clauses dynamically based on which fields are non-nil, or use COALESCE/CASE guards
- [ ] Also check double `c.Bind()` in `internal/admin/handler.go:93-100`
- [ ] **Verify:** Login as admin, navigate to `/admin/users` → user list shows seeded users

### Step 6.2 — Investigate billing payment-methods 500 (GAP-E2E-M2e, B2)
- [ ] **File:** `internal/billing/handler.go:49` — route is registered
- [ ] Check if service `ListPaymentMethods` returns error from unconfigured payment provider
- [ ] Fix: return empty list when provider not configured
- [ ] **Verify:** Navigate to `/billing/payment-methods` → empty list or clean state

---

## Batch 7: Stub Missing Backend Endpoints

For each, add a minimal handler that returns an appropriate empty/default response.

### Step 7.1 — Stub student streak endpoint (GAP-E2E-M5)
- [ ] **File:** `internal/learn/handler.go` — add route `GET /students/:studentId/streak`
- [ ] Return `{ "current_streak": 0, "longest_streak": 0, "last_activity_date": null }`
- [ ] **Verify:** Navigate to `/learning` — no 404 errors on streak endpoints in console

### Step 7.2 — Stub creator verification endpoint (GAP-E2E-M2c, CR9)
- [ ] **File:** `internal/mkt/handler.go` — add route `GET /creator/verification`
- [ ] Return `{ "status": "unverified", "submitted_at": null }`
- [ ] **Verify:** Navigate to `/creator/verification` — page renders with "unverified" status

### Step 7.3 — Stub creator reviews endpoint (GAP-E2E-M2c, CR10)
- [ ] **File:** `internal/mkt/handler.go` — add route `GET /creator/reviews`
- [ ] Return `{ "reviews": [], "average_rating": 0, "total_reviews": 0 }`
- [ ] **Verify:** Navigate to `/creator/reviews` — empty reviews list renders

### Step 7.4 — Stub listing versions endpoint (GAP-E2E-M2d, MK6)
- [ ] **File:** `internal/mkt/handler.go` — add route `GET /listings/:id/versions`
- [ ] Return `{ "versions": [] }`
- [ ] **Verify:** Navigate to `/marketplace/listings/{id}/versions` — empty version list

### Step 7.5 — Stub MFA status endpoint (GAP-E2E-M8, ST13)
- [ ] **File:** `internal/iam/handler.go` — add route `GET /auth/mfa/status`
- [ ] Return `{ "enabled": false, "methods": [] }`
- [ ] **Verify:** Navigate to `/settings/account/mfa` — shows "MFA not enabled" instead of blank page

### Step 7.6 — Stub payout endpoints (GAP-E2E-M2c, CR8)
- [ ] **File:** `internal/mkt/handler.go` — add routes:
  - `GET /creator/payouts/config` → `{ "configured": false }`
  - `GET /creator/payouts/methods` → `{ "methods": [] }`
  - `GET /creator/payouts/history` → `{ "payouts": [] }`
- [ ] **Verify:** Navigate to `/creator/payouts` — page renders without 16 console errors

---

## Quick Reference: Gap → Step Mapping

| Gap ID | Severity | Step | Description |
|--------|----------|------|-------------|
| H0 | HIGH | 2.1 | Billing i18n keys |
| H1 | HIGH | 5.2 | Content player URL fixes |
| H2 | HIGH | 5.1 | Projects URL fix |
| H3 | HIGH | 2.2 | Recommendations i18n keys |
| H4 | HIGH | 4.1-4.5 | Compliance URL fixes |
| H5 | HIGH | 1.2, 1.3, 6.1 | Admin flags/methods URL + users 500 |
| H6 | HIGH | 1.1 | Search URL fix |
| M2c | MED | 7.2, 7.3 | Creator verification/reviews stubs |
| M2d | MED | 7.4 | Listing versions stub |
| M2e | MED | 6.2 | Payment methods fix |
| M5 | MED | 7.1 | Streak stub |
| M8 | MED | 7.5 | MFA stub |
| M10 | MED | 1.4 | Planning templates URL fix |
| M12 | MED | 3.1, 3.2 | Entity not-found states |
| M13 | MED | 3.2 | Add-to-cart toast |
| M14 | MED | — | Checkout (endpoint exists, may need URL fix) |
| L3 | LOW | 3.3 | Notification header fix |
| L5 | LOW | 3.4 | Compliance spinbuttons fix |
