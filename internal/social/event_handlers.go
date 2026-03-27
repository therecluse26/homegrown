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

// ─── CoParentRemovedHandler ──────────────────────────────────────────────────

// CoParentRemovedHandler handles iam.CoParentRemoved by disassociating posts from the removed co-parent. [05-social §17.4]
type CoParentRemovedHandler struct{ svc SocialService }

func NewCoParentRemovedHandler(svc SocialService) *CoParentRemovedHandler {
	return &CoParentRemovedHandler{svc: svc}
}

func (h *CoParentRemovedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.CoParentRemoved)
	if !ok {
		return fmt.Errorf("social.CoParentRemovedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleCoParentRemoved(ctx, e.FamilyID, e.CoParentID)
}
