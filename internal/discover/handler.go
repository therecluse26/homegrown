package discover

import (
	"errors"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// DiscoverHandler holds the discovery HTTP handler dependencies.
type DiscoverHandler struct {
	svc DiscoveryService
}

// NewHandler creates a new DiscoverHandler.
func NewHandler(svc DiscoveryService) *DiscoverHandler {
	return &DiscoverHandler{svc: svc}
}

// Register registers all discovery routes on the public route group.
// All discovery endpoints are unauthenticated. [03-discover §1, §4.1]
func (h *DiscoverHandler) Register(publicGroup *echo.Group) {
	disc := publicGroup.Group("/discovery")
	disc.GET("/quiz", h.getQuiz)
	disc.POST("/quiz/results", h.submitQuiz)
	disc.GET("/quiz/results/:share_id", h.getQuizResult)
	disc.GET("/state-guides", h.listStateGuides)
	disc.GET("/state-guides/:state_code", h.getStateGuide)
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// getQuiz handles GET /v1/discovery/quiz.
//
// @Summary     Get active quiz
// @Tags        discovery
// @Produce     json
// @Success     200  {object} QuizResponse
// @Failure     404  {object} shared.ErrorResponse
// @Router      /discovery/quiz [get]
func (h *DiscoverHandler) getQuiz(c echo.Context) error {
	resp, err := h.svc.GetActiveQuiz(c.Request().Context())
	if err != nil {
		return mapDiscoverError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// submitQuiz handles POST /v1/discovery/quiz/results.
//
// @Summary     Submit quiz answers and get methodology recommendations
// @Tags        discovery
// @Accept      json
// @Produce     json
// @Param       body body SubmitQuizCommand true "Quiz answers"
// @Success     201  {object} QuizResultResponse
// @Failure     404  {object} shared.ErrorResponse
// @Failure     422  {object} shared.ErrorResponse
// @Router      /discovery/quiz/results [post]
func (h *DiscoverHandler) submitQuiz(c echo.Context) error {
	var cmd SubmitQuizCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.SubmitQuiz(c.Request().Context(), cmd)
	if err != nil {
		return mapDiscoverError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// getQuizResult handles GET /v1/discovery/quiz/results/:share_id.
//
// @Summary     Get quiz result by share ID
// @Tags        discovery
// @Produce     json
// @Param       share_id path string true "Share ID"
// @Success     200  {object} QuizResultResponse
// @Failure     404  {object} shared.ErrorResponse
// @Router      /discovery/quiz/results/{share_id} [get]
func (h *DiscoverHandler) getQuizResult(c echo.Context) error {
	shareID := c.Param("share_id")
	resp, err := h.svc.GetQuizResult(c.Request().Context(), shareID)
	if err != nil {
		return mapDiscoverError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// listStateGuides handles GET /v1/discovery/state-guides.
//
// @Summary     List all state guides
// @Tags        discovery
// @Produce     json
// @Success     200  {array}  StateGuideSummaryResponse
// @Router      /discovery/state-guides [get]
func (h *DiscoverHandler) listStateGuides(c echo.Context) error {
	resp, err := h.svc.ListStateGuides(c.Request().Context())
	if err != nil {
		return mapDiscoverError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getStateGuide handles GET /v1/discovery/state-guides/:state_code.
//
// @Summary     Get state guide by state code
// @Tags        discovery
// @Produce     json
// @Param       state_code path string true "Two-letter state code (e.g. CA, NY)"
// @Success     200  {object} StateGuideResponse
// @Failure     404  {object} shared.ErrorResponse
// @Router      /discovery/state-guides/{state_code} [get]
func (h *DiscoverHandler) getStateGuide(c echo.Context) error {
	stateCode := c.Param("state_code")
	resp, err := h.svc.GetStateGuide(c.Request().Context(), stateCode)
	if err != nil {
		return mapDiscoverError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Error Mapping ────────────────────────────────────────────────────────────

// mapDiscoverError converts discover domain errors to shared.AppError HTTP responses.
// Internal error details are never exposed to the client. [CODING §2.2]
func mapDiscoverError(err error) error {
	if err == nil {
		return nil
	}

	var discErr *DiscoverError
	if errors.As(err, &discErr) {
		return discErr.toAppError()
	}

	// Pass through AppError (already mapped, e.g. from shared package).
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Default: internal error — log internally, never expose details. [CODING §2.2]
	return shared.ErrInternal(err)
}
