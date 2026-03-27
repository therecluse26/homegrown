# Domain Spec 17 — Planning & Scheduling (plan::)

## §1 Overview

The Planning domain provides a **unified calendar and scheduling interface** for homeschool
families. It synthesizes data from Learning (activities), Compliance (attendance), and Social
(events) into a single calendar view, and provides schedule creation tools for daily and
weekly planning. Homeschool families plan their days around learning activities, co-op
meetings, and social events — this domain serves that core workflow. `[S§17, ADR-013]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/plan/` |
| **DB prefix** | `plan_` |
| **Complexity class** | Non-complex (no `domain/` subdirectory) `[ARCH §4.5]` |
| **CQRS** | Yes — calendar reads aggregate from multiple domains; schedule writes are simple `[ARCH §4.7]` |
| **External adapter** | None |
| **Key constraint** | Every user-data query family-scoped via `FamilyScope` `[CODING §2.4, §2.5]`; calendar view MUST NOT duplicate data from other domains — aggregates via service interfaces; schedule items owned by plan:: |

**What plan:: owns**: Schedule items (family-created calendar entries), recurring schedule
templates (Phase 2), schedule sharing preferences (Phase 2), calendar view aggregation logic.

**What plan:: does NOT own**: Learning activities (owned by `learn::`). Attendance records
(owned by `comply::`). Social events (owned by `social::`). Student profiles (owned by
`iam::`). Methodology tool configuration (owned by `method::`).

**What plan:: delegates**: Activity data → `learn::LearningService` (read). Attendance data →
`comply::ComplianceService` (read). Event data → `social::SocialService` (read). Student
info → `iam::IamService` (read). Notification delivery → `notify::` (via domain events).

---

## §2 Requirements Traceability

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Calendar view synthesizing activities + events | `[S§18.9]` | §5, §9 |
| Weekly/daily schedule creation | `[S§19 Phase 1]` | §3, §5, §10 |
| Recurring schedule templates | `[S§19 Phase 2]` | §11 (Phase 2) |
| Co-op day coordination | `[S§18.10]` | §12 (Phase 2) |
| Print-friendly schedule output | `[S§17.9]` | §13 |
| Data export for planning data | `[S§8.5]` | §14 |

---

## §3 Database Schema

All tables use the `plan_` prefix. `[ARCH §5.1]`

### §3.1 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_plan_tables.sql
-- =============================================================================

-- Schedule items: family-created calendar entries
-- These are plan::-owned data, NOT duplicates of learning activities or events
CREATE TABLE plan_schedule_items (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    -- Who this item is for (NULL = whole family)
    student_id      UUID REFERENCES iam_students(id),
    -- Schedule details
    title           VARCHAR(200) NOT NULL,
    description     TEXT,
    -- Time block
    start_date      DATE NOT NULL,
    start_time      TIME,                  -- NULL = all-day item
    end_time        TIME,
    duration_minutes INT,                  -- computed or explicit
    -- Categorization
    category        VARCHAR(30) NOT NULL DEFAULT 'custom'
                    CHECK (category IN (
                        'lesson', 'reading', 'activity', 'assessment',
                        'field_trip', 'co_op', 'break', 'custom'
                    )),
    -- Subject (optional, from learn:: taxonomy)
    subject_id      UUID,                  -- references learn_subjects(id)
    -- Color for calendar display
    color           VARCHAR(7),            -- hex color, e.g., "#3B82F6"
    -- Completion status
    is_completed    BOOLEAN NOT NULL DEFAULT false,
    completed_at    TIMESTAMPTZ,
    -- Link to other domain entities (optional)
    linked_activity_id UUID,               -- if this schedule item was logged as an activity
    linked_event_id    UUID,               -- if created from a social event
    -- Recurrence (Phase 2 — stored as RRULE string)
    recurrence_rule VARCHAR(255),          -- e.g., "FREQ=WEEKLY;BYDAY=MO,WE,FR"
    recurrence_end  DATE,
    -- Metadata
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_plan_schedule_items_family_date
    ON plan_schedule_items(family_id, start_date);
CREATE INDEX idx_plan_schedule_items_student
    ON plan_schedule_items(student_id, start_date)
    WHERE student_id IS NOT NULL;
CREATE INDEX idx_plan_schedule_items_linked_activity
    ON plan_schedule_items(linked_activity_id)
    WHERE linked_activity_id IS NOT NULL;

-- Schedule templates (Phase 2): reusable weekly schedule patterns
CREATE TABLE plan_schedule_templates (
    id              UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id       UUID NOT NULL REFERENCES iam_families(id),
    name            VARCHAR(100) NOT NULL,
    description     TEXT,
    -- Template items stored as JSONB array
    -- Each item: { day_of_week, start_time, end_time, title, category, subject_id, color }
    items           JSONB NOT NULL DEFAULT '[]'::JSONB,
    is_active       BOOLEAN NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_plan_schedule_templates_family
    ON plan_schedule_templates(family_id);
```

### §3.2 Row-Level Security

```sql
ALTER TABLE plan_schedule_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY plan_schedule_items_family_scope ON plan_schedule_items
    USING (family_id = current_setting('app.current_family_id')::UUID);

ALTER TABLE plan_schedule_templates ENABLE ROW LEVEL SECURITY;
CREATE POLICY plan_schedule_templates_family_scope ON plan_schedule_templates
    USING (family_id = current_setting('app.current_family_id')::UUID);
```

---

## §4 API Endpoints

```
# Calendar view (aggregated from multiple domains)
GET    /v1/planning/calendar                    # Get calendar view for date range
GET    /v1/planning/calendar/day/:date          # Get detailed day view
GET    /v1/planning/calendar/week/:date         # Get week view (Monday-start)

# Schedule items (plan::-owned)
POST   /v1/planning/schedule-items              # Create schedule item
GET    /v1/planning/schedule-items              # List schedule items (filterable)
GET    /v1/planning/schedule-items/:id          # Get schedule item detail
PATCH  /v1/planning/schedule-items/:id          # Update schedule item
DELETE /v1/planning/schedule-items/:id          # Delete schedule item
PATCH  /v1/planning/schedule-items/:id/complete # Mark item as completed
POST   /v1/planning/schedule-items/:id/log      # Log completed item as learning activity

# Schedule templates (Phase 2)
GET    /v1/planning/templates                   # List templates
POST   /v1/planning/templates                   # Create template
PATCH  /v1/planning/templates/:id               # Update template
DELETE /v1/planning/templates/:id               # Delete template
POST   /v1/planning/templates/:id/apply         # Apply template to a date range

# Print/export
GET    /v1/planning/calendar/print?start=&end=  # Print-friendly HTML view
GET    /v1/planning/calendar/pdf?start=&end=    # PDF export (Phase 2)
```

---

## §5 Service Interface

```go
// PlanningService defines the service contract for planning and scheduling.
type PlanningService interface {
    // === Calendar View (Read) ===

    // GetCalendar returns an aggregated calendar for a date range.
    // Combines: plan:: schedule items + learn:: activities + comply:: attendance
    //           + social:: events.
    GetCalendar(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        params CalendarQuery,
    ) (CalendarResponse, error)

    // GetDayView returns a detailed day view with all items, activities, and events.
    GetDayView(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        date time.Time,
        studentID *uuid.UUID,
    ) (DayViewResponse, error)

    // GetScheduleItem returns a single schedule item by ID with student name enriched.
    GetScheduleItem(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        itemID uuid.UUID,
    ) (ScheduleItemResponse, error)

    // === Schedule Items (Write) ===

    // CreateScheduleItem creates a new schedule item.
    CreateScheduleItem(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        input CreateScheduleItemInput,
    ) (uuid.UUID, error)

    // UpdateScheduleItem updates a schedule item.
    UpdateScheduleItem(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        itemID uuid.UUID,
        input UpdateScheduleItemInput,
    ) error

    // DeleteScheduleItem deletes a schedule item.
    DeleteScheduleItem(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        itemID uuid.UUID,
    ) error

    // CompleteScheduleItem marks a schedule item as completed.
    CompleteScheduleItem(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        itemID uuid.UUID,
    ) error

    // LogAsActivity logs a completed schedule item as a learning activity.
    // Creates an activity in learn:: and links it back to this schedule item.
    // Returns the created activity ID.
    LogAsActivity(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        itemID uuid.UUID,
        input LogAsActivityInput,
    ) (uuid.UUID, error)

    // ListScheduleItems lists schedule items with filters.
    ListScheduleItems(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        params ScheduleItemQuery,
        pagination PaginationParams,
    ) (PaginatedResponse[ScheduleItemResponse], error)

    // === Print/Export ===

    // GetPrintView generates a print-friendly HTML view of the calendar.
    // Returns an HTML string.
    GetPrintView(
        ctx context.Context,
        auth *AuthContext,
        scope *FamilyScope,
        start time.Time,
        end time.Time,
        studentID *uuid.UUID,
    ) (string, error)
}
```

---

## §6 Repository Interfaces

```go
// ScheduleItemRepository defines data access for plan_schedule_items.
type ScheduleItemRepository interface {
    Create(
        ctx context.Context,
        scope *FamilyScope,
        item *ScheduleItem,
    ) error

    FindByID(
        ctx context.Context,
        scope *FamilyScope,
        id uuid.UUID,
    ) (*ScheduleItem, error)

    ListByDateRange(
        ctx context.Context,
        scope *FamilyScope,
        start time.Time,
        end time.Time,
        studentID *uuid.UUID,
    ) ([]ScheduleItem, error)

    ListFiltered(
        ctx context.Context,
        scope *FamilyScope,
        query *ScheduleItemQuery,
        pagination *PaginationParams,
    ) ([]ScheduleItem, error)

    ListAllByFamily(
        ctx context.Context,
        scope *FamilyScope,
    ) ([]ScheduleItem, error)

    Update(
        ctx context.Context,
        scope *FamilyScope,
        id uuid.UUID,
        input *UpdateScheduleItemInput,
    ) error

    MarkCompleted(
        ctx context.Context,
        scope *FamilyScope,
        id uuid.UUID,
        completedAt time.Time,
    ) error

    SetLinkedActivity(
        ctx context.Context,
        scope *FamilyScope,
        id uuid.UUID,
        activityID uuid.UUID,
    ) error

    FindByLinkedEventID(
        ctx context.Context,
        eventID uuid.UUID,
    ) ([]ScheduleItem, error)

    FindByStudentAndDate(
        ctx context.Context,
        scope *FamilyScope,
        studentID uuid.UUID,
        date time.Time,
    ) ([]ScheduleItem, error)

    Delete(
        ctx context.Context,
        scope *FamilyScope,
        id uuid.UUID,
    ) error

    DeleteAllByFamily(
        ctx context.Context,
        scope *FamilyScope,
    ) error
}

// ScheduleTemplateRepository defines data access for plan_schedule_templates.
type ScheduleTemplateRepository interface {
    Create(
        ctx context.Context,
        scope *FamilyScope,
        tmpl *ScheduleTemplate,
    ) error

    ListByFamily(
        ctx context.Context,
        scope *FamilyScope,
    ) ([]ScheduleTemplate, error)

    FindByID(
        ctx context.Context,
        scope *FamilyScope,
        id uuid.UUID,
    ) (*ScheduleTemplate, error)

    Update(
        ctx context.Context,
        scope *FamilyScope,
        id uuid.UUID,
        input *UpdateTemplateInput,
        itemsJSON []byte,
    ) error

    Delete(
        ctx context.Context,
        scope *FamilyScope,
        id uuid.UUID,
    ) error

    DeleteAllByFamily(
        ctx context.Context,
        scope *FamilyScope,
    ) error
}
```

---

## §7 Models (DTOs)

```go
// --- Request types ---

// CalendarQuery represents filter parameters for calendar queries.
type CalendarQuery struct {
    Start     time.Time        `json:"start"      validate:"required"`
    End       time.Time        `json:"end"        validate:"required"`
    StudentID *uuid.UUID       `json:"student_id"`
    // Which sources to include (default: all)
    Sources   []CalendarSource `json:"sources"`
}

// CalendarSource identifies which domain provides calendar items.
type CalendarSource string

const (
    CalendarSourceSchedule   CalendarSource = "schedule"    // plan:: schedule items
    CalendarSourceActivities CalendarSource = "activities"   // learn:: logged activities
    CalendarSourceAttendance CalendarSource = "attendance"   // comply:: attendance records
    CalendarSourceEvents     CalendarSource = "events"       // social:: events
)

// CreateScheduleItemInput is the request body for creating a schedule item.
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

// UpdateScheduleItemInput is the request body for updating a schedule item.
// Pointer fields use nil = "don't update", non-nil = "set to this value".
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

// LogAsActivityInput provides additional details for logging a schedule item
// as a learning activity.
type LogAsActivityInput struct {
    // Additional details for the activity log entry
    Description *string  `json:"description"`
    Tags        []string `json:"tags"`
}

// ScheduleItemQuery provides filter parameters for listing schedule items.
type ScheduleItemQuery struct {
    StartDate   *time.Time        `json:"start_date"`
    EndDate     *time.Time        `json:"end_date"`
    StudentID   *uuid.UUID        `json:"student_id"`
    Category    *ScheduleCategory `json:"category"`
    IsCompleted *bool             `json:"is_completed"`
}

// --- Response types ---

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
    ID              uuid.UUID           `json:"id"`
    Source          CalendarSource      `json:"source"`
    Title           string              `json:"title"`
    StartTime       *string             `json:"start_time"`
    EndTime         *string             `json:"end_time"`
    DurationMinutes *int                `json:"duration_minutes,omitempty"`
    Category        *string             `json:"category"`
    Color           *string             `json:"color"`
    StudentID       *uuid.UUID          `json:"student_id"`
    StudentName     *string             `json:"student_name,omitempty"`
    IsCompleted     *bool               `json:"is_completed"`
    Date            time.Time           `json:"date"`
    // Source-specific details
    Details         CalendarItemDetails `json:"details"`
}

// CalendarItemDetails holds source-specific detail fields.
// The Type field acts as a discriminator (analogous to a tagged union).
type CalendarItemDetails struct {
    Type string `json:"type"` // "schedule", "activity", "attendance", "event"

    // Schedule fields (Type == "schedule")
    Description      *string    `json:"description,omitempty"`
    Notes            *string    `json:"notes,omitempty"`
    LinkedActivityID *uuid.UUID `json:"linked_activity_id,omitempty"`

    // Activity fields (Type == "activity")
    Subject *string  `json:"subject,omitempty"`
    Tags    []string `json:"tags,omitempty"`

    // Attendance fields (Type == "attendance")
    Status *string `json:"status,omitempty"` // "present", "absent", "holiday"

    // Event fields (Type == "event")
    GroupName  *string `json:"group_name,omitempty"`
    Location   *string `json:"location,omitempty"`
    RSVPStatus *string `json:"rsvp_status,omitempty"`
}

// DayViewResponse contains all items for a single day, grouped by source.
type DayViewResponse struct {
    Date          time.Time              `json:"date"`
    ScheduleItems []ScheduleItemResponse `json:"schedule_items"`
    Activities    []ActivitySummary      `json:"activities"`
    Attendance    *AttendanceSummary     `json:"attendance"`
    Events        []EventSummary         `json:"events"`
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
    SubjectName      *string          `json:"subject_name"`
    Color            *string          `json:"color"`
    IsCompleted      bool             `json:"is_completed"`
    CompletedAt      *time.Time       `json:"completed_at"`
    LinkedActivityID *uuid.UUID       `json:"linked_activity_id"`
    Notes            *string          `json:"notes"`
    CreatedAt        time.Time        `json:"created_at"`
}

// AttendanceSummary is a read-only DTO for attendance records from comply::. [§9.1]
type AttendanceSummary struct {
    ID        uuid.UUID  `json:"id"`
    Date      time.Time  `json:"date"`
    StudentID *uuid.UUID `json:"student_id"`
    Status    string     `json:"status"` // "present", "absent", "holiday"
}

// EventSummary is a read-only DTO for social events from social::. [§9.1]
type EventSummary struct {
    ID         uuid.UUID `json:"id"`
    Title      string    `json:"title"`
    Date       time.Time `json:"date"`
    StartTime  *string   `json:"start_time"`
    EndTime    *string   `json:"end_time"`
    GroupName  *string   `json:"group_name"`
    Location   *string   `json:"location"`
    RSVPStatus *string   `json:"rsvp_status,omitempty"`
}

// ActivitySummary is a read-only DTO for learning activities from learn::. [§9.1]
type ActivitySummary struct {
    ID        uuid.UUID  `json:"id"`
    Title     string     `json:"title"`
    Date      time.Time  `json:"date"`
    StudentID *uuid.UUID `json:"student_id"`
    Subject   *string    `json:"subject"`
    Tags      []string   `json:"tags"`
}

// WeekViewResponse contains all items for a 7-day week, structured by day.
type WeekViewResponse struct {
    WeekStart time.Time     `json:"week_start"`
    WeekEnd   time.Time     `json:"week_end"`
    Days      []CalendarDay `json:"days"`
}

// TemplateItem is a single item within a schedule template's Items JSONB. [§11.3]
type TemplateItem struct {
    DayOfWeek string           `json:"day_of_week" validate:"required,oneof=monday tuesday wednesday thursday friday saturday sunday"`
    StartTime *string          `json:"start_time"`
    EndTime   *string          `json:"end_time"`
    Title     string           `json:"title"      validate:"required,max=200"`
    Category  ScheduleCategory `json:"category"`
    SubjectID *uuid.UUID       `json:"subject_id,omitempty"`
    Color     *string          `json:"color,omitempty"`
}

// CreateTemplateInput is the input for creating a schedule template. [§11.3]
type CreateTemplateInput struct {
    Name        string         `json:"name"        validate:"required,max=100"`
    Description *string        `json:"description"`
    Items       []TemplateItem `json:"items"       validate:"required,min=1"`
    IsActive    bool           `json:"is_active"`
}

// UpdateTemplateInput is the input for updating a schedule template (partial update). [§11.3]
type UpdateTemplateInput struct {
    Name        *string         `json:"name"`
    Description *string         `json:"description"`
    Items       *[]TemplateItem `json:"items"`
    IsActive    *bool           `json:"is_active"`
}

// ApplyTemplateInput defines the date range for applying a template. [§11.3]
type ApplyTemplateInput struct {
    StartDate time.Time `json:"start_date" validate:"required"`
    EndDate   time.Time `json:"end_date"   validate:"required"`
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

// --- Enums ---

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
```

---

## §8 Adapter Interfaces

plan:: does NOT have external adapters. It reads from other domains via their service
interfaces:

```go
// PlanningServiceImpl holds dependencies for the planning service.
// Read-only dependencies on other domains are injected as interfaces.
type PlanningServiceImpl struct {
    scheduleRepo      ScheduleItemRepository
    templateRepo      ScheduleTemplateRepository
    // Read-only dependencies on other domains:
    learningService   LearningService
    complianceService ComplianceService
    socialService     SocialService
    iamService        IamService
    eventBus          *EventBus
}
```

---

## §9 Calendar Aggregation (Domain Deep-Dive 1)

### §9.1 Data Sources

The calendar view aggregates four data sources:

| Source | Domain | Data | Read Method |
|--------|--------|------|-------------|
| Schedule items | plan:: | Family-created calendar entries | Direct DB query |
| Activities | learn:: | Logged learning activities with dates | `LearningService.ListActivitiesForCalendar()` |
| Attendance | comply:: | Attendance records by date | `ComplianceService.GetAttendanceRange()` |
| Events | social:: | Social/co-op events | `SocialService.GetEventsForCalendar()` |

### §9.2 Aggregation Strategy

```go
func (s *PlanningServiceImpl) GetCalendar(
    ctx context.Context,
    auth *AuthContext,
    scope *FamilyScope,
    params CalendarQuery,
) (CalendarResponse, error) {
    // Fetch from all sources in parallel using errgroup
    g, ctx := errgroup.WithContext(ctx)

    var scheduleItems []ScheduleItem
    var activities []ActivitySummary
    var attendance []AttendanceRecord
    var events []EventSummary

    g.Go(func() error {
        var err error
        scheduleItems, err = s.scheduleRepo.ListByDateRange(
            ctx, scope, params.Start, params.End, params.StudentID,
        )
        return err
    })

    g.Go(func() error {
        var err error
        activities, err = s.learningService.ListActivitiesForCalendar(
            ctx, auth, scope, params.Start, params.End, params.StudentID,
        )
        return err
    })

    g.Go(func() error {
        var err error
        attendance, err = s.complianceService.GetAttendanceRange(
            ctx, auth, scope, params.Start, params.End, params.StudentID,
        )
        return err
    })

    g.Go(func() error {
        var err error
        events, err = s.socialService.GetEventsForCalendar(
            ctx, auth, scope, params.Start, params.End,
        )
        return err
    })

    if err := g.Wait(); err != nil {
        return CalendarResponse{}, err
    }

    // Merge into CalendarDay structs, sorted by time within each day
    days := s.mergeIntoCalendarDays(
        params.Start, params.End,
        scheduleItems, activities, attendance, events,
    )

    return CalendarResponse{
        Start: params.Start,
        End:   params.End,
        Days:  days,
    }, nil
}
```

### §9.3 Performance Consideration

Calendar queries fetch from 4 sources in parallel (`errgroup.WithContext`). For typical
date ranges (1 week = 7 days), this should complete well within the p99 < 500ms SLO.
For month views, each source returns at most ~30 days of data.

**CQRS applies**: Calendar reads are pure queries with no side effects. Schedule writes
are simple CRUD operations. The read path may be optimized with Redis caching in Phase 2
if needed.

**Revision trigger**: Add Redis cache for calendar queries if p99 latency exceeds 300ms
for weekly view.

---

## §10 Schedule Item Lifecycle (Domain Deep-Dive 2)

### §10.1 Create → Complete → Log Workflow

The primary workflow for schedule items:

1. **Create**: Parent creates a schedule item (e.g., "Math lesson, 9:00-10:00 AM")
2. **During the day**: Schedule item appears on the family calendar
3. **Complete**: Parent (or student in supervised mode) marks the item as completed
4. **Log as activity** (optional): Completed item can be logged as a learning activity

```
Create → [calendar display] → Complete → Log as Activity (optional)
                                    ↓
                              linked_activity_id set
```

### §10.2 Log as Activity

When a parent chooses to log a completed schedule item as a learning activity:

1. Service creates a `learn::` activity via `LearningService.LogActivity()`
2. The new activity's ID is linked back to the schedule item (`linked_activity_id`)
3. This prevents duplicate entries — the calendar shows the schedule item as "logged"
4. The activity inherits: title, date, duration, subject, and student from the schedule item

### §10.3 Event-Sourced Schedule Items

When a social event is added to the family calendar (e.g., a co-op day), a schedule item
is created with `linked_event_id` set. This ensures:
- The event appears in the schedule alongside family-created items
- Changes to the social event propagate to the schedule item (via event handler)
- The schedule item can be independently completed/logged

---

## §11 Recurring Schedules (Phase 2)

### §11.1 Recurrence Model

Recurrence uses the **iCalendar RRULE** format (RFC 5545) stored as a string:

```
FREQ=WEEKLY;BYDAY=MO,WE,FR         # Every Mon, Wed, Fri
FREQ=WEEKLY;BYDAY=TU,TH;UNTIL=20260601  # Every Tue, Thu until June 1
FREQ=DAILY;INTERVAL=2               # Every other day
```

### §11.2 Instance Generation

Recurring items are expanded into concrete instances on read (not pre-generated):
- Calendar queries expand RRULE for the requested date range
- Instances inherit all properties from the recurring item
- Individual instances can be modified (exception) or deleted (exclusion)
- RRULE parsing uses a Go rrule library (e.g., `github.com/teambition/rrule-go`)

### §11.3 Schedule Templates

Templates are named, reusable weekly patterns:

```json
{
  "name": "Standard School Week",
  "items": [
    { "day": "monday", "start_time": "09:00", "end_time": "10:00", "title": "Math", "category": "lesson" },
    { "day": "monday", "start_time": "10:15", "end_time": "11:00", "title": "Reading", "category": "reading" },
    { "day": "tuesday", "start_time": "09:00", "end_time": "12:00", "title": "Co-op Day", "category": "co_op" }
  ]
}
```

Applying a template to a date range creates concrete schedule items for each matching day.

---

## §12 Co-Op Day Coordination (Phase 2)

### §12.1 Integration with Social Events

Co-op days are social events owned by `social::`. The planning domain integrates them:

1. When a family RSVPs to a recurring group event, plan:: creates linked schedule items
2. Co-op day schedule items are tagged with `category = 'co_op'`
3. Changes to the social event (time change, cancellation) trigger updates to linked items

### §12.2 Schedule Sharing (Phase 2)

- Families MAY share their weekly schedule with friends (read-only)
- Sharing is opt-in per schedule or per week
- Shared schedules show time blocks and categories but NOT detailed descriptions (privacy)
- Co-op coordinators can view participating families' availability

---

## §13 Print & PDF Output (Domain Deep-Dive 3)

### §13.1 Print-Friendly Calendar View

`GET /v1/planning/calendar/print` returns an HTML page optimized for printing:

- Clean layout with date headers, time blocks, and category indicators
- No navigation, sidebars, or interactive elements
- Formatted for US Letter (8.5"×11") page width
- Includes: family name, date range, generation timestamp
- Grayscale-safe: categories distinguished by text labels and icons, not just color

### §13.2 PDF Export (Phase 2)

PDF generation via a server-side HTML-to-PDF renderer (e.g., `chromedp` headless Chrome,
`wkhtmltopdf` called as a subprocess, or a Go PDF library such as `jung-kurt/gofpdf`).
Enqueued as a background job for week/month ranges.

---

## §14 Data Export & Deletion Integration

plan:: implements `ExportHandler` and `DeletionHandler` for `lifecycle::` `[15-data-lifecycle §7]`:

### §14.1 Export

Exports schedule items and templates as JSON/CSV:
- `schedule-items.json` — all schedule items with metadata
- `schedule-templates.json` — all saved templates

### §14.2 Deletion

Deletes all `plan_schedule_items` and `plan_schedule_templates` for the family.
Linked activity IDs in `learn::` are NOT deleted (learn:: owns those records).

---

## §15 Events plan:: Publishes

| Event | Payload | Consumers |
|-------|---------|-----------|
| `ScheduleItemCompleted` | `{ family_id, item_id, student_id }` | Potential future use by comply:: for auto-attendance |
| `ScheduleItemLoggedAsActivity` | `{ family_id, item_id, activity_id }` | learn:: (for linking) |

---

## §16 Events plan:: Consumes

| Event | Source | Handler | Phase |
|-------|--------|---------|-------|
| `EventCreated` | social:: | Create linked schedule item if family has auto-add enabled | Phase 2 (co-op coordination) |
| `EventUpdated` | social:: | Update linked schedule item (time, location changes) | Phase 2 (co-op coordination) |
| `EventCancelled` | social:: | Delete linked schedule items | Phase 1 |
| `ActivityLogged` | learn:: | Mark corresponding schedule item as completed and link activity | Phase 1 |

---

## §17 Error Types

```go
import "errors"

var (
    // ErrItemNotFound indicates the schedule item was not found.
    ErrItemNotFound = errors.New("schedule item not found")

    // ErrTemplateNotFound indicates the schedule template was not found.
    ErrTemplateNotFound = errors.New("schedule template not found")

    // ErrInvalidDateRange indicates start must be before end.
    ErrInvalidDateRange = errors.New("invalid date range: start must be before end")

    // ErrDateRangeTooLarge indicates the date range exceeds 90 days.
    ErrDateRangeTooLarge = errors.New("date range too large (maximum 90 days)")

    // ErrAlreadyCompleted indicates the schedule item is already completed.
    ErrAlreadyCompleted = errors.New("schedule item already completed")

    // ErrAlreadyLogged indicates the schedule item is already logged as an activity.
    ErrAlreadyLogged = errors.New("schedule item already logged as activity")

    // ErrNotCompleted indicates the schedule item must be completed before logging.
    ErrNotCompleted = errors.New("schedule item must be completed before logging as activity")

    // ErrInvalidRecurrenceRule indicates the RRULE string is invalid.
    ErrInvalidRecurrenceRule = errors.New("invalid recurrence rule")

    // ErrStudentNotInFamily indicates the student does not belong to the family.
    ErrStudentNotInFamily = errors.New("student not found in family")
)
```

**HTTP mapping**:

| Error | HTTP Status |
|-------|-------------|
| `ErrItemNotFound` | 404 |
| `ErrTemplateNotFound` | 404 |
| `ErrInvalidDateRange` | 400 |
| `ErrDateRangeTooLarge` | 400 |
| `ErrAlreadyCompleted` | 409 Conflict |
| `ErrAlreadyLogged` | 409 Conflict |
| `ErrNotCompleted` | 409 Conflict |
| `ErrInvalidRecurrenceRule` | 400 |
| `ErrStudentNotInFamily` | 404 |

---

## §18 Cross-Domain Interactions

| Direction | Domain | Interaction |
|-----------|--------|-------------|
| plan:: → learn:: | Service call (read) | List activities for calendar aggregation |
| plan:: → learn:: | Service call (write) | Log schedule item as activity |
| plan:: → comply:: | Service call (read) | Get attendance for calendar aggregation |
| plan:: → social:: | Service call (read) | Get events for calendar aggregation |
| plan:: → iam:: | Service call (read) | Get student names for calendar display |
| social:: → plan:: | Domain event | Event created/updated/cancelled → update linked items |
| learn:: → plan:: | Domain event | Activity logged → mark schedule item completed |
| plan:: → notify:: | Domain event | Schedule reminders (Phase 2) |

---

## §19 Phase Scope

### Phase 1

- Calendar view aggregating schedule items + activities + events
- Day and week views
- Schedule item CRUD (create, read, update, delete, complete)
- Log completed schedule item as learning activity
- Print-friendly calendar HTML view
- Basic categories (lesson, reading, activity, assessment, field trip, co-op, break, custom)

### Phase 2

- Recurring schedule items (RRULE)
- Schedule templates (save and apply weekly patterns) — **implemented ahead of schedule in Phase 1**
- PDF export of calendar/schedule
- Co-op day coordination (linked social events)
- Schedule sharing with friends (read-only)
- Schedule reminders via notifications

---

## §20 Verification Checklist

- [ ] Calendar aggregation returns data from all 4 sources
- [ ] Schedule items are family-scoped (RLS enforced)
- [ ] Log-as-activity creates a valid learn:: activity and links back
- [ ] Calendar query for 1 week completes within 500ms (p99)
- [ ] Print view is clean, readable, and hides interactive elements
- [ ] Student filter works across all calendar sources
- [ ] Deletion of schedule items does NOT delete linked activities
- [ ] ExportHandler and DeletionHandler implemented for lifecycle:: integration

---

## §21 Module Structure

```
internal/plan/
├── handler.go          # Echo route handlers
├── service.go          # Calendar aggregation + schedule CRUD
├── repository.go       # plan_ table queries (GORM)
├── models.go           # DTOs (request/response) + GORM model definitions
├── ports.go            # Service + repository interface definitions
└── event_handlers.go   # Handlers for social:: and learn:: events
```
