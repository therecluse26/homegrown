package lifecycle

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler holds the lifecycle HTTP handler dependencies.
type Handler struct {
	svc LifecycleService
}

// NewHandler creates a new lifecycle Handler.
func NewHandler(svc LifecycleService) *Handler {
	return &Handler{svc: svc}
}

// Register registers all lifecycle routes on the provided route groups.
//   - authGroup: authenticated routes requiring session cookie
//   - unauthGroup: public group for account recovery (no auth required)
func (h *Handler) Register(authGroup, unauthGroup *echo.Group) {
	account := authGroup.Group("/account")
	account.POST("/export", h.requestExport)
	account.GET("/export/:id", h.getExportStatus)
	account.GET("/exports", h.listExports)
	account.POST("/deletion", h.requestDeletion)
	account.GET("/deletion", h.getDeletionStatus)
	account.DELETE("/deletion", h.cancelDeletion)
	account.GET("/sessions", h.listSessions)
	account.DELETE("/sessions/:id", h.revokeSession)
	account.DELETE("/sessions", h.revokeAllSessions)

	unauthGroup.POST("/account/recovery", h.initiateRecovery)
	unauthGroup.GET("/account/recovery/:id", h.getRecoveryStatus)
}

// requestExport godoc
//
// @Summary     Request a data export
// @Tags        lifecycle
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body RequestExportInput true "Export options"
// @Success     202 {object} map[string]any
// @Failure     401 {object} shared.AppError
// @Failure     500 {object} shared.AppError
// @Router      /account/export [post]
func (h *Handler) requestExport(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var req RequestExportInput
	if err := c.Bind(&req); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return shared.ValidationError(err)
	}

	exportID, err := h.svc.RequestExport(c.Request().Context(), auth, &scope, &req)
	if err != nil {
		return mapLifecycleError(err)
	}

	return c.JSON(http.StatusAccepted, map[string]any{
		"export_id": exportID,
		"status":    ExportStatusPending,
	})
}

// getExportStatus godoc
//
// @Summary     Get export status
// @Tags        lifecycle
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Export ID"
// @Success     200 {object} ExportStatusResponse
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /account/export/{id} [get]
func (h *Handler) getExportStatus(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	exportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid export ID")
	}

	resp, err := h.svc.GetExportStatus(c.Request().Context(), &scope, exportID)
	if err != nil {
		return mapLifecycleError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// listExports godoc
//
// @Summary     List data exports
// @Tags        lifecycle
// @Produce     json
// @Security    BearerAuth
// @Param       limit  query int false "Results per page"
// @Param       offset query int false "Offset"
// @Success     200 {object} PaginatedExports
// @Failure     401 {object} shared.AppError
// @Router      /account/exports [get]
func (h *Handler) listExports(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	pagination := &PaginationParams{
		Limit:  parseLimitParam(c, 20),
		Offset: parseOffsetParam(c),
	}

	resp, err := h.svc.ListExports(c.Request().Context(), &scope, pagination)
	if err != nil {
		return mapLifecycleError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// requestDeletion godoc
//
// @Summary     Request account deletion
// @Tags        lifecycle
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body RequestDeletionInput true "Deletion confirmation"
// @Success     202 {object} map[string]any
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /account/deletion [post]
func (h *Handler) requestDeletion(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var req RequestDeletionInput
	if err := c.Bind(&req); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return shared.ValidationError(err)
	}

	deletionID, err := h.svc.RequestDeletion(c.Request().Context(), auth, &scope, &req)
	if err != nil {
		return mapLifecycleError(err)
	}

	return c.JSON(http.StatusAccepted, map[string]any{
		"deletion_id": deletionID,
	})
}

// getDeletionStatus godoc
//
// @Summary     Get account deletion status
// @Tags        lifecycle
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} DeletionStatusResponse
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /account/deletion [get]
func (h *Handler) getDeletionStatus(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	resp, err := h.svc.GetDeletionStatus(c.Request().Context(), &scope)
	if err != nil {
		return mapLifecycleError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// cancelDeletion godoc
//
// @Summary     Cancel pending account deletion
// @Tags        lifecycle
// @Security    BearerAuth
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /account/deletion [delete]
func (h *Handler) cancelDeletion(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	if err := h.svc.CancelDeletion(c.Request().Context(), &scope); err != nil {
		return mapLifecycleError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// listSessions godoc
//
// @Summary     List active sessions
// @Tags        lifecycle
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]any
// @Failure     401 {object} shared.AppError
// @Router      /account/sessions [get]
func (h *Handler) listSessions(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}

	sessions, err := h.svc.ListSessions(c.Request().Context(), auth)
	if err != nil {
		return mapLifecycleError(err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"sessions": sessions,
	})
}

// revokeSession godoc
//
// @Summary     Revoke a session
// @Tags        lifecycle
// @Security    BearerAuth
// @Param       id path string true "Session ID"
// @Success     204
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Router      /account/sessions/{id} [delete]
func (h *Handler) revokeSession(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		return shared.ErrBadRequest("session ID is required")
	}

	if err := h.svc.RevokeSession(c.Request().Context(), auth, sessionID); err != nil {
		return mapLifecycleError(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// revokeAllSessions godoc
//
// @Summary     Revoke all other sessions
// @Tags        lifecycle
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]any
// @Failure     401 {object} shared.AppError
// @Router      /account/sessions [delete]
func (h *Handler) revokeAllSessions(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}

	count, err := h.svc.RevokeAllSessions(c.Request().Context(), auth)
	if err != nil {
		return mapLifecycleError(err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"revoked_count": count,
	})
}

// initiateRecovery godoc
//
// @Summary     Initiate account recovery
// @Tags        lifecycle
// @Accept      json
// @Produce     json
// @Param       body body InitiateRecoveryInput true "Recovery details"
// @Success     202 {object} map[string]any
// @Failure     400 {object} shared.AppError
// @Router      /account/recovery [post]
func (h *Handler) initiateRecovery(c echo.Context) error {
	var req InitiateRecoveryInput
	if err := c.Bind(&req); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&req); err != nil {
		return shared.ValidationError(err)
	}

	// Always returns 202 regardless of whether email exists — enumeration prevention.
	// [15-data-lifecycle §13]
	recoveryID, err := h.svc.InitiateRecovery(c.Request().Context(), &req)
	if err != nil {
		return mapLifecycleError(err)
	}

	return c.JSON(http.StatusAccepted, map[string]any{
		"recovery_id": recoveryID,
	})
}

// getRecoveryStatus godoc
//
// @Summary     Get account recovery status
// @Tags        lifecycle
// @Produce     json
// @Param       id path string true "Recovery ID"
// @Success     200 {object} RecoveryStatusResponse
// @Failure     400 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /account/recovery/{id} [get]
func (h *Handler) getRecoveryStatus(c echo.Context) error {
	recoveryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid recovery ID")
	}

	resp, err := h.svc.GetRecoveryStatus(c.Request().Context(), recoveryID)
	if err != nil {
		return mapLifecycleError(err)
	}

	return c.JSON(http.StatusOK, resp)
}

// mapLifecycleError maps lifecycle domain errors to HTTP errors.
func mapLifecycleError(err error) error {
	switch {
	case errors.Is(err, ErrExportNotFound),
		errors.Is(err, ErrDeletionNotFound),
		errors.Is(err, ErrRecoveryNotFound):
		return shared.ErrNotFound()
	case errors.Is(err, ErrExportExpired),
		errors.Is(err, ErrRecoveryExpired):
		return &shared.AppError{Code: "expired", Message: "resource has expired", StatusCode: http.StatusGone}
	case errors.Is(err, ErrDeletionAlreadyPending):
		return shared.ErrConflict("an active deletion request already exists")
	case errors.Is(err, ErrGracePeriodExpired):
		return shared.ErrConflict("grace period has ended — deletion cannot be cancelled")
	case errors.Is(err, ErrNotPrimaryParent):
		return shared.ErrForbidden()
	case errors.Is(err, ErrCannotRevokeCurrent):
		return shared.ErrBadRequest("cannot revoke current session via this endpoint")
	default:
		return err
	}
}

// parseLimitParam parses ?limit= with a default value.
func parseLimitParam(c echo.Context, defaultVal int64) int64 {
	s := c.QueryParam("limit")
	if s == "" {
		return defaultVal
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 1 {
		return defaultVal
	}
	if n > 100 {
		return 100
	}
	return n
}

// parseOffsetParam parses ?offset= defaulting to 0.
func parseOffsetParam(c echo.Context) int64 {
	s := c.QueryParam("offset")
	if s == "" {
		return 0
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
