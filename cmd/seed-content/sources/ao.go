package sources

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// aoAttribution is appended to every listing description per the AO public-domain
// curriculum terms. All AO content is freely available at amblesideonline.org.
const aoAttribution = "Source: Ambleside Online (amblesideonline.org) — Public Domain curriculum"

// AOManifest represents one year of the Ambleside Online curriculum,
// as parsed from a data/ao/year-N.yaml file.
type AOManifest struct {
	Year     int       `yaml:"year"`
	GradeMin int16     `yaml:"grade_min"`
	GradeMax int16     `yaml:"grade_max"`
	Entries  []AOEntry `yaml:"entries"`
}

// AOEntry describes a single listing (book or lesson plan) in the AO curriculum.
type AOEntry struct {
	Title         string   `yaml:"title"`
	Description   string   `yaml:"description"`
	ContentType   string   `yaml:"content_type"`
	SubjectTags   []string `yaml:"subject_tags"`
	PreviewURL    string   `yaml:"preview_url"`
	WorldviewTags []string `yaml:"worldview_tags"`
}

// BuildDescription returns the full listing description with AO attribution appended.
func (e AOEntry) BuildDescription() string {
	if e.Description == "" {
		return aoAttribution
	}
	return e.Description + "\n\n" + aoAttribution
}

// ParseAOManifests reads all *.yaml files from fsys and returns the parsed manifests.
// Exported so tests can call it directly without a database.
func ParseAOManifests(fsys fs.FS) ([]AOManifest, error) {
	names, err := fs.Glob(fsys, "*.yaml")
	if err != nil {
		return nil, fmt.Errorf("ao: glob manifests: %w", err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("ao: no *.yaml files found in manifest FS")
	}

	manifests := make([]AOManifest, 0, len(names))
	for _, name := range names {
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("ao: read %s: %w", name, err)
		}
		var m AOManifest
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, fmt.Errorf("ao: parse %s: %w", name, err)
		}
		manifests = append(manifests, m)
	}
	return manifests, nil
}

// IngestAO reads all AO YAML manifests from manifestFS, deduplicates against
// the platform creator's existing listings, and inserts new published mkt_listings.
//
// Deduplication is by preview_url: entries whose URL is already stored for
// PlatformCreatorID are silently skipped (idempotent re-runs are safe).
//
// Returns the count of newly inserted listings.
func IngestAO(ctx context.Context, db *sql.DB, manifestFS fs.FS) (int, error) {
	manifests, err := ParseAOManifests(manifestFS)
	if err != nil {
		return 0, err
	}

	existing, err := loadExistingPreviewURLs(ctx, db)
	if err != nil {
		return 0, fmt.Errorf("ao: load existing urls: %w", err)
	}

	inserted := 0
	for _, m := range manifests {
		for _, e := range m.Entries {
			if existing[e.PreviewURL] {
				slog.Debug("ao: skip duplicate", "title", e.Title, "url", e.PreviewURL)
				continue
			}
			if err := insertAOListing(ctx, db, m, e); err != nil {
				slog.Error("ao: insert failed", "title", e.Title, "err", err)
				continue
			}
			existing[e.PreviewURL] = true
			inserted++
		}
	}

	slog.Info("ao: ingest complete", "inserted", inserted)
	return inserted, nil
}

// insertAOListing writes a single mkt_listing row for an AO book or lesson plan.
//
// methodology_tags is intentionally empty: the methodology domain uses TEXT slugs
// as primary keys, not UUIDs, so there is no UUID mapping to store here.
// worldview_tags is taken directly from the YAML entry.
func insertAOListing(ctx context.Context, db *sql.DB, m AOManifest, e AOEntry) error {
	subjectArr := toPostgresTextArray(e.SubjectTags)
	worldviewArr := toPostgresTextArray(e.WorldviewTags)
	id := uuid.New().String()
	desc := e.BuildDescription()

	_, err := db.ExecContext(ctx, `
		INSERT INTO mkt_listings
			(id, creator_id, publisher_id, title, description, price_cents,
			 methodology_tags, subject_tags, content_type, worldview_tags,
			 grade_min, grade_max, preview_url, status, published_at)
		VALUES ($1, $2, $3, $4, $5, 0,
			'{}', $6::text[], $7, $8::text[],
			$9, $10, $11, 'published', NOW())`,
		id,
		PlatformCreatorID,
		PlatformPublisherID,
		e.Title,
		desc,
		subjectArr,
		e.ContentType,
		worldviewArr,
		m.GradeMin,
		m.GradeMax,
		e.PreviewURL,
	)
	return err
}
