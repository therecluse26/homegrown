package middleware

import "github.com/labstack/echo/v4"

// SecurityHeaders returns a middleware that sets security-related HTTP response headers.
// These headers are set on every response regardless of route. [ARCH §3.2]
func SecurityHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			// Prevent MIME-type sniffing
			h.Set("X-Content-Type-Options", "nosniff")
			// Prevent clickjacking via iframes
			h.Set("X-Frame-Options", "DENY")
			// Control referrer information
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			// Disable legacy XSS filter (modern browsers don't need it; it can create vulnerabilities)
			h.Set("X-XSS-Protection", "0")
			// HSTS: 2 years, include subdomains, preload-eligible
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
			// CSP: restrict origins, block embedding
			h.Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; "+
					"img-src 'self' data: blob: https://media.homegrown.academy; "+
					"connect-src 'self' wss:; frame-ancestors 'none'")
			// Prevent cross-domain Flash/PDF data loading
			h.Set("X-Permitted-Cross-Domain-Policies", "none")
			return next(c)
		}
	}
}
