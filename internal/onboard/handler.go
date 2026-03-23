package onboard

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler holds the onboard HTTP handler dependencies.
type Handler struct {
	svc OnboardingService
}

// NewHandler creates a new onboard Handler.
func NewHandler(svc OnboardingService) *Handler {
	return &Handler{svc: svc}
}

// Register registers all onboarding routes on the authenticated route group.
// All onboarding endpoints require authentication. [04-onboard §4]
func (h *Handler) Register(authGroup *echo.Group) {
	onb := authGroup.Group("/onboarding")
	onb.GET("/progress", h.getProgress)
	onb.PATCH("/family-profile", h.updateFamilyProfile)
	onb.POST("/children", h.addChild)
	onb.DELETE("/children/:id", h.removeChild)
	onb.PATCH("/methodology", h.selectMethodology)
	onb.POST("/methodology/import-quiz", h.importQuiz)
	onb.GET("/roadmap", h.getRoadmap)
	onb.GET("/recommendations", h.getRecommendations)
	onb.GET("/community", h.getCommunity)
	onb.POST("/complete", h.completeWizard)
	onb.POST("/skip", h.skipWizard)
}

// ─── Handlers ────────────────────────────────────────────────────────────────

// getProgress handles GET /v1/onboarding/progress.
//
// @Summary     Get onboarding wizard progress
// @Tags        onboarding
// @Produce     json
// @Success     200  {object}  WizardProgressResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Router      /onboarding/progress [get]
func (h *Handler) getProgress(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetProgress(c.Request().Context(), &scope)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateFamilyProfile handles PATCH /v1/onboarding/family-profile.
//
// @Summary     Update family profile during onboarding
// @Tags        onboarding
// @Accept      json
// @Produce     json
// @Param       body  body      UpdateFamilyProfileCommand  true  "Profile fields"
// @Success     200   {object}  WizardProgressResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     422   {object}  shared.ErrorResponse
// @Router      /onboarding/family-profile [patch]
func (h *Handler) updateFamilyProfile(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateFamilyProfileCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateFamilyProfile(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// addChild handles POST /v1/onboarding/children.
// Enforces COPPA consent: status must be "consented" or "re_verified". [04-onboard §9.2, iam §4.3]
//
// @Summary     Add child during onboarding
// @Tags        onboarding
// @Accept      json
// @Produce     json
// @Param       body  body      AddChildCommand  true  "Child details"
// @Success     201   {object}  WizardProgressResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     403   {object}  shared.ErrorResponse
// @Failure     422   {object}  shared.ErrorResponse
// @Router      /onboarding/children [post]
func (h *Handler) addChild(c echo.Context) error {
	// COPPA enforcement inline (same pattern as iam.Handler.createStudent)
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	switch auth.CoppaConsentStatus {
	case "consented", "re_verified":
		// Consent active — proceed.
	default:
		return shared.ErrCoppaConsentRequired()
	}

	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd AddChildCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.AddChild(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// removeChild handles DELETE /v1/onboarding/children/:id.
//
// @Summary     Remove child during onboarding
// @Tags        onboarding
// @Param       id  path  string  true  "Student ID"
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Router      /onboarding/children/{id} [delete]
func (h *Handler) removeChild(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student ID")
	}
	if err := h.svc.RemoveChild(c.Request().Context(), &scope, studentID); err != nil {
		return mapOnboardError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// selectMethodology handles PATCH /v1/onboarding/methodology.
//
// @Summary     Select methodology during onboarding
// @Tags        onboarding
// @Accept      json
// @Produce     json
// @Param       body  body      SelectMethodologyCommand  true  "Methodology selection"
// @Success     200   {object}  WizardProgressResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     422   {object}  shared.ErrorResponse
// @Router      /onboarding/methodology [patch]
func (h *Handler) selectMethodology(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd SelectMethodologyCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.SelectMethodology(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// importQuiz handles POST /v1/onboarding/methodology/import-quiz.
//
// @Summary     Import quiz result into onboarding
// @Tags        onboarding
// @Accept      json
// @Produce     json
// @Param       body  body      ImportQuizCommand  true  "Quiz share ID"
// @Success     200   {object}  QuizImportResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     404   {object}  shared.ErrorResponse
// @Failure     409   {object}  shared.ErrorResponse
// @Router      /onboarding/methodology/import-quiz [post]
func (h *Handler) importQuiz(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd ImportQuizCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.ImportQuiz(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getRoadmap handles GET /v1/onboarding/roadmap.
//
// @Summary     Get onboarding roadmap
// @Tags        onboarding
// @Produce     json
// @Success     200  {object}  RoadmapResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Router      /onboarding/roadmap [get]
func (h *Handler) getRoadmap(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetRoadmap(c.Request().Context(), &scope)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getRecommendations handles GET /v1/onboarding/recommendations.
//
// @Summary     Get onboarding starter recommendations
// @Tags        onboarding
// @Produce     json
// @Success     200  {object}  RecommendationsResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Router      /onboarding/recommendations [get]
func (h *Handler) getRecommendations(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetRecommendations(c.Request().Context(), &scope)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getCommunity handles GET /v1/onboarding/community.
//
// @Summary     Get onboarding community suggestions
// @Tags        onboarding
// @Produce     json
// @Success     200  {object}  CommunityResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Router      /onboarding/community [get]
func (h *Handler) getCommunity(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetCommunity(c.Request().Context(), &scope)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// completeWizard handles POST /v1/onboarding/complete.
//
// @Summary     Complete the onboarding wizard
// @Tags        onboarding
// @Produce     json
// @Success     200  {object}  WizardProgressResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     409  {object}  shared.ErrorResponse
// @Failure     422  {object}  shared.ErrorResponse
// @Router      /onboarding/complete [post]
func (h *Handler) completeWizard(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.CompleteWizard(c.Request().Context(), &scope)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// skipWizard handles POST /v1/onboarding/skip.
//
// @Summary     Skip the onboarding wizard
// @Tags        onboarding
// @Produce     json
// @Success     200  {object}  WizardProgressResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     409  {object}  shared.ErrorResponse
// @Router      /onboarding/skip [post]
func (h *Handler) skipWizard(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.SkipWizard(c.Request().Context(), &scope)
	if err != nil {
		return mapOnboardError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Error Mapping ───────────────────────────────────────────────────────────

// mapOnboardError converts onboard domain errors to shared.AppError HTTP responses.
// Internal error details are never exposed to the client. [CODING §2.2]
func mapOnboardError(err error) error {
	if err == nil {
		return nil
	}

	var onbErr *OnboardError
	if errors.As(err, &onbErr) {
		return onbErr.toAppError()
	}

	// Pass through AppError (already mapped, e.g. from shared package or cross-domain calls).
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Default: internal error — log internally, never expose details. [CODING §2.2]
	return shared.ErrInternal(err)
}
