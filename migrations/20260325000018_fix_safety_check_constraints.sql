-- +goose Up

-- ═══════════════════════════════════════════════════════════════════════════════
-- Fix safety domain CHECK constraints to match spec [11-safety §3.1, §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════

-- ─── safety_reports ──────────────────────────────────────────────────────────

-- H1: target_type missing message, group, event, question, quiz, sequence
ALTER TABLE safety_reports DROP CONSTRAINT IF EXISTS safety_reports_target_type_check;
ALTER TABLE safety_reports ADD CONSTRAINT safety_reports_target_type_check
    CHECK (target_type IN (
        'post', 'comment', 'message', 'profile',
        'group', 'event', 'listing', 'review', 'upload',
        'question', 'quiz', 'sequence'
    ));

-- H2: category mismatch — add misinformation, methodology_hostility; remove hate_speech, impersonation
ALTER TABLE safety_reports DROP CONSTRAINT IF EXISTS safety_reports_category_check;
ALTER TABLE safety_reports ADD CONSTRAINT safety_reports_category_check
    CHECK (category IN (
        'inappropriate_content', 'harassment', 'spam',
        'misinformation', 'csam_child_safety',
        'methodology_hostility', 'other'
    ));

-- H14: missing idx_safety_reports_assigned partial index
CREATE INDEX IF NOT EXISTS idx_safety_reports_assigned
    ON safety_reports(assigned_admin_id)
    WHERE assigned_admin_id IS NOT NULL AND status = 'in_review';

-- ─── safety_content_flags ────────────────────────────────────────────────────

-- C1/H5: source CHECK needs 'automated' and 'community_report' (spec values)
ALTER TABLE safety_content_flags DROP CONSTRAINT IF EXISTS safety_content_flags_source_check;
ALTER TABLE safety_content_flags ADD CONSTRAINT safety_content_flags_source_check
    CHECK (source IN ('automated', 'community_report'));

-- H3: target_type missing message, group, event, question, quiz, sequence
ALTER TABLE safety_content_flags DROP CONSTRAINT IF EXISTS safety_content_flags_target_type_check;
ALTER TABLE safety_content_flags ADD CONSTRAINT safety_content_flags_target_type_check
    CHECK (target_type IN (
        'post', 'comment', 'message', 'profile',
        'group', 'event', 'listing', 'review', 'upload',
        'question', 'quiz', 'sequence'
    ));

-- H4: flag_type missing violence, text_violation
ALTER TABLE safety_content_flags DROP CONSTRAINT IF EXISTS safety_content_flags_flag_type_check;
ALTER TABLE safety_content_flags ADD CONSTRAINT safety_content_flags_flag_type_check
    CHECK (flag_type IN (
        'csam', 'explicit_content', 'violence', 'spam',
        'harassment', 'prohibited_content', 'text_violation',
        'suspected_underage_exploitation'
    ));

-- ─── safety_mod_actions ──────────────────────────────────────────────────────

-- C3/C4/H15: action_type needs content_restored, appeal_granted; replace ban_reversed with appeal_granted
ALTER TABLE safety_mod_actions DROP CONSTRAINT IF EXISTS safety_mod_actions_action_type_check;
ALTER TABLE safety_mod_actions ADD CONSTRAINT safety_mod_actions_action_type_check
    CHECK (action_type IN (
        'content_removed', 'warning_issued', 'account_suspended',
        'account_banned', 'content_restored', 'suspension_lifted',
        'appeal_granted', 'escalate_to_csam'
    ));

-- ─── safety_appeals ──────────────────────────────────────────────────────────

-- H6: status missing in_review
ALTER TABLE safety_appeals DROP CONSTRAINT IF EXISTS safety_appeals_status_check;
ALTER TABLE safety_appeals ADD CONSTRAINT safety_appeals_status_check
    CHECK (status IN ('pending', 'in_review', 'granted', 'denied'));

-- ─── safety_ncmec_reports ────────────────────────────────────────────────────

-- C2: Remove FK on family_id — legal evidence must persist even if family is deleted
-- The constraint name is auto-generated; drop by looking it up.
DO $$
DECLARE
    fk_name TEXT;
BEGIN
    SELECT conname INTO fk_name
    FROM pg_constraint
    WHERE conrelid = 'safety_ncmec_reports'::regclass
      AND contype = 'f'
      AND EXISTS (
          SELECT 1 FROM unnest(conkey) AS k
          JOIN pg_attribute a ON a.attrelid = conrelid AND a.attnum = k
          WHERE a.attname = 'family_id'
      );
    IF fk_name IS NOT NULL THEN
        EXECUTE 'ALTER TABLE safety_ncmec_reports DROP CONSTRAINT ' || fk_name;
    END IF;
END $$;

-- H7: status missing 'confirmed'
ALTER TABLE safety_ncmec_reports DROP CONSTRAINT IF EXISTS safety_ncmec_reports_status_check;
ALTER TABLE safety_ncmec_reports ADD CONSTRAINT safety_ncmec_reports_status_check
    CHECK (status IN ('pending', 'submitted', 'confirmed', 'failed'));

-- ─── safety_bot_signals ──────────────────────────────────────────────────────

-- H14: missing idx_safety_bot_signals_family index
CREATE INDEX IF NOT EXISTS idx_safety_bot_signals_family
    ON safety_bot_signals(family_id, created_at DESC);

-- +goose Down

-- Revert bot_signals index
DROP INDEX IF EXISTS idx_safety_bot_signals_family;

-- Revert ncmec_reports status
ALTER TABLE safety_ncmec_reports DROP CONSTRAINT IF EXISTS safety_ncmec_reports_status_check;
ALTER TABLE safety_ncmec_reports ADD CONSTRAINT safety_ncmec_reports_status_check
    CHECK (status IN ('pending', 'submitted', 'failed'));

-- Restore ncmec_reports FK on family_id
ALTER TABLE safety_ncmec_reports
    ADD CONSTRAINT safety_ncmec_reports_family_id_fkey
    FOREIGN KEY (family_id) REFERENCES iam_families(id) ON DELETE CASCADE;

-- Revert appeals status
ALTER TABLE safety_appeals DROP CONSTRAINT IF EXISTS safety_appeals_status_check;
ALTER TABLE safety_appeals ADD CONSTRAINT safety_appeals_status_check
    CHECK (status IN ('pending', 'granted', 'denied'));

-- Revert mod_actions action_type
ALTER TABLE safety_mod_actions DROP CONSTRAINT IF EXISTS safety_mod_actions_action_type_check;
ALTER TABLE safety_mod_actions ADD CONSTRAINT safety_mod_actions_action_type_check
    CHECK (action_type IN (
        'content_removed', 'account_suspended', 'account_banned',
        'warning_issued', 'suspension_lifted', 'ban_reversed',
        'escalate_to_csam'
    ));

-- Revert content_flags flag_type
ALTER TABLE safety_content_flags DROP CONSTRAINT IF EXISTS safety_content_flags_flag_type_check;
ALTER TABLE safety_content_flags ADD CONSTRAINT safety_content_flags_flag_type_check
    CHECK (flag_type IN (
        'explicit_content', 'csam', 'harassment',
        'hate_speech', 'spam', 'prohibited_content',
        'suspected_underage_exploitation'
    ));

-- Revert content_flags target_type
ALTER TABLE safety_content_flags DROP CONSTRAINT IF EXISTS safety_content_flags_target_type_check;
ALTER TABLE safety_content_flags ADD CONSTRAINT safety_content_flags_target_type_check
    CHECK (target_type IN ('post', 'comment', 'upload', 'review', 'listing', 'profile'));

-- Revert content_flags source
ALTER TABLE safety_content_flags DROP CONSTRAINT IF EXISTS safety_content_flags_source_check;
ALTER TABLE safety_content_flags ADD CONSTRAINT safety_content_flags_source_check
    CHECK (source IN ('auto_moderation', 'csam_scan', 'community_report', 'admin_flag'));

-- Revert reports assigned index
DROP INDEX IF EXISTS idx_safety_reports_assigned;

-- Revert reports category
ALTER TABLE safety_reports DROP CONSTRAINT IF EXISTS safety_reports_category_check;
ALTER TABLE safety_reports ADD CONSTRAINT safety_reports_category_check
    CHECK (category IN (
        'csam_child_safety', 'harassment', 'hate_speech',
        'spam', 'inappropriate_content', 'impersonation', 'other'
    ));

-- Revert reports target_type
ALTER TABLE safety_reports DROP CONSTRAINT IF EXISTS safety_reports_target_type_check;
ALTER TABLE safety_reports ADD CONSTRAINT safety_reports_target_type_check
    CHECK (target_type IN ('post', 'comment', 'upload', 'review', 'listing', 'profile'));
