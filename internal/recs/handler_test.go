package recs

// Handler tests for the recommendations domain. [13-recs §5]

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

type echoValidator struct{ v *validator.Validate }

func (cv *echoValidator) Validate(i any) error { return cv.v.Struct(i) }

// ─── Mock RecsService ─────────────────────────────────────────────────────────

type mockRecsService struct {
	getRecommendationsFn func(ctx context.Context, scope *shared.FamilyScope, params RecommendationListParams) (*RecommendationListResponse, error)
	getPreferencesFn     func(ctx context.Context, scope *shared.FamilyScope) (*RecommendationPreferencesResponse, error)
	dismissFn            func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error
}

func (m *mockRecsService) GetRecommendations(ctx context.Context, scope *shared.FamilyScope, params RecommendationListParams) (*RecommendationListResponse, error) {
	if m.getRecommendationsFn != nil {
		return m.getRecommendationsFn(ctx, scope, params)
	}
	return &RecommendationListResponse{Recommendations: []RecommendationResponse{}}, nil
}
func (m *mockRecsService) GetStudentRecommendations(ctx context.Context, scope *shared.FamilyScope, params StudentRecommendationParams) (*RecommendationListResponse, error) {
	return &RecommendationListResponse{Recommendations: []RecommendationResponse{}}, nil
}
func (m *mockRecsService) GetPreferences(ctx context.Context, scope *shared.FamilyScope) (*RecommendationPreferencesResponse, error) {
	if m.getPreferencesFn != nil {
		return m.getPreferencesFn(ctx, scope)
	}
	return &RecommendationPreferencesResponse{}, nil
}
func (m *mockRecsService) RecordSignal(_ context.Context, _ RecordSignalCommand) error { return nil }
func (m *mockRecsService) RegisterListing(_ context.Context, _ RegisterListingCommand) error {
	return nil
}
func (m *mockRecsService) DismissRecommendation(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error {
	if m.dismissFn != nil {
		return m.dismissFn(ctx, scope, id)
	}
	return nil
}
func (m *mockRecsService) BlockRecommendation(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockRecsService) UndoFeedback(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
	return nil
}
func (m *mockRecsService) UpdatePreferences(_ context.Context, _ *shared.FamilyScope, _ UpdatePreferencesCommand) (*RecommendationPreferencesResponse, error) {
	return &RecommendationPreferencesResponse{}, nil
}
func (m *mockRecsService) HandleFamilyDeletion(_ context.Context, _ shared.FamilyID) error {
	return nil
}
func (m *mockRecsService) InvalidateMethodologyCache(_ context.Context) error { return nil }

// Compile-time check.
var _ RecsService = (*mockRecsService)(nil)

// ─── Test Helpers ─────────────────────────────────────────────────────────────

func setupRecsHandlerTest(svc RecsService) (*echo.Echo, *Handler) {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e, NewHandler(svc)
}

func setRecsTestAuth(c echo.Context) {
	shared.SetAuthContext(c, &shared.AuthContext{
		ParentID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		FamilyID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		SubscriptionTier: shared.SubscriptionTierPremium, // recs require premium
	})
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestHandler_GetRecommendations_200(t *testing.T) {
	svc := &mockRecsService{
		getRecommendationsFn: func(_ context.Context, _ *shared.FamilyScope, _ RecommendationListParams) (*RecommendationListResponse, error) {
			return &RecommendationListResponse{Recommendations: []RecommendationResponse{}}, nil
		},
	}
	e, h := setupRecsHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/recommendations", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setRecsTestAuth(c)

	if err := h.getRecommendations(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_GetRecommendations_MissingAuth_Errors(t *testing.T) {
	e, h := setupRecsHandlerTest(&mockRecsService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/recommendations", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// no auth

	if err := h.getRecommendations(c); err == nil {
		t.Fatal("expected error for missing auth")
	}
}

func TestHandler_GetPreferences_200(t *testing.T) {
	svc := &mockRecsService{
		getPreferencesFn: func(_ context.Context, _ *shared.FamilyScope) (*RecommendationPreferencesResponse, error) {
			return &RecommendationPreferencesResponse{}, nil
		},
	}
	e, h := setupRecsHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/recommendations/preferences", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setRecsTestAuth(c)

	if err := h.getPreferences(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_DismissRecommendation_200(t *testing.T) {
	dismissed := false
	svc := &mockRecsService{
		dismissFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
			dismissed = true
			return nil
		},
	}
	e, h := setupRecsHandlerTest(svc)
	recID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/v1/recommendations/"+recID.String()+"/dismiss", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(recID.String())
	setRecsTestAuth(c)

	if err := h.dismissRecommendation(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	if !dismissed {
		t.Error("expected DismissRecommendation to be called")
	}
}

func TestHandler_DismissRecommendation_InvalidID_400(t *testing.T) {
	e, h := setupRecsHandlerTest(&mockRecsService{})
	req := httptest.NewRequest(http.MethodDelete, "/v1/recommendations/not-a-uuid/dismiss", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")
	setRecsTestAuth(c)

	err := h.dismissRecommendation(c)
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}
