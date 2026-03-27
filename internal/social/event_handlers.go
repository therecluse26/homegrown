package social

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/learn"
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

// ─── MilestoneAchievedHandler ────────────────────────────────────────────────

// MilestoneAchievedHandler handles learn.MilestoneAchieved by creating a milestone post. [05-social §17.4]
type MilestoneAchievedHandler struct {
	svc SocialService
}

// NewMilestoneAchievedHandler creates a new MilestoneAchievedHandler.
func NewMilestoneAchievedHandler(svc SocialService) *MilestoneAchievedHandler {
	return &MilestoneAchievedHandler{svc: svc}
}

func (h *MilestoneAchievedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(learn.MilestoneAchieved)
	if !ok {
		return fmt.Errorf("social.MilestoneAchievedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleMilestoneAchieved(ctx, e.FamilyID, MilestoneData{
		StudentName:   e.StudentName,
		MilestoneType: e.MilestoneType,
		Description:   e.Description,
	})
}

// ─── FamilyDeletionScheduledHandler ─────────────────────────────────────────

// FamilyDeletionScheduledHandler handles iam.FamilyDeletionScheduled by preparing cascade. [05-social §17.4]
type FamilyDeletionScheduledHandler struct {
	svc SocialService
}

// NewFamilyDeletionScheduledHandler creates a new FamilyDeletionScheduledHandler.
func NewFamilyDeletionScheduledHandler(svc SocialService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{svc: svc}
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("social.FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFamilyDeletionScheduled(ctx, e.FamilyID)
}

// ─── Deferred Handlers ──────────────────────────────────────────────────────

// CoParentRemovedHandler would handle iam.CoParentRemoved by disassociating posts.
// DEFERRED: iam.CoParentRemoved event does not exist yet. [05-social §17.4]
// When activated, register in main.go:
//   eventBus.Subscribe(reflect.TypeOf(iam.CoParentRemoved{}), social.NewCoParentRemovedHandler(socialSvc))
