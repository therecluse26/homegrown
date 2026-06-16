package sources_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/homegrown-academy/homegrown-academy/cmd/seed-content/sources"
)

func TestClassifyContentType(t *testing.T) {
	cases := []struct {
		subjects []string
		want     string
	}{
		{[]string{"Fiction"}, "book_list"},
		{[]string{"Juvenile Fiction"}, "book_list"},
		{[]string{"Historical fiction"}, "book_list"},
		{[]string{"American poetry"}, "book_list"},
		{[]string{"English drama"}, "book_list"},
		{[]string{"Science fiction"}, "book_list"},
		{[]string{"History"}, "reading_guide"},
		{[]string{"Mathematics"}, "reading_guide"},
		{[]string{"Natural history"}, "reading_guide"},
		{[]string{"Biography"}, "reading_guide"},
		{[]string{}, "reading_guide"},
	}

	for _, tc := range cases {
		got := sources.ClassifyContentType(tc.subjects)
		if got != tc.want {
			t.Errorf("ClassifyContentType(%v) = %q, want %q", tc.subjects, got, tc.want)
		}
	}
}

// newMockServer returns a test HTTP server that always responds with the given JSON body.
func newMockServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, body)
	}))
}

func TestRunGutenberg_DryRun_ReturnsCount(t *testing.T) {
	srv := newMockServer(`{
		"count": 2,
		"next": null,
		"results": [
			{
				"id": 1,
				"title": "Adventures of Huckleberry Finn",
				"authors": [{"name": "Twain, Mark"}],
				"subjects": ["Fiction", "Humorous stories"],
				"formats": {}
			},
			{
				"id": 2,
				"title": "The Origin of Species",
				"authors": [{"name": "Darwin, Charles"}],
				"subjects": ["Natural history", "Evolution (Biology)"],
				"formats": {}
			}
		]
	}`)
	defer srv.Close()

	cfg := sources.Config{
		DryRun:     true,
		Target:     200,
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		// PageDelay: 0 — no inter-page sleep in tests
	}

	n, err := sources.RunGutenberg(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunGutenberg error: %v", err)
	}
	// Both books appear in every topic's response but the seen map deduplicates
	// them after the first topic — so exactly 2 unique listings regardless of
	// how many topics are searched.
	if n != 2 {
		t.Errorf("expected 2 listings, got %d", n)
	}
}

func TestRunGutenberg_SeenDeduplication(t *testing.T) {
	// Book ID 99 appears in every topic response — should only count once.
	srv := newMockServer(`{
		"count": 1,
		"next": null,
		"results": [
			{
				"id": 99,
				"title": "Duplicate Book",
				"authors": [],
				"subjects": ["History"],
				"formats": {}
			}
		]
	}`)
	defer srv.Close()

	cfg := sources.Config{
		DryRun:     true,
		Target:     200,
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}

	n, err := sources.RunGutenberg(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunGutenberg error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 (deduplicated across topics), got %d", n)
	}
}

func TestRunGutenberg_TargetLimitRespected(t *testing.T) {
	// Mock always returns 5 unique books but target is 3.
	srv := newMockServer(`{
		"count": 5,
		"next": null,
		"results": [
			{"id": 10, "title": "Book A", "authors": [], "subjects": ["History"], "formats": {}},
			{"id": 11, "title": "Book B", "authors": [], "subjects": ["Science"], "formats": {}},
			{"id": 12, "title": "Book C", "authors": [], "subjects": ["Mathematics"], "formats": {}},
			{"id": 13, "title": "Book D", "authors": [], "subjects": ["Fiction"], "formats": {}},
			{"id": 14, "title": "Book E", "authors": [], "subjects": ["Poetry"], "formats": {}}
		]
	}`)
	defer srv.Close()

	cfg := sources.Config{
		DryRun:     true,
		Target:     3,
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}

	n, err := sources.RunGutenberg(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunGutenberg error: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 (target limit), got %d", n)
	}
}

func TestRunGutenberg_Pagination(t *testing.T) {
	var srvURL string
	callCount := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page := r.URL.Query().Get("page")
		callCount++

		if page == "" || page == "1" {
			next := srvURL + "?page=2"
			_, _ = fmt.Fprintf(w, `{
				"count": 4,
				"next": %q,
				"results": [
					{"id": 100, "title": "Page One Book A", "authors": [{"name": "Author A"}], "subjects": ["History"], "formats": {}},
					{"id": 101, "title": "Page One Book B", "authors": [{"name": "Author B"}], "subjects": ["Fiction"], "formats": {}}
				]
			}`, next)
		} else {
			_, _ = fmt.Fprint(w, `{
				"count": 4,
				"next": null,
				"results": [
					{"id": 102, "title": "Page Two Book C", "authors": [], "subjects": ["Science"], "formats": {}},
					{"id": 103, "title": "Page Two Book D", "authors": [], "subjects": ["Poetry"], "formats": {}}
				]
			}`)
		}
	}))
	srvURL = srv.URL
	defer srv.Close()

	cfg := sources.Config{
		DryRun:     true,
		Target:     4,
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}

	n, err := sources.RunGutenberg(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunGutenberg error: %v", err)
	}
	if n != 4 {
		t.Errorf("expected 4 listings from 2 pages, got %d", n)
	}
}

func TestRunGutenberg_SkipsEmptyTitle(t *testing.T) {
	srv := newMockServer(`{
		"count": 2,
		"next": null,
		"results": [
			{"id": 200, "title": "", "authors": [], "subjects": ["History"], "formats": {}},
			{"id": 201, "title": "Valid Title", "authors": [], "subjects": ["History"], "formats": {}}
		]
	}`)
	defer srv.Close()

	cfg := sources.Config{
		DryRun:     true,
		Target:     10,
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
	}

	n, err := sources.RunGutenberg(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunGutenberg error: %v", err)
	}
	// Book 200 has empty title and must be skipped.
	if n != 1 {
		t.Errorf("expected 1 (empty title skipped), got %d", n)
	}
}

func TestRunGutenberg_HTTPError_SkipsTopic(t *testing.T) {
	// Server returns 500 for every request — ingester should skip all topics
	// gracefully and return 0 with no error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := sources.Config{
		DryRun:         true,
		Target:         200,
		BaseURL:        srv.URL,
		HTTPClient:     srv.Client(),
		RetryBaseDelay: -1, // disable inter-retry sleep so the test finishes quickly
	}

	n, err := sources.RunGutenberg(context.Background(), cfg)
	if err != nil {
		t.Fatalf("RunGutenberg should not return error on fetch failures, got: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 listings on repeated server errors, got %d", n)
	}
}
