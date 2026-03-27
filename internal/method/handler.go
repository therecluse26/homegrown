package method

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/method/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler holds the method HTTP handler dependencies.
type Handler struct {
	svc MethodologyService
}

// NewHandler creates a new method Handler.
func NewHandler(svc MethodologyService) *Handler {
	return &Handler{svc: svc}
}

// Register registers all method routes on the provided route groups.
//   - publicGroup: unauthenticated routes (methodologies are public data)
//   - authGroup: authenticated routes under /v1
func (h *Handler) Register(publicGroup, authGroup *echo.Group) {
	// Public routes — no auth required. [02-method §4.1]
	methodologies := publicGroup.Group("/methodologies")
	methodologies.GET("", h.listMethodologies)
	methodologies.GET("/:slug", h.getMethodology)
	methodologies.GET("/:slug/tools", h.getMethodologyTools)

	// Authenticated routes under /v1
	authGroup.GET("/families/tools", h.getFamilyTools)
	authGroup.GET("/families/students/:id/tools", h.getStudentTools)
	authGroup.PATCH("/families/methodology", h.updateFamilyMethodology)
	authGroup.GET("/families/methodology-context", h.getMethodologyContext)
	authGroup.PATCH("/families/students/:id/methodology", h.updateStudentMethodology)
}

// ─── Public Handlers ─────────────────────────────────────────────────────────

// listMethodologies handles GET /v1/methodologies.
//
// @Summary     List active methodologies
// @Tags        methodologies
// @Produce     json
// @Success     200  {array}  MethodologySummaryResponse
// @Router      /methodologies [get]
func (h *Handler) listMethodologies(c echo.Context) error {
	resp, err := h.svc.ListMethodologies(c.Request().Context())
	if err != nil {
		return mapMethodError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getMethodology handles GET /v1/methodologies/:slug.
//
// @Summary     Get methodology detail
// @Tags        methodologies
// @Produce     json
// @Param       slug  path     string  true  "Methodology slug"
// @Success     200   {object} MethodologyDetailResponse
// @Failure     404   {object} shared.ErrorResponse
// @Router      /methodologies/{slug} [get]
func (h *Handler) getMethodology(c echo.Context) error {
	slug := c.Param("slug")
	resp, err := h.svc.GetMethodology(c.Request().Context(), slug)
	if err != nil {
		return mapMethodError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getMethodologyTools handles GET /v1/methodologies/:slug/tools.
//
// @Summary     List tools for a methodology
// @Tags        methodologies
// @Produce     json
// @Param       slug  path     string  true  "Methodology slug"
// @Success     200   {array}  ActiveToolResponse
// @Failure     404   {object} shared.ErrorResponse
// @Router      /methodologies/{slug}/tools [get]
func (h *Handler) getMethodologyTools(c echo.Context) error {
	slug := c.Param("slug")
	resp, err := h.svc.GetMethodologyTools(c.Request().Context(), slug)
	if err != nil {
		return mapMethodError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Authenticated Handlers ──────────────────────────────────────────────────

// getFamilyTools handles GET /v1/families/tools.
//
// @Summary     Get family's resolved active tool set
// @Tags        families
// @Produce     json
// @Success     200  {array}  ActiveToolResponse
// @Failure     401  {object} shared.ErrorResponse
// @Router      /families/tools [get]
func (h *Handler) getFamilyTools(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.ResolveFamilyTools(c.Request().Context(), &scope)
	if err != nil {
		return mapMethodError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getStudentTools handles GET /v1/families/students/:id/tools.
//
// @Summary     Get student's resolved tool set
// @Tags        families
// @Produce     json
// @Param       id  path     string  true  "Student ID"
// @Success     200 {array}  ActiveToolResponse
// @Failure     401 {object} shared.ErrorResponse
// @Failure     404 {object} shared.ErrorResponse
// @Router      /families/students/{id}/tools [get]
func (h *Handler) getStudentTools(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student ID")
	}
	resp, err := h.svc.ResolveStudentTools(c.Request().Context(), &scope, studentID)
	if err != nil {
		return mapMethodError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateFamilyMethodology handles PATCH /v1/families/methodology.
//
// @Summary     Update family methodology selection
// @Tags        families
// @Accept      json
// @Produce     json
// @Param       body  body      UpdateMethodologyCommand  true  "Methodology selection"
// @Success     200   {object}  MethodologySelectionResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     422   {object}  shared.ErrorResponse
// @Router      /families/methodology [patch]
func (h *Handler) updateFamilyMethodology(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateMethodologyCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateFamilyMethodology(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapMethodError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getMethodologyContext handles GET /v1/families/methodology-context.
//
// @Summary     Get full methodology context for the family dashboard
// @Tags        families
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} MethodologyContext
// @Failure     401 {object} shared.AppError
// @Router      /families/methodology-context [get]
func (h *Handler) getMethodologyContext(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetMethodologyContext(c.Request().Context(), &scope)
	if err != nil {
		return mapMethodError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateStudentMethodology handles PATCH /v1/families/students/:id/methodology.
//
// @Summary     Set or clear a student's methodology override
// @Tags        families
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path     string                       true  "Student ID"
// @Param       body body     UpdateStudentMethodologyCommand true "Methodology override"
// @Success     200  {object} MethodologySelectionResponse
// @Failure     400  {object} shared.AppError
// @Failure     401  {object} shared.AppError
// @Failure     404  {object} shared.AppError
// @Router      /families/students/{id}/methodology [patch]
func (h *Handler) updateStudentMethodology(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student ID")
	}
	var cmd UpdateStudentMethodologyCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	resp, err := h.svc.UpdateStudentMethodology(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapMethodError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Error Mapping ────────────────────────────────────────────────────────────

// mapMethodError converts method domain errors to shared.AppError HTTP responses. [02-method §10.4]
// Internal error details are never exposed to the client. [CODING §2.2]
func mapMethodError(err error) error {
	if err == nil {
		return nil
	}

	// Check for MethodError (wraps sentinel errors with context)
	var methodErr *domain.MethodError
	if errors.As(err, &methodErr) {
		switch {
		case errors.Is(methodErr.Err, domain.ErrMethodologyNotFound):
			return &shared.AppError{Code: "methodology_not_found", Message: "Methodology not found", StatusCode: http.StatusNotFound}
		case errors.Is(methodErr.Err, domain.ErrMethodologyNotActive):
			return &shared.AppError{Code: "methodology_not_active", Message: "Methodology is not active", StatusCode: http.StatusUnprocessableEntity}
		case errors.Is(methodErr.Err, domain.ErrInvalidMethodologyIDs):
			return &shared.AppError{Code: "invalid_methodology_ids", Message: "One or more methodology slugs are invalid", StatusCode: http.StatusUnprocessableEntity}
		case errors.Is(methodErr.Err, domain.ErrPrimaryInSecondary):
			return &shared.AppError{Code: "primary_in_secondary", Message: "Primary methodology cannot also be a secondary", StatusCode: http.StatusUnprocessableEntity}
		case errors.Is(methodErr.Err, domain.ErrDuplicateSecondary):
			return &shared.AppError{Code: "duplicate_secondary", Message: "Duplicate secondary methodology IDs", StatusCode: http.StatusUnprocessableEntity}
		case errors.Is(methodErr.Err, domain.ErrStudentNotFound):
			return &shared.AppError{Code: "student_not_found", Message: "Student not found", StatusCode: http.StatusNotFound}
		case errors.Is(methodErr.Err, domain.ErrToolNotFound):
			return &shared.AppError{Code: "tool_not_found", Message: "Tool not found", StatusCode: http.StatusNotFound}
		}
	}

	// Pass through AppError (already mapped, e.g. from shared or iam packages)
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Default: internal error — log internally, never expose details. [CODING §2.2]
	return shared.ErrInternal(err)
}
