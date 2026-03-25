package recs

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/middleware"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler holds the recs HTTP handler dependencies.
type Handler struct {
	svc RecsService
}

// NewHandler creates a new recs Handler.
func NewHandler(svc RecsService) *Handler {
	return &Handler{svc: svc}
}

// Register registers all recommendation routes on the authenticated route group.
// All endpoints require RequirePremium — 402 for free-tier families. [13-recs §4]
func (h *Handler) Register(authGroup *echo.Group) {
	recs := authGroup.Group("/recommendations")
	recs.GET("", h.getRecommendations)
	recs.GET("/students/:student_id", h.getStudentRecommendations)
	recs.POST("/:id/dismiss", h.dismissRecommendation)
	recs.POST("/:id/block", h.blockRecommendation)
	recs.DELETE("/:id/feedback", h.undoFeedback)
	recs.GET("/preferences", h.getPreferences)
	recs.PATCH("/preferences", h.updatePreferences)
}

// getRecommendations godoc
//
// @Summary     Get family recommendations
// @Tags        recommendations
// @Produce     json
// @Security    BearerAuth
// @Param       type   query string false "Filter by type" Enums(marketplace_content,activity_idea,reading_suggestion,community_group)
// @Param       cursor query string false "Pagination cursor"
// @Param       limit  query int    false "Results per page (default 20, max 50)"
// @Success     200 {object} RecommendationListResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /recommendations [get]
func (h *Handler) getRecommendations(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var params RecommendationListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}

	resp, err := h.svc.GetRecommendations(c.Request().Context(), &scope, params)
	if err != nil {
		return mapRecsError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getStudentRecommendations godoc
//
// @Summary     Get recommendations for a specific student
// @Tags        recommendations
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path  string  true  "Student UUID"
// @Param       type       query string  false "Filter by type"
// @Param       cursor     query string  false "Pagination cursor"
// @Param       limit      query int     false "Results per page (default 20, max 50)"
// @Success     200 {object} RecommendationListResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /recommendations/students/{student_id} [get]
func (h *Handler) getStudentRecommendations(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var params StudentRecommendationParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	params.StudentID = studentID

	resp, err := h.svc.GetStudentRecommendations(c.Request().Context(), &scope, params)
	if err != nil {
		return mapRecsError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// dismissRecommendation godoc
//
// @Summary     Dismiss a recommendation
// @Tags        recommendations
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Recommendation UUID"
// @Success     200 {object} map[string]string
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /recommendations/{id}/dismiss [post]
func (h *Handler) dismissRecommendation(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid recommendation id")
	}

	if err := h.svc.DismissRecommendation(c.Request().Context(), &scope, id); err != nil {
		return mapRecsError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "dismissed"})
}

// blockRecommendation godoc
//
// @Summary     Block a recommendation's source entity
// @Tags        recommendations
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Recommendation UUID"
// @Success     200 {object} map[string]string
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /recommendations/{id}/block [post]
func (h *Handler) blockRecommendation(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid recommendation id")
	}

	blockedEntityID, err := h.svc.BlockRecommendation(c.Request().Context(), &scope, id)
	if err != nil {
		return mapRecsError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{
		"status":            "blocked",
		"blocked_entity_id": blockedEntityID.String(),
	})
}

// undoFeedback godoc
//
// @Summary     Undo a dismiss or block action
// @Tags        recommendations
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Recommendation UUID"
// @Success     200 {object} map[string]string
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /recommendations/{id}/feedback [delete]
func (h *Handler) undoFeedback(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid recommendation id")
	}

	if err := h.svc.UndoFeedback(c.Request().Context(), &scope, id); err != nil {
		return mapRecsError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "active"})
}

// getPreferences godoc
//
// @Summary     Get family recommendation preferences
// @Tags        recommendations
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} RecommendationPreferencesResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /recommendations/preferences [get]
func (h *Handler) getPreferences(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.GetPreferences(c.Request().Context(), &scope)
	if err != nil {
		return mapRecsError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updatePreferences godoc
//
// @Summary     Update family recommendation preferences
// @Tags        recommendations
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body UpdatePreferencesCommand true "Partial preferences update"
// @Success     200 {object} RecommendationPreferencesResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /recommendations/preferences [patch]
func (h *Handler) updatePreferences(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var cmd UpdatePreferencesCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}

	resp, err := h.svc.UpdatePreferences(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapRecsError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Error Mapping ────────────────────────────────────────────────────────────

// mapRecsError maps domain errors to HTTP-appropriate AppErrors. [13-recs §15, CODING §2.2]
func mapRecsError(err error) error {
	switch {
	case errors.Is(err, ErrRecommendationNotFound):
		return &shared.AppError{Code: "not_found", Message: "Recommendation not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrStudentNotFound):
		return &shared.AppError{Code: "not_found", Message: "Student not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrFeedbackNotFound):
		return &shared.AppError{Code: "not_found", Message: "No feedback found for this recommendation", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrAlreadyHasFeedback):
		return &shared.AppError{Code: "conflict", Message: "Recommendation already dismissed or blocked", StatusCode: http.StatusConflict}
	case errors.Is(err, ErrInvalidRecommendationType):
		return shared.ErrValidation("Invalid recommendation type")
	case errors.Is(err, ErrInvalidExplorationFrequency):
		return shared.ErrValidation("Invalid exploration frequency")
	case errors.Is(err, ErrSignalRecordingFailed), errors.Is(err, ErrDatabaseError), errors.Is(err, ErrInternalError):
		// Never expose internal error details. [CODING §3.1]
		return shared.ErrInternal(err)
	default:
		return shared.ErrInternal(err)
	}
}
