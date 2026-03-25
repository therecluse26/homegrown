package recs

import (
	"context"
	"fmt"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/learn"
	"github.com/homegrown-academy/homegrown-academy/internal/method"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 1: Signal Recording Handlers [13-recs §12]
// ═══════════════════════════════════════════════════════════════════════════════

// ActivityLoggedHandler records an activity signal when a student logs a learning activity.
// Source: learn::events::ActivityLogged [06-learn §18.3]
type ActivityLoggedHandler struct {
	RecsService RecsService
}

func NewActivityLoggedHandler(svc RecsService) *ActivityLoggedHandler {
	return &ActivityLoggedHandler{RecsService: svc}
}

func (h *ActivityLoggedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(learn.ActivityLogged)
	if !ok {
		return fmt.Errorf("ActivityLoggedHandler: unexpected event type %T", event)
	}
	var durationMinutes *int
	if e.DurationMinutes != nil {
		d := int(*e.DurationMinutes)
		durationMinutes = &d
	}
	return h.RecsService.RecordSignal(ctx, RecordSignalCommand{
		FamilyID:        shared.NewFamilyID(e.FamilyID),
		StudentID:       &e.StudentID,
		SignalType:      SignalActivityLogged,
		MethodologySlug: "", // resolved by service from iam_families [13-recs §9.3]
		Payload: map[string]any{
			"subject_tags":     e.SubjectTags,
			"duration_minutes": durationMinutes,
		},
		SignalDate: e.ActivityDate,
	})
}

// BookCompletedHandler records a book completion signal when a student finishes a book.
// Source: learn::events::BookCompleted [06-learn §18.3]
type BookCompletedHandler struct {
	RecsService RecsService
}

func NewBookCompletedHandler(svc RecsService) *BookCompletedHandler {
	return &BookCompletedHandler{RecsService: svc}
}

func (h *BookCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(learn.BookCompleted)
	if !ok {
		return fmt.Errorf("BookCompletedHandler: unexpected event type %T", event)
	}
	return h.RecsService.RecordSignal(ctx, RecordSignalCommand{
		FamilyID:        shared.NewFamilyID(e.FamilyID),
		StudentID:       &e.StudentID,
		SignalType:      SignalBookCompleted,
		MethodologySlug: "", // resolved by service from iam_families [13-recs §9.3]
		Payload: map[string]any{
			"title":           e.ReadingItemTitle,
			"reading_item_id": e.ReadingItemID,
		},
		SignalDate: time.Now().UTC(),
	})
}

// PurchaseCompletedHandler records a purchase signal when a family buys marketplace content.
// Source: mkt::events::PurchaseCompleted [07-mkt §18.3]
type PurchaseCompletedHandler struct {
	RecsService RecsService
}

func NewPurchaseCompletedHandler(svc RecsService) *PurchaseCompletedHandler {
	return &PurchaseCompletedHandler{RecsService: svc}
}

func (h *PurchaseCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.PurchaseCompleted)
	if !ok {
		return fmt.Errorf("PurchaseCompletedHandler: unexpected event type %T", event)
	}
	return h.RecsService.RecordSignal(ctx, RecordSignalCommand{
		FamilyID:        shared.NewFamilyID(e.FamilyID),
		StudentID:       nil, // purchases are family-level, not student-specific
		SignalType:      SignalPurchaseCompleted,
		MethodologySlug: "", // resolved by service from iam_families [13-recs §9.3]
		Payload: map[string]any{
			"listing_id":   e.ListingID,
			"content_type": e.ContentMetadata.ContentType,
		},
		SignalDate: time.Now().UTC(),
	})
}

// ListingPublishedHandler registers a newly published listing in the popularity catalog.
// Source: mkt::events::ListingPublished [07-mkt §18.3]
// NOTE: Does NOT create a recs_signals row — this is a catalog-level event. [13-recs §9.4]
type ListingPublishedHandler struct {
	RecsService RecsService
}

func NewListingPublishedHandler(svc RecsService) *ListingPublishedHandler {
	return &ListingPublishedHandler{RecsService: svc}
}

func (h *ListingPublishedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.ListingPublished)
	if !ok {
		return fmt.Errorf("ListingPublishedHandler: unexpected event type %T", event)
	}
	return h.RecsService.RegisterListing(ctx, RegisterListingCommand{
		ListingID:   e.ListingID,
		PublisherID: e.PublisherID,
		ContentType: e.ContentType,
		SubjectTags: e.SubjectTags,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle Handlers [13-recs §12]
// ═══════════════════════════════════════════════════════════════════════════════

// FamilyDeletionScheduledHandler deletes all recs data for a family when deletion is scheduled.
// Source: iam::events::FamilyDeletionScheduled [01-iam §13.3]
type FamilyDeletionScheduledHandler struct {
	RecsService RecsService
}

func NewFamilyDeletionScheduledHandler(svc RecsService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{RecsService: svc}
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.RecsService.HandleFamilyDeletion(ctx, shared.NewFamilyID(e.FamilyID))
}

// MethodologyConfigUpdatedHandler invalidates cached methodology configuration.
// Source: method::events::MethodologyConfigUpdated [02-method §12]
type MethodologyConfigUpdatedHandler struct {
	RecsService RecsService
}

func NewMethodologyConfigUpdatedHandler(svc RecsService) *MethodologyConfigUpdatedHandler {
	return &MethodologyConfigUpdatedHandler{RecsService: svc}
}

func (h *MethodologyConfigUpdatedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	_, ok := event.(method.MethodologyConfigUpdated)
	if !ok {
		return fmt.Errorf("MethodologyConfigUpdatedHandler: unexpected event type %T", event)
	}
	return h.RecsService.InvalidateMethodologyCache(ctx)
}
