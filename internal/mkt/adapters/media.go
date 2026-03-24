package adapters

import (
	"context"
	"log/slog"

	"github.com/homegrown-academy/homegrown-academy/internal/mkt"
)

// noopMediaAdapter returns placeholder responses for all media operations.
// Used until a dedicated media service is implemented. [07-mkt §7]
type noopMediaAdapter struct{}

// NewNoopMediaAdapter creates a stub media adapter.
func NewNoopMediaAdapter() mkt.MediaAdapter {
	return &noopMediaAdapter{}
}

func (n *noopMediaAdapter) PresignedUpload(_ context.Context, key, contentType string, _ uint64) (string, error) {
	slog.Warn("media adapter not configured — PresignedUpload is a no-op", "key", key, "content_type", contentType)
	return "https://placeholder.local/upload/" + key, nil
}

func (n *noopMediaAdapter) PresignedGet(_ context.Context, key string, _ uint32) (string, error) {
	slog.Warn("media adapter not configured — PresignedGet is a no-op", "key", key)
	return "https://placeholder.local/get/" + key, nil
}
