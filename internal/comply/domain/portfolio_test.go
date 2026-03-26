package domain

import (
	"errors"
	"testing"
)

// ═══════════════════════════════════════════════════════════════════════════════
// A17–A23: ValidatePortfolioTransition
// ═══════════════════════════════════════════════════════════════════════════════

func TestValidatePortfolioTransition_ConfiguringToGenerating(t *testing.T) {
	if err := ValidatePortfolioTransition("configuring", "generating"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePortfolioTransition_GeneratingToReady(t *testing.T) {
	if err := ValidatePortfolioTransition("generating", "ready"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePortfolioTransition_GeneratingToFailed(t *testing.T) {
	if err := ValidatePortfolioTransition("generating", "failed"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePortfolioTransition_FailedToGenerating(t *testing.T) {
	if err := ValidatePortfolioTransition("failed", "generating"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePortfolioTransition_ReadyToExpired(t *testing.T) {
	if err := ValidatePortfolioTransition("ready", "expired"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePortfolioTransition_ConfiguringToReady_Invalid(t *testing.T) {
	err := ValidatePortfolioTransition("configuring", "ready")
	var transErr *InvalidPortfolioTransitionError
	if !errors.As(err, &transErr) {
		t.Fatalf("got %v, want InvalidPortfolioTransitionError", err)
	}
}

func TestValidatePortfolioTransition_ReadyToConfiguring_Invalid(t *testing.T) {
	err := ValidatePortfolioTransition("ready", "configuring")
	var transErr *InvalidPortfolioTransitionError
	if !errors.As(err, &transErr) {
		t.Fatalf("got %v, want InvalidPortfolioTransitionError", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// A24–A27: ValidatePortfolioGenerate
// ═══════════════════════════════════════════════════════════════════════════════

func TestValidatePortfolioGenerate_RejectsEmptyPortfolio(t *testing.T) {
	err := ValidatePortfolioGenerate("configuring", 0, 0, 3)
	if !errors.Is(err, ErrEmptyPortfolio) {
		t.Fatalf("got %v, want ErrEmptyPortfolio", err)
	}
}

func TestValidatePortfolioGenerate_RejectsNonConfiguring(t *testing.T) {
	err := ValidatePortfolioGenerate("ready", 5, 0, 3)
	if !errors.Is(err, ErrPortfolioNotConfiguring) {
		t.Fatalf("got %v, want ErrPortfolioNotConfiguring", err)
	}
}

func TestValidatePortfolioGenerate_RejectsExceededRetries(t *testing.T) {
	err := ValidatePortfolioGenerate("failed", 5, 3, 3)
	if !errors.Is(err, ErrMaxRetriesExceeded) {
		t.Fatalf("got %v, want ErrMaxRetriesExceeded", err)
	}
}

func TestValidatePortfolioGenerate_AllowsFailedWithRetriesRemaining(t *testing.T) {
	if err := ValidatePortfolioGenerate("failed", 5, 1, 3); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
