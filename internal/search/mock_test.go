package search

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Mock Service (for handler tests)
// ═══════════════════════════════════════════════════════════════════════════════

type mockSearchService struct {
	searchFn       func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params *SearchParams) (*SearchResponse, error)
	autocompleteFn func(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params *AutocompleteParams) (*AutocompleteResponse, error)
}

func newMockSearchService() *mockSearchService { return &mockSearchService{} }

func (m *mockSearchService) Search(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params *SearchParams) (*SearchResponse, error) {
	if m.searchFn != nil {
		return m.searchFn(ctx, auth, scope, params)
	}
	panic("Search not mocked")
}

func (m *mockSearchService) Autocomplete(ctx context.Context, auth *shared.AuthContext, scope *shared.FamilyScope, params *AutocompleteParams) (*AutocompleteResponse, error) {
	if m.autocompleteFn != nil {
		return m.autocompleteFn(ctx, auth, scope, params)
	}
	panic("Autocomplete not mocked")
}

func (m *mockSearchService) HandlePostCreated(context.Context, *PostCreated) error              { return nil }
func (m *mockSearchService) HandleListingPublished(context.Context, *ListingPublished) error    { return nil }
func (m *mockSearchService) HandleListingArchived(context.Context, *ListingArchived) error      { return nil }
func (m *mockSearchService) HandleUploadPublished(context.Context, *UploadPublished) error      { return nil }
func (m *mockSearchService) HandleFamilyDeletionScheduled(context.Context, uuid.UUID) error     { return nil }

// ═══════════════════════════════════════════════════════════════════════════════
// Mock Repositories (for service tests)
// ═══════════════════════════════════════════════════════════════════════════════

// ─── Social ──────────────────────────────────────────────────────────────────

type stubSocialRepo struct {
	searchFamiliesFn func(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error)
	searchGroupsFn   func(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologyID *uuid.UUID, limit int, cursor *string) ([]SocialSearchResult, error)
	searchEventsFn   func(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologyID *uuid.UUID, limit int, cursor *string) ([]SocialSearchResult, error)
	searchPostsFn    func(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error)
}

func (s *stubSocialRepo) SearchFamilies(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error) {
	if s.searchFamiliesFn != nil {
		return s.searchFamiliesFn(ctx, searcherFamilyID, query, limit, cursor)
	}
	return nil, nil
}

func (s *stubSocialRepo) SearchGroups(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologyID *uuid.UUID, limit int, cursor *string) ([]SocialSearchResult, error) {
	if s.searchGroupsFn != nil {
		return s.searchGroupsFn(ctx, searcherFamilyID, query, methodologyID, limit, cursor)
	}
	return nil, nil
}

func (s *stubSocialRepo) SearchEvents(ctx context.Context, searcherFamilyID uuid.UUID, query string, methodologyID *uuid.UUID, limit int, cursor *string) ([]SocialSearchResult, error) {
	if s.searchEventsFn != nil {
		return s.searchEventsFn(ctx, searcherFamilyID, query, methodologyID, limit, cursor)
	}
	return nil, nil
}

func (s *stubSocialRepo) SearchPosts(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int, cursor *string) ([]SocialSearchResult, error) {
	if s.searchPostsFn != nil {
		return s.searchPostsFn(ctx, searcherFamilyID, query, limit, cursor)
	}
	return nil, nil
}

// ─── Marketplace ─────────────────────────────────────────────────────────────

type stubMarketplaceRepo struct {
	searchListingsFn func(ctx context.Context, query string, filters *MarketplaceSearchFilters, sort SearchSortOrder, limit int, cursor *string) (*MarketplaceSearchResults, error)
	countFacetsFn    func(ctx context.Context, query string, filters *MarketplaceSearchFilters) (*FacetCounts, error)
}

func (s *stubMarketplaceRepo) SearchListings(ctx context.Context, query string, filters *MarketplaceSearchFilters, sort SearchSortOrder, limit int, cursor *string) (*MarketplaceSearchResults, error) {
	if s.searchListingsFn != nil {
		return s.searchListingsFn(ctx, query, filters, sort, limit, cursor)
	}
	return &MarketplaceSearchResults{}, nil
}

func (s *stubMarketplaceRepo) CountFacets(ctx context.Context, query string, filters *MarketplaceSearchFilters) (*FacetCounts, error) {
	if s.countFacetsFn != nil {
		return s.countFacetsFn(ctx, query, filters)
	}
	return &FacetCounts{}, nil
}

// ─── Learning ────────────────────────────────────────────────────────────────

type stubLearningRepo struct {
	searchLearningFn func(ctx context.Context, familyScope *shared.FamilyScope, query string, filters *LearningSearchFilters, limit int, cursor *string) ([]LearningSearchResult, error)
}

func (s *stubLearningRepo) SearchLearning(ctx context.Context, familyScope *shared.FamilyScope, query string, filters *LearningSearchFilters, limit int, cursor *string) ([]LearningSearchResult, error) {
	if s.searchLearningFn != nil {
		return s.searchLearningFn(ctx, familyScope, query, filters, limit, cursor)
	}
	return nil, nil
}

// ─── Autocomplete ────────────────────────────────────────────────────────────

type stubAutocompleteRepo struct {
	autocompleteMarketplaceFn func(ctx context.Context, query string, limit int) ([]AutocompleteSuggestion, error)
	autocompleteSocialFn      func(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int) ([]AutocompleteSuggestion, error)
	autocompleteLearningFn    func(ctx context.Context, familyScope *shared.FamilyScope, query string, limit int) ([]AutocompleteSuggestion, error)
}

func (s *stubAutocompleteRepo) AutocompleteMarketplace(ctx context.Context, query string, limit int) ([]AutocompleteSuggestion, error) {
	if s.autocompleteMarketplaceFn != nil {
		return s.autocompleteMarketplaceFn(ctx, query, limit)
	}
	return nil, nil
}

func (s *stubAutocompleteRepo) AutocompleteSocial(ctx context.Context, searcherFamilyID uuid.UUID, query string, limit int) ([]AutocompleteSuggestion, error) {
	if s.autocompleteSocialFn != nil {
		return s.autocompleteSocialFn(ctx, searcherFamilyID, query, limit)
	}
	return nil, nil
}

func (s *stubAutocompleteRepo) AutocompleteLearning(ctx context.Context, familyScope *shared.FamilyScope, query string, limit int) ([]AutocompleteSuggestion, error) {
	if s.autocompleteLearningFn != nil {
		return s.autocompleteLearningFn(ctx, familyScope, query, limit)
	}
	return nil, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func testAuth() *shared.AuthContext {
	return &shared.AuthContext{
		FamilyID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		ParentID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
	}
}

func testFamilyScope() *shared.FamilyScope {
	auth := testAuth()
	scope := shared.NewFamilyScopeFromAuth(auth)
	return &scope
}
