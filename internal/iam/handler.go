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
//   - rootGroup: unauthenticated routes under /v1 (student session endpoint)
func (h *Handler) Register(authGroup, hooksGroup *echo.Group) {
	h.RegisterWithStudentSession(authGroup, hooksGroup, nil)
}

// RegisterWithStudentSession registers all IAM routes including the student-session
// endpoint which requires a separate root group (no auth middleware). [§5]
func (h *Handler) RegisterWithStudentSession(authGroup, hooksGroup, rootGroup *echo.Group) {
	// Webhook routes — validated by shared secret, NOT by session cookie. [§4.1]
	hooks := hooksGroup.Group("/kratos", h.webhookSecretMiddleware())
	hooks.POST("/post-registration", h.handlePostRegistration)
	hooks.POST("/post-login", h.handlePostLogin)

	// Authenticated routes under /v1 — all require session cookie via Auth middleware.
	authGroup.GET("/auth/me", h.getMe)
	authGroup.GET("/auth/mfa/status", h.getMfaStatus)

	families := authGroup.Group("/families")
	families.GET("/profile", h.getFamilyProfile)
	families.PATCH("/profile", h.updateFamilyProfile)
	families.GET("/consent", h.getConsentStatus)
	families.POST("/consent", h.submitConsent)

	// Phase 2: co-parent management.
	families.POST("/invites", h.inviteCoParent)
	families.DELETE("/invites/:id", h.cancelInvite)
	families.POST("/invites/:token/accept", h.acceptInvite)
	families.DELETE("/parents/:id", h.removeCoParent)
	families.POST("/primary-parent", h.transferPrimaryParent)
	families.DELETE("/consent", h.withdrawCoppaConsent)
	families.POST("/deletion-request", h.requestFamilyDeletion)
	families.DELETE("/deletion-request", h.cancelFamilyDeletion)

	students := families.Group("/students")
	students.POST("", h.createStudent)
	students.GET("", h.listStudents)
	students.PATCH("/:id", h.updateStudent)
	students.DELETE("/:id", h.deleteStudent)

	// Phase 2: student sessions.
	students.POST("/:student_id/sessions", h.createStudentSession)
	students.GET("/:student_id/sessions", h.listStudentSessions)
	students.DELETE("/:student_id/sessions/:id", h.revokeStudentSession)

	// Student session identity endpoint — uses bearer token, not session cookie.
	if rootGroup != nil {
		rootGroup.GET("/student-session/me", h.getStudentSessionMe, h.requireStudentSessionMiddleware())
	}
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
	if err := c.Validate(&payload); err != nil {
		return shared.ValidationError(err)
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

// ─── Phase 2: Co-parent Handlers ─────────────────────────────────────────────

// inviteCoParent handles POST /v1/families/invites.
// Requires primary parent. [§5]
//
// @Summary     Invite a co-parent
// @Tags        families
// @Accept      json
// @Produce     json
// @Param       body  body      InviteCoParentCommand  true  "Invite details"
// @Success     201   {object}  CoParentInviteResponse
// @Failure     401   {object}  shared.ErrorResponse
// @Failure     403   {object}  shared.ErrorResponse
// @Failure     422   {object}  shared.ErrorResponse
// @Router      /families/invites [post]
func (h *Handler) inviteCoParent(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd InviteCoParentCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.InviteCoParent(c.Request().Context(), &scope, auth, cmd)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// cancelInvite handles DELETE /v1/families/invites/:id.
// Requires primary parent. [§5]
//
// @Summary     Cancel a co-parent invite
// @Tags        families
// @Param       id  path  string  true  "Invite ID"
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     403  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Router      /families/invites/{id} [delete]
func (h *Handler) cancelInvite(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	if !auth.IsPrimaryParent {
		return &shared.AppError{Code: "not_primary_parent", Message: "Only the primary parent can perform this action", StatusCode: http.StatusForbidden}
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	inviteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid invite ID")
	}
	if err := h.svc.CancelInvite(c.Request().Context(), &scope, inviteID); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// acceptInvite handles POST /v1/families/invites/:token/accept.
// Requires auth (any authenticated parent). [§5]
//
// @Summary     Accept a co-parent invite
// @Tags        families
// @Param       token  path  string  true  "Invite token"
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Failure     409  {object}  shared.ErrorResponse
// @Failure     410  {object}  shared.ErrorResponse
// @Router      /families/invites/{token}/accept [post]
func (h *Handler) acceptInvite(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	token := c.Param("token")
	if token == "" {
		return shared.ErrBadRequest("invalid invite token")
	}
	if err := h.svc.AcceptInvite(c.Request().Context(), auth, token); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// removeCoParent handles DELETE /v1/families/parents/:id.
// Requires primary parent. [§5]
//
// @Summary     Remove a co-parent
// @Tags        families
// @Param       id  path  string  true  "Parent ID"
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     403  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Failure     422  {object}  shared.ErrorResponse
// @Router      /families/parents/{id} [delete]
func (h *Handler) removeCoParent(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	parentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid parent ID")
	}
	if err := h.svc.RemoveCoParent(c.Request().Context(), &scope, auth, parentID); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// transferPrimaryParent handles POST /v1/families/primary-parent.
// Requires primary parent. [§5]
//
// @Summary     Transfer primary parent role
// @Tags        families
// @Accept      json
// @Param       body  body  TransferPrimaryCommand  true  "New primary parent ID"
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     403  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Failure     422  {object}  shared.ErrorResponse
// @Router      /families/primary-parent [post]
func (h *Handler) transferPrimaryParent(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd TransferPrimaryCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	if err := h.svc.TransferPrimaryParent(c.Request().Context(), &scope, auth, cmd); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// withdrawCoppaConsent handles DELETE /v1/families/consent.
// Requires primary parent. [§5, §9.2]
//
// @Summary     Withdraw COPPA parental consent
// @Tags        families
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     403  {object}  shared.ErrorResponse
// @Failure     422  {object}  shared.ErrorResponse
// @Router      /families/consent [delete]
func (h *Handler) withdrawCoppaConsent(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.WithdrawCoppaConsent(c.Request().Context(), &scope, auth); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// requestFamilyDeletion handles POST /v1/families/deletion-request.
// Requires primary parent. [§5]
//
// @Summary     Request family account deletion
// @Tags        families
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     403  {object}  shared.ErrorResponse
// @Failure     409  {object}  shared.ErrorResponse
// @Router      /families/deletion-request [post]
func (h *Handler) requestFamilyDeletion(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.RequestFamilyDeletion(c.Request().Context(), &scope, auth); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// cancelFamilyDeletion handles DELETE /v1/families/deletion-request.
// Requires primary parent. [§5]
//
// @Summary     Cancel pending family deletion request
// @Tags        families
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Router      /families/deletion-request [delete]
func (h *Handler) cancelFamilyDeletion(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.CancelFamilyDeletion(c.Request().Context(), &scope); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Phase 2: Student Session Handlers ───────────────────────────────────────

// createStudentSession handles POST /v1/families/students/:student_id/sessions.
//
// @Summary     Create a student session token
// @Tags        students
// @Accept      json
// @Produce     json
// @Param       student_id  path      string                       true  "Student ID"
// @Param       body        body      CreateStudentSessionCommand  true  "Session details"
// @Success     201         {object}  StudentSessionResponse
// @Failure     401         {object}  shared.ErrorResponse
// @Failure     404         {object}  shared.ErrorResponse
// @Failure     422         {object}  shared.ErrorResponse
// @Router      /families/students/{student_id}/sessions [post]
func (h *Handler) createStudentSession(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student ID")
	}
	var cmd CreateStudentSessionCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateStudentSession(c.Request().Context(), &scope, auth, studentID, cmd)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listStudentSessions handles GET /v1/families/students/:student_id/sessions.
//
// @Summary     List active student sessions
// @Tags        students
// @Produce     json
// @Param       student_id  path      string  true  "Student ID"
// @Success     200         {array}   StudentSessionSummaryResponse
// @Failure     401         {object}  shared.ErrorResponse
// @Failure     404         {object}  shared.ErrorResponse
// @Router      /families/students/{student_id}/sessions [get]
func (h *Handler) listStudentSessions(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student ID")
	}
	resp, err := h.svc.ListStudentSessions(c.Request().Context(), &scope, studentID)
	if err != nil {
		return mapIamError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// revokeStudentSession handles DELETE /v1/families/students/:student_id/sessions/:id.
//
// @Summary     Revoke a student session
// @Tags        students
// @Param       student_id  path  string  true  "Student ID"
// @Param       id          path  string  true  "Session ID"
// @Success     204
// @Failure     401  {object}  shared.ErrorResponse
// @Failure     404  {object}  shared.ErrorResponse
// @Router      /families/students/{student_id}/sessions/{id} [delete]
func (h *Handler) revokeStudentSession(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student ID")
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid session ID")
	}
	if err := h.svc.RevokeStudentSession(c.Request().Context(), &scope, studentID, sessionID); err != nil {
		return mapIamError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// getStudentSessionMe handles GET /v1/student-session/me.
// Uses student bearer token, not a session cookie. [§5]
//
// @Summary     Get student session identity
// @Tags        student-session
// @Produce     json
// @Success     200  {object}  StudentSessionIdentityResponse
// @Failure     401  {object}  shared.ErrorResponse
// @Router      /student-session/me [get]
func (h *Handler) getStudentSessionMe(c echo.Context) error {
	identity, ok := c.Get("student_session_identity").(*StudentSessionIdentityResponse)
	if !ok || identity == nil {
		return shared.ErrUnauthorized()
	}
	return c.JSON(http.StatusOK, identity)
}

// requireStudentSessionMiddleware reads the Authorization: Bearer <token> header,
// validates it via GetStudentSessionMe, and stores the identity in the echo context. [§5]
func (h *Handler) requireStudentSessionMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			const prefix = "Bearer "
			if len(authHeader) <= len(prefix) || authHeader[:len(prefix)] != prefix {
				return shared.ErrUnauthorized()
			}
			token := authHeader[len(prefix):]

			identity, err := h.svc.GetStudentSessionMe(c.Request().Context(), token)
			if err != nil {
				return mapIamError(err)
			}
			c.Set("student_session_identity", identity)
			return next(c)
		}
	}
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
	case errors.Is(err, ErrStudentSessionNotFound):
		return &shared.AppError{Code: "student_session_not_found", Message: "Student session not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrStudentSessionExpired):
		return &shared.AppError{Code: "student_session_expired", Message: "Student session expired or inactive", StatusCode: http.StatusUnauthorized}
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

// ─── Stub Handlers (Phase 3) ────────────────────────────────────────────────

// getMfaStatus returns MFA status (stub — always disabled).
func (h *Handler) getMfaStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]any{
		"enabled": false,
		"methods": []any{},
	})
}
