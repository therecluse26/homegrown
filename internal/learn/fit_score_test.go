package learn

import (
	"testing"
)

// Tests for computeLearnFitScore — the learner-profile fit scoring helper for learn content.
// Validates: family-scope via nil confidence, no-profile→nil, badge gate, interest boost.
// [18-learner-profile §6]

func TestComputeLearnFitScore_NilOnLowConfidence(t *testing.T) {
	profile := LearnStudentFitProfile{
		ActivityFormat: ptrF64(0.8),
		Confidence:     0.40, // below 0.60 threshold
	}
	prefTags := map[string]float64{"activity_format": 0.8}
	_, _, ok := computeLearnFitScore(profile, prefTags, nil, "Alex")
	if ok {
		t.Error("expected ok=false when confidence < 0.60")
	}
}

func TestComputeLearnFitScore_NilOnEmptyPreferenceTags(t *testing.T) {
	profile := LearnStudentFitProfile{
		ActivityFormat: ptrF64(0.8),
		Confidence:     0.80,
	}
	_, _, ok := computeLearnFitScore(profile, nil, nil, "Alex")
	if ok {
		t.Error("expected ok=false when preferenceTags is nil")
	}
}

func TestComputeLearnFitScore_ProfileLessChildReturnsNil(t *testing.T) {
	// Zero confidence = no profile submitted yet. Simulates the family-scope path:
	// when a student is not in the family, RLS returns nil profile → confidence=0.
	profile := LearnStudentFitProfile{
		Confidence: 0.0,
	}
	prefTags := map[string]float64{"activity_format": 0.8}
	_, _, ok := computeLearnFitScore(profile, prefTags, nil, "Test")
	if ok {
		t.Error("expected ok=false for profile-less child (zero confidence)")
	}
}

func TestComputeLearnFitScore_AboveBadgeGate(t *testing.T) {
	profile := LearnStudentFitProfile{
		Structure:  ptrF64(0.9),
		Confidence: 0.80,
	}
	prefTags := map[string]float64{"structure": 0.85}
	score, why, ok := computeLearnFitScore(profile, prefTags, nil, "Sam")
	if !ok {
		t.Error("expected ok=true for high-scoring match")
	}
	if score < 0.60 {
		t.Errorf("expected score >= 0.60, got %.3f", score)
	}
	if why == "" {
		t.Error("expected non-empty why-text")
	}
}

func TestComputeLearnFitScore_InterestBoostCrossesGate(t *testing.T) {
	profile := LearnStudentFitProfile{
		ActivityFormat: ptrF64(0.7),
		Confidence:     0.80,
		Interests:      []string{"reading"}, // maps to "reading"
	}
	prefTags := map[string]float64{"activity_format": 0.2} // 0.5 base
	contentSubjectTags := []string{"reading"}
	_, _, ok := computeLearnFitScore(profile, prefTags, contentSubjectTags, "Sam")
	if !ok {
		t.Error("expected ok=true with interest boost crossing badge gate")
	}
}

func TestComputeLearnFitScore_WhyTextHasStudentName(t *testing.T) {
	profile := LearnStudentFitProfile{
		OutdoorKinesthetic: ptrF64(0.9),
		Confidence:         0.80,
	}
	prefTags := map[string]float64{"outdoor_kinesthetic": 0.85}
	_, why, ok := computeLearnFitScore(profile, prefTags, nil, "Mia")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !strContains(why, "Mia") {
		t.Errorf("expected why-text to contain 'Mia', got: %q", why)
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func ptrF64(v float64) *float64 { return &v }

func strContains(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
