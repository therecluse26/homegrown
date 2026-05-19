package sources

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Platform entity UUIDs use the 01900000-... seed namespace so they are
// consistent with the rest of the seeder and always reference valid rows.
const (
	// PlatformPublisherID is the publisher row that represents this platform
	// hosting public-domain content (Project Gutenberg, CK Foundation, etc.).
	PlatformPublisherID = "01900000-0000-7000-8000-000000000205"

	// PlatformCreatorID is the creator row that owns platform-seeded listings.
	// It is linked to the platform admin parent (adminParentID in cmd/seed).
	PlatformCreatorID = "01900000-0000-7000-8000-000000000206"

	// platformParentID is the iam_parents row for the platform family,
	// which is always present after make seed runs.
	platformParentID = "01900000-0000-7000-8000-000000000013"
)

// EnsurePlatformEntities upserts the platform publisher and creator rows that
// content-seeding ingesters depend on.  It is idempotent — safe to call on
// every seeder run.
func EnsurePlatformEntities(db *gorm.DB) error {
	now := time.Now().UTC()

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SET LOCAL row_security = off").Error; err != nil {
			return fmt.Errorf("disable RLS: %w", err)
		}

		pubSQL := `INSERT INTO mkt_publishers
			(id, name, slug, description, website_url, is_platform, is_verified, created_at, updated_at)
			VALUES (?, 'Project Gutenberg / Public Domain', 'project-gutenberg',
				'Free public-domain books and educational resources curated for homeschooling families.',
				'https://www.gutenberg.org', true, true, ?, ?)
			ON CONFLICT (id) DO NOTHING`
		if err := tx.Exec(pubSQL, PlatformPublisherID, now, now).Error; err != nil {
			return fmt.Errorf("upsert platform publisher: %w", err)
		}

		creatorSQL := `INSERT INTO mkt_creators
			(id, parent_id, store_name, onboarding_status, created_at, updated_at)
			VALUES (?, ?, 'Homegrown Academy (Public Domain)', 'active', ?, ?)
			ON CONFLICT (id) DO NOTHING`
		if err := tx.Exec(creatorSQL, PlatformCreatorID, platformParentID, now, now).Error; err != nil {
			return fmt.Errorf("upsert platform creator: %w", err)
		}

		return nil
	})
	return err
}
