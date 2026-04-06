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

// handlePaymentWebhook godoc
//
// @Summary     Handle payment webhook
// @Tags        marketplace-hooks
// @Accept      json
// @Success     200
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Router      /hooks/payments [post]
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

// registerCreator godoc
//
// @Summary     Register as a creator
// @Tags        marketplace
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body RegisterCreatorCommand true "Creator registration"
// @Success     201 {object} CreatorResponse
// @Failure     401 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /marketplace/creators/register [post]
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

// getCreatorProfile godoc
//
// @Summary     Get my creator profile
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} CreatorResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /marketplace/creators/me [get]
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

// updateCreatorProfile godoc
//
// @Summary     Update my creator profile
// @Tags        marketplace
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body UpdateCreatorProfileCommand true "Profile fields"
// @Success     200 {object} CreatorResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /marketplace/creators/me [put]
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

// createOnboardingLink godoc
//
// @Summary     Create creator onboarding link
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} map[string]string
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /marketplace/creators/onboarding-link [post]
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

// getCreatorDashboard godoc
//
// @Summary     Get creator dashboard
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       period query string false "Dashboard period" Enums(last_7_days,last_30_days,last_90_days,all_time)
// @Success     200 {object} CreatorDashboardResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /marketplace/creators/dashboard [get]
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

// getCreatorListings godoc
//
// @Summary     Get my creator listings
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       status query string false "Filter by status"
// @Param       limit  query int    false "Results per page"
// @Param       cursor query string false "Pagination cursor"
// @Success     200 {object} CreatorListingsResult
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /marketplace/creators/listings [get]
func (h *Handler) getCreatorListings(c echo.Context) error {
	cc, err := GetCreatorContext(c)
	if err != nil {
		return err
	}
	var params CreatorListingQueryParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
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

// createPublisher godoc
//
// @Summary     Create a publisher
// @Tags        marketplace
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CreatePublisherCommand true "Publisher details"
// @Success     201 {object} PublisherResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /marketplace/publishers [post]
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

// getPublisher godoc
//
// @Summary     Get a publisher
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       publisher_id path string true "Publisher ID"
// @Success     200 {object} PublisherResponse
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/publishers/{publisher_id} [get]
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

// updatePublisher godoc
//
// @Summary     Update a publisher
// @Tags        marketplace
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       publisher_id path string                 true "Publisher ID"
// @Param       body         body UpdatePublisherCommand  true "Fields to update"
// @Success     200 {object} PublisherResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/publishers/{publisher_id} [put]
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

// getPublisherMembers godoc
//
// @Summary     Get publisher members
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       publisher_id path string true "Publisher ID"
// @Success     200 {array}  PublisherMemberResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/publishers/{publisher_id}/members [get]
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

// addPublisherMember godoc
//
// @Summary     Add a publisher member
// @Tags        marketplace
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       publisher_id path string                    true "Publisher ID"
// @Param       body         body AddPublisherMemberCommand  true "Member details"
// @Success     201 {object} PublisherMemberResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/publishers/{publisher_id}/members [post]
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

// removePublisherMember godoc
//
// @Summary     Remove a publisher member
// @Tags        marketplace
// @Security    BearerAuth
// @Param       publisher_id path string true "Publisher ID"
// @Param       creator_id   path string true "Creator ID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/publishers/{publisher_id}/members/{creator_id} [delete]
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

// createListing godoc
//
// @Summary     Create a listing
// @Tags        marketplace
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body CreateListingCommand true "Listing details"
// @Success     201 {object} ListingDetailResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /marketplace/listings [post]
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

// updateListing godoc
//
// @Summary     Update a listing
// @Tags        marketplace
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       listing_id path string               true "Listing ID"
// @Param       body       body UpdateListingCommand  true "Fields to update"
// @Success     200 {object} ListingDetailResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/listings/{listing_id} [put]
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

// submitListing godoc
//
// @Summary     Submit listing for review
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       listing_id path string true "Listing ID"
// @Success     200 {object} ListingDetailResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/listings/{listing_id}/submit [post]
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

// publishListing godoc
//
// @Summary     Publish a listing
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       listing_id path string true "Listing ID"
// @Success     200 {object} ListingDetailResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/listings/{listing_id}/publish [post]
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

// archiveListing godoc
//
// @Summary     Archive a listing
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       listing_id path string true "Listing ID"
// @Success     200 {object} ListingDetailResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/listings/{listing_id}/archive [post]
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

// uploadListingFile godoc
//
// @Summary     Upload a listing file
// @Tags        marketplace
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       listing_id path string                    true "Listing ID"
// @Param       body       body UploadListingFileCommand   true "File details"
// @Success     201 {object} ListingFileResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     413 {object} shared.AppError
// @Failure     422 {object} shared.AppError
// @Router      /marketplace/listings/{listing_id}/files [post]
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

// browseListings godoc
//
// @Summary     Browse marketplace listings
// @Tags        marketplace
// @Produce     json
// @Param       q          query string false "Search query"
// @Param       category   query string false "Filter by category"
// @Param       sort       query string false "Sort order"
// @Param       limit      query int    false "Results per page"
// @Param       cursor     query string false "Pagination cursor"
// @Success     200 {object} BrowseListingsResult
// @Failure     400 {object} shared.AppError
// @Router      /marketplace/listings [get]
func (h *Handler) browseListings(c echo.Context) error {
	var params BrowseListingsParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.BrowseListings(c.Request().Context(), params)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// autocompleteListings godoc
//
// @Summary     Autocomplete listing search
// @Tags        marketplace
// @Produce     json
// @Param       q query string true "Search query"
// @Success     200 {array}  AutocompleteResult
// @Failure     400 {object} shared.AppError
// @Router      /marketplace/listings/autocomplete [get]
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

// getListing godoc
//
// @Summary     Get a listing
// @Tags        marketplace
// @Produce     json
// @Param       listing_id path string true "Listing ID"
// @Success     200 {object} ListingDetailResponse
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/listings/{listing_id} [get]
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

// getCuratedSections godoc
//
// @Summary     Get curated marketplace sections
// @Tags        marketplace
// @Produce     json
// @Success     200 {array}  CuratedSectionResponse
// @Router      /marketplace/curated-sections [get]
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

// addToCart godoc
//
// @Summary     Add listing to cart
// @Tags        marketplace
// @Accept      json
// @Security    BearerAuth
// @Param       body body AddToCartCommand true "Cart item"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /marketplace/cart/items [post]
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

// removeFromCart godoc
//
// @Summary     Remove listing from cart
// @Tags        marketplace
// @Security    BearerAuth
// @Param       listing_id path string true "Listing ID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/cart/items/{listing_id} [delete]
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

// getCart godoc
//
// @Summary     Get shopping cart
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Success     200 {object} CartResponse
// @Failure     401 {object} shared.AppError
// @Router      /marketplace/cart [get]
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

// createCheckout godoc
//
// @Summary     Create a checkout session
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Success     201 {object} CheckoutSessionResponse
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /marketplace/cart/checkout [post]
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

// getPurchases godoc
//
// @Summary     List purchases
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       limit  query int    false "Results per page"
// @Param       cursor query string false "Pagination cursor"
// @Success     200 {object} PurchaseListResult
// @Failure     401 {object} shared.AppError
// @Router      /marketplace/purchases [get]
func (h *Handler) getPurchases(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var params PurchaseQueryParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
	}

	resp, err := h.svc.GetPurchases(c.Request().Context(), scope, params)
	if err != nil {
		return mapMktError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// getDownloadURL godoc
//
// @Summary     Get download URL for purchased file
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       listing_id path string true "Listing ID"
// @Param       file_id    path string true "File ID"
// @Success     200 {object} DownloadResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/purchases/{listing_id}/download/{file_id} [get]
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

// getFreeListing godoc
//
// @Summary     Get a free listing
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Param       listing_id path string true "Listing ID"
// @Success     201 {object} map[string]string
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /marketplace/listings/{listing_id}/get [post]
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

// createReview godoc
//
// @Summary     Create a listing review
// @Tags        marketplace
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       listing_id path string              true "Listing ID"
// @Param       body       body CreateReviewCommand  true "Review details"
// @Success     201 {object} ReviewResponse
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     409 {object} shared.AppError
// @Router      /marketplace/listings/{listing_id}/reviews [post]
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

// updateReview godoc
//
// @Summary     Update a review
// @Tags        marketplace
// @Accept      json
// @Security    BearerAuth
// @Param       review_id path string              true "Review ID"
// @Param       body      body UpdateReviewCommand  true "Fields to update"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/reviews/{review_id} [put]
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

// deleteReview godoc
//
// @Summary     Delete a review
// @Tags        marketplace
// @Security    BearerAuth
// @Param       review_id path string true "Review ID"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/reviews/{review_id} [delete]
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

// respondToReview godoc
//
// @Summary     Respond to a review (creator)
// @Tags        marketplace
// @Accept      json
// @Security    BearerAuth
// @Param       review_id path string                  true "Review ID"
// @Param       body      body RespondToReviewCommand   true "Response details"
// @Success     204
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/reviews/{review_id}/response [post]
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

// requestPayout godoc
//
// @Summary     Request a creator payout
// @Tags        marketplace
// @Produce     json
// @Security    BearerAuth
// @Success     201 {object} PayoutResult
// @Failure     400 {object} shared.AppError
// @Failure     401 {object} shared.AppError
// @Failure     403 {object} shared.AppError
// @Router      /marketplace/payouts/request [post]
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

// getListingReviews godoc
//
// @Summary     Get reviews for a listing
// @Tags        marketplace
// @Produce     json
// @Param       listing_id path string true "Listing ID"
// @Param       limit      query int   false "Results per page"
// @Param       cursor     query string false "Pagination cursor"
// @Success     200 {object} ReviewListResult
// @Failure     400 {object} shared.AppError
// @Failure     404 {object} shared.AppError
// @Router      /marketplace/listings/{listing_id}/reviews [get]
func (h *Handler) getListingReviews(c echo.Context) error {
	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		return shared.ErrBadRequest("invalid listing ID")
	}
	var params ReviewQueryParams
	if err := c.Bind(&params); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}
	if err := c.Validate(&params); err != nil {
		return shared.ValidationError(err)
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
