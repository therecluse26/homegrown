# Domain Spec 16 — Administration (admin::)

## §1 Overview

The Administration domain provides the **platform operations interface** — a protected
section of the application accessible only to platform administrators. It aggregates
data from all other domains into management views, provides operational tools (user
management, content moderation queue, system health, feature flags), and serves as the
operational backbone for day-to-day platform management. `[S§3.1.5, S§12.7]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/admin/` |
| **DB prefix** | `admin_` |
| **Complexity class** | Non-complex `[ARCH §4.5]` |
| **External adapter** | None (aggregates from other domains' service interfaces) |
| **Key constraint** | All endpoints require `RequireAdmin` middleware `[00-core §13.3]`; admin actions MUST be immutably logged; MUST NOT bypass family-scoped isolation for data reads — use service interfaces |

**What admin:: owns**: Feature flag storage, admin audit log, system health aggregation
endpoints, admin-specific API endpoints that wrap other domains' services, admin dashboard
configuration.

**What admin:: does NOT own**: User accounts (owned by `iam::`). Content moderation pipeline
(owned by `safety::`). Marketplace listings (owned by `mkt::`). Billing records (owned by
`billing::`). Data export/deletion (owned by `lifecycle::`). The admin domain provides
*views into* and *actions on* other domains' data via their service interfaces.

**What admin:: delegates**: User suspension -> `safety.SafetyService`. Content removal ->
`safety.SafetyService`. Session revocation -> `iam.KratosAdapter`. Methodology config
updates -> `method.MethodologyService`. Feature flag evaluation -> in-process (Redis-backed).
Notification delivery -> `notify::` (via domain events).

---

## §2 Requirements Traceability

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Platform Administrator role | `[S§3.1.5]` | §4, §5 |
| Moderation tooling (dashboard, queue, actions) | `[S§12.7]` | §9 (delegates to safety::) |
| Feature flags | `[S§19 Phase 2]` | §10 |
| System health monitoring | `[S§17.5]` | §11 |
| Admin audit logging | `[S§2.3]` | §3, §8 |

---

## §3 Database Schema

All tables use the `admin_` prefix. `[ARCH §5.1]`

### §3.1 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_admin_tables.sql
-- =============================================================================

-- Feature flags: simple key-value feature toggles
CREATE TABLE admin_feature_flags (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    key             VARCHAR(100) NOT NULL UNIQUE,
    description     TEXT NOT NULL,
    -- Flag state
    enabled         BOOLEAN NOT NULL DEFAULT false,
    -- Optional: percentage rollout (0-100, NULL = all-or-nothing)
    rollout_percentage SMALLINT CHECK (
        rollout_percentage IS NULL OR
        (rollout_percentage >= 0 AND rollout_percentage <= 100)
    ),
    -- Optional: restrict to specific family IDs (for beta testing)
    allowed_family_ids UUID[] DEFAULT NULL,
    -- Metadata
    created_by      UUID NOT NULL REFERENCES iam_parents(id),
    updated_by      UUID REFERENCES iam_parents(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Admin audit log: immutable record of all admin actions
CREATE TABLE admin_audit_log (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    admin_id        UUID NOT NULL REFERENCES iam_parents(id),
    action          VARCHAR(50) NOT NULL
                    CHECK (action IN (
                        'user_suspend', 'user_unsuspend', 'user_ban',
                        'content_remove', 'content_restore',
                        'flag_create', 'flag_update', 'flag_delete',
                        'method_config_update',
                        'retention_policy_update',
                        'recovery_resolve', 'recovery_deny',
                        'payout_approve', 'payout_reject',
                        'system_config_update'
                    )),
    -- What was acted upon
    target_type     VARCHAR(30) NOT NULL
                    CHECK (target_type IN (
                        'family', 'parent', 'student',
                        'post', 'comment', 'message', 'listing', 'review',
                        'feature_flag', 'methodology',
                        'retention_policy', 'recovery_request',
                        'payout', 'system'
                    )),
    target_id       UUID,                -- ID of the target entity
    -- Action details (what changed)
    details         JSONB NOT NULL DEFAULT '{}'::JSONB,
    -- IP and user agent for security
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Audit log is append-only — no UPDATE or DELETE policies
CREATE INDEX idx_admin_audit_log_admin ON admin_audit_log(admin_id);
CREATE INDEX idx_admin_audit_log_action ON admin_audit_log(action);
CREATE INDEX idx_admin_audit_log_target ON admin_audit_log(target_type, target_id);
CREATE INDEX idx_admin_audit_log_created ON admin_audit_log(created_at DESC);
```

### §3.2 Notes

- The audit log table has NO update/delete RLS policies — it is append-only by design.
- Feature flags use a simple boolean + optional percentage rollout model. This is
  intentionally simpler than full-featured systems like LaunchDarkly — we're a monolith
  with a single deployment target.
- Admin does NOT have family-scoped tables. All admin tables are platform-wide.

---

## §4 API Endpoints

All endpoints require `RequireAdmin` middleware.

```
# User Management (delegates to iam:: and safety::)
GET    /v1/admin/users                          # Search/list users
GET    /v1/admin/users/:id                      # Get user detail (family + parents + students)
POST   /v1/admin/users/:id/suspend              # Suspend account (delegates to safety::)
POST   /v1/admin/users/:id/unsuspend            # Remove suspension
POST   /v1/admin/users/:id/ban                  # Permanent ban
GET    /v1/admin/users/:id/audit                # Get audit trail for user

# Content Management (delegates to safety::)
GET    /v1/admin/moderation/queue               # Moderation review queue (from safety::)
GET    /v1/admin/moderation/queue/:id           # Get specific flagged item
POST   /v1/admin/moderation/queue/:id/action    # Take moderation action

# Feature Flags
GET    /v1/admin/flags                          # List all feature flags
POST   /v1/admin/flags                          # Create feature flag
GET    /v1/admin/flags/:key                     # Get flag details
PATCH  /v1/admin/flags/:key                     # Update flag (enable/disable/rollout)
DELETE /v1/admin/flags/:key                     # Delete flag

# System Health
GET    /v1/admin/system/health                  # Aggregated system health
GET    /v1/admin/system/jobs                    # Background job queue status
GET    /v1/admin/system/jobs/dead-letter        # Dead-letter queue contents
POST   /v1/admin/system/jobs/dead-letter/:id/retry  # Retry a dead-letter job

# Audit Log
GET    /v1/admin/audit                          # Search/filter audit log

# Methodology Config (delegates to method::)
GET    /v1/admin/methodologies                  # List methodologies with full config
PATCH  /v1/admin/methodologies/:slug            # Update methodology config

# Data Lifecycle (delegates to lifecycle::)
GET    /v1/admin/lifecycle/deletions            # View pending/processing deletions
GET    /v1/admin/lifecycle/recoveries           # View recovery requests
POST   /v1/admin/lifecycle/recoveries/:id/resolve  # Resolve recovery request
```

---

## §5 Service Interface

```go
// internal/admin/ports.go

// AdminService defines the admin domain's service interface.
type AdminService interface {
    // === User Management ===

    // SearchUsers searches users by email, name, or family ID.
    SearchUsers(ctx context.Context, auth *AuthContext, query *UserSearchQuery, pagination *PaginationParams) (*PaginatedResponse[AdminUserSummary], error)

    // GetUserDetail returns detailed user info (family + parents + students + subscription + flags).
    GetUserDetail(ctx context.Context, auth *AuthContext, familyID uuid.UUID) (*AdminUserDetail, error)

    // GetUserAuditTrail returns audit trail for a specific family.
    GetUserAuditTrail(ctx context.Context, auth *AuthContext, familyID uuid.UUID, pagination *PaginationParams) (*PaginatedResponse[AuditLogEntry], error)

    // === Feature Flags ===

    // ListFlags lists all feature flags.
    ListFlags(ctx context.Context, auth *AuthContext) ([]FeatureFlag, error)

    // CreateFlag creates a new feature flag.
    CreateFlag(ctx context.Context, auth *AuthContext, input *CreateFlagInput) (*FeatureFlag, error)

    // UpdateFlag updates a feature flag.
    UpdateFlag(ctx context.Context, auth *AuthContext, key string, input *UpdateFlagInput) (*FeatureFlag, error)

    // DeleteFlag deletes a feature flag.
    DeleteFlag(ctx context.Context, auth *AuthContext, key string) error

    // IsFlagEnabled evaluates whether a flag is enabled for a specific family.
    // Used by other domains to check feature flags at runtime.
    IsFlagEnabled(ctx context.Context, key string, familyID *uuid.UUID) (bool, error)

    // === System Health ===

    // GetSystemHealth returns aggregated system health status.
    GetSystemHealth(ctx context.Context, auth *AuthContext) (*SystemHealthResponse, error)

    // GetJobStatus returns background job queue status.
    GetJobStatus(ctx context.Context, auth *AuthContext) (*JobStatusResponse, error)

    // GetDeadLetterJobs returns dead-letter queue contents.
    GetDeadLetterJobs(ctx context.Context, auth *AuthContext, pagination *PaginationParams) (*PaginatedResponse[DeadLetterJob], error)

    // RetryDeadLetterJob retries a dead-letter job.
    RetryDeadLetterJob(ctx context.Context, auth *AuthContext, jobID string) error

    // === Audit Log ===

    // SearchAuditLog searches/filters the admin audit log.
    SearchAuditLog(ctx context.Context, auth *AuthContext, query *AuditLogQuery, pagination *PaginationParams) (*PaginatedResponse[AuditLogEntry], error)

    // LogAction records an admin action (called internally by other admin methods).
    LogAction(ctx context.Context, auth *AuthContext, action *AdminAction) error
}
```

---

## §6 Repository Interfaces

```go
// internal/admin/ports.go (continued)

// FeatureFlagRepository defines persistence operations for admin_feature_flags.
type FeatureFlagRepository interface {
    ListAll(ctx context.Context) ([]FeatureFlag, error)

    FindByKey(ctx context.Context, key string) (*FeatureFlag, error)

    Create(ctx context.Context, input *CreateFlagInput, adminID uuid.UUID) (*FeatureFlag, error)

    Update(ctx context.Context, key string, input *UpdateFlagInput, adminID uuid.UUID) (*FeatureFlag, error)

    Delete(ctx context.Context, key string) error
}

// AuditLogRepository defines persistence operations for admin_audit_log.
type AuditLogRepository interface {
    // Create appends a new audit log entry (append-only).
    Create(ctx context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error)

    // Search searches audit log with filters.
    Search(ctx context.Context, query *AuditLogQuery, pagination *PaginationParams) ([]AuditLogEntry, error)

    // FindByTarget returns audit entries for a specific target.
    FindByTarget(ctx context.Context, targetType string, targetID uuid.UUID, pagination *PaginationParams) ([]AuditLogEntry, error)
}
```

---

## §7 Models (DTOs)

```go
// internal/admin/models.go

// --- Request types ---

// UserSearchQuery represents search parameters for user lookup.
type UserSearchQuery struct {
    Q            *string    `json:"q"`              // search by email or name
    FamilyID     *uuid.UUID `json:"family_id"`      // filter by family
    Status       *string    `json:"status"`          // "active", "suspended", "banned"
    Subscription *string    `json:"subscription"`    // "free", "premium"
}

// CreateFlagInput represents input for creating a feature flag.
type CreateFlagInput struct {
    Key                string      `json:"key" validate:"required"`
    Description        string      `json:"description" validate:"required"`
    Enabled            bool        `json:"enabled"`
    RolloutPercentage  *int16      `json:"rollout_percentage"`
    AllowedFamilyIDs   []uuid.UUID `json:"allowed_family_ids"`
}

// UpdateFlagInput represents input for updating a feature flag.
type UpdateFlagInput struct {
    Enabled           *bool        `json:"enabled"`
    Description       *string      `json:"description"`
    RolloutPercentage **int16      `json:"rollout_percentage"`     // nil = don't change, *nil = remove
    AllowedFamilyIDs  *[]uuid.UUID `json:"allowed_family_ids"`    // nil = don't change
}

// AuditLogQuery represents search parameters for the audit log.
type AuditLogQuery struct {
    AdminID    *uuid.UUID `json:"admin_id"`
    Action     *string    `json:"action"`
    TargetType *string    `json:"target_type"`
    TargetID   *uuid.UUID `json:"target_id"`
    FromDate   *time.Time `json:"from_date"`
    ToDate     *time.Time `json:"to_date"`
}

// --- Response types ---

// AdminUserSummary represents a summary view of a user for admin listing.
type AdminUserSummary struct {
    FamilyID           uuid.UUID  `json:"family_id"`
    FamilyName         string     `json:"family_name"`
    PrimaryParentEmail string     `json:"primary_parent_email"`
    ParentCount        int32      `json:"parent_count"`
    StudentCount       int32      `json:"student_count"`
    SubscriptionTier   string     `json:"subscription_tier"`
    AccountStatus      string     `json:"account_status"` // "active", "suspended", "banned", "deletion_pending"
    CreatedAt          time.Time  `json:"created_at"`
    LastActiveAt       *time.Time `json:"last_active_at"`
}

// AdminUserDetail represents the detailed view of a user for admin inspection.
type AdminUserDetail struct {
    Family            AdminFamilyInfo            `json:"family"`
    Parents           []AdminParentInfo          `json:"parents"`
    Students          []AdminStudentInfo         `json:"students"`
    Subscription      *AdminSubscriptionInfo     `json:"subscription"`
    ModerationHistory []ModerationActionSummary  `json:"moderation_history"`
    RecentActivity    UserActivitySummary        `json:"recent_activity"`
}

// FeatureFlag represents a feature flag entity.
type FeatureFlag struct {
    ID                uuid.UUID   `json:"id"`
    Key               string      `json:"key"`
    Description       string      `json:"description"`
    Enabled           bool        `json:"enabled"`
    RolloutPercentage *int16      `json:"rollout_percentage"`
    AllowedFamilyIDs  []uuid.UUID `json:"allowed_family_ids"`
    CreatedBy         uuid.UUID   `json:"created_by"`
    UpdatedBy         *uuid.UUID  `json:"updated_by"`
    CreatedAt         time.Time   `json:"created_at"`
    UpdatedAt         time.Time   `json:"updated_at"`
}

// SystemHealthResponse represents the aggregated system health status.
type SystemHealthResponse struct {
    Status     string            `json:"status"` // "healthy", "degraded", "unhealthy"
    Components []ComponentHealth `json:"components"`
    CheckedAt  time.Time         `json:"checked_at"`
}

// ComponentHealth represents the health of a single system component.
type ComponentHealth struct {
    Name      string  `json:"name"`       // "database", "redis", "r2", "kratos"
    Status    string  `json:"status"`     // "healthy", "degraded", "unhealthy"
    LatencyMs *int64  `json:"latency_ms"`
    Details   *string `json:"details"`
}

// JobStatusResponse represents background job queue status.
type JobStatusResponse struct {
    Queues          []QueueStatus `json:"queues"`
    DeadLetterCount int64         `json:"dead_letter_count"`
}

// QueueStatus represents the status of a single job queue.
type QueueStatus struct {
    Name         string `json:"name"` // "critical", "default", "low"
    Pending      int64  `json:"pending"`
    Processing   int64  `json:"processing"`
    Completed24h int64  `json:"completed_24h"`
    Failed24h    int64  `json:"failed_24h"`
}

// DeadLetterJob represents a job in the dead-letter queue.
type DeadLetterJob struct {
    ID           string          `json:"id"`
    Queue        string          `json:"queue"`
    JobType      string          `json:"job_type"`
    Payload      json.RawMessage `json:"payload"`
    ErrorMessage string          `json:"error_message"`
    FailedAt     time.Time       `json:"failed_at"`
    RetryCount   int32           `json:"retry_count"`
}

// AuditLogEntry represents a single audit log entry.
type AuditLogEntry struct {
    ID         uuid.UUID       `json:"id"`
    AdminID    uuid.UUID       `json:"admin_id"`
    AdminEmail *string         `json:"admin_email"` // joined from iam_parents
    Action     string          `json:"action"`
    TargetType string          `json:"target_type"`
    TargetID   *uuid.UUID      `json:"target_id"`
    Details    json.RawMessage `json:"details"`
    CreatedAt  time.Time       `json:"created_at"`
}
```

---

## §8 Admin Audit Logging (Domain Deep-Dive 1)

Every admin action MUST produce an immutable audit log entry. This is enforced at
the service layer — all admin operations call `LogAction()` as part of their execution.

### §8.1 Audit Invariants

- Audit records MUST NOT be updated or deleted. The table has no UPDATE/DELETE RLS policies.
- Every audit entry MUST include: admin ID, action type, target type, target ID, and details.
- Admin IP address and user agent are captured for security forensics.
- Audit log MUST be retained for 2 years per the retention policy `[15-data-lifecycle §11]`.

### §8.2 Audit Integration Pattern

```go
// In AdminServiceImpl — every action follows this pattern:
func (s *AdminServiceImpl) SuspendUser(ctx context.Context, auth *AuthContext, familyID uuid.UUID, reason string) error {
    // 1. Perform the action (delegate to safety::)
    if err := s.safetyService.SuspendAccount(ctx, familyID, reason); err != nil {
        return fmt.Errorf("suspending account: %w", err)
    }

    // 2. Log the action (append-only, immutable)
    if _, err := s.auditRepo.Create(ctx, &CreateAuditLogEntry{
        AdminID:    auth.ParentID,
        Action:     "user_suspend",
        TargetType: "family",
        TargetID:   &familyID,
        Details:    json.RawMessage(fmt.Sprintf(`{"reason": %q}`, reason)),
        IPAddress:  auth.IPAddress,
        UserAgent:  auth.UserAgent,
    }); err != nil {
        return fmt.Errorf("logging audit: %w", err)
    }

    // 3. Publish event for notifications
    s.eventBus.Publish(ctx, &AdminActionTaken{
        Action:         "user_suspend",
        TargetFamilyID: &familyID,
    })

    return nil
}
```

---

## §9 Moderation Queue Integration (Domain Deep-Dive 2)

The admin domain does NOT own the moderation pipeline — that belongs to `safety::`
`[11-safety §11]`. Admin provides the **API surface** for the admin dashboard to interact
with the moderation queue.

### §9.1 Delegation Pattern

```
Admin Dashboard (React) → admin:: handlers → safety.SafetyService
```

- `GET /v1/admin/moderation/queue` calls `safety.SafetyService.GetReviewQueue()`
- `POST /v1/admin/moderation/queue/:id/action` calls `safety.SafetyService.TakeModerationAction()`
- Admin wraps each call with audit logging (§8)
- Safety domain owns the actual moderation state machine and policy enforcement

### §9.2 Why Admin Wraps Safety

The `RequireAdmin` middleware is defined in `00-core` and consumed by admin::. Safety's own
endpoints are for *system-level* operations (CSAM reporting, automated scanning). The
*human review* workflow routes through admin because it needs:
- Audit logging of every moderator action
- Admin-specific enrichment (user history, previous actions)
- Unified admin authentication and authorization

---

## §10 Feature Flags (Domain Deep-Dive 3)

### §10.1 Design

Feature flags are intentionally simple — a boolean toggle with optional percentage rollout
and family allowlist. This is sufficient for:
- Gradual rollout of new features
- Beta testing with specific families
- Kill switches for problematic features
- A/B testing (basic — two flags for variants)

### §10.2 Evaluation

```go
// IsFlagEnabled checks if a feature flag is enabled for a given family.
// Used by other domains at request time.
func (s *AdminServiceImpl) IsFlagEnabled(ctx context.Context, key string, familyID *uuid.UUID) (bool, error) {
    // 1. Check Redis cache first (1-minute TTL)
    cacheKey := fmt.Sprintf("flag:%s", key)
    var cached FeatureFlag
    if err := s.redis.Get(ctx, cacheKey, &cached); err == nil {
        return evaluateFlag(&cached, familyID), nil
    }

    // 2. Fall back to database
    flag, err := s.flagRepo.FindByKey(ctx, key)
    if err != nil {
        return false, fmt.Errorf("looking up flag: %w", err)
    }
    if flag == nil {
        return false, ErrFlagNotFound
    }

    // 3. Cache for 1 minute
    _ = s.redis.SetEx(ctx, cacheKey, flag, 60*time.Second)

    return evaluateFlag(flag, familyID), nil
}

func evaluateFlag(flag *FeatureFlag, familyID *uuid.UUID) bool {
    if !flag.Enabled {
        return false
    }

    // If allowlist exists and family is specified, check membership
    if len(flag.AllowedFamilyIDs) > 0 && familyID != nil {
        for _, allowed := range flag.AllowedFamilyIDs {
            if allowed == *familyID {
                return true
            }
        }
        return false
    }

    // If percentage rollout, hash family_id for deterministic bucket
    if flag.RolloutPercentage != nil && familyID != nil {
        hash := crc32.ChecksumIEEE(familyID[:]) % 100
        return int16(hash) < *flag.RolloutPercentage
    }

    return true
}
```

### §10.3 Usage by Other Domains

```go
// In any domain's service:
enabled, err := s.adminService.IsFlagEnabled(ctx, "new_quiz_builder", &familyID)
if err != nil {
    return err
}
if enabled {
    // Use new quiz builder flow
} else {
    // Use existing flow
}
```

Flag evaluation is cached in Redis (1-minute TTL) to avoid per-request database queries.
Flag changes propagate within 1 minute.

---

## §11 System Health (Domain Deep-Dive 4)

### §11.1 Health Check Components

The system health endpoint checks connectivity and latency to all critical dependencies:

| Component | Check | Healthy | Degraded | Unhealthy |
|-----------|-------|---------|----------|-----------|
| PostgreSQL | `SELECT 1` | < 50ms | 50-500ms | > 500ms or error |
| Redis | `PING` | < 10ms | 10-100ms | > 100ms or error |
| Cloudflare R2 | `HEAD` on test object | < 200ms | 200-1000ms | > 1000ms or error |
| Ory Kratos | `/health/alive` | < 100ms | 100-500ms | > 500ms or error |

Overall status:
- **healthy**: All components healthy
- **degraded**: One or more components degraded, none unhealthy
- **unhealthy**: One or more components unhealthy

### §11.2 Background Job Dashboard

- Queue depths (pending, processing) per tier (critical, default, low)
- Completed and failed counts (last 24 hours)
- Dead-letter queue contents with ability to inspect and retry

---

## §12 Admin Frontend Integration

The admin interface is a **section within the existing React SPA**, not a separate application.
It is accessed via `/admin/*` routes and protected by the `RequireAdmin` middleware on the
backend and a route guard on the frontend.

### §12.1 Route Structure

```
/admin                           # Dashboard overview
/admin/users                     # User search and list
/admin/users/:id                 # User detail
/admin/moderation                # Moderation queue
/admin/moderation/:id            # Review specific item
/admin/flags                     # Feature flag management
/admin/system                    # System health and jobs
/admin/system/dead-letter        # Dead-letter queue
/admin/audit                     # Audit log viewer
/admin/lifecycle/deletions       # Pending account deletions
/admin/lifecycle/recoveries      # Account recovery requests
```

### §12.2 Dashboard Overview

The admin dashboard landing page shows at-a-glance:
- System health status (green/yellow/red)
- Moderation queue depth (with priority breakdown)
- Dead-letter job count (if > 0, highlighted)
- Active user count (24h)
- Pending account deletions
- Pending recovery requests

---

## §13 Error Types

```go
// internal/admin/errors.go

import "errors"

var (
    ErrFlagNotFound       = errors.New("feature flag not found")
    ErrFlagAlreadyExists  = errors.New("feature flag key already exists")
    ErrInvalidFlagKey     = errors.New("invalid flag key format")
    ErrUserNotFound       = errors.New("user not found")
    ErrDeadLetterNotFound = errors.New("dead-letter job not found")
)
```

**HTTP mapping**:

| Error | HTTP Status |
|-------|-------------|
| `ErrFlagNotFound` | 404 |
| `ErrFlagAlreadyExists` | 409 Conflict |
| `ErrInvalidFlagKey` | 400 Bad Request |
| `ErrUserNotFound` | 404 |
| `ErrDeadLetterNotFound` | 404 |

---

## §14 Cross-Domain Interactions

| Direction | Domain | Interaction |
|-----------|--------|-------------|
| admin:: -> iam:: | Service call | User lookup, family details |
| admin:: -> safety:: | Service call | Moderation queue, user suspension/ban |
| admin:: -> method:: | Service call | Methodology config read/update |
| admin:: -> billing:: | Service call | Subscription details for user view |
| admin:: -> lifecycle:: | Service call | Deletion and recovery management |
| admin:: -> learn:: | Service call | Activity counts for user summary |
| admin:: -> social:: | Service call | Post/comment counts for user summary |
| All domains -> admin:: | Service call | Feature flag evaluation (`IsFlagEnabled`) |

---

## §15 Phase Scope

### Phase 1

- User search and detail view
- Moderation queue integration (delegates to safety::)
- System health endpoint
- Background job status and dead-letter queue viewer
- Admin audit log (all actions)
- Basic feature flags (boolean toggle)

### Phase 2

- Feature flag percentage rollout and family allowlists
- Methodology config editing
- Retention policy management
- Account recovery review
- Admin dashboard with at-a-glance metrics
- Operational runbook documentation

---

## §16 Verification Checklist

- [ ] All admin endpoints require `RequireAdmin` middleware
- [ ] Every admin action produces an immutable audit log entry
- [ ] Feature flag evaluation is cached in Redis with reasonable TTL
- [ ] System health endpoint checks all critical dependencies
- [ ] Dead-letter job retry re-enqueues to the correct queue
- [ ] Admin cannot bypass family-scoped isolation (uses service interfaces, not direct DB)
- [ ] Audit log entries cannot be updated or deleted (append-only test)
- [ ] Feature flag changes propagate within 1 minute

---

## §17 Module Structure

```
internal/admin/
├── handler.go          # Echo route handlers
├── service.go          # Admin service orchestration
├── repository.go       # admin_ table queries (flags, audit log)
├── models.go           # GORM models, DTOs (request/response)
├── ports.go            # Service + repository interface definitions
└── errors.go           # Sentinel error values
```
