package middleware

// Tests for role-extractor helpers. Package-internal for consistency with auth_test.go.

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// TestRequirePremium_FreeTier_Returns402 verifies that a free-tier user is
// rejected with HTTP 402 Payment Required. [§15.5]
func TestRequirePremium_FreeTier_Returns402(t *testing.T) {
	c := newContextWithAuth(t, &shared.AuthContext{
		SubscriptionTier: shared.SubscriptionTierFree,
	})

	_, err := RequirePremium(c)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var appErr *shared.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("want *shared.AppError, got %T: %v", err, err)
	}
	if appErr.StatusCode != http.StatusPaymentRequired {
		t.Errorf("want HTTP 402 Payment Required, got %d", appErr.StatusCode)
	}
	if appErr.Code != "premium_required" {
		t.Errorf("want error code 'premium_required', got %q", appErr.Code)
	}
}

// TestRequirePremium_PremiumTier_Succeeds verifies that a premium user passes
// the extractor and receives the AuthContext back unchanged.
func TestRequirePremium_PremiumTier_Succeeds(t *testing.T) {
	want := &shared.AuthContext{
		SubscriptionTier: shared.SubscriptionTierPremium,
	}
	c := newContextWithAuth(t, want)

	got, err := RequirePremium(c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Error("returned AuthContext pointer does not match the one stored on the context")
	}
}

// newContextWithAuth creates a minimal Echo context with the given AuthContext set.
func newContextWithAuth(t *testing.T, auth *shared.AuthContext) echo.Context {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	shared.SetAuthContext(c, auth)
	return c
}
