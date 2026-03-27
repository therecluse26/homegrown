package comply

// Handler tests for the compliance domain. [14-comply §4]

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// ─── Mock ComplianceService ────────────────────────────────────────────────────

type mockComplianceService struct {
	getFamilyConfigFn    func(ctx context.Context, scope shared.FamilyScope) (*FamilyConfigResponse, error)
	listStateConfigsFn   func(ctx context.Context) ([]StateConfigSummaryResponse, error)
	getDashboardFn       func(ctx context.Context, scope shared.FamilyScope) (*ComplianceDashboardResponse, error)
	listSchedulesFn      func(ctx context.Context, scope shared.FamilyScope) ([]ScheduleResponse, error)
}

func (m *mockComplianceService) GetFamilyConfig(ctx context.Context, scope shared.FamilyScope) (*FamilyConfigResponse, error) {
	if m.getFamilyConfigFn != nil {
		return m.getFamilyConfigFn(ctx, scope)
	}
	return &FamilyConfigResponse{}, nil
}
func (m *mockComplianceService) ListStateConfigs(ctx context.Context) ([]StateConfigSummaryResponse, error) {
	if m.listStateConfigsFn != nil {
		return m.listStateConfigsFn(ctx)
	}
	return []StateConfigSummaryResponse{}, nil
}
func (m *mockComplianceService) GetStateConfig(_ context.Context, _ string) (*StateConfigResponse, error) {
	return &StateConfigResponse{}, nil
}
func (m *mockComplianceService) UpsertFamilyConfig(_ context.Context, _ UpsertFamilyConfigCommand, _ shared.FamilyScope) (*FamilyConfigResponse, error) {
	return &FamilyConfigResponse{}, nil
}
func (m *mockComplianceService) CreateSchedule(_ context.Context, _ CreateScheduleCommand, _ shared.FamilyScope) (*ScheduleResponse, error) {
	return &ScheduleResponse{}, nil
}
func (m *mockComplianceService) UpdateSchedule(_ context.Context, _ uuid.UUID, _ UpdateScheduleCommand, _ shared.FamilyScope) (*ScheduleResponse, error) {
	return &ScheduleResponse{}, nil
}
func (m *mockComplianceService) DeleteSchedule(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) error {
	return nil
}
func (m *mockComplianceService) ListSchedules(ctx context.Context, scope shared.FamilyScope) ([]ScheduleResponse, error) {
	if m.listSchedulesFn != nil {
		return m.listSchedulesFn(ctx, scope)
	}
	return []ScheduleResponse{}, nil
}
func (m *mockComplianceService) RecordAttendance(_ context.Context, _ uuid.UUID, _ RecordAttendanceCommand, _ shared.FamilyScope) (*AttendanceResponse, error) {
	return &AttendanceResponse{}, nil
}
func (m *mockComplianceService) BulkRecordAttendance(_ context.Context, _ uuid.UUID, _ BulkRecordAttendanceCommand, _ shared.FamilyScope) ([]AttendanceResponse, error) {
	return []AttendanceResponse{}, nil
}
func (m *mockComplianceService) UpdateAttendance(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ UpdateAttendanceCommand, _ shared.FamilyScope) (*AttendanceResponse, error) {
	return &AttendanceResponse{}, nil
}
func (m *mockComplianceService) DeleteAttendance(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) error {
	return nil
}
func (m *mockComplianceService) ListAttendance(_ context.Context, _ uuid.UUID, _ AttendanceListParams, _ shared.FamilyScope) (*AttendanceListResponse, error) {
	return &AttendanceListResponse{}, nil
}
func (m *mockComplianceService) GetAttendanceSummary(_ context.Context, _ uuid.UUID, _ AttendanceSummaryParams, _ shared.FamilyScope) (*AttendanceSummaryResponse, error) {
	return &AttendanceSummaryResponse{}, nil
}
func (m *mockComplianceService) CreateAssessment(_ context.Context, _ uuid.UUID, _ CreateAssessmentCommand, _ shared.FamilyScope) (*AssessmentResponse, error) {
	return &AssessmentResponse{}, nil
}
func (m *mockComplianceService) UpdateAssessment(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ UpdateAssessmentCommand, _ shared.FamilyScope) (*AssessmentResponse, error) {
	return &AssessmentResponse{}, nil
}
func (m *mockComplianceService) DeleteAssessment(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) error {
	return nil
}
func (m *mockComplianceService) ListAssessments(_ context.Context, _ uuid.UUID, _ AssessmentListParams, _ shared.FamilyScope) (*AssessmentListResponse, error) {
	return &AssessmentListResponse{}, nil
}
func (m *mockComplianceService) CreateTestScore(_ context.Context, _ uuid.UUID, _ CreateTestScoreCommand, _ shared.FamilyScope) (*TestScoreResponse, error) {
	return &TestScoreResponse{}, nil
}
func (m *mockComplianceService) UpdateTestScore(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ UpdateTestScoreCommand, _ shared.FamilyScope) (*TestScoreResponse, error) {
	return &TestScoreResponse{}, nil
}
func (m *mockComplianceService) DeleteTestScore(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) error {
	return nil
}
func (m *mockComplianceService) ListTestScores(_ context.Context, _ uuid.UUID, _ TestListParams, _ shared.FamilyScope) (*TestListResponse, error) {
	return &TestListResponse{}, nil
}
func (m *mockComplianceService) CreatePortfolio(_ context.Context, _ uuid.UUID, _ CreatePortfolioCommand, _ shared.FamilyScope) (*PortfolioResponse, error) {
	return &PortfolioResponse{}, nil
}
func (m *mockComplianceService) AddPortfolioItems(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ AddPortfolioItemsCommand, _ shared.FamilyScope) ([]PortfolioItemResponse, error) {
	return []PortfolioItemResponse{}, nil
}
func (m *mockComplianceService) GeneratePortfolio(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) (*PortfolioResponse, error) {
	return &PortfolioResponse{}, nil
}
func (m *mockComplianceService) GetPortfolio(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) (*PortfolioResponse, error) {
	return &PortfolioResponse{}, nil
}
func (m *mockComplianceService) ListPortfolios(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) ([]PortfolioSummaryResponse, error) {
	return []PortfolioSummaryResponse{}, nil
}
func (m *mockComplianceService) GetPortfolioDownloadURL(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) (string, error) {
	return "https://example.com/portfolio.pdf", nil
}
func (m *mockComplianceService) GetDashboard(ctx context.Context, scope shared.FamilyScope) (*ComplianceDashboardResponse, error) {
	if m.getDashboardFn != nil {
		return m.getDashboardFn(ctx, scope)
	}
	return &ComplianceDashboardResponse{}, nil
}
func (m *mockComplianceService) CreateTranscript(_ context.Context, _ uuid.UUID, _ CreateTranscriptCommand, _ shared.FamilyScope) (*TranscriptResponse, error) {
	return &TranscriptResponse{}, nil
}
func (m *mockComplianceService) GenerateTranscript(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) (*TranscriptResponse, error) {
	return &TranscriptResponse{}, nil
}
func (m *mockComplianceService) DeleteTranscript(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) error {
	return nil
}
func (m *mockComplianceService) GetTranscript(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) (*TranscriptResponse, error) {
	return &TranscriptResponse{}, nil
}
func (m *mockComplianceService) ListTranscripts(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) ([]TranscriptSummaryResponse, error) {
	return []TranscriptSummaryResponse{}, nil
}
func (m *mockComplianceService) GetTranscriptDownloadURL(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) (string, error) {
	return "https://example.com/transcript.pdf", nil
}
func (m *mockComplianceService) CreateCourse(_ context.Context, _ uuid.UUID, _ CreateCourseCommand, _ shared.FamilyScope) (*CourseResponse, error) {
	return &CourseResponse{}, nil
}
func (m *mockComplianceService) UpdateCourse(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ UpdateCourseCommand, _ shared.FamilyScope) (*CourseResponse, error) {
	return &CourseResponse{}, nil
}
func (m *mockComplianceService) DeleteCourse(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ shared.FamilyScope) error {
	return nil
}
func (m *mockComplianceService) ListCourses(_ context.Context, _ uuid.UUID, _ CourseListParams, _ shared.FamilyScope) (*CourseListResponse, error) {
	return &CourseListResponse{}, nil
}
func (m *mockComplianceService) CalculateGPA(_ context.Context, _ uuid.UUID, _ GpaParams, _ shared.FamilyScope) (*GpaResponse, error) {
	return &GpaResponse{}, nil
}
func (m *mockComplianceService) CalculateGPAWhatIf(_ context.Context, _ uuid.UUID, _ GpaWhatIfParams, _ shared.FamilyScope) (*GpaResponse, error) {
	return &GpaResponse{}, nil
}
func (m *mockComplianceService) GetGPAHistory(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) ([]GpaTermResponse, error) {
	return []GpaTermResponse{}, nil
}
func (m *mockComplianceService) HandleActivityLogged(_ context.Context, _ *ActivityLoggedEvent) error {
	return nil
}
func (m *mockComplianceService) HandleStudentDeleted(_ context.Context, _ *StudentDeletedEvent) error {
	return nil
}
func (m *mockComplianceService) HandleFamilyDeletionScheduled(_ context.Context, _ *FamilyDeletionScheduledEvent) error {
	return nil
}
func (m *mockComplianceService) HandleSubscriptionCancelled(_ context.Context, _ *SubscriptionCancelledEvent) error {
	return nil
}

// Compile-time check.
var _ ComplianceService = (*mockComplianceService)(nil)

// ─── Test Helpers ──────────────────────────────────────────────────────────────

func setupComplyHandlerTest(svc ComplianceService) (*echo.Echo, *Handler) {
	e := echo.New()
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e, NewHandler(svc)
}

func setComplyTestAuth(c echo.Context) {
	shared.SetAuthContext(c, &shared.AuthContext{
		ParentID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		FamilyID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		SubscriptionTier: shared.SubscriptionTierPremium, // comply requires premium
	})
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestHandler_GetFamilyConfig_200(t *testing.T) {
	svc := &mockComplianceService{
		getFamilyConfigFn: func(_ context.Context, _ shared.FamilyScope) (*FamilyConfigResponse, error) {
			return &FamilyConfigResponse{}, nil
		},
	}
	e, h := setupComplyHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/config", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setComplyTestAuth(c)

	if err := h.getFamilyConfig(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_GetFamilyConfig_FreeTier_402(t *testing.T) {
	e, h := setupComplyHandlerTest(&mockComplianceService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/config", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Set free tier — comply requires premium
	shared.SetAuthContext(c, &shared.AuthContext{
		ParentID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		FamilyID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		SubscriptionTier: shared.SubscriptionTierFree,
	})

	if err := h.getFamilyConfig(c); err == nil {
		t.Fatal("expected error for free tier")
	}
}

func TestHandler_GetFamilyConfig_MissingAuth_Errors(t *testing.T) {
	e, h := setupComplyHandlerTest(&mockComplianceService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/config", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// no auth

	if err := h.getFamilyConfig(c); err == nil {
		t.Fatal("expected error for missing auth")
	}
}

func TestHandler_ListStateConfigs_200(t *testing.T) {
	svc := &mockComplianceService{
		listStateConfigsFn: func(_ context.Context) ([]StateConfigSummaryResponse, error) {
			return []StateConfigSummaryResponse{}, nil
		},
	}
	e, h := setupComplyHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/state-requirements", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setComplyTestAuth(c)

	if err := h.listStateConfigs(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_GetDashboard_200(t *testing.T) {
	svc := &mockComplianceService{
		getDashboardFn: func(_ context.Context, _ shared.FamilyScope) (*ComplianceDashboardResponse, error) {
			return &ComplianceDashboardResponse{}, nil
		},
	}
	e, h := setupComplyHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/dashboard", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setComplyTestAuth(c)

	if err := h.getDashboard(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_ListSchedules_200(t *testing.T) {
	svc := &mockComplianceService{
		listSchedulesFn: func(_ context.Context, _ shared.FamilyScope) ([]ScheduleResponse, error) {
			return []ScheduleResponse{
				{ID: uuid.New(), Name: "Test Schedule"},
			}, nil
		},
	}
	e, h := setupComplyHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/compliance/schedules", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setComplyTestAuth(c)

	if err := h.listSchedules(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}
