package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/homegrown-academy/homegrown-academy/internal/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpTypesenseAdapter_IndexDocument(t *testing.T) {
	var receivedBody map[string]any
	var receivedAPIKey string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("X-TYPESENSE-API-KEY")
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/collections/products/documents")
		assert.Equal(t, "upsert", r.URL.Query().Get("action"))

		if err := json.NewDecoder(r.Body).Decode(&receivedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"123"}`))
	}))
	defer srv.Close()

	adapter := NewHttpTypesenseAdapter(srv.URL, "test-key")
	err := adapter.IndexDocument(context.Background(), "products", map[string]any{
		"id":    "123",
		"title": "Math Workbook",
	})

	require.NoError(t, err)
	assert.Equal(t, "test-key", receivedAPIKey)
	assert.Equal(t, "123", receivedBody["id"])
	assert.Equal(t, "Math Workbook", receivedBody["title"])
}

func TestHttpTypesenseAdapter_RemoveDocument(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Contains(t, r.URL.Path, "/collections/products/documents/123")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"123"}`))
	}))
	defer srv.Close()

	adapter := NewHttpTypesenseAdapter(srv.URL, "test-key")
	err := adapter.RemoveDocument(context.Background(), "products", "123")
	require.NoError(t, err)
}

func TestHttpTypesenseAdapter_RemoveDocument_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Could not find a document with id: 123"}`))
	}))
	defer srv.Close()

	adapter := NewHttpTypesenseAdapter(srv.URL, "test-key")
	err := adapter.RemoveDocument(context.Background(), "products", "123")
	require.NoError(t, err) // 404 is not an error for remove
}

func TestHttpTypesenseAdapter_BulkIndex(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/collections/products/documents/import")

		w.WriteHeader(http.StatusOK)
		// Typesense bulk import returns one JSON object per line.
		_, _ = w.Write([]byte("{\"success\":true}\n{\"success\":true}\n{\"success\":false,\"error\":\"bad field\"}\n"))
	}))
	defer srv.Close()

	adapter := NewHttpTypesenseAdapter(srv.URL, "test-key")
	result, err := adapter.BulkIndex(context.Background(), "products", []map[string]any{
		{"id": "1", "title": "Item 1"},
		{"id": "2", "title": "Item 2"},
		{"id": "3", "title": "Item 3"},
	})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Indexed)
	assert.Equal(t, 1, result.Failed)
	assert.Contains(t, result.Errors, "bad field")
}

func TestHttpTypesenseAdapter_BulkIndex_Empty(t *testing.T) {
	adapter := NewHttpTypesenseAdapter("http://unused", "test-key")
	result, err := adapter.BulkIndex(context.Background(), "products", nil)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Indexed)
	assert.Equal(t, 0, result.Failed)
}

func TestHttpTypesenseAdapter_Search(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/collections/products/documents/search")
		assert.Equal(t, "math", r.URL.Query().Get("q"))
		assert.Equal(t, "title,description", r.URL.Query().Get("query_by"))

		resp := map[string]any{
			"found": 2,
			"hits": []map[string]any{
				{"document": map[string]any{"id": "1", "title": "Math 101"}},
				{"document": map[string]any{"id": "2", "title": "Math 201"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	adapter := NewHttpTypesenseAdapter(srv.URL, "test-key")
	filterBy := "price_cents:>0"
	result, err := adapter.Search(context.Background(), "products", &search.TypesenseSearchQuery{
		Q:        "math",
		QueryBy:  []string{"title", "description"},
		FilterBy: &filterBy,
		Page:     1,
		PerPage:  10,
	})

	require.NoError(t, err)
	assert.Equal(t, int64(2), result.Found)
	assert.Len(t, result.Hits, 2)
	assert.Equal(t, "Math 101", result.Hits[0]["title"])
}

func TestHttpTypesenseAdapter_CollectionStats(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.True(t, strings.HasSuffix(r.URL.Path, "/collections/products"))

		resp := map[string]any{
			"num_documents":     42,
			"num_memory_shards": 1,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	adapter := NewHttpTypesenseAdapter(srv.URL, "test-key")
	stats, err := adapter.CollectionStats(context.Background(), "products")

	require.NoError(t, err)
	assert.Equal(t, int64(42), stats.NumDocuments)
	assert.Equal(t, int32(1), stats.NumMemoryShards)
}

func TestHttpTypesenseAdapter_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"Bad query"}`))
	}))
	defer srv.Close()

	adapter := NewHttpTypesenseAdapter(srv.URL, "test-key")
	err := adapter.IndexDocument(context.Background(), "products", map[string]any{"id": "1"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 400")
}

func TestNoopTypesenseAdapter_AllMethods(t *testing.T) {
	adapter := &NoopTypesenseAdapter{}
	ctx := context.Background()

	err := adapter.IndexDocument(ctx, "test", map[string]any{"id": "1"})
	assert.NoError(t, err)

	err = adapter.RemoveDocument(ctx, "test", "1")
	assert.NoError(t, err)

	result, err := adapter.BulkIndex(ctx, "test", []map[string]any{{"id": "1"}, {"id": "2"}})
	assert.NoError(t, err)
	assert.Equal(t, 2, result.Indexed)

	searchResult, err := adapter.Search(ctx, "test", &search.TypesenseSearchQuery{Q: "test"})
	assert.NoError(t, err)
	assert.Empty(t, searchResult.Hits)

	stats, err := adapter.CollectionStats(ctx, "test")
	assert.NoError(t, err)
	assert.Equal(t, int64(0), stats.NumDocuments)
}
