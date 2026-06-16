-- +goose Up
-- +goose StatementBegin
ALTER TABLE iam_students
    ADD COLUMN IF NOT EXISTS custom_attributes JSONB NOT NULL DEFAULT '{}';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE iam_students DROP COLUMN IF EXISTS custom_attributes;
-- +goose StatementEnd
