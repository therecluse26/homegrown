package recs

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// learnerProfileAdapter implements LearnerProfilePort using a raw function.
// Avoids circular dependency with learner_profile::. Wired in cmd/server/main.go. [18-learner-profile §7.1]
type learnerProfileAdapter struct {
	getStudentInterestsByFamily func(ctx context.Context, familyID shared.FamilyID) (map[uuid.UUID][]string, error)
}

func (a *learnerProfileAdapter) GetStudentInterestsByFamily(ctx context.Context, familyID shared.FamilyID) (map[uuid.UUID][]string, error) {
	return a.getStudentInterestsByFamily(ctx, familyID)
}

// NewLearnerProfileAdapter creates a LearnerProfilePort adapter from a raw function.
func NewLearnerProfileAdapter(
	getStudentInterestsByFamily func(ctx context.Context, familyID shared.FamilyID) (map[uuid.UUID][]string, error),
) LearnerProfilePort {
	return &learnerProfileAdapter{
		getStudentInterestsByFamily: getStudentInterestsByFamily,
	}
}
