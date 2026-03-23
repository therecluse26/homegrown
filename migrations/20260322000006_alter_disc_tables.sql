-- +goose Up
-- Adds columns missing from the initial disc_ table migrations.
-- Fixes gaps identified against specs/domains/03-discover.md.

-- disc_state_guides: guide content + editorial metadata [03-discover §3.1, §15.17]
ALTER TABLE disc_state_guides
    ADD COLUMN guide_content    TEXT        NOT NULL DEFAULT '',
    ADD COLUMN next_review_due  TIMESTAMPTZ,
    ADD COLUMN reviewed_by      TEXT,
    ADD COLUMN legal_disclaimer TEXT        NOT NULL DEFAULT
        'This information is provided for educational purposes only and does not
constitute legal advice. Laws change frequently. Verify current requirements
with your state education department.';

-- disc_quiz_results: denormalized quiz version for fast retrieval [03-discover §8.2]
ALTER TABLE disc_quiz_results
    ADD COLUMN quiz_version SMALLINT NOT NULL DEFAULT 0;

-- disc_content_pages: spec uses content TEXT + SEO fields, not JSONB body [03-discover §3.1]
ALTER TABLE disc_content_pages
    ADD COLUMN content          TEXT     NOT NULL DEFAULT '',
    ADD COLUMN meta_title       TEXT,
    ADD COLUMN meta_description TEXT,
    ADD COLUMN display_order    SMALLINT NOT NULL DEFAULT 0;

ALTER TABLE disc_content_pages DROP COLUMN body;

-- +goose Down
ALTER TABLE disc_content_pages
    ADD COLUMN body JSONB NOT NULL DEFAULT '{}';

ALTER TABLE disc_content_pages
    DROP COLUMN display_order,
    DROP COLUMN meta_description,
    DROP COLUMN meta_title,
    DROP COLUMN content;

ALTER TABLE disc_quiz_results DROP COLUMN quiz_version;

ALTER TABLE disc_state_guides
    DROP COLUMN legal_disclaimer,
    DROP COLUMN reviewed_by,
    DROP COLUMN next_review_due,
    DROP COLUMN guide_content;
