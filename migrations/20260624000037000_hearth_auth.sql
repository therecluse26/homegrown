-- +goose Up

-- Add Hearth user/org IDs to existing tables [ARCH ADR-018]
ALTER TABLE iam_families
    ADD COLUMN IF NOT EXISTS hearth_org_id UUID UNIQUE;

ALTER TABLE iam_parents
    ADD COLUMN IF NOT EXISTS hearth_user_id UUID UNIQUE;

-- Indexes for auth middleware lookups (FindByHearthUserID, NewFamilyScopeFromClaims)
CREATE INDEX IF NOT EXISTS idx_iam_parents_hearth_user ON iam_parents(hearth_user_id)
    WHERE hearth_user_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_iam_families_hearth_org ON iam_families(hearth_org_id)
    WHERE hearth_org_id IS NOT NULL;

-- Drop Kratos identity linkage — clean cutover, zero production users at migration time. [HOM-166 WS4]
-- PostgreSQL drops the implicit UNIQUE constraint index automatically when the column is dropped,
-- but the named index must be removed first.
DROP INDEX IF EXISTS idx_iam_parents_kratos;
ALTER TABLE iam_parents DROP COLUMN IF EXISTS kratos_identity_id;

-- BFF session store [ARCH ADR-020]
-- access_token and refresh_token are encrypted at rest (AES-256-GCM). [CODING §5.2]
CREATE TABLE IF NOT EXISTS iam_sessions (
    sid              TEXT PRIMARY KEY,               -- opaque random ID (32 crypto bytes, base64url)
    hearth_user_id   UUID NOT NULL,                  -- links to iam_parents.hearth_user_id
    family_id        UUID NOT NULL,                  -- denormalised for fast revocation
    access_token     TEXT NOT NULL,                  -- AES-256-GCM encrypted; NEVER logged
    refresh_token    TEXT NOT NULL,                  -- AES-256-GCM encrypted; NEVER logged
    token_expires_at TIMESTAMPTZ NOT NULL,           -- access token expiry (cleartext for middleware check)
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_iam_sessions_hearth_user ON iam_sessions(hearth_user_id);
CREATE INDEX IF NOT EXISTS idx_iam_sessions_family ON iam_sessions(family_id);
CREATE INDEX IF NOT EXISTS idx_iam_sessions_expires ON iam_sessions(token_expires_at);

-- +goose Down

DROP TABLE IF EXISTS iam_sessions;

DROP INDEX IF EXISTS idx_iam_families_hearth_org;
DROP INDEX IF EXISTS idx_iam_parents_hearth_user;

ALTER TABLE iam_parents ADD COLUMN IF NOT EXISTS kratos_identity_id UUID UNIQUE;
CREATE INDEX IF NOT EXISTS idx_iam_parents_kratos ON iam_parents(kratos_identity_id);

ALTER TABLE iam_parents DROP COLUMN IF EXISTS hearth_user_id;
ALTER TABLE iam_families DROP COLUMN IF EXISTS hearth_org_id;
