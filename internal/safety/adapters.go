package safety

import (
	"context"

	"github.com/google/uuid"
)

// ─── IAM Adapter ─────────────────────────────────────────────────────────────

// iamAdapterImpl implements IamServiceForSafety using a raw function.
// Avoids circular dependency with iam::. Wired in cmd/server/main.go. [ARCH §4.2]
type iamAdapterImpl struct {
	revokeSessions func(ctx context.Context, familyID uuid.UUID) error
}

func (a *iamAdapterImpl) RevokeSessions(ctx context.Context, familyID uuid.UUID) error {
	return a.revokeSessions(ctx, familyID)
}

// NewIamAdapter creates an IamServiceForSafety adapter from a raw function.
func NewIamAdapter(revokeSessions func(ctx context.Context, familyID uuid.UUID) error) IamServiceForSafety {
	return &iamAdapterImpl{revokeSessions: revokeSessions}
}
