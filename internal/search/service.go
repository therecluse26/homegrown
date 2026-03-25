package search

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Service Implementation ──────────────────────────────────────────────────

type searchServiceImpl struct {
	socialRepo       SocialSearchRepository
	marketplaceRepo  MarketplaceSearchRepository
	learningRepo     LearningSearchRepository
	autocompleteRepo AutocompleteRepository
	backend          SearchBackend
}

// NewSearchService creates a new SearchService. [12-search §5]
func NewSearchService(
	socialRepo SocialSearchRepository,
	marketplaceRepo MarketplaceSearchRepository,
	learningRepo LearningSearchRepository,
	autocompleteRepo AutocompleteRepository,
) SearchService {
	return &searchServiceImpl{
		socialRepo:       socialRepo,
		marketplaceRepo:  marketplaceRepo,
		learningRepo:     learningRepo,
		autocompleteRepo: autocompleteRepo,
		backend:          SearchBackendPostgresFts,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Search
// ═══════════════════════════════════════════════════════════════════════════════

func (s *searchServiceImpl) Search(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params *SearchParams) (*SearchResponse, error) {
	// ── Validation ────────────────────────────────────────────────────
	if len(params.Q) < 2 {
		return nil, &SearchError{Err: ErrQueryTooShort, Field: "q"}
	}
	if !isValidScope(params.Scope) {
		return nil, &SearchError{Err: ErrInvalidScope, Field: "scope"}
	}
	if params.Sort != "" && params.Sort != SearchSortRelevance && params.Scope != SearchScopeMarketplace {
		return nil, &SearchError{Err: ErrInvalidSortForScope, Field: "sort"}
	}

	// ── Defaults ─────────────────────────────────────────────────────
	if params.Limit <= 0 {
		params.Limit = 20
	}
	if params.Limit > 50 {
		params.Limit = 50
	}
	if params.Sort == "" {
		params.Sort = SearchSortRelevance
	}

	// ── Dispatch by scope with dual-backend routing [12-search §13.2] ──
	switch params.Scope {
	case SearchScopeLearning:
		// ALWAYS PostgreSQL — privacy-critical, smaller dataset
		return s.searchLearning(ctx, scope, params)

	case SearchScopeMarketplace:
		switch s.backend {
		case SearchBackendPostgresFts:
			return s.searchMarketplace(ctx, params)
		case SearchBackendTypesense:
			return nil, &SearchError{Err: ErrBackendUnavailable}
		}

	case SearchScopeSocial:
		switch s.backend {
		case SearchBackendPostgresFts:
			return s.searchSocial(ctx, auth, params)
		case SearchBackendTypesense:
			return nil, &SearchError{Err: ErrBackendUnavailable}
		}
	}

	return nil, &SearchError{Err: ErrInvalidScope}
}

// ─── Social Search ───────────────────────────────────────────────────────────

// searchSocialPostgres implements social search via PostgreSQL FTS. [12-search §13.2]
func (s *searchServiceImpl) searchSocial(ctx context.Context, auth *shared.AuthContext, params *SearchParams) (*SearchResponse, error) {
	familyID := auth.FamilyID

	if params.SubScope != nil {
		var results []SocialSearchResult
		var err error
		switch *params.SubScope {
		case SocialSubScopeFamilies:
			results, err = s.socialRepo.SearchFamilies(ctx, familyID, params.Q, params.Limit, params.Cursor)
		case SocialSubScopeGroups:
			results, err = s.socialRepo.SearchGroups(ctx, familyID, params.Q, params.MethodologySlug, params.Limit, params.Cursor)
		case SocialSubScopeEvents:
			results, err = s.socialRepo.SearchEvents(ctx, familyID, params.Q, params.MethodologySlug, params.Limit, params.Cursor)
		default:
			return nil, &SearchError{Err: ErrInvalidScope, Field: "sub_scope"}
		}
		if err != nil {
			return nil, err
		}
		return assembleSocialResponse(results), nil
	}

	// No sub_scope → search all four entity types, merge by relevance
	families, err := s.socialRepo.SearchFamilies(ctx, familyID, params.Q, params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	groups, err := s.socialRepo.SearchGroups(ctx, familyID, params.Q, params.MethodologySlug, params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	events, err := s.socialRepo.SearchEvents(ctx, familyID, params.Q, params.MethodologySlug, params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}
	posts, err := s.socialRepo.SearchPosts(ctx, familyID, params.Q, params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}

	merged := make([]SocialSearchResult, 0, len(families)+len(groups)+len(events)+len(posts))
	merged = append(merged, families...)
	merged = append(merged, groups...)
	merged = append(merged, events...)
	merged = append(merged, posts...)

	// Sort by relevance descending
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Relevance > merged[j].Relevance
	})

	// Cap at limit
	if len(merged) > params.Limit {
		merged = merged[:params.Limit]
	}

	return assembleSocialResponse(merged), nil
}

func assembleSocialResponse(results []SocialSearchResult) *SearchResponse {
	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = r.Result
	}
	return &SearchResponse{
		Results:    searchResults,
		TotalCount: int64(len(searchResults)),
	}
}

// ─── Marketplace Search ──────────────────────────────────────────────────────

func (s *searchServiceImpl) searchMarketplace(ctx context.Context, params *SearchParams) (*SearchResponse, error) {
	filters := extractMarketplaceFilters(params)

	mktResults, err := s.marketplaceRepo.SearchListings(ctx, params.Q, filters, params.Sort, params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}

	facets, err := s.marketplaceRepo.CountFacets(ctx, params.Q, filters)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, len(mktResults.Listings))
	for i, l := range mktResults.Listings {
		listing := l
		results[i] = SearchResult{Type: "listing", Listing: &listing}
	}

	return &SearchResponse{
		Results:    results,
		TotalCount: mktResults.TotalCount,
		Facets:     facets,
		NextCursor: mktResults.NextCursor,
	}, nil
}

func extractMarketplaceFilters(params *SearchParams) *MarketplaceSearchFilters {
	return &MarketplaceSearchFilters{
		MethodologyTags: params.MethodologyTags,
		SubjectTags:     params.SubjectTags,
		GradeMin:        params.GradeMin,
		GradeMax:        params.GradeMax,
		PriceMin:        params.PriceMin,
		PriceMax:        params.PriceMax,
		ContentType:     params.ContentType,
		WorldviewTags:   params.WorldviewTags,
		FreeOnly:        params.FreeOnly,
	}
}

// ─── Learning Search ─────────────────────────────────────────────────────────

func (s *searchServiceImpl) searchLearning(ctx context.Context, scope *shared.FamilyScope, params *SearchParams) (*SearchResponse, error) {
	filters := extractLearningFilters(params)

	results, err := s.learningRepo.SearchLearning(ctx, scope, params.Q, filters, params.Limit, params.Cursor)
	if err != nil {
		return nil, err
	}

	searchResults := make([]SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = r.Result
	}

	return &SearchResponse{
		Results:    searchResults,
		TotalCount: int64(len(searchResults)),
	}, nil
}

func extractLearningFilters(params *SearchParams) *LearningSearchFilters {
	return &LearningSearchFilters{
		StudentID:   params.StudentID,
		SourceType:  params.SourceType,
		DateFrom:    params.DateFrom,
		DateTo:      params.DateTo,
		SubjectTags: params.SubjectTags,
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Autocomplete
// ═══════════════════════════════════════════════════════════════════════════════

func (s *searchServiceImpl) Autocomplete(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params *AutocompleteParams) (*AutocompleteResponse, error) {
	// ── Validation ────────────────────────────────────────────────────
	if len(params.Q) < 1 {
		return nil, &SearchError{Err: ErrQueryTooShort, Field: "q"}
	}

	// ── Defaults ─────────────────────────────────────────────────────
	if params.Limit <= 0 {
		params.Limit = 5
	}
	if params.Limit > 10 {
		params.Limit = 10
	}
	if params.Scope == nil {
		defaultScope := SearchScopeMarketplace
		params.Scope = &defaultScope
	}

	// ── Dispatch ─────────────────────────────────────────────────────
	var suggestions []AutocompleteSuggestion
	var err error

	switch *params.Scope {
	case SearchScopeMarketplace:
		suggestions, err = s.autocompleteRepo.AutocompleteMarketplace(ctx, params.Q, params.Limit)
	case SearchScopeSocial:
		suggestions, err = s.autocompleteRepo.AutocompleteSocial(ctx, auth.FamilyID, params.Q, params.Limit)
	case SearchScopeLearning:
		suggestions, err = s.autocompleteRepo.AutocompleteLearning(ctx, scope, params.Q, params.Limit)
	default:
		return nil, &SearchError{Err: ErrInvalidScope, Field: "scope"}
	}
	if err != nil {
		return nil, err
	}

	if suggestions == nil {
		suggestions = []AutocompleteSuggestion{}
	}

	return &AutocompleteResponse{Suggestions: suggestions}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Event Handlers — Phase 1 No-ops [12-search §9]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *searchServiceImpl) HandlePostCreated(_ context.Context, _ *PostCreated) error              { return nil }
func (s *searchServiceImpl) HandleListingPublished(_ context.Context, _ *ListingPublished) error    { return nil }
func (s *searchServiceImpl) HandleListingArchived(_ context.Context, _ *ListingArchived) error      { return nil }
func (s *searchServiceImpl) HandleUploadPublished(_ context.Context, _ *UploadPublished) error      { return nil }
func (s *searchServiceImpl) HandleFamilyDeletionScheduled(_ context.Context, _ uuid.UUID) error     { return nil }

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func isValidScope(scope SearchScope) bool {
	switch scope {
	case SearchScopeSocial, SearchScopeMarketplace, SearchScopeLearning:
		return true
	default:
		return false
	}
}
