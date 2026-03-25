-- +goose Up
-- +goose StatementBegin

-- ─────────────────────────────────────────────────────────────────────────────
-- Domain 13: Recommendations & Signals (recs::)
-- [13-recs §3.2]
--
-- CHECK constraints enforce enums (not PostgreSQL ENUM types) to avoid
-- ALTER TYPE limitations when adding new values in future migrations. [CODING §4.1]
-- ─────────────────────────────────────────────────────────────────────────────

-- recs_signals: Raw Event-Derived Learning Signals
-- Family-scoped. 90-day retention (purged by PurgeStaleSignalsTask). [13-recs §3.2]
CREATE TABLE recs_signals (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id      UUID REFERENCES iam_students(id) ON DELETE SET NULL,
    signal_type     TEXT NOT NULL CHECK (signal_type IN (
                        'activity_logged', 'book_completed', 'purchase_completed'
                    )),
    -- Denormalized methodology snapshot at signal time. [13-recs §9.3]
    methodology_id  UUID NOT NULL,
    -- Signal-specific payload (JSONB). Schema varies by signal_type. [13-recs §3.2]
    payload         JSONB NOT NULL DEFAULT '{}',
    signal_date     DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Primary query: "all signals for this family in the last N days"
CREATE INDEX idx_recs_signals_family_date
    ON recs_signals (family_id, signal_date DESC);

-- Purge task: "delete signals older than 90 days"
CREATE INDEX idx_recs_signals_created_at
    ON recs_signals (created_at);

-- Signal type filtering for algorithm queries
CREATE INDEX idx_recs_signals_family_type
    ON recs_signals (family_id, signal_type);

-- FK index: student deletion (ON DELETE SET NULL)
CREATE INDEX idx_recs_signals_student
    ON recs_signals (student_id) WHERE student_id IS NOT NULL;


-- recs_recommendations: Pre-Computed Recommendations
-- Family-scoped. 14-day TTL, expired by daily batch task. [13-recs §3.2]
CREATE TABLE recs_recommendations (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id           UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    -- NULL = family-wide recommendation; non-NULL = student-specific
    student_id          UUID REFERENCES iam_students(id) ON DELETE CASCADE,
    recommendation_type TEXT NOT NULL CHECK (recommendation_type IN (
                            'marketplace_content', 'activity_idea',
                            'reading_suggestion', 'community_group'
                        )),
    target_entity_id    UUID NOT NULL,
    target_entity_label TEXT NOT NULL,
    source_signal       TEXT NOT NULL CHECK (source_signal IN (
                            'methodology_match', 'popularity', 'seasonal',
                            'progress_gap', 'age_transition', 'purchase_history',
                            'reading_history', 'exploration'
                        )),
    source_label        TEXT NOT NULL,
    score               REAL NOT NULL DEFAULT 0.0,
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN (
                            'active', 'dismissed', 'blocked', 'expired'
                        )),
    expires_at          TIMESTAMPTZ NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Primary query: "active recommendations for this family"
CREATE INDEX idx_recs_recommendations_family_status
    ON recs_recommendations (family_id, status) WHERE status = 'active';

-- Student-specific query: "active recommendations for this student"
CREATE INDEX idx_recs_recommendations_student_status
    ON recs_recommendations (student_id, status) WHERE status = 'active';

-- Dedup: prevent duplicate active recommendations for the same entity per family
CREATE UNIQUE INDEX idx_recs_recommendations_family_entity_active
    ON recs_recommendations (family_id, target_entity_id)
    WHERE status = 'active';

-- Expiry task: "find recommendations past their TTL"
CREATE INDEX idx_recs_recommendations_expires_at
    ON recs_recommendations (expires_at) WHERE status = 'active';


-- recs_recommendation_feedback: Parent Dismiss/Block Actions
-- Family-scoped. Persists after recommendation expires (blocked sources stay blocked). [13-recs §3.2]
CREATE TABLE recs_recommendation_feedback (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id           UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    recommendation_id   UUID NOT NULL REFERENCES recs_recommendations(id) ON DELETE CASCADE,
    action              TEXT NOT NULL CHECK (action IN ('dismiss', 'block')),
    -- For blocks: the entity blocked for future suppression [13-recs §13.2]
    blocked_entity_id   UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One feedback per recommendation (unique constraint)
CREATE UNIQUE INDEX idx_recs_feedback_recommendation
    ON recs_recommendation_feedback (recommendation_id);

-- Query: "all blocked entities for this family" (used by algorithm to filter candidates)
CREATE INDEX idx_recs_feedback_family_blocks
    ON recs_recommendation_feedback (family_id, blocked_entity_id)
    WHERE action = 'block';

-- Family deletion cascade index
CREATE INDEX idx_recs_feedback_family
    ON recs_recommendation_feedback (family_id);


-- recs_popularity_scores: Cross-Family Aggregated Popularity
-- NOT family-scoped (no PII). Per-methodology popularity for listings. [13-recs §3.2, §3.3]
CREATE TABLE recs_popularity_scores (
    id               UUID PRIMARY KEY DEFAULT uuidv7(),
    listing_id       UUID NOT NULL,
    methodology_id   UUID NOT NULL,
    period_start     DATE NOT NULL,
    period_end       DATE NOT NULL,
    -- Weighted purchase count with recency decay (lambda=0.03, half-life ~23 days) [13-recs §10.4]
    popularity_score REAL NOT NULL DEFAULT 0.0,
    purchase_count   INTEGER NOT NULL DEFAULT 0,
    computed_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Unique: one score per listing per methodology per period
CREATE UNIQUE INDEX idx_recs_popularity_listing_method_period
    ON recs_popularity_scores (listing_id, methodology_id, period_start);

-- Query: "top listings for this methodology"
CREATE INDEX idx_recs_popularity_method_score
    ON recs_popularity_scores (methodology_id, popularity_score DESC);

-- Purge: remove expired period windows
CREATE INDEX idx_recs_popularity_period
    ON recs_popularity_scores (period_end);


-- recs_anonymized_interactions: COPPA-Safe Anonymized Data
-- NOT family-scoped (no PII). HMAC-anonymized learning interactions for future AI/ML. [13-recs §14]
-- INVARIANT: This table MUST NOT contain family_id or student_id. [13-recs §14.5]
CREATE TABLE recs_anonymized_interactions (
    id               UUID PRIMARY KEY DEFAULT uuidv7(),
    -- HMAC-SHA256(family_id, server_secret) — one-way, cannot recover family_id [13-recs §14.3]
    anonymous_id     TEXT NOT NULL,
    interaction_type TEXT NOT NULL CHECK (interaction_type IN (
                         'activity_logged', 'book_completed', 'purchase_completed'
                     )),
    methodology_slug TEXT NOT NULL,
    -- Coarsened to 3-year ranges for privacy [13-recs §14.1]
    age_band         TEXT NOT NULL CHECK (age_band IN (
                         '4-6', '7-9', '10-12', '13-15', '16-18'
                     )),
    subject_category TEXT,
    -- Rounded to nearest 5 minutes [13-recs §14.1]
    duration_minutes INTEGER,
    interaction_date DATE NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Analytics: query by methodology and interaction type
CREATE INDEX idx_recs_anon_methodology_type
    ON recs_anonymized_interactions (methodology_slug, interaction_type);

-- Analytics: query by age band
CREATE INDEX idx_recs_anon_age_band
    ON recs_anonymized_interactions (age_band);


-- recs_preferences: Family Recommendation Preferences
-- Family-scoped. One row per family (created on first access with defaults). [13-recs §3.2]
CREATE TABLE recs_preferences (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    enabled_types         TEXT[] NOT NULL DEFAULT ARRAY[
                              'marketplace_content', 'activity_idea',
                              'reading_suggestion', 'community_group'
                          ],
    exploration_frequency TEXT NOT NULL DEFAULT 'occasional' CHECK (
                              exploration_frequency IN ('off', 'occasional', 'frequent')
                          ),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One preference row per family
CREATE UNIQUE INDEX idx_recs_preferences_family
    ON recs_preferences (family_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS recs_preferences;
DROP TABLE IF EXISTS recs_anonymized_interactions;
DROP TABLE IF EXISTS recs_popularity_scores;
DROP TABLE IF EXISTS recs_recommendation_feedback;
DROP TABLE IF EXISTS recs_recommendations;
DROP TABLE IF EXISTS recs_signals;
-- +goose StatementEnd
