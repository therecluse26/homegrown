package mkt

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/mkt/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// creatorSharePercent is the default creator revenue share. [07-mkt §11]
const creatorSharePercent = 75

// marketplaceServiceImpl implements MarketplaceService.
type marketplaceServiceImpl struct {
	creators        CreatorRepository
	publishers      PublisherRepository
	listings        ListingRepository
	listingFiles    ListingFileRepository
	cart            CartRepository
	purchases       PurchaseRepository
	reviews         ReviewRepository
	curatedSections CuratedSectionRepository
	payment         PaymentAdapter
	media           MediaAdapter
	events          *shared.EventBus
	db              *gorm.DB
}

// NewMarketplaceService creates a new MarketplaceService.
// Constructor returns the interface type per [CODING §2.1].
func NewMarketplaceService(
	creators CreatorRepository,
	publishers PublisherRepository,
	listings ListingRepository,
	listingFiles ListingFileRepository,
	cart CartRepository,
	purchases PurchaseRepository,
	reviews ReviewRepository,
	curatedSections CuratedSectionRepository,
	payment PaymentAdapter,
	media MediaAdapter,
	events *shared.EventBus,
	db *gorm.DB,
) MarketplaceService {
	return &marketplaceServiceImpl{
		creators:        creators,
		publishers:      publishers,
		listings:        listings,
		listingFiles:    listingFiles,
		cart:            cart,
		purchases:       purchases,
		reviews:         reviews,
		curatedSections: curatedSections,
		payment:         payment,
		media:           media,
		events:          events,
		db:              db,
	}
}

// Compile-time interface check.
var _ MarketplaceService = (*marketplaceServiceImpl)(nil)

// ═══════════════════════════════════════════════════════════════════════════════
// Creator Onboarding [07-mkt §4.1]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *marketplaceServiceImpl) RegisterCreator(ctx context.Context, cmd RegisterCreatorCommand, auth *shared.AuthContext) (uuid.UUID, error) {
	if !cmd.TOSAccepted {
		return uuid.Nil, shared.ErrValidation("Terms of Service must be accepted")
	}

	// Check for existing creator
	existing, err := s.creators.GetByParentID(ctx, auth.ParentID)
	if err != nil {
		return uuid.Nil, err
	}
	if existing != nil {
		return uuid.Nil, shared.ErrConflict(ErrCreatorAlreadyExists.Error())
	}

	now := time.Now().UTC()
	creator, err := s.creators.Create(ctx, CreateCreator{
		ParentID:      auth.ParentID,
		StoreName:     cmd.StoreName,
		StoreBio:      cmd.StoreBio,
		StoreLogoURL:  cmd.StoreLogoURL,
		TOSAcceptedAt: &now,
	})
	if err != nil {
		return uuid.Nil, err
	}

	_ = s.events.Publish(ctx, CreatorOnboarded{
		CreatorID: creator.ID,
		ParentID:  auth.ParentID,
		StoreName: cmd.StoreName,
	})

	return creator.ID, nil
}

func (s *marketplaceServiceImpl) UpdateCreatorProfile(ctx context.Context, cmd UpdateCreatorProfileCommand, creatorID uuid.UUID) error {
	_, err := s.creators.Update(ctx, creatorID, UpdateCreator(cmd))
	return err
}

func (s *marketplaceServiceImpl) CreateOnboardingLink(ctx context.Context, creatorID uuid.UUID) (string, error) {
	creator, err := s.creators.GetByID(ctx, creatorID)
	if err != nil {
		return "", err
	}

	if creator.OnboardingStatus == "active" {
		return "", shared.ErrBadRequest("creator is already active")
	}

	// Create sub-merchant if no payment account yet
	paymentAccountID := ""
	if creator.PaymentAccountID != nil {
		paymentAccountID = *creator.PaymentAccountID
	} else {
		accountID, createErr := s.payment.CreateSubMerchant(ctx, SubMerchantConfig{
			CreatorID: creatorID,
			StoreName: creator.StoreName,
		})
		if createErr != nil {
			return "", createErr
		}
		if setErr := s.creators.SetPaymentAccountID(ctx, creatorID, accountID); setErr != nil {
			return "", setErr
		}
		paymentAccountID = accountID
	}

	link, err := s.payment.CreateOnboardingLink(ctx, paymentAccountID, "/marketplace/creators/me")
	if err != nil {
		return "", err
	}

	// Transition to onboarding if currently pending
	if creator.OnboardingStatus == "pending" {
		if setErr := s.creators.SetOnboardingStatus(ctx, creatorID, "onboarding"); setErr != nil {
			slog.Error("failed to update onboarding status", "error", setErr)
		}
	}

	return link, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Publisher Management [07-mkt §4.1]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *marketplaceServiceImpl) CreatePublisher(ctx context.Context, cmd CreatePublisherCommand, creatorID uuid.UUID) (uuid.UUID, error) {
	slug := ""
	if cmd.Slug != nil {
		slug = *cmd.Slug
	} else {
		slug = generateSlug(cmd.Name)
	}

	// Check slug uniqueness
	existing, err := s.publishers.GetBySlug(ctx, slug)
	if err != nil {
		return uuid.Nil, err
	}
	if existing != nil {
		return uuid.Nil, shared.ErrConflict(ErrPublisherSlugConflict.Error())
	}

	publisher, err := s.publishers.Create(ctx, CreatePublisher{
		Name:        cmd.Name,
		Slug:        slug,
		Description: cmd.Description,
		LogoURL:     cmd.LogoURL,
		WebsiteURL:  cmd.WebsiteURL,
	})
	if err != nil {
		return uuid.Nil, err
	}

	// Add creator as owner
	if addErr := s.publishers.AddMember(ctx, publisher.ID, creatorID, "owner"); addErr != nil {
		return uuid.Nil, addErr
	}

	return publisher.ID, nil
}

func (s *marketplaceServiceImpl) UpdatePublisher(ctx context.Context, cmd UpdatePublisherCommand, publisherID, creatorID uuid.UUID) error {
	pub, err := s.publishers.GetByID(ctx, publisherID)
	if err != nil {
		return err
	}
	if pub.IsPlatform {
		return shared.ErrForbidden()
	}

	// Check role — only owner/admin can update
	role, err := s.publishers.GetMemberRole(ctx, publisherID, creatorID)
	if err != nil {
		if errors.Is(err, ErrNotPublisherMember) {
			return shared.ErrForbidden()
		}
		return err
	}
	if *role != "owner" && *role != "admin" {
		return shared.ErrForbidden()
	}

	_, err = s.publishers.Update(ctx, publisherID, UpdatePublisher(cmd))
	return err
}

func (s *marketplaceServiceImpl) AddPublisherMember(ctx context.Context, publisherID uuid.UUID, cmd AddPublisherMemberCommand, actingCreatorID uuid.UUID) error {
	pub, err := s.publishers.GetByID(ctx, publisherID)
	if err != nil {
		return err
	}
	if pub.IsPlatform {
		return shared.ErrForbidden()
	}

	// Only owner/admin can add members
	role, err := s.publishers.GetMemberRole(ctx, publisherID, actingCreatorID)
	if err != nil {
		if errors.Is(err, ErrNotPublisherMember) {
			return shared.ErrForbidden()
		}
		return err
	}
	if *role != "owner" && *role != "admin" {
		return shared.ErrForbidden()
	}

	return s.publishers.AddMember(ctx, publisherID, cmd.CreatorID, cmd.Role)
}

func (s *marketplaceServiceImpl) RemovePublisherMember(ctx context.Context, publisherID, memberCreatorID, actingCreatorID uuid.UUID) error {
	pub, err := s.publishers.GetByID(ctx, publisherID)
	if err != nil {
		return err
	}
	if pub.IsPlatform {
		return shared.ErrForbidden()
	}

	// Only owner can remove members
	role, err := s.publishers.GetMemberRole(ctx, publisherID, actingCreatorID)
	if err != nil {
		if errors.Is(err, ErrNotPublisherMember) {
			return shared.ErrForbidden()
		}
		return err
	}
	if role == nil || *role != "owner" {
		return shared.ErrForbidden()
	}

	// Cannot remove last owner
	memberRole, err := s.publishers.GetMemberRole(ctx, publisherID, memberCreatorID)
	if err != nil {
		return err
	}
	if memberRole != nil && *memberRole == "owner" {
		count, countErr := s.publishers.CountOwners(ctx, publisherID)
		if countErr != nil {
			return countErr
		}
		if count <= 1 {
			return shared.ErrBadRequest(ErrCannotRemoveLastOwner.Error())
		}
	}

	return s.publishers.RemoveMember(ctx, publisherID, memberCreatorID)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Listing Lifecycle [07-mkt §4.1, §9]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *marketplaceServiceImpl) CreateListing(ctx context.Context, cmd CreateListingCommand, creatorID uuid.UUID) (uuid.UUID, error) {
	// Validate content type
	if !ValidContentTypes[cmd.ContentType] {
		return uuid.Nil, shared.ErrValidation("invalid content type")
	}

	// Creator must be member of the publisher
	_, err := s.publishers.GetMemberRole(ctx, cmd.PublisherID, creatorID)
	if err != nil {
		if errors.Is(err, ErrNotPublisherMember) {
			return uuid.Nil, shared.ErrForbidden()
		}
		return uuid.Nil, err
	}

	listing, err := s.listings.Create(ctx, CreateListing{
		CreatorID:       creatorID,
		PublisherID:     cmd.PublisherID,
		Title:           cmd.Title,
		Description:     cmd.Description,
		PriceCents:      cmd.PriceCents,
		MethodologyTags: cmd.MethodologyTags,
		SubjectTags:     cmd.SubjectTags,
		GradeMin:        cmd.GradeMin,
		GradeMax:        cmd.GradeMax,
		ContentType:     cmd.ContentType,
		WorldviewTags:   cmd.WorldviewTags,
		PreviewURL:      cmd.PreviewURL,
		ThumbnailURL:    cmd.ThumbnailURL,
	})
	if err != nil {
		return uuid.Nil, err
	}

	return listing.ID, nil
}

func (s *marketplaceServiceImpl) UpdateListing(ctx context.Context, cmd UpdateListingCommand, listingID, creatorID uuid.UUID) error {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing.CreatorID != creatorID {
		return shared.ErrForbidden()
	}

	fileCount, err := s.listings.CountFiles(ctx, listingID)
	if err != nil {
		return err
	}

	// Reconstruct aggregate
	agg := domain.FromPersistence(
		listing.ID, listing.CreatorID, listing.PublisherID,
		listing.Title, listing.Description, listing.PriceCents,
		uuidArrayToSlice(listing.MethodologyTags), []string(listing.SubjectTags),
		listing.GradeMin, listing.GradeMax, listing.ContentType,
		[]string(listing.WorldviewTags), listing.PreviewURL, listing.ThumbnailURL,
		domain.ListingState(listing.Status), listing.RatingAvg, listing.RatingCount,
		listing.Version, listing.PublishedAt, listing.ArchivedAt,
		fileCount, listing.CreatedAt, listing.UpdatedAt,
	)

	snapshot, err := agg.UpdatePublished(cmd.Title, cmd.Description, cmd.PriceCents)
	if err != nil {
		return mapDomainError(err)
	}

	// Save updated aggregate
	if saveErr := s.listings.Save(ctx, agg); saveErr != nil {
		return saveErr
	}

	// Also update remaining GORM fields not tracked by the aggregate
	updateFields := make(map[string]any)
	if cmd.MethodologyTags != nil {
		tags := make(UUIDArray, len(cmd.MethodologyTags))
		for i, t := range cmd.MethodologyTags {
			tags[i] = t.String()
		}
		updateFields["methodology_tags"] = tags
	}
	if cmd.SubjectTags != nil {
		updateFields["subject_tags"] = StringArray(cmd.SubjectTags)
	}
	if cmd.GradeMin != nil {
		updateFields["grade_min"] = cmd.GradeMin
	}
	if cmd.GradeMax != nil {
		updateFields["grade_max"] = cmd.GradeMax
	}
	if cmd.WorldviewTags != nil {
		updateFields["worldview_tags"] = StringArray(cmd.WorldviewTags)
	}
	if cmd.PreviewURL != nil {
		updateFields["preview_url"] = cmd.PreviewURL
	}
	if cmd.ThumbnailURL != nil {
		updateFields["thumbnail_url"] = cmd.ThumbnailURL
	}

	if len(updateFields) > 0 {
		if dbErr := s.db.Model(&MktListing{}).Where("id = ?", listingID).Updates(updateFields).Error; dbErr != nil {
			return shared.ErrDatabase(dbErr)
		}
	}

	// Create version snapshot for published listings [S§9.2.3]
	if snapshot != nil {
		if vsErr := s.listings.CreateVersionSnapshot(ctx, listingID, snapshot.Version, snapshot.Title, snapshot.Description, snapshot.PriceCents, cmd.ChangeSummary); vsErr != nil {
			slog.Error("failed to create version snapshot", "error", vsErr, "listing_id", listingID)
		}
	}

	return nil
}

func (s *marketplaceServiceImpl) SubmitListing(ctx context.Context, listingID, creatorID uuid.UUID) error {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing.CreatorID != creatorID {
		return shared.ErrForbidden()
	}

	fileCount, err := s.listings.CountFiles(ctx, listingID)
	if err != nil {
		return err
	}

	agg := domain.FromPersistence(
		listing.ID, listing.CreatorID, listing.PublisherID,
		listing.Title, listing.Description, listing.PriceCents,
		uuidArrayToSlice(listing.MethodologyTags), []string(listing.SubjectTags),
		listing.GradeMin, listing.GradeMax, listing.ContentType,
		[]string(listing.WorldviewTags), listing.PreviewURL, listing.ThumbnailURL,
		domain.ListingState(listing.Status), listing.RatingAvg, listing.RatingCount,
		listing.Version, listing.PublishedAt, listing.ArchivedAt,
		fileCount, listing.CreatedAt, listing.UpdatedAt,
	)

	evt, err := agg.Submit()
	if err != nil {
		return mapDomainError(err)
	}

	if saveErr := s.listings.Save(ctx, agg); saveErr != nil {
		return saveErr
	}

	_ = s.events.Publish(ctx, ListingSubmitted{
		ListingID: evt.ListingID,
		CreatorID: evt.CreatorID,
	})

	return nil
}

func (s *marketplaceServiceImpl) PublishListing(ctx context.Context, listingID, creatorID uuid.UUID) error {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing.CreatorID != creatorID {
		return shared.ErrForbidden()
	}

	fileCount, err := s.listings.CountFiles(ctx, listingID)
	if err != nil {
		return err
	}

	agg := domain.FromPersistence(
		listing.ID, listing.CreatorID, listing.PublisherID,
		listing.Title, listing.Description, listing.PriceCents,
		uuidArrayToSlice(listing.MethodologyTags), []string(listing.SubjectTags),
		listing.GradeMin, listing.GradeMax, listing.ContentType,
		[]string(listing.WorldviewTags), listing.PreviewURL, listing.ThumbnailURL,
		domain.ListingState(listing.Status), listing.RatingAvg, listing.RatingCount,
		listing.Version, listing.PublishedAt, listing.ArchivedAt,
		fileCount, listing.CreatedAt, listing.UpdatedAt,
	)

	evt, err := agg.Publish()
	if err != nil {
		return mapDomainError(err)
	}

	if saveErr := s.listings.Save(ctx, agg); saveErr != nil {
		return saveErr
	}

	_ = s.events.Publish(ctx, ListingPublished{
		ListingID:   evt.ListingID,
		PublisherID: evt.PublisherID,
		ContentType: evt.ContentType,
		SubjectTags: evt.SubjectTags,
	})

	return nil
}

func (s *marketplaceServiceImpl) ArchiveListing(ctx context.Context, listingID, creatorID uuid.UUID) error {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing.CreatorID != creatorID {
		return shared.ErrForbidden()
	}

	fileCount, err := s.listings.CountFiles(ctx, listingID)
	if err != nil {
		return err
	}

	agg := domain.FromPersistence(
		listing.ID, listing.CreatorID, listing.PublisherID,
		listing.Title, listing.Description, listing.PriceCents,
		uuidArrayToSlice(listing.MethodologyTags), []string(listing.SubjectTags),
		listing.GradeMin, listing.GradeMax, listing.ContentType,
		[]string(listing.WorldviewTags), listing.PreviewURL, listing.ThumbnailURL,
		domain.ListingState(listing.Status), listing.RatingAvg, listing.RatingCount,
		listing.Version, listing.PublishedAt, listing.ArchivedAt,
		fileCount, listing.CreatedAt, listing.UpdatedAt,
	)

	evt, err := agg.Archive()
	if err != nil {
		return mapDomainError(err)
	}

	if saveErr := s.listings.Save(ctx, agg); saveErr != nil {
		return saveErr
	}

	_ = s.events.Publish(ctx, ListingArchived{ListingID: evt.ListingID})

	return nil
}

func (s *marketplaceServiceImpl) UploadListingFile(ctx context.Context, cmd UploadListingFileCommand, listingID, creatorID uuid.UUID) (uuid.UUID, error) {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return uuid.Nil, err
	}
	if listing.CreatorID != creatorID {
		return uuid.Nil, shared.ErrForbidden()
	}
	if listing.Status != "draft" && listing.Status != "published" {
		return uuid.Nil, shared.ErrBadRequest("files can only be added to draft or published listings")
	}

	// Generate storage key
	storageKey := fmt.Sprintf("mkt/listings/%s/%s", listingID.String(), cmd.FileName)

	// Get presigned upload URL
	_, uploadErr := s.media.PresignedUpload(ctx, storageKey, cmd.MimeType, uint64(cmd.FileSizeBytes))
	if uploadErr != nil {
		return uuid.Nil, uploadErr
	}

	// Determine sort order
	existingFiles, err := s.listingFiles.ListByListing(ctx, listingID)
	if err != nil {
		return uuid.Nil, err
	}
	sortOrder := int16(len(existingFiles))

	file, err := s.listingFiles.Create(ctx, CreateListingFile{
		ListingID:     listingID,
		FileName:      cmd.FileName,
		FileSizeBytes: cmd.FileSizeBytes,
		MimeType:      cmd.MimeType,
		StorageKey:    storageKey,
		SortOrder:     sortOrder,
	})
	if err != nil {
		return uuid.Nil, err
	}

	return file.ID, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cart & Checkout [07-mkt §4.1, §11]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *marketplaceServiceImpl) AddToCart(ctx context.Context, listingID uuid.UUID, scope shared.FamilyScope, parentID uuid.UUID) error {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return err
	}
	if listing.Status != "published" {
		return shared.ErrNotFound()
	}

	// Check not already purchased
	_, err = s.purchases.GetByFamilyAndListing(ctx, scope.FamilyID(), listingID)
	if err == nil {
		return shared.ErrConflict(ErrAlreadyPurchased.Error())
	}
	if !errors.Is(err, ErrPurchaseNotFound) {
		return err
	}

	return s.cart.AddItem(ctx, listingID, parentID, scope)
}

func (s *marketplaceServiceImpl) RemoveFromCart(ctx context.Context, listingID uuid.UUID, scope shared.FamilyScope) error {
	return s.cart.RemoveItem(ctx, listingID, scope)
}

func (s *marketplaceServiceImpl) CreateCheckout(ctx context.Context, scope shared.FamilyScope) (*CheckoutSessionResponse, error) {
	items, err := s.cart.GetItems(ctx, scope)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, shared.ErrBadRequest(ErrEmptyCart.Error())
	}

	// Build payment line items and split rules
	lineItems := make([]PaymentLineItem, 0, len(items))
	splitRules := make([]SplitRule, 0, len(items))

	for _, item := range items {
		// Verify each listing is still published
		listing, listingErr := s.listings.GetByID(ctx, item.ListingID)
		if listingErr != nil {
			return nil, listingErr
		}
		if listing == nil || listing.Status != "published" {
			return nil, shared.ErrConflict(ErrStaleCart.Error())
		}

		lineItems = append(lineItems, PaymentLineItem{
			ListingID:   item.ListingID,
			AmountCents: int64(item.PriceCents),
			Description: item.Title,
		})

		// Calculate split [07-mkt §11]
		creatorPayout, _ := calculateSplit(int64(item.PriceCents), creatorSharePercent)

		// Get creator's payment account
		creator, creatorErr := s.creators.GetByID(ctx, listing.CreatorID)
		if creatorErr != nil {
			return nil, creatorErr
		}
		if creator != nil && creator.PaymentAccountID != nil {
			splitRules = append(splitRules, SplitRule{
				RecipientAccountID: *creator.PaymentAccountID,
				AmountCents:        creatorPayout,
			})
		}
	}

	metadata := map[string]string{
		"family_id": scope.FamilyID().String(),
	}

	session, err := s.payment.CreatePayment(ctx, lineItems, splitRules, "/marketplace/purchases", metadata)
	if err != nil {
		return nil, err
	}

	return &CheckoutSessionResponse{
		CheckoutURL:      session.CheckoutURL,
		PaymentSessionID: session.PaymentSessionID,
	}, nil
}

func (s *marketplaceServiceImpl) HandlePaymentWebhook(ctx context.Context, payload []byte, signature string) error {
	valid, err := s.payment.VerifyWebhook(ctx, payload, signature)
	if err != nil || !valid {
		return shared.ErrUnauthorized()
	}

	event, err := s.payment.ParseEvent(ctx, payload)
	if err != nil {
		return shared.ErrBadRequest("malformed webhook payload")
	}

	switch event.Type {
	case "payment_succeeded":
		return s.handlePaymentSucceeded(ctx, event)
	case "refund_succeeded":
		return s.handleRefundSucceeded(ctx, event)
	default:
		slog.Info("unhandled payment event type", "type", event.Type)
		return nil
	}
}

func (s *marketplaceServiceImpl) handlePaymentSucceeded(ctx context.Context, event *PaymentEvent) error {
	// Idempotency check [07-mkt §11]
	_, err := s.purchases.GetByPaymentSessionID(ctx, event.PaymentID)
	if err == nil {
		return nil // Already processed
	}
	if !errors.Is(err, ErrPurchaseNotFound) {
		return err
	}

	familyIDStr, ok := event.Metadata["family_id"]
	if !ok {
		slog.Error("payment webhook missing family_id metadata", "payment_id", event.PaymentID)
		return nil
	}
	familyID, parseErr := uuid.Parse(familyIDStr)
	if parseErr != nil {
		slog.Error("invalid family_id in webhook metadata", "payment_id", event.PaymentID)
		return nil
	}

	// Get cart items for this family (requires RLS bypass since webhook has no auth context)
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		// Fetch cart items via raw query (bypass RLS)
		var cartRows []struct {
			ListingID uuid.UUID
			Title     string
		}
		if dbErr := tx.Table("mkt_cart_items ci").
			Select("ci.listing_id, l.title").
			Joins("JOIN mkt_listings l ON l.id = ci.listing_id").
			Where("ci.family_id = ?", familyID).
			Scan(&cartRows).Error; dbErr != nil {
			return shared.ErrDatabase(dbErr)
		}

		for _, row := range cartRows {
			listing, listingErr := s.listings.GetByID(ctx, row.ListingID)
			if listingErr != nil {
				continue
			}

			creatorPayout, platformFee := calculateSplit(int64(listing.PriceCents), creatorSharePercent)
			paymentID := event.PaymentID

			purchase, createErr := s.purchases.Create(ctx, CreatePurchase{
				FamilyID:           familyID,
				ListingID:          row.ListingID,
				CreatorID:          listing.CreatorID,
				PaymentID:          &paymentID,
				PaymentSessionID:   &event.PaymentID,
				AmountCents:        listing.PriceCents,
				PlatformFeeCents:   int32(platformFee),
				CreatorPayoutCents: int32(creatorPayout),
			})
			if createErr != nil {
				slog.Error("failed to create purchase", "error", createErr, "listing_id", row.ListingID)
				continue
			}

			_ = s.events.Publish(ctx, PurchaseCompleted{
				FamilyID:   familyID,
				PurchaseID: purchase.ID,
				ListingID:  row.ListingID,
				ContentMetadata: PurchaseMetadata{
					ContentType: listing.ContentType,
					ContentIDs:  []uuid.UUID{row.ListingID},
					PublisherID: listing.PublisherID,
				},
			})
		}

		// Clear cart
		if dbErr := tx.Where("family_id = ?", familyID).Delete(&MktCartItem{}).Error; dbErr != nil {
			slog.Error("failed to clear cart after purchase", "error", dbErr, "family_id", familyID)
		}

		return nil
	})
}

func (s *marketplaceServiceImpl) handleRefundSucceeded(ctx context.Context, event *PaymentEvent) error {
	purchase, err := s.purchases.GetByPaymentSessionID(ctx, event.PaymentID)
	if err != nil {
		if errors.Is(err, ErrPurchaseNotFound) {
			slog.Warn("refund webhook for unknown purchase", "payment_id", event.PaymentID)
			return nil
		}
		return err
	}

	if setErr := s.purchases.SetRefund(ctx, purchase.ID, event.RefundID, int32(event.AmountCents)); setErr != nil {
		return setErr
	}

	_ = s.events.Publish(ctx, PurchaseRefunded{
		PurchaseID:        purchase.ID,
		ListingID:         purchase.ListingID,
		FamilyID:          purchase.FamilyID,
		RefundAmountCents: event.AmountCents,
	})

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Reviews [07-mkt §14]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *marketplaceServiceImpl) CreateReview(ctx context.Context, cmd CreateReviewCommand, listingID uuid.UUID, scope shared.FamilyScope) (uuid.UUID, error) {
	// Verify purchase exists [S§9.5]
	purchase, err := s.purchases.GetByFamilyAndListing(ctx, scope.FamilyID(), listingID)
	if err != nil {
		if errors.Is(err, ErrPurchaseNotFound) {
			return uuid.Nil, shared.ErrForbidden()
		}
		return uuid.Nil, err
	}

	// Pre-check for existing review to return 409 instead of relying on DB constraint. [H5]
	exists, err := s.reviews.ExistsByFamilyAndListing(ctx, scope.FamilyID(), listingID)
	if err != nil {
		return uuid.Nil, err
	}
	if exists {
		return uuid.Nil, ErrAlreadyReviewed
	}

	isAnonymous := true
	if cmd.IsAnonymous != nil {
		isAnonymous = *cmd.IsAnonymous
	}

	review, err := s.reviews.Create(ctx, CreateReview{
		ListingID:   listingID,
		PurchaseID:  purchase.ID,
		FamilyID:    scope.FamilyID(),
		Rating:      cmd.Rating,
		ReviewText:  cmd.ReviewText,
		IsAnonymous: isAnonymous,
	})
	if err != nil {
		return uuid.Nil, err
	}

	// Update aggregate rating on listing [§14]
	if ratingErr := s.reviews.UpdateListingRating(ctx, listingID); ratingErr != nil {
		slog.Error("failed to update listing rating", "error", ratingErr, "listing_id", listingID)
	}

	_ = s.events.Publish(ctx, ReviewCreated{
		ReviewID:   review.ID,
		ListingID:  listingID,
		Rating:     cmd.Rating,
		ReviewText: cmd.ReviewText,
	})

	// Send review text to safety:: for moderation
	if cmd.ReviewText != nil {
		_ = s.events.Publish(ctx, ContentSubmittedForModeration{
			ContentID:   review.ID,
			ContentType: "marketplace_review",
			Text:        *cmd.ReviewText,
		})
	}

	return review.ID, nil
}

func (s *marketplaceServiceImpl) UpdateReview(ctx context.Context, cmd UpdateReviewCommand, reviewID uuid.UUID, scope shared.FamilyScope) error {
	review, err := s.reviews.GetByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if review.FamilyID != scope.FamilyID() {
		return shared.ErrForbidden()
	}

	_, err = s.reviews.Update(ctx, reviewID, UpdateReview(cmd))
	if err != nil {
		return err
	}

	// Update aggregate rating
	if ratingErr := s.reviews.UpdateListingRating(ctx, review.ListingID); ratingErr != nil {
		slog.Error("failed to update listing rating", "error", ratingErr)
	}

	return nil
}

func (s *marketplaceServiceImpl) DeleteReview(ctx context.Context, reviewID uuid.UUID, scope shared.FamilyScope) error {
	review, err := s.reviews.GetByID(ctx, reviewID)
	if err != nil {
		return err
	}
	if review.FamilyID != scope.FamilyID() {
		return shared.ErrForbidden()
	}

	if delErr := s.reviews.Delete(ctx, reviewID); delErr != nil {
		return delErr
	}

	// Update aggregate rating
	if ratingErr := s.reviews.UpdateListingRating(ctx, review.ListingID); ratingErr != nil {
		slog.Error("failed to update listing rating", "error", ratingErr)
	}

	return nil
}

func (s *marketplaceServiceImpl) RespondToReview(ctx context.Context, cmd RespondToReviewCommand, reviewID, creatorID uuid.UUID) error {
	review, err := s.reviews.GetByID(ctx, reviewID)
	if err != nil {
		return err
	}

	// Verify creator owns the listing
	listing, err := s.listings.GetByID(ctx, review.ListingID)
	if err != nil {
		return err
	}
	if listing.CreatorID != creatorID {
		return shared.ErrForbidden()
	}

	return s.reviews.SetCreatorResponse(ctx, reviewID, cmd.ResponseText)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Free Content [07-mkt §11]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *marketplaceServiceImpl) GetFreeListing(ctx context.Context, listingID uuid.UUID, scope shared.FamilyScope) (uuid.UUID, error) {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return uuid.Nil, err
	}
	if listing.PriceCents != 0 {
		return uuid.Nil, shared.ErrBadRequest(ErrListingNotFree.Error())
	}
	if listing.Status != "published" {
		return uuid.Nil, shared.ErrBadRequest(ErrListingNotPublished.Error())
	}

	// Check not already purchased
	existing, err := s.purchases.GetByFamilyAndListing(ctx, scope.FamilyID(), listingID)
	if err != nil {
		return uuid.Nil, err
	}
	if existing != nil {
		return uuid.Nil, shared.ErrConflict(ErrAlreadyPurchased.Error())
	}

	purchase, err := s.purchases.Create(ctx, CreatePurchase{
		FamilyID:           scope.FamilyID(),
		ListingID:          listingID,
		CreatorID:          listing.CreatorID,
		AmountCents:        0,
		PlatformFeeCents:   0,
		CreatorPayoutCents: 0,
	})
	if err != nil {
		return uuid.Nil, err
	}

	_ = s.events.Publish(ctx, PurchaseCompleted{
		FamilyID:   scope.FamilyID(),
		PurchaseID: purchase.ID,
		ListingID:  listingID,
		ContentMetadata: PurchaseMetadata{
			ContentType: listing.ContentType,
			ContentIDs:  []uuid.UUID{listingID},
			PublisherID: listing.PublisherID,
		},
	})

	return purchase.ID, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Payouts (Phase 2) [07-mkt §15]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *marketplaceServiceImpl) RequestPayout(ctx context.Context, creatorID uuid.UUID) (*PayoutResult, error) {
	creator, err := s.creators.GetByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}
	if creator.OnboardingStatus != "active" {
		return nil, shared.ErrBadRequest(ErrCreatorNotActive.Error())
	}
	if creator.PaymentAccountID == nil {
		return nil, shared.ErrBadRequest(ErrCreatorNotActive.Error())
	}

	// Calculate unpaid earnings
	allSales, err := s.purchases.GetCreatorSales(ctx, creatorID, time.Time{}, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	var totalEarnings int64
	for _, sale := range allSales {
		totalEarnings += int64(sale.CreatorPayoutCents)
	}

	// Minimum payout threshold: $25.00
	if totalEarnings < 2500 {
		return nil, shared.ErrBadRequest(ErrPayoutThresholdNotMet.Error())
	}

	return s.payment.CreatePayout(ctx, *creator.PaymentAccountID, totalEarnings, "USD")
}

// ═══════════════════════════════════════════════════════════════════════════════
// Event Handlers (cross-domain reactions) [07-mkt §18.4]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *marketplaceServiceImpl) ArchiveListingByContentKey(ctx context.Context, contentKey, reason string) error {
	file, err := s.listingFiles.FindByStorageKey(ctx, contentKey)
	if err != nil {
		return err
	}
	if errors.Is(err, ErrFileNotFound) {
		slog.Warn("mkt: no listing file found for content key", "content_key", contentKey)
		return nil
	}
	return s.HandleContentFlagged(ctx, file.ListingID, reason)
}

func (s *marketplaceServiceImpl) HandleContentFlagged(ctx context.Context, listingID uuid.UUID, reason string) error {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		if errors.Is(err, ErrListingNotFound) {
			return nil
		}
		return err
	}

	// Archive flagged listings
	if listing.Status == "published" {
		fileCount, countErr := s.listings.CountFiles(ctx, listingID)
		if countErr != nil {
			return countErr
		}

		agg := domain.FromPersistence(
			listing.ID, listing.CreatorID, listing.PublisherID,
			listing.Title, listing.Description, listing.PriceCents,
			uuidArrayToSlice(listing.MethodologyTags), []string(listing.SubjectTags),
			listing.GradeMin, listing.GradeMax, listing.ContentType,
			[]string(listing.WorldviewTags), listing.PreviewURL, listing.ThumbnailURL,
			domain.ListingState(listing.Status), listing.RatingAvg, listing.RatingCount,
			listing.Version, listing.PublishedAt, listing.ArchivedAt,
			fileCount, listing.CreatedAt, listing.UpdatedAt,
		)

		evt, archiveErr := agg.Archive()
		if archiveErr != nil {
			return archiveErr
		}

		if saveErr := s.listings.Save(ctx, agg); saveErr != nil {
			return saveErr
		}

		_ = s.events.Publish(ctx, ListingArchived{ListingID: evt.ListingID})
	}

	slog.Info("listing flagged", "listing_id", listingID, "reason", reason)
	return nil
}

func (s *marketplaceServiceImpl) HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error {
	// Anonymize reviews (retain ratings, clear text) [07-mkt §18.4]
	if err := s.reviews.AnonymizeByFamily(ctx, familyID); err != nil {
		slog.Error("failed to anonymize reviews for family deletion", "error", err, "family_id", familyID)
	}

	// Purchase records are intentionally retained (legal/tax requirement) [07-mkt §18.4]

	// Clear cart items — no auth context in event handlers, so bypass RLS [07-mkt §18.4]
	if err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		if dbErr := tx.Where("family_id = ?", familyID).Delete(&MktCartItem{}).Error; dbErr != nil {
			return shared.ErrDatabase(dbErr)
		}
		return nil
	}); err != nil {
		slog.Error("failed to clear cart for family deletion", "error", err, "family_id", familyID)
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Query Side [07-mkt §4.1, §4.2]
// ═══════════════════════════════════════════════════════════════════════════════

func (s *marketplaceServiceImpl) GetCreatorByParentID(ctx context.Context, parentID uuid.UUID) (*CreatorResponse, error) {
	creator, err := s.creators.GetByParentID(ctx, parentID)
	if err != nil {
		if errors.Is(err, ErrCreatorNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return toCreatorResponse(creator), nil
}

func (s *marketplaceServiceImpl) GetCreatorDashboard(ctx context.Context, creatorID uuid.UUID, period DashboardPeriod) (*CreatorDashboardResponse, error) {
	from, to := period.ToDateRange()

	allTimeSales, err := s.purchases.GetCreatorSales(ctx, creatorID, time.Time{}, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	periodSales, err := s.purchases.GetCreatorSales(ctx, creatorID, from, to)
	if err != nil {
		return nil, err
	}

	var totalEarnings int64
	for _, sale := range allTimeSales {
		totalEarnings += int64(sale.CreatorPayoutCents)
	}
	var periodEarnings int64
	for _, sale := range periodSales {
		periodEarnings += int64(sale.CreatorPayoutCents)
	}

	recentSales := make([]SaleSummary, 0, 10)
	for i, sale := range periodSales {
		if i >= 10 {
			break
		}
		recentSales = append(recentSales, SaleSummary{
			PurchaseID:         sale.PurchaseID,
			ListingTitle:       sale.ListingTitle,
			AmountCents:        sale.AmountCents,
			CreatorPayoutCents: sale.CreatorPayoutCents,
			PurchasedAt:        sale.CreatedAt,
		})
	}

	return &CreatorDashboardResponse{
		TotalSalesCount:     int64(len(allTimeSales)),
		TotalEarningsCents:  totalEarnings,
		PeriodSalesCount:    int64(len(periodSales)),
		PeriodEarningsCents: periodEarnings,
		PendingPayoutCents:  0,
		AverageRating:       0,
		TotalReviews:        0,
		RecentSales:         recentSales,
	}, nil
}

func (s *marketplaceServiceImpl) GetCreatorListings(ctx context.Context, creatorID uuid.UUID, params CreatorListingQueryParams) (*shared.PaginatedResponse[ListingDetailResponse], error) {
	listings, total, err := s.listings.GetByCreator(ctx, creatorID, &params)
	if err != nil {
		return nil, err
	}

	items := make([]ListingDetailResponse, len(listings))
	for i, l := range listings {
		files, _ := s.listingFiles.ListByListing(ctx, l.ID)
		items[i] = toListingDetailResponse(&l, files, "")
	}

	var nextCursor *string
	hasMore := int64(len(items)) < total
	if hasMore && len(items) > 0 {
		last := items[len(items)-1]
		c := shared.EncodeCursor(last.ID, last.CreatedAt)
		nextCursor = &c
	}

	return &shared.PaginatedResponse[ListingDetailResponse]{
		Data:       items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}, nil
}

func (s *marketplaceServiceImpl) GetPublisher(ctx context.Context, publisherID uuid.UUID) (*PublisherResponse, error) {
	pub, err := s.publishers.GetByID(ctx, publisherID)
	if err != nil {
		return nil, err
	}
	memberCount, err := s.publishers.CountMembers(ctx, publisherID)
	if err != nil {
		return nil, err
	}
	return toPublisherResponse(pub, memberCount), nil
}

func (s *marketplaceServiceImpl) GetPublisherMembers(ctx context.Context, publisherID, creatorID uuid.UUID) ([]PublisherMemberResponse, error) {
	// Verify caller is a member
	_, err := s.publishers.GetMemberRole(ctx, publisherID, creatorID)
	if err != nil {
		if errors.Is(err, ErrNotPublisherMember) {
			return nil, shared.ErrForbidden()
		}
		return nil, err
	}

	rows, err := s.publishers.GetMembers(ctx, publisherID)
	if err != nil {
		return nil, err
	}

	members := make([]PublisherMemberResponse, len(rows))
	for i, r := range rows {
		members[i] = PublisherMemberResponse{
			CreatorID: r.CreatorID,
			StoreName: r.StoreName,
			Role:      r.Role,
			JoinedAt:  r.CreatedAt,
		}
	}
	return members, nil
}

func (s *marketplaceServiceImpl) VerifyPublisherMembership(ctx context.Context, publisherID, creatorID uuid.UUID) (bool, error) {
	role, err := s.publishers.GetMemberRole(ctx, publisherID, creatorID)
	if err != nil {
		return false, err
	}
	return role != nil, nil
}

func (s *marketplaceServiceImpl) BrowseListings(ctx context.Context, params BrowseListingsParams) (*shared.PaginatedResponse[ListingBrowseResponse], error) {
	rows, total, err := s.listings.Browse(ctx, &params)
	if err != nil {
		return nil, err
	}

	items := make([]ListingBrowseResponse, len(rows))
	for i, r := range rows {
		items[i] = ListingBrowseResponse{
			ID:                 r.ID,
			Title:              r.Title,
			DescriptionPreview: truncateDescription(r.Description, 200),
			PriceCents:         r.PriceCents,
			ContentType:        r.ContentType,
			ThumbnailURL:       r.ThumbnailURL,
			RatingAvg:          r.RatingAvg,
			RatingCount:        r.RatingCount,
			PublisherName:      r.PublisherName,
			CreatorStoreName:   r.CreatorStoreName,
		}
	}

	var nextCursor *string
	browseHasMore := int64(len(items)) < total
	if browseHasMore && len(items) > 0 {
		last := items[len(items)-1]
		c := shared.EncodeCursor(last.ID, time.Now().UTC())
		nextCursor = &c
	}

	return &shared.PaginatedResponse[ListingBrowseResponse]{
		Data:       items,
		NextCursor: nextCursor,
		HasMore:    browseHasMore,
	}, nil
}

func (s *marketplaceServiceImpl) GetListing(ctx context.Context, listingID uuid.UUID) (*ListingDetailResponse, error) {
	listing, err := s.listings.GetByID(ctx, listingID)
	if err != nil {
		return nil, err
	}

	files, err := s.listingFiles.ListByListing(ctx, listingID)
	if err != nil {
		return nil, err
	}

	// Get publisher name
	pub, err := s.publishers.GetByID(ctx, listing.PublisherID)
	if err != nil {
		return nil, err
	}
	publisherName := ""
	if pub != nil {
		publisherName = pub.Name
	}

	resp := toListingDetailResponse(listing, files, publisherName)
	return &resp, nil
}

func (s *marketplaceServiceImpl) AutocompleteListings(ctx context.Context, query string, limit uint8) ([]AutocompleteResult, error) {
	if len(query) < 2 {
		return nil, shared.ErrValidation("query must be at least 2 characters")
	}
	if limit == 0 || limit > 10 {
		limit = 10
	}

	rows, err := s.listings.Autocomplete(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	results := make([]AutocompleteResult, len(rows))
	for i, r := range rows {
		results[i] = AutocompleteResult(r)
	}
	return results, nil
}

func (s *marketplaceServiceImpl) GetCuratedSections(ctx context.Context, itemsPerSection uint8) ([]CuratedSectionResponse, error) {
	if itemsPerSection == 0 {
		itemsPerSection = 8
	}

	sections, err := s.curatedSections.ListActive(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]CuratedSectionResponse, len(sections))
	for i, sec := range sections {
		rows, itemErr := s.curatedSections.GetSectionItems(ctx, sec.ID, itemsPerSection)
		if itemErr != nil {
			slog.Error("failed to get curated section items", "error", itemErr, "section_id", sec.ID)
			rows = nil
		}

		listings := make([]ListingBrowseResponse, len(rows))
		for j, r := range rows {
			listings[j] = ListingBrowseResponse{
				ID:                 r.ID,
				Title:              r.Title,
				DescriptionPreview: truncateDescription(r.Description, 200),
				PriceCents:         r.PriceCents,
				ContentType:        r.ContentType,
				ThumbnailURL:       r.ThumbnailURL,
				RatingAvg:          r.RatingAvg,
				RatingCount:        r.RatingCount,
				PublisherName:      r.PublisherName,
				CreatorStoreName:   r.CreatorStoreName,
			}
		}

		results[i] = CuratedSectionResponse{
			Slug:        sec.Slug,
			DisplayName: sec.DisplayName,
			Description: sec.Description,
			Listings:    listings,
		}
	}

	return results, nil
}

func (s *marketplaceServiceImpl) GetCart(ctx context.Context, scope shared.FamilyScope) (*CartResponse, error) {
	items, err := s.cart.GetItems(ctx, scope)
	if err != nil {
		return nil, err
	}

	respItems := make([]CartItemResponse, len(items))
	var totalCents int64
	for i, item := range items {
		respItems[i] = CartItemResponse{
			ListingID:    item.ListingID,
			Title:        item.Title,
			PriceCents:   item.PriceCents,
			ThumbnailURL: item.ThumbnailURL,
			AddedAt:      item.CreatedAt,
		}
		totalCents += int64(item.PriceCents)
	}

	return &CartResponse{
		Items:      respItems,
		TotalCents: totalCents,
		ItemCount:  int32(len(items)),
	}, nil
}

func (s *marketplaceServiceImpl) GetPurchases(ctx context.Context, scope shared.FamilyScope, params PurchaseQueryParams) (*shared.PaginatedResponse[PurchaseResponse], error) {
	rows, total, err := s.purchases.ListByFamily(ctx, scope, &params)
	if err != nil {
		return nil, err
	}

	items := make([]PurchaseResponse, len(rows))
	for i, r := range rows {
		items[i] = PurchaseResponse{
			ID:           r.ID,
			ListingID:    r.ListingID,
			ListingTitle: r.ListingTitle,
			AmountCents:  r.AmountCents,
			Refunded:     r.RefundedAt != nil,
			CreatedAt:    r.CreatedAt,
		}
	}

	var nextCursor *string
	if int64(len(items)) < total {
		last := rows[len(rows)-1]
		c := shared.EncodeCursor(last.ID, last.CreatedAt)
		nextCursor = &c
	}

	return &shared.PaginatedResponse[PurchaseResponse]{
		Data:       items,
		NextCursor: nextCursor,
		HasMore:    int64(len(items)) < total,
	}, nil
}

func (s *marketplaceServiceImpl) GetDownloadURL(ctx context.Context, listingID, fileID uuid.UUID, scope shared.FamilyScope) (*DownloadResponse, error) {
	// Verify purchase — no subscription tier check [S§9.4]
	_, err := s.purchases.GetByFamilyAndListing(ctx, scope.FamilyID(), listingID)
	if err != nil {
		if errors.Is(err, ErrPurchaseNotFound) {
			return nil, shared.ErrForbidden()
		}
		return nil, err
	}

	file, err := s.listingFiles.GetByID(ctx, listingID, fileID)
	if err != nil {
		return nil, err
	}

	// Generate 1-hour signed URL [ARCH §8.3]
	signedURL, err := s.media.PresignedGet(ctx, file.StorageKey, 3600)
	if err != nil {
		return nil, err
	}

	return &DownloadResponse{
		DownloadURL: signedURL,
		ExpiresAt:   time.Now().UTC().Add(1 * time.Hour),
	}, nil
}

func (s *marketplaceServiceImpl) GetListingFile(ctx context.Context, listingID, fileID uuid.UUID) (*ListingFileResponse, error) {
	file, err := s.listingFiles.GetByID(ctx, listingID, fileID)
	if err != nil {
		return nil, err
	}
	return &ListingFileResponse{
		ID:            file.ID,
		FileName:      file.FileName,
		FileSizeBytes: file.FileSizeBytes,
		MimeType:      file.MimeType,
		Version:       file.Version,
	}, nil
}

func (s *marketplaceServiceImpl) GetReview(ctx context.Context, reviewID uuid.UUID) (*ReviewResponse, error) {
	review, err := s.reviews.GetByID(ctx, reviewID)
	if err != nil {
		return nil, err
	}
	return &ReviewResponse{
		ID:                review.ID,
		ListingID:         review.ListingID,
		Rating:            review.Rating,
		ReviewText:        review.ReviewText,
		IsAnonymous:       review.IsAnonymous,
		CreatorResponse:   review.CreatorResponse,
		CreatorResponseAt: review.CreatorResponseAt,
		CreatedAt:         review.CreatedAt,
	}, nil
}

func (s *marketplaceServiceImpl) GetListingReviews(ctx context.Context, listingID uuid.UUID, params ReviewQueryParams) (*shared.PaginatedResponse[ReviewResponse], error) {
	rows, total, err := s.reviews.ListByListing(ctx, listingID, &params)
	if err != nil {
		return nil, err
	}

	items := make([]ReviewResponse, len(rows))
	for i, r := range rows {
		var reviewerName *string
		if !r.IsAnonymous && r.ReviewerFamilyName != nil {
			reviewerName = r.ReviewerFamilyName
		}

		items[i] = ReviewResponse{
			ID:                r.ID,
			ListingID:         r.ListingID,
			Rating:            r.Rating,
			ReviewText:        r.ReviewText,
			IsAnonymous:       r.IsAnonymous,
			ReviewerName:      reviewerName,
			CreatorResponse:   r.CreatorResponse,
			CreatorResponseAt: r.CreatorResponseAt,
			CreatedAt:         r.CreatedAt,
		}
	}

	var nextCursor *string
	if int64(len(items)) < total {
		last := rows[len(rows)-1]
		c := shared.EncodeCursor(last.ID, last.CreatedAt)
		nextCursor = &c
	}

	return &shared.PaginatedResponse[ReviewResponse]{
		Data:       items,
		NextCursor: nextCursor,
		HasMore:    int64(len(items)) < total,
	}, nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

// calculateSplit calculates the creator/platform split. [07-mkt §11]
func calculateSplit(listingPriceCents int64, sharePercent int) (creatorPayout, platformFee int64) {
	creatorPayout = (listingPriceCents * int64(sharePercent)) / 100
	platformFee = listingPriceCents - creatorPayout
	return creatorPayout, platformFee
}

// generateSlug creates a URL-safe slug from a name.
func generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r == ' ' || r == '-' || r == '_' {
			return '-'
		}
		return -1
	}, slug)
	// Collapse multiple hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	return slug
}

// uuidArrayToSlice converts UUIDArray ([]string) to []uuid.UUID.
func uuidArrayToSlice(arr UUIDArray) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(arr))
	for _, s := range arr {
		id, err := uuid.Parse(s)
		if err == nil {
			result = append(result, id)
		}
	}
	return result
}

// toCreatorResponse converts a GORM model to API response.
func toCreatorResponse(c *MktCreator) *CreatorResponse {
	return &CreatorResponse{
		ID:               c.ID,
		ParentID:         c.ParentID,
		OnboardingStatus: c.OnboardingStatus,
		StoreName:        c.StoreName,
		StoreBio:         c.StoreBio,
		StoreLogoURL:     c.StoreLogoURL,
		StoreBannerURL:   c.StoreBannerURL,
		CreatedAt:        c.CreatedAt,
	}
}

// toPublisherResponse converts a GORM model to API response.
func toPublisherResponse(p *MktPublisher, memberCount int32) *PublisherResponse {
	return &PublisherResponse{
		ID:          p.ID,
		Name:        p.Name,
		Slug:        p.Slug,
		Description: p.Description,
		LogoURL:     p.LogoURL,
		WebsiteURL:  p.WebsiteURL,
		IsVerified:  p.IsVerified,
		MemberCount: memberCount,
	}
}

// toListingDetailResponse converts a GORM model + files to API response.
func toListingDetailResponse(l *MktListing, files []MktListingFile, publisherName string) ListingDetailResponse {
	fileResponses := make([]ListingFileResponse, len(files))
	for i, f := range files {
		fileResponses[i] = ListingFileResponse{
			ID:            f.ID,
			FileName:      f.FileName,
			FileSizeBytes: f.FileSizeBytes,
			MimeType:      f.MimeType,
			Version:       f.Version,
		}
	}

	methodologyTags := make([]string, len(l.MethodologyTags))
	copy(methodologyTags, l.MethodologyTags)

	return ListingDetailResponse{
		ID:              l.ID,
		CreatorID:       l.CreatorID,
		PublisherID:     l.PublisherID,
		PublisherName:   publisherName,
		Title:           l.Title,
		Description:     l.Description,
		PriceCents:      l.PriceCents,
		MethodologyTags: methodologyTags,
		SubjectTags:     []string(l.SubjectTags),
		GradeMin:        l.GradeMin,
		GradeMax:        l.GradeMax,
		ContentType:     l.ContentType,
		WorldviewTags:   []string(l.WorldviewTags),
		PreviewURL:      l.PreviewURL,
		ThumbnailURL:    l.ThumbnailURL,
		Status:          l.Status,
		RatingAvg:       l.RatingAvg,
		RatingCount:     l.RatingCount,
		Version:         l.Version,
		Files:           fileResponses,
		PublishedAt:     l.PublishedAt,
		CreatedAt:       l.CreatedAt,
		UpdatedAt:       l.UpdatedAt,
	}
}

// mapDomainError converts domain errors to shared AppError types. [07-mkt §17]
func mapDomainError(err error) error {
	var domErr *domain.MktDomainError
	if errors.As(err, &domErr) {
		switch domErr.Kind {
		case domain.ErrInvalidStateTransition:
			return &shared.AppError{Code: "invalid_state_transition", Message: domErr.Error(), StatusCode: http.StatusConflict}
		case domain.ErrListingHasNoFiles:
			return &shared.AppError{Code: "listing_has_no_files", Message: domErr.Error(), StatusCode: http.StatusBadRequest}
		case domain.ErrInvalidPrice:
			return &shared.AppError{Code: "invalid_price", Message: domErr.Error(), StatusCode: http.StatusUnprocessableEntity}
		}
	}
	return err
}
