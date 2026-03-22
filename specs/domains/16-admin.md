# Domain Spec 16 — Administration (admin::)

## §1 Overview

The Administration domain provides the **platform operations interface** — a protected
section of the application accessible only to platform administrators. It aggregates
data from all other domains into management views, provides operational tools (user
management, content moderation queue, system health, feature flags), and serves as the
operational backbone for day-to-day platform management. `[S§3.1.5, S§12.7]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/admin/` |
| **DB prefix** | `admin_` |
| **Complexity class** | Non-complex (no `domain/` subdirectory) `[ARCH §4.5]` |
| **External adapter** | None (aggregates from other domains' service interfaces) |
| **Key constraint** | All endpoints require `RequireAdmin` extractor `[00-core §13.3]`; admin actions MUST be immutably logged; MUST NOT bypass family-scoped isolation for data reads — use service interfaces |

**What admin:: owns**: Feature flag storage, admin audit log, system health aggregation
endpoints, admin-specific API endpoints that wrap other domains' services, admin dashboard
configuration.

**What admin:: does NOT own**: User accounts (owned by `iam::`). Content moderation pipeline
(owned by `safety::`). Marketplace listings (owned by `mkt::`). Billing records (owned by
`billing::`). Data export/deletion (owned by `lifecycle::`). The admin domain provides
*views into* and *actions on* other domains' data via their service interfaces.

**What admin:: delegates**: User suspension → `safety::SafetyService`. Content removal →
`safety::SafetyService`. Session revocation → `iam::KratosAdapter`. Methodology config
updates → `method::MethodologyService`. Feature flag evaluation → in-process (Redis-backed).
Notification delivery → `notify::` (via domain events).

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
-- Migration: YYYYMMDD_000001_create_admin_tables.rs
-- =============================================================================

-- Feature flags: simple key-value feature toggles
CREATE TABLE admin_feature_flags (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

All endpoints require `RequireAdmin` extractor.

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

```rust
#[async_trait]
pub trait AdminService: Send + Sync {
    // === User Management ===

    /// Search users by email, name, or family ID.
    async fn search_users(
        &self,
        auth: &AuthContext,
        query: UserSearchQuery,
        pagination: PaginationParams,
    ) -> Result<PaginatedResponse<AdminUserSummary>, AppError>;

    /// Get detailed user info (family + parents + students + subscription + flags).
    async fn get_user_detail(
        &self,
        auth: &AuthContext,
        family_id: Uuid,
    ) -> Result<AdminUserDetail, AppError>;

    /// Get audit trail for a specific family.
    async fn get_user_audit_trail(
        &self,
        auth: &AuthContext,
        family_id: Uuid,
        pagination: PaginationParams,
    ) -> Result<PaginatedResponse<AuditLogEntry>, AppError>;

    // === Feature Flags ===

    /// List all feature flags.
    async fn list_flags(
        &self,
        auth: &AuthContext,
    ) -> Result<Vec<FeatureFlag>, AppError>;

    /// Create a new feature flag.
    async fn create_flag(
        &self,
        auth: &AuthContext,
        input: CreateFlagInput,
    ) -> Result<FeatureFlag, AppError>;

    /// Update a feature flag.
    async fn update_flag(
        &self,
        auth: &AuthContext,
        key: &str,
        input: UpdateFlagInput,
    ) -> Result<FeatureFlag, AppError>;

    /// Delete a feature flag.
    async fn delete_flag(
        &self,
        auth: &AuthContext,
        key: &str,
    ) -> Result<(), AppError>;

    /// Evaluate whether a flag is enabled for a specific family.
    /// Used by other domains to check feature flags at runtime.
    async fn is_flag_enabled(
        &self,
        key: &str,
        family_id: Option<Uuid>,
    ) -> Result<bool, AppError>;

    // === System Health ===

    /// Get aggregated system health status.
    async fn get_system_health(
        &self,
        auth: &AuthContext,
    ) -> Result<SystemHealthResponse, AppError>;

    /// Get background job queue status.
    async fn get_job_status(
        &self,
        auth: &AuthContext,
    ) -> Result<JobStatusResponse, AppError>;

    /// Get dead-letter queue contents.
    async fn get_dead_letter_jobs(
        &self,
        auth: &AuthContext,
        pagination: PaginationParams,
    ) -> Result<PaginatedResponse<DeadLetterJob>, AppError>;

    /// Retry a dead-letter job.
    async fn retry_dead_letter_job(
        &self,
        auth: &AuthContext,
        job_id: &str,
    ) -> Result<(), AppError>;

    // === Audit Log ===

    /// Search/filter the admin audit log.
    async fn search_audit_log(
        &self,
        auth: &AuthContext,
        query: AuditLogQuery,
        pagination: PaginationParams,
    ) -> Result<PaginatedResponse<AuditLogEntry>, AppError>;

    /// Record an admin action (called internally by other admin methods).
    async fn log_action(
        &self,
        auth: &AuthContext,
        action: AdminAction,
    ) -> Result<(), AppError>;
}
```

---

## §6 Repository Interfaces

```rust
#[async_trait]
pub trait FeatureFlagRepository: Send + Sync {
    async fn list_all(&self) -> Result<Vec<FeatureFlag>, DbErr>;

    async fn find_by_key(&self, key: &str) -> Result<Option<FeatureFlag>, DbErr>;

    async fn create(&self, input: &CreateFlagInput, admin_id: Uuid) -> Result<FeatureFlag, DbErr>;

    async fn update(&self, key: &str, input: &UpdateFlagInput, admin_id: Uuid) -> Result<FeatureFlag, DbErr>;

    async fn delete(&self, key: &str) -> Result<(), DbErr>;
}

#[async_trait]
pub trait AuditLogRepository: Send + Sync {
    /// Append-only: create a new audit log entry.
    async fn create(&self, entry: &CreateAuditLogEntry) -> Result<AuditLogEntry, DbErr>;

    /// Search audit log with filters.
    async fn search(
        &self,
        query: &AuditLogQuery,
        pagination: &PaginationParams,
    ) -> Result<Vec<AuditLogEntry>, DbErr>;

    /// Get audit entries for a specific target.
    async fn find_by_target(
        &self,
        target_type: &str,
        target_id: Uuid,
        pagination: &PaginationParams,
    ) -> Result<Vec<AuditLogEntry>, DbErr>;
}
```

---

## §7 Models (DTOs)

```rust
// --- Request types ---

#[derive(Deserialize, ToSchema)]
pub struct UserSearchQuery {
    pub q: Option<String>,             // search by email or name
    pub family_id: Option<Uuid>,       // filter by family
    pub status: Option<String>,        // "active", "suspended", "banned"
    pub subscription: Option<String>,  // "free", "premium"
}

#[derive(Deserialize, ToSchema)]
pub struct CreateFlagInput {
    pub key: String,
    pub description: String,
    pub enabled: bool,
    pub rollout_percentage: Option<i16>,
    pub allowed_family_ids: Option<Vec<Uuid>>,
}

#[derive(Deserialize, ToSchema)]
pub struct UpdateFlagInput {
    pub enabled: Option<bool>,
    pub description: Option<String>,
    pub rollout_percentage: Option<Option<i16>>,  // None = don't change, Some(None) = remove
    pub allowed_family_ids: Option<Option<Vec<Uuid>>>,
}

#[derive(Deserialize, ToSchema)]
pub struct AuditLogQuery {
    pub admin_id: Option<Uuid>,
    pub action: Option<String>,
    pub target_type: Option<String>,
    pub target_id: Option<Uuid>,
    pub from_date: Option<DateTime<Utc>>,
    pub to_date: Option<DateTime<Utc>>,
}

// --- Response types ---

#[derive(Serialize, ToSchema)]
pub struct AdminUserSummary {
    pub family_id: Uuid,
    pub family_name: String,
    pub primary_parent_email: String,
    pub parent_count: i32,
    pub student_count: i32,
    pub subscription_tier: String,
    pub account_status: String,        // "active", "suspended", "banned", "deletion_pending"
    pub created_at: DateTime<Utc>,
    pub last_active_at: Option<DateTime<Utc>>,
}

#[derive(Serialize, ToSchema)]
pub struct AdminUserDetail {
    pub family: AdminFamilyInfo,
    pub parents: Vec<AdminParentInfo>,
    pub students: Vec<AdminStudentInfo>,
    pub subscription: Option<AdminSubscriptionInfo>,
    pub moderation_history: Vec<ModerationActionSummary>,
    pub recent_activity: UserActivitySummary,
}

#[derive(Serialize, ToSchema)]
pub struct FeatureFlag {
    pub id: Uuid,
    pub key: String,
    pub description: String,
    pub enabled: bool,
    pub rollout_percentage: Option<i16>,
    pub allowed_family_ids: Option<Vec<Uuid>>,
    pub created_by: Uuid,
    pub updated_by: Option<Uuid>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Serialize, ToSchema)]
pub struct SystemHealthResponse {
    pub status: String,                // "healthy", "degraded", "unhealthy"
    pub components: Vec<ComponentHealth>,
    pub checked_at: DateTime<Utc>,
}

#[derive(Serialize, ToSchema)]
pub struct ComponentHealth {
    pub name: String,                  // "database", "redis", "r2", "kratos"
    pub status: String,                // "healthy", "degraded", "unhealthy"
    pub latency_ms: Option<i64>,
    pub details: Option<String>,
}

#[derive(Serialize, ToSchema)]
pub struct JobStatusResponse {
    pub queues: Vec<QueueStatus>,
    pub dead_letter_count: i64,
}

#[derive(Serialize, ToSchema)]
pub struct QueueStatus {
    pub name: String,                  // "critical", "default", "low"
    pub pending: i64,
    pub processing: i64,
    pub completed_24h: i64,
    pub failed_24h: i64,
}

#[derive(Serialize, ToSchema)]
pub struct DeadLetterJob {
    pub id: String,
    pub queue: String,
    pub job_type: String,
    pub payload: serde_json::Value,
    pub error_message: String,
    pub failed_at: DateTime<Utc>,
    pub retry_count: i32,
}

#[derive(Serialize, ToSchema)]
pub struct AuditLogEntry {
    pub id: Uuid,
    pub admin_id: Uuid,
    pub admin_email: Option<String>,   // joined from iam_parents
    pub action: String,
    pub target_type: String,
    pub target_id: Option<Uuid>,
    pub details: serde_json::Value,
    pub created_at: DateTime<Utc>,
}
```

---

## §8 Admin Audit Logging (Domain Deep-Dive 1)

Every admin action MUST produce an immutable audit log entry. This is enforced at
the service layer — all admin operations call `log_action()` as part of their execution.

### §8.1 Audit Invariants

- Audit records MUST NOT be updated or deleted. The table has no UPDATE/DELETE RLS policies.
- Every audit entry MUST include: admin ID, action type, target type, target ID, and details.
- Admin IP address and user agent are captured for security forensics.
- Audit log MUST be retained for 2 years per the retention policy `[15-data-lifecycle §11]`.

### §8.2 Audit Integration Pattern

```rust
// In AdminServiceImpl — every action follows this pattern:
pub async fn suspend_user(&self, auth: &AuthContext, family_id: Uuid, reason: &str)
    -> Result<(), AppError>
{
    // 1. Perform the action (delegate to safety::)
    self.safety_service.suspend_account(family_id, reason).await?;

    // 2. Log the action (append-only, immutable)
    self.audit_repo.create(&CreateAuditLogEntry {
        admin_id: auth.parent_id,
        action: "user_suspend",
        target_type: "family",
        target_id: Some(family_id),
        details: json!({ "reason": reason }),
        ip_address: auth.ip_address.clone(),
        user_agent: auth.user_agent.clone(),
    }).await?;

    // 3. Publish event for notifications
    self.event_bus.publish(AdminActionTaken {
        action: "user_suspend",
        target_family_id: Some(family_id),
    }).await;

    Ok(())
}
```

---

## §9 Moderation Queue Integration (Domain Deep-Dive 2)

The admin domain does NOT own the moderation pipeline — that belongs to `safety::`
`[11-safety §11]`. Admin provides the **API surface** for the admin dashboard to interact
with the moderation queue.

### §9.1 Delegation Pattern

```
Admin Dashboard (React) → admin:: handlers → safety::SafetyService
```

- `GET /v1/admin/moderation/queue` calls `safety::SafetyService::get_review_queue()`
- `POST /v1/admin/moderation/queue/:id/action` calls `safety::SafetyService::take_moderation_action()`
- Admin wraps each call with audit logging (§8)
- Safety domain owns the actual moderation state machine and policy enforcement

### §9.2 Why Admin Wraps Safety

The `RequireAdmin` extractor is defined in `00-core` and consumed by admin::. Safety's own
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

```rust
impl AdminServiceImpl {
    /// Check if a feature flag is enabled for a given family.
    /// Used by other domains at request time.
    pub async fn is_flag_enabled(
        &self,
        key: &str,
        family_id: Option<Uuid>,
    ) -> Result<bool, AppError> {
        // 1. Check Redis cache first (1-minute TTL)
        let cache_key = format!("flag:{}", key);
        if let Some(cached) = self.redis.get::<FeatureFlag>(&cache_key).await? {
            return Ok(Self::evaluate_flag(&cached, family_id));
        }

        // 2. Fall back to database
        let flag = self.flag_repo.find_by_key(key).await?
            .ok_or(AppError::NotFound)?;

        // 3. Cache for 1 minute
        self.redis.set_ex(&cache_key, &flag, 60).await?;

        Ok(Self::evaluate_flag(&flag, family_id))
    }

    fn evaluate_flag(flag: &FeatureFlag, family_id: Option<Uuid>) -> bool {
        if !flag.enabled {
            return false;
        }

        // If allowlist exists and family is specified, check membership
        if let (Some(ref allowed), Some(fid)) = (&flag.allowed_family_ids, family_id) {
            if !allowed.is_empty() {
                return allowed.contains(&fid);
            }
        }

        // If percentage rollout, hash family_id for deterministic bucket
        if let (Some(pct), Some(fid)) = (flag.rollout_percentage, family_id) {
            let hash = crc32fast::hash(fid.as_bytes()) % 100;
            return (hash as i16) < pct;
        }

        true
    }
}
```

### §10.3 Usage by Other Domains

```rust
// In any domain's service:
if self.admin_service.is_flag_enabled("new_quiz_builder", Some(family_id)).await? {
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
It is accessed via `/admin/*` routes and protected by the `RequireAdmin` extractor on the
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

```rust
#[derive(Debug, thiserror::Error)]
pub enum AdminError {
    #[error("Feature flag not found: {0}")]
    FlagNotFound(String),

    #[error("Feature flag key already exists: {0}")]
    FlagAlreadyExists(String),

    #[error("Invalid flag key format")]
    InvalidFlagKey,

    #[error("User not found")]
    UserNotFound,

    #[error("Dead-letter job not found")]
    DeadLetterJobNotFound,

    #[error("Database error")]
    Database(#[from] sea_orm::DbErr),
}
```

**HTTP mapping**:

| Error | HTTP Status |
|-------|-------------|
| `FlagNotFound` | 404 |
| `FlagAlreadyExists` | 409 Conflict |
| `InvalidFlagKey` | 400 Bad Request |
| `UserNotFound` | 404 |
| `DeadLetterJobNotFound` | 404 |

---

## §14 Cross-Domain Interactions

| Direction | Domain | Interaction |
|-----------|--------|-------------|
| admin:: → iam:: | Service call | User lookup, family details |
| admin:: → safety:: | Service call | Moderation queue, user suspension/ban |
| admin:: → method:: | Service call | Methodology config read/update |
| admin:: → billing:: | Service call | Subscription details for user view |
| admin:: → lifecycle:: | Service call | Deletion and recovery management |
| admin:: → learn:: | Service call | Activity counts for user summary |
| admin:: → social:: | Service call | Post/comment counts for user summary |
| All domains → admin:: | Service call | Feature flag evaluation (`is_flag_enabled`) |

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

- [ ] All admin endpoints require `RequireAdmin` extractor
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
src/admin/
├── mod.rs              # Re-exports
├── handlers.rs         # Axum route handlers
├── service.rs          # Admin service orchestration
├── repository.rs       # admin_ table queries (flags, audit log)
├── models.rs           # DTOs (request/response)
├── ports.rs            # Service + repository trait definitions
└── entities/           # SeaORM-generated (admin_ tables)
```
