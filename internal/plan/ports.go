package plan

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [17-planning §5]
// ═══════════════════════════════════════════════════════════════════════════════

// PlanningService defines the service contract for planning and scheduling.
type PlanningService interface {
	// === Calendar View (Read) ===

	// GetCalendar returns an aggregated calendar for a date range.
	GetCalendar(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		params CalendarQuery,
	) (CalendarResponse, error)

	// GetDayView returns a detailed day view with all items grouped by source.
	GetDayView(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		date time.Time,
		studentID *uuid.UUID,
	) (DayViewResponse, error)

	// === Schedule Items (Write) ===

	// CreateScheduleItem creates a new schedule item.
	CreateScheduleItem(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		input CreateScheduleItemInput,
	) (uuid.UUID, error)

	// UpdateScheduleItem updates a schedule item (partial update).
	UpdateScheduleItem(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		itemID uuid.UUID,
		input UpdateScheduleItemInput,
	) error

	// DeleteScheduleItem deletes a schedule item.
	DeleteScheduleItem(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		itemID uuid.UUID,
	) error

	// CompleteScheduleItem marks a schedule item as completed.
	CompleteScheduleItem(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		itemID uuid.UUID,
	) error

	// LogAsActivity logs a completed schedule item as a learning activity.
	// Returns the created activity ID.
	LogAsActivity(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		itemID uuid.UUID,
		input LogAsActivityInput,
	) (uuid.UUID, error)

	// ListScheduleItems lists schedule items with filters.
	ListScheduleItems(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		params ScheduleItemQuery,
		pagination *shared.PaginationParams,
	) (*shared.PaginatedResponse[ScheduleItemResponse], error)

	// === Single Item ===

	// GetScheduleItem returns a single schedule item by ID with student name enriched. [17-planning §4]
	GetScheduleItem(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		itemID uuid.UUID,
	) (ScheduleItemResponse, error)

	// === Print/Export ===

	// GetPrintView generates a print-friendly HTML view of the calendar.
	GetPrintView(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		start time.Time,
		end time.Time,
		studentID *uuid.UUID,
	) (string, error)

	// === Schedule Templates [17-planning §11.3] ===

	// CreateTemplate creates a new schedule template.
	CreateTemplate(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		input CreateTemplateInput,
	) (uuid.UUID, error)

	// ListTemplates returns all templates for the family.
	ListTemplates(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
	) ([]TemplateResponse, error)

	// UpdateTemplate updates a schedule template.
	UpdateTemplate(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		templateID uuid.UUID,
		input UpdateTemplateInput,
	) error

	// DeleteTemplate deletes a schedule template.
	DeleteTemplate(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		templateID uuid.UUID,
	) error

	// ApplyTemplate creates schedule items from a template for a date range.
	// Returns the IDs of created schedule items. [17-planning §11.3]
	ApplyTemplate(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		templateID uuid.UUID,
		input ApplyTemplateInput,
	) ([]uuid.UUID, error)

	// === Event Handlers [17-planning §16] ===

	// HandleEventCancelled marks linked schedule items as cancelled when a social event is cancelled.
	HandleEventCancelled(ctx context.Context, eventID uuid.UUID, goingFamilyIDs []uuid.UUID) error

	// HandleActivityLogged marks matching schedule items as completed when an activity is logged.
	HandleActivityLogged(ctx context.Context, familyID, studentID, activityID uuid.UUID) error

	// === Data Lifecycle ===

	// ExportData returns schedule items + templates as JSON for data export. [17-planning §14.1]
	ExportData(ctx context.Context, scope *shared.FamilyScope) ([]byte, error)

	// DeleteData removes all plan_ data for the family. [17-planning §14.2]
	DeleteData(ctx context.Context, scope *shared.FamilyScope) error

	// DeleteStudentData removes schedule items for a specific student. [17-planning §14.3]
	DeleteStudentData(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [17-planning §6]
// ═══════════════════════════════════════════════════════════════════════════════

// ScheduleItemRepository defines data access for plan_schedule_items.
type ScheduleItemRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, item *ScheduleItem) error

	FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ScheduleItem, error)

	// FindByLinkedEventID returns schedule items linked to a social event. [17-planning §16]
	FindByLinkedEventID(ctx context.Context, eventID uuid.UUID) ([]ScheduleItem, error)

	// FindByStudentAndDate returns schedule items for a student on a specific date. [17-planning §16]
	FindByStudentAndDate(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, date time.Time) ([]ScheduleItem, error)

	ListByDateRange(
		ctx context.Context,
		scope *shared.FamilyScope,
		start time.Time,
		end time.Time,
		studentID *uuid.UUID,
	) ([]ScheduleItem, error)

	ListFiltered(
		ctx context.Context,
		scope *shared.FamilyScope,
		query *ScheduleItemQuery,
		pagination *shared.PaginationParams,
	) ([]ScheduleItem, error)

	Update(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, input *UpdateScheduleItemInput) error

	MarkCompleted(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, completedAt time.Time) error

	SetLinkedActivity(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, activityID uuid.UUID) error

	Delete(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error

	DeleteAllByFamily(ctx context.Context, scope *shared.FamilyScope) error

	// DeleteByStudent deletes all schedule items for a specific student. [17-planning §14.3]
	DeleteByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error

	ListAllByFamily(ctx context.Context, scope *shared.FamilyScope) ([]ScheduleItem, error)
}

// ScheduleTemplateRepository defines data access for plan_schedule_templates. [17-planning §6]
type ScheduleTemplateRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, tmpl *ScheduleTemplate) error

	FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*ScheduleTemplate, error)

	ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]ScheduleTemplate, error)

	Update(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID, input *UpdateTemplateInput, itemsJSON []byte) error

	Delete(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) error

	DeleteAllByFamily(ctx context.Context, scope *shared.FamilyScope) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer-Defined Cross-Domain Interfaces [17-planning §8, §18]
// ═══════════════════════════════════════════════════════════════════════════════

// IamServiceForPlan is a consumer-defined interface for cross-domain reads from iam::.
// Implemented by a function adapter in main.go. [ARCH §4.4]
type IamServiceForPlan interface {
	// StudentBelongsToFamily checks if a student belongs to the given family.
	StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error)

	// GetStudentName returns the display name for a student.
	GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error)
}

// LearningServiceForPlan is a consumer-defined interface for cross-domain calls to learn::.
// Implemented by a function adapter in main.go. [ARCH §4.4]
type LearningServiceForPlan interface {
	// ListActivitiesForCalendar returns activities in a date range for calendar aggregation.
	ListActivitiesForCalendar(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		start time.Time,
		end time.Time,
		studentID *uuid.UUID,
	) ([]ActivitySummary, error)

	// LogActivity creates a learning activity from a schedule item.
	// Returns the new activity ID.
	LogActivity(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		title string,
		date time.Time,
		durationMinutes *int,
		studentID *uuid.UUID,
		description *string,
		tags []string,
	) (uuid.UUID, error)
}

// ComplianceServiceForPlan is a consumer-defined interface for cross-domain reads from comply::.
// Implemented by a function adapter in main.go. [ARCH §4.4]
type ComplianceServiceForPlan interface {
	// GetAttendanceRange returns attendance records in a date range.
	GetAttendanceRange(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		start time.Time,
		end time.Time,
		studentID *uuid.UUID,
	) ([]AttendanceSummary, error)
}

// SocialServiceForPlan is a consumer-defined interface for cross-domain reads from social::.
// Implemented by a function adapter in main.go. [ARCH §4.4]
type SocialServiceForPlan interface {
	// GetEventsForCalendar returns events in a date range.
	GetEventsForCalendar(
		ctx context.Context,
		auth *shared.AuthContext,
		scope *shared.FamilyScope,
		start time.Time,
		end time.Time,
	) ([]EventSummary, error)
}
