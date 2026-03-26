package comply

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Repositories [14-comply Phase 0 mock pattern]
// ═══════════════════════════════════════════════════════════════════════════════

// ─── stubStateConfigRepo ──────────────────────────────────────────────────────

type stubStateConfigRepo struct {
	listAllFn         func(ctx context.Context) ([]ComplyStateConfig, error)
	findByStateCodeFn func(ctx context.Context, stateCode string) (*ComplyStateConfig, error)
	upsertFn          func(ctx context.Context, config UpsertStateConfigRow) (*ComplyStateConfig, error)
}

func (s *stubStateConfigRepo) ListAll(ctx context.Context) ([]ComplyStateConfig, error) {
	if s.listAllFn != nil {
		return s.listAllFn(ctx)
	}
	panic("stubStateConfigRepo.ListAll not stubbed")
}

func (s *stubStateConfigRepo) FindByStateCode(ctx context.Context, stateCode string) (*ComplyStateConfig, error) {
	if s.findByStateCodeFn != nil {
		return s.findByStateCodeFn(ctx, stateCode)
	}
	panic("stubStateConfigRepo.FindByStateCode not stubbed")
}

func (s *stubStateConfigRepo) Upsert(ctx context.Context, config UpsertStateConfigRow) (*ComplyStateConfig, error) {
	if s.upsertFn != nil {
		return s.upsertFn(ctx, config)
	}
	panic("stubStateConfigRepo.Upsert not stubbed")
}

// ─── stubFamilyConfigRepo ─────────────────────────────────────────────────────

type stubFamilyConfigRepo struct {
	upsertFn         func(ctx context.Context, scope shared.FamilyScope, input UpsertFamilyConfigRow) (*ComplyFamilyConfig, error)
	findByFamilyFn   func(ctx context.Context, scope shared.FamilyScope) (*ComplyFamilyConfig, error)
	deleteByFamilyFn func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubFamilyConfigRepo) Upsert(ctx context.Context, scope shared.FamilyScope, input UpsertFamilyConfigRow) (*ComplyFamilyConfig, error) {
	if s.upsertFn != nil {
		return s.upsertFn(ctx, scope, input)
	}
	panic("stubFamilyConfigRepo.Upsert not stubbed")
}

func (s *stubFamilyConfigRepo) FindByFamily(ctx context.Context, scope shared.FamilyScope) (*ComplyFamilyConfig, error) {
	if s.findByFamilyFn != nil {
		return s.findByFamilyFn(ctx, scope)
	}
	panic("stubFamilyConfigRepo.FindByFamily not stubbed")
}

func (s *stubFamilyConfigRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	panic("stubFamilyConfigRepo.DeleteByFamily not stubbed")
}

// ─── stubScheduleRepo ─────────────────────────────────────────────────────────

type stubScheduleRepo struct {
	createFn         func(ctx context.Context, scope shared.FamilyScope, input CreateScheduleRow) (*ComplyCustomSchedule, error)
	findByIDFn       func(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) (*ComplyCustomSchedule, error)
	listByFamilyFn   func(ctx context.Context, scope shared.FamilyScope) ([]ComplyCustomSchedule, error)
	updateFn         func(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope, updates UpdateScheduleRow) (*ComplyCustomSchedule, error)
	deleteFn         func(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) error
	deleteByFamilyFn func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubScheduleRepo) Create(ctx context.Context, scope shared.FamilyScope, input CreateScheduleRow) (*ComplyCustomSchedule, error) {
	if s.createFn != nil {
		return s.createFn(ctx, scope, input)
	}
	panic("stubScheduleRepo.Create not stubbed")
}

func (s *stubScheduleRepo) FindByID(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) (*ComplyCustomSchedule, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, scheduleID, scope)
	}
	panic("stubScheduleRepo.FindByID not stubbed")
}

func (s *stubScheduleRepo) ListByFamily(ctx context.Context, scope shared.FamilyScope) ([]ComplyCustomSchedule, error) {
	if s.listByFamilyFn != nil {
		return s.listByFamilyFn(ctx, scope)
	}
	panic("stubScheduleRepo.ListByFamily not stubbed")
}

func (s *stubScheduleRepo) Update(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope, updates UpdateScheduleRow) (*ComplyCustomSchedule, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, scheduleID, scope, updates)
	}
	panic("stubScheduleRepo.Update not stubbed")
}

func (s *stubScheduleRepo) Delete(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, scheduleID, scope)
	}
	panic("stubScheduleRepo.Delete not stubbed")
}

func (s *stubScheduleRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	return nil // default: no-op for tests that don't care
}

// ─── stubAttendanceRepo ───────────────────────────────────────────────────────

type stubAttendanceRepo struct {
	upsertFn               func(ctx context.Context, scope shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error)
	findByIDFn             func(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) (*ComplyAttendance, error)
	findByStudentAndDateFn func(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, date time.Time) (*ComplyAttendance, error)
	listByStudentFn        func(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AttendanceListParams) ([]ComplyAttendance, error)
	summarizeFn            func(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, startDate time.Time, endDate time.Time) (*AttendanceSummaryRow, error)
	updateFn               func(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope, updates UpdateAttendanceRow) (*ComplyAttendance, error)
	deleteFn               func(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) error
	deleteByStudentFn      func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
	deleteByFamilyFn       func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubAttendanceRepo) Upsert(ctx context.Context, scope shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error) {
	if s.upsertFn != nil {
		return s.upsertFn(ctx, scope, input)
	}
	panic("stubAttendanceRepo.Upsert not stubbed")
}

func (s *stubAttendanceRepo) FindByStudentAndDate(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, date time.Time) (*ComplyAttendance, error) {
	if s.findByStudentAndDateFn != nil {
		return s.findByStudentAndDateFn(ctx, studentID, scope, date)
	}
	return nil, nil // default: no existing record
}

func (s *stubAttendanceRepo) FindByID(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) (*ComplyAttendance, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, attendanceID, scope)
	}
	panic("stubAttendanceRepo.FindByID not stubbed")
}

func (s *stubAttendanceRepo) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AttendanceListParams) ([]ComplyAttendance, error) {
	if s.listByStudentFn != nil {
		return s.listByStudentFn(ctx, studentID, scope, params)
	}
	panic("stubAttendanceRepo.ListByStudent not stubbed")
}

func (s *stubAttendanceRepo) Summarize(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, startDate time.Time, endDate time.Time) (*AttendanceSummaryRow, error) {
	if s.summarizeFn != nil {
		return s.summarizeFn(ctx, studentID, scope, startDate, endDate)
	}
	panic("stubAttendanceRepo.Summarize not stubbed")
}

func (s *stubAttendanceRepo) Update(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope, updates UpdateAttendanceRow) (*ComplyAttendance, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, attendanceID, scope, updates)
	}
	panic("stubAttendanceRepo.Update not stubbed")
}

func (s *stubAttendanceRepo) Delete(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, attendanceID, scope)
	}
	panic("stubAttendanceRepo.Delete not stubbed")
}

func (s *stubAttendanceRepo) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	if s.deleteByStudentFn != nil {
		return s.deleteByStudentFn(ctx, studentID, familyID)
	}
	panic("stubAttendanceRepo.DeleteByStudent not stubbed")
}

func (s *stubAttendanceRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	panic("stubAttendanceRepo.DeleteByFamily not stubbed")
}

// ─── stubAssessmentRepo ───────────────────────────────────────────────────────

type stubAssessmentRepo struct {
	createFn          func(ctx context.Context, scope shared.FamilyScope, input CreateAssessmentRow) (*ComplyAssessmentRecord, error)
	findByIDFn        func(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) (*ComplyAssessmentRecord, error)
	listByStudentFn   func(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AssessmentListParams) ([]ComplyAssessmentRecord, error)
	updateFn          func(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope, updates UpdateAssessmentRow) (*ComplyAssessmentRecord, error)
	deleteFn          func(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) error
	deleteByStudentFn func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
	deleteByFamilyFn  func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubAssessmentRepo) Create(ctx context.Context, scope shared.FamilyScope, input CreateAssessmentRow) (*ComplyAssessmentRecord, error) {
	if s.createFn != nil {
		return s.createFn(ctx, scope, input)
	}
	panic("stubAssessmentRepo.Create not stubbed")
}

func (s *stubAssessmentRepo) FindByID(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) (*ComplyAssessmentRecord, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, assessmentID, scope)
	}
	panic("stubAssessmentRepo.FindByID not stubbed")
}

func (s *stubAssessmentRepo) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AssessmentListParams) ([]ComplyAssessmentRecord, error) {
	if s.listByStudentFn != nil {
		return s.listByStudentFn(ctx, studentID, scope, params)
	}
	panic("stubAssessmentRepo.ListByStudent not stubbed")
}

func (s *stubAssessmentRepo) Update(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope, updates UpdateAssessmentRow) (*ComplyAssessmentRecord, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, assessmentID, scope, updates)
	}
	panic("stubAssessmentRepo.Update not stubbed")
}

func (s *stubAssessmentRepo) Delete(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, assessmentID, scope)
	}
	panic("stubAssessmentRepo.Delete not stubbed")
}

func (s *stubAssessmentRepo) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	if s.deleteByStudentFn != nil {
		return s.deleteByStudentFn(ctx, studentID, familyID)
	}
	panic("stubAssessmentRepo.DeleteByStudent not stubbed")
}

func (s *stubAssessmentRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	return nil
}

// ─── stubTestScoreRepo ────────────────────────────────────────────────────────

type stubTestScoreRepo struct {
	createFn          func(ctx context.Context, scope shared.FamilyScope, input CreateTestScoreRow) (*ComplyStandardizedTest, error)
	listByStudentFn   func(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *TestListParams) ([]ComplyStandardizedTest, error)
	updateFn          func(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope, updates UpdateTestScoreRow) (*ComplyStandardizedTest, error)
	deleteFn          func(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope) error
	deleteByStudentFn func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
	deleteByFamilyFn  func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubTestScoreRepo) Create(ctx context.Context, scope shared.FamilyScope, input CreateTestScoreRow) (*ComplyStandardizedTest, error) {
	if s.createFn != nil {
		return s.createFn(ctx, scope, input)
	}
	panic("stubTestScoreRepo.Create not stubbed")
}

func (s *stubTestScoreRepo) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *TestListParams) ([]ComplyStandardizedTest, error) {
	if s.listByStudentFn != nil {
		return s.listByStudentFn(ctx, studentID, scope, params)
	}
	panic("stubTestScoreRepo.ListByStudent not stubbed")
}

func (s *stubTestScoreRepo) Update(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope, updates UpdateTestScoreRow) (*ComplyStandardizedTest, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, testID, scope, updates)
	}
	panic("stubTestScoreRepo.Update not stubbed")
}

func (s *stubTestScoreRepo) Delete(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, testID, scope)
	}
	panic("stubTestScoreRepo.Delete not stubbed")
}

func (s *stubTestScoreRepo) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	if s.deleteByStudentFn != nil {
		return s.deleteByStudentFn(ctx, studentID, familyID)
	}
	return nil
}

func (s *stubTestScoreRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	return nil
}

// ─── stubPortfolioRepo ────────────────────────────────────────────────────────

type stubPortfolioRepo struct {
	createFn          func(ctx context.Context, scope shared.FamilyScope, input CreatePortfolioRow) (*ComplyPortfolio, error)
	findByIDFn        func(ctx context.Context, portfolioID uuid.UUID, scope shared.FamilyScope) (*ComplyPortfolio, error)
	listByStudentFn   func(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyPortfolio, error)
	updateStatusFn    func(ctx context.Context, portfolioID uuid.UUID, status string, uploadID *uuid.UUID, errorMessage *string) (*ComplyPortfolio, error)
	findExpiredFn     func(ctx context.Context, before time.Time) ([]ComplyPortfolio, error)
	deleteByStudentFn func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
	deleteByFamilyFn  func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubPortfolioRepo) Create(ctx context.Context, scope shared.FamilyScope, input CreatePortfolioRow) (*ComplyPortfolio, error) {
	if s.createFn != nil {
		return s.createFn(ctx, scope, input)
	}
	panic("stubPortfolioRepo.Create not stubbed")
}

func (s *stubPortfolioRepo) FindByID(ctx context.Context, portfolioID uuid.UUID, scope shared.FamilyScope) (*ComplyPortfolio, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, portfolioID, scope)
	}
	panic("stubPortfolioRepo.FindByID not stubbed")
}

func (s *stubPortfolioRepo) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyPortfolio, error) {
	if s.listByStudentFn != nil {
		return s.listByStudentFn(ctx, studentID, scope)
	}
	panic("stubPortfolioRepo.ListByStudent not stubbed")
}

func (s *stubPortfolioRepo) UpdateStatus(ctx context.Context, portfolioID uuid.UUID, status string, uploadID *uuid.UUID, errorMessage *string) (*ComplyPortfolio, error) {
	if s.updateStatusFn != nil {
		return s.updateStatusFn(ctx, portfolioID, status, uploadID, errorMessage)
	}
	panic("stubPortfolioRepo.UpdateStatus not stubbed")
}

func (s *stubPortfolioRepo) FindExpired(ctx context.Context, before time.Time) ([]ComplyPortfolio, error) {
	if s.findExpiredFn != nil {
		return s.findExpiredFn(ctx, before)
	}
	panic("stubPortfolioRepo.FindExpired not stubbed")
}

func (s *stubPortfolioRepo) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	if s.deleteByStudentFn != nil {
		return s.deleteByStudentFn(ctx, studentID, familyID)
	}
	return nil
}

func (s *stubPortfolioRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	return nil
}

// ─── stubPortfolioItemRepo ────────────────────────────────────────────────────

type stubPortfolioItemRepo struct {
	createBatchFn       func(ctx context.Context, items []CreatePortfolioItemRow) ([]ComplyPortfolioItem, error)
	listByPortfolioFn   func(ctx context.Context, portfolioID uuid.UUID) ([]ComplyPortfolioItem, error)
	countByPortfolioFn  func(ctx context.Context, portfolioID uuid.UUID) (int32, error)
	deleteByPortfolioFn func(ctx context.Context, portfolioID uuid.UUID) error
	deleteByStudentFn   func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
	deleteByFamilyFn    func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubPortfolioItemRepo) CreateBatch(ctx context.Context, items []CreatePortfolioItemRow) ([]ComplyPortfolioItem, error) {
	if s.createBatchFn != nil {
		return s.createBatchFn(ctx, items)
	}
	panic("stubPortfolioItemRepo.CreateBatch not stubbed")
}

func (s *stubPortfolioItemRepo) ListByPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]ComplyPortfolioItem, error) {
	if s.listByPortfolioFn != nil {
		return s.listByPortfolioFn(ctx, portfolioID)
	}
	panic("stubPortfolioItemRepo.ListByPortfolio not stubbed")
}

func (s *stubPortfolioItemRepo) CountByPortfolio(ctx context.Context, portfolioID uuid.UUID) (int32, error) {
	if s.countByPortfolioFn != nil {
		return s.countByPortfolioFn(ctx, portfolioID)
	}
	return 0, nil
}

func (s *stubPortfolioItemRepo) DeleteByPortfolio(ctx context.Context, portfolioID uuid.UUID) error {
	if s.deleteByPortfolioFn != nil {
		return s.deleteByPortfolioFn(ctx, portfolioID)
	}
	panic("stubPortfolioItemRepo.DeleteByPortfolio not stubbed")
}

func (s *stubPortfolioItemRepo) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	if s.deleteByStudentFn != nil {
		return s.deleteByStudentFn(ctx, studentID, familyID)
	}
	return nil
}

func (s *stubPortfolioItemRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	return nil
}

// ─── stubTranscriptRepo ──────────────────────────────────────────────────────

type stubTranscriptRepo struct {
	createFn          func(ctx context.Context, scope shared.FamilyScope, input CreateTranscriptRow) (*ComplyTranscript, error)
	findByIDFn        func(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) (*ComplyTranscript, error)
	listByStudentFn   func(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyTranscript, error)
	updateStatusFn    func(ctx context.Context, transcriptID uuid.UUID, status string, uploadID *uuid.UUID, gpaUnweighted *float64, gpaWeighted *float64, errorMessage *string) (*ComplyTranscript, error)
	deleteFn          func(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) error
	deleteByStudentFn func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
	deleteByFamilyFn  func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubTranscriptRepo) Create(ctx context.Context, scope shared.FamilyScope, input CreateTranscriptRow) (*ComplyTranscript, error) {
	if s.createFn != nil {
		return s.createFn(ctx, scope, input)
	}
	panic("stubTranscriptRepo.Create not stubbed")
}

func (s *stubTranscriptRepo) FindByID(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) (*ComplyTranscript, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, transcriptID, scope)
	}
	panic("stubTranscriptRepo.FindByID not stubbed")
}

func (s *stubTranscriptRepo) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyTranscript, error) {
	if s.listByStudentFn != nil {
		return s.listByStudentFn(ctx, studentID, scope)
	}
	panic("stubTranscriptRepo.ListByStudent not stubbed")
}

func (s *stubTranscriptRepo) UpdateStatus(ctx context.Context, transcriptID uuid.UUID, status string, uploadID *uuid.UUID, gpaUnweighted *float64, gpaWeighted *float64, errorMessage *string) (*ComplyTranscript, error) {
	if s.updateStatusFn != nil {
		return s.updateStatusFn(ctx, transcriptID, status, uploadID, gpaUnweighted, gpaWeighted, errorMessage)
	}
	panic("stubTranscriptRepo.UpdateStatus not stubbed")
}

func (s *stubTranscriptRepo) Delete(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, transcriptID, scope)
	}
	panic("stubTranscriptRepo.Delete not stubbed")
}

func (s *stubTranscriptRepo) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	if s.deleteByStudentFn != nil {
		return s.deleteByStudentFn(ctx, studentID, familyID)
	}
	return nil
}

func (s *stubTranscriptRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	return nil
}

// ─── stubCourseRepo ──────────────────────────────────────────────────────────

type stubCourseRepo struct {
	createFn          func(ctx context.Context, scope shared.FamilyScope, input CreateCourseRow) (*ComplyCourse, error)
	listByStudentFn   func(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *CourseListParams) ([]ComplyCourse, error)
	updateFn          func(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope, updates UpdateCourseRow) (*ComplyCourse, error)
	deleteFn          func(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope) error
	deleteByStudentFn func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
	deleteByFamilyFn  func(ctx context.Context, familyID uuid.UUID) error
}

func (s *stubCourseRepo) Create(ctx context.Context, scope shared.FamilyScope, input CreateCourseRow) (*ComplyCourse, error) {
	if s.createFn != nil {
		return s.createFn(ctx, scope, input)
	}
	panic("stubCourseRepo.Create not stubbed")
}

func (s *stubCourseRepo) ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *CourseListParams) ([]ComplyCourse, error) {
	if s.listByStudentFn != nil {
		return s.listByStudentFn(ctx, studentID, scope, params)
	}
	panic("stubCourseRepo.ListByStudent not stubbed")
}

func (s *stubCourseRepo) Update(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope, updates UpdateCourseRow) (*ComplyCourse, error) {
	if s.updateFn != nil {
		return s.updateFn(ctx, courseID, scope, updates)
	}
	panic("stubCourseRepo.Update not stubbed")
}

func (s *stubCourseRepo) Delete(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, courseID, scope)
	}
	panic("stubCourseRepo.Delete not stubbed")
}

func (s *stubCourseRepo) DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error {
	if s.deleteByStudentFn != nil {
		return s.deleteByStudentFn(ctx, studentID, familyID)
	}
	return nil
}

func (s *stubCourseRepo) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cross-Domain Service Stubs
// ═══════════════════════════════════════════════════════════════════════════════

// ─── stubIamService ───────────────────────────────────────────────────────────

type stubIamService struct {
	studentBelongsToFamilyFn func(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error)
	getStudentNameFn         func(ctx context.Context, studentID uuid.UUID) (string, error)
}

func (s *stubIamService) StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error) {
	if s.studentBelongsToFamilyFn != nil {
		return s.studentBelongsToFamilyFn(ctx, studentID, familyID)
	}
	panic("stubIamService.StudentBelongsToFamily not stubbed")
}

func (s *stubIamService) GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error) {
	if s.getStudentNameFn != nil {
		return s.getStudentNameFn(ctx, studentID)
	}
	panic("stubIamService.GetStudentName not stubbed")
}

// ─── stubLearningService ──────────────────────────────────────────────────────

type stubLearningService struct {
	getPortfolioItemDataFn func(ctx context.Context, sourceType string, sourceID uuid.UUID) (*PortfolioItemData, error)
}

func (s *stubLearningService) GetPortfolioItemData(ctx context.Context, sourceType string, sourceID uuid.UUID) (*PortfolioItemData, error) {
	if s.getPortfolioItemDataFn != nil {
		return s.getPortfolioItemDataFn(ctx, sourceType, sourceID)
	}
	panic("stubLearningService.GetPortfolioItemData not stubbed")
}

// ─── stubDiscoveryService ─────────────────────────────────────────────────────

type stubDiscoveryService struct {
	getStateRequirementsFn func(ctx context.Context, stateCode string) (*StateRequirementsData, error)
	listStateGuidesFn      func(ctx context.Context) ([]StateGuideSummary, error)
}

func (s *stubDiscoveryService) GetStateRequirements(ctx context.Context, stateCode string) (*StateRequirementsData, error) {
	if s.getStateRequirementsFn != nil {
		return s.getStateRequirementsFn(ctx, stateCode)
	}
	panic("stubDiscoveryService.GetStateRequirements not stubbed")
}

func (s *stubDiscoveryService) ListStateGuides(ctx context.Context) ([]StateGuideSummary, error) {
	if s.listStateGuidesFn != nil {
		return s.listStateGuidesFn(ctx)
	}
	panic("stubDiscoveryService.ListStateGuides not stubbed")
}

// ─── stubMediaService ─────────────────────────────────────────────────────────

type stubMediaService struct {
	requestUploadFn func(ctx context.Context, familyID uuid.UUID, uploadContext string, filename string, contentType string, data []byte) (*uuid.UUID, error)
	presignedGetFn  func(ctx context.Context, uploadID uuid.UUID) (string, error)
}

func (s *stubMediaService) RequestUpload(ctx context.Context, familyID uuid.UUID, uploadContext string, filename string, contentType string, data []byte) (*uuid.UUID, error) {
	if s.requestUploadFn != nil {
		return s.requestUploadFn(ctx, familyID, uploadContext, filename, contentType, data)
	}
	panic("stubMediaService.RequestUpload not stubbed")
}

func (s *stubMediaService) PresignedGet(ctx context.Context, uploadID uuid.UUID) (string, error) {
	if s.presignedGetFn != nil {
		return s.presignedGetFn(ctx, uploadID)
	}
	panic("stubMediaService.PresignedGet not stubbed")
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func testScope() *shared.FamilyScope {
	auth := &shared.AuthContext{
		FamilyID: uuid.Must(uuid.NewV7()),
	}
	s := shared.NewFamilyScopeFromAuth(auth)
	return &s
}

func newTestService(
	stateConfigRepo StateConfigRepository,
	familyConfigRepo FamilyConfigRepository,
	scheduleRepo ScheduleRepository,
	attendanceRepo AttendanceRepository,
	assessmentRepo AssessmentRepository,
	testScoreRepo TestScoreRepository,
	portfolioRepo PortfolioRepository,
	portfolioItemRepo PortfolioItemRepository,
	transcriptRepo TranscriptRepository,
	courseRepo CourseRepository,
	iamSvc IamServiceForComply,
	learnSvc LearningServiceForComply,
	discoverySvc DiscoveryServiceForComply,
	mediaSvc MediaServiceForComply,
) ComplianceService {
	return NewComplianceService(
		stateConfigRepo,
		familyConfigRepo,
		scheduleRepo,
		attendanceRepo,
		assessmentRepo,
		testScoreRepo,
		portfolioRepo,
		portfolioItemRepo,
		transcriptRepo,
		courseRepo,
		iamSvc,
		learnSvc,
		discoverySvc,
		mediaSvc,
		shared.NewEventBus(),
	)
}

// Suppress unused import warnings for test scaffolding.
var (
	_ = json.RawMessage{}
	_ = time.Now
	_ = testScope
)
