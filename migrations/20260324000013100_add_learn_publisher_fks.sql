-- +goose Up
-- Migration: 20260324000013100_add_learn_publisher_fks.sql
-- Adds FK constraints from learn Layer-1 tables to mkt_publishers(id).
-- Deferred from 20260323000011 because mkt:: (20260324000013) must be created first.
-- [06-learn §3.2]

ALTER TABLE learn_activity_defs   ADD CONSTRAINT fk_learn_activity_defs_publisher   FOREIGN KEY (publisher_id) REFERENCES mkt_publishers(id);
ALTER TABLE learn_assessment_defs ADD CONSTRAINT fk_learn_assessment_defs_publisher FOREIGN KEY (publisher_id) REFERENCES mkt_publishers(id);
ALTER TABLE learn_reading_items   ADD CONSTRAINT fk_learn_reading_items_publisher   FOREIGN KEY (publisher_id) REFERENCES mkt_publishers(id);
ALTER TABLE learn_project_defs    ADD CONSTRAINT fk_learn_project_defs_publisher    FOREIGN KEY (publisher_id) REFERENCES mkt_publishers(id);
ALTER TABLE learn_video_defs      ADD CONSTRAINT fk_learn_video_defs_publisher      FOREIGN KEY (publisher_id) REFERENCES mkt_publishers(id);
ALTER TABLE learn_questions       ADD CONSTRAINT fk_learn_questions_publisher       FOREIGN KEY (publisher_id) REFERENCES mkt_publishers(id);
ALTER TABLE learn_quiz_defs       ADD CONSTRAINT fk_learn_quiz_defs_publisher       FOREIGN KEY (publisher_id) REFERENCES mkt_publishers(id);
ALTER TABLE learn_sequence_defs   ADD CONSTRAINT fk_learn_sequence_defs_publisher   FOREIGN KEY (publisher_id) REFERENCES mkt_publishers(id);

-- +goose Down
ALTER TABLE learn_sequence_defs   DROP CONSTRAINT IF EXISTS fk_learn_sequence_defs_publisher;
ALTER TABLE learn_quiz_defs       DROP CONSTRAINT IF EXISTS fk_learn_quiz_defs_publisher;
ALTER TABLE learn_questions       DROP CONSTRAINT IF EXISTS fk_learn_questions_publisher;
ALTER TABLE learn_video_defs      DROP CONSTRAINT IF EXISTS fk_learn_video_defs_publisher;
ALTER TABLE learn_project_defs    DROP CONSTRAINT IF EXISTS fk_learn_project_defs_publisher;
ALTER TABLE learn_reading_items   DROP CONSTRAINT IF EXISTS fk_learn_reading_items_publisher;
ALTER TABLE learn_assessment_defs DROP CONSTRAINT IF EXISTS fk_learn_assessment_defs_publisher;
ALTER TABLE learn_activity_defs   DROP CONSTRAINT IF EXISTS fk_learn_activity_defs_publisher;
