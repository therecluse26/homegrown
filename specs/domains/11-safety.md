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
| **Module path** | `src/safety/` |
| **DB prefix** | `safety_` `[ARCH §5.1]` |
| **Complexity class** | Complex (has `domain/` subdirectory) — moderation state machine, CSAM handling pipeline `[ARCH §4.5]` |
| **CQRS** | No — read and write paths are straightforward; admin queries use the same model |
| **External adapters** | `src/safety/adapters/thorn.rs` (Thorn Safer — CSAM hash matching + NCMEC reporting) `[ARCH §2.13]`, `src/safety/adapters/rekognition.rs` (AWS Rekognition — content moderation labels) `[ARCH §2.13]` |
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
Background job scheduling → sidekiq-rs `[ARCH §12]`. CSAM hash matching API calls → Thorn Safer
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
| Automated screening of all UGC (posts, comments, messages, reviews, listings) | `[S§12.2]` | §11.1 (text scanning), §11.2 (media scanning via events), §11.2.1 (nudity auto-rejection), §11.2.2 (label routing) |
| Nudity/explicit content auto-rejected on upload | `[S§12.2]` | §11.2.1 (nudity auto-rejection policy) |
| Admin-confirmed novel CSAM triggers NCMEC pipeline | `[S§12.1]` | §11.4.1 (CSAM escalation from review queue) |
| Rekognition label routing (auto-reject / flag / ignore) | `[S§12.2]` | §11.2.2 (label routing decision table) |
| Community reporting system | `[S§12.3]` | §11.3 (user reporting flow) |
| Human review of flagged/reported content | `[S§12.3]` | §11.4 (admin review queue) |
| Moderator actions: remove, warn, suspend, ban | `[S§12.2]` | §12.1 (moderation actions) |
| Appeals mechanism reviewed by different moderator | `[S§12.2]` | §12.4 (appeals process) |
| Reportable content types: posts, comments, messages, profiles, groups, events, listings, reviews | `[S§12.3]` | §3.2 (`safety_reports.target_type` CHECK constraint) |
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
-- Target type values: post, comment, message, profile, group, event, listing, review
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
-- Migration: YYYYMMDD_000001_create_safety_tables.rs
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
                              'group', 'event', 'listing', 'review'
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
                              'group', 'event', 'listing', 'review', 'upload'
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
```rust
{
    "target_type": "post",                     // reportable entity type
    "target_id": "uuid",                       // ID of the reported entity
    "category": "harassment",                  // report category
    "description": "This post contains..."     // optional free-text
}
```
- **Response**: `201 Created` → `ReportResponse`
```rust
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
```rust
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
```rust
{
    "action_id": "uuid",                       // the moderation action being appealed
    "appeal_text": "I believe this was..."     // appeal justification
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
```rust
{
    "target_family_id": "uuid",
    "target_parent_id": "uuid",                // optional — for user-specific actions
    "action_type": "account_suspended",
    "reason": "Repeated harassment after warning",
    "report_id": "uuid",                       // optional — originating report
    "suspension_days": 7                       // required for suspensions
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
```rust
{
    "reason": "Repeated policy violations",
    "suspension_days": 7,
    "report_id": "uuid"                        // optional
}
```

#### POST /v1/admin/safety/accounts/:family_id/ban — Ban Account

Permanent ban. Irreversible (except via appeal). `[S§12.2]`

- **Auth**: `RequireAdmin`
- **Body**: `BanAccountCommand`
```rust
{
    "reason": "Severe policy violation",
    "report_id": "uuid"                        // optional
}
```

#### PATCH /v1/admin/safety/appeals/:id — Resolve Appeal

Resolve an appeal. The assigned admin MUST be different from the admin who took the
original action. `[S§12.2]`

- **Auth**: `RequireAdmin`
- **Body**: `ResolveAppealCommand`
```rust
{
    "status": "granted",                       // or "denied"
    "resolution_text": "Upon review, the suspension is lifted..."
}
```
- **Side effects**:
  - If granted and action was suspension: lifts suspension, creates `suspension_lifted` mod action
  - Publishes `AppealResolved` event → `notify::` (notifies the family)

---

## §5 Service Interface

The `SafetyService` trait defines all use cases exposed to handlers, other domains, and background
jobs. Defined in `src/safety/ports.rs`. `[CODING §8.2]`

```rust
#[async_trait]
pub trait SafetyService: Send + Sync {

    // ─── User-Facing Queries ────────────────────────────────────────────

    /// List the caller's submitted reports.
    async fn list_my_reports(
        &self,
        scope: &FamilyScope,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResponse<ReportResponse>, AppError>;

    /// Get a specific report (must belong to caller's family).
    async fn get_my_report(
        &self,
        scope: &FamilyScope,
        report_id: Uuid,
    ) -> Result<ReportResponse, AppError>;

    /// Get the caller's account moderation status.
    async fn get_account_status(
        &self,
        scope: &FamilyScope,
    ) -> Result<AccountStatusResponse, AppError>;

    /// Get a specific appeal (must belong to caller's family).
    async fn get_my_appeal(
        &self,
        scope: &FamilyScope,
        appeal_id: Uuid,
    ) -> Result<AppealResponse, AppError>;

    // ─── User-Facing Commands ───────────────────────────────────────────

    /// Submit a content report.
    async fn submit_report(
        &self,
        scope: &FamilyScope,
        auth: &AuthContext,
        cmd: CreateReportCommand,
    ) -> Result<ReportResponse, AppError>;

    /// Submit an appeal against a moderation action.
    async fn submit_appeal(
        &self,
        scope: &FamilyScope,
        cmd: CreateAppealCommand,
    ) -> Result<AppealResponse, AppError>;

    // ─── Admin Queries ──────────────────────────────────────────────────

    /// List reports (admin queue with filters).
    async fn admin_list_reports(
        &self,
        auth: &AuthContext,
        filter: ReportFilter,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResponse<AdminReportResponse>, AppError>;

    /// Get report detail (admin view — includes target content snapshot).
    async fn admin_get_report(
        &self,
        auth: &AuthContext,
        report_id: Uuid,
    ) -> Result<AdminReportResponse, AppError>;

    /// List content flags (admin review queue).
    async fn admin_list_flags(
        &self,
        auth: &AuthContext,
        filter: FlagFilter,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResponse<ContentFlagResponse>, AppError>;

    /// List moderation actions (audit log).
    async fn admin_list_actions(
        &self,
        auth: &AuthContext,
        filter: ActionFilter,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResponse<ModActionResponse>, AppError>;

    /// Get account moderation state (admin view).
    async fn admin_get_account(
        &self,
        auth: &AuthContext,
        family_id: Uuid,
    ) -> Result<AdminAccountStatusResponse, AppError>;

    /// List appeals (admin queue).
    async fn admin_list_appeals(
        &self,
        auth: &AuthContext,
        filter: AppealFilter,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResponse<AdminAppealResponse>, AppError>;

    /// Dashboard statistics.
    async fn admin_dashboard(
        &self,
        auth: &AuthContext,
    ) -> Result<DashboardStats, AppError>;

    // ─── Admin Commands ─────────────────────────────────────────────────

    /// Update a report (assign admin, change status).
    async fn admin_update_report(
        &self,
        auth: &AuthContext,
        report_id: Uuid,
        cmd: UpdateReportCommand,
    ) -> Result<AdminReportResponse, AppError>;

    /// Review a content flag.
    async fn admin_review_flag(
        &self,
        auth: &AuthContext,
        flag_id: Uuid,
        cmd: ReviewFlagCommand,
    ) -> Result<ContentFlagResponse, AppError>;

    /// Take a moderation action.
    async fn admin_take_action(
        &self,
        auth: &AuthContext,
        cmd: CreateModActionCommand,
    ) -> Result<ModActionResponse, AppError>;

    /// Suspend an account.
    async fn admin_suspend_account(
        &self,
        auth: &AuthContext,
        family_id: Uuid,
        cmd: SuspendAccountCommand,
    ) -> Result<AdminAccountStatusResponse, AppError>;

    /// Ban an account.
    async fn admin_ban_account(
        &self,
        auth: &AuthContext,
        family_id: Uuid,
        cmd: BanAccountCommand,
    ) -> Result<AdminAccountStatusResponse, AppError>;

    /// Lift a suspension.
    async fn admin_lift_suspension(
        &self,
        auth: &AuthContext,
        family_id: Uuid,
        cmd: LiftSuspensionCommand,
    ) -> Result<AdminAccountStatusResponse, AppError>;

    /// Resolve an appeal.
    async fn admin_resolve_appeal(
        &self,
        auth: &AuthContext,
        appeal_id: Uuid,
        cmd: ResolveAppealCommand,
    ) -> Result<AdminAppealResponse, AppError>;

    // ─── Internal (Cross-Domain) Methods ────────────────────────────────

    /// Check whether a family's account is allowed to access the platform.
    /// Called by auth middleware on every authenticated request.
    /// Returns Ok(()) if active, Err(AccountSuspended/AccountBanned) otherwise.
    /// Uses Redis cache with 60s TTL for performance.
    async fn check_account_access(
        &self,
        family_id: Uuid,
    ) -> Result<(), AppError>;

    /// Scan text content for policy violations.
    /// Called synchronously by social:: and mkt:: before persisting content.
    /// Phase 1: keyword + regex matching. Phase 2: ML (AWS Comprehend).
    async fn scan_text(
        &self,
        text: &str,
    ) -> Result<TextScanResult, AppError>;

    /// Record a bot behavioral signal.
    /// Called by domains that detect suspicious patterns.
    async fn record_bot_signal(
        &self,
        family_id: Uuid,
        parent_id: Uuid,
        signal: BotSignalType,
        details: serde_json::Value,
    ) -> Result<(), AppError>;

    /// Process a CSAM detection from media:: pipeline.
    /// Quarantines upload, files NCMEC report, bans account.
    /// Called by SafetyScanAdapter::report_csam() implementation.
    async fn handle_csam_detection(
        &self,
        upload_id: Uuid,
        family_id: Uuid,
        scan_result: &CsamScanResult,
    ) -> Result<(), AppError>;

    /// Escalate flagged/rejected content to CSAM (admin action). [§11.4.1]
    /// Marks the flag as reviewed, then delegates to handle_csam_detection()
    /// for the full §10 pipeline (evidence → NCMEC → ban → session revoke).
    async fn admin_escalate_to_csam(
        &self,
        auth: &AuthContext,
        flag_id: Uuid,
        cmd: EscalateCsamCommand,
    ) -> Result<(), AppError>;
}
```

### §5.1 Service Implementation

```rust
pub struct SafetyServiceImpl {
    report_repo: Arc<dyn ReportRepository>,
    flag_repo: Arc<dyn ContentFlagRepository>,
    action_repo: Arc<dyn ModActionRepository>,
    account_repo: Arc<dyn AccountStatusRepository>,
    appeal_repo: Arc<dyn AppealRepository>,
    ncmec_repo: Arc<dyn NcmecReportRepository>,
    bot_signal_repo: Arc<dyn BotSignalRepository>,
    thorn: Arc<dyn ThornAdapter>,
    rekognition: Arc<dyn RekognitionAdapter>,
    iam_service: Arc<dyn IamService>,
    redis: RedisPool,
    events: Arc<EventBus>,
    config: SafetyConfig,
}
```

---

## §6 Repository Interfaces

### §6.1 ReportRepository

```rust
#[async_trait]
pub trait ReportRepository: Send + Sync {
    async fn create(
        &self,
        scope: &FamilyScope,
        input: CreateReportRow,
    ) -> Result<Report, DbErr>;

    async fn find_by_id(
        &self,
        scope: &FamilyScope,
        report_id: Uuid,
    ) -> Result<Option<Report>, DbErr>;

    /// Admin: find by ID without family scoping.
    /// Unscoped — admin-only access [CODING §2.4].
    async fn find_by_id_unscoped(
        &self,
        report_id: Uuid,
    ) -> Result<Option<Report>, DbErr>;

    async fn list_by_reporter(
        &self,
        scope: &FamilyScope,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResult<Report>, DbErr>;

    /// Admin: list with filters (status, priority, category, assigned).
    async fn list_filtered(
        &self,
        filter: &ReportFilter,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResult<Report>, DbErr>;

    async fn update(
        &self,
        report_id: Uuid,
        updates: ReportUpdate,
    ) -> Result<Report, DbErr>;

    /// Check for duplicate reports within time window.
    async fn exists_recent(
        &self,
        scope: &FamilyScope,
        target_type: &str,
        target_id: Uuid,
        within_hours: u32,
    ) -> Result<bool, DbErr>;
}
```

### §6.2 ContentFlagRepository

```rust
#[async_trait]
pub trait ContentFlagRepository: Send + Sync {
    async fn create(
        &self,
        input: CreateContentFlagRow,
    ) -> Result<ContentFlag, DbErr>;

    async fn find_by_id(
        &self,
        flag_id: Uuid,
    ) -> Result<Option<ContentFlag>, DbErr>;

    /// List unreviewed flags (admin queue).
    async fn list_unreviewed(
        &self,
        filter: &FlagFilter,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResult<ContentFlag>, DbErr>;

    async fn mark_reviewed(
        &self,
        flag_id: Uuid,
        reviewed_by: Uuid,
        action_taken: bool,
    ) -> Result<ContentFlag, DbErr>;

    /// Find all flags for a target entity.
    async fn find_by_target(
        &self,
        target_type: &str,
        target_id: Uuid,
    ) -> Result<Vec<ContentFlag>, DbErr>;
}
```

### §6.3 ModActionRepository

```rust
/// Immutable audit trail — only insert and read operations.
#[async_trait]
pub trait ModActionRepository: Send + Sync {
    /// Insert a new moderation action. Never updates or deletes.
    async fn create(
        &self,
        input: CreateModActionRow,
    ) -> Result<ModAction, DbErr>;

    async fn find_by_id(
        &self,
        action_id: Uuid,
    ) -> Result<Option<ModAction>, DbErr>;

    /// List actions with filters (admin, target_family, action_type).
    async fn list_filtered(
        &self,
        filter: &ActionFilter,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResult<ModAction>, DbErr>;

    /// List all actions against a specific family.
    async fn list_by_target_family(
        &self,
        family_id: Uuid,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResult<ModAction>, DbErr>;
}
```

### §6.4 AccountStatusRepository

```rust
#[async_trait]
pub trait AccountStatusRepository: Send + Sync {
    /// Get or create account status (default: active).
    async fn get_or_create(
        &self,
        family_id: Uuid,
    ) -> Result<AccountStatus, DbErr>;

    async fn update(
        &self,
        family_id: Uuid,
        updates: AccountStatusUpdate,
    ) -> Result<AccountStatus, DbErr>;

    /// Find all restricted accounts (for cache warming).
    async fn find_restricted(
        &self,
    ) -> Result<Vec<AccountStatus>, DbErr>;
}
```

### §6.5 AppealRepository

```rust
#[async_trait]
pub trait AppealRepository: Send + Sync {
    async fn create(
        &self,
        scope: &FamilyScope,
        input: CreateAppealRow,
    ) -> Result<Appeal, DbErr>;

    async fn find_by_id(
        &self,
        scope: &FamilyScope,
        appeal_id: Uuid,
    ) -> Result<Option<Appeal>, DbErr>;

    async fn find_by_id_unscoped(
        &self,
        appeal_id: Uuid,
    ) -> Result<Option<Appeal>, DbErr>;

    async fn list_pending(
        &self,
        filter: &AppealFilter,
        pagination: &PaginationParams,
    ) -> Result<PaginatedResult<Appeal>, DbErr>;

    async fn update(
        &self,
        appeal_id: Uuid,
        updates: AppealUpdate,
    ) -> Result<Appeal, DbErr>;
}
```

### §6.6 NcmecReportRepository

```rust
/// Append-only — legal evidence. Only insert and read.
#[async_trait]
pub trait NcmecReportRepository: Send + Sync {
    async fn create(
        &self,
        input: CreateNcmecReportRow,
    ) -> Result<NcmecReport, DbErr>;

    async fn update_status(
        &self,
        report_id: Uuid,
        status: &str,
        ncmec_report_id: Option<&str>,
        error: Option<&str>,
    ) -> Result<NcmecReport, DbErr>;

    async fn find_pending(
        &self,
    ) -> Result<Vec<NcmecReport>, DbErr>;
}
```

### §6.7 BotSignalRepository

```rust
#[async_trait]
pub trait BotSignalRepository: Send + Sync {
    async fn create(
        &self,
        input: CreateBotSignalRow,
    ) -> Result<BotSignal, DbErr>;

    /// Count signals for a parent within a time window.
    async fn count_recent(
        &self,
        parent_id: Uuid,
        within_minutes: u32,
    ) -> Result<i64, DbErr>;
}
```

---

## §7 Adapter Interfaces

### §7.1 ThornAdapter (CSAM Detection + NCMEC Reporting)

Wraps the Thorn Safer API for PhotoDNA hash matching and automated NCMEC CyberTipline
report submission. `[ARCH §2.13, S§12.1]`

Adapter file: `src/safety/adapters/thorn.rs`

```rust
/// Thorn Safer adapter for CSAM detection and NCMEC reporting.
///
/// Thorn Safer provides:
/// 1. PhotoDNA hash matching against NCMEC's hash database
/// 2. Automated NCMEC CyberTipline report submission
///
/// [ARCH §2.13, S§12.1, 18 U.S.C. § 2258A]
#[async_trait]
pub trait ThornAdapter: Send + Sync {

    /// Scan an image/video for CSAM using PhotoDNA hash matching.
    ///
    /// Reads the file from S3 (via storage_key), computes PhotoDNA hash,
    /// and checks against NCMEC's known CSAM hash database.
    async fn scan_csam(
        &self,
        storage_key: &str,
    ) -> Result<CsamScanResult, ThornError>;

    /// Submit a CyberTipline report to NCMEC.
    ///
    /// Required by federal law (18 U.S.C. § 2258A) when CSAM is detected.
    /// Includes: file hash, upload metadata, uploader info, evidence URL.
    async fn submit_ncmec_report(
        &self,
        report: NcmecReportPayload,
    ) -> Result<NcmecSubmissionResult, ThornError>;
}

/// Configuration for Thorn Safer adapter.
pub struct ThornConfig {
    pub api_url: String,          // THORN_API_URL
    pub api_key: String,          // THORN_API_KEY (secret)
}

#[derive(Debug, thiserror::Error)]
pub enum ThornError {
    #[error("Thorn API unavailable")]
    Unavailable,
    #[error("Thorn API error: {0}")]
    ApiError(String),
    #[error("NCMEC submission failed: {0}")]
    NcmecSubmissionFailed(String),
}
```

### §7.2 RekognitionAdapter (Content Moderation)

Wraps AWS Rekognition's `DetectModerationLabels` API for general content moderation
(explicit content, violence, etc. — NOT CSAM). `[ARCH §2.13, S§12.2]`

Adapter file: `src/safety/adapters/rekognition.rs`

```rust
/// AWS Rekognition adapter for content moderation.
///
/// Thin wrapper around DetectModerationLabels. Returns ALL raw labels from
/// Rekognition without filtering — including categories the platform ignores
/// (drugs, hate symbols, weapons). Label routing (which labels trigger
/// auto-reject vs. flag vs. ignore) is handled by SafetyScanBridge (§11.2.2).
///
/// Rekognition returns labels for:
/// - Explicit/suggestive content → routed by bridge (auto-reject or flag)
/// - Violence/graphic content → routed by bridge (flag for review)
/// - Drugs/tobacco/alcohol → ignored by bridge (educational content)
/// - Hate symbols → ignored by bridge (educational content)
/// - Weapons → ignored by bridge (educational content)
///
/// Does NOT handle CSAM detection — that is ThornAdapter's responsibility.
/// [ARCH §2.13, S§12.2]
#[async_trait]
pub trait RekognitionAdapter: Send + Sync {

    /// Detect moderation labels in an image.
    ///
    /// Returns labels with confidence scores. The caller decides the
    /// threshold for flagging (configurable via SafetyConfig).
    async fn detect_moderation_labels(
        &self,
        storage_key: &str,
    ) -> Result<ModerationResult, RekognitionError>;
}

/// Configuration for Rekognition adapter.
pub struct RekognitionConfig {
    pub region: String,            // AWS_REGION
    pub min_confidence: f64,       // minimum confidence threshold (default: 70.0)
}

#[derive(Debug, thiserror::Error)]
pub enum RekognitionError {
    #[error("Rekognition unavailable")]
    Unavailable,
    #[error("Rekognition error: {0}")]
    ApiError(String),
}
```

### §7.3 SafetyScanAdapter Implementation

`safety::` provides the `SafetyScanAdapter` implementation consumed by `media::` during
the `ProcessUploadJob` pipeline. This adapter bridges `media::`'s port with `safety::`'s
Thorn and Rekognition adapters. `[09-media §7.2]`

```rust
/// Implementation of media::SafetyScanAdapter.
///
/// Bridges media::'s scan port with safety::'s Thorn and Rekognition adapters.
/// Lives in src/safety/adapters/scan_bridge.rs
pub struct SafetyScanBridge {
    thorn: Arc<dyn ThornAdapter>,
    rekognition: Arc<dyn RekognitionAdapter>,
    safety_service: Arc<dyn SafetyService>,
}

#[async_trait]
impl SafetyScanAdapter for SafetyScanBridge {
    async fn scan_csam(
        &self,
        storage_key: &str,
    ) -> Result<CsamScanResult, ScanError> {
        self.thorn.scan_csam(storage_key).await
            .map_err(|e| ScanError::Failed(e.to_string()))
    }

    async fn scan_moderation(
        &self,
        storage_key: &str,
    ) -> Result<ModerationResult, ScanError> {
        // 1. Get raw labels from Rekognition (all categories, unfiltered)
        let raw_result = self.rekognition.detect_moderation_labels(storage_key).await
            .map_err(|e| ScanError::Failed(e.to_string()))?;

        // 2. Apply platform's label routing table (§11.2.2)
        //    - Nudity/explicit → auto_reject = true
        //    - Suggestive/violence → has_violations = true, priority set
        //    - Drugs/hate/weapons → discarded (ignored categories)
        //    - Underage + sexual → critical priority
        Ok(apply_label_routing(raw_result.labels, &self.config))
    }

    async fn report_csam(
        &self,
        upload_id: Uuid,
        scan_result: &CsamScanResult,
    ) -> Result<(), ScanError> {
        // Delegates to SafetyService::handle_csam_detection()
        // which handles NCMEC filing, evidence preservation, and account ban.
        // family_id is looked up from the upload record.
        // This method is called from media::ProcessUploadJob when CSAM is detected.
        todo!("Implementation delegates to safety_service.handle_csam_detection()")
    }
}
```

---

## §8 Models (DTOs)

### §8.1 Request Types

```rust
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateReportCommand {
    pub target_type: String,
    pub target_id: Uuid,
    #[validate(length(max = 2000))]
    pub description: Option<String>,
    pub category: String,
}

#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateAppealCommand {
    pub action_id: Uuid,
    #[validate(length(min = 10, max = 5000))]
    pub appeal_text: String,
}

#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateModActionCommand {
    pub target_family_id: Uuid,
    pub target_parent_id: Option<Uuid>,
    pub action_type: String,
    #[validate(length(min = 5, max = 2000))]
    pub reason: String,
    pub report_id: Option<Uuid>,
    pub suspension_days: Option<i32>,
}

#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct SuspendAccountCommand {
    #[validate(length(min = 5, max = 2000))]
    pub reason: String,
    #[validate(range(min = 1, max = 365))]
    pub suspension_days: i32,
    pub report_id: Option<Uuid>,
}

#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct BanAccountCommand {
    #[validate(length(min = 5, max = 2000))]
    pub reason: String,
    pub report_id: Option<Uuid>,
}

#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct LiftSuspensionCommand {
    #[validate(length(min = 5, max = 2000))]
    pub reason: String,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct UpdateReportCommand {
    pub assigned_admin_id: Option<Uuid>,
    pub status: Option<String>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct ReviewFlagCommand {
    pub action_taken: bool,
}

#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct ResolveAppealCommand {
    pub status: String,       // "granted" or "denied"
    #[validate(length(min = 5, max = 2000))]
    pub resolution_text: String,
}

/// Request to escalate flagged content to CSAM. [§11.4.1]
/// Triggers the full §10 CSAM pipeline (evidence → NCMEC → ban → session revoke).
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct EscalateCsamCommand {
    #[validate(length(min = 5, max = 2000))]
    pub admin_notes: String,  // required justification for audit trail
}
```

### §8.2 Response Types

```rust
#[derive(Debug, Serialize, ToSchema)]
pub struct ReportResponse {
    pub id: Uuid,
    pub target_type: String,
    pub category: String,
    pub status: String,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct AdminReportResponse {
    pub id: Uuid,
    pub reporter_family_id: Uuid,
    pub target_type: String,
    pub target_id: Uuid,
    pub target_family_id: Option<Uuid>,
    pub category: String,
    pub description: Option<String>,
    pub priority: String,
    pub status: String,
    pub assigned_admin_id: Option<Uuid>,
    pub resolved_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ContentFlagResponse {
    pub id: Uuid,
    pub source: String,
    pub target_type: String,
    pub target_id: Uuid,
    pub flag_type: String,
    pub confidence: Option<f64>,
    pub labels: Option<serde_json::Value>,
    pub reviewed: bool,
    pub reviewed_by: Option<Uuid>,
    pub action_taken: Option<bool>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ModActionResponse {
    pub id: Uuid,
    pub admin_id: Uuid,
    pub target_family_id: Uuid,
    pub target_parent_id: Option<Uuid>,
    pub action_type: String,
    pub reason: String,
    pub report_id: Option<Uuid>,
    pub suspension_days: Option<i32>,
    pub suspension_expires_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct AccountStatusResponse {
    pub status: String,
    pub suspended_at: Option<DateTime<Utc>>,
    pub suspension_expires_at: Option<DateTime<Utc>>,
    pub suspension_reason: Option<String>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct AdminAccountStatusResponse {
    pub family_id: Uuid,
    pub status: String,
    pub suspended_at: Option<DateTime<Utc>>,
    pub suspension_expires_at: Option<DateTime<Utc>>,
    pub suspension_reason: Option<String>,
    pub banned_at: Option<DateTime<Utc>>,
    pub ban_reason: Option<String>,
    pub action_history: Vec<ModActionResponse>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct AppealResponse {
    pub id: Uuid,
    pub action_id: Uuid,
    pub status: String,
    pub appeal_text: String,
    pub resolution_text: Option<String>,
    pub resolved_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct AdminAppealResponse {
    pub id: Uuid,
    pub family_id: Uuid,
    pub action_id: Uuid,
    pub original_action: ModActionResponse,
    pub appeal_text: String,
    pub status: String,
    pub assigned_admin_id: Option<Uuid>,
    pub resolution_text: Option<String>,
    pub resolved_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct DashboardStats {
    pub pending_reports: i64,
    pub critical_reports: i64,
    pub unreviewed_flags: i64,
    pub pending_appeals: i64,
    pub active_suspensions: i64,
    pub active_bans: i64,
    pub reports_last_24h: i64,
    pub actions_last_24h: i64,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct TextScanResult {
    pub has_violations: bool,
    pub matched_terms: Vec<String>,
    pub severity: String,          // "none", "low", "high", "critical"
}
```

### §8.3 Filter Types

```rust
#[derive(Debug, Deserialize)]
pub struct ReportFilter {
    pub status: Option<String>,
    pub priority: Option<String>,
    pub category: Option<String>,
    pub assigned_admin_id: Option<Uuid>,
}

#[derive(Debug, Deserialize)]
pub struct FlagFilter {
    pub reviewed: Option<bool>,
    pub flag_type: Option<String>,
    pub target_type: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct ActionFilter {
    pub admin_id: Option<Uuid>,
    pub target_family_id: Option<Uuid>,
    pub action_type: Option<String>,
}

#[derive(Debug, Deserialize)]
pub struct AppealFilter {
    pub status: Option<String>,
}
```

### §8.4 Internal / Adapter Types

```rust
/// CSAM scan result from Thorn Safer (shared with media::).
/// Defined authoritatively in media::models (09-media §8.3).
/// Re-exported here for convenience.
pub use crate::media::models::{CsamScanResult, ModerationResult, ModerationLabel};

/// NCMEC report payload for Thorn Safer submission.
pub struct NcmecReportPayload {
    pub upload_id: Uuid,
    pub csam_hash: Option<String>,
    pub confidence: Option<f64>,
    pub matched_database: Option<String>,
    pub evidence_storage_key: String,
    pub uploader_family_id: Uuid,
    pub uploader_parent_id: Uuid,
    pub upload_timestamp: DateTime<Utc>,
}

/// NCMEC submission result from Thorn Safer.
pub struct NcmecSubmissionResult {
    pub ncmec_report_id: String,
    pub submitted_at: DateTime<Utc>,
}

/// Bot signal type enum.
#[derive(Debug, Clone, Copy)]
pub enum BotSignalType {
    RapidPosting,
    MassFriendRequests,
    RepetitiveContent,
    SuspiciousRegistration,
    RateLimitExceeded,
}

/// Safety configuration.
pub struct SafetyConfig {
    pub rekognition_min_confidence: f64,      // default: 70.0
    pub nudity_auto_reject_labels: Vec<String>, // labels that trigger auto-rejection (§11.2.1)
                                              // default: ["Explicit Nudity", "Nudity",
                                              //           "Graphic Male Nudity", "Graphic Female Nudity"]
    pub bot_signal_threshold: i64,            // signals in window before auto-suspend (default: 5)
    pub bot_signal_window_minutes: u32,       // time window for signal counting (default: 60)
    pub text_scan_word_list_redis_key: String, // Redis key for keyword list
    pub account_status_cache_ttl_seconds: u64, // Redis cache TTL (default: 60)
}
```

---

## §9 RequireAdmin Extractor (Deep-Dive 1)

### §9.1 Design

`RequireAdmin` is a new Axum extractor defined in `00-core` (`src/middleware/extractors.rs`)
that verifies the authenticated user has platform administrator privileges. It is backed by a
new `is_platform_admin` boolean field on `iam_parents` and a corresponding field in
`AuthContext`. `[S§3.1.5]`

This pulls admin access from Phase 3 to Phase 1 because safety:: — which is in the Phase 1
critical path — requires admin endpoints for the moderation dashboard.

### §9.2 Required Changes to 00-core and 01-iam

**00-core changes** (specified in updated `00-core.md` alongside this spec):

1. Add `is_platform_admin: bool` field to `AuthContext` (§7.2)
2. Add `RequireAdmin` extractor to `src/middleware/extractors.rs` (§13.3)
3. Add `AccountSuspended` and `AccountBanned` variants to `AppError` (§6.1)
4. Add safety access check to auth middleware (§13.1, step 5.5)

**01-iam changes** (specified in updated `01-iam.md` alongside this spec):

1. Add `is_platform_admin BOOLEAN NOT NULL DEFAULT false` column to `iam_parents` (§3.1)
2. Populate `is_platform_admin` in auth middleware DB lookup
3. Expose `revoke_sessions(kratos_identity_id)` on `KratosAdapter` trait

### §9.3 Extractor Definition (in 00-core §13.3)

```rust
/// Extracts AuthContext and verifies the user is a platform administrator.
/// Returns 403 Forbidden if the user is not an admin. [S§3.1.5]
pub struct RequireAdmin(pub AuthContext);

#[axum::async_trait]
impl<S> FromRequestParts<S> for RequireAdmin
where
    S: Send + Sync,
{
    type Rejection = AppError;

    async fn from_request_parts(
        parts: &mut Parts,
        _state: &S,
    ) -> Result<Self, Self::Rejection> {
        let auth = parts
            .extensions
            .get::<AuthContext>()
            .cloned()
            .ok_or(AppError::Unauthorized)?;

        if !auth.is_platform_admin {
            return Err(AppError::Forbidden);
        }

        Ok(RequireAdmin(auth))
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

```rust
/// Background job: submit CSAM report to NCMEC via Thorn Safer.
/// Critical queue — target processing within 30 seconds. [ARCH §12.2]
pub struct CsamReportJob {
    pub ncmec_report_id: Uuid,
}

impl CsamReportJob {
    async fn perform(&self, state: &AppState) -> Result<(), JobError> {
        let report = state.ncmec_repo.find_by_id(self.ncmec_report_id).await?;

        let result = state.thorn.submit_ncmec_report(NcmecReportPayload {
            upload_id: report.upload_id,
            csam_hash: report.csam_hash.clone(),
            confidence: report.confidence,
            matched_database: report.matched_database.clone(),
            evidence_storage_key: report.evidence_storage_key.clone(),
            uploader_family_id: report.family_id,
            uploader_parent_id: report.parent_id,
            upload_timestamp: report.created_at,
        }).await;

        match result {
            Ok(submission) => {
                state.ncmec_repo.update_status(
                    self.ncmec_report_id,
                    "submitted",
                    Some(&submission.ncmec_report_id),
                    None,
                ).await?;
            }
            Err(err) => {
                tracing::error!(
                    ncmec_report_id = %self.ncmec_report_id,
                    error = %err,
                    "NCMEC submission failed — will retry"
                );
                state.ncmec_repo.update_status(
                    self.ncmec_report_id,
                    "failed",
                    None,
                    Some(&err.to_string()),
                ).await?;
                return Err(JobError::Retryable(err.to_string()));
            }
        }

        Ok(())
    }
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

```rust
/// Synchronous text scanner. Called by social:: and mkt:: before persisting
/// user-generated text content (posts, comments, messages, reviews).
///
/// Phase 1: Redis-cached keyword list + regex patterns.
/// Phase 2: AWS Comprehend for ML-based detection.
pub struct TextScanner {
    redis: RedisPool,
    config: SafetyConfig,
}

impl TextScanner {
    /// Scan text against keyword list and regex patterns.
    ///
    /// Word lists are stored in Redis as a sorted set at the key
    /// specified by SafetyConfig::text_scan_word_list_redis_key.
    /// Lists are loaded at startup and refreshed via admin API.
    ///
    /// Returns TextScanResult with severity:
    /// - "none": no matches
    /// - "low": soft matches (may be false positives) — log only
    /// - "high": definite policy violations — block + flag
    /// - "critical": child safety terms — block + flag + escalate
    pub async fn scan(&self, text: &str) -> Result<TextScanResult, AppError> {
        // 1. Normalize text (lowercase, strip diacritics, collapse whitespace)
        // 2. Check against keyword list (O(n) scan, n = term count)
        // 3. Check against regex patterns (compiled at startup)
        // 4. Return aggregated result with matched terms
        todo!()
    }
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

```rust
// Pseudocode — actual implementation follows Rust patterns
fn apply_label_routing(
    raw_labels: Vec<ModerationLabel>,
    config: &SafetyConfig,
) -> ModerationResult {
    let mut auto_reject = false;
    let mut has_violations = false;
    let mut priority: Option<String> = None;
    let mut kept_labels = Vec::new();

    let has_underage_indicator = raw_labels.iter().any(|l|
        is_underage_label(&l.name) && l.confidence >= 50.0
    );

    for label in &raw_labels {
        if is_ignored_category(&label.name) {
            continue; // Drugs, hate symbols, weapons — skip entirely
        }

        if config.nudity_auto_reject_labels.contains(&label.name)
            && label.confidence >= config.rekognition_min_confidence
        {
            auto_reject = true;
            has_violations = true;
            kept_labels.push(label.clone());
        } else if is_suggestive(&label.name)
            && label.confidence >= 80.0
        {
            has_violations = true;
            kept_labels.push(label.clone());
            // Upgrade priority if underage indicators co-occur
            if has_underage_indicator {
                priority = Some("critical".into());
            } else if priority.is_none() {
                priority = Some("normal".into());
            }
        } else if is_violence(&label.name)
            && label.confidence >= config.rekognition_min_confidence
        {
            has_violations = true;
            kept_labels.push(label.clone());
            if priority.is_none() {
                priority = Some("normal".into());
            }
        }
    }

    // Underage + any nudity label → critical even if auto-rejecting
    if has_underage_indicator && has_violations && !auto_reject {
        priority = Some("critical".into());
    }

    ModerationResult {
        has_violations,
        auto_reject,
        labels: kept_labels,
        priority: if auto_reject { None } else { priority },
    }
}
```

**Critical priority routing**: When suspected underage + sexual labels co-occur (and the
content is not auto-rejected), the resulting `safety_content_flags` record gets
`flag_type = 'suspected_underage_exploitation'` and is routed to the top of the admin review
queue at `critical` priority. This matches the CSAM report SLA: review within 24 hours.

**Priority field on ModerationResult**: `priority: Option<String>` — `None` for auto-reject
outcomes (rejected uploads don't enter the review queue), `Some("critical"|"high"|"normal")`
for flagged items. The `UploadFlagged` event carries this priority through to the
`safety_content_flags` record. `[09-media §16.3]`

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

```rust
/// Called by other domains when suspicious behavior is detected.
async fn record_bot_signal(
    &self,
    family_id: Uuid,
    parent_id: Uuid,
    signal: BotSignalType,
    details: serde_json::Value,
) -> Result<(), AppError> {
    // 1. Create bot signal record
    self.bot_signal_repo.create(CreateBotSignalRow {
        family_id,
        parent_id,
        signal_type: signal.as_str(),
        details,
    }).await?;

    // 2. Check threshold
    let count = self.bot_signal_repo.count_recent(
        parent_id,
        self.config.bot_signal_window_minutes,
    ).await?;

    if count >= self.config.bot_signal_threshold {
        // Auto-suspend pending review
        self.admin_suspend_account(
            &AuthContext::system(), // system-initiated
            family_id,
            SuspendAccountCommand {
                reason: "Automated suspension: bot-like behavior detected".into(),
                suspension_days: 1, // 24h pending manual review
                report_id: None,
            },
        ).await?;
    }

    Ok(())
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

```rust
#[derive(Debug, thiserror::Error)]
pub enum SafetyError {
    #[error("report not found")]
    ReportNotFound,

    #[error("content flag not found")]
    FlagNotFound,

    #[error("moderation action not found")]
    ActionNotFound,

    #[error("appeal not found")]
    AppealNotFound,

    #[error("account status not found")]
    AccountNotFound,

    #[error("duplicate report within 24 hours")]
    DuplicateReport,

    #[error("appeal already exists for this action")]
    AppealAlreadyExists,

    #[error("CSAM bans are not appealable")]
    CsamBanNotAppealable,

    #[error("appeal reviewer must differ from original action admin")]
    SameAdminAppeal,

    #[error("invalid action type")]
    InvalidActionType,

    #[error("invalid report status transition")]
    InvalidReportTransition,

    #[error("flag already reviewed — cannot escalate")]
    FlagAlreadyReviewed,

    #[error("account is suspended")]
    AccountSuspended,

    #[error("account is banned")]
    AccountBanned,

    #[error("Thorn adapter error")]
    ThornError(#[from] ThornError),

    #[error("Rekognition adapter error")]
    RekognitionError(#[from] RekognitionError),

    #[error("database error")]
    DatabaseError(#[from] sea_orm::DbErr),
}
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

```rust
impl From<SafetyError> for AppError {
    fn from(err: SafetyError) -> Self {
        match err {
            SafetyError::ReportNotFound
            | SafetyError::FlagNotFound
            | SafetyError::ActionNotFound
            | SafetyError::AppealNotFound
            | SafetyError::AccountNotFound => AppError::NotFound,

            SafetyError::DuplicateReport
            | SafetyError::AppealAlreadyExists => AppError::Conflict(err.to_string()),

            SafetyError::CsamBanNotAppealable
            | SafetyError::SameAdminAppeal
            | SafetyError::InvalidActionType
            | SafetyError::InvalidReportTransition
            | SafetyError::FlagAlreadyReviewed => AppError::Validation(err.to_string()),

            SafetyError::AccountSuspended => AppError::AccountSuspended,
            SafetyError::AccountBanned => AppError::AccountBanned,

            SafetyError::ThornError(e) => {
                tracing::error!(error = %e, "Thorn adapter error");
                AppError::Internal(anyhow::anyhow!("safety scan error"))
            }
            SafetyError::RekognitionError(e) => {
                tracing::error!(error = %e, "Rekognition adapter error");
                AppError::Internal(anyhow::anyhow!("safety scan error"))
            }
            SafetyError::DatabaseError(e) => AppError::Database(e),
        }
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

Defined in `src/safety/events.rs`. `[CODING §8.4]`

```rust
// src/safety/events.rs

use uuid::Uuid;
use chrono::{DateTime, Utc};

/// Published when a community report is submitted.
/// Consumed by notify:: (admin queue alert notification).
#[derive(Clone, Debug)]
pub struct ContentReported {
    pub report_id: Uuid,
    pub target_type: String,
    pub target_id: Uuid,
    pub category: String,
    pub priority: String,
}
impl DomainEvent for ContentReported {}

/// Published when a moderation action is taken (EXCEPT for CSAM cases).
/// Consumed by notify:: (notification to affected user),
/// social:: (hide removed content), mkt:: (hide removed listing/review).
#[derive(Clone, Debug)]
pub struct ModerationActionTaken {
    pub action_id: Uuid,
    pub action_type: String,
    pub target_family_id: Uuid,
    pub target_type: Option<String>,       // "post", "listing", etc. (for content actions)
    pub target_id: Option<Uuid>,
}
impl DomainEvent for ModerationActionTaken {}

/// Published when an account is suspended.
/// Consumed by notify:: (suspension notification to user).
#[derive(Clone, Debug)]
pub struct AccountSuspended {
    pub family_id: Uuid,
    pub suspension_days: i32,
    pub expires_at: DateTime<Utc>,
}
impl DomainEvent for AccountSuspended {}

/// Published when an appeal is resolved.
/// Consumed by notify:: (appeal outcome notification to user).
#[derive(Clone, Debug)]
pub struct AppealResolved {
    pub appeal_id: Uuid,
    pub family_id: Uuid,
    pub status: String,                    // "granted" or "denied"
}
impl DomainEvent for AppealResolved {}

/// Published when an upload is auto-rejected by content policy (§11.2.1).
/// Consumed by notify:: (generic rejection notification to uploader).
/// Message: "Your upload was not published because it violates our content guidelines."
/// Does NOT specify which policy was violated (prevents gaming).
#[derive(Clone, Debug)]
pub struct UploadAutoRejectedNotification {
    pub family_id: Uuid,
    pub upload_id: Uuid,
}
impl DomainEvent for UploadAutoRejectedNotification {}
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

```rust
// src/safety/event_handlers.rs

use crate::social::events::{PostCreated, MessageSent, MessageReported};
use crate::mkt::events::ReviewCreated;
use crate::media::events::{UploadQuarantined, UploadRejected, UploadFlagged};

pub struct OnPostCreated {
    safety_service: Arc<dyn SafetyService>,
}

#[async_trait]
impl DomainEventHandler<PostCreated> for OnPostCreated {
    async fn handle(&self, event: &PostCreated) -> Result<(), AppError> {
        // Scan post text content for policy violations
        if let Some(ref content) = event.content {
            let result = self.safety_service.scan_text(content).await?;
            if result.has_violations {
                // Create content flag for admin review
                // Target: post, target_id: event.post_id
            }
        }
        Ok(())
    }
}

pub struct OnUploadQuarantined {
    safety_service: Arc<dyn SafetyService>,
}

#[async_trait]
impl DomainEventHandler<UploadQuarantined> for OnUploadQuarantined {
    async fn handle(&self, event: &UploadQuarantined) -> Result<(), AppError> {
        // Initiate CSAM pipeline: evidence preservation → NCMEC report → ban
        self.safety_service.handle_csam_detection(
            event.upload_id,
            event.family_id,
            // CsamScanResult is retrieved from the media upload record
            &CsamScanResult { is_csam: true, hash: None, confidence: None, matched_database: None },
        ).await
    }
}

pub struct OnUploadRejected {
    flag_repo: Arc<dyn ContentFlagRepository>,
    events: Arc<EventBus>,
}

#[async_trait]
impl DomainEventHandler<UploadRejected> for OnUploadRejected {
    async fn handle(&self, event: &UploadRejected) -> Result<(), AppError> {
        // Create auto-rejected content flag [§11.2.1]
        let max_confidence = event.labels.iter()
            .map(|l| l.confidence)
            .fold(0.0_f64, f64::max);

        self.flag_repo.create(CreateContentFlagRow {
            source: "automated".into(),
            target_type: "upload".into(),
            target_id: event.upload_id,
            target_family_id: Some(event.family_id),
            flag_type: "explicit_content".into(),
            confidence: Some(max_confidence),
            labels: Some(serde_json::to_value(&event.labels)?),
            report_id: None,
            auto_rejected: true,
        }).await?;

        // Notify user via notify:: — generic rejection message
        // "Your upload was not published because it violates our content guidelines."
        self.events.publish(UploadAutoRejectedNotification {
            family_id: event.family_id,
            upload_id: event.upload_id,
        }).await;

        Ok(())
    }
}

pub struct OnUploadFlagged {
    flag_repo: Arc<dyn ContentFlagRepository>,
}

#[async_trait]
impl DomainEventHandler<UploadFlagged> for OnUploadFlagged {
    async fn handle(&self, event: &UploadFlagged) -> Result<(), AppError> {
        // Determine flag_type from labels — upgrade to suspected_underage_exploitation
        // if priority is critical (see §11.2.2)
        let flag_type = if event.priority.as_deref() == Some("critical") {
            "suspected_underage_exploitation"
        } else {
            "explicit_content"
        };

        // Create automated content flag from media moderation result
        self.flag_repo.create(CreateContentFlagRow {
            source: "automated".into(),
            target_type: "upload".into(),
            target_id: event.upload_id,
            target_family_id: Some(event.family_id),
            flag_type: flag_type.into(),
            confidence: None,
            labels: Some(serde_json::to_value(&event.labels)?),
            report_id: None,
            auto_rejected: false,
        }).await?;
        Ok(())
    }
}

pub struct OnReviewCreated {
    safety_service: Arc<dyn SafetyService>,
}

#[async_trait]
impl DomainEventHandler<ReviewCreated> for OnReviewCreated {
    async fn handle(&self, event: &ReviewCreated) -> Result<(), AppError> {
        // Scan review text for policy violations
        if let Some(ref text) = event.review_text {
            let result = self.safety_service.scan_text(text).await?;
            if result.has_violations {
                // Create content flag for admin review
                // Target: review, target_id: event.review_id
            }
        }
        Ok(())
    }
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

9. All user-generated text is scanned via `scan_text()` before persistence
10. Media uploads are scanned via `SafetyScanAdapter` during `ProcessUploadJob`
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
50. All 7 consumed events have registered handlers in `main.rs`
51. All 5 published events have documented consumers
52. `RequireAdmin` extractor is available in `00-core` for all admin endpoints

---

## §19 Domain Model (Aggregate Roots)

`safety::` has two aggregate roots that enforce domain invariants via private fields and
validated state transitions. `[ARCH §4.5]`

### §19.1 ModerationReport

The `ModerationReport` aggregate root enforces the report lifecycle state machine.

```rust
// src/safety/domain/moderation_report.rs

#[derive(Debug)]
pub struct ModerationReport {
    id: Uuid,
    reporter_family_id: Uuid,
    reporter_parent_id: Uuid,
    target_type: String,
    target_id: Uuid,
    target_family_id: Option<Uuid>,
    category: String,
    description: Option<String>,
    priority: ReportPriority,
    status: ReportStatus,
    assigned_admin_id: Option<Uuid>,
    resolved_at: Option<DateTime<Utc>>,
    created_at: DateTime<Utc>,
}

#[derive(Debug, Clone, Copy, PartialEq)]
enum ReportStatus {
    Pending,
    InReview,
    ResolvedActionTaken,
    ResolvedNoAction,
    Dismissed,
}

#[derive(Debug, Clone, Copy, PartialEq)]
enum ReportPriority {
    Critical,
    High,
    Normal,
}

impl ModerationReport {
    /// Create a new report. Priority is derived from category.
    pub fn new(
        reporter_family_id: Uuid,
        reporter_parent_id: Uuid,
        target_type: String,
        target_id: Uuid,
        target_family_id: Option<Uuid>,
        category: String,
        description: Option<String>,
    ) -> Self {
        let priority = match category.as_str() {
            "csam_child_safety" => ReportPriority::Critical,
            "harassment" => ReportPriority::High,
            _ => ReportPriority::Normal,
        };

        Self {
            id: Uuid::new_v4(),
            reporter_family_id,
            reporter_parent_id,
            target_type,
            target_id,
            target_family_id,
            category,
            description,
            priority,
            status: ReportStatus::Pending,
            assigned_admin_id: None,
            resolved_at: None,
            created_at: Utc::now(),
        }
    }

    /// Assign an admin for review. Transitions: Pending → InReview.
    pub fn assign(&mut self, admin_id: Uuid) -> Result<(), SafetyError> {
        if self.status != ReportStatus::Pending && self.status != ReportStatus::InReview {
            return Err(SafetyError::InvalidReportTransition);
        }
        self.status = ReportStatus::InReview;
        self.assigned_admin_id = Some(admin_id);
        Ok(())
    }

    /// Resolve with action taken. Transitions: InReview → ResolvedActionTaken.
    pub fn resolve_action_taken(&mut self) -> Result<(), SafetyError> {
        if self.status != ReportStatus::InReview {
            return Err(SafetyError::InvalidReportTransition);
        }
        self.status = ReportStatus::ResolvedActionTaken;
        self.resolved_at = Some(Utc::now());
        Ok(())
    }

    /// Resolve with no action. Transitions: InReview → ResolvedNoAction.
    pub fn resolve_no_action(&mut self) -> Result<(), SafetyError> {
        if self.status != ReportStatus::InReview {
            return Err(SafetyError::InvalidReportTransition);
        }
        self.status = ReportStatus::ResolvedNoAction;
        self.resolved_at = Some(Utc::now());
        Ok(())
    }

    /// Dismiss. Transitions: Pending|InReview → Dismissed.
    pub fn dismiss(&mut self) -> Result<(), SafetyError> {
        if self.status == ReportStatus::ResolvedActionTaken
            || self.status == ReportStatus::ResolvedNoAction
            || self.status == ReportStatus::Dismissed
        {
            return Err(SafetyError::InvalidReportTransition);
        }
        self.status = ReportStatus::Dismissed;
        self.resolved_at = Some(Utc::now());
        Ok(())
    }

    // Getters for private fields
    pub fn id(&self) -> Uuid { self.id }
    pub fn status(&self) -> &ReportStatus { &self.status }
    pub fn priority(&self) -> &ReportPriority { &self.priority }
    pub fn category(&self) -> &str { &self.category }
}
```

### §19.2 AccountModerationState

The `AccountModerationState` aggregate root enforces the account status state machine.

```rust
// src/safety/domain/account_state.rs

#[derive(Debug)]
pub struct AccountModerationState {
    family_id: Uuid,
    status: AccountModerationStatus,
    suspended_at: Option<DateTime<Utc>>,
    suspension_expires_at: Option<DateTime<Utc>>,
    suspension_reason: Option<String>,
    banned_at: Option<DateTime<Utc>>,
    ban_reason: Option<String>,
    last_action_id: Option<Uuid>,
}

#[derive(Debug, Clone, Copy, PartialEq)]
enum AccountModerationStatus {
    Active,
    Suspended,
    Banned,
}

impl AccountModerationState {
    pub fn new(family_id: Uuid) -> Self {
        Self {
            family_id,
            status: AccountModerationStatus::Active,
            suspended_at: None,
            suspension_expires_at: None,
            suspension_reason: None,
            banned_at: None,
            ban_reason: None,
            last_action_id: None,
        }
    }

    /// Suspend the account. Valid from: Active.
    pub fn suspend(
        &mut self,
        days: i32,
        reason: String,
        action_id: Uuid,
    ) -> Result<AccountSuspended, SafetyError> {
        if self.status == AccountModerationStatus::Banned {
            return Err(SafetyError::AccountBanned);
        }
        let now = Utc::now();
        let expires_at = now + chrono::Duration::days(days as i64);
        self.status = AccountModerationStatus::Suspended;
        self.suspended_at = Some(now);
        self.suspension_expires_at = Some(expires_at);
        self.suspension_reason = Some(reason);
        self.last_action_id = Some(action_id);

        Ok(AccountSuspended {
            family_id: self.family_id,
            suspension_days: days,
            expires_at,
        })
    }

    /// Ban the account permanently. Valid from: Active, Suspended.
    pub fn ban(
        &mut self,
        reason: String,
        action_id: Uuid,
    ) -> Result<(), SafetyError> {
        if self.status == AccountModerationStatus::Banned {
            return Err(SafetyError::AccountBanned);
        }
        self.status = AccountModerationStatus::Banned;
        self.banned_at = Some(Utc::now());
        self.ban_reason = Some(reason);
        self.suspension_expires_at = None; // clear any pending suspension
        self.last_action_id = Some(action_id);
        Ok(())
    }

    /// Lift suspension. Valid from: Suspended.
    pub fn lift_suspension(
        &mut self,
        action_id: Uuid,
    ) -> Result<(), SafetyError> {
        if self.status != AccountModerationStatus::Suspended {
            return Err(SafetyError::InvalidActionType);
        }
        self.status = AccountModerationStatus::Active;
        self.suspended_at = None;
        self.suspension_expires_at = None;
        self.suspension_reason = None;
        self.last_action_id = Some(action_id);
        Ok(())
    }

    /// Check if suspension has expired (lazy expiry).
    pub fn check_expiry(&mut self) -> bool {
        if self.status == AccountModerationStatus::Suspended {
            if let Some(expires_at) = self.suspension_expires_at {
                if Utc::now() >= expires_at {
                    self.status = AccountModerationStatus::Active;
                    self.suspended_at = None;
                    self.suspension_expires_at = None;
                    self.suspension_reason = None;
                    return true; // expired, now active
                }
            }
        }
        false
    }

    /// Unban (via appeal). Valid from: Banned (non-CSAM only).
    pub fn unban(
        &mut self,
        action_id: Uuid,
    ) -> Result<(), SafetyError> {
        if self.status != AccountModerationStatus::Banned {
            return Err(SafetyError::InvalidActionType);
        }
        if self.ban_reason.as_deref() == Some("csam_violation") {
            return Err(SafetyError::CsamBanNotAppealable);
        }
        self.status = AccountModerationStatus::Active;
        self.banned_at = None;
        self.ban_reason = None;
        self.last_action_id = Some(action_id);
        Ok(())
    }

    // Getters
    pub fn family_id(&self) -> Uuid { self.family_id }
    pub fn status(&self) -> &AccountModerationStatus { &self.status }
    pub fn is_restricted(&self) -> bool {
        self.status != AccountModerationStatus::Active
    }
}
```

---

## §20 Module Structure

```
src/safety/
├── mod.rs                        # Module declarations, re-exports
├── handlers.rs                   # Thin HTTP handlers (user + admin endpoints)
├── service.rs                    # SafetyServiceImpl — business logic orchestration
├── repository.rs                 # Repository implementations (7 repos)
├── models.rs                     # Request/response types, DTOs, filter types
├── ports.rs                      # SafetyService trait, adapter traits
├── errors.rs                     # SafetyError enum, From<SafetyError> for AppError
├── events.rs                     # ContentReported, ModerationActionTaken,
│                                 # AccountSuspended, AppealResolved
├── event_handlers.rs             # OnPostCreated, OnUploadQuarantined, OnUploadFlagged,
│                                 # OnReviewCreated, OnMessageSent, OnMessageReported
├── jobs.rs                       # CsamReportJob, CheckCsamHashUpdateJob, CsamRescanJob
├── text_scanner.rs               # TextScanner — keyword + regex matching
├── domain/
│   ├── mod.rs                    # Domain model re-exports
│   ├── moderation_report.rs      # ModerationReport aggregate root
│   └── account_state.rs          # AccountModerationState aggregate root
├── adapters/
│   ├── mod.rs                    # Adapter re-exports
│   ├── thorn.rs                  # ThornAdapter impl — Thorn Safer API client
│   ├── rekognition.rs            # RekognitionAdapter impl — AWS Rekognition client
│   └── scan_bridge.rs            # SafetyScanBridge — media::SafetyScanAdapter impl
└── entities/                     # SeaORM-generated — NEVER hand-edit [CODING §4.2]
```
