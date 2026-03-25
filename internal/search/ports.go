package search

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [12-search §5]
// CQRS: query side (search + autocomplete) and command side (index updates).
// Phase 1: command-side methods are no-ops.
// ═══════════════════════════════════════════════════════════════════════════════

// SearchService defines all search domain use cases.
type SearchService interface {
	// ── Query Side ─────────────────────────────────────────────────────────

	// Search performs unified search across a single scope with privacy enforcement.
	Search(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params *SearchParams) (*SearchResponse, error)

	// Autocomplete returns type-ahead suggestions.
	Autocomplete(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params *AutocompleteParams) (*AutocompleteResponse, error)

	// ── Command Side (Index Updates — Phase 1 no-ops) ─────────────────────

	HandlePostCreated(ctx context.Context, event *PostCreated) error
	HandleListingPublished(ctx context.Context, event *ListingPublished) error
	HandleListingArchived(ctx context.Context, event *ListingArchived) error
	HandleUploadPublished(ctx context.Context, event *UploadPublished) error
	HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [12-search §6]
// All repositories are query-only (no mutations) in Phase 1.
// ═══════════════════════════════════════════════════════════════════════════════

// SocialSearchRepository queries soc_posts, soc_groups, soc_events, iam_families. [12-search §6.1]
type SocialSearchRepository interface {
	SearchFamilies(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error)
	SearchGroups(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologyID *uuid.UUID, limit int, cursor *string) ([]SocialSearchResult, error)
	SearchEvents(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologyID *uuid.UUID, limit int, cursor *string) ([]SocialSearchResult, error)

	// SearchPosts searches posts by content.
	// Privacy: friends-only posts visible to friends, group posts visible to group members.
	// Block exclusion: posts by blocked families are excluded bidirectionally.
	SearchPosts(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error)
}

// MarketplaceSearchRepository queries mkt_listings with faceted filtering. [12-search §6.2]
type MarketplaceSearchRepository interface {
	SearchListings(ctx context.Context, query string, filters *MarketplaceSearchFilters, sort SearchSortOrder, limit int, cursor *string) (*MarketplaceSearchResults, error)
	CountFacets(ctx context.Context, query string, filters *MarketplaceSearchFilters) (*FacetCounts, error)
}

// LearningSearchRepository queries learn_activity_logs, learn_journal_entries, learn_reading_items. [12-search §6.3]
type LearningSearchRepository interface {
	SearchLearning(ctx context.Context, familyScope *shared.FamilyScope, query string, filters *LearningSearchFilters, limit int, cursor *string) ([]LearningSearchResult, error)
}

// AutocompleteRepository provides fast prefix/fuzzy matching for type-ahead. [12-search §6.4]
type AutocompleteRepository interface {
	AutocompleteMarketplace(ctx context.Context, query string, limit int) ([]AutocompleteSuggestion, error)
	AutocompleteSocial(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int) ([]AutocompleteSuggestion, error)
	AutocompleteLearning(ctx context.Context, familyScope *shared.FamilyScope, query string, limit int) ([]AutocompleteSuggestion, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Adapter Interface [12-search §7]
// ═══════════════════════════════════════════════════════════════════════════════

// TypesenseAdapter is the Phase 2+ Typesense search engine adapter. [12-search §7.1]
// Typesense replaces Meilisearch (originally in ARCH §2.6) because:
// - Built-in Raft-based HA clustering in the open-source edition
// - Sub-50ms search latency (C++ engine), field weighting for relevance tuning
// - Production-proven clustering (3 or 5 node) without enterprise licensing
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
