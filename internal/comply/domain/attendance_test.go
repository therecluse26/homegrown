package domain

import (
	"errors"
	"testing"
	"time"
)

// ═══════════════════════════════════════════════════════════════════════════════
// A1–A5: ValidateAttendanceRecord
// ═══════════════════════════════════════════════════════════════════════════════

func TestValidateAttendanceRecord_RejectsFutureDate(t *testing.T) {
	today := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	future := today.AddDate(0, 0, 1)
	err := ValidateAttendanceRecord(future, "present_full", nil, today)
	if !errors.Is(err, ErrFutureAttendanceDate) {
		t.Fatalf("got %v, want ErrFutureAttendanceDate", err)
	}
}

func TestValidateAttendanceRecord_RejectsInvalidStatus(t *testing.T) {
	today := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	err := ValidateAttendanceRecord(today, "sleeping", nil, today)
	if !errors.Is(err, ErrInvalidAttendanceStatus) {
		t.Fatalf("got %v, want ErrInvalidAttendanceStatus", err)
	}
}

func TestValidateAttendanceRecord_RequiresDurationForPartial(t *testing.T) {
	today := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	err := ValidateAttendanceRecord(today, "present_partial", nil, today)
	if !errors.Is(err, ErrDurationRequiredForPartial) {
		t.Fatalf("got %v, want ErrDurationRequiredForPartial", err)
	}
}

func TestValidateAttendanceRecord_RejectsNegativeDuration(t *testing.T) {
	today := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	neg := int16(-10)
	err := ValidateAttendanceRecord(today, "present_partial", &neg, today)
	if !errors.Is(err, ErrNegativeDuration) {
		t.Fatalf("got %v, want ErrNegativeDuration", err)
	}
}

func TestValidateAttendanceRecord_AcceptsAllValidStatuses(t *testing.T) {
	today := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	dur := int16(120)

	statuses := []struct {
		status   string
		duration *int16
	}{
		{"present_full", nil},
		{"present_partial", &dur},
		{"absent", nil},
		{"not_applicable", nil},
	}
	for _, tc := range statuses {
		if err := ValidateAttendanceRecord(today, tc.status, tc.duration, today); err != nil {
			t.Errorf("status %q: unexpected error: %v", tc.status, err)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// A6–A7: ShouldOverride
// ═══════════════════════════════════════════════════════════════════════════════

func TestShouldOverride_ManualOverridesAuto(t *testing.T) {
	if !ShouldOverride(true, true) {
		t.Fatal("manual should override auto")
	}
}

func TestShouldOverride_AutoDoesNotOverrideManual(t *testing.T) {
	if ShouldOverride(false, false) {
		t.Fatal("auto should NOT override manual")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// A8–A12: CalculatePace
// ═══════════════════════════════════════════════════════════════════════════════

func TestCalculatePace_NilRequired_NotApplicable(t *testing.T) {
	got := CalculatePace(50, 100, 180, nil)
	if got != PaceStatusNotApplicable {
		t.Fatalf("got %v, want not_applicable", got)
	}
}

func TestCalculatePace_ZeroElapsed_OnTrack(t *testing.T) {
	req := int16(180)
	got := CalculatePace(0, 0, 180, &req)
	if got != PaceStatusOnTrack {
		t.Fatalf("got %v, want on_track", got)
	}
}

func TestCalculatePace_ProjectedMeetsRequired_OnTrack(t *testing.T) {
	// 50 present in 100 elapsed out of 180 total → projected = 50 * 180/100 = 90
	// required = 90 → projected >= required → on_track
	req := int16(90)
	got := CalculatePace(50, 100, 180, &req)
	if got != PaceStatusOnTrack {
		t.Fatalf("got %v, want on_track", got)
	}
}

func TestCalculatePace_ProjectedAtRisk(t *testing.T) {
	// 45 present in 100 elapsed out of 180 total → projected = 45*180/100 = 81
	// required = 90 → 81/90 = 0.9 → at_risk (90-100%)
	req := int16(90)
	got := CalculatePace(45, 100, 180, &req)
	if got != PaceStatusAtRisk {
		t.Fatalf("got %v, want at_risk", got)
	}
}

func TestCalculatePace_ProjectedBehind(t *testing.T) {
	// 40 present in 100 elapsed out of 180 total → projected = 40*180/100 = 72
	// required = 90 → 72/90 = 0.8 → behind (<90%)
	req := int16(90)
	got := CalculatePace(40, 100, 180, &req)
	if got != PaceStatusBehind {
		t.Fatalf("got %v, want behind", got)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// A13–A16: CountSchoolDays
// ═══════════════════════════════════════════════════════════════════════════════

func TestCountSchoolDays_StandardMonFri(t *testing.T) {
	// Mon–Fri schedule: [Mon=true, Tue=true, Wed=true, Thu=true, Fri=true, Sat=false, Sun=false]
	// time.Weekday: Sun=0, Mon=1, ..., Sat=6
	// schoolDays index: [Sun, Mon, Tue, Wed, Thu, Fri, Sat]
	schedule := [7]bool{false, true, true, true, true, true, false}
	// 2025-03-03 is a Monday → 2025-03-07 is Friday → 5 weekdays
	start := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 7, 0, 0, 0, 0, time.UTC)
	got := CountSchoolDays(start, end, schedule, nil)
	if got != 5 {
		t.Fatalf("got %d, want 5", got)
	}
}

func TestCountSchoolDays_FourDayWeek(t *testing.T) {
	// Mon–Thu only
	schedule := [7]bool{false, true, true, true, true, false, false}
	start := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 7, 0, 0, 0, 0, time.UTC)
	got := CountSchoolDays(start, end, schedule, nil)
	if got != 4 {
		t.Fatalf("got %d, want 4", got)
	}
}

func TestCountSchoolDays_ExcludesExclusionPeriods(t *testing.T) {
	schedule := [7]bool{false, true, true, true, true, true, false}
	// Two weeks: 10 weekdays
	start := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 14, 0, 0, 0, 0, time.UTC)
	// Exclude Wed-Fri of first week (3 days)
	exclusions := []ExclusionPeriod{
		{
			Start: time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2025, 3, 7, 0, 0, 0, 0, time.UTC),
			Label: "Break",
		},
	}
	got := CountSchoolDays(start, end, schedule, exclusions)
	if got != 7 {
		t.Fatalf("got %d, want 7 (10 weekdays - 3 excluded)", got)
	}
}

func TestCountSchoolDays_EmptyRange(t *testing.T) {
	schedule := [7]bool{false, true, true, true, true, true, false}
	start := time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 5, 0, 0, 0, 0, time.UTC)
	got := CountSchoolDays(start, end, schedule, nil)
	if got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}
