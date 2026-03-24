package notify

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [08-notify §5]
// ═══════════════════════════════════════════════════════════════════════════════

// NotificationService defines all use cases exposed to handlers and event handlers.
type NotificationService interface {
	// ─── Commands ─────────────────────────────────────────────────────

	// CreateNotification creates an in-app notification and fans out via WebSocket + email.
	CreateNotification(ctx context.Context, cmd CreateNotificationCommand) error

	// MarkRead marks a single notification as read. Idempotent.
	MarkRead(ctx context.Context, notificationID uuid.UUID, scope *shared.FamilyScope) (NotificationResponse, error)

	// MarkAllRead bulk marks all (optionally category-filtered) notifications as read.
	MarkAllRead(ctx context.Context, scope *shared.FamilyScope, category *string) (int64, error)

	// UpdatePreferences batch upserts notification preferences.
	UpdatePreferences(ctx context.Context, cmd UpdatePreferencesCommand, scope *shared.FamilyScope) ([]PreferenceResponse, error)

	// ProcessUnsubscribe processes a signed unsubscribe token and disables the preference.
	ProcessUnsubscribe(ctx context.Context, token string) error

	// GenerateUnsubscribeToken creates a signed unsubscribe token for a family/type/channel.
	GenerateUnsubscribeToken(familyID uuid.UUID, notificationType string, channel string) (string, error)

	// SendEmail enqueues a transactional email directly, bypassing notification creation.
	// Useful for direct email sends not tied to in-app notifications (e.g., digest compilation). [08-notify §5]
	SendEmail(ctx context.Context, cmd SendEmailCommand) error

	// ─── Event Handlers ──────────────────────────────────────────────

	// social:: events
	HandleFriendRequestSent(ctx context.Context, event FriendRequestSentEvent) error
	HandleFriendRequestAccepted(ctx context.Context, event FriendRequestAcceptedEvent) error
	HandleMessageSent(ctx context.Context, event MessageSentEvent) error
	HandleEventCancelled(ctx context.Context, event EventCancelledEvent) error

	// method:: events
	HandleFamilyMethodologyChanged(ctx context.Context, event FamilyMethodologyChangedEvent) error

	// onboard:: events
	HandleOnboardingCompleted(ctx context.Context, event OnboardingCompletedEvent) error

	// learn:: events
	HandleActivityLogged(ctx context.Context, event ActivityLoggedEvent) error
	HandleMilestoneAchieved(ctx context.Context, event MilestoneAchievedEvent) error
	HandleBookCompleted(ctx context.Context, event BookCompletedEvent) error
	HandleDataExportReady(ctx context.Context, event DataExportReadyEvent) error

	// mkt:: events
	HandlePurchaseCompleted(ctx context.Context, event PurchaseCompletedEvent) error
	HandlePurchaseRefunded(ctx context.Context, event PurchaseRefundedEvent) error
	HandleCreatorOnboarded(ctx context.Context, event CreatorOnboardedEvent) error

	// safety:: events (stub)
	HandleContentFlagged(ctx context.Context, event ContentFlaggedEvent) error

	// iam:: events (Phase 2 stubs)
	HandleCoParentAdded(ctx context.Context, event CoParentAddedEvent) error
	HandleFamilyDeletionScheduled(ctx context.Context, event FamilyDeletionScheduledEvent) error

	// billing:: events (Phase 2 stubs)
	HandleSubscriptionCreated(ctx context.Context, event SubscriptionCreatedEvent) error
	HandleSubscriptionChanged(ctx context.Context, event SubscriptionChangedEvent) error
	HandleSubscriptionCancelled(ctx context.Context, event SubscriptionCancelledEvent) error
	HandlePayoutCompleted(ctx context.Context, event PayoutCompletedEvent) error

	// ─── Queries ─────────────────────────────────────────────────────

	// ListNotifications returns a paginated notification list with unread count.
	ListNotifications(ctx context.Context, params NotificationListParams, scope *shared.FamilyScope) (*NotificationListResponse, error)

	// GetPreferences returns the full type x channel preference matrix with defaults applied.
	GetPreferences(ctx context.Context, scope *shared.FamilyScope) ([]PreferenceResponse, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [08-notify §6]
// ═══════════════════════════════════════════════════════════════════════════════

// NotificationRepository manages notification persistence. [08-notify §6]
type NotificationRepository interface {
	Create(ctx context.Context, cmd CreateNotification) (*NotifyNotification, error)
	GetByID(ctx context.Context, notificationID uuid.UUID, scope *shared.FamilyScope) (*NotifyNotification, error)
	List(ctx context.Context, params *NotificationListParams, scope *shared.FamilyScope) ([]NotifyNotification, error)
	CountUnread(ctx context.Context, scope *shared.FamilyScope) (int64, error)
	MarkRead(ctx context.Context, notificationID uuid.UUID, scope *shared.FamilyScope) (bool, error)
	MarkAllRead(ctx context.Context, scope *shared.FamilyScope, category *string) (int64, error)
	ExistsBySourceEvent(ctx context.Context, familyID uuid.UUID, notificationType string, sourceEventID string) (bool, error)
	DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// PreferenceRepository manages preference persistence. [08-notify §6]
type PreferenceRepository interface {
	GetAll(ctx context.Context, scope *shared.FamilyScope) ([]NotifyPreference, error)
	UpsertBatch(ctx context.Context, scope *shared.FamilyScope, updates []PreferenceUpsert) error
	IsEnabled(ctx context.Context, familyID uuid.UUID, notificationType string, channel string) (bool, error)
	DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// DigestRepository manages digest persistence (Phase 2). [08-notify §6]
type DigestRepository interface {
	Create(ctx context.Context, cmd CreateDigest) (*NotifyDigest, error)
	GetUnsent(ctx context.Context, limit int64) ([]NotifyDigest, error)
	MarkSent(ctx context.Context, digestID uuid.UUID) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Adapter Interfaces [08-notify §7]
// ═══════════════════════════════════════════════════════════════════════════════

// EmailAdapter is a provider-agnostic email delivery interface. [08-notify §7]
type EmailAdapter interface {
	SendTransactional(ctx context.Context, to string, templateAlias string, templateModel map[string]any) error
	SendBatch(ctx context.Context, messages []BatchEmailMessage) error
	SendBroadcast(ctx context.Context, to string, templateAlias string, templateModel map[string]any) error
}

// BatchEmailMessage represents a single message in a batch send.
type BatchEmailMessage struct {
	To            string         `json:"to"`
	TemplateAlias string         `json:"template_alias"`
	TemplateModel map[string]any `json:"template_model"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer-Defined Cross-Domain Interfaces [ARCH §4.2]
// ═══════════════════════════════════════════════════════════════════════════════

// IamServiceForNotify is the subset of iam::IamService that notify:: needs.
type IamServiceForNotify interface {
	GetFamilyPrimaryEmail(ctx context.Context, familyID uuid.UUID) (email string, displayName string, err error)
	GetFamilyIDForParent(ctx context.Context, parentID uuid.UUID) (uuid.UUID, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Event Types consumed by notify:: (mirror types from source domains)
// These are local types to avoid importing source domain packages directly
// into the service interface. Event handlers convert source domain events
// to these types. [ARCH §4.2]
// ═══════════════════════════════════════════════════════════════════════════════

// FriendRequestSentEvent mirrors social.FriendRequestSent.
type FriendRequestSentEvent struct {
	FriendshipID      uuid.UUID
	RequesterFamilyID uuid.UUID
	AccepterFamilyID  uuid.UUID
}

// FriendRequestAcceptedEvent mirrors social.FriendRequestAccepted.
type FriendRequestAcceptedEvent struct {
	FriendshipID      uuid.UUID
	RequesterFamilyID uuid.UUID
	AccepterFamilyID  uuid.UUID
}

// MessageSentEvent mirrors social.MessageSent.
type MessageSentEvent struct {
	MessageID         uuid.UUID
	ConversationID    uuid.UUID
	SenderParentID    uuid.UUID
	SenderFamilyID    uuid.UUID
	RecipientParentID uuid.UUID
	RecipientFamilyID uuid.UUID
}

// EventCancelledEvent mirrors social.EventCancelled.
type EventCancelledEvent struct {
	EventID         uuid.UUID
	CreatorFamilyID uuid.UUID
	Title           string
	GoingFamilyIDs  []uuid.UUID
}

// FamilyMethodologyChangedEvent mirrors method.FamilyMethodologyChanged.
type FamilyMethodologyChangedEvent struct {
	FamilyID uuid.UUID
}

// OnboardingCompletedEvent mirrors onboard.OnboardingCompleted.
type OnboardingCompletedEvent struct {
	FamilyID uuid.UUID
	Skipped  bool
}

// ActivityLoggedEvent mirrors learn.ActivityLogged.
type ActivityLoggedEvent struct {
	FamilyID     uuid.UUID
	StudentID    uuid.UUID
	ActivityDate string // YYYY-MM-DD format for streak key
}

// MilestoneAchievedEvent mirrors learn.MilestoneAchieved.
type MilestoneAchievedEvent struct {
	FamilyID    uuid.UUID
	StudentID   uuid.UUID
	StudentName string
	Description string
}

// BookCompletedEvent mirrors learn.BookCompleted.
type BookCompletedEvent struct {
	FamilyID        uuid.UUID
	StudentID       uuid.UUID
	ReadingItemTitle string
}

// DataExportReadyEvent mirrors learn.DataExportReady.
type DataExportReadyEvent struct {
	FamilyID uuid.UUID
	FileURL  string
}

// PurchaseCompletedEvent mirrors mkt.PurchaseCompleted.
type PurchaseCompletedEvent struct {
	FamilyID   uuid.UUID
	PurchaseID uuid.UUID
}

// PurchaseRefundedEvent mirrors mkt.PurchaseRefunded.
type PurchaseRefundedEvent struct {
	FamilyID   uuid.UUID
	PurchaseID uuid.UUID
}

// CreatorOnboardedEvent mirrors mkt.CreatorOnboarded.
type CreatorOnboardedEvent struct {
	CreatorID uuid.UUID
	ParentID  uuid.UUID
	StoreName string
}
