package lifecycle

import (
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Domain Events [15-data-lifecycle §15]
// ═══════════════════════════════════════════════════════════════════════════════

// DataExportRequested is published when a family requests a data export.
type DataExportRequested struct {
	FamilyID uuid.UUID    `json:"family_id"`
	ExportID uuid.UUID    `json:"export_id"`
	Format   ExportFormat `json:"format"`
}

func (DataExportRequested) EventName() string { return "lifecycle.data_export_requested" }

// DataExportCompleted is published when a data export finishes successfully.
type DataExportCompleted struct {
	FamilyID    uuid.UUID `json:"family_id"`
	ExportID    uuid.UUID `json:"export_id"`
	DownloadURL string    `json:"download_url"`
}

func (DataExportCompleted) EventName() string { return "lifecycle.data_export_completed" }

// AccountDeletionRequested is published when a family requests account deletion.
type AccountDeletionRequested struct {
	FamilyID          uuid.UUID    `json:"family_id"`
	DeletionType      DeletionType `json:"deletion_type"`
	GracePeriodEndsAt time.Time    `json:"grace_period_ends_at"`
}

func (AccountDeletionRequested) EventName() string { return "lifecycle.account_deletion_requested" }

// AccountDeletionCompleted is published when account deletion finishes successfully.
type AccountDeletionCompleted struct {
	FamilyID uuid.UUID `json:"family_id"`
}

func (AccountDeletionCompleted) EventName() string { return "lifecycle.account_deletion_completed" }

// CoppaDeleteRequested is published when a COPPA deletion is requested (audit trail).
type CoppaDeleteRequested struct {
	FamilyID  uuid.UUID `json:"family_id"`
	StudentID uuid.UUID `json:"student_id"`
}

func (CoppaDeleteRequested) EventName() string { return "lifecycle.coppa_delete_requested" }

// SessionRevoked is published when one or more sessions are revoked.
type SessionRevoked struct {
	ParentID   uuid.UUID `json:"parent_id"`
	SessionID  string    `json:"session_id"`
	RevokeType string    `json:"revoke_type"` // "single" or "all"
}

func (SessionRevoked) EventName() string { return "lifecycle.session_revoked" }
