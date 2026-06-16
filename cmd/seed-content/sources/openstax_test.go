package sources_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/homegrown-academy/homegrown-academy/cmd/seed-content/sources"
)

// ─── IsEligible ─────────────────────────────────────────────────────────────

func TestIsEligible(t *testing.T) {
	cases := []struct {
		name string
		book sources.OpenStaxBook
		want bool
	}{
		{
			name: "live book with slug",
			book: sources.OpenStaxBook{Title: "Physics", Slug: "physics", BookState: "live"},
			want: true,
		},
		{
			name: "coming soon",
			book: sources.OpenStaxBook{Title: "Physics", Slug: "physics", BookState: "live", ComingSoon: true},
			want: false,
		},
		{
			name: "not live",
			book: sources.OpenStaxBook{Title: "Physics", Slug: "physics", BookState: "retired"},
			want: false,
		},
		{
			name: "empty slug",
			book: sources.OpenStaxBook{Title: "Physics", Slug: "", BookState: "live"},
			want: false,
		},
		{
			name: "empty title",
			book: sources.OpenStaxBook{Title: "", Slug: "physics", BookState: "live"},
			want: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.book.IsEligible(); got != tc.want {
				t.Errorf("IsEligible() = %v, want %v", got, tc.want)
			}
		})
	}
}

// ─── ContentType ─────────────────────────────────────────────────────────────

func TestContentType(t *testing.T) {
	cases := []struct {
		name     string
		subjects []string
		want     string
	}{
		{"math subject", []string{"Math"}, "curriculum"},
		{"science subject", []string{"Science"}, "curriculum"},
		{"physics subject", []string{"College Physics"}, "curriculum"},
		{"biology subject", []string{"Biology"}, "curriculum"},
		{"chemistry subject", []string{"Chemistry"}, "curriculum"},
		{"statistics subject", []string{"Statistics"}, "curriculum"},
		{"computer science subject", []string{"Computer Science"}, "curriculum"},
		{"social sciences", []string{"Social Sciences"}, "reading_guide"},
		{"humanities", []string{"Humanities"}, "reading_guide"},
		{"business", []string{"Business"}, "reading_guide"},
		{"no subjects", []string{}, "reading_guide"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := sources.OpenStaxBook{Subjects: tc.subjects}
			if got := b.ContentType(); got != tc.want {
				t.Errorf("ContentType(%v) = %q, want %q", tc.subjects, got, tc.want)
			}
		})
	}
}

// ─── BuildDescription ────────────────────────────────────────────────────────

func TestOpenStaxBuildDescription_ContainsAttribution(t *testing.T) {
	b := sources.OpenStaxBook{Title: "Calculus Volume 1", Slug: "calculus-volume-1"}
	desc := b.BuildDescription()
	if !strings.Contains(desc, "openstax.org") {
		t.Errorf("description missing openstax.org attribution: %q", desc)
	}
	if !strings.Contains(desc, "CC BY 4.0") {
		t.Errorf("description missing CC BY 4.0: %q", desc)
	}
}

func TestOpenStaxBuildDescription_UsesProvidedDescription(t *testing.T) {
	b := sources.OpenStaxBook{
		Title:       "Calculus Volume 1",
		Description: "An introduction to differential calculus.",
	}
	desc := b.BuildDescription()
	if !strings.Contains(desc, "differential calculus") {
		t.Errorf("description did not preserve provided description: %q", desc)
	}
}

func TestOpenStaxBuildDescription_FallsBackToTitleWhenEmpty(t *testing.T) {
	b := sources.OpenStaxBook{Title: "Organic Chemistry"}
	desc := b.BuildDescription()
	if !strings.Contains(desc, "Organic Chemistry") {
		t.Errorf("fallback description should mention title: %q", desc)
	}
}

// ─── FetchOpenStaxBooks ──────────────────────────────────────────────────────

func TestFetchOpenStaxBooks_ParsesLiveCatalog(t *testing.T) {
	catalog := map[string]any{
		"books": []map[string]any{
			{"id": 1, "title": "University Physics Volume 1", "slug": "university-physics-volume-1", "book_state": "live", "coming_soon": false, "subjects": []string{"Science"}},
			{"id": 2, "title": "Calculus Volume 1", "slug": "calculus-volume-1", "book_state": "live", "coming_soon": false, "subjects": []string{"Math"}},
			{"id": 3, "title": "Coming Soon Biology", "slug": "biology", "book_state": "live", "coming_soon": true, "subjects": []string{"Science"}},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(catalog); err != nil {
			t.Errorf("encode catalog: %v", err)
		}
	}))
	defer srv.Close()

	// Override the URL by replacing the default client transport with one that
	// redirects openstax.org to the test server.
	client := srv.Client()
	client.Transport = &redirectTransport{target: srv.URL, wrapped: srv.Client().Transport}

	books, err := sources.FetchOpenStaxBooks(context.Background(), client)
	if err != nil {
		t.Fatalf("FetchOpenStaxBooks: %v", err)
	}
	if len(books) != 3 {
		t.Errorf("expected 3 books, got %d", len(books))
	}
	if books[0].Title != "University Physics Volume 1" {
		t.Errorf("unexpected first title: %q", books[0].Title)
	}
}

func TestFetchOpenStaxBooks_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := &http.Client{}
	client.Transport = &redirectTransport{target: srv.URL, wrapped: http.DefaultTransport}

	_, err := sources.FetchOpenStaxBooks(context.Background(), client)
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
}

func TestFetchOpenStaxBooks_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()

	client := &http.Client{}
	client.Transport = &redirectTransport{target: srv.URL, wrapped: http.DefaultTransport}

	_, err := sources.FetchOpenStaxBooks(context.Background(), client)
	if err == nil {
		t.Fatal("expected error for malformed JSON, got nil")
	}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// redirectTransport rewrites any request to target so tests don't require network access.
type redirectTransport struct {
	target  string
	wrapped http.RoundTripper
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(rt.target, "http://")
	return rt.wrapped.RoundTrip(req)
}
