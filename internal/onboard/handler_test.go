package onboard

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

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e
}

func setupOnboardRoutes(e *echo.Echo, svc OnboardingService) {
	auth := e.Group("/v1")
	auth.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			shared.SetAuthContext(c, &shared.AuthContext{
				ParentID:           uuid.Must(uuid.NewV7()),
				FamilyID:           uuid.Must(uuid.NewV7()),
				CoppaConsentStatus: "consented",
			})
			return next(c)
		}
	})
	NewHandler(svc).Register(auth)
}

// ─── Mock OnboardingService ─────────────────────────────────────────────────

type mockOnboardingService struct {
	getProgressFn         func(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error)
	updateFamilyProfileFn func(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyProfileCommand) (*WizardProgressResponse, error)
	addChildFn            func(ctx context.Context, scope *shared.FamilyScope, cmd AddChildCommand) (*WizardProgressResponse, error)
	removeChildFn         func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error
	selectMethodologyFn   func(ctx context.Context, scope *shared.FamilyScope, cmd SelectMethodologyCommand) (*WizardProgressResponse, error)
	importQuizFn          func(ctx context.Context, scope *shared.FamilyScope, cmd ImportQuizCommand) (*QuizImportResponse, error)
	getRoadmapFn          func(ctx context.Context, scope *shared.FamilyScope) (*RoadmapResponse, error)
	getRecommendationsFn  func(ctx context.Context, scope *shared.FamilyScope) (*RecommendationsResponse, error)
	getCommunityFn        func(ctx context.Context, scope *shared.FamilyScope) (*CommunityResponse, error)
	completeWizardFn      func(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error)
	skipWizardFn          func(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error)
}

func (m *mockOnboardingService) GetProgress(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error) {
	return m.getProgressFn(ctx, scope)
}
func (m *mockOnboardingService) UpdateFamilyProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyProfileCommand) (*WizardProgressResponse, error) {
	return m.updateFamilyProfileFn(ctx, scope, cmd)
}
func (m *mockOnboardingService) AddChild(ctx context.Context, scope *shared.FamilyScope, cmd AddChildCommand) (*WizardProgressResponse, error) {
	return m.addChildFn(ctx, scope, cmd)
}
func (m *mockOnboardingService) RemoveChild(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error {
	return m.removeChildFn(ctx, scope, studentID)
}
func (m *mockOnboardingService) SelectMethodology(ctx context.Context, scope *shared.FamilyScope, cmd SelectMethodologyCommand) (*WizardProgressResponse, error) {
	return m.selectMethodologyFn(ctx, scope, cmd)
}
func (m *mockOnboardingService) ImportQuiz(ctx context.Context, scope *shared.FamilyScope, cmd ImportQuizCommand) (*QuizImportResponse, error) {
	return m.importQuizFn(ctx, scope, cmd)
}
func (m *mockOnboardingService) GetRoadmap(ctx context.Context, scope *shared.FamilyScope) (*RoadmapResponse, error) {
	return m.getRoadmapFn(ctx, scope)
}
func (m *mockOnboardingService) GetRecommendations(ctx context.Context, scope *shared.FamilyScope) (*RecommendationsResponse, error) {
	return m.getRecommendationsFn(ctx, scope)
}
func (m *mockOnboardingService) GetCommunity(ctx context.Context, scope *shared.FamilyScope) (*CommunityResponse, error) {
	return m.getCommunityFn(ctx, scope)
}
func (m *mockOnboardingService) CompleteWizard(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error) {
	return m.completeWizardFn(ctx, scope)
}
func (m *mockOnboardingService) SkipWizard(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error) {
	return m.skipWizardFn(ctx, scope)
}
func (m *mockOnboardingService) InitializeWizard(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockOnboardingService) HandleMethodologyChanged(_ context.Context, _ uuid.UUID, _ string, _ []string) error {
	return nil
}

// ─── GET /v1/onboarding/progress ─────────────────────────────────────────────

func TestGetProgress_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		getProgressFn: func(_ context.Context, _ *shared.FamilyScope) (*WizardProgressResponse, error) {
			return &WizardProgressResponse{
				Status:         StatusInProgress,
				CurrentStep:    StepFamilyProfile,
				CompletedSteps: []WizardStep{},
			}, nil
		},
	}
	setupOnboardRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/onboarding/progress", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetProgress_404(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		getProgressFn: func(_ context.Context, _ *shared.FamilyScope) (*WizardProgressResponse, error) {
			return nil, &OnboardError{Err: ErrWizardNotFound}
		},
	}
	setupOnboardRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/onboarding/progress", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── PATCH /v1/onboarding/family-profile ─────────────────────────────────────

func TestUpdateFamilyProfile_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		updateFamilyProfileFn: func(_ context.Context, _ *shared.FamilyScope, _ UpdateFamilyProfileCommand) (*WizardProgressResponse, error) {
			return &WizardProgressResponse{Status: StatusInProgress, CompletedSteps: []WizardStep{StepFamilyProfile}}, nil
		},
	}
	setupOnboardRoutes(e, svc)

	body := `{"display_name":"Smith Family"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/onboarding/family-profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/onboarding/children ────────────────────────────────────────────

func TestAddChild_201(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		addChildFn: func(_ context.Context, _ *shared.FamilyScope, _ AddChildCommand) (*WizardProgressResponse, error) {
			return &WizardProgressResponse{Status: StatusInProgress, CompletedSteps: []WizardStep{StepChildren}}, nil
		},
	}
	setupOnboardRoutes(e, svc)

	body := `{"display_name":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/children", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAddChild_403_NoCoppaConsent(t *testing.T) {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler

	// Override auth middleware to set COPPA consent to "registered" (not consented)
	auth := e.Group("/v1")
	auth.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			shared.SetAuthContext(c, &shared.AuthContext{
				ParentID:           uuid.Must(uuid.NewV7()),
				FamilyID:           uuid.Must(uuid.NewV7()),
				CoppaConsentStatus: "registered",
			})
			return next(c)
		}
	})
	NewHandler(&mockOnboardingService{}).Register(auth)

	body := `{"display_name":"Alice"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/children", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── DELETE /v1/onboarding/children/:id ──────────────────────────────────────

func TestRemoveChild_204(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		removeChildFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
			return nil
		},
	}
	setupOnboardRoutes(e, svc)

	studentID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodDelete, "/v1/onboarding/children/"+studentID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── PATCH /v1/onboarding/methodology ────────────────────────────────────────

func TestSelectMethodology_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		selectMethodologyFn: func(_ context.Context, _ *shared.FamilyScope, _ SelectMethodologyCommand) (*WizardProgressResponse, error) {
			return &WizardProgressResponse{
				Status:         StatusInProgress,
				CompletedSteps: []WizardStep{StepFamilyProfile, StepMethodology},
			}, nil
		},
	}
	setupOnboardRoutes(e, svc)

	body := `{"primary_methodology_slug":"charlotte-mason","methodology_path":"exploration"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/onboarding/methodology", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSelectMethodology_422_Validation(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		selectMethodologyFn: func(_ context.Context, _ *shared.FamilyScope, _ SelectMethodologyCommand) (*WizardProgressResponse, error) {
			return &WizardProgressResponse{CompletedSteps: []WizardStep{}}, nil
		},
	}
	setupOnboardRoutes(e, svc)

	// Missing required fields
	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/onboarding/methodology", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/onboarding/methodology/import-quiz ─────────────────────────────

func TestImportQuiz_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		importQuizFn: func(_ context.Context, _ *shared.FamilyScope, _ ImportQuizCommand) (*QuizImportResponse, error) {
			return &QuizImportResponse{ShareID: "abc123", Recommendations: []OnboardQuizRecommendation{}}, nil
		},
	}
	setupOnboardRoutes(e, svc)

	body := `{"share_id":"abc123"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/methodology/import-quiz", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/onboarding/complete ────────────────────────────────────────────

func TestCompleteWizard_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		completeWizardFn: func(_ context.Context, _ *shared.FamilyScope) (*WizardProgressResponse, error) {
			return &WizardProgressResponse{Status: StatusCompleted, CompletedSteps: []WizardStep{StepFamilyProfile, StepMethodology}}, nil
		},
	}
	setupOnboardRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/complete", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp WizardProgressResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.Status != StatusCompleted {
		t.Errorf("want status=completed, got %s", resp.Status)
	}
}

func TestCompleteWizard_422_RequiredStepsMissing(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		completeWizardFn: func(_ context.Context, _ *shared.FamilyScope) (*WizardProgressResponse, error) {
			return nil, &OnboardError{Err: ErrRequiredStepsMissing, Step: "methodology"}
		},
	}
	setupOnboardRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/complete", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/onboarding/skip ────────────────────────────────────────────────

func TestSkipWizard_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockOnboardingService{
		skipWizardFn: func(_ context.Context, _ *shared.FamilyScope) (*WizardProgressResponse, error) {
			return &WizardProgressResponse{Status: StatusSkipped, CompletedSteps: []WizardStep{}}, nil
		},
	}
	setupOnboardRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/onboarding/skip", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
