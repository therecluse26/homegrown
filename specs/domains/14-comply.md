# Domain Spec 14 — Compliance & Reporting (comply::)

## §1 Overview

The Compliance & Reporting domain is the **state compliance configuration, attendance tracking,
assessment aggregation, and document generation engine** — it manages family compliance settings
(state selection, school year, schedule), daily attendance logging (manual + auto from learning
signals), assessment record aggregation, standardized test score entry, portfolio generation
(PDF), and transcript generation (PDF). It is a premium-only domain: all endpoints require the
`RequirePremium` extractor (402 for free-tier families). `[S§11, V§8, V§10]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/comply/` |
| **DB prefix** | `comply_` `[ARCH §5.1]` |
| **Complexity class** | Complex (has `domain/` subdirectory) — attendance thresholds, GPA calculation, state config rules, portfolio/transcript state machines `[ARCH §4.5]` |
| **CQRS** | Yes — write: attendance mark, assessment record / read: attendance summary, threshold check `[ARCH §4.7]` |
| **External adapter** | None — in-house PDF generation via `typst` crate; no external compliance vendor |
| **Key constraint** | All endpoints premium-only via `RequirePremium` (402 for free tier) `[S§15.2]`; every user-data query family-scoped via `FamilyScope` `[CODING §2.4, §2.5]`; state config cached from `discover::`, not duplicated |

**What comply:: owns**: Family compliance configuration (state, school year, schedule), custom
schedule definitions (4-day weeks, year-round schooling), daily attendance records per student,
assessment record aggregation, standardized test score storage, portfolio metadata and generation
pipeline (PDF), portfolio item selection and caching, transcript metadata and generation pipeline
(PDF, Phase 3), course definitions for transcripts (Phase 3), GPA calculation (Phase 3),
attendance threshold checking and pace monitoring, state compliance config cache
(`comply_state_configs`), domain events for generated documents and threshold warnings.

**What comply:: does NOT own**: State legal guide content and public requirements display (owned
by `discover::` — `disc_state_guides`) `[03-discover §3.2]`, learning activity data and tool
access (owned by `learn::`) `[06-learn §3.2]`, student profiles and family accounts (owned by
`iam::`) `[01-iam §3.2]`, subscription tier enforcement (owned by `billing::` via events +
`iam::` tier column) `[10-billing §10]`, notification delivery (owned by `notify::`), PDF file
storage and CDN delivery (owned by `media::`) `[09-media §5]`, search indexing (owned by
`search::`).

**What comply:: delegates**: Notification delivery → `notify::` (via domain events). State
requirements lookup → `discover::DiscoveryService` (service call for `StateGuideRequirements`).
Learning data for portfolio items → `learn::LearningService` (service call). Student/family
identity resolution → `iam::IamService` (service call). PDF file storage → `media::MediaService`
(upload + presigned URLs). Background job scheduling → sidekiq-rs `[ARCH §12]`.

---

## §2 Requirements Traceability

Every SPEC.md §11 requirement maps to a section in this document. Cross-references from
other spec sections are included where the compliance domain is involved.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| State-specific compliance configuration for all 50 states + DC | `[S§11.1]` | §3.1 (`comply_state_configs` — 51 rows), §4 (config endpoints) |
| Compliance config derived from same source as state legal guides | `[S§11.1]` | §9 (`SyncStateConfigsJob` — syncs from `discover::`) |
| Compliance features are premium only | `[S§11.1, S§15.2]` | §4 (all endpoints use `RequirePremium`) |
| Daily attendance marking per student | `[S§11.2]` | §3.4 (`comply_attendance`), §4 (attendance CRUD endpoints) |
| Attendance statuses: present (full), present (partial), absent, not-applicable | `[S§11.2]` | §3.1 (attendance status CHECK), §3.4 (`status` column) |
| Attendance summaries by date range meeting state day/hour requirements | `[S§11.2]` | §4 (attendance summary endpoint), §12 (threshold logic) |
| Custom schedule definitions (4-day weeks, year-round) | `[S§11.2]` | §3.3 (`comply_custom_schedules`), §4 (schedule endpoints) |
| Aggregate assessment data from learning tools into compliance-ready records | `[S§11.3]` | §3.5 (`comply_assessment_records`), §4 (assessment endpoints) |
| Assessment records filterable by student, subject, date range | `[S§11.3]` | §4 (assessment list endpoint with query params) |
| Standardized test score entry | `[S§11.3]` | §3.6 (`comply_standardized_tests`), §4 (test score endpoints) |
| Generate downloadable portfolios in PDF format | `[S§11.4]` | §3.7-§3.8, §4 (portfolio endpoints), §11 (generation pipeline) |
| Portfolios customizable: parents select which artifacts to include | `[S§11.4]` | §3.8 (`comply_portfolio_items`), §4 (add/remove items) |
| Portfolios organizable by subject, chronologically, or by student | `[S§11.4]` | §8 (`PortfolioOrganization` enum), §11 (PDF layout) |
| Portfolios include cover page with student and family information | `[S§11.4]` | §11 (cover page generation) |
| Portfolio generation is premium | `[S§11.4, S§15.2]` | §4 (all endpoints use `RequirePremium`) |
| Generate formal academic transcripts for grades 9-12 | `[S§11.5]` | §3.9-§3.10 (Phase 3), §4 (transcript endpoints), §13 |
| Transcripts include: student name, courses, grades, credit hours, cumulative GPA | `[S§11.5]` | §3.10 (`comply_courses`), §10 (GPA calculation), §13 |
| Transcripts follow standard high school transcript format | `[S§11.5]` | §13 (PDF format) |
| Multiple GPA calculation methods (4.0, weighted, unweighted) | `[S§11.5]` | §10 (GPA algorithm deep dive) |
| Transcript generation is premium | `[S§11.5, S§15.2]` | §4 (all endpoints use `RequirePremium`) |
| Learning records retained for account lifetime | `[S§11.6]` | §17 (cross-domain — `learn::` owns record retention) |
| Records exportable before account deletion | `[S§11.6]` | §17 (cross-domain — `learn::` data export `[06-learn §4]`) |
| Permanent, irreversible deletion after grace period | `[S§11.6]` | §5 (`handle_family_deletion_scheduled`) |
| Financial/legal records retained per legal requirements | `[S§11.6]` | §17 (cross-domain — `billing::` owns financial records) |
| Compliance data feed from learn:: | `[06-learn §18.1]` | §5 (`handle_activity_logged`), §17 |
| Premium tier gating via RequirePremium | `[S§15.2, 00-core §13.3]` | §4 (all endpoints) |
| Phase 1: no compliance reporting | `[S§19 Phase 1]` | §18 (Phase 1: signal recording only) |
| Phase 2: compliance reporting, attendance, assessments | `[S§19 Phase 2]` | §18 (Phase 2: full compliance features) |
| Phase 3: portfolios, transcripts | `[S§19 Phase 3]` | §18 (Phase 3: document generation) |

> **Coverage note on `[S§11.6]` (record retention)**: SPEC.md §11.6 covers record retention
> policies. Learning record retention is owned by `learn::` (data is in `learn_*` tables).
> Financial record retention is owned by `billing::`. The `comply::` domain handles its own
> data lifecycle (attendance, assessments, portfolios, transcripts) — these follow the same
> pattern: retained for account lifetime, exportable before deletion, permanently deleted
> after the grace period. `comply::` subscribes to `FamilyDeletionScheduled` to cascade
> deletion of its own data.

> **Coverage note on phasing**: SPEC.md §19 places portfolios and transcripts in Phase 3,
> while compliance reporting and attendance are in Phase 2. This spec assigns portfolio
> generation to Phase 2 and transcripts to Phase 3, reflecting the natural dependency:
> attendance/assessment data (Phase 2) feeds into portfolios (Phase 2), which in turn
> inform the transcript format (Phase 3).

---

## §3 Database Schema

The compliance domain stores state compliance configuration (platform-authored, not
family-scoped), family compliance settings, custom schedules, daily attendance records,
assessment records, standardized test scores, portfolio metadata and items, transcript
metadata (Phase 3), and course definitions (Phase 3). All family-data tables are
family-scoped via `family_id` foreign key. `[ARCH §5.1, ARCH §5.2]`

> **Refinement note**: ARCHITECTURE.md §5.1 sketches `comply_attendance`,
> `comply_state_configs`, and `comply_portfolios`. This spec adds: (1) `comply_family_configs`
> for per-family compliance settings, (2) `comply_custom_schedules` for non-standard schedules,
> (3) `comply_assessment_records` for aggregated assessment data, (4) `comply_standardized_tests`
> for test scores, (5) `comply_portfolio_items` for selected items, (6) `comply_transcripts`
> and `comply_courses` for Phase 3 transcript generation.

### §3.1 Enums

Implemented as `CHECK` constraints (not PostgreSQL ENUM types) per `[CODING §4.1]`:

```sql
-- Attendance status values: present_full, present_partial, absent, not_applicable
-- Portfolio status values: configuring, generating, ready, failed, expired
-- Transcript status values: configuring, generating, ready, failed, expired (Phase 3)
-- GPA scale values: standard_4, weighted, custom (Phase 3)
-- Portfolio organization values: by_subject, chronological, by_student
-- Course level values: regular, honors, ap (Phase 3)
```

### §3.2 Table 1: comply_state_configs — State Requirements Cache

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_comply_tables.rs (Phase 1)
-- =============================================================================

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: comply_state_configs — Cached state compliance requirements
-- ═══════════════════════════════════════════════════════════════════════════════
-- Platform-authored data synced from discover::disc_state_guides via
-- SyncStateConfigsJob. NOT family-scoped — this is reference data shared
-- across all families. 51 rows (50 states + DC). Flattened columns (not JSONB)
-- for efficient attendance threshold queries. [S§11.1]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_state_configs (
    state_code                CHAR(2) PRIMARY KEY,         -- e.g., 'CA', 'TX', 'DC'
    state_name                TEXT NOT NULL,
    notification_required     BOOLEAN NOT NULL DEFAULT false,
    notification_details      TEXT,
    required_subjects         TEXT[] NOT NULL DEFAULT '{}', -- from StateGuideRequirements
    assessment_required       BOOLEAN NOT NULL DEFAULT false,
    assessment_details        TEXT,
    record_keeping_required   BOOLEAN NOT NULL DEFAULT false,
    record_keeping_details    TEXT,
    attendance_required       BOOLEAN NOT NULL DEFAULT false,
    attendance_days           SMALLINT,                     -- minimum required days per year
    attendance_hours          SMALLINT,                     -- minimum required hours per year (some states use hours)
    attendance_details        TEXT,
    umbrella_school_available BOOLEAN NOT NULL DEFAULT false,
    umbrella_school_details   TEXT,
    regulation_level          TEXT NOT NULL DEFAULT 'moderate'
                              CHECK (regulation_level IN ('low', 'moderate', 'high')),
    synced_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### §3.3 Tables 2-3: Family Config & Custom Schedules

```sql
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 2: comply_family_configs — Per-family compliance settings [S§11.1]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Stores the family's chosen state for compliance, school year boundaries,
-- and GPA configuration. One row per family. Created when a premium family
-- first sets up compliance.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_family_configs (
    family_id                 UUID PRIMARY KEY REFERENCES iam_families(id) ON DELETE CASCADE,
    state_code                CHAR(2) NOT NULL REFERENCES comply_state_configs(state_code),
    school_year_start         DATE NOT NULL,               -- e.g., 2025-08-15
    school_year_end           DATE NOT NULL,               -- e.g., 2026-05-31
    total_school_days         SMALLINT NOT NULL DEFAULT 180, -- family's planned school days
    custom_schedule_id        UUID REFERENCES comply_custom_schedules(id),
    gpa_scale                 TEXT NOT NULL DEFAULT 'standard_4'
                              CHECK (gpa_scale IN ('standard_4', 'weighted', 'custom')),
    gpa_custom_config         JSONB,                       -- custom scale config (when gpa_scale = 'custom')
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_school_year CHECK (school_year_end > school_year_start)
);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 3: comply_custom_schedules — Custom schedule definitions [S§11.2]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Families with non-standard schedules (4-day weeks, year-round, etc.) define
-- their schedule here. Used by attendance threshold calculations to determine
-- expected school days. If no custom schedule, the standard Mon-Fri schedule
-- is assumed.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_custom_schedules (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                 UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    name                      TEXT NOT NULL,                -- e.g., "4-Day Week", "Year-Round"
    school_days               BOOLEAN[] NOT NULL DEFAULT '{true,true,true,true,true,false,false}',
                                                           -- Mon-Sun, true = school day
    exclusion_periods         JSONB NOT NULL DEFAULT '[]',  -- [{start: "2025-12-20", end: "2026-01-03", label: "Winter Break"}]
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Family's custom schedules
CREATE INDEX idx_comply_custom_schedules_family
    ON comply_custom_schedules(family_id);
```

### §3.4 Table 4: comply_attendance — Daily Attendance

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000002_create_comply_attendance.rs (Phase 1)
-- =============================================================================

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 4: comply_attendance — Daily attendance per student [S§11.2]
-- ═══════════════════════════════════════════════════════════════════════════════
-- One row per student per day. Can be created manually by the parent or
-- automatically from ActivityLogged events. Manual entries take precedence
-- over auto-generated ones (manual_override flag). UNIQUE constraint on
-- (family_id, student_id, attendance_date) prevents duplicates.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_attendance (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                 UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id                UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    attendance_date           DATE NOT NULL,
    status                    TEXT NOT NULL DEFAULT 'present_full'
                              CHECK (status IN (
                                  'present_full', 'present_partial',
                                  'absent', 'not_applicable'
                              )),
    duration_minutes          SMALLINT,                     -- for partial attendance or hour-based states
    notes                     TEXT,
    is_auto                   BOOLEAN NOT NULL DEFAULT false, -- true if generated from ActivityLogged
    manual_override           BOOLEAN NOT NULL DEFAULT false, -- true if parent manually set (overrides auto)
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_attendance_student_date UNIQUE (family_id, student_id, attendance_date)
);

-- Attendance queries: by family + student + date range (most common query pattern)
CREATE INDEX idx_comply_attendance_family_student_date
    ON comply_attendance(family_id, student_id, attendance_date DESC);

-- Attendance summary queries: count by status within date range
CREATE INDEX idx_comply_attendance_status
    ON comply_attendance(family_id, student_id, status, attendance_date);
```

### §3.5 Table 5: comply_assessment_records — Aggregated Assessments

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000003_create_comply_assessment_records.rs (Phase 2)
-- =============================================================================

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 5: comply_assessment_records — Aggregated assessment data [S§11.3]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Compliance-ready assessment records aggregated from learn:: or entered
-- manually. References learn:: activity IDs by UUID without foreign key
-- (no cross-domain FKs). Each record represents a single gradeable item.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_assessment_records (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                 UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id                UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    title                     TEXT NOT NULL,
    subject                   TEXT NOT NULL,                -- subject tag from taxonomy
    assessment_type           TEXT NOT NULL
                              CHECK (assessment_type IN (
                                  'test', 'quiz', 'project', 'assignment',
                                  'presentation', 'portfolio_piece', 'other'
                              )),
    score                     NUMERIC(6,2),                 -- raw score (nullable for pass/fail)
    max_score                 NUMERIC(6,2),                 -- max possible score
    grade_letter              TEXT,                         -- A, B+, etc. (nullable)
    grade_points              NUMERIC(3,2),                 -- 4.0 scale points (nullable)
    is_passing                BOOLEAN,                      -- for pass/fail assessments
    source_activity_id        UUID,                         -- references learn_activities.id (no FK)
    assessment_date           DATE NOT NULL,
    notes                     TEXT,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Assessment queries: by student + subject + date range
CREATE INDEX idx_comply_assessments_family_student
    ON comply_assessment_records(family_id, student_id, assessment_date DESC);

CREATE INDEX idx_comply_assessments_subject
    ON comply_assessment_records(family_id, student_id, subject, assessment_date DESC);
```

### §3.6 Table 6: comply_standardized_tests — Test Scores

```sql
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 6: comply_standardized_tests — Standardized test scores [S§11.3]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Stores standardized test scores for states that require them. Uses JSONB
-- for score sections because test formats vary widely (SAT sections differ
-- from Iowa Assessments differ from state-specific tests).
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_standardized_tests (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                 UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id                UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    test_name                 TEXT NOT NULL,                -- e.g., "Iowa Assessments", "SAT", "ACT"
    test_date                 DATE NOT NULL,
    grade_level               SMALLINT,                    -- grade at time of test
    scores                    JSONB NOT NULL,               -- {"sections": [{"name": "Math", "score": 85, "percentile": 72}]}
    composite_score           NUMERIC(6,2),                 -- overall composite (if applicable)
    percentile                SMALLINT,                     -- overall percentile (if applicable)
    notes                     TEXT,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Test score queries: by student + date
CREATE INDEX idx_comply_tests_family_student
    ON comply_standardized_tests(family_id, student_id, test_date DESC);
```

### §3.7 Table 7: comply_portfolios — Portfolio Metadata

```sql
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 7: comply_portfolios — Portfolio metadata + generation state [S§11.4]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Tracks portfolio configuration and generation state. The actual PDF is
-- stored via media:: (upload_id reference). State machine tracks generation
-- lifecycle: configuring → generating → ready → expired (with failed + retry).
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_portfolios (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                 UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id                UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    title                     TEXT NOT NULL,
    description               TEXT,
    organization              TEXT NOT NULL DEFAULT 'chronological'
                              CHECK (organization IN (
                                  'by_subject', 'chronological', 'by_student'
                              )),
    date_range_start          DATE NOT NULL,
    date_range_end            DATE NOT NULL,
    include_attendance        BOOLEAN NOT NULL DEFAULT true,
    include_assessments       BOOLEAN NOT NULL DEFAULT true,
    status                    TEXT NOT NULL DEFAULT 'configuring'
                              CHECK (status IN (
                                  'configuring', 'generating', 'ready',
                                  'failed', 'expired'
                              )),
    upload_id                 UUID,                         -- references media_uploads.id (no FK — cross-domain)
    generated_at              TIMESTAMPTZ,
    expires_at                TIMESTAMPTZ,                  -- 90 days after generation
    error_message             TEXT,                         -- internal error (not exposed in API)
    retry_count               SMALLINT NOT NULL DEFAULT 0,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_portfolio_date_range CHECK (date_range_end > date_range_start)
);

-- Portfolio queries: by family + student
CREATE INDEX idx_comply_portfolios_family_student
    ON comply_portfolios(family_id, student_id, created_at DESC);

-- Expired portfolio cleanup
CREATE INDEX idx_comply_portfolios_expired
    ON comply_portfolios(expires_at)
    WHERE status = 'ready' AND expires_at IS NOT NULL;
```

### §3.8 Table 8: comply_portfolio_items — Selected Items

```sql
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 8: comply_portfolio_items — Selected learn:: items [S§11.4]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Parents select which learn:: artifacts to include in a portfolio. Display
-- data is cached at selection time to avoid cross-domain queries during
-- PDF generation. References learn:: data by UUID without FK.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_portfolio_items (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id              UUID NOT NULL REFERENCES comply_portfolios(id) ON DELETE CASCADE,
    source_type               TEXT NOT NULL
                              CHECK (source_type IN (
                                  'activity', 'journal', 'project',
                                  'reading_list', 'assessment'
                              )),
    source_id                 UUID NOT NULL,                -- references learn_* table ID (no FK)
    display_order             SMALLINT NOT NULL DEFAULT 0,
    -- Cached display data (snapshot at selection time)
    cached_title              TEXT NOT NULL,
    cached_subject            TEXT,
    cached_date               DATE NOT NULL,
    cached_description        TEXT,
    cached_attachments        JSONB DEFAULT '[]',           -- cached media URLs at selection time
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Items per portfolio, ordered
CREATE INDEX idx_comply_portfolio_items_portfolio
    ON comply_portfolio_items(portfolio_id, display_order);

-- Prevent duplicate items in same portfolio
CREATE UNIQUE INDEX idx_comply_portfolio_items_unique
    ON comply_portfolio_items(portfolio_id, source_type, source_id);
```

### §3.9 Table 9: comply_transcripts — Transcript Metadata (Phase 3)

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000004_create_comply_transcripts.rs (Phase 3)
-- =============================================================================

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 9: comply_transcripts — Transcript metadata + generation state [S§11.5]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Tracks transcript configuration and generation state. Same state machine
-- pattern as portfolios. GPA is computed on-the-fly and snapshotted at
-- generation time (stored in snapshot_gpa).
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_transcripts (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                 UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id                UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    title                     TEXT NOT NULL,                -- e.g., "Official Transcript — 2025-2026"
    student_name              TEXT NOT NULL,                -- cached at creation time
    grade_levels              TEXT[] NOT NULL DEFAULT '{}', -- e.g., ['9', '10', '11', '12']
    status                    TEXT NOT NULL DEFAULT 'configuring'
                              CHECK (status IN (
                                  'configuring', 'generating', 'ready',
                                  'failed', 'expired'
                              )),
    snapshot_gpa_unweighted   NUMERIC(4,3),                 -- snapshotted at generation time
    snapshot_gpa_weighted     NUMERIC(4,3),                 -- snapshotted at generation time
    upload_id                 UUID,                         -- references media_uploads.id (no FK)
    generated_at              TIMESTAMPTZ,
    expires_at                TIMESTAMPTZ,                  -- 90 days after generation
    error_message             TEXT,
    retry_count               SMALLINT NOT NULL DEFAULT 0,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Transcript queries: by family + student
CREATE INDEX idx_comply_transcripts_family_student
    ON comply_transcripts(family_id, student_id, created_at DESC);
```

### §3.10 Table 10: comply_courses — Course Definitions (Phase 3)

```sql
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 10: comply_courses — Course definitions for transcripts [S§11.5]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Parent-defined courses that appear on transcripts. Not directly tied to
-- learn:: data — parents define course metadata (title, credits, grade)
-- independently. GPA is computed from this table.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_courses (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                 UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id                UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    transcript_id             UUID REFERENCES comply_transcripts(id) ON DELETE SET NULL,
    title                     TEXT NOT NULL,                -- e.g., "Algebra II"
    subject                   TEXT NOT NULL,                -- subject category
    grade_level               SMALLINT NOT NULL,            -- 9, 10, 11, or 12
    credits                   NUMERIC(3,2) NOT NULL DEFAULT 1.0,
    grade_letter              TEXT,                         -- A, B+, C, etc.
    grade_points              NUMERIC(3,2),                 -- 4.0 scale
    level                     TEXT NOT NULL DEFAULT 'regular'
                              CHECK (level IN ('regular', 'honors', 'ap')),
    school_year               TEXT NOT NULL,                -- e.g., "2025-2026"
    semester                  TEXT
                              CHECK (semester IS NULL OR semester IN (
                                  'fall', 'spring', 'summer', 'full_year'
                              )),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Course queries: by student + grade level
CREATE INDEX idx_comply_courses_family_student
    ON comply_courses(family_id, student_id, grade_level, school_year);
```

### §3.11 RLS / Family-Scoping

Application-layer enforcement via `FamilyScope` extractor `[CODING §2.4, §2.5, 00-core §8]`:

```sql
-- Application-layer enforcement: every query on comply_family_configs,
-- comply_custom_schedules, comply_attendance, comply_assessment_records,
-- comply_standardized_tests, comply_portfolios, comply_portfolio_items,
-- comply_transcripts, and comply_courses MUST include
-- WHERE family_id = $family_scope.family_id
-- Repository methods accept FamilyScope as first parameter to enforce this.

-- EXCEPTION: comply_state_configs is NOT family-scoped — it is platform-
-- authored reference data shared across all families. No FamilyScope
-- required for state config queries.
```

---

## §4 API Endpoints

**Prefix**: `/v1/compliance`

**Auth**: All endpoints require `RequirePremium(auth)` + `FamilyScope`. Returns `402 Premium
Required` for free-tier families. `[00-core §13.3, S§15.2]`

### §4.1 Phase 1 (0 endpoints)

No API endpoints in Phase 1. The only Phase 1 behavior is the `ActivityLogged` event handler
that records attendance signals into `comply_attendance`. `[S§19 Phase 1]`

### §4.2 Phase 2 (~28 endpoints)

#### Compliance Configuration (4 endpoints)

**GET /v1/compliance/config** — Get family compliance configuration.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `FamilyConfigResponse`
- Returns `null` if no config exists yet (family hasn't set up compliance).

**PUT /v1/compliance/config** — Create or update family compliance configuration.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `UpsertFamilyConfigCommand`
```rust
{
    "state_code": "CA",
    "school_year_start": "2025-08-15",
    "school_year_end": "2026-05-31",
    "total_school_days": 180,
    "custom_schedule_id": null,
    "gpa_scale": "standard_4"
}
```
- **Response**: `200 OK` → `FamilyConfigResponse`
- **Error codes**: `422` (invalid state code, invalid date range)

**GET /v1/compliance/state-requirements** — List all state requirements (from cache).
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `Vec<StateConfigSummaryResponse>`

**GET /v1/compliance/state-requirements/:state_code** — Get requirements for a specific state.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `StateConfigResponse`
- **Error codes**: `404` (unknown state code)

#### Custom Schedules (4 endpoints)

**POST /v1/compliance/schedules** — Create a custom schedule.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `CreateScheduleCommand`
- **Response**: `201 Created` → `ScheduleResponse`

**GET /v1/compliance/schedules** — List family's custom schedules.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `Vec<ScheduleResponse>`

**PATCH /v1/compliance/schedules/:id** — Update a custom schedule.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `UpdateScheduleCommand`
- **Response**: `200 OK` → `ScheduleResponse`
- **Error codes**: `404` (schedule not found or not owned by family)

**DELETE /v1/compliance/schedules/:id** — Delete a custom schedule.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `204 No Content`
- **Error codes**: `404`, `409` (schedule in use by family config)

#### Attendance (6 endpoints)

**POST /v1/compliance/students/:student_id/attendance** — Record attendance for a day.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `RecordAttendanceCommand`
```rust
{
    "attendance_date": "2025-10-15",
    "status": "present_full",
    "duration_minutes": null,
    "notes": null
}
```
- **Response**: `201 Created` → `AttendanceResponse`
- **Side effects**: Sets `manual_override = true`, `is_auto = false`. Upserts on the
  UNIQUE constraint — if an auto-generated record exists for that date, it is replaced.
- **Error codes**: `422` (future date, invalid status), `404` (student not in family)

**GET /v1/compliance/students/:student_id/attendance** — List attendance records.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Query**: `AttendanceListParams { start_date, end_date, status?, cursor?, limit? }`
- **Response**: `200 OK` → `AttendanceListResponse`

**GET /v1/compliance/students/:student_id/attendance/summary** — Attendance summary.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Query**: `AttendanceSummaryParams { start_date, end_date }`
- **Response**: `200 OK` → `AttendanceSummaryResponse`
```rust
{
    "total_days": 92,
    "present_full": 78,
    "present_partial": 5,
    "absent": 4,
    "not_applicable": 5,
    "total_hours": 520.5,
    "state_required_days": 180,
    "state_required_hours": null,
    "pace_status": "on_track",
    "projected_total_days": 185
}
```

**PATCH /v1/compliance/students/:student_id/attendance/:id** — Update an attendance record.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `UpdateAttendanceCommand`
- **Response**: `200 OK` → `AttendanceResponse`
- **Error codes**: `404` (not found or not owned by family)

**DELETE /v1/compliance/students/:student_id/attendance/:id** — Delete an attendance record.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `204 No Content`
- **Error codes**: `404`

**POST /v1/compliance/students/:student_id/attendance/bulk** — Bulk record attendance.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `BulkRecordAttendanceCommand { records: Vec<RecordAttendanceCommand> }`
- **Response**: `201 Created` → `Vec<AttendanceResponse>`
- **Limit**: Max 31 records per request (one month)
- **Error codes**: `422` (exceeds limit, future dates, invalid statuses)

#### Assessment Records (4 endpoints)

**POST /v1/compliance/students/:student_id/assessments** — Record an assessment.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `CreateAssessmentCommand`
- **Response**: `201 Created` → `AssessmentResponse`

**GET /v1/compliance/students/:student_id/assessments** — List assessments.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Query**: `AssessmentListParams { subject?, start_date?, end_date?, cursor?, limit? }`
- **Response**: `200 OK` → `AssessmentListResponse`

**PATCH /v1/compliance/students/:student_id/assessments/:id** — Update an assessment.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `UpdateAssessmentCommand`
- **Response**: `200 OK` → `AssessmentResponse`
- **Error codes**: `404`

**DELETE /v1/compliance/students/:student_id/assessments/:id** — Delete an assessment.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `204 No Content`
- **Error codes**: `404`

#### Standardized Tests (4 endpoints)

**POST /v1/compliance/students/:student_id/tests** — Record a test score.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `CreateTestScoreCommand`
```rust
{
    "test_name": "Iowa Assessments",
    "test_date": "2025-10-20",
    "grade_level": 6,
    "scores": {
        "sections": [
            {"name": "Reading", "score": 85, "percentile": 72},
            {"name": "Math", "score": 92, "percentile": 88}
        ]
    },
    "composite_score": 88.5,
    "percentile": 80
}
```
- **Response**: `201 Created` → `TestScoreResponse`

**GET /v1/compliance/students/:student_id/tests** — List test scores.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Query**: `TestListParams { cursor?, limit? }`
- **Response**: `200 OK` → `TestListResponse`

**PATCH /v1/compliance/students/:student_id/tests/:id** — Update a test score.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `TestScoreResponse`
- **Error codes**: `404`

**DELETE /v1/compliance/students/:student_id/tests/:id** — Delete a test score.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `204 No Content`
- **Error codes**: `404`

#### Portfolios (6 endpoints)

**POST /v1/compliance/students/:student_id/portfolios** — Create a portfolio.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `CreatePortfolioCommand`
```rust
{
    "title": "Fall 2025 Portfolio",
    "description": "Overview of fall semester work",
    "organization": "by_subject",
    "date_range_start": "2025-08-15",
    "date_range_end": "2025-12-20",
    "include_attendance": true,
    "include_assessments": true
}
```
- **Response**: `201 Created` → `PortfolioResponse` (status: `configuring`)

**GET /v1/compliance/students/:student_id/portfolios** — List portfolios.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `Vec<PortfolioSummaryResponse>`

**GET /v1/compliance/students/:student_id/portfolios/:id** — Get portfolio details.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `PortfolioResponse` (includes items)
- **Error codes**: `404`

**POST /v1/compliance/students/:student_id/portfolios/:id/items** — Add items to portfolio.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `AddPortfolioItemsCommand { items: Vec<PortfolioItemInput> }`
```rust
{
    "items": [
        { "source_type": "activity", "source_id": "uuid" },
        { "source_type": "journal", "source_id": "uuid" }
    ]
}
```
- **Response**: `200 OK` → `Vec<PortfolioItemResponse>`
- **Side effects**: Fetches display data from `learn::LearningService` and caches it.
- **Error codes**: `404` (portfolio not found), `409` (portfolio not in `configuring` status),
  `422` (source item not found in learn::)

**POST /v1/compliance/students/:student_id/portfolios/:id/generate** — Generate portfolio PDF.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `202 Accepted` → `PortfolioResponse` (status: `generating`)
- **Side effects**: Enqueues `GeneratePortfolioJob`. Cannot generate if no items added.
- **Error codes**: `404`, `409` (not in `configuring` status or no items),
  `422` (empty portfolio)

**GET /v1/compliance/students/:student_id/portfolios/:id/download** — Download portfolio PDF.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `302 Found` → Redirect to presigned download URL (from `media::`)
- **Error codes**: `404`, `409` (not in `ready` status), `410` (expired)

#### Compliance Dashboard (1 endpoint)

**GET /v1/compliance/dashboard** — Compliance dashboard overview.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `ComplianceDashboardResponse`
```rust
{
    "family_config": { /* FamilyConfigResponse or null */ },
    "students": [
        {
            "student_id": "uuid",
            "student_name": "...",
            "attendance_summary": { /* AttendanceSummaryResponse */ },
            "recent_assessments_count": 12,
            "recent_tests_count": 1,
            "active_portfolios": [{ /* PortfolioSummaryResponse */ }],
            "pace_status": "on_track"
        }
    ]
}
```

### §4.3 Phase 3 (~14 endpoints)

#### Transcripts (6 endpoints)

**POST /v1/compliance/students/:student_id/transcripts** — Create a transcript.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `CreateTranscriptCommand`
- **Response**: `201 Created` → `TranscriptResponse`

**GET /v1/compliance/students/:student_id/transcripts** — List transcripts.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `Vec<TranscriptSummaryResponse>`

**GET /v1/compliance/students/:student_id/transcripts/:id** — Get transcript details.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `TranscriptResponse` (includes courses, GPA)

**POST /v1/compliance/students/:student_id/transcripts/:id/generate** — Generate transcript PDF.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `202 Accepted` → `TranscriptResponse` (status: `generating`)
- **Side effects**: Snapshots GPA at generation time. Enqueues `GenerateTranscriptJob`.

**GET /v1/compliance/students/:student_id/transcripts/:id/download** — Download transcript PDF.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `302 Found` → Redirect to presigned download URL

**DELETE /v1/compliance/students/:student_id/transcripts/:id** — Delete a transcript.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `204 No Content`

#### Courses (5 endpoints)

**POST /v1/compliance/students/:student_id/courses** — Create a course.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `CreateCourseCommand`
- **Response**: `201 Created` → `CourseResponse`

**GET /v1/compliance/students/:student_id/courses** — List courses.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Query**: `CourseListParams { grade_level?, school_year?, cursor?, limit? }`
- **Response**: `200 OK` → `CourseListResponse`

**PATCH /v1/compliance/students/:student_id/courses/:id** — Update a course.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `CourseResponse`

**DELETE /v1/compliance/students/:student_id/courses/:id** — Delete a course.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `204 No Content`

#### GPA (3 endpoints)

**GET /v1/compliance/students/:student_id/gpa** — Calculate current GPA.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Query**: `GpaParams { scale?, grade_levels? }`
- **Response**: `200 OK` → `GpaResponse`
```rust
{
    "unweighted_gpa": 3.65,
    "weighted_gpa": 3.85,
    "total_credits": 24.0,
    "total_courses": 24,
    "by_grade_level": [
        { "grade_level": 9, "unweighted": 3.50, "weighted": 3.70, "credits": 6.0 },
        { "grade_level": 10, "unweighted": 3.80, "weighted": 4.00, "credits": 6.0 }
    ]
}
```

**GET /v1/compliance/students/:student_id/gpa/what-if** — GPA what-if calculator.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Query**: `GpaWhatIfParams { additional_courses: Vec<WhatIfCourse> }`
- **Response**: `200 OK` → `GpaResponse` (projected GPA with hypothetical courses)

**GET /v1/compliance/students/:student_id/gpa/history** — GPA history by term.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `Vec<GpaTermResponse>`

---

## §5 Service Interface

The `ComplianceService` trait defines all use cases. CQRS separation: command methods (writes
with side effects) are separated from query methods (reads). `[ARCH §4.7, CODING §8.2]`

```rust
// src/comply/ports.rs

use crate::shared::types::{FamilyId, FamilyScope, StudentId};
use crate::shared::error::AppError;

#[async_trait]
pub trait ComplianceService: Send + Sync {

    // ─── Command side (writes, side effects) ────────────────────────────

    /// Create or update family compliance configuration.
    async fn upsert_family_config(
        &self,
        cmd: UpsertFamilyConfigCommand,
        scope: FamilyScope,
    ) -> Result<FamilyConfigResponse, AppError>;

    /// Create a custom schedule.
    async fn create_schedule(
        &self,
        cmd: CreateScheduleCommand,
        scope: FamilyScope,
    ) -> Result<ScheduleResponse, AppError>;

    /// Update a custom schedule.
    async fn update_schedule(
        &self,
        schedule_id: Uuid,
        cmd: UpdateScheduleCommand,
        scope: FamilyScope,
    ) -> Result<ScheduleResponse, AppError>;

    /// Delete a custom schedule.
    async fn delete_schedule(
        &self,
        schedule_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    /// Record daily attendance for a student (manual entry).
    async fn record_attendance(
        &self,
        student_id: StudentId,
        cmd: RecordAttendanceCommand,
        scope: FamilyScope,
    ) -> Result<AttendanceResponse, AppError>;

    /// Bulk record attendance for a student.
    async fn bulk_record_attendance(
        &self,
        student_id: StudentId,
        cmd: BulkRecordAttendanceCommand,
        scope: FamilyScope,
    ) -> Result<Vec<AttendanceResponse>, AppError>;

    /// Update an attendance record.
    async fn update_attendance(
        &self,
        student_id: StudentId,
        attendance_id: Uuid,
        cmd: UpdateAttendanceCommand,
        scope: FamilyScope,
    ) -> Result<AttendanceResponse, AppError>;

    /// Delete an attendance record.
    async fn delete_attendance(
        &self,
        student_id: StudentId,
        attendance_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    /// Create an assessment record.
    async fn create_assessment(
        &self,
        student_id: StudentId,
        cmd: CreateAssessmentCommand,
        scope: FamilyScope,
    ) -> Result<AssessmentResponse, AppError>;

    /// Update an assessment record.
    async fn update_assessment(
        &self,
        student_id: StudentId,
        assessment_id: Uuid,
        cmd: UpdateAssessmentCommand,
        scope: FamilyScope,
    ) -> Result<AssessmentResponse, AppError>;

    /// Delete an assessment record.
    async fn delete_assessment(
        &self,
        student_id: StudentId,
        assessment_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    /// Record a standardized test score.
    async fn create_test_score(
        &self,
        student_id: StudentId,
        cmd: CreateTestScoreCommand,
        scope: FamilyScope,
    ) -> Result<TestScoreResponse, AppError>;

    /// Update a test score.
    async fn update_test_score(
        &self,
        student_id: StudentId,
        test_id: Uuid,
        cmd: UpdateTestScoreCommand,
        scope: FamilyScope,
    ) -> Result<TestScoreResponse, AppError>;

    /// Delete a test score.
    async fn delete_test_score(
        &self,
        student_id: StudentId,
        test_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    /// Create a portfolio.
    async fn create_portfolio(
        &self,
        student_id: StudentId,
        cmd: CreatePortfolioCommand,
        scope: FamilyScope,
    ) -> Result<PortfolioResponse, AppError>;

    /// Add items to a portfolio (caches display data from learn::).
    async fn add_portfolio_items(
        &self,
        student_id: StudentId,
        portfolio_id: Uuid,
        cmd: AddPortfolioItemsCommand,
        scope: FamilyScope,
    ) -> Result<Vec<PortfolioItemResponse>, AppError>;

    /// Trigger portfolio PDF generation (async — enqueues job).
    async fn generate_portfolio(
        &self,
        student_id: StudentId,
        portfolio_id: Uuid,
        scope: FamilyScope,
    ) -> Result<PortfolioResponse, AppError>;

    // Phase 3: Transcript commands

    /// Create a transcript. (Phase 3)
    async fn create_transcript(
        &self,
        student_id: StudentId,
        cmd: CreateTranscriptCommand,
        scope: FamilyScope,
    ) -> Result<TranscriptResponse, AppError>;

    /// Trigger transcript PDF generation (async — enqueues job). (Phase 3)
    async fn generate_transcript(
        &self,
        student_id: StudentId,
        transcript_id: Uuid,
        scope: FamilyScope,
    ) -> Result<TranscriptResponse, AppError>;

    /// Delete a transcript. (Phase 3)
    async fn delete_transcript(
        &self,
        student_id: StudentId,
        transcript_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    /// Create a course for transcript. (Phase 3)
    async fn create_course(
        &self,
        student_id: StudentId,
        cmd: CreateCourseCommand,
        scope: FamilyScope,
    ) -> Result<CourseResponse, AppError>;

    /// Update a course. (Phase 3)
    async fn update_course(
        &self,
        student_id: StudentId,
        course_id: Uuid,
        cmd: UpdateCourseCommand,
        scope: FamilyScope,
    ) -> Result<CourseResponse, AppError>;

    /// Delete a course. (Phase 3)
    async fn delete_course(
        &self,
        student_id: StudentId,
        course_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    // ─── Query side (reads, no side effects) ────────────────────────────

    /// Get family compliance configuration.
    async fn get_family_config(
        &self,
        scope: FamilyScope,
    ) -> Result<Option<FamilyConfigResponse>, AppError>;

    /// List all state requirements (from cache).
    async fn list_state_configs(
        &self,
    ) -> Result<Vec<StateConfigSummaryResponse>, AppError>;

    /// Get requirements for a specific state.
    async fn get_state_config(
        &self,
        state_code: &str,
    ) -> Result<StateConfigResponse, AppError>;

    /// List family's custom schedules.
    async fn list_schedules(
        &self,
        scope: FamilyScope,
    ) -> Result<Vec<ScheduleResponse>, AppError>;

    /// List attendance records for a student.
    async fn list_attendance(
        &self,
        student_id: StudentId,
        params: AttendanceListParams,
        scope: FamilyScope,
    ) -> Result<AttendanceListResponse, AppError>;

    /// Get attendance summary for a student within a date range.
    async fn get_attendance_summary(
        &self,
        student_id: StudentId,
        params: AttendanceSummaryParams,
        scope: FamilyScope,
    ) -> Result<AttendanceSummaryResponse, AppError>;

    /// List assessment records for a student.
    async fn list_assessments(
        &self,
        student_id: StudentId,
        params: AssessmentListParams,
        scope: FamilyScope,
    ) -> Result<AssessmentListResponse, AppError>;

    /// List standardized test scores for a student.
    async fn list_test_scores(
        &self,
        student_id: StudentId,
        params: TestListParams,
        scope: FamilyScope,
    ) -> Result<TestListResponse, AppError>;

    /// Get portfolio details (includes items).
    async fn get_portfolio(
        &self,
        student_id: StudentId,
        portfolio_id: Uuid,
        scope: FamilyScope,
    ) -> Result<PortfolioResponse, AppError>;

    /// List portfolios for a student.
    async fn list_portfolios(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
    ) -> Result<Vec<PortfolioSummaryResponse>, AppError>;

    /// Get presigned download URL for a portfolio PDF.
    async fn get_portfolio_download_url(
        &self,
        student_id: StudentId,
        portfolio_id: Uuid,
        scope: FamilyScope,
    ) -> Result<String, AppError>;

    /// Get compliance dashboard overview.
    async fn get_dashboard(
        &self,
        scope: FamilyScope,
    ) -> Result<ComplianceDashboardResponse, AppError>;

    // Phase 3: Transcript queries

    /// Get transcript details (includes courses, GPA). (Phase 3)
    async fn get_transcript(
        &self,
        student_id: StudentId,
        transcript_id: Uuid,
        scope: FamilyScope,
    ) -> Result<TranscriptResponse, AppError>;

    /// List transcripts for a student. (Phase 3)
    async fn list_transcripts(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
    ) -> Result<Vec<TranscriptSummaryResponse>, AppError>;

    /// Get presigned download URL for a transcript PDF. (Phase 3)
    async fn get_transcript_download_url(
        &self,
        student_id: StudentId,
        transcript_id: Uuid,
        scope: FamilyScope,
    ) -> Result<String, AppError>;

    /// List courses for a student. (Phase 3)
    async fn list_courses(
        &self,
        student_id: StudentId,
        params: CourseListParams,
        scope: FamilyScope,
    ) -> Result<CourseListResponse, AppError>;

    /// Calculate current GPA for a student. (Phase 3)
    async fn calculate_gpa(
        &self,
        student_id: StudentId,
        params: GpaParams,
        scope: FamilyScope,
    ) -> Result<GpaResponse, AppError>;

    /// Calculate what-if GPA with hypothetical courses. (Phase 3)
    async fn calculate_gpa_what_if(
        &self,
        student_id: StudentId,
        params: GpaWhatIfParams,
        scope: FamilyScope,
    ) -> Result<GpaResponse, AppError>;

    /// GPA history by term. (Phase 3)
    async fn get_gpa_history(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
    ) -> Result<Vec<GpaTermResponse>, AppError>;

    // ─── Event handlers ─────────────────────────────────────────────────

    /// Handle ActivityLogged event: auto-record attendance. (Phase 1)
    /// Consumed from learn::ActivityLogged. [06-learn §18.1]
    async fn handle_activity_logged(
        &self,
        event: &ActivityLogged,
    ) -> Result<(), AppError>;

    /// Handle StudentDeleted event: cascade delete compliance data.
    /// Consumed from iam::StudentDeleted. [01-iam §13.3]
    async fn handle_student_deleted(
        &self,
        event: &StudentDeleted,
    ) -> Result<(), AppError>;

    /// Handle FamilyDeletionScheduled event: cascade delete all comply:: data.
    /// Consumed from iam::FamilyDeletionScheduled. [01-iam §13.3]
    async fn handle_family_deletion_scheduled(
        &self,
        event: &FamilyDeletionScheduled,
    ) -> Result<(), AppError>;

    /// Handle SubscriptionCancelled event: mark premium features as read-only.
    /// Consumed from billing::SubscriptionCancelled. [10-billing §16.3]
    async fn handle_subscription_cancelled(
        &self,
        event: &SubscriptionCancelled,
    ) -> Result<(), AppError>;
}
```

### `ComplianceServiceImpl`

```rust
// src/comply/service.rs

pub struct ComplianceServiceImpl {
    family_config_repo: Arc<dyn FamilyConfigRepository>,
    schedule_repo: Arc<dyn ScheduleRepository>,
    attendance_repo: Arc<dyn AttendanceRepository>,
    assessment_repo: Arc<dyn AssessmentRepository>,
    test_repo: Arc<dyn TestScoreRepository>,
    portfolio_repo: Arc<dyn PortfolioRepository>,
    portfolio_item_repo: Arc<dyn PortfolioItemRepository>,
    state_config_repo: Arc<dyn StateConfigRepository>,
    transcript_repo: Arc<dyn TranscriptRepository>,     // Phase 3
    course_repo: Arc<dyn CourseRepository>,              // Phase 3
    discovery_service: Arc<dyn DiscoveryService>,        // state requirements
    learning_service: Arc<dyn LearningService>,          // portfolio data
    iam_service: Arc<dyn IamService>,                    // student/family data
    media_service: Arc<dyn MediaService>,                // PDF storage
    events: Arc<EventBus>,
}
```

---

## §6 Repository Interfaces

All repository traits defined in `src/comply/ports.rs`. `StateConfigRepository` is NOT
family-scoped — it is platform-authored reference data. All other repositories are
family-scoped via `FamilyScope`. `[CODING §8.2]`

```rust
// src/comply/ports.rs (continued)

// ─── StateConfigRepository ──────────────────────────────────────────────
// NOT family-scoped — platform-authored reference data (51 rows).
#[async_trait]
pub trait StateConfigRepository: Send + Sync {
    /// List all state configs.
    async fn list_all(&self) -> Result<Vec<ComplyStateConfig>, AppError>;

    /// Get config for a specific state.
    async fn find_by_state_code(
        &self,
        state_code: &str,
    ) -> Result<Option<ComplyStateConfig>, AppError>;

    /// Upsert state config (used by SyncStateConfigsJob).
    async fn upsert(
        &self,
        config: UpsertStateConfigRow,
    ) -> Result<ComplyStateConfig, AppError>;
}

// ─── FamilyConfigRepository ─────────────────────────────────────────────
// Family-scoped (family_id is PK).
#[async_trait]
pub trait FamilyConfigRepository: Send + Sync {
    async fn upsert(
        &self,
        scope: FamilyScope,
        input: UpsertFamilyConfigRow,
    ) -> Result<ComplyFamilyConfig, AppError>;

    async fn find_by_family(
        &self,
        scope: FamilyScope,
    ) -> Result<Option<ComplyFamilyConfig>, AppError>;

    async fn delete_by_family(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;
}

// ─── ScheduleRepository ─────────────────────────────────────────────────
// Family-scoped.
#[async_trait]
pub trait ScheduleRepository: Send + Sync {
    async fn create(
        &self,
        scope: FamilyScope,
        input: CreateScheduleRow,
    ) -> Result<ComplyCustomSchedule, AppError>;

    async fn find_by_id(
        &self,
        schedule_id: Uuid,
        scope: FamilyScope,
    ) -> Result<Option<ComplyCustomSchedule>, AppError>;

    async fn list_by_family(
        &self,
        scope: FamilyScope,
    ) -> Result<Vec<ComplyCustomSchedule>, AppError>;

    async fn update(
        &self,
        schedule_id: Uuid,
        scope: FamilyScope,
        updates: UpdateScheduleRow,
    ) -> Result<ComplyCustomSchedule, AppError>;

    async fn delete(
        &self,
        schedule_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;
}

// ─── AttendanceRepository ───────────────────────────────────────────────
// Family-scoped. UNIQUE on (family_id, student_id, attendance_date).
#[async_trait]
pub trait AttendanceRepository: Send + Sync {
    async fn upsert(
        &self,
        scope: FamilyScope,
        input: UpsertAttendanceRow,
    ) -> Result<ComplyAttendance, AppError>;

    async fn find_by_id(
        &self,
        attendance_id: Uuid,
        scope: FamilyScope,
    ) -> Result<Option<ComplyAttendance>, AppError>;

    async fn list_by_student(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
        params: &AttendanceListParams,
    ) -> Result<Vec<ComplyAttendance>, AppError>;

    async fn summarize(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
        start_date: NaiveDate,
        end_date: NaiveDate,
    ) -> Result<AttendanceSummaryRow, AppError>;

    async fn update(
        &self,
        attendance_id: Uuid,
        scope: FamilyScope,
        updates: UpdateAttendanceRow,
    ) -> Result<ComplyAttendance, AppError>;

    async fn delete(
        &self,
        attendance_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    async fn delete_by_student(
        &self,
        student_id: StudentId,
        family_id: FamilyId,
    ) -> Result<(), AppError>;

    async fn delete_by_family(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;
}

// ─── AssessmentRepository ───────────────────────────────────────────────
// Family-scoped.
#[async_trait]
pub trait AssessmentRepository: Send + Sync {
    async fn create(
        &self,
        scope: FamilyScope,
        input: CreateAssessmentRow,
    ) -> Result<ComplyAssessmentRecord, AppError>;

    async fn find_by_id(
        &self,
        assessment_id: Uuid,
        scope: FamilyScope,
    ) -> Result<Option<ComplyAssessmentRecord>, AppError>;

    async fn list_by_student(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
        params: &AssessmentListParams,
    ) -> Result<Vec<ComplyAssessmentRecord>, AppError>;

    async fn update(
        &self,
        assessment_id: Uuid,
        scope: FamilyScope,
        updates: UpdateAssessmentRow,
    ) -> Result<ComplyAssessmentRecord, AppError>;

    async fn delete(
        &self,
        assessment_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    async fn delete_by_student(
        &self,
        student_id: StudentId,
        family_id: FamilyId,
    ) -> Result<(), AppError>;
}

// ─── TestScoreRepository ────────────────────────────────────────────────
// Family-scoped.
#[async_trait]
pub trait TestScoreRepository: Send + Sync {
    async fn create(
        &self,
        scope: FamilyScope,
        input: CreateTestScoreRow,
    ) -> Result<ComplyStandardizedTest, AppError>;

    async fn list_by_student(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
        params: &TestListParams,
    ) -> Result<Vec<ComplyStandardizedTest>, AppError>;

    async fn update(
        &self,
        test_id: Uuid,
        scope: FamilyScope,
        updates: UpdateTestScoreRow,
    ) -> Result<ComplyStandardizedTest, AppError>;

    async fn delete(
        &self,
        test_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;
}

// ─── PortfolioRepository ────────────────────────────────────────────────
// Family-scoped.
#[async_trait]
pub trait PortfolioRepository: Send + Sync {
    async fn create(
        &self,
        scope: FamilyScope,
        input: CreatePortfolioRow,
    ) -> Result<ComplyPortfolio, AppError>;

    async fn find_by_id(
        &self,
        portfolio_id: Uuid,
        scope: FamilyScope,
    ) -> Result<Option<ComplyPortfolio>, AppError>;

    async fn list_by_student(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
    ) -> Result<Vec<ComplyPortfolio>, AppError>;

    async fn update_status(
        &self,
        portfolio_id: Uuid,
        status: &str,
        upload_id: Option<Uuid>,
        error_message: Option<&str>,
    ) -> Result<ComplyPortfolio, AppError>;

    async fn find_expired(
        &self,
        before: DateTime<Utc>,
    ) -> Result<Vec<ComplyPortfolio>, AppError>;
}

// ─── PortfolioItemRepository ────────────────────────────────────────────
#[async_trait]
pub trait PortfolioItemRepository: Send + Sync {
    async fn create_batch(
        &self,
        items: Vec<CreatePortfolioItemRow>,
    ) -> Result<Vec<ComplyPortfolioItem>, AppError>;

    async fn list_by_portfolio(
        &self,
        portfolio_id: Uuid,
    ) -> Result<Vec<ComplyPortfolioItem>, AppError>;

    async fn delete_by_portfolio(
        &self,
        portfolio_id: Uuid,
    ) -> Result<(), AppError>;
}

// ─── TranscriptRepository (Phase 3) ────────────────────────────────────
#[async_trait]
pub trait TranscriptRepository: Send + Sync {
    async fn create(
        &self,
        scope: FamilyScope,
        input: CreateTranscriptRow,
    ) -> Result<ComplyTranscript, AppError>;

    async fn find_by_id(
        &self,
        transcript_id: Uuid,
        scope: FamilyScope,
    ) -> Result<Option<ComplyTranscript>, AppError>;

    async fn list_by_student(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
    ) -> Result<Vec<ComplyTranscript>, AppError>;

    async fn update_status(
        &self,
        transcript_id: Uuid,
        status: &str,
        upload_id: Option<Uuid>,
        gpa_unweighted: Option<f64>,
        gpa_weighted: Option<f64>,
        error_message: Option<&str>,
    ) -> Result<ComplyTranscript, AppError>;

    async fn delete(
        &self,
        transcript_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;
}

// ─── CourseRepository (Phase 3) ─────────────────────────────────────────
#[async_trait]
pub trait CourseRepository: Send + Sync {
    async fn create(
        &self,
        scope: FamilyScope,
        input: CreateCourseRow,
    ) -> Result<ComplyCourse, AppError>;

    async fn list_by_student(
        &self,
        student_id: StudentId,
        scope: FamilyScope,
        params: &CourseListParams,
    ) -> Result<Vec<ComplyCourse>, AppError>;

    async fn update(
        &self,
        course_id: Uuid,
        scope: FamilyScope,
        updates: UpdateCourseRow,
    ) -> Result<ComplyCourse, AppError>;

    async fn delete(
        &self,
        course_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;
}
```

---

## §7 Adapter Interface

N/A — no external adapter. PDF generation is in-house via the `typst` crate. PDF rendering
happens within `GeneratePortfolioJob` and `GenerateTranscriptJob` background workers. The
generated PDF bytes are uploaded to `media::MediaService` for storage and CDN delivery.

---

## §8 Models (DTOs)

### §8.1 Request Types

```rust
// src/comply/models.rs

use serde::Deserialize;
use chrono::NaiveDate;

/// Body for PUT /v1/compliance/config
#[derive(Debug, Deserialize)]
pub struct UpsertFamilyConfigCommand {
    pub state_code: String,
    pub school_year_start: NaiveDate,
    pub school_year_end: NaiveDate,
    pub total_school_days: i16,
    pub custom_schedule_id: Option<Uuid>,
    pub gpa_scale: String,                 // "standard_4" | "weighted" | "custom"
    pub gpa_custom_config: Option<serde_json::Value>,
}

/// Body for POST /v1/compliance/schedules
#[derive(Debug, Deserialize)]
pub struct CreateScheduleCommand {
    pub name: String,
    pub school_days: Vec<bool>,            // Mon-Sun, 7 elements
    pub exclusion_periods: Vec<ExclusionPeriod>,
}

#[derive(Debug, Deserialize)]
pub struct ExclusionPeriod {
    pub start: NaiveDate,
    pub end: NaiveDate,
    pub label: String,
}

/// Body for PATCH /v1/compliance/schedules/:id
#[derive(Debug, Deserialize)]
pub struct UpdateScheduleCommand {
    pub name: Option<String>,
    pub school_days: Option<Vec<bool>>,
    pub exclusion_periods: Option<Vec<ExclusionPeriod>>,
}

/// Body for POST /v1/compliance/students/:id/attendance
#[derive(Debug, Deserialize)]
pub struct RecordAttendanceCommand {
    pub attendance_date: NaiveDate,
    pub status: String,                    // "present_full" | "present_partial" | "absent" | "not_applicable"
    pub duration_minutes: Option<i16>,
    pub notes: Option<String>,
}

/// Body for POST /v1/compliance/students/:id/attendance/bulk
#[derive(Debug, Deserialize)]
pub struct BulkRecordAttendanceCommand {
    pub records: Vec<RecordAttendanceCommand>,
}

/// Body for PATCH /v1/compliance/students/:id/attendance/:id
#[derive(Debug, Deserialize)]
pub struct UpdateAttendanceCommand {
    pub status: Option<String>,
    pub duration_minutes: Option<Option<i16>>,
    pub notes: Option<Option<String>>,
}

/// Body for POST /v1/compliance/students/:id/assessments
#[derive(Debug, Deserialize)]
pub struct CreateAssessmentCommand {
    pub title: String,
    pub subject: String,
    pub assessment_type: String,
    pub score: Option<f64>,
    pub max_score: Option<f64>,
    pub grade_letter: Option<String>,
    pub grade_points: Option<f64>,
    pub is_passing: Option<bool>,
    pub source_activity_id: Option<Uuid>,
    pub assessment_date: NaiveDate,
    pub notes: Option<String>,
}

/// Body for PATCH /v1/compliance/students/:id/assessments/:id
#[derive(Debug, Deserialize)]
pub struct UpdateAssessmentCommand {
    pub title: Option<String>,
    pub subject: Option<String>,
    pub score: Option<Option<f64>>,
    pub max_score: Option<Option<f64>>,
    pub grade_letter: Option<Option<String>>,
    pub grade_points: Option<Option<f64>>,
    pub is_passing: Option<Option<bool>>,
    pub assessment_date: Option<NaiveDate>,
    pub notes: Option<Option<String>>,
}

/// Body for POST /v1/compliance/students/:id/tests
#[derive(Debug, Deserialize)]
pub struct CreateTestScoreCommand {
    pub test_name: String,
    pub test_date: NaiveDate,
    pub grade_level: Option<i16>,
    pub scores: serde_json::Value,
    pub composite_score: Option<f64>,
    pub percentile: Option<i16>,
    pub notes: Option<String>,
}

/// Body for PATCH /v1/compliance/students/:id/tests/:id
#[derive(Debug, Deserialize)]
pub struct UpdateTestScoreCommand {
    pub test_name: Option<String>,
    pub test_date: Option<NaiveDate>,
    pub scores: Option<serde_json::Value>,
    pub composite_score: Option<Option<f64>>,
    pub percentile: Option<Option<i16>>,
    pub notes: Option<Option<String>>,
}

/// Body for POST /v1/compliance/students/:id/portfolios
#[derive(Debug, Deserialize)]
pub struct CreatePortfolioCommand {
    pub title: String,
    pub description: Option<String>,
    pub organization: String,             // "by_subject" | "chronological" | "by_student"
    pub date_range_start: NaiveDate,
    pub date_range_end: NaiveDate,
    pub include_attendance: bool,
    pub include_assessments: bool,
}

/// Body for POST /v1/compliance/students/:id/portfolios/:id/items
#[derive(Debug, Deserialize)]
pub struct AddPortfolioItemsCommand {
    pub items: Vec<PortfolioItemInput>,
}

#[derive(Debug, Deserialize)]
pub struct PortfolioItemInput {
    pub source_type: String,               // "activity" | "journal" | "project" | "reading_list" | "assessment"
    pub source_id: Uuid,
}

/// Body for POST /v1/compliance/students/:id/transcripts (Phase 3)
#[derive(Debug, Deserialize)]
pub struct CreateTranscriptCommand {
    pub title: String,
    pub grade_levels: Vec<String>,         // e.g., ["9", "10", "11"]
}

/// Body for POST /v1/compliance/students/:id/courses (Phase 3)
#[derive(Debug, Deserialize)]
pub struct CreateCourseCommand {
    pub title: String,
    pub subject: String,
    pub grade_level: i16,
    pub credits: f64,
    pub grade_letter: Option<String>,
    pub grade_points: Option<f64>,
    pub level: String,                     // "regular" | "honors" | "ap"
    pub school_year: String,
    pub semester: Option<String>,
}

/// Body for PATCH /v1/compliance/students/:id/courses/:id (Phase 3)
#[derive(Debug, Deserialize)]
pub struct UpdateCourseCommand {
    pub title: Option<String>,
    pub subject: Option<String>,
    pub credits: Option<f64>,
    pub grade_letter: Option<Option<String>>,
    pub grade_points: Option<Option<f64>>,
    pub level: Option<String>,
    pub semester: Option<Option<String>>,
}

/// Query params for GET /v1/compliance/students/:id/attendance
#[derive(Debug, Deserialize)]
pub struct AttendanceListParams {
    pub start_date: NaiveDate,
    pub end_date: NaiveDate,
    pub status: Option<String>,
    pub cursor: Option<String>,
    pub limit: Option<u8>,                 // Default 31, max 366
}

/// Query params for GET /v1/compliance/students/:id/attendance/summary
#[derive(Debug, Deserialize)]
pub struct AttendanceSummaryParams {
    pub start_date: NaiveDate,
    pub end_date: NaiveDate,
}

/// Query params for GET /v1/compliance/students/:id/assessments
#[derive(Debug, Deserialize)]
pub struct AssessmentListParams {
    pub subject: Option<String>,
    pub start_date: Option<NaiveDate>,
    pub end_date: Option<NaiveDate>,
    pub cursor: Option<String>,
    pub limit: Option<u8>,                 // Default 20, max 100
}

/// Query params for GET /v1/compliance/students/:id/tests
#[derive(Debug, Deserialize)]
pub struct TestListParams {
    pub cursor: Option<String>,
    pub limit: Option<u8>,
}

/// Query params for GET /v1/compliance/students/:id/courses (Phase 3)
#[derive(Debug, Deserialize)]
pub struct CourseListParams {
    pub grade_level: Option<i16>,
    pub school_year: Option<String>,
    pub cursor: Option<String>,
    pub limit: Option<u8>,
}

/// Query params for GET /v1/compliance/students/:id/gpa (Phase 3)
#[derive(Debug, Deserialize)]
pub struct GpaParams {
    pub scale: Option<String>,             // "standard_4" | "weighted" | "custom"
    pub grade_levels: Option<Vec<i16>>,    // filter by grade levels
}

/// Query params for GET /v1/compliance/students/:id/gpa/what-if (Phase 3)
#[derive(Debug, Deserialize)]
pub struct GpaWhatIfParams {
    pub additional_courses: Vec<WhatIfCourse>,
}

#[derive(Debug, Deserialize)]
pub struct WhatIfCourse {
    pub credits: f64,
    pub grade_points: f64,
    pub level: String,                     // "regular" | "honors" | "ap"
}
```

### §8.2 Response Types

```rust
use serde::Serialize;

/// Family compliance configuration.
#[derive(Debug, Serialize)]
pub struct FamilyConfigResponse {
    pub family_id: Uuid,
    pub state_code: String,
    pub state_name: String,
    pub school_year_start: NaiveDate,
    pub school_year_end: NaiveDate,
    pub total_school_days: i16,
    pub custom_schedule_id: Option<Uuid>,
    pub gpa_scale: String,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// State compliance requirements (from cache).
#[derive(Debug, Serialize)]
pub struct StateConfigResponse {
    pub state_code: String,
    pub state_name: String,
    pub notification_required: bool,
    pub notification_details: Option<String>,
    pub required_subjects: Vec<String>,
    pub assessment_required: bool,
    pub assessment_details: Option<String>,
    pub record_keeping_required: bool,
    pub record_keeping_details: Option<String>,
    pub attendance_required: bool,
    pub attendance_days: Option<i16>,
    pub attendance_hours: Option<i16>,
    pub attendance_details: Option<String>,
    pub regulation_level: String,
}

/// State config summary (for listing).
#[derive(Debug, Serialize)]
pub struct StateConfigSummaryResponse {
    pub state_code: String,
    pub state_name: String,
    pub regulation_level: String,
    pub attendance_required: bool,
    pub attendance_days: Option<i16>,
}

/// Custom schedule.
#[derive(Debug, Serialize)]
pub struct ScheduleResponse {
    pub id: Uuid,
    pub name: String,
    pub school_days: Vec<bool>,
    pub exclusion_periods: Vec<ExclusionPeriod>,
    pub created_at: DateTime<Utc>,
}

/// Single attendance record.
#[derive(Debug, Serialize)]
pub struct AttendanceResponse {
    pub id: Uuid,
    pub student_id: Uuid,
    pub attendance_date: NaiveDate,
    pub status: String,
    pub duration_minutes: Option<i16>,
    pub notes: Option<String>,
    pub is_auto: bool,
    pub manual_override: bool,
    pub created_at: DateTime<Utc>,
}

/// Paginated attendance list.
#[derive(Debug, Serialize)]
pub struct AttendanceListResponse {
    pub records: Vec<AttendanceResponse>,
    pub next_cursor: Option<String>,
}

/// Attendance summary with pace calculation.
#[derive(Debug, Serialize)]
pub struct AttendanceSummaryResponse {
    pub total_days: i32,
    pub present_full: i32,
    pub present_partial: i32,
    pub absent: i32,
    pub not_applicable: i32,
    pub total_hours: f64,
    pub state_required_days: Option<i16>,
    pub state_required_hours: Option<i16>,
    pub pace_status: Option<String>,       // "on_track" | "at_risk" | "behind"
    pub projected_total_days: Option<i32>,
}

/// Single assessment record.
#[derive(Debug, Serialize)]
pub struct AssessmentResponse {
    pub id: Uuid,
    pub student_id: Uuid,
    pub title: String,
    pub subject: String,
    pub assessment_type: String,
    pub score: Option<f64>,
    pub max_score: Option<f64>,
    pub grade_letter: Option<String>,
    pub grade_points: Option<f64>,
    pub is_passing: Option<bool>,
    pub assessment_date: NaiveDate,
    pub notes: Option<String>,
    pub created_at: DateTime<Utc>,
}

/// Paginated assessment list.
#[derive(Debug, Serialize)]
pub struct AssessmentListResponse {
    pub records: Vec<AssessmentResponse>,
    pub next_cursor: Option<String>,
}

/// Single test score.
#[derive(Debug, Serialize)]
pub struct TestScoreResponse {
    pub id: Uuid,
    pub student_id: Uuid,
    pub test_name: String,
    pub test_date: NaiveDate,
    pub grade_level: Option<i16>,
    pub scores: serde_json::Value,
    pub composite_score: Option<f64>,
    pub percentile: Option<i16>,
    pub notes: Option<String>,
    pub created_at: DateTime<Utc>,
}

/// Paginated test list.
#[derive(Debug, Serialize)]
pub struct TestListResponse {
    pub tests: Vec<TestScoreResponse>,
    pub next_cursor: Option<String>,
}

/// Portfolio details.
#[derive(Debug, Serialize)]
pub struct PortfolioResponse {
    pub id: Uuid,
    pub student_id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub organization: String,
    pub date_range_start: NaiveDate,
    pub date_range_end: NaiveDate,
    pub include_attendance: bool,
    pub include_assessments: bool,
    pub status: String,
    pub item_count: i32,
    pub generated_at: Option<DateTime<Utc>>,
    pub expires_at: Option<DateTime<Utc>>,
    pub items: Option<Vec<PortfolioItemResponse>>,
    pub created_at: DateTime<Utc>,
}

/// Portfolio summary (for listing).
#[derive(Debug, Serialize)]
pub struct PortfolioSummaryResponse {
    pub id: Uuid,
    pub title: String,
    pub status: String,
    pub item_count: i32,
    pub date_range_start: NaiveDate,
    pub date_range_end: NaiveDate,
    pub generated_at: Option<DateTime<Utc>>,
    pub expires_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

/// Portfolio item (cached display data).
#[derive(Debug, Serialize)]
pub struct PortfolioItemResponse {
    pub id: Uuid,
    pub source_type: String,
    pub source_id: Uuid,
    pub display_order: i16,
    pub cached_title: String,
    pub cached_subject: Option<String>,
    pub cached_date: NaiveDate,
    pub cached_description: Option<String>,
}

/// GPA calculation result. (Phase 3)
#[derive(Debug, Serialize)]
pub struct GpaResponse {
    pub unweighted_gpa: f64,
    pub weighted_gpa: f64,
    pub total_credits: f64,
    pub total_courses: i32,
    pub by_grade_level: Vec<GpaGradeLevelResponse>,
}

#[derive(Debug, Serialize)]
pub struct GpaGradeLevelResponse {
    pub grade_level: i16,
    pub unweighted: f64,
    pub weighted: f64,
    pub credits: f64,
}

/// GPA history by term. (Phase 3)
#[derive(Debug, Serialize)]
pub struct GpaTermResponse {
    pub school_year: String,
    pub semester: Option<String>,
    pub unweighted_gpa: f64,
    pub weighted_gpa: f64,
    pub credits: f64,
    pub course_count: i32,
}

/// Transcript details. (Phase 3)
#[derive(Debug, Serialize)]
pub struct TranscriptResponse {
    pub id: Uuid,
    pub student_id: Uuid,
    pub title: String,
    pub student_name: String,
    pub grade_levels: Vec<String>,
    pub status: String,
    pub gpa_unweighted: Option<f64>,
    pub gpa_weighted: Option<f64>,
    pub courses: Option<Vec<CourseResponse>>,
    pub generated_at: Option<DateTime<Utc>>,
    pub expires_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

/// Transcript summary. (Phase 3)
#[derive(Debug, Serialize)]
pub struct TranscriptSummaryResponse {
    pub id: Uuid,
    pub title: String,
    pub status: String,
    pub grade_levels: Vec<String>,
    pub generated_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

/// Course record. (Phase 3)
#[derive(Debug, Serialize)]
pub struct CourseResponse {
    pub id: Uuid,
    pub student_id: Uuid,
    pub title: String,
    pub subject: String,
    pub grade_level: i16,
    pub credits: f64,
    pub grade_letter: Option<String>,
    pub grade_points: Option<f64>,
    pub level: String,
    pub school_year: String,
    pub semester: Option<String>,
    pub created_at: DateTime<Utc>,
}

/// Paginated course list. (Phase 3)
#[derive(Debug, Serialize)]
pub struct CourseListResponse {
    pub courses: Vec<CourseResponse>,
    pub next_cursor: Option<String>,
}

/// Compliance dashboard overview.
#[derive(Debug, Serialize)]
pub struct ComplianceDashboardResponse {
    pub family_config: Option<FamilyConfigResponse>,
    pub students: Vec<StudentComplianceSummary>,
}

#[derive(Debug, Serialize)]
pub struct StudentComplianceSummary {
    pub student_id: Uuid,
    pub student_name: String,
    pub attendance_summary: AttendanceSummaryResponse,
    pub recent_assessments_count: i32,
    pub recent_tests_count: i32,
    pub active_portfolios: Vec<PortfolioSummaryResponse>,
    pub pace_status: Option<String>,
}
```

---

## §9 Background Jobs

All comply:: background jobs use sidekiq-rs `[ARCH §12]`.

### §9.1 SyncStateConfigsJob

Syncs state compliance requirements from `discover::DiscoveryService` into
`comply_state_configs`. Runs daily to pick up any updates to state legal guides.

```rust
/// Sync state configs from discover:: into comply_state_configs.
/// Queue: Low. Schedule: daily at 4:00 AM UTC.
pub struct SyncStateConfigsJob;

impl SyncStateConfigsJob {
    async fn perform(&self, ctx: &JobContext) -> Result<(), JobError> {
        // 1. Call discover::DiscoveryService.list_state_guides()
        // 2. For each state, call discover::DiscoveryService.get_state_requirements(state_code)
        // 3. Upsert into comply_state_configs
        // 4. Log count of updated/inserted rows
    }
}
```

**Schedule**: `"0 4 * * *"` (daily at 4:00 AM UTC)

### §9.2 GeneratePortfolioJob

Generates a portfolio PDF asynchronously using the `typst` crate.

```rust
/// Generate a portfolio PDF.
/// Queue: Default. Trigger: on-demand (POST .../generate).
pub struct GeneratePortfolioJob {
    pub portfolio_id: Uuid,
    pub family_id: FamilyId,
}

impl GeneratePortfolioJob {
    async fn perform(&self, ctx: &JobContext) -> Result<(), JobError> {
        // 1. Load portfolio + items from repository
        // 2. Load attendance summary (if include_attendance)
        // 3. Load assessment records (if include_assessments)
        // 4. Load student + family info from iam::IamService
        // 5. Render PDF via typst crate (cover page, TOC, body, summaries)
        // 6. Upload PDF to media::MediaService
        // 7. Update portfolio status: generating → ready (or failed)
        // 8. Set expires_at = now() + 90 days
        // 9. Publish PortfolioGenerated event
    }
}
```

**Retry policy**: Max 3 attempts with exponential backoff. On final failure, set
status = `failed` with error_message.

### §9.3 GenerateTranscriptJob (Phase 3)

Generates a transcript PDF asynchronously.

```rust
/// Generate a transcript PDF.
/// Queue: Default. Trigger: on-demand (POST .../generate). Phase 3.
pub struct GenerateTranscriptJob {
    pub transcript_id: Uuid,
    pub family_id: FamilyId,
}

impl GenerateTranscriptJob {
    async fn perform(&self, ctx: &JobContext) -> Result<(), JobError> {
        // 1. Load transcript + courses from repository
        // 2. Calculate GPA (snapshot at generation time)
        // 3. Load student info from iam::IamService
        // 4. Render PDF via typst crate (standard transcript format)
        // 5. Upload PDF to media::MediaService
        // 6. Update transcript: status → ready, snapshot GPA values
        // 7. Set expires_at = now() + 90 days
        // 8. Publish TranscriptGenerated event
    }
}
```

### §9.4 AttendanceThresholdCheckJob

Checks attendance pace against state requirements for all configured families.

```rust
/// Check attendance pace against state requirements.
/// Queue: Low. Schedule: weekly on Sundays at 5:00 AM UTC.
pub struct AttendanceThresholdCheckJob;

impl AttendanceThresholdCheckJob {
    async fn perform(&self, ctx: &JobContext) -> Result<(), JobError> {
        // 1. Load all comply_family_configs
        // 2. For each family + student:
        //    a. Load comply_state_configs for family's state
        //    b. Calculate pace (§12)
        //    c. If at_risk or behind → publish AttendanceThresholdWarning event
    }
}
```

**Schedule**: `"0 5 * * 0"` (weekly on Sundays at 5:00 AM UTC)

---

## §10 GPA Calculation Algorithm (Deep Dive — Phase 3)

### Standard 4.0 Unweighted Scale

```
unweighted_gpa = SUM(grade_points * credits) / SUM(credits)
```

Where `grade_points` is the standard 4.0 mapping:
- A+ / A = 4.0, A- = 3.7
- B+ = 3.3, B = 3.0, B- = 2.7
- C+ = 2.3, C = 2.0, C- = 1.7
- D+ = 1.3, D = 1.0, D- = 0.7
- F = 0.0

### Weighted Scale

```
weighted_gpa = SUM(adjusted_grade_points * credits) / SUM(credits)
```

Where `adjusted_grade_points = grade_points + boost`:
- Regular courses: +0.0 boost
- Honors courses: +0.5 boost (max 4.5)
- AP courses: +1.0 boost (max 5.0)

### Custom Scale

Families with `gpa_scale = 'custom'` define their own mapping via JSONB in
`comply_family_configs.gpa_custom_config`:

```json
{
    "scale_name": "Pass/Fail Weighted",
    "mappings": {
        "P": 4.0,
        "HP": 4.5,
        "F": 0.0
    }
}
```

### Computation

GPA is computed **on-the-fly** from `comply_courses` — it is NOT stored as a persistent
column. This is O(n) where n = number of courses (typically < 50 for a high school student).
GPA is snapshotted only during transcript generation (stored in `comply_transcripts.snapshot_gpa_*`).

```rust
// src/comply/domain/gpa.rs

/// Pure computation — no database access, no side effects.
pub struct GpaCalculator;

impl GpaCalculator {
    /// Calculate GPA from a list of courses.
    pub fn calculate(
        courses: &[ComplyCourse],
        scale: GpaScale,
        custom_config: Option<&serde_json::Value>,
    ) -> GpaResult {
        let mut total_weighted_points = 0.0;
        let mut total_unweighted_points = 0.0;
        let mut total_credits = 0.0;

        for course in courses {
            if let Some(gp) = course.grade_points {
                let credits = course.credits as f64;
                total_unweighted_points += gp as f64 * credits;

                let boost = match course.level.as_str() {
                    "honors" => 0.5,
                    "ap" => 1.0,
                    _ => 0.0,
                };
                total_weighted_points += (gp as f64 + boost) * credits;
                total_credits += credits;
            }
        }

        if total_credits == 0.0 {
            return GpaResult {
                unweighted: 0.0,
                weighted: 0.0,
                total_credits: 0.0,
            };
        }

        GpaResult {
            unweighted: total_unweighted_points / total_credits,
            weighted: total_weighted_points / total_credits,
            total_credits,
        }
    }
}

pub struct GpaResult {
    pub unweighted: f64,
    pub weighted: f64,
    pub total_credits: f64,
}
```

---

## §11 Portfolio Generation Pipeline (Deep Dive)

### End-to-End Flow

```
1. Parent creates portfolio (POST .../portfolios)
   → status: configuring

2. Parent adds items (POST .../portfolios/:id/items)
   → Fetches display data from learn::LearningService
   → Caches title, subject, date, description, attachments in comply_portfolio_items
   → Portfolio remains in configuring status

3. Parent triggers generation (POST .../portfolios/:id/generate)
   → Validates: portfolio has items, portfolio is in configuring status
   → Transitions status: configuring → generating
   → Enqueues GeneratePortfolioJob

4. GeneratePortfolioJob runs (background)
   a. Load portfolio + items from repository
   b. Load attendance summary for date range (if include_attendance)
   c. Load assessment records for date range (if include_assessments)
   d. Load student name + family info from iam::IamService
   e. Render PDF via typst crate:
      - Cover page: student name, family name, date range, school year
      - Table of contents
      - Body sections (organized by portfolio.organization):
        - by_subject: group items by cached_subject
        - chronological: sort items by cached_date
        - by_student: group by student (multi-student portfolios)
      - Attendance summary table (if included)
      - Assessment summary table (if included)
   f. Upload PDF bytes to media::MediaService (context: data_export)
   g. Update portfolio: status → ready, upload_id, generated_at, expires_at
   h. Publish PortfolioGenerated event → notify::

5. Parent downloads PDF (GET .../portfolios/:id/download)
   → Calls media::MediaService.presigned_get() for download URL
   → Returns 302 redirect to presigned URL
```

### State Machine

```
                    ┌───────────────┐
                    │  configuring  │ ← initial state (add/remove items here)
                    └──────┬────────┘
                           │ POST .../generate
                           ▼
                    ┌───────────────┐
               ┌───▶│  generating   │ ← GeneratePortfolioJob running
               │    └──────┬────────┘
               │           │
               │     ┌─────┴─────┐
               │     │           │
               │     ▼           ▼
               │ ┌────────┐  ┌────────┐
               │ │ ready  │  │ failed │
               │ └───┬────┘  └────────┘
               │     │           │
               │     │ 90 days   │ retry (max 3)
               │     ▼           │
               │ ┌─────────┐    │
               │ │ expired │    │
               │ └─────────┘    │
               └────────────────┘
```

### PDF Format

Generated via the `typst` crate (Rust-native typesetting):

1. **Cover page**: Student name, family name, date range, school name (if configured),
   state, school year
2. **Table of contents**: Auto-generated from body sections
3. **Body sections**: Learning artifacts grouped/sorted per `organization` setting
   - Each item: title, date, subject, description, embedded images (if any)
4. **Attendance summary** (optional): Table with day counts by status, total hours,
   state requirement comparison
5. **Assessment summary** (optional): Table with assessment scores by subject

### Expiration

Generated PDFs expire after 90 days (`expires_at = generated_at + 90 days`). After
expiration, the portfolio status transitions to `expired` and the media upload is eligible
for cleanup. Parents can re-generate to create a fresh PDF.

---

## §12 Attendance Threshold Logic (Deep Dive)

### State Requirements

State requirements come from `comply_state_configs` (synced from `discover::`):

- **Day-based**: State requires a minimum number of school days per year
  (e.g., California: 175 days, Texas: 170 days)
- **Hour-based**: Some states require minimum hours instead of or in addition to days
  (e.g., North Carolina: 1080 hours for high school)
- **Both**: Some states require both days and hours
- **None**: Some states have no attendance requirement

### Pace Calculation

```rust
// src/comply/domain/attendance.rs

/// Calculate attendance pace against state requirements.
pub fn calculate_pace(
    actual_present_days: i32,
    elapsed_school_days: i32,
    total_school_days: i32,
    state_required_days: Option<i16>,
) -> PaceStatus {
    let required = match state_required_days {
        Some(r) => r as i32,
        None => return PaceStatus::NotApplicable,
    };

    if elapsed_school_days == 0 {
        return PaceStatus::OnTrack; // school year hasn't started
    }

    // Project total present days based on current pace
    let pace_rate = actual_present_days as f64 / elapsed_school_days as f64;
    let projected_total = (pace_rate * total_school_days as f64).round() as i32;

    if projected_total >= required {
        PaceStatus::OnTrack
    } else if projected_total >= (required as f64 * 0.9).round() as i32 {
        PaceStatus::AtRisk // within 10% of requirement
    } else {
        PaceStatus::Behind
    }
}

pub enum PaceStatus {
    OnTrack,
    AtRisk,   // projected total within 10% of requirement
    Behind,   // projected total below 90% of requirement
    NotApplicable, // state has no attendance requirement
}
```

### Custom Schedule Interaction

When a family has a custom schedule, `elapsed_school_days` is computed by counting
school days according to the schedule:

```rust
/// Count school days between two dates using a custom schedule.
pub fn count_school_days(
    start: NaiveDate,
    end: NaiveDate,
    school_days: &[bool; 7],         // Mon=0 through Sun=6
    exclusion_periods: &[ExclusionPeriod],
) -> i32 {
    let mut count = 0;
    let mut date = start;
    while date <= end {
        let weekday = date.weekday().num_days_from_monday() as usize;
        if school_days[weekday] && !is_excluded(date, exclusion_periods) {
            count += 1;
        }
        date += chrono::Duration::days(1);
    }
    count
}
```

### Weekly Threshold Check

`AttendanceThresholdCheckJob` runs weekly (Sundays at 5:00 AM UTC). For each family
with a `comply_family_config`:

1. Load the family's state config from `comply_state_configs`
2. For each student in the family:
   - Count actual present days within the school year
   - Count elapsed school days (using custom schedule if configured)
   - Calculate pace status
3. If `AtRisk` or `Behind` → publish `AttendanceThresholdWarning` event

The `AttendanceThresholdWarning` event is consumed by `notify::` to send an in-app
notification and optional email to the parent.

---

## §13 Transcript Generation (Deep Dive — Phase 3)

### Data Source

Transcripts use `comply_courses` data — not `learn_*` tables directly. Parents define
courses with titles, credits, grades, and levels. This separation exists because:

1. A "course" is a parent-defined academic unit, not a 1:1 mapping to logged activities
2. Parents may aggregate many activities into one course
3. Transcript grades are deliberate parental decisions, not auto-computed

### PDF Format

Standard high school transcript format:

1. **Header**: Student name, school name (family name or configured name), address
   (optional), date of birth (optional)
2. **Course table by grade level**: For each grade (9-12):
   - Course title, subject, credits, grade, level (regular/honors/AP)
   - Semester GPA subtotals
3. **Cumulative GPA**: Both unweighted and weighted (snapshotted at generation time)
4. **Total credits earned**
5. **Footer**: Generation date, parent signature line (blank)

### GPA Snapshot

At generation time, GPA is calculated from `comply_courses` and snapshotted into
`comply_transcripts.snapshot_gpa_unweighted` and `snapshot_gpa_weighted`. This ensures
the transcript PDF always reflects the GPA at the time it was generated, even if courses
are later modified.

---

## §14 Domain Aggregates

Since `comply::` is a complex domain `[ARCH §4.5]`, the `domain/` subdirectory contains
aggregate roots and value objects that enforce invariants structurally.

### §14.1 Attendance Aggregate

```rust
// src/comply/domain/attendance.rs

/// Attendance aggregate — enforces attendance recording invariants.
pub struct Attendance;

impl Attendance {
    /// Validate a new attendance record.
    pub fn validate_record(
        date: NaiveDate,
        status: &str,
        duration_minutes: Option<i16>,
        today: NaiveDate,
    ) -> Result<(), ComplianceError> {
        // No future dates
        if date > today {
            return Err(ComplianceError::FutureAttendanceDate);
        }

        // Valid status
        if !["present_full", "present_partial", "absent", "not_applicable"].contains(&status) {
            return Err(ComplianceError::InvalidAttendanceStatus);
        }

        // Duration required for partial attendance
        if status == "present_partial" && duration_minutes.is_none() {
            return Err(ComplianceError::DurationRequiredForPartial);
        }

        // No negative duration
        if let Some(d) = duration_minutes {
            if d < 0 {
                return Err(ComplianceError::NegativeDuration);
            }
        }

        Ok(())
    }

    /// Determine precedence: manual entries override auto-generated ones.
    pub fn should_override(existing_is_auto: bool, new_is_manual: bool) -> bool {
        // Manual always wins. Auto never overrides manual.
        new_is_manual || existing_is_auto
    }
}
```

### §14.2 Portfolio Aggregate

```rust
// src/comply/domain/portfolio.rs

/// Portfolio aggregate — state machine with invariant enforcement.
pub struct Portfolio;

impl Portfolio {
    /// Validate state transition.
    pub fn validate_transition(
        current_status: &str,
        target_status: &str,
    ) -> Result<(), ComplianceError> {
        let valid = match (current_status, target_status) {
            ("configuring", "generating") => true,
            ("generating", "ready") => true,
            ("generating", "failed") => true,
            ("failed", "generating") => true,    // retry
            ("ready", "expired") => true,
            _ => false,
        };

        if !valid {
            return Err(ComplianceError::InvalidPortfolioTransition {
                from: current_status.to_string(),
                to: target_status.to_string(),
            });
        }

        Ok(())
    }

    /// Validate portfolio can be generated.
    pub fn validate_generate(
        status: &str,
        item_count: i32,
        retry_count: i16,
        max_retries: i16,
    ) -> Result<(), ComplianceError> {
        if status != "configuring" && status != "failed" {
            return Err(ComplianceError::PortfolioNotConfiguring);
        }

        if item_count == 0 {
            return Err(ComplianceError::EmptyPortfolio);
        }

        if status == "failed" && retry_count >= max_retries {
            return Err(ComplianceError::MaxRetriesExceeded);
        }

        Ok(())
    }
}
```

### §14.3 Transcript Aggregate (Phase 3)

```rust
// src/comply/domain/transcript.rs

/// Transcript aggregate — state machine (same pattern as Portfolio). Phase 3.
pub struct Transcript;

impl Transcript {
    /// Validate state transition (same transitions as Portfolio).
    pub fn validate_transition(
        current_status: &str,
        target_status: &str,
    ) -> Result<(), ComplianceError> {
        // Same valid transitions as Portfolio
        Portfolio::validate_transition(current_status, target_status)
    }
}
```

### §14.4 GpaCalculator Value Object (Phase 3)

Defined in §10. Pure computation function with no side effects or state.

---

## §15 Events comply:: Publishes

Defined in `src/comply/events.rs`. `[CODING §8.4]`

```rust
// src/comply/events.rs

use crate::shared::types::{FamilyId, StudentId};
use uuid::Uuid;

/// Published when a portfolio PDF has been generated and is ready for download.
/// Consumed by notify:: (in-app notification + optional email).
#[derive(Clone, Debug)]
pub struct PortfolioGenerated {
    pub family_id: FamilyId,
    pub student_id: StudentId,
    pub portfolio_id: Uuid,
    pub portfolio_title: String,
}
impl DomainEvent for PortfolioGenerated {}

/// Published when a transcript PDF has been generated and is ready for download. (Phase 3)
/// Consumed by notify:: (in-app notification + optional email).
#[derive(Clone, Debug)]
pub struct TranscriptGenerated {
    pub family_id: FamilyId,
    pub student_id: StudentId,
    pub transcript_id: Uuid,
}
impl DomainEvent for TranscriptGenerated {}

/// Published when a student's attendance pace falls below state requirements.
/// Consumed by notify:: (in-app + email warning to parent).
#[derive(Clone, Debug)]
pub struct AttendanceThresholdWarning {
    pub family_id: FamilyId,
    pub student_id: StudentId,
    pub student_name: String,
    pub pace_status: String,               // "at_risk" | "behind"
    pub actual_days: i32,
    pub expected_days: i32,
    pub required_days: i16,
}
impl DomainEvent for AttendanceThresholdWarning {}
```

---

## §16 Error Types + HTTP Mapping

All compliance errors use `thiserror` and map to HTTP status codes via `AppError`. Internal
details are logged but never exposed in API responses. `[CODING §2.2, §5.2]`

```rust
// src/comply/errors.rs

use thiserror::Error;

#[derive(Debug, Error)]
pub enum ComplianceError {
    // ─── Configuration Errors ──────────────────────────────────────────

    #[error("Family config not found")]
    FamilyConfigNotFound,

    #[error("Invalid state code")]
    InvalidStateCode,

    #[error("State config not found")]
    StateConfigNotFound,

    #[error("Invalid school year date range")]
    InvalidSchoolYearRange,

    // ─── Schedule Errors ───────────────────────────────────────────────

    #[error("Schedule not found")]
    ScheduleNotFound,

    #[error("Schedule in use by family config")]
    ScheduleInUse,

    #[error("Invalid school days array — must have 7 elements")]
    InvalidSchoolDaysArray,

    // ─── Attendance Errors ─────────────────────────────────────────────

    #[error("Attendance record not found")]
    AttendanceNotFound,

    #[error("Cannot record attendance for a future date")]
    FutureAttendanceDate,

    #[error("Invalid attendance status")]
    InvalidAttendanceStatus,

    #[error("Duration is required for partial attendance")]
    DurationRequiredForPartial,

    #[error("Duration cannot be negative")]
    NegativeDuration,

    #[error("Bulk attendance exceeds maximum of 31 records")]
    BulkAttendanceLimitExceeded,

    // ─── Assessment Errors ─────────────────────────────────────────────

    #[error("Assessment record not found")]
    AssessmentNotFound,

    #[error("Invalid assessment type")]
    InvalidAssessmentType,

    // ─── Test Score Errors ─────────────────────────────────────────────

    #[error("Test score not found")]
    TestScoreNotFound,

    // ─── Portfolio Errors ──────────────────────────────────────────────

    #[error("Portfolio not found")]
    PortfolioNotFound,

    #[error("Portfolio is not in configuring status")]
    PortfolioNotConfiguring,

    #[error("Cannot generate an empty portfolio")]
    EmptyPortfolio,

    #[error("Invalid portfolio status transition")]
    InvalidPortfolioTransition { from: String, to: String },

    #[error("Portfolio has expired")]
    PortfolioExpired,

    #[error("Maximum retry attempts exceeded")]
    MaxRetriesExceeded,

    #[error("Portfolio item source not found in learn::")]
    PortfolioItemSourceNotFound,

    #[error("Duplicate item in portfolio")]
    DuplicatePortfolioItem,

    // ─── Transcript Errors (Phase 3) ───────────────────────────────────

    #[error("Transcript not found")]
    TranscriptNotFound,

    #[error("Course not found")]
    CourseNotFound,

    #[error("Invalid course level")]
    InvalidCourseLevel,

    // ─── Student Errors ────────────────────────────────────────────────

    #[error("Student not found in family")]
    StudentNotInFamily,

    // ─── Infrastructure ────────────────────────────────────────────────

    #[error("Database error")]
    DbError(#[from] sea_orm::DbErr),       // internal — NOT exposed in API

    #[error("PDF generation failed")]
    PdfGenerationFailed(String),           // internal — NOT exposed in API

    #[error("Media service error")]
    MediaServiceError(String),             // internal — NOT exposed in API
}
```

### Error-to-HTTP Mapping

| Error Variant | HTTP Status | Response Code | User-Facing Message |
|---------------|-------------|---------------|---------------------|
| `FamilyConfigNotFound` | `404 Not Found` | `config_not_found` | "Compliance configuration not found" |
| `InvalidStateCode` | `422 Unprocessable` | `invalid_state_code` | "Invalid state code" |
| `StateConfigNotFound` | `404 Not Found` | `state_not_found` | "State requirements not found" |
| `InvalidSchoolYearRange` | `422 Unprocessable` | `invalid_date_range` | "School year end must be after start" |
| `ScheduleNotFound` | `404 Not Found` | `schedule_not_found` | "Schedule not found" |
| `ScheduleInUse` | `409 Conflict` | `schedule_in_use` | "Schedule is in use and cannot be deleted" |
| `InvalidSchoolDaysArray` | `422 Unprocessable` | `invalid_school_days` | "School days must have exactly 7 elements" |
| `AttendanceNotFound` | `404 Not Found` | `attendance_not_found` | "Attendance record not found" |
| `FutureAttendanceDate` | `422 Unprocessable` | `future_date` | "Cannot record attendance for a future date" |
| `InvalidAttendanceStatus` | `422 Unprocessable` | `invalid_status` | "Invalid attendance status" |
| `DurationRequiredForPartial` | `422 Unprocessable` | `duration_required` | "Duration is required for partial attendance" |
| `NegativeDuration` | `422 Unprocessable` | `negative_duration` | "Duration cannot be negative" |
| `BulkAttendanceLimitExceeded` | `422 Unprocessable` | `bulk_limit_exceeded` | "Maximum 31 attendance records per request" |
| `AssessmentNotFound` | `404 Not Found` | `assessment_not_found` | "Assessment record not found" |
| `InvalidAssessmentType` | `422 Unprocessable` | `invalid_assessment_type` | "Invalid assessment type" |
| `TestScoreNotFound` | `404 Not Found` | `test_not_found` | "Test score not found" |
| `PortfolioNotFound` | `404 Not Found` | `portfolio_not_found` | "Portfolio not found" |
| `PortfolioNotConfiguring` | `409 Conflict` | `portfolio_not_configuring` | "Portfolio is not in a configurable state" |
| `EmptyPortfolio` | `422 Unprocessable` | `empty_portfolio` | "Cannot generate a portfolio with no items" |
| `InvalidPortfolioTransition` | `409 Conflict` | `invalid_transition` | "Invalid portfolio status transition" |
| `PortfolioExpired` | `410 Gone` | `portfolio_expired` | "Portfolio has expired — please regenerate" |
| `MaxRetriesExceeded` | `409 Conflict` | `max_retries` | "Maximum generation attempts exceeded" |
| `PortfolioItemSourceNotFound` | `422 Unprocessable` | `source_not_found` | "Learning artifact not found" |
| `DuplicatePortfolioItem` | `409 Conflict` | `duplicate_item` | "Item already exists in this portfolio" |
| `TranscriptNotFound` | `404 Not Found` | `transcript_not_found` | "Transcript not found" |
| `CourseNotFound` | `404 Not Found` | `course_not_found` | "Course not found" |
| `InvalidCourseLevel` | `422 Unprocessable` | `invalid_course_level` | "Invalid course level" |
| `StudentNotInFamily` | `404 Not Found` | `student_not_found` | "Student not found" |
| `DbError` | `500 Internal` | `internal_error` | "An unexpected error occurred" |
| `PdfGenerationFailed` | `500 Internal` | `internal_error` | "An unexpected error occurred" |
| `MediaServiceError` | `500 Internal` | `internal_error` | "An unexpected error occurred" |

---

## §17 Cross-Domain Interactions

### §17.1 comply:: Provides (Consumed by Other Domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `PortfolioGenerated` event | `notify::` | Domain event — in-app notification + email |
| `TranscriptGenerated` event (Phase 3) | `notify::` | Domain event — in-app notification + email |
| `AttendanceThresholdWarning` event | `notify::` | Domain event — in-app + email warning |

### §17.2 comply:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| `AuthContext` | `iam::` middleware | User identity on every request `[00-core §7.2]` |
| `FamilyScope` | `iam::` middleware | Family-scoped data access `[00-core §8]` |
| `RequirePremium` | `iam::` middleware | Premium tier gating — all endpoints `[00-core §13.3]` |
| `DiscoveryService::get_state_requirements()` | `discover::` | State requirements for auto-config `[03-discover §5, §8.2]` |
| `DiscoveryService::list_state_guides()` | `discover::` | State list for SyncStateConfigsJob `[03-discover §5]` |
| `StateGuideRequirements` type | `discover::` | Shared type — state requirements data `[03-discover §8.2]` |
| `LearningService` trait methods | `learn::` | Portfolio item data (activity/journal/project details) `[06-learn §5]` |
| `IamService` trait methods | `iam::` | Student name, family info for PDF covers `[01-iam §13.1]` |
| `MediaService::request_upload()` | `media::` | PDF upload after generation `[09-media §5]` |
| `MediaService::presigned_get()` | `media::` | Download URL for generated PDFs `[09-media §5]` |

### §17.3 Events comply:: Publishes

See §15 for full event definitions.

### §17.4 Events comply:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `ActivityLogged { family_id, student_id, activity_id, subject, duration_minutes }` | `learn::` | Auto-record attendance: upsert `comply_attendance` with `is_auto = true`. Only creates if no manual record exists for that date. `[06-learn §18.1, ARCH §4.6]` |
| `StudentDeleted { family_id, student_id }` | `iam::` | Cascade delete: remove all attendance, assessment, test, portfolio, transcript, and course records for the student. `[01-iam §13.3]` |
| `FamilyDeletionScheduled { family_id, delete_after }` | `iam::` | Cascade delete: remove `comply_family_configs`, all student compliance data. Data is permanently deleted (no soft-delete). `[01-iam §13.3]` |
| `SubscriptionCancelled { family_id, effective_at }` | `billing::` | No data deletion. Existing data is preserved (read-only). Generated PDFs remain downloadable until they expire. New compliance operations are blocked by `RequirePremium` extractor. `[10-billing §16.3, S§15.3]` |

```rust
// src/comply/event_handlers.rs

use crate::learn::events::ActivityLogged;
use crate::iam::events::{StudentDeleted, FamilyDeletionScheduled};
use crate::billing::events::SubscriptionCancelled;

// ─── learn:: events ────────────────────────────────────────────────────

pub struct ActivityLoggedHandler {
    compliance_service: Arc<dyn ComplianceService>,
}

#[async_trait]
impl DomainEventHandler<ActivityLogged> for ActivityLoggedHandler {
    async fn handle(&self, event: &ActivityLogged) -> Result<(), AppError> {
        self.compliance_service.handle_activity_logged(event).await
    }
}

// ─── iam:: events ──────────────────────────────────────────────────────

pub struct StudentDeletedHandler {
    compliance_service: Arc<dyn ComplianceService>,
}

#[async_trait]
impl DomainEventHandler<StudentDeleted> for StudentDeletedHandler {
    async fn handle(&self, event: &StudentDeleted) -> Result<(), AppError> {
        self.compliance_service.handle_student_deleted(event).await
    }
}

pub struct FamilyDeletionScheduledHandler {
    compliance_service: Arc<dyn ComplianceService>,
}

#[async_trait]
impl DomainEventHandler<FamilyDeletionScheduled> for FamilyDeletionScheduledHandler {
    async fn handle(&self, event: &FamilyDeletionScheduled) -> Result<(), AppError> {
        self.compliance_service.handle_family_deletion_scheduled(event).await
    }
}

// ─── billing:: events ──────────────────────────────────────────────────

pub struct SubscriptionCancelledHandler {
    compliance_service: Arc<dyn ComplianceService>,
}

#[async_trait]
impl DomainEventHandler<SubscriptionCancelled> for SubscriptionCancelledHandler {
    async fn handle(&self, event: &SubscriptionCancelled) -> Result<(), AppError> {
        self.compliance_service.handle_subscription_cancelled(event).await
    }
}
```

---

## §18 Phase Scope

### Phase 1 — Signal Recording `[S§19 Phase 1]`

**In scope**:
- 2 database tables (`comply_state_configs`, `comply_attendance`)
- 0 API endpoints
- 1 event handler (`ActivityLogged` → auto-record attendance)
- `SyncStateConfigsJob` background job (populate state configs)
- **0 endpoints, 2 tables, 1 event handler, 1 background job**

**Out of scope for Phase 1**: All API endpoints, family config, custom schedules, assessment
records, standardized tests, portfolios, transcripts, courses, GPA, dashboard.

### Phase 2 — Full Compliance `[S§19 Phase 2]`

**In scope**:
- +6 database tables (`comply_family_configs`, `comply_custom_schedules`,
  `comply_assessment_records`, `comply_standardized_tests`, `comply_portfolios`,
  `comply_portfolio_items`)
- ~28 API endpoints (config, schedules, attendance CRUD, assessments, tests, portfolios,
  dashboard)
- 3 additional event handlers (`StudentDeleted`, `FamilyDeletionScheduled`,
  `SubscriptionCancelled`)
- 2 additional background jobs (`GeneratePortfolioJob`, `AttendanceThresholdCheckJob`)
- Domain events: `PortfolioGenerated`, `AttendanceThresholdWarning`
- Portfolio generation pipeline (typst PDF)
- Attendance threshold checking
- **~28 endpoints, +6 tables, +3 event handlers, +2 background jobs**

### Phase 3 — Transcripts & GPA `[S§19 Phase 3]`

**In scope**:
- +2 database tables (`comply_transcripts`, `comply_courses`)
- ~14 API endpoints (transcripts, courses, GPA)
- 1 additional background job (`GenerateTranscriptJob`)
- Domain event: `TranscriptGenerated`
- GPA calculation (standard, weighted, custom)
- Transcript generation pipeline (typst PDF)
- GPA what-if calculator
- **~14 endpoints, +2 tables, +1 background job**

---

## §19 Verification Checklist

Each item is a testable assertion. Implementation is not complete until all assertions pass.

### Family Configuration

1. `PUT /v1/compliance/config` creates or updates family compliance config with valid state code
2. `PUT /v1/compliance/config` rejects invalid state codes with `422`
3. `PUT /v1/compliance/config` rejects invalid school year ranges (end before start) with `422`
4. `GET /v1/compliance/config` returns `null` for families without compliance config
5. State requirements are cached in `comply_state_configs` (51 rows, synced from `discover::`)

### Custom Schedules

6. `POST /v1/compliance/schedules` creates a custom schedule with valid 7-element school_days
7. `DELETE /v1/compliance/schedules/:id` rejects deletion of schedule in use by family config (`409`)
8. Custom schedule exclusion periods are respected in attendance threshold calculations

### Attendance

9. `POST .../attendance` creates a manual attendance record with `manual_override = true`
10. Manual attendance records override auto-generated records for the same date
11. `ActivityLogged` event creates auto-attendance with `is_auto = true` (only if no manual record exists)
12. UNIQUE constraint on `(family_id, student_id, attendance_date)` prevents duplicate attendance
13. `POST .../attendance` rejects future dates with `422`
14. `POST .../attendance` requires `duration_minutes` for `present_partial` status
15. `POST .../attendance/bulk` rejects more than 31 records per request
16. `GET .../attendance/summary` returns correct day counts by status
17. Attendance pace calculation correctly determines `on_track`, `at_risk`, `behind` status

### Assessment Records

18. `POST .../assessments` creates an assessment record with valid type
19. `GET .../assessments` filters by subject, date range correctly
20. Assessment records can reference `learn_activities.id` via `source_activity_id` (no FK)

### Standardized Tests

21. `POST .../tests` stores JSONB scores correctly for various test formats
22. `GET .../tests` returns test scores sorted by date descending

### Portfolios

23. Portfolio state machine enforces valid transitions: configuring → generating → ready/failed
24. `POST .../portfolios/:id/generate` rejects empty portfolios with `422`
25. `POST .../portfolios/:id/generate` rejects portfolios not in `configuring` status with `409`
26. Portfolio item selection caches display data from `learn::` at selection time
27. Generated portfolio PDF includes cover page with student and family information
28. Generated portfolio PDF respects organization setting (by_subject, chronological, by_student)
29. Portfolio PDFs expire after 90 days (status → expired)
30. `GET .../portfolios/:id/download` returns `410 Gone` for expired portfolios

### Transcripts (Phase 3)

31. GPA calculation uses standard 4.0 unweighted scale correctly
32. Weighted GPA applies +0.5 for honors, +1.0 for AP courses
33. GPA is computed on-the-fly (not stored) except during transcript generation (snapshotted)
34. GPA what-if calculator correctly projects GPA with hypothetical courses

### Cross-Domain

35. `ActivityLogged` event handler auto-records attendance (Phase 1)
36. `StudentDeleted` event cascades deletion of all student compliance data
37. `FamilyDeletionScheduled` event cascades deletion of all family compliance data
38. `SubscriptionCancelled` preserves existing data (no deletion) — just blocks new operations
39. `PortfolioGenerated` event is consumed by `notify::` for notification delivery
40. `AttendanceThresholdWarning` event is consumed by `notify::` for pace alerts

### Premium Gating

41. All compliance endpoints return `402 Premium Required` for free-tier families
42. Existing compliance data is preserved after subscription cancellation (read-only)

### Error Handling

43. Zero `.unwrap()` / `.expect()` in production code `[CODING §2.2]`
44. All errors use `ComplianceError` with `thiserror` `[CODING §2.2, §5.2]`
45. Internal error details (PDF generation errors, DB errors) are logged but never exposed in API responses

### Family-Scoping

46. Every query on family-data tables is family-scoped via `FamilyScope` `[CODING §2.4]`
47. `comply_state_configs` is NOT family-scoped (platform-authored reference data)
48. Student ownership is verified (student belongs to family) before all student-scoped operations

---

## §20 Module Structure

```
src/comply/
├── mod.rs                    # Re-exports, domain-level doc comments
├── handlers.rs               # ~28 Phase 2 + ~14 Phase 3 Axum route handlers
│                             #   (thin layer: extractors → service → response)
├── service.rs                # ComplianceServiceImpl — attendance recording,
│                             #   assessment aggregation, portfolio/transcript
│                             #   orchestration, threshold checking, event handling
├── repository.rs             # PgStateConfigRepo, PgFamilyConfigRepo,
│                             #   PgScheduleRepo, PgAttendanceRepo,
│                             #   PgAssessmentRepo, PgTestScoreRepo,
│                             #   PgPortfolioRepo, PgPortfolioItemRepo,
│                             #   PgTranscriptRepo (Phase 3), PgCourseRepo (Phase 3)
│                             #   All user-data queries family-scoped via FamilyScope
├── models.rs                 # Request/response types (serde + utoipa derives),
│                             #   internal types, config types
├── ports.rs                  # ComplianceService trait, all repository traits
│                             #   (CQRS separation: commands vs queries)
├── errors.rs                 # ComplianceError thiserror enum (~25 variants)
├── events.rs                 # PortfolioGenerated, TranscriptGenerated,
│                             #   AttendanceThresholdWarning [ARCH §4.6]
├── event_handlers.rs         # 4 DomainEventHandler structs:
│                             #   ActivityLoggedHandler,
│                             #   StudentDeletedHandler,
│                             #   FamilyDeletionScheduledHandler,
│                             #   SubscriptionCancelledHandler [ARCH §4.6]
├── jobs.rs                   # SyncStateConfigsJob, GeneratePortfolioJob,
│                             #   GenerateTranscriptJob (Phase 3),
│                             #   AttendanceThresholdCheckJob [ARCH §12]
├── domain/                   # Complex domain — aggregate roots + value objects
│   ├── mod.rs                #   [ARCH §4.5]
│   ├── attendance.rs         # Attendance aggregate: date validation, override
│   │                         #   precedence, pace calculation, school day counting
│   ├── portfolio.rs          # Portfolio aggregate: state machine, generate
│   │                         #   validation, item invariants
│   ├── transcript.rs         # Transcript aggregate: state machine (Phase 3)
│   ├── gpa.rs                # GpaCalculator value object: pure GPA computation
│   │                         #   (unweighted, weighted, custom) (Phase 3)
│   └── errors.rs             # Domain-layer error variants (if separated from
│                             #   top-level ComplianceError — optional)
└── entities/                 # SeaORM-generated — never hand-edit [CODING §6.3]
```

> **Complexity class**: Complex (has `domain/` subdirectory). `comply::` has attendance
> threshold invariants, portfolio/transcript state machines, GPA calculation rules, and
> manual-override-auto precedence logic that warrant structural enforcement via aggregate
> roots. `[ARCH §4.5]`
```
