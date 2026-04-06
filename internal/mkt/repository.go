package mkt

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// PgCreatorRepository
// ═══════════════════════════════════════════════════════════════════════════════

type PgCreatorRepository struct{ db *gorm.DB }

func NewPgCreatorRepository(db *gorm.DB) CreatorRepository {
	return &PgCreatorRepository{db: db}
}

func (r *PgCreatorRepository) Create(ctx context.Context, cmd CreateCreator) (*MktCreator, error) {
	creator := MktCreator{
		ParentID:         cmd.ParentID,
		OnboardingStatus: "pending",
		StoreName:        cmd.StoreName,
		StoreBio:         cmd.StoreBio,
		StoreLogoURL:     cmd.StoreLogoURL,
		TOSAcceptedAt:    cmd.TOSAcceptedAt,
	}
	if err := r.db.WithContext(ctx).Create(&creator).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrCreatorAlreadyExists
		}
		return nil, shared.ErrDatabase(err)
	}
	return &creator, nil
}

func (r *PgCreatorRepository) GetByID(ctx context.Context, creatorID uuid.UUID) (*MktCreator, error) {
	var creator MktCreator
	if err := r.db.WithContext(ctx).Where("id = ?", creatorID).First(&creator).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCreatorNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &creator, nil
}

func (r *PgCreatorRepository) GetByParentID(ctx context.Context, parentID uuid.UUID) (*MktCreator, error) {
	var creator MktCreator
	if err := r.db.WithContext(ctx).Where("parent_id = ?", parentID).First(&creator).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCreatorNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &creator, nil
}

func (r *PgCreatorRepository) Update(ctx context.Context, creatorID uuid.UUID, cmd UpdateCreator) (*MktCreator, error) {
	updates := map[string]any{}
	if cmd.StoreName != nil {
		updates["store_name"] = *cmd.StoreName
	}
	if cmd.StoreBio != nil {
		updates["store_bio"] = *cmd.StoreBio
	}
	if cmd.StoreLogoURL != nil {
		updates["store_logo_url"] = *cmd.StoreLogoURL
	}
	if cmd.StoreBannerURL != nil {
		updates["store_banner_url"] = *cmd.StoreBannerURL
	}
	updates["updated_at"] = time.Now().UTC()

	if err := r.db.WithContext(ctx).Model(&MktCreator{}).Where("id = ?", creatorID).Updates(updates).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return r.GetByID(ctx, creatorID)
}

func (r *PgCreatorRepository) SetOnboardingStatus(ctx context.Context, creatorID uuid.UUID, status string) error {
	err := r.db.WithContext(ctx).Model(&MktCreator{}).Where("id = ?", creatorID).
		Updates(map[string]any{"onboarding_status": status, "updated_at": time.Now().UTC()}).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgCreatorRepository) SetPaymentAccountID(ctx context.Context, creatorID uuid.UUID, paymentAccountID string) error {
	err := r.db.WithContext(ctx).Model(&MktCreator{}).Where("id = ?", creatorID).
		Updates(map[string]any{"payment_account_id": paymentAccountID, "updated_at": time.Now().UTC()}).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgPublisherRepository
// ═══════════════════════════════════════════════════════════════════════════════

type PgPublisherRepository struct{ db *gorm.DB }

func NewPgPublisherRepository(db *gorm.DB) PublisherRepository {
	return &PgPublisherRepository{db: db}
}

func (r *PgPublisherRepository) Create(ctx context.Context, cmd CreatePublisher) (*MktPublisher, error) {
	pub := MktPublisher{
		Name:        cmd.Name,
		Slug:        cmd.Slug,
		Description: cmd.Description,
		LogoURL:     cmd.LogoURL,
		WebsiteURL:  cmd.WebsiteURL,
	}
	if err := r.db.WithContext(ctx).Create(&pub).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrPublisherSlugConflict
		}
		return nil, shared.ErrDatabase(err)
	}
	return &pub, nil
}

func (r *PgPublisherRepository) GetByID(ctx context.Context, publisherID uuid.UUID) (*MktPublisher, error) {
	var pub MktPublisher
	if err := r.db.WithContext(ctx).Where("id = ?", publisherID).First(&pub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublisherNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &pub, nil
}

func (r *PgPublisherRepository) GetBySlug(ctx context.Context, slug string) (*MktPublisher, error) {
	var pub MktPublisher
	if err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&pub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublisherNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &pub, nil
}

func (r *PgPublisherRepository) Update(ctx context.Context, publisherID uuid.UUID, cmd UpdatePublisher) (*MktPublisher, error) {
	updates := map[string]any{}
	if cmd.Name != nil {
		updates["name"] = *cmd.Name
	}
	if cmd.Description != nil {
		updates["description"] = *cmd.Description
	}
	if cmd.LogoURL != nil {
		updates["logo_url"] = *cmd.LogoURL
	}
	if cmd.WebsiteURL != nil {
		updates["website_url"] = *cmd.WebsiteURL
	}
	updates["updated_at"] = time.Now().UTC()

	if err := r.db.WithContext(ctx).Model(&MktPublisher{}).Where("id = ?", publisherID).Updates(updates).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return r.GetByID(ctx, publisherID)
}

func (r *PgPublisherRepository) GetPlatformPublisher(ctx context.Context) (*MktPublisher, error) {
	var pub MktPublisher
	if err := r.db.WithContext(ctx).Where("is_platform = true").First(&pub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublisherNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &pub, nil
}

func (r *PgPublisherRepository) CountMembers(ctx context.Context, publisherID uuid.UUID) (int32, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&MktPublisherMember{}).Where("publisher_id = ?", publisherID).Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return int32(count), nil
}

func (r *PgPublisherRepository) AddMember(ctx context.Context, publisherID, creatorID uuid.UUID, role string) error {
	member := MktPublisherMember{
		PublisherID: publisherID,
		CreatorID:   creatorID,
		Role:        role,
	}
	if err := r.db.WithContext(ctx).Create(&member).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrNotPublisherMember // already a member
		}
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgPublisherRepository) RemoveMember(ctx context.Context, publisherID, creatorID uuid.UUID) error {
	result := r.db.WithContext(ctx).Where("publisher_id = ? AND creator_id = ?", publisherID, creatorID).Delete(&MktPublisherMember{})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotPublisherMember
	}
	return nil
}

func (r *PgPublisherRepository) GetMembers(ctx context.Context, publisherID uuid.UUID) ([]PublisherMemberRow, error) {
	var rows []PublisherMemberRow
	err := r.db.WithContext(ctx).
		Table("mkt_publisher_members m").
		Select("m.creator_id, c.store_name, m.role, m.created_at").
		Joins("JOIN mkt_creators c ON c.id = m.creator_id").
		Where("m.publisher_id = ?", publisherID).
		Order("m.created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return rows, nil
}

func (r *PgPublisherRepository) GetMemberRole(ctx context.Context, publisherID, creatorID uuid.UUID) (*string, error) {
	var member MktPublisherMember
	err := r.db.WithContext(ctx).Where("publisher_id = ? AND creator_id = ?", publisherID, creatorID).First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotPublisherMember
		}
		return nil, shared.ErrDatabase(err)
	}
	return &member.Role, nil
}

func (r *PgPublisherRepository) GetPublishersForCreator(ctx context.Context, creatorID uuid.UUID) ([]MktPublisher, error) {
	var publishers []MktPublisher
	err := r.db.WithContext(ctx).
		Table("mkt_publishers p").
		Joins("JOIN mkt_publisher_members m ON m.publisher_id = p.id").
		Where("m.creator_id = ?", creatorID).
		Order("p.name ASC").
		Find(&publishers).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return publishers, nil
}

func (r *PgPublisherRepository) CountOwners(ctx context.Context, publisherID uuid.UUID) (int32, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&MktPublisherMember{}).
		Where("publisher_id = ? AND role = 'owner'", publisherID).Count(&count).Error
	if err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return int32(count), nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgListingRepository
// ═══════════════════════════════════════════════════════════════════════════════

type PgListingRepository struct{ db *gorm.DB }

func NewPgListingRepository(db *gorm.DB) ListingRepository {
	return &PgListingRepository{db: db}
}

func (r *PgListingRepository) Create(ctx context.Context, cmd CreateListing) (*MktListing, error) {
	// Convert UUID slices to our custom array types
	methodTags := make(UUIDArray, len(cmd.MethodologyTags))
	for i, id := range cmd.MethodologyTags {
		methodTags[i] = id.String()
	}
	subjectTags := StringArray(cmd.SubjectTags)
	worldviewTags := StringArray(cmd.WorldviewTags)

	listing := MktListing{
		CreatorID:       cmd.CreatorID,
		PublisherID:     cmd.PublisherID,
		Title:           cmd.Title,
		Description:     cmd.Description,
		PriceCents:      cmd.PriceCents,
		MethodologyTags: methodTags,
		SubjectTags:     subjectTags,
		GradeMin:        cmd.GradeMin,
		GradeMax:        cmd.GradeMax,
		ContentType:     cmd.ContentType,
		WorldviewTags:   worldviewTags,
		PreviewURL:      cmd.PreviewURL,
		ThumbnailURL:    cmd.ThumbnailURL,
		Status:          "draft",
		Version:         1,
	}
	if err := r.db.WithContext(ctx).Create(&listing).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &listing, nil
}

func (r *PgListingRepository) Save(ctx context.Context, listing *domain.MarketplaceListing) error {
	updates := map[string]any{
		"status":     string(listing.State()),
		"version":    listing.Version(),
		"updated_at": time.Now().UTC(),
	}
	if listing.PublishedAt() != nil {
		updates["published_at"] = *listing.PublishedAt()
	}
	if listing.ArchivedAt() != nil {
		updates["archived_at"] = *listing.ArchivedAt()
	}
	err := r.db.WithContext(ctx).Model(&MktListing{}).Where("id = ?", listing.ID()).Updates(updates).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgListingRepository) CreateVersionSnapshot(ctx context.Context, listingID uuid.UUID, version int32, title, description string, priceCents int32, changeSummary *string) error {
	snapshot := MktListingVersion{
		ListingID:     listingID,
		Version:       version,
		Title:         title,
		Description:   description,
		PriceCents:    priceCents,
		ChangeSummary: changeSummary,
	}
	if err := r.db.WithContext(ctx).Create(&snapshot).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgListingRepository) GetByID(ctx context.Context, listingID uuid.UUID) (*MktListing, error) {
	var listing MktListing
	if err := r.db.WithContext(ctx).Where("id = ?", listingID).First(&listing).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrListingNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &listing, nil
}

func (r *PgListingRepository) Browse(ctx context.Context, params *BrowseListingsParams) ([]ListingBrowseRow, int64, error) {
	// Build dynamic query for faceted search [S§9.3]
	query := r.db.WithContext(ctx).
		Table("mkt_listings l").
		Select(`l.id, l.title, l.description, l.price_cents, l.content_type,
				l.thumbnail_url, l.rating_avg, l.rating_count,
				p.name as publisher_name, c.store_name as creator_store_name`).
		Joins("JOIN mkt_publishers p ON p.id = l.publisher_id").
		Joins("JOIN mkt_creators c ON c.id = l.creator_id").
		Where("l.status = 'published'")

	if params.Q != nil && *params.Q != "" {
		query = query.Where("l.search_vector @@ websearch_to_tsquery('english', ?)", *params.Q)
	}
	if len(params.MethodologyIDs) > 0 {
		ids := make([]string, len(params.MethodologyIDs))
		for i, id := range params.MethodologyIDs {
			ids[i] = id.String()
		}
		query = query.Where("l.methodology_tags && ?::uuid[]", "{"+strings.Join(ids, ",")+"}")
	}
	if len(params.SubjectSlugs) > 0 {
		query = query.Where("l.subject_tags && ?::text[]", "{"+strings.Join(params.SubjectSlugs, ",")+"}")
	}
	if params.GradeMin != nil {
		query = query.Where("l.grade_max >= ?", *params.GradeMin)
	}
	if params.GradeMax != nil {
		query = query.Where("l.grade_min <= ?", *params.GradeMax)
	}
	if params.PriceMin != nil {
		query = query.Where("l.price_cents >= ?", *params.PriceMin)
	}
	if params.PriceMax != nil {
		query = query.Where("l.price_cents <= ?", *params.PriceMax)
	}
	if params.ContentType != nil {
		query = query.Where("l.content_type = ?", *params.ContentType)
	}
	if len(params.WorldviewTags) > 0 {
		query = query.Where("l.worldview_tags && ?::text[]", "{"+strings.Join(params.WorldviewTags, ",")+"}")
	}
	if params.MinRating != nil {
		query = query.Where("l.rating_avg >= ?", *params.MinRating)
	}

	// Count total before pagination
	var total int64
	countQuery := *query
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, shared.ErrDatabase(err)
	}

	// Sort
	sortBy := ListingSortByNewest
	if params.SortBy != nil {
		sortBy = *params.SortBy
	}
	switch sortBy {
	case ListingSortByPriceAsc:
		query = query.Order("l.price_cents ASC")
	case ListingSortByPriceDesc:
		query = query.Order("l.price_cents DESC")
	case ListingSortByRating:
		query = query.Order("l.rating_avg DESC")
	case ListingSortByRelevance:
		if params.Q != nil && *params.Q != "" {
			query = query.Order(gorm.Expr("ts_rank(l.search_vector, websearch_to_tsquery('english', ?)) DESC", *params.Q))
		} else {
			query = query.Order("l.published_at DESC")
		}
	default:
		query = query.Order("l.published_at DESC")
	}

	// Pagination
	limit := 20
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = int(*params.Limit)
	}
	query = query.Limit(limit)

	var rows []ListingBrowseRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, shared.ErrDatabase(err)
	}
	return rows, total, nil
}

func (r *PgListingRepository) Autocomplete(ctx context.Context, query string, limit uint8) ([]AutocompleteRow, error) {
	if limit == 0 || limit > 10 {
		limit = 10
	}
	var rows []AutocompleteRow
	err := r.db.WithContext(ctx).
		Table("mkt_listings").
		Select("id as listing_id, title, similarity(title, ?) as similarity", query).
		Where("status = 'published' AND title % ?", query).
		Order("similarity DESC").
		Limit(int(limit)).
		Scan(&rows).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return rows, nil
}

func (r *PgListingRepository) GetByCreator(ctx context.Context, creatorID uuid.UUID, params *CreatorListingQueryParams) ([]MktListing, int64, error) {
	query := r.db.WithContext(ctx).Where("creator_id = ?", creatorID)
	if params.Status != nil {
		query = query.Where("status = ?", *params.Status)
	}

	var total int64
	if err := query.Model(&MktListing{}).Count(&total).Error; err != nil {
		return nil, 0, shared.ErrDatabase(err)
	}

	limit := 20
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = int(*params.Limit)
	}

	var listings []MktListing
	if err := query.Order("created_at DESC").Limit(limit).Find(&listings).Error; err != nil {
		return nil, 0, shared.ErrDatabase(err)
	}
	return listings, total, nil
}

func (r *PgListingRepository) GetVersions(ctx context.Context, listingID uuid.UUID) ([]MktListingVersion, error) {
	var versions []MktListingVersion
	if err := r.db.WithContext(ctx).Where("listing_id = ?", listingID).Order("version DESC").Find(&versions).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return versions, nil
}

func (r *PgListingRepository) CountFiles(ctx context.Context, listingID uuid.UUID) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&MktListingFile{}).Where("listing_id = ?", listingID).Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return count, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgListingFileRepository
// ═══════════════════════════════════════════════════════════════════════════════

type PgListingFileRepository struct{ db *gorm.DB }

func NewPgListingFileRepository(db *gorm.DB) ListingFileRepository {
	return &PgListingFileRepository{db: db}
}

func (r *PgListingFileRepository) Create(ctx context.Context, cmd CreateListingFile) (*MktListingFile, error) {
	file := MktListingFile{
		ListingID:     cmd.ListingID,
		FileName:      cmd.FileName,
		FileSizeBytes: cmd.FileSizeBytes,
		MimeType:      cmd.MimeType,
		StorageKey:    cmd.StorageKey,
		SortOrder:     cmd.SortOrder,
	}
	if err := r.db.WithContext(ctx).Create(&file).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &file, nil
}

func (r *PgListingFileRepository) GetByID(ctx context.Context, listingID, fileID uuid.UUID) (*MktListingFile, error) {
	var file MktListingFile
	if err := r.db.WithContext(ctx).Where("id = ? AND listing_id = ?", fileID, listingID).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &file, nil
}

func (r *PgListingFileRepository) FindByStorageKey(ctx context.Context, storageKey string) (*MktListingFile, error) {
	var file MktListingFile
	if err := r.db.WithContext(ctx).Where("storage_key = ?", storageKey).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFileNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &file, nil
}

func (r *PgListingFileRepository) ListByListing(ctx context.Context, listingID uuid.UUID) ([]MktListingFile, error) {
	var files []MktListingFile
	if err := r.db.WithContext(ctx).Where("listing_id = ?", listingID).Order("sort_order ASC").Find(&files).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return files, nil
}

func (r *PgListingFileRepository) Delete(ctx context.Context, fileID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", fileID).Delete(&MktListingFile{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgCartRepository
// ═══════════════════════════════════════════════════════════════════════════════

type PgCartRepository struct{ db *gorm.DB }

func NewPgCartRepository(db *gorm.DB) CartRepository {
	return &PgCartRepository{db: db}
}

func (r *PgCartRepository) AddItem(ctx context.Context, listingID, parentID uuid.UUID, scope shared.FamilyScope) error {
	item := MktCartItem{
		FamilyID:        scope.FamilyID(),
		ListingID:       listingID,
		AddedByParentID: parentID,
	}
	if err := r.db.WithContext(ctx).Create(&item).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return ErrAlreadyInCart
		}
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgCartRepository) RemoveItem(ctx context.Context, listingID uuid.UUID, scope shared.FamilyScope) error {
	result := r.db.WithContext(ctx).Where("family_id = ? AND listing_id = ?", scope.FamilyID(), listingID).Delete(&MktCartItem{})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotInCart
	}
	return nil
}

func (r *PgCartRepository) GetItems(ctx context.Context, scope shared.FamilyScope) ([]CartItemRow, error) {
	var rows []CartItemRow
	err := r.db.WithContext(ctx).
		Table("mkt_cart_items ci").
		Select("ci.listing_id, l.title, l.price_cents, l.thumbnail_url, ci.created_at").
		Joins("JOIN mkt_listings l ON l.id = ci.listing_id").
		Where("ci.family_id = ?", scope.FamilyID()).
		Order("ci.created_at ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return rows, nil
}

func (r *PgCartRepository) Clear(ctx context.Context, scope shared.FamilyScope) error {
	if err := r.db.WithContext(ctx).Where("family_id = ?", scope.FamilyID()).Delete(&MktCartItem{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgPurchaseRepository
// ═══════════════════════════════════════════════════════════════════════════════

type PgPurchaseRepository struct{ db *gorm.DB }

func NewPgPurchaseRepository(db *gorm.DB) PurchaseRepository {
	return &PgPurchaseRepository{db: db}
}

func (r *PgPurchaseRepository) Create(ctx context.Context, cmd CreatePurchase) (*MktPurchase, error) {
	purchase := MktPurchase{
		FamilyID:           cmd.FamilyID,
		ListingID:          cmd.ListingID,
		CreatorID:          cmd.CreatorID,
		PaymentID:          cmd.PaymentID,
		PaymentSessionID:   cmd.PaymentSessionID,
		AmountCents:        cmd.AmountCents,
		PlatformFeeCents:   cmd.PlatformFeeCents,
		CreatorPayoutCents: cmd.CreatorPayoutCents,
	}
	if err := r.db.WithContext(ctx).Create(&purchase).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return nil, ErrAlreadyPurchased
		}
		return nil, shared.ErrDatabase(err)
	}
	return &purchase, nil
}

func (r *PgPurchaseRepository) GetByFamilyAndListing(ctx context.Context, familyID, listingID uuid.UUID) (*MktPurchase, error) {
	var purchase MktPurchase
	err := r.db.WithContext(ctx).Where("family_id = ? AND listing_id = ?", familyID, listingID).First(&purchase).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &purchase, nil
}

func (r *PgPurchaseRepository) ListByFamily(ctx context.Context, scope shared.FamilyScope, params *PurchaseQueryParams) ([]PurchaseRow, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&MktPurchase{}).Where("family_id = ?", scope.FamilyID()).Count(&total).Error; err != nil {
		return nil, 0, shared.ErrDatabase(err)
	}

	limit := 20
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = int(*params.Limit)
	}

	var rows []PurchaseRow
	err := r.db.WithContext(ctx).
		Table("mkt_purchases p").
		Select("p.id, p.listing_id, l.title as listing_title, p.amount_cents, p.refunded_at, p.created_at").
		Joins("JOIN mkt_listings l ON l.id = p.listing_id").
		Where("p.family_id = ?", scope.FamilyID()).
		Order("p.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, shared.ErrDatabase(err)
	}
	return rows, total, nil
}

func (r *PgPurchaseRepository) GetByPaymentSessionID(ctx context.Context, sessionID string) (*MktPurchase, error) {
	var purchase MktPurchase
	err := r.db.WithContext(ctx).Where("payment_session_id = ?", sessionID).First(&purchase).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPurchaseNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &purchase, nil
}

func (r *PgPurchaseRepository) SetRefund(ctx context.Context, purchaseID uuid.UUID, refundID string, refundAmountCents int32) error {
	err := r.db.WithContext(ctx).Model(&MktPurchase{}).Where("id = ?", purchaseID).
		Updates(map[string]any{
			"refunded_at":        time.Now().UTC(),
			"refund_amount_cents": refundAmountCents,
			"refund_id":          refundID,
		}).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgPurchaseRepository) GetCreatorSales(ctx context.Context, creatorID uuid.UUID, from, to time.Time) ([]SalesRow, error) {
	query := r.db.WithContext(ctx).
		Table("mkt_purchases p").
		Select("p.id as purchase_id, p.listing_id, l.title as listing_title, p.amount_cents, p.creator_payout_cents, p.created_at").
		Joins("JOIN mkt_listings l ON l.id = p.listing_id").
		Where("p.creator_id = ?", creatorID)

	if !from.IsZero() {
		query = query.Where("p.created_at >= ?", from)
	}
	query = query.Where("p.created_at <= ?", to).Order("p.created_at DESC")

	var rows []SalesRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return rows, nil
}

func (r *PgPurchaseRepository) GetAllCreatorSales(ctx context.Context, from, to time.Time) ([]CreatorSalesAggregate, error) {
	var rows []CreatorSalesAggregate
	query := r.db.WithContext(ctx).
		Table("mkt_purchases").
		Select(`creator_id,
			SUM(creator_payout_cents) as total_payout_cents,
			COUNT(*)::int as purchase_count,
			COALESCE(SUM(CASE WHEN refund_amount_cents > 0 THEN refund_amount_cents ELSE 0 END), 0) as refund_deduction_cents`).
		Where("created_at >= ? AND created_at <= ?", from, to).
		Group("creator_id").
		Having("SUM(creator_payout_cents) > 0")

	if err := query.Scan(&rows).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return rows, nil
}

func (r *PgPurchaseRepository) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).Delete(&MktPurchase{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgReviewRepository
// ═══════════════════════════════════════════════════════════════════════════════

type PgReviewRepository struct{ db *gorm.DB }

func NewPgReviewRepository(db *gorm.DB) ReviewRepository {
	return &PgReviewRepository{db: db}
}

func (r *PgReviewRepository) Create(ctx context.Context, cmd CreateReview) (*MktReview, error) {
	review := MktReview{
		ListingID:        cmd.ListingID,
		PurchaseID:       cmd.PurchaseID,
		FamilyID:         cmd.FamilyID,
		Rating:           cmd.Rating,
		ReviewText:       cmd.ReviewText,
		IsAnonymous:      cmd.IsAnonymous,
		ModerationStatus: "pending",
	}
	if err := r.db.WithContext(ctx).Create(&review).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "23505") {
			return nil, ErrAlreadyReviewed
		}
		return nil, shared.ErrDatabase(err)
	}
	return &review, nil
}

func (r *PgReviewRepository) ExistsByFamilyAndListing(ctx context.Context, familyID, listingID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&MktReview{}).
		Where("family_id = ? AND listing_id = ?", familyID, listingID).
		Count(&count).Error; err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count > 0, nil
}

func (r *PgReviewRepository) GetByID(ctx context.Context, reviewID uuid.UUID) (*MktReview, error) {
	var review MktReview
	if err := r.db.WithContext(ctx).Where("id = ?", reviewID).First(&review).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrReviewNotFound
		}
		return nil, shared.ErrDatabase(err)
	}
	return &review, nil
}

func (r *PgReviewRepository) Update(ctx context.Context, reviewID uuid.UUID, cmd UpdateReview) (*MktReview, error) {
	updates := map[string]any{"updated_at": time.Now().UTC()}
	if cmd.Rating != nil {
		updates["rating"] = *cmd.Rating
	}
	if cmd.ReviewText != nil {
		updates["review_text"] = *cmd.ReviewText
	}
	if cmd.IsAnonymous != nil {
		updates["is_anonymous"] = *cmd.IsAnonymous
	}
	if err := r.db.WithContext(ctx).Model(&MktReview{}).Where("id = ?", reviewID).Updates(updates).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return r.GetByID(ctx, reviewID)
}

func (r *PgReviewRepository) Delete(ctx context.Context, reviewID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("id = ?", reviewID).Delete(&MktReview{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgReviewRepository) SetCreatorResponse(ctx context.Context, reviewID uuid.UUID, responseText string) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Model(&MktReview{}).Where("id = ?", reviewID).
		Updates(map[string]any{
			"creator_response":    responseText,
			"creator_response_at": now,
			"updated_at":          now,
		}).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgReviewRepository) ListByListing(ctx context.Context, listingID uuid.UUID, params *ReviewQueryParams) ([]ReviewRow, int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&MktReview{}).
		Where("listing_id = ? AND moderation_status = 'approved'", listingID).Count(&total).Error; err != nil {
		return nil, 0, shared.ErrDatabase(err)
	}

	query := r.db.WithContext(ctx).
		Table("mkt_reviews r").
		Select("r.id, r.listing_id, r.rating, r.review_text, r.is_anonymous, r.creator_response, r.creator_response_at, r.moderation_status, r.created_at").
		Where("r.listing_id = ? AND r.moderation_status = 'approved'", listingID)

	sortBy := ReviewSortByNewest
	if params.SortBy != nil {
		sortBy = *params.SortBy
	}
	switch sortBy {
	case ReviewSortByOldest:
		query = query.Order("r.created_at ASC")
	case ReviewSortByRating:
		query = query.Order("r.rating DESC, r.created_at DESC")
	default:
		query = query.Order("r.created_at DESC")
	}

	limit := 20
	if params.Limit != nil && *params.Limit > 0 && *params.Limit <= 100 {
		limit = int(*params.Limit)
	}
	query = query.Limit(limit)

	var rows []ReviewRow
	if err := query.Scan(&rows).Error; err != nil {
		return nil, 0, shared.ErrDatabase(err)
	}
	return rows, total, nil
}

func (r *PgReviewRepository) GetAggregateRating(ctx context.Context, listingID uuid.UUID) (float64, int32, error) {
	type result struct {
		Avg   float64
		Count int32
	}
	var res result
	err := r.db.WithContext(ctx).
		Table("mkt_reviews").
		Select("COALESCE(AVG(rating), 0) as avg, COUNT(*) as count").
		Where("listing_id = ? AND moderation_status = 'approved'", listingID).
		Scan(&res).Error
	if err != nil {
		return 0, 0, shared.ErrDatabase(err)
	}
	return res.Avg, res.Count, nil
}

func (r *PgReviewRepository) UpdateListingRating(ctx context.Context, listingID uuid.UUID) error {
	avg, count, err := r.GetAggregateRating(ctx, listingID)
	if err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&MktListing{}).Where("id = ?", listingID).
		Updates(map[string]any{"rating_avg": avg, "rating_count": count}).Error
}

func (r *PgReviewRepository) SetModerationStatus(ctx context.Context, reviewID uuid.UUID, status string) error {
	err := r.db.WithContext(ctx).Model(&MktReview{}).Where("id = ?", reviewID).
		Updates(map[string]any{"moderation_status": status, "updated_at": time.Now().UTC()}).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgReviewRepository) AnonymizeByFamily(ctx context.Context, familyID uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&MktReview{}).Where("family_id = ?", familyID).
		Updates(map[string]any{
			"review_text": nil,
			"is_anonymous": true,
			"updated_at":   time.Now().UTC(),
		}).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PgCuratedSectionRepository
// ═══════════════════════════════════════════════════════════════════════════════

type PgCuratedSectionRepository struct{ db *gorm.DB }

func NewPgCuratedSectionRepository(db *gorm.DB) CuratedSectionRepository {
	return &PgCuratedSectionRepository{db: db}
}

func (r *PgCuratedSectionRepository) ListActive(ctx context.Context) ([]MktCuratedSection, error) {
	var sections []MktCuratedSection
	if err := r.db.WithContext(ctx).Where("is_active = true").Order("sort_order ASC").Find(&sections).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return sections, nil
}

func (r *PgCuratedSectionRepository) GetSectionItems(ctx context.Context, sectionID uuid.UUID, limit uint8) ([]ListingBrowseRow, error) {
	if limit == 0 {
		limit = 10
	}
	var rows []ListingBrowseRow
	err := r.db.WithContext(ctx).
		Table("mkt_curated_section_items csi").
		Select(`l.id, l.title, l.description, l.price_cents, l.content_type,
				l.thumbnail_url, l.rating_avg, l.rating_count,
				p.name as publisher_name, c.store_name as creator_store_name`).
		Joins("JOIN mkt_listings l ON l.id = csi.listing_id").
		Joins("JOIN mkt_publishers p ON p.id = l.publisher_id").
		Joins("JOIN mkt_creators c ON c.id = l.creator_id").
		Where("csi.section_id = ? AND l.status = 'published'", sectionID).
		Order("csi.sort_order ASC").
		Limit(int(limit)).
		Scan(&rows).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return rows, nil
}

func (r *PgCuratedSectionRepository) AddItem(ctx context.Context, sectionID, listingID uuid.UUID, sortOrder int16) error {
	item := MktCuratedSectionItem{SectionID: sectionID, ListingID: listingID, SortOrder: sortOrder}
	if err := r.db.WithContext(ctx).Create(&item).Error; err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return errors.New("item already in section")
		}
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgCuratedSectionRepository) RemoveItem(ctx context.Context, sectionID, listingID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("section_id = ? AND listing_id = ?", sectionID, listingID).
		Delete(&MktCuratedSectionItem{})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("item not in section")
	}
	return nil
}

func (r *PgCuratedSectionRepository) RefreshAutoSection(ctx context.Context, sectionSlug string) error {
	var section MktCuratedSection
	if err := r.db.WithContext(ctx).Where("slug = ? AND section_type = 'auto'", sectionSlug).First(&section).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("auto section %q not found", sectionSlug)
		}
		return shared.ErrDatabase(err)
	}

	// Clear existing items for this auto section
	if err := r.db.WithContext(ctx).Where("section_id = ?", section.ID).Delete(&MktCuratedSectionItem{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}

	// Repopulate based on slug strategy
	switch sectionSlug {
	case "trending":
		if err := r.db.WithContext(ctx).Exec(`
			INSERT INTO mkt_curated_section_items (section_id, listing_id, sort_order)
			SELECT ?, p.listing_id, ROW_NUMBER() OVER (ORDER BY COUNT(*) DESC)
			FROM mkt_purchases p
			JOIN mkt_listings l ON l.id = p.listing_id
			WHERE p.created_at > NOW() - INTERVAL '7 days'
			  AND l.status = 'published'
			GROUP BY p.listing_id
			ORDER BY COUNT(*) DESC
			LIMIT 20`, section.ID).Error; err != nil {
			return shared.ErrDatabase(err)
		}
	case "new-arrivals":
		if err := r.db.WithContext(ctx).Exec(`
			INSERT INTO mkt_curated_section_items (section_id, listing_id, sort_order)
			SELECT ?, id, ROW_NUMBER() OVER (ORDER BY published_at DESC)
			FROM mkt_listings
			WHERE status = 'published'
			ORDER BY published_at DESC
			LIMIT 20`, section.ID).Error; err != nil {
			return shared.ErrDatabase(err)
		}
	default:
		return fmt.Errorf("unknown auto section slug %q", sectionSlug)
	}

	return nil
}

// ─── Utility ────────────────────────────────────────────────────────────────

// truncateDescription truncates a description to ~200 chars for browse responses.
func truncateDescription(desc string, maxLen int) string {
	if len(desc) <= maxLen {
		return desc
	}
	// Find last space before maxLen
	truncated := desc[:maxLen]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxLen/2 {
		return truncated[:lastSpace] + "…"
	}
	return truncated + "…"
}

// Ensure fmt is used (for potential future use in error formatting).
var _ = fmt.Sprintf
