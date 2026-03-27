package recs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Req 1–3: ExplorationFrequency.ExplorationRatio [§8]
// ═══════════════════════════════════════════════════════════════════════════════

func TestExplorationRatio_Off(t *testing.T) {
	if got := ExplorationOff.ExplorationRatio(); got != 0.0 {
		t.Fatalf("ExplorationOff.ExplorationRatio = %v, want 0.0", got)
	}
}

func TestExplorationRatio_Occasional(t *testing.T) {
	if got := ExplorationOccasional.ExplorationRatio(); got != 0.10 {
		t.Fatalf("ExplorationOccasional.ExplorationRatio = %v, want 0.10", got)
	}
}

func TestExplorationRatio_Frequent(t *testing.T) {
	if got := ExplorationFrequent.ExplorationRatio(); got != 0.25 {
		t.Fatalf("ExplorationFrequent.ExplorationRatio = %v, want 0.25", got)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 4: CoarsenAgeBand [§14.1]
// ═══════════════════════════════════════════════════════════════════════════════

func TestCoarsenAgeBand(t *testing.T) {
	cases := []struct {
		age  int
		want string
	}{
		{0, ""},   // too young
		{3, ""},   // too young
		{4, "4-6"},
		{6, "4-6"},
		{7, "7-9"},
		{9, "7-9"},
		{10, "10-12"},
		{12, "10-12"},
		{13, "13-15"},
		{15, "13-15"},
		{16, "16-18"},
		{18, "16-18"},
		{19, ""},  // too old
		{25, ""},  // too old
	}
	for _, tc := range cases {
		got := CoarsenAgeBand(tc.age)
		if got != tc.want {
			t.Errorf("CoarsenAgeBand(%d) = %q, want %q", tc.age, got, tc.want)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 5: RoundDurationToNearest5 [§14.1]
// ═══════════════════════════════════════════════════════════════════════════════

func TestRoundDurationToNearest5(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 0},
		{1, 0},
		{2, 0},
		{3, 5},
		{4, 5},
		{5, 5},
		{6, 5},
		{7, 5},
		{8, 10},
		{9, 10},
		{10, 10},
		{12, 10},
		{13, 15},
		{27, 25},
		{28, 30},
	}
	for _, tc := range cases {
		got := RoundDurationToNearest5(tc.in)
		if got != tc.want {
			t.Errorf("RoundDurationToNearest5(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 6: SeasonForMonth [§10.5]
// ═══════════════════════════════════════════════════════════════════════════════

func TestSeasonForMonth(t *testing.T) {
	cases := []struct {
		month time.Month
		want  Season
	}{
		{time.March, SeasonSpring},
		{time.April, SeasonSpring},
		{time.May, SeasonSpring},
		{time.June, SeasonSummer},
		{time.July, SeasonSummer},
		{time.August, SeasonSummer},
		{time.September, SeasonAutumn},
		{time.October, SeasonAutumn},
		{time.November, SeasonAutumn},
		{time.December, SeasonWinter},
		{time.January, SeasonWinter},
		{time.February, SeasonWinter},
	}
	for _, tc := range cases {
		got := SeasonForMonth(tc.month)
		if got != tc.want {
			t.Errorf("SeasonForMonth(%s) = %q, want %q", tc.month, got, tc.want)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 7–8: RecordSignal [§9]
// ═══════════════════════════════════════════════════════════════════════════════

func TestRecordSignal_DelegatesToRepo(t *testing.T) {
	var capturedSignal NewSignal
	svc := newTestService(
		&stubSignalRepo{createFn: func(_ context.Context, s NewSignal) error {
			capturedSignal = s
			return nil
		}},
		nil, nil,
		&stubPopularityRepo{},
		nil, nil, &stubIamService{},
	)

	studentID := uuid.Must(uuid.NewV7())
	cmd := RecordSignalCommand{
		FamilyID:        shared.NewFamilyID(uuid.Must(uuid.NewV7())),
		StudentID:       &studentID,
		SignalType:      SignalActivityLogged,
		MethodologySlug: "charlotte-mason",
		Payload:         map[string]any{"subject_tags": []string{"math"}},
		SignalDate:      time.Now(),
	}

	if err := svc.RecordSignal(context.Background(), cmd); err != nil {
		t.Fatalf("RecordSignal error: %v", err)
	}
	if capturedSignal.SignalType != SignalActivityLogged {
		t.Errorf("captured signal_type = %q, want %q", capturedSignal.SignalType, SignalActivityLogged)
	}
	if capturedSignal.MethodologySlug != "charlotte-mason" {
		t.Errorf("captured methodology_slug = %q, want %q", capturedSignal.MethodologySlug, "charlotte-mason")
	}
}

func TestRecordSignal_ReturnsErrSignalRecordingFailed_OnRepoError(t *testing.T) {
	repoErr := errors.New("db down")
	svc := newTestService(
		&stubSignalRepo{createFn: func(_ context.Context, _ NewSignal) error {
			return repoErr
		}},
		nil, nil,
		&stubPopularityRepo{},
		nil, nil, &stubIamService{},
	)

	err := svc.RecordSignal(context.Background(), RecordSignalCommand{
		FamilyID:   shared.NewFamilyID(uuid.Must(uuid.NewV7())),
		SignalType: SignalBookCompleted,
		SignalDate: time.Now(),
	})
	if !errors.Is(err, ErrSignalRecordingFailed) {
		t.Fatalf("want ErrSignalRecordingFailed, got %v", err)
	}
}

func TestRecordSignal_ResolvesMethodologySlug(t *testing.T) {
	var capturedSignal NewSignal
	svc := newTestService(
		&stubSignalRepo{createFn: func(_ context.Context, s NewSignal) error {
			capturedSignal = s
			return nil
		}},
		nil, nil,
		&stubPopularityRepo{},
		nil, nil,
		&stubIamService{
			getFamilyMethodologySlugFn: func(_ context.Context, _ shared.FamilyID) (string, error) {
				return "classical", nil
			},
		},
	)

	cmd := RecordSignalCommand{
		FamilyID:        shared.NewFamilyID(uuid.Must(uuid.NewV7())),
		SignalType:      SignalActivityLogged,
		MethodologySlug: "", // empty — should be resolved
		SignalDate:      time.Now(),
	}

	if err := svc.RecordSignal(context.Background(), cmd); err != nil {
		t.Fatalf("RecordSignal error: %v", err)
	}
	if capturedSignal.MethodologySlug != "classical" {
		t.Errorf("methodology_slug = %q, want %q", capturedSignal.MethodologySlug, "classical")
	}
}

func TestRecordSignal_ContinuesOnResolutionError(t *testing.T) {
	var capturedSignal NewSignal
	svc := newTestService(
		&stubSignalRepo{createFn: func(_ context.Context, s NewSignal) error {
			capturedSignal = s
			return nil
		}},
		nil, nil,
		&stubPopularityRepo{},
		nil, nil,
		&stubIamService{
			getFamilyMethodologySlugFn: func(_ context.Context, _ shared.FamilyID) (string, error) {
				return "", errors.New("iam down")
			},
		},
	)

	cmd := RecordSignalCommand{
		FamilyID:        shared.NewFamilyID(uuid.Must(uuid.NewV7())),
		SignalType:      SignalActivityLogged,
		MethodologySlug: "", // empty — resolution will fail
		SignalDate:      time.Now(),
	}

	if err := svc.RecordSignal(context.Background(), cmd); err != nil {
		t.Fatalf("RecordSignal should not fail on resolution error, got: %v", err)
	}
	if capturedSignal.MethodologySlug != "" {
		t.Errorf("methodology_slug = %q, want empty string on resolution failure", capturedSignal.MethodologySlug)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 9: RegisterListing [§9.4]
// ═══════════════════════════════════════════════════════════════════════════════

func TestRegisterListing_DelegatesToPopularityRepo(t *testing.T) {
	var upsertCalled bool
	svc := newTestService(
		nil, nil, nil,
		&stubPopularityRepo{upsertFn: func(_ context.Context, _ NewPopularityScore) error {
			upsertCalled = true
			return nil
		}},
		nil, nil, &stubIamService{},
	)

	cmd := RegisterListingCommand{
		ListingID:   uuid.Must(uuid.NewV7()),
		PublisherID: uuid.Must(uuid.NewV7()),
		ContentType: "curriculum",
		SubjectTags: []string{"math"},
	}
	if err := svc.RegisterListing(context.Background(), cmd); err != nil {
		t.Fatalf("RegisterListing error: %v", err)
	}
	if !upsertCalled {
		t.Error("expected PopularityRepository.Upsert to be called")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 10: HandleFamilyDeletion [§12]
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandleFamilyDeletion_CallsAllDeleteMethods(t *testing.T) {
	var signalDeleted, recDeleted, feedbackDeleted, prefDeleted bool
	familyID := shared.NewFamilyID(uuid.Must(uuid.NewV7()))

	svc := newTestService(
		&stubSignalRepo{deleteByFamilyFn: func(_ context.Context, fid shared.FamilyID) (int64, error) {
			signalDeleted = true
			return 1, nil
		}},
		&stubRecommendationRepo{deleteByFamilyFn: func(_ context.Context, fid shared.FamilyID) (int64, error) {
			recDeleted = true
			return 2, nil
		}},
		&stubFeedbackRepo{deleteByFamilyFn: func(_ context.Context, fid shared.FamilyID) (int64, error) {
			feedbackDeleted = true
			return 0, nil
		}},
		&stubPopularityRepo{},
		&stubPreferenceRepo{deleteByFamilyFn: func(_ context.Context, fid shared.FamilyID) (int64, error) {
			prefDeleted = true
			return 1, nil
		}},
		nil, &stubIamService{},
	)

	if err := svc.HandleFamilyDeletion(context.Background(), familyID); err != nil {
		t.Fatalf("HandleFamilyDeletion error: %v", err)
	}
	if !signalDeleted {
		t.Error("expected SignalRepository.DeleteByFamily to be called")
	}
	if !recDeleted {
		t.Error("expected RecommendationRepository.DeleteByFamily to be called")
	}
	if !feedbackDeleted {
		t.Error("expected FeedbackRepository.DeleteByFamily to be called")
	}
	if !prefDeleted {
		t.Error("expected PreferenceRepository.DeleteByFamily to be called")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 11: InvalidateMethodologyCache [§12]
// ═══════════════════════════════════════════════════════════════════════════════

func TestInvalidateMethodologyCache_NoOp(t *testing.T) {
	svc := newTestService(nil, nil, nil, &stubPopularityRepo{}, nil, &stubAnonRepo{}, &stubIamService{})
	if err := svc.InvalidateMethodologyCache(context.Background()); err != nil {
		t.Fatalf("InvalidateMethodologyCache returned error: %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 12–15: Preferences [§5]
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetPreferences_DelegatesToFindOrDefault(t *testing.T) {
	scope := testScope()
	expected := &Preferences{
		EnabledTypes:         []string{"marketplace_content"},
		ExplorationFrequency: "occasional",
	}
	svc := newTestService(
		nil, nil, nil, &stubPopularityRepo{},
		&stubPreferenceRepo{findOrDefaultFn: func(_ context.Context, _ *shared.FamilyScope) (*Preferences, error) {
			return expected, nil
		}},
		nil, &stubIamService{},
	)

	resp, err := svc.GetPreferences(context.Background(), scope)
	if err != nil {
		t.Fatalf("GetPreferences error: %v", err)
	}
	if resp.ExplorationFrequency != "occasional" {
		t.Errorf("exploration_frequency = %q, want %q", resp.ExplorationFrequency, "occasional")
	}
}

func TestUpdatePreferences_ReturnsErrInvalidExplorationFrequency(t *testing.T) {
	scope := testScope()
	svc := newTestService(nil, nil, nil, &stubPopularityRepo{}, &stubPreferenceRepo{}, nil, &stubIamService{})

	bad := "weekly"
	_, err := svc.UpdatePreferences(context.Background(), scope, UpdatePreferencesCommand{
		ExplorationFrequency: &bad,
	})
	if !errors.Is(err, ErrInvalidExplorationFrequency) {
		t.Fatalf("want ErrInvalidExplorationFrequency, got %v", err)
	}
}

func TestUpdatePreferences_ReturnsErrInvalidRecommendationType(t *testing.T) {
	scope := testScope()
	svc := newTestService(nil, nil, nil, &stubPopularityRepo{}, &stubPreferenceRepo{}, nil, &stubIamService{})

	_, err := svc.UpdatePreferences(context.Background(), scope, UpdatePreferencesCommand{
		EnabledTypes: []string{"invalid_type"},
	})
	if !errors.Is(err, ErrInvalidRecommendationType) {
		t.Fatalf("want ErrInvalidRecommendationType, got %v", err)
	}
}

func TestUpdatePreferences_DelegatesToUpsertOnValidInput(t *testing.T) {
	scope := testScope()
	var upsertCalled bool
	current := &Preferences{
		EnabledTypes:         []string{"marketplace_content"},
		ExplorationFrequency: "occasional",
	}
	updated := &Preferences{
		EnabledTypes:         []string{"activity_idea"},
		ExplorationFrequency: "frequent",
	}
	freq := "frequent"
	svc := newTestService(
		nil, nil, nil, &stubPopularityRepo{},
		&stubPreferenceRepo{
			findOrDefaultFn: func(_ context.Context, _ *shared.FamilyScope) (*Preferences, error) {
				return current, nil
			},
			upsertFn: func(_ context.Context, _ *shared.FamilyScope, _ UpdatePreferences) (*Preferences, error) {
				upsertCalled = true
				return updated, nil
			},
		},
		nil, &stubIamService{},
	)

	resp, err := svc.UpdatePreferences(context.Background(), scope, UpdatePreferencesCommand{
		EnabledTypes:         []string{"activity_idea"},
		ExplorationFrequency: &freq,
	})
	if err != nil {
		t.Fatalf("UpdatePreferences error: %v", err)
	}
	if !upsertCalled {
		t.Error("expected PreferenceRepository.Upsert to be called")
	}
	if resp.ExplorationFrequency != "frequent" {
		t.Errorf("exploration_frequency = %q, want frequent", resp.ExplorationFrequency)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 16–17: Recommendation Queries [§5]
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetRecommendations_DelegatesToFindActiveByFamily(t *testing.T) {
	scope := testScope()
	expected := []Recommendation{
		{
			ID:                 uuid.Must(uuid.NewV7()),
			RecommendationType: RecommendationMarketplaceContent,
			TargetEntityID:     uuid.Must(uuid.NewV7()),
			TargetEntityLabel:  "Test Resource",
			SourceSignal:       SourceMethodologyMatch,
			SourceLabel:        "Matches your methodology",
			Score:              0.8,
			Status:             "active",
			ExpiresAt:          time.Now().Add(14 * 24 * time.Hour),
			CreatedAt:          time.Now(),
		},
	}

	svc := newTestService(
		nil,
		&stubRecommendationRepo{findActiveByFamilyFn: func(_ context.Context, _ *shared.FamilyScope, _ *string, _ *string, _ int64) ([]Recommendation, *string, error) {
			return expected, nil, nil
		}},
		nil, &stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	resp, err := svc.GetRecommendations(context.Background(), scope, RecommendationListParams{})
	if err != nil {
		t.Fatalf("GetRecommendations error: %v", err)
	}
	if len(resp.Recommendations) != 1 {
		t.Fatalf("expected 1 recommendation, got %d", len(resp.Recommendations))
	}
	if !resp.Recommendations[0].IsSuggestion {
		t.Error("IsSuggestion should always be true")
	}
}

func TestGetStudentRecommendations_ReturnsErrStudentNotFound(t *testing.T) {
	scope := testScope()
	studentID := uuid.Must(uuid.NewV7())

	svc := newTestService(
		nil, nil, nil, &stubPopularityRepo{}, nil, nil,
		&stubIamService{studentBelongsToFamilyFn: func(_ context.Context, _ uuid.UUID, _ shared.FamilyID) (bool, error) {
			return false, nil
		}},
	)

	_, err := svc.GetStudentRecommendations(context.Background(), scope, StudentRecommendationParams{
		StudentID: studentID,
	})
	if !errors.Is(err, ErrStudentNotFound) {
		t.Fatalf("want ErrStudentNotFound, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 18–20: DismissRecommendation [§13.2]
// ═══════════════════════════════════════════════════════════════════════════════

func TestDismissRecommendation_SetsStatusAndCreatesFeedback(t *testing.T) {
	scope := testScope()
	recID := uuid.Must(uuid.NewV7())
	var statusUpdated, feedbackCreated bool

	svc := newTestService(
		nil,
		&stubRecommendationRepo{updateStatusFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID, status string) error {
			if id == recID && status == "dismissed" {
				statusUpdated = true
			}
			return nil
		}},
		&stubFeedbackRepo{
			findByRecommendationFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Feedback, error) {
				return nil, ErrFeedbackNotFound
			},
			createFn: func(_ context.Context, f NewFeedback) error {
				if f.Action == "dismiss" {
					feedbackCreated = true
				}
				return nil
			},
		},
		&stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	if err := svc.DismissRecommendation(context.Background(), scope, recID); err != nil {
		t.Fatalf("DismissRecommendation error: %v", err)
	}
	if !statusUpdated {
		t.Error("expected recommendation status to be updated to 'dismissed'")
	}
	if !feedbackCreated {
		t.Error("expected feedback record to be created")
	}
}

func TestDismissRecommendation_ReturnsErrAlreadyHasFeedback(t *testing.T) {
	scope := testScope()
	recID := uuid.Must(uuid.NewV7())

	svc := newTestService(
		nil, nil,
		&stubFeedbackRepo{
			findByRecommendationFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Feedback, error) {
				return &Feedback{Action: "dismiss"}, nil
			},
		},
		&stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	err := svc.DismissRecommendation(context.Background(), scope, recID)
	if !errors.Is(err, ErrAlreadyHasFeedback) {
		t.Fatalf("want ErrAlreadyHasFeedback, got %v", err)
	}
}

func TestDismissRecommendation_ReturnsErrRecommendationNotFound_OnStatusError(t *testing.T) {
	scope := testScope()
	recID := uuid.Must(uuid.NewV7())

	svc := newTestService(
		nil,
		&stubRecommendationRepo{updateStatusFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ string) error {
			return ErrRecommendationNotFound
		}},
		&stubFeedbackRepo{
			findByRecommendationFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Feedback, error) {
				return nil, ErrFeedbackNotFound
			},
		},
		&stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	err := svc.DismissRecommendation(context.Background(), scope, recID)
	if !errors.Is(err, ErrRecommendationNotFound) {
		t.Fatalf("want ErrRecommendationNotFound, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 21: BlockRecommendation [§13.2]
// ═══════════════════════════════════════════════════════════════════════════════

func TestBlockRecommendation_ReturnsBlockedEntityID(t *testing.T) {
	scope := testScope()
	recID := uuid.Must(uuid.NewV7())
	entityID := uuid.Must(uuid.NewV7())
	var capturedFeedback NewFeedback

	svc := newTestService(
		nil,
		&stubRecommendationRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, id uuid.UUID) (*Recommendation, error) {
				return &Recommendation{
					ID:             recID,
					TargetEntityID: entityID,
					Status:         "active",
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ string) error {
				return nil
			},
		},
		&stubFeedbackRepo{
			findByRecommendationFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Feedback, error) {
				return nil, ErrFeedbackNotFound
			},
			createFn: func(_ context.Context, f NewFeedback) error {
				capturedFeedback = f
				return nil
			},
		},
		&stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	blockedID, err := svc.BlockRecommendation(context.Background(), scope, recID)
	if err != nil {
		t.Fatalf("BlockRecommendation error: %v", err)
	}
	if blockedID != entityID {
		t.Errorf("returned blocked_entity_id = %v, want %v", blockedID, entityID)
	}
	if capturedFeedback.BlockedEntityID == nil {
		t.Error("expected BlockedEntityID to be non-nil for block feedback")
	}
	if *capturedFeedback.BlockedEntityID != entityID {
		t.Errorf("BlockedEntityID = %v, want %v", *capturedFeedback.BlockedEntityID, entityID)
	}
	if capturedFeedback.Action != "block" {
		t.Errorf("Action = %q, want 'block'", capturedFeedback.Action)
	}
}

func TestBlockRecommendation_ReturnsErrRecommendationNotFound(t *testing.T) {
	scope := testScope()
	recID := uuid.Must(uuid.NewV7())

	svc := newTestService(
		nil,
		&stubRecommendationRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Recommendation, error) {
				return nil, ErrRecommendationNotFound
			},
		},
		nil, &stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	_, err := svc.BlockRecommendation(context.Background(), scope, recID)
	if !errors.Is(err, ErrRecommendationNotFound) {
		t.Fatalf("want ErrRecommendationNotFound, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Req 22–23: UndoFeedback [§13.2]
// ═══════════════════════════════════════════════════════════════════════════════

func TestUndoFeedback_DeletesFeedbackAndRestoresStatus(t *testing.T) {
	scope := testScope()
	recID := uuid.Must(uuid.NewV7())
	var deleteCalled, statusRestored bool

	svc := newTestService(
		nil,
		&stubRecommendationRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Recommendation, error) {
				return &Recommendation{
					ID:        recID,
					Status:    "dismissed",
					ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // not expired
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, status string) error {
				if status == "active" {
					statusRestored = true
				}
				return nil
			},
		},
		&stubFeedbackRepo{
			findByRecommendationFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Feedback, error) {
				return &Feedback{Action: "dismiss"}, nil
			},
			deleteFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
				deleteCalled = true
				return nil
			},
		},
		&stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	if err := svc.UndoFeedback(context.Background(), scope, recID); err != nil {
		t.Fatalf("UndoFeedback error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected FeedbackRepository.Delete to be called")
	}
	if !statusRestored {
		t.Error("expected recommendation status to be restored to 'active'")
	}
}

func TestUndoFeedback_SkipsRestoreForExpiredRecommendation(t *testing.T) {
	scope := testScope()
	recID := uuid.Must(uuid.NewV7())
	var deleteCalled bool
	var statusUpdated bool

	svc := newTestService(
		nil,
		&stubRecommendationRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Recommendation, error) {
				return &Recommendation{
					ID:        recID,
					Status:    "dismissed",
					ExpiresAt: time.Now().Add(-1 * time.Hour), // already expired
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ string) error {
				statusUpdated = true
				return nil
			},
		},
		&stubFeedbackRepo{
			findByRecommendationFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Feedback, error) {
				return &Feedback{Action: "dismiss"}, nil
			},
			deleteFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
				deleteCalled = true
				return nil
			},
		},
		&stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	if err := svc.UndoFeedback(context.Background(), scope, recID); err != nil {
		t.Fatalf("UndoFeedback error: %v", err)
	}
	if !deleteCalled {
		t.Error("expected feedback to be deleted even for expired recommendation")
	}
	if statusUpdated {
		t.Error("expected status NOT to be restored for expired recommendation")
	}
}

func TestUndoFeedback_RestoresStatusForNonExpiredRecommendation(t *testing.T) {
	scope := testScope()
	recID := uuid.Must(uuid.NewV7())
	var statusRestored bool

	svc := newTestService(
		nil,
		&stubRecommendationRepo{
			findByIDFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Recommendation, error) {
				return &Recommendation{
					ID:        recID,
					Status:    "dismissed",
					ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // still valid
				}, nil
			},
			updateStatusFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, status string) error {
				if status == "active" {
					statusRestored = true
				}
				return nil
			},
		},
		&stubFeedbackRepo{
			findByRecommendationFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Feedback, error) {
				return &Feedback{Action: "dismiss"}, nil
			},
			deleteFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
				return nil
			},
		},
		&stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	if err := svc.UndoFeedback(context.Background(), scope, recID); err != nil {
		t.Fatalf("UndoFeedback error: %v", err)
	}
	if !statusRestored {
		t.Error("expected status to be restored to 'active' for non-expired recommendation")
	}
}

func TestUndoFeedback_ReturnsErrFeedbackNotFound(t *testing.T) {
	scope := testScope()
	recID := uuid.Must(uuid.NewV7())

	svc := newTestService(
		nil, nil,
		&stubFeedbackRepo{
			findByRecommendationFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*Feedback, error) {
				return nil, ErrFeedbackNotFound
			},
		},
		&stubPopularityRepo{}, nil, nil, &stubIamService{},
	)

	err := svc.UndoFeedback(context.Background(), scope, recID)
	if !errors.Is(err, ErrFeedbackNotFound) {
		t.Fatalf("want ErrFeedbackNotFound, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Algorithm Pure Function Tests
// ═══════════════════════════════════════════════════════════════════════════════

func TestComputeHMAC_Deterministic(t *testing.T) {
	id := uuid.Must(uuid.NewV7())
	key := []byte("test-secret-key")
	h1 := computeHMAC(id, key)
	h2 := computeHMAC(id, key)
	if h1 != h2 {
		t.Errorf("HMAC not deterministic: %q != %q", h1, h2)
	}
	if len(h1) != 64 { // SHA-256 hex = 64 chars
		t.Errorf("HMAC length = %d, want 64", len(h1))
	}
}

func TestComputeHMAC_DifferentFamilies(t *testing.T) {
	key := []byte("test-secret-key")
	id1 := uuid.Must(uuid.NewV7())
	id2 := uuid.Must(uuid.NewV7())
	h1 := computeHMAC(id1, key)
	h2 := computeHMAC(id2, key)
	if h1 == h2 {
		t.Error("different family IDs should produce different HMACs")
	}
}

func TestComputeJaccardSimilarity(t *testing.T) {
	cases := []struct {
		name string
		a, b []string
		want float32
	}{
		{"empty_a", nil, []string{"x"}, 0.0},
		{"empty_b", []string{"x"}, nil, 0.0},
		{"both_empty", nil, nil, 0.0},
		{"disjoint", []string{"a", "b"}, []string{"c", "d"}, 0.0},
		{"identical", []string{"a", "b"}, []string{"a", "b"}, 1.0},
		{"partial_overlap", []string{"a", "b", "c"}, []string{"b", "c", "d"}, 0.5}, // 2/4
	}
	for _, tc := range cases {
		got := computeJaccardSimilarity(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("computeJaccardSimilarity(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestComputeFreshness(t *testing.T) {
	now := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	// Brand new = 1.0
	fresh := computeFreshness(now, now)
	if fresh < 0.99 || fresh > 1.01 {
		t.Errorf("freshness for today = %v, want ~1.0", fresh)
	}

	// ~23 days ago (half-life) = ~0.5
	halfLife := computeFreshness(now.AddDate(0, 0, -23), now)
	if halfLife < 0.45 || halfLife > 0.55 {
		t.Errorf("freshness for 23 days ago = %v, want ~0.5", halfLife)
	}

	// 90 days ago = very small
	old := computeFreshness(now.AddDate(0, 0, -90), now)
	if old > 0.1 {
		t.Errorf("freshness for 90 days ago = %v, want < 0.1", old)
	}
}

func TestComputeScore_Weights(t *testing.T) {
	// Perfect score across all factors.
	perfect := ComputeScore(ScoringFactors{
		MethodologyMatch: 1.0,
		Popularity:       1.0,
		Relevance:        1.0,
		Freshness:        1.0,
		Exploration:      1.0,
	})
	// Weights sum to 1.0, so perfect score should be 1.0.
	if perfect < 0.99 || perfect > 1.01 {
		t.Errorf("perfect score = %v, want ~1.0", perfect)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Seasonal & Dominant Signal Tests [§10.5, §13.1]
// ═══════════════════════════════════════════════════════════════════════════════

func TestHasSeasonalOverlap(t *testing.T) {
	cases := []struct {
		name   string
		tags   []string
		season Season
		want   bool
	}{
		{"spring_match", []string{"math", "gardening"}, SeasonSpring, true},
		{"spring_no_match", []string{"math", "latin"}, SeasonSpring, false},
		{"winter_match", []string{"astronomy"}, SeasonWinter, true},
		{"empty_tags", nil, SeasonSummer, false},
	}
	for _, tc := range cases {
		got, _ := HasSeasonalOverlap(tc.tags, tc.season)
		if got != tc.want {
			t.Errorf("HasSeasonalOverlap(%s) = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDetermineDominantSignal_HighRelevance(t *testing.T) {
	result := DetermineDominantSignal(ScoringFactors{Relevance: 0.8, Popularity: 0.3}, "cm", false, SeasonSpring)
	if result.Signal != SourcePurchaseHistory {
		t.Errorf("signal = %q, want %q", result.Signal, SourcePurchaseHistory)
	}
}

func TestDetermineDominantSignal_HighPopularity(t *testing.T) {
	result := DetermineDominantSignal(ScoringFactors{Relevance: 0.3, Popularity: 0.8}, "cm", false, SeasonSpring)
	if result.Signal != SourcePopularity {
		t.Errorf("signal = %q, want %q", result.Signal, SourcePopularity)
	}
}

func TestDetermineDominantSignal_Seasonal(t *testing.T) {
	result := DetermineDominantSignal(ScoringFactors{}, "cm", true, SeasonWinter)
	if result.Signal != SourceSeasonal {
		t.Errorf("signal = %q, want %q", result.Signal, SourceSeasonal)
	}
}

func TestDetermineDominantSignal_DefaultMethodology(t *testing.T) {
	result := DetermineDominantSignal(ScoringFactors{Relevance: 0.3, Popularity: 0.3}, "charlotte-mason", false, SeasonSpring)
	if result.Signal != SourceMethodologyMatch {
		t.Errorf("signal = %q, want %q", result.Signal, SourceMethodologyMatch)
	}
}

// NOTE: methodologyBaselineSubjects and methodologyTransitionAges were pure-function
// helpers that have been replaced by DB queries (migration 27). Integration tests
// for these belong in a test that has a real DB connection.

// ═══════════════════════════════════════════════════════════════════════════════
// Enabled Types Filtering Tests [§11.1]
// ═══════════════════════════════════════════════════════════════════════════════

func TestFilterByEnabledTypes(t *testing.T) {
	candidates := []candidate{
		{RecommendationType: RecommendationMarketplaceContent, SourceSignal: SourceMethodologyMatch},
		{RecommendationType: RecommendationActivityIdea, SourceSignal: SourceProgressGap},
		{RecommendationType: RecommendationReadingSuggestion, SourceSignal: SourceReadingHistory},
		{RecommendationType: RecommendationCommunityGroup, SourceSignal: SourceMethodologyMatch},
	}

	// Only allow marketplace_content and activity_idea.
	enabled := []string{"marketplace_content", "activity_idea"}
	filtered := filterByEnabledTypes(candidates, enabled)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 candidates after filtering, got %d", len(filtered))
	}
	if filtered[0].RecommendationType != RecommendationMarketplaceContent {
		t.Errorf("filtered[0] type = %q, want marketplace_content", filtered[0].RecommendationType)
	}
	if filtered[1].RecommendationType != RecommendationActivityIdea {
		t.Errorf("filtered[1] type = %q, want activity_idea", filtered[1].RecommendationType)
	}
}

func TestFilterByEnabledTypes_PreservesExploration(t *testing.T) {
	candidates := []candidate{
		{RecommendationType: RecommendationMarketplaceContent, SourceSignal: SourceExploration},
		{RecommendationType: RecommendationMarketplaceContent, SourceSignal: SourceMethodologyMatch},
		{RecommendationType: RecommendationActivityIdea, SourceSignal: SourceProgressGap},
	}

	// Only allow activity_idea — but exploration should still pass through.
	enabled := []string{"activity_idea"}
	filtered := filterByEnabledTypes(candidates, enabled)

	if len(filtered) != 2 {
		t.Fatalf("expected 2 candidates (1 exploration + 1 activity_idea), got %d", len(filtered))
	}
	// Exploration candidate preserved despite marketplace_content not being enabled.
	if filtered[0].SourceSignal != SourceExploration {
		t.Errorf("filtered[0] should be exploration candidate, got source_signal=%q", filtered[0].SourceSignal)
	}
	if filtered[1].RecommendationType != RecommendationActivityIdea {
		t.Errorf("filtered[1] type = %q, want activity_idea", filtered[1].RecommendationType)
	}
}
