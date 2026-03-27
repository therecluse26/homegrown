package billing

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// iam:: Event Handlers [10-billing §16.4]
// DEFERRED: FamilyDeletionScheduled and PrimaryParentTransferred events are
// not yet defined in iam::events.go. Handlers exist with mirror types;
// wiring in main.go is deferred until iam:: publishes these events.
// ═══════════════════════════════════════════════════════════════════════════════

// FamilyDeletionScheduledHandler handles iam::FamilyDeletionScheduled events.
// Cancels subscription in Hyperswitch and deletes local records.
type FamilyDeletionScheduledHandler struct {
	svc BillingService
}

func NewFamilyDeletionScheduledHandler(svc BillingService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{svc: svc}
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("billing.FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFamilyDeletionScheduled(ctx, FamilyDeletionScheduledEvent{
		FamilyID: e.FamilyID,
	})
}

// PrimaryParentTransferredHandler handles iam::PrimaryParentTransferred events.
// Updates Hyperswitch customer email to new primary parent's email.
type PrimaryParentTransferredHandler struct {
	svc BillingService
}

func NewPrimaryParentTransferredHandler(svc BillingService) *PrimaryParentTransferredHandler {
	return &PrimaryParentTransferredHandler{svc: svc}
}

func (h *PrimaryParentTransferredHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.PrimaryParentTransferred)
	if !ok {
		return fmt.Errorf("billing.PrimaryParentTransferredHandler: unexpected event type %T", event)
	}
	return h.svc.HandlePrimaryParentTransferred(ctx, PrimaryParentTransferredEvent{
		FamilyID:     e.FamilyID,
		NewPrimaryID: e.NewPrimaryID,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// mkt:: Event Handlers (Phase 2) [10-billing §16.4]
// ═══════════════════════════════════════════════════════════════════════════════

// PurchaseCompletedHandler handles mkt::PurchaseCompleted events.
// Records creator earnings for payout aggregation.
type PurchaseCompletedHandler struct {
	svc BillingService
}

func NewPurchaseCompletedHandler(svc BillingService) *PurchaseCompletedHandler {
	return &PurchaseCompletedHandler{svc: svc}
}

func (h *PurchaseCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.PurchaseCompleted)
	if !ok {
		return fmt.Errorf("billing.PurchaseCompletedHandler: unexpected event type %T", event)
	}
	return h.svc.HandlePurchaseCompleted(ctx, PurchaseCompletedEvent{
		FamilyID:   e.FamilyID,
		PurchaseID: e.PurchaseID,
		ListingID:  e.ListingID,
	})
}

// PurchaseRefundedHandler handles mkt::PurchaseRefunded events.
// Deducts refund from creator's unpaid earnings.
type PurchaseRefundedHandler struct {
	svc BillingService
}

func NewPurchaseRefundedHandler(svc BillingService) *PurchaseRefundedHandler {
	return &PurchaseRefundedHandler{svc: svc}
}

func (h *PurchaseRefundedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.PurchaseRefunded)
	if !ok {
		return fmt.Errorf("billing.PurchaseRefundedHandler: unexpected event type %T", event)
	}
	return h.svc.HandlePurchaseRefunded(ctx, PurchaseRefundedEvent{
		PurchaseID:        e.PurchaseID,
		ListingID:         e.ListingID,
		FamilyID:          e.FamilyID,
		RefundAmountCents: e.RefundAmountCents,
	})
}
