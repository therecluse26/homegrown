package adapters

import (
	"testing"

	"github.com/homegrown-academy/homegrown-academy/internal/safety"
)

func defaultConfig() *safety.SafetyConfig {
	cfg := safety.DefaultSafetyConfig()
	return &cfg
}

func TestApplyLabelRouting_nudity_auto_reject(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Explicit Nudity", Confidence: 85.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if !result.AutoReject {
		t.Error("expected auto-reject for nudity at 85%")
	}
	if !result.HasViolations {
		t.Error("expected violations")
	}
}

func TestApplyLabelRouting_nudity_below_threshold(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Nudity", Confidence: 60.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if result.AutoReject {
		t.Error("expected no auto-reject for nudity at 60%")
	}
}

func TestApplyLabelRouting_suggestive_above_80(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Suggestive", Confidence: 85.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if !result.HasViolations {
		t.Error("expected violations for suggestive at 85%")
	}
	if result.AutoReject {
		t.Error("suggestive should not auto-reject")
	}
	if result.Priority == nil || *result.Priority != "normal" {
		t.Errorf("priority = %v, want normal", result.Priority)
	}
}

func TestApplyLabelRouting_suggestive_below_80(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Suggestive", Confidence: 70.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if result.HasViolations {
		t.Error("expected no violations for suggestive at 70%")
	}
}

func TestApplyLabelRouting_violence_flagged(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Violence", Confidence: 75.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if !result.HasViolations {
		t.Error("expected violations for violence at 75%")
	}
	if result.Priority == nil || *result.Priority != "normal" {
		t.Errorf("priority = %v, want normal", result.Priority)
	}
}

func TestApplyLabelRouting_drugs_ignored(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Drugs", Confidence: 99.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if result.HasViolations {
		t.Error("drugs should be ignored")
	}
}

func TestApplyLabelRouting_hate_ignored(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Hate Symbols", Confidence: 99.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if result.HasViolations {
		t.Error("hate symbols should be ignored")
	}
}

func TestApplyLabelRouting_weapons_ignored(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Weapons", Confidence: 99.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if result.HasViolations {
		t.Error("weapons should be ignored")
	}
}

func TestApplyLabelRouting_underage_suggestive_critical(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Minor", Confidence: 60.0},
		{Name: "Suggestive", Confidence: 85.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if !result.HasViolations {
		t.Error("expected violations")
	}
	if result.Priority == nil || *result.Priority != "critical" {
		t.Errorf("priority = %v, want critical", result.Priority)
	}
}

func TestApplyLabelRouting_no_labels(t *testing.T) {
	result := ApplyLabelRouting(nil, defaultConfig())
	if result.HasViolations {
		t.Error("expected no violations for empty labels")
	}
	if result.AutoReject {
		t.Error("expected no auto-reject for empty labels")
	}
}

func TestApplyLabelRouting_mixed_labels(t *testing.T) {
	labels := []safety.ModerationLabel{
		{Name: "Drugs", Confidence: 95.0},
		{Name: "Violence", Confidence: 80.0},
		{Name: "Weapons", Confidence: 90.0},
	}
	result := ApplyLabelRouting(labels, defaultConfig())
	if !result.HasViolations {
		t.Error("expected violations from violence label")
	}
	if len(result.Labels) != 1 {
		t.Errorf("kept labels = %d, want 1 (only violence)", len(result.Labels))
	}
}
