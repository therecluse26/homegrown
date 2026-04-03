package lifecycle

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
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── Test Helpers ────────────────────────────────────────────────────────────

type echoValidator struct{ v *validator.Validate }

func (cv *echoValidator) Validate(i any) error { return cv.v.Struct(i) }

var (
	testFamilyID = uuid.Must(uuid.NewV7())
	testParentID = uuid.Must(uuid.NewV7())
)

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e
}

func setupLifecycleRoutes(e *echo.Echo, svc LifecycleService) {
	auth := e.Group("/v1")
	auth.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			shared.SetAuthContext(c, &shared.AuthContext{
				ParentID:           testParentID,
				FamilyID:           testFamilyID,
				SessionID:          "test-session-id",
				CoppaConsentStatus: "consented",
			})
			return next(c)
		}
	})
	public := e.Group("/v1")
	NewHandler(svc).Register(auth, public)
}

// ─── Function-Pointer Mock Service ──────────────────────────────────────────

type handlerMockService struct {
	requestExportFn     func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, req *RequestExportInput) (uuid.UUID, error)
	getExportStatusFn   func(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (*ExportStatusResponse, error)
	listExportsFn       func(ctx context.Context, scope *shared.FamilyScope, pagination *PaginationParams) (*PaginatedExports, error)
	requestDeletionFn   func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, req *RequestDeletionInput) (uuid.UUID, error)
	getDeletionStatusFn func(ctx context.Context, scope *shared.FamilyScope) (*DeletionStatusResponse, error)
	cancelDeletionFn    func(ctx context.Context, scope *shared.FamilyScope) error
	listSessionsFn      func(ctx context.Context, auth *shared.AuthContext) ([]SessionInfo, error)
	revokeSessionFn     func(ctx context.Context, auth *shared.AuthContext, sessionID string) error
	revokeAllSessionsFn func(ctx context.Context, auth *shared.AuthContext) (uint32, error)
	initiateRecoveryFn  func(ctx context.Context, req *InitiateRecoveryInput) (uuid.UUID, error)
	getRecoveryStatusFn func(ctx context.Context, recoveryID uuid.UUID) (*RecoveryStatusResponse, error)
}

func newMockService() *handlerMockService { return &handlerMockService{} }

func (m *handlerMockService) RequestExport(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, req *RequestExportInput) (uuid.UUID, error) {
	if m.requestExportFn != nil {
		return m.requestExportFn(ctx, auth, scope, req)
	}
	panic("RequestExport not stubbed")
}
func (m *handlerMockService) GetExportStatus(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (*ExportStatusResponse, error) {
	if m.getExportStatusFn != nil {
		return m.getExportStatusFn(ctx, scope, exportID)
	}
	panic("GetExportStatus not stubbed")
}
func (m *handlerMockService) ListExports(ctx context.Context, scope *shared.FamilyScope, pagination *PaginationParams) (*PaginatedExports, error) {
	if m.listExportsFn != nil {
		return m.listExportsFn(ctx, scope, pagination)
	}
	panic("ListExports not stubbed")
}
func (m *handlerMockService) RequestDeletion(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, req *RequestDeletionInput) (uuid.UUID, error) {
	if m.requestDeletionFn != nil {
		return m.requestDeletionFn(ctx, auth, scope, req)
	}
	panic("RequestDeletion not stubbed")
}
func (m *handlerMockService) GetDeletionStatus(ctx context.Context, scope *shared.FamilyScope) (*DeletionStatusResponse, error) {
	if m.getDeletionStatusFn != nil {
		return m.getDeletionStatusFn(ctx, scope)
	}
	panic("GetDeletionStatus not stubbed")
}
func (m *handlerMockService) CancelDeletion(ctx context.Context, scope *shared.FamilyScope) error {
	if m.cancelDeletionFn != nil {
		return m.cancelDeletionFn(ctx, scope)
	}
	panic("CancelDeletion not stubbed")
}
func (m *handlerMockService) ListSessions(ctx context.Context, auth *shared.AuthContext) ([]SessionInfo, error) {
	if m.listSessionsFn != nil {
		return m.listSessionsFn(ctx, auth)
	}
	panic("ListSessions not stubbed")
}
func (m *handlerMockService) RevokeSession(ctx context.Context, auth *shared.AuthContext, sessionID string) error {
	if m.revokeSessionFn != nil {
		return m.revokeSessionFn(ctx, auth, sessionID)
	}
	panic("RevokeSession not stubbed")
}
func (m *handlerMockService) RevokeAllSessions(ctx context.Context, auth *shared.AuthContext) (uint32, error) {
	if m.revokeAllSessionsFn != nil {
		return m.revokeAllSessionsFn(ctx, auth)
	}
	panic("RevokeAllSessions not stubbed")
}
func (m *handlerMockService) InitiateRecovery(ctx context.Context, req *InitiateRecoveryInput) (uuid.UUID, error) {
	if m.initiateRecoveryFn != nil {
		return m.initiateRecoveryFn(ctx, req)
	}
	panic("InitiateRecovery not stubbed")
}
func (m *handlerMockService) GetRecoveryStatus(ctx context.Context, recoveryID uuid.UUID) (*RecoveryStatusResponse, error) {
	if m.getRecoveryStatusFn != nil {
		return m.getRecoveryStatusFn(ctx, recoveryID)
	}
	panic("GetRecoveryStatus not stubbed")
}

// Stub methods not tested at handler level.
func (m *handlerMockService) ProcessExport(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (m *handlerMockService) ProcessDeletion(_ context.Context) error          { return nil }
func (m *handlerMockService) ProcessSingleDeletion(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (m *handlerMockService) HandleFamilyDeletion(_ context.Context, _ uuid.UUID) error {
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Data Export Handler Tests [15-data-lifecycle §8]
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandler_RequestExport_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	exportID := uuid.Must(uuid.NewV7())
	svc.requestExportFn = func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ *RequestExportInput) (uuid.UUID, error) {
		return exportID, nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/account/export", strings.NewReader(`{"format":"json"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, exportID.String(), body["export_id"])
	assert.Equal(t, string(ExportStatusPending), body["status"])
}

func TestHandler_GetExportStatus_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	exportID := uuid.Must(uuid.NewV7())
	now := time.Now().UTC().Truncate(time.Second)
	svc.getExportStatusFn = func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (*ExportStatusResponse, error) {
		assert.Equal(t, exportID, id)
		return &ExportStatusResponse{
			ID:        exportID,
			Status:    ExportStatusCompleted,
			Format:    ExportFormatJSON,
			CreatedAt: now,
			ExpiresAt: now.Add(72 * time.Hour),
		}, nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/export/"+exportID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body ExportStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, ExportStatusCompleted, body.Status)
}

func TestHandler_GetExportStatus_InvalidID(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/export/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_GetExportStatus_NotFound(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.getExportStatusFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportStatusResponse, error) {
		return nil, ErrExportNotFound
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/export/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_ListExports_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.listExportsFn = func(_ context.Context, _ *shared.FamilyScope, _ *PaginationParams) (*PaginatedExports, error) {
		return &PaginatedExports{
			Items: []ExportSummary{{ID: uuid.Must(uuid.NewV7()), Status: ExportStatusPending, Format: ExportFormatJSON}},
			Total: 1,
		}, nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/exports?limit=10", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body PaginatedExports
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, int64(1), body.Total)
	assert.Len(t, body.Items, 1)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Account Deletion Handler Tests [15-data-lifecycle §8]
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandler_RequestDeletion_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	deletionID := uuid.Must(uuid.NewV7())
	svc.requestDeletionFn = func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, req *RequestDeletionInput) (uuid.UUID, error) {
		assert.Equal(t, DeletionTypeFamily, req.DeletionType)
		return deletionID, nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/account/deletion", strings.NewReader(`{"deletion_type":"family"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, deletionID.String(), body["deletion_id"])
}

func TestHandler_RequestDeletion_ValidationError(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	setupLifecycleRoutes(e, svc)

	// Missing required deletion_type.
	req := httptest.NewRequest(http.MethodPost, "/v1/account/deletion", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestHandler_RequestDeletion_AlreadyPending(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.requestDeletionFn = func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ *RequestDeletionInput) (uuid.UUID, error) {
		return uuid.Nil, ErrDeletionAlreadyPending
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/account/deletion", strings.NewReader(`{"deletion_type":"family"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestHandler_GetDeletionStatus_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	now := time.Now().UTC().Truncate(time.Second)
	svc.getDeletionStatusFn = func(_ context.Context, _ *shared.FamilyScope) (*DeletionStatusResponse, error) {
		return &DeletionStatusResponse{
			ID:                uuid.Must(uuid.NewV7()),
			Status:            DeletionStatusGracePeriod,
			DeletionType:      DeletionTypeFamily,
			GracePeriodEndsAt: now.Add(30 * 24 * time.Hour),
			CreatedAt:         now,
		}, nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/deletion", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body DeletionStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, DeletionStatusGracePeriod, body.Status)
}

func TestHandler_GetDeletionStatus_NotFound(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.getDeletionStatusFn = func(_ context.Context, _ *shared.FamilyScope) (*DeletionStatusResponse, error) {
		return nil, ErrDeletionNotFound
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/deletion", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_CancelDeletion_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.cancelDeletionFn = func(_ context.Context, _ *shared.FamilyScope) error {
		return nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodDelete, "/v1/account/deletion", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandler_CancelDeletion_GracePeriodExpired(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.cancelDeletionFn = func(_ context.Context, _ *shared.FamilyScope) error {
		return ErrGracePeriodExpired
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodDelete, "/v1/account/deletion", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Session Management Handler Tests [15-data-lifecycle §8]
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandler_ListSessions_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.listSessionsFn = func(_ context.Context, _ *shared.AuthContext) ([]SessionInfo, error) {
		return []SessionInfo{
			{SessionID: "session-1", IsCurrent: true, LastActive: time.Now().UTC()},
			{SessionID: "session-2", IsCurrent: false, LastActive: time.Now().UTC()},
		}, nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/sessions", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	sessions := body["sessions"].([]any)
	assert.Len(t, sessions, 2)
}

func TestHandler_RevokeSession_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.revokeSessionFn = func(_ context.Context, _ *shared.AuthContext, sessionID string) error {
		assert.Equal(t, "target-session-id", sessionID)
		return nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodDelete, "/v1/account/sessions/target-session-id", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestHandler_RevokeSession_CannotRevokeCurrent(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.revokeSessionFn = func(_ context.Context, _ *shared.AuthContext, _ string) error {
		return ErrCannotRevokeCurrent
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodDelete, "/v1/account/sessions/current-session", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandler_RevokeAllSessions_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.revokeAllSessionsFn = func(_ context.Context, _ *shared.AuthContext) (uint32, error) {
		return 3, nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodDelete, "/v1/account/sessions", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(3), body["revoked_count"])
}

// ═══════════════════════════════════════════════════════════════════════════════
// Account Recovery Handler Tests [15-data-lifecycle §8]
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandler_InitiateRecovery_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	recoveryID := uuid.Must(uuid.NewV7())
	svc.initiateRecoveryFn = func(_ context.Context, req *InitiateRecoveryInput) (uuid.UUID, error) {
		assert.Equal(t, "user@example.com", req.Email)
		return recoveryID, nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/account/recovery", strings.NewReader(`{"email":"user@example.com"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusAccepted, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, recoveryID.String(), body["recovery_id"])
}

func TestHandler_InitiateRecovery_ValidationError(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	setupLifecycleRoutes(e, svc)

	// Missing email.
	req := httptest.NewRequest(http.MethodPost, "/v1/account/recovery", strings.NewReader(`{}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestHandler_GetRecoveryStatus_Success(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	recoveryID := uuid.Must(uuid.NewV7())
	now := time.Now().UTC().Truncate(time.Second)
	svc.getRecoveryStatusFn = func(_ context.Context, id uuid.UUID) (*RecoveryStatusResponse, error) {
		assert.Equal(t, recoveryID, id)
		return &RecoveryStatusResponse{
			ID:                 recoveryID,
			Status:             RecoveryStatusPending,
			VerificationMethod: VerificationMethodEmail,
			CreatedAt:          now,
		}, nil
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/recovery/"+recoveryID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body RecoveryStatusResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, RecoveryStatusPending, body.Status)
}

func TestHandler_GetRecoveryStatus_NotFound(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.getRecoveryStatusFn = func(_ context.Context, _ uuid.UUID) (*RecoveryStatusResponse, error) {
		return nil, ErrRecoveryNotFound
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/recovery/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestHandler_GetRecoveryStatus_InvalidID(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/recovery/bad-id", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Error Mapping Tests [15-data-lifecycle §16]
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandler_MapLifecycleError_NotPrimaryParent(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.requestDeletionFn = func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ *RequestDeletionInput) (uuid.UUID, error) {
		return uuid.Nil, ErrNotPrimaryParent
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/account/deletion", strings.NewReader(`{"deletion_type":"family"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
}

func TestHandler_MapLifecycleError_ExportExpired(t *testing.T) {
	e := newTestEcho()
	svc := newMockService()
	svc.getExportStatusFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ExportStatusResponse, error) {
		return nil, ErrExportExpired
	}
	setupLifecycleRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/account/export/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusGone, rec.Code)
}
