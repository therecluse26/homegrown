package comply

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Typed String Enums [14-comply §3.1]
// ═══════════════════════════════════════════════════════════════════════════════

// AttendanceStatus represents the status of an attendance record.
type AttendanceStatus string

const (
	AttendanceStatusPresentFull    AttendanceStatus = "present_full"
	AttendanceStatusPresentPartial AttendanceStatus = "present_partial"
	AttendanceStatusAbsent         AttendanceStatus = "absent"
	AttendanceStatusNotApplicable  AttendanceStatus = "not_applicable"
)

// PortfolioStatus represents the lifecycle status of a portfolio.
type PortfolioStatus string

const (
	PortfolioStatusConfiguring PortfolioStatus = "configuring"
	PortfolioStatusGenerating  PortfolioStatus = "generating"
	PortfolioStatusReady       PortfolioStatus = "ready"
	PortfolioStatusFailed      PortfolioStatus = "failed"
	PortfolioStatusExpired     PortfolioStatus = "expired"
)

// PortfolioOrganization represents how portfolio items are organized.
type PortfolioOrganization string

const (
	PortfolioOrganizationBySubject     PortfolioOrganization = "by_subject"
	PortfolioOrganizationChronological PortfolioOrganization = "chronological"
	PortfolioOrganizationByStudent     PortfolioOrganization = "by_student"
)

// AssessmentType represents the type of an assessment.
type AssessmentType string

const (
	AssessmentTypeTest           AssessmentType = "test"
	AssessmentTypeQuiz           AssessmentType = "quiz"
	AssessmentTypeProject        AssessmentType = "project"
	AssessmentTypeAssignment     AssessmentType = "assignment"
	AssessmentTypePresentation   AssessmentType = "presentation"
	AssessmentTypePortfolioPiece AssessmentType = "portfolio_piece"
	AssessmentTypeOther          AssessmentType = "other"
)

// CourseLevel represents the level of a course (Phase 3).
type CourseLevel string

const (
	CourseLevelRegular CourseLevel = "regular"
	CourseLevelHonors  CourseLevel = "honors"
	CourseLevelAP      CourseLevel = "ap"
)

// PortfolioItemSourceType represents the source of a portfolio item.
type PortfolioItemSourceType string

const (
	PortfolioItemSourceActivity    PortfolioItemSourceType = "activity"
	PortfolioItemSourceJournal     PortfolioItemSourceType = "journal"
	PortfolioItemSourceProject     PortfolioItemSourceType = "project"
	PortfolioItemSourceReadingList PortfolioItemSourceType = "reading_list"
	PortfolioItemSourceAssessment  PortfolioItemSourceType = "assessment"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Request Types [14-comply §8.1]
// ═══════════════════════════════════════════════════════════════════════════════

// UpsertFamilyConfigCommand is the body for PUT /v1/compliance/config.
// Fields marked omitempty are optional for the simplified setup flow (state + days/hours).
// The service layer sets defaults for missing fields. [M16]
type UpsertFamilyConfigCommand struct {
	StateCode        string          `json:"state_code" validate:"required,len=2"`
	SchoolYearStart  *time.Time      `json:"school_year_start,omitempty"`
	SchoolYearEnd    *time.Time      `json:"school_year_end,omitempty"`
	TotalSchoolDays  *int16          `json:"total_school_days,omitempty"`
	DaysRequired     *int16          `json:"days_required,omitempty"`
	HoursRequired    *int16          `json:"hours_required,omitempty"`
	CustomScheduleID *uuid.UUID      `json:"custom_schedule_id"`
	GpaScale         *string         `json:"gpa_scale,omitempty" validate:"omitempty,oneof=standard_4 weighted custom"`
	GpaCustomConfig  json.RawMessage `json:"gpa_custom_config,omitempty" swaggertype:"object"`
}

// CreateScheduleCommand is the body for POST /v1/compliance/schedules.
type CreateScheduleCommand struct {
	Name             string            `json:"name" validate:"required"`
	SchoolDays       []bool            `json:"school_days" validate:"required,len=7"`
	ExclusionPeriods []ExclusionPeriod `json:"exclusion_periods"`
}

// ExclusionPeriod represents a date range excluded from school days (with JSON tags).
type ExclusionPeriod struct {
	Start time.Time `json:"start" validate:"required"`
	End   time.Time `json:"end" validate:"required"`
	Label string    `json:"label" validate:"required"`
}

// UpdateScheduleCommand is the body for PATCH /v1/compliance/schedules/:id.
type UpdateScheduleCommand struct {
	Name             *string            `json:"name"`
	SchoolDays       *[]bool            `json:"school_days"`
	ExclusionPeriods *[]ExclusionPeriod `json:"exclusion_periods"`
}

// RecordAttendanceCommand is the body for POST /v1/compliance/students/:id/attendance.
type RecordAttendanceCommand struct {
	AttendanceDate  time.Time `json:"attendance_date" validate:"required"`
	Status          string    `json:"status" validate:"required,oneof=present_full present_partial absent not_applicable"`
	DurationMinutes *int16    `json:"duration_minutes"`
	Notes           *string   `json:"notes"`
}

// BulkRecordAttendanceCommand is the body for POST /v1/compliance/students/:id/attendance/bulk.
type BulkRecordAttendanceCommand struct {
	Records []RecordAttendanceCommand `json:"records" validate:"required,max=31,dive"`
}

// UpdateAttendanceCommand is the body for PATCH /v1/compliance/students/:id/attendance/:id.
type UpdateAttendanceCommand struct {
	Status          *string `json:"status"`
	DurationMinutes *int16  `json:"duration_minutes"`
	Notes           *string `json:"notes"`
}

// CreateAssessmentCommand is the body for POST /v1/compliance/students/:id/assessments.
type CreateAssessmentCommand struct {
	Title            string     `json:"title" validate:"required"`
	Subject          string     `json:"subject" validate:"required"`
	AssessmentType   string     `json:"assessment_type" validate:"required,oneof=test quiz project assignment presentation portfolio_piece other"`
	Score            *float64   `json:"score"`
	MaxScore         *float64   `json:"max_score"`
	GradeLetter      *string    `json:"grade_letter"`
	GradePoints      *float64   `json:"grade_points"`
	IsPassing        *bool      `json:"is_passing"`
	SourceActivityID *uuid.UUID `json:"source_activity_id"`
	AssessmentDate   time.Time  `json:"assessment_date" validate:"required"`
	Notes            *string    `json:"notes"`
}

// UpdateAssessmentCommand is the body for PATCH /v1/compliance/students/:id/assessments/:id.
type UpdateAssessmentCommand struct {
	Title          *string    `json:"title"`
	Subject        *string    `json:"subject"`
	Score          *float64   `json:"score"`
	MaxScore       *float64   `json:"max_score"`
	GradeLetter    *string    `json:"grade_letter"`
	GradePoints    *float64   `json:"grade_points"`
	IsPassing      *bool      `json:"is_passing"`
	AssessmentDate *time.Time `json:"assessment_date"`
	Notes          *string    `json:"notes"`
}

// CreateTestScoreCommand is the body for POST /v1/compliance/students/:id/tests.
type CreateTestScoreCommand struct {
	TestName       string          `json:"test_name" validate:"required"`
	TestDate       time.Time       `json:"test_date" validate:"required"`
	GradeLevel     *int16          `json:"grade_level"`
	Scores         json.RawMessage `json:"scores" validate:"required" swaggertype:"object"`
	CompositeScore *float64        `json:"composite_score"`
	Percentile     *int16          `json:"percentile"`
	Notes          *string         `json:"notes"`
}

// UpdateTestScoreCommand is the body for PATCH /v1/compliance/students/:id/tests/:id.
type UpdateTestScoreCommand struct {
	TestName       *string          `json:"test_name"`
	TestDate       *time.Time       `json:"test_date"`
	Scores         *json.RawMessage `json:"scores" swaggertype:"object"`
	CompositeScore *float64         `json:"composite_score"`
	Percentile     *int16           `json:"percentile"`
	Notes          *string          `json:"notes"`
}

// CreatePortfolioCommand is the body for POST /v1/compliance/students/:id/portfolios.
type CreatePortfolioCommand struct {
	Title              string    `json:"title" validate:"required"`
	Description        *string   `json:"description"`
	Organization       string    `json:"organization" validate:"required,oneof=by_subject chronological by_student"`
	DateRangeStart     time.Time `json:"date_range_start" validate:"required"`
	DateRangeEnd       time.Time `json:"date_range_end" validate:"required"`
	IncludeAttendance  bool      `json:"include_attendance"`
	IncludeAssessments bool      `json:"include_assessments"`
}

// AddPortfolioItemsCommand is the body for POST /v1/compliance/students/:id/portfolios/:id/items.
type AddPortfolioItemsCommand struct {
	Items []PortfolioItemInput `json:"items" validate:"required,dive"`
}

// PortfolioItemInput represents a single item to add to a portfolio.
type PortfolioItemInput struct {
	SourceType string    `json:"source_type" validate:"required,oneof=activity journal project reading_list assessment"`
	SourceID   uuid.UUID `json:"source_id" validate:"required"`
}

// CreateTranscriptCommand is the body for POST /v1/compliance/students/:id/transcripts (Phase 3).
type CreateTranscriptCommand struct {
	Title       string   `json:"title" validate:"required"`
	GradeLevels []string `json:"grade_levels"`
}

// CreateCourseCommand is the body for POST /v1/compliance/students/:id/courses (Phase 3).
type CreateCourseCommand struct {
	Title       string   `json:"title" validate:"required"`
	Subject     string   `json:"subject" validate:"required"`
	GradeLevel  int16    `json:"grade_level" validate:"required"`
	Credits     float64  `json:"credits" validate:"required,gt=0"`
	GradeLetter *string  `json:"grade_letter"`
	GradePoints *float64 `json:"grade_points"`
	Level       string   `json:"level" validate:"required,oneof=regular honors ap"`
	SchoolYear  string   `json:"school_year" validate:"required"`
	Semester    *string  `json:"semester" validate:"omitempty,oneof=fall spring summer full_year"`
}

// UpdateCourseCommand is the body for PATCH /v1/compliance/students/:id/courses/:id (Phase 3).
type UpdateCourseCommand struct {
	Title       *string  `json:"title"`
	Subject     *string  `json:"subject"`
	Credits     *float64 `json:"credits"`
	GradeLetter *string  `json:"grade_letter"`
	GradePoints *float64 `json:"grade_points"`
	Level       *string  `json:"level"`
	Semester    *string  `json:"semester"`
}

// ─── List / Query Params ──────────────────────────────────────────────────────

// AttendanceListParams holds query params for GET /v1/compliance/students/:id/attendance.
type AttendanceListParams struct {
	StartDate time.Time `query:"start_date" validate:"required"`
	EndDate   time.Time `query:"end_date" validate:"required"`
	Status    *string   `query:"status"`
	Cursor    *string   `query:"cursor"`
	Limit     *uint8    `query:"limit"`
}

// AttendanceSummaryParams holds query params for attendance summary.
type AttendanceSummaryParams struct {
	StartDate time.Time `query:"start_date" validate:"required"`
	EndDate   time.Time `query:"end_date" validate:"required"`
}

// AssessmentListParams holds query params for GET /v1/compliance/students/:id/assessments.
type AssessmentListParams struct {
	Subject   *string    `query:"subject"`
	StartDate *time.Time `query:"start_date"`
	EndDate   *time.Time `query:"end_date"`
	Cursor    *string    `query:"cursor"`
	Limit     *uint8     `query:"limit"`
}

// TestListParams holds query params for GET /v1/compliance/students/:id/tests.
type TestListParams struct {
	Cursor *string `query:"cursor"`
	Limit  *uint8  `query:"limit"`
}

// CourseListParams holds query params for GET /v1/compliance/students/:id/courses (Phase 3).
type CourseListParams struct {
	GradeLevel *int16  `query:"grade_level"`
	SchoolYear *string `query:"school_year"`
	Cursor     *string `query:"cursor"`
	Limit      *uint8  `query:"limit"`
}

// GpaParams holds query params for GET /v1/compliance/students/:id/gpa (Phase 3).
type GpaParams struct {
	Scale       *string `query:"scale"`
	GradeLevels []int16 `query:"grade_levels"`
}

// GpaWhatIfParams holds query params for GPA what-if calculation (Phase 3).
type GpaWhatIfParams struct {
	AdditionalCourses []WhatIfCourse `json:"additional_courses" validate:"required,dive"`
}

// WhatIfCourse represents a hypothetical course for GPA what-if calculation.
type WhatIfCourse struct {
	Credits     float64 `json:"credits" validate:"required,gt=0"`
	GradePoints float64 `json:"grade_points" validate:"required,gte=0"`
	Level       string  `json:"level" validate:"required,oneof=regular honors ap"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Types [14-comply §8.2]
// ═══════════════════════════════════════════════════════════════════════════════

// FamilyConfigResponse represents family compliance configuration.
// DaysRequired and HoursRequired mirror TotalSchoolDays for the simplified frontend flow.
type FamilyConfigResponse struct {
	FamilyID         uuid.UUID  `json:"family_id"`
	StateCode        string     `json:"state_code"`
	StateName        string     `json:"state_name"`
	SchoolYearStart  time.Time  `json:"school_year_start"`
	SchoolYearEnd    time.Time  `json:"school_year_end"`
	TotalSchoolDays  int16      `json:"total_school_days"`
	DaysRequired     int16      `json:"days_required"`
	HoursRequired    int16      `json:"hours_required"`
	CustomScheduleID *uuid.UUID `json:"custom_schedule_id"`
	GpaScale         string     `json:"gpa_scale"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// StateConfigResponse represents state compliance requirements (from cache).
type StateConfigResponse struct {
	StateCode             string   `json:"state_code"`
	StateName             string   `json:"state_name"`
	NotificationRequired  bool     `json:"notification_required"`
	NotificationDetails   *string  `json:"notification_details"`
	RequiredSubjects      []string `json:"required_subjects"`
	AssessmentRequired    bool     `json:"assessment_required"`
	AssessmentDetails     *string  `json:"assessment_details"`
	RecordKeepingRequired bool     `json:"record_keeping_required"`
	RecordKeepingDetails  *string  `json:"record_keeping_details"`
	AttendanceRequired    bool     `json:"attendance_required"`
	AttendanceDays        *int16   `json:"attendance_days"`
	AttendanceHours       *int16   `json:"attendance_hours"`
	AttendanceDetails     *string  `json:"attendance_details"`
	RegulationLevel       string   `json:"regulation_level"`
}

// StateConfigSummaryResponse represents state config summary (for listing).
type StateConfigSummaryResponse struct {
	StateCode          string `json:"state_code"`
	StateName          string `json:"state_name"`
	RegulationLevel    string `json:"regulation_level"`
	AttendanceRequired bool   `json:"attendance_required"`
	AttendanceDays     *int16 `json:"attendance_days"`
}

// ScheduleResponse represents a custom schedule.
type ScheduleResponse struct {
	ID               uuid.UUID         `json:"id"`
	Name             string            `json:"name"`
	SchoolDays       []bool            `json:"school_days"`
	ExclusionPeriods []ExclusionPeriod `json:"exclusion_periods"`
	CreatedAt        time.Time         `json:"created_at"`
}

// AttendanceResponse represents a single attendance record.
type AttendanceResponse struct {
	ID              uuid.UUID `json:"id"`
	StudentID       uuid.UUID `json:"student_id"`
	AttendanceDate  time.Time `json:"attendance_date"`
	Status          string    `json:"status"`
	DurationMinutes *int16    `json:"duration_minutes"`
	Notes           *string   `json:"notes"`
	IsAuto          bool      `json:"is_auto"`
	ManualOverride  bool      `json:"manual_override"`
	CreatedAt       time.Time `json:"created_at"`
}

// AttendanceListResponse represents a paginated attendance list.
type AttendanceListResponse struct {
	Records    []AttendanceResponse `json:"records"`
	NextCursor *string              `json:"next_cursor"`
}

// AttendanceSummaryResponse represents attendance summary with pace calculation.
type AttendanceSummaryResponse struct {
	TotalDays          int32   `json:"total_days"`
	PresentFull        int32   `json:"present_full"`
	PresentPartial     int32   `json:"present_partial"`
	Absent             int32   `json:"absent"`
	NotApplicable      int32   `json:"not_applicable"`
	TotalHours         float64 `json:"total_hours"`
	StateRequiredDays  *int16  `json:"state_required_days"`
	StateRequiredHours *int16  `json:"state_required_hours"`
	PaceStatus         *string `json:"pace_status"`
	ProjectedTotalDays *int32  `json:"projected_total_days"`
}

// AssessmentResponse represents a single assessment record.
type AssessmentResponse struct {
	ID             uuid.UUID `json:"id"`
	StudentID      uuid.UUID `json:"student_id"`
	Title          string    `json:"title"`
	Subject        string    `json:"subject"`
	AssessmentType string    `json:"assessment_type"`
	Score          *float64  `json:"score"`
	MaxScore       *float64  `json:"max_score"`
	GradeLetter    *string   `json:"grade_letter"`
	GradePoints    *float64  `json:"grade_points"`
	IsPassing      *bool     `json:"is_passing"`
	AssessmentDate time.Time `json:"assessment_date"`
	Notes          *string   `json:"notes"`
	CreatedAt      time.Time `json:"created_at"`
}

// AssessmentListResponse represents a paginated assessment list.
type AssessmentListResponse struct {
	Records    []AssessmentResponse `json:"records"`
	NextCursor *string              `json:"next_cursor"`
}

// TestScoreResponse represents a single test score.
type TestScoreResponse struct {
	ID             uuid.UUID       `json:"id"`
	StudentID      uuid.UUID       `json:"student_id"`
	TestName       string          `json:"test_name"`
	TestDate       time.Time       `json:"test_date"`
	GradeLevel     *int16          `json:"grade_level"`
	Scores         json.RawMessage `json:"scores" swaggertype:"object"`
	CompositeScore *float64        `json:"composite_score"`
	Percentile     *int16          `json:"percentile"`
	Notes          *string         `json:"notes"`
	CreatedAt      time.Time       `json:"created_at"`
}

// TestListResponse represents a paginated test list.
type TestListResponse struct {
	Tests      []TestScoreResponse `json:"tests"`
	NextCursor *string             `json:"next_cursor"`
}

// PortfolioResponse represents portfolio details.
type PortfolioResponse struct {
	ID                 uuid.UUID               `json:"id"`
	StudentID          uuid.UUID               `json:"student_id"`
	Title              string                  `json:"title"`
	Description        *string                 `json:"description"`
	Organization       string                  `json:"organization"`
	DateRangeStart     time.Time               `json:"date_range_start"`
	DateRangeEnd       time.Time               `json:"date_range_end"`
	IncludeAttendance  bool                    `json:"include_attendance"`
	IncludeAssessments bool                    `json:"include_assessments"`
	Status             string                  `json:"status"`
	ItemCount          int32                   `json:"item_count"`
	GeneratedAt        *time.Time              `json:"generated_at"`
	ExpiresAt          *time.Time              `json:"expires_at"`
	Items              []PortfolioItemResponse `json:"items,omitempty"`
	CreatedAt          time.Time               `json:"created_at"`
}

// PortfolioSummaryResponse represents portfolio summary (for listing).
type PortfolioSummaryResponse struct {
	ID             uuid.UUID  `json:"id"`
	Title          string     `json:"title"`
	Status         string     `json:"status"`
	ItemCount      int32      `json:"item_count"`
	DateRangeStart time.Time  `json:"date_range_start"`
	DateRangeEnd   time.Time  `json:"date_range_end"`
	GeneratedAt    *time.Time `json:"generated_at"`
	ExpiresAt      *time.Time `json:"expires_at"`
	CreatedAt      time.Time  `json:"created_at"`
}

// PortfolioItemResponse represents a portfolio item (cached display data).
type PortfolioItemResponse struct {
	ID                uuid.UUID `json:"id"`
	SourceType        string    `json:"source_type"`
	SourceID          uuid.UUID `json:"source_id"`
	DisplayOrder      int16     `json:"display_order"`
	CachedTitle       string    `json:"cached_title"`
	CachedSubject     *string   `json:"cached_subject"`
	CachedDate        time.Time `json:"cached_date"`
	CachedDescription *string   `json:"cached_description"`
}

// GpaResponse represents a GPA calculation result (Phase 3).
type GpaResponse struct {
	UnweightedGPA float64                 `json:"unweighted_gpa"`
	WeightedGPA   float64                 `json:"weighted_gpa"`
	TotalCredits  float64                 `json:"total_credits"`
	TotalCourses  int32                   `json:"total_courses"`
	ByGradeLevel  []GpaGradeLevelResponse `json:"by_grade_level"`
}

// GpaGradeLevelResponse represents GPA breakdown by grade level.
type GpaGradeLevelResponse struct {
	GradeLevel int16   `json:"grade_level"`
	Unweighted float64 `json:"unweighted"`
	Weighted   float64 `json:"weighted"`
	Credits    float64 `json:"credits"`
}

// GpaTermResponse represents GPA history by term (Phase 3).
type GpaTermResponse struct {
	SchoolYear    string  `json:"school_year"`
	Semester      *string `json:"semester"`
	UnweightedGPA float64 `json:"unweighted_gpa"`
	WeightedGPA   float64 `json:"weighted_gpa"`
	Credits       float64 `json:"credits"`
	CourseCount   int32   `json:"course_count"`
}

// TranscriptResponse represents transcript details (Phase 3).
type TranscriptResponse struct {
	ID            uuid.UUID        `json:"id"`
	StudentID     uuid.UUID        `json:"student_id"`
	Title         string           `json:"title"`
	StudentName   string           `json:"student_name"`
	GradeLevels   []string         `json:"grade_levels"`
	Status        string           `json:"status"`
	GPAUnweighted *float64         `json:"gpa_unweighted"`
	GPAWeighted   *float64         `json:"gpa_weighted"`
	Courses       []CourseResponse `json:"courses,omitempty"`
	GeneratedAt   *time.Time       `json:"generated_at"`
	ExpiresAt     *time.Time       `json:"expires_at"`
	CreatedAt     time.Time        `json:"created_at"`
}

// TranscriptSummaryResponse represents transcript summary (Phase 3).
type TranscriptSummaryResponse struct {
	ID          uuid.UUID  `json:"id"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	GradeLevels []string   `json:"grade_levels"`
	GeneratedAt *time.Time `json:"generated_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

// CourseResponse represents a course record (Phase 3).
type CourseResponse struct {
	ID          uuid.UUID `json:"id"`
	StudentID   uuid.UUID `json:"student_id"`
	Title       string    `json:"title"`
	Subject     string    `json:"subject"`
	GradeLevel  int16     `json:"grade_level"`
	Credits     float64   `json:"credits"`
	GradeLetter *string   `json:"grade_letter"`
	GradePoints *float64  `json:"grade_points"`
	Level       string    `json:"level"`
	SchoolYear  string    `json:"school_year"`
	Semester    *string   `json:"semester"`
	CreatedAt   time.Time `json:"created_at"`
}

// CourseListResponse represents a paginated course list (Phase 3).
type CourseListResponse struct {
	Courses    []CourseResponse `json:"courses"`
	NextCursor *string          `json:"next_cursor"`
}

// ComplianceDashboardResponse represents the compliance dashboard overview.
type ComplianceDashboardResponse struct {
	FamilyConfig *FamilyConfigResponse      `json:"family_config"`
	Students     []StudentComplianceSummary `json:"students"`
}

// StudentComplianceSummary represents a student's compliance overview for the dashboard.
type StudentComplianceSummary struct {
	StudentID              uuid.UUID                  `json:"student_id"`
	StudentName            string                     `json:"student_name"`
	AttendanceSummary      AttendanceSummaryResponse  `json:"attendance_summary"`
	RecentAssessmentsCount int32                      `json:"recent_assessments_count"`
	RecentTestsCount       int32                      `json:"recent_tests_count"`
	ActivePortfolios       []PortfolioSummaryResponse `json:"active_portfolios"`
	PaceStatus             *string                    `json:"pace_status"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// GORM / DB Read Models [14-comply §3]
// ═══════════════════════════════════════════════════════════════════════════════

// ComplyStateConfig is the read model for comply_state_configs.
type ComplyStateConfig struct {
	StateCode             string
	StateName             string
	NotificationRequired  bool
	NotificationDetails   *string
	RequiredSubjects      []string
	AssessmentRequired    bool
	AssessmentDetails     *string
	RecordKeepingRequired bool
	RecordKeepingDetails  *string
	AttendanceRequired    bool
	AttendanceDays        *int16
	AttendanceHours       *int16
	AttendanceDetails     *string
	UmbrellaSchoolAvailable bool
	UmbrellaSchoolDetails *string
	RegulationLevel       string
	SyncedAt              time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ComplyFamilyConfig is the read model for comply_family_configs.
type ComplyFamilyConfig struct {
	FamilyID         uuid.UUID
	StateCode        string
	SchoolYearStart  time.Time
	SchoolYearEnd    time.Time
	TotalSchoolDays  int16
	CustomScheduleID *uuid.UUID
	GpaScale         string
	GpaCustomConfig  json.RawMessage
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ComplyCustomSchedule is the read model for comply_custom_schedules.
type ComplyCustomSchedule struct {
	ID               uuid.UUID
	FamilyID         uuid.UUID
	Name             string
	SchoolDays       []bool
	ExclusionPeriods json.RawMessage
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ComplyAttendance is the read model for comply_attendance.
type ComplyAttendance struct {
	ID              uuid.UUID
	FamilyID        uuid.UUID
	StudentID       uuid.UUID
	AttendanceDate  time.Time
	Status          string
	DurationMinutes *int16
	Notes           *string
	IsAuto          bool
	ManualOverride  bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ComplyAssessmentRecord is the read model for comply_assessment_records.
type ComplyAssessmentRecord struct {
	ID               uuid.UUID
	FamilyID         uuid.UUID
	StudentID        uuid.UUID
	Title            string
	Subject          string
	AssessmentType   string
	Score            *float64
	MaxScore         *float64
	GradeLetter      *string
	GradePoints      *float64
	IsPassing        *bool
	SourceActivityID *uuid.UUID
	AssessmentDate   time.Time
	Notes            *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ComplyStandardizedTest is the read model for comply_standardized_tests.
type ComplyStandardizedTest struct {
	ID             uuid.UUID
	FamilyID       uuid.UUID
	StudentID      uuid.UUID
	TestName       string
	TestDate       time.Time
	GradeLevel     *int16
	Scores         json.RawMessage
	CompositeScore *float64
	Percentile     *int16
	Notes          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ComplyPortfolio is the read model for comply_portfolios.
type ComplyPortfolio struct {
	ID                 uuid.UUID
	FamilyID           uuid.UUID
	StudentID          uuid.UUID
	Title              string
	Description        *string
	Organization       string
	DateRangeStart     time.Time
	DateRangeEnd       time.Time
	IncludeAttendance  bool
	IncludeAssessments bool
	Status             string
	UploadID           *uuid.UUID
	GeneratedAt        *time.Time
	ExpiresAt          *time.Time
	ErrorMessage       *string
	RetryCount         int16
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// ComplyPortfolioItem is the read model for comply_portfolio_items.
type ComplyPortfolioItem struct {
	ID                 uuid.UUID
	PortfolioID        uuid.UUID
	SourceType         string
	SourceID           uuid.UUID
	DisplayOrder       int16
	CachedTitle        string
	CachedSubject      *string
	CachedDate         time.Time
	CachedDescription  *string
	CachedAttachments  json.RawMessage
	CreatedAt          time.Time
}

// ComplyTranscript is the read model for comply_transcripts (Phase 3).
type ComplyTranscript struct {
	ID                    uuid.UUID
	FamilyID              uuid.UUID
	StudentID             uuid.UUID
	Title                 string
	StudentName           string
	GradeLevels           []string
	Status                string
	SnapshotGpaUnweighted *float64
	SnapshotGpaWeighted   *float64
	UploadID              *uuid.UUID
	GeneratedAt           *time.Time
	ExpiresAt             *time.Time
	ErrorMessage          *string
	RetryCount            int16
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// ComplyCourse is the read model for comply_courses (Phase 3).
type ComplyCourse struct {
	ID           uuid.UUID
	FamilyID     uuid.UUID
	StudentID    uuid.UUID
	TranscriptID *uuid.UUID
	Title        string
	Subject      string
	GradeLevel   int16
	Credits      float64
	GradeLetter  *string
	GradePoints  *float64
	Level        string
	SchoolYear   string
	Semester     *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Row Types (write models) [14-comply §6]
// ═══════════════════════════════════════════════════════════════════════════════

// UpsertStateConfigRow is the write model for state config upsert.
type UpsertStateConfigRow struct {
	StateCode             string
	StateName             string
	NotificationRequired  bool
	NotificationDetails   *string
	RequiredSubjects      []string
	AssessmentRequired    bool
	AssessmentDetails     *string
	RecordKeepingRequired bool
	RecordKeepingDetails  *string
	AttendanceRequired    bool
	AttendanceDays        *int16
	AttendanceHours       *int16
	AttendanceDetails     *string
	UmbrellaSchoolAvailable bool
	UmbrellaSchoolDetails *string
	RegulationLevel       string
}

// UpsertFamilyConfigRow is the write model for family config upsert.
type UpsertFamilyConfigRow struct {
	StateCode        string
	SchoolYearStart  time.Time
	SchoolYearEnd    time.Time
	TotalSchoolDays  int16
	CustomScheduleID *uuid.UUID
	GpaScale         string
	GpaCustomConfig  json.RawMessage
}

// CreateScheduleRow is the write model for custom schedule creation.
type CreateScheduleRow struct {
	Name             string
	SchoolDays       []bool
	ExclusionPeriods json.RawMessage
}

// UpdateScheduleRow is the write model for custom schedule update.
type UpdateScheduleRow struct {
	Name             *string
	SchoolDays       *[]bool
	ExclusionPeriods *json.RawMessage
}

// UpsertAttendanceRow is the write model for attendance upsert.
type UpsertAttendanceRow struct {
	StudentID       uuid.UUID
	AttendanceDate  time.Time
	Status          string
	DurationMinutes *int16
	Notes           *string
	IsAuto          bool
	ManualOverride  bool
}

// UpdateAttendanceRow is the write model for attendance update.
type UpdateAttendanceRow struct {
	Status          *string
	DurationMinutes *int16
	Notes           *string
}

// CreateAssessmentRow is the write model for assessment creation.
type CreateAssessmentRow struct {
	StudentID        uuid.UUID
	Title            string
	Subject          string
	AssessmentType   string
	Score            *float64
	MaxScore         *float64
	GradeLetter      *string
	GradePoints      *float64
	IsPassing        *bool
	SourceActivityID *uuid.UUID
	AssessmentDate   time.Time
	Notes            *string
}

// UpdateAssessmentRow is the write model for assessment update.
type UpdateAssessmentRow struct {
	Title          *string
	Subject        *string
	Score          *float64
	MaxScore       *float64
	GradeLetter    *string
	GradePoints    *float64
	IsPassing      *bool
	AssessmentDate *time.Time
	Notes          *string
}

// CreateTestScoreRow is the write model for test score creation.
type CreateTestScoreRow struct {
	StudentID      uuid.UUID
	TestName       string
	TestDate       time.Time
	GradeLevel     *int16
	Scores         json.RawMessage
	CompositeScore *float64
	Percentile     *int16
	Notes          *string
}

// UpdateTestScoreRow is the write model for test score update.
type UpdateTestScoreRow struct {
	TestName       *string
	TestDate       *time.Time
	Scores         *json.RawMessage
	CompositeScore *float64
	Percentile     *int16
	Notes          *string
}

// CreatePortfolioRow is the write model for portfolio creation.
type CreatePortfolioRow struct {
	StudentID          uuid.UUID
	Title              string
	Description        *string
	Organization       string
	DateRangeStart     time.Time
	DateRangeEnd       time.Time
	IncludeAttendance  bool
	IncludeAssessments bool
}

// CreatePortfolioItemRow is the write model for portfolio item creation.
type CreatePortfolioItemRow struct {
	PortfolioID       uuid.UUID
	SourceType        string
	SourceID          uuid.UUID
	DisplayOrder      int16
	CachedTitle       string
	CachedSubject     *string
	CachedDate        time.Time
	CachedDescription *string
	CachedAttachments json.RawMessage
}

// CreateTranscriptRow is the write model for transcript creation (Phase 3).
type CreateTranscriptRow struct {
	StudentID   uuid.UUID
	Title       string
	StudentName string
	GradeLevels []string
}

// CreateCourseRow is the write model for course creation (Phase 3).
type CreateCourseRow struct {
	StudentID    uuid.UUID
	TranscriptID *uuid.UUID
	Title        string
	Subject      string
	GradeLevel   int16
	Credits      float64
	GradeLetter  *string
	GradePoints  *float64
	Level        string
	SchoolYear   string
	Semester     *string
}

// UpdateCourseRow is the write model for course update (Phase 3).
type UpdateCourseRow struct {
	Title       *string
	Subject     *string
	Credits     *float64
	GradeLetter *string
	GradePoints *float64
	Level       *string
	Semester    *string
}

// AttendanceSummaryRow is the result of an attendance summary query.
type AttendanceSummaryRow struct {
	PresentFull    int32
	PresentPartial int32
	Absent         int32
	NotApplicable  int32
	TotalMinutes   int64
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cross-Domain Data Types [14-comply §17]
// ═══════════════════════════════════════════════════════════════════════════════

// PortfolioItemData is the cross-domain data returned by LearningServiceForComply.
type PortfolioItemData struct {
	Title       string
	Subject     *string
	Date        time.Time
	Description *string
	Attachments json.RawMessage
}

// StateRequirementsData is the cross-domain data returned by DiscoveryServiceForComply.
type StateRequirementsData struct {
	StateCode             string
	StateName             string
	NotificationRequired  bool
	NotificationDetails   *string
	RequiredSubjects      []string
	AssessmentRequired    bool
	AssessmentDetails     *string
	RecordKeepingRequired bool
	RecordKeepingDetails  *string
	AttendanceRequired    bool
	AttendanceDays        *int16
	AttendanceHours       *int16
	AttendanceDetails     *string
	UmbrellaSchoolAvailable bool
	UmbrellaSchoolDetails *string
	RegulationLevel       string
}

// StateGuideSummary is the cross-domain data for listing state guides.
type StateGuideSummary struct {
	StateCode       string
	StateName       string
	RegulationLevel string
}

// ═══════════════════════════════════════════════════════════════════════════════
// Consumed Event Projections (comply-local) [14-comply §5]
// ═══════════════════════════════════════════════════════════════════════════════

// ActivityLoggedEvent is comply's local projection of learn.ActivityLogged.
type ActivityLoggedEvent struct {
	FamilyID        uuid.UUID
	StudentID       uuid.UUID
	ActivityID      uuid.UUID
	SubjectTags     []string
	DurationMinutes *int16
	ActivityDate    time.Time
}

// StudentDeletedEvent is comply's local projection of iam.StudentDeleted.
type StudentDeletedEvent struct {
	FamilyID  uuid.UUID
	StudentID uuid.UUID
}

// FamilyDeletionScheduledEvent is comply's local projection of iam.FamilyDeletionScheduled.
type FamilyDeletionScheduledEvent struct {
	FamilyID    uuid.UUID
	DeleteAfter time.Time
}

// SubscriptionCancelledEvent is comply's local projection of billing.SubscriptionCancelled.
type SubscriptionCancelledEvent struct {
	FamilyID    uuid.UUID
	EffectiveAt time.Time
}
