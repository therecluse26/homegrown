package recs

import "errors"

// Sentinel error variables for the Recommendations & Signals domain. [13-recs §15]

var (
	// ErrRecommendationNotFound is returned when a recommendation cannot be found for the family.
	ErrRecommendationNotFound = errors.New("recommendation not found")

	// ErrStudentNotFound is returned when a student ID is not found or does not belong to the family.
	ErrStudentNotFound = errors.New("student not found or does not belong to family")

	// ErrFeedbackNotFound is returned when no feedback exists for a recommendation on undo.
	ErrFeedbackNotFound = errors.New("feedback not found for this recommendation")

	// ErrAlreadyHasFeedback is returned when a recommendation already has a dismiss/block record.
	ErrAlreadyHasFeedback = errors.New("recommendation already has feedback")

	// ErrInvalidRecommendationType is returned when an unknown recommendation type is supplied.
	ErrInvalidRecommendationType = errors.New("invalid recommendation type")

	// ErrInvalidExplorationFrequency is returned when an unknown exploration frequency is supplied.
	ErrInvalidExplorationFrequency = errors.New("invalid exploration frequency")

	// ErrPremiumRequired is returned when a premium-only operation is attempted by a free-tier family.
	ErrPremiumRequired = errors.New("premium subscription required")

	// ErrSignalRecordingFailed is returned when the signal pipeline fails to persist a signal.
	ErrSignalRecordingFailed = errors.New("signal recording failed")

	// ErrDatabaseError is returned for unrecoverable repository errors.
	ErrDatabaseError = errors.New("database error")

	// ErrInternalError is returned for unexpected internal failures.
	ErrInternalError = errors.New("internal error")
)
