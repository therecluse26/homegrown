package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"golang.org/x/sync/errgroup"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Implementation [17-planning §5]
// ═══════════════════════════════════════════════════════════════════════════════

// PlanningServiceImpl implements PlanningService.
type PlanningServiceImpl struct {
	scheduleRepo ScheduleItemRepository
	templateRepo ScheduleTemplateRepository
	iamSvc       IamServiceForPlan
	learnSvc     LearningServiceForPlan
	complySvc    ComplianceServiceForPlan
	socialSvc    SocialServiceForPlan
}

// NewPlanningService creates a PlanningService with all required dependencies.
func NewPlanningService(
	scheduleRepo ScheduleItemRepository,
	templateRepo ScheduleTemplateRepository,
	iamSvc IamServiceForPlan,
	learnSvc LearningServiceForPlan,
	complySvc ComplianceServiceForPlan,
	socialSvc SocialServiceForPlan,
) PlanningService {
	return &PlanningServiceImpl{
		scheduleRepo: scheduleRepo,
		templateRepo: templateRepo,
		iamSvc:       iamSvc,
		learnSvc:     learnSvc,
		complySvc:    complySvc,
		socialSvc:    socialSvc,
	}
}

// maxCalendarRangeDays is the maximum date range for calendar queries. [17-planning §17]
const maxCalendarRangeDays = 90

// ─── Schedule Item CRUD ──────────────────────────────────────────────────────

func (s *PlanningServiceImpl) CreateScheduleItem(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	input CreateScheduleItemInput,
) (uuid.UUID, error) {
	// Validate student belongs to family if specified. [17-planning §17]
	if input.StudentID != nil {
		belongs, err := s.iamSvc.StudentBelongsToFamily(ctx, *input.StudentID, scope.FamilyID())
		if err != nil {
			return uuid.Nil, fmt.Errorf("plan: check student membership: %w", err)
		}
		if !belongs {
			return uuid.Nil, ErrStudentNotInFamily
		}
	}

	// Default category to "custom" when nil. [17-planning §7]
	category := ScheduleCategoryCustom
	if input.Category != nil {
		category = *input.Category
	}

	item := &ScheduleItem{
		ID:              uuid.Must(uuid.NewV7()),
		FamilyID:        scope.FamilyID(),
		StudentID:       input.StudentID,
		Title:           input.Title,
		Description:     input.Description,
		StartDate:       input.StartDate,
		StartTime:       input.StartTime,
		EndTime:         input.EndTime,
		DurationMinutes: input.DurationMinutes,
		Category:        category,
		SubjectID:       input.SubjectID,
		Color:           input.Color,
		Notes:           input.Notes,
	}

	if err := s.scheduleRepo.Create(ctx, scope, item); err != nil {
		return uuid.Nil, fmt.Errorf("plan: create schedule item: %w", err)
	}

	return item.ID, nil
}

func (s *PlanningServiceImpl) UpdateScheduleItem(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	itemID uuid.UUID,
	input UpdateScheduleItemInput,
) error {
	existing, err := s.scheduleRepo.FindByID(ctx, scope, itemID)
	if err != nil {
		return fmt.Errorf("plan: find schedule item: %w", err)
	}
	if existing == nil {
		return ErrItemNotFound
	}

	if err := s.scheduleRepo.Update(ctx, scope, itemID, &input); err != nil {
		return fmt.Errorf("plan: update schedule item: %w", err)
	}
	return nil
}

func (s *PlanningServiceImpl) DeleteScheduleItem(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	itemID uuid.UUID,
) error {
	existing, err := s.scheduleRepo.FindByID(ctx, scope, itemID)
	if err != nil {
		return fmt.Errorf("plan: find schedule item: %w", err)
	}
	if existing == nil {
		return ErrItemNotFound
	}

	if err := s.scheduleRepo.Delete(ctx, scope, itemID); err != nil {
		return fmt.Errorf("plan: delete schedule item: %w", err)
	}
	return nil
}

// ─── Single Item ─────────────────────────────────────────────────────────────

func (s *PlanningServiceImpl) GetScheduleItem(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	itemID uuid.UUID,
) (ScheduleItemResponse, error) {
	item, err := s.scheduleRepo.FindByID(ctx, scope, itemID)
	if err != nil {
		return ScheduleItemResponse{}, fmt.Errorf("plan: find schedule item: %w", err)
	}
	if item == nil {
		return ScheduleItemResponse{}, ErrItemNotFound
	}

	resp := toScheduleItemResponse(*item)
	if item.StudentID != nil {
		name, nameErr := s.iamSvc.GetStudentName(ctx, *item.StudentID)
		if nameErr == nil {
			resp.StudentName = &name
		}
	}
	return resp, nil
}

// ─── Completion Workflow ─────────────────────────────────────────────────────

func (s *PlanningServiceImpl) CompleteScheduleItem(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	itemID uuid.UUID,
) error {
	existing, err := s.scheduleRepo.FindByID(ctx, scope, itemID)
	if err != nil {
		return fmt.Errorf("plan: find schedule item: %w", err)
	}
	if existing == nil {
		return ErrItemNotFound
	}
	if existing.IsCompleted {
		return ErrAlreadyCompleted
	}

	if err := s.scheduleRepo.MarkCompleted(ctx, scope, itemID, time.Now()); err != nil {
		return fmt.Errorf("plan: mark completed: %w", err)
	}
	return nil
}

// ─── Log as Activity ─────────────────────────────────────────────────────────

func (s *PlanningServiceImpl) LogAsActivity(
	ctx context.Context,
	auth *shared.AuthContext,
	scope *shared.FamilyScope,
	itemID uuid.UUID,
	input LogAsActivityInput,
) (uuid.UUID, error) {
	existing, err := s.scheduleRepo.FindByID(ctx, scope, itemID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("plan: find schedule item: %w", err)
	}
	if existing == nil {
		return uuid.Nil, ErrItemNotFound
	}
	if !existing.IsCompleted {
		return uuid.Nil, ErrNotCompleted
	}
	if existing.LinkedActivityID != nil {
		return uuid.Nil, ErrAlreadyLogged
	}

	// Create activity in learn:: via cross-domain call. [17-planning §10.2]
	activityID, err := s.learnSvc.LogActivity(
		ctx, auth, scope,
		existing.Title,
		existing.StartDate,
		existing.DurationMinutes,
		existing.StudentID,
		input.Description,
		input.Tags,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("plan: log activity: %w", err)
	}

	// Link the activity back to the schedule item.
	if err := s.scheduleRepo.SetLinkedActivity(ctx, scope, itemID, activityID); err != nil {
		return uuid.Nil, fmt.Errorf("plan: set linked activity: %w", err)
	}

	return activityID, nil
}

// ─── List / Query ────────────────────────────────────────────────────────────

func (s *PlanningServiceImpl) ListScheduleItems(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	params ScheduleItemQuery,
	pagination *shared.PaginationParams,
) (*shared.PaginatedResponse[ScheduleItemResponse], error) {
	items, err := s.scheduleRepo.ListFiltered(ctx, scope, &params, pagination)
	if err != nil {
		return nil, fmt.Errorf("plan: list schedule items: %w", err)
	}

	responses := make([]ScheduleItemResponse, len(items))
	for i, item := range items {
		responses[i] = toScheduleItemResponse(item)
	}

	return &shared.PaginatedResponse[ScheduleItemResponse]{
		Data:    responses,
		HasMore: false, // simplified for Phase 1; repo can handle cursor logic
	}, nil
}

// ─── Calendar View ───────────────────────────────────────────────────────────

func (s *PlanningServiceImpl) GetCalendar(
	ctx context.Context,
	auth *shared.AuthContext,
	scope *shared.FamilyScope,
	params CalendarQuery,
) (CalendarResponse, error) {
	if err := validateDateRange(params.Start, params.End); err != nil {
		return CalendarResponse{}, err
	}

	wantSource := sourceFilter(params.Sources)

	// Parallel fetch from all requested sources using errgroup. [17-planning §9.2]
	var (
		mu             sync.Mutex
		scheduleItems  []ScheduleItem
		activities     []ActivitySummary
		attendance     []AttendanceSummary
		events         []EventSummary
	)

	g, gCtx := errgroup.WithContext(ctx)

	if wantSource(CalendarSourceSchedule) {
		g.Go(func() error {
			items, err := s.scheduleRepo.ListByDateRange(gCtx, scope, params.Start, params.End, params.StudentID)
			if err != nil {
				return fmt.Errorf("plan: list schedule items: %w", err)
			}
			mu.Lock()
			scheduleItems = items
			mu.Unlock()
			return nil
		})
	}

	if wantSource(CalendarSourceActivities) {
		g.Go(func() error {
			acts, err := s.learnSvc.ListActivitiesForCalendar(gCtx, auth, scope, params.Start, params.End, params.StudentID)
			if err != nil {
				return fmt.Errorf("plan: list activities: %w", err)
			}
			mu.Lock()
			activities = acts
			mu.Unlock()
			return nil
		})
	}

	if wantSource(CalendarSourceAttendance) {
		g.Go(func() error {
			att, err := s.complySvc.GetAttendanceRange(gCtx, auth, scope, params.Start, params.End, params.StudentID)
			if err != nil {
				return fmt.Errorf("plan: get attendance: %w", err)
			}
			mu.Lock()
			attendance = att
			mu.Unlock()
			return nil
		})
	}

	if wantSource(CalendarSourceEvents) {
		g.Go(func() error {
			evts, err := s.socialSvc.GetEventsForCalendar(gCtx, auth, scope, params.Start, params.End)
			if err != nil {
				return fmt.Errorf("plan: get events: %w", err)
			}
			mu.Lock()
			events = evts
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return CalendarResponse{}, err
	}

	studentNames := s.resolveStudentNames(ctx, scheduleItems, activities)
	days := mergeIntoCalendarDays(params.Start, params.End, scheduleItems, activities, attendance, events, studentNames)

	return CalendarResponse{
		Start: params.Start,
		End:   params.End,
		Days:  days,
	}, nil
}

// ─── Day View ────────────────────────────────────────────────────────────────

func (s *PlanningServiceImpl) GetDayView(
	ctx context.Context,
	auth *shared.AuthContext,
	scope *shared.FamilyScope,
	date time.Time,
	studentID *uuid.UUID,
) (DayViewResponse, error) {
	dayStart := truncateToDate(date)
	dayEnd := dayStart.Add(24 * time.Hour)

	// Fetch schedule items.
	items, err := s.scheduleRepo.ListByDateRange(ctx, scope, dayStart, dayEnd, studentID)
	if err != nil {
		return DayViewResponse{}, fmt.Errorf("plan: list schedule items: %w", err)
	}

	// Enrich with student names. [17-planning §7]
	responses := make([]ScheduleItemResponse, len(items))
	for i, item := range items {
		resp := toScheduleItemResponse(item)
		if item.StudentID != nil {
			name, nameErr := s.iamSvc.GetStudentName(ctx, *item.StudentID)
			if nameErr == nil {
				resp.StudentName = &name
			}
		}
		responses[i] = resp
	}

	// Fetch activities.
	activities, err := s.learnSvc.ListActivitiesForCalendar(ctx, auth, scope, dayStart, dayEnd, studentID)
	if err != nil {
		return DayViewResponse{}, fmt.Errorf("plan: list activities: %w", err)
	}

	// Fetch attendance.
	attendanceRecords, err := s.complySvc.GetAttendanceRange(ctx, auth, scope, dayStart, dayEnd, studentID)
	if err != nil {
		return DayViewResponse{}, fmt.Errorf("plan: get attendance: %w", err)
	}
	var attendanceSummary *AttendanceSummary
	if len(attendanceRecords) > 0 {
		attendanceSummary = &attendanceRecords[0]
	}

	// Fetch events.
	events, err := s.socialSvc.GetEventsForCalendar(ctx, auth, scope, dayStart, dayEnd)
	if err != nil {
		return DayViewResponse{}, fmt.Errorf("plan: get events: %w", err)
	}

	return DayViewResponse{
		Date:          dayStart,
		ScheduleItems: responses,
		Activities:    activities,
		Attendance:    attendanceSummary,
		Events:        events,
	}, nil
}

// ─── Print/Export ────────────────────────────────────────────────────────────

func (s *PlanningServiceImpl) GetPrintView(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	start time.Time,
	end time.Time,
	studentID *uuid.UUID,
) (string, error) {
	if err := validateDateRange(start, end); err != nil {
		return "", err
	}

	items, err := s.scheduleRepo.ListByDateRange(ctx, scope, start, end, studentID)
	if err != nil {
		return "", fmt.Errorf("plan: list schedule items: %w", err)
	}

	return renderPrintHTML(start, end, items), nil
}

// ─── Schedule Templates [17-planning §11.3] ─────────────────────────────────

func (s *PlanningServiceImpl) CreateTemplate(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	input CreateTemplateInput,
) (uuid.UUID, error) {
	itemsJSON, err := json.Marshal(input.Items)
	if err != nil {
		return uuid.Nil, fmt.Errorf("plan: marshal template items: %w", err)
	}

	tmpl := &ScheduleTemplate{
		ID:          uuid.Must(uuid.NewV7()),
		FamilyID:    scope.FamilyID(),
		Name:        input.Name,
		Description: input.Description,
		Items:       itemsJSON,
		IsActive:    input.IsActive,
	}

	if err := s.templateRepo.Create(ctx, scope, tmpl); err != nil {
		return uuid.Nil, fmt.Errorf("plan: create template: %w", err)
	}
	return tmpl.ID, nil
}

func (s *PlanningServiceImpl) ListTemplates(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
) ([]TemplateResponse, error) {
	templates, err := s.templateRepo.ListByFamily(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("plan: list templates: %w", err)
	}

	responses := make([]TemplateResponse, len(templates))
	for i, tmpl := range templates {
		responses[i] = toTemplateResponse(tmpl)
	}
	return responses, nil
}

func (s *PlanningServiceImpl) UpdateTemplate(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	templateID uuid.UUID,
	input UpdateTemplateInput,
) error {
	existing, err := s.templateRepo.FindByID(ctx, scope, templateID)
	if err != nil {
		return fmt.Errorf("plan: find template: %w", err)
	}
	if existing == nil {
		return ErrTemplateNotFound
	}

	var itemsJSON []byte
	if input.Items != nil {
		itemsJSON, err = json.Marshal(*input.Items)
		if err != nil {
			return fmt.Errorf("plan: marshal template items: %w", err)
		}
	}

	if err := s.templateRepo.Update(ctx, scope, templateID, &input, itemsJSON); err != nil {
		return fmt.Errorf("plan: update template: %w", err)
	}
	return nil
}

func (s *PlanningServiceImpl) DeleteTemplate(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	templateID uuid.UUID,
) error {
	existing, err := s.templateRepo.FindByID(ctx, scope, templateID)
	if err != nil {
		return fmt.Errorf("plan: find template: %w", err)
	}
	if existing == nil {
		return ErrTemplateNotFound
	}

	if err := s.templateRepo.Delete(ctx, scope, templateID); err != nil {
		return fmt.Errorf("plan: delete template: %w", err)
	}
	return nil
}

func (s *PlanningServiceImpl) ApplyTemplate(
	ctx context.Context,
	_ *shared.AuthContext,
	scope *shared.FamilyScope,
	templateID uuid.UUID,
	input ApplyTemplateInput,
) ([]uuid.UUID, error) {
	if err := validateDateRange(input.StartDate, input.EndDate); err != nil {
		return nil, err
	}

	tmpl, err := s.templateRepo.FindByID(ctx, scope, templateID)
	if err != nil {
		return nil, fmt.Errorf("plan: find template: %w", err)
	}
	if tmpl == nil {
		return nil, ErrTemplateNotFound
	}

	var items []TemplateItem
	if err := json.Unmarshal(tmpl.Items, &items); err != nil {
		return nil, fmt.Errorf("plan: unmarshal template items: %w", err)
	}

	// Map day_of_week strings to time.Weekday for matching.
	dayMap := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
	}

	var createdIDs []uuid.UUID
	startDate := truncateToDate(input.StartDate)
	endDate := truncateToDate(input.EndDate)

	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		weekday := d.Weekday()
		for _, tmplItem := range items {
			targetDay, ok := dayMap[strings.ToLower(tmplItem.DayOfWeek)]
			if !ok || targetDay != weekday {
				continue
			}
			item := &ScheduleItem{
				ID:        uuid.Must(uuid.NewV7()),
				FamilyID:  scope.FamilyID(),
				Title:     tmplItem.Title,
				StartDate: d,
				StartTime: tmplItem.StartTime,
				EndTime:   tmplItem.EndTime,
				Category:  tmplItem.Category,
				SubjectID: tmplItem.SubjectID,
				Color:     tmplItem.Color,
			}
			if err := s.scheduleRepo.Create(ctx, scope, item); err != nil {
				return nil, fmt.Errorf("plan: create schedule item from template: %w", err)
			}
			createdIDs = append(createdIDs, item.ID)
		}
	}

	return createdIDs, nil
}

// ─── Event Handlers [17-planning §16] ────────────────────────────────────────

func (s *PlanningServiceImpl) HandleEventCancelled(
	ctx context.Context,
	eventID uuid.UUID,
	_ []uuid.UUID,
) error {
	items, err := s.scheduleRepo.FindByLinkedEventID(ctx, eventID)
	if err != nil {
		return fmt.Errorf("plan: find items by linked event: %w", err)
	}
	// No-op if no linked items. [17-planning §16]
	for _, item := range items {
		scope := shared.NewFamilyScopeFromID(item.FamilyID)
		if delErr := s.scheduleRepo.Delete(ctx, &scope, item.ID); delErr != nil {
			return fmt.Errorf("plan: delete cancelled event item: %w", delErr)
		}
	}
	return nil
}

func (s *PlanningServiceImpl) HandleActivityLogged(
	ctx context.Context,
	familyID, studentID, activityID uuid.UUID,
) error {
	scope := shared.NewFamilyScopeFromID(familyID)
	items, err := s.scheduleRepo.FindByStudentAndDate(ctx, &scope, studentID, truncateToDate(time.Now()))
	if err != nil {
		return fmt.Errorf("plan: find items for student: %w", err)
	}
	// Mark the first non-completed item as completed and link the activity. [17-planning §16]
	for _, item := range items {
		if !item.IsCompleted {
			if markErr := s.scheduleRepo.MarkCompleted(ctx, &scope, item.ID, time.Now()); markErr != nil {
				return fmt.Errorf("plan: mark item completed: %w", markErr)
			}
			if linkErr := s.scheduleRepo.SetLinkedActivity(ctx, &scope, item.ID, activityID); linkErr != nil {
				return fmt.Errorf("plan: link activity: %w", linkErr)
			}
			return nil
		}
	}
	// No-op if no matching items.
	return nil
}

// ─── Data Lifecycle ─────────────────────────────────────────────────────────

func (s *PlanningServiceImpl) ExportData(
	ctx context.Context,
	scope *shared.FamilyScope,
) ([]byte, error) {
	items, err := s.scheduleRepo.ListAllByFamily(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("plan: export data: %w", err)
	}

	templates, err := s.templateRepo.ListByFamily(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("plan: export templates: %w", err)
	}

	export := struct {
		ScheduleItems []ScheduleItem     `json:"schedule_items"`
		Templates     []ScheduleTemplate `json:"templates"`
	}{
		ScheduleItems: items,
		Templates:     templates,
	}

	data, err := json.Marshal(export)
	if err != nil {
		return nil, fmt.Errorf("plan: marshal export: %w", err)
	}
	return data, nil
}

func (s *PlanningServiceImpl) DeleteData(
	ctx context.Context,
	scope *shared.FamilyScope,
) error {
	if err := s.templateRepo.DeleteAllByFamily(ctx, scope); err != nil {
		return fmt.Errorf("plan: delete templates: %w", err)
	}
	if err := s.scheduleRepo.DeleteAllByFamily(ctx, scope); err != nil {
		return fmt.Errorf("plan: delete data: %w", err)
	}
	return nil
}

// DeleteStudentData removes schedule items for a specific student. [17-planning §14.3]
func (s *PlanningServiceImpl) DeleteStudentData(
	ctx context.Context,
	scope *shared.FamilyScope,
	studentID uuid.UUID,
) error {
	if err := s.scheduleRepo.DeleteByStudent(ctx, scope, studentID); err != nil {
		return fmt.Errorf("plan: delete student data: %w", err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Pure Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// validateDateRange checks start < end and range ≤ 90 days.
func validateDateRange(start, end time.Time) error {
	if !start.Before(end) {
		return ErrInvalidDateRange
	}
	if end.Sub(start) > time.Duration(maxCalendarRangeDays)*24*time.Hour {
		return ErrDateRangeTooLarge
	}
	return nil
}

// sourceFilter returns a function that checks if a source is wanted.
// If sources is empty, all sources are wanted.
func sourceFilter(sources []CalendarSource) func(CalendarSource) bool {
	if len(sources) == 0 {
		return func(_ CalendarSource) bool { return true }
	}
	return func(src CalendarSource) bool {
		return slices.Contains(sources, src)
	}
}

// truncateToDate returns the date at midnight UTC.
func truncateToDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// toScheduleItemResponse converts a ScheduleItem to its response representation.
func toScheduleItemResponse(item ScheduleItem) ScheduleItemResponse {
	return ScheduleItemResponse{
		ID:               item.ID,
		Title:            item.Title,
		Description:      item.Description,
		StudentID:        item.StudentID,
		StartDate:        item.StartDate,
		StartTime:        item.StartTime,
		EndTime:          item.EndTime,
		DurationMinutes:  item.DurationMinutes,
		Category:         item.Category,
		SubjectID:        item.SubjectID,
		Color:            item.Color,
		IsCompleted:      item.IsCompleted,
		CompletedAt:      item.CompletedAt,
		LinkedActivityID: item.LinkedActivityID,
		Notes:            item.Notes,
		CreatedAt:        item.CreatedAt,
	}
}

// toTemplateResponse converts a ScheduleTemplate to its response representation.
func toTemplateResponse(tmpl ScheduleTemplate) TemplateResponse {
	var items []TemplateItem
	_ = json.Unmarshal(tmpl.Items, &items)
	if items == nil {
		items = []TemplateItem{}
	}
	return TemplateResponse{
		ID:          tmpl.ID,
		Name:        tmpl.Name,
		Description: tmpl.Description,
		Items:       items,
		IsActive:    tmpl.IsActive,
		CreatedAt:   tmpl.CreatedAt,
		UpdatedAt:   tmpl.UpdatedAt,
	}
}

// resolveStudentNames collects unique student IDs from schedule items and activities,
// calls iamSvc.GetStudentName for each, and returns a lookup map. Errors are silently
// ignored — a missing name is non-fatal for calendar display. [17-planning §9.2]
func (s *PlanningServiceImpl) resolveStudentNames(
	ctx context.Context,
	scheduleItems []ScheduleItem,
	activities []ActivitySummary,
) map[uuid.UUID]string {
	seen := make(map[uuid.UUID]struct{})
	for _, item := range scheduleItems {
		if item.StudentID != nil {
			seen[*item.StudentID] = struct{}{}
		}
	}
	for _, a := range activities {
		if a.StudentID != nil {
			seen[*a.StudentID] = struct{}{}
		}
	}
	names := make(map[uuid.UUID]string, len(seen))
	for id := range seen {
		if name, err := s.iamSvc.GetStudentName(ctx, id); err == nil {
			names[id] = name
		}
	}
	return names
}

// mergeIntoCalendarDays merges all calendar sources into CalendarDay slices.
// Creates empty days for dates with no items. [17-planning §9.2]
func mergeIntoCalendarDays(
	start, end time.Time,
	scheduleItems []ScheduleItem,
	activities []ActivitySummary,
	attendance []AttendanceSummary,
	events []EventSummary,
	studentNames map[uuid.UUID]string,
) []CalendarDay {
	// Build a map of date → items.
	dayMap := make(map[string]*CalendarDay)

	// Pre-populate all dates in range (ensures empty days exist).
	startDate := truncateToDate(start)
	endDate := truncateToDate(end)
	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		dayMap[key] = &CalendarDay{Date: d, Items: []CalendarItem{}}
	}

	// Add schedule items.
	for _, item := range scheduleItems {
		key := truncateToDate(item.StartDate).Format("2006-01-02")
		day, ok := dayMap[key]
		if !ok {
			d := truncateToDate(item.StartDate)
			day = &CalendarDay{Date: d, Items: []CalendarItem{}}
			dayMap[key] = day
		}
		cat := string(item.Category)
		completed := item.IsCompleted
		var studentName *string
		if item.StudentID != nil {
			if name, ok2 := studentNames[*item.StudentID]; ok2 {
				studentName = &name
			}
		}
		day.Items = append(day.Items, CalendarItem{
			ID:              item.ID,
			Source:          CalendarSourceSchedule,
			Title:           item.Title,
			StartTime:       item.StartTime,
			EndTime:         item.EndTime,
			DurationMinutes: item.DurationMinutes,
			Category:        &cat,
			Color:           item.Color,
			StudentID:       item.StudentID,
			StudentName:     studentName,
			IsCompleted:     &completed,
			Date:            truncateToDate(item.StartDate),
			Details: CalendarItemDetails{
				Type:             "schedule",
				Description:      item.Description,
				Notes:            item.Notes,
				LinkedActivityID: item.LinkedActivityID,
			},
		})
	}

	// Add activities.
	for _, a := range activities {
		key := truncateToDate(a.Date).Format("2006-01-02")
		day, ok := dayMap[key]
		if !ok {
			d := truncateToDate(a.Date)
			day = &CalendarDay{Date: d, Items: []CalendarItem{}}
			dayMap[key] = day
		}
		var studentName *string
		if a.StudentID != nil {
			if name, ok2 := studentNames[*a.StudentID]; ok2 {
				studentName = &name
			}
		}
		day.Items = append(day.Items, CalendarItem{
			ID:          a.ID,
			Source:      CalendarSourceActivities,
			Title:       a.Title,
			StudentID:   a.StudentID,
			StudentName: studentName,
			Category:    a.Subject,
			Date:        truncateToDate(a.Date),
			Details: CalendarItemDetails{
				Type:    "activity",
				Subject: a.Subject,
				Tags:    a.Tags,
			},
		})
	}

	// Add attendance.
	for _, att := range attendance {
		key := truncateToDate(att.Date).Format("2006-01-02")
		day, ok := dayMap[key]
		if !ok {
			d := truncateToDate(att.Date)
			day = &CalendarDay{Date: d, Items: []CalendarItem{}}
			dayMap[key] = day
		}
		title := "Attendance: " + att.Status
		day.Items = append(day.Items, CalendarItem{
			ID:        att.ID,
			Source:    CalendarSourceAttendance,
			Title:     title,
			StudentID: att.StudentID,
			Date:      truncateToDate(att.Date),
			Details: CalendarItemDetails{
				Type:   "attendance",
				Status: &att.Status,
			},
		})
	}

	// Add events.
	for _, e := range events {
		key := truncateToDate(e.Date).Format("2006-01-02")
		day, ok := dayMap[key]
		if !ok {
			d := truncateToDate(e.Date)
			day = &CalendarDay{Date: d, Items: []CalendarItem{}}
			dayMap[key] = day
		}
		day.Items = append(day.Items, CalendarItem{
			ID:        e.ID,
			Source:    CalendarSourceEvents,
			Title:     e.Title,
			StartTime: e.StartTime,
			EndTime:   e.EndTime,
			Date:      truncateToDate(e.Date),
			Details: CalendarItemDetails{
				Type:       "event",
				GroupName:  e.GroupName,
				Location:   e.Location,
				RSVPStatus: e.RSVPStatus,
			},
		})
	}

	// Flatten map to sorted slice.
	days := make([]CalendarDay, 0, len(dayMap))
	for d := startDate; d.Before(endDate); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		if day, ok := dayMap[key]; ok {
			days = append(days, *day)
		}
	}
	return days
}

// renderPrintHTML generates a simple print-friendly HTML page. [17-planning §13.1]
func renderPrintHTML(start, end time.Time, items []ScheduleItem) string {
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>Schedule</title>
<style>
body { font-family: serif; max-width: 8.5in; margin: 0 auto; padding: 0.5in; }
h1 { font-size: 1.2em; margin-bottom: 0.5em; }
.day { margin-bottom: 1em; }
.day-header { font-weight: bold; border-bottom: 1px solid #333; margin-bottom: 0.25em; }
.item { margin-left: 1em; margin-bottom: 0.25em; }
.category { font-style: italic; color: #666; }
@media print { body { padding: 0; } }
</style>
</head>
<body>
`)
	fmt.Fprintf(&b, "<h1>Schedule: %s – %s</h1>\n",
		start.Format("Jan 2, 2006"), end.Add(-24*time.Hour).Format("Jan 2, 2006"))

	// Group items by date.
	grouped := make(map[string][]ScheduleItem)
	for _, item := range items {
		key := truncateToDate(item.StartDate).Format("2006-01-02")
		grouped[key] = append(grouped[key], item)
	}

	for d := truncateToDate(start); d.Before(truncateToDate(end)); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		b.WriteString(`<div class="day">`)
		fmt.Fprintf(&b, `<div class="day-header">%s</div>`, d.Format("Monday, January 2"))
		dayItems := grouped[key]
		if len(dayItems) == 0 {
			b.WriteString(`<div class="item">No scheduled items</div>`)
		}
		for _, item := range dayItems {
			timeStr := ""
			if item.StartTime != nil {
				timeStr = *item.StartTime
				if item.EndTime != nil {
					timeStr += " – " + *item.EndTime
				}
				timeStr += " "
			}
			fmt.Fprintf(&b, `<div class="item">%s%s <span class="category">[%s]</span></div>`,
				timeStr, item.Title, item.Category)
		}
		b.WriteString("</div>\n")
	}

	b.WriteString("</body>\n</html>")
	return b.String()
}
