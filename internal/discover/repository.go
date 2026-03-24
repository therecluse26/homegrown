package discover

import (
	"context"
	"errors"
	"strings"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ─── Quiz Definition Repository ───────────────────────────────────────────────

// PgQuizDefinitionRepository implements QuizDefinitionRepository using PostgreSQL via GORM.
// NOT family-scoped — disc_quiz_definitions is global platform config. [03-discover §3.2]
type PgQuizDefinitionRepository struct {
	db *gorm.DB
}

// NewPgQuizDefinitionRepository creates a new PgQuizDefinitionRepository.
func NewPgQuizDefinitionRepository(db *gorm.DB) QuizDefinitionRepository {
	return &PgQuizDefinitionRepository{db: db}
}

func (r *PgQuizDefinitionRepository) FindActive(ctx context.Context) (*QuizDefinition, error) {
	var def QuizDefinition
	err := r.db.WithContext(ctx).Where("status = ?", "active").First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &DiscoverError{Err: ErrNoActiveQuiz}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &def, nil
}

func (r *PgQuizDefinitionRepository) FindByID(ctx context.Context, id any) (*QuizDefinition, error) {
	var def QuizDefinition
	err := r.db.WithContext(ctx).First(&def, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &DiscoverError{Err: ErrNoActiveQuiz}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &def, nil
}

// ─── Quiz Result Repository ───────────────────────────────────────────────────

// PgQuizResultRepository implements QuizResultRepository using PostgreSQL via GORM.
// NOT family-scoped — results are anonymous until claimed. [03-discover §3.2]
type PgQuizResultRepository struct {
	db *gorm.DB
}

// NewPgQuizResultRepository creates a new PgQuizResultRepository.
func NewPgQuizResultRepository(db *gorm.DB) QuizResultRepository {
	return &PgQuizResultRepository{db: db}
}

func (r *PgQuizResultRepository) Create(ctx context.Context, input CreateQuizResult) (*QuizResult, error) {
	result := QuizResult{
		QuizDefinitionID: input.QuizDefinitionID,
		ShareID:          input.ShareID,
		SessionToken:     input.SessionToken,
		Answers:          input.Answers,
		Scores:           input.Scores,
		Recommendations:  input.Recommendations,
		QuizVersion:      input.QuizVersion,
	}
	if err := r.db.WithContext(ctx).Create(&result).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return &result, nil
}

func (r *PgQuizResultRepository) FindByShareID(ctx context.Context, shareID string) (*QuizResult, error) {
	var result QuizResult
	err := r.db.WithContext(ctx).Where("share_id = ?", shareID).First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &DiscoverError{Err: ErrQuizResultNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &result, nil
}

func (r *PgQuizResultRepository) ClaimForFamily(ctx context.Context, shareID string, familyID any) error {
	// Only update if family_id is currently NULL (anonymous) or already owned by this family.
	// If owned by a different family, the conditional WHERE prevents the update (0 rows affected = conflict).
	result := r.db.WithContext(ctx).
		Model(&QuizResult{}).
		Where("share_id = ? AND (family_id IS NULL OR family_id = ?)", shareID, familyID).
		Update("family_id", familyID)
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		// Either not found or already claimed by another family.
		// Check which case it is.
		var count int64
		if err := r.db.WithContext(ctx).
			Model(&QuizResult{}).
			Where("share_id = ?", shareID).
			Count(&count).Error; err != nil {
			return shared.ErrDatabase(err)
		}
		if count == 0 {
			return &DiscoverError{Err: ErrQuizResultNotFound}
		}
		return &DiscoverError{Err: ErrQuizResultAlreadyClaimed}
	}
	return nil
}

// ─── Content Page Repository ──────────────────────────────────────────────────

// PgContentPageRepository implements ContentPageRepository using PostgreSQL via GORM.
// NOT family-scoped — disc_content_pages is global platform content. [03-discover §6.4]
type PgContentPageRepository struct {
	db *gorm.DB
}

// NewPgContentPageRepository creates a new PgContentPageRepository.
func NewPgContentPageRepository(db *gorm.DB) ContentPageRepository {
	return &PgContentPageRepository{db: db}
}

func (r *PgContentPageRepository) FindBySlug(ctx context.Context, slug string) (*ContentPage, error) {
	var page ContentPage
	err := r.db.WithContext(ctx).
		Where("slug = ? AND status = ?", slug, "published").
		First(&page).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &DiscoverError{Err: ErrContentPageNotFound, Slug: slug}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &page, nil
}

func (r *PgContentPageRepository) ListPublished(ctx context.Context) ([]ContentPageSummary, error) {
	var summaries []ContentPageSummary
	err := r.db.WithContext(ctx).
		Model(&ContentPage{}).
		Select("slug, category, title").
		Where("status = ?", "published").
		Order("category ASC, display_order ASC").
		Scan(&summaries).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return summaries, nil
}

// ─── State Guide Repository ───────────────────────────────────────────────────

// PgStateGuideRepository implements StateGuideRepository using PostgreSQL via GORM.
// NOT family-scoped — disc_state_guides is global platform config. [03-discover §3.2]
type PgStateGuideRepository struct {
	db *gorm.DB
}

// NewPgStateGuideRepository creates a new PgStateGuideRepository.
func NewPgStateGuideRepository(db *gorm.DB) StateGuideRepository {
	return &PgStateGuideRepository{db: db}
}

func (r *PgStateGuideRepository) ListAll(ctx context.Context) ([]StateGuideSummary, error) {
	var summaries []StateGuideSummary
	err := r.db.WithContext(ctx).
		Model(&StateGuide{}).
		Select("state_code, state_name, status, last_reviewed_at").
		Order("state_name ASC").
		Scan(&summaries).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return summaries, nil
}

func (r *PgStateGuideRepository) FindByStateCode(ctx context.Context, stateCode string) (*StateGuide, error) {
	var guide StateGuide
	// Normalize to uppercase so 'ca' and 'CA' both match. [03-discover §8.3]
	err := r.db.WithContext(ctx).
		Where("state_code = ?", strings.ToUpper(stateCode)).
		First(&guide).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &DiscoverError{Err: ErrStateGuideNotFound, StateCode: stateCode}
		}
		return nil, shared.ErrDatabase(err)
	}
	// Return the row but signal not-published separately so the service can
	// emit the correct 404 (not 403). [03-discover §3.2, §15.16]
	if guide.Status != "published" {
		return nil, &DiscoverError{Err: ErrStateGuideNotPublished, StateCode: stateCode}
	}
	return &guide, nil
}

// FindRequirementsByStateCode returns a state guide regardless of published status.
// Used by GetStateRequirements() so comply:: can act on requirements before guides are published.
// Returns &DiscoverError{Err: ErrStateGuideNotFound} if no row matches. [03-discover §5, §13.1]
func (r *PgStateGuideRepository) FindRequirementsByStateCode(ctx context.Context, stateCode string) (*StateGuide, error) {
	var guide StateGuide
	err := r.db.WithContext(ctx).
		Where("state_code = ?", strings.ToUpper(stateCode)).
		First(&guide).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &DiscoverError{Err: ErrStateGuideNotFound, StateCode: stateCode}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &guide, nil
}
