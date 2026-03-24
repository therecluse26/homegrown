package learn

import (
	"context"

	"github.com/google/uuid"
)

// mktStubAdapter implements MktServiceForLearn as a stub.
// Returns true for all publisher membership checks until mkt:: is implemented.
// TODO: Replace with real adapter when mkt:: domain exists. [ARCH §4.2]
type mktStubAdapter struct{}

func (a *mktStubAdapter) IsPublisherMember(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return true, nil // stub: all callers are assumed to be publisher members
}

// NewMktStubAdapter creates a stub MktServiceForLearn that always returns true.
func NewMktStubAdapter() MktServiceForLearn {
	return &mktStubAdapter{}
}
