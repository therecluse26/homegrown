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

// Register registers all admin routes on the shared adminGroup.
// All endpoints require RequireAdmin. [16-admin §4]
// The adminGroup is created in app.go and shared with the safety domain.
func (h *Handler) Register(authGroup *echo.Group, adminGroup *echo.Group) {
	// User Management
	adminGroup.GET("/users", h.searchUsers)
	adminGroup.GET("/users/:id", h.getUserDetail)
	adminGroup.GET("/users/:id/audit", h.getUserAuditTrail)
	adminGroup.POST("/users/:id/suspend", h.suspendUser)
	adminGroup.POST("/users/:id/unsuspend", h.unsuspendUser)
	adminGroup.POST("/users/:id/ban", h.banUser)

	// Feature Flags
	adminGroup.GET("/flags", h.listFlags)
	adminGroup.GET("/flags/:key", h.getFlag)
	adminGroup.POST("/flags", h.createFlag)
	adminGroup.PATCH("/flags/:key", h.updateFlag)
	adminGroup.DELETE("/flags/:key", h.deleteFlag)

	// Moderation Queue
	adminGroup.GET("/moderation/queue", h.getModerationQueue)
	adminGroup.GET("/moderation/queue/:id", h.getModerationQueueItem)
	adminGroup.POST("/moderation/queue/:id/action", h.takeModerationAction)

	// Methodology Config
	adminGroup.GET("/methodologies", h.listMethodologies)
	adminGroup.PATCH("/methodologies/:slug", h.updateMethodologyConfig)

	// Lifecycle Management
	adminGroup.GET("/lifecycle/deletions", h.getPendingDeletions)
	adminGroup.GET("/lifecycle/recoveries", h.getRecoveryRequests)
	adminGroup.POST("/lifecycle/recoveries/:id/resolve", h.resolveRecoveryRequest)

	// System Health
	adminGroup.GET("/system/health", h.getSystemHealth)
	adminGroup.GET("/system/jobs", h.getJobStatus)
	adminGroup.GET("/system/jobs/dead-letter", h.getDeadLetterJobs)
	adminGroup.POST("/system/jobs/dead-letter/:id/retry", h.retryDeadLetterJob)

	// Audit Log
	adminGroup.GET("/audit", h.searchAuditLog)
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

func (h *Handler) suspendUser(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}

	var input SuspendUserInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.SuspendUser(c.Request().Context(), auth, familyID, input.Reason); err != nil {
		return mapAdminError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) unsuspendUser(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}

	if err := h.svc.UnsuspendUser(c.Request().Context(), auth, familyID); err != nil {
		return mapAdminError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) banUser(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	familyID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}

	var input BanUserInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.BanUser(c.Request().Context(), auth, familyID, input.Reason); err != nil {
		return mapAdminError(err)
	}
	return c.NoContent(http.StatusNoContent)
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

func (h *Handler) getFlag(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	key := c.Param("key")
	flag, err := h.svc.GetFlag(c.Request().Context(), auth, key)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, flag)
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

// ─── Moderation Queue ───────────────────────────────────────────────────────

func (h *Handler) getModerationQueue(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination parameters")
	}

	result, err := h.svc.GetModerationQueue(c.Request().Context(), auth, &pagination)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handler) getModerationQueueItem(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid item ID")
	}

	item, err := h.svc.GetModerationQueueItem(c.Request().Context(), auth, itemID)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, item)
}

func (h *Handler) takeModerationAction(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid item ID")
	}

	var input ModerationActionInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.TakeModerationAction(c.Request().Context(), auth, itemID, &input); err != nil {
		return mapAdminError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Methodology Config ─────────────────────────────────────────────────────

func (h *Handler) listMethodologies(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	configs, err := h.svc.ListMethodologies(c.Request().Context(), auth)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, configs)
}

func (h *Handler) updateMethodologyConfig(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	slug := c.Param("slug")

	var input UpdateMethodologyInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}

	config, err := h.svc.UpdateMethodologyConfig(c.Request().Context(), auth, slug, &input)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, config)
}

// ─── Lifecycle Management ───────────────────────────────────────────────────

func (h *Handler) getPendingDeletions(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination parameters")
	}

	result, err := h.svc.GetPendingDeletions(c.Request().Context(), auth, &pagination)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handler) getRecoveryRequests(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination parameters")
	}

	result, err := h.svc.GetRecoveryRequests(c.Request().Context(), auth, &pagination)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, result)
}

func (h *Handler) resolveRecoveryRequest(c echo.Context) error {
	auth, err := middleware.RequireAdmin(c)
	if err != nil {
		return err
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid request ID")
	}

	var input ResolveRecoveryInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}

	if err := h.svc.ResolveRecoveryRequest(c.Request().Context(), auth, requestID, input.Approved); err != nil {
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
	case errors.Is(err, ErrModerationItemNotFound):
		return shared.ErrNotFound()
	case errors.Is(err, ErrMethodologyNotFound):
		return shared.ErrNotFound()
	case errors.Is(err, ErrRecoveryRequestNotFound):
		return shared.ErrNotFound()
	case errors.Is(err, ErrFlagAlreadyExists):
		return shared.ErrConflict("feature flag key already exists")
	case errors.Is(err, ErrInvalidFlagKey):
		return shared.ErrBadRequest("invalid flag key format: must be lowercase alphanumeric with hyphens/underscores, 1-100 chars")
	default:
		return shared.ErrInternal(err)
	}
}
