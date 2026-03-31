-- +goose Up
-- Migration: 20260322000001_create_iam_tables.sql
-- IAM domain: families, parents, students, COPPA consent, student sessions.
-- Phase 2 tables (co-parent invites, student sessions) included for schema stability.

-- COPPA consent status enum [S§17.2, ARCH §6.3]
-- StatementBegin/End prevents goose from splitting the DO block on internal semicolons
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE iam_coppa_consent_enum AS ENUM (
        'registered',  -- Account created, no COPPA notice shown yet
        'noticed',     -- Parent has acknowledged COPPA notice
        'consented',   -- Parent has provided verifiable consent
        're_verified', -- Parent has re-verified consent
        'withdrawn'    -- Parent has withdrawn consent; student data must be deleted
    );
EXCEPTION WHEN duplicate_object THEN null;
END $$;
-- +goose StatementEnd

-- Co-parent invite status enum [S§3.4]
-- +goose StatementBegin
DO $$ BEGIN
    CREATE TYPE iam_invite_status_enum AS ENUM (
        'pending',   -- Invite sent, awaiting acceptance
        'accepted',  -- Invite accepted, co-parent joined family
        'expired',   -- Invite expired (72-hour window)
        'cancelled'  -- Primary parent cancelled the invite
    );
EXCEPTION WHEN duplicate_object THEN null;
END $$;
-- +goose StatementEnd

-- Top-level family entity [S§3.1.1]
CREATE TABLE iam_families (
    id                          UUID PRIMARY KEY DEFAULT uuidv7(),
    display_name                TEXT NOT NULL,                              -- [S§6.2]
    state_code                  CHAR(2),                                   -- for compliance [S§6.2]
    location_region             TEXT,                                      -- coarse location [S§7.8]
    location_point              GEOMETRY(Point, 4326),                     -- PostGIS centroid [ARCH §5.4]
    primary_parent_id           UUID,                                      -- set after first parent created
    primary_methodology_slug    TEXT NOT NULL,                             -- FK to method_definitions [S§4.3]
    secondary_methodology_slugs TEXT[] NOT NULL DEFAULT '{}',              -- [S§4.3]
    subscription_tier           TEXT NOT NULL DEFAULT 'free'
                                CHECK (subscription_tier IN ('free', 'premium')),  -- [S§15.2]
    coppa_consent_status        iam_coppa_consent_enum NOT NULL DEFAULT 'registered', -- [S§17.2, ARCH §6.3]
    coppa_consented_at          TIMESTAMPTZ,                               -- when consent was granted
    coppa_consent_method        TEXT,                                      -- e.g. 'credit_card_verification'
    deletion_requested_at       TIMESTAMPTZ,                               -- NULL if no deletion pending [S§16.3]
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- NOTE: FK to method_definitions(slug) deferred until method:: migration runs.
-- Phase 1 uses 'charlotte-mason' as the application-level default slug.

CREATE INDEX idx_iam_families_subscription_tier ON iam_families(subscription_tier);
CREATE INDEX idx_iam_families_location ON iam_families USING GIST(location_point)
    WHERE location_point IS NOT NULL;
CREATE INDEX idx_iam_families_deletion ON iam_families(deletion_requested_at)
    WHERE deletion_requested_at IS NOT NULL;

-- Parent users [S§3.1.2]
CREATE TABLE iam_parents (
    id                 UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id          UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    kratos_identity_id UUID NOT NULL UNIQUE,                  -- links to Ory Kratos identity
    display_name       TEXT NOT NULL,                          -- [S§6.2]
    email              TEXT NOT NULL,                          -- synced from Kratos traits
    is_primary         BOOLEAN NOT NULL DEFAULT false,         -- [S§3.1.1]
    is_platform_admin  BOOLEAN NOT NULL DEFAULT false,         -- [S§3.1.5, 11-safety §9]
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_iam_parents_family ON iam_parents(family_id);
CREATE INDEX idx_iam_parents_kratos ON iam_parents(kratos_identity_id);

-- Student profiles [S§3.1.3]
-- Students do NOT have credentials — they are parent-mediated [S§3.3]
CREATE TABLE iam_students (
    id                       UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id                UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    display_name             TEXT NOT NULL,                    -- [S§3.1.3]
    birth_year               SMALLINT,                        -- [S§3.1.3]
    grade_level              TEXT,                             -- optional [S§3.1.3]
    methodology_override_slug TEXT,                           -- FK to method_definitions(slug) [S§4.6]
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_iam_students_family ON iam_students(family_id);

-- Co-parent invitations [S§3.4] — Phase 2, schema included for stability
CREATE TABLE iam_co_parent_invites (
    id         UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id  UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    invited_by UUID NOT NULL REFERENCES iam_parents(id),   -- always the primary parent
    email      TEXT NOT NULL,                               -- invitee's email
    token      TEXT NOT NULL UNIQUE,                        -- secure random token (pgcrypto)
    status     iam_invite_status_enum NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,                        -- 72 hours after creation
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_iam_invites_family ON iam_co_parent_invites(family_id);
CREATE INDEX idx_iam_invites_token ON iam_co_parent_invites(token) WHERE status = 'pending';
CREATE INDEX idx_iam_invites_email ON iam_co_parent_invites(email);

-- COPPA consent audit log [S§17.2]
-- Separate from iam_families to maintain immutable audit trail.
CREATE TABLE iam_coppa_audit_log (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    action          TEXT NOT NULL,                             -- 'consent_granted', 'consent_noticed', etc.
    method          TEXT,                                      -- 'credit_card_verification', etc.
    previous_status iam_coppa_consent_enum NOT NULL,
    new_status      iam_coppa_consent_enum NOT NULL,
    performed_by    UUID NOT NULL REFERENCES iam_parents(id),
    ip_hash         TEXT,                                      -- hashed IP, NOT raw IP [CODING §5.2]
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_iam_coppa_audit_family ON iam_coppa_audit_log(family_id);

-- Supervised student sessions [S§8.6.1] — Phase 2, schema included for stability
-- Parent-initiated sessions that grant students limited platform access.
-- Students do NOT have independent credentials — sessions are created by parents.
CREATE TABLE iam_student_sessions (
    id          UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id   UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id  UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    created_by  UUID NOT NULL REFERENCES iam_parents(id),
    token_hash  TEXT NOT NULL UNIQUE,            -- bcrypt hash of session token
    expires_at  TIMESTAMPTZ NOT NULL,
    is_active   BOOLEAN NOT NULL DEFAULT true,
    permissions JSONB NOT NULL DEFAULT '[]',     -- allowed tool slugs
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_iam_student_sessions_family ON iam_student_sessions(family_id);
CREATE INDEX idx_iam_student_sessions_student ON iam_student_sessions(student_id);
CREATE INDEX idx_iam_student_sessions_active ON iam_student_sessions(is_active, expires_at)
    WHERE is_active = true;

-- ─── Row-Level Security ───────────────────────────────────────────────────────
-- RLS provides defense-in-depth. Even if application code omits family_id filter,
-- the database rejects cross-family access. [ARCH §2.5, S§16.2]
--
-- The Go API sets `SET LOCAL app.current_family_id = '<uuid>'` at the start of
-- each scoped transaction via shared.ScopedTransaction.
--
-- Unscoped operations (auth middleware lookups, registration webhooks) bypass RLS
-- Family scoping is enforced at the GORM level via ScopedTransaction (ADR-008).
-- PostgreSQL RLS is NOT used. See specs/ARCHITECTURE.md for rationale.

-- +goose Down

-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS iam_student_sessions;
DROP TABLE IF EXISTS iam_coppa_audit_log;
DROP TABLE IF EXISTS iam_co_parent_invites;
DROP TABLE IF EXISTS iam_students;
DROP TABLE IF EXISTS iam_parents;
DROP TABLE IF EXISTS iam_families;

-- Drop enums
DROP TYPE IF EXISTS iam_invite_status_enum;
DROP TYPE IF EXISTS iam_coppa_consent_enum;
