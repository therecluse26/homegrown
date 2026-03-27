package notify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Mock Implementations
// ═══════════════════════════════════════════════════════════════════════════════

// mockNotificationRepo mocks NotificationRepository.
type mockNotificationRepo struct{ mock.Mock }

func (m *mockNotificationRepo) Create(ctx context.Context, cmd CreateNotification) (*NotifyNotification, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*NotifyNotification), args.Error(1)
}

func (m *mockNotificationRepo) GetByID(ctx context.Context, id uuid.UUID, scope *shared.FamilyScope) (*NotifyNotification, error) {
	args := m.Called(ctx, id, scope)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*NotifyNotification), args.Error(1)
}

func (m *mockNotificationRepo) List(ctx context.Context, params *NotificationListParams, scope *shared.FamilyScope) ([]NotifyNotification, error) {
	args := m.Called(ctx, params, scope)
	return args.Get(0).([]NotifyNotification), args.Error(1)
}

func (m *mockNotificationRepo) CountUnread(ctx context.Context, scope *shared.FamilyScope) (int64, error) {
	args := m.Called(ctx, scope)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockNotificationRepo) MarkRead(ctx context.Context, id uuid.UUID, scope *shared.FamilyScope) (bool, error) {
	args := m.Called(ctx, id, scope)
	return args.Bool(0), args.Error(1)
}

func (m *mockNotificationRepo) MarkAllRead(ctx context.Context, scope *shared.FamilyScope, category *string) (int64, error) {
	args := m.Called(ctx, scope, category)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockNotificationRepo) ExistsBySourceEvent(ctx context.Context, familyID uuid.UUID, notificationType string, sourceEventID string) (bool, error) {
	args := m.Called(ctx, familyID, notificationType, sourceEventID)
	return args.Bool(0), args.Error(1)
}

func (m *mockNotificationRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	args := m.Called(ctx, familyID)
	return args.Error(0)
}

// mockPreferenceRepo mocks PreferenceRepository.
type mockPreferenceRepo struct{ mock.Mock }

func (m *mockPreferenceRepo) GetAll(ctx context.Context, scope *shared.FamilyScope) ([]NotifyPreference, error) {
	args := m.Called(ctx, scope)
	return args.Get(0).([]NotifyPreference), args.Error(1)
}

func (m *mockPreferenceRepo) UpsertBatch(ctx context.Context, scope *shared.FamilyScope, updates []PreferenceUpsert) error {
	args := m.Called(ctx, scope, updates)
	return args.Error(0)
}

func (m *mockPreferenceRepo) IsEnabled(ctx context.Context, familyID uuid.UUID, notificationType string, channel string) (bool, error) {
	args := m.Called(ctx, familyID, notificationType, channel)
	return args.Bool(0), args.Error(1)
}

func (m *mockPreferenceRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	args := m.Called(ctx, familyID)
	return args.Error(0)
}

// mockDigestRepo mocks DigestRepository (Phase 2 stub).
type mockDigestRepo struct{ mock.Mock }

func (m *mockDigestRepo) Create(_ context.Context, _ CreateDigest) (*NotifyDigest, error) {
	return nil, nil
}
func (m *mockDigestRepo) GetUnsent(_ context.Context, _ int64) ([]NotifyDigest, error) {
	return nil, nil
}
func (m *mockDigestRepo) MarkSent(_ context.Context, _ uuid.UUID) error { return nil }

// mockEmailAdapter mocks EmailAdapter.
type mockEmailAdapter struct{ mock.Mock }

func (m *mockEmailAdapter) SendTransactional(ctx context.Context, to string, templateAlias string, templateModel map[string]any) error {
	args := m.Called(ctx, to, templateAlias, templateModel)
	return args.Error(0)
}

func (m *mockEmailAdapter) SendBatch(ctx context.Context, messages []BatchEmailMessage) error {
	args := m.Called(ctx, messages)
	return args.Error(0)
}

func (m *mockEmailAdapter) SendBroadcast(ctx context.Context, to string, templateAlias string, templateModel map[string]any) error {
	args := m.Called(ctx, to, templateAlias, templateModel)
	return args.Error(0)
}

// mockIamService mocks IamServiceForNotify.
type mockIamService struct{ mock.Mock }

func (m *mockIamService) GetFamilyPrimaryEmail(ctx context.Context, familyID uuid.UUID) (string, string, error) {
	args := m.Called(ctx, familyID)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *mockIamService) GetFamilyIDForParent(ctx context.Context, parentID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, parentID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockIamService) GetFamilyIDForCreator(ctx context.Context, creatorID uuid.UUID) (uuid.UUID, error) {
	args := m.Called(ctx, creatorID)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// mockCache mocks shared.Cache.
type mockCache struct{ mock.Mock }

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *mockCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *mockCache) IncrementWithExpiry(ctx context.Context, key string, window time.Duration) (int64, error) {
	args := m.Called(ctx, key, window)
	return args.Get(0).(int64), args.Error(1)
}

func (m *mockCache) Close() error { return nil }

// ─── Test Helpers ────────────────────────────────────────────────────────────

func testFamilyScope() *shared.FamilyScope {
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.New()})
	return &scope
}

func newTestService(
	notifRepo *mockNotificationRepo,
	prefRepo *mockPreferenceRepo,
	iamSvc *mockIamService,
	cache *mockCache,
) *NotificationServiceImpl {
	return NewNotificationService(
		notifRepo,
		prefRepo,
		&mockDigestRepo{},
		&mockEmailAdapter{},
		iamSvc,
		cache,
		shared.NoopPubSub{},
		shared.NoopJobEnqueuer{},
		"test-secret",
	)
}

func newTestServiceFull(
	notifRepo *mockNotificationRepo,
	prefRepo *mockPreferenceRepo,
	iamSvc *mockIamService,
	cache *mockCache,
	pubsub shared.PubSub,
	jobs shared.JobEnqueuer,
) *NotificationServiceImpl {
	return NewNotificationService(
		notifRepo,
		prefRepo,
		&mockDigestRepo{},
		&mockEmailAdapter{},
		iamSvc,
		cache,
		pubsub,
		jobs,
		"test-secret",
	)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group A: Validation Helpers (7 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestValidationHelpers(t *testing.T) {
	// A1. All 20 registered types return true from IsValidNotificationType.
	t.Run("A1_all_registered_types_valid", func(t *testing.T) {
		allTypes := []string{
			TypeFriendRequestSent, TypeFriendRequestAccepted, TypeMessageReceived, TypeEventCancelled,
			TypeMethodologyChanged, TypeOnboardingCompleted, TypeActivityStreak, TypeMilestoneAchieved,
			TypeBookCompleted, TypeDataExportReady,
			TypePurchaseCompleted, TypePurchaseRefunded, TypeCreatorOnboarded,
			TypeContentFlagged, TypeCoParentAdded, TypeFamilyDeletionScheduled,
			TypeSubscriptionCreated, TypeSubscriptionChanged, TypeSubscriptionCancelled, TypePayoutCompleted,
		}
		for _, nt := range allTypes {
			assert.True(t, IsValidNotificationType(nt), "expected valid: %s", nt)
		}
		assert.Equal(t, 20, len(allTypes))
	})

	// A2. Unknown type returns false.
	t.Run("A2_unknown_type_invalid", func(t *testing.T) {
		assert.False(t, IsValidNotificationType("unknown_type"))
		assert.False(t, IsValidNotificationType(""))
	})

	// A3. Valid channels return true; "push" returns false.
	t.Run("A3_valid_channels", func(t *testing.T) {
		assert.True(t, IsValidChannel("in_app"))
		assert.True(t, IsValidChannel("email"))
		assert.False(t, IsValidChannel("push"))
		assert.False(t, IsValidChannel(""))
	})

	// A4. Valid categories return true.
	t.Run("A4_valid_categories", func(t *testing.T) {
		assert.True(t, IsValidCategory("social"))
		assert.True(t, IsValidCategory("learning"))
		assert.True(t, IsValidCategory("marketplace"))
		assert.True(t, IsValidCategory("system"))
		assert.False(t, IsValidCategory("unknown"))
	})

	// A5. System-critical types identified correctly.
	t.Run("A5_system_critical_types", func(t *testing.T) {
		assert.True(t, IsSystemCritical(TypeContentFlagged))
		assert.True(t, IsSystemCritical(TypeCoParentAdded))
		assert.True(t, IsSystemCritical(TypeFamilyDeletionScheduled))
	})

	// A6. Non-critical type returns false from IsSystemCritical.
	t.Run("A6_non_critical_type", func(t *testing.T) {
		assert.False(t, IsSystemCritical(TypeFriendRequestSent))
		assert.False(t, IsSystemCritical(TypePurchaseCompleted))
		assert.False(t, IsSystemCritical(TypeActivityStreak))
	})

	// A7. Every registered type maps to a known category.
	t.Run("A7_all_types_have_category", func(t *testing.T) {
		for nt := range ValidNotificationTypes {
			cat, ok := TypeToCategory[nt]
			assert.True(t, ok, "type %s has no category mapping", nt)
			assert.True(t, IsValidCategory(cat), "type %s mapped to invalid category %s", nt, cat)
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group B: Preferences — Service Layer (9 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestPreferences(t *testing.T) {
	ctx := context.Background()

	// B1. GetPreferences returns full 32-cell matrix with defaults.
	t.Run("B1_full_matrix_with_defaults", func(t *testing.T) {
		prefRepo := new(mockPreferenceRepo)
		prefRepo.On("GetAll", mock.Anything, mock.Anything).Return([]NotifyPreference{}, nil)
		svc := newTestService(new(mockNotificationRepo), prefRepo, new(mockIamService), new(mockCache))

		result, err := svc.GetPreferences(ctx, testFamilyScope())
		require.NoError(t, err)
		assert.Equal(t, 32, len(result)) // 16 types x 2 channels
		for _, p := range result {
			assert.True(t, p.Enabled)
			assert.Equal(t, DigestImmediate, p.DigestFrequency)
		}
	})

	// B2. GetPreferences applies stored overrides.
	t.Run("B2_applies_overrides", func(t *testing.T) {
		prefRepo := new(mockPreferenceRepo)
		prefRepo.On("GetAll", mock.Anything, mock.Anything).Return([]NotifyPreference{
			{NotificationType: TypeFriendRequestSent, Channel: ChannelEmail, Enabled: false, DigestFrequency: DigestOff},
		}, nil)
		svc := newTestService(new(mockNotificationRepo), prefRepo, new(mockIamService), new(mockCache))

		result, err := svc.GetPreferences(ctx, testFamilyScope())
		require.NoError(t, err)

		var found bool
		for _, p := range result {
			if p.NotificationType == TypeFriendRequestSent && p.Channel == ChannelEmail {
				assert.False(t, p.Enabled)
				assert.Equal(t, DigestOff, p.DigestFrequency)
				found = true
			}
		}
		assert.True(t, found)
	})

	// B3. GetPreferences marks system-critical types.
	t.Run("B3_system_critical_flag", func(t *testing.T) {
		prefRepo := new(mockPreferenceRepo)
		prefRepo.On("GetAll", mock.Anything, mock.Anything).Return([]NotifyPreference{}, nil)
		svc := newTestService(new(mockNotificationRepo), prefRepo, new(mockIamService), new(mockCache))

		result, err := svc.GetPreferences(ctx, testFamilyScope())
		require.NoError(t, err)

		for _, p := range result {
			if IsSystemCritical(p.NotificationType) {
				assert.True(t, p.SystemCritical, "type %s should be system-critical", p.NotificationType)
			} else {
				assert.False(t, p.SystemCritical, "type %s should not be system-critical", p.NotificationType)
			}
		}
	})

	// B4. UpdatePreferences happy path calls repo.UpsertBatch.
	t.Run("B4_update_happy_path", func(t *testing.T) {
		prefRepo := new(mockPreferenceRepo)
		prefRepo.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		prefRepo.On("GetAll", mock.Anything, mock.Anything).Return([]NotifyPreference{}, nil)
		svc := newTestService(new(mockNotificationRepo), prefRepo, new(mockIamService), new(mockCache))

		_, err := svc.UpdatePreferences(ctx, UpdatePreferencesCommand{
			Preferences: []PreferenceUpdate{
				{NotificationType: TypeFriendRequestSent, Channel: ChannelEmail, Enabled: false},
			},
		}, testFamilyScope())
		require.NoError(t, err)
		prefRepo.AssertCalled(t, "UpsertBatch", mock.Anything, mock.Anything, mock.Anything)
	})

	// B5. UpdatePreferences rejects disabling system-critical type.
	t.Run("B5_reject_disable_system_critical", func(t *testing.T) {
		svc := newTestService(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		_, err := svc.UpdatePreferences(ctx, UpdatePreferencesCommand{
			Preferences: []PreferenceUpdate{
				{NotificationType: TypeContentFlagged, Channel: ChannelEmail, Enabled: false},
			},
		}, testFamilyScope())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCannotDisableSystemCritical)
	})

	// B6. UpdatePreferences rejects invalid notification type.
	t.Run("B6_reject_invalid_type", func(t *testing.T) {
		svc := newTestService(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		_, err := svc.UpdatePreferences(ctx, UpdatePreferencesCommand{
			Preferences: []PreferenceUpdate{
				{NotificationType: "bogus_type", Channel: ChannelEmail, Enabled: false},
			},
		}, testFamilyScope())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidNotificationType)
	})

	// B7. UpdatePreferences rejects invalid channel.
	t.Run("B7_reject_invalid_channel", func(t *testing.T) {
		svc := newTestService(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		_, err := svc.UpdatePreferences(ctx, UpdatePreferencesCommand{
			Preferences: []PreferenceUpdate{
				{NotificationType: TypeFriendRequestSent, Channel: "push", Enabled: false},
			},
		}, testFamilyScope())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidChannel)
	})

	// B8. Batch opt-out of 13 non-system-critical email types succeeds.
	t.Run("B8_batch_optout_non_critical", func(t *testing.T) {
		prefRepo := new(mockPreferenceRepo)
		prefRepo.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		prefRepo.On("GetAll", mock.Anything, mock.Anything).Return([]NotifyPreference{}, nil)
		svc := newTestService(new(mockNotificationRepo), prefRepo, new(mockIamService), new(mockCache))

		var updates []PreferenceUpdate
		for _, nt := range Phase1PreferenceTypes {
			if !IsSystemCritical(nt) {
				updates = append(updates, PreferenceUpdate{
					NotificationType: nt,
					Channel:          ChannelEmail,
					Enabled:          false,
				})
			}
		}
		assert.Equal(t, 13, len(updates))

		_, err := svc.UpdatePreferences(ctx, UpdatePreferencesCommand{Preferences: updates}, testFamilyScope())
		require.NoError(t, err)
	})

	// B9. Batch including system-critical disable fails.
	t.Run("B9_batch_with_system_critical_fails", func(t *testing.T) {
		svc := newTestService(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		updates := []PreferenceUpdate{
			{NotificationType: TypeFriendRequestSent, Channel: ChannelEmail, Enabled: false},
			{NotificationType: TypeContentFlagged, Channel: ChannelEmail, Enabled: false}, // system-critical
		}

		_, err := svc.UpdatePreferences(ctx, UpdatePreferencesCommand{Preferences: updates}, testFamilyScope())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCannotDisableSystemCritical)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group C: Notification CRUD — Service Layer (9 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestNotificationCRUD(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.New()
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})

	notifID := uuid.New()
	sampleNotif := NotifyNotification{
		ID: notifID, FamilyID: familyID, NotificationType: TypeFriendRequestSent,
		Category: CategorySocial, Title: "Test", Body: "Body", IsRead: false,
		CreatedAt: time.Now(), Metadata: json.RawMessage(`{}`),
	}

	// C1. ListNotifications returns sorted results with unread count.
	t.Run("C1_list_notifications", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("List", mock.Anything, mock.Anything, mock.Anything).Return([]NotifyNotification{sampleNotif}, nil)
		notifRepo.On("CountUnread", mock.Anything, mock.Anything).Return(int64(5), nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		result, err := svc.ListNotifications(ctx, NotificationListParams{}, &scope)
		require.NoError(t, err)
		assert.Equal(t, 1, len(result.Notifications))
		assert.Equal(t, int64(5), result.UnreadCount)
	})

	// C2. ListNotifications passes category filter.
	t.Run("C2_list_with_category_filter", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		cat := CategorySocial
		notifRepo.On("List", mock.Anything, mock.Anything, mock.Anything).Return([]NotifyNotification{sampleNotif}, nil)
		notifRepo.On("CountUnread", mock.Anything, mock.Anything).Return(int64(1), nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		result, err := svc.ListNotifications(ctx, NotificationListParams{Category: &cat}, &scope)
		require.NoError(t, err)
		assert.Equal(t, 1, len(result.Notifications))
	})

	// C3. ListNotifications passes unread_only filter.
	t.Run("C3_list_unread_only", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		unread := true
		notifRepo.On("List", mock.Anything, mock.Anything, mock.Anything).Return([]NotifyNotification{sampleNotif}, nil)
		notifRepo.On("CountUnread", mock.Anything, mock.Anything).Return(int64(1), nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		result, err := svc.ListNotifications(ctx, NotificationListParams{UnreadOnly: &unread}, &scope)
		require.NoError(t, err)
		assert.Equal(t, 1, len(result.Notifications))
	})

	// C4. MarkRead happy path.
	t.Run("C4_mark_read_happy", func(t *testing.T) {
		readNotif := sampleNotif
		readNotif.IsRead = true
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("MarkRead", mock.Anything, notifID, &scope).Return(true, nil)
		notifRepo.On("GetByID", mock.Anything, notifID, &scope).Return(&readNotif, nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		resp, err := svc.MarkRead(ctx, notifID, &scope)
		require.NoError(t, err)
		assert.True(t, resp.IsRead)
	})

	// C5. MarkRead already-read is idempotent.
	t.Run("C5_mark_read_idempotent", func(t *testing.T) {
		readNotif := sampleNotif
		readNotif.IsRead = true
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("MarkRead", mock.Anything, notifID, &scope).Return(true, nil)
		notifRepo.On("GetByID", mock.Anything, notifID, &scope).Return(&readNotif, nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		resp, err := svc.MarkRead(ctx, notifID, &scope)
		require.NoError(t, err)
		assert.True(t, resp.IsRead)
	})

	// C6. MarkRead not found.
	t.Run("C6_mark_read_not_found", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("MarkRead", mock.Anything, mock.Anything, &scope).Return(false, nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		_, err := svc.MarkRead(ctx, uuid.New(), &scope)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotificationNotFound)
	})

	// C7. MarkRead wrong family → ErrNotificationNotFound (not 403).
	t.Run("C7_mark_read_wrong_family", func(t *testing.T) {
		otherScope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.New()})
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("MarkRead", mock.Anything, mock.Anything, &otherScope).Return(false, nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		_, err := svc.MarkRead(ctx, notifID, &otherScope)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotificationNotFound)
	})

	// C8. MarkAllRead returns updated count.
	t.Run("C8_mark_all_read", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("MarkAllRead", mock.Anything, &scope, (*string)(nil)).Return(int64(10), nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		count, err := svc.MarkAllRead(ctx, &scope, nil)
		require.NoError(t, err)
		assert.Equal(t, int64(10), count)
	})

	// C9. MarkAllRead with category filter.
	t.Run("C9_mark_all_read_with_category", func(t *testing.T) {
		cat := CategorySocial
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("MarkAllRead", mock.Anything, &scope, &cat).Return(int64(3), nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		count, err := svc.MarkAllRead(ctx, &scope, &cat)
		require.NoError(t, err)
		assert.Equal(t, int64(3), count)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group D: CreateNotification Pipeline (10 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreateNotificationPipeline(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.New()

	makeCmd := func(notifType string) CreateNotificationCommand {
		return CreateNotificationCommand{
			FamilyID:         familyID,
			NotificationType: notifType,
			Title:            "Test Title",
			Body:             "Test Body",
			Metadata:         map[string]any{"source_event_id": uuid.New().String()},
		}
	}

	// D1. In-app enabled → creates notification record.
	t.Run("D1_inapp_enabled_creates_record", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		iamSvc := new(mockIamService)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, NotificationType: TypeFriendRequestSent,
			Category: CategorySocial, Title: "Test", Body: "Body",
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		svc := newTestServiceFull(notifRepo, prefRepo, iamSvc, new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.CreateNotification(ctx, makeCmd(TypeFriendRequestSent))
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
	})

	// D2. In-app disabled → skips creation.
	t.Run("D2_inapp_disabled_skips_creation", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelInApp).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelEmail).Return(false, nil)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.CreateNotification(ctx, makeCmd(TypeFriendRequestSent))
		require.NoError(t, err)
		notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	// D3. System-critical → skips preference check, always creates.
	t.Run("D3_system_critical_always_creates", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		iamSvc := new(mockIamService)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, NotificationType: TypeContentFlagged,
			Category: CategorySystem, Title: "Flagged", Body: "Content flagged",
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		iamSvc.On("GetFamilyPrimaryEmail", mock.Anything, familyID).Return("test@example.com", "Test", nil)
		svc := newTestServiceFull(notifRepo, prefRepo, iamSvc, new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.CreateNotification(ctx, makeCmd(TypeContentFlagged))
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
		// Preference IsEnabled should NOT have been called for system-critical type.
		prefRepo.AssertNotCalled(t, "IsEnabled", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	// D4. Publishes WebSocket frame via PubSub.
	t.Run("D4_publishes_websocket_frame", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		pubsubMock := new(mockPubSub)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, NotificationType: TypeFriendRequestSent,
			Category: CategorySocial, Title: "Test", Body: "Body",
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		expectedChannel := fmt.Sprintf("notifications:%s", familyID)
		pubsubMock.On("Publish", mock.Anything, expectedChannel, mock.Anything).Return(nil)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), pubsubMock, shared.NoopJobEnqueuer{})

		err := svc.CreateNotification(ctx, makeCmd(TypeFriendRequestSent))
		require.NoError(t, err)
		pubsubMock.AssertCalled(t, "Publish", mock.Anything, expectedChannel, mock.Anything)

		// Verify frame format.
		frameArg := pubsubMock.Calls[0].Arguments[2].([]byte)
		var frame WebSocketFrame
		require.NoError(t, json.Unmarshal(frameArg, &frame))
		assert.Equal(t, "notification", frame.MsgType)
	})

	// D5. Email enabled + immediate → enqueues SendEmailTask.
	t.Run("D5_email_enabled_enqueues_task", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		iamSvc := new(mockIamService)
		jobsMock := new(mockJobEnqueuer)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelEmail).Return(true, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, NotificationType: TypeFriendRequestSent,
			Category: CategorySocial, Title: "Test", Body: "Body",
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		iamSvc.On("GetFamilyPrimaryEmail", mock.Anything, familyID).Return("parent@example.com", "Parent", nil)
		jobsMock.On("Enqueue", mock.Anything, mock.AnythingOfType("SendEmailTaskPayload")).Return(nil)
		svc := newTestServiceFull(notifRepo, prefRepo, iamSvc, new(mockCache), shared.NoopPubSub{}, jobsMock)

		err := svc.CreateNotification(ctx, makeCmd(TypeFriendRequestSent))
		require.NoError(t, err)
		jobsMock.AssertCalled(t, "Enqueue", mock.Anything, mock.AnythingOfType("SendEmailTaskPayload"))
	})

	// D6. Email disabled → skips email.
	t.Run("D6_email_disabled_skips_email", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		jobsMock := new(mockJobEnqueuer)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, NotificationType: TypeFriendRequestSent,
			Category: CategorySocial, Title: "Test", Body: "Body",
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, jobsMock)

		err := svc.CreateNotification(ctx, makeCmd(TypeFriendRequestSent))
		require.NoError(t, err)
		jobsMock.AssertNotCalled(t, "Enqueue", mock.Anything, mock.Anything)
	})

	// D7. System-critical → always sends email regardless of preference.
	t.Run("D7_system_critical_always_emails", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		iamSvc := new(mockIamService)
		jobsMock := new(mockJobEnqueuer)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, NotificationType: TypeContentFlagged,
			Category: CategorySystem, Title: "Flagged", Body: "Content flagged",
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		iamSvc.On("GetFamilyPrimaryEmail", mock.Anything, familyID).Return("parent@example.com", "Parent", nil)
		jobsMock.On("Enqueue", mock.Anything, mock.AnythingOfType("SendEmailTaskPayload")).Return(nil)
		svc := newTestServiceFull(notifRepo, prefRepo, iamSvc, new(mockCache), shared.NoopPubSub{}, jobsMock)

		err := svc.CreateNotification(ctx, makeCmd(TypeContentFlagged))
		require.NoError(t, err)
		// No preference check for system-critical.
		prefRepo.AssertNotCalled(t, "IsEnabled", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		jobsMock.AssertCalled(t, "Enqueue", mock.Anything, mock.AnythingOfType("SendEmailTaskPayload"))
	})

	// D8. Duplicate source_event_id → returns nil (idempotent).
	t.Run("D8_duplicate_source_event_idempotent", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(true, nil)
		svc := newTestServiceFull(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.CreateNotification(ctx, makeCmd(TypeFriendRequestSent))
		require.NoError(t, err)
		notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	// D9. Email enqueue failure doesn't prevent in-app creation.
	t.Run("D9_email_failure_doesnt_prevent_inapp", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		iamSvc := new(mockIamService)
		jobsMock := new(mockJobEnqueuer)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelEmail).Return(true, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, NotificationType: TypeFriendRequestSent,
			Category: CategorySocial, Title: "Test", Body: "Body",
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		iamSvc.On("GetFamilyPrimaryEmail", mock.Anything, familyID).Return("parent@example.com", "Parent", nil)
		jobsMock.On("Enqueue", mock.Anything, mock.AnythingOfType("SendEmailTaskPayload")).Return(fmt.Errorf("redis down"))
		svc := newTestServiceFull(notifRepo, prefRepo, iamSvc, new(mockCache), shared.NoopPubSub{}, jobsMock)

		err := svc.CreateNotification(ctx, makeCmd(TypeFriendRequestSent))
		require.NoError(t, err) // In-app creation succeeded despite email failure.
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
	})

	// D10. WebSocket frame format: { msg_type: "notification", data: NotificationResponse }
	t.Run("D10_websocket_frame_format", func(t *testing.T) {
		frame := WebSocketFrame{
			MsgType: "notification",
			Data: NotificationResponse{
				ID: uuid.New(), NotificationType: TypeFriendRequestSent,
				Category: CategorySocial, Title: "Test", Body: "Body",
				IsRead: false, CreatedAt: time.Now(),
			},
		}
		data, err := json.Marshal(frame)
		require.NoError(t, err)

		var decoded map[string]any
		require.NoError(t, json.Unmarshal(data, &decoded))
		assert.Equal(t, "notification", decoded["msg_type"])
		assert.NotNil(t, decoded["data"])
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group E: Unsubscribe (5 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestUnsubscribe(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.New()

	// E1. Generate/verify unsubscribe token roundtrip.
	t.Run("E1_token_roundtrip", func(t *testing.T) {
		prefRepo := new(mockPreferenceRepo)
		prefRepo.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		svc := newTestService(new(mockNotificationRepo), prefRepo, new(mockIamService), new(mockCache))

		token, err := svc.GenerateUnsubscribeToken(familyID, TypeFriendRequestSent, ChannelEmail)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		err = svc.ProcessUnsubscribe(ctx, token)
		require.NoError(t, err)
		prefRepo.AssertCalled(t, "UpsertBatch", mock.Anything, mock.Anything, mock.MatchedBy(func(updates []PreferenceUpsert) bool {
			return len(updates) == 1 && updates[0].NotificationType == TypeFriendRequestSent && !updates[0].Enabled
		}))
	})

	// E2. Valid token disables preference.
	t.Run("E2_valid_token_disables", func(t *testing.T) {
		prefRepo := new(mockPreferenceRepo)
		prefRepo.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		svc := newTestService(new(mockNotificationRepo), prefRepo, new(mockIamService), new(mockCache))

		token, err := svc.GenerateUnsubscribeToken(familyID, TypeMessageReceived, ChannelEmail)
		require.NoError(t, err)

		err = svc.ProcessUnsubscribe(ctx, token)
		require.NoError(t, err)
	})

	// E3. Expired token → ErrInvalidUnsubscribeToken.
	t.Run("E3_expired_token", func(t *testing.T) {
		svc := newTestService(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		// Manually craft an expired token.
		payload := unsubscribePayload{
			FamilyID:         familyID.String(),
			NotificationType: TypeFriendRequestSent,
			Channel:          ChannelEmail,
			Exp:              time.Now().Add(-1 * time.Hour).Unix(), // Already expired
		}
		payloadJSON, _ := json.Marshal(payload)
		payloadB64 := base64.URLEncoding.EncodeToString(payloadJSON)
		sig := svc.signPayload(payloadB64)
		token := payloadB64 + "." + sig

		err := svc.ProcessUnsubscribe(ctx, token)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidUnsubscribeToken)
	})

	// E4. Invalid signature → ErrInvalidUnsubscribeToken.
	t.Run("E4_invalid_signature", func(t *testing.T) {
		svc := newTestService(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		token, _ := svc.GenerateUnsubscribeToken(familyID, TypeFriendRequestSent, ChannelEmail)
		// Tamper with the signature.
		token = token[:len(token)-5] + "XXXXX"

		err := svc.ProcessUnsubscribe(ctx, token)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidUnsubscribeToken)
	})

	// E5. System-critical type → ErrCannotDisableSystemCritical.
	t.Run("E5_system_critical_unsubscribe", func(t *testing.T) {
		svc := newTestService(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		token, err := svc.GenerateUnsubscribeToken(familyID, TypeContentFlagged, ChannelEmail)
		require.NoError(t, err)

		err = svc.ProcessUnsubscribe(ctx, token)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCannotDisableSystemCritical)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group F: Social Event Handlers (5 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestSocialEventHandlers(t *testing.T) {
	ctx := context.Background()

	setupCreateMocks := func(familyID uuid.UUID) (*mockNotificationRepo, *mockPreferenceRepo) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.FamilyID == familyID
		})).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, Category: CategorySocial,
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		return notifRepo, prefRepo
	}

	// F1. FriendRequestSent → notification for target (accepter) family.
	t.Run("F1_friend_request_sent", func(t *testing.T) {
		accepterID := uuid.New()
		notifRepo, prefRepo := setupCreateMocks(accepterID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleFriendRequestSent(ctx, FriendRequestSentEvent{
			FriendshipID:      uuid.New(),
			RequesterFamilyID: uuid.New(),
			AccepterFamilyID:  accepterID,
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.FamilyID == accepterID && cmd.NotificationType == TypeFriendRequestSent
		}))
	})

	// F2. FriendRequestAccepted → notification for requester family.
	t.Run("F2_friend_request_accepted", func(t *testing.T) {
		requesterID := uuid.New()
		notifRepo, prefRepo := setupCreateMocks(requesterID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleFriendRequestAccepted(ctx, FriendRequestAcceptedEvent{
			FriendshipID:      uuid.New(),
			RequesterFamilyID: requesterID,
			AccepterFamilyID:  uuid.New(),
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.FamilyID == requesterID && cmd.NotificationType == TypeFriendRequestAccepted
		}))
	})

	// F3. MessageSent → notification for recipient family.
	t.Run("F3_message_sent", func(t *testing.T) {
		recipientID := uuid.New()
		notifRepo, prefRepo := setupCreateMocks(recipientID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleMessageSent(ctx, MessageSentEvent{
			MessageID:         uuid.New(),
			ConversationID:    uuid.New(),
			SenderParentID:    uuid.New(),
			SenderFamilyID:    uuid.New(),
			RecipientParentID: uuid.New(),
			RecipientFamilyID: recipientID,
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.FamilyID == recipientID && cmd.NotificationType == TypeMessageReceived
		}))
	})

	// F4. EventCancelled → one notification per going family.
	t.Run("F4_event_cancelled_multiple", func(t *testing.T) {
		family1 := uuid.New()
		family2 := uuid.New()

		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, mock.Anything, mock.Anything, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, mock.Anything, mock.Anything, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), Category: CategorySocial,
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleEventCancelled(ctx, EventCancelledEvent{
			EventID:         uuid.New(),
			CreatorFamilyID: uuid.New(),
			Title:           "Study Group",
			GoingFamilyIDs:  []uuid.UUID{family1, family2},
		})
		require.NoError(t, err)
		assert.Equal(t, 2, len(notifRepo.Calls)-2) // 2 ExistsBySourceEvent + 2 Create = 4, minus 2 exists = 2 creates
	})

	// F5. EventCancelled with empty going list → noop.
	t.Run("F5_event_cancelled_empty_list", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		svc := newTestServiceFull(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleEventCancelled(ctx, EventCancelledEvent{
			EventID:        uuid.New(),
			GoingFamilyIDs: []uuid.UUID{},
		})
		require.NoError(t, err)
		notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group G: Method + Onboard Event Handlers (3 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestMethodOnboardHandlers(t *testing.T) {
	ctx := context.Background()

	setupMocks := func(familyID uuid.UUID) (*mockNotificationRepo, *mockPreferenceRepo) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID,
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		return notifRepo, prefRepo
	}

	// G1. FamilyMethodologyChanged → notification.
	t.Run("G1_methodology_changed", func(t *testing.T) {
		familyID := uuid.New()
		notifRepo, prefRepo := setupMocks(familyID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleFamilyMethodologyChanged(ctx, FamilyMethodologyChangedEvent{FamilyID: familyID})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.NotificationType == TypeMethodologyChanged
		}))
	})

	// G2. OnboardingCompleted → notification.
	t.Run("G2_onboarding_completed", func(t *testing.T) {
		familyID := uuid.New()
		notifRepo, prefRepo := setupMocks(familyID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleOnboardingCompleted(ctx, OnboardingCompletedEvent{FamilyID: familyID, Skipped: false})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.NotificationType == TypeOnboardingCompleted
		}))
	})

	// G3. OnboardingCompleted with skipped: true → no notification.
	t.Run("G3_onboarding_skipped", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		svc := newTestServiceFull(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleOnboardingCompleted(ctx, OnboardingCompletedEvent{FamilyID: uuid.New(), Skipped: true})
		require.NoError(t, err)
		notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group H: Learn Event Handlers (3 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestLearnEventHandlers(t *testing.T) {
	ctx := context.Background()

	setupMocks := func(familyID uuid.UUID) (*mockNotificationRepo, *mockPreferenceRepo) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID,
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		return notifRepo, prefRepo
	}

	// H1. MilestoneAchieved → notification with student name + description.
	t.Run("H1_milestone_achieved", func(t *testing.T) {
		familyID := uuid.New()
		notifRepo, prefRepo := setupMocks(familyID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleMilestoneAchieved(ctx, MilestoneAchievedEvent{
			FamilyID:    familyID,
			StudentID:   uuid.New(),
			StudentName: "Alice",
			Description: "Completed 100 hours of reading",
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.NotificationType == TypeMilestoneAchieved &&
				cmd.Title == "Alice achieved: Completed 100 hours of reading"
		}))
	})

	// H2. BookCompleted → notification with book title.
	t.Run("H2_book_completed", func(t *testing.T) {
		familyID := uuid.New()
		notifRepo, prefRepo := setupMocks(familyID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleBookCompleted(ctx, BookCompletedEvent{
			FamilyID:        familyID,
			StudentID:       uuid.New(),
			ReadingItemTitle: "Charlotte's Web",
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.NotificationType == TypeBookCompleted &&
				cmd.Title == "Finished 'Charlotte's Web'"
		}))
	})

	// H3. DataExportReady → notification with download URL.
	t.Run("H3_data_export_ready", func(t *testing.T) {
		familyID := uuid.New()
		notifRepo, prefRepo := setupMocks(familyID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleDataExportReady(ctx, DataExportReadyEvent{
			FamilyID: familyID,
			FileURL:  "https://cdn.example.com/export/123.zip",
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.NotificationType == TypeDataExportReady &&
				cmd.ActionURL != nil && *cmd.ActionURL == "https://cdn.example.com/export/123.zip"
		}))
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group I: Streak Detection (5 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestStreakDetection(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.New()
	studentID := uuid.New()
	today := time.Now().Format("2006-01-02")

	// I1. ActivityLogged increments Redis streak counter with 48h TTL.
	t.Run("I1_increments_streak_counter", func(t *testing.T) {
		cache := new(mockCache)
		streakKey := fmt.Sprintf("notify:streak:%s:%s", studentID, today)
		cache.On("IncrementWithExpiry", mock.Anything, streakKey, 48*time.Hour).Return(int64(1), nil)
		// No previous days.
		cache.On("Get", mock.Anything, mock.Anything).Return("", nil)

		svc := newTestServiceFull(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), cache, shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleActivityLogged(ctx, ActivityLoggedEvent{
			FamilyID: familyID, StudentID: studentID, ActivityDate: today,
		})
		require.NoError(t, err)
		cache.AssertCalled(t, "IncrementWithExpiry", mock.Anything, streakKey, 48*time.Hour)
	})

	// I2. 7-day streak → creates activity_streak notification.
	t.Run("I2_seven_day_streak", func(t *testing.T) {
		cache := new(mockCache)
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)

		streakKey := fmt.Sprintf("notify:streak:%s:%s", studentID, today)
		cache.On("IncrementWithExpiry", mock.Anything, streakKey, 48*time.Hour).Return(int64(1), nil)
		// Return "1" for previous 6 days (total = 7 consecutive), then miss to break loop.
		cache.On("Get", mock.Anything, mock.Anything).Return("1", nil).Times(6)
		cache.On("Get", mock.Anything, mock.Anything).Return("", nil).Once()

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID,
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)

		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), cache, shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleActivityLogged(ctx, ActivityLoggedEvent{
			FamilyID: familyID, StudentID: studentID, ActivityDate: today,
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.NotificationType == TypeActivityStreak
		}))
	})

	// I3. Non-milestone day count → no notification.
	t.Run("I3_non_milestone_no_notification", func(t *testing.T) {
		cache := new(mockCache)
		notifRepo := new(mockNotificationRepo)

		streakKey := fmt.Sprintf("notify:streak:%s:%s", studentID, today)
		cache.On("IncrementWithExpiry", mock.Anything, streakKey, 48*time.Hour).Return(int64(1), nil)
		// 5 previous days = 6 total (not a milestone).
		cache.On("Get", mock.Anything, mock.Anything).Return("1", nil).Times(5)
		// 6th previous day is a miss.
		cache.On("Get", mock.Anything, mock.Anything).Return("", nil).Once()

		svc := newTestServiceFull(notifRepo, new(mockPreferenceRepo), new(mockIamService), cache, shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleActivityLogged(ctx, ActivityLoggedEvent{
			FamilyID: familyID, StudentID: studentID, ActivityDate: today,
		})
		require.NoError(t, err)
		notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})

	// I4. 14-day streak → also triggers notification.
	t.Run("I4_fourteen_day_streak", func(t *testing.T) {
		cache := new(mockCache)
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)

		streakKey := fmt.Sprintf("notify:streak:%s:%s", studentID, today)
		cache.On("IncrementWithExpiry", mock.Anything, streakKey, 48*time.Hour).Return(int64(1), nil)
		cache.On("Get", mock.Anything, mock.Anything).Return("1", nil).Times(13)
		cache.On("Get", mock.Anything, mock.Anything).Return("", nil).Once()

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID,
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)

		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), cache, shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleActivityLogged(ctx, ActivityLoggedEvent{
			FamilyID: familyID, StudentID: studentID, ActivityDate: today,
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.NotificationType == TypeActivityStreak
		}))
	})

	// I5. Same student+date duplicate → no double count.
	t.Run("I5_duplicate_no_double_count", func(t *testing.T) {
		cache := new(mockCache)
		notifRepo := new(mockNotificationRepo)

		streakKey := fmt.Sprintf("notify:streak:%s:%s", studentID, today)
		// Second increment returns 2 → skip.
		cache.On("IncrementWithExpiry", mock.Anything, streakKey, 48*time.Hour).Return(int64(2), nil)

		svc := newTestServiceFull(notifRepo, new(mockPreferenceRepo), new(mockIamService), cache, shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleActivityLogged(ctx, ActivityLoggedEvent{
			FamilyID: familyID, StudentID: studentID, ActivityDate: today,
		})
		require.NoError(t, err)
		notifRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group J: Marketplace Event Handlers (3 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestMarketplaceHandlers(t *testing.T) {
	ctx := context.Background()

	setupMocks := func(familyID uuid.UUID) (*mockNotificationRepo, *mockPreferenceRepo) {
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, mock.Anything, ChannelEmail).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID,
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		return notifRepo, prefRepo
	}

	// J1. PurchaseCompleted → notification.
	t.Run("J1_purchase_completed", func(t *testing.T) {
		familyID := uuid.New()
		notifRepo, prefRepo := setupMocks(familyID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandlePurchaseCompleted(ctx, PurchaseCompletedEvent{
			FamilyID: familyID, PurchaseID: uuid.New(),
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.NotificationType == TypePurchaseCompleted
		}))
	})

	// J2. PurchaseRefunded → notification.
	t.Run("J2_purchase_refunded", func(t *testing.T) {
		familyID := uuid.New()
		notifRepo, prefRepo := setupMocks(familyID)
		svc := newTestServiceFull(notifRepo, prefRepo, new(mockIamService), new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandlePurchaseRefunded(ctx, PurchaseRefundedEvent{
			FamilyID: familyID, PurchaseID: uuid.New(),
		})
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.NotificationType == TypePurchaseRefunded
		}))
	})

	// J3. CreatorOnboarded → looks up family via parent_id, then creates notification.
	t.Run("J3_creator_onboarded", func(t *testing.T) {
		familyID := uuid.New()
		parentID := uuid.New()
		notifRepo, prefRepo := setupMocks(familyID)
		iamSvc := new(mockIamService)
		iamSvc.On("GetFamilyIDForParent", mock.Anything, parentID).Return(familyID, nil)
		svc := newTestServiceFull(notifRepo, prefRepo, iamSvc, new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleCreatorOnboarded(ctx, CreatorOnboardedEvent{
			CreatorID: uuid.New(), ParentID: parentID, StoreName: "Alice's Shop",
		})
		require.NoError(t, err)
		iamSvc.AssertCalled(t, "GetFamilyIDForParent", mock.Anything, parentID)
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.MatchedBy(func(cmd CreateNotification) bool {
			return cmd.FamilyID == familyID && cmd.NotificationType == TypeCreatorOnboarded
		}))
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group K: ContentFlagged Handler (1 test)
// ═══════════════════════════════════════════════════════════════════════════════

func TestContentFlaggedHandler(t *testing.T) {
	ctx := context.Background()

	// K1. System-critical → always delivers regardless of preferences.
	t.Run("K1_content_flagged_always_delivers", func(t *testing.T) {
		familyID := uuid.New()
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		iamSvc := new(mockIamService)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, Category: CategorySystem,
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		iamSvc.On("GetFamilyPrimaryEmail", mock.Anything, familyID).Return("parent@example.com", "Parent", nil)

		svc := newTestServiceFull(notifRepo, prefRepo, iamSvc, new(mockCache), shared.NoopPubSub{}, shared.NoopJobEnqueuer{})

		err := svc.HandleContentFlagged(ctx, ContentFlaggedEvent{
			FamilyID: familyID, ContentID: uuid.New(), ContentType: "post", Reason: "inappropriate",
		})
		require.NoError(t, err)
		// Must create notification without checking preferences.
		notifRepo.AssertCalled(t, "Create", mock.Anything, mock.Anything)
		prefRepo.AssertNotCalled(t, "IsEnabled", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group M: Error Mapping (3 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestErrorMapping(t *testing.T) {
	// M1. All sentinels map to correct HTTP status/code.
	t.Run("M1_all_sentinels_mapped", func(t *testing.T) {
		tests := []struct {
			err        error
			wantCode   string
			wantStatus int
		}{
			{ErrNotificationNotFound, "notification_not_found", 404},
			{ErrCannotDisableSystemCritical, "cannot_disable_system_critical", 422},
			{ErrInvalidNotificationType, "invalid_notification_type", 422},
			{ErrInvalidChannel, "invalid_channel", 422},
			{ErrInvalidCategory, "invalid_category", 422},
			{ErrInvalidUnsubscribeToken, "invalid_unsubscribe_token", 400},
			{ErrDuplicateNotification, "duplicate_notification", 409},
		}
		for _, tt := range tests {
			t.Run(tt.err.Error(), func(t *testing.T) {
				ne := &NotifyError{Err: tt.err}
				appErr := ne.toAppError()
				assert.Equal(t, tt.wantCode, appErr.Code)
				assert.Equal(t, tt.wantStatus, appErr.StatusCode)
			})
		}
	})

	// M2. NotificationNotOwned → 404 (enumeration prevention).
	t.Run("M2_not_owned_maps_to_404", func(t *testing.T) {
		ne := &NotifyError{Err: ErrNotificationNotOwned}
		appErr := ne.toAppError()
		assert.Equal(t, 404, appErr.StatusCode)
		assert.Equal(t, "notification_not_found", appErr.Code)
	})

	// M3. Unknown error → generic 500.
	t.Run("M3_unknown_error_500", func(t *testing.T) {
		ne := &NotifyError{Err: fmt.Errorf("some internal issue")}
		appErr := ne.toAppError()
		assert.Equal(t, 500, appErr.StatusCode)
		assert.Equal(t, "internal_error", appErr.Code)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Additional Mock Types (PubSub, JobEnqueuer)
// ═══════════════════════════════════════════════════════════════════════════════

type mockPubSub struct{ mock.Mock }

func (m *mockPubSub) Publish(ctx context.Context, channel string, message []byte) error {
	args := m.Called(ctx, channel, message)
	return args.Error(0)
}
func (m *mockPubSub) Subscribe(ctx context.Context, channel string) (shared.Subscription, error) {
	args := m.Called(ctx, channel)
	return args.Get(0).(shared.Subscription), args.Error(1)
}
func (m *mockPubSub) Close() error { return nil }

type mockJobEnqueuer struct{ mock.Mock }

func (m *mockJobEnqueuer) Enqueue(ctx context.Context, payload shared.JobPayload) error {
	args := m.Called(ctx, payload)
	return args.Error(0)
}
func (m *mockJobEnqueuer) EnqueueIn(ctx context.Context, payload shared.JobPayload, delay time.Duration) error {
	args := m.Called(ctx, payload, delay)
	return args.Error(0)
}
func (m *mockJobEnqueuer) Close() error { return nil }

// ═══════════════════════════════════════════════════════════════════════════════
// Group N: Gap Fix Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestGapFixes(t *testing.T) {
	ctx := context.Background()

	// N1. Email template model includes unsubscribe_url. [GAP 2]
	t.Run("N1_email_template_includes_unsubscribe_url", func(t *testing.T) {
		familyID := uuid.New()
		notifRepo := new(mockNotificationRepo)
		prefRepo := new(mockPreferenceRepo)
		iamSvc := new(mockIamService)
		jobsMock := new(mockJobEnqueuer)

		notifRepo.On("ExistsBySourceEvent", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelInApp).Return(true, nil)
		prefRepo.On("IsEnabled", mock.Anything, familyID, TypeFriendRequestSent, ChannelEmail).Return(true, nil)
		notifRepo.On("Create", mock.Anything, mock.Anything).Return(&NotifyNotification{
			ID: uuid.New(), FamilyID: familyID, NotificationType: TypeFriendRequestSent,
			Category: CategorySocial, Title: "Test", Body: "Body",
			Metadata: json.RawMessage(`{}`), CreatedAt: time.Now(),
		}, nil)
		iamSvc.On("GetFamilyPrimaryEmail", mock.Anything, familyID).Return("test@example.com", "Test", nil)
		jobsMock.On("Enqueue", mock.Anything, mock.AnythingOfType("SendEmailTaskPayload")).Return(nil)
		svc := newTestServiceFull(notifRepo, prefRepo, iamSvc, new(mockCache), shared.NoopPubSub{}, jobsMock)

		err := svc.CreateNotification(ctx, CreateNotificationCommand{
			FamilyID:         familyID,
			NotificationType: TypeFriendRequestSent,
			Title:            "Test",
			Body:             "Body",
			Metadata:         map[string]any{"source_event_id": uuid.New().String()},
		})
		require.NoError(t, err)
		jobsMock.AssertCalled(t, "Enqueue", mock.Anything, mock.MatchedBy(func(p SendEmailTaskPayload) bool {
			url, ok := p.TemplateModel["unsubscribe_url"].(string)
			return ok && url != "" && len(url) > 30
		}))
	})

	// N2. Invalid digest frequency returns ErrInvalidDigestFrequency. [GAP 6]
	t.Run("N2_invalid_digest_frequency", func(t *testing.T) {
		svc := newTestService(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		badFreq := "every_five_minutes"
		_, err := svc.UpdatePreferences(ctx, UpdatePreferencesCommand{
			Preferences: []PreferenceUpdate{
				{NotificationType: TypeFriendRequestSent, Channel: ChannelEmail, Enabled: true, DigestFrequency: &badFreq},
			},
		}, testFamilyScope())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidDigestFrequency)
	})

	// N3. Valid digest frequency is accepted. [GAP 6 negative]
	t.Run("N3_valid_digest_frequency_accepted", func(t *testing.T) {
		prefRepo := new(mockPreferenceRepo)
		prefRepo.On("UpsertBatch", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		prefRepo.On("GetAll", mock.Anything, mock.Anything).Return([]NotifyPreference{}, nil)
		svc := newTestService(new(mockNotificationRepo), prefRepo, new(mockIamService), new(mockCache))

		daily := DigestDaily
		_, err := svc.UpdatePreferences(ctx, UpdatePreferencesCommand{
			Preferences: []PreferenceUpdate{
				{NotificationType: TypeFriendRequestSent, Channel: ChannelEmail, Enabled: true, DigestFrequency: &daily},
			},
		}, testFamilyScope())
		require.NoError(t, err)
	})

	// N4. ErrInvalidDigestFrequency maps to 422. [GAP 7]
	t.Run("N4_digest_frequency_error_maps_to_422", func(t *testing.T) {
		ne := &NotifyError{Err: ErrInvalidDigestFrequency}
		appErr := ne.toAppError()
		assert.Equal(t, 422, appErr.StatusCode)
		assert.Equal(t, "invalid_digest_frequency", appErr.Code)
	})

	// N5. ErrEmailDeliveryFailed maps to 502. [GAP 7]
	t.Run("N5_email_delivery_failed_maps_to_502", func(t *testing.T) {
		ne := &NotifyError{Err: ErrEmailDeliveryFailed}
		appErr := ne.toAppError()
		assert.Equal(t, 502, appErr.StatusCode)
		assert.Equal(t, "email_delivery_failed", appErr.Code)
	})

	// N6. SendEmail enqueues a SendEmailTask. [GAP 10]
	t.Run("N6_send_email_enqueues_task", func(t *testing.T) {
		jobsMock := new(mockJobEnqueuer)
		jobsMock.On("Enqueue", mock.Anything, mock.AnythingOfType("SendEmailTaskPayload")).Return(nil)
		svc := newTestServiceFull(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache), shared.NoopPubSub{}, jobsMock)

		err := svc.SendEmail(ctx, SendEmailCommand{
			To:            "test@example.com",
			TemplateAlias: "digest-daily",
			TemplateModel: map[string]any{"key": "value"},
		})
		require.NoError(t, err)
		jobsMock.AssertCalled(t, "Enqueue", mock.Anything, mock.MatchedBy(func(p SendEmailTaskPayload) bool {
			return p.To == "test@example.com" && p.TemplateAlias == "digest-daily"
		}))
	})

	// N7. SendEmail propagates enqueue failure. [GAP 10]
	t.Run("N7_send_email_enqueue_failure", func(t *testing.T) {
		jobsMock := new(mockJobEnqueuer)
		jobsMock.On("Enqueue", mock.Anything, mock.AnythingOfType("SendEmailTaskPayload")).Return(fmt.Errorf("redis down"))
		svc := newTestServiceFull(new(mockNotificationRepo), new(mockPreferenceRepo), new(mockIamService), new(mockCache), shared.NoopPubSub{}, jobsMock)

		err := svc.SendEmail(ctx, SendEmailCommand{
			To:            "test@example.com",
			TemplateAlias: "digest-daily",
			TemplateModel: map[string]any{},
		})
		require.Error(t, err)
	})

	// N8. DeleteByFamily interfaces are callable. [GAP 8]
	t.Run("N8_delete_by_family_interfaces", func(t *testing.T) {
		familyID := uuid.New()

		notifRepo := new(mockNotificationRepo)
		notifRepo.On("DeleteByFamily", mock.Anything, familyID).Return(nil)
		err := notifRepo.DeleteByFamily(ctx, familyID)
		require.NoError(t, err)
		notifRepo.AssertCalled(t, "DeleteByFamily", mock.Anything, familyID)

		prefRepo := new(mockPreferenceRepo)
		prefRepo.On("DeleteByFamily", mock.Anything, familyID).Return(nil)
		err = prefRepo.DeleteByFamily(ctx, familyID)
		require.NoError(t, err)
		prefRepo.AssertCalled(t, "DeleteByFamily", mock.Anything, familyID)
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group P: Cursor Pagination (4 tests)
// ═══════════════════════════════════════════════════════════════════════════════

func TestCursorPagination(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.New()
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: familyID})

	makeNotifs := func(count int) []NotifyNotification {
		notifs := make([]NotifyNotification, count)
		for i := range count {
			notifs[i] = NotifyNotification{
				ID: uuid.New(), FamilyID: familyID, NotificationType: TypeFriendRequestSent,
				Category: CategorySocial, Title: fmt.Sprintf("Notif %d", i), Body: "Body",
				IsRead: false, CreatedAt: time.Now().Add(-time.Duration(i) * time.Minute),
				Metadata: json.RawMessage(`{}`),
			}
		}
		return notifs
	}

	// P1. Empty result → no NextCursor.
	t.Run("P1_empty_result_no_cursor", func(t *testing.T) {
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("List", mock.Anything, mock.Anything, mock.Anything).Return([]NotifyNotification{}, nil)
		notifRepo.On("CountUnread", mock.Anything, mock.Anything).Return(int64(0), nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		result, err := svc.ListNotifications(ctx, NotificationListParams{}, &scope)
		require.NoError(t, err)
		assert.Empty(t, result.Notifications)
		assert.Nil(t, result.NextCursor)
	})

	// P2. Partial page (fewer than limit) → no NextCursor.
	t.Run("P2_partial_page_no_cursor", func(t *testing.T) {
		limit := uint8(20)
		notifs := makeNotifs(5) // Only 5 rows returned (repo would return limit+1 if more existed)
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("List", mock.Anything, mock.Anything, mock.Anything).Return(notifs, nil)
		notifRepo.On("CountUnread", mock.Anything, mock.Anything).Return(int64(5), nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		result, err := svc.ListNotifications(ctx, NotificationListParams{Limit: &limit}, &scope)
		require.NoError(t, err)
		assert.Equal(t, 5, len(result.Notifications))
		assert.Nil(t, result.NextCursor)
	})

	// P3. Full page with extra row → NextCursor populated, extra row trimmed.
	t.Run("P3_full_page_with_next_cursor", func(t *testing.T) {
		limit := uint8(3)
		notifs := makeNotifs(4) // limit+1 rows returned → hasMore
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("List", mock.Anything, mock.Anything, mock.Anything).Return(notifs, nil)
		notifRepo.On("CountUnread", mock.Anything, mock.Anything).Return(int64(10), nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		result, err := svc.ListNotifications(ctx, NotificationListParams{Limit: &limit}, &scope)
		require.NoError(t, err)
		assert.Equal(t, 3, len(result.Notifications), "extra row should be trimmed")
		require.NotNil(t, result.NextCursor, "NextCursor should be populated")

		// Verify cursor decodes to the last visible row's ID and CreatedAt.
		cursorID, cursorTS, decErr := shared.DecodeCursor(*result.NextCursor)
		require.NoError(t, decErr)
		lastVisible := notifs[2]
		assert.Equal(t, lastVisible.ID, cursorID)
		assert.Equal(t, lastVisible.CreatedAt.UnixMilli(), cursorTS.UnixMilli())
	})

	// P4. Exactly limit rows → no NextCursor (no extra row means no more pages).
	t.Run("P4_exact_limit_no_cursor", func(t *testing.T) {
		limit := uint8(5)
		notifs := makeNotifs(5) // Exactly limit rows, no extra
		notifRepo := new(mockNotificationRepo)
		notifRepo.On("List", mock.Anything, mock.Anything, mock.Anything).Return(notifs, nil)
		notifRepo.On("CountUnread", mock.Anything, mock.Anything).Return(int64(5), nil)
		svc := newTestService(notifRepo, new(mockPreferenceRepo), new(mockIamService), new(mockCache))

		result, err := svc.ListNotifications(ctx, NotificationListParams{Limit: &limit}, &scope)
		require.NoError(t, err)
		assert.Equal(t, 5, len(result.Notifications))
		assert.Nil(t, result.NextCursor, "no extra row means no more pages")
	})
}

