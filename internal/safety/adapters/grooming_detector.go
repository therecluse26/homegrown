package adapters

import (
	"context"
	"log/slog"
	"strings"

	"github.com/homegrown-academy/homegrown-academy/internal/safety"
)

// NoopGroomingDetector returns a zero score for all text.
// Placeholder until an ML service (Comprehend, Perspective API) is integrated.
type NoopGroomingDetector struct{}

func NewNoopGroomingDetector() *NoopGroomingDetector {
	return &NoopGroomingDetector{}
}

func (d *NoopGroomingDetector) Analyze(_ context.Context, text string) (*safety.GroomingAnalysisResult, error) {
	slog.Debug("grooming detector: noop analysis", "text_length", len(text))
	return &safety.GroomingAnalysisResult{
		Score:        0.0,
		ModelVersion: "noop-v1",
		Flagged:      false,
	}, nil
}

// KeywordGroomingDetector is a basic keyword-matching detector for development.
// It flags text containing known grooming patterns using heuristic scoring.
// Production systems should use ML-based detection (Comprehend, Perspective API).
type KeywordGroomingDetector struct{}

func NewKeywordGroomingDetector() *KeywordGroomingDetector {
	return &KeywordGroomingDetector{}
}

// groomingPatterns are terms commonly associated with online grooming behavior.
var groomingPatterns = []string{
	"don't tell anyone",
	"our little secret",
	"keep this between us",
	"you're so mature",
	"how old are you",
	"send me a picture",
	"are you home alone",
}

func (d *KeywordGroomingDetector) Analyze(_ context.Context, text string) (*safety.GroomingAnalysisResult, error) {
	normalized := strings.ToLower(text)
	matchCount := 0

	for _, pattern := range groomingPatterns {
		if strings.Contains(normalized, pattern) {
			matchCount++
		}
	}

	score := float64(matchCount) * 0.3 // Each match contributes 0.3 to score, capped at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return &safety.GroomingAnalysisResult{
		Score:        score,
		ModelVersion: "keyword-v1",
		Flagged:      score >= 0.6,
	}, nil
}
