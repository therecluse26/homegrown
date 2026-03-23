package method

import (
	"context"
	"errors"

	"github.com/homegrown-academy/homegrown-academy/internal/method/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ─── Methodology Definition Repository ───────────────────────────────────────

// PgMethodologyDefinitionRepository implements MethodologyDefinitionRepository using PostgreSQL via GORM.
// NOT family-scoped — method_ tables are global config, no RLS. [02-method §3.2]
type PgMethodologyDefinitionRepository struct {
	db *gorm.DB
}

// NewPgMethodologyDefinitionRepository creates a new PgMethodologyDefinitionRepository.
func NewPgMethodologyDefinitionRepository(db *gorm.DB) MethodologyDefinitionRepository {
	return &PgMethodologyDefinitionRepository{db: db}
}

func (r *PgMethodologyDefinitionRepository) ListActive(ctx context.Context) ([]MethodologyDefinition, error) {
	var defs []MethodologyDefinition
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("display_order ASC").
		Find(&defs).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return defs, nil
}

func (r *PgMethodologyDefinitionRepository) FindBySlug(ctx context.Context, slug string) (*MethodologyDefinition, error) {
	var def MethodologyDefinition
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&def).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &domain.MethodError{Err: domain.ErrMethodologyNotFound, Slug: slug}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &def, nil
}

func (r *PgMethodologyDefinitionRepository) FindBySlugs(ctx context.Context, slugs []MethodologyID) ([]MethodologyDefinition, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	strs := make([]string, len(slugs))
	for i, s := range slugs {
		strs[i] = string(s)
	}
	var defs []MethodologyDefinition
	if err := r.db.WithContext(ctx).Where("slug IN ?", strs).Find(&defs).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return defs, nil
}

func (r *PgMethodologyDefinitionRepository) AllActive(ctx context.Context, slugs []MethodologyID) (bool, error) {
	if len(slugs) == 0 {
		return true, nil
	}
	strs := make([]string, len(slugs))
	for i, s := range slugs {
		strs[i] = string(s)
	}
	var count int64
	if err := r.db.WithContext(ctx).Model(&MethodologyDefinition{}).
		Where("slug IN ? AND is_active = ?", strs, true).
		Count(&count).Error; err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count == int64(len(slugs)), nil
}

// ─── Tool Repository ──────────────────────────────────────────────────────────

// PgToolRepository implements ToolRepository using PostgreSQL via GORM.
// NOT family-scoped — global catalog. [02-method §3.2]
type PgToolRepository struct {
	db *gorm.DB
}

// NewPgToolRepository creates a new PgToolRepository.
func NewPgToolRepository(db *gorm.DB) ToolRepository {
	return &PgToolRepository{db: db}
}

func (r *PgToolRepository) ListActive(ctx context.Context) ([]Tool, error) {
	var tools []Tool
	if err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&tools).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return tools, nil
}

func (r *PgToolRepository) FindBySlug(ctx context.Context, slug string) (*Tool, error) {
	var tool Tool
	err := r.db.WithContext(ctx).Where("slug = ?", slug).First(&tool).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &domain.MethodError{Err: domain.ErrToolNotFound, Slug: slug}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &tool, nil
}

// ─── Tool Activation Repository ──────────────────────────────────────────────

// PgToolActivationRepository implements ToolActivationRepository using PostgreSQL via GORM.
// Uses raw SQL with JOIN for the activation+tool query. NOT family-scoped. [02-method §3.2]
type PgToolActivationRepository struct {
	db *gorm.DB
}

// NewPgToolActivationRepository creates a new PgToolActivationRepository.
func NewPgToolActivationRepository(db *gorm.DB) ToolActivationRepository {
	return &PgToolActivationRepository{db: db}
}

func (r *PgToolActivationRepository) ListByMethodology(ctx context.Context, slug MethodologyID) ([]domain.ToolActivationWithTool, error) {
	return r.listBySlugs(ctx, []MethodologyID{slug})
}

func (r *PgToolActivationRepository) ListByMethodologies(ctx context.Context, slugs []MethodologyID) ([]domain.ToolActivationWithTool, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	return r.listBySlugs(ctx, slugs)
}

func (r *PgToolActivationRepository) listBySlugs(ctx context.Context, slugs []MethodologyID) ([]domain.ToolActivationWithTool, error) {
	strs := make([]string, len(slugs))
	for i, s := range slugs {
		strs[i] = string(s)
	}
	var results []domain.ToolActivationWithTool
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			a.methodology_slug,
			a.tool_slug,
			t.display_name AS tool_display_name,
			t.description  AS tool_description,
			t.tier         AS tool_tier,
			t.is_active    AS tool_is_active,
			a.config_overrides,
			a.sort_order
		FROM method_tool_activations a
		JOIN method_tools t ON t.slug = a.tool_slug
		WHERE a.methodology_slug IN ?
		ORDER BY a.sort_order ASC
	`, strs).Scan(&results).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return results, nil
}
