package learn

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
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

// ─── FamilyDeletionScheduledHandler ─────────────────────────────────────────

// FamilyDeletionScheduledHandler handles iam.FamilyDeletionScheduled by
// triggering a final data export opportunity. [06-learn §17.4]
type FamilyDeletionScheduledHandler struct {
	svc LearningService
}

// NewFamilyDeletionScheduledHandler creates a new FamilyDeletionScheduledHandler.
func NewFamilyDeletionScheduledHandler(svc LearningService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{svc: svc}
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("learn.FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFamilyDeletionScheduled(ctx, e.FamilyID)
}

// ─── PurchaseCompletedHandler ─────────────────────────────────────────────────

// PurchaseCompletedHandler handles mkt.PurchaseCompleted by integrating
// purchased content into the family's learning resources. [06-learn §17.4]
type PurchaseCompletedHandler struct {
	svc LearningService
}

// NewPurchaseCompletedHandler creates a new PurchaseCompletedHandler.
func NewPurchaseCompletedHandler(svc LearningService) *PurchaseCompletedHandler {
	return &PurchaseCompletedHandler{svc: svc}
}

func (h *PurchaseCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.PurchaseCompleted)
	if !ok {
		return fmt.Errorf("learn.PurchaseCompletedHandler: unexpected event type %T", event)
	}
	return h.svc.HandlePurchaseCompleted(ctx, e.FamilyID, PurchaseMetadata{
		ContentType: e.ContentMetadata.ContentType,
		ContentIDs:  e.ContentMetadata.ContentIDs,
		PublisherID: e.ContentMetadata.PublisherID,
	})
}

// ─── Deferred Handlers ──────────────────────────────────────────────────────

// MethodologyConfigUpdatedHandler would handle method.MethodologyConfigUpdated
// by invalidating the tool resolution cache.
// DEFERRED: Phase 2 — tool cache invalidation. [06-learn §17.4]
