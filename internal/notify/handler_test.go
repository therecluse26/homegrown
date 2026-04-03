package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// ─── Test Helpers ────────────────────────────────────────────────────────────

type echoValidator struct{ v *validator.Validate }

func (cv *echoValidator) Validate(i any) error { return cv.v.Struct(i) }

var (
	handlerFamilyID = uuid.Must(uuid.NewV7())
	handlerParentID = uuid.Must(uuid.NewV7())
)

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e
}

func setupNotifyRoutes(e *echo.Echo, svc NotificationService) {
	auth := e.Group("/v1")
	auth.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			shared.SetAuthContext(c, &shared.AuthContext{
				ParentID:           handlerParentID,
				FamilyID:           handlerFamilyID,
				CoppaConsentStatus: "consented",
			})
			return next(c)
		}
	})
	public := e.Group("/v1")
	NewHandler(svc, "test-secret").Register(auth, public)
}

// ─── Function-Pointer Mock Service ──────────────────────────────────────────

type handlerMockService struct {
	listNotificationsFn    func(ctx context.Context, params NotificationListParams, scope *shared.FamilyScope) (*NotificationListResponse, error)
	markReadFn             func(ctx context.Context, id uuid.UUID, scope *shared.FamilyScope) (NotificationResponse, error)
	markAllReadFn          func(ctx context.Context, scope *shared.FamilyScope, category *string) (int64, error)
	getPreferencesFn       func(ctx context.Context, scope *shared.FamilyScope) ([]PreferenceResponse, error)
	updatePreferencesFn    func(ctx context.Context, cmd UpdatePreferencesCommand, scope *shared.FamilyScope) ([]PreferenceResponse, error)
	processUnsubscribeFn   func(ctx context.Context, token string) error
	generateUnsubscribeFn  func(familyID uuid.UUID, notificationType string, channel string) (string, error)
	createNotificationFn   func(ctx context.Context, cmd CreateNotificationCommand) error
	getUnreadCountFn       func(ctx context.Context, scope *shared.FamilyScope) (int64, error)
}

func newHandlerMockService() *handlerMockService { return &handlerMockService{} }

func (m *handlerMockService) ListNotifications(ctx context.Context, params NotificationListParams, scope *shared.FamilyScope) (*NotificationListResponse, error) {
	if m.listNotificationsFn != nil {
		return m.listNotificationsFn(ctx, params, scope)
	}
	panic("ListNotifications not stubbed")
}
func (m *handlerMockService) MarkRead(ctx context.Context, id uuid.UUID, scope *shared.FamilyScope) (NotificationResponse, error) {
	if m.markReadFn != nil {
		return m.markReadFn(ctx, id, scope)
	}
	panic("MarkRead not stubbed")
}
func (m *handlerMockService) MarkAllRead(ctx context.Context, scope *shared.FamilyScope, category *string) (int64, error) {
	if m.markAllReadFn != nil {
		return m.markAllReadFn(ctx, scope, category)
	}
	panic("MarkAllRead not stubbed")
}
func (m *handlerMockService) GetPreferences(ctx context.Context, scope *shared.FamilyScope) ([]PreferenceResponse, error) {
	if m.getPreferencesFn != nil {
		return m.getPreferencesFn(ctx, scope)
	}
	panic("GetPreferences not stubbed")
}
func (m *handlerMockService) UpdatePreferences(ctx context.Context, cmd UpdatePreferencesCommand, scope *shared.FamilyScope) ([]PreferenceResponse, error) {
	if m.updatePreferencesFn != nil {
		return m.updatePreferencesFn(ctx, cmd, scope)
	}
	panic("UpdatePreferences not stubbed")
}
func (m *handlerMockService) ProcessUnsubscribe(ctx context.Context, token string) error {
	if m.processUnsubscribeFn != nil {
		return m.processUnsubscribeFn(ctx, token)
	}
	panic("ProcessUnsubscribe not stubbed")
}
func (m *handlerMockService) GenerateUnsubscribeToken(familyID uuid.UUID, notificationType string, channel string) (string, error) {
	if m.generateUnsubscribeFn != nil {
		return m.generateUnsubscribeFn(familyID, notificationType, channel)
	}
	panic("GenerateUnsubscribeToken not stubbed")
}
func (m *handlerMockService) CreateNotification(ctx context.Context, cmd CreateNotificationCommand) error {
	if m.createNotificationFn != nil {
		return m.createNotificationFn(ctx, cmd)
	}
	panic("CreateNotification not stubbed")
}
func (m *handlerMockService) GetUnreadCount(ctx context.Context, scope *shared.FamilyScope) (int64, error) {
	if m.getUnreadCountFn != nil {
		return m.getUnreadCountFn(ctx, scope)
	}
	return 0, nil
}
func (m *handlerMockService) SendEmail(_ context.Context, _ SendEmailCommand) error {
	return nil
}

// Stub all event handlers — not tested at the handler level.
func (m *handlerMockService) HandleFriendRequestSent(_ context.Context, _ FriendRequestSentEvent) error {
	return nil
}
func (m *handlerMockService) HandleFriendRequestAccepted(_ context.Context, _ FriendRequestAcceptedEvent) error {
	return nil
}
func (m *handlerMockService) HandleMessageSent(_ context.Context, _ MessageSentEvent) error {
	return nil
}
func (m *handlerMockService) HandleEventCancelled(_ context.Context, _ EventCancelledEvent) error {
	return nil
}
func (m *handlerMockService) HandleFamilyMethodologyChanged(_ context.Context, _ FamilyMethodologyChangedEvent) error {
	return nil
}
func (m *handlerMockService) HandleOnboardingCompleted(_ context.Context, _ OnboardingCompletedEvent) error {
	return nil
}
func (m *handlerMockService) HandleActivityLogged(_ context.Context, _ ActivityLoggedEvent) error {
	return nil
}
func (m *handlerMockService) HandleMilestoneAchieved(_ context.Context, _ MilestoneAchievedEvent) error {
	return nil
}
func (m *handlerMockService) HandleBookCompleted(_ context.Context, _ BookCompletedEvent) error {
	return nil
}
func (m *handlerMockService) HandleDataExportReady(_ context.Context, _ DataExportReadyEvent) error {
	return nil
}
func (m *handlerMockService) HandlePurchaseCompleted(_ context.Context, _ PurchaseCompletedEvent) error {
	return nil
}
func (m *handlerMockService) HandlePurchaseRefunded(_ context.Context, _ PurchaseRefundedEvent) error {
	return nil
}
func (m *handlerMockService) HandleCreatorOnboarded(_ context.Context, _ CreatorOnboardedEvent) error {
	return nil
}
func (m *handlerMockService) HandleContentFlagged(_ context.Context, _ ContentFlaggedEvent) error {
	return nil
}
func (m *handlerMockService) HandleCoParentAdded(_ context.Context, _ CoParentAddedEvent) error {
	return nil
}
func (m *handlerMockService) HandleFamilyDeletionScheduled(_ context.Context, _ FamilyDeletionScheduledEvent) error {
	return nil
}
func (m *handlerMockService) HandleSubscriptionCreated(_ context.Context, _ SubscriptionCreatedEvent) error {
	return nil
}
func (m *handlerMockService) HandleSubscriptionChanged(_ context.Context, _ SubscriptionChangedEvent) error {
	return nil
}
func (m *handlerMockService) HandleSubscriptionCancelled(_ context.Context, _ SubscriptionCancelledEvent) error {
	return nil
}
func (m *handlerMockService) HandlePayoutCompleted(_ context.Context, _ PayoutCompletedEvent) error {
	return nil
}
func (m *handlerMockService) HandleRecommendationsReady(_ context.Context, _ RecommendationsReadyEvent) error {
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Group L: HTTP Handler Tests [08-notify §4]
// ═══════════════════════════════════════════════════════════════════════════════

// L1. GET /v1/notifications — 200 success.
func TestListNotifications_Success(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	mock.listNotificationsFn = func(_ context.Context, _ NotificationListParams, _ *shared.FamilyScope) (*NotificationListResponse, error) {
		return &NotificationListResponse{
			Notifications: []NotificationResponse{{ID: uuid.Must(uuid.NewV7()), Title: "Test"}},
			UnreadCount:   1,
		}, nil
	}
	setupNotifyRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/notifications", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp NotificationListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Notifications) != 1 {
		t.Errorf("expected 1 notification, got %d", len(resp.Notifications))
	}
}

// L2. PATCH /v1/notifications/:id/read — 200 success.
func TestMarkRead_Success(t *testing.T) {
	e := newTestEcho()
	notifID := uuid.Must(uuid.NewV7())
	mock := newHandlerMockService()
	mock.markReadFn = func(_ context.Context, id uuid.UUID, _ *shared.FamilyScope) (NotificationResponse, error) {
		return NotificationResponse{ID: id, IsRead: true}, nil
	}
	setupNotifyRoutes(e, mock)

	req := httptest.NewRequest(http.MethodPatch, "/v1/notifications/"+notifID.String()+"/read", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// L3. PATCH /v1/notifications/:id/read — 404 not found.
func TestMarkRead_NotFound(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	mock.markReadFn = func(_ context.Context, _ uuid.UUID, _ *shared.FamilyScope) (NotificationResponse, error) {
		return NotificationResponse{}, &NotifyError{Err: ErrNotificationNotFound}
	}
	setupNotifyRoutes(e, mock)

	req := httptest.NewRequest(http.MethodPatch, "/v1/notifications/"+uuid.Must(uuid.NewV7()).String()+"/read", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// L4. PATCH /v1/notifications/:id/read — 400 invalid UUID.
func TestMarkRead_InvalidID(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	setupNotifyRoutes(e, mock)

	req := httptest.NewRequest(http.MethodPatch, "/v1/notifications/not-a-uuid/read", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// L5. PATCH /v1/notifications/read-all — 200 success.
func TestMarkAllRead_Success(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	mock.markAllReadFn = func(_ context.Context, _ *shared.FamilyScope, _ *string) (int64, error) {
		return 5, nil
	}
	setupNotifyRoutes(e, mock)

	req := httptest.NewRequest(http.MethodPatch, "/v1/notifications/read-all", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp MarkAllReadResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.UpdatedCount != 5 {
		t.Errorf("expected updated_count 5, got %d", resp.UpdatedCount)
	}
}

// L6. GET /v1/notifications/preferences — 200 success.
func TestGetPreferences_Success(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	mock.getPreferencesFn = func(_ context.Context, _ *shared.FamilyScope) ([]PreferenceResponse, error) {
		return []PreferenceResponse{{NotificationType: TypeFriendRequestSent, Channel: ChannelInApp, Enabled: true}}, nil
	}
	setupNotifyRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/notifications/preferences", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// L7. PATCH /v1/notifications/preferences — 200 success.
func TestUpdatePreferences_Success(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	mock.updatePreferencesFn = func(_ context.Context, _ UpdatePreferencesCommand, _ *shared.FamilyScope) ([]PreferenceResponse, error) {
		return []PreferenceResponse{{NotificationType: TypeFriendRequestSent, Channel: ChannelEmail, Enabled: false}}, nil
	}
	setupNotifyRoutes(e, mock)

	body := `{"preferences":[{"notification_type":"friend_request_sent","channel":"email","enabled":false}]}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/notifications/preferences", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// L8. PATCH /v1/notifications/preferences — 422 system-critical disable.
func TestUpdatePreferences_SystemCritical(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	mock.updatePreferencesFn = func(_ context.Context, _ UpdatePreferencesCommand, _ *shared.FamilyScope) ([]PreferenceResponse, error) {
		return nil, &NotifyError{Err: ErrCannotDisableSystemCritical}
	}
	setupNotifyRoutes(e, mock)

	body := `{"preferences":[{"notification_type":"content_flagged","channel":"email","enabled":false}]}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/notifications/preferences", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

// L9. GET /v1/notifications/unsubscribe — 200 success returns HTML.
func TestUnsubscribe_Success(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	mock.processUnsubscribeFn = func(_ context.Context, _ string) error {
		return nil
	}
	setupNotifyRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/notifications/unsubscribe?token=validtoken", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	// Must return HTML, not JSON — email clients don't execute JavaScript. [CAN-SPAM]
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %s", ct)
	}
	if !strings.Contains(rec.Body.String(), "You've been unsubscribed") {
		t.Error("expected success HTML to contain unsubscribe confirmation")
	}
}

// L10. GET /v1/notifications/unsubscribe — 400 missing token returns HTML error.
func TestUnsubscribe_MissingToken(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	setupNotifyRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/notifications/unsubscribe", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %s", ct)
	}
}

// L11. GET /v1/notifications/unsubscribe — error case returns HTML error page.
func TestUnsubscribe_ErrorReturnsHTML(t *testing.T) {
	e := newTestEcho()
	mock := newHandlerMockService()
	mock.processUnsubscribeFn = func(_ context.Context, _ string) error {
		return &NotifyError{Err: ErrInvalidUnsubscribeToken}
	}
	setupNotifyRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/notifications/unsubscribe?token=invalidtoken", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content-type, got %s", ct)
	}
	if !strings.Contains(rec.Body.String(), "Unable to unsubscribe") {
		t.Error("expected error HTML page content")
	}
}
