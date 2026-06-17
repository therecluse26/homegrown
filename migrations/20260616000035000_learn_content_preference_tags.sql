-- +goose Up
-- Add preference_tags to learn content so fit scoring works on learn browse surfaces.
-- NULL = untagged; badge suppressed for untagged content (fail-safe). [18-learner-profile §2.2]

ALTER TABLE learn_activity_defs ADD COLUMN IF NOT EXISTS preference_tags JSONB;
ALTER TABLE learn_reading_items  ADD COLUMN IF NOT EXISTS preference_tags JSONB;

COMMENT ON COLUMN learn_activity_defs.preference_tags IS
    'Learner preference alignment tags. NULL = untagged (badge suppressed). '
    'Keys match learner_profiles dimension columns. Values 0.0–1.0.';

COMMENT ON COLUMN learn_reading_items.preference_tags IS
    'Learner preference alignment tags. NULL = untagged (badge suppressed). '
    'Keys match learner_profiles dimension columns. Values 0.0–1.0.';

-- +goose Down
ALTER TABLE learn_reading_items  DROP COLUMN IF EXISTS preference_tags;
ALTER TABLE learn_activity_defs  DROP COLUMN IF EXISTS preference_tags;
