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
	// Public flag evaluation endpoint — any authenticated user can check flags. [P2-9]
	authGroup.GET("/feature-flags/evaluate", h.evaluateFlag)
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

// searchUsers godoc
//
// @Summary     Search users
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       q      query  string  false  "Search query"
// @Param       status query  string  false  "Filter by status"
// @Param       page   query  int     false  "Page number"
// @Param       limit  query  int     false  "Results per page"
// @Success     200 {object} UserSearchResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     500 {object} shared.AppError
// @Router      /admin/users [get]
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

// getUserDetail godoc
//
// @Summary     Get user detail
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Family ID"
// @Success     200 {object} AdminUserDetail
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/users/{id} [get]
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

// getUserAuditTrail godoc
//
// @Summary     Get user audit trail
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       id    path  string true  "Family ID"
// @Param       page  query int    false "Page number"
// @Param       limit query int    false "Results per page"
// @Success     200 {object} AuditLogResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/users/{id}/audit [get]
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

// suspendUser godoc
//
// @Summary     Suspend a user
// @Tags        admin
// @Accept      json
// @Security    BearerAuth
// @Param       id   path string         true "Family ID"
// @Param       body body SuspendUserInput true "Suspension reason"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/users/{id}/suspend [post]
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

// unsuspendUser godoc
//
// @Summary     Unsuspend a user
// @Tags        admin
// @Security    BearerAuth
// @Param       id path string true "Family ID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/users/{id}/unsuspend [post]
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

// banUser godoc
//
// @Summary     Ban a user
// @Tags        admin
// @Accept      json
// @Security    BearerAuth
// @Param       id   path string       true "Family ID"
// @Param       body body BanUserInput  true "Ban reason"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/users/{id}/ban [post]
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

// listFlags godoc
//
// @Summary     List all feature flags
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  FeatureFlag
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/flags [get]
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

// getFlag godoc
//
// @Summary     Get a feature flag by key
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       key path string true "Feature flag key"
// @Success     200 {object} FeatureFlag
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/flags/{key} [get]
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

// createFlag godoc
//
// @Summary     Create a feature flag
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CreateFlagInput true "Feature flag details"
// @Success     201 {object} FeatureFlag
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /admin/flags [post]
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

// updateFlag godoc
//
// @Summary     Update a feature flag
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       key  path string          true "Feature flag key"
// @Param       body body UpdateFlagInput  true "Fields to update"
// @Success     200 {object} FeatureFlag
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/flags/{key} [patch]
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
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	flag, err := h.svc.UpdateFlag(c.Request().Context(), auth, key, &input)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, flag)
}

// deleteFlag godoc
//
// @Summary     Delete a feature flag
// @Tags        admin
// @Security    BearerAuth
// @Param       key path string true "Feature flag key"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/flags/{key} [delete]
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

// getModerationQueue godoc
//
// @Summary     Get moderation queue
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       page  query int false "Page number"
// @Param       limit query int false "Results per page"
// @Success     200 {object} ModerationQueueResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/moderation/queue [get]
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

// getModerationQueueItem godoc
//
// @Summary     Get a moderation queue item
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Queue item ID"
// @Success     200 {object} ModerationQueueItem
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/moderation/queue/{id} [get]
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

// takeModerationAction godoc
//
// @Summary     Take action on a moderation queue item
// @Tags        admin
// @Accept      json
// @Security    BearerAuth
// @Param       id   path string                true "Queue item ID"
// @Param       body body ModerationActionInput  true "Action details"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/moderation/queue/{id}/action [post]
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

// listMethodologies godoc
//
// @Summary     List methodology configurations
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  MethodologyConfig
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/methodologies [get]
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

// updateMethodologyConfig godoc
//
// @Summary     Update methodology configuration
// @Tags        admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       slug path string                  true "Methodology slug"
// @Param       body body UpdateMethodologyInput   true "Fields to update"
// @Success     200 {object} MethodologyConfig
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/methodologies/{slug} [patch]
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
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	config, err := h.svc.UpdateMethodologyConfig(c.Request().Context(), auth, slug, &input)
	if err != nil {
		return mapAdminError(err)
	}
	return c.JSON(http.StatusOK, config)
}

// ─── Lifecycle Management ───────────────────────────────────────────────────

// getPendingDeletions godoc
//
// @Summary     Get pending account deletions
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       page  query int false "Page number"
// @Param       limit query int false "Results per page"
// @Success     200 {object} PendingDeletionsResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/lifecycle/deletions [get]
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

// getRecoveryRequests godoc
//
// @Summary     Get account recovery requests
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       page  query int false "Page number"
// @Param       limit query int false "Results per page"
// @Success     200 {object} RecoveryRequestsResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/lifecycle/recoveries [get]
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

// resolveRecoveryRequest godoc
//
// @Summary     Resolve an account recovery request
// @Tags        admin
// @Accept      json
// @Security    BearerAuth
// @Param       id   path string                true "Recovery request ID"
// @Param       body body ResolveRecoveryInput   true "Resolution decision"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/lifecycle/recoveries/{id}/resolve [post]
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
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.ResolveRecoveryRequest(c.Request().Context(), auth, requestID, input.Approved); err != nil {
		return mapAdminError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── System Health ──────────────────────────────────────────────────────────

// getSystemHealth godoc
//
// @Summary     Get system health status
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} SystemHealthResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/system/health [get]
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

// getJobStatus godoc
//
// @Summary     Get background job status
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} JobStatusResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/system/jobs [get]
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

// getDeadLetterJobs godoc
//
// @Summary     List dead-letter jobs
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       page  query int false "Page number"
// @Param       limit query int false "Results per page"
// @Success     200 {object} DeadLetterJobsResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/system/jobs/dead-letter [get]
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

// retryDeadLetterJob godoc
//
// @Summary     Retry a dead-letter job
// @Tags        admin
// @Security    BearerAuth
// @Param       id path string true "Job ID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/system/jobs/dead-letter/{id}/retry [post]
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

// searchAuditLog godoc
//
// @Summary     Search audit log
// @Tags        admin
// @Produce     json
// @Security    BearerAuth
// @Param       action    query string false "Filter by action"
// @Param       family_id query string false "Filter by family ID"
// @Param       page      query int    false "Page number"
// @Param       limit     query int    false "Results per page"
// @Success     200 {object} AuditLogResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/audit [get]
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

// ─── Feature Flag Evaluation (Public API) ────────────────────────────────────

// FlagEvaluationResponse is the response for the public flag evaluation endpoint.
type FlagEvaluationResponse struct {
	Key     string `json:"key"`
	Enabled bool   `json:"enabled"`
}

// evaluateFlag godoc
//
// @Summary     Evaluate a feature flag for the current family
// @Tags        feature-flags
// @Produce     json
// @Security    BearerAuth
// @Param       key query string true "Feature flag key"
// @Success     200 {object} FlagEvaluationResponse
// @Failure     401 {object} shared.AppError
// @Router      /feature-flags/evaluate [get]
func (h *Handler) evaluateFlag(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}

	key := c.QueryParam("key")
	if key == "" {
		return shared.ErrBadRequest("key query parameter is required")
	}

	familyID := auth.FamilyID
	enabled, err := h.svc.IsFlagEnabled(c.Request().Context(), key, &familyID)
	if err != nil {
		if errors.Is(err, ErrFlagNotFound) {
			// Unknown flags are treated as disabled — don't expose flag existence. [P2-9]
			return c.JSON(http.StatusOK, FlagEvaluationResponse{Key: key, Enabled: false})
		}
		return mapAdminError(err)
	}

	return c.JSON(http.StatusOK, FlagEvaluationResponse{Key: key, Enabled: enabled})
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
