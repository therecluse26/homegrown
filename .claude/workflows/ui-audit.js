/**
 * ui-audit — Multi-lens UI/UX audit workflow for Homegrown Academy
 *
 * Invocation:
 *   /workflow ui-audit
 *   /workflow ui-audit {"baseUrl":"http://localhost:5673"}
 *   /workflow ui-audit {"lenses":["visual-polish","token-conformance"],"groups":["social","learning"]}
 *
 * Prerequisites:
 *   1. Frontend dev server running:  cd frontend && npm run dev -- --port 5673
 *   2. Backend running:              make dev-api
 *   3. Seed data fresh:              make seed DB=homegrown
 *
 * Architecture:
 *   Phase 1 — Sweep:   pipeline(SURFACE_GROUPS × LENSES) → each agent renders all surfaces
 *                       in its group through its assigned lens at both viewports.
 *   Phase 2 — Verify:  per-finding skeptic agents try to *refute* each finding (default: refuted=true).
 *   Phase 3 — Ledger:  synthesise confirmed findings + emit E9 coverage ledger.
 */

export const meta = {
  name: 'ui-audit',
  description: 'Multi-lens UI audit: lens×group fan-out, adversarial verify, E9 coverage ledger',
  phases: [
    { title: 'Sweep', detail: 'Per-lens agents render and judge each surface group at both viewports (E1–E3, E6, E7)' },
    { title: 'Verify', detail: 'Skeptic agents attempt to refute every finding — default stance: refuted=true' },
    { title: 'Ledger', detail: 'Synthesise confirmed findings and emit E9 coverage ledger' },
  ],
};

// ─── Configuration (override via args) ──────────────────────────────────────

const BASE_URL = (args && args.baseUrl) || 'http://localhost:5673';
const PARENT_EMAIL = (args && args.parentEmail) || 'seed@example.com';
const PARENT_PASS  = (args && args.parentPass)  || 'SeedPassword123!';
const ADMIN_EMAIL  = (args && args.adminEmail)  || 'admin@example.com';
const ADMIN_PASS   = (args && args.adminPass)   || 'SeedPassword123!';

// Filter args: restrict which lenses / surface groups to run (default: all)
const FILTER_LENSES = (args && args.lenses)  || null;
const FILTER_GROUPS = (args && args.groups)  || null;

// ─── Lens definitions ────────────────────────────────────────────────────────

const ALL_LENSES = [
  {
    key: 'visual-polish',
    label: 'Visual Polish',
    focus: `E2 Visual-Quality Rubric (4 criteria, each scored 0–3):
  A — Hierarchy Visible: primary (text-primary/bg-primary), secondary (text-on-surface-variant), tertiary visually subordinate.
  B — Spacing on Token Scale: all gaps from token scale (spacing-1=0.35rem through spacing-16=5.6rem). No arbitrary values.
  C — Ruthless Alignment: every element aligns to a grid, shared baseline, or common edge.
  D — Type System: headlines = font-display (Plus Jakarta Sans), body = font-sans (Manrope). Sizes from type scale only.

Aggregate VQ score 0–3:
  3 = all four criteria pass cleanly (score ≥2 each)
  2 = three criteria pass
  1 = two criteria pass
  0 = zero or one criteria pass — FAIL

Verdict rule: VQ=0 or VQ=1 → FAIL. VQ=2 → WARN. VQ=3 → PASS (assuming no console errors).
Do NOT report E3 token violations here — those belong to the token-conformance lens.
Report every surface that scores VQ < 3 as a finding. Surfaces scoring VQ=3 are still reported in coverageRows but not as findings.`,
  },
  {
    key: 'token-conformance',
    label: 'Token Conformance',
    focus: `E3 Token & Curated-Hearth Conformance Scan — 10-item violation checklist:
  E3-A (High→FAIL): hex color not in token palette. Correct: use text-primary, bg-surface-container, text-error, etc.
  E3-B (High→FAIL): 1px solid borders for sectioning. Forbidden — use bg-surface-container-low vs bg-surface shifts.
  E3-C (High→FAIL): horizontal dividers (hr, divide-y, border-b) between list items. Forbidden — use spacing-4 gap.
  E3-D (Med→WARN): arbitrary z-index (z-[50], inline z-index:999). Use Tailwind named z-scale.
  E3-E (Med→WARN): off-scale spacing (p-[13px], gap-[22px], inline margin:7px). Replace with nearest token step.
  E3-F (Med→WARN): wrong typeface for context (Manrope used for display-lg headline). Headlines: font-display.
  E3-G (Med→WARN): flat drop shadows, tight blur (<8px). Ambient only: 32px+ blur, 4% opacity, on-surface tint.
  E3-H (Low→WARN): wrong card radius. Student cards: rounded-xl (1.5rem). Parent data: rounded-lg (1rem).
  E3-I (Low→WARN): primary buttons not using bg-primary (#0c5252) + text-on-primary (#ffffff).
  E3-J (Low→WARN): nav using opaque bg instead of surface-container-low at 80% + backdrop-blur.

For each violation: cite offending element, observed value, correct token/rule.
Inspect the DOM snapshot and screenshot carefully — token violations may not be visible at a glance.
Look at the computed styles in the snapshot for color, spacing, and border indicators.

**D4 calibration (source-inspection rule):** E3-B and E3-C violations are frequently invisible
in screenshots but detectable in the DOM snapshot. After the visual scan, explicitly look for
class names: \`border-t\`, \`border-b\`, \`divide-y\`, \`divide-*\`, \`hr\` on list containers and
between sibling elements. D4 confirmed this pattern caught 3 High E3-C violations (list dividers
in social/marketplace/settings) that screenshot review alone would have missed.`,
  },
  {
    key: 'wcag-a11y',
    label: 'WCAG / Accessibility',
    focus: `WCAG 2.1 AA accessibility audit. Check ALL of the following:
  1.4.3 Contrast: text vs background must meet 4.5:1 (normal) / 3:1 (large ≥18pt/14pt bold).
       Primary #0c5252 on white #ffffff = 8.9:1 ✓. Watch for light-grey text on white or low-contrast labels.
  2.4.7 Focus visible: every interactive element must show a visible focus ring.
       Focus ring MUST use primary color token (#0c5252), not browser default blue.
  1.1.1 Alt text: all images must have descriptive alt text (not empty unless decorative).
  4.1.2 ARIA: interactive elements need role/name/state. Buttons must have accessible names.
       No orphaned aria-labelledby or aria-describedby references.
  2.1.1 Keyboard: all interactive elements reachable by Tab. No keyboard traps.
  2.5.5 Target size: touch targets ≥ 44×44px (check on mobile viewport).
  1.3.1 Info & relationships: headings in logical h1→h2→h3 order. No skipped levels.
  3.3.1 Error identification: form errors identified and described in text (not color alone).
  1.4.4 Resize text: no content loss up to 200% zoom (check snapshot for overflow).
  2.4.3 Focus order: tab order follows visual reading order (left-to-right, top-to-bottom).

Use mcp__pw__browser_snapshot to inspect the accessibility tree. Tab through interactive elements.
Report each violation with: element, issue, WCAG criterion, severity (critical/high/medium/low).`,
  },
  {
    key: 'interaction-flow',
    label: 'Interaction & Flow',
    focus: `E7 State-Coverage Matrix + interaction quality. For each surface:
  States to exercise:
    Default   — normal loaded state with seed data
    Loading   — in-flight skeleton/spinner (observe on first render or slow connection simulation)
    Empty     — no data (filter to empty result, or navigate empty collection)
    Error     — API error state (disconnect backend or use invalid ID)
    No-perm   — unauthorized access attempt (wrong role)
    Overflow  — long text, many items
    Focus/hover/active — tab through, hover buttons/cards

  Interaction quality checks:
    - Loading states: skeleton or spinner present (not blank white). Score VQ=0 if absent.
    - Empty states: designed empty state (not raw "No items found" text). VQ=0 if absent.
    - Error states: designed error component (not blank page or raw JS error). VQ=0 if absent.
    - Form validation: inline errors on invalid submit. Errors use text-error (#ba1a1a).
    - Navigation transitions: no jarring jumps or flashes.
    - Button/link feedback: hover and active states visible within 100ms.
    - Modals and drawers: focus trapped inside, Escape closes, background dimmed.
    - Optimistic UI or loading feedback on write operations.

  Report missing/broken states as findings. Mark states not exercised in coverageRows with "—" and a reason.`,
  },
  {
    key: 'content-ia',
    label: 'Content & IA',
    focus: `Information architecture, content quality, and mental model alignment:
  1. Navigation IA: does the nav structure match the user's mental model for a homeschooling app?
     Check that section labels are plain-language (not jargon). F-pattern scan — is primary content in top-left?
  2. Page titles & headings: every page has an h1 that describes the current context.
     Breadcrumbs or back-navigation where depth > 1 level.
  3. Copy quality: Plain Language — 8th-grade reading level or below for instructional copy.
     No passive voice in CTAs. Button labels say what happens ("Save changes", not "OK").
     Error messages explain the cause and suggest a fix (not just "Error occurred").
  4. Empty-state copy: includes a clear call-to-action, not just "Nothing here."
  5. Form labels: every input has a visible label (not just placeholder). Labels precede their inputs.
  6. Progressive disclosure: complex forms and workflows reveal steps incrementally.
     No wall-of-form-fields on first load.
  7. Inverted pyramid: most important info first on each page. Supporting detail below.
  8. Scan-ability: lists use bullets or cards, not dense paragraphs.
  9. Confirmation patterns: destructive actions (delete, cancel subscription) have a confirm step.
  10. Zeigarnik / Goal-gradient: multi-step flows show progress indicator.

  Report each IA/content defect with: surface, element, issue, severity, and suggested fix.`,
  },
];

// ─── Surface groups ───────────────────────────────────────────────────────────

const ALL_SURFACE_GROUPS = [
  {
    key: 'auth-legal',
    label: 'Auth & Legal',
    auth: 'none',
    surfaces: [
      { id: 'A1', route: '/auth/login',                          label: 'Login' },
      { id: 'A2', route: '/auth/register',                       label: 'Register' },
      { id: 'A3', route: '/auth/recovery',                       label: 'Account Recovery' },
      { id: 'A4', route: '/auth/verification',                   label: 'Email Verification' },
      { id: 'A5', route: '/auth/coppa/verify',                   label: 'COPPA Micro-Charge' },
      { id: 'A6', route: '/auth/accept-invite/test-token-123',   label: 'Accept Invitation' },
      { id: 'L1', route: '/legal/terms',                         label: 'Terms of Service' },
      { id: 'L2', route: '/legal/privacy',                       label: 'Privacy Policy' },
      { id: 'L3', route: '/legal/guidelines',                    label: 'Community Guidelines' },
    ],
  },
  {
    key: 'social',
    label: 'Social & Notifications',
    auth: 'parent',
    surfaces: [
      { id: 'O1',  route: '/onboarding',                                              label: 'Onboarding Wizard' },
      { id: 'S1',  route: '/',                                                         label: 'Feed' },
      { id: 'S2',  route: '/friends',                                                   label: 'Friends List' },
      { id: 'S3',  route: '/friends/discover',                                          label: 'Friend Discovery' },
      { id: 'S4',  route: '/messages',                                                   label: 'Direct Messages' },
      { id: 'S5',  route: '/messages/01900000-0000-7000-8000-000000000091',              label: 'Conversation' },
      { id: 'S6',  route: '/groups',                                                      label: 'Groups List' },
      { id: 'S7',  route: '/groups/new',                                                  label: 'Create Group' },
      { id: 'S8',  route: '/groups/01900000-0000-7000-8000-000000000051',                 label: 'Group Detail' },
      { id: 'S9',  route: '/groups/01900000-0000-7000-8000-000000000051/manage',          label: 'Group Management' },
      { id: 'S10', route: '/events',                                                       label: 'Events List' },
      { id: 'S11', route: '/events/new',                                                   label: 'Create Event' },
      { id: 'S12', route: '/events/01900000-0000-7000-8000-000000000111',                  label: 'Event Detail' },
      { id: 'S13', route: '/post/01900000-0000-7000-8000-000000000061',                    label: 'Post Detail' },
      { id: 'S14', route: '/family/01900000-0000-7000-8000-000000000001',                  label: 'Family Profile' },
      { id: 'OA1', route: '/recommendations',                                               label: 'Recommendations' },
      { id: 'OA2', route: '/search',                                                         label: 'Search Results' },
      { id: 'OA3', route: '/notifications',                                                   label: 'Notification Center' },
    ],
  },
  {
    key: 'learning',
    label: 'Learning',
    auth: 'parent',
    surfaces: [
      { id: 'LR1',  route: '/learning',                                                               label: 'Learning Dashboard' },
      { id: 'LR2',  route: '/learning/activities',                                                     label: 'Activity Log' },
      { id: 'LR3',  route: '/learning/journals',                                                        label: 'Journal List' },
      { id: 'LR4',  route: '/learning/journals/new',                                                    label: 'New Journal' },
      { id: 'LR5',  route: '/learning/reading-lists',                                                   label: 'Reading Lists' },
      { id: 'LR6',  route: '/learning/progress/01900000-0000-7000-8000-000000000021',                   label: 'Progress (Emma)' },
      { id: 'LR7',  route: '/learning/progress/01900000-0000-7000-8000-000000000022',                   label: 'Progress (James)' },
      { id: 'LR8',  route: '/learning/grades',                                                           label: 'Tests & Grades' },
      { id: 'LR9',  route: '/learning/quiz/01900000-0000-7000-8000-000000000381',                        label: 'Quiz Player' },
      { id: 'LR10', route: '/learning/video/01900000-0000-7000-8000-000000000306',                       label: 'Video Player' },
      { id: 'LR11', route: '/learning/read/01900000-0000-7000-8000-000000000301',                        label: 'Content Viewer' },
      { id: 'LR12', route: '/learning/sequence/01900000-0000-7000-8000-000000000385',                    label: 'Sequence View' },
      { id: 'LR13', route: '/learning/session-log/01900000-0000-7000-8000-000000000033',                 label: 'Session Log' },
      { id: 'LR14', route: '/learning/session',                                                           label: 'Session Launcher' },
      { id: 'LR15', route: '/learning/projects',                                                          label: 'Projects' },
      { id: 'LR16', route: '/learning/tools',                                                             label: 'Tool Assignment' },
      { id: 'LR17', route: '/learning/nature-journal',                                                    label: 'Nature Journal' },
      { id: 'LR18', route: '/learning/trivium-tracker',                                                   label: 'Trivium Tracker' },
      { id: 'LR19', route: '/learning/rhythm-planner',                                                    label: 'Rhythm Planner' },
      { id: 'LR20', route: '/learning/observation-logs',                                                  label: 'Observation Logs' },
      { id: 'LR21', route: '/learning/habit-tracking',                                                    label: 'Habit Tracking' },
      { id: 'LR22', route: '/learning/interest-led-log',                                                  label: 'Interest-Led Log' },
      { id: 'LR23', route: '/learning/handwork-projects',                                                 label: 'Handwork Projects' },
      { id: 'LR24', route: '/learning/practical-life',                                                    label: 'Practical Life' },
      { id: 'LR25', route: '/learning/quiz/01900000-0000-7000-8000-000000000381/score',                   label: 'Parent Quiz Scoring' },
    ],
  },
  {
    key: 'marketplace-creator',
    label: 'Marketplace & Creator',
    auth: 'parent',
    surfaces: [
      { id: 'MK1', route: '/marketplace',                                                                      label: 'Browse' },
      { id: 'MK2', route: '/marketplace/listings/01900000-0000-7000-8000-000000000211',                        label: 'Listing Detail' },
      { id: 'MK3', route: '/marketplace/cart',                                                                  label: 'Cart' },
      { id: 'MK4', route: '/marketplace/purchases',                                                             label: 'Purchase History' },
      { id: 'MK5', route: '/marketplace/purchases/01900000-0000-7000-8000-000000000221/refund',                 label: 'Refund Request' },
      { id: 'MK6', route: '/marketplace/listings/01900000-0000-7000-8000-000000000211/versions',                label: 'Listing Versions' },
      { id: 'CR1', route: '/creator',                                                                            label: 'Creator Dashboard' },
      { id: 'CR2', route: '/creator/listings/new',                                                               label: 'Create Listing' },
      { id: 'CR3', route: '/creator/listings/01900000-0000-7000-8000-000000000211/edit',                        label: 'Edit Listing' },
      { id: 'CR4', route: '/creator/quiz-builder',                                                               label: 'Quiz Builder (new)' },
      { id: 'CR5', route: '/creator/quiz-builder/01900000-0000-7000-8000-000000000381',                         label: 'Quiz Builder (edit)' },
      { id: 'CR6', route: '/creator/sequence-builder',                                                           label: 'Sequence Builder (new)' },
      { id: 'CR7', route: '/creator/sequence-builder/01900000-0000-7000-8000-000000000385',                     label: 'Sequence Builder (edit)' },
      { id: 'CR8', route: '/creator/payouts',                                                                    label: 'Payout Setup' },
      { id: 'CR9', route: '/creator/verification',                                                               label: 'Creator Verification' },
      { id: 'CR10', route: '/creator/reviews',                                                                   label: 'Creator Reviews' },
    ],
  },
  {
    key: 'billing-settings',
    label: 'Billing & Settings',
    auth: 'parent',
    surfaces: [
      { id: 'B1',  route: '/billing',                                                                            label: 'Pricing Page' },
      { id: 'B2',  route: '/billing/payment-methods',                                                            label: 'Payment Methods' },
      { id: 'B3',  route: '/billing/transactions',                                                               label: 'Transaction History' },
      { id: 'B4',  route: '/billing/subscription',                                                               label: 'Subscription Mgmt' },
      { id: 'B5',  route: '/billing/invoices',                                                                   label: 'Invoice History' },
      { id: 'ST1', route: '/settings',                                                                            label: 'Family Settings' },
      { id: 'ST2', route: '/settings/notifications',                                                              label: 'Notification Prefs' },
      { id: 'ST3', route: '/settings/notifications/history',                                                      label: 'Notification History' },
      { id: 'ST4', route: '/settings/subscription',                                                               label: 'Subscription Upgrade' },
      { id: 'ST5', route: '/settings/account',                                                                    label: 'Account Settings' },
      { id: 'ST6', route: '/settings/account/sessions',                                                           label: 'Session Management' },
      { id: 'ST7', route: '/settings/account/export',                                                             label: 'Data Export' },
      { id: 'ST8', route: '/settings/account/delete',                                                             label: 'Account Deletion' },
      { id: 'ST9', route: '/settings/account/delete/student/01900000-0000-7000-8000-000000000021', label: 'Student Deletion' },
      { id: 'ST10', route: '/settings/account/appeals',                                                           label: 'Moderation Appeals' },
      { id: 'ST11', route: '/settings/blocks',                                                                    label: 'Block Management' },
      { id: 'ST12', route: '/settings/privacy',                                                                   label: 'Privacy Controls' },
      { id: 'ST13', route: '/settings/account/mfa',                                                               label: 'MFA Setup' },
      { id: 'ST14', route: '/settings/subscription/manage',                                                       label: 'Subscription Manager' },
    ],
  },
  {
    key: 'planning-compliance-admin',
    label: 'Planning, Compliance & Admin',
    auth: 'mixed',
    surfaces: [
      { id: 'PL1', route: '/calendar',                                                                   label: 'Calendar View', auth: 'parent' },
      { id: 'PL2', route: '/calendar/day/2026-04-05',                                                   label: 'Calendar Day', auth: 'parent' },
      { id: 'PL3', route: '/calendar/week/2026-04-05',                                                  label: 'Calendar Week', auth: 'parent' },
      { id: 'PL4', route: '/schedule/new',                                                               label: 'New Schedule Item', auth: 'parent' },
      { id: 'PL5', route: '/schedule/01900000-0000-7000-8000-000000000801/edit',                        label: 'Edit Schedule Item', auth: 'parent' },
      { id: 'PL6', route: '/planning/templates',                                                          label: 'Schedule Templates', auth: 'parent' },
      { id: 'PL7', route: '/planning/print',                                                              label: 'Print Schedule', auth: 'parent' },
      { id: 'PL8', route: '/planning/coop',                                                               label: 'Co-op Coordination', auth: 'parent' },
      { id: 'CP1', route: '/compliance',                                                                   label: 'Compliance Setup', auth: 'parent' },
      { id: 'CP2', route: '/compliance/attendance',                                                        label: 'Attendance Tracker', auth: 'parent' },
      { id: 'CP3', route: '/compliance/assessments',                                                       label: 'Assessment Records', auth: 'parent' },
      { id: 'CP4', route: '/compliance/tests',                                                             label: 'Standardized Tests', auth: 'parent' },
      { id: 'CP5', route: '/compliance/portfolios',                                                        label: 'Portfolio List', auth: 'parent' },
      { id: 'CP6', route: '/compliance/portfolios/01900000-0000-7000-8000-000000000850',                  label: 'Portfolio Builder', auth: 'parent' },
      { id: 'CP7', route: '/compliance/transcripts',                                                       label: 'Transcript List', auth: 'parent' },
      { id: 'CP8', route: '/compliance/transcripts/01900000-0000-7000-8000-000000000840',                 label: 'Transcript Builder', auth: 'parent' },
      { id: 'AD1', route: '/admin',                                                                         label: 'Admin Dashboard', auth: 'admin' },
      { id: 'AD2', route: '/admin/users',                                                                   label: 'User Management', auth: 'admin' },
      { id: 'AD3', route: '/admin/users/01900000-0000-7000-8000-000000000011',                             label: 'User Detail', auth: 'admin' },
      { id: 'AD4', route: '/admin/moderation',                                                             label: 'Moderation Queue', auth: 'admin' },
      { id: 'AD5', route: '/admin/flags',                                                                  label: 'Feature Flags', auth: 'admin' },
      { id: 'AD6', route: '/admin/audit',                                                                  label: 'Audit Log', auth: 'admin' },
      { id: 'AD7', route: '/admin/methodologies',                                                          label: 'Methodology Config', auth: 'admin' },
    ],
  },
];

// ─── Structured output schemas ───────────────────────────────────────────────

const FINDINGS_SCHEMA = {
  type: 'object',
  required: ['findings', 'coverageRows'],
  properties: {
    findings: {
      type: 'array',
      items: {
        type: 'object',
        required: ['surface', 'route', 'viewport', 'lens', 'rubricItem', 'severity', 'title', 'observed', 'expected', 'verdict'],
        properties: {
          surface:      { type: 'string', description: 'Surface ID, e.g. S1, LR3' },
          route:        { type: 'string', description: 'URL path, e.g. /learning/journals' },
          viewport:     { type: 'string', enum: ['desktop', 'mobile', 'both'] },
          lens:         { type: 'string', enum: ['visual-polish', 'token-conformance', 'wcag-a11y', 'interaction-flow', 'content-ia'] },
          rubricItem:   { type: 'string', description: 'E.g. E2-B, E3-C, WCAG 2.4.7, E7-empty' },
          severity:     { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
          title:        { type: 'string', description: 'Short defect title' },
          observed:     { type: 'string', description: 'What was seen' },
          expected:     { type: 'string', description: 'What should have been seen' },
          correctToken: { type: 'string', description: 'Correct design token or rule (for E3 violations)' },
          screenshotRef:{ type: 'string', description: 'Screenshot filename hint, e.g. social-S1-desktop' },
          vqScore:      { type: 'string', description: 'Visual quality score, e.g. 1/3' },
          e3Violations: { type: 'array', items: { type: 'string' }, description: 'E3-A through E3-J codes found' },
          verdict:      { type: 'string', enum: ['PASS', 'WARN', 'FAIL', 'BLOCKED'] },
        },
      },
    },
    coverageRows: {
      type: 'array',
      description: 'One row per surface × viewport — the E9 coverage ledger input',
      items: {
        type: 'object',
        required: ['surface', 'route', 'viewport', 'defaultState', 'verdict', 'vqScore'],
        properties: {
          surface:       { type: 'string' },
          route:         { type: 'string' },
          viewport:      { type: 'string', enum: ['desktop', 'mobile'] },
          defaultState:  { type: 'string', enum: ['✓', 'N/A', '—'] },
          loadingState:  { type: 'string', enum: ['✓', 'N/A', '—'] },
          emptyState:    { type: 'string', enum: ['✓', 'N/A', '—'] },
          errorState:    { type: 'string', enum: ['✓', 'N/A', '—'] },
          noPermState:   { type: 'string', enum: ['✓', 'N/A', '—'] },
          overflowState: { type: 'string', enum: ['✓', 'N/A', '—'] },
          focusHoverState:{ type: 'string', enum: ['✓', 'N/A', '—'] },
          notes:         { type: 'string' },
          vqScore:       { type: 'string', description: 'E.g. 3/3, 2/3, 1/3, 0/3, or N/A' },
          verdict:       { type: 'string', enum: ['PASS', 'WARN', 'FAIL', 'BLOCKED'] },
        },
      },
    },
  },
};

const ADVERSARIAL_SCHEMA = {
  type: 'object',
  required: ['findingKey', 'refuted', 'confidence', 'refutationReason', 'confirmedSeverity'],
  properties: {
    findingKey:         { type: 'string', description: 'surface:viewport:rubricItem identifier' },
    refuted:            { type: 'boolean', description: 'true = finding is wrong; false = finding is confirmed' },
    confidence:         { type: 'string', enum: ['high', 'medium', 'low'] },
    refutationReason:   { type: 'string', description: 'Why the finding is wrong (if refuted) or why it survives skepticism (if not)' },
    confirmedSeverity:  { type: 'string', enum: ['critical', 'high', 'medium', 'low'] },
  },
};

// ─── Prompt builders ──────────────────────────────────────────────────────────

function buildAuthInstructions(auth) {
  if (auth === 'none') {
    return 'These surfaces are public — do NOT log in. Navigate directly to each URL.';
  }
  if (auth === 'admin') {
    return `Log in as the ADMIN account before navigating to any surface:
1. mcp__pw__browser_navigate → ${BASE_URL}/auth/login
2. mcp__pw__browser_fill_form → { "Email": "${ADMIN_EMAIL}", "Password": "${ADMIN_PASS}" }
3. mcp__pw__browser_click → Submit
4. mcp__pw__browser_wait_for → text visible: "Admin"
5. Handle onboarding gate if redirected: click "Skip all"`;
  }
  return `Log in as the PARENT account before navigating to any surface:
1. mcp__pw__browser_navigate → ${BASE_URL}/auth/login
2. mcp__pw__browser_fill_form → { "Email": "${PARENT_EMAIL}", "Password": "${PARENT_PASS}" }
3. mcp__pw__browser_click → Submit
4. mcp__pw__browser_wait_for → text visible: "Feed" or "/" content
5. If redirected to /onboarding: click "Skip all", then continue`;
}

function buildMixedAuthInstructions() {
  return `This group contains both parent and admin surfaces.
- Surfaces with auth=parent: log in as ${PARENT_EMAIL} / ${PARENT_PASS}
- Surfaces with auth=admin (AD1–AD7): switch to ${ADMIN_EMAIL} / ${ADMIN_PASS}

Login procedure:
1. mcp__pw__browser_navigate → ${BASE_URL}/auth/login
2. mcp__pw__browser_fill_form → { "Email": "<email>", "Password": "<pass>" }
3. mcp__pw__browser_click → Submit
4. Handle onboarding gate: click "Skip all" if redirected to /onboarding

Switch accounts by navigating to the login page again and re-authenticating.`;
}

function buildSweepPrompt(group, lens) {
  const surfaceList = group.surfaces.map(s =>
    `  ${s.id}: ${BASE_URL}${s.route}  (${s.label})`
  ).join('\n');

  const authInstructions = group.auth === 'mixed'
    ? buildMixedAuthInstructions()
    : buildAuthInstructions(group.auth);

  return `You are a specialized UI auditor for the Homegrown Academy homeschooling platform.

## Your assignment
- Surface group: ${group.label}
- Lens: ${lens.label}
- Viewports: Desktop (1440×900) AND Mobile (390×844)

## Authentication
${authInstructions}

## The E1 Render-AND-Judge Gate (MANDATORY)
For every surface you MUST follow this exact pipeline:
  RENDER → JUDGE → FILE

Never file a verdict without completing judgment. "Content present, no console errors" is NOT sufficient.

For each surface at each viewport:
1. mcp__pw__browser_resize → { width: 1440, height: 900 }  [or 390 × 844 for mobile]
2. mcp__pw__browser_navigate → the surface URL
3. mcp__pw__browser_wait_for → page to stabilise (look for main content)
4. mcp__pw__browser_snapshot → capture accessibility tree
5. mcp__pw__browser_console_messages → level: "error" (note any JS errors)
6. mcp__pw__browser_take_screenshot → type: "png"  ← REQUIRED
7. JUDGE using the lens rubric below
8. FILE verdict: PASS / WARN / FAIL / BLOCKED

Verdict definitions:
  PASS    — renders, VQ ≥ 2/3, no critical E3 violations, no console errors
  WARN    — renders but VQ = 1/3, or console warnings, or medium/low E3 violations
  FAIL    — page crashes / JS error / VQ = 0/3 / critical E3 violation (E3-A, E3-B, E3-C)
  BLOCKED — cannot reach (auth issue, redirect loop, requires token auth)

## Your lens: ${lens.label}
${lens.focus}

## Surfaces to audit
${surfaceList}

## Output requirements
Return structured output via StructuredOutput. Your output must include:
- findings[]: every defect you find (even VQ=3 surfaces may have findings for this lens)
  - Only include PASS surfaces in findings if they have a specific issue for THIS lens.
  - VQ=0 or VQ=1 surfaces MUST appear as findings.
  - For each finding: surface, route, viewport, lens="${lens.key}", rubricItem, severity, title, observed, expected, correctToken (if applicable), screenshotRef hint, vqScore, e3Violations[], verdict
- coverageRows[]: one row per surface × viewport (both 'desktop' and 'mobile' rows for every surface)
  - All 7 state columns (use "✓" tested, "N/A" impossible, "—" not tested this run)
  - notes: reason for any "—" or "N/A" cells
  - vqScore: the VQ for this surface (e.g. "3/3", "2/3", "1/3", "0/3", or "N/A" if BLOCKED)
  - verdict: PASS / WARN / FAIL / BLOCKED

Screenshot naming convention: {group}-{id}-{d|m}-{verdict}.png
Example: auth-legal-A1-d-PASS.png, social-S5-m-WARN.png

IMPORTANT: Do not skip any surface in the list. If you cannot reach a surface, mark it BLOCKED in coverageRows and explain in notes. Coverage ledger must have zero silent omissions.`;
}

function buildVerifyPrompt(finding) {
  const key = `${finding.surface}:${finding.viewport}:${finding.rubricItem}`;
  return `You are an adversarial UI reviewer. Your job is to REFUTE the following finding.

Default stance: assume the finding is WRONG. Try hard to prove it is a false positive before you accept it as real.

## Finding to challenge
- Key: ${key}
- Surface: ${finding.surface} (${finding.route})
- Viewport: ${finding.viewport}
- Lens: ${finding.lens}
- Rubric item: ${finding.rubricItem}
- Severity: ${finding.severity}
- Title: ${finding.title}
- Observed: ${finding.observed}
- Expected: ${finding.expected}
${finding.correctToken ? `- Correct token: ${finding.correctToken}` : ''}
${finding.vqScore ? `- VQ score claimed: ${finding.vqScore}` : ''}
${finding.e3Violations && finding.e3Violations.length ? `- E3 violations claimed: ${finding.e3Violations.join(', ')}` : ''}

## How to challenge it
1. Navigate to the surface: ${BASE_URL}${finding.route}
2. Resize to the ${finding.viewport} viewport (desktop: 1440×900, mobile: 390×844)
3. Take a screenshot and inspect carefully
4. Look for evidence that contradicts the finding:
   - Is the claimed value actually correct per the token spec?
   - Was the screenshot taken in an intermediate state (loading, transition)?
   - Is the element the finding references actually present at this viewport?
   - Does the rubric item genuinely apply to this surface type?
   - Could the finding be a misread of an accessibility tree artifact?

## Rubric for acceptance
- If you find clear evidence the finding is a false positive → refuted=true
- If you can partially confirm the finding but at lower severity → refuted=false, adjust confirmedSeverity
- If the finding is clearly correct → refuted=false, confirmedSeverity matches original

Return your verdict as structured output. Be specific in refutationReason.`;
}

// ─── Apply filters ────────────────────────────────────────────────────────────

const LENSES = FILTER_LENSES
  ? ALL_LENSES.filter(l => FILTER_LENSES.includes(l.key))
  : ALL_LENSES;

const SURFACE_GROUPS = FILTER_GROUPS
  ? ALL_SURFACE_GROUPS.filter(g => FILTER_GROUPS.includes(g.key))
  : ALL_SURFACE_GROUPS;

// ─── Work unit matrix: lens × group ─────────────────────────────────────────

const WORK_UNITS = [];
for (const lens of LENSES) {
  for (const group of SURFACE_GROUPS) {
    WORK_UNITS.push({ lens, group });
  }
}

const totalSurfaces = SURFACE_GROUPS.reduce((n, g) => n + g.surfaces.length, 0);
log(`Audit matrix: ${LENSES.length} lens(es) × ${SURFACE_GROUPS.length} group(s) = ${WORK_UNITS.length} sweep agents`);
log(`Total surfaces: ${totalSurfaces} × 2 viewports = ${totalSurfaces * 2} coverage rows expected`);

// ─── Phase 1 + 2: Sweep then adversarial verify (pipeline — no barrier) ──────

phase('Sweep');

const allResults = await pipeline(
  WORK_UNITS,

  // Stage 1 — sweep
  (unit) => agent(
    buildSweepPrompt(unit.group, unit.lens),
    {
      label: `sweep:${unit.lens.key}:${unit.group.key}`,
      phase: 'Sweep',
      schema: FINDINGS_SCHEMA,
    }
  ),

  // Stage 2 — adversarial verify (starts as soon as Stage 1 emits for each unit)
  (sweep, unit) => {
    if (!sweep || !sweep.findings || sweep.findings.length === 0) {
      log(`No findings from sweep:${unit.lens.key}:${unit.group.key} — skipping verify stage`);
      return { findings: [], coverageRows: sweep ? sweep.coverageRows : [], verified: [] };
    }

    phase('Verify');
    return parallel(
      sweep.findings.map((finding, idx) => () =>
        agent(
          buildVerifyPrompt(finding),
          {
            label: `verify:${finding.surface}:${finding.viewport}:${finding.rubricItem}`,
            phase: 'Verify',
            schema: ADVERSARIAL_SCHEMA,
          }
        ).then(verdict => ({ finding, verdict }))
      )
    ).then(verdicts => ({
      findings: sweep.findings,
      coverageRows: sweep.coverageRows,
      verified: verdicts.filter(Boolean),
    }));
  }
);

// ─── Phase 3: Synthesise ledger ───────────────────────────────────────────────

phase('Ledger');

// Flatten all results, filter nulls
const validResults = allResults.filter(Boolean);

// Collect all coverage rows (dedupe by surface+viewport — last writer wins per group)
const ledgerMap = new Map();
for (const result of validResults) {
  for (const row of (result.coverageRows || [])) {
    const key = `${row.surface}:${row.viewport}`;
    if (!ledgerMap.has(key)) {
      ledgerMap.set(key, row);
    } else {
      // Merge: take the worst verdict and union of tested states
      const existing = ledgerMap.get(key);
      const verdictRank = { FAIL: 0, BLOCKED: 1, WARN: 2, PASS: 3 };
      if ((verdictRank[row.verdict] || 3) < (verdictRank[existing.verdict] || 3)) {
        existing.verdict = row.verdict;
      }
      // State columns: ✓ > N/A > —
      const stateRank = { '✓': 2, 'N/A': 1, '—': 0 };
      for (const col of ['defaultState','loadingState','emptyState','errorState','noPermState','overflowState','focusHoverState']) {
        if ((stateRank[row[col]] || 0) > (stateRank[existing[col]] || 0)) {
          existing[col] = row[col];
        }
      }
    }
  }
}

// Collect confirmed / refuted findings
const confirmedFindings = [];
const refutedFindings   = [];
for (const result of validResults) {
  for (const { finding, verdict } of (result.verified || [])) {
    if (!verdict || verdict.refuted) {
      refutedFindings.push({ finding, verdict });
    } else {
      confirmedFindings.push({
        ...finding,
        severity: verdict.confirmedSeverity || finding.severity,
        adversarialNote: verdict.refutationReason,
      });
    }
  }
}

// Sort confirmed findings: critical → high → medium → low
const SEV_ORDER = { critical: 0, high: 1, medium: 2, low: 3 };
confirmedFindings.sort((a, b) => (SEV_ORDER[a.severity] || 3) - (SEV_ORDER[b.severity] || 3));

// Build coverage ledger rows (sorted by surface id then viewport)
const ledgerRows = Array.from(ledgerMap.values()).sort((a, b) => {
  const aNum = parseInt(a.surface.replace(/\D/g, ''), 10) || 0;
  const bNum = parseInt(b.surface.replace(/\D/g, ''), 10) || 0;
  if (a.surface.replace(/\d/g, '') !== b.surface.replace(/\d/g, '')) {
    return a.surface.localeCompare(b.surface);
  }
  if (aNum !== bNum) return aNum - bNum;
  return a.viewport.localeCompare(b.viewport);
});

// Coverage statistics
const totalExpected = totalSurfaces * 2; // both viewports
const totalCovered  = ledgerRows.length;
const silentTruncationCount = totalExpected - totalCovered;

// Verdict distribution
const verdictCounts = { PASS: 0, WARN: 0, FAIL: 0, BLOCKED: 0 };
const vqDist        = { '3/3': 0, '2/3': 0, '1/3': 0, '0/3': 0, 'N/A': 0 };
for (const row of ledgerRows) {
  if (row.verdict) verdictCounts[row.verdict] = (verdictCounts[row.verdict] || 0) + 1;
  if (row.vqScore) vqDist[row.vqScore]         = (vqDist[row.vqScore] || 0) + 1;
}

// Severity distribution
const sevCounts = { critical: 0, high: 0, medium: 0, low: 0 };
for (const f of confirmedFindings) {
  sevCounts[f.severity] = (sevCounts[f.severity] || 0) + 1;
}

// Lens breakdown
const lensCounts = {};
for (const f of confirmedFindings) {
  lensCounts[f.lens] = (lensCounts[f.lens] || 0) + 1;
}

// E3 violation counts by category
const e3Counts = {};
for (const f of confirmedFindings) {
  for (const code of (f.e3Violations || [])) {
    e3Counts[code] = (e3Counts[code] || 0) + 1;
  }
}

log(`Sweep complete. Findings: ${confirmedFindings.length} confirmed / ${refutedFindings.length} refuted by adversarial pass`);
log(`Coverage: ${totalCovered}/${totalExpected} rows. Silent truncation count: ${silentTruncationCount}`);

// ─── Final report ─────────────────────────────────────────────────────────────

const report = {
  meta: {
    baseUrl:           BASE_URL,
    lensesRun:         LENSES.map(l => l.key),
    groupsRun:         SURFACE_GROUPS.map(g => g.key),
    sweepAgentsLaunched: WORK_UNITS.length,
  },
  summary: {
    surfacesTotal:     totalSurfaces,
    coverageRowsTotal: totalExpected,
    coverageRowsFiled: totalCovered,
    silentTruncationCount,
    verdictCounts,
    vqDistribution:    vqDist,
    findingsRaw:       validResults.reduce((n, r) => n + (r.findings || []).length, 0),
    findingsConfirmed: confirmedFindings.length,
    findingsRefuted:   refutedFindings.length,
    bySeverity:        sevCounts,
    byLens:            lensCounts,
    e3ViolationCodes:  e3Counts,
  },
  confirmedFindings,
  refutedFindings: refutedFindings.map(r => ({
    surface: r.finding.surface,
    route:   r.finding.route,
    lens:    r.finding.lens,
    title:   r.finding.title,
    reason:  r.verdict ? r.verdict.refutationReason : 'No verdict returned',
  })),
  coverageLedger: ledgerRows,
};

return report;
