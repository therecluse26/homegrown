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
| **Module path** | `internal/comply/` |
| **DB prefix** | `comply_` `[ARCH §5.1]` |
| **Complexity class** | Complex (has `domain/` subdirectory) — attendance thresholds, GPA calculation, state config rules, portfolio/transcript state machines `[ARCH §4.5]` |
| **CQRS** | Yes — write: attendance mark, assessment record / read: attendance summary, threshold check `[ARCH §4.7]` |
| **External adapter** | None — in-house PDF generation via `jung-kurt/gofpdf`; no external compliance vendor |
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
(upload + presigned URLs). Background job scheduling → asynq `[ARCH §12]`.

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
-- Migration: YYYYMMDD_000001_create_comply_tables.sql (goose, Phase 1)
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
-- Migration: YYYYMMDD_000002_create_comply_attendance.sql (goose, Phase 1)
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
-- Migration: YYYYMMDD_000003_create_comply_assessment_records.sql (goose, Phase 2)
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
-- Migration: YYYYMMDD_000004_create_comply_transcripts.sql (goose, Phase 3)
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
```json
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
- **Response**: `200 OK` → `[]StateConfigSummaryResponse`

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
- **Response**: `200 OK` → `[]ScheduleResponse`

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
```json
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
```json
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
- **Body**: `BulkRecordAttendanceCommand { Records: []RecordAttendanceCommand }`
- **Response**: `201 Created` → `[]AttendanceResponse`
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
```json
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
```json
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
- **Response**: `200 OK` → `[]PortfolioSummaryResponse`

**GET /v1/compliance/students/:student_id/portfolios/:id** — Get portfolio details.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `PortfolioResponse` (includes items)
- **Error codes**: `404`

**POST /v1/compliance/students/:student_id/portfolios/:id/items** — Add items to portfolio.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Body**: `AddPortfolioItemsCommand { Items: []PortfolioItemInput }`
```json
{
    "items": [
        { "source_type": "activity", "source_id": "uuid" },
        { "source_type": "journal", "source_id": "uuid" }
    ]
}
```
- **Response**: `200 OK` → `[]PortfolioItemResponse`
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
```json
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
- **Response**: `200 OK` → `[]TranscriptSummaryResponse`

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
```json
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
- **Query**: `GpaWhatIfParams { AdditionalCourses: []WhatIfCourse }`
- **Response**: `200 OK` → `GpaResponse` (projected GPA with hypothetical courses)

**GET /v1/compliance/students/:student_id/gpa/history** — GPA history by term.
- **Auth**: `RequirePremium` + `FamilyScope`
- **Response**: `200 OK` → `[]GpaTermResponse`

---

## §5 Service Interface

The `ComplianceService` trait defines all use cases. CQRS separation: command methods (writes
with side effects) are separated from query methods (reads). `[ARCH §4.7, CODING §8.2]`

```go
// internal/comply/ports.go

import (
    "context"

    "github.com/google/uuid"
    "homegrown-academy/internal/shared"
)

// ComplianceService defines all compliance use cases.
// CQRS separation: command methods (writes) vs query methods (reads).
type ComplianceService interface {

    // ─── Command side (writes, side effects) ────────────────────────────

    // UpsertFamilyConfig creates or updates family compliance configuration.
    UpsertFamilyConfig(ctx context.Context, cmd UpsertFamilyConfigCommand, scope shared.FamilyScope) (*FamilyConfigResponse, error)

    // CreateSchedule creates a custom schedule.
    CreateSchedule(ctx context.Context, cmd CreateScheduleCommand, scope shared.FamilyScope) (*ScheduleResponse, error)

    // UpdateSchedule updates a custom schedule.
    UpdateSchedule(ctx context.Context, scheduleID uuid.UUID, cmd UpdateScheduleCommand, scope shared.FamilyScope) (*ScheduleResponse, error)

    // DeleteSchedule deletes a custom schedule.
    DeleteSchedule(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) error

    // RecordAttendance records daily attendance for a student (manual entry).
    RecordAttendance(ctx context.Context, studentID uuid.UUID, cmd RecordAttendanceCommand, scope shared.FamilyScope) (*AttendanceResponse, error)

    // BulkRecordAttendance bulk records attendance for a student.
    BulkRecordAttendance(ctx context.Context, studentID uuid.UUID, cmd BulkRecordAttendanceCommand, scope shared.FamilyScope) ([]AttendanceResponse, error)

    // UpdateAttendance updates an attendance record.
    UpdateAttendance(ctx context.Context, studentID uuid.UUID, attendanceID uuid.UUID, cmd UpdateAttendanceCommand, scope shared.FamilyScope) (*AttendanceResponse, error)

    // DeleteAttendance deletes an attendance record.
    DeleteAttendance(ctx context.Context, studentID uuid.UUID, attendanceID uuid.UUID, scope shared.FamilyScope) error

    // CreateAssessment creates an assessment record.
    CreateAssessment(ctx context.Context, studentID uuid.UUID, cmd CreateAssessmentCommand, scope shared.FamilyScope) (*AssessmentResponse, error)

    // UpdateAssessment updates an assessment record.
    UpdateAssessment(ctx context.Context, studentID uuid.UUID, assessmentID uuid.UUID, cmd UpdateAssessmentCommand, scope shared.FamilyScope) (*AssessmentResponse, error)

    // DeleteAssessment deletes an assessment record.
    DeleteAssessment(ctx context.Context, studentID uuid.UUID, assessmentID uuid.UUID, scope shared.FamilyScope) error

    // CreateTestScore records a standardized test score.
    CreateTestScore(ctx context.Context, studentID uuid.UUID, cmd CreateTestScoreCommand, scope shared.FamilyScope) (*TestScoreResponse, error)

    // UpdateTestScore updates a test score.
    UpdateTestScore(ctx context.Context, studentID uuid.UUID, testID uuid.UUID, cmd UpdateTestScoreCommand, scope shared.FamilyScope) (*TestScoreResponse, error)

    // DeleteTestScore deletes a test score.
    DeleteTestScore(ctx context.Context, studentID uuid.UUID, testID uuid.UUID, scope shared.FamilyScope) error

    // CreatePortfolio creates a portfolio.
    CreatePortfolio(ctx context.Context, studentID uuid.UUID, cmd CreatePortfolioCommand, scope shared.FamilyScope) (*PortfolioResponse, error)

    // AddPortfolioItems adds items to a portfolio (caches display data from learn::).
    AddPortfolioItems(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, cmd AddPortfolioItemsCommand, scope shared.FamilyScope) ([]PortfolioItemResponse, error)

    // GeneratePortfolio triggers portfolio PDF generation (async — enqueues job).
    GeneratePortfolio(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, scope shared.FamilyScope) (*PortfolioResponse, error)

    // Phase 3: Transcript commands

    // CreateTranscript creates a transcript. (Phase 3)
    CreateTranscript(ctx context.Context, studentID uuid.UUID, cmd CreateTranscriptCommand, scope shared.FamilyScope) (*TranscriptResponse, error)

    // GenerateTranscript triggers transcript PDF generation (async — enqueues job). (Phase 3)
    GenerateTranscript(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) (*TranscriptResponse, error)

    // DeleteTranscript deletes a transcript. (Phase 3)
    DeleteTranscript(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) error

    // CreateCourse creates a course for transcript. (Phase 3)
    CreateCourse(ctx context.Context, studentID uuid.UUID, cmd CreateCourseCommand, scope shared.FamilyScope) (*CourseResponse, error)

    // UpdateCourse updates a course. (Phase 3)
    UpdateCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID, cmd UpdateCourseCommand, scope shared.FamilyScope) (*CourseResponse, error)

    // DeleteCourse deletes a course. (Phase 3)
    DeleteCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID, scope shared.FamilyScope) error

    // ─── Query side (reads, no side effects) ────────────────────────────

    // GetFamilyConfig gets family compliance configuration.
    GetFamilyConfig(ctx context.Context, scope shared.FamilyScope) (*FamilyConfigResponse, error)

    // ListStateConfigs lists all state requirements (from cache).
    ListStateConfigs(ctx context.Context) ([]StateConfigSummaryResponse, error)

    // GetStateConfig gets requirements for a specific state.
    GetStateConfig(ctx context.Context, stateCode string) (*StateConfigResponse, error)

    // ListSchedules lists family's custom schedules.
    ListSchedules(ctx context.Context, scope shared.FamilyScope) ([]ScheduleResponse, error)

    // ListAttendance lists attendance records for a student.
    ListAttendance(ctx context.Context, studentID uuid.UUID, params AttendanceListParams, scope shared.FamilyScope) (*AttendanceListResponse, error)

    // GetAttendanceSummary gets attendance summary for a student within a date range.
    GetAttendanceSummary(ctx context.Context, studentID uuid.UUID, params AttendanceSummaryParams, scope shared.FamilyScope) (*AttendanceSummaryResponse, error)

    // ListAssessments lists assessment records for a student.
    ListAssessments(ctx context.Context, studentID uuid.UUID, params AssessmentListParams, scope shared.FamilyScope) (*AssessmentListResponse, error)

    // ListTestScores lists standardized test scores for a student.
    ListTestScores(ctx context.Context, studentID uuid.UUID, params TestListParams, scope shared.FamilyScope) (*TestListResponse, error)

    // GetPortfolio gets portfolio details (includes items).
    GetPortfolio(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, scope shared.FamilyScope) (*PortfolioResponse, error)

    // ListPortfolios lists portfolios for a student.
    ListPortfolios(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]PortfolioSummaryResponse, error)

    // GetPortfolioDownloadURL gets presigned download URL for a portfolio PDF.
    GetPortfolioDownloadURL(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, scope shared.FamilyScope) (string, error)

    // GetDashboard gets compliance dashboard overview.
    GetDashboard(ctx context.Context, scope shared.FamilyScope) (*ComplianceDashboardResponse, error)

    // Phase 3: Transcript queries

    // GetTranscript gets transcript details (includes courses, GPA). (Phase 3)
    GetTranscript(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) (*TranscriptResponse, error)

    // ListTranscripts lists transcripts for a student. (Phase 3)
    ListTranscripts(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]TranscriptSummaryResponse, error)

    // GetTranscriptDownloadURL gets presigned download URL for a transcript PDF. (Phase 3)
    GetTranscriptDownloadURL(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) (string, error)

    // ListCourses lists courses for a student. (Phase 3)
    ListCourses(ctx context.Context, studentID uuid.UUID, params CourseListParams, scope shared.FamilyScope) (*CourseListResponse, error)

    // CalculateGPA calculates current GPA for a student. (Phase 3)
    CalculateGPA(ctx context.Context, studentID uuid.UUID, params GpaParams, scope shared.FamilyScope) (*GpaResponse, error)

    // CalculateGPAWhatIf calculates what-if GPA with hypothetical courses. (Phase 3)
    CalculateGPAWhatIf(ctx context.Context, studentID uuid.UUID, params GpaWhatIfParams, scope shared.FamilyScope) (*GpaResponse, error)

    // GetGPAHistory returns GPA history by term. (Phase 3)
    GetGPAHistory(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]GpaTermResponse, error)

    // ─── Event handlers ─────────────────────────────────────────────────

    // HandleActivityLogged handles ActivityLogged event: auto-record attendance. (Phase 1)
    // Consumed from learn::ActivityLogged. [06-learn §18.1]
    HandleActivityLogged(ctx context.Context, event *ActivityLogged) error

    // HandleStudentDeleted handles StudentDeleted event: cascade delete compliance data.
    // Consumed from iam::StudentDeleted. [01-iam §13.3]
    HandleStudentDeleted(ctx context.Context, event *StudentDeleted) error

    // HandleFamilyDeletionScheduled handles FamilyDeletionScheduled event: cascade delete all comply:: data.
    // Consumed from iam::FamilyDeletionScheduled. [01-iam §13.3]
    HandleFamilyDeletionScheduled(ctx context.Context, event *FamilyDeletionScheduled) error

    // HandleSubscriptionCancelled handles SubscriptionCancelled event: mark premium features as read-only.
    // Consumed from billing::SubscriptionCancelled. [10-billing §16.3]
    HandleSubscriptionCancelled(ctx context.Context, event *SubscriptionCancelled) error
}
```

### `ComplianceServiceImpl`

```go
// internal/comply/service.go

type ComplianceServiceImpl struct {
    familyConfigRepo FamilyConfigRepository
    scheduleRepo     ScheduleRepository
    attendanceRepo   AttendanceRepository
    assessmentRepo   AssessmentRepository
    testRepo         TestScoreRepository
    portfolioRepo    PortfolioRepository
    portfolioItemRepo PortfolioItemRepository
    stateConfigRepo  StateConfigRepository
    transcriptRepo   TranscriptRepository     // Phase 3
    courseRepo        CourseRepository          // Phase 3
    discoveryService DiscoveryService          // state requirements
    learningService  LearningService           // portfolio data
    iamService       IamService                // student/family data
    mediaService     MediaService              // PDF storage
    events           *EventBus
}
```

---

## §6 Repository Interfaces

All repository interfaces defined in `internal/comply/ports.go`. `StateConfigRepository` is NOT
family-scoped — it is platform-authored reference data. All other repositories are
family-scoped via `FamilyScope`. `[CODING §8.2]`

```go
// internal/comply/ports.go (continued)

import (
    "context"
    "time"

    "github.com/google/uuid"
    "homegrown-academy/internal/shared"
)

// ─── StateConfigRepository ──────────────────────────────────────────────
// NOT family-scoped — platform-authored reference data (51 rows).
type StateConfigRepository interface {
    // ListAll lists all state configs.
    ListAll(ctx context.Context) ([]ComplyStateConfig, error)

    // FindByStateCode gets config for a specific state.
    FindByStateCode(ctx context.Context, stateCode string) (*ComplyStateConfig, error)

    // Upsert upserts a state config (used by SyncStateConfigsJob).
    Upsert(ctx context.Context, config UpsertStateConfigRow) (*ComplyStateConfig, error)
}

// ─── FamilyConfigRepository ─────────────────────────────────────────────
// Family-scoped (family_id is PK).
type FamilyConfigRepository interface {
    Upsert(ctx context.Context, scope shared.FamilyScope, input UpsertFamilyConfigRow) (*ComplyFamilyConfig, error)
    FindByFamily(ctx context.Context, scope shared.FamilyScope) (*ComplyFamilyConfig, error)
    DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// ─── ScheduleRepository ─────────────────────────────────────────────────
// Family-scoped.
type ScheduleRepository interface {
    Create(ctx context.Context, scope shared.FamilyScope, input CreateScheduleRow) (*ComplyCustomSchedule, error)
    FindByID(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) (*ComplyCustomSchedule, error)
    ListByFamily(ctx context.Context, scope shared.FamilyScope) ([]ComplyCustomSchedule, error)
    Update(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope, updates UpdateScheduleRow) (*ComplyCustomSchedule, error)
    Delete(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) error
}

// ─── AttendanceRepository ───────────────────────────────────────────────
// Family-scoped. UNIQUE on (family_id, student_id, attendance_date).
type AttendanceRepository interface {
    Upsert(ctx context.Context, scope shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error)
    FindByID(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) (*ComplyAttendance, error)
    ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AttendanceListParams) ([]ComplyAttendance, error)
    Summarize(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, startDate time.Time, endDate time.Time) (*AttendanceSummaryRow, error)
    Update(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope, updates UpdateAttendanceRow) (*ComplyAttendance, error)
    Delete(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) error
    DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
    DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// ─── AssessmentRepository ───────────────────────────────────────────────
// Family-scoped.
type AssessmentRepository interface {
    Create(ctx context.Context, scope shared.FamilyScope, input CreateAssessmentRow) (*ComplyAssessmentRecord, error)
    FindByID(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) (*ComplyAssessmentRecord, error)
    ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AssessmentListParams) ([]ComplyAssessmentRecord, error)
    Update(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope, updates UpdateAssessmentRow) (*ComplyAssessmentRecord, error)
    Delete(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) error
    DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
}

// ─── TestScoreRepository ────────────────────────────────────────────────
// Family-scoped.
type TestScoreRepository interface {
    Create(ctx context.Context, scope shared.FamilyScope, input CreateTestScoreRow) (*ComplyStandardizedTest, error)
    ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *TestListParams) ([]ComplyStandardizedTest, error)
    Update(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope, updates UpdateTestScoreRow) (*ComplyStandardizedTest, error)
    Delete(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope) error
}

// ─── PortfolioRepository ────────────────────────────────────────────────
// Family-scoped.
type PortfolioRepository interface {
    Create(ctx context.Context, scope shared.FamilyScope, input CreatePortfolioRow) (*ComplyPortfolio, error)
    FindByID(ctx context.Context, portfolioID uuid.UUID, scope shared.FamilyScope) (*ComplyPortfolio, error)
    ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyPortfolio, error)
    UpdateStatus(ctx context.Context, portfolioID uuid.UUID, status string, uploadID *uuid.UUID, errorMessage *string) (*ComplyPortfolio, error)
    FindExpired(ctx context.Context, before time.Time) ([]ComplyPortfolio, error)
}

// ─── PortfolioItemRepository ────────────────────────────────────────────
type PortfolioItemRepository interface {
    CreateBatch(ctx context.Context, items []CreatePortfolioItemRow) ([]ComplyPortfolioItem, error)
    ListByPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]ComplyPortfolioItem, error)
    DeleteByPortfolio(ctx context.Context, portfolioID uuid.UUID) error
}

// ─── TranscriptRepository (Phase 3) ────────────────────────────────────
type TranscriptRepository interface {
    Create(ctx context.Context, scope shared.FamilyScope, input CreateTranscriptRow) (*ComplyTranscript, error)
    FindByID(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) (*ComplyTranscript, error)
    ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyTranscript, error)
    UpdateStatus(ctx context.Context, transcriptID uuid.UUID, status string, uploadID *uuid.UUID, gpaUnweighted *float64, gpaWeighted *float64, errorMessage *string) (*ComplyTranscript, error)
    Delete(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) error
}

// ─── CourseRepository (Phase 3) ─────────────────────────────────────────
type CourseRepository interface {
    Create(ctx context.Context, scope shared.FamilyScope, input CreateCourseRow) (*ComplyCourse, error)
    ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *CourseListParams) ([]ComplyCourse, error)
    Update(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope, updates UpdateCourseRow) (*ComplyCourse, error)
    Delete(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope) error
}
```

---

## §7 Adapter Interface

N/A — no external adapter. PDF generation is in-house via `jung-kurt/gofpdf`. PDF rendering
happens within `GeneratePortfolioJob` and `GenerateTranscriptJob` background workers. The
generated PDF bytes are uploaded to `media::MediaService` for storage and CDN delivery.

---

## §8 Models (DTOs)

### §8.1 Request Types

```go
// internal/comply/models.go

import (
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

// UpsertFamilyConfigCommand is the body for PUT /v1/compliance/config.
type UpsertFamilyConfigCommand struct {
    StateCode        string          `json:"state_code" validate:"required,len=2"`
    SchoolYearStart  time.Time       `json:"school_year_start" validate:"required"`
    SchoolYearEnd    time.Time       `json:"school_year_end" validate:"required"`
    TotalSchoolDays  int16           `json:"total_school_days" validate:"required,gt=0"`
    CustomScheduleID *uuid.UUID      `json:"custom_schedule_id"`
    GpaScale         string          `json:"gpa_scale" validate:"required,oneof=standard_4 weighted custom"`
    GpaCustomConfig  json.RawMessage `json:"gpa_custom_config"`
}

// CreateScheduleCommand is the body for POST /v1/compliance/schedules.
type CreateScheduleCommand struct {
    Name             string            `json:"name" validate:"required"`
    SchoolDays       []bool            `json:"school_days" validate:"required,len=7"` // Mon-Sun, 7 elements
    ExclusionPeriods []ExclusionPeriod `json:"exclusion_periods"`
}

type ExclusionPeriod struct {
    Start time.Time `json:"start" validate:"required"`
    End   time.Time `json:"end" validate:"required"`
    Label string    `json:"label" validate:"required"`
}

// UpdateScheduleCommand is the body for PATCH /v1/compliance/schedules/:id.
type UpdateScheduleCommand struct {
    Name             *string            `json:"name"`
    SchoolDays       *[]bool            `json:"school_days"`
    ExclusionPeriods *[]ExclusionPeriod `json:"exclusion_periods"`
}

// RecordAttendanceCommand is the body for POST /v1/compliance/students/:id/attendance.
type RecordAttendanceCommand struct {
    AttendanceDate  time.Time `json:"attendance_date" validate:"required"`
    Status          string    `json:"status" validate:"required,oneof=present_full present_partial absent not_applicable"`
    DurationMinutes *int16    `json:"duration_minutes"`
    Notes           *string   `json:"notes"`
}

// BulkRecordAttendanceCommand is the body for POST /v1/compliance/students/:id/attendance/bulk.
type BulkRecordAttendanceCommand struct {
    Records []RecordAttendanceCommand `json:"records" validate:"required,max=31,dive"`
}

// UpdateAttendanceCommand is the body for PATCH /v1/compliance/students/:id/attendance/:id.
type UpdateAttendanceCommand struct {
    Status          *string `json:"status"`
    DurationMinutes *int16  `json:"duration_minutes"`
    Notes           *string `json:"notes"`
}

// CreateAssessmentCommand is the body for POST /v1/compliance/students/:id/assessments.
type CreateAssessmentCommand struct {
    Title            string     `json:"title" validate:"required"`
    Subject          string     `json:"subject" validate:"required"`
    AssessmentType   string     `json:"assessment_type" validate:"required,oneof=test quiz project assignment presentation portfolio_piece other"`
    Score            *float64   `json:"score"`
    MaxScore         *float64   `json:"max_score"`
    GradeLetter      *string    `json:"grade_letter"`
    GradePoints      *float64   `json:"grade_points"`
    IsPassing        *bool      `json:"is_passing"`
    SourceActivityID *uuid.UUID `json:"source_activity_id"`
    AssessmentDate   time.Time  `json:"assessment_date" validate:"required"`
    Notes            *string    `json:"notes"`
}

// UpdateAssessmentCommand is the body for PATCH /v1/compliance/students/:id/assessments/:id.
type UpdateAssessmentCommand struct {
    Title          *string    `json:"title"`
    Subject        *string    `json:"subject"`
    Score          *float64   `json:"score"`
    MaxScore       *float64   `json:"max_score"`
    GradeLetter    *string    `json:"grade_letter"`
    GradePoints    *float64   `json:"grade_points"`
    IsPassing      *bool      `json:"is_passing"`
    AssessmentDate *time.Time `json:"assessment_date"`
    Notes          *string    `json:"notes"`
}

// CreateTestScoreCommand is the body for POST /v1/compliance/students/:id/tests.
type CreateTestScoreCommand struct {
    TestName       string          `json:"test_name" validate:"required"`
    TestDate       time.Time       `json:"test_date" validate:"required"`
    GradeLevel     *int16          `json:"grade_level"`
    Scores         json.RawMessage `json:"scores" validate:"required"`
    CompositeScore *float64        `json:"composite_score"`
    Percentile     *int16          `json:"percentile"`
    Notes          *string         `json:"notes"`
}

// UpdateTestScoreCommand is the body for PATCH /v1/compliance/students/:id/tests/:id.
type UpdateTestScoreCommand struct {
    TestName       *string          `json:"test_name"`
    TestDate       *time.Time       `json:"test_date"`
    Scores         *json.RawMessage `json:"scores"`
    CompositeScore *float64         `json:"composite_score"`
    Percentile     *int16           `json:"percentile"`
    Notes          *string          `json:"notes"`
}

// CreatePortfolioCommand is the body for POST /v1/compliance/students/:id/portfolios.
type CreatePortfolioCommand struct {
    Title              string    `json:"title" validate:"required"`
    Description        *string   `json:"description"`
    Organization       string    `json:"organization" validate:"required,oneof=by_subject chronological by_student"`
    DateRangeStart     time.Time `json:"date_range_start" validate:"required"`
    DateRangeEnd       time.Time `json:"date_range_end" validate:"required"`
    IncludeAttendance  bool      `json:"include_attendance"`
    IncludeAssessments bool      `json:"include_assessments"`
}

// AddPortfolioItemsCommand is the body for POST /v1/compliance/students/:id/portfolios/:id/items.
type AddPortfolioItemsCommand struct {
    Items []PortfolioItemInput `json:"items" validate:"required,dive"`
}

type PortfolioItemInput struct {
    SourceType string    `json:"source_type" validate:"required,oneof=activity journal project reading_list assessment"`
    SourceID   uuid.UUID `json:"source_id" validate:"required"`
}

// CreateTranscriptCommand is the body for POST /v1/compliance/students/:id/transcripts (Phase 3).
type CreateTranscriptCommand struct {
    Title       string   `json:"title" validate:"required"`
    GradeLevels []string `json:"grade_levels" validate:"required"` // e.g., ["9", "10", "11"]
}

// CreateCourseCommand is the body for POST /v1/compliance/students/:id/courses (Phase 3).
type CreateCourseCommand struct {
    Title       string   `json:"title" validate:"required"`
    Subject     string   `json:"subject" validate:"required"`
    GradeLevel  int16    `json:"grade_level" validate:"required"`
    Credits     float64  `json:"credits" validate:"required,gt=0"`
    GradeLetter *string  `json:"grade_letter"`
    GradePoints *float64 `json:"grade_points"`
    Level       string   `json:"level" validate:"required,oneof=regular honors ap"`
    SchoolYear  string   `json:"school_year" validate:"required"`
    Semester    *string  `json:"semester" validate:"omitempty,oneof=fall spring summer full_year"`
}

// UpdateCourseCommand is the body for PATCH /v1/compliance/students/:id/courses/:id (Phase 3).
type UpdateCourseCommand struct {
    Title       *string  `json:"title"`
    Subject     *string  `json:"subject"`
    Credits     *float64 `json:"credits"`
    GradeLetter *string  `json:"grade_letter"`
    GradePoints *float64 `json:"grade_points"`
    Level       *string  `json:"level"`
    Semester    *string  `json:"semester"`
}

// AttendanceListParams holds query params for GET /v1/compliance/students/:id/attendance.
type AttendanceListParams struct {
    StartDate time.Time `query:"start_date" validate:"required"`
    EndDate   time.Time `query:"end_date" validate:"required"`
    Status    *string   `query:"status"`
    Cursor    *string   `query:"cursor"`
    Limit     *uint8    `query:"limit"` // Default 31, max 366
}

// AttendanceSummaryParams holds query params for GET /v1/compliance/students/:id/attendance/summary.
type AttendanceSummaryParams struct {
    StartDate time.Time `query:"start_date" validate:"required"`
    EndDate   time.Time `query:"end_date" validate:"required"`
}

// AssessmentListParams holds query params for GET /v1/compliance/students/:id/assessments.
type AssessmentListParams struct {
    Subject   *string    `query:"subject"`
    StartDate *time.Time `query:"start_date"`
    EndDate   *time.Time `query:"end_date"`
    Cursor    *string    `query:"cursor"`
    Limit     *uint8     `query:"limit"` // Default 20, max 100
}

// TestListParams holds query params for GET /v1/compliance/students/:id/tests.
type TestListParams struct {
    Cursor *string `query:"cursor"`
    Limit  *uint8  `query:"limit"`
}

// CourseListParams holds query params for GET /v1/compliance/students/:id/courses (Phase 3).
type CourseListParams struct {
    GradeLevel *int16  `query:"grade_level"`
    SchoolYear *string `query:"school_year"`
    Cursor     *string `query:"cursor"`
    Limit      *uint8  `query:"limit"`
}

// GpaParams holds query params for GET /v1/compliance/students/:id/gpa (Phase 3).
type GpaParams struct {
    Scale       *string `query:"scale"`        // "standard_4" | "weighted" | "custom"
    GradeLevels []int16 `query:"grade_levels"` // filter by grade levels
}

// GpaWhatIfParams holds query params for GET /v1/compliance/students/:id/gpa/what-if (Phase 3).
type GpaWhatIfParams struct {
    AdditionalCourses []WhatIfCourse `json:"additional_courses" validate:"required,dive"`
}

type WhatIfCourse struct {
    Credits     float64 `json:"credits" validate:"required,gt=0"`
    GradePoints float64 `json:"grade_points" validate:"required,gte=0"`
    Level       string  `json:"level" validate:"required,oneof=regular honors ap"`
}
```

### §8.2 Response Types

```go
// internal/comply/models.go (continued — response types)

import (
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

// FamilyConfigResponse represents family compliance configuration.
type FamilyConfigResponse struct {
    FamilyID         uuid.UUID  `json:"family_id"`
    StateCode        string     `json:"state_code"`
    StateName        string     `json:"state_name"`
    SchoolYearStart  time.Time  `json:"school_year_start"`
    SchoolYearEnd    time.Time  `json:"school_year_end"`
    TotalSchoolDays  int16      `json:"total_school_days"`
    CustomScheduleID *uuid.UUID `json:"custom_schedule_id"`
    GpaScale         string     `json:"gpa_scale"`
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
}

// StateConfigResponse represents state compliance requirements (from cache).
type StateConfigResponse struct {
    StateCode             string   `json:"state_code"`
    StateName             string   `json:"state_name"`
    NotificationRequired  bool     `json:"notification_required"`
    NotificationDetails   *string  `json:"notification_details"`
    RequiredSubjects      []string `json:"required_subjects"`
    AssessmentRequired    bool     `json:"assessment_required"`
    AssessmentDetails     *string  `json:"assessment_details"`
    RecordKeepingRequired bool     `json:"record_keeping_required"`
    RecordKeepingDetails  *string  `json:"record_keeping_details"`
    AttendanceRequired    bool     `json:"attendance_required"`
    AttendanceDays        *int16   `json:"attendance_days"`
    AttendanceHours       *int16   `json:"attendance_hours"`
    AttendanceDetails     *string  `json:"attendance_details"`
    RegulationLevel       string   `json:"regulation_level"`
}

// StateConfigSummaryResponse represents state config summary (for listing).
type StateConfigSummaryResponse struct {
    StateCode          string `json:"state_code"`
    StateName          string `json:"state_name"`
    RegulationLevel    string `json:"regulation_level"`
    AttendanceRequired bool   `json:"attendance_required"`
    AttendanceDays     *int16 `json:"attendance_days"`
}

// ScheduleResponse represents a custom schedule.
type ScheduleResponse struct {
    ID               uuid.UUID         `json:"id"`
    Name             string            `json:"name"`
    SchoolDays       []bool            `json:"school_days"`
    ExclusionPeriods []ExclusionPeriod `json:"exclusion_periods"`
    CreatedAt        time.Time         `json:"created_at"`
}

// AttendanceResponse represents a single attendance record.
type AttendanceResponse struct {
    ID              uuid.UUID `json:"id"`
    StudentID       uuid.UUID `json:"student_id"`
    AttendanceDate  time.Time `json:"attendance_date"`
    Status          string    `json:"status"`
    DurationMinutes *int16    `json:"duration_minutes"`
    Notes           *string   `json:"notes"`
    IsAuto          bool      `json:"is_auto"`
    ManualOverride  bool      `json:"manual_override"`
    CreatedAt       time.Time `json:"created_at"`
}

// AttendanceListResponse represents a paginated attendance list.
type AttendanceListResponse struct {
    Records    []AttendanceResponse `json:"records"`
    NextCursor *string              `json:"next_cursor"`
}

// AttendanceSummaryResponse represents attendance summary with pace calculation.
type AttendanceSummaryResponse struct {
    TotalDays          int32   `json:"total_days"`
    PresentFull        int32   `json:"present_full"`
    PresentPartial     int32   `json:"present_partial"`
    Absent             int32   `json:"absent"`
    NotApplicable      int32   `json:"not_applicable"`
    TotalHours         float64 `json:"total_hours"`
    StateRequiredDays  *int16  `json:"state_required_days"`
    StateRequiredHours *int16  `json:"state_required_hours"`
    PaceStatus         *string `json:"pace_status"` // "on_track" | "at_risk" | "behind"
    ProjectedTotalDays *int32  `json:"projected_total_days"`
}

// AssessmentResponse represents a single assessment record.
type AssessmentResponse struct {
    ID             uuid.UUID `json:"id"`
    StudentID      uuid.UUID `json:"student_id"`
    Title          string    `json:"title"`
    Subject        string    `json:"subject"`
    AssessmentType string    `json:"assessment_type"`
    Score          *float64  `json:"score"`
    MaxScore       *float64  `json:"max_score"`
    GradeLetter    *string   `json:"grade_letter"`
    GradePoints    *float64  `json:"grade_points"`
    IsPassing      *bool     `json:"is_passing"`
    AssessmentDate time.Time `json:"assessment_date"`
    Notes          *string   `json:"notes"`
    CreatedAt      time.Time `json:"created_at"`
}

// AssessmentListResponse represents a paginated assessment list.
type AssessmentListResponse struct {
    Records    []AssessmentResponse `json:"records"`
    NextCursor *string              `json:"next_cursor"`
}

// TestScoreResponse represents a single test score.
type TestScoreResponse struct {
    ID             uuid.UUID       `json:"id"`
    StudentID      uuid.UUID       `json:"student_id"`
    TestName       string          `json:"test_name"`
    TestDate       time.Time       `json:"test_date"`
    GradeLevel     *int16          `json:"grade_level"`
    Scores         json.RawMessage `json:"scores"`
    CompositeScore *float64        `json:"composite_score"`
    Percentile     *int16          `json:"percentile"`
    Notes          *string         `json:"notes"`
    CreatedAt      time.Time       `json:"created_at"`
}

// TestListResponse represents a paginated test list.
type TestListResponse struct {
    Tests      []TestScoreResponse `json:"tests"`
    NextCursor *string             `json:"next_cursor"`
}

// PortfolioResponse represents portfolio details.
type PortfolioResponse struct {
    ID                 uuid.UUID               `json:"id"`
    StudentID          uuid.UUID               `json:"student_id"`
    Title              string                  `json:"title"`
    Description        *string                 `json:"description"`
    Organization       string                  `json:"organization"`
    DateRangeStart     time.Time               `json:"date_range_start"`
    DateRangeEnd       time.Time               `json:"date_range_end"`
    IncludeAttendance  bool                    `json:"include_attendance"`
    IncludeAssessments bool                    `json:"include_assessments"`
    Status             string                  `json:"status"`
    ItemCount          int32                   `json:"item_count"`
    GeneratedAt        *time.Time              `json:"generated_at"`
    ExpiresAt          *time.Time              `json:"expires_at"`
    Items              []PortfolioItemResponse `json:"items,omitempty"`
    CreatedAt          time.Time               `json:"created_at"`
}

// PortfolioSummaryResponse represents portfolio summary (for listing).
type PortfolioSummaryResponse struct {
    ID             uuid.UUID  `json:"id"`
    Title          string     `json:"title"`
    Status         string     `json:"status"`
    ItemCount      int32      `json:"item_count"`
    DateRangeStart time.Time  `json:"date_range_start"`
    DateRangeEnd   time.Time  `json:"date_range_end"`
    GeneratedAt    *time.Time `json:"generated_at"`
    ExpiresAt      *time.Time `json:"expires_at"`
    CreatedAt      time.Time  `json:"created_at"`
}

// PortfolioItemResponse represents a portfolio item (cached display data).
type PortfolioItemResponse struct {
    ID                uuid.UUID `json:"id"`
    SourceType        string    `json:"source_type"`
    SourceID          uuid.UUID `json:"source_id"`
    DisplayOrder      int16     `json:"display_order"`
    CachedTitle       string    `json:"cached_title"`
    CachedSubject     *string   `json:"cached_subject"`
    CachedDate        time.Time `json:"cached_date"`
    CachedDescription *string   `json:"cached_description"`
}

// GpaResponse represents a GPA calculation result. (Phase 3)
type GpaResponse struct {
    UnweightedGPA float64                 `json:"unweighted_gpa"`
    WeightedGPA   float64                 `json:"weighted_gpa"`
    TotalCredits  float64                 `json:"total_credits"`
    TotalCourses  int32                   `json:"total_courses"`
    ByGradeLevel  []GpaGradeLevelResponse `json:"by_grade_level"`
}

type GpaGradeLevelResponse struct {
    GradeLevel int16   `json:"grade_level"`
    Unweighted float64 `json:"unweighted"`
    Weighted   float64 `json:"weighted"`
    Credits    float64 `json:"credits"`
}

// GpaTermResponse represents GPA history by term. (Phase 3)
type GpaTermResponse struct {
    SchoolYear    string  `json:"school_year"`
    Semester      *string `json:"semester"`
    UnweightedGPA float64 `json:"unweighted_gpa"`
    WeightedGPA   float64 `json:"weighted_gpa"`
    Credits       float64 `json:"credits"`
    CourseCount   int32   `json:"course_count"`
}

// TranscriptResponse represents transcript details. (Phase 3)
type TranscriptResponse struct {
    ID             uuid.UUID        `json:"id"`
    StudentID      uuid.UUID        `json:"student_id"`
    Title          string           `json:"title"`
    StudentName    string           `json:"student_name"`
    GradeLevels    []string         `json:"grade_levels"`
    Status         string           `json:"status"`
    GPAUnweighted  *float64         `json:"gpa_unweighted"`
    GPAWeighted    *float64         `json:"gpa_weighted"`
    Courses        []CourseResponse `json:"courses,omitempty"`
    GeneratedAt    *time.Time       `json:"generated_at"`
    ExpiresAt      *time.Time       `json:"expires_at"`
    CreatedAt      time.Time        `json:"created_at"`
}

// TranscriptSummaryResponse represents transcript summary. (Phase 3)
type TranscriptSummaryResponse struct {
    ID          uuid.UUID  `json:"id"`
    Title       string     `json:"title"`
    Status      string     `json:"status"`
    GradeLevels []string   `json:"grade_levels"`
    GeneratedAt *time.Time `json:"generated_at"`
    CreatedAt   time.Time  `json:"created_at"`
}

// CourseResponse represents a course record. (Phase 3)
type CourseResponse struct {
    ID          uuid.UUID `json:"id"`
    StudentID   uuid.UUID `json:"student_id"`
    Title       string    `json:"title"`
    Subject     string    `json:"subject"`
    GradeLevel  int16     `json:"grade_level"`
    Credits     float64   `json:"credits"`
    GradeLetter *string   `json:"grade_letter"`
    GradePoints *float64  `json:"grade_points"`
    Level       string    `json:"level"`
    SchoolYear  string    `json:"school_year"`
    Semester    *string   `json:"semester"`
    CreatedAt   time.Time `json:"created_at"`
}

// CourseListResponse represents a paginated course list. (Phase 3)
type CourseListResponse struct {
    Courses    []CourseResponse `json:"courses"`
    NextCursor *string          `json:"next_cursor"`
}

// ComplianceDashboardResponse represents the compliance dashboard overview.
type ComplianceDashboardResponse struct {
    FamilyConfig *FamilyConfigResponse      `json:"family_config"`
    Students     []StudentComplianceSummary `json:"students"`
}

type StudentComplianceSummary struct {
    StudentID              uuid.UUID                  `json:"student_id"`
    StudentName            string                     `json:"student_name"`
    AttendanceSummary      AttendanceSummaryResponse   `json:"attendance_summary"`
    RecentAssessmentsCount int32                       `json:"recent_assessments_count"`
    RecentTestsCount       int32                       `json:"recent_tests_count"`
    ActivePortfolios       []PortfolioSummaryResponse `json:"active_portfolios"`
    PaceStatus             *string                    `json:"pace_status"`
}
```

---

## §9 Background Jobs

All comply:: background jobs use asynq `[ARCH §12]`.

### §9.1 SyncStateConfigsJob

Syncs state compliance requirements from `discover::DiscoveryService` into
`comply_state_configs`. Runs daily to pick up any updates to state legal guides.

```go
// internal/comply/jobs.go

// SyncStateConfigsJob syncs state configs from discover:: into comply_state_configs.
// Queue: Low. Schedule: daily at 4:00 AM UTC.
type SyncStateConfigsJob struct {
    stateConfigRepo  StateConfigRepository
    discoveryService DiscoveryService
}

func (j *SyncStateConfigsJob) ProcessTask(ctx context.Context, t *asynq.Task) error {
    // 1. Call discover::DiscoveryService.ListStateGuides()
    // 2. For each state, call discover::DiscoveryService.GetStateRequirements(stateCode)
    // 3. Upsert into comply_state_configs
    // 4. Log count of updated/inserted rows
    return nil
}
```

**Schedule**: `"0 4 * * *"` (daily at 4:00 AM UTC)

### §9.2 GeneratePortfolioJob

Generates a portfolio PDF asynchronously using `jung-kurt/gofpdf`.

```go
// internal/comply/jobs.go (continued)

// GeneratePortfolioJob generates a portfolio PDF.
// Queue: Default. Trigger: on-demand (POST .../generate).
type GeneratePortfolioJob struct {
    PortfolioID uuid.UUID
    FamilyID    uuid.UUID
}

func (j *GeneratePortfolioJob) ProcessTask(ctx context.Context, t *asynq.Task) error {
    // 1. Load portfolio + items from repository
    // 2. Load attendance summary (if include_attendance)
    // 3. Load assessment records (if include_assessments)
    // 4. Load student + family info from iam::IamService
    // 5. Render PDF via gofpdf (cover page, TOC, body, summaries)
    // 6. Upload PDF to media::MediaService
    // 7. Update portfolio status: generating → ready (or failed)
    // 8. Set expires_at = now() + 90 days
    // 9. Publish PortfolioGenerated event
    return nil
}
```

**Retry policy**: Max 3 attempts with exponential backoff. On final failure, set
status = `failed` with error_message.

### §9.3 GenerateTranscriptJob (Phase 3)

Generates a transcript PDF asynchronously.

```go
// internal/comply/jobs.go (continued)

// GenerateTranscriptJob generates a transcript PDF.
// Queue: Default. Trigger: on-demand (POST .../generate). Phase 3.
type GenerateTranscriptJob struct {
    TranscriptID uuid.UUID
    FamilyID     uuid.UUID
}

func (j *GenerateTranscriptJob) ProcessTask(ctx context.Context, t *asynq.Task) error {
    // 1. Load transcript + courses from repository
    // 2. Calculate GPA (snapshot at generation time)
    // 3. Load student info from iam::IamService
    // 4. Render PDF via gofpdf (standard transcript format)
    // 5. Upload PDF to media::MediaService
    // 6. Update transcript: status → ready, snapshot GPA values
    // 7. Set expires_at = now() + 90 days
    // 8. Publish TranscriptGenerated event
    return nil
}
```

### §9.4 AttendanceThresholdCheckJob

Checks attendance pace against state requirements for all configured families.

```go
// internal/comply/jobs.go (continued)

// AttendanceThresholdCheckJob checks attendance pace against state requirements.
// Queue: Low. Schedule: weekly on Sundays at 5:00 AM UTC.
type AttendanceThresholdCheckJob struct {
    familyConfigRepo FamilyConfigRepository
    stateConfigRepo  StateConfigRepository
    attendanceRepo   AttendanceRepository
    iamService       IamService
    events           *EventBus
}

func (j *AttendanceThresholdCheckJob) ProcessTask(ctx context.Context, t *asynq.Task) error {
    // 1. Load all comply_family_configs
    // 2. For each family + student:
    //    a. Load comply_state_configs for family's state
    //    b. Calculate pace (§12)
    //    c. If at_risk or behind → publish AttendanceThresholdWarning event
    return nil
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

```go
// internal/comply/domain/gpa.go

import "encoding/json"

// GpaCalculator is a pure computation — no database access, no side effects.

// CalculateGPA calculates GPA from a list of courses.
func CalculateGPA(courses []ComplyCourse, scale GpaScale, customConfig json.RawMessage) GpaResult {
    var totalWeightedPoints float64
    var totalUnweightedPoints float64
    var totalCredits float64

    for _, course := range courses {
        if course.GradePoints == nil {
            continue
        }
        gp := *course.GradePoints
        credits := course.Credits
        totalUnweightedPoints += gp * credits

        var boost float64
        switch course.Level {
        case "honors":
            boost = 0.5
        case "ap":
            boost = 1.0
        default:
            boost = 0.0
        }
        totalWeightedPoints += (gp + boost) * credits
        totalCredits += credits
    }

    if totalCredits == 0.0 {
        return GpaResult{
            Unweighted:   0.0,
            Weighted:     0.0,
            TotalCredits: 0.0,
        }
    }

    return GpaResult{
        Unweighted:   totalUnweightedPoints / totalCredits,
        Weighted:     totalWeightedPoints / totalCredits,
        TotalCredits: totalCredits,
    }
}

type GpaResult struct {
    Unweighted   float64
    Weighted     float64
    TotalCredits float64
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
   e. Render PDF via gofpdf:
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

Generated via `jung-kurt/gofpdf` (Go PDF library):

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

```go
// internal/comply/domain/attendance.go

import "math"

// PaceStatus represents attendance pace relative to state requirements.
type PaceStatus string

const (
    PaceStatusOnTrack       PaceStatus = "on_track"
    PaceStatusAtRisk        PaceStatus = "at_risk"        // projected total within 10% of requirement
    PaceStatusBehind        PaceStatus = "behind"          // projected total below 90% of requirement
    PaceStatusNotApplicable PaceStatus = "not_applicable"  // state has no attendance requirement
)

// CalculatePace calculates attendance pace against state requirements.
func CalculatePace(
    actualPresentDays int32,
    elapsedSchoolDays int32,
    totalSchoolDays int32,
    stateRequiredDays *int16,
) PaceStatus {
    if stateRequiredDays == nil {
        return PaceStatusNotApplicable
    }
    required := int32(*stateRequiredDays)

    if elapsedSchoolDays == 0 {
        return PaceStatusOnTrack // school year hasn't started
    }

    // Project total present days based on current pace
    paceRate := float64(actualPresentDays) / float64(elapsedSchoolDays)
    projectedTotal := int32(math.Round(paceRate * float64(totalSchoolDays)))

    if projectedTotal >= required {
        return PaceStatusOnTrack
    } else if projectedTotal >= int32(math.Round(float64(required)*0.9)) {
        return PaceStatusAtRisk
    }
    return PaceStatusBehind
}
```

### Custom Schedule Interaction

When a family has a custom schedule, `elapsed_school_days` is computed by counting
school days according to the schedule:

```go
// internal/comply/domain/attendance.go (continued)

// CountSchoolDays counts school days between two dates using a custom schedule.
func CountSchoolDays(
    start time.Time,
    end time.Time,
    schoolDays [7]bool,              // Mon=0 through Sun=6
    exclusionPeriods []ExclusionPeriod,
) int32 {
    var count int32
    date := start
    for !date.After(end) {
        weekday := int(date.Weekday()+6) % 7 // convert Sunday=0 to Mon=0..Sun=6
        if schoolDays[weekday] && !isExcluded(date, exclusionPeriods) {
            count++
        }
        date = date.AddDate(0, 0, 1)
    }
    return count
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

```go
// internal/comply/domain/attendance.go (continued)

import "time"

// ValidateAttendanceRecord validates a new attendance record.
func ValidateAttendanceRecord(
    date time.Time,
    status string,
    durationMinutes *int16,
    today time.Time,
) error {
    // No future dates
    if date.After(today) {
        return ErrFutureAttendanceDate
    }

    // Valid status
    validStatuses := map[string]bool{
        "present_full": true, "present_partial": true,
        "absent": true, "not_applicable": true,
    }
    if !validStatuses[status] {
        return ErrInvalidAttendanceStatus
    }

    // Duration required for partial attendance
    if status == "present_partial" && durationMinutes == nil {
        return ErrDurationRequiredForPartial
    }

    // No negative duration
    if durationMinutes != nil && *durationMinutes < 0 {
        return ErrNegativeDuration
    }

    return nil
}

// ShouldOverride determines precedence: manual entries override auto-generated ones.
// Manual always wins. Auto never overrides manual.
func ShouldOverride(existingIsAuto bool, newIsManual bool) bool {
    return newIsManual || existingIsAuto
}
```

### §14.2 Portfolio Aggregate

```go
// internal/comply/domain/portfolio.go

// ValidatePortfolioTransition validates a portfolio state transition.
func ValidatePortfolioTransition(currentStatus, targetStatus string) error {
    valid := false
    switch {
    case currentStatus == "configuring" && targetStatus == "generating":
        valid = true
    case currentStatus == "generating" && targetStatus == "ready":
        valid = true
    case currentStatus == "generating" && targetStatus == "failed":
        valid = true
    case currentStatus == "failed" && targetStatus == "generating": // retry
        valid = true
    case currentStatus == "ready" && targetStatus == "expired":
        valid = true
    }

    if !valid {
        return &InvalidPortfolioTransitionError{From: currentStatus, To: targetStatus}
    }

    return nil
}

// ValidatePortfolioGenerate validates that a portfolio can be generated.
func ValidatePortfolioGenerate(status string, itemCount int32, retryCount int16, maxRetries int16) error {
    if status != "configuring" && status != "failed" {
        return ErrPortfolioNotConfiguring
    }

    if itemCount == 0 {
        return ErrEmptyPortfolio
    }

    if status == "failed" && retryCount >= maxRetries {
        return ErrMaxRetriesExceeded
    }

    return nil
}
```

### §14.3 Transcript Aggregate (Phase 3)

```go
// internal/comply/domain/transcript.go

// ValidateTranscriptTransition validates a transcript state transition (same pattern as Portfolio). Phase 3.
// Delegates to ValidatePortfolioTransition since transitions are identical.
func ValidateTranscriptTransition(currentStatus, targetStatus string) error {
    return ValidatePortfolioTransition(currentStatus, targetStatus)
}
```

### §14.4 GpaCalculator Value Object (Phase 3)

Defined in §10. Pure computation function with no side effects or state.

---

## §15 Events comply:: Publishes

Defined in `internal/comply/events.go`. `[CODING §8.4]`

```go
// internal/comply/events.go

import "github.com/google/uuid"

// PortfolioGenerated is published when a portfolio PDF has been generated and is ready for download.
// Consumed by notify:: (in-app notification + optional email).
type PortfolioGenerated struct {
    FamilyID       uuid.UUID `json:"family_id"`
    StudentID      uuid.UUID `json:"student_id"`
    PortfolioID    uuid.UUID `json:"portfolio_id"`
    PortfolioTitle string    `json:"portfolio_title"`
}

func (e PortfolioGenerated) EventName() string { return "comply.portfolio_generated" }

// TranscriptGenerated is published when a transcript PDF has been generated and is ready for download. (Phase 3)
// Consumed by notify:: (in-app notification + optional email).
type TranscriptGenerated struct {
    FamilyID     uuid.UUID `json:"family_id"`
    StudentID    uuid.UUID `json:"student_id"`
    TranscriptID uuid.UUID `json:"transcript_id"`
}

func (e TranscriptGenerated) EventName() string { return "comply.transcript_generated" }

// AttendanceThresholdWarning is published when a student's attendance pace falls below state requirements.
// Consumed by notify:: (in-app + email warning to parent).
type AttendanceThresholdWarning struct {
    FamilyID     uuid.UUID `json:"family_id"`
    StudentID    uuid.UUID `json:"student_id"`
    StudentName  string    `json:"student_name"`
    PaceStatus   string    `json:"pace_status"` // "at_risk" | "behind"
    ActualDays   int32     `json:"actual_days"`
    ExpectedDays int32     `json:"expected_days"`
    RequiredDays int16     `json:"required_days"`
}

func (e AttendanceThresholdWarning) EventName() string { return "comply.attendance_threshold_warning" }
```

---

## §16 Error Types + HTTP Mapping

All compliance errors use custom Go error types with `errors.Is`/`errors.As` and map to HTTP
status codes via `AppError`. Internal details are logged but never exposed in API
responses. `[CODING §2.2, §5.2]`

```go
// internal/comply/errors.go

import (
    "errors"
    "fmt"
)

// ─── Configuration Errors ──────────────────────────────────────────

var (
    ErrFamilyConfigNotFound = errors.New("family config not found")
    ErrInvalidStateCode     = errors.New("invalid state code")
    ErrStateConfigNotFound  = errors.New("state config not found")
    ErrInvalidSchoolYearRange = errors.New("invalid school year date range")
)

// ─── Schedule Errors ───────────────────────────────────────────────

var (
    ErrScheduleNotFound      = errors.New("schedule not found")
    ErrScheduleInUse         = errors.New("schedule in use by family config")
    ErrInvalidSchoolDaysArray = errors.New("invalid school days array — must have 7 elements")
)

// ─── Attendance Errors ─────────────────────────────────────────────

var (
    ErrAttendanceNotFound          = errors.New("attendance record not found")
    ErrFutureAttendanceDate        = errors.New("cannot record attendance for a future date")
    ErrInvalidAttendanceStatus     = errors.New("invalid attendance status")
    ErrDurationRequiredForPartial  = errors.New("duration is required for partial attendance")
    ErrNegativeDuration            = errors.New("duration cannot be negative")
    ErrBulkAttendanceLimitExceeded = errors.New("bulk attendance exceeds maximum of 31 records")
)

// ─── Assessment Errors ─────────────────────────────────────────────

var (
    ErrAssessmentNotFound  = errors.New("assessment record not found")
    ErrInvalidAssessmentType = errors.New("invalid assessment type")
)

// ─── Test Score Errors ─────────────────────────────────────────────

var ErrTestScoreNotFound = errors.New("test score not found")

// ─── Portfolio Errors ──────────────────────────────────────────────

var (
    ErrPortfolioNotFound          = errors.New("portfolio not found")
    ErrPortfolioNotConfiguring    = errors.New("portfolio is not in configuring status")
    ErrEmptyPortfolio             = errors.New("cannot generate an empty portfolio")
    ErrPortfolioExpired           = errors.New("portfolio has expired")
    ErrMaxRetriesExceeded         = errors.New("maximum retry attempts exceeded")
    ErrPortfolioItemSourceNotFound = errors.New("portfolio item source not found in learn::")
    ErrDuplicatePortfolioItem     = errors.New("duplicate item in portfolio")
)

// InvalidPortfolioTransitionError represents an invalid portfolio status transition.
type InvalidPortfolioTransitionError struct {
    From string
    To   string
}

func (e *InvalidPortfolioTransitionError) Error() string {
    return fmt.Sprintf("invalid portfolio status transition from %s to %s", e.From, e.To)
}

// ─── Transcript Errors (Phase 3) ───────────────────────────────────

var (
    ErrTranscriptNotFound = errors.New("transcript not found")
    ErrCourseNotFound     = errors.New("course not found")
    ErrInvalidCourseLevel = errors.New("invalid course level")
)

// ─── Student Errors ────────────────────────────────────────────────

var ErrStudentNotInFamily = errors.New("student not found in family")

// ─── Infrastructure ────────────────────────────────────────────────

// DbError wraps a database error — internal, NOT exposed in API.
type DbError struct {
    Err error
}

func (e *DbError) Error() string { return fmt.Sprintf("database error: %v", e.Err) }
func (e *DbError) Unwrap() error { return e.Err }

// PdfGenerationError wraps a PDF generation failure — internal, NOT exposed in API.
type PdfGenerationError struct {
    Detail string
}

func (e *PdfGenerationError) Error() string { return fmt.Sprintf("PDF generation failed: %s", e.Detail) }

// MediaServiceError wraps a media service error — internal, NOT exposed in API.
type MediaServiceError struct {
    Detail string
}

func (e *MediaServiceError) Error() string { return fmt.Sprintf("media service error: %s", e.Detail) }
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

```go
// internal/comply/event_handlers.go

import (
    "context"

    "homegrown-academy/internal/learn"
    "homegrown-academy/internal/iam"
    "homegrown-academy/internal/billing"
)

// ─── learn:: events ────────────────────────────────────────────────────

type ActivityLoggedHandler struct {
    complianceService ComplianceService
}

func (h *ActivityLoggedHandler) Handle(ctx context.Context, event *learn.ActivityLogged) error {
    return h.complianceService.HandleActivityLogged(ctx, event)
}

// ─── iam:: events ──────────────────────────────────────────────────────

type StudentDeletedHandler struct {
    complianceService ComplianceService
}

func (h *StudentDeletedHandler) Handle(ctx context.Context, event *iam.StudentDeleted) error {
    return h.complianceService.HandleStudentDeleted(ctx, event)
}

type FamilyDeletionScheduledHandler struct {
    complianceService ComplianceService
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event *iam.FamilyDeletionScheduled) error {
    return h.complianceService.HandleFamilyDeletionScheduled(ctx, event)
}

// ─── billing:: events ──────────────────────────────────────────────────

type SubscriptionCancelledHandler struct {
    complianceService ComplianceService
}

func (h *SubscriptionCancelledHandler) Handle(ctx context.Context, event *billing.SubscriptionCancelled) error {
    return h.complianceService.HandleSubscriptionCancelled(ctx, event)
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
- Portfolio generation pipeline (gofpdf PDF)
- Attendance threshold checking
- **~28 endpoints, +6 tables, +3 event handlers, +2 background jobs**

### Phase 3 — Transcripts & GPA `[S§19 Phase 3]`

**In scope**:
- +2 database tables (`comply_transcripts`, `comply_courses`)
- ~14 API endpoints (transcripts, courses, GPA)
- 1 additional background job (`GenerateTranscriptJob`)
- Domain event: `TranscriptGenerated`
- GPA calculation (standard, weighted, custom)
- Transcript generation pipeline (gofpdf PDF)
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

43. All errors checked (`if err != nil`) in production code `[CODING §2.2]`
44. All errors use `ComplianceError` types with `errors.Is`/`errors.As` `[CODING §2.2, §5.2]`
45. Internal error details (PDF generation errors, DB errors) are logged but never exposed in API responses

### Family-Scoping

46. Every query on family-data tables is family-scoped via `FamilyScope` `[CODING §2.4]`
47. `comply_state_configs` is NOT family-scoped (platform-authored reference data)
48. Student ownership is verified (student belongs to family) before all student-scoped operations

---

## §20 Module Structure

```
internal/comply/
├── handler.go                # ~28 Phase 2 + ~14 Phase 3 Echo route handlers
│                             #   (thin layer: extractors → service → response)
├── service.go                # ComplianceServiceImpl — attendance recording,
│                             #   assessment aggregation, portfolio/transcript
│                             #   orchestration, threshold checking, event handling
├── repository.go             # PgStateConfigRepo, PgFamilyConfigRepo,
│                             #   PgScheduleRepo, PgAttendanceRepo,
│                             #   PgAssessmentRepo, PgTestScoreRepo,
│                             #   PgPortfolioRepo, PgPortfolioItemRepo,
│                             #   PgTranscriptRepo (Phase 3), PgCourseRepo (Phase 3)
│                             #   All user-data queries family-scoped via FamilyScope
├── models.go                 # Request/response types (struct tags + swag annotations),
│                             #   GORM models, internal types, config types
├── ports.go                  # ComplianceService interface, all repository interfaces
│                             #   (CQRS separation: commands vs queries)
├── errors.go                 # ComplianceError sentinel errors + custom error types (~25 variants)
├── events.go                 # PortfolioGenerated, TranscriptGenerated,
│                             #   AttendanceThresholdWarning [ARCH §4.6]
├── event_handlers.go         # 4 DomainEventHandler structs:
│                             #   ActivityLoggedHandler,
│                             #   StudentDeletedHandler,
│                             #   FamilyDeletionScheduledHandler,
│                             #   SubscriptionCancelledHandler [ARCH §4.6]
├── jobs.go                   # SyncStateConfigsJob, GeneratePortfolioJob,
│                             #   GenerateTranscriptJob (Phase 3),
│                             #   AttendanceThresholdCheckJob [ARCH §12]
└── domain/                   # Complex domain — aggregate roots + value objects
    │                         #   [ARCH §4.5]
    ├── attendance.go         # Attendance aggregate: date validation, override
    │                         #   precedence, pace calculation, school day counting
    ├── portfolio.go          # Portfolio aggregate: state machine, generate
    │                         #   validation, item invariants
    ├── transcript.go         # Transcript aggregate: state machine (Phase 3)
    ├── gpa.go                # GpaCalculator value object: pure GPA computation
    │                         #   (unweighted, weighted, custom) (Phase 3)
    └── errors.go             # Domain-layer error variants (if separated from
                              #   top-level ComplianceError — optional)
```

> **Complexity class**: Complex (has `domain/` subdirectory). `comply::` has attendance
> threshold invariants, portfolio/transcript state machines, GPA calculation rules, and
> manual-override-auto precedence logic that warrant structural enforcement via aggregate
> roots. `[ARCH §4.5]`
