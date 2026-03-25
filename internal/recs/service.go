package recs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Service Implementation [13-recs §5]
// ═══════════════════════════════════════════════════════════════════════════════

// RecsServiceImpl implements RecsService.
type RecsServiceImpl struct {
	signalRepo     SignalRepository
	recRepo        RecommendationRepository
	feedbackRepo   FeedbackRepository
	popularityRepo PopularityRepository
	prefRepo       PreferenceRepository
	anonRepo       AnonymizedInteractionRepository
	iamSvc         IamServiceForRecs
}

// NewRecsService creates a RecsService backed by the provided repositories.
func NewRecsService(
	signalRepo SignalRepository,
	recRepo RecommendationRepository,
	feedbackRepo FeedbackRepository,
	popularityRepo PopularityRepository,
	prefRepo PreferenceRepository,
	anonRepo AnonymizedInteractionRepository,
	iamSvc IamServiceForRecs,
) RecsService {
	return &RecsServiceImpl{
		signalRepo:     signalRepo,
		recRepo:        recRepo,
		feedbackRepo:   feedbackRepo,
		popularityRepo: popularityRepo,
		prefRepo:       prefRepo,
		anonRepo:       anonRepo,
		iamSvc:         iamSvc,
	}
}

// ─── Commands (write side) ────────────────────────────────────────────────────

// RecordSignal records a learning signal derived from a domain event. [13-recs §9]
func (s *RecsServiceImpl) RecordSignal(ctx context.Context, command RecordSignalCommand) error {
	// Resolve methodology slug from IAM if not provided by the event handler. [13-recs §9.3]
	if command.MethodologySlug == "" {
		slug, err := s.iamSvc.GetFamilyMethodologySlug(ctx, command.FamilyID)
		if err != nil {
			slog.Error("recs: resolve methodology slug", "family_id", command.FamilyID, "error", err)
			// Non-fatal — record signal with empty slug rather than dropping it.
		} else {
			command.MethodologySlug = slug
		}
	}

	signal := NewSignal(command)

	if err := s.signalRepo.Create(ctx, signal); err != nil {
		slog.Error("recs: signal recording failed",
			"signal_type", command.SignalType,
			"family_id", command.FamilyID,
			"error", err,
		)
		return fmt.Errorf("%w: %w", ErrSignalRecordingFailed, err)
	}
	return nil
}

// RegisterListing registers a newly published listing in the popularity catalog. [13-recs §9.4]
// Creates an initial zero-score record so the listing can accumulate popularity scores
// once families start purchasing it. The AggregatePopularityTask will compute real scores.
func (s *RecsServiceImpl) RegisterListing(ctx context.Context, command RegisterListingCommand) error {
	score := NewPopularityScore{
		ListingID:       command.ListingID,
		MethodologySlug: "", // methodology-agnostic seed — real scores computed by AggregatePopularityTask
		PopularityScore: 0.0,
		PurchaseCount:   0,
	}

	if err := s.popularityRepo.Upsert(ctx, score); err != nil {
		return fmt.Errorf("recs: register listing: %w", err)
	}
	return nil
}

// DismissRecommendation marks a recommendation as dismissed and records feedback. [13-recs §13.2]
func (s *RecsServiceImpl) DismissRecommendation(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) error {
	// Check for existing feedback (idempotency guard).
	existing, err := s.feedbackRepo.FindByRecommendation(ctx, scope, recommendationID)
	if err != nil && !errors.Is(err, ErrFeedbackNotFound) {
		return fmt.Errorf("recs: dismiss recommendation: %w", err)
	}
	if existing != nil {
		return ErrAlreadyHasFeedback
	}

	// Update the recommendation status to "dismissed".
	if err := s.recRepo.UpdateStatus(ctx, scope, recommendationID, "dismissed"); err != nil {
		if errors.Is(err, ErrRecommendationNotFound) {
			return ErrRecommendationNotFound
		}
		return fmt.Errorf("recs: dismiss recommendation update status: %w", err)
	}

	// Create the feedback record.
	feedback := NewFeedback{
		FamilyID:         shared.NewFamilyID(scope.FamilyID()),
		RecommendationID: recommendationID,
		Action:           "dismiss",
	}
	if err := s.feedbackRepo.Create(ctx, feedback); err != nil {
		return fmt.Errorf("recs: dismiss recommendation create feedback: %w", err)
	}
	return nil
}

// BlockRecommendation marks a recommendation as blocked and records feedback with the
// blocked entity ID for future suppression. Returns the blocked entity ID. [13-recs §13.2]
func (s *RecsServiceImpl) BlockRecommendation(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) (uuid.UUID, error) {
	// Direct lookup instead of scanning all recommendations. [Gap #4 perf fix]
	rec, err := s.recRepo.FindByID(ctx, scope, recommendationID)
	if err != nil {
		if errors.Is(err, ErrRecommendationNotFound) {
			return uuid.Nil, ErrRecommendationNotFound
		}
		return uuid.Nil, fmt.Errorf("recs: block recommendation: %w", err)
	}

	// Check for existing feedback.
	existing, err := s.feedbackRepo.FindByRecommendation(ctx, scope, recommendationID)
	if err != nil && !errors.Is(err, ErrFeedbackNotFound) {
		return uuid.Nil, fmt.Errorf("recs: block recommendation: %w", err)
	}
	if existing != nil {
		return uuid.Nil, ErrAlreadyHasFeedback
	}

	// Update the recommendation status to "blocked".
	if err := s.recRepo.UpdateStatus(ctx, scope, recommendationID, "blocked"); err != nil {
		if errors.Is(err, ErrRecommendationNotFound) {
			return uuid.Nil, ErrRecommendationNotFound
		}
		return uuid.Nil, fmt.Errorf("recs: block recommendation update status: %w", err)
	}

	// Create the feedback record with the blocked entity ID.
	targetEntityID := rec.TargetEntityID
	feedback := NewFeedback{
		FamilyID:         shared.NewFamilyID(scope.FamilyID()),
		RecommendationID: recommendationID,
		Action:           "block",
		BlockedEntityID:  &targetEntityID,
	}
	if err := s.feedbackRepo.Create(ctx, feedback); err != nil {
		return uuid.Nil, fmt.Errorf("recs: block recommendation create feedback: %w", err)
	}
	return targetEntityID, nil
}

// UndoFeedback deletes a dismiss or block feedback and restores the recommendation to active. [13-recs §13.2]
func (s *RecsServiceImpl) UndoFeedback(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) error {
	// Verify feedback exists before attempting deletion.
	existing, err := s.feedbackRepo.FindByRecommendation(ctx, scope, recommendationID)
	if err != nil {
		if errors.Is(err, ErrFeedbackNotFound) {
			return ErrFeedbackNotFound
		}
		return fmt.Errorf("recs: undo feedback: %w", err)
	}
	if existing == nil {
		return ErrFeedbackNotFound
	}

	// Delete the feedback record.
	if err := s.feedbackRepo.Delete(ctx, scope, recommendationID); err != nil {
		return fmt.Errorf("recs: undo feedback delete: %w", err)
	}

	// Restore the recommendation to active status only if it has not expired. [13-recs §4]
	rec, err := s.recRepo.FindByID(ctx, scope, recommendationID)
	if err != nil {
		// Feedback was deleted — that's the primary goal. Log but don't fail.
		slog.Error("recs: undo feedback lookup for expiry check", "recommendation_id", recommendationID, "error", err)
		return nil
	}
	if rec.ExpiresAt.Before(time.Now().UTC()) {
		// Recommendation has expired — feedback is removed (unblocks entity) but rec stays expired.
		return nil
	}
	if err := s.recRepo.UpdateStatus(ctx, scope, recommendationID, "active"); err != nil {
		return fmt.Errorf("recs: undo feedback restore status: %w", err)
	}
	return nil
}

// UpdatePreferences updates the family's recommendation preferences. [13-recs §5]
func (s *RecsServiceImpl) UpdatePreferences(ctx context.Context, scope *shared.FamilyScope, command UpdatePreferencesCommand) (*RecommendationPreferencesResponse, error) {
	// Validate exploration frequency if provided.
	if command.ExplorationFrequency != nil {
		if !IsValidExplorationFrequency(*command.ExplorationFrequency) {
			return nil, ErrInvalidExplorationFrequency
		}
	}

	// Validate enabled types if provided.
	for _, t := range command.EnabledTypes {
		if !IsValidRecommendationType(t) {
			return nil, ErrInvalidRecommendationType
		}
	}

	// Load current preferences to apply partial update.
	current, err := s.prefRepo.FindOrDefault(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("recs: update preferences: %w", err)
	}

	update := UpdatePreferences{
		EnabledTypes:         current.EnabledTypes,
		ExplorationFrequency: current.ExplorationFrequency,
	}
	if len(command.EnabledTypes) > 0 {
		update.EnabledTypes = command.EnabledTypes
	}
	if command.ExplorationFrequency != nil {
		update.ExplorationFrequency = *command.ExplorationFrequency
	}

	prefs, err := s.prefRepo.Upsert(ctx, scope, update)
	if err != nil {
		return nil, fmt.Errorf("recs: update preferences upsert: %w", err)
	}
	return prefsToResponse(prefs), nil
}

// ─── Queries (read side) ──────────────────────────────────────────────────────

// GetRecommendations returns the family's active recommendations. [13-recs §4]
func (s *RecsServiceImpl) GetRecommendations(ctx context.Context, scope *shared.FamilyScope, params RecommendationListParams) (*RecommendationListResponse, error) {
	limit := effectiveLimit(params.Limit)

	recs, nextCursor, err := s.recRepo.FindActiveByFamily(ctx, scope, params.Type, params.Cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("recs: get recommendations: %w", err)
	}
	return &RecommendationListResponse{
		Recommendations: recsToResponses(recs),
		NextCursor:      nextCursor,
	}, nil
}

// GetStudentRecommendations returns recommendations specific to a student. [13-recs §4]
func (s *RecsServiceImpl) GetStudentRecommendations(ctx context.Context, scope *shared.FamilyScope, params StudentRecommendationParams) (*RecommendationListResponse, error) {
	belongs, err := s.iamSvc.StudentBelongsToFamily(ctx, params.StudentID, shared.NewFamilyID(scope.FamilyID()))
	if err != nil {
		return nil, fmt.Errorf("recs: get student recommendations: %w", err)
	}
	if !belongs {
		return nil, ErrStudentNotFound
	}

	limit := effectiveLimit(params.Limit)

	recs, nextCursor, err := s.recRepo.FindActiveByStudent(ctx, scope, params.StudentID, params.Type, params.Cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("recs: get student recommendations: %w", err)
	}
	return &RecommendationListResponse{
		Recommendations: recsToResponses(recs),
		NextCursor:      nextCursor,
	}, nil
}

// GetPreferences returns the family's recommendation preferences. [13-recs §4]
func (s *RecsServiceImpl) GetPreferences(ctx context.Context, scope *shared.FamilyScope) (*RecommendationPreferencesResponse, error) {
	prefs, err := s.prefRepo.FindOrDefault(ctx, scope)
	if err != nil {
		return nil, fmt.Errorf("recs: get preferences: %w", err)
	}
	return prefsToResponse(prefs), nil
}

// ─── Lifecycle Handlers ───────────────────────────────────────────────────────

// HandleFamilyDeletion deletes all recs data for a family. [13-recs §12]
func (s *RecsServiceImpl) HandleFamilyDeletion(ctx context.Context, familyID shared.FamilyID) error {
	if _, err := s.signalRepo.DeleteByFamily(ctx, familyID); err != nil {
		return fmt.Errorf("recs: family deletion signals: %w", err)
	}
	if _, err := s.recRepo.DeleteByFamily(ctx, familyID); err != nil {
		return fmt.Errorf("recs: family deletion recommendations: %w", err)
	}
	if _, err := s.feedbackRepo.DeleteByFamily(ctx, familyID); err != nil {
		return fmt.Errorf("recs: family deletion feedback: %w", err)
	}
	// recs_preferences has ON DELETE CASCADE from iam_families; also delete explicitly.
	if _, err := s.prefRepo.DeleteByFamily(ctx, familyID); err != nil {
		return fmt.Errorf("recs: family deletion preferences: %w", err)
	}
	return nil
}

// InvalidateMethodologyCache is a no-op in Phase 1 (no in-memory cache yet). [13-recs §12]
func (s *RecsServiceImpl) InvalidateMethodologyCache(_ context.Context) error {
	return nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func effectiveLimit(p *int64) int64 {
	if p == nil {
		return 20
	}
	if *p > 50 {
		return 50
	}
	if *p <= 0 {
		return 20
	}
	return *p
}

func recToResponse(r Recommendation) RecommendationResponse {
	return RecommendationResponse{
		ID:                 r.ID,
		RecommendationType: r.RecommendationType.String(),
		TargetEntityID:     r.TargetEntityID,
		TargetEntityLabel:  r.TargetEntityLabel,
		SourceSignal:       r.SourceSignal.String(),
		SourceLabel:        r.SourceLabel,
		Score:              r.Score,
		IsSuggestion:       true, // always true [S§10.4]
		StudentID:          r.StudentID,
		CreatedAt:          r.CreatedAt,
		ExpiresAt:          r.ExpiresAt,
	}
}

func recsToResponses(recs []Recommendation) []RecommendationResponse {
	out := make([]RecommendationResponse, len(recs))
	for i, r := range recs {
		out[i] = recToResponse(r)
	}
	return out
}

func prefsToResponse(p *Preferences) *RecommendationPreferencesResponse {
	return &RecommendationPreferencesResponse{
		EnabledTypes:         p.EnabledTypes,
		ExplorationFrequency: p.ExplorationFrequency,
	}
}
