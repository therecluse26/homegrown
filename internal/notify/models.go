package notify

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Notification Type Constants [08-notify §8.3, §9]
// ═══════════════════════════════════════════════════════════════════════════════

const (
	// Social
	TypeFriendRequestSent     = "friend_request_sent"
	TypeFriendRequestAccepted = "friend_request_accepted"
	TypeMessageReceived       = "message_received"
	TypeEventCancelled        = "event_cancelled"

	// Learning
	TypeMethodologyChanged  = "methodology_changed"
	TypeOnboardingCompleted = "onboarding_completed"
	TypeActivityStreak      = "activity_streak"
	TypeMilestoneAchieved   = "milestone_achieved"
	TypeBookCompleted       = "book_completed"
	TypeDataExportReady     = "data_export_ready"

	// Marketplace
	TypePurchaseCompleted = "purchase_completed"
	TypePurchaseRefunded  = "purchase_refunded"
	TypeCreatorOnboarded  = "creator_onboarded"

	// System
	TypeContentFlagged          = "content_flagged"
	TypeCoParentAdded           = "co_parent_added"
	TypeFamilyDeletionScheduled = "family_deletion_scheduled"

	// Billing (Phase 2)
	TypeSubscriptionCreated   = "subscription_created"
	TypeSubscriptionChanged   = "subscription_changed"
	TypeSubscriptionCancelled = "subscription_cancelled"
	TypePayoutCompleted       = "payout_completed"
)

// ─── Category Constants ──────────────────────────────────────────────────────

const (
	CategorySocial      = "social"
	CategoryLearning    = "learning"
	CategoryMarketplace = "marketplace"
	CategorySystem      = "system"
)

// ─── Channel Constants ───────────────────────────────────────────────────────

const (
	ChannelInApp = "in_app"
	ChannelEmail = "email"
)

// ─── Digest Frequency Constants ──────────────────────────────────────────────

const (
	DigestImmediate = "immediate"
	DigestDaily     = "daily"
	DigestWeekly    = "weekly"
	DigestOff       = "off"
)

// ─── System-Critical Types ───────────────────────────────────────────────────

// SystemCriticalTypes cannot be disabled via preferences. [S§13.3]
var SystemCriticalTypes = []string{
	TypeContentFlagged,
	TypeCoParentAdded,
	TypeFamilyDeletionScheduled,
}

// ─── Lookup Maps ─────────────────────────────────────────────────────────────

// ValidNotificationTypes is the set of all registered notification types.
var ValidNotificationTypes = map[string]bool{
	TypeFriendRequestSent:     true,
	TypeFriendRequestAccepted: true,
	TypeMessageReceived:       true,
	TypeEventCancelled:        true,
	TypeMethodologyChanged:    true,
	TypeOnboardingCompleted:   true,
	TypeActivityStreak:        true,
	TypeMilestoneAchieved:     true,
	TypeBookCompleted:         true,
	TypeDataExportReady:       true,
	TypePurchaseCompleted:     true,
	TypePurchaseRefunded:      true,
	TypeCreatorOnboarded:      true,
	TypeContentFlagged:        true,
	TypeCoParentAdded:         true,
	TypeFamilyDeletionScheduled: true,
	TypeSubscriptionCreated:   true,
	TypeSubscriptionChanged:   true,
	TypeSubscriptionCancelled: true,
	TypePayoutCompleted:       true,
}

// ValidCategories is the set of valid notification categories.
var ValidCategories = map[string]bool{
	CategorySocial:      true,
	CategoryLearning:    true,
	CategoryMarketplace: true,
	CategorySystem:      true,
}

// ValidChannels is the set of valid delivery channels.
var ValidChannels = map[string]bool{
	ChannelInApp: true,
	ChannelEmail: true,
}

// ValidDigestFrequencies is the set of valid digest frequency values.
var ValidDigestFrequencies = map[string]bool{
	DigestImmediate: true,
	DigestDaily:     true,
	DigestWeekly:    true,
	DigestOff:       true,
}

// TypeToCategory maps notification types to their category.
var TypeToCategory = map[string]string{
	TypeFriendRequestSent:       CategorySocial,
	TypeFriendRequestAccepted:   CategorySocial,
	TypeMessageReceived:         CategorySocial,
	TypeEventCancelled:          CategorySocial,
	TypeMethodologyChanged:      CategoryLearning,
	TypeOnboardingCompleted:     CategoryLearning,
	TypeActivityStreak:          CategoryLearning,
	TypeMilestoneAchieved:       CategoryLearning,
	TypeBookCompleted:           CategoryLearning,
	TypeDataExportReady:         CategoryLearning,
	TypePurchaseCompleted:       CategoryMarketplace,
	TypePurchaseRefunded:        CategoryMarketplace,
	TypeCreatorOnboarded:        CategoryMarketplace,
	TypeContentFlagged:          CategorySystem,
	TypeCoParentAdded:           CategorySystem,
	TypeFamilyDeletionScheduled: CategorySystem,
	TypeSubscriptionCreated:     CategorySystem,
	TypeSubscriptionChanged:     CategorySystem,
	TypeSubscriptionCancelled:   CategorySystem,
	TypePayoutCompleted:         CategoryMarketplace,
}

// TypeToTitleTemplate maps notification types to their title template string. [08-notify §9]
var TypeToTitleTemplate = map[string]string{
	TypeFriendRequestSent:       "{family_name} sent you a friend request",
	TypeFriendRequestAccepted:   "{family_name} accepted your friend request",
	TypeMessageReceived:         "New message from {sender_name}",
	TypeEventCancelled:          "'{event_title}' has been cancelled",
	TypeMethodologyChanged:      "Your methodology has been updated",
	TypeOnboardingCompleted:     "Welcome to Homegrown Academy!",
	TypeActivityStreak:          "{student_name} reached a {days}-day streak!",
	TypeMilestoneAchieved:       "{student_name} achieved: {description}",
	TypeBookCompleted:           "{student_name} finished '{book_title}'",
	TypeDataExportReady:         "Your data export is ready to download",
	TypePurchaseCompleted:       "Your purchase is confirmed",
	TypePurchaseRefunded:        "Your refund has been processed",
	TypeCreatorOnboarded:        "Welcome to the Creator Program!",
	TypeContentFlagged:          "Content moderation update",
	TypeCoParentAdded:           "A co-parent has been added to your family",
	TypeFamilyDeletionScheduled: "Your account is scheduled for deletion",
	TypeSubscriptionCreated:     "Welcome to Homegrown Premium!",
	TypeSubscriptionChanged:     "Your subscription has been updated",
	TypeSubscriptionCancelled:   "Your subscription has ended",
	TypePayoutCompleted:         "Your payout of {amount} has been sent",
}

// TypeToTemplateAlias maps notification types to Postmark email template aliases. [08-notify §12]
var TypeToTemplateAlias = map[string]string{
	TypeFriendRequestSent:       "friend-request",
	TypeFriendRequestAccepted:   "friend-accepted",
	TypeMessageReceived:         "new-message",
	TypeEventCancelled:          "event-cancelled",
	TypeOnboardingCompleted:     "welcome",
	TypeActivityStreak:          "streak-milestone",
	TypeMilestoneAchieved:       "milestone",
	TypeDataExportReady:         "export-ready",
	TypePurchaseCompleted:       "purchase-receipt",
	TypePurchaseRefunded:        "refund-confirmation",
	TypeCreatorOnboarded:        "creator-welcome",
	TypeContentFlagged:          "content-flagged",
	TypeCoParentAdded:           "co-parent-added",
	TypeFamilyDeletionScheduled: "deletion-scheduled",
}

// ─── Validation Helpers ──────────────────────────────────────────────────────

// IsValidNotificationType checks if a notification type is registered.
func IsValidNotificationType(t string) bool {
	return ValidNotificationTypes[t]
}

// IsValidChannel checks if a channel is valid.
func IsValidChannel(ch string) bool {
	return ValidChannels[ch]
}

// IsValidCategory checks if a category is valid.
func IsValidCategory(cat string) bool {
	return ValidCategories[cat]
}

// IsSystemCritical checks if a notification type is system-critical (cannot be disabled).
func IsSystemCritical(t string) bool {
	return slices.Contains(SystemCriticalTypes, t)
}

// ─── Phase 1 Preference Matrix Types ─────────────────────────────────────────

// Phase1PreferenceTypes are the 16 notification types included in the preference matrix.
// Excludes Phase 2 billing types from the preference UI. [08-notify §13]
var Phase1PreferenceTypes = []string{
	TypeFriendRequestSent,
	TypeFriendRequestAccepted,
	TypeMessageReceived,
	TypeEventCancelled,
	TypeMethodologyChanged,
	TypeOnboardingCompleted,
	TypeActivityStreak,
	TypeMilestoneAchieved,
	TypeBookCompleted,
	TypeDataExportReady,
	TypePurchaseCompleted,
	TypePurchaseRefunded,
	TypeCreatorOnboarded,
	TypeContentFlagged,
	TypeCoParentAdded,
	TypeFamilyDeletionScheduled,
}

// ─── Streak Milestone Thresholds ─────────────────────────────────────────────

// StreakMilestones are the consecutive-day counts that trigger a streak notification.
var StreakMilestones = map[int]bool{
	7: true, 14: true, 30: true, 60: true, 100: true,
}

// ═══════════════════════════════════════════════════════════════════════════════
// Request Types [08-notify §8.1]
// ═══════════════════════════════════════════════════════════════════════════════

// NotificationListParams holds query parameters for GET /v1/notifications.
type NotificationListParams struct {
	Cursor     *string `query:"cursor"`
	Limit      *uint8  `query:"limit"`
	Category   *string `query:"category"`
	UnreadOnly *bool   `query:"unread_only"`
}

// UpdatePreferencesCommand holds the body for PATCH /v1/notifications/preferences.
type UpdatePreferencesCommand struct {
	Preferences []PreferenceUpdate `json:"preferences" validate:"required,dive"`
}

// PreferenceUpdate represents a single preference change within a batch update.
type PreferenceUpdate struct {
	NotificationType string  `json:"notification_type" validate:"required"`
	Channel          string  `json:"channel" validate:"required"`
	Enabled          bool    `json:"enabled"`
	DigestFrequency  *string `json:"digest_frequency,omitempty"`
}

// MarkAllReadRequest holds the body for PATCH /v1/notifications/read-all.
type MarkAllReadRequest struct {
	Category *string `json:"category"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Types [08-notify §8.2]
// ═══════════════════════════════════════════════════════════════════════════════

// NotificationResponse represents a single notification in the feed.
type NotificationResponse struct {
	ID               uuid.UUID `json:"id"`
	NotificationType string    `json:"notification_type"`
	Category         string    `json:"category"`
	Title            string    `json:"title"`
	Body             string    `json:"body"`
	ActionURL        *string   `json:"action_url,omitempty"`
	IsRead           bool      `json:"is_read"`
	CreatedAt        time.Time `json:"created_at"`
}

// NotificationListResponse is a paginated notification list with unread badge count.
type NotificationListResponse struct {
	Notifications []NotificationResponse `json:"notifications"`
	UnreadCount   int64                  `json:"unread_count"`
	NextCursor    *string                `json:"next_cursor,omitempty"`
}

// PreferenceResponse represents a single entry in the preference matrix.
type PreferenceResponse struct {
	NotificationType string `json:"notification_type"`
	Channel          string `json:"channel"`
	Enabled          bool   `json:"enabled"`
	DigestFrequency  string `json:"digest_frequency"`
	SystemCritical   bool   `json:"system_critical"`
}

// MarkAllReadResponse holds the result of a bulk mark-all-read operation.
type MarkAllReadResponse struct {
	UpdatedCount int64 `json:"updated_count"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Internal Types [08-notify §8.3]
// ═══════════════════════════════════════════════════════════════════════════════

// CreateNotificationCommand is an internal command used by event handlers.
type CreateNotificationCommand struct {
	FamilyID         uuid.UUID
	NotificationType string
	Title            string
	Body             string
	ActionURL        *string
	Metadata         map[string]any
}

// SendEmailCommand is an internal command to send an email.
type SendEmailCommand struct {
	To            string         `json:"to"`
	TemplateAlias string         `json:"template_alias"`
	TemplateModel map[string]any `json:"template_model"`
}

// WebSocketFrame is the frame pushed to connected clients via Redis pub/sub. [08-notify §11]
type WebSocketFrame struct {
	MsgType string               `json:"msg_type"`
	Data    NotificationResponse `json:"data"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// GORM Models [08-notify §3]
// ═══════════════════════════════════════════════════════════════════════════════

// NotifyNotification maps to the notify_notifications table.
type NotifyNotification struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID         uuid.UUID       `gorm:"type:uuid;not null"`
	NotificationType string          `gorm:"type:text;not null"`
	Category         string          `gorm:"type:text;not null"`
	Title            string          `gorm:"type:text;not null"`
	Body             string          `gorm:"type:text;not null"`
	ActionURL        *string         `gorm:"type:text"`
	Metadata         json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	IsRead           bool            `gorm:"not null;default:false"`
	CreatedAt        time.Time       `gorm:"not null;default:now()"`
}

func (NotifyNotification) TableName() string { return "notify_notifications" }

// NotifyPreference maps to the notify_preferences table.
type NotifyPreference struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID         uuid.UUID `gorm:"type:uuid;not null"`
	NotificationType string    `gorm:"type:text;not null"`
	Channel          string    `gorm:"type:text;not null"`
	Enabled          bool      `gorm:"not null;default:true"`
	DigestFrequency  string    `gorm:"type:text;not null;default:'immediate'"`
}

func (NotifyPreference) TableName() string { return "notify_preferences" }

// NotifyDigest maps to the notify_digests table (Phase 2).
type NotifyDigest struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID    uuid.UUID       `gorm:"type:uuid;not null"`
	DigestType  string          `gorm:"type:text;not null"`
	PeriodStart time.Time       `gorm:"not null"`
	PeriodEnd   time.Time       `gorm:"not null"`
	ContentJSON json.RawMessage `gorm:"type:jsonb;not null"`
	SentAt      *time.Time      `gorm:"type:timestamptz"`
	CreatedAt   time.Time       `gorm:"not null;default:now()"`
}

func (NotifyDigest) TableName() string { return "notify_digests" }

// NotifyEmailStatus maps to the notify_email_status table.
type NotifyEmailStatus struct {
	ID         uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID   uuid.UUID       `gorm:"type:uuid;not null"`
	Email      string          `gorm:"type:text;not null"`
	BounceType string          `gorm:"type:text;not null"`
	BouncedAt  time.Time       `gorm:"not null;default:now()"`
	RawPayload json.RawMessage `gorm:"type:jsonb"`
}

func (NotifyEmailStatus) TableName() string { return "notify_email_status" }

// ═══════════════════════════════════════════════════════════════════════════════
// Repository-Level Types
// ═══════════════════════════════════════════════════════════════════════════════

// CreateNotification is the repo-level input for creating a notification.
type CreateNotification struct {
	FamilyID         uuid.UUID
	NotificationType string
	Category         string
	Title            string
	Body             string
	ActionURL        *string
	Metadata         json.RawMessage
}

// PreferenceUpsert is the repo-level input for upserting a preference.
type PreferenceUpsert struct {
	NotificationType string
	Channel          string
	Enabled          bool
	DigestFrequency  string
}

// CreateDigest is the repo-level input for creating a digest (Phase 2).
type CreateDigest struct {
	FamilyID    uuid.UUID
	DigestType  string
	PeriodStart time.Time
	PeriodEnd   time.Time
	ContentJSON json.RawMessage
}

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Event Types for Missing Domains
// ═══════════════════════════════════════════════════════════════════════════════
// These events implement shared.DomainEvent and will be replaced when their
// owning domains (safety::, iam::, billing::) define the real event types.

// ContentFlaggedEvent is a stub for safety::ContentFlagged. [08-notify §17.1]
type ContentFlaggedEvent struct {
	FamilyID    uuid.UUID `json:"family_id"`
	ContentID   uuid.UUID `json:"content_id"`
	ContentType string    `json:"content_type"`
	Reason      string    `json:"reason"`
}

func (ContentFlaggedEvent) EventName() string { return "safety.ContentFlagged" }

// CoParentAddedEvent is a stub for iam::CoParentAdded (Phase 2).
type CoParentAddedEvent struct {
	FamilyID      uuid.UUID `json:"family_id"`
	CoParentID    uuid.UUID `json:"co_parent_id"`
	CoParentName  string    `json:"co_parent_name"`
}

func (CoParentAddedEvent) EventName() string { return "iam.CoParentAdded" }

// FamilyDeletionScheduledEvent is a stub for iam::FamilyDeletionScheduled (Phase 2).
type FamilyDeletionScheduledEvent struct {
	FamilyID       uuid.UUID `json:"family_id"`
	DeleteAfterDate time.Time `json:"delete_after_date"`
}

func (FamilyDeletionScheduledEvent) EventName() string { return "iam.FamilyDeletionScheduled" }

// SubscriptionCreatedEvent is a stub for billing::SubscriptionCreated (Phase 2).
type SubscriptionCreatedEvent struct {
	FamilyID uuid.UUID `json:"family_id"`
}

func (SubscriptionCreatedEvent) EventName() string { return "billing.SubscriptionCreated" }

// SubscriptionChangedEvent is a stub for billing::SubscriptionChanged (Phase 2).
type SubscriptionChangedEvent struct {
	FamilyID uuid.UUID `json:"family_id"`
}

func (SubscriptionChangedEvent) EventName() string { return "billing.SubscriptionChanged" }

// SubscriptionCancelledEvent is a stub for billing::SubscriptionCancelled (Phase 2).
type SubscriptionCancelledEvent struct {
	FamilyID uuid.UUID `json:"family_id"`
}

func (SubscriptionCancelledEvent) EventName() string { return "billing.SubscriptionCancelled" }

// PayoutCompletedEvent is a stub for billing::PayoutCompleted (Phase 2).
type PayoutCompletedEvent struct {
	FamilyID uuid.UUID `json:"family_id"`
}

func (PayoutCompletedEvent) EventName() string { return "billing.PayoutCompleted" }

// Ensure stub events implement DomainEvent at compile time.
var (
	_ shared.DomainEvent = ContentFlaggedEvent{}
	_ shared.DomainEvent = CoParentAddedEvent{}
	_ shared.DomainEvent = FamilyDeletionScheduledEvent{}
	_ shared.DomainEvent = SubscriptionCreatedEvent{}
	_ shared.DomainEvent = SubscriptionChangedEvent{}
	_ shared.DomainEvent = SubscriptionCancelledEvent{}
	_ shared.DomainEvent = PayoutCompletedEvent{}
)
