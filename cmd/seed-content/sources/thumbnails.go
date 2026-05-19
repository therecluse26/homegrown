package sources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" // register PNG decoder
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"golang.org/x/image/draw"
	"gorm.io/gorm"
)

// ThumbnailUploader uploads raw image bytes to object storage and presigns a URL.
// Consumer-defined interface for the thumbnail fetcher. [CODING §2.5]
type ThumbnailUploader interface {
	PutObject(ctx context.Context, key string, data []byte, contentType string) error
	PresignedGet(ctx context.Context, key string, expiresSeconds uint32) (string, error)
}

const (
	thumbWidth  = 400
	thumbHeight = 300
	// thumbPresignTTL is a 3-year presigned URL TTL — long-lived for seed data, matching uploader.go.
	thumbPresignTTL = uint32(3 * 365 * 24 * 3600)
	maxImageBytes   = 20 * 1024 * 1024 // 20 MB guard
	maxJSONBytes    = 1 * 1024 * 1024  // 1 MB for API JSON responses

	thumbRetries = 3
)

var (
	jpegMagic = []byte{0xFF, 0xD8, 0xFF}
	pngMagic  = []byte{0x89, 0x50, 0x4E, 0x47}
)

// RunThumbnailFetcher queries all platform-seeded listings with a null thumbnail_url,
// fetches a cover image from Wikimedia Commons (primary) or Smithsonian Open Access
// (fallback, requires SMITHSONIAN_API_KEY env var), resizes to 400×300 JPEG, uploads
// to R2, and UPDATE-s thumbnail_url. Idempotent — listings with an existing thumbnail
// are skipped. On dry-run (cfg.DryRun), prints candidate image URLs without uploading;
// requires cfg.DB to query which listings need thumbnails.
func RunThumbnailFetcher(ctx context.Context, cfg Config) (int, error) {
	if cfg.DB == nil {
		if cfg.DryRun {
			slog.Info("thumbnails: dry-run with no DB — open a DB connection to preview candidates; skipping")
			return 0, nil
		}
		return 0, fmt.Errorf("thumbnails: DB required")
	}
	if !cfg.DryRun && cfg.ThumbnailStorage == nil {
		slog.Warn("thumbnails: no R2 storage configured — skipping thumbnail fetcher")
		return 0, nil
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	smithsonianKey := os.Getenv("SMITHSONIAN_API_KEY")

	listings, err := loadListingsWithoutThumbnail(ctx, cfg.DB)
	if err != nil {
		return 0, fmt.Errorf("thumbnails: load listings: %w", err)
	}
	slog.Info("thumbnails: found listings without thumbnail", "count", len(listings))

	updated := 0
	for _, listing := range listings {
		if ctx.Err() != nil {
			return updated, ctx.Err()
		}

		imgURL, err := findThumbnailURL(ctx, client, listing.title, smithsonianKey, cfg.RetryBaseDelay)
		if err != nil || imgURL == "" {
			slog.Warn("thumbnails: no image found",
				"listing_id", listing.id, "title", listing.title)
			continue
		}

		if cfg.DryRun {
			fmt.Printf("dry-run: listing_id=%s  title=%q  candidate_url=%s\n",
				listing.id, listing.title, imgURL)
			updated++
			continue
		}

		thumbURL, err := downloadResizeUpload(ctx, client, cfg.ThumbnailStorage, listing.id, imgURL)
		if err != nil {
			slog.Warn("thumbnails: process failed, skipping",
				"listing_id", listing.id, "title", listing.title, "err", err)
			continue
		}

		if err := updateThumbnailURL(ctx, cfg.DB, listing.id, thumbURL); err != nil {
			slog.Error("thumbnails: db update failed",
				"listing_id", listing.id, "err", err)
			continue
		}

		updated++
		slog.Info("thumbnails: updated", "listing_id", listing.id, "title", listing.title)
	}

	slog.Info("thumbnails: complete", "updated", updated)
	return updated, nil
}

type thumbListing struct {
	id    string
	title string
}

// loadListingsWithoutThumbnail returns all platform-seeded listings with null thumbnail_url.
func loadListingsWithoutThumbnail(ctx context.Context, db *gorm.DB) ([]thumbListing, error) {
	var rows []struct {
		ID    string `gorm:"column:id"`
		Title string `gorm:"column:title"`
	}
	err := db.WithContext(ctx).
		Table("mkt_listings").
		Select("id, title").
		Where("creator_id = ? AND thumbnail_url IS NULL AND status = 'published'", PlatformCreatorID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]thumbListing, len(rows))
	for i, r := range rows {
		out[i] = thumbListing{id: r.ID, title: r.Title}
	}
	return out, nil
}

// findThumbnailURL tries Wikimedia Commons first, then Smithsonian, returning
// the first usable direct image URL (JPEG or PNG).
func findThumbnailURL(ctx context.Context, client *http.Client, title, smithsonianKey string, retryBase time.Duration) (string, error) {
	if u, err := wikimediaImageURL(ctx, client, title, retryBase); err == nil && u != "" {
		return u, nil
	}
	if smithsonianKey != "" {
		if u, err := smithsonianImageURL(ctx, client, title, smithsonianKey, retryBase); err == nil && u != "" {
			return u, nil
		}
	}
	return "", nil
}

// ── Wikimedia Commons ─────────────────────────────────────────────────────────

type wikimediaSearchResp struct {
	Query struct {
		Search []struct {
			Title string `json:"title"`
		} `json:"search"`
	} `json:"query"`
}

type wikimediaInfoResp struct {
	Query struct {
		Pages map[string]struct {
			ImageInfo []struct {
				URL string `json:"url"`
			} `json:"imageinfo"`
		} `json:"pages"`
	} `json:"query"`
}

// wikimediaImageURL searches Wikimedia Commons (File namespace) for images matching
// title, returning the first JPEG or PNG direct URL found.
func wikimediaImageURL(ctx context.Context, client *http.Client, title string, retryBase time.Duration) (string, error) {
	q := url.Values{
		"action":      {"query"},
		"list":        {"search"},
		"srsearch":    {title},
		"srnamespace": {"6"},
		"srlimit":     {"5"},
		"format":      {"json"},
	}
	searchURL := "https://commons.wikimedia.org/w/api.php?" + q.Encode()

	data, err := getJSONBytes(ctx, client, searchURL, retryBase)
	if err != nil {
		return "", fmt.Errorf("wikimedia search: %w", err)
	}

	var searchResp wikimediaSearchResp
	if err := json.Unmarshal(data, &searchResp); err != nil {
		return "", fmt.Errorf("wikimedia search decode: %w", err)
	}

	for _, result := range searchResp.Query.Search {
		fileName := strings.TrimPrefix(result.Title, "File:")
		ext := strings.ToLower(path.Ext(fileName))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			continue // skip SVG, TIFF, OGG, WebP, etc.
		}

		infoQ := url.Values{
			"action": {"query"},
			"titles": {"File:" + fileName},
			"prop":   {"imageinfo"},
			"iiprop": {"url"},
			"format": {"json"},
		}
		infoURL := "https://commons.wikimedia.org/w/api.php?" + infoQ.Encode()

		infoData, err := getJSONBytes(ctx, client, infoURL, retryBase)
		if err != nil {
			slog.Debug("wikimedia info fetch failed", "file", fileName, "err", err)
			continue
		}

		var infoResp wikimediaInfoResp
		if err := json.Unmarshal(infoData, &infoResp); err != nil {
			continue
		}

		for _, page := range infoResp.Query.Pages {
			if len(page.ImageInfo) > 0 && page.ImageInfo[0].URL != "" {
				return page.ImageInfo[0].URL, nil
			}
		}
	}
	return "", nil
}

// ── Smithsonian Open Access ───────────────────────────────────────────────────

type smithsonianResp struct {
	Response struct {
		Rows []struct {
			OnlineMedia struct {
				Media []struct {
					Content string `json:"content"`
					Type    string `json:"type"`
				} `json:"media"`
			} `json:"online_media"`
		} `json:"rows"`
	} `json:"response"`
}

// smithsonianImageURL searches the Smithsonian Open Access API for an image matching
// title, returning the first content URL found.
func smithsonianImageURL(ctx context.Context, client *http.Client, title, apiKey string, retryBase time.Duration) (string, error) {
	q := url.Values{
		"q":                 {title},
		"media":             {"Images"},
		"online_media_type": {"Images"},
		"api_key":           {apiKey},
		"rows":              {"5"},
	}
	searchURL := "https://api.si.edu/openaccess/api/v1.0/search?" + q.Encode()

	data, err := getJSONBytes(ctx, client, searchURL, retryBase)
	if err != nil {
		return "", fmt.Errorf("smithsonian search: %w", err)
	}

	var resp smithsonianResp
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("smithsonian decode: %w", err)
	}

	for _, row := range resp.Response.Rows {
		for _, m := range row.OnlineMedia.Media {
			if m.Type == "Images" && m.Content != "" {
				return m.Content, nil
			}
		}
	}
	return "", nil
}

// ── Image processing ──────────────────────────────────────────────────────────

// downloadResizeUpload downloads imgURL, validates image magic bytes, decodes,
// resizes to thumbWidth×thumbHeight, encodes as JPEG, uploads to R2, and returns
// the presigned URL.
func downloadResizeUpload(
	ctx context.Context,
	client *http.Client,
	storage ThumbnailUploader,
	listingID, imgURL string,
) (string, error) {
	data, err := downloadImageBytes(ctx, client, imgURL)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}

	// Validate magic bytes before feeding to image decoder. [CODING §5.2]
	if !isValidImageMagic(data) {
		return "", fmt.Errorf("invalid image magic bytes for %s", imgURL)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	resized := resizeBilinear(img, thumbWidth, thumbHeight)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85}); err != nil {
		return "", fmt.Errorf("jpeg encode: %w", err)
	}

	storageKey := "thumbnails/" + listingID + ".jpg"
	if err := storage.PutObject(ctx, storageKey, buf.Bytes(), "image/jpeg"); err != nil {
		return "", fmt.Errorf("R2 upload: %w", err)
	}

	signedURL, err := storage.PresignedGet(ctx, storageKey, thumbPresignTTL)
	if err != nil {
		return "", fmt.Errorf("presign: %w", err)
	}

	return signedURL, nil
}

// downloadImageBytes fetches imgURL and returns the raw bytes, capped at maxImageBytes.
func downloadImageBytes(ctx context.Context, client *http.Client, imgURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, imgURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "HomegrownAcademy/1.0 (content seeder; educational use)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, imgURL)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxImageBytes))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	return data, nil
}

// isValidImageMagic reports whether data begins with JPEG (FF D8 FF) or PNG (89 50 4E 47) magic bytes.
func isValidImageMagic(data []byte) bool {
	if len(data) >= len(jpegMagic) && bytes.Equal(data[:len(jpegMagic)], jpegMagic) {
		return true
	}
	if len(data) >= len(pngMagic) && bytes.Equal(data[:len(pngMagic)], pngMagic) {
		return true
	}
	return false
}

// resizeBilinear returns a new RGBA image scaled to w×h using bilinear interpolation.
func resizeBilinear(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// updateThumbnailURL sets thumbnail_url on a single listing, bypassing RLS.
func updateThumbnailURL(ctx context.Context, db *gorm.DB, listingID, thumbURL string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL row_security = off").Error; err != nil {
			return err
		}
		return tx.Exec(
			"UPDATE mkt_listings SET thumbnail_url = ? WHERE id = ?",
			thumbURL, listingID,
		).Error
	})
}

// getJSONBytes performs a GET with exponential backoff on 429/5xx and returns the
// response body (capped at maxJSONBytes). Follows the same retry pattern as fetchPage.
func getJSONBytes(ctx context.Context, client *http.Client, reqURL string, retryBase time.Duration) ([]byte, error) {
	base := retryBase
	if base == 0 {
		base = time.Second
	}

	var lastErr error
	for attempt := 0; attempt <= thumbRetries; attempt++ {
		if attempt > 0 && base > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * base
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "HomegrownAcademy/1.0 (content seeder; educational use)")

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
			return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, reqURL)
		}

		data, readErr := io.ReadAll(io.LimitReader(resp.Body, maxJSONBytes))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read response: %w", readErr)
		}
		return data, nil
	}
	return nil, fmt.Errorf("after %d attempts: %w", thumbRetries+1, lastErr)
}
