package sources

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

//go:embed ck_manifest.yaml
var ckManifestYAML []byte

// ckNamespace is a fixed UUID v5 namespace for deriving deterministic listing and
// file IDs from CK Foundation storage keys, enabling idempotent re-runs.
var ckNamespace = uuid.MustParse("019a5f00-0000-7000-8000-000000000001")

// Uploader uploads a PDF from a remote URL to object storage and returns the
// presigned download URL and file size in bytes.
// Implemented by *R2Uploader in cmd/seed-content; nil disables hosted uploads. [consumer-defined interface, CODING §2.5]
type Uploader interface {
	Upload(ctx context.Context, srcURL, storageKey string) (signedURL string, sizeBytes int64, err error)
}

// ckEntry is one listing in the embedded CK Foundation manifest.
type ckEntry struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	URL         string   `yaml:"url"`
	ContentType string   `yaml:"content_type"`
	GradeMin    int      `yaml:"grade_min"`
	GradeMax    int      `yaml:"grade_max"`
	Subjects    []string `yaml:"subjects"`
}

type ckManifest struct {
	Listings []ckEntry `yaml:"listings"`
}

func loadCKManifest() (*ckManifest, error) {
	var m ckManifest
	if err := yaml.Unmarshal(ckManifestYAML, &m); err != nil {
		return nil, fmt.Errorf("parse ck_manifest.yaml: %w", err)
	}
	return &m, nil
}

// RunCKFoundation downloads, validates, and uploads each PDF from the embedded
// CK Foundation manifest to R2, then inserts a mkt_listings row and a
// mkt_listing_files row for each successfully uploaded document.
//
// Deduplication is by storage key: entries whose key already exists in
// mkt_listing_files are skipped so the command is idempotent.
//
// If cfg.Storage is nil the ingester warns and returns 0 — R2 credentials are
// required for this hosted ingester.
// If cfg.DryRun is true no network or DB operations are performed.
func RunCKFoundation(ctx context.Context, cfg Config) (int, error) {
	manifest, err := loadCKManifest()
	if err != nil {
		return 0, fmt.Errorf("ck_foundation: %w", err)
	}

	if cfg.DryRun {
		target := cfg.Target
		if target <= 0 || target > len(manifest.Listings) {
			target = len(manifest.Listings)
		}
		for i, entry := range manifest.Listings {
			if i >= target {
				break
			}
			fmt.Printf("dry-run: id=%s  title=%q  url=%s\n",
				entry.ID, entry.Title, entry.URL)
		}
		return target, nil
	}

	if cfg.Storage == nil {
		slog.Warn("ck_foundation: no R2 storage configured — skipping hosted ingester")
		return 0, nil
	}

	existing, err := loadExistingCKStorageKeys(ctx, cfg.DB)
	if err != nil {
		return 0, fmt.Errorf("ck_foundation: load existing keys: %w", err)
	}

	inserted := 0
	for _, entry := range manifest.Listings {
		if ctx.Err() != nil {
			return inserted, ctx.Err()
		}
		if cfg.Target > 0 && inserted >= cfg.Target {
			break
		}

		storageKey := "ck-foundation/" + entry.ID + ".pdf"

		if existing[storageKey] {
			slog.Debug("ck_foundation: skip duplicate", "id", entry.ID)
			continue
		}

		signedURL, sizeBytes, uploadErr := cfg.Storage.Upload(ctx, entry.URL, storageKey)
		if uploadErr != nil {
			slog.Warn("ck_foundation: upload failed, skipping entry",
				"id", entry.ID, "url", entry.URL, "err", uploadErr)
			continue
		}

		if insertErr := insertCKListing(ctx, cfg.DB, entry, signedURL, storageKey, sizeBytes); insertErr != nil {
			slog.Error("ck_foundation: db insert failed, skipping entry",
				"id", entry.ID, "err", insertErr)
			continue
		}

		existing[storageKey] = true
		inserted++
		slog.Info("ck_foundation: inserted listing",
			"id", entry.ID, "title", entry.Title)
	}

	slog.Info("ck_foundation: ingest complete", "inserted", inserted)
	return inserted, nil
}

// loadExistingCKStorageKeys returns a set of ck-foundation/* storage keys
// already present in mkt_listing_files, used to skip duplicates on re-runs.
// Returns an empty map when db is nil (test-only; upload failures prevent DB use).
func loadExistingCKStorageKeys(ctx context.Context, db *gorm.DB) (map[string]bool, error) {
	if db == nil {
		return map[string]bool{}, nil
	}
	var keys []string
	if err := db.WithContext(ctx).
		Table("mkt_listing_files").
		Select("storage_key").
		Where("storage_key LIKE ?", "ck-foundation/%").
		Scan(&keys).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(keys))
	for _, k := range keys {
		seen[k] = true
	}
	return seen, nil
}

// insertCKListing writes one mkt_listings row and one mkt_listing_files row
// inside a single RLS-bypassed transaction.
// Listing ID and file ID are deterministic (UUID v5 from storageKey) so the
// ON CONFLICT DO NOTHING guards are a safety net, not the primary dedup path.
func insertCKListing(ctx context.Context, db *gorm.DB, entry ckEntry, signedURL, storageKey string, sizeBytes int64) error {
	now := time.Now().UTC()
	listingID := uuid.NewSHA1(ckNamespace, []byte(storageKey))
	fileID := uuid.NewSHA1(ckNamespace, []byte("file:"+storageKey))

	gradeMin := int16(entry.GradeMin)
	gradeMax := int16(entry.GradeMax)
	subjectStr := toPostgresTextArray(entry.Subjects)

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Bypass RLS — seed operations have no family context. [CODING §2.4]
		if err := tx.Exec("SET LOCAL row_security = off").Error; err != nil {
			return err
		}

		// methodology_tags intentionally empty: method_definitions uses TEXT slugs as PKs,
		// not UUIDs, so there is no stable UUID↔slug mapping. Consistent with all other
		// content ingesters. [see openstax.go for rationale]
		if err := tx.Exec(`
			INSERT INTO mkt_listings
				(id, creator_id, publisher_id, title, description, price_cents,
				 methodology_tags, subject_tags, content_type,
				 grade_min, grade_max, preview_url,
				 status, published_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, 0,
				'{}', ?::text[], ?,
				?, ?, ?,
				'published', ?, ?, ?)
			ON CONFLICT (id) DO NOTHING`,
			listingID.String(),
			PlatformCreatorID,
			PlatformPublisherID,
			entry.Title,
			entry.Description,
			subjectStr,
			entry.ContentType,
			gradeMin,
			gradeMax,
			signedURL,
			now, now, now,
		).Error; err != nil {
			return fmt.Errorf("insert listing: %w", err)
		}

		if err := tx.Exec(`
			INSERT INTO mkt_listing_files
				(id, listing_id, file_name, file_size_bytes, mime_type, storage_key, sort_order, version)
			VALUES (?, ?, ?, ?, 'application/pdf', ?, 0, 1)
			ON CONFLICT (id) DO NOTHING`,
			fileID.String(),
			listingID.String(),
			entry.ID+".pdf",
			sizeBytes,
			storageKey,
		).Error; err != nil {
			return fmt.Errorf("insert listing file: %w", err)
		}

		return nil
	})
}
