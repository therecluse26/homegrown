-- +goose Up
-- Migration: 20260327000027_method_recs_config.sql
-- Adds recs-specific config fields to method_definitions so recs:: can look up
-- baseline subjects and stage transition ages without branching on methodology slug.
-- Resolves CODING-1 violation in internal/recs/tasks.go. [13-recs §10.2, §10.6]

ALTER TABLE method_definitions
    ADD COLUMN IF NOT EXISTS baseline_subjects TEXT[]   NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS transition_ages   JSONB    NOT NULL DEFAULT '[]';

-- Populate baseline subjects and transition ages for each built-in methodology.
-- baseline_subjects: core subject areas the methodology emphasizes.
-- transition_ages: [{stage, age}] — developmental stage transition points.

UPDATE method_definitions SET
    baseline_subjects = ARRAY['reading','mathematics','writing','science','history','art','nature_study','music','handicrafts','narration'],
    transition_ages   = '[{"stage":"Form II","age":9},{"stage":"Form III","age":12},{"stage":"Form IV","age":15}]'
WHERE slug = 'charlotte-mason';

UPDATE method_definitions SET
    baseline_subjects = ARRAY['reading','mathematics','writing','science','history','art','latin','logic','rhetoric','grammar'],
    transition_ages   = '[{"stage":"Logic","age":10},{"stage":"Rhetoric","age":14}]'
WHERE slug = 'classical';

UPDATE method_definitions SET
    baseline_subjects = ARRAY['reading','mathematics','writing','science','history','art'],
    transition_ages   = '[]'
WHERE slug = 'unschooling';

UPDATE method_definitions SET
    baseline_subjects = ARRAY['reading','mathematics','writing','science','history','art','practical_life','sensorial','geography'],
    transition_ages   = '[{"stage":"2nd Plane","age":6},{"stage":"3rd Plane","age":12}]'
WHERE slug = 'montessori';

UPDATE method_definitions SET
    baseline_subjects = ARRAY['reading','mathematics','writing','science','history','art','handwork','music','eurythmy','painting'],
    transition_ages   = '[{"stage":"Grade School","age":7},{"stage":"High School","age":14}]'
WHERE slug = 'waldorf';

UPDATE method_definitions SET
    baseline_subjects = ARRAY['reading','mathematics','writing','science','history','art'],
    transition_ages   = '[]'
WHERE slug IN ('traditional', 'eclectic');

-- +goose Down
ALTER TABLE method_definitions
    DROP COLUMN IF EXISTS baseline_subjects,
    DROP COLUMN IF EXISTS transition_ages;
