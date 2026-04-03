package notify

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/billing"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/learn"
	"github.com/homegrown-academy/homegrown-academy/internal/method"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
	"github.com/homegrown-academy/homegrown-academy/internal/onboard"
	"github.com/homegrown-academy/homegrown-academy/internal/recs"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Social Event Handlers [08-notify §17.1]
// ═══════════════════════════════════════════════════════════════════════════════

// FriendRequestSentHandler handles social.FriendRequestSent events.
type FriendRequestSentHandler struct{ svc NotificationService }

func NewFriendRequestSentHandler(svc NotificationService) *FriendRequestSentHandler {
	return &FriendRequestSentHandler{svc: svc}
}

func (h *FriendRequestSentHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(social.FriendRequestSent)
	if !ok {
		return fmt.Errorf("notify.FriendRequestSentHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFriendRequestSent(ctx, FriendRequestSentEvent{
		FriendshipID:      e.FriendshipID,
		RequesterFamilyID: e.RequesterFamilyID,
		AccepterFamilyID:  e.AccepterFamilyID,
	})
}

// FriendRequestAcceptedHandler handles social.FriendRequestAccepted events.
type FriendRequestAcceptedHandler struct{ svc NotificationService }

func NewFriendRequestAcceptedHandler(svc NotificationService) *FriendRequestAcceptedHandler {
	return &FriendRequestAcceptedHandler{svc: svc}
}

func (h *FriendRequestAcceptedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(social.FriendRequestAccepted)
	if !ok {
		return fmt.Errorf("notify.FriendRequestAcceptedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFriendRequestAccepted(ctx, FriendRequestAcceptedEvent{
		FriendshipID:      e.FriendshipID,
		RequesterFamilyID: e.RequesterFamilyID,
		AccepterFamilyID:  e.AccepterFamilyID,
	})
}

// MessageSentHandler handles social.MessageSent events.
type MessageSentHandler struct{ svc NotificationService }

func NewMessageSentHandler(svc NotificationService) *MessageSentHandler {
	return &MessageSentHandler{svc: svc}
}

func (h *MessageSentHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(social.MessageSent)
	if !ok {
		return fmt.Errorf("notify.MessageSentHandler: unexpected event type %T", event)
	}
	return h.svc.HandleMessageSent(ctx, MessageSentEvent{
		MessageID:         e.MessageID,
		ConversationID:    e.ConversationID,
		SenderParentID:    e.SenderParentID,
		SenderFamilyID:    e.SenderFamilyID,
		RecipientParentID: e.RecipientParentID,
		RecipientFamilyID: e.RecipientFamilyID,
	})
}

// EventCancelledHandler handles social.EventCancelled events.
type EventCancelledHandler struct{ svc NotificationService }

func NewEventCancelledHandler(svc NotificationService) *EventCancelledHandler {
	return &EventCancelledHandler{svc: svc}
}

func (h *EventCancelledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(social.EventCancelled)
	if !ok {
		return fmt.Errorf("notify.EventCancelledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleEventCancelled(ctx, EventCancelledEvent{
		EventID:         e.EventID,
		CreatorFamilyID: e.CreatorFamilyID,
		Title:           e.Title,
		GoingFamilyIDs:  e.GoingFamilyIDs,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Method Event Handler
// ═══════════════════════════════════════════════════════════════════════════════

// FamilyMethodologyChangedHandler handles method.FamilyMethodologyChanged events.
type FamilyMethodologyChangedHandler struct{ svc NotificationService }

func NewFamilyMethodologyChangedHandler(svc NotificationService) *FamilyMethodologyChangedHandler {
	return &FamilyMethodologyChangedHandler{svc: svc}
}

func (h *FamilyMethodologyChangedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(method.FamilyMethodologyChanged)
	if !ok {
		return fmt.Errorf("notify.FamilyMethodologyChangedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFamilyMethodologyChanged(ctx, FamilyMethodologyChangedEvent{
		FamilyID: e.FamilyID,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Onboard Event Handler
// ═══════════════════════════════════════════════════════════════════════════════

// OnboardingCompletedHandler handles onboard.OnboardingCompleted events.
type OnboardingCompletedHandler struct{ svc NotificationService }

func NewOnboardingCompletedHandler(svc NotificationService) *OnboardingCompletedHandler {
	return &OnboardingCompletedHandler{svc: svc}
}

func (h *OnboardingCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(onboard.OnboardingCompleted)
	if !ok {
		return fmt.Errorf("notify.OnboardingCompletedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleOnboardingCompleted(ctx, OnboardingCompletedEvent{
		FamilyID: e.FamilyID,
		Skipped:  e.Skipped,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Learn Event Handlers
// ═══════════════════════════════════════════════════════════════════════════════

// ActivityLoggedHandler handles learn.ActivityLogged events.
type ActivityLoggedHandler struct{ svc NotificationService }

func NewActivityLoggedHandler(svc NotificationService) *ActivityLoggedHandler {
	return &ActivityLoggedHandler{svc: svc}
}

func (h *ActivityLoggedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(learn.ActivityLogged)
	if !ok {
		return fmt.Errorf("notify.ActivityLoggedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleActivityLogged(ctx, ActivityLoggedEvent{
		FamilyID:     e.FamilyID,
		StudentID:    e.StudentID,
		ActivityDate: e.ActivityDate.Format("2006-01-02"),
	})
}

// MilestoneAchievedHandler handles learn.MilestoneAchieved events.
type MilestoneAchievedHandler struct{ svc NotificationService }

func NewMilestoneAchievedHandler(svc NotificationService) *MilestoneAchievedHandler {
	return &MilestoneAchievedHandler{svc: svc}
}

func (h *MilestoneAchievedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(learn.MilestoneAchieved)
	if !ok {
		return fmt.Errorf("notify.MilestoneAchievedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleMilestoneAchieved(ctx, MilestoneAchievedEvent{
		FamilyID:    e.FamilyID,
		StudentID:   e.StudentID,
		StudentName: e.StudentName,
		Description: e.Description,
	})
}

// BookCompletedHandler handles learn.BookCompleted events.
type BookCompletedHandler struct{ svc NotificationService }

func NewBookCompletedHandler(svc NotificationService) *BookCompletedHandler {
	return &BookCompletedHandler{svc: svc}
}

func (h *BookCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(learn.BookCompleted)
	if !ok {
		return fmt.Errorf("notify.BookCompletedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleBookCompleted(ctx, BookCompletedEvent{
		FamilyID:        e.FamilyID,
		StudentID:       e.StudentID,
		ReadingItemTitle: e.ReadingItemTitle,
	})
}

// DataExportReadyHandler handles learn.DataExportReady events.
type DataExportReadyHandler struct{ svc NotificationService }

func NewDataExportReadyHandler(svc NotificationService) *DataExportReadyHandler {
	return &DataExportReadyHandler{svc: svc}
}

func (h *DataExportReadyHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(learn.DataExportReady)
	if !ok {
		return fmt.Errorf("notify.DataExportReadyHandler: unexpected event type %T", event)
	}
	return h.svc.HandleDataExportReady(ctx, DataExportReadyEvent{
		FamilyID: e.FamilyID,
		FileURL:  e.FileURL,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Marketplace Event Handlers
// ═══════════════════════════════════════════════════════════════════════════════

// PurchaseCompletedHandler handles mkt.PurchaseCompleted events.
type PurchaseCompletedHandler struct{ svc NotificationService }

func NewPurchaseCompletedHandler(svc NotificationService) *PurchaseCompletedHandler {
	return &PurchaseCompletedHandler{svc: svc}
}

func (h *PurchaseCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.PurchaseCompleted)
	if !ok {
		return fmt.Errorf("notify.PurchaseCompletedHandler: unexpected event type %T", event)
	}
	return h.svc.HandlePurchaseCompleted(ctx, PurchaseCompletedEvent{
		FamilyID:   e.FamilyID,
		PurchaseID: e.PurchaseID,
	})
}

// PurchaseRefundedHandler handles mkt.PurchaseRefunded events.
type PurchaseRefundedHandler struct{ svc NotificationService }

func NewPurchaseRefundedHandler(svc NotificationService) *PurchaseRefundedHandler {
	return &PurchaseRefundedHandler{svc: svc}
}

func (h *PurchaseRefundedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.PurchaseRefunded)
	if !ok {
		return fmt.Errorf("notify.PurchaseRefundedHandler: unexpected event type %T", event)
	}
	return h.svc.HandlePurchaseRefunded(ctx, PurchaseRefundedEvent{
		FamilyID:   e.FamilyID,
		PurchaseID: e.PurchaseID,
	})
}

// CreatorOnboardedHandler handles mkt.CreatorOnboarded events.
type CreatorOnboardedHandler struct{ svc NotificationService }

func NewCreatorOnboardedHandler(svc NotificationService) *CreatorOnboardedHandler {
	return &CreatorOnboardedHandler{svc: svc}
}

func (h *CreatorOnboardedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(mkt.CreatorOnboarded)
	if !ok {
		return fmt.Errorf("notify.CreatorOnboardedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleCreatorOnboarded(ctx, CreatorOnboardedEvent{
		CreatorID: e.CreatorID,
		ParentID:  e.ParentID,
		StoreName: e.StoreName,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Deferred Handlers (Missing Domains)
// ═══════════════════════════════════════════════════════════════════════════════

// ContentFlaggedHandler handles safety.ContentFlagged events.
// Uses consumer-defined interface to avoid circular dependency with safety::. [08-notify §17.3]
type ContentFlaggedHandler struct{ svc NotificationService }

// NewContentFlaggedHandler creates a new ContentFlaggedHandler.
func NewContentFlaggedHandler(svc NotificationService) *ContentFlaggedHandler {
	return &ContentFlaggedHandler{svc: svc}
}

func (h *ContentFlaggedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	type contentFlagged interface {
		GetContentKey() string
		GetFamilyID() uuid.UUID
		GetFlagType() string
	}
	e, ok := event.(contentFlagged)
	if !ok {
		return fmt.Errorf("notify.ContentFlaggedHandler: unexpected event %s", event.EventName())
	}
	return h.svc.HandleContentFlagged(ctx, ContentFlaggedEvent{
		FamilyID:    e.GetFamilyID(),
		ContentType: "flagged_content",
		Reason:      e.GetFlagType(),
	})
}

// ─── iam:: Event Handlers ─────────────────────────────────────────────────────

// CoParentAddedHandler handles iam.CoParentAdded events.
// Sends a welcome notification to the new co-parent. [08-notify §17.1]
type CoParentAddedHandler struct{ svc NotificationService }

func NewCoParentAddedHandler(svc NotificationService) *CoParentAddedHandler {
	return &CoParentAddedHandler{svc: svc}
}

func (h *CoParentAddedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.CoParentAdded)
	if !ok {
		return fmt.Errorf("notify.CoParentAddedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleCoParentAdded(ctx, CoParentAddedEvent{
		FamilyID:     e.FamilyID,
		CoParentID:   e.CoParentID,
		CoParentName: e.CoParentName,
	})
}

// FamilyDeletionScheduledHandler handles iam.FamilyDeletionScheduled events.
// Removes notification preferences and history. [08-notify §17.1]
type FamilyDeletionScheduledHandler struct{ svc NotificationService }

func NewFamilyDeletionScheduledHandler(svc NotificationService) *FamilyDeletionScheduledHandler {
	return &FamilyDeletionScheduledHandler{svc: svc}
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(iam.FamilyDeletionScheduled)
	if !ok {
		return fmt.Errorf("notify.FamilyDeletionScheduledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleFamilyDeletionScheduled(ctx, FamilyDeletionScheduledEvent{
		FamilyID:        e.FamilyID,
		DeleteAfterDate: e.DeleteAfter,
	})
}

// ─── Billing Event Handlers ──────────────────────────────────────────────────

// SubscriptionCreatedHandler handles billing.SubscriptionCreated events. [08-notify §17.1]
type SubscriptionCreatedHandler struct{ svc NotificationService }

func NewSubscriptionCreatedHandler(svc NotificationService) *SubscriptionCreatedHandler {
	return &SubscriptionCreatedHandler{svc: svc}
}

func (h *SubscriptionCreatedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(billing.SubscriptionCreated)
	if !ok {
		return fmt.Errorf("notify.SubscriptionCreatedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleSubscriptionCreated(ctx, SubscriptionCreatedEvent{
		FamilyID:         e.FamilyID,
		Tier:             e.Tier,
		BillingInterval:  e.BillingInterval,
		CurrentPeriodEnd: e.CurrentPeriodEnd,
	})
}

// SubscriptionChangedHandler handles billing.SubscriptionChanged events. [08-notify §17.1]
type SubscriptionChangedHandler struct{ svc NotificationService }

func NewSubscriptionChangedHandler(svc NotificationService) *SubscriptionChangedHandler {
	return &SubscriptionChangedHandler{svc: svc}
}

func (h *SubscriptionChangedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(billing.SubscriptionChanged)
	if !ok {
		return fmt.Errorf("notify.SubscriptionChangedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleSubscriptionChanged(ctx, SubscriptionChangedEvent{
		FamilyID:         e.FamilyID,
		Tier:             e.Tier,
		BillingInterval:  e.BillingInterval,
		CurrentPeriodEnd: e.CurrentPeriodEnd,
		ChangeType:       e.ChangeType,
	})
}

// SubscriptionCancelledHandler handles billing.SubscriptionCancelled events. [08-notify §17.1]
type SubscriptionCancelledHandler struct{ svc NotificationService }

func NewSubscriptionCancelledHandler(svc NotificationService) *SubscriptionCancelledHandler {
	return &SubscriptionCancelledHandler{svc: svc}
}

func (h *SubscriptionCancelledHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(billing.SubscriptionCancelled)
	if !ok {
		return fmt.Errorf("notify.SubscriptionCancelledHandler: unexpected event type %T", event)
	}
	return h.svc.HandleSubscriptionCancelled(ctx, SubscriptionCancelledEvent{
		FamilyID:    e.FamilyID,
		EffectiveAt: e.EffectiveAt,
	})
}

// PayoutCompletedHandler handles billing.PayoutCompleted events. [08-notify §17.1]
type PayoutCompletedHandler struct{ svc NotificationService }

func NewPayoutCompletedHandler(svc NotificationService) *PayoutCompletedHandler {
	return &PayoutCompletedHandler{svc: svc}
}

func (h *PayoutCompletedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(billing.PayoutCompleted)
	if !ok {
		return fmt.Errorf("notify.PayoutCompletedHandler: unexpected event type %T", event)
	}
	return h.svc.HandlePayoutCompleted(ctx, PayoutCompletedEvent{
		CreatorID:   e.CreatorID,
		PayoutID:    e.PayoutID,
		AmountCents: e.AmountCents,
		Currency:    e.Currency,
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Recs Event Handlers [13-recs §12, P3-1]
// ═══════════════════════════════════════════════════════════════════════════════

// RecommendationsGeneratedHandler handles recs.RecommendationsGenerated events.
// Notifies the family that new recommendations are available.
type RecommendationsGeneratedHandler struct{ svc NotificationService }

func NewRecommendationsGeneratedHandler(svc NotificationService) *RecommendationsGeneratedHandler {
	return &RecommendationsGeneratedHandler{svc: svc}
}

func (h *RecommendationsGeneratedHandler) Handle(ctx context.Context, event shared.DomainEvent) error {
	e, ok := event.(recs.RecommendationsGenerated)
	if !ok {
		return fmt.Errorf("notify.RecommendationsGeneratedHandler: unexpected event type %T", event)
	}
	return h.svc.HandleRecommendationsReady(ctx, RecommendationsReadyEvent{
		FamilyID: e.FamilyID,
		Count:    e.Count,
	})
}
