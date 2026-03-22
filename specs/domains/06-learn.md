# Domain Spec 06 — Learning Layer (learn::)

## §1 Overview

The Learning domain is the **heart of the platform** — it implements all methodology-scoped
learning tools including **interactive student-facing features**: online quiz-taking, in-platform
content viewing, video playback with progress tracking, and structured lesson sequences. Beyond
tracking what happens offline, students engage with content directly on the platform. The domain
owns a three-layer data model: published content definitions (owned by Publishers), artifact links
between content, and per-student family tracking data. Tool behavior is resolved via `method::`
config lookup — never methodology branching. `[S§8, V§5, V§8]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/learn/` |
| **DB prefix** | `learn_` |
| **Complexity class** | Complex (has `domain/` subdirectory) `[ARCH §4.5]` |
| **CQRS** | Yes — activity log writes separated from progress analytics reads `[ARCH §4.7]` |
| **External adapter** | None (media uploads delegated to `media::`) |
| **Key constraint** | Tool behavior resolved via `method::` config lookup, never methodology branching `[S§4.2, CODING §4.1]` |

**What learn:: owns**: Activity definitions and logs, assessment definitions and results,
reading item definitions and progress, journal entries, project definitions and progress (Phase 2),
video lesson definitions and progress, reading lists, subject taxonomy (platform-managed), custom
subjects (family-scoped), progress tracking and analytics, artifact links between published
content, data export, methodology-specific tool implementations (Phase 3), **interactive
assessment engine** (question bank, quiz definitions, quiz sessions), **lesson sequence engine**
(sequence definitions, sequence items, sequence progress), and **student assignment system**
(content assignment, status tracking).

**What learn:: does NOT own**: Tool registry and tool activation mappings (owned by `method::`),
methodology configuration and terminology (owned by `method::`), Publisher entities and membership
(owned by `mkt::`), file uploads and media processing (owned by `media::`), compliance reporting
(owned by `comply::`), search indexing (owned by `search::`), notification delivery (owned by
`notify::`), recommendations (owned by `recs::`), student profiles (owned by `iam::`).

**What learn:: delegates**: Media upload/validation → `media::` (via adapter interface). Tool
resolution → `method::MethodologyService` (via service dependency). Publisher ownership checks →
`mkt::` (via service dependency). Search indexing → `search::` (via `ActivityLogged` event).
Notification delivery → `notify::` (via domain events). Compliance data feed → `comply::` (via
domain events). Student/family data lookup → `iam::IamService`.

---

## §2 Requirements Traceability

Every SPEC.md §8 requirement maps to a section in this document. Cross-references from
other spec sections are included where the learning domain is involved.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Log daily learning activities with title, description, tags, date, duration | `[S§8.1.1]` | §3.2 (`learn_activity_defs`, `learn_activity_logs`), §4.2, §5 |
| Activities support attachments (photos, files) | `[S§8.1.1]` | §3.2 (`attachments` JSONB), §8 |
| Activities filterable by student, subject, date range, methodology | `[S§8.1.1]` | §4.2, §6 |
| Activities tier: Free | `[S§8.1.1]` | §16 |
| Record assessments with title, subject, student, date, score, weight | `[S§8.1.2]` | §3.2 (`learn_assessment_defs`, `learn_assessment_results`), §18 (Phase 2) |
| Calculate running averages per subject per student | `[S§8.1.2]` | §12, §18 (Phase 2) |
| Define grading scales (A/B/C/D/F, pass/fail, custom) | `[S§8.1.2]` | §3.2 (`learn_grading_scales`), §18 (Phase 2) |
| Assessments tier: Free (basic), Premium (advanced analytics) | `[S§8.1.2]` | §16 |
| Create and manage book lists with title, author, tags, status | `[S§8.1.3]` | §3.2 (`learn_reading_items`, `learn_reading_progress`, `learn_reading_lists`), §4.2 |
| Import books by ISBN or title search | `[S§8.1.3]` | §18 (Phase 2) |
| Completed books auto-generate activity log entry | `[S§8.1.3]` | §5 (`complete_reading`) |
| Share reading lists with friends or groups | `[S§8.1.3]` | §18 (Phase 2) |
| Reading lists tier: Free | `[S§8.1.3]` | §16 |
| Create journal entries with text, date, student, tags, attachments | `[S§8.1.4]` | §3.2 (`learn_journal_entries`), §4.2 |
| Entry types: free-form, narration, reflection | `[S§8.1.4]` | §3.2 (CHECK constraint on `entry_type`) |
| Entries searchable by keyword, filterable by student, date, type | `[S§8.1.4]` | §4.2, §6 |
| Journaling tier: Free (basic), Premium (enhanced storage) | `[S§8.1.4]` | §16 |
| Multi-step projects with milestones | `[S§8.1.5]` | §3.2 (`learn_project_defs`, `learn_project_progress`), §18 (Phase 2) |
| Project status lifecycle: planning → in-progress → completed | `[S§8.1.5]` | §18 (Phase 2) |
| Projects tier: Free (basic), Premium (portfolio integration) | `[S§8.1.5]` | §16 |
| Video lesson progress tracking per student | `[S§8.1.6]` | §3.2 (`learn_video_defs`, `learn_video_progress`), §18 (Phase 2) |
| Video lessons loggable as activities | `[S§8.1.6]` | §18 (Phase 2) |
| Video tier: Free (marketplace purchase required) | `[S§8.1.6]` | §16 |
| Basic progress tracking: activity counts, reading completion, hours/week | `[S§8.1.7]` | §4.3, §5, §12 |
| Advanced analytics: trends, subject balance, methodology benchmarks | `[S§8.1.7]` | §12, §18 (Phase 2 — Premium) |
| Progress per-student with date range filtering | `[S§8.1.7]` | §4.3, §6 |
| Progress tier: Free (basic), Premium (advanced) | `[S§8.1.7]` | §16 |
| 8 methodology-specific tools follow core data patterns | `[S§8.1.8]` | §15, §18 (Phase 3) |
| Per-student tool assignment with methodology override | `[S§8.2]` | §10, §5 (`get_resolved_tools`) |
| Tool assignment defaults to family methodology-derived set | `[S§8.2]` | §10 |
| Parents can manually activate/deactivate tools per student | `[S§8.2]` | §10, §18 (Phase 2) |
| Hierarchical subject taxonomy (Category → Subject → Topic) | `[S§8.3]` | §3.2 (`learn_subject_taxonomy`), §13 |
| Taxonomy extensible by admins without code changes | `[S§8.3]` | §13 |
| Users tag activities/assessments/projects with taxonomy subjects | `[S§8.3]` | §3.2 (`subject_tags TEXT[]`) |
| Users create custom subjects within family scope | `[S§8.3]` | §3.2 (`learn_custom_subjects`), §4.3, §13 |
| Taxonomy consistent across learning, marketplace, compliance | `[S§8.3]` | §13 |
| Learning tools surface methodology-specific guidance contextually | `[S§8.4]` | §10 (guidance from `method::` config_overrides) |
| Guidance sourced from methodology philosophy/mastery paths | `[S§8.4]` | §10 (delegates to `method::`) |
| Guidance accessible via help panel, not blocking workflow | `[S§8.4]` | §8 (`ActiveToolResponse.guidance`) |
| All learning data owned by the family | `[S§8.5]` | §14 |
| Export in standard machine-readable format (CSV, JSON) | `[S§8.5]` | §14 |
| Export includes all learning data types | `[S§8.5]` | §14 |
| Export available regardless of subscription tier | `[S§8.5]` | §14, §16 |
| Export completable within 24 hours | `[S§8.5]` | §14 |
| Tool registry pattern: master catalog + per-methodology activation | `[S§4.2]` | §10 (consumes from `method::`) |
| Per-student methodology overrides | `[S§4.6]` | §10 (delegates to `method::resolve_student_tools`) |
| Learning search (family-scoped) | `[S§14.1]` | §3.2 (`search_vector` columns), §4.3 |
| Free tier: basic learning tools | `[S§15.1]` | §16 |
| Premium tier: advanced analytics, enhanced storage, advanced tools | `[S§15.2]` | §16 |
| Learning → Compliance data flow | `[S§18.2]` | §17 |
| Learning → Social milestone events | `[S§18.3]` | §17 |
| Marketplace → Learning content integration | `[S§18.4]` | §17 |
| Learning → AI anonymized signals | `[S§18.5]` | §17 |
| Interactive assessment engine: question types, quiz building, auto-scoring | `[S§8.1.9]` | §3.2 (`learn_questions`, `learn_quiz_defs`, `learn_quiz_questions`, `learn_quiz_sessions`), §4.1, §5 |
| Students take quizzes online; scores auto-flow to assessment results | `[S§8.1.9]` | §3.2 (`learn_quiz_sessions`), §5 |
| Assessment engine methodology-scoped (Traditional, Classical; optional Montessori) | `[S§8.1.9]` | §10 (tool resolution via `method::`) |
| In-platform content viewer for purchased PDFs/documents | `[S§8.1.10]` | §8 (frontend-only — no new DB tables; viewing progress tracked in `learn_video_progress` or activity logs) |
| Video player: self-hosted HLS + external embeds | `[S§8.1.11]` | §3.2 (`learn_video_defs` extended), §4.1, §5 |
| Video watch position, completion %, auto-log as activity | `[S§8.1.11]` | §3.2 (`learn_video_progress`), §5 |
| Lesson sequences: ordered content paths | `[S§8.1.12]` | §3.2 (`learn_sequence_defs`, `learn_sequence_items`, `learn_sequence_progress`), §4.1, §5 |
| Sequences: linear vs recommended-order modes | `[S§8.1.12]` | §3.2 (`is_linear` on `learn_sequence_defs`) |
| Parent can override sequence order, skip, unlock ahead | `[S§8.1.12]` | §4.1, §5 |
| Supervised student views for assigned content | `[S§8.6]` | §4.1 (assignment endpoints), §17 (delegates session management to `iam::`) |
| Student assignments: assign, track, complete content | `[S§8.6.3]` | §3.2 (`learn_student_assignments`), §4.1, §5 |
| Quiz completion auto-generates assessment result | `[S§8.1.9]` | §5 (`score_quiz_session`) |
| Interactive content purchase grants quiz/sequence access | `[S§18.7]` | §17 (`PurchaseCompleted` handler) |
| QuizCompleted, SequenceAdvanced, SequenceCompleted events | `[S§18.8]` | §17 |

---

## §3 Database Schema

The learning domain uses a **three-layer data model** that cleanly separates content
definitions from family tracking data:

1. **Layer 1 — Published Content**: Structured definitions owned by Publishers (multi-user
   entities like GitHub Organizations). NOT family-scoped. Public reads, publisher-based
   writes. `[S§3.1.4, S§9]`
2. **Layer 2 — Artifact Links**: Relationships between published content ("this test is
   about this book"). NOT family-scoped. `[§9]`
3. **Layer 3 — Family Tracking**: Per-student logs and progress. Family-scoped via
   `FamilyScope`. Standalone data that *optionally* references published content.

> **Refinement note**: This refines the flat schema in ARCHITECTURE.md §5.2 into a
> content/tracking split to support marketplace curriculum integration and cross-content
> artifact linking. The existing SPEC.md §3.1.4 "Creator Account" concept is extended
> into a multi-user **Publisher** entity (owned by `mkt::`) that owns and manages all
> published educational content.

All tables use the `learn_` prefix. `[ARCH §5.1]`

### §3.1 Enums

All enums use CHECK constraints for consistency with other domain specs and to avoid
PostgreSQL enum migration limitations. `[ARCH §5.2]`

```sql
-- Reading status, journal entry type, export status, artifact link relationship,
-- and project status are all enforced via CHECK constraints on their respective
-- columns rather than as PostgreSQL enum types. This avoids ALTER TYPE limitations
-- when adding new values in future migrations. [ARCH §5.2]
```

### §3.2 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_learn_tables.go (goose migration)
-- =============================================================================

-- ═══════════════════════════════════════════════════════════════════════════════
-- LAYER 1: Published Content (owned by Publishers, NOT family-scoped)
-- ═══════════════════════════════════════════════════════════════════════════════
--
-- All definition tables reference publisher_id (the Publisher entity that owns
-- the content, living in mkt_publishers). The platform itself is a special
-- built-in Publisher for platform-provided content. [S§3.1.4, S§9]
-- ═══════════════════════════════════════════════════════════════════════════════

-- Activity/lesson definitions [S§8.1.1]
-- Structured templates that families optionally reference when logging activities.
CREATE TABLE learn_activity_defs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',   -- from taxonomy [S§8.3]
    methodology_id        UUID REFERENCES method_definitions(id),
    tool_id               UUID REFERENCES method_tools(id),
    est_duration_minutes  SMALLINT,
    attachments           JSONB NOT NULL DEFAULT '[]',    -- array of media references
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_activity_defs_publisher ON learn_activity_defs(publisher_id);
CREATE INDEX idx_learn_activity_defs_methodology ON learn_activity_defs(methodology_id)
    WHERE methodology_id IS NOT NULL;
CREATE INDEX idx_learn_activity_defs_subject ON learn_activity_defs USING GIN(subject_tags);

-- Full-text search on activity definitions [S§14.1]
ALTER TABLE learn_activity_defs ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_learn_activity_defs_search ON learn_activity_defs USING GIN(search_vector);

-- Assessment/test definitions [S§8.1.2] (Phase 2)
CREATE TABLE learn_assessment_defs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    scoring_type          TEXT NOT NULL DEFAULT 'percentage'
                          CHECK (scoring_type IN ('percentage', 'letter', 'points', 'pass_fail')),
    max_score             NUMERIC,                       -- NULL for pass_fail
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_assessment_defs_publisher ON learn_assessment_defs(publisher_id);

-- Reading item definitions [S§8.1.3]
-- Book/reading material definitions. Can be created by families or publishers.
CREATE TABLE learn_reading_items (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- Full-text search on reading items
ALTER TABLE learn_reading_items ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(author, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_learn_reading_items_search ON learn_reading_items USING GIN(search_vector);

-- Project definitions [S§8.1.5] (Phase 2)
CREATE TABLE learn_project_defs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    milestone_templates   JSONB NOT NULL DEFAULT '[]',   -- [{name, description, sort_order}]
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_project_defs_publisher ON learn_project_defs(publisher_id);

-- Video lesson definitions [S§8.1.6, S§8.1.11]
-- Marketplace integration — metadata for purchased video content.
-- Supports both self-hosted (HLS) and external (YouTube/Vimeo) video.
CREATE TABLE learn_video_defs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    methodology_id        UUID REFERENCES method_definitions(id),
    duration_seconds      INTEGER,
    thumbnail_url         TEXT,
    video_url             TEXT NOT NULL,                  -- HLS playlist URL (self-hosted) or external URL
    video_source          TEXT NOT NULL DEFAULT 'self_hosted'
                          CHECK (video_source IN ('self_hosted', 'youtube', 'vimeo')),
    external_video_id     TEXT,                           -- YouTube/Vimeo video ID (NULL for self-hosted)
    transcode_job_id      UUID,                           -- references media_transcode_jobs (NULL for external)
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_video_defs_publisher ON learn_video_defs(publisher_id);
CREATE INDEX idx_learn_video_defs_methodology ON learn_video_defs(methodology_id)
    WHERE methodology_id IS NOT NULL;

-- Question bank [S§8.1.9]
-- Individual questions created by publishers via authoring tools.
CREATE TABLE learn_questions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    question_type         TEXT NOT NULL
                          CHECK (question_type IN (
                              'multiple_choice', 'fill_in_blank', 'true_false',
                              'matching', 'ordering', 'short_answer'
                          )),
    content               TEXT NOT NULL,                  -- question text (markdown supported)
    media_attachments     JSONB NOT NULL DEFAULT '[]',    -- array of media references
    answer_data           JSONB NOT NULL,                 -- type-specific answer structure:
                          -- multiple_choice: {choices: [{text, is_correct}], allow_multiple: bool}
                          -- fill_in_blank: {acceptable_answers: [string], case_sensitive: bool}
                          -- true_false: {correct_answer: bool}
                          -- matching: {pairs: [{left, right}]}
                          -- ordering: {correct_order: [string]}
                          -- short_answer: {rubric: string} (parent-scored, no auto answer)
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    methodology_id        UUID REFERENCES method_definitions(id),
    difficulty_level      SMALLINT CHECK (difficulty_level BETWEEN 1 AND 5),
    auto_scorable         BOOLEAN NOT NULL DEFAULT true,  -- false for short_answer
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

-- Full-text search on questions
ALTER TABLE learn_questions ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(content, ''))
    ) STORED;
CREATE INDEX idx_learn_questions_search ON learn_questions USING GIN(search_vector);

-- Quiz definitions [S§8.1.9]
-- Assembled from questions by publishers via quiz builder authoring tools.
CREATE TABLE learn_quiz_defs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    methodology_id        UUID REFERENCES method_definitions(id),
    time_limit_minutes    SMALLINT,                       -- NULL = no time limit
    passing_score_percent SMALLINT NOT NULL DEFAULT 70,
    shuffle_questions     BOOLEAN NOT NULL DEFAULT false,
    show_correct_after    BOOLEAN NOT NULL DEFAULT true,  -- show correct answers after submission
    question_count        SMALLINT NOT NULL DEFAULT 0,    -- denormalized from join table
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_quiz_defs_publisher ON learn_quiz_defs(publisher_id);
CREATE INDEX idx_learn_quiz_defs_subject ON learn_quiz_defs USING GIN(subject_tags);
CREATE INDEX idx_learn_quiz_defs_methodology ON learn_quiz_defs(methodology_id)
    WHERE methodology_id IS NOT NULL;

-- Full-text search on quiz definitions
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
    points_override       NUMERIC,                        -- NULL = use question's default points
    PRIMARY KEY (quiz_def_id, question_id)
);

CREATE INDEX idx_learn_quiz_questions_question ON learn_quiz_questions(question_id);

-- Sequence definitions [S§8.1.12]
-- Ordered content paths created by publishers via sequence builder.
CREATE TABLE learn_sequence_defs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    methodology_id        UUID REFERENCES method_definitions(id),
    is_linear             BOOLEAN NOT NULL DEFAULT true,  -- true = must complete in order; false = recommended order
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_sequence_defs_publisher ON learn_sequence_defs(publisher_id);
CREATE INDEX idx_learn_sequence_defs_subject ON learn_sequence_defs USING GIN(subject_tags);
CREATE INDEX idx_learn_sequence_defs_methodology ON learn_sequence_defs(methodology_id)
    WHERE methodology_id IS NOT NULL;

-- Full-text search on sequence definitions
ALTER TABLE learn_sequence_defs ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_learn_sequence_defs_search ON learn_sequence_defs USING GIN(search_vector);

-- Sequence items [S§8.1.12]
-- Individual content steps within a sequence.
CREATE TABLE learn_sequence_items (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sequence_def_id       UUID NOT NULL REFERENCES learn_sequence_defs(id) ON DELETE CASCADE,
    sort_order            SMALLINT NOT NULL,
    content_type          TEXT NOT NULL
                          CHECK (content_type IN (
                              'activity_def', 'assessment_def', 'reading_item',
                              'video_def', 'quiz_def'
                          )),
    content_id            UUID NOT NULL,                  -- references the appropriate _defs table
    is_required           BOOLEAN NOT NULL DEFAULT true,
    unlock_after_previous BOOLEAN NOT NULL DEFAULT false, -- only enforced when sequence is_linear = true
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_sequence_items_sequence ON learn_sequence_items(sequence_def_id, sort_order);

-- ═══════════════════════════════════════════════════════════════════════════════
-- LAYER 2: Artifact Links (NOT family-scoped)
-- ═══════════════════════════════════════════════════════════════════════════════

-- Polymorphic links between published content [§9 deep-dive]
-- "This test is about this book", "This activity is part of this project"
CREATE TABLE learn_artifact_links (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
--
-- Per-student logs and progress. Some types optionally reference published
-- content definitions, others are purely standalone. All have family_id +
-- student_id and require FamilyScope for all queries. [CODING §2.4]
-- ═══════════════════════════════════════════════════════════════════════════════

-- ─── Standalone tracking (no required content reference) ──────────────────────

-- Activity logs [S§8.1.1]
-- Daily activity entries. Optional content_id references an activity_def for
-- curriculum-linked activities; NULL for ad-hoc family-logged activities.
CREATE TABLE learn_activity_logs (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    title                 TEXT NOT NULL,
    description           TEXT,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',   -- from taxonomy [S§8.3]
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

-- Full-text search on activity logs (family-scoped) [S§14.1]
ALTER TABLE learn_activity_logs ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, ''))
    ) STORED;
CREATE INDEX idx_learn_activity_logs_search ON learn_activity_logs USING GIN(search_vector);

-- Journal entries [S§8.1.4]
-- Narration records, free-form journals, and reflections.
-- Optional content_id for prompted entries (e.g., from a curriculum).
CREATE TABLE learn_journal_entries (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    entry_type            TEXT NOT NULL CHECK (entry_type IN ('freeform', 'narration', 'reflection')),
    title                 TEXT,
    content               TEXT NOT NULL,
    subject_tags          TEXT[] NOT NULL DEFAULT '{}',
    content_id            UUID,                          -- optional ref to a definition
    attachments           JSONB NOT NULL DEFAULT '[]',
    entry_date            DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_journal_entries_family_student
    ON learn_journal_entries(family_id, student_id, entry_date DESC);
CREATE INDEX idx_learn_journal_entries_type
    ON learn_journal_entries(family_id, entry_type);

-- Full-text search on journal entries [S§14.1]
ALTER TABLE learn_journal_entries ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        to_tsvector('english', coalesce(title, '') || ' ' || coalesce(content, ''))
    ) STORED;
CREATE INDEX idx_learn_journal_entries_search ON learn_journal_entries USING GIN(search_vector);

-- Progress snapshots [S§8.1.7]
-- Periodic computed summaries. Written by background job, read by dashboard.
CREATE TABLE learn_progress_snapshots (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    snapshot_date         DATE NOT NULL,
    data                  JSONB NOT NULL,                 -- {activity_count, hours_by_subject, reading_completed, ...}
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_learn_progress_snapshots_unique
    ON learn_progress_snapshots(family_id, student_id, snapshot_date);

-- ─── Content-referenced tracking (references published definitions) ───────────

-- Reading progress [S§8.1.3]
-- Per-student tracking of reading status for a specific reading item.
CREATE TABLE learn_reading_progress (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    reading_item_id       UUID NOT NULL REFERENCES learn_reading_items(id) ON DELETE CASCADE,
    reading_list_id       UUID REFERENCES learn_reading_lists(id) ON DELETE SET NULL,
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

-- Assessment results [S§8.1.2] (Phase 2)
-- Per-student test scores referencing an assessment definition.
CREATE TABLE learn_assessment_results (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- Project progress [S§8.1.5] (Phase 2)
-- Per-student project tracking with milestone completion.
CREATE TABLE learn_project_progress (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    project_def_id        UUID NOT NULL REFERENCES learn_project_defs(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'planning'
                          CHECK (status IN ('planning', 'in_progress', 'completed')),
    milestones            JSONB NOT NULL DEFAULT '[]',    -- [{name, completed, completed_at, notes}]
    started_at            TIMESTAMPTZ,
    completed_at          TIMESTAMPTZ,
    notes                 TEXT,
    attachments           JSONB NOT NULL DEFAULT '[]',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_project_progress_family_student
    ON learn_project_progress(family_id, student_id, status);

-- Video progress [S§8.1.6] (Phase 2)
-- Per-student video watch progress.
CREATE TABLE learn_video_progress (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- Quiz sessions [S§8.1.9] (family-scoped)
-- Per-student quiz-taking sessions. Tracks in-progress and completed quiz attempts.
CREATE TABLE learn_quiz_sessions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    quiz_def_id           UUID NOT NULL REFERENCES learn_quiz_defs(id) ON DELETE CASCADE,
    status                TEXT NOT NULL DEFAULT 'not_started'
                          CHECK (status IN ('not_started', 'in_progress', 'submitted', 'scored')),
    started_at            TIMESTAMPTZ,
    submitted_at          TIMESTAMPTZ,
    scored_at             TIMESTAMPTZ,
    score                 NUMERIC,                        -- total points earned
    max_score             NUMERIC,                        -- total possible points
    passed                BOOLEAN,                        -- NULL until scored
    answers               JSONB NOT NULL DEFAULT '[]',    -- [{question_id, response, is_correct, points_awarded}]
    scored_by             UUID REFERENCES iam_parents(id), -- NULL = auto-scored, parent_id = parent-scored
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

-- Sequence progress [S§8.1.12] (family-scoped)
-- Per-student progress through a lesson sequence.
CREATE TABLE learn_sequence_progress (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    sequence_def_id       UUID NOT NULL REFERENCES learn_sequence_defs(id) ON DELETE CASCADE,
    current_item_index    SMALLINT NOT NULL DEFAULT 0,
    status                TEXT NOT NULL DEFAULT 'not_started'
                          CHECK (status IN ('not_started', 'in_progress', 'completed')),
    item_completions      JSONB NOT NULL DEFAULT '[]',    -- [{item_id, completed_at, score?}]
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

-- Student assignments [S§8.6.3] (family-scoped)
-- Parent-assigned content for supervised student sessions.
CREATE TABLE learn_student_assignments (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    assigned_by           UUID NOT NULL REFERENCES iam_parents(id),
    content_type          TEXT NOT NULL
                          CHECK (content_type IN (
                              'activity_def', 'reading_item', 'video_def',
                              'quiz_def', 'sequence_def'
                          )),
    content_id            UUID NOT NULL,
    due_date              DATE,                            -- optional
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

-- ─── Supporting tables ────────────────────────────────────────────────────────

-- Subject taxonomy [S§8.3]
-- Platform-managed hierarchical taxonomy. No family_id — shared across platform.
-- Admin-editable, no code changes required to add subjects.
CREATE TABLE learn_subject_taxonomy (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id             UUID REFERENCES learn_subject_taxonomy(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    slug                  TEXT NOT NULL UNIQUE,           -- e.g., 'math.algebra.linear-equations'
    level                 SMALLINT NOT NULL,              -- 0 = Category, 1 = Subject, 2 = Topic
    display_order         SMALLINT NOT NULL DEFAULT 0,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_subject_taxonomy_parent ON learn_subject_taxonomy(parent_id);
CREATE INDEX idx_learn_subject_taxonomy_level ON learn_subject_taxonomy(level, display_order);

-- Family-scoped custom subjects [S§8.3]
-- Extend (not replace) the platform taxonomy within a family's scope.
CREATE TABLE learn_custom_subjects (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    parent_taxonomy_id    UUID REFERENCES learn_subject_taxonomy(id),
    name                  TEXT NOT NULL,
    slug                  TEXT NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_custom_subject UNIQUE (family_id, slug)
);

CREATE INDEX idx_learn_custom_subjects_family ON learn_custom_subjects(family_id);

-- Reading lists [S§8.1.3]
-- Named groupings of reading items. Family-scoped.
CREATE TABLE learn_reading_lists (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,
    description           TEXT,
    student_id            UUID REFERENCES iam_students(id) ON DELETE SET NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_reading_lists_family ON learn_reading_lists(family_id);

-- Reading list items (join table)
CREATE TABLE learn_reading_list_items (
    reading_list_id       UUID NOT NULL REFERENCES learn_reading_lists(id) ON DELETE CASCADE,
    reading_item_id       UUID NOT NULL REFERENCES learn_reading_items(id) ON DELETE CASCADE,
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    added_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (reading_list_id, reading_item_id)
);

-- Grading scales [S§8.1.2] (Phase 2)
-- Custom grading scales per family.
CREATE TABLE learn_grading_scales (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    name                  TEXT NOT NULL,                  -- e.g., "Standard Letter Grades"
    scale_type            TEXT NOT NULL CHECK (scale_type IN ('letter', 'pass_fail', 'custom')),
    grades                JSONB NOT NULL,                 -- [{label, min_score, max_score}]
    is_default            BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_grading_scales_family ON learn_grading_scales(family_id);

-- Data export requests [S§8.5]
CREATE TABLE learn_export_requests (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    requested_by          UUID NOT NULL REFERENCES iam_parents(id),
    status                TEXT NOT NULL DEFAULT 'pending'
                          CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    file_url              TEXT,                           -- media:: managed download URL
    expires_at            TIMESTAMPTZ,                    -- 24-hour download window
    error_message         TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_learn_export_requests_family ON learn_export_requests(family_id, created_at DESC);
```

### §3.3 RLS Policies

The learning domain has three distinct RLS strategies matching its three-layer data model.
`[CODING §3.2]`

```sql
-- ═══ Layer 3: Family Tracking — standard family-scoped RLS ═══════════════════

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

-- ═══ Layer 1: Published Content — NO family-scoped RLS ═══════════════════════
-- Access control is publisher-based for writes and public for reads.
-- Publisher membership checks happen at application level via mkt:: service.
-- No RLS on: learn_activity_defs, learn_assessment_defs, learn_reading_items,
--            learn_project_defs, learn_video_defs, learn_questions,
--            learn_quiz_defs, learn_quiz_questions, learn_sequence_defs,
--            learn_sequence_items

-- ═══ Layer 2: Artifact Links — NO RLS ════════════════════════════════════════
-- Links between published content are shared. No family_id column.
-- No RLS on: learn_artifact_links

-- ═══ Supporting: Subject Taxonomy — NO RLS ═══════════════════════════════════
-- Platform-managed taxonomy is shared across all users.
-- No RLS on: learn_subject_taxonomy
```

---

## §4 API Endpoints

All learning endpoints require authentication. Layer 3 (family tracking) endpoints require
`FamilyScope`. Layer 1 (published content) endpoints require authentication but use
publisher-based access control for writes. `[S§3.2]`

### §4.1 Phase 1 Endpoints

#### Activity Definitions (Layer 1)

##### `POST /v1/learning/activities/defs`

Creates an activity definition. Caller must be a member of the specified publisher.

- **Auth**: Required (publisher membership check)
- **Body**: `CreateActivityDefCommand` (`publisher_id`, `title`, `description?`, `subject_tags`, `methodology_id?`, `tool_id?`, `est_duration_minutes?`, `attachments`)
- **Validation**: `title` max 500 chars; `subject_tags` must exist in taxonomy; `publisher_id` caller must be member
- **Response**: `ActivityDefResponse` (201 Created)

##### `GET /v1/learning/activities/defs`

Browse/search activity definitions.

- **Auth**: Required
- **Query**: `?subject=<slug>&methodology_id=<uuid>&publisher_id=<uuid>&q=<search>&cursor=<uuid>&limit=20`
- **Response**: `PaginatedResponse[ActivityDefSummaryResponse]` (200 OK)

##### `GET /v1/learning/activities/defs/:id`

Returns a single activity definition.

- **Auth**: Required
- **Response**: `ActivityDefResponse` (200 OK)
- **404**: Definition not found

##### `PATCH /v1/learning/activities/defs/:id`

Updates an activity definition. Publisher member only.

- **Auth**: Required (publisher membership check)
- **Body**: `UpdateActivityDefCommand` (partial fields)
- **Response**: `ActivityDefResponse` (200 OK)
- **Error codes**: `not_publisher_member` (403)

##### `DELETE /v1/learning/activities/defs/:id`

Soft-deletes an activity definition (sets `is_active = false`). Publisher member only.

- **Auth**: Required (publisher membership check)
- **Response**: 204 No Content
- **Error codes**: `not_publisher_member` (403)

#### Reading Items (Layer 1)

##### `POST /v1/learning/reading-items`

Creates a reading item definition.

- **Auth**: Required (publisher membership check)
- **Body**: `CreateReadingItemCommand` (`publisher_id`, `title`, `author?`, `isbn?`, `subject_tags`, `description?`, `cover_image_url?`, `page_count?`)
- **Response**: `ReadingItemResponse` (201 Created)

##### `GET /v1/learning/reading-items`

Browse/search reading items.

- **Auth**: Required
- **Query**: `?q=<search>&subject=<slug>&isbn=<isbn>&cursor=<uuid>&limit=20`
- **Response**: `PaginatedResponse[ReadingItemSummaryResponse]` (200 OK)

##### `GET /v1/learning/reading-items/:id`

Returns a single reading item with linked artifacts.

- **Auth**: Required
- **Response**: `ReadingItemDetailResponse` (200 OK)
- **404**: Item not found

##### `PATCH /v1/learning/reading-items/:id`

Updates a reading item. Publisher member only.

- **Auth**: Required (publisher membership check)
- **Body**: `UpdateReadingItemCommand` (partial fields)
- **Response**: `ReadingItemResponse` (200 OK)

#### Artifact Links (Layer 2)

##### `POST /v1/learning/links`

Creates a link between two published content items.

- **Auth**: Required (publisher membership check — must own source content)
- **Body**: `CreateArtifactLinkCommand` (`source_type`, `source_id`, `target_type`, `target_id`, `relationship?`)
- **Response**: `ArtifactLinkResponse` (201 Created)
- **Error codes**: `source_not_found` (404), `target_not_found` (404), `duplicate_link` (409)

##### `GET /v1/learning/links`

Gets all artifacts linked to a specific content item.

- **Auth**: Required
- **Query**: `?type=<content_type>&id=<uuid>&direction=source|target|both`
- **Response**: `[]ArtifactLinkResponse` (200 OK)

##### `DELETE /v1/learning/links/:id`

Removes an artifact link. Must own the source content.

- **Auth**: Required (publisher membership check)
- **Response**: 204 No Content

#### Activity Logs (Layer 3 — family-scoped)

##### `POST /v1/learning/students/:student_id/activities`

Logs an activity for a student. May reference an activity definition or be ad-hoc. `[S§8.1.1]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `LogActivityCommand` (`title`, `description?`, `subject_tags`, `content_id?`, `methodology_id?`, `tool_id?`, `duration_minutes?`, `attachments`, `activity_date?`)
- **Validation**: Student must belong to family; subject tags must exist in taxonomy or custom subjects; `activity_date` cannot be in the future; duration non-negative
- **Response**: `ActivityLogResponse` (201 Created)
- **Events**: `ActivityLogged`

##### `GET /v1/learning/students/:student_id/activities`

Lists activity logs for a student, filterable. `[S§8.1.1]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?subject=<slug>&date_from=<date>&date_to=<date>&cursor=<uuid>&limit=20`
- **Response**: `PaginatedResponse[ActivityLogResponse]` (200 OK)

##### `GET /v1/learning/students/:student_id/activities/:id`

Returns a single activity log entry.

- **Auth**: Required (`FamilyScope`)
- **Response**: `ActivityLogResponse` (200 OK)
- **404**: Activity not found

##### `PATCH /v1/learning/students/:student_id/activities/:id`

Updates an activity log entry.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateActivityLogCommand` (partial fields)
- **Response**: `ActivityLogResponse` (200 OK)

##### `DELETE /v1/learning/students/:student_id/activities/:id`

Deletes an activity log entry.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

#### Journal Entries (Layer 3 — family-scoped)

##### `POST /v1/learning/students/:student_id/journals`

Creates a journal entry. `[S§8.1.4]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `CreateJournalEntryCommand` (`entry_type`, `title?`, `content`, `subject_tags`, `content_id?`, `attachments`, `entry_date?`)
- **Validation**: `entry_type` must be valid; student must belong to family; `entry_date` cannot be in the future
- **Response**: `JournalEntryResponse` (201 Created)

##### `GET /v1/learning/students/:student_id/journals`

Lists journal entries for a student. `[S§8.1.4]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?entry_type=<type>&date_from=<date>&date_to=<date>&q=<search>&cursor=<uuid>&limit=20`
- **Response**: `PaginatedResponse[JournalEntryResponse]` (200 OK)

##### `GET /v1/learning/students/:student_id/journals/:id`

Returns a single journal entry.

- **Auth**: Required (`FamilyScope`)
- **Response**: `JournalEntryResponse` (200 OK)

##### `PATCH /v1/learning/students/:student_id/journals/:id`

Updates a journal entry.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateJournalEntryCommand` (partial fields)
- **Response**: `JournalEntryResponse` (200 OK)

##### `DELETE /v1/learning/students/:student_id/journals/:id`

Deletes a journal entry.

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

#### Reading Progress (Layer 3 — family-scoped)

##### `POST /v1/learning/students/:student_id/reading`

Starts tracking a reading item for a student. `[S§8.1.3]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `StartReadingCommand` (`reading_item_id`, `reading_list_id?`)
- **Validation**: Student must belong to family; reading item must exist; no duplicate tracking
- **Response**: `ReadingProgressResponse` (201 Created)

##### `PATCH /v1/learning/students/:student_id/reading/:id`

Updates reading progress (status, notes). `[S§8.1.3]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateReadingProgressCommand` (`status?`, `notes?`)
- **Validation**: Status transitions: to_read → in_progress → completed; completing auto-generates activity log
- **Response**: `ReadingProgressResponse` (200 OK)
- **Events**: `BookCompleted` (when status → completed)

##### `GET /v1/learning/students/:student_id/reading`

Lists reading progress for a student.

- **Auth**: Required (`FamilyScope`)
- **Query**: `?status=<status>&cursor=<uuid>&limit=20`
- **Response**: `PaginatedResponse[ReadingProgressResponse]` (200 OK)

#### Reading Lists (Layer 3 — family-scoped)

##### `POST /v1/learning/reading-lists`

Creates a named reading list. `[S§8.1.3]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `CreateReadingListCommand` (`name`, `description?`, `student_id?`, `reading_item_ids?`)
- **Response**: `ReadingListResponse` (201 Created)

##### `GET /v1/learning/reading-lists`

Lists the family's reading lists.

- **Auth**: Required (`FamilyScope`)
- **Response**: `[]ReadingListSummaryResponse` (200 OK)

##### `GET /v1/learning/reading-lists/:id`

Returns a reading list with items and student progress.

- **Auth**: Required (`FamilyScope`)
- **Response**: `ReadingListDetailResponse` (200 OK)

##### `PATCH /v1/learning/reading-lists/:id`

Updates reading list metadata or items.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateReadingListCommand` (`name?`, `description?`, `add_item_ids?`, `remove_item_ids?`)
- **Response**: `ReadingListResponse` (200 OK)

##### `DELETE /v1/learning/reading-lists/:id`

Deletes a reading list (does not delete reading items or progress).

- **Auth**: Required (`FamilyScope`)
- **Response**: 204 No Content

#### Questions (Layer 1 — publisher-scoped) `[S§8.1.9]`

##### `POST /v1/learning/questions`

Creates a question. Caller must be a member of the specified publisher.

- **Auth**: Required (publisher membership check)
- **Body**: `CreateQuestionCommand` (`publisher_id`, `question_type`, `content`, `media_attachments`, `answer_data`, `subject_tags`, `methodology_id?`, `difficulty_level?`, `points?`)
- **Validation**: `question_type` must be valid; `answer_data` must match question type schema; `subject_tags` must exist in taxonomy
- **Response**: `QuestionResponse` (201 Created)

##### `GET /v1/learning/questions`

Browse/search questions for quiz building.

- **Auth**: Required (publisher membership check for write context; any auth for read)
- **Query**: `?publisher_id=<uuid>&question_type=<type>&subject=<slug>&methodology_id=<uuid>&q=<search>&cursor=<uuid>&limit=20`
- **Response**: `PaginatedResponse[QuestionSummaryResponse]` (200 OK)

##### `PATCH /v1/learning/questions/:id`

Updates a question. Publisher member only.

- **Auth**: Required (publisher membership check)
- **Body**: `UpdateQuestionCommand` (partial fields)
- **Response**: `QuestionResponse` (200 OK)

#### Quiz Definitions (Layer 1 — publisher-scoped) `[S§8.1.9]`

##### `POST /v1/learning/quizzes`

Creates a quiz from existing questions. Publisher member only.

- **Auth**: Required (publisher membership check)
- **Body**: `CreateQuizDefCommand` (`publisher_id`, `title`, `description?`, `subject_tags`, `methodology_id?`, `time_limit_minutes?`, `passing_score_percent?`, `shuffle_questions?`, `show_correct_after?`, `question_ids: [{question_id, sort_order, points_override?}]`)
- **Validation**: All question IDs must exist and belong to same publisher; at least 1 question
- **Response**: `QuizDefResponse` (201 Created)

##### `GET /v1/learning/quizzes/:id`

Returns a quiz definition with questions (without answers if requested by student session).

- **Auth**: Required
- **Query**: `?include_answers=false` (for student view — omit correct answers)
- **Response**: `QuizDefDetailResponse` (200 OK)

##### `PATCH /v1/learning/quizzes/:id`

Updates a quiz definition. Publisher member only.

- **Auth**: Required (publisher membership check)
- **Body**: `UpdateQuizDefCommand` (partial fields, including question list updates)
- **Response**: `QuizDefResponse` (200 OK)

#### Quiz Sessions (Layer 3 — family-scoped) `[S§8.1.9]`

##### `POST /v1/learning/students/:student_id/quiz-sessions`

Starts a quiz session for a student.

- **Auth**: Required (`FamilyScope`)
- **Body**: `StartQuizSessionCommand` (`quiz_def_id`)
- **Validation**: Student must belong to family; quiz must be active; assessment-engine tool must be in student's resolved tools
- **Response**: `QuizSessionResponse` (201 Created)
- **Note**: Returns quiz questions (without correct answers) in the response

##### `PATCH /v1/learning/students/:student_id/quiz-sessions/:id`

Saves progress or submits a quiz session.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateQuizSessionCommand` (`answers?`, `submit?: bool`)
- **Validation**: Session must be in `not_started` or `in_progress` status; cannot modify after submission
- **Response**: `QuizSessionResponse` (200 OK)
- **Side effects**: On submit: auto-scores objective questions, sets status to `submitted` (if has short_answer) or `scored` (if all auto-scorable), generates `learn_assessment_results` entry
- **Events**: `QuizCompleted` (when fully scored)

##### `GET /v1/learning/students/:student_id/quiz-sessions/:id`

Returns quiz session state (for resume or review).

- **Auth**: Required (`FamilyScope`)
- **Response**: `QuizSessionResponse` (200 OK) — includes answers and scores if submitted

##### `POST /v1/learning/students/:student_id/quiz-sessions/:id/score`

Parent scores open-ended (short_answer) questions on a submitted quiz.

- **Auth**: Required (`FamilyScope` — must be parent, not student session)
- **Body**: `ScoreQuizCommand` (`scores: [{question_id, points_awarded, feedback?}]`)
- **Validation**: Session must be in `submitted` status; only short_answer questions can be manually scored
- **Response**: `QuizSessionResponse` (200 OK)
- **Events**: `QuizCompleted`

#### Sequence Definitions (Layer 1 — publisher-scoped) `[S§8.1.12]`

##### `POST /v1/learning/sequences`

Creates a lesson sequence. Publisher member only.

- **Auth**: Required (publisher membership check)
- **Body**: `CreateSequenceDefCommand` (`publisher_id`, `title`, `description?`, `subject_tags`, `methodology_id?`, `is_linear?`, `items: [{content_type, content_id, sort_order, is_required?, unlock_after_previous?}]`)
- **Validation**: All content IDs must exist; at least 1 item
- **Response**: `SequenceDefResponse` (201 Created)

##### `GET /v1/learning/sequences/:id`

Returns a sequence definition with items.

- **Auth**: Required
- **Response**: `SequenceDefDetailResponse` (200 OK) — includes items with content summaries

##### `PATCH /v1/learning/sequences/:id`

Updates a sequence definition. Publisher member only.

- **Auth**: Required (publisher membership check)
- **Body**: `UpdateSequenceDefCommand` (partial fields, including item list updates)
- **Response**: `SequenceDefResponse` (200 OK)

#### Sequence Progress (Layer 3 — family-scoped) `[S§8.1.12]`

##### `POST /v1/learning/students/:student_id/sequence-progress`

Starts a sequence for a student.

- **Auth**: Required (`FamilyScope`)
- **Body**: `StartSequenceCommand` (`sequence_def_id`)
- **Validation**: Student must belong to family; sequence must be active; lesson-sequences tool must be in student's resolved tools
- **Response**: `SequenceProgressResponse` (201 Created)

##### `GET /v1/learning/students/:student_id/sequence-progress/:id`

Returns sequence progress with per-item completion status.

- **Auth**: Required (`FamilyScope`)
- **Response**: `SequenceProgressResponse` (200 OK)

##### `PATCH /v1/learning/students/:student_id/sequence-progress/:id`

Advances, skips, or unlocks items in a sequence.

- **Auth**: Required (`FamilyScope`)
- **Body**: `UpdateSequenceProgressCommand` (`complete_item_id?`, `skip_item_id?`, `unlock_item_id?`)
- **Validation**: If linear sequence, cannot skip required items (unless parent override); if unlocking, must be parent (not student session)
- **Response**: `SequenceProgressResponse` (200 OK)
- **Events**: `SequenceAdvanced` (on item completion), `SequenceCompleted` (when all required items done)

#### Student Assignments (Layer 3 — family-scoped) `[S§8.6.3]`

##### `POST /v1/learning/students/:student_id/assignments`

Assigns content to a student. Parent auth required.

- **Auth**: Required (`FamilyScope` — must be parent, not student session)
- **Body**: `CreateAssignmentCommand` (`content_type`, `content_id`, `due_date?`)
- **Validation**: Student must belong to family; content must exist and be active
- **Response**: `AssignmentResponse` (201 Created)

##### `GET /v1/learning/students/:student_id/assignments`

Lists assignments for a student. Accessible by both parent and student sessions.

- **Auth**: Required (`FamilyScope` or student session)
- **Query**: `?status=<status>&due_before=<date>&cursor=<uuid>&limit=20`
- **Response**: `PaginatedResponse[AssignmentResponse]` (200 OK)

##### `PATCH /v1/learning/students/:student_id/assignments/:id`

Updates assignment status (e.g., mark in-progress or completed).

- **Auth**: Required (`FamilyScope` or student session for status updates only)
- **Body**: `UpdateAssignmentCommand` (`status?`, `due_date?`)
- **Response**: `AssignmentResponse` (200 OK)
- **Events**: `AssignmentCompleted` (when status → completed)

##### `DELETE /v1/learning/students/:student_id/assignments/:id`

Removes an assignment. Parent auth required.

- **Auth**: Required (`FamilyScope` — must be parent, not student session)
- **Response**: 204 No Content

### §4.2 Cross-Cutting Endpoints

#### Progress

##### `GET /v1/learning/progress/:student_id`

Returns progress summary for a student. `[S§8.1.7]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?date_from=<date>&date_to=<date>`
- **Response**: `ProgressSummaryResponse` (200 OK) — activity counts, hours by subject, reading completion

##### `GET /v1/learning/progress/:student_id/subjects`

Returns per-subject breakdown for a student.

- **Auth**: Required (`FamilyScope`)
- **Query**: `?date_from=<date>&date_to=<date>`
- **Response**: `[]SubjectProgressResponse` (200 OK)

##### `GET /v1/learning/progress/:student_id/timeline`

Returns activity timeline for a student.

- **Auth**: Required (`FamilyScope`)
- **Query**: `?date_from=<date>&date_to=<date>&cursor=<uuid>&limit=50`
- **Response**: `PaginatedResponse[TimelineEntryResponse]` (200 OK)

#### Tools

##### `GET /v1/learning/tools`

Returns the family's resolved tool set. Delegates to `method::`. `[S§4.2]`

- **Auth**: Required (`FamilyScope`)
- **Response**: `[]ActiveToolResponse` (200 OK)

##### `GET /v1/learning/tools/:student_id`

Returns a student-specific resolved tool set. `[S§4.6, S§8.2]`

- **Auth**: Required (`FamilyScope`)
- **Response**: `[]ActiveToolResponse` (200 OK)

#### Subject Taxonomy

##### `GET /v1/learning/subjects`

Returns the subject taxonomy tree (platform + family custom subjects). `[S§8.3]`

- **Auth**: Required (`FamilyScope`)
- **Query**: `?level=<0|1|2>&parent_id=<uuid>`
- **Response**: `[]SubjectTaxonomyResponse` (200 OK)

##### `POST /v1/learning/subjects/custom`

Creates a family-scoped custom subject. `[S§8.3]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `CreateCustomSubjectCommand` (`name`, `parent_taxonomy_id?`)
- **Validation**: Name must be unique within family; slug auto-generated
- **Response**: `CustomSubjectResponse` (201 Created)
- **Error codes**: `duplicate_custom_subject` (409)

#### Data Export

##### `POST /v1/learning/export`

Requests a data export. Async — generates in background. `[S§8.5]`

- **Auth**: Required (`FamilyScope`)
- **Body**: `RequestExportCommand` (`format?`: `json` | `csv`, default `json`)
- **Validation**: No concurrent export in progress for this family
- **Response**: `ExportRequestResponse` (202 Accepted)
- **Events**: `DataExportReady` (when generation completes, via `notify::`)
- **Error codes**: `export_already_in_progress` (409)

##### `GET /v1/learning/export/:id`

Downloads a completed export or checks status.

- **Auth**: Required (`FamilyScope`)
- **Response**: `ExportRequestResponse` (200 OK) with `file_url` when completed
- **Error codes**: `export_not_ready` (202), `export_expired` (410)

### §4.3 Phase 2 Endpoints

- Project definition CRUD (5 endpoints — same pattern as activity defs)
- Project progress CRUD per student (5 endpoints)
- Grading scale CRUD per family (4 endpoints)
- Advanced analytics endpoints (3 endpoints)
- Reading list sharing (2 endpoints)
- ISBN book search/import (1 endpoint)
- Content annotations/bookmarks (3 endpoints)

### §4.4 Phase 3 Endpoints

- Methodology-specific tool definition + instance endpoints (per tool — see §15)

**Phase 1 total**: ~72 endpoints (Definitions: 12, Activity Logs: 5, Journals: 5,
Reading Progress: 3, Reading Lists: 5, Progress: 3, Tools: 2, Taxonomy: 2,
Export: 2, Links: 3, **Questions: 3, Quizzes: 3, Quiz Sessions: 4,
Sequences: 3, Sequence Progress: 3, Assignments: 4**).

**Phase 2 total**: ~23 endpoints (Projects: 10, Grading: 4, Advanced analytics: 3,
Sharing: 2, Import: 1, Annotations: 3).

**Phase 3 total**: ~24 endpoints (8 methodology-specific tools × ~3 endpoints each).

---

## §5 Service Interface

The `LearningService` interface defines all use cases exposed to handlers and other domains.
Defined in `internal/learn/ports.go`. Methods are organized with CQRS separation: command
methods (writes with side effects) are separated from query methods (reads). `[CODING §8.2, ARCH §4.7]`

```go
// internal/learn/ports.go

// LearningService defines all learning domain use cases.
type LearningService interface {
    // === COMMAND SIDE (writes with side effects) ============================

    // --- Definition Commands (Layer 1 — publisher-based access) -------------

    // CreateActivityDef creates an activity definition. Publisher membership required. [S§8.1.1]
    CreateActivityDef(ctx context.Context, cmd CreateActivityDefCommand) (ActivityDefResponse, error)

    // UpdateActivityDef updates an activity definition. Publisher membership required.
    UpdateActivityDef(ctx context.Context, defID uuid.UUID, cmd UpdateActivityDefCommand) (ActivityDefResponse, error)

    // DeleteActivityDef soft-deletes an activity definition. Publisher membership required.
    DeleteActivityDef(ctx context.Context, defID uuid.UUID, callerID uuid.UUID) error

    // CreateReadingItem creates a reading item definition. Publisher membership required. [S§8.1.3]
    CreateReadingItem(ctx context.Context, cmd CreateReadingItemCommand) (ReadingItemResponse, error)

    // UpdateReadingItem updates a reading item. Publisher membership required.
    UpdateReadingItem(ctx context.Context, itemID uuid.UUID, cmd UpdateReadingItemCommand) (ReadingItemResponse, error)

    // LinkArtifacts creates an artifact link between two published content items. [§9]
    LinkArtifacts(ctx context.Context, cmd CreateArtifactLinkCommand) (ArtifactLinkResponse, error)

    // UnlinkArtifacts removes an artifact link. Must own source content.
    UnlinkArtifacts(ctx context.Context, linkID uuid.UUID, callerID uuid.UUID) error

    // --- Instance Commands (Layer 3 — FamilyScope required) -----------------

    // LogActivity logs an activity for a student. [S§8.1.1]
    LogActivity(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, cmd LogActivityCommand) (ActivityLogResponse, error)

    // UpdateActivityLog updates an activity log entry.
    UpdateActivityLog(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, logID uuid.UUID, cmd UpdateActivityLogCommand) (ActivityLogResponse, error)

    // DeleteActivityLog deletes an activity log entry.
    DeleteActivityLog(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, logID uuid.UUID) error

    // CreateJournalEntry creates a journal entry for a student. [S§8.1.4]
    CreateJournalEntry(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, cmd CreateJournalEntryCommand) (JournalEntryResponse, error)

    // UpdateJournalEntry updates a journal entry.
    UpdateJournalEntry(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, entryID uuid.UUID, cmd UpdateJournalEntryCommand) (JournalEntryResponse, error)

    // DeleteJournalEntry deletes a journal entry.
    DeleteJournalEntry(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, entryID uuid.UUID) error

    // StartReading starts tracking a reading item for a student. [S§8.1.3]
    StartReading(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, cmd StartReadingCommand) (ReadingProgressResponse, error)

    // UpdateReadingProgress updates reading progress. Completing triggers BookCompleted event.
    UpdateReadingProgress(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateReadingProgressCommand) (ReadingProgressResponse, error)

    // CreateReadingList creates a named reading list. [S§8.1.3]
    CreateReadingList(ctx context.Context, scope *FamilyScope, cmd CreateReadingListCommand) (ReadingListResponse, error)

    // UpdateReadingList updates a reading list (metadata and items).
    UpdateReadingList(ctx context.Context, scope *FamilyScope, listID uuid.UUID, cmd UpdateReadingListCommand) (ReadingListResponse, error)

    // DeleteReadingList deletes a reading list.
    DeleteReadingList(ctx context.Context, scope *FamilyScope, listID uuid.UUID) error

    // CreateCustomSubject creates a family-scoped custom subject. [S§8.3]
    CreateCustomSubject(ctx context.Context, scope *FamilyScope, cmd CreateCustomSubjectCommand) (CustomSubjectResponse, error)

    // RequestDataExport requests an async data export. [S§8.5]
    RequestDataExport(ctx context.Context, scope *FamilyScope, cmd RequestExportCommand) (ExportRequestResponse, error)

    // --- Event Handlers ------------------------------------------------------

    // HandleStudentCreated handles StudentCreated event — initialize student learning defaults.
    HandleStudentCreated(ctx context.Context, familyID FamilyID, studentID uuid.UUID) error

    // HandleStudentDeleted handles StudentDeleted event — cascade-delete learning data.
    HandleStudentDeleted(ctx context.Context, familyID FamilyID, studentID uuid.UUID) error

    // HandleFamilyDeletionScheduled handles FamilyDeletionScheduled — trigger export opportunity.
    HandleFamilyDeletionScheduled(ctx context.Context, familyID FamilyID) error

    // HandlePurchaseCompleted handles PurchaseCompleted — integrate purchased content (including interactive content).
    HandlePurchaseCompleted(ctx context.Context, familyID FamilyID, purchaseMetadata PurchaseMetadata) error

    // HandleMethodologyConfigUpdated handles MethodologyConfigUpdated — invalidate tool cache.
    HandleMethodologyConfigUpdated(ctx context.Context) error

    // --- Assessment Engine Commands (Layer 1 + Layer 3) ----- [S§8.1.9]

    // CreateQuestion creates a question. Publisher membership required.
    CreateQuestion(ctx context.Context, cmd CreateQuestionCommand) (QuestionResponse, error)

    // UpdateQuestion updates a question. Publisher membership required.
    UpdateQuestion(ctx context.Context, questionID uuid.UUID, cmd UpdateQuestionCommand) (QuestionResponse, error)

    // CreateQuizDef creates a quiz definition from questions. Publisher membership required.
    CreateQuizDef(ctx context.Context, cmd CreateQuizDefCommand) (QuizDefResponse, error)

    // UpdateQuizDef updates a quiz definition. Publisher membership required.
    UpdateQuizDef(ctx context.Context, quizDefID uuid.UUID, cmd UpdateQuizDefCommand) (QuizDefResponse, error)

    // StartQuizSession starts a quiz session for a student.
    StartQuizSession(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, cmd StartQuizSessionCommand) (QuizSessionResponse, error)

    // UpdateQuizSession saves progress or submits a quiz session.
    UpdateQuizSession(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd UpdateQuizSessionCommand) (QuizSessionResponse, error)

    // ScoreQuizSession allows a parent to score short-answer questions on a submitted quiz.
    ScoreQuizSession(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd ScoreQuizCommand) (QuizSessionResponse, error)

    // --- Sequence Engine Commands (Layer 1 + Layer 3) ------- [S§8.1.12]

    // CreateSequenceDef creates a lesson sequence. Publisher membership required.
    CreateSequenceDef(ctx context.Context, cmd CreateSequenceDefCommand) (SequenceDefResponse, error)

    // UpdateSequenceDef updates a sequence definition. Publisher membership required.
    UpdateSequenceDef(ctx context.Context, sequenceDefID uuid.UUID, cmd UpdateSequenceDefCommand) (SequenceDefResponse, error)

    // StartSequence starts a sequence for a student.
    StartSequence(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, cmd StartSequenceCommand) (SequenceProgressResponse, error)

    // UpdateSequenceProgress advances, skips, or unlocks items in a sequence.
    UpdateSequenceProgress(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateSequenceProgressCommand) (SequenceProgressResponse, error)

    // --- Assignment Commands (Layer 3) ---------------------- [S§8.6.3]

    // CreateAssignment assigns content to a student. Parent auth required.
    CreateAssignment(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, cmd CreateAssignmentCommand) (AssignmentResponse, error)

    // UpdateAssignment updates assignment status.
    UpdateAssignment(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID, cmd UpdateAssignmentCommand) (AssignmentResponse, error)

    // DeleteAssignment removes an assignment. Parent auth required.
    DeleteAssignment(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID) error

    // === QUERY SIDE (reads, no side effects) ================================

    // --- Definition Queries (Layer 1 — no FamilyScope) ----------------------

    // ListActivityDefs lists activity definitions with filtering.
    ListActivityDefs(ctx context.Context, query ActivityDefQuery) (PaginatedResponse[ActivityDefSummaryResponse], error)

    // GetActivityDef returns a single activity definition.
    GetActivityDef(ctx context.Context, defID uuid.UUID) (ActivityDefResponse, error)

    // ListReadingItems lists reading items with filtering.
    ListReadingItems(ctx context.Context, query ReadingItemQuery) (PaginatedResponse[ReadingItemSummaryResponse], error)

    // GetReadingItem returns a single reading item with linked artifacts.
    GetReadingItem(ctx context.Context, itemID uuid.UUID) (ReadingItemDetailResponse, error)

    // GetLinkedArtifacts gets all artifacts linked to a content item. [§9]
    GetLinkedArtifacts(ctx context.Context, contentType string, contentID uuid.UUID, direction LinkDirection) ([]ArtifactLinkResponse, error)

    // --- Instance Queries (Layer 3 — FamilyScope required) ------------------

    // ListActivityLogs lists activity logs for a student with filtering. [S§8.1.1]
    ListActivityLogs(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query ActivityLogQuery) (PaginatedResponse[ActivityLogResponse], error)

    // GetActivityLog returns a single activity log entry.
    GetActivityLog(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, logID uuid.UUID) (ActivityLogResponse, error)

    // ListJournalEntries lists journal entries for a student. [S§8.1.4]
    ListJournalEntries(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query JournalEntryQuery) (PaginatedResponse[JournalEntryResponse], error)

    // GetJournalEntry returns a single journal entry.
    GetJournalEntry(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, entryID uuid.UUID) (JournalEntryResponse, error)

    // ListReadingProgress lists reading progress for a student. [S§8.1.3]
    ListReadingProgress(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query ReadingProgressQuery) (PaginatedResponse[ReadingProgressResponse], error)

    // ListReadingLists lists the family's reading lists. [S§8.1.3]
    ListReadingLists(ctx context.Context, scope *FamilyScope) ([]ReadingListSummaryResponse, error)

    // GetReadingList returns a reading list with items and student progress.
    GetReadingList(ctx context.Context, scope *FamilyScope, listID uuid.UUID) (ReadingListDetailResponse, error)

    // GetProgressSummary returns progress summary for a student. [S§8.1.7]
    GetProgressSummary(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query ProgressQuery) (ProgressSummaryResponse, error)

    // GetSubjectBreakdown returns per-subject breakdown for a student.
    GetSubjectBreakdown(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query ProgressQuery) ([]SubjectProgressResponse, error)

    // GetActivityTimeline returns activity timeline for a student.
    GetActivityTimeline(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query TimelineQuery) (PaginatedResponse[TimelineEntryResponse], error)

    // GetResolvedTools returns the family's resolved tool set. Delegates to method::. [S§4.2]
    GetResolvedTools(ctx context.Context, scope *FamilyScope) ([]ActiveToolResponse, error)

    // GetStudentTools returns a student-specific resolved tool set. [S§4.6, S§8.2]
    GetStudentTools(ctx context.Context, scope *FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error)

    // GetSubjectTaxonomy returns subject taxonomy tree. [S§8.3]
    GetSubjectTaxonomy(ctx context.Context, scope *FamilyScope, query TaxonomyQuery) ([]SubjectTaxonomyResponse, error)

    // GetExportRequest returns export request status.
    GetExportRequest(ctx context.Context, scope *FamilyScope, exportID uuid.UUID) (ExportRequestResponse, error)

    // --- Assessment Engine Queries ---------------------- [S§8.1.9]

    // ListQuestions lists questions with filtering (for quiz building).
    ListQuestions(ctx context.Context, query QuestionQuery) (PaginatedResponse[QuestionSummaryResponse], error)

    // GetQuizDef returns a quiz definition with questions.
    GetQuizDef(ctx context.Context, quizDefID uuid.UUID, includeAnswers bool) (QuizDefDetailResponse, error)

    // GetQuizSession returns a quiz session (for resume or review).
    GetQuizSession(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, sessionID uuid.UUID) (QuizSessionResponse, error)

    // --- Sequence Engine Queries ------------------------ [S§8.1.12]

    // GetSequenceDef returns a sequence definition with items.
    GetSequenceDef(ctx context.Context, sequenceDefID uuid.UUID) (SequenceDefDetailResponse, error)

    // GetSequenceProgress returns sequence progress with per-item completion status.
    GetSequenceProgress(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, progressID uuid.UUID) (SequenceProgressResponse, error)

    // --- Assignment Queries ----------------------------- [S§8.6.3]

    // ListAssignments lists assignments for a student.
    ListAssignments(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query AssignmentQuery) (PaginatedResponse[AssignmentResponse], error)
}
```

**Implementation**: `LearningServiceImpl` in `internal/learn/service.go`. Constructor receives:
- `ActivityDefRepository` (interface)
- `ActivityLogRepository` (interface)
- `ReadingItemRepository` (interface)
- `ReadingProgressRepository` (interface)
- `ReadingListRepository` (interface)
- `JournalEntryRepository` (interface)
- `ArtifactLinkRepository` (interface)
- `ProgressRepository` (interface)
- `SubjectTaxonomyRepository` (interface)
- `ExportRepository` (interface)
- `QuestionRepository` (interface, assessment engine)
- `QuizDefRepository` (interface, assessment engine)
- `QuizSessionRepository` (interface, assessment engine)
- `SequenceDefRepository` (interface, sequence engine)
- `SequenceProgressRepository` (interface, sequence engine)
- `AssignmentRepository` (interface, student assignments)
- `MethodologyService` (interface, for tool resolution)
- `IamService` (interface, for student/family data lookup)
- `EventBus`

---

## §6 Repository Interfaces

Defined in `internal/learn/ports.go`. Repositories are organized by data layer. Layer 3
repositories require `FamilyScope` for all operations. Layer 1 repositories use
publisher-based access control at the application level. `[CODING §2.4, CODING §8.2]`

```go
// internal/learn/ports.go (continued)

// === Layer 1: Definition Repositories (no FamilyScope) =======================

// ActivityDefRepository manages activity definition persistence.
type ActivityDefRepository interface {
    // Create creates an activity definition.
    Create(ctx context.Context, def *CreateActivityDefRecord) (*ActivityDef, error)

    // FindByID finds an activity definition by ID.
    FindByID(ctx context.Context, defID uuid.UUID) (*ActivityDef, error)

    // List lists activity definitions with filtering and pagination.
    List(ctx context.Context, query *ActivityDefQuery) ([]ActivityDef, error)

    // Update updates an activity definition.
    Update(ctx context.Context, defID uuid.UUID, update *UpdateActivityDefRecord) (*ActivityDef, error)

    // SoftDelete soft-deletes an activity definition (sets is_active = false).
    SoftDelete(ctx context.Context, defID uuid.UUID) error
}

// ReadingItemRepository manages reading item persistence.
type ReadingItemRepository interface {
    // Create creates a reading item definition.
    Create(ctx context.Context, item *CreateReadingItemRecord) (*ReadingItem, error)

    // FindByID finds a reading item by ID.
    FindByID(ctx context.Context, itemID uuid.UUID) (*ReadingItem, error)

    // List lists reading items with filtering and pagination.
    List(ctx context.Context, query *ReadingItemQuery) ([]ReadingItem, error)

    // Update updates a reading item.
    Update(ctx context.Context, itemID uuid.UUID, update *UpdateReadingItemRecord) (*ReadingItem, error)

    // FindByIDs finds reading items by IDs (batch load for reading list hydration).
    FindByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]ReadingItem, error)
}

// ArtifactLinkRepository manages artifact link persistence.
type ArtifactLinkRepository interface {
    // Create creates an artifact link.
    Create(ctx context.Context, link *CreateArtifactLinkRecord) (*ArtifactLink, error)

    // FindByContent finds links by source or target.
    FindByContent(ctx context.Context, contentType string, contentID uuid.UUID, direction LinkDirection) ([]ArtifactLink, error)

    // FindByID finds a link by ID.
    FindByID(ctx context.Context, linkID uuid.UUID) (*ArtifactLink, error)

    // Delete deletes a link.
    Delete(ctx context.Context, linkID uuid.UUID) error
}

// === Layer 3: Instance Repositories (FamilyScope required) ===================

// ActivityLogRepository manages activity log persistence.
type ActivityLogRepository interface {
    // Create creates an activity log entry.
    Create(ctx context.Context, scope *FamilyScope, log *CreateActivityLogRecord) (*ActivityLog, error)

    // FindByID finds an activity log by ID.
    FindByID(ctx context.Context, scope *FamilyScope, logID uuid.UUID) (*ActivityLog, error)

    // ListByStudent lists activity logs for a student with filtering.
    ListByStudent(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query *ActivityLogQuery) ([]ActivityLog, error)

    // Update updates an activity log.
    Update(ctx context.Context, scope *FamilyScope, logID uuid.UUID, update *UpdateActivityLogRecord) (*ActivityLog, error)

    // Delete deletes an activity log.
    Delete(ctx context.Context, scope *FamilyScope, logID uuid.UUID) error

    // CountByStudentDateRange counts activities in a date range for a student (progress queries).
    CountByStudentDateRange(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) (int64, error)

    // HoursBySubject aggregates hours by subject for a student in a date range.
    HoursBySubject(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) ([]SubjectHours, error)
}

// JournalEntryRepository manages journal entry persistence.
type JournalEntryRepository interface {
    // Create creates a journal entry.
    Create(ctx context.Context, scope *FamilyScope, entry *CreateJournalEntryRecord) (*JournalEntry, error)

    // FindByID finds a journal entry by ID.
    FindByID(ctx context.Context, scope *FamilyScope, entryID uuid.UUID) (*JournalEntry, error)

    // ListByStudent lists journal entries for a student with filtering.
    ListByStudent(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query *JournalEntryQuery) ([]JournalEntry, error)

    // Update updates a journal entry.
    Update(ctx context.Context, scope *FamilyScope, entryID uuid.UUID, update *UpdateJournalEntryRecord) (*JournalEntry, error)

    // Delete deletes a journal entry.
    Delete(ctx context.Context, scope *FamilyScope, entryID uuid.UUID) error
}

// ReadingProgressRepository manages reading progress persistence.
type ReadingProgressRepository interface {
    // Create creates a reading progress record.
    Create(ctx context.Context, scope *FamilyScope, progress *CreateReadingProgressRecord) (*ReadingProgress, error)

    // FindByID finds reading progress by ID.
    FindByID(ctx context.Context, scope *FamilyScope, progressID uuid.UUID) (*ReadingProgress, error)

    // ListByStudent lists reading progress for a student.
    ListByStudent(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, query *ReadingProgressQuery) ([]ReadingProgress, error)

    // Update updates reading progress.
    Update(ctx context.Context, scope *FamilyScope, progressID uuid.UUID, update *UpdateReadingProgressRecord) (*ReadingProgress, error)

    // Exists checks if a student is already tracking a reading item.
    Exists(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, readingItemID uuid.UUID) (bool, error)

    // CountCompleted counts completed books for a student in a date range.
    CountCompleted(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) (int64, error)
}

// ReadingListRepository manages reading list persistence.
type ReadingListRepository interface {
    // Create creates a reading list.
    Create(ctx context.Context, scope *FamilyScope, list *CreateReadingListRecord) (*ReadingList, error)

    // FindByID finds a reading list by ID.
    FindByID(ctx context.Context, scope *FamilyScope, listID uuid.UUID) (*ReadingList, error)

    // ListByFamily lists reading lists for a family.
    ListByFamily(ctx context.Context, scope *FamilyScope) ([]ReadingList, error)

    // Update updates a reading list.
    Update(ctx context.Context, scope *FamilyScope, listID uuid.UUID, update *UpdateReadingListRecord) (*ReadingList, error)

    // Delete deletes a reading list.
    Delete(ctx context.Context, scope *FamilyScope, listID uuid.UUID) error

    // AddItems adds items to a reading list.
    AddItems(ctx context.Context, scope *FamilyScope, listID uuid.UUID, itemIDs []uuid.UUID) error

    // RemoveItems removes items from a reading list.
    RemoveItems(ctx context.Context, scope *FamilyScope, listID uuid.UUID, itemIDs []uuid.UUID) error

    // ListItems lists items in a reading list with sort order.
    ListItems(ctx context.Context, listID uuid.UUID) ([]ReadingListItem, error)
}

// ProgressRepository manages progress snapshot persistence.
type ProgressRepository interface {
    // CreateSnapshot stores a progress snapshot.
    CreateSnapshot(ctx context.Context, scope *FamilyScope, snapshot *CreateProgressSnapshotRecord) (*ProgressSnapshot, error)

    // GetLatestSnapshot gets the latest progress snapshot for a student.
    GetLatestSnapshot(ctx context.Context, scope *FamilyScope, studentID uuid.UUID) (*ProgressSnapshot, error)

    // ListSnapshots gets snapshots in a date range for trend analysis.
    ListSnapshots(ctx context.Context, scope *FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) ([]ProgressSnapshot, error)
}

// ExportRepository manages export request persistence.
type ExportRepository interface {
    // Create creates an export request.
    Create(ctx context.Context, scope *FamilyScope, request *CreateExportRequestRecord) (*ExportRequest, error)

    // FindByID finds an export request by ID.
    FindByID(ctx context.Context, scope *FamilyScope, exportID uuid.UUID) (*ExportRequest, error)

    // HasActiveExport checks if there is an active (pending/processing) export for a family.
    HasActiveExport(ctx context.Context, scope *FamilyScope) (bool, error)

    // UpdateStatus updates export status and file URL.
    UpdateStatus(ctx context.Context, exportID uuid.UUID, status string, fileURL *string, expiresAt *time.Time, errorMessage *string) (*ExportRequest, error)
}

// === Platform Repositories (no FamilyScope) ==================================

// SubjectTaxonomyRepository manages subject taxonomy persistence.
type SubjectTaxonomyRepository interface {
    // List lists taxonomy nodes with optional filtering by level and parent.
    List(ctx context.Context, query *TaxonomyQuery) ([]SubjectTaxonomy, error)

    // FindBySlug finds a taxonomy node by slug.
    FindBySlug(ctx context.Context, slug string) (*SubjectTaxonomy, error)

    // ValidateSlugs validates that all slugs exist in the taxonomy.
    ValidateSlugs(ctx context.Context, slugs []string) (bool, error)

    // ListCustomSubjects lists family-scoped custom subjects.
    ListCustomSubjects(ctx context.Context, scope *FamilyScope) ([]CustomSubject, error)

    // CreateCustomSubject creates a family-scoped custom subject.
    CreateCustomSubject(ctx context.Context, scope *FamilyScope, subject *CreateCustomSubjectRecord) (*CustomSubject, error)
}
```

---

## §7 Adapter Interfaces

The learning domain has one adapter interface for media operations. No external third-party
service dependencies. `[CODING §8.1]`

```go
// internal/learn/ports.go (continued)

// MediaAdapter handles file upload/download operations.
// Delegates to media:: domain for actual storage and validation.
type MediaAdapter interface {
    // ValidateAttachment validates an attachment (magic bytes, size limit, MIME type).
    ValidateAttachment(ctx context.Context, attachment *AttachmentInput) error

    // GetUploadURL generates a pre-signed upload URL for direct client upload.
    GetUploadURL(ctx context.Context, contentType string, filename string) (*UploadURLResponse, error)
}
```

---

## §8 Models (DTOs)

All types defined in `internal/learn/models.go`. API-facing types use struct tags:
`json:"field"` for serialization, `validate:"required"` for validation (go-playground/validator),
and swaggo/swag comment annotations for OpenAPI generation. `[CODING §2.3]`

### §8.1 Request Types

```go
// internal/learn/models.go

// --- Layer 1: Definition Commands -------------------------------------------

// CreateActivityDefCommand creates an activity definition. [S§8.1.1]
// @Description Create activity definition request
type CreateActivityDefCommand struct {
    PublisherID       uuid.UUID         `json:"publisher_id" validate:"required"`
    Title             string            `json:"title" validate:"required,min=1,max=500"`
    Description       *string           `json:"description,omitempty" validate:"omitempty,max=5000"`
    SubjectTags       []string          `json:"subject_tags"`
    MethodologyID     *uuid.UUID        `json:"methodology_id,omitempty"`
    ToolID            *uuid.UUID        `json:"tool_id,omitempty"`
    EstDurationMinutes *int16           `json:"est_duration_minutes,omitempty"`
    Attachments       []AttachmentInput `json:"attachments"`
}

// UpdateActivityDefCommand updates an activity definition.
// @Description Update activity definition request
type UpdateActivityDefCommand struct {
    Title              *string           `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
    Description        *string           `json:"description,omitempty" validate:"omitempty,max=5000"`
    SubjectTags        *[]string         `json:"subject_tags,omitempty"`
    MethodologyID      *uuid.UUID        `json:"methodology_id,omitempty"`
    ToolID             *uuid.UUID        `json:"tool_id,omitempty"`
    EstDurationMinutes *int16            `json:"est_duration_minutes,omitempty"`
    Attachments        *[]AttachmentInput `json:"attachments,omitempty"`
}

// CreateReadingItemCommand creates a reading item. [S§8.1.3]
// @Description Create reading item request
type CreateReadingItemCommand struct {
    PublisherID    uuid.UUID  `json:"publisher_id" validate:"required"`
    Title          string     `json:"title" validate:"required,min=1,max=500"`
    Author         *string    `json:"author,omitempty" validate:"omitempty,max=300"`
    ISBN           *string    `json:"isbn,omitempty" validate:"omitempty,max=20"`
    SubjectTags    []string   `json:"subject_tags"`
    Description    *string    `json:"description,omitempty" validate:"omitempty,max=2000"`
    CoverImageURL  *string    `json:"cover_image_url,omitempty"`
    PageCount      *int16     `json:"page_count,omitempty"`
}

// UpdateReadingItemCommand updates a reading item.
// @Description Update reading item request
type UpdateReadingItemCommand struct {
    Title          *string    `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
    Author         *string    `json:"author,omitempty" validate:"omitempty,max=300"`
    ISBN           *string    `json:"isbn,omitempty"`
    SubjectTags    *[]string  `json:"subject_tags,omitempty"`
    Description    *string    `json:"description,omitempty"`
    CoverImageURL  *string    `json:"cover_image_url,omitempty"`
    PageCount      *int16     `json:"page_count,omitempty"`
}

// CreateArtifactLinkCommand creates an artifact link. [§9]
// @Description Create artifact link request
type CreateArtifactLinkCommand struct {
    SourceType   string     `json:"source_type" validate:"required"`
    SourceID     uuid.UUID  `json:"source_id" validate:"required"`
    TargetType   string     `json:"target_type" validate:"required"`
    TargetID     uuid.UUID  `json:"target_id" validate:"required"`
    Relationship *string    `json:"relationship,omitempty"` // defaults to "about"
}

// --- Layer 3: Instance Commands ---------------------------------------------

// LogActivityCommand logs an activity. [S§8.1.1]
// @Description Log activity request
type LogActivityCommand struct {
    Title           string            `json:"title" validate:"required,min=1,max=500"`
    Description     *string           `json:"description,omitempty" validate:"omitempty,max=5000"`
    SubjectTags     []string          `json:"subject_tags"`
    ContentID       *uuid.UUID        `json:"content_id,omitempty"`       // optional ref to activity_def
    MethodologyID   *uuid.UUID        `json:"methodology_id,omitempty"`
    ToolID          *uuid.UUID        `json:"tool_id,omitempty"`
    DurationMinutes *int16            `json:"duration_minutes,omitempty"`
    Attachments     []AttachmentInput `json:"attachments"`
    ActivityDate    *time.Time        `json:"activity_date,omitempty"`    // defaults to today
}

// UpdateActivityLogCommand updates an activity log.
// @Description Update activity log request
type UpdateActivityLogCommand struct {
    Title           *string            `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
    Description     *string            `json:"description,omitempty" validate:"omitempty,max=5000"`
    SubjectTags     *[]string          `json:"subject_tags,omitempty"`
    DurationMinutes *int16             `json:"duration_minutes,omitempty"`
    Attachments     *[]AttachmentInput `json:"attachments,omitempty"`
    ActivityDate    *time.Time         `json:"activity_date,omitempty"`
}

// CreateJournalEntryCommand creates a journal entry. [S§8.1.4]
// @Description Create journal entry request
type CreateJournalEntryCommand struct {
    EntryType   string            `json:"entry_type" validate:"required"` // "freeform" | "narration" | "reflection"
    Title       *string           `json:"title,omitempty" validate:"omitempty,max=500"`
    Content     string            `json:"content" validate:"required,min=1,max=50000"`
    SubjectTags []string          `json:"subject_tags"`
    ContentID   *uuid.UUID        `json:"content_id,omitempty"`
    Attachments []AttachmentInput `json:"attachments"`
    EntryDate   *time.Time        `json:"entry_date,omitempty"`
}

// UpdateJournalEntryCommand updates a journal entry.
// @Description Update journal entry request
type UpdateJournalEntryCommand struct {
    EntryType   *string            `json:"entry_type,omitempty"`
    Title       *string            `json:"title,omitempty" validate:"omitempty,max=500"`
    Content     *string            `json:"content,omitempty" validate:"omitempty,min=1,max=50000"`
    SubjectTags *[]string          `json:"subject_tags,omitempty"`
    Attachments *[]AttachmentInput `json:"attachments,omitempty"`
    EntryDate   *time.Time         `json:"entry_date,omitempty"`
}

// StartReadingCommand starts reading a book. [S§8.1.3]
// @Description Start reading request
type StartReadingCommand struct {
    ReadingItemID uuid.UUID  `json:"reading_item_id" validate:"required"`
    ReadingListID *uuid.UUID `json:"reading_list_id,omitempty"`
}

// UpdateReadingProgressCommand updates reading progress.
// @Description Update reading progress request
type UpdateReadingProgressCommand struct {
    Status *string `json:"status,omitempty"`          // "to_read" | "in_progress" | "completed"
    Notes  *string `json:"notes,omitempty" validate:"omitempty,max=2000"`
}

// CreateReadingListCommand creates a reading list. [S§8.1.3]
// @Description Create reading list request
type CreateReadingListCommand struct {
    Name           string      `json:"name" validate:"required,min=1,max=200"`
    Description    *string     `json:"description,omitempty" validate:"omitempty,max=2000"`
    StudentID      *uuid.UUID  `json:"student_id,omitempty"`
    ReadingItemIDs []uuid.UUID `json:"reading_item_ids"`
}

// UpdateReadingListCommand updates a reading list.
// @Description Update reading list request
type UpdateReadingListCommand struct {
    Name          *string     `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
    Description   *string     `json:"description,omitempty" validate:"omitempty,max=2000"`
    AddItemIDs    *[]uuid.UUID `json:"add_item_ids,omitempty"`
    RemoveItemIDs *[]uuid.UUID `json:"remove_item_ids,omitempty"`
}

// CreateCustomSubjectCommand creates a custom subject. [S§8.3]
// @Description Create custom subject request
type CreateCustomSubjectCommand struct {
    Name             string     `json:"name" validate:"required,min=1,max=200"`
    ParentTaxonomyID *uuid.UUID `json:"parent_taxonomy_id,omitempty"`
}

// RequestExportCommand requests a data export. [S§8.5]
// @Description Request data export
type RequestExportCommand struct {
    Format *string `json:"format,omitempty"` // "json" | "csv", defaults to "json"
}

// AttachmentInput is a shared attachment input type.
// @Description Attachment input
type AttachmentInput struct {
    URL         string  `json:"url" validate:"required"`
    ContentType string  `json:"content_type" validate:"required"` // MIME type
    Filename    *string `json:"filename,omitempty"`
}
```

### §8.2 Response Types

```go
// --- Layer 1: Definition Responses ------------------------------------------

// ActivityDefResponse is the activity definition response. [S§8.1.1]
// @Description Activity definition
type ActivityDefResponse struct {
    ID                 uuid.UUID         `json:"id"`
    PublisherID        uuid.UUID         `json:"publisher_id"`
    Title              string            `json:"title"`
    Description        *string           `json:"description,omitempty"`
    SubjectTags        []string          `json:"subject_tags"`
    MethodologyID      *uuid.UUID        `json:"methodology_id,omitempty"`
    ToolID             *uuid.UUID        `json:"tool_id,omitempty"`
    EstDurationMinutes *int16            `json:"est_duration_minutes,omitempty"`
    Attachments        []AttachmentInput `json:"attachments"`
    CreatedAt          time.Time         `json:"created_at"`
    UpdatedAt          time.Time         `json:"updated_at"`
}

// ActivityDefSummaryResponse is the activity definition summary (for list views).
// @Description Activity definition summary
type ActivityDefSummaryResponse struct {
    ID                 uuid.UUID  `json:"id"`
    Title              string     `json:"title"`
    SubjectTags        []string   `json:"subject_tags"`
    MethodologyID      *uuid.UUID `json:"methodology_id,omitempty"`
    EstDurationMinutes *int16     `json:"est_duration_minutes,omitempty"`
}

// ReadingItemResponse is the reading item response. [S§8.1.3]
// @Description Reading item
type ReadingItemResponse struct {
    ID            uuid.UUID `json:"id"`
    PublisherID   uuid.UUID `json:"publisher_id"`
    Title         string    `json:"title"`
    Author        *string   `json:"author,omitempty"`
    ISBN          *string   `json:"isbn,omitempty"`
    SubjectTags   []string  `json:"subject_tags"`
    Description   *string   `json:"description,omitempty"`
    CoverImageURL *string   `json:"cover_image_url,omitempty"`
    PageCount     *int16    `json:"page_count,omitempty"`
    CreatedAt     time.Time `json:"created_at"`
}

// ReadingItemSummaryResponse is the reading item summary (for list views).
// @Description Reading item summary
type ReadingItemSummaryResponse struct {
    ID            uuid.UUID `json:"id"`
    Title         string    `json:"title"`
    Author        *string   `json:"author,omitempty"`
    SubjectTags   []string  `json:"subject_tags"`
    CoverImageURL *string   `json:"cover_image_url,omitempty"`
}

// ReadingItemDetailResponse is the reading item detail with linked artifacts.
// @Description Reading item detail
type ReadingItemDetailResponse struct {
    ReadingItemResponse
    LinkedArtifacts []ArtifactLinkResponse `json:"linked_artifacts"`
}

// ArtifactLinkResponse is the artifact link response. [§9]
// @Description Artifact link
type ArtifactLinkResponse struct {
    ID           uuid.UUID `json:"id"`
    SourceType   string    `json:"source_type"`
    SourceID     uuid.UUID `json:"source_id"`
    TargetType   string    `json:"target_type"`
    TargetID     uuid.UUID `json:"target_id"`
    Relationship string    `json:"relationship"`
    CreatedAt    time.Time `json:"created_at"`
}

// --- Layer 3: Instance Responses --------------------------------------------

// ActivityLogResponse is the activity log response. [S§8.1.1]
// @Description Activity log entry
type ActivityLogResponse struct {
    ID              uuid.UUID         `json:"id"`
    StudentID       uuid.UUID         `json:"student_id"`
    Title           string            `json:"title"`
    Description     *string           `json:"description,omitempty"`
    SubjectTags     []string          `json:"subject_tags"`
    ContentID       *uuid.UUID        `json:"content_id,omitempty"`
    ContentTitle    *string           `json:"content_title,omitempty"` // resolved from activity_def if linked
    MethodologyID   *uuid.UUID        `json:"methodology_id,omitempty"`
    ToolID          *uuid.UUID        `json:"tool_id,omitempty"`
    DurationMinutes *int16            `json:"duration_minutes,omitempty"`
    Attachments     []AttachmentInput `json:"attachments"`
    ActivityDate    time.Time         `json:"activity_date"`
    CreatedAt       time.Time         `json:"created_at"`
}

// JournalEntryResponse is the journal entry response. [S§8.1.4]
// @Description Journal entry
type JournalEntryResponse struct {
    ID          uuid.UUID         `json:"id"`
    StudentID   uuid.UUID         `json:"student_id"`
    EntryType   string            `json:"entry_type"`
    Title       *string           `json:"title,omitempty"`
    Content     string            `json:"content"`
    SubjectTags []string          `json:"subject_tags"`
    Attachments []AttachmentInput `json:"attachments"`
    EntryDate   time.Time         `json:"entry_date"`
    CreatedAt   time.Time         `json:"created_at"`
}

// ReadingProgressResponse is the reading progress response. [S§8.1.3]
// @Description Reading progress
type ReadingProgressResponse struct {
    ID            uuid.UUID                  `json:"id"`
    StudentID     uuid.UUID                  `json:"student_id"`
    ReadingItem   ReadingItemSummaryResponse `json:"reading_item"`
    ReadingListID *uuid.UUID                 `json:"reading_list_id,omitempty"`
    Status        string                     `json:"status"`
    StartedAt     *time.Time                 `json:"started_at,omitempty"`
    CompletedAt   *time.Time                 `json:"completed_at,omitempty"`
    Notes         *string                    `json:"notes,omitempty"`
}

// ReadingListSummaryResponse is the reading list summary.
// @Description Reading list summary
type ReadingListSummaryResponse struct {
    ID             uuid.UUID  `json:"id"`
    Name           string     `json:"name"`
    Description    *string    `json:"description,omitempty"`
    StudentID      *uuid.UUID `json:"student_id,omitempty"`
    ItemCount      int64      `json:"item_count"`
    CompletedCount int64      `json:"completed_count"`
}

// ReadingListResponse is the reading list response.
// @Description Reading list
type ReadingListResponse struct {
    ID          uuid.UUID  `json:"id"`
    Name        string     `json:"name"`
    Description *string    `json:"description,omitempty"`
    StudentID   *uuid.UUID `json:"student_id,omitempty"`
    CreatedAt   time.Time  `json:"created_at"`
}

// ReadingListDetailResponse is the reading list detail with items and progress.
// @Description Reading list detail
type ReadingListDetailResponse struct {
    ID          uuid.UUID                    `json:"id"`
    Name        string                       `json:"name"`
    Description *string                      `json:"description,omitempty"`
    StudentID   *uuid.UUID                   `json:"student_id,omitempty"`
    Items       []ReadingListItemWithProgress `json:"items"`
    CreatedAt   time.Time                    `json:"created_at"`
}

// ReadingListItemWithProgress is a reading list item with optional student progress.
// @Description Reading list item with progress
type ReadingListItemWithProgress struct {
    ReadingItem ReadingItemSummaryResponse `json:"reading_item"`
    SortOrder   int16                      `json:"sort_order"`
    Progress    *ReadingProgressResponse   `json:"progress,omitempty"`
}

// --- Cross-Cutting Responses ------------------------------------------------

// ProgressSummaryResponse is the progress summary. [S§8.1.7]
// @Description Progress summary
type ProgressSummaryResponse struct {
    StudentID       uuid.UUID              `json:"student_id"`
    DateFrom        time.Time              `json:"date_from"`
    DateTo          time.Time              `json:"date_to"`
    TotalActivities int64                  `json:"total_activities"`
    TotalHours      float64                `json:"total_hours"`
    HoursBySubject  []SubjectHoursResponse `json:"hours_by_subject"`
    BooksCompleted  int64                  `json:"books_completed"`
    JournalEntries  int64                  `json:"journal_entries"`
}

// SubjectHoursResponse is hours per subject.
// @Description Subject hours
type SubjectHoursResponse struct {
    SubjectSlug string  `json:"subject_slug"`
    SubjectName string  `json:"subject_name"`
    Hours       float64 `json:"hours"`
}

// SubjectProgressResponse is per-subject progress breakdown.
// @Description Subject progress
type SubjectProgressResponse struct {
    SubjectSlug    string  `json:"subject_slug"`
    SubjectName    string  `json:"subject_name"`
    ActivityCount  int64   `json:"activity_count"`
    TotalHours     float64 `json:"total_hours"`
    JournalCount   int64   `json:"journal_count"`
    BooksCompleted int64   `json:"books_completed"`
}

// TimelineEntryResponse is an activity timeline entry (union of activity logs and journal entries).
// @Description Timeline entry
type TimelineEntryResponse struct {
    ID          uuid.UUID `json:"id"`
    EntryType   string    `json:"entry_type"` // "activity" | "journal" | "reading_completed"
    Title       string    `json:"title"`
    Description *string   `json:"description,omitempty"`
    SubjectTags []string  `json:"subject_tags"`
    Date        time.Time `json:"date"`
    CreatedAt   time.Time `json:"created_at"`
}

// SubjectTaxonomyResponse is a subject taxonomy node. [S§8.3]
// @Description Subject taxonomy node
type SubjectTaxonomyResponse struct {
    ID       uuid.UUID                 `json:"id"`
    ParentID *uuid.UUID                `json:"parent_id,omitempty"`
    Name     string                    `json:"name"`
    Slug     string                    `json:"slug"`
    Level    int16                     `json:"level"`
    Children []SubjectTaxonomyResponse `json:"children"`
    IsCustom bool                      `json:"is_custom"` // true for family-scoped custom subjects
}

// CustomSubjectResponse is the custom subject response.
// @Description Custom subject
type CustomSubjectResponse struct {
    ID               uuid.UUID  `json:"id"`
    Name             string     `json:"name"`
    Slug             string     `json:"slug"`
    ParentTaxonomyID *uuid.UUID `json:"parent_taxonomy_id,omitempty"`
}

// ExportRequestResponse is the export request response. [S§8.5]
// @Description Export request
type ExportRequestResponse struct {
    ID        uuid.UUID  `json:"id"`
    Status    string     `json:"status"`
    FileURL   *string    `json:"file_url,omitempty"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
    CreatedAt time.Time  `json:"created_at"`
}

// ActiveToolResponse is the active tool response (from method:: delegation). [S§4.2]
// Re-exported from method:: types for convenience.
// @Description Active tool
type ActiveToolResponse struct {
    ToolID          uuid.UUID              `json:"tool_id"`
    Slug            string                 `json:"slug"`
    DisplayName     string                 `json:"display_name"`
    Label           *string                `json:"label,omitempty"`           // methodology-specific override
    Description     *string                `json:"description,omitempty"`
    Tier            string                 `json:"tier"`                      // "free" | "premium"
    Guidance        *string                `json:"guidance,omitempty"`        // methodology-specific guidance [S§8.4]
    ConfigOverrides map[string]interface{} `json:"config_overrides"`
    SortOrder       int16                  `json:"sort_order"`
}

// UploadURLResponse is the upload URL response (from media:: adapter).
// @Description Upload URL
type UploadURLResponse struct {
    UploadURL string `json:"upload_url"`
    FileKey   string `json:"file_key"`
}
```

### §8.3 Internal Types

```go
// ActivityDef is the internal activity definition record.
type ActivityDef struct {
    ID                 uuid.UUID
    PublisherID        uuid.UUID
    Title              string
    Description        *string
    SubjectTags        []string
    MethodologyID      *uuid.UUID
    ToolID             *uuid.UUID
    EstDurationMinutes *int16
    Attachments        json.RawMessage
    IsActive           bool
    CreatedAt          time.Time
    UpdatedAt          time.Time
}

// ReadingItem is the internal reading item record.
type ReadingItem struct {
    ID            uuid.UUID
    PublisherID   uuid.UUID
    Title         string
    Author        *string
    ISBN          *string
    SubjectTags   []string
    Description   *string
    CoverImageURL *string
    PageCount     *int16
    IsActive      bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// ArtifactLink is the internal artifact link record.
type ArtifactLink struct {
    ID           uuid.UUID
    SourceType   string
    SourceID     uuid.UUID
    TargetType   string
    TargetID     uuid.UUID
    Relationship string
    CreatedAt    time.Time
}

// ActivityLog is the internal activity log record.
type ActivityLog struct {
    ID              uuid.UUID
    FamilyID        uuid.UUID
    StudentID       uuid.UUID
    Title           string
    Description     *string
    SubjectTags     []string
    ContentID       *uuid.UUID
    MethodologyID   *uuid.UUID
    ToolID          *uuid.UUID
    DurationMinutes *int16
    Attachments     json.RawMessage
    ActivityDate    time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// JournalEntry is the internal journal entry record.
type JournalEntry struct {
    ID          uuid.UUID
    FamilyID    uuid.UUID
    StudentID   uuid.UUID
    EntryType   string
    Title       *string
    Content     string
    SubjectTags []string
    ContentID   *uuid.UUID
    Attachments json.RawMessage
    EntryDate   time.Time
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// ReadingProgress is the internal reading progress record.
type ReadingProgress struct {
    ID            uuid.UUID
    FamilyID      uuid.UUID
    StudentID     uuid.UUID
    ReadingItemID uuid.UUID
    ReadingListID *uuid.UUID
    Status        string
    StartedAt     *time.Time
    CompletedAt   *time.Time
    Notes         *string
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

// ReadingList is the internal reading list record.
type ReadingList struct {
    ID          uuid.UUID
    FamilyID    uuid.UUID
    Name        string
    Description *string
    StudentID   *uuid.UUID
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// ReadingListItem is the internal reading list item.
type ReadingListItem struct {
    ReadingListID uuid.UUID
    ReadingItemID uuid.UUID
    SortOrder     int16
    AddedAt       time.Time
}

// ProgressSnapshot is the internal progress snapshot record.
type ProgressSnapshot struct {
    ID           uuid.UUID
    FamilyID     uuid.UUID
    StudentID    uuid.UUID
    SnapshotDate time.Time
    Data         json.RawMessage
    CreatedAt    time.Time
}

// ExportRequest is the internal export request record.
type ExportRequest struct {
    ID           uuid.UUID
    FamilyID     uuid.UUID
    RequestedBy  uuid.UUID
    Status       string
    FileURL      *string
    ExpiresAt    *time.Time
    ErrorMessage *string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// SubjectTaxonomy is the internal subject taxonomy node.
type SubjectTaxonomy struct {
    ID           uuid.UUID
    ParentID     *uuid.UUID
    Name         string
    Slug         string
    Level        int16
    DisplayOrder int16
    IsActive     bool
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

// CustomSubject is the internal custom subject record.
type CustomSubject struct {
    ID               uuid.UUID
    FamilyID         uuid.UUID
    ParentTaxonomyID *uuid.UUID
    Name             string
    Slug             string
    CreatedAt        time.Time
}

// SubjectHours is a subject hours aggregation (from repository query).
type SubjectHours struct {
    SubjectSlug  string
    TotalMinutes int64
}

// LinkDirection is a link direction filter for artifact queries.
type LinkDirection int

const (
    LinkDirectionSource LinkDirection = iota
    LinkDirectionTarget
    LinkDirectionBoth
)

// ActivityDefQuery contains query parameters for activity definitions.
type ActivityDefQuery struct {
    Subject       *string
    MethodologyID *uuid.UUID
    PublisherID   *uuid.UUID
    Search        *string
    Cursor        *uuid.UUID
    Limit         int64
}

// ReadingItemQuery contains query parameters for reading items.
type ReadingItemQuery struct {
    Search *string
    Subject *string
    ISBN   *string
    Cursor *uuid.UUID
    Limit  int64
}

// ActivityLogQuery contains query parameters for activity logs.
type ActivityLogQuery struct {
    Subject  *string
    DateFrom *time.Time
    DateTo   *time.Time
    Cursor   *uuid.UUID
    Limit    int64
}

// JournalEntryQuery contains query parameters for journal entries.
type JournalEntryQuery struct {
    EntryType *string
    DateFrom  *time.Time
    DateTo    *time.Time
    Search    *string
    Cursor    *uuid.UUID
    Limit     int64
}

// ReadingProgressQuery contains query parameters for reading progress.
type ReadingProgressQuery struct {
    Status *string
    Cursor *uuid.UUID
    Limit  int64
}

// ProgressQuery contains query parameters for progress.
type ProgressQuery struct {
    DateFrom *time.Time
    DateTo   *time.Time
}

// TimelineQuery contains query parameters for timeline.
type TimelineQuery struct {
    DateFrom *time.Time
    DateTo   *time.Time
    Cursor   *uuid.UUID
    Limit    int64
}

// TaxonomyQuery contains query parameters for taxonomy.
type TaxonomyQuery struct {
    Level    *int16
    ParentID *uuid.UUID
}

// PurchaseMetadata is metadata from a marketplace purchase (received via event).
type PurchaseMetadata struct {
    ContentType string
    ContentIDs  []uuid.UUID
    PublisherID uuid.UUID
}
```

---

## §9 Artifact Linking (Domain Deep-Dive 1)

Published content can reference each other — a test is *about* a book, an activity is *part of*
a unit study project, etc. Links live at the **published content layer** (Layer 2), not the
family tracking layer. `[S§9.7]`

### §9.1 Link Model

Links are polymorphic — both source and target carry a type discriminator:

```sql
-- See §3.2 for full schema
-- source_type + source_id → target_type + target_id
-- UNIQUE constraint prevents duplicate links
-- Indexed on both (source_type, source_id) and (target_type, target_id)
```

### §9.2 Relationship Types

| Relationship | Meaning | Example |
|-------------|---------|---------|
| `about` | Source is about the target content | Assessment → Reading Item ("test on Charlotte's Web") |
| `part_of` | Source is a component of the target | Activity Def → Project Def ("lesson 3 of unit study") |
| `follow_up` | Source follows the target sequentially | Activity Def → Activity Def ("do this after that") |
| `inspired_by` | Loose thematic connection | Activity Def → Reading Item ("inspired by the book") |

### §9.3 Access Control

- Links can be created by any member of the Publisher that owns the **source** content
- Links are publicly readable (no RLS) — visibility is controlled by the content tables themselves
- When a content definition is deleted, an ON DELETE trigger cleans up referencing links
- When a curriculum package is purchased from the marketplace, its internal links come with it

### §9.4 Query Patterns

**"What is this test about?"** — Find all targets where source = assessment_def:
```sql
SELECT * FROM learn_artifact_links
WHERE source_type = 'assessment_def' AND source_id = $1;
```

**"Show everything about this book"** — Find all sources that target this reading_item:
```sql
SELECT * FROM learn_artifact_links
WHERE target_type = 'reading_item' AND target_id = $1;
```

**Bidirectional lookup** — Used by `get_reading_item` detail view to show all related content:
```sql
SELECT * FROM learn_artifact_links
WHERE (source_type = $1 AND source_id = $2)
   OR (target_type = $1 AND target_id = $2);
```

---

## §10 Tool Resolution Flow (Domain Deep-Dive 2)

How `learn::` resolves which tools a family/student can use. This is a **read-only delegation**
to `method::` — learn:: never owns the tool registry. `[S§4.2, S§8.2]`

### §10.1 Resolution Sequence

```
GET /v1/learning/tools
    │
    ▼
1. Handler extracts FamilyScope
    │
    ▼
2. LearningService.GetResolvedTools(scope)
    │
    ▼
3. Delegates to method::MethodologyService.ResolveFamilyTools(scope)
    │  ├── Loads family's primary + secondary methodology IDs from iam_families
    │  ├── Queries method_tool_activations for all methodologies
    │  ├── Constructs ToolResolver with primary methodology ID
    │  └── Returns []ActiveToolResponse
    │
    ▼
4. Returns tool set to handler
```

### §10.2 Per-Student Override Flow

```
GET /v1/learning/tools/:student_id
    │
    ▼
1. Handler extracts FamilyScope, validates student belongs to family
    │
    ▼
2. LearningService.GetStudentTools(scope, studentID)
    │
    ▼
3. Delegates to method::MethodologyService.ResolveStudentTools(scope, studentID)
    │  ├── Checks iam_students.methodology_override_id for student
    │  ├── If override exists → resolve for override methodology only
    │  └── If no override → fall through to family resolution
    │
    ▼
4. Returns student-specific tool set
```

### §10.3 Guidance Surface

Each `ActiveToolResponse` includes a `guidance` field from the methodology's `config_overrides`.
The frontend renders this as a collapsible help panel within the tool interface. `[S§8.4]`

Guidance content is sourced from `method_tool_activations.config_overrides.guidance` and
references the methodology's philosophy modules and mastery paths. The learn:: domain
surfaces it — it does not own or manage it.

### §10.4 Tool ID Routing

`learn::` uses `tool_id` (from `method_tools`) to route to the correct tool implementation.
Each core tool (activities, reading lists, journals, etc.) has a fixed tool slug, and
learn:: maps slugs to handler modules:

| Tool Slug | Module | Phase |
|-----------|--------|-------|
| `activities` | `internal/learn/handler.go` (activity log endpoints) | 1 |
| `reading-lists` | `internal/learn/handler.go` (reading list endpoints) | 1 |
| `journaling` | `internal/learn/handler.go` (journal entry endpoints) | 1 |
| `progress-tracking` | `internal/learn/handler.go` (progress endpoints) | 1 |
| `assessment-engine` | `internal/learn/handler.go` (quiz/question endpoints) | 1 |
| `content-viewer` | Frontend-only (no backend handler — uses `media::` signed URLs) | 1 |
| `video-player` | `internal/learn/handler.go` (video progress endpoints) | 1 |
| `lesson-sequences` | `internal/learn/handler.go` (sequence endpoints) | 1 |
| `tests-grades` | `internal/learn/handler.go` (assessment result endpoints) | 1* |
| `projects` | `internal/learn/handler.go` (project endpoints) | 2 |
| `video-lessons` | `internal/learn/handler.go` (marketplace video integration) | 1* |
| `nature-journals` | Phase 3 | 3 |
| `trivium-tracker` | Phase 3 | 3 |
| `rhythm-planner` | Phase 3 | 3 |
| `observation-logs` | Phase 3 | 3 |
| `habit-tracking` | Phase 3 | 3 |
| `interest-led-logs` | Phase 3 | 3 |
| `handwork-tracker` | Phase 3 | 3 |
| `practical-life` | Phase 3 | 3 |

*`tests-grades` retains existing score recording (Phase 1) but advanced analytics remain Phase 2. `video-lessons` is now Phase 1 with the video player infrastructure.

---

## §11 Activity Logging Invariants (Domain Deep-Dive 3)

The `Activity` aggregate root in `internal/learn/domain/activity.go` enforces these invariants
structurally. `[ARCH §4.5]`

### §11.1 Invariants

1. **Student must belong to family**: Service verifies `iam_students.family_id = scope.family_id`
   before any student-scoped operation.
2. **Subject tags must exist**: All tags in `subject_tags` must exist in either
   `learn_subject_taxonomy` or `learn_custom_subjects` for the family.
3. **Duration must be non-negative**: `duration_minutes >= 0` (enforced by service validation,
   not CHECK constraint — allows NULL for "untracked" duration).
4. **Activity date cannot be in the future**: `activity_date <= CURRENT_DATE`. Enforced in
   service layer to avoid timezone edge cases in database constraints.
5. **Attachments validated via media::**: Before persisting, each attachment is validated through
   `MediaAdapter.ValidateAttachment` (magic bytes, size limits). `[CODING §4.2]`
6. **Content ID reference must be valid**: If `content_id` is provided, it must reference an
   active `learn_activity_defs` row.
7. **Tool ID must be in family's active tool set**: If `tool_id` is provided, it must be in the
   resolved tool set for the student (or family if no student override).

### §11.2 Aggregate Root

```go
// internal/learn/domain/activity.go

// Activity is the aggregate root for activity logging — enforces invariants.
type Activity struct {
    studentID       uuid.UUID
    title           string
    subjectTags     []string
    activityDate    time.Time
    durationMinutes *int16
    // ... private fields
}

// NewActivity creates a validated Activity, enforcing domain invariants.
func NewActivity(
    studentID uuid.UUID,
    title string,
    subjectTags []string,
    activityDate time.Time,
    durationMinutes *int16,
) (*Activity, error) {
    if durationMinutes != nil && *durationMinutes < 0 {
        return nil, ErrNegativeDuration
    }
    if activityDate.After(time.Now().Truncate(24 * time.Hour)) {
        return nil, ErrFutureDateNotAllowed
    }
    return &Activity{
        studentID:       studentID,
        title:           title,
        subjectTags:     subjectTags,
        activityDate:    activityDate,
        durationMinutes: durationMinutes,
    }, nil
}
```

---

## §12 Progress Tracking Architecture (Domain Deep-Dive 4)

### §12.1 Free Tier (Basic Progress) `[S§15.1]`

Computed on-the-fly from raw data. No materialized views or background jobs needed.

| Metric | Source | Query |
|--------|--------|-------|
| Activity count | `learn_activity_logs` | `COUNT(*)` with date range filter |
| Hours per subject per week | `learn_activity_logs` | `SUM(duration_minutes)` GROUP BY `subject_tags`, `date_trunc('week', activity_date)` |
| Reading completion | `learn_reading_progress` | `COUNT(*) WHERE status = 'completed'` |
| Journal entry count | `learn_journal_entries` | `COUNT(*)` with date range filter |

### §12.2 Premium Tier (Advanced Analytics) `[S§15.2]` — Phase 2

Requires periodic materialized snapshots in `learn_progress_snapshots`:

| Metric | Computation | Storage |
|--------|-------------|---------|
| Trend visualization | Weekly aggregates over time | JSONB in progress_snapshots |
| Subject balance | % time per subject vs methodology benchmarks | Compared at query time to method:: config |
| Methodology benchmarks | Average hours/activities from methodology config | Read from method_definitions.mastery_paths |

### §12.3 Snapshot Strategy

1. **Background job** runs weekly (Sunday midnight UTC) for each active student
2. Computes all metrics and writes a `learn_progress_snapshots` row
3. Dashboard queries read snapshots for trend data
4. No real-time computation for trends — latest snapshot is at most 7 days old
5. "Current week" data is computed on-the-fly and merged with historical snapshots

### §12.4 Per-Student Isolation

All progress queries include `student_id` in the WHERE clause. There is no cross-student
aggregation. Each student's data is fully isolated. Parents see one student at a time
(selected in the UI).

---

## §13 Subject Taxonomy (Domain Deep-Dive 5)

### §13.1 Hierarchy

Three-level tree: `[S§8.3]`

```
Category (level 0)
└── Subject (level 1)
    └── Topic (level 2)

Example:
Mathematics (level 0)
├── Algebra (level 1)
│   ├── Linear Equations (level 2)
│   └── Quadratic Equations (level 2)
├── Geometry (level 1)
│   ├── Triangles (level 2)
│   └── Circles (level 2)
└── Calculus (level 1)

Language Arts (level 0)
├── Writing (level 1)
│   ├── Creative Writing (level 2)
│   └── Essay Writing (level 2)
├── Reading Comprehension (level 1)
└── Grammar (level 1)
```

### §13.2 Platform-Managed

The canonical taxonomy lives in `learn_subject_taxonomy`. It is:
- Admin-editable (database operations only — no code changes required) `[S§8.3]`
- Seeded during initial migration with a comprehensive set of subjects
- Referenced by slug in `subject_tags TEXT[]` columns across all learning tables

### §13.3 Family Custom Subjects

Families can extend (not replace) the taxonomy via `learn_custom_subjects`:
- Custom subjects have an optional `parent_taxonomy_id` for hierarchy placement
- Custom subject slugs are family-scoped (unique within a family, may collide across families)
- When querying taxonomy, custom subjects are merged with platform subjects for the family

### §13.4 Cross-Domain Consistency

The same taxonomy is used for: `[S§8.3]`
- **Learning tools**: `subject_tags` on activities, journals, assessments, projects
- **Marketplace**: Content listing tagging (via `mkt_listings.subject_tags`)
- **Compliance**: Subject-hours reporting (via `comply::` consuming `learn::` data)

---

## §14 Data Export (Domain Deep-Dive 6)

### §14.1 Export Contents `[S§8.5]`

| Data Type | Included | Format |
|-----------|:---:|--------|
| Activity logs | ✓ | Array of activity objects |
| Journal entries | ✓ | Array of journal objects |
| Reading lists + progress | ✓ | Lists with nested progress |
| Assessment results | ✓ (Phase 2) | Array of result objects |
| Project progress | ✓ (Phase 2) | Array with milestone data |
| Progress snapshots | ✓ | Historical trend data |
| Custom subjects | ✓ | Array of custom taxonomy nodes |

### §14.2 Export Flow

```
POST /v1/learning/export
    │
    ▼
1. Check no active export (ExportRepository.HasActiveExport)
    │
    ▼
2. Create export_request (status = 'pending')
    │
    ▼
3. Enqueue ExportGenerationTask (background via asynq)
    │
    ▼
4. Task executes:
    ├── Set status = 'processing'
    ├── Query all family learning data (FamilyScope)
    ├── Generate JSON/CSV bundle
    ├── Upload to media:: (get signed URL)
    ├── Set status = 'completed', file_url, expires_at = now() + 24h
    └── Publish DataExportReady event → notify::
    │
    ▼
5. Family downloads via GET /v1/learning/export/:id → file_url
```

### §14.3 Constraints

- **Available regardless of subscription tier** `[S§8.5]`
- **24-hour download window** — `expires_at` is set to creation + 24 hours
- **One active export at a time** — prevents abuse; returns `ExportAlreadyInProgress` if concurrent
- **Async generation** — export runs in background task; notification sent on completion
- **All student data** — export includes data for all students in the family

---

## §15 Methodology-Specific Tools (Domain Deep-Dive 7) — Phase 3

Each methodology-specific tool follows the core tool data patterns: taggable, searchable,
exportable, per-student. All tools resolved via `method::` — never branching on methodology
name. `[S§8.1.8]`

### §15.1 Tool Specifications

| Tool | Methodology | Key Fields | DB Table (Phase 3) |
|------|-------------|------------|---------------------|
| **Nature Journals** | Charlotte Mason | observation_type, species, location_name, weather, drawing_url, photo_urls | `learn_nature_journals` |
| **Trivium Tracker** | Classical | subject, stage (grammar/logic/rhetoric), assessment_notes, stage_evidence | `learn_trivium_entries` |
| **Rhythm Planner** | Waldorf | day_of_week, blocks [{type, label, start_time, end_time}], is_template | `learn_rhythm_plans` |
| **Observation Logs** | Montessori | work_chosen, duration, concentration_level (1-5), social_interactions, follow_up_notes | `learn_observation_logs` |
| **Habit Tracking** | Charlotte Mason | habit_name, target_frequency, completions[], streak_count, parent_notes | `learn_habit_tracks` |
| **Interest-Led Logs** | Unschooling | trigger (what sparked it), description, auto_subject_tags, duration, follow_up | `learn_interest_logs` |
| **Handwork Tracker** | Waldorf | project_name, materials[], techniques[], photos[], status | `learn_handwork_entries` |
| **Practical Life** | Montessori | activity_name, life_skill_category, age_appropriate (bool), mastery_level (intro/developing/mastered) | `learn_practical_life` |

### §15.2 Common Patterns

All Phase 3 tools share these patterns:
- `family_id UUID NOT NULL` + `student_id UUID NOT NULL` (FamilyScope + per-student)
- `subject_tags TEXT[]` for taxonomy integration
- `entry_date DATE NOT NULL` for timeline integration
- `attachments JSONB` for media
- Family-scoped RLS policy
- Full-text search vector where applicable
- CRUD endpoints under `/v1/learning/students/:student_id/{tool-slug}`

---

## §16 Premium Tier Gating (Domain Deep-Dive 8)

### §16.1 Enforcement Strategy `[S§15.1, S§15.2]`

Premium gating is enforced at the **service layer**, not the handler or database layer.
The service checks `iam_families.subscription_tier` before allowing premium operations.

### §16.2 Tool Tier Matrix

| Tool | Free | Premium |
|------|:---:|:---:|
| Activities | ✓ | ✓ |
| Reading Lists | ✓ | ✓ |
| Journaling & Narration | ✓ (basic) | ✓ (enhanced storage) |
| Basic Progress Tracking | ✓ | ✓ |
| Assessment Engine | ✓ | ✓ |
| Content Viewer | ✓ (purchase required) | ✓ |
| Video Player | ✓ (purchase required) | ✓ |
| Lesson Sequences | ✓ (purchase required) | ✓ |
| Student Assignments | ✓ | ✓ |
| Tests & Grades (Phase 2) | ✓ (basic) | ✓ (advanced analytics) |
| Projects (Phase 2) | ✓ (basic) | ✓ (portfolio integration) |
| Content Annotations (Phase 2) | ✗ | ✓ |
| Advanced Analytics (Phase 2) | ✗ | ✓ |
| Compliance Reporting (Phase 2) | ✗ | ✓ |
| Methodology-Specific Tools (Phase 3) | ✓ (basic) | ✓ (advanced) |
| Data Export | ✓ | ✓ |

### §16.3 Downgrade Behavior

When a family downgrades from premium to free:
- **Historical data preserved** — all existing records remain readable
- **New entries blocked** for premium-only features (e.g., advanced analytics views return 403)
- **Premium tool data enters read-only mode** — existing entries visible, new creation returns `PremiumRequired`

---

## §17 Error Types

`LearningError` types defined in `internal/learn/domain/errors.go`. Maps to `AppError` via
custom error types and `errors.Is`/`errors.As` `[00-core §6.4]`. `[CODING §2.2, CODING §8.3]`

```go
// internal/learn/domain/errors.go

package domain

import "errors"

// --- Student ----------------------------------------------------------------

// ErrStudentNotFound indicates the student was not found.
var ErrStudentNotFound = errors.New("student not found")

// ErrStudentNotInFamily indicates the student does not belong to this family.
var ErrStudentNotInFamily = errors.New("student does not belong to this family")

// --- Activity ---------------------------------------------------------------

// ErrActivityNotFound indicates the activity was not found.
var ErrActivityNotFound = errors.New("activity not found")

// ErrActivityDefNotFound indicates the activity definition was not found.
var ErrActivityDefNotFound = errors.New("activity definition not found")

// --- Journal ----------------------------------------------------------------

// ErrJournalNotFound indicates the journal entry was not found.
var ErrJournalNotFound = errors.New("journal entry not found")

// ErrInvalidEntryType indicates an invalid journal entry type.
type ErrInvalidEntryType struct {
    EntryType string
}

func (e *ErrInvalidEntryType) Error() string {
    return "invalid entry type: " + e.EntryType
}

// --- Reading ----------------------------------------------------------------

// ErrReadingItemNotFound indicates the reading item was not found.
var ErrReadingItemNotFound = errors.New("reading item not found")

// ErrReadingListNotFound indicates the reading list was not found.
var ErrReadingListNotFound = errors.New("reading list not found")

// ErrReadingProgressNotFound indicates the reading progress was not found.
var ErrReadingProgressNotFound = errors.New("reading progress not found")

// ErrDuplicateReadingProgress indicates duplicate tracking of the same reading item.
var ErrDuplicateReadingProgress = errors.New("already tracking this reading item")

// ErrInvalidReadingStatusTransition indicates an invalid reading status transition.
var ErrInvalidReadingStatusTransition = errors.New("invalid reading status transition")

// --- Subject Taxonomy -------------------------------------------------------

// ErrInvalidSubjectTag indicates an invalid subject tag.
type ErrInvalidSubjectTag struct {
    Tag string
}

func (e *ErrInvalidSubjectTag) Error() string {
    return "invalid subject tag: " + e.Tag
}

// ErrDuplicateCustomSubject indicates a duplicate custom subject.
var ErrDuplicateCustomSubject = errors.New("duplicate custom subject")

// --- Validation -------------------------------------------------------------

// ErrFutureDateNotAllowed indicates the activity date cannot be in the future.
var ErrFutureDateNotAllowed = errors.New("activity date cannot be in the future")

// ErrNegativeDuration indicates the duration cannot be negative.
var ErrNegativeDuration = errors.New("duration cannot be negative")

// --- Artifact Links ---------------------------------------------------------

// ErrSourceNotFound indicates the source content was not found.
var ErrSourceNotFound = errors.New("source content not found")

// ErrTargetNotFound indicates the target content was not found.
var ErrTargetNotFound = errors.New("target content not found")

// ErrDuplicateLink indicates a duplicate artifact link.
var ErrDuplicateLink = errors.New("duplicate artifact link")

// ErrInvalidArtifactType indicates an invalid artifact type.
type ErrInvalidArtifactType struct {
    ArtifactType string
}

func (e *ErrInvalidArtifactType) Error() string {
    return "invalid artifact type: " + e.ArtifactType
}

// --- Tools & Tier -----------------------------------------------------------

// ErrToolNotActive indicates the tool is not active for this student.
var ErrToolNotActive = errors.New("tool not active for this student")

// ErrPremiumRequired indicates a premium subscription is required.
var ErrPremiumRequired = errors.New("premium subscription required")

// --- Export -----------------------------------------------------------------

// ErrExportAlreadyInProgress indicates an export is already in progress.
var ErrExportAlreadyInProgress = errors.New("export already in progress")

// ErrExportNotReady indicates the export is not ready yet.
var ErrExportNotReady = errors.New("export not ready")

// ErrExportExpired indicates the export has expired.
var ErrExportExpired = errors.New("export has expired")

// --- Publisher --------------------------------------------------------------

// ErrNotPublisherMember indicates the caller is not a member of this publisher.
var ErrNotPublisherMember = errors.New("not a member of this publisher")

// --- Attachments ------------------------------------------------------------

// ErrAttachmentTooLarge indicates the attachment is too large.
var ErrAttachmentTooLarge = errors.New("attachment too large")

// ErrInvalidAttachmentType indicates an invalid attachment type.
var ErrInvalidAttachmentType = errors.New("invalid attachment type")

// --- Infrastructure ---------------------------------------------------------

// ErrDatabase wraps a database error.
type ErrDatabase struct {
    Err error
}

func (e *ErrDatabase) Error() string {
    return "database error"
}

func (e *ErrDatabase) Unwrap() error {
    return e.Err
}
```

### §17.1 Error-to-HTTP Mapping

| LearningError | HTTP Status | Error Code |
|--------------|-------------|------------|
| `ErrStudentNotFound` | 404 | `student_not_found` |
| `ErrStudentNotInFamily` | 403 | `student_not_in_family` |
| `ErrActivityNotFound` | 404 | `activity_not_found` |
| `ErrActivityDefNotFound` | 404 | `activity_def_not_found` |
| `ErrJournalNotFound` | 404 | `journal_not_found` |
| `ErrInvalidEntryType` | 422 | `invalid_entry_type` |
| `ErrReadingItemNotFound` | 404 | `reading_item_not_found` |
| `ErrReadingListNotFound` | 404 | `reading_list_not_found` |
| `ErrReadingProgressNotFound` | 404 | `reading_progress_not_found` |
| `ErrDuplicateReadingProgress` | 409 | `duplicate_reading_progress` |
| `ErrInvalidReadingStatusTransition` | 422 | `invalid_status_transition` |
| `ErrInvalidSubjectTag` | 422 | `invalid_subject_tag` |
| `ErrDuplicateCustomSubject` | 409 | `duplicate_custom_subject` |
| `ErrFutureDateNotAllowed` | 422 | `future_date_not_allowed` |
| `ErrNegativeDuration` | 422 | `negative_duration` |
| `ErrSourceNotFound` | 404 | `source_not_found` |
| `ErrTargetNotFound` | 404 | `target_not_found` |
| `ErrDuplicateLink` | 409 | `duplicate_link` |
| `ErrInvalidArtifactType` | 422 | `invalid_artifact_type` |
| `ErrToolNotActive` | 403 | `tool_not_active` |
| `ErrPremiumRequired` | 403 | `premium_required` |
| `ErrExportAlreadyInProgress` | 409 | `export_already_in_progress` |
| `ErrExportNotReady` | 202 | `export_not_ready` |
| `ErrExportExpired` | 410 | `export_expired` |
| `ErrNotPublisherMember` | 403 | `not_publisher_member` |
| `ErrAttachmentTooLarge` | 413 | `attachment_too_large` |
| `ErrInvalidAttachmentType` | 422 | `invalid_attachment_type` |
| `ErrDatabase` | 500 | `internal_error` |

---

## §18 Cross-Domain Interactions

### §18.1 learn:: Provides (consumed by other domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `LearningService` interface methods | `comply::` | `LearningService` interface via AppState |
| `ActivityLogged` event | `comply::`, `recs::`, `notify::` | Domain event — attendance tracking, recommendation signal, streak check |
| `MilestoneAchieved` event | `notify::`, `social::` | Domain event — notification, optional milestone post |
| `BookCompleted` event | `notify::` | Domain event — streak check, reading milestone |
| `DataExportReady` event | `notify::` | Domain event — export download notification |
| `QuizCompleted` event | `notify::`, `recs::` | Domain event — notify parent of quiz score, recommendation signal |
| `SequenceAdvanced` event | `recs::` | Domain event — recommendation signal for sequence engagement |
| `SequenceCompleted` event | `notify::`, `recs::` | Domain event — notify parent of sequence completion, recommendation signal |
| `AssignmentCompleted` event | `notify::` | Domain event — notify parent of assignment completion |

### §18.2 learn:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| Tool resolution | `method::MethodologyService` | Resolve family/student active tool set `[S§4.2]` |
| Methodology config | `method::MethodologyService` | Tool guidance, terminology `[S§8.4]` |
| Publisher membership | `mkt::` service | Verify publisher membership for content definitions |
| Student profiles | `iam::IamService` | Validate student belongs to family |
| Family subscription tier | `iam::IamService` | Premium tier gating `[S§15.1, S§15.2]` |
| Purchased content metadata | `mkt::` events | Content integration on purchase `[S§18.4]` |

### §18.3 Events learn:: Publishes

Defined in `internal/learn/events.go`. `[CODING §8.4]`

```go
// internal/learn/events.go

// ActivityLogged is published when an activity is logged for a student.
type ActivityLogged struct {
    FamilyID        FamilyID    `json:"family_id"`
    StudentID       uuid.UUID   `json:"student_id"`
    ActivityID      uuid.UUID   `json:"activity_id"`
    SubjectTags     []string    `json:"subject_tags"`
    DurationMinutes *int16      `json:"duration_minutes"`
    ActivityDate    time.Time   `json:"activity_date"`
}

func (e ActivityLogged) EventName() string { return "learn.activity_logged" }

// MilestoneAchieved is published when a student reaches a learning milestone.
type MilestoneAchieved struct {
    FamilyID      FamilyID  `json:"family_id"`
    StudentID     uuid.UUID `json:"student_id"`
    StudentName   string    `json:"student_name"`
    MilestoneType string    `json:"milestone_type"` // "books_completed", "activity_streak", "subject_hours"
    Description   string    `json:"description"`
}

func (e MilestoneAchieved) EventName() string { return "learn.milestone_achieved" }

// BookCompleted is published when a student finishes a book.
type BookCompleted struct {
    FamilyID         FamilyID  `json:"family_id"`
    StudentID        uuid.UUID `json:"student_id"`
    ReadingItemID    uuid.UUID `json:"reading_item_id"`
    ReadingItemTitle string    `json:"reading_item_title"`
}

func (e BookCompleted) EventName() string { return "learn.book_completed" }

// DataExportReady is published when a data export is ready for download.
type DataExportReady struct {
    FamilyID  FamilyID  `json:"family_id"`
    ExportID  uuid.UUID `json:"export_id"`
    FileURL   string    `json:"file_url"`
    ExpiresAt time.Time `json:"expires_at"`
}

func (e DataExportReady) EventName() string { return "learn.data_export_ready" }

// --- Interactive Learning Events ----------------------- [S§8.1.9, S§8.1.12, S§8.6]

// QuizCompleted is published when a quiz is fully scored.
type QuizCompleted struct {
    FamilyID      FamilyID  `json:"family_id"`
    StudentID     uuid.UUID `json:"student_id"`
    QuizDefID     uuid.UUID `json:"quiz_def_id"`
    QuizSessionID uuid.UUID `json:"quiz_session_id"`
    Score         float64   `json:"score"`
    MaxScore      float64   `json:"max_score"`
    Passed        bool      `json:"passed"`
}

func (e QuizCompleted) EventName() string { return "learn.quiz_completed" }

// SequenceAdvanced is published when a student completes a sequence item.
type SequenceAdvanced struct {
    FamilyID        FamilyID  `json:"family_id"`
    StudentID       uuid.UUID `json:"student_id"`
    SequenceDefID   uuid.UUID `json:"sequence_def_id"`
    ItemIndex       int16     `json:"item_index"`
    ItemContentType string    `json:"item_content_type"`
    ItemContentID   uuid.UUID `json:"item_content_id"`
}

func (e SequenceAdvanced) EventName() string { return "learn.sequence_advanced" }

// SequenceCompleted is published when a student completes all required items in a sequence.
type SequenceCompleted struct {
    FamilyID      FamilyID  `json:"family_id"`
    StudentID     uuid.UUID `json:"student_id"`
    SequenceDefID uuid.UUID `json:"sequence_def_id"`
}

func (e SequenceCompleted) EventName() string { return "learn.sequence_completed" }

// AssignmentCompleted is published when a student completes an assignment.
type AssignmentCompleted struct {
    FamilyID     FamilyID  `json:"family_id"`
    StudentID    uuid.UUID `json:"student_id"`
    AssignmentID uuid.UUID `json:"assignment_id"`
    ContentType  string    `json:"content_type"`
    ContentID    uuid.UUID `json:"content_id"`
}

func (e AssignmentCompleted) EventName() string { return "learn.assignment_completed" }
```

### §18.4 Events learn:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `StudentCreated { family_id, student_id }` | `iam::` | Initialize student learning defaults (no-op for now; placeholder for Phase 2+ setup) |
| `StudentDeleted { family_id, student_id }` | `iam::` | Cascade-delete all learning data for the student |
| `FamilyDeletionScheduled { family_id }` | `iam::` | Trigger data export opportunity; prepare for cascade deletion `[S§16.3]` |
| `PurchaseCompleted { family_id, content_metadata }` | `mkt::` | Integrate purchased content into family's tool access `[S§18.4]` |
| `MethodologyConfigUpdated` | `method::` | Invalidate cached tool resolution results |

```go
// internal/learn/event_handlers.go

import (
    "context"

    iamevents "internal/iam/events"
    mktevents "internal/mkt/events"
    methodevents "internal/method/events"
)

// StudentCreatedHandler handles the StudentCreated event for the learning domain.
type StudentCreatedHandler struct {
    learningService LearningService
}

// Handle processes a StudentCreated event.
func (h *StudentCreatedHandler) Handle(ctx context.Context, event *iamevents.StudentCreated) error {
    return h.learningService.HandleStudentCreated(
        ctx,
        event.FamilyID,
        event.StudentID,
    )
}

// StudentDeletedHandler handles the StudentDeleted event for the learning domain.
type StudentDeletedHandler struct {
    learningService LearningService
}

// Handle processes a StudentDeleted event.
func (h *StudentDeletedHandler) Handle(ctx context.Context, event *iamevents.StudentDeleted) error {
    return h.learningService.HandleStudentDeleted(
        ctx,
        event.FamilyID,
        event.StudentID,
    )
}

// FamilyDeletionScheduledHandler handles the FamilyDeletionScheduled event.
type FamilyDeletionScheduledHandler struct {
    learningService LearningService
}

// Handle processes a FamilyDeletionScheduled event.
func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event *iamevents.FamilyDeletionScheduled) error {
    return h.learningService.HandleFamilyDeletionScheduled(ctx, event.FamilyID)
}

// PurchaseCompletedHandler handles the PurchaseCompleted event.
type PurchaseCompletedHandler struct {
    learningService LearningService
}

// Handle processes a PurchaseCompleted event.
func (h *PurchaseCompletedHandler) Handle(ctx context.Context, event *mktevents.PurchaseCompleted) error {
    return h.learningService.HandlePurchaseCompleted(
        ctx,
        event.FamilyID,
        event.ContentMetadata,
    )
}

// MethodologyConfigUpdatedHandler handles the MethodologyConfigUpdated event.
type MethodologyConfigUpdatedHandler struct {
    learningService LearningService
}

// Handle processes a MethodologyConfigUpdated event.
func (h *MethodologyConfigUpdatedHandler) Handle(ctx context.Context, event *methodevents.MethodologyConfigUpdated) error {
    return h.learningService.HandleMethodologyConfigUpdated(ctx)
}
```

---

## §19 Phase Scope

### Phase 1 — Foundation `[S§19]`

**In scope**:
- Layer 1 tables: `learn_activity_defs`, `learn_reading_items`, `learn_questions`,
  `learn_quiz_defs`, `learn_quiz_questions`, `learn_sequence_defs`,
  `learn_sequence_items`, `learn_video_defs` (8 definition tables)
- Layer 2 table: `learn_artifact_links` (1 link table)
- Layer 3 tables: `learn_activity_logs`, `learn_journal_entries`, `learn_reading_progress`,
  `learn_progress_snapshots`, `learn_reading_lists`, `learn_reading_list_items`,
  `learn_custom_subjects`, `learn_export_requests`, `learn_quiz_sessions`,
  `learn_sequence_progress`, `learn_student_assignments`,
  `learn_video_progress` (12 tracking tables)
- Platform table: `learn_subject_taxonomy` (1 taxonomy table, seeded)
- RLS policies for all Layer 3 tables
- Activity log CRUD with validation (invariants enforced)
- Journal entry CRUD (freeform, narration, reflection)
- Reading item definition CRUD
- Reading progress tracking (start, update, complete)
- Reading list management (create, add/remove items, delete)
- Artifact links (create, query, delete)
- Basic progress tracking (activity counts, hours by subject, reading completion)
- Interactive assessment engine (question bank, quiz builder, quiz sessions, auto-scoring)
- In-platform content viewer (PDF rendering, page tracking)
- Video player (self-hosted HLS + external embeds, progress tracking)
- Lesson sequences (ordered content paths, progression tracking)
- Student assignments (parent assigns content, due dates, status tracking)
- Tool resolution (delegates to `method::`)
- Subject taxonomy (platform + family custom)
- Data export (async, JSON/CSV)
- ~72 Phase 1 endpoints
- `LearningService` interface + `LearningServiceImpl`
- 16 repository interfaces + PostgreSQL implementations
- `MediaAdapter` interface
- Domain `domain/` subdirectory: activity.go, journal.go, reading_list.go, progress.go,
  taxonomy.go, quiz_session.go, sequence.go, assignment.go, errors.go
- `LearningError` types + HTTP mapping
- Domain events: `ActivityLogged`, `MilestoneAchieved`, `BookCompleted`, `DataExportReady`, `QuizCompleted`, `SequenceAdvanced`, `SequenceCompleted`, `AssignmentCompleted`
- Event handlers: `StudentCreatedHandler`, `StudentDeletedHandler`,
  `FamilyDeletionScheduledHandler`, `PurchaseCompletedHandler`,
  `MethodologyConfigUpdatedHandler`
- All Phase 1 models (request, response, internal)
- OpenAPI spec + TypeScript type generation

### Phase 2 — Depth

**In scope**:
- Tests & Grades (compliance-focused): `learn_assessment_defs`, `learn_assessment_results`, `learn_grading_scales`
- Projects: `learn_project_defs`, `learn_project_progress`
- Content annotations/bookmarks (in-platform content viewer enhancement)
- Advanced analytics (trend visualization, subject balance, methodology benchmarks)
- ISBN book search/import
- Reading list sharing (to friends/groups via `social::`)
- Per-student manual tool activation/deactivation
- ~30 Phase 2 endpoints

### Phase 3 — Specialize

**In scope**:
- All 8 methodology-specific tools (see §15)
- Methodology-specific definition + instance tables
- Mastery paths integration
- ~24 Phase 3 endpoints

---

## §20 Verification Checklist

Numbered assertions that MUST be true when the learn:: implementation is correct.

### Activity Logging

1. `POST /v1/learning/students/:student_id/activities` creates an activity log and publishes `ActivityLogged` event
2. Activity with `activity_date` in the future returns 422 (`future_date_not_allowed`)
3. Activity with negative `duration_minutes` returns 422 (`negative_duration`)
4. Activity with invalid `subject_tags` returns 422 (`invalid_subject_tag`)
5. Activity with `content_id` referencing a nonexistent definition returns 404
6. Activity for a student not in the caller's family returns 403 (`student_not_in_family`)
7. Activities are filterable by subject, date range, and student

### Journaling

8. `POST /v1/learning/students/:student_id/journals` creates a journal entry with valid `entry_type`
9. Journal entry with invalid `entry_type` returns 422 (`invalid_entry_type`)
10. Journal entries are searchable by keyword via full-text search
11. Journal entries are filterable by type, date range, and student

### Reading

12. `POST /v1/learning/students/:student_id/reading` starts tracking a reading item
13. Duplicate tracking of the same reading item for the same student returns 409
14. Reading status transitions follow: to_read → in_progress → completed (invalid transitions return 422)
15. Completing a reading item publishes `BookCompleted` event
16. Completing a reading item auto-generates an activity log entry
17. Reading lists group reading items with sort order
18. Deleting a reading list does not delete the reading items or progress records

### Content Definitions

19. Activity definitions require publisher membership for create/update/delete
20. Reading items require publisher membership for create/update
21. Definitions are publicly browsable by any authenticated user
22. Soft-deleting a definition sets `is_active = false` but preserves existing references

### Artifact Links

23. Links require ownership of the source content's publisher
24. Duplicate links (same source + target) return 409
25. Bidirectional queries work correctly ("what is this about" and "everything about this")
26. Deleting a content definition cascades to remove its artifact links

### Progress

27. `GET /v1/learning/progress/:student_id` returns activity counts, hours by subject, and reading completion
28. Progress queries respect date range filters
29. All progress data is per-student (no cross-student aggregation)

### Tools

30. `GET /v1/learning/tools` returns the family's resolved tool set from `method::`
31. `GET /v1/learning/tools/:student_id` respects per-student methodology overrides
32. Tool guidance from `method::` config_overrides is included in the response

### Subject Taxonomy

33. `GET /v1/learning/subjects` returns platform taxonomy merged with family custom subjects
34. Custom subjects are family-scoped (visible only to the creating family)
35. Custom subject slugs are unique within a family

### Data Export

36. `POST /v1/learning/export` creates an async export job
37. Concurrent export requests for the same family return 409 (`export_already_in_progress`)
38. Export includes all learning data types (activities, journals, reading, progress)
39. Export is available regardless of subscription tier
40. Completed exports have a 24-hour download window
41. `DataExportReady` event is published when export generation completes

### Premium Gating

42. Premium-only features return 403 (`premium_required`) for free-tier families
43. Downgrading from premium preserves historical data in read-only mode
44. Data export is always available regardless of tier

### Privacy & Security

45. All Layer 3 tables have family-scoped RLS policies
46. Layer 1 tables have no family-scoped RLS (publisher-based access at app level)
47. Attachments are validated via `media::` (magic bytes, not just extension)
48. No methodology name branching — all tool behavior resolved via `method::` config

### Event Handling

49. `StudentDeleted` event cascade-deletes all learning data for the student
50. `FamilyDeletionScheduled` event triggers export opportunity
51. `PurchaseCompleted` event integrates purchased content metadata
52. `MethodologyConfigUpdated` event invalidates cached tool resolution

---

## §21 Module Structure

```
internal/learn/
├── handler.go                # Echo route handlers (thin layer only)
├── service.go                # LearningServiceImpl — orchestration
├── repository.go             # GormActivityDefRepository, GormActivityLogRepository,
│                             # GormReadingItemRepository, GormReadingProgressRepository,
│                             # GormReadingListRepository, GormJournalEntryRepository,
│                             # GormArtifactLinkRepository, GormProgressRepository,
│                             # GormSubjectTaxonomyRepository, GormExportRepository
├── models.go                 # Request/response types, internal types, query params,
│                             # GORM model definitions
├── ports.go                  # LearningService interface, all repository interfaces,
│                             # MediaAdapter interface
├── events.go                 # ActivityLogged, MilestoneAchieved, BookCompleted, QuizCompleted, SequenceAdvanced, SequenceCompleted, AssignmentCompleted,
│                             # DataExportReady
├── event_handlers.go         # StudentCreatedHandler, StudentDeletedHandler,
│                             # FamilyDeletionScheduledHandler,
│                             # PurchaseCompletedHandler,
│                             # MethodologyConfigUpdatedHandler
├── export.go                 # ExportGenerationTask — background worker (asynq) for
│                             # async data export generation
└── domain/
    ├── activity.go           # Activity aggregate root — invariant enforcement
    ├── journal.go            # Journal entry validation — entry type rules
    ├── reading_list.go       # Reading list aggregate — item management, status transitions
    ├── progress.go           # Progress computation — metric aggregation logic
    ├── taxonomy.go           # Taxonomy validation — slug generation, hierarchy checks
    └── errors.go             # LearningError types
```
