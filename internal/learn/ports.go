package learn

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface — all learning domain use cases. [06-learn §5]
// CQRS: commands modify state; queries are read-only. [ARCH §4.7]
// ═══════════════════════════════════════════════════════════════════════════════

// LearningService defines all learning domain use cases.
type LearningService interface {
	// ─── Definition Commands (Layer 1 — publisher-based access) ──────────

	// CreateActivityDef creates an activity definition. Publisher membership required.
	CreateActivityDef(ctx context.Context, cmd CreateActivityDefCommand) (ActivityDefResponse, error)
	// UpdateActivityDef updates an activity definition. Publisher membership required.
	UpdateActivityDef(ctx context.Context, defID uuid.UUID, cmd UpdateActivityDefCommand) (ActivityDefResponse, error)
	// DeleteActivityDef soft-deletes an activity definition. Publisher membership required.
	DeleteActivityDef(ctx context.Context, defID uuid.UUID, callerID uuid.UUID) error

	// CreateReadingItem creates a reading item definition. Publisher membership required.
	CreateReadingItem(ctx context.Context, cmd CreateReadingItemCommand) (ReadingItemResponse, error)
	// UpdateReadingItem updates a reading item. Publisher membership required.
	UpdateReadingItem(ctx context.Context, itemID uuid.UUID, cmd UpdateReadingItemCommand) (ReadingItemResponse, error)

	// LinkArtifacts creates an artifact link between two published content items.
	LinkArtifacts(ctx context.Context, cmd CreateArtifactLinkCommand) (ArtifactLinkResponse, error)
	// UnlinkArtifacts removes an artifact link. Must own source content.
	UnlinkArtifacts(ctx context.Context, linkID uuid.UUID, callerID uuid.UUID) error

	// ─── Instance Commands (Layer 3 — FamilyScope required) ─────────────

	// LogActivity logs an activity for a student.
	LogActivity(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd LogActivityCommand) (ActivityLogResponse, error)
	// UpdateActivityLog updates an activity log entry.
	UpdateActivityLog(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID, cmd UpdateActivityLogCommand) (ActivityLogResponse, error)
	// DeleteActivityLog deletes an activity log entry.
	DeleteActivityLog(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID) error

	// CreateJournalEntry creates a journal entry for a student.
	CreateJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd CreateJournalEntryCommand) (JournalEntryResponse, error)
	// UpdateJournalEntry updates a journal entry.
	UpdateJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID, cmd UpdateJournalEntryCommand) (JournalEntryResponse, error)
	// DeleteJournalEntry deletes a journal entry.
	DeleteJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID) error

	// StartReading starts tracking a reading item for a student.
	StartReading(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartReadingCommand) (ReadingProgressResponse, error)
	// UpdateReadingProgress updates reading progress. Completing triggers BookCompleted event.
	UpdateReadingProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateReadingProgressCommand) (ReadingProgressResponse, error)

	// CreateReadingList creates a named reading list.
	CreateReadingList(ctx context.Context, scope *shared.FamilyScope, cmd CreateReadingListCommand) (ReadingListResponse, error)
	// UpdateReadingList updates a reading list (metadata and items).
	UpdateReadingList(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID, cmd UpdateReadingListCommand) (ReadingListResponse, error)
	// DeleteReadingList deletes a reading list.
	DeleteReadingList(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) error

	// CreateCustomSubject creates a family-scoped custom subject.
	CreateCustomSubject(ctx context.Context, scope *shared.FamilyScope, cmd CreateCustomSubjectCommand) (CustomSubjectResponse, error)

	// RequestDataExport requests an async data export.
	RequestDataExport(ctx context.Context, scope *shared.FamilyScope, cmd RequestExportCommand) (ExportRequestResponse, error)

	// ─── Assessment Engine Commands (Layer 1 + Layer 3) ─────────────────

	// CreateQuestion creates a question. Publisher membership required.
	CreateQuestion(ctx context.Context, cmd CreateQuestionCommand) (QuestionResponse, error)
	// UpdateQuestion updates a question. Publisher membership required.
	UpdateQuestion(ctx context.Context, questionID uuid.UUID, cmd UpdateQuestionCommand) (QuestionResponse, error)

	// CreateQuizDef creates a quiz definition from questions. Publisher membership required.
	CreateQuizDef(ctx context.Context, cmd CreateQuizDefCommand) (QuizDefResponse, error)
	// UpdateQuizDef updates a quiz definition. Publisher membership required.
	UpdateQuizDef(ctx context.Context, quizDefID uuid.UUID, cmd UpdateQuizDefCommand) (QuizDefResponse, error)

	// StartQuizSession starts a quiz session for a student.
	StartQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartQuizSessionCommand) (QuizSessionResponse, error)
	// UpdateQuizSession saves progress or submits a quiz session.
	UpdateQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd UpdateQuizSessionCommand) (QuizSessionResponse, error)
	// ScoreQuizSession allows a parent to score short-answer questions on a submitted quiz.
	ScoreQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd ScoreQuizCommand) (QuizSessionResponse, error)

	// ─── Sequence Engine Commands (Layer 1 + Layer 3) ───────────────────

	// CreateSequenceDef creates a lesson sequence. Publisher membership required.
	CreateSequenceDef(ctx context.Context, cmd CreateSequenceDefCommand) (SequenceDefResponse, error)
	// UpdateSequenceDef updates a sequence definition. Publisher membership required.
	UpdateSequenceDef(ctx context.Context, sequenceDefID uuid.UUID, cmd UpdateSequenceDefCommand) (SequenceDefResponse, error)

	// StartSequence starts a sequence for a student.
	StartSequence(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartSequenceCommand) (SequenceProgressResponse, error)
	// UpdateSequenceProgress advances, skips, or unlocks items in a sequence.
	UpdateSequenceProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateSequenceProgressCommand) (SequenceProgressResponse, error)

	// ─── Assignment Commands (Layer 3 — parent auth required) ───────────

	// CreateAssignment assigns content to a student.
	CreateAssignment(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd CreateAssignmentCommand) (AssignmentResponse, error)
	// UpdateAssignment updates assignment status.
	UpdateAssignment(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID, cmd UpdateAssignmentCommand) (AssignmentResponse, error)
	// DeleteAssignment removes an assignment.
	DeleteAssignment(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID) error

	// ─── Video Commands (Layer 3) ──────────────────────────────────────

	// UpdateVideoProgress updates watch position/completion for a video.
	UpdateVideoProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd UpdateVideoProgressCommand) (VideoProgressResponse, error)

	// ─── Definition Queries (Layer 1 — no FamilyScope) ──────────────────

	// ListActivityDefs lists activity definitions with filtering.
	ListActivityDefs(ctx context.Context, query ActivityDefQuery) (PaginatedResponse[ActivityDefSummaryResponse], error)
	// GetActivityDef returns a single activity definition.
	GetActivityDef(ctx context.Context, defID uuid.UUID) (ActivityDefResponse, error)

	// ListReadingItems lists reading items with filtering.
	ListReadingItems(ctx context.Context, query ReadingItemQuery) (PaginatedResponse[ReadingItemSummaryResponse], error)
	// GetReadingItem returns a single reading item with linked artifacts.
	GetReadingItem(ctx context.Context, itemID uuid.UUID) (ReadingItemDetailResponse, error)

	// GetLinkedArtifacts gets all artifacts linked to a content item.
	GetLinkedArtifacts(ctx context.Context, contentType string, contentID uuid.UUID, direction LinkDirection) ([]ArtifactLinkResponse, error)

	// ListVideoDefs lists video definitions with filtering.
	ListVideoDefs(ctx context.Context, query VideoDefQuery) (PaginatedResponse[VideoDefResponse], error)
	// GetVideoDef returns a single video definition.
	GetVideoDef(ctx context.Context, defID uuid.UUID) (VideoDefResponse, error)

	// ─── Instance Queries (Layer 3 — FamilyScope required) ──────────────

	// ListActivityLogs lists activity logs for a student with filtering.
	ListActivityLogs(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ActivityLogQuery) (PaginatedResponse[ActivityLogResponse], error)
	// GetActivityLog returns a single activity log entry.
	GetActivityLog(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID) (ActivityLogResponse, error)

	// ListJournalEntries lists journal entries for a student.
	ListJournalEntries(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query JournalEntryQuery) (PaginatedResponse[JournalEntryResponse], error)
	// GetJournalEntry returns a single journal entry.
	GetJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID) (JournalEntryResponse, error)

	// ListReadingProgress lists reading progress for a student.
	ListReadingProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ReadingProgressQuery) (PaginatedResponse[ReadingProgressResponse], error)

	// ListReadingLists lists the family's reading lists.
	ListReadingLists(ctx context.Context, scope *shared.FamilyScope) ([]ReadingListSummaryResponse, error)
	// GetReadingList returns a reading list with items and student progress.
	GetReadingList(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) (ReadingListDetailResponse, error)

	// GetProgressSummary returns progress summary for a student.
	GetProgressSummary(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ProgressQuery) (ProgressSummaryResponse, error)
	// GetSubjectBreakdown returns per-subject breakdown for a student.
	GetSubjectBreakdown(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ProgressQuery) ([]SubjectProgressResponse, error)
	// GetActivityTimeline returns activity timeline for a student.
	GetActivityTimeline(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query TimelineQuery) (PaginatedResponse[TimelineEntryResponse], error)

	// GetResolvedTools returns the family's resolved tool set. Delegates to method::.
	GetResolvedTools(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error)
	// GetStudentTools returns a student-specific resolved tool set.
	GetStudentTools(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error)

	// GetSubjectTaxonomy returns subject taxonomy tree (platform + family custom).
	GetSubjectTaxonomy(ctx context.Context, scope *shared.FamilyScope, query TaxonomyQuery) ([]SubjectTaxonomyResponse, error)

	// GetExportRequest returns export request status.
	GetExportRequest(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (ExportRequestResponse, error)

	// ─── Assessment Engine Queries ──────────────────────────────────────

	// ListQuestions lists questions with filtering (for quiz building).
	ListQuestions(ctx context.Context, query QuestionQuery) (PaginatedResponse[QuestionSummaryResponse], error)
	// GetQuizDef returns a quiz definition with questions.
	GetQuizDef(ctx context.Context, quizDefID uuid.UUID, includeAnswers bool) (QuizDefDetailResponse, error)
	// GetQuizSession returns a quiz session (for resume or review).
	GetQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID) (QuizSessionResponse, error)

	// ─── Sequence Engine Queries ────────────────────────────────────────

	// GetSequenceDef returns a sequence definition with items.
	GetSequenceDef(ctx context.Context, sequenceDefID uuid.UUID) (SequenceDefDetailResponse, error)
	// GetSequenceProgress returns sequence progress with per-item completion status.
	GetSequenceProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID) (SequenceProgressResponse, error)

	// ─── Assignment Queries ─────────────────────────────────────────────

	// ListAssignments lists assignments for a student.
	ListAssignments(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query AssignmentQuery) (PaginatedResponse[AssignmentResponse], error)

	// ─── Video Queries ──────────────────────────────────────────────────

	// GetVideoProgress returns video progress for a student.
	GetVideoProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, videoDefID uuid.UUID) (VideoProgressResponse, error)

	// ─── Event Handlers (no auth context) ───────────────────────────────

	// HandleStudentCreated handles StudentCreated event — initialize student learning defaults.
	HandleStudentCreated(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error
	// HandleStudentDeleted handles StudentDeleted event — cascade-delete learning data.
	HandleStudentDeleted(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error
	// HandleFamilyDeletionScheduled handles FamilyDeletionScheduled — trigger export opportunity.
	HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error
	// HandlePurchaseCompleted handles PurchaseCompleted — integrate purchased content.
	HandlePurchaseCompleted(ctx context.Context, familyID uuid.UUID, purchaseMetadata PurchaseMetadata) error
	// HandleMethodologyConfigUpdated handles MethodologyConfigUpdated — invalidate tool cache.
	HandleMethodologyConfigUpdated(ctx context.Context) error

	// ─── Background Jobs ──────────────────────────────────────────────────
	// SnapshotProgress computes and stores weekly progress snapshots for all active students.
	// Runs as a scheduled background job (weekly, Sunday midnight UTC). [06-learn §12.3]
	SnapshotProgress(ctx context.Context) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [06-learn §6]
// Layer 1 repos: no FamilyScope (publisher-based access at app level).
// Layer 3 repos: FamilyScope required for all operations (RLS enforcement).
// ═══════════════════════════════════════════════════════════════════════════════

// ─── Layer 1: Definition Repositories (no FamilyScope) ──────────────────────

// ActivityDefRepository manages activity definition persistence. [06-learn §6]
type ActivityDefRepository interface {
	Create(ctx context.Context, def *ActivityDefModel) error
	FindByID(ctx context.Context, defID uuid.UUID) (*ActivityDefModel, error)
	List(ctx context.Context, query *ActivityDefQuery) ([]ActivityDefModel, error)
	Update(ctx context.Context, def *ActivityDefModel) error
	SoftDelete(ctx context.Context, defID uuid.UUID) error
}

// ReadingItemRepository manages reading item persistence. [06-learn §6]
type ReadingItemRepository interface {
	Create(ctx context.Context, item *ReadingItemModel) error
	FindByID(ctx context.Context, itemID uuid.UUID) (*ReadingItemModel, error)
	List(ctx context.Context, query *ReadingItemQuery) ([]ReadingItemModel, error)
	Update(ctx context.Context, item *ReadingItemModel) error
	FindByIDs(ctx context.Context, itemIDs []uuid.UUID) ([]ReadingItemModel, error)
}

// ArtifactLinkRepository manages artifact link persistence. [06-learn §9]
type ArtifactLinkRepository interface {
	Create(ctx context.Context, link *ArtifactLinkModel) error
	FindByID(ctx context.Context, linkID uuid.UUID) (*ArtifactLinkModel, error)
	FindByContent(ctx context.Context, contentType string, contentID uuid.UUID, direction LinkDirection) ([]ArtifactLinkModel, error)
	Delete(ctx context.Context, linkID uuid.UUID) error
}

// QuestionRepository manages question persistence. [06-learn §6]
type QuestionRepository interface {
	Create(ctx context.Context, q *QuestionModel) error
	FindByID(ctx context.Context, questionID uuid.UUID) (*QuestionModel, error)
	List(ctx context.Context, query *QuestionQuery) ([]QuestionModel, error)
	Update(ctx context.Context, q *QuestionModel) error
	FindByIDs(ctx context.Context, questionIDs []uuid.UUID) ([]QuestionModel, error)
}

// QuizDefRepository manages quiz definition persistence. [06-learn §6]
type QuizDefRepository interface {
	Create(ctx context.Context, def *QuizDefModel) error
	FindByID(ctx context.Context, quizDefID uuid.UUID) (*QuizDefModel, error)
	Update(ctx context.Context, def *QuizDefModel) error
	// SetQuestions replaces the quiz's question set.
	SetQuestions(ctx context.Context, quizDefID uuid.UUID, questions []QuizQuestionModel) error
	// ListQuestions returns the quiz's questions in sort order.
	ListQuestions(ctx context.Context, quizDefID uuid.UUID) ([]QuizQuestionModel, error)
}

// SequenceDefRepository manages sequence definition persistence. [06-learn §6]
type SequenceDefRepository interface {
	Create(ctx context.Context, def *SequenceDefModel) error
	FindByID(ctx context.Context, sequenceDefID uuid.UUID) (*SequenceDefModel, error)
	Update(ctx context.Context, def *SequenceDefModel) error
	// SetItems replaces the sequence's item set.
	SetItems(ctx context.Context, sequenceDefID uuid.UUID, items []SequenceItemModel) error
	// ListItems returns the sequence's items in sort order.
	ListItems(ctx context.Context, sequenceDefID uuid.UUID) ([]SequenceItemModel, error)
}

// VideoDefRepository manages video definition persistence. [06-learn §6]
type VideoDefRepository interface {
	Create(ctx context.Context, def *VideoDefModel) error
	FindByID(ctx context.Context, defID uuid.UUID) (*VideoDefModel, error)
	List(ctx context.Context, query *VideoDefQuery) ([]VideoDefModel, error)
	Update(ctx context.Context, def *VideoDefModel) error
}

// ─── Layer 3: Instance Repositories (FamilyScope required) ──────────────────

// ActivityLogRepository manages activity log persistence. [06-learn §6]
type ActivityLogRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, log *ActivityLogModel) error
	FindByID(ctx context.Context, scope *shared.FamilyScope, logID uuid.UUID) (*ActivityLogModel, error)
	ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *ActivityLogQuery) ([]ActivityLogModel, error)
	Update(ctx context.Context, scope *shared.FamilyScope, log *ActivityLogModel) error
	Delete(ctx context.Context, scope *shared.FamilyScope, logID uuid.UUID) error
	// CountByStudentDateRange counts activities in a date range for progress queries.
	CountByStudentDateRange(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) (int64, error)
	// HoursBySubject aggregates hours by subject for a student in a date range.
	HoursBySubject(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) ([]SubjectHours, error)
}

// JournalEntryRepository manages journal entry persistence. [06-learn §6]
type JournalEntryRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, entry *JournalEntryModel) error
	FindByID(ctx context.Context, scope *shared.FamilyScope, entryID uuid.UUID) (*JournalEntryModel, error)
	ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *JournalEntryQuery) ([]JournalEntryModel, error)
	Update(ctx context.Context, scope *shared.FamilyScope, entry *JournalEntryModel) error
	Delete(ctx context.Context, scope *shared.FamilyScope, entryID uuid.UUID) error
	// CountByStudentDateRange counts journal entries in a date range for progress queries.
	CountByStudentDateRange(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) (int64, error)
}

// ReadingProgressRepository manages reading progress persistence. [06-learn §6]
type ReadingProgressRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, progress *ReadingProgressModel) error
	FindByID(ctx context.Context, scope *shared.FamilyScope, progressID uuid.UUID) (*ReadingProgressModel, error)
	ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *ReadingProgressQuery) ([]ReadingProgressModel, error)
	Update(ctx context.Context, scope *shared.FamilyScope, progress *ReadingProgressModel) error
	// Exists checks if a student is already tracking a reading item.
	Exists(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, readingItemID uuid.UUID) (bool, error)
	// CountCompleted counts completed books for a student in a date range.
	CountCompleted(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) (int64, error)
}

// ReadingListRepository manages reading list persistence. [06-learn §6]
type ReadingListRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, list *ReadingListModel) error
	FindByID(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) (*ReadingListModel, error)
	ListByFamily(ctx context.Context, scope *shared.FamilyScope) ([]ReadingListModel, error)
	Update(ctx context.Context, scope *shared.FamilyScope, list *ReadingListModel) error
	Delete(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) error
	// AddItems adds items to a reading list.
	AddItems(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID, itemIDs []uuid.UUID) error
	// RemoveItems removes items from a reading list.
	RemoveItems(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID, itemIDs []uuid.UUID) error
	// ListItems lists items in a reading list with sort order.
	ListItems(ctx context.Context, listID uuid.UUID) ([]ReadingListItemModel, error)
}

// ProgressRepository manages progress snapshot persistence. [06-learn §6]
type ProgressRepository interface {
	CreateSnapshot(ctx context.Context, scope *shared.FamilyScope, snapshot *ProgressSnapshotModel) error
	GetLatestSnapshot(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*ProgressSnapshotModel, error)
	ListSnapshots(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, dateFrom time.Time, dateTo time.Time) ([]ProgressSnapshotModel, error)
}

// ExportRepository manages export request persistence. [06-learn §6]
type ExportRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, request *ExportRequestModel) error
	FindByID(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (*ExportRequestModel, error)
	// HasActiveExport checks if there is an active (pending/processing) export for a family.
	HasActiveExport(ctx context.Context, scope *shared.FamilyScope) (bool, error)
	// UpdateStatus updates export status and file URL.
	UpdateStatus(ctx context.Context, exportID uuid.UUID, status string, fileURL *string, expiresAt *time.Time, errorMessage *string) error
}

// QuizSessionRepository manages quiz session persistence. [06-learn §6]
type QuizSessionRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, session *QuizSessionModel) error
	FindByID(ctx context.Context, scope *shared.FamilyScope, sessionID uuid.UUID) (*QuizSessionModel, error)
	Update(ctx context.Context, scope *shared.FamilyScope, session *QuizSessionModel) error
}

// SequenceProgressRepository manages sequence progress persistence. [06-learn §6]
type SequenceProgressRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, progress *SequenceProgressModel) error
	FindByID(ctx context.Context, scope *shared.FamilyScope, progressID uuid.UUID) (*SequenceProgressModel, error)
	Update(ctx context.Context, scope *shared.FamilyScope, progress *SequenceProgressModel) error
}

// AssignmentRepository manages assignment persistence. [06-learn §6]
type AssignmentRepository interface {
	Create(ctx context.Context, scope *shared.FamilyScope, assignment *StudentAssignmentModel) error
	FindByID(ctx context.Context, scope *shared.FamilyScope, assignmentID uuid.UUID) (*StudentAssignmentModel, error)
	ListByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query *AssignmentQuery) ([]StudentAssignmentModel, error)
	Update(ctx context.Context, scope *shared.FamilyScope, assignment *StudentAssignmentModel) error
	Delete(ctx context.Context, scope *shared.FamilyScope, assignmentID uuid.UUID) error
}

// VideoProgressRepository manages video progress persistence. [06-learn §6]
type VideoProgressRepository interface {
	Upsert(ctx context.Context, scope *shared.FamilyScope, progress *VideoProgressModel) error
	FindByStudentAndVideo(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, videoDefID uuid.UUID) (*VideoProgressModel, error)
}

// ─── Platform Repositories (no FamilyScope) ─────────────────────────────────

// SubjectTaxonomyRepository manages subject taxonomy persistence. [06-learn §6]
type SubjectTaxonomyRepository interface {
	// List lists taxonomy nodes with optional filtering by level and parent.
	List(ctx context.Context, query *TaxonomyQuery) ([]SubjectTaxonomyModel, error)
	// FindBySlug finds a taxonomy node by slug.
	FindBySlug(ctx context.Context, slug string) (*SubjectTaxonomyModel, error)
	// ValidateSlugs validates that all slugs exist in the taxonomy.
	ValidateSlugs(ctx context.Context, slugs []string) (bool, error)
	// ListCustomSubjects lists family-scoped custom subjects.
	ListCustomSubjects(ctx context.Context, scope *shared.FamilyScope) ([]CustomSubjectModel, error)
	// CreateCustomSubject creates a family-scoped custom subject.
	CreateCustomSubject(ctx context.Context, scope *shared.FamilyScope, subject *CustomSubjectModel) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Adapter Interfaces [06-learn §7]
// ═══════════════════════════════════════════════════════════════════════════════

// MediaAdapter handles file upload/download operations.
// Delegates to media:: domain for actual storage and validation. [CODING §8.1]
type MediaAdapter interface {
	// ValidateAttachment validates an attachment (magic bytes, size limit, MIME type).
	ValidateAttachment(ctx context.Context, attachment *AttachmentInput) error
	// GetUploadURL generates a pre-signed upload URL for direct client upload.
	GetUploadURL(ctx context.Context, contentType string, filename string) (*UploadURLResponse, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer-Defined Cross-Domain Interfaces [ARCH §4.2]
// Narrow interfaces for cross-domain service calls. Adapters wired in main.go.
// ═══════════════════════════════════════════════════════════════════════════════

// IamServiceForLearn is the subset of iam::IamService that learn:: needs.
type IamServiceForLearn interface {
	// StudentBelongsToFamily checks if a student belongs to a family.
	StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error)
	// GetStudentName returns the student's display name.
	GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error)
}

// MethodServiceForLearn is the subset of method::MethodologyService that learn:: needs.
type MethodServiceForLearn interface {
	// ResolveFamilyTools returns the family's resolved tool set.
	ResolveFamilyTools(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error)
	// ResolveStudentTools returns a student-specific resolved tool set.
	ResolveStudentTools(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error)
}

// MktServiceForLearn is the subset of mkt::MarketplaceService that learn:: needs.
// Stubbed until mkt:: is implemented. [ARCH §4.2]
type MktServiceForLearn interface {
	// IsPublisherMember checks if a caller is a member of a publisher.
	IsPublisherMember(ctx context.Context, callerID uuid.UUID, publisherID uuid.UUID) (bool, error)
}
