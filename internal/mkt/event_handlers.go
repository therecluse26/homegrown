package mkt

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/method"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// contentFlaggedData is a consumer-defined interface for safety.ContentFlagged events.
// Direct import of safety:: would create a cycle (safety:: imports mkt::). [ARCH §4.4]
type contentFlaggedData interface {
	GetContentKey() string
	GetFamilyID() uuid.UUID
	GetFlagType() string
}

// ─── MethodologyConfigUpdatedHandler ─────────────────────────────────────────

// MethodologyConfigUpdatedHandler handles method.MethodologyConfigUpdated events
// by invalidating methodology tag caches used for listing browse. [07-mkt §17.4]
type MethodologyConfigUpdatedHandler struct {
	cache shared.Cache
}

// NewMethodologyConfigUpdatedHandler creates a new MethodologyConfigUpdatedHandler.
func NewMethodologyConfigUpdatedHandler(cache shared.Cache) *MethodologyConfigUpdatedHandler {
	return &MethodologyConfigUpdatedHandler{cache: cache}
}

func (h *MethodologyConfigUpdatedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	_, ok := event.(method.MethodologyConfigUpdated)
	if !ok {
		return fmt.Errorf("mkt.MethodologyConfigUpdatedHandler: unexpected event type %T", event)
	}

	if err := h.cache.Delete(ctx, "mkt:methodology_tags"); err != nil {
		slog.Warn("mkt: failed to invalidate methodology tag cache", "error", err)
	}
	return nil
}

// ─── ContentFlaggedHandler ───────────────────────────────────────────────────

// ContentFlaggedHandler handles safety.ContentFlagged by logging flagged content.
// Auto-archiving via ArchiveListingByContentKey is not yet implemented. [07-mkt §17.4]
type ContentFlaggedHandler struct {
	svc MarketplaceService
}

// NewContentFlaggedHandler creates a new ContentFlaggedHandler.
func NewContentFlaggedHandler(svc MarketplaceService) *ContentFlaggedHandler {
	return &ContentFlaggedHandler{svc: svc}
}

func (h *ContentFlaggedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(contentFlaggedData)
	if !ok {
		return fmt.Errorf("mkt.ContentFlaggedHandler: unexpected event type %T", event)
	}
	slog.Info("mkt: content flagged — listing review pending",
		"content_key", e.GetContentKey(),
		"flag_type", e.GetFlagType(),
		"family_id", e.GetFamilyID(),
	)
	slog.Warn("mkt: ArchiveListingByContentKey not yet implemented — flagged listing not auto-archived",
		"content_key", e.GetContentKey(),
		"flag_type", e.GetFlagType(),
		"family_id", e.GetFamilyID(),
	)
	return nil
}

// ─── FamilyDeletionScheduledHandler ─────────────────────────────────────────

// FamilyDeletionScheduledHandler handles iam.FamilyDeletionScheduled by
// anonymizing reviews and cleaning up cart data. [07-mkt §17.4]
type FamilyDeletionScheduledHandler struct {
	svc MarketplaceService
}

// NewFamilyDeletionScheduledHandler creates a new FamilyDeletionScheduledHandler.
func NewFamilyDeletionScheduledHandler(svc MarketplaceService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{svc: svc}
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("mkt.FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFamilyDeletionScheduled(ctx, e.FamilyID)
}
