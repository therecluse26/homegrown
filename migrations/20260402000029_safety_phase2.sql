-- +goose Up
-- Safety Phase 2: parental controls, admin roles, grooming detection scores
-- [11-safety §14]

-- Parental controls: per-family configurable content filtering settings
CREATE TABLE IF NOT EXISTS safety_parental_controls (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    control_type    TEXT NOT NULL CHECK (control_type IN (
        'restrict_direct_messages',
        'age_filter',
        'content_category_block',
        'search_safe_mode',
        'marketplace_purchases'
    )),
    enabled         BOOLEAN NOT NULL DEFAULT true,
    settings        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(family_id, control_type)
);
CREATE INDEX IF NOT EXISTS idx_safety_parental_controls_family ON safety_parental_controls(family_id);

-- Admin roles: granular permission roles for platform administrators
CREATE TABLE IF NOT EXISTS safety_admin_roles (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Admin role assignments: which parents have which admin roles
CREATE TABLE IF NOT EXISTS safety_admin_role_assignments (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    parent_id   UUID NOT NULL REFERENCES iam_parents(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES safety_admin_roles(id) ON DELETE CASCADE,
    granted_by  UUID REFERENCES iam_parents(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(parent_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_safety_admin_role_assignments_parent ON safety_admin_role_assignments(parent_id);

-- Grooming detection scores: ML-based text analysis results
CREATE TABLE IF NOT EXISTS safety_grooming_scores (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    content_type    TEXT NOT NULL CHECK (content_type IN ('message', 'post', 'comment', 'review')),
    content_id      UUID NOT NULL,
    author_family_id UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    score           NUMERIC(5,4) NOT NULL,
    model_version   TEXT NOT NULL,
    flagged         BOOLEAN NOT NULL DEFAULT false,
    reviewed        BOOLEAN NOT NULL DEFAULT false,
    reviewed_by     UUID REFERENCES iam_parents(id),
    reviewed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_safety_grooming_scores_flagged ON safety_grooming_scores(flagged) WHERE flagged = true AND reviewed = false;

-- Seed default admin roles
INSERT INTO safety_admin_roles (id, name, description, permissions) VALUES
    (uuidv7(), 'platform_admin', 'Full platform administrator', ARRAY['safety:*', 'billing:*', 'content:*', 'users:*']),
    (uuidv7(), 'moderator', 'Content moderation only', ARRAY['safety:reports', 'safety:flags', 'safety:actions', 'safety:appeals']),
    (uuidv7(), 'billing_support', 'Billing and subscription support', ARRAY['billing:view', 'billing:refund', 'billing:subscription']),
    (uuidv7(), 'creator_support', 'Creator and marketplace support', ARRAY['content:listings', 'content:reviews', 'content:publishers'])
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS safety_grooming_scores;
DROP TABLE IF EXISTS safety_admin_role_assignments;
DROP TABLE IF EXISTS safety_admin_roles;
DROP TABLE IF EXISTS safety_parental_controls;
