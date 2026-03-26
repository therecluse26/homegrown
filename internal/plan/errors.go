package plan

import "errors"

// Sentinel errors for the planning domain. [17-planning §17]
var (
	ErrItemNotFound     = errors.New("schedule item not found")
	ErrTemplateNotFound = errors.New("schedule template not found")

	ErrInvalidDateRange  = errors.New("invalid date range: start must be before end")
	ErrDateRangeTooLarge = errors.New("date range too large (maximum 90 days)")

	ErrAlreadyCompleted = errors.New("schedule item already completed")
	ErrAlreadyLogged    = errors.New("schedule item already logged as activity")

	ErrNotCompleted = errors.New("schedule item must be completed before logging as activity")

	ErrInvalidRecurrenceRule = errors.New("invalid recurrence rule")
	ErrStudentNotInFamily    = errors.New("student not found in family")
)
