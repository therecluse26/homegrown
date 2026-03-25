# TODO: Frontend Implementation Roadmap

> **Scope**: Full React SPA — from empty scaffolding to production-ready.
> **Current state**: React 19 + Vite + TanStack Query wired. No CSS, no routes,
> no components, no features. Only `api/client.ts` and `api/generated/schema.ts` exist.

---

## Dependency Graph

```
Phase 1 ─── CSS Infrastructure
   │
Phase 2 ─── Typography, Base Styles, Token Completion
   │
Phase 3 ─── Shared UI Component Library
   │
Phase 4 ─── Routing, Layouts & Auth Context
   │
Phase 5 ─── Auth Flows (Ory Kratos)
   │
Phase 6 ─── Onboarding Wizard
   │
Phase 7 ─── Parent Dashboard & Family Management
   │
   ├──────────────────┬──────────────────┐
Phase 8               Phase 9            Phase 10
Learning Tools        Social, Mkt,       Compliance, Admin,
& Progress            Discovery          Polish & QA
```

Phases 8–10 may proceed in parallel once Phase 7 is complete.

---

## Cross-Cutting Concerns

These apply to **every** phase and are not listed per-item:

| Concern | Rule | Ref |
|---------|------|-----|
| No `any` | Zero `any` in all TypeScript | CODING_STANDARDS §3 |
| Generated types only | All API types from `src/api/generated/schema.ts` | CODING_STANDARDS §3 |
| Token classes only | No hardcoded hex, no Tailwind default palette | DESIGN_TOKENS §2, CODING_STANDARDS §3 |
| No inline styles | Tailwind classes exclusively | CODING_STANDARDS §3 |
| No 1px borders | Use tonal surface shifts for sectioning | DESIGN §4.1 |
| Accessibility | WCAG 2.1 AA, visible labels, focus management, 44px tap targets | CODING_STANDARDS §3.8 |
| No methodology branching | Config lookup via `useMethodologyContext()` | CODING_STANDARDS §3.7 |
| Custom hooks for queries | No TanStack Query in components directly | CODING_STANDARDS §3.5 |
| File naming | kebab-case for files, PascalCase for components | CODING_STANDARDS §3.6 |
| Privacy | No GPS, no PII in logs, COPPA compliance | SPEC §7, ARCHITECTURE §8 |

---

## API Type Generation & Consumption

### Generation Pipeline

Before starting **any phase** that consumes API data, run:

```bash
make full-generate
```

This regenerates `frontend/src/api/generated/schema.ts` from Go source. The pipeline:
`swag init` → `openapi/swagger.json` → `swagger2openapi` → `openapi/openapi3.json` →
`openapi-typescript` → `schema.ts`.

**Always run this first** — even if you think types are current. Backend changes from
prior sessions may not be reflected yet.

### Type Consumption Pattern

The generated file exports two key interfaces: `paths` (route definitions) and
`components` (schemas). All API types are accessed via:

```typescript
import type { components } from "@/api/generated/schema";

// Create a local type alias for ergonomics
type CurrentUser = components["schemas"]["iam.CurrentUserResponse"];
type Student     = components["schemas"]["iam.StudentResponse"];
type ErrorResp   = components["schemas"]["shared.ErrorResponse"];
```

Schema names follow the Go package-qualified convention: `{package}.{TypeName}`.

### Hook Pattern

Every API call goes through `apiClient<T>` with the generated type as the generic:

```typescript
import { apiClient } from "@/api/client";
import type { components } from "@/api/generated/schema";

type FamilyProfile = components["schemas"]["iam.FamilyProfileResponse"];

export function useFamilyProfile() {
  return useQuery({
    queryKey: ["family", "profile"],
    queryFn: () => apiClient<FamilyProfile>("/v1/families/profile"),
  });
}
```

**Rules** (from CODING_STANDARDS §3.1, §3.4, §6.2):
- NEVER hand-write API request/response types — use generated types only
- NEVER import `fetch`/`axios` in hooks or components — use `apiClient`
- NEVER hand-edit files in `src/api/generated/`
- Create type aliases at the top of hook files for readability

---

## Phase 1: CSS Infrastructure & Tailwind v4 Wiring

**Goal**: Get Tailwind v4 running with the Vite plugin so that token-based utility
classes are available for all subsequent phases.

**Why first**: Every component, layout, and feature depends on CSS tokens.
Nothing visual can be built without this.

### Package & Config Changes

- [ ] `npm install @tailwindcss/vite lucide-react` — add missing deps (DESIGN_TOKENS §1.3, §13.1)
- [ ] `npm uninstall autoprefixer postcss` — unnecessary with Tailwind v4 (DESIGN_TOKENS §1.3)
- [ ] Update `frontend/vite.config.ts` — add `import tailwindcss from "@tailwindcss/vite"` and include `tailwindcss()` in `plugins[]` (DESIGN_TOKENS §1.3)
- [ ] Remove `postcss.config.*` if present — Tailwind v4 needs no PostCSS config

### CSS File Structure

Create `frontend/src/styles/` with the following files (DESIGN_TOKENS §1.2):

- [ ] `tokens.css` — `@theme` block with all CSS custom properties:
  - Surface colors (7 tiers), primary (3), secondary (4), tertiary (2), feedback (6), text/outline (4), inverse (2) (DESIGN_TOKENS §3)
  - Font families: `--font-display`, `--font-body` (DESIGN_TOKENS §4)
  - Spacing semantic aliases (6 tokens) (DESIGN_TOKENS §6)
  - Border radii (7 tokens) (DESIGN_TOKENS §7)
  - Shadows (4 ambient + ghost-border) (DESIGN_TOKENS §8)
  - Z-index scale (8 tiers) (DESIGN_TOKENS §9)
  - Animation durations (4) + easings (3) (DESIGN_TOKENS §10)
  - Breakpoint: `--breakpoint-3xl: 1920px` (DESIGN_TOKENS §15)
  - `:root` block for non-theme tokens (gradients, focus-ring) (DESIGN_TOKENS §11)
- [ ] `base.css` — `@font-face` declarations, CSS reset additions, `:focus-visible` ring, `prefers-reduced-motion` (DESIGN_TOKENS §4, §10.2)
- [ ] `components.css` — type-scale composite classes (`.text-display-lg` through `.text-label-sm`, 15 steps) (DESIGN_TOKENS §5)
- [ ] `utilities.css` — z-index utilities (`.z-base` through `.z-tooltip`), `.touch-target` class, parent/student Tailwind custom variants (`parent:`, `student:`) (DESIGN_TOKENS §9, §12, §15.1)
- [ ] `print.css` — `@media print` token overrides + hide rules (DESIGN_TOKENS §16)
- [ ] `app.css` — entry point: `@import "tailwindcss"` then import all above files (DESIGN_TOKENS §1.2)

### Wiring

- [ ] Add `import "./styles/app.css"` to `frontend/src/main.tsx` (before QueryClientProvider)
- [ ] Verify: Vite dev server starts without CSS errors
- [ ] Verify: Tailwind utility classes (e.g. `bg-primary`, `text-on-surface`) resolve correctly in a test `<div>`

### References
- DESIGN_TOKENS §1–§3, §6–§11, §15–§17
- ARCHITECTURE §11.1 (Tailwind v4 decision)

---

## Phase 2: Typography, Base Styles & Design Token Completion

**Goal**: Self-host fonts, wire up the type scale, and complete all token-dependent
base styles so the visual foundation is pixel-accurate.

**Why second**: Typography + colors define the entire "Curated Hearth" aesthetic.
Components built in Phase 3 inherit these automatically.

### Font Hosting

- [ ] Download WOFF2 files for **Plus Jakarta Sans** (SemiBold 600, Bold 700) and **Manrope** (Regular 400, Medium 500, SemiBold 600) — COPPA: no Google Fonts CDN (DESIGN_TOKENS §4.1, SPEC §7)
- [ ] Create `frontend/public/fonts/` directory and place WOFF2 files there
- [ ] Add font preload hints to `frontend/index.html`:
  ```html
  <link rel="preload" href="/fonts/PlusJakartaSans-SemiBold.woff2" as="font" type="font/woff2" crossorigin />
  <link rel="preload" href="/fonts/Manrope-Regular.woff2" as="font" type="font/woff2" crossorigin />
  ```
- [ ] Write `@font-face` rules in `base.css` with `font-display: swap` (DESIGN_TOKENS §4.2)

### Type Scale Verification

- [ ] Confirm all 15 type-scale composite classes render correctly (DESIGN_TOKENS §5):
  - Display: `display-lg` (3.5rem), `display-md` (2.8125rem), `display-sm` (2.25rem)
  - Headline: `headline-lg` (2rem), `headline-md` (1.75rem), `headline-sm` (1.5rem)
  - Title: `title-lg` (1.375rem), `title-md` (1rem), `title-sm` (0.875rem)
  - Body: `body-lg` (1rem/1.6), `body-md` (0.875rem/1.55), `body-sm` (0.75rem)
  - Label: `label-lg` (0.875rem), `label-md` (0.75rem), `label-sm` (0.6875rem)
- [ ] Verify display/headline fonts use Plus Jakarta Sans (SemiBold/Bold)
- [ ] Verify body/label fonts use Manrope (Regular/Medium/SemiBold)

### Base Styles

- [ ] Set `body` background to `surface` (#faf9f5), text color to `on-surface` (#1b1c1a) (DESIGN_TOKENS §3.1, DESIGN §3.1)
- [ ] Configure `::selection` style (primary-container bg, on-primary text)
- [ ] Implement `:focus-visible` ring: 2px solid `focus-ring` (#0c5252), 2px offset (DESIGN_TOKENS §10.3)
- [ ] Implement `prefers-reduced-motion: reduce` — collapse all transitions to 0.01ms (DESIGN_TOKENS §10.2)

### Verification

- [ ] Verify: `npm run type-check` passes
- [ ] Verify: Vite dev server renders correct fonts and colors on a test page
- [ ] Verify: all token utilities available in Tailwind IntelliSense

### References
- DESIGN_TOKENS §3–§5, §10
- DESIGN §3.1–§3.4 (Curated Hearth aesthetic)

---

## Phase 3: Shared UI Component Library

**Goal**: Build the reusable primitive components that every feature will compose.
All components use token classes exclusively.

**Why third**: Features cannot be built without buttons, inputs, cards, modals.
Building these first prevents duplication and ensures consistency.

### Directory Setup

- [ ] Establish `frontend/src/components/ui/` structure
- [ ] Establish `frontend/src/components/common/` structure
- [ ] Create barrel exports (`index.ts`) for each directory

### Core Primitives (`components/ui/`)

- [ ] `button.tsx` — 3 variants: primary (solid primary bg), secondary (secondary-container fill), tertiary (transparent, hover surface-container-low). All use `radius-button` (0.75rem). Min tap target 44px on mobile. (DESIGN §5.1, DESIGN_TOKENS §7)
- [ ] `input.tsx` — background `surface-container-highest`, no border. Focus: ghost border via `primary`. Error: `error-container` bg + `error` text. Visible `<label>` required. (DESIGN §5.3, CODING_STANDARDS §3.8)
- [ ] `textarea.tsx` — same styling rules as input, auto-resize optional
- [ ] `select.tsx` — custom select with accessible keyboard nav, token styling
- [ ] `checkbox.tsx` — accessible checkbox with `primary` accent
- [ ] `radio.tsx` — accessible radio group with `primary` accent
- [ ] `card.tsx` — accepts `context` prop for parent (`radius-lg`) vs student (`radius-xl`). Background `surface-container-lowest` (#fff) for "lifted" cards. Shadow `ambient-sm`. (DESIGN §4.2, DESIGN_TOKENS §7–§8)
- [ ] `badge.tsx` — pill shape (`radius-full`), methodology-aware coloring
- [ ] `avatar.tsx` — circular, fallback initials, sizes xs–xl
- [ ] `modal.tsx` — focus trap, return focus on close, overlay at `z-modal`, `Escape` to close. (CODING_STANDARDS §3.8, DESIGN_TOKENS §9)
- [ ] `toast.tsx` — auto-dismiss, `z-notification`, `aria-live="polite"`. Success/error/warning variants using feedback tokens. (DESIGN_TOKENS §3.5, §9)
- [ ] `spinner.tsx` — loading indicator with `primary` color
- [ ] `skeleton.tsx` — content placeholder with pulse animation
- [ ] `icon.tsx` — Lucide icon wrapper with standard sizes (xs=12, sm=16, md=20, lg=24, xl=32, 2xl=48) (DESIGN_TOKENS §13)
- [ ] `tooltip.tsx` — `z-tooltip`, accessible (shows on focus), delay on hover
- [ ] `dropdown-menu.tsx` — `z-popover`, keyboard navigable, focus management
- [ ] `tabs.tsx` — accessible tab panel with `aria-selected`, keyboard arrow nav
- [ ] `progress-bar.tsx` — progress ribbon: `tertiary-fixed` bg, `primary` fill (DESIGN §5.4)
- [ ] `empty-state.tsx` — illustration slot + message + CTA button

### Common Components (`components/common/`)

- [ ] `skip-link.tsx` — "Skip to main content" as first focusable element on every page (CODING_STANDARDS §3.8)
- [ ] `methodology-badge.tsx` — methodology chip with config-driven label (not hardcoded)
- [ ] `user-avatar.tsx` — wraps avatar with user/student data
- [ ] `tier-gate.tsx` — shows upgrade prompt when free user hits premium feature (SPEC §10)
- [ ] `error-boundary.tsx` — React error boundary with friendly fallback UI
- [ ] `page-title.tsx` — sets `document.title` + renders `<h1>` for focus target on route transitions

### Form Utilities

- [ ] `form-field.tsx` — wraps input + label + error message with consistent spacing
- [ ] `file-upload.tsx` — drag-and-drop zone, file type + size validation (client-side), progress indicator. Extension validation pre-upload, magic byte validation happens server-side. (SPEC §9, CODING_STANDARDS §3.9)

### Verification

- [ ] Verify: all components render correctly in isolation (consider Storybook or a dev route)
- [ ] Verify: `npm run type-check` passes
- [ ] Verify: tab navigation works through all interactive components
- [ ] Verify: screen reader announces all components correctly

### References
- DESIGN §4–§5 (component visual spec)
- DESIGN_TOKENS §3–§13 (token values)
- CODING_STANDARDS §3.8 (accessibility rules)

---

## Phase 4: Routing, Layouts & Auth Context

**Goal**: Wire React Router v7, build the two shell layouts (parent + student),
create auth/methodology contexts, and protect routes.

**Why fourth**: Everything after this phase is a "feature page" that lives inside
a layout and depends on auth state. This is the app skeleton.

### Project Structure Refactor

- [ ] Create `frontend/src/hooks/` directory
- [ ] Create `frontend/src/lib/` directory
- [ ] Move `src/query-client.ts` → `src/lib/query-client.ts` (update import in `main.tsx`)
- [ ] Create `frontend/src/types/` directory with `index.ts` for shared frontend types
- [ ] Create `frontend/src/components/layout/` directory

### Auth Context & Hook

- [ ] `hooks/use-auth.ts` — custom hook wrapping TanStack Query for `GET /auth/me`:
  - Returns `{ user, isLoading, isAuthenticated, isParent, isPrimaryParent, tier, coppaStatus }`
  - Uses `CurrentUserResponse` from generated types
  - Handles 401 (not logged in) gracefully — sets `isAuthenticated: false`
  - Query key: `["auth", "me"]`
- [ ] `features/auth/auth-provider.tsx` — `AuthContext` provider wrapping the app, uses `use-auth` internally (ARCHITECTURE §11.2)

### Methodology Context & Hook

- [ ] `hooks/use-methodology.ts` — wraps TanStack Query for `GET /families/tools` and methodology config:
  - Returns tools, terminology labels, active methodology slug
  - Depends on auth (only fetches when authenticated)
  - Query key: `["family", "tools"]`
- [ ] `features/auth/methodology-provider.tsx` — `MethodologyContext` provider (ARCHITECTURE §11.2)

### Layout Components (`components/layout/`)

- [ ] `app-shell.tsx` — main authenticated layout:
  - Sidebar navigation (desktop) / bottom nav (mobile)
  - Header with search bar, notification bell, user menu
  - `<main>` content area with `data-context="parent"` attribute
  - Skip-link as first child
  - Responsive: sidebar collapses below `lg` breakpoint
  - Floating nav: `surface-container-low` at 80% opacity + `backdrop-blur: 20px` (DESIGN §4.4)
- [ ] `student-shell.tsx` — supervised student layout:
  - Simplified nav (no social, no marketplace, no settings)
  - `data-context="student"` on `<main>` (enables `student:` variant)
  - Larger tap targets, more rounded corners (`radius-xl`)
  - Back-to-parent button always visible
- [ ] `auth-layout.tsx` — unauthenticated layout for login/register/recovery pages (centered card, no sidebar)
- [ ] `onboarding-layout.tsx` — minimal layout for the onboarding wizard (progress indicator, no full nav)

### Route Guards

- [ ] `components/layout/protected-route.tsx` — redirects to `/auth/login` if not authenticated
- [ ] `components/layout/onboarding-guard.tsx` — redirects to `/onboarding` if authenticated but onboarding not complete (check `WizardProgressResponse.status !== "completed"` and `!== "skipped"`)
- [ ] `components/layout/admin-guard.tsx` — redirects to `/` if not admin (SPEC §16)
- [ ] `components/layout/student-guard.tsx` — validates active student session

### Router Setup

- [ ] `src/routes.tsx` — full route tree using React Router v7 with lazy loading:
  ```
  / (ProtectedRoute + OnboardingGuard + AppShell)
    index → Feed
    /friends → FriendsList
    /messages → DirectMessages
    /messages/:conversationId → Conversation
    /groups → GroupsList
    /groups/:groupId → GroupDetail
    /events → EventsList
    /learning → LearningDashboard
    /learning/activities → ActivityLog
    /learning/journals → JournalList
    /learning/journals/new → JournalEditor
    /learning/reading-lists → ReadingLists
    /learning/progress/:studentId → ProgressView
    /learning/quiz/:sessionId → QuizPlayer
    /learning/video/:videoId → VideoPlayer
    /learning/read/:contentId → ContentViewer
    /learning/sequence/:progressId → SequenceView
    /marketplace → MarketplaceBrowse
    /marketplace/listings/:id → ListingDetail
    /marketplace/cart → Cart
    /marketplace/purchases → PurchaseHistory
    /creator → CreatorDashboard
    /creator/listings/new → CreateListing
    /creator/listings/:id/edit → EditListing
    /creator/quiz-builder → QuizBuilder
    /creator/quiz-builder/:id → QuizBuilder
    /creator/sequence-builder → SequenceBuilder
    /creator/sequence-builder/:id → SequenceBuilder
    /settings → FamilySettings
    /settings/notifications → NotificationPrefs
    /settings/subscription → SubscriptionManager
    /search → SearchResults
    /family/:familyId → FamilyProfile
    /calendar → CalendarView
  /auth (AuthLayout)
    /auth/login → Login
    /auth/register → Register
    /auth/recovery → AccountRecovery
    /auth/verification → EmailVerification
  /onboarding (ProtectedRoute + OnboardingLayout)
    index → OnboardingWizard
  /student (ProtectedRoute + StudentGuard + StudentShell)
    index → StudentDashboard
    /student/quiz/:sessionId → StudentQuiz
    /student/video/:videoId → StudentVideo
    /student/read/:contentId → StudentReader
    /student/sequence/:progressId → StudentSequence
  ```
  (ARCHITECTURE §11.3)
- [ ] Update `src/App.tsx` — replace stub with `RouterProvider` + providers (Auth → Methodology → Router)
- [ ] Update `src/main.tsx` — ensure provider ordering: `StrictMode → QueryClientProvider → App`

### Route Transition Accessibility

- [ ] On every route change, move focus to the page's `<h1>` or main content region (CODING_STANDARDS §3.8)
- [ ] Announce page title changes to screen readers via `aria-live` or `document.title` update

### Verification

- [ ] Verify: unauthenticated user sees `/auth/login`
- [ ] Verify: authenticated user with incomplete onboarding redirects to `/onboarding`
- [ ] Verify: authenticated user with complete onboarding sees AppShell at `/`
- [ ] Verify: all routes lazy-load correctly (check network tab)
- [ ] Verify: `npm run type-check` passes
- [ ] Verify: keyboard navigation through sidebar/nav links works

### References
- ARCHITECTURE §11.2–§11.3 (auth strategy, route table)
- CODING_STANDARDS §3.5 (TanStack Query rules)
- CODING_STANDARDS §3.8 (accessibility)
- DESIGN §4.4 (floating nav)

---

## Phase 5: Auth Flows (Ory Kratos Integration)

**Goal**: Implement login, registration, account recovery, and email verification
screens using Ory Kratos Browser API. Wire COPPA consent flow.

**Why fifth**: Users must be able to authenticate before any feature is usable.
Depends on Phase 4 auth context and layout.

### Kratos Integration Utilities

- [ ] `lib/kratos.ts` — helper functions for Kratos Browser API:
  - `initLoginFlow()` — fetches login flow from Kratos
  - `initRegistrationFlow()` — fetches registration flow
  - `initRecoveryFlow()` — fetches recovery flow
  - `initVerificationFlow()` — fetches verification flow
  - `submitFlow(flowId, body)` — submits a flow to Kratos
  - Error mapping: Kratos validation errors → form field errors
  - CSRF token handling
  - (ARCHITECTURE §11.2)

### Auth Pages (`features/auth/`)

- [ ] `login.tsx` — email/password form + OAuth buttons (Google, Facebook, Apple). Error display for invalid credentials. Link to register + recovery. (SPEC §1, ARCHITECTURE §11.2)
- [ ] `register.tsx` — email/password + optional OAuth. On success: Kratos webhook triggers `POST /hooks/kratos/post-registration` → family+parent created automatically. Redirect to `/onboarding`. (SPEC §1, ARCHITECTURE §11.2)
- [ ] `account-recovery.tsx` — email input for password reset link. Success confirmation message.
- [ ] `email-verification.tsx` — handles verification token from URL. Shows success/error state.

### COPPA Consent Flow

- [ ] `hooks/use-consent.ts` — wraps `GET /families/consent` + `POST /families/consent`:
  - Returns `{ consentStatus, acknowledge, provideConsent }`
  - Query key: `["family", "consent"]`
- [ ] `features/auth/coppa-consent.tsx` — consent gate component:
  - Shown after registration before adding students
  - Status flow: `registered → noticed → consented` (SPEC §7.3)
  - Must be completed before any student can be created
  - Clear, parent-friendly language explaining data collection

### Verification

- [ ] Verify: login flow works end-to-end with Kratos (or mock for dev)
- [ ] Verify: registration creates family + parent via webhook
- [ ] Verify: COPPA consent blocks student creation until consented
- [ ] Verify: recovery email flow works
- [ ] Verify: OAuth buttons present and functional
- [ ] Verify: `npm run type-check` passes

### References
- ARCHITECTURE §11.2 (Kratos integration)
- SPEC §1 (auth requirements)
- SPEC §7 (COPPA compliance)
- Domain spec: `specs/domains/01-iam.md`

---

## Phase 6: Onboarding Wizard

**Goal**: Build the 4-step onboarding wizard that new families complete after
registration. This is the first real "feature" and validates the full stack.

**Why sixth**: First authenticated feature. Natural next step after login.
Tests the entire data flow: auth → API → TanStack Query → UI.

### Onboarding Hooks

- [ ] `hooks/use-onboarding.ts` — wraps onboarding API endpoints:
  - `useOnboardingProgress()` — `GET /onboarding/progress` (query key: `["onboarding", "progress"]`)
  - `useUpdateFamilyProfile()` — `PATCH /onboarding/family-profile`
  - `useAddChild()` / `useRemoveChild()` — `POST` / `DELETE /onboarding/children`
  - `useUpdateMethodology()` — `PATCH /onboarding/methodology`
  - `useImportQuiz()` — `POST /onboarding/methodology/import-quiz`
  - `useOnboardingRoadmap()` — `GET /onboarding/roadmap`
  - `useOnboardingRecommendations()` — `GET /onboarding/recommendations`
  - `useOnboardingCommunity()` — `GET /onboarding/community`
  - `useCompleteOnboarding()` — `POST /onboarding/complete`
  - `useSkipOnboarding()` — `POST /onboarding/skip`

### Wizard UI (`features/onboarding/`)

- [ ] `onboarding-wizard.tsx` — wizard container:
  - Progress indicator showing 4 steps with current/completed/upcoming states
  - Step navigation (back/next) with validation
  - Skip button always available (`POST /onboarding/skip`)
  - Reads `WizardProgressResponse.current_step` and `completed_steps[]` to determine state
  - Step enum: `family_profile → children → methodology → roadmap_review`
- [ ] `steps/family-profile-step.tsx` — family name, state selection, location region
  - Validate required fields before allowing next
- [ ] `steps/children-step.tsx` — add student profiles:
  - Display name, birth year, grade level
  - Add/remove students dynamically
  - COPPA consent must be complete before this step (gate check)
  - Optional step — can proceed with zero students
- [ ] `steps/methodology-step.tsx` — three methodology paths:
  - **Quiz-informed**: Import results from pre-registration quiz via `share_id`, or link to take quiz
  - **Exploration**: Browse methodology cards (GET `/methodologies`), drill into detail (GET `/methodologies/{slug}`), select one
  - **Skip**: Proceed with no methodology selected
  - Display methodology tools preview for selected methodology
- [ ] `steps/roadmap-review-step.tsx` — personalized roadmap:
  - GET `/onboarding/roadmap` — age-adapted recommendations (5 age brackets)
  - GET `/onboarding/recommendations` — starter resources
  - GET `/onboarding/community` — local community info
  - "Complete" button → `POST /onboarding/complete` → redirect to `/`

### Methodology Explorer (shared — also used in settings)

- [ ] `hooks/use-methodologies.ts` — wraps `GET /methodologies` and `GET /methodologies/{slug}`:
  - `useMethodologyList()` — query key: `["methodologies"]`
  - `useMethodologyDetail(slug)` — query key: `["methodologies", slug]`
  - `useMethodologyTools(slug)` — query key: `["methodologies", slug, "tools"]`
- [ ] `components/common/methodology-card.tsx` — methodology summary card for browsing

### Verification

- [ ] Verify: wizard renders correct step based on `current_step`
- [ ] Verify: completed steps show checkmarks, allow revisiting
- [ ] Verify: skip onboarding redirects to `/` with `status: "skipped"`
- [ ] Verify: methodology import from quiz works
- [ ] Verify: roadmap displays age-appropriate recommendations
- [ ] Verify: `npm run type-check` passes

### References
- SPEC §4 (onboarding requirements)
- Domain spec: `specs/domains/04-onboard.md`
- ARCHITECTURE §11.3 (onboarding routes)

---

## Phase 7: Parent Dashboard & Family Management

**Goal**: Build the settings pages and family management features that parents
use daily. This establishes the "home base" experience.

**Why seventh**: After onboarding, parents need to manage their family. These
pages exercise CRUD patterns that all subsequent features follow.

### Family Management Hooks

- [ ] `hooks/use-family.ts` — wraps family API endpoints:
  - `useFamilyProfile()` — `GET /families/profile` (query key: `["family", "profile"]`)
  - `useUpdateFamilyProfile()` — `PATCH /families/profile`
  - `useStudents()` — `GET /families/students` (query key: `["family", "students"]`)
  - `useCreateStudent()` — `POST /families/students`
  - `useUpdateStudent(id)` — `PATCH /families/students/{id}`
  - `useDeleteStudent(id)` — `DELETE /families/students/{id}`
  - `useFamilyTools()` — `GET /families/tools` (query key: `["family", "tools"]`)
  - `useStudentTools(id)` — `GET /families/students/{id}/tools` (query key: `["family", "students", id, "tools"]`)

### Settings Pages (`features/settings/`)

- [ ] `family-settings.tsx` — main settings page:
  - Edit family display name, state, location region
  - Change primary methodology (PATCH `/families/methodology`)
  - Manage secondary methodology slugs
  - View subscription tier
- [ ] `student-management.tsx` — student CRUD embedded in settings:
  - List students with edit/delete
  - Add student form (requires COPPA consent)
  - Per-student methodology override
  - Per-student tool display
- [ ] `notification-prefs.tsx` — notification preference management:
  - Per-type per-channel toggles
  - Digest frequency: immediate/daily/weekly/off
  - System-critical notifications shown as non-toggleable
  - (SPEC §8, domain spec `specs/domains/08-notify.md`)
- [ ] `subscription-manager.tsx` — subscription management:
  - Current plan display
  - Upgrade/downgrade flow
  - Billing cycle info
  - Payment method management
  - (SPEC §10, domain spec `specs/domains/10-billing.md`)

### Notification Center

- [ ] `hooks/use-notifications.ts` — notification queries + mutations:
  - `useNotifications()` — paginated notification list
  - `useUnreadCount()` — unread notification count (for bell badge)
  - `useMarkRead(id)` — mark single notification read
  - `useMarkAllRead()` — mark all read
- [ ] `components/layout/notification-bell.tsx` — bell icon in header with unread count badge
- [ ] `features/settings/notification-center.tsx` — dropdown/panel listing recent notifications

### API Schema Prerequisite

- [ ] Run `make full-generate` to pull in notification + billing + social endpoints (they exist in Go backend but are not yet in `schema.ts`)

### Verification

- [ ] Verify: family profile edits persist and reflect in UI
- [ ] Verify: student CRUD works, COPPA gate enforced
- [ ] Verify: methodology change updates tools across the app
- [ ] Verify: notification preferences save correctly
- [ ] Verify: `npm run type-check` passes

### References
- SPEC §1.4 (family management), §8 (notifications), §10 (billing)
- Domain specs: `specs/domains/01-iam.md`, `specs/domains/08-notify.md`, `specs/domains/10-billing.md`

---

## Phase 8: Learning Tools & Progress

**Goal**: Build the full learning domain — activity logging, journals, reading
lists, progress views, quiz player, video player, content viewer, and sequence
engine. This is the largest feature surface.

**Why here**: Core value proposition. Depends on auth, methodology context,
and family management from prior phases.

### Prerequisites

- [ ] Run `make full-generate` if not already done — learning endpoints must be in `schema.ts`

### Learning Hooks (`hooks/`)

- [ ] `use-activities.ts` — activity definitions + activity log CRUD:
  - `useActivityDefs()`, `useCreateActivityDef()`, `useLogActivity()`, etc.
  - Query keys: `["learning", "activity-defs"]`, `["learning", "activity-log", filters]`
- [ ] `use-journals.ts` — journal entry CRUD:
  - `useJournalEntries(filters)`, `useCreateJournalEntry()`, `useUpdateJournalEntry()`, etc.
  - Query keys: `["learning", "journals", filters]`
- [ ] `use-reading.ts` — reading items, progress, lists:
  - `useReadingItems()`, `useReadingLists()`, `useUpdateReadingProgress()`, etc.
  - Query keys: `["learning", "reading-items"]`, `["learning", "reading-lists"]`
- [ ] `use-progress.ts` — student progress aggregation:
  - `useStudentProgress(studentId)` — activity counts, hours, reading completion
  - Query key: `["learning", "progress", studentId]`
- [ ] `use-quiz.ts` — quiz session management:
  - `useQuizSession(sessionId)`, `useSubmitAnswer()`, `useCompleteQuiz()`, etc.
  - Query keys: `["learning", "quiz", sessionId]`
- [ ] `use-video.ts` — video definitions + progress:
  - `useVideoProgress(videoId)`, `useUpdateVideoProgress()`, etc.
- [ ] `use-sequences.ts` — sequence definitions + progress:
  - `useSequenceProgress(progressId)`, `useAdvanceSequence()`, etc.
- [ ] `use-assignments.ts` — parent assigns content to students:
  - `useAssignments(studentId)`, `useCreateAssignment()`, etc.
- [ ] `use-subjects.ts` — subject taxonomy:
  - `useSubjectTaxonomy()`, `useCreateCustomSubject()`

### Learning Pages (`features/learning/`)

- [ ] `learning-dashboard.tsx` — overview landing page:
  - Quick stats per student (recent activities, reading progress)
  - Methodology-aware tool labels (via `useMethodologyContext()`)
  - Quick-add buttons for logging, journaling, etc.
  - Premium features gated with `<TierGate>`
- [ ] `activity-log.tsx` — log and browse activities:
  - Activity log table/list with date, subject, duration, student
  - Add activity form: title, description, subject tags (from taxonomy), date, duration, student selector
  - Filter by student, date range, subject
- [ ] `journal-list.tsx` — browse journal entries:
  - List view with entry type badge (freeform/narration/reflection)
  - Filter by student, type, date range
- [ ] `journal-editor.tsx` — create/edit journal entry:
  - Rich text area (or structured form per entry type)
  - File attachment support (uses `file-upload.tsx` component)
  - Student selector
  - Subject tags
- [ ] `reading-lists.tsx` — manage reading lists and items:
  - List view with status badges (to_read/in_progress/completed)
  - Add book form: title, author, ISBN, student
  - Status transition buttons
  - Reading list grouping/organization
- [ ] `progress-view.tsx` — per-student progress dashboard (`/learning/progress/:studentId`):
  - Activity counts by subject
  - Reading completion metrics
  - Hours per week chart
  - Assessment scores overview
  - Export button (async export generation)

### Interactive Learning Players

- [ ] `quiz-player.tsx` — interactive quiz (`/learning/quiz/:sessionId`):
  - Session lifecycle: `not_started → in_progress → submitted → scored`
  - Question display (multiple choice, free response, etc.)
  - Answer submission
  - Score display on completion
  - `aria-live` for quiz feedback (CODING_STANDARDS §3.8)
- [ ] `video-player.tsx` — video playback (`/learning/video/:videoId`):
  - HLS streaming support + external video URLs
  - Progress tracking (last position, completion percentage)
  - Accessible controls
- [ ] `content-viewer.tsx` — document/content viewer (`/learning/read/:contentId`):
  - PDF/document rendering
  - Progress tracking
- [ ] `sequence-view.tsx` — lesson sequence (`/learning/sequence/:progressId`):
  - Linear progression display
  - Current step highlight
  - Unlock logic visualization
  - Navigation between sequence items

### Student Features (`features/student/`)

- [ ] `student-dashboard.tsx` — simplified student home:
  - Assigned content list
  - Current sequence progress
  - No social/marketplace access
- [ ] `student-quiz.tsx` — student-facing quiz (simplified wrapper of quiz-player)
- [ ] `student-video.tsx` — student-facing video player
- [ ] `student-reader.tsx` — student-facing content viewer
- [ ] `student-sequence.tsx` — student-facing sequence progression

### Student Session Management

- [ ] `hooks/use-student-session.ts` — manages which student is active:
  - Parent switches between students for logging/viewing
  - Student shell: student is fixed from parent's selection
  - Stored in context (not server state)
  - (ARCHITECTURE §11.2)

### Verification

- [ ] Verify: activity logging creates entries and appears in log
- [ ] Verify: journal creation with attachments works
- [ ] Verify: reading list status transitions work correctly
- [ ] Verify: progress view aggregates data accurately
- [ ] Verify: quiz player handles full session lifecycle
- [ ] Verify: video player tracks progress
- [ ] Verify: sequence navigation enforces unlock logic
- [ ] Verify: student shell restricts navigation
- [ ] Verify: methodology terminology used throughout (not hardcoded labels)
- [ ] Verify: `npm run type-check` passes

### References
- SPEC §6 (learning requirements)
- Domain spec: `specs/domains/06-learn.md`
- Backend TODO: `specs/TODO-06-learn.md` (all backend batches complete)
- ARCHITECTURE §11.3 (learning routes)

---

## Phase 9: Social, Marketplace & Discovery

**Goal**: Build the social feed, messaging, groups, events, marketplace browse/
purchase flow, and creator tools. These are the community features.

**Why here**: These are large feature surfaces that depend on the foundation
from Phases 1–7 but are independent of Phase 8 (learning).

### Prerequisites

- [ ] Run `make full-generate` — social, marketplace, and search endpoints must be in `schema.ts`

### WebSocket Infrastructure

- [ ] `lib/websocket.ts` — WebSocket connection manager:
  - Connect to `wss://api.homegrown.academy/ws` (or dev proxy)
  - Message types: `new_message`, `notification`, `friend_request`
  - Auto-reconnect with exponential backoff
  - (ARCHITECTURE §11.5)
- [ ] `hooks/use-websocket.ts` — hook that connects on mount, dispatches to TanStack Query invalidation:
  - `new_message` → invalidate `["messages", conversationId]`
  - `notification` → invalidate `["notifications"]`
  - `friend_request` → invalidate `["friends", "requests"]`

### Social Features (`features/social/`)

- [ ] `feed.tsx` — social feed (index route `/`):
  - Reverse-chronological posts from friends only
  - 6 post types: text, photo, milestone, event_share, marketplace_review, resource_share
  - Post cards with type-specific rendering
  - Infinite scroll / pagination
  - `aria-live` region for new posts
- [ ] `post-composer.tsx` — create new post:
  - Type selector
  - Text input + photo upload (for photo posts)
  - Student mention (for milestone posts)
- [ ] `post-detail.tsx` — single post view with comments:
  - One level of comment threading
  - Comment composer
- [ ] `friends-list.tsx` — friend management (`/friends`):
  - Friends list
  - Pending requests (incoming/outgoing)
  - Friend search (by display name)
  - Block (silent — blocked user sees no change)
- [ ] `direct-messages.tsx` — DM inbox (`/messages`):
  - Conversation list (friends only, parent-to-parent)
  - Unread indicators
  - Real-time via WebSocket
- [ ] `conversation.tsx` — DM thread (`/messages/:conversationId`):
  - Message list with timestamps
  - Message composer
  - Real-time message delivery
- [ ] `groups-list.tsx` — group directory (`/groups`):
  - Platform-managed groups (by methodology) + user-created
  - Join/leave functionality
- [ ] `group-detail.tsx` — group page (`/groups/:groupId`):
  - Group feed
  - Member list
  - Group info
- [ ] `events-list.tsx` — events directory (`/events`):
  - Event cards with RSVP
  - Filter by date, location region
- [ ] `family-profile.tsx` — public family profile (`/family/:familyId`):
  - Friends-only visibility
  - Family info, methodology, member count
- [ ] Report button component — reusable "Report" action for posts, comments, messages, listings (SPEC §11)

### Marketplace Features (`features/marketplace/`)

- [ ] `marketplace-browse.tsx` — browse listings (`/marketplace`):
  - Faceted filtering: methodology, subject, grade, price, rating, content type, worldview
  - Full-text search
  - Curated sections: Featured, Trending, New Arrivals, Staff Picks
  - Sort: relevance, price, rating, recency
- [ ] `listing-detail.tsx` — listing page (`/marketplace/listings/:id`):
  - Full listing info, preview, reviews
  - Add to cart button
  - Verified-purchaser review display (1-5 stars, anonymous by default)
- [ ] `cart.tsx` — shopping cart (`/marketplace/cart`):
  - Cart items, quantities, total
  - Checkout flow
- [ ] `purchase-history.tsx` — past purchases (`/marketplace/purchases`):
  - Purchase list with download links
  - Content access

### Creator Features (`features/marketplace/creator/`)

- [ ] `creator-dashboard.tsx` — creator home (`/creator`):
  - Sales overview, earnings, payout status
  - Analytics
- [ ] `create-listing.tsx` — new listing (`/creator/listings/new`):
  - Listing form: title, description, price, category, content upload
  - Preview before publish
- [ ] `edit-listing.tsx` — edit existing listing (`/creator/listings/:id/edit`)
- [ ] `quiz-builder.tsx` — create quizzes for marketplace (`/creator/quiz-builder`):
  - Question editor (multiple types)
  - Preview and test
  - Keyboard alternative for any drag-and-drop (CODING_STANDARDS §3.8)
- [ ] `sequence-builder.tsx` — create lesson sequences (`/creator/sequence-builder`):
  - Step editor with ordering
  - Content assignment per step
  - Keyboard alternative for reordering (CODING_STANDARDS §3.8)

### Search (`features/search/`)

- [ ] `search-results.tsx` — global search results (`/search`):
  - Scopes: Social (users, groups, events), Marketplace (listings), Learning (family-scoped)
  - Faceted filtering for marketplace results
  - Sort: relevance, price, rating, recency
- [ ] `components/layout/search-bar.tsx` — persistent search input in header/sidebar:
  - Typeahead/autocomplete
  - Navigates to `/search?q=...`
  - (SPEC §12, domain spec `specs/domains/12-search.md`)

### Verification

- [ ] Verify: social feed displays posts from friends only
- [ ] Verify: DM real-time delivery works via WebSocket
- [ ] Verify: marketplace filtering and search work
- [ ] Verify: cart → purchase flow completes
- [ ] Verify: creator listing authoring works
- [ ] Verify: search returns results across all scopes
- [ ] Verify: report buttons present on all user content
- [ ] Verify: no public profiles — friends-only visibility enforced
- [ ] Verify: `npm run type-check` passes

### References
- SPEC §5 (social), §7 (marketplace), §12 (search)
- Domain specs: `specs/domains/05-social.md`, `specs/domains/07-mkt.md`, `specs/domains/12-search.md`
- ARCHITECTURE §11.3 (routes), §11.5 (WebSocket)

---

## Phase 10: Compliance, Admin, Planning, Polish & Quality Gates

**Goal**: Build premium compliance features, admin panel, calendar/planning,
and complete all cross-cutting polish work. Final quality verification.

**Why last**: These features are either premium-gated, admin-only, or polish
tasks that should only happen after core features are stable.

### Compliance Features (`features/compliance/`) — Premium Only

- [ ] `compliance-setup.tsx` — state compliance configuration:
  - Select state requirements
  - Configure tracking thresholds
  - `<TierGate>` wrapper for free-tier users
- [ ] `attendance-tracker.tsx` — daily attendance marking:
  - Per-student daily attendance toggle
  - Summary with threshold tracking
  - Calendar heatmap view
- [ ] `assessment-records.tsx` — assessment record management:
  - Link assessments to compliance requirements
  - Score tracking
- [ ] `portfolio-generator.tsx` — portfolio PDF generation:
  - Select student + date range
  - Preview portfolio contents
  - Generate + download PDF
  - Print-ready layout (uses `print.css` tokens)
- [ ] References: SPEC §14, domain spec `specs/domains/14-comply.md`

### Planning & Calendar (`features/planning/`)

- [ ] `calendar-view.tsx` — unified calendar (`/calendar`):
  - Synthesize: learning activities + social events + compliance attendance
  - Daily/weekly view toggle
  - Drag-to-schedule (with keyboard alternative)
  - Print-friendly output (MUST be printable — SPEC §17)
- [ ] `schedule-editor.tsx` — create/edit daily/weekly schedules
- [ ] References: SPEC §17, domain spec `specs/domains/17-planning.md`

### Admin Panel (`features/admin/`) — Admin Only

- [ ] `admin-dashboard.tsx` — system health overview (`/admin`):
  - User counts, content stats, system metrics
  - `<AdminGuard>` wrapper
- [ ] `user-management.tsx` — user admin:
  - Search users, view family details
  - Account actions (suspend, delete)
- [ ] `moderation-queue.tsx` — content moderation:
  - Reported content queue
  - Review + action (approve, remove, warn)
  - Moderation states visible to content owners
- [ ] `feature-flags.tsx` — feature flag management
- [ ] References: SPEC §16, domain spec `specs/domains/16-admin.md`

### Data Lifecycle (`features/settings/`)

- [ ] Add data export button to settings — request full data export (GDPR/privacy) (SPEC §15)
- [ ] Add account deletion flow to settings — with confirmation and consequences (SPEC §15)
- [ ] References: domain spec `specs/domains/15-data-lifecycle.md`

### Cross-Cutting Polish

- [ ] Responsive audit — verify all pages work at all breakpoints (sm/md/lg/xl/2xl/3xl)
- [ ] Touch target audit — verify all interactive elements ≥ 44×44px below `md` breakpoint
- [ ] Focus management audit — verify focus moves to `<h1>` on every route change
- [ ] Screen reader audit — verify `aria-live` regions for dynamic content (feed, quiz, notifications)
- [ ] Print stylesheet audit — verify print output for compliance docs, schedules, portfolios
- [ ] `prefers-reduced-motion` audit — verify all animations collapse
- [ ] Surface hierarchy audit — verify no `1px solid` borders, only tonal shifts
- [ ] Token compliance audit — grep for hardcoded hex, arbitrary z-index, Tailwind default palette
- [ ] Error boundary coverage — verify all route segments have error boundaries
- [ ] Loading state coverage — verify skeleton/spinner states for all async data
- [ ] Empty state coverage — verify all list views have empty states with CTAs
- [ ] 404 page — friendly not-found page within AppShell

### Performance

- [ ] Route code-splitting — verify all feature routes lazy-load
- [ ] Image optimization — verify all images use appropriate formats, lazy loading
- [ ] Bundle analysis — check for unexpectedly large dependencies
- [ ] TanStack Query optimization — verify staleTime/gcTime tuned per query type

### Final Quality Gates

- [ ] `npm run type-check` — zero TypeScript errors
- [ ] All pages render without console errors
- [ ] All interactive elements keyboard accessible
- [ ] Lighthouse accessibility score ≥ 90 on all primary pages
- [ ] No `any` types anywhere in codebase (search: `as any`, `: any`)
- [ ] No hardcoded hex colors (search: `#[0-9a-f]`)
- [ ] No `style={{ }}` inline styles
- [ ] No direct `fetch()` calls outside `api/client.ts`
- [ ] No TanStack Query usage outside custom hooks
- [ ] All API types from `src/api/generated/schema.ts` only

### References
- SPEC §14 (compliance), §16 (admin), §17 (planning), §15 (data lifecycle)
- CODING_STANDARDS §3 (all frontend rules)
- DESIGN_TOKENS §18 (implementation checklist)
- DESIGN §3–§5 (visual rules)

---

## Appendix: Generated API Schema Coverage

Types currently in `schema.ts` (domains 01-04):

| Domain | Available Types | Status |
|--------|----------------|--------|
| IAM | `CurrentUserResponse`, `FamilyProfileResponse`, `StudentResponse`, `ConsentStatusResponse`, `CreateStudentCommand`, `UpdateStudentCommand`, `UpdateFamilyCommand`, `CoppaConsentCommand` | Ready |
| Method | `MethodologyID`, `MethodologySummaryResponse`, `MethodologyDetailResponse`, `ActiveToolResponse`, `MethodologySelectionCommand` | Ready |
| Discover | `QuizResponse`, `QuizResultResponse`, `StateGuideSummaryResponse`, `StateGuideResponse` | Ready |
| Onboard | `WizardProgressResponse`, `WizardStatus`, `WizardStep`, `QuizImportResponse`, `CommunityResponse`, `RecommendationsResponse`, `RoadmapResponse`, family/children/methodology commands | Ready |
| Social | — | **Needs `generate-types` run** |
| Learning | — | **Needs `generate-types` run** |
| Marketplace | — | **Needs `generate-types` run** |
| Notify | — | **Needs `generate-types` run** |
| Media | — | **Needs `generate-types` run** |
| Billing | — | **Needs `generate-types` run** |
| Search | — | **Backend in progress** |
| Compliance | — | **Backend not started** |
| Planning | — | **Backend not started** |

Before starting any phase that consumes API types beyond domains 01-04, run:
```bash
make full-generate
```
