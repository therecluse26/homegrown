package discover

import "context"

// ─── Service Interface ────────────────────────────────────────────────────────

// DiscoveryService defines all Phase 1 use cases for the discovery domain.
// All methods are public (no FamilyScope required). [03-discover §1]
// Implementation: discoveryServiceImpl in service.go.
type DiscoveryService interface {
	// GetActiveQuiz returns the currently active quiz definition with weights stripped.
	// Returns ErrNoActiveQuiz (→ 404) if no quiz has status='active'. [03-discover §8.1]
	GetActiveQuiz(ctx context.Context) (*QuizResponse, error)

	// SubmitQuiz scores the submitted answers and persists the result.
	// Returns a share_id for retrieving the result later. [03-discover §8.2]
	// Partial submissions (missing questions) are valid — missing answers contribute 0 weight.
	SubmitQuiz(ctx context.Context, cmd SubmitQuizCommand) (*QuizResultResponse, error)

	// GetQuizResult retrieves a previously submitted quiz result by its share_id.
	// Returns ErrQuizResultNotFound (→ 404) if the share_id does not exist. [03-discover §8.2]
	GetQuizResult(ctx context.Context, shareID string) (*QuizResultResponse, error)

	// ListStateGuides returns a summary of all 51 state guides.
	// IsAvailable=true only for published guides. [03-discover §8.3]
	ListStateGuides(ctx context.Context) ([]StateGuideSummaryResponse, error)

	// GetStateGuide returns the full state guide for a state code.
	// Returns ErrStateGuideNotFound (→ 404) if not found or not published. [03-discover §8.3]
	GetStateGuide(ctx context.Context, stateCode string) (*StateGuideResponse, error)

	// GetStateRequirements returns structured requirements for a state for comply:: consumption.
	// Not an HTTP endpoint — internal service call only. [03-discover §5, §13.1]
	// Returns requirements regardless of guide published status (comply:: needs data before publish).
	GetStateRequirements(ctx context.Context, stateCode string) (*StateGuideRequirements, error)

	// ClaimQuizResult sets family_id on a quiz result.
	// Used by onboard:: to claim a quiz result during import. [04-onboard §9.4]
	// Returns ErrQuizResultAlreadyClaimed if already owned by a different family.
	ClaimQuizResult(ctx context.Context, shareID string, familyID any) error

	// GetContentBySlug returns a published content page by its slug. [03-discover §8.4]
	// Returns ErrContentPageNotFound (→ 404) if not found or not published.
	GetContentBySlug(ctx context.Context, slug string) (*ContentPage, error)
}

// ─── Repository Interfaces ────────────────────────────────────────────────────

// QuizDefinitionRepository provides read-only access to quiz definitions.
// NOT family-scoped — disc_quiz_definitions is global config. [03-discover §3.2]
type QuizDefinitionRepository interface {
	// FindActive returns the quiz definition with status='active'.
	// Returns &DiscoverError{Err: ErrNoActiveQuiz} if none exists.
	FindActive(ctx context.Context) (*QuizDefinition, error)

	// FindByID returns a quiz definition by its UUID.
	FindByID(ctx context.Context, id any) (*QuizDefinition, error)
}

// QuizResultRepository provides create/read access to quiz results.
// NOT family-scoped — results are anonymous until claimed. [03-discover §3.2]
type QuizResultRepository interface {
	// Create persists a new quiz result. Fails if share_id already exists (DB unique constraint).
	Create(ctx context.Context, input CreateQuizResult) (*QuizResult, error)

	// FindByShareID returns a quiz result by its URL-safe share_id.
	// Returns &DiscoverError{Err: ErrQuizResultNotFound} if not found.
	FindByShareID(ctx context.Context, shareID string) (*QuizResult, error)

	// ClaimForFamily sets family_id on a quiz result for the requesting family.
	// Returns ErrQuizResultAlreadyClaimed if already owned by a different family.
	// Defined here for Phase 2 but implemented now. [03-discover §3.1]
	ClaimForFamily(ctx context.Context, shareID string, familyID any) error
}

// StateGuideRepository provides read-only access to state legal guides.
// NOT family-scoped — disc_state_guides is global config. [03-discover §3.2]
type StateGuideRepository interface {
	// ListAll returns a summary of all state guides (all statuses).
	ListAll(ctx context.Context) ([]StateGuideSummary, error)

	// FindByStateCode returns the state guide for the given 2-char state code.
	// Returns &DiscoverError{Err: ErrStateGuideNotFound} if no row matches.
	// Returns &DiscoverError{Err: ErrStateGuideNotPublished} if found but status != 'published'.
	FindByStateCode(ctx context.Context, stateCode string) (*StateGuide, error)

	// FindRequirementsByStateCode returns any state guide regardless of published status.
	// Used by GetStateRequirements() — comply:: needs data even before guides are published.
	// Returns &DiscoverError{Err: ErrStateGuideNotFound} if no row matches.
	FindRequirementsByStateCode(ctx context.Context, stateCode string) (*StateGuide, error)
}

// ContentPageRepository provides read-only access to content pages (Phase 2).
// Defined here for interface completeness; implementation deferred to Phase 2.
type ContentPageRepository interface {
	// FindBySlug returns a published content page by slug.
	FindBySlug(ctx context.Context, slug string) (*ContentPage, error)

	// ListPublished returns all published content pages.
	ListPublished(ctx context.Context) ([]ContentPageSummary, error)
}

// ─── Consumer-Defined Cross-Domain Interface ──────────────────────────────────

// MethodologyServiceForDiscover is the subset of method::MethodologyService that
// discover:: needs. Defined here (consumer side) to avoid importing method::. [ARCH §4.2]
//
// Bridged in cmd/server/main.go via discover.NewMethodAdapter.
type MethodologyServiceForDiscover interface {
	// GetMethodologyDisplayName returns the human-readable display name for a methodology slug.
	// Returns the slug itself if the methodology is not found (graceful fallback).
	GetMethodologyDisplayName(ctx context.Context, slug string) (string, error)
}
