package method

import "github.com/google/uuid"

// Domain events published by the method domain. [CODING §8.4, 02-method §11.3]
// All events implement shared.DomainEvent.
// Subscribers are registered in cmd/server/main.go via eventBus.Subscribe().

// FamilyMethodologyChanged is published when a family updates their methodology selection.
// Subscribers:
//   - learn:: recalculates family's active tool set
//   - social:: updates family profile methodology display
//   - notify:: sends "methodology updated" notification
//   - onboard:: updates getting-started roadmap if in progress
type FamilyMethodologyChanged struct {
	FamilyID                  uuid.UUID      `json:"family_id"`
	PrimaryMethodologySlug    MethodologyID  `json:"primary_methodology_slug"`
	SecondaryMethodologySlugs []MethodologyID `json:"secondary_methodology_slugs"`
}

func (FamilyMethodologyChanged) EventName() string { return "method.FamilyMethodologyChanged" }

// StudentMethodologyChanged is published when a student's methodology override is set or cleared.
// Subscribers:
//   - learn:: recalculates the student's active tool set
type StudentMethodologyChanged struct {
	FamilyID                uuid.UUID      `json:"family_id"`
	StudentID               uuid.UUID      `json:"student_id"`
	MethodologyOverrideSlug *MethodologyID `json:"methodology_override_slug"` // nil means override cleared
}

func (StudentMethodologyChanged) EventName() string { return "method.StudentMethodologyChanged" }

// MethodologyConfigUpdated is published when admin changes methodology definitions
// or tool activations (Phase 3+). All domains invalidate methodology config caches.
type MethodologyConfigUpdated struct{}

func (MethodologyConfigUpdated) EventName() string { return "method.MethodologyConfigUpdated" }
