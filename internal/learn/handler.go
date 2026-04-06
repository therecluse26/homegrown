package learn

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler holds the learning HTTP handler dependencies.
type Handler struct {
	svc LearningService
}

// NewHandler creates a new learning Handler.
func NewHandler(svc LearningService) *Handler {
	return &Handler{svc: svc}
}

// Register registers all learning routes on the authenticated route group.
// All learning endpoints require authentication. [06-learn §4]
func (h *Handler) Register(authGroup *echo.Group) {
	learn := authGroup.Group("/learning")

	// Activity Definitions (Layer 1)
	learn.POST("/activity-defs", h.createActivityDef)
	learn.GET("/activity-defs", h.listActivityDefs)
	learn.GET("/activity-defs/:id", h.getActivityDef)
	learn.PATCH("/activity-defs/:id", h.updateActivityDef)
	learn.DELETE("/activity-defs/:id", h.deleteActivityDef)

	// Reading Items (Layer 1)
	learn.POST("/reading-items", h.createReadingItem)
	learn.GET("/reading-items", h.listReadingItems)
	learn.GET("/reading-items/:id", h.getReadingItem)
	learn.PATCH("/reading-items/:id", h.updateReadingItem)

	// Activity Logs (Layer 3 — student-scoped)
	learn.POST("/students/:studentId/activities", h.logActivity)
	learn.GET("/students/:studentId/activities", h.listActivityLogs)
	learn.GET("/students/:studentId/activities/:id", h.getActivityLog)
	learn.PATCH("/students/:studentId/activities/:id", h.updateActivityLog)
	learn.DELETE("/students/:studentId/activities/:id", h.deleteActivityLog)

	// Reading Progress (Layer 3 — student-scoped)
	learn.POST("/students/:studentId/reading", h.startReading)
	learn.GET("/students/:studentId/reading", h.listReadingProgress)
	learn.PATCH("/students/:studentId/reading/:id", h.updateReadingProgress)

	// Journal Entries (Layer 3 — student-scoped)
	learn.POST("/students/:studentId/journal", h.createJournalEntry)
	learn.GET("/students/:studentId/journal", h.listJournalEntries)
	learn.GET("/students/:studentId/journal/:id", h.getJournalEntry)
	learn.PATCH("/students/:studentId/journal/:id", h.updateJournalEntry)
	learn.DELETE("/students/:studentId/journal/:id", h.deleteJournalEntry)

	// Reading Lists (Layer 3 — family-scoped)
	learn.POST("/reading-lists", h.createReadingList)
	learn.GET("/reading-lists", h.listReadingLists)
	learn.GET("/reading-lists/:id", h.getReadingList)
	learn.PATCH("/reading-lists/:id", h.updateReadingList)
	learn.DELETE("/reading-lists/:id", h.deleteReadingList)

	// Subject Taxonomy
	learn.GET("/taxonomy", h.getSubjectTaxonomy)
	learn.POST("/taxonomy/custom", h.createCustomSubject)

	// Artifact Links (Layer 1 — polymorphic)
	learn.POST("/artifact-links", h.linkArtifacts)
	learn.DELETE("/artifact-links/:id", h.unlinkArtifacts)
	learn.GET("/content/:type/:id/links", h.getLinkedArtifacts)

	// Progress (Layer 3 — computed on-the-fly)
	learn.GET("/students/:studentId/progress", h.getProgressSummary)
	learn.GET("/students/:studentId/progress/subjects", h.getSubjectBreakdown)
	learn.GET("/students/:studentId/progress/timeline", h.getActivityTimeline)

	// Export (Layer 3)
	learn.POST("/export", h.requestDataExport)
	learn.GET("/export/:id", h.getExportRequest)

	// Tools (delegation to method::)
	learn.GET("/tools", h.getResolvedTools)
	learn.GET("/students/:studentId/tools", h.getStudentTools)

	// Questions (Layer 1)
	learn.POST("/questions", h.createQuestion)
	learn.GET("/questions", h.listQuestions)
	learn.PATCH("/questions/:id", h.updateQuestion)

	// Quiz Definitions (Layer 1)
	learn.POST("/quiz-defs", h.createQuizDef)
	learn.GET("/quiz-defs/:id", h.getQuizDef)
	learn.PATCH("/quiz-defs/:id", h.updateQuizDef)

	// Quiz Sessions (Layer 3 — student-scoped)
	learn.POST("/students/:studentId/quiz-sessions", h.startQuizSession)
	learn.GET("/students/:studentId/quiz-sessions/:id", h.getQuizSession)
	learn.PATCH("/students/:studentId/quiz-sessions/:id", h.updateQuizSession)
	learn.POST("/students/:studentId/quiz-sessions/:id/score", h.scoreQuizSession)

	// Sequence Definitions (Layer 1)
	learn.POST("/sequences", h.createSequenceDef)
	learn.GET("/sequences/:id", h.getSequenceDef)
	learn.PATCH("/sequences/:id", h.updateSequenceDef)

	// Sequence Progress (Layer 3 — student-scoped)
	learn.POST("/students/:studentId/sequence-progress", h.startSequence)
	learn.GET("/students/:studentId/sequence-progress/:id", h.getSequenceProgress)
	learn.PATCH("/students/:studentId/sequence-progress/:id", h.updateSequenceProgress)

	// Assignments (Layer 3 — student-scoped)
	learn.POST("/students/:studentId/assignments", h.createAssignment)
	learn.GET("/students/:studentId/assignments", h.listAssignments)
	learn.PATCH("/students/:studentId/assignments/:id", h.updateAssignment)
	learn.DELETE("/students/:studentId/assignments/:id", h.deleteAssignment)

	// Video Definitions (Layer 1)
	learn.GET("/videos", h.listVideoDefs)
	learn.GET("/videos/:id", h.getVideoDef)

	// Video Progress (Layer 3 — student-scoped)
	learn.PATCH("/students/:studentId/video-progress", h.updateVideoProgress)
	learn.GET("/students/:studentId/video-progress/:videoDefId", h.getVideoProgress)

	// Assessment Definitions (Layer 1 — Phase 2)
	learn.POST("/assessment-defs", h.createAssessmentDef)
	learn.GET("/assessment-defs", h.listAssessmentDefs)
	learn.GET("/assessment-defs/:id", h.getAssessmentDef)
	learn.PATCH("/assessment-defs/:id", h.updateAssessmentDef)
	learn.DELETE("/assessment-defs/:id", h.deleteAssessmentDef)

	// Project Definitions (Layer 1 — Phase 2)
	learn.POST("/project-defs", h.createProjectDef)
	learn.GET("/project-defs", h.listProjectDefs)
	learn.GET("/project-defs/:id", h.getProjectDef)
	learn.PATCH("/project-defs/:id", h.updateProjectDef)
	learn.DELETE("/project-defs/:id", h.deleteProjectDef)

	// Assessment Results (Layer 3 — student-scoped, Phase 2)
	learn.POST("/students/:studentId/assessments", h.recordAssessmentResult)
	learn.GET("/students/:studentId/assessments", h.listAssessmentResults)
	learn.GET("/students/:studentId/assessments/:id", h.getAssessmentResult)
	learn.PATCH("/students/:studentId/assessments/:id", h.updateAssessmentResult)
	learn.DELETE("/students/:studentId/assessments/:id", h.deleteAssessmentResult)

	// Project Progress (Layer 3 — student-scoped, Phase 2)
	learn.POST("/students/:studentId/projects", h.startProject)
	learn.GET("/students/:studentId/projects", h.listProjectProgress)
	learn.GET("/students/:studentId/projects/:id", h.getProjectProgress)
	learn.PATCH("/students/:studentId/projects/:id", h.updateProjectProgress)
	learn.DELETE("/students/:studentId/projects/:id", h.deleteProjectProgress)

	// Grading Scales (Layer 3 — family-scoped, Phase 2)
	learn.POST("/grading-scales", h.createGradingScale)
	learn.GET("/grading-scales", h.listGradingScales)
	learn.GET("/grading-scales/:id", h.getGradingScale)
	learn.PATCH("/grading-scales/:id", h.updateGradingScale)
	learn.DELETE("/grading-scales/:id", h.deleteGradingScale)
}

// ─── Activity Definition Handlers ───────────────────────────────────────────

// createActivityDef godoc
//
//	@Summary      Create an activity definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateActivityDefCommand  true  "Activity definition payload"
//	@Success      201   {object}  ActivityDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/activity-defs [post]
func (h *Handler) createActivityDef(c echo.Context) error {
	var cmd CreateActivityDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateActivityDef(c.Request().Context(), cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listActivityDefs godoc
//
//	@Summary      List activity definitions
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        limit          query     int     false  "Page size (default 20, max 50)"
//	@Param        subject        query     string  false  "Filter by subject"
//	@Param        methodology_id query     string  false  "Filter by methodology UUID"
//	@Param        publisher_id   query     string  false  "Filter by publisher UUID"
//	@Param        search         query     string  false  "Full-text search"
//	@Param        cursor         query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   ActivityDefSummaryResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/activity-defs [get]
func (h *Handler) listActivityDefs(c echo.Context) error {
	query := ActivityDefQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("subject"); s != "" {
		query.Subject = &s
	}
	if m := c.QueryParam("methodology_id"); m != "" {
		if id, err := uuid.Parse(m); err == nil {
			query.MethodologyID = &id
		}
	}
	if p := c.QueryParam("publisher_id"); p != "" {
		if id, err := uuid.Parse(p); err == nil {
			query.PublisherID = &id
		}
	}
	if s := c.QueryParam("search"); s != "" {
		query.Search = &s
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListActivityDefs(c.Request().Context(), query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getActivityDef godoc
//
//	@Summary      Get an activity definition by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Activity definition UUID"
//	@Success      200  {object}  ActivityDefResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/activity-defs/{id} [get]
func (h *Handler) getActivityDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	resp, err := h.svc.GetActivityDef(c.Request().Context(), defID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateActivityDef godoc
//
//	@Summary      Update an activity definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id    path      string                    true  "Activity definition UUID"
//	@Param        body  body      UpdateActivityDefCommand   true  "Fields to update"
//	@Success      200   {object}  ActivityDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      404   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/activity-defs/{id} [patch]
func (h *Handler) updateActivityDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var cmd UpdateActivityDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateActivityDef(c.Request().Context(), defID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteActivityDef godoc
//
//	@Summary      Delete an activity definition
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Activity definition UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/activity-defs/{id} [delete]
func (h *Handler) deleteActivityDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteActivityDef(c.Request().Context(), defID, auth.ParentID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Activity Log Handlers ──────────────────────────────────────────────────

// logActivity godoc
//
//	@Summary      Log a student activity
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string              true  "Student UUID"
//	@Param        body       body      LogActivityCommand  true  "Activity log payload"
//	@Success      201        {object}  ActivityLogResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/activities [post]
func (h *Handler) logActivity(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd LogActivityCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.LogActivity(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listActivityLogs godoc
//
//	@Summary      List activity logs for a student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true   "Student UUID"
//	@Param        limit      query     int     false  "Page size (default 20, max 50)"
//	@Param        subject    query     string  false  "Filter by subject"
//	@Param        date_from  query     string  false  "Start date (YYYY-MM-DD)"
//	@Param        date_to    query     string  false  "End date (YYYY-MM-DD)"
//	@Param        cursor     query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   ActivityLogResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/activities [get]
func (h *Handler) listActivityLogs(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	query := ActivityLogQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("subject"); s != "" {
		query.Subject = &s
	}
	if df := c.QueryParam("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			query.DateFrom = &t
		}
	}
	if dt := c.QueryParam("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			query.DateTo = &t
		}
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListActivityLogs(c.Request().Context(), &scope, studentID, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getActivityLog godoc
//
//	@Summary      Get a single activity log
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Activity log UUID"
//	@Success      200  {object}  ActivityLogResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/activities/{id} [get]
func (h *Handler) getActivityLog(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	logID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetActivityLog(c.Request().Context(), &scope, studentID, logID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateActivityLog godoc
//
//	@Summary      Update an activity log
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                    true  "Student UUID"
//	@Param        id         path      string                    true  "Activity log UUID"
//	@Param        body       body      UpdateActivityLogCommand   true  "Fields to update"
//	@Success      200        {object}  ActivityLogResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      404        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/activities/{id} [patch]
func (h *Handler) updateActivityLog(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	logID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateActivityLogCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateActivityLog(c.Request().Context(), &scope, studentID, logID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteActivityLog godoc
//
//	@Summary      Delete an activity log
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Activity log UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/activities/{id} [delete]
func (h *Handler) deleteActivityLog(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	logID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteActivityLog(c.Request().Context(), &scope, studentID, logID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Reading Item Handlers ──────────────────────────────────────────────────

// createReadingItem godoc
//
//	@Summary      Create a reading item
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateReadingItemCommand  true  "Reading item payload"
//	@Success      201   {object}  ReadingItemResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/reading-items [post]
func (h *Handler) createReadingItem(c echo.Context) error {
	var cmd CreateReadingItemCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateReadingItem(c.Request().Context(), cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listReadingItems godoc
//
//	@Summary      List reading items
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        limit    query     int     false  "Page size (default 20, max 50)"
//	@Param        subject  query     string  false  "Filter by subject"
//	@Param        search   query     string  false  "Full-text search"
//	@Param        isbn     query     string  false  "Filter by ISBN"
//	@Param        cursor   query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   ReadingItemSummaryResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/reading-items [get]
func (h *Handler) listReadingItems(c echo.Context) error {
	query := ReadingItemQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("subject"); s != "" {
		query.Subject = &s
	}
	if s := c.QueryParam("search"); s != "" {
		query.Search = &s
	}
	if s := c.QueryParam("isbn"); s != "" {
		query.ISBN = &s
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListReadingItems(c.Request().Context(), query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getReadingItem godoc
//
//	@Summary      Get a reading item by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Reading item UUID"
//	@Success      200  {object}  ReadingItemDetailResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/reading-items/{id} [get]
func (h *Handler) getReadingItem(c echo.Context) error {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	resp, err := h.svc.GetReadingItem(c.Request().Context(), itemID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateReadingItem godoc
//
//	@Summary      Update a reading item
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id    path      string                     true  "Reading item UUID"
//	@Param        body  body      UpdateReadingItemCommand    true  "Fields to update"
//	@Success      200   {object}  ReadingItemResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      404   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/reading-items/{id} [patch]
func (h *Handler) updateReadingItem(c echo.Context) error {
	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var cmd UpdateReadingItemCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateReadingItem(c.Request().Context(), itemID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Reading Progress Handlers ──────────────────────────────────────────────

// startReading godoc
//
//	@Summary      Start reading progress for a student
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string               true  "Student UUID"
//	@Param        body       body      StartReadingCommand   true  "Start reading payload"
//	@Success      201        {object}  ReadingProgressResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/reading [post]
func (h *Handler) startReading(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd StartReadingCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.StartReading(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listReadingProgress godoc
//
//	@Summary      List reading progress for a student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true   "Student UUID"
//	@Param        limit      query     int     false  "Page size (default 20, max 50)"
//	@Param        status     query     string  false  "Filter by status"
//	@Param        cursor     query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   ReadingProgressResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/reading [get]
func (h *Handler) listReadingProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	query := ReadingProgressQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("status"); s != "" {
		query.Status = &s
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListReadingProgress(c.Request().Context(), &scope, studentID, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateReadingProgress godoc
//
//	@Summary      Update reading progress
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                         true  "Student UUID"
//	@Param        id         path      string                         true  "Reading progress UUID"
//	@Param        body       body      UpdateReadingProgressCommand    true  "Fields to update"
//	@Success      200        {object}  ReadingProgressResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      404        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/reading/{id} [patch]
func (h *Handler) updateReadingProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	progressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateReadingProgressCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateReadingProgress(c.Request().Context(), &scope, studentID, progressID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Journal Entry Handlers ─────────────────────────────────────────────────

// createJournalEntry godoc
//
//	@Summary      Create a journal entry for a student
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                      true  "Student UUID"
//	@Param        body       body      CreateJournalEntryCommand    true  "Journal entry payload"
//	@Success      201        {object}  JournalEntryResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/journal [post]
func (h *Handler) createJournalEntry(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd CreateJournalEntryCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateJournalEntry(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listJournalEntries godoc
//
//	@Summary      List journal entries for a student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId   path      string  true   "Student UUID"
//	@Param        limit       query     int     false  "Page size (default 20, max 50)"
//	@Param        entry_type  query     string  false  "Filter by entry type"
//	@Param        search      query     string  false  "Full-text search"
//	@Param        date_from   query     string  false  "Start date (YYYY-MM-DD)"
//	@Param        date_to     query     string  false  "End date (YYYY-MM-DD)"
//	@Param        cursor      query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   JournalEntryResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/journal [get]
func (h *Handler) listJournalEntries(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	query := JournalEntryQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("entry_type"); s != "" {
		query.EntryType = &s
	}
	if s := c.QueryParam("search"); s != "" {
		query.Search = &s
	}
	if df := c.QueryParam("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			query.DateFrom = &t
		}
	}
	if dt := c.QueryParam("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			query.DateTo = &t
		}
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListJournalEntries(c.Request().Context(), &scope, studentID, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getJournalEntry godoc
//
//	@Summary      Get a journal entry by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Journal entry UUID"
//	@Success      200  {object}  JournalEntryResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/journal/{id} [get]
func (h *Handler) getJournalEntry(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetJournalEntry(c.Request().Context(), &scope, studentID, entryID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateJournalEntry godoc
//
//	@Summary      Update a journal entry
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                       true  "Student UUID"
//	@Param        id         path      string                       true  "Journal entry UUID"
//	@Param        body       body      UpdateJournalEntryCommand     true  "Fields to update"
//	@Success      200        {object}  JournalEntryResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      404        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/journal/{id} [patch]
func (h *Handler) updateJournalEntry(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateJournalEntryCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateJournalEntry(c.Request().Context(), &scope, studentID, entryID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteJournalEntry godoc
//
//	@Summary      Delete a journal entry
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Journal entry UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/journal/{id} [delete]
func (h *Handler) deleteJournalEntry(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteJournalEntry(c.Request().Context(), &scope, studentID, entryID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Reading List Handlers ──────────────────────────────────────────────────

// createReadingList godoc
//
//	@Summary      Create a reading list
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateReadingListCommand  true  "Reading list payload"
//	@Success      201   {object}  ReadingListResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/reading-lists [post]
func (h *Handler) createReadingList(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd CreateReadingListCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateReadingList(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listReadingLists godoc
//
//	@Summary      List reading lists for the family
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200  {array}   ReadingListResponse
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/reading-lists [get]
func (h *Handler) listReadingLists(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.ListReadingLists(c.Request().Context(), &scope)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getReadingList godoc
//
//	@Summary      Get a reading list by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Reading list UUID"
//	@Success      200  {object}  ReadingListDetailResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/reading-lists/{id} [get]
func (h *Handler) getReadingList(c echo.Context) error {
	listID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetReadingList(c.Request().Context(), &scope, listID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateReadingList godoc
//
//	@Summary      Update a reading list
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id    path      string                      true  "Reading list UUID"
//	@Param        body  body      UpdateReadingListCommand     true  "Fields to update"
//	@Success      200   {object}  ReadingListResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      404   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/reading-lists/{id} [patch]
func (h *Handler) updateReadingList(c echo.Context) error {
	listID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateReadingListCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateReadingList(c.Request().Context(), &scope, listID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteReadingList godoc
//
//	@Summary      Delete a reading list
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Reading list UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/reading-lists/{id} [delete]
func (h *Handler) deleteReadingList(c echo.Context) error {
	listID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteReadingList(c.Request().Context(), &scope, listID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Subject Taxonomy Handlers ──────────────────────────────────────────────

// getSubjectTaxonomy godoc
//
//	@Summary      Get the subject taxonomy
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        level      query     int     false  "Filter by taxonomy level"
//	@Param        parent_id  query     string  false  "Filter by parent subject UUID"
//	@Success      200  {object}  SubjectTaxonomyResponse
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/taxonomy [get]
func (h *Handler) getSubjectTaxonomy(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var query TaxonomyQuery
	if l := c.QueryParam("level"); l != "" {
		if n, err := strconv.ParseInt(l, 10, 16); err == nil {
			level := int16(n)
			query.Level = &level
		}
	}
	if p := c.QueryParam("parent_id"); p != "" {
		if id, err := uuid.Parse(p); err == nil {
			query.ParentID = &id
		}
	}
	resp, err := h.svc.GetSubjectTaxonomy(c.Request().Context(), &scope, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// createCustomSubject godoc
//
//	@Summary      Create a custom subject
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateCustomSubjectCommand  true  "Custom subject payload"
//	@Success      201   {object}  CustomSubjectResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/taxonomy/custom [post]
func (h *Handler) createCustomSubject(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd CreateCustomSubjectCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateCustomSubject(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// ─── Artifact Link Handlers ─────────────────────────────────────────────────

// linkArtifacts godoc
//
//	@Summary      Link two artifacts together
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateArtifactLinkCommand  true  "Artifact link payload"
//	@Success      201   {object}  ArtifactLinkResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/artifact-links [post]
func (h *Handler) linkArtifacts(c echo.Context) error {
	var cmd CreateArtifactLinkCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.LinkArtifacts(c.Request().Context(), cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// unlinkArtifacts godoc
//
//	@Summary      Remove an artifact link
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Artifact link UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/artifact-links/{id} [delete]
func (h *Handler) unlinkArtifacts(c echo.Context) error {
	linkID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	if err := h.svc.UnlinkArtifacts(c.Request().Context(), linkID, auth.ParentID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// getLinkedArtifacts godoc
//
//	@Summary      Get linked artifacts for a content item
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        type  path      string  true  "Content type"
//	@Param        id    path      string  true  "Content UUID"
//	@Success      200   {array}   ArtifactLinkResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/content/{type}/{id}/links [get]
func (h *Handler) getLinkedArtifacts(c echo.Context) error {
	contentType := c.Param("type")
	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	resp, err := h.svc.GetLinkedArtifacts(c.Request().Context(), contentType, contentID, LinkDirectionBoth)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Progress Handlers ─────────────────────────────────────────────────────

// getProgressSummary godoc
//
//	@Summary      Get a student's progress summary
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true   "Student UUID"
//	@Param        date_from  query     string  false  "Start date (YYYY-MM-DD)"
//	@Param        date_to    query     string  false  "End date (YYYY-MM-DD)"
//	@Success      200  {object}  ProgressSummaryResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/progress [get]
func (h *Handler) getProgressSummary(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var query ProgressQuery
	if df := c.QueryParam("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			query.DateFrom = &t
		}
	}
	if dt := c.QueryParam("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			query.DateTo = &t
		}
	}
	resp, err := h.svc.GetProgressSummary(c.Request().Context(), &scope, studentID, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getSubjectBreakdown godoc
//
//	@Summary      Get subject-level progress breakdown for a student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true   "Student UUID"
//	@Param        date_from  query     string  false  "Start date (YYYY-MM-DD)"
//	@Param        date_to    query     string  false  "End date (YYYY-MM-DD)"
//	@Success      200  {array}   SubjectProgressResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/progress/subjects [get]
func (h *Handler) getSubjectBreakdown(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var query ProgressQuery
	if df := c.QueryParam("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			query.DateFrom = &t
		}
	}
	if dt := c.QueryParam("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			query.DateTo = &t
		}
	}
	resp, err := h.svc.GetSubjectBreakdown(c.Request().Context(), &scope, studentID, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getActivityTimeline godoc
//
//	@Summary      Get activity timeline for a student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true   "Student UUID"
//	@Param        limit      query     int     false  "Page size (default 20, max 50)"
//	@Param        date_from  query     string  false  "Start date (YYYY-MM-DD)"
//	@Param        date_to    query     string  false  "End date (YYYY-MM-DD)"
//	@Param        cursor     query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   TimelineEntryResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/progress/timeline [get]
func (h *Handler) getActivityTimeline(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	query := TimelineQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if df := c.QueryParam("date_from"); df != "" {
		if t, err := time.Parse("2006-01-02", df); err == nil {
			query.DateFrom = &t
		}
	}
	if dt := c.QueryParam("date_to"); dt != "" {
		if t, err := time.Parse("2006-01-02", dt); err == nil {
			query.DateTo = &t
		}
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.GetActivityTimeline(c.Request().Context(), &scope, studentID, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Export Handlers ────────────────────────────────────────────────────────

// requestDataExport godoc
//
//	@Summary      Request a data export
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      RequestExportCommand  true  "Export request payload"
//	@Success      201   {object}  ExportRequestResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/export [post]
func (h *Handler) requestDataExport(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd RequestExportCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.RequestDataExport(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusAccepted, resp)
}

// getExportRequest godoc
//
//	@Summary      Get an export request status
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Export request UUID"
//	@Success      200  {object}  ExportRequestResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/export/{id} [get]
func (h *Handler) getExportRequest(c echo.Context) error {
	exportID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetExportRequest(c.Request().Context(), &scope, exportID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Tool Handlers ──────────────────────────────────────────────────────────

// getResolvedTools godoc
//
//	@Summary      Get resolved learning tools for the family
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200  {array}   ActiveToolResponse
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/tools [get]
func (h *Handler) getResolvedTools(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetResolvedTools(c.Request().Context(), &scope)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getStudentTools godoc
//
//	@Summary      Get learning tools for a specific student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Success      200  {array}   ActiveToolResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/tools [get]
func (h *Handler) getStudentTools(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetStudentTools(c.Request().Context(), &scope, studentID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Question Handlers ──────────────────────────────────────────────────────

// createQuestion godoc
//
//	@Summary      Create a question
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateQuestionCommand  true  "Question payload"
//	@Success      201   {object}  QuestionResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/questions [post]
func (h *Handler) createQuestion(c echo.Context) error {
	var cmd CreateQuestionCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateQuestion(c.Request().Context(), cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listQuestions godoc
//
//	@Summary      List questions
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        limit          query     int     false  "Page size (default 20, max 50)"
//	@Param        publisher_id   query     string  false  "Filter by publisher UUID"
//	@Param        question_type  query     string  false  "Filter by question type"
//	@Param        subject        query     string  false  "Filter by subject"
//	@Param        methodology_id query     string  false  "Filter by methodology UUID"
//	@Param        search         query     string  false  "Full-text search"
//	@Param        cursor         query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   QuestionSummaryResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/questions [get]
func (h *Handler) listQuestions(c echo.Context) error {
	query := QuestionQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("publisher_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			query.PublisherID = &id
		}
	}
	if s := c.QueryParam("question_type"); s != "" {
		query.QuestionType = &s
	}
	if s := c.QueryParam("subject"); s != "" {
		query.Subject = &s
	}
	if s := c.QueryParam("methodology_id"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			query.MethodologyID = &id
		}
	}
	if s := c.QueryParam("search"); s != "" {
		query.Search = &s
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListQuestions(c.Request().Context(), query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateQuestion godoc
//
//	@Summary      Update a question
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id    path      string                  true  "Question UUID"
//	@Param        body  body      UpdateQuestionCommand    true  "Fields to update"
//	@Success      200   {object}  QuestionResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      404   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/questions/{id} [patch]
func (h *Handler) updateQuestion(c echo.Context) error {
	questionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var cmd UpdateQuestionCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateQuestion(c.Request().Context(), questionID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Quiz Definition Handlers ──────────────────────────────────────────────

// createQuizDef godoc
//
//	@Summary      Create a quiz definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateQuizDefCommand  true  "Quiz definition payload"
//	@Success      201   {object}  QuizDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/quiz-defs [post]
func (h *Handler) createQuizDef(c echo.Context) error {
	var cmd CreateQuizDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateQuizDef(c.Request().Context(), cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// getQuizDef godoc
//
//	@Summary      Get a quiz definition by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id               path      string  true   "Quiz definition UUID"
//	@Param        include_answers  query     bool    false  "Include correct answers"
//	@Success      200  {object}  QuizDefDetailResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/quiz-defs/{id} [get]
func (h *Handler) getQuizDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	includeAnswers := c.QueryParam("include_answers") == "true"
	resp, err := h.svc.GetQuizDef(c.Request().Context(), defID, includeAnswers)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateQuizDef godoc
//
//	@Summary      Update a quiz definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id    path      string                 true  "Quiz definition UUID"
//	@Param        body  body      UpdateQuizDefCommand    true  "Fields to update"
//	@Success      200   {object}  QuizDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      404   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/quiz-defs/{id} [patch]
func (h *Handler) updateQuizDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var cmd UpdateQuizDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateQuizDef(c.Request().Context(), defID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Quiz Session Handlers ─────────────────────────────────────────────────

// startQuizSession godoc
//
//	@Summary      Start a quiz session for a student
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                    true  "Student UUID"
//	@Param        body       body      StartQuizSessionCommand    true  "Quiz session payload"
//	@Success      201        {object}  QuizSessionResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/quiz-sessions [post]
func (h *Handler) startQuizSession(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd StartQuizSessionCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.StartQuizSession(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// getQuizSession godoc
//
//	@Summary      Get a quiz session by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Quiz session UUID"
//	@Success      200  {object}  QuizSessionResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/quiz-sessions/{id} [get]
func (h *Handler) getQuizSession(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetQuizSession(c.Request().Context(), &scope, studentID, sessionID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateQuizSession godoc
//
//	@Summary      Update a quiz session
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                      true  "Student UUID"
//	@Param        id         path      string                      true  "Quiz session UUID"
//	@Param        body       body      UpdateQuizSessionCommand     true  "Fields to update"
//	@Success      200        {object}  QuizSessionResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      404        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/quiz-sessions/{id} [patch]
func (h *Handler) updateQuizSession(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateQuizSessionCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateQuizSession(c.Request().Context(), &scope, studentID, sessionID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// scoreQuizSession godoc
//
//	@Summary      Score a quiz session
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string            true  "Student UUID"
//	@Param        id         path      string            true  "Quiz session UUID"
//	@Param        body       body      ScoreQuizCommand   true  "Scoring payload"
//	@Success      200        {object}  QuizSessionResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      404        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/quiz-sessions/{id}/score [post]
func (h *Handler) scoreQuizSession(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd ScoreQuizCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.ScoreQuizSession(c.Request().Context(), &scope, studentID, sessionID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Sequence Definition Handlers ───────────────────────────────────────────

// createSequenceDef godoc
//
//	@Summary      Create a sequence definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateSequenceDefCommand  true  "Sequence definition payload"
//	@Success      201   {object}  SequenceDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/sequences [post]
func (h *Handler) createSequenceDef(c echo.Context) error {
	var cmd CreateSequenceDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return err
	}
	resp, err := h.svc.CreateSequenceDef(c.Request().Context(), cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// getSequenceDef godoc
//
//	@Summary      Get a sequence definition by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Sequence definition UUID"
//	@Success      200  {object}  SequenceDefDetailResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/sequences/{id} [get]
func (h *Handler) getSequenceDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid sequence def ID")
	}
	resp, err := h.svc.GetSequenceDef(c.Request().Context(), defID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateSequenceDef godoc
//
//	@Summary      Update a sequence definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id    path      string                      true  "Sequence definition UUID"
//	@Param        body  body      UpdateSequenceDefCommand     true  "Fields to update"
//	@Success      200   {object}  SequenceDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      404   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/sequences/{id} [patch]
func (h *Handler) updateSequenceDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid sequence def ID")
	}
	var cmd UpdateSequenceDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateSequenceDef(c.Request().Context(), defID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Sequence Progress Handlers ─────────────────────────────────────────────

// startSequence godoc
//
//	@Summary      Start a sequence for a student
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                  true  "Student UUID"
//	@Param        body       body      StartSequenceCommand     true  "Start sequence payload"
//	@Success      201        {object}  SequenceProgressResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/sequence-progress [post]
func (h *Handler) startSequence(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd StartSequenceCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return err
	}
	resp, err := h.svc.StartSequence(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// getSequenceProgress godoc
//
//	@Summary      Get sequence progress by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Sequence progress UUID"
//	@Success      200  {object}  SequenceProgressResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/sequence-progress/{id} [get]
func (h *Handler) getSequenceProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
	}
	progressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid progress ID")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetSequenceProgress(c.Request().Context(), &scope, studentID, progressID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateSequenceProgress godoc
//
//	@Summary      Update sequence progress
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                           true  "Student UUID"
//	@Param        id         path      string                           true  "Sequence progress UUID"
//	@Param        body       body      UpdateSequenceProgressCommand     true  "Fields to update"
//	@Success      200        {object}  SequenceProgressResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      404        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/sequence-progress/{id} [patch]
func (h *Handler) updateSequenceProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
	}
	progressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid progress ID")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateSequenceProgressCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateSequenceProgress(c.Request().Context(), &scope, studentID, progressID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Assignment Handlers ────────────────────────────────────────────────────

// createAssignment godoc
//
//	@Summary      Create an assignment for a student
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                    true  "Student UUID"
//	@Param        body       body      CreateAssignmentCommand    true  "Assignment payload"
//	@Success      201        {object}  AssignmentResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/assignments [post]
func (h *Handler) createAssignment(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd CreateAssignmentCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return err
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	cmd.AssignedBy = auth.ParentID
	resp, err := h.svc.CreateAssignment(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listAssignments godoc
//
//	@Summary      List assignments for a student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId   path      string  true   "Student UUID"
//	@Param        limit       query     int     false  "Page size (default 20, max 50)"
//	@Param        status      query     string  false  "Filter by status"
//	@Param        due_before  query     string  false  "Due before date (YYYY-MM-DD)"
//	@Param        cursor      query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   AssignmentResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/assignments [get]
func (h *Handler) listAssignments(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	query := AssignmentQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("status"); s != "" {
		query.Status = &s
	}
	if d := c.QueryParam("due_before"); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			query.DueBefore = &t
		}
	}
	if cursor := c.QueryParam("cursor"); cursor != "" {
		if id, err := uuid.Parse(cursor); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListAssignments(c.Request().Context(), &scope, studentID, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateAssignment godoc
//
//	@Summary      Update an assignment
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                     true  "Student UUID"
//	@Param        id         path      string                     true  "Assignment UUID"
//	@Param        body       body      UpdateAssignmentCommand     true  "Fields to update"
//	@Success      200        {object}  AssignmentResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      404        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/assignments/{id} [patch]
func (h *Handler) updateAssignment(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
	}
	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid assignment ID")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateAssignmentCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateAssignment(c.Request().Context(), &scope, studentID, assignmentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteAssignment godoc
//
//	@Summary      Delete an assignment
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Assignment UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/assignments/{id} [delete]
func (h *Handler) deleteAssignment(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
	}
	assignmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid assignment ID")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteAssignment(c.Request().Context(), &scope, studentID, assignmentID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Video Definition Handlers ──────────────────────────────────────────────

// listVideoDefs godoc
//
//	@Summary      List video definitions
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        limit          query     int     false  "Page size (default 20, max 50)"
//	@Param        subject        query     string  false  "Filter by subject"
//	@Param        methodology_id query     string  false  "Filter by methodology UUID"
//	@Param        publisher_id   query     string  false  "Filter by publisher UUID"
//	@Param        search         query     string  false  "Full-text search"
//	@Param        cursor         query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   VideoDefResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/videos [get]
func (h *Handler) listVideoDefs(c echo.Context) error {
	query := VideoDefQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("subject"); s != "" {
		query.Subject = &s
	}
	if m := c.QueryParam("methodology_id"); m != "" {
		if id, err := uuid.Parse(m); err == nil {
			query.MethodologyID = &id
		}
	}
	if p := c.QueryParam("publisher_id"); p != "" {
		if id, err := uuid.Parse(p); err == nil {
			query.PublisherID = &id
		}
	}
	if s := c.QueryParam("search"); s != "" {
		query.Search = &s
	}
	if cursor := c.QueryParam("cursor"); cursor != "" {
		if id, err := uuid.Parse(cursor); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListVideoDefs(c.Request().Context(), query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getVideoDef godoc
//
//	@Summary      Get a video definition by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Video definition UUID"
//	@Success      200  {object}  VideoDefResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/videos/{id} [get]
func (h *Handler) getVideoDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid video def ID")
	}
	resp, err := h.svc.GetVideoDef(c.Request().Context(), defID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Video Progress Handlers ────────────────────────────────────────────────

// updateVideoProgress godoc
//
//	@Summary      Update video progress for a student
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                       true  "Student UUID"
//	@Param        body       body      UpdateVideoProgressCommand    true  "Video progress payload"
//	@Success      200        {object}  VideoProgressResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/video-progress [patch]
func (h *Handler) updateVideoProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateVideoProgressCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(cmd); err != nil {
		return err
	}
	resp, err := h.svc.UpdateVideoProgress(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getVideoProgress godoc
//
//	@Summary      Get video progress for a student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId   path      string  true  "Student UUID"
//	@Param        videoDefId  path      string  true  "Video definition UUID"
//	@Success      200  {object}  VideoProgressResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/video-progress/{videoDefId} [get]
func (h *Handler) getVideoProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student ID")
	}
	videoDefID, err := uuid.Parse(c.Param("videoDefId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid video def ID")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetVideoProgress(c.Request().Context(), &scope, studentID, videoDefID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func parseLimit(s string) int64 {
	if s == "" {
		return 20
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n <= 0 {
		return 20
	}
	if n > 50 {
		return 50
	}
	return n
}

// ─── Assessment Definition Handlers (Phase 2) ───────────────────────────────

// createAssessmentDef godoc
//
//	@Summary      Create an assessment definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateAssessmentDefCommand  true  "Assessment definition payload"
//	@Success      201   {object}  AssessmentDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/assessment-defs [post]
func (h *Handler) createAssessmentDef(c echo.Context) error {
	var cmd CreateAssessmentDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	cmd.CallerID = auth.ParentID
	resp, err := h.svc.CreateAssessmentDef(c.Request().Context(), cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listAssessmentDefs godoc
//
//	@Summary      List assessment definitions
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        limit         query     int     false  "Page size (default 20, max 50)"
//	@Param        subject       query     string  false  "Filter by subject"
//	@Param        scoring_type  query     string  false  "Filter by scoring type"
//	@Param        publisher_id  query     string  false  "Filter by publisher UUID"
//	@Param        search        query     string  false  "Full-text search"
//	@Param        cursor        query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   AssessmentDefSummaryResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/assessment-defs [get]
func (h *Handler) listAssessmentDefs(c echo.Context) error {
	query := AssessmentDefQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("subject"); s != "" {
		query.Subject = &s
	}
	if st := c.QueryParam("scoring_type"); st != "" {
		query.ScoringType = &st
	}
	if p := c.QueryParam("publisher_id"); p != "" {
		if id, err := uuid.Parse(p); err == nil {
			query.PublisherID = &id
		}
	}
	if s := c.QueryParam("search"); s != "" {
		query.Search = &s
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListAssessmentDefs(c.Request().Context(), query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getAssessmentDef godoc
//
//	@Summary      Get an assessment definition by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Assessment definition UUID"
//	@Success      200  {object}  AssessmentDefResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/assessment-defs/{id} [get]
func (h *Handler) getAssessmentDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	resp, err := h.svc.GetAssessmentDef(c.Request().Context(), defID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateAssessmentDef godoc
//
//	@Summary      Update an assessment definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id    path      string                        true  "Assessment definition UUID"
//	@Param        body  body      UpdateAssessmentDefCommand     true  "Fields to update"
//	@Success      200   {object}  AssessmentDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      404   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/assessment-defs/{id} [patch]
func (h *Handler) updateAssessmentDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var cmd UpdateAssessmentDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	cmd.CallerID = auth.ParentID
	resp, err := h.svc.UpdateAssessmentDef(c.Request().Context(), defID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteAssessmentDef godoc
//
//	@Summary      Delete an assessment definition
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Assessment definition UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/assessment-defs/{id} [delete]
func (h *Handler) deleteAssessmentDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteAssessmentDef(c.Request().Context(), defID, auth.ParentID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Project Definition Handlers (Phase 2) ───────────────────────────────────

// createProjectDef godoc
//
//	@Summary      Create a project definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateProjectDefCommand  true  "Project definition payload"
//	@Success      201   {object}  ProjectDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/project-defs [post]
func (h *Handler) createProjectDef(c echo.Context) error {
	var cmd CreateProjectDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	cmd.CallerID = auth.ParentID
	resp, err := h.svc.CreateProjectDef(c.Request().Context(), cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listProjectDefs godoc
//
//	@Summary      List project definitions
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        limit         query     int     false  "Page size (default 20, max 50)"
//	@Param        subject       query     string  false  "Filter by subject"
//	@Param        publisher_id  query     string  false  "Filter by publisher UUID"
//	@Param        search        query     string  false  "Full-text search"
//	@Param        cursor        query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   ProjectDefSummaryResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/project-defs [get]
func (h *Handler) listProjectDefs(c echo.Context) error {
	query := ProjectDefQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("subject"); s != "" {
		query.Subject = &s
	}
	if p := c.QueryParam("publisher_id"); p != "" {
		if id, err := uuid.Parse(p); err == nil {
			query.PublisherID = &id
		}
	}
	if s := c.QueryParam("search"); s != "" {
		query.Search = &s
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListProjectDefs(c.Request().Context(), query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getProjectDef godoc
//
//	@Summary      Get a project definition by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Project definition UUID"
//	@Success      200  {object}  ProjectDefResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/project-defs/{id} [get]
func (h *Handler) getProjectDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	resp, err := h.svc.GetProjectDef(c.Request().Context(), defID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateProjectDef godoc
//
//	@Summary      Update a project definition
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id    path      string                     true  "Project definition UUID"
//	@Param        body  body      UpdateProjectDefCommand     true  "Fields to update"
//	@Success      200   {object}  ProjectDefResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      404   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/project-defs/{id} [patch]
func (h *Handler) updateProjectDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	var cmd UpdateProjectDefCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	cmd.CallerID = auth.ParentID
	resp, err := h.svc.UpdateProjectDef(c.Request().Context(), defID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteProjectDef godoc
//
//	@Summary      Delete a project definition
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Project definition UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/project-defs/{id} [delete]
func (h *Handler) deleteProjectDef(c echo.Context) error {
	defID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteProjectDef(c.Request().Context(), defID, auth.ParentID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Assessment Result Handlers (Phase 2) ────────────────────────────────────

// recordAssessmentResult godoc
//
//	@Summary      Record an assessment result for a student
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                          true  "Student UUID"
//	@Param        body       body      RecordAssessmentResultCommand    true  "Assessment result payload"
//	@Success      201        {object}  AssessmentResultResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/assessments [post]
func (h *Handler) recordAssessmentResult(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd RecordAssessmentResultCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.RecordAssessmentResult(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listAssessmentResults godoc
//
//	@Summary      List assessment results for a student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId          path      string  true   "Student UUID"
//	@Param        limit              query     int     false  "Page size (default 20, max 50)"
//	@Param        assessment_def_id  query     string  false  "Filter by assessment definition UUID"
//	@Param        cursor             query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   AssessmentResultResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/assessments [get]
func (h *Handler) listAssessmentResults(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	query := AssessmentResultQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if d := c.QueryParam("assessment_def_id"); d != "" {
		if id, err := uuid.Parse(d); err == nil {
			query.AssessmentDefID = &id
		}
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListAssessmentResults(c.Request().Context(), &scope, studentID, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getAssessmentResult godoc
//
//	@Summary      Get an assessment result by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Assessment result UUID"
//	@Success      200  {object}  AssessmentResultResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/assessments/{id} [get]
func (h *Handler) getAssessmentResult(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	resultID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetAssessmentResult(c.Request().Context(), &scope, studentID, resultID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateAssessmentResult godoc
//
//	@Summary      Update an assessment result
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                            true  "Student UUID"
//	@Param        id         path      string                            true  "Assessment result UUID"
//	@Param        body       body      UpdateAssessmentResultCommand      true  "Fields to update"
//	@Success      200        {object}  AssessmentResultResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      404        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/assessments/{id} [patch]
func (h *Handler) updateAssessmentResult(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	resultID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateAssessmentResultCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	resp, err := h.svc.UpdateAssessmentResult(c.Request().Context(), &scope, studentID, resultID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteAssessmentResult godoc
//
//	@Summary      Delete an assessment result
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Assessment result UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/assessments/{id} [delete]
func (h *Handler) deleteAssessmentResult(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	resultID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteAssessmentResult(c.Request().Context(), &scope, studentID, resultID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Project Progress Handlers (Phase 2) ─────────────────────────────────────

// startProject godoc
//
//	@Summary      Start a project for a student
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string               true  "Student UUID"
//	@Param        body       body      StartProjectCommand   true  "Start project payload"
//	@Success      201        {object}  ProjectProgressResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/projects [post]
func (h *Handler) startProject(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd StartProjectCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.StartProject(c.Request().Context(), &scope, studentID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listProjectProgress godoc
//
//	@Summary      List project progress for a student
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId       path      string  true   "Student UUID"
//	@Param        limit           query     int     false  "Page size (default 20, max 50)"
//	@Param        status          query     string  false  "Filter by status"
//	@Param        project_def_id  query     string  false  "Filter by project definition UUID"
//	@Param        cursor          query     string  false  "Cursor UUID for pagination"
//	@Success      200  {array}   ProjectProgressResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/projects [get]
func (h *Handler) listProjectProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	query := ProjectProgressQuery{
		Limit: parseLimit(c.QueryParam("limit")),
	}
	if s := c.QueryParam("status"); s != "" {
		query.Status = &s
	}
	if d := c.QueryParam("project_def_id"); d != "" {
		if id, err := uuid.Parse(d); err == nil {
			query.ProjectDefID = &id
		}
	}
	if cur := c.QueryParam("cursor"); cur != "" {
		if id, err := uuid.Parse(cur); err == nil {
			query.Cursor = &id
		}
	}
	resp, err := h.svc.ListProjectProgress(c.Request().Context(), &scope, studentID, query)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getProjectProgress godoc
//
//	@Summary      Get project progress by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Project progress UUID"
//	@Success      200  {object}  ProjectProgressResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/projects/{id} [get]
func (h *Handler) getProjectProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	progressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetProjectProgress(c.Request().Context(), &scope, studentID, progressID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateProjectProgress godoc
//
//	@Summary      Update project progress
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string                          true  "Student UUID"
//	@Param        id         path      string                          true  "Project progress UUID"
//	@Param        body       body      UpdateProjectProgressCommand     true  "Fields to update"
//	@Success      200        {object}  ProjectProgressResponse
//	@Failure      400        {object}  shared.AppError
//	@Failure      401        {object}  shared.AppError
//	@Failure      404        {object}  shared.AppError
//	@Failure      500        {object}  shared.AppError
//	@Router       /learning/students/{studentId}/projects/{id} [patch]
func (h *Handler) updateProjectProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	progressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateProjectProgressCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	resp, err := h.svc.UpdateProjectProgress(c.Request().Context(), &scope, studentID, progressID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteProjectProgress godoc
//
//	@Summary      Delete project progress
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        studentId  path      string  true  "Student UUID"
//	@Param        id         path      string  true  "Project progress UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/students/{studentId}/projects/{id} [delete]
func (h *Handler) deleteProjectProgress(c echo.Context) error {
	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid student id")
	}
	progressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteProjectProgress(c.Request().Context(), &scope, studentID, progressID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Grading Scale Handlers (Phase 2) ────────────────────────────────────────

// createGradingScale godoc
//
//	@Summary      Create a grading scale
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body  body      CreateGradingScaleCommand  true  "Grading scale payload"
//	@Success      201   {object}  GradingScaleResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/grading-scales [post]
func (h *Handler) createGradingScale(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd CreateGradingScaleCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateGradingScale(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// listGradingScales godoc
//
//	@Summary      List grading scales for the family
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200  {array}   GradingScaleResponse
//	@Failure      401  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/grading-scales [get]
func (h *Handler) listGradingScales(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.ListGradingScales(c.Request().Context(), &scope)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getGradingScale godoc
//
//	@Summary      Get a grading scale by ID
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Grading scale UUID"
//	@Success      200  {object}  GradingScaleResponse
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/grading-scales/{id} [get]
func (h *Handler) getGradingScale(c echo.Context) error {
	scaleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetGradingScale(c.Request().Context(), &scope, scaleID)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// updateGradingScale godoc
//
//	@Summary      Update a grading scale
//	@Tags         learn
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id    path      string                       true  "Grading scale UUID"
//	@Param        body  body      UpdateGradingScaleCommand     true  "Fields to update"
//	@Success      200   {object}  GradingScaleResponse
//	@Failure      400   {object}  shared.AppError
//	@Failure      401   {object}  shared.AppError
//	@Failure      404   {object}  shared.AppError
//	@Failure      500   {object}  shared.AppError
//	@Router       /learning/grading-scales/{id} [patch]
func (h *Handler) updateGradingScale(c echo.Context) error {
	scaleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateGradingScaleCommand
	if err := c.Bind(&cmd); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	resp, err := h.svc.UpdateGradingScale(c.Request().Context(), &scope, scaleID, cmd)
	if err != nil {
		return mapLearningError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// deleteGradingScale godoc
//
//	@Summary      Delete a grading scale
//	@Tags         learn
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path      string  true  "Grading scale UUID"
//	@Success      204  "No Content"
//	@Failure      400  {object}  shared.AppError
//	@Failure      401  {object}  shared.AppError
//	@Failure      404  {object}  shared.AppError
//	@Failure      500  {object}  shared.AppError
//	@Router       /learning/grading-scales/{id} [delete]
func (h *Handler) deleteGradingScale(c echo.Context) error {
	scaleID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid id")
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	if err := h.svc.DeleteGradingScale(c.Request().Context(), &scope, scaleID); err != nil {
		return mapLearningError(err)
	}
	return c.NoContent(http.StatusNoContent)
}
