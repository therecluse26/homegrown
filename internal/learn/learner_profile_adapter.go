package learn

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Function Adapter for LearnerProfileForLearn ─────────────────────────────
// Bridges learner_profile:: repository → learn.LearnerProfileForLearn interface.
// Wired via inline closures in main.go (composition root pattern). [ARCH §4.2]

type learnLPFuncAdapter struct {
	getProfile     func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*LearnStudentFitProfile, error)
	getDisplayName func(ctx context.Context, studentID uuid.UUID) (string, error)
}

// NewLearnLearnerProfileAdapter creates a LearnerProfileForLearn from plain function closures.
func NewLearnLearnerProfileAdapter(
	getProfile func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*LearnStudentFitProfile, error),
	getDisplayName func(ctx context.Context, studentID uuid.UUID) (string, error),
) LearnerProfileForLearn {
	return &learnLPFuncAdapter{
		getProfile:     getProfile,
		getDisplayName: getDisplayName,
	}
}

func (a *learnLPFuncAdapter) GetStudentFitProfile(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*LearnStudentFitProfile, error) {
	return a.getProfile(ctx, scope, studentID)
}

func (a *learnLPFuncAdapter) GetStudentDisplayName(ctx context.Context, studentID uuid.UUID) (string, error) {
	return a.getDisplayName(ctx, studentID)
}

// Compile-time interface check.
var _ LearnerProfileForLearn = (*learnLPFuncAdapter)(nil)
