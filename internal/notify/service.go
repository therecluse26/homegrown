package notify

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// NotificationServiceImpl implements NotificationService. [08-notify §5]
type NotificationServiceImpl struct {
	notificationRepo NotificationRepository
	preferenceRepo   PreferenceRepository
	digestRepo       DigestRepository
	emailAdapter     EmailAdapter
	iamService       IamServiceForNotify
	cache            shared.Cache
	pubsub           shared.PubSub
	jobEnqueuer      shared.JobEnqueuer
	unsubscribeSecret string
}

// NewNotificationService creates a new NotificationServiceImpl.
func NewNotificationService(
	notificationRepo NotificationRepository,
	preferenceRepo PreferenceRepository,
	digestRepo DigestRepository,
	emailAdapter EmailAdapter,
	iamService IamServiceForNotify,
	cache shared.Cache,
	pubsub shared.PubSub,
	jobEnqueuer shared.JobEnqueuer,
	unsubscribeSecret string,
) *NotificationServiceImpl {
	return &NotificationServiceImpl{
		notificationRepo:  notificationRepo,
		preferenceRepo:    preferenceRepo,
		digestRepo:        digestRepo,
		emailAdapter:      emailAdapter,
		iamService:        iamService,
		cache:             cache,
		pubsub:            pubsub,
		jobEnqueuer:       jobEnqueuer,
		unsubscribeSecret: unsubscribeSecret,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Queries
// ═══════════════════════════════════════════════════════════════════════════════

func (s *NotificationServiceImpl) ListNotifications(ctx context.Context, params NotificationListParams, scope *shared.FamilyScope) (*NotificationListResponse, error) {
	notifications, err := s.notificationRepo.List(ctx, &params, scope)
	if err != nil {
		return nil, &NotifyError{Err: fmt.Errorf("list notifications: %w", err)}
	}

	unreadCount, err := s.notificationRepo.CountUnread(ctx, scope)
	if err != nil {
		return nil, &NotifyError{Err: fmt.Errorf("count unread: %w", err)}
	}

	// Determine effective limit for hasMore detection (repo fetches limit+1).
	limit := 20
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = int(*params.Limit)
	}

	var nextCursor *string
	if len(notifications) > limit {
		// Trim the extra row and encode cursor from the last visible row.
		notifications = notifications[:limit]
		last := notifications[len(notifications)-1]
		c := shared.EncodeCursor(last.ID, last.CreatedAt)
		nextCursor = &c
	}

	items := make([]NotificationResponse, len(notifications))
	for i, n := range notifications {
		items[i] = notificationToResponse(n)
	}

	return &NotificationListResponse{
		Notifications: items,
		UnreadCount:   unreadCount,
		NextCursor:    nextCursor,
	}, nil
}

func (s *NotificationServiceImpl) GetPreferences(ctx context.Context, scope *shared.FamilyScope) ([]PreferenceResponse, error) {
	stored, err := s.preferenceRepo.GetAll(ctx, scope)
	if err != nil {
		return nil, &NotifyError{Err: fmt.Errorf("get preferences: %w", err)}
	}

	// Build a lookup map of stored overrides keyed by "type:channel".
	overrides := make(map[string]NotifyPreference, len(stored))
	for _, p := range stored {
		key := p.NotificationType + ":" + p.Channel
		overrides[key] = p
	}

	// Build the full 16 types x 2 channels = 32-cell matrix with defaults applied.
	channels := []string{ChannelInApp, ChannelEmail}
	result := make([]PreferenceResponse, 0, len(Phase1PreferenceTypes)*len(channels))

	for _, nt := range Phase1PreferenceTypes {
		for _, ch := range channels {
			key := nt + ":" + ch
			resp := PreferenceResponse{
				NotificationType: nt,
				Channel:          ch,
				Enabled:          true,
				DigestFrequency:  DigestImmediate,
				SystemCritical:   IsSystemCritical(nt),
			}
			if override, ok := overrides[key]; ok {
				resp.Enabled = override.Enabled
				resp.DigestFrequency = override.DigestFrequency
			}
			result = append(result, resp)
		}
	}

	return result, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Commands
// ═══════════════════════════════════════════════════════════════════════════════

func (s *NotificationServiceImpl) MarkRead(ctx context.Context, notificationID uuid.UUID, scope *shared.FamilyScope) (NotificationResponse, error) {
	// Attempt to mark read. Returns false if the notification doesn't exist or wrong family.
	updated, err := s.notificationRepo.MarkRead(ctx, notificationID, scope)
	if err != nil {
		return NotificationResponse{}, &NotifyError{Err: fmt.Errorf("mark read: %w", err)}
	}
	if !updated {
		return NotificationResponse{}, &NotifyError{Err: ErrNotificationNotFound}
	}

	// Re-fetch for response.
	n, err := s.notificationRepo.GetByID(ctx, notificationID, scope)
	if err != nil {
		return NotificationResponse{}, &NotifyError{Err: fmt.Errorf("get notification after mark read: %w", err)}
	}
	return notificationToResponse(*n), nil
}

func (s *NotificationServiceImpl) MarkAllRead(ctx context.Context, scope *shared.FamilyScope, category *string) (int64, error) {
	if category != nil && !IsValidCategory(*category) {
		return 0, &NotifyError{Err: ErrInvalidCategory}
	}
	count, err := s.notificationRepo.MarkAllRead(ctx, scope, category)
	if err != nil {
		return 0, &NotifyError{Err: fmt.Errorf("mark all read: %w", err)}
	}
	return count, nil
}

func (s *NotificationServiceImpl) UpdatePreferences(ctx context.Context, cmd UpdatePreferencesCommand, scope *shared.FamilyScope) ([]PreferenceResponse, error) {
	// Validate all updates before applying any.
	upserts := make([]PreferenceUpsert, len(cmd.Preferences))
	for i, p := range cmd.Preferences {
		if !IsValidNotificationType(p.NotificationType) {
			return nil, &NotifyError{Err: ErrInvalidNotificationType}
		}
		if !IsValidChannel(p.Channel) {
			return nil, &NotifyError{Err: ErrInvalidChannel}
		}
		if !p.Enabled && IsSystemCritical(p.NotificationType) {
			return nil, &NotifyError{Err: ErrCannotDisableSystemCritical}
		}
		if p.DigestFrequency != nil && !ValidDigestFrequencies[*p.DigestFrequency] {
			return nil, &NotifyError{Err: ErrInvalidDigestFrequency}
		}

		freq := DigestImmediate
		if p.DigestFrequency != nil {
			freq = *p.DigestFrequency
		}
		upserts[i] = PreferenceUpsert{
			NotificationType: p.NotificationType,
			Channel:          p.Channel,
			Enabled:          p.Enabled,
			DigestFrequency:  freq,
		}
	}

	if err := s.preferenceRepo.UpsertBatch(ctx, scope, upserts); err != nil {
		return nil, &NotifyError{Err: fmt.Errorf("upsert preferences: %w", err)}
	}

	return s.GetPreferences(ctx, scope)
}

func (s *NotificationServiceImpl) CreateNotification(ctx context.Context, cmd CreateNotificationCommand) error {
	category, ok := TypeToCategory[cmd.NotificationType]
	if !ok {
		return &NotifyError{Err: ErrInvalidNotificationType}
	}

	isCritical := IsSystemCritical(cmd.NotificationType)

	// Idempotency check via source_event_id in metadata.
	if sourceEventID, ok := cmd.Metadata["source_event_id"]; ok {
		if sid, ok := sourceEventID.(string); ok && sid != "" {
			exists, err := s.notificationRepo.ExistsBySourceEvent(ctx, cmd.FamilyID, cmd.NotificationType, sid)
			if err != nil {
				slog.Error("idempotency check failed", "error", err)
			} else if exists {
				return nil // Duplicate — silently skip.
			}
		}
	}

	// ─── In-App Path ─────────────────────────────────────────────────
	inAppEnabled := isCritical
	if !isCritical {
		enabled, err := s.preferenceRepo.IsEnabled(ctx, cmd.FamilyID, cmd.NotificationType, ChannelInApp)
		if err != nil {
			slog.Error("preference check failed (in_app)", "error", err)
			enabled = true // Default to enabled on error.
		}
		inAppEnabled = enabled
	}

	var created *NotifyNotification
	if inAppEnabled {
		metadataJSON, err := json.Marshal(cmd.Metadata)
		if err != nil {
			return &NotifyError{Err: fmt.Errorf("marshal metadata: %w", err)}
		}

		created, err = s.notificationRepo.Create(ctx, CreateNotification{
			FamilyID:         cmd.FamilyID,
			NotificationType: cmd.NotificationType,
			Category:         category,
			Title:            cmd.Title,
			Body:             cmd.Body,
			ActionURL:        cmd.ActionURL,
			Metadata:         metadataJSON,
		})
		if err != nil {
			return &NotifyError{Err: fmt.Errorf("create notification: %w", err)}
		}

		// WebSocket push via Redis pub/sub. [08-notify §11]
		resp := notificationToResponse(*created)
		frame := WebSocketFrame{
			MsgType: "notification",
			Data:    resp,
		}
		frameJSON, err := json.Marshal(frame)
		if err != nil {
			slog.Error("marshal websocket frame", "error", err)
		} else {
			channel := fmt.Sprintf("notifications:%s", cmd.FamilyID)
			if pubErr := s.pubsub.Publish(ctx, channel, frameJSON); pubErr != nil {
				slog.Error("websocket publish failed", "error", pubErr)
			}
		}
	}

	// ─── Email Path ──────────────────────────────────────────────────
	emailEnabled := isCritical
	if !isCritical {
		enabled, err := s.preferenceRepo.IsEnabled(ctx, cmd.FamilyID, cmd.NotificationType, ChannelEmail)
		if err != nil {
			slog.Error("preference check failed (email)", "error", err)
			enabled = true
		}
		emailEnabled = enabled
	}

	if emailEnabled {
		templateAlias, hasTemplate := TypeToTemplateAlias[cmd.NotificationType]
		if hasTemplate {
			email, _, emailErr := s.iamService.GetFamilyPrimaryEmail(ctx, cmd.FamilyID)
			if emailErr != nil {
				slog.Error("email lookup failed", "family_id", cmd.FamilyID, "error", emailErr)
			} else if email != "" {
				idempotencyKey := fmt.Sprintf("%s:%s", cmd.NotificationType, cmd.FamilyID)
				if sid, ok := cmd.Metadata["source_event_id"]; ok {
					idempotencyKey = fmt.Sprintf("%s:%v", cmd.NotificationType, sid)
				}

				templateModel := make(map[string]any)
				maps.Copy(templateModel, cmd.Metadata)

				// Add unsubscribe URL for CAN-SPAM compliance. [08-notify §13]
				unsubToken, tokenErr := s.GenerateUnsubscribeToken(cmd.FamilyID, cmd.NotificationType, ChannelEmail)
				if tokenErr != nil {
					slog.Error("generate unsubscribe token", "error", tokenErr)
				} else {
					templateModel["unsubscribe_url"] = fmt.Sprintf("/v1/notifications/unsubscribe?token=%s", unsubToken)
				}

				taskPayload := SendEmailTaskPayload{
					To:             email,
					TemplateAlias:  templateAlias,
					TemplateModel:  templateModel,
					IdempotencyKey: idempotencyKey,
				}
				if enqueueErr := s.jobEnqueuer.Enqueue(ctx, taskPayload); enqueueErr != nil {
					// Email enqueue failure doesn't prevent in-app creation. [08-notify §10]
					slog.Error("email task enqueue failed", "error", enqueueErr)
				}
			}
		}
	}

	_ = created // Suppress unused warning when in-app is disabled.
	return nil
}

// SendEmail enqueues a transactional email directly, bypassing notification creation.
// Useful for direct email sends not tied to in-app notifications (e.g., digest compilation). [08-notify §5]
func (s *NotificationServiceImpl) SendEmail(ctx context.Context, cmd SendEmailCommand) error {
	taskPayload := SendEmailTaskPayload{
		To:             cmd.To,
		TemplateAlias:  cmd.TemplateAlias,
		TemplateModel:  cmd.TemplateModel,
		IdempotencyKey: fmt.Sprintf("direct:%s:%s", cmd.TemplateAlias, cmd.To),
	}
	if err := s.jobEnqueuer.Enqueue(ctx, taskPayload); err != nil {
		return &NotifyError{Err: fmt.Errorf("enqueue email: %w", err)}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Unsubscribe Token [08-notify §13]
// ═══════════════════════════════════════════════════════════════════════════════

type unsubscribePayload struct {
	FamilyID         string `json:"f"`
	NotificationType string `json:"t"`
	Channel          string `json:"c"`
	Exp              int64  `json:"e"` // Unix timestamp
}

func (s *NotificationServiceImpl) GenerateUnsubscribeToken(familyID uuid.UUID, notificationType string, channel string) (string, error) {
	payload := unsubscribePayload{
		FamilyID:         familyID.String(),
		NotificationType: notificationType,
		Channel:          channel,
		Exp:              time.Now().Add(30 * 24 * time.Hour).Unix(), // 30-day TTL
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal unsubscribe payload: %w", err)
	}

	payloadB64 := base64.URLEncoding.EncodeToString(payloadJSON)
	sig := s.signPayload(payloadB64)
	return payloadB64 + "." + sig, nil
}

func (s *NotificationServiceImpl) ProcessUnsubscribe(ctx context.Context, token string) error {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return &NotifyError{Err: ErrInvalidUnsubscribeToken}
	}

	payloadB64, sig := parts[0], parts[1]

	// Verify HMAC signature.
	expectedSig := s.signPayload(payloadB64)
	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return &NotifyError{Err: ErrInvalidUnsubscribeToken}
	}

	// Decode payload.
	payloadJSON, err := base64.URLEncoding.DecodeString(payloadB64)
	if err != nil {
		return &NotifyError{Err: ErrInvalidUnsubscribeToken}
	}

	var payload unsubscribePayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return &NotifyError{Err: ErrInvalidUnsubscribeToken}
	}

	// Check expiration.
	if time.Now().Unix() > payload.Exp {
		return &NotifyError{Err: ErrInvalidUnsubscribeToken}
	}

	// Reject system-critical types.
	if IsSystemCritical(payload.NotificationType) {
		return &NotifyError{Err: ErrCannotDisableSystemCritical}
	}

	familyID, err := uuid.Parse(payload.FamilyID)
	if err != nil {
		return &NotifyError{Err: ErrInvalidUnsubscribeToken}
	}

	// Construct a FamilyScope for the preference update.
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})

	return s.preferenceRepo.UpsertBatch(ctx, &scope, []PreferenceUpsert{
		{
			NotificationType: payload.NotificationType,
			Channel:          payload.Channel,
			Enabled:          false,
			DigestFrequency:  DigestImmediate,
		},
	})
}

func (s *NotificationServiceImpl) signPayload(payload string) string {
	mac := hmac.New(sha256.New, []byte(s.unsubscribeSecret))
	mac.Write([]byte(payload))
	return base64.URLEncoding.EncodeToString(mac.Sum(nil))
}

// ═══════════════════════════════════════════════════════════════════════════════
// Social Event Handlers [08-notify §17.1]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *NotificationServiceImpl) HandleFriendRequestSent(ctx context.Context, event FriendRequestSentEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.AccepterFamilyID,
		NotificationType: TypeFriendRequestSent,
		Title:            TypeToTitleTemplate[TypeFriendRequestSent],
		Body:             "You have a new friend request",
		ActionURL:        strPtr("/friends/requests"),
		Metadata: map[string]any{
			"source_event_id":    event.FriendshipID.String(),
			"requester_family_id": event.RequesterFamilyID.String(),
		},
	})
}

func (s *NotificationServiceImpl) HandleFriendRequestAccepted(ctx context.Context, event FriendRequestAcceptedEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.RequesterFamilyID,
		NotificationType: TypeFriendRequestAccepted,
		Title:            TypeToTitleTemplate[TypeFriendRequestAccepted],
		Body:             "Your friend request was accepted",
		ActionURL:        strPtr("/friends"),
		Metadata: map[string]any{
			"source_event_id":   event.FriendshipID.String(),
			"accepter_family_id": event.AccepterFamilyID.String(),
		},
	})
}

func (s *NotificationServiceImpl) HandleMessageSent(ctx context.Context, event MessageSentEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.RecipientFamilyID,
		NotificationType: TypeMessageReceived,
		Title:            TypeToTitleTemplate[TypeMessageReceived],
		Body:             "You have a new message",
		ActionURL:        strPtr(fmt.Sprintf("/messages/%s", event.ConversationID)),
		Metadata: map[string]any{
			"source_event_id":    event.MessageID.String(),
			"conversation_id":    event.ConversationID.String(),
			"sender_family_id":   event.SenderFamilyID.String(),
		},
	})
}

func (s *NotificationServiceImpl) HandleEventCancelled(ctx context.Context, event EventCancelledEvent) error {
	// TODO(phase2): Use emailAdapter.SendBatch() for multi-family email delivery
	// instead of per-family CreateNotification email paths. [08-notify §17.4]
	for _, familyID := range event.GoingFamilyIDs {
		err := s.CreateNotification(ctx, CreateNotificationCommand{
			FamilyID:         familyID,
			NotificationType: TypeEventCancelled,
			Title:            fmt.Sprintf("'%s' has been cancelled", event.Title),
			Body:             "An event you were attending has been cancelled",
			ActionURL:        strPtr("/events"),
			Metadata: map[string]any{
				"source_event_id": event.EventID.String(),
				"event_title":     event.Title,
			},
		})
		if err != nil {
			slog.Error("event cancelled notification failed", "family_id", familyID, "error", err)
		}
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Method + Onboard Event Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (s *NotificationServiceImpl) HandleFamilyMethodologyChanged(ctx context.Context, event FamilyMethodologyChangedEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypeMethodologyChanged,
		Title:            TypeToTitleTemplate[TypeMethodologyChanged],
		Body:             "Your homeschooling methodology has been updated",
		ActionURL:        strPtr("/settings/methodology"),
		Metadata: map[string]any{
			"source_event_id": event.FamilyID.String(),
		},
	})
}

func (s *NotificationServiceImpl) HandleOnboardingCompleted(ctx context.Context, event OnboardingCompletedEvent) error {
	if event.Skipped {
		return nil // No notification for skipped onboarding. [08-notify §9]
	}
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypeOnboardingCompleted,
		Title:            TypeToTitleTemplate[TypeOnboardingCompleted],
		Body:             "Your family is all set up! Start exploring.",
		ActionURL:        strPtr("/dashboard"),
		Metadata: map[string]any{
			"source_event_id": event.FamilyID.String(),
		},
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Learn Event Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (s *NotificationServiceImpl) HandleActivityLogged(ctx context.Context, event ActivityLoggedEvent) error {
	// Streak detection via Redis counters. [08-notify §9]
	streakKey := fmt.Sprintf("notify:streak:%s:%s", event.StudentID, event.ActivityDate)

	count, err := s.cache.IncrementWithExpiry(ctx, streakKey, 48*time.Hour)
	if err != nil {
		slog.Error("streak increment failed", "error", err)
		return nil
	}

	// Only count once per student per day (first increment = 1).
	if count > 1 {
		return nil
	}

	// Count consecutive days by checking Redis keys backwards.
	consecutiveDays := 1
	today, parseErr := time.Parse("2006-01-02", event.ActivityDate)
	if parseErr != nil {
		return nil
	}
	for i := 1; i <= 100; i++ {
		prevDate := today.AddDate(0, 0, -i).Format("2006-01-02")
		prevKey := fmt.Sprintf("notify:streak:%s:%s", event.StudentID, prevDate)
		val, cacheErr := s.cache.Get(ctx, prevKey)
		if cacheErr != nil || val == "" {
			break
		}
		consecutiveDays++
	}

	if StreakMilestones[consecutiveDays] {
		return s.CreateNotification(ctx, CreateNotificationCommand{
			FamilyID:         event.FamilyID,
			NotificationType: TypeActivityStreak,
			Title:            fmt.Sprintf("%d-day activity streak!", consecutiveDays),
			Body:             fmt.Sprintf("Keep up the great work! %d consecutive days of learning.", consecutiveDays),
			ActionURL:        strPtr("/dashboard"),
			Metadata: map[string]any{
				"source_event_id": fmt.Sprintf("streak:%s:%d", event.StudentID, consecutiveDays),
				"student_id":      event.StudentID.String(),
				"streak_days":     consecutiveDays,
			},
		})
	}

	return nil
}

func (s *NotificationServiceImpl) HandleMilestoneAchieved(ctx context.Context, event MilestoneAchievedEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypeMilestoneAchieved,
		Title:            fmt.Sprintf("%s achieved: %s", event.StudentName, event.Description),
		Body:             event.Description,
		ActionURL:        strPtr("/dashboard"),
		Metadata: map[string]any{
			"source_event_id": fmt.Sprintf("milestone:%s:%s", event.StudentID, event.Description),
			"student_name":    event.StudentName,
			"description":     event.Description,
		},
	})
}

func (s *NotificationServiceImpl) HandleBookCompleted(ctx context.Context, event BookCompletedEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypeBookCompleted,
		Title:            fmt.Sprintf("Finished '%s'", event.ReadingItemTitle),
		Body:             fmt.Sprintf("Congratulations on finishing '%s'!", event.ReadingItemTitle),
		ActionURL:        strPtr("/learning/reading"),
		Metadata: map[string]any{
			"source_event_id": fmt.Sprintf("book:%s:%s", event.StudentID, event.ReadingItemTitle),
			"book_title":      event.ReadingItemTitle,
		},
	})
}

func (s *NotificationServiceImpl) HandleDataExportReady(ctx context.Context, event DataExportReadyEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypeDataExportReady,
		Title:            TypeToTitleTemplate[TypeDataExportReady],
		Body:             "Your data export is ready. Click to download.",
		ActionURL:        &event.FileURL,
		Metadata: map[string]any{
			"source_event_id": event.FamilyID.String(),
			"download_url":    event.FileURL,
		},
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Marketplace Event Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (s *NotificationServiceImpl) HandlePurchaseCompleted(ctx context.Context, event PurchaseCompletedEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypePurchaseCompleted,
		Title:            TypeToTitleTemplate[TypePurchaseCompleted],
		Body:             "Your purchase is confirmed and ready to use",
		ActionURL:        strPtr("/marketplace/purchases"),
		Metadata: map[string]any{
			"source_event_id": event.PurchaseID.String(),
		},
	})
}

func (s *NotificationServiceImpl) HandlePurchaseRefunded(ctx context.Context, event PurchaseRefundedEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypePurchaseRefunded,
		Title:            TypeToTitleTemplate[TypePurchaseRefunded],
		Body:             "Your refund has been processed",
		ActionURL:        strPtr("/marketplace/purchases"),
		Metadata: map[string]any{
			"source_event_id": event.PurchaseID.String(),
		},
	})
}

func (s *NotificationServiceImpl) HandleCreatorOnboarded(ctx context.Context, event CreatorOnboardedEvent) error {
	// Look up family ID from parent ID. [08-notify §17.1]
	familyID, err := s.iamService.GetFamilyIDForParent(ctx, event.ParentID)
	if err != nil {
		return fmt.Errorf("lookup family for parent %s: %w", event.ParentID, err)
	}
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         familyID,
		NotificationType: TypeCreatorOnboarded,
		Title:            TypeToTitleTemplate[TypeCreatorOnboarded],
		Body:             fmt.Sprintf("Welcome, %s! Your creator account is ready.", event.StoreName),
		ActionURL:        strPtr("/marketplace/creator/dashboard"),
		Metadata: map[string]any{
			"source_event_id": event.CreatorID.String(),
			"store_name":      event.StoreName,
		},
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Safety Event Handler (Stub)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *NotificationServiceImpl) HandleContentFlagged(ctx context.Context, event ContentFlaggedEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypeContentFlagged,
		Title:            TypeToTitleTemplate[TypeContentFlagged],
		Body:             "A content moderation action has been taken on your content",
		Metadata: map[string]any{
			"source_event_id": event.ContentID.String(),
			"content_type":    event.ContentType,
			"reason":          event.Reason,
		},
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2 Stubs (return nil)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *NotificationServiceImpl) HandleCoParentAdded(_ context.Context, _ CoParentAddedEvent) error {
	return nil
}

func (s *NotificationServiceImpl) HandleFamilyDeletionScheduled(_ context.Context, _ FamilyDeletionScheduledEvent) error {
	return nil
}

func (s *NotificationServiceImpl) HandleSubscriptionCreated(ctx context.Context, event SubscriptionCreatedEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypeSubscriptionCreated,
		Title:            TypeToTitleTemplate[TypeSubscriptionCreated],
		Body:             fmt.Sprintf("Your %s subscription is now active", event.Tier),
		ActionURL:        strPtr("/settings/billing"),
		Metadata: map[string]any{
			"tier":             event.Tier,
			"billing_interval": event.BillingInterval,
		},
	})
}

func (s *NotificationServiceImpl) HandleSubscriptionChanged(ctx context.Context, event SubscriptionChangedEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypeSubscriptionChanged,
		Title:            TypeToTitleTemplate[TypeSubscriptionChanged],
		Body:             fmt.Sprintf("Your subscription has been updated to %s (%s)", event.Tier, event.BillingInterval),
		ActionURL:        strPtr("/settings/billing"),
		Metadata: map[string]any{
			"tier":             event.Tier,
			"billing_interval": event.BillingInterval,
			"change_type":      event.ChangeType,
		},
	})
}

func (s *NotificationServiceImpl) HandleSubscriptionCancelled(ctx context.Context, event SubscriptionCancelledEvent) error {
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         event.FamilyID,
		NotificationType: TypeSubscriptionCancelled,
		Title:            TypeToTitleTemplate[TypeSubscriptionCancelled],
		Body:             fmt.Sprintf("Your subscription ended on %s", event.EffectiveAt.Format("January 2, 2006")),
		ActionURL:        strPtr("/settings/billing"),
		Metadata:         map[string]any{},
	})
}

func (s *NotificationServiceImpl) HandlePayoutCompleted(ctx context.Context, event PayoutCompletedEvent) error {
	familyID, err := s.iamService.GetFamilyIDForCreator(ctx, event.CreatorID)
	if err != nil {
		return fmt.Errorf("notify.HandlePayoutCompleted: lookup family for creator %s: %w", event.CreatorID, err)
	}
	amountFormatted := fmt.Sprintf("%.2f %s", float64(event.AmountCents)/100, event.Currency)
	return s.CreateNotification(ctx, CreateNotificationCommand{
		FamilyID:         familyID,
		NotificationType: TypePayoutCompleted,
		Title:            TypeToTitleTemplate[TypePayoutCompleted],
		Body:             fmt.Sprintf("Your payout of %s has been sent", amountFormatted),
		ActionURL:        strPtr("/marketplace/creator/payouts"),
		Metadata: map[string]any{
			"payout_id":    event.PayoutID.String(),
			"amount_cents": event.AmountCents,
			"currency":     event.Currency,
		},
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func notificationToResponse(n NotifyNotification) NotificationResponse {
	return NotificationResponse{
		ID:               n.ID,
		NotificationType: n.NotificationType,
		Category:         n.Category,
		Title:            n.Title,
		Body:             n.Body,
		ActionURL:        n.ActionURL,
		IsRead:           n.IsRead,
		CreatedAt:        n.CreatedAt,
	}
}

func strPtr(s string) *string {
	return &s
}
