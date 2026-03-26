package domain

// Transcript state machine. [14-comply §14.3]
// Delegates to portfolio since transitions are identical.

// ValidateTranscriptTransition validates a transcript state transition.
// Same rules as portfolio — delegates to ValidatePortfolioTransition.
func ValidateTranscriptTransition(currentStatus, targetStatus string) error {
	return ValidatePortfolioTransition(currentStatus, targetStatus)
}
