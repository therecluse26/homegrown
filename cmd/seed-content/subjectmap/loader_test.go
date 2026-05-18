package subjectmap_test

import (
	"slices"
	"testing"

	"github.com/homegrown-academy/homegrown-academy/cmd/seed-content/subjectmap"
)

func TestMapSubjects_KnownSubjects(t *testing.T) {
	cases := []struct {
		input []string
		want  []string
	}{
		{
			input: []string{"Mathematics"},
			want:  []string{"math"},
		},
		{
			input: []string{"Juvenile Fiction"},
			want:  []string{"language-arts"},
		},
		{
			input: []string{"Natural history"},
			want:  []string{"science"},
		},
		{
			input: []string{"History"},
			want:  []string{"history"},
		},
		{
			// Multi-slug mapping: Biology maps to ["science", "science.biology"]
			input: []string{"Biology"},
			want:  []string{"science", "science.biology"},
		},
	}

	for _, tc := range cases {
		got := subjectmap.MapSubjects(tc.input)
		if !equalSlices(got, tc.want) {
			t.Errorf("MapSubjects(%v) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestMapSubjects_UnknownSubjectReturnsNil(t *testing.T) {
	got := subjectmap.MapSubjects([]string{"Underwater Basket Weaving"})
	if len(got) != 0 {
		t.Errorf("expected empty result for unknown subject, got %v", got)
	}
}

func TestMapSubjects_MixedKnownAndUnknown(t *testing.T) {
	got := subjectmap.MapSubjects([]string{"Mathematics", "Underwater Basket Weaving", "Biology"})
	// Should contain math, science, science.biology — in that order, unknown skipped
	if len(got) == 0 {
		t.Fatal("expected non-empty result")
	}
	if !contains(got, "math") {
		t.Errorf("expected 'math' in result %v", got)
	}
	if !contains(got, "science") {
		t.Errorf("expected 'science' in result %v", got)
	}
	if !contains(got, "science.biology") {
		t.Errorf("expected 'science.biology' in result %v", got)
	}
}

func TestMapSubjects_DeduplicatesOverlappingSlugs(t *testing.T) {
	// Both "Biology" and "Botany" map to "science" (among others).
	// "science" must appear only once in the result.
	got := subjectmap.MapSubjects([]string{"Biology", "Botany"})
	count := 0
	for _, s := range got {
		if s == "science" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 'science' exactly once, got %d times in %v", count, got)
	}
}

func TestMapSubjects_EmptyInput(t *testing.T) {
	got := subjectmap.MapSubjects([]string{})
	if len(got) != 0 {
		t.Errorf("expected empty result for empty input, got %v", got)
	}
}

func TestMapSubjects_NilInput(t *testing.T) {
	got := subjectmap.MapSubjects(nil)
	if len(got) != 0 {
		t.Errorf("expected empty result for nil input, got %v", got)
	}
}

// equalSlices reports whether a and b contain the same elements in the same order.
func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func contains(slice []string, s string) bool {
	return slices.Contains(slice, s)
}
