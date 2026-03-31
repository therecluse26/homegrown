-- +goose Up
-- =============================================================================
-- Migration: 20260327000025_create_lifecycle_tables.sql
-- Domain: 15-lifecycle (Data Lifecycle — GDPR/COPPA)
-- =============================================================================

-- Export requests: family data export queue [15-data-lifecycle §3.1]
CREATE TABLE lifecycle_export_requests (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id           UUID NOT NULL REFERENCES iam_families(id),
    requested_by        UUID NOT NULL REFERENCES iam_parents(id),
    status              VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'expired')),
    format              VARCHAR(10) NOT NULL DEFAULT 'json'
                        CHECK (format IN ('json', 'csv')),
    include_domains     TEXT[],
    archive_key         TEXT,                          -- R2 object key once complete
    download_expires_at TIMESTAMPTZ,
    size_bytes          BIGINT,
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '7 days'
);

CREATE INDEX idx_lifecycle_exports_family
    ON lifecycle_export_requests(family_id, created_at DESC);
CREATE INDEX idx_lifecycle_exports_status
    ON lifecycle_export_requests(status)
    WHERE status IN ('pending', 'processing');
CREATE INDEX idx_lifecycle_exports_expires
    ON lifecycle_export_requests(expires_at)
    WHERE status = 'completed';

-- Deletion requests: account and student deletion queue [15-data-lifecycle §3.1]
CREATE TABLE lifecycle_deletion_requests (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id           UUID NOT NULL REFERENCES iam_families(id),
    requested_by        UUID NOT NULL REFERENCES iam_parents(id),
    reason              TEXT,
    deletion_type       VARCHAR(20) NOT NULL
                        CHECK (deletion_type IN ('family', 'student', 'coppa')),
    student_id          UUID REFERENCES iam_students(id),
    status              VARCHAR(20) NOT NULL DEFAULT 'grace_period'
                        CHECK (status IN ('pending', 'grace_period', 'processing', 'completed', 'cancelled')),
    grace_period_ends_at TIMESTAMPTZ NOT NULL,
    export_offered      BOOLEAN NOT NULL DEFAULT false,
    export_request_id   UUID REFERENCES lifecycle_export_requests(id),
    domain_status       JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    cancelled_at        TIMESTAMPTZ
);

CREATE INDEX idx_lifecycle_deletions_family
    ON lifecycle_deletion_requests(family_id, created_at DESC);
CREATE INDEX idx_lifecycle_deletions_status
    ON lifecycle_deletion_requests(status, grace_period_ends_at)
    WHERE status IN ('grace_period', 'processing');

-- Recovery requests: unauthenticated account recovery [15-data-lifecycle §3.1]
-- NOTE: No family_id — these are pre-auth. RLS is NOT applied. [15-data-lifecycle §3.2]
CREATE TABLE lifecycle_recovery_requests (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    email               VARCHAR(254) NOT NULL,
    verification_method VARCHAR(30) NOT NULL DEFAULT 'email'
                        CHECK (verification_method IN ('email', 'support_ticket', 'identity_document')),
    status              VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'verified', 'escalated', 'completed', 'denied')),
    support_ticket_id   VARCHAR(100),
    resolved_parent_id  UUID REFERENCES iam_parents(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at         TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '7 days'
);

CREATE INDEX idx_lifecycle_recovery_email
    ON lifecycle_recovery_requests(email, created_at DESC);
CREATE INDEX idx_lifecycle_recovery_status
    ON lifecycle_recovery_requests(status)
    WHERE status IN ('pending', 'escalated');

-- Retention policies: per-category data retention rules [15-data-lifecycle §3.1]
CREATE TABLE lifecycle_retention_policies (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    data_category   VARCHAR(60) UNIQUE NOT NULL,
    retention_days  INT NOT NULL,
    applies_to      VARCHAR(30) NOT NULL CHECK (applies_to IN ('family', 'student', 'system')),
    description     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed default retention policies [15-data-lifecycle §3.1]
INSERT INTO lifecycle_retention_policies (data_category, retention_days, applies_to, description) VALUES
    ('activity_logs',     365 * 7, 'student', 'Learning activity history'),
    ('journal_entries',   365 * 7, 'student', 'Student journal entries'),
    ('assessment_records',365 * 7, 'student', 'Assessment results and test scores'),
    ('attendance_records',365 * 7, 'student', 'Attendance records for compliance'),
    ('transcripts',       365 * 7, 'student', 'Academic transcripts'),
    ('social_posts',      365 * 3, 'family',  'Community posts and comments'),
    ('messages',          365 * 2, 'family',  'Direct and group messages'),
    ('purchase_records',  365 * 7, 'family',  'Transaction and purchase history (legal minimum)'),
    ('export_archives',   7,       'family',  'Generated data export archives');

-- Family scoping is enforced at the GORM level via ScopedTransaction (ADR-008).
-- PostgreSQL RLS is NOT used.

-- +goose Down

DROP TABLE IF EXISTS lifecycle_retention_policies;
DROP TABLE IF EXISTS lifecycle_recovery_requests;
DROP TABLE IF EXISTS lifecycle_deletion_requests;
DROP TABLE IF EXISTS lifecycle_export_requests;
