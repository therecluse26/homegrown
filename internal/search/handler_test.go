package search

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

type echoValidator struct{ v *validator.Validate }

func (cv *echoValidator) Validate(i any) error { return cv.v.Struct(i) }

// stubRateLimitDeps satisfies the rateLimitDeps interface for handler tests.
type stubRateLimitDeps struct{}

func (stubRateLimitDeps) GetCache() shared.Cache { return nil }

func setupHandlerTest(svc SearchService) (*echo.Echo, *Handler) {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	h := NewHandler(svc, stubRateLimitDeps{})
	return e, h
}

func setAuthContext(c echo.Context) {
	c.Set("auth_context", testAuth())
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 9: Handler Tests (#36–#42)
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandler_Search_ValidParams_200(t *testing.T) {
	mockSvc := newMockSearchService()
	mockSvc.searchFn = func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ *SearchParams) (*SearchResponse, error) {
		return &SearchResponse{Results: []SearchResult{}, TotalCount: 0}, nil
	}
	e, h := setupHandlerTest(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=test+query&scope=marketplace", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c)

	err := h.search(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_Search_MissingQ_422(t *testing.T) {
	mockSvc := newMockSearchService()
	e, h := setupHandlerTest(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?scope=marketplace", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c)

	err := h.search(c)
	if err == nil {
		t.Fatal("expected error for missing q")
	}
	assertHTTPStatus(t, err, http.StatusUnprocessableEntity)
}

func TestHandler_Search_QueryTooShort_422(t *testing.T) {
	mockSvc := newMockSearchService()
	mockSvc.searchFn = func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ *SearchParams) (*SearchResponse, error) {
		return nil, &SearchError{Err: ErrQueryTooShort}
	}
	e, h := setupHandlerTest(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=a&scope=marketplace", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c)

	err := h.search(c)
	if err == nil {
		t.Fatal("expected error for short query")
	}
	assertHTTPStatus(t, err, http.StatusUnprocessableEntity)
}

func TestHandler_Search_InvalidScope_400(t *testing.T) {
	mockSvc := newMockSearchService()
	mockSvc.searchFn = func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ *SearchParams) (*SearchResponse, error) {
		return nil, &SearchError{Err: ErrInvalidScope}
	}
	e, h := setupHandlerTest(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=test&scope=invalid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c)

	err := h.search(c)
	if err == nil {
		t.Fatal("expected error for invalid scope")
	}
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestHandler_Autocomplete_ValidParams_200(t *testing.T) {
	mockSvc := newMockSearchService()
	mockSvc.autocompleteFn = func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ *AutocompleteParams) (*AutocompleteResponse, error) {
		return &AutocompleteResponse{Suggestions: []AutocompleteSuggestion{}}, nil
	}
	e, h := setupHandlerTest(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/search/autocomplete?q=test&scope=marketplace", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c)

	err := h.autocomplete(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_Autocomplete_MissingQ_400(t *testing.T) {
	mockSvc := newMockSearchService()
	e, h := setupHandlerTest(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/search/autocomplete?scope=marketplace", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c)

	err := h.autocomplete(c)
	if err == nil {
		t.Fatal("expected error for missing q")
	}
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

func TestHandler_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"query too short → 422", &SearchError{Err: ErrQueryTooShort}, http.StatusUnprocessableEntity},
		{"invalid scope → 400", &SearchError{Err: ErrInvalidScope}, http.StatusBadRequest},
		{"invalid sort → 400", &SearchError{Err: ErrInvalidSortForScope}, http.StatusBadRequest},
		{"backend unavailable → 503", &SearchError{Err: ErrBackendUnavailable}, http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapped := mapSearchError(tt.err)
			var appErr *shared.AppError
			if ok := isAppError(mapped, &appErr); !ok {
				t.Fatalf("expected *shared.AppError, got %T", mapped)
			}
			if appErr.StatusCode != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, appErr.StatusCode)
			}
		})
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 10: Suggestions Stub (#47)
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandler_Suggestions_ValidQuery_200(t *testing.T) {
	mockSvc := newMockSearchService()
	mockSvc.autocompleteFn = func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ *AutocompleteParams) (*AutocompleteResponse, error) {
		return &AutocompleteResponse{Suggestions: []AutocompleteSuggestion{}}, nil
	}
	e, h := setupHandlerTest(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/search/suggestions?q=test", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c)

	err := h.suggestions(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandler_Suggestions_MissingQ_400(t *testing.T) {
	mockSvc := newMockSearchService()
	e, h := setupHandlerTest(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/v1/search/suggestions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAuthContext(c)

	err := h.suggestions(c)
	if err == nil {
		t.Fatal("expected error for missing q")
	}
	assertHTTPStatus(t, err, http.StatusBadRequest)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Handler Test Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func assertHTTPStatus(t *testing.T, err error, wantStatus int) {
	t.Helper()
	var appErr *shared.AppError
	if ok := isAppError(err, &appErr); ok {
		if appErr.StatusCode != wantStatus {
			t.Errorf("expected status %d, got %d", wantStatus, appErr.StatusCode)
		}
		return
	}
	// Try echo.HTTPError
	if he, ok := err.(*echo.HTTPError); ok {
		if he.Code != wantStatus {
			t.Errorf("expected status %d, got %d", wantStatus, he.Code)
		}
		return
	}
	t.Errorf("error type %T does not contain HTTP status", err)
}

func isAppError(err error, target **shared.AppError) bool {
	if ae, ok := err.(*shared.AppError); ok {
		*target = ae
		return true
	}
	return false
}

