# Domain Spec 17 — Planning & Scheduling (plan::)

## §1 Overview

The Planning domain provides a **unified calendar and scheduling interface** for homeschool
families. It synthesizes data from Learning (activities), Compliance (attendance), and Social
(events) into a single calendar view, and provides schedule creation tools for daily and
weekly planning. Homeschool families plan their days around learning activities, co-op
meetings, and social events — this domain serves that core workflow. `[S§17, ADR-013]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `src/plan/` |
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
-- Migration: YYYYMMDD_000001_create_plan_tables.rs
-- =============================================================================

-- Schedule items: family-created calendar entries
-- These are plan::-owned data, NOT duplicates of learning activities or events
CREATE TABLE plan_schedule_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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

```rust
#[async_trait]
pub trait PlanningService: Send + Sync {
    // === Calendar View (Read) ===

    /// Get aggregated calendar for a date range.
    /// Combines: plan:: schedule items + learn:: activities + comply:: attendance
    ///           + social:: events.
    async fn get_calendar(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        params: CalendarQuery,
    ) -> Result<CalendarResponse, AppError>;

    /// Get detailed day view with all items, activities, and events.
    async fn get_day_view(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        date: NaiveDate,
        student_id: Option<Uuid>,
    ) -> Result<DayViewResponse, AppError>;

    // === Schedule Items (Write) ===

    /// Create a new schedule item.
    async fn create_schedule_item(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        input: CreateScheduleItemInput,
    ) -> Result<ScheduleItemId, AppError>;

    /// Update a schedule item.
    async fn update_schedule_item(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        item_id: Uuid,
        input: UpdateScheduleItemInput,
    ) -> Result<(), AppError>;

    /// Delete a schedule item.
    async fn delete_schedule_item(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        item_id: Uuid,
    ) -> Result<(), AppError>;

    /// Mark a schedule item as completed.
    async fn complete_schedule_item(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        item_id: Uuid,
    ) -> Result<(), AppError>;

    /// Log a completed schedule item as a learning activity.
    /// Creates an activity in learn:: and links it back to this schedule item.
    async fn log_as_activity(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        item_id: Uuid,
        input: LogAsActivityInput,
    ) -> Result<Uuid, AppError>; // returns the created activity ID

    /// List schedule items with filters.
    async fn list_schedule_items(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        params: ScheduleItemQuery,
        pagination: PaginationParams,
    ) -> Result<PaginatedResponse<ScheduleItemResponse>, AppError>;

    // === Print/Export ===

    /// Generate a print-friendly HTML view of the calendar.
    async fn get_print_view(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        start: NaiveDate,
        end: NaiveDate,
        student_id: Option<Uuid>,
    ) -> Result<String, AppError>; // returns HTML string
}
```

---

## §6 Repository Interfaces

```rust
#[async_trait]
pub trait ScheduleItemRepository: Send + Sync {
    async fn create(
        &self,
        scope: &FamilyScope,
        input: &CreateScheduleItem,
    ) -> Result<ScheduleItem, DbErr>;

    async fn find_by_id(
        &self,
        scope: &FamilyScope,
        id: Uuid,
    ) -> Result<Option<ScheduleItem>, DbErr>;

    async fn list_by_date_range(
        &self,
        scope: &FamilyScope,
        start: NaiveDate,
        end: NaiveDate,
        student_id: Option<Uuid>,
    ) -> Result<Vec<ScheduleItem>, DbErr>;

    async fn list_filtered(
        &self,
        scope: &FamilyScope,
        query: &ScheduleItemQuery,
        pagination: &PaginationParams,
    ) -> Result<Vec<ScheduleItem>, DbErr>;

    async fn update(
        &self,
        scope: &FamilyScope,
        id: Uuid,
        input: &UpdateScheduleItem,
    ) -> Result<(), DbErr>;

    async fn mark_completed(
        &self,
        scope: &FamilyScope,
        id: Uuid,
    ) -> Result<(), DbErr>;

    async fn set_linked_activity(
        &self,
        scope: &FamilyScope,
        id: Uuid,
        activity_id: Uuid,
    ) -> Result<(), DbErr>;

    async fn delete(
        &self,
        scope: &FamilyScope,
        id: Uuid,
    ) -> Result<(), DbErr>;
}

#[async_trait]
pub trait ScheduleTemplateRepository: Send + Sync {
    async fn create(
        &self,
        scope: &FamilyScope,
        input: &CreateScheduleTemplate,
    ) -> Result<ScheduleTemplate, DbErr>;

    async fn list_by_family(
        &self,
        scope: &FamilyScope,
    ) -> Result<Vec<ScheduleTemplate>, DbErr>;

    async fn find_by_id(
        &self,
        scope: &FamilyScope,
        id: Uuid,
    ) -> Result<Option<ScheduleTemplate>, DbErr>;

    async fn update(
        &self,
        scope: &FamilyScope,
        id: Uuid,
        input: &UpdateScheduleTemplate,
    ) -> Result<(), DbErr>;

    async fn delete(
        &self,
        scope: &FamilyScope,
        id: Uuid,
    ) -> Result<(), DbErr>;
}
```

---

## §7 Models (DTOs)

```rust
// --- Request types ---

#[derive(Deserialize, ToSchema)]
pub struct CalendarQuery {
    pub start: NaiveDate,
    pub end: NaiveDate,
    pub student_id: Option<Uuid>,
    /// Which sources to include (default: all)
    pub sources: Option<Vec<CalendarSource>>,
}

#[derive(Deserialize, ToSchema)]
#[serde(rename_all = "snake_case")]
pub enum CalendarSource {
    Schedule,      // plan:: schedule items
    Activities,    // learn:: logged activities
    Attendance,    // comply:: attendance records
    Events,        // social:: events
}

#[derive(Deserialize, ToSchema)]
pub struct CreateScheduleItemInput {
    pub title: String,
    pub description: Option<String>,
    pub student_id: Option<Uuid>,
    pub start_date: NaiveDate,
    pub start_time: Option<NaiveTime>,
    pub end_time: Option<NaiveTime>,
    pub duration_minutes: Option<i32>,
    pub category: Option<ScheduleCategory>,
    pub subject_id: Option<Uuid>,
    pub color: Option<String>,
    pub notes: Option<String>,
}

#[derive(Deserialize, ToSchema)]
pub struct UpdateScheduleItemInput {
    pub title: Option<String>,
    pub description: Option<String>,
    pub student_id: Option<Option<Uuid>>,
    pub start_date: Option<NaiveDate>,
    pub start_time: Option<Option<NaiveTime>>,
    pub end_time: Option<Option<NaiveTime>>,
    pub duration_minutes: Option<Option<i32>>,
    pub category: Option<ScheduleCategory>,
    pub subject_id: Option<Option<Uuid>>,
    pub color: Option<Option<String>>,
    pub notes: Option<Option<String>>,
}

#[derive(Deserialize, ToSchema)]
pub struct LogAsActivityInput {
    /// Additional details for the activity log entry
    pub description: Option<String>,
    pub tags: Option<Vec<String>>,
}

#[derive(Deserialize, ToSchema)]
pub struct ScheduleItemQuery {
    pub start_date: Option<NaiveDate>,
    pub end_date: Option<NaiveDate>,
    pub student_id: Option<Uuid>,
    pub category: Option<ScheduleCategory>,
    pub is_completed: Option<bool>,
}

// --- Response types ---

#[derive(Serialize, ToSchema)]
pub struct CalendarResponse {
    pub start: NaiveDate,
    pub end: NaiveDate,
    pub days: Vec<CalendarDay>,
}

#[derive(Serialize, ToSchema)]
pub struct CalendarDay {
    pub date: NaiveDate,
    pub items: Vec<CalendarItem>,
}

#[derive(Serialize, ToSchema)]
pub struct CalendarItem {
    pub id: Uuid,
    pub source: CalendarSource,
    pub title: String,
    pub start_time: Option<NaiveTime>,
    pub end_time: Option<NaiveTime>,
    pub duration_minutes: Option<i32>,
    pub category: Option<String>,
    pub color: Option<String>,
    pub student_id: Option<Uuid>,
    pub student_name: Option<String>,
    pub is_completed: Option<bool>,
    /// Source-specific details
    pub details: CalendarItemDetails,
}

#[derive(Serialize, ToSchema)]
#[serde(tag = "type")]
pub enum CalendarItemDetails {
    Schedule {
        description: Option<String>,
        notes: Option<String>,
        linked_activity_id: Option<Uuid>,
    },
    Activity {
        subject: Option<String>,
        tags: Vec<String>,
    },
    Attendance {
        status: String,  // "present", "absent", "holiday"
    },
    Event {
        group_name: Option<String>,
        location: Option<String>,
        rsvp_status: Option<String>,
    },
}

#[derive(Serialize, ToSchema)]
pub struct DayViewResponse {
    pub date: NaiveDate,
    pub schedule_items: Vec<ScheduleItemResponse>,
    pub activities: Vec<ActivitySummary>,
    pub attendance: Option<AttendanceSummary>,
    pub events: Vec<EventSummary>,
}

#[derive(Serialize, ToSchema)]
pub struct ScheduleItemResponse {
    pub id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub student_id: Option<Uuid>,
    pub student_name: Option<String>,
    pub start_date: NaiveDate,
    pub start_time: Option<NaiveTime>,
    pub end_time: Option<NaiveTime>,
    pub duration_minutes: Option<i32>,
    pub category: ScheduleCategory,
    pub subject_id: Option<Uuid>,
    pub subject_name: Option<String>,
    pub color: Option<String>,
    pub is_completed: bool,
    pub completed_at: Option<DateTime<Utc>>,
    pub linked_activity_id: Option<Uuid>,
    pub notes: Option<String>,
    pub created_at: DateTime<Utc>,
}

// --- Enums ---

#[derive(Serialize, Deserialize, ToSchema)]
#[serde(rename_all = "snake_case")]
pub enum ScheduleCategory {
    Lesson,
    Reading,
    Activity,
    Assessment,
    FieldTrip,
    CoOp,
    Break,
    Custom,
}
```

---

## §8 Adapter Interfaces

plan:: does NOT have external adapters. It reads from other domains via their service
interfaces:

```rust
// Dependencies injected into PlanningServiceImpl:
pub struct PlanningServiceImpl {
    schedule_repo: Arc<dyn ScheduleItemRepository>,
    template_repo: Arc<dyn ScheduleTemplateRepository>,
    // Read-only dependencies on other domains:
    learning_service: Arc<dyn LearningService>,
    compliance_service: Arc<dyn ComplianceService>,
    social_service: Arc<dyn SocialService>,
    iam_service: Arc<dyn IamService>,
    event_bus: Arc<EventBus>,
}
```

---

## §9 Calendar Aggregation (Domain Deep-Dive 1)

### §9.1 Data Sources

The calendar view aggregates four data sources:

| Source | Domain | Data | Read Method |
|--------|--------|------|-------------|
| Schedule items | plan:: | Family-created calendar entries | Direct DB query |
| Activities | learn:: | Logged learning activities with dates | `LearningService::list_activities()` |
| Attendance | comply:: | Attendance records by date | `ComplianceService::get_attendance()` |
| Events | social:: | Social/co-op events | `SocialService::get_events()` |

### §9.2 Aggregation Strategy

```rust
impl PlanningServiceImpl {
    async fn get_calendar(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        params: CalendarQuery,
    ) -> Result<CalendarResponse, AppError> {
        // Fetch from all sources in parallel
        let (schedule_items, activities, attendance, events) = tokio::try_join!(
            self.schedule_repo.list_by_date_range(
                scope, params.start, params.end, params.student_id
            ),
            self.learning_service.list_activities_for_calendar(
                auth, scope, params.start, params.end, params.student_id
            ),
            self.compliance_service.get_attendance_range(
                auth, scope, params.start, params.end, params.student_id
            ),
            self.social_service.get_events_for_calendar(
                auth, scope, params.start, params.end
            ),
        )?;

        // Merge into CalendarDay structs, sorted by time within each day
        let days = self.merge_into_calendar_days(
            params.start, params.end,
            schedule_items, activities, attendance, events,
        );

        Ok(CalendarResponse {
            start: params.start,
            end: params.end,
            days,
        })
    }
}
```

### §9.3 Performance Consideration

Calendar queries fetch from 4 sources in parallel (`tokio::try_join!`). For typical
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

1. Service creates a `learn::` activity via `LearningService::log_activity()`
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
- RRULE parsing uses the `rrule` Rust crate

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

PDF generation via a server-side HTML-to-PDF renderer (e.g., `weasyprint` or `wkhtmltopdf`
called as a subprocess, or a Rust PDF library). Enqueued as a background job for
week/month ranges.

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

| Event | Source | Handler |
|-------|--------|---------|
| `EventCreated` | social:: | Create linked schedule item if family has auto-add enabled |
| `EventUpdated` | social:: | Update linked schedule item (time, location changes) |
| `EventCancelled` | social:: | Mark linked schedule item as cancelled |
| `ActivityLogged` | learn:: | Optional: mark corresponding schedule item as completed |

---

## §17 Error Types

```rust
#[derive(Debug, thiserror::Error)]
pub enum PlanningError {
    #[error("Schedule item not found")]
    ItemNotFound,

    #[error("Schedule template not found")]
    TemplateNotFound,

    #[error("Invalid date range: start must be before end")]
    InvalidDateRange,

    #[error("Date range too large (maximum 90 days)")]
    DateRangeTooLarge,

    #[error("Schedule item already completed")]
    AlreadyCompleted,

    #[error("Schedule item already logged as activity")]
    AlreadyLogged,

    #[error("Invalid recurrence rule")]
    InvalidRecurrenceRule,

    #[error("Student not found in family")]
    StudentNotInFamily,

    #[error("Database error")]
    Database(#[from] sea_orm::DbErr),
}
```

**HTTP mapping**:

| Error | HTTP Status |
|-------|-------------|
| `ItemNotFound` | 404 |
| `TemplateNotFound` | 404 |
| `InvalidDateRange` | 400 |
| `DateRangeTooLarge` | 400 |
| `AlreadyCompleted` | 409 Conflict |
| `AlreadyLogged` | 409 Conflict |
| `InvalidRecurrenceRule` | 400 |
| `StudentNotInFamily` | 404 |

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
- Schedule templates (save and apply weekly patterns)
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
src/plan/
├── mod.rs              # Re-exports
├── handlers.rs         # Axum route handlers
├── service.rs          # Calendar aggregation + schedule CRUD
├── repository.rs       # plan_ table queries
├── models.rs           # DTOs (request/response)
├── ports.rs            # Service + repository trait definitions
├── event_handlers.rs   # Handlers for social:: and learn:: events
└── entities/           # SeaORM-generated (plan_ tables)
```
