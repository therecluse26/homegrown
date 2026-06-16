-- +goose Up
-- +goose StatementBegin
ALTER TABLE recs_recommendations
    DROP CONSTRAINT recs_recommendations_source_signal_check;

ALTER TABLE recs_recommendations
    ADD CONSTRAINT recs_recommendations_source_signal_check
        CHECK (source_signal IN (
            'methodology_match', 'popularity', 'seasonal',
            'progress_gap', 'age_transition', 'purchase_history',
            'reading_history', 'exploration', 'learner_profile_fit'
        ));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE recs_recommendations
    DROP CONSTRAINT recs_recommendations_source_signal_check;

ALTER TABLE recs_recommendations
    ADD CONSTRAINT recs_recommendations_source_signal_check
        CHECK (source_signal IN (
            'methodology_match', 'popularity', 'seasonal',
            'progress_gap', 'age_transition', 'purchase_history',
            'reading_history', 'exploration'
        ));
-- +goose StatementEnd
