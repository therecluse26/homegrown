-- +goose Up
-- +goose StatementBegin

-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 21: Fix methodology_id UUID → methodology_slug TEXT
--
-- The entire codebase uses methodology string slugs (iam_families.primary_methodology_slug,
-- method_definitions.slug TEXT PRIMARY KEY). The original migration incorrectly created
-- UUID columns in recs_signals and recs_popularity_scores. This migration corrects the
-- schema to use TEXT slugs, matching the rest of the system. [13-recs §3.2]
-- ─────────────────────────────────────────────────────────────────────────────

-- 1. recs_signals: replace methodology_id UUID with methodology_slug TEXT
ALTER TABLE recs_signals DROP COLUMN methodology_id;
ALTER TABLE recs_signals ADD COLUMN methodology_slug TEXT NOT NULL DEFAULT '';

-- 2. recs_popularity_scores: replace methodology_id UUID with methodology_slug TEXT
--    Drop dependent indexes first, then recreate with new column.
DROP INDEX IF EXISTS idx_recs_popularity_listing_method_period;
DROP INDEX IF EXISTS idx_recs_popularity_method_score;

ALTER TABLE recs_popularity_scores DROP COLUMN methodology_id;
ALTER TABLE recs_popularity_scores ADD COLUMN methodology_slug TEXT NOT NULL DEFAULT '';

-- Recreate unique index with methodology_slug
CREATE UNIQUE INDEX idx_recs_popularity_listing_method_period
    ON recs_popularity_scores (listing_id, methodology_slug, period_start);

-- Recreate query index with methodology_slug
CREATE INDEX idx_recs_popularity_method_score
    ON recs_popularity_scores (methodology_slug, popularity_score DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse: restore methodology_id UUID columns
DROP INDEX IF EXISTS idx_recs_popularity_method_score;
DROP INDEX IF EXISTS idx_recs_popularity_listing_method_period;

ALTER TABLE recs_popularity_scores DROP COLUMN methodology_slug;
ALTER TABLE recs_popularity_scores ADD COLUMN methodology_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';

CREATE UNIQUE INDEX idx_recs_popularity_listing_method_period
    ON recs_popularity_scores (listing_id, methodology_id, period_start);
CREATE INDEX idx_recs_popularity_method_score
    ON recs_popularity_scores (methodology_id, popularity_score DESC);

ALTER TABLE recs_signals DROP COLUMN methodology_slug;
ALTER TABLE recs_signals ADD COLUMN methodology_id UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000';

-- +goose StatementEnd
