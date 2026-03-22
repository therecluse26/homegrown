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
// (e.g. GetDB() for IAM parent lookup, KratosAdapter for session validation).
type authDeps interface {
	// GetKratosPublicURL returns the Kratos public API base URL for session validation.
	GetKratosPublicURL() string
}

// Auth returns an Echo middleware that validates Kratos sessions and populates AuthContext.
//
// Stub implementation — full auth wiring is added in 01-iam when KratosAdapter is
// provided. Until then, all requests to authenticated routes receive 401 Unauthorized.
// [§13.1]
// Auth receives deps so that when 01-iam replaces this stub, the caller (app.NewApp)
// requires no signature change. The parameter is intentionally unused in this stub.
func Auth(deps authDeps) echo.MiddlewareFunc { //nolint:unparam // stub; deps used in 01-iam
	_ = deps // suppress unparam until IAM wires KratosAdapter
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// TODO(01-iam): Replace this stub with full Kratos session validation:
			// 1. Extract session cookie from Cookie header
			// 2. Validate via KratosAdapter.ValidateSession(ctx, cookie) using deps.GetKratosPublicURL()
			// 3. Look up parent in local DB by kratos_identity_id (UnscopedTransaction — no FamilyScope yet)
			// 4. Look up parent's family from iam_families
			// 5. Build AuthContext (including coppa_consent_status, subscription_tier, is_primary_parent)
			// 5.5 Check account access via SafetyService.CheckAccountAccess(familyID) [11-safety §12.3]
			// 6. shared.SetAuthContext(c, auth)
			// 7. return next(c)
			//
			// Error responses:
			// - No cookie present → 401 Unauthorized
			// - Kratos session invalid/expired → 401 Unauthorized
			// - Parent not found (orphaned Kratos identity) → 401 Unauthorized
			return shared.ErrUnauthorized()
		}
	}
}
