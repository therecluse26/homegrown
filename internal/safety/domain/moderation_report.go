package domain

import (
	"time"

	"github.com/google/uuid"
)

// ReportStatus represents the lifecycle state of a moderation report. [11-safety §11.6]
type ReportStatus string

const (
	ReportStatusPending            ReportStatus = "pending"
	ReportStatusInReview           ReportStatus = "in_review"
	ReportStatusResolvedActionTaken ReportStatus = "resolved_action_taken"
	ReportStatusResolvedNoAction   ReportStatus = "resolved_no_action"
	ReportStatusDismissed          ReportStatus = "dismissed"
)

// ReportPriority represents the priority level of a moderation report. [11-safety §11.3]
type ReportPriority string

const (
	ReportPriorityCritical ReportPriority = "critical"
	ReportPriorityHigh     ReportPriority = "high"
	ReportPriorityNormal   ReportPriority = "normal"
)

// ModerationReport is the aggregate root for content moderation reports.
// All fields are unexported; state transitions happen via methods only. [ARCH §4.5]
type ModerationReport struct {
	id              uuid.UUID
	reporterFamilyID uuid.UUID
	reporterParentID uuid.UUID
	targetType      string
	targetID        uuid.UUID
	targetFamilyID  *uuid.UUID
	category        string
	description     *string
	priority        ReportPriority
	status          ReportStatus
	assignedAdminID *uuid.UUID
	resolvedAt      *time.Time
	createdAt       time.Time
	updatedAt       time.Time
}

// DerivePriority determines report priority from category. [11-safety §11.3]
func DerivePriority(category string) ReportPriority {
	switch category {
	case "csam_child_safety":
		return ReportPriorityCritical
	case "harassment":
		return ReportPriorityHigh
	default:
		return ReportPriorityNormal
	}
}

// NewModerationReport creates a new report in pending status with derived priority.
func NewModerationReport(
	id, reporterFamilyID, reporterParentID uuid.UUID,
	targetType string,
	targetID uuid.UUID,
	targetFamilyID *uuid.UUID,
	category string,
	description *string,
) *ModerationReport {
	now := time.Now().UTC()
	return &ModerationReport{
		id:               id,
		reporterFamilyID: reporterFamilyID,
		reporterParentID: reporterParentID,
		targetType:       targetType,
		targetID:         targetID,
		targetFamilyID:   targetFamilyID,
		category:         category,
		description:      description,
		priority:         DerivePriority(category),
		status:           ReportStatusPending,
		createdAt:        now,
		updatedAt:        now,
	}
}

// ReportFromPersistence reconstructs a ModerationReport from persisted data.
func ReportFromPersistence(
	id, reporterFamilyID, reporterParentID uuid.UUID,
	targetType string,
	targetID uuid.UUID,
	targetFamilyID *uuid.UUID,
	category string,
	description *string,
	priority ReportPriority,
	status ReportStatus,
	assignedAdminID *uuid.UUID,
	resolvedAt *time.Time,
	createdAt, updatedAt time.Time,
) *ModerationReport {
	return &ModerationReport{
		id:               id,
		reporterFamilyID: reporterFamilyID,
		reporterParentID: reporterParentID,
		targetType:       targetType,
		targetID:         targetID,
		targetFamilyID:   targetFamilyID,
		category:         category,
		description:      description,
		priority:         priority,
		status:           status,
		assignedAdminID:  assignedAdminID,
		resolvedAt:       resolvedAt,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
	}
}

// ─── Queries ─────────────────────────────────────────────────────────

func (r *ModerationReport) ID() uuid.UUID               { return r.id }
func (r *ModerationReport) ReporterFamilyID() uuid.UUID  { return r.reporterFamilyID }
func (r *ModerationReport) ReporterParentID() uuid.UUID  { return r.reporterParentID }
func (r *ModerationReport) TargetType() string           { return r.targetType }
func (r *ModerationReport) TargetID() uuid.UUID          { return r.targetID }
func (r *ModerationReport) TargetFamilyID() *uuid.UUID   { return r.targetFamilyID }
func (r *ModerationReport) Category() string             { return r.category }
func (r *ModerationReport) Description() *string         { return r.description }
func (r *ModerationReport) Priority() ReportPriority     { return r.priority }
func (r *ModerationReport) Status() ReportStatus         { return r.status }
func (r *ModerationReport) AssignedAdminID() *uuid.UUID  { return r.assignedAdminID }
func (r *ModerationReport) ResolvedAt() *time.Time       { return r.resolvedAt }
func (r *ModerationReport) CreatedAt() time.Time         { return r.createdAt }

// ─── State Transitions ──────────────────────────────────────────────

// Assign assigns an admin to review the report.
// Valid from: pending, in_review (reassign). Invalid from: resolved_*, dismissed.
func (r *ModerationReport) Assign(adminID uuid.UUID) error {
	switch r.status {
	case ReportStatusPending, ReportStatusInReview:
		r.status = ReportStatusInReview
		r.assignedAdminID = &adminID
		r.updatedAt = time.Now().UTC()
		return nil
	default:
		return ErrInvalidReportTransition
	}
}

// ResolveActionTaken resolves the report with action taken.
// Valid from: in_review only.
func (r *ModerationReport) ResolveActionTaken() error {
	if r.status != ReportStatusInReview {
		return ErrInvalidReportTransition
	}
	now := time.Now().UTC()
	r.status = ReportStatusResolvedActionTaken
	r.resolvedAt = &now
	r.updatedAt = now
	return nil
}

// ResolveNoAction resolves the report with no action taken.
// Valid from: in_review only.
func (r *ModerationReport) ResolveNoAction() error {
	if r.status != ReportStatusInReview {
		return ErrInvalidReportTransition
	}
	now := time.Now().UTC()
	r.status = ReportStatusResolvedNoAction
	r.resolvedAt = &now
	r.updatedAt = now
	return nil
}

// Dismiss dismisses the report.
// Valid from: pending, in_review. Invalid from: resolved_*, dismissed.
func (r *ModerationReport) Dismiss() error {
	switch r.status {
	case ReportStatusPending, ReportStatusInReview:
		now := time.Now().UTC()
		r.status = ReportStatusDismissed
		r.resolvedAt = &now
		r.updatedAt = now
		return nil
	default:
		return ErrInvalidReportTransition
	}
}
