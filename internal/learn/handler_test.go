package learn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/learn/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// ─── Test Helpers ────────────────────────────────────────────────────────────

type echoValidator struct{ v *validator.Validate }

func (cv *echoValidator) Validate(i any) error { return cv.v.Struct(i) }

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e
}

var (
	testParentID  = uuid.Must(uuid.NewV7())
	testFamilyID  = uuid.Must(uuid.NewV7())
	testStudentID = uuid.Must(uuid.NewV7())
)

func setupLearnRoutes(e *echo.Echo, svc LearningService) {
	auth := e.Group("/v1")
	auth.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			shared.SetAuthContext(c, &shared.AuthContext{
				ParentID:           testParentID,
				FamilyID:           testFamilyID,
				CoppaConsentStatus: "consented",
			})
			return next(c)
		}
	})
	NewHandler(svc).Register(auth)
}

// ─── Activity Def Handler Tests ─────────────────────────────────────────────

func TestCreateActivityDef_Success(t *testing.T) {
	e := newTestEcho()
	now := time.Now()
	expectedResp := ActivityDefResponse{
		ID:          uuid.Must(uuid.NewV7()),
		PublisherID: testFamilyID,
		Title:       "Math Worksheet",
		SubjectTags: []string{"math"},
		Attachments: []AttachmentInput{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mock := newMockLearningService()
	mock.createActivityDefFn = func(_ context.Context, cmd CreateActivityDefCommand) (ActivityDefResponse, error) {
		if cmd.Title != "Math Worksheet" {
			t.Errorf("expected title Math Worksheet, got %s", cmd.Title)
		}
		return expectedResp, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"publisher_id":"` + testFamilyID.String() + `","title":"Math Worksheet","subject_tags":["math"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/activity-defs", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListActivityDefs_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.listActivityDefsFn = func(_ context.Context, _ ActivityDefQuery) (PaginatedResponse[ActivityDefSummaryResponse], error) {
		return PaginatedResponse[ActivityDefSummaryResponse]{
			Data:    []ActivityDefSummaryResponse{{ID: uuid.Must(uuid.NewV7()), Title: "Test"}},
			HasMore: false,
		}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/activity-defs?subject=math", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetActivityDef_NotFound(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getActivityDefFn = func(_ context.Context, _ uuid.UUID) (ActivityDefResponse, error) {
		return ActivityDefResponse{}, &LearningError{Err: errActivityDefNotFound}
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/activity-defs/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Activity Log Handler Tests ─────────────────────────────────────────────

func TestLogActivity_Success(t *testing.T) {
	e := newTestEcho()
	now := time.Now()
	expectedResp := ActivityLogResponse{
		ID:           uuid.Must(uuid.NewV7()),
		StudentID:    testStudentID,
		Title:        "Read a book",
		SubjectTags:  []string{"language-arts"},
		Attachments:  []AttachmentInput{},
		ActivityDate: now,
		CreatedAt:    now,
	}

	mock := newMockLearningService()
	mock.logActivityFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, cmd LogActivityCommand) (ActivityLogResponse, error) {
		if cmd.Title != "Read a book" {
			t.Errorf("expected title 'Read a book', got %s", cmd.Title)
		}
		return expectedResp, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"title":"Read a book","subject_tags":["language-arts"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/"+testStudentID.String()+"/activities", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp ActivityLogResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if resp.Title != "Read a book" {
		t.Errorf("expected title 'Read a book', got %s", resp.Title)
	}
}

func TestLogActivity_InvalidStudentID(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	setupLearnRoutes(e, mock)

	body := `{"title":"Test"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/not-a-uuid/activities", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListActivityLogs_WithFilters(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.listActivityLogsFn = func(_ context.Context, _ *shared.FamilyScope, studentID uuid.UUID, query ActivityLogQuery) (PaginatedResponse[ActivityLogResponse], error) {
		if studentID != testStudentID {
			t.Errorf("expected student %s, got %s", testStudentID, studentID)
		}
		if query.Subject == nil || *query.Subject != "math" {
			t.Errorf("expected subject filter 'math'")
		}
		return PaginatedResponse[ActivityLogResponse]{Data: []ActivityLogResponse{}, HasMore: false}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/activities?subject=math", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteActivityLog_Success(t *testing.T) {
	e := newTestEcho()
	logID := uuid.Must(uuid.NewV7())

	mock := newMockLearningService()
	mock.deleteActivityLogFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, id uuid.UUID) error {
		if id != logID {
			t.Errorf("expected log %s, got %s", logID, id)
		}
		return nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodDelete, "/v1/learning/students/"+testStudentID.String()+"/activities/"+logID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Reading Item Handler Tests ─────────────────────────────────────────────

func TestCreateReadingItem_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.createReadingItemFn = func(_ context.Context, cmd CreateReadingItemCommand) (ReadingItemResponse, error) {
		if cmd.Title != "Charlotte's Web" {
			t.Errorf("expected title 'Charlotte's Web', got %s", cmd.Title)
		}
		return ReadingItemResponse{ID: uuid.Must(uuid.NewV7()), Title: cmd.Title, SubjectTags: []string{}, CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"publisher_id":"` + testFamilyID.String() + `","title":"Charlotte's Web","subject_tags":["language-arts"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/reading-items", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListReadingItems_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.listReadingItemsFn = func(_ context.Context, _ ReadingItemQuery) (PaginatedResponse[ReadingItemSummaryResponse], error) {
		return PaginatedResponse[ReadingItemSummaryResponse]{Data: []ReadingItemSummaryResponse{}, HasMore: false}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/reading-items?subject=math", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Reading Progress Handler Tests ─────────────────────────────────────────

func TestStartReading_Success(t *testing.T) {
	e := newTestEcho()
	itemID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.startReadingFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, cmd StartReadingCommand) (ReadingProgressResponse, error) {
		if cmd.ReadingItemID != itemID {
			t.Errorf("expected item %s, got %s", itemID, cmd.ReadingItemID)
		}
		return ReadingProgressResponse{ID: uuid.Must(uuid.NewV7()), Status: "to_read"}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"reading_item_id":"` + itemID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/"+testStudentID.String()+"/reading", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestStartReading_DuplicateReturns409(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.startReadingFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ StartReadingCommand) (ReadingProgressResponse, error) {
		return ReadingProgressResponse{}, &LearningError{Err: domain.ErrDuplicateReadingProgress}
	}
	setupLearnRoutes(e, mock)

	body := `{"reading_item_id":"` + uuid.Must(uuid.NewV7()).String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/"+testStudentID.String()+"/reading", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Journal Entry Handler Tests ────────────────────────────────────────────

func TestCreateJournalEntry_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.createJournalEntryFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, cmd CreateJournalEntryCommand) (JournalEntryResponse, error) {
		if cmd.EntryType != "narration" {
			t.Errorf("expected entry_type narration, got %s", cmd.EntryType)
		}
		return JournalEntryResponse{
			ID:          uuid.Must(uuid.NewV7()),
			EntryType:   cmd.EntryType,
			Content:     cmd.Content,
			SubjectTags: []string{},
			Attachments: []AttachmentInput{},
			EntryDate:   time.Now(),
			CreatedAt:   time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"entry_type":"narration","content":"Today we read about the Romans."}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/"+testStudentID.String()+"/journal", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteJournalEntry_Success(t *testing.T) {
	e := newTestEcho()
	entryID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.deleteJournalEntryFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, id uuid.UUID) error {
		if id != entryID {
			t.Errorf("expected entry %s, got %s", entryID, id)
		}
		return nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodDelete, "/v1/learning/students/"+testStudentID.String()+"/journal/"+entryID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Reading List Handler Tests ─────────────────────────────────────────────

func TestCreateReadingList_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.createReadingListFn = func(_ context.Context, _ *shared.FamilyScope, cmd CreateReadingListCommand) (ReadingListResponse, error) {
		return ReadingListResponse{ID: uuid.Must(uuid.NewV7()), Name: cmd.Name, CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"name":"Summer Reading"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/reading-lists", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListReadingLists_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.listReadingListsFn = func(_ context.Context, _ *shared.FamilyScope) ([]ReadingListSummaryResponse, error) {
		return []ReadingListSummaryResponse{{ID: uuid.Must(uuid.NewV7()), Name: "Summer Reading"}}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/reading-lists", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Taxonomy Handler Tests ─────────────────────────────────────────────────

func TestGetSubjectTaxonomy_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getSubjectTaxonomyFn = func(_ context.Context, _ *shared.FamilyScope, _ TaxonomyQuery) ([]SubjectTaxonomyResponse, error) {
		return []SubjectTaxonomyResponse{{ID: uuid.Must(uuid.NewV7()), Name: "Math", Slug: "math", Children: []SubjectTaxonomyResponse{}}}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/taxonomy", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateCustomSubject_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.createCustomSubjectFn = func(_ context.Context, _ *shared.FamilyScope, cmd CreateCustomSubjectCommand) (CustomSubjectResponse, error) {
		return CustomSubjectResponse{ID: uuid.Must(uuid.NewV7()), Name: cmd.Name, Slug: "robotics"}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"name":"Robotics"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/taxonomy/custom", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Artifact Link Handler Tests ────────────────────────────────────────────

func TestLinkArtifacts_Success(t *testing.T) {
	e := newTestEcho()
	sourceID := uuid.Must(uuid.NewV7())
	targetID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.linkArtifactsFn = func(_ context.Context, cmd CreateArtifactLinkCommand) (ArtifactLinkResponse, error) {
		if cmd.SourceID != sourceID || cmd.TargetID != targetID {
			t.Errorf("unexpected source/target IDs")
		}
		return ArtifactLinkResponse{
			ID: uuid.Must(uuid.NewV7()), SourceType: cmd.SourceType, SourceID: cmd.SourceID,
			TargetType: cmd.TargetType, TargetID: cmd.TargetID, Relationship: "about", CreatedAt: time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"source_type":"reading_item","source_id":"` + sourceID.String() + `","target_type":"activity_def","target_id":"` + targetID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/artifact-links", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestLinkArtifacts_DuplicateReturns409(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.linkArtifactsFn = func(_ context.Context, _ CreateArtifactLinkCommand) (ArtifactLinkResponse, error) {
		return ArtifactLinkResponse{}, &LearningError{Err: domain.ErrDuplicateLink}
	}
	setupLearnRoutes(e, mock)

	body := `{"source_type":"reading_item","source_id":"` + uuid.Must(uuid.NewV7()).String() + `","target_type":"activity_def","target_id":"` + uuid.Must(uuid.NewV7()).String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/artifact-links", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUnlinkArtifacts_Success(t *testing.T) {
	e := newTestEcho()
	linkID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.unlinkArtifactsFn = func(_ context.Context, id uuid.UUID, _ uuid.UUID) error {
		if id != linkID {
			t.Errorf("expected link %s, got %s", linkID, id)
		}
		return nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodDelete, "/v1/learning/artifact-links/"+linkID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetLinkedArtifacts_Success(t *testing.T) {
	e := newTestEcho()
	contentID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.getLinkedArtifactsFn = func(_ context.Context, cType string, cID uuid.UUID, _ LinkDirection) ([]ArtifactLinkResponse, error) {
		if cType != "reading_item" || cID != contentID {
			t.Errorf("unexpected content type/id")
		}
		return []ArtifactLinkResponse{}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/content/reading_item/"+contentID.String()+"/links", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Progress Handler Tests ────────────────────────────────────────────────

func TestGetProgressSummary_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getProgressSummaryFn = func(_ context.Context, _ *shared.FamilyScope, studentID uuid.UUID, _ ProgressQuery) (ProgressSummaryResponse, error) {
		if studentID != testStudentID {
			t.Errorf("expected student %s, got %s", testStudentID, studentID)
		}
		return ProgressSummaryResponse{
			StudentID:       studentID,
			TotalActivities: 5,
			TotalHours:      2.5,
			HoursBySubject:  []SubjectHoursResponse{},
		}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/progress", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetSubjectBreakdown_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getSubjectBreakdownFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ ProgressQuery) ([]SubjectProgressResponse, error) {
		return []SubjectProgressResponse{{SubjectSlug: "math", SubjectName: "Math", ActivityCount: 3}}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/progress/subjects", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetActivityTimeline_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getActivityTimelineFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ TimelineQuery) (PaginatedResponse[TimelineEntryResponse], error) {
		return PaginatedResponse[TimelineEntryResponse]{Data: []TimelineEntryResponse{}, HasMore: false}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/progress/timeline", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Export Handler Tests ───────────────────────────────────────────────────

func TestRequestDataExport_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.requestDataExportFn = func(_ context.Context, _ *shared.FamilyScope, _ RequestExportCommand) (ExportRequestResponse, error) {
		return ExportRequestResponse{ID: uuid.Must(uuid.NewV7()), Status: "pending", CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"format":"json"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/export", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("expected 202, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRequestDataExport_AlreadyInProgressReturns409(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.requestDataExportFn = func(_ context.Context, _ *shared.FamilyScope, _ RequestExportCommand) (ExportRequestResponse, error) {
		return ExportRequestResponse{}, &LearningError{Err: domain.ErrExportAlreadyInProgress}
	}
	setupLearnRoutes(e, mock)

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/export", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetExportRequest_Success(t *testing.T) {
	e := newTestEcho()
	exportID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.getExportRequestFn = func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (ExportRequestResponse, error) {
		if id != exportID {
			t.Errorf("expected export %s, got %s", exportID, id)
		}
		return ExportRequestResponse{ID: id, Status: "completed", CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/export/"+exportID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Tool Handler Tests ─────────────────────────────────────────────────────

func TestGetResolvedTools_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getResolvedToolsFn = func(_ context.Context, _ *shared.FamilyScope) ([]ActiveToolResponse, error) {
		return []ActiveToolResponse{{ToolID: uuid.Must(uuid.NewV7()), Slug: "mathletics", DisplayName: "Mathletics"}}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/tools", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetStudentTools_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getStudentToolsFn = func(_ context.Context, _ *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error) {
		if studentID != testStudentID {
			t.Errorf("expected student %s, got %s", testStudentID, studentID)
		}
		return []ActiveToolResponse{}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/tools", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Question Handler Tests ────────────────────────────────────────────────

func TestCreateQuestion_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.createQuestionFn = func(_ context.Context, cmd CreateQuestionCommand) (QuestionResponse, error) {
		if cmd.QuestionType != "multiple_choice" {
			t.Errorf("expected multiple_choice, got %s", cmd.QuestionType)
		}
		return QuestionResponse{
			ID: uuid.Must(uuid.NewV7()), PublisherID: cmd.PublisherID,
			QuestionType: cmd.QuestionType, Content: cmd.Content,
			SubjectTags: []string{}, AutoScorable: true, Points: 1.0, CreatedAt: time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"publisher_id":"` + testFamilyID.String() + `","question_type":"multiple_choice","content":"What is 2+2?","answer_data":{"correct":"4"}}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/questions", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListQuestions_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.listQuestionsFn = func(_ context.Context, query QuestionQuery) (PaginatedResponse[QuestionSummaryResponse], error) {
		if query.QuestionType == nil || *query.QuestionType != "multiple_choice" {
			t.Errorf("expected question_type filter 'multiple_choice'")
		}
		return PaginatedResponse[QuestionSummaryResponse]{Data: []QuestionSummaryResponse{}, HasMore: false}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/questions?question_type=multiple_choice", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Quiz Definition Handler Tests ─────────────────────────────────────────

func TestCreateQuizDef_Success(t *testing.T) {
	e := newTestEcho()
	qID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.createQuizDefFn = func(_ context.Context, cmd CreateQuizDefCommand) (QuizDefResponse, error) {
		if cmd.Title != "Math Quiz" {
			t.Errorf("expected title Math Quiz, got %s", cmd.Title)
		}
		return QuizDefResponse{
			ID: uuid.Must(uuid.NewV7()), PublisherID: cmd.PublisherID, Title: cmd.Title,
			SubjectTags: []string{}, PassingScorePercent: 70, QuestionCount: 1, CreatedAt: time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"publisher_id":"` + testFamilyID.String() + `","title":"Math Quiz","question_ids":[{"question_id":"` + qID.String() + `","sort_order":0}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/quiz-defs", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetQuizDef_Success(t *testing.T) {
	e := newTestEcho()
	defID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.getQuizDefFn = func(_ context.Context, id uuid.UUID, includeAnswers bool) (QuizDefDetailResponse, error) {
		if id != defID {
			t.Errorf("expected def %s, got %s", defID, id)
		}
		return QuizDefDetailResponse{
			QuizDefResponse: QuizDefResponse{ID: id, Title: "Test Quiz", SubjectTags: []string{}, CreatedAt: time.Now()},
			Questions:       []QuizQuestionResponse{},
		}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/quiz-defs/"+defID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetQuizDef_NotFoundReturns404(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getQuizDefFn = func(_ context.Context, _ uuid.UUID, _ bool) (QuizDefDetailResponse, error) {
		return QuizDefDetailResponse{}, &LearningError{Err: domain.ErrQuizDefNotFound}
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/quiz-defs/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Quiz Session Handler Tests ─────────────────────────────────────────────

func TestStartQuizSession_Success(t *testing.T) {
	e := newTestEcho()
	quizDefID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.startQuizSessionFn = func(_ context.Context, _ *shared.FamilyScope, studentID uuid.UUID, cmd StartQuizSessionCommand) (QuizSessionResponse, error) {
		if studentID != testStudentID {
			t.Errorf("expected student %s, got %s", testStudentID, studentID)
		}
		if cmd.QuizDefID != quizDefID {
			t.Errorf("expected quiz def %s, got %s", quizDefID, cmd.QuizDefID)
		}
		return QuizSessionResponse{
			ID: uuid.Must(uuid.NewV7()), StudentID: studentID, QuizDefID: cmd.QuizDefID,
			Status: "in_progress", CreatedAt: time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"quiz_def_id":"` + quizDefID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/"+testStudentID.String()+"/quiz-sessions", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetQuizSession_Success(t *testing.T) {
	e := newTestEcho()
	sessionID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.getQuizSessionFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, sID uuid.UUID) (QuizSessionResponse, error) {
		if sID != sessionID {
			t.Errorf("expected session %s, got %s", sessionID, sID)
		}
		return QuizSessionResponse{ID: sID, Status: "in_progress", CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/quiz-sessions/"+sessionID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetQuizSession_NotFoundReturns404(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getQuizSessionFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ uuid.UUID) (QuizSessionResponse, error) {
		return QuizSessionResponse{}, &LearningError{Err: domain.ErrQuizSessionNotFound}
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/quiz-sessions/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestScoreQuizSession_Success(t *testing.T) {
	e := newTestEcho()
	sessionID := uuid.Must(uuid.NewV7())
	qID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.scoreQuizSessionFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, sID uuid.UUID, cmd ScoreQuizCommand) (QuizSessionResponse, error) {
		if sID != sessionID {
			t.Errorf("expected session %s, got %s", sessionID, sID)
		}
		if len(cmd.Scores) != 1 {
			t.Errorf("expected 1 score, got %d", len(cmd.Scores))
		}
		score := 8.0
		maxScore := 10.0
		passed := true
		return QuizSessionResponse{ID: sID, Status: "scored", Score: &score, MaxScore: &maxScore, Passed: &passed, CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"scores":[{"question_id":"` + qID.String() + `","points_awarded":8}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/"+testStudentID.String()+"/quiz-sessions/"+sessionID.String()+"/score", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestScoreQuizSession_NotSubmittedReturns422(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.scoreQuizSessionFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ uuid.UUID, _ ScoreQuizCommand) (QuizSessionResponse, error) {
		return QuizSessionResponse{}, &LearningError{Err: domain.ErrQuizSessionNotSubmitted}
	}
	setupLearnRoutes(e, mock)

	body := `{"scores":[{"question_id":"` + uuid.Must(uuid.NewV7()).String() + `","points_awarded":5}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/"+testStudentID.String()+"/quiz-sessions/"+uuid.Must(uuid.NewV7()).String()+"/score", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateQuizSession_Success(t *testing.T) {
	e := newTestEcho()
	sessionID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.updateQuizSessionFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, sID uuid.UUID, _ UpdateQuizSessionCommand) (QuizSessionResponse, error) {
		return QuizSessionResponse{ID: sID, Status: "in_progress", CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"answers":{"q1":"a"}}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/learning/students/"+testStudentID.String()+"/quiz-sessions/"+sessionID.String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Sequence Definition Handler Tests ─────────────────────────────────────

func TestCreateSequenceDef_Success(t *testing.T) {
	e := newTestEcho()
	cID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.createSequenceDefFn = func(_ context.Context, cmd CreateSequenceDefCommand) (SequenceDefResponse, error) {
		if cmd.Title != "Math Sequence" {
			t.Errorf("expected title Math Sequence, got %s", cmd.Title)
		}
		return SequenceDefResponse{
			ID: uuid.Must(uuid.NewV7()), PublisherID: cmd.PublisherID, Title: cmd.Title,
			SubjectTags: []string{}, IsLinear: true, CreatedAt: time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"publisher_id":"` + testFamilyID.String() + `","title":"Math Sequence","items":[{"content_type":"activity_def","content_id":"` + cID.String() + `","sort_order":0}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/sequences", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetSequenceDef_Success(t *testing.T) {
	e := newTestEcho()
	defID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.getSequenceDefFn = func(_ context.Context, id uuid.UUID) (SequenceDefDetailResponse, error) {
		if id != defID {
			t.Errorf("expected def %s, got %s", defID, id)
		}
		return SequenceDefDetailResponse{
			SequenceDefResponse: SequenceDefResponse{ID: id, Title: "Test", SubjectTags: []string{}, CreatedAt: time.Now()},
			Items:               []SequenceItemResponse{},
		}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/sequences/"+defID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetSequenceDef_NotFoundReturns404(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getSequenceDefFn = func(_ context.Context, _ uuid.UUID) (SequenceDefDetailResponse, error) {
		return SequenceDefDetailResponse{}, &LearningError{Err: domain.ErrSequenceDefNotFound}
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/sequences/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Sequence Progress Handler Tests ───────────────────────────────────────

func TestStartSequence_Success(t *testing.T) {
	e := newTestEcho()
	seqDefID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.startSequenceFn = func(_ context.Context, _ *shared.FamilyScope, studentID uuid.UUID, cmd StartSequenceCommand) (SequenceProgressResponse, error) {
		if studentID != testStudentID {
			t.Errorf("expected student %s, got %s", testStudentID, studentID)
		}
		return SequenceProgressResponse{
			ID: uuid.Must(uuid.NewV7()), StudentID: studentID, SequenceDefID: cmd.SequenceDefID,
			Status: "in_progress", CreatedAt: time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"sequence_def_id":"` + seqDefID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/"+testStudentID.String()+"/sequence-progress", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetSequenceProgress_Success(t *testing.T) {
	e := newTestEcho()
	progressID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.getSequenceProgressFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, pID uuid.UUID) (SequenceProgressResponse, error) {
		return SequenceProgressResponse{ID: pID, Status: "in_progress", CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/sequence-progress/"+progressID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Assignment Handler Tests ──────────────────────────────────────────────

func TestCreateAssignment_Success(t *testing.T) {
	e := newTestEcho()
	contentID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.createAssignmentFn = func(_ context.Context, _ *shared.FamilyScope, studentID uuid.UUID, cmd CreateAssignmentCommand) (AssignmentResponse, error) {
		if studentID != testStudentID {
			t.Errorf("expected student %s, got %s", testStudentID, studentID)
		}
		return AssignmentResponse{
			ID: uuid.Must(uuid.NewV7()), StudentID: studentID, AssignedBy: cmd.AssignedBy,
			ContentType: cmd.ContentType, ContentID: cmd.ContentID, Status: "assigned",
			AssignedAt: time.Now(), CreatedAt: time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"content_type":"reading_item","content_id":"` + contentID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/learning/students/"+testStudentID.String()+"/assignments", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestListAssignments_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.listAssignmentsFn = func(_ context.Context, _ *shared.FamilyScope, studentID uuid.UUID, query AssignmentQuery) (PaginatedResponse[AssignmentResponse], error) {
		if studentID != testStudentID {
			t.Errorf("expected student %s, got %s", testStudentID, studentID)
		}
		if query.Status == nil || *query.Status != "assigned" {
			t.Errorf("expected status filter 'assigned'")
		}
		return PaginatedResponse[AssignmentResponse]{Data: []AssignmentResponse{}, HasMore: false}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/assignments?status=assigned", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAssignment_Success(t *testing.T) {
	e := newTestEcho()
	assignmentID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.deleteAssignmentFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, id uuid.UUID) error {
		if id != assignmentID {
			t.Errorf("expected assignment %s, got %s", assignmentID, id)
		}
		return nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodDelete, "/v1/learning/students/"+testStudentID.String()+"/assignments/"+assignmentID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateAssignment_InvalidTransitionReturns422(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.updateAssignmentFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ uuid.UUID, _ UpdateAssignmentCommand) (AssignmentResponse, error) {
		return AssignmentResponse{}, &LearningError{Err: domain.ErrInvalidAssignmentStatusTransition}
	}
	setupLearnRoutes(e, mock)

	body := `{"status":"completed"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/learning/students/"+testStudentID.String()+"/assignments/"+uuid.Must(uuid.NewV7()).String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Video Definition Handler Tests ────────────────────────────────────────

func TestListVideoDefs_Success(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.listVideoDefsFn = func(_ context.Context, query VideoDefQuery) (PaginatedResponse[VideoDefResponse], error) {
		if query.Subject == nil || *query.Subject != "science" {
			t.Errorf("expected subject filter 'science'")
		}
		return PaginatedResponse[VideoDefResponse]{Data: []VideoDefResponse{}, HasMore: false}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/videos?subject=science", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetVideoDef_Success(t *testing.T) {
	e := newTestEcho()
	defID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.getVideoDefFn = func(_ context.Context, id uuid.UUID) (VideoDefResponse, error) {
		if id != defID {
			t.Errorf("expected def %s, got %s", defID, id)
		}
		return VideoDefResponse{ID: id, Title: "Test Video", SubjectTags: []string{}, VideoSource: "youtube", CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/videos/"+defID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetVideoDef_NotFoundReturns404(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	mock.getVideoDefFn = func(_ context.Context, _ uuid.UUID) (VideoDefResponse, error) {
		return VideoDefResponse{}, &LearningError{Err: domain.ErrVideoDefNotFound}
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/videos/"+uuid.Must(uuid.NewV7()).String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Video Progress Handler Tests ──────────────────────────────────────────

func TestUpdateVideoProgress_Success(t *testing.T) {
	e := newTestEcho()
	videoDefID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.updateVideoProgressFn = func(_ context.Context, _ *shared.FamilyScope, studentID uuid.UUID, cmd UpdateVideoProgressCommand) (VideoProgressResponse, error) {
		if studentID != testStudentID {
			t.Errorf("expected student %s, got %s", testStudentID, studentID)
		}
		return VideoProgressResponse{
			ID: uuid.Must(uuid.NewV7()), StudentID: studentID, VideoDefID: cmd.VideoDefID,
			WatchedSeconds: 120, CreatedAt: time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"video_def_id":"` + videoDefID.String() + `","watched_seconds":120,"last_position_seconds":120}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/learning/students/"+testStudentID.String()+"/video-progress", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Reading Item Update Handler Tests ───────────────────────────────────────

func TestUpdateReadingItem_Success(t *testing.T) {
	e := newTestEcho()
	itemID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.updateReadingItemFn = func(_ context.Context, id uuid.UUID, cmd UpdateReadingItemCommand) (ReadingItemResponse, error) {
		if id != itemID {
			t.Errorf("expected item %s, got %s", itemID, id)
		}
		if cmd.Title == nil || *cmd.Title != "Updated Title" {
			t.Errorf("expected title 'Updated Title'")
		}
		return ReadingItemResponse{ID: itemID, Title: "Updated Title", SubjectTags: []string{}, CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"title":"Updated Title"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/learning/reading-items/"+itemID.String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateReadingItem_InvalidID(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	setupLearnRoutes(e, mock)

	body := `{"title":"Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/learning/reading-items/not-a-uuid", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── Question Update Handler Tests ──────────────────────────────────────────

func TestUpdateQuestion_Success(t *testing.T) {
	e := newTestEcho()
	questionID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.updateQuestionFn = func(_ context.Context, id uuid.UUID, cmd UpdateQuestionCommand) (QuestionResponse, error) {
		if id != questionID {
			t.Errorf("expected question %s, got %s", questionID, id)
		}
		if cmd.Content == nil || *cmd.Content != "Updated content" {
			t.Errorf("expected content 'Updated content'")
		}
		return QuestionResponse{
			ID: questionID, Content: "Updated content", SubjectTags: []string{},
			MediaAttachments: json.RawMessage("[]"), AnswerData: json.RawMessage("{}"),
			CreatedAt: time.Now(),
		}, nil
	}
	setupLearnRoutes(e, mock)

	body := `{"content":"Updated content"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/learning/questions/"+questionID.String(), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdateQuestion_InvalidID(t *testing.T) {
	e := newTestEcho()
	mock := newMockLearningService()
	setupLearnRoutes(e, mock)

	body := `{"content":"Updated"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/learning/questions/not-a-uuid", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetVideoProgress_Success(t *testing.T) {
	e := newTestEcho()
	videoDefID := uuid.Must(uuid.NewV7())
	mock := newMockLearningService()
	mock.getVideoProgressFn = func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, vID uuid.UUID) (VideoProgressResponse, error) {
		if vID != videoDefID {
			t.Errorf("expected video %s, got %s", videoDefID, vID)
		}
		return VideoProgressResponse{ID: uuid.Must(uuid.NewV7()), VideoDefID: vID, WatchedSeconds: 60, CreatedAt: time.Now()}, nil
	}
	setupLearnRoutes(e, mock)

	req := httptest.NewRequest(http.MethodGet, "/v1/learning/students/"+testStudentID.String()+"/video-progress/"+videoDefID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
