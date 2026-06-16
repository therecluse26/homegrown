package learner_profile

import (
	"math"
	"testing"
)

func pf64(v float64) *float64 { return &v }

func TestComputeVector_AllAnswered(t *testing.T) {
	// Answer all 12 questions: activity_format = 1.0, session_length = 0.0, rest = 0.5
	answers := []QuizAnswer{
		{QuestionID: 1, Value: pf64(1.0)},
		{QuestionID: 2, Value: pf64(1.0)},
		{QuestionID: 3, Value: pf64(0.0)},
		{QuestionID: 4, Value: pf64(0.0)},
		{QuestionID: 5, Value: pf64(0.5)},
		{QuestionID: 6, Value: pf64(0.5)},
		{QuestionID: 7, Value: pf64(0.5)},
		{QuestionID: 8, Value: pf64(0.5)},
		{QuestionID: 9, Value: pf64(0.5)},
		{QuestionID: 10, Value: pf64(0.5)},
		{QuestionID: 11, Value: pf64(0.5)},
		{QuestionID: 12, Value: pf64(0.5)},
	}

	vec := ComputeVector(answers)

	if vec.AnsweredCount != 12 {
		t.Errorf("answered_count: got %d want 12", vec.AnsweredCount)
	}
	if math.Abs(vec.Confidence-1.0) > 1e-9 {
		t.Errorf("confidence: got %f want 1.0", vec.Confidence)
	}
	if vec.ActivityFormat == nil || math.Abs(*vec.ActivityFormat-1.0) > 1e-9 {
		t.Errorf("activity_format: got %v want 1.0", vec.ActivityFormat)
	}
	if vec.SessionLength == nil || math.Abs(*vec.SessionLength-0.0) > 1e-9 {
		t.Errorf("session_length: got %v want 0.0", vec.SessionLength)
	}
}

func TestComputeVector_PartialAnswers(t *testing.T) {
	// Skip questions 3 and 4 (session_length)
	answers := []QuizAnswer{
		{QuestionID: 1, Value: pf64(0.8)},
		{QuestionID: 2, Value: pf64(0.6)},
		// Q3, Q4 skipped
		{QuestionID: 5, Value: pf64(0.5)},
		{QuestionID: 6, Value: pf64(0.5)},
		{QuestionID: 7, Value: pf64(0.5)},
		{QuestionID: 8, Value: pf64(0.5)},
		{QuestionID: 9, Value: pf64(0.5)},
		{QuestionID: 10, Value: pf64(0.5)},
		{QuestionID: 11, Value: pf64(0.5)},
		{QuestionID: 12, Value: pf64(0.5)},
	}

	vec := ComputeVector(answers)

	if vec.AnsweredCount != 10 {
		t.Errorf("answered_count: got %d want 10", vec.AnsweredCount)
	}
	if vec.SessionLength != nil {
		t.Errorf("session_length should be nil (unanswered), got %v", *vec.SessionLength)
	}
	if vec.ActivityFormat == nil {
		t.Fatal("activity_format should not be nil")
	}
	expected := (0.8 + 0.6) / 2.0
	if math.Abs(*vec.ActivityFormat-expected) > 1e-9 {
		t.Errorf("activity_format: got %f want %f", *vec.ActivityFormat, expected)
	}
}

func TestComputeFitScore_BasicMatch(t *testing.T) {
	vec := DimensionVector{
		ActivityFormat: pf64(0.8),
		SessionLength:  pf64(0.7),
		AnsweredCount:  8,
		Confidence:     8.0 / 12.0,
	}
	preferenceTags := map[string]float64{
		"activity_format": 0.9,
		"session_length":  0.8,
	}

	score, whyText, ok := ComputeFitScore(vec, preferenceTags, nil, nil, "Maya")

	if !ok {
		t.Errorf("expected badge gate to pass, got score=%f", score)
	}
	// mean(1-|0.8-0.9|, 1-|0.7-0.8|) = mean(0.9, 0.9) = 0.9
	if math.Abs(score-0.9) > 1e-9 {
		t.Errorf("score: got %f want 0.9", score)
	}
	if whyText == "" {
		t.Error("expected non-empty why text")
	}
}

func TestComputeFitScore_InterestBoost(t *testing.T) {
	vec := DimensionVector{
		ActivityFormat: pf64(0.5),
		AnsweredCount:  2,
		Confidence:     2.0 / 12.0,
	}
	preferenceTags := map[string]float64{
		"activity_format": 0.55, // score = 1 - 0.05 = 0.95, mean = 0.95 → base score ≥ 0.6
	}
	interests := []string{"art"}
	contentSubjectTags := []string{"art"}

	score, _, ok := ComputeFitScore(vec, preferenceTags, interests, contentSubjectTags, "Lily")
	if !ok {
		t.Errorf("expected badge gate to pass, got score=%f", score)
	}
	// 0.95 + 0.10 interest boost = 1.0 (capped)
	if math.Abs(score-1.0) > 1e-9 {
		t.Errorf("score after interest boost: got %f want 1.0", score)
	}
}

func TestComputeFitScore_BelowGate(t *testing.T) {
	vec := DimensionVector{
		ActivityFormat: pf64(0.0),
		AnsweredCount:  2,
		Confidence:     2.0 / 12.0,
	}
	preferenceTags := map[string]float64{
		"activity_format": 1.0, // score = 1 - 1.0 = 0.0
	}

	_, _, ok := ComputeFitScore(vec, preferenceTags, nil, nil, "Sam")
	if ok {
		t.Error("expected badge gate to fail for score=0.0")
	}
}

func TestComputeFitScore_NoSharedDimensions(t *testing.T) {
	vec := DimensionVector{
		ActivityFormat: pf64(0.8),
		AnsweredCount:  2,
		Confidence:     2.0 / 12.0,
	}
	preferenceTags := map[string]float64{
		"session_length": 0.9, // different dimension than profile has
	}

	_, _, ok := ComputeFitScore(vec, preferenceTags, nil, nil, "Jo")
	if ok {
		t.Error("expected no badge when no shared dimensions")
	}
}
