-- +goose Up
-- +goose StatementBegin

-- Strip literal quotation marks that were stored in titles by earlier tool forms.
UPDATE learn_activity_logs
SET title = trim(both '"' from title)
WHERE title ~ '^".*"$';

-- Add structured metadata column for tool-specific key-value fields.
-- Keeps description as a pure free-text notes field.
ALTER TABLE learn_activity_logs
ADD COLUMN IF NOT EXISTS metadata JSONB NOT NULL DEFAULT '{}';

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE learn_activity_logs DROP COLUMN IF EXISTS metadata;

-- +goose StatementEnd
