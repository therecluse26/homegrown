package domain

import "time"

// Pure attendance domain logic. [14-comply §14.1, §12]

// PaceStatus represents attendance pace relative to state requirements.
type PaceStatus string

const (
	PaceStatusOnTrack       PaceStatus = "on_track"
	PaceStatusAtRisk        PaceStatus = "at_risk"
	PaceStatusBehind        PaceStatus = "behind"
	PaceStatusNotApplicable PaceStatus = "not_applicable"
)

// ExclusionPeriod represents a date range excluded from school days (e.g. winter break).
// Plain struct — no JSON tags. The models.go DTO has JSON tags.
type ExclusionPeriod struct {
	Start time.Time
	End   time.Time
	Label string
}

// ValidateAttendanceRecord validates a new attendance record.
func ValidateAttendanceRecord(
	date time.Time,
	status string,
	durationMinutes *int16,
	today time.Time,
) error {
	panic("not implemented")
}

// ShouldOverride determines precedence: manual entries override auto-generated ones.
// Manual always wins. Auto never overrides manual.
func ShouldOverride(existingIsAuto bool, newIsManual bool) bool {
	panic("not implemented")
}

// CalculatePace calculates attendance pace against state requirements.
func CalculatePace(
	actualPresentDays int32,
	elapsedSchoolDays int32,
	totalSchoolDays int32,
	stateRequiredDays *int16,
) PaceStatus {
	panic("not implemented")
}

// CountSchoolDays counts school days between two dates using a custom schedule.
func CountSchoolDays(
	start time.Time,
	end time.Time,
	schoolDays [7]bool,
	exclusionPeriods []ExclusionPeriod,
) int32 {
	panic("not implemented")
}
