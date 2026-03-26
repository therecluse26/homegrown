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
| **Module path** | `internal/lifecycle/` |
| **DB prefix** | `lifecycle_` |
| **Complexity class** | Non-complex `[ARCH §4.5]` |
| **External adapter** | None (orchestrates other domains' service interfaces) |
| **Key constraint** | Every user-data query family-scoped via `FamilyScope` `[CODING §2.4, §2.5]`; COPPA deletion MUST complete within regulatory timeframe; export MUST complete within 24 hours `[S§8.5]` |

**What lifecycle:: owns**: Export request records, export manifests, deletion request records,
deletion grace period management, retention policy configuration, account recovery request
records, cross-domain orchestration for export and deletion workflows.

**What lifecycle:: does NOT own**: The actual user data in any domain. Credential management
(owned by `iam::` / Ory Kratos). Subscription cancellation (owned by `billing::`). Media
file storage (owned by `media::`). Marketplace purchase records retained for legal compliance
(owned by `mkt::`).

**What lifecycle:: delegates**: Per-domain data extraction -> each domain's `ExportHandler`
implementation. Per-domain data deletion -> each domain's `DeletionHandler` implementation.
Notification delivery -> `notify::` (via domain events). Subscription cancellation ->
`billing.BillingService`. Session revocation -> `iam.KratosAdapter.RevokeSessions()`.
Background job scheduling -> `JobEnqueuer` `[ARCH §12]`.

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
-- Migration: YYYYMMDD_000001_create_lifecycle_tables.sql
-- =============================================================================

-- Export requests: tracks family data export jobs
CREATE TABLE lifecycle_export_requests (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
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
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
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
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
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
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
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

```go
// internal/lifecycle/ports.go

// LifecycleService defines the data lifecycle domain's service interface.
type LifecycleService interface {
    // === Data Export ===

    // RequestExport requests a full data export for the family.
    // Enqueues a background job that calls each domain's ExportHandler.
    RequestExport(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, req *RequestExportInput) (uuid.UUID, error)

    // GetExportStatus returns export request status. Returns download URL if completed.
    GetExportStatus(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (*ExportStatusResponse, error)

    // ListExports lists past export requests for the family.
    ListExports(ctx context.Context, scope *shared.FamilyScope, pagination *PaginationParams) (*PaginatedExports, error)

    // ProcessExport executes the cross-domain data export for a given export request.
    // Called by the background job worker.
    ProcessExport(ctx context.Context, exportID uuid.UUID, familyID uuid.UUID) error

    // === Account Deletion ===

    // RequestDeletion requests account deletion. Starts a grace period.
    // Sends confirmation email and offers data export.
    RequestDeletion(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, req *RequestDeletionInput) (uuid.UUID, error)

    // GetDeletionStatus returns active deletion request status.
    GetDeletionStatus(ctx context.Context, scope *shared.FamilyScope) (*DeletionStatusResponse, error)

    // CancelDeletion cancels a pending deletion during the grace period.
    CancelDeletion(ctx context.Context, scope *shared.FamilyScope) error

    // ProcessDeletion processes deletion requests whose grace period has expired
    // or that are stuck in processing status (retry). Called by the recurring background job.
    ProcessDeletion(ctx context.Context) error

    // ProcessSingleDeletion processes a specific deletion request by ID.
    // Called by the background job worker for COPPA immediate deletions.
    // Verifies familyID matches the deletion request as a safety check.
    ProcessSingleDeletion(ctx context.Context, deletionID uuid.UUID, familyID uuid.UUID) error

    // === Account Recovery ===

    // InitiateRecovery initiates account recovery (unauthenticated).
    InitiateRecovery(ctx context.Context, req *InitiateRecoveryInput) (uuid.UUID, error)

    // GetRecoveryStatus checks recovery request status.
    GetRecoveryStatus(ctx context.Context, recoveryID uuid.UUID) (*RecoveryStatusResponse, error)

    // === Session Management ===

    // ListSessions lists active sessions for the current user.
    ListSessions(ctx context.Context, auth *shared.AuthContext) ([]SessionInfo, error)

    // RevokeSession revokes a specific session.
    RevokeSession(ctx context.Context, auth *shared.AuthContext, sessionID string) error

    // RevokeAllSessions revokes all sessions except the current one ("sign out everywhere").
    // Returns count of revoked sessions.
    RevokeAllSessions(ctx context.Context, auth *shared.AuthContext) (uint32, error)
}
```

---

## §6 Repository Interfaces

```go
// internal/lifecycle/ports.go (continued)

// ExportRequestRepository defines persistence operations for lifecycle_export_requests.
type ExportRequestRepository interface {
    Create(ctx context.Context, scope *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error)

    FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ExportRequest, error)

    ListByFamily(ctx context.Context, scope *shared.FamilyScope, pagination *PaginationParams) ([]ExportRequest, int64, error)

    UpdateStatus(ctx context.Context, id uuid.UUID, status ExportStatus, archiveKey *string, sizeBytes *int64, errorMessage *string) error
}

// DeletionRequestRepository defines persistence operations for lifecycle_deletion_requests.
type DeletionRequestRepository interface {
    Create(ctx context.Context, scope *shared.FamilyScope, input *CreateDeletionRequest) (*DeletionRequest, error)

    FindActiveByFamily(ctx context.Context, scope *shared.FamilyScope) (*DeletionRequest, error)

    // FindByID loads a deletion request by primary key (no FamilyScope — background job context).
    FindByID(ctx context.Context, id uuid.UUID) (*DeletionRequest, error)

    UpdateStatus(ctx context.Context, id uuid.UUID, status DeletionStatus) error

    UpdateDomainStatus(ctx context.Context, id uuid.UUID, domain string, completed bool) error

    Cancel(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error

    // FindReadyForDeletion returns deletion requests in grace_period status whose grace
    // period has expired, plus requests stuck in processing status (for retry).
    FindReadyForDeletion(ctx context.Context) ([]DeletionRequest, error)
}

// RecoveryRequestRepository defines persistence operations for lifecycle_recovery_requests.
type RecoveryRequestRepository interface {
    Create(ctx context.Context, input *CreateRecoveryRequest) (*RecoveryRequest, error)

    FindByID(ctx context.Context, id uuid.UUID) (*RecoveryRequest, error)

    UpdateStatus(ctx context.Context, id uuid.UUID, status RecoveryStatus, resolvedParentID *uuid.UUID) error
}
```

---

## §7 Domain Export & Deletion Contracts

Each domain that stores family data MUST implement these interfaces so lifecycle:: can
orchestrate cross-domain export and deletion:

```go
// internal/lifecycle/ports.go (continued)

// ExportHandler is implemented by each domain that has exportable family data.
// Registered at application startup.
type ExportHandler interface {
    // DomainName returns the domain identifier (e.g., "learning", "social", "compliance").
    DomainName() string

    // ExportFamilyData exports all family data for this domain in the requested format.
    // Returns a slice of (filename, contentBytes) pairs.
    ExportFamilyData(ctx context.Context, familyID uuid.UUID, format ExportFormat) ([]ExportFile, error)
}

// ExportFile represents a single file in a data export.
type ExportFile struct {
    Filename string
    Content  []byte
}

// DeletionHandler is implemented by each domain that stores deletable family data.
// Registered at application startup.
type DeletionHandler interface {
    // DomainName returns the domain identifier (e.g., "learning", "social", "compliance").
    DomainName() string

    // DeleteFamilyData deletes all family data for this domain.
    // MUST be idempotent — calling twice with the same family_id MUST NOT error.
    DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error

    // DeleteStudentData deletes data for a specific student within a family.
    DeleteStudentData(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error
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

```go
// internal/lifecycle/models.go

// --- Request types ---

// RequestExportInput represents a data export request.
type RequestExportInput struct {
    // Export format: "json" or "csv"
    Format *ExportFormat `json:"format,omitempty"`
    // Specific domains to include (nil = all)
    IncludeDomains []string `json:"include_domains,omitempty"`
}

// RequestDeletionInput represents an account deletion request.
type RequestDeletionInput struct {
    // Deletion type
    DeletionType DeletionType `json:"deletion_type" validate:"required"`
    // For student-specific deletion
    StudentID *uuid.UUID `json:"student_id,omitempty"`
    // Optional reason
    Reason *string `json:"reason,omitempty"`
}

// InitiateRecoveryInput represents an account recovery initiation request.
type InitiateRecoveryInput struct {
    // Email address associated with the account
    Email string `json:"email" validate:"required,email"`
}

// --- Response types ---

// ExportStatusResponse represents the status of a data export request.
type ExportStatusResponse struct {
    ID          uuid.UUID    `json:"id"`
    Status      ExportStatus `json:"status"`
    Format      ExportFormat `json:"format"`
    SizeBytes   *int64       `json:"size_bytes"`
    DownloadURL *string      `json:"download_url"` // pre-signed R2 URL, short TTL
    CreatedAt   time.Time    `json:"created_at"`
    CompletedAt *time.Time   `json:"completed_at"`
    ExpiresAt   time.Time    `json:"expires_at"`
}

// DeletionStatusResponse represents the status of a deletion request.
type DeletionStatusResponse struct {
    ID                uuid.UUID    `json:"id"`
    Status            DeletionStatus `json:"status"`
    DeletionType      DeletionType `json:"deletion_type"`
    GracePeriodEndsAt time.Time    `json:"grace_period_ends_at"`
    ExportOffered     bool         `json:"export_offered"`
    ExportRequestID   *uuid.UUID   `json:"export_request_id"`
    CreatedAt         time.Time    `json:"created_at"`
}

// RecoveryStatusResponse represents the status of a recovery request.
type RecoveryStatusResponse struct {
    ID                 uuid.UUID          `json:"id"`
    Status             RecoveryStatus     `json:"status"`
    VerificationMethod VerificationMethod `json:"verification_method"`
    CreatedAt          time.Time          `json:"created_at"`
}

// SessionInfo represents an active login session.
type SessionInfo struct {
    SessionID  string     `json:"session_id"`
    DeviceType *string    `json:"device_type"`  // "desktop", "mobile", "tablet"
    UserAgent  *string    `json:"user_agent"`
    IPAddress  *string    `json:"ip_address"`   // coarse — city-level only
    LastActive time.Time  `json:"last_active"`
    IsCurrent  bool       `json:"is_current"`
}

// --- Enums ---

type ExportFormat string

const (
    ExportFormatJSON ExportFormat = "json"
    ExportFormatCSV  ExportFormat = "csv"
)

type ExportStatus string

const (
    ExportStatusPending    ExportStatus = "pending"
    ExportStatusProcessing ExportStatus = "processing"
    ExportStatusCompleted  ExportStatus = "completed"
    ExportStatusFailed     ExportStatus = "failed"
    ExportStatusExpired    ExportStatus = "expired"
)

type DeletionType string

const (
    DeletionTypeFamily  DeletionType = "family"
    DeletionTypeStudent DeletionType = "student"
    DeletionTypeCoppa   DeletionType = "coppa"
)

type DeletionStatus string

const (
    DeletionStatusPending     DeletionStatus = "pending"
    DeletionStatusGracePeriod DeletionStatus = "grace_period"
    DeletionStatusProcessing  DeletionStatus = "processing"
    DeletionStatusCompleted   DeletionStatus = "completed"
    DeletionStatusCancelled   DeletionStatus = "cancelled"
)

type RecoveryStatus string

const (
    RecoveryStatusPending   RecoveryStatus = "pending"
    RecoveryStatusVerified  RecoveryStatus = "verified"
    RecoveryStatusEscalated RecoveryStatus = "escalated"
    RecoveryStatusCompleted RecoveryStatus = "completed"
    RecoveryStatusDenied    RecoveryStatus = "denied"
)

type VerificationMethod string

const (
    VerificationMethodEmail            VerificationMethod = "email"
    VerificationMethodSupportTicket    VerificationMethod = "support_ticket"
    VerificationMethodIdentityDocument VerificationMethod = "identity_document"
)
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
   c. Each handler returns `[]ExportFile` for the family
   d. Job assembles all files into a ZIP archive
   e. Uploads archive to Cloudflare R2 at `exports/{family_id}/{request_id}.zip`
   f. Sets status to `completed`, stores `archive_key` and `size_bytes`
   g. Publishes `DataExportCompleted` event -> `notify::` sends email with download link

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
5. Publishes `AccountDeletionRequested` event -> `notify::` sends confirmation email
6. During grace period: family can log in, use the platform, and cancel deletion
7. After grace period: `ProcessDeletionJob` (recurring daily check) transitions to `processing`
8. Deletion job:
   a. Revokes all Kratos sessions for the family
   b. Cancels active subscriptions via `billing.BillingService`
   c. Iterates over registered `DeletionHandler` implementations
   d. Each handler deletes its domain's data for the family
   e. Updates `domain_status` JSONB as each domain completes
   f. After all domains complete: sets status to `completed`
   g. Deletes the IAM family record last (cascade)
9. Publishes `AccountDeletionCompleted` event (for audit log)

### §10.2 Student Profile Deletion

Same workflow as family deletion, but scoped to a single student:
- Grace period: 7 days (shorter — less data)
- Only calls `DeleteStudentData()` on each handler
- Family account remains active
- Marketplace purchases associated with the student remain (legal retention)

### §10.3 COPPA Deletion

When a parent requests deletion of a child's data under COPPA:
- Grace period: 0 days (immediate processing per COPPA requirements)
- Status transitions directly from `pending` -> `processing`
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
- Revocation calls `iam.KratosAdapter.RevokeSession()` and clears Redis session cache
- Publishes `SessionRevoked` event for audit logging

### §12.3 Suspicious Session Detection

Phase 2: If concurrent sessions are detected from geographically distant locations within
a short time window (e.g., US East and Europe within 30 minutes), publish a
`SuspiciousSessionDetected` event -> `notify::` sends security alert email.

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
| `DataExportRequested` | `{ FamilyID, ExportID, Format }` | `notify::` (confirmation email) |
| `DataExportCompleted` | `{ FamilyID, ExportID, DownloadURL }` | `notify::` (download ready email) |
| `AccountDeletionRequested` | `{ FamilyID, DeletionType, GracePeriodEndsAt }` | `notify::` (confirmation email) |
| `AccountDeletionCompleted` | `{ FamilyID }` | Audit log |
| `CoppaDeleteRequested` | `{ FamilyID, StudentID }` | Audit log |
| `SessionRevoked` | `{ ParentID, SessionID, RevokeType }` | Audit log |

---

## §16 Error Types

```go
// internal/lifecycle/errors.go

import "errors"

var (
    ErrExportNotFound        = errors.New("export request not found")
    ErrExportExpired         = errors.New("export has expired")
    ErrDeletionAlreadyPending = errors.New("an active deletion request already exists")
    ErrGracePeriodExpired    = errors.New("cannot cancel deletion — grace period has ended")
    ErrNotPrimaryParent      = errors.New("only the primary parent can request family deletion")
    ErrRecoveryNotFound      = errors.New("recovery request not found or expired")
    ErrRecoveryExpired       = errors.New("recovery request has expired")
    ErrCannotRevokeCurrent   = errors.New("cannot revoke current session via this endpoint")
)
```

**HTTP mapping**:

| Error | HTTP Status |
|-------|-------------|
| `ErrExportNotFound` | 404 |
| `ErrExportExpired` | 410 Gone |
| `ErrDeletionAlreadyPending` | 409 Conflict |
| `ErrGracePeriodExpired` | 409 Conflict |
| `ErrNotPrimaryParent` | 403 Forbidden |
| `ErrRecoveryNotFound` | 404 |
| `ErrRecoveryExpired` | 410 Gone |
| `ErrCannotRevokeCurrent` | 400 Bad Request |

---

## §17 Cross-Domain Interactions

| Direction | Domain | Interaction |
|-----------|--------|-------------|
| lifecycle:: -> iam:: | Service call | Look up family, revoke Kratos sessions |
| lifecycle:: -> billing:: | Service call | Cancel active subscriptions before deletion |
| lifecycle:: -> all domains | ExportHandler interface | Each domain exports its family data |
| lifecycle:: -> all domains | DeletionHandler interface | Each domain deletes its family data |
| lifecycle:: -> notify:: | Domain event | Export/deletion notifications |
| lifecycle:: -> media:: | DeletionHandler | Delete R2 objects for family |
| admin:: -> lifecycle:: | Service call | View/manage recovery requests, retention policies |

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
internal/lifecycle/
├── handler.go          # Echo route handlers
├── service.go          # Orchestration logic
├── repository.go       # lifecycle_ table queries
├── models.go           # GORM models, DTOs (request/response)
├── ports.go            # Service + repository interface definitions
├── errors.go           # Sentinel error values
└── events.go           # Domain events (export/deletion lifecycle)
```
