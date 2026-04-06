package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/media"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ThornSaferAdapter implements media.SafetyScanAdapter using the Thorn Safer API.
// Performs PhotoDNA hash matching for CSAM detection and NCMEC reporting.
//
// Config-gated: only constructed when THORN_API_KEY is non-empty.
// Falls back to NoopSafetyScanAdapter otherwise. [09-media §7.2, 11-safety §8]
type ThornSaferAdapter struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// Compile-time interface check.
var _ media.SafetyScanAdapter = (*ThornSaferAdapter)(nil)

// ThornConfig holds the configuration for the Thorn Safer adapter.
type ThornConfig struct {
	BaseURL string // Thorn Safer API base URL (e.g., "https://safer.thorn.org")
	APIKey  string // Thorn Safer API key
}

// NewThornSaferAdapter constructs a ThornSaferAdapter. Panics if APIKey is empty —
// callers must gate construction on config presence.
func NewThornSaferAdapter(cfg ThornConfig) *ThornSaferAdapter {
	if cfg.APIKey == "" {
		panic("thorn: API key is required — gate adapter creation on config presence")
	}
	return &ThornSaferAdapter{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ScanCSAM scans an uploaded media object for CSAM using Thorn Safer PhotoDNA hash matching.
// The storageKey is the S3 key of the uploaded file. [09-media §7.2]
func (a *ThornSaferAdapter) ScanCSAM(ctx context.Context, storageKey string) (*media.CSAMScanResult, error) {
	body := map[string]any{
		"media_url": storageKey,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("thorn: marshal CSAM scan request: %w", err)
	}

	endpoint := a.baseURL + "/v1/scan"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("thorn: create CSAM scan request: %w", err)
	}
	a.setHeaders(req)

	resp, err := shared.RetryableHTTPDo(ctx, a.client, req, nil)
	if err != nil {
		return nil, fmt.Errorf("thorn: CSAM scan request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("thorn: CSAM scan API error", "status", resp.StatusCode, "body", string(respBody))
		return nil, fmt.Errorf("thorn: CSAM scan HTTP %d", resp.StatusCode)
	}

	var result struct {
		IsCSAM     bool     `json:"is_csam"`
		Hash       *string  `json:"hash"`
		Confidence *float64 `json:"confidence"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("thorn: decode CSAM scan response: %w", err)
	}

	return &media.CSAMScanResult{
		IsCSAM:     result.IsCSAM,
		Hash:       result.Hash,
		Confidence: result.Confidence,
	}, nil
}

// ScanModeration scans for content moderation violations.
// Thorn Safer focuses on CSAM — moderation is a secondary capability.
// Returns clean result if the API does not support moderation scanning. [09-media §7.2]
func (a *ThornSaferAdapter) ScanModeration(ctx context.Context, storageKey string) (*media.ModerationResult, error) {
	body := map[string]any{
		"media_url": storageKey,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("thorn: marshal moderation scan request: %w", err)
	}

	endpoint := a.baseURL + "/v1/moderate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("thorn: create moderation scan request: %w", err)
	}
	a.setHeaders(req)

	resp, err := shared.RetryableHTTPDo(ctx, a.client, req, nil)
	if err != nil {
		return nil, fmt.Errorf("thorn: moderation scan request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("thorn: moderation scan API error", "status", resp.StatusCode, "body", string(respBody))
		return nil, fmt.Errorf("thorn: moderation scan HTTP %d", resp.StatusCode)
	}

	var result struct {
		HasViolations bool `json:"has_violations"`
		AutoReject    bool `json:"auto_reject"`
		Labels        []struct {
			Name       string  `json:"name"`
			Confidence float64 `json:"confidence"`
			ParentName *string `json:"parent_name"`
		} `json:"labels"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("thorn: decode moderation scan response: %w", err)
	}

	labels := make([]media.ModerationLabel, len(result.Labels))
	for i, l := range result.Labels {
		labels[i] = media.ModerationLabel{
			Name:       l.Name,
			Confidence: l.Confidence,
			ParentName: l.ParentName,
		}
	}

	return &media.ModerationResult{
		HasViolations: result.HasViolations,
		AutoReject:    result.AutoReject,
		Labels:        labels,
	}, nil
}

// ReportCSAM reports confirmed/suspected CSAM to NCMEC via the Thorn Safer API.
// Per 18 U.S.C. §2258A, ESPs must report CSAM within a legally mandated timeframe.
func (a *ThornSaferAdapter) ReportCSAM(ctx context.Context, uploadID uuid.UUID, scanResult *media.CSAMScanResult) error {
	body := map[string]any{
		"upload_id":  uploadID.String(),
		"hash":       scanResult.Hash,
		"confidence": scanResult.Confidence,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("thorn: marshal NCMEC report request: %w", err)
	}

	endpoint := a.baseURL + "/v1/report"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("thorn: create NCMEC report request: %w", err)
	}
	a.setHeaders(req)

	resp, err := shared.RetryableHTTPDo(ctx, a.client, req, nil)
	if err != nil {
		return fmt.Errorf("thorn: NCMEC report request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		slog.Error("thorn: NCMEC report API error", "status", resp.StatusCode, "body", string(respBody))
		return fmt.Errorf("thorn: NCMEC report HTTP %d", resp.StatusCode)
	}

	slog.Info("thorn: NCMEC report submitted", "upload_id", uploadID)
	return nil
}

// setHeaders sets common headers for Thorn Safer API requests.
func (a *ThornSaferAdapter) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
}
