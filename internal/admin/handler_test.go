package admin

// Handler tests for the admin domain. [16-admin §4]

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// ─── Mock AdminService ────────────────────────────────────────────────────────

type mockAdminService struct {
	searchUsersFn    func(ctx context.Context, auth *shared.AuthContext, query *UserSearchQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error)
	listFlagsFn      func(ctx context.Context, auth *shared.AuthContext) ([]FeatureFlag, error)
	getSystemHealthFn func(ctx context.Context, auth *shared.AuthContext) (*SystemHealthResponse, error)
}

func (m *mockAdminService) SearchUsers(ctx context.Context, auth *shared.AuthContext, query *UserSearchQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error) {
	if m.searchUsersFn != nil {
		return m.searchUsersFn(ctx, auth, query, pagination)
	}
	return &shared.PaginatedResponse[AdminUserSummary]{Data: []AdminUserSummary{}}, nil
}
func (m *mockAdminService) GetUserDetail(_ context.Context, _ *shared.AuthContext, _ uuid.UUID) (*AdminUserDetail, error) {
	return &AdminUserDetail{}, nil
}
func (m *mockAdminService) GetUserAuditTrail(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ *shared.PaginationParams) (*shared.PaginatedResponse[AuditLogEntry], error) {
	return &shared.PaginatedResponse[AuditLogEntry]{Data: []AuditLogEntry{}}, nil
}
func (m *mockAdminService) SuspendUser(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockAdminService) UnsuspendUser(_ context.Context, _ *shared.AuthContext, _ uuid.UUID) error {
	return nil
}
func (m *mockAdminService) BanUser(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockAdminService) ListFlags(ctx context.Context, auth *shared.AuthContext) ([]FeatureFlag, error) {
	if m.listFlagsFn != nil {
		return m.listFlagsFn(ctx, auth)
	}
	return []FeatureFlag{}, nil
}
func (m *mockAdminService) GetFlag(_ context.Context, _ *shared.AuthContext, _ string) (*FeatureFlag, error) {
	return &FeatureFlag{}, nil
}
func (m *mockAdminService) CreateFlag(_ context.Context, _ *shared.AuthContext, _ *CreateFlagInput) (*FeatureFlag, error) {
	return &FeatureFlag{}, nil
}
func (m *mockAdminService) UpdateFlag(_ context.Context, _ *shared.AuthContext, _ string, _ *UpdateFlagInput) (*FeatureFlag, error) {
	return &FeatureFlag{}, nil
}
func (m *mockAdminService) DeleteFlag(_ context.Context, _ *shared.AuthContext, _ string) error {
	return nil
}
func (m *mockAdminService) IsFlagEnabled(_ context.Context, _ string, _ *uuid.UUID) (bool, error) {
	return false, nil
}
func (m *mockAdminService) GetModerationQueue(_ context.Context, _ *shared.AuthContext, _ *shared.PaginationParams) (*shared.PaginatedResponse[ModerationQueueItem], error) {
	return &shared.PaginatedResponse[ModerationQueueItem]{Data: []ModerationQueueItem{}}, nil
}
func (m *mockAdminService) GetModerationQueueItem(_ context.Context, _ *shared.AuthContext, _ uuid.UUID) (*ModerationQueueItem, error) {
	return &ModerationQueueItem{}, nil
}
func (m *mockAdminService) TakeModerationAction(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ *ModerationActionInput) error {
	return nil
}
func (m *mockAdminService) ListMethodologies(_ context.Context, _ *shared.AuthContext) ([]MethodologyConfig, error) {
	return []MethodologyConfig{}, nil
}
func (m *mockAdminService) UpdateMethodologyConfig(_ context.Context, _ *shared.AuthContext, _ string, _ *UpdateMethodologyInput) (*MethodologyConfig, error) {
	return &MethodologyConfig{}, nil
}
func (m *mockAdminService) GetPendingDeletions(_ context.Context, _ *shared.AuthContext, _ *shared.PaginationParams) (*shared.PaginatedResponse[DeletionSummary], error) {
	return &shared.PaginatedResponse[DeletionSummary]{Data: []DeletionSummary{}}, nil
}
func (m *mockAdminService) GetRecoveryRequests(_ context.Context, _ *shared.AuthContext, _ *shared.PaginationParams) (*shared.PaginatedResponse[RecoverySummary], error) {
	return &shared.PaginatedResponse[RecoverySummary]{Data: []RecoverySummary{}}, nil
}
func (m *mockAdminService) ResolveRecoveryRequest(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ bool) error {
	return nil
}
func (m *mockAdminService) GetSystemHealth(ctx context.Context, auth *shared.AuthContext) (*SystemHealthResponse, error) {
	if m.getSystemHealthFn != nil {
		return m.getSystemHealthFn(ctx, auth)
	}
	return &SystemHealthResponse{}, nil
}
func (m *mockAdminService) GetJobStatus(_ context.Context, _ *shared.AuthContext) (*JobStatusResponse, error) {
	return &JobStatusResponse{}, nil
}
func (m *mockAdminService) GetDeadLetterJobs(_ context.Context, _ *shared.AuthContext, _ *shared.PaginationParams) (*shared.PaginatedResponse[DeadLetterJob], error) {
	return &shared.PaginatedResponse[DeadLetterJob]{Data: []DeadLetterJob{}}, nil
}
func (m *mockAdminService) RetryDeadLetterJob(_ context.Context, _ *shared.AuthContext, _ string) error {
	return nil
}
func (m *mockAdminService) SearchAuditLog(_ context.Context, _ *shared.AuthContext, _ *AuditLogQuery, _ *shared.PaginationParams) (*shared.PaginatedResponse[AuditLogEntry], error) {
	return &shared.PaginatedResponse[AuditLogEntry]{Data: []AuditLogEntry{}}, nil
}
func (m *mockAdminService) LogAction(_ context.Context, _ *shared.AuthContext, _ *AdminAction) error {
	return nil
}

// Compile-time check.
var _ AdminService = (*mockAdminService)(nil)

// ─── Test Helpers ─────────────────────────────────────────────────────────────

func setupAdminHandlerTest(svc AdminService) (*echo.Echo, *Handler) {
	e := echo.New()
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e, NewHandler(svc)
}

func setAdminTestAuth(c echo.Context) {
	shared.SetAuthContext(c, &shared.AuthContext{
		ParentID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		FamilyID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		IsPlatformAdmin: true,
	})
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestHandler_SearchUsers_200(t *testing.T) {
	svc := &mockAdminService{
		searchUsersFn: func(_ context.Context, _ *shared.AuthContext, _ *UserSearchQuery, _ *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error) {
			return &shared.PaginatedResponse[AdminUserSummary]{Data: []AdminUserSummary{}}, nil
		},
	}
	e, h := setupAdminHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users?q=test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminTestAuth(c)

	if err := h.searchUsers(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_SearchUsers_NotAdmin_403(t *testing.T) {
	e, h := setupAdminHandlerTest(&mockAdminService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Set non-admin auth
	shared.SetAuthContext(c, &shared.AuthContext{
		ParentID:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		FamilyID:        uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		IsPlatformAdmin: false,
	})

	if err := h.searchUsers(c); err == nil {
		t.Fatal("expected error for non-admin")
	}
}

func TestHandler_SearchUsers_MissingAuth_Errors(t *testing.T) {
	e, h := setupAdminHandlerTest(&mockAdminService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// no auth

	if err := h.searchUsers(c); err == nil {
		t.Fatal("expected error for missing auth")
	}
}

func TestHandler_ListFlags_200(t *testing.T) {
	svc := &mockAdminService{
		listFlagsFn: func(_ context.Context, _ *shared.AuthContext) ([]FeatureFlag, error) {
			return []FeatureFlag{}, nil
		},
	}
	e, h := setupAdminHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/flags", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminTestAuth(c)

	if err := h.listFlags(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_GetSystemHealth_200(t *testing.T) {
	svc := &mockAdminService{
		getSystemHealthFn: func(_ context.Context, _ *shared.AuthContext) (*SystemHealthResponse, error) {
			return &SystemHealthResponse{}, nil
		},
	}
	e, h := setupAdminHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/admin/system/health", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminTestAuth(c)

	if err := h.getSystemHealth(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}
