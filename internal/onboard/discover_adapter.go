package onboard

import (
	"context"

	"github.com/google/uuid"
)

// discoverAdapter implements DiscoveryServiceForOnboard using raw functions.
// Wired in cmd/server/main.go via NewDiscoverAdapter. [ARCH §4.2]
type discoverAdapter struct {
	getQuizResult  func(ctx context.Context, shareID string) (*OnboardQuizResult, error)
	claimQuizResult func(ctx context.Context, shareID string, familyID uuid.UUID) error
}

func (a *discoverAdapter) GetQuizResult(ctx context.Context, shareID string) (*OnboardQuizResult, error) {
	return a.getQuizResult(ctx, shareID)
}

func (a *discoverAdapter) ClaimQuizResult(ctx context.Context, shareID string, familyID uuid.UUID) error {
	return a.claimQuizResult(ctx, shareID, familyID)
}

// NewDiscoverAdapter creates a DiscoveryServiceForOnboard adapter from raw functions.
func NewDiscoverAdapter(
	getQuizResult func(ctx context.Context, shareID string) (*OnboardQuizResult, error),
	claimQuizResult func(ctx context.Context, shareID string, familyID uuid.UUID) error,
) DiscoveryServiceForOnboard {
	return &discoverAdapter{
		getQuizResult:   getQuizResult,
		claimQuizResult: claimQuizResult,
	}
}
