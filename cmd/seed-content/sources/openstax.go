// Package sources provides content ingestion from external educational sources.
// Each source fetches a public catalog, maps subjects via the subjectmap package,
// and upserts mkt_listings rows using deterministic platform creator/publisher UUIDs.
package sources

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/cmd/seed-content/subjectmap"
)

const (
	openstaxCatalogURL = "https://openstax.org/api/books?format=json"
	openstaxBookURLFmt = "https://openstax.org/details/books/%s"

	// openstaxAttribution is appended to every listing description per CC BY 4.0 terms.
	openstaxAttribution = "Source: OpenStax (openstax.org) — CC BY 4.0"

	openstaxHTTPTimeout = 15 * time.Second
)

// OpenStaxBook is a book record from the OpenStax catalog API.
// Exported so tests and the main command can construct test fixtures directly.
type OpenStaxBook struct {
	ID          int      `json:"id"`
	Title       string   `json:"title"`
	Subjects    []string `json:"subjects"`
	Slug        string   `json:"slug"`
	BookState   string   `json:"book_state"`
	ComingSoon  bool     `json:"coming_soon"`
	Description string   `json:"description"`
}

// openStaxCatalog is the top-level API response envelope.
type openStaxCatalog struct {
	Books []OpenStaxBook `json:"books"`
}

// IsEligible reports whether the book should be imported as a listing.
// Excludes coming-soon titles, books with no slug, and non-live books.
func (b OpenStaxBook) IsEligible() bool {
	return b.BookState == "live" && !b.ComingSoon && b.Slug != "" && b.Title != ""
}

// ContentType maps OpenStax subject names to mkt_listing content_type values.
// Textbooks with hard-science or math subjects are tagged as 'curriculum' (full course);
// social-science and humanities titles are tagged as 'reading_guide'.
func (b OpenStaxBook) ContentType() string {
	for _, s := range b.Subjects {
		sl := strings.ToLower(s)
		// "Social Sciences" must not match the bare "science" stem.
		isSocialScience := strings.Contains(sl, "social") && strings.Contains(sl, "science")
		switch {
		case strings.Contains(sl, "math"),
			strings.Contains(sl, "physics"),
			strings.Contains(sl, "biology"),
			strings.Contains(sl, "chemistry"),
			strings.Contains(sl, "statistics"),
			strings.Contains(sl, "calculus"),
			strings.Contains(sl, "algebra"),
			strings.Contains(sl, "computer"),
			(!isSocialScience && strings.Contains(sl, "science")):
			return "curriculum"
		}
	}
	return "reading_guide"
}

// BuildDescription constructs the listing description with attribution appended.
func (b OpenStaxBook) BuildDescription() string {
	body := b.Description
	if body == "" {
		body = fmt.Sprintf(
			"%s is a peer-reviewed open-access textbook freely available to students and educators.",
			b.Title,
		)
	}
	return body + "\n\n" + openstaxAttribution
}

// FetchOpenStaxBooks fetches the live OpenStax catalog and returns all book records.
// The caller may pass a custom client (e.g. httptest transport); nil uses a default
// client with a 15-second timeout.
func FetchOpenStaxBooks(ctx context.Context, client *http.Client) ([]OpenStaxBook, error) {
	if client == nil {
		client = &http.Client{Timeout: openstaxHTTPTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, openstaxCatalogURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET catalog: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("catalog returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var catalog openStaxCatalog
	if err := json.Unmarshal(body, &catalog); err != nil {
		return nil, fmt.Errorf("parse catalog JSON: %w", err)
	}
	return catalog.Books, nil
}

// IngestOpenStax fetches the OpenStax catalog, filters eligible books, maps subjects,
// and inserts new mkt_listings rows.  Deduplication is by (creator_id, preview_url):
// books whose URL already exists for the platform creator are skipped.
//
// The client parameter may be nil (uses a default HTTP client).
// Returns the count of newly inserted listings.
func IngestOpenStax(ctx context.Context, db *sql.DB, client *http.Client) (int, error) {
	books, err := FetchOpenStaxBooks(ctx, client)
	if err != nil {
		return 0, fmt.Errorf("openstax: %w", err)
	}
	slog.Info("openstax: catalog fetched", "total", len(books))

	existing, err := loadExistingPreviewURLs(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("openstax: load existing urls: %w", err)
	}

	inserted := 0
	for _, b := range books {
		if !b.IsEligible() {
			continue
		}
		previewURL := fmt.Sprintf(openstaxBookURLFmt, b.Slug)
		if existing[previewURL] {
			slog.Debug("openstax: skip duplicate", "title", b.Title)
			continue
		}
		if err := insertListing(ctx, db, b, previewURL); err != nil {
			slog.Error("openstax: insert failed", "title", b.Title, "err", err)
			continue
		}
		existing[previewURL] = true
		inserted++
	}

	slog.Info("openstax: ingest complete", "inserted", inserted)
	return inserted, nil
}

// loadExistingPreviewURLs returns a set of preview_urls already stored for the
// platform content creator, so we can skip duplicates without a unique-index.
func loadExistingPreviewURLs(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT preview_url FROM mkt_listings
		  WHERE creator_id = $1 AND preview_url IS NOT NULL`,
		PlatformCreatorID,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	seen := make(map[string]bool)
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		seen[u] = true
	}
	return seen, rows.Err()
}

// insertListing writes a single mkt_listing row for the given book.
// methodology_tags is intentionally empty: the methodology domain uses TEXT slugs
// as primary keys, not UUIDs, so there is no UUID mapping to store here.
// worldview_tags defaults to '{secular}' per OpenStax's openly secular content.
func insertListing(ctx context.Context, db *sql.DB, b OpenStaxBook, previewURL string) error {
	slugs := subjectmap.MapSubjects(b.Subjects)
	subjectArr := toPostgresTextArray(slugs)

	desc := b.BuildDescription()
	id := uuid.New().String()

	_, err := db.ExecContext(ctx, `
		INSERT INTO mkt_listings
			(id, creator_id, publisher_id, title, description, price_cents,
			 methodology_tags, subject_tags, content_type, worldview_tags,
			 preview_url, status, published_at)
		VALUES ($1, $2, $3, $4, $5, 0,
			'{}', $6::text[], $7, '{secular}',
			$8, 'published', NOW())`,
		id,
		PlatformCreatorID,
		PlatformPublisherID,
		b.Title,
		desc,
		subjectArr,
		b.ContentType(),
		previewURL,
	)
	return err
}

// toPostgresTextArray converts a Go string slice to a PostgreSQL array literal,
// e.g. []string{"a", "b"} → "{a,b}".
func toPostgresTextArray(ss []string) string {
	if len(ss) == 0 {
		return "{}"
	}
	return "{" + strings.Join(ss, ",") + "}"
}
