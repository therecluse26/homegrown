package mkt

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ─── Custom DB Array Types ──────────────────────────────────────────────────

// StringArray is a custom type for PostgreSQL TEXT[] columns.
// Implements database/sql.Scanner and driver.Valuer without requiring lib/pq.
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	return "{" + strings.Join(a, ",") + "}", nil
}

func (a *StringArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("StringArray.Scan: unsupported type %T", src)
	}
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*a = StringArray{}
		return nil
	}
	parts := strings.Split(str, ",")
	result := make(StringArray, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	*a = result
	return nil
}

// UUIDArray is a custom type for PostgreSQL UUID[] columns.
type UUIDArray []string

func (a UUIDArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	return "{" + strings.Join(a, ",") + "}", nil
}

func (a *UUIDArray) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}
	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("UUIDArray.Scan: unsupported type %T", src)
	}
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*a = UUIDArray{}
		return nil
	}
	parts := strings.Split(str, ",")
	result := make(UUIDArray, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(p)
	}
	*a = result
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// GORM Models
// ═══════════════════════════════════════════════════════════════════════════════

// MktPublisher is the GORM model for mkt_publishers.
type MktPublisher struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Name        string    `gorm:"not null"`
	Slug        string    `gorm:"not null;uniqueIndex"`
	Description *string
	LogoURL     *string   `gorm:"column:logo_url"`
	WebsiteURL  *string   `gorm:"column:website_url"`
	IsPlatform  bool      `gorm:"not null;default:false"`
	IsVerified  bool      `gorm:"not null;default:false"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (MktPublisher) TableName() string { return "mkt_publishers" }

// MktPublisherMember is the GORM model for mkt_publisher_members.
type MktPublisherMember struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PublisherID uuid.UUID `gorm:"type:uuid;not null"`
	CreatorID   uuid.UUID `gorm:"type:uuid;not null"`
	Role        string    `gorm:"not null;default:'member'"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
}

func (MktPublisherMember) TableName() string { return "mkt_publisher_members" }

// MktCreator is the GORM model for mkt_creators.
type MktCreator struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ParentID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	PaymentAccountID *string    `gorm:"column:payment_account_id"`
	OnboardingStatus string     `gorm:"not null;default:'pending'"`
	StoreName        string     `gorm:"not null"`
	StoreBio         *string
	StoreLogoURL     *string    `gorm:"column:store_logo_url"`
	StoreBannerURL   *string    `gorm:"column:store_banner_url"`
	TOSAcceptedAt    *time.Time `gorm:"column:tos_accepted_at"`
	CreatedAt        time.Time  `gorm:"not null;default:now()"`
	UpdatedAt        time.Time  `gorm:"not null;default:now()"`
}

func (MktCreator) TableName() string { return "mkt_creators" }

// MktListing is the GORM model for mkt_listings.
type MktListing struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatorID       uuid.UUID      `gorm:"type:uuid;not null"`
	PublisherID     uuid.UUID      `gorm:"type:uuid;not null"`
	Title           string         `gorm:"not null"`
	Description     string         `gorm:"not null"`
	PriceCents      int32          `gorm:"not null"`
	MethodologyTags UUIDArray   `gorm:"type:uuid[];not null"`
	SubjectTags     StringArray `gorm:"type:text[];not null"`
	GradeMin        *int16
	GradeMax        *int16
	ContentType     string         `gorm:"not null"`
	WorldviewTags   StringArray `gorm:"type:text[];default:'{}'"`
	PreviewURL      *string        `gorm:"column:preview_url"`
	ThumbnailURL    *string        `gorm:"column:thumbnail_url"`
	Status          string         `gorm:"not null;default:'draft'"`
	RatingAvg       float64        `gorm:"type:numeric(3,2);default:0"`
	RatingCount     int32          `gorm:"default:0"`
	Version         int32          `gorm:"not null;default:1"`
	PublishedAt     *time.Time
	ArchivedAt      *time.Time
	CreatedAt       time.Time `gorm:"not null;default:now()"`
	UpdatedAt       time.Time `gorm:"not null;default:now()"`
}

func (MktListing) TableName() string { return "mkt_listings" }

// MktListingFile is the GORM model for mkt_listing_files.
type MktListingFile struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ListingID     uuid.UUID `gorm:"type:uuid;not null"`
	FileName      string    `gorm:"not null"`
	FileSizeBytes int64     `gorm:"not null"`
	MimeType      string    `gorm:"not null"`
	StorageKey    string    `gorm:"not null"`
	SortOrder     int16     `gorm:"not null;default:0"`
	Version       int32     `gorm:"not null;default:1"`
	CreatedAt     time.Time `gorm:"not null;default:now()"`
}

func (MktListingFile) TableName() string { return "mkt_listing_files" }

// MktListingVersion is the GORM model for mkt_listing_versions.
type MktListingVersion struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ListingID          uuid.UUID `gorm:"type:uuid;not null"`
	Version            int32     `gorm:"not null"`
	Title              string    `gorm:"not null"`
	Description        string    `gorm:"not null"`
	PriceCents         int32     `gorm:"not null"`
	ChangeSummary      *string
	UpgradePolicy      *string `gorm:"default:'free'" json:"upgrade_policy,omitempty"`
	UpgradeDiscountPct *int16  `json:"upgrade_discount_pct,omitempty"`
	CreatedAt          time.Time `gorm:"not null;default:now()"`
}

func (MktListingVersion) TableName() string { return "mkt_listing_versions" }

// MktPurchase is the GORM model for mkt_purchases.
type MktPurchase struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID          uuid.UUID  `gorm:"type:uuid;not null"`
	ListingID         uuid.UUID  `gorm:"type:uuid;not null"`
	CreatorID         uuid.UUID  `gorm:"type:uuid;not null"`
	PaymentID         *string    `gorm:"column:payment_id"`
	PaymentSessionID  *string    `gorm:"column:payment_session_id;uniqueIndex"`
	AmountCents       int32      `gorm:"not null"`
	PlatformFeeCents  int32      `gorm:"not null"`
	CreatorPayoutCents int32     `gorm:"not null"`
	RefundedAt        *time.Time
	RefundAmountCents *int32
	RefundID          *string    `gorm:"column:refund_id"`
	CreatedAt         time.Time  `gorm:"not null;default:now()"`
}

func (MktPurchase) TableName() string { return "mkt_purchases" }

// MktReview is the GORM model for mkt_reviews.
type MktReview struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ListingID         uuid.UUID  `gorm:"type:uuid;not null"`
	PurchaseID        uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	FamilyID          uuid.UUID  `gorm:"type:uuid;not null"`
	Rating            int16      `gorm:"not null"`
	ReviewText        *string
	IsAnonymous       bool       `gorm:"not null;default:true"`
	ModerationStatus  string     `gorm:"not null;default:'pending'"`
	CreatorResponse   *string
	CreatorResponseAt *time.Time
	CreatedAt         time.Time  `gorm:"not null;default:now()"`
	UpdatedAt         time.Time  `gorm:"not null;default:now()"`
}

func (MktReview) TableName() string { return "mkt_reviews" }

// MktCartItem is the GORM model for mkt_cart_items.
type MktCartItem struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID       uuid.UUID `gorm:"type:uuid;not null"`
	ListingID      uuid.UUID `gorm:"type:uuid;not null"`
	AddedByParentID uuid.UUID `gorm:"type:uuid;not null;column:added_by_parent_id"`
	CreatedAt      time.Time `gorm:"not null;default:now()"`
}

func (MktCartItem) TableName() string { return "mkt_cart_items" }

// MktCuratedSection is the GORM model for mkt_curated_sections.
type MktCuratedSection struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Slug        string    `gorm:"not null;uniqueIndex"`
	DisplayName string    `gorm:"not null"`
	Description *string
	SectionType string    `gorm:"not null"`
	SortOrder   int16     `gorm:"not null;default:0"`
	IsActive    bool      `gorm:"not null;default:true"`
	CreatedAt   time.Time `gorm:"not null;default:now()"`
	UpdatedAt   time.Time `gorm:"not null;default:now()"`
}

func (MktCuratedSection) TableName() string { return "mkt_curated_sections" }

// MktCuratedSectionItem is the GORM model for mkt_curated_section_items.
type MktCuratedSectionItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	SectionID uuid.UUID `gorm:"type:uuid;not null"`
	ListingID uuid.UUID `gorm:"type:uuid;not null"`
	SortOrder int16     `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (MktCuratedSectionItem) TableName() string { return "mkt_curated_section_items" }

// ═══════════════════════════════════════════════════════════════════════════════
// Request Types (API input) [CODING §2.3]
// ═══════════════════════════════════════════════════════════════════════════════

type RegisterCreatorCommand struct {
	StoreName    string  `json:"store_name" validate:"required,min=1,max=100"`
	StoreBio     *string `json:"store_bio,omitempty"`
	StoreLogoURL *string `json:"store_logo_url,omitempty"`
	TOSAccepted  bool    `json:"tos_accepted" validate:"required"`
}

type UpdateCreatorProfileCommand struct {
	StoreName      *string `json:"store_name,omitempty" validate:"omitempty,min=1,max=100"`
	StoreBio       *string `json:"store_bio,omitempty"`
	StoreLogoURL   *string `json:"store_logo_url,omitempty"`
	StoreBannerURL *string `json:"store_banner_url,omitempty"`
}

type CreatePublisherCommand struct {
	Name        string  `json:"name" validate:"required,min=1,max=100"`
	Slug        *string `json:"slug,omitempty"`
	Description *string `json:"description,omitempty"`
	LogoURL     *string `json:"logo_url,omitempty"`
	WebsiteURL  *string `json:"website_url,omitempty"`
}

type UpdatePublisherCommand struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description *string `json:"description,omitempty"`
	LogoURL     *string `json:"logo_url,omitempty"`
	WebsiteURL  *string `json:"website_url,omitempty"`
}

type CreateListingCommand struct {
	PublisherID     uuid.UUID   `json:"publisher_id" validate:"required"`
	Title           string      `json:"title" validate:"required,min=1,max=200"`
	Description     string      `json:"description" validate:"required,min=1,max=10000"`
	PriceCents      int32       `json:"price_cents" validate:"gte=0"`
	MethodologyTags []uuid.UUID `json:"methodology_tags" validate:"required,min=1"`
	SubjectTags     []string    `json:"subject_tags" validate:"required,min=1"`
	GradeMin        *int16      `json:"grade_min,omitempty"`
	GradeMax        *int16      `json:"grade_max,omitempty"`
	ContentType     string      `json:"content_type" validate:"required"`
	WorldviewTags   []string    `json:"worldview_tags,omitempty"`
	PreviewURL      *string     `json:"preview_url,omitempty"`
	ThumbnailURL    *string     `json:"thumbnail_url,omitempty"`
}

type UpdateListingCommand struct {
	Title           *string      `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description     *string      `json:"description,omitempty" validate:"omitempty,min=1,max=10000"`
	PriceCents      *int32       `json:"price_cents,omitempty" validate:"omitempty,gte=0"`
	MethodologyTags []uuid.UUID  `json:"methodology_tags,omitempty"`
	SubjectTags     []string     `json:"subject_tags,omitempty"`
	GradeMin        *int16       `json:"grade_min,omitempty"`
	GradeMax        *int16       `json:"grade_max,omitempty"`
	WorldviewTags   []string     `json:"worldview_tags,omitempty"`
	PreviewURL      *string      `json:"preview_url,omitempty"`
	ThumbnailURL    *string      `json:"thumbnail_url,omitempty"`
	ChangeSummary   *string      `json:"change_summary,omitempty"`
}

type UploadListingFileCommand struct {
	FileName      string `json:"file_name" validate:"required"`
	FileSizeBytes int64  `json:"file_size_bytes" validate:"required"`
	MimeType      string `json:"mime_type" validate:"required"`
}

type AddToCartCommand struct {
	ListingID uuid.UUID `json:"listing_id" validate:"required"`
}

type CreateReviewCommand struct {
	Rating      int16   `json:"rating" validate:"required,min=1,max=5"`
	ReviewText  *string `json:"review_text,omitempty" validate:"omitempty,max=5000"`
	IsAnonymous *bool   `json:"is_anonymous,omitempty"`
}

type UpdateReviewCommand struct {
	Rating      *int16  `json:"rating,omitempty" validate:"omitempty,min=1,max=5"`
	ReviewText  *string `json:"review_text,omitempty" validate:"omitempty,max=5000"`
	IsAnonymous *bool   `json:"is_anonymous,omitempty"`
}

type RespondToReviewCommand struct {
	ResponseText string `json:"response_text" validate:"required"`
}

type AddPublisherMemberCommand struct {
	CreatorID uuid.UUID `json:"creator_id" validate:"required"`
	Role      string    `json:"role" validate:"required,oneof=owner admin member"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Types (API output)
// ═══════════════════════════════════════════════════════════════════════════════

type CreatorResponse struct {
	ID               uuid.UUID  `json:"id"`
	ParentID         uuid.UUID  `json:"parent_id"`
	OnboardingStatus string     `json:"onboarding_status"`
	StoreName        string     `json:"store_name"`
	StoreBio         *string    `json:"store_bio,omitempty"`
	StoreLogoURL     *string    `json:"store_logo_url,omitempty"`
	StoreBannerURL   *string    `json:"store_banner_url,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type PublisherResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description *string   `json:"description,omitempty"`
	LogoURL     *string   `json:"logo_url,omitempty"`
	WebsiteURL  *string   `json:"website_url,omitempty"`
	IsVerified  bool      `json:"is_verified"`
	MemberCount int32     `json:"member_count"`
}

type PublisherMemberResponse struct {
	CreatorID uuid.UUID `json:"creator_id"`
	StoreName string    `json:"store_name"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}

type ListingBrowseResponse struct {
	ID                 uuid.UUID `json:"id"`
	Title              string    `json:"title"`
	DescriptionPreview string    `json:"description_preview"`
	PriceCents         int32     `json:"price_cents"`
	ContentType        string    `json:"content_type"`
	ThumbnailURL       *string   `json:"thumbnail_url,omitempty"`
	RatingAvg          float64   `json:"rating_avg"`
	RatingCount        int32     `json:"rating_count"`
	PublisherName      string    `json:"publisher_name"`
	CreatorStoreName   string    `json:"creator_store_name"`
}

type ListingDetailResponse struct {
	ID              uuid.UUID             `json:"id"`
	CreatorID       uuid.UUID             `json:"creator_id"`
	PublisherID     uuid.UUID             `json:"publisher_id"`
	PublisherName   string                `json:"publisher_name"`
	Title           string                `json:"title"`
	Description     string                `json:"description"`
	PriceCents      int32                 `json:"price_cents"`
	MethodologyTags []string              `json:"methodology_tags"`
	SubjectTags     []string              `json:"subject_tags"`
	GradeMin        *int16                `json:"grade_min,omitempty"`
	GradeMax        *int16                `json:"grade_max,omitempty"`
	ContentType     string                `json:"content_type"`
	WorldviewTags   []string              `json:"worldview_tags"`
	PreviewURL      *string               `json:"preview_url,omitempty"`
	ThumbnailURL    *string               `json:"thumbnail_url,omitempty"`
	Status          string                `json:"status"`
	RatingAvg       float64               `json:"rating_avg"`
	RatingCount     int32                 `json:"rating_count"`
	Version         int32                 `json:"version"`
	Files           []ListingFileResponse `json:"files"`
	PublishedAt     *time.Time            `json:"published_at,omitempty"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

type ListingFileResponse struct {
	ID            uuid.UUID `json:"id"`
	FileName      string    `json:"file_name"`
	FileSizeBytes int64     `json:"file_size_bytes"`
	MimeType      string    `json:"mime_type"`
	Version       int32     `json:"version"`
}

type CartResponse struct {
	Items      []CartItemResponse `json:"items"`
	TotalCents int64              `json:"total_cents"`
	ItemCount  int32              `json:"item_count"`
}

type CartItemResponse struct {
	ListingID    uuid.UUID `json:"listing_id"`
	Title        string    `json:"title"`
	PriceCents   int32     `json:"price_cents"`
	ThumbnailURL *string   `json:"thumbnail_url,omitempty"`
	AddedAt      time.Time `json:"added_at"`
}

type PurchaseResponse struct {
	ID           uuid.UUID `json:"id"`
	ListingID    uuid.UUID `json:"listing_id"`
	ListingTitle string    `json:"listing_title"`
	AmountCents  int32     `json:"amount_cents"`
	Refunded     bool      `json:"refunded"`
	CreatedAt    time.Time `json:"created_at"`
}

type ReviewResponse struct {
	ID                uuid.UUID  `json:"id"`
	ListingID         uuid.UUID  `json:"listing_id"`
	Rating            int16      `json:"rating"`
	ReviewText        *string    `json:"review_text,omitempty"`
	IsAnonymous       bool       `json:"is_anonymous"`
	ReviewerName      *string    `json:"reviewer_name,omitempty"`
	CreatorResponse   *string    `json:"creator_response,omitempty"`
	CreatorResponseAt *time.Time `json:"creator_response_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type DownloadResponse struct {
	DownloadURL string    `json:"download_url"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type AutocompleteResult struct {
	ListingID  uuid.UUID `json:"listing_id"`
	Title      string    `json:"title"`
	Similarity float32   `json:"similarity"`
}

type CuratedSectionResponse struct {
	Slug        string                 `json:"slug"`
	DisplayName string                 `json:"display_name"`
	Description *string                `json:"description,omitempty"`
	Listings    []ListingBrowseResponse `json:"listings"`
}

type CreatorDashboardResponse struct {
	TotalSalesCount     int64         `json:"total_sales_count"`
	TotalEarningsCents  int64         `json:"total_earnings_cents"`
	PeriodSalesCount    int64         `json:"period_sales_count"`
	PeriodEarningsCents int64         `json:"period_earnings_cents"`
	PendingPayoutCents  int64         `json:"pending_payout_cents"`
	AverageRating       float64       `json:"average_rating"`
	TotalReviews        int32         `json:"total_reviews"`
	RecentSales         []SaleSummary `json:"recent_sales"`
}

type SaleSummary struct {
	PurchaseID         uuid.UUID `json:"purchase_id"`
	ListingTitle       string    `json:"listing_title"`
	AmountCents        int32     `json:"amount_cents"`
	CreatorPayoutCents int32     `json:"creator_payout_cents"`
	PurchasedAt        time.Time `json:"purchased_at"`
}

type ListingVersionResponse struct {
	Version       int32     `json:"version"`
	Title         string    `json:"title"`
	PriceCents    int32     `json:"price_cents"`
	ChangeSummary *string   `json:"change_summary,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

type CheckoutSessionResponse struct {
	CheckoutURL      string `json:"checkout_url"`
	PaymentSessionID string `json:"payment_session_id"`
}

type OnboardingLinkResponse struct {
	OnboardingURL string `json:"onboarding_url"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Query Parameter Types (internal, not API responses)
// ═══════════════════════════════════════════════════════════════════════════════

// ListingSortBy defines valid sort options for listing browse. [07-mkt §8.3]
type ListingSortBy string

const (
	ListingSortByRelevance ListingSortBy = "relevance"
	ListingSortByPriceAsc  ListingSortBy = "price_asc"
	ListingSortByPriceDesc ListingSortBy = "price_desc"
	ListingSortByRating    ListingSortBy = "rating"
	ListingSortByNewest    ListingSortBy = "newest"
)

// ReviewSortBy defines valid sort options for review queries. [07-mkt §8.3]
type ReviewSortBy string

const (
	ReviewSortByNewest ReviewSortBy = "newest"
	ReviewSortByRating ReviewSortBy = "rating"
	ReviewSortByOldest ReviewSortBy = "oldest"
)

type BrowseListingsParams struct {
	Q              *string        `query:"q"`
	MethodologyIDs []uuid.UUID    `query:"methodology_ids"`
	SubjectSlugs   []string       `query:"subject_slugs"`
	GradeMin       *int16         `query:"grade_min"`
	GradeMax       *int16         `query:"grade_max"`
	ContentType    *string        `query:"content_type"`
	WorldviewTags  []string       `query:"worldview_tags"`
	PriceMin       *int32         `query:"price_min"`
	PriceMax       *int32         `query:"price_max"`
	MinRating      *float64       `query:"min_rating"`
	SortBy         *ListingSortBy `query:"sort_by"`
	Cursor         *string        `query:"cursor"`
	Limit          *uint8         `query:"limit"`
}

type CreatorListingQueryParams struct {
	Status *string `query:"status"`
	Cursor *string `query:"cursor"`
	Limit  *uint8  `query:"limit"`
}

type PurchaseQueryParams struct {
	Cursor *string `query:"cursor"`
	Limit  *uint8  `query:"limit"`
}

type ReviewQueryParams struct {
	SortBy *ReviewSortBy `query:"sort_by"`
	Cursor *string       `query:"cursor"`
	Limit  *uint8        `query:"limit"`
}

type DashboardPeriod string

const (
	DashboardPeriodLast7Days  DashboardPeriod = "last_7_days"
	DashboardPeriodLast30Days DashboardPeriod = "last_30_days"
	DashboardPeriodLast90Days DashboardPeriod = "last_90_days"
	DashboardPeriodAllTime    DashboardPeriod = "all_time"
)

// ToDateRange converts a DashboardPeriod to a time range.
func (p DashboardPeriod) ToDateRange() (time.Time, time.Time) {
	now := time.Now().UTC()
	switch p {
	case DashboardPeriodLast7Days:
		return now.AddDate(0, 0, -7), now
	case DashboardPeriodLast90Days:
		return now.AddDate(0, -3, 0), now
	case DashboardPeriodAllTime:
		return time.Time{}, now
	default:
		return now.AddDate(0, 0, -30), now
	}
}

// ParseDashboardPeriod validates a period string.
func ParseDashboardPeriod(s string) DashboardPeriod {
	switch s {
	case "last_7_days":
		return DashboardPeriodLast7Days
	case "last_90_days":
		return DashboardPeriodLast90Days
	case "all_time":
		return DashboardPeriodAllTime
	default:
		return DashboardPeriodLast30Days
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Internal Row Types (returned by repositories, not exposed in API)
// ═══════════════════════════════════════════════════════════════════════════════

type ListingBrowseRow struct {
	ID               uuid.UUID
	Title            string
	Description      string
	PriceCents       int32
	ContentType      string
	ThumbnailURL     *string
	RatingAvg        float64
	RatingCount      int32
	PublisherName    string
	CreatorStoreName string
}

type AutocompleteRow struct {
	ListingID  uuid.UUID
	Title      string
	Similarity float32
}

type CartItemRow struct {
	ListingID    uuid.UUID
	Title        string
	PriceCents   int32
	ThumbnailURL *string
	CreatedAt    time.Time
}

type PurchaseRow struct {
	ID           uuid.UUID
	ListingID    uuid.UUID
	ListingTitle string
	AmountCents  int32
	RefundedAt   *time.Time
	CreatedAt    time.Time
}

type ReviewRow struct {
	ID                 uuid.UUID
	ListingID          uuid.UUID
	Rating             int16
	ReviewText         *string
	IsAnonymous        bool
	ReviewerFamilyName *string
	CreatorResponse    *string
	CreatorResponseAt  *time.Time
	ModerationStatus   string
	CreatedAt          time.Time
}

type SalesRow struct {
	PurchaseID         uuid.UUID
	ListingID          uuid.UUID
	ListingTitle       string
	AmountCents        int32
	CreatorPayoutCents int32
	CreatedAt          time.Time
}

type PublisherMemberRow struct {
	CreatorID uuid.UUID
	StoreName string
	Role      string
	CreatedAt time.Time
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Command Types (used as inputs to repository Create/Update methods)
// ═══════════════════════════════════════════════════════════════════════════════

type CreateCreator struct {
	ParentID      uuid.UUID
	StoreName     string
	StoreBio      *string
	StoreLogoURL  *string
	TOSAcceptedAt *time.Time
}

type UpdateCreator struct {
	StoreName      *string
	StoreBio       *string
	StoreLogoURL   *string
	StoreBannerURL *string
}

type CreatePublisher struct {
	Name        string
	Slug        string
	Description *string
	LogoURL     *string
	WebsiteURL  *string
}

type UpdatePublisher struct {
	Name        *string
	Description *string
	LogoURL     *string
	WebsiteURL  *string
}

type CreateListing struct {
	CreatorID       uuid.UUID
	PublisherID     uuid.UUID
	Title           string
	Description     string
	PriceCents      int32
	MethodologyTags []uuid.UUID
	SubjectTags     []string
	GradeMin        *int16
	GradeMax        *int16
	ContentType     string
	WorldviewTags   []string
	PreviewURL      *string
	ThumbnailURL    *string
}

type CreateListingFile struct {
	ListingID     uuid.UUID
	FileName      string
	FileSizeBytes int64
	MimeType      string
	StorageKey    string
	SortOrder     int16
}

type CreatePurchase struct {
	FamilyID           uuid.UUID
	ListingID          uuid.UUID
	CreatorID          uuid.UUID
	PaymentID          *string
	PaymentSessionID   *string
	AmountCents        int32
	PlatformFeeCents   int32
	CreatorPayoutCents int32
}

type CreateReview struct {
	ListingID   uuid.UUID
	PurchaseID  uuid.UUID
	FamilyID    uuid.UUID
	Rating      int16
	ReviewText  *string
	IsAnonymous bool
}

type UpdateReview struct {
	Rating      *int16
	ReviewText  *string
	IsAnonymous *bool
}

// ValidContentTypes is the set of valid content types for listings. [S§9.2.1]
var ValidContentTypes = map[string]bool{
	"curriculum":       true,
	"worksheet":        true,
	"unit_study":       true,
	"video":            true,
	"book_list":        true,
	"assessment":       true,
	"lesson_plan":      true,
	"printable":        true,
	"project_guide":    true,
	"reading_guide":    true,
	"course":           true,
	"interactive_quiz": true,
	"lesson_sequence":  true,
}
