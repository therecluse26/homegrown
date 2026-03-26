package lifecycle

import (
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Enums [15-data-lifecycle §8]
// ═══════════════════════════════════════════════════════════════════════════════

// ExportFormat represents the output format for data exports.
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
)

// ExportStatus represents the lifecycle state of an export request.
type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
	ExportStatusExpired    ExportStatus = "expired"
)

// DeletionType represents the scope of a deletion request.
type DeletionType string

const (
	DeletionTypeFamily  DeletionType = "family"
	DeletionTypeStudent DeletionType = "student"
	DeletionTypeCoppa   DeletionType = "coppa"
)

// DeletionStatus represents the lifecycle state of a deletion request.
type DeletionStatus string

const (
	DeletionStatusPending     DeletionStatus = "pending"
	DeletionStatusGracePeriod DeletionStatus = "grace_period"
	DeletionStatusProcessing  DeletionStatus = "processing"
	DeletionStatusCompleted   DeletionStatus = "completed"
	DeletionStatusCancelled   DeletionStatus = "cancelled"
)

// RecoveryStatus represents the lifecycle state of a recovery request.
type RecoveryStatus string

const (
	RecoveryStatusPending   RecoveryStatus = "pending"
	RecoveryStatusVerified  RecoveryStatus = "verified"
	RecoveryStatusEscalated RecoveryStatus = "escalated"
	RecoveryStatusCompleted RecoveryStatus = "completed"
	RecoveryStatusDenied    RecoveryStatus = "denied"
)

// VerificationMethod represents the method used to verify account recovery.
type VerificationMethod string

const (
	VerificationMethodEmail            VerificationMethod = "email"
	VerificationMethodSupportTicket    VerificationMethod = "support_ticket"
	VerificationMethodIdentityDocument VerificationMethod = "identity_document"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Domain Models (DB entities) [15-data-lifecycle §3]
// ═══════════════════════════════════════════════════════════════════════════════

// ExportRequest represents a lifecycle_export_requests row.
type ExportRequest struct {
	ID             uuid.UUID
	FamilyID       uuid.UUID
	RequestedBy    uuid.UUID
	Status         ExportStatus
	Format         ExportFormat
	IncludeDomains []string
	ArchiveKey     *string
	DownloadURL    *string
	SizeBytes      *int64
	CreatedAt      time.Time
	CompletedAt    *time.Time
	ExpiresAt      time.Time
}

// DeletionRequest represents a lifecycle_deletion_requests row.
type DeletionRequest struct {
	ID                uuid.UUID
	FamilyID          uuid.UUID
	RequestedBy       uuid.UUID
	Reason            *string
	DeletionType      DeletionType
	StudentID         *uuid.UUID
	Status            DeletionStatus
	GracePeriodEndsAt time.Time
	ExportOffered     bool
	ExportRequestID   *uuid.UUID
	DomainStatus      map[string]bool
	CreatedAt         time.Time
	CompletedAt       *time.Time
	CancelledAt       *time.Time
}

// RecoveryRequest represents a lifecycle_recovery_requests row.
type RecoveryRequest struct {
	ID                 uuid.UUID
	Email              string
	VerificationMethod VerificationMethod
	Status             RecoveryStatus
	ResolvedParentID   *uuid.UUID
	CreatedAt          time.Time
	ResolvedAt         *time.Time
	ExpiresAt          time.Time
}

// ═══════════════════════════════════════════════════════════════════════════════
// Request Types (input DTOs) [15-data-lifecycle §8]
// ═══════════════════════════════════════════════════════════════════════════════

// RequestExportInput represents a data export request from the API.
type RequestExportInput struct {
	Format         *ExportFormat `json:"format,omitempty"`
	IncludeDomains []string      `json:"include_domains,omitempty"`
}

// RequestDeletionInput represents an account deletion request from the API.
type RequestDeletionInput struct {
	DeletionType DeletionType `json:"deletion_type" validate:"required"`
	StudentID    *uuid.UUID   `json:"student_id,omitempty"`
	Reason       *string      `json:"reason,omitempty"`
}

// InitiateRecoveryInput represents an account recovery initiation request.
type InitiateRecoveryInput struct {
	Email string `json:"email" validate:"required,email"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Input Types [15-data-lifecycle §6]
// ═══════════════════════════════════════════════════════════════════════════════

// CreateExportRequest is the input for ExportRequestRepository.Create.
type CreateExportRequest struct {
	RequestedBy    uuid.UUID
	Format         ExportFormat
	IncludeDomains []string
}

// CreateDeletionRequest is the input for DeletionRequestRepository.Create.
type CreateDeletionRequest struct {
	RequestedBy       uuid.UUID
	DeletionType      DeletionType
	StudentID         *uuid.UUID
	Reason            *string
	GracePeriodEndsAt time.Time
	Status            DeletionStatus
}

// CreateRecoveryRequest is the input for RecoveryRequestRepository.Create.
type CreateRecoveryRequest struct {
	Email              string
	VerificationMethod VerificationMethod
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Types (output DTOs) [15-data-lifecycle §8]
// ═══════════════════════════════════════════════════════════════════════════════

// ExportStatusResponse represents the status of a data export request.
type ExportStatusResponse struct {
	ID          uuid.UUID    `json:"id"`
	Status      ExportStatus `json:"status"`
	Format      ExportFormat `json:"format"`
	SizeBytes   *int64       `json:"size_bytes"`
	DownloadURL *string      `json:"download_url"`
	CreatedAt   time.Time    `json:"created_at"`
	CompletedAt *time.Time   `json:"completed_at"`
	ExpiresAt   time.Time    `json:"expires_at"`
}

// ExportSummary is a lighter DTO used in list responses.
type ExportSummary struct {
	ID        uuid.UUID    `json:"id"`
	Status    ExportStatus `json:"status"`
	Format    ExportFormat `json:"format"`
	SizeBytes *int64       `json:"size_bytes"`
	CreatedAt time.Time    `json:"created_at"`
}

// DeletionStatusResponse represents the status of a deletion request.
type DeletionStatusResponse struct {
	ID                uuid.UUID      `json:"id"`
	Status            DeletionStatus `json:"status"`
	DeletionType      DeletionType   `json:"deletion_type"`
	GracePeriodEndsAt time.Time      `json:"grace_period_ends_at"`
	ExportOffered     bool           `json:"export_offered"`
	ExportRequestID   *uuid.UUID     `json:"export_request_id"`
	CreatedAt         time.Time      `json:"created_at"`
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
	SessionID  string    `json:"session_id"`
	DeviceType *string   `json:"device_type"`
	UserAgent  *string   `json:"user_agent"`
	IPAddress  *string   `json:"ip_address"`
	LastActive time.Time `json:"last_active"`
	IsCurrent  bool      `json:"is_current"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Pagination [15-data-lifecycle §8]
// ═══════════════════════════════════════════════════════════════════════════════

// PaginationParams contains pagination parameters for list queries.
type PaginationParams struct {
	Limit  int64 `json:"limit"`
	Offset int64 `json:"offset"`
}

// PaginatedExports is a paginated list of export summaries.
type PaginatedExports struct {
	Items []ExportSummary `json:"items"`
	Total int64           `json:"total"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Export File (used by ExportHandler) [15-data-lifecycle §7]
// ═══════════════════════════════════════════════════════════════════════════════

// ExportFile represents a single file in a data export.
type ExportFile struct {
	Filename string
	Content  []byte
}

// ═══════════════════════════════════════════════════════════════════════════════
// Job Payloads [15-data-lifecycle §14]
// ═══════════════════════════════════════════════════════════════════════════════

// DataExportJob is the job payload for processing a data export.
type DataExportJob struct {
	ExportID uuid.UUID `json:"export_id"`
	FamilyID uuid.UUID `json:"family_id"`
}

func (DataExportJob) TaskType() string { return "lifecycle:data_export" }

// ProcessDeletionJob is the job payload for processing account deletions past grace period.
type ProcessDeletionJob struct {
	DeletionID uuid.UUID `json:"deletion_id"`
	FamilyID   uuid.UUID `json:"family_id"`
}

func (ProcessDeletionJob) TaskType() string { return "lifecycle:process_deletion" }

// ═══════════════════════════════════════════════════════════════════════════════
// Grace Period Computation [15-data-lifecycle §10]
// ═══════════════════════════════════════════════════════════════════════════════

// GracePeriodFor returns the grace period duration for the given deletion type.
func GracePeriodFor(dt DeletionType) time.Duration {
	switch dt {
	case DeletionTypeFamily:
		return 30 * 24 * time.Hour // 30 days
	case DeletionTypeStudent:
		return 7 * 24 * time.Hour // 7 days
	case DeletionTypeCoppa:
		return 0 // immediate
	default:
		return 30 * 24 * time.Hour // default to family grace period
	}
}
