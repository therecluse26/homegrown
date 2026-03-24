package domain

import (
	"time"

	"github.com/google/uuid"
)

// Activity is the aggregate root for activity logging — enforces invariants.
// [06-learn §11.2, ARCH §4.5]
//
// Invariants:
//  1. Duration must be non-negative (if provided).
//  2. Activity date cannot be in the future.
type Activity struct {
	studentID       uuid.UUID
	title           string
	subjectTags     []string
	activityDate    time.Time
	durationMinutes *int16
}

// NewActivity creates a validated Activity, enforcing domain invariants.
func NewActivity(
	studentID uuid.UUID,
	title string,
	subjectTags []string,
	activityDate time.Time,
	durationMinutes *int16,
) (*Activity, error) {
	if durationMinutes != nil && *durationMinutes < 0 {
		return nil, ErrNegativeDuration
	}
	// Compare date-only (truncate to start of day) to handle timezone edge cases.
	today := time.Now().Truncate(24 * time.Hour)
	actDate := activityDate.Truncate(24 * time.Hour)
	if actDate.After(today) {
		return nil, ErrFutureDateNotAllowed
	}
	return &Activity{
		studentID:       studentID,
		title:           title,
		subjectTags:     subjectTags,
		activityDate:    activityDate,
		durationMinutes: durationMinutes,
	}, nil
}

// StudentID returns the student ID.
func (a *Activity) StudentID() uuid.UUID { return a.studentID }

// Title returns the title.
func (a *Activity) Title() string { return a.title }

// SubjectTags returns the subject tags.
func (a *Activity) SubjectTags() []string { return a.subjectTags }

// ActivityDate returns the activity date.
func (a *Activity) ActivityDate() time.Time { return a.activityDate }

// DurationMinutes returns the duration in minutes.
func (a *Activity) DurationMinutes() *int16 { return a.durationMinutes }
