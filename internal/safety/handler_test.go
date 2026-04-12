package safety

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Test Helpers ──────────────────────────────────────────────────────────────

func setupEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &testValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e
}

type testValidator struct {
	v *validator.Validate
}

func (tv *testValidator) Validate(i interface{}) error {
	return tv.v.Struct(i)
}

func setUserAuth(c echo.Context) *shared.AuthContext {
	auth := &shared.AuthContext{
		ParentID: uuid.Must(uuid.NewV7()),
		FamilyID: uuid.Must(uuid.NewV7()),
	}
	shared.SetAuthContext(c, auth)
	return auth
}

func setAdminAuth(c echo.Context) *shared.AuthContext {
	auth := &shared.AuthContext{
		ParentID:        uuid.Must(uuid.NewV7()),
		FamilyID:        uuid.Must(uuid.NewV7()),
		IsPlatformAdmin: true,
	}
	shared.SetAuthContext(c, auth)
	return auth
}

// ─── POST /v1/safety/reports ───────────────────────────────────────────────────

func TestHandler_SubmitReport_201(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	reportID := uuid.Must(uuid.NewV7())
	svc.submitReportFn = func(_ context.Context, _ shared.FamilyScope, _ *shared.AuthContext, cmd CreateReportCommand) (*ReportResponse, error) {
		if cmd.Category != "harassment" {
			t.Errorf("category = %q, want %q", cmd.Category, "harassment")
		}
		return &ReportResponse{
			ID:       reportID,
			Category: cmd.Category,
			Status:   "pending",
		}, nil
	}

	body := `{"target_type":"post","target_id":"` + uuid.Must(uuid.NewV7()).String() + `","category":"harassment"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/safety/reports", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserAuth(c)

	err := h.submitReport(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestHandler_SubmitReport_422_validation(t *testing.T) {
	e := setupEcho()
	h := NewHandler(&mockSafetyService{})

	// Missing required fields
	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/v1/safety/reports", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserAuth(c)

	err := h.submitReport(c)
	if err == nil {
		t.Fatal("expected validation error")
	}
	var appErr *shared.AppError
	if ok := err.(*shared.AppError); ok != nil {
		appErr = ok
	}
	if appErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusUnprocessableEntity)
	}
}

// ─── GET /v1/safety/reports ────────────────────────────────────────────────────

func TestHandler_ListMyReports_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.listMyReportsFn = func(_ context.Context, _ shared.FamilyScope, _ shared.PaginationParams) (*shared.PaginatedResponse[ReportResponse], error) {
		return &shared.PaginatedResponse[ReportResponse]{
			Data: []ReportResponse{{ID: uuid.Must(uuid.NewV7()), Status: "pending"}},
		}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/safety/reports", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserAuth(c)

	err := h.listMyReports(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── GET /v1/safety/reports/:id ────────────────────────────────────────────────

func TestHandler_GetMyReport_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	reportID := uuid.Must(uuid.NewV7())
	svc.getMyReportFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*ReportResponse, error) {
		return &ReportResponse{ID: id, Status: "pending"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/safety/reports/"+reportID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(reportID.String())
	setUserAuth(c)

	err := h.getMyReport(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_GetMyReport_404(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.getMyReportFn = func(_ context.Context, _ shared.FamilyScope, _ uuid.UUID) (*ReportResponse, error) {
		return nil, &SafetyError{Err: ErrReportNotFound}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/safety/reports/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setUserAuth(c)

	err := h.getMyReport(c)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusNotFound)
	}
}

// ─── GET /v1/safety/account-status ─────────────────────────────────────────────

func TestHandler_GetAccountStatus_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.getAccountStatusFn = func(_ context.Context, _ shared.FamilyScope) (*AccountStatusResponse, error) {
		return &AccountStatusResponse{Status: "active"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/safety/account-status", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserAuth(c)

	err := h.getAccountStatus(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── POST /v1/safety/appeals ───────────────────────────────────────────────────

func TestHandler_SubmitAppeal_201(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.submitAppealFn = func(_ context.Context, _ shared.FamilyScope, cmd CreateAppealCommand) (*AppealResponse, error) {
		return &AppealResponse{
			ID:         uuid.Must(uuid.NewV7()),
			ActionID:   cmd.ActionID,
			Status:     "pending",
			AppealText: cmd.AppealText,
			CreatedAt:  time.Now(),
		}, nil
	}

	body := `{"action_id":"` + uuid.Must(uuid.NewV7()).String() + `","appeal_text":"This was a mistake, please reconsider"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/safety/appeals", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserAuth(c)

	err := h.submitAppeal(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestHandler_SubmitAppeal_422_validation(t *testing.T) {
	e := setupEcho()
	h := NewHandler(&mockSafetyService{})

	// appeal_text too short
	body := `{"action_id":"` + uuid.Must(uuid.NewV7()).String() + `","appeal_text":"short"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/safety/appeals", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserAuth(c)

	err := h.submitAppeal(c)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

// ─── GET /v1/safety/appeals/:id ────────────────────────────────────────────────

func TestHandler_GetMyAppeal_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	appealID := uuid.Must(uuid.NewV7())
	svc.getMyAppealFn = func(_ context.Context, _ shared.FamilyScope, id uuid.UUID) (*AppealResponse, error) {
		return &AppealResponse{ID: id, Status: "pending"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/safety/appeals/"+appealID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(appealID.String())
	setUserAuth(c)

	err := h.getMyAppeal(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_GetMyAppeal_404(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.getMyAppealFn = func(_ context.Context, _ shared.FamilyScope, _ uuid.UUID) (*AppealResponse, error) {
		return nil, &SafetyError{Err: ErrAppealNotFound}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/safety/appeals/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setUserAuth(c)

	err := h.getMyAppeal(c)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusNotFound)
	}
}

// ─── GET /v1/safety/appeals ──────────────────────────────────────────────────────

func TestHandler_ListMyAppeals_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	appealID := uuid.Must(uuid.NewV7())
	svc.listMyAppealsFn = func(_ context.Context, _ shared.FamilyScope) ([]AppealResponse, error) {
		return []AppealResponse{{ID: appealID, Status: "pending"}}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/safety/appeals", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserAuth(c)

	err := h.listMyAppeals(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_ListMyAppeals_empty(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.listMyAppealsFn = func(_ context.Context, _ shared.FamilyScope) ([]AppealResponse, error) {
		return []AppealResponse{}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/safety/appeals", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setUserAuth(c)

	err := h.listMyAppeals(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── Admin: GET /v1/admin/safety/reports ────────────────────────────────────────

func TestHandler_AdminListReports_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/reports", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminAuth(c)

	err := h.adminListReports(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── Admin: GET /v1/admin/safety/reports/:id ────────────────────────────────────

func TestHandler_AdminGetReport_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	reportID := uuid.Must(uuid.NewV7())
	svc.adminGetReportFn = func(_ context.Context, _ *shared.AuthContext, id uuid.UUID) (*AdminReportResponse, error) {
		return &AdminReportResponse{ID: id, Status: "pending"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/reports/"+reportID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(reportID.String())
	setAdminAuth(c)

	err := h.adminGetReport(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_AdminGetReport_404(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminGetReportFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID) (*AdminReportResponse, error) {
		return nil, &SafetyError{Err: ErrReportNotFound}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/reports/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminGetReport(c)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusNotFound)
	}
}

// ─── Admin: PATCH /v1/admin/safety/reports/:id ──────────────────────────────────

func TestHandler_AdminUpdateReport_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminUpdateReportFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ UpdateReportCommand) (*AdminReportResponse, error) {
		return &AdminReportResponse{ID: uuid.Must(uuid.NewV7()), Status: "in_review"}, nil
	}

	adminID := uuid.Must(uuid.NewV7()).String()
	body := `{"assigned_admin_id":"` + adminID + `"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/safety/reports/"+uuid.Must(uuid.NewV7()).String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminUpdateReport(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── Admin: GET /v1/admin/safety/flags ──────────────────────────────────────────

func TestHandler_AdminListFlags_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/flags", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminAuth(c)

	err := h.adminListFlags(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── Admin: PATCH /v1/admin/safety/flags/:id ────────────────────────────────────

func TestHandler_AdminReviewFlag_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminReviewFlagFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ ReviewFlagCommand) (*ContentFlagResponse, error) {
		return &ContentFlagResponse{ID: uuid.Must(uuid.NewV7()), Reviewed: true}, nil
	}

	body := `{"action_taken":true}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/safety/flags/"+uuid.Must(uuid.NewV7()).String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminReviewFlag(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_AdminReviewFlag_404(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminReviewFlagFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ ReviewFlagCommand) (*ContentFlagResponse, error) {
		return nil, &SafetyError{Err: ErrFlagNotFound}
	}

	body := `{"action_taken":false}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/safety/flags/"+uuid.Must(uuid.NewV7()).String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminReviewFlag(c)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusNotFound)
	}
}

// ─── Admin: POST /v1/admin/safety/actions ───────────────────────────────────────

func TestHandler_AdminTakeAction_201(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminTakeActionFn = func(_ context.Context, _ *shared.AuthContext, cmd CreateModActionCommand) (*ModActionResponse, error) {
		return &ModActionResponse{
			ID:         uuid.Must(uuid.NewV7()),
			ActionType: cmd.ActionType,
			Reason:     cmd.Reason,
		}, nil
	}

	body := `{"target_family_id":"` + uuid.Must(uuid.NewV7()).String() + `","action_type":"content_removed","reason":"Violated community guidelines for content"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/actions", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminAuth(c)

	err := h.adminTakeAction(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
}

func TestHandler_AdminTakeAction_422_validation(t *testing.T) {
	e := setupEcho()
	h := NewHandler(&mockSafetyService{})

	body := `{"action_type":"content_removed"}` // missing required fields
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/actions", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminAuth(c)

	err := h.adminTakeAction(c)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

// ─── Admin: GET /v1/admin/safety/actions ────────────────────────────────────────

func TestHandler_AdminListActions_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/actions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminAuth(c)

	err := h.adminListActions(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── Admin: GET /v1/admin/safety/accounts/:family_id ────────────────────────────

func TestHandler_AdminGetAccount_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	familyID := uuid.Must(uuid.NewV7())
	svc.adminGetAccountFn = func(_ context.Context, _ *shared.AuthContext, fid uuid.UUID) (*AdminAccountStatusResponse, error) {
		return &AdminAccountStatusResponse{FamilyID: fid, Status: "active"}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/accounts/"+familyID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("family_id")
	c.SetParamValues(familyID.String())
	setAdminAuth(c)

	err := h.adminGetAccount(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_AdminGetAccount_404(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminGetAccountFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID) (*AdminAccountStatusResponse, error) {
		return nil, &SafetyError{Err: ErrActionNotFound}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/accounts/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("family_id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminGetAccount(c)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusNotFound)
	}
}

// ─── Admin: POST /v1/admin/safety/accounts/:family_id/suspend ───────────────────

func TestHandler_AdminSuspendAccount_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	familyID := uuid.Must(uuid.NewV7())
	svc.adminSuspendAccountFn = func(_ context.Context, _ *shared.AuthContext, fid uuid.UUID, _ SuspendAccountCommand) (*AdminAccountStatusResponse, error) {
		return &AdminAccountStatusResponse{FamilyID: fid, Status: "suspended"}, nil
	}

	body := `{"reason":"Repeated harassment of other users","suspension_days":7}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/accounts/"+familyID.String()+"/suspend", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("family_id")
	c.SetParamValues(familyID.String())
	setAdminAuth(c)

	err := h.adminSuspendAccount(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_AdminSuspendAccount_422(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminSuspendAccountFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ SuspendAccountCommand) (*AdminAccountStatusResponse, error) {
		return nil, &SafetyError{Err: ErrAccountBanned}
	}

	body := `{"reason":"Repeated harassment of other users","suspension_days":7}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/accounts/"+uuid.Must(uuid.NewV7()).String()+"/suspend", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("family_id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminSuspendAccount(c)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusForbidden)
	}
}

// ─── Admin: POST /v1/admin/safety/accounts/:family_id/ban ───────────────────────

func TestHandler_AdminBanAccount_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	familyID := uuid.Must(uuid.NewV7())
	svc.adminBanAccountFn = func(_ context.Context, _ *shared.AuthContext, fid uuid.UUID, _ BanAccountCommand) (*AdminAccountStatusResponse, error) {
		return &AdminAccountStatusResponse{FamilyID: fid, Status: "banned"}, nil
	}

	body := `{"reason":"Severe violation of community guidelines"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/accounts/"+familyID.String()+"/ban", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("family_id")
	c.SetParamValues(familyID.String())
	setAdminAuth(c)

	err := h.adminBanAccount(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── Admin: POST /v1/admin/safety/accounts/:family_id/lift ──────────────────────

func TestHandler_AdminLiftSuspension_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	familyID := uuid.Must(uuid.NewV7())
	svc.adminLiftSuspensionFn = func(_ context.Context, _ *shared.AuthContext, fid uuid.UUID, _ LiftSuspensionCommand) (*AdminAccountStatusResponse, error) {
		return &AdminAccountStatusResponse{FamilyID: fid, Status: "active"}, nil
	}

	body := `{"reason":"Suspension period reviewed and lifted early"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/accounts/"+familyID.String()+"/lift", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("family_id")
	c.SetParamValues(familyID.String())
	setAdminAuth(c)

	err := h.adminLiftSuspension(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_AdminLiftSuspension_422(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminLiftSuspensionFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ LiftSuspensionCommand) (*AdminAccountStatusResponse, error) {
		return nil, &SafetyError{Err: ErrInvalidActionType}
	}

	body := `{"reason":"Attempting to lift non-existent suspension"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/admin/safety/accounts/"+uuid.Must(uuid.NewV7()).String()+"/lift", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("family_id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminLiftSuspension(c)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusUnprocessableEntity)
	}
}

// ─── Admin: GET /v1/admin/safety/appeals ────────────────────────────────────────

func TestHandler_AdminListAppeals_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/appeals", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminAuth(c)

	err := h.adminListAppeals(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

// ─── Admin: PATCH /v1/admin/safety/appeals/:id ──────────────────────────────────

func TestHandler_AdminResolveAppeal_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminResolveAppealFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ ResolveAppealCommand) (*AdminAppealResponse, error) {
		return &AdminAppealResponse{ID: uuid.Must(uuid.NewV7()), Status: "granted"}, nil
	}

	body := `{"status":"granted","resolution_text":"After review the appeal is granted"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/safety/appeals/"+uuid.Must(uuid.NewV7()).String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminResolveAppeal(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_AdminResolveAppeal_422(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminResolveAppealFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ ResolveAppealCommand) (*AdminAppealResponse, error) {
		return nil, &SafetyError{Err: ErrSameAdminAppeal}
	}

	body := `{"status":"granted","resolution_text":"After review the appeal is granted"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/safety/appeals/"+uuid.Must(uuid.NewV7()).String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminResolveAppeal(c)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusUnprocessableEntity)
	}
}

// ─── Admin: GET /v1/admin/safety/dashboard ──────────────────────────────────────

func TestHandler_AdminDashboard_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminDashboardFn = func(_ context.Context, _ *shared.AuthContext) (*DashboardStats, error) {
		return &DashboardStats{PendingReports: 5, CriticalReports: 1}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/dashboard", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setAdminAuth(c)

	err := h.adminDashboard(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	var stats DashboardStats
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if stats.PendingReports != 5 {
		t.Errorf("pending_reports = %d, want 5", stats.PendingReports)
	}
}

// ─── Admin: PATCH /v1/admin/safety/flags/:id/escalate-csam ──────────────────────

func TestHandler_AdminEscalateToCsam_200(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	body := `{"admin_notes":"This content appears to contain CSAM material"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/safety/flags/"+uuid.Must(uuid.NewV7()).String()+"/escalate-csam", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminEscalateToCsam(c)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestHandler_AdminEscalateToCsam_422(t *testing.T) {
	e := setupEcho()
	svc := &mockSafetyService{}
	h := NewHandler(svc)

	svc.adminEscalateToCsamFn = func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ EscalateCsamCommand) error {
		return &SafetyError{Err: ErrFlagAlreadyReviewed}
	}

	body := `{"admin_notes":"This content appears to contain CSAM material"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/admin/safety/flags/"+uuid.Must(uuid.NewV7()).String()+"/escalate-csam", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("id")
	c.SetParamValues(uuid.Must(uuid.NewV7()).String())
	setAdminAuth(c)

	err := h.adminEscalateToCsam(c)
	if err == nil {
		t.Fatal("expected error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusUnprocessableEntity)
	}
}

// ─── Admin Auth Enforcement ─────────────────────────────────────────────────────

func TestHandler_AdminEndpoint_RequiresAdmin(t *testing.T) {
	e := setupEcho()
	h := NewHandler(&mockSafetyService{})

	req := httptest.NewRequest(http.MethodGet, "/v1/admin/safety/dashboard", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Set non-admin auth
	setUserAuth(c)

	err := h.adminDashboard(c)
	if err == nil {
		t.Fatal("expected forbidden error")
	}
	appErr := err.(*shared.AppError)
	if appErr.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusForbidden)
	}
}
