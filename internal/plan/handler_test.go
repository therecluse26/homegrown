package plan

// Handler tests for the planning domain. [17-planning §4]

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// ─── Mock PlanningService ─────────────────────────────────────────────────────

type mockPlanningService struct {
	getCalendarFn       func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params CalendarQuery) (CalendarResponse, error)
	listScheduleItemsFn func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params ScheduleItemQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[ScheduleItemResponse], error)
	createScheduleItemFn func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, input CreateScheduleItemInput) (uuid.UUID, error)
	deleteScheduleItemFn func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, itemID uuid.UUID) error
}

func (m *mockPlanningService) GetCalendar(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params CalendarQuery) (CalendarResponse, error) {
	if m.getCalendarFn != nil {
		return m.getCalendarFn(ctx, auth, scope, params)
	}
	return CalendarResponse{Days: []CalendarDay{}}, nil
}
func (m *mockPlanningService) GetDayView(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ time.Time, _ *uuid.UUID) (DayViewResponse, error) {
	return DayViewResponse{}, nil
}
func (m *mockPlanningService) CreateScheduleItem(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, input CreateScheduleItemInput) (uuid.UUID, error) {
	if m.createScheduleItemFn != nil {
		return m.createScheduleItemFn(ctx, auth, scope, input)
	}
	return uuid.New(), nil
}
func (m *mockPlanningService) UpdateScheduleItem(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ uuid.UUID, _ UpdateScheduleItemInput) error {
	return nil
}
func (m *mockPlanningService) DeleteScheduleItem(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, itemID uuid.UUID) error {
	if m.deleteScheduleItemFn != nil {
		return m.deleteScheduleItemFn(ctx, auth, scope, itemID)
	}
	return nil
}
func (m *mockPlanningService) CompleteScheduleItem(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ uuid.UUID) error {
	return nil
}
func (m *mockPlanningService) LogAsActivity(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ uuid.UUID, _ LogAsActivityInput) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockPlanningService) ListScheduleItems(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params ScheduleItemQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[ScheduleItemResponse], error) {
	if m.listScheduleItemsFn != nil {
		return m.listScheduleItemsFn(ctx, auth, scope, params, pagination)
	}
	return &shared.PaginatedResponse[ScheduleItemResponse]{Data: []ScheduleItemResponse{}}, nil
}
func (m *mockPlanningService) GetScheduleItem(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ uuid.UUID) (ScheduleItemResponse, error) {
	return ScheduleItemResponse{}, nil
}
func (m *mockPlanningService) GetPrintView(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ time.Time, _ time.Time, _ *uuid.UUID) (string, error) {
	return "<html></html>", nil
}
func (m *mockPlanningService) CreateTemplate(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ CreateTemplateInput) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockPlanningService) ListTemplates(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope) ([]TemplateResponse, error) {
	return []TemplateResponse{}, nil
}
func (m *mockPlanningService) UpdateTemplate(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ uuid.UUID, _ UpdateTemplateInput) error {
	return nil
}
func (m *mockPlanningService) DeleteTemplate(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ uuid.UUID) error {
	return nil
}
func (m *mockPlanningService) ApplyTemplate(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ uuid.UUID, _ ApplyTemplateInput) ([]uuid.UUID, error) {
	return []uuid.UUID{}, nil
}
func (m *mockPlanningService) HandleEventCancelled(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}
func (m *mockPlanningService) HandleActivityLogged(_ context.Context, _, _, _ uuid.UUID) error {
	return nil
}
func (m *mockPlanningService) ExportData(_ context.Context, _ *shared.FamilyScope) ([]byte, error) {
	return []byte("{}"), nil
}
func (m *mockPlanningService) DeleteData(_ context.Context, _ *shared.FamilyScope) error {
	return nil
}
func (m *mockPlanningService) DeleteStudentData(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
	return nil
}

// Compile-time check.
var _ PlanningService = (*mockPlanningService)(nil)

// ─── Test Helpers ─────────────────────────────────────────────────────────────

type echoValidator struct{ v *validator.Validate }

func (cv *echoValidator) Validate(i any) error { return cv.v.Struct(i) }

func setupPlanHandlerTest(svc PlanningService) (*echo.Echo, *Handler) {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e, NewHandler(svc)
}

func setPlanTestAuth(c echo.Context) {
	shared.SetAuthContext(c, &shared.AuthContext{
		ParentID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		FamilyID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
	})
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestHandler_GetCalendar_200(t *testing.T) {
	svc := &mockPlanningService{
		getCalendarFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ CalendarQuery) (CalendarResponse, error) {
			return CalendarResponse{Days: []CalendarDay{}}, nil
		},
	}
	e, h := setupPlanHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/planning/calendar", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setPlanTestAuth(c)

	if err := h.getCalendar(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_GetCalendar_MissingAuth_Errors(t *testing.T) {
	e, h := setupPlanHandlerTest(&mockPlanningService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/planning/calendar", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// no auth

	if err := h.getCalendar(c); err == nil {
		t.Fatal("expected error for missing auth")
	}
}

func TestHandler_ListScheduleItems_200(t *testing.T) {
	svc := &mockPlanningService{
		listScheduleItemsFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ ScheduleItemQuery, _ *shared.PaginationParams) (*shared.PaginatedResponse[ScheduleItemResponse], error) {
			return &shared.PaginatedResponse[ScheduleItemResponse]{Data: []ScheduleItemResponse{}}, nil
		},
	}
	e, h := setupPlanHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/planning/schedule-items", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setPlanTestAuth(c)

	if err := h.listScheduleItems(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_DeleteScheduleItem_204(t *testing.T) {
	deleted := false
	svc := &mockPlanningService{
		deleteScheduleItemFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	e, h := setupPlanHandlerTest(svc)
	itemID := uuid.New()
	req := httptest.NewRequest(http.MethodDelete, "/v1/planning/schedule-items/"+itemID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(itemID.String())
	setPlanTestAuth(c)

	if err := h.deleteScheduleItem(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d", rec.Code)
	}
	if !deleted {
		t.Error("expected DeleteScheduleItem to be called")
	}
}

func TestHandler_GetDayView_InvalidDate_400(t *testing.T) {
	e, h := setupPlanHandlerTest(&mockPlanningService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/planning/calendar/day/not-a-date", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("date")
	c.SetParamValues("not-a-date")
	setPlanTestAuth(c)

	err := h.getDayView(c)
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}
