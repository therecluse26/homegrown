# Domain Spec 12 — Search (search::)

## §1 Overview

The Search domain is a **read-only cross-cutting query system** — it provides full-text search,
autocomplete, and faceted filtering across three scopes (social, marketplace, learning), each
with a distinct privacy enforcement model. Search does not own content data; it reads
`search_vector` columns and GIN indexes maintained on domain tables via PostgreSQL's `GENERATED
ALWAYS AS ... STORED` columns. In Phase 1, all search is PostgreSQL FTS. In Phase 2+, marketplace
and social search migrate to Typesense while learning search remains on PostgreSQL (smaller
dataset, strict family scoping). `[S§14, S§9.3, V§7, V§8, V§9]`

| Attribute | Value |
|-----------|-------|
| **Module path** | `internal/search/` |
| **DB prefix** | `search_` (Phase 2 only — no owned tables in Phase 1) |
| **Complexity class** | Simple (no `domain/` subdirectory) — indexing and retrieval, no business invariants `[ARCH §4.5]` |
| **CQRS** | Yes — command side (index updates, Phase 1 no-ops) / query side (search, autocomplete) `[ARCH §4.7]` |
| **External adapter** | `internal/search/adapters/typesense.go` (Phase 2 — Typesense search engine) |
| **Key constraint** | Social search MUST enforce friendship + discovery visibility; learning search MUST be family-scoped; cross-family learning data MUST NOT be searchable `[S§14.2]` |

**What search:: owns**: Search query routing (scope dispatch, backend selection), autocomplete
logic (`pg_trgm` for marketplace, `ILIKE` prefix for social), faceted filter construction for
marketplace, privacy enforcement query decorations (friendship checks, block exclusion, family
scoping), Phase 2 search index state tracking (`search_index_state`), Phase 2 Typesense adapter,
background jobs (`IndexContentJob`, `BulkIndexJob` — Phase 2).

**What search:: does NOT own**: Source data (owned by `social::`, `mkt::`, `learn::`), search
vector columns and GIN indexes (owned by source domain schemas — `soc_posts.search_vector`,
`mkt_listings.search_vector`, `learn_activity_logs.search_vector`,
`learn_journal_entries.search_vector`), friendship/block relationships (owned by `social::`),
listing status lifecycle (owned by `mkt::`), family membership and `FamilyScope` (owned by
`iam::`), curated sections and discovery browsing (owned by `mkt::` and `social::`).

**What search:: delegates**: Friendship status checks → reads `soc_friendships` directly (no
service call — pure SQL JOIN). Block checks → reads `soc_blocks` directly. Family scoping →
`FamilyScope` extractor from `00-core`. Methodology display name resolution → reads
`method_definitions` directly. Background job scheduling → asynq `[ARCH §12]`. Phase 2
search engine → Typesense client via `TypesenseAdapter`.

---

## §2 Requirements Traceability

Every SPEC.md §14 and §9.3 requirement maps to a section in this document.

| Requirement | SPEC Reference | Domain Spec Section |
|-------------|---------------|---------------------|
| Social search scope: users by name, groups by name/description, events by title/description/location | `[S§14.1]` | §10.1 (social search SQL) |
| Learning search scope: activities, journal entries, reading lists (family-scoped) | `[S§14.1]` | §10.3 (learning search SQL) |
| Marketplace search scope: listings by title, description, creator, tags | `[S§14.1]` | §10.2 (marketplace search SQL) |
| Search results < 500ms p95 | `[S§14.2, S§17.3]` | §4 (rate limiting), §13 (Typesense migration trigger) |
| Autocomplete / type-ahead suggestions | `[S§14.2]` | §11 (autocomplete deep dive) |
| Marketplace faceted filtering (methodology, subject, grade, price, rating, content type, worldview) | `[S§14.2, S§9.3]` | §8.1 (`MarketplaceSearchFilters`), §10.2 (facet SQL) |
| Social search respects privacy — friends OR discovery opt-in only | `[S§14.2]` | §10.1 (privacy SQL), §12 (privacy enforcement) |
| Learning search scoped to authenticated family — no cross-family searchable | `[S§14.2]` | §10.3 (`FamilyScope`), §12 (privacy enforcement) |
| Methodology-scoped Explore sections (social + marketplace) | `[S§14.3]` | §10.1 (methodology filter), §10.2 (methodology facet) |
| Discovery respects all privacy settings | `[S§14.3]` | §12 (privacy enforcement) |
| Marketplace full-text search on titles and descriptions | `[S§9.3]` | §10.2 (weighted `search_vector`) |
| Marketplace curated sections | `[S§9.3]` | Out of scope — owned by `mkt::` (`mkt_curated_sections`) |
| Marketplace sort by relevance, price, rating, recency | `[S§9.3]` | §8.1 (`SearchSortOrder`), §10.2 (ORDER BY) |
| Search indexing decoupled from primary data storage | `[S§17.4]` | §9 (event handlers), §13 (Typesense migration) |

> **Coverage note on `[S§14.3]` discovery**: SPEC.md §14.3 specifies methodology-scoped "Explore"
> sections. The search domain provides the **query infrastructure** for these (methodology-filtered
> search and browse), but the Explore page composition and curated sections are owned by `mkt::`
> (marketplace browse — `07-mkt.md §13`) and `social::` (social discovery — `05-social.md §15`).
> Search provides the full-text search endpoints that Explore pages use, not the Explore page
> layout itself.

> **Note on `[S§9.3]` sort options**: The `sort` query parameter on marketplace search supports
> `relevance` (default, `ts_rank`-based), `price_asc`, `price_desc`, `rating` (`rating_avg DESC`),
> and `recency` (`published_at DESC`). These are implemented as conditional `ORDER BY` clauses
> in §10.2.

---

## §3 Database Schema

The search domain has **no owned tables in Phase 1**. All full-text search is performed against
`search_vector` columns and GIN indexes defined on source domain tables. Phase 2 introduces a
single tracking table for Typesense index synchronization state.

### §3.1 External FTS Index Inventory

These columns and indexes are defined by their owning domain specs and are **read** by `search::`.
Listed here for reference only — search does not create or modify these.

| Source Table | Owner Spec | `search_vector` Definition | GIN Index |
|-------------|-----------|---------------------------|-----------|
| `soc_posts` | `05-social.md §3.2` | `GENERATED ALWAYS AS (to_tsvector('english', coalesce(content, ''))) STORED` | `idx_soc_posts_search` |
| `soc_groups` | `ARCH §9.1` ¹ | _(index-only, no stored column)_ | `idx_soc_groups_search` on `to_tsvector('english', coalesce(name, '') \|\| ' ' \|\| coalesce(description, ''))` |
| `soc_events` | `ARCH §9.1` ¹ | _(index-only, no stored column)_ | `idx_soc_events_search` on `to_tsvector('english', coalesce(title, '') \|\| ' ' \|\| coalesce(description, ''))` |
| `mkt_listings` | `07-mkt.md §3.2` | `GENERATED ALWAYS AS (setweight(to_tsvector('english', coalesce(title, '')), 'A') \|\| setweight(to_tsvector('english', coalesce(description, '')), 'B')) STORED` | `idx_mkt_listings_search` |
| `mkt_listings.title` | `07-mkt.md §3.2` | _(trigram, no tsvector)_ | `idx_mkt_listings_title_trgm` on `title gin_trgm_ops` |
| `learn_activity_logs` | `06-learn.md §3.2` | `GENERATED ALWAYS AS (to_tsvector('english', coalesce(title, '') \|\| ' ' \|\| coalesce(description, ''))) STORED` | `idx_learn_activity_logs_search` |
| `learn_journal_entries` | `06-learn.md §3.2` | `GENERATED ALWAYS AS (to_tsvector('english', coalesce(title, '') \|\| ' ' \|\| coalesce(content, ''))) STORED` | `idx_learn_journal_entries_search` |
| `learn_reading_items` | `06-learn.md §3.2` | `GENERATED ALWAYS AS (to_tsvector('english', coalesce(title, '') \|\| ' ' \|\| coalesce(author, '') \|\| ' ' \|\| coalesce(description, ''))) STORED` | `idx_learn_reading_items_search` |

> ¹ GIN indexes on `soc_groups` and `soc_events` are defined in migration 009 (`idx_soc_groups_search`,
> `idx_soc_events_search`) as expression indexes on `to_tsvector(...)`. These are used by the
> `websearch_to_tsquery` expressions in §10.1 group/event search queries.

**Additional indexes read by search (faceted filtering):**

| Index | Source Table | Type | Purpose |
|-------|-------------|------|---------|
| `idx_mkt_listings_methodology` | `mkt_listings` | GIN on `methodology_tags` | Methodology facet filter |
| `idx_mkt_listings_subject` | `mkt_listings` | GIN on `subject_tags` | Subject facet filter |
| `idx_mkt_listings_worldview` | `mkt_listings` | GIN on `worldview_tags` | Worldview facet filter |

**Privacy-enforcement tables read by search (no ownership):**

| Table | Owner Spec | Purpose in Search |
|-------|-----------|-------------------|
| `soc_friendships` | `05-social.md §3.2` | Social search: include friends (status = 'accepted') |
| `soc_blocks` | `05-social.md §3.2` | Social search: exclude blocked families bidirectionally |
| `soc_profiles` | `05-social.md §3.2` | Social search: check `location_visible` for discovery opt-in |
| `soc_group_members` | `05-social.md §3.2` | Social search: filter group-visibility events to group members |
| `iam_families` | `01-iam.md §3.2` | Social search: `display_name` for family name search |
| `learn_reading_progress` | `06-learn.md §3.2` | Learning search: family-scoped reading item access via JOIN |
| `method_definitions` | `02-method.md §3.2` | Methodology display name resolution for search results |

### §3.2 Tables

#### `search_index_state` — Typesense Sync Tracking (Phase 2)

```sql
-- ═══════════════════════════════════════════════════════════════════════════════
-- TABLE 1: search_index_state — Typesense index sync tracking (Phase 2)
-- ═══════════════════════════════════════════════════════════════════════════════
-- Tracks the last-synced position for each Typesense collection to enable
-- incremental indexing. Created in Phase 1 migration but unused until Phase 2.
-- ═══════════════════════════════════════════════════════════════════════════════
CREATE TABLE search_index_state (
    id              UUID        PRIMARY KEY DEFAULT uuidv7(),
    collection_name TEXT        NOT NULL UNIQUE
                    CHECK (collection_name IN (
                        'marketplace_listings', 'social_posts',
                        'social_groups', 'social_events'
                    )),
    last_synced_at  TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01T00:00:00Z',
    last_synced_id  UUID,                   -- last processed entity ID (cursor for incremental sync)
    document_count  BIGINT      NOT NULL DEFAULT 0,
    status          TEXT        NOT NULL DEFAULT 'inactive'
                    CHECK (status IN ('inactive', 'syncing', 'active', 'error')),
    error_message   TEXT,                   -- last error details (internal only, never exposed in API)
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### §3.3 RLS Policies

```sql
-- search_index_state: system role only (background jobs manage sync state).
-- Phase 2 table — RLS defined in Phase 1 migration for forward compatibility.
ALTER TABLE search_index_state ENABLE ROW LEVEL SECURITY;

CREATE POLICY search_index_state_system_all
    ON search_index_state FOR ALL
    WITH CHECK (current_setting('app.role') = 'system');
```

> **Note on search query RLS**: Search queries against source domain tables (`soc_posts`,
> `mkt_listings`, `learn_activity_logs`, etc.) are subject to the RLS policies defined in
> their owning domain specs. However, search queries execute with the `system` role and apply
> privacy filtering **in application SQL** (not via RLS) because cross-family reads are
> required for social and marketplace scopes. Learning search queries set
> `app.family_id` and rely on the owning domain's RLS policies for defense-in-depth.

---

## §4 API Endpoints

All endpoints are prefixed with `/v1/search`. Auth requirements use extractors defined in
`00-core §13`: `AuthContext` for authenticated users, `FamilyScope` for family-scoped data
access. `[CODING §2.1]`

**Rate limit**: 60 req/min per authenticated user `[ARCH §10.6]`

### §4.1 Phase 1 (2 endpoints)

#### `GET /v1/search`

Unified search across social, marketplace, or learning content.

- **Auth**: `AuthContext` (required for all scopes); `FamilyScope` (required, used by learning scope and social privacy enforcement)
- **Query**: `SearchParams { Q string, Scope SearchScope, Cursor *string, Limit *int (default 20, max 50), Sort *SearchSortOrder, filters: scope-specific }`
- **Scope-specific query parameters**:
  - `scope=social`: `sub_scope?: SocialSubScope (families|groups|events)`, `methodology_slug?: string`
  - `scope=marketplace`: `methodology_tags?: []uuid.UUID`, `subject_tags?: []string`, `grade_min?: *int16`, `grade_max?: *int16`, `price_max?: *int32`, `price_min?: *int32 (default 0)`, `content_type?: *string`, `worldview_tags?: []string`, `free_only?: *bool`
  - `scope=learning`: `student_id?: *uuid.UUID`, `source_type?: LearningSourceType (activity|journal|reading)`, `date_from?: *time.Time`, `date_to?: *time.Time`, `subject_tags?: []string`
- **Response**: `200 OK` → `SearchResponse { Results []SearchResult, TotalCount int64, Facets *FacetCounts, NextCursor *string }`
- **Pagination**: Cursor-based on `(rank DESC, id)` for stable ordering across pages
- **Sort options** (marketplace only): `relevance` (default), `price_asc`, `price_desc`, `rating`, `recency`
- **Error codes**: `400` (missing/invalid q, invalid scope, invalid filter values), `401` (unauthenticated), `422` (q too short — minimum 2 characters)

> **Design note**: A single unified endpoint with `scope` parameter was chosen over three
> separate endpoints (`/v1/search/social`, `/v1/search/marketplace`, `/v1/search/learning`)
> because: (1) the frontend search UI is a single input with a scope selector, (2) shared
> infrastructure (rate limiting, pagination, error handling) is DRY, (3) future cross-scope
> search can be added without new endpoints.

#### `GET /v1/search/autocomplete`

Type-ahead suggestions for search input. Must return within 200ms.

- **Auth**: `AuthContext` + `FamilyScope`
- **Query**: `AutocompleteParams { Q string, Scope *SearchScope (default: inferred from context), Limit *int (default 5, max 10) }`
- **Response**: `200 OK` → `AutocompleteResponse { Suggestions []AutocompleteSuggestion }`
- **Implementation**:
  - Marketplace: `pg_trgm` `similarity()` on `mkt_listings.title` (fuzzy)
  - Social: `ILIKE` prefix match on `iam_families.display_name` and `soc_groups.name` (exact prefix)
  - Learning: `ILIKE` prefix match on `learn_activity_logs.title` and `learn_journal_entries.title` (family-scoped)
- **Error codes**: `400` (q too short — minimum 1 character), `401` (unauthenticated)

### §4.2 Phase 2 (0 new endpoints)

No new endpoints in Phase 2. The existing endpoints gain Typesense-backed performance for
social and marketplace scopes via the `SearchBackend` enum (§13). The API contract is unchanged.

### §4.3 Phase 3+ (1 new endpoint)

#### `GET /v1/search/suggestions`

AI-powered search suggestions based on family's methodology and past activity. Depends on
`recs::` domain.

- **Auth**: `AuthContext` + `FamilyScope`
- **Response**: `200 OK` → `SearchSuggestionsResponse { Suggestions []SearchSuggestion }`

---

## §5 Service Interface

The search service follows lightweight CQRS: command side handles index update events (Phase 1
no-ops), query side handles search and autocomplete requests. `[ARCH §4.7]`

```go
// SearchService — query side (search + autocomplete) and command side (index updates).
//
// Phase 1: All queries use PostgreSQL FTS. Command-side methods are no-ops because
// GENERATED ALWAYS columns auto-update search vectors on source table writes.
//
// Phase 2+: SearchBackend enum routes marketplace and social queries to Typesense
// when enabled. Command-side methods trigger Typesense index updates via IndexContentJob.
// Learning search always stays on PostgreSQL FTS.
type SearchService interface {
    // ── Query Side ─────────────────────────────────────────────────────────

    // Search performs unified search across a single scope with privacy enforcement.
    // Dispatches to scope-specific repository based on params.Scope.
    Search(ctx context.Context, auth *AuthContext, scope *FamilyScope, params *SearchParams) (*SearchResponse, error)

    // Autocomplete returns type-ahead suggestions.
    // Returns within 200ms target. Scope-specific implementation.
    Autocomplete(ctx context.Context, auth *AuthContext, scope *FamilyScope, params *AutocompleteParams) (*AutocompleteResponse, error)

    // ── Command Side (Index Updates) ───────────────────────────────────────

    // HandlePostCreated handles new post creation — update social search index.
    // Phase 1: no-op (GENERATED ALWAYS column auto-updates).
    // Phase 2: enqueues IndexContentJob for Typesense.
    HandlePostCreated(ctx context.Context, event *PostCreated) error

    // HandleListingPublished handles listing publication — add/update marketplace search index.
    // Phase 1: no-op (GENERATED ALWAYS column auto-updates).
    // Phase 2: enqueues IndexContentJob for Typesense.
    HandleListingPublished(ctx context.Context, event *ListingPublished) error

    // HandleListingArchived handles listing archival — remove from marketplace search index.
    // Phase 1: no-op (listing status change excludes from WHERE status = 'published').
    // Phase 2: removes document from Typesense collection.
    HandleListingArchived(ctx context.Context, event *ListingArchived) error

    // HandleUploadPublished handles media upload publication — index media metadata.
    // Phase 1: no-op (media is not directly searchable in Phase 1).
    // Phase 2: indexes media metadata into relevant Typesense collection.
    HandleUploadPublished(ctx context.Context, event *UploadPublished) error

    // HandleFamilyDeletionScheduled handles family deletion — remove family data from all search indexes.
    // Phase 1: no-op (source table CASCADE DELETE removes rows, GENERATED columns follow).
    // Phase 2: removes all family-related documents from Typesense collections.
    HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error
}
```

### SearchServiceImpl

```go
type SearchServiceImpl struct {
    socialRepo      SocialSearchRepository
    marketplaceRepo MarketplaceSearchRepository
    learningRepo    LearningSearchRepository
    autocompleteRepo AutocompleteRepository
    backend         SearchBackend          // Phase 1: always PostgresFts
    // Phase 2 additions:
    // typesense TypesenseAdapter
    // jobQueue  asynq.Client
}

func NewSearchService(
    socialRepo SocialSearchRepository,
    marketplaceRepo MarketplaceSearchRepository,
    learningRepo LearningSearchRepository,
    autocompleteRepo AutocompleteRepository,
) *SearchServiceImpl {
    return &SearchServiceImpl{
        socialRepo:       socialRepo,
        marketplaceRepo:  marketplaceRepo,
        learningRepo:     learningRepo,
        autocompleteRepo: autocompleteRepo,
        backend:          SearchBackendPostgresFts,
    }
}
```

---

## §6 Repository Interfaces

Four repositories, one per search scope plus autocomplete. All repositories are query-only
(no mutations) in Phase 1. Repositories accept privacy context as parameters — they do not
trust callers to pre-filter.

### §6.1 `SocialSearchRepository`

```go
// SocialSearchRepository queries soc_posts, soc_groups, soc_events, iam_families.
// All queries enforce friendship + discovery visibility and block exclusion.
type SocialSearchRepository interface {
    // SearchFamilies searches families by display name.
    // Returns only: (1) friends (soc_friendships.status = 'accepted'), or
    // (2) families with soc_profiles.location_visible = true (discovery opt-in).
    // Excludes: blocked families (bidirectional soc_blocks check).
    SearchFamilies(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error)

    // SearchGroups searches groups by name and description.
    // Returns all non-private groups (searchable by any authenticated user).
    // Block exclusion: groups created by blocked families are excluded.
    SearchGroups(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologySlug *string, limit int, cursor *string) ([]SocialSearchResult, error)

    // SearchEvents searches events by title, description, and location.
    // Visibility enforcement:
    // - 'discoverable': visible to all authenticated users
    // - 'friends': visible only to friends of the creator
    // - 'group': visible only to members of the associated group
    // Excludes: events by blocked families, cancelled events.
    SearchEvents(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologySlug *string, limit int, cursor *string) ([]SocialSearchResult, error)

    // SearchPosts searches posts by content.
    // Privacy: friends-only posts visible to friends, group posts visible to group members.
    // Block exclusion: posts by blocked families are excluded bidirectionally.
    SearchPosts(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error)
}
```

### §6.2 `MarketplaceSearchRepository`

```go
// MarketplaceSearchRepository queries mkt_listings with faceted filtering.
// Only returns listings with status = 'published'. No family scoping needed.
type MarketplaceSearchRepository interface {
    // SearchListings performs full-text search with faceted filtering and sorting.
    // Uses weighted search_vector (title = 'A', description = 'B') for relevance ranking.
    SearchListings(ctx context.Context, query string, filters *MarketplaceSearchFilters, sort SearchSortOrder, limit int, cursor *string) (*MarketplaceSearchResults, error)

    // CountFacets counts listings per facet value for the current query + filters.
    // Returns counts for: methodology_tags, subject_tags, content_type,
    // worldview_tags, price_ranges, rating_ranges.
    CountFacets(ctx context.Context, query string, filters *MarketplaceSearchFilters) (*FacetCounts, error)
}
```

### §6.3 `LearningSearchRepository`

```go
// LearningSearchRepository queries learn_activity_logs, learn_journal_entries,
// learn_reading_items (via learn_reading_progress JOIN).
// ALWAYS family-scoped via FamilyScope. NEVER uses Typesense, even in Phase 2.
type LearningSearchRepository interface {
    // SearchLearning searches family's own learning data across activities, journals, and reading items.
    // UNION ALL across multiple tables, always filtered by family_id.
    // Optional filters: student_id, source_type, date range, subject_tags.
    SearchLearning(ctx context.Context, familyScope *FamilyScope, query string, filters *LearningSearchFilters, limit int, cursor *string) ([]LearningSearchResult, error)
}
```

### §6.4 `AutocompleteRepository`

```go
// AutocompleteRepository provides fast prefix/fuzzy matching for type-ahead.
// Target latency: < 200ms.
type AutocompleteRepository interface {
    // AutocompleteMarketplace uses pg_trgm similarity on listing titles.
    // Only searches published listings.
    AutocompleteMarketplace(ctx context.Context, query string, limit int) ([]AutocompleteSuggestion, error)

    // AutocompleteSocial uses ILIKE prefix match on family display names
    // and group names. Respects friendship/discovery visibility and blocks.
    AutocompleteSocial(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int) ([]AutocompleteSuggestion, error)

    // AutocompleteLearning uses ILIKE prefix match on activity and journal titles.
    // Always family-scoped.
    AutocompleteLearning(ctx context.Context, familyScope *FamilyScope, query string, limit int) ([]AutocompleteSuggestion, error)
}
```

---

## §7 Adapter Interface

### §7.1 Phase 2: `TypesenseAdapter`

```go
// TypesenseAdapter is a Typesense search engine adapter for Phase 2+ migration.
// Provides high-performance search with typo tolerance, faceted filtering,
// and built-in Raft-based HA clustering.
//
// Typesense replaces Meilisearch (originally specified in ARCH §2.6) because:
// - Built-in Raft-based HA clustering in the open-source edition
//   (Meilisearch requires Enterprise for HA)
// - Sub-50ms search latency (C++ engine), field weighting for relevance tuning
// - Production-proven clustering (3 or 5 node) without enterprise licensing
// - Better fit for a solo developer running a COPPA-regulated platform
type TypesenseAdapter interface {
    // IndexDocument indexes a single document into a collection.
    IndexDocument(ctx context.Context, collection string, document map[string]any) error

    // RemoveDocument removes a document from a collection by ID.
    RemoveDocument(ctx context.Context, collection string, documentID string) error

    // BulkIndex bulk indexes multiple documents (used by BulkIndexJob).
    BulkIndex(ctx context.Context, collection string, documents []map[string]any) (*BulkIndexResult, error)

    // Search executes a search query against a collection.
    Search(ctx context.Context, collection string, query *TypesenseSearchQuery) (*TypesenseSearchResponse, error)

    // CollectionStats gets collection health/stats (used for monitoring).
    CollectionStats(ctx context.Context, collection string) (*CollectionStats, error)
}
```

---

## §8 Models (DTOs)

### §8.1 Request Types

```go
// SearchScope determines which repository and privacy model to use.
type SearchScope string

const (
    SearchScopeSocial      SearchScope = "social"
    SearchScopeMarketplace SearchScope = "marketplace"
    SearchScopeLearning    SearchScope = "learning"
)

// SocialSubScope optionally narrows social search to a specific entity type.
type SocialSubScope string

const (
    SocialSubScopeFamilies SocialSubScope = "families"
    SocialSubScopeGroups   SocialSubScope = "groups"
    SocialSubScopeEvents   SocialSubScope = "events"
)

// LearningSourceType optionally narrows learning search to a specific source.
type LearningSourceType string

const (
    LearningSourceTypeActivity LearningSourceType = "activity"
    LearningSourceTypeJournal  LearningSourceType = "journal"
    LearningSourceTypeReading  LearningSourceType = "reading"
)

// SearchSortOrder defines sort order for marketplace search results.
type SearchSortOrder string

const (
    SearchSortRelevance SearchSortOrder = "relevance"
    SearchSortPriceAsc  SearchSortOrder = "price_asc"
    SearchSortPriceDesc SearchSortOrder = "price_desc"
    SearchSortRating    SearchSortOrder = "rating"
    SearchSortRecency   SearchSortOrder = "recency"
)

// SearchParams is the unified search request parameters.
// Deserialized from query string on GET /v1/search.
type SearchParams struct {
    // Q is the search query text. Minimum 2 characters.
    Q     string      `json:"q" query:"q" validate:"required,min=2"`
    // Scope determines which scope to search within.
    Scope SearchScope `json:"scope" query:"scope" validate:"required,oneof=social marketplace learning"`
    // Cursor for pagination (opaque string from previous response).
    Cursor *string    `json:"cursor,omitempty" query:"cursor"`
    // Limit is results per page (default 20, max 50).
    Limit  int        `json:"limit" query:"limit" validate:"omitempty,min=1,max=50"`
    // Sort order (marketplace only, ignored for other scopes).
    Sort   SearchSortOrder `json:"sort,omitempty" query:"sort"`

    // ── Social-specific filters ────────────────────────────────────────
    // SubScope narrows social search to families, groups, or events.
    SubScope        *SocialSubScope `json:"sub_scope,omitempty" query:"sub_scope"`
    // MethodologySlug filters by methodology slug (social scope).
    MethodologySlug *string         `json:"methodology_slug,omitempty" query:"methodology_slug"`

    // ── Marketplace-specific filters ───────────────────────────────────
    MethodologyTags []uuid.UUID `json:"methodology_tags,omitempty" query:"methodology_tags"`
    SubjectTags     []string    `json:"subject_tags,omitempty" query:"subject_tags"`
    GradeMin        *int16      `json:"grade_min,omitempty" query:"grade_min"`
    GradeMax        *int16      `json:"grade_max,omitempty" query:"grade_max"`
    PriceMin        *int32      `json:"price_min,omitempty" query:"price_min"`
    PriceMax        *int32      `json:"price_max,omitempty" query:"price_max"`
    ContentType     *string     `json:"content_type,omitempty" query:"content_type"`
    WorldviewTags   []string    `json:"worldview_tags,omitempty" query:"worldview_tags"`
    FreeOnly        *bool       `json:"free_only,omitempty" query:"free_only"`

    // ── Learning-specific filters ──────────────────────────────────────
    StudentID  *uuid.UUID          `json:"student_id,omitempty" query:"student_id"`
    SourceType *LearningSourceType `json:"source_type,omitempty" query:"source_type"`
    DateFrom   *time.Time          `json:"date_from,omitempty" query:"date_from"`
    DateTo     *time.Time          `json:"date_to,omitempty" query:"date_to"`
    // SubjectTags reused for learning scope
}

// MarketplaceSearchFilters is the marketplace-specific filter struct (extracted from SearchParams for repository).
type MarketplaceSearchFilters struct {
    MethodologyTags []uuid.UUID `json:"methodology_tags,omitempty"`
    SubjectTags     []string    `json:"subject_tags,omitempty"`
    GradeMin        *int16      `json:"grade_min,omitempty"`
    GradeMax        *int16      `json:"grade_max,omitempty"`
    PriceMin        *int32      `json:"price_min,omitempty"`
    PriceMax        *int32      `json:"price_max,omitempty"`
    ContentType     *string     `json:"content_type,omitempty"`
    WorldviewTags   []string    `json:"worldview_tags,omitempty"`
    FreeOnly        *bool       `json:"free_only,omitempty"`
}

// LearningSearchFilters is the learning-specific filter struct (extracted from SearchParams for repository).
type LearningSearchFilters struct {
    StudentID  *uuid.UUID          `json:"student_id,omitempty"`
    SourceType *LearningSourceType `json:"source_type,omitempty"`
    DateFrom   *time.Time          `json:"date_from,omitempty"`
    DateTo     *time.Time          `json:"date_to,omitempty"`
    SubjectTags []string           `json:"subject_tags,omitempty"`
}

// AutocompleteParams is the autocomplete request parameters.
type AutocompleteParams struct {
    // Q is the search query text. Minimum 1 character.
    Q     string       `json:"q" query:"q" validate:"required,min=1"`
    // Scope to autocomplete within (default: marketplace if unspecified).
    Scope *SearchScope `json:"scope,omitempty" query:"scope"`
    // Limit is max suggestions to return (default 5, max 10).
    Limit int          `json:"limit,omitempty" query:"limit" validate:"omitempty,min=1,max=10"`
}
```

### §8.2 Response Types

```go
// SearchResponse is the unified search response.
type SearchResponse struct {
    // Results are the search results (polymorphic — type depends on scope).
    Results    []SearchResult `json:"results"`
    // TotalCount is the total count of matching results (for "X results found" display).
    TotalCount int64          `json:"total_count"`
    // Facets are marketplace scope only (nil for other scopes).
    Facets     *FacetCounts   `json:"facets,omitempty"`
    // NextCursor for next page (nil if no more results).
    NextCursor *string        `json:"next_cursor,omitempty"`
}

// SearchResult is a polymorphic search result — tagged union by scope.
// The Type field discriminates the variant.
type SearchResult struct {
    Type string `json:"type"` // "family", "group", "event", "post", "listing", "activity", "journal", "reading_item"

    // Social search results
    *FamilySearchResult  `json:"family,omitempty"`
    *GroupSearchResult   `json:"group,omitempty"`
    *EventSearchResult   `json:"event,omitempty"`
    *PostSearchResult    `json:"post,omitempty"`

    // Marketplace search result
    *ListingSearchResult `json:"listing,omitempty"`

    // Learning search results
    *ActivitySearchResult    `json:"activity,omitempty"`
    *JournalSearchResult     `json:"journal,omitempty"`
    *ReadingItemSearchResult `json:"reading_item,omitempty"`
}

// FamilySearchResult is a social family search result.
type FamilySearchResult struct {
    FamilyID       uuid.UUID `json:"family_id"`
    DisplayName    string    `json:"display_name"`
    MethodologyName *string  `json:"methodology_name,omitempty"`
    LocationRegion *string   `json:"location_region,omitempty"`
    IsFriend       bool      `json:"is_friend"`
    Relevance      float32   `json:"relevance"`
}

// GroupSearchResult is a social group search result.
type GroupSearchResult struct {
    GroupID        uuid.UUID `json:"group_id"`
    Name           string    `json:"name"`
    Description    *string   `json:"description,omitempty"`
    MemberCount    int32     `json:"member_count"`
    MethodologyName *string  `json:"methodology_name,omitempty"`
    Relevance      float32   `json:"relevance"`
}

// EventSearchResult is a social event search result.
type EventSearchResult struct {
    EventID       uuid.UUID `json:"event_id"`
    Title         string    `json:"title"`
    Description   *string   `json:"description,omitempty"`
    EventDate     time.Time `json:"event_date"`
    LocationName  *string   `json:"location_name,omitempty"`
    IsVirtual     bool      `json:"is_virtual"`
    Visibility    string    `json:"visibility"`
    AttendeeCount int32     `json:"attendee_count"`
    Relevance     float32   `json:"relevance"`
}

// PostSearchResult is a social post search result.
type PostSearchResult struct {
    PostID            uuid.UUID `json:"post_id"`
    ContentSnippet    string    `json:"content_snippet"`
    AuthorFamilyID    uuid.UUID `json:"author_family_id"`
    AuthorDisplayName string    `json:"author_display_name"`
    GroupName         *string   `json:"group_name,omitempty"`
    CreatedAt         time.Time `json:"created_at"`
    Relevance         float32   `json:"relevance"`
}

// ListingSearchResult is a marketplace listing search result.
type ListingSearchResult struct {
    ListingID       uuid.UUID   `json:"listing_id"`
    Title           string      `json:"title"`
    DescriptionSnippet string   `json:"description_snippet"`
    PriceCents      int32       `json:"price_cents"`
    ContentType     string      `json:"content_type"`
    RatingAvg       *float64    `json:"rating_avg,omitempty"`
    RatingCount     int32       `json:"rating_count"`
    PublisherName   string      `json:"publisher_name"`
    MethodologyTags []uuid.UUID `json:"methodology_tags"`
    SubjectTags     []string    `json:"subject_tags"`
    PublishedAt     time.Time   `json:"published_at"`
    Relevance       float32     `json:"relevance"`
}

// ActivitySearchResult is a learning activity search result.
type ActivitySearchResult struct {
    ActivityID   uuid.UUID `json:"activity_id"`
    Title        string    `json:"title"`
    Description  *string   `json:"description,omitempty"`
    StudentID    uuid.UUID `json:"student_id"`
    StudentName  string    `json:"student_name"`
    ActivityDate time.Time `json:"activity_date"`
    SubjectTags  []string  `json:"subject_tags"`
    Relevance    float32   `json:"relevance"`
}

// JournalSearchResult is a learning journal entry search result.
type JournalSearchResult struct {
    JournalID      uuid.UUID `json:"journal_id"`
    Title          string    `json:"title"`
    ContentSnippet string    `json:"content_snippet"`
    StudentID      uuid.UUID `json:"student_id"`
    StudentName    string    `json:"student_name"`
    EntryDate      time.Time `json:"entry_date"`
    EntryType      string    `json:"entry_type"`
    Relevance      float32   `json:"relevance"`
}

// ReadingItemSearchResult is a learning reading item search result (via reading_progress JOIN).
type ReadingItemSearchResult struct {
    ReadingItemID uuid.UUID `json:"reading_item_id"`
    Title         string    `json:"title"`
    Author        *string   `json:"author,omitempty"`
    Description   *string   `json:"description,omitempty"`
    StudentID     uuid.UUID `json:"student_id"`
    StudentName   string    `json:"student_name"`
    Status        string    `json:"status"`
    Relevance     float32   `json:"relevance"`
}

// FacetCounts holds marketplace facet counts for filter UI.
type FacetCounts struct {
    MethodologyTags []FacetBucket `json:"methodology_tags"`
    SubjectTags     []FacetBucket `json:"subject_tags"`
    ContentType     []FacetBucket `json:"content_type"`
    WorldviewTags   []FacetBucket `json:"worldview_tags"`
    PriceRanges     []FacetBucket `json:"price_ranges"`
    RatingRanges    []FacetBucket `json:"rating_ranges"`
}

// FacetBucket is a single facet bucket with value and document count.
type FacetBucket struct {
    Value       string `json:"value"`
    DisplayName string `json:"display_name"`
    Count       int64  `json:"count"`
}

// AutocompleteResponse is the autocomplete response.
type AutocompleteResponse struct {
    Suggestions []AutocompleteSuggestion `json:"suggestions"`
}

// AutocompleteSuggestion is a single autocomplete suggestion.
type AutocompleteSuggestion struct {
    // Text is the suggestion text to display.
    Text       string    `json:"text"`
    // EntityType is the entity type (family, group, listing, activity, etc.).
    EntityType string    `json:"entity_type"`
    // EntityID is the entity ID for direct navigation.
    EntityID   uuid.UUID `json:"entity_id"`
    // Score is the similarity/relevance score (0.0 - 1.0).
    Score      float32   `json:"score"`
}
```

### §8.3 Internal Types

```go
// SearchBackend routes queries to PostgreSQL FTS or Typesense.
// Phase 1: always PostgresFts.
// Phase 2: configurable per-scope via feature flag or threshold trigger.
type SearchBackend int

const (
    SearchBackendPostgresFts SearchBackend = iota
    SearchBackendTypesense
)

// MarketplaceSearchResults is a marketplace search result container with pagination metadata.
type MarketplaceSearchResults struct {
    Listings   []ListingSearchResult `json:"listings"`
    TotalCount int64                 `json:"total_count"`
    Facets     FacetCounts           `json:"facets"`
}

// TypesenseSearchQuery is a Typesense search query (Phase 2 internal type).
type TypesenseSearchQuery struct {
    Q       string   `json:"q"`
    QueryBy []string `json:"query_by"`
    FilterBy *string `json:"filter_by,omitempty"`
    SortBy   *string `json:"sort_by,omitempty"`
    FacetBy  []string `json:"facet_by,omitempty"`
    Page     int      `json:"page"`
    PerPage  int      `json:"per_page"`
}

// TypesenseSearchResponse is a Typesense search response (Phase 2 internal type).
type TypesenseSearchResponse struct {
    Found       int64                    `json:"found"`
    Hits        []map[string]any         `json:"hits"`
    FacetCounts []map[string]any         `json:"facet_counts,omitempty"`
}

// BulkIndexResult is a bulk index result (Phase 2).
type BulkIndexResult struct {
    Indexed int `json:"indexed"`
    Failed  int `json:"failed"`
    Errors  []string `json:"errors"`
}

// CollectionStats holds Typesense collection stats (Phase 2).
type CollectionStats struct {
    NumDocuments   int64 `json:"num_documents"`
    NumMemoryShards int32 `json:"num_memory_shards"`
}
```

---

## §9 Event Subscriptions

Search subscribes to domain events from `social::`, `mkt::`, `media::`, and `iam::`. In Phase 1,
all handlers are **no-ops** because `GENERATED ALWAYS` columns and `WHERE status = 'published'`
filters handle index maintenance automatically. In Phase 2, these handlers trigger Typesense
index updates.

| Event | Source Domain | Handler | Phase 1 | Phase 2 |
|-------|-------------|---------|---------|---------|
| `PostCreated` | `social::` | `PostCreatedHandler` | No-op (search_vector auto-updates) | Enqueue `IndexContentJob { Collection: "social_posts", EntityID }` |
| `ListingPublished` | `mkt::` | `ListingPublishedHandler` | No-op (search_vector auto-updates) | Enqueue `IndexContentJob { Collection: "marketplace_listings", EntityID }` |
| `ListingArchived` | `mkt::` | `ListingArchivedHandler` | No-op (WHERE status = 'published' excludes) | Remove document from `marketplace_listings` collection |
| `UploadPublished` | `media::` | `UploadPublishedHandler` | No-op (media not directly searchable) | Index media metadata into relevant collection |
| `FamilyDeletionScheduled` | `iam::` | `FamilyDeletionScheduledHandler` | No-op (CASCADE DELETE handles cleanup) | Remove all family documents from Typesense collections |

### Event Handler Implementations (Phase 1)

```go
// Phase 1: All event handlers are seams — they exist for forward compatibility
// but perform no work because PostgreSQL's GENERATED ALWAYS columns auto-maintain
// search vectors, and WHERE clauses exclude non-searchable rows.

type PostCreatedHandler struct {
    searchService SearchService
}

func (h *PostCreatedHandler) Handle(ctx context.Context, event *PostCreated) error {
    if err := h.searchService.HandlePostCreated(ctx, event); err != nil {
        return fmt.Errorf("search: handle post created: %w", err)
    }
    return nil
}

type ListingPublishedHandler struct {
    searchService SearchService
}

func (h *ListingPublishedHandler) Handle(ctx context.Context, event *ListingPublished) error {
    if err := h.searchService.HandleListingPublished(ctx, event); err != nil {
        return fmt.Errorf("search: handle listing published: %w", err)
    }
    return nil
}

type ListingArchivedHandler struct {
    searchService SearchService
}

func (h *ListingArchivedHandler) Handle(ctx context.Context, event *ListingArchived) error {
    if err := h.searchService.HandleListingArchived(ctx, event); err != nil {
        return fmt.Errorf("search: handle listing archived: %w", err)
    }
    return nil
}

type UploadPublishedHandler struct {
    searchService SearchService
}

func (h *UploadPublishedHandler) Handle(ctx context.Context, event *UploadPublished) error {
    if err := h.searchService.HandleUploadPublished(ctx, event); err != nil {
        return fmt.Errorf("search: handle upload published: %w", err)
    }
    return nil
}

type FamilyDeletionScheduledHandler struct {
    searchService SearchService
}

func (h *FamilyDeletionScheduledHandler) Handle(ctx context.Context, event *FamilyDeletionScheduled) error {
    if err := h.searchService.HandleFamilyDeletionScheduled(ctx, event.FamilyID); err != nil {
        return fmt.Errorf("search: handle family deletion scheduled: %w", err)
    }
    return nil
}
```

---

## §10 Search Scopes Deep Dive

This section documents the exact SQL queries for each search scope with privacy enforcement.
These queries are the Phase 1 implementation; Phase 2 Typesense queries are documented in §13.

### §10.1 Social Search

Social search operates across four entity types with scope-specific privacy enforcement.

#### Family Search

```sql
-- Search families by display name.
-- Privacy: friends OR discoverable (location_visible = true).
-- Blocks: excluded bidirectionally.
-- Ref: [ARCH §9.1], [S§14.2]
SELECT
    f.id AS family_id,
    f.display_name,
    md.display_name AS methodology_name,
    CASE WHEN sp.location_visible THEN f.location_region ELSE NULL END AS location_region,
    EXISTS (
        SELECT 1 FROM soc_friendships sf
        WHERE sf.status = 'accepted'
        AND (
            (sf.requester_family_id = $1 AND sf.accepter_family_id = f.id)
            OR (sf.accepter_family_id = $1 AND sf.requester_family_id = f.id)
        )
    ) AS is_friend,
    ts_rank(to_tsvector('english', f.display_name), plainto_tsquery('english', $2)) AS relevance
FROM iam_families f
JOIN soc_profiles sp ON sp.family_id = f.id
LEFT JOIN method_definitions md ON md.slug = f.primary_methodology_slug
WHERE f.id != $1  -- exclude self
AND (
    -- Friends [S§14.2]
    EXISTS (
        SELECT 1 FROM soc_friendships sf
        WHERE sf.status = 'accepted'
        AND (
            (sf.requester_family_id = $1 AND sf.accepter_family_id = f.id)
            OR (sf.accepter_family_id = $1 AND sf.requester_family_id = f.id)
        )
    )
    -- Or discoverable (opted into location-based discovery) [S§7.8]
    OR sp.location_visible = true
)
-- Block exclusion (bidirectional) [S§7.9]
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = f.id)
       OR (sb.blocker_family_id = f.id AND sb.blocked_family_id = $1)
)
AND f.display_name ILIKE '%' || $2 || '%'
ORDER BY relevance DESC, f.id
LIMIT $3;
```

#### Group Search

```sql
-- Search groups by name and description.
-- All non-private groups are searchable by any authenticated user.
-- Block exclusion: exclude groups created by blocked families.
-- Optional methodology filter.
SELECT
    g.id AS group_id,
    g.name,
    g.description,
    (SELECT COUNT(*) FROM soc_group_members gm WHERE gm.group_id = g.id AND gm.status = 'active') AS member_count,
    md.display_name AS methodology_name,
    ts_rank(
        to_tsvector('english', coalesce(g.name, '') || ' ' || coalesce(g.description, '')),
        websearch_to_tsquery('english', $2)
    ) AS relevance
FROM soc_groups g
LEFT JOIN method_definitions md ON md.slug = g.methodology_slug
WHERE to_tsvector('english', coalesce(g.name, '') || ' ' || coalesce(g.description, ''))
    @@ websearch_to_tsquery('english', $2)
-- Block exclusion: exclude groups created by blocked families
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = g.creator_family_id)
       OR (sb.blocker_family_id = g.creator_family_id AND sb.blocked_family_id = $1)
)
-- Optional methodology filter
AND ($3::text IS NULL OR g.methodology_slug = $3)
ORDER BY relevance DESC, g.id
LIMIT $4;
```

#### Event Search

```sql
-- Search events by title, description, and location.
-- Visibility enforcement:
--   'discoverable' → visible to all authenticated users
--   'friends' → visible only to friends of the creator
--   'group' → visible only to members of the associated group
-- Block exclusion. Cancelled events excluded.
-- Ref: [S§14.1], [S§7.7]
SELECT
    e.id AS event_id,
    e.title,
    e.description,
    e.event_date,
    e.location_name,
    e.is_virtual,
    e.visibility,
    e.attendee_count,
    ts_rank(
        to_tsvector('english', coalesce(e.title, '') || ' ' || coalesce(e.description, '')),
        websearch_to_tsquery('english', $2)
    ) AS relevance
FROM soc_events e
WHERE e.status = 'active'
AND to_tsvector('english', coalesce(e.title, '') || ' ' || coalesce(e.description, ''))
    @@ websearch_to_tsquery('english', $2)
-- Visibility enforcement
AND (
    -- Discoverable events: visible to all
    e.visibility = 'discoverable'
    -- Friends-only events: visible to friends of creator
    OR (
        e.visibility = 'friends'
        AND EXISTS (
            SELECT 1 FROM soc_friendships sf
            WHERE sf.status = 'accepted'
            AND (
                (sf.requester_family_id = $1 AND sf.accepter_family_id = e.creator_family_id)
                OR (sf.accepter_family_id = $1 AND sf.requester_family_id = e.creator_family_id)
            )
        )
    )
    -- Group events: visible to group members
    OR (
        e.visibility = 'group'
        AND e.group_id IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM soc_group_members gm
            WHERE gm.group_id = e.group_id
            AND gm.family_id = $1
            AND gm.status = 'active'
        )
    )
    -- Own events: always visible to creator
    OR e.creator_family_id = $1
)
-- Block exclusion (bidirectional)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = e.creator_family_id)
       OR (sb.blocker_family_id = e.creator_family_id AND sb.blocked_family_id = $1)
)
-- Optional methodology filter
AND ($3::text IS NULL OR e.methodology_slug = $3)
ORDER BY relevance DESC, e.id
LIMIT $4;
```

#### Post Search

```sql
-- Search posts by content.
-- Privacy: only posts visible to the searcher.
--   'friends' visibility → searcher must be friend of post author
--   'group' visibility → searcher must be member of the group
-- Block exclusion. No public posts exist (CHECK constraint).
-- Ref: [S§7.2.2]
SELECT
    p.id AS post_id,
    LEFT(p.content, 200) AS content_snippet,
    p.family_id AS author_family_id,
    f.display_name AS author_display_name,
    g.name AS group_name,
    p.created_at,
    ts_rank(p.search_vector, websearch_to_tsquery('english', $2)) AS relevance
FROM soc_posts p
JOIN iam_families f ON f.id = p.family_id
LEFT JOIN soc_groups g ON g.id = p.group_id
WHERE p.search_vector @@ websearch_to_tsquery('english', $2)
AND (
    -- Own posts
    p.family_id = $1
    -- Friends' posts (visibility = 'friends')
    OR (
        p.visibility = 'friends'
        AND EXISTS (
            SELECT 1 FROM soc_friendships sf
            WHERE sf.status = 'accepted'
            AND (
                (sf.requester_family_id = $1 AND sf.accepter_family_id = p.family_id)
                OR (sf.accepter_family_id = $1 AND sf.requester_family_id = p.family_id)
            )
        )
    )
    -- Group posts (visibility = 'group')
    OR (
        p.visibility = 'group'
        AND p.group_id IS NOT NULL
        AND EXISTS (
            SELECT 1 FROM soc_group_members gm
            WHERE gm.group_id = p.group_id
            AND gm.family_id = $1
            AND gm.status = 'active'
        )
    )
)
-- Block exclusion (bidirectional)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = p.family_id)
       OR (sb.blocker_family_id = p.family_id AND sb.blocked_family_id = $1)
)
ORDER BY relevance DESC, p.id
LIMIT $3;
```

### §10.2 Marketplace Search

```sql
-- Marketplace full-text search with faceted filtering.
-- Only published listings. No family scoping needed.
-- Weighted search_vector: title = 'A', description = 'B'.
-- Ref: [ARCH §9.1], [S§9.3]
SELECT
    l.id AS listing_id,
    l.title,
    LEFT(l.description, 200) AS description_snippet,
    l.price_cents,
    l.content_type,
    l.rating_avg,
    l.rating_count,
    p.name AS publisher_name,
    l.methodology_tags,
    l.subject_tags,
    l.published_at,
    ts_rank(l.search_vector, websearch_to_tsquery('english', $1)) AS relevance
FROM mkt_listings l
JOIN mkt_publishers p ON p.id = l.publisher_id
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
  -- Faceted filters [S§9.3]
  AND ($2::uuid[] IS NULL OR l.methodology_tags && $2)       -- methodology filter
  AND ($3::text[] IS NULL OR l.subject_tags && $3)            -- subject filter
  AND ($4::smallint IS NULL OR l.grade_min <= $4)             -- grade range max
  AND ($5::smallint IS NULL OR l.grade_max >= $5)             -- grade range min
  AND ($6::int IS NULL OR l.price_cents <= $6)                -- price max
  AND ($7::int IS NULL OR l.price_cents >= $7)                -- price min
  AND ($8::text IS NULL OR l.content_type = $8)               -- content type
  AND ($9::text[] IS NULL OR l.worldview_tags && $9)           -- worldview filter
  AND ($10::bool IS NULL OR NOT $10 OR l.price_cents = 0)     -- free_only filter
ORDER BY
    CASE WHEN $11 = 'relevance' THEN ts_rank(l.search_vector, websearch_to_tsquery('english', $1)) END DESC,
    CASE WHEN $11 = 'price_asc' THEN l.price_cents END ASC,
    CASE WHEN $11 = 'price_desc' THEN l.price_cents END DESC,
    CASE WHEN $11 = 'rating' THEN l.rating_avg END DESC,
    CASE WHEN $11 = 'recency' THEN l.published_at END DESC,
    l.id  -- tiebreaker for stable pagination
LIMIT $12 OFFSET $13;
```

#### Facet Count Queries

```sql
-- Count listings per methodology tag (for filter UI).
-- Applied to the same query + filters, excluding the methodology filter itself.
SELECT unnest(l.methodology_tags) AS tag_value,
       md.display_name,
       COUNT(*) AS count
FROM mkt_listings l
LEFT JOIN method_definitions md ON md.id = unnest(l.methodology_tags)
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
  -- Apply all filters EXCEPT methodology
  AND ($3::text[] IS NULL OR l.subject_tags && $3)
  -- ... (other filters)
GROUP BY tag_value, md.display_name
ORDER BY count DESC;

-- Count listings per content type.
SELECT l.content_type AS value,
       l.content_type AS display_name,
       COUNT(*) AS count
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
  -- Apply all filters EXCEPT content_type
GROUP BY l.content_type
ORDER BY count DESC;

-- Price range buckets: free, $1-$10, $10-$25, $25-$50, $50+
SELECT
    CASE
        WHEN l.price_cents = 0 THEN 'free'
        WHEN l.price_cents BETWEEN 1 AND 1000 THEN '1_to_10'
        WHEN l.price_cents BETWEEN 1001 AND 2500 THEN '10_to_25'
        WHEN l.price_cents BETWEEN 2501 AND 5000 THEN '25_to_50'
        ELSE '50_plus'
    END AS value,
    COUNT(*) AS count
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.search_vector @@ websearch_to_tsquery('english', $1)
GROUP BY value
ORDER BY count DESC;
```

### §10.3 Learning Search (Family-Scoped)

```sql
-- Learning search — ALWAYS scoped to the authenticated family's data.
-- UNION ALL across activity logs, journal entries, and reading items.
-- cross-family learning data MUST NOT be searchable [S§14.2].
-- Ref: [ARCH §9.1], [S§14.1]

-- Activities
SELECT
    al.id,
    'activity' AS source_type,
    al.title,
    LEFT(al.description, 200) AS description_snippet,
    al.student_id,
    s.display_name AS student_name,
    al.activity_date AS date,
    al.subject_tags,
    ts_rank(al.search_vector, websearch_to_tsquery('english', $2)) AS relevance
FROM learn_activity_logs al
JOIN iam_students s ON s.id = al.student_id
WHERE al.family_id = $1  -- ALWAYS family-scoped
  AND al.search_vector @@ websearch_to_tsquery('english', $2)
  AND ($3::uuid IS NULL OR al.student_id = $3)             -- optional student filter
  AND ($4::date IS NULL OR al.activity_date >= $4)          -- date range start
  AND ($5::date IS NULL OR al.activity_date <= $5)          -- date range end
  AND ($6::text[] IS NULL OR al.subject_tags && $6)         -- subject filter

UNION ALL

-- Journal entries
SELECT
    je.id,
    'journal' AS source_type,
    je.title,
    LEFT(je.content, 200) AS description_snippet,
    je.student_id,
    s.display_name AS student_name,
    je.entry_date AS date,
    je.subject_tags,
    ts_rank(je.search_vector, websearch_to_tsquery('english', $2)) AS relevance
FROM learn_journal_entries je
JOIN iam_students s ON s.id = je.student_id
WHERE je.family_id = $1  -- ALWAYS family-scoped
  AND je.search_vector @@ websearch_to_tsquery('english', $2)
  AND ($3::uuid IS NULL OR je.student_id = $3)
  AND ($4::date IS NULL OR je.entry_date >= $4)
  AND ($5::date IS NULL OR je.entry_date <= $5)
  AND ($6::text[] IS NULL OR je.subject_tags && $6)

UNION ALL

-- Reading items (via reading_progress JOIN for family scoping)
SELECT
    ri.id,
    'reading' AS source_type,
    ri.title,
    LEFT(ri.description, 200) AS description_snippet,
    rp.student_id,
    s.display_name AS student_name,
    rp.created_at::date AS date,
    ri.subject_tags,
    ts_rank(ri.search_vector, websearch_to_tsquery('english', $2)) AS relevance
FROM learn_reading_items ri
JOIN learn_reading_progress rp ON rp.reading_item_id = ri.id AND rp.family_id = $1
JOIN iam_students s ON s.id = rp.student_id
WHERE ri.search_vector @@ websearch_to_tsquery('english', $2)
  AND ($3::uuid IS NULL OR rp.student_id = $3)
  AND ($4::date IS NULL OR rp.created_at::date >= $4)
  AND ($5::date IS NULL OR rp.created_at::date <= $5)
  AND ($6::text[] IS NULL OR ri.subject_tags && $6)

ORDER BY relevance DESC, id
LIMIT $7;
```

> **Reading item scoping note**: `learn_reading_items` has no `family_id` column (it's
> publisher-level content). Family scoping is achieved by JOINing through `learn_reading_progress`
> which has `family_id` and `student_id`. This ensures families can only search reading items
> they have associated with a student via reading progress tracking.

---

## §11 Autocomplete

Autocomplete provides type-ahead suggestions with a target latency of < 200ms. Each scope
uses a different matching strategy optimized for its data characteristics.

### §11.1 Marketplace Autocomplete (`pg_trgm`)

```sql
-- Trigram-based fuzzy autocomplete on listing titles.
-- Uses the idx_mkt_listings_title_trgm GIN index for fast similarity matching.
-- Only published listings.
-- Ref: [ARCH §9.1], [S§14.2]
SELECT DISTINCT
    l.title AS text,
    'listing' AS entity_type,
    l.id AS entity_id,
    similarity(l.title, $1) AS score
FROM mkt_listings l
WHERE l.status = 'published'
  AND l.title % $1  -- trigram similarity match (default threshold 0.3)
ORDER BY score DESC
LIMIT $2;
```

### §11.2 Social Autocomplete (`ILIKE` prefix)

```sql
-- Prefix-based autocomplete on family display names and group names.
-- Respects friendship/discovery visibility and block exclusion.
-- Two queries UNIONed for families and groups.

-- Families
SELECT
    f.display_name AS text,
    'family' AS entity_type,
    f.id AS entity_id,
    1.0 AS score  -- exact prefix match, no scoring needed
FROM iam_families f
JOIN soc_profiles sp ON sp.family_id = f.id
WHERE f.id != $1
AND f.display_name ILIKE $2 || '%'
AND (
    EXISTS (
        SELECT 1 FROM soc_friendships sf
        WHERE sf.status = 'accepted'
        AND (
            (sf.requester_family_id = $1 AND sf.accepter_family_id = f.id)
            OR (sf.accepter_family_id = $1 AND sf.requester_family_id = f.id)
        )
    )
    OR sp.location_visible = true
)
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = f.id)
       OR (sb.blocker_family_id = f.id AND sb.blocked_family_id = $1)
)
LIMIT $3

UNION ALL

-- Groups
SELECT
    g.name AS text,
    'group' AS entity_type,
    g.id AS entity_id,
    1.0 AS score
FROM soc_groups g
WHERE g.name ILIKE $2 || '%'
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = g.creator_family_id)
       OR (sb.blocker_family_id = g.creator_family_id AND sb.blocked_family_id = $1)
)
LIMIT $3

ORDER BY text
LIMIT $3;
```

### §11.3 Learning Autocomplete (`ILIKE` prefix, family-scoped)

```sql
-- Prefix-based autocomplete on activity and journal titles.
-- ALWAYS family-scoped.

SELECT title AS text,
       'activity' AS entity_type,
       id AS entity_id,
       1.0 AS score
FROM learn_activity_logs
WHERE family_id = $1
AND title ILIKE $2 || '%'

UNION ALL

SELECT title AS text,
       'journal' AS entity_type,
       id AS entity_id,
       1.0 AS score
FROM learn_journal_entries
WHERE family_id = $1
AND title ILIKE $2 || '%'

ORDER BY text
LIMIT $3;
```

---

## §12 Privacy & Access Control

Privacy enforcement is the primary complexity of the search domain. Each scope has a
fundamentally different privacy model. This section summarizes the enforcement rules
documented in §10 and §11 SQL queries.

### §12.1 Social Scope Privacy

| Entity | Visibility Rule | Enforcement Mechanism |
|--------|----------------|----------------------|
| **Families** | Friends (accepted friendship) OR discoverable (`location_visible = true`) | `soc_friendships` JOIN + `soc_profiles.location_visible` check |
| **Groups** | All non-private groups visible to authenticated users | No additional filtering beyond authentication |
| **Events (discoverable)** | Visible to all authenticated users | `WHERE visibility = 'discoverable'` |
| **Events (friends)** | Visible only to friends of the event creator | `soc_friendships` JOIN on `creator_family_id` |
| **Events (group)** | Visible only to active members of the associated group | `soc_group_members` JOIN on `group_id` |
| **Posts (friends)** | Visible only to friends of the post author | `soc_friendships` JOIN on `family_id` |
| **Posts (group)** | Visible only to active members of the post's group | `soc_group_members` JOIN on `group_id` |

**Block enforcement** (applies to all social entities):
- Bidirectional: if A blocks B, neither A nor B can see each other's content in search
- Silent: blocked users receive zero results for the blocker (not an error or indication)
- SQL: `NOT EXISTS (SELECT 1 FROM soc_blocks WHERE ...)` with both directions checked

**Location privacy**:
- `location_region` is only included in family search results if `soc_profiles.location_visible = true`
- GPS coordinates are never stored or returned `[S§16.2]`

### §12.2 Marketplace Scope Privacy

| Rule | Enforcement |
|------|------------|
| Only published listings | `WHERE l.status = 'published'` |
| No family scoping | Marketplace content is public to all authenticated users |
| Creator identity visible | Publisher name shown via `mkt_publishers` JOIN |

### §12.3 Learning Scope Privacy

| Rule | Enforcement |
|------|------------|
| **ALWAYS family-scoped** | `WHERE family_id = $1` on every query (via `FamilyScope`) |
| **Cross-family data MUST NOT be searchable** | No query path exists without family_id filter `[S§14.2]` |
| **Reading items via progress JOIN** | `learn_reading_items` accessed only through `learn_reading_progress` JOIN with `family_id` |
| **NEVER uses Typesense** | Even in Phase 2+, learning search stays on PostgreSQL FTS (smaller dataset, strict privacy, defense-in-depth) |

> **Defense-in-depth for learning search**: Learning queries set `app.family_id` via the
> `FamilyScope` extractor, enabling the owning domain's RLS policies as a second line of
> defense. Even if application-level WHERE clauses were accidentally removed, RLS would
> prevent cross-family data leakage.

---

## §13 Phase 2+ Typesense Migration

### §13.1 Migration Triggers

Migrate to Typesense when **any** of:
- Marketplace exceeds ~100K listings
- Social posts exceed ~500K rows
- Search latency exceeds 500ms p95 `[S§17.3]`

### §13.2 Dual-Backend Architecture

```go
// Search service with dual-backend support.
// Learning search ALWAYS uses PostgreSQL — never Typesense.
func (s *SearchServiceImpl) Search(ctx context.Context, auth *AuthContext, scope *FamilyScope, params *SearchParams) (*SearchResponse, error) {
    switch params.Scope {
    case SearchScopeLearning:
        // ALWAYS PostgreSQL — privacy-critical, smaller dataset
        return s.learningRepo.SearchLearning(ctx, scope, params.Q, &filters, params.Limit, cursor)

    case SearchScopeMarketplace:
        switch s.backend {
        case SearchBackendPostgresFts:
            return s.marketplaceRepo.SearchListings(ctx, params.Q, &filters, params.Sort, params.Limit, cursor)
        case SearchBackendTypesense:
            tsQuery := buildTypesenseMarketplaceQuery(params)
            tsResponse, err := s.typesense.Search(ctx, "marketplace_listings", tsQuery)
            if err != nil {
                return nil, err
            }
            return convertTypesenseToSearchResponse(tsResponse)
        }

    case SearchScopeSocial:
        switch s.backend {
        case SearchBackendPostgresFts:
            // Dispatch to sub-scope repositories
            return s.searchSocialPostgres(ctx, auth, scope, params)
        case SearchBackendTypesense:
            // Typesense query with post-filter privacy enforcement
            return s.searchSocialTypesense(ctx, auth, scope, params)
        }
    }

    return nil, fmt.Errorf("unsupported search scope: %s", params.Scope)
}
```

### §13.3 Migration Strategy (Zero-Downtime)

| Step | Action | Rollback |
|------|--------|----------|
| 1 | Deploy Typesense cluster (3-node minimum for Raft HA) | Terminate cluster |
| 2 | Create collections with schema matching source tables | Drop collections |
| 3 | Run `BulkIndexJob` to index existing PostgreSQL data | No rollback needed |
| 4 | Enable shadow mode: query both backends, compare results, log discrepancies | Disable shadow mode |
| 5 | Switch reads to Typesense (`SearchBackendTypesense`) via config flag | Revert config to `PostgresFts` |
| 6 | Maintain PostgreSQL FTS indexes as fallback for 30+ days | N/A |
| 7 | Remove PostgreSQL FTS indexes only after Typesense proven stable | Recreate indexes from columns |

### §13.4 Typesense Collection Configuration

```json
// Collection: marketplace_listings
{
  "name": "marketplace_listings",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "title", "type": "string", "sort": true},
    {"name": "description", "type": "string"},
    {"name": "price_cents", "type": "int32", "facet": false, "sort": true},
    {"name": "content_type", "type": "string", "facet": true},
    {"name": "rating_avg", "type": "float", "facet": false, "sort": true},
    {"name": "rating_count", "type": "int32"},
    {"name": "publisher_name", "type": "string"},
    {"name": "methodology_tags", "type": "string[]", "facet": true},
    {"name": "subject_tags", "type": "string[]", "facet": true},
    {"name": "worldview_tags", "type": "string[]", "facet": true},
    {"name": "grade_min", "type": "int32"},
    {"name": "grade_max", "type": "int32"},
    {"name": "published_at", "type": "int64", "sort": true}
  ],
  "default_sorting_field": "published_at",
  "token_separators": ["-", "_"]
}

// Collection: social_posts
{
  "name": "social_posts",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "content", "type": "string"},
    {"name": "family_id", "type": "string", "facet": true},
    {"name": "group_id", "type": "string", "optional": true},
    {"name": "visibility", "type": "string", "facet": true},
    {"name": "created_at", "type": "int64", "sort": true}
  ],
  "default_sorting_field": "created_at"
}

// Collection: social_groups
{
  "name": "social_groups",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "name", "type": "string", "sort": true},
    {"name": "description", "type": "string"},
    {"name": "methodology_id", "type": "string", "optional": true, "facet": true},
    {"name": "member_count", "type": "int32", "sort": true},
    {"name": "creator_family_id", "type": "string"}
  ],
  "default_sorting_field": "member_count"
}

// Collection: social_events
{
  "name": "social_events",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "title", "type": "string"},
    {"name": "description", "type": "string"},
    {"name": "event_date", "type": "int64", "sort": true},
    {"name": "location_name", "type": "string", "optional": true},
    {"name": "location_region", "type": "string", "optional": true},
    {"name": "is_virtual", "type": "bool"},
    {"name": "visibility", "type": "string", "facet": true},
    {"name": "creator_family_id", "type": "string"},
    {"name": "group_id", "type": "string", "optional": true},
    {"name": "methodology_id", "type": "string", "optional": true, "facet": true},
    {"name": "attendee_count", "type": "int32", "sort": true}
  ],
  "default_sorting_field": "event_date"
}
```

### §13.5 Typesense HA Clustering

Typesense provides built-in Raft-based high availability in its open-source edition:

- **Minimum cluster**: 3 nodes (tolerates 1 node failure)
- **Recommended cluster**: 5 nodes (tolerates 2 node failures)
- **Replication**: Automatic via Raft consensus — writes go to leader, replicated to followers
- **Failover**: Automatic leader election on node failure, no manual intervention
- **No enterprise license required** (unlike Meilisearch HA which requires Enterprise)

> **Phase 2 deployment**: Start with a 3-node Typesense cluster behind the application.
> Each node runs on a separate host for fault isolation. Phase 3: scale to 5 nodes if
> search traffic warrants.

---

## §14 Background Jobs

### §14.1 `IndexContentJob` (Phase 2)

```go
// IndexContentJob indexes a single document into the Typesense collection.
// Triggered by domain event handlers (PostCreated, ListingPublished, etc.)
// when SearchBackend is Typesense.
//
// Queue: search_index (dedicated queue for search jobs)
// Retry: 3 attempts with exponential backoff (1s, 4s, 16s)
// Timeout: 10 seconds
type IndexContentJob struct {
    Collection string    `json:"collection"` // "marketplace_listings", "social_posts", etc.
    EntityID   uuid.UUID `json:"entity_id"`  // Source entity ID
    Action     IndexAction `json:"action"`   // Upsert or Remove
}

type IndexAction string

const (
    IndexActionUpsert IndexAction = "upsert"
    IndexActionRemove IndexAction = "remove"
)
```

### §14.2 `BulkIndexJob` (Phase 2)

```go
// BulkIndexJob bulk indexes all documents from a source table into a Typesense collection.
// Used during initial migration and periodic re-indexing.
//
// Queue: search_bulk (separate queue to avoid blocking incremental updates)
// Retry: 1 attempt (manual re-trigger for bulk jobs)
// Timeout: 30 minutes
// Batch size: 1000 documents per Typesense API call
type BulkIndexJob struct {
    Collection  string     `json:"collection"`
    SourceTable string     `json:"source_table"`
    BatchSize   int        `json:"batch_size"`   // Default 1000
    Cursor      *uuid.UUID `json:"cursor"`       // Resume from last processed ID
}
```

---

## §15 Testing Strategy

### §15.1 Social Search Tests (18 assertions)

| # | Test | Assertion |
|---|------|-----------|
| 1 | Family search returns friends | Accepted friendship → family appears in results |
| 2 | Family search returns discoverable families | `location_visible = true` → appears in results |
| 3 | Family search excludes non-friends non-discoverable | No friendship + `location_visible = false` → NOT in results |
| 4 | Family search excludes blocked families (forward) | Searcher blocks target → target NOT in results |
| 5 | Family search excludes blocked families (reverse) | Target blocks searcher → target NOT in results |
| 6 | Family search excludes self | Searcher's own family NOT in results |
| 7 | Family search returns methodology name | `primary_methodology_id` → resolved display_name |
| 8 | Family search hides location when not visible | `location_visible = false` → `location_region = null` |
| 9 | Group search returns matching groups | Text matches group name/description → appears |
| 10 | Group search excludes blocked creator groups | Group creator blocked → group NOT in results |
| 11 | Group search filters by methodology | `methodology_id` filter → only matching groups returned |
| 12 | Event search returns discoverable events | `visibility = 'discoverable'` → all users see it |
| 13 | Event search returns friends-only events to friends | Friend of creator → event visible |
| 14 | Event search hides friends-only events from non-friends | Not friend of creator → event NOT visible |
| 15 | Event search returns group events to members | Group member → event visible |
| 16 | Event search hides group events from non-members | Not group member → event NOT visible |
| 17 | Event search excludes cancelled events | `status = 'cancelled'` → NOT in results |
| 18 | Post search respects visibility constraints | Friends-only post visible to friends, hidden from non-friends |

### §15.2 Marketplace Search Tests (16 assertions)

| # | Test | Assertion |
|---|------|-----------|
| 19 | Search returns published listings only | `status = 'published'` → appears; other statuses → NOT |
| 20 | Search ranks title matches above description matches | Title match (weight A) scores higher than description match (weight B) |
| 21 | Methodology filter works | `methodology_tags` filter → only matching listings |
| 22 | Subject filter works | `subject_tags` filter → only matching listings |
| 23 | Grade range filter works | `grade_min/max` filter → correct grade overlap |
| 24 | Price range filter works | `price_min/max` → only listings in range |
| 25 | Free-only filter works | `free_only = true` → only `price_cents = 0` listings |
| 26 | Content type filter works | `content_type` filter → only matching type |
| 27 | Worldview filter works | `worldview_tags` filter → only matching listings |
| 28 | Sort by price ascending | Listings ordered by `price_cents ASC` |
| 29 | Sort by price descending | Listings ordered by `price_cents DESC` |
| 30 | Sort by rating | Listings ordered by `rating_avg DESC` |
| 31 | Sort by recency | Listings ordered by `published_at DESC` |
| 32 | Facet counts reflect current filters | Changing one filter updates counts for other facets |
| 33 | Pagination returns stable results | Cursor-based pagination produces consistent pages |
| 34 | Empty query with filters returns filtered browse | `q=""` with facet filters → filtered results |

### §15.3 Learning Search Tests (10 assertions)

| # | Test | Assertion |
|---|------|-----------|
| 35 | Activity search returns family's activities | Own family's activities appear |
| 36 | Activity search excludes other families | Other family's activities do NOT appear |
| 37 | Journal search returns family's entries | Own family's journal entries appear |
| 38 | Journal search excludes other families | Other family's entries do NOT appear |
| 39 | Reading item search uses progress JOIN | Reading items appear only if `learn_reading_progress` exists for the family |
| 40 | Student filter works | Only specified student's data returned |
| 41 | Date range filter works | Only entries within date range returned |
| 42 | Subject tag filter works | Only entries with matching subject tags returned |
| 43 | Source type filter works | `source_type = 'activity'` → only activities returned |
| 44 | Cross-family leakage test | Create data for family A and B, search as A → only A's data |

### §15.4 Autocomplete Tests (5 assertions)

| # | Test | Assertion |
|---|------|-----------|
| 45 | Marketplace autocomplete returns fuzzy matches | Typo in query still returns similar titles |
| 46 | Social autocomplete respects privacy | Non-friend, non-discoverable families NOT suggested |
| 47 | Learning autocomplete is family-scoped | Only own family's titles suggested |
| 48 | Autocomplete returns within 200ms | Response time < 200ms for reasonable query |
| 49 | Autocomplete limits results | `limit` parameter caps number of suggestions |

---

## §16 Error Handling

```go
// SearchError represents search domain errors.
// Internal error details are logged but never exposed in API responses.
// [CODING_STANDARDS §4]

import (
    "errors"
    "fmt"
    "net/http"
)

var (
    // ErrQueryTooShort indicates query text is too short.
    ErrQueryTooShort = errors.New("query too short")

    // ErrInvalidScope indicates an invalid search scope was provided.
    ErrInvalidScope = errors.New("invalid search scope")

    // ErrInvalidSortForScope indicates an invalid sort order for the given scope.
    ErrInvalidSortForScope = errors.New("sort order not supported for scope")

    // ErrInvalidFilter indicates an invalid filter value.
    ErrInvalidFilter = errors.New("invalid filter value")

    // ErrBackendUnavailable indicates search backend is temporarily unavailable (Phase 2 — Typesense down).
    ErrBackendUnavailable = errors.New("search service temporarily unavailable")
)

// SearchError wraps a search-specific error with additional context.
type SearchError struct {
    Err     error
    Field   string // Optional: which field caused the error
    Reason  string // Optional: additional detail
    MinLen  int    // For QueryTooShort
}

func (e *SearchError) Error() string {
    if e.Field != "" {
        return fmt.Sprintf("%s: field=%s reason=%s", e.Err.Error(), e.Field, e.Reason)
    }
    return e.Err.Error()
}

func (e *SearchError) Unwrap() error {
    return e.Err
}

// StatusCode maps SearchError to HTTP status code.
// Internal details are never exposed to clients.
func (e *SearchError) StatusCode() int {
    switch {
    case errors.Is(e.Err, ErrQueryTooShort):
        return http.StatusUnprocessableEntity // 422
    case errors.Is(e.Err, ErrInvalidScope):
        return http.StatusBadRequest // 400
    case errors.Is(e.Err, ErrInvalidSortForScope):
        return http.StatusBadRequest // 400
    case errors.Is(e.Err, ErrInvalidFilter):
        return http.StatusBadRequest // 400
    case errors.Is(e.Err, ErrBackendUnavailable):
        return http.StatusServiceUnavailable // 503
    default:
        return http.StatusInternalServerError // 500
    }
}
```

| Error Variant | HTTP Status | Client Message |
|--------------|-------------|----------------|
| `ErrQueryTooShort` | 422 | `"Query too short: minimum {n} characters required"` |
| `ErrInvalidScope` | 400 | `"Invalid search scope"` |
| `ErrInvalidSortForScope` | 400 | `"Sort order not supported for this scope"` |
| `ErrInvalidFilter` | 400 | `"Invalid filter value"` |
| `ErrBackendUnavailable` | 503 | `"Search service temporarily unavailable"` |
| Database error | 500 | `"An internal error occurred"` (log actual error internally) |
| Internal error | 500 | `"An internal error occurred"` (log actual error internally) |

---

## §17 Cross-Domain Interactions

### §17.1 Provides (consumed by other domains)

Search is a **terminal read-only domain** — it does not publish domain events and no other
domain depends on search results for correctness.

| Capability | Consumer | Mechanism |
|-----------|----------|-----------|
| _(none)_ | — | Search is read-only; other domains query their own data directly |

### §17.2 Consumes (reads from other domains)

| Source Domain | Tables Read | Purpose |
|-------------|-------------|---------|
| `iam::` | `iam_families`, `iam_students` | Family display name for social search; student display name for learning results |
| `social::` | `soc_posts`, `soc_groups`, `soc_events`, `soc_friendships`, `soc_blocks`, `soc_profiles`, `soc_group_members` | Social search content + privacy enforcement |
| `mkt::` | `mkt_listings`, `mkt_publishers` | Marketplace search content + publisher name |
| `learn::` | `learn_activity_logs`, `learn_journal_entries`, `learn_reading_items`, `learn_reading_progress` | Learning search content + family-scoped access |
| `method::` | `method_definitions` | Methodology display name resolution |

### §17.3 Events Subscribed To

| Event | Source Domain | Spec Reference | Handler |
|-------|-------------|----------------|---------|
| `PostCreated` | `social::` | `05-social.md §17.3` | `PostCreatedHandler` (§9) |
| `ListingPublished` | `mkt::` | `07-mkt.md §18.3` | `ListingPublishedHandler` (§9) |
| `ListingArchived` | `mkt::` | `07-mkt.md §18.3` | `ListingArchivedHandler` (§9) |
| `UploadPublished` | `media::` | `09-media.md §16.3` | `UploadPublishedHandler` (§9) |
| `FamilyDeletionScheduled` | `iam::` | `01-iam.md §13.3` | `FamilyDeletionScheduledHandler` (§9) |

### §17.4 Event Handler Detail

| Event | Phase 1 Behavior | Phase 2 Behavior |
|-------|-----------------|-----------------|
| `PostCreated` | No-op: `soc_posts.search_vector` auto-updates via `GENERATED ALWAYS` | Enqueue `IndexContentJob { Collection: "social_posts", EntityID: event.PostID, Action: Upsert }` |
| `ListingPublished` | No-op: `mkt_listings.search_vector` auto-updates; listing appears when `status = 'published'` | Enqueue `IndexContentJob { Collection: "marketplace_listings", EntityID: event.ListingID, Action: Upsert }` |
| `ListingArchived` | No-op: `WHERE status = 'published'` excludes archived listings | Enqueue `IndexContentJob { Collection: "marketplace_listings", EntityID: event.ListingID, Action: Remove }` |
| `UploadPublished` | No-op: media is not directly searchable in Phase 1 | Index media metadata into the relevant collection (context-dependent) |
| `FamilyDeletionScheduled` | No-op: `ON DELETE CASCADE` handles cleanup when family is deleted | Remove all documents with `family_id` from Typesense `social_posts` and `social_events` collections |

---

## §18 Phase Scope

### Phase 1 (MVP)

- `GET /v1/search` with 3 scopes (social, marketplace, learning)
- `GET /v1/search/autocomplete` with scope-specific strategies
- PostgreSQL FTS for all scopes (`tsvector`, `pg_trgm`, `ILIKE`)
- Full privacy enforcement: friendship, blocks, discovery opt-in, family scoping
- 7 marketplace facet dimensions with facet count queries
- 5 marketplace sort options (relevance, price asc/desc, rating, recency)
- Cursor-based pagination
- Rate limiting: 60 req/min
- Event handler seams (no-ops): `PostCreated`, `ListingPublished`, `ListingArchived`, `UploadPublished`, `FamilyDeletionScheduled`
- `search_index_state` table created but unused

### Phase 2

- Typesense deployment (3-node Raft cluster)
- `SearchBackend` enum with dual-backend support
- `TypesenseAdapter` implementation
- Shadow mode for migration validation
- `IndexContentJob` activated (event handlers become active)
- `BulkIndexJob` for initial data migration
- Typesense-backed marketplace and social search
- Learning search remains on PostgreSQL FTS
- `search_index_state` activated for sync tracking

### Phase 3+

- `GET /v1/search/suggestions` — AI-powered search suggestions (depends on `recs::` domain)
- 5-node Typesense cluster for higher availability
- Cross-scope search (search all scopes in a single query)
- Search analytics (popular queries, zero-result queries)
- Personalized search ranking based on family's methodology and activity patterns

---

## §19 Verification Checklist

- [ ] All table names verified against source domain specs:
  - `soc_posts` (`05-social.md §3.2`), `soc_groups` (`05-social.md §3.2`),
    `soc_events` (`05-social.md §3.2`), `soc_friendships` (`05-social.md §3.2`),
    `soc_blocks` (`05-social.md §3.2`), `soc_profiles` (`05-social.md §3.2`),
    `soc_group_members` (`05-social.md §3.2`)
  - `mkt_listings` (`07-mkt.md §3.2`), `mkt_publishers` (`07-mkt.md §3.2`)
  - `learn_activity_logs` (`06-learn.md §3.2`), `learn_journal_entries` (`06-learn.md §3.2`),
    `learn_reading_items` (`06-learn.md §3.2`), `learn_reading_progress` (`06-learn.md §3.2`)
  - `iam_families` (`01-iam.md §3.2`), `iam_students` (`01-iam.md §3.2`)
  - `method_definitions` (`02-method.md §3.2`)
- [ ] All event names verified against source domain specs:
  - `PostCreated` (`05-social.md §17.3`), `ListingPublished` (`07-mkt.md §18.3`),
    `ListingArchived` (`07-mkt.md §18.3`), `UploadPublished` (`09-media.md §16.3`),
    `FamilyDeletionScheduled` (`01-iam.md §13.3`)
- [ ] Section numbering is consistent (§1-§20)
- [ ] All SPEC.md §14 requirements traced in §2
- [ ] All SPEC.md §9.3 requirements traced in §2
- [ ] Privacy model covers all S§14.2 requirements:
  - Social: friends OR discovery opt-in only (§12.1)
  - Learning: family-scoped, no cross-family searchable (§12.3)
  - Marketplace: published only (§12.2)
- [ ] `search_vector` definitions match source domain schema definitions (§3.1)
- [ ] GIN indexes for `soc_groups` and `soc_events` added to `05-social.md` (see §3.1 gap note)
- [ ] Block enforcement is bidirectional in all social queries (§10.1)
- [ ] Event visibility (friends/group/discoverable) enforced per-event (§10.1)
- [ ] Reading items scoped via `learn_reading_progress` JOIN (§10.3)
- [ ] Typesense replaces Meilisearch throughout (§7, §13)
- [ ] All 49 test assertions cover search correctness + privacy (§15)

---

## §20 Module Structure

Flat layout matching all other domains (no subdirectories). `[CODING §2.1]`

```
internal/search/
├── ports.go            # SearchService interface + 4 repository interfaces
├── errors.go           # Sentinel errors (ErrQueryTooShort, etc.) + SearchError wrapper
├── models.go           # All DTOs, enums, request/response types
├── service.go          # SearchServiceImpl (validation, scope dispatch, response assembly)
├── repository.go       # Pg*Repository impls (PostgreSQL FTS SQL for all 4 repos)
├── handler.go          # Echo handlers: GET /v1/search, GET /v1/search/autocomplete
├── event_handlers.go   # 5 event handler structs (Phase 1 no-ops, Phase 2 index triggers)
├── service_test.go     # Service unit tests (38 tests)
├── handler_test.go     # Handler unit tests (8 tests)
└── mock_test.go        # Function-pointer stubs for repos + service
```

> **Note on GORM models**: Search has no owned GORM models in Phase 1 (no owned tables active).
> The `search_index_state` GORM model will be added in `models.go` in Phase 2 when the table
> becomes active. The migration (`20260325000019_create_search_tables.sql`) is already in place.
