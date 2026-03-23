package onboard

import "github.com/google/uuid"

// Domain events published by the onboard domain. [CODING §8.4, 04-onboard §11.3]
// All events implement shared.DomainEvent.
// Subscribers are registered in cmd/server/main.go via eventBus.Subscribe().

// OnboardingCompleted is published when the onboarding wizard finishes (completed or skipped).
// Subscribers:
//   - notify:: sends "welcome complete" or "setup skipped" email
//   - social:: may trigger initial social profile population
type OnboardingCompleted struct {
	FamilyID uuid.UUID `json:"family_id"`
	Skipped  bool      `json:"skipped"` // true if wizard was skipped, false if completed normally
}

func (OnboardingCompleted) EventName() string { return "onboard.OnboardingCompleted" }
