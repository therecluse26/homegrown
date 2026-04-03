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
