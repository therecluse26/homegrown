# Domain Spec 07 — Marketplace (mkt::)

## §1 Overview

The Marketplace domain is the **revenue engine and content distribution platform** — it connects
curriculum creators to homeschooling families through a commerce layer that handles creator
onboarding, content listing lifecycle, cart-based checkout with split payments, verified-purchaser
reviews, and secure file delivery. Publishers (multi-user organizations) bridge into the learning
domain where all published content definitions reference `mkt_publishers`. Revenue sharing is
enforced at payment time via Hyperswitch split payment rules. `[S§9, V§9, V§10]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/mkt/` |
| **DB prefix** | `mkt_` |
| **Complexity class** | Complex (has `domain/` subdirectory) `[ARCH §4.5]` |
| **CQRS** | Yes — listing commands separated from faceted browse/search reads `[ARCH §4.7]` |
| **External adapter** | `internal/mkt/adapters/payment.go` (Hyperswitch — processor-agnostic payment orchestration) |
| **Key constraint** | Payment adapter is processor-agnostic; Stripe is configured in Hyperswitch, swappable without code changes `[supersedes ADR-007]` |

**What mkt:: owns**: Creator accounts, publisher organizations and membership, content listings
(lifecycle, versioning, files), cart and checkout, purchase records, reviews, curated browse
sections, creator dashboard data, faceted marketplace search, and the Hyperswitch payment adapter
for marketplace transactions (split payments, creator sub-merchant onboarding, payouts).

**What mkt:: does NOT own**: Subscription billing (owned by `billing::`), file upload and media
processing (owned by `media::`), content moderation and CSAM scanning (owned by `safety::`),
notification delivery (owned by `notify::`), search indexing infrastructure (owned by `search::`),
subject taxonomy (owned by `learn::`), methodology definitions (owned by `method::`), student
profiles and family accounts (owned by `iam::`), AI recommendations (owned by `ai::`).

**What mkt:: delegates**: Media upload/validation → `media::` (via adapter interface). Content
safety scanning → `safety::` (via `ListingPublished` event). Notification delivery → `notify::`
(via domain events). Search indexing → `search::` (via domain events). Student/family data
lookup → `iam::IamService`. Methodology tag validation → `method::MethodologyService`.

---

## §2 Requirements Traceability

Every SPEC.md §9 requirement maps to a section in this document. Cross-references from
other spec sections are included where the marketplace domain is involved.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Creator onboarding: legal name, email, TOS agreement | `[S§9.1]` | §3.2 (`mkt_creators`), §4.1, §5 |
| Identity verification and tax info before payouts | `[S§9.1]` | §3.2 (`onboarding_status`), §7 (`PaymentAdapter`), §11 |
| Creator store profile: display name, bio, logo/banner | `[S§9.1]` | §3.2 (`mkt_creators`), §4.1 |
| Onboarding communicates revenue share, payout schedule, policies | `[S§9.1]` | §11, §15 |
| Listing fields: title, description, price, methodology tags, subject tags, grade range, content type, worldview tags, preview, thumbnail, files | `[S§9.2.1]` | §3.2 (`mkt_listings`, `mkt_listing_files`), §8 |
| Listing lifecycle: Draft → Submitted → Published → Archived | `[S§9.2.2]` | §3.2 (`status` CHECK), §9 |
| Creators can update published listings (description, price, files) | `[S§9.2.3]` | §3.2 (`mkt_listing_versions`), §4.1, §5 |
| File updates reflected for existing purchasers | `[S§9.2.3]` | §12 |
| Price changes do not affect existing purchases | `[S§9.2.3]` | §3.2 (`mkt_purchases.amount_cents` immutable) |
| Version history for creator reference | `[S§9.2.3]` | §3.2 (`mkt_listing_versions`), §4.2 |
| Faceted filtering: methodology, subject, grade, content type, price, worldview, rating | `[S§9.3]` | §3.2 (indexes), §13 |
| Full-text search on titles and descriptions | `[S§9.3]` | §3.2 (`search_vector`), §13 |
| Curated sections: Featured, Trending, New Arrivals, Staff Picks | `[S§9.3]` | §3.2 (`mkt_curated_sections`), §13 |
| Methodology-scoped browsing | `[S§9.3]` | §13 |
| Sort by relevance, price, rating, recency | `[S§9.3]` | §13 |
| Cart-based purchase flow | `[S§9.4]` | §3.2 (`mkt_cart_items`), §4.1, §11 |
| Free content "Get" flow | `[S§9.4]` | §11 |
| Content bundles (Phase 3) | `[S§9.4]` | §19 (Phase 3) |
| Immediate access after purchase | `[S§9.4]` | §11, §12 |
| Purchase receipts via email | `[S§9.4]` | §18 (`PurchaseCompleted` → `notify::`) |
| Purchased content accessible regardless of subscription tier | `[S§9.4]` | §12 (no tier check on download) |
| Verified-purchaser ratings only | `[S§9.5]` | §3.2 (`mkt_reviews.purchase_id`), §14 |
| 1-5 star rating scale | `[S§9.5]` | §3.2 (`rating` CHECK), §14 |
| Text feedback on reviews | `[S§9.5]` | §3.2 (`review_text`), §14 |
| Anonymous reviews by default | `[S§9.5]` | §3.2 (`is_anonymous`), §14 |
| Reviews subject to content moderation | `[S§9.5]` | §14 (`safety::` integration) |
| Creator can respond to reviews | `[S§9.5]` | §3.2 (`creator_response`), §4.2, §14 |
| Aggregate rating and review count on listings | `[S§9.5]` | §3.2 (`rating_avg`, `rating_count`), §14 |
| Revenue share model (creator majority) | `[S§9.6]` | §3.2 (`mkt_purchases` split columns), §11, §15 |
| Creator dashboard: sales, earnings, payouts, analytics | `[S§9.6]` | §4.2, §15 |
| Defined payout schedule with minimum threshold | `[S§9.6]` | §15 (Phase 2) |
| Refund deductions from creator earnings | `[S§9.6]` | §3.2 (`refund_*` columns), §11 |
| 1099-K forms for qualifying creators | `[S§9.6]` | §15 (offloaded to processor via Hyperswitch) |
| Purchased content accessible from learning tools | `[S§9.7]` | §18 (`PurchaseCompleted` event → `learn::`) |
| AI recommendations draw from catalog | `[S§9.7]` | §18 (`ListingPublished` event → `ai::`) |
| Content safety / moderation | `[S§3.1.4]` | §14 (`safety::` integration), §18 |
| Marketplace search scope | `[S§14]` | §13 |
| Free tier: marketplace browse and purchase | `[S§15.1]` | §19 |
| Content neutrality — worldview tagging | `[V§12]` | §3.2 (`worldview_tags`), §13 |
| Privacy — family-scoped purchases and cart | `[S§19]` | §3.3 (RLS policies) |
| Security — no internal error details in responses | `[S§18]` | §17 |

---

## §3 Database Schema

The marketplace domain stores creator/publisher identity, content listings with full-text search,
purchase records, reviews, and cart state. Publishers are the cross-domain bridge — all `learn_*_defs`
tables reference `mkt_publishers(id)` as a foreign key. `[ARCH §5.1, ARCH §5.2]`

> **Refinement note**: This spec significantly refines the skeletal marketplace schema in
> ARCHITECTURE.md §5.2. Key changes: (1) `stripe_account_id` → `payment_account_id`
> (processor-agnostic, Hyperswitch merchant ID) `[supersedes ADR-007]`, (2) `stripe_payment_id` →
> `payment_id`, (3) new `mkt_publishers` table as cross-domain bridge, (4) new `mkt_listing_files`
> and `mkt_listing_versions` tables, (5) `mkt_cart_items` for persistent family-scoped cart,
> (6) `mkt_curated_sections` for editorial curation, (7) expanded `mkt_reviews` with moderation
> and anonymity.

### §3.1 Enums

All enums use CHECK constraints for consistency with other domain specs and to avoid
PostgreSQL enum migration limitations. `[ARCH §5.2]`

```sql
-- Listing status, content type, creator onboarding status, publisher role,
-- review moderation status, and worldview tags are all enforced via CHECK
-- constraints on their respective columns rather than as PostgreSQL enum
-- types. This avoids ALTER TYPE limitations when adding new values in
-- future migrations. [ARCH §5.2]
--
-- Listing status values: draft, submitted, published, archived
-- Content type values: curriculum, worksheet, unit_study, video, book_list,
--                      assessment, lesson_plan, printable, project_guide,
--                      reading_guide, course, interactive_quiz, lesson_sequence
-- Creator onboarding status: pending, onboarding, active, suspended
-- Publisher role values: owner, admin, member
-- Review moderation status: pending, approved, rejected
-- Worldview tag values: secular, christian, jewish, islamic, neutral, other
```

### §3.2 Tables

```sql
-- =============================================================================
-- Migration: YYYYMMDD_000001_create_mkt_tables.sql (goose)
-- =============================================================================
-- IMPORTANT: mkt_publishers MUST be created before any learn_*_defs tables,
-- as those tables reference mkt_publishers(id) as a foreign key. [06-learn §3.2]
-- =============================================================================

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: mkt_publishers — Multi-user content organizations [S§3.1.4]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Publishers are organizations (like GitHub orgs) that own and manage published
-- educational content. All learn_*_defs tables reference publisher_id. The
-- platform itself has a built-in publisher (is_platform = true) for
-- platform-provided content. [06-learn §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_publishers (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT NOT NULL,
    slug                  TEXT NOT NULL UNIQUE,          -- URL-safe identifier
    description           TEXT,
    logo_url              TEXT,
    website_url           TEXT,
    is_platform           BOOLEAN NOT NULL DEFAULT false, -- true for the built-in platform publisher
    is_verified           BOOLEAN NOT NULL DEFAULT false, -- admin-verified publishers
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Ensure exactly one platform publisher exists (enforced at app level, not DB constraint)
CREATE INDEX idx_mkt_publishers_platform ON mkt_publishers(is_platform) WHERE is_platform = true;
CREATE INDEX idx_mkt_publishers_slug ON mkt_publishers(slug);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 2: mkt_publisher_members — Publisher membership [S§3.1.4]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Maps creators to publishers with role-based access. A creator can belong to
-- multiple publishers. A publisher must have at least one owner (enforced at
-- app level).
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_publisher_members (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id) ON DELETE CASCADE,
    creator_id            UUID NOT NULL REFERENCES mkt_creators(id) ON DELETE CASCADE,
    role                  TEXT NOT NULL DEFAULT 'member'
                          CHECK (role IN ('owner', 'admin', 'member')),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (publisher_id, creator_id)
);

CREATE INDEX idx_mkt_publisher_members_publisher ON mkt_publisher_members(publisher_id);
CREATE INDEX idx_mkt_publisher_members_creator ON mkt_publisher_members(creator_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 3: mkt_creators — Creator accounts [S§9.1]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Refinement of ARCH §5.2: stripe_account_id → payment_account_id (Hyperswitch
-- merchant/sub-merchant ID). Adds onboarding_status for lifecycle tracking and
-- tos_accepted_at for legal compliance. [supersedes ADR-007]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_creators (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id             UUID NOT NULL UNIQUE REFERENCES iam_parents(id),
    payment_account_id    TEXT,                          -- Hyperswitch sub-merchant ID [supersedes ADR-007]
    onboarding_status     TEXT NOT NULL DEFAULT 'pending'
                          CHECK (onboarding_status IN ('pending', 'onboarding', 'active', 'suspended')),
    store_name            TEXT NOT NULL,
    store_bio             TEXT,
    store_logo_url        TEXT,
    store_banner_url      TEXT,
    tos_accepted_at       TIMESTAMPTZ,                  -- marketplace TOS acceptance timestamp
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_mkt_creators_parent ON mkt_creators(parent_id);
CREATE INDEX idx_mkt_creators_status ON mkt_creators(onboarding_status);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 4: mkt_listings — Content listings [S§9.2]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Refinement of ARCH §5.2: adds published_at, archived_at timestamps for
-- lifecycle tracking. publisher_id links listing to its owning publisher org.
-- subject_tags uses slugs from learn_subject_taxonomy for cross-domain
-- taxonomy consistency. [06-learn §13]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_listings (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creator_id            UUID NOT NULL REFERENCES mkt_creators(id),
    publisher_id          UUID NOT NULL REFERENCES mkt_publishers(id),
    title                 TEXT NOT NULL,
    description           TEXT NOT NULL,
    price_cents           INTEGER NOT NULL CHECK (price_cents >= 0),  -- 0 = free
    methodology_tags      UUID[] NOT NULL,              -- references method_definitions (app-level, no DB FK)
    subject_tags          TEXT[] NOT NULL,               -- slugs from learn_subject_taxonomy [06-learn §13]
    grade_min             SMALLINT,
    grade_max             SMALLINT,
    content_type          TEXT NOT NULL CHECK (content_type IN (
                              'curriculum', 'worksheet', 'unit_study',
                              'video', 'book_list', 'assessment',
                              'lesson_plan', 'printable', 'project_guide',
                              'reading_guide', 'course',
                              'interactive_quiz', 'lesson_sequence'
                          )),                            -- [S§9.2.1, S§8.1.9, S§8.1.12]
    worldview_tags        TEXT[] DEFAULT '{}',           -- [S§9.2.1, V§12]
    preview_url           TEXT,
    thumbnail_url         TEXT,
    status                TEXT NOT NULL DEFAULT 'draft'
                          CHECK (status IN ('draft', 'submitted', 'published', 'archived')),
    rating_avg            NUMERIC(3,2) DEFAULT 0,       -- denormalized from mkt_reviews
    rating_count          INTEGER DEFAULT 0,            -- denormalized from mkt_reviews
    version               INTEGER NOT NULL DEFAULT 1,
    published_at          TIMESTAMPTZ,                  -- set on first publish
    archived_at           TIMESTAMPTZ,                  -- set on archive
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Full-text search on listings [S§9.3, ARCH §9.1]
ALTER TABLE mkt_listings ADD COLUMN search_vector tsvector
    GENERATED ALWAYS AS (
        setweight(to_tsvector('english', coalesce(title, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(description, '')), 'B')
    ) STORED;
CREATE INDEX idx_mkt_listings_search ON mkt_listings USING GIN(search_vector);

-- Faceted filtering indexes [S§9.3]
CREATE INDEX idx_mkt_listings_methodology ON mkt_listings USING GIN(methodology_tags);
CREATE INDEX idx_mkt_listings_subject ON mkt_listings USING GIN(subject_tags);
CREATE INDEX idx_mkt_listings_worldview ON mkt_listings USING GIN(worldview_tags);
CREATE INDEX idx_mkt_listings_status ON mkt_listings(status) WHERE status = 'published';
CREATE INDEX idx_mkt_listings_content_type ON mkt_listings(content_type) WHERE status = 'published';
CREATE INDEX idx_mkt_listings_price ON mkt_listings(price_cents) WHERE status = 'published';
CREATE INDEX idx_mkt_listings_rating ON mkt_listings(rating_avg DESC) WHERE status = 'published';
CREATE INDEX idx_mkt_listings_published_at ON mkt_listings(published_at DESC) WHERE status = 'published';
CREATE INDEX idx_mkt_listings_creator ON mkt_listings(creator_id);
CREATE INDEX idx_mkt_listings_publisher ON mkt_listings(publisher_id);

-- Trigram index for fuzzy autocomplete [ARCH §9.1]
CREATE INDEX idx_mkt_listings_title_trgm ON mkt_listings USING GIN(title gin_trgm_ops);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 5: mkt_listing_files — Files attached to listings [S§9.2.1]
-- ═══════════════════════════════════════════════════════════════════════════════
-- New table (missing from ARCH §5.2). Tracks individual files attached to a
-- listing, with storage keys for R2 signed-URL delivery. [ARCH §8.3]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_listing_files (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    file_name             TEXT NOT NULL,                 -- original filename
    file_size_bytes       BIGINT NOT NULL,
    mime_type             TEXT NOT NULL,                 -- validated via magic bytes [CODING §5.2]
    storage_key           TEXT NOT NULL,                 -- R2 object key
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    version               INTEGER NOT NULL DEFAULT 1,   -- incremented on file update [S§9.2.3]
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_mkt_listing_files_listing ON mkt_listing_files(listing_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 6: mkt_listing_versions — Version history [S§9.2.3]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Snapshot of listing state at each version, for creator reference. Created
-- automatically when a listing is updated after publication.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_listing_versions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    version               INTEGER NOT NULL,
    title                 TEXT NOT NULL,
    description           TEXT NOT NULL,
    price_cents           INTEGER NOT NULL,
    change_summary        TEXT,                          -- creator-provided change description
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (listing_id, version)
);

CREATE INDEX idx_mkt_listing_versions_listing ON mkt_listing_versions(listing_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 7: mkt_purchases — Family-scoped purchase records [S§9.4]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Refinement of ARCH §5.2: stripe_payment_id → payment_id (Hyperswitch payment
-- ID). Adds refund tracking columns and payment_session_id for idempotency.
-- Revenue split recorded immutably at purchase time. [supersedes ADR-007]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_purchases (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id),
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id),
    creator_id            UUID NOT NULL REFERENCES mkt_creators(id),
    payment_id            TEXT,                          -- Hyperswitch payment ID (NULL for free content)
    payment_session_id    TEXT UNIQUE,                   -- idempotency key (Hyperswitch payment ID)
    amount_cents          INTEGER NOT NULL,              -- total charged to family
    platform_fee_cents    INTEGER NOT NULL,              -- platform's share
    creator_payout_cents  INTEGER NOT NULL,              -- creator's share
    refunded_at           TIMESTAMPTZ,                   -- NULL if not refunded
    refund_amount_cents   INTEGER,                       -- partial or full refund amount
    refund_id             TEXT,                          -- Hyperswitch refund ID
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (family_id, listing_id)                      -- one purchase per listing per family
);

CREATE INDEX idx_mkt_purchases_family ON mkt_purchases(family_id);
CREATE INDEX idx_mkt_purchases_listing ON mkt_purchases(listing_id);
CREATE INDEX idx_mkt_purchases_creator ON mkt_purchases(creator_id);
CREATE INDEX idx_mkt_purchases_payment ON mkt_purchases(payment_id) WHERE payment_id IS NOT NULL;

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 8: mkt_reviews — Verified-purchaser reviews [S§9.5]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Refinement of ARCH §5.2: adds moderation_status for safety:: integration,
-- is_anonymous flag for privacy, and creator_response_at timestamp.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_reviews (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id),
    purchase_id           UUID NOT NULL REFERENCES mkt_purchases(id),
    family_id             UUID NOT NULL REFERENCES iam_families(id),
    rating                SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    review_text           TEXT,
    is_anonymous          BOOLEAN NOT NULL DEFAULT true, -- anonymous by default [S§9.5]
    moderation_status     TEXT NOT NULL DEFAULT 'pending'
                          CHECK (moderation_status IN ('pending', 'approved', 'rejected')),
    creator_response      TEXT,                          -- [S§9.5]
    creator_response_at   TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (purchase_id)                                -- one review per purchase
);

CREATE INDEX idx_mkt_reviews_listing ON mkt_reviews(listing_id);
CREATE INDEX idx_mkt_reviews_family ON mkt_reviews(family_id);
CREATE INDEX idx_mkt_reviews_moderation ON mkt_reviews(moderation_status) WHERE moderation_status = 'pending';

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 9: mkt_cart_items — Persistent family-scoped cart [S§9.4]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Cart is family-scoped: if Parent A adds an item, Parent B sees it. This
-- mirrors the family-level purchase model. Persistent in DB (not session-based).
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_cart_items (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_id             UUID NOT NULL REFERENCES iam_families(id) ON DELETE CASCADE,
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    added_by_parent_id    UUID NOT NULL REFERENCES iam_parents(id),
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (family_id, listing_id)                      -- no duplicate items
);

CREATE INDEX idx_mkt_cart_items_family ON mkt_cart_items(family_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 10: mkt_curated_sections — Admin-managed curated sections [S§9.3]
-- ═══════════════════════════════════════════════════════════════════════════════
-- Manual sections (Featured, Staff Picks) have items managed by admins.
-- Auto-computed sections (Trending, New Arrivals) are populated by background
-- jobs. [S§9.3]
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE mkt_curated_sections (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                  TEXT NOT NULL UNIQUE,          -- e.g., 'featured', 'trending'
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
CREATE TABLE mkt_curated_section_items (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    section_id            UUID NOT NULL REFERENCES mkt_curated_sections(id) ON DELETE CASCADE,
    listing_id            UUID NOT NULL REFERENCES mkt_listings(id) ON DELETE CASCADE,
    sort_order            SMALLINT NOT NULL DEFAULT 0,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (section_id, listing_id)
);

CREATE INDEX idx_mkt_curated_section_items_section ON mkt_curated_section_items(section_id);

-- ═══════════════════════════════════════════════════════════════════════════════
-- SEED DATA: Built-in curated sections [S§9.3]
-- ═══════════════════════════════════════════════════════════════════════════════
INSERT INTO mkt_curated_sections (slug, display_name, description, section_type, sort_order)
VALUES
    ('featured',     'Featured',     'Hand-picked by the Homegrown Academy team',     'manual', 1),
    ('staff-picks',  'Staff Picks',  'Curriculum our team loves',                     'manual', 2),
    ('trending',     'Trending',     'Popular with families this week',               'auto',   3),
    ('new-arrivals', 'New Arrivals', 'Recently published content',                    'auto',   4);

-- ═══════════════════════════════════════════════════════════════════════════════
-- SEED DATA: Built-in platform publisher [06-learn §3.2]
-- ═══════════════════════════════════════════════════════════════════════════════
-- The platform publisher owns all platform-provided content definitions.
-- This row MUST exist before any learn_*_defs rows are inserted.
INSERT INTO mkt_publishers (name, slug, description, is_platform, is_verified)
VALUES (
    'Homegrown Academy',
    'homegrown-academy',
    'Platform-provided educational content and definitions',
    true,
    true
);
```

### §3.3 RLS Policies

Row-level security enforces family-scoped data access for cart items, purchases, and reviews.
Listings and publishers are readable by all authenticated users. `[CODING §4.1, 00-core §8]`

```sql
-- Cart items: family-scoped read/write
-- Only the owning family can view or modify their cart.
ALTER TABLE mkt_cart_items ENABLE ROW LEVEL SECURITY;
CREATE POLICY cart_items_family_scope ON mkt_cart_items
    USING (family_id = current_setting('app.current_family_id')::uuid);

-- Purchases: family-scoped read, system-write (created during checkout)
-- Families can view their own purchases. Writes happen through the service
-- layer which sets the family scope.
ALTER TABLE mkt_purchases ENABLE ROW LEVEL SECURITY;
CREATE POLICY purchases_family_read ON mkt_purchases
    FOR SELECT USING (family_id = current_setting('app.current_family_id')::uuid);

-- Reviews: read-all (public), write-family (own reviews only)
-- All authenticated users can read reviews. Only the owning family can
-- create or update their reviews.
ALTER TABLE mkt_reviews ENABLE ROW LEVEL SECURITY;
CREATE POLICY reviews_read_all ON mkt_reviews
    FOR SELECT USING (true);
CREATE POLICY reviews_write_family ON mkt_reviews
    FOR ALL USING (family_id = current_setting('app.current_family_id')::uuid);
```

---

## §4 API Endpoints

All endpoints are prefixed with `/v1/marketplace`. Auth requirements use middleware defined
in `00-core §13`: `AuthContext` for authenticated users, `RequireCreator` for creator-only
endpoints (see §16 for middleware implementation). `[CODING §2.1]`

### §4.1 Phase 1 (~25 endpoints)

#### Creator Onboarding (4 endpoints)

##### `POST /v1/marketplace/creators/register`

Register current user as a marketplace creator.

- **Auth**: `AuthContext`
- **Body**: `RegisterCreatorCommand { store_name, store_bio?, store_logo_url?, tos_accepted: bool }`
- **Validation**: `store_name` 1-100 chars, `tos_accepted` must be true
- **Response**: `201 Created` → `CreatorResponse`
- **Events**: `CreatorOnboarded { creator_id, parent_id }`
- **Side effects**: Creates `mkt_creators` row with `onboarding_status = 'pending'`
- **Error codes**: `409 Conflict` (already registered), `422` (validation failure)

##### `POST /v1/marketplace/creators/onboarding-link`

Generate a Hyperswitch sub-merchant onboarding link for identity verification and payout setup.

- **Auth**: `RequireCreator`
- **Body**: None
- **Response**: `200 OK` → `{ onboarding_url: string }`
- **Side effects**: Calls `PaymentAdapter.CreateOnboardingLink()`. Sets `onboarding_status = 'onboarding'` if currently `pending`.
- **Error codes**: `400` (already active), `502` (payment provider error)

##### `GET /v1/marketplace/creators/me`

Get current user's creator profile and onboarding status.

- **Auth**: `RequireCreator`
- **Response**: `200 OK` → `CreatorResponse`
- **Error codes**: `404` (no creator account)

##### `PUT /v1/marketplace/creators/me`

Update creator store profile.

- **Auth**: `RequireCreator`
- **Body**: `{ store_name?, store_bio?, store_logo_url?, store_banner_url? }`
- **Validation**: `store_name` 1-100 chars if provided
- **Response**: `200 OK` → `CreatorResponse`
- **Error codes**: `422` (validation failure)

#### Publisher Management (4 endpoints)

##### `POST /v1/marketplace/publishers`

Create a new publisher organization. The creating creator becomes the owner.

- **Auth**: `RequireCreator`
- **Body**: `CreatePublisherCommand { name, slug?, description?, logo_url?, website_url? }`
- **Validation**: `name` 1-100 chars, `slug` auto-generated from name if not provided, slug must be unique
- **Response**: `201 Created` → `PublisherResponse`
- **Side effects**: Creates `mkt_publisher_members` row with `role = 'owner'`
- **Error codes**: `409` (slug conflict), `422` (validation)

##### `GET /v1/marketplace/publishers/:publisher_id`

Get publisher details.

- **Auth**: `AuthContext`
- **Response**: `200 OK` → `PublisherResponse`
- **Error codes**: `404`

##### `PUT /v1/marketplace/publishers/:publisher_id`

Update publisher details.

- **Auth**: `RequireCreator` + publisher owner/admin check
- **Body**: `{ name?, description?, logo_url?, website_url? }`
- **Response**: `200 OK` → `PublisherResponse`
- **Error codes**: `403` (not owner/admin), `404`, `422`

##### `GET /v1/marketplace/publishers/:publisher_id/members`

List publisher members.

- **Auth**: `RequireCreator` + publisher member check
- **Response**: `200 OK` → `[]PublisherMemberResponse`
- **Error codes**: `403` (not a member), `404`

#### Listing Commands (5 endpoints)

##### `POST /v1/marketplace/listings`

Create a new listing in draft status.

- **Auth**: `RequireCreator`
- **Body**: `CreateListingCommand { publisher_id, title, description, price_cents, methodology_tags, subject_tags, grade_min?, grade_max?, content_type, worldview_tags?, preview_url?, thumbnail_url? }`
- **Validation**: Title 1-200 chars, description 1-10000 chars, `price_cents >= 0`, at least one methodology tag, at least one subject tag, `grade_min <= grade_max` if both provided, creator must be member of publisher
- **Response**: `201 Created` → `ListingDetailResponse`
- **Error codes**: `403` (not publisher member), `422` (validation)

##### `PUT /v1/marketplace/listings/:listing_id`

Update a listing (allowed in draft or published status).

- **Auth**: `RequireCreator` + listing owner check
- **Body**: `UpdateListingCommand { title?, description?, price_cents?, methodology_tags?, subject_tags?, grade_min?, grade_max?, worldview_tags?, preview_url?, thumbnail_url? }`
- **Validation**: Same rules as create for provided fields
- **Response**: `200 OK` → `ListingDetailResponse`
- **Side effects**: If listing is published, creates a `mkt_listing_versions` snapshot before applying changes. Increments `version`. `[S§9.2.3]`
- **Error codes**: `403` (not listing owner), `404`, `422`

##### `POST /v1/marketplace/listings/:listing_id/submit`

Submit a draft listing for automated content screening.

- **Auth**: `RequireCreator` + listing owner check
- **Body**: None
- **Validation**: Listing must be in `draft` status, must have at least one file attached
- **Response**: `200 OK` → `ListingDetailResponse`
- **Events**: `ListingSubmitted { listing_id, creator_id }`
- **Side effects**: Aggregate root state transition: Draft → Submitted. `[§9]`
- **Error codes**: `400` (no files), `409` (invalid state), `403`, `404`

##### `POST /v1/marketplace/listings/:listing_id/publish`

Publish a submitted listing (Phase 1: auto-publish after submission screening passes).

- **Auth**: `RequireCreator` + listing owner check (Phase 1); admin-only (Phase 3)
- **Body**: None
- **Validation**: Listing must be in `submitted` status
- **Response**: `200 OK` → `ListingDetailResponse`
- **Events**: `ListingPublished { listing_id, publisher_id, content_type, subject_tags }`
- **Side effects**: Aggregate root state transition: Submitted → Published. Sets `published_at`. `[§9]`
- **Error codes**: `409` (invalid state), `403`, `404`

##### `POST /v1/marketplace/listings/:listing_id/archive`

Archive a published listing. Existing purchasers retain access.

- **Auth**: `RequireCreator` + listing owner check
- **Body**: None
- **Validation**: Listing must be in `published` status
- **Response**: `200 OK` → `ListingDetailResponse`
- **Events**: `ListingArchived { listing_id }`
- **Side effects**: Aggregate root state transition: Published → Archived. Sets `archived_at`. `[§9]`
- **Error codes**: `409` (invalid state), `403`, `404`

#### Listing Browse / Queries (4 endpoints)

##### `GET /v1/marketplace/listings`

Browse/search published listings with faceted filtering.

- **Auth**: `AuthContext` (or public — TBD based on discovery needs)
- **Query**: `BrowseListingsParams { q?, methodology_ids?, subject_slugs?, grade_min?, grade_max?, content_type?, worldview_tags?, price_min?, price_max?, min_rating?, sort_by?, cursor?, limit? }`
- **Sort options**: `relevance` (default when `q` provided), `price_asc`, `price_desc`, `rating`, `newest`
- **Response**: `200 OK` → `PaginatedResponse<ListingBrowseResponse>`
- **Error codes**: `422` (invalid filter params)

##### `GET /v1/marketplace/listings/:listing_id`

Get listing detail.

- **Auth**: `AuthContext`
- **Response**: `200 OK` → `ListingDetailResponse`
- **Side effects**: None (view counting deferred to Phase 2)
- **Error codes**: `404`

##### `GET /v1/marketplace/listings/autocomplete`

Trigram-based autocomplete for listing titles.

- **Auth**: `AuthContext`
- **Query**: `{ q: string, limit?: uint8 }`
- **Validation**: `q` minimum 2 chars, `limit` max 10
- **Response**: `200 OK` → `[]AutocompleteResult`
- **Error codes**: `422`

##### `GET /v1/marketplace/curated-sections`

Get all active curated sections with their listings.

- **Auth**: `AuthContext`
- **Query**: `{ items_per_section?: uint8 }`
- **Response**: `200 OK` → `[]CuratedSectionResponse`

#### Cart & Checkout (4 endpoints)

##### `POST /v1/marketplace/cart/items`

Add a listing to the family cart.

- **Auth**: `AuthContext`
- **Body**: `AddToCartCommand { listing_id }`
- **Validation**: Listing must be published, not already purchased by family, not already in cart
- **Response**: `201 Created` → `CartResponse`
- **Error codes**: `409` (already in cart or purchased), `404` (listing not found or not published)

##### `DELETE /v1/marketplace/cart/items/:listing_id`

Remove a listing from the family cart.

- **Auth**: `AuthContext`
- **Response**: `200 OK` → `CartResponse`
- **Error codes**: `404` (not in cart)

##### `GET /v1/marketplace/cart`

Get the family's current cart.

- **Auth**: `AuthContext`
- **Response**: `200 OK` → `CartResponse`

##### `POST /v1/marketplace/cart/checkout`

Create a Hyperswitch payment session for cart items and redirect to checkout.

- **Auth**: `AuthContext`
- **Body**: `CreateCheckoutCommand {}` (empty — uses current cart contents)
- **Validation**: Cart must not be empty, all cart items must still be published
- **Response**: `200 OK` → `{ checkout_url: string, payment_session_id: string }`
- **Side effects**: Calls `PaymentAdapter.CreatePayment()` with split rules per listing. `[§11]`
- **Error codes**: `400` (empty cart), `409` (stale cart — items unpublished), `502` (payment provider)

#### Purchases & Downloads (2 endpoints)

##### `GET /v1/marketplace/purchases`

List the family's purchases.

- **Auth**: `AuthContext`
- **Query**: `PurchaseQueryParams { cursor?, limit? }`
- **Response**: `200 OK` → `PaginatedResponse<PurchaseResponse>`

##### `GET /v1/marketplace/purchases/:listing_id/download/:file_id`

Download a purchased file via time-limited signed URL.

- **Auth**: `AuthContext`
- **Response**: `200 OK` → `DownloadResponse { download_url, expires_at }`
- **Validation**: Family must have purchased the listing (no tier check — accessible regardless of subscription) `[S§9.4]`
- **Side effects**: Generates 1-hour R2 signed URL via `media::` adapter `[ARCH §8.3]`
- **Error codes**: `403` (not purchased), `404` (file not found)

#### Reviews (1 endpoint)

##### `POST /v1/marketplace/listings/:listing_id/reviews`

Create a review for a purchased listing.

- **Auth**: `AuthContext`
- **Body**: `CreateReviewCommand { rating, review_text?, is_anonymous? }`
- **Validation**: Rating 1-5, `review_text` max 5000 chars, family must have purchased the listing, one review per purchase
- **Response**: `201 Created` → `ReviewResponse`
- **Events**: `ReviewCreated { review_id, listing_id, rating }`
- **Side effects**: Updates `mkt_listings.rating_avg` and `rating_count` `[§14]`. Sends review text to `safety::` for moderation.
- **Error codes**: `403` (not purchased), `409` (already reviewed), `422` (validation)

#### Webhooks (1 endpoint)

##### `POST /v1/webhooks/payments`

Receive Hyperswitch payment event webhooks.

- **Auth**: Webhook signature verification via `PaymentAdapter.VerifyWebhook()`
- **Body**: Raw Hyperswitch webhook payload
- **Response**: `200 OK`
- **Side effects**: On `payment_succeeded`: create `mkt_purchases` rows, clear cart items, publish `PurchaseCompleted` events. On `payment_failed`: log failure. On `refund_succeeded`: update purchase refund columns. `[§11]`
- **Error codes**: `401` (invalid signature), `400` (malformed payload)

### §4.2 Phase 2 (~8 endpoints)

##### `PUT /v1/marketplace/listings/:listing_id/reviews/:review_id`

Update own review.

- **Auth**: `AuthContext` + review owner check
- **Body**: `UpdateReviewCommand { rating?, review_text?, is_anonymous? }`
- **Response**: `200 OK` → `ReviewResponse`

##### `DELETE /v1/marketplace/listings/:listing_id/reviews/:review_id`

Delete own review.

- **Auth**: `AuthContext` + review owner check
- **Response**: `204 No Content`

##### `POST /v1/marketplace/listings/:listing_id/reviews/:review_id/respond`

Creator responds to a review.

- **Auth**: `RequireCreator` + listing owner check
- **Body**: `RespondToReviewCommand { response_text }`
- **Response**: `200 OK` → `ReviewResponse`

##### `POST /v1/marketplace/listings/:listing_id/get`

"Get" flow for free content (no payment processor involvement).

- **Auth**: `AuthContext`
- **Validation**: Listing must be published, `price_cents == 0`, not already owned
- **Response**: `201 Created` → `PurchaseResponse`
- **Events**: `PurchaseCompleted` (with `amount_cents = 0`)
- **Side effects**: Direct insert into `mkt_purchases` with zero amounts

##### `GET /v1/marketplace/creators/dashboard`

Creator dashboard with sales and earnings data.

- **Auth**: `RequireCreator`
- **Query**: `{ period?: string }` (default: last 30 days)
- **Response**: `200 OK` → `CreatorDashboardResponse`

##### `GET /v1/marketplace/creators/listings`

List creator's own listings (all statuses).

- **Auth**: `RequireCreator`
- **Query**: `CreatorListingQueryParams { status?, cursor?, limit? }`
- **Response**: `200 OK` → `PaginatedResponse<ListingDetailResponse>`

##### `POST /v1/marketplace/creators/payouts`

Request a payout of accumulated earnings via Hyperswitch payout API.

- **Auth**: `RequireCreator`
- **Validation**: `onboarding_status` must be `active`, minimum payout threshold met
- **Response**: `200 OK` → `{ payout_id: string, amount_cents: int64, status: string }`
- **Side effects**: Calls `PaymentAdapter.CreatePayout()`

##### `POST /v1/marketplace/publishers/:publisher_id/members`

Add a member to a publisher.

- **Auth**: `RequireCreator` + publisher owner/admin check
- **Body**: `{ creator_id: uuid.UUID, role: string }`
- **Response**: `201 Created` → `PublisherMemberResponse`

### §4.3 Phase 3 (~4 endpoints)

##### `POST /v1/marketplace/listings/:listing_id/bundle`

Create a content bundle linking multiple listings.

- **Auth**: `RequireCreator`
- **Body**: `{ listing_ids: []uuid.UUID, bundle_price_cents: int64 }`

##### `POST /v1/marketplace/admin/listings/:listing_id/publish`

Admin-only publish for moderated content.

- **Auth**: Admin role (TBD)

##### `DELETE /v1/marketplace/publishers/:publisher_id/members/:member_id`

Remove a member from a publisher.

- **Auth**: `RequireCreator` + publisher owner check

##### `GET /v1/marketplace/listings/:listing_id/versions`

List version history for a listing.

- **Auth**: `RequireCreator` + listing owner check
- **Response**: `200 OK` → `[]ListingVersionResponse`

---

## §5 Service Interface

The `MarketplaceService` interface defines all use cases exposed to handlers and other domains.
Commands and queries are clearly separated per CQRS conventions. `[ARCH §4.7, CODING §8.2]`

```go
// internal/mkt/ports.go

// MarketplaceService defines all marketplace use cases.
// Command methods return only IDs or error — never rich reads after write. [CODING §8.5]
type MarketplaceService interface {
    // ─── Command side (write, has side effects) ─────────────────────────

    // Creator onboarding
    RegisterCreator(ctx context.Context, cmd RegisterCreatorCommand, auth *AuthContext) (uuid.UUID, error)
    UpdateCreatorProfile(ctx context.Context, cmd UpdateCreatorProfileCommand, creatorID uuid.UUID) error
    CreateOnboardingLink(ctx context.Context, creatorID uuid.UUID) (string, error)

    // Publisher management
    CreatePublisher(ctx context.Context, cmd CreatePublisherCommand, creatorID uuid.UUID) (uuid.UUID, error)
    UpdatePublisher(ctx context.Context, cmd UpdatePublisherCommand, publisherID, creatorID uuid.UUID) error
    AddPublisherMember(ctx context.Context, publisherID, creatorID uuid.UUID, role string, actingCreatorID uuid.UUID) error
    RemovePublisherMember(ctx context.Context, publisherID, memberID, actingCreatorID uuid.UUID) error

    // Listing lifecycle
    CreateListing(ctx context.Context, cmd CreateListingCommand, creatorID uuid.UUID) (uuid.UUID, error)
    UpdateListing(ctx context.Context, cmd UpdateListingCommand, listingID, creatorID uuid.UUID) error
    SubmitListing(ctx context.Context, listingID, creatorID uuid.UUID) error
    PublishListing(ctx context.Context, listingID, creatorID uuid.UUID) error
    ArchiveListing(ctx context.Context, listingID, creatorID uuid.UUID) error
    UploadListingFile(ctx context.Context, cmd UploadListingFileCommand, listingID, creatorID uuid.UUID) (uuid.UUID, error)

    // Cart & checkout
    AddToCart(ctx context.Context, listingID uuid.UUID, scope FamilyScope, parentID uuid.UUID) error
    RemoveFromCart(ctx context.Context, listingID uuid.UUID, scope FamilyScope) error
    CreateCheckout(ctx context.Context, scope FamilyScope) (*CheckoutSession, error)
    HandlePaymentWebhook(ctx context.Context, payload []byte, signature string) error

    // Reviews
    CreateReview(ctx context.Context, cmd CreateReviewCommand, listingID uuid.UUID, scope FamilyScope) (uuid.UUID, error)
    UpdateReview(ctx context.Context, cmd UpdateReviewCommand, reviewID uuid.UUID, scope FamilyScope) error
    DeleteReview(ctx context.Context, reviewID uuid.UUID, scope FamilyScope) error
    RespondToReview(ctx context.Context, cmd RespondToReviewCommand, reviewID, creatorID uuid.UUID) error

    // Free content acquisition
    GetFreeListing(ctx context.Context, listingID uuid.UUID, scope FamilyScope) (uuid.UUID, error)

    // Payouts (Phase 2)
    RequestPayout(ctx context.Context, creatorID uuid.UUID) (*PayoutResult, error)

    // Event handlers (cross-domain reactions)
    HandleContentFlagged(ctx context.Context, listingID uuid.UUID, reason string) error
    HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error

    // ─── Query side (read, no side effects) ─────────────────────────────

    // Creator queries
    GetCreatorByParentID(ctx context.Context, parentID uuid.UUID) (*CreatorResponse, error)
    GetCreatorDashboard(ctx context.Context, creatorID uuid.UUID, period DashboardPeriod) (*CreatorDashboardResponse, error)
    GetCreatorListings(ctx context.Context, creatorID uuid.UUID, params CreatorListingQueryParams) (*PaginatedResponse[ListingDetailResponse], error)

    // Publisher queries
    GetPublisher(ctx context.Context, publisherID uuid.UUID) (*PublisherResponse, error)
    GetPublisherMembers(ctx context.Context, publisherID, creatorID uuid.UUID) ([]PublisherMemberResponse, error)
    // VerifyPublisherMembership verifies that creatorID is a member of publisherID.
    // Consumed by learn:: for publisher ownership checks. [06-learn §18.2]
    VerifyPublisherMembership(ctx context.Context, publisherID, creatorID uuid.UUID) (bool, error)

    // Listing browse
    BrowseListings(ctx context.Context, params BrowseListingsParams) (*PaginatedResponse[ListingBrowseResponse], error)
    GetListing(ctx context.Context, listingID uuid.UUID) (*ListingDetailResponse, error)
    AutocompleteListings(ctx context.Context, query string, limit uint8) ([]AutocompleteResult, error)
    GetCuratedSections(ctx context.Context, itemsPerSection uint8) ([]CuratedSectionResponse, error)

    // Cart queries
    GetCart(ctx context.Context, scope FamilyScope) (*CartResponse, error)

    // Purchase queries
    GetPurchases(ctx context.Context, scope FamilyScope, params PurchaseQueryParams) (*PaginatedResponse[PurchaseResponse], error)
    GetDownloadURL(ctx context.Context, listingID, fileID uuid.UUID, scope FamilyScope) (*DownloadResponse, error)

    // Review queries
    GetListingReviews(ctx context.Context, listingID uuid.UUID, params ReviewQueryParams) (*PaginatedResponse[ReviewResponse], error)
}
```

### Implementation

```go
// internal/mkt/service.go

type MarketplaceServiceImpl struct {
    creators       CreatorRepository
    publishers     PublisherRepository
    listings       ListingRepository
    listingFiles   ListingFileRepository
    cart           CartRepository
    purchases      PurchaseRepository
    reviews        ReviewRepository
    curatedSections CuratedSectionRepository
    payment        PaymentAdapter
    media          MediaAdapter
    events         *EventBus
}

func NewMarketplaceService(
    creators CreatorRepository,
    publishers PublisherRepository,
    listings ListingRepository,
    listingFiles ListingFileRepository,
    cart CartRepository,
    purchases PurchaseRepository,
    reviews ReviewRepository,
    curatedSections CuratedSectionRepository,
    payment PaymentAdapter,
    media MediaAdapter,
    events *EventBus,
) *MarketplaceServiceImpl {
    return &MarketplaceServiceImpl{
        creators:       creators,
        publishers:     publishers,
        listings:       listings,
        listingFiles:   listingFiles,
        cart:           cart,
        purchases:      purchases,
        reviews:        reviews,
        curatedSections: curatedSections,
        payment:        payment,
        media:          media,
        events:         events,
    }
}

// MarketplaceServiceImpl implements MarketplaceService.
var _ MarketplaceService = (*MarketplaceServiceImpl)(nil)
```

---

## §6 Repository Interfaces

Each repository trait maps to one or more database tables. Family-scoped methods accept
`FamilyScope` to enforce privacy. Non-family-scoped methods (creators, publishers, listings)
are documented as such. `[CODING §8.2]`

```go
// internal/mkt/ports.go (continued)

// ─── CreatorRepository ──────────────────────────────────────────────────
// NOT family-scoped — creator accounts are per-parent, not per-family.
type CreatorRepository interface {
    Create(ctx context.Context, cmd CreateCreator) (*MktCreator, error)
    GetByID(ctx context.Context, creatorID uuid.UUID) (*MktCreator, error)
    GetByParentID(ctx context.Context, parentID uuid.UUID) (*MktCreator, error)
    Update(ctx context.Context, creatorID uuid.UUID, cmd UpdateCreator) (*MktCreator, error)
    SetOnboardingStatus(ctx context.Context, creatorID uuid.UUID, status string) error
    SetPaymentAccountID(ctx context.Context, creatorID uuid.UUID, paymentAccountID string) error
}

// ─── PublisherRepository ────────────────────────────────────────────────
// NOT family-scoped — publishers are organization-level entities.
type PublisherRepository interface {
    Create(ctx context.Context, cmd CreatePublisher) (*MktPublisher, error)
    GetByID(ctx context.Context, publisherID uuid.UUID) (*MktPublisher, error)
    GetBySlug(ctx context.Context, slug string) (*MktPublisher, error)
    Update(ctx context.Context, publisherID uuid.UUID, cmd UpdatePublisher) (*MktPublisher, error)
    GetPlatformPublisher(ctx context.Context) (*MktPublisher, error)

    // Membership
    AddMember(ctx context.Context, publisherID, creatorID uuid.UUID, role string) error
    RemoveMember(ctx context.Context, publisherID, creatorID uuid.UUID) error
    GetMembers(ctx context.Context, publisherID uuid.UUID) ([]PublisherMemberRow, error)
    GetMemberRole(ctx context.Context, publisherID, creatorID uuid.UUID) (*string, error)
    GetPublishersForCreator(ctx context.Context, creatorID uuid.UUID) ([]MktPublisher, error)
}

// ─── ListingRepository ──────────────────────────────────────────────────
// NOT family-scoped — listings are creator-owned, publicly browsable.
// CQRS: command methods (create/update/save) and query methods (browse/get)
// are clearly separated. [CODING §8.5]
type ListingRepository interface {
    // Command side
    Create(ctx context.Context, cmd CreateListing) (*MktListing, error)
    Save(ctx context.Context, listing *MarketplaceListing) error
    CreateVersionSnapshot(ctx context.Context, listingID uuid.UUID, version int32, title, description string, priceCents int32, changeSummary *string) error

    // Query side
    GetByID(ctx context.Context, listingID uuid.UUID) (*MktListing, error)
    Browse(ctx context.Context, params *BrowseListingsParams) (*PaginatedResponse[ListingBrowseRow], error)
    Autocomplete(ctx context.Context, query string, limit uint8) ([]AutocompleteRow, error)
    GetByCreator(ctx context.Context, creatorID uuid.UUID, params *CreatorListingQueryParams) (*PaginatedResponse[MktListing], error)
    GetVersions(ctx context.Context, listingID uuid.UUID) ([]MktListingVersion, error)
    CountFiles(ctx context.Context, listingID uuid.UUID) (int64, error)
}

// ─── ListingFileRepository ──────────────────────────────────────────────
type ListingFileRepository interface {
    Create(ctx context.Context, cmd CreateListingFile) (*MktListingFile, error)
    GetByID(ctx context.Context, listingID, fileID uuid.UUID) (*MktListingFile, error)
    ListByListing(ctx context.Context, listingID uuid.UUID) ([]MktListingFile, error)
    Delete(ctx context.Context, fileID uuid.UUID) error
}

// ─── CartRepository ─────────────────────────────────────────────────────
// Family-scoped — cart belongs to the family. [00-core §8]
type CartRepository interface {
    AddItem(ctx context.Context, listingID, parentID uuid.UUID, scope FamilyScope) error
    RemoveItem(ctx context.Context, listingID uuid.UUID, scope FamilyScope) error
    GetItems(ctx context.Context, scope FamilyScope) ([]CartItemRow, error)
    Clear(ctx context.Context, scope FamilyScope) error
}

// ─── PurchaseRepository ─────────────────────────────────────────────────
// Family-scoped on read queries. System-scoped writes (from webhook handler).
type PurchaseRepository interface {
    Create(ctx context.Context, cmd CreatePurchase) (*MktPurchase, error)
    GetByFamilyAndListing(ctx context.Context, familyID, listingID uuid.UUID) (*MktPurchase, error)
    ListByFamily(ctx context.Context, scope FamilyScope, params *PurchaseQueryParams) (*PaginatedResponse[PurchaseRow], error)
    GetByPaymentSessionID(ctx context.Context, sessionID string) (*MktPurchase, error)
    SetRefund(ctx context.Context, purchaseID uuid.UUID, refundID string, refundAmountCents int32) error
    GetCreatorSales(ctx context.Context, creatorID uuid.UUID, from, to time.Time) ([]SalesRow, error)
    DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// ─── ReviewRepository ───────────────────────────────────────────────────
// Family-scoped writes, public reads.
type ReviewRepository interface {
    Create(ctx context.Context, cmd CreateReview) (*MktReview, error)
    GetByID(ctx context.Context, reviewID uuid.UUID) (*MktReview, error)
    Update(ctx context.Context, reviewID uuid.UUID, cmd UpdateReview) (*MktReview, error)
    Delete(ctx context.Context, reviewID uuid.UUID) error
    SetCreatorResponse(ctx context.Context, reviewID uuid.UUID, responseText string) error
    ListByListing(ctx context.Context, listingID uuid.UUID, params *ReviewQueryParams) (*PaginatedResponse[ReviewRow], error)
    GetAggregateRating(ctx context.Context, listingID uuid.UUID) (float64, int32, error)
    SetModerationStatus(ctx context.Context, reviewID uuid.UUID, status string) error
}

// ─── CuratedSectionRepository ───────────────────────────────────────────
type CuratedSectionRepository interface {
    ListActive(ctx context.Context) ([]MktCuratedSection, error)
    GetSectionItems(ctx context.Context, sectionID uuid.UUID, limit uint8) ([]ListingBrowseRow, error)
    AddItem(ctx context.Context, sectionID, listingID uuid.UUID, sortOrder int16) error
    RemoveItem(ctx context.Context, sectionID, listingID uuid.UUID) error
    RefreshAutoSection(ctx context.Context, sectionSlug string) error
}
```

---

## §7 Adapter Interfaces

### PaymentAdapter (Hyperswitch)

The `PaymentAdapter` trait wraps Hyperswitch's REST API. It is processor-agnostic — Stripe is
configured as the initial connector in Hyperswitch, swappable without code changes. `[ARCH §4.3]`

> **Refinement note**: This supersedes ARCHITECTURE.md ADR-007 (direct Stripe Connect). The
> adapter talks to Hyperswitch (self-hosted Docker sidecar), which orchestrates the underlying
> payment processor. `mkt::` owns Hyperswitch for marketplace payments (split payments, creator
> sub-merchants, payouts). `billing::` owns Hyperswitch for subscription billing. Both talk to
> the same Hyperswitch instance but use different profiles/business units.

> **Infrastructure note**: Hyperswitch is self-hosted as a Docker sidecar alongside the app:
> `hyperswitch-app` + its own Postgres + Redis containers. Configured with Stripe as the initial
> payment connector — can add Adyen, Xendit, etc. later without code changes. See §18.5 for
> deployment topology.

```go
// internal/mkt/adapters/payment.go

// PaymentAdapter is a processor-agnostic payment adapter backed by Hyperswitch.
// Implementations: HyperswitchPaymentAdapter (production), MockPaymentAdapter (tests).
// Uses net/http to communicate with the Hyperswitch REST API.
type PaymentAdapter interface {
    // ─── Account Management ─────────────────────────────────────────────
    // Sub-merchant onboarding for creator payouts. Maps to Hyperswitch's
    // Organization → Merchant → Profile hierarchy.

    // CreateSubMerchant creates a sub-merchant account in Hyperswitch for a creator.
    // Returns the Hyperswitch merchant/sub-merchant ID.
    CreateSubMerchant(ctx context.Context, config SubMerchantConfig) (string, error)

    // CreateOnboardingLink generates an onboarding link for creator KYC/identity verification.
    CreateOnboardingLink(ctx context.Context, paymentAccountID, returnURL string) (string, error)

    // GetAccountStatus checks the onboarding/verification status of a sub-merchant account.
    GetAccountStatus(ctx context.Context, paymentAccountID string) (PaymentAccountStatus, error)

    // ─── Payments ───────────────────────────────────────────────────────

    // CreatePayment creates a payment session with split payment rules.
    CreatePayment(ctx context.Context, lineItems []PaymentLineItem, splitRules []SplitRule, returnURL string, metadata map[string]string) (*PaymentSession, error)

    // GetPaymentStatus checks the status of a payment.
    GetPaymentStatus(ctx context.Context, paymentID string) (PaymentStatus, error)

    // ─── Payouts (Phase 2) ──────────────────────────────────────────────

    // CreatePayout creates a payout to a creator's connected account.
    CreatePayout(ctx context.Context, paymentAccountID string, amountCents int64, currency string) (*PayoutResult, error)

    // ─── Refunds ────────────────────────────────────────────────────────

    // CreateRefund creates a refund for a payment.
    CreateRefund(ctx context.Context, paymentID string, amountCents int64, reason string) (*RefundResult, error)

    // ─── Webhooks ───────────────────────────────────────────────────────

    // VerifyWebhook verifies the signature of an incoming Hyperswitch webhook.
    VerifyWebhook(ctx context.Context, payload []byte, signature string) (bool, error)

    // ParseEvent parses a verified webhook payload into a domain event.
    ParseEvent(ctx context.Context, payload []byte) (*PaymentEvent, error)
}

// ─── Supporting Types ───────────────────────────────────────────────────

type SubMerchantConfig struct {
    CreatorID uuid.UUID `json:"creator_id"`
    StoreName string    `json:"store_name"`
    Email     string    `json:"email"`
    Country   string    `json:"country"` // ISO 3166-1 alpha-2
}

type PaymentAccountStatus int

const (
    PaymentAccountStatusPending    PaymentAccountStatus = iota // Account created but onboarding not started
    PaymentAccountStatusOnboarding                             // Creator is going through KYC/verification
    PaymentAccountStatusActive                                 // Fully verified, can receive payouts
    PaymentAccountStatusSuspended                              // Account suspended (compliance issue)
)

type PaymentLineItem struct {
    ListingID   uuid.UUID `json:"listing_id"`
    AmountCents int64     `json:"amount_cents"`
    Description string    `json:"description"`
}

type SplitRule struct {
    RecipientAccountID string `json:"recipient_account_id"` // creator's Hyperswitch sub-merchant ID
    AmountCents        int64  `json:"amount_cents"`         // creator's share
}

type PaymentSession struct {
    CheckoutURL      string `json:"checkout_url"`
    PaymentSessionID string `json:"payment_session_id"` // Hyperswitch payment ID (idempotency key)
}

type PaymentStatus int

const (
    PaymentStatusProcessing PaymentStatus = iota
    PaymentStatusSucceeded
    PaymentStatusFailed
    PaymentStatusCancelled
)

type PayoutResult struct {
    PayoutID    string `json:"payout_id"`
    AmountCents int64  `json:"amount_cents"`
    Status      string `json:"status"`
}

type RefundResult struct {
    RefundID    string `json:"refund_id"`
    AmountCents int64  `json:"amount_cents"`
    Status      string `json:"status"`
}

type PaymentEvent struct {
    Type        string            // "payment_succeeded", "payment_failed", "refund_succeeded", "payout_completed"
    PaymentID   string
    Metadata    map[string]string
    Reason      string
    RefundID    string
    AmountCents int64
    MerchantID  string
    PayoutID    string
}
```

### MediaAdapter

File upload and signed-URL generation delegated to `media::`. `[ARCH §4.2]`

```go
// Consumed from media:: — not defined here. See media:: domain spec.
// Used for: presigned upload URLs (creator uploads), presigned download
// URLs (purchaser downloads). [ARCH §8.3]
type MediaAdapter interface {
    PresignedUpload(ctx context.Context, key, contentType string, maxSizeBytes uint64) (string, error)
    PresignedGet(ctx context.Context, key string, expiresSeconds uint32) (string, error)
}
```

---

## §8 Models (DTOs)

All request/response types use struct tags for JSON serialization and swaggo annotations
for OpenAPI code generation. `[CODING §2.1]`

### §8.1 Request Types

```go
// internal/mkt/models.go

type RegisterCreatorCommand struct {
    StoreName    string  `json:"store_name" validate:"required,min=1,max=100"`
    StoreBio     *string `json:"store_bio,omitempty"`
    StoreLogoURL *string `json:"store_logo_url,omitempty"`
    TOSAccepted  bool    `json:"tos_accepted" validate:"required"`
}

type UpdateCreatorProfileCommand struct {
    StoreName     *string `json:"store_name,omitempty" validate:"omitempty,min=1,max=100"`
    StoreBio      *string `json:"store_bio,omitempty"`
    StoreLogoURL  *string `json:"store_logo_url,omitempty"`
    StoreBannerURL *string `json:"store_banner_url,omitempty"`
}

type CreatePublisherCommand struct {
    Name        string  `json:"name" validate:"required,min=1,max=100"`
    Slug        *string `json:"slug,omitempty"`
    Description *string `json:"description,omitempty"`
    LogoURL     *string `json:"logo_url,omitempty"`
    WebsiteURL  *string `json:"website_url,omitempty"`
}

type UpdatePublisherCommand struct {
    Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
    Description *string `json:"description,omitempty"`
    LogoURL     *string `json:"logo_url,omitempty"`
    WebsiteURL  *string `json:"website_url,omitempty"`
}

type CreateListingCommand struct {
    PublisherID    uuid.UUID   `json:"publisher_id" validate:"required"`
    Title          string      `json:"title" validate:"required,min=1,max=200"`
    Description    string      `json:"description" validate:"required,min=1,max=10000"`
    PriceCents     int32       `json:"price_cents" validate:"gte=0"`
    MethodologyTags []uuid.UUID `json:"methodology_tags" validate:"required,min=1"`
    SubjectTags    []string    `json:"subject_tags" validate:"required,min=1"`
    GradeMin       *int16      `json:"grade_min,omitempty"`
    GradeMax       *int16      `json:"grade_max,omitempty"`
    ContentType    string      `json:"content_type" validate:"required"`
    WorldviewTags  []string    `json:"worldview_tags,omitempty"`
    PreviewURL     *string     `json:"preview_url,omitempty"`
    ThumbnailURL   *string     `json:"thumbnail_url,omitempty"`
}

type UpdateListingCommand struct {
    Title          *string      `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
    Description    *string      `json:"description,omitempty" validate:"omitempty,min=1,max=10000"`
    PriceCents     *int32       `json:"price_cents,omitempty" validate:"omitempty,gte=0"`
    MethodologyTags []uuid.UUID `json:"methodology_tags,omitempty"`
    SubjectTags    []string     `json:"subject_tags,omitempty"`
    GradeMin       *int16       `json:"grade_min,omitempty"`
    GradeMax       *int16       `json:"grade_max,omitempty"`
    WorldviewTags  []string     `json:"worldview_tags,omitempty"`
    PreviewURL     *string      `json:"preview_url,omitempty"`
    ThumbnailURL   *string      `json:"thumbnail_url,omitempty"`
}

type UploadListingFileCommand struct {
    FileName      string `json:"file_name" validate:"required"`
    FileSizeBytes int64  `json:"file_size_bytes" validate:"required"`
    MimeType      string `json:"mime_type" validate:"required"`
}

type AddToCartCommand struct {
    ListingID uuid.UUID `json:"listing_id" validate:"required"`
}

type CreateCheckoutCommand struct{}

type CreateReviewCommand struct {
    Rating      int16   `json:"rating" validate:"required,min=1,max=5"`
    ReviewText  *string `json:"review_text,omitempty" validate:"omitempty,max=5000"`
    IsAnonymous *bool   `json:"is_anonymous,omitempty"`
}

type UpdateReviewCommand struct {
    Rating      *int16  `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
    ReviewText  *string `json:"review_text,omitempty" validate:"omitempty,max=5000"`
    IsAnonymous *bool   `json:"is_anonymous,omitempty"`
}

type RespondToReviewCommand struct {
    ResponseText string `json:"response_text" validate:"required"`
}
```

### §8.2 Response Types

```go
type CreatorResponse struct {
    ID               uuid.UUID  `json:"id"`
    ParentID         uuid.UUID  `json:"parent_id"`
    OnboardingStatus string     `json:"onboarding_status"`
    StoreName        string     `json:"store_name"`
    StoreBio         *string    `json:"store_bio,omitempty"`
    StoreLogoURL     *string    `json:"store_logo_url,omitempty"`
    StoreBannerURL   *string    `json:"store_banner_url,omitempty"`
    CreatedAt        time.Time  `json:"created_at"`
}

type PublisherResponse struct {
    ID          uuid.UUID `json:"id"`
    Name        string    `json:"name"`
    Slug        string    `json:"slug"`
    Description *string   `json:"description,omitempty"`
    LogoURL     *string   `json:"logo_url,omitempty"`
    WebsiteURL  *string   `json:"website_url,omitempty"`
    IsVerified  bool      `json:"is_verified"`
    MemberCount int32     `json:"member_count"`
}

type PublisherMemberResponse struct {
    CreatorID uuid.UUID `json:"creator_id"`
    StoreName string    `json:"store_name"`
    Role      string    `json:"role"`
    JoinedAt  time.Time `json:"joined_at"`
}

type ListingBrowseResponse struct {
    ID                 uuid.UUID `json:"id"`
    Title              string    `json:"title"`
    DescriptionPreview string    `json:"description_preview"` // truncated to ~200 chars
    PriceCents         int32     `json:"price_cents"`
    ContentType        string    `json:"content_type"`
    ThumbnailURL       *string   `json:"thumbnail_url,omitempty"`
    RatingAvg          float64   `json:"rating_avg"`
    RatingCount        int32     `json:"rating_count"`
    PublisherName      string    `json:"publisher_name"`
    CreatorStoreName   string    `json:"creator_store_name"`
}

type ListingDetailResponse struct {
    ID              uuid.UUID            `json:"id"`
    CreatorID       uuid.UUID            `json:"creator_id"`
    PublisherID     uuid.UUID            `json:"publisher_id"`
    PublisherName   string               `json:"publisher_name"`
    Title           string               `json:"title"`
    Description     string               `json:"description"`
    PriceCents      int32                `json:"price_cents"`
    MethodologyTags []uuid.UUID          `json:"methodology_tags"`
    SubjectTags     []string             `json:"subject_tags"`
    GradeMin        *int16               `json:"grade_min,omitempty"`
    GradeMax        *int16               `json:"grade_max,omitempty"`
    ContentType     string               `json:"content_type"`
    WorldviewTags   []string             `json:"worldview_tags"`
    PreviewURL      *string              `json:"preview_url,omitempty"`
    ThumbnailURL    *string              `json:"thumbnail_url,omitempty"`
    Status          string               `json:"status"`
    RatingAvg       float64              `json:"rating_avg"`
    RatingCount     int32                `json:"rating_count"`
    Version         int32                `json:"version"`
    Files           []ListingFileResponse `json:"files"`
    PublishedAt     *time.Time           `json:"published_at,omitempty"`
    CreatedAt       time.Time            `json:"created_at"`
    UpdatedAt       time.Time            `json:"updated_at"`
}

type ListingFileResponse struct {
    ID            uuid.UUID `json:"id"`
    FileName      string    `json:"file_name"`
    FileSizeBytes int64     `json:"file_size_bytes"`
    MimeType      string    `json:"mime_type"`
    Version       int32     `json:"version"`
}

type CartResponse struct {
    Items      []CartItemResponse `json:"items"`
    TotalCents int64              `json:"total_cents"`
    ItemCount  int32              `json:"item_count"`
}

type CartItemResponse struct {
    ListingID    uuid.UUID `json:"listing_id"`
    Title        string    `json:"title"`
    PriceCents   int32     `json:"price_cents"`
    ThumbnailURL *string   `json:"thumbnail_url,omitempty"`
    AddedAt      time.Time `json:"added_at"`
}

type PurchaseResponse struct {
    ID           uuid.UUID `json:"id"`
    ListingID    uuid.UUID `json:"listing_id"`
    ListingTitle string    `json:"listing_title"`
    AmountCents  int32     `json:"amount_cents"`
    Refunded     bool      `json:"refunded"`
    CreatedAt    time.Time `json:"created_at"`
}

type ReviewResponse struct {
    ID                uuid.UUID  `json:"id"`
    ListingID         uuid.UUID  `json:"listing_id"`
    Rating            int16      `json:"rating"`
    ReviewText        *string    `json:"review_text,omitempty"`
    IsAnonymous       bool       `json:"is_anonymous"`
    ReviewerName      *string    `json:"reviewer_name,omitempty"` // nil if anonymous
    CreatorResponse   *string    `json:"creator_response,omitempty"`
    CreatorResponseAt *time.Time `json:"creator_response_at,omitempty"`
    CreatedAt         time.Time  `json:"created_at"`
}

type DownloadResponse struct {
    DownloadURL string    `json:"download_url"`
    ExpiresAt   time.Time `json:"expires_at"`
}

type AutocompleteResult struct {
    ListingID  uuid.UUID `json:"listing_id"`
    Title      string    `json:"title"`
    Similarity float32   `json:"similarity"`
}

type CuratedSectionResponse struct {
    Slug        string                 `json:"slug"`
    DisplayName string                 `json:"display_name"`
    Description *string                `json:"description,omitempty"`
    Listings    []ListingBrowseResponse `json:"listings"`
}

type CreatorDashboardResponse struct {
    TotalSalesCount     int64         `json:"total_sales_count"`
    TotalEarningsCents  int64         `json:"total_earnings_cents"`
    PeriodSalesCount    int64         `json:"period_sales_count"`
    PeriodEarningsCents int64         `json:"period_earnings_cents"`
    PendingPayoutCents  int64         `json:"pending_payout_cents"`
    AverageRating       float64       `json:"average_rating"`
    TotalReviews        int32         `json:"total_reviews"`
    RecentSales         []SaleSummary `json:"recent_sales"`
}

type SaleSummary struct {
    PurchaseID        uuid.UUID `json:"purchase_id"`
    ListingTitle      string    `json:"listing_title"`
    AmountCents       int32     `json:"amount_cents"`
    CreatorPayoutCents int32    `json:"creator_payout_cents"`
    PurchasedAt       time.Time `json:"purchased_at"`
}

type ListingVersionResponse struct {
    Version       int32      `json:"version"`
    Title         string     `json:"title"`
    PriceCents    int32      `json:"price_cents"`
    ChangeSummary *string    `json:"change_summary,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
}
```

### §8.3 Internal Types

```go
// Query parameter types (not exposed in API responses)

type BrowseListingsParams struct {
    Q              *string      `query:"q"`
    MethodologyIDs []uuid.UUID  `query:"methodology_ids"`
    SubjectSlugs   []string     `query:"subject_slugs"`
    GradeMin       *int16       `query:"grade_min"`
    GradeMax       *int16       `query:"grade_max"`
    ContentType    *string      `query:"content_type"`
    WorldviewTags  []string     `query:"worldview_tags"`
    PriceMin       *int32       `query:"price_min"`
    PriceMax       *int32       `query:"price_max"`
    MinRating      *float64     `query:"min_rating"`
    SortBy         *string      `query:"sort_by"` // "relevance", "price_asc", "price_desc", "rating", "newest"
    Cursor         *string      `query:"cursor"`
    Limit          *uint8       `query:"limit"`
}

type ListingSortBy string

const (
    ListingSortByRelevance ListingSortBy = "relevance"
    ListingSortByPriceAsc  ListingSortBy = "price_asc"
    ListingSortByPriceDesc ListingSortBy = "price_desc"
    ListingSortByRating    ListingSortBy = "rating"
    ListingSortByNewest    ListingSortBy = "newest"
)

type CreatorListingQueryParams struct {
    Status *string `query:"status"`
    Cursor *string `query:"cursor"`
    Limit  *uint8  `query:"limit"`
}

type PurchaseQueryParams struct {
    Cursor *string `query:"cursor"`
    Limit  *uint8  `query:"limit"`
}

type ReviewQueryParams struct {
    SortBy *string `query:"sort_by"` // "newest", "oldest", "highest_rating", "lowest_rating"
    Cursor *string `query:"cursor"`
    Limit  *uint8  `query:"limit"`
}

type ReviewSortBy string

const (
    ReviewSortByNewest        ReviewSortBy = "newest"
    ReviewSortByOldest        ReviewSortBy = "oldest"
    ReviewSortByHighestRating ReviewSortBy = "highest_rating"
    ReviewSortByLowestRating  ReviewSortBy = "lowest_rating"
)

type DashboardPeriod string

const (
    DashboardPeriodLast7Days  DashboardPeriod = "last_7_days"
    DashboardPeriodLast30Days DashboardPeriod = "last_30_days"
    DashboardPeriodLast90Days DashboardPeriod = "last_90_days"
    DashboardPeriodAllTime    DashboardPeriod = "all_time"
)

// Internal row types returned by repositories
type ListingBrowseRow struct {
    ID               uuid.UUID
    Title            string
    Description      string
    PriceCents       int32
    ContentType      string
    ThumbnailURL     *string
    RatingAvg        float64
    RatingCount      int32
    PublisherName    string
    CreatorStoreName string
}

type AutocompleteRow struct {
    ListingID  uuid.UUID
    Title      string
    Similarity float32
}

type CartItemRow struct {
    ListingID    uuid.UUID
    Title        string
    PriceCents   int32
    ThumbnailURL *string
    CreatedAt    time.Time
}

type PurchaseRow struct {
    ID           uuid.UUID
    ListingID    uuid.UUID
    ListingTitle string
    AmountCents  int32
    RefundedAt   *time.Time
    CreatedAt    time.Time
}

type ReviewRow struct {
    ID                 uuid.UUID
    ListingID          uuid.UUID
    Rating             int16
    ReviewText         *string
    IsAnonymous        bool
    ReviewerFamilyName *string
    CreatorResponse    *string
    CreatorResponseAt  *time.Time
    ModerationStatus   string
    CreatedAt          time.Time
}

type SalesRow struct {
    PurchaseID        uuid.UUID
    ListingID         uuid.UUID
    ListingTitle      string
    AmountCents       int32
    CreatorPayoutCents int32
    CreatedAt         time.Time
}

type PublisherMemberRow struct {
    CreatorID uuid.UUID
    StoreName string
    Role      string
    CreatedAt time.Time
}
```

---

## §9 Listing Lifecycle State Machine (Domain Deep-Dive 1)

The `MarketplaceListing` aggregate root enforces the listing lifecycle state machine via private
fields and method-only transitions. Invalid state transitions are compile-time impossible (you
cannot set `status` directly) and runtime-checked (methods return `Result`). `[ARCH §4.5]`

### State Machine

```
                ┌──────────────────────────────┐
                │                              │
    ┌───────┐   │   ┌───────────┐   ┌──────────┴──┐   ┌──────────┐
    │ Draft │───┴──▶│ Submitted │──▶│  Published  │──▶│ Archived │
    └───────┘       └─────┬─────┘   └─────────────┘   └──────────┘
        ▲                 │
        │    rejection    │
        └─────────────────┘
```

### Transition Table

| From | To | Method | Preconditions | Side Effects | Event |
|------|----|--------|---------------|--------------|-------|
| Draft | Submitted | `Submit()` | At least 1 file attached, valid price, title/description present | — | `ListingSubmitted` |
| Submitted | Published | `Publish()` | Content screening passed | Sets `published_at` | `ListingPublished` |
| Submitted | Draft | `Reject()` | Content screening failed or creator withdraws | — | — |
| Published | Archived | `Archive()` | — | Sets `archived_at` | `ListingArchived` |
| Published | Published | `Update()` | — | Creates version snapshot, increments `version` | — |

### Invariants

1. A listing **must have at least one file** to be submitted. Enforced by checking file count
   in the `Submit()` method.
2. A listing **must have a valid price** (`price_cents >= 0`) — enforced by the `Price` value
   object constructor.
3. Published listings **always have a `published_at` timestamp** — set atomically during the
   `Publish()` transition.
4. Archived listings **retain `published_at`** — `archived_at` is additive, not a replacement.
5. Version snapshots are **only created for updates to published listings** — draft edits do
   not generate versions.

### Aggregate Root

```go
// internal/mkt/domain/listing.go

type ListingState string

const (
    ListingStateDraft     ListingState = "draft"
    ListingStateSubmitted ListingState = "submitted"
    ListingStatePublished ListingState = "published"
    ListingStateArchived  ListingState = "archived"
)

type MarketplaceListing struct {
    id              uuid.UUID
    creatorID       uuid.UUID
    publisherID     uuid.UUID
    title           string
    description     string
    priceCents      int32
    methodologyTags []uuid.UUID
    subjectTags     []string
    gradeMin        *int16
    gradeMax        *int16
    contentType     string
    worldviewTags   []string
    previewURL      *string
    thumbnailURL    *string
    state           ListingState
    ratingAvg       float64
    ratingCount     int32
    version         int32
    publishedAt     *time.Time
    archivedAt      *time.Time
    fileCount       int64 // denormalized for submit check
    createdAt       time.Time
    updatedAt       time.Time
}

// FromPersistence reconstructs from persistence (all fields provided by repository).
func FromPersistence( /* all fields */ ) *MarketplaceListing { /* ... */ }

// ─── Queries ────────────────────────────────────────────────────────

func (l *MarketplaceListing) ID() uuid.UUID          { return l.id }
func (l *MarketplaceListing) State() ListingState     { return l.state }
func (l *MarketplaceListing) CreatorID() uuid.UUID    { return l.creatorID }
func (l *MarketplaceListing) PublisherID() uuid.UUID  { return l.publisherID }
func (l *MarketplaceListing) Version() int32          { return l.version }

// ─── State Transitions ──────────────────────────────────────────────

func (l *MarketplaceListing) Submit() (*ListingSubmittedEvent, error) {
    if l.state != ListingStateDraft {
        return nil, &MktDomainError{
            Kind:   ErrInvalidStateTransition,
            From:   string(l.state),
            Action: "submit",
        }
    }
    if l.fileCount == 0 {
        return nil, &MktDomainError{Kind: ErrListingHasNoFiles}
    }
    l.state = ListingStateSubmitted
    return &ListingSubmittedEvent{
        ListingID: l.id,
        CreatorID: l.creatorID,
    }, nil
}

func (l *MarketplaceListing) Publish() (*ListingPublishedEvent, error) {
    if l.state != ListingStateSubmitted {
        return nil, &MktDomainError{
            Kind:   ErrInvalidStateTransition,
            From:   string(l.state),
            Action: "publish",
        }
    }
    l.state = ListingStatePublished
    now := time.Now().UTC()
    l.publishedAt = &now
    return &ListingPublishedEvent{
        ListingID:   l.id,
        PublisherID: l.publisherID,
        ContentType: l.contentType,
        SubjectTags: l.subjectTags,
    }, nil
}

func (l *MarketplaceListing) Reject() error {
    if l.state != ListingStateSubmitted {
        return &MktDomainError{
            Kind:   ErrInvalidStateTransition,
            From:   string(l.state),
            Action: "reject",
        }
    }
    l.state = ListingStateDraft
    return nil
}

func (l *MarketplaceListing) Archive() (*ListingArchivedEvent, error) {
    if l.state != ListingStatePublished {
        return nil, &MktDomainError{
            Kind:   ErrInvalidStateTransition,
            From:   string(l.state),
            Action: "archive",
        }
    }
    l.state = ListingStateArchived
    now := time.Now().UTC()
    l.archivedAt = &now
    return &ListingArchivedEvent{ListingID: l.id}, nil
}

// UpdatePublished updates a published listing. Returns the pre-update state for version snapshot.
func (l *MarketplaceListing) UpdatePublished(title, description *string, priceCents *int32) (*VersionSnapshot, error) {
    if l.state != ListingStatePublished && l.state != ListingStateDraft {
        return nil, &MktDomainError{
            Kind:   ErrInvalidStateTransition,
            From:   string(l.state),
            Action: "update",
        }
    }

    var snapshot *VersionSnapshot
    if l.state == ListingStatePublished {
        snapshot = &VersionSnapshot{
            Version:    l.version,
            Title:      l.title,
            Description: l.description,
            PriceCents: l.priceCents,
        }
        l.version++
    }

    if title != nil { l.title = *title }
    if description != nil { l.description = *description }
    if priceCents != nil { l.priceCents = *priceCents }
    l.updatedAt = time.Now().UTC()

    return snapshot, nil
}

type VersionSnapshot struct {
    Version     int32
    Title       string
    Description string
    PriceCents  int32
}
```

### Domain Errors

```go
// internal/mkt/domain/errors.go

type MktDomainErrorKind int

const (
    ErrInvalidStateTransition MktDomainErrorKind = iota
    ErrListingHasNoFiles
    ErrInvalidPrice
)

type MktDomainError struct {
    Kind   MktDomainErrorKind
    From   string // for InvalidStateTransition
    Action string // for InvalidStateTransition
}

func (e *MktDomainError) Error() string {
    switch e.Kind {
    case ErrInvalidStateTransition:
        return fmt.Sprintf("invalid state transition from %s via %s", e.From, e.Action)
    case ErrListingHasNoFiles:
        return "listing has no files attached"
    case ErrInvalidPrice:
        return "invalid price: must be >= 0"
    default:
        return "unknown domain error"
    }
}
```

---

## §10 Publisher Model & Content Bridge (Domain Deep-Dive 2)

### Publisher-as-Organization Pattern

Publishers are multi-user organizations (like GitHub orgs) that own published educational content.
This decouples content ownership from individual creator accounts, enabling:

1. **Team content management** — Multiple creators can manage listings under one publisher
2. **Platform content** — The built-in platform publisher (`is_platform = true`) owns
   platform-provided content definitions in `learn_*_defs` tables
3. **Cross-domain bridge** — All `learn_*_defs` tables reference `mkt_publishers(id)` as FK

### Built-in Platform Publisher

The platform publisher is seeded during database initialization (see §3.2 seed data). It:

- Has `is_platform = true` and `is_verified = true`
- Owns all platform-provided content definitions (activity templates, reading items, etc.)
- Cannot be deleted or modified through the API
- Is used by `learn::` when creating platform-provided content `[06-learn §3.2]`

### Membership Roles

| Role | Permissions |
|------|------------|
| **owner** | Full control: edit publisher, manage members, create/edit/publish listings |
| **admin** | Edit publisher details, create/edit/publish listings, view members |
| **member** | Create and edit own listings under the publisher |

### learn:: Cross-Domain Dependency

The `mkt_publishers` table is the **most critical cross-domain dependency** in the system. Five
`learn_*_defs` tables reference it:

- `learn_activity_defs.publisher_id → mkt_publishers(id)` `[06-learn:156]`
- `learn_assessment_defs.publisher_id → mkt_publishers(id)` `[06-learn:184]`
- `learn_reading_items.publisher_id → mkt_publishers(id)` `[06-learn:202]`
- `learn_project_defs.publisher_id → mkt_publishers(id)` `[06-learn:229]`
- `learn_video_defs.publisher_id → mkt_publishers(id)` `[06-learn:245]`

**Migration ordering**: The `mkt_publishers` table and its seed data (platform publisher) MUST
be created in a migration that precedes all `learn_*_defs` table migrations. The migration
filename ordering must reflect this dependency.

### VerifyPublisherMembership()

The `MarketplaceService.VerifyPublisherMembership()` method is consumed by `learn::` to
verify that a creator has permission to create/edit content definitions under a publisher.
`[06-learn §18.2]`

```go
// Called by learn:: service when creating/updating content definitions
isMember, err := marketplaceService.VerifyPublisherMembership(ctx, publisherID, creatorID)
if err != nil {
    return err
}
if !isMember {
    return apperr.Forbidden
}
```

---

## §11 Purchase Flow & Checkout (Domain Deep-Dive 3)

### Sequence Diagram

```
Family (Parent)           mkt:: Service            PaymentAdapter           Hyperswitch
     │                         │                        │                       │
     │  POST /cart/items       │                        │                       │
     │────────────────────────▶│                        │                       │
     │  (add items to cart)    │                        │                       │
     │◀────────────────────────│                        │                       │
     │                         │                        │                       │
     │  POST /cart/checkout    │                        │                       │
     │────────────────────────▶│                        │                       │
     │                         │  create_payment()      │                       │
     │                         │───────────────────────▶│                       │
     │                         │  (line_items +         │  POST /payments       │
     │                         │   split_rules)         │──────────────────────▶│
     │                         │                        │                       │
     │                         │                        │◀──────────────────────│
     │                         │◀───────────────────────│  PaymentSession       │
     │                         │  { checkout_url,       │  { checkout_url,      │
     │  { checkout_url }       │    payment_session_id }│    payment_id }       │
     │◀────────────────────────│                        │                       │
     │                         │                        │                       │
     │  (redirect to checkout) │                        │                       │
     │─────────────────────────────────────────────────────────────────────────▶│
     │                         │                        │                       │
     │  (payment completes)    │                        │                       │
     │                         │                        │  webhook: payment_    │
     │                         │                        │  succeeded            │
     │                         │  POST /webhooks/       │◀──────────────────────│
     │                         │  payments              │                       │
     │                         │◀───────────────────────│                       │
     │                         │                        │                       │
     │                         │  verify_webhook()      │                       │
     │                         │  parse_event()         │                       │
     │                         │                        │                       │
     │                         │  For each cart item:   │                       │
     │                         │  - Create mkt_purchase │                       │
     │                         │  - Publish             │                       │
     │                         │    PurchaseCompleted   │                       │
     │                         │  Clear cart items      │                       │
     │                         │                        │                       │
```

### Revenue Split Calculation

Revenue split is configurable at the platform level (stored as system config, default 75% to
creator) and recorded immutably on each purchase record at transaction time. `[S§9.6]`

```go
// Split calculation in checkout service
func calculateSplit(listingPriceCents int64, creatorSharePercent uint8) (creatorPayout, platformFee int64) {
    creatorPayout = (listingPriceCents * int64(creatorSharePercent)) / 100
    platformFee = listingPriceCents - creatorPayout
    return creatorPayout, platformFee
}

// Split rules sent to Hyperswitch
splitRules := make([]SplitRule, 0, len(cartItems))
for _, item := range cartItems {
    creatorPayout, _ := calculateSplit(
        int64(item.PriceCents),
        config.CreatorSharePercent, // e.g., 75
    )
    splitRules = append(splitRules, SplitRule{
        RecipientAccountID: item.CreatorPaymentAccountID,
        AmountCents:        creatorPayout,
    })
}
```

### Free Content "Get" Flow

Free content (`price_cents == 0`) bypasses the payment processor entirely. `[S§9.4]`

```go
// POST /v1/marketplace/listings/:listing_id/get
func (s *MarketplaceServiceImpl) GetFreeListing(ctx context.Context, listingID uuid.UUID, scope FamilyScope) (uuid.UUID, error) {
    listing, err := s.listings.GetByID(ctx, listingID)
    if err != nil {
        return uuid.Nil, err
    }
    if listing == nil {
        return uuid.Nil, ErrListingNotFound
    }

    if listing.PriceCents != 0 {
        return uuid.Nil, ErrListingNotFree
    }
    if listing.Status != "published" {
        return uuid.Nil, ErrListingNotPublished
    }

    // Check not already purchased
    existing, err := s.purchases.GetByFamilyAndListing(ctx, scope.FamilyID(), listingID)
    if err != nil {
        return uuid.Nil, err
    }
    if existing != nil {
        return uuid.Nil, ErrAlreadyPurchased
    }

    // Direct insert — no payment processor
    purchase, err := s.purchases.Create(ctx, CreatePurchase{
        FamilyID:          scope.FamilyID(),
        ListingID:         listingID,
        CreatorID:         listing.CreatorID,
        PaymentID:         nil,
        PaymentSessionID:  nil,
        AmountCents:       0,
        PlatformFeeCents:  0,
        CreatorPayoutCents: 0,
    })
    if err != nil {
        return uuid.Nil, err
    }

    s.events.Publish(PurchaseCompleted{
        FamilyID:   scope.FamilyID(),
        PurchaseID: purchase.ID,
        ListingID:  listingID,
        ContentMetadata: PurchaseMetadata{
            ContentType: listing.ContentType,
            ContentIDs:  []uuid.UUID{listingID},
            PublisherID: listing.PublisherID,
        },
    })

    return purchase.ID, nil
}
```

### Idempotency

Payment webhook processing is idempotent via `payment_session_id`:

```go
// In webhook handler
existing, err := s.purchases.GetByPaymentSessionID(ctx, paymentID)
if err != nil {
    return err
}
if existing != nil {
    // Already processed — return 200 OK to acknowledge
    return nil
}
```

### Refund Flow

Refunds are processed via Hyperswitch within a 30-day window. `[S§9.6]`

1. Family requests refund (Phase 2 endpoint, or admin-initiated)
2. Service calls `PaymentAdapter.CreateRefund(paymentID, amountCents, reason)`
3. On `RefundSucceeded` webhook: update `mkt_purchases.refunded_at`, `refund_amount_cents`, `refund_id`
4. Publish `PurchaseRefunded` event
5. Refund deducted from creator's pending earnings

---

## §12 File Upload & Signed-URL Delivery (Domain Deep-Dive 4)

### Creator File Upload

Creators upload files to listings via `media::` presigned URLs. `[ARCH §8.3, CODING §5.2]`

1. Creator calls `POST /v1/marketplace/listings/:id/files` with file metadata
2. Service validates: listing exists, creator owns it, listing is in draft/published status
3. Service calls `MediaAdapter.PresignedUpload()` to get a presigned R2 upload URL
4. Service creates `mkt_listing_files` record with `storage_key`
5. Returns presigned URL to client; client uploads directly to R2
6. **File validation**: MIME type validated via magic bytes on the server side when the
   upload completes (webhook or polling). `[CODING §5.2]`

### Purchaser Download

Purchasers download files via time-limited R2 signed URLs. `[ARCH §8.3]`

```go
// GET /v1/marketplace/purchases/:listing_id/download/:file_id
func (s *MarketplaceServiceImpl) GetDownloadURL(ctx context.Context, listingID, fileID uuid.UUID, scope FamilyScope) (*DownloadResponse, error) {
    // Verify purchase exists — no subscription tier check [S§9.4]
    purchase, err := s.purchases.GetByFamilyAndListing(ctx, scope.FamilyID(), listingID)
    if err != nil {
        return nil, err
    }
    if purchase == nil {
        return nil, ErrNotPurchased
    }

    file, err := s.listingFiles.GetByID(ctx, listingID, fileID)
    if err != nil {
        return nil, err
    }
    if file == nil {
        return nil, ErrFileNotFound
    }

    // Generate 1-hour signed URL
    signedURL, err := s.media.PresignedGet(ctx, file.StorageKey, 3600)
    if err != nil {
        return nil, err
    }

    expiresAt := time.Now().UTC().Add(1 * time.Hour)

    return &DownloadResponse{
        DownloadURL: signedURL,
        ExpiresAt:   expiresAt,
    }, nil
}
```

### Interactive Content Access `[S§9.7, S§18.7]`

Purchased interactive content grants the family access beyond file downloads:

| Content Type | Access Granted |
|-------------|---------------|
| `interactive_quiz` | Family can create `learn_quiz_sessions` against the quiz definition |
| `lesson_sequence` | Family can create `learn_sequence_progress` against the sequence definition |
| `video` | Family can stream video via signed HLS URLs; progress tracked in `learn_video_progress` |

Access is granted by the `PurchaseCompleted` event handler in `learn::`. The learning domain
checks for active purchases when a student attempts to start a quiz session, begin a sequence,
or stream a video. No separate "license" table — purchase records in `mkt_purchases` serve
as the access grant.

**Access check flow:**
1. Student (or parent) requests to start quiz/sequence/video
2. `learn::` service checks `mkt::PurchaseRepository.GetByFamilyAndListing(familyID, listingID)`
3. If purchase exists, access is granted; if not, return `content_not_purchased` (403)

### File Versioning

When a creator updates files on a published listing: `[S§9.2.3]`

1. New file upload creates a new `mkt_listing_files` row with incremented `version`
2. Old file row is retained (for audit trail) but the latest version is served to purchasers
3. Existing purchasers automatically get the latest version on next download
4. Price changes on the listing do NOT affect existing purchase records

---

## §13 Faceted Search & Discovery (Domain Deep-Dive 5)

### PostgreSQL Full-Text Search

Marketplace search uses PostgreSQL FTS with weighted ranking, matching the pattern
established in ARCHITECTURE.md §9.1. `[S§9.3, ARCH §9.1]`

```sql
-- Browse query with faceted filtering [S§9.3]
SELECT
    l.id, l.title, l.description, l.price_cents,
    l.content_type, l.thumbnail_url,
    l.rating_avg, l.rating_count,
    p.name as publisher_name,
    c.store_name as creator_store_name,
    ts_rank(l.search_vector, websearch_to_tsquery('english', $1)) as relevance
FROM mkt_listings l
JOIN mkt_publishers p ON p.id = l.publisher_id
JOIN mkt_creators c ON c.id = l.creator_id
WHERE l.status = 'published'
  -- Full-text search (optional — omitted if no query string)
  AND ($1::text IS NULL OR l.search_vector @@ websearch_to_tsquery('english', $1))
  -- Faceted filters [S§9.3]
  AND ($2::uuid[] IS NULL OR l.methodology_tags && $2)    -- methodology filter
  AND ($3::text[] IS NULL OR l.subject_tags && $3)         -- subject filter
  AND ($4::smallint IS NULL OR l.grade_min <= $4)          -- grade range
  AND ($5::smallint IS NULL OR l.grade_max >= $5)
  AND ($6::int IS NULL OR l.price_cents >= $6)             -- price range
  AND ($7::int IS NULL OR l.price_cents <= $7)
  AND ($8::text IS NULL OR l.content_type = $8)            -- content type
  AND ($9::text[] IS NULL OR l.worldview_tags && $9)       -- worldview filter
  AND ($10::numeric IS NULL OR l.rating_avg >= $10)        -- minimum rating
ORDER BY
    CASE WHEN $11 = 'relevance' THEN ts_rank(l.search_vector, websearch_to_tsquery('english', $1)) END DESC,
    CASE WHEN $11 = 'price_asc' THEN l.price_cents END ASC,
    CASE WHEN $11 = 'price_desc' THEN l.price_cents END DESC,
    CASE WHEN $11 = 'rating' THEN l.rating_avg END DESC,
    CASE WHEN $11 = 'newest' THEN l.published_at END DESC
LIMIT $12 OFFSET $13;
```

### Trigram Autocomplete

```sql
-- Autocomplete query [ARCH §9.1]
SELECT DISTINCT l.id, l.title, similarity(l.title, $1) as sim
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.title % $1    -- trigram similarity match
ORDER BY sim DESC
LIMIT $2;
```

### Curated Sections

| Section | Type | Population Strategy |
|---------|------|-------------------|
| **Featured** | `manual` | Admin-curated via `mkt_curated_section_items` |
| **Staff Picks** | `manual` | Admin-curated |
| **Trending** | `auto` | Background job: top listings by purchase count in last 7 days |
| **New Arrivals** | `auto` | Background job: most recently published listings |

Auto-computed sections are refreshed by a periodic background job (e.g., every hour) that
replaces `mkt_curated_section_items` rows for `auto` sections.

### Meilisearch Migration Trigger

When PostgreSQL FTS becomes insufficient (>50k listings, complex facet combinations causing
slow queries, or p95 search latency exceeding 500ms), migrate to Meilisearch. The contract:

- `ListingRepository.Browse()` switches from SQL to Meilisearch HTTP client
- `ListingPublished` / `ListingArchived` events trigger index updates
- The search API response shape (`ListingBrowseResponse`) remains unchanged

---

## §14 Review System & Moderation (Domain Deep-Dive 6)

### Verified-Purchaser Enforcement

Only families who have purchased a listing can leave a review. Enforced via `purchase_id`
foreign key and a service-level check. `[S§9.5]`

```go
func (s *MarketplaceServiceImpl) CreateReview(ctx context.Context, cmd CreateReviewCommand, listingID uuid.UUID, scope FamilyScope) (uuid.UUID, error) {
    // Verify purchase exists
    purchase, err := s.purchases.GetByFamilyAndListing(ctx, scope.FamilyID(), listingID)
    if err != nil {
        return uuid.Nil, err
    }
    if purchase == nil {
        return uuid.Nil, ErrNotPurchased
    }

    // One review per purchase
    // (enforced by UNIQUE constraint on purchase_id, but check first for better error)

    isAnonymous := true
    if cmd.IsAnonymous != nil {
        isAnonymous = *cmd.IsAnonymous
    }

    review, err := s.reviews.Create(ctx, CreateReview{
        ListingID:   listingID,
        PurchaseID:  purchase.ID,
        FamilyID:    scope.FamilyID(),
        Rating:      cmd.Rating,
        ReviewText:  cmd.ReviewText,
        IsAnonymous: isAnonymous,
    })
    if err != nil {
        return uuid.Nil, err
    }

    // Update aggregate rating on listing
    _, _, err = s.reviews.GetAggregateRating(ctx, listingID)
    if err != nil {
        return uuid.Nil, err
    }
    // (listing rating_avg and rating_count updated in repository)

    // Publish event
    s.events.Publish(ReviewCreated{
        ReviewID:  review.ID,
        ListingID: listingID,
        Rating:    cmd.Rating,
    })

    // Send review text to safety:: for moderation
    if cmd.ReviewText != nil {
        s.events.Publish(ContentSubmittedForModeration{
            ContentID:   review.ID,
            ContentType: "marketplace_review",
            Text:        *cmd.ReviewText,
        })
    }

    return review.ID, nil
}
```

### Anonymous by Default

Reviews are anonymous by default (`is_anonymous = true`). When `is_anonymous = false`, the
family display name is shown. The reviewer can toggle anonymity on update. `[S§9.5]`

### Aggregate Rating Calculation

Rating average and count are denormalized on `mkt_listings` for query performance.
Recalculated on review create/update/delete:

```sql
-- Recalculate aggregate rating
UPDATE mkt_listings
SET rating_avg = COALESCE((
    SELECT AVG(rating)::NUMERIC(3,2)
    FROM mkt_reviews
    WHERE listing_id = $1
      AND moderation_status = 'approved'
), 0),
rating_count = (
    SELECT COUNT(*)
    FROM mkt_reviews
    WHERE listing_id = $1
      AND moderation_status = 'approved'
)
WHERE id = $1;
```

### Creator Response Flow

Creators can respond to reviews (one response per review). `[S§9.5]`

1. Creator calls `POST /v1/marketplace/listings/:id/reviews/:id/respond`
2. Service verifies creator owns the listing
3. Sets `creator_response` and `creator_response_at`
4. Response is visible to all users viewing the review

### safety:: Integration

Review text is sent to `safety::` for content moderation via a domain event. Reviews start
with `moderation_status = 'pending'`. The safety domain processes the text and calls back:

- **Approved**: `moderation_status` set to `'approved'` — review visible and included in
  aggregate rating calculation
- **Rejected**: `moderation_status` set to `'rejected'` — review hidden from public view

---

## §15 Creator Dashboard & Revenue (Domain Deep-Dive 7)

### Dashboard Data

The creator dashboard aggregates sales, earnings, and analytics data. `[S§9.6]`

```go
func (s *MarketplaceServiceImpl) GetCreatorDashboard(ctx context.Context, creatorID uuid.UUID, period DashboardPeriod) (*CreatorDashboardResponse, error) {
    from, to := period.ToDateRange()

    allTimeSales, err := s.purchases.GetCreatorSales(ctx, creatorID, time.Time{}, time.Now().UTC())
    if err != nil {
        return nil, err
    }

    periodSales, err := s.purchases.GetCreatorSales(ctx, creatorID, from, to)
    if err != nil {
        return nil, err
    }

    var totalEarnings int64
    for _, sale := range allTimeSales {
        totalEarnings += int64(sale.CreatorPayoutCents)
    }
    var periodEarnings int64
    for _, sale := range periodSales {
        periodEarnings += int64(sale.CreatorPayoutCents)
    }

    recentSales := make([]SaleSummary, 0, 10)
    for i, sale := range periodSales {
        if i >= 10 { break }
        recentSales = append(recentSales, SaleSummary{
            PurchaseID:        sale.PurchaseID,
            ListingTitle:      sale.ListingTitle,
            AmountCents:       sale.AmountCents,
            CreatorPayoutCents: sale.CreatorPayoutCents,
            PurchasedAt:       sale.CreatedAt,
        })
    }

    return &CreatorDashboardResponse{
        TotalSalesCount:     int64(len(allTimeSales)),
        TotalEarningsCents:  totalEarnings,
        PeriodSalesCount:    int64(len(periodSales)),
        PeriodEarningsCents: periodEarnings,
        PendingPayoutCents:  0, // Phase 2: calculated from unpaid earnings
        AverageRating:       0, // aggregate from listings
        TotalReviews:        0, // aggregate from listings
        RecentSales:         recentSales,
    }, nil
}
```

### Revenue Share Model

| Parameter | Value | Notes |
|-----------|-------|-------|
| Default creator share | 75% | Stored as system config, not hardcoded `[S§9.6]` |
| Platform fee | 25% | Remainder after creator share |
| Split enforcement | At payment time | Via Hyperswitch split payment rules |
| Split immutability | Per-purchase | `mkt_purchases` records the split at creation time |

### Payout Tracking (Phase 2)

Creator payouts are managed via Hyperswitch's payout API:

1. Creator requests payout from dashboard
2. Service validates: `onboarding_status = 'active'`, minimum threshold met ($25 default)
3. Service calls `PaymentAdapter.CreatePayout()`
4. Hyperswitch routes payout to creator's connected bank account via underlying processor
5. `PayoutCompleted` webhook updates payout status

### 1099-K Compliance (Phase 2)

Tax reporting is offloaded to the underlying payment processor (Stripe) via Hyperswitch.
Stripe automatically generates and files 1099-K forms for creators meeting IRS thresholds.
The platform does not handle tax forms directly. `[S§9.6]`

---

## §16 RequireCreator Middleware (Domain Deep-Dive 8)

### Decision

**Dedicated middleware in `internal/mkt/middleware.go`** — does NOT add `creator_id` to `AuthContext`.

This resolves the open question from 00-core §13.3, which documented two approaches:

1. ~~Preferred (00-core): Add `CreatorID *uuid.UUID` to `AuthContext`~~ — **rejected**
2. **Chosen: Query `mkt::` service from the middleware via `AppState`**

### Rationale

1. **Shared kernel minimality** — `AuthContext` is in `internal/shared/types.go` and used by ALL
   domains. Adding `creator_id` (a marketplace-specific concept) to the shared kernel violates
   the bounded context principle. Only ~10% of requests are creator-related.

2. **No unnecessary DB overhead** — Adding `creator_id` to `AuthContext` means the auth
   middleware would need to LEFT JOIN `mkt_creators` on EVERY authenticated request, even
   though most requests don't need it.

3. **Redis caching eliminates performance concern** — The middleware queries `mkt_creators` by
   `parent_id`, which is cached in Redis with a short TTL (5 minutes). The cache hit rate for
   active creators will be near 100%.

4. **Domain ownership** — The creator lookup logic belongs in `mkt::`, not in the shared auth
   middleware. This keeps the auth middleware simple and domain-agnostic.

### Implementation

```go
// internal/mkt/middleware.go

// CreatorContext holds the authenticated user's creator information.
// Extracted by RequireCreator middleware and stored in Echo context.
//
// Uses Redis caching to avoid DB query on every request.
// Cache key: "mkt:creator:{parent_id}" with 5-minute TTL.
type CreatorContext struct {
    Auth      *AuthContext
    CreatorID uuid.UUID
}

// RequireCreator is an Echo middleware that verifies the user has a creator account.
// Returns 403 Forbidden if no creator account exists. [S§3.1.4]
func RequireCreator(appState *AppState) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            auth, ok := c.Get("auth").(*AuthContext)
            if !ok || auth == nil {
                return apperr.Unauthorized
            }

            cacheKey := fmt.Sprintf("mkt:creator:%s", auth.ParentID)

            // Try Redis cache first
            creatorIDStr, err := appState.Redis.Get(c.Request().Context(), cacheKey).Result()
            if err == nil {
                creatorID, _ := uuid.Parse(creatorIDStr)
                c.Set("creator", &CreatorContext{Auth: auth, CreatorID: creatorID})
                return next(c)
            }

            // Cache miss — query DB
            creator, err := appState.MarketplaceService.GetCreatorByParentID(c.Request().Context(), auth.ParentID)
            if err != nil {
                return err
            }
            if creator == nil {
                return apperr.Forbidden
            }

            // Cache for 5 minutes
            appState.Redis.Set(c.Request().Context(), cacheKey, creator.ID.String(), 5*time.Minute)

            c.Set("creator", &CreatorContext{Auth: auth, CreatorID: creator.ID})
            return next(c)
        }
    }
}
```

### Usage in Handlers

```go
// internal/mkt/handler.go

func (h *Handler) CreateListing(c echo.Context) error {
    cc := c.Get("creator").(*CreatorContext)

    var cmd CreateListingCommand
    if err := c.Bind(&cmd); err != nil {
        return err
    }

    listingID, err := h.service.CreateListing(c.Request().Context(), cmd, cc.CreatorID)
    if err != nil {
        return err
    }

    listing, err := h.service.GetListing(c.Request().Context(), listingID)
    if err != nil {
        return err
    }

    return c.JSON(http.StatusCreated, listing)
}

func (h *Handler) GetCreatorProfile(c echo.Context) error {
    cc := c.Get("creator").(*CreatorContext)

    creator, err := h.service.GetCreatorByParentID(c.Request().Context(), cc.Auth.ParentID)
    if err != nil {
        return err
    }
    if creator == nil {
        return apperr.NotFound
    }

    return c.JSON(http.StatusOK, creator)
}
```

---

## §17 Error Types

All marketplace errors use custom error types with `errors.Is`/`errors.As` and map to HTTP
status codes via `AppError`. Internal details are logged but never exposed in API
responses. `[CODING §5.2, S§18]`

```go
// internal/mkt/errors.go

import "errors"

// ─── Creator Errors ─────────────────────────────────────────────────
var (
    ErrCreatorAlreadyExists = errors.New("creator account already exists for this user")
    ErrCreatorNotFound      = errors.New("creator account not found")
    ErrTOSNotAccepted       = errors.New("creator must accept Terms of Service")
    ErrCreatorNotActive     = errors.New("creator onboarding not complete")
    ErrCreatorSuspended     = errors.New("creator account is suspended")
)

// ─── Publisher Errors ───────────────────────────────────────────────
var (
    ErrPublisherNotFound           = errors.New("publisher not found")
    ErrPublisherSlugConflict       = errors.New("publisher slug already taken")
    ErrNotPublisherMember          = errors.New("not a member of this publisher")
    ErrInsufficientPublisherRole   = errors.New("insufficient publisher role for this action")
    ErrCannotRemoveLastOwner       = errors.New("cannot remove the last owner of a publisher")
    ErrCannotModifyPlatformPublisher = errors.New("cannot modify the platform publisher")
)

// ─── Listing Errors ─────────────────────────────────────────────────
var (
    ErrListingNotFound    = errors.New("listing not found")
    ErrListingNotPublished = errors.New("listing is not published")
    ErrListingNotFree     = errors.New("listing is not free")
    ErrListingHasNoFiles  = errors.New("listing has no files attached")
    ErrNotListingOwner    = errors.New("not the owner of this listing")
    ErrInvalidContentType = errors.New("invalid content type")
    ErrInvalidPrice       = errors.New("invalid price: must be >= 0")
)

// ─── File Errors ────────────────────────────────────────────────────
var (
    ErrFileNotFound   = errors.New("file not found")
    ErrInvalidFileType = errors.New("invalid file type")
    ErrFileTooLarge   = errors.New("file too large")
)

// ─── Cart Errors ────────────────────────────────────────────────────
var (
    ErrAlreadyInCart = errors.New("item already in cart")
    ErrNotInCart     = errors.New("item not in cart")
    ErrEmptyCart     = errors.New("cart is empty")
    ErrStaleCart     = errors.New("cart contains unpublished listings")
)

// ─── Purchase Errors ────────────────────────────────────────────────
var (
    ErrAlreadyPurchased   = errors.New("already purchased this listing")
    ErrPurchaseNotFound   = errors.New("purchase not found")
    ErrNotPurchased       = errors.New("not purchased — cannot download")
    ErrRefundWindowExpired = errors.New("refund window has expired (30 days)")
    ErrAlreadyRefunded    = errors.New("purchase already refunded")
)

// ─── Review Errors ──────────────────────────────────────────────────
var (
    ErrAlreadyReviewed = errors.New("already reviewed this purchase")
    ErrReviewNotFound  = errors.New("review not found")
    ErrNotReviewOwner  = errors.New("not the owner of this review")
    ErrInvalidRating   = errors.New("invalid rating: must be between 1 and 5")
)

// ─── Payment Errors (processor-agnostic) ────────────────────────────
var (
    ErrPaymentProviderUnavailable = errors.New("payment provider unavailable")
    ErrPaymentCreationFailed      = errors.New("payment creation failed")
    ErrInvalidWebhookSignature    = errors.New("invalid webhook signature")
    ErrMalformedWebhookPayload    = errors.New("webhook payload malformed")
    ErrPayoutThresholdNotMet      = errors.New("payout threshold not met")
)
```

### Error-to-HTTP Mapping

| Error Variant | HTTP Status | Response Code |
|--------------|-------------|---------------|
| `CreatorAlreadyExists` | `409 Conflict` | `creator_already_exists` |
| `CreatorNotFound` | `404 Not Found` | `creator_not_found` |
| `TosNotAccepted` | `422 Unprocessable` | `tos_not_accepted` |
| `CreatorNotActive` | `403 Forbidden` | `creator_not_active` |
| `CreatorSuspended` | `403 Forbidden` | `creator_suspended` |
| `PublisherNotFound` | `404 Not Found` | `publisher_not_found` |
| `PublisherSlugConflict` | `409 Conflict` | `publisher_slug_conflict` |
| `NotPublisherMember` | `403 Forbidden` | `not_publisher_member` |
| `InsufficientPublisherRole` | `403 Forbidden` | `insufficient_publisher_role` |
| `CannotRemoveLastOwner` | `400 Bad Request` | `cannot_remove_last_owner` |
| `CannotModifyPlatformPublisher` | `403 Forbidden` | `cannot_modify_platform_publisher` |
| `ListingNotFound` | `404 Not Found` | `listing_not_found` |
| `ListingNotPublished` | `400 Bad Request` | `listing_not_published` |
| `ListingNotFree` | `400 Bad Request` | `listing_not_free` |
| `InvalidStateTransition` | `409 Conflict` | `invalid_state_transition` |
| `ListingHasNoFiles` | `400 Bad Request` | `listing_has_no_files` |
| `NotListingOwner` | `403 Forbidden` | `not_listing_owner` |
| `InvalidContentType` | `422 Unprocessable` | `invalid_content_type` |
| `InvalidPrice` | `422 Unprocessable` | `invalid_price` |
| `FileNotFound` | `404 Not Found` | `file_not_found` |
| `InvalidFileType` | `422 Unprocessable` | `invalid_file_type` |
| `FileTooLarge` | `413 Payload Too Large` | `file_too_large` |
| `AlreadyInCart` | `409 Conflict` | `already_in_cart` |
| `NotInCart` | `404 Not Found` | `not_in_cart` |
| `EmptyCart` | `400 Bad Request` | `empty_cart` |
| `StaleCart` | `409 Conflict` | `stale_cart` |
| `AlreadyPurchased` | `409 Conflict` | `already_purchased` |
| `PurchaseNotFound` | `404 Not Found` | `purchase_not_found` |
| `NotPurchased` | `403 Forbidden` | `not_purchased` |
| `RefundWindowExpired` | `400 Bad Request` | `refund_window_expired` |
| `AlreadyRefunded` | `409 Conflict` | `already_refunded` |
| `AlreadyReviewed` | `409 Conflict` | `already_reviewed` |
| `ReviewNotFound` | `404 Not Found` | `review_not_found` |
| `NotReviewOwner` | `403 Forbidden` | `not_review_owner` |
| `InvalidRating` | `422 Unprocessable` | `invalid_rating` |
| `PaymentProviderUnavailable` | `502 Bad Gateway` | `payment_provider_unavailable` |
| `PaymentCreationFailed` | `502 Bad Gateway` | `payment_creation_failed` |
| `InvalidWebhookSignature` | `401 Unauthorized` | `invalid_webhook_signature` |
| `MalformedWebhookPayload` | `400 Bad Request` | `malformed_webhook_payload` |
| `PayoutThresholdNotMet` | `400 Bad Request` | `payout_threshold_not_met` |
| `DatabaseError` | `500 Internal` | `internal_error` (no details exposed) |
| `CacheError` | `500 Internal` | `internal_error` (no details exposed) |

---

## §17b Creator Authoring Tools `[S§9.1, V§9]`

Creators build interactive content using platform-provided authoring tools. These tools
create content in the `learn::` domain via `learn::` APIs, but the authoring workflow
is owned by `mkt::` as part of the creator content management experience.

### Quiz Builder

The quiz builder is a creator-facing UI for building questions and assembling quizzes:

1. **Question creation**: Creator writes question text, defines answer data by type
   (multiple-choice options, fill-in-the-blank answers, matching pairs, etc.), adds
   optional media attachments, and tags with subject/methodology
2. **Quiz assembly**: Creator selects questions from their publisher's question bank,
   sets quiz metadata (title, passing score, time limit, shuffle settings), and orders questions
3. **Preview**: Creator can preview how the quiz appears to students
4. **Publication**: Quiz follows same Draft → Submitted → Published lifecycle as other listings

**API flow**: Quiz builder frontend calls `learn::` endpoints (`POST /v1/learning/questions`,
`POST /v1/learning/quizzes`) with the creator's publisher credentials. The listing is
created in `mkt::` with `content_type = 'interactive_quiz'` and references the
`learn_quiz_defs.id` via the listing's content metadata.

### Sequence Builder

The sequence builder is a creator-facing UI for ordering content into lesson paths:

1. **Content selection**: Creator browses their publisher's content definitions
   (activity defs, reading items, video defs, quiz defs) and adds them to the sequence
2. **Ordering**: Creator arranges items in order, sets required/optional flags and
   unlock-after-previous settings
3. **Mode selection**: Creator chooses linear (strict order) or recommended-order mode
4. **Publication**: Sequence follows same listing lifecycle

**API flow**: Sequence builder frontend calls `learn::` endpoints
(`POST /v1/learning/sequences`) with publisher credentials. The listing is created
in `mkt::` with `content_type = 'lesson_sequence'`.

### Content Moderation

All creator-authored interactive content passes through `safety::` content moderation:
- Question text and answer text are screened via text scanning
- Media attachments follow standard image/video moderation pipeline
- Content that fails moderation is blocked from publication
- Screening runs on the `ListingPublished` event for marketplace-distributed content

### Publisher Requirement

Authoring tools require publisher membership — creators must belong to at least one
`mkt_publishers` organization. This is consistent with all other content creation
flows (`learn::` Layer 1 definitions require `publisher_id`).

---

## §18 Cross-Domain Interactions

### §18.1 mkt:: Provides (consumed by other domains)

| Export | Consumers | Mechanism |
|--------|-----------|-----------|
| `MarketplaceService` interface methods | `learn`, `onboard` (Phase 2) | `MarketplaceService` interface via AppState |
| `VerifyPublisherMembership()` | `learn::` | Service method — publisher ownership checks `[06-learn §18.2]` |
| `PurchaseCompleted` event | `learn::`, `billing::`, `notify::` | Domain event — tool access grant, creator earnings, receipt email `[ARCH §4.6]` |
| `ListingPublished` event | `search::`, `ai::` | Domain event — search index update, recommendation catalog |
| `ListingArchived` event | `search::` | Domain event — remove from search index |
| `ReviewCreated` event | `safety::` | Domain event — content moderation queue |
| `CreatorOnboarded` event | `notify::` | Domain event — welcome email to creator |
| `PurchaseRefunded` event | `billing::`, `notify::` | Domain event — earnings adjustment, refund notification |

### §18.2 mkt:: Consumes

| Dependency | Source | Purpose |
|-----------|--------|---------|
| `AuthContext` | `iam::` middleware | User identity on every request `[00-core §7.2]` |
| `FamilyScope` | `iam::` middleware | Family-scoped data access `[00-core §8]` |
| `MediaAdapter` | `media::` | Presigned upload/download URLs `[ARCH §8.3]` |
| `PaymentAdapter` | Hyperswitch (self-hosted) | Payment orchestration `[supersedes ADR-007]` |
| `ContentFlagged` event | `safety::` | Listing/review moderation actions |
| `FamilyDeletionScheduled` event | `iam::` | Prepare marketplace data for cascade deletion |
| `MethodologyConfigUpdated` event | `method::` | Invalidate cached methodology tag lookups |

### §18.3 Events mkt:: Publishes

Defined in `internal/mkt/events.go`. `[CODING §8.4]`

```go
// internal/mkt/events.go

// PurchaseCompleted is published when a family completes a marketplace purchase.
// Consumed by learn:: (tool access), billing:: (creator earnings), notify:: (receipt).
//
// IMPORTANT: The ContentMetadata field shape is a cross-domain contract
// consumed by learn::event_handlers.go [06-learn:3171]. Changes to this
// struct require coordinated updates to all consumers.
type PurchaseCompleted struct {
    FamilyID        uuid.UUID        `json:"family_id"`
    PurchaseID      uuid.UUID        `json:"purchase_id"`
    ListingID       uuid.UUID        `json:"listing_id"`
    ContentMetadata PurchaseMetadata `json:"content_metadata"`
}

// PurchaseMetadata holds metadata about purchased content.
// Defined authoritatively by mkt::events and imported by learn:: [06-learn:2532].
type PurchaseMetadata struct {
    ContentType string      `json:"content_type"`
    ContentIDs  []uuid.UUID `json:"content_ids"`
    PublisherID uuid.UUID   `json:"publisher_id"`
}

// ListingPublished is published when a listing transitions to Published state.
// Consumed by search:: (index update), ai:: (recommendation catalog).
type ListingPublished struct {
    ListingID   uuid.UUID `json:"listing_id"`
    PublisherID uuid.UUID `json:"publisher_id"`
    ContentType string    `json:"content_type"`
    SubjectTags []string  `json:"subject_tags"`
}

// ListingArchived is published when a listing is archived.
// Consumed by search:: (remove from index).
type ListingArchived struct {
    ListingID uuid.UUID `json:"listing_id"`
}

// ReviewCreated is published when a verified-purchaser review is created.
// Consumed by safety:: (content moderation scan).
type ReviewCreated struct {
    ReviewID   uuid.UUID `json:"review_id"`
    ListingID  uuid.UUID `json:"listing_id"`
    Rating     int16     `json:"rating"`
    ReviewText *string   `json:"review_text,omitempty"` // for safety:: text scanning [11-safety §11.2]
}

// CreatorOnboarded is published when a creator completes registration.
// Consumed by notify:: (welcome email).
type CreatorOnboarded struct {
    CreatorID uuid.UUID `json:"creator_id"`
    ParentID  uuid.UUID `json:"parent_id"`
    StoreName string    `json:"store_name"`
}

// PurchaseRefunded is published when a purchase is refunded.
// Consumed by billing:: (earnings adjustment), notify:: (refund notification).
type PurchaseRefunded struct {
    PurchaseID       uuid.UUID `json:"purchase_id"`
    ListingID        uuid.UUID `json:"listing_id"`
    FamilyID         uuid.UUID `json:"family_id"`
    RefundAmountCents int64    `json:"refund_amount_cents"`
}

// ListingSubmitted is published when a listing is submitted for content screening.
// Consumed by safety:: (automated content screening).
type ListingSubmitted struct {
    ListingID uuid.UUID `json:"listing_id"`
    CreatorID uuid.UUID `json:"creator_id"`
}
```

### §18.4 Events mkt:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `ContentFlagged { content_id, content_type, action }` | `safety::` | If `content_type == "listing"`: archive listing. If `"review"`: reject review. |
| `FamilyDeletionScheduled { family_id, delete_after }` | `iam::` | Anonymize reviews (retain ratings, clear text), retain purchase records (legal requirement), clear cart items. |
| `MethodologyConfigUpdated` | `method::` | Invalidate cached methodology tag display names used in listing browse responses. |

```go
// internal/mkt/event_handlers.go

type ContentFlaggedHandler struct {
    marketplaceService MarketplaceService
}

func (h *ContentFlaggedHandler) Handle(ctx context.Context, event *ContentFlagged) error {
    return h.marketplaceService.HandleContentFlagged(ctx, event.ContentID, event.Action)
}

type FamilyDeletionScheduledHandler struct {
    marketplaceService MarketplaceService
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event *FamilyDeletionScheduled) error {
    return h.marketplaceService.HandleFamilyDeletionScheduled(ctx, event.FamilyID)
}

type MethodologyConfigUpdatedHandler struct {
    // Invalidate Redis cache for methodology tag display names
    redis *redis.Client
}

func (h *MethodologyConfigUpdatedHandler) Handle(ctx context.Context, event *MethodologyConfigUpdated) error {
    return h.redis.Del(ctx, "mkt:methodology_tags:*").Err()
}
```

### §18.5 Infrastructure Note: Hyperswitch Deployment

Hyperswitch is self-hosted as a Docker sidecar alongside the main application. The deployment
topology:

```
┌─────────────────────────────────────────────────────────────┐
│  Docker Compose / Kubernetes Pod                            │
│                                                             │
│  ┌─────────────────┐     ┌──────────────────────────────┐  │
│  │  homegrown-app   │────▶│  hyperswitch-app (Rust)      │  │
│  │  (Echo server)   │     │  :8080                       │  │
│  └────────┬─────────┘     └──────────┬───────────────────┘  │
│           │                          │                       │
│  ┌────────▼─────────┐     ┌──────────▼───────────────────┐  │
│  │  PostgreSQL       │     │  hyperswitch-postgres         │  │
│  │  (app database)   │     │  (Hyperswitch database)       │  │
│  └──────────────────┘     └──────────────────────────────┘  │
│                                                             │
│  ┌──────────────────┐     ┌──────────────────────────────┐  │
│  │  Redis            │     │  hyperswitch-redis             │  │
│  │  (app cache)      │     │  (Hyperswitch cache)           │  │
│  └──────────────────┘     └──────────────────────────────┘  │
│                                                             │
│  External connections:                                      │
│  hyperswitch-app ──▶ Stripe API (initial connector)         │
│  hyperswitch-app ──▶ Adyen API (future connector)           │
└─────────────────────────────────────────────────────────────┘
```

**Shared instance, separate concerns**: Both `mkt::` (marketplace payments) and `billing::`
(subscription payments) communicate with the same Hyperswitch instance. They use different
Hyperswitch business profiles to isolate marketplace payment flows from subscription flows.
This means one Hyperswitch deployment handles all payment needs, but the two domains have
independent configuration and reporting.

---

## §19 Phase Scope

### Phase 1 — Foundation `[S§19]`

**In scope**:
- Creator registration and profile management (4 endpoints)
- Publisher creation and management (4 endpoints)
- Listing CRUD with state machine: Draft → Submitted → Published → Archived (5 endpoints)
- Listing browse with faceted search and FTS (4 endpoints)
- Cart and checkout with Hyperswitch payment integration (4 endpoints)
- Purchase records and file download via signed URLs (2 endpoints)
- Basic review creation (1 endpoint)
- Hyperswitch payment webhook receiver (1 endpoint)
- **~25 endpoints, 11 tables**

**Out of scope for Phase 1**: Review CRUD, free-get flow, creator dashboard, creator listings
view, payouts, publisher member management, content bundles, admin publish, version history
endpoint.

### Phase 2 — Depth & Revenue `[S§19]`

**In scope**:
- Review update, delete, creator response (3 endpoints)
- Free content "Get" flow (1 endpoint)
- Creator dashboard with sales/earnings data (1 endpoint)
- Creator own-listings view (1 endpoint)
- Creator payouts via Hyperswitch payout API (1 endpoint)
- Publisher member addition (1 endpoint)
- 1099-K compliance (offloaded to Stripe via Hyperswitch)
- View counting for listings
- **~8 additional endpoints**

### Phase 3 — Scale & Control `[S§19]`

**In scope**:
- Content bundles (1 endpoint)
- Admin-moderated publish flow (1 endpoint)
- Publisher member removal (1 endpoint)
- Listing version history endpoint (1 endpoint)
- Meilisearch migration (if FTS performance insufficient)
- Advanced analytics in creator dashboard
- **~4 additional endpoints**

---

## §20 Verification Checklist

Each item is a testable assertion. Implementation is not complete until all assertions pass.

### Creator Onboarding

1. `POST /v1/marketplace/creators/register` creates a creator with `onboarding_status = 'pending'`
2. Registering twice returns `409 Conflict`
3. `tos_accepted = false` returns `422`
4. `POST /v1/marketplace/creators/onboarding-link` calls `PaymentAdapter.CreateOnboardingLink()`
5. Creator profile update validates `store_name` length (1-100)
6. `CreatorOnboarded` event published on registration

### Publisher Management

7. `POST /v1/marketplace/publishers` creates publisher and `owner` membership for creator
8. Duplicate slug returns `409 Conflict`
9. Only `owner`/`admin` can update publisher details
10. `VerifyPublisherMembership()` returns true for members, false for non-members
11. Platform publisher (`is_platform = true`) cannot be modified via API
12. Platform publisher seed data exists after migration

### Listing Lifecycle

13. New listing starts in `draft` status
14. `Submit()` fails if listing has no files (returns `ListingHasNoFiles`)
15. `Submit()` fails from non-`draft` status (returns `InvalidStateTransition`)
16. `Publish()` only succeeds from `submitted` status
17. `Publish()` sets `published_at` timestamp
18. `Archive()` only succeeds from `published` status
19. `Archive()` sets `archived_at` timestamp; `published_at` retained
20. Updating a published listing creates a version snapshot in `mkt_listing_versions`
21. Updating a published listing increments the `version` field
22. Creator must be member of the listing's publisher to create/edit

### Purchase Flow

23. Adding to cart fails if listing not published (`404`)
24. Adding to cart fails if already purchased (`409`)
25. Cart is family-scoped: Parent A's additions visible to Parent B
26. `POST /cart/checkout` calls `PaymentAdapter.CreatePayment()` with correct split rules
27. Empty cart checkout returns `400`
28. Payment webhook creates `mkt_purchases` rows for each cart item
29. Payment webhook clears cart items after successful purchase
30. Duplicate webhook processing is idempotent (no duplicate purchases)
31. `PurchaseCompleted` event published with correct `PurchaseMetadata` shape
32. `PurchaseMetadata` contains `content_type`, `content_ids`, `publisher_id`
33. Free listing "Get" flow creates purchase with `amount_cents = 0`, no payment adapter call

### Reviews

34. Review creation requires verified purchase (JOIN to `mkt_purchases`)
35. One review per purchase (UNIQUE constraint)
36. Reviews are anonymous by default (`is_anonymous = true`)
37. Aggregate `rating_avg` and `rating_count` updated on `mkt_listings` after review create/update/delete
38. Creator can respond to review; response visible publicly
39. Review text sent to `safety::` for moderation

### Search & Discovery

40. FTS search on title (weight A) and description (weight B)
41. Faceted filters: methodology, subject, grade, price range, content type, worldview, rating
42. Sort options: relevance, price asc/desc, rating, newest
43. Trigram autocomplete returns results for partial title matches
44. Curated sections returned with nested listing data

### File Delivery

45. Download requires verified purchase (no subscription tier check)
46. Signed URL expires after 1 hour
47. File MIME type validated via magic bytes, not just extension

### Cross-Domain Events

48. `PurchaseCompleted` event shape matches learn::event_handlers expectation: `{ FamilyID: uuid.UUID, ContentMetadata: PurchaseMetadata }`
49. `ListingPublished` event triggers search index update
50. `ContentFlagged` event from `safety::` archives flagged listings / rejects flagged reviews
51. `FamilyDeletionScheduled` event anonymizes reviews and clears cart

### Error Handling

52. Internal errors (DB, cache) return `500` with generic message — no details exposed
53. Payment provider errors return `502 Bad Gateway`
54. All `MktError` variants map to documented HTTP status codes

---

## §21 Module Structure

```
internal/mkt/
├── handler.go                # Echo route handlers (thin layer only)
├── service.go                # MarketplaceServiceImpl — orchestration
├── repository.go             # PgCreatorRepository, PgPublisherRepository,
│                             # PgListingRepository, PgListingFileRepository,
│                             # PgCartRepository, PgPurchaseRepository,
│                             # PgReviewRepository, PgCuratedSectionRepository
├── models.go                 # Request/response types, internal types, query params
│                             # (includes GORM models — no separate entities dir)
├── ports.go                  # MarketplaceService interface, all repository interfaces,
│                             # PaymentAdapter interface, MediaAdapter interface
├── errors.go                 # MktError sentinel errors (errors.Is/As)
├── events.go                 # PurchaseCompleted, ListingPublished, ListingArchived,
│                             # ReviewCreated, CreatorOnboarded, PurchaseRefunded,
│                             # ListingSubmitted
├── event_handlers.go         # ContentFlaggedHandler, FamilyDeletionScheduledHandler,
│                             # MethodologyConfigUpdatedHandler
├── middleware.go              # RequireCreator middleware [00-core §13.3 resolution]
├── adapters/
│   └── payment.go            # HyperswitchPaymentAdapter — wraps Hyperswitch REST API
│                             # via net/http, returns domain types only [supersedes ADR-007]
└── domain/
    ├── listing.go            # MarketplaceListing aggregate root — state machine
    ├── value_objects.go      # Price, ListingTitle, etc.
    └── errors.go             # MktDomainError type
```

---

## §22 Addendum: Content Versioning & Purchase Updates `[S§20.10]`

*Added to address spec gaps in how marketplace content updates affect existing purchasers.*

### §22.1 Versioning Model

Marketplace listings already support versioning (§9 — Listing Lifecycle). This addendum
specifies the behavior when a creator publishes an updated version:

**Minor updates** (within the same major version):
- Typo fixes, additional resources, updated answer keys
- Automatically available to all existing purchasers
- Quiz/sequence items reference the listing version at purchase time; minor updates to
  content text propagate, but structural changes (adding/removing questions) do NOT
  affect in-progress quiz sessions
- Publisher updates the listing in-place (status remains `published`)

**Major version updates** (new major version, e.g., v1 → v2):
- Significant rewrites, restructured sequences, new quiz questions
- Creator publishes as a new version (creates a new `mkt_listing_versions` row)
- Existing purchasers of v1 continue to access v1 content
- Creator MAY offer v2 as:
  - Free upgrade (all v1 purchasers automatically get v2 access)
  - Discounted upgrade (v1 purchasers see a reduced price for v2)
  - Full-price purchase (no discount — v2 treated as a separate product)
- The upgrade policy is set per listing via a `version_upgrade_policy` field

### §22.2 Schema Addition

```sql
-- Added to mkt_listing_versions (existing table from §3)
ALTER TABLE mkt_listing_versions ADD COLUMN
    upgrade_policy VARCHAR(20) DEFAULT 'free'
    CHECK (upgrade_policy IN ('free', 'discounted', 'full_price'));

ALTER TABLE mkt_listing_versions ADD COLUMN
    upgrade_discount_percentage SMALLINT
    CHECK (upgrade_discount_percentage IS NULL OR
           (upgrade_discount_percentage > 0 AND upgrade_discount_percentage <= 100));
```

### §22.3 Quiz Session Version Pinning

When a student starts a quiz session, the session records the `listing_version_id`.
All quiz questions and answers for that session are frozen to the version at session start.
This ensures:
- A student's in-progress quiz is never affected by a content update
- Completed quiz scores reference the exact question set that was presented
- Analytics can track scores across different versions

### §22.4 Sequence Progress Version Handling

Lesson sequences that are in-progress when a minor update occurs:
- New items added to the end: student sees them after completing existing items
- Items removed from the middle: student skips them (marked as "removed by publisher")
- Items reordered: student's position is preserved by item ID, not ordinal position

For major version updates: student continues on their current version until they complete
or a parent explicitly switches to the new version (which resets progress).

---

## §23 Addendum: Abuse Prevention Beyond Content Moderation `[S§12, S§17.10]`

*Added to address spec gaps in marketplace-specific fraud and abuse vectors.*

### §23.1 Marketplace Fraud Vectors

| Vector | Description | Mitigation |
|--------|-------------|------------|
| **Fake reviews** | Creator creates alt accounts to leave positive reviews | Verified-purchaser requirement (already in §14); minimum account age (30 days) before reviews; statistical outlier detection on review patterns |
| **Self-purchasing** | Creator buys own content to boost sales numbers | Revenue from self-purchases is not paid out; flagged in creator analytics |
| **Price manipulation** | Creator rapidly changes price to exploit ranking algorithms | Price change rate limited to 2 changes per listing per 24 hours |
| **Copyrighted content** | Creator uploads content they don't own rights to | DMCA takedown process `[S§17.12]`; creator agreement includes IP attestation |
| **Review bombing** | Coordinated negative reviews on a competitor's listing | Review velocity limits (1 review per listing per account); automated spike detection |
| **Refund abuse** | Buyer repeatedly purchases and refunds to access content for free | Maximum 3 refunds per account per quarter; flagged accounts reviewed by admin |

### §23.2 Automated Detection

The following signals are monitored by the marketplace (integrated with safety:: for action):

```go
// Marketplace fraud signals — checked on relevant operations

type MktFraudSignalType string

const (
    // SuspiciousReviewTiming: Same family purchased and reviewed within minutes
    FraudSignalSuspiciousReviewTiming MktFraudSignalType = "suspicious_review_timing"
    // SelfPurchaseSuspected: Creator's IP matches a purchaser's IP
    FraudSignalSelfPurchaseSuspected  MktFraudSignalType = "self_purchase_suspected"
    // ReviewSpike: Unusual spike in reviews for a listing (>5 in 24 hours)
    FraudSignalReviewSpike            MktFraudSignalType = "review_spike"
    // RefundAbuse: Excessive refund requests from one account
    FraudSignalRefundAbuse            MktFraudSignalType = "refund_abuse"
    // PriceManipulation: Price changed more than twice in 24 hours
    FraudSignalPriceManipulation      MktFraudSignalType = "price_manipulation"
)

type MktFraudSignal struct {
    Type              MktFraudSignalType
    PurchaseID        *uuid.UUID
    ReviewID          *uuid.UUID
    CreatorID         *uuid.UUID
    PurchaserFamilyID *uuid.UUID
    ListingID         *uuid.UUID
    FamilyID          *uuid.UUID
    Count             uint32
    PeriodDays        uint32
}
```

Signals above threshold are published as `MktFraudSignalDetected` events →
`safety::` creates a moderation report for admin review.
