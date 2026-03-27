package iam

import (
	"time"

	"github.com/google/uuid"
)

// Domain events published by the IAM domain. [CODING §8.4, §13.3]
// All events implement shared.DomainEvent.
// Subscribers are registered in cmd/server/main.go via eventBus.Subscribe().

// FamilyCreated is published after a new family + primary parent are created atomically
// via HandlePostRegistration. [§10.1]
// Subscribers:
//   - social:: creates social profile for family
//   - onboard:: starts onboarding wizard
type FamilyCreated struct {
	FamilyID uuid.UUID
	ParentID uuid.UUID
}

func (FamilyCreated) EventName() string { return "iam.FamilyCreated" }

// StudentCreated is published after a student profile is created. [§10, §4.3]
// Subscribers:
//   - learn:: initializes tool access for the student
type StudentCreated struct {
	FamilyID  uuid.UUID
	StudentID uuid.UUID
}

func (StudentCreated) EventName() string { return "iam.StudentCreated" }

// StudentDeleted is published after a student profile is deleted. [§10.5, §4.3]
// Subscribers:
//   - learn:: cleans up learning data and tool access
type StudentDeleted struct {
	FamilyID  uuid.UUID
	StudentID uuid.UUID
}

func (StudentDeleted) EventName() string { return "iam.StudentDeleted" }

// CoppaConsentGranted is published when COPPA consent transitions to Consented or ReVerified.
// [§9.2, §4.3]
// Subscribers:
//   - learn:: enables student-facing tools
type CoppaConsentGranted struct {
	FamilyID uuid.UUID
}

func (CoppaConsentGranted) EventName() string { return "iam.CoppaConsentGranted" }

// FamilyDeletionScheduled is published when a family requests account deletion. [§10, §4.3]
// The family's data will be purged after DeleteAfter.
// Subscribers:
//   - social:: removes social profile and content
//   - search:: removes family data from all search indexes
//   - billing:: cancels subscriptions
//   - learn:: removes learning data
//   - mkt:: removes marketplace data
//   - notify:: removes notification preferences and history
type FamilyDeletionScheduled struct {
	FamilyID    uuid.UUID
	DeleteAfter time.Time
}

func (FamilyDeletionScheduled) EventName() string { return "iam.FamilyDeletionScheduled" }

// InviteCreated is published when a co-parent invite is generated. [§5]
// Subscribers:
//   - notify:: sends the invite email with the accept link
type InviteCreated struct {
	FamilyID  uuid.UUID
	InviteID  uuid.UUID
	Email     string    // PII — never log [CODING §5.2]
	Token     string    // plaintext token for accept URL; never stored in DB [CODING §5.2]
	ExpiresAt time.Time
}

func (InviteCreated) EventName() string { return "iam.InviteCreated" }

// CoParentAdded is published when a co-parent is added to an existing family. [§4.3]
// Subscribers:
//   - social:: shares family posts with new co-parent
//   - notify:: welcomes new co-parent
type CoParentAdded struct {
	FamilyID     uuid.UUID
	CoParentID   uuid.UUID
	CoParentEmail string // PII — used by notify:: for welcome email; never log [CODING §5.2]
	CoParentName  string
}

func (CoParentAdded) EventName() string { return "iam.CoParentAdded" }

// CoParentRemoved is published when a co-parent is removed from a family. [§4.3]
// Subscribers:
//   - social:: disassociates posts from removed co-parent
type CoParentRemoved struct {
	FamilyID   uuid.UUID
	CoParentID uuid.UUID
}

func (CoParentRemoved) EventName() string { return "iam.CoParentRemoved" }

// PrimaryParentTransferred is published when primary ownership is transferred to another parent. [§4.3]
// Subscribers:
//   - billing:: updates Hyperswitch customer email to new primary
type PrimaryParentTransferred struct {
	FamilyID      uuid.UUID
	NewPrimaryID  uuid.UUID
	PrevPrimaryID uuid.UUID
}

func (PrimaryParentTransferred) EventName() string { return "iam.PrimaryParentTransferred" }
