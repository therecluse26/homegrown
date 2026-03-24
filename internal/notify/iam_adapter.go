package notify

import (
	"context"

	"github.com/google/uuid"
)

// iamAdapter implements IamServiceForNotify using raw functions.
// Avoids circular dependency between notify:: and iam::. Wired in cmd/server/main.go. [ARCH §4.2]
type iamAdapter struct {
	getFamilyPrimaryEmail func(ctx context.Context, familyID uuid.UUID) (string, string, error)
	getFamilyIDForParent  func(ctx context.Context, parentID uuid.UUID) (uuid.UUID, error)
}

func (a *iamAdapter) GetFamilyPrimaryEmail(ctx context.Context, familyID uuid.UUID) (string, string, error) {
	return a.getFamilyPrimaryEmail(ctx, familyID)
}

func (a *iamAdapter) GetFamilyIDForParent(ctx context.Context, parentID uuid.UUID) (uuid.UUID, error) {
	return a.getFamilyIDForParent(ctx, parentID)
}

// NewIamAdapter creates an IamServiceForNotify adapter from raw functions.
func NewIamAdapter(
	getFamilyPrimaryEmail func(ctx context.Context, familyID uuid.UUID) (string, string, error),
	getFamilyIDForParent func(ctx context.Context, parentID uuid.UUID) (uuid.UUID, error),
) IamServiceForNotify {
	return &iamAdapter{
		getFamilyPrimaryEmail: getFamilyPrimaryEmail,
		getFamilyIDForParent:  getFamilyIDForParent,
	}
}
