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
