-- +goose Up
-- Learner Profile v0: per-student preference vector + fit-badge plumbing
-- [18-learner-profile §2]

-- 1. Extend wizard step enum for the optional learner-profile onboarding step.
ALTER TYPE onb_wizard_step_enum ADD VALUE IF NOT EXISTS 'learner_profile';

-- 2. Core learner profile table: family-scoped, one row per student.
CREATE TABLE IF NOT EXISTS learner_profiles (
    id                  UUID        PRIMARY KEY DEFAULT uuidv7(),
    family_id           UUID        NOT NULL REFERENCES iam_families(id)  ON DELETE CASCADE,
    student_id          UUID        NOT NULL REFERENCES iam_students(id)  ON DELETE CASCADE,

    -- Preference vector: 6 dimensions, each 0.0–1.0.
    -- NULL = unanswered; do NOT use 0.0 as "unanswered" sentinel.
    activity_format     NUMERIC(5,4),   -- 0=text/listen, 1=hands-on/build/move
    session_length      NUMERIC(5,4),   -- 0=short-bursts, 1=long-deep-dives
    motivation          NUMERIC(5,4),   -- 0=mastery, 1=discovery
    solo_collaborative  NUMERIC(5,4),   -- 0=solo, 1=collaborative
    structure           NUMERIC(5,4),   -- 0=step-by-step, 1=open-ended
    outdoor_kinesthetic NUMERIC(5,4),   -- 0=not-important, 1=think-better-moving

    -- Interest taxonomy: controlled vocabulary matching mkt_listings.subject_tags
    interests           TEXT[]      NOT NULL DEFAULT '{}',

    -- Confidence and provenance
    answered_count      SMALLINT    NOT NULL DEFAULT 0,
    confidence          NUMERIC(4,3) NOT NULL DEFAULT 0.000,
    source              TEXT        NOT NULL DEFAULT 'declared',
    respondent          TEXT        NOT NULL DEFAULT 'parent',

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT learner_profiles_student_unique     UNIQUE (student_id),
    CONSTRAINT learner_profiles_confidence_range   CHECK (confidence BETWEEN 0 AND 1),
    CONSTRAINT learner_profiles_source_values      CHECK (source IN ('declared', 'blended')),
    CONSTRAINT learner_profiles_respondent_values  CHECK (respondent IN ('parent', 'child'))
);

ALTER TABLE learner_profiles ENABLE ROW LEVEL SECURITY;

CREATE POLICY learner_profiles_family_rls ON learner_profiles
    USING (family_id = current_setting('app.family_id')::UUID);

CREATE INDEX IF NOT EXISTS idx_learner_profiles_family  ON learner_profiles (family_id);
CREATE INDEX IF NOT EXISTS idx_learner_profiles_student ON learner_profiles (student_id);

-- 3. Content preference tagging: nullable JSONB added to marketplace listings.
--    NULL = untagged; badge is suppressed for untagged content (fail-safe).
ALTER TABLE mkt_listings ADD COLUMN IF NOT EXISTS preference_tags JSONB;

COMMENT ON COLUMN mkt_listings.preference_tags IS
    'Learner preference alignment tags. NULL = untagged (badge suppressed). '
    'Keys match learner_profiles dimension columns. Values 0.0–1.0.';

-- 4. Fit score columns on recs_recommendations (pre-computed by daily task).
ALTER TABLE recs_recommendations
    ADD COLUMN IF NOT EXISTS fit_score NUMERIC(4,3),
    ADD COLUMN IF NOT EXISTS fit_why   TEXT;

-- +goose Down
ALTER TABLE recs_recommendations DROP COLUMN IF EXISTS fit_why;
ALTER TABLE recs_recommendations DROP COLUMN IF EXISTS fit_score;
ALTER TABLE mkt_listings DROP COLUMN IF EXISTS preference_tags;
DROP TABLE IF EXISTS learner_profiles;
-- Note: cannot DROP a pg enum value; onb_wizard_step_enum retains 'learner_profile'
