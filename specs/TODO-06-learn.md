# TODO: learn:: Domain Implementation (Phase 1)

## Batch 0: Database Migration + Taxonomy Seed
- [x] `migrations/20260323000011_create_learn_tables.sql` — All 22 tables, indexes, RLS policies (spec §3.2 + §3.3)
- [x] `migrations/20260323000012_seed_learn_subject_taxonomy.sql` — 3-level subject taxonomy seed
- [x] Verify: `go test ./...` still passes

## Batch 1: Foundation Types
- [x] `internal/learn/domain/errors.go` — ~30 sentinel errors + structured types (spec §17)
- [x] `internal/learn/models.go` — GORM models (22 tables), request/response/internal/query types (spec §8)
- [x] `internal/learn/ports.go` — `LearningService` (~52 methods), 16 repo interfaces, `MediaAdapter`, consumer interfaces
- [x] `internal/learn/errors.go` — `LearningError` wrapper + `mapLearningError()` (spec §17.1)
- [x] `internal/learn/events.go` — 8 domain events (spec §18.3)
- [x] `internal/learn/iam_adapter.go` — function-based adapter for `IamServiceForLearn`
- [x] `internal/learn/method_adapter.go` — function-based adapter for `MethodServiceForLearn`
- [x] `internal/learn/mkt_adapter.go` — stub adapter for `MktServiceForLearn`
- [x] Verify: `go build ./internal/learn/...` compiles

## Batch 2: Activity Logging (10 endpoints)
- [x] `internal/learn/domain/activity.go` — aggregate root: duration ≥ 0, date not in future
- [x] `internal/learn/repository.go` — `PgActivityDefRepository`, `PgActivityLogRepository`, partial `PgSubjectTaxonomyRepository`
- [x] `internal/learn/service.go` — constructor + activity def/log CRUD + `ActivityLogged` event + validation helpers
- [x] `internal/learn/handler.go` — Handler struct + `Register()` + activity def (5) + activity log (5) handlers
- [x] `internal/learn/handler_test.go` — mock service + test setup + activity endpoint tests
- [x] `internal/learn/service_test.go` — activity invariant tests
- [x] Verify: `go test ./internal/learn/...` passes

## Batch 3: Reading (10 endpoints)
- [x] `internal/learn/domain/reading_list.go` — status transitions: `to_read` → `in_progress` → `completed`
- [x] Extend `repository.go` — `PgReadingItemRepository`, `PgReadingProgressRepository`, `PgReadingListRepository`
- [x] Extend `service.go` — reading item CRUD, progress tracking + BookCompleted event, reading list management
- [x] Extend `handler.go` — reading items (3) + progress (3) + lists (5) handlers
- [x] Extend tests — status transitions, duplicate tracking, BookCompleted event
- [x] Verify: `go test ./internal/learn/...` passes

## Batch 4: Journal Entries + Subject Taxonomy (7 endpoints)
- [x] `internal/learn/domain/journal.go` — entry type validation (freeform/narration/reflection)
- [x] `internal/learn/domain/taxonomy.go` — slug generation, hierarchy checks
- [x] Extend `repository.go` — `PgJournalEntryRepository`, complete `PgSubjectTaxonomyRepository`
- [x] Extend `service.go` — journal CRUD, GetSubjectTaxonomy, CreateCustomSubject
- [x] Extend `handler.go` — journal (5) + taxonomy (2) handlers
- [x] Extend tests — entry type validation, custom subject uniqueness, taxonomy merge
- [x] Verify: `go test ./internal/learn/...` passes

## Batch 5: Progress + Artifact Links + Export + Tools (10 endpoints)
- [x] `internal/learn/domain/progress.go` — metric aggregation logic
- [x] `internal/learn/export.go` — ExportGenerationTask (asynq background worker)
- [x] Extend `repository.go` — `PgArtifactLinkRepository`, `PgProgressRepository`, `PgExportRepository`
- [x] Extend `service.go` — progress queries, artifact links, export (async), tools (delegate to method::)
- [x] Extend `handler.go` — progress (3) + links (3) + export (2) + tools (2) handlers
- [x] Extend tests — progress accuracy, link uniqueness, concurrent export blocking, tool delegation
- [x] Verify: `go test ./internal/learn/...` passes

## Batch 6: Assessment Engine (10 endpoints)
- [x] `internal/learn/domain/quiz_session.go` — session lifecycle, auto-scoring, parent scoring
- [x] Extend `repository.go` — `PgQuestionRepository`, `PgQuizDefRepository`, `PgQuizSessionRepository`
- [x] Extend `service.go` — question CRUD, quiz builder, quiz sessions, QuizCompleted event
- [x] Extend `handler.go` — questions (3) + quiz defs (3) + quiz sessions (4) handlers
- [x] Extend tests — answer_data validation, auto-scoring, session transitions, parent scoring
- [x] Verify: `go test ./internal/learn/...` passes

## Batch 7: Sequence Engine + Student Assignments (10 endpoints)
- [x] `internal/learn/domain/sequence.go` — linear vs recommended-order, unlock logic
- [x] `internal/learn/domain/assignment.go` — status transitions, content validation
- [x] Extend `repository.go` — `PgSequenceDefRepository`, `PgSequenceProgressRepository`, `PgAssignmentRepository`
- [x] Extend `service.go` — sequence CRUD + progress, assignment CRUD, events
- [x] Extend `handler.go` — sequence defs (3) + progress (3) + assignments (4) handlers
- [x] Extend tests — linear enforcement, parent override, events, parent-only auth
- [x] Verify: `go test ./internal/learn/...` passes

## Batch 8: Video Progress (5 endpoints)
- [x] Extend `repository.go` — `PgVideoDefRepository`, `PgVideoProgressRepository`
- [x] Extend `service.go` — video def browsing, video progress tracking
- [x] Extend `handler.go` — video defs (3) + video progress (2) handlers
- [x] Extend tests
- [x] Verify: `go test ./internal/learn/...` passes

## Batch 9: Event Handlers + Wiring + Integration
- [x] `internal/learn/event_handlers.go` — StudentCreatedHandler, StudentDeletedHandler, 3 deferred
- [x] Extend `cmd/server/main.go` — Step 7f: repos, adapters, service, event subscriptions
- [x] Extend `internal/app/app.go` — `Learn learn.LearningService` + route registration
- [x] Verify: `golangci-lint run` zero warnings
- [x] Verify: `go test ./...` all pass
- [x] Verify: server starts with learn:: wired
