package method

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/method/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Ensure mockMethodologyService satisfies MethodologyService at compile time.
var _ MethodologyService = (*mockMethodologyService)(nil)

// ─── Test: GET /v1/methodologies ─────────────────────────────────────────────

func TestListMethodologies_Returns200(t *testing.T) {
	svc := &mockMethodologyService{
		listMethodologies: func(_ context.Context) ([]MethodologySummaryResponse, error) {
			return []MethodologySummaryResponse{
				{Slug: "charlotte-mason", DisplayName: "Charlotte Mason", ShortDesc: "Living books"},
				{Slug: "traditional", DisplayName: "Traditional", ShortDesc: "Textbook-based"},
			}, nil
		},
	}
	h := NewHandler(svc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/methodologies", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.listMethodologies(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	var resp []MethodologySummaryResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 2 {
		t.Errorf("want 2 methodologies, got %d", len(resp))
	}
}

// ─── Test: GET /v1/methodologies/:slug → 404 ────────────────────────────────

func TestGetMethodology_NotFound_Returns404(t *testing.T) {
	svc := &mockMethodologyService{
		getMethodology: func(_ context.Context, slug string) (*MethodologyDetailResponse, error) {
			return nil, &domain.MethodError{Err: domain.ErrMethodologyNotFound, Slug: slug}
		},
	}
	h := NewHandler(svc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/methodologies/nonexistent", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("slug")
	c.SetParamValues("nonexistent")

	err := h.getMethodology(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *shared.AppError, got %T: %v", err, err)
	}
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("want 404, got %d", appErr.StatusCode)
	}
	if appErr.Code != "methodology_not_found" {
		t.Errorf("want code 'methodology_not_found', got %q", appErr.Code)
	}
}

// ─── Test: PATCH /v1/families/methodology → 422 invalid slugs ──────────────

func TestUpdateMethodology_InvalidSlugs_Returns422(t *testing.T) {
	svc := &mockMethodologyService{
		updateFamilyMethodology: func(_ context.Context, _ *shared.FamilyScope, _ UpdateMethodologyCommand) (*MethodologySelectionResponse, error) {
			return nil, &domain.MethodError{Err: domain.ErrInvalidMethodologyIDs, Slugs: []string{"bad-slug"}}
		},
	}
	h := NewHandler(svc)
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}

	body := `{"primary_methodology_slug":"bad-slug"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/families/methodology", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	shared.SetAuthContext(c, &shared.AuthContext{FamilyID: uuid.New()})

	err := h.updateFamilyMethodology(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *shared.AppError, got %T: %v", err, err)
	}
	if appErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", appErr.StatusCode)
	}
}

// ─── Test: PATCH /v1/families/methodology → 422 primary in secondary ────────

func TestUpdateMethodology_PrimaryInSecondary_Returns422(t *testing.T) {
	svc := &mockMethodologyService{
		updateFamilyMethodology: func(_ context.Context, _ *shared.FamilyScope, _ UpdateMethodologyCommand) (*MethodologySelectionResponse, error) {
			return nil, &domain.MethodError{Err: domain.ErrPrimaryInSecondary, Slug: "charlotte-mason"}
		},
	}
	h := NewHandler(svc)
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}

	body := `{"primary_methodology_slug":"charlotte-mason","secondary_methodology_slugs":["charlotte-mason"]}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/families/methodology", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	shared.SetAuthContext(c, &shared.AuthContext{FamilyID: uuid.New()})

	err := h.updateFamilyMethodology(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *shared.AppError, got %T: %v", err, err)
	}
	if appErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d", appErr.StatusCode)
	}
	if appErr.Code != "primary_in_secondary" {
		t.Errorf("want code 'primary_in_secondary', got %q", appErr.Code)
	}
}

// ─── Test: GET /v1/families/students/:id/tools → invalid UUID → 400 ────────

func TestGetStudentTools_InvalidUUID_Returns400(t *testing.T) {
	h := NewHandler(&mockMethodologyService{})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/families/students/not-a-uuid/tools", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")

	shared.SetAuthContext(c, &shared.AuthContext{FamilyID: uuid.New()})

	err := h.getStudentTools(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *shared.AppError, got %T: %v", err, err)
	}
	if appErr.StatusCode != http.StatusBadRequest {
		t.Errorf("want 400, got %d", appErr.StatusCode)
	}
}

// ─── Test: GET /v1/methodologies/:slug/tools → 200 ─────────────────────────

func TestGetMethodologyTools_Returns200(t *testing.T) {
	svc := &mockMethodologyService{
		getMethodologyTools: func(_ context.Context, slug string) ([]ActiveToolResponse, error) {
			return []ActiveToolResponse{
				{Slug: "activities", DisplayName: "Activities", Tier: "free"},
			}, nil
		},
	}
	h := NewHandler(svc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/methodologies/charlotte-mason/tools", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("slug")
	c.SetParamValues("charlotte-mason")

	err := h.getMethodologyTools(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	var resp []ActiveToolResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 1 {
		t.Errorf("want 1 tool, got %d", len(resp))
	}
	if resp[0].Slug != "activities" {
		t.Errorf("want slug 'activities', got %q", resp[0].Slug)
	}
}

// ─── Test: GET /v1/families/tools → 200 ─────────────────────────────────────

func TestGetFamilyTools_Returns200(t *testing.T) {
	svc := &mockMethodologyService{
		resolveFamilyTools: func(_ context.Context, _ *shared.FamilyScope) ([]ActiveToolResponse, error) {
			return []ActiveToolResponse{
				{Slug: "activities", DisplayName: "Activities", Tier: "free"},
				{Slug: "reading-lists", DisplayName: "Reading Lists", Tier: "free"},
			}, nil
		},
	}
	h := NewHandler(svc)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/v1/families/tools", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	shared.SetAuthContext(c, &shared.AuthContext{FamilyID: uuid.New()})

	err := h.getFamilyTools(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	var resp []ActiveToolResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 2 {
		t.Errorf("want 2 tools, got %d", len(resp))
	}
}

// ─── Test: PATCH /v1/families/methodology → 200 success ─────────────────────

func TestUpdateFamilyMethodology_Success_Returns200(t *testing.T) {
	svc := &mockMethodologyService{
		updateFamilyMethodology: func(_ context.Context, _ *shared.FamilyScope, cmd UpdateMethodologyCommand) (*MethodologySelectionResponse, error) {
			return &MethodologySelectionResponse{
				Primary: MethodologySummaryResponse{
					Slug:        cmd.PrimaryMethodologySlug,
					DisplayName: "Charlotte Mason",
				},
				Secondary:       []MethodologySummaryResponse{},
				ActiveToolCount: 10,
			}, nil
		},
	}
	h := NewHandler(svc)
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}

	body := `{"primary_methodology_slug":"charlotte-mason"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/families/methodology", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	shared.SetAuthContext(c, &shared.AuthContext{FamilyID: uuid.New()})

	err := h.updateFamilyMethodology(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
	var resp MethodologySelectionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.ActiveToolCount != 10 {
		t.Errorf("want active_tool_count 10, got %d", resp.ActiveToolCount)
	}
	if resp.Primary.Slug != "charlotte-mason" {
		t.Errorf("want primary slug 'charlotte-mason', got %q", resp.Primary.Slug)
	}
}

// ─── Mock ─────────────────────────────────────────────────────────────────────

type mockMethodologyService struct {
	listMethodologies       func(ctx context.Context) ([]MethodologySummaryResponse, error)
	getMethodology          func(ctx context.Context, slug string) (*MethodologyDetailResponse, error)
	getMethodologyTools     func(ctx context.Context, slug string) ([]ActiveToolResponse, error)
	resolveFamilyTools      func(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error)
	updateFamilyMethodology func(ctx context.Context, scope *shared.FamilyScope, cmd UpdateMethodologyCommand) (*MethodologySelectionResponse, error)
}

func (m *mockMethodologyService) ListMethodologies(ctx context.Context) ([]MethodologySummaryResponse, error) {
	if m.listMethodologies != nil {
		return m.listMethodologies(ctx)
	}
	return nil, nil
}
func (m *mockMethodologyService) GetMethodology(ctx context.Context, slug string) (*MethodologyDetailResponse, error) {
	if m.getMethodology != nil {
		return m.getMethodology(ctx, slug)
	}
	return &MethodologyDetailResponse{}, nil
}
func (m *mockMethodologyService) GetMethodologyTools(ctx context.Context, slug string) ([]ActiveToolResponse, error) {
	if m.getMethodologyTools != nil {
		return m.getMethodologyTools(ctx, slug)
	}
	return nil, nil
}
func (m *mockMethodologyService) ValidateMethodologySlug(_ context.Context, _ MethodologyID) (bool, error) {
	return true, nil
}
func (m *mockMethodologyService) GetDefaultMethodologySlug(_ context.Context) (MethodologyID, error) {
	return MethodologyCharlotteMason, nil
}
func (m *mockMethodologyService) ResolveFamilyTools(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error) {
	if m.resolveFamilyTools != nil {
		return m.resolveFamilyTools(ctx, scope)
	}
	return nil, nil
}
func (m *mockMethodologyService) ResolveStudentTools(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) ([]ActiveToolResponse, error) {
	return nil, nil
}
func (m *mockMethodologyService) GetMethodologyContext(_ context.Context, _ *shared.FamilyScope) (*MethodologyContext, error) {
	return nil, nil
}
func (m *mockMethodologyService) UpdateFamilyMethodology(ctx context.Context, scope *shared.FamilyScope, cmd UpdateMethodologyCommand) (*MethodologySelectionResponse, error) {
	if m.updateFamilyMethodology != nil {
		return m.updateFamilyMethodology(ctx, scope, cmd)
	}
	return &MethodologySelectionResponse{}, nil
}
func (m *mockMethodologyService) UpdateStudentMethodology(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ UpdateStudentMethodologyCommand) (*MethodologySelectionResponse, error) {
	return nil, nil
}

// testValidator adapts go-playground/validator to Echo's Validator interface.
type testValidator struct {
	v *validator.Validate
}

func (tv *testValidator) Validate(i interface{}) error {
	return tv.v.Struct(i)
}
