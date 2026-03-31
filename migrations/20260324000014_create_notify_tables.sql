-- +goose Up
-- =============================================================================
-- Migration 014: Create notifications (notify) domain tables
-- =============================================================================
-- Tables: notify_notifications, notify_preferences, notify_digests,
-- notify_email_status. All family-scoped with RLS. [08-notify §3]
-- =============================================================================

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: notify_notifications — In-app notification store [S§13.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS notify_notifications (
    id                UUID        PRIMARY KEY DEFAULT uuidv7(),
    family_id         UUID        NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    notification_type TEXT        NOT NULL CHECK (notification_type IN (
        'friend_request_sent', 'friend_request_accepted',
        'message_received', 'event_cancelled',
        'methodology_changed', 'onboarding_completed',
        'activity_streak', 'milestone_achieved',
        'book_completed', 'data_export_ready',
        'purchase_completed', 'purchase_refunded',
        'creator_onboarded', 'content_flagged',
        'co_parent_added', 'family_deletion_scheduled',
        'subscription_created', 'subscription_changed',
        'subscription_cancelled', 'payout_completed'
    )),
    category          TEXT        NOT NULL CHECK (category IN (
        'social', 'learning', 'marketplace', 'system'
    )),
    title             TEXT        NOT NULL,
    body              TEXT        NOT NULL,
    action_url        TEXT,
    metadata          JSONB       NOT NULL DEFAULT '{}',
    is_read           BOOLEAN     NOT NULL DEFAULT false,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notify_notifications_family_created
    ON notify_notifications (family_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_notify_notifications_family_unread
    ON notify_notifications (family_id)
    WHERE is_read = false;

CREATE UNIQUE INDEX IF NOT EXISTS idx_notify_notifications_idempotency
    ON notify_notifications (family_id, notification_type, ((metadata->>'source_event_id')))
    WHERE metadata->>'source_event_id' IS NOT NULL;

-- Family scoping is enforced at the GORM level via ScopedTransaction (ADR-008).
-- PostgreSQL RLS is NOT used.

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 2: notify_preferences — Per-type per-channel preference overrides [S§13.3]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS notify_preferences (
    id                UUID        PRIMARY KEY DEFAULT uuidv7(),
    family_id         UUID        NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    notification_type TEXT        NOT NULL CHECK (notification_type IN (
        'friend_request_sent', 'friend_request_accepted',
        'message_received', 'event_cancelled',
        'methodology_changed', 'onboarding_completed',
        'activity_streak', 'milestone_achieved',
        'book_completed', 'data_export_ready',
        'purchase_completed', 'purchase_refunded',
        'creator_onboarded', 'content_flagged',
        'co_parent_added', 'family_deletion_scheduled',
        'subscription_created', 'subscription_changed',
        'subscription_cancelled', 'payout_completed'
    )),
    channel           TEXT        NOT NULL CHECK (channel IN ('in_app', 'email')),
    enabled           BOOLEAN     NOT NULL DEFAULT true,
    digest_frequency  TEXT        NOT NULL DEFAULT 'immediate' CHECK (digest_frequency IN (
        'immediate', 'daily', 'weekly', 'off'
    )),
    UNIQUE (family_id, notification_type, channel)
);

CREATE INDEX IF NOT EXISTS idx_notify_preferences_family
    ON notify_preferences (family_id);


-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 3: notify_digests — Compiled digest snapshots (Phase 2) [ARCH §12]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS notify_digests (
    id              UUID        PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID        NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    digest_type     TEXT        NOT NULL CHECK (digest_type IN ('daily', 'weekly')),
    period_start    TIMESTAMPTZ NOT NULL,
    period_end      TIMESTAMPTZ NOT NULL,
    content_json    JSONB       NOT NULL,
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notify_digests_unsent
    ON notify_digests (created_at)
    WHERE sent_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_notify_digests_family_period
    ON notify_digests (family_id, digest_type, period_start);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 4: notify_email_status — Email bounce tracking [08-notify §21.6]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS notify_email_status (
    id              UUID        PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID        NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    email           TEXT        NOT NULL,
    bounce_type     TEXT        NOT NULL CHECK (bounce_type IN ('hard', 'soft', 'spam_complaint')),
    bounced_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    raw_payload     JSONB
);

CREATE INDEX IF NOT EXISTS idx_notify_email_status_family
    ON notify_email_status (family_id);

CREATE INDEX IF NOT EXISTS idx_notify_email_status_email
    ON notify_email_status (email);

-- +goose Down
DROP TABLE IF EXISTS notify_email_status;
DROP TABLE IF EXISTS notify_digests;
DROP TABLE IF EXISTS notify_preferences;
DROP TABLE IF EXISTS notify_notifications;
