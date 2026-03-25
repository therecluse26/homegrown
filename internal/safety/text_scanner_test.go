package safety

import (
	"context"
	"testing"
)

func TestTextScanner_clean_text(t *testing.T) {
	s := NewTextScanner(DefaultSafetyConfig())
	result, err := s.Scan(context.Background(), "This is a perfectly normal message about learning math.")
	if err != nil {
		t.Fatal(err)
	}
	if result.HasViolations {
		t.Error("expected no violations")
	}
	if result.Severity != "none" {
		t.Errorf("severity = %s, want none", result.Severity)
	}
}

func TestTextScanner_keyword_match(t *testing.T) {
	s := NewTextScanner(DefaultSafetyConfig())
	result, err := s.Scan(context.Background(), "I am going to kill yourself")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasViolations {
		t.Error("expected violations")
	}
	if len(result.MatchedTerms) == 0 {
		t.Error("expected matched terms")
	}
}

func TestTextScanner_case_insensitive(t *testing.T) {
	s := NewTextScanner(DefaultSafetyConfig())
	result, err := s.Scan(context.Background(), "KILL YOURSELF")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasViolations {
		t.Error("expected case-insensitive match")
	}
}

func TestTextScanner_critical_severity(t *testing.T) {
	s := NewTextScanner(DefaultSafetyConfig())
	result, err := s.Scan(context.Background(), "this is about child exploitation material")
	if err != nil {
		t.Fatal(err)
	}
	if result.Severity != "critical" {
		t.Errorf("severity = %s, want critical", result.Severity)
	}
}

func TestTextScanner_high_severity(t *testing.T) {
	s := NewTextScanner(DefaultSafetyConfig())
	result, err := s.Scan(context.Background(), "I will kill you")
	if err != nil {
		t.Fatal(err)
	}
	if result.Severity != "high" {
		t.Errorf("severity = %s, want high", result.Severity)
	}
}

func TestTextScanner_empty_text(t *testing.T) {
	s := NewTextScanner(DefaultSafetyConfig())
	result, err := s.Scan(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if result.HasViolations {
		t.Error("expected no violations for empty text")
	}
	if result.Severity != "none" {
		t.Errorf("severity = %s, want none", result.Severity)
	}
}
