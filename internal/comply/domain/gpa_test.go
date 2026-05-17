package domain

import (
	"math"
	"testing"
)

func floatPtr(f float64) *float64 { return &f }

const epsilon = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// ═══════════════════════════════════════════════════════════════════════════════
// A28–A34: CalculateGPA
// ═══════════════════════════════════════════════════════════════════════════════

// A28: Single regular course → unweighted = grade_points.
func TestCalculateGPA_SingleCourse_Unweighted(t *testing.T) {
	courses := []CourseForGpa{
		{GradePoints: floatPtr(3.0), Credits: 1.0, Level: "regular"},
	}
	result := CalculateGPA(courses, GpaScaleStandard4, nil)
	if !approxEqual(result.Unweighted, 3.0) {
		t.Fatalf("Unweighted: got %f, want 3.0", result.Unweighted)
	}
	if !approxEqual(result.TotalCredits, 1.0) {
		t.Fatalf("TotalCredits: got %f, want 1.0", result.TotalCredits)
	}
}

// A29: Multiple courses with different credits → weighted average.
func TestCalculateGPA_MultipleCourses_Unweighted(t *testing.T) {
	courses := []CourseForGpa{
		{GradePoints: floatPtr(4.0), Credits: 3.0, Level: "regular"}, // 12.0
		{GradePoints: floatPtr(3.0), Credits: 1.0, Level: "regular"}, //  3.0
	}
	// Expected: (12.0 + 3.0) / (3.0 + 1.0) = 15.0 / 4.0 = 3.75
	result := CalculateGPA(courses, GpaScaleStandard4, nil)
	if !approxEqual(result.Unweighted, 3.75) {
		t.Fatalf("Unweighted: got %f, want 3.75", result.Unweighted)
	}
	if !approxEqual(result.TotalCredits, 4.0) {
		t.Fatalf("TotalCredits: got %f, want 4.0", result.TotalCredits)
	}
}

// A30: Honors course gets +0.5 boost on Weighted only.
func TestCalculateGPA_HonorsBoost(t *testing.T) {
	courses := []CourseForGpa{
		{GradePoints: floatPtr(3.5), Credits: 1.0, Level: "honors"},
	}
	result := CalculateGPA(courses, GpaScaleStandard4, nil)
	// Unweighted = 3.5 (no boost)
	if !approxEqual(result.Unweighted, 3.5) {
		t.Fatalf("Unweighted: got %f, want 3.5", result.Unweighted)
	}
	// Weighted = 3.5 + 0.5 = 4.0
	if !approxEqual(result.Weighted, 4.0) {
		t.Fatalf("Weighted: got %f, want 4.0", result.Weighted)
	}
}

// A31: AP course gets +1.0 boost on Weighted only.
func TestCalculateGPA_APBoost(t *testing.T) {
	courses := []CourseForGpa{
		{GradePoints: floatPtr(3.7), Credits: 1.0, Level: "ap"},
	}
	result := CalculateGPA(courses, GpaScaleStandard4, nil)
	// Unweighted = 3.7
	if !approxEqual(result.Unweighted, 3.7) {
		t.Fatalf("Unweighted: got %f, want 3.7", result.Unweighted)
	}
	// Weighted = 3.7 + 1.0 = 4.7
	if !approxEqual(result.Weighted, 4.7) {
		t.Fatalf("Weighted: got %f, want 4.7", result.Weighted)
	}
}

// A32: Course with nil GradePoints is excluded from calculation.
func TestCalculateGPA_NilGradePointsSkipped(t *testing.T) {
	courses := []CourseForGpa{
		{GradePoints: floatPtr(4.0), Credits: 1.0, Level: "regular"},
		{GradePoints: nil, Credits: 1.0, Level: "regular"}, // skipped
	}
	result := CalculateGPA(courses, GpaScaleStandard4, nil)
	if !approxEqual(result.Unweighted, 4.0) {
		t.Fatalf("Unweighted: got %f, want 4.0", result.Unweighted)
	}
	// Only the non-nil course counts toward credits.
	if !approxEqual(result.TotalCredits, 1.0) {
		t.Fatalf("TotalCredits: got %f, want 1.0", result.TotalCredits)
	}
}

// A33: Empty slice → {0.0, 0.0, 0.0}.
func TestCalculateGPA_ZeroCourses(t *testing.T) {
	result := CalculateGPA([]CourseForGpa{}, GpaScaleStandard4, nil)
	if result.Unweighted != 0.0 {
		t.Fatalf("Unweighted: got %f, want 0.0", result.Unweighted)
	}
	if result.Weighted != 0.0 {
		t.Fatalf("Weighted: got %f, want 0.0", result.Weighted)
	}
	if result.TotalCredits != 0.0 {
		t.Fatalf("TotalCredits: got %f, want 0.0", result.TotalCredits)
	}
}

// A34: Mixed regular/honors/AP → correct unweighted + weighted.
func TestCalculateGPA_MixedLevels(t *testing.T) {
	courses := []CourseForGpa{
		{GradePoints: floatPtr(4.0), Credits: 3.0, Level: "regular"}, // unw: 12.0, w: 12.0
		{GradePoints: floatPtr(3.5), Credits: 3.0, Level: "honors"},  // unw: 10.5, w: (3.5+0.5)*3=12.0
		{GradePoints: floatPtr(3.0), Credits: 4.0, Level: "ap"},      // unw: 12.0, w: (3.0+1.0)*4=16.0
	}
	totalCredits := 3.0 + 3.0 + 4.0 // 10.0
	wantUnweighted := (12.0 + 10.5 + 12.0) / totalCredits // 34.5 / 10.0 = 3.45
	wantWeighted := (12.0 + 12.0 + 16.0) / totalCredits   // 40.0 / 10.0 = 4.0

	result := CalculateGPA(courses, GpaScaleStandard4, nil)
	if !approxEqual(result.Unweighted, wantUnweighted) {
		t.Fatalf("Unweighted: got %f, want %f", result.Unweighted, wantUnweighted)
	}
	if !approxEqual(result.Weighted, wantWeighted) {
		t.Fatalf("Weighted: got %f, want %f", result.Weighted, wantWeighted)
	}
	if !approxEqual(result.TotalCredits, totalCredits) {
		t.Fatalf("TotalCredits: got %f, want %f", result.TotalCredits, totalCredits)
	}
}

// A35: All courses have nil GradePoints → {0.0, 0.0, 0.0} (no gradeable work yet).
func TestCalculateGPA_AllNilGradePoints(t *testing.T) {
	courses := []CourseForGpa{
		{GradePoints: nil, Credits: 1.0, Level: "regular"},
		{GradePoints: nil, Credits: 2.0, Level: "honors"},
	}
	result := CalculateGPA(courses, GpaScaleStandard4, nil)
	if result.Unweighted != 0.0 {
		t.Fatalf("Unweighted: got %f, want 0.0", result.Unweighted)
	}
	if result.TotalCredits != 0.0 {
		t.Fatalf("TotalCredits: got %f, want 0.0", result.TotalCredits)
	}
}

// A36: Course with 0 credits but valid grade points → does not pollute total credits.
func TestCalculateGPA_ZeroCreditsCourseIgnored(t *testing.T) {
	courses := []CourseForGpa{
		{GradePoints: floatPtr(4.0), Credits: 1.0, Level: "regular"},
		{GradePoints: floatPtr(2.0), Credits: 0.0, Level: "regular"}, // 0 credits — contributes 0 points
	}
	// Only the 1-credit course contributes to the average: 4.0 / 1.0 = 4.0
	result := CalculateGPA(courses, GpaScaleStandard4, nil)
	if !approxEqual(result.Unweighted, 4.0) {
		t.Fatalf("Unweighted: got %f, want 4.0", result.Unweighted)
	}
	if !approxEqual(result.TotalCredits, 1.0) {
		t.Fatalf("TotalCredits: got %f, want 1.0", result.TotalCredits)
	}
}

// A37: GpaScaleWeighted passed as scale parameter — output identical since scale is not
// used to gate unweighted vs weighted (both are always computed). Ensures the function
// handles any scale value without panicking.
func TestCalculateGPA_WeightedScaleParamDoesNotPanic(t *testing.T) {
	courses := []CourseForGpa{
		{GradePoints: floatPtr(3.3), Credits: 1.0, Level: "ap"},
	}
	result := CalculateGPA(courses, GpaScaleWeighted, nil)
	if !approxEqual(result.Unweighted, 3.3) {
		t.Fatalf("Unweighted: got %f, want 3.3", result.Unweighted)
	}
	// weighted = 3.3 + 1.0 (AP boost) = 4.3
	if !approxEqual(result.Weighted, 4.3) {
		t.Fatalf("Weighted: got %f, want 4.3", result.Weighted)
	}
}
