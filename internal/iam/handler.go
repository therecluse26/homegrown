package iam

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler holds the IAM HTTP handler dependencies.
type Handler struct {
	svc           IamService
	webhookSecret string
}

// NewHandler creates a new IAM Handler.
func NewHandler(svc IamService, webhookSecret string) *Handler {
	return &Handler{svc: svc, webhookSecret: webhookSecret}
}

// Register registers all IAM routes on the provided route groups.
//   - authGroup: authenticated routes under /v1
//   - hooksGroup: webhook routes under /hooks (rate-limited, no auth middleware)
func (h *Handler) Register(authGroup, hooksGroup *echo.Group) {
	// Webhook routes — validated by shared secret, NOT by session cookie. [§4.1]
	hooks := hooksGroup.Group("/kratos", h.webhookSecretMiddleware())
	hooks.POST("/post-registration", h.handlePostRegistration)
	hooks.POST("/post-login", h.handlePostLogin)

	// Authenticated routes under /v1 — all require session cookie via Auth middleware.
	authGroup.GET("/auth/me", h.getMe)

	families := authGroup.Group("/families")
	families.GET("/profile", h.getFamilyProfile)
	families.PATCH("/profile", h.updateFamilyProfile)
	families.GET("/consent", h.getConsentStatus)
	families.POST("/consent", h.submitConsent)

	students := families.Group("/students")
	students.POST("", h.createStudent)
	students.GET("", h.listStudents)
	students.PATCH("/:id", h.updateStudent)
	students.DELETE("/:id", h.deleteStudent)
}

// ─── Webhook Handlers ─────────────────────────────────────────────────────────

// handlePostRegistration handles POST /hooks/kratos/post-registration.
//
// @Summary     Kratos post-registration webhook
// @Tags        hooks
// @Accept      json
// @Produce     json
// @Success     200
// @Failure     400  {object}  shared.ErrorResponse
// @Failure     500  {object}  shared.ErrorResponse
// @Router      /hooks/kratos/post-registration [post]
func (h *Handler) handlePostRegistration(c echo.Context) error {
	var payload KratosWebhookPayload
	if err := c.Bind(&payload); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&payload); err != nil {
		return shared.ValidationError(err)
	}
	if err := h.svc.HandlePostRegistration(c.Request().Context(), payload); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusOK)
}

// handlePostLogin handles POST /hooks/kratos/post-login.
//
// @Summary     Kratos post-login webhook
// @Tags        hooks
// @Accept      json
// @Produce     json
// @Success     200
// @Failure     400  {object}  shared.ErrorResponse
// @Router      /hooks/kratos/post-login [post]
func (h *Handler) handlePostLogin(c echo.Context) error {
	var payload KratosWebhookPayload
	if err := c.Bind(&payload); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := h.svc.HandlePostLogin(c.Request().Context(), payload); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusOK)
}

// ─── Auth Handlers ────────────────────────────────────────────────────────────

// getMe handles GET /v1/auth/me.
//
// @Summary     Get current user
// @Tags        auth
// @Produce     json
// @Success     200  {object}  CurrentUserResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Router      /auth/me [get]
func (h *Handler) getMe(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetCurrentUser(c.Request().Context(), auth)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Family Profile Handlers ──────────────────────────────────────────────────

// getFamilyProfile handles GET /v1/families/profile.
//
// @Summary     Get family profile
// @Tags        families
// @Produce     json
// @Success     200  {object}  FamilyProfileResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Router      /families/profile [get]
func (h *Handler) getFamilyProfile(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetFamilyProfile(c.Request().Context(), &scope)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateFamilyProfile handles PATCH /v1/families/profile.
//
// @Summary     Update family profile
// @Tags        families
// @Accept      json
// @Produce     json
// @Param       body  body      UpdateFamilyCommand  true  "Fields to update"
// @Success     200   {object}  FamilyProfileResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     422   {object}  shared.ErrorResponse
// @Router      /families/profile [patch]
func (h *Handler) updateFamilyProfile(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateFamilyCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateFamilyProfile(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Consent Handlers ─────────────────────────────────────────────────────────

// getConsentStatus handles GET /v1/families/consent.
//
// @Summary     Get COPPA consent status
// @Tags        families
// @Produce     json
// @Success     200  {object}  ConsentStatusResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Router      /families/consent [get]
func (h *Handler) getConsentStatus(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetConsentStatus(c.Request().Context(), &scope)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// submitConsent handles POST /v1/families/consent.
//
// @Summary     Submit COPPA consent
// @Tags        families
// @Accept      json
// @Produce     json
// @Param       body  body      CoppaConsentCommand  true  "Consent details"
// @Success     200   {object}  ConsentStatusResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     422   {object}  shared.ErrorResponse
// @Router      /families/consent [post]
func (h *Handler) submitConsent(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	var cmd CoppaConsentCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.SubmitCoppaConsent(c.Request().Context(), &scope, auth, cmd)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Student Handlers ─────────────────────────────────────────────────────────

// createStudent handles POST /v1/families/students.
// Enforces COPPA consent: status must be "consented" or "re_verified". [§4.3, §9]
//
// @Summary     Create student profile
// @Tags        students
// @Accept      json
// @Produce     json
// @Param       body  body      CreateStudentCommand  true  "Student details"
// @Success     201   {object}  StudentResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     403   {object}  shared.ErrorResponse
// @Failure     422   {object}  shared.ErrorResponse
// @Router      /families/students [post]
func (h *Handler) createStudent(c echo.Context) error {
	// Enforce COPPA consent before student creation. AuthContext is populated by auth middleware
	// with coppa_consent_status from iam_families — no extra DB query needed. [§11.2]
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
	var cmd CreateStudentCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateStudent(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listStudents handles GET /v1/families/students.
//
// @Summary     List family students
// @Tags        students
// @Produce     json
// @Success     200  {array}  StudentResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Router      /families/students [get]
func (h *Handler) listStudents(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.ListStudents(c.Request().Context(), &scope)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateStudent handles PATCH /v1/families/students/:id.
//
// @Summary     Update student profile
// @Tags        students
// @Accept      json
// @Produce     json
// @Param       id    path      string                true  "Student ID"
// @Param       body  body      UpdateStudentCommand  true  "Fields to update"
// @Success     200   {object}  StudentResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     404   {object}  shared.ErrorResponse
// @Failure     422   {object}  shared.ErrorResponse
// @Router      /families/students/{id} [patch]
func (h *Handler) updateStudent(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student ID")
	}
	var cmd UpdateStudentCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateStudent(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteStudent handles DELETE /v1/families/students/:id.
//
// @Summary     Delete student profile
// @Tags        students
// @Param       id  path  string  true  "Student ID"
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Router      /families/students/{id} [delete]
func (h *Handler) deleteStudent(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	studentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student ID")
	}
	if err := h.svc.DeleteStudent(c.Request().Context(), &scope, studentID); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Middleware ───────────────────────────────────────────────────────────────

// webhookSecretMiddleware validates the X-Webhook-Secret header against the configured secret.
// Applied to all webhook routes. [§4.1, §15.14]
func (h *Handler) webhookSecretMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			secret := c.Request().Header.Get("X-Webhook-Secret")
			if secret != h.webhookSecret {
				return shared.ErrUnauthorized()
			}
			return next(c)
		}
	}
}

// ─── Error Mapping ────────────────────────────────────────────────────────────

// mapIamError converts IAM sentinel errors to shared.AppError HTTP responses. [§12.1]
// Internal error details are never exposed to the client. [CODING §2.2]
func mapIamError(err error) error {
	if err == nil {
		return nil
	}

	// Check for structured errors first.
	var consentErr *InvalidConsentTransitionError
	if errors.As(err, &consentErr) {
		return &shared.AppError{
			Code:       "invalid_consent_transition",
			Message:    "Invalid COPPA consent transition",
			StatusCode: http.StatusUnprocessableEntity,
		}
	}

	// Check sentinel errors.
	switch {
	case errors.Is(err, ErrFamilyNotFound):
		return &shared.AppError{Code: "family_not_found", Message: "Family not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrParentNotFound):
		return &shared.AppError{Code: "parent_not_found", Message: "Parent not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrStudentNotFound):
		return &shared.AppError{Code: "student_not_found", Message: "Student not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrInviteNotFound):
		return &shared.AppError{Code: "invite_not_found", Message: "Invite not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrNoPendingDeletion):
		return &shared.AppError{Code: "no_pending_deletion", Message: "No pending deletion request", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrInviteExpired):
		return &shared.AppError{Code: "invite_expired", Message: "Invite has expired", StatusCode: http.StatusGone}
	case errors.Is(err, ErrInviteAlreadyAccepted):
		return &shared.AppError{Code: "invite_already_accepted", Message: "Invite already accepted", StatusCode: http.StatusConflict}
	case errors.Is(err, ErrParentAlreadyInFamily):
		return &shared.AppError{Code: "parent_already_in_family", Message: "Parent already in this family", StatusCode: http.StatusConflict}
	case errors.Is(err, ErrEmailAlreadyAssociated):
		return &shared.AppError{Code: "email_already_associated", Message: "Email already associated with a family", StatusCode: http.StatusConflict}
	case errors.Is(err, ErrDeletionAlreadyRequested):
		return &shared.AppError{Code: "deletion_already_requested", Message: "Deletion already requested", StatusCode: http.StatusConflict}
	case errors.Is(err, ErrConsentVerificationFailed):
		return &shared.AppError{Code: "consent_verification_failed", Message: "Consent verification failed", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(err, ErrCannotRemovePrimaryParent):
		return &shared.AppError{Code: "cannot_remove_primary_parent", Message: "Cannot remove primary parent", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(err, ErrCannotTransferToSelf):
		return &shared.AppError{Code: "cannot_transfer_to_self", Message: "Cannot transfer primary role to self", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(err, ErrCoppaConsentRequired):
		return &shared.AppError{Code: "coppa_consent_required", Message: "COPPA parental consent required", StatusCode: http.StatusForbidden}
	case errors.Is(err, ErrNotPrimaryParent):
		return &shared.AppError{Code: "not_primary_parent", Message: "Only the primary parent can perform this action", StatusCode: http.StatusForbidden}
	case errors.Is(err, ErrPremiumRequired):
		return &shared.AppError{Code: "premium_required", Message: "Premium subscription required", StatusCode: http.StatusPaymentRequired}
	case errors.Is(err, ErrKratosError):
		return &shared.AppError{Code: "auth_service_unavailable", Message: "Authentication service temporarily unavailable", StatusCode: http.StatusBadGateway}
	}

	// Pass through AppError (already mapped, e.g. from shared package).
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Default: internal error — log internally, never expose details. [CODING §2.2]
	return shared.ErrInternal(err)
}
