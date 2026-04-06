package lifecycle

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Custom DB Types [15-data-lifecycle §6, CODING §2.3]
// ═══════════════════════════════════════════════════════════════════════════════

// textArray is a PostgreSQL TEXT[] type for include_domains. [CODING §2.3]
type textArray []string

func (a textArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	return "{" + strings.Join(a, ",") + "}", nil
}

func (a *textArray) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}
	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("textArray.Scan: unsupported type %T", src)
	}
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*a = textArray{}
		return nil
	}
	parts := strings.Split(str, ",")
	result := make(textArray, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	*a = result
	return nil
}

// domainStatusMap is a JSONB map[string]bool for deletion domain_status. [15-data-lifecycle §6]
type domainStatusMap map[string]bool

func (m domainStatusMap) Value() (driver.Value, error) {
	if m == nil {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

func (m *domainStatusMap) Scan(src any) error {
	if src == nil {
		*m = domainStatusMap{}
		return nil
	}
	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("domainStatusMap.Scan: unsupported type %T", src)
	}
	result := make(domainStatusMap)
	if err := json.Unmarshal(b, &result); err != nil {
		return err
	}
	*m = result
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// GORM Row Models [15-data-lifecycle §6]
// ═══════════════════════════════════════════════════════════════════════════════

// exportRequestRow is the GORM model for lifecycle_export_requests.
type exportRequestRow struct {
	ID                uuid.UUID   `gorm:"column:id;primaryKey"`
	FamilyID          uuid.UUID   `gorm:"column:family_id"`
	RequestedBy       uuid.UUID   `gorm:"column:requested_by"`
	Status            string      `gorm:"column:status"`
	Format            string      `gorm:"column:format"`
	IncludeDomains    textArray   `gorm:"column:include_domains;type:text[]"`
	ArchiveKey        *string     `gorm:"column:archive_key"`
	DownloadExpiresAt *time.Time  `gorm:"column:download_expires_at"`
	SizeBytes         *int64      `gorm:"column:size_bytes"`
	ErrorMessage      *string     `gorm:"column:error_message"`
	CreatedAt         time.Time   `gorm:"column:created_at"`
	CompletedAt       *time.Time  `gorm:"column:completed_at"`
	ExpiresAt         time.Time   `gorm:"column:expires_at"`
}

func (exportRequestRow) TableName() string { return "lifecycle_export_requests" }

func (r exportRequestRow) toDomain() ExportRequest {
	return ExportRequest{
		ID:                r.ID,
		FamilyID:          r.FamilyID,
		RequestedBy:       r.RequestedBy,
		Status:            ExportStatus(r.Status),
		Format:            ExportFormat(r.Format),
		IncludeDomains:    []string(r.IncludeDomains),
		ArchiveKey:        r.ArchiveKey,
		DownloadExpiresAt: r.DownloadExpiresAt,
		SizeBytes:         r.SizeBytes,
		ErrorMessage:      r.ErrorMessage,
		CreatedAt:         r.CreatedAt,
		CompletedAt:       r.CompletedAt,
		ExpiresAt:         r.ExpiresAt,
	}
}

// deletionRequestRow is the GORM model for lifecycle_deletion_requests.
type deletionRequestRow struct {
	ID                 uuid.UUID        `gorm:"column:id;primaryKey"`
	FamilyID           uuid.UUID        `gorm:"column:family_id"`
	RequestedBy        uuid.UUID        `gorm:"column:requested_by"`
	Reason             *string          `gorm:"column:reason"`
	DeletionType       string           `gorm:"column:deletion_type"`
	StudentID          *uuid.UUID       `gorm:"column:student_id"`
	Status             string           `gorm:"column:status"`
	GracePeriodEndsAt  time.Time        `gorm:"column:grace_period_ends_at"`
	ExportOffered      bool             `gorm:"column:export_offered"`
	ExportRequestID    *uuid.UUID       `gorm:"column:export_request_id"`
	DomainStatus       domainStatusMap  `gorm:"column:domain_status;type:jsonb"`
	CreatedAt          time.Time        `gorm:"column:created_at"`
	CompletedAt        *time.Time       `gorm:"column:completed_at"`
	CancelledAt        *time.Time       `gorm:"column:cancelled_at"`
}

func (deletionRequestRow) TableName() string { return "lifecycle_deletion_requests" }

func (r deletionRequestRow) toDomain() DeletionRequest {
	return DeletionRequest{
		ID:                r.ID,
		FamilyID:          r.FamilyID,
		RequestedBy:       r.RequestedBy,
		Reason:            r.Reason,
		DeletionType:      DeletionType(r.DeletionType),
		StudentID:         r.StudentID,
		Status:            DeletionStatus(r.Status),
		GracePeriodEndsAt: r.GracePeriodEndsAt,
		ExportOffered:     r.ExportOffered,
		ExportRequestID:   r.ExportRequestID,
		DomainStatus:      map[string]bool(r.DomainStatus),
		CreatedAt:         r.CreatedAt,
		CompletedAt:       r.CompletedAt,
		CancelledAt:       r.CancelledAt,
	}
}

// recoveryRequestRow is the GORM model for lifecycle_recovery_requests.
type recoveryRequestRow struct {
	ID                 uuid.UUID  `gorm:"column:id;primaryKey"`
	Email              string     `gorm:"column:email"`
	VerificationMethod string     `gorm:"column:verification_method"`
	Status             string     `gorm:"column:status"`
	SupportTicketID    *string    `gorm:"column:support_ticket_id"`
	ResolvedParentID   *uuid.UUID `gorm:"column:resolved_parent_id"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	ResolvedAt         *time.Time `gorm:"column:resolved_at"`
	ExpiresAt          time.Time  `gorm:"column:expires_at"`
}

func (recoveryRequestRow) TableName() string { return "lifecycle_recovery_requests" }

func (r recoveryRequestRow) toDomain() RecoveryRequest {
	return RecoveryRequest{
		ID:                 r.ID,
		Email:              r.Email,
		VerificationMethod: VerificationMethod(r.VerificationMethod),
		Status:             RecoveryStatus(r.Status),
		SupportTicketID:    r.SupportTicketID,
		ResolvedParentID:   r.ResolvedParentID,
		CreatedAt:          r.CreatedAt,
		ResolvedAt:         r.ResolvedAt,
		ExpiresAt:          r.ExpiresAt,
	}
}

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
	ArchiveKey        *string
	DownloadURL       *string
	DownloadExpiresAt *time.Time
	SizeBytes         *int64
	ErrorMessage   *string
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
	SupportTicketID    *string
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

// DeletionSweepPayload is the cron job payload for the nightly deletion sweep.
// The payload is empty — the handler calls ProcessDeletion which iterates all
// expired grace-period requests itself. [15-data-lifecycle §10.1]
type DeletionSweepPayload struct{}

func (DeletionSweepPayload) TaskType() string { return "lifecycle:sweep_deletions" }

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
