package admin

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/homegrown-academy/homegrown-academy/internal/middleware"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Handler handles HTTP requests for the admin domain. [16-admin §4]
type Handler struct {
	svc AdminService
}

// NewHandler constructs an admin Handler.
func NewHandler(svc AdminService) *Handler {
	return &Handler{svc: svc}
}

// Register registers all admin routes. All endpoints require RequireAdmin. [16-admin §4]
func (h *Handler) Register(authGroup *echo.Group) {
	admin := authGroup.Group("/admin")

	// User Management
	admin.GET("/users", h.searchUsers)
	admin.GET("/users/:id", h.getUserDetail)
	admin.GET("/users/:id/audit", h.getUserAuditTrail)

	// Feature Flags
	admin.GET("/flags", h.listFlags)
	admin.POST("/flags", h.createFlag)
	admin.PATCH("/flags/:key", h.updateFlag)
	admin.DELETE("/flags/:key", h.deleteFlag)

	// System Health
	admin.GET("/system/health", h.getSystemHealth)
	admin.GET("/system/jobs", h.getJobStatus)
	admin.GET("/system/jobs/dead-letter", h.getDeadLetterJobs)
	admin.POST("/system/jobs/dead-letter/:id/retry", h.retryDeadLetterJob)

	// Audit Log
	admin.GET("/audit", h.searchAuditLog)
}

// ─── User Management ────────────────────────────────────────────────────────

func (h *Handler) searchUsers(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	var query UserSearchQuery
	if err := c.Bind(&query); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination parameters")
	}

	result, err := h.svc.SearchUsers(c.Request().Context(), auth, &query, &pagination)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handler) getUserDetail(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}

	detail, err := h.svc.GetUserDetail(c.Request().Context(), auth, familyID)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, detail)
}

func (h *Handler) getUserAuditTrail(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination parameters")
	}

	result, err := h.svc.GetUserAuditTrail(c.Request().Context(), auth, familyID, &pagination)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, result)
}

// ─── Feature Flags ──────────────────────────────────────────────────────────

func (h *Handler) listFlags(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	flags, err := h.svc.ListFlags(c.Request().Context(), auth)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, flags)
}

func (h *Handler) createFlag(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	var input CreateFlagInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	flag, err := h.svc.CreateFlag(c.Request().Context(), auth, &input)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusCreated, flag)
}

func (h *Handler) updateFlag(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	key := c.Param("key")

	var input UpdateFlagInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}

	flag, err := h.svc.UpdateFlag(c.Request().Context(), auth, key, &input)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, flag)
}

func (h *Handler) deleteFlag(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	key := c.Param("key")
	if err := h.svc.DeleteFlag(c.Request().Context(), auth, key); err != nil {
		return mapAdminError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── System Health ──────────────────────────────────────────────────────────

func (h *Handler) getSystemHealth(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	health, err := h.svc.GetSystemHealth(c.Request().Context(), auth)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, health)
}

func (h *Handler) getJobStatus(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	status, err := h.svc.GetJobStatus(c.Request().Context(), auth)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, status)
}

func (h *Handler) getDeadLetterJobs(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination parameters")
	}

	result, err := h.svc.GetDeadLetterJobs(c.Request().Context(), auth, &pagination)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handler) retryDeadLetterJob(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	jobID := c.Param("id")
	if err := h.svc.RetryDeadLetterJob(c.Request().Context(), auth, jobID); err != nil {
		return mapAdminError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Audit Log ──────────────────────────────────────────────────────────────

func (h *Handler) searchAuditLog(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	var query AuditLogQuery
	if err := c.Bind(&query); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination parameters")
	}

	result, err := h.svc.SearchAuditLog(c.Request().Context(), auth, &query, &pagination)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, result)
}

// ─── Error Mapping [16-admin §13] ───────────────────────────────────────────

func mapAdminError(err error) error {
	switch {
	case errors.Is(err, ErrFlagNotFound):
		return shared.ErrNotFound()
	case errors.Is(err, ErrUserNotFound):
		return shared.ErrNotFound()
	case errors.Is(err, ErrDeadLetterNotFound):
		return shared.ErrNotFound()
	case errors.Is(err, ErrFlagAlreadyExists):
		return shared.ErrConflict("feature flag key already exists")
	case errors.Is(err, ErrInvalidFlagKey):
		return shared.ErrBadRequest("invalid flag key format: must be lowercase alphanumeric with hyphens/underscores, 1-100 chars")
	default:
		return shared.ErrInternal(err)
	}
}
