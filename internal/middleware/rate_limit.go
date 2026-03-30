package middleware

import (
	"fmt"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// rateLimitDeps is the interface that AppState must satisfy for rate limiting.
// Defined here to avoid importing app/ (which would create a circular dependency).
type rateLimitDeps interface {
	GetCache() shared.Cache
}

// RateLimit returns a middleware that enforces per-IP or per-user request rate limits
// using cache atomic counters. [§13.2, ARCH §3.3]
//
// Key format: rl:{scope}:{identifier}:{window_start_unix}
// On exceeded limit: returns 429 with Retry-After header (seconds until window expires).
func RateLimit(deps rateLimitDeps, limit int, window time.Duration) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			scope, identifier := resolveRateLimitIdentity(c)
			key := buildRateLimitKey(scope, identifier, window)

			count, err := shared.CacheIncrementWithExpiry(ctx, deps.GetCache(), key, window)
			if err != nil {
				// Cache failure should not block requests — degrade gracefully.
				return next(c)
			}

			if count > int64(limit) {
				retryAfter := retryAfterSeconds(window)
				c.Response().Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				return shared.ErrRateLimited()
			}

			return next(c)
		}
	}
}

// resolveRateLimitIdentity returns the scope and identifier for rate limit key construction.
// Authenticated users are scoped by user ID; unauthenticated requests by IP.
// IP addresses are hashed before use — never stored or logged in plaintext. [CODING §5.2]



func resolveRateLimitIdentity(c echo.Context) (scope, identifier string) {
	auth, err := shared.GetAuthContext(c)
	if err == nil {
		// Authenticated: scope by parent ID
		return "user", auth.ParentID.String()
	}
	// Unauthenticated: scope by real IP (hashed — never stored in plaintext)
	return "ip", hashIP(c.RealIP())
}

// hashIP returns a deterministic non-reversible representation of an IP address.
// We never log or store raw IPs — only use them as opaque rate limit bucket keys. [CODING §5.2]
func hashIP(ip string) string {
	var h uint64
	for _, ch := range ip {
		h = h*31 + uint64(ch) //nolint:gosec // integer overflow is fine for hash bucketing
	}
	return fmt.Sprintf("%x", h)
}

// buildRateLimitKey constructs the cache key for a given scope/identifier/window.
func buildRateLimitKey(scope, identifier string, window time.Duration) string {
	// window_start is the Unix timestamp rounded down to the window boundary,
	// creating a fixed slot so all requests in the same period share one counter.
	windowSeconds := int64(window.Seconds())
	windowStart := time.Now().Unix() / windowSeconds * windowSeconds
	return fmt.Sprintf("rl:%s:%s:%d", scope, identifier, windowStart)
}

// retryAfterSeconds returns the number of seconds until the current window expires.
func retryAfterSeconds(window time.Duration) int64 {
	windowSeconds := int64(window.Seconds())
	windowStart := time.Now().Unix() / windowSeconds * windowSeconds
	windowEnd := windowStart + windowSeconds
	remaining := windowEnd - time.Now().Unix()
	if remaining < 1 {
		return 1
	}
	return remaining
}
