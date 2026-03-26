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

// validAttendanceStatuses contains the allowed status strings.
var validAttendanceStatuses = map[string]bool{
	"present_full":    true,
	"present_partial": true,
	"absent":          true,
	"not_applicable":  true,
}

// ValidateAttendanceRecord validates a new attendance record.
func ValidateAttendanceRecord(
	date time.Time,
	status string,
	durationMinutes *int16,
	today time.Time,
) error {
	if date.After(today) {
		return ErrFutureAttendanceDate
	}
	if !validAttendanceStatuses[status] {
		return ErrInvalidAttendanceStatus
	}
	if durationMinutes != nil && *durationMinutes < 0 {
		return ErrNegativeDuration
	}
	if status == "present_partial" && durationMinutes == nil {
		return ErrDurationRequiredForPartial
	}
	return nil
}

// ShouldOverride determines precedence: manual entries override auto-generated ones.
// Manual always wins. Auto never overrides manual.
// existingIsAuto=true means the existing record is auto-generated.
// newIsManual=true means the new record is a manual entry.
func ShouldOverride(existingIsAuto bool, newIsManual bool) bool {
	// If the new entry is manual, it always overrides.
	// If the new entry is auto, it only overrides if the existing is also auto.
	return newIsManual || existingIsAuto
}

// CalculatePace calculates attendance pace against state requirements.
func CalculatePace(
	actualPresentDays int32,
	elapsedSchoolDays int32,
	totalSchoolDays int32,
	stateRequiredDays *int16,
) PaceStatus {
	if stateRequiredDays == nil {
		return PaceStatusNotApplicable
	}
	if elapsedSchoolDays == 0 {
		return PaceStatusOnTrack
	}

	required := float64(*stateRequiredDays)
	// Project total present days based on current pace
	projected := float64(actualPresentDays) * float64(totalSchoolDays) / float64(elapsedSchoolDays)
	ratio := projected / required

	switch {
	case ratio >= 1.0:
		return PaceStatusOnTrack
	case ratio >= 0.9:
		return PaceStatusAtRisk
	default:
		return PaceStatusBehind
	}
}

// CountSchoolDays counts school days between two dates (inclusive) using a custom schedule.
// schoolDays is indexed by time.Weekday: [Sun=0, Mon=1, ..., Sat=6].
func CountSchoolDays(
	start time.Time,
	end time.Time,
	schoolDays [7]bool,
	exclusionPeriods []ExclusionPeriod,
) int32 {
	if end.Before(start) {
		return 0
	}

	var count int32
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if !schoolDays[d.Weekday()] {
			continue
		}
		excluded := false
		for _, ep := range exclusionPeriods {
			if !d.Before(ep.Start) && !d.After(ep.End) {
				excluded = true
				break
			}
		}
		if !excluded {
			count++
		}
	}
	return count
}
