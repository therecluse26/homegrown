# Domain Spec 01 — Identity & Access Management (IAM)

## §1 Overview

The IAM domain owns **identity data** and **authorization logic** for the entire platform. It
does not own authentication — that is delegated to Ory Kratos, which runs as a sidecar and
manages credentials, sessions, MFA, and OAuth flows. IAM owns what happens *after*
authentication: mapping Kratos identities to families, enforcing family-scoped access,
managing COPPA consent, and providing the `AuthContext` and `FamilyScope` types that every
other domain depends on.

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/iam/` |
| **DB prefix** | `iam_` |
| **Complexity class** | Non-complex (no `domain/` subdirectory) `[ARCH §4.5]` |
| **External adapter** | `src/iam/adapters/kratos.rs` |
| **Key constraint** | Auth delegated to Ory Kratos; IAM owns identity *data* and *authorization* |

**What IAM owns**: Family accounts, parent users, student profiles, COPPA consent tracking,
co-parent invitations, Kratos webhook handlers, auth middleware *implementation* (calls
`KratosAdapter`), `AuthContext` *population* logic.

**Shared infrastructure** (defined in 00-core, consumed by IAM and all other domains):
`AuthContext` type (00-core §7.2), `FamilyScope` type (00-core §8), `AppError` base variants
(00-core §6), role extractors `RequirePremium`, `RequireCreator`, `RequireCoppaConsent`,
`RequirePrimaryParent` (00-core §13.3).

**What IAM delegates**: Credential management, session storage, MFA, OAuth flows → Ory Kratos.
Social profile creation → `social::` (via `FamilyCreated` event). Methodology validation →
`method::` service. Email delivery → `notify::` (via domain events).

---

## §2 Requirements Traceability

Every requirement from SPEC.md that touches IAM is mapped to the section of this document
that addresses it.

| Requirement | Source | Addressed In |
|-------------|--------|--------------|
| Family Account is the top-level entity | `[S§3.1.1]` | §3, §10 |
| Every family MUST have at least one Parent User | `[S§3.1.1]` | §3, §10 |
| Family MUST have a designated primary parent | `[S§3.1.1]` | §3, §5, §10 |
| Co-parents can be added | `[S§3.1.1]` | §3, §4, §10 |
| Parent Users authenticate with own credentials | `[S§3.1.2]` | §7 (Kratos) |
| All parents have equal access to family data | `[S§3.1.2]` | §6, §11 |
| Student Profiles have no independent credentials | `[S§3.1.3]` | §3 |
| Student Profile associated with age, grade, name | `[S§3.1.3]` | §3, §8 |
| Student methodology overrides | `[S§3.1.3, S§4.6]` | §3 |
| Permission matrix (free vs. premium, roles) | `[S§3.2]` | §11 |
| Parent-mediated access model | `[S§3.3]` | §11 |
| Family account lifecycle (create, delete, co-parent) | `[S§3.4]` | §10 |
| Multi-child family support | `[S§3.5]` | §3, §4 |
| Account creation atomicity (parent + family) | `[S§6.1]` | §4 (W1), §10 |
| COPPA notice during account creation | `[S§6.1]` | §9 |
| Email verification required | `[S§6.1]` | §7 (Kratos) |
| Co-parent addition during/after setup | `[S§6.2]` | §4, §10 |
| Data lifecycle (creation, deletion, export) | `[S§16.2-16.3]` | §10 |
| COPPA compliance checklist | `[S§17.2]` | §9 |
| MFA support | `[S§17.1]` | §7 (Kratos) |
| Session management and remote revocation | `[S§17.1]` | §7 |
| Family-scoped data isolation | `[S§16.2]` | §3 (RLS), §6, §11 |
| Coarse location only (no GPS coordinates) | `[S§7.8]` | §3 |

---

## §3 Database Schema

All tables use the `iam_` prefix. Columns include spec references as comments. RLS policies
provide defense-in-depth for family-scoped isolation. `[ARCH §5.2]`

### §3.1 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_iam_tables.rs
-- =============================================================================

-- PostgreSQL extensions (uuid-ossp, pgcrypto, postgis, pg_trgm) are installed
-- by the bootstrap migration (00-core §9.4). They are available to all domains.

-- COPPA consent status enum [S§17.2, ARCH §6.3]
CREATE TYPE iam_coppa_consent_enum AS ENUM (
    'registered',   -- Account created, no COPPA notice shown yet
    'noticed',      -- Parent has acknowledged COPPA notice
    'consented',    -- Parent has provided verifiable consent
    're_verified',  -- Parent has re-verified consent
    'withdrawn'     -- Parent has withdrawn consent; student data must be deleted
);

-- Co-parent invite status enum [S§3.4]
CREATE TYPE iam_invite_status_enum AS ENUM (
    'pending',      -- Invite sent, awaiting acceptance
    'accepted',     -- Invite accepted, co-parent joined family
    'expired',      -- Invite expired (72-hour window)
    'cancelled'     -- Primary parent cancelled the invite
);

-- Top-level family entity [S§3.1.1]
CREATE TABLE iam_families (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name              TEXT NOT NULL,                              -- [S§6.2]
    state_code                CHAR(2),                                   -- for compliance [S§6.2]
    location_region           TEXT,                                      -- coarse location [S§7.8]
    location_point            GEOMETRY(Point, 4326),                     -- PostGIS centroid [ARCH §5.4]
    primary_parent_id         UUID,                                      -- set after first parent created
    primary_methodology_id    UUID NOT NULL,                             -- FK to method_definitions [S§4.3]
    secondary_methodology_ids UUID[] NOT NULL DEFAULT '{}',              -- array of method IDs [S§4.3]
    subscription_tier         TEXT NOT NULL DEFAULT 'free'
                              CHECK (subscription_tier IN ('free', 'premium')),  -- [S§15.2]
    coppa_consent_status      iam_coppa_consent_enum NOT NULL DEFAULT 'registered', -- [S§17.2, ARCH §6.3]
    coppa_consented_at        TIMESTAMPTZ,                               -- when consent was granted
    coppa_consent_method      TEXT,                                      -- e.g., 'credit_card_verification'
    deletion_requested_at     TIMESTAMPTZ,                               -- NULL if no deletion pending [S§16.3]
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- NOTE: FK to method_definitions deferred until method:: migration runs.
-- Phase 1 uses a CHECK constraint or application-level validation instead.
-- The FK will be added via ALTER TABLE in the methodology migration.

CREATE INDEX idx_iam_families_subscription_tier ON iam_families(subscription_tier);
CREATE INDEX idx_iam_families_location ON iam_families USING GIST(location_point)
    WHERE location_point IS NOT NULL;
CREATE INDEX idx_iam_families_deletion ON iam_families(deletion_requested_at)
    WHERE deletion_requested_at IS NOT NULL;

-- Parent users [S§3.1.2]
CREATE TABLE iam_parents (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id                UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    display_name             TEXT NOT NULL,                    -- [S§3.1.3]
    birth_year               SMALLINT,                        -- [S§3.1.3]
    grade_level              TEXT,                             -- optional [S§3.1.3]
    methodology_override_id  UUID,                            -- FK to method_definitions [S§4.6]
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_iam_students_family ON iam_students(family_id);

-- Co-parent invitations [S§3.4]
CREATE TABLE iam_co_parent_invites (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id     UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    invited_by    UUID NOT NULL REFERENCES iam_parents(id),   -- always the primary parent
    email         TEXT NOT NULL,                               -- invitee's email
    token         TEXT NOT NULL UNIQUE,                        -- secure random token (pgcrypto)
    status        iam_invite_status_enum NOT NULL DEFAULT 'pending',
    expires_at    TIMESTAMPTZ NOT NULL,                        -- 72 hours after creation
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_iam_invites_family ON iam_co_parent_invites(family_id);
CREATE INDEX idx_iam_invites_token ON iam_co_parent_invites(token) WHERE status = 'pending';
CREATE INDEX idx_iam_invites_email ON iam_co_parent_invites(email);

-- COPPA consent audit log [S§17.2]
-- Separate from iam_families to maintain immutable audit trail
CREATE TABLE iam_coppa_audit_log (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id      UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    action         TEXT NOT NULL,                             -- 'consent_granted', 'consent_withdrawn', etc.
    method         TEXT,                                      -- 'credit_card_verification', etc.
    previous_status iam_coppa_consent_enum NOT NULL,
    new_status     iam_coppa_consent_enum NOT NULL,
    performed_by   UUID NOT NULL REFERENCES iam_parents(id),
    ip_hash        TEXT,                                      -- hashed IP, NOT raw IP [CODING §5.2]
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_iam_coppa_audit_family ON iam_coppa_audit_log(family_id);

-- Supervised student sessions [S§8.6.1]
-- Parent-initiated sessions that grant students limited platform access.
-- Students do NOT have independent credentials — sessions are created by parents.
CREATE TABLE iam_student_sessions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    student_id            UUID NOT NULL REFERENCES iam_students(id) ON DELETE CASCADE,
    created_by            UUID NOT NULL REFERENCES iam_parents(id),
    token_hash            TEXT NOT NULL UNIQUE,            -- bcrypt hash of session token
    expires_at            TIMESTAMPTZ NOT NULL,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    permissions           JSONB NOT NULL DEFAULT '[]',     -- allowed tool slugs
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_iam_student_sessions_family ON iam_student_sessions(family_id);
CREATE INDEX idx_iam_student_sessions_student ON iam_student_sessions(student_id);
CREATE INDEX idx_iam_student_sessions_active ON iam_student_sessions(is_active, expires_at)
    WHERE is_active = true;
```

### §3.2 Row-Level Security Policies

RLS provides defense-in-depth. Even if application code has a bug that omits the `family_id`
filter, the database rejects cross-family access. `[ARCH §2.5, S§16.2]`

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000002_create_iam_rls_policies.rs
-- =============================================================================

-- Enable RLS on all IAM tables with user data
ALTER TABLE iam_families ENABLE ROW LEVEL SECURITY;
ALTER TABLE iam_parents ENABLE ROW LEVEL SECURITY;
ALTER TABLE iam_students ENABLE ROW LEVEL SECURITY;
ALTER TABLE iam_co_parent_invites ENABLE ROW LEVEL SECURITY;
ALTER TABLE iam_coppa_audit_log ENABLE ROW LEVEL SECURITY;
ALTER TABLE iam_student_sessions ENABLE ROW LEVEL SECURITY;

-- Application role used by the Rust API (not the superuser)
-- The app sets `SET LOCAL app.current_family_id = '<uuid>'` per transaction
CREATE POLICY iam_families_isolation ON iam_families
    USING (id = current_setting('app.current_family_id', true)::uuid);

CREATE POLICY iam_parents_isolation ON iam_parents
    USING (family_id = current_setting('app.current_family_id', true)::uuid);

CREATE POLICY iam_students_isolation ON iam_students
    USING (family_id = current_setting('app.current_family_id', true)::uuid);

CREATE POLICY iam_invites_isolation ON iam_co_parent_invites
    USING (family_id = current_setting('app.current_family_id', true)::uuid);

CREATE POLICY iam_coppa_audit_isolation ON iam_coppa_audit_log
    USING (family_id = current_setting('app.current_family_id', true)::uuid);

CREATE POLICY iam_student_sessions_family_policy ON iam_student_sessions
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- The superuser/migration role bypasses RLS (PostgreSQL default).
-- The application role MUST NOT be a superuser.
```

**RLS implementation note**: The Rust API sets `SET LOCAL app.current_family_id` at the start
of each database transaction via a helper in `src/shared/db.rs`. This ensures RLS is active
for the lifetime of the transaction and automatically cleared on commit/rollback. Queries
that intentionally bypass family scope (e.g., `find_parent_by_kratos_id` in auth middleware)
run under the superuser role or with RLS temporarily disabled via `SET LOCAL` — these cases
MUST be documented and reviewed.

---

## §4 API Endpoints

All endpoints return JSON. Authentication is via Kratos session cookie validated by auth
middleware. Error responses follow `AppError` → HTTP status mapping (§12).

### §4.1 Phase 1 Endpoints

| # | Method | Path | Auth | Description | Status Codes |
|---|--------|------|------|-------------|-------------|
| 1 | `GET` | `/v1/auth/me` | Required | Current authenticated user info | 200, 401 |
| 2 | `GET` | `/v1/families/profile` | Required | Get family profile | 200, 401 |
| 3 | `PATCH` | `/v1/families/profile` | Required | Update family profile | 200, 401, 422 |
| 4 | `POST` | `/v1/families/students` | Required + COPPA | Create student profile | 201, 401, 403, 422 |
| 5 | `GET` | `/v1/families/students` | Required | List family's students | 200, 401 |
| 6 | `PATCH` | `/v1/families/students/:id` | Required | Update student profile | 200, 401, 404, 422 |
| 7 | `DELETE` | `/v1/families/students/:id` | Required | Remove student profile | 204, 401, 404 |
| 8 | `POST` | `/v1/families/consent` | Required | Submit COPPA consent | 200, 401, 422 |
| 9 | `GET` | `/v1/families/consent` | Required | Get consent status | 200, 401 |
| W1 | `POST` | `/hooks/kratos/post-registration` | Webhook secret | Post-registration: create family + parent | 200, 400, 500 |
| W2 | `POST` | `/hooks/kratos/post-login` | Webhook secret | Post-login: update last_login, sync traits | 200, 400 |

**Phase 1 total**: 9 public endpoints + 2 webhooks = 11 routes.

### §4.2 Phase 2 Endpoints

| # | Method | Path | Auth | Description | Status Codes |
|---|--------|------|------|-------------|-------------|
| 10 | `POST` | `/v1/families/invites` | Required + Primary | Invite co-parent | 201, 401, 403, 422 |
| 11 | `DELETE` | `/v1/families/invites/:id` | Required + Primary | Cancel invite | 204, 401, 403, 404 |
| 12 | `POST` | `/v1/families/invites/:token/accept` | Required | Accept co-parent invite | 200, 400, 404, 410 |
| 13 | `DELETE` | `/v1/families/parents/:id` | Required + Primary | Remove co-parent | 204, 401, 403, 404 |
| 14 | `POST` | `/v1/families/primary-parent` | Required + Primary | Transfer primary role | 200, 401, 403, 422 |
| 15 | `DELETE` | `/v1/families/consent` | Required + Primary | Withdraw COPPA consent | 200, 401, 403 |
| 16 | `POST` | `/v1/families/deletion-request` | Required + Primary | Request family deletion | 200, 401, 403 |
| 17 | `DELETE` | `/v1/families/deletion-request` | Required + Primary | Cancel deletion request | 200, 401, 403, 404 |

### §4.3 Endpoint Details

#### `GET /v1/auth/me`

Returns the currently authenticated user's info. Used by the frontend on every page load to
establish identity and permissions.

- **Extractors**: `AuthContext`
- **FamilyScope**: No (reads from AuthContext directly)
- **Response**: `CurrentUserResponse`

#### `GET /v1/families/profile`

Returns the authenticated user's family profile including methodology selection,
subscription tier, and COPPA status.

- **Extractors**: `AuthContext`, `FamilyScope`
- **Response**: `FamilyProfileResponse`

#### `PATCH /v1/families/profile`

Updates family display name, state code, or location region. Does NOT update methodology
(that goes through `method::` endpoints) or subscription tier (that goes through `billing::`).

- **Extractors**: `AuthContext`, `FamilyScope`, `Json<UpdateFamilyCommand>`
- **Validation**: `display_name` length 1-100, `state_code` valid US state
- **Response**: `FamilyProfileResponse`

#### `POST /v1/families/students`

Creates a new student profile. Blocked if COPPA consent is not `Consented` or `ReVerified`.

- **Extractors**: `AuthContext`, `FamilyScope`, `RequireCoppaConsent`, `Json<CreateStudentCommand>`
- **Validation**: `display_name` required, `birth_year` if present must be reasonable
- **Response**: `StudentResponse` (201 Created)
- **Events**: `StudentCreated`

#### `GET /v1/families/students`

Lists all students in the authenticated user's family.

- **Extractors**: `AuthContext`, `FamilyScope`
- **Response**: `Vec<StudentResponse>`

#### `PATCH /v1/families/students/:id`

Updates a student's display name, birth year, grade level, or methodology override.

- **Extractors**: `AuthContext`, `FamilyScope`, `Path<StudentId>`, `Json<UpdateStudentCommand>`
- **Response**: `StudentResponse`

#### `DELETE /v1/families/students/:id`

Removes a student profile. In Phase 1, deletion is immediate. Phase 2 adds data export offer
before deletion.

- **Extractors**: `AuthContext`, `FamilyScope`, `Path<StudentId>`
- **Response**: 204 No Content
- **Events**: `StudentDeleted`

#### `POST /v1/families/consent`

Submits COPPA parental consent. Phase 1 uses credit card micro-charge verification via Stripe.

- **Extractors**: `AuthContext`, `FamilyScope`, `Json<CoppaConsentCommand>`
- **Transitions**: `Registered → Noticed → Consented`, or `Noticed → Consented`
- **Response**: `ConsentStatusResponse`
- **Events**: `CoppaConsentGranted`
- **Audit**: Creates `iam_coppa_audit_log` entry

#### `GET /v1/families/consent`

Returns current COPPA consent status and history.

- **Extractors**: `AuthContext`, `FamilyScope`
- **Response**: `ConsentStatusResponse`

#### `POST /hooks/kratos/post-registration` (Webhook)

Called by Kratos after successful registration. Creates family + parent atomically.

- **Auth**: Webhook shared secret (not Kratos session)
- **Body**: `KratosWebhookPayload` (identity ID, traits: name, email)
- **Transaction**: Atomic — family, parent, and primary_parent_id set in one transaction
- **Events**: `FamilyCreated`

#### `POST /hooks/kratos/post-login` (Webhook)

Called by Kratos after successful login. Syncs Kratos traits (email, name) to local DB.

- **Auth**: Webhook shared secret
- **Body**: `KratosWebhookPayload`

#### Student Sessions `[S§8.6.1]`

##### `POST /v1/families/students/:student_id/sessions`

Creates a supervised student session. Parent auth required.

- **Auth**: Required (`FamilyScope`, parent only)
- **Body**: `CreateStudentSessionCommand` (`expires_in_hours?`: default 2, `allowed_tool_slugs?`: defaults to student's resolved tool set)
- **Validation**: Student must belong to family; student's `birth_year` must indicate age 10+; parent must have COPPA consent
- **Response**: `StudentSessionResponse` (201 Created) — includes the session token (only returned once, on creation)

##### `GET /v1/families/students/:student_id/sessions`

Lists active sessions for a student. Parent auth required.

- **Auth**: Required (`FamilyScope`, parent only)
- **Response**: `Vec<StudentSessionSummaryResponse>` (200 OK)

##### `DELETE /v1/families/students/:student_id/sessions/:id`

Revokes a student session immediately. Parent auth required.

- **Auth**: Required (`FamilyScope`, parent only)
- **Response**: 204 No Content

##### `GET /v1/student-session/me`

Returns the current student session's identity and permissions. Used by the student view frontend.

- **Auth**: Student session token (in cookie or Authorization header)
- **Response**: `StudentSessionIdentityResponse` (200 OK) — includes student_id, family_id, allowed_tool_slugs, expires_at

---

## §5 Service Interface

The `IamService` trait defines all use cases exposed to handlers and other domains.
Defined in `src/iam/ports.rs`. `[CODING §8.2]`

```rust
#[async_trait]
pub trait IamService: Send + Sync {
    // ─── Queries ───────────────────────────────────────────────────────

    /// Returns the current user's info (parent + family summary).
    /// Used by GET /v1/auth/me.
    async fn get_current_user(
        &self,
        auth: &AuthContext,
    ) -> Result<CurrentUserResponse, AppError>;

    /// Returns the family profile for the given scope.
    /// Used by GET /v1/families/profile.
    async fn get_family_profile(
        &self,
        scope: &FamilyScope,
    ) -> Result<FamilyProfileResponse, AppError>;

    /// Lists all students in the family.
    /// Used by GET /v1/families/students.
    async fn list_students(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<StudentResponse>, AppError>;

    /// Returns COPPA consent status and audit history.
    /// Used by GET /v1/families/consent.
    async fn get_consent_status(
        &self,
        scope: &FamilyScope,
    ) -> Result<ConsentStatusResponse, AppError>;

    // ─── Commands ──────────────────────────────────────────────────────

    /// Handles Kratos post-registration webhook.
    /// Creates family + parent atomically. Publishes FamilyCreated.
    async fn handle_post_registration(
        &self,
        payload: KratosWebhookPayload,
    ) -> Result<(), AppError>;

    /// Handles Kratos post-login webhook.
    /// Syncs traits (email, name) from Kratos to local DB.
    async fn handle_post_login(
        &self,
        payload: KratosWebhookPayload,
    ) -> Result<(), AppError>;

    /// Updates family profile fields (display_name, state_code, location_region).
    /// Does NOT handle methodology or subscription changes.
    async fn update_family_profile(
        &self,
        scope: &FamilyScope,
        cmd: UpdateFamilyCommand,
    ) -> Result<FamilyProfileResponse, AppError>;

    /// Creates a student profile. Requires COPPA consent (enforced by caller via extractor).
    /// Publishes StudentCreated event.
    async fn create_student(
        &self,
        scope: &FamilyScope,
        cmd: CreateStudentCommand,
    ) -> Result<StudentResponse, AppError>;

    /// Updates a student profile.
    async fn update_student(
        &self,
        scope: &FamilyScope,
        student_id: Uuid,
        cmd: UpdateStudentCommand,
    ) -> Result<StudentResponse, AppError>;

    /// Deletes a student profile. Publishes StudentDeleted event.
    async fn delete_student(
        &self,
        scope: &FamilyScope,
        student_id: Uuid,
    ) -> Result<(), AppError>;

    /// Submits COPPA parental consent. Validates consent method (e.g., credit card
    /// micro-charge). Publishes CoppaConsentGranted on success.
    async fn submit_coppa_consent(
        &self,
        scope: &FamilyScope,
        auth: &AuthContext,
        cmd: CoppaConsentCommand,
    ) -> Result<ConsentStatusResponse, AppError>;

    // ─── Phase 2 Commands ──────────────────────────────────────────────

    /// Invites a co-parent via email. Generates secure token with 72h expiry.
    /// Publishes event for notify:: to send invite email.
    async fn invite_co_parent(
        &self,
        scope: &FamilyScope,
        auth: &AuthContext,
        cmd: InviteCoParentCommand,
    ) -> Result<CoParentInviteResponse, AppError>;

    /// Cancels a pending co-parent invite.
    async fn cancel_invite(
        &self,
        scope: &FamilyScope,
        invite_id: Uuid,
    ) -> Result<(), AppError>;

    /// Accepts a co-parent invite. The accepting user's Kratos identity is linked
    /// to the inviting family. Returns updated family info.
    async fn accept_invite(
        &self,
        auth: &AuthContext,
        token: String,
    ) -> Result<FamilyProfileResponse, AppError>;

    /// Removes a co-parent from the family. Primary parent only.
    /// Revokes the removed parent's Kratos sessions.
    async fn remove_co_parent(
        &self,
        scope: &FamilyScope,
        parent_id: Uuid,
    ) -> Result<(), AppError>;

    /// Transfers the primary parent role to another parent in the family.
    async fn transfer_primary_parent(
        &self,
        scope: &FamilyScope,
        auth: &AuthContext,
        new_primary_id: Uuid,
    ) -> Result<(), AppError>;

    /// Withdraws COPPA consent. Triggers student data export + deletion.
    /// Publishes CoppaConsentWithdrawn.
    async fn withdraw_coppa_consent(
        &self,
        scope: &FamilyScope,
        auth: &AuthContext,
    ) -> Result<(), AppError>;

    /// Requests family account deletion. Starts grace period.
    /// Publishes FamilyDeletionScheduled.
    async fn request_family_deletion(
        &self,
        scope: &FamilyScope,
        auth: &AuthContext,
    ) -> Result<(), AppError>;

    /// Cancels a pending family deletion request.
    async fn cancel_family_deletion(
        &self,
        scope: &FamilyScope,
    ) -> Result<(), AppError>;
}
```

**Implementation**: `IamServiceImpl` in `src/iam/service.rs`. Constructor receives:
- `Arc<dyn FamilyRepository>`
- `Arc<dyn ParentRepository>`
- `Arc<dyn StudentRepository>`
- `Arc<dyn CoParentInviteRepository>` (Phase 2)
- `Arc<dyn KratosAdapter>`
- `Arc<EventBus>`

---

## §6 Repository Interfaces

Defined in `src/iam/ports.rs`. Each method documents its `FamilyScope` requirement.
`[CODING §2.4, §8.2]`

```rust
#[async_trait]
pub trait FamilyRepository: Send + Sync {
    /// Creates a new family. NOT family-scoped (family does not exist yet).
    async fn create(&self, cmd: CreateFamily) -> Result<Family, AppError>;

    /// Finds a family by ID. NOT family-scoped — used by auth middleware
    /// before FamilyScope is constructed.
    async fn find_by_id(&self, id: Uuid) -> Result<Option<Family>, AppError>;

    /// Updates family profile fields. Family-scoped.
    async fn update(
        &self,
        scope: &FamilyScope,
        cmd: UpdateFamily,
    ) -> Result<Family, AppError>;

    /// Sets the primary parent ID on the family. NOT family-scoped —
    /// used during registration before scope exists.
    async fn set_primary_parent(
        &self,
        family_id: Uuid,
        parent_id: Uuid,
    ) -> Result<(), AppError>;

    /// Sets the COPPA consent status. Family-scoped.
    async fn update_consent_status(
        &self,
        scope: &FamilyScope,
        status: CoppaConsentStatus,
        method: Option<String>,
    ) -> Result<Family, AppError>;

    /// Sets the methodology IDs on the family. Called by method:: service
    /// via IamService. Family-scoped.
    async fn set_methodology(
        &self,
        scope: &FamilyScope,
        primary_id: Uuid,
        secondary_ids: Vec<Uuid>,
    ) -> Result<(), AppError>;

    /// Sets deletion_requested_at. Family-scoped.
    async fn set_deletion_requested(
        &self,
        scope: &FamilyScope,
        requested_at: Option<DateTime<Utc>>,
    ) -> Result<(), AppError>;
}

#[async_trait]
pub trait ParentRepository: Send + Sync {
    /// Creates a new parent. NOT family-scoped (used during registration
    /// and co-parent invite acceptance).
    async fn create(&self, cmd: CreateParent) -> Result<Parent, AppError>;

    /// Finds a parent by Kratos identity ID. NOT family-scoped — used by
    /// auth middleware before scope is constructed. This is the lookup path
    /// for every authenticated request.
    async fn find_by_kratos_id(
        &self,
        kratos_identity_id: Uuid,
    ) -> Result<Option<Parent>, AppError>;

    /// Lists all parents in a family. Family-scoped.
    async fn list_by_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<Parent>, AppError>;

    /// Finds a specific parent by ID. Family-scoped.
    async fn find_by_id(
        &self,
        scope: &FamilyScope,
        parent_id: Uuid,
    ) -> Result<Option<Parent>, AppError>;

    /// Updates parent fields (display_name, email sync). Family-scoped.
    async fn update(
        &self,
        scope: &FamilyScope,
        parent_id: Uuid,
        cmd: UpdateParent,
    ) -> Result<Parent, AppError>;

    /// Removes a parent from the family. Family-scoped.
    async fn delete(
        &self,
        scope: &FamilyScope,
        parent_id: Uuid,
    ) -> Result<(), AppError>;

    /// Updates is_primary flag. Family-scoped.
    async fn set_primary(
        &self,
        scope: &FamilyScope,
        parent_id: Uuid,
        is_primary: bool,
    ) -> Result<(), AppError>;
}

#[async_trait]
pub trait StudentRepository: Send + Sync {
    /// Creates a student profile. Family-scoped.
    async fn create(
        &self,
        scope: &FamilyScope,
        cmd: CreateStudent,
    ) -> Result<Student, AppError>;

    /// Lists all students in the family. Family-scoped.
    async fn list_by_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<Student>, AppError>;

    /// Finds a specific student by ID. Family-scoped.
    async fn find_by_id(
        &self,
        scope: &FamilyScope,
        student_id: Uuid,
    ) -> Result<Option<Student>, AppError>;

    /// Updates a student profile. Family-scoped.
    async fn update(
        &self,
        scope: &FamilyScope,
        student_id: Uuid,
        cmd: UpdateStudent,
    ) -> Result<Student, AppError>;

    /// Deletes a student profile. Family-scoped.
    async fn delete(
        &self,
        scope: &FamilyScope,
        student_id: Uuid,
    ) -> Result<(), AppError>;
}

/// Phase 2
#[async_trait]
pub trait CoParentInviteRepository: Send + Sync {
    /// Creates a co-parent invite. Family-scoped.
    async fn create(
        &self,
        scope: &FamilyScope,
        cmd: CreateInvite,
    ) -> Result<CoParentInvite, AppError>;

    /// Finds an invite by its secure token. NOT family-scoped —
    /// the accepting user is not yet part of the family.
    async fn find_by_token(
        &self,
        token: &str,
    ) -> Result<Option<CoParentInvite>, AppError>;

    /// Finds an invite by ID. Family-scoped.
    async fn find_by_id(
        &self,
        scope: &FamilyScope,
        invite_id: Uuid,
    ) -> Result<Option<CoParentInvite>, AppError>;

    /// Updates invite status. Family-scoped.
    async fn update_status(
        &self,
        scope: &FamilyScope,
        invite_id: Uuid,
        status: InviteStatus,
    ) -> Result<(), AppError>;

    /// Lists pending invites for a family. Family-scoped.
    async fn list_pending(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<CoParentInvite>, AppError>;

    /// Expires all invites past their expiry time. NOT family-scoped —
    /// runs as a background cleanup job across all families.
    async fn expire_stale_invites(&self) -> Result<u64, AppError>;
}
```

**FamilyScope exception documentation**: Methods marked "NOT family-scoped" include a
comment explaining why. These exceptions are:

1. **`find_by_kratos_id`** — runs in auth middleware before FamilyScope is constructed
2. **`create` (family/parent)** — entity does not exist yet; no family to scope to
3. **`find_by_token` (invites)** — accepting user is not yet a family member
4. **`expire_stale_invites`** — batch cleanup job, crosses family boundaries by design
5. **`find_by_id` (family)** — used by auth middleware and webhook handlers

---

## §7 Kratos Adapter Interface

Defined in `src/iam/ports.rs`. The adapter wraps Kratos SDK calls and returns domain types
only. `[CODING §8.1, ARCH §4.2]`

```rust
#[async_trait]
pub trait KratosAdapter: Send + Sync {
    /// Validates a Kratos session cookie/token.
    /// Returns the Kratos identity ID if valid.
    async fn validate_session(
        &self,
        session_cookie: &str,
    ) -> Result<KratosSession, AppError>;

    /// Retrieves identity traits (email, name) from Kratos.
    async fn get_identity(
        &self,
        identity_id: Uuid,
    ) -> Result<KratosIdentity, AppError>;

    /// Deletes a Kratos identity (used during family deletion).
    async fn delete_identity(
        &self,
        identity_id: Uuid,
    ) -> Result<(), AppError>;

    /// Revokes all active sessions for an identity (used when removing a co-parent).
    async fn revoke_sessions(
        &self,
        identity_id: Uuid,
    ) -> Result<(), AppError>;
}

/// Domain types returned by KratosAdapter — NOT Kratos SDK types
pub struct KratosSession {
    pub identity_id: Uuid,
    pub active: bool,
    pub authenticated_at: DateTime<Utc>,
}

pub struct KratosIdentity {
    pub id: Uuid,
    pub email: String,
    pub name: String,
}
```

**Implementation**: `KratosAdapterImpl` in `src/iam/adapters/kratos.rs`. Uses the Kratos
Admin API (internal sidecar URL, not public). The adapter maps Kratos errors to `AppError`
variants — no Kratos SDK types leak beyond this file.

---

## §8 Models (DTOs)

All types defined in `src/iam/models.rs`. All derive `serde::Serialize`, `serde::Deserialize`,
and `utoipa::ToSchema`. Request types additionally derive `validator::Validate`. `[CODING §2.3]`

### §8.1 Request Types

```rust
/// POST /v1/families/students
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CreateStudentCommand {
    #[validate(length(min = 1, max = 100))]
    pub display_name: String,
    /// Four-digit year (e.g., 2018). Optional.
    #[validate(range(min = 2000, max = 2030))]
    pub birth_year: Option<i16>,
    /// Free-text grade level (e.g., "3rd", "PreK", "9th"). Optional.
    #[validate(length(max = 20))]
    pub grade_level: Option<String>,
    /// Override family methodology for this student. Optional. [S§4.6]
    pub methodology_override_id: Option<Uuid>,
}

/// PATCH /v1/families/students/:id
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct UpdateStudentCommand {
    #[validate(length(min = 1, max = 100))]
    pub display_name: Option<String>,
    #[validate(range(min = 2000, max = 2030))]
    pub birth_year: Option<i16>,
    #[validate(length(max = 20))]
    pub grade_level: Option<String>,
    pub methodology_override_id: Option<Option<Uuid>>,  // None = don't change, Some(None) = clear
}

/// PATCH /v1/families/profile
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct UpdateFamilyCommand {
    #[validate(length(min = 1, max = 100))]
    pub display_name: Option<String>,
    /// Two-letter US state code (e.g., "TX", "CA"). [S§6.2]
    #[validate(length(equal = 2))]
    pub state_code: Option<String>,
    /// Coarse location (city/region name). [S§7.8]
    #[validate(length(max = 200))]
    pub location_region: Option<String>,
}

/// POST /v1/families/consent
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct CoppaConsentCommand {
    /// The consent method being used (e.g., "credit_card_verification")
    pub method: String,
    /// Method-specific payload (e.g., Stripe payment method token for credit card verification)
    pub verification_token: String,
    /// Parent acknowledges COPPA notice (must be true to proceed)
    pub coppa_notice_acknowledged: bool,
}

/// POST /v1/families/invites (Phase 2)
#[derive(Debug, Deserialize, Validate, ToSchema)]
pub struct InviteCoParentCommand {
    #[validate(email)]
    pub email: String,
}
```

### §8.2 Response Types

```rust
/// GET /v1/auth/me
#[derive(Debug, Serialize, ToSchema)]
pub struct CurrentUserResponse {
    pub parent_id: Uuid,
    pub family_id: Uuid,
    pub display_name: String,
    pub email: String,
    pub is_primary_parent: bool,
    pub subscription_tier: String,       // "free" or "premium"
    pub coppa_consent_status: String,
    pub family_display_name: String,
}

/// GET /v1/families/profile, PATCH /v1/families/profile
#[derive(Debug, Serialize, ToSchema)]
pub struct FamilyProfileResponse {
    pub id: Uuid,
    pub display_name: String,
    pub state_code: Option<String>,
    pub location_region: Option<String>,
    pub primary_methodology_id: Uuid,
    pub secondary_methodology_ids: Vec<Uuid>,
    pub subscription_tier: String,
    pub coppa_consent_status: String,
    pub parents: Vec<ParentSummary>,
    pub student_count: usize,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ParentSummary {
    pub id: Uuid,
    pub display_name: String,
    pub is_primary: bool,
}

/// Student CRUD responses
#[derive(Debug, Serialize, ToSchema)]
pub struct StudentResponse {
    pub id: Uuid,
    pub display_name: String,
    pub birth_year: Option<i16>,
    pub grade_level: Option<String>,
    pub methodology_override_id: Option<Uuid>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// GET /v1/families/consent, POST /v1/families/consent
#[derive(Debug, Serialize, ToSchema)]
pub struct ConsentStatusResponse {
    pub status: String,
    pub consented_at: Option<DateTime<Utc>>,
    pub consent_method: Option<String>,
    pub can_create_students: bool,
}

/// POST /v1/families/invites (Phase 2)
#[derive(Debug, Serialize, ToSchema)]
pub struct CoParentInviteResponse {
    pub id: Uuid,
    pub email: String,
    pub expires_at: DateTime<Utc>,
    pub status: String,
}
```

### §8.3 Internal Types (not API-facing)

```rust
/// Payload from Kratos webhooks
pub struct KratosWebhookPayload {
    pub identity_id: Uuid,
    pub traits: KratosTraits,
}

pub struct KratosTraits {
    pub email: String,
    pub name: String,
}

/// Internal family representation (from DB, not returned directly to API)
pub struct Family {
    pub id: Uuid,
    pub display_name: String,
    pub state_code: Option<String>,
    pub location_region: Option<String>,
    pub primary_parent_id: Option<Uuid>,
    pub primary_methodology_id: Uuid,
    pub secondary_methodology_ids: Vec<Uuid>,
    pub subscription_tier: String,
    pub coppa_consent_status: CoppaConsentStatus,
    pub coppa_consented_at: Option<DateTime<Utc>>,
    pub coppa_consent_method: Option<String>,
    pub deletion_requested_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal parent representation
pub struct Parent {
    pub id: Uuid,
    pub family_id: Uuid,
    pub kratos_identity_id: Uuid,
    pub display_name: String,
    pub email: String,
    pub is_primary: bool,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Internal student representation
pub struct Student {
    pub id: Uuid,
    pub family_id: Uuid,
    pub display_name: String,
    pub birth_year: Option<i16>,
    pub grade_level: Option<String>,
    pub methodology_override_id: Option<Uuid>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

/// Create commands used by repository layer
pub struct CreateFamily {
    pub display_name: String,
    pub primary_methodology_id: Uuid,
}

pub struct CreateParent {
    pub family_id: Uuid,
    pub kratos_identity_id: Uuid,
    pub display_name: String,
    pub email: String,
    pub is_primary: bool,
}

pub struct CreateStudent {
    pub display_name: String,
    pub birth_year: Option<i16>,
    pub grade_level: Option<String>,
    pub methodology_override_id: Option<Uuid>,
}

pub struct UpdateFamily {
    pub display_name: Option<String>,
    pub state_code: Option<String>,
    pub location_region: Option<String>,
}

pub struct UpdateParent {
    pub display_name: Option<String>,
    pub email: Option<String>,
}

pub struct UpdateStudent {
    pub display_name: Option<String>,
    pub birth_year: Option<i16>,
    pub grade_level: Option<String>,
    pub methodology_override_id: Option<Option<Uuid>>,
}

pub struct CreateInvite {
    pub invited_by: Uuid,
    pub email: String,
    pub token: String,
    pub expires_at: DateTime<Utc>,
}
```

---

## §9 COPPA Consent Flow

### §9.1 State Machine

The COPPA consent state machine governs when families can create student profiles and
access student-related features. `[S§17.2, ARCH §6.3]`

```
                    ┌─────────────┐
                    │  REGISTERED │  ← Account just created
                    │ (no consent)│
                    └──────┬──────┘
                           │
                    Parent acknowledges
                    COPPA notice
                           │
                    ┌──────▼──────┐
                    │   NOTICED   │  ← Parent has seen the notice
                    │             │
                    └──────┬──────┘
                           │
                    Parent provides
                    verifiable consent
                           │
                    ┌──────▼──────┐
                    │  CONSENTED  │◄──── Can add student profiles
                    │             │      Can use learning tools
                    └──────┬──────┘
                           │
                    ┌──────┴──────┐
                    │             │
             Parent re-verifies  Parent withdraws
                    │             │
             ┌──────▼──────┐ ┌───▼────────┐
             │ RE-VERIFIED │ │  WITHDRAWN  │
             │             │ │             │
             └─────────────┘ └─────────────┘
                              Student data exported
                              then deleted [S§16.3]
```

### §9.2 Transition Rules

| From | To | Precondition | Action | Effects | Audit |
|------|----|-------------|--------|---------|-------|
| `Registered` | `Noticed` | Parent views COPPA notice page | `submit_coppa_consent` with `coppa_notice_acknowledged: true` only | Status updated | Log entry |
| `Noticed` | `Consented` | Parent provides verifiable consent | Validate consent method (e.g., Stripe micro-charge) | Status updated, `coppa_consented_at` set, `coppa_consent_method` set | Log entry |
| `Registered` | `Consented` | Combined flow: acknowledge + consent in one step | Validate consent method | Same as Noticed → Consented | Log entry |
| `Consented` | `ReVerified` | Parent re-verifies consent (e.g., annually) | Validate consent method | `coppa_consented_at` updated | Log entry |
| `Consented` | `Withdrawn` | Parent requests withdrawal | `withdraw_coppa_consent` | 48h grace period → student data export offered → student profiles deleted | Log entry |
| `ReVerified` | `Withdrawn` | Parent requests withdrawal | Same as above | Same as above | Log entry |

**Invalid transitions** (service MUST reject):

| From | To | Reason |
|------|----|--------|
| `Registered` | `ReVerified` | Must consent before re-verifying |
| `Registered` | `Withdrawn` | Nothing to withdraw |
| `Noticed` | `Withdrawn` | No consent was given to withdraw |
| `Withdrawn` | `Consented` | Must re-register (create new account) |
| `Withdrawn` | `ReVerified` | Cannot re-verify withdrawn consent |

### §9.3 Verifiable Consent Methods (Phase 1)

Phase 1 supports one verifiable consent method as permitted by the FTC's COPPA Rule:

**Credit card verification via Stripe**: A micro-charge ($0.50) is placed on the parent's
credit card via Stripe. The charge is immediately refunded. The successful charge serves as
verification that the consenting party has access to a financial instrument, which the FTC
considers a valid method of verifiable parental consent.

- The `verification_token` in `CoppaConsentCommand` is a Stripe payment method token
- The service calls Stripe (via `billing::` adapter or a dedicated COPPA adapter) to
  process the micro-charge and refund
- On success, consent status transitions and audit log is created
- On failure, return 422 with a generic error (do not expose Stripe error details)

### §9.4 Consent Withdrawal Process

1. Parent calls `DELETE /v1/families/consent` (primary parent only)
2. Service transitions status to `Withdrawn`
3. Publishes `CoppaConsentWithdrawn` event
4. 48-hour grace period starts (allows parent to change their mind)
5. During grace period: student profiles are read-only (no new data)
6. After grace period: data export is generated and emailed to parent
7. All student profiles and associated learning data are permanently deleted
8. The family account itself remains active (parents can still use social features)

### §9.5 Student Sessions and COPPA

**Student Sessions and COPPA**: Supervised student sessions (§8.6) do not create independent accounts or collect new PII. All student session activity is logged under the parent's family account. The age gate (10+) is enforced based on `iam_students.birth_year`. Session tokens are scoped to the family and carry no PII. This approach maintains COPPA compliance by operating entirely within the parent-controlled account structure.

---

## §10 Family Account Lifecycle

Step-by-step processes for each lifecycle event. `[S§3.4, S§16.3]`

### §10.1 Creation

Triggered by Kratos post-registration webhook. `[S§6.1]`

1. Kratos completes registration (email/password or OAuth)
2. Kratos calls `POST /hooks/kratos/post-registration` with identity ID + traits
3. Service begins database transaction
4. Creates `iam_families` row with:
   - `display_name` from Kratos traits (parent's name + " Family" or just name)
   - `primary_methodology_id` set to a platform default (e.g., "traditional")
   - `coppa_consent_status` = `Registered`
   - All other fields at defaults
5. Creates `iam_parents` row with:
   - `family_id` pointing to new family
   - `kratos_identity_id` from webhook payload
   - `is_primary` = true
6. Updates `iam_families.primary_parent_id` to the new parent's ID
7. Commits transaction
8. Publishes `FamilyCreated` event (consumed by `social::` to create profile, `onboard::` to start wizard)

**Atomicity**: Steps 4-7 are a single database transaction. If any step fails, the entire
registration is rolled back.

### §10.2 Add Co-Parent (Phase 2)

Triggered by primary parent. `[S§3.4]`

1. Primary parent calls `POST /v1/families/invites` with co-parent's email
2. Service validates: caller is primary parent, email is not already a family member
3. Generates secure random token (32 bytes, base64url-encoded)
4. Creates `iam_co_parent_invites` row with 72-hour expiry
5. Publishes `CoParentInvited` event → `notify::` sends invite email with link
6. Co-parent receives email with link: `https://app.homegrown.academy/invite/{token}`
7. Co-parent creates a Kratos account (or logs in with existing account)
8. Co-parent calls `POST /v1/families/invites/{token}/accept`
9. Service validates: token exists, status is `pending`, not expired
10. Creates new `iam_parents` row with `is_primary` = false
11. Updates invite status to `accepted`
12. Publishes `CoParentAdded` event

### §10.3 Remove Co-Parent (Phase 2)

Primary parent only. `[S§3.4]`

1. Primary parent calls `DELETE /v1/families/parents/:id`
2. Service validates:
   - Caller is primary parent
   - Target is NOT the primary parent (cannot remove self)
   - Target belongs to the same family
3. Revokes all Kratos sessions for the removed parent (`KratosAdapter::revoke_sessions`)
4. Deletes `iam_parents` row (content preservation: social posts by this parent remain
   but are disassociated from the family — handled by `social::` event handler)
5. Publishes `CoParentRemoved` event

### §10.4 Transfer Primary Parent (Phase 2)

Primary parent only. `[S§3.4]`

1. Primary parent calls `POST /v1/families/primary-parent` with `new_primary_id`
2. Service validates:
   - Caller is current primary parent
   - `new_primary_id` is a parent in the same family
   - `new_primary_id` != caller (cannot transfer to self)
3. In a single transaction:
   - Set current primary parent's `is_primary` = false
   - Set new primary parent's `is_primary` = true
   - Update `iam_families.primary_parent_id` to new parent
4. Publishes `PrimaryParentTransferred` event
5. **Billing responsibility**: The new primary parent assumes billing responsibility.
   `billing::` reacts to the event to update Stripe customer association.

### §10.5 Remove Student

Any parent. `[S§3.4]`

1. Parent calls `DELETE /v1/families/students/:id`
2. Service validates student belongs to the family (FamilyScope)
3. Phase 1: Immediate deletion. Phase 2: Data export offered first.
4. Deletes `iam_students` row (cascades to learning data via FK constraints)
5. Publishes `StudentDeleted` event → `learn::` cleans up tool access

### §10.6 Delete Family (Phase 2)

Primary parent only. `[S§3.4, S§16.3]`

1. Primary parent calls `POST /v1/families/deletion-request`
2. Service sets `iam_families.deletion_requested_at` = now()
3. Publishes `FamilyDeletionScheduled` event:
   - `billing::` → cancels active subscriptions
   - `notify::` → sends confirmation email with cancellation link
   - All domains → prepare for cascade
4. **Grace period**: 30 days minimum `[S§16.3]`
5. During grace period: family can cancel via `DELETE /v1/families/deletion-request`
6. After grace period, a background job executes the cascade:
   - Export all family data (offered via email)
   - Delete all student profiles and learning data
   - Delete social posts and connections
   - Delete notification preferences and history
   - Retain marketplace purchase records (legal requirement) `[S§16.3]`
   - Delete all `iam_parents` rows
   - Delete `iam_families` row
   - Delete all Kratos identities for family members
7. Deletion is permanent and irreversible

---

## §11 Middleware & Extractors

The shared types and extractors that IAM depends on and populates are defined in
**00-core §7** (types), **00-core §8** (FamilyScope), and **00-core §13** (middleware
and extractors). This section documents IAM-specific behavior only.

### §11.1 AuthContext Population

IAM owns the *population* of `AuthContext` (type defined in 00-core §7.2). The auth
middleware (`src/middleware/auth.rs`, defined in 00-core §13.1) calls IAM's
`KratosAdapter::validate_session()` and queries IAM repositories to build the `AuthContext`.

**Population flow**:

1. Auth middleware extracts session cookie from request
2. Calls `KratosAdapter::validate_session()` (§7) → returns `kratos_identity_id`
3. Calls `ParentRepository::find_by_kratos_id()` (§6) → returns parent record
4. Calls `FamilyRepository::find_by_id()` (§6) → returns family record
5. Constructs `AuthContext` from parent + family data:
   - `parent_id` from parent record
   - `family_id` from parent record
   - `kratos_identity_id` from Kratos session
   - `is_primary_parent` from parent record
   - `is_platform_admin` from parent record `[S§3.1.5, 11-safety §9]`
   - `subscription_tier` from family record
   - `email` from parent record (NOT logged — PII)
   - `coppa_consent_status` from family record (as String, for RequireCoppaConsent)

**Behavior**: Returns 401 Unauthorized if:
- No session cookie present
- Kratos session is invalid or expired
- Parent not found in local database (orphaned Kratos identity)

### §11.2 COPPA Consent Check

The `RequireCoppaConsent` extractor (00-core §13.3) checks `coppa_consent_status` from
`AuthContext`, which IAM populates from `iam_families.coppa_consent_status` during auth
middleware execution. This avoids an extra DB query per request — the auth middleware already
queries `iam_families` for subscription tier.

### §11.3 Extractor Summary

All extractors are defined in 00-core §13.3. IAM provides the data they operate on:

| Extractor | Defined In | Data Source (IAM) | Behavior |
|-----------|------------|-------------------|----------|
| `AuthContext` | 00-core §7.2 | Auth middleware (§11.1) | 401 if unauthenticated |
| `FamilyScope` | 00-core §8 | Derived from `AuthContext.family_id` | 401 if unauthenticated |
| `RequirePremium` | 00-core §13.3 | `AuthContext.subscription_tier` | 402 if Free |
| `RequireCreator` | 00-core §13.3 | `mkt_creators` lookup via parent_id | 403 if no creator account |
| `RequireCoppaConsent` | 00-core §13.3 | `AuthContext.coppa_consent_status` | 403 if not consented |
| `RequireAdmin` | 00-core §13.3 | `AuthContext.is_platform_admin` | 403 if not admin `[11-safety §9]` |
| `RequirePrimaryParent` | 00-core §13.3 | `AuthContext.is_primary_parent` | 403 if not primary (Phase 2) |

### §11.4 Student Session Permissions

**Student session permissions**: A student session token carries a limited set of permissions defined by the parent at session creation time. The default permission set matches the student's resolved tool set from `method::`. Student sessions can only access: assigned content viewing, quiz-taking, video watching, sequence progression, and assignment status viewing. They cannot access: social features, marketplace, messaging, account settings, other students' data, or parent-level functionality.

---

## §12 Error Types

`IamError` enum defined in `src/iam/` (service-level errors). These map to `AppError`
(defined in 00-core §6) in handlers via `From<IamError> for AppError` (see 00-core §6.4
for the conversion pattern). `[CODING §2.2]`

```rust
#[derive(Debug, thiserror::Error)]
pub enum IamError {
    #[error("family not found")]
    FamilyNotFound,

    #[error("parent not found")]
    ParentNotFound,

    #[error("student not found")]
    StudentNotFound,

    #[error("invite not found")]
    InviteNotFound,

    #[error("invite expired")]
    InviteExpired,

    #[error("invite already accepted")]
    InviteAlreadyAccepted,

    #[error("invalid COPPA consent transition from {from} to {to}")]
    InvalidConsentTransition { from: String, to: String },

    #[error("COPPA consent required")]
    CoppaConsentRequired,

    #[error("consent verification failed")]
    ConsentVerificationFailed,

    #[error("not the primary parent")]
    NotPrimaryParent,

    #[error("cannot remove primary parent")]
    CannotRemovePrimaryParent,

    #[error("cannot transfer primary to self")]
    CannotTransferToSelf,

    #[error("parent already exists in this family")]
    ParentAlreadyInFamily,

    #[error("email already associated with a family")]
    EmailAlreadyAssociated,

    #[error("family deletion already requested")]
    DeletionAlreadyRequested,

    #[error("no pending deletion request")]
    NoPendingDeletion,

    #[error("premium subscription required")]
    PremiumRequired,

    #[error("kratos communication error")]
    KratosError,

    #[error("database error")]
    DatabaseError(#[from] sea_orm::DbErr),
}
```

### §12.1 Error-to-HTTP Mapping

| IamError Variant | HTTP Status | Error Code |
|-----------------|-------------|------------|
| `FamilyNotFound` | 404 Not Found | `family_not_found` |
| `ParentNotFound` | 404 Not Found | `parent_not_found` |
| `StudentNotFound` | 404 Not Found | `student_not_found` |
| `InviteNotFound` | 404 Not Found | `invite_not_found` |
| `InviteExpired` | 410 Gone | `invite_expired` |
| `InviteAlreadyAccepted` | 409 Conflict | `invite_already_accepted` |
| `InvalidConsentTransition` | 422 Unprocessable Entity | `invalid_consent_transition` |
| `CoppaConsentRequired` | 403 Forbidden | `coppa_consent_required` |
| `ConsentVerificationFailed` | 422 Unprocessable Entity | `consent_verification_failed` |
| `NotPrimaryParent` | 403 Forbidden | `not_primary_parent` |
| `CannotRemovePrimaryParent` | 422 Unprocessable Entity | `cannot_remove_primary_parent` |
| `CannotTransferToSelf` | 422 Unprocessable Entity | `cannot_transfer_to_self` |
| `ParentAlreadyInFamily` | 409 Conflict | `parent_already_in_family` |
| `EmailAlreadyAssociated` | 409 Conflict | `email_already_associated` |
| `DeletionAlreadyRequested` | 409 Conflict | `deletion_already_requested` |
| `NoPendingDeletion` | 404 Not Found | `no_pending_deletion` |
| `PremiumRequired` | 402 Payment Required | `premium_required` |
| `KratosError` | 502 Bad Gateway | `auth_service_unavailable` |
| `DatabaseError` | 500 Internal Server Error | `internal_error` |

**API error responses** MUST NOT expose internal details. The error codes above are returned
as `{"error": "<code>", "message": "<user-friendly message>"}`. Internal details (SQL errors,
Kratos response bodies) are logged server-side only. `[CODING §2.2, §5.2]`

---

## §13 Cross-Domain Interactions

### §13.1 IAM Provides (consumed by all domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `AuthContext` | All domains | Axum request extension (middleware) |
| `FamilyScope` | All domains | Axum extractor (from AuthContext) |
| `RequirePremium` | `learn::`, `comply::`, `ai::` | Axum extractor |
| `RequireCreator` | `mkt::` | Axum extractor |
| `RequireCoppaConsent` | `learn::`, `social::` (student features) | Middleware |
| `IamService` trait methods | `method::`, `onboard::`, `billing::` | `Arc<dyn IamService>` |

### §13.2 IAM Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| Default methodology ID | `method::` service | Used during registration to set initial `primary_methodology_id` |
| Stripe micro-charge | `billing::` adapter (or dedicated adapter) | COPPA credit card verification |

### §13.3 Events IAM Publishes

Defined in `src/iam/events.rs`. `[CODING §8.4]`

| Event | Subscribers | Effect |
|-------|------------|--------|
| `FamilyCreated { family_id, parent_id }` | `social::` | Create social profile for family |
| | `onboard::` | Start onboarding wizard |
| `StudentCreated { family_id, student_id }` | `learn::` | Initialize tool access for student |
| `StudentDeleted { family_id, student_id }` | `learn::` | Clean up learning data and tool access |
| `CoppaConsentGranted { family_id }` | `learn::` | Enable student-facing tools |
| `CoppaConsentWithdrawn { family_id }` | `learn::` | Disable student tools, schedule data deletion |
| `CoParentAdded { family_id, parent_id }` | `notify::` | Send welcome email to co-parent |
| `CoParentRemoved { family_id, parent_id }` | `social::` | Disassociate posts from family |
| `FamilyDeletionScheduled { family_id, delete_after }` | `billing::` | Cancel subscriptions |
| | `notify::` | Send confirmation + cancellation link |
| | All domains | Prepare for cascade deletion |
| `PrimaryParentTransferred { family_id, old_primary, new_primary }` | `billing::` | Update Hyperswitch customer `[10-billing §16.4]` |

### §13.4 Events IAM Subscribes To

None in Phase 1. IAM is a foundational domain that other domains depend on, not the reverse.
Future phases may subscribe to `billing::SubscriptionChanged` to update `subscription_tier`
on `iam_families`.

---

## §14 Phase 1 Scope

### In Scope (Phase 1)

- Family + parent creation via Kratos post-registration webhook
- Login webhook (trait sync)
- `GET /v1/auth/me` — current user info
- Family profile CRUD (get, update)
- Student CRUD (create, list, update, delete)
- COPPA consent flow (submit, get status)
- All middleware extractors: `AuthContext`, `FamilyScope`, `RequirePremium`, `RequireCreator`, `RequireCoppaConsent`
- RLS policies for defense-in-depth
- Domain events: `FamilyCreated`, `StudentCreated`, `StudentDeleted`, `CoppaConsentGranted`
- Error types and HTTP mapping

### Out of Scope (Phase 2)

- Co-parent invite flow (create, cancel, accept)
- Co-parent removal
- Primary parent transfer
- COPPA consent withdrawal (with data export + deletion)
- Family deletion request (with grace period + cascade)
- `RequirePrimaryParent` extractor
- `iam_co_parent_invites` table and repository
- Events: `CoParentAdded`, `CoParentRemoved`, `FamilyDeletionScheduled`, `CoppaConsentWithdrawn`, `PrimaryParentTransferred`

### Deferred (Phase 3+)

- ~~Platform administrator access and audit logging `[S§3.1.5]`~~ **Moved to Phase 1** —
  `is_platform_admin` column on `iam_parents` and `RequireAdmin` extractor are required by
  `safety::` which is in the Phase 1 critical path. Granular admin role-based sub-permissions
  remain Phase 2. `[11-safety §9]`
- Supervised student views for ages 10+ `[S§3.3]`
- GDPR compliance features (right to be forgotten, data portability) `[S§17.2]`

---

## §15 Verification Checklist

Numbered assertions that MUST be true when the IAM implementation is correct. Use these as
acceptance criteria for code review and integration testing.

1. Registration creates family + parent atomically (no partial state on failure)
2. Student CRUD is family-scoped (cross-family access is impossible)
3. COPPA state transitions follow the state machine exactly (§9.2 transition rules)
4. Student creation is blocked without COPPA consent (`Consented` or `ReVerified`)
5. `RequirePremium` returns 402 for free-tier users
6. Auth middleware returns 401 for invalid/expired sessions
7. No PII appears in application logs (emails, names, IPs)
8. RLS policies prevent cross-family data access at the database level
9. Co-parent invite tokens expire after 72 hours (Phase 2)
10. Only the primary parent can remove co-parents, transfer primary role, request deletion, or withdraw COPPA consent (Phase 2)
11. Removing a co-parent revokes their Kratos sessions immediately (Phase 2)
12. Family deletion cascade preserves marketplace purchase records (Phase 2)
13. COPPA consent withdrawal triggers student data export before deletion (Phase 2)
14. Webhook endpoints validate shared secret (not Kratos session)
15. Error responses never expose internal details (SQL errors, stack traces, Kratos internals)

---

## §16 Implementation Checklist

### Phase 1 — Foundation (must complete before any other domain)

#### Database
- [ ] Create migration: `iam_families` table with all columns including `coppa_consent_status`
- [ ] Create migration: `iam_parents` table
- [ ] Create migration: `iam_students` table
- [ ] Create migration: `iam_coppa_audit_log` table
- [ ] Create migration: RLS policies for all IAM tables
- [ ] Create migration: PostGIS `location_point` column on `iam_families`
- [ ] Regenerate SeaORM entities from migrations

#### Shared Infrastructure (prerequisite — see 00-core)
- [ ] Verify 00-core §19 checklist is complete (AppError, AuthContext, FamilyScope,
      extractors, middleware, DB pool, Redis pool, EventBus)
- [ ] Implement `From<IamError> for AppError` conversion (00-core §6.4 pattern)

#### Kratos Adapter
- [ ] Implement `KratosAdapter` trait in `src/iam/ports.rs`
- [ ] Implement `KratosAdapterImpl` in `src/iam/adapters/kratos.rs`
- [ ] Configure Kratos YAML (OIDC providers, webhooks, session config)

#### Ports & Traits
- [ ] Define `IamService` trait in `src/iam/ports.rs`
- [ ] Define `FamilyRepository` trait in `src/iam/ports.rs`
- [ ] Define `ParentRepository` trait in `src/iam/ports.rs`
- [ ] Define `StudentRepository` trait in `src/iam/ports.rs`

#### Repository Implementations
- [ ] Implement `PgFamilyRepository`
- [ ] Implement `PgParentRepository`
- [ ] Implement `PgStudentRepository`

#### Service Implementation
- [ ] Implement `IamServiceImpl` with all Phase 1 methods

#### API Endpoints
- [ ] `POST /hooks/kratos/post-registration` — atomic family + parent creation
- [ ] `POST /hooks/kratos/post-login` — session tracking, trait sync
- [ ] `GET  /v1/auth/me` — current user info
- [ ] `GET  /v1/families/profile` — family profile
- [ ] `PATCH /v1/families/profile` — update family profile
- [ ] `POST /v1/families/students` — create student (COPPA-gated)
- [ ] `GET  /v1/families/students` — list students
- [ ] `PATCH /v1/families/students/:id` — update student
- [ ] `DELETE /v1/families/students/:id` — remove student
- [ ] `POST /v1/families/consent` — submit COPPA consent
- [ ] `GET  /v1/families/consent` — get consent status

#### Models (DTOs)
- [ ] `CreateStudentCommand` with validator derives
- [ ] `UpdateStudentCommand` with validator derives
- [ ] `UpdateFamilyCommand` with validator derives
- [ ] `CoppaConsentCommand` with validator derives
- [ ] `StudentResponse` with serde + utoipa derives
- [ ] `FamilyProfileResponse` with serde + utoipa derives
- [ ] `CurrentUserResponse` with serde + utoipa derives
- [ ] `ConsentStatusResponse` with serde + utoipa derives

#### Domain Events
- [ ] Define `FamilyCreated` event
- [ ] Define `StudentCreated` event
- [ ] Define `StudentDeleted` event
- [ ] Define `CoppaConsentGranted` event
- [ ] Register event subscriptions in `app.rs`

#### Tests
- [ ] Integration test: registration creates family + parent atomically
- [ ] Integration test: student CRUD is family-scoped
- [ ] Integration test: student creation blocked without COPPA consent
- [ ] Integration test: auth middleware returns 401 for invalid sessions
- [ ] Integration test: RequirePremium returns 402 for free-tier
- [ ] Unit test: COPPA state transitions (valid transitions succeed, invalid rejected)
- [ ] Verify: no PII in application log output
- [ ] Verify: `cargo clippy -- -D warnings` passes
- [ ] Verify: `cargo test` passes

#### Code Generation
- [ ] Generate OpenAPI spec from Rust types
- [ ] Generate TypeScript types from OpenAPI spec

### Phase 2 — Co-Parent & Lifecycle

#### Database
- [ ] Create migration: `iam_co_parent_invites` table
- [ ] Regenerate SeaORM entities

#### Endpoints
- [ ] `POST /v1/families/invites` — invite co-parent
- [ ] `DELETE /v1/families/invites/:id` — cancel invite
- [ ] `POST /v1/families/invites/:token/accept` — accept invite
- [ ] `DELETE /v1/families/parents/:id` — remove co-parent
- [ ] `POST /v1/families/primary-parent` — transfer primary role
- [ ] `DELETE /v1/families/consent` — withdraw COPPA consent
- [ ] `POST /v1/families/deletion-request` — request family deletion
- [ ] `DELETE /v1/families/deletion-request` — cancel deletion

#### Supporting Work
- [ ] Implement `RequirePrimaryParent` extractor
- [ ] Implement `CoParentInviteRepository` trait + `PgCoParentInviteRepository`
- [ ] Implement co-parent invite email (via `notify::` domain event)
- [ ] Implement family deletion grace period + cascade
- [ ] Implement COPPA consent withdrawal + student data export
- [ ] Define `CoParentAdded`, `CoParentRemoved`, `CoppaConsentWithdrawn`, `FamilyDeletionScheduled`, `PrimaryParentTransferred` events

#### Tests
- [ ] Integration test: co-parent invite flow (create, accept, verify family membership)
- [ ] Integration test: only primary parent can remove co-parents
- [ ] Integration test: primary parent transfer
- [ ] Integration test: family deletion cascade
- [ ] Integration test: co-parent invite token expires after 72 hours
- [ ] Integration test: COPPA withdrawal triggers student data export

---

## Appendix A: Domain Build Order

This document (01-iam) is the first of 14 domain specs. Implementation follows dependency
order — each domain depends only on domains with lower numbers.

| # | Domain | Module | Rationale |
|---|--------|--------|-----------|
| 01 | Identity & Access | `iam::` | Foundation — every other domain depends on AuthContext and FamilyScope |
| 02 | Methodology | `method::` | Cross-cutting config consumed by most other domains |
| 03 | Content & Media | `media::` | Infrastructure — upload/storage needed by social, learning, marketplace |
| 04 | Notifications | `notify::` | Infrastructure — event-driven notifications consumed by all domains |
| 05 | Discovery | `discover::` | Public content, no auth dependency beyond quiz API |
| 06 | Onboarding | `onboard::` | Depends on IAM + Methodology |
| 07 | Social | `social::` | Depends on IAM + Media |
| 08 | Learning | `learn::` | Depends on IAM + Methodology + Media |
| 09 | Search | `search::` | Indexes content from social + learning + marketplace |
| 10 | Marketplace | `mkt::` | Depends on IAM + Media + Search |
| 11 | Billing & Subscriptions | `billing::` | Depends on IAM + Marketplace |
| 12 | Trust & Safety | `safety::` | Depends on Media + Social + Marketplace |
| 13 | Compliance & Reporting | `comply::` | Depends on Learning + IAM |
| 14 | AI & Recommendations | `ai::` | Depends on Learning + Marketplace + Social |
| 15 | Data Lifecycle | `lifecycle::` | Depends on IAM + all domains (export/deletion orchestration) |
| 16 | Administration | `admin::` | Depends on IAM + Safety + all domains (admin views) |
| 17 | Planning & Scheduling | `plan::` | Depends on Learning + Compliance + Social |

---

## §17 Addendum: Session Management & Multi-Device `[S§17.1, S§17.10, S§20.11]`

*Added to address spec gaps in session visibility, multi-device support, and account recovery.*

### §17.1 Session Visibility

IAM provides the underlying Kratos session data that `lifecycle::` exposes via
`GET /v1/account/sessions`. IAM's responsibility is:

- Wrapping Kratos Admin API to list active sessions for a parent identity
- Enriching session data with device type parsed from User-Agent
- Providing IP → city-level location mapping (coarse only, no GPS) `[S§7.8]`
- Exposing `revoke_session(session_id)` and `revoke_all_sessions(identity_id)` via `KratosAdapter`

### §17.2 Concurrent Session Policy

- No hard limit on concurrent sessions per parent `[S§20.11]`
- Parents can have sessions on multiple devices simultaneously (desktop + phone)
- Each co-parent has independent sessions (they are separate Kratos identities)
- Student supervised sessions (§8 in 01-iam) are ephemeral and do not count toward parent sessions

### §17.3 Auth State Change Handling

When authorization state changes mid-session (e.g., co-parent removed from family, subscription
downgraded, account suspended):

- Session revocation for removed co-parents: `FamilyMemberRemoved` event → `iam::` revokes
  all sessions for the removed parent's Kratos identity
- Subscription changes: Cached permission claims in Redis are invalidated on `SubscriptionChanged`
  event. Next request re-evaluates permissions from current subscription state.
- Account suspension: `safety::` sets a Redis flag (`suspended:{family_id}`). Auth middleware
  checks this flag and returns 403 for any request from suspended families. Sessions are NOT
  revoked (so the suspension can be lifted without requiring re-authentication).

### §17.4 Account Recovery Integration

IAM delegates account recovery orchestration to `lifecycle::` but provides the Kratos
integration layer:

```rust
// In KratosAdapter:

/// Initiate Kratos recovery flow for an email address.
/// Returns a recovery flow ID (Kratos-managed).
async fn initiate_recovery_flow(
    &self,
    email: &str,
) -> Result<String, KratosError>;

/// Admin-level password reset (for escalated recovery, called by admin::).
/// Requires RequireAdmin context.
async fn admin_reset_credentials(
    &self,
    identity_id: &str,
) -> Result<(), KratosError>;
```

### §17.5 IAM Implementation Updates

The following changes to existing IAM sections are needed:

- **§7 Kratos Adapter**: Add `list_sessions()`, `revoke_session()`, `revoke_all_sessions()`,
  `initiate_recovery_flow()`, and `admin_reset_credentials()` methods
- **§10 Family Account Lifecycle**: Add `FamilyMemberRemoved` event handling (session revocation)
- **§11 Middleware & Extractors**: Add suspension check via Redis flag in auth middleware
- **§13 Cross-Domain Interactions**: Add interaction with `lifecycle::` (session data provider)
  and `admin::` (admin-level credential reset)
