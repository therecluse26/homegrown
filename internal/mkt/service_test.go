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
		Return(nil, ErrListingNotFound)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     listingRepo,
		listingFiles: &mockListingFileRepo{},
	})

	_, err := svc.GetListing(context.Background(), uuid.New())

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrListingNotFound))
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

// ─── RegisterCreator Success ──────────────────────────────────────────────────

func TestService_RegisterCreator_Success(t *testing.T) {
	parentID := uuid.New()
	creatorID := uuid.New()
	creatorRepo := &mockCreatorRepo{}
	creatorRepo.On("GetByParentID", mock.Anything, parentID).Return(nil, nil)
	creatorRepo.On("Create", mock.Anything, mock.AnythingOfType("mkt.CreateCreator")).
		Return(&MktCreator{ID: creatorID, StoreName: "My Store"}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     creatorRepo,
		publishers:   &mockPublisherRepo{},
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	auth := &shared.AuthContext{ParentID: parentID, FamilyID: uuid.New()}
	id, err := svc.RegisterCreator(context.Background(), RegisterCreatorCommand{
		StoreName:   "My Store",
		TOSAccepted: true,
	}, auth)

	require.NoError(t, err)
	assert.Equal(t, creatorID, id)
	creatorRepo.AssertExpectations(t)
}

func TestService_RegisterCreator_RepoError_Propagates(t *testing.T) {
	parentID := uuid.New()
	creatorRepo := &mockCreatorRepo{}
	repoErr := errors.New("db error")
	creatorRepo.On("GetByParentID", mock.Anything, parentID).Return(nil, repoErr)

	svc := newMktService(mktServiceDeps{
		creators:     creatorRepo,
		publishers:   &mockPublisherRepo{},
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	auth := &shared.AuthContext{ParentID: parentID, FamilyID: uuid.New()}
	_, err := svc.RegisterCreator(context.Background(), RegisterCreatorCommand{
		StoreName:   "My Store",
		TOSAccepted: true,
	}, auth)

	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}

// ─── UpdateCreatorProfile Tests ───────────────────────────────────────────────

func TestService_UpdateCreatorProfile_Success(t *testing.T) {
	creatorID := uuid.New()
	creatorRepo := &mockCreatorRepo{}
	creatorRepo.On("Update", mock.Anything, creatorID, mock.AnythingOfType("mkt.UpdateCreator")).
		Return(&MktCreator{ID: creatorID}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     creatorRepo,
		publishers:   &mockPublisherRepo{},
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	err := svc.UpdateCreatorProfile(context.Background(), UpdateCreatorProfileCommand{}, creatorID)
	require.NoError(t, err)
	creatorRepo.AssertExpectations(t)
}

// ─── CreatePublisher Tests ────────────────────────────────────────────────────

func TestService_CreatePublisher_Success(t *testing.T) {
	creatorID := uuid.New()
	publisherID := uuid.New()
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetBySlug", mock.Anything, mock.AnythingOfType("string")).Return(nil, nil)
	pubRepo.On("Create", mock.Anything, mock.AnythingOfType("mkt.CreatePublisher")).
		Return(&MktPublisher{ID: publisherID, Name: "My Publisher"}, nil)
	pubRepo.On("AddMember", mock.Anything, publisherID, creatorID, "owner").Return(nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	id, err := svc.CreatePublisher(context.Background(), CreatePublisherCommand{
		Name: "My Publisher",
	}, creatorID)

	require.NoError(t, err)
	assert.Equal(t, publisherID, id)
	pubRepo.AssertExpectations(t)
}

func TestService_CreatePublisher_SlugConflict(t *testing.T) {
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetBySlug", mock.Anything, mock.AnythingOfType("string")).
		Return(&MktPublisher{ID: uuid.New()}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	_, err := svc.CreatePublisher(context.Background(), CreatePublisherCommand{
		Name: "My Publisher",
	}, uuid.New())

	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 409, appErr.StatusCode)
}

// ─── UpdatePublisher Tests ────────────────────────────────────────────────────

func TestService_UpdatePublisher_PlatformPublisher_Forbidden(t *testing.T) {
	publisherID := uuid.New()
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetByID", mock.Anything, publisherID).
		Return(&MktPublisher{ID: publisherID, IsPlatform: true}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	err := svc.UpdatePublisher(context.Background(), UpdatePublisherCommand{}, publisherID, uuid.New())
	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 403, appErr.StatusCode)
}

func TestService_UpdatePublisher_NonOwner_Forbidden(t *testing.T) {
	publisherID := uuid.New()
	creatorID := uuid.New()
	memberRole := "member"
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetByID", mock.Anything, publisherID).
		Return(&MktPublisher{ID: publisherID, IsPlatform: false}, nil)
	pubRepo.On("GetMemberRole", mock.Anything, publisherID, creatorID).
		Return(&memberRole, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	err := svc.UpdatePublisher(context.Background(), UpdatePublisherCommand{}, publisherID, creatorID)
	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 403, appErr.StatusCode)
}

func TestService_UpdatePublisher_Owner_Success(t *testing.T) {
	publisherID := uuid.New()
	creatorID := uuid.New()
	ownerRole := "owner"
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetByID", mock.Anything, publisherID).
		Return(&MktPublisher{ID: publisherID, IsPlatform: false}, nil)
	pubRepo.On("GetMemberRole", mock.Anything, publisherID, creatorID).
		Return(&ownerRole, nil)
	pubRepo.On("Update", mock.Anything, publisherID, mock.AnythingOfType("mkt.UpdatePublisher")).
		Return(&MktPublisher{ID: publisherID}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	err := svc.UpdatePublisher(context.Background(), UpdatePublisherCommand{}, publisherID, creatorID)
	require.NoError(t, err)
	pubRepo.AssertExpectations(t)
}

// ─── AddPublisherMember Tests ─────────────────────────────────────────────────

func TestService_AddPublisherMember_PlatformPublisher_Forbidden(t *testing.T) {
	publisherID := uuid.New()
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetByID", mock.Anything, publisherID).
		Return(&MktPublisher{ID: publisherID, IsPlatform: true}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	err := svc.AddPublisherMember(context.Background(), publisherID, AddPublisherMemberCommand{
		CreatorID: uuid.New(),
		Role:      "member",
	}, uuid.New())
	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 403, appErr.StatusCode)
}

func TestService_AddPublisherMember_NonOwner_Forbidden(t *testing.T) {
	publisherID := uuid.New()
	actingCreatorID := uuid.New()
	memberRole := "member"
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetByID", mock.Anything, publisherID).
		Return(&MktPublisher{ID: publisherID, IsPlatform: false}, nil)
	pubRepo.On("GetMemberRole", mock.Anything, publisherID, actingCreatorID).
		Return(&memberRole, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	err := svc.AddPublisherMember(context.Background(), publisherID, AddPublisherMemberCommand{
		CreatorID: uuid.New(),
		Role:      "member",
	}, actingCreatorID)
	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 403, appErr.StatusCode)
}

// ─── RemovePublisherMember Tests ──────────────────────────────────────────────

func TestService_RemovePublisherMember_PlatformPublisher_Forbidden(t *testing.T) {
	publisherID := uuid.New()
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetByID", mock.Anything, publisherID).
		Return(&MktPublisher{ID: publisherID, IsPlatform: true}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	err := svc.RemovePublisherMember(context.Background(), publisherID, uuid.New(), uuid.New())
	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 403, appErr.StatusCode)
}

func TestService_RemovePublisherMember_LastOwner_Rejected(t *testing.T) {
	publisherID := uuid.New()
	actingCreatorID := uuid.New()
	memberCreatorID := uuid.New()
	ownerRole := "owner"
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetByID", mock.Anything, publisherID).
		Return(&MktPublisher{ID: publisherID, IsPlatform: false}, nil)
	pubRepo.On("GetMemberRole", mock.Anything, publisherID, actingCreatorID).
		Return(&ownerRole, nil)
	pubRepo.On("GetMemberRole", mock.Anything, publisherID, memberCreatorID).
		Return(&ownerRole, nil)
	pubRepo.On("CountOwners", mock.Anything, publisherID).
		Return(int32(1), nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	err := svc.RemovePublisherMember(context.Background(), publisherID, memberCreatorID, actingCreatorID)
	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.StatusCode)
}

// ─── CreateListing Tests ──────────────────────────────────────────────────────

func TestService_CreateListing_InvalidContentType(t *testing.T) {
	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	_, err := svc.CreateListing(context.Background(), CreateListingCommand{
		ContentType: "invalid_type",
		PublisherID: uuid.New(),
		Title:       "Test",
	}, uuid.New())

	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 422, appErr.StatusCode)
}

func TestService_CreateListing_NonMember_Forbidden(t *testing.T) {
	publisherID := uuid.New()
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetMemberRole", mock.Anything, publisherID, mock.AnythingOfType("uuid.UUID")).
		Return(nil, ErrNotPublisherMember)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	_, err := svc.CreateListing(context.Background(), CreateListingCommand{
		ContentType: "curriculum",
		PublisherID: publisherID,
		Title:       "Test",
	}, uuid.New())

	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 403, appErr.StatusCode)
}

func TestService_CreateListing_Success(t *testing.T) {
	publisherID := uuid.New()
	creatorID := uuid.New()
	listingID := uuid.New()
	ownerRole := "owner"
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetMemberRole", mock.Anything, publisherID, creatorID).
		Return(&ownerRole, nil)

	listingRepo := &mockListingRepo{}
	listingRepo.On("Create", mock.Anything, mock.AnythingOfType("mkt.CreateListing")).
		Return(&MktListing{ID: listingID}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     listingRepo,
		listingFiles: &mockListingFileRepo{},
	})

	id, err := svc.CreateListing(context.Background(), CreateListingCommand{
		ContentType: "curriculum",
		PublisherID: publisherID,
		Title:       "Test Listing",
		Description: "A test listing",
		PriceCents:  999,
	}, creatorID)

	require.NoError(t, err)
	assert.Equal(t, listingID, id)
	pubRepo.AssertExpectations(t)
	listingRepo.AssertExpectations(t)
}

// ─── UpdateListing Tests ──────────────────────────────────────────────────────

func TestService_UpdateListing_NotOwner_Forbidden(t *testing.T) {
	listingID := uuid.New()
	listingRepo := &mockListingRepo{}
	listingRepo.On("GetByID", mock.Anything, listingID).
		Return(&MktListing{ID: listingID, CreatorID: uuid.New()}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     listingRepo,
		listingFiles: &mockListingFileRepo{},
	})

	err := svc.UpdateListing(context.Background(), UpdateListingCommand{}, listingID, uuid.New())
	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 403, appErr.StatusCode)
}

// ─── GetCreatorByParentID Tests ───────────────────────────────────────────────

func TestService_GetCreatorByParentID_NotFound(t *testing.T) {
	parentID := uuid.New()
	creatorRepo := &mockCreatorRepo{}
	creatorRepo.On("GetByParentID", mock.Anything, parentID).Return(nil, ErrCreatorNotFound)

	svc := newMktService(mktServiceDeps{
		creators:     creatorRepo,
		publishers:   &mockPublisherRepo{},
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	result, err := svc.GetCreatorByParentID(context.Background(), parentID)
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestService_GetCreatorByParentID_Success(t *testing.T) {
	parentID := uuid.New()
	creatorID := uuid.New()
	creatorRepo := &mockCreatorRepo{}
	creatorRepo.On("GetByParentID", mock.Anything, parentID).
		Return(&MktCreator{ID: creatorID, StoreName: "My Store"}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     creatorRepo,
		publishers:   &mockPublisherRepo{},
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	result, err := svc.GetCreatorByParentID(context.Background(), parentID)
	require.NoError(t, err)
	assert.Equal(t, creatorID, result.ID)
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

func TestService_BrowseListings_EmptyResults(t *testing.T) {
	listingRepo := &mockListingRepo{}
	listingRepo.On("Browse", mock.Anything, mock.AnythingOfType("*mkt.BrowseListingsParams")).
		Return([]ListingBrowseRow{}, int64(0), nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     listingRepo,
		listingFiles: &mockListingFileRepo{},
	})

	result, err := svc.BrowseListings(context.Background(), BrowseListingsParams{})

	require.NoError(t, err)
	assert.Empty(t, result.Data)
}

func TestService_BrowseListings_RepoError(t *testing.T) {
	listingRepo := &mockListingRepo{}
	repoErr := errors.New("db error")
	listingRepo.On("Browse", mock.Anything, mock.AnythingOfType("*mkt.BrowseListingsParams")).
		Return([]ListingBrowseRow{}, int64(0), repoErr)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     listingRepo,
		listingFiles: &mockListingFileRepo{},
	})

	_, err := svc.BrowseListings(context.Background(), BrowseListingsParams{})
	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}

// ─── GetCreatorListings Tests ─────────────────────────────────────────────────

func TestService_GetCreatorListings_Success(t *testing.T) {
	creatorID := uuid.New()
	listingID := uuid.New()
	listingRepo := &mockListingRepo{}
	listingRepo.On("GetByCreator", mock.Anything, creatorID, mock.AnythingOfType("*mkt.CreatorListingQueryParams")).
		Return([]MktListing{{ID: listingID, Title: "Listing 1"}}, int64(1), nil)

	listingFileRepo := &mockListingFileRepo{}
	listingFileRepo.On("ListByListing", mock.Anything, listingID).
		Return([]MktListingFile{}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     listingRepo,
		listingFiles: listingFileRepo,
	})

	result, err := svc.GetCreatorListings(context.Background(), creatorID, CreatorListingQueryParams{})
	require.NoError(t, err)
	require.Len(t, result.Data, 1)
}

// ─── Autocomplete Tests ────────────────────────────────────────────────────────

func TestService_AutocompleteListings_Success(t *testing.T) {
	listingRepo := &mockListingRepo{}
	listingRepo.On("Autocomplete", mock.Anything, "test", uint8(10)).
		Return([]AutocompleteRow{{ListingID: uuid.New(), Title: "Test Listing"}}, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     listingRepo,
		listingFiles: &mockListingFileRepo{},
	})

	results, err := svc.AutocompleteListings(context.Background(), "test", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Test Listing", results[0].Title)
}

func TestService_AutocompleteListings_QueryTooShort(t *testing.T) {
	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   &mockPublisherRepo{},
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	_, err := svc.AutocompleteListings(context.Background(), "a", 10)
	require.Error(t, err)
	var appErr *shared.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 422, appErr.StatusCode)
}

// ─── GetPublisher Tests ────────────────────────────────────────────────────────

func TestService_GetPublisher_NotFound(t *testing.T) {
	publisherID := uuid.New()
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetByID", mock.Anything, publisherID).Return(nil, ErrPublisherNotFound)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	_, err := svc.GetPublisher(context.Background(), publisherID)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPublisherNotFound))
}

func TestService_GetPublisher_Success(t *testing.T) {
	publisherID := uuid.New()
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetByID", mock.Anything, publisherID).
		Return(&MktPublisher{ID: publisherID, Name: "My Pub"}, nil)
	pubRepo.On("CountMembers", mock.Anything, publisherID).
		Return(int32(3), nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	result, err := svc.GetPublisher(context.Background(), publisherID)
	require.NoError(t, err)
	assert.Equal(t, "My Pub", result.Name)
	pubRepo.AssertExpectations(t)
}

// ─── VerifyPublisherMembership Tests ──────────────────────────────────────────

func TestService_VerifyPublisherMembership_NotMember(t *testing.T) {
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetMemberRole", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).
		Return(nil, ErrNotPublisherMember)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	isMember, err := svc.VerifyPublisherMembership(context.Background(), uuid.New(), uuid.New())
	require.Error(t, err)
	assert.False(t, isMember)
}

func TestService_VerifyPublisherMembership_IsMember(t *testing.T) {
	ownerRole := "owner"
	pubRepo := &mockPublisherRepo{}
	pubRepo.On("GetMemberRole", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID")).
		Return(&ownerRole, nil)

	svc := newMktService(mktServiceDeps{
		creators:     &mockCreatorRepo{},
		publishers:   pubRepo,
		listings:     &mockListingRepo{},
		listingFiles: &mockListingFileRepo{},
	})

	isMember, err := svc.VerifyPublisherMembership(context.Background(), uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.True(t, isMember)
}
