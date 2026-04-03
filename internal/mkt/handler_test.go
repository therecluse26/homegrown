package mkt

// Handler tests for the marketplace domain. [07-mkt §16]
// Creator routes (RequireCreator middleware) require a DB query and are tested
// via integration tests. These unit tests cover public + authenticated buyer routes.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

type echoValidator struct{ v *validator.Validate }

func (cv *echoValidator) Validate(i any) error { return cv.v.Struct(i) }

// ─── Mock MarketplaceService ──────────────────────────────────────────────────

type mockMarketplaceService struct {
	browseListingsFn func(ctx context.Context, params BrowseListingsParams) (*shared.PaginatedResponse[ListingBrowseResponse], error)
	getListingFn     func(ctx context.Context, listingID uuid.UUID) (*ListingDetailResponse, error)
	getCartFn        func(ctx context.Context, scope shared.FamilyScope) (*CartResponse, error)
}

func (m *mockMarketplaceService) BrowseListings(ctx context.Context, params BrowseListingsParams) (*shared.PaginatedResponse[ListingBrowseResponse], error) {
	if m.browseListingsFn != nil {
		return m.browseListingsFn(ctx, params)
	}
	return &shared.PaginatedResponse[ListingBrowseResponse]{Data: []ListingBrowseResponse{}}, nil
}
func (m *mockMarketplaceService) GetListing(ctx context.Context, listingID uuid.UUID) (*ListingDetailResponse, error) {
	if m.getListingFn != nil {
		return m.getListingFn(ctx, listingID)
	}
	return &ListingDetailResponse{}, nil
}
func (m *mockMarketplaceService) GetCart(ctx context.Context, scope shared.FamilyScope) (*CartResponse, error) {
	if m.getCartFn != nil {
		return m.getCartFn(ctx, scope)
	}
	return &CartResponse{Items: []CartItemResponse{}}, nil
}
func (m *mockMarketplaceService) RegisterCreator(_ context.Context, _ RegisterCreatorCommand, _ *shared.AuthContext) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockMarketplaceService) UpdateCreatorProfile(_ context.Context, _ UpdateCreatorProfileCommand, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) CreateOnboardingLink(_ context.Context, _ uuid.UUID) (string, error) {
	return "https://example.com/onboard", nil
}
func (m *mockMarketplaceService) CreatePublisher(_ context.Context, _ CreatePublisherCommand, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockMarketplaceService) UpdatePublisher(_ context.Context, _ UpdatePublisherCommand, _, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) AddPublisherMember(_ context.Context, _ uuid.UUID, _ AddPublisherMemberCommand, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) RemovePublisherMember(_ context.Context, _, _, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) CreateListing(_ context.Context, _ CreateListingCommand, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockMarketplaceService) UpdateListing(_ context.Context, _ UpdateListingCommand, _, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) SubmitListing(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockMarketplaceService) PublishListing(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) ArchiveListing(_ context.Context, _, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) UploadListingFile(_ context.Context, _ UploadListingFileCommand, _, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockMarketplaceService) AddToCart(_ context.Context, _ uuid.UUID, _ shared.FamilyScope, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) RemoveFromCart(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) error {
	return nil
}
func (m *mockMarketplaceService) CreateCheckout(_ context.Context, _ shared.FamilyScope) (*CheckoutSessionResponse, error) {
	return &CheckoutSessionResponse{}, nil
}
func (m *mockMarketplaceService) HandlePaymentWebhook(_ context.Context, _ []byte, _ string) error {
	return nil
}
func (m *mockMarketplaceService) CreateReview(_ context.Context, _ CreateReviewCommand, _ uuid.UUID, _ shared.FamilyScope) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockMarketplaceService) UpdateReview(_ context.Context, _ UpdateReviewCommand, _ uuid.UUID, _ shared.FamilyScope) error {
	return nil
}
func (m *mockMarketplaceService) DeleteReview(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) error {
	return nil
}
func (m *mockMarketplaceService) RespondToReview(_ context.Context, _ RespondToReviewCommand, _, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) GetFreeListing(_ context.Context, _ uuid.UUID, _ shared.FamilyScope) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *mockMarketplaceService) RequestPayout(_ context.Context, _ uuid.UUID) (*PayoutResult, error) {
	return &PayoutResult{}, nil
}
func (m *mockMarketplaceService) HandleContentFlagged(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockMarketplaceService) ArchiveListingByContentKey(_ context.Context, _, _ string) error {
	return nil
}
func (m *mockMarketplaceService) HandleFamilyDeletionScheduled(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockMarketplaceService) GetCreatorByParentID(_ context.Context, _ uuid.UUID) (*CreatorResponse, error) {
	return &CreatorResponse{}, nil
}
func (m *mockMarketplaceService) GetCreatorDashboard(_ context.Context, _ uuid.UUID, _ DashboardPeriod) (*CreatorDashboardResponse, error) {
	return &CreatorDashboardResponse{}, nil
}
func (m *mockMarketplaceService) GetCreatorListings(_ context.Context, _ uuid.UUID, _ CreatorListingQueryParams) (*shared.PaginatedResponse[ListingDetailResponse], error) {
	return &shared.PaginatedResponse[ListingDetailResponse]{Data: []ListingDetailResponse{}}, nil
}
func (m *mockMarketplaceService) GetPublisher(_ context.Context, _ uuid.UUID) (*PublisherResponse, error) {
	return &PublisherResponse{}, nil
}
func (m *mockMarketplaceService) GetPublisherMembers(_ context.Context, _, _ uuid.UUID) ([]PublisherMemberResponse, error) {
	return []PublisherMemberResponse{}, nil
}
func (m *mockMarketplaceService) VerifyPublisherMembership(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return true, nil
}
func (m *mockMarketplaceService) AutocompleteListings(_ context.Context, _ string, _ uint8) ([]AutocompleteResult, error) {
	return []AutocompleteResult{}, nil
}
func (m *mockMarketplaceService) GetCuratedSections(_ context.Context, _ uint8) ([]CuratedSectionResponse, error) {
	return []CuratedSectionResponse{}, nil
}
func (m *mockMarketplaceService) GetPurchases(_ context.Context, _ shared.FamilyScope, _ PurchaseQueryParams) (*shared.PaginatedResponse[PurchaseResponse], error) {
	return &shared.PaginatedResponse[PurchaseResponse]{Data: []PurchaseResponse{}}, nil
}
func (m *mockMarketplaceService) GetDownloadURL(_ context.Context, _, _ uuid.UUID, _ shared.FamilyScope) (*DownloadResponse, error) {
	return &DownloadResponse{}, nil
}
func (m *mockMarketplaceService) GetListingFile(_ context.Context, _, _ uuid.UUID) (*ListingFileResponse, error) {
	return &ListingFileResponse{}, nil
}
func (m *mockMarketplaceService) GetReview(_ context.Context, _ uuid.UUID) (*ReviewResponse, error) {
	return &ReviewResponse{}, nil
}
func (m *mockMarketplaceService) GetListingReviews(_ context.Context, _ uuid.UUID, _ ReviewQueryParams) (*shared.PaginatedResponse[ReviewResponse], error) {
	return &shared.PaginatedResponse[ReviewResponse]{Data: []ReviewResponse{}}, nil
}

// Compile-time check.
var _ MarketplaceService = (*mockMarketplaceService)(nil)

// ─── Test Helpers ─────────────────────────────────────────────────────────────

func setupMktHandlerTest(svc MarketplaceService) (*echo.Echo, *Handler) {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	// nil cache: RequireCreator middleware is not called by tested routes.
	return e, NewHandler(svc, nil)
}

func setMktTestAuth(c echo.Context) {
	shared.SetAuthContext(c, &shared.AuthContext{
		ParentID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		FamilyID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
	})
}

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestHandler_BrowseListings_200(t *testing.T) {
	svc := &mockMarketplaceService{
		browseListingsFn: func(_ context.Context, _ BrowseListingsParams) (*shared.PaginatedResponse[ListingBrowseResponse], error) {
			return &shared.PaginatedResponse[ListingBrowseResponse]{Data: []ListingBrowseResponse{}}, nil
		},
	}
	e, h := setupMktHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/marketplace/listings", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// browse is public — no auth needed

	if err := h.browseListings(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_GetListing_200(t *testing.T) {
	listingID := uuid.New()
	svc := &mockMarketplaceService{
		getListingFn: func(_ context.Context, id uuid.UUID) (*ListingDetailResponse, error) {
			return &ListingDetailResponse{ID: id}, nil
		},
	}
	e, h := setupMktHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/marketplace/listings/"+listingID.String(), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("listing_id")
	c.SetParamValues(listingID.String())

	if err := h.getListing(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_GetListing_InvalidID_400(t *testing.T) {
	e, h := setupMktHandlerTest(&mockMarketplaceService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketplace/listings/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("listing_id")
	c.SetParamValues("not-a-uuid")

	err := h.getListing(c)
	if err == nil {
		t.Fatal("expected error for invalid listing ID")
	}
}

func TestHandler_GetCart_200(t *testing.T) {
	svc := &mockMarketplaceService{
		getCartFn: func(_ context.Context, _ shared.FamilyScope) (*CartResponse, error) {
			return &CartResponse{Items: []CartItemResponse{}}, nil
		},
	}
	e, h := setupMktHandlerTest(svc)
	req := httptest.NewRequest(http.MethodGet, "/v1/marketplace/cart", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	setMktTestAuth(c)

	if err := h.getCart(c); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d", rec.Code)
	}
}

func TestHandler_GetCart_MissingAuth_Errors(t *testing.T) {
	e, h := setupMktHandlerTest(&mockMarketplaceService{})
	req := httptest.NewRequest(http.MethodGet, "/v1/marketplace/cart", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// no auth

	if err := h.getCart(c); err == nil {
		t.Fatal("expected error for missing auth")
	}
}
