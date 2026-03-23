package method

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// iamAdapter defines the raw IAM service methods that method:: needs.
// The adapter pattern allows method:: to depend on iam:: without importing the iam package
// directly. Wired in cmd/server/main.go. [ARCH §4.2]
type iamAdapter struct {
	getFamilyMethodologyIDs func(ctx context.Context, scope *shared.FamilyScope) (MethodologyID, []MethodologyID, error)
	getStudent              func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*StudentInfo, error)
	setFamilyMethodology    func(ctx context.Context, scope *shared.FamilyScope, primarySlug MethodologyID, secondarySlugs []MethodologyID) error
}

func (a *iamAdapter) GetFamilyMethodologyIDs(ctx context.Context, scope *shared.FamilyScope) (MethodologyID, []MethodologyID, error) {
	return a.getFamilyMethodologyIDs(ctx, scope)
}

func (a *iamAdapter) GetStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*StudentInfo, error) {
	return a.getStudent(ctx, scope, studentID)
}

func (a *iamAdapter) SetFamilyMethodology(ctx context.Context, scope *shared.FamilyScope, primarySlug MethodologyID, secondarySlugs []MethodologyID) error {
	return a.setFamilyMethodology(ctx, scope, primarySlug, secondarySlugs)
}

// NewIamAdapter creates an IamServiceForMethod adapter from raw functions.
// This is the wiring point used in cmd/server/main.go.
func NewIamAdapter(
	getFamilyMethodologyIDs func(ctx context.Context, scope *shared.FamilyScope) (MethodologyID, []MethodologyID, error),
	getStudent func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*StudentInfo, error),
	setFamilyMethodology func(ctx context.Context, scope *shared.FamilyScope, primarySlug MethodologyID, secondarySlugs []MethodologyID) error,
) IamServiceForMethod {
	return &iamAdapter{
		getFamilyMethodologyIDs: getFamilyMethodologyIDs,
		getStudent:              getStudent,
		setFamilyMethodology:    setFamilyMethodology,
	}
}
