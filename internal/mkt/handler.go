package mkt

import (
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Handler exposes marketplace HTTP endpoints. [07-mkt §16]
type Handler struct {
	svc   MarketplaceService
	cache shared.Cache
}

// NewHandler creates a new marketplace handler.
func NewHandler(svc MarketplaceService, cache shared.Cache) *Handler {
	return &Handler{svc: svc, cache: cache}
}

// Register mounts marketplace routes on the given route groups.
// - hooksGroup: unauthenticated webhook routes (/hooks/...)
// - authGroup:  authenticated routes (/v1/...)
// - pubGroup:   public unauthenticated routes (/v1/...)
func (h *Handler) Register(authGroup, hooksGroup, pubGroup *echo.Group) {
	// ─── Webhook routes (no auth middleware) ────────────────────────
	hooksGroup.POST("/payments", h.handlePaymentWebhook)

	// ─── Public browse routes (no auth required) ────────────────────
	browse := pubGroup.Group("/marketplace")
	browse.GET("/listings", h.browseListings)
	browse.GET("/listings/autocomplete", h.autocompleteListings)
	browse.GET("/listings/:listing_id", h.getListing)
	browse.GET("/listings/:listing_id/reviews", h.getListingReviews)
	browse.GET("/curated-sections", h.getCuratedSections)

	// ─── Authenticated routes ────────────────────────────────────────
	mkt := authGroup.Group("/marketplace")

	// Creator registration (auth required, but no creator account yet)
	mkt.POST("/creators/register", h.registerCreator)

	// Publisher viewing (any authenticated user) [07-mkt §4.1]
	mkt.GET("/publishers/:publisher_id", h.getPublisher)

	// Creator-only routes (RequireCreator middleware)
	requireCreator := RequireCreator(h.cache, h.svc.GetCreatorByParentID)
	creator := mkt.Group("", requireCreator)

	creator.GET("/creators/me", h.getCreatorProfile)
	creator.PUT("/creators/me", h.updateCreatorProfile)
	creator.POST("/creators/onboarding-link", h.createOnboardingLink)
	creator.GET("/creators/dashboard", h.getCreatorDashboard)
	creator.GET("/creators/listings", h.getCreatorListings)

	creator.POST("/publishers", h.createPublisher)
	creator.PUT("/publishers/:publisher_id", h.updatePublisher)
	creator.GET("/publishers/:publisher_id/members", h.getPublisherMembers)
	creator.POST("/publishers/:publisher_id/members", h.addPublisherMember)
	creator.DELETE("/publishers/:publisher_id/members/:creator_id", h.removePublisherMember)

	creator.POST("/listings", h.createListing)
	creator.PUT("/listings/:listing_id", h.updateListing)
	creator.POST("/listings/:listing_id/submit", h.submitListing)
	creator.POST("/listings/:listing_id/publish", h.publishListing)
	creator.POST("/listings/:listing_id/archive", h.archiveListing)
	creator.POST("/listings/:listing_id/files", h.uploadListingFile)

	// Buyer routes (auth required, family-scoped)
	mkt.POST("/cart/items", h.addToCart)
	mkt.DELETE("/cart/items/:listing_id", h.removeFromCart)
	mkt.GET("/cart", h.getCart)
	mkt.POST("/cart/checkout", h.createCheckout)

	mkt.GET("/purchases", h.getPurchases)
	mkt.GET("/purchases/:listing_id/download/:file_id", h.getDownloadURL)

	mkt.POST("/listings/:listing_id/reviews", h.createReview)
	mkt.PUT("/reviews/:review_id", h.updateReview)
	mkt.DELETE("/reviews/:review_id", h.deleteReview)

	mkt.POST("/listings/:listing_id/get", h.getFreeListing)

	// Creator-only review response and payouts
	creator.POST("/reviews/:review_id/response", h.respondToReview)
	creator.POST("/payouts/request", h.requestPayout)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Webhook Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) handlePaymentWebhook(c echo.Context) error {
	payload, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return shared.ErrBadRequest("unable to read request body")
	}
	signature := c.Request().Header.Get("X-Webhook-Signature")

	if err := h.svc.HandlePaymentWebhook(c.Request().Context(), payload, signature); err != nil {
		return mapMktError(err)
	}
	return c.NoContent(http.StatusOK)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Creator Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) registerCreator(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	var cmd RegisterCreatorCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	_, err = h.svc.RegisterCreator(c.Request().Context(), cmd, auth)
	if err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetCreatorByParentID(c.Request().Context(), auth.ParentID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) getCreatorProfile(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetCreatorByParentID(c.Request().Context(), cc.Auth.ParentID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) updateCreatorProfile(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	var cmd UpdateCreatorProfileCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.UpdateCreatorProfile(c.Request().Context(), cmd, cc.CreatorID); err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetCreatorByParentID(c.Request().Context(), cc.Auth.ParentID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) createOnboardingLink(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	url, err := h.svc.CreateOnboardingLink(c.Request().Context(), cc.CreatorID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, map[string]string{"url": url})
}

func (h *Handler) getCreatorDashboard(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	period := DashboardPeriod(c.QueryParam("period"))
	if period == "" {
		period = DashboardPeriodLast30Days
	}

	resp, err := h.svc.GetCreatorDashboard(c.Request().Context(), cc.CreatorID, period)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getCreatorListings(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	var params CreatorListingQueryParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}

	resp, err := h.svc.GetCreatorListings(c.Request().Context(), cc.CreatorID, params)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Publisher Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) createPublisher(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	var cmd CreatePublisherCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	id, err := h.svc.CreatePublisher(c.Request().Context(), cmd, cc.CreatorID)
	if err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetPublisher(c.Request().Context(), id)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) getPublisher(c echo.Context) error {
	publisherID, err := uuid.Parse(c.Param("publisher_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid publisher ID")
	}

	resp, err := h.svc.GetPublisher(c.Request().Context(), publisherID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) updatePublisher(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	publisherID, err := uuid.Parse(c.Param("publisher_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid publisher ID")
	}
	var cmd UpdatePublisherCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.UpdatePublisher(c.Request().Context(), cmd, publisherID, cc.CreatorID); err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetPublisher(c.Request().Context(), publisherID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getPublisherMembers(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	publisherID, err := uuid.Parse(c.Param("publisher_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid publisher ID")
	}

	resp, err := h.svc.GetPublisherMembers(c.Request().Context(), publisherID, cc.CreatorID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) addPublisherMember(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	publisherID, err := uuid.Parse(c.Param("publisher_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid publisher ID")
	}
	var cmd AddPublisherMemberCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.AddPublisherMember(c.Request().Context(), publisherID, cmd, cc.CreatorID); err != nil {
		return mapMktError(err)
	}
	// Read-after-write: return the member details from the members list. [07-mkt §4.1]
	members, err := h.svc.GetPublisherMembers(c.Request().Context(), publisherID, cc.CreatorID)
	if err != nil {
		return mapMktError(err)
	}
	for _, m := range members {
		if m.CreatorID == cmd.CreatorID {
			return c.JSON(http.StatusCreated, m)
		}
	}
	// Fallback: member was added but not found in list (shouldn't happen)
	return c.NoContent(http.StatusCreated)
}

func (h *Handler) removePublisherMember(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	publisherID, err := uuid.Parse(c.Param("publisher_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid publisher ID")
	}
	memberCreatorID, err := uuid.Parse(c.Param("creator_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid creator ID")
	}

	if err := h.svc.RemovePublisherMember(c.Request().Context(), publisherID, memberCreatorID, cc.CreatorID); err != nil {
		return mapMktError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Listing Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) createListing(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	var cmd CreateListingCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	id, err := h.svc.CreateListing(c.Request().Context(), cmd, cc.CreatorID)
	if err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetListing(c.Request().Context(), id)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) updateListing(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}
	var cmd UpdateListingCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.UpdateListing(c.Request().Context(), cmd, listingID, cc.CreatorID); err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetListing(c.Request().Context(), listingID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) submitListing(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}

	if err := h.svc.SubmitListing(c.Request().Context(), listingID, cc.CreatorID); err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetListing(c.Request().Context(), listingID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) publishListing(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}

	if err := h.svc.PublishListing(c.Request().Context(), listingID, cc.CreatorID); err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetListing(c.Request().Context(), listingID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) archiveListing(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}

	if err := h.svc.ArchiveListing(c.Request().Context(), listingID, cc.CreatorID); err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetListing(c.Request().Context(), listingID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) uploadListingFile(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}
	var cmd UploadListingFileCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	fileID, err := h.svc.UploadListingFile(c.Request().Context(), cmd, listingID, cc.CreatorID)
	if err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetListingFile(c.Request().Context(), listingID, fileID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Browse Handlers (public)
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) browseListings(c echo.Context) error {
	var params BrowseListingsParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}

	resp, err := h.svc.BrowseListings(c.Request().Context(), params)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) autocompleteListings(c echo.Context) error {
	query := c.QueryParam("q")
	if query == "" {
		return shared.ErrBadRequest("query parameter 'q' is required")
	}

	results, err := h.svc.AutocompleteListings(c.Request().Context(), query, 10)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, results)
}

func (h *Handler) getListing(c echo.Context) error {
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}

	resp, err := h.svc.GetListing(c.Request().Context(), listingID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getCuratedSections(c echo.Context) error {
	resp, err := h.svc.GetCuratedSections(c.Request().Context(), 8)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Cart & Checkout Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) addToCart(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd AddToCartCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.AddToCart(c.Request().Context(), cmd.ListingID, scope, auth.ParentID); err != nil {
		return mapMktError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) removeFromCart(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}

	if err := h.svc.RemoveFromCart(c.Request().Context(), listingID, scope); err != nil {
		return mapMktError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) getCart(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	resp, err := h.svc.GetCart(c.Request().Context(), scope)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) createCheckout(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}

	resp, err := h.svc.CreateCheckout(c.Request().Context(), scope)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Purchase Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) getPurchases(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var params PurchaseQueryParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}

	resp, err := h.svc.GetPurchases(c.Request().Context(), scope, params)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getDownloadURL(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}
	fileID, err := uuid.Parse(c.Param("file_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid file ID")
	}

	resp, err := h.svc.GetDownloadURL(c.Request().Context(), listingID, fileID, scope)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getFreeListing(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}

	purchaseID, err := h.svc.GetFreeListing(c.Request().Context(), listingID, scope)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusCreated, map[string]string{"id": purchaseID.String()})
}

// ═══════════════════════════════════════════════════════════════════════════════
// Review Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) createReview(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}
	var cmd CreateReviewCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	id, err := h.svc.CreateReview(c.Request().Context(), cmd, listingID, scope)
	if err != nil {
		return mapMktError(err)
	}
	resp, err := h.svc.GetReview(c.Request().Context(), id)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) updateReview(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	reviewID, err := uuid.Parse(c.Param("review_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid review ID")
	}
	var cmd UpdateReviewCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.UpdateReview(c.Request().Context(), cmd, reviewID, scope); err != nil {
		return mapMktError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) deleteReview(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	reviewID, err := uuid.Parse(c.Param("review_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid review ID")
	}

	if err := h.svc.DeleteReview(c.Request().Context(), reviewID, scope); err != nil {
		return mapMktError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) respondToReview(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	reviewID, err := uuid.Parse(c.Param("review_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid review ID")
	}
	var cmd RespondToReviewCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}

	if err := h.svc.RespondToReview(c.Request().Context(), cmd, reviewID, cc.CreatorID); err != nil {
		return mapMktError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Payout Handlers
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) requestPayout(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}

	resp, err := h.svc.RequestPayout(c.Request().Context(), cc.CreatorID)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Review listing route (public)
// ═══════════════════════════════════════════════════════════════════════════════

func (h *Handler) getListingReviews(c echo.Context) error {
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}
	var params ReviewQueryParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}

	resp, err := h.svc.GetListingReviews(c.Request().Context(), listingID, params)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Error Mapping
// ═══════════════════════════════════════════════════════════════════════════════

// mapMktError maps domain errors to HTTP-appropriate shared.AppError types. [07-mkt §17]
func mapMktError(err error) error {
	if err == nil {
		return nil
	}

	// shared.AppError passes through (already mapped by service layer)
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// ─── Creator errors ─────────────────────────────────────────────── [07-mkt §17]
	switch {
	case errors.Is(err, ErrCreatorAlreadyExists):
		return &shared.AppError{Code: "creator_already_exists", Message: err.Error(), StatusCode: http.StatusConflict}
	case errors.Is(err, ErrCreatorNotFound):
		return &shared.AppError{Code: "creator_not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrTOSNotAccepted):
		return &shared.AppError{Code: "tos_not_accepted", Message: err.Error(), StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(err, ErrCreatorNotActive):
		return &shared.AppError{Code: "creator_not_active", Message: "Access denied", StatusCode: http.StatusForbidden}
	case errors.Is(err, ErrCreatorSuspended):
		return &shared.AppError{Code: "creator_suspended", Message: "Access denied", StatusCode: http.StatusForbidden}

	// ─── Publisher errors ─────────────────────────────────────────────
	case errors.Is(err, ErrPublisherNotFound):
		return &shared.AppError{Code: "publisher_not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrPublisherSlugConflict):
		return &shared.AppError{Code: "publisher_slug_conflict", Message: err.Error(), StatusCode: http.StatusConflict}
	case errors.Is(err, ErrNotPublisherMember):
		return &shared.AppError{Code: "not_publisher_member", Message: "Access denied", StatusCode: http.StatusForbidden}
	case errors.Is(err, ErrInsufficientPublisherRole):
		return &shared.AppError{Code: "insufficient_publisher_role", Message: "Access denied", StatusCode: http.StatusForbidden}
	case errors.Is(err, ErrCannotRemoveLastOwner):
		return &shared.AppError{Code: "cannot_remove_last_owner", Message: err.Error(), StatusCode: http.StatusBadRequest}
	case errors.Is(err, ErrCannotModifyPlatformPublisher):
		return &shared.AppError{Code: "cannot_modify_platform_publisher", Message: "Access denied", StatusCode: http.StatusForbidden}

	// ─── Listing errors ───────────────────────────────────────────────
	case errors.Is(err, ErrListingNotFound):
		return &shared.AppError{Code: "listing_not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrListingNotPublished):
		return &shared.AppError{Code: "listing_not_published", Message: err.Error(), StatusCode: http.StatusBadRequest}
	case errors.Is(err, ErrListingNotFree):
		return &shared.AppError{Code: "listing_not_free", Message: err.Error(), StatusCode: http.StatusBadRequest}
	case errors.Is(err, ErrNotListingOwner):
		return &shared.AppError{Code: "not_listing_owner", Message: "Access denied", StatusCode: http.StatusForbidden}
	case errors.Is(err, ErrInvalidContentType):
		return &shared.AppError{Code: "invalid_content_type", Message: err.Error(), StatusCode: http.StatusUnprocessableEntity}

	// ─── File errors ──────────────────────────────────────────────────
	case errors.Is(err, ErrFileNotFound):
		return &shared.AppError{Code: "file_not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrInvalidFileType):
		return &shared.AppError{Code: "invalid_file_type", Message: err.Error(), StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(err, ErrFileTooLarge):
		return &shared.AppError{Code: "file_too_large", Message: err.Error(), StatusCode: http.StatusRequestEntityTooLarge}

	// ─── Cart errors ──────────────────────────────────────────────────
	case errors.Is(err, ErrAlreadyInCart):
		return &shared.AppError{Code: "already_in_cart", Message: err.Error(), StatusCode: http.StatusConflict}
	case errors.Is(err, ErrNotInCart):
		return &shared.AppError{Code: "not_in_cart", Message: "Resource not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrEmptyCart):
		return &shared.AppError{Code: "empty_cart", Message: err.Error(), StatusCode: http.StatusBadRequest}
	case errors.Is(err, ErrStaleCart):
		return &shared.AppError{Code: "stale_cart", Message: err.Error(), StatusCode: http.StatusConflict}

	// ─── Purchase errors ──────────────────────────────────────────────
	case errors.Is(err, ErrAlreadyPurchased):
		return &shared.AppError{Code: "already_purchased", Message: err.Error(), StatusCode: http.StatusConflict}
	case errors.Is(err, ErrPurchaseNotFound):
		return &shared.AppError{Code: "purchase_not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrNotPurchased):
		return &shared.AppError{Code: "not_purchased", Message: "Access denied", StatusCode: http.StatusForbidden}
	case errors.Is(err, ErrRefundWindowExpired):
		return &shared.AppError{Code: "refund_window_expired", Message: err.Error(), StatusCode: http.StatusBadRequest}
	case errors.Is(err, ErrAlreadyRefunded):
		return &shared.AppError{Code: "already_refunded", Message: err.Error(), StatusCode: http.StatusConflict}

	// ─── Review errors ────────────────────────────────────────────────
	case errors.Is(err, ErrAlreadyReviewed):
		return &shared.AppError{Code: "already_reviewed", Message: err.Error(), StatusCode: http.StatusConflict}
	case errors.Is(err, ErrReviewNotFound):
		return &shared.AppError{Code: "review_not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
	case errors.Is(err, ErrNotReviewOwner):
		return &shared.AppError{Code: "not_review_owner", Message: "Access denied", StatusCode: http.StatusForbidden}
	case errors.Is(err, ErrInvalidRating):
		return &shared.AppError{Code: "invalid_rating", Message: err.Error(), StatusCode: http.StatusUnprocessableEntity}

	// ─── Payment errors ───────────────────────────────────────────────
	case errors.Is(err, ErrPaymentProviderUnavailable):
		return &shared.AppError{Code: "payment_provider_unavailable", Message: err.Error(), StatusCode: http.StatusBadGateway}
	case errors.Is(err, ErrPaymentCreationFailed):
		return &shared.AppError{Code: "payment_creation_failed", Message: err.Error(), StatusCode: http.StatusBadGateway}
	case errors.Is(err, ErrInvalidWebhookSignature):
		return &shared.AppError{Code: "invalid_webhook_signature", Message: "Authentication required", StatusCode: http.StatusUnauthorized}
	case errors.Is(err, ErrMalformedWebhookPayload):
		return &shared.AppError{Code: "malformed_webhook_payload", Message: err.Error(), StatusCode: http.StatusBadRequest}
	case errors.Is(err, ErrPayoutThresholdNotMet):
		return &shared.AppError{Code: "payout_threshold_not_met", Message: err.Error(), StatusCode: http.StatusBadRequest}

	default:
		return shared.ErrInternal(err)
	}
}
