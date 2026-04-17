package middleware

import (
	"errors"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
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

// RequireTier extracts AuthContext and verifies the user's subscription tier meets
// or exceeds the required tier. Tier ordering: free < plus < premium. [S§3.2]
//
// Returns 402 Payment Required (code `tier_required`) if the user's tier is too low.
// Populated from AuthContext.SubscriptionTier — no DB query per request.
//
// Consuming domains: learn:: (advanced tools/analytics → plus), comply::/recs:: (premium).
func RequireTier(c echo.Context, required shared.SubscriptionTier) (*shared.AuthContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	if !auth.SubscriptionTier.MeetsTier(required) {
		return nil, shared.ErrTierRequired(required)
	}

	return auth, nil
}

// RequireTierMiddleware returns an Echo middleware that blocks requests whose
// subscription tier is below `required`. Use to gate entire route groups. [S§3.2]
//
// Handlers inside the gated group may still call shared.GetAuthContext — the
// middleware does not consume or alter the auth context.
func RequireTierMiddleware(required shared.SubscriptionTier) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if _, err := RequireTier(c, required); err != nil {
				return err
			}
			return next(c)
		}
	}
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

// RequireCreator extracts AuthContext and verifies the user has a creator account
// by querying mkt_creators. Returns 403 Forbidden if no active creator account exists. [S§3.1.4]
// Consuming domains: billing::
func RequireCreator(c echo.Context, db *gorm.DB) (*CreatorContext, error) {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return nil, err
	}

	type creatorRow struct {
		ID uuid.UUID `gorm:"column:id"`
	}
	var row creatorRow
	err = db.WithContext(c.Request().Context()).
		Table("mkt_creators").
		Select("id").
		Where("parent_id = ? AND onboarding_status = 'active'", auth.ParentID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, shared.ErrForbidden()
	}
	if err != nil {
		return nil, err
	}
	return &CreatorContext{Auth: auth, CreatorID: row.ID}, nil
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
