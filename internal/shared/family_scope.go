package shared

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// FamilyScope wraps a family_id for privacy-enforcing database queries.
//
// The unexported field ensures FamilyScope can only be created from:
//  1. AuthContext (via NewFamilyScopeFromAuth) — the normal authenticated path
//  2. newFamilyScope — for auth middleware and registration flows (package-internal)
//
// Every repository method that touches user-generated data MUST accept a FamilyScope
// parameter and filter by it. [CODING §2.4, ARCH §1.5]
type FamilyScope struct {
	familyID uuid.UUID
}

// newFamilyScope creates a FamilyScope from a raw family_id.
// Package-internal use only (auth middleware, registration flows).
func newFamilyScope(familyID uuid.UUID) FamilyScope {
	return FamilyScope{familyID: familyID}
}

// FamilyID returns the wrapped family_id. The only public accessor for the value.
func (s FamilyScope) FamilyID() uuid.UUID {
	return s.familyID
}

// NewFamilyScopeFromAuth creates a FamilyScope from an AuthContext.
// Use this in authenticated handlers/services to scope database queries.
func NewFamilyScopeFromAuth(auth *AuthContext) FamilyScope {
	return newFamilyScope(auth.FamilyID)
}

// NewFamilyScopeFromID creates a FamilyScope from a raw family_id.
// Use this in event handlers and background jobs where AuthContext is not available
// but the family_id is known from the domain event payload. [CODING §2.4]
func NewFamilyScopeFromID(familyID uuid.UUID) FamilyScope {
	return newFamilyScope(familyID)
}

// GetFamilyScope extracts a FamilyScope from the Echo context's AuthContext.
// Returns an error if AuthContext is not present (i.e. handler is not behind auth middleware).
func GetFamilyScope(c echo.Context) (FamilyScope, error) {
	auth, err := GetAuthContext(c)
	if err != nil {
		return FamilyScope{}, err
	}
	return NewFamilyScopeFromAuth(auth), nil
}
