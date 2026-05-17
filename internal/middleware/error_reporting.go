package middleware

import (
	"fmt"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// ErrorReporting returns a middleware that recovers from panics and reports them
// to the provided ErrorReporter. It must be placed outermost in the middleware
// chain so it catches panics from every layer below it. [§5.3]
func ErrorReporting(reporter shared.ErrorReporter) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					var panicErr error
					if e, ok := r.(error); ok {
						panicErr = e
					} else {
						panicErr = fmt.Errorf("panic: %v", r)
					}
					reporter.CaptureException(panicErr)
					err = c.JSON(http.StatusInternalServerError, shared.ErrorResponse{
						Error: shared.ErrorBody{
							Code:    "internal_error",
							Message: "An internal error occurred",
						},
					})
				}
			}()
			return next(c)
		}
	}
}
