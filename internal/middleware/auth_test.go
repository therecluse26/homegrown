package middleware

// Tests for the Auth middleware. Package-internal test file because authDeps
// is an unexported interface — external test packages cannot satisfy it. [§15.6]

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
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

// TestAuth_NoCookie_Returns401 verifies that a request missing the sid cookie
// is rejected immediately with 401. [§15.6, ADR-D]
func TestAuth_NoCookie_Returns401(t *testing.T) {
	e := echo.New()
	deps := &testAuthDeps{
		validator: &testSessionValidator{}, // never called — no cookie short-circuits
	}
	mw := Auth(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	// Intentionally no sid cookie.
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

// TestAuth_OldKratosCookie_Returns401 verifies that the legacy Kratos cookie is
// not accepted — the middleware reads only the sid cookie now. [ADR-D]
func TestAuth_OldKratosCookie_Returns401(t *testing.T) {
	e := echo.New()
	deps := &testAuthDeps{
		validator: &testSessionValidator{},
	}
	mw := Auth(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: "ory_kratos_session", Value: "some-kratos-token"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := mw(func(_ echo.Context) error {
		t.Error("inner handler should not be reached when kratos cookie is used")
		return nil
	})(c)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertUnauthorized(t, err)
}

// TestAuth_InvalidSession_Returns401 verifies that a valid-looking sid cookie
// whose session is rejected by the validator produces a 401. [§15.6]
func TestAuth_InvalidSession_Returns401(t *testing.T) {
	e := echo.New()
	deps := &testAuthDeps{
		validator: &testSessionValidator{
			err: shared.ErrUnauthorized(),
		},
		// db is nil — validation fails before the DB is ever consulted.
	}
	mw := Auth(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: sidCookieName, Value: "expired-or-invalid-sid"})
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

// TestAuth_ValidSession_NilOrgID_Returns401 verifies that a session with a zero OrgID
// (missing oid claim) is rejected — the family scope invariant requires oid. [ADR-B]
func TestAuth_ValidSession_NilOrgID_Returns401(t *testing.T) {
	e := echo.New()
	deps := &testAuthDeps{
		validator: &testSessionValidator{
			session: &shared.Session{
				IdentityID: uuid.New(),
				OrgID:      uuid.Nil, // missing oid — should be rejected
				SessionID:  "test-sid",
				Email:      "test@example.com",
			},
		},
	}
	mw := Auth(deps)

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: sidCookieName, Value: "some-sid"})
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := mw(func(_ echo.Context) error {
		t.Error("inner handler should not be reached when OrgID is nil")
		return nil
	})(c)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	assertUnauthorized(t, err)
}

// ─── JWT minting helpers for adapter tests ────────────────────────────────────

// mintTestJWT creates a structurally valid EdDSA JWT signed with key.
// Used to test the HearthAdapter in isolation. The hearth.ParseClaims() call
// in the adapter does NOT verify the signature, so any Ed25519 key works.
func mintTestJWT(t *testing.T, key ed25519.PrivateKey, claims map[string]any) string {
	t.Helper()

	header := map[string]string{"alg": "EdDSA", "typ": "JWT"}
	hdrJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("mintTestJWT: marshal header: %v", err)
	}
	payJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("mintTestJWT: marshal payload: %v", err)
	}

	hdrB64 := base64.RawURLEncoding.EncodeToString(hdrJSON)
	payB64 := base64.RawURLEncoding.EncodeToString(payJSON)
	sigInput := hdrB64 + "." + payB64

	sig := ed25519.Sign(key, []byte(sigInput))
	sigB64 := base64.RawURLEncoding.EncodeToString(sig)

	return sigInput + "." + sigB64
}

// TestMintTestJWT verifies that mintTestJWT produces a structurally valid JWT
// (three dot-separated base64url segments). The hearth SDK parses this format. [ADR-A]
func TestMintTestJWT(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	familyID := uuid.New()
	userID := uuid.New()
	jwt := mintTestJWT(t, priv, map[string]any{
		"sub":   userID.String(),
		"oid":   familyID.String(),
		"email": "parent@example.com",
		"exp":   fmt.Sprintf("%d", time.Now().Add(15*time.Minute).Unix()),
	})

	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		t.Fatalf("expected 3 JWT segments, got %d", len(parts))
	}
	for i, p := range parts {
		if p == "" {
			t.Errorf("JWT segment %d is empty", i)
		}
		if _, err := base64.RawURLEncoding.DecodeString(p); err != nil {
			t.Errorf("JWT segment %d is not valid base64url: %v", i, err)
		}
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

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
