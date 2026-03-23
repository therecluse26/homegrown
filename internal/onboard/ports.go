package onboard

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Service Interface ───────────────────────────────────────────────────────

// OnboardingService defines all Phase 1 use cases for the onboarding domain.
// All methods are family-scoped (auth required) except InitializeWizard. [04-onboard §1]
// Implementation: onboardingServiceImpl in service.go.
type OnboardingService interface {
	// ─── Queries ──────────────────────────────────────────────────────────────

	// GetProgress returns the current wizard progress for the family.
	GetProgress(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error)

	// GetRoadmap returns materialized roadmap items for the family.
	GetRoadmap(ctx context.Context, scope *shared.FamilyScope) (*RoadmapResponse, error)

	// GetRecommendations returns materialized starter recommendations for the family.
	GetRecommendations(ctx context.Context, scope *shared.FamilyScope) (*RecommendationsResponse, error)

	// GetCommunity returns materialized community suggestions for the family.
	GetCommunity(ctx context.Context, scope *shared.FamilyScope) (*CommunityResponse, error)

	// ─── Commands ─────────────────────────────────────────────────────────────

	// UpdateFamilyProfile delegates to iam::UpdateFamilyProfile and advances wizard. [04-onboard §9.1]
	UpdateFamilyProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyProfileCommand) (*WizardProgressResponse, error)

	// AddChild creates a student via iam::CreateStudent and advances wizard. [04-onboard §9.2]
	AddChild(ctx context.Context, scope *shared.FamilyScope, cmd AddChildCommand) (*WizardProgressResponse, error)

	// RemoveChild deletes a student via iam::DeleteStudent. [04-onboard §9.2]
	RemoveChild(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error

	// SelectMethodology validates and persists methodology selection, triggers materialization. [04-onboard §9.3]
	SelectMethodology(ctx context.Context, scope *shared.FamilyScope, cmd SelectMethodologyCommand) (*WizardProgressResponse, error)

	// ImportQuiz imports a pre-completion quiz result into the wizard. [04-onboard §9.4]
	ImportQuiz(ctx context.Context, scope *shared.FamilyScope, cmd ImportQuizCommand) (*QuizImportResponse, error)

	// CompleteWizard marks the wizard as completed if all required steps are done. [04-onboard §9.5]
	CompleteWizard(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error)

	// SkipWizard marks the wizard as skipped. [04-onboard §9.6]
	SkipWizard(ctx context.Context, scope *shared.FamilyScope) (*WizardProgressResponse, error)

	// ─── Event-Driven (no auth context) ──────────────────────────────────────

	// InitializeWizard creates the wizard progress row for a new family.
	// Called from FamilyCreatedHandler — no auth context available. [04-onboard §11.1]
	InitializeWizard(ctx context.Context, familyID uuid.UUID) error

	// HandleMethodologyChanged re-materializes guidance when methodology changes.
	// Called from FamilyMethodologyChangedHandler — no auth context available. [04-onboard §11.2]
	HandleMethodologyChanged(ctx context.Context, familyID uuid.UUID, primarySlug string, secondarySlugs []string) error
}

// ─── Repository Interfaces ───────────────────────────────────────────────────

// WizardProgressRepository provides persistence for wizard progress tracking.
// Family-scoped — all queries MUST include family_id filter. [CODING §2.4]
type WizardProgressRepository interface {
	Create(ctx context.Context, progress *WizardProgress) error
	FindByFamilyID(ctx context.Context, familyID uuid.UUID) (*WizardProgress, error)
	Update(ctx context.Context, progress *WizardProgress) error
}

// RoadmapItemRepository provides persistence for materialized roadmap items.
type RoadmapItemRepository interface {
	ListByFamilyID(ctx context.Context, familyID uuid.UUID) ([]RoadmapItem, error)
	DeleteByFamilyID(ctx context.Context, familyID uuid.UUID) error
	BatchCreate(ctx context.Context, items []RoadmapItem) error
}

// StarterRecommendationRepository provides persistence for materialized recommendations.
type StarterRecommendationRepository interface {
	ListByFamilyID(ctx context.Context, familyID uuid.UUID) ([]StarterRecommendation, error)
	DeleteByFamilyID(ctx context.Context, familyID uuid.UUID) error
	BatchCreate(ctx context.Context, items []StarterRecommendation) error
}

// CommunitySuggestionRepository provides persistence for materialized community suggestions.
type CommunitySuggestionRepository interface {
	ListByFamilyID(ctx context.Context, familyID uuid.UUID) ([]CommunitySuggestion, error)
	DeleteByFamilyID(ctx context.Context, familyID uuid.UUID) error
	BatchCreate(ctx context.Context, items []CommunitySuggestion) error
}

// ─── Consumer-Defined Cross-Domain Interfaces ────────────────────────────────
// Narrow interfaces for cross-domain service calls. Adapters wired in main.go. [ARCH §4.2]

// IamServiceForOnboard is the subset of iam::IamService that onboard:: needs.
type IamServiceForOnboard interface {
	UpdateFamilyProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateFamilyProfileCommand) error
	CreateStudent(ctx context.Context, scope *shared.FamilyScope, cmd AddChildCommand) error
	DeleteStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID) error
	ListStudents(ctx context.Context, familyID uuid.UUID) ([]OnboardStudentInfo, error)
}

// MethodologyServiceForOnboard is the subset of method::MethodologyService that onboard:: needs.
type MethodologyServiceForOnboard interface {
	GetMethodology(ctx context.Context, slug string) (*OnboardMethodologyConfig, error)
	GetDefaultMethodologySlug(ctx context.Context) (string, error)
	ValidateMethodologySlugs(ctx context.Context, slugs []string) (bool, error)
	UpdateFamilyMethodology(ctx context.Context, scope *shared.FamilyScope, primarySlug string, secondarySlugs []string) error
}

// DiscoveryServiceForOnboard is the subset of discover::DiscoveryService that onboard:: needs.
type DiscoveryServiceForOnboard interface {
	GetQuizResult(ctx context.Context, shareID string) (*OnboardQuizResult, error)
	ClaimQuizResult(ctx context.Context, shareID string, familyID uuid.UUID) error
}
