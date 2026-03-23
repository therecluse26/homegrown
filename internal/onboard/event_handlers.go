package onboard

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/method"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── FamilyCreatedHandler ────────────────────────────────────────────────────

// FamilyCreatedHandler handles iam.FamilyCreated events by initializing
// the onboarding wizard for newly created families. [04-onboard §11.1]
type FamilyCreatedHandler struct {
	svc OnboardingService
}

// NewFamilyCreatedHandler creates a new FamilyCreatedHandler.
func NewFamilyCreatedHandler(svc OnboardingService) *FamilyCreatedHandler {
	return &FamilyCreatedHandler{svc: svc}
}

func (h *FamilyCreatedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyCreated)
	if !ok {
		return fmt.Errorf("FamilyCreatedHandler: unexpected event type %T", event)
	}
	return h.svc.InitializeWizard(ctx, e.FamilyID)
}

// ─── FamilyMethodologyChangedHandler ─────────────────────────────────────────

// FamilyMethodologyChangedHandler handles method.FamilyMethodologyChanged events
// by re-materializing onboarding guidance. [04-onboard §11.2]
type FamilyMethodologyChangedHandler struct {
	svc OnboardingService
}

// NewFamilyMethodologyChangedHandler creates a new FamilyMethodologyChangedHandler.
func NewFamilyMethodologyChangedHandler(svc OnboardingService) *FamilyMethodologyChangedHandler {
	return &FamilyMethodologyChangedHandler{svc: svc}
}

func (h *FamilyMethodologyChangedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(method.FamilyMethodologyChanged)
	if !ok {
		return fmt.Errorf("FamilyMethodologyChangedHandler: unexpected event type %T", event)
	}
	secondarySlugs := make([]string, len(e.SecondaryMethodologySlugs))
	for i, s := range e.SecondaryMethodologySlugs {
		secondarySlugs[i] = string(s)
	}
	return h.svc.HandleMethodologyChanged(ctx, e.FamilyID, string(e.PrimaryMethodologySlug), secondarySlugs)
}
