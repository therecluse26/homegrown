package domain

import "time"

// DefaultDateRange returns a date range with sensible defaults. [06-learn §14]
// If dateFrom is nil, defaults to 1 month ago (start of day).
// If dateTo is nil, defaults to end of today.
func DefaultDateRange(dateFrom, dateTo *time.Time) (time.Time, time.Time) {
	now := time.Now()
	from := now.AddDate(0, -1, 0).Truncate(24 * time.Hour)
	to := now.Truncate(24 * time.Hour).Add(24*time.Hour - time.Nanosecond)
	if dateFrom != nil {
		from = *dateFrom
	}
	if dateTo != nil {
		to = *dateTo
	}
	return from, to
}
