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
| **Module path** | `src/mkt/` |
| **DB prefix** | `mkt_` |
| **Complexity class** | Complex (has `domain/` subdirectory) `[ARCH §4.5]` |
| **CQRS** | Yes — listing commands separated from faceted browse/search reads `[ARCH §4.7]` |
| **External adapter** | `src/mkt/adapters/payment.rs` (Hyperswitch — processor-agnostic payment orchestration) |
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
-- Migration: YYYYMMDD_000001_create_mkt_tables.rs
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

All endpoints are prefixed with `/v1/marketplace`. Auth requirements use extractors defined
in `00-core §13`: `AuthContext` for authenticated users, `RequireCreator` for creator-only
endpoints (see §16 for extractor implementation). `[CODING §2.1]`

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
- **Response**: `200 OK` → `{ onboarding_url: String }`
- **Side effects**: Calls `PaymentAdapter::create_onboarding_link()`. Sets `onboarding_status = 'onboarding'` if currently `pending`.
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
- **Response**: `200 OK` → `Vec<PublisherMemberResponse>`
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
- **Query**: `{ q: String, limit?: u8 }`
- **Validation**: `q` minimum 2 chars, `limit` max 10
- **Response**: `200 OK` → `Vec<AutocompleteResult>`
- **Error codes**: `422`

##### `GET /v1/marketplace/curated-sections`

Get all active curated sections with their listings.

- **Auth**: `AuthContext`
- **Query**: `{ items_per_section?: u8 }`
- **Response**: `200 OK` → `Vec<CuratedSectionResponse>`

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
- **Response**: `200 OK` → `{ checkout_url: String, payment_session_id: String }`
- **Side effects**: Calls `PaymentAdapter::create_payment()` with split rules per listing. `[§11]`
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

- **Auth**: Webhook signature verification via `PaymentAdapter::verify_webhook()`
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
- **Query**: `{ period?: String }` (default: last 30 days)
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
- **Response**: `200 OK` → `{ payout_id: String, amount_cents: i64, status: String }`
- **Side effects**: Calls `PaymentAdapter::create_payout()`

##### `POST /v1/marketplace/publishers/:publisher_id/members`

Add a member to a publisher.

- **Auth**: `RequireCreator` + publisher owner/admin check
- **Body**: `{ creator_id: Uuid, role: String }`
- **Response**: `201 Created` → `PublisherMemberResponse`

### §4.3 Phase 3 (~4 endpoints)

##### `POST /v1/marketplace/listings/:listing_id/bundle`

Create a content bundle linking multiple listings.

- **Auth**: `RequireCreator`
- **Body**: `{ listing_ids: Vec<Uuid>, bundle_price_cents: i64 }`

##### `POST /v1/marketplace/admin/listings/:listing_id/publish`

Admin-only publish for moderated content.

- **Auth**: Admin role (TBD)

##### `DELETE /v1/marketplace/publishers/:publisher_id/members/:member_id`

Remove a member from a publisher.

- **Auth**: `RequireCreator` + publisher owner check

##### `GET /v1/marketplace/listings/:listing_id/versions`

List version history for a listing.

- **Auth**: `RequireCreator` + listing owner check
- **Response**: `200 OK` → `Vec<ListingVersionResponse>`

---

## §5 Service Interface

The `MarketplaceService` trait defines all use cases exposed to handlers and other domains.
Commands and queries are clearly separated per CQRS conventions. `[ARCH §4.7, CODING §8.2]`

```rust
// src/mkt/ports.rs

use crate::shared::types::{FamilyId, FamilyScope};
use crate::shared::error::AppError;
use crate::shared::pagination::PaginatedResponse;

#[async_trait]
pub trait MarketplaceService: Send + Sync {
    // ─── Command side (write, has side effects) ─────────────────────────
    // Return only IDs or () — never rich reads after write. [CODING §8.5]

    // Creator onboarding
    async fn register_creator(
        &self,
        cmd: RegisterCreatorCommand,
        auth: &AuthContext,
    ) -> Result<Uuid, AppError>;

    async fn update_creator_profile(
        &self,
        cmd: UpdateCreatorProfileCommand,
        creator_id: Uuid,
    ) -> Result<(), AppError>;

    async fn create_onboarding_link(
        &self,
        creator_id: Uuid,
    ) -> Result<String, AppError>;

    // Publisher management
    async fn create_publisher(
        &self,
        cmd: CreatePublisherCommand,
        creator_id: Uuid,
    ) -> Result<Uuid, AppError>;

    async fn update_publisher(
        &self,
        cmd: UpdatePublisherCommand,
        publisher_id: Uuid,
        creator_id: Uuid,
    ) -> Result<(), AppError>;

    async fn add_publisher_member(
        &self,
        publisher_id: Uuid,
        creator_id: Uuid,
        role: String,
        acting_creator_id: Uuid,
    ) -> Result<(), AppError>;

    async fn remove_publisher_member(
        &self,
        publisher_id: Uuid,
        member_id: Uuid,
        acting_creator_id: Uuid,
    ) -> Result<(), AppError>;

    // Listing lifecycle
    async fn create_listing(
        &self,
        cmd: CreateListingCommand,
        creator_id: Uuid,
    ) -> Result<Uuid, AppError>;

    async fn update_listing(
        &self,
        cmd: UpdateListingCommand,
        listing_id: Uuid,
        creator_id: Uuid,
    ) -> Result<(), AppError>;

    async fn submit_listing(
        &self,
        listing_id: Uuid,
        creator_id: Uuid,
    ) -> Result<(), AppError>;

    async fn publish_listing(
        &self,
        listing_id: Uuid,
        creator_id: Uuid,
    ) -> Result<(), AppError>;

    async fn archive_listing(
        &self,
        listing_id: Uuid,
        creator_id: Uuid,
    ) -> Result<(), AppError>;

    async fn upload_listing_file(
        &self,
        cmd: UploadListingFileCommand,
        listing_id: Uuid,
        creator_id: Uuid,
    ) -> Result<Uuid, AppError>;

    // Cart & checkout
    async fn add_to_cart(
        &self,
        listing_id: Uuid,
        scope: FamilyScope,
        parent_id: Uuid,
    ) -> Result<(), AppError>;

    async fn remove_from_cart(
        &self,
        listing_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    async fn create_checkout(
        &self,
        scope: FamilyScope,
    ) -> Result<CheckoutSession, AppError>;

    async fn handle_payment_webhook(
        &self,
        payload: Vec<u8>,
        signature: String,
    ) -> Result<(), AppError>;

    // Reviews
    async fn create_review(
        &self,
        cmd: CreateReviewCommand,
        listing_id: Uuid,
        scope: FamilyScope,
    ) -> Result<Uuid, AppError>;

    async fn update_review(
        &self,
        cmd: UpdateReviewCommand,
        review_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    async fn delete_review(
        &self,
        review_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;

    async fn respond_to_review(
        &self,
        cmd: RespondToReviewCommand,
        review_id: Uuid,
        creator_id: Uuid,
    ) -> Result<(), AppError>;

    // Free content acquisition
    async fn get_free_listing(
        &self,
        listing_id: Uuid,
        scope: FamilyScope,
    ) -> Result<Uuid, AppError>;

    // Payouts (Phase 2)
    async fn request_payout(
        &self,
        creator_id: Uuid,
    ) -> Result<PayoutResult, AppError>;

    // Event handlers (cross-domain reactions)
    async fn handle_content_flagged(
        &self,
        listing_id: Uuid,
        reason: String,
    ) -> Result<(), AppError>;

    async fn handle_family_deletion_scheduled(
        &self,
        family_id: FamilyId,
    ) -> Result<(), AppError>;

    // ─── Query side (read, no side effects) ─────────────────────────────

    // Creator queries
    async fn get_creator_by_parent_id(
        &self,
        parent_id: Uuid,
    ) -> Result<Option<CreatorResponse>, AppError>;

    async fn get_creator_dashboard(
        &self,
        creator_id: Uuid,
        period: DashboardPeriod,
    ) -> Result<CreatorDashboardResponse, AppError>;

    async fn get_creator_listings(
        &self,
        creator_id: Uuid,
        params: CreatorListingQueryParams,
    ) -> Result<PaginatedResponse<ListingDetailResponse>, AppError>;

    // Publisher queries
    async fn get_publisher(
        &self,
        publisher_id: Uuid,
    ) -> Result<PublisherResponse, AppError>;

    async fn get_publisher_members(
        &self,
        publisher_id: Uuid,
        creator_id: Uuid,
    ) -> Result<Vec<PublisherMemberResponse>, AppError>;

    /// Verify that creator_id is a member of publisher_id with the given
    /// minimum role. Consumed by learn:: for publisher ownership checks.
    /// [06-learn §18.2]
    async fn verify_publisher_membership(
        &self,
        publisher_id: Uuid,
        creator_id: Uuid,
    ) -> Result<bool, AppError>;

    // Listing browse
    async fn browse_listings(
        &self,
        params: BrowseListingsParams,
    ) -> Result<PaginatedResponse<ListingBrowseResponse>, AppError>;

    async fn get_listing(
        &self,
        listing_id: Uuid,
    ) -> Result<ListingDetailResponse, AppError>;

    async fn autocomplete_listings(
        &self,
        query: String,
        limit: u8,
    ) -> Result<Vec<AutocompleteResult>, AppError>;

    async fn get_curated_sections(
        &self,
        items_per_section: u8,
    ) -> Result<Vec<CuratedSectionResponse>, AppError>;

    // Cart queries
    async fn get_cart(
        &self,
        scope: FamilyScope,
    ) -> Result<CartResponse, AppError>;

    // Purchase queries
    async fn get_purchases(
        &self,
        scope: FamilyScope,
        params: PurchaseQueryParams,
    ) -> Result<PaginatedResponse<PurchaseResponse>, AppError>;

    async fn get_download_url(
        &self,
        listing_id: Uuid,
        file_id: Uuid,
        scope: FamilyScope,
    ) -> Result<DownloadResponse, AppError>;

    // Review queries
    async fn get_listing_reviews(
        &self,
        listing_id: Uuid,
        params: ReviewQueryParams,
    ) -> Result<PaginatedResponse<ReviewResponse>, AppError>;
}
```

### Implementation

```rust
// src/mkt/service.rs

pub struct MarketplaceServiceImpl {
    creators: Arc<dyn CreatorRepository>,
    publishers: Arc<dyn PublisherRepository>,
    listings: Arc<dyn ListingRepository>,
    listing_files: Arc<dyn ListingFileRepository>,
    cart: Arc<dyn CartRepository>,
    purchases: Arc<dyn PurchaseRepository>,
    reviews: Arc<dyn ReviewRepository>,
    curated_sections: Arc<dyn CuratedSectionRepository>,
    payment: Arc<dyn PaymentAdapter>,
    media: Arc<dyn MediaAdapter>,
    events: Arc<EventBus>,
}

impl MarketplaceServiceImpl {
    pub fn new(
        creators: Arc<dyn CreatorRepository>,
        publishers: Arc<dyn PublisherRepository>,
        listings: Arc<dyn ListingRepository>,
        listing_files: Arc<dyn ListingFileRepository>,
        cart: Arc<dyn CartRepository>,
        purchases: Arc<dyn PurchaseRepository>,
        reviews: Arc<dyn ReviewRepository>,
        curated_sections: Arc<dyn CuratedSectionRepository>,
        payment: Arc<dyn PaymentAdapter>,
        media: Arc<dyn MediaAdapter>,
        events: Arc<EventBus>,
    ) -> Self {
        Self {
            creators,
            publishers,
            listings,
            listing_files,
            cart,
            purchases,
            reviews,
            curated_sections,
            payment,
            media,
            events,
        }
    }
}

impl MarketplaceService for MarketplaceServiceImpl { /* ... */ }
```

---

## §6 Repository Interfaces

Each repository trait maps to one or more database tables. Family-scoped methods accept
`FamilyScope` to enforce privacy. Non-family-scoped methods (creators, publishers, listings)
are documented as such. `[CODING §8.2]`

```rust
// src/mkt/ports.rs (continued)

// ─── CreatorRepository ──────────────────────────────────────────────────
// NOT family-scoped — creator accounts are per-parent, not per-family.
#[async_trait]
pub trait CreatorRepository: Send + Sync {
    async fn create(&self, cmd: CreateCreator) -> Result<MktCreator, AppError>;
    async fn get_by_id(&self, creator_id: Uuid) -> Result<Option<MktCreator>, AppError>;
    async fn get_by_parent_id(&self, parent_id: Uuid) -> Result<Option<MktCreator>, AppError>;
    async fn update(&self, creator_id: Uuid, cmd: UpdateCreator) -> Result<MktCreator, AppError>;
    async fn set_onboarding_status(
        &self,
        creator_id: Uuid,
        status: String,
    ) -> Result<(), AppError>;
    async fn set_payment_account_id(
        &self,
        creator_id: Uuid,
        payment_account_id: String,
    ) -> Result<(), AppError>;
}

// ─── PublisherRepository ────────────────────────────────────────────────
// NOT family-scoped — publishers are organization-level entities.
#[async_trait]
pub trait PublisherRepository: Send + Sync {
    async fn create(&self, cmd: CreatePublisher) -> Result<MktPublisher, AppError>;
    async fn get_by_id(&self, publisher_id: Uuid) -> Result<Option<MktPublisher>, AppError>;
    async fn get_by_slug(&self, slug: &str) -> Result<Option<MktPublisher>, AppError>;
    async fn update(
        &self,
        publisher_id: Uuid,
        cmd: UpdatePublisher,
    ) -> Result<MktPublisher, AppError>;
    async fn get_platform_publisher(&self) -> Result<MktPublisher, AppError>;

    // Membership
    async fn add_member(
        &self,
        publisher_id: Uuid,
        creator_id: Uuid,
        role: &str,
    ) -> Result<(), AppError>;
    async fn remove_member(
        &self,
        publisher_id: Uuid,
        creator_id: Uuid,
    ) -> Result<(), AppError>;
    async fn get_members(
        &self,
        publisher_id: Uuid,
    ) -> Result<Vec<PublisherMemberRow>, AppError>;
    async fn get_member_role(
        &self,
        publisher_id: Uuid,
        creator_id: Uuid,
    ) -> Result<Option<String>, AppError>;
    async fn get_publishers_for_creator(
        &self,
        creator_id: Uuid,
    ) -> Result<Vec<MktPublisher>, AppError>;
}

// ─── ListingRepository ──────────────────────────────────────────────────
// NOT family-scoped — listings are creator-owned, publicly browsable.
// CQRS: command methods (create/update/save) and query methods (browse/get)
// are clearly separated. [CODING §8.5]
#[async_trait]
pub trait ListingRepository: Send + Sync {
    // Command side
    async fn create(&self, cmd: CreateListing) -> Result<MktListing, AppError>;
    async fn save(&self, listing: &MarketplaceListing) -> Result<(), AppError>;
    async fn create_version_snapshot(
        &self,
        listing_id: Uuid,
        version: i32,
        title: &str,
        description: &str,
        price_cents: i32,
        change_summary: Option<&str>,
    ) -> Result<(), AppError>;

    // Query side
    async fn get_by_id(&self, listing_id: Uuid) -> Result<Option<MktListing>, AppError>;
    async fn browse(
        &self,
        params: &BrowseListingsParams,
    ) -> Result<PaginatedResponse<ListingBrowseRow>, AppError>;
    async fn autocomplete(
        &self,
        query: &str,
        limit: u8,
    ) -> Result<Vec<AutocompleteRow>, AppError>;
    async fn get_by_creator(
        &self,
        creator_id: Uuid,
        params: &CreatorListingQueryParams,
    ) -> Result<PaginatedResponse<MktListing>, AppError>;
    async fn get_versions(
        &self,
        listing_id: Uuid,
    ) -> Result<Vec<MktListingVersion>, AppError>;
    async fn count_files(&self, listing_id: Uuid) -> Result<i64, AppError>;
}

// ─── ListingFileRepository ──────────────────────────────────────────────
#[async_trait]
pub trait ListingFileRepository: Send + Sync {
    async fn create(&self, cmd: CreateListingFile) -> Result<MktListingFile, AppError>;
    async fn get_by_id(
        &self,
        listing_id: Uuid,
        file_id: Uuid,
    ) -> Result<Option<MktListingFile>, AppError>;
    async fn list_by_listing(
        &self,
        listing_id: Uuid,
    ) -> Result<Vec<MktListingFile>, AppError>;
    async fn delete(&self, file_id: Uuid) -> Result<(), AppError>;
}

// ─── CartRepository ─────────────────────────────────────────────────────
// Family-scoped — cart belongs to the family. [00-core §8]
#[async_trait]
pub trait CartRepository: Send + Sync {
    async fn add_item(
        &self,
        listing_id: Uuid,
        parent_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;
    async fn remove_item(
        &self,
        listing_id: Uuid,
        scope: FamilyScope,
    ) -> Result<(), AppError>;
    async fn get_items(
        &self,
        scope: FamilyScope,
    ) -> Result<Vec<CartItemRow>, AppError>;
    async fn clear(&self, scope: FamilyScope) -> Result<(), AppError>;
}

// ─── PurchaseRepository ─────────────────────────────────────────────────
// Family-scoped on read queries. System-scoped writes (from webhook handler).
#[async_trait]
pub trait PurchaseRepository: Send + Sync {
    async fn create(&self, cmd: CreatePurchase) -> Result<MktPurchase, AppError>;
    async fn get_by_family_and_listing(
        &self,
        family_id: Uuid,
        listing_id: Uuid,
    ) -> Result<Option<MktPurchase>, AppError>;
    async fn list_by_family(
        &self,
        scope: FamilyScope,
        params: &PurchaseQueryParams,
    ) -> Result<PaginatedResponse<PurchaseRow>, AppError>;
    async fn get_by_payment_session_id(
        &self,
        session_id: &str,
    ) -> Result<Option<MktPurchase>, AppError>;
    async fn set_refund(
        &self,
        purchase_id: Uuid,
        refund_id: &str,
        refund_amount_cents: i32,
    ) -> Result<(), AppError>;
    async fn get_creator_sales(
        &self,
        creator_id: Uuid,
        from: DateTime<Utc>,
        to: DateTime<Utc>,
    ) -> Result<Vec<SalesRow>, AppError>;
    async fn delete_by_family(
        &self,
        family_id: Uuid,
    ) -> Result<(), AppError>;
}

// ─── ReviewRepository ───────────────────────────────────────────────────
// Family-scoped writes, public reads.
#[async_trait]
pub trait ReviewRepository: Send + Sync {
    async fn create(&self, cmd: CreateReview) -> Result<MktReview, AppError>;
    async fn get_by_id(&self, review_id: Uuid) -> Result<Option<MktReview>, AppError>;
    async fn update(&self, review_id: Uuid, cmd: UpdateReview) -> Result<MktReview, AppError>;
    async fn delete(&self, review_id: Uuid) -> Result<(), AppError>;
    async fn set_creator_response(
        &self,
        review_id: Uuid,
        response_text: &str,
    ) -> Result<(), AppError>;
    async fn list_by_listing(
        &self,
        listing_id: Uuid,
        params: &ReviewQueryParams,
    ) -> Result<PaginatedResponse<ReviewRow>, AppError>;
    async fn get_aggregate_rating(
        &self,
        listing_id: Uuid,
    ) -> Result<(f64, i32), AppError>;
    async fn set_moderation_status(
        &self,
        review_id: Uuid,
        status: &str,
    ) -> Result<(), AppError>;
}

// ─── CuratedSectionRepository ───────────────────────────────────────────
#[async_trait]
pub trait CuratedSectionRepository: Send + Sync {
    async fn list_active(&self) -> Result<Vec<MktCuratedSection>, AppError>;
    async fn get_section_items(
        &self,
        section_id: Uuid,
        limit: u8,
    ) -> Result<Vec<ListingBrowseRow>, AppError>;
    async fn add_item(
        &self,
        section_id: Uuid,
        listing_id: Uuid,
        sort_order: i16,
    ) -> Result<(), AppError>;
    async fn remove_item(
        &self,
        section_id: Uuid,
        listing_id: Uuid,
    ) -> Result<(), AppError>;
    async fn refresh_auto_section(
        &self,
        section_slug: &str,
    ) -> Result<(), AppError>;
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

```rust
// src/mkt/adapters/payment.rs

/// Processor-agnostic payment adapter backed by Hyperswitch.
/// Implementations: HyperswitchPaymentAdapter (production), MockPaymentAdapter (tests).
#[async_trait]
pub trait PaymentAdapter: Send + Sync {
    // ─── Account Management ─────────────────────────────────────────────
    // Sub-merchant onboarding for creator payouts. Maps to Hyperswitch's
    // Organization → Merchant → Profile hierarchy.

    /// Create a sub-merchant account in Hyperswitch for a creator.
    /// Returns the Hyperswitch merchant/sub-merchant ID.
    async fn create_sub_merchant(
        &self,
        config: SubMerchantConfig,
    ) -> Result<String, PaymentError>;

    /// Generate an onboarding link for creator KYC/identity verification.
    /// The link redirects to the underlying processor's onboarding flow
    /// (e.g., Stripe Connect onboarding) orchestrated through Hyperswitch.
    async fn create_onboarding_link(
        &self,
        payment_account_id: &str,
        return_url: &str,
    ) -> Result<String, PaymentError>;

    /// Check the onboarding/verification status of a sub-merchant account.
    async fn get_account_status(
        &self,
        payment_account_id: &str,
    ) -> Result<AccountStatus, PaymentError>;

    // ─── Payments ───────────────────────────────────────────────────────
    // Split payments for marketplace transactions. Hyperswitch handles
    // routing to the configured payment processor.

    /// Create a payment session with split payment rules.
    /// Returns a checkout URL and payment session ID.
    async fn create_payment(
        &self,
        line_items: Vec<PaymentLineItem>,
        split_rules: Vec<SplitRule>,
        return_url: &str,
        metadata: HashMap<String, String>,
    ) -> Result<PaymentSession, PaymentError>;

    /// Check the status of a payment.
    async fn get_payment_status(
        &self,
        payment_id: &str,
    ) -> Result<PaymentStatus, PaymentError>;

    // ─── Payouts (Phase 2) ──────────────────────────────────────────────

    /// Create a payout to a creator's connected account.
    async fn create_payout(
        &self,
        payment_account_id: &str,
        amount_cents: i64,
        currency: &str,
    ) -> Result<PayoutResult, PaymentError>;

    // ─── Refunds ────────────────────────────────────────────────────────

    /// Create a refund for a payment.
    async fn create_refund(
        &self,
        payment_id: &str,
        amount_cents: i64,
        reason: &str,
    ) -> Result<RefundResult, PaymentError>;

    // ─── Webhooks ───────────────────────────────────────────────────────

    /// Verify the signature of an incoming Hyperswitch webhook.
    async fn verify_webhook(
        &self,
        payload: &[u8],
        signature: &str,
    ) -> Result<bool, PaymentError>;

    /// Parse a verified webhook payload into a domain event.
    async fn parse_event(
        &self,
        payload: &[u8],
    ) -> Result<PaymentEvent, PaymentError>;
}

// ─── Supporting Types ───────────────────────────────────────────────────

pub struct SubMerchantConfig {
    pub creator_id: Uuid,
    pub store_name: String,
    pub email: String,
    pub country: String,     // ISO 3166-1 alpha-2
}

pub enum AccountStatus {
    /// Account created but onboarding not started
    Pending,
    /// Creator is going through KYC/verification
    Onboarding,
    /// Fully verified, can receive payouts
    Active,
    /// Account suspended (compliance issue)
    Suspended,
}

pub struct PaymentLineItem {
    pub listing_id: Uuid,
    pub amount_cents: i64,
    pub description: String,
}

pub struct SplitRule {
    pub recipient_account_id: String,   // creator's Hyperswitch sub-merchant ID
    pub amount_cents: i64,              // creator's share
}

pub struct PaymentSession {
    pub checkout_url: String,
    pub payment_session_id: String,     // Hyperswitch payment ID (idempotency key)
}

pub enum PaymentStatus {
    Processing,
    Succeeded,
    Failed { reason: String },
    Cancelled,
}

pub struct PayoutResult {
    pub payout_id: String,
    pub amount_cents: i64,
    pub status: String,
}

pub struct RefundResult {
    pub refund_id: String,
    pub amount_cents: i64,
    pub status: String,
}

pub enum PaymentEvent {
    PaymentSucceeded {
        payment_id: String,
        metadata: HashMap<String, String>,
    },
    PaymentFailed {
        payment_id: String,
        reason: String,
    },
    RefundSucceeded {
        payment_id: String,
        refund_id: String,
        amount_cents: i64,
    },
    PayoutCompleted {
        payout_id: String,
        merchant_id: String,
        amount_cents: i64,
    },
}
```

### MediaAdapter

File upload and signed-URL generation delegated to `media::`. `[ARCH §4.2]`

```rust
// Consumed from media:: — not defined here. See media:: domain spec.
// Used for: presigned upload URLs (creator uploads), presigned download
// URLs (purchaser downloads). [ARCH §8.3]
#[async_trait]
pub trait MediaAdapter: Send + Sync {
    async fn presigned_upload(
        &self,
        key: &str,
        content_type: &str,
        max_size_bytes: u64,
    ) -> Result<String, AppError>;

    async fn presigned_get(
        &self,
        key: &str,
        expires_seconds: u32,
    ) -> Result<String, AppError>;
}
```

---

## §8 Models (DTOs)

All request/response types derive `serde::Serialize`/`Deserialize` and OpenAPI schema derives
for code generation. `[CODING §2.1]`

### §8.1 Request Types

```rust
// src/mkt/models.rs

#[derive(Debug, Deserialize, ToSchema)]
pub struct RegisterCreatorCommand {
    pub store_name: String,
    pub store_bio: Option<String>,
    pub store_logo_url: Option<String>,
    pub tos_accepted: bool,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct UpdateCreatorProfileCommand {
    pub store_name: Option<String>,
    pub store_bio: Option<String>,
    pub store_logo_url: Option<String>,
    pub store_banner_url: Option<String>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct CreatePublisherCommand {
    pub name: String,
    pub slug: Option<String>,
    pub description: Option<String>,
    pub logo_url: Option<String>,
    pub website_url: Option<String>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct UpdatePublisherCommand {
    pub name: Option<String>,
    pub description: Option<String>,
    pub logo_url: Option<String>,
    pub website_url: Option<String>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct CreateListingCommand {
    pub publisher_id: Uuid,
    pub title: String,
    pub description: String,
    pub price_cents: i32,
    pub methodology_tags: Vec<Uuid>,
    pub subject_tags: Vec<String>,
    pub grade_min: Option<i16>,
    pub grade_max: Option<i16>,
    pub content_type: String,
    pub worldview_tags: Option<Vec<String>>,
    pub preview_url: Option<String>,
    pub thumbnail_url: Option<String>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct UpdateListingCommand {
    pub title: Option<String>,
    pub description: Option<String>,
    pub price_cents: Option<i32>,
    pub methodology_tags: Option<Vec<Uuid>>,
    pub subject_tags: Option<Vec<String>>,
    pub grade_min: Option<i16>,
    pub grade_max: Option<i16>,
    pub worldview_tags: Option<Vec<String>>,
    pub preview_url: Option<String>,
    pub thumbnail_url: Option<String>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct UploadListingFileCommand {
    pub file_name: String,
    pub file_size_bytes: i64,
    pub mime_type: String,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct AddToCartCommand {
    pub listing_id: Uuid,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct CreateCheckoutCommand {}

#[derive(Debug, Deserialize, ToSchema)]
pub struct CreateReviewCommand {
    pub rating: i16,
    pub review_text: Option<String>,
    pub is_anonymous: Option<bool>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct UpdateReviewCommand {
    pub rating: Option<i16>,
    pub review_text: Option<String>,
    pub is_anonymous: Option<bool>,
}

#[derive(Debug, Deserialize, ToSchema)]
pub struct RespondToReviewCommand {
    pub response_text: String,
}
```

### §8.2 Response Types

```rust
#[derive(Debug, Serialize, ToSchema)]
pub struct CreatorResponse {
    pub id: Uuid,
    pub parent_id: Uuid,
    pub onboarding_status: String,
    pub store_name: String,
    pub store_bio: Option<String>,
    pub store_logo_url: Option<String>,
    pub store_banner_url: Option<String>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct PublisherResponse {
    pub id: Uuid,
    pub name: String,
    pub slug: String,
    pub description: Option<String>,
    pub logo_url: Option<String>,
    pub website_url: Option<String>,
    pub is_verified: bool,
    pub member_count: i32,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct PublisherMemberResponse {
    pub creator_id: Uuid,
    pub store_name: String,
    pub role: String,
    pub joined_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ListingBrowseResponse {
    pub id: Uuid,
    pub title: String,
    pub description_preview: String,    // truncated to ~200 chars
    pub price_cents: i32,
    pub content_type: String,
    pub thumbnail_url: Option<String>,
    pub rating_avg: f64,
    pub rating_count: i32,
    pub publisher_name: String,
    pub creator_store_name: String,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ListingDetailResponse {
    pub id: Uuid,
    pub creator_id: Uuid,
    pub publisher_id: Uuid,
    pub publisher_name: String,
    pub title: String,
    pub description: String,
    pub price_cents: i32,
    pub methodology_tags: Vec<Uuid>,
    pub subject_tags: Vec<String>,
    pub grade_min: Option<i16>,
    pub grade_max: Option<i16>,
    pub content_type: String,
    pub worldview_tags: Vec<String>,
    pub preview_url: Option<String>,
    pub thumbnail_url: Option<String>,
    pub status: String,
    pub rating_avg: f64,
    pub rating_count: i32,
    pub version: i32,
    pub files: Vec<ListingFileResponse>,
    pub published_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
    pub updated_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ListingFileResponse {
    pub id: Uuid,
    pub file_name: String,
    pub file_size_bytes: i64,
    pub mime_type: String,
    pub version: i32,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct CartResponse {
    pub items: Vec<CartItemResponse>,
    pub total_cents: i64,
    pub item_count: i32,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct CartItemResponse {
    pub listing_id: Uuid,
    pub title: String,
    pub price_cents: i32,
    pub thumbnail_url: Option<String>,
    pub added_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct PurchaseResponse {
    pub id: Uuid,
    pub listing_id: Uuid,
    pub listing_title: String,
    pub amount_cents: i32,
    pub refunded: bool,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ReviewResponse {
    pub id: Uuid,
    pub listing_id: Uuid,
    pub rating: i16,
    pub review_text: Option<String>,
    pub is_anonymous: bool,
    pub reviewer_name: Option<String>,  // None if anonymous
    pub creator_response: Option<String>,
    pub creator_response_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct DownloadResponse {
    pub download_url: String,
    pub expires_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct AutocompleteResult {
    pub listing_id: Uuid,
    pub title: String,
    pub similarity: f32,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct CuratedSectionResponse {
    pub slug: String,
    pub display_name: String,
    pub description: Option<String>,
    pub listings: Vec<ListingBrowseResponse>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct CreatorDashboardResponse {
    pub total_sales_count: i64,
    pub total_earnings_cents: i64,
    pub period_sales_count: i64,
    pub period_earnings_cents: i64,
    pub pending_payout_cents: i64,
    pub average_rating: f64,
    pub total_reviews: i32,
    pub recent_sales: Vec<SaleSummary>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct SaleSummary {
    pub purchase_id: Uuid,
    pub listing_title: String,
    pub amount_cents: i32,
    pub creator_payout_cents: i32,
    pub purchased_at: DateTime<Utc>,
}

#[derive(Debug, Serialize, ToSchema)]
pub struct ListingVersionResponse {
    pub version: i32,
    pub title: String,
    pub price_cents: i32,
    pub change_summary: Option<String>,
    pub created_at: DateTime<Utc>,
}
```

### §8.3 Internal Types

```rust
// Query parameter types (not exposed in API responses)

#[derive(Debug, Deserialize)]
pub struct BrowseListingsParams {
    pub q: Option<String>,
    pub methodology_ids: Option<Vec<Uuid>>,
    pub subject_slugs: Option<Vec<String>>,
    pub grade_min: Option<i16>,
    pub grade_max: Option<i16>,
    pub content_type: Option<String>,
    pub worldview_tags: Option<Vec<String>>,
    pub price_min: Option<i32>,
    pub price_max: Option<i32>,
    pub min_rating: Option<f64>,
    pub sort_by: Option<ListingSortBy>,
    pub cursor: Option<String>,
    pub limit: Option<u8>,
}

#[derive(Debug, Deserialize)]
pub enum ListingSortBy {
    Relevance,
    PriceAsc,
    PriceDesc,
    Rating,
    Newest,
}

#[derive(Debug, Deserialize)]
pub struct CreatorListingQueryParams {
    pub status: Option<String>,
    pub cursor: Option<String>,
    pub limit: Option<u8>,
}

#[derive(Debug, Deserialize)]
pub struct PurchaseQueryParams {
    pub cursor: Option<String>,
    pub limit: Option<u8>,
}

#[derive(Debug, Deserialize)]
pub struct ReviewQueryParams {
    pub sort_by: Option<ReviewSortBy>,
    pub cursor: Option<String>,
    pub limit: Option<u8>,
}

#[derive(Debug, Deserialize)]
pub enum ReviewSortBy {
    Newest,
    Oldest,
    HighestRating,
    LowestRating,
}

pub enum DashboardPeriod {
    Last7Days,
    Last30Days,
    Last90Days,
    AllTime,
}

// Internal row types returned by repositories
pub struct ListingBrowseRow {
    pub id: Uuid,
    pub title: String,
    pub description: String,
    pub price_cents: i32,
    pub content_type: String,
    pub thumbnail_url: Option<String>,
    pub rating_avg: f64,
    pub rating_count: i32,
    pub publisher_name: String,
    pub creator_store_name: String,
}

pub struct AutocompleteRow {
    pub listing_id: Uuid,
    pub title: String,
    pub similarity: f32,
}

pub struct CartItemRow {
    pub listing_id: Uuid,
    pub title: String,
    pub price_cents: i32,
    pub thumbnail_url: Option<String>,
    pub created_at: DateTime<Utc>,
}

pub struct PurchaseRow {
    pub id: Uuid,
    pub listing_id: Uuid,
    pub listing_title: String,
    pub amount_cents: i32,
    pub refunded_at: Option<DateTime<Utc>>,
    pub created_at: DateTime<Utc>,
}

pub struct ReviewRow {
    pub id: Uuid,
    pub listing_id: Uuid,
    pub rating: i16,
    pub review_text: Option<String>,
    pub is_anonymous: bool,
    pub reviewer_family_name: Option<String>,
    pub creator_response: Option<String>,
    pub creator_response_at: Option<DateTime<Utc>>,
    pub moderation_status: String,
    pub created_at: DateTime<Utc>,
}

pub struct SalesRow {
    pub purchase_id: Uuid,
    pub listing_id: Uuid,
    pub listing_title: String,
    pub amount_cents: i32,
    pub creator_payout_cents: i32,
    pub created_at: DateTime<Utc>,
}

pub struct PublisherMemberRow {
    pub creator_id: Uuid,
    pub store_name: String,
    pub role: String,
    pub created_at: DateTime<Utc>,
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
| Draft | Submitted | `submit()` | At least 1 file attached, valid price, title/description present | — | `ListingSubmitted` |
| Submitted | Published | `publish()` | Content screening passed | Sets `published_at` | `ListingPublished` |
| Submitted | Draft | `reject()` | Content screening failed or creator withdraws | — | — |
| Published | Archived | `archive()` | — | Sets `archived_at` | `ListingArchived` |
| Published | Published | `update()` | — | Creates version snapshot, increments `version` | — |

### Invariants

1. A listing **must have at least one file** to be submitted. Enforced by checking file count
   in the `submit()` method.
2. A listing **must have a valid price** (`price_cents >= 0`) — enforced by the `Price` value
   object constructor.
3. Published listings **always have a `published_at` timestamp** — set atomically during the
   `publish()` transition.
4. Archived listings **retain `published_at`** — `archived_at` is additive, not a replacement.
5. Version snapshots are **only created for updates to published listings** — draft edits do
   not generate versions.

### Aggregate Root

```rust
// src/mkt/domain/listing.rs

use chrono::{DateTime, Utc};
use uuid::Uuid;

#[derive(Debug, Clone, PartialEq)]
pub enum ListingState {
    Draft,
    Submitted,
    Published,
    Archived,
}

#[derive(Debug)]
pub struct MarketplaceListing {
    id: Uuid,
    creator_id: Uuid,
    publisher_id: Uuid,
    title: String,
    description: String,
    price_cents: i32,
    methodology_tags: Vec<Uuid>,
    subject_tags: Vec<String>,
    grade_min: Option<i16>,
    grade_max: Option<i16>,
    content_type: String,
    worldview_tags: Vec<String>,
    preview_url: Option<String>,
    thumbnail_url: Option<String>,
    state: ListingState,
    rating_avg: f64,
    rating_count: i32,
    version: i32,
    published_at: Option<DateTime<Utc>>,
    archived_at: Option<DateTime<Utc>>,
    file_count: i64,       // denormalized for submit check
    created_at: DateTime<Utc>,
    updated_at: DateTime<Utc>,
}

impl MarketplaceListing {
    /// Reconstruct from persistence (all fields provided by repository).
    pub fn from_persistence(/* all fields */) -> Self { /* ... */ }

    // ─── Queries ────────────────────────────────────────────────────────

    pub fn id(&self) -> Uuid { self.id }
    pub fn state(&self) -> &ListingState { &self.state }
    pub fn creator_id(&self) -> Uuid { self.creator_id }
    pub fn publisher_id(&self) -> Uuid { self.publisher_id }
    pub fn version(&self) -> i32 { self.version }

    // ─── State Transitions ──────────────────────────────────────────────

    pub fn submit(&mut self) -> Result<ListingSubmittedEvent, MktDomainError> {
        if self.state != ListingState::Draft {
            return Err(MktDomainError::InvalidStateTransition {
                from: format!("{:?}", self.state),
                action: "submit".to_string(),
            });
        }
        if self.file_count == 0 {
            return Err(MktDomainError::ListingHasNoFiles);
        }
        self.state = ListingState::Submitted;
        Ok(ListingSubmittedEvent {
            listing_id: self.id,
            creator_id: self.creator_id,
        })
    }

    pub fn publish(&mut self) -> Result<ListingPublishedEvent, MktDomainError> {
        if self.state != ListingState::Submitted {
            return Err(MktDomainError::InvalidStateTransition {
                from: format!("{:?}", self.state),
                action: "publish".to_string(),
            });
        }
        self.state = ListingState::Published;
        self.published_at = Some(Utc::now());
        Ok(ListingPublishedEvent {
            listing_id: self.id,
            publisher_id: self.publisher_id,
            content_type: self.content_type.clone(),
            subject_tags: self.subject_tags.clone(),
        })
    }

    pub fn reject(&mut self) -> Result<(), MktDomainError> {
        if self.state != ListingState::Submitted {
            return Err(MktDomainError::InvalidStateTransition {
                from: format!("{:?}", self.state),
                action: "reject".to_string(),
            });
        }
        self.state = ListingState::Draft;
        Ok(())
    }

    pub fn archive(&mut self) -> Result<ListingArchivedEvent, MktDomainError> {
        if self.state != ListingState::Published {
            return Err(MktDomainError::InvalidStateTransition {
                from: format!("{:?}", self.state),
                action: "archive".to_string(),
            });
        }
        self.state = ListingState::Archived;
        self.archived_at = Some(Utc::now());
        Ok(ListingArchivedEvent {
            listing_id: self.id,
        })
    }

    /// Update a published listing. Returns the pre-update state for version snapshot.
    pub fn update_published(
        &mut self,
        title: Option<String>,
        description: Option<String>,
        price_cents: Option<i32>,
    ) -> Result<Option<VersionSnapshot>, MktDomainError> {
        if self.state != ListingState::Published && self.state != ListingState::Draft {
            return Err(MktDomainError::InvalidStateTransition {
                from: format!("{:?}", self.state),
                action: "update".to_string(),
            });
        }

        let snapshot = if self.state == ListingState::Published {
            let snap = VersionSnapshot {
                version: self.version,
                title: self.title.clone(),
                description: self.description.clone(),
                price_cents: self.price_cents,
            };
            self.version += 1;
            Some(snap)
        } else {
            None
        };

        if let Some(t) = title { self.title = t; }
        if let Some(d) = description { self.description = d; }
        if let Some(p) = price_cents { self.price_cents = p; }
        self.updated_at = Utc::now();

        Ok(snapshot)
    }
}

pub struct VersionSnapshot {
    pub version: i32,
    pub title: String,
    pub description: String,
    pub price_cents: i32,
}
```

### Domain Errors

```rust
// src/mkt/domain/errors.rs

use thiserror::Error;

#[derive(Debug, Error)]
pub enum MktDomainError {
    #[error("Invalid state transition from {from} via {action}")]
    InvalidStateTransition { from: String, action: String },

    #[error("Listing has no files attached")]
    ListingHasNoFiles,

    #[error("Invalid price: must be >= 0")]
    InvalidPrice,
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

### verify_publisher_membership()

The `MarketplaceService::verify_publisher_membership()` method is consumed by `learn::` to
verify that a creator has permission to create/edit content definitions under a publisher.
`[06-learn §18.2]`

```rust
// Called by learn:: service when creating/updating content definitions
let is_member = marketplace_service
    .verify_publisher_membership(publisher_id, creator_id)
    .await?;
if !is_member {
    return Err(AppError::Forbidden);
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

```rust
// Split calculation in checkout service
fn calculate_split(listing_price_cents: i64, creator_share_percent: u8) -> (i64, i64) {
    let creator_payout = (listing_price_cents * creator_share_percent as i64) / 100;
    let platform_fee = listing_price_cents - creator_payout;
    (creator_payout, platform_fee)
}

// Split rules sent to Hyperswitch
let split_rules: Vec<SplitRule> = cart_items.iter().map(|item| {
    let (creator_payout, _platform_fee) = calculate_split(
        item.price_cents as i64,
        config.creator_share_percent,  // e.g., 75
    );
    SplitRule {
        recipient_account_id: item.creator_payment_account_id.clone(),
        amount_cents: creator_payout,
    }
}).collect();
```

### Free Content "Get" Flow

Free content (`price_cents == 0`) bypasses the payment processor entirely. `[S§9.4]`

```rust
// POST /v1/marketplace/listings/:listing_id/get
async fn get_free_listing(
    &self,
    listing_id: Uuid,
    scope: FamilyScope,
) -> Result<Uuid, AppError> {
    let listing = self.listings.get_by_id(listing_id).await?
        .ok_or(MktError::ListingNotFound)?;

    if listing.price_cents != 0 {
        return Err(MktError::ListingNotFree.into());
    }
    if listing.status != "published" {
        return Err(MktError::ListingNotPublished.into());
    }

    // Check not already purchased
    if self.purchases.get_by_family_and_listing(
        scope.family_id(), listing_id
    ).await?.is_some() {
        return Err(MktError::AlreadyPurchased.into());
    }

    // Direct insert — no payment processor
    let purchase_id = self.purchases.create(CreatePurchase {
        family_id: scope.family_id(),
        listing_id,
        creator_id: listing.creator_id,
        payment_id: None,
        payment_session_id: None,
        amount_cents: 0,
        platform_fee_cents: 0,
        creator_payout_cents: 0,
    }).await?.id;

    self.events.publish(PurchaseCompleted {
        family_id: FamilyId::from(scope.family_id()),
        purchase_id,
        listing_id,
        content_metadata: PurchaseMetadata {
            content_type: listing.content_type.clone(),
            content_ids: vec![listing_id],
            publisher_id: listing.publisher_id,
        },
    })?;

    Ok(purchase_id)
}
```

### Idempotency

Payment webhook processing is idempotent via `payment_session_id`:

```rust
// In webhook handler
if self.purchases.get_by_payment_session_id(&payment_id).await?.is_some() {
    // Already processed — return 200 OK to acknowledge
    return Ok(());
}
```

### Refund Flow

Refunds are processed via Hyperswitch within a 30-day window. `[S§9.6]`

1. Family requests refund (Phase 2 endpoint, or admin-initiated)
2. Service calls `PaymentAdapter::create_refund(payment_id, amount_cents, reason)`
3. On `RefundSucceeded` webhook: update `mkt_purchases.refunded_at`, `refund_amount_cents`, `refund_id`
4. Publish `PurchaseRefunded` event
5. Refund deducted from creator's pending earnings

---

## §12 File Upload & Signed-URL Delivery (Domain Deep-Dive 4)

### Creator File Upload

Creators upload files to listings via `media::` presigned URLs. `[ARCH §8.3, CODING §5.2]`

1. Creator calls `POST /v1/marketplace/listings/:id/files` with file metadata
2. Service validates: listing exists, creator owns it, listing is in draft/published status
3. Service calls `MediaAdapter::presigned_upload()` to get a presigned R2 upload URL
4. Service creates `mkt_listing_files` record with `storage_key`
5. Returns presigned URL to client; client uploads directly to R2
6. **File validation**: MIME type validated via magic bytes on the server side when the
   upload completes (webhook or polling). `[CODING §5.2]`

### Purchaser Download

Purchasers download files via time-limited R2 signed URLs. `[ARCH §8.3]`

```rust
// GET /v1/marketplace/purchases/:listing_id/download/:file_id
async fn get_download_url(
    &self,
    listing_id: Uuid,
    file_id: Uuid,
    scope: FamilyScope,
) -> Result<DownloadResponse, AppError> {
    // Verify purchase exists — no subscription tier check [S§9.4]
    let _purchase = self.purchases
        .get_by_family_and_listing(scope.family_id(), listing_id)
        .await?
        .ok_or(MktError::NotPurchased)?;

    let file = self.listing_files
        .get_by_id(listing_id, file_id)
        .await?
        .ok_or(MktError::FileNotFound)?;

    // Generate 1-hour signed URL
    let signed_url = self.media
        .presigned_get(&file.storage_key, 3600)
        .await?;

    let expires_at = Utc::now() + chrono::Duration::hours(1);

    Ok(DownloadResponse {
        download_url: signed_url,
        expires_at,
    })
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
2. `learn::` service checks `mkt::PurchaseRepository::get_by_family_and_listing(family_id, listing_id)`
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

- `ListingRepository::browse()` switches from SQL to Meilisearch HTTP client
- `ListingPublished` / `ListingArchived` events trigger index updates
- The search API response shape (`ListingBrowseResponse`) remains unchanged

---

## §14 Review System & Moderation (Domain Deep-Dive 6)

### Verified-Purchaser Enforcement

Only families who have purchased a listing can leave a review. Enforced via `purchase_id`
foreign key and a service-level check. `[S§9.5]`

```rust
async fn create_review(
    &self,
    cmd: CreateReviewCommand,
    listing_id: Uuid,
    scope: FamilyScope,
) -> Result<Uuid, AppError> {
    // Verify purchase exists
    let purchase = self.purchases
        .get_by_family_and_listing(scope.family_id(), listing_id)
        .await?
        .ok_or(MktError::NotPurchased)?;

    // One review per purchase
    // (enforced by UNIQUE constraint on purchase_id, but check first for better error)

    let review_id = self.reviews.create(CreateReview {
        listing_id,
        purchase_id: purchase.id,
        family_id: scope.family_id(),
        rating: cmd.rating,
        review_text: cmd.review_text.clone(),
        is_anonymous: cmd.is_anonymous.unwrap_or(true),
    }).await?.id;

    // Update aggregate rating on listing
    let (avg, count) = self.reviews.get_aggregate_rating(listing_id).await?;
    // (listing rating_avg and rating_count updated in repository)

    // Publish event
    self.events.publish(ReviewCreated {
        review_id,
        listing_id,
        rating: cmd.rating,
    })?;

    // Send review text to safety:: for moderation
    if let Some(text) = &cmd.review_text {
        self.events.publish(ContentSubmittedForModeration {
            content_id: review_id,
            content_type: "marketplace_review".to_string(),
            text: text.clone(),
        })?;
    }

    Ok(review_id)
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

```rust
async fn get_creator_dashboard(
    &self,
    creator_id: Uuid,
    period: DashboardPeriod,
) -> Result<CreatorDashboardResponse, AppError> {
    let (from, to) = period.to_date_range();

    let all_time_sales = self.purchases.get_creator_sales(
        creator_id,
        DateTime::<Utc>::MIN_UTC,
        Utc::now(),
    ).await?;

    let period_sales = self.purchases.get_creator_sales(
        creator_id, from, to,
    ).await?;

    Ok(CreatorDashboardResponse {
        total_sales_count: all_time_sales.len() as i64,
        total_earnings_cents: all_time_sales.iter()
            .map(|s| s.creator_payout_cents as i64).sum(),
        period_sales_count: period_sales.len() as i64,
        period_earnings_cents: period_sales.iter()
            .map(|s| s.creator_payout_cents as i64).sum(),
        pending_payout_cents: 0, // Phase 2: calculated from unpaid earnings
        average_rating: /* aggregate from listings */,
        total_reviews: /* aggregate from listings */,
        recent_sales: period_sales.into_iter().take(10).map(|s| SaleSummary {
            purchase_id: s.purchase_id,
            listing_title: s.listing_title,
            amount_cents: s.amount_cents,
            creator_payout_cents: s.creator_payout_cents,
            purchased_at: s.created_at,
        }).collect(),
    })
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
3. Service calls `PaymentAdapter::create_payout()`
4. Hyperswitch routes payout to creator's connected bank account via underlying processor
5. `PayoutCompleted` webhook updates payout status

### 1099-K Compliance (Phase 2)

Tax reporting is offloaded to the underlying payment processor (Stripe) via Hyperswitch.
Stripe automatically generates and files 1099-K forms for creators meeting IRS thresholds.
The platform does not handle tax forms directly. `[S§9.6]`

---

## §16 RequireCreator Extractor (Domain Deep-Dive 8)

### Decision

**Dedicated extractor in `src/mkt/extractors.rs`** — does NOT add `creator_id` to `AuthContext`.

This resolves the open question from 00-core §13.3, which documented two approaches:

1. ~~Preferred (00-core): Add `creator_id: Option<Uuid>` to `AuthContext`~~ — **rejected**
2. **Chosen: Query `mkt::` service from the extractor via `AppState`**

### Rationale

1. **Shared kernel minimality** — `AuthContext` is in `src/shared/types.rs` and used by ALL
   domains. Adding `creator_id` (a marketplace-specific concept) to the shared kernel violates
   the bounded context principle. Only ~10% of requests are creator-related.

2. **No unnecessary DB overhead** — Adding `creator_id` to `AuthContext` means the auth
   middleware would need to LEFT JOIN `mkt_creators` on EVERY authenticated request, even
   though most requests don't need it.

3. **Redis caching eliminates performance concern** — The extractor queries `mkt_creators` by
   `parent_id`, which is cached in Redis with a short TTL (5 minutes). The cache hit rate for
   active creators will be near 100%.

4. **Domain ownership** — The creator lookup logic belongs in `mkt::`, not in the shared auth
   middleware. This keeps the auth middleware simple and domain-agnostic.

### Implementation

```rust
// src/mkt/extractors.rs

use axum::extract::{FromRequestParts, State};
use axum::http::request::Parts;

use crate::shared::error::AppError;
use crate::shared::types::AuthContext;

/// Extracts AuthContext and verifies the user has a creator account.
/// Returns 403 Forbidden if no creator account exists. [S§3.1.4]
///
/// Uses Redis caching to avoid DB query on every request.
/// Cache key: `mkt:creator:{parent_id}` with 5-minute TTL.
pub struct RequireCreator {
    pub auth: AuthContext,
    pub creator_id: Uuid,
}

#[axum::async_trait]
impl<S> FromRequestParts<S> for RequireCreator
where
    S: Send + Sync + AsRef<AppState>,
{
    type Rejection = AppError;

    async fn from_request_parts(
        parts: &mut Parts,
        state: &S,
    ) -> Result<Self, Self::Rejection> {
        let auth = parts
            .extensions
            .get::<AuthContext>()
            .cloned()
            .ok_or(AppError::Unauthorized)?;

        let app_state = state.as_ref();
        let cache_key = format!("mkt:creator:{}", auth.parent_id);

        // Try Redis cache first
        if let Some(creator_id) = app_state.redis
            .get::<Option<Uuid>>(&cache_key)
            .await?
        {
            return Ok(RequireCreator { auth, creator_id });
        }

        // Cache miss — query DB
        let creator = app_state.marketplace_service
            .get_creator_by_parent_id(auth.parent_id)
            .await?
            .ok_or(AppError::Forbidden)?;

        // Cache for 5 minutes
        app_state.redis
            .set_ex(&cache_key, &creator.id, 300)
            .await?;

        Ok(RequireCreator {
            auth,
            creator_id: creator.id,
        })
    }
}
```

### Usage in Handlers

```rust
// src/mkt/handlers.rs

pub async fn create_listing(
    RequireCreator { auth, creator_id }: RequireCreator,
    State(state): State<AppState>,
    Json(cmd): Json<CreateListingCommand>,
) -> Result<Json<ListingDetailResponse>, AppError> {
    let listing_id = state.marketplace_service
        .create_listing(cmd, creator_id)
        .await?;
    let listing = state.marketplace_service
        .get_listing(listing_id)
        .await?;
    Ok(Json(listing))
}

pub async fn get_creator_profile(
    RequireCreator { auth, creator_id }: RequireCreator,
    State(state): State<AppState>,
) -> Result<Json<CreatorResponse>, AppError> {
    let creator = state.marketplace_service
        .get_creator_by_parent_id(auth.parent_id)
        .await?
        .ok_or(AppError::NotFound)?;
    Ok(Json(creator))
}
```

---

## §17 Error Types

All marketplace errors use `thiserror` and map to HTTP status codes via `AppError`. Internal
details are logged but never exposed in API responses. `[CODING §5.2, S§18]`

```rust
// src/mkt/errors.rs

use thiserror::Error;

#[derive(Debug, Error)]
pub enum MktError {
    // ─── Creator Errors ─────────────────────────────────────────────────
    #[error("Creator account already exists for this user")]
    CreatorAlreadyExists,

    #[error("Creator account not found")]
    CreatorNotFound,

    #[error("Creator must accept Terms of Service")]
    TosNotAccepted,

    #[error("Creator onboarding not complete")]
    CreatorNotActive,

    #[error("Creator account is suspended")]
    CreatorSuspended,

    // ─── Publisher Errors ───────────────────────────────────────────────
    #[error("Publisher not found")]
    PublisherNotFound,

    #[error("Publisher slug already taken")]
    PublisherSlugConflict,

    #[error("Not a member of this publisher")]
    NotPublisherMember,

    #[error("Insufficient publisher role for this action")]
    InsufficientPublisherRole,

    #[error("Cannot remove the last owner of a publisher")]
    CannotRemoveLastOwner,

    #[error("Cannot modify the platform publisher")]
    CannotModifyPlatformPublisher,

    // ─── Listing Errors ─────────────────────────────────────────────────
    #[error("Listing not found")]
    ListingNotFound,

    #[error("Listing is not published")]
    ListingNotPublished,

    #[error("Listing is not free")]
    ListingNotFree,

    #[error("Invalid listing state transition: {from} → {action}")]
    InvalidStateTransition { from: String, action: String },

    #[error("Listing has no files attached")]
    ListingHasNoFiles,

    #[error("Not the owner of this listing")]
    NotListingOwner,

    #[error("Invalid content type")]
    InvalidContentType,

    #[error("Invalid price: must be >= 0")]
    InvalidPrice,

    // ─── File Errors ────────────────────────────────────────────────────
    #[error("File not found")]
    FileNotFound,

    #[error("Invalid file type")]
    InvalidFileType,

    #[error("File too large")]
    FileTooLarge,

    // ─── Cart Errors ────────────────────────────────────────────────────
    #[error("Item already in cart")]
    AlreadyInCart,

    #[error("Item not in cart")]
    NotInCart,

    #[error("Cart is empty")]
    EmptyCart,

    #[error("Cart contains unpublished listings")]
    StaleCart,

    // ─── Purchase Errors ────────────────────────────────────────────────
    #[error("Already purchased this listing")]
    AlreadyPurchased,

    #[error("Purchase not found")]
    PurchaseNotFound,

    #[error("Not purchased — cannot download")]
    NotPurchased,

    #[error("Refund window has expired (30 days)")]
    RefundWindowExpired,

    #[error("Purchase already refunded")]
    AlreadyRefunded,

    // ─── Review Errors ──────────────────────────────────────────────────
    #[error("Already reviewed this purchase")]
    AlreadyReviewed,

    #[error("Review not found")]
    ReviewNotFound,

    #[error("Not the owner of this review")]
    NotReviewOwner,

    #[error("Invalid rating: must be between 1 and 5")]
    InvalidRating,

    // ─── Payment Errors (processor-agnostic) ────────────────────────────
    #[error("Payment provider unavailable")]
    PaymentProviderUnavailable,

    #[error("Payment creation failed")]
    PaymentCreationFailed,

    #[error("Invalid webhook signature")]
    InvalidWebhookSignature,

    #[error("Webhook payload malformed")]
    MalformedWebhookPayload,

    #[error("Payout threshold not met")]
    PayoutThresholdNotMet,

    // ─── Infrastructure ─────────────────────────────────────────────────
    #[error("Database error")]
    DatabaseError(#[from] sea_orm::DbErr),

    #[error("Cache error")]
    CacheError(String),
}
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
| `MarketplaceService` trait methods | `learn::`, `onboard::` (Phase 2) | `Arc<dyn MarketplaceService>` via AppState |
| `verify_publisher_membership()` | `learn::` | Service method — publisher ownership checks `[06-learn §18.2]` |
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

Defined in `src/mkt/events.rs`. `[CODING §8.4]`

```rust
// src/mkt/events.rs

use crate::shared::types::FamilyId;
use chrono::{DateTime, Utc};
use uuid::Uuid;

/// Published when a family completes a marketplace purchase.
/// Consumed by learn:: (tool access), billing:: (creator earnings), notify:: (receipt).
///
/// IMPORTANT: The `content_metadata` field shape is a cross-domain contract
/// consumed by learn::event_handlers.rs [06-learn:3171]. Changes to this
/// struct require coordinated updates to all consumers.
#[derive(Clone, Debug)]
pub struct PurchaseCompleted {
    pub family_id: FamilyId,
    pub purchase_id: Uuid,
    pub listing_id: Uuid,
    pub content_metadata: PurchaseMetadata,
}
impl DomainEvent for PurchaseCompleted {}

/// Metadata about purchased content. Defined authoritatively by mkt::events
/// and imported by learn:: [06-learn:2532].
#[derive(Clone, Debug)]
pub struct PurchaseMetadata {
    pub content_type: String,
    pub content_ids: Vec<Uuid>,
    pub publisher_id: Uuid,
}

/// Published when a listing transitions to Published state.
/// Consumed by search:: (index update), ai:: (recommendation catalog).
#[derive(Clone, Debug)]
pub struct ListingPublished {
    pub listing_id: Uuid,
    pub publisher_id: Uuid,
    pub content_type: String,
    pub subject_tags: Vec<String>,
}
impl DomainEvent for ListingPublished {}

/// Published when a listing is archived.
/// Consumed by search:: (remove from index).
#[derive(Clone, Debug)]
pub struct ListingArchived {
    pub listing_id: Uuid,
}
impl DomainEvent for ListingArchived {}

/// Published when a verified-purchaser review is created.
/// Consumed by safety:: (content moderation scan).
#[derive(Clone, Debug)]
pub struct ReviewCreated {
    pub review_id: Uuid,
    pub listing_id: Uuid,
    pub rating: i16,
    pub review_text: Option<String>,  // for safety:: text scanning [11-safety §11.2]
}
impl DomainEvent for ReviewCreated {}

/// Published when a creator completes registration.
/// Consumed by notify:: (welcome email).
#[derive(Clone, Debug)]
pub struct CreatorOnboarded {
    pub creator_id: Uuid,
    pub parent_id: Uuid,
    pub store_name: String,
}
impl DomainEvent for CreatorOnboarded {}

/// Published when a purchase is refunded.
/// Consumed by billing:: (earnings adjustment), notify:: (refund notification).
#[derive(Clone, Debug)]
pub struct PurchaseRefunded {
    pub purchase_id: Uuid,
    pub listing_id: Uuid,
    pub family_id: FamilyId,
    pub refund_amount_cents: i64,
}
impl DomainEvent for PurchaseRefunded {}

/// Published when a listing is submitted for content screening.
/// Consumed by safety:: (automated content screening).
#[derive(Clone, Debug)]
pub struct ListingSubmitted {
    pub listing_id: Uuid,
    pub creator_id: Uuid,
}
impl DomainEvent for ListingSubmitted {}
```

### §18.4 Events mkt:: Subscribes To

| Event | Source | Effect |
|-------|--------|--------|
| `ContentFlagged { content_id, content_type, action }` | `safety::` | If `content_type == "listing"`: archive listing. If `"review"`: reject review. |
| `FamilyDeletionScheduled { family_id, delete_after }` | `iam::` | Anonymize reviews (retain ratings, clear text), retain purchase records (legal requirement), clear cart items. |
| `MethodologyConfigUpdated` | `method::` | Invalidate cached methodology tag display names used in listing browse responses. |

```rust
// src/mkt/event_handlers.rs

use crate::safety::events::ContentFlagged;
use crate::iam::events::FamilyDeletionScheduled;
use crate::method::events::MethodologyConfigUpdated;

pub struct ContentFlaggedHandler {
    marketplace_service: Arc<dyn MarketplaceService>,
}

#[async_trait]
impl DomainEventHandler<ContentFlagged> for ContentFlaggedHandler {
    async fn handle(&self, event: &ContentFlagged) -> Result<(), AppError> {
        self.marketplace_service.handle_content_flagged(
            event.content_id,
            event.action.clone(),
        ).await
    }
}

pub struct FamilyDeletionScheduledHandler {
    marketplace_service: Arc<dyn MarketplaceService>,
}

#[async_trait]
impl DomainEventHandler<FamilyDeletionScheduled> for FamilyDeletionScheduledHandler {
    async fn handle(&self, event: &FamilyDeletionScheduled) -> Result<(), AppError> {
        self.marketplace_service.handle_family_deletion_scheduled(
            event.family_id,
        ).await
    }
}

pub struct MethodologyConfigUpdatedHandler {
    // Invalidate Redis cache for methodology tag display names
    redis: Arc<RedisPool>,
}

#[async_trait]
impl DomainEventHandler<MethodologyConfigUpdated> for MethodologyConfigUpdatedHandler {
    async fn handle(&self, _event: &MethodologyConfigUpdated) -> Result<(), AppError> {
        self.redis.del_pattern("mkt:methodology_tags:*").await?;
        Ok(())
    }
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
│  │  (Axum server)   │     │  :8080                       │  │
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
4. `POST /v1/marketplace/creators/onboarding-link` calls `PaymentAdapter::create_onboarding_link()`
5. Creator profile update validates `store_name` length (1-100)
6. `CreatorOnboarded` event published on registration

### Publisher Management

7. `POST /v1/marketplace/publishers` creates publisher and `owner` membership for creator
8. Duplicate slug returns `409 Conflict`
9. Only `owner`/`admin` can update publisher details
10. `verify_publisher_membership()` returns true for members, false for non-members
11. Platform publisher (`is_platform = true`) cannot be modified via API
12. Platform publisher seed data exists after migration

### Listing Lifecycle

13. New listing starts in `draft` status
14. `submit()` fails if listing has no files (returns `ListingHasNoFiles`)
15. `submit()` fails from non-`draft` status (returns `InvalidStateTransition`)
16. `publish()` only succeeds from `submitted` status
17. `publish()` sets `published_at` timestamp
18. `archive()` only succeeds from `published` status
19. `archive()` sets `archived_at` timestamp; `published_at` retained
20. Updating a published listing creates a version snapshot in `mkt_listing_versions`
21. Updating a published listing increments the `version` field
22. Creator must be member of the listing's publisher to create/edit

### Purchase Flow

23. Adding to cart fails if listing not published (`404`)
24. Adding to cart fails if already purchased (`409`)
25. Cart is family-scoped: Parent A's additions visible to Parent B
26. `POST /cart/checkout` calls `PaymentAdapter::create_payment()` with correct split rules
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

48. `PurchaseCompleted` event shape matches learn::event_handlers expectation: `{ family_id: FamilyId, content_metadata: PurchaseMetadata }`
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
src/mkt/
├── mod.rs                    # Re-exports, domain-level doc comments
├── handlers.rs               # Axum route handlers (thin layer only)
├── service.rs                # MarketplaceServiceImpl — orchestration
├── repository.rs             # PgCreatorRepository, PgPublisherRepository,
│                             # PgListingRepository, PgListingFileRepository,
│                             # PgCartRepository, PgPurchaseRepository,
│                             # PgReviewRepository, PgCuratedSectionRepository
├── models.rs                 # Request/response types, internal types, query params
├── ports.rs                  # MarketplaceService trait, all repository traits,
│                             # PaymentAdapter trait, MediaAdapter trait
├── errors.rs                 # MktError thiserror enum
├── events.rs                 # PurchaseCompleted, ListingPublished, ListingArchived,
│                             # ReviewCreated, CreatorOnboarded, PurchaseRefunded,
│                             # ListingSubmitted
├── event_handlers.rs         # ContentFlaggedHandler, FamilyDeletionScheduledHandler,
│                             # MethodologyConfigUpdatedHandler
├── extractors.rs             # RequireCreator extractor [00-core §13.3 resolution]
├── adapters/
│   ├── mod.rs
│   └── payment.rs            # HyperswitchPaymentAdapter — wraps Hyperswitch REST API,
│                             # returns domain types only [supersedes ADR-007]
├── domain/
│   ├── mod.rs
│   ├── listing.rs            # MarketplaceListing aggregate root — state machine
│   ├── value_objects.rs      # Price, ListingTitle, etc.
│   └── errors.rs             # MktDomainError enum
└── entities/                 # SeaORM-generated — never hand-edit [CODING §6.3]
```
