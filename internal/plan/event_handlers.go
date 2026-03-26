package plan

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/learn"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Event Handlers [17-planning §16]
// ═══════════════════════════════════════════════════════════════════════════════

// EventCancelledHandler handles social.EventCancelled events.
// When a social event is cancelled, linked schedule items are deleted. [17-planning §16]
type EventCancelledHandler struct {
	svc PlanningService
}

// NewEventCancelledHandler creates a new EventCancelledHandler.
func NewEventCancelledHandler(svc PlanningService) *EventCancelledHandler {
	return &EventCancelledHandler{svc: svc}
}

// Handle processes a social.EventCancelled event.
func (h *EventCancelledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(social.EventCancelled)
	if !ok {
		return fmt.Errorf("EventCancelledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleEventCancelled(ctx, e.EventID, e.GoingFamilyIDs)
}

// ActivityLoggedHandler handles learn.ActivityLogged events.
// When an activity is logged, matching schedule items may be marked as completed. [17-planning §16]
type ActivityLoggedHandler struct {
	svc PlanningService
}

// NewActivityLoggedHandler creates a new ActivityLoggedHandler.
func NewActivityLoggedHandler(svc PlanningService) *ActivityLoggedHandler {
	return &ActivityLoggedHandler{svc: svc}
}

// Handle processes a learn.ActivityLogged event.
func (h *ActivityLoggedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(learn.ActivityLogged)
	if !ok {
		return fmt.Errorf("ActivityLoggedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleActivityLogged(ctx, e.FamilyID, e.StudentID, e.ActivityID)
}
