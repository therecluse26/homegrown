-- +goose Up
-- =============================================================================
-- Migration 013: Create marketplace (mkt) domain tables
-- =============================================================================
-- IMPORTANT: mkt_publishers MUST be created before any learn_*_defs tables,
-- as those tables reference mkt_publishers(id) as a foreign key. [06-learn §3.2]
-- =============================================================================

-- Enable pg_trgm extension for trigram autocomplete [ARCH §9.1]
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: mkt_publishers — Multi-user content organizations [S§3.1.4]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_publishers (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    name                  TEXT NOT NULL,
    slug                  TEXT NOT NULL UNIQUE,
    description           TEXT,
    logo_url              TEXT,
    website_url           TEXT,
    is_platform           BOOLEAN NOT NULL DEFAULT false,
    is_verified           BOOLEAN NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mkt_publishers_platform ON mkt_publishers(is_platform) WHERE is_platform = true;
CREATE INDEX IF NOT EXISTS idx_mkt_publishers_slug ON mkt_publishers(slug);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 2: mkt_creators — Creator accounts [S§9.1]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_creators (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    parent_id             UUID NOT NULL UNIQUE REFERENCES iam_parents(id),
    payment_account_id    TEXT,
    onboarding_status     TEXT NOT NULL DEFAULT 'pending'
                          CHECK (onboarding_status IN ('pending', 'onboarding', 'active', 'suspended')),
    store_name            TEXT NOT NULL,
    store_bio             TEXT,
    store_logo_url        TEXT,
    store_banner_url      TEXT,
    tos_accepted_at       TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mkt_creators_parent ON mkt_creators(parent_id);
CREATE INDEX IF NOT EXISTS idx_mkt_creators_status ON mkt_creators(onboarding_status);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 3: mkt_publisher_members — Publisher membership [S§3.1.4]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_publisher_members (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id) ON DELETE CASCADE,
    creator_id            UUID NOT NULL REFERENCES mkt_creators(id) ON DELETE CASCADE,
    role                  TEXT NOT NULL DEFAULT 'member'
                          CHECK (role IN ('owner', 'admin', 'member')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (publisher_id, creator_id)
);

CREATE INDEX IF NOT EXISTS idx_mkt_publisher_members_publisher ON mkt_publisher_members(publisher_id);
CREATE INDEX IF NOT EXISTS idx_mkt_publisher_members_creator ON mkt_publisher_members(creator_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 4: mkt_listings — Content listings [S§9.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_listings (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    creator_id            UUID NOT NULL REFERENCES mkt_creators(id),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT NOT NULL,
    price_cents           INTEGER NOT NULL CHECK (price_cents >= 0),
    methodology_tags      UUID[] NOT NULL,
    subject_tags          TEXT[] NOT NULL,
    grade_min             SMALLINT,
    grade_max             SMALLINT,
    content_type          TEXT NOT NULL CHECK (content_type IN (
                              'curriculum', 'worksheet', 'unit_study',
                              'video', 'book_list', 'assessment',
                              'lesson_plan', 'printable', 'project_guide',
                              'reading_guide', 'course',
                              'interactive_quiz', 'lesson_sequence'
                          )),
    worldview_tags        TEXT[] DEFAULT '{}',
    preview_url           TEXT,
    thumbnail_url         TEXT,
    status                TEXT NOT NULL DEFAULT 'draft'
                          CHECK (status IN ('draft', 'submitted', 'published', 'archived')),
    rating_avg            NUMERIC(3,2) DEFAULT 0,
    rating_count          INTEGER DEFAULT 0,
    version               INTEGER NOT NULL DEFAULT 1,
    published_at          TIMESTAMPTZ,
    archived_at           TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Full-text search on listings [S§9.3, ARCH §9.1]
ALTER TABLE mkt_listings ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX IF NOT EXISTS idx_mkt_listings_search ON mkt_listings USING GIN(search_vector);

-- Faceted filtering indexes [S§9.3]
CREATE INDEX IF NOT EXISTS idx_mkt_listings_methodology ON mkt_listings USING GIN(methodology_tags);
CREATE INDEX IF NOT EXISTS idx_mkt_listings_subject ON mkt_listings USING GIN(subject_tags);
CREATE INDEX IF NOT EXISTS idx_mkt_listings_worldview ON mkt_listings USING GIN(worldview_tags);
CREATE INDEX IF NOT EXISTS idx_mkt_listings_status ON mkt_listings(status) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_mkt_listings_content_type ON mkt_listings(content_type) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_mkt_listings_price ON mkt_listings(price_cents) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_mkt_listings_rating ON mkt_listings(rating_avg DESC) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_mkt_listings_published_at ON mkt_listings(published_at DESC) WHERE status = 'published';
CREATE INDEX IF NOT EXISTS idx_mkt_listings_creator ON mkt_listings(creator_id);
CREATE INDEX IF NOT EXISTS idx_mkt_listings_publisher ON mkt_listings(publisher_id);

-- Trigram index for fuzzy autocomplete [ARCH §9.1]
CREATE INDEX IF NOT EXISTS idx_mkt_listings_title_trgm ON mkt_listings USING GIN(title gin_trgm_ops);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 5: mkt_listing_files — Files attached to listings [S§9.2.1]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_listing_files (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    file_name             TEXT NOT NULL,
    file_size_bytes       BIGINT NOT NULL,
    mime_type             TEXT NOT NULL,
    storage_key           TEXT NOT NULL,
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    version               INTEGER NOT NULL DEFAULT 1,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_mkt_listing_files_listing ON mkt_listing_files(listing_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 6: mkt_listing_versions — Version history [S§9.2.3]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_listing_versions (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    version               INTEGER NOT NULL,
    title                 TEXT NOT NULL,
    description           TEXT NOT NULL,
    price_cents           INTEGER NOT NULL,
    change_summary        TEXT,
    upgrade_policy        TEXT DEFAULT 'free'
                          CHECK (upgrade_policy IN ('free', 'discounted', 'full_price')),
    upgrade_discount_pct  SMALLINT
                          CHECK (upgrade_discount_pct IS NULL OR
                                 (upgrade_discount_pct > 0 AND upgrade_discount_pct <= 100)),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (listing_id, version)
);

CREATE INDEX IF NOT EXISTS idx_mkt_listing_versions_listing ON mkt_listing_versions(listing_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 7: mkt_purchases — Family-scoped purchase records [S§9.4]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_purchases (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id),
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id),
    creator_id            UUID NOT NULL REFERENCES mkt_creators(id),
    payment_id            TEXT,
    payment_session_id    TEXT UNIQUE,
    amount_cents          INTEGER NOT NULL,
    platform_fee_cents    INTEGER NOT NULL,
    creator_payout_cents  INTEGER NOT NULL,
    refunded_at           TIMESTAMPTZ,
    refund_amount_cents   INTEGER,
    refund_id             TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (family_id, listing_id)
);

CREATE INDEX IF NOT EXISTS idx_mkt_purchases_family ON mkt_purchases(family_id);
CREATE INDEX IF NOT EXISTS idx_mkt_purchases_listing ON mkt_purchases(listing_id);
CREATE INDEX IF NOT EXISTS idx_mkt_purchases_creator ON mkt_purchases(creator_id);
CREATE INDEX IF NOT EXISTS idx_mkt_purchases_payment ON mkt_purchases(payment_id) WHERE payment_id IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 8: mkt_reviews — Verified-purchaser reviews [S§9.5]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_reviews (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id),
    purchase_id           UUID NOT NULL REFERENCES mkt_purchases(id),
    family_id             UUID NOT NULL REFERENCES iam_families(id),
    rating                SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    review_text           TEXT,
    is_anonymous          BOOLEAN NOT NULL DEFAULT true,
    moderation_status     TEXT NOT NULL DEFAULT 'pending'
                          CHECK (moderation_status IN ('pending', 'approved', 'rejected')),
    creator_response      TEXT,
    creator_response_at   TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (purchase_id)
);

CREATE INDEX IF NOT EXISTS idx_mkt_reviews_listing ON mkt_reviews(listing_id);
CREATE INDEX IF NOT EXISTS idx_mkt_reviews_family ON mkt_reviews(family_id);
CREATE INDEX IF NOT EXISTS idx_mkt_reviews_moderation ON mkt_reviews(moderation_status) WHERE moderation_status = 'pending';

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 9: mkt_cart_items — Persistent family-scoped cart [S§9.4]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_cart_items (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    added_by_parent_id    UUID NOT NULL REFERENCES iam_parents(id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (family_id, listing_id)
);

CREATE INDEX IF NOT EXISTS idx_mkt_cart_items_family ON mkt_cart_items(family_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 10: mkt_curated_sections — Admin-managed curated sections [S§9.3]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_curated_sections (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    slug                  TEXT NOT NULL UNIQUE,
    display_name          TEXT NOT NULL,
    description           TEXT,
    section_type          TEXT NOT NULL CHECK (section_type IN ('manual', 'auto')),
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 11: mkt_curated_section_items — Links listings to curated sections
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE IF NOT EXISTS mkt_curated_section_items (
    id                    UUID PRIMARY KEY DEFAULT uuidv7(),
    section_id            UUID NOT NULL REFERENCES mkt_curated_sections(id) ON DELETE CASCADE,
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (section_id, listing_id)
);

CREATE INDEX IF NOT EXISTS idx_mkt_curated_section_items_section ON mkt_curated_section_items(section_id);

-- Family scoping is enforced at the GORM level via ScopedTransaction (ADR-008).
-- PostgreSQL RLS is NOT used.

-- ═══════════════════════════════════════════════════════════════════════════════
-- SEED DATA: Built-in curated sections [S§9.3]
-- ═══════════════════════════════════════════════════════════════════════════════
INSERT INTO mkt_curated_sections (slug, display_name, description, section_type, sort_order)
VALUES
    ('featured',     'Featured',     'Hand-picked by the Homegrown Academy team',     'manual', 1),
    ('staff-picks',  'Staff Picks',  'Curriculum our team loves',                     'manual', 2),
    ('trending',     'Trending',     'Popular with families this week',               'auto',   3),
    ('new-arrivals', 'New Arrivals', 'Recently published content',                    'auto',   4)
ON CONFLICT (slug) DO NOTHING;

-- ═══════════════════════════════════════════════════════════════════════════════
-- SEED DATA: Built-in platform publisher [06-learn §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
INSERT INTO mkt_publishers (name, slug, description, is_platform, is_verified)
VALUES (
    'Homegrown Academy',
    'homegrown-academy',
    'Platform-provided educational content and definitions',
    true,
    true
)
ON CONFLICT (slug) DO NOTHING;


-- +goose Down
DROP TABLE IF EXISTS mkt_curated_section_items;
DROP TABLE IF EXISTS mkt_curated_sections;
DROP TABLE IF EXISTS mkt_cart_items;
DROP TABLE IF EXISTS mkt_reviews;
DROP TABLE IF EXISTS mkt_purchases;
DROP TABLE IF EXISTS mkt_listing_versions;
DROP TABLE IF EXISTS mkt_listing_files;
DROP TABLE IF EXISTS mkt_listings;
DROP TABLE IF EXISTS mkt_publisher_members;
DROP TABLE IF EXISTS mkt_creators;
DROP TABLE IF EXISTS mkt_publishers;
