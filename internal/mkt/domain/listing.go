package domain

import (
	"time"

	"github.com/google/uuid"
)

// ListingState represents the lifecycle state of a marketplace listing.
type ListingState string

const (
	ListingStateDraft     ListingState = "draft"
	ListingStateSubmitted ListingState = "submitted"
	ListingStatePublished ListingState = "published"
	ListingStateArchived  ListingState = "archived"
)

// MarketplaceListing is the aggregate root for content listings.
// All fields are unexported; state transitions happen via methods only. [ARCH §4.5]
type MarketplaceListing struct {
	id              uuid.UUID
	creatorID       uuid.UUID
	publisherID     uuid.UUID
	title           string
	description     string
	priceCents      int32
	methodologyTags []uuid.UUID
	subjectTags     []string
	gradeMin        *int16
	gradeMax        *int16
	contentType     string
	worldviewTags   []string
	previewURL      *string
	thumbnailURL    *string
	state           ListingState
	ratingAvg       float64
	ratingCount     int32
	version         int32
	publishedAt     *time.Time
	archivedAt      *time.Time
	fileCount       int64
	createdAt       time.Time
	updatedAt       time.Time
}

// FromPersistence reconstructs a MarketplaceListing from persisted data.
func FromPersistence(
	id, creatorID, publisherID uuid.UUID,
	title, description string,
	priceCents int32,
	methodologyTags []uuid.UUID,
	subjectTags []string,
	gradeMin, gradeMax *int16,
	contentType string,
	worldviewTags []string,
	previewURL, thumbnailURL *string,
	state ListingState,
	ratingAvg float64,
	ratingCount int32,
	version int32,
	publishedAt, archivedAt *time.Time,
	fileCount int64,
	createdAt, updatedAt time.Time,
) *MarketplaceListing {
	return &MarketplaceListing{
		id:              id,
		creatorID:       creatorID,
		publisherID:     publisherID,
		title:           title,
		description:     description,
		priceCents:      priceCents,
		methodologyTags: methodologyTags,
		subjectTags:     subjectTags,
		gradeMin:        gradeMin,
		gradeMax:        gradeMax,
		contentType:     contentType,
		worldviewTags:   worldviewTags,
		previewURL:      previewURL,
		thumbnailURL:    thumbnailURL,
		state:           state,
		ratingAvg:       ratingAvg,
		ratingCount:     ratingCount,
		version:         version,
		publishedAt:     publishedAt,
		archivedAt:      archivedAt,
		fileCount:       fileCount,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}
}

// ─── Queries ────────────────────────────────────────────────────────

func (l *MarketplaceListing) ID() uuid.UUID          { return l.id }
func (l *MarketplaceListing) State() ListingState     { return l.state }
func (l *MarketplaceListing) CreatorID() uuid.UUID    { return l.creatorID }
func (l *MarketplaceListing) PublisherID() uuid.UUID  { return l.publisherID }
func (l *MarketplaceListing) Version() int32          { return l.version }
func (l *MarketplaceListing) Title() string           { return l.title }
func (l *MarketplaceListing) Description() string     { return l.description }
func (l *MarketplaceListing) PriceCents() int32       { return l.priceCents }
func (l *MarketplaceListing) ContentType() string     { return l.contentType }
func (l *MarketplaceListing) SubjectTags() []string   { return l.subjectTags }
func (l *MarketplaceListing) PublishedAt() *time.Time { return l.publishedAt }
func (l *MarketplaceListing) ArchivedAt() *time.Time  { return l.archivedAt }

// ─── State Transitions ──────────────────────────────────────────────

// ListingSubmittedEvent is emitted when a listing transitions Draft → Submitted.
type ListingSubmittedEvent struct {
	ListingID uuid.UUID
	CreatorID uuid.UUID
}

// ListingPublishedEvent is emitted when a listing transitions Submitted → Published.
type ListingPublishedEvent struct {
	ListingID   uuid.UUID
	PublisherID uuid.UUID
	ContentType string
	SubjectTags []string
}

// ListingArchivedEvent is emitted when a listing transitions Published → Archived.
type ListingArchivedEvent struct {
	ListingID uuid.UUID
}

// Submit transitions Draft → Submitted. Requires at least 1 file.
func (l *MarketplaceListing) Submit() (*ListingSubmittedEvent, error) {
	if l.state != ListingStateDraft {
		return nil, &MktDomainError{
			Kind:   ErrInvalidStateTransition,
			From:   string(l.state),
			Action: "submit",
		}
	}
	if l.fileCount == 0 {
		return nil, &MktDomainError{Kind: ErrListingHasNoFiles}
	}
	l.state = ListingStateSubmitted
	return &ListingSubmittedEvent{
		ListingID: l.id,
		CreatorID: l.creatorID,
	}, nil
}

// Publish transitions Submitted → Published. Sets published_at.
func (l *MarketplaceListing) Publish() (*ListingPublishedEvent, error) {
	if l.state != ListingStateSubmitted {
		return nil, &MktDomainError{
			Kind:   ErrInvalidStateTransition,
			From:   string(l.state),
			Action: "publish",
		}
	}
	l.state = ListingStatePublished
	now := time.Now().UTC()
	l.publishedAt = &now
	return &ListingPublishedEvent{
		ListingID:   l.id,
		PublisherID: l.publisherID,
		ContentType: l.contentType,
		SubjectTags: l.subjectTags,
	}, nil
}

// Reject transitions Submitted → Draft (content screening failed).
func (l *MarketplaceListing) Reject() error {
	if l.state != ListingStateSubmitted {
		return &MktDomainError{
			Kind:   ErrInvalidStateTransition,
			From:   string(l.state),
			Action: "reject",
		}
	}
	l.state = ListingStateDraft
	return nil
}

// Archive transitions Published → Archived. Sets archived_at.
func (l *MarketplaceListing) Archive() (*ListingArchivedEvent, error) {
	if l.state != ListingStatePublished {
		return nil, &MktDomainError{
			Kind:   ErrInvalidStateTransition,
			From:   string(l.state),
			Action: "archive",
		}
	}
	l.state = ListingStateArchived
	now := time.Now().UTC()
	l.archivedAt = &now
	return &ListingArchivedEvent{ListingID: l.id}, nil
}

// VersionSnapshot holds pre-update state for version history.
type VersionSnapshot struct {
	Version     int32
	Title       string
	Description string
	PriceCents  int32
}

// UpdatePublished updates a published or draft listing.
// Returns a VersionSnapshot (non-nil only for published listings) for version history.
func (l *MarketplaceListing) UpdatePublished(title, description *string, priceCents *int32) (*VersionSnapshot, error) {
	if l.state != ListingStatePublished && l.state != ListingStateDraft {
		return nil, &MktDomainError{
			Kind:   ErrInvalidStateTransition,
			From:   string(l.state),
			Action: "update",
		}
	}

	var snapshot *VersionSnapshot
	if l.state == ListingStatePublished {
		snapshot = &VersionSnapshot{
			Version:     l.version,
			Title:       l.title,
			Description: l.description,
			PriceCents:  l.priceCents,
		}
		l.version++
	}

	if title != nil {
		l.title = *title
	}
	if description != nil {
		l.description = *description
	}
	if priceCents != nil {
		l.priceCents = *priceCents
	}
	l.updatedAt = time.Now().UTC()

	return snapshot, nil
}
