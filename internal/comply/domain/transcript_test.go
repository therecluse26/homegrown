package domain

import (
	"errors"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// A35: ValidateTranscriptTransition delegates to ValidatePortfolioTransition
// ═══════════════════════════════════════════════════════════════════════════════

func TestValidateTranscriptTransition_DelegatesToPortfolio(t *testing.T) {
	// Valid transition succeeds (same as portfolio).
	if err := ValidateTranscriptTransition("configuring", "generating"); err != nil {
		t.Fatalf("valid transition failed: %v", err)
	}

	// Invalid transition returns InvalidPortfolioTransitionError (same as portfolio).
	err := ValidateTranscriptTransition("configuring", "ready")
	var transErr *InvalidPortfolioTransitionError
	if !errors.As(err, &transErr) {
		t.Fatalf("got %v, want InvalidPortfolioTransitionError", err)
	}
}
