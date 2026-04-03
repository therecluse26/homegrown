package search

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 1: Error Types
// ═══════════════════════════════════════════════════════════════════════════════

func TestSearchError_StatusCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"query too short → 422", ErrQueryTooShort, 422},
		{"invalid scope → 400", ErrInvalidScope, 400},
		{"invalid sort for scope → 400", ErrInvalidSortForScope, 400},
		{"invalid filter → 400", ErrInvalidFilter, 400},
		{"backend unavailable → 503", ErrBackendUnavailable, 503},
		{"unknown → 500", errors.New("unexpected"), 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &SearchError{Err: tt.err}
			if got := se.StatusCode(); got != tt.wantCode {
				t.Errorf("StatusCode() = %d, want %d", got, tt.wantCode)
			}
		})
	}
}

func TestSearchError_Unwrap(t *testing.T) {
	se := &SearchError{Err: ErrQueryTooShort}
	if !errors.Is(se, ErrQueryTooShort) {
		t.Error("expected errors.Is to match ErrQueryTooShort")
	}
}

func TestSearchError_ErrorWithField(t *testing.T) {
	se := &SearchError{Err: ErrInvalidFilter, Field: "price_max", Reason: "must be positive"}
	got := se.Error()
	if got != "invalid filter value: field=price_max reason=must be positive" {
		t.Errorf("unexpected error string: %s", got)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 2: Validation (#1–#6)
// ═══════════════════════════════════════════════════════════════════════════════

func newTestService() (*searchServiceImpl, *stubSocialRepo, *stubMarketplaceRepo, *stubLearningRepo, *stubAutocompleteRepo) {
	social := &stubSocialRepo{}
	mkt := &stubMarketplaceRepo{}
	learn := &stubLearningRepo{}
	ac := &stubAutocompleteRepo{}
	svc := NewSearchService(social, mkt, learn, ac, &stubTypesenseAdapter{}).(*searchServiceImpl)
	return svc, social, mkt, learn, ac
}

func TestSearch_QueryTooShort_OneChar(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "a", Scope: SearchScopeMarketplace,
	})
	assertSearchError(t, err, ErrQueryTooShort)
}

func TestSearch_QueryTooShort_Empty(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "", Scope: SearchScopeMarketplace,
	})
	assertSearchError(t, err, ErrQueryTooShort)
}

func TestSearch_InvalidScope(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test query", Scope: "invalid",
	})
	assertSearchError(t, err, ErrInvalidScope)
}

func TestSearch_SortOnNonMarketplaceScope(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test query", Scope: SearchScopeSocial, Sort: SearchSortPriceAsc,
	})
	assertSearchError(t, err, ErrInvalidSortForScope)
}

func TestSearch_ValidSortOnMarketplace(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	resp, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test query", Scope: SearchScopeMarketplace, Sort: SearchSortPriceAsc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestAutocomplete_QueryTooShort(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	scope := SearchScopeMarketplace
	_, err := svc.Autocomplete(context.Background(), testAuth(), testFamilyScope(), &AutocompleteParams{
		Q: "", Scope: &scope,
	})
	assertSearchError(t, err, ErrQueryTooShort)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 3: Defaults (#7–#12)
// ═══════════════════════════════════════════════════════════════════════════════

func TestSearch_DefaultLimit(t *testing.T) {
	svc, _, mktRepo, _, _ := newTestService()
	var capturedLimit int
	mktRepo.searchListingsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters, _ SearchSortOrder, limit int, _ *string) (*MarketplaceSearchResults, error) {
		capturedLimit = limit
		return &MarketplaceSearchResults{}, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeMarketplace, Limit: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != 20 {
		t.Errorf("expected default limit 20, got %d", capturedLimit)
	}
}

func TestSearch_CapsLimitAt50(t *testing.T) {
	svc, _, mktRepo, _, _ := newTestService()
	var capturedLimit int
	mktRepo.searchListingsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters, _ SearchSortOrder, limit int, _ *string) (*MarketplaceSearchResults, error) {
		capturedLimit = limit
		return &MarketplaceSearchResults{}, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeMarketplace, Limit: 100,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != 50 {
		t.Errorf("expected capped limit 50, got %d", capturedLimit)
	}
}

func TestSearch_DefaultSort(t *testing.T) {
	svc, _, mktRepo, _, _ := newTestService()
	var capturedSort SearchSortOrder
	mktRepo.searchListingsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters, sort SearchSortOrder, _ int, _ *string) (*MarketplaceSearchResults, error) {
		capturedSort = sort
		return &MarketplaceSearchResults{}, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeMarketplace,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedSort != SearchSortRelevance {
		t.Errorf("expected default sort 'relevance', got %q", capturedSort)
	}
}

func TestAutocomplete_DefaultLimit(t *testing.T) {
	svc, _, _, _, acRepo := newTestService()
	var capturedLimit int
	acRepo.autocompleteMarketplaceFn = func(_ context.Context, _ string, limit int) ([]AutocompleteSuggestion, error) {
		capturedLimit = limit
		return nil, nil
	}
	scope := SearchScopeMarketplace
	_, err := svc.Autocomplete(context.Background(), testAuth(), testFamilyScope(), &AutocompleteParams{
		Q: "t", Scope: &scope, Limit: 0,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != 5 {
		t.Errorf("expected default autocomplete limit 5, got %d", capturedLimit)
	}
}

func TestAutocomplete_CapsLimitAt10(t *testing.T) {
	svc, _, _, _, acRepo := newTestService()
	var capturedLimit int
	acRepo.autocompleteMarketplaceFn = func(_ context.Context, _ string, limit int) ([]AutocompleteSuggestion, error) {
		capturedLimit = limit
		return nil, nil
	}
	scope := SearchScopeMarketplace
	_, err := svc.Autocomplete(context.Background(), testAuth(), testFamilyScope(), &AutocompleteParams{
		Q: "t", Scope: &scope, Limit: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != 10 {
		t.Errorf("expected capped autocomplete limit 10, got %d", capturedLimit)
	}
}

func TestAutocomplete_DefaultScopeIsMarketplace(t *testing.T) {
	svc, _, _, _, acRepo := newTestService()
	called := false
	acRepo.autocompleteMarketplaceFn = func(_ context.Context, _ string, _ int) ([]AutocompleteSuggestion, error) {
		called = true
		return nil, nil
	}
	_, err := svc.Autocomplete(context.Background(), testAuth(), testFamilyScope(), &AutocompleteParams{
		Q: "t", Scope: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected marketplace autocomplete to be called when scope is nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 4: Social Scope Dispatch (#13–#16)
// ═══════════════════════════════════════════════════════════════════════════════

func TestSearch_Social_SubScopeFamilies(t *testing.T) {
	svc, socRepo, _, _, _ := newTestService()
	called := false
	subScope := SocialSubScopeFamilies
	socRepo.searchFamiliesFn = func(_ context.Context, searcherFamilyID uuid.UUID, _ string, _ int, _ *string) ([]SocialSearchResult, error) {
		called = true
		if searcherFamilyID != testAuth().FamilyID {
			t.Errorf("expected searcher family ID %s, got %s", testAuth().FamilyID, searcherFamilyID)
		}
		return nil, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeSocial, SubScope: &subScope,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected SearchFamilies to be called")
	}
}

func TestSearch_Social_SubScopeGroups(t *testing.T) {
	svc, socRepo, _, _, _ := newTestService()
	called := false
	subScope := SocialSubScopeGroups
	socRepo.searchGroupsFn = func(_ context.Context, _ uuid.UUID, _ string, _ *string, _ int, _ *string) ([]SocialSearchResult, error) {
		called = true
		return nil, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeSocial, SubScope: &subScope,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected SearchGroups to be called")
	}
}

func TestSearch_Social_SubScopeEvents(t *testing.T) {
	svc, socRepo, _, _, _ := newTestService()
	called := false
	subScope := SocialSubScopeEvents
	socRepo.searchEventsFn = func(_ context.Context, _ uuid.UUID, _ string, _ *string, _ int, _ *string) ([]SocialSearchResult, error) {
		called = true
		return nil, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeSocial, SubScope: &subScope,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected SearchEvents to be called")
	}
}

func TestSearch_Social_NoSubScope_MergesAll(t *testing.T) {
	svc, socRepo, _, _, _ := newTestService()
	familiesCalled, groupsCalled, eventsCalled, postsCalled := false, false, false, false
	socRepo.searchFamiliesFn = func(_ context.Context, _ uuid.UUID, _ string, _ int, _ *string) ([]SocialSearchResult, error) {
		familiesCalled = true
		return []SocialSearchResult{{Relevance: 0.9, Result: SearchResult{Type: "family", Family: &FamilySearchResult{Relevance: 0.9}}}}, nil
	}
	socRepo.searchGroupsFn = func(_ context.Context, _ uuid.UUID, _ string, _ *string, _ int, _ *string) ([]SocialSearchResult, error) {
		groupsCalled = true
		return []SocialSearchResult{{Relevance: 0.8, Result: SearchResult{Type: "group", Group: &GroupSearchResult{Relevance: 0.8}}}}, nil
	}
	socRepo.searchEventsFn = func(_ context.Context, _ uuid.UUID, _ string, _ *string, _ int, _ *string) ([]SocialSearchResult, error) {
		eventsCalled = true
		return []SocialSearchResult{{Relevance: 0.7, Result: SearchResult{Type: "event", Event: &EventSearchResult{Relevance: 0.7}}}}, nil
	}
	socRepo.searchPostsFn = func(_ context.Context, _ uuid.UUID, _ string, _ int, _ *string) ([]SocialSearchResult, error) {
		postsCalled = true
		return []SocialSearchResult{{Relevance: 0.6, Result: SearchResult{Type: "post", Post: &PostSearchResult{Relevance: 0.6}}}}, nil
	}
	resp, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeSocial,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !familiesCalled || !groupsCalled || !eventsCalled || !postsCalled {
		t.Error("expected all four social repos to be called")
	}
	if len(resp.Results) != 4 {
		t.Errorf("expected 4 merged results, got %d", len(resp.Results))
	}
	// Check sorted by relevance descending
	if resp.Results[0].Type != "family" {
		t.Errorf("expected first result to be 'family' (highest relevance), got %q", resp.Results[0].Type)
	}
	if resp.Results[3].Type != "post" {
		t.Errorf("expected last result to be 'post' (lowest relevance), got %q", resp.Results[3].Type)
	}
}

// #22 — social search passes auth.FamilyID to repo methods
func TestSearch_Social_PassesFamilyID(t *testing.T) {
	svc, socRepo, _, _, _ := newTestService()
	subScope := SocialSubScopeFamilies
	var capturedFamilyID uuid.UUID
	socRepo.searchFamiliesFn = func(_ context.Context, searcherFamilyID uuid.UUID, _ string, _ int, _ *string) ([]SocialSearchResult, error) {
		capturedFamilyID = searcherFamilyID
		return nil, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeSocial, SubScope: &subScope,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFamilyID != testAuth().FamilyID {
		t.Errorf("expected FamilyID %s, got %s", testAuth().FamilyID, capturedFamilyID)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 5: Marketplace + Learning Dispatch (#17–#22)
// ═══════════════════════════════════════════════════════════════════════════════

func TestSearch_Marketplace_CallsSearchListings(t *testing.T) {
	svc, _, mktRepo, _, _ := newTestService()
	called := false
	mktRepo.searchListingsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters, _ SearchSortOrder, _ int, _ *string) (*MarketplaceSearchResults, error) {
		called = true
		return &MarketplaceSearchResults{}, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeMarketplace,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected SearchListings to be called")
	}
}

func TestSearch_Marketplace_CallsCountFacets(t *testing.T) {
	svc, _, mktRepo, _, _ := newTestService()
	facetsCalled := false
	mktRepo.searchListingsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters, _ SearchSortOrder, _ int, _ *string) (*MarketplaceSearchResults, error) {
		return &MarketplaceSearchResults{}, nil
	}
	mktRepo.countFacetsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters) (*FacetCounts, error) {
		facetsCalled = true
		return &FacetCounts{}, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeMarketplace,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !facetsCalled {
		t.Error("expected CountFacets to be called for marketplace")
	}
}

func TestSearch_Learning_CallsSearchLearning(t *testing.T) {
	svc, _, _, learnRepo, _ := newTestService()
	called := false
	learnRepo.searchLearningFn = func(_ context.Context, familyScope *shared.FamilyScope, _ string, _ *LearningSearchFilters, _ int, _ *string) ([]LearningSearchResult, error) {
		called = true
		if familyScope.FamilyID() != testAuth().FamilyID {
			t.Errorf("expected FamilyScope with ID %s, got %s", testAuth().FamilyID, familyScope.FamilyID())
		}
		return nil, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeLearning,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected SearchLearning to be called")
	}
}

func TestSearch_Marketplace_ExtractsFilters(t *testing.T) {
	svc, _, mktRepo, _, _ := newTestService()
	var capturedFilters *MarketplaceSearchFilters
	priceMax := int32(5000)
	freeOnly := true
	mktRepo.searchListingsFn = func(_ context.Context, _ string, filters *MarketplaceSearchFilters, _ SearchSortOrder, _ int, _ *string) (*MarketplaceSearchResults, error) {
		capturedFilters = filters
		return &MarketplaceSearchResults{}, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeMarketplace,
		PriceMax:    &priceMax,
		FreeOnly:    &freeOnly,
		SubjectTags: []string{"math", "science"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilters == nil {
		t.Fatal("expected filters to be captured")
	}
	if capturedFilters.PriceMax == nil || *capturedFilters.PriceMax != 5000 {
		t.Error("expected PriceMax 5000")
	}
	if capturedFilters.FreeOnly == nil || !*capturedFilters.FreeOnly {
		t.Error("expected FreeOnly true")
	}
	if len(capturedFilters.SubjectTags) != 2 {
		t.Errorf("expected 2 subject tags, got %d", len(capturedFilters.SubjectTags))
	}
}

func TestSearch_Learning_ExtractsFilters(t *testing.T) {
	svc, _, _, learnRepo, _ := newTestService()
	var capturedFilters *LearningSearchFilters
	studentID := uuid.New()
	sourceType := LearningSourceTypeActivity
	learnRepo.searchLearningFn = func(_ context.Context, _ *shared.FamilyScope, _ string, filters *LearningSearchFilters, _ int, _ *string) ([]LearningSearchResult, error) {
		capturedFilters = filters
		return nil, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeLearning,
		StudentID:   &studentID,
		SourceType:  &sourceType,
		SubjectTags: []string{"art"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFilters == nil {
		t.Fatal("expected filters to be captured")
	}
	if capturedFilters.StudentID == nil || *capturedFilters.StudentID != studentID {
		t.Error("expected StudentID to match")
	}
	if capturedFilters.SourceType == nil || *capturedFilters.SourceType != LearningSourceTypeActivity {
		t.Error("expected SourceType to be activity")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 6: Response Assembly (#23–#26)
// ═══════════════════════════════════════════════════════════════════════════════

func TestSearch_Marketplace_ResponseHasFacets(t *testing.T) {
	svc, _, mktRepo, _, _ := newTestService()
	mktRepo.searchListingsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters, _ SearchSortOrder, _ int, _ *string) (*MarketplaceSearchResults, error) {
		return &MarketplaceSearchResults{TotalCount: 5}, nil
	}
	mktRepo.countFacetsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters) (*FacetCounts, error) {
		return &FacetCounts{ContentType: []FacetBucket{{Value: "ebook", Count: 5}}}, nil
	}
	resp, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeMarketplace,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Facets == nil {
		t.Error("expected non-nil Facets for marketplace")
	}
}

func TestSearch_Social_ResponseHasNilFacets(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	subScope := SocialSubScopeFamilies
	resp, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeSocial, SubScope: &subScope,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Facets != nil {
		t.Error("expected nil Facets for social scope")
	}
}

func TestSearch_Learning_ResponseHasNilFacets(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	resp, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeLearning,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Facets != nil {
		t.Error("expected nil Facets for learning scope")
	}
}

func TestSearch_Marketplace_NextCursorPropagated(t *testing.T) {
	svc, _, mktRepo, _, _ := newTestService()
	cursor := "next-page-cursor"
	mktRepo.searchListingsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters, _ SearchSortOrder, _ int, _ *string) (*MarketplaceSearchResults, error) {
		return &MarketplaceSearchResults{NextCursor: &cursor}, nil
	}
	resp, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeMarketplace,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NextCursor == nil || *resp.NextCursor != cursor {
		t.Error("expected NextCursor to be propagated")
	}
}

func TestSearch_Marketplace_TotalCountSet(t *testing.T) {
	svc, _, mktRepo, _, _ := newTestService()
	mktRepo.searchListingsFn = func(_ context.Context, _ string, _ *MarketplaceSearchFilters, _ SearchSortOrder, _ int, _ *string) (*MarketplaceSearchResults, error) {
		return &MarketplaceSearchResults{TotalCount: 42}, nil
	}
	resp, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test", Scope: SearchScopeMarketplace,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 42 {
		t.Errorf("expected TotalCount 42, got %d", resp.TotalCount)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 7: Autocomplete Dispatch (#27–#30)
// ═══════════════════════════════════════════════════════════════════════════════

func TestAutocomplete_Marketplace(t *testing.T) {
	svc, _, _, _, acRepo := newTestService()
	called := false
	acRepo.autocompleteMarketplaceFn = func(_ context.Context, _ string, _ int) ([]AutocompleteSuggestion, error) {
		called = true
		return []AutocompleteSuggestion{{Text: "math workbook", EntityType: "listing"}}, nil
	}
	scope := SearchScopeMarketplace
	resp, err := svc.Autocomplete(context.Background(), testAuth(), testFamilyScope(), &AutocompleteParams{
		Q: "math", Scope: &scope,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected AutocompleteMarketplace to be called")
	}
	if len(resp.Suggestions) != 1 {
		t.Errorf("expected 1 suggestion, got %d", len(resp.Suggestions))
	}
}

func TestAutocomplete_Social(t *testing.T) {
	svc, _, _, _, acRepo := newTestService()
	called := false
	acRepo.autocompleteSocialFn = func(_ context.Context, searcherFamilyID uuid.UUID, _ string, _ int) ([]AutocompleteSuggestion, error) {
		called = true
		if searcherFamilyID != testAuth().FamilyID {
			t.Errorf("expected searcher family ID %s, got %s", testAuth().FamilyID, searcherFamilyID)
		}
		return nil, nil
	}
	scope := SearchScopeSocial
	_, err := svc.Autocomplete(context.Background(), testAuth(), testFamilyScope(), &AutocompleteParams{
		Q: "s", Scope: &scope,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected AutocompleteSocial to be called")
	}
}

func TestAutocomplete_Learning(t *testing.T) {
	svc, _, _, _, acRepo := newTestService()
	called := false
	acRepo.autocompleteLearningFn = func(_ context.Context, familyScope *shared.FamilyScope, _ string, _ int) ([]AutocompleteSuggestion, error) {
		called = true
		if familyScope.FamilyID() != testAuth().FamilyID {
			t.Errorf("expected FamilyScope ID %s, got %s", testAuth().FamilyID, familyScope.FamilyID())
		}
		return nil, nil
	}
	scope := SearchScopeLearning
	_, err := svc.Autocomplete(context.Background(), testAuth(), testFamilyScope(), &AutocompleteParams{
		Q: "r", Scope: &scope,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected AutocompleteLearning to be called")
	}
}

func TestAutocomplete_WrapsSuggestions(t *testing.T) {
	svc, _, _, _, acRepo := newTestService()
	acRepo.autocompleteMarketplaceFn = func(_ context.Context, _ string, _ int) ([]AutocompleteSuggestion, error) {
		return []AutocompleteSuggestion{
			{Text: "suggestion1"},
			{Text: "suggestion2"},
		}, nil
	}
	scope := SearchScopeMarketplace
	resp, err := svc.Autocomplete(context.Background(), testAuth(), testFamilyScope(), &AutocompleteParams{
		Q: "s", Scope: &scope,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(resp.Suggestions))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 8: Event Handlers — Phase 1 No-ops (#31–#35)
// ═══════════════════════════════════════════════════════════════════════════════

func TestHandlePostCreated_NoOp(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	err := svc.HandlePostCreated(context.Background(), &PostCreated{PostID: uuid.New()})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestHandleListingPublished_NoOp(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	err := svc.HandleListingPublished(context.Background(), &ListingPublished{ListingID: uuid.New()})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestHandleListingArchived_NoOp(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	err := svc.HandleListingArchived(context.Background(), &ListingArchived{ListingID: uuid.New()})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestHandleUploadPublished_NoOp(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	err := svc.HandleUploadPublished(context.Background(), &UploadPublished{UploadID: uuid.New()})
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestHandleFamilyDeletionScheduled_NoOp(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	err := svc.HandleFamilyDeletionScheduled(context.Background(), uuid.New())
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cycle 10: Dual-Backend Routing (#43–#46)
// ═══════════════════════════════════════════════════════════════════════════════

func TestSearch_Typesense_MarketplaceUnavailable(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	svc.backend = SearchBackendTypesense
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test query", Scope: SearchScopeMarketplace,
	})
	assertSearchError(t, err, ErrBackendUnavailable)
}

func TestSearch_Typesense_SocialUnavailable(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	svc.backend = SearchBackendTypesense
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test query", Scope: SearchScopeSocial,
	})
	assertSearchError(t, err, ErrBackendUnavailable)
}

func TestSearch_Typesense_LearningAlwaysPostgres(t *testing.T) {
	svc, _, _, learnRepo, _ := newTestService()
	svc.backend = SearchBackendTypesense
	called := false
	learnRepo.searchLearningFn = func(_ context.Context, _ *shared.FamilyScope, _ string, _ *LearningSearchFilters, _ int, _ *string) ([]LearningSearchResult, error) {
		called = true
		return nil, nil
	}
	_, err := svc.Search(context.Background(), testAuth(), testFamilyScope(), &SearchParams{
		Q: "test query", Scope: SearchScopeLearning,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected learning search to use PostgreSQL even when backend is Typesense")
	}
}

func TestSearch_PostgresFts_DefaultBackend(t *testing.T) {
	svc, _, _, _, _ := newTestService()
	if svc.backend != SearchBackendPostgresFts {
		t.Errorf("expected default backend PostgresFts, got %d", svc.backend)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func assertSearchError(t *testing.T, err error, sentinel error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	var se *SearchError
	if !errors.As(err, &se) {
		t.Fatalf("expected *SearchError, got %T: %v", err, err)
	}
	if !errors.Is(se.Err, sentinel) {
		t.Errorf("expected sentinel %v, got %v", sentinel, se.Err)
	}
}
