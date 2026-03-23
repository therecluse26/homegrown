package onboard

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// iamAdapter implements IamServiceForOnboard using raw functions.
// The adapter pattern allows onboard:: to consume iam:: without importing the iam package
// directly, avoiding circular dependencies. Wired in cmd/server/main.go. [ARCH §4.2]
type iamAdapter struct {
	updateFamilyProfile func(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyProfileCommand) error
	createStudent       func(ctx context.Context, scope *shared.FamilyScope, cmd AddChildCommand) error
	deleteStudent       func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error
	listStudents        func(ctx context.Context, familyID uuid.UUID) ([]OnboardStudentInfo, error)
}

func (a *iamAdapter) UpdateFamilyProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyProfileCommand) error {
	return a.updateFamilyProfile(ctx, scope, cmd)
}

func (a *iamAdapter) CreateStudent(ctx context.Context, scope *shared.FamilyScope, cmd AddChildCommand) error {
	return a.createStudent(ctx, scope, cmd)
}

func (a *iamAdapter) DeleteStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error {
	return a.deleteStudent(ctx, scope, studentID)
}

func (a *iamAdapter) ListStudents(ctx context.Context, familyID uuid.UUID) ([]OnboardStudentInfo, error) {
	return a.listStudents(ctx, familyID)
}

// NewIamAdapter creates an IamServiceForOnboard adapter from raw functions.
func NewIamAdapter(
	updateFamilyProfile func(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyProfileCommand) error,
	createStudent func(ctx context.Context, scope *shared.FamilyScope, cmd AddChildCommand) error,
	deleteStudent func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error,
	listStudents func(ctx context.Context, familyID uuid.UUID) ([]OnboardStudentInfo, error),
) IamServiceForOnboard {
	return &iamAdapter{
		updateFamilyProfile: updateFamilyProfile,
		createStudent:       createStudent,
		deleteStudent:       deleteStudent,
		listStudents:        listStudents,
	}
}
