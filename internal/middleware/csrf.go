package middleware

import (
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
)

// CSRF returns a middleware that enforces CSRF protection using the double-submit
// cookie pattern. The token is stored in a cookie and the client must send it back
// in the X-CSRF-Token header on state-mutating requests (POST/PUT/PATCH/DELETE).
//
// Webhook endpoints are exempt because they use HMAC signature verification.
// API-key-only routes and safe methods (GET/HEAD/OPTIONS) are also exempt. [CRIT-2]
func CSRF() echo.MiddlewareFunc {
	return echomw.CSRFWithConfig(echomw.CSRFConfig{
		TokenLookup:    "header:X-CSRF-Token",
		CookieName:     "_csrf",
		CookiePath:     "/",
		CookieHTTPOnly: false,                  // Client JS needs to read the cookie value
		CookieSameSite: http.SameSiteStrictMode, // Strict — same-site only
		Skipper: func(c echo.Context) bool {
			// Skip CSRF for webhook endpoints — they use signature verification.
			path := c.Path()
			if strings.HasPrefix(path, "/hooks/") || strings.HasPrefix(path, "/hooks") {
				return true
			}
			// Skip CSRF for the student session endpoint — uses bearer token, not cookies.
			if path == "/v1/students/me" {
				return true
			}
			return false
		},
	})
}
