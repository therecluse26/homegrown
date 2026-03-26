package domain

// Pure portfolio state machine logic. [14-comply §14.2]

// allowedTransitions maps current status → set of valid target statuses.
var allowedTransitions = map[string]map[string]bool{
	"configuring": {"generating": true},
	"generating":  {"ready": true, "failed": true},
	"failed":      {"generating": true},
	"ready":       {"expired": true},
}

// ValidatePortfolioTransition validates a portfolio state transition.
func ValidatePortfolioTransition(currentStatus, targetStatus string) error {
	targets, ok := allowedTransitions[currentStatus]
	if !ok || !targets[targetStatus] {
		return &InvalidPortfolioTransitionError{From: currentStatus, To: targetStatus}
	}
	return nil
}

// ValidatePortfolioGenerate validates that a portfolio can be generated.
func ValidatePortfolioGenerate(status string, itemCount int32, retryCount, maxRetries int16) error {
	if itemCount == 0 {
		return ErrEmptyPortfolio
	}
	if status == "failed" {
		if retryCount >= maxRetries {
			return ErrMaxRetriesExceeded
		}
		return nil
	}
	if status != "configuring" {
		return ErrPortfolioNotConfiguring
	}
	return nil
}
