package recs

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Stub Repositories [13-recs plan mock pattern]
// ═══════════════════════════════════════════════════════════════════════════════

// ─── stubSignalRepo ───────────────────────────────────────────────────────────

type stubSignalRepo struct {
	createFn         func(ctx context.Context, signal NewSignal) error
	findByFamilyFn   func(ctx context.Context, scope *shared.FamilyScope, since time.Time) ([]Signal, error)
	deleteByFamilyFn func(ctx context.Context, familyID shared.FamilyID) (int64, error)
	deleteStaleFn    func(ctx context.Context, before time.Time) (int64, error)
}

func (s *stubSignalRepo) Create(ctx context.Context, signal NewSignal) error {
	if s.createFn != nil {
		return s.createFn(ctx, signal)
	}
	panic("stubSignalRepo.Create not stubbed")
}

func (s *stubSignalRepo) FindByFamily(ctx context.Context, scope *shared.FamilyScope, since time.Time) ([]Signal, error) {
	if s.findByFamilyFn != nil {
		return s.findByFamilyFn(ctx, scope, since)
	}
	panic("stubSignalRepo.FindByFamily not stubbed")
}

func (s *stubSignalRepo) DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error) {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	panic("stubSignalRepo.DeleteByFamily not stubbed")
}

func (s *stubSignalRepo) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	if s.deleteStaleFn != nil {
		return s.deleteStaleFn(ctx, before)
	}
	panic("stubSignalRepo.DeleteStale not stubbed")
}

// ─── stubRecommendationRepo ───────────────────────────────────────────────────

type stubRecommendationRepo struct {
	createBatchFn         func(ctx context.Context, recs []NewRecommendation) (int64, error)
	findByIDFn            func(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*Recommendation, error)
	findActiveByFamilyFn  func(ctx context.Context, scope *shared.FamilyScope, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error)
	findActiveByStudentFn func(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error)
	updateStatusFn        func(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID, status string) error
	expireStaleFn         func(ctx context.Context) (int64, error)
	deleteByFamilyFn      func(ctx context.Context, familyID shared.FamilyID) (int64, error)
}

func (s *stubRecommendationRepo) CreateBatch(ctx context.Context, recs []NewRecommendation) (int64, error) {
	if s.createBatchFn != nil {
		return s.createBatchFn(ctx, recs)
	}
	panic("stubRecommendationRepo.CreateBatch not stubbed")
}

func (s *stubRecommendationRepo) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*Recommendation, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(ctx, scope, id)
	}
	panic("stubRecommendationRepo.FindByID not stubbed")
}

func (s *stubRecommendationRepo) FindActiveByFamily(ctx context.Context, scope *shared.FamilyScope, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error) {
	if s.findActiveByFamilyFn != nil {
		return s.findActiveByFamilyFn(ctx, scope, recommendationType, cursor, limit)
	}
	panic("stubRecommendationRepo.FindActiveByFamily not stubbed")
}

func (s *stubRecommendationRepo) FindActiveByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error) {
	if s.findActiveByStudentFn != nil {
		return s.findActiveByStudentFn(ctx, scope, studentID, recommendationType, cursor, limit)
	}
	panic("stubRecommendationRepo.FindActiveByStudent not stubbed")
}

func (s *stubRecommendationRepo) UpdateStatus(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID, status string) error {
	if s.updateStatusFn != nil {
		return s.updateStatusFn(ctx, scope, recommendationID, status)
	}
	panic("stubRecommendationRepo.UpdateStatus not stubbed")
}

func (s *stubRecommendationRepo) ExpireStale(ctx context.Context) (int64, error) {
	if s.expireStaleFn != nil {
		return s.expireStaleFn(ctx)
	}
	panic("stubRecommendationRepo.ExpireStale not stubbed")
}

func (s *stubRecommendationRepo) DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error) {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	panic("stubRecommendationRepo.DeleteByFamily not stubbed")
}

// ─── stubFeedbackRepo ─────────────────────────────────────────────────────────

type stubFeedbackRepo struct {
	createFn               func(ctx context.Context, feedback NewFeedback) error
	findByRecommendationFn func(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) (*Feedback, error)
	findBlockedByFamilyFn  func(ctx context.Context, scope *shared.FamilyScope) ([]uuid.UUID, error)
	deleteFn               func(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) error
	deleteByFamilyFn       func(ctx context.Context, familyID shared.FamilyID) (int64, error)
}

func (s *stubFeedbackRepo) Create(ctx context.Context, feedback NewFeedback) error {
	if s.createFn != nil {
		return s.createFn(ctx, feedback)
	}
	panic("stubFeedbackRepo.Create not stubbed")
}

func (s *stubFeedbackRepo) FindByRecommendation(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) (*Feedback, error) {
	if s.findByRecommendationFn != nil {
		return s.findByRecommendationFn(ctx, scope, recommendationID)
	}
	panic("stubFeedbackRepo.FindByRecommendation not stubbed")
}

func (s *stubFeedbackRepo) FindBlockedByFamily(ctx context.Context, scope *shared.FamilyScope) ([]uuid.UUID, error) {
	if s.findBlockedByFamilyFn != nil {
		return s.findBlockedByFamilyFn(ctx, scope)
	}
	panic("stubFeedbackRepo.FindBlockedByFamily not stubbed")
}

func (s *stubFeedbackRepo) Delete(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) error {
	if s.deleteFn != nil {
		return s.deleteFn(ctx, scope, recommendationID)
	}
	panic("stubFeedbackRepo.Delete not stubbed")
}

func (s *stubFeedbackRepo) DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error) {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	panic("stubFeedbackRepo.DeleteByFamily not stubbed")
}

// ─── stubPopularityRepo ───────────────────────────────────────────────────────

type stubPopularityRepo struct {
	upsertFn            func(ctx context.Context, score NewPopularityScore) error
	findByMethodologyFn func(ctx context.Context, methodologySlug string, limit int64) ([]PopularityScore, error)
	deleteStaleFn       func(ctx context.Context, before time.Time) (int64, error)
}

func (s *stubPopularityRepo) Upsert(ctx context.Context, score NewPopularityScore) error {
	if s.upsertFn != nil {
		return s.upsertFn(ctx, score)
	}
	panic("stubPopularityRepo.Upsert not stubbed")
}

func (s *stubPopularityRepo) FindByMethodology(ctx context.Context, methodologySlug string, limit int64) ([]PopularityScore, error) {
	if s.findByMethodologyFn != nil {
		return s.findByMethodologyFn(ctx, methodologySlug, limit)
	}
	panic("stubPopularityRepo.FindByMethodology not stubbed")
}

func (s *stubPopularityRepo) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	if s.deleteStaleFn != nil {
		return s.deleteStaleFn(ctx, before)
	}
	panic("stubPopularityRepo.DeleteStale not stubbed")
}

// ─── stubPreferenceRepo ───────────────────────────────────────────────────────

type stubPreferenceRepo struct {
	findOrDefaultFn  func(ctx context.Context, scope *shared.FamilyScope) (*Preferences, error)
	upsertFn         func(ctx context.Context, scope *shared.FamilyScope, preferences UpdatePreferences) (*Preferences, error)
	deleteByFamilyFn func(ctx context.Context, familyID shared.FamilyID) (int64, error)
}

func (s *stubPreferenceRepo) FindOrDefault(ctx context.Context, scope *shared.FamilyScope) (*Preferences, error) {
	if s.findOrDefaultFn != nil {
		return s.findOrDefaultFn(ctx, scope)
	}
	panic("stubPreferenceRepo.FindOrDefault not stubbed")
}

func (s *stubPreferenceRepo) Upsert(ctx context.Context, scope *shared.FamilyScope, preferences UpdatePreferences) (*Preferences, error) {
	if s.upsertFn != nil {
		return s.upsertFn(ctx, scope, preferences)
	}
	panic("stubPreferenceRepo.Upsert not stubbed")
}

func (s *stubPreferenceRepo) DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error) {
	if s.deleteByFamilyFn != nil {
		return s.deleteByFamilyFn(ctx, familyID)
	}
	return 0, nil // safe default for tests that don't need this
}

// ─── stubAnonRepo ─────────────────────────────────────────────────────────────

type stubAnonRepo struct {
	createBatchFn func(ctx context.Context, interactions []NewAnonymizedInteraction) (int64, error)
}

func (s *stubAnonRepo) CreateBatch(ctx context.Context, interactions []NewAnonymizedInteraction) (int64, error) {
	if s.createBatchFn != nil {
		return s.createBatchFn(ctx, interactions)
	}
	panic("stubAnonRepo.CreateBatch not stubbed")
}

// ─── stubIamService ───────────────────────────────────────────────────────────

type stubIamService struct {
	studentBelongsToFamilyFn     func(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error)
	getFamilyMethodologySlugFn   func(ctx context.Context, familyID shared.FamilyID) (string, error)
}

func (s *stubIamService) StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error) {
	if s.studentBelongsToFamilyFn != nil {
		return s.studentBelongsToFamilyFn(ctx, studentID, familyID)
	}
	panic("stubIamService.StudentBelongsToFamily not stubbed")
}

func (s *stubIamService) GetFamilyMethodologySlug(ctx context.Context, familyID shared.FamilyID) (string, error) {
	if s.getFamilyMethodologySlugFn != nil {
		return s.getFamilyMethodologySlugFn(ctx, familyID)
	}
	// Default: return empty string (non-fatal fallback in service).
	return "", nil
}

// ─── Test Helpers ─────────────────────────────────────────────────────────────

func testScope() *shared.FamilyScope {
	auth := &shared.AuthContext{
		FamilyID: uuid.Must(uuid.NewV7()),
	}
	s := shared.NewFamilyScopeFromAuth(auth)
	return &s
}

func newTestService(
	signalRepo SignalRepository,
	recRepo RecommendationRepository,
	feedbackRepo FeedbackRepository,
	popularityRepo PopularityRepository,
	prefRepo PreferenceRepository,
	anonRepo AnonymizedInteractionRepository,
	iamSvc IamServiceForRecs,
) RecsService {
	return NewRecsService(signalRepo, recRepo, feedbackRepo, popularityRepo, prefRepo, anonRepo, iamSvc)
}
