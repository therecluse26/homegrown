// Package adapters contains external service adapters for the IAM domain.
// Kratos-specific types are isolated here and MUST NOT leak into application-layer code.
// [CODING §8.1, ARCH §4.2]
package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

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
		httpClient: &http.Client{Timeout: 30 * time.Second},
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
		SessionID:  whoami.ID, // Kratos session ID — for current-session detection [15-lifecycle §12]
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

// ListSessionsForIdentity returns all active sessions for a Kratos identity.
// Calls GET /admin/identities/{identityID}/sessions. [15-data-lifecycle §12]
func (a *KratosAdapterImpl) ListSessionsForIdentity(ctx context.Context, identityID uuid.UUID) ([]iam.KratosAdminSession, error) {
	url := fmt.Sprintf("%s/admin/identities/%s/sessions?active=true", a.adminURL, identityID.String())
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
		return []iam.KratosAdminSession{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: list sessions returned %d", iam.ErrKratosError, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read sessions response", iam.ErrKratosError)
	}

	var raw []kratosAdminSessionIn
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: failed to parse sessions response", iam.ErrKratosError)
	}

	sessions := make([]iam.KratosAdminSession, 0, len(raw))
	for _, s := range raw {
		sess := iam.KratosAdminSession{
			SessionID:  s.ID,
			LastActive: s.AuthenticatedAt,
		}
		if len(s.Devices) > 0 {
			sess.UserAgent = &s.Devices[0].UserAgent
			sess.IPAddress = &s.Devices[0].IPAddress
		}
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

// RevokeSpecificSession revokes a single Kratos session by session ID.
// Calls DELETE /admin/sessions/{sessionID}. [15-data-lifecycle §12]
func (a *KratosAdapterImpl) RevokeSpecificSession(ctx context.Context, sessionID string) error {
	url := fmt.Sprintf("%s/admin/sessions/%s", a.adminURL, sessionID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return nil // already revoked — idempotent
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: revoke session returned %d", iam.ErrKratosError, resp.StatusCode)
	}
	return nil
}

// InitiateAccountRecovery triggers a Kratos recovery flow for the given email address.
// Calls POST /self-service/recovery/api then submits the email.
// Email enumeration prevention is the caller's responsibility. [15-data-lifecycle §13]
func (a *KratosAdapterImpl) InitiateAccountRecovery(ctx context.Context, email string) error {
	// Step 1: create a recovery flow via the public API.
	flowReq, err := http.NewRequestWithContext(ctx, http.MethodGet, a.publicURL+"/self-service/recovery/api", nil)
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}

	flowResp, err := a.httpClient.Do(flowReq)
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}
	defer flowResp.Body.Close() //nolint:errcheck

	if flowResp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: create recovery flow returned %d", iam.ErrKratosError, flowResp.StatusCode)
	}

	flowBody, err := io.ReadAll(flowResp.Body)
	if err != nil {
		return fmt.Errorf("%w: failed to read recovery flow response", iam.ErrKratosError)
	}

	var flow kratosFlowIn
	if err := json.Unmarshal(flowBody, &flow); err != nil {
		return fmt.Errorf("%w: failed to parse recovery flow response", iam.ErrKratosError)
	}

	// Step 2: submit the email address to trigger the recovery email.
	submitURL := fmt.Sprintf("%s/self-service/recovery?flow=%s", a.publicURL, flow.ID)
	body, err := json.Marshal(map[string]string{"email": email, "method": "link"})
	if err != nil {
		return fmt.Errorf("%w: failed to marshal recovery submit body", iam.ErrKratosError)
	}

	submitReq, err := http.NewRequestWithContext(ctx, http.MethodPost, submitURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}
	submitReq.Header.Set("Content-Type", "application/json")
	submitReq.Header.Set("Accept", "application/json")

	submitResp, err := a.httpClient.Do(submitReq)
	if err != nil {
		return fmt.Errorf("%w: %v", iam.ErrKratosError, err)
	}
	defer submitResp.Body.Close() //nolint:errcheck

	// 200 = recovery email sent; 400/422 = validation error (email not found is NOT an error — enumeration prevention)
	if submitResp.StatusCode != http.StatusOK && submitResp.StatusCode != http.StatusUnprocessableEntity {
		return fmt.Errorf("%w: recovery submit returned %d", iam.ErrKratosError, submitResp.StatusCode)
	}
	return nil
}

// ─── Unexported Kratos Response Types ────────────────────────────────────────
// These types model the Kratos API response JSON.
// They MUST NOT be exported or used outside this file. [ARCH §4.2]

type kratosWhoamiResponse struct {
	ID       string           `json:"id"` // Kratos session ID
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
	ID     string         `json:"id"`
	Traits kratosTraitsIn `json:"traits"`
}

type kratosAdminSessionDevice struct {
	UserAgent string `json:"user_agent"`
	IPAddress string `json:"ip_address"`
}

type kratosAdminSessionIn struct {
	ID              string                     `json:"id"`
	AuthenticatedAt time.Time                  `json:"authenticated_at"`
	Devices         []kratosAdminSessionDevice `json:"devices"`
}

type kratosFlowIn struct {
	ID string `json:"id"`
}
