-- +goose Up
-- Migration: 20260322000004_create_disc_tables.sql
-- Discovery domain: quiz definitions, quiz results, state guides, content pages.
-- These are public content tables — NO RLS policies. [ARCH §5.1, 03-discover §3.2]
-- All disc_ tables store platform-authored content; no user data except quiz results'
-- nullable family_id (set on claim) and optional session_token.

-- Quiz status enum
CREATE TYPE disc_quiz_status_enum AS ENUM ('draft', 'active', 'archived');

-- State guide status enum
CREATE TYPE disc_guide_status_enum AS ENUM ('draft', 'published', 'review_due');

-- Content page status enum
CREATE TYPE disc_content_status_enum AS ENUM ('draft', 'published');

-- Quiz definitions: questions + scoring weights (internal), explanations per methodology.
-- Only one quiz may be active at a time (enforced by partial unique index below).
CREATE TABLE disc_quiz_definitions (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    version      INTEGER      NOT NULL DEFAULT 1,
    title        TEXT         NOT NULL,
    description  TEXT         NOT NULL DEFAULT '',
    status       disc_quiz_status_enum NOT NULL DEFAULT 'draft',
    -- questions: [{id, category, text, help_text, answers: [{id, text, weights: {slug: float}}]}]
    -- weights are internal-only and MUST NOT be returned to API clients. [03-discover §3.1]
    questions    JSONB        NOT NULL DEFAULT '[]',
    -- explanations: {slug: {match_text, mismatch_text}} — score >= 0.5 → match_text
    explanations JSONB        NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Enforce at most one active quiz definition. [03-discover §3.1]
CREATE UNIQUE INDEX idx_disc_quiz_definitions_active
    ON disc_quiz_definitions(status)
    WHERE status = 'active';

-- Quiz results: stores anonymous quiz submissions with a URL-safe share_id.
-- family_id is nullable — set when a logged-in family claims a result. [03-discover §3.1]
CREATE TABLE disc_quiz_results (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    quiz_definition_id UUID        NOT NULL REFERENCES disc_quiz_definitions(id),
    share_id           TEXT        NOT NULL UNIQUE,                -- 12-char base62 nanoid
    session_token      TEXT,                                       -- anonymous session tracking
    answers            JSONB       NOT NULL DEFAULT '{}',          -- {question_id: answer_id}
    scores             JSONB       NOT NULL DEFAULT '{}',          -- {methodology_slug: score_percentage}
    recommendations    JSONB       NOT NULL DEFAULT '[]',          -- [{slug, display_name, score_percentage, explanation}]
    family_id          UUID,                                       -- NULL until claimed
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_disc_quiz_results_share_id ON disc_quiz_results(share_id);
CREATE INDEX idx_disc_quiz_results_session  ON disc_quiz_results(session_token) WHERE session_token IS NOT NULL;
CREATE INDEX idx_disc_quiz_results_family   ON disc_quiz_results(family_id) WHERE family_id IS NOT NULL;

-- State legal guides: one row per US state + DC. [03-discover §3.2]
-- status='draft' until content is reviewed and approved by an admin.
CREATE TABLE disc_state_guides (
    id               UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    state_code       CHAR(2)               NOT NULL UNIQUE, -- e.g., 'CA', 'NY', 'DC'
    state_name       TEXT                  NOT NULL,
    status           disc_guide_status_enum NOT NULL DEFAULT 'draft',
    -- requirements: {notification, assessment, subjects, recordkeeping, notes}
    requirements     JSONB                 NOT NULL DEFAULT '{}',
    last_reviewed_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ           NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ           NOT NULL DEFAULT now()
);

CREATE INDEX idx_disc_state_guides_status ON disc_state_guides(status);

-- Content pages: Homeschooling 101 and advocacy content metadata. [03-discover §3.3]
CREATE TABLE disc_content_pages (
    id         UUID                    PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       TEXT                    NOT NULL UNIQUE,
    category   TEXT                    NOT NULL, -- '101', 'advocacy'
    title      TEXT                    NOT NULL,
    body       JSONB                   NOT NULL DEFAULT '{}', -- structured content blocks
    status     disc_content_status_enum NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ             NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ             NOT NULL DEFAULT now()
);

CREATE INDEX idx_disc_content_pages_category ON disc_content_pages(category, status);

-- +goose Down
DROP TABLE IF EXISTS disc_content_pages;
DROP TABLE IF EXISTS disc_state_guides;
DROP TABLE IF EXISTS disc_quiz_results;
DROP TABLE IF EXISTS disc_quiz_definitions;
DROP TYPE IF EXISTS disc_content_status_enum;
DROP TYPE IF EXISTS disc_guide_status_enum;
DROP TYPE IF EXISTS disc_quiz_status_enum;
