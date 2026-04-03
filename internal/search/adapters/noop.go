package adapters

import (
	"context"
	"log/slog"

	"github.com/homegrown-academy/homegrown-academy/internal/search"
)

// NoopTypesenseAdapter is a no-op implementation of search.TypesenseAdapter
// used when Typesense is not configured. All write operations log and succeed;
// Search returns empty results. [12-search §7.1]
type NoopTypesenseAdapter struct{}

// Compile-time interface check.
var _ search.TypesenseAdapter = (*NoopTypesenseAdapter)(nil)

func (a *NoopTypesenseAdapter) IndexDocument(_ context.Context, _ string, _ map[string]any) error {
	return nil
}

func (a *NoopTypesenseAdapter) RemoveDocument(_ context.Context, _ string, _ string) error {
	return nil
}

func (a *NoopTypesenseAdapter) BulkIndex(_ context.Context, _ string, documents []map[string]any) (*search.BulkIndexResult, error) {
	return &search.BulkIndexResult{Indexed: len(documents)}, nil
}

func (a *NoopTypesenseAdapter) Search(_ context.Context, _ string, _ *search.TypesenseSearchQuery) (*search.TypesenseSearchResponse, error) {
	slog.Warn("typesense not configured — search returning empty results")
	return &search.TypesenseSearchResponse{Hits: []map[string]any{}}, nil
}

func (a *NoopTypesenseAdapter) CollectionStats(_ context.Context, _ string) (*search.CollectionStats, error) {
	return &search.CollectionStats{}, nil
}
