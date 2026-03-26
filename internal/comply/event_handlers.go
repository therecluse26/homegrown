package comply

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/billing"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/learn"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Event handlers implementing shared.DomainEventHandler. [14-comply §5, CODING §8.4]
// Each handler asserts the concrete event type (value type assertion pattern),
// converts to a comply-local projection, and delegates to ComplianceService.

// ═══════════════════════════════════════════════════════════════════════════════
// ActivityLoggedHandler — learn::ActivityLogged → auto-attendance
// ═══════════════════════════════════════════════════════════════════════════════

// ActivityLoggedHandler records auto-attendance when a student logs a learning activity.
// Source: learn::events::ActivityLogged [06-learn §18.3]
type ActivityLoggedHandler struct {
	Service ComplianceService
}

func NewActivityLoggedHandler(svc ComplianceService) *ActivityLoggedHandler {
	return &ActivityLoggedHandler{Service: svc}
}

func (h *ActivityLoggedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(learn.ActivityLogged)
	if !ok {
		return fmt.Errorf("ActivityLoggedHandler: unexpected event type %T", event)
	}
	return h.Service.HandleActivityLogged(ctx, &ActivityLoggedEvent{
		FamilyID:        e.FamilyID,
		StudentID:       e.StudentID,
		ActivityID:      e.ActivityID,
		SubjectTags:     e.SubjectTags,
		DurationMinutes: e.DurationMinutes,
		ActivityDate:    e.ActivityDate,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// StudentDeletedHandler — iam::StudentDeleted → cascade delete student data
// ═══════════════════════════════════════════════════════════════════════════════

// StudentDeletedHandler cascades deletion of all compliance data for a deleted student.
// Source: iam::events::StudentDeleted [01-iam §13.3]
type StudentDeletedHandler struct {
	Service ComplianceService
}

func NewStudentDeletedHandler(svc ComplianceService) *StudentDeletedHandler {
	return &StudentDeletedHandler{Service: svc}
}

func (h *StudentDeletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.StudentDeleted)
	if !ok {
		return fmt.Errorf("StudentDeletedHandler: unexpected event type %T", event)
	}
	return h.Service.HandleStudentDeleted(ctx, &StudentDeletedEvent{
		FamilyID:  e.FamilyID,
		StudentID: e.StudentID,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// FamilyDeletionScheduledHandler — iam::FamilyDeletionScheduled → cascade delete all data
// ═══════════════════════════════════════════════════════════════════════════════

// FamilyDeletionScheduledHandler cascades deletion of all comply:: data for a family.
// Source: iam::events::FamilyDeletionScheduled [01-iam §13.3]
type FamilyDeletionScheduledHandler struct {
	Service ComplianceService
}

func NewFamilyDeletionScheduledHandler(svc ComplianceService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{Service: svc}
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.Service.HandleFamilyDeletionScheduled(ctx, &FamilyDeletionScheduledEvent{
		FamilyID:    e.FamilyID,
		DeleteAfter: e.DeleteAfter,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// SubscriptionCancelledHandler — billing::SubscriptionCancelled → preserve data
// ═══════════════════════════════════════════════════════════════════════════════

// SubscriptionCancelledHandler handles subscription cancellation (preserves data, no deletion).
// Source: billing::events::SubscriptionCancelled [10-billing §16.3]
type SubscriptionCancelledHandler struct {
	Service ComplianceService
}

func NewSubscriptionCancelledHandler(svc ComplianceService) *SubscriptionCancelledHandler {
	return &SubscriptionCancelledHandler{Service: svc}
}

func (h *SubscriptionCancelledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(billing.SubscriptionCancelled)
	if !ok {
		return fmt.Errorf("SubscriptionCancelledHandler: unexpected event type %T", event)
	}
	return h.Service.HandleSubscriptionCancelled(ctx, &SubscriptionCancelledEvent{
		FamilyID:    e.FamilyID,
		EffectiveAt: e.EffectiveAt,
	})
}
