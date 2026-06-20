-- +goose Up

-- Enable RLS on iam_students and add family-scoped policy.
-- The empty-string branch allows BypassRLSTransaction callers (which don't set
-- the variable) to read all rows; ScopedTransaction callers must set it.
ALTER TABLE iam_students ENABLE ROW LEVEL SECURITY;
-- FORCE ensures the policy applies to superusers and the table owner as well,
-- closing the bypass that would otherwise exist for the postgres test user.
ALTER TABLE iam_students FORCE ROW LEVEL SECURITY;

CREATE POLICY iam_students_family_rls ON iam_students
    USING (
        current_setting('app.current_family_id', true) = ''
        OR family_id = current_setting('app.current_family_id', true)::UUID
    );

-- Reconcile learner_profiles to use the canonical variable name (app.current_family_id).
-- The original migration used app.family_id; unify here without editing that file.
DROP POLICY IF EXISTS learner_profiles_family_rls ON learner_profiles;
ALTER TABLE learner_profiles FORCE ROW LEVEL SECURITY;
CREATE POLICY learner_profiles_family_rls ON learner_profiles
    USING (
        current_setting('app.current_family_id', true) = ''
        OR family_id = current_setting('app.current_family_id', true)::UUID
    );

-- +goose Down

DROP POLICY IF EXISTS iam_students_family_rls ON iam_students;
ALTER TABLE iam_students NO FORCE ROW LEVEL SECURITY;
ALTER TABLE iam_students DISABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS learner_profiles_family_rls ON learner_profiles;
ALTER TABLE learner_profiles NO FORCE ROW LEVEL SECURITY;
CREATE POLICY learner_profiles_family_rls ON learner_profiles
    USING (family_id = current_setting('app.family_id')::UUID);
