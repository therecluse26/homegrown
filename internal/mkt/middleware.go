package mkt

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// CreatorContext holds the authenticated user's creator information.
// Extracted by RequireCreator middleware and stored in Echo context.
type CreatorContext struct {
	Auth      *shared.AuthContext
	CreatorID uuid.UUID
}

// creatorLookup is the function type for looking up a creator by parent ID.
// Injected at composition root to avoid direct dependency on MarketplaceService.
type creatorLookup func(ctx context.Context, parentID uuid.UUID) (*CreatorResponse, error)

// RequireCreator is an Echo middleware that verifies the user has a creator account.
// Returns 403 Forbidden if no creator account exists. [07-mkt §16]
//
// Uses Redis caching to avoid DB query on every request.
// Cache key: "mkt:creator:{parent_id}" with 5-minute TTL.
func RequireCreator(cache shared.Cache, lookup creatorLookup) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			auth, err := shared.GetAuthContext(c)
			if err != nil {
				return err
			}

			cacheKey := fmt.Sprintf("mkt:creator:%s", auth.ParentID)

			// Try cache first
			creatorIDStr, cacheErr := cache.Get(c.Request().Context(), cacheKey)
			if cacheErr == nil && creatorIDStr != "" {
				creatorID, parseErr := uuid.Parse(creatorIDStr)
				if parseErr == nil {
					c.Set("creator", &CreatorContext{Auth: auth, CreatorID: creatorID})
					return next(c)
				}
			}

			// Cache miss — query service
			creator, lookupErr := lookup(c.Request().Context(), auth.ParentID)
			if lookupErr != nil {
				return lookupErr
			}
			if creator == nil {
				return shared.ErrForbidden()
			}

			// Cache for 5 minutes (ignore set error — cache is non-critical)
			_ = cache.Set(c.Request().Context(), cacheKey, creator.ID.String(), 5*time.Minute)

			c.Set("creator", &CreatorContext{Auth: auth, CreatorID: creator.ID})
			return next(c)
		}
	}
}

// GetCreatorContext retrieves the CreatorContext from the Echo context.
// Only valid in handlers behind RequireCreator middleware.
func GetCreatorContext(c echo.Context) (*CreatorContext, error) {
	val := c.Get("creator")
	if val == nil {
		return nil, shared.ErrForbidden()
	}
	cc, ok := val.(*CreatorContext)
	if !ok {
		return nil, shared.ErrForbidden()
	}
	return cc, nil
}
