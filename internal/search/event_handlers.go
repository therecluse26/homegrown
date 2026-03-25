package search

import (
	"context"
	"fmt"

	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/media"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social"
)

// ─── PostCreatedHandler ──────────────────────────────────────────────────────

// PostCreatedHandler handles social::PostCreated events. [12-search §9]
// Phase 1: no-op (GENERATED ALWAYS column auto-updates search_vector).
type PostCreatedHandler struct {
	svc SearchService
}

func NewPostCreatedHandler(svc SearchService) *PostCreatedHandler {
	return &PostCreatedHandler{svc: svc}
}

func (h *PostCreatedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(social.PostCreated)
	if !ok {
		return fmt.Errorf("PostCreatedHandler: unexpected event type %T", event)
	}
	return h.svc.HandlePostCreated(ctx, &PostCreated{
		PostID:   e.PostID,
		FamilyID: e.FamilyID,
	})
}

// ─── ListingPublishedHandler ─────────────────────────────────────────────────

// ListingPublishedHandler handles mkt::ListingPublished events. [12-search §9]
type ListingPublishedHandler struct {
	svc SearchService
}

func NewListingPublishedHandler(svc SearchService) *ListingPublishedHandler {
	return &ListingPublishedHandler{svc: svc}
}

func (h *ListingPublishedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.ListingPublished)
	if !ok {
		return fmt.Errorf("ListingPublishedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleListingPublished(ctx, &ListingPublished{
		ListingID: e.ListingID,
	})
}

// ─── ListingArchivedHandler ──────────────────────────────────────────────────

// ListingArchivedHandler handles mkt::ListingArchived events. [12-search §9]
type ListingArchivedHandler struct {
	svc SearchService
}

func NewListingArchivedHandler(svc SearchService) *ListingArchivedHandler {
	return &ListingArchivedHandler{svc: svc}
}

func (h *ListingArchivedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.ListingArchived)
	if !ok {
		return fmt.Errorf("ListingArchivedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleListingArchived(ctx, &ListingArchived{
		ListingID: e.ListingID,
	})
}

// ─── UploadPublishedHandler ──────────────────────────────────────────────────

// UploadPublishedHandler handles media::UploadPublished events. [12-search §9]
type UploadPublishedHandler struct {
	svc SearchService
}

func NewUploadPublishedHandler(svc SearchService) *UploadPublishedHandler {
	return &UploadPublishedHandler{svc: svc}
}

func (h *UploadPublishedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(media.UploadPublished)
	if !ok {
		return fmt.Errorf("UploadPublishedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleUploadPublished(ctx, &UploadPublished{
		UploadID: e.UploadID,
	})
}

// ─── FamilyDeletionScheduledHandler ──────────────────────────────────────────

// FamilyDeletionScheduledHandler handles iam::FamilyDeletionScheduled events. [12-search §9]
// Phase 1: no-op (Typesense indexes don't exist yet — nothing to purge).
type FamilyDeletionScheduledHandler struct {
	svc SearchService
}

func NewFamilyDeletionScheduledHandler(svc SearchService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{svc: svc}
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFamilyDeletionScheduled(ctx, e.FamilyID)
}
