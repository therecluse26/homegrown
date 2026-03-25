package recs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer-Defined Cross-Domain Interfaces [MEMORY cross-domain patterns]
// ═══════════════════════════════════════════════════════════════════════════════

// IamServiceForRecs is a consumer-defined interface for cross-domain reads from iam::.
// Implemented by a function adapter in main.go over iam.Service.
// [13-recs §5, MEMORY consumer-defined interfaces]
type IamServiceForRecs interface {
	StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error)
	GetFamilyMethodologySlug(ctx context.Context, familyID shared.FamilyID) (string, error)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service Interface [13-recs §5]
// ═══════════════════════════════════════════════════════════════════════════════

// RecsService is the primary service interface for the Recommendations & Signals domain.
// Injected into HTTP handlers and event handlers.
type RecsService interface {
	// ── Commands (write side) ──────────────────────────────────

	// RecordSignal records a learning signal from a domain event.
	RecordSignal(ctx context.Context, command RecordSignalCommand) error

	// RegisterListing registers a newly published listing in the popularity catalog.
	RegisterListing(ctx context.Context, command RegisterListingCommand) error

	// DismissRecommendation marks a recommendation as dismissed and creates a feedback record.
	DismissRecommendation(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) error

	// BlockRecommendation marks a recommendation as blocked and creates feedback with blocked_entity_id.
	// Returns the blocked entity ID for inclusion in the API response. [13-recs §4]
	BlockRecommendation(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) (uuid.UUID, error)

	// UndoFeedback deletes a dismiss or block feedback record and restores the recommendation to active.
	UndoFeedback(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) error

	// UpdatePreferences updates family recommendation preferences.
	UpdatePreferences(ctx context.Context, scope *shared.FamilyScope, command UpdatePreferencesCommand) (*RecommendationPreferencesResponse, error)

	// ── Queries (read side) ────────────────────────────────────

	// GetRecommendations returns active recommendations for the family, filterable by type.
	GetRecommendations(ctx context.Context, scope *shared.FamilyScope, params RecommendationListParams) (*RecommendationListResponse, error)

	// GetStudentRecommendations returns active recommendations for a specific student.
	GetStudentRecommendations(ctx context.Context, scope *shared.FamilyScope, params StudentRecommendationParams) (*RecommendationListResponse, error)

	// GetPreferences returns the family's recommendation preferences (or defaults if none set).
	GetPreferences(ctx context.Context, scope *shared.FamilyScope) (*RecommendationPreferencesResponse, error)

	// ── Lifecycle event handlers ───────────────────────────────

	// HandleFamilyDeletion deletes all recs data for the family (family deletion cascade).
	HandleFamilyDeletion(ctx context.Context, familyID shared.FamilyID) error

	// InvalidateMethodologyCache invalidates cached methodology config.
	// Phase 1: no-op. Phase 2+: clears in-memory methodology config cache.
	InvalidateMethodologyCache(ctx context.Context) error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Repository Interfaces [13-recs §6]
// ═══════════════════════════════════════════════════════════════════════════════

// SignalRepository is the repository for recs_signals. [13-recs §6]
type SignalRepository interface {
	Create(ctx context.Context, signal NewSignal) error
	FindByFamily(ctx context.Context, scope *shared.FamilyScope, since time.Time) ([]Signal, error)
	DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error)
	DeleteStale(ctx context.Context, before time.Time) (int64, error)
}

// RecommendationRepository is the repository for recs_recommendations. [13-recs §6]
type RecommendationRepository interface {
	CreateBatch(ctx context.Context, recommendations []NewRecommendation) (int64, error)
	FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*Recommendation, error)
	FindActiveByFamily(ctx context.Context, scope *shared.FamilyScope, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error)
	FindActiveByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error)
	UpdateStatus(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID, status string) error
	ExpireStale(ctx context.Context) (int64, error)
	DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error)
}

// FeedbackRepository is the repository for recs_recommendation_feedback. [13-recs §6]
type FeedbackRepository interface {
	Create(ctx context.Context, feedback NewFeedback) error
	FindByRecommendation(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) (*Feedback, error)
	FindBlockedByFamily(ctx context.Context, scope *shared.FamilyScope) ([]uuid.UUID, error)
	Delete(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) error
	DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error)
}

// PopularityRepository is the repository for recs_popularity_scores.
// NOT family-scoped — cross-family aggregated data, no PII. [13-recs §3.3]
type PopularityRepository interface {
	Upsert(ctx context.Context, score NewPopularityScore) error
	FindByMethodology(ctx context.Context, methodologySlug string, limit int64) ([]PopularityScore, error)
	DeleteStale(ctx context.Context, before time.Time) (int64, error)
}

// PreferenceRepository is the repository for recs_preferences. [13-recs §6]
type PreferenceRepository interface {
	FindOrDefault(ctx context.Context, scope *shared.FamilyScope) (*Preferences, error)
	Upsert(ctx context.Context, scope *shared.FamilyScope, preferences UpdatePreferences) (*Preferences, error)
	DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error)
}

// AnonymizedInteractionRepository is the repository for recs_anonymized_interactions.
// NOT family-scoped — insert-only, HMAC-anonymized data (no PII). [13-recs §14]
type AnonymizedInteractionRepository interface {
	CreateBatch(ctx context.Context, interactions []NewAnonymizedInteraction) (int64, error)
}
