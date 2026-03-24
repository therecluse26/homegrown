package adapters

import (
	"context"

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
