-- +goose Up
-- Learning domain tables (learn::). [06-learn §3.2]
-- Three-layer data model:
--   Layer 1: Published Content (NOT family-scoped, publisher-based access)
--   Layer 2: Artifact Links (NOT family-scoped, polymorphic cross-references)
--   Layer 3: Family Tracking (family-scoped via FamilyScope + RLS)

-- ═══════════════════════════════════════════════════════════════════════════════
-- LAYER 1: Published Content (owned by Publishers, NOT family-scoped)
-- ═══════════════════════════════════════════════════════════════════════════════

-- Activity/lesson definitions [S§8.1.1]
CREATE TABLE learn_activity_defs (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    methodology_id        UUID REFERENCES method_definitions(id),
    tool_id               UUID REFERENCES method_tools(id),
    est_duration_minutes  SMALLINT,
    attachments           JSONB NOT NULL DEFAULT '[]',
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_activity_defs_publisher ON learn_activity_defs(publisher_id);
CREATE INDEX idx_learn_activity_defs_methodology ON learn_activity_defs(methodology_id)
    WHERE methodology_id IS NOT NULL;
CREATE INDEX idx_learn_activity_defs_subject ON learn_activity_defs USING GIN(subject_tags);

ALTER TABLE learn_activity_defs ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_learn_activity_defs_search ON learn_activity_defs USING GIN(search_vector);

-- Assessment/test definitions [S§8.1.2] (Phase 2 CRUD, DDL now)
CREATE TABLE learn_assessment_defs (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    scoring_type          TEXT NOT NULL DEFAULT 'percentage'
                          CHECK (scoring_type IN ('percentage', 'letter', 'points', 'pass_fail')),
    max_score             NUMERIC,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_assessment_defs_publisher ON learn_assessment_defs(publisher_id);

-- Reading item definitions [S§8.1.3]
CREATE TABLE learn_reading_items (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    author                TEXT,
    isbn                  TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    description           TEXT,
    cover_image_url       TEXT,
    page_count            SMALLINT,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_reading_items_publisher ON learn_reading_items(publisher_id);
CREATE INDEX idx_learn_reading_items_isbn ON learn_reading_items(isbn) WHERE isbn IS NOT NULL;
CREATE INDEX idx_learn_reading_items_subject ON learn_reading_items USING GIN(subject_tags);

ALTER TABLE learn_reading_items ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(author, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_learn_reading_items_search ON learn_reading_items USING GIN(search_vector);

-- Project definitions [S§8.1.5] (Phase 2 CRUD, DDL now)
CREATE TABLE learn_project_defs (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    milestone_templates   JSONB NOT NULL DEFAULT '[]',
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_project_defs_publisher ON learn_project_defs(publisher_id);

-- Video lesson definitions [S§8.1.6, S§8.1.11]
CREATE TABLE learn_video_defs (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    methodology_id        UUID REFERENCES method_definitions(id),
    duration_seconds      INTEGER,
    thumbnail_url         TEXT,
    video_url             TEXT NOT NULL,
    video_source          TEXT NOT NULL DEFAULT 'self_hosted'
                          CHECK (video_source IN ('self_hosted', 'youtube', 'vimeo')),
    external_video_id     TEXT,
    transcode_job_id      UUID,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_video_defs_publisher ON learn_video_defs(publisher_id);
CREATE INDEX idx_learn_video_defs_methodology ON learn_video_defs(methodology_id)
    WHERE methodology_id IS NOT NULL;

-- Question bank [S§8.1.9]
CREATE TABLE learn_questions (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    question_type         TEXT NOT NULL
                          CHECK (question_type IN (
                              'multiple_choice', 'fill_in_blank', 'true_false',
                              'matching', 'ordering', 'short_answer'
                          )),
    content               TEXT NOT NULL,
    media_attachments     JSONB NOT NULL DEFAULT '[]',
    answer_data           JSONB NOT NULL,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    methodology_id        UUID REFERENCES method_definitions(id),
    difficulty_level      SMALLINT CHECK (difficulty_level BETWEEN 1 AND 5),
    auto_scorable         BOOLEAN NOT NULL DEFAULT true,
    points                NUMERIC NOT NULL DEFAULT 1,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_questions_publisher ON learn_questions(publisher_id);
CREATE INDEX idx_learn_questions_type ON learn_questions(question_type);
CREATE INDEX idx_learn_questions_subject ON learn_questions USING GIN(subject_tags);
CREATE INDEX idx_learn_questions_methodology ON learn_questions(methodology_id)
    WHERE methodology_id IS NOT NULL;

ALTER TABLE learn_questions ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(content, ''))
    ) STORED;
CREATE INDEX idx_learn_questions_search ON learn_questions USING GIN(search_vector);

-- Quiz definitions [S§8.1.9]
CREATE TABLE learn_quiz_defs (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    methodology_id        UUID REFERENCES method_definitions(id),
    time_limit_minutes    SMALLINT,
    passing_score_percent SMALLINT NOT NULL DEFAULT 70,
    shuffle_questions     BOOLEAN NOT NULL DEFAULT false,
    show_correct_after    BOOLEAN NOT NULL DEFAULT true,
    question_count        SMALLINT NOT NULL DEFAULT 0,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_quiz_defs_publisher ON learn_quiz_defs(publisher_id);
CREATE INDEX idx_learn_quiz_defs_subject ON learn_quiz_defs USING GIN(subject_tags);
CREATE INDEX idx_learn_quiz_defs_methodology ON learn_quiz_defs(methodology_id)
    WHERE methodology_id IS NOT NULL;

ALTER TABLE learn_quiz_defs ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_learn_quiz_defs_search ON learn_quiz_defs USING GIN(search_vector);

-- Quiz-question join table [S§8.1.9]
CREATE TABLE learn_quiz_questions (
    quiz_def_id           UUID NOT NULL REFERENCES learn_quiz_defs(id) ON DELETE CASCADE,
    question_id           UUID NOT NULL REFERENCES learn_questions(id) ON DELETE CASCADE,
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    points_override       NUMERIC,
    PRIMARY KEY (quiz_def_id, question_id)
);

CREATE INDEX idx_learn_quiz_questions_question ON learn_quiz_questions(question_id);

-- Sequence definitions [S§8.1.12]
CREATE TABLE learn_sequence_defs (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    methodology_id        UUID REFERENCES method_definitions(id),
    is_linear             BOOLEAN NOT NULL DEFAULT true,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_sequence_defs_publisher ON learn_sequence_defs(publisher_id);
CREATE INDEX idx_learn_sequence_defs_subject ON learn_sequence_defs USING GIN(subject_tags);
CREATE INDEX idx_learn_sequence_defs_methodology ON learn_sequence_defs(methodology_id)
    WHERE methodology_id IS NOT NULL;

ALTER TABLE learn_sequence_defs ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_learn_sequence_defs_search ON learn_sequence_defs USING GIN(search_vector);

-- Sequence items [S§8.1.12]
CREATE TABLE learn_sequence_items (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    sequence_def_id       UUID NOT NULL REFERENCES learn_sequence_defs(id) ON DELETE CASCADE,
    sort_order            SMALLINT NOT NULL,
    content_type          TEXT NOT NULL
                          CHECK (content_type IN (
                              'activity_def', 'assessment_def', 'reading_item',
                              'video_def', 'quiz_def'
                          )),
    content_id            UUID NOT NULL,
    is_required           BOOLEAN NOT NULL DEFAULT true,
    unlock_after_previous BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_sequence_items_sequence ON learn_sequence_items(sequence_def_id, sort_order);

-- ═══════════════════════════════════════════════════════════════════════════════
-- LAYER 2: Artifact Links (NOT family-scoped)
-- ═══════════════════════════════════════════════════════════════════════════════

-- Polymorphic links between published content [06-learn §9]
CREATE TABLE learn_artifact_links (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    source_type           TEXT NOT NULL CHECK (source_type IN (
                              'activity_def', 'assessment_def', 'reading_item',
                              'project_def', 'video_def', 'quiz_def', 'sequence_def'
                          )),
    source_id             UUID NOT NULL,
    target_type           TEXT NOT NULL CHECK (target_type IN (
                              'activity_def', 'assessment_def', 'reading_item',
                              'project_def', 'video_def', 'quiz_def', 'sequence_def'
                          )),
    target_id             UUID NOT NULL,
    relationship          TEXT NOT NULL DEFAULT 'about'
                          CHECK (relationship IN ('about', 'part_of', 'follow_up', 'inspired_by')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_artifact_link UNIQUE (source_type, source_id, target_type, target_id)
);

CREATE INDEX idx_learn_artifact_links_source ON learn_artifact_links(source_type, source_id);
CREATE INDEX idx_learn_artifact_links_target ON learn_artifact_links(target_type, target_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- LAYER 3: Family Tracking (family-scoped via FamilyScope)
-- ═══════════════════════════════════════════════════════════════════════════════

-- Activity logs [S§8.1.1]
CREATE TABLE learn_activity_logs (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    content_id            UUID REFERENCES learn_activity_defs(id) ON DELETE SET NULL,
    methodology_id        UUID REFERENCES method_definitions(id),
    tool_id               UUID REFERENCES method_tools(id),
    duration_minutes      SMALLINT,
    attachments           JSONB NOT NULL DEFAULT '[]',
    activity_date         DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_activity_logs_family_student
    ON learn_activity_logs(family_id, student_id, activity_date DESC);
CREATE INDEX idx_learn_activity_logs_subject
    ON learn_activity_logs USING GIN(subject_tags);
CREATE INDEX idx_learn_activity_logs_date
    ON learn_activity_logs(family_id, activity_date DESC);

ALTER TABLE learn_activity_logs ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_learn_activity_logs_search ON learn_activity_logs USING GIN(search_vector);

-- Journal entries [S§8.1.4]
CREATE TABLE learn_journal_entries (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    entry_type            TEXT NOT NULL CHECK (entry_type IN ('freeform', 'narration', 'reflection')),
    title                 TEXT,
    content               TEXT NOT NULL,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    content_id            UUID,
    attachments           JSONB NOT NULL DEFAULT '[]',
    entry_date            DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_journal_entries_family_student
    ON learn_journal_entries(family_id, student_id, entry_date DESC);
CREATE INDEX idx_learn_journal_entries_type
    ON learn_journal_entries(family_id, entry_type);

ALTER TABLE learn_journal_entries ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, ''))
    ) STORED;
CREATE INDEX idx_learn_journal_entries_search ON learn_journal_entries USING GIN(search_vector);

-- Progress snapshots [S§8.1.7]
CREATE TABLE learn_progress_snapshots (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    snapshot_date         DATE NOT NULL,
    data                  JSONB NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_learn_progress_snapshots_unique
    ON learn_progress_snapshots(family_id, student_id, snapshot_date);

-- Reading progress [S§8.1.3]
CREATE TABLE learn_reading_progress (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    reading_item_id       UUID NOT NULL REFERENCES learn_reading_items(id) ON DELETE CASCADE,
    reading_list_id       UUID,  -- FK added after learn_reading_lists is created
    status                TEXT NOT NULL DEFAULT 'to_read'
                          CHECK (status IN ('to_read', 'in_progress', 'completed')),
    started_at            TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    notes                 TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_reading_progress UNIQUE (family_id, student_id, reading_item_id)
);

CREATE INDEX idx_learn_reading_progress_family_student
    ON learn_reading_progress(family_id, student_id, status);

-- Assessment results [S§8.1.2] (Phase 2 CRUD, DDL now)
CREATE TABLE learn_assessment_results (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    assessment_def_id     UUID NOT NULL REFERENCES learn_assessment_defs(id) ON DELETE CASCADE,
    score                 NUMERIC NOT NULL,
    max_score             NUMERIC,
    weight                NUMERIC DEFAULT 1.0,
    notes                 TEXT,
    assessment_date       DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_assessment_results_family_student
    ON learn_assessment_results(family_id, student_id, assessment_date DESC);

-- Project progress [S§8.1.5] (Phase 2 CRUD, DDL now)
CREATE TABLE learn_project_progress (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    project_def_id        UUID NOT NULL REFERENCES learn_project_defs(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'planning'
                          CHECK (status IN ('planning', 'in_progress', 'completed')),
    milestones            JSONB NOT NULL DEFAULT '[]',
    started_at            TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    notes                 TEXT,
    attachments           JSONB NOT NULL DEFAULT '[]',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_project_progress_family_student
    ON learn_project_progress(family_id, student_id, status);

-- Video progress [S§8.1.6, S§8.1.11]
CREATE TABLE learn_video_progress (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    video_def_id          UUID NOT NULL REFERENCES learn_video_defs(id) ON DELETE CASCADE,
    watched_seconds       INTEGER NOT NULL DEFAULT 0,
    completed             BOOLEAN NOT NULL DEFAULT false,
    last_position_seconds INTEGER NOT NULL DEFAULT 0,
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_video_progress UNIQUE (family_id, student_id, video_def_id)
);

CREATE INDEX idx_learn_video_progress_family_student
    ON learn_video_progress(family_id, student_id);

-- Quiz sessions [S§8.1.9]
CREATE TABLE learn_quiz_sessions (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    quiz_def_id           UUID NOT NULL REFERENCES learn_quiz_defs(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'not_started'
                          CHECK (status IN ('not_started', 'in_progress', 'submitted', 'scored')),
    started_at            TIMESTAMPTZ,
    submitted_at          TIMESTAMPTZ,
    scored_at             TIMESTAMPTZ,
    score                 NUMERIC,
    max_score             NUMERIC,
    passed                BOOLEAN,
    answers               JSONB NOT NULL DEFAULT '[]',
    scored_by             UUID REFERENCES iam_parents(id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_quiz_sessions_family_student
    ON learn_quiz_sessions(family_id, student_id, created_at DESC);
CREATE INDEX idx_learn_quiz_sessions_quiz
    ON learn_quiz_sessions(quiz_def_id);
CREATE INDEX idx_learn_quiz_sessions_status
    ON learn_quiz_sessions(family_id, status)
    WHERE status IN ('not_started', 'in_progress');

-- Sequence progress [S§8.1.12]
CREATE TABLE learn_sequence_progress (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    sequence_def_id       UUID NOT NULL REFERENCES learn_sequence_defs(id) ON DELETE CASCADE,
    current_item_index    SMALLINT NOT NULL DEFAULT 0,
    status                TEXT NOT NULL DEFAULT 'not_started'
                          CHECK (status IN ('not_started', 'in_progress', 'completed')),
    item_completions      JSONB NOT NULL DEFAULT '[]',
    started_at            TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_sequence_progress UNIQUE (family_id, student_id, sequence_def_id)
);

CREATE INDEX idx_learn_sequence_progress_family_student
    ON learn_sequence_progress(family_id, student_id, status);
CREATE INDEX idx_learn_sequence_progress_sequence
    ON learn_sequence_progress(sequence_def_id);

-- Student assignments [S§8.6.3]
CREATE TABLE learn_student_assignments (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    assigned_by           UUID NOT NULL REFERENCES iam_parents(id),
    content_type          TEXT NOT NULL
                          CHECK (content_type IN (
                              'activity_def', 'reading_item', 'video_def',
                              'quiz_def', 'sequence_def'
                          )),
    content_id            UUID NOT NULL,
    due_date              DATE,
    status                TEXT NOT NULL DEFAULT 'assigned'
                          CHECK (status IN ('assigned', 'in_progress', 'completed', 'skipped')),
    assigned_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_assignments_family_student
    ON learn_student_assignments(family_id, student_id, status);
CREATE INDEX idx_learn_assignments_due
    ON learn_student_assignments(family_id, due_date)
    WHERE due_date IS NOT NULL AND status IN ('assigned', 'in_progress');

-- ═══════════════════════════════════════════════════════════════════════════════
-- Supporting tables
-- ═══════════════════════════════════════════════════════════════════════════════

-- Subject taxonomy [S§8.3]
CREATE TABLE learn_subject_taxonomy (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    parent_id             UUID REFERENCES learn_subject_taxonomy(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    slug                  TEXT NOT NULL UNIQUE,
    level                 SMALLINT NOT NULL,
    display_order         SMALLINT NOT NULL DEFAULT 0,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_subject_taxonomy_parent ON learn_subject_taxonomy(parent_id);
CREATE INDEX idx_learn_subject_taxonomy_level ON learn_subject_taxonomy(level, display_order);

-- Family-scoped custom subjects [S§8.3]
CREATE TABLE learn_custom_subjects (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    parent_taxonomy_id    UUID REFERENCES learn_subject_taxonomy(id),
    name                  TEXT NOT NULL,
    slug                  TEXT NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_custom_subject UNIQUE (family_id, slug)
);

CREATE INDEX idx_learn_custom_subjects_family ON learn_custom_subjects(family_id);

-- Reading lists [S§8.1.3]
CREATE TABLE learn_reading_lists (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    description           TEXT,
    student_id            UUID REFERENCES iam_students(id) ON DELETE SET NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_reading_lists_family ON learn_reading_lists(family_id);

-- Add FK from reading_progress to reading_lists now that the table exists
ALTER TABLE learn_reading_progress
    ADD CONSTRAINT fk_reading_progress_list
    FOREIGN KEY (reading_list_id) REFERENCES learn_reading_lists(id) ON DELETE SET NULL;

-- Reading list items (join table)
CREATE TABLE learn_reading_list_items (
    reading_list_id       UUID NOT NULL REFERENCES learn_reading_lists(id) ON DELETE CASCADE,
    reading_item_id       UUID NOT NULL REFERENCES learn_reading_items(id) ON DELETE CASCADE,
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    added_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (reading_list_id, reading_item_id)
);

-- Grading scales [S§8.1.2] (Phase 2 CRUD, DDL now)
CREATE TABLE learn_grading_scales (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    scale_type            TEXT NOT NULL CHECK (scale_type IN ('letter', 'pass_fail', 'custom')),
    grades                JSONB NOT NULL,
    is_default            BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_grading_scales_family ON learn_grading_scales(family_id);

-- Data export requests [S§8.5]
CREATE TABLE learn_export_requests (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    requested_by          UUID NOT NULL REFERENCES iam_parents(id),
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    file_url              TEXT,
    expires_at            TIMESTAMPTZ,
    error_message         TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_export_requests_family ON learn_export_requests(family_id, created_at DESC);

-- ═══════════════════════════════════════════════════════════════════════════════
-- RLS Policies [06-learn §3.3]
-- ═══════════════════════════════════════════════════════════════════════════════

-- Layer 3: Family Tracking — standard family-scoped RLS
ALTER TABLE learn_activity_logs ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_activity_logs_family_policy ON learn_activity_logs
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_journal_entries ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_journal_entries_family_policy ON learn_journal_entries
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_progress_snapshots ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_progress_snapshots_family_policy ON learn_progress_snapshots
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_reading_progress ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_reading_progress_family_policy ON learn_reading_progress
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_assessment_results ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_assessment_results_family_policy ON learn_assessment_results
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_project_progress ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_project_progress_family_policy ON learn_project_progress
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_video_progress ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_video_progress_family_policy ON learn_video_progress
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_reading_lists ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_reading_lists_family_policy ON learn_reading_lists
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_custom_subjects ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_custom_subjects_family_policy ON learn_custom_subjects
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_grading_scales ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_grading_scales_family_policy ON learn_grading_scales
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_export_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_export_requests_family_policy ON learn_export_requests
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_quiz_sessions ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_quiz_sessions_family_policy ON learn_quiz_sessions
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_sequence_progress ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_sequence_progress_family_policy ON learn_sequence_progress
    USING (family_id = current_setting('app.current_family_id')::uuid);

ALTER TABLE learn_student_assignments ENABLE ROW LEVEL SECURITY;
CREATE POLICY learn_student_assignments_family_policy ON learn_student_assignments
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- Layer 1: Published Content — NO RLS (publisher-based access at app level)
-- Layer 2: Artifact Links — NO RLS (shared across platform)
-- Platform: Subject Taxonomy — NO RLS (shared across platform)

-- +goose Down
DROP TABLE IF EXISTS learn_reading_list_items CASCADE;
DROP TABLE IF EXISTS learn_reading_lists CASCADE;
DROP TABLE IF EXISTS learn_export_requests CASCADE;
DROP TABLE IF EXISTS learn_grading_scales CASCADE;
DROP TABLE IF EXISTS learn_custom_subjects CASCADE;
DROP TABLE IF EXISTS learn_student_assignments CASCADE;
DROP TABLE IF EXISTS learn_sequence_progress CASCADE;
DROP TABLE IF EXISTS learn_quiz_sessions CASCADE;
DROP TABLE IF EXISTS learn_video_progress CASCADE;
DROP TABLE IF EXISTS learn_project_progress CASCADE;
DROP TABLE IF EXISTS learn_assessment_results CASCADE;
DROP TABLE IF EXISTS learn_reading_progress CASCADE;
DROP TABLE IF EXISTS learn_progress_snapshots CASCADE;
DROP TABLE IF EXISTS learn_journal_entries CASCADE;
DROP TABLE IF EXISTS learn_activity_logs CASCADE;
DROP TABLE IF EXISTS learn_artifact_links CASCADE;
DROP TABLE IF EXISTS learn_sequence_items CASCADE;
DROP TABLE IF EXISTS learn_sequence_defs CASCADE;
DROP TABLE IF EXISTS learn_quiz_questions CASCADE;
DROP TABLE IF EXISTS learn_quiz_defs CASCADE;
DROP TABLE IF EXISTS learn_questions CASCADE;
DROP TABLE IF EXISTS learn_video_defs CASCADE;
DROP TABLE IF EXISTS learn_project_defs CASCADE;
DROP TABLE IF EXISTS learn_reading_items CASCADE;
DROP TABLE IF EXISTS learn_assessment_defs CASCADE;
DROP TABLE IF EXISTS learn_activity_defs CASCADE;
DROP TABLE IF EXISTS learn_subject_taxonomy CASCADE;
