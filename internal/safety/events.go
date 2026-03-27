package safety

import (
	"time"

	"github.com/google/uuid"
)

// Domain events published by the safety domain. [CODING §8.4, 11-safety §16.3]
// All events implement shared.DomainEvent.

// ContentReported is published when a user submits a content report.
type ContentReported struct {
	ReportID   uuid.UUID `json:"report_id"`
	FamilyID   uuid.UUID `json:"family_id"`
	TargetType string    `json:"target_type"`
	TargetID   uuid.UUID `json:"target_id"`
	Category   string    `json:"category"`
	Priority   string    `json:"priority"`
}

func (ContentReported) EventName() string { return "safety.ContentReported" }

// ModerationActionTaken is published when a moderation action is taken (EXCEPT for CSAM cases).
// Consumed by notify (notification to affected user), social (hide removed content),
// mkt (hide removed listing/review). [11-safety §16.3]
type ModerationActionTaken struct {
	ActionID       uuid.UUID  `json:"action_id"`
	ActionType     string     `json:"action_type"`
	TargetFamilyID uuid.UUID  `json:"target_family_id"`
	TargetType     *string    `json:"target_type,omitempty"`
	TargetID       *uuid.UUID `json:"target_id,omitempty"`
}

func (ModerationActionTaken) EventName() string { return "safety.ModerationActionTaken" }

// AccountSuspended is published when an account is suspended.
// Consumed by notify (suspension notification to user). [11-safety §16.3]
type AccountSuspended struct {
	FamilyID       uuid.UUID `json:"family_id"`
	SuspensionDays int32     `json:"suspension_days"`
	ExpiresAt      time.Time `json:"expires_at"`
}

func (AccountSuspended) EventName() string { return "safety.AccountSuspended" }

// AccountBanned is published when an account is banned (non-CSAM only).
type AccountBanned struct {
	FamilyID uuid.UUID `json:"family_id"`
	AdminID  uuid.UUID `json:"admin_id"`
	Reason   string    `json:"reason"`
}

func (AccountBanned) EventName() string { return "safety.AccountBanned" }

// AppealResolved is published when an appeal is resolved.
// Consumed by notify (appeal outcome notification to user). [11-safety §16.3]
type AppealResolved struct {
	AppealID uuid.UUID `json:"appeal_id"`
	FamilyID uuid.UUID `json:"family_id"`
	Status   string    `json:"status"` // "granted" or "denied"
}

func (AppealResolved) EventName() string { return "safety.AppealResolved" }

// UploadAutoRejectedNotification is published when an upload is auto-rejected by content
// policy (§11.2.1). Consumed by notify (generic rejection notification to uploader).
// Message: "Your upload was not published because it violates our content guidelines."
// Does NOT specify which policy was violated (prevents gaming). [11-safety §16.3]
type UploadAutoRejectedNotification struct {
	FamilyID uuid.UUID `json:"family_id"`
	UploadID uuid.UUID `json:"upload_id"`
}

func (UploadAutoRejectedNotification) EventName() string { return "safety.UploadAutoRejected" }

// ContentFlagged is published when content is flagged by automated scanning or manual review.
// Consumed by mkt:: (archive flagged listing), notify:: (creator notification). [11-safety §16.3]
type ContentFlagged struct {
	ContentKey string    `json:"content_key"`
	FamilyID   uuid.UUID `json:"family_id"`
	FlagType   string    `json:"flag_type"` // "csam" | "text_violation" | "manual_review"
}

func (ContentFlagged) EventName() string          { return "safety.ContentFlagged" }
func (e ContentFlagged) GetContentKey() string    { return e.ContentKey }
func (e ContentFlagged) GetFamilyID() uuid.UUID   { return e.FamilyID }
func (e ContentFlagged) GetFlagType() string      { return e.FlagType }
