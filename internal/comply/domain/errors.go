package domain

import (
	"errors"
	"fmt"
)

// Domain-layer error sentinels used by pure functions in domain/.
// These are separate from service-layer errors (comply/errors.go) to avoid
// circular imports — domain/ cannot import parent comply package. [14-comply §16]

var (
	// ErrFutureAttendanceDate is returned when attendance is recorded for a future date.
	ErrFutureAttendanceDate = errors.New("cannot record attendance for a future date")

	// ErrInvalidAttendanceStatus is returned for an unrecognized attendance status.
	ErrInvalidAttendanceStatus = errors.New("invalid attendance status")

	// ErrDurationRequiredForPartial is returned when present_partial has no duration.
	ErrDurationRequiredForPartial = errors.New("duration is required for partial attendance")

	// ErrNegativeDuration is returned when duration_minutes is negative.
	ErrNegativeDuration = errors.New("duration cannot be negative")

	// ErrPortfolioNotConfiguring is returned when a portfolio is not in configuring/failed status.
	ErrPortfolioNotConfiguring = errors.New("portfolio is not in configuring status")

	// ErrEmptyPortfolio is returned when trying to generate a portfolio with no items.
	ErrEmptyPortfolio = errors.New("cannot generate an empty portfolio")

	// ErrMaxRetriesExceeded is returned when max generation retries have been exhausted.
	ErrMaxRetriesExceeded = errors.New("maximum retry attempts exceeded")
)

// InvalidPortfolioTransitionError represents an invalid portfolio/transcript status transition.
type InvalidPortfolioTransitionError struct {
	From string
	To   string
}

func (e *InvalidPortfolioTransitionError) Error() string {
	return fmt.Sprintf("invalid portfolio status transition from %s to %s", e.From, e.To)
}
