package learn

import (
	"context"

	"github.com/google/uuid"
)

// ─── Function-based Real Adapter ─────────────────────────────────────────────

// mktAdapter implements MktServiceForLearn by delegating to injected functions.
// Wired in main.go to bridge mkt::MarketplaceService → learn::MktServiceForLearn. [ARCH §4.2]
type mktAdapter struct {
	isPublisherMemberFn func(ctx context.Context, callerID, publisherID uuid.UUID) (bool, error)
}

func (a *mktAdapter) IsPublisherMember(ctx context.Context, callerID, publisherID uuid.UUID) (bool, error) {
	return a.isPublisherMemberFn(ctx, callerID, publisherID)
}

// NewMktAdapter creates a real MktServiceForLearn adapter backed by mkt:: service functions.
func NewMktAdapter(
	isPublisherMemberFn func(ctx context.Context, callerID, publisherID uuid.UUID) (bool, error),
) MktServiceForLearn {
	return &mktAdapter{isPublisherMemberFn: isPublisherMemberFn}
}

// ─── Stub Adapter (for development/testing) ──────────────────────────────────

// mktStubAdapter implements MktServiceForLearn as a stub.
// Returns true for all publisher membership checks. [ARCH §4.2]
type mktStubAdapter struct{}

func (a *mktStubAdapter) IsPublisherMember(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return true, nil // stub: all callers are assumed to be publisher members
}

// NewMktStubAdapter creates a stub MktServiceForLearn that always returns true.
func NewMktStubAdapter() MktServiceForLearn {
	return &mktStubAdapter{}
}
