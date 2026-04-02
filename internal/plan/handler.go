package plan

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Handler provides HTTP route handlers for the planning domain. [17-planning §4]
type Handler struct {
	svc PlanningService
}

// NewHandler creates a new planning domain handler.
func NewHandler(svc PlanningService) *Handler {
	return &Handler{svc: svc}
}

// Register registers planning routes on the authenticated group. [17-planning §4]
func (h *Handler) Register(auth *echo.Group) {
	g := auth.Group("/planning")

	// Calendar views
	g.GET("/calendar", h.getCalendar)
	g.GET("/calendar/day/:date", h.getDayView)
	g.GET("/calendar/week/:date", h.getWeekView)
	g.GET("/calendar/print", h.getPrintView)

	// Schedule items
	g.POST("/schedule-items", h.createScheduleItem)
	g.GET("/schedule-items", h.listScheduleItems)
	g.GET("/schedule-items/:id", h.getScheduleItem)
	g.PATCH("/schedule-items/:id", h.updateScheduleItem)
	g.DELETE("/schedule-items/:id", h.deleteScheduleItem)
	g.PATCH("/schedule-items/:id/complete", h.completeScheduleItem)
	g.POST("/schedule-items/:id/log", h.logAsActivity)

	// Schedule templates (Phase 2)
	g.GET("/templates", h.listTemplates)
	g.POST("/templates", h.createTemplate)
	g.PATCH("/templates/:id", h.updateTemplate)
	g.DELETE("/templates/:id", h.deleteTemplate)
	g.POST("/templates/:id/apply", h.applyTemplate)
}

// ─── Calendar Views ──────────────────────────────────────────────────────────

// getCalendar godoc
//
//	@Summary     Get aggregated calendar
//	@Tags        planning
//	@Produce     json
//	@Security    BearerAuth
//	@Param       start      query string false "Start date (YYYY-MM-DD)"
//	@Param       end        query string false "End date (YYYY-MM-DD)"
//	@Param       student_id query string false "Filter by student ID"
//	@Success     200 {object} CalendarResponse
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Router      /planning/calendar [get]
func (h *Handler) getCalendar(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	params, err := parseCalendarQuery(c)
	if err != nil {
		return err
	}

	result, err := h.svc.GetCalendar(c.Request().Context(), auth, &scope, params)
	if err != nil {
		return mapPlanError(err)
	}
	return c.JSON(http.StatusOK, result)
}

// getDayView godoc
//
//	@Summary     Get day view
//	@Tags        planning
//	@Produce     json
//	@Security    BearerAuth
//	@Param       date       path  string true  "Date (YYYY-MM-DD)"
//	@Param       student_id query string false "Filter by student ID"
//	@Success     200 {object} DayViewResponse
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Router      /planning/calendar/day/{date} [get]
func (h *Handler) getDayView(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		return shared.ErrBadRequest("invalid date format, expected YYYY-MM-DD")
	}

	studentID := parseOptionalUUID(c.QueryParam("student_id"))

	result, err := h.svc.GetDayView(c.Request().Context(), auth, &scope, date, studentID)
	if err != nil {
		return mapPlanError(err)
	}
	return c.JSON(http.StatusOK, result)
}

// getWeekView godoc
//
//	@Summary     Get week view
//	@Tags        planning
//	@Produce     json
//	@Security    BearerAuth
//	@Param       date       path  string true  "Any date within the week (YYYY-MM-DD)"
//	@Param       student_id query string false "Filter by student ID"
//	@Success     200 {object} WeekViewResponse
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Router      /planning/calendar/week/{date} [get]
func (h *Handler) getWeekView(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	date, err := time.Parse("2006-01-02", c.Param("date"))
	if err != nil {
		return shared.ErrBadRequest("invalid date format, expected YYYY-MM-DD")
	}

	studentID := parseOptionalUUID(c.QueryParam("student_id"))

	// Calculate week boundaries (Monday–Sunday).
	weekday := int(date.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	weekStart := date.AddDate(0, 0, -(weekday - 1))
	weekEnd := weekStart.AddDate(0, 0, 7)

	calParams := CalendarQuery{
		Start:     weekStart,
		End:       weekEnd,
		StudentID: studentID,
	}

	result, err := h.svc.GetCalendar(c.Request().Context(), auth, &scope, calParams)
	if err != nil {
		return mapPlanError(err)
	}

	return c.JSON(http.StatusOK, WeekViewResponse{
		WeekStart: weekStart,
		WeekEnd:   weekEnd,
		Days:      result.Days,
	})
}

// getPrintView godoc
//
//	@Summary     Get print-friendly calendar
//	@Tags        planning
//	@Produce     html
//	@Security    BearerAuth
//	@Param       start      query string true  "Start date (YYYY-MM-DD)"
//	@Param       end        query string true  "End date (YYYY-MM-DD)"
//	@Param       student_id query string false "Filter by student ID"
//	@Success     200 {string} string "HTML content"
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Router      /planning/calendar/print [get]
func (h *Handler) getPrintView(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	start, err := time.Parse("2006-01-02", c.QueryParam("start"))
	if err != nil {
		return shared.ErrBadRequest("invalid start date, expected YYYY-MM-DD")
	}
	end, err := time.Parse("2006-01-02", c.QueryParam("end"))
	if err != nil {
		return shared.ErrBadRequest("invalid end date, expected YYYY-MM-DD")
	}

	studentID := parseOptionalUUID(c.QueryParam("student_id"))

	html, err := h.svc.GetPrintView(c.Request().Context(), auth, &scope, start, end, studentID)
	if err != nil {
		return mapPlanError(err)
	}
	return c.HTML(http.StatusOK, html)
}

// ─── Schedule Items ──────────────────────────────────────────────────────────

// createScheduleItem godoc
//
//	@Summary     Create a schedule item
//	@Tags        planning
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       body body CreateScheduleItemInput true "Schedule item details"
//	@Success     201 {object} map[string]uuid.UUID
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Router      /planning/schedule-items [post]
func (h *Handler) createScheduleItem(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var input CreateScheduleItemInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	id, err := h.svc.CreateScheduleItem(c.Request().Context(), auth, &scope, input)
	if err != nil {
		return mapPlanError(err)
	}
	return c.JSON(http.StatusCreated, map[string]uuid.UUID{"id": id})
}

// listScheduleItems godoc
//
//	@Summary     List schedule items
//	@Tags        planning
//	@Produce     json
//	@Security    BearerAuth
//	@Param       start_date   query string false "Start date filter (YYYY-MM-DD)"
//	@Param       end_date     query string false "End date filter (YYYY-MM-DD)"
//	@Param       student_id   query string false "Filter by student ID"
//	@Param       category     query string false "Filter by category"
//	@Param       is_completed query bool   false "Filter by completion status"
//	@Param       cursor       query string false "Pagination cursor"
//	@Param       limit        query int    false "Results per page (default 20, max 100)"
//	@Success     200 {object} ScheduleItemListResponse
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Router      /planning/schedule-items [get]
func (h *Handler) listScheduleItems(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	query := parseScheduleItemQuery(c)

	var pagination shared.PaginationParams
	if err := c.Bind(&pagination); err != nil {
		return shared.ErrBadRequest("invalid pagination parameters")
	}
	if err := c.Validate(&pagination); err != nil {
		return shared.ValidationError(err)
	}

	result, err := h.svc.ListScheduleItems(c.Request().Context(), auth, &scope, query, &pagination)
	if err != nil {
		return mapPlanError(err)
	}
	return c.JSON(http.StatusOK, result)
}

// getScheduleItem godoc
//
//	@Summary     Get a schedule item
//	@Tags        planning
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id path string true "Schedule item ID"
//	@Success     200 {object} ScheduleItemResponse
//	@Failure     401 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Router      /planning/schedule-items/{id} [get]
func (h *Handler) getScheduleItem(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid schedule item ID")
	}

	result, err := h.svc.GetScheduleItem(c.Request().Context(), auth, &scope, itemID)
	if err != nil {
		return mapPlanError(err)
	}
	return c.JSON(http.StatusOK, result)
}

// updateScheduleItem godoc
//
//	@Summary     Update a schedule item
//	@Tags        planning
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id   path string                true "Schedule item ID"
//	@Param       body body UpdateScheduleItemInput true "Fields to update"
//	@Success     204
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Router      /planning/schedule-items/{id} [patch]
func (h *Handler) updateScheduleItem(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid schedule item ID")
	}

	var input UpdateScheduleItemInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&input); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.UpdateScheduleItem(c.Request().Context(), auth, &scope, itemID, input); err != nil {
		return mapPlanError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// deleteScheduleItem godoc
//
//	@Summary     Delete a schedule item
//	@Tags        planning
//	@Security    BearerAuth
//	@Param       id path string true "Schedule item ID"
//	@Success     204
//	@Failure     401 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Router      /planning/schedule-items/{id} [delete]
func (h *Handler) deleteScheduleItem(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid schedule item ID")
	}

	if err := h.svc.DeleteScheduleItem(c.Request().Context(), auth, &scope, itemID); err != nil {
		return mapPlanError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// completeScheduleItem godoc
//
//	@Summary     Mark a schedule item as completed
//	@Tags        planning
//	@Security    BearerAuth
//	@Param       id path string true "Schedule item ID"
//	@Success     204
//	@Failure     401 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Failure     409 {object} shared.AppError
//	@Router      /planning/schedule-items/{id}/complete [patch]
func (h *Handler) completeScheduleItem(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid schedule item ID")
	}

	if err := h.svc.CompleteScheduleItem(c.Request().Context(), auth, &scope, itemID); err != nil {
		return mapPlanError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// logAsActivity godoc
//
//	@Summary     Log a completed schedule item as a learning activity
//	@Tags        planning
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id   path string           true "Schedule item ID"
//	@Param       body body LogAsActivityInput true "Activity details"
//	@Success     201 {object} map[string]uuid.UUID
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Failure     409 {object} shared.AppError
//	@Router      /planning/schedule-items/{id}/log [post]
func (h *Handler) logAsActivity(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	itemID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid schedule item ID")
	}

	var input LogAsActivityInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&input); err != nil {
		return shared.ValidationError(err)
	}

	activityID, err := h.svc.LogAsActivity(c.Request().Context(), auth, &scope, itemID, input)
	if err != nil {
		return mapPlanError(err)
	}
	return c.JSON(http.StatusCreated, map[string]uuid.UUID{"activity_id": activityID})
}

// ─── Schedule Templates ──────────────────────────────────────────────────────

// listTemplates godoc
//
//	@Summary     List schedule templates
//	@Tags        planning
//	@Produce     json
//	@Security    BearerAuth
//	@Success     200 {array}  TemplateResponse
//	@Failure     401 {object} shared.AppError
//	@Router      /planning/templates [get]
func (h *Handler) listTemplates(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	result, err := h.svc.ListTemplates(c.Request().Context(), auth, &scope)
	if err != nil {
		return mapPlanError(err)
	}
	return c.JSON(http.StatusOK, result)
}

// createTemplate godoc
//
//	@Summary     Create a schedule template
//	@Tags        planning
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       body body CreateTemplateInput true "Template details"
//	@Success     201 {object} map[string]uuid.UUID
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Router      /planning/templates [post]
func (h *Handler) createTemplate(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	var input CreateTemplateInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	id, err := h.svc.CreateTemplate(c.Request().Context(), auth, &scope, input)
	if err != nil {
		return mapPlanError(err)
	}
	return c.JSON(http.StatusCreated, map[string]uuid.UUID{"id": id})
}

// updateTemplate godoc
//
//	@Summary     Update a schedule template
//	@Tags        planning
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id   path string              true "Template ID"
//	@Param       body body UpdateTemplateInput true "Fields to update"
//	@Success     204
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Router      /planning/templates/{id} [patch]
func (h *Handler) updateTemplate(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid template ID")
	}

	var input UpdateTemplateInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&input); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.UpdateTemplate(c.Request().Context(), auth, &scope, templateID, input); err != nil {
		return mapPlanError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// deleteTemplate godoc
//
//	@Summary     Delete a schedule template
//	@Tags        planning
//	@Security    BearerAuth
//	@Param       id path string true "Template ID"
//	@Success     204
//	@Failure     401 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Router      /planning/templates/{id} [delete]
func (h *Handler) deleteTemplate(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid template ID")
	}

	if err := h.svc.DeleteTemplate(c.Request().Context(), auth, &scope, templateID); err != nil {
		return mapPlanError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// applyTemplate godoc
//
//	@Summary     Apply a template to create schedule items for a date range
//	@Tags        planning
//	@Accept      json
//	@Produce     json
//	@Security    BearerAuth
//	@Param       id   path string             true "Template ID"
//	@Param       body body ApplyTemplateInput true "Date range to apply"
//	@Success     201 {object} map[string][]uuid.UUID
//	@Failure     400 {object} shared.AppError
//	@Failure     401 {object} shared.AppError
//	@Failure     404 {object} shared.AppError
//	@Router      /planning/templates/{id}/apply [post]
func (h *Handler) applyTemplate(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	templateID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid template ID")
	}

	var input ApplyTemplateInput
	if err := c.Bind(&input); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(input); err != nil {
		return shared.ValidationError(err)
	}

	ids, err := h.svc.ApplyTemplate(c.Request().Context(), auth, &scope, templateID, input)
	if err != nil {
		return mapPlanError(err)
	}
	return c.JSON(http.StatusCreated, map[string][]uuid.UUID{"created_ids": ids})
}

// ─── Error Mapping [17-planning §17] ─────────────────────────────────────────

func mapPlanError(err error) error {
	switch {
	case errors.Is(err, ErrItemNotFound):
		return shared.ErrNotFound()
	case errors.Is(err, ErrTemplateNotFound):
		return shared.ErrNotFound()
	case errors.Is(err, ErrStudentNotInFamily):
		return shared.ErrNotFound()
	case errors.Is(err, ErrInvalidDateRange):
		return shared.ErrBadRequest(err.Error())
	case errors.Is(err, ErrDateRangeTooLarge):
		return shared.ErrBadRequest(err.Error())
	case errors.Is(err, ErrAlreadyCompleted):
		return shared.ErrConflict(err.Error())
	case errors.Is(err, ErrAlreadyLogged):
		return shared.ErrConflict(err.Error())
	case errors.Is(err, ErrNotCompleted):
		return shared.ErrBadRequest(err.Error())
	default:
		return shared.ErrInternal(err)
	}
}

// ─── Request Parsing Helpers ─────────────────────────────────────────────────

func parseCalendarQuery(c echo.Context) (CalendarQuery, error) {
	var params CalendarQuery

	startStr := c.QueryParam("start")
	endStr := c.QueryParam("end")

	if startStr == "" || endStr == "" {
		// Default to current week if no dates provided.
		now := time.Now().UTC()
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		params.Start = now.AddDate(0, 0, -(weekday - 1)).Truncate(24 * time.Hour)
		params.End = params.Start.AddDate(0, 0, 7)
	} else {
		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			return CalendarQuery{}, shared.ErrBadRequest("invalid start date, expected YYYY-MM-DD")
		}
		end, err := time.Parse("2006-01-02", endStr)
		if err != nil {
			return CalendarQuery{}, shared.ErrBadRequest("invalid end date, expected YYYY-MM-DD")
		}
		params.Start = start
		params.End = end
	}

	params.StudentID = parseOptionalUUID(c.QueryParam("student_id"))
	return params, nil
}

func parseScheduleItemQuery(c echo.Context) ScheduleItemQuery {
	var query ScheduleItemQuery

	if s := c.QueryParam("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			query.StartDate = &t
		}
	}
	if s := c.QueryParam("end_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			query.EndDate = &t
		}
	}
	query.StudentID = parseOptionalUUID(c.QueryParam("student_id"))
	if s := c.QueryParam("category"); s != "" {
		cat := ScheduleCategory(s)
		query.Category = &cat
	}
	if s := c.QueryParam("is_completed"); s == "true" {
		v := true
		query.IsCompleted = &v
	} else if s == "false" {
		v := false
		query.IsCompleted = &v
	}

	return query
}

func parseOptionalUUID(s string) *uuid.UUID {
	if s == "" {
		return nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return nil
	}
	return &id
}
