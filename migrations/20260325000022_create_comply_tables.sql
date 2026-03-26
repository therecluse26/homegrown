-- +goose Up
-- Compliance & Reporting domain (comply::) — 10 tables [14-comply §3]

-- ═══════════════════════════════════════════════════════════════════════════════
-- 1. comply_state_configs — platform-authored reference data (NOT family-scoped)
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_state_configs (
    state_code              CHAR(2) PRIMARY KEY,
    state_name              TEXT NOT NULL,
    notification_required   BOOLEAN NOT NULL DEFAULT FALSE,
    notification_details    TEXT,
    required_subjects       TEXT[] NOT NULL DEFAULT '{}',
    assessment_required     BOOLEAN NOT NULL DEFAULT FALSE,
    assessment_details      TEXT,
    record_keeping_required BOOLEAN NOT NULL DEFAULT FALSE,
    record_keeping_details  TEXT,
    attendance_required     BOOLEAN NOT NULL DEFAULT FALSE,
    attendance_days         SMALLINT,
    attendance_hours        SMALLINT,
    attendance_details      TEXT,
    umbrella_school_available BOOLEAN NOT NULL DEFAULT FALSE,
    umbrella_school_details TEXT,
    regulation_level        TEXT NOT NULL DEFAULT 'moderate'
                            CHECK (regulation_level IN ('low', 'moderate', 'high')),
    synced_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ═══════════════════════════════════════════════════════════════════════════════
-- 2. comply_custom_schedules — must be created BEFORE comply_family_configs (FK)
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_custom_schedules (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id         UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    name              TEXT NOT NULL,
    school_days       BOOLEAN[] NOT NULL DEFAULT '{true,true,true,true,true,false,false}',
    exclusion_periods JSONB NOT NULL DEFAULT '[]',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comply_custom_schedules_family ON comply_custom_schedules(family_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- 3. comply_family_configs — references state_configs + custom_schedules
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_family_configs (
    family_id          UUID PRIMARY KEY REFERENCES iam_families(id) ON DELETE CASCADE,
    state_code         CHAR(2) NOT NULL REFERENCES comply_state_configs(state_code),
    school_year_start  DATE NOT NULL,
    school_year_end    DATE NOT NULL,
    total_school_days  SMALLINT NOT NULL CHECK (total_school_days > 0),
    custom_schedule_id UUID REFERENCES comply_custom_schedules(id) ON DELETE SET NULL,
    gpa_scale          TEXT NOT NULL DEFAULT 'standard_4'
                       CHECK (gpa_scale IN ('standard_4', 'weighted', 'custom')),
    gpa_custom_config  JSONB,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ═══════════════════════════════════════════════════════════════════════════════
-- 4. comply_attendance — UNIQUE on (family_id, student_id, attendance_date)
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_attendance (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id        UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id       UUID NOT NULL,
    attendance_date  DATE NOT NULL,
    status           TEXT NOT NULL CHECK (status IN ('present_full', 'present_partial', 'absent', 'not_applicable')),
    duration_minutes SMALLINT,
    notes            TEXT,
    is_auto          BOOLEAN NOT NULL DEFAULT FALSE,
    manual_override  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (family_id, student_id, attendance_date)
);

CREATE INDEX idx_comply_attendance_student ON comply_attendance(family_id, student_id, attendance_date);

-- ═══════════════════════════════════════════════════════════════════════════════
-- 5. comply_assessment_records
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_assessment_records (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id         UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id        UUID NOT NULL,
    title             TEXT NOT NULL,
    subject           TEXT NOT NULL,
    assessment_type   TEXT NOT NULL CHECK (assessment_type IN ('test', 'quiz', 'project', 'assignment', 'presentation', 'portfolio_piece', 'other')),
    score             DOUBLE PRECISION,
    max_score         DOUBLE PRECISION,
    grade_letter      TEXT,
    grade_points      DOUBLE PRECISION,
    is_passing        BOOLEAN,
    source_activity_id UUID,
    assessment_date   DATE NOT NULL,
    notes             TEXT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comply_assessments_student ON comply_assessment_records(family_id, student_id, assessment_date);

-- ═══════════════════════════════════════════════════════════════════════════════
-- 6. comply_standardized_tests
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_standardized_tests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id      UUID NOT NULL,
    test_name       TEXT NOT NULL,
    test_date       DATE NOT NULL,
    grade_level     SMALLINT,
    scores          JSONB NOT NULL DEFAULT '{}',
    composite_score DOUBLE PRECISION,
    percentile      SMALLINT,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comply_tests_student ON comply_standardized_tests(family_id, student_id, test_date DESC);

-- ═══════════════════════════════════════════════════════════════════════════════
-- 7. comply_portfolios — state machine CHECK constraint
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_portfolios (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id           UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id          UUID NOT NULL,
    title               TEXT NOT NULL,
    description         TEXT,
    organization        TEXT NOT NULL CHECK (organization IN ('by_subject', 'chronological', 'by_student')),
    date_range_start    DATE NOT NULL,
    date_range_end      DATE NOT NULL,
    include_attendance  BOOLEAN NOT NULL DEFAULT FALSE,
    include_assessments BOOLEAN NOT NULL DEFAULT FALSE,
    status              TEXT NOT NULL DEFAULT 'configuring'
                        CHECK (status IN ('configuring', 'generating', 'ready', 'failed', 'expired')),
    upload_id           UUID,
    generated_at        TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ,
    error_message       TEXT,
    retry_count         SMALLINT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comply_portfolios_student ON comply_portfolios(family_id, student_id);
CREATE INDEX idx_comply_portfolios_status ON comply_portfolios(status) WHERE status IN ('generating', 'ready');

-- ═══════════════════════════════════════════════════════════════════════════════
-- 8. comply_portfolio_items — references comply_portfolios
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_portfolio_items (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portfolio_id       UUID NOT NULL REFERENCES comply_portfolios(id) ON DELETE CASCADE,
    source_type        TEXT NOT NULL CHECK (source_type IN ('activity', 'journal', 'project', 'reading_list', 'assessment')),
    source_id          UUID NOT NULL,
    display_order      SMALLINT NOT NULL DEFAULT 0,
    cached_title       TEXT NOT NULL,
    cached_subject     TEXT,
    cached_date        DATE NOT NULL,
    cached_description TEXT,
    cached_attachments JSONB,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comply_portfolio_items_portfolio ON comply_portfolio_items(portfolio_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- 9. comply_transcripts — Phase 3
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_transcripts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id               UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id              UUID NOT NULL,
    title                   TEXT NOT NULL,
    student_name            TEXT NOT NULL,
    grade_levels            TEXT[] NOT NULL DEFAULT '{}',
    status                  TEXT NOT NULL DEFAULT 'configuring'
                            CHECK (status IN ('configuring', 'generating', 'ready', 'failed', 'expired')),
    snapshot_gpa_unweighted DOUBLE PRECISION,
    snapshot_gpa_weighted   DOUBLE PRECISION,
    upload_id               UUID,
    generated_at            TIMESTAMPTZ,
    expires_at              TIMESTAMPTZ,
    error_message           TEXT,
    retry_count             SMALLINT NOT NULL DEFAULT 0,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comply_transcripts_student ON comply_transcripts(family_id, student_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- 10. comply_courses — Phase 3, references comply_transcripts
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE comply_courses (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id     UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id    UUID NOT NULL,
    transcript_id UUID REFERENCES comply_transcripts(id) ON DELETE SET NULL,
    title         TEXT NOT NULL,
    subject       TEXT NOT NULL,
    grade_level   SMALLINT NOT NULL,
    credits       DOUBLE PRECISION NOT NULL CHECK (credits > 0),
    grade_letter  TEXT,
    grade_points  DOUBLE PRECISION,
    level         TEXT NOT NULL DEFAULT 'regular'
                  CHECK (level IN ('regular', 'honors', 'ap')),
    school_year   TEXT NOT NULL,
    semester      TEXT CHECK (semester IS NULL OR semester IN ('fall', 'spring', 'summer', 'full_year')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_comply_courses_student ON comply_courses(family_id, student_id);
CREATE INDEX idx_comply_courses_transcript ON comply_courses(transcript_id) WHERE transcript_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS comply_courses;
DROP TABLE IF EXISTS comply_transcripts;
DROP TABLE IF EXISTS comply_portfolio_items;
DROP TABLE IF EXISTS comply_portfolios;
DROP TABLE IF EXISTS comply_standardized_tests;
DROP TABLE IF EXISTS comply_assessment_records;
DROP TABLE IF EXISTS comply_attendance;
DROP TABLE IF EXISTS comply_family_configs;
DROP TABLE IF EXISTS comply_custom_schedules;
DROP TABLE IF EXISTS comply_state_configs;
