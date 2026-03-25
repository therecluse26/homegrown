package safety

import (
	"context"
	"strings"
)

// TextScanner performs synchronous text scanning. [11-safety §11.1]
// Phase 1: keyword matching (case-insensitive).
// Phase 2: AWS Comprehend for ML-based detection.
type TextScanner struct {
	config SafetyConfig
}

// NewTextScanner creates a new TextScanner.
func NewTextScanner(config SafetyConfig) *TextScanner {
	return &TextScanner{config: config}
}

// criticalTerms are child safety related terms. [11-safety §11.1]
var criticalTerms = []string{
	"csam", "child porn", "child exploitation",
	"minor sexual", "underage sexual",
}

// highTerms are harassment-related terms. [11-safety §11.1]
var highTerms = []string{
	"kill yourself", "kys", "i will kill you",
	"death threat",
}

// Scan checks text against keyword lists and returns violations. [11-safety §11.1]
func (s *TextScanner) Scan(_ context.Context, text string) (*TextScanResult, error) {
	if text == "" {
		return &TextScanResult{Severity: "none"}, nil
	}

	normalized := strings.ToLower(text)
	var matchedTerms []string
	severity := "none"

	for _, term := range criticalTerms {
		if strings.Contains(normalized, term) {
			matchedTerms = append(matchedTerms, term)
			severity = "critical"
		}
	}

	for _, term := range highTerms {
		if strings.Contains(normalized, term) {
			matchedTerms = append(matchedTerms, term)
			if severity != "critical" {
				severity = "high"
			}
		}
	}

	return &TextScanResult{
		HasViolations: len(matchedTerms) > 0,
		MatchedTerms:  matchedTerms,
		Severity:      severity,
	}, nil
}
