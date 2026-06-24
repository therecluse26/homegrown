// Package adapters contains external service adapters for the IAM domain.
// Hearth-specific types MUST NOT leak into application-layer code. [CODING §8.1, ARCH §4.2]
package adapters

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	hearthsdk "github.com/hearth-auth/hearth/sdks/go/hearth"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// HearthAdapter implements shared.SessionValidator for the Hearth BFF auth flow.
//
// ValidateSession: sid cookie → session store lookup → local JWT claim decode → shared.Session.
// Zero calls to Hearth in steady state. Signature trust is established at code-exchange time
// (server-to-server) and tokens are stored encrypted in Postgres. [ADR-A, ADR-D]
type HearthAdapter struct {
	client *hearthsdk.Client
	store  iam.SessionStore
}

// NewHearthAdapter creates a HearthAdapter.
//   - client: configured Hearth SDK client (realm pre-set)
//   - store:  AES-256-GCM encrypted Postgres-backed session store
func NewHearthAdapter(client *hearthsdk.Client, store iam.SessionStore) *HearthAdapter {
	return &HearthAdapter{
		client: client,
		store:  store,
	}
}

// Client returns the underlying Hearth SDK client.
// Used by BFF auth endpoints (WS3) for code exchange, token refresh, and logout.
func (a *HearthAdapter) Client() *hearthsdk.Client { return a.client }

// ValidateSession reads the sid cookie value, loads the stored access token,
// decodes JWT claims locally, and returns a populated shared.Session.
// No network call to Hearth in steady state. [ADR-A]
//
// Returns shared.ErrUnauthorized() when:
//   - sid absent from session store (deleted on logout or TTL exceeded)
//   - access token is expired (silent refresh is WS3's responsibility)
//   - JWT claims are malformed or required claims (sub, oid) are missing
func (a *HearthAdapter) ValidateSession(ctx context.Context, sid string) (*shared.Session, error) {
	sess, err := a.store.Get(ctx, sid)
	if err != nil {
		if errors.Is(err, iam.ErrSessionNotFound) {
			return nil, shared.ErrUnauthorized()
		}
		return nil, fmt.Errorf("%w: session store: %v", iam.ErrHearthError, err)
	}

	// Local expiry check — zero Hearth calls.
	// The BFF auth endpoints (WS3) are responsible for proactive silent refresh.
	if time.Now().After(sess.ExpiresAt) {
		return nil, shared.ErrUnauthorized()
	}

	// Decode JWT claims locally. Trust is established at code-exchange time: the
	// backend received this token directly from Hearth (server-to-server) and stored
	// it encrypted. We do not re-verify the Ed25519 signature per request. [ADR-A]
	claims, err := hearthsdk.ParseClaims(sess.AccessToken)
	if err != nil {
		return nil, shared.ErrUnauthorized()
	}

	// sub = Hearth user ID. Maps to iam_parents.hearth_user_id. [ADR-B]
	sub := claims.Subject()
	if sub == "" {
		return nil, shared.ErrUnauthorized()
	}
	// Fast path: HearthUserID is already stored in the session row — use it
	// as a defence-in-depth check against a tampered token.
	claimSubID, parseErr := uuid.Parse(sub)
	if parseErr != nil || claimSubID != sess.HearthUserID {
		return nil, shared.ErrUnauthorized()
	}

	// oid = Hearth org ID = family_id (org-per-family, ADR-B).
	// FamilyScope derived from the JWT — no DB JOIN required. [ADR-B, CODING §2.4]
	// Cross-check against the denormalized family_id stored in the session row.
	oid := claims.OrganizationId()
	if oid == "" {
		return nil, shared.ErrUnauthorized()
	}
	claimOrgID, parseErr := uuid.Parse(oid)
	if parseErr != nil || claimOrgID != sess.FamilyID {
		return nil, shared.ErrUnauthorized()
	}

	// Extract email — PII, never log. [CODING §5.2]
	var email string
	if raw := claims.Get("email"); raw != nil {
		_ = json.Unmarshal(raw, &email) // best-effort; empty string on failure is acceptable
	}

	return &shared.Session{
		IdentityID: sess.HearthUserID, // JWT sub — maps to iam_parents.hearth_user_id
		OrgID:      sess.FamilyID,     // JWT oid — maps to family_id (ADR-B)
		SessionID:  sid,               // opaque sid for current-session detection [15-lifecycle §12]
		Email:      email,             // PII — never log [CODING §5.2]
	}, nil
}
