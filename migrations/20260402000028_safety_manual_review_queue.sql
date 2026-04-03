-- +goose Up
-- =============================================================================
-- Migration: 20260402000028_safety_manual_review_queue.sql
-- Domain: 11-safety (manual review queue for CSAM detection)
-- =============================================================================
-- Stores content flagged for manual CSAM review when real Thorn/PhotoDNA
-- integration is unavailable. [11-safety §7.1, CRIT-1]

CREATE TABLE safety_manual_review_queue (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    storage_key     TEXT NOT NULL,
    review_type     VARCHAR(30) NOT NULL DEFAULT 'csam_scan'
                    CHECK (review_type IN ('csam_scan', 'moderation', 'report')),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'reviewed', 'escalated', 'dismissed')),
    reviewer_notes  TEXT,
    reviewed_by     UUID,
    reviewed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_safety_manual_review_status
    ON safety_manual_review_queue(status, created_at)
    WHERE status = 'pending';

-- +goose Down
DROP INDEX IF EXISTS idx_safety_manual_review_status;
DROP TABLE IF EXISTS safety_manual_review_queue;
