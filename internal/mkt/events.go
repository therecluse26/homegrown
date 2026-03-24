package mkt

import "github.com/google/uuid"

// PurchaseCompleted is published when a family completes a marketplace purchase.
// Consumed by learn:: (tool access), billing:: (creator earnings), notify:: (receipt).
type PurchaseCompleted struct {
	FamilyID        uuid.UUID        `json:"family_id"`
	PurchaseID      uuid.UUID        `json:"purchase_id"`
	ListingID       uuid.UUID        `json:"listing_id"`
	ContentMetadata PurchaseMetadata `json:"content_metadata"`
}

func (PurchaseCompleted) EventName() string { return "mkt.PurchaseCompleted" }

// PurchaseMetadata holds metadata about purchased content.
// Cross-domain contract consumed by learn::event_handlers. [06-learn:2532]
type PurchaseMetadata struct {
	ContentType string      `json:"content_type"`
	ContentIDs  []uuid.UUID `json:"content_ids"`
	PublisherID uuid.UUID   `json:"publisher_id"`
}

// ListingPublished is published when a listing transitions to Published state.
// Consumed by search:: (index update), recs:: (recommendation catalog).
type ListingPublished struct {
	ListingID   uuid.UUID `json:"listing_id"`
	PublisherID uuid.UUID `json:"publisher_id"`
	ContentType string    `json:"content_type"`
	SubjectTags []string  `json:"subject_tags"`
}

func (ListingPublished) EventName() string { return "mkt.ListingPublished" }

// ListingArchived is published when a listing is archived.
// Consumed by search:: (remove from index).
type ListingArchived struct {
	ListingID uuid.UUID `json:"listing_id"`
}

func (ListingArchived) EventName() string { return "mkt.ListingArchived" }

// ListingSubmitted is published when a listing is submitted for content screening.
// Consumed by safety:: (automated content screening).
type ListingSubmitted struct {
	ListingID uuid.UUID `json:"listing_id"`
	CreatorID uuid.UUID `json:"creator_id"`
}

func (ListingSubmitted) EventName() string { return "mkt.ListingSubmitted" }

// ReviewCreated is published when a verified-purchaser review is created.
// Consumed by safety:: (content moderation scan).
type ReviewCreated struct {
	ReviewID   uuid.UUID `json:"review_id"`
	ListingID  uuid.UUID `json:"listing_id"`
	Rating     int16     `json:"rating"`
	ReviewText *string   `json:"review_text,omitempty"`
}

func (ReviewCreated) EventName() string { return "mkt.ReviewCreated" }

// CreatorOnboarded is published when a creator completes registration.
// Consumed by notify:: (welcome email).
type CreatorOnboarded struct {
	CreatorID uuid.UUID `json:"creator_id"`
	ParentID  uuid.UUID `json:"parent_id"`
	StoreName string    `json:"store_name"`
}

func (CreatorOnboarded) EventName() string { return "mkt.CreatorOnboarded" }

// PurchaseRefunded is published when a purchase is refunded.
// Consumed by billing:: (earnings adjustment), notify:: (refund notification).
type PurchaseRefunded struct {
	PurchaseID        uuid.UUID `json:"purchase_id"`
	ListingID         uuid.UUID `json:"listing_id"`
	FamilyID          uuid.UUID `json:"family_id"`
	RefundAmountCents int64     `json:"refund_amount_cents"`
}

func (PurchaseRefunded) EventName() string { return "mkt.PurchaseRefunded" }

// ContentSubmittedForModeration is published when review or listing content needs moderation.
// Consumed by safety:: (text scanning). [11-safety §11.2]
type ContentSubmittedForModeration struct {
	ContentID   uuid.UUID `json:"content_id"`
	ContentType string    `json:"content_type"`
	Text        string    `json:"text"`
}

func (ContentSubmittedForModeration) EventName() string {
	return "mkt.ContentSubmittedForModeration"
}
