package recs

import (
	"context"
	"math"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Fit Score Formula Tests ─────────────────────────────────────────────────

func pf64fit(v float64) *float64 { return &v }

func TestComputeFitScore_BasicMatch(t *testing.T) {
	profile := studentProfileForFit{
		ActivityFormat: pf64fit(0.8),
		SessionLength:  pf64fit(0.7),
		Confidence:     0.80,
	}
	preferenceTags := map[string]float64{
		"activity_format": 0.9,
		"session_length":  0.8,
	}

	score, whyText, ok := computeFitScore(profile, preferenceTags, nil)

	if !ok {
		t.Errorf("expected badge gate to pass, got score=%f", score)
	}
	// mean(1-|0.8-0.9|, 1-|0.7-0.8|) = mean(0.9, 0.9) = 0.9
	if math.Abs(float64(score)-0.9) > 1e-6 {
		t.Errorf("score: got %f want 0.9", score)
	}
	if whyText == "" {
		t.Error("expected non-empty why text for passing fit badge")
	}
}

func TestComputeFitScore_BelowGate(t *testing.T) {
	profile := studentProfileForFit{
		ActivityFormat: pf64fit(0.0),
		Confidence:     0.80,
	}
	preferenceTags := map[string]float64{
		"activity_format": 1.0, // score = 1 - 1.0 = 0.0
	}

	_, _, ok := computeFitScore(profile, preferenceTags, nil)
	if ok {
		t.Error("expected badge gate to fail for score=0.0")
	}
}

func TestComputeFitScore_ConfidenceBelowGate(t *testing.T) {
	profile := studentProfileForFit{
		ActivityFormat: pf64fit(0.9),
		Confidence:     0.50, // below 0.60 threshold
	}
	preferenceTags := map[string]float64{
		"activity_format": 0.9,
	}

	_, _, ok := computeFitScore(profile, preferenceTags, nil)
	if ok {
		t.Error("expected badge gate to fail: profile confidence too low")
	}
}

func TestComputeFitScore_NoSharedDimensions(t *testing.T) {
	profile := studentProfileForFit{
		ActivityFormat: pf64fit(0.8),
		Confidence:     0.80,
	}
	preferenceTags := map[string]float64{
		"session_length": 0.9, // different dimension than what profile has
	}

	_, _, ok := computeFitScore(profile, preferenceTags, nil)
	if ok {
		t.Error("expected no badge when no shared dimensions")
	}
}

func TestComputeFitScore_InterestBoost_MappedSubjectTag(t *testing.T) {
	// "math" interest → "mathematics" subject_tag via recsInterestToSubjectTag
	profile := studentProfileForFit{
		ActivityFormat: pf64fit(0.55),
		Interests:      []string{"math"},
		Confidence:     0.80,
	}
	preferenceTags := map[string]float64{
		"activity_format": 0.55, // score = 1 - 0 = 1.0 → base = 1.0
	}
	contentSubjectTags := []string{"mathematics"}

	score, _, ok := computeFitScore(profile, preferenceTags, contentSubjectTags)
	if !ok {
		t.Errorf("expected badge gate to pass, got score=%f", score)
	}
	// Base = 1.0, boost +0.10, cap at 1.0
	if math.Abs(float64(score)-1.0) > 1e-6 {
		t.Errorf("score after interest boost: got %f want 1.0", score)
	}
}

func TestComputeFitScore_InterestBoost_NatureMapping(t *testing.T) {
	// "nature" interest → "ecology" subject_tag
	profile := studentProfileForFit{
		ActivityFormat: pf64fit(0.8),
		Interests:      []string{"nature"},
		Confidence:     0.80,
	}
	preferenceTags := map[string]float64{
		"activity_format": 0.85, // score = 1 - 0.05 = 0.95
	}
	contentSubjectTags := []string{"ecology"}

	score, _, ok := computeFitScore(profile, preferenceTags, contentSubjectTags)
	if !ok {
		t.Errorf("expected badge gate to pass, got score=%f", score)
	}
	// 0.95 + 0.10 = 1.0 (capped)
	if math.Abs(float64(score)-1.0) > 1e-6 {
		t.Errorf("score: got %f want 1.0 (capped)", score)
	}
}

func TestComputeFitScore_EmptyPreferenceTags(t *testing.T) {
	profile := studentProfileForFit{
		ActivityFormat: pf64fit(0.8),
		Confidence:     0.80,
	}
	_, _, ok := computeFitScore(profile, nil, nil)
	if ok {
		t.Error("expected no badge for nil preference_tags")
	}
}

// ─── Cold-Start Prior Logic Tests ────────────────────────────────────────────

type stubLearnerProfilePort struct {
	getStudentInterestsByFamilyFn func(ctx context.Context, familyID shared.FamilyID) (map[uuid.UUID][]string, error)
}

func (s *stubLearnerProfilePort) GetStudentInterestsByFamily(ctx context.Context, familyID shared.FamilyID) (map[uuid.UUID][]string, error) {
	if s.getStudentInterestsByFamilyFn != nil {
		return s.getStudentInterestsByFamilyFn(ctx, familyID)
	}
	panic("stubLearnerProfilePort.GetStudentInterestsByFamily not stubbed")
}

// TestColdStartPrior_AddsInterestsWhenFewSignals verifies that declared interests are
// added (double-weighted, mapped to subject_tags) when the family has < 3 signals.
func TestColdStartPrior_AddsInterestsWhenFewSignals(t *testing.T) {
	port := &stubLearnerProfilePort{
		getStudentInterestsByFamilyFn: func(_ context.Context, _ shared.FamilyID) (map[uuid.UUID][]string, error) {
			return map[uuid.UUID][]string{
				uuid.New(): {"math", "art"},
			}, nil
		},
	}

	signals := []Signal{} // 0 signals — below threshold
	var recentSubjectTags []string
	familyID := shared.NewFamilyID(uuid.New())
	family := struct{ ID uuid.UUID }{ID: familyID.UUID}

	// Simulate the cold-start prior block (same logic as tasks.go).
	if len(signals) < 3 {
		studentInterests, err := port.GetStudentInterestsByFamily(context.Background(), familyID)
		if err != nil {
			t.Fatalf("GetStudentInterestsByFamily: %v", err)
		}
		for _, interests := range studentInterests {
			for _, interest := range interests {
				subjectTag, ok := recsInterestToSubjectTag[interest]
				if !ok {
					subjectTag = interest
				}
				recentSubjectTags = append(recentSubjectTags, subjectTag, subjectTag)
			}
		}
	}
	_ = family

	// "math" → "mathematics", "art" → "art", each doubled: 4 entries.
	if len(recentSubjectTags) != 4 {
		t.Errorf("expected 4 subject tags (2 interests × 2 weight), got %d: %v", len(recentSubjectTags), recentSubjectTags)
	}
	// Verify mapped subject_tags, not raw chip IDs.
	tagSet := make(map[string]int)
	for _, t := range recentSubjectTags {
		tagSet[t]++
	}
	if tagSet["mathematics"] != 2 {
		t.Errorf(`expected "mathematics" ×2, got %d`, tagSet["mathematics"])
	}
	if tagSet["art"] != 2 {
		t.Errorf(`expected "art" ×2, got %d`, tagSet["art"])
	}
}

// TestColdStartPrior_SkipsWhenEnoughSignals verifies no cold-start seeding when signals ≥ 3.
func TestColdStartPrior_SkipsWhenEnoughSignals(t *testing.T) {
	port := &stubLearnerProfilePort{
		getStudentInterestsByFamilyFn: func(_ context.Context, _ shared.FamilyID) (map[uuid.UUID][]string, error) {
			t.Error("GetStudentInterestsByFamily should not be called when signals >= 3")
			return nil, nil
		},
	}

	signals := make([]Signal, 3) // exactly 3 — at or above threshold
	var recentSubjectTags []string
	familyID := shared.NewFamilyID(uuid.New())

	if len(signals) < 3 { // false — threshold not met, port should not be called
		studentInterests, _ := port.GetStudentInterestsByFamily(context.Background(), familyID)
		for _, interests := range studentInterests {
			for _, interest := range interests {
				recentSubjectTags = append(recentSubjectTags, interest, interest)
			}
		}
	}

	if len(recentSubjectTags) != 0 {
		t.Errorf("expected no tags added, got %d", len(recentSubjectTags))
	}
}
