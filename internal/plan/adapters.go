package plan

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── IAM Adapter ────────────────────────────────────────────────────────────

// iamAdapter implements IamServiceForPlan using raw functions.
// Avoids circular dependency with iam::. Wired in cmd/server/main.go. [ARCH §4.2]
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

// ─── Learning Adapter ────────────────────────────────────────────────────────

// learnAdapter implements LearningServiceForPlan using raw functions.
type learnAdapter struct {
	listActivitiesForCalendar func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]ActivitySummary, error)
	logActivity               func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, title string, date time.Time, durationMinutes *int, studentID *uuid.UUID, description *string, tags []string) (uuid.UUID, error)
}

func (a *learnAdapter) ListActivitiesForCalendar(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]ActivitySummary, error) {
	return a.listActivitiesForCalendar(ctx, auth, scope, start, end, studentID)
}

func (a *learnAdapter) LogActivity(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, title string, date time.Time, durationMinutes *int, studentID *uuid.UUID, description *string, tags []string) (uuid.UUID, error) {
	return a.logActivity(ctx, auth, scope, title, date, durationMinutes, studentID, description, tags)
}

// NewLearnAdapter creates a LearningServiceForPlan adapter from raw functions.
func NewLearnAdapter(
	listActivitiesForCalendar func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, start, end time.Time, studentID *uuid.UUID) ([]ActivitySummary, error),
	logActivity func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, title string, date time.Time, durationMinutes *int, studentID *uuid.UUID, description *string, tags []string) (uuid.UUID, error),
) LearningServiceForPlan {
	return &learnAdapter{
		listActivitiesForCalendar: listActivitiesForCalendar,
		logActivity:               logActivity,
	}
}

// ─── IAM Adapter ────────────────────────────────────────────────────────────

// NewIamAdapter creates an IamServiceForPlan adapter from raw functions.
func NewIamAdapter(
	studentBelongsToFamily func(ctx context.Context, studentID uuid.UUID, familyID uuid.UUID) (bool, error),
	getStudentName func(ctx context.Context, studentID uuid.UUID) (string, error),
) IamServiceForPlan {
	return &iamAdapter{
		studentBelongsToFamily: studentBelongsToFamily,
		getStudentName:         getStudentName,
	}
}
