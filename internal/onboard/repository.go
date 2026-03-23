package onboard

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ─── Wizard Progress Repository ──────────────────────────────────────────────

// PgWizardProgressRepository implements WizardProgressRepository using PostgreSQL via GORM.
type PgWizardProgressRepository struct {
	db *gorm.DB
}

func NewPgWizardProgressRepository(db *gorm.DB) WizardProgressRepository {
	return &PgWizardProgressRepository{db: db}
}

func (r *PgWizardProgressRepository) Create(ctx context.Context, progress *WizardProgress) error {
	if err := r.db.WithContext(ctx).Create(progress).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgWizardProgressRepository) FindByFamilyID(ctx context.Context, familyID uuid.UUID) (*WizardProgress, error) {
	var progress WizardProgress
	err := r.db.WithContext(ctx).Where("family_id = ?", familyID).First(&progress).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &OnboardError{Err: ErrWizardNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &progress, nil
}

func (r *PgWizardProgressRepository) Update(ctx context.Context, progress *WizardProgress) error {
	if err := r.db.WithContext(ctx).Save(progress).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Roadmap Item Repository ─────────────────────────────────────────────────

// PgRoadmapItemRepository implements RoadmapItemRepository using PostgreSQL via GORM.
type PgRoadmapItemRepository struct {
	db *gorm.DB
}

func NewPgRoadmapItemRepository(db *gorm.DB) RoadmapItemRepository {
	return &PgRoadmapItemRepository{db: db}
}

func (r *PgRoadmapItemRepository) ListByFamilyID(ctx context.Context, familyID uuid.UUID) ([]RoadmapItem, error) {
	var items []RoadmapItem
	err := r.db.WithContext(ctx).
		Where("family_id = ?", familyID).
		Order("sort_order ASC").
		Find(&items).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return items, nil
}

func (r *PgRoadmapItemRepository) DeleteByFamilyID(ctx context.Context, familyID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).Delete(&RoadmapItem{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgRoadmapItemRepository) BatchCreate(ctx context.Context, items []RoadmapItem) error {
	if len(items) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&items).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Starter Recommendation Repository ───────────────────────────────────────

// PgStarterRecommendationRepository implements StarterRecommendationRepository.
type PgStarterRecommendationRepository struct {
	db *gorm.DB
}

func NewPgStarterRecommendationRepository(db *gorm.DB) StarterRecommendationRepository {
	return &PgStarterRecommendationRepository{db: db}
}

func (r *PgStarterRecommendationRepository) ListByFamilyID(ctx context.Context, familyID uuid.UUID) ([]StarterRecommendation, error) {
	var items []StarterRecommendation
	err := r.db.WithContext(ctx).
		Where("family_id = ?", familyID).
		Order("sort_order ASC").
		Find(&items).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return items, nil
}

func (r *PgStarterRecommendationRepository) DeleteByFamilyID(ctx context.Context, familyID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).Delete(&StarterRecommendation{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgStarterRecommendationRepository) BatchCreate(ctx context.Context, items []StarterRecommendation) error {
	if len(items) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&items).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Community Suggestion Repository ─────────────────────────────────────────

// PgCommunitySuggestionRepository implements CommunitySuggestionRepository.
type PgCommunitySuggestionRepository struct {
	db *gorm.DB
}

func NewPgCommunitySuggestionRepository(db *gorm.DB) CommunitySuggestionRepository {
	return &PgCommunitySuggestionRepository{db: db}
}

func (r *PgCommunitySuggestionRepository) ListByFamilyID(ctx context.Context, familyID uuid.UUID) ([]CommunitySuggestion, error) {
	var items []CommunitySuggestion
	err := r.db.WithContext(ctx).
		Where("family_id = ?", familyID).
		Order("sort_order ASC").
		Find(&items).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return items, nil
}

func (r *PgCommunitySuggestionRepository) DeleteByFamilyID(ctx context.Context, familyID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Where("family_id = ?", familyID).Delete(&CommunitySuggestion{}).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgCommunitySuggestionRepository) BatchCreate(ctx context.Context, items []CommunitySuggestion) error {
	if len(items) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).Create(&items).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}
