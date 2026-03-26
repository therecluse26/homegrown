package plan

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Area A: Schedule Item CRUD [17-planning §5, §10.1, §17]
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreateScheduleItem_A1_ValidInputReturnsNewUUID(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	var createdItem *ScheduleItem
	repo := &stubScheduleItemRepo{
		createFn: func(_ context.Context, _ *shared.FamilyScope, item *ScheduleItem) error {
			item.ID = uuid.Must(uuid.NewV7())
			createdItem = item
			return nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	id, err := svc.CreateScheduleItem(context.Background(), auth, scope, CreateScheduleItemInput{
		Title:     "Math Lesson",
		StartDate: time.Now(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if createdItem == nil {
		t.Fatal("expected repo.Create to be called")
	}
	if createdItem.Title != "Math Lesson" {
		t.Errorf("expected title 'Math Lesson', got %q", createdItem.Title)
	}
}

func TestCreateScheduleItem_A2_DefaultsCategoryToCustom(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	var createdItem *ScheduleItem
	repo := &stubScheduleItemRepo{
		createFn: func(_ context.Context, _ *shared.FamilyScope, item *ScheduleItem) error {
			item.ID = uuid.Must(uuid.NewV7())
			createdItem = item
			return nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	_, err := svc.CreateScheduleItem(context.Background(), auth, scope, CreateScheduleItemInput{
		Title:     "Study Time",
		StartDate: time.Now(),
		// Category is nil — should default to "custom"
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdItem.Category != ScheduleCategoryCustom {
		t.Errorf("expected category 'custom', got %q", createdItem.Category)
	}
}

func TestCreateScheduleItem_A3_ValidatesStudentBelongsToFamily(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	studentID := uuid.Must(uuid.NewV7())

	iamSvc := &stubIamService{
		studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
			return false, nil // student NOT in family
		},
	}
	svc := newTestService(&stubScheduleItemRepo{}, iamSvc, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	_, err := svc.CreateScheduleItem(context.Background(), auth, scope, CreateScheduleItemInput{
		Title:     "Math",
		StartDate: time.Now(),
		StudentID: &studentID,
	})
	if !errors.Is(err, ErrStudentNotInFamily) {
		t.Errorf("expected ErrStudentNotInFamily, got %v", err)
	}
}

func TestUpdateScheduleItem_A4_UpdatesExistingItem(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return &ScheduleItem{ID: itemID, Title: "Old Title"}, nil
		},
		updateFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ *UpdateScheduleItemInput) error {
			return nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	newTitle := "New Title"
	err := svc.UpdateScheduleItem(context.Background(), auth, scope, itemID, UpdateScheduleItemInput{
		Title: &newTitle,
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateScheduleItem_A5_ReturnsErrItemNotFound(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return nil, nil // not found
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.UpdateScheduleItem(context.Background(), auth, scope, itemID, UpdateScheduleItemInput{})
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestDeleteScheduleItem_A6_DeletesExistingItem(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())

	var deleteCalled bool
	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return &ScheduleItem{ID: itemID}, nil
		},
		deleteFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.DeleteScheduleItem(context.Background(), auth, scope, itemID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected repo.Delete to be called")
	}
}

func TestDeleteScheduleItem_A7_ReturnsErrItemNotFound(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.DeleteScheduleItem(context.Background(), auth, scope, itemID)
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area B: Completion Workflow [17-planning §10.1, §17]
// ═══════════════════════════════════════════════════════════════════════════════

func TestCompleteScheduleItem_B8_MarksItemCompleted(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())

	var markCompletedCalled bool
	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return &ScheduleItem{ID: itemID, IsCompleted: false}, nil
		},
		markCompletedFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ time.Time) error {
			markCompletedCalled = true
			return nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.CompleteScheduleItem(context.Background(), auth, scope, itemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !markCompletedCalled {
		t.Error("expected repo.MarkCompleted to be called")
	}
}

func TestCompleteScheduleItem_B9_ReturnsErrItemNotFound(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.CompleteScheduleItem(context.Background(), auth, scope, uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestCompleteScheduleItem_B10_ReturnsErrAlreadyCompleted(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())
	now := time.Now()

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return &ScheduleItem{ID: itemID, IsCompleted: true, CompletedAt: &now}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.CompleteScheduleItem(context.Background(), auth, scope, itemID)
	if !errors.Is(err, ErrAlreadyCompleted) {
		t.Errorf("expected ErrAlreadyCompleted, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area C: Log as Activity [17-planning §10.2, §17]
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogAsActivity_C11_CreatesActivityAndLinksBack(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())
	activityID := uuid.Must(uuid.NewV7())
	now := time.Now()

	var linkedCalled bool
	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return &ScheduleItem{
				ID:          itemID,
				Title:       "Math Lesson",
				StartDate:   now,
				IsCompleted: true,
				CompletedAt: &now,
			}, nil
		},
		setLinkedActivityFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, aID uuid.UUID) error {
			if aID != activityID {
				t.Errorf("expected activity ID %v, got %v", activityID, aID)
			}
			linkedCalled = true
			return nil
		},
	}
	learnSvc := &stubLearningService{
		logActivityFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _ string, _ time.Time, _ *int, _ *uuid.UUID, _ *string, _ []string) (uuid.UUID, error) {
			return activityID, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, learnSvc, &stubComplianceService{}, &stubSocialService{})

	result, err := svc.LogAsActivity(context.Background(), auth, scope, itemID, LogAsActivityInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != activityID {
		t.Errorf("expected activity ID %v, got %v", activityID, result)
	}
	if !linkedCalled {
		t.Error("expected repo.SetLinkedActivity to be called")
	}
}

func TestLogAsActivity_C12_ReturnsErrItemNotFound(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	_, err := svc.LogAsActivity(context.Background(), auth, scope, uuid.Must(uuid.NewV7()), LogAsActivityInput{})
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

func TestLogAsActivity_C13_ReturnsErrAlreadyLogged(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())
	existingActivityID := uuid.Must(uuid.NewV7())
	now := time.Now()

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return &ScheduleItem{
				ID:               itemID,
				IsCompleted:      true,
				CompletedAt:      &now,
				LinkedActivityID: &existingActivityID,
			}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	_, err := svc.LogAsActivity(context.Background(), auth, scope, itemID, LogAsActivityInput{})
	if !errors.Is(err, ErrAlreadyLogged) {
		t.Errorf("expected ErrAlreadyLogged, got %v", err)
	}
}

func TestLogAsActivity_C14_RequiresCompletion(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return &ScheduleItem{
				ID:          itemID,
				IsCompleted: false, // NOT completed
			}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	_, err := svc.LogAsActivity(context.Background(), auth, scope, itemID, LogAsActivityInput{})
	if !errors.Is(err, ErrNotCompleted) {
		t.Errorf("expected ErrNotCompleted, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area D: List / Query [17-planning §5]
// ═══════════════════════════════════════════════════════════════════════════════

func TestListScheduleItems_D15_ReturnsPaginatedResults(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	now := time.Now()

	items := []ScheduleItem{
		{ID: uuid.Must(uuid.NewV7()), Title: "Item 1", Category: ScheduleCategoryLesson, StartDate: now, CreatedAt: now},
		{ID: uuid.Must(uuid.NewV7()), Title: "Item 2", Category: ScheduleCategoryCustom, StartDate: now, CreatedAt: now},
	}
	repo := &stubScheduleItemRepo{
		listFilteredFn: func(_ context.Context, _ *shared.FamilyScope, _ *ScheduleItemQuery, _ *shared.PaginationParams) ([]ScheduleItem, error) {
			return items, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	result, err := svc.ListScheduleItems(context.Background(), auth, scope, ScheduleItemQuery{}, &shared.PaginationParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Data))
	}
	if result.Data[0].Title != "Item 1" {
		t.Errorf("expected first item title 'Item 1', got %q", result.Data[0].Title)
	}
}

func TestListScheduleItems_D16_PassesFiltersToRepository(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	studentID := uuid.Must(uuid.NewV7())
	completed := true

	var capturedQuery *ScheduleItemQuery
	repo := &stubScheduleItemRepo{
		listFilteredFn: func(_ context.Context, _ *shared.FamilyScope, query *ScheduleItemQuery, _ *shared.PaginationParams) ([]ScheduleItem, error) {
			capturedQuery = query
			return []ScheduleItem{}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	_, err := svc.ListScheduleItems(context.Background(), auth, scope, ScheduleItemQuery{
		StudentID:   &studentID,
		IsCompleted: &completed,
	}, &shared.PaginationParams{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedQuery == nil {
		t.Fatal("expected query to be passed to repo")
	}
	if capturedQuery.StudentID == nil || *capturedQuery.StudentID != studentID {
		t.Error("expected student_id filter to be passed")
	}
	if capturedQuery.IsCompleted == nil || *capturedQuery.IsCompleted != true {
		t.Error("expected is_completed filter to be passed")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area E: Calendar Aggregation [17-planning §9, §17]
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetCalendar_E17_ReturnsErrInvalidDateRange(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	svc := newTestService(&stubScheduleItemRepo{}, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	now := time.Now()
	_, err := svc.GetCalendar(context.Background(), auth, scope, CalendarQuery{
		Start: now,
		End:   now.Add(-24 * time.Hour), // end before start
	})
	if !errors.Is(err, ErrInvalidDateRange) {
		t.Errorf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestGetCalendar_E18_ReturnsErrDateRangeTooLarge(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	svc := newTestService(&stubScheduleItemRepo{}, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	now := time.Now()
	_, err := svc.GetCalendar(context.Background(), auth, scope, CalendarQuery{
		Start: now,
		End:   now.Add(91 * 24 * time.Hour), // > 90 days
	})
	if !errors.Is(err, ErrDateRangeTooLarge) {
		t.Errorf("expected ErrDateRangeTooLarge, got %v", err)
	}
}

func TestGetCalendar_E19_AggregatesScheduleItems(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	now := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{
				{ID: uuid.Must(uuid.NewV7()), Title: "Math", StartDate: now, Category: ScheduleCategoryLesson},
			}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	result, err := svc.GetCalendar(context.Background(), auth, scope, CalendarQuery{
		Start: now,
		End:   now.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have at least one day with items
	found := false
	for _, day := range result.Days {
		for _, item := range day.Items {
			if item.Source == CalendarSourceSchedule && item.Title == "Math" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected schedule item 'Math' in calendar days")
	}
}

func TestGetCalendar_E20_AggregatesActivities(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	now := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	learnSvc := &stubLearningService{
		listActivitiesForCalendarFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ActivitySummary, error) {
			return []ActivitySummary{
				{ID: uuid.Must(uuid.NewV7()), Title: "Reading Log", Date: now},
			}, nil
		},
	}
	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, learnSvc, &stubComplianceService{}, &stubSocialService{})

	result, err := svc.GetCalendar(context.Background(), auth, scope, CalendarQuery{Start: now, End: now.Add(24 * time.Hour)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, day := range result.Days {
		for _, item := range day.Items {
			if item.Source == CalendarSourceActivities && item.Title == "Reading Log" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected activity 'Reading Log' in calendar days")
	}
}

func TestGetCalendar_E21_AggregatesAttendance(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	now := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	complySvc := &stubComplianceService{
		getAttendanceRangeFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]AttendanceSummary, error) {
			return []AttendanceSummary{
				{Date: now, Status: "present"},
			}, nil
		},
	}
	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, complySvc, &stubSocialService{})

	result, err := svc.GetCalendar(context.Background(), auth, scope, CalendarQuery{Start: now, End: now.Add(24 * time.Hour)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, day := range result.Days {
		for _, item := range day.Items {
			if item.Source == CalendarSourceAttendance {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected attendance record in calendar days")
	}
}

func TestGetCalendar_E22_AggregatesEvents(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	now := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	socialSvc := &stubSocialService{
		getEventsForCalendarFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _, _ time.Time) ([]EventSummary, error) {
			return []EventSummary{
				{ID: uuid.Must(uuid.NewV7()), Title: "Co-op Day", Date: now},
			}, nil
		},
	}
	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, socialSvc)

	result, err := svc.GetCalendar(context.Background(), auth, scope, CalendarQuery{Start: now, End: now.Add(24 * time.Hour)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, day := range result.Days {
		for _, item := range day.Items {
			if item.Source == CalendarSourceEvents && item.Title == "Co-op Day" {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected event 'Co-op Day' in calendar days")
	}
}

func TestGetCalendar_E23_MergesSourcesIntoCalendarDays(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	day1 := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 3, 24, 0, 0, 0, 0, time.UTC)

	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{
				{ID: uuid.Must(uuid.NewV7()), Title: "Math", StartDate: day1, Category: ScheduleCategoryLesson},
			}, nil
		},
	}
	learnSvc := &stubLearningService{
		listActivitiesForCalendarFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ActivitySummary, error) {
			return []ActivitySummary{
				{ID: uuid.Must(uuid.NewV7()), Title: "Reading", Date: day2},
			}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, learnSvc, &stubComplianceService{}, &stubSocialService{})

	result, err := svc.GetCalendar(context.Background(), auth, scope, CalendarQuery{
		Start: day1,
		End:   day2.Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Days) < 2 {
		t.Fatalf("expected at least 2 days, got %d", len(result.Days))
	}
}

func TestGetCalendar_E24_FiltersBySource(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	now := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{
				{ID: uuid.Must(uuid.NewV7()), Title: "Math", StartDate: now, Category: ScheduleCategoryLesson},
			}, nil
		},
	}
	// Activities stub should NOT be called if filtered to schedule-only
	learnSvc := &stubLearningService{
		listActivitiesForCalendarFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ActivitySummary, error) {
			return []ActivitySummary{
				{ID: uuid.Must(uuid.NewV7()), Title: "Should Not Appear", Date: now},
			}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, learnSvc, &stubComplianceService{}, &stubSocialService{})

	result, err := svc.GetCalendar(context.Background(), auth, scope, CalendarQuery{
		Start:   now,
		End:     now.Add(24 * time.Hour),
		Sources: []CalendarSource{CalendarSourceSchedule}, // only schedule
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, day := range result.Days {
		for _, item := range day.Items {
			if item.Source == CalendarSourceActivities {
				t.Error("activities should be filtered out when Sources = [schedule]")
			}
		}
	}
}

func TestGetCalendar_E25_ReturnsEmptyDaysForDatesWithNoItems(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	start := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 26, 0, 0, 0, 0, time.UTC) // 3-day range

	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{}, nil // nothing
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	result, err := svc.GetCalendar(context.Background(), auth, scope, CalendarQuery{Start: start, End: end})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should have 3 days: 23, 24, 25
	if len(result.Days) != 3 {
		t.Errorf("expected 3 days (empty), got %d", len(result.Days))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area F: Day View [17-planning §5]
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetDayView_F26_ReturnsDayGroupedBySource(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	day := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	endOfDay := day.Add(24 * time.Hour)

	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{
				{ID: uuid.Must(uuid.NewV7()), Title: "Math", StartDate: day, Category: ScheduleCategoryLesson, CreatedAt: day},
			}, nil
		},
	}
	learnSvc := &stubLearningService{
		listActivitiesForCalendarFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ActivitySummary, error) {
			return []ActivitySummary{
				{ID: uuid.Must(uuid.NewV7()), Title: "Reading", Date: day},
			}, nil
		},
	}
	socialSvc := &stubSocialService{
		getEventsForCalendarFn: func(_ context.Context, _ *shared.AuthContext, _ *shared.FamilyScope, s, e time.Time) ([]EventSummary, error) {
			if !s.Equal(day) || !e.Equal(endOfDay) {
				t.Error("expected day boundaries to be passed")
			}
			return []EventSummary{
				{ID: uuid.Must(uuid.NewV7()), Title: "Co-op", Date: day},
			}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, learnSvc, &stubComplianceService{}, socialSvc)

	result, err := svc.GetDayView(context.Background(), auth, scope, day, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ScheduleItems) != 1 {
		t.Errorf("expected 1 schedule item, got %d", len(result.ScheduleItems))
	}
	if len(result.Activities) != 1 {
		t.Errorf("expected 1 activity, got %d", len(result.Activities))
	}
	if len(result.Events) != 1 {
		t.Errorf("expected 1 event, got %d", len(result.Events))
	}
}

func TestGetDayView_F27_EnrichesWithStudentNames(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	day := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	studentID := uuid.Must(uuid.NewV7())

	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{
				{ID: uuid.Must(uuid.NewV7()), Title: "Math", StartDate: day, StudentID: &studentID, Category: ScheduleCategoryLesson, CreatedAt: day},
			}, nil
		},
	}
	iamSvc := &stubIamService{
		getStudentNameFn: func(_ context.Context, id uuid.UUID) (string, error) {
			if id == studentID {
				return "Alice", nil
			}
			return "", nil
		},
	}
	svc := newTestService(repo, iamSvc, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	result, err := svc.GetDayView(context.Background(), auth, scope, day, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.ScheduleItems) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.ScheduleItems))
	}
	if result.ScheduleItems[0].StudentName == nil || *result.ScheduleItems[0].StudentName != "Alice" {
		t.Errorf("expected student name 'Alice', got %v", result.ScheduleItems[0].StudentName)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area G: Print View [17-planning §13, §17]
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetPrintView_G28_ReturnsErrInvalidDateRange(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	svc := newTestService(&stubScheduleItemRepo{}, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	now := time.Now()
	_, err := svc.GetPrintView(context.Background(), auth, scope, now, now.Add(-time.Hour), nil)
	if !errors.Is(err, ErrInvalidDateRange) {
		t.Errorf("expected ErrInvalidDateRange, got %v", err)
	}
}

func TestGetPrintView_G29_ReturnsHTMLString(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	day := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)

	repo := &stubScheduleItemRepo{
		listByDateRangeFn: func(_ context.Context, _ *shared.FamilyScope, _, _ time.Time, _ *uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{
				{ID: uuid.Must(uuid.NewV7()), Title: "Math Lesson", StartDate: day, Category: ScheduleCategoryLesson},
			}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	html, err := svc.GetPrintView(context.Background(), auth, scope, day, day.Add(24*time.Hour), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(html, "<html") {
		t.Error("expected HTML output to contain <html tag")
	}
	if !strings.Contains(html, "Math Lesson") {
		t.Error("expected HTML to contain schedule item title")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area H: Data Lifecycle [17-planning §14]
// ═══════════════════════════════════════════════════════════════════════════════

func TestExportData_H30_ReturnsJSON(t *testing.T) {
	scope := testScope()
	now := time.Now()

	repo := &stubScheduleItemRepo{
		listAllByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) ([]ScheduleItem, error) {
			return []ScheduleItem{
				{ID: uuid.Must(uuid.NewV7()), Title: "Math", StartDate: now, Category: ScheduleCategoryLesson, CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	data, err := svc.ExportData(context.Background(), scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(data) {
		t.Error("expected valid JSON")
	}
	if !strings.Contains(string(data), "Math") {
		t.Error("expected export data to contain schedule item")
	}
}

func TestDeleteData_H31_RemovesAllPlanData(t *testing.T) {
	scope := testScope()

	var deleteItemsCalled, deleteTemplatesCalled bool
	repo := &stubScheduleItemRepo{
		deleteAllByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) error {
			deleteItemsCalled = true
			return nil
		},
	}
	tmplRepo := &stubScheduleTemplateRepo{
		deleteAllByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) error {
			deleteTemplatesCalled = true
			return nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: repo, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	err := svc.DeleteData(context.Background(), scope)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !deleteItemsCalled {
		t.Error("expected repo.DeleteAllByFamily to be called")
	}
	if !deleteTemplatesCalled {
		t.Error("expected templateRepo.DeleteAllByFamily to be called")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area I: Get Single Schedule Item [17-planning §4, §17]
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetScheduleItem_I32_ReturnsItemWithStudentName(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	itemID := uuid.Must(uuid.NewV7())
	studentID := uuid.Must(uuid.NewV7())
	now := time.Now()

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (*ScheduleItem, error) {
			if id != itemID {
				return nil, nil
			}
			return &ScheduleItem{
				ID: itemID, Title: "Math Lesson", StudentID: &studentID,
				StartDate: now, Category: ScheduleCategoryLesson, CreatedAt: now,
			}, nil
		},
	}
	iamSvc := &stubIamService{
		getStudentNameFn: func(_ context.Context, id uuid.UUID) (string, error) {
			if id == studentID {
				return "Alice", nil
			}
			return "", nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: repo, templateRepo: &stubScheduleTemplateRepo{},
		iamSvc: iamSvc, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	result, err := svc.GetScheduleItem(context.Background(), auth, scope, itemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Title != "Math Lesson" {
		t.Errorf("expected title 'Math Lesson', got %q", result.Title)
	}
	if result.StudentName == nil || *result.StudentName != "Alice" {
		t.Errorf("expected student name 'Alice', got %v", result.StudentName)
	}
}

func TestGetScheduleItem_I33_ReturnsErrItemNotFound(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	repo := &stubScheduleItemRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleItem, error) {
			return nil, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	_, err := svc.GetScheduleItem(context.Background(), auth, scope, uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrItemNotFound) {
		t.Errorf("expected ErrItemNotFound, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area J: Schedule Templates [17-planning §6, §11.3, §17]
// ═══════════════════════════════════════════════════════════════════════════════

func TestCreateTemplate_J34_ValidInputReturnsNewUUID(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	var createdTmpl *ScheduleTemplate
	tmplRepo := &stubScheduleTemplateRepo{
		createFn: func(_ context.Context, _ *shared.FamilyScope, tmpl *ScheduleTemplate) error {
			createdTmpl = tmpl
			return nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: &stubScheduleItemRepo{}, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	id, err := svc.CreateTemplate(context.Background(), auth, scope, CreateTemplateInput{
		Name: "Weekly Math",
		Items: []TemplateItem{
			{DayOfWeek: "monday", Title: "Math", Category: ScheduleCategoryLesson},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id == uuid.Nil {
		t.Error("expected non-nil UUID")
	}
	if createdTmpl == nil {
		t.Fatal("expected templateRepo.Create to be called")
	}
	if createdTmpl.Name != "Weekly Math" {
		t.Errorf("expected name 'Weekly Math', got %q", createdTmpl.Name)
	}
}

func TestListTemplates_J35_ReturnsAllTemplatesForFamily(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	now := time.Now()

	tmplRepo := &stubScheduleTemplateRepo{
		listByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) ([]ScheduleTemplate, error) {
			return []ScheduleTemplate{
				{ID: uuid.Must(uuid.NewV7()), Name: "Template 1", Items: []byte(`[]`), CreatedAt: now, UpdatedAt: now},
				{ID: uuid.Must(uuid.NewV7()), Name: "Template 2", Items: []byte(`[]`), CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: &stubScheduleItemRepo{}, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	result, err := svc.ListTemplates(context.Background(), auth, scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("expected 2 templates, got %d", len(result))
	}
}

func TestUpdateTemplate_J36_UpdatesExistingTemplate(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	tmplID := uuid.Must(uuid.NewV7())

	var updateCalled bool
	tmplRepo := &stubScheduleTemplateRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (*ScheduleTemplate, error) {
			if id == tmplID {
				return &ScheduleTemplate{ID: tmplID, Name: "Old Name", Items: []byte(`[]`)}, nil
			}
			return nil, nil
		},
		updateFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ *UpdateTemplateInput, _ []byte) error {
			updateCalled = true
			return nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: &stubScheduleItemRepo{}, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	newName := "New Name"
	err := svc.UpdateTemplate(context.Background(), auth, scope, tmplID, UpdateTemplateInput{Name: &newName})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !updateCalled {
		t.Error("expected templateRepo.Update to be called")
	}
}

func TestUpdateTemplate_J37_ReturnsErrTemplateNotFound(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	tmplRepo := &stubScheduleTemplateRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleTemplate, error) {
			return nil, nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: &stubScheduleItemRepo{}, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	err := svc.UpdateTemplate(context.Background(), auth, scope, uuid.Must(uuid.NewV7()), UpdateTemplateInput{})
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestDeleteTemplate_J38_DeletesExistingTemplate(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	tmplID := uuid.Must(uuid.NewV7())

	var deleteCalled bool
	tmplRepo := &stubScheduleTemplateRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (*ScheduleTemplate, error) {
			if id == tmplID {
				return &ScheduleTemplate{ID: tmplID, Items: []byte(`[]`)}, nil
			}
			return nil, nil
		},
		deleteFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
			deleteCalled = true
			return nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: &stubScheduleItemRepo{}, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	err := svc.DeleteTemplate(context.Background(), auth, scope, tmplID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected templateRepo.Delete to be called")
	}
}

func TestDeleteTemplate_J39_ReturnsErrTemplateNotFound(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	tmplRepo := &stubScheduleTemplateRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleTemplate, error) {
			return nil, nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: &stubScheduleItemRepo{}, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	err := svc.DeleteTemplate(context.Background(), auth, scope, uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestApplyTemplate_J40_CreatesScheduleItemsFromTemplate(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)
	tmplID := uuid.Must(uuid.NewV7())

	var createdCount int
	repo := &stubScheduleItemRepo{
		createFn: func(_ context.Context, _ *shared.FamilyScope, _ *ScheduleItem) error {
			createdCount++
			return nil
		},
	}
	tmplRepo := &stubScheduleTemplateRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (*ScheduleTemplate, error) {
			if id == tmplID {
				return &ScheduleTemplate{
					ID:   tmplID,
					Name: "Weekly Plan",
					Items: []byte(`[
						{"day_of_week": "monday", "title": "Math", "category": "lesson"},
						{"day_of_week": "wednesday", "title": "Science", "category": "lesson"},
						{"day_of_week": "friday", "title": "Reading", "category": "reading"}
					]`),
				}, nil
			}
			return nil, nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: repo, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	// Apply for a full week: Mon Mar 23 2026 to Sun Mar 29 2026
	start := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC) // Monday
	end := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)   // Next Monday (exclusive)

	ids, err := svc.ApplyTemplate(context.Background(), auth, scope, tmplID, ApplyTemplateInput{
		StartDate: start,
		EndDate:   end,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Mon + Wed + Fri = 3 items
	if len(ids) != 3 {
		t.Errorf("expected 3 created items, got %d", len(ids))
	}
	if createdCount != 3 {
		t.Errorf("expected 3 repo.Create calls, got %d", createdCount)
	}
}

func TestApplyTemplate_J41_ReturnsErrTemplateNotFound(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	tmplRepo := &stubScheduleTemplateRepo{
		findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*ScheduleTemplate, error) {
			return nil, nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: &stubScheduleItemRepo{}, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	start := time.Date(2026, 3, 23, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 30, 0, 0, 0, 0, time.UTC)

	_, err := svc.ApplyTemplate(context.Background(), auth, scope, uuid.Must(uuid.NewV7()), ApplyTemplateInput{
		StartDate: start, EndDate: end,
	})
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Errorf("expected ErrTemplateNotFound, got %v", err)
	}
}

func TestApplyTemplate_J42_ValidatesDateRange(t *testing.T) {
	auth := testAuth()
	scope := testScopeFromAuth(auth)

	svc := newTestService(&stubScheduleItemRepo{}, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	now := time.Now()
	_, err := svc.ApplyTemplate(context.Background(), auth, scope, uuid.Must(uuid.NewV7()), ApplyTemplateInput{
		StartDate: now,
		EndDate:   now.Add(-time.Hour), // end before start
	})
	if !errors.Is(err, ErrInvalidDateRange) {
		t.Errorf("expected ErrInvalidDateRange, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area K: Event Handlers [17-planning §16]
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleEventCancelled_K43_MarksLinkedItemsCancelled(t *testing.T) {
	eventID := uuid.Must(uuid.NewV7())
	familyID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())

	var deletedIDs []uuid.UUID
	repo := &stubScheduleItemRepo{
		findByLinkedEventIDFn: func(_ context.Context, eid uuid.UUID) ([]ScheduleItem, error) {
			if eid == eventID {
				return []ScheduleItem{
					{ID: itemID, FamilyID: familyID, LinkedEventID: &eventID},
				}, nil
			}
			return nil, nil
		},
		deleteFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) error {
			deletedIDs = append(deletedIDs, id)
			return nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.HandleEventCancelled(context.Background(), eventID, []uuid.UUID{familyID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deletedIDs) != 1 || deletedIDs[0] != itemID {
		t.Errorf("expected item %v to be deleted, got %v", itemID, deletedIDs)
	}
}

func TestHandleEventCancelled_K44_NoOpIfNoLinkedItems(t *testing.T) {
	repo := &stubScheduleItemRepo{
		findByLinkedEventIDFn: func(_ context.Context, _ uuid.UUID) ([]ScheduleItem, error) {
			return []ScheduleItem{}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.HandleEventCancelled(context.Background(), uuid.Must(uuid.NewV7()), nil)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestHandleActivityLogged_K45_MarksMatchingItemCompleted(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	studentID := uuid.Must(uuid.NewV7())
	activityID := uuid.Must(uuid.NewV7())
	itemID := uuid.Must(uuid.NewV7())

	var markCompletedCalled, linkActivityCalled bool
	repo := &stubScheduleItemRepo{
		findByStudentAndDateFn: func(_ context.Context, _ *shared.FamilyScope, sid uuid.UUID, _ time.Time) ([]ScheduleItem, error) {
			if sid == studentID {
				return []ScheduleItem{
					{ID: itemID, FamilyID: familyID, StudentID: &studentID, IsCompleted: false},
				}, nil
			}
			return nil, nil
		},
		markCompletedFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID, _ time.Time) error {
			if id == itemID {
				markCompletedCalled = true
			}
			return nil
		},
		setLinkedActivityFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID, aid uuid.UUID) error {
			if id == itemID && aid == activityID {
				linkActivityCalled = true
			}
			return nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.HandleActivityLogged(context.Background(), familyID, studentID, activityID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !markCompletedCalled {
		t.Error("expected repo.MarkCompleted to be called")
	}
	if !linkActivityCalled {
		t.Error("expected repo.SetLinkedActivity to be called")
	}
}

func TestHandleActivityLogged_K46_NoOpIfNoMatchingItems(t *testing.T) {
	repo := &stubScheduleItemRepo{
		findByStudentAndDateFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ time.Time) ([]ScheduleItem, error) {
			return []ScheduleItem{}, nil
		},
	}
	svc := newTestService(repo, &stubIamService{}, &stubLearningService{}, &stubComplianceService{}, &stubSocialService{})

	err := svc.HandleActivityLogged(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area L: Export Data includes templates [17-planning §14]
// ═══════════════════════════════════════════════════════════════════════════════

func TestExportData_L47_IncludesTemplates(t *testing.T) {
	scope := testScope()
	now := time.Now()

	repo := &stubScheduleItemRepo{
		listAllByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) ([]ScheduleItem, error) {
			return []ScheduleItem{}, nil
		},
	}
	tmplRepo := &stubScheduleTemplateRepo{
		listByFamilyFn: func(_ context.Context, _ *shared.FamilyScope) ([]ScheduleTemplate, error) {
			return []ScheduleTemplate{
				{ID: uuid.Must(uuid.NewV7()), Name: "My Template", Items: []byte(`[]`), CreatedAt: now, UpdatedAt: now},
			}, nil
		},
	}
	svc := newTestServiceFull(testDeps{
		repo: repo, templateRepo: tmplRepo,
		iamSvc: &stubIamService{}, learnSvc: &stubLearningService{},
		complySvc: &stubComplianceService{}, socialSvc: &stubSocialService{},
	})

	data, err := svc.ExportData(context.Background(), scope)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !json.Valid(data) {
		t.Error("expected valid JSON")
	}
	if !strings.Contains(string(data), "My Template") {
		t.Error("expected export data to contain template name")
	}
	if !strings.Contains(string(data), "templates") {
		t.Error("expected export data to contain templates key")
	}
}
