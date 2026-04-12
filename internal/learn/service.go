package learn

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/learn/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ─── Service Implementation ──────────────────────────────────────────────────

type learningServiceImpl struct {
	activityDefRepo     ActivityDefRepository
	activityLogRepo     ActivityLogRepository
	readingItemRepo     ReadingItemRepository
	readingProgressRepo ReadingProgressRepository
	readingListRepo     ReadingListRepository
	journalEntryRepo    JournalEntryRepository
	artifactLinkRepo    ArtifactLinkRepository
	progressRepo        ProgressRepository
	taxonomyRepo        SubjectTaxonomyRepository
	exportRepo          ExportRepository
	questionRepo        QuestionRepository
	quizDefRepo         QuizDefRepository
	quizSessionRepo     QuizSessionRepository
	sequenceDefRepo     SequenceDefRepository
	sequenceProgressRepo SequenceProgressRepository
	assignmentRepo      AssignmentRepository
	videoDefRepo        VideoDefRepository
	videoProgressRepo   VideoProgressRepository
	assessmentDefRepo   AssessmentDefRepository
	projectDefRepo      ProjectDefRepository
	assessmentResultRepo AssessmentResultRepository
	projectProgressRepo ProjectProgressRepository
	gradingScaleRepo    GradingScaleRepository
	iam                 IamServiceForLearn
	method              MethodServiceForLearn
	mkt                 MktServiceForLearn
	eventBus            *shared.EventBus
	db                  *gorm.DB
}

// NewLearningService creates a new LearningService.
func NewLearningService(
	activityDefRepo ActivityDefRepository,
	activityLogRepo ActivityLogRepository,
	readingItemRepo ReadingItemRepository,
	readingProgressRepo ReadingProgressRepository,
	readingListRepo ReadingListRepository,
	journalEntryRepo JournalEntryRepository,
	artifactLinkRepo ArtifactLinkRepository,
	progressRepo ProgressRepository,
	taxonomyRepo SubjectTaxonomyRepository,
	exportRepo ExportRepository,
	questionRepo QuestionRepository,
	quizDefRepo QuizDefRepository,
	quizSessionRepo QuizSessionRepository,
	sequenceDefRepo SequenceDefRepository,
	sequenceProgressRepo SequenceProgressRepository,
	assignmentRepo AssignmentRepository,
	videoDefRepo VideoDefRepository,
	videoProgressRepo VideoProgressRepository,
	assessmentDefRepo AssessmentDefRepository,
	projectDefRepo ProjectDefRepository,
	assessmentResultRepo AssessmentResultRepository,
	projectProgressRepo ProjectProgressRepository,
	gradingScaleRepo GradingScaleRepository,
	iam IamServiceForLearn,
	method MethodServiceForLearn,
	mkt MktServiceForLearn,
	eventBus *shared.EventBus,
	db *gorm.DB,
) LearningService {
	return &learningServiceImpl{
		activityDefRepo:      activityDefRepo,
		activityLogRepo:      activityLogRepo,
		readingItemRepo:      readingItemRepo,
		readingProgressRepo:  readingProgressRepo,
		readingListRepo:      readingListRepo,
		journalEntryRepo:     journalEntryRepo,
		artifactLinkRepo:     artifactLinkRepo,
		progressRepo:         progressRepo,
		taxonomyRepo:         taxonomyRepo,
		exportRepo:           exportRepo,
		questionRepo:         questionRepo,
		quizDefRepo:          quizDefRepo,
		quizSessionRepo:      quizSessionRepo,
		sequenceDefRepo:      sequenceDefRepo,
		sequenceProgressRepo: sequenceProgressRepo,
		assignmentRepo:       assignmentRepo,
		videoDefRepo:          videoDefRepo,
		videoProgressRepo:     videoProgressRepo,
		assessmentDefRepo:     assessmentDefRepo,
		projectDefRepo:        projectDefRepo,
		assessmentResultRepo:  assessmentResultRepo,
		projectProgressRepo:   projectProgressRepo,
		gradingScaleRepo:      gradingScaleRepo,
		iam:                   iam,
		method:               method,
		mkt:                  mkt,
		eventBus:             eventBus,
		db:                   db,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// verifyStudentInFamily checks that a student belongs to the family in scope.
func (s *learningServiceImpl) verifyStudentInFamily(ctx context.Context, studentID uuid.UUID, scope *shared.FamilyScope) error {
	belongs, err := s.iam.StudentBelongsToFamily(ctx, studentID, scope.FamilyID())
	if err != nil {
		return err
	}
	if !belongs {
		return &LearningError{Err: domain.ErrStudentNotInFamily}
	}
	return nil
}

// validateSubjectTags validates that all tags exist in the taxonomy or as custom subjects.
func (s *learningServiceImpl) validateSubjectTags(ctx context.Context, tags []string, scope *shared.FamilyScope) error {
	if len(tags) == 0 {
		return nil
	}

	// Check platform taxonomy first.
	allValid, err := s.taxonomyRepo.ValidateSlugs(ctx, tags)
	if err != nil {
		return err
	}
	if allValid {
		return nil
	}

	// Check custom subjects for any tags not in platform taxonomy.
	customs, err := s.taxonomyRepo.ListCustomSubjects(ctx, scope)
	if err != nil {
		return err
	}
	customSlugs := make(map[string]bool, len(customs))
	for _, c := range customs {
		customSlugs[c.Slug] = true
	}

	// Re-validate: check each tag against both platform + custom.
	for _, tag := range tags {
		_, nodeErr := s.taxonomyRepo.FindBySlug(ctx, tag)
		if nodeErr != nil {
			if errors.Is(nodeErr, domain.ErrTaxonomyNotFound) && !customSlugs[tag] {
				return &LearningError{Err: &domain.ErrInvalidSubjectTag{Tag: tag}}
			} else if !errors.Is(nodeErr, domain.ErrTaxonomyNotFound) {
				return nodeErr
			}
			// taxonomy not found but it's a valid custom slug — continue
		}
	}
	return nil
}

// checkPublisherMembership verifies the caller is a member of the publisher.
func (s *learningServiceImpl) checkPublisherMembership(ctx context.Context, callerID uuid.UUID, publisherID uuid.UUID) error {
	isMember, err := s.mkt.IsPublisherMember(ctx, callerID, publisherID)
	if err != nil {
		return err
	}
	if !isMember {
		return &LearningError{Err: domain.ErrNotPublisherMember}
	}
	return nil
}

// defaultActivityDate returns today if the provided date is nil.
func defaultActivityDate(date *time.Time) time.Time {
	if date != nil {
		return *date
	}
	return time.Now().Truncate(24 * time.Hour)
}

// buildQuizQuestionInfos maps DB models to domain-layer quiz question infos for scoring.
func buildQuizQuestionInfos(quizQuestions []QuizQuestionModel, questionMap map[uuid.UUID]*QuestionModel) []domain.QuizQuestionInfo {
	infos := make([]domain.QuizQuestionInfo, 0, len(quizQuestions))
	for _, qq := range quizQuestions {
		q := questionMap[qq.QuestionID]
		if q == nil {
			continue
		}
		infos = append(infos, domain.QuizQuestionInfo{
			QuestionID:     qq.QuestionID,
			Points:         q.Points,
			PointsOverride: qq.PointsOverride,
			AutoScorable:   q.AutoScorable,
		})
	}
	return infos
}

// ═══════════════════════════════════════════════════════════════════════════════
// Activity Definition CRUD (Layer 1)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateActivityDef(ctx context.Context, cmd CreateActivityDefCommand) (ActivityDefResponse, error) {
	if err := s.checkPublisherMembership(ctx, cmd.PublisherID, cmd.PublisherID); err != nil {
		return ActivityDefResponse{}, err
	}

	attachments, _ := json.Marshal(cmd.Attachments)
	if cmd.Attachments == nil {
		attachments = json.RawMessage("[]")
	}

	def := &ActivityDefModel{
		PublisherID:        cmd.PublisherID,
		Title:              cmd.Title,
		Description:        cmd.Description,
		SubjectTags:        StringArray(cmd.SubjectTags),
		MethodologyID:      cmd.MethodologyID,
		ToolID:             cmd.ToolID,
		EstDurationMinutes: cmd.EstDurationMinutes,
		Attachments:        attachments,
	}
	if err := s.activityDefRepo.Create(ctx, def); err != nil {
		return ActivityDefResponse{}, err
	}
	return activityDefToResponse(def), nil
}

func (s *learningServiceImpl) UpdateActivityDef(ctx context.Context, defID uuid.UUID, cmd UpdateActivityDefCommand) (ActivityDefResponse, error) {
	def, err := s.activityDefRepo.FindByID(ctx, defID)
	if err != nil {
		return ActivityDefResponse{}, err
	}

	if cmd.Title != nil {
		def.Title = *cmd.Title
	}
	if cmd.Description != nil {
		def.Description = cmd.Description
	}
	if cmd.SubjectTags != nil {
		def.SubjectTags = StringArray(*cmd.SubjectTags)
	}
	if cmd.MethodologyID != nil {
		def.MethodologyID = cmd.MethodologyID
	}
	if cmd.ToolID != nil {
		def.ToolID = cmd.ToolID
	}
	if cmd.EstDurationMinutes != nil {
		def.EstDurationMinutes = cmd.EstDurationMinutes
	}
	if cmd.Attachments != nil {
		attachments, _ := json.Marshal(*cmd.Attachments)
		def.Attachments = attachments
	}
	def.UpdatedAt = time.Now()

	if err := s.activityDefRepo.Update(ctx, def); err != nil {
		return ActivityDefResponse{}, err
	}
	return activityDefToResponse(def), nil
}

func (s *learningServiceImpl) DeleteActivityDef(ctx context.Context, defID uuid.UUID, callerID uuid.UUID) error {
	def, err := s.activityDefRepo.FindByID(ctx, defID)
	if err != nil {
		return err
	}
	if err := s.checkPublisherMembership(ctx, callerID, def.PublisherID); err != nil {
		return err
	}
	return s.activityDefRepo.SoftDelete(ctx, defID)
}

func (s *learningServiceImpl) ListActivityDefs(ctx context.Context, query ActivityDefQuery) (PaginatedResponse[ActivityDefSummaryResponse], error) {
	defs, err := s.activityDefRepo.List(ctx, &query)
	if err != nil {
		return PaginatedResponse[ActivityDefSummaryResponse]{}, err
	}

	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	hasMore := int64(len(defs)) > limit
	if hasMore {
		defs = defs[:limit]
	}

	items := make([]ActivityDefSummaryResponse, len(defs))
	for i, d := range defs {
		items[i] = activityDefToSummary(&d)
	}

	var nextCursor *uuid.UUID
	if hasMore && len(items) > 0 {
		id := defs[len(defs)-1].ID
		nextCursor = &id
	}

	return PaginatedResponse[ActivityDefSummaryResponse]{
		Data:       items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *learningServiceImpl) GetActivityDef(ctx context.Context, defID uuid.UUID) (ActivityDefResponse, error) {
	def, err := s.activityDefRepo.FindByID(ctx, defID)
	if err != nil {
		return ActivityDefResponse{}, err
	}
	return activityDefToResponse(def), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Activity Log CRUD (Layer 3)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) LogActivity(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd LogActivityCommand) (ActivityLogResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return ActivityLogResponse{}, err
	}

	actDate := defaultActivityDate(cmd.ActivityDate)

	// Enforce domain invariants via aggregate root.
	if _, err := domain.NewActivity(studentID, cmd.Title, cmd.SubjectTags, actDate, cmd.DurationMinutes); err != nil {
		return ActivityLogResponse{}, &LearningError{Err: err}
	}

	// Validate subject tags against taxonomy.
	if err := s.validateSubjectTags(ctx, cmd.SubjectTags, scope); err != nil {
		return ActivityLogResponse{}, err
	}

	// Validate content_id reference if provided.
	if cmd.ContentID != nil {
		if _, err := s.activityDefRepo.FindByID(ctx, *cmd.ContentID); err != nil {
			return ActivityLogResponse{}, err
		}
	}

	attachments, _ := json.Marshal(cmd.Attachments)
	if cmd.Attachments == nil {
		attachments = json.RawMessage("[]")
	}

	// Resolve tool_id: accept UUID strings or ignore non-UUID slugs gracefully. [H9]
	var toolID *uuid.UUID
	if cmd.ToolSlug != nil {
		if parsed, parseErr := uuid.Parse(*cmd.ToolSlug); parseErr == nil {
			toolID = &parsed
		}
		// Non-UUID slugs (e.g. "nature-journal") are silently ignored — the tool
		// reference is informational and the activity is still logged correctly.
	}

	log := &ActivityLogModel{
		StudentID:       studentID,
		Title:           cmd.Title,
		Description:     cmd.Description,
		SubjectTags:     StringArray(cmd.SubjectTags),
		ContentID:       cmd.ContentID,
		MethodologyID:   cmd.MethodologyID,
		ToolID:          toolID,
		DurationMinutes: cmd.DurationMinutes,
		Attachments:     attachments,
		ActivityDate:    actDate,
	}

	if err := s.activityLogRepo.Create(ctx, scope, log); err != nil {
		return ActivityLogResponse{}, err
	}

	// Publish ActivityLogged event.
	_ = s.eventBus.Publish(ctx, ActivityLogged{
		FamilyID:        scope.FamilyID(),
		StudentID:       studentID,
		ActivityID:      log.ID,
		SubjectTags:     cmd.SubjectTags,
		DurationMinutes: cmd.DurationMinutes,
		ActivityDate:    actDate,
	})

	return activityLogToResponse(log), nil
}

func (s *learningServiceImpl) UpdateActivityLog(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID, cmd UpdateActivityLogCommand) (ActivityLogResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return ActivityLogResponse{}, err
	}

	log, err := s.activityLogRepo.FindByID(ctx, scope, logID)
	if err != nil {
		return ActivityLogResponse{}, err
	}
	if log.StudentID != studentID {
		return ActivityLogResponse{}, &LearningError{Err: domain.ErrActivityNotFound}
	}

	// Apply updates.
	if cmd.Title != nil {
		log.Title = *cmd.Title
	}
	if cmd.Description != nil {
		log.Description = cmd.Description
	}
	if cmd.SubjectTags != nil {
		log.SubjectTags = StringArray(*cmd.SubjectTags)
	}
	if cmd.DurationMinutes != nil {
		log.DurationMinutes = cmd.DurationMinutes
	}
	if cmd.Attachments != nil {
		attachments, _ := json.Marshal(*cmd.Attachments)
		log.Attachments = attachments
	}
	if cmd.ActivityDate != nil {
		log.ActivityDate = *cmd.ActivityDate
	}

	// Re-validate invariants.
	if _, valErr := domain.NewActivity(studentID, log.Title, []string(log.SubjectTags), log.ActivityDate, log.DurationMinutes); valErr != nil {
		return ActivityLogResponse{}, &LearningError{Err: valErr}
	}

	// Validate subject tags if changed.
	if cmd.SubjectTags != nil {
		if err := s.validateSubjectTags(ctx, *cmd.SubjectTags, scope); err != nil {
			return ActivityLogResponse{}, err
		}
	}

	log.UpdatedAt = time.Now()
	if err := s.activityLogRepo.Update(ctx, scope, log); err != nil {
		return ActivityLogResponse{}, err
	}
	return activityLogToResponse(log), nil
}

func (s *learningServiceImpl) DeleteActivityLog(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}
	log, err := s.activityLogRepo.FindByID(ctx, scope, logID)
	if err != nil {
		return err
	}
	if log.StudentID != studentID {
		return &LearningError{Err: domain.ErrActivityNotFound}
	}
	return s.activityLogRepo.Delete(ctx, scope, logID)
}

func (s *learningServiceImpl) ListActivityLogs(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ActivityLogQuery) (PaginatedResponse[ActivityLogResponse], error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return PaginatedResponse[ActivityLogResponse]{}, err
	}

	logs, err := s.activityLogRepo.ListByStudent(ctx, scope, studentID, &query)
	if err != nil {
		return PaginatedResponse[ActivityLogResponse]{}, err
	}

	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	hasMore := int64(len(logs)) > limit
	if hasMore {
		logs = logs[:limit]
	}

	items := make([]ActivityLogResponse, len(logs))
	for i, l := range logs {
		items[i] = activityLogToResponse(&l)
	}

	var nextCursor *uuid.UUID
	if hasMore && len(items) > 0 {
		id := logs[len(logs)-1].ID
		nextCursor = &id
	}

	return PaginatedResponse[ActivityLogResponse]{
		Data:       items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *learningServiceImpl) GetActivityLog(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, logID uuid.UUID) (ActivityLogResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return ActivityLogResponse{}, err
	}
	log, err := s.activityLogRepo.FindByID(ctx, scope, logID)
	if err != nil {
		return ActivityLogResponse{}, err
	}
	if log.StudentID != studentID {
		return ActivityLogResponse{}, &LearningError{Err: domain.ErrActivityNotFound}
	}
	return activityLogToResponse(log), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Model → Response Mappers
// ═══════════════════════════════════════════════════════════════════════════════

func activityDefToResponse(def *ActivityDefModel) ActivityDefResponse {
	var attachments []AttachmentInput
	_ = json.Unmarshal(def.Attachments, &attachments)
	if attachments == nil {
		attachments = []AttachmentInput{}
	}
	tags := []string(def.SubjectTags)
	if tags == nil {
		tags = []string{}
	}
	return ActivityDefResponse{
		ID:                 def.ID,
		PublisherID:        def.PublisherID,
		Title:              def.Title,
		Description:        def.Description,
		SubjectTags:        tags,
		MethodologyID:      def.MethodologyID,
		ToolID:             def.ToolID,
		EstDurationMinutes: def.EstDurationMinutes,
		Attachments:        attachments,
		CreatedAt:          def.CreatedAt,
		UpdatedAt:          def.UpdatedAt,
	}
}

func activityDefToSummary(def *ActivityDefModel) ActivityDefSummaryResponse {
	tags := []string(def.SubjectTags)
	if tags == nil {
		tags = []string{}
	}
	return ActivityDefSummaryResponse{
		ID:                 def.ID,
		Title:              def.Title,
		SubjectTags:        tags,
		MethodologyID:      def.MethodologyID,
		EstDurationMinutes: def.EstDurationMinutes,
	}
}

func activityLogToResponse(log *ActivityLogModel) ActivityLogResponse {
	var attachments []AttachmentInput
	_ = json.Unmarshal(log.Attachments, &attachments)
	if attachments == nil {
		attachments = []AttachmentInput{}
	}
	tags := []string(log.SubjectTags)
	if tags == nil {
		tags = []string{}
	}
	return ActivityLogResponse{
		ID:              log.ID,
		StudentID:       log.StudentID,
		Title:           log.Title,
		Description:     log.Description,
		SubjectTags:     tags,
		ContentID:       log.ContentID,
		MethodologyID:   log.MethodologyID,
		ToolID:          log.ToolID,
		DurationMinutes: log.DurationMinutes,
		Attachments:     attachments,
		ActivityDate:    log.ActivityDate,
		CreatedAt:       log.CreatedAt,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Reading Item CRUD (Layer 1)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateReadingItem(ctx context.Context, cmd CreateReadingItemCommand) (ReadingItemResponse, error) {
	if err := s.checkPublisherMembership(ctx, cmd.PublisherID, cmd.PublisherID); err != nil {
		return ReadingItemResponse{}, err
	}
	item := &ReadingItemModel{
		PublisherID:   cmd.PublisherID,
		Title:         cmd.Title,
		Author:        cmd.Author,
		ISBN:          cmd.ISBN,
		SubjectTags:   StringArray(cmd.SubjectTags),
		Description:   cmd.Description,
		CoverImageURL: cmd.CoverImageURL,
		PageCount:     cmd.PageCount,
	}
	if err := s.readingItemRepo.Create(ctx, item); err != nil {
		return ReadingItemResponse{}, err
	}
	return readingItemToResponse(item), nil
}

func (s *learningServiceImpl) UpdateReadingItem(ctx context.Context, itemID uuid.UUID, cmd UpdateReadingItemCommand) (ReadingItemResponse, error) {
	item, err := s.readingItemRepo.FindByID(ctx, itemID)
	if err != nil {
		return ReadingItemResponse{}, err
	}
	if cmd.Title != nil {
		item.Title = *cmd.Title
	}
	if cmd.Author != nil {
		item.Author = cmd.Author
	}
	if cmd.ISBN != nil {
		item.ISBN = cmd.ISBN
	}
	if cmd.SubjectTags != nil {
		item.SubjectTags = StringArray(*cmd.SubjectTags)
	}
	if cmd.Description != nil {
		item.Description = cmd.Description
	}
	if cmd.CoverImageURL != nil {
		item.CoverImageURL = cmd.CoverImageURL
	}
	if cmd.PageCount != nil {
		item.PageCount = cmd.PageCount
	}
	item.UpdatedAt = time.Now()
	if err := s.readingItemRepo.Update(ctx, item); err != nil {
		return ReadingItemResponse{}, err
	}
	return readingItemToResponse(item), nil
}

func (s *learningServiceImpl) ListReadingItems(ctx context.Context, query ReadingItemQuery) (PaginatedResponse[ReadingItemSummaryResponse], error) {
	items, err := s.readingItemRepo.List(ctx, &query)
	if err != nil {
		return PaginatedResponse[ReadingItemSummaryResponse]{}, err
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	hasMore := int64(len(items)) > limit
	if hasMore {
		items = items[:limit]
	}
	results := make([]ReadingItemSummaryResponse, len(items))
	for i, item := range items {
		results[i] = readingItemToSummary(&item)
	}
	var nextCursor *uuid.UUID
	if hasMore && len(results) > 0 {
		id := items[len(items)-1].ID
		nextCursor = &id
	}
	return PaginatedResponse[ReadingItemSummaryResponse]{Data: results, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (s *learningServiceImpl) GetReadingItem(ctx context.Context, itemID uuid.UUID) (ReadingItemDetailResponse, error) {
	item, err := s.readingItemRepo.FindByID(ctx, itemID)
	if err != nil {
		return ReadingItemDetailResponse{}, err
	}
	links, err := s.artifactLinkRepo.FindByContent(ctx, "reading_item", itemID, LinkDirectionBoth)
	if err != nil {
		return ReadingItemDetailResponse{}, err
	}
	linkedArtifacts := make([]ArtifactLinkResponse, len(links))
	for i, l := range links {
		linkedArtifacts[i] = artifactLinkToResponse(&l)
	}
	return ReadingItemDetailResponse{
		ReadingItemResponse: readingItemToResponse(item),
		LinkedArtifacts:     linkedArtifacts,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Reading Progress (Layer 3)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) StartReading(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartReadingCommand) (ReadingProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return ReadingProgressResponse{}, err
	}
	// Verify reading item exists.
	item, err := s.readingItemRepo.FindByID(ctx, cmd.ReadingItemID)
	if err != nil {
		return ReadingProgressResponse{}, err
	}
	// Prevent duplicate tracking.
	exists, err := s.readingProgressRepo.Exists(ctx, scope, studentID, cmd.ReadingItemID)
	if err != nil {
		return ReadingProgressResponse{}, err
	}
	if exists {
		return ReadingProgressResponse{}, &LearningError{Err: domain.ErrDuplicateReadingProgress}
	}
	now := time.Now()
	progress := &ReadingProgressModel{
		StudentID:     studentID,
		ReadingItemID: cmd.ReadingItemID,
		ReadingListID: cmd.ReadingListID,
		Status:        domain.ReadingStatusToRead,
		StartedAt:     &now,
	}
	if err := s.readingProgressRepo.Create(ctx, scope, progress); err != nil {
		return ReadingProgressResponse{}, err
	}
	return readingProgressToResponse(progress, item), nil
}

func (s *learningServiceImpl) UpdateReadingProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateReadingProgressCommand) (ReadingProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return ReadingProgressResponse{}, err
	}
	progress, err := s.readingProgressRepo.FindByID(ctx, scope, progressID)
	if err != nil {
		return ReadingProgressResponse{}, err
	}
	if progress.StudentID != studentID {
		return ReadingProgressResponse{}, &LearningError{Err: domain.ErrReadingProgressNotFound}
	}
	if cmd.Notes != nil {
		progress.Notes = cmd.Notes
	}
	if cmd.Status != nil {
		if err := domain.ValidateReadingStatusTransition(progress.Status, *cmd.Status); err != nil {
			return ReadingProgressResponse{}, &LearningError{Err: err}
		}
		progress.Status = *cmd.Status
		if *cmd.Status == domain.ReadingStatusCompleted {
			now := time.Now()
			progress.CompletedAt = &now
		}
	}
	progress.UpdatedAt = time.Now()
	if err := s.readingProgressRepo.Update(ctx, scope, progress); err != nil {
		return ReadingProgressResponse{}, err
	}
	// Look up the reading item for the response.
	item, err := s.readingItemRepo.FindByID(ctx, progress.ReadingItemID)
	if err != nil {
		return ReadingProgressResponse{}, err
	}
	// Publish BookCompleted event and auto-log activity on completion.
	if cmd.Status != nil && *cmd.Status == domain.ReadingStatusCompleted {
		_ = s.eventBus.Publish(ctx, BookCompleted{
			FamilyID:         scope.FamilyID(),
			StudentID:        studentID,
			ReadingItemID:    progress.ReadingItemID,
			ReadingItemTitle: item.Title,
		})
		// Auto-log activity for completed book.
		autoLog := &ActivityLogModel{
			StudentID:    studentID,
			Title:        "Completed: " + item.Title,
			SubjectTags:  item.SubjectTags,
			ContentID:    &progress.ReadingItemID,
			ActivityDate: time.Now().Truncate(24 * time.Hour),
			Attachments:  json.RawMessage("[]"),
		}
		_ = s.activityLogRepo.Create(ctx, scope, autoLog)
	}
	return readingProgressToResponse(progress, item), nil
}

func (s *learningServiceImpl) ListReadingProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ReadingProgressQuery) (PaginatedResponse[ReadingProgressResponse], error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return PaginatedResponse[ReadingProgressResponse]{}, err
	}
	records, err := s.readingProgressRepo.ListByStudent(ctx, scope, studentID, &query)
	if err != nil {
		return PaginatedResponse[ReadingProgressResponse]{}, err
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	hasMore := int64(len(records)) > limit
	if hasMore {
		records = records[:limit]
	}
	// Collect reading item IDs and batch-fetch them.
	itemIDs := make([]uuid.UUID, len(records))
	for i, r := range records {
		itemIDs[i] = r.ReadingItemID
	}
	items, err := s.readingItemRepo.FindByIDs(ctx, itemIDs)
	if err != nil {
		return PaginatedResponse[ReadingProgressResponse]{}, err
	}
	itemMap := make(map[uuid.UUID]*ReadingItemModel, len(items))
	for i := range items {
		itemMap[items[i].ID] = &items[i]
	}
	results := make([]ReadingProgressResponse, len(records))
	for i, r := range records {
		results[i] = readingProgressToResponse(&r, itemMap[r.ReadingItemID])
	}
	var nextCursor *uuid.UUID
	if hasMore && len(results) > 0 {
		id := records[len(records)-1].ID
		nextCursor = &id
	}
	return PaginatedResponse[ReadingProgressResponse]{Data: results, NextCursor: nextCursor, HasMore: hasMore}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Reading List CRUD (Layer 3)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateReadingList(ctx context.Context, scope *shared.FamilyScope, cmd CreateReadingListCommand) (ReadingListResponse, error) {
	list := &ReadingListModel{
		Name:        cmd.Name,
		Description: cmd.Description,
		StudentID:   cmd.StudentID,
	}
	if err := s.readingListRepo.Create(ctx, scope, list); err != nil {
		return ReadingListResponse{}, err
	}
	if len(cmd.ReadingItemIDs) > 0 {
		if err := s.readingListRepo.AddItems(ctx, scope, list.ID, cmd.ReadingItemIDs); err != nil {
			return ReadingListResponse{}, err
		}
	}
	return readingListToResponse(list), nil
}

func (s *learningServiceImpl) UpdateReadingList(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID, cmd UpdateReadingListCommand) (ReadingListResponse, error) {
	list, err := s.readingListRepo.FindByID(ctx, scope, listID)
	if err != nil {
		return ReadingListResponse{}, err
	}
	if cmd.Name != nil {
		list.Name = *cmd.Name
	}
	if cmd.Description != nil {
		list.Description = cmd.Description
	}
	list.UpdatedAt = time.Now()
	if err := s.readingListRepo.Update(ctx, scope, list); err != nil {
		return ReadingListResponse{}, err
	}
	if cmd.AddItemIDs != nil && len(*cmd.AddItemIDs) > 0 {
		if err := s.readingListRepo.AddItems(ctx, scope, listID, *cmd.AddItemIDs); err != nil {
			return ReadingListResponse{}, err
		}
	}
	if cmd.RemoveItemIDs != nil && len(*cmd.RemoveItemIDs) > 0 {
		if err := s.readingListRepo.RemoveItems(ctx, scope, listID, *cmd.RemoveItemIDs); err != nil {
			return ReadingListResponse{}, err
		}
	}
	return readingListToResponse(list), nil
}

func (s *learningServiceImpl) DeleteReadingList(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) error {
	return s.readingListRepo.Delete(ctx, scope, listID)
}

func (s *learningServiceImpl) ListReadingLists(ctx context.Context, scope *shared.FamilyScope) ([]ReadingListSummaryResponse, error) {
	lists, err := s.readingListRepo.ListByFamily(ctx, scope)
	if err != nil {
		return nil, err
	}
	results := make([]ReadingListSummaryResponse, len(lists))
	for i, l := range lists {
		// Count items per list.
		items, listErr := s.readingListRepo.ListItems(ctx, l.ID)
		if listErr != nil {
			return nil, listErr
		}
		results[i] = ReadingListSummaryResponse{
			ID:          l.ID,
			Name:        l.Name,
			Description: l.Description,
			StudentID:   l.StudentID,
			ItemCount:   int64(len(items)),
		}
	}
	return results, nil
}

func (s *learningServiceImpl) GetReadingList(ctx context.Context, scope *shared.FamilyScope, listID uuid.UUID) (ReadingListDetailResponse, error) {
	list, err := s.readingListRepo.FindByID(ctx, scope, listID)
	if err != nil {
		return ReadingListDetailResponse{}, err
	}
	listItems, err := s.readingListRepo.ListItems(ctx, listID)
	if err != nil {
		return ReadingListDetailResponse{}, err
	}
	// Batch-fetch reading items.
	itemIDs := make([]uuid.UUID, len(listItems))
	for i, li := range listItems {
		itemIDs[i] = li.ReadingItemID
	}
	items, err := s.readingItemRepo.FindByIDs(ctx, itemIDs)
	if err != nil {
		return ReadingListDetailResponse{}, err
	}
	itemMap := make(map[uuid.UUID]*ReadingItemModel, len(items))
	for i := range items {
		itemMap[items[i].ID] = &items[i]
	}
	respItems := make([]ReadingListItemWithProgress, len(listItems))
	for i, li := range listItems {
		item := itemMap[li.ReadingItemID]
		var summary ReadingItemSummaryResponse
		if item != nil {
			summary = readingItemToSummary(item)
		}
		respItems[i] = ReadingListItemWithProgress{
			ReadingItem: summary,
			SortOrder:   li.SortOrder,
		}
	}
	return ReadingListDetailResponse{
		ID:          list.ID,
		Name:        list.Name,
		Description: list.Description,
		StudentID:   list.StudentID,
		Items:       respItems,
		CreatedAt:   list.CreatedAt,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Journal Entry CRUD (Layer 3)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd CreateJournalEntryCommand) (JournalEntryResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return JournalEntryResponse{}, err
	}
	if err := domain.ValidateEntryType(cmd.EntryType); err != nil {
		return JournalEntryResponse{}, &LearningError{Err: err}
	}
	if err := s.validateSubjectTags(ctx, cmd.SubjectTags, scope); err != nil {
		return JournalEntryResponse{}, err
	}
	attachments, _ := json.Marshal(cmd.Attachments)
	if cmd.Attachments == nil {
		attachments = json.RawMessage("[]")
	}
	entryDate := defaultActivityDate(cmd.EntryDate)
	entry := &JournalEntryModel{
		StudentID:   studentID,
		EntryType:   cmd.EntryType,
		Title:       cmd.Title,
		Content:     cmd.Content,
		SubjectTags: StringArray(cmd.SubjectTags),
		ContentID:   cmd.ContentID,
		Attachments: attachments,
		EntryDate:   entryDate,
	}
	if err := s.journalEntryRepo.Create(ctx, scope, entry); err != nil {
		return JournalEntryResponse{}, err
	}
	return journalEntryToResponse(entry), nil
}

func (s *learningServiceImpl) UpdateJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID, cmd UpdateJournalEntryCommand) (JournalEntryResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return JournalEntryResponse{}, err
	}
	entry, err := s.journalEntryRepo.FindByID(ctx, scope, entryID)
	if err != nil {
		return JournalEntryResponse{}, err
	}
	if entry.StudentID != studentID {
		return JournalEntryResponse{}, &LearningError{Err: domain.ErrJournalNotFound}
	}
	if cmd.EntryType != nil {
		if err := domain.ValidateEntryType(*cmd.EntryType); err != nil {
			return JournalEntryResponse{}, &LearningError{Err: err}
		}
		entry.EntryType = *cmd.EntryType
	}
	if cmd.Title != nil {
		entry.Title = cmd.Title
	}
	if cmd.Content != nil {
		entry.Content = *cmd.Content
	}
	if cmd.SubjectTags != nil {
		if err := s.validateSubjectTags(ctx, *cmd.SubjectTags, scope); err != nil {
			return JournalEntryResponse{}, err
		}
		entry.SubjectTags = StringArray(*cmd.SubjectTags)
	}
	if cmd.Attachments != nil {
		attachments, _ := json.Marshal(*cmd.Attachments)
		entry.Attachments = attachments
	}
	if cmd.EntryDate != nil {
		entry.EntryDate = *cmd.EntryDate
	}
	entry.UpdatedAt = time.Now()
	if err := s.journalEntryRepo.Update(ctx, scope, entry); err != nil {
		return JournalEntryResponse{}, err
	}
	return journalEntryToResponse(entry), nil
}

func (s *learningServiceImpl) DeleteJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}
	entry, err := s.journalEntryRepo.FindByID(ctx, scope, entryID)
	if err != nil {
		return err
	}
	if entry.StudentID != studentID {
		return &LearningError{Err: domain.ErrJournalNotFound}
	}
	return s.journalEntryRepo.Delete(ctx, scope, entryID)
}

func (s *learningServiceImpl) ListJournalEntries(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query JournalEntryQuery) (PaginatedResponse[JournalEntryResponse], error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return PaginatedResponse[JournalEntryResponse]{}, err
	}
	entries, err := s.journalEntryRepo.ListByStudent(ctx, scope, studentID, &query)
	if err != nil {
		return PaginatedResponse[JournalEntryResponse]{}, err
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	hasMore := int64(len(entries)) > limit
	if hasMore {
		entries = entries[:limit]
	}
	results := make([]JournalEntryResponse, len(entries))
	for i, e := range entries {
		results[i] = journalEntryToResponse(&e)
	}
	var nextCursor *uuid.UUID
	if hasMore && len(results) > 0 {
		id := entries[len(entries)-1].ID
		nextCursor = &id
	}
	return PaginatedResponse[JournalEntryResponse]{Data: results, NextCursor: nextCursor, HasMore: hasMore}, nil
}

func (s *learningServiceImpl) GetJournalEntry(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, entryID uuid.UUID) (JournalEntryResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return JournalEntryResponse{}, err
	}
	entry, err := s.journalEntryRepo.FindByID(ctx, scope, entryID)
	if err != nil {
		return JournalEntryResponse{}, err
	}
	if entry.StudentID != studentID {
		return JournalEntryResponse{}, &LearningError{Err: domain.ErrJournalNotFound}
	}
	return journalEntryToResponse(entry), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Subject Taxonomy + Custom Subjects
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) GetSubjectTaxonomy(ctx context.Context, scope *shared.FamilyScope, query TaxonomyQuery) ([]SubjectTaxonomyResponse, error) {
	nodes, err := s.taxonomyRepo.List(ctx, &query)
	if err != nil {
		return nil, err
	}
	// Convert platform taxonomy nodes.
	results := make([]SubjectTaxonomyResponse, len(nodes))
	for i, n := range nodes {
		results[i] = SubjectTaxonomyResponse{
			ID:       n.ID,
			ParentID: n.ParentID,
			Name:     n.Name,
			Slug:     n.Slug,
			Level:    n.Level,
			Children: []SubjectTaxonomyResponse{},
			IsCustom: false,
		}
	}
	// Append custom subjects.
	customs, err := s.taxonomyRepo.ListCustomSubjects(ctx, scope)
	if err != nil {
		return nil, err
	}
	for _, c := range customs {
		results = append(results, SubjectTaxonomyResponse{
			ID:       c.ID,
			ParentID: c.ParentTaxonomyID,
			Name:     c.Name,
			Slug:     c.Slug,
			Level:    3, // custom subjects are leaf level
			Children: []SubjectTaxonomyResponse{},
			IsCustom: true,
		})
	}

	// Build tree: index by ID, then attach children to parents.
	byID := make(map[uuid.UUID]*SubjectTaxonomyResponse, len(results))
	for i := range results {
		byID[results[i].ID] = &results[i]
	}
	var roots []SubjectTaxonomyResponse
	for i := range results {
		node := &results[i]
		if node.ParentID != nil {
			if parent, ok := byID[*node.ParentID]; ok {
				parent.Children = append(parent.Children, *node)
				continue
			}
		}
		roots = append(roots, *node)
	}
	// Sort roots and children alphabetically for consistent ordering.
	sort.Slice(roots, func(i, j int) bool { return roots[i].Name < roots[j].Name })
	for i := range roots {
		sort.Slice(roots[i].Children, func(a, b int) bool {
			return roots[i].Children[a].Name < roots[i].Children[b].Name
		})
	}
	return roots, nil
}

func (s *learningServiceImpl) CreateCustomSubject(ctx context.Context, scope *shared.FamilyScope, cmd CreateCustomSubjectCommand) (CustomSubjectResponse, error) {
	slug := domain.Slugify(cmd.Name)
	// Check for duplicates: both platform taxonomy and family custom subjects.
	_, err := s.taxonomyRepo.FindBySlug(ctx, slug)
	if err != nil && !errors.Is(err, domain.ErrTaxonomyNotFound) {
		return CustomSubjectResponse{}, err
	}
	if err == nil {
		return CustomSubjectResponse{}, &LearningError{Err: domain.ErrDuplicateCustomSubject}
	}
	customs, err := s.taxonomyRepo.ListCustomSubjects(ctx, scope)
	if err != nil {
		return CustomSubjectResponse{}, err
	}
	for _, c := range customs {
		if c.Slug == slug {
			return CustomSubjectResponse{}, &LearningError{Err: domain.ErrDuplicateCustomSubject}
		}
	}
	subject := &CustomSubjectModel{
		ParentTaxonomyID: cmd.ParentTaxonomyID,
		Name:             cmd.Name,
		Slug:             slug,
	}
	if err := s.taxonomyRepo.CreateCustomSubject(ctx, scope, subject); err != nil {
		return CustomSubjectResponse{}, err
	}
	return CustomSubjectResponse{
		ID:               subject.ID,
		Name:             subject.Name,
		Slug:             subject.Slug,
		ParentTaxonomyID: subject.ParentTaxonomyID,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Model → Response Mappers (Reading / Journal / Taxonomy)
// ═══════════════════════════════════════════════════════════════════════════════

func readingItemToResponse(item *ReadingItemModel) ReadingItemResponse {
	tags := []string(item.SubjectTags)
	if tags == nil {
		tags = []string{}
	}
	return ReadingItemResponse{
		ID:            item.ID,
		PublisherID:   item.PublisherID,
		Title:         item.Title,
		Author:        item.Author,
		ISBN:          item.ISBN,
		SubjectTags:   tags,
		Description:   item.Description,
		CoverImageURL: item.CoverImageURL,
		PageCount:     item.PageCount,
		CreatedAt:     item.CreatedAt,
	}
}

func readingItemToSummary(item *ReadingItemModel) ReadingItemSummaryResponse {
	tags := []string(item.SubjectTags)
	if tags == nil {
		tags = []string{}
	}
	return ReadingItemSummaryResponse{
		ID:            item.ID,
		Title:         item.Title,
		Author:        item.Author,
		SubjectTags:   tags,
		CoverImageURL: item.CoverImageURL,
	}
}

func readingProgressToResponse(p *ReadingProgressModel, item *ReadingItemModel) ReadingProgressResponse {
	var itemSummary ReadingItemSummaryResponse
	if item != nil {
		itemSummary = readingItemToSummary(item)
	}
	return ReadingProgressResponse{
		ID:            p.ID,
		StudentID:     p.StudentID,
		ReadingItem:   itemSummary,
		ReadingListID: p.ReadingListID,
		Status:        p.Status,
		StartedAt:     p.StartedAt,
		CompletedAt:   p.CompletedAt,
		Notes:         p.Notes,
	}
}

func readingListToResponse(list *ReadingListModel) ReadingListResponse {
	return ReadingListResponse{
		ID:          list.ID,
		Name:        list.Name,
		Description: list.Description,
		StudentID:   list.StudentID,
		CreatedAt:   list.CreatedAt,
	}
}

func journalEntryToResponse(entry *JournalEntryModel) JournalEntryResponse {
	var attachments []AttachmentInput
	_ = json.Unmarshal(entry.Attachments, &attachments)
	if attachments == nil {
		attachments = []AttachmentInput{}
	}
	tags := []string(entry.SubjectTags)
	if tags == nil {
		tags = []string{}
	}
	return JournalEntryResponse{
		ID:          entry.ID,
		StudentID:   entry.StudentID,
		EntryType:   entry.EntryType,
		Title:       entry.Title,
		Content:     entry.Content,
		SubjectTags: tags,
		Attachments: attachments,
		EntryDate:   entry.EntryDate,
		CreatedAt:   entry.CreatedAt,
	}
}


// ═══════════════════════════════════════════════════════════════════════════════
// Artifact Links (Layer 1 — polymorphic)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) LinkArtifacts(ctx context.Context, cmd CreateArtifactLinkCommand) (ArtifactLinkResponse, error) {
	relationship := "about"
	if cmd.Relationship != nil && *cmd.Relationship != "" {
		relationship = *cmd.Relationship
	}
	link := &ArtifactLinkModel{
		SourceType:   cmd.SourceType,
		SourceID:     cmd.SourceID,
		TargetType:   cmd.TargetType,
		TargetID:     cmd.TargetID,
		Relationship: relationship,
	}
	if err := s.artifactLinkRepo.Create(ctx, link); err != nil {
		return ArtifactLinkResponse{}, err
	}
	return artifactLinkToResponse(link), nil
}

func (s *learningServiceImpl) UnlinkArtifacts(ctx context.Context, linkID uuid.UUID, _ uuid.UUID) error {
	return s.artifactLinkRepo.Delete(ctx, linkID)
}

func (s *learningServiceImpl) GetLinkedArtifacts(ctx context.Context, contentType string, contentID uuid.UUID, direction LinkDirection) ([]ArtifactLinkResponse, error) {
	links, err := s.artifactLinkRepo.FindByContent(ctx, contentType, contentID, direction)
	if err != nil {
		return nil, err
	}
	results := make([]ArtifactLinkResponse, len(links))
	for i, l := range links {
		results[i] = artifactLinkToResponse(&l)
	}
	return results, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Data Export (Layer 3)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) RequestDataExport(ctx context.Context, scope *shared.FamilyScope, _ RequestExportCommand) (ExportRequestResponse, error) {
	active, err := s.exportRepo.HasActiveExport(ctx, scope)
	if err != nil {
		return ExportRequestResponse{}, err
	}
	if active {
		return ExportRequestResponse{}, &LearningError{Err: domain.ErrExportAlreadyInProgress}
	}
	req := &ExportRequestModel{
		RequestedBy: scope.FamilyID(),
		Status:      "pending",
	}
	if err := s.exportRepo.Create(ctx, scope, req); err != nil {
		return ExportRequestResponse{}, err
	}
	return exportRequestToResponse(req), nil
}

func (s *learningServiceImpl) GetExportRequest(ctx context.Context, scope *shared.FamilyScope, exportID uuid.UUID) (ExportRequestResponse, error) {
	req, err := s.exportRepo.FindByID(ctx, scope, exportID)
	if err != nil {
		return ExportRequestResponse{}, err
	}
	return exportRequestToResponse(req), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Question CRUD (Layer 1)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateQuestion(ctx context.Context, cmd CreateQuestionCommand) (QuestionResponse, error) {
	autoScorable := cmd.QuestionType != "short_answer"
	points := 1.0
	if cmd.Points != nil {
		points = *cmd.Points
	}
	attachments := cmd.MediaAttachments
	if attachments == nil {
		attachments = json.RawMessage("[]")
	}
	q := &QuestionModel{
		PublisherID:      cmd.PublisherID,
		QuestionType:     cmd.QuestionType,
		Content:          cmd.Content,
		MediaAttachments: attachments,
		AnswerData:       cmd.AnswerData,
		SubjectTags:      StringArray(cmd.SubjectTags),
		MethodologyID:    cmd.MethodologyID,
		DifficultyLevel:  cmd.DifficultyLevel,
		AutoScorable:     autoScorable,
		Points:           points,
	}
	if err := s.questionRepo.Create(ctx, q); err != nil {
		return QuestionResponse{}, err
	}
	return questionToResponse(q), nil
}

func (s *learningServiceImpl) UpdateQuestion(ctx context.Context, questionID uuid.UUID, cmd UpdateQuestionCommand) (QuestionResponse, error) {
	q, err := s.questionRepo.FindByID(ctx, questionID)
	if err != nil {
		return QuestionResponse{}, err
	}
	if cmd.Content != nil {
		q.Content = *cmd.Content
	}
	if cmd.MediaAttachments != nil {
		q.MediaAttachments = *cmd.MediaAttachments
	}
	if cmd.AnswerData != nil {
		q.AnswerData = *cmd.AnswerData
	}
	if cmd.SubjectTags != nil {
		q.SubjectTags = StringArray(*cmd.SubjectTags)
	}
	if cmd.DifficultyLevel != nil {
		q.DifficultyLevel = cmd.DifficultyLevel
	}
	if cmd.Points != nil {
		q.Points = *cmd.Points
	}
	q.UpdatedAt = time.Now()
	if err := s.questionRepo.Update(ctx, q); err != nil {
		return QuestionResponse{}, err
	}
	return questionToResponse(q), nil
}

func (s *learningServiceImpl) ListQuestions(ctx context.Context, query QuestionQuery) (PaginatedResponse[QuestionSummaryResponse], error) {
	questions, err := s.questionRepo.List(ctx, &query)
	if err != nil {
		return PaginatedResponse[QuestionSummaryResponse]{}, err
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	hasMore := int64(len(questions)) > limit
	if hasMore {
		questions = questions[:limit]
	}
	results := make([]QuestionSummaryResponse, len(questions))
	for i, q := range questions {
		results[i] = questionToSummary(&q)
	}
	var nextCursor *uuid.UUID
	if hasMore && len(results) > 0 {
		id := questions[len(questions)-1].ID
		nextCursor = &id
	}
	return PaginatedResponse[QuestionSummaryResponse]{Data: results, NextCursor: nextCursor, HasMore: hasMore}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Quiz Definition CRUD (Layer 1)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateQuizDef(ctx context.Context, cmd CreateQuizDefCommand) (QuizDefResponse, error) {
	if len(cmd.QuestionIDs) == 0 {
		return QuizDefResponse{}, &LearningError{Err: domain.ErrQuizNoQuestions}
	}
	passingPercent := int16(70)
	if cmd.PassingScorePercent != nil {
		passingPercent = *cmd.PassingScorePercent
	}
	shuffleQuestions := false
	if cmd.ShuffleQuestions != nil {
		shuffleQuestions = *cmd.ShuffleQuestions
	}
	showCorrectAfter := true
	if cmd.ShowCorrectAfter != nil {
		showCorrectAfter = *cmd.ShowCorrectAfter
	}
	def := &QuizDefModel{
		PublisherID:         cmd.PublisherID,
		Title:               cmd.Title,
		Description:         cmd.Description,
		SubjectTags:         StringArray(cmd.SubjectTags),
		MethodologyID:       cmd.MethodologyID,
		TimeLimitMinutes:    cmd.TimeLimitMinutes,
		PassingScorePercent: passingPercent,
		ShuffleQuestions:    shuffleQuestions,
		ShowCorrectAfter:    showCorrectAfter,
		QuestionCount:       int16(len(cmd.QuestionIDs)),
	}
	if err := s.quizDefRepo.Create(ctx, def); err != nil {
		return QuizDefResponse{}, err
	}
	quizQuestions := make([]QuizQuestionModel, len(cmd.QuestionIDs))
	for i, qi := range cmd.QuestionIDs {
		quizQuestions[i] = QuizQuestionModel{
			QuizDefID:      def.ID,
			QuestionID:     qi.QuestionID,
			SortOrder:      qi.SortOrder,
			PointsOverride: qi.PointsOverride,
		}
	}
	if err := s.quizDefRepo.SetQuestions(ctx, def.ID, quizQuestions); err != nil {
		return QuizDefResponse{}, err
	}
	return quizDefToResponse(def), nil
}

func (s *learningServiceImpl) UpdateQuizDef(ctx context.Context, quizDefID uuid.UUID, cmd UpdateQuizDefCommand) (QuizDefResponse, error) {
	def, err := s.quizDefRepo.FindByID(ctx, quizDefID)
	if err != nil {
		return QuizDefResponse{}, err
	}
	if cmd.Title != nil {
		def.Title = *cmd.Title
	}
	if cmd.Description != nil {
		def.Description = cmd.Description
	}
	if cmd.SubjectTags != nil {
		def.SubjectTags = StringArray(*cmd.SubjectTags)
	}
	if cmd.TimeLimitMinutes != nil {
		def.TimeLimitMinutes = cmd.TimeLimitMinutes
	}
	if cmd.PassingScorePercent != nil {
		def.PassingScorePercent = *cmd.PassingScorePercent
	}
	if cmd.ShuffleQuestions != nil {
		def.ShuffleQuestions = *cmd.ShuffleQuestions
	}
	if cmd.ShowCorrectAfter != nil {
		def.ShowCorrectAfter = *cmd.ShowCorrectAfter
	}
	def.UpdatedAt = time.Now()
	if err := s.quizDefRepo.Update(ctx, def); err != nil {
		return QuizDefResponse{}, err
	}
	if cmd.QuestionIDs != nil {
		questions := *cmd.QuestionIDs
		quizQuestions := make([]QuizQuestionModel, len(questions))
		for i, qi := range questions {
			quizQuestions[i] = QuizQuestionModel{
				QuizDefID:      def.ID,
				QuestionID:     qi.QuestionID,
				SortOrder:      qi.SortOrder,
				PointsOverride: qi.PointsOverride,
			}
		}
		if err := s.quizDefRepo.SetQuestions(ctx, def.ID, quizQuestions); err != nil {
			return QuizDefResponse{}, err
		}
		def.QuestionCount = int16(len(questions))
	}
	return quizDefToResponse(def), nil
}

func (s *learningServiceImpl) GetQuizDef(ctx context.Context, quizDefID uuid.UUID, includeAnswers bool) (QuizDefDetailResponse, error) {
	def, err := s.quizDefRepo.FindByID(ctx, quizDefID)
	if err != nil {
		return QuizDefDetailResponse{}, err
	}
	quizQuestions, err := s.quizDefRepo.ListQuestions(ctx, quizDefID)
	if err != nil {
		return QuizDefDetailResponse{}, err
	}
	// Batch-fetch question data.
	questionIDs := make([]uuid.UUID, len(quizQuestions))
	for i, qq := range quizQuestions {
		questionIDs[i] = qq.QuestionID
	}
	questions, err := s.questionRepo.FindByIDs(ctx, questionIDs)
	if err != nil {
		return QuizDefDetailResponse{}, err
	}
	questionMap := make(map[uuid.UUID]*QuestionModel, len(questions))
	for i := range questions {
		questionMap[questions[i].ID] = &questions[i]
	}
	respQuestions := make([]QuizQuestionResponse, len(quizQuestions))
	for i, qq := range quizQuestions {
		q := questionMap[qq.QuestionID]
		var pointsVal float64
		if qq.PointsOverride != nil {
			pointsVal = *qq.PointsOverride
		} else if q != nil {
			pointsVal = q.Points
		}
		var answerData json.RawMessage
		if q != nil && len(q.AnswerData) > 0 {
			if includeAnswers {
				answerData = q.AnswerData
			} else {
				// Strip correct_answer but keep choices so students can see options.
				answerData = stripCorrectAnswer(q.AnswerData)
			}
		}
		var qType, content string
		var autoScorable bool
		if q != nil {
			qType = q.QuestionType
			content = q.Content
			autoScorable = q.AutoScorable
		}
		respQuestions[i] = QuizQuestionResponse{
			QuestionID:   qq.QuestionID,
			SortOrder:    qq.SortOrder,
			Points:       pointsVal,
			QuestionType: qType,
			Content:      content,
			AnswerData:   answerData,
			AutoScorable: autoScorable,
		}
	}
	return QuizDefDetailResponse{
		QuizDefResponse: quizDefToResponse(def),
		Questions:       respQuestions,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Quiz Sessions (Layer 3)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) StartQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartQuizSessionCommand) (QuizSessionResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return QuizSessionResponse{}, err
	}
	if _, err := s.quizDefRepo.FindByID(ctx, cmd.QuizDefID); err != nil {
		return QuizSessionResponse{}, err
	}
	now := time.Now()
	session := &QuizSessionModel{
		StudentID: studentID,
		QuizDefID: cmd.QuizDefID,
		Status:    "in_progress",
		StartedAt: &now,
		Answers:   json.RawMessage("[]"),
	}
	if err := s.quizSessionRepo.Create(ctx, scope, session); err != nil {
		return QuizSessionResponse{}, err
	}
	return quizSessionToResponse(session), nil
}

func (s *learningServiceImpl) UpdateQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd UpdateQuizSessionCommand) (QuizSessionResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return QuizSessionResponse{}, err
	}
	session, err := s.quizSessionRepo.FindByID(ctx, scope, sessionID)
	if err != nil {
		return QuizSessionResponse{}, err
	}
	if session.StudentID != studentID {
		return QuizSessionResponse{}, &LearningError{Err: domain.ErrQuizSessionNotFound}
	}
	if session.Status == "submitted" || session.Status == "scored" {
		return QuizSessionResponse{}, &LearningError{Err: domain.ErrQuizSessionAlreadySubmitted}
	}
	if cmd.Answers != nil {
		session.Answers = cmd.Answers
	}
	// Submit and auto-score if requested.
	if cmd.Submit != nil && *cmd.Submit {
		now := time.Now()
		session.SubmittedAt = &now
		quizQuestions, qerr := s.quizDefRepo.ListQuestions(ctx, session.QuizDefID)
		if qerr != nil {
			return QuizSessionResponse{}, qerr
		}
		questionIDs := make([]uuid.UUID, len(quizQuestions))
		for i, qq := range quizQuestions {
			questionIDs[i] = qq.QuestionID
		}
		questions, qerr := s.questionRepo.FindByIDs(ctx, questionIDs)
		if qerr != nil {
			return QuizSessionResponse{}, qerr
		}
		questionMap := make(map[uuid.UUID]*QuestionModel, len(questions))
		for i := range questions {
			questionMap[questions[i].ID] = &questions[i]
		}
		infos := buildQuizQuestionInfos(quizQuestions, questionMap)
		maxScore, allAutoScorable := domain.AutoScoreQuiz(infos)
		session.MaxScore = &maxScore
		if allAutoScorable {
			// All auto-scorable: award full score and mark as scored.
			session.Score = &maxScore
			session.Status = "scored"
			scoredAt := time.Now()
			session.ScoredAt = &scoredAt
			passed := maxScore > 0
			session.Passed = &passed
			_ = s.eventBus.Publish(ctx, QuizCompleted{
				FamilyID:      scope.FamilyID(),
				StudentID:     studentID,
				QuizDefID:     session.QuizDefID,
				QuizSessionID: session.ID,
				Score:         maxScore,
				MaxScore:      maxScore,
				Passed:        passed,
			})
		} else {
			session.Status = "submitted"
		}
	}
	session.UpdatedAt = time.Now()
	if err := s.quizSessionRepo.Update(ctx, scope, session); err != nil {
		return QuizSessionResponse{}, err
	}
	return quizSessionToResponse(session), nil
}

func (s *learningServiceImpl) ScoreQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID, cmd ScoreQuizCommand) (QuizSessionResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return QuizSessionResponse{}, err
	}
	session, err := s.quizSessionRepo.FindByID(ctx, scope, sessionID)
	if err != nil {
		return QuizSessionResponse{}, err
	}
	if session.StudentID != studentID {
		return QuizSessionResponse{}, &LearningError{Err: domain.ErrQuizSessionNotFound}
	}
	if session.Status != "submitted" {
		return QuizSessionResponse{}, &LearningError{Err: domain.ErrQuizSessionNotSubmitted}
	}
	// Compute total score from auto-scored + parent-scored questions.
	quizQuestions, err := s.quizDefRepo.ListQuestions(ctx, session.QuizDefID)
	if err != nil {
		return QuizSessionResponse{}, err
	}
	questionIDs := make([]uuid.UUID, len(quizQuestions))
	for i, qq := range quizQuestions {
		questionIDs[i] = qq.QuestionID
	}
	questions, err := s.questionRepo.FindByIDs(ctx, questionIDs)
	if err != nil {
		return QuizSessionResponse{}, err
	}
	questionMap := make(map[uuid.UUID]*QuestionModel, len(questions))
	for i := range questions {
		questionMap[questions[i].ID] = &questions[i]
	}
	infos := buildQuizQuestionInfos(quizQuestions, questionMap)
	parentScores := make(map[uuid.UUID]float64, len(cmd.Scores))
	for _, sc := range cmd.Scores {
		parentScores[sc.QuestionID] = sc.PointsAwarded
	}
	totalScore, maxScore := domain.ComputeParentScore(infos, parentScores)
	now := time.Now()
	session.Score = &totalScore
	session.MaxScore = &maxScore
	session.ScoredAt = &now
	session.Status = "scored"
	def, err := s.quizDefRepo.FindByID(ctx, session.QuizDefID)
	if err != nil {
		return QuizSessionResponse{}, err
	}
	passed := maxScore > 0 && (totalScore/maxScore*100) >= float64(def.PassingScorePercent)
	session.Passed = &passed
	session.UpdatedAt = now
	if err := s.quizSessionRepo.Update(ctx, scope, session); err != nil {
		return QuizSessionResponse{}, err
	}
	_ = s.eventBus.Publish(ctx, QuizCompleted{
		FamilyID:      scope.FamilyID(),
		StudentID:     studentID,
		QuizDefID:     session.QuizDefID,
		QuizSessionID: session.ID,
		Score:         totalScore,
		MaxScore:      maxScore,
		Passed:        passed,
	})
	return quizSessionToResponse(session), nil
}

func (s *learningServiceImpl) GetQuizSession(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, sessionID uuid.UUID) (QuizSessionResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return QuizSessionResponse{}, err
	}
	session, err := s.quizSessionRepo.FindByID(ctx, scope, sessionID)
	if err != nil {
		return QuizSessionResponse{}, err
	}
	if session.StudentID != studentID {
		return QuizSessionResponse{}, &LearningError{Err: domain.ErrQuizSessionNotFound}
	}
	return quizSessionToResponse(session), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Progress Queries (computed on-the-fly from raw data)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) GetProgressSummary(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ProgressQuery) (ProgressSummaryResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return ProgressSummaryResponse{}, err
	}
	dateFrom, dateTo := domain.DefaultDateRange(query.DateFrom, query.DateTo)
	totalActivities, err := s.activityLogRepo.CountByStudentDateRange(ctx, scope, studentID, dateFrom, dateTo)
	if err != nil {
		return ProgressSummaryResponse{}, err
	}
	hoursBySubject, err := s.activityLogRepo.HoursBySubject(ctx, scope, studentID, dateFrom, dateTo)
	if err != nil {
		return ProgressSummaryResponse{}, err
	}
	var totalHours float64
	subjectHours := make([]SubjectHoursResponse, len(hoursBySubject))
	for i, h := range hoursBySubject {
		hours := float64(h.TotalMinutes) / 60.0
		totalHours += hours
		subjectHours[i] = SubjectHoursResponse{
			SubjectSlug: h.SubjectSlug,
			SubjectName: h.SubjectSlug,
			Hours:       hours,
		}
	}
	booksCompleted, err := s.readingProgressRepo.CountCompleted(ctx, scope, studentID, dateFrom, dateTo)
	if err != nil {
		return ProgressSummaryResponse{}, err
	}
	journalEntries, err := s.journalEntryRepo.CountByStudentDateRange(ctx, scope, studentID, dateFrom, dateTo)
	if err != nil {
		return ProgressSummaryResponse{}, err
	}
	return ProgressSummaryResponse{
		StudentID:       studentID,
		DateFrom:        dateFrom,
		DateTo:          dateTo,
		TotalActivities: totalActivities,
		TotalHours:      totalHours,
		HoursBySubject:  subjectHours,
		BooksCompleted:  booksCompleted,
		JournalEntries:  journalEntries,
	}, nil
}

func (s *learningServiceImpl) GetSubjectBreakdown(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ProgressQuery) ([]SubjectProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}
	dateFrom, dateTo := domain.DefaultDateRange(query.DateFrom, query.DateTo)
	hoursBySubject, err := s.activityLogRepo.HoursBySubject(ctx, scope, studentID, dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	results := make([]SubjectProgressResponse, len(hoursBySubject))
	for i, h := range hoursBySubject {
		results[i] = SubjectProgressResponse{
			SubjectSlug: h.SubjectSlug,
			SubjectName: h.SubjectSlug,
			TotalHours:  float64(h.TotalMinutes) / 60.0,
		}
	}
	return results, nil
}

func (s *learningServiceImpl) GetActivityTimeline(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query TimelineQuery) (PaginatedResponse[TimelineEntryResponse], error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return PaginatedResponse[TimelineEntryResponse]{}, err
	}
	dateFrom, dateTo := domain.DefaultDateRange(query.DateFrom, query.DateTo)
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	// Fetch activity logs in date range.
	logQuery := ActivityLogQuery{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		Limit:    limit + 1,
		Cursor:   query.Cursor,
	}
	logs, err := s.activityLogRepo.ListByStudent(ctx, scope, studentID, &logQuery)
	if err != nil {
		return PaginatedResponse[TimelineEntryResponse]{}, err
	}
	var entries []TimelineEntryResponse
	for _, l := range logs {
		tags := []string(l.SubjectTags)
		if tags == nil {
			tags = []string{}
		}
		entries = append(entries, TimelineEntryResponse{
			ID:          l.ID,
			EntryType:   "activity",
			Title:       l.Title,
			Description: l.Description,
			SubjectTags: tags,
			Date:        l.ActivityDate,
			CreatedAt:   l.CreatedAt,
		})
	}
	// Fetch journal entries in date range.
	journalQuery := JournalEntryQuery{
		DateFrom: &dateFrom,
		DateTo:   &dateTo,
		Limit:    limit + 1,
		Cursor:   query.Cursor,
	}
	journals, err := s.journalEntryRepo.ListByStudent(ctx, scope, studentID, &journalQuery)
	if err != nil {
		return PaginatedResponse[TimelineEntryResponse]{}, err
	}
	for _, j := range journals {
		tags := []string(j.SubjectTags)
		if tags == nil {
			tags = []string{}
		}
		title := j.EntryType
		if j.Title != nil {
			title = *j.Title
		}
		entries = append(entries, TimelineEntryResponse{
			ID:          j.ID,
			EntryType:   "journal",
			Title:       title,
			SubjectTags: tags,
			Date:        j.EntryDate,
			CreatedAt:   j.CreatedAt,
		})
	}
	// Sort merged timeline by date descending.
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date.After(entries[j].Date)
	})
	hasMore := int64(len(entries)) > limit
	if int64(len(entries)) > limit {
		entries = entries[:limit]
	}
	var nextCursor *uuid.UUID
	if hasMore && len(entries) > 0 {
		id := entries[len(entries)-1].ID
		nextCursor = &id
	}
	return PaginatedResponse[TimelineEntryResponse]{Data: entries, NextCursor: nextCursor, HasMore: hasMore}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Tools (delegation to method:: adapter)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) GetResolvedTools(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error) {
	return s.method.ResolveFamilyTools(ctx, scope)
}

func (s *learningServiceImpl) GetStudentTools(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return nil, err
	}
	return s.method.ResolveStudentTools(ctx, scope, studentID)
}


// ═══════════════════════════════════════════════════════════════════════════════
// Model → Response Mappers (Batch 5-6)
// ═══════════════════════════════════════════════════════════════════════════════

func artifactLinkToResponse(link *ArtifactLinkModel) ArtifactLinkResponse {
	return ArtifactLinkResponse{
		ID:           link.ID,
		SourceType:   link.SourceType,
		SourceID:     link.SourceID,
		TargetType:   link.TargetType,
		TargetID:     link.TargetID,
		Relationship: link.Relationship,
		CreatedAt:    link.CreatedAt,
	}
}

func questionToResponse(q *QuestionModel) QuestionResponse {
	tags := []string(q.SubjectTags)
	if tags == nil {
		tags = []string{}
	}
	return QuestionResponse{
		ID:               q.ID,
		PublisherID:       q.PublisherID,
		QuestionType:      q.QuestionType,
		Content:           q.Content,
		MediaAttachments:  q.MediaAttachments,
		AnswerData:        q.AnswerData,
		SubjectTags:       tags,
		MethodologyID:     q.MethodologyID,
		DifficultyLevel:   q.DifficultyLevel,
		AutoScorable:      q.AutoScorable,
		Points:            q.Points,
		CreatedAt:         q.CreatedAt,
	}
}

func questionToSummary(q *QuestionModel) QuestionSummaryResponse {
	tags := []string(q.SubjectTags)
	if tags == nil {
		tags = []string{}
	}
	return QuestionSummaryResponse{
		ID:              q.ID,
		QuestionType:    q.QuestionType,
		Content:         q.Content,
		SubjectTags:     tags,
		DifficultyLevel: q.DifficultyLevel,
		Points:          q.Points,
		AutoScorable:    q.AutoScorable,
		MethodologyID:   q.MethodologyID,
	}
}

// stripCorrectAnswer removes the correct_answer key from answer_data JSON
// so students can see answer choices without the solution.
func stripCorrectAnswer(raw json.RawMessage) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return raw // not an object — return as-is
	}
	delete(m, "correct_answer")
	delete(m, "correct")
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return out
}

func quizDefToResponse(def *QuizDefModel) QuizDefResponse {
	tags := []string(def.SubjectTags)
	if tags == nil {
		tags = []string{}
	}
	return QuizDefResponse{
		ID:                  def.ID,
		PublisherID:         def.PublisherID,
		Title:               def.Title,
		Description:         def.Description,
		SubjectTags:         tags,
		MethodologyID:       def.MethodologyID,
		TimeLimitMinutes:    def.TimeLimitMinutes,
		PassingScorePercent: def.PassingScorePercent,
		ShuffleQuestions:    def.ShuffleQuestions,
		ShowCorrectAfter:    def.ShowCorrectAfter,
		QuestionCount:       def.QuestionCount,
		CreatedAt:           def.CreatedAt,
	}
}

func quizSessionToResponse(session *QuizSessionModel) QuizSessionResponse {
	return QuizSessionResponse{
		ID:          session.ID,
		StudentID:   session.StudentID,
		QuizDefID:   session.QuizDefID,
		Status:      session.Status,
		StartedAt:   session.StartedAt,
		SubmittedAt: session.SubmittedAt,
		ScoredAt:    session.ScoredAt,
		Score:       session.Score,
		MaxScore:    session.MaxScore,
		Passed:      session.Passed,
		Answers:     session.Answers,
		CreatedAt:   session.CreatedAt,
	}
}

func exportRequestToResponse(req *ExportRequestModel) ExportRequestResponse {
	return ExportRequestResponse{
		ID:        req.ID,
		Status:    req.Status,
		FileURL:   req.FileURL,
		ExpiresAt: req.ExpiresAt,
		CreatedAt: req.CreatedAt,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Sequence Engine — [06-learn §8.1.12]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateSequenceDef(ctx context.Context, cmd CreateSequenceDefCommand) (SequenceDefResponse, error) {
	if len(cmd.Items) == 0 {
		return SequenceDefResponse{}, &LearningError{Err: domain.ErrSequenceNoItems}
	}
	isLinear := true
	if cmd.IsLinear != nil {
		isLinear = *cmd.IsLinear
	}
	def := SequenceDefModel{
		PublisherID:   cmd.PublisherID,
		Title:         cmd.Title,
		Description:   cmd.Description,
		SubjectTags:   StringArray(cmd.SubjectTags),
		MethodologyID: cmd.MethodologyID,
		IsLinear:      isLinear,
	}
	if err := s.sequenceDefRepo.Create(ctx, &def); err != nil {
		return SequenceDefResponse{}, err
	}
	items := make([]SequenceItemModel, len(cmd.Items))
	for i, item := range cmd.Items {
		isRequired := true
		if item.IsRequired != nil {
			isRequired = *item.IsRequired
		}
		unlockAfterPrev := false
		if item.UnlockAfterPrevious != nil {
			unlockAfterPrev = *item.UnlockAfterPrevious
		}
		items[i] = SequenceItemModel{
			SequenceDefID:       def.ID,
			SortOrder:           item.SortOrder,
			ContentType:         item.ContentType,
			ContentID:           item.ContentID,
			IsRequired:          isRequired,
			UnlockAfterPrevious: unlockAfterPrev,
		}
	}
	if err := s.sequenceDefRepo.SetItems(ctx, def.ID, items); err != nil {
		return SequenceDefResponse{}, err
	}
	return sequenceDefToResponse(&def), nil
}

func (s *learningServiceImpl) UpdateSequenceDef(ctx context.Context, defID uuid.UUID, cmd UpdateSequenceDefCommand) (SequenceDefResponse, error) {
	def, err := s.sequenceDefRepo.FindByID(ctx, defID)
	if err != nil {
		return SequenceDefResponse{}, err
	}
	if cmd.Title != nil {
		def.Title = *cmd.Title
	}
	if cmd.Description != nil {
		def.Description = cmd.Description
	}
	if cmd.SubjectTags != nil {
		def.SubjectTags = StringArray(*cmd.SubjectTags)
	}
	if cmd.IsLinear != nil {
		def.IsLinear = *cmd.IsLinear
	}
	if err := s.sequenceDefRepo.Update(ctx, def); err != nil {
		return SequenceDefResponse{}, err
	}
	if cmd.Items != nil {
		items := make([]SequenceItemModel, len(*cmd.Items))
		for i, item := range *cmd.Items {
			isRequired := true
			if item.IsRequired != nil {
				isRequired = *item.IsRequired
			}
			unlockAfterPrev := false
			if item.UnlockAfterPrevious != nil {
				unlockAfterPrev = *item.UnlockAfterPrevious
			}
			items[i] = SequenceItemModel{
				SequenceDefID:       defID,
				SortOrder:           item.SortOrder,
				ContentType:         item.ContentType,
				ContentID:           item.ContentID,
				IsRequired:          isRequired,
				UnlockAfterPrevious: unlockAfterPrev,
			}
		}
		if err := s.sequenceDefRepo.SetItems(ctx, defID, items); err != nil {
			return SequenceDefResponse{}, err
		}
	}
	return sequenceDefToResponse(def), nil
}

func (s *learningServiceImpl) GetSequenceDef(ctx context.Context, defID uuid.UUID) (SequenceDefDetailResponse, error) {
	def, err := s.sequenceDefRepo.FindByID(ctx, defID)
	if err != nil {
		return SequenceDefDetailResponse{}, err
	}
	items, err := s.sequenceDefRepo.ListItems(ctx, defID)
	if err != nil {
		return SequenceDefDetailResponse{}, err
	}
	itemResponses := make([]SequenceItemResponse, len(items))
	for i, item := range items {
		itemResponses[i] = SequenceItemResponse{
			ID:                  item.ID,
			SortOrder:           item.SortOrder,
			ContentType:         item.ContentType,
			ContentID:           item.ContentID,
			IsRequired:          item.IsRequired,
			UnlockAfterPrevious: item.UnlockAfterPrevious,
		}
	}
	return SequenceDefDetailResponse{
		SequenceDefResponse: sequenceDefToResponse(def),
		Items:               itemResponses,
	}, nil
}

func (s *learningServiceImpl) StartSequence(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartSequenceCommand) (SequenceProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return SequenceProgressResponse{}, err
	}
	// Verify sequence exists
	if _, err := s.sequenceDefRepo.FindByID(ctx, cmd.SequenceDefID); err != nil {
		return SequenceProgressResponse{}, err
	}
	now := time.Now()
	progress := SequenceProgressModel{
		StudentID:        studentID,
		SequenceDefID:    cmd.SequenceDefID,
		CurrentItemIndex: 0,
		Status:           "in_progress",
		ItemCompletions:  json.RawMessage("[]"),
		StartedAt:        &now,
	}
	if err := s.sequenceProgressRepo.Create(ctx, scope, &progress); err != nil {
		return SequenceProgressResponse{}, err
	}
	return sequenceProgressToResponse(&progress), nil
}

func (s *learningServiceImpl) UpdateSequenceProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateSequenceProgressCommand) (SequenceProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return SequenceProgressResponse{}, err
	}
	progress, err := s.sequenceProgressRepo.FindByID(ctx, scope, progressID)
	if err != nil {
		return SequenceProgressResponse{}, err
	}
	// Load sequence items for validation
	items, err := s.sequenceDefRepo.ListItems(ctx, progress.SequenceDefID)
	if err != nil {
		return SequenceProgressResponse{}, err
	}
	def, err := s.sequenceDefRepo.FindByID(ctx, progress.SequenceDefID)
	if err != nil {
		return SequenceProgressResponse{}, err
	}

	completions, _ := domain.ParseItemCompletions(progress.ItemCompletions)
	completedIDs := make(map[string]bool, len(completions))
	for _, c := range completions {
		completedIDs[c.ItemID] = true
	}

	if cmd.CompleteItemID != nil {
		// Find the item in the sequence
		var targetItem *SequenceItemModel
		var targetIdx int
		for i := range items {
			if items[i].ID == *cmd.CompleteItemID {
				targetItem = &items[i]
				targetIdx = i
				break
			}
		}
		if targetItem == nil {
			return SequenceProgressResponse{}, &LearningError{Err: domain.ErrSequenceItemLocked}
		}
		// Linear enforcement: previous required items must be complete
		if def.IsLinear {
			for i := 0; i < targetIdx; i++ {
				if items[i].IsRequired && !completedIDs[items[i].ID.String()] {
					return SequenceProgressResponse{}, &LearningError{Err: domain.ErrSequenceItemLocked}
				}
			}
		}
		if !completedIDs[targetItem.ID.String()] {
			completions = append(completions, domain.ItemCompletion{
				ItemID:      targetItem.ID.String(),
				CompletedAt: time.Now().Format(time.RFC3339),
			})
			if int16(targetIdx) >= progress.CurrentItemIndex {
				progress.CurrentItemIndex = int16(targetIdx) + 1
			}
			_ = s.eventBus.Publish(ctx, SequenceAdvanced{
				FamilyID:        scope.FamilyID(),
				StudentID:       studentID,
				SequenceDefID:   progress.SequenceDefID,
				ItemIndex:       int16(targetIdx),
				ItemContentType: targetItem.ContentType,
				ItemContentID:   targetItem.ContentID,
			})
		}
	}

	if cmd.SkipItemID != nil {
		if !completedIDs[cmd.SkipItemID.String()] {
			completions = append(completions, domain.ItemCompletion{
				ItemID:      cmd.SkipItemID.String(),
				CompletedAt: time.Now().Format(time.RFC3339),
				Skipped:     true,
			})
		}
	}

	// Check for sequence completion: all required items done
	updatedCompletedIDs := make(map[string]bool, len(completions))
	for _, c := range completions {
		updatedCompletedIDs[c.ItemID] = true
	}
	allRequiredDone := true
	for _, item := range items {
		if item.IsRequired && !updatedCompletedIDs[item.ID.String()] {
			allRequiredDone = false
			break
		}
	}
	if allRequiredDone && progress.Status != "completed" {
		progress.Status = "completed"
		now := time.Now()
		progress.CompletedAt = &now
		_ = s.eventBus.Publish(ctx, SequenceCompleted{
			FamilyID:      scope.FamilyID(),
			StudentID:     studentID,
			SequenceDefID: progress.SequenceDefID,
		})
	}

	completionsJSON, _ := json.Marshal(completions)
	progress.ItemCompletions = completionsJSON

	if err := s.sequenceProgressRepo.Update(ctx, scope, progress); err != nil {
		return SequenceProgressResponse{}, err
	}
	return sequenceProgressToResponse(progress), nil
}

func (s *learningServiceImpl) GetSequenceProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID) (SequenceProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return SequenceProgressResponse{}, err
	}
	progress, err := s.sequenceProgressRepo.FindByID(ctx, scope, progressID)
	if err != nil {
		return SequenceProgressResponse{}, err
	}
	return sequenceProgressToResponse(progress), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Student Assignments — [06-learn §8.6.3]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateAssignment(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd CreateAssignmentCommand) (AssignmentResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return AssignmentResponse{}, err
	}
	assignment := StudentAssignmentModel{
		StudentID:   studentID,
		AssignedBy:  cmd.AssignedBy,
		ContentType: cmd.ContentType,
		ContentID:   cmd.ContentID,
		DueDate:     cmd.DueDate,
		Status:      "assigned",
	}
	if err := s.assignmentRepo.Create(ctx, scope, &assignment); err != nil {
		return AssignmentResponse{}, err
	}
	return assignmentToResponse(&assignment), nil
}

func (s *learningServiceImpl) UpdateAssignment(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID, cmd UpdateAssignmentCommand) (AssignmentResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return AssignmentResponse{}, err
	}
	assignment, err := s.assignmentRepo.FindByID(ctx, scope, assignmentID)
	if err != nil {
		return AssignmentResponse{}, err
	}
	if cmd.Status != nil {
		if err := domain.ValidateAssignmentStatusTransition(assignment.Status, *cmd.Status); err != nil {
			return AssignmentResponse{}, &LearningError{Err: err}
		}
		assignment.Status = *cmd.Status
		if *cmd.Status == "completed" {
			now := time.Now()
			assignment.CompletedAt = &now
			_ = s.eventBus.Publish(ctx, AssignmentCompleted{
				FamilyID:     scope.FamilyID(),
				StudentID:    studentID,
				AssignmentID: assignmentID,
				ContentType:  assignment.ContentType,
				ContentID:    assignment.ContentID,
			})
		}
	}
	if cmd.DueDate != nil {
		assignment.DueDate = cmd.DueDate
	}
	if err := s.assignmentRepo.Update(ctx, scope, assignment); err != nil {
		return AssignmentResponse{}, err
	}
	return assignmentToResponse(assignment), nil
}

func (s *learningServiceImpl) DeleteAssignment(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, assignmentID uuid.UUID) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}
	return s.assignmentRepo.Delete(ctx, scope, assignmentID)
}

func (s *learningServiceImpl) ListAssignments(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query AssignmentQuery) (PaginatedResponse[AssignmentResponse], error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return PaginatedResponse[AssignmentResponse]{}, err
	}
	models, err := s.assignmentRepo.ListByStudent(ctx, scope, studentID, &query)
	if err != nil {
		return PaginatedResponse[AssignmentResponse]{}, err
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	hasMore := int64(len(models)) > limit
	if hasMore {
		models = models[:limit]
	}
	responses := make([]AssignmentResponse, len(models))
	for i := range models {
		responses[i] = assignmentToResponse(&models[i])
	}
	return PaginatedResponse[AssignmentResponse]{Data: responses, HasMore: hasMore}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Video — [06-learn §8.1.11]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) ListVideoDefs(ctx context.Context, query VideoDefQuery) (PaginatedResponse[VideoDefResponse], error) {
	models, err := s.videoDefRepo.List(ctx, &query)
	if err != nil {
		return PaginatedResponse[VideoDefResponse]{}, err
	}
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	hasMore := int64(len(models)) > limit
	if hasMore {
		models = models[:limit]
	}
	responses := make([]VideoDefResponse, len(models))
	for i := range models {
		responses[i] = videoDefToResponse(&models[i])
	}
	return PaginatedResponse[VideoDefResponse]{Data: responses, HasMore: hasMore}, nil
}

func (s *learningServiceImpl) GetVideoDef(ctx context.Context, videoDefID uuid.UUID) (VideoDefResponse, error) {
	def, err := s.videoDefRepo.FindByID(ctx, videoDefID)
	if err != nil {
		return VideoDefResponse{}, err
	}
	return videoDefToResponse(def), nil
}

func (s *learningServiceImpl) UpdateVideoProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd UpdateVideoProgressCommand) (VideoProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return VideoProgressResponse{}, err
	}
	// Try to find existing progress; create new if not found
	progress, err := s.videoProgressRepo.FindByStudentAndVideo(ctx, scope, studentID, cmd.VideoDefID)
	if err != nil {
		var le *LearningError
		if errors.As(err, &le) && errors.Is(le.Err, domain.ErrVideoProgressNotFound) {
			progress = &VideoProgressModel{
				StudentID:  studentID,
				VideoDefID: cmd.VideoDefID,
			}
		} else {
			return VideoProgressResponse{}, err
		}
	}
	if cmd.WatchedSeconds != nil && *cmd.WatchedSeconds > progress.WatchedSeconds {
		progress.WatchedSeconds = *cmd.WatchedSeconds
	}
	if cmd.LastPositionSeconds != nil {
		progress.LastPositionSeconds = *cmd.LastPositionSeconds
	}
	if cmd.Completed != nil && *cmd.Completed && !progress.Completed {
		progress.Completed = true
		now := time.Now()
		progress.CompletedAt = &now
	}
	if err := s.videoProgressRepo.Upsert(ctx, scope, progress); err != nil {
		return VideoProgressResponse{}, err
	}
	return videoProgressToResponse(progress), nil
}

func (s *learningServiceImpl) GetVideoProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, videoDefID uuid.UUID) (VideoProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return VideoProgressResponse{}, err
	}
	progress, err := s.videoProgressRepo.FindByStudentAndVideo(ctx, scope, studentID, videoDefID)
	if err != nil {
		return VideoProgressResponse{}, err
	}
	return videoProgressToResponse(progress), nil
}

// ─── Batch 7-8 Mappers ──────────────────────────────────────────────────────

func sequenceDefToResponse(def *SequenceDefModel) SequenceDefResponse {
	return SequenceDefResponse{
		ID:            def.ID,
		PublisherID:   def.PublisherID,
		Title:         def.Title,
		Description:   def.Description,
		SubjectTags:   []string(def.SubjectTags),
		MethodologyID: def.MethodologyID,
		IsLinear:      def.IsLinear,
		CreatedAt:     def.CreatedAt,
	}
}

func sequenceProgressToResponse(p *SequenceProgressModel) SequenceProgressResponse {
	return SequenceProgressResponse{
		ID:               p.ID,
		StudentID:        p.StudentID,
		SequenceDefID:    p.SequenceDefID,
		CurrentItemIndex: p.CurrentItemIndex,
		Status:           p.Status,
		ItemCompletions:  p.ItemCompletions,
		StartedAt:        p.StartedAt,
		CompletedAt:      p.CompletedAt,
		CreatedAt:        p.CreatedAt,
	}
}

func assignmentToResponse(a *StudentAssignmentModel) AssignmentResponse {
	return AssignmentResponse{
		ID:          a.ID,
		StudentID:   a.StudentID,
		AssignedBy:  a.AssignedBy,
		ContentType: a.ContentType,
		ContentID:   a.ContentID,
		DueDate:     a.DueDate,
		Status:      a.Status,
		AssignedAt:  a.AssignedAt,
		CompletedAt: a.CompletedAt,
		CreatedAt:   a.CreatedAt,
	}
}

func videoDefToResponse(def *VideoDefModel) VideoDefResponse {
	return VideoDefResponse{
		ID:              def.ID,
		PublisherID:     def.PublisherID,
		Title:           def.Title,
		Description:     def.Description,
		SubjectTags:     []string(def.SubjectTags),
		MethodologyID:   def.MethodologyID,
		DurationSeconds: def.DurationSeconds,
		ThumbnailURL:    def.ThumbnailURL,
		VideoURL:        def.VideoURL,
		VideoSource:     def.VideoSource,
		CreatedAt:       def.CreatedAt,
	}
}

func videoProgressToResponse(p *VideoProgressModel) VideoProgressResponse {
	return VideoProgressResponse{
		ID:                  p.ID,
		StudentID:           p.StudentID,
		VideoDefID:          p.VideoDefID,
		WatchedSeconds:      p.WatchedSeconds,
		Completed:           p.Completed,
		LastPositionSeconds: p.LastPositionSeconds,
		CompletedAt:         p.CompletedAt,
		CreatedAt:           p.CreatedAt,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Event Handlers — [06-learn §17.4]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) HandleStudentCreated(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	// No-op: student learning defaults are created lazily on first use.
	return nil
}

func (s *learningServiceImpl) HandleStudentDeleted(ctx context.Context, familyID uuid.UUID, studentID uuid.UUID) error {
	// Cascade-delete all learning data for this student.
	// Uses BypassRLSTransaction because event handlers run outside auth context.
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		tables := []string{
			"learn_activity_logs",
			"learn_reading_progress",
			"learn_journal_entries",
			"learn_quiz_sessions",
			"learn_sequence_progress",
			"learn_student_assignments",
			"learn_video_progress",
			"learn_assessment_results",
		}
		for _, table := range tables {
			if err := tx.Exec("DELETE FROM "+table+" WHERE student_id = ? AND family_id = ?", studentID, familyID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetPortfolioItemSummary returns summary data for an activity log or journal entry.
// Used by comply:: domain for portfolio item display. Uses BypassRLSTransaction
// since the caller (comply adapter) has familyID but no auth context. [06-learn §15]
func (s *learningServiceImpl) GetPortfolioItemSummary(ctx context.Context, familyID uuid.UUID, sourceType string, sourceID uuid.UUID) (*PortfolioItemSummary, error) {
	scope := shared.NewFamilyScopeFromID(familyID)
	var summary PortfolioItemSummary
	err := shared.ScopedTransaction(ctx, s.db, scope, func(tx *gorm.DB) error {
		switch sourceType {
		case "activity_log":
			var log ActivityLogModel
			if err := tx.Where("id = ?", sourceID).First(&log).Error; err != nil {
				return err
			}
			summary.Title = log.Title
			summary.Description = log.Description
			summary.Date = log.ActivityDate
			if len(log.SubjectTags) > 0 {
				summary.Subject = &log.SubjectTags[0]
			}
		case "journal_entry":
			var entry JournalEntryModel
			if err := tx.Where("id = ?", sourceID).First(&entry).Error; err != nil {
				return err
			}
			if entry.Title != nil {
				summary.Title = *entry.Title
			} else {
				summary.Title = entry.EntryType
			}
			summary.Description = &entry.Content
			summary.Date = entry.EntryDate
		default:
			summary.Title = sourceType + " " + sourceID.String()
			summary.Date = time.Now()
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &summary, nil
}

func (s *learningServiceImpl) HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error {
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		// Delete all family-scoped learning data. Order: dependent tables first.
		tables := []string{
			"learn_reading_list_items", // FK → learn_reading_lists
			"learn_reading_lists",
			"learn_activity_logs",
			"learn_reading_progress",
			"learn_journal_entries",
			"learn_quiz_sessions",
			"learn_sequence_progress",
			"learn_student_assignments",
			"learn_video_progress",
			"learn_assessment_results",
			"learn_project_progress",
			"learn_grading_scales",
			"learn_custom_subjects",
			"learn_progress_snapshots",
			"learn_export_requests",
		}
		for _, table := range tables {
			if err := tx.Exec("DELETE FROM "+table+" WHERE family_id = ?", familyID).Error; err != nil {
				return fmt.Errorf("learn: delete %s: %w", table, err)
			}
		}
		return nil
	})
}

func (s *learningServiceImpl) HandlePurchaseCompleted(_ context.Context, _ uuid.UUID, _ PurchaseMetadata) error {
	// No-op for learn:: — purchased content is served directly by mkt:: domain.
	return nil
}

func (s *learningServiceImpl) HandleMethodologyConfigUpdated(_ context.Context) error {
	// P1-5: Tool resolution is computed fresh on each request (no in-memory or Redis cache),
	// so there is no stale-cache risk. If caching is added in the future, invalidation
	// must be triggered here by clearing the relevant cache key prefix.
	return nil
}

// ─── Background Jobs ─────────────────────────────────────────────────────────

// SnapshotProgress computes and stores weekly progress snapshots for all active students.
// Uses BypassRLSTransaction to list students across all families, then constructs a
// per-family scope to compute metrics using family-scoped repositories. [06-learn §12.3]
func (s *learningServiceImpl) SnapshotProgress(ctx context.Context) error {
	type studentRow struct {
		StudentID uuid.UUID `gorm:"column:id"`
		FamilyID  uuid.UUID `gorm:"column:family_id"`
	}
	var students []studentRow
	if err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		return tx.Table("iam_students").
			Select("id, family_id").
			Where("deleted_at IS NULL").
			Scan(&students).Error
	}); err != nil {
		return err
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	// Weekly window: past 7 days
	weekAgo := today.AddDate(0, 0, -7)

	var lastErr error
	for _, s2 := range students {
		scope := shared.NewFamilyScopeFromID(s2.FamilyID)

		totalActivities, err := s.activityLogRepo.CountByStudentDateRange(ctx, &scope, s2.StudentID, weekAgo, today)
		if err != nil {
			lastErr = err
			continue
		}
		hoursBySubject, err := s.activityLogRepo.HoursBySubject(ctx, &scope, s2.StudentID, weekAgo, today)
		if err != nil {
			lastErr = err
			continue
		}
		var totalHours float64
		subjectHours := make([]SubjectHoursResponse, len(hoursBySubject))
		for i, h := range hoursBySubject {
			hours := float64(h.TotalMinutes) / 60.0
			totalHours += hours
			subjectHours[i] = SubjectHoursResponse{SubjectSlug: h.SubjectSlug, SubjectName: h.SubjectSlug, Hours: hours}
		}
		booksCompleted, err := s.readingProgressRepo.CountCompleted(ctx, &scope, s2.StudentID, weekAgo, today)
		if err != nil {
			lastErr = err
			continue
		}
		journalEntries, err := s.journalEntryRepo.CountByStudentDateRange(ctx, &scope, s2.StudentID, weekAgo, today)
		if err != nil {
			lastErr = err
			continue
		}

		summary := ProgressSummaryResponse{
			StudentID:       s2.StudentID,
			DateFrom:        weekAgo,
			DateTo:          today,
			TotalActivities: totalActivities,
			TotalHours:      totalHours,
			HoursBySubject:  subjectHours,
			BooksCompleted:  booksCompleted,
			JournalEntries:  journalEntries,
		}
		data, err := json.Marshal(summary)
		if err != nil {
			lastErr = err
			continue
		}

		snap := &ProgressSnapshotModel{
			StudentID:    s2.StudentID,
			SnapshotDate: today,
			Data:         data,
		}
		if err := s.progressRepo.CreateSnapshot(ctx, &scope, snap); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Assessment Definitions (Layer 1)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateAssessmentDef(ctx context.Context, cmd CreateAssessmentDefCommand) (AssessmentDefResponse, error) {
	ok, err := s.mkt.IsPublisherMember(ctx, cmd.CallerID, cmd.PublisherID)
	if err != nil {
		return AssessmentDefResponse{}, err
	}
	if !ok {
		return AssessmentDefResponse{}, &LearningError{Err: domain.ErrNotPublisherMember}
	}

	model := &AssessmentDefModel{
		PublisherID:  cmd.PublisherID,
		Title:        cmd.Title,
		Description:  cmd.Description,
		SubjectTags:  StringArray(cmd.SubjectTags),
		ScoringType:  cmd.ScoringType,
		MaxScore:     cmd.MaxScore,
	}
	if err := s.assessmentDefRepo.Create(ctx, model); err != nil {
		return AssessmentDefResponse{}, err
	}
	return toAssessmentDefResponse(model), nil
}

func (s *learningServiceImpl) UpdateAssessmentDef(ctx context.Context, defID uuid.UUID, cmd UpdateAssessmentDefCommand) (AssessmentDefResponse, error) {
	model, err := s.assessmentDefRepo.FindByID(ctx, defID)
	if err != nil {
		return AssessmentDefResponse{}, err
	}
	ok, err := s.mkt.IsPublisherMember(ctx, cmd.CallerID, model.PublisherID)
	if err != nil {
		return AssessmentDefResponse{}, err
	}
	if !ok {
		return AssessmentDefResponse{}, &LearningError{Err: domain.ErrNotPublisherMember}
	}

	if cmd.Title != nil {
		model.Title = *cmd.Title
	}
	if cmd.Description != nil {
		model.Description = cmd.Description
	}
	if cmd.SubjectTags != nil {
		model.SubjectTags = StringArray(cmd.SubjectTags)
	}
	if cmd.ScoringType != nil {
		model.ScoringType = *cmd.ScoringType
	}
	if cmd.MaxScore != nil {
		model.MaxScore = cmd.MaxScore
	}

	if err := s.assessmentDefRepo.Update(ctx, model); err != nil {
		return AssessmentDefResponse{}, err
	}
	return toAssessmentDefResponse(model), nil
}

func (s *learningServiceImpl) DeleteAssessmentDef(ctx context.Context, defID uuid.UUID, callerID uuid.UUID) error {
	model, err := s.assessmentDefRepo.FindByID(ctx, defID)
	if err != nil {
		return err
	}
	ok, err := s.mkt.IsPublisherMember(ctx, callerID, model.PublisherID)
	if err != nil {
		return err
	}
	if !ok {
		return &LearningError{Err: domain.ErrNotPublisherMember}
	}
	return s.assessmentDefRepo.SoftDelete(ctx, defID)
}

func (s *learningServiceImpl) ListAssessmentDefs(ctx context.Context, query AssessmentDefQuery) (PaginatedResponse[AssessmentDefSummaryResponse], error) {
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	query.Limit = limit + 1 // fetch one extra for hasMore

	models, err := s.assessmentDefRepo.List(ctx, &query)
	if err != nil {
		return PaginatedResponse[AssessmentDefSummaryResponse]{}, err
	}

	hasMore := int64(len(models)) > limit
	if hasMore {
		models = models[:limit]
	}

	items := make([]AssessmentDefSummaryResponse, len(models))
	for i, m := range models {
		items[i] = AssessmentDefSummaryResponse{
			ID:          m.ID,
			Title:       m.Title,
			SubjectTags: []string(m.SubjectTags),
			ScoringType: m.ScoringType,
			MaxScore:    m.MaxScore,
		}
	}

	var nextCursor *uuid.UUID
	if hasMore && len(items) > 0 {
		nextCursor = &items[len(items)-1].ID
	}

	return PaginatedResponse[AssessmentDefSummaryResponse]{Data: items, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *learningServiceImpl) GetAssessmentDef(ctx context.Context, defID uuid.UUID) (AssessmentDefResponse, error) {
	model, err := s.assessmentDefRepo.FindByID(ctx, defID)
	if err != nil {
		return AssessmentDefResponse{}, err
	}
	return toAssessmentDefResponse(model), nil
}

func toAssessmentDefResponse(m *AssessmentDefModel) AssessmentDefResponse {
	return AssessmentDefResponse{
		ID:          m.ID,
		PublisherID: m.PublisherID,
		Title:       m.Title,
		Description: m.Description,
		SubjectTags: []string(m.SubjectTags),
		ScoringType: m.ScoringType,
		MaxScore:    m.MaxScore,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Project Definitions (Layer 1)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateProjectDef(ctx context.Context, cmd CreateProjectDefCommand) (ProjectDefResponse, error) {
	ok, err := s.mkt.IsPublisherMember(ctx, cmd.CallerID, cmd.PublisherID)
	if err != nil {
		return ProjectDefResponse{}, err
	}
	if !ok {
		return ProjectDefResponse{}, &LearningError{Err: domain.ErrNotPublisherMember}
	}

	milestones := cmd.MilestoneTemplates
	if milestones == nil {
		milestones = json.RawMessage("[]")
	}

	model := &ProjectDefModel{
		PublisherID:        cmd.PublisherID,
		Title:              cmd.Title,
		Description:        cmd.Description,
		SubjectTags:        StringArray(cmd.SubjectTags),
		MilestoneTemplates: milestones,
	}
	if err := s.projectDefRepo.Create(ctx, model); err != nil {
		return ProjectDefResponse{}, err
	}
	return toProjectDefResponse(model), nil
}

func (s *learningServiceImpl) UpdateProjectDef(ctx context.Context, defID uuid.UUID, cmd UpdateProjectDefCommand) (ProjectDefResponse, error) {
	model, err := s.projectDefRepo.FindByID(ctx, defID)
	if err != nil {
		return ProjectDefResponse{}, err
	}
	ok, err := s.mkt.IsPublisherMember(ctx, cmd.CallerID, model.PublisherID)
	if err != nil {
		return ProjectDefResponse{}, err
	}
	if !ok {
		return ProjectDefResponse{}, &LearningError{Err: domain.ErrNotPublisherMember}
	}

	if cmd.Title != nil {
		model.Title = *cmd.Title
	}
	if cmd.Description != nil {
		model.Description = cmd.Description
	}
	if cmd.SubjectTags != nil {
		model.SubjectTags = StringArray(cmd.SubjectTags)
	}
	if cmd.MilestoneTemplates != nil {
		model.MilestoneTemplates = cmd.MilestoneTemplates
	}

	if err := s.projectDefRepo.Update(ctx, model); err != nil {
		return ProjectDefResponse{}, err
	}
	return toProjectDefResponse(model), nil
}

func (s *learningServiceImpl) DeleteProjectDef(ctx context.Context, defID uuid.UUID, callerID uuid.UUID) error {
	model, err := s.projectDefRepo.FindByID(ctx, defID)
	if err != nil {
		return err
	}
	ok, err := s.mkt.IsPublisherMember(ctx, callerID, model.PublisherID)
	if err != nil {
		return err
	}
	if !ok {
		return &LearningError{Err: domain.ErrNotPublisherMember}
	}
	return s.projectDefRepo.SoftDelete(ctx, defID)
}

func (s *learningServiceImpl) ListProjectDefs(ctx context.Context, query ProjectDefQuery) (PaginatedResponse[ProjectDefSummaryResponse], error) {
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	query.Limit = limit + 1

	models, err := s.projectDefRepo.List(ctx, &query)
	if err != nil {
		return PaginatedResponse[ProjectDefSummaryResponse]{}, err
	}

	hasMore := int64(len(models)) > limit
	if hasMore {
		models = models[:limit]
	}

	items := make([]ProjectDefSummaryResponse, len(models))
	for i, m := range models {
		items[i] = ProjectDefSummaryResponse{
			ID:          m.ID,
			Title:       m.Title,
			SubjectTags: []string(m.SubjectTags),
		}
	}

	var nextCursor *uuid.UUID
	if hasMore && len(items) > 0 {
		nextCursor = &items[len(items)-1].ID
	}

	return PaginatedResponse[ProjectDefSummaryResponse]{Data: items, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *learningServiceImpl) GetProjectDef(ctx context.Context, defID uuid.UUID) (ProjectDefResponse, error) {
	model, err := s.projectDefRepo.FindByID(ctx, defID)
	if err != nil {
		return ProjectDefResponse{}, err
	}
	return toProjectDefResponse(model), nil
}

func toProjectDefResponse(m *ProjectDefModel) ProjectDefResponse {
	return ProjectDefResponse{
		ID:                 m.ID,
		PublisherID:        m.PublisherID,
		Title:              m.Title,
		Description:        m.Description,
		SubjectTags:        []string(m.SubjectTags),
		MilestoneTemplates: m.MilestoneTemplates,
		CreatedAt:          m.CreatedAt,
		UpdatedAt:          m.UpdatedAt,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Assessment Results (Layer 3)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) RecordAssessmentResult(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd RecordAssessmentResultCommand) (AssessmentResultResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return AssessmentResultResponse{}, err
	}

	// Verify assessment def exists.
	if _, err := s.assessmentDefRepo.FindByID(ctx, cmd.AssessmentDefID); err != nil {
		return AssessmentResultResponse{}, err
	}

	assessmentDate := time.Now()
	if cmd.AssessmentDate != nil {
		parsed, err := time.Parse("2006-01-02", *cmd.AssessmentDate)
		if err != nil {
			return AssessmentResultResponse{}, &LearningError{Err: errors.New("invalid assessment_date format, expected YYYY-MM-DD")}
		}
		assessmentDate = parsed
	}

	weight := 1.0
	if cmd.Weight != nil {
		weight = *cmd.Weight
	}

	model := &AssessmentResultModel{
		StudentID:       studentID,
		AssessmentDefID: cmd.AssessmentDefID,
		Score:           cmd.Score,
		MaxScore:        cmd.MaxScore,
		Weight:          weight,
		Notes:           cmd.Notes,
		AssessmentDate:  assessmentDate,
	}
	if err := s.assessmentResultRepo.Create(ctx, scope, model); err != nil {
		return AssessmentResultResponse{}, err
	}
	return toAssessmentResultResponse(model), nil
}

func (s *learningServiceImpl) UpdateAssessmentResult(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, resultID uuid.UUID, cmd UpdateAssessmentResultCommand) (AssessmentResultResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return AssessmentResultResponse{}, err
	}

	model, err := s.assessmentResultRepo.FindByID(ctx, scope, resultID)
	if err != nil {
		return AssessmentResultResponse{}, err
	}

	if cmd.Score != nil {
		model.Score = *cmd.Score
	}
	if cmd.MaxScore != nil {
		model.MaxScore = cmd.MaxScore
	}
	if cmd.Weight != nil {
		model.Weight = *cmd.Weight
	}
	if cmd.Notes != nil {
		model.Notes = cmd.Notes
	}
	if cmd.AssessmentDate != nil {
		parsed, err := time.Parse("2006-01-02", *cmd.AssessmentDate)
		if err != nil {
			return AssessmentResultResponse{}, &LearningError{Err: errors.New("invalid assessment_date format, expected YYYY-MM-DD")}
		}
		model.AssessmentDate = parsed
	}

	if err := s.assessmentResultRepo.Update(ctx, scope, model); err != nil {
		return AssessmentResultResponse{}, err
	}
	return toAssessmentResultResponse(model), nil
}

func (s *learningServiceImpl) DeleteAssessmentResult(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, resultID uuid.UUID) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}
	return s.assessmentResultRepo.Delete(ctx, scope, resultID)
}

func (s *learningServiceImpl) ListAssessmentResults(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query AssessmentResultQuery) (PaginatedResponse[AssessmentResultResponse], error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return PaginatedResponse[AssessmentResultResponse]{}, err
	}

	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	query.Limit = limit + 1

	models, err := s.assessmentResultRepo.ListByStudent(ctx, scope, studentID, &query)
	if err != nil {
		return PaginatedResponse[AssessmentResultResponse]{}, err
	}

	hasMore := int64(len(models)) > limit
	if hasMore {
		models = models[:limit]
	}

	items := make([]AssessmentResultResponse, len(models))
	for i, m := range models {
		items[i] = toAssessmentResultResponse(&m)
	}

	var nextCursor *uuid.UUID
	if hasMore && len(items) > 0 {
		nextCursor = &items[len(items)-1].ID
	}

	return PaginatedResponse[AssessmentResultResponse]{Data: items, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *learningServiceImpl) GetAssessmentResult(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, resultID uuid.UUID) (AssessmentResultResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return AssessmentResultResponse{}, err
	}
	model, err := s.assessmentResultRepo.FindByID(ctx, scope, resultID)
	if err != nil {
		return AssessmentResultResponse{}, err
	}
	return toAssessmentResultResponse(model), nil
}

func toAssessmentResultResponse(m *AssessmentResultModel) AssessmentResultResponse {
	return AssessmentResultResponse{
		ID:              m.ID,
		StudentID:       m.StudentID,
		AssessmentDefID: m.AssessmentDefID,
		Score:           m.Score,
		MaxScore:        m.MaxScore,
		Weight:          m.Weight,
		Notes:           m.Notes,
		AssessmentDate:  m.AssessmentDate,
		CreatedAt:       m.CreatedAt,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Project Progress (Layer 3)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) StartProject(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd StartProjectCommand) (ProjectProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return ProjectProgressResponse{}, err
	}

	// Verify project def exists.
	if _, err := s.projectDefRepo.FindByID(ctx, cmd.ProjectDefID); err != nil {
		return ProjectProgressResponse{}, err
	}

	now := time.Now()
	model := &ProjectProgressModel{
		StudentID:    studentID,
		ProjectDefID: cmd.ProjectDefID,
		Status:       "planning",
		Milestones:   json.RawMessage("[]"),
		StartedAt:    &now,
		Notes:        cmd.Notes,
		Attachments:  json.RawMessage("[]"),
	}
	if err := s.projectProgressRepo.Create(ctx, scope, model); err != nil {
		return ProjectProgressResponse{}, err
	}
	return toProjectProgressResponse(model), nil
}

func (s *learningServiceImpl) UpdateProjectProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID, cmd UpdateProjectProgressCommand) (ProjectProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return ProjectProgressResponse{}, err
	}

	model, err := s.projectProgressRepo.FindByID(ctx, scope, progressID)
	if err != nil {
		return ProjectProgressResponse{}, err
	}

	if cmd.Status != nil {
		if !isValidProjectStatusTransition(model.Status, *cmd.Status) {
			return ProjectProgressResponse{}, &LearningError{Err: domain.ErrInvalidProjectStatusTransition}
		}
		model.Status = *cmd.Status
		if *cmd.Status == "completed" {
			now := time.Now()
			model.CompletedAt = &now
		}
	}
	if cmd.Milestones != nil {
		model.Milestones = cmd.Milestones
	}
	if cmd.Notes != nil {
		model.Notes = cmd.Notes
	}
	if cmd.Attachments != nil {
		model.Attachments = cmd.Attachments
	}

	if err := s.projectProgressRepo.Update(ctx, scope, model); err != nil {
		return ProjectProgressResponse{}, err
	}
	return toProjectProgressResponse(model), nil
}

func (s *learningServiceImpl) DeleteProjectProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID) error {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return err
	}
	return s.projectProgressRepo.Delete(ctx, scope, progressID)
}

func (s *learningServiceImpl) ListProjectProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, query ProjectProgressQuery) (PaginatedResponse[ProjectProgressResponse], error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return PaginatedResponse[ProjectProgressResponse]{}, err
	}

	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	query.Limit = limit + 1

	models, err := s.projectProgressRepo.ListByStudent(ctx, scope, studentID, &query)
	if err != nil {
		return PaginatedResponse[ProjectProgressResponse]{}, err
	}

	hasMore := int64(len(models)) > limit
	if hasMore {
		models = models[:limit]
	}

	items := make([]ProjectProgressResponse, len(models))
	for i, m := range models {
		items[i] = toProjectProgressResponse(&m)
	}

	var nextCursor *uuid.UUID
	if hasMore && len(items) > 0 {
		nextCursor = &items[len(items)-1].ID
	}

	return PaginatedResponse[ProjectProgressResponse]{Data: items, HasMore: hasMore, NextCursor: nextCursor}, nil
}

func (s *learningServiceImpl) GetProjectProgress(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, progressID uuid.UUID) (ProjectProgressResponse, error) {
	if err := s.verifyStudentInFamily(ctx, studentID, scope); err != nil {
		return ProjectProgressResponse{}, err
	}
	model, err := s.projectProgressRepo.FindByID(ctx, scope, progressID)
	if err != nil {
		return ProjectProgressResponse{}, err
	}
	return toProjectProgressResponse(model), nil
}

func toProjectProgressResponse(m *ProjectProgressModel) ProjectProgressResponse {
	return ProjectProgressResponse{
		ID:           m.ID,
		StudentID:    m.StudentID,
		ProjectDefID: m.ProjectDefID,
		Status:       m.Status,
		Milestones:   m.Milestones,
		StartedAt:    m.StartedAt,
		CompletedAt:  m.CompletedAt,
		Notes:        m.Notes,
		Attachments:  m.Attachments,
		CreatedAt:    m.CreatedAt,
	}
}

func isValidProjectStatusTransition(from, to string) bool {
	switch from {
	case "planning":
		return to == "in_progress" || to == "completed"
	case "in_progress":
		return to == "completed" || to == "planning"
	case "completed":
		return to == "in_progress"
	default:
		return false
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase 2: Grading Scales (Layer 3)
// ═══════════════════════════════════════════════════════════════════════════════

func (s *learningServiceImpl) CreateGradingScale(ctx context.Context, scope *shared.FamilyScope, cmd CreateGradingScaleCommand) (GradingScaleResponse, error) {
	if cmd.IsDefault {
		if err := s.gradingScaleRepo.ClearDefault(ctx, scope); err != nil {
			return GradingScaleResponse{}, err
		}
	}

	model := &GradingScaleModel{
		Name:      cmd.Name,
		ScaleType: cmd.ScaleType,
		Grades:    cmd.Grades,
		IsDefault: cmd.IsDefault,
	}
	if err := s.gradingScaleRepo.Create(ctx, scope, model); err != nil {
		return GradingScaleResponse{}, err
	}
	return toGradingScaleResponse(model), nil
}

func (s *learningServiceImpl) UpdateGradingScale(ctx context.Context, scope *shared.FamilyScope, scaleID uuid.UUID, cmd UpdateGradingScaleCommand) (GradingScaleResponse, error) {
	model, err := s.gradingScaleRepo.FindByID(ctx, scope, scaleID)
	if err != nil {
		return GradingScaleResponse{}, err
	}

	if cmd.Name != nil {
		model.Name = *cmd.Name
	}
	if cmd.Grades != nil {
		model.Grades = cmd.Grades
	}
	if cmd.IsDefault != nil {
		if *cmd.IsDefault {
			if err := s.gradingScaleRepo.ClearDefault(ctx, scope); err != nil {
				return GradingScaleResponse{}, err
			}
		}
		model.IsDefault = *cmd.IsDefault
	}

	if err := s.gradingScaleRepo.Update(ctx, scope, model); err != nil {
		return GradingScaleResponse{}, err
	}
	return toGradingScaleResponse(model), nil
}

func (s *learningServiceImpl) DeleteGradingScale(ctx context.Context, scope *shared.FamilyScope, scaleID uuid.UUID) error {
	return s.gradingScaleRepo.Delete(ctx, scope, scaleID)
}

func (s *learningServiceImpl) ListGradingScales(ctx context.Context, scope *shared.FamilyScope) ([]GradingScaleResponse, error) {
	models, err := s.gradingScaleRepo.ListByFamily(ctx, scope)
	if err != nil {
		return nil, err
	}

	results := make([]GradingScaleResponse, len(models))
	for i, m := range models {
		results[i] = toGradingScaleResponse(&m)
	}
	return results, nil
}

func (s *learningServiceImpl) GetGradingScale(ctx context.Context, scope *shared.FamilyScope, scaleID uuid.UUID) (GradingScaleResponse, error) {
	model, err := s.gradingScaleRepo.FindByID(ctx, scope, scaleID)
	if err != nil {
		return GradingScaleResponse{}, err
	}
	return toGradingScaleResponse(model), nil
}

func toGradingScaleResponse(m *GradingScaleModel) GradingScaleResponse {
	return GradingScaleResponse{
		ID:        m.ID,
		Name:      m.Name,
		ScaleType: m.ScaleType,
		Grades:    m.Grades,
		IsDefault: m.IsDefault,
		CreatedAt: m.CreatedAt,
	}
}
