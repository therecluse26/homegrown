package middleware

import (
	"log/slog"

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

	// GetDB returns the database pool for parent lookup.
	GetDB() *gorm.DB
}

// authLookup holds the result of the query used to build AuthContext.
// Only parent-specific fields are fetched; family_id comes from the JWT oid claim. [ADR-B]
type authLookup struct {
	ParentID        string
	FamilyID        string
	DisplayName     string
	Email           string
	IsPrimary       bool
	IsPlatformAdmin bool
	Tier            string
	ConsentStatus   string
}

// sidCookieName is the server-issued session cookie name for the Hearth BFF flow. [ADR-D]
const sidCookieName = "sid"

// Auth returns an Echo middleware that validates Hearth BFF sessions and populates AuthContext.
//
// Flow: [01-iam §11.1, ADR-A, ADR-D]
//  1. Extract the sid cookie from the request
//  2. Validate via SessionValidator.ValidateSession — sid → session store → local JWT decode
//  3. Derive FamilyScope from session.OrgID (JWT oid claim); no DB JOIN required [ADR-B]
//  4. Query iam_parents by hearth_user_id (= JWT sub) for parent + family details
//  5. Build AuthContext and store it on the Echo context; call next handler
//
// Returns 401 if:
//   - No sid cookie is present
//   - Session not found in store, or access token expired
//   - Parent not found by hearth_user_id (unprovisioned identity)
func Auth(deps authDeps) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			validator := deps.GetAuthValidator()
			if validator == nil {
				return shared.ErrUnauthorized()
			}

			// Step 1: Extract the sid cookie. No fallback header — BFF sessions are
			// always cookie-based; Bearer tokens are not accepted here. [ADR-D]
			cookie, err := c.Request().Cookie(sidCookieName)
			if err != nil {
				return shared.ErrUnauthorized()
			}
			sid := cookie.Value
			if sid == "" {
				return shared.ErrUnauthorized()
			}

			// Step 2: Validate the session — sid → store lookup → local JWT decode.
			// Zero calls to Hearth in steady state. [ADR-A]
			session, err := validator.ValidateSession(c.Request().Context(), sid)
			if err != nil {
				return shared.ErrUnauthorized()
			}

			// Step 3: Derive FamilyScope from the verified JWT oid claim.
			// The family_id is embedded in the token — no DB JOIN needed. [ADR-B]
			if session.OrgID == uuid.Nil {
				return shared.ErrUnauthorized()
			}

			db := deps.GetDB()

			// Step 4: Look up parent details by hearth_user_id (= JWT sub).
			// Family details are fetched via JOIN to get subscription tier and COPPA status.
			// family_id from the JOIN is validated against session.OrgID as a defence-in-depth check.
			var lookup authLookup
			err = db.WithContext(c.Request().Context()).Raw(`
				SELECT
					p.id                   AS parent_id,
					p.family_id            AS family_id,
					p.display_name         AS display_name,
					p.email                AS email,
					p.is_primary           AS is_primary,
					p.is_platform_admin    AS is_platform_admin,
					f.subscription_tier    AS tier,
					f.coppa_consent_status AS consent_status
				FROM iam_parents p
				JOIN iam_families f ON f.id = p.family_id
				WHERE p.hearth_user_id = ?
			`, session.IdentityID).Scan(&lookup).Error
			if err != nil {
				slog.Error("auth middleware: db error", "error", err)
				return shared.ErrUnauthorized()
			}

			if lookup.ParentID == "" {
				// hearth_user_id not found — identity not yet provisioned in local DB.
				return shared.ErrUnauthorized()
			}

			// Step 5: Parse UUIDs and verify the JWT family claim matches the DB row.
			parentID, err := uuid.Parse(lookup.ParentID)
			if err != nil {
				return shared.ErrUnauthorized()
			}
			dbFamilyID, err := uuid.Parse(lookup.FamilyID)
			if err != nil {
				return shared.ErrUnauthorized()
			}
			// Defence-in-depth: JWT oid must agree with the DB-stored family_id. [ADR-B]
			if dbFamilyID != session.OrgID {
				slog.Warn("auth middleware: JWT oid mismatch with DB family_id",
					"jwt_oid", session.OrgID,
					"db_family_id", dbFamilyID,
				)
				return shared.ErrUnauthorized()
			}

			// Step 6: Build and store AuthContext. FamilyScope comes from the JWT. [ADR-B]
			auth := &shared.AuthContext{
				ParentID:           parentID,
				FamilyID:           session.OrgID, // from JWT — no extra lookup [ADR-B]
				IdentityID:         session.IdentityID,
				DisplayName:        lookup.DisplayName,
				IsPrimaryParent:    lookup.IsPrimary,
				IsPlatformAdmin:    lookup.IsPlatformAdmin,
				SubscriptionTier:   shared.ParseSubscriptionTier(lookup.Tier),
				CoppaConsentStatus: lookup.ConsentStatus,
				Email:              session.Email, // PII — not logged [CODING §5.2]
				SessionID:          session.SessionID,
			}
			shared.SetAuthContext(c, auth)

			return next(c)
		}
	}
}
