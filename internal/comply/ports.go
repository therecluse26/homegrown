package comply

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer-Defined Cross-Domain Interfaces [14-comply §17]
// ═══════════════════════════════════════════════════════════════════════════════

// IamServiceForComply is a consumer-defined interface for cross-domain reads from iam::.
// Implemented by a function adapter in main.go over iam.Service.
type IamServiceForComply interface {
	StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error)
	GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error)
}

// LearningServiceForComply is a consumer-defined interface for cross-domain reads from learn::.
// Used by portfolio item selection to fetch cached display data.
type LearningServiceForComply interface {
	GetPortfolioItemData(ctx context.Context, sourceType string, sourceID uuid.UUID) (*PortfolioItemData, error)
}

// DiscoveryServiceForComply is a consumer-defined interface for cross-domain reads from discover::.
// Used by SyncStateConfigsJob and state requirement lookups.
type DiscoveryServiceForComply interface {
	GetStateRequirements(ctx context.Context, stateCode string) (*StateRequirementsData, error)
	ListStateGuides(ctx context.Context) ([]StateGuideSummary, error)
}

// MediaServiceForComply is a consumer-defined interface for cross-domain calls to media::.
// Used by portfolio/transcript PDF generation for upload and presigned download.
type MediaServiceForComply interface {
	RequestUpload(ctx context.Context, familyID uuid.UUID, uploadContext string, filename string, contentType string, data []byte) (*uuid.UUID, error)
	PresignedGet(ctx context.Context, uploadID uuid.UUID) (string, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [14-comply §5]
// ═══════════════════════════════════════════════════════════════════════════════

// ComplianceService defines all compliance use cases.
// CQRS separation: command methods (writes) vs query methods (reads).
type ComplianceService interface {

	// ─── Command side (writes, side effects) ────────────────────────────

	// UpsertFamilyConfig creates or updates family compliance configuration.
	UpsertFamilyConfig(ctx context.Context, cmd UpsertFamilyConfigCommand, scope shared.FamilyScope) (*FamilyConfigResponse, error)

	// CreateSchedule creates a custom schedule.
	CreateSchedule(ctx context.Context, cmd CreateScheduleCommand, scope shared.FamilyScope) (*ScheduleResponse, error)

	// UpdateSchedule updates a custom schedule.
	UpdateSchedule(ctx context.Context, scheduleID uuid.UUID, cmd UpdateScheduleCommand, scope shared.FamilyScope) (*ScheduleResponse, error)

	// DeleteSchedule deletes a custom schedule.
	DeleteSchedule(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) error

	// RecordAttendance records daily attendance for a student (manual entry).
	RecordAttendance(ctx context.Context, studentID uuid.UUID, cmd RecordAttendanceCommand, scope shared.FamilyScope) (*AttendanceResponse, error)

	// BulkRecordAttendance bulk records attendance for a student.
	BulkRecordAttendance(ctx context.Context, studentID uuid.UUID, cmd BulkRecordAttendanceCommand, scope shared.FamilyScope) ([]AttendanceResponse, error)

	// UpdateAttendance updates an attendance record.
	UpdateAttendance(ctx context.Context, studentID uuid.UUID, attendanceID uuid.UUID, cmd UpdateAttendanceCommand, scope shared.FamilyScope) (*AttendanceResponse, error)

	// DeleteAttendance deletes an attendance record.
	DeleteAttendance(ctx context.Context, studentID uuid.UUID, attendanceID uuid.UUID, scope shared.FamilyScope) error

	// CreateAssessment creates an assessment record.
	CreateAssessment(ctx context.Context, studentID uuid.UUID, cmd CreateAssessmentCommand, scope shared.FamilyScope) (*AssessmentResponse, error)

	// UpdateAssessment updates an assessment record.
	UpdateAssessment(ctx context.Context, studentID uuid.UUID, assessmentID uuid.UUID, cmd UpdateAssessmentCommand, scope shared.FamilyScope) (*AssessmentResponse, error)

	// DeleteAssessment deletes an assessment record.
	DeleteAssessment(ctx context.Context, studentID uuid.UUID, assessmentID uuid.UUID, scope shared.FamilyScope) error

	// CreateTestScore records a standardized test score.
	CreateTestScore(ctx context.Context, studentID uuid.UUID, cmd CreateTestScoreCommand, scope shared.FamilyScope) (*TestScoreResponse, error)

	// UpdateTestScore updates a test score.
	UpdateTestScore(ctx context.Context, studentID uuid.UUID, testID uuid.UUID, cmd UpdateTestScoreCommand, scope shared.FamilyScope) (*TestScoreResponse, error)

	// DeleteTestScore deletes a test score.
	DeleteTestScore(ctx context.Context, studentID uuid.UUID, testID uuid.UUID, scope shared.FamilyScope) error

	// CreatePortfolio creates a portfolio.
	CreatePortfolio(ctx context.Context, studentID uuid.UUID, cmd CreatePortfolioCommand, scope shared.FamilyScope) (*PortfolioResponse, error)

	// AddPortfolioItems adds items to a portfolio (caches display data from learn::).
	AddPortfolioItems(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, cmd AddPortfolioItemsCommand, scope shared.FamilyScope) ([]PortfolioItemResponse, error)

	// GeneratePortfolio triggers portfolio PDF generation (async — enqueues job).
	GeneratePortfolio(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, scope shared.FamilyScope) (*PortfolioResponse, error)

	// CreateTranscript creates a transcript (Phase 3).
	CreateTranscript(ctx context.Context, studentID uuid.UUID, cmd CreateTranscriptCommand, scope shared.FamilyScope) (*TranscriptResponse, error)

	// GenerateTranscript triggers transcript PDF generation (Phase 3).
	GenerateTranscript(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) (*TranscriptResponse, error)

	// DeleteTranscript deletes a transcript (Phase 3).
	DeleteTranscript(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) error

	// CreateCourse creates a course for transcript (Phase 3).
	CreateCourse(ctx context.Context, studentID uuid.UUID, cmd CreateCourseCommand, scope shared.FamilyScope) (*CourseResponse, error)

	// UpdateCourse updates a course (Phase 3).
	UpdateCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID, cmd UpdateCourseCommand, scope shared.FamilyScope) (*CourseResponse, error)

	// DeleteCourse deletes a course (Phase 3).
	DeleteCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID, scope shared.FamilyScope) error

	// ─── Query side (reads, no side effects) ────────────────────────────

	// GetFamilyConfig gets family compliance configuration.
	GetFamilyConfig(ctx context.Context, scope shared.FamilyScope) (*FamilyConfigResponse, error)

	// ListStateConfigs lists all state requirements (from cache).
	ListStateConfigs(ctx context.Context) ([]StateConfigSummaryResponse, error)

	// GetStateConfig gets requirements for a specific state.
	GetStateConfig(ctx context.Context, stateCode string) (*StateConfigResponse, error)

	// ListSchedules lists family's custom schedules.
	ListSchedules(ctx context.Context, scope shared.FamilyScope) ([]ScheduleResponse, error)

	// ListAttendance lists attendance records for a student.
	ListAttendance(ctx context.Context, studentID uuid.UUID, params AttendanceListParams, scope shared.FamilyScope) (*AttendanceListResponse, error)

	// GetAttendanceSummary gets attendance summary for a student.
	GetAttendanceSummary(ctx context.Context, studentID uuid.UUID, params AttendanceSummaryParams, scope shared.FamilyScope) (*AttendanceSummaryResponse, error)

	// ListAssessments lists assessment records for a student.
	ListAssessments(ctx context.Context, studentID uuid.UUID, params AssessmentListParams, scope shared.FamilyScope) (*AssessmentListResponse, error)

	// ListTestScores lists standardized test scores for a student.
	ListTestScores(ctx context.Context, studentID uuid.UUID, params TestListParams, scope shared.FamilyScope) (*TestListResponse, error)

	// GetPortfolio gets portfolio details (includes items).
	GetPortfolio(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, scope shared.FamilyScope) (*PortfolioResponse, error)

	// ListPortfolios lists portfolios for a student.
	ListPortfolios(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]PortfolioSummaryResponse, error)

	// GetPortfolioDownloadURL gets presigned download URL for a portfolio PDF.
	GetPortfolioDownloadURL(ctx context.Context, studentID uuid.UUID, portfolioID uuid.UUID, scope shared.FamilyScope) (string, error)

	// GetDashboard gets compliance dashboard overview.
	GetDashboard(ctx context.Context, scope shared.FamilyScope) (*ComplianceDashboardResponse, error)

	// GetTranscript gets transcript details (Phase 3).
	GetTranscript(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) (*TranscriptResponse, error)

	// ListTranscripts lists transcripts for a student (Phase 3).
	ListTranscripts(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]TranscriptSummaryResponse, error)

	// GetTranscriptDownloadURL gets presigned download URL for a transcript PDF (Phase 3).
	GetTranscriptDownloadURL(ctx context.Context, studentID uuid.UUID, transcriptID uuid.UUID, scope shared.FamilyScope) (string, error)

	// ListCourses lists courses for a student (Phase 3).
	ListCourses(ctx context.Context, studentID uuid.UUID, params CourseListParams, scope shared.FamilyScope) (*CourseListResponse, error)

	// CalculateGPA calculates current GPA for a student (Phase 3).
	CalculateGPA(ctx context.Context, studentID uuid.UUID, params GpaParams, scope shared.FamilyScope) (*GpaResponse, error)

	// CalculateGPAWhatIf calculates what-if GPA with hypothetical courses (Phase 3).
	CalculateGPAWhatIf(ctx context.Context, studentID uuid.UUID, params GpaWhatIfParams, scope shared.FamilyScope) (*GpaResponse, error)

	// GetGPAHistory returns GPA history by term (Phase 3).
	GetGPAHistory(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]GpaTermResponse, error)

	// ─── Event handlers ─────────────────────────────────────────────────

	// HandleActivityLogged handles ActivityLogged event: auto-record attendance (Phase 1).
	HandleActivityLogged(ctx context.Context, event *ActivityLoggedEvent) error

	// HandleStudentDeleted handles StudentDeleted event: cascade delete compliance data.
	HandleStudentDeleted(ctx context.Context, event *StudentDeletedEvent) error

	// HandleFamilyDeletionScheduled handles FamilyDeletionScheduled: cascade delete all comply:: data.
	HandleFamilyDeletionScheduled(ctx context.Context, event *FamilyDeletionScheduledEvent) error

	// HandleSubscriptionCancelled handles SubscriptionCancelled: preserve data, no deletion.
	HandleSubscriptionCancelled(ctx context.Context, event *SubscriptionCancelledEvent) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [14-comply §6]
// ═══════════════════════════════════════════════════════════════════════════════

// StateConfigRepository is the repository for comply_state_configs.
// NOT family-scoped — platform-authored reference data (51 rows).
type StateConfigRepository interface {
	ListAll(ctx context.Context) ([]ComplyStateConfig, error)
	FindByStateCode(ctx context.Context, stateCode string) (*ComplyStateConfig, error)
	Upsert(ctx context.Context, config UpsertStateConfigRow) (*ComplyStateConfig, error)
}

// FamilyConfigRepository is the repository for comply_family_configs.
// Family-scoped (family_id is PK).
type FamilyConfigRepository interface {
	Upsert(ctx context.Context, scope shared.FamilyScope, input UpsertFamilyConfigRow) (*ComplyFamilyConfig, error)
	FindByFamily(ctx context.Context, scope shared.FamilyScope) (*ComplyFamilyConfig, error)
	DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// ScheduleRepository is the repository for comply_custom_schedules.
// Family-scoped.
type ScheduleRepository interface {
	Create(ctx context.Context, scope shared.FamilyScope, input CreateScheduleRow) (*ComplyCustomSchedule, error)
	FindByID(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) (*ComplyCustomSchedule, error)
	ListByFamily(ctx context.Context, scope shared.FamilyScope) ([]ComplyCustomSchedule, error)
	Update(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope, updates UpdateScheduleRow) (*ComplyCustomSchedule, error)
	Delete(ctx context.Context, scheduleID uuid.UUID, scope shared.FamilyScope) error
}

// AttendanceRepository is the repository for comply_attendance.
// Family-scoped. UNIQUE on (family_id, student_id, attendance_date).
type AttendanceRepository interface {
	Upsert(ctx context.Context, scope shared.FamilyScope, input UpsertAttendanceRow) (*ComplyAttendance, error)
	FindByID(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) (*ComplyAttendance, error)
	ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AttendanceListParams) ([]ComplyAttendance, error)
	Summarize(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, startDate time.Time, endDate time.Time) (*AttendanceSummaryRow, error)
	Update(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope, updates UpdateAttendanceRow) (*ComplyAttendance, error)
	Delete(ctx context.Context, attendanceID uuid.UUID, scope shared.FamilyScope) error
	DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
	DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// AssessmentRepository is the repository for comply_assessment_records.
// Family-scoped.
type AssessmentRepository interface {
	Create(ctx context.Context, scope shared.FamilyScope, input CreateAssessmentRow) (*ComplyAssessmentRecord, error)
	FindByID(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) (*ComplyAssessmentRecord, error)
	ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *AssessmentListParams) ([]ComplyAssessmentRecord, error)
	Update(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope, updates UpdateAssessmentRow) (*ComplyAssessmentRecord, error)
	Delete(ctx context.Context, assessmentID uuid.UUID, scope shared.FamilyScope) error
	DeleteByStudent(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) error
}

// TestScoreRepository is the repository for comply_standardized_tests.
// Family-scoped.
type TestScoreRepository interface {
	Create(ctx context.Context, scope shared.FamilyScope, input CreateTestScoreRow) (*ComplyStandardizedTest, error)
	ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *TestListParams) ([]ComplyStandardizedTest, error)
	Update(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope, updates UpdateTestScoreRow) (*ComplyStandardizedTest, error)
	Delete(ctx context.Context, testID uuid.UUID, scope shared.FamilyScope) error
}

// PortfolioRepository is the repository for comply_portfolios.
// Family-scoped.
type PortfolioRepository interface {
	Create(ctx context.Context, scope shared.FamilyScope, input CreatePortfolioRow) (*ComplyPortfolio, error)
	FindByID(ctx context.Context, portfolioID uuid.UUID, scope shared.FamilyScope) (*ComplyPortfolio, error)
	ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyPortfolio, error)
	UpdateStatus(ctx context.Context, portfolioID uuid.UUID, status string, uploadID *uuid.UUID, errorMessage *string) (*ComplyPortfolio, error)
	FindExpired(ctx context.Context, before time.Time) ([]ComplyPortfolio, error)
}

// PortfolioItemRepository is the repository for comply_portfolio_items.
type PortfolioItemRepository interface {
	CreateBatch(ctx context.Context, items []CreatePortfolioItemRow) ([]ComplyPortfolioItem, error)
	ListByPortfolio(ctx context.Context, portfolioID uuid.UUID) ([]ComplyPortfolioItem, error)
	DeleteByPortfolio(ctx context.Context, portfolioID uuid.UUID) error
}

// TranscriptRepository is the repository for comply_transcripts (Phase 3).
type TranscriptRepository interface {
	Create(ctx context.Context, scope shared.FamilyScope, input CreateTranscriptRow) (*ComplyTranscript, error)
	FindByID(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) (*ComplyTranscript, error)
	ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope) ([]ComplyTranscript, error)
	UpdateStatus(ctx context.Context, transcriptID uuid.UUID, status string, uploadID *uuid.UUID, gpaUnweighted *float64, gpaWeighted *float64, errorMessage *string) (*ComplyTranscript, error)
	Delete(ctx context.Context, transcriptID uuid.UUID, scope shared.FamilyScope) error
}

// CourseRepository is the repository for comply_courses (Phase 3).
type CourseRepository interface {
	Create(ctx context.Context, scope shared.FamilyScope, input CreateCourseRow) (*ComplyCourse, error)
	ListByStudent(ctx context.Context, studentID uuid.UUID, scope shared.FamilyScope, params *CourseListParams) ([]ComplyCourse, error)
	Update(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope, updates UpdateCourseRow) (*ComplyCourse, error)
	Delete(ctx context.Context, courseID uuid.UUID, scope shared.FamilyScope) error
}

