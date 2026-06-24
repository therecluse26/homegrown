package iam

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	hearthsdk "github.com/hearth-auth/hearth/sdks/go/hearth"
	"github.com/labstack/echo/v4"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// bffSidCookieName is the HttpOnly session cookie name for the BFF pattern.
// Must match sidCookieName in internal/middleware/auth.go. [ARCH ADR-020]
const bffSidCookieName = "sid"

// bffStateCookieName stores the PKCE state+verifier during the OAuth redirect. [ARCH ADR-020]
const bffStateCookieName = "_auth_state"

// pkceStateTTL is how long the PKCE state cookie lives before the flow must restart.
const pkceStateTTL = 10 * time.Minute

// AuthHandler handles BFF auth endpoints for the Hearth OIDC flow. [ARCH ADR-020]
type AuthHandler struct {
	svc           IamService
	store         SessionStore
	client        *hearthsdk.Client
	clientID      string // PKCE public client UUID (stable UUID-v5 from app key)
	callbackURL   string // absolute callback URL e.g. "http://localhost:3500/v1/auth/callback"
	frontendURL   string // SPA base URL for post-login redirect e.g. "http://localhost:5673"
	hearthBaseURL string // Hearth public URL for server-side API calls (revoke, etc.)
	realmSlug     string // realm slug for browser-facing URL paths (e.g. "homegrown")
	realmID       string // realm UUID for X-Realm-ID headers on server-side calls
	webhookSecret string // HMAC-SHA256 secret for Hearth signed webhooks
	secure        bool   // set true in prod (HTTPS only)
}

// NewAuthHandler creates an AuthHandler.
// realmSlug is used in browser redirect URL paths; realmID (UUID) is used in X-Realm-ID headers.
func NewAuthHandler(
	svc IamService,
	store SessionStore,
	client *hearthsdk.Client,
	clientID, callbackURL, frontendURL, hearthBaseURL, realmSlug, realmID, webhookSecret string,
	secure bool,
) *AuthHandler {
	return &AuthHandler{
		svc:           svc,
		store:         store,
		client:        client,
		clientID:      clientID,
		callbackURL:   callbackURL,
		frontendURL:   frontendURL,
		hearthBaseURL: hearthBaseURL,
		realmSlug:     realmSlug,
		realmID:       realmID,
		webhookSecret: webhookSecret,
		secure:        secure,
	}
}

// Register mounts BFF auth routes. [ARCH ADR-020, §10.1]
//   - pub: public group under /v1 (no auth middleware)
//   - hooks: webhook group under /hooks
func (h *AuthHandler) Register(pub *echo.Group, hooks *echo.Group) {
	pub.GET("/auth/login", h.login)
	pub.GET("/auth/callback", h.callback)
	pub.POST("/auth/register", h.register)
	pub.POST("/auth/refresh", h.refresh)
	pub.POST("/auth/logout", h.logout)
	hooks.POST("/hearth/webhook", h.hearthWebhook)
}

// ─── GET /v1/auth/login ───────────────────────────────────────────────────────

// login initiates the PAR-based PKCE login flow. [§A2, ARCH ADR-020, RFC 9126]
//
// Two-step flow:
//  1. BFF POSTs PKCE params to Hearth /as/par (server-side, X-Realm-ID: <realm-uuid>)
//     → Hearth returns {"request_uri": "urn:..."}
//  2. BFF redirects the browser to /ui/login?request_uri=<value>&client_id=<uuid>
func (h *AuthHandler) login(c echo.Context) error {
	pkce, err := hearthsdk.GeneratePKCE()
	if err != nil {
		return shared.ErrInternal(err)
	}

	// Generate opaque state to prevent CSRF.
	stateBuf := make([]byte, 24)
	if _, err := rand.Read(stateBuf); err != nil {
		return shared.ErrInternal(err)
	}
	state := hex.EncodeToString(stateBuf)

	// Store verifier + state server-side in a short-lived cookie.
	// Format: "<state>:<verifier>"
	stateValue := state + ":" + pkce.Verifier
	c.SetCookie(&http.Cookie{
		Name:     bffStateCookieName,
		Value:    stateValue,
		Path:     "/",
		MaxAge:   int(pkceStateTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode, // Lax so the redirect back works across origins
	})

	// Step 1: POST authorization params to /as/par (RFC 9126). [ARCH ADR-020]
	// X-Realm-ID requires the realm UUID for all server-side Hearth API calls.
	parBody := url.Values{}
	parBody.Set("response_type", "code")
	parBody.Set("client_id", h.clientID)
	parBody.Set("redirect_uri", h.callbackURL)
	parBody.Set("scope", "openid email profile offline_access")
	parBody.Set("state", state)
	parBody.Set("code_challenge", pkce.Challenge)
	parBody.Set("code_challenge_method", pkce.Method)

	parReq, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost,
		h.hearthBaseURL+"/as/par", strings.NewReader(parBody.Encode()))
	if err != nil {
		return shared.ErrInternal(err)
	}
	parReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	parReq.Header.Set("X-Realm-ID", h.realmID)

	parResp, err := http.DefaultClient.Do(parReq)
	if err != nil {
		slog.Error("hearth PAR request failed", "error", err)
		return shared.ErrInternal(err)
	}
	defer parResp.Body.Close() //nolint:errcheck

	if parResp.StatusCode >= 400 {
		_, _ = io.Copy(io.Discard, parResp.Body)
		slog.Error("hearth PAR error", "status", parResp.StatusCode)
		return shared.ErrInternal(fmt.Errorf("hearth PAR returned HTTP %d", parResp.StatusCode)) //nolint:goerr113
	}

	var parResult struct {
		RequestURI string `json:"request_uri"`
	}
	if err := json.NewDecoder(parResp.Body).Decode(&parResult); err != nil {
		return shared.ErrInternal(err)
	}
	if parResult.RequestURI == "" {
		return shared.ErrInternal(fmt.Errorf("hearth PAR: empty request_uri in response")) //nolint:goerr113
	}

	// Step 2: Redirect browser to /ui/login with the opaque request_uri. [RFC 9126, ARCH ADR-020]
	loginURL, err := url.Parse(h.hearthBaseURL + "/ui/login")
	if err != nil {
		return shared.ErrInternal(err)
	}
	q := loginURL.Query()
	q.Set("request_uri", parResult.RequestURI)
	q.Set("client_id", h.clientID)
	loginURL.RawQuery = q.Encode()

	return c.Redirect(http.StatusFound, loginURL.String())
}

// ─── GET /v1/auth/callback ────────────────────────────────────────────────────

// callback exchanges the PKCE authorization code for tokens and creates the BFF session.
func (h *AuthHandler) callback(c echo.Context) error {
	// Validate state.
	stateCookie, err := c.Request().Cookie(bffStateCookieName)
	if err != nil {
		return shared.ErrBadRequest("missing auth state cookie")
	}
	parts := strings.SplitN(stateCookie.Value, ":", 2)
	if len(parts) != 2 {
		return shared.ErrBadRequest("malformed auth state cookie")
	}
	savedState, verifier := parts[0], parts[1]

	queryState := c.QueryParam("state")
	if queryState == "" || queryState != savedState {
		return shared.ErrBadRequest("state mismatch — possible CSRF")
	}
	code := c.QueryParam("code")
	if code == "" {
		errParam := c.QueryParam("error")
		if errParam != "" {
			slog.Warn("hearth auth callback error", "error", errParam)
		}
		return shared.ErrBadRequest("missing authorization code")
	}

	// Clear state cookie.
	c.SetCookie(&http.Cookie{
		Name:     bffStateCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteLaxMode,
	})

	// Exchange code for tokens via Hearth token endpoint.
	tokenResp, err := h.client.ExchangeCode(c.Request().Context(), hearthsdk.TokenRequest{
		ClientID:     h.clientID,
		GrantType:    "authorization_code",
		Code:         code,
		RedirectURI:  h.callbackURL,
		CodeVerifier: verifier,
	})
	if err != nil {
		slog.Error("hearth token exchange failed", "error", err)
		return shared.ErrUnauthorized()
	}

	// Parse expiry from ExpiresIn.
	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Parse claims to get hearth_user_id and family_id for session record.
	claims, err := hearthsdk.ParseClaims(tokenResp.AccessToken)
	if err != nil {
		slog.Error("hearth callback: failed to parse access token claims", "error", err)
		return shared.ErrUnauthorized()
	}
	// Hearth JWT sub is prefixed with "user_" — strip before UUID parsing. [ARCH ADR-020]
	rawSub := strings.TrimPrefix(claims.Subject(), "user_")
	hearthUserID, uErr := parseUUID(rawSub)
	if uErr != nil {
		slog.Error("hearth callback: invalid sub claim", "sub", claims.Subject())
		return shared.ErrUnauthorized()
	}
	// Hearth JWT oid is the org/family UUID. It is only populated by Hearth when the
	// authorization request includes organization_id — which we cannot know before login.
	// Fallback: look up family_id from app DB using hearth_user_id. [ARCH ADR-020]
	var familyID uuid.UUID
	rawOid := claims.OrganizationId()
	if rawOid != "" {
		var fErr error
		familyID, fErr = parseUUID(rawOid)
		if fErr != nil {
			slog.Error("hearth callback: invalid oid claim", "oid", rawOid)
			return shared.ErrUnauthorized()
		}
	} else {
		var lookupErr error
		familyID, lookupErr = h.svc.GetFamilyIDByHearthUserID(c.Request().Context(), hearthUserID)
		if lookupErr != nil {
			slog.Error("hearth callback: family not found for hearth user", "hearth_user_id", hearthUserID)
			return shared.ErrUnauthorized()
		}
	}

	// Create server-side BFF session. Tokens stored encrypted. [ARCH ADR-020]
	sid, err := h.store.Create(c.Request().Context(), CreateServerSession{
		HearthUserID: hearthUserID,
		FamilyID:     familyID,
		AccessToken:  tokenResp.AccessToken,  // NEVER logged
		RefreshToken: tokenResp.RefreshToken, // NEVER logged
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		slog.Error("hearth callback: failed to create BFF session", "error", err)
		return shared.ErrInternal(err)
	}

	// Set HttpOnly sid cookie — browser never sees OAuth tokens. [ARCH ADR-020]
	c.SetCookie(&http.Cookie{
		Name:     bffSidCookieName,
		Value:    sid,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})

	return c.Redirect(http.StatusFound, h.frontendURL)
}

// ─── POST /v1/auth/register ───────────────────────────────────────────────────

// register performs app-orchestrated registration. [§10.1, ARCH ADR-019]
//
// @Summary     Register a new family account
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       body body RegisterCommand true "Registration details"
// @Success     201 {object} map[string]string
// @Failure     400 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /auth/register [post]
func (h *AuthHandler) register(c echo.Context) error {
	var cmd RegisterCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ErrValidation(err.Error())
	}

	_, err := h.svc.Register(c.Request().Context(), cmd)
	if err != nil {
		return mapIamError(err)
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Registration successful. Please log in.",
	})
}

// ─── POST /v1/auth/refresh ────────────────────────────────────────────────────

// refresh silently refreshes the access token using the stored refresh token.
// The browser never sees the OAuth tokens — only the sid cookie is updated.
func (h *AuthHandler) refresh(c echo.Context) error {
	sidCookie, err := c.Request().Cookie(bffSidCookieName)
	if err != nil || sidCookie.Value == "" {
		return shared.ErrUnauthorized()
	}
	sid := sidCookie.Value

	sess, err := h.store.Get(c.Request().Context(), sid)
	if err != nil {
		return shared.ErrUnauthorized()
	}

	newTokens, err := h.client.RefreshTokens(c.Request().Context(), h.clientID, sess.RefreshToken)
	if err != nil {
		slog.Warn("bff refresh: token refresh failed", "error", err)
		// Clear stale session and cookie.
		_ = h.store.Delete(c.Request().Context(), sid)
		clearSidCookie(c, h.secure)
		return shared.ErrUnauthorized()
	}

	expiresAt := time.Now().Add(time.Duration(newTokens.ExpiresIn) * time.Second)
	if err := h.store.UpdateTokens(c.Request().Context(), sid, newTokens.AccessToken, newTokens.RefreshToken, expiresAt); err != nil {
		slog.Error("bff refresh: failed to update session store", "error", err)
		return shared.ErrInternal(err)
	}

	return c.NoContent(http.StatusNoContent)
}

// ─── POST /v1/auth/logout ─────────────────────────────────────────────────────

// logout revokes the refresh token (RFC 7009), deletes the BFF session, and clears the cookie.
func (h *AuthHandler) logout(c echo.Context) error {
	sidCookie, err := c.Request().Cookie(bffSidCookieName)
	if err != nil || sidCookie.Value == "" {
		// Already logged out — idempotent.
		return c.NoContent(http.StatusNoContent)
	}
	sid := sidCookie.Value
	ctx := c.Request().Context()

	// Attempt to revoke the refresh token at Hearth (RFC 7009). Best-effort.
	if sess, getErr := h.store.Get(ctx, sid); getErr == nil {
		if revokeErr := h.revokeToken(ctx, sess.RefreshToken); revokeErr != nil {
			slog.Warn("bff logout: token revocation failed (proceeding)", "error", revokeErr)
		}
	}

	// Delete server-side session regardless of revocation result.
	_ = h.store.Delete(ctx, sid)
	clearSidCookie(c, h.secure)

	return c.NoContent(http.StatusNoContent)
}

// revokeToken calls the Hearth revocation endpoint (RFC 7009). [ARCH ADR-020]
func (h *AuthHandler) revokeToken(ctx context.Context, refreshToken string) error {
	// NEVER log refreshToken. [CODING §5.2]
	body := url.Values{
		"token":           {refreshToken},
		"token_type_hint": {"refresh_token"},
		"client_id":       {h.clientID},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		h.hearthBaseURL+"/revoke",
		strings.NewReader(body.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Realm-ID", h.realmID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("revoke: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// ─── POST /hooks/hearth/webhook ───────────────────────────────────────────────

// hearthWebhook handles Hearth signed webhook events (user.* backstop). [§10.1 ADR-019]
func (h *AuthHandler) hearthWebhook(c echo.Context) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return shared.ErrBadRequest("failed to read body")
	}

	// Verify HMAC-SHA256 signature. [§10.1]
	sig := c.Request().Header.Get("X-Hearth-Signature")
	if !verifyHMACSignature(body, sig, h.webhookSecret) {
		slog.Warn("hearth webhook: signature verification failed")
		return shared.ErrUnauthorized()
	}

	// Event type — just log for now; out-of-band changes are handled as backstop.
	eventType := c.Request().Header.Get("X-Hearth-Event")
	slog.Info("hearth webhook received", "event_type", eventType)

	return c.NoContent(http.StatusNoContent)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func clearSidCookie(c echo.Context, secure bool) {
	c.SetCookie(&http.Cookie{
		Name:     bffSidCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func verifyHMACSignature(body []byte, sig, secret string) bool {
	if sig == "" || secret == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(sig), []byte(expected))
}
