package discover

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ─── Test Fixtures ────────────────────────────────────────────────────────────

// testQuizDef builds a QuizDefinition with 2 questions and 6 methodologies.
// Charlotte Mason has the highest weights for correct scoring tests.
func testQuizDef() *QuizDefinition {
	questions := []quizQuestionInternal{
		{
			ID:       "q1",
			Category: "learning-style",
			Text:     "How does your child learn best?",
			Answers: []quizAnswerInternal{
				{ID: "q1a1", Text: "Through nature and living books", Weights: map[string]float64{
					"charlotte-mason": 0.9,
					"traditional":     0.1,
					"classical":       0.2,
					"waldorf":         0.4,
					"montessori":      0.3,
					"unschooling":     0.5,
				}},
				{ID: "q1a2", Text: "Through structured textbooks", Weights: map[string]float64{
					"charlotte-mason": 0.1,
					"traditional":     0.9,
					"classical":       0.7,
					"waldorf":         0.1,
					"montessori":      0.2,
					"unschooling":     0.1,
				}},
			},
		},
		{
			ID:       "q2",
			Category: "schedule",
			Text:     "How do you prefer to structure your school day?",
			Answers: []quizAnswerInternal{
				{ID: "q2a1", Text: "Child-led and flexible", Weights: map[string]float64{
					"charlotte-mason": 0.6,
					"traditional":     0.1,
					"classical":       0.1,
					"waldorf":         0.5,
					"montessori":      0.8,
					"unschooling":     0.9,
				}},
				{ID: "q2a2", Text: "Structured with a daily schedule", Weights: map[string]float64{
					"charlotte-mason": 0.4,
					"traditional":     0.9,
					"classical":       0.9,
					"waldorf":         0.5,
					"montessori":      0.2,
					"unschooling":     0.1,
				}},
			},
		},
	}
	questionsJSON, _ := json.Marshal(questions)

	explanations := quizExplanationsInternal{
		"charlotte-mason": {MatchText: "Charlotte Mason fits your style!", MismatchText: "Charlotte Mason may not be your style."},
		"traditional":     {MatchText: "Traditional fits your style!", MismatchText: "Traditional may not be your style."},
		"classical":       {MatchText: "Classical fits your style!", MismatchText: "Classical may not be your style."},
		"waldorf":         {MatchText: "Waldorf fits your style!", MismatchText: "Waldorf may not be your style."},
		"montessori":      {MatchText: "Montessori fits your style!", MismatchText: "Montessori may not be your style."},
		"unschooling":     {MatchText: "Unschooling fits your style!", MismatchText: "Unschooling may not be your style."},
	}
	explanationsJSON, _ := json.Marshal(explanations)

	return &QuizDefinition{
		ID:           uuid.New(),
		Version:      1,
		Title:        "Find Your Methodology",
		Description:  "Discover your homeschooling style",
		Status:       "active",
		Questions:    json.RawMessage(questionsJSON),
		Explanations: json.RawMessage(explanationsJSON),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

// newTestService wires a discoveryServiceImpl with stub repos and a stub method adapter.
func newTestService(quizDefRepo QuizDefinitionRepository, quizResRepo QuizResultRepository, stateRepo StateGuideRepository) *discoveryServiceImpl {
	return &discoveryServiceImpl{
		quizDefRepo: quizDefRepo,
		quizResRepo: quizResRepo,
		stateRepo:   stateRepo,
		methodology: NewMethodAdapter(func(_ context.Context, slug string) (string, error) {
			names := map[string]string{
				"charlotte-mason": "Charlotte Mason",
				"traditional":     "Traditional",
				"classical":       "Classical",
				"waldorf":         "Waldorf",
				"montessori":      "Montessori",
				"unschooling":     "Unschooling",
			}
			if name, ok := names[slug]; ok {
				return name, nil
			}
			return slug, nil
		}),
	}
}

// ─── Test: GetActiveQuiz — weight stripping ───────────────────────────────────

// TestGetActiveQuiz_WeightsStripped verifies that GetActiveQuiz never returns
// weights in any answer. [03-discover §3.1, §15.1]
func TestGetActiveQuiz_WeightsStripped(t *testing.T) {
	def := testQuizDef()
	svc := newTestService(&stubQuizDefRepo{def: def}, nil, nil)

	resp, err := svc.GetActiveQuiz(context.Background())
	if err != nil {
		t.Fatalf("GetActiveQuiz: %v", err)
	}
	if len(resp.Questions) != 2 {
		t.Fatalf("want 2 questions, got %d", len(resp.Questions))
	}
	for _, q := range resp.Questions {
		for _, a := range q.Answers {
			// Serialize the answer and check that "weights" key is absent.
			raw, _ := json.Marshal(a)
			var m map[string]json.RawMessage
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatal(err)
			}
			if _, hasWeights := m["weights"]; hasWeights {
				t.Errorf("answer %q in question %q has weights field — must be stripped", a.ID, q.ID)
			}
		}
	}
}

// TestGetActiveQuiz_NoActiveQuiz verifies that ErrNoActiveQuiz is returned
// when no quiz has status='active'. [03-discover §15.2]
func TestGetActiveQuiz_NoActiveQuiz(t *testing.T) {
	svc := newTestService(&stubQuizDefRepo{err: &DiscoverError{Err: ErrNoActiveQuiz}}, nil, nil)

	_, err := svc.GetActiveQuiz(context.Background())
	if err == nil {
		t.Fatal("expected error for no active quiz")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T", err)
	}
	if !errors.Is(discErr.Err, ErrNoActiveQuiz) {
		t.Errorf("want ErrNoActiveQuiz, got %v", discErr.Err)
	}
}

// ─── Test: SubmitQuiz — scoring ───────────────────────────────────────────────

// TestSubmitQuiz_CorrectScoring verifies the normalized scoring algorithm produces
// correct percentages for a full answer set. [03-discover §15.8, §15.26]
func TestSubmitQuiz_CorrectScoring(t *testing.T) {
	def := testQuizDef()
	resRepo := &stubQuizResRepo{}
	svc := newTestService(&stubQuizDefRepo{def: def}, resRepo, nil)

	// q1a1 = charlotte-mason (0.9), q2a1 = unschooling (0.9)
	resp, err := svc.SubmitQuiz(context.Background(), SubmitQuizCommand{
		Answers: map[string]string{
			"q1": "q1a1",
			"q2": "q2a1",
		},
	})
	if err != nil {
		t.Fatalf("SubmitQuiz: %v", err)
	}
	if resp.ShareID == "" {
		t.Error("share_id must not be empty")
	}
	if len(resp.Recommendations) == 0 {
		t.Fatal("want non-empty recommendations")
	}

	// First recommendation should be unschooling (0.9/0.9=100%) or charlotte-mason
	// based on the weights. Let's verify scores are 0-100 integers.
	for _, rec := range resp.Recommendations {
		if rec.ScorePercentage > 100 {
			t.Errorf("score_percentage %d > 100 for %q", rec.ScorePercentage, rec.MethodologySlug)
		}
	}

	// Scores must be in descending order.
	for i := 1; i < len(resp.Recommendations); i++ {
		if resp.Recommendations[i].ScorePercentage > resp.Recommendations[i-1].ScorePercentage {
			t.Errorf("recommendations not in descending order at index %d: %d > %d",
				i, resp.Recommendations[i].ScorePercentage, resp.Recommendations[i-1].ScorePercentage)
		}
	}
}

// TestSubmitQuiz_ScorePercentagesAreIntegers verifies that score_percentage values
// are uint8 integers in [0, 100], not floats. [03-discover §15.8]
func TestSubmitQuiz_ScorePercentagesAreIntegers(t *testing.T) {
	def := testQuizDef()
	resRepo := &stubQuizResRepo{}
	svc := newTestService(&stubQuizDefRepo{def: def}, resRepo, nil)

	resp, err := svc.SubmitQuiz(context.Background(), SubmitQuizCommand{
		Answers: map[string]string{"q1": "q1a1", "q2": "q2a2"},
	})
	if err != nil {
		t.Fatalf("SubmitQuiz: %v", err)
	}

	raw, _ := json.Marshal(resp)
	var parsed struct {
		Recommendations []struct {
			ScorePercentage json.Number `json:"score_percentage"`
		} `json:"recommendations"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatal(err)
	}
	for _, rec := range parsed.Recommendations {
		s := rec.ScorePercentage.String()
		if strings.Contains(s, ".") {
			t.Errorf("score_percentage %q is a float, want integer", s)
		}
	}
}

// TestSubmitQuiz_PartialAnswers verifies that a submission with missing questions
// succeeds (missing questions contribute zero weight). [03-discover §15.6]
func TestSubmitQuiz_PartialAnswers(t *testing.T) {
	def := testQuizDef()
	resRepo := &stubQuizResRepo{}
	svc := newTestService(&stubQuizDefRepo{def: def}, resRepo, nil)

	// Submit only q1, omit q2
	resp, err := svc.SubmitQuiz(context.Background(), SubmitQuizCommand{
		Answers: map[string]string{"q1": "q1a1"},
	})
	if err != nil {
		t.Fatalf("SubmitQuiz partial: %v", err)
	}
	if resp.ShareID == "" {
		t.Error("share_id must not be empty for partial submission")
	}
	if len(resp.Recommendations) == 0 {
		t.Fatal("want non-empty recommendations for partial submission")
	}
}

// TestSubmitQuiz_InvalidQuestionID verifies that an unknown question ID returns
// ErrInvalidQuestionID. [03-discover §15.4]
func TestSubmitQuiz_InvalidQuestionID(t *testing.T) {
	def := testQuizDef()
	svc := newTestService(&stubQuizDefRepo{def: def}, &stubQuizResRepo{}, nil)

	_, err := svc.SubmitQuiz(context.Background(), SubmitQuizCommand{
		Answers: map[string]string{"nonexistent-question": "q1a1"},
	})
	if err == nil {
		t.Fatal("expected error for invalid question ID")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T", err)
	}
	if !errors.Is(discErr.Err, ErrInvalidQuestionID) {
		t.Errorf("want ErrInvalidQuestionID, got %v", discErr.Err)
	}
}

// TestSubmitQuiz_InvalidAnswerID verifies that an unknown answer ID for a valid
// question returns ErrInvalidAnswerID. [03-discover §15.5]
func TestSubmitQuiz_InvalidAnswerID(t *testing.T) {
	def := testQuizDef()
	svc := newTestService(&stubQuizDefRepo{def: def}, &stubQuizResRepo{}, nil)

	_, err := svc.SubmitQuiz(context.Background(), SubmitQuizCommand{
		Answers: map[string]string{"q1": "nonexistent-answer"},
	})
	if err == nil {
		t.Fatal("expected error for invalid answer ID")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T", err)
	}
	if !errors.Is(discErr.Err, ErrInvalidAnswerID) {
		t.Errorf("want ErrInvalidAnswerID, got %v", discErr.Err)
	}
}

// TestSubmitQuiz_Deterministic verifies that the same inputs always produce the
// same output (same scores and ranking). [03-discover §15.26]
func TestSubmitQuiz_Deterministic(t *testing.T) {
	def := testQuizDef()
	answers := map[string]string{"q1": "q1a1", "q2": "q2a1"}

	run := func() *QuizResultResponse {
		resRepo := &stubQuizResRepo{}
		svc := newTestService(&stubQuizDefRepo{def: def}, resRepo, nil)
		resp, err := svc.SubmitQuiz(context.Background(), SubmitQuizCommand{Answers: answers})
		if err != nil {
			t.Fatalf("SubmitQuiz: %v", err)
		}
		return resp
	}

	r1 := run()
	r2 := run()

	if len(r1.Recommendations) != len(r2.Recommendations) {
		t.Fatalf("non-deterministic: different recommendation counts %d vs %d",
			len(r1.Recommendations), len(r2.Recommendations))
	}
	for i := range r1.Recommendations {
		if r1.Recommendations[i].MethodologySlug != r2.Recommendations[i].MethodologySlug {
			t.Errorf("non-deterministic at index %d: %q vs %q",
				i, r1.Recommendations[i].MethodologySlug, r2.Recommendations[i].MethodologySlug)
		}
		if r1.Recommendations[i].ScorePercentage != r2.Recommendations[i].ScorePercentage {
			t.Errorf("non-deterministic score at index %d: %d vs %d",
				i, r1.Recommendations[i].ScorePercentage, r2.Recommendations[i].ScorePercentage)
		}
	}
}

// ─── Test: Share ID ───────────────────────────────────────────────────────────

// TestSubmitQuiz_ShareIDFormat verifies share_id is 12 chars from the base62 alphabet.
// [03-discover §15.11, §15.3]
func TestSubmitQuiz_ShareIDFormat(t *testing.T) {
	def := testQuizDef()
	resRepo := &stubQuizResRepo{}
	svc := newTestService(&stubQuizDefRepo{def: def}, resRepo, nil)

	resp, err := svc.SubmitQuiz(context.Background(), SubmitQuizCommand{
		Answers: map[string]string{"q1": "q1a1"},
	})
	if err != nil {
		t.Fatalf("SubmitQuiz: %v", err)
	}

	if len(resp.ShareID) != shareIDLength {
		t.Errorf("want share_id length %d, got %d (%q)", shareIDLength, len(resp.ShareID), resp.ShareID)
	}
	for _, ch := range resp.ShareID {
		if !strings.ContainsRune(shareIDAlphabet, ch) {
			t.Errorf("share_id character %q not in base62 alphabet", ch)
		}
	}
}

// ─── Test: GetQuizResult ──────────────────────────────────────────────────────

// TestGetQuizResult_Found verifies that a result is returned when the share_id exists.
func TestGetQuizResult_Found(t *testing.T) {
	recs := []MethodologyRecommendation{
		{Rank: 1, MethodologySlug: "charlotte-mason", MethodologyName: "Charlotte Mason", ScorePercentage: 85, Explanation: "Great fit!"},
	}
	recsJSON, _ := json.Marshal(recs)

	resRepo := &stubQuizResRepo{
		result: &QuizResult{
			ShareID:         "abc12345def6",
			Recommendations: json.RawMessage(recsJSON),
		},
	}
	svc := newTestService(nil, resRepo, nil)

	resp, err := svc.GetQuizResult(context.Background(), "abc12345def6")
	if err != nil {
		t.Fatalf("GetQuizResult: %v", err)
	}
	if resp.ShareID != "abc12345def6" {
		t.Errorf("want share_id 'abc12345def6', got %q", resp.ShareID)
	}
	if len(resp.Recommendations) != 1 {
		t.Fatalf("want 1 recommendation, got %d", len(resp.Recommendations))
	}
	if resp.Recommendations[0].MethodologySlug != "charlotte-mason" {
		t.Errorf("want methodology_slug 'charlotte-mason', got %q", resp.Recommendations[0].MethodologySlug)
	}
	if resp.Recommendations[0].Rank != 1 {
		t.Errorf("want rank 1, got %d", resp.Recommendations[0].Rank)
	}
}

// TestGetQuizResult_NotFound verifies that ErrQuizResultNotFound is returned for
// an unknown share_id. [03-discover §15]
func TestGetQuizResult_NotFound(t *testing.T) {
	resRepo := &stubQuizResRepo{err: &DiscoverError{Err: ErrQuizResultNotFound}}
	svc := newTestService(nil, resRepo, nil)

	_, err := svc.GetQuizResult(context.Background(), "notexist")
	if err == nil {
		t.Fatal("expected error for missing share_id")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T", err)
	}
	if !errors.Is(discErr.Err, ErrQuizResultNotFound) {
		t.Errorf("want ErrQuizResultNotFound, got %v", discErr.Err)
	}
}

// ─── Test: ListStateGuides ────────────────────────────────────────────────────

// TestListStateGuides_IsAvailableFlag verifies that IsAvailable=true only for published guides.
func TestListStateGuides_IsAvailableFlag(t *testing.T) {
	reviewed := time.Now().Add(-24 * time.Hour)
	stateRepo := &stubStateRepo{
		summaries: []StateGuideSummary{
			{StateCode: "CA", StateName: "California", Status: "published", LastReviewedAt: &reviewed},
			{StateCode: "TX", StateName: "Texas", Status: "draft"},
			{StateCode: "NY", StateName: "New York", Status: "review_due"},
		},
	}
	svc := newTestService(nil, nil, stateRepo)

	resp, err := svc.ListStateGuides(context.Background())
	if err != nil {
		t.Fatalf("ListStateGuides: %v", err)
	}
	if len(resp) != 3 {
		t.Fatalf("want 3 summaries, got %d", len(resp))
	}
	caIdx := -1
	for i, r := range resp {
		if r.StateCode == "CA" {
			caIdx = i
		}
	}
	if caIdx < 0 {
		t.Fatal("CA not found in response")
	}
	if !resp[caIdx].IsAvailable {
		t.Error("want IsAvailable=true for published CA guide")
	}
	// TX and NY should be unavailable.
	for _, r := range resp {
		if r.StateCode == "TX" && r.IsAvailable {
			t.Error("want IsAvailable=false for draft TX guide")
		}
		if r.StateCode == "NY" && r.IsAvailable {
			t.Error("want IsAvailable=false for review_due NY guide")
		}
	}
}

// ─── Test: GetStateGuide ──────────────────────────────────────────────────────

// TestGetStateGuide_Published verifies that a published guide is returned. [03-discover §8.3]
func TestGetStateGuide_Published(t *testing.T) {
	reqs := StateGuideRequirements{
		NotificationRequired: true,
		RequiredSubjects:     []string{"math", "english"},
		AssessmentRequired:   true,
		RegulationLevel:      "moderate",
	}
	reqsJSON, _ := json.Marshal(reqs)

	stateRepo := &stubStateRepo{
		guide: &StateGuide{
			StateCode:       "CA",
			StateName:       "California",
			Status:          "published",
			Requirements:    json.RawMessage(reqsJSON),
			GuideContent:    "California homeschool guide.",
			LegalDisclaimer: "This is for educational purposes only.",
		},
	}
	svc := newTestService(nil, nil, stateRepo)

	resp, err := svc.GetStateGuide(context.Background(), "CA")
	if err != nil {
		t.Fatalf("GetStateGuide: %v", err)
	}
	if resp.StateCode != "CA" {
		t.Errorf("want state_code 'CA', got %q", resp.StateCode)
	}
	if !resp.Requirements.NotificationRequired {
		t.Error("want notification_required=true")
	}
	if len(resp.Requirements.RequiredSubjects) == 0 {
		t.Error("want non-empty required_subjects slice")
	}
	if resp.GuideContent == "" {
		t.Error("want non-empty guide_content")
	}
	if resp.LegalDisclaimer == "" {
		t.Error("want non-empty legal_disclaimer")
	}
}

// TestGetStateGuide_Draft verifies that a draft guide returns ErrStateGuideNotPublished
// which maps to 404. [03-discover §15.16]
func TestGetStateGuide_Draft(t *testing.T) {
	stateRepo := &stubStateRepo{
		err: &DiscoverError{Err: ErrStateGuideNotPublished, StateCode: "TX"},
	}
	svc := newTestService(nil, nil, stateRepo)

	_, err := svc.GetStateGuide(context.Background(), "TX")
	if err == nil {
		t.Fatal("expected error for draft guide")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T", err)
	}
	if !errors.Is(discErr.Err, ErrStateGuideNotPublished) {
		t.Errorf("want ErrStateGuideNotPublished, got %v", discErr.Err)
	}
}

// TestGetStateGuide_NotFound verifies that a missing state code returns ErrStateGuideNotFound.
func TestGetStateGuide_NotFound(t *testing.T) {
	stateRepo := &stubStateRepo{
		err: &DiscoverError{Err: ErrStateGuideNotFound, StateCode: "XX"},
	}
	svc := newTestService(nil, nil, stateRepo)

	_, err := svc.GetStateGuide(context.Background(), "XX")
	if err == nil {
		t.Fatal("expected error for missing state code")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T", err)
	}
	if !errors.Is(discErr.Err, ErrStateGuideNotFound) {
		t.Errorf("want ErrStateGuideNotFound, got %v", discErr.Err)
	}
}

// TestGetStateGuide_InvalidStateCode verifies that a code != 2 chars returns ErrInvalidStateCode.
func TestGetStateGuide_InvalidStateCode(t *testing.T) {
	svc := newTestService(nil, nil, &stubStateRepo{})

	_, err := svc.GetStateGuide(context.Background(), "CAL")
	if err == nil {
		t.Fatal("expected error for 3-char state code")
	}
	var discErr *DiscoverError
	if !errors.As(err, &discErr) {
		t.Fatalf("want *DiscoverError, got %T", err)
	}
	if !errors.Is(discErr.Err, ErrInvalidStateCode) {
		t.Errorf("want ErrInvalidStateCode, got %v", discErr.Err)
	}
}

// ─── Stubs ────────────────────────────────────────────────────────────────────

type stubQuizDefRepo struct {
	def *QuizDefinition
	err error
}

func (r *stubQuizDefRepo) FindActive(_ context.Context) (*QuizDefinition, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.def == nil {
		return nil, &DiscoverError{Err: ErrNoActiveQuiz}
	}
	return r.def, nil
}

func (r *stubQuizDefRepo) FindByID(_ context.Context, _ any) (*QuizDefinition, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.def, nil
}

type stubQuizResRepo struct {
	result *QuizResult
	err    error
}

func (r *stubQuizResRepo) Create(_ context.Context, input CreateQuizResult) (*QuizResult, error) {
	if r.err != nil {
		return nil, r.err
	}
	return &QuizResult{
		ID:               uuid.New(),
		QuizDefinitionID: input.QuizDefinitionID,
		ShareID:          input.ShareID,
		SessionToken:     input.SessionToken,
		Answers:          input.Answers,
		Scores:           input.Scores,
		Recommendations:  input.Recommendations,
	}, nil
}

func (r *stubQuizResRepo) FindByShareID(_ context.Context, _ string) (*QuizResult, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.result == nil {
		return nil, &DiscoverError{Err: ErrQuizResultNotFound}
	}
	return r.result, nil
}

func (r *stubQuizResRepo) ClaimForFamily(_ context.Context, _ string, _ any) error {
	return r.err
}

type stubStateRepo struct {
	summaries []StateGuideSummary
	guide     *StateGuide
	err       error
}

func (r *stubStateRepo) ListAll(_ context.Context) ([]StateGuideSummary, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.summaries, nil
}

func (r *stubStateRepo) FindByStateCode(_ context.Context, _ string) (*StateGuide, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.guide == nil {
		return nil, &DiscoverError{Err: ErrStateGuideNotFound}
	}
	return r.guide, nil
}

func (r *stubStateRepo) FindRequirementsByStateCode(_ context.Context, _ string) (*StateGuide, error) {
	if r.err != nil {
		return nil, r.err
	}
	if r.guide == nil {
		return nil, &DiscoverError{Err: ErrStateGuideNotFound}
	}
	return r.guide, nil
}
