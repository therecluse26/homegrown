package social

import (
	"context"

	"github.com/google/uuid"
)

// iamAdapter implements IamServiceForSocial using raw functions.
// The adapter pattern allows social:: to consume iam:: without importing the iam package
// directly, avoiding circular dependencies. Wired in cmd/server/main.go. [ARCH §4.2]
type iamAdapter struct {
	getFamilyDisplayName func(ctx context.Context, familyID uuid.UUID) (string, error)
	getParentDisplayName func(ctx context.Context, parentID uuid.UUID) (string, error)
	getFamilyInfo        func(ctx context.Context, familyID uuid.UUID) (*SocialFamilyInfo, error)
	getParentInfo        func(ctx context.Context, parentID uuid.UUID) (*SocialParentInfo, error)
}

func (a *iamAdapter) GetFamilyDisplayName(ctx context.Context, familyID uuid.UUID) (string, error) {
	return a.getFamilyDisplayName(ctx, familyID)
}

func (a *iamAdapter) GetParentDisplayName(ctx context.Context, parentID uuid.UUID) (string, error) {
	return a.getParentDisplayName(ctx, parentID)
}

func (a *iamAdapter) GetFamilyInfo(ctx context.Context, familyID uuid.UUID) (*SocialFamilyInfo, error) {
	return a.getFamilyInfo(ctx, familyID)
}

func (a *iamAdapter) GetParentInfo(ctx context.Context, parentID uuid.UUID) (*SocialParentInfo, error) {
	return a.getParentInfo(ctx, parentID)
}

// NewIamAdapter creates an IamServiceForSocial adapter from raw functions.
func NewIamAdapter(
	getFamilyDisplayName func(ctx context.Context, familyID uuid.UUID) (string, error),
	getParentDisplayName func(ctx context.Context, parentID uuid.UUID) (string, error),
	getFamilyInfo func(ctx context.Context, familyID uuid.UUID) (*SocialFamilyInfo, error),
	getParentInfo func(ctx context.Context, parentID uuid.UUID) (*SocialParentInfo, error),
) IamServiceForSocial {
	return &iamAdapter{
		getFamilyDisplayName: getFamilyDisplayName,
		getParentDisplayName: getParentDisplayName,
		getFamilyInfo:        getFamilyInfo,
		getParentInfo:        getParentInfo,
	}
}
