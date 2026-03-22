# Domain Spec 15 — Data Lifecycle (lifecycle::)

## §1 Overview

The Data Lifecycle domain orchestrates **data export, account deletion, data retention
enforcement, and account recovery** across the entire platform. It does not own user data
directly — each domain owns its own data. Instead, lifecycle:: provides the coordination
layer that ensures cross-domain operations (like "delete my family account") happen
atomically and completely, satisfying COPPA requirements `[S§17.2]` and the platform's
data portability commitments `[S§8.5, S§16.3]`.

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/lifecycle/` |
| **DB prefix** | `lifecycle_` |
| **Complexity class** | Non-complex (no `domain/` subdirectory) `[ARCH §4.5]` |
| **External adapter** | None (orchestrates other domains' service interfaces) |
| **Key constraint** | Every user-data query family-scoped via `FamilyScope` `[CODING §2.4, §2.5]`; COPPA deletion MUST complete within regulatory timeframe; export MUST complete within 24 hours `[S§8.5]` |

**What lifecycle:: owns**: Export request records, export manifests, deletion request records,
deletion grace period management, retention policy configuration, account recovery request
records, cross-domain orchestration for export and deletion workflows.

**What lifecycle:: does NOT own**: The actual user data in any domain. Credential management
(owned by `iam::` / Ory Kratos). Subscription cancellation (owned by `billing::`). Media
file storage (owned by `media::`). Marketplace purchase records retained for legal compliance
(owned by `mkt::`).

**What lifecycle:: delegates**: Per-domain data extraction → each domain's `ExportHandler`
implementation. Per-domain data deletion → each domain's `DeletionHandler` implementation.
Notification delivery → `notify::` (via domain events). Subscription cancellation →
`billing::BillingService`. Session revocation → `iam::KratosAdapter::revoke_sessions()`.
Background job scheduling → sidekiq-rs `[ARCH §12]`.

---

## §2 Requirements Traceability

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Learning data export in standard format (CSV, JSON) | `[S§8.5]` | §5, §9 |
| Export available regardless of subscription tier | `[S§8.5]` | §5, §9 |
| Export completable within 24 hours | `[S§8.5]` | §5, §9 |
| Student profile deletion with export + grace period | `[S§16.3]` | §5, §10 |
| Family account deletion with full export offered | `[S§16.3]` | §5, §10 |
| COPPA deletion request within regulatory timeframe | `[S§16.3, S§17.2]` | §10.3 |
| Creator account deactivation | `[S§16.3]` | §10.4 |
| Content removal retention for audit trail | `[S§16.3]` | §11 |
| Session management with remote revocation | `[S§17.1]` | §12 |
| Data retention policies | `[S§17.12]` | §11 |

---

## §3 Database Schema

All tables use the `lifecycle_` prefix. `[ARCH §5.1]`

### §3.1 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_lifecycle_tables.rs
-- =============================================================================

-- Export requests: tracks family data export jobs
CREATE TABLE lifecycle_export_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    requested_by    UUID NOT NULL REFERENCES iam_parents(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'expired')),
    format          VARCHAR(10) NOT NULL DEFAULT 'json'
                    CHECK (format IN ('json', 'csv')),
    -- Which domains to include (NULL = all)
    include_domains TEXT[] DEFAULT NULL,
    -- R2 key for the completed export archive
    archive_key     VARCHAR(512),
    -- Signed download URL (generated on demand, short TTL)
    download_expires_at TIMESTAMPTZ,
    -- Export size in bytes (populated on completion)
    size_bytes      BIGINT,
    error_message   TEXT,                -- internal error (never exposed to user)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '7 days'
);

CREATE INDEX idx_lifecycle_export_requests_family
    ON lifecycle_export_requests(family_id);
CREATE INDEX idx_lifecycle_export_requests_status
    ON lifecycle_export_requests(status)
    WHERE status IN ('pending', 'processing');

-- Deletion requests: tracks account deletion with grace period
CREATE TABLE lifecycle_deletion_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    requested_by    UUID NOT NULL REFERENCES iam_parents(id),
    reason          TEXT,                -- optional user-provided reason
    deletion_type   VARCHAR(20) NOT NULL
                    CHECK (deletion_type IN ('family', 'student', 'coppa')),
    -- For student-specific deletion
    student_id      UUID REFERENCES iam_students(id),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'grace_period', 'processing',
                                      'completed', 'cancelled')),
    -- Grace period (30 days for family, 7 days for COPPA)
    grace_period_ends_at TIMESTAMPTZ NOT NULL,
    -- Export was offered/completed before deletion
    export_offered  BOOLEAN NOT NULL DEFAULT true,
    export_request_id UUID REFERENCES lifecycle_export_requests(id),
    -- Tracks which domains have completed their deletion
    domain_status   JSONB NOT NULL DEFAULT '{}'::JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    cancelled_at    TIMESTAMPTZ
);

CREATE INDEX idx_lifecycle_deletion_requests_family
    ON lifecycle_deletion_requests(family_id);
CREATE INDEX idx_lifecycle_deletion_requests_grace
    ON lifecycle_deletion_requests(grace_period_ends_at)
    WHERE status = 'grace_period';

-- Account recovery requests
CREATE TABLE lifecycle_recovery_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL,
    -- Verification method used
    verification_method VARCHAR(20) NOT NULL
                    CHECK (verification_method IN ('email', 'support_ticket', 'identity_document')),
    status          VARCHAR(20) NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'verified', 'escalated', 'completed', 'denied')),
    -- Support ticket reference (if escalated)
    support_ticket_id VARCHAR(100),
    -- Resolved parent account (populated on verification)
    resolved_parent_id UUID REFERENCES iam_parents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ,
    -- Recovery requests expire after 7 days
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT now() + INTERVAL '7 days'
);

CREATE INDEX idx_lifecycle_recovery_requests_email
    ON lifecycle_recovery_requests(email);

-- Retention policy configuration
CREATE TABLE lifecycle_retention_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    data_category   VARCHAR(50) NOT NULL UNIQUE
                    CHECK (data_category IN (
                        'user_content',         -- posts, comments, messages
                        'learning_data',        -- activities, journals, progress
                        'moderation_records',   -- flags, reports, actions
                        'audit_logs',           -- security events
                        'marketplace_records',  -- purchases, payouts (legal retention)
                        'export_archives',      -- completed export files
                        'session_data'          -- login sessions, tokens
                    )),
    retention_days  INT NOT NULL,
    -- Whether this policy applies after account deletion or to active accounts
    applies_to      VARCHAR(20) NOT NULL DEFAULT 'post_deletion'
                    CHECK (applies_to IN ('post_deletion', 'active', 'both')),
    description     TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed default retention policies
INSERT INTO lifecycle_retention_policies (data_category, retention_days, applies_to, description) VALUES
    ('user_content',        0,    'post_deletion', 'Deleted immediately on account deletion'),
    ('learning_data',       0,    'post_deletion', 'Deleted immediately (export offered first)'),
    ('moderation_records',  365,  'post_deletion', 'Retained 1 year for audit trail after deletion'),
    ('audit_logs',          730,  'both',          'Retained 2 years for security compliance'),
    ('marketplace_records', 2555, 'post_deletion', 'Retained 7 years for tax/legal compliance'),
    ('export_archives',     7,    'active',        'Export files expire after 7 days'),
    ('session_data',        30,   'active',        'Sessions expire after 30 days of inactivity');
```

### §3.2 Row-Level Security

```sql
-- Family-scoped RLS for export and deletion requests
ALTER TABLE lifecycle_export_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY lifecycle_export_family_scope ON lifecycle_export_requests
    USING (family_id = current_setting('app.current_family_id')::UUID);

ALTER TABLE lifecycle_deletion_requests ENABLE ROW LEVEL SECURITY;
CREATE POLICY lifecycle_deletion_family_scope ON lifecycle_deletion_requests
    USING (family_id = current_setting('app.current_family_id')::UUID);

-- Recovery requests are NOT family-scoped (user may not be authenticated)
-- Admin access only for reviewing recovery requests
```

---

## §4 API Endpoints

All endpoints require authentication except account recovery initiation.

```
POST   /v1/account/export                  # Request data export
GET    /v1/account/export/:id              # Check export status / download
GET    /v1/account/exports                 # List past export requests

POST   /v1/account/deletion                # Request account deletion (starts grace period)
GET    /v1/account/deletion                # Get active deletion request status
DELETE /v1/account/deletion                # Cancel pending deletion (during grace period)

POST   /v1/account/recovery                # Initiate account recovery (unauthenticated)
GET    /v1/account/recovery/:id            # Check recovery status

GET    /v1/account/sessions                # List active sessions
DELETE /v1/account/sessions/:id            # Revoke a specific session
DELETE /v1/account/sessions                # Revoke all sessions

# Admin endpoints (RequireAdmin)
GET    /v1/admin/lifecycle/deletions       # List pending/processing deletions
GET    /v1/admin/lifecycle/recoveries      # List pending recovery requests
POST   /v1/admin/lifecycle/recoveries/:id/resolve  # Resolve recovery request
GET    /v1/admin/lifecycle/retention        # View retention policies
PATCH  /v1/admin/lifecycle/retention/:id    # Update retention policy
```

---

## §5 Service Interface

```rust
#[async_trait]
pub trait LifecycleService: Send + Sync {
    // === Data Export ===

    /// Request a full data export for the family.
    /// Enqueues a background job that calls each domain's ExportHandler.
    async fn request_export(
        &self,
        auth: &AuthContext,
        req: RequestExportInput,
    ) -> Result<ExportRequestId, AppError>;

    /// Get export request status. Returns download URL if completed.
    async fn get_export_status(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        export_id: Uuid,
    ) -> Result<ExportStatusResponse, AppError>;

    /// List past export requests for the family.
    async fn list_exports(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        pagination: PaginationParams,
    ) -> Result<PaginatedResponse<ExportSummary>, AppError>;

    // === Account Deletion ===

    /// Request account deletion. Starts a grace period.
    /// Sends confirmation email and offers data export.
    async fn request_deletion(
        &self,
        auth: &AuthContext,
        req: RequestDeletionInput,
    ) -> Result<DeletionRequestId, AppError>;

    /// Get active deletion request status.
    async fn get_deletion_status(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
    ) -> Result<Option<DeletionStatusResponse>, AppError>;

    /// Cancel a pending deletion during the grace period.
    async fn cancel_deletion(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
    ) -> Result<(), AppError>;

    // === Account Recovery ===

    /// Initiate account recovery (unauthenticated).
    async fn initiate_recovery(
        &self,
        req: InitiateRecoveryInput,
    ) -> Result<RecoveryRequestId, AppError>;

    /// Check recovery request status.
    async fn get_recovery_status(
        &self,
        recovery_id: Uuid,
    ) -> Result<RecoveryStatusResponse, AppError>;

    // === Session Management ===

    /// List active sessions for the current user.
    async fn list_sessions(
        &self,
        auth: &AuthContext,
    ) -> Result<Vec<SessionInfo>, AppError>;

    /// Revoke a specific session.
    async fn revoke_session(
        &self,
        auth: &AuthContext,
        session_id: String,
    ) -> Result<(), AppError>;

    /// Revoke all sessions except the current one ("sign out everywhere").
    async fn revoke_all_sessions(
        &self,
        auth: &AuthContext,
    ) -> Result<u32, AppError>; // returns count of revoked sessions
}
```

---

## §6 Repository Interfaces

```rust
#[async_trait]
pub trait ExportRequestRepository: Send + Sync {
    async fn create(
        &self,
        scope: &FamilyScope,
        input: &CreateExportRequest,
    ) -> Result<ExportRequest, DbErr>;

    async fn find_by_id(
        &self,
        scope: &FamilyScope,
        id: Uuid,
    ) -> Result<Option<ExportRequest>, DbErr>;

    async fn list_by_family(
        &self,
        scope: &FamilyScope,
        pagination: &PaginationParams,
    ) -> Result<Vec<ExportRequest>, DbErr>;

    async fn update_status(
        &self,
        id: Uuid,
        status: ExportStatus,
        archive_key: Option<String>,
        size_bytes: Option<i64>,
    ) -> Result<(), DbErr>;
}

#[async_trait]
pub trait DeletionRequestRepository: Send + Sync {
    async fn create(
        &self,
        scope: &FamilyScope,
        input: &CreateDeletionRequest,
    ) -> Result<DeletionRequest, DbErr>;

    async fn find_active_by_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Option<DeletionRequest>, DbErr>;

    async fn update_status(
        &self,
        id: Uuid,
        status: DeletionStatus,
    ) -> Result<(), DbErr>;

    async fn update_domain_status(
        &self,
        id: Uuid,
        domain: &str,
        completed: bool,
    ) -> Result<(), DbErr>;

    async fn cancel(
        &self,
        scope: &FamilyScope,
        id: Uuid,
    ) -> Result<(), DbErr>;

    /// Find all deletion requests whose grace period has expired.
    async fn find_ready_for_deletion(&self) -> Result<Vec<DeletionRequest>, DbErr>;
}

#[async_trait]
pub trait RecoveryRequestRepository: Send + Sync {
    async fn create(
        &self,
        input: &CreateRecoveryRequest,
    ) -> Result<RecoveryRequest, DbErr>;

    async fn find_by_id(
        &self,
        id: Uuid,
    ) -> Result<Option<RecoveryRequest>, DbErr>;

    async fn update_status(
        &self,
        id: Uuid,
        status: RecoveryStatus,
        resolved_parent_id: Option<Uuid>,
    ) -> Result<(), DbErr>;
}
```

---

## §7 Domain Export & Deletion Contracts

Each domain that stores family data MUST implement these traits so lifecycle:: can
orchestrate cross-domain export and deletion:

```rust
/// Implemented by each domain that has exportable family data.
/// Registered at application startup in app.rs.
#[async_trait]
pub trait ExportHandler: Send + Sync {
    /// Domain identifier (e.g., "learning", "social", "compliance")
    fn domain_name(&self) -> &'static str;

    /// Export all family data for this domain in the requested format.
    /// Returns a Vec of (filename, content_bytes) pairs.
    async fn export_family_data(
        &self,
        family_id: Uuid,
        format: ExportFormat,
    ) -> Result<Vec<(String, Vec<u8>)>, AppError>;
}

/// Implemented by each domain that stores deletable family data.
/// Registered at application startup in app.rs.
#[async_trait]
pub trait DeletionHandler: Send + Sync {
    /// Domain identifier (e.g., "learning", "social", "compliance")
    fn domain_name(&self) -> &'static str;

    /// Delete all family data for this domain.
    /// MUST be idempotent — calling twice with the same family_id MUST NOT error.
    async fn delete_family_data(
        &self,
        family_id: Uuid,
    ) -> Result<(), AppError>;

    /// Delete data for a specific student within a family.
    async fn delete_student_data(
        &self,
        family_id: Uuid,
        student_id: Uuid,
    ) -> Result<(), AppError>;
}

// Domains that MUST implement ExportHandler + DeletionHandler:
// - iam::       (family profiles, parent info, student profiles)
// - learn::     (activities, journals, reading lists, assessments, progress, quiz sessions)
// - social::    (posts, comments, messages, friend lists, group memberships)
// - mkt::       (purchase history — export only; purchase records retained per legal policy)
// - comply::    (attendance, compliance reports, portfolios)
// - notify::    (notification preferences, notification history)
// - plan::      (schedule items, calendar preferences)
// - media::     (uploaded files — deletion of R2 objects)
```

---

## §8 Models (DTOs)

```rust
// --- Request types ---

#[derive(Deserialize, ToSchema)]
pub struct RequestExportInput {
    /// Export format: "json" or "csv"
    pub format: Option<ExportFormat>,
    /// Specific domains to include (None = all)
    pub include_domains: Option<Vec<String>>,
}

#[derive(Deserialize, ToSchema)]
pub struct RequestDeletionInput {
    /// Deletion type
    pub deletion_type: DeletionType,
    /// For student-specific deletion
    pub student_id: Option<Uuid>,
    /// Optional reason
    pub reason: Option<String>,
}

#[derive(Deserialize, ToSchema)]
pub struct InitiateRecoveryInput {
    /// Email address associated with the account
    pub email: String,
}

// --- Response types ---

#[derive(Serialize, ToSchema)]
pub struct ExportStatusResponse {
    pub id: Uuid,
    pub status: ExportStatus,
    pub format: ExportFormat,
    pub size_bytes: Option<i64>,
    pub download_url: Option<String>,  // pre-signed R2 URL, short TTL
    pub created_at: DateTime<Utc>,
    pub completed_at: Option<DateTime<Utc>>,
    pub expires_at: DateTime<Utc>,
}

#[derive(Serialize, ToSchema)]
pub struct DeletionStatusResponse {
    pub id: Uuid,
    pub status: DeletionStatus,
    pub deletion_type: DeletionType,
    pub grace_period_ends_at: DateTime<Utc>,
    pub export_offered: bool,
    pub export_request_id: Option<Uuid>,
    pub created_at: DateTime<Utc>,
}

#[derive(Serialize, ToSchema)]
pub struct RecoveryStatusResponse {
    pub id: Uuid,
    pub status: RecoveryStatus,
    pub verification_method: VerificationMethod,
    pub created_at: DateTime<Utc>,
}

#[derive(Serialize, ToSchema)]
pub struct SessionInfo {
    pub session_id: String,
    pub device_type: Option<String>,   // "desktop", "mobile", "tablet"
    pub user_agent: Option<String>,
    pub ip_address: Option<String>,    // coarse — city-level only
    pub last_active: DateTime<Utc>,
    pub is_current: bool,
}

// --- Enums ---

#[derive(Serialize, Deserialize, ToSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExportFormat { Json, Csv }

#[derive(Serialize, Deserialize, ToSchema)]
#[serde(rename_all = "snake_case")]
pub enum ExportStatus { Pending, Processing, Completed, Failed, Expired }

#[derive(Serialize, Deserialize, ToSchema)]
#[serde(rename_all = "snake_case")]
pub enum DeletionType { Family, Student, Coppa }

#[derive(Serialize, Deserialize, ToSchema)]
#[serde(rename_all = "snake_case")]
pub enum DeletionStatus { Pending, GracePeriod, Processing, Completed, Cancelled }

#[derive(Serialize, Deserialize, ToSchema)]
#[serde(rename_all = "snake_case")]
pub enum RecoveryStatus { Pending, Verified, Escalated, Completed, Denied }

#[derive(Serialize, Deserialize, ToSchema)]
#[serde(rename_all = "snake_case")]
pub enum VerificationMethod { Email, SupportTicket, IdentityDocument }
```

---

## §9 Data Export Pipeline (Domain Deep-Dive 1)

### §9.1 Export Workflow

1. Parent requests export via `POST /v1/account/export`
2. Service creates `lifecycle_export_requests` record with status `pending`
3. Service enqueues `DataExportJob` (Default queue)
4. Background job:
   a. Sets status to `processing`
   b. Iterates over registered `ExportHandler` implementations
   c. Each handler returns `Vec<(filename, bytes)>` for the family
   d. Job assembles all files into a ZIP archive
   e. Uploads archive to Cloudflare R2 at `exports/{family_id}/{request_id}.zip`
   f. Sets status to `completed`, stores `archive_key` and `size_bytes`
   g. Publishes `DataExportCompleted` event → `notify::` sends email with download link

### §9.2 Export File Structure

```
homegrown-academy-export-{date}/
├── metadata.json              # Export metadata (family info, date, version)
├── learning/
│   ├── activities.json        # or .csv
│   ├── journals.json
│   ├── reading-lists.json
│   ├── assessments.json
│   ├── quiz-sessions.json
│   └── progress.json
├── social/
│   ├── posts.json
│   ├── comments.json
│   ├── messages.json
│   └── friends.json
├── compliance/
│   ├── attendance.json
│   └── reports.json
├── marketplace/
│   └── purchases.json         # Purchase history (no downloaded content)
├── planning/
│   └── schedule-items.json
├── notifications/
│   └── preferences.json
└── media/
    └── manifest.json          # List of media files with R2 URLs (time-limited)
```

### §9.3 Export Guarantees

- Export MUST complete within 24 hours of request `[S§8.5]`
- Export MUST be available regardless of subscription tier `[S§8.5]`
- Export archive MUST be downloadable for 7 days after completion
- Export MUST NOT include data from other families
- Export MUST include all learning data: activities, assessments, reading lists, journal
  entries, projects, progress records, quiz sessions, sequence progress `[S§8.5]`
- Media files are NOT embedded in the export (too large). Instead, `media/manifest.json`
  contains time-limited download URLs for each file.

---

## §10 Account Deletion Pipeline (Domain Deep-Dive 2)

### §10.1 Family Account Deletion

1. Parent (must be primary parent) requests deletion via `POST /v1/account/deletion`
2. Service validates: requester is primary parent, no active deletion request exists
3. Service offers data export (links to export endpoint)
4. Service creates `lifecycle_deletion_requests` record with:
   - `status = 'grace_period'`
   - `grace_period_ends_at = now() + 30 days`
5. Publishes `AccountDeletionRequested` event → `notify::` sends confirmation email
6. During grace period: family can log in, use the platform, and cancel deletion
7. After grace period: `ProcessDeletionJob` (recurring daily check) transitions to `processing`
8. Deletion job:
   a. Revokes all Kratos sessions for the family
   b. Cancels active subscriptions via `billing::BillingService`
   c. Iterates over registered `DeletionHandler` implementations
   d. Each handler deletes its domain's data for the family
   e. Updates `domain_status` JSONB as each domain completes
   f. After all domains complete: sets status to `completed`
   g. Deletes the IAM family record last (cascade)
9. Publishes `AccountDeletionCompleted` event (for audit log)

### §10.2 Student Profile Deletion

Same workflow as family deletion, but scoped to a single student:
- Grace period: 7 days (shorter — less data)
- Only calls `delete_student_data()` on each handler
- Family account remains active
- Marketplace purchases associated with the student remain (legal retention)

### §10.3 COPPA Deletion

When a parent requests deletion of a child's data under COPPA:
- Grace period: 0 days (immediate processing per COPPA requirements)
- Status transitions directly from `pending` → `processing`
- MUST complete within the timeframe required by COPPA regulations
- Publishes `CoppaDeleteRequested` event for audit trail

### §10.4 Creator Account Deactivation

When a creator deactivates their creator status:
- Published listings MAY remain accessible to existing purchasers `[S§16.3]`
- New purchases MUST be disabled
- Pending payouts MUST be settled
- Creator profile and analytics data are deleted
- Content files remain in R2 for existing purchasers until all purchases expire

### §10.5 Deletion Guarantees

- Deletion MUST complete within 30 days after grace period ends `[S§16.3]`
- All family-scoped data MUST be deleted across all domains
- Marketplace purchase records are retained per legal policy (7 years)
- Moderation records (flags, reports) are retained per audit policy (1 year)
- Audit logs are retained per compliance policy (2 years)
- Deletion is irreversible after grace period — make this clear in the confirmation flow

---

## §11 Data Retention Policy Enforcement (Domain Deep-Dive 3)

The `lifecycle_retention_policies` table defines how long data is retained after specific
events. A daily recurring job (`RetentionEnforcementJob`, Low queue) scans for data that
exceeds its retention window and triggers cleanup.

### §11.1 Default Retention Periods

| Category | Retention (post-deletion) | Rationale |
|----------|--------------------------|-----------|
| User content (posts, comments, messages) | Immediate | User's data, no legal hold |
| Learning data (activities, journals, progress) | Immediate | User's data, export offered first |
| Moderation records (flags, reports, actions) | 1 year | Audit trail for appeals and legal |
| Audit logs (security events) | 2 years | Security compliance |
| Marketplace records (purchases, payouts) | 7 years | Tax/legal compliance (IRS 1099-K) |
| Export archives | 7 days (active accounts) | Storage cost management |
| Session data | 30 days inactivity | Kratos session TTL |

### §11.2 Enforcement

- `RetentionEnforcementJob` runs daily at 4:00 AM UTC
- Scans `lifecycle_deletion_requests` where `status = 'completed'`
- For each completed deletion, checks `domain_status` and retention windows
- Deletes retention-expired records (moderation, audit) when their windows close
- Cleans up expired export archives from R2

---

## §12 Session Management (Domain Deep-Dive 4)

Session management provides visibility and control over active login sessions.
`[S§17.1, S§20.11]`

### §12.1 Session Listing

- Lists all active Kratos sessions for the authenticated parent
- Enriches with device info parsed from User-Agent
- IP addresses shown at city-level only (no precise geolocation) `[S§7.8]`
- Marks the current session so users don't accidentally revoke it

### §12.2 Session Revocation

- **Single session**: Revoke via `DELETE /v1/account/sessions/:id`
- **All sessions**: Revoke via `DELETE /v1/account/sessions` (keeps current session)
- Revocation calls `iam::KratosAdapter::revoke_session()` and clears Redis session cache
- Publishes `SessionRevoked` event for audit logging

### §12.3 Suspicious Session Detection

Phase 2: If concurrent sessions are detected from geographically distant locations within
a short time window (e.g., US East and Europe within 30 minutes), publish a
`SuspiciousSessionDetected` event → `notify::` sends security alert email.

---

## §13 Account Recovery (Domain Deep-Dive 5)

### §13.1 Recovery Flow

1. User visits `/account/recovery` (unauthenticated)
2. Submits email address
3. If email matches an account: sends recovery email via Kratos recovery flow
4. If email does NOT match: still shows "recovery email sent" (prevent enumeration)
5. Recovery email contains a one-time link to reset password (Kratos flow)

### §13.2 Escalated Recovery (Phase 2)

For cases where email-based recovery fails (compromised email, lost access):
1. User submits a support ticket via recovery form
2. Request enters `escalated` status
3. Admin reviews the request in the admin dashboard
4. Admin may request identity verification (government ID, proof of account ownership)
5. On verification, admin manually resets the account via Kratos admin API

---

## §14 Background Jobs

| Job | Queue | Schedule | Description |
|-----|-------|----------|-------------|
| `DataExportJob` | Default | On-demand | Execute cross-domain data export |
| `ProcessDeletionJob` | Default | Daily 5:00 AM UTC | Process deletion requests past grace period |
| `RetentionEnforcementJob` | Low | Daily 4:00 AM UTC | Clean up retention-expired records |
| `ExpireExportArchivesJob` | Low | Daily 6:00 AM UTC | Delete expired export archives from R2 |

---

## §15 Events lifecycle:: Publishes

| Event | Payload | Consumers |
|-------|---------|-----------|
| `DataExportRequested` | `{ family_id, export_id, format }` | `notify::` (confirmation email) |
| `DataExportCompleted` | `{ family_id, export_id, download_url }` | `notify::` (download ready email) |
| `AccountDeletionRequested` | `{ family_id, deletion_type, grace_period_ends_at }` | `notify::` (confirmation email) |
| `AccountDeletionCompleted` | `{ family_id }` | Audit log |
| `CoppaDeleteRequested` | `{ family_id, student_id }` | Audit log |
| `SessionRevoked` | `{ parent_id, session_id, revoke_type }` | Audit log |

---

## §16 Error Types

```rust
#[derive(Debug, thiserror::Error)]
pub enum LifecycleError {
    #[error("Export request not found")]
    ExportNotFound,

    #[error("Export has expired")]
    ExportExpired,

    #[error("An active deletion request already exists")]
    DeletionAlreadyPending,

    #[error("Cannot cancel deletion — grace period has ended")]
    GracePeriodExpired,

    #[error("Only the primary parent can request family deletion")]
    NotPrimaryParent,

    #[error("Recovery request not found or expired")]
    RecoveryNotFound,

    #[error("Recovery request has expired")]
    RecoveryExpired,

    #[error("Cannot revoke current session via this endpoint")]
    CannotRevokeCurrent,

    #[error("Database error")]
    Database(#[from] sea_orm::DbErr),
}
```

**HTTP mapping**:

| Error | HTTP Status |
|-------|-------------|
| `ExportNotFound` | 404 |
| `ExportExpired` | 410 Gone |
| `DeletionAlreadyPending` | 409 Conflict |
| `GracePeriodExpired` | 409 Conflict |
| `NotPrimaryParent` | 403 Forbidden |
| `RecoveryNotFound` | 404 |
| `RecoveryExpired` | 410 Gone |
| `CannotRevokeCurrent` | 400 Bad Request |

---

## §17 Cross-Domain Interactions

| Direction | Domain | Interaction |
|-----------|--------|-------------|
| lifecycle:: → iam:: | Service call | Look up family, revoke Kratos sessions |
| lifecycle:: → billing:: | Service call | Cancel active subscriptions before deletion |
| lifecycle:: → all domains | ExportHandler trait | Each domain exports its family data |
| lifecycle:: → all domains | DeletionHandler trait | Each domain deletes its family data |
| lifecycle:: → notify:: | Domain event | Export/deletion notifications |
| lifecycle:: → media:: | DeletionHandler | Delete R2 objects for family |
| admin:: → lifecycle:: | Service call | View/manage recovery requests, retention policies |

---

## §18 Phase Scope

### Phase 1

- Family data export (JSON and CSV)
- Family account deletion with 30-day grace period
- Student profile deletion with 7-day grace period
- COPPA deletion (immediate processing)
- Session listing and revocation
- Basic account recovery (email-based, via Kratos)
- Default retention policies

### Phase 2

- Escalated account recovery (support ticket, identity verification)
- Suspicious session detection
- Export format enhancements (PDF summary, media download links)
- Admin retention policy management UI

---

## §19 Verification Checklist

- [ ] Every `ExportHandler` and `DeletionHandler` is registered at startup
- [ ] Export produces valid JSON/CSV that can be re-imported (round-trip test)
- [ ] Deletion removes all family data from all domains (integration test)
- [ ] COPPA deletion completes within regulatory timeframe
- [ ] Grace period cancellation works correctly
- [ ] Session revocation invalidates Kratos sessions and Redis cache
- [ ] Export archives expire and are cleaned from R2
- [ ] Retention enforcement runs without errors on empty/full datasets
- [ ] Account recovery does not reveal whether an email exists (enumeration prevention)

---

## §20 Module Structure

```
src/lifecycle/
├── mod.rs              # Re-exports
├── handlers.rs         # Axum route handlers
├── service.rs          # Orchestration logic
├── repository.rs       # lifecycle_ table queries
├── models.rs           # DTOs (request/response)
├── ports.rs            # Service + repository trait definitions
├── events.rs           # Domain events (export/deletion lifecycle)
└── entities/           # SeaORM-generated (lifecycle_ tables)
```
