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
	g.GET("/appeals/:id", h.getMyAppeal)

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
}

// ─── User-Facing Endpoints ──────────────────────────────────────────────────────

// POST /v1/safety/reports
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

// GET /v1/safety/reports
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

// GET /v1/safety/reports/:id
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

// GET /v1/safety/account-status
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

// POST /v1/safety/appeals
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

// GET /v1/safety/appeals/:id
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

// ─── Admin Endpoints ────────────────────────────────────────────────────────────

// GET /v1/admin/safety/reports
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

// GET /v1/admin/safety/reports/:id
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

// PATCH /v1/admin/safety/reports/:id
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

// GET /v1/admin/safety/flags
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

// PATCH /v1/admin/safety/flags/:id
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

// PATCH /v1/admin/safety/flags/:id/escalate-csam
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

// POST /v1/admin/safety/actions
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

// GET /v1/admin/safety/actions
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

// GET /v1/admin/safety/accounts/:family_id
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

// POST /v1/admin/safety/accounts/:family_id/suspend
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

// POST /v1/admin/safety/accounts/:family_id/ban
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

// POST /v1/admin/safety/accounts/:family_id/lift
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

// GET /v1/admin/safety/appeals
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

// PATCH /v1/admin/safety/appeals/:id
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

// GET /v1/admin/safety/dashboard
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

