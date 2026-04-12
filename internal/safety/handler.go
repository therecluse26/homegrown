package safety

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Handler handles HTTP requests for the safety domain. [11-safety §4]
type Handler struct {
	svc SafetyService
}

// NewHandler constructs a safety Handler.
func NewHandler(svc SafetyService) *Handler {
	return &Handler{svc: svc}
}

// Register registers safety routes on authenticated and admin route groups. [11-safety §4.1]
func (h *Handler) Register(authGroup *echo.Group, adminGroup *echo.Group) {
	// User-facing routes
	g := authGroup.Group("/safety")
	g.POST("/reports", h.submitReport)
	g.GET("/reports", h.listMyReports)
	g.GET("/reports/:id", h.getMyReport)
	g.GET("/account-status", h.getAccountStatus)
	g.POST("/appeals", h.submitAppeal)
	g.GET("/appeals", h.listMyAppeals)
	g.GET("/appeals/:id", h.getMyAppeal)

	// Phase 2: Parental controls (user-facing)
	g.GET("/parental-controls", h.getParentalControls)
	g.PUT("/parental-controls", h.upsertParentalControl)
	g.DELETE("/parental-controls/:id", h.deleteParentalControl)

	// Admin routes
	a := adminGroup.Group("/safety")
	a.GET("/reports", h.adminListReports)
	a.GET("/reports/:id", h.adminGetReport)
	a.PATCH("/reports/:id", h.adminUpdateReport)
	a.GET("/flags", h.adminListFlags)
	a.PATCH("/flags/:id", h.adminReviewFlag)
	a.PATCH("/flags/:id/escalate-csam", h.adminEscalateToCsam)
	a.POST("/actions", h.adminTakeAction)
	a.GET("/actions", h.adminListActions)
	a.GET("/accounts/:family_id", h.adminGetAccount)
	a.POST("/accounts/:family_id/suspend", h.adminSuspendAccount)
	a.POST("/accounts/:family_id/ban", h.adminBanAccount)
	a.POST("/accounts/:family_id/lift", h.adminLiftSuspension)
	a.GET("/appeals", h.adminListAppeals)
	a.PATCH("/appeals/:id", h.adminResolveAppeal)
	a.GET("/dashboard", h.adminDashboard)

	// Phase 2: Admin roles
	a.GET("/roles", h.adminListRoles)
	a.POST("/roles", h.adminCreateRole)
	a.POST("/roles/:role_id/assign", h.adminAssignRole)
	a.DELETE("/roles/:role_id/assignments/:parent_id", h.adminRevokeRole)
	a.GET("/roles/:role_id/assignments", h.adminListRoleAssignments)

	// Phase 2: Grooming scores
	a.GET("/grooming-scores", h.adminListGroomingScores)
	a.PATCH("/grooming-scores/:id", h.adminReviewGroomingScore)
}

// ─── User-Facing Endpoints ──────────────────────────────────────────────────────

// submitReport godoc
//
// @Summary     Submit a safety report
// @Tags        safety
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CreateReportCommand true "Report details"
// @Success     201 {object} ReportResponse
// @Failure     401 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /safety/reports [post]
func (h *Handler) submitReport(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var cmd CreateReportCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.SubmitReport(c.Request().Context(), scope, auth, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listMyReports godoc
//
// @Summary     List my safety reports
// @Tags        safety
// @Produce     json
// @Security    BearerAuth
// @Param       page  query int false "Page number"
// @Param       limit query int false "Results per page"
// @Success     200 {object} ReportListResult
// @Failure     401 {object} shared.AppError
// @Router      /safety/reports [get]
func (h *Handler) listMyReports(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination")
	}

	resp, err := h.svc.ListMyReports(c.Request().Context(), scope, pagination)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getMyReport godoc
//
// @Summary     Get one of my safety reports
// @Tags        safety
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Report ID"
// @Success     200 {object} ReportResponse
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /safety/reports/{id} [get]
func (h *Handler) getMyReport(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid report ID")
	}

	resp, err := h.svc.GetMyReport(c.Request().Context(), scope, id)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getAccountStatus godoc
//
// @Summary     Get account safety status
// @Tags        safety
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} AccountStatusResponse
// @Failure     401 {object} shared.AppError
// @Router      /safety/account-status [get]
func (h *Handler) getAccountStatus(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.GetAccountStatus(c.Request().Context(), scope)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// submitAppeal godoc
//
// @Summary     Submit an appeal
// @Tags        safety
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CreateAppealCommand true "Appeal details"
// @Success     201 {object} AppealResponse
// @Failure     401 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /safety/appeals [post]
func (h *Handler) submitAppeal(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var cmd CreateAppealCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.SubmitAppeal(c.Request().Context(), scope, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// getMyAppeal godoc
//
// @Summary     Get one of my appeals
// @Tags        safety
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Appeal ID"
// @Success     200 {object} AppealResponse
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /safety/appeals/{id} [get]
func (h *Handler) getMyAppeal(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid appeal ID")
	}

	resp, err := h.svc.GetMyAppeal(c.Request().Context(), scope, id)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// listMyAppeals godoc
//
// @Summary     List my appeals
// @Tags        safety
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  AppealResponse
// @Failure     401 {object} shared.AppError
// @Router      /safety/appeals [get]
func (h *Handler) listMyAppeals(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.ListMyAppeals(c.Request().Context(), scope)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Admin Endpoints ────────────────────────────────────────────────────────────

// adminListReports godoc
//
// @Summary     List all safety reports (admin)
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Param       status query string false "Filter by status"
// @Param       page   query int    false "Page number"
// @Param       limit  query int    false "Results per page"
// @Success     200 {object} AdminReportListResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/safety/reports [get]
func (h *Handler) adminListReports(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	var filter ReportFilter
	if err := c.Bind(&filter); err != nil {
		return shared.ErrBadRequest("invalid filter")
	}
	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination")
	}

	resp, err := h.svc.AdminListReports(c.Request().Context(), auth, filter, pagination)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminGetReport godoc
//
// @Summary     Get a safety report (admin)
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Report ID"
// @Success     200 {object} ReportResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/reports/{id} [get]
func (h *Handler) adminGetReport(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid report ID")
	}

	resp, err := h.svc.AdminGetReport(c.Request().Context(), auth, id)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminUpdateReport godoc
//
// @Summary     Update a safety report (admin)
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string              true "Report ID"
// @Param       body body UpdateReportCommand  true "Fields to update"
// @Success     200 {object} ReportResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/reports/{id} [patch]
func (h *Handler) adminUpdateReport(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid report ID")
	}

	var cmd UpdateReportCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AdminUpdateReport(c.Request().Context(), auth, id, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminListFlags godoc
//
// @Summary     List safety flags (admin)
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Param       severity query string false "Filter by severity"
// @Param       page     query int    false "Page number"
// @Param       limit    query int    false "Results per page"
// @Success     200 {object} ContentFlagListResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/safety/flags [get]
func (h *Handler) adminListFlags(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	var filter FlagFilter
	if err := c.Bind(&filter); err != nil {
		return shared.ErrBadRequest("invalid filter")
	}
	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination")
	}

	resp, err := h.svc.AdminListFlags(c.Request().Context(), auth, filter, pagination)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminReviewFlag godoc
//
// @Summary     Review a safety flag (admin)
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string             true "Flag ID"
// @Param       body body ReviewFlagCommand   true "Review decision"
// @Success     200 {object} ContentFlagResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/flags/{id} [patch]
func (h *Handler) adminReviewFlag(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid flag ID")
	}

	var cmd ReviewFlagCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AdminReviewFlag(c.Request().Context(), auth, id, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminEscalateToCsam godoc
//
// @Summary     Escalate flag to CSAM (admin)
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string               true "Flag ID"
// @Param       body body EscalateCsamCommand   true "Escalation details"
// @Success     200 {object} map[string]string
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/flags/{id}/escalate-csam [patch]
func (h *Handler) adminEscalateToCsam(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid flag ID")
	}

	var cmd EscalateCsamCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.AdminEscalateToCsam(c.Request().Context(), auth, id, cmd); err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "escalated"})
}

// adminTakeAction godoc
//
// @Summary     Take a moderation action (admin)
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CreateModActionCommand true "Action details"
// @Success     201 {object} ModActionResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /admin/safety/actions [post]
func (h *Handler) adminTakeAction(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	var cmd CreateModActionCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AdminTakeAction(c.Request().Context(), auth, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// adminListActions godoc
//
// @Summary     List moderation actions (admin)
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Param       action_type query string false "Filter by action type"
// @Param       page        query int    false "Page number"
// @Param       limit       query int    false "Results per page"
// @Success     200 {object} ModActionListResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/safety/actions [get]
func (h *Handler) adminListActions(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	var filter ActionFilter
	if err := c.Bind(&filter); err != nil {
		return shared.ErrBadRequest("invalid filter")
	}
	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination")
	}

	resp, err := h.svc.AdminListActions(c.Request().Context(), auth, filter, pagination)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminGetAccount godoc
//
// @Summary     Get account safety details (admin)
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Param       family_id path string true "Family ID"
// @Success     200 {object} AdminAccountStatusResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/accounts/{family_id} [get]
func (h *Handler) adminGetAccount(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	familyID, err := uuid.Parse(c.Param("family_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}

	resp, err := h.svc.AdminGetAccount(c.Request().Context(), auth, familyID)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminSuspendAccount godoc
//
// @Summary     Suspend an account (admin)
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       family_id path string                true "Family ID"
// @Param       body      body SuspendAccountCommand  true "Suspension details"
// @Success     200 {object} AdminAccountStatusResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/accounts/{family_id}/suspend [post]
func (h *Handler) adminSuspendAccount(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	familyID, err := uuid.Parse(c.Param("family_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}

	var cmd SuspendAccountCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AdminSuspendAccount(c.Request().Context(), auth, familyID, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminBanAccount godoc
//
// @Summary     Ban an account (admin)
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       family_id path string            true "Family ID"
// @Param       body      body BanAccountCommand  true "Ban details"
// @Success     200 {object} AdminAccountStatusResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/accounts/{family_id}/ban [post]
func (h *Handler) adminBanAccount(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	familyID, err := uuid.Parse(c.Param("family_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}

	var cmd BanAccountCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AdminBanAccount(c.Request().Context(), auth, familyID, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminLiftSuspension godoc
//
// @Summary     Lift account suspension (admin)
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       family_id path string                 true "Family ID"
// @Param       body      body LiftSuspensionCommand   true "Lift details"
// @Success     200 {object} AdminAccountStatusResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/accounts/{family_id}/lift [post]
func (h *Handler) adminLiftSuspension(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	familyID, err := uuid.Parse(c.Param("family_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}

	var cmd LiftSuspensionCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AdminLiftSuspension(c.Request().Context(), auth, familyID, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminListAppeals godoc
//
// @Summary     List appeals (admin)
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Param       status query string false "Filter by status"
// @Param       page   query int    false "Page number"
// @Param       limit  query int    false "Results per page"
// @Success     200 {object} AdminAppealListResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/safety/appeals [get]
func (h *Handler) adminListAppeals(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	var filter AppealFilter
	if err := c.Bind(&filter); err != nil {
		return shared.ErrBadRequest("invalid filter")
	}
	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination")
	}

	resp, err := h.svc.AdminListAppeals(c.Request().Context(), auth, filter, pagination)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminResolveAppeal godoc
//
// @Summary     Resolve an appeal (admin)
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string                true "Appeal ID"
// @Param       body body ResolveAppealCommand   true "Resolution details"
// @Success     200 {object} AppealResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/appeals/{id} [patch]
func (h *Handler) adminResolveAppeal(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid appeal ID")
	}

	var cmd ResolveAppealCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AdminResolveAppeal(c.Request().Context(), auth, id, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminDashboard godoc
//
// @Summary     Get safety dashboard (admin)
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} DashboardStats
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/safety/dashboard [get]
func (h *Handler) adminDashboard(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	resp, err := h.svc.AdminDashboard(c.Request().Context(), auth)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Phase 2: Parental Controls ─────────────────────────────────────────────────

// getParentalControls godoc
//
// @Summary     Get parental controls
// @Tags        safety
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  ParentalControlResponse
// @Failure     401 {object} shared.AppError
// @Router      /safety/parental-controls [get]
func (h *Handler) getParentalControls(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.GetParentalControls(c.Request().Context(), scope)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// upsertParentalControl godoc
//
// @Summary     Create or update a parental control
// @Tags        safety
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body UpsertParentalControlCommand true "Control details"
// @Success     200 {object} ParentalControlResponse
// @Failure     401 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /safety/parental-controls [put]
func (h *Handler) upsertParentalControl(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var cmd UpsertParentalControlCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.UpsertParentalControl(c.Request().Context(), scope, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteParentalControl godoc
//
// @Summary     Delete a parental control
// @Tags        safety
// @Security    BearerAuth
// @Param       id path string true "Control ID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /safety/parental-controls/{id} [delete]
func (h *Handler) deleteParentalControl(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return shared.ErrUnauthorized()
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid control ID")
	}

	if err := h.svc.DeleteParentalControl(c.Request().Context(), scope, id); err != nil {
		return mapSafetyError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Phase 2: Admin Roles ───────────────────────────────────────────────────────

// adminListRoles godoc
//
// @Summary     List admin roles
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array}  AdminRoleResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/safety/roles [get]
func (h *Handler) adminListRoles(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	resp, err := h.svc.ListAdminRoles(c.Request().Context(), auth)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminCreateRole godoc
//
// @Summary     Create an admin role
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CreateAdminRoleCommand true "Role details"
// @Success     201 {object} AdminRoleResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /admin/safety/roles [post]
func (h *Handler) adminCreateRole(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	var cmd CreateAdminRoleCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.CreateAdminRole(c.Request().Context(), auth, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// adminAssignRole godoc
//
// @Summary     Assign an admin role to a parent
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       role_id path string                  true "Role ID"
// @Param       body    body AssignAdminRoleCommand   true "Assignment details"
// @Success     201 {object} AdminRoleAssignmentResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/roles/{role_id}/assign [post]
func (h *Handler) adminAssignRole(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid role ID")
	}

	var cmd AssignAdminRoleCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AssignAdminRole(c.Request().Context(), auth, roleID, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// adminRevokeRole godoc
//
// @Summary     Revoke an admin role assignment
// @Tags        safety-admin
// @Security    BearerAuth
// @Param       role_id   path string true "Role ID"
// @Param       parent_id path string true "Parent ID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/roles/{role_id}/assignments/{parent_id} [delete]
func (h *Handler) adminRevokeRole(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid role ID")
	}

	parentID, err := uuid.Parse(c.Param("parent_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid parent ID")
	}

	if err := h.svc.RevokeAdminRole(c.Request().Context(), auth, roleID, parentID); err != nil {
		return mapSafetyError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// adminListRoleAssignments godoc
//
// @Summary     List role assignments
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Param       role_id path string true "Role ID"
// @Success     200 {array}  AdminRoleAssignmentResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/roles/{role_id}/assignments [get]
func (h *Handler) adminListRoleAssignments(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	roleID, err := uuid.Parse(c.Param("role_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid role ID")
	}

	resp, err := h.svc.ListAdminRoleAssignments(c.Request().Context(), auth, roleID)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Phase 2: Grooming Scores ───────────────────────────────────────────────────

// adminListGroomingScores godoc
//
// @Summary     List grooming detection scores (admin)
// @Tags        safety-admin
// @Produce     json
// @Security    BearerAuth
// @Param       page  query int false "Page number"
// @Param       limit query int false "Results per page"
// @Success     200 {object} GroomingScoreListResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /admin/safety/grooming-scores [get]
func (h *Handler) adminListGroomingScores(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination")
	}

	resp, err := h.svc.AdminListGroomingScores(c.Request().Context(), auth, pagination)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// adminReviewGroomingScore godoc
//
// @Summary     Review a grooming detection score (admin)
// @Tags        safety-admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string                      true "Grooming score ID"
// @Param       body body ReviewGroomingScoreCommand   true "Review decision"
// @Success     200 {object} GroomingScoreResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /admin/safety/grooming-scores/{id} [patch]
func (h *Handler) adminReviewGroomingScore(c echo.Context) error {
	auth, err := requireAdmin(c)
	if err != nil {
		return err
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid grooming score ID")
	}

	var cmd ReviewGroomingScoreCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AdminReviewGroomingScore(c.Request().Context(), auth, id, cmd)
	if err != nil {
		return mapSafetyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Helpers ────────────────────────────────────────────────────────────────────

func requireAdmin(c echo.Context) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, shared.ErrUnauthorized()
	}
	if !auth.IsPlatformAdmin {
		return nil, shared.ErrForbidden()
	}
	return auth, nil
}

