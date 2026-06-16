// Package sources provides content ingesters that fetch public-domain educational
// material from external APIs and insert them as mkt_listings.
package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/cmd/seed-content/subjectmap"
	"gorm.io/gorm"
)

const (
	gutendexDefaultURL  = "https://gutendex.com/books"
	gutenbergBookURLFmt = "https://www.gutenberg.org/ebooks/%d"

	defaultTarget   = 200
	insertBatchSize = 100
	maxRetries      = 3
)

// listingNamespace is a fixed UUID v5 namespace that produces deterministic
// listing IDs from Gutenberg book URLs, enabling idempotent re-runs.
var listingNamespace = uuid.MustParse("018f1234-0000-7000-8000-000000000099")

// searchTopics are Gutendex topic queries iterated in order until Target is met.
var searchTopics = []string{
	"History", "Natural history", "Science", "Mathematics",
	"Fiction", "Juvenile Fiction", "Poetry", "Philosophy",
	"Biography", "Astronomy", "Geography", "Education",
	"Economics", "Technology", "Religion",
}

// Config controls ingester behaviour.
type Config struct {
	DB             *gorm.DB
	DryRun         bool
	Target         int           // number of listings to insert; defaults to 200
	BaseURL        string        // Gutendex endpoint; defaults to gutendexDefaultURL
	HTTPClient     *http.Client  // defaults to a 30-second client
	PageDelay      time.Duration // inter-page courtesy delay; 0 = none (useful for tests)
	RetryBaseDelay time.Duration // base wait between HTTP retries; 0 → 1 s; negative → no delay (tests)
	Storage          Uploader          // R2 uploader for hosted ingesters (e.g. CK Foundation); nil = skip uploads
	ThumbnailStorage ThumbnailUploader // raw object-storage for thumbnail upload; nil = skip thumbnail fetcher
}

type gutendexAuthor struct {
	Name string `json:"name"`
}

type gutendexBook struct {
	ID       int               `json:"id"`
	Title    string            `json:"title"`
	Authors  []gutendexAuthor  `json:"authors"`
	Subjects []string          `json:"subjects"`
	Formats  map[string]string `json:"formats"`
}

type gutendexPage struct {
	Count   int            `json:"count"`
	Next    *string        `json:"next"`
	Results []gutendexBook `json:"results"`
}

// RunGutenberg fetches books from the Gutendex API and inserts them as
// published free mkt_listings.  Returns the count inserted (or that would be
// inserted in dry-run mode).
func RunGutenberg(ctx context.Context, cfg Config) (int, error) {
	if cfg.Target <= 0 {
		cfg.Target = defaultTarget
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = gutendexDefaultURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	}

	seen := make(map[int]struct{}) // book IDs seen this run
	total := 0

	for _, topic := range searchTopics {
		if ctx.Err() != nil {
			return total, ctx.Err()
		}
		if total >= cfg.Target {
			break
		}

		pageURL := cfg.BaseURL + "?topic=" + url.QueryEscape(topic) + "&languages=en"

		for pageURL != "" {
			if ctx.Err() != nil {
				return total, ctx.Err()
			}
			if total >= cfg.Target {
				break
			}

			page, err := fetchPage(ctx, cfg.HTTPClient, pageURL, cfg.RetryBaseDelay)
			if err != nil {
				slog.Warn("gutenberg: fetch failed, skipping topic",
					"topic", topic, "err", err)
				break
			}

			rows := buildRows(page.Results, seen, cfg.Target-total)

			if cfg.DryRun {
				for _, row := range rows {
					fmt.Printf("dry-run: id=%s  title=%q  url=%s\n",
						row[0], row[3], row[13])
					total++
				}
			} else if len(rows) > 0 {
				n, err := insertListings(cfg.DB, rows)
				if err != nil {
					return total, fmt.Errorf("insert (topic %s): %w", topic, err)
				}
				total += n
				slog.Info("gutenberg: batch inserted",
					"topic", topic, "rows", len(rows), "affected", n, "total", total)
			}

			if page.Next == nil || *page.Next == "" {
				break
			}
			pageURL = *page.Next

			if cfg.PageDelay > 0 {
				select {
				case <-time.After(cfg.PageDelay):
				case <-ctx.Done():
					return total, ctx.Err()
				}
			}
		}
	}

	return total, nil
}

// buildRows converts Gutendex results to mkt_listings insert row slices.
// It updates seen in place and stops once limit rows have been added.
func buildRows(books []gutendexBook, seen map[int]struct{}, limit int) [][]any {
	var rows [][]any
	now := time.Now().UTC()

	for _, book := range books {
		if len(rows) >= limit {
			break
		}
		if _, dup := seen[book.ID]; dup {
			continue
		}
		if book.Title == "" {
			continue
		}
		seen[book.ID] = struct{}{}

		previewURL := fmt.Sprintf(gutenbergBookURLFmt, book.ID)
		subjects := subjectmap.MapSubjects(book.Subjects)
		subjectTagStr := "{}"
		if len(subjects) > 0 {
			subjectTagStr = "{" + strings.Join(subjects, ",") + "}"
		}
		listingID := uuid.NewSHA1(listingNamespace, []byte(previewURL))

		rows = append(rows, []any{
			listingID.String(),              // id
			PlatformCreatorID,              // creator_id
			PlatformPublisherID,            // publisher_id
			buildTitle(book),               // title
			buildDescription(book, previewURL), // description
			0,                              // price_cents
			"{}",                           // methodology_tags (empty UUID[])
			subjectTagStr,                  // subject_tags
			ClassifyContentType(book.Subjects), // content_type
			"published",                    // status
			0,                              // rating_avg
			0,                              // rating_count
			1,                              // version
			previewURL,                     // preview_url
			now,                            // published_at
			now,                            // created_at
			now,                            // updated_at
		})
	}

	return rows
}

// ClassifyContentType returns "book_list" for fiction/poetry/drama subjects,
// "reading_guide" for everything else.
func ClassifyContentType(subjects []string) string {
	for _, s := range subjects {
		sl := strings.ToLower(s)
		if strings.Contains(sl, "fiction") ||
			strings.Contains(sl, "poetry") ||
			strings.Contains(sl, "drama") {
			return "book_list"
		}
	}
	return "reading_guide"
}

func buildTitle(book gutendexBook) string {
	if len(book.Authors) == 0 {
		return book.Title
	}
	return book.Title + " — " + book.Authors[0].Name
}

func buildDescription(book gutendexBook, previewURL string) string {
	var b strings.Builder
	if len(book.Authors) > 0 {
		names := make([]string, len(book.Authors))
		for i, a := range book.Authors {
			names[i] = a.Name
		}
		b.WriteString("Author(s): ")
		b.WriteString(strings.Join(names, ", "))
		b.WriteString("\n")
	}
	if len(book.Subjects) > 0 {
		shown := book.Subjects[:min(3, len(book.Subjects))]
		b.WriteString("Subjects: ")
		b.WriteString(strings.Join(shown, ", "))
		b.WriteString("\n")
	}
	b.WriteString("Read free at: ")
	b.WriteString(previewURL)
	b.WriteString("\n")
	b.WriteString("Source: Project Gutenberg — Public Domain")
	return b.String()
}

// fetchPage performs one HTTP GET with up to maxRetries retries on 429/5xx.
// retryBase controls the backoff unit: 0 → 1 s, negative → no delay (tests).
func fetchPage(ctx context.Context, client *http.Client, pageURL string, retryBase time.Duration) (*gutendexPage, error) {
	base := retryBase
	if base == 0 {
		base = time.Second
	}
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 && base > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * base
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, pageURL)
		}

		var page gutendexPage
		decodeErr := json.NewDecoder(resp.Body).Decode(&page)
		_ = resp.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("decode response: %w", decodeErr)
		}
		return &page, nil
	}
	return nil, fmt.Errorf("after %d attempts: %w", maxRetries+1, lastErr)
}

// insertListings bulk-inserts listing rows into mkt_listings inside a single
// RLS-bypassed transaction.  Returns the number of rows actually written
// (skipped by ON CONFLICT DO NOTHING are excluded).
func insertListings(db *gorm.DB, rows [][]any) (int, error) {
	const (
		prefix = "INSERT INTO mkt_listings " +
			"(id, creator_id, publisher_id, title, description, price_cents, " +
			"methodology_tags, subject_tags, content_type, status, " +
			"rating_avg, rating_count, version, preview_url, " +
			"published_at, created_at, updated_at) VALUES "
		tmpl   = "(?, ?, ?, ?, ?, ?, ?::UUID[], ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"
		suffix = "ON CONFLICT DO NOTHING"
	)

	var total int64
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL row_security = off").Error; err != nil {
			return err
		}

		for i := 0; i < len(rows); i += insertBatchSize {
			batch := rows[i:min(i+insertBatchSize, len(rows))]

			vals := make([]string, len(batch))
			args := make([]any, 0, len(batch)*17)
			for j, row := range batch {
				vals[j] = tmpl
				args = append(args, row...)
			}

			result := tx.Exec(prefix+strings.Join(vals, ", ")+" "+suffix, args...)
			if result.Error != nil {
				return fmt.Errorf("batch at offset %d: %w", i, result.Error)
			}
			total += result.RowsAffected
		}
		return nil
	})
	return int(total), err
}
