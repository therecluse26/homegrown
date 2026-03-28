# TODO: Frontend Implementation Roadmap

> **Scope**: Full React SPA — from empty scaffolding to production-ready.
> **Current state**: React 19 + Vite + TanStack Query wired. Phases 1–6 complete.
> Phase 7 (Parent Dashboard & Family Management) in progress:
> `hooks/use-family.ts` (full family CRUD + co-parent management hooks),
> `hooks/use-notifications.ts` (notification queries + mutations + unread count),
> `hooks/use-data-lifecycle.ts` (export + deletion hooks),
> `features/settings/family-settings.tsx` (3-tab settings: profile, students, co-parents),
> `features/settings/account-settings.tsx`, `notification-prefs.tsx`, `subscription-upgrade.tsx`,
> `features/settings/privacy-controls.tsx`, `session-management.tsx`, `data-export.tsx`,
> `features/settings/account-deletion.tsx`, `student-deletion.tsx`, `notification-center.tsx`,
> `components/layout/notification-bell.tsx` (unread badge in header).
> All pages validated with Playwright. `npm run type-check` passes clean.
> Remaining Phase 5 items: CAPTCHA, mfa-setup, terms versioning,
> COPPA re-verification, end-to-end verification with running Kratos/backend.
> Remaining Phase 7 items: notification-history, billing pages (P2), free tier verification.
>
> **Out of Scope**: The Discovery domain (methodology quiz, explorer pages, state
> guides, Homeschooling 101) lives in the Astro SSG public site per ARCHITECTURE
> §2.4. The only Discovery-adjacent SPA feature is the **quiz result import** during
> onboarding (entering a `share_id` from a quiz taken on the public site). A separate
> TODO for the Astro public site will be created later.

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
Phase 5 ─── Auth Flows (Ory Kratos) + WebSocket Foundation
   │
Phase 6 ─── Onboarding Wizard
   │
Phase 7 ─── Parent Dashboard & Family Management
   │
   ├──────────────────┬──────────────────┐
Phase 8               Phase 9            Phase 10
Learning Tools        Social, Mkt,       Compliance,
& Progress            Search & Admin     Planning & QA
```

Phases 8–10 may proceed in parallel once Phase 7 is complete.

---

## Backend Prerequisites

These backend features are referenced by the TODO but do not yet have HTTP
handlers. They MUST be implemented before the frontend phases that depend on them.

| Backend Gap | Blocks Phase | Notes |
|-------------|-------------|-------|
| Data lifecycle handler (`internal/lifecycle/handler.go`) + route registration | Phase 7 (data export, account deletion) | Service layer + tests exist; needs handler + main.go wiring |
| ~~Planning domain (`internal/planning/`)~~ | ~~Phase 10~~ | ✅ Backend complete (migration 24, 47 unit tests, wired in main.go) |
| Session list/revoke API in IAM | Phase 5 (session-management.tsx) | Kratos admin API could proxy this; needs IAM handler endpoints |

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
| Family-scoped data | All data queries scoped by family — never show cross-family data | ARCHITECTURE §5 |
| File naming | kebab-case for files, PascalCase for components | CODING_STANDARDS §3.6 |
| Privacy | No GPS, no PII in logs, COPPA compliance | SPEC §7, ARCHITECTURE §8 |
| i18n readiness | All user-facing strings externalized via react-intl; no hardcoded English | SPEC §17.7, ARCHITECTURE §11.6 |
| axe-core CI | Zero critical/serious accessibility violations in automated checks | SPEC §17.6.6 |
| Error handling | TanStack Query retries (3× w/ backoff), graceful offline/error states on all pages | CODING_STANDARDS §3.5 |
| Print readiness | Compliance, scheduling, and progress output must be printable via `print.css` | SPEC §17.9 |
| Dark mode readiness | No `dark:` prefixes; all colors via token classes; CSS-only theme switch structure | DESIGN_TOKENS §2.9 |
| Family-scoped validation | All data-fetching hooks scope queries by family. Never display cross-family data. Include `familyId` in query keys. | ARCHITECTURE §5 |
| API error handling | Never expose internal error details in UI. Map `apiClient` errors to user-friendly messages. Generic "Something went wrong" for 500s; specific messages only for 422 validation errors. | CODING_STANDARDS §5.2 |
| Loading states | Every async page/component MUST show `<Skeleton>` or `<Spinner>` during loading. | CODING_STANDARDS §3.5 |
| Empty states | Every list view MUST show `<EmptyState>` with contextual message + CTA when empty. | DESIGN §5 |

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

- [x] `npm install @tailwindcss/vite lucide-react` — add missing deps (DESIGN_TOKENS §1.3, §13.1) `[P1]`
- [x] `npm uninstall autoprefixer postcss` — unnecessary with Tailwind v4 (DESIGN_TOKENS §1.3) `[P1]`
- [x] Update `frontend/vite.config.ts` — add `import tailwindcss from "@tailwindcss/vite"` and include `tailwindcss()` in `plugins[]` (DESIGN_TOKENS §1.3) `[P1]`
- [x] Remove `postcss.config.*` if present — Tailwind v4 needs no PostCSS config `[P1]`

### CSS File Structure

Create `frontend/src/styles/` with the following files (DESIGN_TOKENS §1.2):

- [x] `tokens.css` — `@theme` block with all CSS custom properties: `[P1]`
  - Surface colors (7 tiers), primary (3), secondary (4), tertiary (2), feedback (6), text/outline (4), inverse (2) (DESIGN_TOKENS §3)
  - Font families: `--font-display`, `--font-body` (DESIGN_TOKENS §4)
  - Spacing semantic aliases (6 tokens) (DESIGN_TOKENS §6)
  - Border radii (7 tokens) (DESIGN_TOKENS §7)
  - Shadows (4 ambient + ghost-border) (DESIGN_TOKENS §8)
  - Z-index scale (8 tiers) (DESIGN_TOKENS §9)
  - Animation durations (4) + easings (3) (DESIGN_TOKENS §10)
  - Breakpoint: `--breakpoint-3xl: 1920px` (DESIGN_TOKENS §15)
  - Opacity tokens (8 values): `--opacity-disabled` (0.38), `--opacity-hover` (0.08), `--opacity-pressed` (0.12), `--opacity-focus` (0.12), `--opacity-dragged` (0.16), `--opacity-glass` (0.8), `--opacity-ghost-border` (0.2), `--opacity-scrim` (0.32) (DESIGN_TOKENS §11)
  - Container width tokens: `--width-content` (72rem), `--width-content-narrow` (48rem), `--width-sidebar` (16rem), `--width-sidebar-collapsed` (4rem) (DESIGN_TOKENS §8.2)
  - `:root` block for non-theme tokens (gradients, focus-ring) (DESIGN_TOKENS §11)
  - Gradient tokens in `:root` block: `--gradient-primary` (135deg primary→primary-container), `--gradient-surface` (180deg surface→surface-container-low) (DESIGN_TOKENS §16)
- [x] `base.css` — `@font-face` declarations, CSS reset additions, `:focus-visible` ring, `prefers-reduced-motion` (DESIGN_TOKENS §4, §10.2) `[P1]`
- [x] `components.css` — type-scale composite classes (`.text-display-lg` through `.text-label-sm`, 15 steps) (DESIGN_TOKENS §5) `[P1]`
- [x] `utilities.css` — z-index utilities (`.z-base` through `.z-tooltip`), `.touch-target` class using `::after` pseudo-element for 44×44px hit area (DESIGN_TOKENS §14.3), parent/student Tailwind custom variants (`parent:`, `student:`) using `@custom-variant` with `:where([data-context])` selectors (DESIGN_TOKENS §9, §12, §15.1), MD3 state layer utility classes: `.state-hover`, `.state-pressed`, `.state-focus`, `.state-dragged` applying opacity token overlays via `::before` pseudo-element (DESIGN_TOKENS §14.1) `[P1]`
- [x] `print.css` — `@media print` token overrides + hide rules (DESIGN_TOKENS §16) `[P1]`
- [x] `app.css` — entry point: `@import "tailwindcss"` then import all above files (DESIGN_TOKENS §1.2) `[P1]`

### Wiring

- [x] Add `import "./styles/app.css"` to `frontend/src/main.tsx` (before QueryClientProvider) `[P1]`
- [x] Verify: Vite dev server starts without CSS errors `[P1]`
- [x] Verify: Tailwind utility classes (e.g. `bg-primary`, `text-on-surface`) resolve correctly in a test `<div>` `[P1]`

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

- [x] Download WOFF2 files for **Plus Jakarta Sans** (SemiBold 600, Bold 700) and **Manrope** (Regular 400, Medium 500, SemiBold 600) — COPPA: no Google Fonts CDN (DESIGN_TOKENS §4.1, SPEC §7) `[P1]`
- [x] Create `frontend/public/fonts/` directory and place WOFF2 files there `[P1]`
- [x] Add font preload hints to `frontend/index.html`: `[P1]`
  ```html
  <link rel="preload" href="/fonts/PlusJakartaSans-SemiBold.woff2" as="font" type="font/woff2" crossorigin />
  <link rel="preload" href="/fonts/Manrope-Regular.woff2" as="font" type="font/woff2" crossorigin />
  ```
- [x] Write `@font-face` rules in `base.css` with `font-display: swap` (DESIGN_TOKENS §4.2) `[P1]`

### Type Scale Verification

- [x] Confirm all 15 type-scale composite classes render correctly (DESIGN_TOKENS §5): `[P1]`
  - Display: `display-lg` (3.5rem), `display-md` (2.8125rem), `display-sm` (2.25rem)
  - Headline: `headline-lg` (2rem), `headline-md` (1.75rem), `headline-sm` (1.5rem)
  - Title: `title-lg` (1.375rem), `title-md` (1rem), `title-sm` (0.875rem)
  - Body: `body-lg` (1rem/1.6), `body-md` (0.875rem/1.55), `body-sm` (0.75rem)
  - Label: `label-lg` (0.875rem), `label-md` (0.75rem), `label-sm` (0.6875rem)
- [x] Verify display/headline fonts use Plus Jakarta Sans (SemiBold/Bold) `[P1]`
- [x] Verify body/label fonts use Manrope (Regular/Medium/SemiBold) `[P1]`

### Base Styles

- [x] Set `body` background to `surface` (#faf9f5), text color to `on-surface` (#1b1c1a) (DESIGN_TOKENS §3.1, DESIGN §3.1) `[P1]`
- [x] Configure `::selection` style (primary-container bg, on-primary text) `[P1]`
- [x] Implement `:focus-visible` ring: 2px solid `focus-ring` (#0c5252), 2px offset (DESIGN_TOKENS §10.3) `[P1]`
- [x] Implement `prefers-reduced-motion: reduce` — collapse all transitions to 0.01ms (DESIGN_TOKENS §10.2) `[P1]`

### Token & Behavior Verification

- [x] No-flash guarantee: background transitions use `--duration-normal` or longer; no transition applied on first paint (DESIGN_TOKENS §7.4) `[P1]`
- [x] Semantic spacing alias verification: all 6 spacing aliases (`--space-xs` through `--space-3xl`) resolve to correct token values (DESIGN_TOKENS §4.2) `[P1]`
- [x] Print stylesheet verification: `print.css` renders correctly in browser print preview — no token colors bleeding into print, appropriate page breaks (DESIGN_TOKENS §16) `[P1]`

### Verification

- [x] Verify: `npm run type-check` passes `[P1]`
- [x] Verify: Vite dev server renders correct fonts and colors on a test page `[P1]`
- [x] Verify: all token utilities available in Tailwind IntelliSense `[P1]`

### References
- DESIGN_TOKENS §3–§5, §7.4, §10, §16
- DESIGN §3.1–§3.4 (Curated Hearth aesthetic)

---

## Phase 3: Shared UI Component Library

**Goal**: Build the reusable primitive components that every feature will compose.
All components use token classes exclusively.

**Why third**: Features cannot be built without buttons, inputs, cards, modals.
Building these first prevents duplication and ensures consistency.

### Directory Setup

- [x] Establish `frontend/src/components/ui/` structure `[P1]`
- [x] Establish `frontend/src/components/common/` structure `[P1]`
- [x] Create barrel exports (`index.ts`) for each directory `[P1]`

### Core Primitives (`components/ui/`)

- [x] `button.tsx` — 3 variants: primary (solid primary bg), secondary (secondary-container fill), tertiary (transparent, hover surface-container-low). All use `radius-button` (0.75rem). Min tap target 44px on mobile. Full MD3 state matrix: hover (8% on-* overlay), active/pressed (12% overlay), focus (focus ring), disabled (38% opacity), loading (spinner centered, text hidden). Context-dependent typography: parent context = `label-md` UPPERCASE; student context = `title-sm` sentence case. High-impact CTA gradient variant: `--gradient-primary` bg for "Start Lesson"-type CTAs. (DESIGN §2, §5.1, DESIGN_TOKENS §7, §14.2) `[P1]`
- [x] `input.tsx` — full 5-state matrix: default (bg `surface-container-highest`, no border), hover (bg `surface-container-high`), focus (2px inset `primary` ghost border via `box-shadow` — NOT standard border to avoid layout shift), error (bg `error-container`, 2px inset `error` border, error text below with `aria-describedby`), disabled (38% opacity, `pointer-events: none`). Visible `<label>` required. (DESIGN §5.3, DESIGN_TOKENS §6.2, §14.2, CODING_STANDARDS §3.8) `[P1]`
- [x] `textarea.tsx` — same styling rules as input, auto-resize optional `[P1]`
- [x] `select.tsx` — custom select with accessible keyboard nav, token styling `[P1]`
- [x] `checkbox.tsx` — accessible checkbox with `primary` accent `[P1]`
- [x] `radio.tsx` — accessible radio group with `primary` accent `[P1]`
- [x] `card.tsx` — accepts `context` prop for parent (`radius-lg`) vs student (`radius-xl`). Background `surface-container-lowest` (#fff) for "lifted" cards. Interactive card states: non-interactive (no shadow, depth via tonal shift only), interactive hover (`ambient-sm` shadow — subtle lift), interactive active (bg shifts to `surface-container-low`), interactive focus (focus ring). Shadow rule: tonal shifts for standard elements, shadows ONLY for floating/hover states. (DESIGN §4.2, DESIGN_TOKENS §7–§8, §14.2) `[P1]`
- [x] `badge.tsx` — pill shape (`radius-full`), methodology-aware coloring `[P1]`
- [x] `avatar.tsx` — circular, fallback initials, sizes xs–xl `[P1]`
- [x] `modal.tsx` — focus trap, return focus on close, overlay at `z-modal`, `Escape` to close. (CODING_STANDARDS §3.8, DESIGN_TOKENS §9) `[P1]`
- [x] `toast.tsx` — auto-dismiss, `z-notification`, `aria-live="polite"`. Success/error/warning variants using feedback tokens. (DESIGN_TOKENS §3.5, §9) `[P1]`
- [x] `spinner.tsx` — loading indicator with `primary` color `[P1]`
- [x] `skeleton.tsx` — content placeholder with pulse animation `[P1]`
- [x] `icon.tsx` — Lucide icon wrapper with standard sizes (xs=12, sm=16, md=20, lg=24, xl=32, 2xl=48) (DESIGN_TOKENS §13) `[P1]`
- [x] `tooltip.tsx` — `z-tooltip`, accessible (shows on focus), delay on hover `[P1]`
- [x] `dropdown-menu.tsx` — `z-popover`, keyboard navigable, focus management `[P1]`
- [x] `tabs.tsx` — accessible tab panel with `aria-selected`, keyboard arrow nav `[P1]`
- [x] `progress-bar.tsx` — progress ribbon: `tertiary-fixed` bg, `primary` fill (DESIGN §5.4) `[P1]`
- [x] `empty-state.tsx` — illustration slot + message + CTA button `[P1]`
- [x] `date-picker.tsx` — WAI-ARIA date picker, keyboard navigable (arrow keys, Escape), used by events/attendance/planning `[P1]`
- [x] `calendar.tsx` — shared calendar grid component for planning domain's unified view `[P1]`
- [x] `rich-text-editor.tsx` — structured text editor for journals, post creation, listing descriptions; toolbar with basic formatting `[P1]`
- [x] `star-rating.tsx` — 1–5 star input/display for marketplace reviews, keyboard accessible (arrow keys) `[P1]`
- [x] `faceted-filter.tsx` — checkbox/range filter panel for marketplace browse and global search `[P1]`
- [x] `infinite-scroll.tsx` — Intersection Observer-based pagination trigger for feeds and list views `[P1]`
- [x] `data-table.tsx` — sortable/filterable table for admin views, compliance logs, activity history `[P2]`
- [x] `confirmation-dialog.tsx` — reusable delete/destructive-action confirmation dialog with customizable message `[P1]`
- [x] `stat-card.tsx` — reusable metric display card for dashboards (value, label, trend indicator) `[P1]`
- [x] `breadcrumb.tsx` — nested route breadcrumbs for deep navigation in settings, compliance, admin `[P1]`
- [x] `image-gallery.tsx` — photo grid with lightbox overlay for social posts, journals, portfolios `[P1]`
- [x] `link.tsx` — styled anchor with state matrix: default (`primary` color, no decoration), hover (`primary-container` color, underline), active (`primary`, underline), focus (focus ring), visited (same as default — privacy, no visited indicator). `external` prop adds `rel="noopener noreferrer"` + optional icon. (DESIGN_TOKENS §14.2) `[P1]`
- [x] `list.tsx` — list container enforcing "no divider" rule: items separated by `spacing-list-gap` (1.4rem) OR alternating `surface`/`surface-container-low` bg. NEVER 1px borders between items. (DESIGN §5.2) `[P1]`

### Common Components (`components/common/`)

- [x] `skip-link.tsx` — "Skip to main content" as first focusable element on every page (CODING_STANDARDS §3.8) `[P1]`
- [x] `methodology-badge.tsx` — methodology chip with config-driven label (not hardcoded) `[P1]`
- [x] `user-avatar.tsx` — wraps avatar with user/student data `[P1]`
- [x] `tier-gate.tsx` — shows upgrade prompt when free user hits premium feature; links to `/settings/subscription`, shows specific feature name being gated (SPEC §10) `[P1]`
- [x] `error-boundary.tsx` — React error boundary with friendly fallback UI `[P1]`
- [x] `page-title.tsx` — sets `document.title` + renders `<h1>` for focus target on route transitions `[P1]`
- [x] `report-button.tsx` — report dialog accepting `targetType` + `targetId` (supports all 11 entity types per safety domain spec); links to community guidelines. Report category dropdown: inappropriate content, harassment, spam, misinformation, CSAM/child safety, methodology hostility, other. Free-text description field (optional). Confirmation acknowledgment after submission. (SPEC §11, §12.3, 11-safety §3.1) `[P1]`
- [x] `network-status.tsx` — offline/online banner/toast; detects connectivity via `navigator.onLine` + `online`/`offline` events `[P1]`

### Form Utilities

- [x] `form-field.tsx` — wraps input + label + error message with consistent spacing. Visible `<label>` required (never placeholder-only), `aria-describedby` linking error message `<span>` to input, `aria-live="assertive"` on error message container for screen reader announcements (CODING_STANDARDS §3.8) `[P1]`
- [x] `file-upload.tsx` — drag-and-drop zone, file type + size validation (client-side), progress indicator. Extension validation pre-upload, magic byte validation happens server-side. Upload progress bar with `aria-valuenow`/`aria-valuemax`. Error states: file too large (show max size), wrong type (show allowed types), server rejection (generic message). (SPEC §9, CODING_STANDARDS §3.9, 09-media §9) `[P1]`

### Verification

- [x] Verify: all components render correctly in isolation (consider Storybook or a dev route) `[P1]`
- [x] Verify: `npm run type-check` passes `[P1]`
- [x] Verify: tab navigation works through all interactive components `[P1]`
- [x] Verify: screen reader announces all components correctly `[P1]`
- [x] Verify: date-picker keyboard nav per WAI-ARIA (arrow keys, Enter, Escape) `[P1]`
- [x] Verify: star-rating keyboard accessible (arrow keys to change value) `[P1]`
- [x] Verify: infinite-scroll announces new content via `aria-live` region `[P1]`

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

- [x] Create `frontend/src/hooks/` directory `[P1]`
- [x] Create `frontend/src/lib/` directory `[P1]`
- [x] Move `src/query-client.ts` → `src/lib/query-client.ts` (update import in `main.tsx`) `[P1]`
- [x] Configure `lib/query-client.ts` — retry (3× exponential backoff), staleTime (5 min), gcTime (10 min) (CODING_STANDARDS §3.5) `[P1]`
- [x] Create `frontend/src/types/` directory with `index.ts` for shared frontend types `[P1]`
- [x] Create `frontend/src/components/layout/` directory `[P1]`
- [x] Create `frontend/src/features/` subdirectories matching all backend domains: `auth/`, `onboarding/`, `social/`, `learning/`, `marketplace/`, `settings/`, `billing/`, `compliance/`, `admin/`, `planning/`, `recommendations/`, `search/`, `legal/`, `student/` (ARCHITECTURE §11.1) `[P1]`
- [x] `hooks/use-focus-on-mount.ts` — reusable focus management hook for route transitions; accepts ref, focuses element on mount (CODING_STANDARDS §3.8) `[P1]`

### Auth Context & Hook

- [x] `hooks/use-auth.ts` — custom hook wrapping TanStack Query for `GET /auth/me`: `[P1]`
  - Returns `{ user, isLoading, isAuthenticated, isParent, isPrimaryParent, tier, coppaStatus }`
  - Uses `CurrentUserResponse` from generated types
  - Handles 401 (not logged in) gracefully — sets `isAuthenticated: false`
  - Query key: `["auth", "me"]`
- [x] `features/auth/auth-provider.tsx` — `AuthContext` provider wrapping the app, uses `use-auth` internally (ARCHITECTURE §11.2) `[P1]`

### Methodology Context & Hook

- [x] `hooks/use-methodology.ts` — wraps TanStack Query for `GET /families/tools` and methodology config: `[P1]`
  - Returns tools, terminology labels, active methodology slug
  - Depends on auth (only fetches when authenticated)
  - Query key: `["family", "tools"]`
- [x] `features/auth/methodology-provider.tsx` — `MethodologyContext` provider (ARCHITECTURE §11.2) `[P1]`

### i18n Infrastructure

- [x] `lib/i18n.ts` — react-intl setup with lazy-loaded locale data `[P1]`
- [x] `locales/en.json` — initial English string catalog (all user-facing strings extracted here) `[P1]`
- [x] Wrap `App.tsx` with `<IntlProvider>` using locale from `lib/i18n.ts` `[P1]`

### Layout Components (`components/layout/`)

- [x] `app-shell.tsx` — main authenticated layout: `[P1]`
  - Sidebar navigation (desktop) / bottom nav (mobile)
  - Header with search bar, notification bell, user menu
  - `<main>` content area with `data-context="parent"` attribute
  - Skip-link as first child
  - Responsive: sidebar collapses below `lg` breakpoint
  - Floating nav: `surface-container-low` at 80% opacity + `backdrop-blur: 20px` (DESIGN §4.4)
  - Note: `data-context="parent"` enables `parent:` Tailwind variant; `data-context="student"` enables `student:` variant
  - Intentional Asymmetry: hero sections support offset `display-lg` headline overlapping floating card at 4rem offset (DESIGN §1)
- [x] `student-shell.tsx` — supervised student layout: `[P1]`
  - Simplified nav (no social, no marketplace, no settings)
  - `data-context="student"` on `<main>` (enables `student:` variant)
  - Larger tap targets, more rounded corners (`radius-xl`)
  - Back-to-parent button always visible
- [x] `auth-layout.tsx` — unauthenticated layout for login/register/recovery pages (centered card, no sidebar) `[P1]`
- [x] `onboarding-layout.tsx` — minimal layout for the onboarding wizard (progress indicator, no full nav) `[P1]`
- [x] `admin-shell.tsx` — admin-specific layout with admin navigation sidebar, system health indicators, and admin-only actions `[P1]`

### Route Guards

- [x] `components/layout/protected-route.tsx` — redirects to `/auth/login` if not authenticated `[P1]`
- [x] `components/layout/onboarding-guard.tsx` — redirects to `/onboarding` if authenticated but onboarding not complete (check `WizardProgressResponse.status !== "completed"` and `!== "skipped"`) `[P1]`
- [x] `components/layout/admin-guard.tsx` — redirects to `/` if not `is_platform_admin` (SPEC §16) `[P1]`
- [x] `components/layout/student-guard.tsx` — validates active student session `[P1]`

### Router Setup

- [x] `src/routes.tsx` — full route tree using React Router v7 with lazy loading: `[P1]`
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
    /marketplace/purchases/:id/refund → RefundRequest      [P1]
    /creator → CreatorDashboard
    /creator/listings/new → CreateListing
    /creator/listings/:id/edit → EditListing
    /creator/quiz-builder → QuizBuilder
    /creator/quiz-builder/:id → QuizBuilder
    /creator/sequence-builder → SequenceBuilder
    /creator/sequence-builder/:id → SequenceBuilder
    /creator/payouts → PayoutSetup                         [P2]
    /settings → FamilySettings
    /settings/notifications → NotificationPrefs
    /settings/subscription → SubscriptionManager
    /settings/account → AccountSettings                    [P1]
    /settings/account/sessions → SessionManagement         [P1]
    /settings/account/export → DataExport                  [P1]
    /settings/account/delete → AccountDeletion             [P1]
    /settings/account/appeals → ModerationAppeals          [P1]
    /settings/privacy → PrivacyControls                    [P1]
    /search → SearchResults
    /family/:familyId → FamilyProfile
    /calendar → CalendarView
    /calendar/day/:date → DayView                          [P1]
    /calendar/week/:date → WeekView                        [P1]
    /planning/templates → ScheduleTemplates                [P2]
    /legal/terms → TermsOfService                          [P1]
    /legal/privacy → PrivacyPolicy                         [P1]
    /legal/guidelines → CommunityGuidelines                [P1]
  /auth (AuthLayout)
    /auth/login → Login
    /auth/register → Register
    /auth/recovery → AccountRecovery
    /auth/verification → EmailVerification
    /auth/accept-invite/:token → AcceptInvitation          [P1]
  /onboarding (ProtectedRoute + OnboardingLayout)
    index → OnboardingWizard
  /student (ProtectedRoute + StudentGuard + StudentShell)
    index → StudentDashboard
    /student/quiz/:sessionId → StudentQuiz
    /student/video/:videoId → StudentVideo
    /student/read/:contentId → StudentReader
    /student/sequence/:progressId → StudentSequence
  /admin (ProtectedRoute + AdminGuard + AdminShell)        [P1]
    index → AdminDashboard
    /admin/users → UserManagement                          [P1]
    /admin/users/:id → UserDetail                          [P1]
    /admin/moderation → ModerationQueue                    [P1]
    /admin/flags → FeatureFlags                            [P2]
    /admin/audit → AuditLog                                [P1]
    /admin/methodologies → MethodologyConfig               [P2]
  /compliance (ProtectedRoute + TierGate)                  [P2]
    index → ComplianceSetup
    /compliance/attendance → AttendanceTracker
    /compliance/assessments → AssessmentRecords
    /compliance/tests → StandardizedTests                  [P2]
    /compliance/portfolios → PortfolioList                 [P3]
    /compliance/portfolios/:id → PortfolioBuilder          [P3]
    /compliance/transcripts → TranscriptList               [P3]
    /compliance/transcripts/:id → TranscriptBuilder        [P3]
    * → NotFoundPage                                      [P1]
  ```
  (ARCHITECTURE §11.3)
- [x] Update `src/App.tsx` — replace stub with `RouterProvider` + providers (Auth → Methodology → IntlProvider → Router) `[P1]`
- [x] Update `src/main.tsx` — ensure provider ordering: `StrictMode → QueryClientProvider → App` `[P1]`

### Error Boundaries

- [x] Wrap each top-level route segment in `<RouteErrorBoundary>` with friendly fallback UI + "Go home" CTA `[P1]`
  - Segments: auth, onboarding, learning, social, marketplace, creator, settings, compliance, admin, planning, student, legal, search
- [x] `features/auth/accept-invitation.tsx` — co-parent invitation acceptance: validates token, accept/decline buttons, join confirmation (SPEC §3.4, 01-iam) `[P1]` (simplified: `InvitationInfoResponse` not in schema; shows generic invite UI)

### Route Transition Accessibility

- [x] On every route change, move focus to the page's `<h1>` or main content region (CODING_STANDARDS §3.8) `[P1]`
- [x] Announce page title changes to screen readers via `aria-live` or `document.title` update `[P1]`

### Verification

- [x] Verify: unauthenticated user sees `/auth/login` `[P1]`
- [ ] Verify: authenticated user with incomplete onboarding redirects to `/onboarding` `[P1]`
- [ ] Verify: authenticated user with complete onboarding sees AppShell at `/` `[P1]`
- [x] Verify: all routes lazy-load correctly (check network tab) `[P1]`
- [x] Verify: `npm run type-check` passes `[P1]`
- [ ] Verify: keyboard navigation through sidebar/nav links works `[P1]`
- [ ] Verify: admin routes accessible only to `is_platform_admin` users `[P1]`
- [ ] Verify: compliance routes show TierGate for free-tier families `[P2]`
- [x] Verify: route tree covers all 17 domains (IAM, method, discover-import, onboard, social, learn, mkt, notify, media, billing, safety, search, recs, comply, data-lifecycle, admin, planning) `[P1]`
- [x] Verify: skip link targets `#main-content` and is the first focusable element on every layout `[P1]`
- [x] Verify: `apiClient` is the sole fetch wrapper — no direct `fetch()` calls elsewhere in the codebase `[P1]`

### References
- ARCHITECTURE §11.1 (project structure), §11.2–§11.3 (auth strategy, route table)
- CODING_STANDARDS §3.5 (TanStack Query rules)
- CODING_STANDARDS §3.8 (accessibility)
- DESIGN §4.4 (floating nav)

---

## Phase 5: Auth Flows (Ory Kratos Integration) + WebSocket Foundation

**Goal**: Implement login, registration, account recovery, and email verification
screens using Ory Kratos Browser API. Wire COPPA consent flow.

**Why fifth**: Users must be able to authenticate before any feature is usable.
Depends on Phase 4 auth context and layout.

### Kratos Integration Utilities

- [x] `lib/kratos.ts` — helper functions for Kratos Browser API: `[P1]`
  - `initLoginFlow()` — fetches login flow from Kratos
  - `initRegistrationFlow()` — fetches registration flow
  - `initRecoveryFlow()` — fetches recovery flow
  - `initVerificationFlow()` — fetches verification flow
  - `submitFlow(flowId, body)` — submits a flow to Kratos (returns `FlowResult` discriminated union)
  - Error mapping: Kratos validation errors → form field errors
  - CSRF token handling
  - (ARCHITECTURE §11.2)

### Auth Pages (`features/auth/`)

- [x] `login.tsx` — email/password form + OAuth buttons (Google, Facebook, Apple). Error display for invalid credentials. Link to register + recovery. (SPEC §1, ARCHITECTURE §11.2) `[P1]`
- [x] `register.tsx` — email/password + optional OAuth. On success: Kratos webhook triggers `POST /hooks/kratos/post-registration` → family+parent created automatically. Redirect to `/onboarding`. (SPEC §1, ARCHITECTURE §11.2) `[P1]`
- [x] `account-recovery.tsx` — email input for password reset link. Success confirmation message. `[P1]`
- [x] `email-verification.tsx` — handles `?flow=xxx` re-entry from URL. Shows success/error state. `[P1]`
- [x] `oauth-button.tsx` — reusable OAuth button component with provider-specific icons (Google, Facebook, Apple) + Kratos OAuth redirect initiation `[P1]`
- [x] Terms of service / privacy policy acceptance checkbox on registration form (must accept before submit) `[P1]`
- [ ] CAPTCHA integration on registration (hCaptcha or Turnstile) to prevent automated signups `[P1]`

### Session & MFA Management

- [x] `features/settings/session-management.tsx` — list active sessions, revoke individual sessions, "log out all devices" button `[P1]`
- [ ] `features/settings/mfa-setup.tsx` — TOTP MFA setup: QR code display, verification input, recovery codes display + download `[P2]`
- [x] `features/auth/session-timeout-warning.tsx` — overlay 5min before session expiry: countdown timer, "Extend Session" button, auto-redirect to `/auth/login` on timeout, `aria-live="assertive"` for countdown announcements (SPEC §17.1) `[P1]`

### Legal Pages (`features/legal/`)

- [x] `terms-of-service.tsx` — `/legal/terms` — rendered ToS content `[P1]`
- [x] `privacy-policy.tsx` — `/legal/privacy` — rendered privacy policy `[P1]`
- [x] `community-guidelines.tsx` — `/legal/guidelines` — community guidelines linked from report dialog `[P1]`
- [ ] Terms versioning: re-acceptance prompt banner when policy version changes; dismissable only by accepting new terms. COPPA re-verification trigger for families with students when ToS version changes (SPEC §7.3) `[P2]`

### COPPA Consent Flow

- [x] `hooks/use-consent.ts` — wraps `GET /families/consent` + `POST /families/consent`: `[P1]`
  - Returns `{ consentStatus, acknowledge, provideConsent }`
  - Query key: `["family", "consent"]`
- [x] `features/auth/coppa-consent.tsx` — consent gate component: `[P1]`
  - Shown after registration before adding students
  - Status flow: `registered → noticed → consented` (SPEC §7.3)
  - Must be completed before any student can be created
  - Clear, parent-friendly language explaining data collection

### Auth UX Enhancements

- [x] Password strength indicator on registration form — visual meter (colored bar) + descriptive text label (weak/fair/strong), colors use token feedback palette (SPEC §1) `[P1]`
- [x] Rate limiting feedback on login — `429` response → friendly "Too many attempts" message + retry countdown timer (SPEC §1.2) `[P1]`
- [x] Email verification resend button with 60-second cooldown timer — disabled state + countdown during cooldown (SPEC §1.3) `[P1]`
- [ ] COPPA consent re-verification prompt when ToS version changes for families with students already added (SPEC §7.3) `[P1]`
- [ ] `features/auth/coppa-micro-charge.tsx` — COPPA micro-charge verification: micro-charge explanation, amount verification input, retry on mismatch (10-billing §13) `[P2]`

### WebSocket Foundation

> Placed in Phase 5 because WebSocket is a foundation layer needed by notifications
> (Phase 7) and social/messaging (Phase 9). ARCHITECTURE §11.4 places it alongside
> core infrastructure.

- [x] `lib/websocket.ts` — WebSocket connection manager: `[P1]`
  - Connect to `/v1/social/ws` (proxied via Vite in dev, full URL in production)
  - Message types: `new_message`, `notification`, `friend_request`
  - Auto-reconnect with exponential backoff
  - (ARCHITECTURE §11.5)
- [x] `hooks/use-websocket.ts` — hook that connects on mount, dispatches to TanStack Query invalidation: `[P1]`
  - `new_message` → invalidate `["messages", conversationId]`
  - `notification` → invalidate `["notifications"]`
  - `friend_request` → invalidate `["friends", "requests"]`

### Verification

- [ ] Verify: login flow works end-to-end with Kratos (or mock for dev) `[P1]`
- [ ] Verify: registration creates family + parent via webhook `[P1]`
- [ ] Verify: COPPA consent blocks student creation until consented `[P1]`
- [ ] Verify: recovery email flow works `[P1]`
- [ ] Verify: OAuth buttons present and functional `[P1]`
- [ ] Verify: `npm run type-check` passes `[P1]`
- [ ] Verify: ToS/privacy acceptance required before registration completes `[P1]`
- [ ] Verify: session management lists and revokes sessions correctly `[P1]`
- [ ] Verify: password strength indicator updates reactively `[P1]`
- [ ] Verify: rate limiting shows countdown and re-enables login `[P1]`
- [ ] Verify: email verification resend cooldown works `[P1]`
- [ ] Verify: WebSocket connects, reconnects on disconnect, and dispatches invalidations `[P1]`

### References
- ARCHITECTURE §11.2 (Kratos integration), §11.5 (WebSocket)
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

- [x] `hooks/use-onboarding.ts` — wraps onboarding API endpoints: `[P1]`
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

- [x] `onboarding-wizard.tsx` — wizard container: `[P1]`
  - Progress indicator showing 4 steps with current/completed/upcoming states
  - Step navigation (back/next) with validation
  - Skip button always available (`POST /onboarding/skip`)
  - Reads `WizardProgressResponse.current_step` and `completed_steps[]` to determine state
  - Step enum: `family_profile → children → methodology → roadmap_review`
- [x] `steps/family-profile-step.tsx` — family name, state selection, location region `[P1]`
  - Validate required fields before allowing next
  - State selection in family-profile-step feeds into compliance domain — selected state is used by `GET /v1/compliance/state-requirements/:state_code` to surface relevant requirements later
- [x] `steps/children-step.tsx` — add student profiles: `[P1]`
  - Display name, birth year, grade level
  - Add/remove students dynamically
  - COPPA consent must be complete before this step (gate check)
  - Optional step — can proceed with zero students
- [x] `steps/methodology-step.tsx` — three methodology paths: `[P1]`
  - **Quiz-informed**: text input for `share_id` or full quiz URL, "Take the quiz first" outbound link to public site, success state showing matched methodology + confidence score
  - **Exploration**: Browse methodology cards (GET `/methodologies`), drill into detail (GET `/methodologies/{slug}`), select one
  - **Skip**: Proceed with no methodology selected
  - Display methodology tools preview for selected methodology
- [x] `steps/roadmap-review-step.tsx` — personalized roadmap: `[P1]`
  - GET `/onboarding/roadmap` — age-adapted recommendations with visual age bracket indicator (0–4, 5–7, 8–10, 11–13, 14–18), different content per bracket
  - GET `/onboarding/recommendations` — starter curriculum recommendation cards: top 3 per methodology+grade
  - GET `/onboarding/community` — methodology groups, nearby families count, mentor suggestions
  - Co-parent invite CTA (optional, can defer to settings)
  - "Complete" button → `POST /onboarding/complete` → redirect to `/`
  - Note: Co-parent invitation acceptance has a dedicated `/auth/accept-invite/:token` page (component in Phase 4/5) — SPEC §3.4

### Methodology Explorer (shared — also used in settings)

- [x] `hooks/use-methodologies.ts` — wraps `GET /methodologies` and `GET /methodologies/{slug}`: `[P1]`
  - `useMethodologyList()` — query key: `["methodologies"]`
  - `useMethodologyDetail(slug)` — query key: `["methodologies", slug]`
  - `useMethodologyTools(slug)` — query key: `["methodologies", slug, "tools"]`
- [x] `components/common/methodology-card.tsx` — methodology summary card for browsing `[P1]`

### Verification

- [x] Verify: wizard renders correct step based on `current_step` `[P1]`
- [x] Verify: completed steps show checkmarks, allow revisiting `[P1]`
- [x] Verify: skip onboarding redirects to `/` with `status: "skipped"` `[P1]`
- [ ] Verify: methodology import from quiz works (valid share_id → matched methodology + confidence) `[P1]` _(requires live quiz share_id from public site — not testable in dev)_
- [x] Verify: roadmap displays age-appropriate recommendations per bracket `[P1]`
- [x] Verify: starter curriculum cards display for selected methodology `[P1]`
- [x] Verify: `npm run type-check` passes `[P1]`

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

- [x] `hooks/use-family.ts` — wraps family API endpoints: `[P1]`
  - `useFamilyProfile()` — `GET /families/profile` (query key: `["family", "profile"]`)
  - `useUpdateFamilyProfile()` — `PATCH /families/profile`
  - `useStudents()` — `GET /families/students` (query key: `["family", "students"]`)
  - `useCreateStudent()` — `POST /families/students`
  - `useUpdateStudent(id)` — `PATCH /families/students/{id}`
  - `useDeleteStudent(id)` — `DELETE /families/students/{id}`
  - `useFamilyTools()` — `GET /families/tools` (query key: `["family", "tools"]`)
  - `useStudentTools(id)` — `GET /families/students/{id}/tools` (query key: `["family", "students", id, "tools"]`)

### Settings Pages (`features/settings/`)

- [x] `family-settings.tsx` — main settings page: `[P1]`
  - Edit family display name, state, location region
  - Change primary methodology (PATCH `/families/methodology`)
  - Manage secondary methodology slugs
  - View subscription tier
- [x] `student-management.tsx` — student CRUD embedded in settings (as StudentsTab in family-settings.tsx): `[P1]`
  - List students with edit/delete
  - Add student form (requires COPPA consent)
  - Per-student methodology override
  - Per-student tool display
- [x] `notification-prefs.tsx` — notification preference management: `[P1]`
  - Per-type per-channel toggle grid (in-app column, email column)
  - Digest frequency with preview text explaining each option (immediate/daily/weekly/off)
  - Lock icon on system-critical non-toggleable notifications
  - "Opt out of all non-essential email" batch toggle
  - CAN-SPAM compliant unsubscribe mechanism
  - (SPEC §13.3, domain spec `specs/domains/08-notify.md`)
- [x] `subscription-upgrade.tsx` — minimal upgrade flow: tier comparison, "Subscribe" CTA linking to checkout (Hyperswitch). Required for `<TierGate>` to have a working destination. `[P1]`
- [ ] `subscription-manager.tsx` — full subscription management: `[P2]`
  - Current plan display
  - Upgrade/downgrade flow with downgrade consequence warning modal listing: "Learning data preserved", "Premium tools become read-only", "Compliance reports remain downloadable", "AI recommendations disabled". Explicit confirmation checkbox required.
  - Billing cycle info
  - Payment method management
  - (SPEC §10, §15.3, domain spec `specs/domains/10-billing.md`)
- [x] `account-settings.tsx` — account-level settings (`/settings/account`): email, password change, linked OAuth accounts `[P1]`
- [x] `privacy-controls.tsx` — per-field visibility toggles for family profile (friends-only / hidden); controls what other families can see (SPEC §5.2) `[P1]`

### Co-Parent Management

- [x] `features/settings/co-parent-management.tsx` — co-parent invite + management (as CoParentsTab in family-settings.tsx): `[P1]`
  - Invite by email (sends invitation)
  - List co-parents with role badges (primary / co-parent)
  - Remove co-parent with confirmation dialog
  - Transfer primary parent role (with billing responsibility warning)
- [x] Primary parent indicator badge in family settings showing who holds billing responsibility `[P1]`

### Billing & Payments

- [ ] `features/billing/pricing-page.tsx` — tier comparison table, monthly/annual toggle, feature matrix with CTA buttons `[P2]`
- [ ] `features/billing/payment-methods.tsx` — list saved payment methods, add/remove, set default (Hyperswitch integration) `[P2]`
- [ ] `features/billing/transaction-history.tsx` — purchases, subscription payments, creator payouts with date range filter `[P2]`

### Data Lifecycle

- [x] `features/settings/data-export.tsx` — data export request UI: format selection (JSON/CSV), domain selector checkboxes (learn, social, comply, etc.), async polling (5-second interval via `useExportStatus()`), download link with expiry countdown timer, past exports list. Hooks: `useRequestExport()`, `useExportStatus()`, `useExportList()` `[P1]`
- [x] `features/settings/account-deletion.tsx` — account deletion flow: consequences display, export offer, grace period countdown timer (days/hours remaining), confirmation input (type family name), cancellation option during grace period. `useDeletionStatus()` hook polls for grace period state. Hooks: `useRequestDeletion()`, `useCancelDeletion()`, `useDeletionStatus()` `[P1]`
- [x] `features/settings/student-deletion.tsx` — student profile deletion: separate from full family deletion, COPPA-compliant immediate deletion option (no grace period for child data) `[P1]`

### Notification Center

- [x] `hooks/use-notifications.ts` — notification queries + mutations: `[P1]`
  - `useNotifications()` — paginated notification list
  - `useUnreadCount()` — unread notification count (for bell badge)
  - `useMarkRead(id)` — mark single notification read
  - `useMarkAllRead()` — mark all read
- [x] `components/layout/notification-bell.tsx` — bell icon in header with unread count badge `[P1]`
- [x] `features/settings/notification-center.tsx` — dropdown/panel listing recent notifications with type-specific rendering: `[P1]`
  - Each notification type renders with: category icon (Lucide), action text, deep link to related content, timestamp
  - `friend_request_received` → user avatar + "sent you a friend request" + Accept/Decline buttons
  - `friend_request_accepted` → user avatar + "accepted your friend request" + View Profile link
  - `message_received` → user avatar + message preview + link to conversation
  - `content_flagged` → warning icon + "Your [post/comment] was flagged for review" + link to content
  - `event_cancelled` → calendar icon + "Event [name] was cancelled" + link to events
  - System notifications (ToS update, maintenance) → system icon + non-dismissable until acknowledged
- [x] `features/settings/notification-history.tsx` — full notification history page (`/settings/notifications/history`) with filters by type, date range, and read/unread status `[P1]`

### Notification Type → Phase Mapping

Notification types delivered via WebSocket + notification center, phased as follows:

- **Phase 1 (P1)**: `friend_request_received`, `friend_request_accepted`, `message_received`, `content_flagged`, `event_cancelled`
- **Phase 2 (P2)**: `purchase_completed`, `review_received`, `subscription_created`, `subscription_cancelled`, `subscription_renewed`, `streak_milestone`, `learning_milestone`, `attendance_threshold_warning`, `payout_completed`
- Source: 08-notify §9

### Free Tier Verification

- [ ] Verify free-tier features accessible without premium subscription: social feed, basic learning tools (activity log, journals, reading lists), marketplace browse/purchase, methodology selection, discovery quiz import, onboarding, data export. Premium-gated features (compliance, recommendations, advanced analytics) show `<TierGate>` upgrade prompt. (SPEC §15.1) `[P1]`

### API Schema Prerequisite

- [x] Run `make full-generate` to pull in notification + billing + social endpoints (they exist in Go backend but are not yet in `schema.ts`) `[P1]`

### Verification

- [ ] Verify: family profile edits persist and reflect in UI `[P1]`
- [ ] Verify: student CRUD works, COPPA gate enforced `[P1]`
- [ ] Verify: methodology change updates tools across the app `[P1]`
- [ ] Verify: notification preferences save correctly `[P1]`
- [ ] Verify: co-parent invite sends and co-parent appears in list `[P1]`
- [ ] Verify: data export request completes and download works `[P1]`
- [x] Verify: account deletion flow shows consequences and respects grace period `[P1]`
- [x] Verify: `npm run type-check` passes `[P1]`

### References
- SPEC §1.4 (family management), §8 (notifications), §10 (billing), §15 (data lifecycle)
- Domain specs: `specs/domains/01-iam.md`, `specs/domains/08-notify.md`, `specs/domains/10-billing.md`, `specs/domains/15-data-lifecycle.md`

---

## Phase 8: Learning Tools & Progress

**Goal**: Build the full learning domain — activity logging, journals, reading
lists, progress views, quiz player, video player, content viewer, and sequence
engine. This is the largest feature surface.

**Why here**: Core value proposition. Depends on auth, methodology context,
and family management from prior phases.

### Prerequisites

- [x] Run `make full-generate` if not already done — learning endpoints must be in `schema.ts` `[P1]`

### Learning Hooks (`hooks/`)

- [x] `use-activities.ts` — activity definitions + activity log CRUD: `[P1]`
  - `useActivityDefs()`, `useCreateActivityDef()`, `useLogActivity()`, etc.
  - Query keys: `["learning", "activity-defs"]`, `["learning", "activity-log", filters]`
- [x] `use-journals.ts` — journal entry CRUD: `[P1]`
  - `useJournalEntries(filters)`, `useCreateJournalEntry()`, `useUpdateJournalEntry()`, etc.
  - Query keys: `["learning", "journals", filters]`
- [x] `use-reading.ts` — reading items, progress, lists: `[P1]`
  - `useReadingItems()`, `useReadingLists()`, `useUpdateReadingProgress()`, etc.
  - Query keys: `["learning", "reading-items"]`, `["learning", "reading-lists"]`
- [x] `use-progress.ts` — student progress aggregation: `[P1]`
  - `useStudentProgress(studentId)` — activity counts, hours, reading completion
  - Query key: `["learning", "progress", studentId]`
- [x] `use-quiz.ts` — quiz session management: `[P1]`
  - `useQuizSession(sessionId)`, `useSubmitAnswer()`, `useCompleteQuiz()`, etc.
  - Query keys: `["learning", "quiz", sessionId]`
- [x] `use-video.ts` — video definitions + progress: `[P1]`
  - `useVideoProgress(videoId)`, `useUpdateVideoProgress()`, etc.
- [x] `use-sequences.ts` — sequence definitions + progress: `[P1]`
  - `useSequenceProgress(progressId)`, `useAdvanceSequence()`, etc.
- [x] `use-assignments.ts` — parent assigns content to students: `[P1]`
  - `useAssignments(studentId)`, `useCreateAssignment()`, etc.
- [x] `use-subjects.ts` — subject taxonomy: `[P1]`
  - `useSubjectTaxonomy()`, `useCreateCustomSubject()`
- [x] `use-assessments.ts` — assessment/test score CRUD: `[P1]`
  - `useAssessments(studentId, filters)`, `useCreateAssessment()`, `useGradingScales()`, etc.
  - Query keys: `["learning", "assessments", studentId]`, `["learning", "grading-scales"]`

### Subject Taxonomy Picker

- [x] `components/common/subject-picker.tsx` — 3-level hierarchical picker (Category → Subject → Topic) with inline custom subject creation; used in activity logging, assessments, journals, schedules `[P1]`

### Learning Pages (`features/learning/`)

- [x] `learning-dashboard.tsx` — overview landing page: `[P1]`
  - Quick stats per student (recent activities, reading progress) using `<StatCard>`
  - Methodology-aware tool labels (via `useMethodologyContext()`)
  - Quick-add buttons for logging, journaling, etc.
  - Premium features gated with `<TierGate>`
- [x] `activity-log.tsx` — log and browse activities: `[P1]`
  - Activity log table/list with date, subject, duration, student
  - Add activity form: title, description, subject tags (from taxonomy via `<SubjectPicker>`), date, duration, student selector
  - Filter by student, date range, subject
- [x] `journal-list.tsx` — browse journal entries: `[P1]`
  - List view with entry type badge (freeform/narration/reflection)
  - Filter by student, type, date range
- [x] `journal-editor.tsx` — create/edit journal entry: `[P1]`
  - Rich text editor (uses `rich-text-editor.tsx` component)
  - File attachment support (uses `file-upload.tsx` component)
  - Student selector
  - Subject tags via `<SubjectPicker>`
- [x] `reading-lists.tsx` — manage reading lists and items: `[P1]`
  - List view with status badges (to_read/in_progress/completed)
  - Add book form: title, author, ISBN (with auto-populate lookup, fallback to manual entry `[P2]`), student
  - Status transition buttons
  - Reading list grouping/organization
  - Reading list sharing: share button → select friends/groups → shareable link, recipients can view or copy list `[P2]`
- [x] `progress-view.tsx` — per-student progress dashboard (`/learning/progress/:studentId`): `[P1]`
  - Activity counts by subject
  - Reading completion metrics
  - Hours per week chart
  - Assessment scores overview
  - Export button (async export generation via data lifecycle hooks)
- [x] `tests-and-grades.tsx` — assessment entry and grade tracking: `[P1]`
  - Assessment entry form: title, subject (via `<SubjectPicker>`), student, date, score type (points/percentage/letter), weight
  - Grading scale configuration (per-family custom scales)
  - Running averages by subject and student
  - Hooks: `useAssessments()`, `useGradingScales()`
- [ ] `projects.tsx` — project creation and tracking: `[P2]`
  - Project creation with milestones (title, description, due date)
  - Status lifecycle: planning → in_progress → completed
  - Per-student project list with progress indicators

### Methodology Integration

- [ ] All learning tool labels MUST come from methodology config via `useMethodologyContext()` — no hardcoded "Activity Log", "Journal", etc. `[P1]` *(labels currently use i18n strings; methodology-config integration deferred until methodology context hook is built)*
- [x] `components/common/parent-education-panel.tsx` — expandable guidance panel sourcing content from `ActiveToolResponse.guidance`; includes "Why this tool?" explanation from methodology philosophy `[P1]`

### Tool Assignment

- [ ] `features/learning/tool-assignment.tsx` — per-student tool activation/deactivation with methodology defaults; parent can show/hide tools per student `[P2]`

### Interactive Learning Players

- [x] `quiz-player.tsx` — interactive quiz (`/learning/quiz/:sessionId`): `[P1]`
  - Session lifecycle: `not_started → in_progress → submitted → scored`
  - Question types: multiple choice, fill-in-the-blank, true/false, matching, ordering, short answer
  - Auto-save on each answer for save-and-resume (no data loss on browser crash/navigation)
  - "Save & Exit" button preserving progress; returning restores exact state
  - Short-answer questions show "Awaiting parent review" instead of auto-score
  - Score display on completion
  - `aria-live` for quiz feedback (CODING_STANDARDS §3.8)
  - (SPEC §8.1.9)
- [x] `parent-quiz-scoring.tsx` — parent scoring interface for short-answer questions: `[P1]`
  - Pending-review list with notification badge on learning dashboard when reviews are pending
  - Question + student response display
  - Score input (correct / partial / incorrect)
  - "Score All" batch action for multiple pending answers
  - (SPEC §8.1.9)
- [x] `video-player.tsx` — video playback (`/learning/video/:videoId`): `[P1]`
  - HLS streaming support + external video URLs
  - Progress tracking (last position, completion percentage)
  - Accessible controls
  - Caption file support (VTT/SRT) `[P1]`
  - Caption language selection dropdown when multiple tracks are available (SPEC §17.6.2) `[P1]`
  - Caption styling options (font size, background opacity) stored in localStorage `[P2]`
- [x] `content-viewer.tsx` — document/content viewer (`/learning/read/:contentId`): `[P1]`
  - PDF/document rendering
  - Progress tracking
- [x] `sequence-view.tsx` — lesson sequence (`/learning/sequence/:progressId`): `[P1]`
  - Linear progression display
  - Current step highlight
  - Unlock logic visualization
  - Navigation between sequence items
  - Parent override controls: "Skip" button (marks item skipped, advances), "Unlock" button (unlocks ahead of progression). Confirmation dialog explaining override; override actions logged. (SPEC §8.1.12)

### Content Assignment UX

- [x] Content assignment notification: confirmation toast for parent on successful assignment, "New" badge on student dashboard for unstarted assignments (SPEC §6.5) `[P1]`
- [x] Glassmorphism progress overlay for student sessions — semi-transparent `secondary-container` background + backdrop blur showing session progress, time remaining, and current activity (DESIGN §2.6) `[P1]`

### Student Features (`features/student/`)

- [x] `student-dashboard.tsx` — simplified student home: `[P1]`
  - Assigned content list with "New" badge for unstarted assignments
  - Current sequence progress
  - No social/marketplace access
- [x] `student-quiz.tsx` — student-facing quiz (simplified wrapper of quiz-player) `[P1]`
- [x] `student-video.tsx` — student-facing video player `[P1]`
- [x] `student-reader.tsx` — student-facing content viewer `[P1]`
- [x] `student-sequence.tsx` — student-facing sequence progression `[P1]`

### Student Session Management

- [x] `hooks/use-student-session.ts` — manages which student is active: `[P1]`
  - Parent switches between students for logging/viewing
  - Student shell: student is fixed from parent's selection
  - Stored in context (not server state)
  - (ARCHITECTURE §11.2)
- [x] Supervised student session detail: age gate UI (10+ verification), session creation flow (parent selects student → confirms → enters student shell), session duration presets (1h / 2h / 4h / end-of-day), timeout warning at 5 minutes remaining, session revocation from parent view, session activity log visible to parent `[P1]`
- [x] `features/learning/student-session-activity-log.tsx` — parent-visible log of all actions taken during a student session (pages visited, content viewed, time per item) (SPEC §6.5) `[P1]`

### Learning Data Export

- [x] Learning data export button on `progress-view.tsx` — triggers domain-scoped export via data lifecycle hooks `[P1]`

### Streak & Milestone Display

- [x] Streak indicator on learning dashboard: flame/star icon + day count, milestone badges at 7/14/30/60/100 days `[P1]`
- [ ] Milestone celebration toast on WebSocket event (`streak_milestone`, `learning_milestone`) `[P1]` *(requires WebSocket infrastructure from Phase 5)*
- Source: SPEC §13.1, 08-notify §9

### Phase 3 Methodology-Specific Tools (future placeholders)

These tools are methodology-specific extensions. Listed here as `[P3]` placeholders to ensure they are tracked:

- [ ] Nature journal (Charlotte Mason) — `observation_type`, species, weather, drawing/photo upload `[P3]`
- [ ] Trivium tracker (Classical) — grammar/logic/rhetoric stage tracking per subject `[P3]`
- [ ] Rhythm planner (Waldorf) — day-of-week time blocks, rhythm templates `[P3]`
- [ ] Observation logs (Montessori) — work chosen, duration, concentration level `[P3]`
- [ ] Habit tracking (Charlotte Mason) — habit goals, streaks, parent notes `[P3]`
- [ ] Interest-led activity log (Unschooling) — auto-tagging activities to subjects `[P3]`
- [ ] Handwork project tracker (Waldorf) — materials, techniques, photos `[P3]`
- [ ] Practical life activities (Montessori) — life skill categories, mastery levels `[P3]`

### Verification

- [ ] Verify: activity logging creates entries and appears in log `[P1]`
- [ ] Verify: journal creation with attachments works `[P1]`
- [ ] Verify: reading list status transitions work correctly `[P1]`
- [ ] Verify: progress view aggregates data accurately `[P1]`
- [ ] Verify: quiz player handles full session lifecycle `[P1]`
- [ ] Verify: video player tracks progress `[P1]`
- [ ] Verify: sequence navigation enforces unlock logic `[P1]`
- [ ] Verify: student shell restricts navigation `[P1]`
- [ ] Verify: methodology terminology used throughout (not hardcoded labels) `[P1]`
- [ ] Verify: assessment entry with grading scales works `[P1]`
- [ ] Verify: subject taxonomy picker supports 3-level hierarchy + custom subjects `[P1]`
- [ ] Verify: `npm run type-check` passes `[P1]`

### References
- SPEC §6 (learning requirements)
- Domain spec: `specs/domains/06-learn.md`
- Backend TODO: `specs/TODO-06-learn.md` (all backend batches complete)
- ARCHITECTURE §11.3 (learning routes)

---

## Phase 9: Social, Marketplace, Search & Admin

**Goal**: Build the social feed, messaging, groups, events, marketplace browse/
purchase flow, creator tools, global search, and core admin/moderation. These are
the community features plus the moderation tools needed to operate them safely.

**Why here**: These are large feature surfaces that depend on the foundation
from Phases 1–7 but are independent of Phase 8 (learning). Admin/moderation is
co-located here because reporting without review is an incomplete workflow.

### Prerequisites

- [x] Run `make full-generate` — social, marketplace, and search endpoints must be in `schema.ts` `[P1]`
  - Note: No swagger annotations exist for social/mkt/search/admin/safety handlers; TypeScript interfaces defined inline in hook files matching Go backend structs

### WebSocket Infrastructure

> **Moved to Phase 5.** WebSocket connection manager (`lib/websocket.ts`) and the
> `use-websocket.ts` hook are implemented as part of Phase 5's "WebSocket Foundation"
> subsection. Social and messaging features in this phase consume that foundation.

### Social Features (`features/social/`)

- [x] `feed.tsx` — social feed (index route `/`): `[P1]`
  - Reverse-chronological posts from friends only
  - Post type-specific rendering for all 6 types: text, photo, milestone, event_share, marketplace_review, resource_share
  - Post cards with type-specific UI variants
  - Infinite scroll / pagination (uses `infinite-scroll.tsx`)
  - `aria-live` region for new posts
- [x] `post-composer.tsx` — create new post: `[P1]`
  - Type selector with type-specific UI variants per post type
  - Text input + photo upload (for photo posts)
  - Student mention (for milestone posts)
  - Post visibility indicator (friends icon / group icon)
- [x] `post-detail.tsx` — single post view with comments: `[P1]`
  - Like button (one like per family per post, toggle)
  - Comment threading: reply button, nested replies (one level), visual indentation
  - Comment composer
  - Post visibility indicator
- [ ] Post edit action for author family — pencil icon, inline editor, "(edited)" timestamp indicator `[P2]`
- [ ] Post delete action with confirmation dialog — removes post and all comments `[P1]`
- [ ] Comment edit/delete by comment author; post author can delete any comment on their post `[P1]`
- [x] `friends-list.tsx` — friend management (`/friends`): `[P1]`
  - Friends list
  - Pending requests (incoming/outgoing)
  - Friend search (by display name)
  - Block (silent — blocked user sees no change)
- [ ] `friend-discovery.tsx` — find new friends: `[P1]`
  - Methodology match suggestions
  - Location-based suggestions (if location sharing enabled)
  - Shared groups indicator
  - Name search
- [x] `block-management.tsx` — block list in settings (view blocked users, unblock) `[P1]`
- [ ] Unfriend action with confirmation dialog (silent, no notification sent) `[P1]`
- [x] `direct-messages.tsx` — DM inbox (`/messages`): `[P1]`
  - Conversation list (friends only, parent-to-parent)
  - Unread indicators
  - Real-time via WebSocket
- [x] `conversation.tsx` — DM thread (`/messages/:conversationId`): `[P1]`
  - Message list with timestamps
  - Message composer
  - Real-time message delivery
  - Image attachment button using `file-upload.tsx`, single image per message `[P1]`
  - Inline image preview in message bubble, tap for lightbox (SPEC §7.5) `[P1]`
- [ ] Conversation mute toggle — bell-off icon, server-persisted mute state, muted conversations show muted indicator in inbox (SPEC §5.4) `[P1]`
- [ ] Message search within conversation — debounced search input, `Ctrl+F` keyboard shortcut, highlights matching messages (SPEC §5.4) `[P1]`
- [x] `groups-list.tsx` — group directory (`/groups`): `[P1]`
  - Platform-managed groups (by methodology) + user-created
  - Join/leave functionality
- [x] `group-detail.tsx` — group page (`/groups/:groupId`): `[P1]`
  - Group feed
  - Member list
  - Group info
- [ ] `group-creation.tsx` — create new group: name, description, join policy (open / request / invite-only) `[P2]`
- [ ] `group-management.tsx` — group admin: promote moderator, remove member, pin posts, approve join requests `[P2]`
- [ ] Group role management UI — owner → moderator → member hierarchy display, role badges, role change confirmation dialogs (SPEC §5.5) `[P2]`
- [x] `events-list.tsx` — events directory (`/events`): `[P1]`
  - Event cards with RSVP 3-state button (going / interested / not going)
  - Capacity indicator: "X of Y spots" / "Full" badge (disable RSVP "going" when full)
  - Virtual event: video call link shown only to RSVPed attendees
  - Filter by date, location region, methodology tag
- [ ] `event-creation.tsx` — create event: title, description, date/time, location type selector (in-person / virtual / hybrid), virtual meeting URL field, capacity number input (optional), visibility (friends / group), methodology tag, group-linked option `[P1]`
- [ ] Event attendee list for organizer — RSVP list with going/interested/not-going counts, attendee names, CSV export of attendee list (SPEC §5.6) `[P1]`
- [ ] Event cancellation with attendee notification confirmation dialog `[P1]`
- [ ] Recurring events support (weekly/monthly/custom) `[P2]`
- [x] `family-profile.tsx` — public family profile (`/family/:familyId`): `[P1]`
  - Friends-only visibility
  - Family info, methodology, member count
- [ ] Report button component — reusable "Report" action for posts, comments, messages, listings (uses `report-button.tsx` from Phase 3) (SPEC §11) `[P1]`

### Moderation Appeals (`features/settings/`)

- [x] `moderation-appeals.tsx` — moderation appeals UI at `/settings/account/appeals`: `[P1]`
  - List of moderation actions taken against user's content
  - Appeal form: free-text explanation (one appeal per action)
  - Status tracker: pending → in_review → granted / denied
  - Resolution notification
  - Hooks: `useMyModerationActions()`, `useSubmitAppeal()`, `useAppealStatus()`
  - (SPEC §12.2, 11-safety §12.4)

### Location Features

- [ ] Location sharing toggle in profile settings (opt-in, never stores GPS coordinates — region only) `[P1]`
- [ ] Nearby discovery sections in friends/groups/events browsing (when location sharing enabled) `[P1]`

### Marketplace Features (`features/marketplace/`)

- [x] `marketplace-browse.tsx` — browse listings (`/marketplace`): `[P1]`
  - Faceted filtering: methodology, subject, grade, price, rating, content type, worldview
  - Full-text search
  - Curated sections: Featured, Trending, New Arrivals, Staff Picks
  - Sort: relevance, price, rating, recency
  - Worldview tag badges on listing cards + worldview filter option
- [x] `listing-detail.tsx` — listing page (`/marketplace/listings/:id`): `[P1]`
  - Full listing info, preview, reviews (using `star-rating.tsx`)
  - Add to cart button
  - Verified-purchaser review display (1-5 stars, anonymous by default)
  - Listing lifecycle state badge (draft / submitted / published / archived)
  - Content licensing badge (license type label + info tooltip explaining usage rights) (SPEC §9.2) `[P1]`
- [x] `cart.tsx` — shopping cart (`/marketplace/cart`): `[P1]`
  - Cart items, quantities, total
  - Checkout flow
  - Cart groups bundle items with discount applied (SPEC §9.4) `[P2]`
- [ ] Content bundle purchase: bundle badge on listing cards, "Buy Bundle" CTA on detail page (SPEC §9.4) `[P2]`
- [x] `purchase-history.tsx` — past purchases (`/marketplace/purchases`): `[P1]`
  - Purchase list with download links
  - Content access
- [x] `refund-request.tsx` — refund flow (`/marketplace/purchases/:id/refund`): refund reason selector (dropdown), 7-day eligibility window check, refund status tracking (pending → approved → processed / denied), confirmation dialog (SPEC §9.5) `[P1]`

### Creator Features (`features/marketplace/creator/`)

- [x] `creator-dashboard.tsx` — creator home (`/creator`): `[P1]`
  - Sales overview, earnings, payout status
  - Analytics: sales chart (line/bar, date range selector), earnings breakdown by listing, payout schedule with next payout date, per-listing metrics (views, purchases, ratings) (SPEC §9.6)
- [ ] `payout-setup.tsx` — creator payout onboarding (`/creator/payouts`): payout method selection + account setup, payout history, minimum threshold display (SPEC §9.6) `[P2]`
- [ ] `creator-verification.tsx` — creator identity verification: legal name, tax info (masked SSN/EIN), verification status indicator. Required before first payout. (SPEC §9.1, 07-mkt §11) `[P2]`
- [x] `create-listing.tsx` — new listing (`/creator/listings/new`): `[P1]`
  - Listing form: title, description, price, category, content upload
  - Preview before publish
  - Listing lifecycle state transition buttons (draft → submitted → published → archived)
  - Note: creators can create listings without identity verification in Phase 1. Verification (`creator-verification.tsx` [P2]) gates **payouts only**, not listing creation. Unverified creators see a persistent banner: "Complete verification to receive payouts."
- [x] `edit-listing.tsx` — edit existing listing (`/creator/listings/:id/edit`) `[P1]`
- [ ] `listing-version-history.tsx` — version list with upload date, file size, "current" badge. View-only (no rollback in v1). (SPEC §9.2.3) `[P2]`
- [ ] `creator-reviews.tsx` — view reviews on own listings, respond to reviews `[P2]`
- [x] `quiz-builder.tsx` — create quizzes for marketplace (`/creator/quiz-builder`): `[P1]`
  - Question editor (multiple types)
  - Preview and test
  - Keyboard alternative for drag-and-drop: arrow keys to select question, Enter to grab, arrow keys to move, Enter to drop, Escape to cancel. `aria-live` announcements for position changes (CODING_STANDARDS §3.8)
- [x] `sequence-builder.tsx` — create lesson sequences (`/creator/sequence-builder`): `[P1]`
  - Step editor with ordering
  - Content assignment per step
  - Keyboard alternative for reordering: arrow keys to select step, Enter to grab, arrow keys to move, Enter to drop, Escape to cancel. `aria-live` announcements for position changes (CODING_STANDARDS §3.8)

### Search (`features/search/`)

- [x] `search-results.tsx` — global search results (`/search`): `[P1]`
  - Scope switching tabs: Social / Marketplace / Learning (family-scoped)
  - Faceted filtering for marketplace results (uses `faceted-filter.tsx`)
  - Sort: relevance, price, rating, recency
- [ ] `components/layout/search-bar.tsx` — persistent search input in header/sidebar: `[P1]`
  - Debounced autocomplete (300ms delay, top 5 suggestions, keyboard navigation)
  - Navigates to `/search?q=...`
  - (SPEC §12, domain spec `specs/domains/12-search.md`)

### Admin & Moderation (`features/admin/`) — Admin Only

> Moved from Phase 10: reporting without review is an incomplete workflow.
> Day-1 moderation capability is essential alongside social/marketplace launch.

- [x] `admin-dashboard.tsx` — system health overview (`/admin`): `[P1]`
  - User counts, content stats, system metrics
  - `<AdminGuard>` wrapper
- [x] `user-management.tsx` — user admin: `[P1]`
  - Search users by email/name, filter by status
  - View family details
  - Account actions: suspend (with reason), ban (with reason), reactivate
- [x] `user-detail.tsx` — individual user detail view (`/admin/users/:id`) `[P1]`
- [x] `moderation-queue.tsx` — content moderation: `[P1]`
  - Reported content queue (from safety domain reports)
  - Content preview panel (shows reported content inline without navigating away)
  - Review + action with reasons dropdown (approve, remove, warn) + action reason selection
  - Bulk actions for multiple reports (select all, bulk approve/remove)
  - Admin notes field per moderation action (internal, not visible to content owner)
  - Moderation states visible to content owners
  - **Appeals tab**: pending appeals with original action context, appeal text, Grant/Deny actions. Different admin than original action (enforced by backend, UI shows warning if same admin). (SPEC §12.2, 11-safety §12.4) `[P1]`
- [x] `audit-log.tsx` — admin audit log viewer: filterable by admin user, action type, target entity, date range `[P1]`
- References: SPEC §16, domain spec `specs/domains/16-admin.md`

### Verification

- [ ] Verify: social feed displays posts from friends only `[P1]`
- [ ] Verify: all 6 post types render with type-specific UI `[P1]`
- [ ] Verify: like toggle and comment threading work `[P1]`
- [ ] Verify: DM real-time delivery works via WebSocket `[P1]`
- [ ] Verify: marketplace filtering and search work `[P1]`
- [ ] Verify: cart → purchase flow completes `[P1]`
- [ ] Verify: creator listing authoring works `[P1]`
- [ ] Verify: search returns results across all scopes with debounced autocomplete `[P1]`
- [ ] Verify: report buttons present on all user content `[P1]`
- [ ] Verify: no public profiles — friends-only visibility enforced `[P1]`
- [ ] Verify: RSVP state persists and updates event attendee count `[P1]`
- [ ] Verify: moderation queue receives reports and admin actions persist `[P1]`
- [ ] Verify: audit log records admin actions with correct metadata `[P1]`
- [x] Verify: `npm run type-check` passes `[P1]`

### References
- SPEC §5 (social), §7 (marketplace), §12 (search), §16 (admin)
- Domain specs: `specs/domains/05-social.md`, `specs/domains/07-mkt.md`, `specs/domains/12-search.md`, `specs/domains/16-admin.md`
- ARCHITECTURE §11.3 (routes), §11.5 (WebSocket)

---

## Phase 10: Compliance, Planning, Polish & Quality Gates

**Goal**: Build premium compliance features, calendar/planning, remaining admin
config, recommendations, and complete all cross-cutting polish work. Final quality
verification.

**Why last**: These features are either premium-gated or polish tasks that should
only happen after core features are stable. Core admin/moderation moved to Phase 9.

### Compliance Features (`features/compliance/`) — Premium Only

- [ ] `compliance-setup.tsx` — state compliance configuration: `[P2]`
  - Select state requirements
  - Configure tracking thresholds
  - `<TierGate>` wrapper for free-tier users
- [ ] `attendance-tracker.tsx` — daily attendance marking: `[P2]`
  - Per-student daily attendance with 4 states: present / absent / partial / excused
  - Summary with threshold tracking
  - Calendar heatmap view (color-coded by status per student) with color legend
  - Attendance threshold/pace indicator (ahead / on-track / behind) with progress bar vs state requirement
  - Auto-generated attendance entries from logged learning activities with visual indicator distinguishing auto vs manual entries
  - `aria-live` announcements for attendance state changes
- [ ] `assessment-records.tsx` — assessment record management: `[P2]`
  - Link assessments to compliance requirements
  - Score tracking
- [ ] `standardized-tests.tsx` — standardized test score entry form (title, test name, date, scores by section) `[P2]`
- [ ] `portfolio-list.tsx` — portfolio management: `[P3]`
  - List portfolios per student
  - Create new portfolio (select student + date range + template)
- [ ] `portfolio-builder.tsx` — portfolio construction and PDF generation: `[P3]`
  - Item selection UI with filters (by subject, date range, type)
  - Organization type selector (chronological / by-subject / by-type)
  - Drag/arrange portfolio sections (work samples, assessments, attendance) with keyboard alternative (arrow keys + Enter/Escape)
  - Cover page customization (student name, date range, family logo optional)
  - Preview modal showing formatted portfolio before generation
  - Status lifecycle: draft → generating → ready
  - Generate + download PDF
  - Print-ready layout (uses `print.css` tokens)
- [ ] `transcript-list.tsx` — transcript management per student `[P3]`
- [ ] `transcript-builder.tsx` — transcript construction: `[P3]`
  - Course entry with level selector: regular / honors / AP / dual-enrollment
  - Multi-semester tab navigation
  - Weighted GPA calculation: honors +0.5, AP/dual-enrollment +1.0 weighting
  - Multi-method GPA toggle: 4.0 scale / percentage / pass-fail display
  - `aria-live` announcements for GPA recalculations on grade/course changes
  - PDF generation + print-ready layout
- References: SPEC §14, domain spec `specs/domains/14-comply.md`

### Planning & Calendar (`features/planning/`)

- [ ] `calendar-view.tsx` — unified calendar (`/calendar`): `[P1]`
  - Synthesize: learning activities + social events + compliance attendance
  - Daily/weekly view toggle (`/calendar/day/:date`, `/calendar/week/:date`)
  - Three data sources with distinct color coding: learning (tertiary), events (primary), attendance (secondary) + color legend
  - Print-friendly output (MUST be printable — SPEC §17)
- [ ] `schedule-editor.tsx` — schedule item CRUD form: `[P1]`
  - Fields: title, description, student, date, time, duration, category enum, subject (via `<SubjectPicker>`), color, notes
  - Schedule completion checkbox + auto-log workflow: completion checkbox prompts "Log as learning activity?" → auto-creates `learn::` activity with pre-populated fields, links via `linked_activity_id` (17-planning §3.1)
- [ ] Drag-to-schedule with keyboard alternative (arrow keys + Enter to place items) `[P1]`
- [ ] Print-friendly schedule output (separate from calendar print) `[P2]`
- [ ] `schedule-templates.tsx` — recurring schedule templates (weekly patterns, methodology-specific defaults) `[P2]`
- [ ] Co-op coordination view (shared schedules between families in a group) `[P2]`
- [ ] Schedule sharing/export (CSV, iCal formats) `[P2]`
- References: SPEC §17, domain spec `specs/domains/17-planning.md`

### Admin Config (`features/admin/`) — Admin Only

> Core admin (dashboard, user management, moderation queue, audit log) moved to Phase 9.
> These remaining items are P2 configuration features.

- [ ] `feature-flags.tsx` — feature flag management: toggle, rollout percentage, family whitelist `[P2]`
- [ ] `methodology-config.tsx` — methodology configuration editing (tool labels, descriptions, philosophy text) `[P2]`
- References: SPEC §16, domain spec `specs/domains/16-admin.md`

### Recommendations (`features/recommendations/`)

- [ ] Recommendation cards/carousel on learning dashboard (content suggestions based on methodology + student profile) `[P2]`
- [ ] Recommendation preferences: dismiss individual recommendation, block category, undo dismiss `[P2]`
- [ ] Transparency labels: "Why recommended?" expandable explanation + AI-generated content badge where applicable `[P2]`
- References: SPEC §13, domain spec `specs/domains/13-recs.md`

### Data Lifecycle

Data export, account deletion, and student deletion are implemented in **Phase 7** under
"Data Lifecycle" (see `features/settings/data-export.tsx`, `account-deletion.tsx`,
`student-deletion.tsx`). Phase 10 verification ensures they integrate correctly with
compliance exports and admin oversight.

- References: domain spec `specs/domains/15-data-lifecycle.md`

### Cross-Cutting Polish

- [ ] Responsive audit — verify all pages work at all breakpoints (sm/md/lg/xl/2xl/3xl) `[P1]`
- [ ] Touch target audit — verify all interactive elements ≥ 44×44px below `md` breakpoint `[P1]`
- [ ] Focus management audit — verify focus moves to `<h1>` on every route change `[P1]`
- [ ] Screen reader audit — verify `aria-live` regions for all dynamic content: social feed, quiz feedback, notifications, attendance state changes, GPA recalculations, drag-and-drop position changes, search results, form validation errors, export status updates, session timeout warnings `[P1]`
- [ ] Print stylesheet audit — verify print output for all printable pages: compliance docs, schedules, portfolios, transcripts, progress reports `[P1]`
- [ ] `prefers-reduced-motion` audit — verify all animations collapse `[P1]`
- [ ] Surface hierarchy audit — verify no `1px solid` borders, only tonal shifts `[P1]`
- [ ] Token compliance audit — grep for hardcoded hex, arbitrary z-index, Tailwind default palette `[P1]`
- [ ] Dark mode architecture readiness audit — zero `dark:` Tailwind prefixes anywhere in codebase, all colors via token classes only, CSS-only theme switch structure ready (DESIGN_TOKENS §2.9) `[P1]`
- [ ] Parent/student context audit — `data-context` attribute present on layout wrappers, `parent:`/`student:` custom variant selectors functional in all applicable components `[P1]`
- [ ] `aria-live` region audit — comprehensive list of all dynamic content areas that must have `aria-live` regions: form errors, toast notifications, feed updates, quiz scores, search results, export progress, session timers, attendance changes, GPA updates, drag reorder confirmations `[P1]`
- [ ] Skip link audit — verify skip link present and functional on every layout (AppShell, StudentShell, AdminShell, OnboardingLayout, AuthLayout) `[P1]`
- [ ] Drag-and-drop keyboard alternative audit — verify all drag interfaces have keyboard alternatives: quiz-builder, sequence-builder, portfolio-builder, schedule drag-to-schedule, calendar item reorder `[P1]`
- [ ] Image alt text audit — verify all `<img>` elements have meaningful `alt` attributes; decorative images use `alt=""` `[P1]`
- [ ] Print style verification for all printable pages — compliance docs, schedules, portfolios, transcripts, progress reports, calendar views `[P1]`
- [ ] Error boundary coverage — verify all route segments have error boundaries `[P1]`
- [ ] Loading state coverage — verify skeleton/spinner states for all async data `[P1]`
- [ ] Empty state coverage — verify all list views have empty states with CTAs `[P1]`
- [ ] 404 page — friendly not-found page within AppShell `[P1]`
- [ ] i18n string externalization audit — no hardcoded English strings in components `[P1]`
- [ ] axe-core CI integration — Playwright + axe-core in GitHub Actions CI pipeline, zero critical/serious violations, PR comment reporting with violation details `[P1]`
- [ ] 200% zoom verification across all pages (WCAG 1.4.4) `[P1]`
- [ ] Video caption file support verification (VTT/SRT in video player) `[P1]`
- [ ] Community guidelines page exists and linked from report dialog `[P1]`
- [ ] Error retry/offline handling — TanStack Query retry config (3× exponential backoff) + `<NetworkStatus>` banner `[P1]`
- [ ] VPAT (Voluntary Product Accessibility Template) documentation `[P2]`

### Testing Infrastructure

- [ ] Playwright accessibility regression test suite — covers critical user journeys (login, onboarding, activity logging, quiz taking, marketplace purchase, messaging) with axe-core assertions `[P1]`
- [ ] axe-core CI pipeline — GitHub Actions workflow running Playwright a11y tests on every PR, failure blocks merge, PR comment with violation summary and links `[P1]`
- [ ] Screen reader test matrix documentation — document tested combinations (NVDA + Firefox, VoiceOver + Safari, JAWS + Chrome) with pass/fail per critical journey `[P2]`
- [ ] Image alt text automation — ESLint `jsx-a11y` rule enforcement for `img-redundant-alt`, `alt-text`, and `img-has-alt`; CI gate `[P1]`

### Performance

- [ ] Route code-splitting — verify all feature routes lazy-load `[P1]`
- [ ] Image optimization — verify all images use appropriate formats, lazy loading `[P1]`
- [ ] Bundle analysis — check for unexpectedly large dependencies `[P1]`
- [ ] TanStack Query optimization — verify staleTime/gcTime tuned per query type `[P1]`

### Final Quality Gates

- [ ] `npm run type-check` — zero TypeScript errors `[P1]`
- [ ] All pages render without console errors `[P1]`
- [ ] All interactive elements keyboard accessible `[P1]`
- [ ] Lighthouse accessibility score ≥ 90 on all primary pages `[P1]`
- [ ] No `any` types anywhere in codebase (search: `as any`, `: any`) `[P1]`
- [ ] No hardcoded hex colors (search: `#[0-9a-f]`) `[P1]`
- [ ] No `style={{ }}` inline styles `[P1]`
- [ ] No direct `fetch()` calls outside `api/client.ts` `[P1]`
- [ ] No TanStack Query usage outside custom hooks `[P1]`
- [ ] All API types from `src/api/generated/schema.ts` only `[P1]`
- [ ] No hardcoded English strings — all user-facing text from i18n catalogs `[P1]`
- [ ] axe-core: zero critical/serious violations `[P1]`
- [ ] All pages usable at 200% zoom `[P1]`
- [ ] Community guidelines page exists and linked from report dialog `[P1]`

### References
- SPEC §14 (compliance), §16 (admin), §17 (planning), §15 (data lifecycle), §13 (recommendations)
- CODING_STANDARDS §3 (all frontend rules)
- DESIGN_TOKENS §18 (implementation checklist)
- DESIGN §3–§5 (visual rules)

---

## Appendix A: Generated API Schema Coverage

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
| Search | — | Ready (handler + routes wired) — **needs `generate-types` run** |
| Compliance | — | Ready (30+ endpoints wired) — **needs `generate-types` run** |
| Lifecycle | — | Service layer only — **needs handler + routes before Phase 7** |
| Planning | — | Spec complete (`17-planning.md`) — **needs full backend before Phase 10 calendar** |
| Sessions | — | **No list/revoke API** — needs IAM enhancement before Phase 5 `session-management.tsx` |

Before starting any phase that consumes API types beyond domains 01-04, run:
```bash
make full-generate
```

---

## Appendix B: Phase Scope Reference

Maps SPEC.md §19 release phases to this TODO's implementation phases:

| SPEC §19 Phase | Definition | TODO Phases |
|----------------|-----------|-------------|
| **Phase 1 (MVP)** | Core auth (password strength indicator, rate limit feedback, email verification resend, session timeout warning, co-parent invitation acceptance), family management, privacy controls, onboarding, learning tools (activity log, journals, reading, quiz save-and-resume, parent quiz scoring, sequence parent override, streak display, student session activity log), social feed (post/comment deletion, conversation mute/search, DM attachments, event capacity/virtual, event attendee list), messaging, events, marketplace browse/purchase (refund requests, licensing display, creator analytics), moderation appeals, WebSocket foundation, notifications (type icons, moderation preview), admin basics (moderation content preview, appeals queue, async export polling), safety reporting, data export/deletion | TODO Phases 1–9 `[P1]` items + Phase 10 `[P1]` items |
| **Phase 2 (Enhanced)** | Compliance (attendance, assessments), billing/payments, advanced groups, recurring events, schedule templates, feature flags, methodology config, MFA, recommendations, data tables, ISBN book import, reading list sharing, content bundles, creator identity verification, listing version history, COPPA micro-charge, Playwright tests, VPAT | Phase 10 `[P2]` items + scattered `[P2]` items in earlier phases |
| **Phase 3 (Advanced)** | Portfolios, transcripts, GPA calculations, methodology-specific tools (nature journal, trivium tracker, rhythm planner, etc.) | Phase 8 `[P3]` items + Phase 10 `[P3]` items |

**Tag convention**: Every checklist item is tagged `[P1]`, `[P2]`, or `[P3]` to indicate
which SPEC §19 release phase it belongs to. Filter by tag to scope work to a specific release.
