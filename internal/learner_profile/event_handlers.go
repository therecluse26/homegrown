package learner_profile

import (
	"context"
	"fmt"
	"reflect"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── StudentDeletedHandler ────────────────────────────────────────────────────

// StudentDeletedHandler deletes the learner profile when a student is deleted.
// Belt-and-suspenders cleanup on top of ON DELETE CASCADE. [18-learner-profile §9]
type StudentDeletedHandler struct {
	svc LearnerProfileService
}

func NewStudentDeletedHandler(svc LearnerProfileService) *StudentDeletedHandler {
	return &StudentDeletedHandler{svc: svc}
}

func (h *StudentDeletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.StudentDeleted)
	if !ok {
		return fmt.Errorf("learner_profile.StudentDeletedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleStudentDeletion(ctx, e.StudentID)
}

// ─── FamilyDeletionScheduledHandler ──────────────────────────────────────────

// FamilyDeletionScheduledHandler deletes all learner profiles for a family
// when the family schedules account deletion. [18-learner-profile §9]
type FamilyDeletionScheduledHandler struct {
	svc LearnerProfileService
}

func NewFamilyDeletionScheduledHandler(svc LearnerProfileService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{svc: svc}
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("learner_profile.FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFamilyDeletion(ctx, shared.NewFamilyID(e.FamilyID))
}

// RegisterEventHandlers subscribes the learner profile domain to IAM lifecycle events.
func RegisterEventHandlers(bus *shared.EventBus, svc LearnerProfileService) {
	bus.Subscribe(reflect.TypeFor[iam.StudentDeleted](), NewStudentDeletedHandler(svc))
	bus.Subscribe(reflect.TypeFor[iam.FamilyDeletionScheduled](), NewFamilyDeletionScheduledHandler(svc))
}
