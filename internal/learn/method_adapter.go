package learn

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// methodAdapter implements MethodServiceForLearn using raw functions.
// The adapter pattern allows learn:: to consume method:: without importing
// the method package directly. Wired in cmd/server/main.go. [ARCH §4.2]
type methodAdapter struct {
	resolveFamilyTools  func(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error)
	resolveStudentTools func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error)
}

func (a *methodAdapter) ResolveFamilyTools(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error) {
	return a.resolveFamilyTools(ctx, scope)
}

func (a *methodAdapter) ResolveStudentTools(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error) {
	return a.resolveStudentTools(ctx, scope, studentID)
}

// NewMethodAdapter creates a MethodServiceForLearn adapter from raw functions.
func NewMethodAdapter(
	resolveFamilyTools func(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error),
	resolveStudentTools func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error),
) MethodServiceForLearn {
	return &methodAdapter{
		resolveFamilyTools:  resolveFamilyTools,
		resolveStudentTools: resolveStudentTools,
	}
}
