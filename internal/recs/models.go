package recs

import (
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Enums & Constants [13-recs §8]
// ═══════════════════════════════════════════════════════════════════════════════

// SignalType represents learning signal types derived from domain events.
type SignalType string

const (
	SignalActivityLogged    SignalType = "activity_logged"
	SignalBookCompleted     SignalType = "book_completed"
	SignalPurchaseCompleted SignalType = "purchase_completed"
)

func (s SignalType) String() string { return string(s) }

// RecommendationType represents recommendation output categories. [13-recs §10.2]
type RecommendationType string

const (
	RecommendationMarketplaceContent RecommendationType = "marketplace_content"
	RecommendationActivityIdea       RecommendationType = "activity_idea"
	RecommendationReadingSuggestion  RecommendationType = "reading_suggestion"
	RecommendationCommunityGroup     RecommendationType = "community_group"
)

func (r RecommendationType) String() string { return string(r) }

// validRecommendationTypes is the set of valid RecommendationType values.
var validRecommendationTypes = map[RecommendationType]struct{}{
	RecommendationMarketplaceContent: {},
	RecommendationActivityIdea:       {},
	RecommendationReadingSuggestion:  {},
	RecommendationCommunityGroup:     {},
}

// IsValidRecommendationType returns true if the value is a known RecommendationType.
func IsValidRecommendationType(s string) bool {
	_, ok := validRecommendationTypes[RecommendationType(s)]
	return ok
}

// SourceSignalType represents the signal that produced a recommendation (transparency). [13-recs §13.1]
type SourceSignalType string

const (
	SourceMethodologyMatch SourceSignalType = "methodology_match"
	SourcePopularity       SourceSignalType = "popularity"
	SourceSeasonal         SourceSignalType = "seasonal"
	SourceProgressGap      SourceSignalType = "progress_gap"
	SourceAgeTransition    SourceSignalType = "age_transition"
	SourcePurchaseHistory  SourceSignalType = "purchase_history"
	SourceReadingHistory   SourceSignalType = "reading_history"
	SourceExploration      SourceSignalType = "exploration"
)

func (s SourceSignalType) String() string { return string(s) }

// ExplorationFrequency controls how often recommendations venture outside typical patterns.
// Powers filter-bubble prevention. [S§10.4, 13-recs §10.7]
type ExplorationFrequency string

const (
	ExplorationOff        ExplorationFrequency = "off"
	ExplorationOccasional ExplorationFrequency = "occasional"
	ExplorationFrequent   ExplorationFrequency = "frequent"
)

func (e ExplorationFrequency) String() string { return string(e) }

// ExplorationRatio returns the fraction of recommendation slots reserved for exploration.
// [13-recs §10.7, §8]
func (e ExplorationFrequency) ExplorationRatio() float32 {
	switch e {
	case ExplorationOff:
		return 0.0
	case ExplorationOccasional:
		return 0.10
	case ExplorationFrequent:
		return 0.25
	default:
		return 0.10
	}
}

// validExplorationFrequencies is the set of valid ExplorationFrequency values.
var validExplorationFrequencies = map[ExplorationFrequency]struct{}{
	ExplorationOff:        {},
	ExplorationOccasional: {},
	ExplorationFrequent:   {},
}

// IsValidExplorationFrequency returns true if the value is a known ExplorationFrequency.
func IsValidExplorationFrequency(s string) bool {
	_, ok := validExplorationFrequencies[ExplorationFrequency(s)]
	return ok
}

// ═══════════════════════════════════════════════════════════════════════════════
// Request / Response Types (API boundary) [13-recs §8]
// ═══════════════════════════════════════════════════════════════════════════════

// RecommendationListParams holds query parameters for GET /v1/recommendations.
type RecommendationListParams struct {
	Type   *string `query:"type"`
	Cursor *string `query:"cursor"`
	Limit  *int64  `query:"limit"`
}

// StudentRecommendationParams holds query parameters for GET /v1/recommendations/students/:student_id.
type StudentRecommendationParams struct {
	StudentID uuid.UUID `param:"student_id"`
	Type      *string   `query:"type"`
	Cursor    *string   `query:"cursor"`
	Limit     *int64    `query:"limit"`
}

// UpdatePreferencesCommand is the request body for PATCH /v1/recommendations/preferences.
type UpdatePreferencesCommand struct {
	EnabledTypes         []string `json:"enabled_types,omitempty"`
	ExplorationFrequency *string  `json:"exploration_frequency,omitempty"`
}

// RecommendationListResponse is the response for recommendation list endpoints.
type RecommendationListResponse struct {
	Recommendations []RecommendationResponse `json:"recommendations"`
	NextCursor      *string                  `json:"next_cursor,omitempty"`
}

// RecommendationResponse is a single recommendation in a list response.
type RecommendationResponse struct {
	ID                 uuid.UUID  `json:"id"`
	RecommendationType string     `json:"recommendation_type"`
	TargetEntityID     uuid.UUID  `json:"target_entity_id"`
	TargetEntityLabel  string     `json:"target_entity_label"`
	SourceSignal       string     `json:"source_signal"`
	SourceLabel        string     `json:"source_label"`
	Score              float32    `json:"score"`
	IsSuggestion       bool       `json:"is_suggestion"` // always true [S§10.4]
	StudentID          *uuid.UUID `json:"student_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	ExpiresAt          time.Time  `json:"expires_at"`
}

// RecommendationPreferencesResponse is the response for preferences endpoints.
type RecommendationPreferencesResponse struct {
	EnabledTypes         []string `json:"enabled_types"`
	ExplorationFrequency string   `json:"exploration_frequency"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Command Types (write side) [13-recs §5]
// ═══════════════════════════════════════════════════════════════════════════════

// RecordSignalCommand records a learning signal from a domain event.
type RecordSignalCommand struct {
	FamilyID        shared.FamilyID
	StudentID       *uuid.UUID
	SignalType      SignalType
	MethodologySlug string
	Payload         map[string]any
	SignalDate      time.Time
}

// RegisterListingCommand registers a newly published listing in the popularity catalog.
type RegisterListingCommand struct {
	ListingID   uuid.UUID
	PublisherID uuid.UUID
	ContentType string
	SubjectTags []string
}

// ═══════════════════════════════════════════════════════════════════════════════
// Internal Data Types (repository boundary) [13-recs §6]
// ═══════════════════════════════════════════════════════════════════════════════

// NewSignal is the write model for recs_signals.
type NewSignal struct {
	FamilyID        shared.FamilyID
	StudentID       *uuid.UUID
	SignalType      SignalType
	MethodologySlug string
	Payload         map[string]any
	SignalDate      time.Time
}

// Signal is the read model for recs_signals.
type Signal struct {
	ID              uuid.UUID
	FamilyID        shared.FamilyID
	StudentID       *uuid.UUID
	SignalType      SignalType
	MethodologySlug string
	Payload         map[string]any
	SignalDate      time.Time
	CreatedAt       time.Time
}

// NewRecommendation is the write model for recs_recommendations.
type NewRecommendation struct {
	FamilyID           shared.FamilyID
	StudentID          *uuid.UUID
	RecommendationType RecommendationType
	TargetEntityID     uuid.UUID
	TargetEntityLabel  string
	SourceSignal       SourceSignalType
	SourceLabel        string
	Score              float32
	ExpiresAt          time.Time
}

// Recommendation is the read model for recs_recommendations.
type Recommendation struct {
	ID                 uuid.UUID
	FamilyID           shared.FamilyID
	StudentID          *uuid.UUID
	RecommendationType RecommendationType
	TargetEntityID     uuid.UUID
	TargetEntityLabel  string
	SourceSignal       SourceSignalType
	SourceLabel        string
	Score              float32
	Status             string
	ExpiresAt          time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// NewFeedback is the write model for recs_recommendation_feedback.
type NewFeedback struct {
	FamilyID         shared.FamilyID
	RecommendationID uuid.UUID
	Action           string // "dismiss" or "block"
	BlockedEntityID  *uuid.UUID
}

// Feedback is the read model for recs_recommendation_feedback.
type Feedback struct {
	ID               uuid.UUID
	FamilyID         shared.FamilyID
	RecommendationID uuid.UUID
	Action           string
	BlockedEntityID  *uuid.UUID
	CreatedAt        time.Time
}

// NewPopularityScore is the write model for recs_popularity_scores.
type NewPopularityScore struct {
	ListingID       uuid.UUID
	MethodologySlug string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	PopularityScore float32
	PurchaseCount   int
}

// PopularityScore is the read model for recs_popularity_scores.
type PopularityScore struct {
	ID              uuid.UUID
	ListingID       uuid.UUID
	MethodologySlug string
	PeriodStart     time.Time
	PeriodEnd       time.Time
	PopularityScore float32
	PurchaseCount   int
	ComputedAt      time.Time
}

// Preferences is the read model for recs_preferences.
type Preferences struct {
	ID                   uuid.UUID
	FamilyID             shared.FamilyID
	EnabledTypes         []string
	ExplorationFrequency string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// UpdatePreferences is the write model for PreferenceRepository.Upsert.
type UpdatePreferences struct {
	EnabledTypes         []string
	ExplorationFrequency string
}

// NewAnonymizedInteraction is the write model for recs_anonymized_interactions.
// MUST NOT contain family_id or student_id. [13-recs §14.5]
type NewAnonymizedInteraction struct {
	AnonymousID     string  // HMAC-SHA256(family_id, server_secret) [13-recs §14.3]
	InteractionType string
	MethodologySlug string
	AgeBand         string
	SubjectCategory *string
	DurationMinutes *int
	InteractionDate time.Time
}
