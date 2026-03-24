package mkt

import "errors"

// ─── Creator Errors ─────────────────────────────────────────────────
var (
	ErrCreatorAlreadyExists = errors.New("creator account already exists for this user")
	ErrCreatorNotFound      = errors.New("creator account not found")
	ErrTOSNotAccepted       = errors.New("creator must accept Terms of Service")
	ErrCreatorNotActive     = errors.New("creator onboarding not complete")
	ErrCreatorSuspended     = errors.New("creator account is suspended")
)

// ─── Publisher Errors ───────────────────────────────────────────────
var (
	ErrPublisherNotFound             = errors.New("publisher not found")
	ErrPublisherSlugConflict         = errors.New("publisher slug already taken")
	ErrNotPublisherMember            = errors.New("not a member of this publisher")
	ErrInsufficientPublisherRole     = errors.New("insufficient publisher role for this action")
	ErrCannotRemoveLastOwner         = errors.New("cannot remove the last owner of a publisher")
	ErrCannotModifyPlatformPublisher = errors.New("cannot modify the platform publisher")
)

// ─── Listing Errors ─────────────────────────────────────────────────
var (
	ErrListingNotFound     = errors.New("listing not found")
	ErrListingNotPublished = errors.New("listing is not published")
	ErrListingNotFree      = errors.New("listing is not free")
	ErrNotListingOwner     = errors.New("not the owner of this listing")
	ErrInvalidContentType  = errors.New("invalid content type")
)

// ─── File Errors ────────────────────────────────────────────────────
var (
	ErrFileNotFound    = errors.New("file not found")
	ErrInvalidFileType = errors.New("invalid file type")
	ErrFileTooLarge    = errors.New("file too large")
)

// ─── Cart Errors ────────────────────────────────────────────────────
var (
	ErrAlreadyInCart = errors.New("item already in cart")
	ErrNotInCart     = errors.New("item not in cart")
	ErrEmptyCart     = errors.New("cart is empty")
	ErrStaleCart     = errors.New("cart contains unpublished listings")
)

// ─── Purchase Errors ────────────────────────────────────────────────
var (
	ErrAlreadyPurchased    = errors.New("already purchased this listing")
	ErrPurchaseNotFound    = errors.New("purchase not found")
	ErrNotPurchased        = errors.New("not purchased — cannot download")
	ErrRefundWindowExpired = errors.New("refund window has expired (30 days)")
	ErrAlreadyRefunded     = errors.New("purchase already refunded")
)

// ─── Review Errors ──────────────────────────────────────────────────
var (
	ErrAlreadyReviewed = errors.New("already reviewed this purchase")
	ErrReviewNotFound  = errors.New("review not found")
	ErrNotReviewOwner  = errors.New("not the owner of this review")
	ErrInvalidRating   = errors.New("invalid rating: must be between 1 and 5")
)

// ─── Payment Errors (processor-agnostic) ────────────────────────────
var (
	ErrPaymentProviderUnavailable = errors.New("payment provider unavailable")
	ErrPaymentCreationFailed      = errors.New("payment creation failed")
	ErrInvalidWebhookSignature    = errors.New("invalid webhook signature")
	ErrMalformedWebhookPayload    = errors.New("webhook payload malformed")
	ErrPayoutThresholdNotMet      = errors.New("payout threshold not met")
)
