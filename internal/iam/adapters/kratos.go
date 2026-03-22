// Package adapters contains external service adapters for the IAM domain.
// Kratos-specific types are isolated here and MUST NOT leak into application-layer code.
// [CODING §8.1, ARCH §4.2]
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/iam"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// KratosAdapterImpl implements both shared.SessionValidator (for auth middleware)
// and iam.KratosAdapter (for service-layer admin operations).
//
// Uses net/http directly — no Kratos SDK dependency. [ARCH §4.2]
type KratosAdapterImpl struct {
	adminURL   string
	publicURL  string
	httpClient *http.Client
}

// NewKratosAdapter creates a new KratosAdapterImpl.
//   - adminURL: internal sidecar admin API (e.g. "http://kratos:4434")
//   - publicURL: browser-facing public API (e.g. "http://kratos:4433")
func NewKratosAdapter(adminURL, publicURL string) *KratosAdapterImpl {
	return &KratosAdapterImpl{
		adminURL:   adminURL,
		publicURL:  publicURL,
		httpClient: &http.Client{},
	}
}

// ─── shared.SessionValidator ─────────────────────────────────────────────────

// ValidateSession validates a Kratos session cookie/token via the public whoami endpoint.
// Returns shared.Session on success; shared.ErrUnauthorized() on 401; wrapped error on failure.
// Implements shared.SessionValidator for auth middleware. [00-core §13.1, 01-iam §11.1]
func (a *KratosAdapterImpl) ValidateSession(ctx context.Context, sessionCookie string) (*shared.Session, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.publicURL+"/sessions/whoami", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}
	req.Header.Set("Cookie", sessionCookie)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, shared.ErrUnauthorized()
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: whoami returned %d", iam.ErrKratosError, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read whoami response", iam.ErrKratosError)
	}

	// kratosWhoamiResponse is unexported — Kratos types never leak beyond this file.
	var whoami kratosWhoamiResponse
	if err := json.Unmarshal(body, &whoami); err != nil {
		return nil, fmt.Errorf("%w: failed to parse whoami response", iam.ErrKratosError)
	}

	if !whoami.Active {
		return nil, shared.ErrUnauthorized()
	}

	identityID, err := uuid.Parse(whoami.Identity.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid identity ID in whoami response", iam.ErrKratosError)
	}

	return &shared.Session{
		IdentityID: identityID,
		Email:      whoami.Identity.Traits.Email, // PII — never log [CODING §5.2]
	}, nil
}

// ─── iam.KratosAdapter ────────────────────────────────────────────────────────

// GetIdentity retrieves identity traits from the Kratos Admin API.
// Implements iam.KratosAdapter. [§7]
func (a *KratosAdapterImpl) GetIdentity(ctx context.Context, identityID uuid.UUID) (*iam.KratosIdentity, error) {
	url := fmt.Sprintf("%s/admin/identities/%s", a.adminURL, identityID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return nil, iam.ErrParentNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: get identity returned %d", iam.ErrKratosError, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read identity response", iam.ErrKratosError)
	}

	var identity kratosIdentityResponse
	if err := json.Unmarshal(body, &identity); err != nil {
		return nil, fmt.Errorf("%w: failed to parse identity response", iam.ErrKratosError)
	}

	id, err := uuid.Parse(identity.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid identity ID in response", iam.ErrKratosError)
	}

	return &iam.KratosIdentity{
		ID:    id,
		Email: identity.Traits.Email, // PII — never log [CODING §5.2]
		Name:  identity.Traits.Name,
	}, nil
}

// DeleteIdentity deletes a Kratos identity. Used during family deletion (Phase 2).
func (a *KratosAdapterImpl) DeleteIdentity(ctx context.Context, identityID uuid.UUID) error {
	url := fmt.Sprintf("%s/admin/identities/%s", a.adminURL, identityID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: delete identity returned %d", iam.ErrKratosError, resp.StatusCode)
	}
	return nil
}

// RevokeSessions revokes all active sessions for a Kratos identity.
// Used when removing a co-parent (Phase 2).
func (a *KratosAdapterImpl) RevokeSessions(ctx context.Context, identityID uuid.UUID) error {
	url := fmt.Sprintf("%s/admin/identities/%s/sessions", a.adminURL, identityID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: revoke sessions returned %d", iam.ErrKratosError, resp.StatusCode)
	}
	return nil
}

// ─── Unexported Kratos Response Types ────────────────────────────────────────
// These types model the Kratos API response JSON.
// They MUST NOT be exported or used outside this file. [ARCH §4.2]

type kratosWhoamiResponse struct {
	Active   bool             `json:"active"`
	Identity kratosIdentityIn `json:"identity"`
}

type kratosIdentityIn struct {
	ID     string        `json:"id"`
	Traits kratosTraitsIn `json:"traits"`
}

type kratosTraitsIn struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type kratosIdentityResponse struct {
	ID     string        `json:"id"`
	Traits kratosTraitsIn `json:"traits"`
}
