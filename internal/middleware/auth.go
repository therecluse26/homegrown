package middleware

import (
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// authDeps is the interface that AppState must satisfy for the auth middleware.
// Defined here (not in app/) to avoid a circular import: app imports middleware,
// so middleware MUST NOT import app. [Plan: Circular Import Avoidance]
//
// When 01-iam implements the full auth flow, additional methods will be added here
// (e.g. GetDB() for IAM parent lookup). The SessionValidator is passed through
// GetAuthValidator() and used in the stub below.
type authDeps interface {
	// GetAuthValidator returns the SessionValidator used to validate browser sessions.
	// Returns nil until 01-iam wires a concrete implementation.
	GetAuthValidator() shared.SessionValidator
}

// Auth returns an Echo middleware that validates sessions and populates AuthContext.
//
// Stub implementation — full auth wiring is added in 01-iam when a concrete
// SessionValidator is provided via deps.GetAuthValidator(). Until then, all requests
// to authenticated routes receive 401 Unauthorized. [§13.1]
func Auth(deps authDeps) echo.MiddlewareFunc { //nolint:unparam // stub; deps used in 01-iam
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if deps.GetAuthValidator() == nil {
				// No SessionValidator wired yet (pre 01-iam). Reject all requests.
				return shared.ErrUnauthorized()
			}

			// TODO(01-iam): Replace this stub with full session validation:
			// 1. Extract session cookie from Cookie header
			// 2. Validate via deps.GetAuthValidator().ValidateSession(ctx, cookie)
			// 3. Look up parent in local DB by identity_id (UnscopedTransaction — no FamilyScope yet)
			// 4. Look up parent's family from iam_families
			// 5. Build AuthContext (including coppa_consent_status, subscription_tier, is_primary_parent)
			// 5.5 Check account access via SafetyService.CheckAccountAccess(familyID) [11-safety §12.3]
			// 6. shared.SetAuthContext(c, auth)
			// 7. return next(c)
			//
			// Error responses:
			// - No cookie present → 401 Unauthorized
			// - Session invalid/expired → 401 Unauthorized
			// - Parent not found (orphaned identity) → 401 Unauthorized
			return shared.ErrUnauthorized()
		}
	}
}
