package admin

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [16-admin §5]
// ═══════════════════════════════════════════════════════════════════════════════

// AdminService defines the admin domain's service interface.
type AdminService interface {
	// === User Management ===

	// SearchUsers searches users by email, name, or family ID.
	SearchUsers(ctx context.Context, auth *shared.AuthContext, query *UserSearchQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error)

	// GetUserDetail returns detailed user info (family + parents + students + subscription + flags).
	GetUserDetail(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID) (*AdminUserDetail, error)

	// GetUserAuditTrail returns audit trail for a specific family.
	GetUserAuditTrail(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AuditLogEntry], error)

	// SuspendUser suspends a family account. [16-admin §4]
	SuspendUser(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, reason string) error

	// UnsuspendUser lifts a suspension on a family account. [16-admin §4]
	UnsuspendUser(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID) error

	// BanUser permanently bans a family account. [16-admin §4]
	BanUser(ctx context.Context, auth *shared.AuthContext, familyID uuid.UUID, reason string) error

	// === Feature Flags ===

	// ListFlags lists all feature flags.
	ListFlags(ctx context.Context, auth *shared.AuthContext) ([]FeatureFlag, error)

	// GetFlag returns a single feature flag by key. [16-admin §4]
	GetFlag(ctx context.Context, auth *shared.AuthContext, key string) (*FeatureFlag, error)

	// CreateFlag creates a new feature flag.
	CreateFlag(ctx context.Context, auth *shared.AuthContext, input *CreateFlagInput) (*FeatureFlag, error)

	// UpdateFlag updates a feature flag.
	UpdateFlag(ctx context.Context, auth *shared.AuthContext, key string, input *UpdateFlagInput) (*FeatureFlag, error)

	// DeleteFlag deletes a feature flag.
	DeleteFlag(ctx context.Context, auth *shared.AuthContext, key string) error

	// IsFlagEnabled evaluates whether a flag is enabled for a specific family.
	// Used by other domains to check feature flags at runtime. [16-admin §10.2]
	IsFlagEnabled(ctx context.Context, key string, familyID *uuid.UUID) (bool, error)

	// === Moderation Queue ===

	// GetModerationQueue returns the moderation review queue. [16-admin §4]
	GetModerationQueue(ctx context.Context, auth *shared.AuthContext, pagination *shared.PaginationParams) (*shared.PaginatedResponse[ModerationQueueItem], error)

	// GetModerationQueueItem returns a single moderation queue item. [16-admin §4]
	GetModerationQueueItem(ctx context.Context, auth *shared.AuthContext, itemID uuid.UUID) (*ModerationQueueItem, error)

	// TakeModerationAction performs an action on a moderation queue item. [16-admin §4]
	TakeModerationAction(ctx context.Context, auth *shared.AuthContext, itemID uuid.UUID, input *ModerationActionInput) error

	// === Methodology Config ===

	// ListMethodologies returns all methodology configurations for admin editing. [16-admin §4]
	ListMethodologies(ctx context.Context, auth *shared.AuthContext) ([]MethodologyConfig, error)

	// UpdateMethodologyConfig updates a methodology configuration. [16-admin §4]
	UpdateMethodologyConfig(ctx context.Context, auth *shared.AuthContext, slug string, input *UpdateMethodologyInput) (*MethodologyConfig, error)

	// === Lifecycle Management ===

	// GetPendingDeletions returns accounts pending deletion. [16-admin §4]
	GetPendingDeletions(ctx context.Context, auth *shared.AuthContext, pagination *shared.PaginationParams) (*shared.PaginatedResponse[DeletionSummary], error)

	// GetRecoveryRequests returns pending account recovery requests. [16-admin §4]
	GetRecoveryRequests(ctx context.Context, auth *shared.AuthContext, pagination *shared.PaginationParams) (*shared.PaginatedResponse[RecoverySummary], error)

	// ResolveRecoveryRequest approves or denies an account recovery request. [16-admin §4]
	ResolveRecoveryRequest(ctx context.Context, auth *shared.AuthContext, requestID uuid.UUID, approved bool) error

	// === System Health ===

	// GetSystemHealth returns aggregated system health status.
	GetSystemHealth(ctx context.Context, auth *shared.AuthContext) (*SystemHealthResponse, error)

	// GetJobStatus returns background job queue status.
	GetJobStatus(ctx context.Context, auth *shared.AuthContext) (*JobStatusResponse, error)

	// GetDeadLetterJobs returns dead-letter queue contents.
	GetDeadLetterJobs(ctx context.Context, auth *shared.AuthContext, pagination *shared.PaginationParams) (*shared.PaginatedResponse[DeadLetterJob], error)

	// RetryDeadLetterJob retries a dead-letter job.
	RetryDeadLetterJob(ctx context.Context, auth *shared.AuthContext, jobID string) error

	// === Audit Log ===

	// SearchAuditLog searches/filters the admin audit log.
	SearchAuditLog(ctx context.Context, auth *shared.AuthContext, query *AuditLogQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AuditLogEntry], error)

	// LogAction records an admin action (called internally by other admin methods).
	LogAction(ctx context.Context, auth *shared.AuthContext, action *AdminAction) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [16-admin §6]
// ═══════════════════════════════════════════════════════════════════════════════

// FeatureFlagRepository defines persistence operations for admin_feature_flags.
// Not family-scoped — feature flags are platform-wide. [16-admin §3.2]
type FeatureFlagRepository interface {
	ListAll(ctx context.Context) ([]FeatureFlag, error)

	FindByKey(ctx context.Context, key string) (*FeatureFlag, error)

	Create(ctx context.Context, input *CreateFlagInput, adminID uuid.UUID) (*FeatureFlag, error)

	Update(ctx context.Context, key string, input *UpdateFlagInput, adminID uuid.UUID) (*FeatureFlag, error)

	Delete(ctx context.Context, key string) error
}

// AuditLogRepository defines persistence operations for admin_audit_log.
// Append-only — no Update or Delete operations. [16-admin §3.2, §8.1]
type AuditLogRepository interface {
	// Create appends a new audit log entry.
	Create(ctx context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error)

	// Search searches audit log with filters.
	Search(ctx context.Context, query *AuditLogQuery, pagination *shared.PaginationParams) ([]AuditLogEntry, error)

	// FindByTarget returns audit entries for a specific target.
	FindByTarget(ctx context.Context, targetType string, targetID uuid.UUID, pagination *shared.PaginationParams) ([]AuditLogEntry, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Infrastructure Interfaces [16-admin §11]
// ═══════════════════════════════════════════════════════════════════════════════

// HealthChecker abstracts connectivity checks for all critical dependencies. [16-admin §11.1]
type HealthChecker interface {
	CheckAll(ctx context.Context) []ComponentHealth
}

// JobInspector abstracts background job queue inspection. [16-admin §11.2]
type JobInspector interface {
	GetQueueStatus(ctx context.Context) (*JobStatusResponse, error)
	GetDeadLetterJobs(ctx context.Context, pagination *shared.PaginationParams) ([]DeadLetterJob, error)
	RetryDeadLetterJob(ctx context.Context, jobID string) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer-Defined Cross-Domain Interfaces [16-admin §14]
// ═══════════════════════════════════════════════════════════════════════════════

// IamServiceForAdmin is a consumer-defined interface for cross-domain calls to iam::.
// Implemented by a function adapter in main.go. [ARCH §4.4]
type IamServiceForAdmin interface {
	SearchUsers(ctx context.Context, query *UserSearchQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error)
	GetFamilyDetail(ctx context.Context, familyID uuid.UUID) (*AdminFamilyInfo, error)
	GetParents(ctx context.Context, familyID uuid.UUID) ([]AdminParentInfo, error)
	GetStudents(ctx context.Context, familyID uuid.UUID) ([]AdminStudentInfo, error)
}

// SafetyServiceForAdmin is a consumer-defined interface for cross-domain calls to safety::.
// Implemented by a function adapter in main.go. [ARCH §4.4]
type SafetyServiceForAdmin interface {
	GetModerationHistory(ctx context.Context, familyID uuid.UUID) ([]ModerationActionSummary, error)
	SuspendAccount(ctx context.Context, familyID uuid.UUID, reason string) error
	UnsuspendAccount(ctx context.Context, familyID uuid.UUID) error
	BanAccount(ctx context.Context, familyID uuid.UUID, reason string) error
	GetReviewQueue(ctx context.Context, pagination *shared.PaginationParams) ([]ModerationQueueItem, error)
	GetReviewQueueItem(ctx context.Context, itemID uuid.UUID) (*ModerationQueueItem, error)
	TakeModerationAction(ctx context.Context, itemID uuid.UUID, action string, reason string) error
}

// BillingServiceForAdmin is a consumer-defined interface for cross-domain calls to billing::.
// Implemented by a function adapter in main.go. [ARCH §4.4]
type BillingServiceForAdmin interface {
	GetSubscriptionInfo(ctx context.Context, familyID uuid.UUID) (*AdminSubscriptionInfo, error)
}

// MethodologyServiceForAdmin is a consumer-defined interface for cross-domain calls to method::.
// Implemented by a function adapter in main.go. [ARCH §4.4]
type MethodologyServiceForAdmin interface {
	ListMethodologies(ctx context.Context) ([]MethodologyConfig, error)
	UpdateMethodologyConfig(ctx context.Context, slug string, input *UpdateMethodologyInput) (*MethodologyConfig, error)
}

// LifecycleServiceForAdmin is a consumer-defined interface for cross-domain calls to iam:: lifecycle.
// Implemented by a function adapter in main.go. [ARCH §4.4]
type LifecycleServiceForAdmin interface {
	GetPendingDeletions(ctx context.Context, pagination *shared.PaginationParams) ([]DeletionSummary, error)
	GetRecoveryRequests(ctx context.Context, pagination *shared.PaginationParams) ([]RecoverySummary, error)
	ResolveRecoveryRequest(ctx context.Context, requestID uuid.UUID, approved bool) error
}
