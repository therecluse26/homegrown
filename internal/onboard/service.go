package onboard

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ─── Service Implementation ──────────────────────────────────────────────────

// onboardingServiceImpl implements OnboardingService.
type onboardingServiceImpl struct {
	wizardRepo     WizardProgressRepository
	roadmapRepo    RoadmapItemRepository
	recRepo        StarterRecommendationRepository
	communityRepo  CommunitySuggestionRepository
	iam            IamServiceForOnboard
	methodology    MethodologyServiceForOnboard
	discovery      DiscoveryServiceForOnboard
	eventBus       *shared.EventBus
	db             *gorm.DB
}

// NewOnboardingService creates a new OnboardingService.
func NewOnboardingService(
	wizardRepo WizardProgressRepository,
	roadmapRepo RoadmapItemRepository,
	recRepo StarterRecommendationRepository,
	communityRepo CommunitySuggestionRepository,
	iam IamServiceForOnboard,
	methodology MethodologyServiceForOnboard,
	discovery DiscoveryServiceForOnboard,
	eventBus *shared.EventBus,
	db *gorm.DB,
) OnboardingService {
	return &onboardingServiceImpl{
		wizardRepo:    wizardRepo,
		roadmapRepo:   roadmapRepo,
		recRepo:       recRepo,
		communityRepo: communityRepo,
		iam:           iam,
		methodology:   methodology,
		discovery:     discovery,
		eventBus:      eventBus,
		db:            db,
	}
}

// ─── GetProgress ─────────────────────────────────────────────────────────────

func (s *onboardingServiceImpl) GetProgress(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error) {
	var progress *WizardProgress
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgWizardProgressRepository{db: tx}
		var findErr error
		progress, findErr = repo.FindByFamilyID(ctx, scope.FamilyID())
		return findErr
	})
	if err != nil {
		return nil, err
	}
	return toProgressResponse(progress), nil
}

// ─── GetRoadmap ──────────────────────────────────────────────────────────────

func (s *onboardingServiceImpl) GetRoadmap(ctx context.Context, scope *shared.FamilyScope) (*RoadmapResponse, error) {
	var items []RoadmapItem
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgRoadmapItemRepository{db: tx}
		var findErr error
		items, findErr = repo.ListByFamilyID(ctx, scope.FamilyID())
		return findErr
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, &OnboardError{Err: ErrNoRoadmapItems}
	}
	// Build flat response items
	flat := make([]RoadmapItemResponse, len(items))
	for i, item := range items {
		flat[i] = RoadmapItemResponse{
			ID:              item.ID,
			MethodologySlug: item.MethodologySlug,
			ItemType:        item.ItemType,
			Title:           item.Title,
			Description:     item.Description,
			LinkURL:         item.LinkURL,
			AgeGroup:        item.AgeGroup,
			SortOrder:       item.SortOrder,
			IsCompleted:     item.IsCompleted,
		}
	}
	buckets := groupByAge(flat, func(r RoadmapItemResponse) *string { return r.AgeGroup })
	groups := make([]RoadmapAgeGroup, len(buckets))
	for i, b := range buckets {
		groups[i] = RoadmapAgeGroup(b)
	}
	return &RoadmapResponse{Groups: groups}, nil
}

// ─── GetRecommendations ──────────────────────────────────────────────────────

func (s *onboardingServiceImpl) GetRecommendations(ctx context.Context, scope *shared.FamilyScope) (*RecommendationsResponse, error) {
	var items []StarterRecommendation
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgStarterRecommendationRepository{db: tx}
		var findErr error
		items, findErr = repo.ListByFamilyID(ctx, scope.FamilyID())
		return findErr
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, &OnboardError{Err: ErrNoRecommendations}
	}
	// Build flat response items
	flat := make([]RecommendationItemResponse, len(items))
	for i, item := range items {
		flat[i] = RecommendationItemResponse{
			ID:              item.ID,
			MethodologySlug: item.MethodologySlug,
			Title:           item.Title,
			Description:     item.Description,
			LinkURL:         item.LinkURL,
			PriceType:       item.PriceType,
			AgeGroup:        item.AgeGroup,
			SortOrder:       item.SortOrder,
		}
	}
	buckets := groupByAge(flat, func(r RecommendationItemResponse) *string { return r.AgeGroup })
	groups := make([]RecommendationAgeGroup, len(buckets))
	for i, b := range buckets {
		groups[i] = RecommendationAgeGroup(b)
	}
	return &RecommendationsResponse{Groups: groups}, nil
}

// ─── GetCommunity ────────────────────────────────────────────────────────────

func (s *onboardingServiceImpl) GetCommunity(ctx context.Context, scope *shared.FamilyScope) (*CommunityResponse, error) {
	var items []CommunitySuggestion
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgCommunitySuggestionRepository{db: tx}
		var findErr error
		items, findErr = repo.ListByFamilyID(ctx, scope.FamilyID())
		return findErr
	})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, &OnboardError{Err: ErrNoCommunity}
	}
	resp := &CommunityResponse{Items: make([]CommunitySuggestionResponse, len(items))}
	for i, item := range items {
		resp.Items[i] = CommunitySuggestionResponse{
			ID:              item.ID,
			MethodologySlug: item.MethodologySlug,
			SuggestionType:  item.SuggestionType,
			Title:           item.Title,
			Description:     item.Description,
			ReferenceID:     item.ReferenceID,
			SortOrder:       item.SortOrder,
		}
	}
	return resp, nil
}

// ─── UpdateFamilyProfile ─────────────────────────────────────────────────────

func (s *onboardingServiceImpl) UpdateFamilyProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyProfileCommand) (*WizardProgressResponse, error) {
	// Delegate profile update to iam::
	if err := s.iam.UpdateFamilyProfile(ctx, scope, cmd); err != nil {
		return nil, err
	}

	// Advance wizard
	var progress *WizardProgress
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgWizardProgressRepository{db: tx}
		var findErr error
		progress, findErr = repo.FindByFamilyID(ctx, scope.FamilyID())
		if findErr != nil {
			return findErr
		}
		if progress.Status != StatusInProgress {
			return &OnboardError{Err: ErrWizardNotInProgress}
		}
		advanceStep(progress, StepFamilyProfile)
		return repo.Update(ctx, progress)
	})
	if err != nil {
		return nil, err
	}
	return toProgressResponse(progress), nil
}

// ─── AddChild ────────────────────────────────────────────────────────────────

func (s *onboardingServiceImpl) AddChild(ctx context.Context, scope *shared.FamilyScope, cmd AddChildCommand) (*WizardProgressResponse, error) {
	// Delegate student creation to iam::
	if err := s.iam.CreateStudent(ctx, scope, cmd); err != nil {
		return nil, err
	}

	// Advance wizard
	var progress *WizardProgress
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgWizardProgressRepository{db: tx}
		var findErr error
		progress, findErr = repo.FindByFamilyID(ctx, scope.FamilyID())
		if findErr != nil {
			return findErr
		}
		if progress.Status != StatusInProgress {
			return &OnboardError{Err: ErrWizardNotInProgress}
		}
		advanceStep(progress, StepChildren)
		return repo.Update(ctx, progress)
	})
	if err != nil {
		return nil, err
	}
	return toProgressResponse(progress), nil
}

// ─── RemoveChild ─────────────────────────────────────────────────────────────

func (s *onboardingServiceImpl) RemoveChild(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error {
	return s.iam.DeleteStudent(ctx, scope, studentID)
}

// ─── SelectMethodology ───────────────────────────────────────────────────────

func (s *onboardingServiceImpl) SelectMethodology(ctx context.Context, scope *shared.FamilyScope, cmd SelectMethodologyCommand) (*WizardProgressResponse, error) {
	// Skip path: auto-assign default methodology, clear secondaries. [04-onboard §9.3]
	if cmd.MethodologyPath == string(PathSkip) {
		defaultSlug, defaultErr := s.methodology.GetDefaultMethodologySlug(ctx)
		if defaultErr != nil {
			return nil, defaultErr
		}
		cmd.PrimaryMethodologySlug = defaultSlug
		cmd.SecondaryMethodologySlugs = nil
	}

	// Validate: secondary methodologies require acknowledgment
	if len(cmd.SecondaryMethodologySlugs) > 0 && !cmd.ExplanationAcknowledged {
		return nil, &OnboardError{Err: ErrSecondaryWithoutAck}
	}

	// Validate all methodology slugs
	allSlugs := append([]string{cmd.PrimaryMethodologySlug}, cmd.SecondaryMethodologySlugs...)
	valid, err := s.methodology.ValidateMethodologySlugs(ctx, allSlugs)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, &OnboardError{Err: ErrInvalidMethodologySlug}
	}

	// Delegate methodology update to method:: → iam::
	if err := s.methodology.UpdateFamilyMethodology(ctx, scope, cmd.PrimaryMethodologySlug, cmd.SecondaryMethodologySlugs); err != nil {
		return nil, err
	}

	// Update wizard progress and trigger materialization
	var progress *WizardProgress
	err = shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgWizardProgressRepository{db: tx}
		var findErr error
		progress, findErr = repo.FindByFamilyID(ctx, scope.FamilyID())
		if findErr != nil {
			return findErr
		}
		if progress.Status != StatusInProgress {
			return &OnboardError{Err: ErrWizardNotInProgress}
		}
		path := cmd.MethodologyPath
		progress.MethodologyPath = &path
		advanceStep(progress, StepMethodology)
		return repo.Update(ctx, progress)
	})
	if err != nil {
		return nil, err
	}

	// Materialize guidance (roadmap, recommendations, community) within a scoped transaction.
	if matErr := s.materializeGuidanceScoped(ctx, scope, progress.ID, cmd.PrimaryMethodologySlug, cmd.SecondaryMethodologySlugs); matErr != nil {
		slog.Error("materialization failed during SelectMethodology", "error", matErr, "family_id", scope.FamilyID())
		// Non-fatal: wizard progress is already updated. Materialization can be retried.
	}

	return toProgressResponse(progress), nil
}

// ─── ImportQuiz ──────────────────────────────────────────────────────────────

func (s *onboardingServiceImpl) ImportQuiz(ctx context.Context, scope *shared.FamilyScope, cmd ImportQuizCommand) (*QuizImportResponse, error) {
	// Retrieve quiz result from discover::
	quizResult, err := s.discovery.GetQuizResult(ctx, cmd.ShareID)
	if err != nil {
		return nil, &OnboardError{Err: ErrQuizResultNotFound}
	}

	// Claim quiz result for this family (graceful degradation — log warning on failure)
	if claimErr := s.discovery.ClaimQuizResult(ctx, cmd.ShareID, scope.FamilyID()); claimErr != nil {
		slog.Warn("failed to claim quiz result", "error", claimErr, "share_id", cmd.ShareID, "family_id", scope.FamilyID())
	}

	// Record quiz_share_id on wizard progress
	err = shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgWizardProgressRepository{db: tx}
		progress, findErr := repo.FindByFamilyID(ctx, scope.FamilyID())
		if findErr != nil {
			return findErr
		}
		if progress.QuizShareID != nil {
			return &OnboardError{Err: ErrQuizAlreadyImported}
		}
		progress.QuizShareID = &cmd.ShareID
		return repo.Update(ctx, progress)
	})
	if err != nil {
		return nil, err
	}

	suggested := ""
	if len(quizResult.Recommendations) > 0 {
		suggested = quizResult.Recommendations[0].MethodologySlug
	}
	return &QuizImportResponse{
		ShareID:                    quizResult.ShareID,
		SuggestedPrimarySlug:       suggested,
		MethodologyRecommendations: quizResult.Recommendations,
	}, nil
}

// ─── CompleteWizard ──────────────────────────────────────────────────────────

// validateCompleteWizard checks that a wizard can be completed.
func validateCompleteWizard(progress *WizardProgress) error {
	if progress.Status == StatusCompleted {
		return &OnboardError{Err: ErrWizardAlreadyCompleted}
	}
	if progress.Status == StatusSkipped {
		return &OnboardError{Err: ErrWizardAlreadySkipped}
	}
	completedSet := make(map[WizardStep]bool, len(progress.CompletedSteps))
	for _, step := range progress.CompletedSteps {
		completedSet[step] = true
	}
	for step := range requiredSteps {
		if !completedSet[step] {
			return &OnboardError{Err: ErrRequiredStepsMissing, Step: string(step)}
		}
	}
	return nil
}

// applyCompleteWizard marks the wizard as completed.
func applyCompleteWizard(progress *WizardProgress) {
	now := time.Now()
	progress.Status = StatusCompleted
	progress.CompletedAt = &now
}

func (s *onboardingServiceImpl) CompleteWizard(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error) {
	var progress *WizardProgress
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgWizardProgressRepository{db: tx}
		var findErr error
		progress, findErr = repo.FindByFamilyID(ctx, scope.FamilyID())
		if findErr != nil {
			return findErr
		}
		if err := validateCompleteWizard(progress); err != nil {
			return err
		}
		applyCompleteWizard(progress)
		return repo.Update(ctx, progress)
	})
	if err != nil {
		return nil, err
	}

	// Publish OnboardingCompleted event
	_ = s.eventBus.Publish(ctx, OnboardingCompleted{
		FamilyID: scope.FamilyID(),
		Skipped:  false,
	})

	return toProgressResponse(progress), nil
}

// ─── SkipWizard ──────────────────────────────────────────────────────────────

// validateSkipWizard checks that a wizard can be skipped.
func validateSkipWizard(progress *WizardProgress) error {
	if progress.Status == StatusCompleted {
		return &OnboardError{Err: ErrWizardAlreadyCompleted}
	}
	if progress.Status == StatusSkipped {
		return &OnboardError{Err: ErrWizardAlreadySkipped}
	}
	return nil
}

// applySkipWizard marks the wizard as skipped.
func applySkipWizard(progress *WizardProgress) {
	now := time.Now()
	progress.Status = StatusSkipped
	progress.CompletedAt = &now
}

func (s *onboardingServiceImpl) SkipWizard(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error) {
	var progress *WizardProgress
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgWizardProgressRepository{db: tx}
		var findErr error
		progress, findErr = repo.FindByFamilyID(ctx, scope.FamilyID())
		if findErr != nil {
			return findErr
		}
		if err := validateSkipWizard(progress); err != nil {
			return err
		}
		applySkipWizard(progress)
		return repo.Update(ctx, progress)
	})
	if err != nil {
		return nil, err
	}

	// Publish OnboardingCompleted event with Skipped=true
	_ = s.eventBus.Publish(ctx, OnboardingCompleted{
		FamilyID: scope.FamilyID(),
		Skipped:  true,
	})

	return toProgressResponse(progress), nil
}

// ─── InitializeWizard (Event-Driven) ─────────────────────────────────────────

func (s *onboardingServiceImpl) InitializeWizard(ctx context.Context, familyID uuid.UUID) error {
	// RLS bypass: called from FamilyCreatedHandler — no auth context available.
	// The family was just created; we must insert the wizard row without a session-scoped FamilyScope.
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		repo := &PgWizardProgressRepository{db: tx}
		return repo.Create(ctx, &WizardProgress{
			FamilyID:       familyID,
			Status:         StatusInProgress,
			CurrentStep:    StepFamilyProfile,
			CompletedSteps: WizardStepArray{},
		})
	})
}

// ─── HandleMethodologyChanged (Event-Driven) ─────────────────────────────────

func (s *onboardingServiceImpl) HandleMethodologyChanged(ctx context.Context, familyID uuid.UUID, primarySlug string, secondarySlugs []string) error {
	// RLS bypass: called from event handler — no auth context available.
	var wizardID uuid.UUID
	var skipMaterialization bool
	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		repo := &PgWizardProgressRepository{db: tx}
		progress, findErr := repo.FindByFamilyID(ctx, familyID)
		if findErr != nil {
			return findErr
		}
		if progress.Status != StatusInProgress {
			skipMaterialization = true
			return nil
		}
		wizardID = progress.ID
		return nil
	})
	if err != nil {
		return err
	}
	if skipMaterialization {
		return nil // wizard complete/skipped — don't re-materialize
	}

	return s.materializeGuidanceUnscoped(ctx, familyID, wizardID, primarySlug, secondarySlugs)
}

// ─── Materialization ─────────────────────────────────────────────────────────

// materializeGuidanceScoped materializes roadmap, recommendations, and community
// within a ScopedTransaction. Used by SelectMethodology (has auth context).
func (s *onboardingServiceImpl) materializeGuidanceScoped(ctx context.Context, scope *shared.FamilyScope, wizardID uuid.UUID, primarySlug string, secondarySlugs []string) error {
	items, err := s.buildMaterializedData(ctx, scope.FamilyID(), wizardID, primarySlug, secondarySlugs)
	if err != nil {
		return err
	}
	return shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		return s.persistMaterializedData(ctx, tx, scope.FamilyID(), items)
	})
}

// materializeGuidanceUnscoped materializes using BypassRLSTransaction.
// Used by HandleMethodologyChanged (event handler, no auth context).
func (s *onboardingServiceImpl) materializeGuidanceUnscoped(ctx context.Context, familyID uuid.UUID, wizardID uuid.UUID, primarySlug string, secondarySlugs []string) error {
	items, err := s.buildMaterializedData(ctx, familyID, wizardID, primarySlug, secondarySlugs)
	if err != nil {
		return err
	}
	// RLS bypass: called from event handler — no auth context available.
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		return s.persistMaterializedData(ctx, tx, familyID, items)
	})
}

type materializedData struct {
	roadmapItems   []RoadmapItem
	recommendations []StarterRecommendation
	community      []CommunitySuggestion
}

// buildMaterializedData fetches methodology configs and students, then builds
// all materialized data structures.
func (s *onboardingServiceImpl) buildMaterializedData(ctx context.Context, familyID uuid.UUID, wizardID uuid.UUID, primarySlug string, secondarySlugs []string) (*materializedData, error) {
	allSlugs := append([]string{primarySlug}, secondarySlugs...)

	// Fetch students for age group resolution
	students, err := s.iam.ListStudents(ctx, familyID)
	if err != nil {
		slog.Warn("failed to list students for materialization", "error", err, "family_id", familyID)
		students = nil // Continue without age filtering
	}

	currentYear := time.Now().Year()
	familyAgeGroups := make(map[AgeGroup]bool)
	for _, student := range students {
		if ag := AgeGroupFromBirthYear(student.BirthYear, currentYear); ag != "" {
			familyAgeGroups[ag] = true
		}
	}

	var result materializedData

	for _, slug := range allSlugs {
		config, configErr := s.methodology.GetMethodology(ctx, slug)
		if configErr != nil {
			slog.Warn("failed to get methodology config", "error", configErr, "slug", slug)
			continue
		}

		// Parse onboarding_config
		var onbConfig onboardingConfigInternal
		if len(config.OnboardingConfig) > 0 {
			if parseErr := json.Unmarshal(config.OnboardingConfig, &onbConfig); parseErr != nil {
				slog.Warn("failed to parse onboarding_config", "error", parseErr, "slug", slug)
			}
		}

		// Parse community_config
		var commConfig communityConfigInternal
		if len(config.CommunityConfig) > 0 {
			if parseErr := json.Unmarshal(config.CommunityConfig, &commConfig); parseErr != nil {
				slog.Warn("failed to parse community_config", "error", parseErr, "slug", slug)
			}
		}

		// Build roadmap items, filtering by family age groups
		for _, step := range onbConfig.RoadmapSteps {
			if !matchesAgeGroup(step.AgeGroup, familyAgeGroups) {
				continue
			}
			desc := step.Description
			link := step.LinkURL
			ageGroup := step.AgeGroup
			item := RoadmapItem{
				FamilyID:        familyID,
				WizardID:        wizardID,
				MethodologySlug: slug,
				ItemType:        RoadmapItemType(step.Type),
				Title:           step.Title,
				SortOrder:       step.SortOrder,
			}
			if desc != "" {
				item.Description = &desc
			}
			if link != "" {
				item.LinkURL = &link
			}
			if ageGroup != "" {
				item.AgeGroup = &ageGroup
			}
			result.roadmapItems = append(result.roadmapItems, item)
		}

		// Build starter recommendations, filtering by age group and capping at 6 per group
		ageGroupCounts := make(map[string]int)
		for _, rec := range onbConfig.StarterRecs {
			if !matchesAgeGroup(rec.AgeGroup, familyAgeGroups) {
				continue
			}
			ag := rec.AgeGroup
			if ag == "" {
				ag = "_all" // universal items counted separately
			}
			if ageGroupCounts[ag] >= 6 {
				continue // max 6 per age group [04-onboard §10.4]
			}
			ageGroupCounts[ag]++

			desc := rec.Description
			link := rec.LinkURL
			ageGroup := rec.AgeGroup
			item := StarterRecommendation{
				FamilyID:        familyID,
				WizardID:        wizardID,
				MethodologySlug: slug,
				Title:           rec.Title,
				PriceType:       rec.PriceType,
				SortOrder:       rec.SortOrder,
			}
			if desc != "" {
				item.Description = &desc
			}
			if link != "" {
				item.LinkURL = &link
			}
			if ageGroup != "" {
				item.AgeGroup = &ageGroup
			}
			result.recommendations = append(result.recommendations, item)
		}

		// Build community suggestions
		for i, group := range commConfig.DefaultGroups {
			desc := group.Description
			item := CommunitySuggestion{
				FamilyID:        familyID,
				WizardID:        wizardID,
				MethodologySlug: slug,
				SuggestionType:  group.Type,
				Title:           group.Title,
				SortOrder:       int16(i),
			}
			if desc != "" {
				item.Description = &desc
			}
			result.community = append(result.community, item)
		}
	}

	return &result, nil
}

// persistMaterializedData atomically deletes old data and inserts new data.
func (s *onboardingServiceImpl) persistMaterializedData(ctx context.Context, tx *gorm.DB, familyID uuid.UUID, data *materializedData) error {
	roadmapRepo := &PgRoadmapItemRepository{db: tx}
	recRepo := &PgStarterRecommendationRepository{db: tx}
	commRepo := &PgCommunitySuggestionRepository{db: tx}

	// Delete existing materialized data (re-materialization)
	if err := roadmapRepo.DeleteByFamilyID(ctx, familyID); err != nil {
		return err
	}
	if err := recRepo.DeleteByFamilyID(ctx, familyID); err != nil {
		return err
	}
	if err := commRepo.DeleteByFamilyID(ctx, familyID); err != nil {
		return err
	}

	// Insert new data
	if err := roadmapRepo.BatchCreate(ctx, data.roadmapItems); err != nil {
		return err
	}
	if err := recRepo.BatchCreate(ctx, data.recommendations); err != nil {
		return err
	}
	if err := commRepo.BatchCreate(ctx, data.community); err != nil {
		return err
	}

	return nil
}

// matchesAgeGroup returns true if the item should be included for the family.
// Items with no age group are always included. If the family has no students, all items are included.
func matchesAgeGroup(itemAgeGroup string, familyAgeGroups map[AgeGroup]bool) bool {
	if itemAgeGroup == "" {
		return true // universal item
	}
	if len(familyAgeGroups) == 0 {
		return true // no students yet — include all items
	}
	return familyAgeGroups[AgeGroup(itemAgeGroup)]
}

// ─── Wizard State Machine ────────────────────────────────────────────────────

// advanceStep adds a step to completed_steps (idempotent) and advances current_step
// to the next step in sequence. [04-onboard §9]
func advanceStep(progress *WizardProgress, completedStep WizardStep) {
	// Idempotent: skip if already completed
	if slices.Contains(progress.CompletedSteps, completedStep) {
		return
	}
	progress.CompletedSteps = append(progress.CompletedSteps, completedStep)
	progress.UpdatedAt = time.Now()

	// Advance current_step to the next incomplete step in sequence
	for _, step := range wizardStepOrder {
		if !slices.Contains(progress.CompletedSteps, step) {
			progress.CurrentStep = step
			return
		}
	}
	// All steps completed — stay on last step
	progress.CurrentStep = wizardStepOrder[len(wizardStepOrder)-1]
}

// ─── Grouping ────────────────────────────────────────────────────────────────

// groupByAge groups items by age bracket. nil age_group ("all ages") is listed first.
// Preserves original item order within each group. [04-onboard §10.3]
func groupByAge[T any](items []T, getAgeGroup func(T) *string) []ageGroupBucket[T] {
	order := make([]string, 0)   // track insertion order; "" = nil/all-ages
	buckets := make(map[string][]T)
	for _, item := range items {
		ag := getAgeGroup(item)
		key := ""
		if ag != nil {
			key = *ag
		}
		if _, seen := buckets[key]; !seen {
			order = append(order, key)
		}
		buckets[key] = append(buckets[key], item)
	}

	// Move "all ages" (key="") to front if present
	allIdx := -1
	for i, k := range order {
		if k == "" {
			allIdx = i
			break
		}
	}
	if allIdx > 0 {
		order = append([]string{""}, append(order[:allIdx], order[allIdx+1:]...)...)
	}

	result := make([]ageGroupBucket[T], len(order))
	for i, key := range order {
		var ag *string
		if key != "" {
			k := key
			ag = &k
		}
		result[i] = ageGroupBucket[T]{AgeGroup: ag, Items: buckets[key]}
	}
	return result
}

type ageGroupBucket[T any] struct {
	AgeGroup *string
	Items    []T
}

// ─── Response Builders ───────────────────────────────────────────────────────

func toProgressResponse(p *WizardProgress) *WizardProgressResponse {
	steps := make([]WizardStep, len(p.CompletedSteps))
	copy(steps, p.CompletedSteps)
	return &WizardProgressResponse{
		ID:              p.ID,
		FamilyID:        p.FamilyID,
		Status:          p.Status,
		CurrentStep:     p.CurrentStep,
		CompletedSteps:  steps,
		QuizShareID:     p.QuizShareID,
		MethodologyPath: p.MethodologyPath,
		CompletedAt:     p.CompletedAt,
		CreatedAt:       p.CreatedAt,
	}
}
