package learn

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/learn/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Re-export domain errors for test convenience.
var errActivityDefNotFound = domain.ErrActivityDefNotFound

// mockLearningService implements LearningService with function pointers for testing.
type mockLearningService struct {
	// Activity defs
	createActivityDefFn func(ctx context.Context, cmd CreateActivityDefCommand) (ActivityDefResponse, error)
	updateActivityDefFn func(ctx context.Context, defID uuid.UUID, cmd UpdateActivityDefCommand) (ActivityDefResponse, error)
	deleteActivityDefFn func(ctx context.Context, defID uuid.UUID, callerID uuid.UUID) error
	listActivityDefsFn  func(ctx context.Context, query ActivityDefQuery) (PaginatedResponse[ActivityDefSummaryResponse], error)
	getActivityDefFn    func(ctx context.Context, defID uuid.UUID) (ActivityDefResponse, error)

	// Activity logs
	logActivityFn       func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd LogActivityCommand) (ActivityLogResponse, error)
	updateActivityLogFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID, cmd UpdateActivityLogCommand) (ActivityLogResponse, error)
	deleteActivityLogFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID) error
	listActivityLogsFn  func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ActivityLogQuery) (PaginatedResponse[ActivityLogResponse], error)
	getActivityLogFn    func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID) (ActivityLogResponse, error)

	// Reading items
	createReadingItemFn func(ctx context.Context, cmd CreateReadingItemCommand) (ReadingItemResponse, error)
	updateReadingItemFn func(ctx context.Context, itemID uuid.UUID, cmd UpdateReadingItemCommand) (ReadingItemResponse, error)
	listReadingItemsFn  func(ctx context.Context, query ReadingItemQuery) (PaginatedResponse[ReadingItemSummaryResponse], error)
	getReadingItemFn    func(ctx context.Context, itemID uuid.UUID) (ReadingItemDetailResponse, error)

	// Reading progress
	startReadingFn         func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartReadingCommand) (ReadingProgressResponse, error)
	updateReadingProgressFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateReadingProgressCommand) (ReadingProgressResponse, error)
	listReadingProgressFn  func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ReadingProgressQuery) (PaginatedResponse[ReadingProgressResponse], error)

	// Journal entries
	createJournalEntryFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd CreateJournalEntryCommand) (JournalEntryResponse, error)
	updateJournalEntryFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID, cmd UpdateJournalEntryCommand) (JournalEntryResponse, error)
	deleteJournalEntryFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID) error
	listJournalEntriesFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query JournalEntryQuery) (PaginatedResponse[JournalEntryResponse], error)
	getJournalEntryFn    func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID) (JournalEntryResponse, error)

	// Reading lists
	createReadingListFn func(ctx context.Context, scope *shared.FamilyScope, cmd CreateReadingListCommand) (ReadingListResponse, error)
	updateReadingListFn func(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID, cmd UpdateReadingListCommand) (ReadingListResponse, error)
	deleteReadingListFn func(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) error
	listReadingListsFn  func(ctx context.Context, scope *shared.FamilyScope) ([]ReadingListSummaryResponse, error)
	getReadingListFn    func(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) (ReadingListDetailResponse, error)

	// Taxonomy
	getSubjectTaxonomyFn  func(ctx context.Context, scope *shared.FamilyScope, query TaxonomyQuery) ([]SubjectTaxonomyResponse, error)
	createCustomSubjectFn func(ctx context.Context, scope *shared.FamilyScope, cmd CreateCustomSubjectCommand) (CustomSubjectResponse, error)

	// Artifact links
	linkArtifactsFn    func(ctx context.Context, cmd CreateArtifactLinkCommand) (ArtifactLinkResponse, error)
	unlinkArtifactsFn  func(ctx context.Context, linkID uuid.UUID, callerID uuid.UUID) error
	getLinkedArtifactsFn func(ctx context.Context, contentType string, contentID uuid.UUID, direction LinkDirection) ([]ArtifactLinkResponse, error)

	// Progress
	getProgressSummaryFn  func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ProgressQuery) (ProgressSummaryResponse, error)
	getSubjectBreakdownFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ProgressQuery) ([]SubjectProgressResponse, error)
	getActivityTimelineFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query TimelineQuery) (PaginatedResponse[TimelineEntryResponse], error)

	// Export
	requestDataExportFn func(ctx context.Context, scope *shared.FamilyScope, cmd RequestExportCommand) (ExportRequestResponse, error)
	getExportRequestFn  func(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (ExportRequestResponse, error)

	// Tools
	getResolvedToolsFn func(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error)
	getStudentToolsFn  func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error)

	// Questions
	createQuestionFn func(ctx context.Context, cmd CreateQuestionCommand) (QuestionResponse, error)
	updateQuestionFn func(ctx context.Context, questionID uuid.UUID, cmd UpdateQuestionCommand) (QuestionResponse, error)
	listQuestionsFn  func(ctx context.Context, query QuestionQuery) (PaginatedResponse[QuestionSummaryResponse], error)

	// Quiz defs
	createQuizDefFn func(ctx context.Context, cmd CreateQuizDefCommand) (QuizDefResponse, error)
	updateQuizDefFn func(ctx context.Context, quizDefID uuid.UUID, cmd UpdateQuizDefCommand) (QuizDefResponse, error)
	getQuizDefFn    func(ctx context.Context, quizDefID uuid.UUID, includeAnswers bool) (QuizDefDetailResponse, error)

	// Quiz sessions
	startQuizSessionFn  func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartQuizSessionCommand) (QuizSessionResponse, error)
	updateQuizSessionFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd UpdateQuizSessionCommand) (QuizSessionResponse, error)
	scoreQuizSessionFn  func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd ScoreQuizCommand) (QuizSessionResponse, error)
	getQuizSessionFn    func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID) (QuizSessionResponse, error)

	// Sequence defs
	createSequenceDefFn func(ctx context.Context, cmd CreateSequenceDefCommand) (SequenceDefResponse, error)
	updateSequenceDefFn func(ctx context.Context, defID uuid.UUID, cmd UpdateSequenceDefCommand) (SequenceDefResponse, error)
	getSequenceDefFn    func(ctx context.Context, defID uuid.UUID) (SequenceDefDetailResponse, error)

	// Sequence progress
	startSequenceFn          func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartSequenceCommand) (SequenceProgressResponse, error)
	updateSequenceProgressFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateSequenceProgressCommand) (SequenceProgressResponse, error)
	getSequenceProgressFn    func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID) (SequenceProgressResponse, error)

	// Assignments
	createAssignmentFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd CreateAssignmentCommand) (AssignmentResponse, error)
	updateAssignmentFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID, cmd UpdateAssignmentCommand) (AssignmentResponse, error)
	deleteAssignmentFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID) error
	listAssignmentsFn  func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query AssignmentQuery) (PaginatedResponse[AssignmentResponse], error)

	// Video defs
	listVideoDefsFn func(ctx context.Context, query VideoDefQuery) (PaginatedResponse[VideoDefResponse], error)
	getVideoDefFn   func(ctx context.Context, defID uuid.UUID) (VideoDefResponse, error)

	// Video progress
	updateVideoProgressFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd UpdateVideoProgressCommand) (VideoProgressResponse, error)
	getVideoProgressFn    func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, videoDefID uuid.UUID) (VideoProgressResponse, error)
}

func newMockLearningService() *mockLearningService {
	return &mockLearningService{}
}

// ─── Activity Def Methods ───────────────────────────────────────────────────

func (m *mockLearningService) CreateActivityDef(ctx context.Context, cmd CreateActivityDefCommand) (ActivityDefResponse, error) {
	if m.createActivityDefFn != nil {
		return m.createActivityDefFn(ctx, cmd)
	}
	panic("CreateActivityDef not mocked")
}

func (m *mockLearningService) UpdateActivityDef(ctx context.Context, defID uuid.UUID, cmd UpdateActivityDefCommand) (ActivityDefResponse, error) {
	if m.updateActivityDefFn != nil {
		return m.updateActivityDefFn(ctx, defID, cmd)
	}
	panic("UpdateActivityDef not mocked")
}

func (m *mockLearningService) DeleteActivityDef(ctx context.Context, defID uuid.UUID, callerID uuid.UUID) error {
	if m.deleteActivityDefFn != nil {
		return m.deleteActivityDefFn(ctx, defID, callerID)
	}
	panic("DeleteActivityDef not mocked")
}

func (m *mockLearningService) ListActivityDefs(ctx context.Context, query ActivityDefQuery) (PaginatedResponse[ActivityDefSummaryResponse], error) {
	if m.listActivityDefsFn != nil {
		return m.listActivityDefsFn(ctx, query)
	}
	panic("ListActivityDefs not mocked")
}

func (m *mockLearningService) GetActivityDef(ctx context.Context, defID uuid.UUID) (ActivityDefResponse, error) {
	if m.getActivityDefFn != nil {
		return m.getActivityDefFn(ctx, defID)
	}
	panic("GetActivityDef not mocked")
}

// ─── Activity Log Methods ───────────────────────────────────────────────────

func (m *mockLearningService) LogActivity(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd LogActivityCommand) (ActivityLogResponse, error) {
	if m.logActivityFn != nil {
		return m.logActivityFn(ctx, scope, studentID, cmd)
	}
	panic("LogActivity not mocked")
}

func (m *mockLearningService) UpdateActivityLog(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID, cmd UpdateActivityLogCommand) (ActivityLogResponse, error) {
	if m.updateActivityLogFn != nil {
		return m.updateActivityLogFn(ctx, scope, studentID, logID, cmd)
	}
	panic("UpdateActivityLog not mocked")
}

func (m *mockLearningService) DeleteActivityLog(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID) error {
	if m.deleteActivityLogFn != nil {
		return m.deleteActivityLogFn(ctx, scope, studentID, logID)
	}
	panic("DeleteActivityLog not mocked")
}

func (m *mockLearningService) ListActivityLogs(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ActivityLogQuery) (PaginatedResponse[ActivityLogResponse], error) {
	if m.listActivityLogsFn != nil {
		return m.listActivityLogsFn(ctx, scope, studentID, query)
	}
	panic("ListActivityLogs not mocked")
}

func (m *mockLearningService) GetActivityLog(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID) (ActivityLogResponse, error) {
	if m.getActivityLogFn != nil {
		return m.getActivityLogFn(ctx, scope, studentID, logID)
	}
	panic("GetActivityLog not mocked")
}

// ─── Reading Item Methods ───────────────────────────────────────────────────

func (m *mockLearningService) CreateReadingItem(ctx context.Context, cmd CreateReadingItemCommand) (ReadingItemResponse, error) {
	if m.createReadingItemFn != nil {
		return m.createReadingItemFn(ctx, cmd)
	}
	panic("CreateReadingItem not mocked")
}

func (m *mockLearningService) UpdateReadingItem(ctx context.Context, itemID uuid.UUID, cmd UpdateReadingItemCommand) (ReadingItemResponse, error) {
	if m.updateReadingItemFn != nil {
		return m.updateReadingItemFn(ctx, itemID, cmd)
	}
	panic("UpdateReadingItem not mocked")
}

// ─── Reading Progress Methods ───────────────────────────────────────────────

func (m *mockLearningService) StartReading(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartReadingCommand) (ReadingProgressResponse, error) {
	if m.startReadingFn != nil {
		return m.startReadingFn(ctx, scope, studentID, cmd)
	}
	panic("StartReading not mocked")
}

func (m *mockLearningService) UpdateReadingProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateReadingProgressCommand) (ReadingProgressResponse, error) {
	if m.updateReadingProgressFn != nil {
		return m.updateReadingProgressFn(ctx, scope, studentID, progressID, cmd)
	}
	panic("UpdateReadingProgress not mocked")
}

// ─── Reading List Methods ───────────────────────────────────────────────────

func (m *mockLearningService) CreateReadingList(ctx context.Context, scope *shared.FamilyScope, cmd CreateReadingListCommand) (ReadingListResponse, error) {
	if m.createReadingListFn != nil {
		return m.createReadingListFn(ctx, scope, cmd)
	}
	panic("CreateReadingList not mocked")
}

func (m *mockLearningService) UpdateReadingList(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID, cmd UpdateReadingListCommand) (ReadingListResponse, error) {
	if m.updateReadingListFn != nil {
		return m.updateReadingListFn(ctx, scope, listID, cmd)
	}
	panic("UpdateReadingList not mocked")
}

func (m *mockLearningService) DeleteReadingList(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) error {
	if m.deleteReadingListFn != nil {
		return m.deleteReadingListFn(ctx, scope, listID)
	}
	panic("DeleteReadingList not mocked")
}

// ─── Journal Entry Methods ──────────────────────────────────────────────────

func (m *mockLearningService) CreateJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd CreateJournalEntryCommand) (JournalEntryResponse, error) {
	if m.createJournalEntryFn != nil {
		return m.createJournalEntryFn(ctx, scope, studentID, cmd)
	}
	panic("CreateJournalEntry not mocked")
}

func (m *mockLearningService) UpdateJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID, cmd UpdateJournalEntryCommand) (JournalEntryResponse, error) {
	if m.updateJournalEntryFn != nil {
		return m.updateJournalEntryFn(ctx, scope, studentID, entryID, cmd)
	}
	panic("UpdateJournalEntry not mocked")
}

func (m *mockLearningService) DeleteJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID) error {
	if m.deleteJournalEntryFn != nil {
		return m.deleteJournalEntryFn(ctx, scope, studentID, entryID)
	}
	panic("DeleteJournalEntry not mocked")
}

func (m *mockLearningService) CreateCustomSubject(ctx context.Context, scope *shared.FamilyScope, cmd CreateCustomSubjectCommand) (CustomSubjectResponse, error) {
	if m.createCustomSubjectFn != nil {
		return m.createCustomSubjectFn(ctx, scope, cmd)
	}
	panic("CreateCustomSubject not mocked")
}

// ─── Artifact Link Methods ──────────────────────────────────────────────────

func (m *mockLearningService) LinkArtifacts(ctx context.Context, cmd CreateArtifactLinkCommand) (ArtifactLinkResponse, error) {
	if m.linkArtifactsFn != nil {
		return m.linkArtifactsFn(ctx, cmd)
	}
	panic("LinkArtifacts not mocked")
}
func (m *mockLearningService) UnlinkArtifacts(ctx context.Context, linkID uuid.UUID, callerID uuid.UUID) error {
	if m.unlinkArtifactsFn != nil {
		return m.unlinkArtifactsFn(ctx, linkID, callerID)
	}
	panic("UnlinkArtifacts not mocked")
}
func (m *mockLearningService) GetLinkedArtifacts(ctx context.Context, contentType string, contentID uuid.UUID, direction LinkDirection) ([]ArtifactLinkResponse, error) {
	if m.getLinkedArtifactsFn != nil {
		return m.getLinkedArtifactsFn(ctx, contentType, contentID, direction)
	}
	panic("GetLinkedArtifacts not mocked")
}

// ─── Progress Methods ───────────────────────────────────────────────────────

func (m *mockLearningService) GetProgressSummary(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ProgressQuery) (ProgressSummaryResponse, error) {
	if m.getProgressSummaryFn != nil {
		return m.getProgressSummaryFn(ctx, scope, studentID, query)
	}
	panic("GetProgressSummary not mocked")
}
func (m *mockLearningService) GetSubjectBreakdown(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ProgressQuery) ([]SubjectProgressResponse, error) {
	if m.getSubjectBreakdownFn != nil {
		return m.getSubjectBreakdownFn(ctx, scope, studentID, query)
	}
	panic("GetSubjectBreakdown not mocked")
}
func (m *mockLearningService) GetActivityTimeline(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query TimelineQuery) (PaginatedResponse[TimelineEntryResponse], error) {
	if m.getActivityTimelineFn != nil {
		return m.getActivityTimelineFn(ctx, scope, studentID, query)
	}
	panic("GetActivityTimeline not mocked")
}

// ─── Export Methods ──────────────────────────────────────────────────────────

func (m *mockLearningService) RequestDataExport(ctx context.Context, scope *shared.FamilyScope, cmd RequestExportCommand) (ExportRequestResponse, error) {
	if m.requestDataExportFn != nil {
		return m.requestDataExportFn(ctx, scope, cmd)
	}
	panic("RequestDataExport not mocked")
}
func (m *mockLearningService) GetExportRequest(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (ExportRequestResponse, error) {
	if m.getExportRequestFn != nil {
		return m.getExportRequestFn(ctx, scope, exportID)
	}
	panic("GetExportRequest not mocked")
}

// ─── Tool Methods ────────────────────────────────────────────────────────────

func (m *mockLearningService) GetResolvedTools(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error) {
	if m.getResolvedToolsFn != nil {
		return m.getResolvedToolsFn(ctx, scope)
	}
	panic("GetResolvedTools not mocked")
}
func (m *mockLearningService) GetStudentTools(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error) {
	if m.getStudentToolsFn != nil {
		return m.getStudentToolsFn(ctx, scope, studentID)
	}
	panic("GetStudentTools not mocked")
}

// ─── Question Methods ────────────────────────────────────────────────────────

func (m *mockLearningService) CreateQuestion(ctx context.Context, cmd CreateQuestionCommand) (QuestionResponse, error) {
	if m.createQuestionFn != nil {
		return m.createQuestionFn(ctx, cmd)
	}
	panic("CreateQuestion not mocked")
}
func (m *mockLearningService) UpdateQuestion(ctx context.Context, questionID uuid.UUID, cmd UpdateQuestionCommand) (QuestionResponse, error) {
	if m.updateQuestionFn != nil {
		return m.updateQuestionFn(ctx, questionID, cmd)
	}
	panic("UpdateQuestion not mocked")
}
func (m *mockLearningService) ListQuestions(ctx context.Context, query QuestionQuery) (PaginatedResponse[QuestionSummaryResponse], error) {
	if m.listQuestionsFn != nil {
		return m.listQuestionsFn(ctx, query)
	}
	panic("ListQuestions not mocked")
}

// ─── Quiz Definition Methods ────────────────────────────────────────────────

func (m *mockLearningService) CreateQuizDef(ctx context.Context, cmd CreateQuizDefCommand) (QuizDefResponse, error) {
	if m.createQuizDefFn != nil {
		return m.createQuizDefFn(ctx, cmd)
	}
	panic("CreateQuizDef not mocked")
}
func (m *mockLearningService) UpdateQuizDef(ctx context.Context, quizDefID uuid.UUID, cmd UpdateQuizDefCommand) (QuizDefResponse, error) {
	if m.updateQuizDefFn != nil {
		return m.updateQuizDefFn(ctx, quizDefID, cmd)
	}
	panic("UpdateQuizDef not mocked")
}
func (m *mockLearningService) GetQuizDef(ctx context.Context, quizDefID uuid.UUID, includeAnswers bool) (QuizDefDetailResponse, error) {
	if m.getQuizDefFn != nil {
		return m.getQuizDefFn(ctx, quizDefID, includeAnswers)
	}
	panic("GetQuizDef not mocked")
}

// ─── Quiz Session Methods ───────────────────────────────────────────────────

func (m *mockLearningService) StartQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartQuizSessionCommand) (QuizSessionResponse, error) {
	if m.startQuizSessionFn != nil {
		return m.startQuizSessionFn(ctx, scope, studentID, cmd)
	}
	panic("StartQuizSession not mocked")
}
func (m *mockLearningService) UpdateQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd UpdateQuizSessionCommand) (QuizSessionResponse, error) {
	if m.updateQuizSessionFn != nil {
		return m.updateQuizSessionFn(ctx, scope, studentID, sessionID, cmd)
	}
	panic("UpdateQuizSession not mocked")
}
func (m *mockLearningService) ScoreQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd ScoreQuizCommand) (QuizSessionResponse, error) {
	if m.scoreQuizSessionFn != nil {
		return m.scoreQuizSessionFn(ctx, scope, studentID, sessionID, cmd)
	}
	panic("ScoreQuizSession not mocked")
}
func (m *mockLearningService) GetQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID) (QuizSessionResponse, error) {
	if m.getQuizSessionFn != nil {
		return m.getQuizSessionFn(ctx, scope, studentID, sessionID)
	}
	panic("GetQuizSession not mocked")
}

// ─── Reading Item/List/Progress/Journal Query Methods ───────────────────────

func (m *mockLearningService) ListReadingItems(ctx context.Context, query ReadingItemQuery) (PaginatedResponse[ReadingItemSummaryResponse], error) {
	if m.listReadingItemsFn != nil {
		return m.listReadingItemsFn(ctx, query)
	}
	panic("ListReadingItems not mocked")
}
func (m *mockLearningService) GetReadingItem(ctx context.Context, itemID uuid.UUID) (ReadingItemDetailResponse, error) {
	if m.getReadingItemFn != nil {
		return m.getReadingItemFn(ctx, itemID)
	}
	panic("GetReadingItem not mocked")
}
func (m *mockLearningService) ListJournalEntries(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query JournalEntryQuery) (PaginatedResponse[JournalEntryResponse], error) {
	if m.listJournalEntriesFn != nil {
		return m.listJournalEntriesFn(ctx, scope, studentID, query)
	}
	panic("ListJournalEntries not mocked")
}
func (m *mockLearningService) GetJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID) (JournalEntryResponse, error) {
	if m.getJournalEntryFn != nil {
		return m.getJournalEntryFn(ctx, scope, studentID, entryID)
	}
	panic("GetJournalEntry not mocked")
}
func (m *mockLearningService) ListReadingProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ReadingProgressQuery) (PaginatedResponse[ReadingProgressResponse], error) {
	if m.listReadingProgressFn != nil {
		return m.listReadingProgressFn(ctx, scope, studentID, query)
	}
	panic("ListReadingProgress not mocked")
}
func (m *mockLearningService) ListReadingLists(ctx context.Context, scope *shared.FamilyScope) ([]ReadingListSummaryResponse, error) {
	if m.listReadingListsFn != nil {
		return m.listReadingListsFn(ctx, scope)
	}
	panic("ListReadingLists not mocked")
}
func (m *mockLearningService) GetReadingList(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) (ReadingListDetailResponse, error) {
	if m.getReadingListFn != nil {
		return m.getReadingListFn(ctx, scope, listID)
	}
	panic("GetReadingList not mocked")
}
func (m *mockLearningService) GetSubjectTaxonomy(ctx context.Context, scope *shared.FamilyScope, query TaxonomyQuery) ([]SubjectTaxonomyResponse, error) {
	if m.getSubjectTaxonomyFn != nil {
		return m.getSubjectTaxonomyFn(ctx, scope, query)
	}
	panic("GetSubjectTaxonomy not mocked")
}

// ─── Sequence Definition Methods ─────────────────────────────────────────────

func (m *mockLearningService) CreateSequenceDef(ctx context.Context, cmd CreateSequenceDefCommand) (SequenceDefResponse, error) {
	if m.createSequenceDefFn != nil {
		return m.createSequenceDefFn(ctx, cmd)
	}
	panic("CreateSequenceDef not mocked")
}
func (m *mockLearningService) UpdateSequenceDef(ctx context.Context, defID uuid.UUID, cmd UpdateSequenceDefCommand) (SequenceDefResponse, error) {
	if m.updateSequenceDefFn != nil {
		return m.updateSequenceDefFn(ctx, defID, cmd)
	}
	panic("UpdateSequenceDef not mocked")
}
func (m *mockLearningService) GetSequenceDef(ctx context.Context, defID uuid.UUID) (SequenceDefDetailResponse, error) {
	if m.getSequenceDefFn != nil {
		return m.getSequenceDefFn(ctx, defID)
	}
	panic("GetSequenceDef not mocked")
}

// ─── Sequence Progress Methods ───────────────────────────────────────────────

func (m *mockLearningService) StartSequence(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartSequenceCommand) (SequenceProgressResponse, error) {
	if m.startSequenceFn != nil {
		return m.startSequenceFn(ctx, scope, studentID, cmd)
	}
	panic("StartSequence not mocked")
}
func (m *mockLearningService) UpdateSequenceProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateSequenceProgressCommand) (SequenceProgressResponse, error) {
	if m.updateSequenceProgressFn != nil {
		return m.updateSequenceProgressFn(ctx, scope, studentID, progressID, cmd)
	}
	panic("UpdateSequenceProgress not mocked")
}
func (m *mockLearningService) GetSequenceProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID) (SequenceProgressResponse, error) {
	if m.getSequenceProgressFn != nil {
		return m.getSequenceProgressFn(ctx, scope, studentID, progressID)
	}
	panic("GetSequenceProgress not mocked")
}

// ─── Assignment Methods ──────────────────────────────────────────────────────

func (m *mockLearningService) CreateAssignment(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd CreateAssignmentCommand) (AssignmentResponse, error) {
	if m.createAssignmentFn != nil {
		return m.createAssignmentFn(ctx, scope, studentID, cmd)
	}
	panic("CreateAssignment not mocked")
}
func (m *mockLearningService) UpdateAssignment(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID, cmd UpdateAssignmentCommand) (AssignmentResponse, error) {
	if m.updateAssignmentFn != nil {
		return m.updateAssignmentFn(ctx, scope, studentID, assignmentID, cmd)
	}
	panic("UpdateAssignment not mocked")
}
func (m *mockLearningService) DeleteAssignment(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID) error {
	if m.deleteAssignmentFn != nil {
		return m.deleteAssignmentFn(ctx, scope, studentID, assignmentID)
	}
	panic("DeleteAssignment not mocked")
}
func (m *mockLearningService) ListAssignments(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query AssignmentQuery) (PaginatedResponse[AssignmentResponse], error) {
	if m.listAssignmentsFn != nil {
		return m.listAssignmentsFn(ctx, scope, studentID, query)
	}
	panic("ListAssignments not mocked")
}

// ─── Video Definition Methods ────────────────────────────────────────────────

func (m *mockLearningService) ListVideoDefs(ctx context.Context, query VideoDefQuery) (PaginatedResponse[VideoDefResponse], error) {
	if m.listVideoDefsFn != nil {
		return m.listVideoDefsFn(ctx, query)
	}
	panic("ListVideoDefs not mocked")
}
func (m *mockLearningService) GetVideoDef(ctx context.Context, defID uuid.UUID) (VideoDefResponse, error) {
	if m.getVideoDefFn != nil {
		return m.getVideoDefFn(ctx, defID)
	}
	panic("GetVideoDef not mocked")
}

// ─── Video Progress Methods ──────────────────────────────────────────────────

func (m *mockLearningService) UpdateVideoProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd UpdateVideoProgressCommand) (VideoProgressResponse, error) {
	if m.updateVideoProgressFn != nil {
		return m.updateVideoProgressFn(ctx, scope, studentID, cmd)
	}
	panic("UpdateVideoProgress not mocked")
}
func (m *mockLearningService) GetVideoProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, videoDefID uuid.UUID) (VideoProgressResponse, error) {
	if m.getVideoProgressFn != nil {
		return m.getVideoProgressFn(ctx, scope, studentID, videoDefID)
	}
	panic("GetVideoProgress not mocked")
}

// ─── Event Handler Stubs (Batch 9) ──────────────────────────────────────────

func (m *mockLearningService) HandleStudentCreated(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (m *mockLearningService) HandleStudentDeleted(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}
func (m *mockLearningService) HandleFamilyDeletionScheduled(context.Context, uuid.UUID) error {
	return nil
}
func (m *mockLearningService) HandlePurchaseCompleted(context.Context, uuid.UUID, PurchaseMetadata) error {
	return nil
}
func (m *mockLearningService) HandleMethodologyConfigUpdated(context.Context) error {
	return nil
}
func (m *mockLearningService) SnapshotProgress(context.Context) error {
	return nil
}
