package shared

import (
	"context"

	"github.com/google/uuid"
)

// Session contains the validated identity fields returned by the auth provider.
// Email is present for service-layer use but MUST NOT be logged or serialized. [CODING §5.2]
type Session struct {
	IdentityID uuid.UUID
	Email      string // PII — never log [CODING §5.2]
}

// SessionValidator is the generic auth port for validating browser sessions.
// The concrete implementation (KratosSessionValidator) is wired in 01-iam and lives
// in internal/iam/adapters/ — no Kratos types leak into application-layer code. [ARCH §4.1]
type SessionValidator interface {
	// ValidateSession validates the session cookie and returns the associated identity.
	// Returns an error if the session is missing, expired, or invalid.
	ValidateSession(ctx context.Context, sessionCookie string) (*Session, error)
}
