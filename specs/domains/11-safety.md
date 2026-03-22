# Domain Spec 11 — Trust & Safety (safety::)

## §1 Overview

The Trust & Safety domain is the **content moderation, CSAM detection, user reporting, and account
enforcement engine** — it protects the platform's users (especially children) from harmful content,
abusive behavior, and bot-driven spam. It owns the full moderation pipeline from automated screening
through human review, the CSAM detection and NCMEC reporting pipeline (a federal obligation under
18 U.S.C. § 2258A), account suspension/ban mechanics, the appeals process, and bot detection
signals. It provides the `SafetyScanAdapter` consumed by `media::` for upload-time content
screening, and the `RequireAdmin` extractor consumed by all domains needing admin-only
endpoints. `[S§12, V§7]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/safety/` |
| **DB prefix** | `safety_` `[ARCH §5.1]` |
| **Complexity class** | Complex (has `domain/` subdirectory) — moderation state machine, CSAM handling pipeline `[ARCH §4.5]` |
| **CQRS** | No — read and write paths are straightforward; admin queries use the same model |
| **External adapters** | `internal/safety/adapters/thorn.go` (Thorn Safer — CSAM hash matching + NCMEC reporting) `[ARCH §2.13]`, `internal/safety/adapters/rekognition.go` (AWS Rekognition — content moderation labels) `[ARCH §2.13]` |
| **Key constraint** | CSAM reporting is a federal legal obligation `[S§12.1]`; every user-data query family-scoped via `FamilyScope` `[CODING §2.4, §2.5]`; moderation actions are immutable audit records; zero user notification on CSAM detection `[S§12.1]` |

**What safety:: owns**: Content moderation pipeline (automated screening + community reports +
human review), CSAM detection adapter integration (Thorn Safer PhotoDNA), NCMEC report filing,
content flag records, moderation action records (immutable audit trail), account suspension and ban
mechanics, account status enforcement (auth middleware integration via Redis), user reporting
system (report submission + admin review queue), appeals process, bot behavioral detection signals,
`SafetyScanAdapter` implementation consumed by `media::`, `RequireAdmin` extractor definition
(in `00-core`, backed by `iam_parents.is_platform_admin`), text content scanning (keyword +
regex in Phase 1, ML in Phase 2), community guidelines enforcement.

**What safety:: does NOT own**: CAPTCHA on registration (owned by `iam::` — delegated from
safety:: requirements) `[S§12.4]`, rate limiting (owned by `00-core` middleware) `[00-core §13.2]`,
file upload lifecycle and storage (owned by `media::`) `[09-media]`, social content CRUD (owned by
`social::`) `[05-social]`, marketplace listing/review CRUD (owned by `mkt::`) `[07-mkt]`,
notification delivery (owned by `notify::`) `[08-notify]`, user accounts and family membership
(owned by `iam::`) `[01-iam]`, group-level moderation roles (owned by `social::` — group admins
moderate their own groups) `[S§12.7]`.

**What safety:: delegates**: Notification delivery → `notify::` (via domain events).
User/family lookup → `iam::IamService`. Session revocation → `iam::KratosAdapter::revoke_sessions()`.
Background job scheduling → hibiken/asynq `[ARCH §12]`. CSAM hash matching API calls → Thorn Safer
(external service). Content moderation label detection → AWS Rekognition (external service).
CAPTCHA enforcement on registration → `iam::` (iam:: implements the CAPTCHA check during
registration flow per safety:: requirements).

---

## §2 Requirements Traceability

Every SPEC.md §12 requirement and §3.1.5 (admin) requirement maps to a section in this document.
Cross-references from other spec sections are included where the safety domain is involved.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Automated CSAM detection on all uploaded images/videos | `[S§12.1]` | §10 (CSAM detection pipeline) |
| Immediate removal of confirmed/suspected CSAM | `[S§12.1]` | §10.2 (quarantine step) |
| NCMEC reporting as required by federal law | `[S§12.1]` | §10.3 (NCMEC filing via Thorn) |
| Evidence preservation as required by law enforcement | `[S§12.1]` | §10.4 (immutable evidence) |
| No notification to offending user | `[S§12.1]` | §10.5 (zero user notification) |
| Immediate permanent suspension of CSAM-associated accounts | `[S§12.1]` | §10.6 (permanent ban), §12.2 (ban state) |
| Zero-tolerance CSAM policy | `[S§12.1]` | §10 (entire pipeline) |
| Automated screening of all UGC (posts, comments, messages, reviews, listings, questions, quizzes, sequences) | `[S§12.2]` | §11.1 (text scanning), §11.2 (media scanning via events), §11.2.1 (nudity auto-rejection), §11.2.2 (label routing), §11.2.3 (creator-authored interactive content) |
| Nudity/explicit content auto-rejected on upload | `[S§12.2]` | §11.2.1 (nudity auto-rejection policy) |
| Admin-confirmed novel CSAM triggers NCMEC pipeline | `[S§12.1]` | §11.4.1 (CSAM escalation from review queue) |
| Rekognition label routing (auto-reject / flag / ignore) | `[S§12.2]` | §11.2.2 (label routing decision table) |
| Community reporting system | `[S§12.3]` | §11.3 (user reporting flow) |
| Human review of flagged/reported content | `[S§12.3]` | §11.4 (admin review queue) |
| Moderator actions: remove, warn, suspend, ban | `[S§12.2]` | §12.1 (moderation actions) |
| Appeals mechanism reviewed by different moderator | `[S§12.2]` | §12.4 (appeals process) |
| Reportable content types: posts, comments, messages, profiles, groups, events, listings, reviews, questions, quizzes, sequences | `[S§12.3]` | §3.2 (`safety_reports.target_type` CHECK constraint) |
| Report categories: inappropriate, harassment, spam, misinformation, CSAM/child safety, other | `[S§12.3]` | §3.2 (`safety_reports.category` CHECK constraint) |
| Report triage by priority (child safety < 24h, other < 72h) | `[S§12.3]` | §11.4 (priority queue), §3.2 (`priority` column) |
| Reporter acknowledgment and outcome notification | `[S§12.3]` | §11.5 (reporter notification via `notify::`) |
| CAPTCHA or equivalent bot detection on registration | `[S§12.4]` | §13 (delegated to `iam::`) |
| Behavioral bot detection (rapid posting, mass friend requests) | `[S§12.4]` | §13.2 (behavioral signals) |
| Rate limiting on user actions | `[S§12.4]` | §13 (delegated to `00-core` middleware) |
| Bot account flagging and automatic suspension | `[S§12.4]` | §13.2 (bot signal → auto-suspend) |
| Community guidelines prohibiting methodology hostility | `[S§12.5]` | §11.1 (keyword lists include methodology hostility terms) |
| Methodology hostility as distinct report category | `[S§12.5]` | §3.2 (`category` CHECK includes `methodology_hostility`) |
| Grooming behavior monitoring | `[S§12.6]` | §14.2 (Phase 2 — ML-based grooming detection) |
| Age-appropriate content filtering for students | `[S§12.6]` | §14.1 (content filtering for student-accessible surfaces) |
| Adults cannot initiate direct contact with student profiles | `[S§12.6]` | §14.1 (enforced by `social::` — safety:: monitors violations) |
| Parental controls for content exposure | `[S§12.6]` | §14.3 (Phase 2 — parental override settings) |
| Moderation dashboard with content review queue | `[S§12.7]` | §4.2 (admin endpoints), §11.4 (queue) |
| Audit-logged moderation actions | `[S§12.7]` | §3.2 (`safety_mod_actions` — immutable), §12.1 |
| Group-level moderation roles | `[S§12.7]` | §14.4 (delegated to `social::` group admin system) |
| Platform Administrator elevated access | `[S§3.1.5]` | §9 (`RequireAdmin` extractor) |
| Administrator actions audit-logged | `[S§3.1.5]` | §3.2 (`safety_mod_actions.admin_id`) |
| Admin role-based sub-permissions (moderation-only, billing-support-only) | `[S§3.1.5]` | §9.3 (Phase 2 — granular admin roles) |

> **Coverage note on `[S§12.7]` (group-level moderation)**: Group moderators are managed by
> `social::` (group admin adds/removes moderators within their group). `safety::` handles
> platform-level moderation only. Group moderators can remove content and mute users within
> their group; platform moderators can take cross-group actions including account suspension.
> This split is consistent with `[05-social §14]`.

---

## §3 Database Schema

The safety domain stores reports, content flags, moderation actions (immutable audit trail),
account moderation status, appeals, NCMEC report records, and bot detection signals. Reports are
family-scoped (the reporter's family); moderation actions and account status are admin-scoped
(not family-scoped — they cross family boundaries). `[ARCH §5.1, ARCH §5.2]`

> **Refinement note**: ARCHITECTURE.md §5.1 sketches `safety_reports`, `safety_mod_actions`,
> and `safety_content_flags`. This spec adds: (1) `safety_account_status` for account-level
> moderation state, (2) `safety_appeals` for the appeals workflow, (3) `safety_ncmec_reports`
> for NCMEC filing audit trail, (4) `safety_bot_signals` for behavioral bot detection,
> (5) comprehensive CHECK constraints for all status/type/category columns.

### §3.1 Enums

Implemented as `CHECK` constraints (not PostgreSQL ENUM types) per `[CODING §4.1]`:

```sql
-- Report status values: pending, in_review, resolved_action_taken, resolved_no_action, dismissed
-- Report category values: inappropriate_content, harassment, spam, misinformation,
--                         csam_child_safety, methodology_hostility, other
-- Report priority values: critical, high, normal
-- Target type values: post, comment, message, profile, group, event, listing, review,
--                     question, quiz, sequence
-- Mod action type values: content_removed, warning_issued, account_suspended,
--                         account_banned, content_restored, suspension_lifted,
--                         appeal_granted, escalate_to_csam
-- Account status values: active, suspended, banned
-- Appeal status values: pending, in_review, granted, denied
-- NCMEC report status values: pending, submitted, confirmed, failed
-- Bot signal type values: rapid_posting, mass_friend_requests, repetitive_content,
--                         suspicious_registration, rate_limit_exceeded
```

### §3.2 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_safety_tables.sql (goose)
-- =============================================================================

-- TABLE 1: safety_reports — User-submitted content reports [S§12.3]
CREATE TABLE safety_reports (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_family_id    UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    reporter_parent_id    UUID NOT NULL REFERENCES iam_parents(id),

    -- What is being reported
    target_type           TEXT NOT NULL
                          CHECK (target_type IN (
                              'post', 'comment', 'message', 'profile',
                              'group', 'event', 'listing', 'review',
                              'question', 'quiz', 'sequence'
                          )),
    target_id             UUID NOT NULL,           -- ID of the reported entity
    target_family_id      UUID,                    -- family that owns the reported content (NULL for groups/events)

    -- Report details
    category              TEXT NOT NULL
                          CHECK (category IN (
                              'inappropriate_content', 'harassment', 'spam',
                              'misinformation', 'csam_child_safety',
                              'methodology_hostility', 'other'
                          )),
    description           TEXT,                    -- optional free-text from reporter [S§12.3]
    priority              TEXT NOT NULL DEFAULT 'normal'
                          CHECK (priority IN ('critical', 'high', 'normal')),

    -- Resolution
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN (
                              'pending', 'in_review', 'resolved_action_taken',
                              'resolved_no_action', 'dismissed'
                          )),
    assigned_admin_id     UUID REFERENCES iam_parents(id),
    resolved_at           TIMESTAMPTZ,

    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Reporter's reports (family-scoped reads)
CREATE INDEX idx_safety_reports_reporter_family
    ON safety_reports(reporter_family_id, created_at DESC);

-- Admin queue: pending reports ordered by priority then age
CREATE INDEX idx_safety_reports_pending_priority
    ON safety_reports(priority, created_at)
    WHERE status IN ('pending', 'in_review');

-- Target lookup: find all reports against a specific entity
CREATE INDEX idx_safety_reports_target
    ON safety_reports(target_type, target_id);

-- Assigned admin's queue
CREATE INDEX idx_safety_reports_assigned
    ON safety_reports(assigned_admin_id)
    WHERE assigned_admin_id IS NOT NULL AND status = 'in_review';

-- TABLE 2: safety_content_flags — Automated content screening results [S§12.2]
CREATE TABLE safety_content_flags (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source                TEXT NOT NULL
                          CHECK (source IN ('automated', 'community_report')),

    -- What was flagged
    target_type           TEXT NOT NULL
                          CHECK (target_type IN (
                              'post', 'comment', 'message', 'profile',
                              'group', 'event', 'listing', 'review', 'upload',
                              'question', 'quiz', 'sequence'
                          )),
    target_id             UUID NOT NULL,
    target_family_id      UUID REFERENCES iam_families(id) ON DELETE CASCADE,

    -- Flag details
    flag_type             TEXT NOT NULL
                          CHECK (flag_type IN (
                              'csam', 'explicit_content', 'violence', 'spam',
                              'harassment', 'prohibited_content', 'text_violation',
                              'suspected_underage_exploitation'
                          )),
    confidence            DOUBLE PRECISION,        -- 0.0-1.0 for automated flags
    labels                JSONB,                   -- moderation labels from Rekognition or text scanner
    report_id             UUID REFERENCES safety_reports(id), -- NULL for automated, set for community reports
    auto_rejected         BOOLEAN NOT NULL DEFAULT false,    -- true when upload was auto-rejected (§11.2.1)

    -- Resolution
    reviewed              BOOLEAN NOT NULL DEFAULT false,
    reviewed_by           UUID REFERENCES iam_parents(id),
    reviewed_at           TIMESTAMPTZ,
    action_taken          BOOLEAN,                 -- NULL until reviewed

    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unreviewed flags queue (admin)
CREATE INDEX idx_safety_content_flags_unreviewed
    ON safety_content_flags(created_at)
    WHERE reviewed = false;

-- Target lookup: find all flags for a specific entity
CREATE INDEX idx_safety_content_flags_target
    ON safety_content_flags(target_type, target_id);

-- TABLE 3: safety_mod_actions — Immutable moderation action audit trail [S§12.7]
-- This table is APPEND-ONLY. Rows are never updated or deleted.
CREATE TABLE safety_mod_actions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id              UUID NOT NULL REFERENCES iam_parents(id),
    target_family_id      UUID NOT NULL REFERENCES iam_families(id),
    target_parent_id      UUID REFERENCES iam_parents(id),     -- NULL for family-level actions

    -- Action details
    action_type           TEXT NOT NULL
                          CHECK (action_type IN (
                              'content_removed', 'warning_issued', 'account_suspended',
                              'account_banned', 'content_restored', 'suspension_lifted',
                              'appeal_granted', 'escalate_to_csam'
                          )),
    reason                TEXT NOT NULL,            -- admin's rationale [S§12.7]
    report_id             UUID REFERENCES safety_reports(id),  -- originating report (if any)
    content_snapshot      JSONB,                   -- snapshot of removed content for audit

    -- Suspension-specific fields
    suspension_days       INTEGER,                 -- NULL for non-suspension actions
    suspension_expires_at TIMESTAMPTZ,             -- NULL for permanent bans

    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
    -- No updated_at — this table is immutable
);

-- Admin activity log
CREATE INDEX idx_safety_mod_actions_admin
    ON safety_mod_actions(admin_id, created_at DESC);

-- Target family action history
CREATE INDEX idx_safety_mod_actions_target_family
    ON safety_mod_actions(target_family_id, created_at DESC);

-- TABLE 4: safety_account_status — Current account moderation state [S§12.2]
CREATE TABLE safety_account_status (
    family_id             UUID PRIMARY KEY REFERENCES iam_families(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'active'
                          CHECK (status IN ('active', 'suspended', 'banned')),
    suspended_at          TIMESTAMPTZ,
    suspension_expires_at TIMESTAMPTZ,             -- NULL for bans (permanent)
    suspension_reason     TEXT,
    banned_at             TIMESTAMPTZ,
    ban_reason            TEXT,
    last_action_id        UUID REFERENCES safety_mod_actions(id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Quick lookup of suspended/banned accounts (for auth middleware cache warming)
CREATE INDEX idx_safety_account_status_restricted
    ON safety_account_status(status)
    WHERE status IN ('suspended', 'banned');

-- TABLE 5: safety_appeals — Moderation action appeals [S§12.2]
CREATE TABLE safety_appeals (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    action_id             UUID NOT NULL REFERENCES safety_mod_actions(id),
    appeal_text           TEXT NOT NULL,
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'in_review', 'granted', 'denied')),
    assigned_admin_id     UUID REFERENCES iam_parents(id),  -- MUST differ from original action admin [S§12.2]
    resolution_text       TEXT,
    resolved_at           TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Enforce: one appeal per moderation action
    CONSTRAINT uq_safety_appeals_action UNIQUE (action_id)
);

-- Pending appeals queue (admin)
CREATE INDEX idx_safety_appeals_pending
    ON safety_appeals(created_at)
    WHERE status IN ('pending', 'in_review');

-- Family's appeals
CREATE INDEX idx_safety_appeals_family
    ON safety_appeals(family_id);

-- TABLE 6: safety_ncmec_reports — NCMEC filing audit trail [S§12.1, 18 U.S.C. § 2258A]
-- This table is APPEND-ONLY. Rows are never updated or deleted (legal evidence).
CREATE TABLE safety_ncmec_reports (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    upload_id             UUID NOT NULL,            -- the quarantined upload (NOT FK — upload may be preserved separately)
    family_id             UUID NOT NULL,            -- the offending family (NOT FK — family may be deleted but report persists)
    parent_id             UUID NOT NULL,            -- the offending user (NOT FK — same reason)

    -- Thorn Safer / PhotoDNA results
    csam_hash             TEXT,                     -- PhotoDNA hash
    confidence            DOUBLE PRECISION,
    matched_database      TEXT,

    -- NCMEC filing
    ncmec_report_id       TEXT,                     -- NCMEC-assigned report ID after submission
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'submitted', 'confirmed', 'failed')),
    submitted_at          TIMESTAMPTZ,
    error_message         TEXT,                     -- internal only, NEVER exposed [CODING §5.2]

    -- Evidence preservation
    evidence_storage_key  TEXT NOT NULL,            -- S3 key where evidence is preserved (separate from normal uploads)

    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
    -- No updated_at — this table is append-only for legal compliance
);

-- Find pending NCMEC reports for retry
CREATE INDEX idx_safety_ncmec_pending
    ON safety_ncmec_reports(status)
    WHERE status IN ('pending', 'failed');

-- TABLE 7: safety_bot_signals — Behavioral bot detection signals [S§12.4]
CREATE TABLE safety_bot_signals (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    parent_id             UUID NOT NULL REFERENCES iam_parents(id),
    signal_type           TEXT NOT NULL
                          CHECK (signal_type IN (
                              'rapid_posting', 'mass_friend_requests',
                              'repetitive_content', 'suspicious_registration',
                              'rate_limit_exceeded'
                          )),
    details               JSONB NOT NULL DEFAULT '{}',  -- signal-specific metadata
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Recent signals for a parent (for threshold checking)
CREATE INDEX idx_safety_bot_signals_parent_recent
    ON safety_bot_signals(parent_id, created_at DESC);

-- Aggregate signals per family (for auto-suspend threshold)
CREATE INDEX idx_safety_bot_signals_family
    ON safety_bot_signals(family_id, created_at DESC);
```

### §3.3 RLS / Family-Scoping

Application-layer enforcement via `FamilyScope` extractor `[CODING §2.4, §2.5, 00-core §8]`:

```sql
-- Family-scoped tables:
-- safety_reports: scoped via reporter_family_id (reporters see their own reports)
-- safety_appeals: scoped via family_id (families see their own appeals)
-- safety_bot_signals: scoped via family_id
--
-- Admin-scoped tables (NOT family-scoped — admins access cross-family):
-- safety_content_flags: admin-only reads (cross-family by design)
-- safety_mod_actions: admin-only writes, immutable audit trail
-- safety_account_status: admin writes, auth middleware reads (cross-family)
-- safety_ncmec_reports: admin-only, cross-family, legal evidence
--
-- Repository methods for family-scoped tables accept FamilyScope as first parameter.
-- Repository methods for admin-scoped tables accept AdminContext (§9) and
-- document why unscoped access is required per [CODING §2.4].
```

---

## §4 API Endpoints

**Prefix**: `/v1/safety` (user-facing), `/v1/admin/safety` (admin-facing)

**Auth**: User endpoints require `AuthContext` (`FamilyScope`). Admin endpoints require
`RequireAdmin` (§9). `[00-core §7.2, §8]`

### §4.1 Phase 1 — User Endpoints (6 endpoints)

| # | Method | Path | Auth | Description | Status Codes |
|---|--------|------|------|-------------|-------------|
| 1 | `POST` | `/v1/safety/reports` | Required | Submit a content report | 201, 400, 401, 422 |
| 2 | `GET` | `/v1/safety/reports` | Required | List my submitted reports | 200, 401 |
| 3 | `GET` | `/v1/safety/reports/:id` | Required | Get report status | 200, 401, 404 |
| 4 | `GET` | `/v1/safety/account-status` | Required | Get my account moderation status | 200, 401 |
| 5 | `POST` | `/v1/safety/appeals` | Required | Submit an appeal | 201, 401, 404, 409, 422 |
| 6 | `GET` | `/v1/safety/appeals/:id` | Required | Get appeal status | 200, 401, 404 |

### §4.2 Phase 1 — Admin Endpoints (15 endpoints)

| # | Method | Path | Auth | Description | Status Codes |
|---|--------|------|------|-------------|-------------|
| A1 | `GET` | `/v1/admin/safety/reports` | Admin | List reports (filterable queue) | 200, 401, 403 |
| A2 | `GET` | `/v1/admin/safety/reports/:id` | Admin | Get report detail | 200, 401, 403, 404 |
| A3 | `PATCH` | `/v1/admin/safety/reports/:id` | Admin | Update report (assign, change status) | 200, 401, 403, 404, 422 |
| A4 | `GET` | `/v1/admin/safety/flags` | Admin | List content flags (review queue) | 200, 401, 403 |
| A5 | `PATCH` | `/v1/admin/safety/flags/:id` | Admin | Review flag (mark reviewed, action) | 200, 401, 403, 404 |
| A6 | `POST` | `/v1/admin/safety/actions` | Admin | Take moderation action | 201, 401, 403, 422 |
| A7 | `GET` | `/v1/admin/safety/actions` | Admin | List moderation actions (audit log) | 200, 401, 403 |
| A8 | `GET` | `/v1/admin/safety/accounts/:family_id` | Admin | Get account moderation state | 200, 401, 403, 404 |
| A9 | `POST` | `/v1/admin/safety/accounts/:family_id/suspend` | Admin | Suspend account | 200, 401, 403, 404, 422 |
| A10 | `POST` | `/v1/admin/safety/accounts/:family_id/ban` | Admin | Ban account | 200, 401, 403, 404 |
| A11 | `POST` | `/v1/admin/safety/accounts/:family_id/lift` | Admin | Lift suspension | 200, 401, 403, 404 |
| A12 | `GET` | `/v1/admin/safety/appeals` | Admin | List appeals (filterable queue) | 200, 401, 403 |
| A13 | `PATCH` | `/v1/admin/safety/appeals/:id` | Admin | Resolve appeal (grant/deny) | 200, 401, 403, 404, 422 |
| A14 | `GET` | `/v1/admin/safety/dashboard` | Admin | Dashboard stats (counts, trends) | 200, 401, 403 |
| A15 | `PATCH` | `/v1/admin/safety/flags/:id/escalate-csam` | Admin | Escalate flagged content to CSAM — triggers full §10 pipeline | 200, 401, 403, 404, 422 |

### §4.3 User Endpoint Details

#### POST /v1/safety/reports — Submit Report

Submit a content report against any reportable entity. `[S§12.3]`

- **Auth**: `AuthContext` + `FamilyScope`
- **Body**: `CreateReportCommand`
```json
{
    "target_type": "post",
    "target_id": "uuid",
    "category": "harassment",
    "description": "This post contains..."
}
```
- **Response**: `201 Created` → `ReportResponse`
```json
{
    "id": "uuid",
    "target_type": "post",
    "category": "harassment",
    "status": "pending",
    "created_at": "2026-03-21T..."
}
```
- **Validation**:
  - `target_type` must be in allowed types
  - `target_id` must reference an existing entity (validated via cross-domain lookup)
  - `category` must be in allowed categories
  - `description` max 2000 characters
  - Duplicate prevention: one report per (reporter, target_type, target_id) pair per 24 hours
- **Side effects**:
  - Creates `safety_reports` record with status `pending`
  - Sets `priority` to `critical` if category is `csam_child_safety`, `high` if `harassment`, `normal` otherwise
  - Creates `safety_content_flags` record with `source = 'community_report'`
  - Publishes `ContentReported` event
- **Error codes**:
  - `422` — invalid target_type or category (`ValidationError`)
  - `409` — duplicate report within 24h (`DuplicateReport`)

#### GET /v1/safety/reports — List My Reports

List the authenticated user's submitted reports with pagination.

- **Auth**: `AuthContext` + `FamilyScope`
- **Query**: `?cursor=...&limit=20`
- **Response**: `200 OK` → `PaginatedResponse<ReportResponse>`
- **Scoping**: `WHERE reporter_family_id = $family_scope.family_id`

#### GET /v1/safety/reports/:id — Get Report Status

Get the current status of a submitted report. Returns status and outcome (if resolved)
but does NOT reveal actions taken against the reported user. `[S§12.3]`

- **Auth**: `AuthContext` + `FamilyScope`
- **Response**: `200 OK` → `ReportResponse`
- **Scoping**: Report must belong to the authenticated user's family

#### GET /v1/safety/account-status — Get My Account Status

Return the authenticated family's current moderation status (active, suspended, banned).

- **Auth**: `AuthContext` + `FamilyScope`
- **Response**: `200 OK` → `AccountStatusResponse`
```json
{
    "status": "active",
    "suspended_at": null,
    "suspension_expires_at": null,
    "suspension_reason": null
}
```

#### POST /v1/safety/appeals — Submit Appeal

Submit an appeal against a moderation action. `[S§12.2]`

- **Auth**: `AuthContext` + `FamilyScope`
- **Body**: `CreateAppealCommand`
```json
{
    "action_id": "uuid",
    "appeal_text": "I believe this was..."
}
```
- **Response**: `201 Created` → `AppealResponse`
- **Validation**:
  - `action_id` must reference a mod action against the caller's family
  - `appeal_text` max 5000 characters
  - Only one appeal per action (enforced by unique constraint)
  - Cannot appeal a CSAM ban (action_type `account_banned` with CSAM reason)
- **Error codes**:
  - `404` — action not found or not against caller's family
  - `409` — appeal already exists for this action
  - `422` — CSAM bans are not appealable

#### GET /v1/safety/appeals/:id — Get Appeal Status

- **Auth**: `AuthContext` + `FamilyScope`
- **Response**: `200 OK` → `AppealResponse`
- **Scoping**: Appeal must belong to the authenticated user's family

### §4.4 Admin Endpoint Details

#### POST /v1/admin/safety/actions — Take Moderation Action

The core admin action endpoint. Creates an immutable mod action record and updates account
status as needed. `[S§12.2, S§12.7]`

- **Auth**: `RequireAdmin`
- **Body**: `CreateModActionCommand`
```json
{
    "target_family_id": "uuid",
    "target_parent_id": "uuid",
    "action_type": "account_suspended",
    "reason": "Repeated harassment after warning",
    "report_id": "uuid",
    "suspension_days": 7
}
```
- **Response**: `201 Created` → `ModActionResponse`
- **Side effects**:
  - Creates immutable `safety_mod_actions` record
  - Updates `safety_account_status` (§12)
  - For suspensions/bans: invalidates Redis cache, publishes `AccountSuspended`/`AccountBanned` event
  - For bans: triggers session revocation via `iam::KratosAdapter::revoke_sessions()`
  - Resolves associated report (if `report_id` provided)
  - Publishes `ModerationActionTaken` event → `notify::` (except for CSAM cases)

#### POST /v1/admin/safety/accounts/:family_id/suspend — Suspend Account

Convenience endpoint for account suspension (wraps `POST /actions`).

- **Auth**: `RequireAdmin`
- **Body**: `SuspendAccountCommand`
```json
{
    "reason": "Repeated policy violations",
    "suspension_days": 7,
    "report_id": "uuid"
}
```

#### POST /v1/admin/safety/accounts/:family_id/ban — Ban Account

Permanent ban. Irreversible (except via appeal). `[S§12.2]`

- **Auth**: `RequireAdmin`
- **Body**: `BanAccountCommand`
```json
{
    "reason": "Severe policy violation",
    "report_id": "uuid"
}
```

#### PATCH /v1/admin/safety/appeals/:id — Resolve Appeal

Resolve an appeal. The assigned admin MUST be different from the admin who took the
original action. `[S§12.2]`

- **Auth**: `RequireAdmin`
- **Body**: `ResolveAppealCommand`
```json
{
    "status": "granted",
    "resolution_text": "Upon review, the suspension is lifted..."
}
```
- **Side effects**:
  - If granted and action was suspension: lifts suspension, creates `suspension_lifted` mod action
  - Publishes `AppealResolved` event → `notify::` (notifies the family)

---

## §5 Service Interface

The `SafetyService` interface defines all use cases exposed to handlers, other domains, and background
jobs. Defined in `internal/safety/ports.go`. `[CODING §8.2]`

```go
// internal/safety/ports.go

// SafetyService defines all safety and moderation use cases.
type SafetyService interface {

    // ─── User-Facing Queries ────────────────────────────────────────────

    // ListMyReports lists the caller's submitted reports.
    ListMyReports(ctx context.Context, scope *shared.FamilyScope, pagination *shared.PaginationParams) (*shared.PaginatedResponse[ReportResponse], error)

    // GetMyReport gets a specific report (must belong to caller's family).
    GetMyReport(ctx context.Context, scope *shared.FamilyScope, reportID uuid.UUID) (*ReportResponse, error)

    // GetAccountStatus gets the caller's account moderation status.
    GetAccountStatus(ctx context.Context, scope *shared.FamilyScope) (*AccountStatusResponse, error)

    // GetMyAppeal gets a specific appeal (must belong to caller's family).
    GetMyAppeal(ctx context.Context, scope *shared.FamilyScope, appealID uuid.UUID) (*AppealResponse, error)

    // ─── User-Facing Commands ───────────────────────────────────────────

    // SubmitReport submits a content report.
    SubmitReport(ctx context.Context, scope *shared.FamilyScope, auth *shared.AuthContext, cmd CreateReportCommand) (*ReportResponse, error)

    // SubmitAppeal submits an appeal against a moderation action.
    SubmitAppeal(ctx context.Context, scope *shared.FamilyScope, cmd CreateAppealCommand) (*AppealResponse, error)

    // ─── Admin Queries ──────────────────────────────────────────────────

    // AdminListReports lists reports (admin queue with filters).
    AdminListReports(ctx context.Context, auth *shared.AuthContext, filter ReportFilter, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AdminReportResponse], error)

    // AdminGetReport gets report detail (admin view — includes target content snapshot).
    AdminGetReport(ctx context.Context, auth *shared.AuthContext, reportID uuid.UUID) (*AdminReportResponse, error)

    // AdminListFlags lists content flags (admin review queue).
    AdminListFlags(ctx context.Context, auth *shared.AuthContext, filter FlagFilter, pagination *shared.PaginationParams) (*shared.PaginatedResponse[ContentFlagResponse], error)

    // AdminListActions lists moderation actions (audit log).
    AdminListActions(ctx context.Context, auth *shared.AuthContext, filter ActionFilter, pagination *shared.PaginationParams) (*shared.PaginatedResponse[ModActionResponse], error)

    // AdminGetAccount gets account moderation state (admin view).
    AdminGetAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID) (*AdminAccountStatusResponse, error)

    // AdminListAppeals lists appeals (admin queue).
    AdminListAppeals(ctx context.Context, auth *shared.AuthContext, filter AppealFilter, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AdminAppealResponse], error)

    // AdminDashboard returns dashboard statistics.
    AdminDashboard(ctx context.Context, auth *shared.AuthContext) (*DashboardStats, error)

    // ─── Admin Commands ─────────────────────────────────────────────────

    // AdminUpdateReport updates a report (assign admin, change status).
    AdminUpdateReport(ctx context.Context, auth *shared.AuthContext, reportID uuid.UUID, cmd UpdateReportCommand) (*AdminReportResponse, error)

    // AdminReviewFlag reviews a content flag.
    AdminReviewFlag(ctx context.Context, auth *shared.AuthContext, flagID uuid.UUID, cmd ReviewFlagCommand) (*ContentFlagResponse, error)

    // AdminTakeAction takes a moderation action.
    AdminTakeAction(ctx context.Context, auth *shared.AuthContext, cmd CreateModActionCommand) (*ModActionResponse, error)

    // AdminSuspendAccount suspends an account.
    AdminSuspendAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd SuspendAccountCommand) (*AdminAccountStatusResponse, error)

    // AdminBanAccount bans an account.
    AdminBanAccount(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd BanAccountCommand) (*AdminAccountStatusResponse, error)

    // AdminLiftSuspension lifts a suspension.
    AdminLiftSuspension(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, cmd LiftSuspensionCommand) (*AdminAccountStatusResponse, error)

    // AdminResolveAppeal resolves an appeal.
    AdminResolveAppeal(ctx context.Context, auth *shared.AuthContext, appealID uuid.UUID, cmd ResolveAppealCommand) (*AdminAppealResponse, error)

    // ─── Internal (Cross-Domain) Methods ────────────────────────────────

    // CheckAccountAccess checks whether a family's account is allowed to access the platform.
    // Called by auth middleware on every authenticated request.
    // Returns nil if active, ErrAccountSuspended/ErrAccountBanned otherwise.
    // Uses Redis cache with 60s TTL for performance.
    CheckAccountAccess(ctx context.Context, familyID uuid.UUID) error

    // ScanText scans text content for policy violations.
    // Called synchronously by social, mkt, and learn before persisting content.
    // Phase 1: keyword + regex matching. Phase 2: ML (AWS Comprehend).
    ScanText(ctx context.Context, text string) (*TextScanResult, error)

    // RecordBotSignal records a bot behavioral signal.
    // Called by domains that detect suspicious patterns.
    RecordBotSignal(ctx context.Context, familyID uuid.UUID, parentID uuid.UUID, signal BotSignalType, details json.RawMessage) error

    // HandleCsamDetection processes a CSAM detection from the media pipeline.
    // Quarantines upload, files NCMEC report, bans account.
    // Called by SafetyScanAdapter.ReportCsam() implementation.
    HandleCsamDetection(ctx context.Context, uploadID uuid.UUID, familyID uuid.UUID, scanResult *CsamScanResult) error

    // AdminEscalateToCsam escalates flagged/rejected content to CSAM (admin action). [§11.4.1]
    // Marks the flag as reviewed, then delegates to HandleCsamDetection()
    // for the full §10 pipeline (evidence → NCMEC → ban → session revoke).
    AdminEscalateToCsam(ctx context.Context, auth *shared.AuthContext, flagID uuid.UUID, cmd EscalateCsamCommand) error
}
```

### §5.1 Service Implementation

```go
// internal/safety/service.go

type SafetyServiceImpl struct {
    reportRepo     ReportRepository
    flagRepo       ContentFlagRepository
    actionRepo     ModActionRepository
    accountRepo    AccountStatusRepository
    appealRepo     AppealRepository
    ncmecRepo      NcmecReportRepository
    botSignalRepo  BotSignalRepository
    thorn          ThornAdapter
    rekognition    RekognitionAdapter
    iamService     iam.IamService
    redis          *redis.Client
    events         *shared.EventBus
    config         SafetyConfig
}
```

---

## §6 Repository Interfaces

### §6.1 ReportRepository

```go
// internal/safety/ports.go

type ReportRepository interface {
    Create(ctx context.Context, scope *shared.FamilyScope, input CreateReportRow) (*Report, error)

    FindByID(ctx context.Context, scope *shared.FamilyScope, reportID uuid.UUID) (*Report, error)

    // FindByIDUnscoped — admin-only access [CODING §2.4].
    FindByIDUnscoped(ctx context.Context, reportID uuid.UUID) (*Report, error)

    ListByReporter(ctx context.Context, scope *shared.FamilyScope, pagination *shared.PaginationParams) (*shared.PaginatedResult[Report], error)

    // ListFiltered — admin: list with filters (status, priority, category, assigned).
    ListFiltered(ctx context.Context, filter *ReportFilter, pagination *shared.PaginationParams) (*shared.PaginatedResult[Report], error)

    Update(ctx context.Context, reportID uuid.UUID, updates ReportUpdate) (*Report, error)

    // ExistsRecent checks for duplicate reports within time window.
    ExistsRecent(ctx context.Context, scope *shared.FamilyScope, targetType string, targetID uuid.UUID, withinHours uint32) (bool, error)
}
```

### §6.2 ContentFlagRepository

```go
type ContentFlagRepository interface {
    Create(ctx context.Context, input CreateContentFlagRow) (*ContentFlag, error)

    FindByID(ctx context.Context, flagID uuid.UUID) (*ContentFlag, error)

    // ListUnreviewed lists unreviewed flags (admin queue).
    ListUnreviewed(ctx context.Context, filter *FlagFilter, pagination *shared.PaginationParams) (*shared.PaginatedResult[ContentFlag], error)

    MarkReviewed(ctx context.Context, flagID uuid.UUID, reviewedBy uuid.UUID, actionTaken bool) (*ContentFlag, error)

    // FindByTarget finds all flags for a target entity.
    FindByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]ContentFlag, error)
}
```

### §6.3 ModActionRepository

```go
// ModActionRepository is an immutable audit trail — only insert and read operations.
type ModActionRepository interface {
    // Create inserts a new moderation action. Never updates or deletes.
    Create(ctx context.Context, input CreateModActionRow) (*ModAction, error)

    FindByID(ctx context.Context, actionID uuid.UUID) (*ModAction, error)

    // ListFiltered lists actions with filters (admin, target_family, action_type).
    ListFiltered(ctx context.Context, filter *ActionFilter, pagination *shared.PaginationParams) (*shared.PaginatedResult[ModAction], error)

    // ListByTargetFamily lists all actions against a specific family.
    ListByTargetFamily(ctx context.Context, familyID uuid.UUID, pagination *shared.PaginationParams) (*shared.PaginatedResult[ModAction], error)
}
```

### §6.4 AccountStatusRepository

```go
type AccountStatusRepository interface {
    // GetOrCreate gets or creates account status (default: active).
    GetOrCreate(ctx context.Context, familyID uuid.UUID) (*AccountStatus, error)

    Update(ctx context.Context, familyID uuid.UUID, updates AccountStatusUpdate) (*AccountStatus, error)

    // FindRestricted finds all restricted accounts (for cache warming).
    FindRestricted(ctx context.Context) ([]AccountStatus, error)
}
```

### §6.5 AppealRepository

```go
type AppealRepository interface {
    Create(ctx context.Context, scope *shared.FamilyScope, input CreateAppealRow) (*Appeal, error)

    FindByID(ctx context.Context, scope *shared.FamilyScope, appealID uuid.UUID) (*Appeal, error)

    FindByIDUnscoped(ctx context.Context, appealID uuid.UUID) (*Appeal, error)

    ListPending(ctx context.Context, filter *AppealFilter, pagination *shared.PaginationParams) (*shared.PaginatedResult[Appeal], error)

    Update(ctx context.Context, appealID uuid.UUID, updates AppealUpdate) (*Appeal, error)
}
```

### §6.6 NcmecReportRepository

```go
// NcmecReportRepository is append-only — legal evidence. Only insert and read.
type NcmecReportRepository interface {
    Create(ctx context.Context, input CreateNcmecReportRow) (*NcmecReport, error)

    UpdateStatus(ctx context.Context, reportID uuid.UUID, status string, ncmecReportID *string, errMsg *string) (*NcmecReport, error)

    FindPending(ctx context.Context) ([]NcmecReport, error)
}
```

### §6.7 BotSignalRepository

```go
type BotSignalRepository interface {
    Create(ctx context.Context, input CreateBotSignalRow) (*BotSignal, error)

    // CountRecent counts signals for a parent within a time window.
    CountRecent(ctx context.Context, parentID uuid.UUID, withinMinutes uint32) (int64, error)
}
```

---

## §7 Adapter Interfaces

### §7.1 ThornAdapter (CSAM Detection + NCMEC Reporting)

Wraps the Thorn Safer API for PhotoDNA hash matching and automated NCMEC CyberTipline
report submission. `[ARCH §2.13, S§12.1]`

Adapter file: `internal/safety/adapters/thorn.go`

```go
// internal/safety/adapters/thorn.go

// ThornAdapter wraps the Thorn Safer API for CSAM detection and NCMEC reporting.
//
// Thorn Safer provides:
// 1. PhotoDNA hash matching against NCMEC's hash database
// 2. Automated NCMEC CyberTipline report submission
//
// [ARCH §2.13, S§12.1, 18 U.S.C. § 2258A]
type ThornAdapter interface {
    // ScanCsam scans an image/video for CSAM using PhotoDNA hash matching.
    // Reads the file from S3 (via storageKey), computes PhotoDNA hash,
    // and checks against NCMEC's known CSAM hash database.
    ScanCsam(ctx context.Context, storageKey string) (*CsamScanResult, error)

    // SubmitNcmecReport submits a CyberTipline report to NCMEC.
    // Required by federal law (18 U.S.C. § 2258A) when CSAM is detected.
    // Includes: file hash, upload metadata, uploader info, evidence URL.
    SubmitNcmecReport(ctx context.Context, report NcmecReportPayload) (*NcmecSubmissionResult, error)
}

// ThornConfig holds configuration for Thorn Safer adapter.
type ThornConfig struct {
    APIUrl string // THORN_API_URL
    APIKey string // THORN_API_KEY (secret)
}

// Thorn adapter errors.
var (
    ErrThornUnavailable        = errors.New("thorn API unavailable")
    ErrThornAPI                = errors.New("thorn API error")
    ErrNcmecSubmissionFailed   = errors.New("NCMEC submission failed")
)
```

### §7.2 RekognitionAdapter (Content Moderation)

Wraps AWS Rekognition's `DetectModerationLabels` API for general content moderation
(explicit content, violence, etc. — NOT CSAM). `[ARCH §2.13, S§12.2]`

Adapter file: `internal/safety/adapters/rekognition.go`

```go
// internal/safety/adapters/rekognition.go

// RekognitionAdapter wraps AWS Rekognition's DetectModerationLabels API for content moderation.
//
// Thin wrapper around DetectModerationLabels. Returns ALL raw labels from
// Rekognition without filtering — including categories the platform ignores
// (drugs, hate symbols, weapons). Label routing (which labels trigger
// auto-reject vs. flag vs. ignore) is handled by SafetyScanBridge (§11.2.2).
//
// Rekognition returns labels for:
// - Explicit/suggestive content → routed by bridge (auto-reject or flag)
// - Violence/graphic content → routed by bridge (flag for review)
// - Drugs/tobacco/alcohol → ignored by bridge (educational content)
// - Hate symbols → ignored by bridge (educational content)
// - Weapons → ignored by bridge (educational content)
//
// Does NOT handle CSAM detection — that is ThornAdapter's responsibility.
// [ARCH §2.13, S§12.2]
type RekognitionAdapter interface {
    // DetectModerationLabels detects moderation labels in an image.
    // Returns labels with confidence scores. The caller decides the
    // threshold for flagging (configurable via SafetyConfig).
    DetectModerationLabels(ctx context.Context, storageKey string) (*ModerationResult, error)
}

// RekognitionConfig holds configuration for Rekognition adapter.
type RekognitionConfig struct {
    Region        string  // AWS_REGION
    MinConfidence float64 // minimum confidence threshold (default: 70.0)
}

// Rekognition adapter errors.
var (
    ErrRekognitionUnavailable = errors.New("rekognition unavailable")
    ErrRekognitionAPI         = errors.New("rekognition API error")
)
```

### §7.3 SafetyScanAdapter Implementation

`safety::` provides the `SafetyScanAdapter` implementation consumed by `media::` during
the `ProcessUploadJob` pipeline. This adapter bridges `media::`'s port with `safety::`'s
Thorn and Rekognition adapters. `[09-media §7.2]`

```go
// internal/safety/adapters/scan_bridge.go

// SafetyScanBridge implements media.SafetyScanAdapter.
// Bridges media's scan port with safety's Thorn and Rekognition adapters.
type SafetyScanBridge struct {
    thorn         ThornAdapter
    rekognition   RekognitionAdapter
    safetyService SafetyService
    config        SafetyConfig
}

// ScanCsam implements media.SafetyScanAdapter.
func (b *SafetyScanBridge) ScanCsam(ctx context.Context, storageKey string) (*CsamScanResult, error) {
    return b.thorn.ScanCsam(ctx, storageKey)
}

// ScanModeration implements media.SafetyScanAdapter.
func (b *SafetyScanBridge) ScanModeration(ctx context.Context, storageKey string) (*ModerationResult, error) {
    // 1. Get raw labels from Rekognition (all categories, unfiltered)
    rawResult, err := b.rekognition.DetectModerationLabels(ctx, storageKey)
    if err != nil {
        return nil, fmt.Errorf("rekognition scan failed: %w", err)
    }

    // 2. Apply platform's label routing table (§11.2.2)
    //    - Nudity/explicit → AutoReject = true
    //    - Suggestive/violence → HasViolations = true, priority set
    //    - Drugs/hate/weapons → discarded (ignored categories)
    //    - Underage + sexual → critical priority
    return applyLabelRouting(rawResult.Labels, &b.config), nil
}

// ReportCsam implements media.SafetyScanAdapter.
func (b *SafetyScanBridge) ReportCsam(ctx context.Context, uploadID uuid.UUID, scanResult *CsamScanResult) error {
    // Delegates to SafetyService.HandleCsamDetection()
    // which handles NCMEC filing, evidence preservation, and account ban.
    // familyID is looked up from the upload record.
    // This method is called from media.ProcessUploadJob when CSAM is detected.
    panic("not implemented: delegates to safetyService.HandleCsamDetection()")
}
```

---

## §8 Models (DTOs)

### §8.1 Request Types

```go
// internal/safety/models.go

type CreateReportCommand struct {
    TargetType  string     `json:"target_type" validate:"required"`
    TargetID    uuid.UUID  `json:"target_id" validate:"required"`
    Description *string    `json:"description,omitempty" validate:"omitempty,max=2000"`
    Category    string     `json:"category" validate:"required"`
}

type CreateAppealCommand struct {
    ActionID   uuid.UUID `json:"action_id" validate:"required"`
    AppealText string    `json:"appeal_text" validate:"required,min=10,max=5000"`
}

type CreateModActionCommand struct {
    TargetFamilyID uuid.UUID  `json:"target_family_id" validate:"required"`
    TargetParentID *uuid.UUID `json:"target_parent_id,omitempty"`
    ActionType     string     `json:"action_type" validate:"required"`
    Reason         string     `json:"reason" validate:"required,min=5,max=2000"`
    ReportID       *uuid.UUID `json:"report_id,omitempty"`
    SuspensionDays *int32     `json:"suspension_days,omitempty"`
}

type SuspendAccountCommand struct {
    Reason         string     `json:"reason" validate:"required,min=5,max=2000"`
    SuspensionDays int32      `json:"suspension_days" validate:"required,min=1,max=365"`
    ReportID       *uuid.UUID `json:"report_id,omitempty"`
}

type BanAccountCommand struct {
    Reason   string     `json:"reason" validate:"required,min=5,max=2000"`
    ReportID *uuid.UUID `json:"report_id,omitempty"`
}

type LiftSuspensionCommand struct {
    Reason string `json:"reason" validate:"required,min=5,max=2000"`
}

type UpdateReportCommand struct {
    AssignedAdminID *uuid.UUID `json:"assigned_admin_id,omitempty"`
    Status          *string    `json:"status,omitempty"`
}

type ReviewFlagCommand struct {
    ActionTaken bool `json:"action_taken"`
}

type ResolveAppealCommand struct {
    Status         string `json:"status" validate:"required"`         // "granted" or "denied"
    ResolutionText string `json:"resolution_text" validate:"required,min=5,max=2000"`
}

// EscalateCsamCommand requests escalation of flagged content to CSAM. [§11.4.1]
// Triggers the full §10 CSAM pipeline (evidence → NCMEC → ban → session revoke).
type EscalateCsamCommand struct {
    AdminNotes string `json:"admin_notes" validate:"required,min=5,max=2000"` // required justification for audit trail
}
```

### §8.2 Response Types

```go
type ReportResponse struct {
    ID         uuid.UUID `json:"id"`
    TargetType string    `json:"target_type"`
    Category   string    `json:"category"`
    Status     string    `json:"status"`
    CreatedAt  time.Time `json:"created_at"`
}

type AdminReportResponse struct {
    ID              uuid.UUID  `json:"id"`
    ReporterFamilyID uuid.UUID `json:"reporter_family_id"`
    TargetType      string     `json:"target_type"`
    TargetID        uuid.UUID  `json:"target_id"`
    TargetFamilyID  *uuid.UUID `json:"target_family_id,omitempty"`
    Category        string     `json:"category"`
    Description     *string    `json:"description,omitempty"`
    Priority        string     `json:"priority"`
    Status          string     `json:"status"`
    AssignedAdminID *uuid.UUID `json:"assigned_admin_id,omitempty"`
    ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
    CreatedAt       time.Time  `json:"created_at"`
}

type ContentFlagResponse struct {
    ID          uuid.UUID        `json:"id"`
    Source      string           `json:"source"`
    TargetType  string           `json:"target_type"`
    TargetID    uuid.UUID        `json:"target_id"`
    FlagType    string           `json:"flag_type"`
    Confidence  *float64         `json:"confidence,omitempty"`
    Labels      json.RawMessage  `json:"labels,omitempty"`
    Reviewed    bool             `json:"reviewed"`
    ReviewedBy  *uuid.UUID       `json:"reviewed_by,omitempty"`
    ActionTaken *bool            `json:"action_taken,omitempty"`
    CreatedAt   time.Time        `json:"created_at"`
}

type ModActionResponse struct {
    ID                 uuid.UUID  `json:"id"`
    AdminID            uuid.UUID  `json:"admin_id"`
    TargetFamilyID     uuid.UUID  `json:"target_family_id"`
    TargetParentID     *uuid.UUID `json:"target_parent_id,omitempty"`
    ActionType         string     `json:"action_type"`
    Reason             string     `json:"reason"`
    ReportID           *uuid.UUID `json:"report_id,omitempty"`
    SuspensionDays     *int32     `json:"suspension_days,omitempty"`
    SuspensionExpiresAt *time.Time `json:"suspension_expires_at,omitempty"`
    CreatedAt          time.Time  `json:"created_at"`
}

type AccountStatusResponse struct {
    Status              string     `json:"status"`
    SuspendedAt         *time.Time `json:"suspended_at,omitempty"`
    SuspensionExpiresAt *time.Time `json:"suspension_expires_at,omitempty"`
    SuspensionReason    *string    `json:"suspension_reason,omitempty"`
}

type AdminAccountStatusResponse struct {
    FamilyID            uuid.UUID           `json:"family_id"`
    Status              string              `json:"status"`
    SuspendedAt         *time.Time          `json:"suspended_at,omitempty"`
    SuspensionExpiresAt *time.Time          `json:"suspension_expires_at,omitempty"`
    SuspensionReason    *string             `json:"suspension_reason,omitempty"`
    BannedAt            *time.Time          `json:"banned_at,omitempty"`
    BanReason           *string             `json:"ban_reason,omitempty"`
    ActionHistory       []ModActionResponse `json:"action_history"`
}

type AppealResponse struct {
    ID             uuid.UUID  `json:"id"`
    ActionID       uuid.UUID  `json:"action_id"`
    Status         string     `json:"status"`
    AppealText     string     `json:"appeal_text"`
    ResolutionText *string    `json:"resolution_text,omitempty"`
    ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
    CreatedAt      time.Time  `json:"created_at"`
}

type AdminAppealResponse struct {
    ID              uuid.UUID         `json:"id"`
    FamilyID        uuid.UUID         `json:"family_id"`
    ActionID        uuid.UUID         `json:"action_id"`
    OriginalAction  ModActionResponse `json:"original_action"`
    AppealText      string            `json:"appeal_text"`
    Status          string            `json:"status"`
    AssignedAdminID *uuid.UUID        `json:"assigned_admin_id,omitempty"`
    ResolutionText  *string           `json:"resolution_text,omitempty"`
    ResolvedAt      *time.Time        `json:"resolved_at,omitempty"`
    CreatedAt       time.Time         `json:"created_at"`
}

type DashboardStats struct {
    PendingReports    int64 `json:"pending_reports"`
    CriticalReports   int64 `json:"critical_reports"`
    UnreviewedFlags   int64 `json:"unreviewed_flags"`
    PendingAppeals    int64 `json:"pending_appeals"`
    ActiveSuspensions int64 `json:"active_suspensions"`
    ActiveBans        int64 `json:"active_bans"`
    ReportsLast24h    int64 `json:"reports_last_24h"`
    ActionsLast24h    int64 `json:"actions_last_24h"`
}

type TextScanResult struct {
    HasViolations bool     `json:"has_violations"`
    MatchedTerms  []string `json:"matched_terms"`
    Severity      string   `json:"severity"` // "none", "low", "high", "critical"
}
```

### §8.3 Filter Types

```go
type ReportFilter struct {
    Status          *string    `query:"status"`
    Priority        *string    `query:"priority"`
    Category        *string    `query:"category"`
    AssignedAdminID *uuid.UUID `query:"assigned_admin_id"`
}

type FlagFilter struct {
    Reviewed   *bool   `query:"reviewed"`
    FlagType   *string `query:"flag_type"`
    TargetType *string `query:"target_type"`
}

type ActionFilter struct {
    AdminID        *uuid.UUID `query:"admin_id"`
    TargetFamilyID *uuid.UUID `query:"target_family_id"`
    ActionType     *string    `query:"action_type"`
}

type AppealFilter struct {
    Status *string `query:"status"`
}
```

### §8.4 Internal / Adapter Types

```go
// CSAM scan result from Thorn Safer (shared with media package).
// Defined authoritatively in internal/media/models.go (09-media §8.3).
// Imported here: media.CsamScanResult, media.ModerationResult, media.ModerationLabel

// NcmecReportPayload is the payload for Thorn Safer NCMEC submission.
type NcmecReportPayload struct {
    UploadID           uuid.UUID `json:"upload_id"`
    CsamHash           *string   `json:"csam_hash,omitempty"`
    Confidence         *float64  `json:"confidence,omitempty"`
    MatchedDatabase    *string   `json:"matched_database,omitempty"`
    EvidenceStorageKey string    `json:"evidence_storage_key"`
    UploaderFamilyID   uuid.UUID `json:"uploader_family_id"`
    UploaderParentID   uuid.UUID `json:"uploader_parent_id"`
    UploadTimestamp    time.Time `json:"upload_timestamp"`
}

// NcmecSubmissionResult is the result from a Thorn Safer NCMEC submission.
type NcmecSubmissionResult struct {
    NcmecReportID string    `json:"ncmec_report_id"`
    SubmittedAt   time.Time `json:"submitted_at"`
}

// BotSignalType enumerates bot signal types.
type BotSignalType string

const (
    BotSignalRapidPosting          BotSignalType = "rapid_posting"
    BotSignalMassFriendRequests    BotSignalType = "mass_friend_requests"
    BotSignalRepetitiveContent     BotSignalType = "repetitive_content"
    BotSignalSuspiciousRegistration BotSignalType = "suspicious_registration"
    BotSignalRateLimitExceeded     BotSignalType = "rate_limit_exceeded"
)

// SafetyConfig holds safety domain configuration.
type SafetyConfig struct {
    RekognitionMinConfidence     float64  // default: 70.0
    NudityAutoRejectLabels       []string // labels that trigger auto-rejection (§11.2.1)
                                          // default: ["Explicit Nudity", "Nudity",
                                          //           "Graphic Male Nudity", "Graphic Female Nudity"]
    BotSignalThreshold           int64    // signals in window before auto-suspend (default: 5)
    BotSignalWindowMinutes       uint32   // time window for signal counting (default: 60)
    TextScanWordListRedisKey     string   // Redis key for keyword list
    AccountStatusCacheTTLSeconds uint64   // Redis cache TTL (default: 60)
}
```

---

## §9 RequireAdmin Extractor (Deep-Dive 1)

### §9.1 Design

`RequireAdmin` is Echo middleware defined in `00-core` (`internal/shared/middleware/`)
that verifies the authenticated user has platform administrator privileges. It is backed by a
new `is_platform_admin` boolean field on `iam_parents` and a corresponding field in
`AuthContext`. `[S§3.1.5]`

This pulls admin access from Phase 3 to Phase 1 because safety:: — which is in the Phase 1
critical path — requires admin endpoints for the moderation dashboard.

### §9.2 Required Changes to 00-core and 01-iam

**00-core changes** (specified in updated `00-core.md` alongside this spec):

1. Add `is_platform_admin: bool` field to `AuthContext` (§7.2)
2. Add `RequireAdmin` middleware to `internal/shared/middleware/` (§13.3)
3. Add `AccountSuspended` and `AccountBanned` variants to `AppError` (§6.1)
4. Add safety access check to auth middleware (§13.1, step 5.5)

**01-iam changes** (specified in updated `01-iam.md` alongside this spec):

1. Add `is_platform_admin BOOLEAN NOT NULL DEFAULT false` column to `iam_parents` (§3.1)
2. Populate `is_platform_admin` in auth middleware DB lookup
3. Expose `revoke_sessions(kratos_identity_id)` on `KratosAdapter` trait

### §9.3 Middleware Definition (in 00-core §13.3)

```go
// internal/shared/middleware/require_admin.go

// RequireAdmin returns Echo middleware that verifies the user is a platform administrator.
// Returns 403 Forbidden if the user is not an admin. [S§3.1.5]
func RequireAdmin() echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            auth, ok := c.Get("auth").(*shared.AuthContext)
            if !ok || auth == nil {
                return shared.ErrUnauthorized
            }

            if !auth.IsPlatformAdmin {
                return shared.ErrForbidden
            }

            return next(c)
        }
    }
}
```

**Phase 2**: Granular admin roles (moderation-only, billing-support-only) per `[S§3.1.5]`.
Phase 1 uses a single boolean — sufficient for the initial moderation dashboard.

---

## §10 CSAM Detection Pipeline (Deep-Dive 2)

The CSAM detection pipeline is a federal legal obligation under 18 U.S.C. § 2258A. Electronic
service providers that become aware of CSAM on their platform MUST report it to NCMEC. Failure
to report is a criminal offense. This pipeline is designed for zero ambiguity and maximum
reliability. `[S§12.1, V§7]`

### §10.1 End-to-End Flow

```
media::ProcessUploadJob
    │
    ├─ Step 3: CSAM scan via SafetyScanAdapter::scan_csam()
    │          → ThornAdapter::scan_csam() (PhotoDNA hash matching)
    │
    │  if scan_result.is_csam:
    │
    ├─ Step 3a: media:: quarantines upload immediately
    │           status → quarantined (irreversible)
    │           Publishes UploadQuarantined event
    │
    ├─ Step 3b: SafetyScanAdapter::report_csam()
    │           → SafetyService::handle_csam_detection()
    │
    └─ [Pipeline short-circuits — no further processing]


SafetyService::handle_csam_detection()
    │
    ├─ 1. Preserve evidence
    │     Copy quarantined file to evidence bucket (separate S3 path)
    │     Evidence is NEVER deleted — legal requirement [S§12.1]
    │
    ├─ 2. Create safety_ncmec_reports record (status: pending)
    │
    ├─ 3. Enqueue CsamReportJob (Critical queue — 30s target) [ARCH §12.2]
    │
    ├─ 4. Ban the account immediately
    │     Create safety_mod_actions record (action_type: account_banned)
    │     Update safety_account_status (status: banned)
    │     Invalidate Redis cache for this family_id
    │
    ├─ 5. Revoke all sessions
    │     iam::KratosAdapter::revoke_sessions(kratos_identity_id)
    │
    └─ 6. Do NOT notify the user [S§12.1]
          No event to notify::
          No email, no in-app notification
          Account simply becomes inaccessible
```

### §10.2 Quarantine Step

Quarantine is handled by `media::` (not `safety::`) because the upload record is owned by
`media::`. When CSAM is detected:

1. `media::` sets `status = 'quarantined'` on the upload record (irreversible)
2. `media::` publishes `UploadQuarantined` event
3. `safety::` handles the event by initiating the NCMEC pipeline

The quarantine status is **irreversible** — there is no `quarantined → published` transition
in the media status state machine `[09-media §3.1]`.

### §10.3 NCMEC Filing via Thorn

The `CsamReportJob` (Critical queue) handles the actual NCMEC CyberTipline submission:

```go
// internal/safety/jobs.go

// CsamReportJob submits CSAM report to NCMEC via Thorn Safer.
// Critical queue — target processing within 30 seconds. [ARCH §12.2]
type CsamReportJob struct {
    NcmecReportID uuid.UUID `json:"ncmec_report_id"`
}

func (j *CsamReportJob) ProcessTask(ctx context.Context, state *AppState) error {
    report, err := state.NcmecRepo.FindByID(ctx, j.NcmecReportID)
    if err != nil {
        return fmt.Errorf("find ncmec report: %w", err)
    }

    submission, err := state.Thorn.SubmitNcmecReport(ctx, NcmecReportPayload{
        UploadID:           report.UploadID,
        CsamHash:           report.CsamHash,
        Confidence:         report.Confidence,
        MatchedDatabase:    report.MatchedDatabase,
        EvidenceStorageKey: report.EvidenceStorageKey,
        UploaderFamilyID:   report.FamilyID,
        UploaderParentID:   report.ParentID,
        UploadTimestamp:    report.CreatedAt,
    })

    if err != nil {
        slog.Error("NCMEC submission failed — will retry",
            "ncmec_report_id", j.NcmecReportID,
            "error", err,
        )
        errMsg := err.Error()
        if _, updateErr := state.NcmecRepo.UpdateStatus(
            ctx, j.NcmecReportID, "failed", nil, &errMsg,
        ); updateErr != nil {
            slog.Error("failed to update ncmec report status", "error", updateErr)
        }
        return fmt.Errorf("ncmec submission retryable: %w", err)
    }

    if _, err := state.NcmecRepo.UpdateStatus(
        ctx, j.NcmecReportID, "submitted", &submission.NcmecReportID, nil,
    ); err != nil {
        return fmt.Errorf("update ncmec report status: %w", err)
    }

    return nil
}
```

### §10.4 Evidence Preservation

CSAM evidence MUST be preserved as required by law enforcement `[S§12.1]`:

- Evidence is copied to a separate S3 path: `evidence/csam/{ncmec_report_id}/{upload_id}`
- The evidence bucket/path has a **separate retention policy** — no lifecycle rules, no
  automatic deletion
- The `safety_ncmec_reports.evidence_storage_key` records the exact S3 key
- Evidence files are NEVER served via the CDN or any API endpoint
- Access to the evidence path requires separate IAM credentials (not the application's
  normal S3 credentials)

### §10.5 Zero User Notification

Per `[S§12.1]`: "The platform MUST NOT notify the offending user of the detection or report."

- No `ContentFlagged` or `ModerationActionTaken` event is published for CSAM cases
- No notification to `notify::` domain
- No email, no in-app notification, no WebSocket push
- The user's session is silently revoked via Kratos
- On next access attempt, the auth middleware returns 403 (account banned) with a generic
  message that does not mention CSAM

### §10.6 Permanent Ban

CSAM-associated accounts are immediately and permanently banned `[S§12.1]`:

- `safety_account_status.status = 'banned'`
- `safety_account_status.ban_reason = 'csam_violation'`
- CSAM bans are NOT appealable (enforced in `submit_appeal()` validation)
- All active sessions are revoked via `iam::KratosAdapter::revoke_sessions()`

### §10.7 CSAM Rescan (Event-Driven)

> **Refinement of ARCH §12.3**: The architecture sketch specifies a daily
> `PeriodicCsamRescanJob`. This spec changes the trigger from a daily cron schedule to
> **event-driven** — rescans are triggered when Thorn notifies us of updated hash databases,
> not on a fixed schedule. Rationale: at scale (millions of uploads), a daily blanket rescan
> generates astronomical load and API traffic for negligible incremental safety benefit. An
> event-driven approach rescans only when new hashes are actually available.

`CsamRescanJob` (Low queue, triggered by hash database update notification):

- **Trigger**: Thorn Safer webhook or polling endpoint indicates new hashes are available.
  The `ThornAdapter` exposes a `check_hash_update()` method; a lightweight
  `CheckCsamHashUpdateJob` (Low queue, daily) checks for updates and enqueues `CsamRescanJob`
  only when new hashes exist.
- Uses `media_uploads.last_csam_scanned_at` column (added to `09-media`) to batch scans
  — only uploads scanned before the hash update timestamp need rescanning
- Processes uploads in batches (100 per iteration) to control throughput
- New CSAM matches trigger the full pipeline (quarantine → NCMEC → ban)
- If no hash update has occurred, no rescan runs — zero wasted work

---

## §11 Content Moderation Pipeline (Deep-Dive 3)

### §11.1 Text Content Scanning

Phase 1 uses keyword + regex matching for synchronous text screening. This avoids external
API call latency while providing basic protection. `[S§12.2]`

```go
// internal/safety/text_scanner.go

// TextScanner performs synchronous text scanning. Called by social, mkt, and learn
// before persisting user-generated text content (posts, comments, messages, reviews,
// question text, answer text, quiz titles/descriptions, sequence titles/descriptions).
//
// Phase 1: Redis-cached keyword list + regex patterns.
// Phase 2: AWS Comprehend for ML-based detection.
type TextScanner struct {
    redis  *redis.Client
    config SafetyConfig
}

// Scan checks text against keyword list and regex patterns.
//
// Word lists are stored in Redis as a sorted set at the key
// specified by SafetyConfig.TextScanWordListRedisKey.
// Lists are loaded at startup and refreshed via admin API.
//
// Returns TextScanResult with severity:
// - "none": no matches
// - "low": soft matches (may be false positives) — log only
// - "high": definite policy violations — block + flag
// - "critical": child safety terms — block + flag + escalate
func (s *TextScanner) Scan(ctx context.Context, text string) (*TextScanResult, error) {
    // 1. Normalize text (lowercase, strip diacritics, collapse whitespace)
    // 2. Check against keyword list (O(n) scan, n = term count)
    // 3. Check against regex patterns (compiled at startup)
    // 4. Return aggregated result with matched terms
    panic("not implemented")
}
```

**Word list management**: Admins update word lists via a dedicated admin endpoint (Phase 2).
Phase 1 loads word lists from a seed file at startup and caches in Redis.

### §11.2 Media Content Scanning

Media content (images, video) is scanned during the `media::ProcessUploadJob` pipeline:

1. **CSAM scan** (step 3) — via `ThornAdapter::scan_csam()` — see §10
2. **Content moderation** (step 4) — via `RekognitionAdapter::detect_moderation_labels()`

When `media::` publishes `UploadRejected` or `UploadFlagged`, `safety::` creates a
`safety_content_flags` record with `source = 'automated'` and the moderation labels.

### §11.2.1 Nudity Auto-Rejection Policy

**Platform policy**: All nudity and explicit sexual content is auto-rejected on upload. This
is a children's platform; there is no legitimate use case for nude imagery in user-generated
content in Phase 1. `[S§12.2, V§7]`

**Triggering labels**: Rekognition labels at confidence ≥ `rekognition_min_confidence`
(default 70%):
- `Explicit Nudity`
- `Nudity`
- `Graphic Male Nudity`
- `Graphic Female Nudity`

The specific label names are configurable via `SafetyConfig::nudity_auto_reject_labels`
(see §8.4) to accommodate future Rekognition API changes without code modification.

**Mechanism**: `SafetyScanBridge::scan_moderation()` reads the raw Rekognition labels,
applies the label routing table (§11.2.2), and returns a `ModerationResult` with
`auto_reject: bool`. When `auto_reject` is true, `media::ProcessUploadJob` sets status →
`rejected` (instead of `flagged`) and publishes an `UploadRejected` event. `[09-media §10.4]`

**User notification**: On `UploadRejected` event, `safety::` publishes a notification via
`notify::` with a generic message: *"Your upload was not published because it violates our
content guidelines."* The notification does NOT specify which policy was violated — this
prevents users from gaming the detection system by iteratively testing boundary content.

**Appeals**: Rejected uploads are appealable through the standard appeals flow (§12.4).
This handles the anatomy textbook edge case: a parent can appeal, an admin reviews the
content and context, and if it is legitimate educational material, the admin overrides the
upload status to `published`. `[09-media §3.1: rejected → published]`

**Audit trail**: When an upload is auto-rejected, `safety::` creates a `safety_content_flags`
record with:
- `source = 'automated'`
- `flag_type = 'explicit_content'`
- `auto_rejected = true` (new column — see §3.2 TABLE 2)
- `labels` = the Rekognition labels that triggered rejection (JSONB)
- `confidence` = highest confidence among triggering labels

### §11.2.2 Rekognition Label Routing

`SafetyScanBridge::scan_moderation()` applies a decision table to map raw Rekognition labels
to upload outcomes. The routing logic lives entirely in the bridge — the `RekognitionAdapter`
returns raw labels without filtering. `[09-media §7.2, §8.3]`

**Decision table**:

| Label Category | Example Labels | Confidence Threshold | Outcome |
|---|---|---|---|
| Nudity/Explicit | `Explicit Nudity`, `Nudity`, `Graphic Male Nudity`, `Graphic Female Nudity` | ≥ 70% | **Auto-reject** |
| Suggestive | `Suggestive`, `Female Swimwear Or Underwear`, `Male Swimwear Or Underwear`, `Revealing Clothes` | ≥ 80% | **Flag for review** (normal priority) |
| Violence/Graphic | `Violence`, `Graphic Violence Or Gore`, `Self-Injury` | ≥ 70% | **Flag for review** (normal priority) — age-appropriateness check, appealable |
| Drugs/Tobacco/Alcohol | all | — | **Ignored** — legitimate educational content (science, health, history) |
| Hate symbols | all | — | **Ignored** — legitimate educational content (history, civics, fiction) |
| Weapons | all | — | **Ignored** — legitimate educational content (history, military studies) |
| Suspected underage + sexual | Any nudity/suggestive label co-occurring with Rekognition's age-related indicators (e.g. `Minor` label or age range estimate) | ≥ 50% (lower threshold for child safety) | **Flag for review** (critical priority — same SLA as CSAM reports: < 24h) |

> **Rationale for ignored categories**: This is a homeschooling platform. Educational resources
> routinely contain imagery of historical atrocities (hate symbols), substance abuse education
> (drugs/alcohol), and warfare (weapons/violence). Flagging or rejecting these would render
> the platform unusable for history, science, and health curricula. Only content that poses
> direct harm to children (sexual content, exploitation) warrants automated enforcement.

**Routing logic** (`SafetyScanBridge::scan_moderation()`):

```go
// Pseudocode — actual implementation follows Go patterns
func applyLabelRouting(rawLabels []ModerationLabel, config *SafetyConfig) *ModerationResult {
    autoReject := false
    hasViolations := false
    var priority *string
    var keptLabels []ModerationLabel

    hasUnderageIndicator := false
    for _, l := range rawLabels {
        if isUnderageLabel(l.Name) && l.Confidence >= 50.0 {
            hasUnderageIndicator = true
            break
        }
    }

    for _, label := range rawLabels {
        if isIgnoredCategory(label.Name) {
            continue // Drugs, hate symbols, weapons — skip entirely
        }

        if contains(config.NudityAutoRejectLabels, label.Name) &&
            label.Confidence >= config.RekognitionMinConfidence {
            autoReject = true
            hasViolations = true
            keptLabels = append(keptLabels, label)
        } else if isSuggestive(label.Name) && label.Confidence >= 80.0 {
            hasViolations = true
            keptLabels = append(keptLabels, label)
            // Upgrade priority if underage indicators co-occur
            if hasUnderageIndicator {
                p := "critical"
                priority = &p
            } else if priority == nil {
                p := "normal"
                priority = &p
            }
        } else if isViolence(label.Name) &&
            label.Confidence >= config.RekognitionMinConfidence {
            hasViolations = true
            keptLabels = append(keptLabels, label)
            if priority == nil {
                p := "normal"
                priority = &p
            }
        }
    }

    // Underage + any nudity label → critical even if auto-rejecting
    if hasUnderageIndicator && hasViolations && !autoReject {
        p := "critical"
        priority = &p
    }

    var resultPriority *string
    if !autoReject {
        resultPriority = priority
    }

    return &ModerationResult{
        HasViolations: hasViolations,
        AutoReject:    autoReject,
        Labels:        keptLabels,
        Priority:      resultPriority,
    }
}
```

**Critical priority routing**: When suspected underage + sexual labels co-occur (and the
content is not auto-rejected), the resulting `safety_content_flags` record gets
`flag_type = 'suspected_underage_exploitation'` and is routed to the top of the admin review
queue at `critical` priority. This matches the CSAM report SLA: review within 24 hours.

**Priority field on ModerationResult**: `Priority *string` — `nil` for auto-reject
outcomes (rejected uploads don't enter the review queue), `Some("critical"|"high"|"normal")`
for flagged items. The `UploadFlagged` event carries this priority through to the
`safety_content_flags` record. `[09-media §16.3]`

### §11.2.3 Creator-Authored Interactive Content

**Creator-Authored Interactive Content**: Questions, quizzes, and lesson sequences created via
the platform's authoring tools (`learn::` domain, triggered by `mkt::` creator workflows) MUST
pass through the same content moderation pipeline as other UGC. Specifically:

- Question text and answer text are screened for inappropriate content via text scanning
  (`SafetyService::scan_text()`)
- Media attachments on questions follow the standard image/video moderation pipeline (CSAM scan
  + content moderation labels) via the `media::ProcessUploadJob` → `SafetyScanAdapter` path
- Quiz and sequence titles/descriptions are screened as text content via
  `SafetyService::scan_text()`
- Content that fails moderation is blocked from publication (same Draft → Submitted → Published
  lifecycle as marketplace listings)
- Screening occurs on the `ListingPublished` event for marketplace-distributed content, or on
  creation for platform-internal content

The content types screened by the moderation pipeline now include:

| Content Type | Source Domain | Table | Text Fields Scanned | Media Fields Scanned |
|---|---|---|---|---|
| Posts, comments | `social::` | `social_posts`, `social_comments` | Body text | Attached uploads |
| Messages | `social::` | `social_messages` | Message text | Attached uploads |
| Reviews | `mkt::` | `mkt_reviews` | Review text | — |
| Listings | `mkt::` | `mkt_listings` | Title, description | Attached uploads |
| **Questions** | `learn::` | `learn_questions` | Question text, answer text | Media attachments |
| **Quizzes** | `learn::` | `learn_quiz_defs` | Title, description | — |
| **Sequences** | `learn::` | `learn_sequence_defs` | Title, description | — |

### §11.3 Community Reporting Flow

Users report content via `POST /v1/safety/reports`. `[S§12.3]`

```
User submits report
    │
    ├─ 1. Validate target exists (cross-domain lookup)
    ├─ 2. Check for duplicate (same reporter + target within 24h)
    ├─ 3. Create safety_reports record
    ├─ 4. Set priority based on category:
    │     csam_child_safety → critical (< 24h SLA)
    │     harassment → high
    │     all others → normal (< 72h SLA)
    ├─ 5. Create safety_content_flags record (source: community_report)
    ├─ 6. Publish ContentReported event → notify:: (admin queue alert)
    └─ 7. Return acknowledgment to reporter [S§12.3]
```

### §11.4 Admin Review Queue

The moderation queue is a prioritized list of unresolved reports and unreviewed flags.

**Priority ordering**:
1. `critical` priority reports (CSAM/child safety) — must review within 24 hours `[S§12.3]`
2. `high` priority reports (harassment) — should review within 72 hours
3. Automated flags with high confidence
4. `normal` priority reports

**Queue API**: `GET /v1/admin/safety/reports?status=pending&sort=priority` returns reports
sorted by priority (critical first), then by creation time (oldest first within each priority).

**Queue includes auto-rejected content flags**: Flags with `auto_rejected = true` appear in
the admin queue as informational audit records. While the upload was already rejected, admins
can review these records, and if the flag is a `suspected_underage_exploitation` type, it
receives critical priority and must be reviewed for potential CSAM escalation (§11.4.1).

### §11.4.1 CSAM Escalation from Review Queue

When an admin reviewing flagged or auto-rejected content determines the content is actual
CSAM (novel material not in PhotoDNA's hash database), they escalate it through the same
§10 pipeline used for automated CSAM detection. This is a federal legal obligation —
18 U.S.C. § 2258A applies regardless of whether CSAM was identified by software or a
human reviewer. `[S§12.1]`

```
Admin confirms CSAM (PATCH /v1/admin/safety/flags/:id/escalate-csam)
    │
    ├─ 1. Validate: flag exists and is unreviewed
    ├─ 2. Validate: caller is platform admin (RequireAdmin)
    ├─ 3. Mark flag as reviewed (reviewed = true, action_taken = true)
    ├─ 4. Create safety_mod_actions record (action_type: escalate_to_csam)
    │     with admin_notes as reason (required audit trail justification)
    ├─ 5. Delegate to SafetyService::handle_csam_detection()
    │     → Same §10 pipeline: evidence preservation → NCMEC → ban → session revoke
    │     → The CsamScanResult passed in has:
    │       is_csam = true, hash = None (human-identified, not hash-matched),
    │       confidence = None, matched_database = None
    ├─ 6. media:: status changes from flagged/rejected → quarantined (irreversible)
    └─ 7. Zero user notification (same as §10.5)
```

**Key design points**:
- Reuses the existing `handle_csam_detection()` service method — no new pipeline needed.
  The method already accepts `CsamScanResult` with optional hash fields.
- The `safety_ncmec_reports` record will have `csam_hash = NULL` and
  `matched_database = NULL` (human-identified, not hash-matched). The `confidence` field
  is NULL. Thorn's API supports reports without a hash match (human-identified CSAM is
  still reportable to NCMEC).
- The `escalate_to_csam` mod action type creates a distinct audit trail entry showing
  that a human escalated content to CSAM (vs. automated detection via §10).
- CSAM bans from this path are equally non-appealable (same §10.6 rule).
- The upload's status transitions to `quarantined` regardless of its previous state
  (`flagged` or `rejected`). This is a new valid transition:
  `flagged → quarantined` and `rejected → quarantined` (CSAM escalation only).

**Endpoint**: `PATCH /v1/admin/safety/flags/:id/escalate-csam` — see §4.2 (A15).

**Request type**: `EscalateCsamCommand` — see §8.1.

### §11.5 Reporter Notification

Per `[S§12.3]`: "Reporters MUST receive acknowledgment of their report and notification of
the outcome (without revealing details of actions taken against the reported user)."

- On report submission: immediate `201 Created` response serves as acknowledgment
- On report resolution: `notify::` sends a notification to the reporter with:
  - "Your report has been reviewed"
  - "Appropriate action has been taken" (if action taken) OR "No violation was found"
  - Does NOT reveal specific actions (suspension duration, content removed, etc.)

### §11.6 Report State Machine

```
pending ──► in_review ──► resolved_action_taken
                     ──► resolved_no_action
                     ──► dismissed
```

- `pending`: initial state, visible in admin queue
- `in_review`: admin has assigned themselves, actively reviewing
- `resolved_action_taken`: reviewed, moderation action taken (linked via `report_id` on mod action)
- `resolved_no_action`: reviewed, no violation found
- `dismissed`: invalid report or duplicate

---

## §12 Moderation Actions & Account Management (Deep-Dive 4)

### §12.1 Moderation Actions

All moderation actions are **immutable audit records** in `safety_mod_actions`. `[S§12.7]`

| Action Type | Effect | Reversible? |
|-------------|--------|-------------|
| `content_removed` | Target content hidden from all users | Yes (via `content_restored`) |
| `warning_issued` | Warning notification sent to family | N/A |
| `account_suspended` | Temporary access restriction | Yes (expires or lifted) |
| `account_banned` | Permanent access restriction | Only via appeal (except CSAM) |
| `content_restored` | Previously removed content made visible again | N/A |
| `suspension_lifted` | Suspension ended early (before expiry) | N/A |
| `appeal_granted` | Appeal approved, reverses original action | N/A |
| `escalate_to_csam` | Admin escalated flagged content to CSAM — triggers §10 pipeline | No (irreversible, same as automated CSAM) |

### §12.2 Account Status State Machine

```
active ──► suspended ──► active        (suspension expired or lifted)
                    ──► banned         (escalation during suspension)

active ──► banned                      (immediate ban, e.g., CSAM)

banned ──► active                      (appeal granted — NOT for CSAM bans)
```

**Status enforcement**: The auth middleware calls `SafetyService::check_account_access()`
to verify account status on every authenticated request. See §12.3.

### §12.3 Auth Middleware Integration

The auth middleware (defined in `00-core §13.1`) adds a safety check step after building
`AuthContext`:

```
Auth Middleware Flow (updated):
1. Extract session cookie
2. Validate session via Kratos
3. Look up parent + family in local DB
4. Build AuthContext (including is_platform_admin)
5. [NEW] Check account access via SafetyService::check_account_access()
   → Redis cache lookup: key = "safety:account:{family_id}"
   → Cache miss: query safety_account_status table, cache result with 60s TTL
   → If suspended/banned: return AccountSuspended/AccountBanned error
6. Insert AuthContext into request extensions
7. Call next handler
```

**Redis cache strategy**:
- Key format: `safety:account:{family_id}`
- Value: account status string (`"active"`, `"suspended"`, `"banned"`)
- TTL: 60 seconds
- Cache invalidation: when a moderation action changes account status, the cache key is
  explicitly deleted (forcing a DB lookup on next request)
- Performance impact: ~0.01% of accounts are restricted at any time; the Redis lookup
  adds < 1ms to every request, which is negligible compared to the Kratos session check

**Latency budget**: The cache lookup is an in-memory Redis GET (~0.1ms). Even on cache miss,
the DB query on `safety_account_status` (indexed by PK) is < 1ms. Total impact on the auth
middleware hot path: < 1ms in all cases.

### §12.4 Appeals Process

Per `[S§12.2]`: "Appeals MUST be reviewed by a different moderator than the one who took the
original action."

```
User submits appeal (POST /v1/safety/appeals)
    │
    ├─ Validate: action exists and targets caller's family
    ├─ Validate: no existing appeal for this action (unique constraint)
    ├─ Validate: action is not a CSAM ban (not appealable)
    ├─ Create safety_appeals record (status: pending)
    └─ Publish AppealSubmitted event → notify:: (admin queue alert)

Admin resolves appeal (PATCH /v1/admin/safety/appeals/:id)
    │
    ├─ Validate: assigned admin ≠ original action admin [S§12.2]
    ├─ If granted:
    │   ├─ Create suspension_lifted or content_restored mod action
    │   ├─ Update account status if applicable
    │   ├─ Invalidate Redis cache
    │   └─ Publish AppealResolved event → notify:: (user notification)
    └─ If denied:
        └─ Publish AppealResolved event → notify:: (user notification)
```

### §12.5 Suspension Expiry

Suspensions have a `suspension_expires_at` timestamp. Two mechanisms handle expiry:

1. **Lazy check**: `check_account_access()` compares `suspension_expires_at` against `now()`.
   If expired, it updates the status to `active` and invalidates the cache.
2. **Background job** (Phase 2): `ExpireSuspensionsJob` runs hourly, finds expired suspensions,
   and creates `suspension_lifted` mod action records for audit completeness.

Phase 1 uses only the lazy check — this is sufficient because suspended accounts are checked
on every request attempt.

---

## §13 Bot Prevention (Deep-Dive 5)

### §13.1 CAPTCHA (Delegated to iam::)

Per `[S§12.4]`: "Account creation MUST include CAPTCHA or equivalent bot detection."

CAPTCHA is implemented in `iam::` during the registration flow (Kratos post-registration
webhook). `safety::` specifies the requirement; `iam::` implements it. The CAPTCHA service
is a Turnstile (Cloudflare) integration in the frontend, validated server-side during
registration.

### §13.2 Behavioral Bot Detection

`safety::` monitors for bot-like patterns via signals recorded by other domains. `[S§12.4]`

| Signal | Source Domain | Threshold | Action |
|--------|--------------|-----------|--------|
| `rapid_posting` | `social::` | > 10 posts in 5 minutes | Auto-flag + record signal |
| `mass_friend_requests` | `social::` | > 20 requests in 1 hour | Auto-flag + record signal |
| `repetitive_content` | `social::` | > 3 identical posts in 1 hour | Auto-flag + record signal |
| `suspicious_registration` | `iam::` | CAPTCHA score below threshold | Record signal |
| `rate_limit_exceeded` | `00-core` middleware | > 5 rate limit hits in 10 minutes | Record signal |

**Auto-suspend threshold**: When a parent accumulates `SafetyConfig::bot_signal_threshold`
(default: 5) signals within `SafetyConfig::bot_signal_window_minutes` (default: 60 minutes),
the account is automatically suspended pending review.

```go
// RecordBotSignal is called by other domains when suspicious behavior is detected.
func (s *SafetyServiceImpl) RecordBotSignal(
    ctx context.Context,
    familyID uuid.UUID,
    parentID uuid.UUID,
    signal BotSignalType,
    details json.RawMessage,
) error {
    // 1. Create bot signal record
    if _, err := s.botSignalRepo.Create(ctx, CreateBotSignalRow{
        FamilyID:   familyID,
        ParentID:   parentID,
        SignalType:  string(signal),
        Details:     details,
    }); err != nil {
        return fmt.Errorf("create bot signal: %w", err)
    }

    // 2. Check threshold
    count, err := s.botSignalRepo.CountRecent(ctx, parentID, s.config.BotSignalWindowMinutes)
    if err != nil {
        return fmt.Errorf("count recent signals: %w", err)
    }

    if count >= s.config.BotSignalThreshold {
        // Auto-suspend pending review
        if _, err := s.AdminSuspendAccount(
            ctx,
            shared.SystemAuthContext(), // system-initiated
            familyID,
            SuspendAccountCommand{
                Reason:         "Automated suspension: bot-like behavior detected",
                SuspensionDays: 1, // 24h pending manual review
            },
        ); err != nil {
            return fmt.Errorf("auto-suspend account: %w", err)
        }
    }

    return nil
}
```

---

## §14 Child Safety Beyond CSAM (Deep-Dive 6)

### §14.1 Age-Appropriate Content Filtering

Per `[S§12.6]`: "The platform MUST enforce age-appropriate content filtering for any content
surfaces accessible to Student Profiles."

Student profiles are parent-mediated (`[S§3.3]`) — students do not have independent accounts.
Content filtering applies to surfaces where a parent views content on behalf of a student:

- `social::` enforces visibility rules: student-tagged content is filtered based on family
  privacy settings `[05-social §14]`
- `safety::` provides the `scan_text()` method used by `social::` and `mkt::` to screen
  user-generated content before persistence
- Flagged content is hidden until admin review

Per `[S§12.6]`: "The platform MUST prohibit adults from initiating direct contact with
Student Profiles."

This is enforced by `social::` (students have no messaging capability — `[S§3.3]`).
`safety::` monitors for violations if the constraint is ever bypassed.

### §14.2 Grooming Detection (Phase 2)

Per `[S§12.6]`: "The platform MUST implement monitoring for grooming behavior patterns."

Phase 2 adds ML-based grooming pattern detection via AWS Comprehend:
- Analyzes message patterns (frequency, content, relationship asymmetry)
- Flags conversations matching grooming behavioral profiles
- Routes flags to priority review queue
- This is a Phase 2 feature due to ML model training requirements

### §14.3 Parental Controls (Phase 2)

Per `[S§12.6]`: "Parental controls MUST allow parents to restrict their students' content
exposure beyond platform defaults."

Phase 2 adds configurable parental control settings:
- Content category filters (beyond platform defaults)
- Contact restrictions (limit who can interact with family content)
- Time-of-day restrictions for student-facing features

### §14.4 Group-Level Moderation

Per `[S§12.7]`: "The moderation system MUST support moderator roles for user-created groups."

Group-level moderation is owned by `social::` (`[05-social §14]`). Group admins can:
- Remove posts/comments within their group
- Mute users within their group
- Escalate to platform moderators via the reporting system

`safety::` handles platform-level moderation only. The reporting system (§11.3) is the
bridge — group admins report severe issues to platform moderators.

---

## §15 Error Types

### §15.1 SafetyError Enum

```go
// internal/safety/errors.go

var (
    ErrReportNotFound          = errors.New("report not found")
    ErrFlagNotFound            = errors.New("content flag not found")
    ErrActionNotFound          = errors.New("moderation action not found")
    ErrAppealNotFound          = errors.New("appeal not found")
    ErrAccountNotFound         = errors.New("account status not found")
    ErrDuplicateReport         = errors.New("duplicate report within 24 hours")
    ErrAppealAlreadyExists     = errors.New("appeal already exists for this action")
    ErrCsamBanNotAppealable    = errors.New("CSAM bans are not appealable")
    ErrSameAdminAppeal         = errors.New("appeal reviewer must differ from original action admin")
    ErrInvalidActionType       = errors.New("invalid action type")
    ErrInvalidReportTransition = errors.New("invalid report status transition")
    ErrFlagAlreadyReviewed     = errors.New("flag already reviewed — cannot escalate")
    ErrAccountSuspended        = errors.New("account is suspended")
    ErrAccountBanned           = errors.New("account is banned")
)
```

### §15.2 Error-to-HTTP Mapping

| Variant | HTTP Status | JSON `code` | User-Facing Message |
|---------|-------------|-------------|---------------------|
| `ReportNotFound` | 404 | `not_found` | "Resource not found" |
| `FlagNotFound` | 404 | `not_found` | "Resource not found" |
| `ActionNotFound` | 404 | `not_found` | "Resource not found" |
| `AppealNotFound` | 404 | `not_found` | "Resource not found" |
| `AccountNotFound` | 404 | `not_found` | "Resource not found" |
| `DuplicateReport` | 409 | `duplicate_report` | "A report for this content was already submitted recently" |
| `AppealAlreadyExists` | 409 | `appeal_exists` | "An appeal already exists for this action" |
| `CsamBanNotAppealable` | 422 | `not_appealable` | "This action cannot be appealed" |
| `SameAdminAppeal` | 422 | `same_admin` | "Appeal must be reviewed by a different administrator" |
| `InvalidActionType` | 422 | `validation_error` | "Invalid action type" |
| `InvalidReportTransition` | 422 | `validation_error` | "Invalid status transition" |
| `FlagAlreadyReviewed` | 422 | `flag_already_reviewed` | "This flag has already been reviewed" |
| `AccountSuspended` | 403 | `account_suspended` | "Your account has been temporarily suspended" |
| `AccountBanned` | 403 | `account_banned` | "Your account has been permanently restricted" |
| `ThornError` | 502 | `internal_error` | "An internal error occurred" |
| `RekognitionError` | 502 | `internal_error` | "An internal error occurred" |
| `DatabaseError` | 500 | `internal_error` | "An internal error occurred" |

### §15.3 AppError Conversion

```go
// ToAppError maps safety domain errors to shared AppError responses.
// Used in handlers to convert service errors to HTTP responses.
func ToAppError(err error) *shared.AppError {
    switch {
    case errors.Is(err, ErrReportNotFound),
         errors.Is(err, ErrFlagNotFound),
         errors.Is(err, ErrActionNotFound),
         errors.Is(err, ErrAppealNotFound),
         errors.Is(err, ErrAccountNotFound):
        return shared.NewNotFoundError()

    case errors.Is(err, ErrDuplicateReport),
         errors.Is(err, ErrAppealAlreadyExists):
        return shared.NewConflictError(err.Error())

    case errors.Is(err, ErrCsamBanNotAppealable),
         errors.Is(err, ErrSameAdminAppeal),
         errors.Is(err, ErrInvalidActionType),
         errors.Is(err, ErrInvalidReportTransition),
         errors.Is(err, ErrFlagAlreadyReviewed):
        return shared.NewValidationError(err.Error())

    case errors.Is(err, ErrAccountSuspended):
        return shared.NewAccountSuspendedError()

    case errors.Is(err, ErrAccountBanned):
        return shared.NewAccountBannedError()

    case errors.Is(err, ErrThornUnavailable),
         errors.Is(err, ErrThornAPI),
         errors.Is(err, ErrNcmecSubmissionFailed):
        slog.Error("Thorn adapter error", "error", err)
        return shared.NewInternalError("safety scan error")

    case errors.Is(err, ErrRekognitionUnavailable),
         errors.Is(err, ErrRekognitionAPI):
        slog.Error("Rekognition adapter error", "error", err)
        return shared.NewInternalError("safety scan error")

    default:
        slog.Error("unexpected safety error", "error", err)
        return shared.NewInternalError("internal error")
    }
}
```

---

## §16 Cross-Domain Interactions

### §16.1 safety:: Provides (Consumed by Other Domains)

| Consumer | Interface | Usage |
|----------|-----------|-------|
| `media::` | `SafetyScanAdapter` impl (`SafetyScanBridge`) | CSAM + moderation scanning during ProcessUploadJob `[09-media §7.2]` |
| `social::` | `SafetyService::scan_text()` | Text screening before persisting posts/comments/messages |
| `mkt::` | `SafetyService::scan_text()` | Text screening before persisting reviews/listings |
| `learn::` | `SafetyService::scan_text()` | Text screening before persisting question text, answer text, quiz/sequence titles and descriptions |
| `social::` | `SafetyService::record_bot_signal()` | Behavioral bot detection signals |
| `00-core` | `SafetyService::check_account_access()` | Auth middleware account status check |
| `00-core` | `RequireAdmin` extractor | Admin-only endpoint gating |

### §16.2 safety:: Consumes

| Provider | Interface | Usage |
|----------|-----------|-------|
| `iam::` | `AuthContext` extractor | Authentication `[00-core §7.2]` |
| `iam::` | `FamilyScope` extractor | Family-scoped data access `[00-core §8]` |
| `iam::` | `IamService` | User/family lookup for report validation |
| `iam::` | `KratosAdapter::revoke_sessions()` | Session revocation on ban |
| `media::` | `UploadQuarantined` event | CSAM pipeline trigger |
| `media::` | `UploadRejected` event | Auto-rejected content flag + user notification `[§11.2.1]` |
| `media::` | `UploadFlagged` event | Moderation flag creation |

### §16.3 Events safety:: Publishes

Defined in `internal/safety/events.go`. `[CODING §8.4]`

```go
// internal/safety/events.go

// ContentReported is published when a community report is submitted.
// Consumed by notify (admin queue alert notification).
type ContentReported struct {
    ReportID   uuid.UUID `json:"report_id"`
    TargetType string    `json:"target_type"`
    TargetID   uuid.UUID `json:"target_id"`
    Category   string    `json:"category"`
    Priority   string    `json:"priority"`
}

func (ContentReported) EventName() string { return "safety.content_reported" }

// ModerationActionTaken is published when a moderation action is taken (EXCEPT for CSAM cases).
// Consumed by notify (notification to affected user),
// social (hide removed content), mkt (hide removed listing/review).
type ModerationActionTaken struct {
    ActionID       uuid.UUID  `json:"action_id"`
    ActionType     string     `json:"action_type"`
    TargetFamilyID uuid.UUID  `json:"target_family_id"`
    TargetType     *string    `json:"target_type,omitempty"`     // "post", "listing", etc. (for content actions)
    TargetID       *uuid.UUID `json:"target_id,omitempty"`
}

func (ModerationActionTaken) EventName() string { return "safety.moderation_action_taken" }

// AccountSuspendedEvent is published when an account is suspended.
// Consumed by notify (suspension notification to user).
type AccountSuspendedEvent struct {
    FamilyID       uuid.UUID `json:"family_id"`
    SuspensionDays int32     `json:"suspension_days"`
    ExpiresAt      time.Time `json:"expires_at"`
}

func (AccountSuspendedEvent) EventName() string { return "safety.account_suspended" }

// AppealResolved is published when an appeal is resolved.
// Consumed by notify (appeal outcome notification to user).
type AppealResolved struct {
    AppealID uuid.UUID `json:"appeal_id"`
    FamilyID uuid.UUID `json:"family_id"`
    Status   string    `json:"status"` // "granted" or "denied"
}

func (AppealResolved) EventName() string { return "safety.appeal_resolved" }

// UploadAutoRejectedNotification is published when an upload is auto-rejected by content policy (§11.2.1).
// Consumed by notify (generic rejection notification to uploader).
// Message: "Your upload was not published because it violates our content guidelines."
// Does NOT specify which policy was violated (prevents gaming).
type UploadAutoRejectedNotification struct {
    FamilyID uuid.UUID `json:"family_id"`
    UploadID uuid.UUID `json:"upload_id"`
}

func (UploadAutoRejectedNotification) EventName() string { return "safety.upload_auto_rejected" }
```

### §16.4 Events safety:: Subscribes To

| Event | Source | Handler | Effect |
|-------|--------|---------|--------|
| `PostCreated` | `social::` | `OnPostCreated` | Scan text content, create flag if violation found |
| `MessageSent` | `social::` | `OnMessageSent` | Grooming detection (Phase 2). Note: `MessageSent` event does not include message content (PII concern). Handler retrieves text via `social::SocialService::get_message()`. |
| `ReviewCreated` | `mkt::` | `OnReviewCreated` | Scan review text, create flag if violation found |
| `UploadQuarantined` | `media::` | `OnUploadQuarantined` | Initiate CSAM pipeline (§10) |
| `UploadRejected` | `media::` | `OnUploadRejected` | Create auto-rejected content flag + notify user (§11.2.1) |
| `UploadFlagged` | `media::` | `OnUploadFlagged` | Create content flag record |
| `MessageReported` | `social::` | `OnMessageReported` | Create report + flag from social message report |

```go
// internal/safety/event_handlers.go

// Imports: social.PostCreated, social.MessageSent, social.MessageReported,
//          mkt.ReviewCreated, media.UploadQuarantined, media.UploadRejected, media.UploadFlagged

type OnPostCreated struct {
    safetyService SafetyService
}

func (h *OnPostCreated) Handle(ctx context.Context, event *social.PostCreated) error {
    // Scan post text content for policy violations
    if event.Content != nil {
        result, err := h.safetyService.ScanText(ctx, *event.Content)
        if err != nil {
            return fmt.Errorf("scan post text: %w", err)
        }
        if result.HasViolations {
            // Create content flag for admin review
            // Target: post, target_id: event.PostID
        }
    }
    return nil
}

type OnUploadQuarantined struct {
    safetyService SafetyService
}

func (h *OnUploadQuarantined) Handle(ctx context.Context, event *media.UploadQuarantined) error {
    // Initiate CSAM pipeline: evidence preservation → NCMEC report → ban
    return h.safetyService.HandleCsamDetection(
        ctx,
        event.UploadID,
        event.FamilyID,
        // CsamScanResult is retrieved from the media upload record
        &media.CsamScanResult{IsCsam: true},
    )
}

type OnUploadRejected struct {
    flagRepo ContentFlagRepository
    events   *shared.EventBus
}

func (h *OnUploadRejected) Handle(ctx context.Context, event *media.UploadRejected) error {
    // Create auto-rejected content flag [§11.2.1]
    maxConfidence := 0.0
    for _, l := range event.Labels {
        if l.Confidence > maxConfidence {
            maxConfidence = l.Confidence
        }
    }

    labelsJSON, err := json.Marshal(event.Labels)
    if err != nil {
        return fmt.Errorf("marshal labels: %w", err)
    }

    if _, err := h.flagRepo.Create(ctx, CreateContentFlagRow{
        Source:         "automated",
        TargetType:     "upload",
        TargetID:       event.UploadID,
        TargetFamilyID: &event.FamilyID,
        FlagType:       "explicit_content",
        Confidence:     &maxConfidence,
        Labels:         labelsJSON,
        AutoRejected:   true,
    }); err != nil {
        return fmt.Errorf("create content flag: %w", err)
    }

    // Notify user via notify — generic rejection message
    // "Your upload was not published because it violates our content guidelines."
    h.events.Publish(ctx, UploadAutoRejectedNotification{
        FamilyID: event.FamilyID,
        UploadID: event.UploadID,
    })

    return nil
}

type OnUploadFlagged struct {
    flagRepo ContentFlagRepository
}

func (h *OnUploadFlagged) Handle(ctx context.Context, event *media.UploadFlagged) error {
    // Determine flagType from labels — upgrade to suspected_underage_exploitation
    // if priority is critical (see §11.2.2)
    flagType := "explicit_content"
    if event.Priority != nil && *event.Priority == "critical" {
        flagType = "suspected_underage_exploitation"
    }

    labelsJSON, err := json.Marshal(event.Labels)
    if err != nil {
        return fmt.Errorf("marshal labels: %w", err)
    }

    // Create automated content flag from media moderation result
    if _, err := h.flagRepo.Create(ctx, CreateContentFlagRow{
        Source:         "automated",
        TargetType:     "upload",
        TargetID:       event.UploadID,
        TargetFamilyID: &event.FamilyID,
        FlagType:       flagType,
        Labels:         labelsJSON,
        AutoRejected:   false,
    }); err != nil {
        return fmt.Errorf("create content flag: %w", err)
    }
    return nil
}

type OnReviewCreated struct {
    safetyService SafetyService
}

func (h *OnReviewCreated) Handle(ctx context.Context, event *mkt.ReviewCreated) error {
    // Scan review text for policy violations
    if event.ReviewText != nil {
        result, err := h.safetyService.ScanText(ctx, *event.ReviewText)
        if err != nil {
            return fmt.Errorf("scan review text: %w", err)
        }
        if result.HasViolations {
            // Create content flag for admin review
            // Target: review, target_id: event.ReviewID
        }
    }
    return nil
}
```

### §16.5 Required Updates to Other Specs

| Spec | Update | Section |
|------|--------|---------|
| `00-core.md` | Add `is_platform_admin: bool` to `AuthContext` | §7.2 |
| `00-core.md` | Add `RequireAdmin` extractor | §13.3 |
| `00-core.md` | Add `AccountSuspended`, `AccountBanned` to `AppError` | §6.1 |
| `00-core.md` | Add safety access check to auth middleware (step 5.5) | §13.1 |
| `01-iam.md` | Add `is_platform_admin BOOLEAN NOT NULL DEFAULT false` to `iam_parents` | §3.1 |
| `01-iam.md` | Expose `revoke_sessions()` on `KratosAdapter` | §7 |
| `01-iam.md` | Move admin access from Phase 3 to Phase 1 | §17 |
| `09-media.md` | Add `last_csam_scanned_at TIMESTAMPTZ` to `media_uploads` | §3.2 |
| `09-media.md` | Add `rejected` status to upload state machine and CHECK constraint | §3.1, §3.2 |
| `09-media.md` | Add `UploadRejected` event, update `UploadFlagged` with priority | §16.3 |
| `09-media.md` | Add `auto_reject` and `priority` fields to `ModerationResult` | §8.3 |
| `09-media.md` | Update `ProcessUploadJob` §10.4 for auto-reject vs. flag routing | §10.4 |

---

## §17 Phase Scope

### Phase 1 (MVP)

- **API**: 6 user endpoints + 15 admin endpoints = 21 endpoints
- **Database**: 7 tables (`safety_reports`, `safety_content_flags`, `safety_mod_actions`,
  `safety_account_status`, `safety_appeals`, `safety_ncmec_reports`, `safety_bot_signals`)
- **Adapters**: Thorn Safer (CSAM), AWS Rekognition (moderation), SafetyScanBridge (media:: port)
- **CSAM pipeline**: Full end-to-end (detect → quarantine → NCMEC → ban → session revoke),
  including admin CSAM escalation from review queue (§11.4.1)
- **Content moderation**: Nudity auto-rejection (§11.2.1), label routing table (§11.2.2),
  flag-for-review for suggestive/violence, ignored categories (drugs/hate/weapons)
- **Text scanning**: Keyword + regex matching (synchronous)
- **Account enforcement**: Suspend/ban/lift with Redis-cached auth middleware check
- **Reporting**: Full user reporting flow with priority queue
- **Appeals**: Full appeals workflow with different-admin constraint
- **Bot detection**: Signal recording + auto-suspend threshold
- **Admin dashboard**: Basic stats endpoint
- **Events published**: 5 (`ContentReported`, `ModerationActionTaken`, `AccountSuspended`,
  `AppealResolved`, `UploadAutoRejectedNotification`)
- **Events consumed**: 7 (`PostCreated`, `MessageSent`, `ReviewCreated`, `UploadQuarantined`,
  `UploadRejected`, `UploadFlagged`, `MessageReported`)
- **Jobs**: `CsamReportJob` (Critical), `CheckCsamHashUpdateJob` (Low, daily),
  `CsamRescanJob` (Low, event-driven — only when new hashes available)

### Phase 2

- **ML text scanning**: AWS Comprehend integration (replaces keyword matching)
- **Grooming detection**: ML-based messaging pattern analysis `[S§12.6]`
- **Parental controls**: Configurable content filtering per family `[S§12.6]`
- **Admin roles**: Granular sub-permissions (moderation-only, billing-support-only) `[S§3.1.5]`
- **Suspension expiry job**: `ExpireSuspensionsJob` for audit-complete expiry records
- **Word list management**: Admin API for keyword list CRUD
- **Report analytics**: Trends, patterns, moderator performance metrics
- **Bulk moderation**: Multi-select actions in admin dashboard

### Phase 3

- **Intelligence**: Cross-reference reports to identify repeat offenders, coordinated abuse
- **ML moderation**: Custom-trained models for platform-specific policy enforcement
- **Automated escalation**: ML-driven priority adjustments based on historical patterns
- **Federation**: Shared block lists with other child-safety platforms

---

## §18 Verification Checklist

### CSAM Pipeline

1. CSAM detection via Thorn Safer triggers immediate upload quarantine (irreversible)
2. NCMEC report is filed via `CsamReportJob` on Critical queue (< 30s target)
3. Evidence is preserved in separate S3 path (never deleted)
4. Offending account is immediately and permanently banned
5. All sessions are revoked via Kratos
6. No notification is sent to the offending user
7. CSAM bans are not appealable
8. `CsamRescanJob` runs only when Thorn signals new hashes are available (not on a fixed daily schedule)

### Content Moderation & Label Routing

9. All user-generated text is scanned via `scan_text()` before persistence (including creator-authored question text, answer text, quiz/sequence titles and descriptions)
10. Media uploads are scanned via `SafetyScanAdapter` during `ProcessUploadJob` (including media attachments on `learn_questions`)
11. Nudity labels above threshold → upload auto-rejected (not just flagged) `[§11.2.1]`
12. Suspected underage + sexual labels → critical priority flag `[§11.2.2]`
13. Drugs/tobacco/alcohol, hate symbols, weapons → ignored (educational content) `[§11.2.2]`
14. Suggestive/violence labels → flagged for review at normal priority `[§11.2.2]`
15. Auto-rejected upload generates generic user notification; CSAM quarantine does not
16. Community reports create both `safety_reports` and `safety_content_flags` records
17. Report priority is set based on category (csam_child_safety → critical)
18. Admin review queue is sorted by priority then age
19. Reporter receives acknowledgment on submission and outcome notification on resolution

### Moderation Actions

20. All moderation actions create immutable `safety_mod_actions` records
21. `safety_mod_actions` table is append-only (no UPDATE, no DELETE)
22. Content removal, warning, suspension, ban, and CSAM escalation actions are available
23. Account suspensions have configurable duration (1-365 days)
24. Account bans are permanent (reversible only via appeal, except CSAM)

### CSAM Escalation from Review Queue

25. Admin CSAM escalation triggers full §10 pipeline (evidence → NCMEC → ban → revoke)
26. `safety_ncmec_reports` accepts NULL `csam_hash` for human-identified CSAM
27. Escalated content transitions to `quarantined` status (from `flagged` or `rejected`)
28. `escalate_to_csam` action type creates distinct audit trail entry

### Account Enforcement

29. Auth middleware checks account status via Redis cache (60s TTL)
30. Cache is explicitly invalidated when account status changes
31. Suspended accounts receive 403 with `account_suspended` code
32. Banned accounts receive 403 with `account_banned` code
33. Suspension expiry is handled lazily in `check_account_access()`

### Appeals

34. Users can submit one appeal per moderation action
35. CSAM bans are not appealable (validation enforced)
36. Rejected uploads are appealable; quarantined (CSAM) uploads are not
37. Appeal reviewer MUST be different from original action admin
38. Granted appeals reverse the original action (lift suspension, restore content)
39. Both parties (user and admin) are notified of appeal outcome

### Bot Prevention

40. CAPTCHA is required on registration (delegated to `iam::`)
41. Bot signals are recorded from multiple domains (social, iam, core)
42. Auto-suspend triggers when signal threshold is exceeded within time window
43. Auto-suspensions are 24h pending manual review

### Family-Scoping & Privacy

44. User-facing report queries are family-scoped via `FamilyScope`
45. Admin queries bypass family scope (documented per `[CODING §2.4]`)
46. Internal error details are never exposed in API responses `[CODING §2.2]`
47. PII (email, names) is never logged `[CODING §5.2]`

### Cross-Domain

48. `SafetyScanAdapter` implementation correctly bridges to Thorn and Rekognition
49. `SafetyScanBridge` applies label routing table (§11.2.2) — not the adapter itself
50. All 7 consumed events have registered handlers in `main.go`
51. All 5 published events have documented consumers
52. `RequireAdmin` extractor is available in `00-core` for all admin endpoints

---

## §19 Domain Model (Aggregate Roots)

`safety::` has two aggregate roots that enforce domain invariants via private fields and
validated state transitions. `[ARCH §4.5]`

### §19.1 ModerationReport

The `ModerationReport` aggregate root enforces the report lifecycle state machine.

```go
// internal/safety/domain/moderation_report.go

type ReportStatus string

const (
    ReportStatusPending             ReportStatus = "pending"
    ReportStatusInReview            ReportStatus = "in_review"
    ReportStatusResolvedActionTaken ReportStatus = "resolved_action_taken"
    ReportStatusResolvedNoAction    ReportStatus = "resolved_no_action"
    ReportStatusDismissed           ReportStatus = "dismissed"
)

type ReportPriority string

const (
    ReportPriorityCritical ReportPriority = "critical"
    ReportPriorityHigh     ReportPriority = "high"
    ReportPriorityNormal   ReportPriority = "normal"
)

type ModerationReport struct {
    id               uuid.UUID
    reporterFamilyID uuid.UUID
    reporterParentID uuid.UUID
    targetType       string
    targetID         uuid.UUID
    targetFamilyID   *uuid.UUID
    category         string
    description      *string
    priority         ReportPriority
    status           ReportStatus
    assignedAdminID  *uuid.UUID
    resolvedAt       *time.Time
    createdAt        time.Time
}

// NewModerationReport creates a new report. Priority is derived from category.
func NewModerationReport(
    reporterFamilyID uuid.UUID,
    reporterParentID uuid.UUID,
    targetType string,
    targetID uuid.UUID,
    targetFamilyID *uuid.UUID,
    category string,
    description *string,
) *ModerationReport {
    var priority ReportPriority
    switch category {
    case "csam_child_safety":
        priority = ReportPriorityCritical
    case "harassment":
        priority = ReportPriorityHigh
    default:
        priority = ReportPriorityNormal
    }

    return &ModerationReport{
        id:               uuid.New(),
        reporterFamilyID: reporterFamilyID,
        reporterParentID: reporterParentID,
        targetType:       targetType,
        targetID:         targetID,
        targetFamilyID:   targetFamilyID,
        category:         category,
        description:      description,
        priority:         priority,
        status:           ReportStatusPending,
        createdAt:        time.Now().UTC(),
    }
}

// Assign assigns an admin for review. Transitions: Pending → InReview.
func (r *ModerationReport) Assign(adminID uuid.UUID) error {
    if r.status != ReportStatusPending && r.status != ReportStatusInReview {
        return ErrInvalidReportTransition
    }
    r.status = ReportStatusInReview
    r.assignedAdminID = &adminID
    return nil
}

// ResolveActionTaken resolves with action taken. Transitions: InReview → ResolvedActionTaken.
func (r *ModerationReport) ResolveActionTaken() error {
    if r.status != ReportStatusInReview {
        return ErrInvalidReportTransition
    }
    r.status = ReportStatusResolvedActionTaken
    now := time.Now().UTC()
    r.resolvedAt = &now
    return nil
}

// ResolveNoAction resolves with no action. Transitions: InReview → ResolvedNoAction.
func (r *ModerationReport) ResolveNoAction() error {
    if r.status != ReportStatusInReview {
        return ErrInvalidReportTransition
    }
    r.status = ReportStatusResolvedNoAction
    now := time.Now().UTC()
    r.resolvedAt = &now
    return nil
}

// Dismiss dismisses the report. Transitions: Pending|InReview → Dismissed.
func (r *ModerationReport) Dismiss() error {
    if r.status == ReportStatusResolvedActionTaken ||
        r.status == ReportStatusResolvedNoAction ||
        r.status == ReportStatusDismissed {
        return ErrInvalidReportTransition
    }
    r.status = ReportStatusDismissed
    now := time.Now().UTC()
    r.resolvedAt = &now
    return nil
}

// Getters for private fields
func (r *ModerationReport) ID() uuid.UUID           { return r.id }
func (r *ModerationReport) Status() ReportStatus     { return r.status }
func (r *ModerationReport) Priority() ReportPriority  { return r.priority }
func (r *ModerationReport) Category() string          { return r.category }
```

### §19.2 AccountModerationState

The `AccountModerationState` aggregate root enforces the account status state machine.

```go
// internal/safety/domain/account_state.go

type AccountModerationStatus string

const (
    AccountStatusActive    AccountModerationStatus = "active"
    AccountStatusSuspended AccountModerationStatus = "suspended"
    AccountStatusBanned    AccountModerationStatus = "banned"
)

type AccountModerationState struct {
    familyID            uuid.UUID
    status              AccountModerationStatus
    suspendedAt         *time.Time
    suspensionExpiresAt *time.Time
    suspensionReason    *string
    bannedAt            *time.Time
    banReason           *string
    lastActionID        *uuid.UUID
}

func NewAccountModerationState(familyID uuid.UUID) *AccountModerationState {
    return &AccountModerationState{
        familyID: familyID,
        status:   AccountStatusActive,
    }
}

// Suspend suspends the account. Valid from: Active.
func (s *AccountModerationState) Suspend(days int32, reason string, actionID uuid.UUID) (*AccountSuspendedEvent, error) {
    if s.status == AccountStatusBanned {
        return nil, ErrAccountBanned
    }
    now := time.Now().UTC()
    expiresAt := now.Add(time.Duration(days) * 24 * time.Hour)
    s.status = AccountStatusSuspended
    s.suspendedAt = &now
    s.suspensionExpiresAt = &expiresAt
    s.suspensionReason = &reason
    s.lastActionID = &actionID

    return &AccountSuspendedEvent{
        FamilyID:       s.familyID,
        SuspensionDays: days,
        ExpiresAt:      expiresAt,
    }, nil
}

// Ban permanently bans the account. Valid from: Active, Suspended.
func (s *AccountModerationState) Ban(reason string, actionID uuid.UUID) error {
    if s.status == AccountStatusBanned {
        return ErrAccountBanned
    }
    now := time.Now().UTC()
    s.status = AccountStatusBanned
    s.bannedAt = &now
    s.banReason = &reason
    s.suspensionExpiresAt = nil // clear any pending suspension
    s.lastActionID = &actionID
    return nil
}

// LiftSuspension lifts the suspension. Valid from: Suspended.
func (s *AccountModerationState) LiftSuspension(actionID uuid.UUID) error {
    if s.status != AccountStatusSuspended {
        return ErrInvalidActionType
    }
    s.status = AccountStatusActive
    s.suspendedAt = nil
    s.suspensionExpiresAt = nil
    s.suspensionReason = nil
    s.lastActionID = &actionID
    return nil
}

// CheckExpiry checks if suspension has expired (lazy expiry).
// Returns true if the suspension was expired and status was changed to active.
func (s *AccountModerationState) CheckExpiry() bool {
    if s.status == AccountStatusSuspended && s.suspensionExpiresAt != nil {
        if time.Now().UTC().After(*s.suspensionExpiresAt) {
            s.status = AccountStatusActive
            s.suspendedAt = nil
            s.suspensionExpiresAt = nil
            s.suspensionReason = nil
            return true // expired, now active
        }
    }
    return false
}

// Unban unbans via appeal. Valid from: Banned (non-CSAM only).
func (s *AccountModerationState) Unban(actionID uuid.UUID) error {
    if s.status != AccountStatusBanned {
        return ErrInvalidActionType
    }
    if s.banReason != nil && *s.banReason == "csam_violation" {
        return ErrCsamBanNotAppealable
    }
    s.status = AccountStatusActive
    s.bannedAt = nil
    s.banReason = nil
    s.lastActionID = &actionID
    return nil
}

// Getters
func (s *AccountModerationState) FamilyID() uuid.UUID             { return s.familyID }
func (s *AccountModerationState) Status() AccountModerationStatus  { return s.status }
func (s *AccountModerationState) IsRestricted() bool {
    return s.status != AccountStatusActive
}
```

---

## §20 Module Structure

```
internal/safety/
├── handler.go                    # Thin HTTP handlers (user + admin endpoints)
├── service.go                    # SafetyServiceImpl — business logic orchestration
├── repository.go                 # Repository implementations (7 repos)
├── models.go                     # Request/response types, DTOs, filter types, GORM models
├── ports.go                      # SafetyService interface, adapter interfaces
├── errors.go                     # Error sentinel vars, ToAppError mapping
├── events.go                     # ContentReported, ModerationActionTaken,
│                                 # AccountSuspendedEvent, AppealResolved
├── event_handlers.go             # OnPostCreated, OnUploadQuarantined, OnUploadFlagged,
│                                 # OnReviewCreated, OnMessageSent, OnMessageReported
├── jobs.go                       # CsamReportJob, CheckCsamHashUpdateJob, CsamRescanJob
├── text_scanner.go               # TextScanner — keyword + regex matching
├── domain/
│   ├── moderation_report.go      # ModerationReport aggregate root
│   └── account_state.go          # AccountModerationState aggregate root
└── adapters/
    ├── thorn.go                  # ThornAdapter impl — Thorn Safer API client (net/http)
    ├── rekognition.go            # RekognitionAdapter impl — AWS Rekognition client (aws-sdk-go-v2)
    └── scan_bridge.go            # SafetyScanBridge — media.SafetyScanAdapter impl
```

---

## §21 Addendum: Extended Abuse Prevention `[S§12, S§17.10]`

*Added to address spec gaps in abuse vectors beyond CSAM and content moderation.*

### §21.1 Abuse Vector Catalog

The safety domain owns detection and response for all abuse vectors beyond content
moderation (§11) and CSAM (§10). This addendum catalogs the remaining vectors and
specifies mitigation strategies.

| Vector | Description | Detection | Response |
|--------|-------------|-----------|----------|
| **Account takeover** | Attacker gains access to a family account | Impossible travel detection (concurrent sessions from distant geolocations); password change after unusual login pattern | Security alert email; temporary session lock pending re-authentication; admin notification |
| **Credential stuffing** | Automated login attempts with leaked credentials | Rate limiting on `/v1/auth/login` (10/min per IP, via 00-core middleware); Kratos failed login tracking | CAPTCHA escalation after 5 failures; IP-level block after 20 failures (24h cooldown) |
| **Social spam** | Mass friend requests, message spam, group post flooding | Behavioral signals (§13.2 existing): rapid posting, mass friend requests, high message volume | Auto-suspend pending review; rate limits on social actions |
| **Marketplace fraud** | Fake reviews, self-purchasing, price manipulation | Delegated to `mkt::` fraud signals `[07-mkt §23]` | `MktFraudSignalDetected` event → safety:: creates moderation report |
| **Payment fraud** | Stolen card usage for marketplace purchases | Delegated to Hyperswitch/Stripe fraud detection | Chargeback webhook → auto-suspend purchasing; admin review |
| **Group abuse** | Malicious group creation (hate groups, grooming) | Group name/description scanning via text scanner; community reports | Group takedown; creator suspension; escalation for child safety concerns |
| **Profile abuse** | Inappropriate profile content, impersonation | Profile image scanning via Rekognition; username screening | Content removal; profile suspension |

### §21.2 Account Takeover Detection

```go
// ImpossibleTravelDetector flags concurrent sessions from distant geolocations.
type ImpossibleTravelDetector struct {
    MaxSpeedKmh float64 // ~900 km/h (commercial airline speed)
}

// IsSuspicious checks if a new session is suspicious based on the user's recent sessions.
// Returns true if the new session location is impossibly distant from the
// most recent session location given the time elapsed.
func (d *ImpossibleTravelDetector) IsSuspicious(
    newSession *SessionGeoInfo,
    recentSessions []SessionGeoInfo,
) bool {
    // Uses city-level geolocation only (no GPS) [S§7.8]
    // City centers are approximate — allow generous margins
    for _, session := range recentSessions {
        timeHours := newSession.StartedAt.Sub(session.LastActiveAt).Minutes() / 60.0
        distanceKm := haversineDistance(
            session.CityLat, session.CityLon,
            newSession.CityLat, newSession.CityLon,
        )
        if timeHours > 0.0 && distanceKm/timeHours > d.MaxSpeedKmh {
            return true
        }
    }
    return false
}
```

**Flow**: On new session creation → check against recent sessions → if suspicious →
publish `SuspiciousSessionDetected` event → `notify::` sends security alert email →
`admin_audit_log` records the event.

Note: City-level coordinates are used ONLY for this calculation and are derived from IP
geolocation at session creation time. They are NOT stored as user data and are NOT
associated with the user's profile. `[S§7.8]`

### §21.3 Credential Stuffing Protection

Credential stuffing protection is layered:

1. **Rate limiting** (00-core middleware): 10 login attempts per minute per IP `[ARCH §10.6]`
2. **CAPTCHA escalation**: After 5 failed login attempts for an email, subsequent attempts
   require CAPTCHA verification (delegated to `iam::` registration CAPTCHA infrastructure)
3. **IP blocking**: After 20 failed attempts from a single IP in 24 hours, the IP is
   temporarily blocked (24-hour cooldown via Redis)
4. **Breach detection** (Phase 2): Integrate with Have I Been Pwned API to warn users
   if their email appears in known breaches

```go
// RecordFailedLogin tracks failed login attempts for credential stuffing detection.
func RecordFailedLogin(ctx context.Context, rdb *redis.Client, email, ip string) (LoginAttemptStatus, error) {
    emailKey := fmt.Sprintf("login:fail:email:%s", email)
    ipKey := fmt.Sprintf("login:fail:ip:%s", ip)

    emailCount, err := rdb.Incr(ctx, emailKey).Result()
    if err != nil {
        return LoginAttemptStatusNormal, fmt.Errorf("incr email key: %w", err)
    }
    rdb.Expire(ctx, emailKey, time.Hour) // 1-hour window

    ipCount, err := rdb.Incr(ctx, ipKey).Result()
    if err != nil {
        return LoginAttemptStatusNormal, fmt.Errorf("incr ip key: %w", err)
    }
    rdb.Expire(ctx, ipKey, 24*time.Hour) // 24-hour window

    if ipCount >= 20 {
        return LoginAttemptStatusIPBlocked, nil
    }
    if emailCount >= 5 {
        return LoginAttemptStatusCaptchaRequired, nil
    }
    return LoginAttemptStatusNormal, nil
}
```

### §21.4 Payment Fraud Handling

Payment fraud detection is primarily handled by Hyperswitch/Stripe's built-in fraud
detection systems. Safety domain's role is *responding* to fraud signals:

- **Chargeback received** (Hyperswitch webhook): Auto-suspend purchasing for the family.
  Admin notified. Family can appeal.
- **Stripe Radar high-risk flag**: Payment blocked by Stripe. No platform action needed
  (payment simply fails). Logged for monitoring.
- **Repeated payment failures**: After 5 consecutive failures in 24 hours, flag account
  for review (possible stolen card testing).

### §21.5 Social Spam Signals

Extended behavioral signals beyond §13.2:

| Signal | Threshold | Action |
|--------|-----------|--------|
| Friend requests sent | > 20 per hour | Auto-rate-limit (1 per 5 min); flag for review |
| Messages sent to non-friends | > 10 per hour | Block messaging; flag for review |
| Group posts | > 10 per hour across groups | Rate-limit posting; flag for review |
| Profile views (scraping) | > 100 per hour | IP-level rate limit; CAPTCHA on next action |
| Event creation | > 5 per day | Block event creation; flag for review |

### §21.6 Integration with Admin and Moderation

All abuse signals from this addendum create entries in the existing moderation pipeline
(§11). Admin views them in the unified moderation queue (16-admin §9). Priority
assignment follows the existing model:

- **Critical** (< 24h response): Account takeover, payment fraud, child safety
- **High** (< 48h response): Credential stuffing attacks, social spam floods
- **Normal** (< 72h response): Marketplace fraud signals, profile abuse, group abuse
