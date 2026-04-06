package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/homegrown-academy/homegrown-academy/internal/search"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// HttpTypesenseAdapter implements search.TypesenseAdapter using the Typesense REST API.
// Uses raw net/http to avoid the typesense-go dependency conflict with genproto. [12-search §7.1]
type HttpTypesenseAdapter struct {
	baseURL string
	apiKey  string
	client  *http.Client
}

// Compile-time interface check.
var _ search.TypesenseAdapter = (*HttpTypesenseAdapter)(nil)

// NewHttpTypesenseAdapter creates a new HTTP-based Typesense adapter.
func NewHttpTypesenseAdapter(baseURL, apiKey string) *HttpTypesenseAdapter {
	return &HttpTypesenseAdapter{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// IndexDocument indexes a single document via POST /collections/{collection}/documents.
func (a *HttpTypesenseAdapter) IndexDocument(ctx context.Context, collection string, document map[string]any) error {
	body, err := json.Marshal(document)
	if err != nil {
		return fmt.Errorf("typesense: marshal document: %w", err)
	}

	endpoint := fmt.Sprintf("%s/collections/%s/documents", a.baseURL, url.PathEscape(collection))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"?action=upsert", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("typesense: create request: %w", err)
	}
	a.setHeaders(req)

	resp, err := shared.RetryableHTTPDo(ctx, a.client, req, nil)
	if err != nil {
		return fmt.Errorf("typesense: index document: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return a.readError(resp)
	}
	return nil
}

// RemoveDocument removes a document via DELETE /collections/{collection}/documents/{id}.
func (a *HttpTypesenseAdapter) RemoveDocument(ctx context.Context, collection string, documentID string) error {
	endpoint := fmt.Sprintf("%s/collections/%s/documents/%s", a.baseURL, url.PathEscape(collection), url.PathEscape(documentID))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("typesense: create request: %w", err)
	}
	a.setHeaders(req)

	resp, err := shared.RetryableHTTPDo(ctx, a.client, req, nil)
	if err != nil {
		return fmt.Errorf("typesense: remove document: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 404 is acceptable — document may have been already removed.
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode >= 400 {
		return a.readError(resp)
	}
	return nil
}

// BulkIndex bulk-imports documents via POST /collections/{collection}/documents/import (JSONL format).
func (a *HttpTypesenseAdapter) BulkIndex(ctx context.Context, collection string, documents []map[string]any) (*search.BulkIndexResult, error) {
	if len(documents) == 0 {
		return &search.BulkIndexResult{}, nil
	}

	// Build JSONL body.
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, doc := range documents {
		if err := enc.Encode(doc); err != nil {
			return nil, fmt.Errorf("typesense: marshal bulk document: %w", err)
		}
	}

	endpoint := fmt.Sprintf("%s/collections/%s/documents/import?action=upsert", a.baseURL, url.PathEscape(collection))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return nil, fmt.Errorf("typesense: create request: %w", err)
	}
	a.setHeaders(req)

	resp, err := shared.RetryableHTTPDo(ctx, a.client, req, nil)
	if err != nil {
		return nil, fmt.Errorf("typesense: bulk index: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, a.readError(resp)
	}

	// Typesense returns one JSON object per line in the response.
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("typesense: read bulk response: %w", err)
	}

	result := &search.BulkIndexResult{}
	for line := range strings.SplitSeq(strings.TrimSpace(string(respBody)), "\n") {
		if line == "" {
			continue
		}
		var entry struct {
			Success bool   `json:"success"`
			Error   string `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("parse response line: %v", err))
			continue
		}
		if entry.Success {
			result.Indexed++
		} else {
			result.Failed++
			if entry.Error != "" {
				result.Errors = append(result.Errors, entry.Error)
			}
		}
	}

	return result, nil
}

// Search executes a search query via GET /collections/{collection}/documents/search.
func (a *HttpTypesenseAdapter) Search(ctx context.Context, collection string, query *search.TypesenseSearchQuery) (*search.TypesenseSearchResponse, error) {
	endpoint := fmt.Sprintf("%s/collections/%s/documents/search", a.baseURL, url.PathEscape(collection))

	params := url.Values{}
	params.Set("q", query.Q)
	if len(query.QueryBy) > 0 {
		params.Set("query_by", strings.Join(query.QueryBy, ","))
	}
	if query.FilterBy != nil {
		params.Set("filter_by", *query.FilterBy)
	}
	if query.SortBy != nil {
		params.Set("sort_by", *query.SortBy)
	}
	if len(query.FacetBy) > 0 {
		params.Set("facet_by", strings.Join(query.FacetBy, ","))
	}
	if query.Page > 0 {
		params.Set("page", fmt.Sprintf("%d", query.Page))
	}
	if query.PerPage > 0 {
		params.Set("per_page", fmt.Sprintf("%d", query.PerPage))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("typesense: create request: %w", err)
	}
	a.setHeaders(req)

	resp, err := shared.RetryableHTTPDo(ctx, a.client, req, nil)
	if err != nil {
		return nil, fmt.Errorf("typesense: search: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, a.readError(resp)
	}

	// Parse the Typesense search response.
	var raw struct {
		Found       int64 `json:"found"`
		Hits        []struct {
			Document map[string]any `json:"document"`
		} `json:"hits"`
		FacetCounts []map[string]any `json:"facet_counts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("typesense: decode search response: %w", err)
	}

	hits := make([]map[string]any, len(raw.Hits))
	for i, h := range raw.Hits {
		hits[i] = h.Document
	}

	return &search.TypesenseSearchResponse{
		Found:       raw.Found,
		Hits:        hits,
		FacetCounts: raw.FacetCounts,
	}, nil
}

// CollectionStats returns document count and shard info via GET /collections/{collection}.
func (a *HttpTypesenseAdapter) CollectionStats(ctx context.Context, collection string) (*search.CollectionStats, error) {
	endpoint := fmt.Sprintf("%s/collections/%s", a.baseURL, url.PathEscape(collection))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("typesense: create request: %w", err)
	}
	a.setHeaders(req)

	resp, err := shared.RetryableHTTPDo(ctx, a.client, req, nil)
	if err != nil {
		return nil, fmt.Errorf("typesense: collection stats: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, a.readError(resp)
	}

	var raw struct {
		NumDocuments    int64 `json:"num_documents"`
		NumMemoryShards int32 `json:"num_memory_shards"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("typesense: decode collection stats: %w", err)
	}

	return &search.CollectionStats{
		NumDocuments:    raw.NumDocuments,
		NumMemoryShards: raw.NumMemoryShards,
	}, nil
}

// setHeaders sets common headers for Typesense requests.
func (a *HttpTypesenseAdapter) setHeaders(req *http.Request) {
	req.Header.Set("X-TYPESENSE-API-KEY", a.apiKey)
	req.Header.Set("Content-Type", "application/json")
}

// readError reads an error response body and returns a formatted error.
func (a *HttpTypesenseAdapter) readError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	slog.Error("typesense: API error",
		"status", resp.StatusCode,
		"body", string(body),
	)
	return fmt.Errorf("typesense: HTTP %d: %s", resp.StatusCode, string(body))
}
