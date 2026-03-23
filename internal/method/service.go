package method

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/method/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// methodologyServiceImpl implements MethodologyService.
type methodologyServiceImpl struct {
	defRepo        MethodologyDefinitionRepository
	toolRepo       ToolRepository
	activationRepo ToolActivationRepository
	iamService     IamServiceForMethod
	eventBus       *shared.EventBus
}

// NewMethodologyService creates a new MethodologyService.
// Constructor returns the interface type per [CODING §2.1].
func NewMethodologyService(
	defRepo MethodologyDefinitionRepository,
	toolRepo ToolRepository,
	activationRepo ToolActivationRepository,
	iamService IamServiceForMethod,
	eventBus *shared.EventBus,
) MethodologyService {
	return &methodologyServiceImpl{
		defRepo:        defRepo,
		toolRepo:       toolRepo,
		activationRepo: activationRepo,
		iamService:     iamService,
		eventBus:       eventBus,
	}
}

// ─── Public Queries ──────────────────────────────────────────────────────────

func (s *methodologyServiceImpl) ListMethodologies(ctx context.Context) ([]MethodologySummaryResponse, error) {
	defs, err := s.defRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]MethodologySummaryResponse, len(defs))
	for i := range defs {
		result[i] = toSummaryResponse(&defs[i])
	}
	return result, nil
}

func (s *methodologyServiceImpl) GetMethodology(ctx context.Context, slug string) (*MethodologyDetailResponse, error) {
	def, err := s.defRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return toDetailResponse(def), nil
}

func (s *methodologyServiceImpl) GetMethodologyTools(ctx context.Context, slug string) ([]ActiveToolResponse, error) {
	// Validate slug exists before querying activations (ensures correct 404 behavior).
	if _, err := s.defRepo.FindBySlug(ctx, slug); err != nil {
		return nil, err
	}
	activations, err := s.activationRepo.ListByMethodology(ctx, MethodologyID(slug))
	if err != nil {
		return nil, err
	}
	return activationsToToolResponses(activations), nil
}

func (s *methodologyServiceImpl) ValidateMethodologySlug(ctx context.Context, slug MethodologyID) (bool, error) {
	def, err := s.defRepo.FindBySlug(ctx, string(slug))
	if err != nil {
		return false, nil // not found = not valid
	}
	return def.IsActive, nil
}

func (s *methodologyServiceImpl) GetDefaultMethodologySlug(ctx context.Context) (MethodologyID, error) {
	defs, err := s.defRepo.ListActive(ctx)
	if err != nil {
		return "", err
	}
	if len(defs) == 0 {
		return "", &domain.MethodError{Err: domain.ErrMethodologyNotFound}
	}
	// First active by display_order (ListActive returns ordered)
	return defs[0].Slug, nil
}

// ─── Family-Scoped Queries ───────────────────────────────────────────────────

func (s *methodologyServiceImpl) ResolveFamilyTools(ctx context.Context, scope *shared.FamilyScope) ([]ActiveToolResponse, error) {
	primarySlug, secondarySlugs, err := s.iamService.GetFamilyMethodologyIDs(ctx, scope)
	if err != nil {
		return nil, err
	}
	return s.resolveTools(ctx, primarySlug, secondarySlugs)
}

func (s *methodologyServiceImpl) ResolveStudentTools(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) ([]ActiveToolResponse, error) {
	student, err := s.iamService.GetStudent(ctx, scope, studentID)
	if err != nil {
		return nil, err
	}

	// If student has a methodology override, resolve tools for that methodology only. [S§4.6]
	if student.MethodologyOverrideSlug != nil {
		activations, err := s.activationRepo.ListByMethodology(ctx, *student.MethodologyOverrideSlug)
		if err != nil {
			return nil, err
		}
		return activationsToToolResponses(activations), nil
	}

	// No override — fall through to family tool resolution
	return s.ResolveFamilyTools(ctx, scope)
}

func (s *methodologyServiceImpl) GetMethodologyContext(ctx context.Context, scope *shared.FamilyScope) (*MethodologyContext, error) {
	primarySlug, secondarySlugs, err := s.iamService.GetFamilyMethodologyIDs(ctx, scope)
	if err != nil {
		return nil, err
	}

	allSlugs := append([]MethodologyID{primarySlug}, secondarySlugs...)
	defs, err := s.defRepo.FindBySlugs(ctx, allSlugs)
	if err != nil {
		return nil, err
	}

	defMap := make(map[MethodologyID]*MethodologyDefinition, len(defs))
	for i := range defs {
		defMap[defs[i].Slug] = &defs[i]
	}

	primary, ok := defMap[primarySlug]
	if !ok {
		return nil, &domain.MethodError{Err: domain.ErrMethodologyNotFound, Slug: string(primarySlug)}
	}

	secondary := make([]MethodologySummaryResponse, 0, len(secondarySlugs))
	for _, slug := range secondarySlugs {
		if def, ok := defMap[slug]; ok {
			secondary = append(secondary, toSummaryResponse(def))
		}
	}

	return &MethodologyContext{
		Primary:     toSummaryResponse(primary),
		Secondary:   secondary,
		Terminology: primary.Terminology,
	}, nil
}

// ─── Commands ────────────────────────────────────────────────────────────────

func (s *methodologyServiceImpl) UpdateFamilyMethodology(ctx context.Context, scope *shared.FamilyScope, cmd UpdateMethodologyCommand) (*MethodologySelectionResponse, error) {
	// Validate: primary must not also be in secondary list
	for _, secSlug := range cmd.SecondaryMethodologySlugs {
		if secSlug == cmd.PrimaryMethodologySlug {
			return nil, &domain.MethodError{Err: domain.ErrPrimaryInSecondary, Slug: string(cmd.PrimaryMethodologySlug)}
		}
	}

	// Validate: no duplicate secondary slugs
	seen := make(map[MethodologyID]bool, len(cmd.SecondaryMethodologySlugs))
	for _, slug := range cmd.SecondaryMethodologySlugs {
		if seen[slug] {
			strs := make([]string, len(cmd.SecondaryMethodologySlugs))
			for i, s := range cmd.SecondaryMethodologySlugs {
				strs[i] = string(s)
			}
			return nil, &domain.MethodError{Err: domain.ErrDuplicateSecondary, Slugs: strs}
		}
		seen[slug] = true
	}

	// Validate: all slugs must reference active methodology rows
	allSlugs := append([]MethodologyID{cmd.PrimaryMethodologySlug}, cmd.SecondaryMethodologySlugs...)
	allActive, err := s.defRepo.AllActive(ctx, allSlugs)
	if err != nil {
		return nil, err
	}
	if !allActive {
		strs := make([]string, len(allSlugs))
		for i, s := range allSlugs {
			strs[i] = string(s)
		}
		return nil, &domain.MethodError{Err: domain.ErrInvalidMethodologyIDs, Slugs: strs}
	}

	// Persist via IAM service (bounded context boundary — iam:: owns the columns) [02-method §11.2]
	if err := s.iamService.SetFamilyMethodology(ctx, scope, cmd.PrimaryMethodologySlug, cmd.SecondaryMethodologySlugs); err != nil {
		return nil, err
	}

	// Publish event after persistence succeeds
	if pubErr := s.eventBus.Publish(ctx, FamilyMethodologyChanged{
		FamilyID:                  scope.FamilyID(),
		PrimaryMethodologySlug:    cmd.PrimaryMethodologySlug,
		SecondaryMethodologySlugs: cmd.SecondaryMethodologySlugs,
	}); pubErr != nil {
		slog.Error("failed to publish FamilyMethodologyChanged",
			"family_id", scope.FamilyID(), "error", pubErr)
	}

	// Build response with methodology summaries and tool count
	defs, err := s.defRepo.FindBySlugs(ctx, allSlugs)
	if err != nil {
		return nil, err
	}

	defMap := make(map[MethodologyID]*MethodologyDefinition, len(defs))
	for i := range defs {
		defMap[defs[i].Slug] = &defs[i]
	}

	primary, ok := defMap[cmd.PrimaryMethodologySlug]
	if !ok {
		return nil, &domain.MethodError{Err: domain.ErrMethodologyNotFound, Slug: string(cmd.PrimaryMethodologySlug)}
	}

	secondary := make([]MethodologySummaryResponse, 0, len(cmd.SecondaryMethodologySlugs))
	for _, slug := range cmd.SecondaryMethodologySlugs {
		if def, ok := defMap[slug]; ok {
			secondary = append(secondary, toSummaryResponse(def))
		}
	}

	// Resolve tools to get count
	tools, err := s.resolveTools(ctx, cmd.PrimaryMethodologySlug, cmd.SecondaryMethodologySlugs)
	if err != nil {
		return nil, err
	}

	return &MethodologySelectionResponse{
		Primary:         toSummaryResponse(primary),
		Secondary:       secondary,
		ActiveToolCount: len(tools),
	}, nil
}

func (s *methodologyServiceImpl) UpdateStudentMethodology(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ UpdateStudentMethodologyCommand) (*MethodologySelectionResponse, error) {
	// Phase 2 — not implemented yet
	return nil, &shared.AppError{
		Code:       "not_implemented",
		Message:    "Student methodology override is not yet available",
		StatusCode: 501,
	}
}

// ─── Internal Helpers ────────────────────────────────────────────────────────

// resolveTools loads tool activations for the given methodology slugs and runs
// the ToolResolver deduplication algorithm. [02-method §10.1, §10.2]
func (s *methodologyServiceImpl) resolveTools(ctx context.Context, primarySlug MethodologyID, secondarySlugs []MethodologyID) ([]ActiveToolResponse, error) {
	allSlugs := append([]MethodologyID{primarySlug}, secondarySlugs...)
	activations, err := s.activationRepo.ListByMethodologies(ctx, allSlugs)
	if err != nil {
		return nil, err
	}

	resolver := domain.NewToolResolver(activations, string(primarySlug))
	resolved, err := resolver.Resolve()
	if err != nil {
		return nil, err
	}

	return resolvedToToolResponses(resolved), nil
}

// activationsToToolResponses converts raw activations to API responses (no dedup needed).
func activationsToToolResponses(activations []domain.ToolActivationWithTool) []ActiveToolResponse {
	result := make([]ActiveToolResponse, 0, len(activations))
	for _, a := range activations {
		if !a.ToolIsActive {
			continue
		}
		result = append(result, activationToToolResponse(&a))
	}
	return result
}

// resolvedToToolResponses converts resolved tools to API responses.
func resolvedToToolResponses(resolved []domain.ResolvedTool) []ActiveToolResponse {
	result := make([]ActiveToolResponse, len(resolved))
	for i, r := range resolved {
		result[i] = ActiveToolResponse{
			Slug:            ToolSlug(r.Slug),
			DisplayName:     r.DisplayName,
			Description:     r.Description,
			Tier:            r.Tier,
			Label:           extractLabel(r.ConfigOverrides),
			Guidance:        extractGuidance(r.ConfigOverrides),
			ConfigOverrides: r.ConfigOverrides,
			SortOrder:       r.SortOrder,
		}
	}
	return result
}

func activationToToolResponse(a *domain.ToolActivationWithTool) ActiveToolResponse {
	return ActiveToolResponse{
		Slug:            ToolSlug(a.ToolSlug),
		DisplayName:     a.ToolDisplayName,
		Description:     a.ToolDescription,
		Tier:            a.ToolTier,
		Label:           extractLabel(a.ConfigOverrides),
		Guidance:        extractGuidance(a.ConfigOverrides),
		ConfigOverrides: a.ConfigOverrides,
		SortOrder:       a.SortOrder,
	}
}

// extractLabel extracts the "label" field from config_overrides JSON.
func extractLabel(overrides json.RawMessage) *string {
	if len(overrides) == 0 {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(overrides, &obj); err != nil {
		return nil
	}
	raw, ok := obj["label"]
	if !ok {
		return nil
	}
	var label string
	if err := json.Unmarshal(raw, &label); err != nil {
		return nil
	}
	return &label
}

// extractGuidance extracts the "guidance" field from config_overrides JSON.
func extractGuidance(overrides json.RawMessage) *string {
	if len(overrides) == 0 {
		return nil
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(overrides, &obj); err != nil {
		return nil
	}
	raw, ok := obj["guidance"]
	if !ok {
		return nil
	}
	var guidance string
	if err := json.Unmarshal(raw, &guidance); err != nil {
		return nil
	}
	return &guidance
}
