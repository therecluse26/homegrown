package mkt

import (
	"testing"
)

// Tests for computeMktFitScore — the learner-profile fit scoring helper.
// Validates: family-scope via nil profile, no-profile→nil, badge gate, interest boost.
// [18-learner-profile §6]

func TestComputeMktFitScore_NilOnLowConfidence(t *testing.T) {
	profile := StudentFitProfile{
		ActivityFormat: ptr64(0.8),
		Confidence:     0.40, // below 0.60 threshold
	}
	prefTags := map[string]float64{"activity_format": 0.8}
	_, _, ok := computeMktFitScore(profile, prefTags, nil, "Alex")
	if ok {
		t.Error("expected ok=false when confidence < 0.60")
	}
}

func TestComputeMktFitScore_NilOnEmptyPreferenceTags(t *testing.T) {
	profile := StudentFitProfile{
		ActivityFormat: ptr64(0.8),
		Confidence:     0.80,
	}
	_, _, ok := computeMktFitScore(profile, nil, nil, "Alex")
	if ok {
		t.Error("expected ok=false when preferenceTags is nil")
	}
}

func TestComputeMktFitScore_NilOnNoDimensionOverlap(t *testing.T) {
	// Profile has activity_format only; content tags have session_length only.
	profile := StudentFitProfile{
		ActivityFormat: ptr64(0.8),
		Confidence:     0.80,
	}
	prefTags := map[string]float64{"session_length": 0.5}
	_, _, ok := computeMktFitScore(profile, prefTags, nil, "Alex")
	if ok {
		t.Error("expected ok=false when no dimensional overlap")
	}
}

func TestComputeMktFitScore_BelowBadgeGate(t *testing.T) {
	// Perfect mismatch on one dimension → score = 0.0
	profile := StudentFitProfile{
		ActivityFormat: ptr64(0.0),
		Confidence:     0.80,
	}
	prefTags := map[string]float64{"activity_format": 1.0}
	score, _, ok := computeMktFitScore(profile, prefTags, nil, "Alex")
	if ok {
		t.Errorf("expected ok=false for score %.2f below badge gate", score)
	}
}

func TestComputeMktFitScore_AboveBadgeGate(t *testing.T) {
	// Near-perfect match on activity_format.
	profile := StudentFitProfile{
		ActivityFormat: ptr64(0.9),
		Confidence:     0.80,
	}
	prefTags := map[string]float64{"activity_format": 0.8}
	score, why, ok := computeMktFitScore(profile, prefTags, nil, "Alex")
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

func TestComputeMktFitScore_InterestBoost(t *testing.T) {
	// Moderate base score; interest boost should push it over the badge gate.
	profile := StudentFitProfile{
		ActivityFormat: ptr64(0.7),
		Confidence:     0.80,
		Interests:      []string{"math"}, // maps to "mathematics"
	}
	prefTags := map[string]float64{"activity_format": 0.2} // score = 0.5 without boost
	contentSubjectTags := []string{"mathematics"}          // matching interest
	score, _, ok := computeMktFitScore(profile, prefTags, contentSubjectTags, "Alex")
	// 0.5 + 0.10 = 0.60 → exactly at badge gate
	if !ok {
		t.Errorf("expected ok=true with interest boost, score=%.3f", score)
	}
}

func TestComputeMktFitScore_WhyTextContainsStudentName(t *testing.T) {
	profile := StudentFitProfile{
		SessionLength: ptr64(0.9),
		Confidence:    0.80,
	}
	prefTags := map[string]float64{"session_length": 0.85}
	_, why, ok := computeMktFitScore(profile, prefTags, nil, "Jordan")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if why == "" {
		t.Fatal("expected non-empty why-text")
	}
	// Why text should contain the student's name.
	if !containsStr(why, "Jordan") {
		t.Errorf("expected why-text to contain 'Jordan', got: %q", why)
	}
}

func TestComputeMktFitScore_FamilyScopeEnforcement(t *testing.T) {
	// When the caller passes a nil profile (returned when student is not in family via RLS),
	// the service must NOT produce a fit score. This test verifies the nil-profile → nil-score path.
	//
	// The actual family-scope check is enforced in the BrowseListings service method:
	// profile, err := s.learnerProfile.GetStudentFitProfile(ctx, scope, studentID)
	// If the student is not in the family, RLS causes the repo to return nil → no fit score.
	//
	// Simulate: nil profile data → score should be absent.
	profile := StudentFitProfile{
		Confidence: 0.0, // zero confidence = profile-less child
	}
	prefTags := map[string]float64{"activity_format": 0.8}
	_, _, ok := computeMktFitScore(profile, prefTags, nil, "Test")
	if ok {
		t.Error("expected ok=false for profile-less child (zero confidence)")
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func ptr64(v float64) *float64 { return &v }

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && (s[:len(sub)] == sub || containsStr(s[1:], sub)))
}
