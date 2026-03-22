# Homegrown Academy — Design Token Specification

## §0 Authority & Scope

This document is a **normative companion** to `specs/DESIGN.md` (the creative brief). It
operationalizes every visual decision from DESIGN.md into concrete, enforceable token values.

- RFC 2119 keywords apply throughout: **MUST** / **MUST NOT** are absolute;
  **SHOULD** / **SHOULD NOT** are strong recommendations; **MAY** is discretionary.
- This document has the **same authority** as `specs/CODING_STANDARDS.md`. Violations are bugs.
- **Relationship to other specs:**
  - `specs/DESIGN.md` — aesthetic intent ("what it should feel like"). This doc says *exactly
    what values to use*.
  - `specs/ARCHITECTURE.md` §2.3 — selects Tailwind CSS v4, React 19, Vite. §11.5 — WCAG 2.1
    AA requirements, color contrast, focus management, touch targets.
  - `specs/CODING_STANDARDS.md` §3.6 — forbids inline styles; mandates Tailwind classes. §3.7 —
    accessibility rules. §3.8 — print style requirements.
- **When DESIGN.md and this document conflict, this document wins** — it contains the verified,
  WCAG-compliant values derived from DESIGN.md's creative direction.

### Tailwind v4 Context

Tailwind CSS v4 uses **CSS-native configuration** via `@theme` blocks in CSS files. There is
**no `tailwind.config.ts`** (that is a v3 pattern). All design tokens are CSS custom properties
defined inside `@theme { }` blocks, which Tailwind automatically registers as utility classes.

> **Note**: `specs/ARCHITECTURE.md` §11.1 file tree references `tailwind.config.ts` — this is
> outdated. Tailwind v4 replaces it with `tokens.css` containing `@theme` blocks.

The Vite plugin `@tailwindcss/vite` handles processing (not PostCSS).

---

## §1 File Architecture

### 1.1 CSS File Structure

```
frontend/src/styles/
├── app.css             # Entry point: imports + Tailwind directive
├── tokens.css          # @theme block: all design tokens
├── base.css            # @layer base: @font-face, resets, html/body defaults
├── components.css      # @layer components: type scale classes, shared patterns
├── utilities.css       # @layer utilities: custom utilities (touch-target, etc.)
└── print.css           # @media print: token overrides + structural rules
```

### 1.2 Import Order

`app.css` MUST use this exact import order:

```css
/* app.css — Entry point */
@import "tailwindcss";
@import "./tokens.css";
@import "./base.css";
@import "./components.css";
@import "./utilities.css";
@import "./print.css";
```

**Rule**: `@import "tailwindcss"` MUST be the first directive. Token definitions MUST precede
`base.css` so that base styles can reference token values.

### 1.3 Vite Wiring

The Vite config MUST use `@tailwindcss/vite` (not PostCSS):

```typescript
// vite.config.ts
import tailwindcss from "@tailwindcss/vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  // ...existing config
});
```

The `postcss` and `autoprefixer` devDependencies in `package.json` SHOULD be removed — they
are not needed with `@tailwindcss/vite`.

### 1.4 CSS Entry Point

`main.tsx` MUST import the CSS entry point:

```typescript
import "./styles/app.css";
```

---

## §2 Color Tokens

All colors are derived from `DESIGN.md` §2. Colors not explicitly specified in DESIGN.md are
derived to complete the MD3 tonal system while maintaining the warm, earthy palette.

### 2.1 Surface Palette

| Token | Hex | Source | Usage |
|-------|-----|--------|-------|
| `--color-surface` | `#faf9f5` | DESIGN.md §2 | Page background, base layer |
| `--color-surface-dim` | `#dddcd8` | Derived | Dimmed/recessed surfaces |
| `--color-surface-container-lowest` | `#ffffff` | DESIGN.md §2 | Cards that "lift" against surface |
| `--color-surface-container-low` | `#f5f4f0` | Derived | Sidebar backgrounds, hover fills |
| `--color-surface-container` | `#efeeea` | DESIGN.md §2 | Secondary content areas |
| `--color-surface-container-high` | `#e9e8e5` | Derived | Emphasized sections, alternating rows |
| `--color-surface-container-highest` | `#e3e2df` | DESIGN.md §5 | Input field backgrounds |

### 2.2 Primary Palette

| Token | Hex | Source | Usage |
|-------|-----|--------|-------|
| `--color-primary` | `#0c5252` | DESIGN.md §2 | Primary actions, links, focus rings |
| `--color-on-primary` | `#ffffff` | DESIGN.md §5 | Text on primary backgrounds |
| `--color-primary-container` | `#2d6a6a` | DESIGN.md §2 | Gradient endpoint, emphasis fills |
| `--color-on-primary-container` | `#ffffff` | Derived | Text on primary-container backgrounds |

### 2.3 Secondary Palette

| Token | Hex | Source | Usage |
|-------|-----|--------|-------|
| `--color-secondary` | `#7a4f1e` | Derived | Secondary actions |
| `--color-on-secondary` | `#ffffff` | Derived | Text on secondary backgrounds |
| `--color-secondary-container` | `#fdd6b4` | DESIGN.md §4 | Secondary fills, glassmorphism base |
| `--color-on-secondary-container` | `#2d1600` | Derived | Text on secondary-container |

### 2.4 Tertiary Palette

| Token | Hex | Source | Usage |
|-------|-----|--------|-------|
| `--color-tertiary` | `#6d5e00` | Derived | Tertiary accents |
| `--color-on-tertiary` | `#ffffff` | Derived | Text on tertiary backgrounds |
| `--color-tertiary-fixed` | `#ffdf96` | DESIGN.md §5 | Progress ribbon background |
| `--color-on-tertiary-fixed` | `#221b00` | Derived | Text on tertiary-fixed |

### 2.5 Feedback Colors

| Token | Hex | Source | Usage |
|-------|-----|--------|-------|
| `--color-error` | `#ba1a1a` | DESIGN.md §5 | Error text, destructive actions |
| `--color-on-error` | `#ffffff` | Derived | Text on error backgrounds |
| `--color-error-container` | `#ffdad6` | DESIGN.md §5 | Error field backgrounds |
| `--color-on-error-container` | `#410002` | Derived | Text on error-container |
| `--color-success` | `#386a1f` | Derived | Success indicators |
| `--color-on-success` | `#ffffff` | Derived | Text on success backgrounds |
| `--color-success-container` | `#d4f5c0` | Derived | Success field backgrounds |
| `--color-on-success-container` | `#0a2000` | Derived | Text on success-container |
| `--color-warning` | `#7a5900` | Derived | Warning indicators |
| `--color-on-warning` | `#ffffff` | Derived | Text on warning backgrounds |
| `--color-warning-container` | `#ffdea3` | Derived | Warning field backgrounds |
| `--color-on-warning-container` | `#261a00` | Derived | Text on warning-container |

### 2.6 Outline & On-Surface

| Token | Hex | Source | Usage |
|-------|-----|--------|-------|
| `--color-on-surface` | `#1b1c1a` | DESIGN.md §6 | Body text, headings — never pure black |
| `--color-on-surface-variant` | `#636968` | Derived | Secondary text, captions, placeholders |
| `--color-outline` | `#737978` | Derived | Subtle borders (when required) |
| `--color-outline-variant` | `#bfc8c8` | DESIGN.md §4 | Ghost border base (used at 20% opacity) |

### 2.7 Inverse & Scrim

| Token | Hex | Usage |
|-------|-----|-------|
| `--color-inverse-surface` | `#303030` | Snackbars, toasts |
| `--color-inverse-on-surface` | `#f1f0ec` | Text on inverse-surface |
| `--color-inverse-primary` | `#84d4d4` | Links on inverse-surface |
| `--color-scrim` | `#000000` | Modal backdrop (used at 32% opacity) |

### 2.8 WCAG Contrast Verification

All text/background pairings MUST meet **WCAG 2.1 AA**: 4.5:1 for normal text, 3:1 for large
text (≥18pt or ≥14pt bold). Non-text UI components MUST meet 3:1.

| Foreground | Background | Ratio | Passes |
|------------|------------|-------|--------|
| `on-surface` (#1b1c1a) | `surface` (#faf9f5) | 16.2:1 | AA, AAA |
| `on-surface` (#1b1c1a) | `surface-container` (#efeeea) | 14.7:1 | AA, AAA |
| `on-surface` (#1b1c1a) | `surface-container-highest` (#e3e2df) | 13.2:1 | AA, AAA |
| `on-surface` (#1b1c1a) | `surface-container-lowest` (#ffffff) | 17.4:1 | AA, AAA |
| `on-primary` (#ffffff) | `primary` (#0c5252) | 9.0:1 | AA, AAA |
| `on-primary-container` (#ffffff) | `primary-container` (#2d6a6a) | 6.2:1 | AA, AAA |
| `primary` (#0c5252) | `surface` (#faf9f5) | 8.5:1 | AA, AAA |
| `primary` (#0c5252) | `surface-container` (#efeeea) | 7.7:1 | AA, AAA |
| `primary` (#0c5252) | `surface-container-lowest` (#ffffff) | 9.0:1 | AA, AAA |
| `on-secondary-container` (#2d1600) | `secondary-container` (#fdd6b4) | 12.6:1 | AA, AAA |
| `on-tertiary-fixed` (#221b00) | `tertiary-fixed` (#ffdf96) | 13.3:1 | AA, AAA |
| `error` (#ba1a1a) | `surface` (#faf9f5) | 6.1:1 | AA |
| `error` (#ba1a1a) | `error-container` (#ffdad6) | 5.0:1 | AA |
| `on-error-container` (#410002) | `error-container` (#ffdad6) | 13.3:1 | AA, AAA |
| `on-surface-variant` (#636968) | `surface` (#faf9f5) | 5.3:1 | AA |
| `on-surface-variant` (#636968) | `surface-container-lowest` (#ffffff) | 5.7:1 | AA |
| `on-surface-variant` (#636968) | `surface-container-highest` (#e3e2df) | 4.5:1 | AA |

**Non-text verification** (focus rings, borders — 3:1 minimum):

| Element | Against | Ratio | Passes |
|---------|---------|-------|--------|
| `--color-focus-ring` (#0c5252) | `surface` (#faf9f5) | 8.5:1 | 3:1 ✓ |
| `--color-focus-ring` (#0c5252) | `surface-container-highest` (#e3e2df) | 6.9:1 | 3:1 ✓ |
| `--color-focus-ring` (#0c5252) | `surface-container-lowest` (#ffffff) | 9.0:1 | 3:1 ✓ |
| `--color-focus-ring` (#0c5252) | `secondary-container` (#fdd6b4) | 6.6:1 | 3:1 ✓ |
| `--color-focus-ring` (#0c5252) | `error-container` (#ffdad6) | 6.9:1 | 3:1 ✓ |
| `outline` (#737978) | `surface` (#faf9f5) | 4.2:1 | 3:1 ✓ |

### 2.9 Dark Mode Architecture

Dark mode is **deferred** but structurally prepared. The token architecture supports it via
CSS custom property reassignment:

```css
/* Future: tokens.css addition */
@media (prefers-color-scheme: dark) {
  :root {
    --color-surface: #1b1c1a;
    --color-on-surface: #e3e2df;
    /* ... reassign all tokens ... */
  }
}
```

**Rules for dark mode readiness:**
- MUST NOT use `dark:` Tailwind prefixes in JSX. All dark mode adaptation happens via token
  reassignment in CSS.
- MUST NOT hardcode hex color values in component code. Always use token-derived Tailwind
  classes (`bg-surface`, `text-on-surface`).
- Components that use token classes will automatically adapt when dark mode tokens are defined.

---

## §3 Typography Tokens

### 3.1 Font Families

| Token | Value | Usage |
|-------|-------|-------|
| `--font-display` | `"Plus Jakarta Sans", system-ui, sans-serif` | Display, headline, title scales |
| `--font-body` | `"Manrope", system-ui, sans-serif` | Body, label scales |
| `--font-mono` | `"JetBrains Mono", ui-monospace, monospace` | Code blocks (if needed) |

**`system-ui` fallback**: Ensures readable text before web fonts load.

### 3.2 Self-Hosted Font Loading

Fonts MUST be self-hosted. Google Fonts CDN leaks user IP addresses and browsing data, which
is unacceptable for a COPPA-compliant, privacy-first platform. `[ARCH §1.5]`

**Font files location**: `frontend/public/fonts/`

```
frontend/public/fonts/
├── plus-jakarta-sans/
│   ├── PlusJakartaSans-Regular.woff2
│   ├── PlusJakartaSans-Medium.woff2
│   ├── PlusJakartaSans-SemiBold.woff2
│   └── PlusJakartaSans-Bold.woff2
└── manrope/
    ├── Manrope-Regular.woff2
    ├── Manrope-Medium.woff2
    └── Manrope-SemiBold.woff2
```

**WOFF2 only** — all modern browsers support it. No need for WOFF/TTF fallbacks.

**`@font-face` declarations** MUST be in `base.css`:

```css
/* base.css — @font-face declarations */
@layer base {
  @font-face {
    font-family: "Plus Jakarta Sans";
    src: url("/fonts/plus-jakarta-sans/PlusJakartaSans-Regular.woff2") format("woff2");
    font-weight: 400;
    font-style: normal;
    font-display: swap;
  }
  @font-face {
    font-family: "Plus Jakarta Sans";
    src: url("/fonts/plus-jakarta-sans/PlusJakartaSans-Medium.woff2") format("woff2");
    font-weight: 500;
    font-style: normal;
    font-display: swap;
  }
  @font-face {
    font-family: "Plus Jakarta Sans";
    src: url("/fonts/plus-jakarta-sans/PlusJakartaSans-SemiBold.woff2") format("woff2");
    font-weight: 600;
    font-style: normal;
    font-display: swap;
  }
  @font-face {
    font-family: "Plus Jakarta Sans";
    src: url("/fonts/plus-jakarta-sans/PlusJakartaSans-Bold.woff2") format("woff2");
    font-weight: 700;
    font-style: normal;
    font-display: swap;
  }
  @font-face {
    font-family: "Manrope";
    src: url("/fonts/manrope/Manrope-Regular.woff2") format("woff2");
    font-weight: 400;
    font-style: normal;
    font-display: swap;
  }
  @font-face {
    font-family: "Manrope";
    src: url("/fonts/manrope/Manrope-Medium.woff2") format("woff2");
    font-weight: 500;
    font-style: normal;
    font-display: swap;
  }
  @font-face {
    font-family: "Manrope";
    src: url("/fonts/manrope/Manrope-SemiBold.woff2") format("woff2");
    font-weight: 600;
    font-style: normal;
    font-display: swap;
  }
}
```

**Preload hints** MUST be in `index.html` for the two most critical weights:

```html
<link rel="preload" href="/fonts/plus-jakarta-sans/PlusJakartaSans-SemiBold.woff2"
      as="font" type="font/woff2" crossorigin>
<link rel="preload" href="/fonts/manrope/Manrope-Regular.woff2"
      as="font" type="font/woff2" crossorigin>
```

**Rule**: `font-display: swap` is required on all `@font-face` rules. This ensures text is
immediately visible with the system font fallback, then swaps to the web font when loaded.
No FOIT (Flash of Invisible Text) is acceptable.

### 3.3 Font Weight Tokens

| Token | Value | Usage |
|-------|-------|-------|
| `--font-weight-regular` | `400` | Body text, default |
| `--font-weight-medium` | `500` | Labels, subtle emphasis |
| `--font-weight-semibold` | `600` | Headings, titles |
| `--font-weight-bold` | `700` | Display text, strong emphasis |

### 3.4 MD3 Type Scale

The full 15-step MD3 type scale. All sizes in `rem` for accessibility (respects user font-size
preferences).

#### Display Scale (Plus Jakarta Sans — for hero/masthead text)

| Token | Size | Line Height | Weight | Letter Spacing |
|-------|------|-------------|--------|----------------|
| `--text-display-lg` | `3.5rem` | `1.12` | 700 | `-0.02em` |
| `--text-display-md` | `2.8125rem` | `1.16` | 700 | `-0.015em` |
| `--text-display-sm` | `2.25rem` | `1.2` | 600 | `-0.01em` |

> `display-lg` values come directly from DESIGN.md §3: "3.5rem with tighter letter-spacing
> (-0.02em) to create a bold, magazine masthead feel."

#### Headline Scale (Plus Jakarta Sans — for page/section headings)

| Token | Size | Line Height | Weight | Letter Spacing |
|-------|------|-------------|--------|----------------|
| `--text-headline-lg` | `2rem` | `1.25` | 600 | `-0.005em` |
| `--text-headline-md` | `1.75rem` | `1.29` | 600 | `0` |
| `--text-headline-sm` | `1.5rem` | `1.33` | 600 | `0` |

#### Title Scale (Plus Jakarta Sans — for card/section titles)

| Token | Size | Line Height | Weight | Letter Spacing |
|-------|------|-------------|--------|----------------|
| `--text-title-lg` | `1.375rem` | `1.27` | 600 | `0` |
| `--text-title-md` | `1rem` | `1.5` | 600 | `0.009em` |
| `--text-title-sm` | `0.875rem` | `1.43` | 500 | `0.007em` |

#### Body Scale (Manrope — for running text)

| Token | Size | Line Height | Weight | Letter Spacing |
|-------|------|-------------|--------|----------------|
| `--text-body-lg` | `1rem` | `1.6` | 400 | `0` |
| `--text-body-md` | `0.875rem` | `1.5` | 400 | `0.01em` |
| `--text-body-sm` | `0.75rem` | `1.4` | 400 | `0.02em` |

> `body-lg` values come directly from DESIGN.md §3: "1rem with generous line-height (1.6)."

#### Label Scale (Manrope — for buttons, captions, metadata)

| Token | Size | Line Height | Weight | Letter Spacing |
|-------|------|-------------|--------|----------------|
| `--text-label-lg` | `0.875rem` | `1.43` | 500 | `0.02em` |
| `--text-label-md` | `0.75rem` | `1.33` | 500 | `0.03em` |
| `--text-label-sm` | `0.6875rem` | `1.45` | 500 | `0.04em` |

### 3.5 Type Scale Utility Classes

Since each type scale step combines font-family + size + line-height + weight + letter-spacing,
composite utility classes MUST be defined in `components.css`:

```css
/* components.css — Type scale composite classes */
@layer components {
  .type-display-lg {
    font-family: var(--font-display);
    font-size: var(--text-display-lg);
    line-height: 1.12;
    font-weight: 700;
    letter-spacing: -0.02em;
  }
  /* ... one class per type scale step ... */
  .type-body-lg {
    font-family: var(--font-body);
    font-size: var(--text-body-lg);
    line-height: 1.6;
    font-weight: 400;
    letter-spacing: 0;
  }
}
```

**Rule**: Developers MAY use individual Tailwind utilities (`text-body-lg font-body`) when the
type scale class is too rigid. But for standard text, `type-{scale}-{size}` classes SHOULD
be preferred to ensure correct font-family + weight pairing.

### 3.6 Parent vs Student Typography Context

Per DESIGN.md §5:
- **Parent context**: Primary buttons use `label-md` with `text-transform: uppercase`.
- **Student context**: Primary buttons use `title-sm` with sentence case (no transform).

These differences are handled by the Parent/Student theming system (§9), not by separate
CSS files or hardcoded conditionals.

---

## §4 Spacing Tokens

### 4.1 Base Scale

Use Tailwind's default spacing scale (0.25rem increments). Do NOT override the default scale.

### 4.2 Semantic Aliases

Semantic spacing tokens provide meaning-based names that map to the base scale:

| Token | Value | Usage |
|-------|-------|-------|
| `--spacing-list-gap` | `1.4rem` | DESIGN.md §5: "spacing-4 (1.4rem) of vertical whitespace" between list items |
| `--spacing-card-padding` | `1.5rem` | Internal padding for cards |
| `--spacing-section-gap` | `3rem` | Vertical gap between page sections |
| `--spacing-page-x` | `1.5rem` | Horizontal page margin (mobile) |
| `--spacing-page-x-lg` | `3rem` | Horizontal page margin (desktop) |
| `--spacing-asymmetric-offset` | `4rem` | DESIGN.md §6: "spacing-12 (4rem) to create bespoke feel" |

### 4.3 Touch Target Enforcement

Per `[ARCH §11.5]`: all interactive elements MUST have a minimum tap target of **44×44 CSS
pixels** on viewports under 768px.

A utility class MUST be provided in `utilities.css` (see §14 for the full implementation).

---

## §5 Border Radius Tokens

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | `0.25rem` | Chips, small badges |
| `--radius-md` | `0.5rem` | Buttons (DESIGN.md §5: "md 0.75rem" — adjusted for consistency) |
| `--radius-button` | `0.75rem` | Primary button radius (DESIGN.md §5) |
| `--radius-lg` | `1rem` | Parent cards (DESIGN.md §5: "lg for sophisticated feel") |
| `--radius-xl` | `1.5rem` | Student cards (DESIGN.md §5: "xl for inviting feel") |
| `--radius-2xl` | `2rem` | Large containers, modals |
| `--radius-full` | `9999px` | Pill shapes — reserved for small Chips only (DESIGN.md §6) |

**Context-dependent card radius** (see §9 Parent/Student Theming):

```css
/* Parent context: --radius-lg (1rem) for sophisticated feel */
/* Student context: --radius-xl (1.5rem) for inviting feel */
```

---

## §6 Shadow & Elevation

DESIGN.md §4 establishes that depth is primarily achieved through **tonal layering** (background
color shifts), not shadows. Shadows are reserved for floating elements only.

### 6.1 Ambient Shadow Scale

Shadows use oklch with an `on-surface` tint to maintain the warm aesthetic. `[DESIGN.md §4]`

| Token | Value | Usage |
|-------|-------|-------|
| `--shadow-ambient-sm` | `0 2px 8px oklch(0.17 0.005 163 / 0.03)` | Subtle card lift |
| `--shadow-ambient-md` | `0 4px 16px oklch(0.17 0.005 163 / 0.04)` | Dropdowns, popovers |
| `--shadow-ambient-lg` | `0 8px 32px oklch(0.17 0.005 163 / 0.04)` | Modals, floating elements |
| `--shadow-ambient-xl` | `0 12px 48px oklch(0.17 0.005 163 / 0.05)` | Student badge popups |

> The oklch values approximate `#1b1c1a` (on-surface) in oklch color space with a slight
> warm tint. The 4% opacity at 32px+ blur creates the "soft, natural sunlight" effect
> described in DESIGN.md §4.

### 6.2 Ghost Border Token

Per DESIGN.md §4: "If accessibility requirements demand a border, use outline-variant at
20% opacity."

| Token | Value | Usage |
|-------|-------|-------|
| `--shadow-ghost-border` | `inset 0 0 0 1px oklch(0.81 0.015 180 / 0.2)` | Ghost border fallback |

The ghost border is implemented as an `inset box-shadow` rather than a `border` to avoid
affecting layout dimensions. The oklch value approximates `outline-variant` (#bfc8c8).

**Rule**: Standard 1px solid borders are **strictly prohibited** for sectioning. `[DESIGN.md §2]`

---

## §7 Motion & Animation

### 7.1 Duration Scale

| Token | Value | Usage |
|-------|-------|-------|
| `--duration-fast` | `100ms` | Micro-interactions: hover color, opacity toggle |
| `--duration-normal` | `200ms` | Standard transitions: button press, input focus |
| `--duration-slow` | `300ms` | Layout transitions: accordion, sidebar collapse |
| `--duration-slower` | `500ms` | Complex animations: page enter, modal appear |

### 7.2 Easing Curves

| Token | Value | Usage |
|-------|-------|-------|
| `--ease-standard` | `cubic-bezier(0.2, 0.0, 0, 1.0)` | Default for most transitions |
| `--ease-decelerate` | `cubic-bezier(0.0, 0.0, 0, 1.0)` | Elements entering the screen |
| `--ease-accelerate` | `cubic-bezier(0.3, 0.0, 1, 1.0)` | Elements leaving the screen |
| `--ease-spring` | `cubic-bezier(0.175, 0.885, 0.32, 1.275)` | Playful bounce (Student context) |

### 7.3 Reduced Motion

```css
/* base.css */
@layer base {
  @media (prefers-reduced-motion: reduce) {
    *, *::before, *::after {
      animation-duration: 0.01ms !important;
      animation-iteration-count: 1 !important;
      transition-duration: 0.01ms !important;
      scroll-behavior: auto !important;
    }
  }
}
```

**Rule**: This global reduction MUST be included in `base.css`. Individual animations MUST NOT
override this media query.

### 7.4 No-Flash Guarantee

- Background color transitions MUST use `--duration-normal` or longer to prevent flashing.
- Theme switches (future dark mode) MUST use `--duration-slow` to prevent jarring shifts.
- On initial page load, no color transition is applied (transitions activate after first paint).

---

## §8 Responsive Breakpoints & Layout

### 8.1 Breakpoints

Use Tailwind's default breakpoints plus one custom addition:

| Token | Value | Tailwind Prefix |
|-------|-------|-----------------|
| (default) | `< 640px` | (none — mobile-first) |
| `--breakpoint-sm` | `640px` | `sm:` |
| `--breakpoint-md` | `768px` | `md:` |
| `--breakpoint-lg` | `1024px` | `lg:` |
| `--breakpoint-xl` | `1280px` | `xl:` |
| `--breakpoint-2xl` | `1536px` | `2xl:` |
| `--breakpoint-3xl` | `1920px` | `3xl:` |

Only `--breakpoint-3xl` needs to be defined in `tokens.css` — the others are Tailwind defaults.

### 8.2 Container Width Tokens

| Token | Value | Usage |
|-------|-------|-------|
| `--width-content` | `72rem` | Max-width for main content area (1152px) |
| `--width-content-narrow` | `48rem` | Max-width for focused content (768px) |
| `--width-sidebar` | `16rem` | Sidebar width (256px) |
| `--width-sidebar-collapsed` | `4rem` | Collapsed sidebar (64px, icons only) |

### 8.3 Responsive Behavior Rules

| Breakpoint | Layout |
|------------|--------|
| `< 640px` (mobile) | Single column. Sidebar hidden (hamburger menu). Touch targets 44×44. |
| `640px–767px` (sm) | Single column. Sidebar as overlay drawer. |
| `768px–1023px` (md) | Sidebar collapsed (icons). Content fills remaining width. |
| `1024px–1279px` (lg) | Sidebar expanded. Content max-width `--width-content`. |
| `1280px+` (xl) | Full layout. Content centered with generous margins. |
| `1920px+` (3xl) | Wide layout. Optional secondary sidebar for contextual panels. |

---

## §9 Parent/Student Theming

DESIGN.md describes two visual contexts — Parent (professional, data-focused) and Student
(inviting, pathway-focused). These contexts are implemented via a **data attribute**, not
separate CSS files or duplicated components.

### 9.1 Data Attribute

The `<main>` element (or nearest layout wrapper) MUST carry a `data-context` attribute:

```html
<main data-context="parent">...</main>
<!-- or -->
<main data-context="student">...</main>
```

The default context is `parent` if no attribute is present.

### 9.2 Tailwind Custom Variants

Define custom variants in `tokens.css`:

```css
/* tokens.css — Custom variants for Parent/Student context */
@custom-variant parent (:where([data-context="parent"]) &);
@custom-variant student (:where([data-context="student"]) &);
```

The `:where()` wrapper gives the selector **zero specificity**, so context-specific styles
don't escalate the specificity war.

**Usage in JSX:**

```tsx
<div className="rounded-lg parent:rounded-lg student:rounded-xl">
  <h2 className="type-headline-md parent:uppercase student:normal-case">
    Dashboard
  </h2>
</div>
```

### 9.3 Rules

- MUST NOT create separate CSS files for Parent and Student themes.
- MUST NOT duplicate component implementations for each context.
- Context differences MUST be limited to: border radius, typography casing, spacing density,
  and color emphasis — not structural layout changes.
- The `data-context` attribute MUST be set by the application's auth/routing layer based on
  the active user type.

---

## §10 Z-Index Scale

All z-index values MUST use tokens from this controlled scale. Arbitrary `z-[n]` values
are forbidden.

| Token | Value | Usage |
|-------|-------|-------|
| `--z-base` | `0` | Default stacking context |
| `--z-raised` | `100` | Cards lifted above siblings |
| `--z-sticky` | `200` | Sticky headers, table headers |
| `--z-overlay` | `300` | Overlays, sidebar drawers |
| `--z-modal` | `400` | Modal dialogs |
| `--z-notification` | `500` | Toast notifications |
| `--z-popover` | `600` | Popovers, select dropdowns |
| `--z-tooltip` | `700` | Tooltips (highest user-facing layer) |

**Implementation**: Since Tailwind v4 does not have a `--z-*` theme namespace, these tokens
are defined as plain CSS custom properties in `tokens.css` and applied via `z-[var(--z-modal)]`
or through utility classes in `utilities.css`:

```css
/* utilities.css */
@utility z-base { z-index: var(--z-base); }
@utility z-raised { z-index: var(--z-raised); }
@utility z-sticky { z-index: var(--z-sticky); }
@utility z-overlay { z-index: var(--z-overlay); }
@utility z-modal { z-index: var(--z-modal); }
@utility z-notification { z-index: var(--z-notification); }
@utility z-popover { z-index: var(--z-popover); }
@utility z-tooltip { z-index: var(--z-tooltip); }
```

---

## §11 Opacity Tokens

MD3 state layer opacities and common transparency values:

| Token | Value | Usage |
|-------|-------|-------|
| `--opacity-disabled` | `0.38` | Disabled elements (MD3 standard) |
| `--opacity-hover` | `0.08` | Hover state layer (MD3) |
| `--opacity-pressed` | `0.12` | Pressed/active state layer (MD3) |
| `--opacity-focus` | `0.12` | Focus state layer (MD3) |
| `--opacity-dragged` | `0.16` | Dragged state layer (MD3) |
| `--opacity-glass` | `0.8` | Glass/frosted surfaces (DESIGN.md §2: "80% opacity") |
| `--opacity-ghost-border` | `0.2` | Ghost border (DESIGN.md §4: "20% opacity") |
| `--opacity-scrim` | `0.32` | Modal backdrop overlay |

---

## §12 Focus Ring Specification

Focus indicators are critical for keyboard navigation and accessibility.
`[CODING_STANDARDS §3.7]` `[ARCH §11.5]`

### 12.1 Default Focus Ring

| Property | Value |
|----------|-------|
| Color | `--color-focus-ring: #0c5252` (same as primary) |
| Style | `2px solid` |
| Offset | `2px` (gap between element and ring) |
| Border radius | Inherits from element |

```css
/* base.css — Global focus ring */
@layer base {
  :focus-visible {
    outline: 2px solid var(--color-focus-ring);
    outline-offset: 2px;
  }
}
```

### 12.2 Input Field Focus Variant

Input fields use the ghost border pattern from DESIGN.md §5 on focus:

```css
@layer components {
  .input-focus {
    outline: none;
    box-shadow: inset 0 0 0 2px var(--color-primary);
  }
}
```

This replaces the outline with an inset box-shadow for a more integrated look while
maintaining the same contrast ratio.

### 12.3 Contrast Verification

The focus ring (#0c5252) passes 3:1 minimum contrast against all surfaces it may appear on
(verified in §2.8 Non-text verification table). WCAG 2.1 Success Criterion 1.4.11 requires
3:1 contrast for non-text UI components.

---

## §13 Icon System

### 13.1 Library

**Lucide React** (`lucide-react`) is the recommended icon library.

- **Tree-shakeable** — only imported icons are bundled.
- **MIT licensed** — no usage restrictions.
- **SVG-based** — crisp at any size, color-inheriting.
- **Broad set** — 1000+ icons covering all needed UI patterns.

### 13.2 Size Scale

| Token | Value | Usage |
|-------|-------|-------|
| `--icon-xs` | `0.75rem` (12px) | Inline badges, status dots |
| `--icon-sm` | `1rem` (16px) | Inline with body text |
| `--icon-md` | `1.25rem` (20px) | Buttons, input adornments |
| `--icon-lg` | `1.5rem` (24px) | Navigation items, card headers |
| `--icon-xl` | `2rem` (32px) | Feature icons, empty states |
| `--icon-2xl` | `3rem` (48px) | Hero illustrations, onboarding |

### 13.3 Accessibility Rules

- Decorative icons (paired with visible text) MUST use `aria-hidden="true"`.
- Standalone icons (icon-only buttons) MUST have an `aria-label` on the parent `<button>`.
- Icon color MUST be inherited from the parent text color (`currentColor`). MUST NOT hardcode
  icon colors unless the icon represents a specific status (error, success, warning).

---

## §14 Interactive States

### 14.1 MD3 State Layer Model

Interactive states use semi-transparent overlays on top of base colors. The overlay color is
the "content" color for that surface (e.g., `on-primary` on `primary` backgrounds).

| State | Overlay Opacity | Additional Effect |
|-------|----------------|-------------------|
| Default | 0% | — |
| Hover | 8% | `cursor: pointer` |
| Focus | 12% | Focus ring (§12) |
| Pressed | 12% | Scale 0.98 (optional, for buttons) |
| Dragged | 16% | Ambient shadow elevation increase |
| Disabled | — | Element at 38% opacity, no pointer events |
| Loading | — | Content hidden, spinner visible |

### 14.2 State Matrix

#### Primary Button

| State | Background | Text | Border | Other |
|-------|-----------|------|--------|-------|
| Default | `primary` | `on-primary` | none | — |
| Hover | `primary` + 8% `on-primary` overlay | `on-primary` | none | `cursor: pointer` |
| Active | `primary` + 12% `on-primary` overlay | `on-primary` | none | — |
| Focus | `primary` | `on-primary` | none | Focus ring (§12) |
| Disabled | `on-surface` at 12% opacity | `on-surface` at 38% opacity | none | `pointer-events: none` |
| Loading | `primary` | hidden | none | Spinner centered |

#### Secondary Button

| State | Background | Text | Border | Other |
|-------|-----------|------|--------|-------|
| Default | `secondary-container` | `on-secondary-container` | none | — |
| Hover | `secondary-container` + 8% `on-secondary-container` overlay | `on-secondary-container` | none | — |
| Active | `secondary-container` + 12% `on-secondary-container` overlay | `on-secondary-container` | none | — |
| Focus | `secondary-container` | `on-secondary-container` | none | Focus ring |
| Disabled | `on-surface` at 12% | `on-surface` at 38% | none | `pointer-events: none` |

#### Tertiary Button (Text Button)

| State | Background | Text | Border | Other |
|-------|-----------|------|--------|-------|
| Default | transparent | `primary` | none | — |
| Hover | `surface-container-low` | `primary` | none | DESIGN.md §5 |
| Active | `surface-container` | `primary` | none | — |
| Focus | transparent | `primary` | none | Focus ring |
| Disabled | transparent | `on-surface` at 38% | none | `pointer-events: none` |

#### Input Field

| State | Background | Text | Border | Other |
|-------|-----------|------|--------|-------|
| Default | `surface-container-highest` | `on-surface` | none | DESIGN.md §5 |
| Hover | `surface-container-high` | `on-surface` | none | — |
| Focus | `surface-container-highest` | `on-surface` | 2px inset `primary` | Ghost border transition |
| Error | `error-container` | `on-surface` | 2px inset `error` | Error text below |
| Disabled | `surface-container-highest` at 38% | `on-surface` at 38% | none | — |

#### Card

| State | Background | Text | Shadow | Other |
|-------|-----------|------|--------|-------|
| Default | `surface-container-lowest` | `on-surface` | none | — |
| Hover (interactive) | `surface-container-lowest` | `on-surface` | `ambient-sm` | Subtle lift |
| Active (interactive) | `surface-container-low` | `on-surface` | none | — |
| Focus (interactive) | `surface-container-lowest` | `on-surface` | none | Focus ring |

#### Link

| State | Color | Decoration | Other |
|-------|-------|-----------|-------|
| Default | `primary` | none | — |
| Hover | `primary-container` | underline | — |
| Active | `primary` | underline | — |
| Focus | `primary` | none | Focus ring |
| Visited | `primary` | none | Same as default (privacy: no visited indicator) |

### 14.3 Touch Target Utility

```css
/* utilities.css */
@utility touch-target {
  position: relative;

  &::after {
    content: "";
    position: absolute;
    inset: -6px;
    min-width: 44px;
    min-height: 44px;
  }
}
```

This expands the clickable area using a pseudo-element without changing the visual size.
MUST be applied to all interactive elements on viewports under 768px (`md:`).

---

## §15 Print Tokens

Print styles implement `[CODING_STANDARDS §3.8]` requirements through token reassignment,
so components auto-adapt without per-component `@media print` rules.

### 15.1 Token Overrides

```css
/* print.css */
@media print {
  :root {
    /* Flatten to pure black and white */
    --color-surface: #ffffff;
    --color-surface-container-lowest: #ffffff;
    --color-surface-container-low: #ffffff;
    --color-surface-container: #f5f5f5;
    --color-surface-container-high: #eeeeee;
    --color-surface-container-highest: #e0e0e0;
    --color-on-surface: #000000;
    --color-primary: #000000;
    --color-on-primary: #ffffff;

    /* Remove decorative shadows */
    --shadow-ambient-sm: none;
    --shadow-ambient-md: none;
    --shadow-ambient-lg: none;
    --shadow-ambient-xl: none;
    --shadow-ghost-border: none;

    /* Disable animations */
    --duration-fast: 0ms;
    --duration-normal: 0ms;
    --duration-slow: 0ms;
    --duration-slower: 0ms;
  }
}
```

### 15.2 Structural Hide Rules

```css
@media print {
  nav,
  aside,
  [data-print-hide],
  .no-print {
    display: none !important;
  }
}
```

Elements that should be hidden in print MUST use `data-print-hide` attribute or `no-print` class.

### 15.3 Page Break Utilities

```css
/* utilities.css */
@utility print-break-before { @media print { break-before: page; } }
@utility print-break-after { @media print { break-after: page; } }
@utility print-avoid-break { @media print { break-inside: avoid; } }
```

### 15.4 Grayscale Compliance

Per `[CODING_STANDARDS §3.8]`: "Color output MUST NOT be required for print readability."
All meaning conveyed by color MUST also be conveyed by text, icons, or patterns. The print
token overrides flatten colors to grayscale-safe values, but semantic meaning must not rely
on color alone in the first place.

---

## §16 Gradient Tokens

Per DESIGN.md §2: "For high-impact areas, use a subtle linear gradient from primary to
primary-container at a 135-degree angle."

| Token | Value | Usage |
|-------|-------|-------|
| `--gradient-primary` | `linear-gradient(135deg, var(--color-primary), var(--color-primary-container))` | "Start Lesson" CTAs, "Monthly Review" headers |
| `--gradient-surface` | `linear-gradient(180deg, var(--color-surface), var(--color-surface-container-low))` | Subtle page section transitions |

**Rule**: Gradients MUST NOT be applied to large background areas. Use them sparingly for
emphasis per DESIGN.md §2 guidance.

---

## §17 Reference: Complete `@theme` Block

This is the assembled `tokens.css` file that implementation MUST produce. It contains all
tokens from §2–§16 in a single, copy-paste-ready block.

```css
/* tokens.css — Homegrown Academy Design Tokens */
/* Source of truth: specs/DESIGN_TOKENS.md */
/* DO NOT edit values here without updating the spec. */

/* ─── Custom Variants ──────────────────────────────────────────────── */
@custom-variant parent (:where([data-context="parent"]) &);
@custom-variant student (:where([data-context="student"]) &);

/* ─── Theme Tokens ─────────────────────────────────────────────────── */
@theme {

  /* ── Colors: Surface ─────────────────────────────────────────────── */
  --color-surface:                    #faf9f5;
  --color-surface-dim:                #dddcd8;
  --color-surface-container-lowest:   #ffffff;
  --color-surface-container-low:      #f5f4f0;
  --color-surface-container:          #efeeea;
  --color-surface-container-high:     #e9e8e5;
  --color-surface-container-highest:  #e3e2df;

  /* ── Colors: Primary ─────────────────────────────────────────────── */
  --color-primary:                    #0c5252;
  --color-on-primary:                 #ffffff;
  --color-primary-container:          #2d6a6a;
  --color-on-primary-container:       #ffffff;

  /* ── Colors: Secondary ───────────────────────────────────────────── */
  --color-secondary:                  #7a4f1e;
  --color-on-secondary:               #ffffff;
  --color-secondary-container:        #fdd6b4;
  --color-on-secondary-container:     #2d1600;

  /* ── Colors: Tertiary ────────────────────────────────────────────── */
  --color-tertiary:                   #6d5e00;
  --color-on-tertiary:                #ffffff;
  --color-tertiary-fixed:             #ffdf96;
  --color-on-tertiary-fixed:          #221b00;

  /* ── Colors: Error ───────────────────────────────────────────────── */
  --color-error:                      #ba1a1a;
  --color-on-error:                   #ffffff;
  --color-error-container:            #ffdad6;
  --color-on-error-container:         #410002;

  /* ── Colors: Success ─────────────────────────────────────────────── */
  --color-success:                    #386a1f;
  --color-on-success:                 #ffffff;
  --color-success-container:          #d4f5c0;
  --color-on-success-container:       #0a2000;

  /* ── Colors: Warning ─────────────────────────────────────────────── */
  --color-warning:                    #7a5900;
  --color-on-warning:                 #ffffff;
  --color-warning-container:          #ffdea3;
  --color-on-warning-container:       #261a00;

  /* ── Colors: Outline & On-Surface ────────────────────────────────── */
  --color-on-surface:                 #1b1c1a;
  --color-on-surface-variant:         #636968;
  --color-outline:                    #737978;
  --color-outline-variant:            #bfc8c8;

  /* ── Colors: Inverse & Scrim ─────────────────────────────────────── */
  --color-inverse-surface:            #303030;
  --color-inverse-on-surface:         #f1f0ec;
  --color-inverse-primary:            #84d4d4;
  --color-scrim:                      #000000;

  /* ── Colors: Focus Ring ──────────────────────────────────────────── */
  --color-focus-ring:                 #0c5252;

  /* ── Font Families ───────────────────────────────────────────────── */
  --font-display: "Plus Jakarta Sans", system-ui, sans-serif;
  --font-body:    "Manrope", system-ui, sans-serif;
  --font-mono:    "JetBrains Mono", ui-monospace, monospace;

  /* ── Font Weights ────────────────────────────────────────────────── */
  --font-weight-regular:  400;
  --font-weight-medium:   500;
  --font-weight-semibold: 600;
  --font-weight-bold:     700;

  /* ── Font Sizes (Type Scale) ─────────────────────────────────────── */
  /* Display — Plus Jakarta Sans */
  --text-display-lg: 3.5rem;
  --text-display-lg--line-height: 1.12;
  --text-display-lg--letter-spacing: -0.02em;

  --text-display-md: 2.8125rem;
  --text-display-md--line-height: 1.16;
  --text-display-md--letter-spacing: -0.015em;

  --text-display-sm: 2.25rem;
  --text-display-sm--line-height: 1.2;
  --text-display-sm--letter-spacing: -0.01em;

  /* Headline — Plus Jakarta Sans */
  --text-headline-lg: 2rem;
  --text-headline-lg--line-height: 1.25;
  --text-headline-lg--letter-spacing: -0.005em;

  --text-headline-md: 1.75rem;
  --text-headline-md--line-height: 1.29;

  --text-headline-sm: 1.5rem;
  --text-headline-sm--line-height: 1.33;

  /* Title — Plus Jakarta Sans */
  --text-title-lg: 1.375rem;
  --text-title-lg--line-height: 1.27;

  --text-title-md: 1rem;
  --text-title-md--line-height: 1.5;
  --text-title-md--letter-spacing: 0.009em;

  --text-title-sm: 0.875rem;
  --text-title-sm--line-height: 1.43;
  --text-title-sm--letter-spacing: 0.007em;

  /* Body — Manrope */
  --text-body-lg: 1rem;
  --text-body-lg--line-height: 1.6;

  --text-body-md: 0.875rem;
  --text-body-md--line-height: 1.5;
  --text-body-md--letter-spacing: 0.01em;

  --text-body-sm: 0.75rem;
  --text-body-sm--line-height: 1.4;
  --text-body-sm--letter-spacing: 0.02em;

  /* Label — Manrope */
  --text-label-lg: 0.875rem;
  --text-label-lg--line-height: 1.43;
  --text-label-lg--letter-spacing: 0.02em;

  --text-label-md: 0.75rem;
  --text-label-md--line-height: 1.33;
  --text-label-md--letter-spacing: 0.03em;

  --text-label-sm: 0.6875rem;
  --text-label-sm--line-height: 1.45;
  --text-label-sm--letter-spacing: 0.04em;

  /* ── Letter Spacing ──────────────────────────────────────────────── */
  --tracking-tighter: -0.02em;
  --tracking-tight:   -0.01em;
  --tracking-normal:  0;
  --tracking-wide:    0.02em;
  --tracking-wider:   0.04em;

  /* ── Spacing (Semantic Aliases) ──────────────────────────────────── */
  --spacing-list-gap:          1.4rem;
  --spacing-card-padding:      1.5rem;
  --spacing-section-gap:       3rem;
  --spacing-page-x:            1.5rem;
  --spacing-page-x-lg:         3rem;
  --spacing-asymmetric-offset: 4rem;

  /* ── Border Radius ───────────────────────────────────────────────── */
  --radius-sm:     0.25rem;
  --radius-md:     0.5rem;
  --radius-button: 0.75rem;
  --radius-lg:     1rem;
  --radius-xl:     1.5rem;
  --radius-2xl:    2rem;
  --radius-full:   9999px;

  /* ── Shadows ─────────────────────────────────────────────────────── */
  --shadow-ambient-sm: 0 2px 8px oklch(0.17 0.005 163 / 0.03);
  --shadow-ambient-md: 0 4px 16px oklch(0.17 0.005 163 / 0.04);
  --shadow-ambient-lg: 0 8px 32px oklch(0.17 0.005 163 / 0.04);
  --shadow-ambient-xl: 0 12px 48px oklch(0.17 0.005 163 / 0.05);

  /* ── Easing Curves ───────────────────────────────────────────────── */
  --ease-standard:   cubic-bezier(0.2, 0.0, 0, 1.0);
  --ease-decelerate: cubic-bezier(0.0, 0.0, 0, 1.0);
  --ease-accelerate: cubic-bezier(0.3, 0.0, 1, 1.0);
  --ease-spring:     cubic-bezier(0.175, 0.885, 0.32, 1.275);

  /* ── Breakpoints (custom addition only) ──────────────────────────── */
  --breakpoint-3xl: 1920px;
}

/* ─── Non-@theme Tokens (no Tailwind utility generation needed) ─── */
:root {
  /* Ghost Border */
  --shadow-ghost-border: inset 0 0 0 1px oklch(0.81 0.015 180 / 0.2);

  /* Z-Index Scale */
  --z-base:         0;
  --z-raised:       100;
  --z-sticky:       200;
  --z-overlay:      300;
  --z-modal:        400;
  --z-notification: 500;
  --z-popover:      600;
  --z-tooltip:      700;

  /* Opacity Scale */
  --opacity-disabled:     0.38;
  --opacity-hover:        0.08;
  --opacity-pressed:      0.12;
  --opacity-focus:        0.12;
  --opacity-dragged:      0.16;
  --opacity-glass:        0.8;
  --opacity-ghost-border: 0.2;
  --opacity-scrim:        0.32;

  /* Durations */
  --duration-fast:   100ms;
  --duration-normal: 200ms;
  --duration-slow:   300ms;
  --duration-slower: 500ms;

  /* Container Widths */
  --width-content:            72rem;
  --width-content-narrow:     48rem;
  --width-sidebar:            16rem;
  --width-sidebar-collapsed:  4rem;

  /* Icon Sizes */
  --icon-xs:  0.75rem;
  --icon-sm:  1rem;
  --icon-md:  1.25rem;
  --icon-lg:  1.5rem;
  --icon-xl:  2rem;
  --icon-2xl: 3rem;

  /* Gradients */
  --gradient-primary: linear-gradient(135deg, var(--color-primary), var(--color-primary-container));
  --gradient-surface: linear-gradient(180deg, var(--color-surface), var(--color-surface-container-low));
}
```

---

## §18 Implementation Checklist

Ordered creation steps for implementing this token system:

### Phase 1: Infrastructure

- [ ] Install `@tailwindcss/vite` and add to Vite plugins
- [ ] Remove `postcss` and `autoprefixer` from devDependencies (if unused elsewhere)
- [ ] Create `frontend/src/styles/` directory
- [ ] Create `app.css` with imports (§1.2)
- [ ] Create `tokens.css` with full `@theme` block (§17)
- [ ] Add `import "./styles/app.css"` to `main.tsx`

### Phase 2: Typography

- [ ] Download Plus Jakarta Sans (Regular, Medium, SemiBold, Bold) WOFF2 files
- [ ] Download Manrope (Regular, Medium, SemiBold) WOFF2 files
- [ ] Place in `frontend/public/fonts/` per §3.2 directory structure
- [ ] Create `base.css` with `@font-face` declarations (§3.2)
- [ ] Add preload hints to `index.html` (§3.2)

### Phase 3: Base Styles

- [ ] Add reduced motion rules to `base.css` (§7.3)
- [ ] Add global `:focus-visible` rule to `base.css` (§12.1)
- [ ] Add `html` / `body` base styles (font-family, background color, text color)

### Phase 4: Component & Utility Layers

- [ ] Create `components.css` with type scale composite classes (§3.5)
- [ ] Create `utilities.css` with z-index utilities (§10)
- [ ] Add touch-target utility (§14.3)
- [ ] Add print utilities (§15.3)

### Phase 5: Print

- [ ] Create `print.css` with token overrides (§15.1) and hide rules (§15.2)

### Verification Criteria

- [ ] `npm run dev` starts without CSS errors
- [ ] Tailwind utility classes generate correctly (`bg-primary`, `text-on-surface`, etc.)
- [ ] Font files load (check Network tab — no 404s)
- [ ] `type-display-lg` class renders Plus Jakarta Sans Bold at 3.5rem
- [ ] `type-body-lg` class renders Manrope Regular at 1rem with 1.6 line-height
- [ ] `parent:` and `student:` variants work with `data-context` attribute
- [ ] Print preview shows flattened colors and hidden navigation
- [ ] `prefers-reduced-motion: reduce` disables all animations
- [ ] Focus ring is visible on all interactive elements via keyboard navigation
