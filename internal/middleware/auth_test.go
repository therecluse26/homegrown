package middleware

// Tests for the Auth middleware. Package-internal test file because authDeps
// is an unexported interface — external test packages cannot satisfy it. [§15.6]

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// testAuthDeps is a test double that satisfies the unexported authDeps interface.
type testAuthDeps struct {
	validator shared.SessionValidator
	db        *gorm.DB
}

func (d *testAuthDeps) GetAuthValidator() shared.SessionValidator { return d.validator }
func (d *testAuthDeps) GetDB() *gorm.DB                          { return d.db }

// testSessionValidator is a mock SessionValidator used in auth middleware tests.
type testSessionValidator struct {
	session *shared.Session
	err     error
}

func (v *testSessionValidator) ValidateSession(_ context.Context, _ string) (*shared.Session, error) {
	return v.session, v.err
}

// TestAuth_NoCookie_Returns401 verifies that a request missing the Kratos
// session cookie is rejected immediately with 401. [§15.6]
func TestAuth_NoCookie_Returns401(t *testing.T) {
	e := echo.New()
	deps := &testAuthDeps{
		validator: &testSessionValidator{}, // never called — no cookie short-circuits
	}
	mw := Auth(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	// Intentionally no cookie.
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := mw(func(_ echo.Context) error {
		t.Error("inner handler should not be reached when no cookie is present")
		return nil
	})(c)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertUnauthorized(t, err)
}

// TestAuth_InvalidSession_Returns401 verifies that a valid-looking cookie whose
// Kratos session has expired (or is otherwise rejected) produces a 401. [§15.6]
func TestAuth_InvalidSession_Returns401(t *testing.T) {
	e := echo.New()
	deps := &testAuthDeps{
		validator: &testSessionValidator{
			err: errors.New("kratos: session not found"),
		},
		// db is nil — validation fails before the DB is ever consulted.
	}
	mw := Auth(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: kratosSessionCookieName, Value: "expired-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := mw(func(_ echo.Context) error {
		t.Error("inner handler should not be reached when session is invalid")
		return nil
	})(c)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertUnauthorized(t, err)
}

// assertUnauthorized is a test helper that checks err is an AppError with HTTP 401.
func assertUnauthorized(t *testing.T, err error) {
	t.Helper()
	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *shared.AppError, got %T: %v", err, err)
	}
	if appErr.StatusCode != http.StatusUnauthorized {
		t.Errorf("want HTTP 401 Unauthorized, got %d", appErr.StatusCode)
	}
}
