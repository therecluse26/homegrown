package plan

import (
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Enums [17-planning §7]
// ═══════════════════════════════════════════════════════════════════════════════

// ScheduleCategory classifies schedule items.
type ScheduleCategory string

const (
	ScheduleCategoryLesson     ScheduleCategory = "lesson"
	ScheduleCategoryReading    ScheduleCategory = "reading"
	ScheduleCategoryActivity   ScheduleCategory = "activity"
	ScheduleCategoryAssessment ScheduleCategory = "assessment"
	ScheduleCategoryFieldTrip  ScheduleCategory = "field_trip"
	ScheduleCategoryCoOp       ScheduleCategory = "co_op"
	ScheduleCategoryBreak      ScheduleCategory = "break"
	ScheduleCategoryCustom     ScheduleCategory = "custom"
)

// CalendarSource identifies which domain provides calendar items. [17-planning §7]
type CalendarSource string

const (
	CalendarSourceSchedule   CalendarSource = "schedule"
	CalendarSourceActivities CalendarSource = "activities"
	CalendarSourceAttendance CalendarSource = "attendance"
	CalendarSourceEvents     CalendarSource = "events"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Database Row Type [17-planning §3]
// ═══════════════════════════════════════════════════════════════════════════════

// ScheduleItem is the GORM model for plan_schedule_items.
type ScheduleItem struct {
	ID               uuid.UUID        `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID         uuid.UUID        `gorm:"type:uuid;not null"`
	StudentID        *uuid.UUID       `gorm:"type:uuid"`
	Title            string           `gorm:"type:varchar(200);not null"`
	Description      *string          `gorm:"type:text"`
	StartDate        time.Time        `gorm:"type:date;not null"`
	StartTime        *string          `gorm:"type:time"`
	EndTime          *string          `gorm:"type:time"`
	DurationMinutes  *int             `gorm:"type:int"`
	Category         ScheduleCategory `gorm:"type:varchar(30);not null;default:'custom'"`
	SubjectID        *uuid.UUID       `gorm:"type:uuid"`
	Color            *string          `gorm:"type:varchar(7)"`
	IsCompleted      bool             `gorm:"not null;default:false"`
	CompletedAt      *time.Time       `gorm:"type:timestamptz"`
	LinkedActivityID *uuid.UUID       `gorm:"type:uuid"`
	LinkedEventID    *uuid.UUID       `gorm:"type:uuid"`
	RecurrenceRule   *string          `gorm:"type:varchar(255)"`
	RecurrenceEnd    *time.Time       `gorm:"type:date"`
	Notes            *string          `gorm:"type:text"`
	CreatedAt        time.Time        `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt        time.Time        `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the PostgreSQL table name for GORM.
func (ScheduleItem) TableName() string { return "plan_schedule_items" }

// ═══════════════════════════════════════════════════════════════════════════════
// Request Types [17-planning §7]
// ═══════════════════════════════════════════════════════════════════════════════

// CreateScheduleItemInput is the input for creating a schedule item.
type CreateScheduleItemInput struct {
	Title           string            `json:"title"            validate:"required,max=200"`
	Description     *string           `json:"description"`
	StudentID       *uuid.UUID        `json:"student_id"`
	StartDate       time.Time         `json:"start_date"       validate:"required"`
	StartTime       *string           `json:"start_time"`
	EndTime         *string           `json:"end_time"`
	DurationMinutes *int              `json:"duration_minutes"`
	Category        *ScheduleCategory `json:"category"`
	SubjectID       *uuid.UUID        `json:"subject_id"`
	Color           *string           `json:"color"`
	Notes           *string           `json:"notes"`
}

// UpdateScheduleItemInput is the input for updating a schedule item (partial update).
type UpdateScheduleItemInput struct {
	Title           *string           `json:"title"`
	Description     *string           `json:"description"`
	StudentID       *uuid.UUID        `json:"student_id"`
	StartDate       *time.Time        `json:"start_date"`
	StartTime       *string           `json:"start_time"`
	EndTime         *string           `json:"end_time"`
	DurationMinutes *int              `json:"duration_minutes"`
	Category        *ScheduleCategory `json:"category"`
	SubjectID       *uuid.UUID        `json:"subject_id"`
	Color           *string           `json:"color"`
	Notes           *string           `json:"notes"`
}

// LogAsActivityInput provides additional details for logging as a learning activity. [17-planning §10.2]
type LogAsActivityInput struct {
	Description *string  `json:"description"`
	Tags        []string `json:"tags"`
}

// CalendarQuery represents filter parameters for calendar queries. [17-planning §7]
type CalendarQuery struct {
	Start     time.Time        `json:"start"      validate:"required"`
	End       time.Time        `json:"end"        validate:"required"`
	StudentID *uuid.UUID       `json:"student_id"`
	Sources   []CalendarSource `json:"sources"`
}

// ScheduleItemQuery provides filter parameters for listing schedule items. [17-planning §7]
type ScheduleItemQuery struct {
	StartDate   *time.Time        `json:"start_date"`
	EndDate     *time.Time        `json:"end_date"`
	StudentID   *uuid.UUID        `json:"student_id"`
	Category    *ScheduleCategory `json:"category"`
	IsCompleted *bool             `json:"is_completed"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Types [17-planning §7]
// ═══════════════════════════════════════════════════════════════════════════════

// CalendarResponse contains the aggregated calendar for a date range.
type CalendarResponse struct {
	Start time.Time     `json:"start"`
	End   time.Time     `json:"end"`
	Days  []CalendarDay `json:"days"`
}

// CalendarDay contains all calendar items for a single date.
type CalendarDay struct {
	Date  time.Time      `json:"date"`
	Items []CalendarItem `json:"items"`
}

// CalendarItem represents a single entry on the calendar from any source.
type CalendarItem struct {
	ID          uuid.UUID      `json:"id"`
	Source      CalendarSource `json:"source"`
	Title       string         `json:"title"`
	StartTime   *string        `json:"start_time"`
	EndTime     *string        `json:"end_time"`
	Category    *string        `json:"category"`
	Color       *string        `json:"color"`
	StudentID   *uuid.UUID     `json:"student_id"`
	IsCompleted *bool          `json:"is_completed"`
	Date        time.Time      `json:"date"`
}

// ScheduleItemResponse is the response representation of a schedule item.
type ScheduleItemResponse struct {
	ID               uuid.UUID        `json:"id"`
	Title            string           `json:"title"`
	Description      *string          `json:"description"`
	StudentID        *uuid.UUID       `json:"student_id"`
	StudentName      *string          `json:"student_name"`
	StartDate        time.Time        `json:"start_date"`
	StartTime        *string          `json:"start_time"`
	EndTime          *string          `json:"end_time"`
	DurationMinutes  *int             `json:"duration_minutes"`
	Category         ScheduleCategory `json:"category"`
	SubjectID        *uuid.UUID       `json:"subject_id"`
	Color            *string          `json:"color"`
	IsCompleted      bool             `json:"is_completed"`
	CompletedAt      *time.Time       `json:"completed_at"`
	LinkedActivityID *uuid.UUID       `json:"linked_activity_id"`
	Notes            *string          `json:"notes"`
	CreatedAt        time.Time        `json:"created_at"`
}

// DayViewResponse contains all items for a single day, grouped by source. [17-planning §7]
type DayViewResponse struct {
	Date          time.Time              `json:"date"`
	ScheduleItems []ScheduleItemResponse `json:"schedule_items"`
	Activities    []ActivitySummary      `json:"activities"`
	Attendance    *AttendanceSummary     `json:"attendance"`
	Events        []EventSummary         `json:"events"`
}

// ActivitySummary is a read-only DTO for learning activities from learn::. [17-planning §9.1]
type ActivitySummary struct {
	ID        uuid.UUID  `json:"id"`
	Title     string     `json:"title"`
	Date      time.Time  `json:"date"`
	StudentID *uuid.UUID `json:"student_id"`
	Subject   *string    `json:"subject"`
	Tags      []string   `json:"tags"`
}

// AttendanceSummary is a read-only DTO for attendance records from comply::. [17-planning §9.1]
type AttendanceSummary struct {
	Date      time.Time  `json:"date"`
	StudentID *uuid.UUID `json:"student_id"`
	Status    string     `json:"status"`
}

// EventSummary is a read-only DTO for social events from social::. [17-planning §9.1]
type EventSummary struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	Date      time.Time `json:"date"`
	StartTime *string   `json:"start_time"`
	EndTime   *string   `json:"end_time"`
	GroupName *string   `json:"group_name"`
	Location  *string   `json:"location"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Schedule Template Types [17-planning §3.1, §11.3]
// ═══════════════════════════════════════════════════════════════════════════════

// ScheduleTemplate is the GORM model for plan_schedule_templates.
type ScheduleTemplate struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID    uuid.UUID `gorm:"type:uuid;not null"`
	Name        string    `gorm:"type:varchar(100);not null"`
	Description *string   `gorm:"type:text"`
	Items       []byte    `gorm:"type:jsonb;not null;default:'[]'"` // JSON array of TemplateItem
	IsActive    bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt   time.Time `gorm:"type:timestamptz;not null;default:now()"`
}

// TableName returns the PostgreSQL table name for GORM.
func (ScheduleTemplate) TableName() string { return "plan_schedule_templates" }

// TemplateItem is a single item within a schedule template's Items JSONB. [17-planning §11.3]
type TemplateItem struct {
	DayOfWeek string           `json:"day_of_week" validate:"required,oneof=monday tuesday wednesday thursday friday saturday sunday"`
	StartTime *string          `json:"start_time"`
	EndTime   *string          `json:"end_time"`
	Title     string           `json:"title"      validate:"required,max=200"`
	Category  ScheduleCategory `json:"category"`
	SubjectID *uuid.UUID       `json:"subject_id,omitempty"`
	Color     *string          `json:"color,omitempty"`
}

// CreateTemplateInput is the input for creating a schedule template.
type CreateTemplateInput struct {
	Name        string         `json:"name"        validate:"required,max=100"`
	Description *string        `json:"description"`
	Items       []TemplateItem `json:"items"       validate:"required,min=1"`
	IsActive    bool           `json:"is_active"`
}

// UpdateTemplateInput is the input for updating a schedule template (partial update).
type UpdateTemplateInput struct {
	Name        *string         `json:"name"`
	Description *string         `json:"description"`
	Items       *[]TemplateItem `json:"items"`
	IsActive    *bool           `json:"is_active"`
}

// ApplyTemplateInput defines the date range for applying a template. [17-planning §11.3]
type ApplyTemplateInput struct {
	StartDate time.Time `json:"start_date" validate:"required"`
	EndDate   time.Time `json:"end_date"   validate:"required"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Week View Types [17-planning §7]
// ═══════════════════════════════════════════════════════════════════════════════

// WeekViewResponse contains all items for a 7-day week, structured by day.
type WeekViewResponse struct {
	WeekStart time.Time     `json:"week_start"`
	WeekEnd   time.Time     `json:"week_end"`
	Days      []CalendarDay `json:"days"`
}

// ScheduleItemListResponse is the paginated response for schedule items (swag-compatible alias).
type ScheduleItemListResponse struct {
	Data       []ScheduleItemResponse `json:"data"`
	NextCursor *string                `json:"next_cursor"`
	HasMore    bool                   `json:"has_more"`
}

// TemplateResponse is the response representation of a schedule template.
type TemplateResponse struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Description *string        `json:"description"`
	Items       []TemplateItem `json:"items"`
	IsActive    bool           `json:"is_active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

