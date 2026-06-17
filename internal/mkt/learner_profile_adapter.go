package mkt

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Function Adapter for LearnerProfileForMkt ─────────────────────────────
// Bridges learner_profile:: repository → mkt.LearnerProfileForMkt interface.
// Wired via inline closures in main.go (composition root pattern). [ARCH §4.2]

type learnerProfileFuncAdapter struct {
	getProfile      func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*StudentFitProfile, error)
	getDisplayName  func(ctx context.Context, studentID uuid.UUID) (string, error)
}

// NewLearnerProfileAdapter creates a LearnerProfileForMkt from plain function closures.
// This is the canonical wiring pattern for consumer-defined cross-domain interfaces. [ARCH §4.2]
func NewLearnerProfileAdapter(
	getProfile func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*StudentFitProfile, error),
	getDisplayName func(ctx context.Context, studentID uuid.UUID) (string, error),
) LearnerProfileForMkt {
	return &learnerProfileFuncAdapter{
		getProfile:     getProfile,
		getDisplayName: getDisplayName,
	}
}

func (a *learnerProfileFuncAdapter) GetStudentFitProfile(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*StudentFitProfile, error) {
	return a.getProfile(ctx, scope, studentID)
}

func (a *learnerProfileFuncAdapter) GetStudentDisplayName(ctx context.Context, studentID uuid.UUID) (string, error) {
	return a.getDisplayName(ctx, studentID)
}

// Compile-time interface check.
var _ LearnerProfileForMkt = (*learnerProfileFuncAdapter)(nil)
