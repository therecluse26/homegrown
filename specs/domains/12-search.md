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
| **Module path** | `src/search/` |
| **DB prefix** | `search_` (Phase 2 only — no owned tables in Phase 1) |
| **Complexity class** | Simple (no `domain/` subdirectory) — indexing and retrieval, no business invariants `[ARCH §4.5]` |
| **CQRS** | Yes — command side (index updates, Phase 1 no-ops) / query side (search, autocomplete) `[ARCH §4.7]` |
| **External adapter** | `src/search/adapters/typesense.rs` (Phase 2 — Typesense search engine) |
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
`method_definitions` directly. Background job scheduling → sidekiq-rs `[ARCH §12]`. Phase 2
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

> ¹ **Gap note**: `ARCH §9.1` defines expression-based GIN indexes on `soc_groups` and
> `soc_events` for full-text search, but `05-social.md §3.2` does not yet include them.
> These indexes MUST be added to `05-social.md` before implementing search queries. Without
> these GIN indexes, the `to_tsvector(...)` expressions in §10.1 group/event search queries
> will fall back to sequential scans. Proposed addition to `05-social.md`:
> ```sql
> CREATE INDEX idx_soc_groups_search ON soc_groups
>     USING GIN(to_tsvector('english', coalesce(name, '') || ' ' || coalesce(description, '')));
> CREATE INDEX idx_soc_events_search ON soc_events
>     USING GIN(to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, '')));
> ```

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
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
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
- **Query**: `SearchParams { q: String, scope: SearchScope, cursor?: String, limit?: i32 (default 20, max 50), sort?: SearchSortOrder, filters?: scope-specific }`
- **Scope-specific query parameters**:
  - `scope=social`: `sub_scope?: SocialSubScope (families|groups|events)`, `methodology_id?: Uuid`
  - `scope=marketplace`: `methodology_tags?: Vec<Uuid>`, `subject_tags?: Vec<String>`, `grade_min?: i16`, `grade_max?: i16`, `price_max?: i32`, `price_min?: i32 (default 0)`, `content_type?: String`, `worldview_tags?: Vec<String>`, `free_only?: bool`
  - `scope=learning`: `student_id?: Uuid`, `source_type?: LearningSourceType (activity|journal|reading)`, `date_from?: NaiveDate`, `date_to?: NaiveDate`, `subject_tags?: Vec<String>`
- **Response**: `200 OK` → `SearchResponse { results: Vec<SearchResult>, total_count: i64, facets?: FacetCounts, next_cursor?: String }`
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
- **Query**: `AutocompleteParams { q: String, scope?: SearchScope (default: inferred from context), limit?: i32 (default 5, max 10) }`
- **Response**: `200 OK` → `AutocompleteResponse { suggestions: Vec<AutocompleteSuggestion> }`
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
`ai::` domain (not yet specified).

- **Auth**: `AuthContext` + `FamilyScope`
- **Response**: `200 OK` → `SearchSuggestionsResponse { suggestions: Vec<SearchSuggestion> }`

---

## §5 Service Interface

The search service follows lightweight CQRS: command side handles index update events (Phase 1
no-ops), query side handles search and autocomplete requests. `[ARCH §4.7]`

```rust
/// Search service — query side (search + autocomplete) and command side (index updates).
///
/// Phase 1: All queries use PostgreSQL FTS. Command-side methods are no-ops because
/// `GENERATED ALWAYS` columns auto-update search vectors on source table writes.
///
/// Phase 2+: `SearchBackend` enum routes marketplace and social queries to Typesense
/// when enabled. Command-side methods trigger Typesense index updates via `IndexContentJob`.
/// Learning search always stays on PostgreSQL FTS.
#[async_trait]
pub trait SearchService: Send + Sync {
    // ── Query Side ─────────────────────────────────────────────────────────

    /// Unified search across a single scope with privacy enforcement.
    /// Dispatches to scope-specific repository based on `params.scope`.
    async fn search(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        params: &SearchParams,
    ) -> Result<SearchResponse, SearchError>;

    /// Type-ahead autocomplete suggestions.
    /// Returns within 200ms target. Scope-specific implementation.
    async fn autocomplete(
        &self,
        auth: &AuthContext,
        scope: &FamilyScope,
        params: &AutocompleteParams,
    ) -> Result<AutocompleteResponse, SearchError>;

    // ── Command Side (Index Updates) ───────────────────────────────────────

    /// Handle new post creation — update social search index.
    /// Phase 1: no-op (GENERATED ALWAYS column auto-updates).
    /// Phase 2: enqueues IndexContentJob for Typesense.
    async fn handle_post_created(&self, event: &PostCreated) -> Result<(), SearchError>;

    /// Handle listing publication — add/update marketplace search index.
    /// Phase 1: no-op (GENERATED ALWAYS column auto-updates).
    /// Phase 2: enqueues IndexContentJob for Typesense.
    async fn handle_listing_published(&self, event: &ListingPublished) -> Result<(), SearchError>;

    /// Handle listing archival — remove from marketplace search index.
    /// Phase 1: no-op (listing status change excludes from `WHERE status = 'published'`).
    /// Phase 2: removes document from Typesense collection.
    async fn handle_listing_archived(&self, event: &ListingArchived) -> Result<(), SearchError>;

    /// Handle media upload publication — index media metadata.
    /// Phase 1: no-op (media is not directly searchable in Phase 1).
    /// Phase 2: indexes media metadata into relevant Typesense collection.
    async fn handle_upload_published(&self, event: &UploadPublished) -> Result<(), SearchError>;

    /// Handle family deletion — remove family data from all search indexes.
    /// Phase 1: no-op (source table CASCADE DELETE removes rows, GENERATED columns follow).
    /// Phase 2: removes all family-related documents from Typesense collections.
    async fn handle_family_deletion_scheduled(
        &self,
        family_id: Uuid,
    ) -> Result<(), SearchError>;
}
```

### SearchServiceImpl

```rust
pub struct SearchServiceImpl {
    social_repo: Arc<dyn SocialSearchRepository>,
    marketplace_repo: Arc<dyn MarketplaceSearchRepository>,
    learning_repo: Arc<dyn LearningSearchRepository>,
    autocomplete_repo: Arc<dyn AutocompleteRepository>,
    backend: SearchBackend,          // Phase 1: always PostgresFts
    // Phase 2 additions:
    // typesense: Option<Arc<dyn TypesenseAdapter>>,
    // job_queue: Arc<dyn JobQueue>,
}

impl SearchServiceImpl {
    pub fn new(
        social_repo: Arc<dyn SocialSearchRepository>,
        marketplace_repo: Arc<dyn MarketplaceSearchRepository>,
        learning_repo: Arc<dyn LearningSearchRepository>,
        autocomplete_repo: Arc<dyn AutocompleteRepository>,
    ) -> Self {
        Self {
            social_repo,
            marketplace_repo,
            learning_repo,
            autocomplete_repo,
            backend: SearchBackend::PostgresFts,
        }
    }
}
```

---

## §6 Repository Interfaces

Four repositories, one per search scope plus autocomplete. All repositories are query-only
(no mutations) in Phase 1. Repositories accept privacy context as parameters — they do not
trust callers to pre-filter.

### §6.1 `SocialSearchRepository`

```rust
/// Social search repository — queries soc_posts, soc_groups, soc_events, iam_families.
/// All queries enforce friendship + discovery visibility and block exclusion.
#[async_trait]
pub trait SocialSearchRepository: Send + Sync {
    /// Search families by display name.
    /// Returns only: (1) friends (soc_friendships.status = 'accepted'), or
    /// (2) families with soc_profiles.location_visible = true (discovery opt-in).
    /// Excludes: blocked families (bidirectional soc_blocks check).
    async fn search_families(
        &self,
        searcher_family_id: Uuid,
        query: &str,
        limit: i32,
        cursor: Option<&str>,
    ) -> Result<Vec<SocialSearchResult>, SearchError>;

    /// Search groups by name and description.
    /// Returns all non-private groups (searchable by any authenticated user).
    /// Block exclusion: groups created by blocked families are excluded.
    async fn search_groups(
        &self,
        searcher_family_id: Uuid,
        query: &str,
        methodology_id: Option<Uuid>,
        limit: i32,
        cursor: Option<&str>,
    ) -> Result<Vec<SocialSearchResult>, SearchError>;

    /// Search events by title, description, and location.
    /// Visibility enforcement:
    /// - 'discoverable': visible to all authenticated users
    /// - 'friends': visible only to friends of the creator
    /// - 'group': visible only to members of the associated group
    /// Excludes: events by blocked families, cancelled events.
    async fn search_events(
        &self,
        searcher_family_id: Uuid,
        query: &str,
        methodology_id: Option<Uuid>,
        limit: i32,
        cursor: Option<&str>,
    ) -> Result<Vec<SocialSearchResult>, SearchError>;
}
```

### §6.2 `MarketplaceSearchRepository`

```rust
/// Marketplace search repository — queries mkt_listings with faceted filtering.
/// Only returns listings with status = 'published'. No family scoping needed.
#[async_trait]
pub trait MarketplaceSearchRepository: Send + Sync {
    /// Full-text search with faceted filtering and sorting.
    /// Uses weighted search_vector (title = 'A', description = 'B') for relevance ranking.
    async fn search_listings(
        &self,
        query: &str,
        filters: &MarketplaceSearchFilters,
        sort: SearchSortOrder,
        limit: i32,
        cursor: Option<&str>,
    ) -> Result<MarketplaceSearchResults, SearchError>;

    /// Count listings per facet value for the current query + filters.
    /// Returns counts for: methodology_tags, subject_tags, content_type,
    /// worldview_tags, price_ranges, rating_ranges.
    async fn count_facets(
        &self,
        query: &str,
        filters: &MarketplaceSearchFilters,
    ) -> Result<FacetCounts, SearchError>;
}
```

### §6.3 `LearningSearchRepository`

```rust
/// Learning search repository — queries learn_activity_logs, learn_journal_entries,
/// learn_reading_items (via learn_reading_progress JOIN).
/// ALWAYS family-scoped via FamilyScope. NEVER uses Typesense, even in Phase 2.
#[async_trait]
pub trait LearningSearchRepository: Send + Sync {
    /// Search family's own learning data across activities, journals, and reading items.
    /// UNION ALL across multiple tables, always filtered by family_id.
    /// Optional filters: student_id, source_type, date range, subject_tags.
    async fn search_learning(
        &self,
        family_scope: &FamilyScope,
        query: &str,
        filters: &LearningSearchFilters,
        limit: i32,
        cursor: Option<&str>,
    ) -> Result<Vec<LearningSearchResult>, SearchError>;
}
```

### §6.4 `AutocompleteRepository`

```rust
/// Autocomplete repository — fast prefix/fuzzy matching for type-ahead.
/// Target latency: < 200ms.
#[async_trait]
pub trait AutocompleteRepository: Send + Sync {
    /// Marketplace autocomplete using pg_trgm similarity on listing titles.
    /// Only searches published listings.
    async fn autocomplete_marketplace(
        &self,
        query: &str,
        limit: i32,
    ) -> Result<Vec<AutocompleteSuggestion>, SearchError>;

    /// Social autocomplete using ILIKE prefix match on family display names
    /// and group names. Respects friendship/discovery visibility and blocks.
    async fn autocomplete_social(
        &self,
        searcher_family_id: Uuid,
        query: &str,
        limit: i32,
    ) -> Result<Vec<AutocompleteSuggestion>, SearchError>;

    /// Learning autocomplete using ILIKE prefix match on activity and journal titles.
    /// Always family-scoped.
    async fn autocomplete_learning(
        &self,
        family_scope: &FamilyScope,
        query: &str,
        limit: i32,
    ) -> Result<Vec<AutocompleteSuggestion>, SearchError>;
}
```

---

## §7 Adapter Interface

### §7.1 Phase 2: `TypesenseAdapter`

```rust
/// Typesense search engine adapter for Phase 2+ migration.
/// Provides high-performance search with typo tolerance, faceted filtering,
/// and built-in Raft-based HA clustering.
///
/// Typesense replaces Meilisearch (originally specified in ARCH §2.6) because:
/// - Built-in Raft-based HA clustering in the open-source edition
///   (Meilisearch requires Enterprise for HA)
/// - Sub-50ms search latency (C++ engine), field weighting for relevance tuning
/// - Production-proven clustering (3 or 5 node) without enterprise licensing
/// - Better fit for a solo developer running a COPPA-regulated platform
#[async_trait]
pub trait TypesenseAdapter: Send + Sync {
    /// Index a single document into a collection.
    async fn index_document(
        &self,
        collection: &str,
        document: serde_json::Value,
    ) -> Result<(), SearchError>;

    /// Remove a document from a collection by ID.
    async fn remove_document(
        &self,
        collection: &str,
        document_id: &str,
    ) -> Result<(), SearchError>;

    /// Bulk index multiple documents (used by BulkIndexJob).
    async fn bulk_index(
        &self,
        collection: &str,
        documents: Vec<serde_json::Value>,
    ) -> Result<BulkIndexResult, SearchError>;

    /// Execute a search query against a collection.
    async fn search(
        &self,
        collection: &str,
        query: &TypesenseSearchQuery,
    ) -> Result<TypesenseSearchResponse, SearchError>;

    /// Get collection health/stats (used for monitoring).
    async fn collection_stats(
        &self,
        collection: &str,
    ) -> Result<CollectionStats, SearchError>;
}
```

---

## §8 Models (DTOs)

### §8.1 Request Types

```rust
/// Search scope — determines which repository and privacy model to use.
#[derive(Debug, Clone, Deserialize, PartialEq)]
#[serde(rename_all = "snake_case")]
pub enum SearchScope {
    Social,
    Marketplace,
    Learning,
}

/// Social search sub-scope — optionally narrow social search to a specific entity type.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum SocialSubScope {
    Families,
    Groups,
    Events,
}

/// Learning source type — optionally narrow learning search to a specific source.
#[derive(Debug, Clone, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum LearningSourceType {
    Activity,
    Journal,
    Reading,
}

/// Sort order for marketplace search results.
#[derive(Debug, Clone, Deserialize, Default)]
#[serde(rename_all = "snake_case")]
pub enum SearchSortOrder {
    #[default]
    Relevance,
    PriceAsc,
    PriceDesc,
    Rating,
    Recency,
}

/// Unified search request parameters.
/// Deserialized from query string on GET /v1/search.
#[derive(Debug, Clone, Deserialize)]
pub struct SearchParams {
    /// Search query text. Minimum 2 characters.
    pub q: String,
    /// Which scope to search within.
    pub scope: SearchScope,
    /// Cursor for pagination (opaque string from previous response).
    pub cursor: Option<String>,
    /// Results per page (default 20, max 50).
    #[serde(default = "default_search_limit")]
    pub limit: i32,
    /// Sort order (marketplace only, ignored for other scopes).
    #[serde(default)]
    pub sort: SearchSortOrder,

    // ── Social-specific filters ────────────────────────────────────────
    /// Narrow social search to families, groups, or events.
    pub sub_scope: Option<SocialSubScope>,
    /// Filter by methodology (social and marketplace).
    pub methodology_id: Option<Uuid>,

    // ── Marketplace-specific filters ───────────────────────────────────
    pub methodology_tags: Option<Vec<Uuid>>,
    pub subject_tags: Option<Vec<String>>,
    pub grade_min: Option<i16>,
    pub grade_max: Option<i16>,
    pub price_min: Option<i32>,
    pub price_max: Option<i32>,
    pub content_type: Option<String>,
    pub worldview_tags: Option<Vec<String>>,
    pub free_only: Option<bool>,

    // ── Learning-specific filters ──────────────────────────────────────
    pub student_id: Option<Uuid>,
    pub source_type: Option<LearningSourceType>,
    pub date_from: Option<NaiveDate>,
    pub date_to: Option<NaiveDate>,
    // subject_tags reused for learning scope
}

fn default_search_limit() -> i32 { 20 }

/// Marketplace-specific filter struct (extracted from SearchParams for repository).
#[derive(Debug, Clone, Default)]
pub struct MarketplaceSearchFilters {
    pub methodology_tags: Option<Vec<Uuid>>,
    pub subject_tags: Option<Vec<String>>,
    pub grade_min: Option<i16>,
    pub grade_max: Option<i16>,
    pub price_min: Option<i32>,
    pub price_max: Option<i32>,
    pub content_type: Option<String>,
    pub worldview_tags: Option<Vec<String>>,
    pub free_only: Option<bool>,
}

/// Learning-specific filter struct (extracted from SearchParams for repository).
#[derive(Debug, Clone, Default)]
pub struct LearningSearchFilters {
    pub student_id: Option<Uuid>,
    pub source_type: Option<LearningSourceType>,
    pub date_from: Option<NaiveDate>,
    pub date_to: Option<NaiveDate>,
    pub subject_tags: Option<Vec<String>>,
}

/// Autocomplete request parameters.
#[derive(Debug, Clone, Deserialize)]
pub struct AutocompleteParams {
    /// Search query text. Minimum 1 character.
    pub q: String,
    /// Scope to autocomplete within (default: marketplace if unspecified).
    pub scope: Option<SearchScope>,
    /// Max suggestions to return (default 5, max 10).
    #[serde(default = "default_autocomplete_limit")]
    pub limit: Option<i32>,
}

fn default_autocomplete_limit() -> i32 { 5 }
```

### §8.2 Response Types

```rust
/// Unified search response.
#[derive(Debug, Clone, Serialize)]
pub struct SearchResponse {
    /// Search results (polymorphic — type depends on scope).
    pub results: Vec<SearchResult>,
    /// Total count of matching results (for "X results found" display).
    pub total_count: i64,
    /// Facet counts (marketplace scope only, None for other scopes).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub facets: Option<FacetCounts>,
    /// Cursor for next page (None if no more results).
    #[serde(skip_serializing_if = "Option::is_none")]
    pub next_cursor: Option<String>,
}

/// Polymorphic search result — tagged union by scope.
#[derive(Debug, Clone, Serialize)]
#[serde(tag = "type", rename_all = "snake_case")]
pub enum SearchResult {
    /// Social search result — family, group, or event.
    Family(FamilySearchResult),
    Group(GroupSearchResult),
    Event(EventSearchResult),
    Post(PostSearchResult),

    /// Marketplace search result — listing.
    Listing(ListingSearchResult),

    /// Learning search result — activity, journal, or reading item.
    Activity(ActivitySearchResult),
    Journal(JournalSearchResult),
    ReadingItem(ReadingItemSearchResult),
}

/// Social: family search result.
#[derive(Debug, Clone, Serialize)]
pub struct FamilySearchResult {
    pub family_id: Uuid,
    pub display_name: String,
    pub methodology_name: Option<String>,
    pub location_region: Option<String>,
    pub is_friend: bool,
    pub relevance: f32,
}

/// Social: group search result.
#[derive(Debug, Clone, Serialize)]
pub struct GroupSearchResult {
    pub group_id: Uuid,
    pub name: String,
    pub description: Option<String>,
    pub member_count: i32,
    pub methodology_name: Option<String>,
    pub relevance: f32,
}

/// Social: event search result.
#[derive(Debug, Clone, Serialize)]
pub struct EventSearchResult {
    pub event_id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub event_date: DateTime<Utc>,
    pub location_name: Option<String>,
    pub is_virtual: bool,
    pub visibility: String,
    pub attendee_count: i32,
    pub relevance: f32,
}

/// Social: post search result.
#[derive(Debug, Clone, Serialize)]
pub struct PostSearchResult {
    pub post_id: Uuid,
    pub content_snippet: String,
    pub author_family_id: Uuid,
    pub author_display_name: String,
    pub group_name: Option<String>,
    pub created_at: DateTime<Utc>,
    pub relevance: f32,
}

/// Marketplace: listing search result.
#[derive(Debug, Clone, Serialize)]
pub struct ListingSearchResult {
    pub listing_id: Uuid,
    pub title: String,
    pub description_snippet: String,
    pub price_cents: i32,
    pub content_type: String,
    pub rating_avg: Option<f64>,
    pub rating_count: i32,
    pub publisher_name: String,
    pub methodology_tags: Vec<Uuid>,
    pub subject_tags: Vec<String>,
    pub published_at: DateTime<Utc>,
    pub relevance: f32,
}

/// Learning: activity search result.
#[derive(Debug, Clone, Serialize)]
pub struct ActivitySearchResult {
    pub activity_id: Uuid,
    pub title: String,
    pub description: Option<String>,
    pub student_id: Uuid,
    pub student_name: String,
    pub activity_date: NaiveDate,
    pub subject_tags: Vec<String>,
    pub relevance: f32,
}

/// Learning: journal entry search result.
#[derive(Debug, Clone, Serialize)]
pub struct JournalSearchResult {
    pub journal_id: Uuid,
    pub title: String,
    pub content_snippet: String,
    pub student_id: Uuid,
    pub student_name: String,
    pub entry_date: NaiveDate,
    pub entry_type: String,
    pub relevance: f32,
}

/// Learning: reading item search result (via reading_progress JOIN).
#[derive(Debug, Clone, Serialize)]
pub struct ReadingItemSearchResult {
    pub reading_item_id: Uuid,
    pub title: String,
    pub author: Option<String>,
    pub description: Option<String>,
    pub student_id: Uuid,
    pub student_name: String,
    pub status: String,
    pub relevance: f32,
}

/// Marketplace facet counts for filter UI.
#[derive(Debug, Clone, Serialize)]
pub struct FacetCounts {
    pub methodology_tags: Vec<FacetBucket>,
    pub subject_tags: Vec<FacetBucket>,
    pub content_type: Vec<FacetBucket>,
    pub worldview_tags: Vec<FacetBucket>,
    pub price_ranges: Vec<FacetBucket>,
    pub rating_ranges: Vec<FacetBucket>,
}

/// A single facet bucket with value and document count.
#[derive(Debug, Clone, Serialize)]
pub struct FacetBucket {
    pub value: String,
    pub display_name: String,
    pub count: i64,
}

/// Autocomplete response.
#[derive(Debug, Clone, Serialize)]
pub struct AutocompleteResponse {
    pub suggestions: Vec<AutocompleteSuggestion>,
}

/// A single autocomplete suggestion.
#[derive(Debug, Clone, Serialize)]
pub struct AutocompleteSuggestion {
    /// The suggestion text to display.
    pub text: String,
    /// The entity type (family, group, listing, activity, etc.).
    pub entity_type: String,
    /// Entity ID for direct navigation.
    pub entity_id: Uuid,
    /// Similarity/relevance score (0.0 - 1.0).
    pub score: f32,
}
```

### §8.3 Internal Types

```rust
/// Search backend enum — routes queries to PostgreSQL FTS or Typesense.
/// Phase 1: always PostgresFts.
/// Phase 2: configurable per-scope via feature flag or threshold trigger.
#[derive(Debug, Clone)]
pub enum SearchBackend {
    PostgresFts,
    Typesense(Arc<dyn TypesenseAdapter>),
}

/// Marketplace search result container with pagination metadata.
#[derive(Debug, Clone)]
pub struct MarketplaceSearchResults {
    pub listings: Vec<ListingSearchResult>,
    pub total_count: i64,
    pub facets: FacetCounts,
}

/// Typesense search query (Phase 2 internal type).
#[derive(Debug, Clone)]
pub struct TypesenseSearchQuery {
    pub q: String,
    pub query_by: Vec<String>,
    pub filter_by: Option<String>,
    pub sort_by: Option<String>,
    pub facet_by: Option<Vec<String>>,
    pub page: i32,
    pub per_page: i32,
}

/// Typesense search response (Phase 2 internal type).
#[derive(Debug, Clone, Deserialize)]
pub struct TypesenseSearchResponse {
    pub found: i64,
    pub hits: Vec<serde_json::Value>,
    pub facet_counts: Option<Vec<serde_json::Value>>,
}

/// Bulk index result (Phase 2).
#[derive(Debug, Clone)]
pub struct BulkIndexResult {
    pub indexed: usize,
    pub failed: usize,
    pub errors: Vec<String>,
}

/// Typesense collection stats (Phase 2).
#[derive(Debug, Clone, Deserialize)]
pub struct CollectionStats {
    pub num_documents: i64,
    pub num_memory_shards: i32,
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
| `PostCreated` | `social::` | `PostCreatedHandler` | No-op (search_vector auto-updates) | Enqueue `IndexContentJob { collection: "social_posts", entity_id }` |
| `ListingPublished` | `mkt::` | `ListingPublishedHandler` | No-op (search_vector auto-updates) | Enqueue `IndexContentJob { collection: "marketplace_listings", entity_id }` |
| `ListingArchived` | `mkt::` | `ListingArchivedHandler` | No-op (WHERE status = 'published' excludes) | Remove document from `marketplace_listings` collection |
| `UploadPublished` | `media::` | `UploadPublishedHandler` | No-op (media not directly searchable) | Index media metadata into relevant collection |
| `FamilyDeletionScheduled` | `iam::` | `FamilyDeletionScheduledHandler` | No-op (CASCADE DELETE handles cleanup) | Remove all family documents from Typesense collections |

### Event Handler Implementations (Phase 1)

```rust
/// Phase 1: All event handlers are seams — they exist for forward compatibility
/// but perform no work because PostgreSQL's GENERATED ALWAYS columns auto-maintain
/// search vectors, and WHERE clauses exclude non-searchable rows.

pub struct PostCreatedHandler {
    search_service: Arc<dyn SearchService>,
}

#[async_trait]
impl DomainEventHandler<PostCreated> for PostCreatedHandler {
    async fn handle(&self, event: &PostCreated) -> Result<(), AppError> {
        self.search_service.handle_post_created(event).await
            .map_err(|e| AppError::internal(e.to_string()))
    }
}

pub struct ListingPublishedHandler {
    search_service: Arc<dyn SearchService>,
}

#[async_trait]
impl DomainEventHandler<ListingPublished> for ListingPublishedHandler {
    async fn handle(&self, event: &ListingPublished) -> Result<(), AppError> {
        self.search_service.handle_listing_published(event).await
            .map_err(|e| AppError::internal(e.to_string()))
    }
}

pub struct ListingArchivedHandler {
    search_service: Arc<dyn SearchService>,
}

#[async_trait]
impl DomainEventHandler<ListingArchived> for ListingArchivedHandler {
    async fn handle(&self, event: &ListingArchived) -> Result<(), AppError> {
        self.search_service.handle_listing_archived(event).await
            .map_err(|e| AppError::internal(e.to_string()))
    }
}

pub struct UploadPublishedHandler {
    search_service: Arc<dyn SearchService>,
}

#[async_trait]
impl DomainEventHandler<UploadPublished> for UploadPublishedHandler {
    async fn handle(&self, event: &UploadPublished) -> Result<(), AppError> {
        self.search_service.handle_upload_published(event).await
            .map_err(|e| AppError::internal(e.to_string()))
    }
}

pub struct FamilyDeletionScheduledHandler {
    search_service: Arc<dyn SearchService>,
}

#[async_trait]
impl DomainEventHandler<FamilyDeletionScheduled> for FamilyDeletionScheduledHandler {
    async fn handle(&self, event: &FamilyDeletionScheduled) -> Result<(), AppError> {
        self.search_service.handle_family_deletion_scheduled(event.family_id).await
            .map_err(|e| AppError::internal(e.to_string()))
    }
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
LEFT JOIN method_definitions md ON md.id = f.primary_methodology_id
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
LEFT JOIN method_definitions md ON md.id = g.methodology_id
WHERE to_tsvector('english', coalesce(g.name, '') || ' ' || coalesce(g.description, ''))
    @@ websearch_to_tsquery('english', $2)
-- Block exclusion: exclude groups created by blocked families
AND NOT EXISTS (
    SELECT 1 FROM soc_blocks sb
    WHERE (sb.blocker_family_id = $1 AND sb.blocked_family_id = g.creator_family_id)
       OR (sb.blocker_family_id = g.creator_family_id AND sb.blocked_family_id = $1)
)
-- Optional methodology filter
AND ($3::uuid IS NULL OR g.methodology_id = $3)
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
AND ($3::uuid IS NULL OR e.methodology_id = $3)
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

```rust
/// Search service with dual-backend support.
/// Learning search ALWAYS uses PostgreSQL — never Typesense.
impl SearchServiceImpl {
    async fn search(&self, auth: &AuthContext, scope: &FamilyScope, params: &SearchParams)
        -> Result<SearchResponse, SearchError>
    {
        match params.scope {
            SearchScope::Learning => {
                // ALWAYS PostgreSQL — privacy-critical, smaller dataset
                self.learning_repo.search_learning(scope, &params.q, &filters, params.limit, cursor).await
            }
            SearchScope::Marketplace => match &self.backend {
                SearchBackend::PostgresFts => {
                    self.marketplace_repo.search_listings(&params.q, &filters, params.sort, params.limit, cursor).await
                }
                SearchBackend::Typesense(adapter) => {
                    let ts_query = build_typesense_marketplace_query(params);
                    let ts_response = adapter.search("marketplace_listings", &ts_query).await?;
                    convert_typesense_to_search_response(ts_response)
                }
            }
            SearchScope::Social => match &self.backend {
                SearchBackend::PostgresFts => {
                    // Dispatch to sub-scope repositories
                    self.search_social_postgres(auth, scope, params).await
                }
                SearchBackend::Typesense(adapter) => {
                    // Typesense query with post-filter privacy enforcement
                    self.search_social_typesense(adapter, auth, scope, params).await
                }
            }
        }
    }
}
```

### §13.3 Migration Strategy (Zero-Downtime)

| Step | Action | Rollback |
|------|--------|----------|
| 1 | Deploy Typesense cluster (3-node minimum for Raft HA) | Terminate cluster |
| 2 | Create collections with schema matching source tables | Drop collections |
| 3 | Run `BulkIndexJob` to index existing PostgreSQL data | No rollback needed |
| 4 | Enable shadow mode: query both backends, compare results, log discrepancies | Disable shadow mode |
| 5 | Switch reads to Typesense (`SearchBackend::Typesense`) via config flag | Revert config to `PostgresFts` |
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

```rust
/// Indexes a single document into the Typesense collection.
/// Triggered by domain event handlers (PostCreated, ListingPublished, etc.)
/// when SearchBackend is Typesense.
///
/// Queue: search_index (dedicated queue for search jobs)
/// Retry: 3 attempts with exponential backoff (1s, 4s, 16s)
/// Timeout: 10 seconds
pub struct IndexContentJob {
    pub collection: String,      // "marketplace_listings", "social_posts", etc.
    pub entity_id: Uuid,         // Source entity ID
    pub action: IndexAction,     // Upsert or Remove
}

pub enum IndexAction {
    Upsert,
    Remove,
}
```

### §14.2 `BulkIndexJob` (Phase 2)

```rust
/// Bulk indexes all documents from a source table into a Typesense collection.
/// Used during initial migration and periodic re-indexing.
///
/// Queue: search_bulk (separate queue to avoid blocking incremental updates)
/// Retry: 1 attempt (manual re-trigger for bulk jobs)
/// Timeout: 30 minutes
/// Batch size: 1000 documents per Typesense API call
pub struct BulkIndexJob {
    pub collection: String,
    pub source_table: String,
    pub batch_size: i32,         // Default 1000
    pub cursor: Option<Uuid>,    // Resume from last processed ID
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

```rust
/// Search domain errors.
/// Internal error details are logged but never exposed in API responses.
/// [CODING_STANDARDS §4]
#[derive(Debug, thiserror::Error)]
pub enum SearchError {
    /// Query text is too short (minimum 2 characters for search, 1 for autocomplete).
    #[error("Query too short: minimum {min_length} characters required")]
    QueryTooShort { min_length: usize },

    /// Invalid search scope provided.
    #[error("Invalid search scope: {0}")]
    InvalidScope(String),

    /// Invalid sort order for the given scope.
    #[error("Sort order '{sort}' is not supported for scope '{scope}'")]
    InvalidSortForScope { sort: String, scope: String },

    /// Invalid filter value (type mismatch, out of range, etc.).
    #[error("Invalid filter value for '{field}': {reason}")]
    InvalidFilter { field: String, reason: String },

    /// Search backend is temporarily unavailable (Phase 2 — Typesense down).
    #[error("Search service temporarily unavailable")]
    BackendUnavailable,

    /// Database query error.
    #[error("Database error: {0}")]
    Database(#[from] sea_orm::DbErr),

    /// Internal error (catch-all for unexpected failures).
    #[error("Internal search error")]
    Internal(String),
}

impl SearchError {
    /// Map SearchError to HTTP status code.
    /// Internal details are never exposed to clients.
    pub fn status_code(&self) -> StatusCode {
        match self {
            Self::QueryTooShort { .. } => StatusCode::UNPROCESSABLE_ENTITY,  // 422
            Self::InvalidScope(_) => StatusCode::BAD_REQUEST,                 // 400
            Self::InvalidSortForScope { .. } => StatusCode::BAD_REQUEST,      // 400
            Self::InvalidFilter { .. } => StatusCode::BAD_REQUEST,            // 400
            Self::BackendUnavailable => StatusCode::SERVICE_UNAVAILABLE,      // 503
            Self::Database(_) => StatusCode::INTERNAL_SERVER_ERROR,           // 500
            Self::Internal(_) => StatusCode::INTERNAL_SERVER_ERROR,           // 500
        }
    }
}
```

| Error Variant | HTTP Status | Client Message |
|--------------|-------------|----------------|
| `QueryTooShort` | 422 | `"Query too short: minimum {n} characters required"` |
| `InvalidScope` | 400 | `"Invalid search scope"` |
| `InvalidSortForScope` | 400 | `"Sort order not supported for this scope"` |
| `InvalidFilter` | 400 | `"Invalid filter value"` |
| `BackendUnavailable` | 503 | `"Search service temporarily unavailable"` |
| `Database` | 500 | `"An internal error occurred"` (log actual error internally) |
| `Internal` | 500 | `"An internal error occurred"` (log actual error internally) |

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
| `PostCreated` | No-op: `soc_posts.search_vector` auto-updates via `GENERATED ALWAYS` | Enqueue `IndexContentJob { collection: "social_posts", entity_id: event.post_id, action: Upsert }` |
| `ListingPublished` | No-op: `mkt_listings.search_vector` auto-updates; listing appears when `status = 'published'` | Enqueue `IndexContentJob { collection: "marketplace_listings", entity_id: event.listing_id, action: Upsert }` |
| `ListingArchived` | No-op: `WHERE status = 'published'` excludes archived listings | Enqueue `IndexContentJob { collection: "marketplace_listings", entity_id: event.listing_id, action: Remove }` |
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

- `GET /v1/search/suggestions` — AI-powered search suggestions (depends on `ai::` domain)
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

```
src/search/
├── mod.rs                 # Module root — re-exports public types
├── handlers.rs            # Axum handlers (thin: extractors → service → response)
├── service.rs             # SearchServiceImpl (CQRS query + command dispatch)
├── models.rs              # Request/response DTOs (SearchParams, SearchResponse, etc.)
├── error.rs               # SearchError thiserror enum
├── repository/
│   ├── mod.rs             # Re-exports repository traits
│   ├── social.rs          # SocialSearchRepository impl (PostgreSQL FTS)
│   ├── marketplace.rs     # MarketplaceSearchRepository impl (PostgreSQL FTS)
│   ├── learning.rs        # LearningSearchRepository impl (PostgreSQL FTS)
│   └── autocomplete.rs    # AutocompleteRepository impl (pg_trgm + ILIKE)
├── events/
│   ├── mod.rs             # Re-exports event handlers
│   ├── post_created.rs    # PostCreatedHandler (Phase 1 no-op)
│   ├── listing_published.rs  # ListingPublishedHandler (Phase 1 no-op)
│   ├── listing_archived.rs   # ListingArchivedHandler (Phase 1 no-op)
│   ├── upload_published.rs   # UploadPublishedHandler (Phase 1 no-op)
│   └── family_deletion.rs    # FamilyDeletionScheduledHandler (Phase 1 no-op)
└── adapters/
    └── typesense.rs       # TypesenseAdapter impl (Phase 2)
```

> **Note on entities/**: Search has no owned entities in Phase 1 (no SeaORM-generated entity
> files). The `search_index_state` table entity will be generated in Phase 2 when the table
> becomes active. The `entities/` directory is omitted until then.
