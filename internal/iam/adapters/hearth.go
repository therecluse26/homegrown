// Package adapters contains external service adapters for the IAM domain.
// Hearth-specific types MUST NOT leak into application-layer code. [CODING §8.1, ARCH §4.2]
package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	hearthsdk "github.com/hearth-auth/hearth/sdks/go/hearth"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// HearthAdapterImpl implements iam.HearthAdapter (admin ops) and shared.SessionValidator
// (BFF session validation for auth middleware).
//
// Admin ops use the Hearth SDK admin client authenticated with a static admin token.
// ValidateSession: sid cookie → session store → local JWT claim decode → shared.Session.
// Zero steady-state calls to Hearth per request. [ARCH ADR-017, ADR-020]
type HearthAdapterImpl struct {
	client     *hearthsdk.Client
	adminURL   string
	realmID    string
	adminToken string // static service-account token (HEARTH_ADMIN_TOKEN)
	clientID   string // PKCE public client ID e.g. "homegrown-spa"
	store      iam.SessionStore
	httpClient *http.Client
}

// NewHearthAdapter creates a HearthAdapterImpl.
func NewHearthAdapter(
	client *hearthsdk.Client,
	adminURL, realmID, adminToken, clientID string,
	store iam.SessionStore,
) *HearthAdapterImpl {
	return &HearthAdapterImpl{
		client:     client,
		adminURL:   adminURL,
		realmID:    realmID,
		adminToken: adminToken,
		clientID:   clientID,
		store:      store,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// Client returns the underlying Hearth SDK client for use by BFF auth endpoints.
func (a *HearthAdapterImpl) Client() *hearthsdk.Client { return a.client }

// ─── iam.HearthAdapter ────────────────────────────────────────────────────────

func (a *HearthAdapterImpl) CreateUser(ctx context.Context, email, displayName string) (uuid.UUID, error) {
	user, err := a.client.Admin(a.adminToken).CreateUser(ctx, hearthsdk.CreateUserRequest{
		Email:       email,
		DisplayName: displayName,
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: CreateUser: %v", iam.ErrHearthError, err)
	}
	id, err := uuid.Parse(user.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: CreateUser: invalid user ID %q", iam.ErrHearthError, user.ID)
	}
	return id, nil
}

func (a *HearthAdapterImpl) CreateOrgForFamily(ctx context.Context, familyDisplayName string) (uuid.UUID, error) {
	// Hearth SDK admin client has no CreateOrg — call the admin HTTP API directly.
	body, err := json.Marshal(map[string]string{"name": familyDisplayName})
	if err != nil {
		return uuid.Nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.adminURL+"/admin/orgs", bytes.NewReader(body))
	if err != nil {
		return uuid.Nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Realm-ID", a.realmID)
	req.Header.Set("Authorization", "Bearer "+a.adminToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: CreateOrgForFamily: %v", iam.ErrHearthError, err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode >= 400 {
		return uuid.Nil, fmt.Errorf("%w: CreateOrgForFamily: HTTP %d", iam.ErrHearthError, resp.StatusCode)
	}
	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return uuid.Nil, fmt.Errorf("%w: CreateOrgForFamily: decode response: %v", iam.ErrHearthError, err)
	}
	id, err := uuid.Parse(result.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: CreateOrgForFamily: invalid org ID %q", iam.ErrHearthError, result.ID)
	}
	return id, nil
}

func (a *HearthAdapterImpl) AssignUserToOrg(ctx context.Context, hearthUserID, hearthOrgID uuid.UUID) error {
	_, err := a.client.Admin(a.adminToken).AddOrgMember(ctx, hearthOrgID.String(), hearthsdk.AddOrgMemberRequest{
		UserID: hearthUserID.String(),
	})
	if err != nil {
		return fmt.Errorf("%w: AssignUserToOrg: %v", iam.ErrHearthError, err)
	}
	return nil
}

func (a *HearthAdapterImpl) RemoveUserFromOrg(ctx context.Context, hearthUserID, hearthOrgID uuid.UUID) error {
	if err := a.client.Admin(a.adminToken).RemoveOrgMember(ctx, hearthOrgID.String(), hearthUserID.String()); err != nil {
		return fmt.Errorf("%w: RemoveUserFromOrg: %v", iam.ErrHearthError, err)
	}
	return nil
}

func (a *HearthAdapterImpl) DeleteUser(ctx context.Context, hearthUserID uuid.UUID) error {
	if err := a.client.Admin(a.adminToken).DeleteUser(ctx, hearthUserID.String()); err != nil {
		return fmt.Errorf("%w: DeleteUser: %v", iam.ErrHearthError, err)
	}
	return nil
}

func (a *HearthAdapterImpl) RevokeUserSessions(ctx context.Context, hearthUserID uuid.UUID) error {
	// Not in the SDK — call the admin HTTP API directly.
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		fmt.Sprintf("%s/admin/users/%s/sessions/revoke", a.adminURL, hearthUserID.String()), nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Realm-ID", a.realmID)
	req.Header.Set("Authorization", "Bearer "+a.adminToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: RevokeUserSessions: %v", iam.ErrHearthError, err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode >= 400 && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("%w: RevokeUserSessions: HTTP %d", iam.ErrHearthError, resp.StatusCode)
	}
	return nil
}

// ─── shared.SessionValidator ──────────────────────────────────────────────────

// ValidateSession reads the sid, loads the stored access token, silently refreshes
// if within 60 s of expiry, decodes JWT claims locally, and returns a shared.Session.
// Zero steady-state calls to Hearth per request. [ARCH ADR-017, ADR-020, §11.1]
func (a *HearthAdapterImpl) ValidateSession(ctx context.Context, sid string) (*shared.Session, error) {
	sess, err := a.store.Get(ctx, sid)
	if err != nil {
		if errors.Is(err, iam.ErrSessionNotFound) {
			return nil, shared.ErrUnauthorized()
		}
		return nil, fmt.Errorf("%w: session store: %v", iam.ErrHearthError, err)
	}

	accessToken := sess.AccessToken

	// Silent refresh when within 60 s of expiry. [§11.1 step 3]
	if time.Until(sess.ExpiresAt) < 60*time.Second {
		newTokens, refreshErr := a.client.RefreshTokens(ctx, a.clientID, sess.RefreshToken)
		if refreshErr == nil {
			expiresAt := time.Now().Add(time.Duration(newTokens.ExpiresIn) * time.Second)
			if updateErr := a.store.UpdateTokens(ctx, sid, newTokens.AccessToken, newTokens.RefreshToken, expiresAt); updateErr == nil {
				accessToken = newTokens.AccessToken
			}
		} else if time.Now().After(sess.ExpiresAt) {
			// Expired AND refresh failed — token unrecoverable.
			return nil, shared.ErrUnauthorized()
		}
		// Refresh failed but token still valid — proceed with existing token.
	}

	// Decode JWT claims locally — trust established at server-side code exchange. [ADR-017]
	claims, err := hearthsdk.ParseClaims(accessToken)
	if err != nil {
		return nil, shared.ErrUnauthorized()
	}

	// sub = Hearth user ID → iam_parents.hearth_user_id [ADR-018]
	// Hearth prefixes sub with "user_" — strip before UUID parsing. [ARCH ADR-020]
	sub := strings.TrimPrefix(claims.Subject(), "user_")
	if sub == "" {
		return nil, shared.ErrUnauthorized()
	}
	identityID, err := uuid.Parse(sub)
	if err != nil {
		return nil, shared.ErrUnauthorized()
	}

	// family_id = org/family UUID. Use the value stored in the BFF session (set during
	// callback) rather than re-reading the JWT oid claim. Hearth v0.1.0 only populates
	// oid when the authorize request carries organization_id, which login doesn't have.
	// Authoritative source: session store. Zero-phone-home preserved. [ARCH ADR-020]
	orgID := sess.FamilyID
	if orgID == uuid.Nil {
		return nil, shared.ErrUnauthorized()
	}

	// email — PII, never log. [CODING §5.2]
	var email string
	if raw := claims.Get("email"); raw != nil {
		_ = json.Unmarshal(raw, &email) // best-effort
	}

	return &shared.Session{
		IdentityID: identityID, // JWT sub → hearth_user_id [ADR-018]
		OrgID:      orgID,      // JWT oid → family_id [ADR-018]
		SessionID:  sid,        // BFF sid cookie value [15-lifecycle §12]
		Email:      email,      // PII — never log [CODING §5.2]
	}, nil
}
