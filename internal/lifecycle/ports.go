package lifecycle

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer-Defined Cross-Domain Interfaces [MEMORY cross-domain patterns]
// ═══════════════════════════════════════════════════════════════════════════════

// IamServiceForLifecycle is a consumer-defined interface for cross-domain calls to iam::.
// Implemented by a function adapter in main.go over iam.Service / KratosAdapter.
// [15-data-lifecycle §17, MEMORY consumer-defined interfaces]
type IamServiceForLifecycle interface {
	// InitiateRecoveryFlow triggers the Kratos recovery flow for the given email.
	InitiateRecoveryFlow(ctx context.Context, email string) error

	// ListSessions returns active sessions for a parent.
	ListSessions(ctx context.Context, parentID uuid.UUID) ([]SessionInfo, error)

	// RevokeSession revokes a specific Kratos session.
	RevokeSession(ctx context.Context, sessionID string) error

	// RevokeAllSessions revokes all sessions except the current one. Returns count revoked.
	RevokeAllSessions(ctx context.Context, parentID uuid.UUID, currentSessionID string) (uint32, error)

	// RevokeFamilySessions revokes all Kratos sessions for a family (used during deletion).
	RevokeFamilySessions(ctx context.Context, familyID uuid.UUID) error
}

// BillingServiceForLifecycle is a consumer-defined interface for cross-domain calls to billing::.
// [15-data-lifecycle §17, MEMORY consumer-defined interfaces]
type BillingServiceForLifecycle interface {
	// CancelFamilySubscriptions cancels all active subscriptions for a family.
	CancelFamilySubscriptions(ctx context.Context, familyID uuid.UUID) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [15-data-lifecycle §5]
// ═══════════════════════════════════════════════════════════════════════════════

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

	// RevokeAllSessions revokes all sessions except the current one.
	// Returns count of revoked sessions.
	RevokeAllSessions(ctx context.Context, auth *shared.AuthContext) (uint32, error)

	// HandleFamilyDeletion accelerates any pending deletion request for a family
	// when the iam::FamilyDeletionScheduled event fires. [15-data-lifecycle §17]
	HandleFamilyDeletion(ctx context.Context, familyID uuid.UUID) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [15-data-lifecycle §6]
// ═══════════════════════════════════════════════════════════════════════════════

// ExportRequestRepository defines persistence for lifecycle_export_requests.
type ExportRequestRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, input *CreateExportRequest) (*ExportRequest, error)
	FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ExportRequest, error)
	ListByFamily(ctx context.Context, scope *shared.FamilyScope, pagination *PaginationParams) ([]ExportRequest, int64, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status ExportStatus, archiveKey *string, sizeBytes *int64, errorMessage *string) error
}

// DeletionRequestRepository defines persistence for lifecycle_deletion_requests.
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

// RecoveryRequestRepository defines persistence for lifecycle_recovery_requests.
type RecoveryRequestRepository interface {
	Create(ctx context.Context, input *CreateRecoveryRequest) (*RecoveryRequest, error)
	FindByID(ctx context.Context, id uuid.UUID) (*RecoveryRequest, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status RecoveryStatus, resolvedParentID *uuid.UUID) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Domain Export & Deletion Contracts [15-data-lifecycle §7]
// ═══════════════════════════════════════════════════════════════════════════════

// ExportHandler is implemented by each domain that has exportable family data.
// Registered at application startup.
type ExportHandler interface {
	// DomainName returns the domain identifier (e.g., "learning", "social").
	DomainName() string

	// ExportFamilyData exports all family data for this domain in the requested format.
	ExportFamilyData(ctx context.Context, familyID uuid.UUID, format ExportFormat) ([]ExportFile, error)
}

// DeletionHandler is implemented by each domain that stores deletable family data.
// Registered at application startup.
type DeletionHandler interface {
	// DomainName returns the domain identifier (e.g., "learning", "social").
	DomainName() string

	// DeleteFamilyData deletes all family data for this domain.
	// MUST be idempotent.
	DeleteFamilyData(ctx context.Context, familyID uuid.UUID) error

	// DeleteStudentData deletes data for a specific student within a family.
	DeleteStudentData(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error
}
