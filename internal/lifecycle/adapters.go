package lifecycle

import (
	"context"

	"github.com/google/uuid"
)

// ─── IAM Adapter ────────────────────────────────────────────────────────────

// iamAdapter implements IamServiceForLifecycle using raw functions.
// Avoids circular dependency with iam::. Wired in cmd/server/main.go. [ARCH §4.2]
type iamAdapter struct {
	initiateRecoveryFlow  func(ctx context.Context, email string) error
	listSessions          func(ctx context.Context, parentID uuid.UUID) ([]SessionInfo, error)
	revokeSession         func(ctx context.Context, sessionID string) error
	revokeAllSessions     func(ctx context.Context, parentID uuid.UUID, currentSessionID string) (uint32, error)
	revokeFamilySessions  func(ctx context.Context, familyID uuid.UUID) error
}

func (a *iamAdapter) InitiateRecoveryFlow(ctx context.Context, email string) error {
	return a.initiateRecoveryFlow(ctx, email)
}

func (a *iamAdapter) ListSessions(ctx context.Context, parentID uuid.UUID) ([]SessionInfo, error) {
	return a.listSessions(ctx, parentID)
}

func (a *iamAdapter) RevokeSession(ctx context.Context, sessionID string) error {
	return a.revokeSession(ctx, sessionID)
}

func (a *iamAdapter) RevokeAllSessions(ctx context.Context, parentID uuid.UUID, currentSessionID string) (uint32, error) {
	return a.revokeAllSessions(ctx, parentID, currentSessionID)
}

func (a *iamAdapter) RevokeFamilySessions(ctx context.Context, familyID uuid.UUID) error {
	return a.revokeFamilySessions(ctx, familyID)
}

// NewIamAdapter creates an IamServiceForLifecycle adapter from raw functions.
func NewIamAdapter(
	initiateRecoveryFlow func(ctx context.Context, email string) error,
	listSessions func(ctx context.Context, parentID uuid.UUID) ([]SessionInfo, error),
	revokeSession func(ctx context.Context, sessionID string) error,
	revokeAllSessions func(ctx context.Context, parentID uuid.UUID, currentSessionID string) (uint32, error),
	revokeFamilySessions func(ctx context.Context, familyID uuid.UUID) error,
) IamServiceForLifecycle {
	return &iamAdapter{
		initiateRecoveryFlow: initiateRecoveryFlow,
		listSessions:         listSessions,
		revokeSession:        revokeSession,
		revokeAllSessions:    revokeAllSessions,
		revokeFamilySessions: revokeFamilySessions,
	}
}

// ─── Billing Adapter ─────────────────────────────────────────────────────────

// billingAdapter implements BillingServiceForLifecycle using a raw function.
type billingAdapter struct {
	cancelFamilySubscriptions func(ctx context.Context, familyID uuid.UUID) error
}

func (a *billingAdapter) CancelFamilySubscriptions(ctx context.Context, familyID uuid.UUID) error {
	return a.cancelFamilySubscriptions(ctx, familyID)
}

// NewBillingAdapter creates a BillingServiceForLifecycle adapter from a raw function.
func NewBillingAdapter(
	cancelFamilySubscriptions func(ctx context.Context, familyID uuid.UUID) error,
) BillingServiceForLifecycle {
	return &billingAdapter{cancelFamilySubscriptions: cancelFamilySubscriptions}
}
