package shared

import (
	"context"

	"github.com/google/uuid"
)

// Session contains the validated identity fields returned by the auth provider.
// Email is present for service-layer use but MUST NOT be logged or serialized. [CODING §5.2]
type Session struct {
	// IdentityID is the auth-provider user ID (Hearth JWT sub claim).
	// Maps to iam_parents.hearth_user_id after the WS4 migration. [ADR-B]
	IdentityID uuid.UUID
	// OrgID is the Hearth org ID (JWT oid claim) which equals the family_id
	// under the org-per-family model. Used to derive FamilyScope without a DB JOIN. [ADR-B]
	OrgID uuid.UUID
	// SessionID is the opaque session identifier for current-session detection. [15-lifecycle §12]
	SessionID string
	// Email is PII — never log or serialize. [CODING §5.2]
	Email string
}

// SessionValidator is the generic auth port for validating browser sessions.
// The concrete implementation is wired in 01-iam/adapters and lives in
// internal/iam/adapters/ — no auth-provider types leak into application-layer code. [ARCH §4.1]
type SessionValidator interface {
	// ValidateSession validates the session and returns the associated identity.
	// For the Hearth BFF adapter: sid → session store → local JWT decode.
	// Returns an error if the session is missing, expired, or invalid.
	ValidateSession(ctx context.Context, sessionCookie string) (*Session, error)
}
