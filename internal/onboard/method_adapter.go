package onboard

import (
	"context"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// methodAdapter implements MethodologyServiceForOnboard using raw functions.
// Wired in cmd/server/main.go via NewMethodAdapter. [ARCH §4.2]
type methodAdapter struct {
	getMethodology          func(ctx context.Context, slug string) (*OnboardMethodologyConfig, error)
	getDefaultSlug          func(ctx context.Context) (string, error)
	validateSlugs           func(ctx context.Context, slugs []string) (bool, error)
	updateFamilyMethodology func(ctx context.Context, scope *shared.FamilyScope, primarySlug string, secondarySlugs []string) error
}

func (a *methodAdapter) GetMethodology(ctx context.Context, slug string) (*OnboardMethodologyConfig, error) {
	return a.getMethodology(ctx, slug)
}

func (a *methodAdapter) GetDefaultMethodologySlug(ctx context.Context) (string, error) {
	return a.getDefaultSlug(ctx)
}

func (a *methodAdapter) ValidateMethodologySlugs(ctx context.Context, slugs []string) (bool, error) {
	return a.validateSlugs(ctx, slugs)
}

func (a *methodAdapter) UpdateFamilyMethodology(ctx context.Context, scope *shared.FamilyScope, primarySlug string, secondarySlugs []string) error {
	return a.updateFamilyMethodology(ctx, scope, primarySlug, secondarySlugs)
}

// NewMethodAdapter creates a MethodologyServiceForOnboard adapter from raw functions.
func NewMethodAdapter(
	getMethodology func(ctx context.Context, slug string) (*OnboardMethodologyConfig, error),
	getDefaultSlug func(ctx context.Context) (string, error),
	validateSlugs func(ctx context.Context, slugs []string) (bool, error),
	updateFamilyMethodology func(ctx context.Context, scope *shared.FamilyScope, primarySlug string, secondarySlugs []string) error,
) MethodologyServiceForOnboard {
	return &methodAdapter{
		getMethodology:          getMethodology,
		getDefaultSlug:          getDefaultSlug,
		validateSlugs:           validateSlugs,
		updateFamilyMethodology: updateFamilyMethodology,
	}
}
