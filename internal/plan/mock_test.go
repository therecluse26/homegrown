package plan

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Repository: ScheduleItemRepository [17-planning mock pattern]
// ═══════════════════════════════════════════════════════════════════════════════

type stubScheduleItemRepo struct {
	createFn              func(ctx context.Context, scope *shared.FamilyScope, item *ScheduleItem) error
	findByIDFn            func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ScheduleItem, error)
	findByLinkedEventIDFn func(ctx context.Context, eventID uuid.UUID) ([]ScheduleItem, error)
	findByStudentAndDateFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, date time.Time) ([]ScheduleItem, error)
	listByDateRangeFn     func(ctx context.Context, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]ScheduleItem, error)
	listFilteredFn        func(ctx context.Context, scope *shared.FamilyScope, query *ScheduleItemQuery, pagination *shared.PaginationParams) ([]ScheduleItem, error)
	updateFn              func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, input *UpdateScheduleItemInput) error
	markCompletedFn       func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, completedAt time.Time) error
	setLinkedActivityFn   func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, activityID uuid.UUID) error
	deleteFn              func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error
	deleteAllByFamilyFn   func(ctx context.Context, scope *shared.FamilyScope) error
	listAllByFamilyFn     func(ctx context.Context, scope *shared.FamilyScope) ([]ScheduleItem, error)
}

func (s *stubScheduleItemRepo) Create(ctx context.Context, scope *shared.FamilyScope, item *ScheduleItem) error {
	if s.createFn != nil {
		return s.createFn(ctx, scope, item)
	}
	panic("stubScheduleItemRepo.Create not stubbed")
}

func (s *stubScheduleItemRepo) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ScheduleItem, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, scope, id)
	}
	panic("stubScheduleItemRepo.FindByID not stubbed")
}

func (s *stubScheduleItemRepo) FindByLinkedEventID(ctx context.Context, eventID uuid.UUID) ([]ScheduleItem, error) {
	if s.findByLinkedEventIDFn != nil {
		return s.findByLinkedEventIDFn(ctx, eventID)
	}
	return []ScheduleItem{}, nil
}

func (s *stubScheduleItemRepo) FindByStudentAndDate(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, date time.Time) ([]ScheduleItem, error) {
	if s.findByStudentAndDateFn != nil {
		return s.findByStudentAndDateFn(ctx, scope, studentID, date)
	}
	return []ScheduleItem{}, nil
}

func (s *stubScheduleItemRepo) ListByDateRange(ctx context.Context, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]ScheduleItem, error) {
	if s.listByDateRangeFn != nil {
		return s.listByDateRangeFn(ctx, scope, start, end, studentID)
	}
	panic("stubScheduleItemRepo.ListByDateRange not stubbed")
}

func (s *stubScheduleItemRepo) ListFiltered(ctx context.Context, scope *shared.FamilyScope, query *ScheduleItemQuery, pagination *shared.PaginationParams) ([]ScheduleItem, error) {
	if s.listFilteredFn != nil {
		return s.listFilteredFn(ctx, scope, query, pagination)
	}
	panic("stubScheduleItemRepo.ListFiltered not stubbed")
}

func (s *stubScheduleItemRepo) Update(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, input *UpdateScheduleItemInput) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, scope, id, input)
	}
	panic("stubScheduleItemRepo.Update not stubbed")
}

func (s *stubScheduleItemRepo) MarkCompleted(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, completedAt time.Time) error {
	if s.markCompletedFn != nil {
		return s.markCompletedFn(ctx, scope, id, completedAt)
	}
	panic("stubScheduleItemRepo.MarkCompleted not stubbed")
}

func (s *stubScheduleItemRepo) SetLinkedActivity(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, activityID uuid.UUID) error {
	if s.setLinkedActivityFn != nil {
		return s.setLinkedActivityFn(ctx, scope, id, activityID)
	}
	panic("stubScheduleItemRepo.SetLinkedActivity not stubbed")
}

func (s *stubScheduleItemRepo) Delete(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, scope, id)
	}
	panic("stubScheduleItemRepo.Delete not stubbed")
}

func (s *stubScheduleItemRepo) DeleteAllByFamily(ctx context.Context, scope *shared.FamilyScope) error {
	if s.deleteAllByFamilyFn != nil {
		return s.deleteAllByFamilyFn(ctx, scope)
	}
	return nil
}

func (s *stubScheduleItemRepo) DeleteByStudent(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
	return nil
}

func (s *stubScheduleItemRepo) ListAllByFamily(ctx context.Context, scope *shared.FamilyScope) ([]ScheduleItem, error) {
	if s.listAllByFamilyFn != nil {
		return s.listAllByFamilyFn(ctx, scope)
	}
	return []ScheduleItem{}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Repository: ScheduleTemplateRepository [17-planning mock pattern]
// ═══════════════════════════════════════════════════════════════════════════════

type stubScheduleTemplateRepo struct {
	createFn            func(ctx context.Context, scope *shared.FamilyScope, tmpl *ScheduleTemplate) error
	findByIDFn          func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ScheduleTemplate, error)
	listByFamilyFn      func(ctx context.Context, scope *shared.FamilyScope) ([]ScheduleTemplate, error)
	updateFn            func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, input *UpdateTemplateInput, itemsJSON []byte) error
	deleteFn            func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error
	deleteAllByFamilyFn func(ctx context.Context, scope *shared.FamilyScope) error
}

func (s *stubScheduleTemplateRepo) Create(ctx context.Context, scope *shared.FamilyScope, tmpl *ScheduleTemplate) error {
	if s.createFn != nil {
		return s.createFn(ctx, scope, tmpl)
	}
	return nil
}

func (s *stubScheduleTemplateRepo) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ScheduleTemplate, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, scope, id)
	}
	return nil, nil
}

func (s *stubScheduleTemplateRepo) ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]ScheduleTemplate, error) {
	if s.listByFamilyFn != nil {
		return s.listByFamilyFn(ctx, scope)
	}
	return []ScheduleTemplate{}, nil
}

func (s *stubScheduleTemplateRepo) Update(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, input *UpdateTemplateInput, itemsJSON []byte) error {
	if s.updateFn != nil {
		return s.updateFn(ctx, scope, id, input, itemsJSON)
	}
	return nil
}

func (s *stubScheduleTemplateRepo) Delete(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, scope, id)
	}
	return nil
}

func (s *stubScheduleTemplateRepo) DeleteAllByFamily(ctx context.Context, scope *shared.FamilyScope) error {
	if s.deleteAllByFamilyFn != nil {
		return s.deleteAllByFamilyFn(ctx, scope)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Cross-Domain Services
// ═══════════════════════════════════════════════════════════════════════════════

// ─── stubIamService ─────────────────────────────────────────────────────────

type stubIamService struct {
	studentBelongsToFamilyFn func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error)
	getStudentNameFn         func(ctx context.Context, studentID uuid.UUID) (string, error)
}

func (s *stubIamService) StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error) {
	if s.studentBelongsToFamilyFn != nil {
		return s.studentBelongsToFamilyFn(ctx, studentID, familyID)
	}
	return true, nil // safe default
}

func (s *stubIamService) GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error) {
	if s.getStudentNameFn != nil {
		return s.getStudentNameFn(ctx, studentID)
	}
	return "Test Student", nil
}

// ─── stubLearningService ────────────────────────────────────────────────────

type stubLearningService struct {
	listActivitiesForCalendarFn func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]ActivitySummary, error)
	logActivityFn               func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, title string, date time.Time, durationMinutes *int, studentID *uuid.UUID, description *string, tags []string) (uuid.UUID, error)
}

func (s *stubLearningService) ListActivitiesForCalendar(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]ActivitySummary, error) {
	if s.listActivitiesForCalendarFn != nil {
		return s.listActivitiesForCalendarFn(ctx, auth, scope, start, end, studentID)
	}
	return []ActivitySummary{}, nil
}

func (s *stubLearningService) LogActivity(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, title string, date time.Time, durationMinutes *int, studentID *uuid.UUID, description *string, tags []string) (uuid.UUID, error) {
	if s.logActivityFn != nil {
		return s.logActivityFn(ctx, auth, scope, title, date, durationMinutes, studentID, description, tags)
	}
	return uuid.Must(uuid.NewV7()), nil
}

// ─── stubComplianceService ──────────────────────────────────────────────────

type stubComplianceService struct {
	getAttendanceRangeFn func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]AttendanceSummary, error)
}

func (s *stubComplianceService) GetAttendanceRange(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]AttendanceSummary, error) {
	if s.getAttendanceRangeFn != nil {
		return s.getAttendanceRangeFn(ctx, auth, scope, start, end, studentID)
	}
	return []AttendanceSummary{}, nil
}

// ─── stubSocialService ──────────────────────────────────────────────────────

type stubSocialService struct {
	getEventsForCalendarFn func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time) ([]EventSummary, error)
}

func (s *stubSocialService) GetEventsForCalendar(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time) ([]EventSummary, error) {
	if s.getEventsForCalendarFn != nil {
		return s.getEventsForCalendarFn(ctx, auth, scope, start, end)
	}
	return []EventSummary{}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func testAuth() *shared.AuthContext {
	return &shared.AuthContext{
		ParentID:        uuid.Must(uuid.NewV7()),
		FamilyID:        uuid.Must(uuid.NewV7()),
		IdentityID:      uuid.Must(uuid.NewV7()),
		IsPrimaryParent: true,
	}
}

func testScope() *shared.FamilyScope {
	auth := testAuth()
	s := shared.NewFamilyScopeFromAuth(auth)
	return &s
}

func testScopeFromAuth(auth *shared.AuthContext) *shared.FamilyScope {
	s := shared.NewFamilyScopeFromAuth(auth)
	return &s
}

type testDeps struct {
	repo         ScheduleItemRepository
	templateRepo ScheduleTemplateRepository
	iamSvc       IamServiceForPlan
	learnSvc     LearningServiceForPlan
	complySvc    ComplianceServiceForPlan
	socialSvc    SocialServiceForPlan
}

// newTestService creates a PlanningService with the legacy 5-arg signature for existing tests.
func newTestService(
	repo ScheduleItemRepository,
	iamSvc IamServiceForPlan,
	learnSvc LearningServiceForPlan,
	complySvc ComplianceServiceForPlan,
	socialSvc SocialServiceForPlan,
) PlanningService {
	return NewPlanningService(repo, &stubScheduleTemplateRepo{}, iamSvc, learnSvc, complySvc, socialSvc)
}

// newTestServiceFull creates a PlanningService with all dependencies.
func newTestServiceFull(d testDeps) PlanningService {
	return NewPlanningService(d.repo, d.templateRepo, d.iamSvc, d.learnSvc, d.complySvc, d.socialSvc)
}
