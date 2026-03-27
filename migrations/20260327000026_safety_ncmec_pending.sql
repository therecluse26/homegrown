-- +goose Up
-- =============================================================================
-- Migration: 20260327000026_safety_ncmec_pending.sql
-- Domain: 11-safety (NCMEC pending report queue)
-- =============================================================================
-- Stores NCMEC reports queued for manual filing when real Thorn integration
-- is unavailable. Ensures no CSAM-adjacent events are silently discarded. [11-safety §7.1]

CREATE TABLE safety_ncmec_pending_reports (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    upload_id           UUID NOT NULL,             -- media upload that triggered report
    uploader_family_id  UUID NOT NULL,
    uploader_parent_id  UUID NOT NULL,
    evidence_key        TEXT NOT NULL,             -- R2 storage key for evidence
    csam_hash           VARCHAR(255),
    confidence          NUMERIC(5, 4),
    matched_database    VARCHAR(100),
    upload_timestamp    TIMESTAMPTZ NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'queued'
                        CHECK (status IN ('queued', 'filed', 'dismissed')),
    notes               TEXT,
    filed_at            TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_safety_ncmec_pending_status
    ON safety_ncmec_pending_reports(status, created_at)
    WHERE status = 'queued';

-- +goose Down
DROP INDEX IF EXISTS idx_safety_ncmec_pending_status;
DROP TABLE IF EXISTS safety_ncmec_pending_reports;
