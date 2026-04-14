package comply

import (
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/comply/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/middleware"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler holds the comply HTTP handler dependencies.
type Handler struct {
	svc ComplianceService
}

// NewHandler creates a new comply Handler.
func NewHandler(svc ComplianceService) *Handler {
	return &Handler{svc: svc}
}

// Register registers all compliance routes on the authenticated route group.
// All endpoints require RequirePremium — 402 for free-tier families. [14-comply §4]
func (h *Handler) Register(authGroup *echo.Group) {
	c := authGroup.Group("/compliance")

	// ─── Config ──────────────────────────────────────────────────────
	c.GET("/config", h.getFamilyConfig)
	c.PUT("/config", h.upsertFamilyConfig)

	// ─── State Requirements ──────────────────────────────────────────
	c.GET("/state-requirements", h.listStateConfigs)
	c.GET("/state-requirements/:state_code", h.getStateConfig)

	// ─── Schedules ───────────────────────────────────────────────────
	c.POST("/schedules", h.createSchedule)
	c.GET("/schedules", h.listSchedules)
	c.PATCH("/schedules/:id", h.updateSchedule)
	c.DELETE("/schedules/:id", h.deleteSchedule)

	// ─── Attendance ──────────────────────────────────────────────────
	c.POST("/students/:student_id/attendance", h.recordAttendance)
	c.GET("/students/:student_id/attendance", h.listAttendance)
	c.PATCH("/students/:student_id/attendance/:id", h.updateAttendance)
	c.DELETE("/students/:student_id/attendance/:id", h.deleteAttendance)
	c.POST("/students/:student_id/attendance/bulk", h.bulkRecordAttendance)
	c.GET("/students/:student_id/attendance/summary", h.getAttendanceSummary)

	// ─── Assessments ─────────────────────────────────────────────────
	c.POST("/students/:student_id/assessments", h.createAssessment)
	c.GET("/students/:student_id/assessments", h.listAssessments)
	c.PATCH("/students/:student_id/assessments/:id", h.updateAssessment)
	c.DELETE("/students/:student_id/assessments/:id", h.deleteAssessment)

	// ─── Standardized Tests ──────────────────────────────────────────
	c.POST("/students/:student_id/tests", h.createTestScore)
	c.GET("/students/:student_id/tests", h.listTestScores)
	c.PATCH("/students/:student_id/tests/:id", h.updateTestScore)
	c.DELETE("/students/:student_id/tests/:id", h.deleteTestScore)

	// ─── Portfolios ──────────────────────────────────────────────────
	c.POST("/students/:student_id/portfolios", h.createPortfolio)
	c.GET("/students/:student_id/portfolios", h.listPortfolios)
	c.GET("/students/:student_id/portfolios/candidates", h.getPortfolioCandidates)
	c.GET("/students/:student_id/portfolios/:id", h.getPortfolio)
	c.POST("/students/:student_id/portfolios/:id/items", h.addPortfolioItems)
	c.POST("/students/:student_id/portfolios/:id/generate", h.generatePortfolio)
	c.GET("/students/:student_id/portfolios/:id/download", h.getPortfolioDownloadURL)

	// ─── Dashboard ───────────────────────────────────────────────────
	c.GET("/dashboard", h.getDashboard)

	// ─── Transcripts (Phase 3) ───────────────────────────────────────
	c.POST("/students/:student_id/transcripts", h.createTranscript)
	c.GET("/students/:student_id/transcripts", h.listTranscripts)
	c.GET("/students/:student_id/transcripts/:id", h.getTranscript)
	c.DELETE("/students/:student_id/transcripts/:id", h.deleteTranscript)
	c.POST("/students/:student_id/transcripts/:id/generate", h.generateTranscript)
	c.GET("/students/:student_id/transcripts/:id/download", h.getTranscriptDownloadURL)

	// ─── Courses (Phase 3) ───────────────────────────────────────────
	c.POST("/students/:student_id/courses", h.createCourse)
	c.GET("/students/:student_id/courses", h.listCourses)
	c.PATCH("/students/:student_id/courses/:id", h.updateCourse)
	c.DELETE("/students/:student_id/courses/:id", h.deleteCourse)

	// ─── GPA (Phase 3) ──────────────────────────────────────────────
	c.GET("/students/:student_id/gpa", h.calculateGPA)
	c.GET("/students/:student_id/gpa/what-if", h.calculateGPAWhatIf)
	c.GET("/students/:student_id/gpa/history", h.getGPAHistory)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Config Handlers
// ═══════════════════════════════════════════════════════════════════════════════

// getFamilyConfig godoc
//
// @Summary     Get family compliance configuration
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} FamilyConfigResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/config [get]
func (h *Handler) getFamilyConfig(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.GetFamilyConfig(c.Request().Context(), scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// upsertFamilyConfig godoc
//
// @Summary     Create or update family compliance configuration
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body UpsertFamilyConfigCommand true "Family config"
// @Success     200 {object} FamilyConfigResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /compliance/config [put]
func (h *Handler) upsertFamilyConfig(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var cmd UpsertFamilyConfigCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.UpsertFamilyConfig(c.Request().Context(), cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// State Config Handlers
// ═══════════════════════════════════════════════════════════════════════════════

// listStateConfigs godoc
//
// @Summary     List all state compliance requirements
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} StateConfigSummaryResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/state-requirements [get]
func (h *Handler) listStateConfigs(c echo.Context) error {
	if _, err := middleware.RequirePremium(c); err != nil {
		return err
	}

	resp, err := h.svc.ListStateConfigs(c.Request().Context())
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getStateConfig godoc
//
// @Summary     Get state compliance requirements
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       state_code path string true "Two-letter state code"
// @Success     200 {object} StateConfigResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/state-requirements/{state_code} [get]
func (h *Handler) getStateConfig(c echo.Context) error {
	if _, err := middleware.RequirePremium(c); err != nil {
		return err
	}

	stateCode := c.Param("state_code")
	resp, err := h.svc.GetStateConfig(c.Request().Context(), stateCode)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Schedule Handlers
// ═══════════════════════════════════════════════════════════════════════════════

// createSchedule godoc
//
// @Summary     Create a custom schedule
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CreateScheduleCommand true "Schedule"
// @Success     201 {object} ScheduleResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /compliance/schedules [post]
func (h *Handler) createSchedule(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	var cmd CreateScheduleCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.CreateSchedule(c.Request().Context(), cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listSchedules godoc
//
// @Summary     List family's custom schedules
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Success     200 {array} ScheduleResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/schedules [get]
func (h *Handler) listSchedules(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.ListSchedules(c.Request().Context(), scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateSchedule godoc
//
// @Summary     Update a custom schedule
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       id   path string              true "Schedule UUID"
// @Param       body body UpdateScheduleCommand true "Partial update"
// @Success     200 {object} ScheduleResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/schedules/{id} [patch]
func (h *Handler) updateSchedule(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid schedule id")
	}

	var cmd UpdateScheduleCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.UpdateSchedule(c.Request().Context(), id, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteSchedule godoc
//
// @Summary     Delete a custom schedule
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       id path string true "Schedule UUID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /compliance/schedules/{id} [delete]
func (h *Handler) deleteSchedule(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid schedule id")
	}

	if err := h.svc.DeleteSchedule(c.Request().Context(), id, scope); err != nil {
		return mapComplyError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Attendance Handlers
// ═══════════════════════════════════════════════════════════════════════════════

// recordAttendance godoc
//
// @Summary     Record daily attendance
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                   true "Student UUID"
// @Param       body       body RecordAttendanceCommand  true "Attendance record"
// @Success     201 {object} AttendanceResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /compliance/students/{student_id}/attendance [post]
func (h *Handler) recordAttendance(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var cmd RecordAttendanceCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.RecordAttendance(c.Request().Context(), studentID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// bulkRecordAttendance godoc
//
// @Summary     Bulk record attendance (up to 31 records)
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                       true "Student UUID"
// @Param       body       body BulkRecordAttendanceCommand  true "Bulk records"
// @Success     201 {array} AttendanceResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /compliance/students/{student_id}/attendance/bulk [post]
func (h *Handler) bulkRecordAttendance(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var cmd BulkRecordAttendanceCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.BulkRecordAttendance(c.Request().Context(), studentID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listAttendance godoc
//
// @Summary     List attendance records for a student
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path  string true  "Student UUID"
// @Param       start_date query string true  "Start date"
// @Param       end_date   query string true  "End date"
// @Param       status     query string false "Filter by status"
// @Param       cursor     query string false "Pagination cursor"
// @Param       limit      query int    false "Results per page"
// @Success     200 {object} AttendanceListResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/attendance [get]
func (h *Handler) listAttendance(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var params AttendanceListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.ListAttendance(c.Request().Context(), studentID, params, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getAttendanceSummary godoc
//
// @Summary     Get attendance summary with pace calculation
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path  string true "Student UUID"
// @Param       start_date query string true "Start date"
// @Param       end_date   query string true "End date"
// @Success     200 {object} AttendanceSummaryResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/attendance/summary [get]
func (h *Handler) getAttendanceSummary(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var params AttendanceSummaryParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.GetAttendanceSummary(c.Request().Context(), studentID, params, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateAttendance godoc
//
// @Summary     Update an attendance record
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                   true "Student UUID"
// @Param       id         path string                   true "Attendance UUID"
// @Param       body       body UpdateAttendanceCommand  true "Partial update"
// @Success     200 {object} AttendanceResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/attendance/{id} [patch]
func (h *Handler) updateAttendance(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	attendanceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid attendance id")
	}

	var cmd UpdateAttendanceCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.UpdateAttendance(c.Request().Context(), studentID, attendanceID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteAttendance godoc
//
// @Summary     Delete an attendance record
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Attendance UUID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/attendance/{id} [delete]
func (h *Handler) deleteAttendance(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	attendanceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid attendance id")
	}

	if err := h.svc.DeleteAttendance(c.Request().Context(), studentID, attendanceID, scope); err != nil {
		return mapComplyError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Assessment Handlers
// ═══════════════════════════════════════════════════════════════════════════════

// createAssessment godoc
//
// @Summary     Create an assessment record
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                  true "Student UUID"
// @Param       body       body CreateAssessmentCommand true "Assessment"
// @Success     201 {object} AssessmentResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /compliance/students/{student_id}/assessments [post]
func (h *Handler) createAssessment(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var cmd CreateAssessmentCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.CreateAssessment(c.Request().Context(), studentID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listAssessments godoc
//
// @Summary     List assessment records for a student
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path  string true  "Student UUID"
// @Param       subject    query string false "Filter by subject"
// @Param       start_date query string false "Start date"
// @Param       end_date   query string false "End date"
// @Param       cursor     query string false "Pagination cursor"
// @Param       limit      query int    false "Results per page"
// @Success     200 {object} AssessmentListResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/assessments [get]
func (h *Handler) listAssessments(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var params AssessmentListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ErrValidation(err.Error())
	}

	resp, err := h.svc.ListAssessments(c.Request().Context(), studentID, params, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateAssessment godoc
//
// @Summary     Update an assessment record
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                  true "Student UUID"
// @Param       id         path string                  true "Assessment UUID"
// @Param       body       body UpdateAssessmentCommand true "Partial update"
// @Success     200 {object} AssessmentResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/assessments/{id} [patch]
func (h *Handler) updateAssessment(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	assessmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid assessment id")
	}

	var cmd UpdateAssessmentCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.UpdateAssessment(c.Request().Context(), studentID, assessmentID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteAssessment godoc
//
// @Summary     Delete an assessment record
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Assessment UUID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/assessments/{id} [delete]
func (h *Handler) deleteAssessment(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	assessmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid assessment id")
	}

	if err := h.svc.DeleteAssessment(c.Request().Context(), studentID, assessmentID, scope); err != nil {
		return mapComplyError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Standardized Test Handlers
// ═══════════════════════════════════════════════════════════════════════════════

// createTestScore godoc
//
// @Summary     Record a standardized test score
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                 true "Student UUID"
// @Param       body       body CreateTestScoreCommand true "Test score"
// @Success     201 {object} TestScoreResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/tests [post]
func (h *Handler) createTestScore(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var cmd CreateTestScoreCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.CreateTestScore(c.Request().Context(), studentID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listTestScores godoc
//
// @Summary     List standardized test scores
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path  string true  "Student UUID"
// @Param       cursor     query string false "Pagination cursor"
// @Param       limit      query int    false "Results per page"
// @Success     200 {object} TestListResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/tests [get]
func (h *Handler) listTestScores(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var params TestListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ErrValidation(err.Error())
	}

	resp, err := h.svc.ListTestScores(c.Request().Context(), studentID, params, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateTestScore godoc
//
// @Summary     Update a test score
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                 true "Student UUID"
// @Param       id         path string                 true "Test UUID"
// @Param       body       body UpdateTestScoreCommand true "Partial update"
// @Success     200 {object} TestScoreResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/tests/{id} [patch]
func (h *Handler) updateTestScore(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	testID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid test id")
	}

	var cmd UpdateTestScoreCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.UpdateTestScore(c.Request().Context(), studentID, testID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteTestScore godoc
//
// @Summary     Delete a test score
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Test UUID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/tests/{id} [delete]
func (h *Handler) deleteTestScore(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	testID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid test id")
	}

	if err := h.svc.DeleteTestScore(c.Request().Context(), studentID, testID, scope); err != nil {
		return mapComplyError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Portfolio Handlers
// ═══════════════════════════════════════════════════════════════════════════════

// createPortfolio godoc
//
// @Summary     Create a portfolio
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                  true "Student UUID"
// @Param       body       body CreatePortfolioCommand  true "Portfolio"
// @Success     201 {object} PortfolioResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /compliance/students/{student_id}/portfolios [post]
func (h *Handler) createPortfolio(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var cmd CreatePortfolioCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.CreatePortfolio(c.Request().Context(), studentID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listPortfolios godoc
//
// @Summary     List portfolios for a student
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Success     200 {array} PortfolioSummaryResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/portfolios [get]
func (h *Handler) listPortfolios(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	resp, err := h.svc.ListPortfolios(c.Request().Context(), studentID, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getPortfolio godoc
//
// @Summary     Get portfolio details
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Portfolio UUID"
// @Success     200 {object} PortfolioResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/portfolios/{id} [get]
func (h *Handler) getPortfolio(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	portfolioID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid portfolio id")
	}

	resp, err := h.svc.GetPortfolio(c.Request().Context(), studentID, portfolioID, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// addPortfolioItems godoc
//
// @Summary     Add items to a portfolio
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                   true "Student UUID"
// @Param       id         path string                   true "Portfolio UUID"
// @Param       body       body AddPortfolioItemsCommand true "Items to add"
// @Success     201 {array} PortfolioItemResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /compliance/students/{student_id}/portfolios/{id}/items [post]
func (h *Handler) addPortfolioItems(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	portfolioID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid portfolio id")
	}

	var cmd AddPortfolioItemsCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.AddPortfolioItems(c.Request().Context(), studentID, portfolioID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// generatePortfolio godoc
//
// @Summary     Trigger portfolio PDF generation
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Portfolio UUID"
// @Success     202 {object} PortfolioResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /compliance/students/{student_id}/portfolios/{id}/generate [post]
func (h *Handler) generatePortfolio(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	portfolioID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid portfolio id")
	}

	resp, err := h.svc.GeneratePortfolio(c.Request().Context(), studentID, portfolioID, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusAccepted, resp)
}

// getPortfolioDownloadURL godoc
//
// @Summary     Get portfolio download URL
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Portfolio UUID"
// @Success     200 {object} map[string]string
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Failure     410 {object} shared.AppError
// @Router      /compliance/students/{student_id}/portfolios/{id}/download [get]
func (h *Handler) getPortfolioDownloadURL(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	portfolioID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid portfolio id")
	}

	url, err := h.svc.GetPortfolioDownloadURL(c.Request().Context(), studentID, portfolioID, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"download_url": url})
}

// getPortfolioCandidates godoc
//
// @Summary     List portfolio artifact candidates for a student
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Success     200 {array} object
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/portfolios/candidates [get]
func (h *Handler) getPortfolioCandidates(c echo.Context) error {
	_, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, []any{})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Dashboard Handler
// ═══════════════════════════════════════════════════════════════════════════════

// getDashboard godoc
//
// @Summary     Get compliance dashboard overview
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} ComplianceDashboardResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/dashboard [get]
func (h *Handler) getDashboard(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)

	resp, err := h.svc.GetDashboard(c.Request().Context(), scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Transcript Handlers (Phase 3)
// ═══════════════════════════════════════════════════════════════════════════════

// createTranscript godoc
//
// @Summary     Create a transcript
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string                   true "Student UUID"
// @Param       body       body CreateTranscriptCommand  true "Transcript"
// @Success     201 {object} TranscriptResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/transcripts [post]
func (h *Handler) createTranscript(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var cmd CreateTranscriptCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.CreateTranscript(c.Request().Context(), studentID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listTranscripts godoc
//
// @Summary     List transcripts for a student
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Success     200 {array} TranscriptSummaryResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/transcripts [get]
func (h *Handler) listTranscripts(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	resp, err := h.svc.ListTranscripts(c.Request().Context(), studentID, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getTranscript godoc
//
// @Summary     Get transcript details
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Transcript UUID"
// @Success     200 {object} TranscriptResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/transcripts/{id} [get]
func (h *Handler) getTranscript(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	transcriptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid transcript id")
	}

	resp, err := h.svc.GetTranscript(c.Request().Context(), studentID, transcriptID, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteTranscript godoc
//
// @Summary     Delete a transcript
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Transcript UUID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/transcripts/{id} [delete]
func (h *Handler) deleteTranscript(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	transcriptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid transcript id")
	}

	if err := h.svc.DeleteTranscript(c.Request().Context(), studentID, transcriptID, scope); err != nil {
		return mapComplyError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// generateTranscript godoc
//
// @Summary     Trigger transcript PDF generation
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Transcript UUID"
// @Success     202 {object} TranscriptResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/transcripts/{id}/generate [post]
func (h *Handler) generateTranscript(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	transcriptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid transcript id")
	}

	resp, err := h.svc.GenerateTranscript(c.Request().Context(), studentID, transcriptID, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusAccepted, resp)
}

// getTranscriptDownloadURL godoc
//
// @Summary     Get transcript download URL
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Transcript UUID"
// @Success     200 {object} map[string]string
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Failure     410 {object} shared.AppError
// @Router      /compliance/students/{student_id}/transcripts/{id}/download [get]
func (h *Handler) getTranscriptDownloadURL(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	transcriptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid transcript id")
	}

	url, err := h.svc.GetTranscriptDownloadURL(c.Request().Context(), studentID, transcriptID, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"download_url": url})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Course Handlers (Phase 3)
// ═══════════════════════════════════════════════════════════════════════════════

// createCourse godoc
//
// @Summary     Create a course
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string              true "Student UUID"
// @Param       body       body CreateCourseCommand true "Course"
// @Success     201 {object} CourseResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /compliance/students/{student_id}/courses [post]
func (h *Handler) createCourse(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var cmd CreateCourseCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.CreateCourse(c.Request().Context(), studentID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listCourses godoc
//
// @Summary     List courses for a student
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id  path  string true  "Student UUID"
// @Param       grade_level query int    false "Filter by grade level"
// @Param       school_year query string false "Filter by school year"
// @Param       cursor      query string false "Pagination cursor"
// @Param       limit       query int    false "Results per page"
// @Success     200 {object} CourseListResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/courses [get]
func (h *Handler) listCourses(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var params CourseListParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ErrValidation(err.Error())
	}

	resp, err := h.svc.ListCourses(c.Request().Context(), studentID, params, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateCourse godoc
//
// @Summary     Update a course
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string              true "Student UUID"
// @Param       id         path string              true "Course UUID"
// @Param       body       body UpdateCourseCommand true "Partial update"
// @Success     200 {object} CourseResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/courses/{id} [patch]
func (h *Handler) updateCourse(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	courseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid course id")
	}

	var cmd UpdateCourseCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.UpdateCourse(c.Request().Context(), studentID, courseID, cmd, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteCourse godoc
//
// @Summary     Delete a course
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Param       id         path string true "Course UUID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /compliance/students/{student_id}/courses/{id} [delete]
func (h *Handler) deleteCourse(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}
	courseID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid course id")
	}

	if err := h.svc.DeleteCourse(c.Request().Context(), studentID, courseID, scope); err != nil {
		return mapComplyError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ═══════════════════════════════════════════════════════════════════════════════
// GPA Handlers (Phase 3)
// ═══════════════════════════════════════════════════════════════════════════════

// calculateGPA godoc
//
// @Summary     Calculate current GPA
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path  string true  "Student UUID"
// @Param       scale      query string false "GPA scale"
// @Success     200 {object} GpaResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/gpa [get]
func (h *Handler) calculateGPA(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var params GpaParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query parameters")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ErrValidation(err.Error())
	}

	resp, err := h.svc.CalculateGPA(c.Request().Context(), studentID, params, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// calculateGPAWhatIf godoc
//
// @Summary     Calculate what-if GPA with hypothetical courses
// @Tags        compliance
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string         true "Student UUID"
// @Param       body       body GpaWhatIfParams true "What-if params"
// @Success     200 {object} GpaResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/gpa/what-if [get]
func (h *Handler) calculateGPAWhatIf(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	var params GpaWhatIfParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.CalculateGPAWhatIf(c.Request().Context(), studentID, params, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getGPAHistory godoc
//
// @Summary     Get GPA history by term
// @Tags        compliance
// @Produce     json
// @Security    BearerAuth
// @Param       student_id path string true "Student UUID"
// @Success     200 {array} GpaTermResponse
// @Failure     401 {object} shared.AppError
// @Failure     402 {object} shared.AppError
// @Router      /compliance/students/{student_id}/gpa/history [get]
func (h *Handler) getGPAHistory(c echo.Context) error {
	auth, err := middleware.RequirePremium(c)
	if err != nil {
		return err
	}
	scope := shared.NewFamilyScopeFromAuth(auth)
	studentID, err := uuid.Parse(c.Param("student_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid student_id")
	}

	resp, err := h.svc.GetGPAHistory(c.Request().Context(), studentID, scope)
	if err != nil {
		return mapComplyError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Error Mapping [14-comply §16, CODING §2.2]
// ═══════════════════════════════════════════════════════════════════════════════

// mapComplyError maps domain errors to HTTP-appropriate AppErrors.
// Internal error details are never exposed to the client.
func mapComplyError(err error) error {
	switch {
	// ─── Not Found ─────────────────────────────────────
	case errors.Is(err, ErrAttendanceNotFound):
		return &shared.AppError{Code: "not_found", Message: "Attendance record not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrAssessmentNotFound):
		return &shared.AppError{Code: "not_found", Message: "Assessment record not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrTestScoreNotFound):
		return &shared.AppError{Code: "not_found", Message: "Test score not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrPortfolioNotFound):
		return &shared.AppError{Code: "not_found", Message: "Portfolio not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrTranscriptNotFound):
		return &shared.AppError{Code: "not_found", Message: "Transcript not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrCourseNotFound):
		return &shared.AppError{Code: "not_found", Message: "Course not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrScheduleNotFound):
		return &shared.AppError{Code: "not_found", Message: "Schedule not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrStateConfigNotFound):
		return &shared.AppError{Code: "not_found", Message: "State config not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrStudentNotInFamily):
		return &shared.AppError{Code: "not_found", Message: "Student not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrPortfolioItemSourceNotFound):
		return &shared.AppError{Code: "not_found", Message: "Portfolio item source not found", StatusCode: http.StatusNotFound}

	// ─── Validation / 422 ──────────────────────────────
	case errors.Is(err, domain.ErrFutureAttendanceDate):
		return shared.ErrValidation("Cannot record attendance for a future date")
	case errors.Is(err, domain.ErrInvalidAttendanceStatus):
		return shared.ErrValidation("Invalid attendance status")
	case errors.Is(err, domain.ErrDurationRequiredForPartial):
		return shared.ErrValidation("Duration is required for partial attendance")
	case errors.Is(err, domain.ErrNegativeDuration):
		return shared.ErrValidation("Duration cannot be negative")
	case errors.Is(err, domain.ErrEmptyPortfolio):
		return shared.ErrValidation("Cannot generate an empty portfolio")
	case errors.Is(err, ErrInvalidStateCode):
		return shared.ErrValidation("Invalid state code")
	case errors.Is(err, ErrInvalidSchoolYearRange):
		return shared.ErrValidation("Invalid school year date range")
	case errors.Is(err, ErrInvalidSchoolDaysArray):
		return shared.ErrValidation("School days array must have exactly 7 elements")
	case errors.Is(err, ErrBulkAttendanceLimitExceeded):
		return shared.ErrValidation("Bulk attendance exceeds maximum of 31 records")
	case errors.Is(err, ErrInvalidCourseLevel):
		return shared.ErrValidation("Invalid course level")
	case errors.Is(err, ErrInvalidAssessmentType):
		return shared.ErrValidation("Invalid assessment type")

	// ─── Conflict / 409 ────────────────────────────────
	case errors.Is(err, ErrScheduleInUse):
		return shared.ErrConflict("Schedule is in use by family config")
	case errors.Is(err, ErrDuplicatePortfolioItem):
		return shared.ErrConflict("Duplicate item in portfolio")
	case errors.Is(err, domain.ErrPortfolioNotConfiguring):
		return shared.ErrConflict("Portfolio is not in a configurable state")
	case errors.Is(err, domain.ErrMaxRetriesExceeded):
		return shared.ErrConflict("Maximum generation attempts exceeded")
	case errors.Is(err, ErrPortfolioNotReady):
		return shared.ErrConflict("Portfolio is not ready for download")

	// ─── Gone / 410 ───────────────────────────────────
	case errors.Is(err, ErrPortfolioExpired):
		return &shared.AppError{Code: "gone", Message: "Portfolio has expired", StatusCode: http.StatusGone}

	// ─── InvalidPortfolioTransitionError → 409 ────────
	default:
		var transErr *domain.InvalidPortfolioTransitionError
		if errors.As(err, &transErr) {
			return shared.ErrConflict("Invalid portfolio status transition")
		}
		// Never expose internal error details. [CODING §3.1]
		return shared.ErrInternal(err)
	}
}
