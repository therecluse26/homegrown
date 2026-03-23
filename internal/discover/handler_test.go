package discover

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// ─── Test Infrastructure ──────────────────────────────────────────────────────

// customValidator adapts go-playground/validator to Echo's Validator interface.
type customValidator struct{ v *validator.Validate }

func (cv *customValidator) Validate(i interface{}) error { return cv.v.Struct(i) }

// newTestEcho creates a minimal Echo instance suitable for handler tests.
func newTestEcho() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	e.Validator = &customValidator{v: validator.New()}
	return e
}

// setupDiscoveryRoutes registers the discover routes against an Echo instance.
func setupDiscoveryRoutes(e *echo.Echo, svc DiscoveryService) {
	pub := e.Group("/v1")
	NewHandler(svc).Register(pub)
}

// ─── Mock Service ─────────────────────────────────────────────────────────────

// mockDiscoveryService implements DiscoveryService for handler tests.
type mockDiscoveryService struct {
	getActiveQuizFn       func(ctx context.Context) (*QuizResponse, error)
	submitQuizFn          func(ctx context.Context, cmd SubmitQuizCommand) (*QuizResultResponse, error)
	getQuizResultFn       func(ctx context.Context, shareID string) (*QuizResultResponse, error)
	listStateGuidesFn     func(ctx context.Context) ([]StateGuideSummaryResponse, error)
	getStateGuideFn       func(ctx context.Context, stateCode string) (*StateGuideResponse, error)
	getStateRequirementsFn func(ctx context.Context, stateCode string) (*StateGuideRequirements, error)
}

func (m *mockDiscoveryService) GetActiveQuiz(ctx context.Context) (*QuizResponse, error) {
	return m.getActiveQuizFn(ctx)
}
func (m *mockDiscoveryService) SubmitQuiz(ctx context.Context, cmd SubmitQuizCommand) (*QuizResultResponse, error) {
	return m.submitQuizFn(ctx, cmd)
}
func (m *mockDiscoveryService) GetQuizResult(ctx context.Context, shareID string) (*QuizResultResponse, error) {
	return m.getQuizResultFn(ctx, shareID)
}
func (m *mockDiscoveryService) ListStateGuides(ctx context.Context) ([]StateGuideSummaryResponse, error) {
	return m.listStateGuidesFn(ctx)
}
func (m *mockDiscoveryService) GetStateGuide(ctx context.Context, stateCode string) (*StateGuideResponse, error) {
	return m.getStateGuideFn(ctx, stateCode)
}
func (m *mockDiscoveryService) GetStateRequirements(ctx context.Context, stateCode string) (*StateGuideRequirements, error) {
	if m.getStateRequirementsFn != nil {
		return m.getStateRequirementsFn(ctx, stateCode)
	}
	return nil, &DiscoverError{Err: ErrStateGuideNotFound, StateCode: stateCode}
}

// ─── GET /v1/discovery/quiz ───────────────────────────────────────────────────

func TestGetQuiz_200WhenActive(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		getActiveQuizFn: func(_ context.Context) (*QuizResponse, error) {
			return &QuizResponse{Title: "Find Your Methodology", Questions: []QuizQuestionResponse{}}, nil
		},
	}
	setupDiscoveryRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/discovery/quiz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify quiz_id field name (not "id"). [03-discover §8.1 gap fix]
	var body map[string]json.RawMessage
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, hasQuizID := body["quiz_id"]; !hasQuizID {
		t.Error("want 'quiz_id' field in quiz response, not 'id'")
	}
	if _, hasID := body["id"]; hasID {
		t.Error("must not have 'id' field in quiz response — should be 'quiz_id'")
	}
}

func TestGetQuiz_404WhenNoActiveQuiz(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		getActiveQuizFn: func(_ context.Context) (*QuizResponse, error) {
			return nil, &DiscoverError{Err: ErrNoActiveQuiz}
		},
	}
	setupDiscoveryRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/discovery/quiz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/discovery/quiz/results ─────────────────────────────────────────

func TestSubmitQuiz_201WithValidAnswers(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		submitQuizFn: func(_ context.Context, cmd SubmitQuizCommand) (*QuizResultResponse, error) {
			return &QuizResultResponse{
				ShareID:         "abc12345def6",
				QuizVersion:     1,
				IsClaimed:       false,
				Recommendations: []MethodologyRecommendation{},
			}, nil
		},
	}
	setupDiscoveryRoutes(e, svc)

	body, _ := json.Marshal(SubmitQuizCommand{Answers: map[string]string{"q1": "q1a1"}})
	req := httptest.NewRequest(http.MethodPost, "/v1/discovery/quiz/results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp QuizResultResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.ShareID != "abc12345def6" {
		t.Errorf("want share_id 'abc12345def6', got %q", resp.ShareID)
	}
	// Verify new fields are serialized. [03-discover §8.2 gap fix]
	if resp.QuizVersion != 1 {
		t.Errorf("want quiz_version 1, got %d", resp.QuizVersion)
	}
	if resp.IsClaimed {
		t.Error("want is_claimed=false for a fresh submission")
	}
}

func TestSubmitQuiz_422WithInvalidAnswerID(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		submitQuizFn: func(_ context.Context, cmd SubmitQuizCommand) (*QuizResultResponse, error) {
			return nil, &DiscoverError{Err: ErrInvalidAnswerID, AnswerID: "bad-answer"}
		},
	}
	setupDiscoveryRoutes(e, svc)

	body, _ := json.Marshal(SubmitQuizCommand{Answers: map[string]string{"q1": "bad-answer"}})
	req := httptest.NewRequest(http.MethodPost, "/v1/discovery/quiz/results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSubmitQuiz_400WithMissingAnswers(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		submitQuizFn: func(_ context.Context, cmd SubmitQuizCommand) (*QuizResultResponse, error) {
			return nil, nil
		},
	}
	setupDiscoveryRoutes(e, svc)

	// Missing required `answers` field
	body := []byte(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/discovery/quiz/results", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// `answers` is required — validation should fail
	if rec.Code == http.StatusCreated {
		t.Errorf("want non-201 for missing required answers, got %d", rec.Code)
	}
}

// ─── GET /v1/discovery/quiz/results/:share_id ─────────────────────────────────

func TestGetQuizResult_200WithValidShareID(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		getQuizResultFn: func(_ context.Context, shareID string) (*QuizResultResponse, error) {
			return &QuizResultResponse{
				ShareID:         shareID,
				QuizVersion:     2,
				IsClaimed:       true,
				Recommendations: []MethodologyRecommendation{},
			}, nil
		},
	}
	setupDiscoveryRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/discovery/quiz/results/abc12345def6", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp QuizResultResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.QuizVersion != 2 {
		t.Errorf("want quiz_version 2, got %d", resp.QuizVersion)
	}
	if !resp.IsClaimed {
		t.Error("want is_claimed=true for claimed result")
	}
}

func TestGetQuizResult_404ForMissingShareID(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		getQuizResultFn: func(_ context.Context, shareID string) (*QuizResultResponse, error) {
			return nil, &DiscoverError{Err: ErrQuizResultNotFound}
		},
	}
	setupDiscoveryRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/discovery/quiz/results/notexist", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/discovery/state-guides ──────────────────────────────────────────

func TestListStateGuides_200ReturnsArray(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		listStateGuidesFn: func(_ context.Context) ([]StateGuideSummaryResponse, error) {
			return []StateGuideSummaryResponse{
				{StateCode: "CA", StateName: "California", IsAvailable: true},
				{StateCode: "TX", StateName: "Texas", IsAvailable: false},
			}, nil
		},
	}
	setupDiscoveryRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/discovery/state-guides", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp []StateGuideSummaryResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if len(resp) != 2 {
		t.Errorf("want 2 guides, got %d", len(resp))
	}
}

// ─── GET /v1/discovery/state-guides/:state_code ───────────────────────────────

func TestGetStateGuide_200ForPublishedGuide(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		getStateGuideFn: func(_ context.Context, stateCode string) (*StateGuideResponse, error) {
			return &StateGuideResponse{
				StateCode: stateCode,
				StateName: "California",
				Requirements: StateGuideRequirements{
					NotificationRequired: true,
					RequiredSubjects:     []string{"math"},
					RegulationLevel:      "moderate",
				},
				GuideContent:    "California homeschool guide content.",
				LegalDisclaimer: "This information is for educational purposes only.",
			}, nil
		},
	}
	setupDiscoveryRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/discovery/state-guides/CA", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify new fields are present. [03-discover §15.17 gap fix]
	var resp StateGuideResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.GuideContent == "" {
		t.Error("want non-empty guide_content in state guide response")
	}
	if resp.LegalDisclaimer == "" {
		t.Error("want non-empty legal_disclaimer in state guide response")
	}
	if !resp.Requirements.NotificationRequired {
		t.Error("want notification_required=true")
	}
}

func TestGetStateGuide_404ForDraftGuide(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		getStateGuideFn: func(_ context.Context, stateCode string) (*StateGuideResponse, error) {
			return nil, &DiscoverError{Err: ErrStateGuideNotPublished, StateCode: stateCode}
		},
	}
	setupDiscoveryRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/discovery/state-guides/TX", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404 for draft guide, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetStateGuide_404ForInvalidCode(t *testing.T) {
	e := newTestEcho()
	svc := &mockDiscoveryService{
		getStateGuideFn: func(_ context.Context, stateCode string) (*StateGuideResponse, error) {
			return nil, &DiscoverError{Err: ErrStateGuideNotFound, StateCode: stateCode}
		},
	}
	setupDiscoveryRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/discovery/state-guides/XX", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404 for invalid code, got %d: %s", rec.Code, rec.Body.String())
	}
}
