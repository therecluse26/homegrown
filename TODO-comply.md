# TODO: Domain 14 — Compliance & Reporting (comply::)

93 TDD cycles, built bottom-up: pure domain → service layer → infrastructure.
Each checkbox is one unit of work. Work through them in order.

Reference files for patterns:
- Function-pointer stubs → `internal/recs/mock_test.go`
- FamilyScope in tests → `internal/onboard/service_test.go`
- Error assertions → `internal/recs/service_test.go`
- Event types → `internal/learn/events.go`
- Sentinel errors → `internal/recs/errors.go`
- AppError mapping → `internal/shared/error.go`
- Service constructor → `internal/recs/service.go`

---

## Phase 0: Scaffolding

Create all files so tests compile. No logic yet — just type definitions, interfaces, and stubs.

- [x] `internal/comply/domain/errors.go` — domain error sentinels
- [x] `internal/comply/errors.go` — top-level error sentinels + custom types
- [x] `internal/comply/domain/attendance.go` — pure attendance logic (stub signatures)
- [x] `internal/comply/domain/portfolio.go` — pure portfolio state machine (stub signatures)
- [x] `internal/comply/domain/gpa.go` — pure GPA calculation (stub signatures)
- [x] `internal/comply/domain/transcript.go` — transcript state machine (stub signatures)
- [x] `internal/comply/models.go` — all DTOs, request/response types, GORM models
- [x] `internal/comply/ports.go` — service + repository interfaces
- [x] `internal/comply/events.go` — domain events
- [x] `internal/comply/event_handlers.go` — event handler structs (stubs)
- [x] `internal/comply/mock_test.go` — function-pointer stubs for all repos + cross-domain mocks
- [x] `internal/comply/service.go` — ComplianceServiceImpl (constructor + empty methods)
- [x] `internal/comply/service_test.go` — test file with helper setup
- [x] Verify: `go build ./internal/comply/...` compiles with zero errors

---

## Group A: Pure Domain Logic (35 cycles)

No mocks, no DB — pure functions only. Fastest TDD cycles.

### Attendance Validation (`domain/attendance.go`)

- [x] **A1.** ValidateAttendanceRecord rejects future dates → ErrFutureAttendanceDate
- [x] **A2.** ValidateAttendanceRecord rejects invalid status strings → ErrInvalidAttendanceStatus
- [x] **A3.** ValidateAttendanceRecord requires duration_minutes for present_partial → ErrDurationRequiredForPartial
- [x] **A4.** ValidateAttendanceRecord rejects negative duration → ErrNegativeDuration
- [x] **A5.** ValidateAttendanceRecord accepts valid records (all 4 statuses)
- [x] **A6.** ShouldOverride: manual entry overrides auto-generated
- [x] **A7.** ShouldOverride: auto entry does NOT override manual

### Pace Calculation (`domain/attendance.go`)

- [x] **A8.** CalculatePace returns not_applicable when stateRequiredDays is nil
- [x] **A9.** CalculatePace returns on_track when school year hasn't started (elapsedSchoolDays=0)
- [x] **A10.** CalculatePace returns on_track when projected total ≥ required
- [x] **A11.** CalculatePace returns at_risk when projected total is within 90–100% of required
- [x] **A12.** CalculatePace returns behind when projected total < 90% of required

### School Day Counting (`domain/attendance.go`)

- [x] **A13.** CountSchoolDays counts Mon–Fri correctly for standard schedule
- [x] **A14.** CountSchoolDays respects 4-day week schedule
- [x] **A15.** CountSchoolDays excludes exclusion periods (e.g., winter break)
- [x] **A16.** CountSchoolDays returns 0 for empty date range

### Portfolio State Machine (`domain/portfolio.go`)

- [x] **A17.** ValidatePortfolioTransition: configuring → generating is valid
- [x] **A18.** ValidatePortfolioTransition: generating → ready is valid
- [x] **A19.** ValidatePortfolioTransition: generating → failed is valid
- [x] **A20.** ValidatePortfolioTransition: failed → generating (retry) is valid
- [x] **A21.** ValidatePortfolioTransition: ready → expired is valid
- [x] **A22.** ValidatePortfolioTransition: configuring → ready is INVALID
- [x] **A23.** ValidatePortfolioTransition: ready → configuring is INVALID
- [x] **A24.** ValidatePortfolioGenerate: rejects empty portfolio (0 items) → ErrEmptyPortfolio
- [x] **A25.** ValidatePortfolioGenerate: rejects non-configuring status → ErrPortfolioNotConfiguring
- [x] **A26.** ValidatePortfolioGenerate: rejects exceeded retries → ErrMaxRetriesExceeded
- [x] **A27.** ValidatePortfolioGenerate: allows failed portfolio with retries remaining

### GPA Calculation (`domain/gpa.go` — Phase 3)

- [x] **A28.** CalculateGPA: standard 4.0 unweighted with single course
- [x] **A29.** CalculateGPA: standard 4.0 unweighted with multiple courses + credits
- [x] **A30.** CalculateGPA: weighted GPA applies +0.5 for honors
- [x] **A31.** CalculateGPA: weighted GPA applies +1.0 for AP
- [x] **A32.** CalculateGPA: courses with nil grade_points are skipped
- [x] **A33.** CalculateGPA: zero courses returns 0.0 GPA
- [x] **A34.** CalculateGPA: mixed regular/honors/AP courses

### Transcript State Machine (`domain/transcript.go` — Phase 3)

- [x] **A35.** ValidateTranscriptTransition delegates to ValidatePortfolioTransition (same rules)

---

## Group B–C: Config & Schedules (17 cycles)

Service-layer tests with mock repos.

### B: Family Config & State Config (10 cycles)

- [x] **B1.** UpsertFamilyConfig: creates new config with valid state code → returns FamilyConfigResponse
- [x] **B2.** UpsertFamilyConfig: rejects invalid state code → ErrInvalidStateCode
- [x] **B3.** UpsertFamilyConfig: rejects invalid school year range (end ≤ start) → ErrInvalidSchoolYearRange
- [x] **B4.** UpsertFamilyConfig: updates existing config (upsert behavior)
- [x] **B5.** UpsertFamilyConfig: validates custom_schedule_id belongs to family if provided → ErrScheduleNotFound
- [x] **B6.** GetFamilyConfig: returns nil for families without config
- [x] **B7.** GetFamilyConfig: returns existing config
- [x] **B8.** ListStateConfigs: returns all cached state configs
- [x] **B9.** GetStateConfig: returns config for valid state code
- [x] **B10.** GetStateConfig: returns ErrStateConfigNotFound for unknown state

### C: Custom Schedules (7 cycles)

- [x] **C1.** CreateSchedule: creates schedule with valid 7-element school_days
- [x] **C2.** CreateSchedule: rejects school_days array with ≠ 7 elements → ErrInvalidSchoolDaysArray
- [x] **C3.** ListSchedules: returns family's schedules
- [x] **C4.** UpdateSchedule: updates existing schedule
- [x] **C5.** UpdateSchedule: returns ErrScheduleNotFound for non-existent or wrong family
- [x] **C6.** DeleteSchedule: deletes schedule not in use
- [x] **C7.** DeleteSchedule: rejects deletion of schedule in use by family config → ErrScheduleInUse

---

## Group D: Attendance Service (14 cycles)

Needs mock repos + state config from Groups B–C.

- [x] **D1.** RecordAttendance: creates manual record with manual_override=true, is_auto=false
- [x] **D2.** RecordAttendance: upserts over auto-generated record for same date (manual wins)
- [x] **D3.** RecordAttendance: rejects future date → ErrFutureAttendanceDate
- [x] **D4.** RecordAttendance: rejects present_partial without duration → ErrDurationRequiredForPartial
- [x] **D5.** RecordAttendance: validates student belongs to family → ErrStudentNotInFamily
- [x] **D6.** BulkRecordAttendance: creates up to 31 records
- [x] **D7.** BulkRecordAttendance: rejects > 31 records → ErrBulkAttendanceLimitExceeded
- [x] **D8.** BulkRecordAttendance: validates each record individually
- [x] **D9.** UpdateAttendance: updates existing record
- [x] **D10.** UpdateAttendance: returns ErrAttendanceNotFound for non-existent
- [x] **D11.** DeleteAttendance: deletes existing record
- [x] **D12.** ListAttendance: returns records in date range for student
- [x] **D13.** GetAttendanceSummary: returns correct day counts by status
- [x] **D14.** GetAttendanceSummary: includes pace calculation (on_track/at_risk/behind)

---

## Group E–F: Assessments & Tests (11 cycles)

Straightforward CRUD.

### E: Assessments (6 cycles)

- [x] **E1.** CreateAssessment: creates record with valid assessment type
- [x] **E2.** CreateAssessment: validates student belongs to family
- [x] **E3.** ListAssessments: filters by subject and date range
- [x] **E4.** UpdateAssessment: updates existing record
- [x] **E5.** UpdateAssessment: returns ErrAssessmentNotFound
- [x] **E6.** DeleteAssessment: deletes record

### F: Standardized Tests (5 cycles)

- [x] **F1.** CreateTestScore: stores JSONB scores correctly
- [x] **F2.** CreateTestScore: validates student belongs to family
- [x] **F3.** ListTestScores: returns scores sorted by date descending
- [x] **F4.** UpdateTestScore: updates existing record
- [x] **F5.** DeleteTestScore: deletes record

---

## Group G: Portfolios (14 cycles)

Complex state machine + cross-domain mocks (learn:: for source items).

- [x] **G1.** CreatePortfolio: creates in configuring status
- [x] **G2.** CreatePortfolio: rejects date_range_end ≤ date_range_start
- [x] **G3.** AddPortfolioItems: caches display data from learn:: at selection time
- [x] **G4.** AddPortfolioItems: rejects if portfolio not in configuring status → ErrPortfolioNotConfiguring
- [x] **G5.** AddPortfolioItems: rejects if source item not found in learn:: → ErrPortfolioItemSourceNotFound
- [x] **G6.** GeneratePortfolio: transitions configuring → generating
- [x] **G7.** GeneratePortfolio: rejects empty portfolio → ErrEmptyPortfolio
- [x] **G8.** GeneratePortfolio: rejects non-configuring portfolio → ErrPortfolioNotConfiguring
- [x] **G9.** GetPortfolio: returns portfolio with items
- [x] **G10.** GetPortfolio: returns ErrPortfolioNotFound
- [x] **G11.** ListPortfolios: returns portfolios for student
- [x] **G12.** GetPortfolioDownloadURL: returns presigned URL for ready portfolio
- [x] **G13.** GetPortfolioDownloadURL: returns ErrPortfolioNotConfiguring for non-ready portfolio
- [x] **G14.** GetPortfolioDownloadURL: returns ErrPortfolioExpired for expired portfolio

---

## Group H: Event Handlers (5 cycles)

Cross-domain event types (learn::, iam::, billing::).

- [x] **H1.** HandleActivityLogged: creates auto-attendance (is_auto=true) when no record exists
- [x] **H2.** HandleActivityLogged: does NOT override existing manual record for same date
- [x] **H3.** HandleStudentDeleted: cascades deletion of all student compliance data
- [x] **H4.** HandleFamilyDeletionScheduled: cascades deletion of all family compliance data
- [x] **H5.** HandleSubscriptionCancelled: preserves data, no deletion

---

## Group I: Dashboard (2 cycles)

Aggregation query across attendance, assessments, portfolios.

- [x] **I1.** GetDashboard: returns null family_config when unconfigured
- [x] **I2.** GetDashboard: returns student summaries with attendance, assessment counts, portfolios

---

## Group J–L: Phase 3 — Transcripts, Courses, GPA (15 cycles)

### J: Transcripts (7 cycles)

- [x] **J1.** CreateTranscript: creates in configuring status
- [x] **J2.** GenerateTranscript: transitions configuring → generating
- [x] **J3.** GenerateTranscript: rejects non-configuring transcript
- [x] **J4.** GetTranscript: returns transcript with courses and GPA
- [x] **J5.** ListTranscripts: returns transcripts for student
- [x] **J6.** DeleteTranscript: deletes transcript
- [x] **J7.** GetTranscriptDownloadURL: returns presigned URL for ready transcript

### K: Courses (5 cycles)

- [x] **K1.** CreateCourse: creates course with valid level
- [x] **K2.** CreateCourse: validates student belongs to family
- [x] **K3.** ListCourses: filters by grade_level, school_year
- [x] **K4.** UpdateCourse: updates existing course
- [x] **K5.** DeleteCourse: deletes course

### L: GPA Service (3 cycles)

- [x] **L1.** CalculateGPA service: returns GPA from courses with by_grade_level breakdown
- [x] **L2.** CalculateGPAWhatIf: projects GPA with hypothetical courses
- [x] **L3.** GetGPAHistory: returns GPA by school year/term

---

## Infrastructure

Build after all TDD cycles are green.

### Repository Implementations

- [x] `internal/comply/repository.go` — PostgreSQL implementations for all repository interfaces

### Handler

- [x] `internal/comply/handler.go` — Echo HTTP handlers with swag annotations
- [x] `internal/comply/jobs.go` — background jobs (portfolio generation, expiry)

### Migrations

- [x] Migration: create comply_state_configs + comply_attendance tables (Phase 1)
- [x] Migration: create comply_family_configs + comply_custom_schedules tables (Phase 2)
- [x] Migration: create comply_assessments + comply_standardized_tests tables (Phase 2)
- [x] Migration: create comply_portfolios + comply_portfolio_items tables (Phase 2)
- [x] Migration: create comply_transcripts + comply_courses tables (Phase 3)

### Wiring & Integration

- [x] Wire ComplianceService in `main.go` (constructor + dependency injection)
- [x] Register comply routes in `app.go`
- [x] Register event handlers (ActivityLogged, StudentDeleted, FamilyDeletion, SubscriptionCancelled)
- [x] `make full-generate` — regenerate OpenAPI + TypeScript types

### Quality Gates

- [x] `go test ./internal/comply/...` — all tests pass
- [x] `golangci-lint run ./internal/comply/...` — zero warnings
- [x] `make check` — full quality gate green
- [x] Update `specs/domains/14-comply.md` if any spec decisions changed during implementation
