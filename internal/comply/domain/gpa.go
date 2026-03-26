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

// CalculateGPA calculates GPA from a list of courses. [14-comply §10]
// Courses with nil GradePoints are skipped. Zero courses returns {0,0,0}.
// Weighted GPA applies boosts: honors +0.5, AP +1.0.
func CalculateGPA(courses []CourseForGpa, scale GpaScale, customConfig json.RawMessage) GpaResult {
	var totalUnweightedPoints float64
	var totalWeightedPoints float64
	var totalCredits float64

	for _, c := range courses {
		if c.GradePoints == nil {
			continue
		}
		gp := *c.GradePoints
		credits := c.Credits
		totalUnweightedPoints += gp * credits

		var boost float64
		switch c.Level {
		case "honors":
			boost = 0.5
		case "ap":
			boost = 1.0
		}
		totalWeightedPoints += (gp + boost) * credits
		totalCredits += credits
	}

	if totalCredits == 0.0 {
		return GpaResult{}
	}

	return GpaResult{
		Unweighted:   totalUnweightedPoints / totalCredits,
		Weighted:     totalWeightedPoints / totalCredits,
		TotalCredits: totalCredits,
	}
}
