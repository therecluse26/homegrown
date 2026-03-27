package lifecycle

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// FamilyDeletionScheduledHandler — iam::FamilyDeletionScheduled → accelerate deletion
// ═══════════════════════════════════════════════════════════════════════════════

// FamilyDeletionScheduledHandler accelerates any pending lifecycle deletion request
// when IAM schedules a family deletion (admin-initiated or COPPA).
// Transitions the deletion from grace_period → processing and enqueues the job.
// Source: iam::events::FamilyDeletionScheduled [01-iam §13.3, 15-data-lifecycle §17]
type FamilyDeletionScheduledHandler struct {
	Service LifecycleService
}

// NewFamilyDeletionScheduledHandler creates a handler for iam::FamilyDeletionScheduled.
func NewFamilyDeletionScheduledHandler(svc LifecycleService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{Service: svc}
}

// Handle implements shared.DomainEventHandler.
func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.Service.HandleFamilyDeletion(ctx, e.FamilyID)
}
