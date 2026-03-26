package domain

import "encoding/json"

// Pure GPA calculation logic. [14-comply §10, §14.4]

// GpaScale represents the GPA calculation method.
type GpaScale string

const (
	GpaScaleStandard4 GpaScale = "standard_4"
	GpaScaleWeighted  GpaScale = "weighted"
	GpaScaleCustom    GpaScale = "custom"
)

// GpaResult holds the computed GPA values.
type GpaResult struct {
	Unweighted   float64
	Weighted     float64
	TotalCredits float64
}

// CourseForGpa holds minimal course data needed for GPA calculation.
// Separate from ComplyCourse to avoid circular import with parent package.
type CourseForGpa struct {
	GradePoints *float64
	Credits     float64
	Level       string // "regular", "honors", "ap"
}

// CalculateGPA calculates GPA from a list of courses.
func CalculateGPA(courses []CourseForGpa, scale GpaScale, customConfig json.RawMessage) GpaResult {
	panic("not implemented")
}
