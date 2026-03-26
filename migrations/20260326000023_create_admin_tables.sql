-- +goose Up
-- =============================================================================
-- Migration: 20260326_000023_create_admin_tables.sql
-- Domain: 16-admin (Administration)
-- =============================================================================

-- Feature flags: simple key-value feature toggles [16-admin §3.1]
CREATE TABLE admin_feature_flags (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    key             VARCHAR(100) NOT NULL UNIQUE,
    description     TEXT NOT NULL,
    -- Flag state
    enabled         BOOLEAN NOT NULL DEFAULT false,
    -- Optional: percentage rollout (0-100, NULL = all-or-nothing)
    rollout_percentage SMALLINT CHECK (
        rollout_percentage IS NULL OR
        (rollout_percentage >= 0 AND rollout_percentage <= 100)
    ),
    -- Optional: restrict to specific family IDs (for beta testing)
    allowed_family_ids UUID[] DEFAULT NULL,
    -- Metadata
    created_by      UUID NOT NULL REFERENCES iam_parents(id),
    updated_by      UUID REFERENCES iam_parents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Admin audit log: immutable record of all admin actions [16-admin §3.1, §8]
CREATE TABLE admin_audit_log (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    admin_id        UUID NOT NULL REFERENCES iam_parents(id),
    action          VARCHAR(50) NOT NULL
                    CHECK (action IN (
                        'user_suspend', 'user_unsuspend', 'user_ban',
                        'content_remove', 'content_restore',
                        'flag_create', 'flag_update', 'flag_delete',
                        'method_config_update',
                        'retention_policy_update',
                        'recovery_resolve', 'recovery_deny',
                        'payout_approve', 'payout_reject',
                        'system_config_update'
                    )),
    -- What was acted upon
    target_type     VARCHAR(30) NOT NULL
                    CHECK (target_type IN (
                        'family', 'parent', 'student',
                        'post', 'comment', 'message', 'listing', 'review',
                        'feature_flag', 'methodology',
                        'retention_policy', 'recovery_request',
                        'payout', 'system'
                    )),
    target_id       UUID,
    -- Action details (what changed)
    details         JSONB NOT NULL DEFAULT '{}'::JSONB,
    -- IP and user agent for security
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Audit log is append-only — no UPDATE or DELETE policies [16-admin §8.1]
CREATE INDEX idx_admin_audit_log_admin ON admin_audit_log(admin_id);
CREATE INDEX idx_admin_audit_log_action ON admin_audit_log(action);
CREATE INDEX idx_admin_audit_log_target ON admin_audit_log(target_type, target_id);
CREATE INDEX idx_admin_audit_log_created ON admin_audit_log(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_admin_audit_log_created;
DROP INDEX IF EXISTS idx_admin_audit_log_target;
DROP INDEX IF EXISTS idx_admin_audit_log_action;
DROP INDEX IF EXISTS idx_admin_audit_log_admin;
DROP TABLE IF EXISTS admin_audit_log;
DROP TABLE IF EXISTS admin_feature_flags;
