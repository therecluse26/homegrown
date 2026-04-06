package mkt

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [CODING §8.2]
// ═══════════════════════════════════════════════════════════════════════════════

// MarketplaceService defines all marketplace use cases.
// Command methods return only IDs or error — never rich reads after write. [CODING §8.5]
type MarketplaceService interface {
	// ─── Command side (write, has side effects) ─────────────────────────

	// Creator onboarding
	RegisterCreator(ctx context.Context, cmd RegisterCreatorCommand, auth *shared.AuthContext) (uuid.UUID, error)
	UpdateCreatorProfile(ctx context.Context, cmd UpdateCreatorProfileCommand, creatorID uuid.UUID) error
	CreateOnboardingLink(ctx context.Context, creatorID uuid.UUID) (string, error)

	// Publisher management
	CreatePublisher(ctx context.Context, cmd CreatePublisherCommand, creatorID uuid.UUID) (uuid.UUID, error)
	UpdatePublisher(ctx context.Context, cmd UpdatePublisherCommand, publisherID, creatorID uuid.UUID) error
	AddPublisherMember(ctx context.Context, publisherID uuid.UUID, cmd AddPublisherMemberCommand, actingCreatorID uuid.UUID) error
	RemovePublisherMember(ctx context.Context, publisherID, memberCreatorID, actingCreatorID uuid.UUID) error

	// Listing lifecycle
	CreateListing(ctx context.Context, cmd CreateListingCommand, creatorID uuid.UUID) (uuid.UUID, error)
	UpdateListing(ctx context.Context, cmd UpdateListingCommand, listingID, creatorID uuid.UUID) error
	SubmitListing(ctx context.Context, listingID, creatorID uuid.UUID) error
	PublishListing(ctx context.Context, listingID, creatorID uuid.UUID) error
	ArchiveListing(ctx context.Context, listingID, creatorID uuid.UUID) error
	UploadListingFile(ctx context.Context, cmd UploadListingFileCommand, listingID, creatorID uuid.UUID) (uuid.UUID, error)

	// Cart & checkout
	AddToCart(ctx context.Context, listingID uuid.UUID, scope shared.FamilyScope, parentID uuid.UUID) error
	RemoveFromCart(ctx context.Context, listingID uuid.UUID, scope shared.FamilyScope) error
	CreateCheckout(ctx context.Context, scope shared.FamilyScope) (*CheckoutSessionResponse, error)
	HandlePaymentWebhook(ctx context.Context, payload []byte, signature string) error

	// Reviews
	CreateReview(ctx context.Context, cmd CreateReviewCommand, listingID uuid.UUID, scope shared.FamilyScope) (uuid.UUID, error)
	UpdateReview(ctx context.Context, cmd UpdateReviewCommand, reviewID uuid.UUID, scope shared.FamilyScope) error
	DeleteReview(ctx context.Context, reviewID uuid.UUID, scope shared.FamilyScope) error
	RespondToReview(ctx context.Context, cmd RespondToReviewCommand, reviewID, creatorID uuid.UUID) error

	// Free content acquisition
	GetFreeListing(ctx context.Context, listingID uuid.UUID, scope shared.FamilyScope) (uuid.UUID, error)

	// Payouts (Phase 2)
	RequestPayout(ctx context.Context, creatorID uuid.UUID) (*PayoutResult, error)

	// Event handlers (cross-domain reactions)
	HandleContentFlagged(ctx context.Context, listingID uuid.UUID, reason string) error
	ArchiveListingByContentKey(ctx context.Context, contentKey, reason string) error
	HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error

	// ─── Query side (read, no side effects) ─────────────────────────────

	// Creator queries
	GetCreatorByParentID(ctx context.Context, parentID uuid.UUID) (*CreatorResponse, error)
	GetCreatorDashboard(ctx context.Context, creatorID uuid.UUID, period DashboardPeriod) (*CreatorDashboardResponse, error)
	GetCreatorListings(ctx context.Context, creatorID uuid.UUID, params CreatorListingQueryParams) (*shared.PaginatedResponse[ListingDetailResponse], error)

	// Publisher queries
	GetPublisher(ctx context.Context, publisherID uuid.UUID) (*PublisherResponse, error)
	GetPublisherMembers(ctx context.Context, publisherID, creatorID uuid.UUID) ([]PublisherMemberResponse, error)
	VerifyPublisherMembership(ctx context.Context, publisherID, creatorID uuid.UUID) (bool, error)

	// Listing browse
	BrowseListings(ctx context.Context, params BrowseListingsParams) (*shared.PaginatedResponse[ListingBrowseResponse], error)
	GetListing(ctx context.Context, listingID uuid.UUID) (*ListingDetailResponse, error)
	AutocompleteListings(ctx context.Context, query string, limit uint8) ([]AutocompleteResult, error)
	GetCuratedSections(ctx context.Context, itemsPerSection uint8) ([]CuratedSectionResponse, error)

	// Cart queries
	GetCart(ctx context.Context, scope shared.FamilyScope) (*CartResponse, error)

	// Purchase queries
	GetPurchases(ctx context.Context, scope shared.FamilyScope, params PurchaseQueryParams) (*shared.PaginatedResponse[PurchaseResponse], error)
	GetDownloadURL(ctx context.Context, listingID, fileID uuid.UUID, scope shared.FamilyScope) (*DownloadResponse, error)

	// File queries
	GetListingFile(ctx context.Context, listingID, fileID uuid.UUID) (*ListingFileResponse, error)

	// Review queries
	GetReview(ctx context.Context, reviewID uuid.UUID) (*ReviewResponse, error)
	GetListingReviews(ctx context.Context, listingID uuid.UUID, params ReviewQueryParams) (*shared.PaginatedResponse[ReviewResponse], error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [CODING §8.2]
// ═══════════════════════════════════════════════════════════════════════════════

// CreatorRepository — NOT family-scoped (per-parent, not per-family).
type CreatorRepository interface {
	Create(ctx context.Context, cmd CreateCreator) (*MktCreator, error)
	GetByID(ctx context.Context, creatorID uuid.UUID) (*MktCreator, error)
	GetByParentID(ctx context.Context, parentID uuid.UUID) (*MktCreator, error)
	Update(ctx context.Context, creatorID uuid.UUID, cmd UpdateCreator) (*MktCreator, error)
	SetOnboardingStatus(ctx context.Context, creatorID uuid.UUID, status string) error
	SetPaymentAccountID(ctx context.Context, creatorID uuid.UUID, paymentAccountID string) error
}

// PublisherRepository — NOT family-scoped (organization-level).
type PublisherRepository interface {
	Create(ctx context.Context, cmd CreatePublisher) (*MktPublisher, error)
	GetByID(ctx context.Context, publisherID uuid.UUID) (*MktPublisher, error)
	GetBySlug(ctx context.Context, slug string) (*MktPublisher, error)
	Update(ctx context.Context, publisherID uuid.UUID, cmd UpdatePublisher) (*MktPublisher, error)
	GetPlatformPublisher(ctx context.Context) (*MktPublisher, error)
	CountMembers(ctx context.Context, publisherID uuid.UUID) (int32, error)

	// Membership
	AddMember(ctx context.Context, publisherID, creatorID uuid.UUID, role string) error
	RemoveMember(ctx context.Context, publisherID, creatorID uuid.UUID) error
	GetMembers(ctx context.Context, publisherID uuid.UUID) ([]PublisherMemberRow, error)
	GetMemberRole(ctx context.Context, publisherID, creatorID uuid.UUID) (*string, error)
	GetPublishersForCreator(ctx context.Context, creatorID uuid.UUID) ([]MktPublisher, error)
	CountOwners(ctx context.Context, publisherID uuid.UUID) (int32, error)
}

// ListingRepository — NOT family-scoped (publicly browsable).
type ListingRepository interface {
	// Command side
	Create(ctx context.Context, cmd CreateListing) (*MktListing, error)
	Save(ctx context.Context, listing *domain.MarketplaceListing) error
	CreateVersionSnapshot(ctx context.Context, listingID uuid.UUID, version int32, title, description string, priceCents int32, changeSummary *string) error

	// Query side
	GetByID(ctx context.Context, listingID uuid.UUID) (*MktListing, error)
	Browse(ctx context.Context, params *BrowseListingsParams) ([]ListingBrowseRow, int64, error)
	Autocomplete(ctx context.Context, query string, limit uint8) ([]AutocompleteRow, error)
	GetByCreator(ctx context.Context, creatorID uuid.UUID, params *CreatorListingQueryParams) ([]MktListing, int64, error)
	GetVersions(ctx context.Context, listingID uuid.UUID) ([]MktListingVersion, error)
	CountFiles(ctx context.Context, listingID uuid.UUID) (int64, error)
}

// ListingFileRepository
type ListingFileRepository interface {
	Create(ctx context.Context, cmd CreateListingFile) (*MktListingFile, error)
	GetByID(ctx context.Context, listingID, fileID uuid.UUID) (*MktListingFile, error)
	FindByStorageKey(ctx context.Context, storageKey string) (*MktListingFile, error)
	ListByListing(ctx context.Context, listingID uuid.UUID) ([]MktListingFile, error)
	Delete(ctx context.Context, fileID uuid.UUID) error
}

// CartRepository — family-scoped. [00-core §8]
type CartRepository interface {
	AddItem(ctx context.Context, listingID, parentID uuid.UUID, scope shared.FamilyScope) error
	RemoveItem(ctx context.Context, listingID uuid.UUID, scope shared.FamilyScope) error
	GetItems(ctx context.Context, scope shared.FamilyScope) ([]CartItemRow, error)
	Clear(ctx context.Context, scope shared.FamilyScope) error
}

// PurchaseRepository — family-scoped on reads, system-scoped on webhook writes.
type PurchaseRepository interface {
	Create(ctx context.Context, cmd CreatePurchase) (*MktPurchase, error)
	GetByFamilyAndListing(ctx context.Context, familyID, listingID uuid.UUID) (*MktPurchase, error)
	ListByFamily(ctx context.Context, scope shared.FamilyScope, params *PurchaseQueryParams) ([]PurchaseRow, int64, error)
	GetByPaymentSessionID(ctx context.Context, sessionID string) (*MktPurchase, error)
	SetRefund(ctx context.Context, purchaseID uuid.UUID, refundID string, refundAmountCents int32) error
	GetCreatorSales(ctx context.Context, creatorID uuid.UUID, from, to time.Time) ([]SalesRow, error)
	GetAllCreatorSales(ctx context.Context, from, to time.Time) ([]CreatorSalesAggregate, error)
	DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// ReviewRepository — family-scoped writes, public reads.
type ReviewRepository interface {
	Create(ctx context.Context, cmd CreateReview) (*MktReview, error)
	GetByID(ctx context.Context, reviewID uuid.UUID) (*MktReview, error)
	ExistsByFamilyAndListing(ctx context.Context, familyID, listingID uuid.UUID) (bool, error)
	Update(ctx context.Context, reviewID uuid.UUID, cmd UpdateReview) (*MktReview, error)
	Delete(ctx context.Context, reviewID uuid.UUID) error
	SetCreatorResponse(ctx context.Context, reviewID uuid.UUID, responseText string) error
	ListByListing(ctx context.Context, listingID uuid.UUID, params *ReviewQueryParams) ([]ReviewRow, int64, error)
	GetAggregateRating(ctx context.Context, listingID uuid.UUID) (float64, int32, error)
	UpdateListingRating(ctx context.Context, listingID uuid.UUID) error
	SetModerationStatus(ctx context.Context, reviewID uuid.UUID, status string) error
	AnonymizeByFamily(ctx context.Context, familyID uuid.UUID) error
}

// CuratedSectionRepository
type CuratedSectionRepository interface {
	ListActive(ctx context.Context) ([]MktCuratedSection, error)
	GetSectionItems(ctx context.Context, sectionID uuid.UUID, limit uint8) ([]ListingBrowseRow, error)
	AddItem(ctx context.Context, sectionID, listingID uuid.UUID, sortOrder int16) error
	RemoveItem(ctx context.Context, sectionID, listingID uuid.UUID) error
	RefreshAutoSection(ctx context.Context, sectionSlug string) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Adapter Interfaces [CODING §8.1]
// ═══════════════════════════════════════════════════════════════════════════════

// PaymentAdapter wraps the payment processor (Hyperswitch). [ARCH §4.3]
type PaymentAdapter interface {
	CreateSubMerchant(ctx context.Context, config SubMerchantConfig) (string, error)
	CreateOnboardingLink(ctx context.Context, paymentAccountID, returnURL string) (string, error)
	GetAccountStatus(ctx context.Context, paymentAccountID string) (PaymentAccountStatus, error)
	CreatePayment(ctx context.Context, lineItems []PaymentLineItem, splitRules []SplitRule, returnURL string, metadata map[string]string) (*PaymentSession, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (PaymentStatus, error)
	CreatePayout(ctx context.Context, paymentAccountID string, amountCents int64, currency string) (*PayoutResult, error)
	CreateRefund(ctx context.Context, paymentID string, amountCents int64, reason string) (*RefundResult, error)
	VerifyWebhook(ctx context.Context, payload []byte, signature string) (bool, error)
	ParseEvent(ctx context.Context, payload []byte) (*PaymentEvent, error)
}

// MediaAdapter delegates file operations to media::. [ARCH §4.2]
type MediaAdapter interface {
	PresignedUpload(ctx context.Context, key, contentType string, maxSizeBytes uint64) (string, error)
	PresignedGet(ctx context.Context, key string, expiresSeconds uint32) (string, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Adapter Supporting Types [CODING §8.1a]
// ═══════════════════════════════════════════════════════════════════════════════

type SubMerchantConfig struct {
	CreatorID uuid.UUID `json:"creator_id"`
	StoreName string    `json:"store_name"`
	Email     string    `json:"email"`
	Country   string    `json:"country"`
}

type PaymentAccountStatus int

const (
	PaymentAccountStatusPending    PaymentAccountStatus = iota
	PaymentAccountStatusOnboarding
	PaymentAccountStatusActive
	PaymentAccountStatusSuspended
)

type PaymentLineItem struct {
	ListingID   uuid.UUID `json:"listing_id"`
	AmountCents int64     `json:"amount_cents"`
	Description string    `json:"description"`
}

type SplitRule struct {
	RecipientAccountID string `json:"recipient_account_id"`
	AmountCents        int64  `json:"amount_cents"`
}

type PaymentSession struct {
	CheckoutURL      string `json:"checkout_url"`
	PaymentSessionID string `json:"payment_session_id"`
}

type PaymentStatus int

const (
	PaymentStatusProcessing PaymentStatus = iota
	PaymentStatusSucceeded
	PaymentStatusFailed
	PaymentStatusCancelled
)

type PayoutResult struct {
	PayoutID    string `json:"payout_id"`
	AmountCents int64  `json:"amount_cents"`
	Status      string `json:"status"`
}

type RefundResult struct {
	RefundID    string `json:"refund_id"`
	AmountCents int64  `json:"amount_cents"`
	Status      string `json:"status"`
}

type PaymentEvent struct {
	Type        string
	PaymentID   string
	Metadata    map[string]string
	Reason      string
	RefundID    string
	AmountCents int64
	MerchantID  string
	PayoutID    string
}
