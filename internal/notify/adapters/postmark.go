package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/notify"
)

// NoopEmailAdapter satisfies notify.EmailAdapter for tests and environments without Postmark.
type NoopEmailAdapter struct{}

func (NoopEmailAdapter) SendTransactional(_ context.Context, _ string, _ string, _ map[string]any) error {
	return nil
}

func (NoopEmailAdapter) SendBatch(_ context.Context, _ []notify.BatchEmailMessage) error {
	return nil
}

func (NoopEmailAdapter) SendBroadcast(_ context.Context, _ string, _ string, _ map[string]any) error {
	return nil
}

// Compile-time interface check.
var _ notify.EmailAdapter = NoopEmailAdapter{}

// ─── PostmarkEmailAdapter ────────────────────────────────────────────────────

// PostmarkEmailAdapter sends email via the Postmark API. [08-notify §7]
type PostmarkEmailAdapter struct {
	serverToken string
	httpClient  *http.Client
}

// NewPostmarkEmailAdapter creates a PostmarkEmailAdapter with the given server token.
func NewPostmarkEmailAdapter(serverToken string) *PostmarkEmailAdapter {
	return &PostmarkEmailAdapter{
		serverToken: serverToken,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Compile-time interface check.
var _ notify.EmailAdapter = (*PostmarkEmailAdapter)(nil)

const postmarkBaseURL = "https://api.postmarkapp.com"

// postmarkTemplateRequest is the request body for the withTemplate endpoint.
type postmarkTemplateRequest struct {
	From          string         `json:"From"`
	To            string         `json:"To"`
	TemplateAlias string         `json:"TemplateAlias"`
	TemplateModel map[string]any `json:"TemplateModel"`
	MessageStream string         `json:"MessageStream,omitempty"`
}

// postmarkBatchItem is a single item in a batch send.
type postmarkBatchItem struct {
	From          string         `json:"From"`
	To            string         `json:"To"`
	TemplateAlias string         `json:"TemplateAlias"`
	TemplateModel map[string]any `json:"TemplateModel"`
}

func (a *PostmarkEmailAdapter) do(ctx context.Context, url string, body any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("postmark: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("postmark: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Postmark-Server-Token", a.serverToken)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("postmark: send request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("postmark: request failed with status %d", resp.StatusCode)
	}
	return nil
}

// SendTransactional sends a single transactional email via Postmark withTemplate. [08-notify §7.1]
func (a *PostmarkEmailAdapter) SendTransactional(ctx context.Context, to, templateAlias string, templateModel map[string]any) error {
	return a.do(ctx, postmarkBaseURL+"/email/withTemplate", postmarkTemplateRequest{
		To:            to,
		TemplateAlias: templateAlias,
		TemplateModel: templateModel,
	})
}

// SendBatch sends multiple transactional emails in one Postmark API call. [08-notify §7.2]
func (a *PostmarkEmailAdapter) SendBatch(ctx context.Context, messages []notify.BatchEmailMessage) error {
	items := make([]postmarkBatchItem, 0, len(messages))
	for _, m := range messages {
		items = append(items, postmarkBatchItem{
			To:            m.To,
			TemplateAlias: m.TemplateAlias,
			TemplateModel: m.TemplateModel,
		})
	}
	return a.do(ctx, postmarkBaseURL+"/email/batchWithTemplates", map[string]any{"Messages": items})
}

// SendBroadcast sends a broadcast email via Postmark using the broadcast message stream. [08-notify §7.3]
func (a *PostmarkEmailAdapter) SendBroadcast(ctx context.Context, to, templateAlias string, templateModel map[string]any) error {
	return a.do(ctx, postmarkBaseURL+"/email/withTemplate", postmarkTemplateRequest{
		To:            to,
		TemplateAlias: templateAlias,
		TemplateModel: templateModel,
		MessageStream: "broadcast",
	})
}
