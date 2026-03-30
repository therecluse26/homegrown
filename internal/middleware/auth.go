package middleware

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// authDeps is the interface that AppState must satisfy for the auth middleware.
// Defined here (not in app/) to avoid a circular import: app imports middleware,
// so middleware MUST NOT import app. [Plan: Circular Import Avoidance]
type authDeps interface {
	// GetAuthValidator returns the SessionValidator used to validate browser sessions.
	GetAuthValidator() shared.SessionValidator

	// GetDB returns the database pool for parent+family lookup.
	GetDB() *gorm.DB
}

// authLookup holds the result of the JOIN query used to build AuthContext.
// Defined here (not in iam/) to avoid a circular import: iam imports middleware... no wait,
// actually iam does not import middleware. But to avoid any future circular dependency risk,
// we keep this struct middleware-local and use raw SQL. [01-iam §11.1]
type authLookup struct {
	ParentID        string
	FamilyID        string
	IdentityID      string
	DisplayName     string
	Email           string
	IsPrimary       bool
	IsPlatformAdmin bool
	Tier            string
	ConsentStatus   string
}

// kratosSessionCookieName is the default Ory Kratos session cookie name.
const kratosSessionCookieName = "ory_kratos_session"

// Auth returns an Echo middleware that validates Kratos sessions and populates AuthContext.
//
// Flow: [01-iam §11.1]
//  1. Extract Ory Kratos session cookie from the request
//  2. Validate session via SessionValidator.ValidateSession (calls Kratos /sessions/whoami)
//  3. JOIN-query iam_parents + iam_families by kratos_identity_id (RLS bypassed)
//  4. Build AuthContext with parent + family fields
//  5. Set AuthContext on Echo context; call next handler
//
// Returns 401 if:
//   - No session cookie is present
//   - Kratos session is invalid or expired
//   - Parent not found in local DB (orphaned Kratos identity)
func Auth(deps authDeps) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			validator := deps.GetAuthValidator()
			if validator == nil {
				// No SessionValidator wired yet — should not happen in production.
				return shared.ErrUnauthorized()
			}

			// Step 1: Extract session cookie.
			cookie, err := c.Request().Cookie(kratosSessionCookieName)
			if err != nil {
				if err == http.ErrNoCookie {
					return shared.ErrUnauthorized()
				}
				return shared.ErrUnauthorized()
			}
			cookieHeader := fmt.Sprintf("%s=%s", cookie.Name, cookie.Value)

			// Step 2: Validate session with Kratos.
			session, err := validator.ValidateSession(c.Request().Context(), cookieHeader)
			if err != nil {
				return shared.ErrUnauthorized()
			}

			db := deps.GetDB()

			// Step 3: JOIN-query to build AuthContext.
			// RLS bypassed via SET LOCAL row_security = off — no family scope is available
			// yet (we're finding the parent by Kratos identity_id). [01-iam §11.1]
			var lookup authLookup
			err = db.WithContext(c.Request().Context()).Transaction(func(tx *gorm.DB) error {
				// Bypass RLS: auth middleware runs before FamilyScope is constructed.
				if execErr := tx.Exec("SET LOCAL row_security = off").Error; execErr != nil {
					return execErr
				}
				result := tx.Raw(`
					SELECT
						p.id           AS parent_id,
						p.family_id    AS family_id,
						p.kratos_identity_id AS identity_id,
						p.display_name AS display_name,
						p.email        AS email,
						p.is_primary   AS is_primary,
						p.is_platform_admin AS is_platform_admin,
						f.subscription_tier AS tier,
						f.coppa_consent_status AS consent_status
					FROM iam_parents p
					JOIN iam_families f ON f.id = p.family_id
					WHERE p.kratos_identity_id = ?
				`, session.IdentityID).Scan(&lookup)
				return result.Error
			})
			if err != nil {
				slog.Error("auth middleware: db error", "error", err)
				return shared.ErrUnauthorized()
			}

			// Step 4: Check parent was found.
			if lookup.ParentID == "" {
				// Orphaned Kratos identity — parent not found in local DB.
				// This can happen if the registration webhook failed.
				return shared.ErrUnauthorized()
			}

			parentID, err := uuid.Parse(lookup.ParentID)
			if err != nil {
				return shared.ErrUnauthorized()
			}
			familyID, err := uuid.Parse(lookup.FamilyID)
			if err != nil {
				return shared.ErrUnauthorized()
			}
			identityID, err := uuid.Parse(lookup.IdentityID)
			if err != nil {
				return shared.ErrUnauthorized()
			}

			// Step 5: Build and store AuthContext.
			auth := &shared.AuthContext{
				ParentID:           parentID,
				FamilyID:           familyID,
				IdentityID:         identityID,
				DisplayName:        lookup.DisplayName,
				IsPrimaryParent:    lookup.IsPrimary,
				IsPlatformAdmin:    lookup.IsPlatformAdmin,
				SubscriptionTier:   shared.ParseSubscriptionTier(lookup.Tier),
				CoppaConsentStatus: lookup.ConsentStatus,
				Email:              lookup.Email,      // PII — not logged [CODING §5.2]
				SessionID:          session.SessionID, // for current-session detection [15-lifecycle §12]
			}
			shared.SetAuthContext(c, auth)

			return next(c)
		}
	}
}
