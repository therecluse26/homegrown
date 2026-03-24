package media

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// ─── NoopStorageAdapter ──────────────────────────────────────────────────────

// NoopStorageAdapter returns placeholder responses for all storage operations.
// Used in development and test environments. [09-media §7.1]
type NoopStorageAdapter struct{}

// Compile-time interface check.
var _ ObjectStorageAdapter = NoopStorageAdapter{}

func (NoopStorageAdapter) PresignedPut(_ context.Context, key string, _ uint64, _ string, _ uint32) (string, error) {
	slog.Warn("storage not configured — PresignedPut is a no-op", "key", key)
	return "https://placeholder.local/upload/" + key, nil
}

func (NoopStorageAdapter) PresignedGet(_ context.Context, key string, _ uint32) (string, error) {
	slog.Warn("storage not configured — PresignedGet is a no-op", "key", key)
	return "https://placeholder.local/get/" + key, nil
}

func (NoopStorageAdapter) PutObject(_ context.Context, key string, _ []byte, _ string) error {
	slog.Warn("storage not configured — PutObject is a no-op", "key", key)
	return nil
}

func (NoopStorageAdapter) GetObjectHead(_ context.Context, key string) (*ObjectMetadata, error) {
	slog.Warn("storage not configured — GetObjectHead is a no-op", "key", key)
	return &ObjectMetadata{ContentLength: 0}, nil
}

func (NoopStorageAdapter) GetObjectBytes(_ context.Context, key string, _ uint64, _ uint64) ([]byte, error) {
	slog.Warn("storage not configured — GetObjectBytes is a no-op", "key", key)
	return make([]byte, 16), nil
}

func (NoopStorageAdapter) DeleteObject(_ context.Context, key string) error {
	slog.Warn("storage not configured — DeleteObject is a no-op", "key", key)
	return nil
}

// ─── NoopSafetyScanAdapter ───────────────────────────────────────────────────

// NoopSafetyScanAdapter returns clean/clear results for all safety operations.
// Used until the safety:: domain is implemented. [09-media §7.2]
type NoopSafetyScanAdapter struct{}

// Compile-time interface check.
var _ SafetyScanAdapter = NoopSafetyScanAdapter{}

func (NoopSafetyScanAdapter) ScanCSAM(_ context.Context, _ string) (*CSAMScanResult, error) {
	return &CSAMScanResult{IsCSAM: false}, nil
}

func (NoopSafetyScanAdapter) ScanModeration(_ context.Context, _ string) (*ModerationResult, error) {
	return &ModerationResult{HasViolations: false, AutoReject: false}, nil
}

func (NoopSafetyScanAdapter) ReportCSAM(_ context.Context, _ uuid.UUID, _ *CSAMScanResult) error {
	slog.Warn("safety scan not configured — ReportCSAM is a no-op")
	return nil
}
