package learner_profile

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler provides HTTP route handlers for the learner profile domain.
// [18-learner-profile §4]
type Handler struct {
	svc LearnerProfileService
}

// NewHandler creates a new learner profile handler.
func NewHandler(svc LearnerProfileService) *Handler {
	return &Handler{svc: svc}
}

// Register registers learner profile routes on the authenticated group.
// Routes follow the student-scoped pattern to enforce ownership checking. [18-learner-profile §4]
func (h *Handler) Register(auth *echo.Group) {
	g := auth.Group("/students/:student_id/learner-profile")
	g.POST("/submissions", h.submitProfile)
	g.GET("", h.getProfile)
}

// submitProfile godoc
//
//	@Summary     Submit learner profile quiz
//	@Description Processes a quiz submission and upserts the learner profile for a student.
//	@Tags        learner-profile
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       student_id path  string              true "Student UUID"
//	@Param       body       body  SubmitProfileCommand true "Quiz answers"
//	@Success     200 {object} LearnerProfileResponse
//	@Failure     400 {object} shared.AppError
//	@Failure     403 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Router      /students/{student_id}/learner-profile/submissions [post]
func (h *Handler) submitProfile(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return &shared.AppError{Code: "invalid_param", Message: "invalid student_id", StatusCode: http.StatusBadRequest}
	}

	var cmd SubmitProfileCommand
	if err := c.Bind(&cmd); err != nil {
		return &shared.AppError{Code: "invalid_body", Message: "invalid request body", StatusCode: http.StatusBadRequest}
	}
	if err := c.Validate(&cmd); err != nil {
		return err
	}

	resp, err := h.svc.SubmitProfile(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getProfile godoc
//
//	@Summary     Get learner profile
//	@Description Returns the current learner profile for a student. 404 if no quiz has been submitted.
//	@Tags        learner-profile
//	@Produce     json
//	@Security    BearerAuth
//	@Param       student_id path string true "Student UUID"
//	@Success     200 {object} LearnerProfileResponse
//	@Failure     403 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Router      /students/{student_id}/learner-profile [get]
func (h *Handler) getProfile(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return &shared.AppError{Code: "invalid_param", Message: "invalid student_id", StatusCode: http.StatusBadRequest}
	}

	resp, err := h.svc.GetProfile(c.Request().Context(), &scope, studentID)
	if err != nil {
		return mapError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// mapError converts domain errors to AppError for consistent HTTP responses.
// Never exposes internal error details per CODING_STANDARDS. [CODING §2.2]
func mapError(err error) error {
	var ae *shared.AppError
	if errors.As(err, &ae) {
		return ae
	}
	return shared.ErrInternal(err)
}
