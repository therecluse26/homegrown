package search

import (
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Enums [12-search §8.1]
// ═══════════════════════════════════════════════════════════════════════════════

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

// SearchBackend routes queries to PostgreSQL FTS or Typesense. [12-search §8.3]
type SearchBackend int

const (
	SearchBackendPostgresFts SearchBackend = iota
	SearchBackendTypesense
)

// ═══════════════════════════════════════════════════════════════════════════════
// Request Types [12-search §8.1]
// ═══════════════════════════════════════════════════════════════════════════════

// SearchParams is the unified search request parameters.
type SearchParams struct {
	Q     string      `json:"q" query:"q"`
	Scope SearchScope `json:"scope" query:"scope"`
	// Cursor for pagination (opaque string from previous response).
	Cursor *string `json:"cursor,omitempty" query:"cursor"`
	// Limit is results per page (default 20, max 50).
	Limit int `json:"limit" query:"limit"`
	// Sort order (marketplace only, ignored for other scopes).
	Sort SearchSortOrder `json:"sort,omitempty" query:"sort"`

	// ── Social-specific filters ────────────────────────────────────────
	SubScope      *SocialSubScope `json:"sub_scope,omitempty" query:"sub_scope"`
	MethodologyID *uuid.UUID      `json:"methodology_id,omitempty" query:"methodology_id"`

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
}

// MarketplaceSearchFilters is extracted from SearchParams for the marketplace repository.
type MarketplaceSearchFilters struct {
	MethodologyTags []uuid.UUID
	SubjectTags     []string
	GradeMin        *int16
	GradeMax        *int16
	PriceMin        *int32
	PriceMax        *int32
	ContentType     *string
	WorldviewTags   []string
	FreeOnly        *bool
}

// LearningSearchFilters is extracted from SearchParams for the learning repository.
type LearningSearchFilters struct {
	StudentID   *uuid.UUID
	SourceType  *LearningSourceType
	DateFrom    *time.Time
	DateTo      *time.Time
	SubjectTags []string
}

// AutocompleteParams is the autocomplete request parameters.
type AutocompleteParams struct {
	Q     string       `json:"q" query:"q"`
	Scope *SearchScope `json:"scope,omitempty" query:"scope"`
	Limit int          `json:"limit,omitempty" query:"limit"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Types [12-search §8.2]
// ═══════════════════════════════════════════════════════════════════════════════

// SearchResponse is the unified search response.
type SearchResponse struct {
	Results    []SearchResult `json:"results"`
	TotalCount int64          `json:"total_count"`
	Facets     *FacetCounts   `json:"facets,omitempty"`
	NextCursor *string        `json:"next_cursor,omitempty"`
}

// SearchResult is a polymorphic search result — tagged union by type.
type SearchResult struct {
	Type string `json:"type"`

	// Social
	Family *FamilySearchResult `json:"family,omitempty"`
	Group  *GroupSearchResult  `json:"group,omitempty"`
	Event  *EventSearchResult  `json:"event,omitempty"`
	Post   *PostSearchResult   `json:"post,omitempty"`

	// Marketplace
	Listing *ListingSearchResult `json:"listing,omitempty"`

	// Learning
	Activity    *ActivitySearchResult    `json:"activity,omitempty"`
	Journal     *JournalSearchResult     `json:"journal,omitempty"`
	ReadingItem *ReadingItemSearchResult `json:"reading_item,omitempty"`
}

// SocialSearchResult is returned by SocialSearchRepository methods.
type SocialSearchResult struct {
	Result    SearchResult
	Relevance float32
}

// LearningSearchResult is returned by LearningSearchRepository.
type LearningSearchResult struct {
	Result    SearchResult
	Relevance float32
}

// ─── Scope-Specific Result Types ─────────────────────────────────────────────

type FamilySearchResult struct {
	FamilyID        uuid.UUID `json:"family_id"`
	DisplayName     string    `json:"display_name"`
	MethodologyName *string   `json:"methodology_name,omitempty"`
	LocationRegion  *string   `json:"location_region,omitempty"`
	IsFriend        bool      `json:"is_friend"`
	Relevance       float32   `json:"relevance"`
}

type GroupSearchResult struct {
	GroupID         uuid.UUID `json:"group_id"`
	Name            string    `json:"name"`
	Description     *string   `json:"description,omitempty"`
	MemberCount     int32     `json:"member_count"`
	MethodologyName *string   `json:"methodology_name,omitempty"`
	Relevance       float32   `json:"relevance"`
}

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

type PostSearchResult struct {
	PostID            uuid.UUID `json:"post_id"`
	ContentSnippet    string    `json:"content_snippet"`
	AuthorFamilyID    uuid.UUID `json:"author_family_id"`
	AuthorDisplayName string    `json:"author_display_name"`
	GroupName         *string   `json:"group_name,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	Relevance         float32   `json:"relevance"`
}

type ListingSearchResult struct {
	ListingID          uuid.UUID   `json:"listing_id"`
	Title              string      `json:"title"`
	DescriptionSnippet string      `json:"description_snippet"`
	PriceCents         int32       `json:"price_cents"`
	ContentType        string      `json:"content_type"`
	RatingAvg          *float64    `json:"rating_avg,omitempty"`
	RatingCount        int32       `json:"rating_count"`
	PublisherName      string      `json:"publisher_name"`
	MethodologyTags    []uuid.UUID `json:"methodology_tags"`
	SubjectTags        []string    `json:"subject_tags"`
	PublishedAt        time.Time   `json:"published_at"`
	Relevance          float32     `json:"relevance"`
}

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

// ─── Facets ──────────────────────────────────────────────────────────────────

type FacetCounts struct {
	MethodologyTags []FacetBucket `json:"methodology_tags"`
	SubjectTags     []FacetBucket `json:"subject_tags"`
	ContentType     []FacetBucket `json:"content_type"`
	WorldviewTags   []FacetBucket `json:"worldview_tags"`
	PriceRanges     []FacetBucket `json:"price_ranges"`
	RatingRanges    []FacetBucket `json:"rating_ranges"`
}

type FacetBucket struct {
	Value       string `json:"value"`
	DisplayName string `json:"display_name"`
	Count       int64  `json:"count"`
}

// ─── Autocomplete ────────────────────────────────────────────────────────────

type AutocompleteResponse struct {
	Suggestions []AutocompleteSuggestion `json:"suggestions"`
}

type AutocompleteSuggestion struct {
	Text       string    `json:"text"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Score      float32   `json:"score"`
}

// ─── Typesense Internal Types [12-search §8.3] ──────────────────────────────

// TypesenseSearchQuery is a Typesense search query (Phase 2 internal type).
type TypesenseSearchQuery struct {
	Q        string   `json:"q"`
	QueryBy  []string `json:"query_by"`
	FilterBy *string  `json:"filter_by,omitempty"`
	SortBy   *string  `json:"sort_by,omitempty"`
	FacetBy  []string `json:"facet_by,omitempty"`
	Page     int      `json:"page"`
	PerPage  int      `json:"per_page"`
}

// TypesenseSearchResponse is a Typesense search response (Phase 2 internal type).
type TypesenseSearchResponse struct {
	Found       int64            `json:"found"`
	Hits        []map[string]any `json:"hits"`
	FacetCounts []map[string]any `json:"facet_counts,omitempty"`
}

// BulkIndexResult is a bulk index result (Phase 2).
type BulkIndexResult struct {
	Indexed int      `json:"indexed"`
	Failed  int      `json:"failed"`
	Errors  []string `json:"errors"`
}

// CollectionStats holds Typesense collection stats (Phase 2).
type CollectionStats struct {
	NumDocuments    int64 `json:"num_documents"`
	NumMemoryShards int32 `json:"num_memory_shards"`
}

// ─── Background Job Types [12-search §14] ────────────────────────────────────

// IndexAction defines the action for an IndexContentJob.
type IndexAction string

const (
	IndexActionUpsert IndexAction = "upsert"
	IndexActionRemove IndexAction = "remove"
)

// IndexContentJob indexes a single document into a Typesense collection (Phase 2).
// Triggered by domain event handlers when SearchBackend is Typesense.
type IndexContentJob struct {
	Collection string      `json:"collection"`
	EntityID   uuid.UUID   `json:"entity_id"`
	Action     IndexAction `json:"action"`
}

// BulkIndexJob bulk indexes all documents from a source table into a Typesense collection (Phase 2).
type BulkIndexJob struct {
	Collection  string     `json:"collection"`
	SourceTable string     `json:"source_table"`
	BatchSize   int        `json:"batch_size"`
	Cursor      *uuid.UUID `json:"cursor"`
}

// ─── Phase 3 Suggestions Types [12-search §4.3] ─────────────────────────────

// SearchSuggestionsResponse is the response for GET /v1/search/suggestions.
type SearchSuggestionsResponse struct {
	Suggestions []SearchSuggestion `json:"suggestions"`
}

// SearchSuggestion is an AI-powered search suggestion (Phase 3, depends on recs:: domain).
type SearchSuggestion struct {
	Text       string    `json:"text"`
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Score      float32   `json:"score"`
}

// ─── Marketplace Search Results (internal container) ─────────────────────────

type MarketplaceSearchResults struct {
	Listings   []ListingSearchResult `json:"listings"`
	TotalCount int64                 `json:"total_count"`
	NextCursor *string               `json:"next_cursor,omitempty"`
}

// ─── Event Types (consumed from other domains) ───────────────────────────────

// PostCreated is the search domain's local type for social::PostCreated events.
type PostCreated struct {
	PostID   uuid.UUID `json:"post_id"`
	FamilyID uuid.UUID `json:"family_id"`
}

// ListingPublished is the search domain's local type for mkt::ListingPublished events.
type ListingPublished struct {
	ListingID uuid.UUID `json:"listing_id"`
}

// ListingArchived is the search domain's local type for mkt::ListingArchived events.
type ListingArchived struct {
	ListingID uuid.UUID `json:"listing_id"`
}

// UploadPublished is the search domain's local type for media::UploadPublished events.
type UploadPublished struct {
	UploadID uuid.UUID `json:"upload_id"`
}
