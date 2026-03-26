package domain

// Pure portfolio state machine logic. [14-comply §14.2]

// ValidatePortfolioTransition validates a portfolio state transition.
func ValidatePortfolioTransition(currentStatus, targetStatus string) error {
	panic("not implemented")
}

// ValidatePortfolioGenerate validates that a portfolio can be generated.
func ValidatePortfolioGenerate(status string, itemCount int32, retryCount, maxRetries int16) error {
	panic("not implemented")
}
