package learner_profile

import (
	"context"

	"github.com/google/uuid"
)

// iamAdapter implements IamServiceForLearnerProfile using injected functions.
// Wired in cmd/server/main.go to bridge iam.IamService → learner_profile port.
// Avoids circular dependency with iam::. [ARCH §4.2, MEMORY consumer-defined interfaces]
type iamAdapter struct {
	studentBelongsToFamily func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error)
	getStudentDisplayName  func(ctx context.Context, studentID uuid.UUID) (string, error)
}

// NewIamAdapter constructs the IAM adapter from caller-supplied functions.
func NewIamAdapter(
	studentBelongsToFamily func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error),
	getStudentDisplayName func(ctx context.Context, studentID uuid.UUID) (string, error),
) IamServiceForLearnerProfile {
	return &iamAdapter{
		studentBelongsToFamily: studentBelongsToFamily,
		getStudentDisplayName:  getStudentDisplayName,
	}
}

func (a *iamAdapter) StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error) {
	return a.studentBelongsToFamily(ctx, studentID, familyID)
}

func (a *iamAdapter) GetStudentDisplayName(ctx context.Context, studentID uuid.UUID) (string, error) {
	return a.getStudentDisplayName(ctx, studentID)
}

// Compile-time interface check.
var _ IamServiceForLearnerProfile = (*iamAdapter)(nil)
