package middleware

import (
	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// ─── Role Extractors ──────────────────────────────────────────────────────────
// Helper functions that extract AuthContext with additional permission checks.
// Shared infrastructure — not IAM-specific. [§13.3]

// RequirePremium extracts AuthContext and verifies the user has a premium subscription.
// Returns 402 Payment Required if the user is on the free tier. [S§3.2]
// Consuming domains: learn::, comply::, recs::
func RequirePremium(c echo.Context) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	if auth.SubscriptionTier != shared.SubscriptionTierPremium {
		return nil, shared.ErrPremiumRequired()
	}

	return auth, nil
}

// RequireCoppaConsent extracts AuthContext and verifies the family has active COPPA consent.
// Returns 403 Forbidden with code `coppa_consent_required` if consent status is not
// "consented" or "re_verified". [ARCH §6.3]
//
// CoppaConsentStatus is populated by auth middleware from iam_families at login —
// no additional DB query per request. [§13.3 approach 1]
// Consuming domains: learn::, social:: (student-facing features)
func RequireCoppaConsent(c echo.Context) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	switch auth.CoppaConsentStatus {
	case "consented", "re_verified":
		return auth, nil
	default:
		return nil, shared.ErrCoppaConsentRequired()
	}
}

// CreatorContext holds auth context plus verified creator ID.
type CreatorContext struct {
	Auth      *shared.AuthContext
	CreatorID uuid.UUID
}

// RequireCreator extracts AuthContext and verifies the user has a creator account.
// Returns 403 Forbidden if no creator account exists. [S§3.1.4]
//
// TODO(07-mkt): Replace stub with creator lookup from mkt_creators table.
// Preferred approach: add CreatorID *uuid.UUID to AuthContext (populated during
// auth middleware from a JOIN) — no cross-domain call per request. [§13.3]
// Consuming domains: mkt::
func RequireCreator(c echo.Context) (*CreatorContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	// TODO(07-mkt): Look up creator account for auth.ParentID in mkt_creators.
	// Until marketplace domain is implemented, all creator checks deny access.
	_ = auth
	return nil, shared.ErrForbidden()
}

// RequireAdmin extracts AuthContext and verifies the user is a platform administrator.
// Returns 403 Forbidden if the user is not an admin. [S§3.1.5, 11-safety §9]
//
// Backed by iam_parents.is_platform_admin column (01-iam §3.1).
// Phase 1: single boolean. Phase 2: granular admin roles. [11-safety §9]
// Consuming domains: safety:: (moderation dashboard, admin actions)
func RequireAdmin(c echo.Context) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	if !auth.IsPlatformAdmin {
		return nil, shared.ErrForbidden()
	}

	return auth, nil
}

// RequirePrimaryParent extracts AuthContext and verifies the user is the family's
// primary parent. Returns 403 Forbidden if not. [S§3.4]
//
// Phase 2: used by co-parent management, family deletion, and COPPA withdrawal endpoints.
// IsPrimaryParent is populated by auth middleware from iam_parents at login.
func RequirePrimaryParent(c echo.Context) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	if !auth.IsPrimaryParent {
		return nil, shared.ErrForbidden()
	}

	return auth, nil
}
