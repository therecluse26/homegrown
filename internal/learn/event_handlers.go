package learn

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── StudentCreatedHandler ─────────────────────────────────────────────────

// StudentCreatedHandler handles iam.StudentCreated events by initializing
// learning defaults for newly created students. [06-learn §17.4]
type StudentCreatedHandler struct {
	svc LearningService
}

// NewStudentCreatedHandler creates a new StudentCreatedHandler.
func NewStudentCreatedHandler(svc LearningService) *StudentCreatedHandler {
	return &StudentCreatedHandler{svc: svc}
}

func (h *StudentCreatedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.StudentCreated)
	if !ok {
		return fmt.Errorf("learn.StudentCreatedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleStudentCreated(ctx, e.FamilyID, e.StudentID)
}

// ─── StudentDeletedHandler ─────────────────────────────────────────────────

// StudentDeletedHandler handles iam.StudentDeleted events by cascade-deleting
// all learning data associated with the student. [06-learn §17.4]
type StudentDeletedHandler struct {
	svc LearningService
}

// NewStudentDeletedHandler creates a new StudentDeletedHandler.
func NewStudentDeletedHandler(svc LearningService) *StudentDeletedHandler {
	return &StudentDeletedHandler{svc: svc}
}

func (h *StudentDeletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.StudentDeleted)
	if !ok {
		return fmt.Errorf("learn.StudentDeletedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleStudentDeleted(ctx, e.FamilyID, e.StudentID)
}

// ─── Deferred Handlers ──────────────────────────────────────────────────────
// These handlers are defined but their event subscriptions are deferred until
// the required events exist in their respective domains.

// FamilyDeletionScheduledHandler would handle iam.FamilyDeletionScheduled by
// triggering a final data export opportunity.
// DEFERRED: iam.FamilyDeletionScheduled event does not exist yet. [06-learn §17.4]
// When activated, register in main.go:
//   eventBus.Subscribe(reflect.TypeOf(iam.FamilyDeletionScheduled{}), learn.NewFamilyDeletionScheduledHandler(learnSvc))

// PurchaseCompletedHandler would handle mkt.PurchaseCompleted by integrating
// purchased content into the family's learning resources.
// DEFERRED: mkt:: domain not implemented. [06-learn §17.4]
// When activated, register in main.go:
//   eventBus.Subscribe(reflect.TypeOf(mkt.PurchaseCompleted{}), learn.NewPurchaseCompletedHandler(learnSvc))

// MethodologyConfigUpdatedHandler would handle method.MethodologyConfigUpdated
// by invalidating the tool resolution cache.
// DEFERRED: Phase 2 — tool cache invalidation. [06-learn §17.4]
