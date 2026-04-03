package mkt

// Service unit tests for the marketplace domain. [07-mkt §4]
// Tests cover key business logic paths using mock repositories.

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ─── Mock Repositories ────────────────────────────────────────────────────────

type mockCreatorRepo struct{ mock.Mock }

func (m *mockCreatorRepo) Create(ctx context.Context, cmd CreateCreator) (*MktCreator, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktCreator), args.Error(1)
}
func (m *mockCreatorRepo) GetByID(ctx context.Context, id uuid.UUID) (*MktCreator, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktCreator), args.Error(1)
}
func (m *mockCreatorRepo) GetByParentID(ctx context.Context, parentID uuid.UUID) (*MktCreator, error) {
	args := m.Called(ctx, parentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktCreator), args.Error(1)
}
func (m *mockCreatorRepo) Update(ctx context.Context, id uuid.UUID, cmd UpdateCreator) (*MktCreator, error) {
	args := m.Called(ctx, id, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktCreator), args.Error(1)
}
func (m *mockCreatorRepo) SetOnboardingStatus(ctx context.Context, id uuid.UUID, status string) error {
	return m.Called(ctx, id, status).Error(0)
}
func (m *mockCreatorRepo) SetPaymentAccountID(ctx context.Context, id uuid.UUID, paymentAccountID string) error {
	return m.Called(ctx, id, paymentAccountID).Error(0)
}

type mockListingRepo struct{ mock.Mock }

func (m *mockListingRepo) Create(ctx context.Context, cmd CreateListing) (*MktListing, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktListing), args.Error(1)
}
func (m *mockListingRepo) Save(ctx context.Context, l *domain.MarketplaceListing) error {
	return m.Called(ctx, l).Error(0)
}
func (m *mockListingRepo) CreateVersionSnapshot(ctx context.Context, listingID uuid.UUID, version int32, title, description string, priceCents int32, changeSummary *string) error {
	return m.Called(ctx, listingID, version, title, description, priceCents, changeSummary).Error(0)
}
func (m *mockListingRepo) GetByID(ctx context.Context, id uuid.UUID) (*MktListing, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktListing), args.Error(1)
}
func (m *mockListingRepo) Browse(ctx context.Context, params *BrowseListingsParams) ([]ListingBrowseRow, int64, error) {
	args := m.Called(ctx, params)
	return args.Get(0).([]ListingBrowseRow), args.Get(1).(int64), args.Error(2)
}
func (m *mockListingRepo) Autocomplete(ctx context.Context, query string, limit uint8) ([]AutocompleteRow, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]AutocompleteRow), args.Error(1)
}
func (m *mockListingRepo) GetByCreator(ctx context.Context, creatorID uuid.UUID, params *CreatorListingQueryParams) ([]MktListing, int64, error) {
	args := m.Called(ctx, creatorID, params)
	return args.Get(0).([]MktListing), args.Get(1).(int64), args.Error(2)
}
func (m *mockListingRepo) GetVersions(ctx context.Context, id uuid.UUID) ([]MktListingVersion, error) {
	args := m.Called(ctx, id)
	return args.Get(0).([]MktListingVersion), args.Error(1)
}
func (m *mockListingRepo) CountFiles(ctx context.Context, id uuid.UUID) (int64, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int64), args.Error(1)
}

type mockListingFileRepo struct{ mock.Mock }

func (m *mockListingFileRepo) Create(ctx context.Context, cmd CreateListingFile) (*MktListingFile, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktListingFile), args.Error(1)
}
func (m *mockListingFileRepo) GetByID(ctx context.Context, listingID, fileID uuid.UUID) (*MktListingFile, error) {
	args := m.Called(ctx, listingID, fileID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktListingFile), args.Error(1)
}
func (m *mockListingFileRepo) ListByListing(ctx context.Context, listingID uuid.UUID) ([]MktListingFile, error) {
	args := m.Called(ctx, listingID)
	return args.Get(0).([]MktListingFile), args.Error(1)
}
func (m *mockListingFileRepo) FindByStorageKey(ctx context.Context, storageKey string) (*MktListingFile, error) {
	args := m.Called(ctx, storageKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktListingFile), args.Error(1)
}
func (m *mockListingFileRepo) Delete(ctx context.Context, fileID uuid.UUID) error {
	return m.Called(ctx, fileID).Error(0)
}

type mockPublisherRepo struct{ mock.Mock }

func (m *mockPublisherRepo) Create(ctx context.Context, cmd CreatePublisher) (*MktPublisher, error) {
	args := m.Called(ctx, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktPublisher), args.Error(1)
}
func (m *mockPublisherRepo) GetByID(ctx context.Context, id uuid.UUID) (*MktPublisher, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktPublisher), args.Error(1)
}
func (m *mockPublisherRepo) GetBySlug(ctx context.Context, slug string) (*MktPublisher, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktPublisher), args.Error(1)
}
func (m *mockPublisherRepo) Update(ctx context.Context, id uuid.UUID, cmd UpdatePublisher) (*MktPublisher, error) {
	args := m.Called(ctx, id, cmd)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktPublisher), args.Error(1)
}
func (m *mockPublisherRepo) GetPlatformPublisher(ctx context.Context) (*MktPublisher, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*MktPublisher), args.Error(1)
}
func (m *mockPublisherRepo) CountMembers(ctx context.Context, id uuid.UUID) (int32, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int32), args.Error(1)
}
func (m *mockPublisherRepo) AddMember(ctx context.Context, publisherID, creatorID uuid.UUID, role string) error {
	return m.Called(ctx, publisherID, creatorID, role).Error(0)
}
func (m *mockPublisherRepo) RemoveMember(ctx context.Context, publisherID, creatorID uuid.UUID) error {
	return m.Called(ctx, publisherID, creatorID).Error(0)
}
func (m *mockPublisherRepo) GetMembers(ctx context.Context, publisherID uuid.UUID) ([]PublisherMemberRow, error) {
	args := m.Called(ctx, publisherID)
	return args.Get(0).([]PublisherMemberRow), args.Error(1)
}
func (m *mockPublisherRepo) GetMemberRole(ctx context.Context, publisherID, creatorID uuid.UUID) (*string, error) {
	args := m.Called(ctx, publisherID, creatorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*string), args.Error(1)
}
func (m *mockPublisherRepo) GetPublishersForCreator(ctx context.Context, creatorID uuid.UUID) ([]MktPublisher, error) {
	args := m.Called(ctx, creatorID)
	return args.Get(0).([]MktPublisher), args.Error(1)
}
func (m *mockPublisherRepo) CountOwners(ctx context.Context, id uuid.UUID) (int32, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(int32), args.Error(1)
}

// ─── Service Constructor Helper ───────────────────────────────────────────────

type mktServiceDeps struct {
	creators     *mockCreatorRepo
	publishers   *mockPublisherRepo
	listings     *mockListingRepo
	listingFiles *mockListingFileRepo
}

func newMktService(deps mktServiceDeps) MarketplaceService {
	bus := shared.NewEventBus()
	return NewMarketplaceService(
		deps.creators,
		deps.publishers,
		deps.listings,
		deps.listingFiles,
		nil, // cart
		nil, // purchases
		nil, // reviews
		nil, // curatedSections
		nil, // payment
		nil, // media
		bus,
		nil, // db
	)
}

// ─── RegisterCreator Tests ────────────────────────────────────────────────────

func TestService_RegisterCreator_TOSNotAccepted_ValidationError(t *testing.T) {
	// Pure validation — no repos needed.
	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	auth := &shared.AuthContext{ParentID: uuid.New(), FamilyID: uuid.New()}
	_, err := svc.RegisterCreator(context.Background(), RegisterCreatorCommand{
		StoreName:   "My Store",
		TOSAccepted: false,
	}, auth)

	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 422, appErr.StatusCode)
}

func TestService_RegisterCreator_AlreadyExists_ConflictError(t *testing.T) {
	parentID := uuid.New()
	creatorRepo := &mockCreatorRepo{}
	creatorRepo.On("GetByParentID", mock.Anything, parentID).
		Return(&MktCreator{ID: uuid.New(), StoreName: "Existing Store"}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     creatorRepo,
		publishers:   &mockPublisherRepo{},
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	auth := &shared.AuthContext{ParentID: parentID, FamilyID: uuid.New()}
	_, err := svc.RegisterCreator(context.Background(), RegisterCreatorCommand{
		StoreName:   "New Store",
		TOSAccepted: true,
	}, auth)

	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 409, appErr.StatusCode)
	creatorRepo.AssertExpectations(t)
}

// ─── GetListing Tests ─────────────────────────────────────────────────────────

func TestService_GetListing_NotFound_ReturnsNotFound(t *testing.T) {
	listingRepo := &mockListingRepo{}
	listingRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return(nil, nil) // nil, nil = not found

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     listingRepo,
		listingFiles: &mockListingFileRepo{},
	})

	_, err := svc.GetListing(context.Background(), uuid.New())

	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.StatusCode)
	listingRepo.AssertExpectations(t)
}

func TestService_GetListing_RepositoryError_Propagates(t *testing.T) {
	listingRepo := &mockListingRepo{}
	repoErr := errors.New("db connection failed")
	listingRepo.On("GetByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).
		Return(nil, repoErr)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     listingRepo,
		listingFiles: &mockListingFileRepo{},
	})

	_, err := svc.GetListing(context.Background(), uuid.New())

	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
	listingRepo.AssertExpectations(t)
}

// ─── BrowseListings Tests ─────────────────────────────────────────────────────

func TestService_BrowseListings_MapsResults(t *testing.T) {
	listingID := uuid.New()
	listingRepo := &mockListingRepo{}
	listingRepo.On("Browse", mock.Anything, mock.AnythingOfType("*mkt.BrowseListingsParams")).
		Return([]ListingBrowseRow{
			{
				ID:               listingID,
				Title:            "Test Listing",
				Description:      "A very long description that should be truncated",
				PriceCents:       999,
				PublisherName:    "Test Publisher",
				CreatorStoreName: "Test Store",
			},
		}, int64(1), nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     listingRepo,
		listingFiles: &mockListingFileRepo{},
	})

	result, err := svc.BrowseListings(context.Background(), BrowseListingsParams{})

	require.NoError(t, err)
	require.Len(t, result.Data, 1)
	assert.Equal(t, listingID, result.Data[0].ID)
	assert.Equal(t, "Test Listing", result.Data[0].Title)
	assert.Equal(t, int32(999), result.Data[0].PriceCents)
	listingRepo.AssertExpectations(t)
}
