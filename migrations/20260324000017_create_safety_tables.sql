-- +goose Up

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: safety_reports — User-submitted content reports [11-safety §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE safety_reports (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    reporter_family_id  UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    reporter_parent_id  UUID NOT NULL,
    target_type         TEXT NOT NULL
                        CHECK (target_type IN ('post', 'comment', 'upload', 'review', 'listing', 'profile')),
    target_id           UUID NOT NULL,
    target_family_id    UUID,
    category            TEXT NOT NULL
                        CHECK (category IN (
                            'csam_child_safety', 'harassment', 'hate_speech',
                            'spam', 'inappropriate_content', 'impersonation', 'other'
                        )),
    description         TEXT,
    priority            TEXT NOT NULL DEFAULT 'normal'
                        CHECK (priority IN ('critical', 'high', 'normal')),
    status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN (
                            'pending', 'in_review', 'resolved_action_taken',
                            'resolved_no_action', 'dismissed'
                        )),
    assigned_admin_id   UUID,
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_safety_reports_reporter
    ON safety_reports(reporter_family_id, created_at DESC);

CREATE INDEX idx_safety_reports_status
    ON safety_reports(status)
    WHERE status IN ('pending', 'in_review');

CREATE INDEX idx_safety_reports_priority
    ON safety_reports(priority, created_at)
    WHERE status = 'pending';

CREATE INDEX idx_safety_reports_target
    ON safety_reports(target_type, target_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 2: safety_content_flags — Auto and manual content flags [11-safety §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE safety_content_flags (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    source              TEXT NOT NULL
                        CHECK (source IN ('auto_moderation', 'csam_scan', 'community_report', 'admin_flag')),
    target_type         TEXT NOT NULL
                        CHECK (target_type IN ('post', 'comment', 'upload', 'review', 'listing', 'profile')),
    target_id           UUID NOT NULL,
    target_family_id    UUID,
    flag_type           TEXT NOT NULL
                        CHECK (flag_type IN (
                            'explicit_content', 'csam', 'harassment',
                            'hate_speech', 'spam', 'prohibited_content',
                            'suspected_underage_exploitation'
                        )),
    confidence          DOUBLE PRECISION,
    labels              JSONB,
    report_id           UUID REFERENCES safety_reports(id),
    auto_rejected       BOOLEAN NOT NULL DEFAULT false,
    reviewed            BOOLEAN NOT NULL DEFAULT false,
    reviewed_by         UUID,
    reviewed_at         TIMESTAMPTZ,
    action_taken        BOOLEAN,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_safety_flags_unreviewed
    ON safety_content_flags(created_at)
    WHERE reviewed = false;

CREATE INDEX idx_safety_flags_target
    ON safety_content_flags(target_type, target_id);

CREATE INDEX idx_safety_flags_type
    ON safety_content_flags(flag_type, created_at DESC);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 3: safety_mod_actions — Admin moderation action log [11-safety §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE safety_mod_actions (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    admin_id                UUID NOT NULL,
    target_family_id        UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    target_parent_id        UUID,
    action_type             TEXT NOT NULL
                            CHECK (action_type IN (
                                'content_removed', 'account_suspended', 'account_banned',
                                'warning_issued', 'suspension_lifted', 'ban_reversed',
                                'escalate_to_csam'
                            )),
    reason                  TEXT NOT NULL,
    report_id               UUID REFERENCES safety_reports(id),
    content_snapshot        JSONB,
    suspension_days         INTEGER,
    suspension_expires_at   TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_safety_actions_target_family
    ON safety_mod_actions(target_family_id, created_at DESC);

CREATE INDEX idx_safety_actions_admin
    ON safety_mod_actions(admin_id, created_at DESC);

CREATE INDEX idx_safety_actions_report
    ON safety_mod_actions(report_id)
    WHERE report_id IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 4: safety_account_status — Family account moderation state [11-safety §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE safety_account_status (
    family_id               UUID PRIMARY KEY REFERENCES iam_families(id) ON DELETE CASCADE,
    status                  TEXT NOT NULL DEFAULT 'active'
                            CHECK (status IN ('active', 'suspended', 'banned')),
    suspended_at            TIMESTAMPTZ,
    suspension_expires_at   TIMESTAMPTZ,
    suspension_reason       TEXT,
    banned_at               TIMESTAMPTZ,
    ban_reason              TEXT,
    last_action_id          UUID REFERENCES safety_mod_actions(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_safety_account_status_active
    ON safety_account_status(status)
    WHERE status IN ('suspended', 'banned');

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 5: safety_appeals — User appeals of moderation actions [11-safety §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE safety_appeals (
    id                  UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id           UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    action_id           UUID NOT NULL REFERENCES safety_mod_actions(id),
    appeal_text         TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending', 'granted', 'denied')),
    assigned_admin_id   UUID,
    resolution_text     TEXT,
    resolved_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_safety_appeals_family
    ON safety_appeals(family_id, created_at DESC);

CREATE INDEX idx_safety_appeals_status
    ON safety_appeals(status)
    WHERE status = 'pending';

CREATE UNIQUE INDEX idx_safety_appeals_action
    ON safety_appeals(action_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 6: safety_ncmec_reports — CSAM evidence and NCMEC submission tracking [11-safety §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE safety_ncmec_reports (
    id                      UUID PRIMARY KEY DEFAULT uuidv7(),
    upload_id               UUID NOT NULL,
    family_id               UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    parent_id               UUID NOT NULL,
    csam_hash               TEXT,
    confidence              DOUBLE PRECISION,
    matched_database        TEXT,
    ncmec_report_id         TEXT,
    status                  TEXT NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'submitted', 'failed')),
    submitted_at            TIMESTAMPTZ,
    error_message           TEXT,
    evidence_storage_key    TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_safety_ncmec_pending
    ON safety_ncmec_reports(status)
    WHERE status = 'pending';

CREATE INDEX idx_safety_ncmec_upload
    ON safety_ncmec_reports(upload_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 7: safety_bot_signals — Bot detection signal log [11-safety §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE safety_bot_signals (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id   UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    parent_id   UUID NOT NULL,
    signal_type TEXT NOT NULL
                CHECK (signal_type IN (
                    'rapid_posting', 'mass_friend_requests', 'repetitive_content',
                    'suspicious_registration', 'rate_limit_exceeded'
                )),
    details     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_safety_bot_signals_parent_recent
    ON safety_bot_signals(parent_id, created_at DESC);

-- +goose Down
DROP TABLE IF EXISTS safety_bot_signals;
DROP TABLE IF EXISTS safety_ncmec_reports;
DROP TABLE IF EXISTS safety_appeals;
DROP TABLE IF EXISTS safety_account_status;
DROP TABLE IF EXISTS safety_mod_actions;
DROP TABLE IF EXISTS safety_content_flags;
DROP TABLE IF EXISTS safety_reports;
