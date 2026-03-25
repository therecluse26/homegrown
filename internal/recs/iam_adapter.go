package recs

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// iamAdapter implements IamServiceForRecs using raw functions.
// Avoids circular dependency with iam::. Wired in cmd/server/main.go. [ARCH §4.2]
type iamAdapter struct {
	studentBelongsToFamily     func(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error)
	getFamilyMethodologySlug   func(ctx context.Context, familyID shared.FamilyID) (string, error)
}

func (a *iamAdapter) StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error) {
	return a.studentBelongsToFamily(ctx, studentID, familyID)
}

func (a *iamAdapter) GetFamilyMethodologySlug(ctx context.Context, familyID shared.FamilyID) (string, error) {
	return a.getFamilyMethodologySlug(ctx, familyID)
}

// NewIamAdapter creates an IamServiceForRecs adapter from raw functions.
func NewIamAdapter(
	studentBelongsToFamily func(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error),
	getFamilyMethodologySlug func(ctx context.Context, familyID shared.FamilyID) (string, error),
) IamServiceForRecs {
	return &iamAdapter{
		studentBelongsToFamily:   studentBelongsToFamily,
		getFamilyMethodologySlug: getFamilyMethodologySlug,
	}
}
