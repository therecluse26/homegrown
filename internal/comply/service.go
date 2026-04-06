package comply

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/comply/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Implementation [14-comply §5]
// ═══════════════════════════════════════════════════════════════════════════════

// ComplianceServiceImpl implements ComplianceService.
type ComplianceServiceImpl struct {
	stateConfigRepo   StateConfigRepository
	familyConfigRepo  FamilyConfigRepository
	scheduleRepo      ScheduleRepository
	attendanceRepo    AttendanceRepository
	assessmentRepo    AssessmentRepository
	testRepo          TestScoreRepository
	portfolioRepo     PortfolioRepository
	portfolioItemRepo PortfolioItemRepository
	transcriptRepo    TranscriptRepository
	courseRepo        CourseRepository
	iamSvc            IamServiceForComply
	learnSvc          LearningServiceForComply
	discoverySvc      DiscoveryServiceForComply
	mediaSvc          MediaServiceForComply
	events            *shared.EventBus
}

// NewComplianceService creates a ComplianceService backed by the provided dependencies.
func NewComplianceService(
	stateConfigRepo StateConfigRepository,
	familyConfigRepo FamilyConfigRepository,
	scheduleRepo ScheduleRepository,
	attendanceRepo AttendanceRepository,
	assessmentRepo AssessmentRepository,
	testRepo TestScoreRepository,
	portfolioRepo PortfolioRepository,
	portfolioItemRepo PortfolioItemRepository,
	transcriptRepo TranscriptRepository,
	courseRepo CourseRepository,
	iamSvc IamServiceForComply,
	learnSvc LearningServiceForComply,
	discoverySvc DiscoveryServiceForComply,
	mediaSvc MediaServiceForComply,
	events *shared.EventBus,
) ComplianceService {
	return &ComplianceServiceImpl{
		stateConfigRepo:   stateConfigRepo,
		familyConfigRepo:  familyConfigRepo,
		scheduleRepo:      scheduleRepo,
		attendanceRepo:    attendanceRepo,
		assessmentRepo:    assessmentRepo,
		testRepo:          testRepo,
		portfolioRepo:     portfolioRepo,
		portfolioItemRepo: portfolioItemRepo,
		transcriptRepo:    transcriptRepo,
		courseRepo:         courseRepo,
		iamSvc:            iamSvc,
		learnSvc:          learnSvc,
		discoverySvc:      discoverySvc,
		mediaSvc:          mediaSvc,
		events:            events,
	}
}

// maxPortfolioRetries is the default maximum retry count for portfolio generation.
const maxPortfolioRetries = 3

// ─── Command side ─────────────────────────────────────────────────────────────

func (s *ComplianceServiceImpl) UpsertFamilyConfig(ctx context.Context, cmd UpsertFamilyConfigCommand, scope shared.FamilyScope) (*FamilyConfigResponse, error) {
	// Validate state code exists
	state, err := s.stateConfigRepo.FindByStateCode(ctx, cmd.StateCode)
	if err != nil {
		if errors.Is(err, ErrStateConfigNotFound) {
			return nil, ErrInvalidStateCode
		}
		return nil, err
	}

	// Apply defaults for simplified setup flow (state + days/hours only). [M16]
	now := time.Now().UTC()
	schoolYearStart := time.Date(now.Year(), time.September, 1, 0, 0, 0, 0, time.UTC)
	if cmd.SchoolYearStart != nil {
		schoolYearStart = *cmd.SchoolYearStart
	}
	schoolYearEnd := time.Date(now.Year()+1, time.June, 30, 0, 0, 0, 0, time.UTC)
	if cmd.SchoolYearEnd != nil {
		schoolYearEnd = *cmd.SchoolYearEnd
	}

	// Validate school year range
	if !schoolYearEnd.After(schoolYearStart) {
		return nil, ErrInvalidSchoolYearRange
	}

	totalSchoolDays := int16(180) // default
	if cmd.TotalSchoolDays != nil {
		totalSchoolDays = *cmd.TotalSchoolDays
	} else if cmd.DaysRequired != nil {
		totalSchoolDays = *cmd.DaysRequired
	}

	gpaScale := "standard_4"
	if cmd.GpaScale != nil {
		gpaScale = *cmd.GpaScale
	}

	// Validate custom schedule belongs to family if provided
	if cmd.CustomScheduleID != nil {
		if _, err := s.scheduleRepo.FindByID(ctx, *cmd.CustomScheduleID, scope); err != nil {
			return nil, err
		}
	}

	row := UpsertFamilyConfigRow{
		StateCode:        cmd.StateCode,
		SchoolYearStart:  schoolYearStart,
		SchoolYearEnd:    schoolYearEnd,
		TotalSchoolDays:  totalSchoolDays,
		CustomScheduleID: cmd.CustomScheduleID,
		GpaScale:         gpaScale,
		GpaCustomConfig:  cmd.GpaCustomConfig,
	}

	config, err := s.familyConfigRepo.Upsert(ctx, scope, row)
	if err != nil {
		return nil, err
	}
	return mapFamilyConfigToResponse(config, state.StateName), nil
}

func (s *ComplianceServiceImpl) CreateSchedule(ctx context.Context, cmd CreateScheduleCommand, scope shared.FamilyScope) (*ScheduleResponse, error) {
	if len(cmd.SchoolDays) != 7 {
		return nil, ErrInvalidSchoolDaysArray
	}

	epJSON, err := json.Marshal(cmd.ExclusionPeriods)
	if err != nil {
		return nil, err
	}

	sched, err := s.scheduleRepo.Create(ctx, scope, CreateScheduleRow{
		Name:             cmd.Name,
		SchoolDays:       cmd.SchoolDays,
		ExclusionPeriods: epJSON,
	})
	if err != nil {
		return nil, err
	}
	return mapScheduleToResponse(sched), nil
}

func (s *ComplianceServiceImpl) UpdateSchedule(ctx context.Context, scheduleID uuid.UUID, cmd UpdateScheduleCommand, scope shared.FamilyScope) (*ScheduleResponse, error) {
	if _, err := s.scheduleRepo.FindByID(ctx, scheduleID, scope); err != nil {
		return nil, err
	}

	// Validate school days length if provided
	if cmd.SchoolDays != nil && len(*cmd.SchoolDays) != 7 {
		return nil, ErrInvalidSchoolDaysArray
	}

	var epJSON *json.RawMessage
	if cmd.ExclusionPeriods != nil {
		data, err := json.Marshal(cmd.ExclusionPeriods)
		if err != nil {
			return nil, err
		}
		raw := json.RawMessage(data)
		epJSON = &raw
	}

	updated, err := s.scheduleRepo.Update(ctx, scheduleID, scope, UpdateScheduleRow{
		Name:             cmd.Name,
		SchoolDays:       cmd.SchoolDays,
		ExclusionPeriods: epJSON,
	})
	if err != nil {
		return nil, err
	}
	return mapScheduleToResponse(updated), nil
}

func (s *ComplianceServiceImpl) DeleteSchedule(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) error {
	if _, err := s.scheduleRepo.FindByID(ctx, scheduleID, scope); err != nil {
		return err
	}

	// Check if schedule is in use by family config
	config, err := s.familyConfigRepo.FindByFamily(ctx, scope)
	if err != nil && !errors.Is(err, ErrFamilyConfigNotFound) {
		return err
	}
	if config != nil && config.CustomScheduleID != nil && *config.CustomScheduleID == scheduleID {
		return ErrScheduleInUse
	}

	return s.scheduleRepo.Delete(ctx, scheduleID, scope)
}

func (s *ComplianceServiceImpl) RecordAttendance(ctx context.Context, studentID uuid.UUID, cmd RecordAttendanceCommand, scope shared.FamilyScope) (*AttendanceResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	if err := domain.ValidateAttendanceRecord(cmd.AttendanceDate, cmd.Status, cmd.DurationMinutes, today); err != nil {
		return nil, err
	}

	record, err := s.attendanceRepo.Upsert(ctx, scope, UpsertAttendanceRow{
		StudentID:       studentID,
		AttendanceDate:  cmd.AttendanceDate,
		Status:          cmd.Status,
		DurationMinutes: cmd.DurationMinutes,
		Notes:           cmd.Notes,
		IsAuto:          false,
		ManualOverride:  true,
	})
	if err != nil {
		return nil, err
	}
	return mapAttendanceToResponse(record), nil
}

func (s *ComplianceServiceImpl) BulkRecordAttendance(ctx context.Context, studentID uuid.UUID, cmd BulkRecordAttendanceCommand, scope shared.FamilyScope) ([]AttendanceResponse, error) {
	if len(cmd.Records) > 31 {
		return nil, ErrBulkAttendanceLimitExceeded
	}

	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	for _, rec := range cmd.Records {
		if err := domain.ValidateAttendanceRecord(rec.AttendanceDate, rec.Status, rec.DurationMinutes, today); err != nil {
			return nil, err
		}
	}

	results := make([]AttendanceResponse, 0, len(cmd.Records))
	for _, rec := range cmd.Records {
		record, err := s.attendanceRepo.Upsert(ctx, scope, UpsertAttendanceRow{
			StudentID:       studentID,
			AttendanceDate:  rec.AttendanceDate,
			Status:          rec.Status,
			DurationMinutes: rec.DurationMinutes,
			Notes:           rec.Notes,
			IsAuto:          false,
			ManualOverride:  true,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, *mapAttendanceToResponse(record))
	}
	return results, nil
}

func (s *ComplianceServiceImpl) UpdateAttendance(ctx context.Context, studentID uuid.UUID, attendanceID uuid.UUID, cmd UpdateAttendanceCommand, scope shared.FamilyScope) (*AttendanceResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	if _, err := s.attendanceRepo.FindByID(ctx, attendanceID, scope); err != nil {
		return nil, err
	}

	updated, err := s.attendanceRepo.Update(ctx, attendanceID, scope, UpdateAttendanceRow(cmd))
	if err != nil {
		return nil, err
	}
	return mapAttendanceToResponse(updated), nil
}

func (s *ComplianceServiceImpl) DeleteAttendance(ctx context.Context, studentID uuid.UUID, attendanceID uuid.UUID, scope shared.FamilyScope) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}

	if _, err := s.attendanceRepo.FindByID(ctx, attendanceID, scope); err != nil {
		return err
	}

	return s.attendanceRepo.Delete(ctx, attendanceID, scope)
}

func (s *ComplianceServiceImpl) CreateAssessment(ctx context.Context, studentID uuid.UUID, cmd CreateAssessmentCommand, scope shared.FamilyScope) (*AssessmentResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	record, err := s.assessmentRepo.Create(ctx, scope, CreateAssessmentRow{
		StudentID:        studentID,
		Title:            cmd.Title,
		Subject:          cmd.Subject,
		AssessmentType:   cmd.AssessmentType,
		Score:            cmd.Score,
		MaxScore:         cmd.MaxScore,
		GradeLetter:      cmd.GradeLetter,
		GradePoints:      cmd.GradePoints,
		IsPassing:        cmd.IsPassing,
		SourceActivityID: cmd.SourceActivityID,
		AssessmentDate:   cmd.AssessmentDate,
		Notes:            cmd.Notes,
	})
	if err != nil {
		return nil, err
	}
	return mapAssessmentToResponse(record), nil
}

func (s *ComplianceServiceImpl) UpdateAssessment(ctx context.Context, studentID uuid.UUID, assessmentID uuid.UUID, cmd UpdateAssessmentCommand, scope shared.FamilyScope) (*AssessmentResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	if _, err := s.assessmentRepo.FindByID(ctx, assessmentID, scope); err != nil {
		return nil, err
	}

	updated, err := s.assessmentRepo.Update(ctx, assessmentID, scope, UpdateAssessmentRow(cmd))
	if err != nil {
		return nil, err
	}
	return mapAssessmentToResponse(updated), nil
}

func (s *ComplianceServiceImpl) DeleteAssessment(ctx context.Context, studentID uuid.UUID, assessmentID uuid.UUID, scope shared.FamilyScope) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}

	record, err := s.assessmentRepo.FindByID(ctx, assessmentID, scope)
	if err != nil {
		return err
	}
	if record == nil {
		return ErrAssessmentNotFound
	}

	return s.assessmentRepo.Delete(ctx, assessmentID, scope)
}

func (s *ComplianceServiceImpl) CreateTestScore(ctx context.Context, studentID uuid.UUID, cmd CreateTestScoreCommand, scope shared.FamilyScope) (*TestScoreResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	record, err := s.testRepo.Create(ctx, scope, CreateTestScoreRow{
		StudentID:      studentID,
		TestName:       cmd.TestName,
		TestDate:       cmd.TestDate,
		GradeLevel:     cmd.GradeLevel,
		Scores:         cmd.Scores,
		CompositeScore: cmd.CompositeScore,
		Percentile:     cmd.Percentile,
		Notes:          cmd.Notes,
	})
	if err != nil {
		return nil, err
	}
	return mapTestScoreToResponse(record), nil
}

func (s *ComplianceServiceImpl) UpdateTestScore(ctx context.Context, studentID uuid.UUID, testID uuid.UUID, cmd UpdateTestScoreCommand, scope shared.FamilyScope) (*TestScoreResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	updated, err := s.testRepo.Update(ctx, testID, scope, UpdateTestScoreRow(cmd))
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrTestScoreNotFound
	}
	return mapTestScoreToResponse(updated), nil
}

func (s *ComplianceServiceImpl) DeleteTestScore(ctx context.Context, studentID uuid.UUID, testID uuid.UUID, scope shared.FamilyScope) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}
	return s.testRepo.Delete(ctx, testID, scope)
}

func (s *ComplianceServiceImpl) CreatePortfolio(ctx context.Context, studentID uuid.UUID, cmd CreatePortfolioCommand, scope shared.FamilyScope) (*PortfolioResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	if !cmd.DateRangeEnd.After(cmd.DateRangeStart) {
		return nil, ErrInvalidSchoolYearRange
	}

	portfolio, err := s.portfolioRepo.Create(ctx, scope, CreatePortfolioRow{
		StudentID:          studentID,
		Title:              cmd.Title,
		Description:        cmd.Description,
		Organization:       cmd.Organization,
		DateRangeStart:     cmd.DateRangeStart,
		DateRangeEnd:       cmd.DateRangeEnd,
		IncludeAttendance:  cmd.IncludeAttendance,
		IncludeAssessments: cmd.IncludeAssessments,
	})
	if err != nil {
		return nil, err
	}
	return mapPortfolioToResponse(portfolio, nil), nil
}

func (s *ComplianceServiceImpl) AddPortfolioItems(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, cmd AddPortfolioItemsCommand, scope shared.FamilyScope) ([]PortfolioItemResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	portfolio, err := s.portfolioRepo.FindByID(ctx, portfolioID, scope)
	if err != nil {
		return nil, err
	}
	if portfolio.Status != string(PortfolioStatusConfiguring) {
		return nil, domain.ErrPortfolioNotConfiguring
	}

	// Fetch display data from learn:: and build item rows
	rows := make([]CreatePortfolioItemRow, 0, len(cmd.Items))
	for i, item := range cmd.Items {
		data, err := s.learnSvc.GetPortfolioItemData(ctx, scope.FamilyID(), item.SourceType, item.SourceID)
		if err != nil {
			return nil, err
		}
		if data == nil {
			return nil, ErrPortfolioItemSourceNotFound
		}
		rows = append(rows, CreatePortfolioItemRow{
			PortfolioID:       portfolioID,
			SourceType:        item.SourceType,
			SourceID:          item.SourceID,
			DisplayOrder:      int16(i + 1),
			CachedTitle:       data.Title,
			CachedSubject:     data.Subject,
			CachedDate:        data.Date,
			CachedDescription: data.Description,
			CachedAttachments: data.Attachments,
		})
	}

	items, err := s.portfolioItemRepo.CreateBatch(ctx, rows)
	if err != nil {
		return nil, err
	}

	results := make([]PortfolioItemResponse, len(items))
	for i, item := range items {
		results[i] = mapPortfolioItemToResponse(&item)
	}
	return results, nil
}

func (s *ComplianceServiceImpl) GeneratePortfolio(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, scope shared.FamilyScope) (*PortfolioResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	portfolio, err := s.portfolioRepo.FindByID(ctx, portfolioID, scope)
	if err != nil {
		return nil, err
	}

	// Count items
	items, err := s.portfolioItemRepo.ListByPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	if err := domain.ValidatePortfolioGenerate(portfolio.Status, int32(len(items)), portfolio.RetryCount, maxPortfolioRetries); err != nil {
		return nil, err
	}

	updated, err := s.portfolioRepo.UpdateStatus(ctx, portfolioID, string(PortfolioStatusGenerating), nil, nil)
	if err != nil {
		return nil, err
	}
	return mapPortfolioToResponse(updated, nil), nil
}

func (s *ComplianceServiceImpl) CreateTranscript(ctx context.Context, studentID uuid.UUID, cmd CreateTranscriptCommand, scope shared.FamilyScope) (*TranscriptResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	studentName, err := s.iamSvc.GetStudentName(ctx, studentID)
	if err != nil {
		return nil, err
	}

	transcript, err := s.transcriptRepo.Create(ctx, scope, CreateTranscriptRow{
		StudentID:   studentID,
		Title:       cmd.Title,
		StudentName: studentName,
		GradeLevels: cmd.GradeLevels,
	})
	if err != nil {
		return nil, err
	}
	return mapTranscriptToResponse(transcript, nil, nil), nil
}

func (s *ComplianceServiceImpl) GenerateTranscript(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) (*TranscriptResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	transcript, err := s.transcriptRepo.FindByID(ctx, transcriptID, scope)
	if err != nil {
		return nil, err
	}

	if err := domain.ValidateTranscriptTransition(transcript.Status, string(PortfolioStatusGenerating)); err != nil {
		return nil, err
	}

	updated, err := s.transcriptRepo.UpdateStatus(ctx, transcriptID, string(PortfolioStatusGenerating), nil, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	return mapTranscriptToResponse(updated, nil, nil), nil
}

func (s *ComplianceServiceImpl) DeleteTranscript(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}

	if _, err := s.transcriptRepo.FindByID(ctx, transcriptID, scope); err != nil {
		return err
	}

	return s.transcriptRepo.Delete(ctx, transcriptID, scope)
}

func (s *ComplianceServiceImpl) CreateCourse(ctx context.Context, studentID uuid.UUID, cmd CreateCourseCommand, scope shared.FamilyScope) (*CourseResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	course, err := s.courseRepo.Create(ctx, scope, CreateCourseRow{
		StudentID:   studentID,
		Title:       cmd.Title,
		Subject:     cmd.Subject,
		GradeLevel:  cmd.GradeLevel,
		Credits:     cmd.Credits,
		GradeLetter: cmd.GradeLetter,
		GradePoints: cmd.GradePoints,
		Level:       cmd.Level,
		SchoolYear:  cmd.SchoolYear,
		Semester:    cmd.Semester,
	})
	if err != nil {
		return nil, err
	}
	return mapCourseToResponse(course), nil
}

func (s *ComplianceServiceImpl) UpdateCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID, cmd UpdateCourseCommand, scope shared.FamilyScope) (*CourseResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	updated, err := s.courseRepo.Update(ctx, courseID, scope, UpdateCourseRow(cmd))
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrCourseNotFound
	}
	return mapCourseToResponse(updated), nil
}

func (s *ComplianceServiceImpl) DeleteCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID, scope shared.FamilyScope) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}
	return s.courseRepo.Delete(ctx, courseID, scope)
}

// ─── Query side ───────────────────────────────────────────────────────────────

func (s *ComplianceServiceImpl) GetFamilyConfig(ctx context.Context, scope shared.FamilyScope) (*FamilyConfigResponse, error) {
	config, err := s.familyConfigRepo.FindByFamily(ctx, scope)
	if err != nil {
		if errors.Is(err, ErrFamilyConfigNotFound) {
			return nil, nil // no config yet — valid state
		}
		return nil, err
	}

	state, err := s.stateConfigRepo.FindByStateCode(ctx, config.StateCode)
	if err != nil && !errors.Is(err, ErrStateConfigNotFound) {
		return nil, err
	}
	stateName := ""
	if state != nil {
		stateName = state.StateName
	}
	return mapFamilyConfigToResponse(config, stateName), nil
}

func (s *ComplianceServiceImpl) ListStateConfigs(ctx context.Context) ([]StateConfigSummaryResponse, error) {
	configs, err := s.stateConfigRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]StateConfigSummaryResponse, len(configs))
	for i, c := range configs {
		results[i] = StateConfigSummaryResponse{
			StateCode:          c.StateCode,
			StateName:          c.StateName,
			RegulationLevel:    c.RegulationLevel,
			AttendanceRequired: c.AttendanceRequired,
			AttendanceDays:     c.AttendanceDays,
		}
	}
	return results, nil
}

func (s *ComplianceServiceImpl) GetStateConfig(ctx context.Context, stateCode string) (*StateConfigResponse, error) {
	config, err := s.stateConfigRepo.FindByStateCode(ctx, stateCode)
	if err != nil {
		return nil, err
	}
	return mapStateConfigToResponse(config), nil
}

func (s *ComplianceServiceImpl) ListSchedules(ctx context.Context, scope shared.FamilyScope) ([]ScheduleResponse, error) {
	schedules, err := s.scheduleRepo.ListByFamily(ctx, scope)
	if err != nil {
		return nil, err
	}
	results := make([]ScheduleResponse, len(schedules))
	for i, sched := range schedules {
		results[i] = *mapScheduleToResponse(&sched)
	}
	return results, nil
}

func (s *ComplianceServiceImpl) ListAttendance(ctx context.Context, studentID uuid.UUID, params AttendanceListParams, scope shared.FamilyScope) (*AttendanceListResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	records, err := s.attendanceRepo.ListByStudent(ctx, studentID, scope, &params)
	if err != nil {
		return nil, err
	}

	results := make([]AttendanceResponse, len(records))
	for i, rec := range records {
		results[i] = *mapAttendanceToResponse(&rec)
	}
	return &AttendanceListResponse{Records: results}, nil
}

func (s *ComplianceServiceImpl) GetAttendanceSummary(ctx context.Context, studentID uuid.UUID, params AttendanceSummaryParams, scope shared.FamilyScope) (*AttendanceSummaryResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	summary, err := s.attendanceRepo.Summarize(ctx, studentID, scope, params.StartDate, params.EndDate)
	if err != nil {
		return nil, err
	}

	totalDays := summary.PresentFull + summary.PresentPartial + summary.Absent + summary.NotApplicable
	totalHours := float64(summary.TotalMinutes) / 60.0

	resp := &AttendanceSummaryResponse{
		TotalDays:      totalDays,
		PresentFull:    summary.PresentFull,
		PresentPartial: summary.PresentPartial,
		Absent:         summary.Absent,
		NotApplicable:  summary.NotApplicable,
		TotalHours:     totalHours,
	}

	// Get family config for pace calculation
	config, err := s.familyConfigRepo.FindByFamily(ctx, scope)
	if err != nil && !errors.Is(err, ErrFamilyConfigNotFound) {
		return nil, err
	}
	if config != nil {
		stateConfig, err := s.stateConfigRepo.FindByStateCode(ctx, config.StateCode)
		if err != nil && !errors.Is(err, ErrStateConfigNotFound) {
			return nil, err
		}
		if stateConfig != nil && stateConfig.AttendanceRequired {
			resp.StateRequiredDays = stateConfig.AttendanceDays
			resp.StateRequiredHours = stateConfig.AttendanceHours

			presentDays := summary.PresentFull + summary.PresentPartial
			elapsedDays := totalDays
			pace := domain.CalculatePace(presentDays, elapsedDays, int32(config.TotalSchoolDays), stateConfig.AttendanceDays)
			paceStr := string(pace)
			resp.PaceStatus = &paceStr
		}
	}

	return resp, nil
}

func (s *ComplianceServiceImpl) ListAssessments(ctx context.Context, studentID uuid.UUID, params AssessmentListParams, scope shared.FamilyScope) (*AssessmentListResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	records, err := s.assessmentRepo.ListByStudent(ctx, studentID, scope, &params)
	if err != nil {
		return nil, err
	}

	results := make([]AssessmentResponse, len(records))
	for i, rec := range records {
		results[i] = *mapAssessmentToResponse(&rec)
	}
	return &AssessmentListResponse{Records: results}, nil
}

func (s *ComplianceServiceImpl) ListTestScores(ctx context.Context, studentID uuid.UUID, params TestListParams, scope shared.FamilyScope) (*TestListResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	records, err := s.testRepo.ListByStudent(ctx, studentID, scope, &params)
	if err != nil {
		return nil, err
	}

	results := make([]TestScoreResponse, len(records))
	for i, rec := range records {
		results[i] = *mapTestScoreToResponse(&rec)
	}
	return &TestListResponse{Tests: results}, nil
}

func (s *ComplianceServiceImpl) GetPortfolio(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, scope shared.FamilyScope) (*PortfolioResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	portfolio, err := s.portfolioRepo.FindByID(ctx, portfolioID, scope)
	if err != nil {
		return nil, err
	}

	items, err := s.portfolioItemRepo.ListByPortfolio(ctx, portfolioID)
	if err != nil {
		return nil, err
	}

	return mapPortfolioToResponse(portfolio, items), nil
}

func (s *ComplianceServiceImpl) ListPortfolios(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]PortfolioSummaryResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	portfolios, err := s.portfolioRepo.ListByStudent(ctx, studentID, scope)
	if err != nil {
		return nil, err
	}

	results := make([]PortfolioSummaryResponse, len(portfolios))
	for i, p := range portfolios {
		count, err := s.portfolioItemRepo.CountByPortfolio(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		results[i] = PortfolioSummaryResponse{
			ID:             p.ID,
			Title:          p.Title,
			Status:         p.Status,
			ItemCount:      count,
			DateRangeStart: p.DateRangeStart,
			DateRangeEnd:   p.DateRangeEnd,
			GeneratedAt:    p.GeneratedAt,
			ExpiresAt:      p.ExpiresAt,
			CreatedAt:      p.CreatedAt,
		}
	}
	return results, nil
}

func (s *ComplianceServiceImpl) GetPortfolioDownloadURL(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, scope shared.FamilyScope) (string, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return "", err
	}

	portfolio, err := s.portfolioRepo.FindByID(ctx, portfolioID, scope)
	if err != nil {
		return "", err
	}

	if portfolio.Status != string(PortfolioStatusReady) {
		return "", ErrPortfolioNotReady
	}

	// Check expiry
	if portfolio.ExpiresAt != nil && portfolio.ExpiresAt.Before(time.Now().UTC()) {
		return "", ErrPortfolioExpired
	}

	return s.mediaSvc.PresignedGet(ctx, *portfolio.UploadID)
}

func (s *ComplianceServiceImpl) GetDashboard(ctx context.Context, scope shared.FamilyScope) (*ComplianceDashboardResponse, error) {
	resp := &ComplianceDashboardResponse{
		Students: []StudentComplianceSummary{},
	}

	config, err := s.familyConfigRepo.FindByFamily(ctx, scope)
	if err != nil && !errors.Is(err, ErrFamilyConfigNotFound) {
		return nil, err
	}

	if config != nil {
		state, err := s.stateConfigRepo.FindByStateCode(ctx, config.StateCode)
		if err != nil && !errors.Is(err, ErrStateConfigNotFound) {
			return nil, err
		}
		stateName := ""
		if state != nil {
			stateName = state.StateName
		}
		resp.FamilyConfig = mapFamilyConfigToResponse(config, stateName)
	}

	return resp, nil
}

func (s *ComplianceServiceImpl) GetTranscript(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) (*TranscriptResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	transcript, err := s.transcriptRepo.FindByID(ctx, transcriptID, scope)
	if err != nil {
		return nil, err
	}

	courses, err := s.courseRepo.ListByStudent(ctx, studentID, scope, nil)
	if err != nil {
		return nil, err
	}

	courseResponses := make([]CourseResponse, len(courses))
	for i, c := range courses {
		courseResponses[i] = *mapCourseToResponse(&c)
	}

	scale, customConfig, err := s.getGpaScaleForFamily(ctx, scope)
	if err != nil {
		return nil, err
	}

	gpaInput := coursesToGpaInput(courses)
	gpaResult := domain.CalculateGPA(gpaInput, scale, customConfig)
	gpaResp := &GpaResponse{
		UnweightedGPA: gpaResult.Unweighted,
		WeightedGPA:   gpaResult.Weighted,
		TotalCredits:  gpaResult.TotalCredits,
		TotalCourses:  int32(len(courses)),
	}

	return mapTranscriptToResponse(transcript, courseResponses, gpaResp), nil
}

func (s *ComplianceServiceImpl) ListTranscripts(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]TranscriptSummaryResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	transcripts, err := s.transcriptRepo.ListByStudent(ctx, studentID, scope)
	if err != nil {
		return nil, err
	}

	results := make([]TranscriptSummaryResponse, len(transcripts))
	for i, t := range transcripts {
		results[i] = mapTranscriptToSummary(&t)
	}
	return results, nil
}

func (s *ComplianceServiceImpl) GetTranscriptDownloadURL(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) (string, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return "", err
	}

	transcript, err := s.transcriptRepo.FindByID(ctx, transcriptID, scope)
	if err != nil {
		return "", err
	}

	if transcript.Status != string(PortfolioStatusReady) {
		return "", ErrPortfolioNotReady
	}

	if transcript.ExpiresAt != nil && transcript.ExpiresAt.Before(time.Now().UTC()) {
		return "", ErrPortfolioExpired
	}

	return s.mediaSvc.PresignedGet(ctx, *transcript.UploadID)
}

func (s *ComplianceServiceImpl) ListCourses(ctx context.Context, studentID uuid.UUID, params CourseListParams, scope shared.FamilyScope) (*CourseListResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	courses, err := s.courseRepo.ListByStudent(ctx, studentID, scope, &params)
	if err != nil {
		return nil, err
	}

	results := make([]CourseResponse, len(courses))
	for i, c := range courses {
		results[i] = *mapCourseToResponse(&c)
	}
	return &CourseListResponse{Courses: results}, nil
}

func (s *ComplianceServiceImpl) CalculateGPA(ctx context.Context, studentID uuid.UUID, params GpaParams, scope shared.FamilyScope) (*GpaResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	allCourses, err := s.courseRepo.ListByStudent(ctx, studentID, scope, nil)
	if err != nil {
		return nil, err
	}

	// Filter by grade levels if specified. [14-comply §4.3]
	courses := allCourses
	if len(params.GradeLevels) > 0 {
		glSet := make(map[int16]bool, len(params.GradeLevels))
		for _, gl := range params.GradeLevels {
			glSet[gl] = true
		}
		filtered := make([]ComplyCourse, 0, len(allCourses))
		for _, c := range allCourses {
			if glSet[c.GradeLevel] {
				filtered = append(filtered, c)
			}
		}
		courses = filtered
	}

	scale, customConfig, err := s.getGpaScaleForFamily(ctx, scope)
	if err != nil {
		return nil, err
	}
	if params.Scale != nil {
		scale = domain.GpaScale(*params.Scale)
	}

	gpaInput := coursesToGpaInput(courses)
	overall := domain.CalculateGPA(gpaInput, scale, customConfig)

	// Group by grade level for breakdown
	gradeGroups := make(map[int16][]ComplyCourse)
	for _, c := range courses {
		gradeGroups[c.GradeLevel] = append(gradeGroups[c.GradeLevel], c)
	}

	byGradeLevel := make([]GpaGradeLevelResponse, 0, len(gradeGroups))
	for gl, groupCourses := range gradeGroups {
		groupInput := coursesToGpaInput(groupCourses)
		result := domain.CalculateGPA(groupInput, scale, customConfig)
		byGradeLevel = append(byGradeLevel, GpaGradeLevelResponse{
			GradeLevel: gl,
			Unweighted: result.Unweighted,
			Weighted:   result.Weighted,
			Credits:    result.TotalCredits,
		})
	}
	sort.Slice(byGradeLevel, func(i, j int) bool {
		return byGradeLevel[i].GradeLevel < byGradeLevel[j].GradeLevel
	})

	return &GpaResponse{
		UnweightedGPA: overall.Unweighted,
		WeightedGPA:   overall.Weighted,
		TotalCredits:  overall.TotalCredits,
		TotalCourses:  int32(len(courses)),
		ByGradeLevel:  byGradeLevel,
	}, nil
}

func (s *ComplianceServiceImpl) CalculateGPAWhatIf(ctx context.Context, studentID uuid.UUID, params GpaWhatIfParams, scope shared.FamilyScope) (*GpaResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	courses, err := s.courseRepo.ListByStudent(ctx, studentID, scope, nil)
	if err != nil {
		return nil, err
	}

	scale, customConfig, err := s.getGpaScaleForFamily(ctx, scope)
	if err != nil {
		return nil, err
	}

	gpaInput := coursesToGpaInput(courses)
	for _, wc := range params.AdditionalCourses {
		gp := wc.GradePoints
		gpaInput = append(gpaInput, domain.CourseForGpa{
			GradePoints: &gp,
			Credits:     wc.Credits,
			Level:       wc.Level,
		})
	}

	result := domain.CalculateGPA(gpaInput, scale, customConfig)

	return &GpaResponse{
		UnweightedGPA: result.Unweighted,
		WeightedGPA:   result.Weighted,
		TotalCredits:  result.TotalCredits,
		TotalCourses:  int32(len(gpaInput)),
	}, nil
}

func (s *ComplianceServiceImpl) GetGPAHistory(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]GpaTermResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}

	courses, err := s.courseRepo.ListByStudent(ctx, studentID, scope, nil)
	if err != nil {
		return nil, err
	}

	scale, customConfig, err := s.getGpaScaleForFamily(ctx, scope)
	if err != nil {
		return nil, err
	}

	type termKey struct {
		SchoolYear string
		Semester   string
	}
	termGroups := make(map[termKey][]ComplyCourse)
	termOrder := make([]termKey, 0)
	for _, c := range courses {
		sem := ""
		if c.Semester != nil {
			sem = *c.Semester
		}
		key := termKey{SchoolYear: c.SchoolYear, Semester: sem}
		if _, exists := termGroups[key]; !exists {
			termOrder = append(termOrder, key)
		}
		termGroups[key] = append(termGroups[key], c)
	}

	sort.Slice(termOrder, func(i, j int) bool {
		if termOrder[i].SchoolYear != termOrder[j].SchoolYear {
			return termOrder[i].SchoolYear < termOrder[j].SchoolYear
		}
		return termOrder[i].Semester < termOrder[j].Semester
	})

	results := make([]GpaTermResponse, 0, len(termGroups))
	for _, key := range termOrder {
		groupCourses := termGroups[key]
		groupInput := coursesToGpaInput(groupCourses)
		result := domain.CalculateGPA(groupInput, scale, customConfig)
		resp := GpaTermResponse{
			SchoolYear:    key.SchoolYear,
			UnweightedGPA: result.Unweighted,
			WeightedGPA:   result.Weighted,
			Credits:       result.TotalCredits,
			CourseCount:   int32(len(groupCourses)),
		}
		if key.Semester != "" {
			sem := key.Semester
			resp.Semester = &sem
		}
		results = append(results, resp)
	}
	return results, nil
}

// ─── Event handlers ───────────────────────────────────────────────────────────

func (s *ComplianceServiceImpl) HandleActivityLogged(ctx context.Context, event *ActivityLoggedEvent) error {
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: event.FamilyID})

	// Check if a manual record already exists for this date — do not override. [14-comply §17.4]
	existing, err := s.attendanceRepo.FindByStudentAndDate(ctx, event.StudentID, scope, event.ActivityDate)
	if err != nil {
		if !errors.Is(err, ErrAttendanceNotFound) {
			return err
		}
		// No existing record — proceed to upsert.
	} else if existing.ManualOverride {
		return nil // manual record takes precedence
	}

	_, err = s.attendanceRepo.Upsert(ctx, scope, UpsertAttendanceRow{
		StudentID:       event.StudentID,
		AttendanceDate:  event.ActivityDate,
		Status:          string(AttendanceStatusPresentFull),
		DurationMinutes: event.DurationMinutes,
		IsAuto:          true,
		ManualOverride:  false,
	})
	return err
}

func (s *ComplianceServiceImpl) HandleStudentDeleted(ctx context.Context, event *StudentDeletedEvent) error {
	// Cascade delete all student compliance data. [14-comply §17.4]
	// Order: items before portfolios (FK dependency).
	if err := s.portfolioItemRepo.DeleteByStudent(ctx, event.StudentID, event.FamilyID); err != nil {
		return err
	}
	if err := s.attendanceRepo.DeleteByStudent(ctx, event.StudentID, event.FamilyID); err != nil {
		return err
	}
	if err := s.assessmentRepo.DeleteByStudent(ctx, event.StudentID, event.FamilyID); err != nil {
		return err
	}
	if err := s.testRepo.DeleteByStudent(ctx, event.StudentID, event.FamilyID); err != nil {
		return err
	}
	if err := s.portfolioRepo.DeleteByStudent(ctx, event.StudentID, event.FamilyID); err != nil {
		return err
	}
	if err := s.courseRepo.DeleteByStudent(ctx, event.StudentID, event.FamilyID); err != nil {
		return err
	}
	if err := s.transcriptRepo.DeleteByStudent(ctx, event.StudentID, event.FamilyID); err != nil {
		return err
	}
	return nil
}

func (s *ComplianceServiceImpl) HandleFamilyDeletionScheduled(ctx context.Context, event *FamilyDeletionScheduledEvent) error {
	// Cascade delete all family compliance data. [14-comply §17.4]
	// Order: items before portfolios (FK dependency), config + schedules last.
	if err := s.portfolioItemRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		return err
	}
	if err := s.attendanceRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		return err
	}
	if err := s.assessmentRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		return err
	}
	if err := s.testRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		return err
	}
	if err := s.portfolioRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		return err
	}
	if err := s.courseRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		return err
	}
	if err := s.transcriptRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		return err
	}
	if err := s.scheduleRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		return err
	}
	if err := s.familyConfigRepo.DeleteByFamily(ctx, event.FamilyID); err != nil {
		return err
	}
	return nil
}

func (s *ComplianceServiceImpl) HandleSubscriptionCancelled(_ context.Context, _ *SubscriptionCancelledEvent) error {
	// Preserve data — no deletion on subscription cancellation
	return nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (s *ComplianceServiceImpl) verifyStudentInFamily(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) error {
	belongs, err := s.iamSvc.StudentBelongsToFamily(ctx, studentID, shared.NewFamilyID(scope.FamilyID()))
	if err != nil {
		return err
	}
	if !belongs {
		return ErrStudentNotInFamily
	}
	return nil
}

// ─── Mappers ──────────────────────────────────────────────────────────────────

func mapFamilyConfigToResponse(c *ComplyFamilyConfig, stateName string) *FamilyConfigResponse {
	return &FamilyConfigResponse{
		FamilyID:         c.FamilyID,
		StateCode:        c.StateCode,
		StateName:        stateName,
		SchoolYearStart:  c.SchoolYearStart,
		SchoolYearEnd:    c.SchoolYearEnd,
		TotalSchoolDays:  c.TotalSchoolDays,
		DaysRequired:     c.TotalSchoolDays, // alias for simplified frontend flow [M16]
		HoursRequired:    0,                 // not stored in DB; frontend derives from state requirements
		CustomScheduleID: c.CustomScheduleID,
		GpaScale:         c.GpaScale,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func mapStateConfigToResponse(c *ComplyStateConfig) *StateConfigResponse {
	return &StateConfigResponse{
		StateCode:             c.StateCode,
		StateName:             c.StateName,
		NotificationRequired:  c.NotificationRequired,
		NotificationDetails:   c.NotificationDetails,
		RequiredSubjects:      c.RequiredSubjects,
		AssessmentRequired:    c.AssessmentRequired,
		AssessmentDetails:     c.AssessmentDetails,
		RecordKeepingRequired: c.RecordKeepingRequired,
		RecordKeepingDetails:  c.RecordKeepingDetails,
		AttendanceRequired:    c.AttendanceRequired,
		AttendanceDays:        c.AttendanceDays,
		AttendanceHours:       c.AttendanceHours,
		AttendanceDetails:     c.AttendanceDetails,
		RegulationLevel:       c.RegulationLevel,
	}
}

func mapScheduleToResponse(s *ComplyCustomSchedule) *ScheduleResponse {
	var eps []ExclusionPeriod
	if s.ExclusionPeriods != nil {
		_ = json.Unmarshal(s.ExclusionPeriods, &eps)
	}
	return &ScheduleResponse{
		ID:               s.ID,
		Name:             s.Name,
		SchoolDays:       s.SchoolDays,
		ExclusionPeriods: eps,
		CreatedAt:        s.CreatedAt,
	}
}

func mapAttendanceToResponse(a *ComplyAttendance) *AttendanceResponse {
	return &AttendanceResponse{
		ID:              a.ID,
		StudentID:       a.StudentID,
		AttendanceDate:  a.AttendanceDate,
		Status:          a.Status,
		DurationMinutes: a.DurationMinutes,
		Notes:           a.Notes,
		IsAuto:          a.IsAuto,
		ManualOverride:  a.ManualOverride,
		CreatedAt:       a.CreatedAt,
	}
}

func mapAssessmentToResponse(a *ComplyAssessmentRecord) *AssessmentResponse {
	return &AssessmentResponse{
		ID:             a.ID,
		StudentID:      a.StudentID,
		Title:          a.Title,
		Subject:        a.Subject,
		AssessmentType: a.AssessmentType,
		Score:          a.Score,
		MaxScore:       a.MaxScore,
		GradeLetter:    a.GradeLetter,
		GradePoints:    a.GradePoints,
		IsPassing:      a.IsPassing,
		AssessmentDate: a.AssessmentDate,
		Notes:          a.Notes,
		CreatedAt:      a.CreatedAt,
	}
}

func mapTestScoreToResponse(t *ComplyStandardizedTest) *TestScoreResponse {
	return &TestScoreResponse{
		ID:             t.ID,
		StudentID:      t.StudentID,
		TestName:       t.TestName,
		TestDate:       t.TestDate,
		GradeLevel:     t.GradeLevel,
		Scores:         t.Scores,
		CompositeScore: t.CompositeScore,
		Percentile:     t.Percentile,
		Notes:          t.Notes,
		CreatedAt:      t.CreatedAt,
	}
}

func mapPortfolioToResponse(p *ComplyPortfolio, items []ComplyPortfolioItem) *PortfolioResponse {
	resp := &PortfolioResponse{
		ID:                 p.ID,
		StudentID:          p.StudentID,
		Title:              p.Title,
		Description:        p.Description,
		Organization:       p.Organization,
		DateRangeStart:     p.DateRangeStart,
		DateRangeEnd:       p.DateRangeEnd,
		IncludeAttendance:  p.IncludeAttendance,
		IncludeAssessments: p.IncludeAssessments,
		Status:             p.Status,
		ItemCount:          int32(len(items)),
		GeneratedAt:        p.GeneratedAt,
		ExpiresAt:          p.ExpiresAt,
		CreatedAt:          p.CreatedAt,
	}

	if items != nil {
		resp.Items = make([]PortfolioItemResponse, len(items))
		for i, item := range items {
			resp.Items[i] = mapPortfolioItemToResponse(&item)
		}
	}
	return resp
}

func mapPortfolioItemToResponse(item *ComplyPortfolioItem) PortfolioItemResponse {
	return PortfolioItemResponse{
		ID:                item.ID,
		SourceType:        item.SourceType,
		SourceID:          item.SourceID,
		DisplayOrder:      item.DisplayOrder,
		CachedTitle:       item.CachedTitle,
		CachedSubject:     item.CachedSubject,
		CachedDate:        item.CachedDate,
		CachedDescription: item.CachedDescription,
	}
}

func mapTranscriptToResponse(t *ComplyTranscript, courses []CourseResponse, gpa *GpaResponse) *TranscriptResponse {
	resp := &TranscriptResponse{
		ID:            t.ID,
		StudentID:     t.StudentID,
		Title:         t.Title,
		StudentName:   t.StudentName,
		GradeLevels:   t.GradeLevels,
		Status:        t.Status,
		GPAUnweighted: t.SnapshotGpaUnweighted,
		GPAWeighted:   t.SnapshotGpaWeighted,
		GeneratedAt:   t.GeneratedAt,
		ExpiresAt:     t.ExpiresAt,
		CreatedAt:     t.CreatedAt,
	}
	if courses != nil {
		resp.Courses = courses
	}
	if gpa != nil {
		resp.GPAUnweighted = &gpa.UnweightedGPA
		resp.GPAWeighted = &gpa.WeightedGPA
	}
	return resp
}

func mapTranscriptToSummary(t *ComplyTranscript) TranscriptSummaryResponse {
	return TranscriptSummaryResponse{
		ID:          t.ID,
		Title:       t.Title,
		Status:      t.Status,
		GradeLevels: t.GradeLevels,
		GeneratedAt: t.GeneratedAt,
		CreatedAt:   t.CreatedAt,
	}
}

func mapCourseToResponse(c *ComplyCourse) *CourseResponse {
	return &CourseResponse{
		ID:          c.ID,
		StudentID:   c.StudentID,
		Title:       c.Title,
		Subject:     c.Subject,
		GradeLevel:  c.GradeLevel,
		Credits:     c.Credits,
		GradeLetter: c.GradeLetter,
		GradePoints: c.GradePoints,
		Level:       c.Level,
		SchoolYear:  c.SchoolYear,
		Semester:    c.Semester,
		CreatedAt:   c.CreatedAt,
	}
}

func coursesToGpaInput(courses []ComplyCourse) []domain.CourseForGpa {
	result := make([]domain.CourseForGpa, len(courses))
	for i, c := range courses {
		result[i] = domain.CourseForGpa{
			GradePoints: c.GradePoints,
			Credits:     c.Credits,
			Level:       c.Level,
		}
	}
	return result
}

func (s *ComplianceServiceImpl) getGpaScaleForFamily(ctx context.Context, scope shared.FamilyScope) (domain.GpaScale, json.RawMessage, error) {
	config, err := s.familyConfigRepo.FindByFamily(ctx, scope)
	if err != nil {
		if errors.Is(err, ErrFamilyConfigNotFound) {
			return domain.GpaScaleStandard4, nil, nil // no config — use defaults
		}
		return domain.GpaScaleStandard4, nil, err
	}
	return domain.GpaScale(config.GpaScale), config.GpaCustomConfig, nil
}
