package comply

import (
	"context"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── IAM Adapter ────────────────────────────────────────────────────────────

// iamAdapter implements IamServiceForComply using raw functions.
// Avoids circular dependency with iam::. Wired in cmd/server/main.go. [ARCH §4.2]
type iamAdapter struct {
	studentBelongsToFamily func(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error)
	getStudentName         func(ctx context.Context, studentID uuid.UUID) (string, error)
}

func (a *iamAdapter) StudentBelongsToFamily(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error) {
	return a.studentBelongsToFamily(ctx, studentID, familyID)
}

func (a *iamAdapter) GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error) {
	return a.getStudentName(ctx, studentID)
}

// NewIamAdapter creates an IamServiceForComply adapter from raw functions.
func NewIamAdapter(
	studentBelongsToFamily func(ctx context.Context, studentID uuid.UUID, familyID shared.FamilyID) (bool, error),
	getStudentName func(ctx context.Context, studentID uuid.UUID) (string, error),
) IamServiceForComply {
	return &iamAdapter{
		studentBelongsToFamily: studentBelongsToFamily,
		getStudentName:         getStudentName,
	}
}

// ─── Learning Adapter ───────────────────────────────────────────────────────

// learnAdapter implements LearningServiceForComply using raw functions.
type learnAdapter struct {
	getPortfolioItemData func(ctx context.Context, familyID uuid.UUID, sourceType string, sourceID uuid.UUID) (*PortfolioItemData, error)
}

func (a *learnAdapter) GetPortfolioItemData(ctx context.Context, familyID uuid.UUID, sourceType string, sourceID uuid.UUID) (*PortfolioItemData, error) {
	return a.getPortfolioItemData(ctx, familyID, sourceType, sourceID)
}

// NewLearnAdapter creates a LearningServiceForComply adapter from raw functions.
func NewLearnAdapter(
	getPortfolioItemData func(ctx context.Context, familyID uuid.UUID, sourceType string, sourceID uuid.UUID) (*PortfolioItemData, error),
) LearningServiceForComply {
	return &learnAdapter{getPortfolioItemData: getPortfolioItemData}
}

// ─── Discovery Adapter ──────────────────────────────────────────────────────

// discoveryAdapter implements DiscoveryServiceForComply using raw functions.
type discoveryAdapter struct {
	getStateRequirements func(ctx context.Context, stateCode string) (*StateRequirementsData, error)
	listStateGuides      func(ctx context.Context) ([]StateGuideSummary, error)
}

func (a *discoveryAdapter) GetStateRequirements(ctx context.Context, stateCode string) (*StateRequirementsData, error) {
	return a.getStateRequirements(ctx, stateCode)
}

func (a *discoveryAdapter) ListStateGuides(ctx context.Context) ([]StateGuideSummary, error) {
	return a.listStateGuides(ctx)
}

// NewDiscoveryAdapter creates a DiscoveryServiceForComply adapter from raw functions.
func NewDiscoveryAdapter(
	getStateRequirements func(ctx context.Context, stateCode string) (*StateRequirementsData, error),
	listStateGuides func(ctx context.Context) ([]StateGuideSummary, error),
) DiscoveryServiceForComply {
	return &discoveryAdapter{
		getStateRequirements: getStateRequirements,
		listStateGuides:      listStateGuides,
	}
}

// ─── Media Adapter ──────────────────────────────────────────────────────────

// mediaAdapter implements MediaServiceForComply using raw functions.
type mediaAdapter struct {
	requestUpload func(ctx context.Context, familyID uuid.UUID, uploadContext string, filename string, contentType string, data []byte) (*uuid.UUID, error)
	presignedGet  func(ctx context.Context, uploadID uuid.UUID) (string, error)
}

func (a *mediaAdapter) RequestUpload(ctx context.Context, familyID uuid.UUID, uploadContext string, filename string, contentType string, data []byte) (*uuid.UUID, error) {
	return a.requestUpload(ctx, familyID, uploadContext, filename, contentType, data)
}

func (a *mediaAdapter) PresignedGet(ctx context.Context, uploadID uuid.UUID) (string, error) {
	return a.presignedGet(ctx, uploadID)
}

// NewMediaAdapter creates a MediaServiceForComply adapter from raw functions.
func NewMediaAdapter(
	requestUpload func(ctx context.Context, familyID uuid.UUID, uploadContext string, filename string, contentType string, data []byte) (*uuid.UUID, error),
	presignedGet func(ctx context.Context, uploadID uuid.UUID) (string, error),
) MediaServiceForComply {
	return &mediaAdapter{
		requestUpload: requestUpload,
		presignedGet:  presignedGet,
	}
}
