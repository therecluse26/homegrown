package shared

import (
	"time"

	"github.com/google/uuid"
)

// HearthClaims contains verified claims extracted from a Hearth JWT.
// These fields mirror shared.Session but are typed for clarity within the adapter.
// Email MUST NOT be logged. [CODING §5.2, ARCH ADR-017]
type HearthClaims struct {
	Sub   uuid.UUID // hearth_user_id (JWT sub — iam_parents.hearth_user_id)
	Oid   uuid.UUID // hearth_org_id  (JWT oid — iam_families.hearth_org_id = family_id)
	Email string    // NEVER log — PII [CODING §5.2]
	Roles []string
	Exp   time.Time
}
