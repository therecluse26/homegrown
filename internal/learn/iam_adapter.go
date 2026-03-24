package learn

import (
	"context"

	"github.com/google/uuid"
)

// iamAdapter implements IamServiceForLearn using raw functions.
// The adapter pattern allows learn:: to consume iam:: without importing the iam package
// directly, avoiding circular dependencies. Wired in cmd/server/main.go. [ARCH §4.2]
type iamAdapter struct {
	studentBelongsToFamily func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error)
	getStudentName         func(ctx context.Context, studentID uuid.UUID) (string, error)
}

func (a *iamAdapter) StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error) {
	return a.studentBelongsToFamily(ctx, studentID, familyID)
}

func (a *iamAdapter) GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error) {
	return a.getStudentName(ctx, studentID)
}

// NewIamAdapter creates an IamServiceForLearn adapter from raw functions.
func NewIamAdapter(
	studentBelongsToFamily func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error),
	getStudentName func(ctx context.Context, studentID uuid.UUID) (string, error),
) IamServiceForLearn {
	return &iamAdapter{
		studentBelongsToFamily: studentBelongsToFamily,
		getStudentName:         getStudentName,
	}
}
