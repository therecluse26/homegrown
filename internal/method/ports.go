package method

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/method/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Service Interface ────────────────────────────────────────────────────────

// MethodologyService defines all use cases for methodology configuration.
// Defined here per [CODING §8.2]. Implementation: methodologyServiceImpl in service.go.
type MethodologyService interface {
	// ─── Public Queries (no auth required) ──────────────────────────────

	// ListMethodologies lists all active methodologies, ordered by display_order.
	// Used by GET /v1/methodologies and consumed by onboard:: and discover::.
	ListMethodologies(ctx context.Context) ([]MethodologySummaryResponse, error)

	// GetMethodology returns full methodology detail by slug, including philosophy module.
	// Used by GET /v1/methodologies/:slug.
	GetMethodology(ctx context.Context, slug string) (*MethodologyDetailResponse, error)

	// GetMethodologyTools returns tools activated for a specific methodology.
	// Used by GET /v1/methodologies/:slug/tools.
	GetMethodologyTools(ctx context.Context, slug string) ([]ActiveToolResponse, error)

	// ValidateMethodologySlug validates that a methodology slug exists and is active.
	// Used by iam:: during registration to validate default methodology.
	ValidateMethodologySlug(ctx context.Context, slug MethodologyID) (bool, error)

	// GetDefaultMethodologySlug returns the default methodology slug (first active by display_order).
	// Used by iam:: during registration when no methodology is specified.
	GetDefaultMethodologySlug(ctx context.Context) (MethodologyID, error)

	// ─── Family-Scoped Queries (auth required) ─────────────────────────

	// ResolveFamilyTools resolves the family's active tool set (union of all methodology tools,
	// deduplicated). Used by GET /v1/families/tools and consumed by learn::. [S§4.2]
	ResolveFamilyTools(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error)

	// ResolveStudentTools resolves a student's active tool set, considering methodology overrides.
	// Used by GET /v1/families/students/:id/tools. [S§4.6]
	ResolveStudentTools(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error)

	// GetMethodologyContext returns full methodology context for the family dashboard.
	// Used by GET /v1/families/methodology-context (Phase 2). [ARCH §7.2]
	GetMethodologyContext(ctx context.Context, scope *shared.FamilyScope) (*MethodologyContext, error)

	// ─── Commands ───────────────────────────────────────────────────────

	// UpdateFamilyMethodology updates the family's methodology selection (primary + secondary).
	// Validates all slugs, delegates persistence to iam::FamilyRepository.
	// Publishes FamilyMethodologyChanged. [S§4.3]
	UpdateFamilyMethodology(ctx context.Context, scope *shared.FamilyScope, cmd UpdateMethodologyCommand) (*MethodologySelectionResponse, error)

	// UpdateStudentMethodology sets or clears a student's methodology override.
	// Delegates persistence to iam::StudentRepository. [S§4.6] (Phase 2)
	UpdateStudentMethodology(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, cmd UpdateStudentMethodologyCommand) (*MethodologySelectionResponse, error)
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

// MethodologyDefinitionRepository provides access to methodology definitions.
// NOT family-scoped — method_ tables contain global platform config. [CODING §2.4, §8.2]
type MethodologyDefinitionRepository interface {
	// ListActive lists all active methodologies, ordered by display_order.
	ListActive(ctx context.Context) ([]MethodologyDefinition, error)

	// FindBySlug finds a methodology by slug.
	FindBySlug(ctx context.Context, slug string) (*MethodologyDefinition, error)

	// FindBySlugs finds multiple methodologies by slugs.
	// Used by tool resolution to batch-load primary + secondary methodologies.
	FindBySlugs(ctx context.Context, slugs []MethodologyID) ([]MethodologyDefinition, error)

	// AllActive checks that all provided slugs reference active methodology rows.
	// Returns true only if ALL slugs are valid and active.
	AllActive(ctx context.Context, slugs []MethodologyID) (bool, error)
}

// ToolRepository provides access to the master tool catalog.
// NOT family-scoped — global catalog. [CODING §8.2]
type ToolRepository interface {
	// ListActive lists all active tools.
	ListActive(ctx context.Context) ([]Tool, error)

	// FindBySlug finds a tool by slug.
	FindBySlug(ctx context.Context, slug string) (*Tool, error)
}

// ToolActivationRepository provides access to per-methodology tool activations.
// NOT family-scoped — global config. [CODING §8.2]
type ToolActivationRepository interface {
	// ListByMethodology lists all tool activations for a methodology, ordered by sort_order.
	// Joins with method_tools to include tool metadata.
	ListByMethodology(ctx context.Context, slug MethodologyID) ([]domain.ToolActivationWithTool, error)

	// ListByMethodologies lists all tool activations for multiple methodologies.
	// Used by tool resolution (union across primary + secondary).
	ListByMethodologies(ctx context.Context, slugs []MethodologyID) ([]domain.ToolActivationWithTool, error)
}

// ─── IAM Service Interface (consumed by method::) ─────────────────────────────

// IamServiceForMethod defines the subset of IamService methods that method:: needs.
// This avoids importing the full iam package. [ARCH §4.2]
type IamServiceForMethod interface {
	// GetFamilyMethodologyIDs returns the family's primary and secondary methodology slugs.
	GetFamilyMethodologyIDs(ctx context.Context, scope *shared.FamilyScope) (primary MethodologyID, secondary []MethodologyID, err error)

	// GetStudent returns a single student by ID. Used for student tool resolution.
	GetStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) (*StudentInfo, error)

	// SetFamilyMethodology persists the family's methodology selection.
	SetFamilyMethodology(ctx context.Context, scope *shared.FamilyScope, primarySlug MethodologyID, secondarySlugs []MethodologyID) error
}

// StudentInfo is the minimal student data needed by method:: for tool resolution.
type StudentInfo struct {
	ID                      uuid.UUID
	MethodologyOverrideSlug *MethodologyID
}
