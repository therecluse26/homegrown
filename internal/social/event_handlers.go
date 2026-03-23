package social

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── FamilyCreatedHandler ────────────────────────────────────────────────────

// FamilyCreatedHandler handles iam.FamilyCreated events by creating
// a social profile for newly created families. [05-social §17.4]
type FamilyCreatedHandler struct {
	svc SocialService
}

// NewFamilyCreatedHandler creates a new FamilyCreatedHandler.
func NewFamilyCreatedHandler(svc SocialService) *FamilyCreatedHandler {
	return &FamilyCreatedHandler{svc: svc}
}

func (h *FamilyCreatedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyCreated)
	if !ok {
		return fmt.Errorf("social.FamilyCreatedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFamilyCreated(ctx, e.FamilyID)
}

// ─── Deferred Handlers ──────────────────────────────────────────────────────
// These handlers are defined but their event subscriptions are deferred until
// the required events exist in their respective domains.

// CoParentRemovedHandler would handle iam.CoParentRemoved by disassociating posts.
// DEFERRED: iam.CoParentRemoved event does not exist yet. [05-social §17.4]
// When activated, register in main.go:
//   eventBus.Subscribe(reflect.TypeOf(iam.CoParentRemoved{}), social.NewCoParentRemovedHandler(socialSvc))

// MilestoneAchievedHandler would handle learn.MilestoneAchieved by creating a milestone post.
// DEFERRED: learn:: domain not implemented. [05-social §17.4]

// FamilyDeletionScheduledHandler would handle iam.FamilyDeletionScheduled by preparing cascade.
// DEFERRED: iam.FamilyDeletionScheduled event does not exist yet. [05-social §17.4]
