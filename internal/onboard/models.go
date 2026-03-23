package onboard

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── Enum Constants ──────────────────────────────────────────────────────────

// WizardStep represents a step in the onboarding wizard. [04-onboard §3.1]
type WizardStep string

const (
	StepFamilyProfile WizardStep = "family_profile"
	StepChildren      WizardStep = "children"
	StepMethodology   WizardStep = "methodology"
	StepRoadmapReview WizardStep = "roadmap_review"
)

// wizardStepOrder defines the sequential order of wizard steps.
var wizardStepOrder = []WizardStep{
	StepFamilyProfile,
	StepChildren,
	StepMethodology,
	StepRoadmapReview,
}

// requiredSteps are steps that MUST be completed before the wizard can finish.
var requiredSteps = map[WizardStep]bool{
	StepFamilyProfile: true,
	StepMethodology:   true,
}

// WizardStatus represents the overall wizard state. [04-onboard §3.1]
type WizardStatus string

const (
	StatusInProgress WizardStatus = "in_progress"
	StatusCompleted  WizardStatus = "completed"
	StatusSkipped    WizardStatus = "skipped"
)

// MethodologyPath represents how the user selected their methodology. [04-onboard §9.3]
type MethodologyPath string

const (
	PathQuizInformed MethodologyPath = "quiz_informed"
	PathExploration  MethodologyPath = "exploration"
	PathSkip         MethodologyPath = "skip"
)

// RoadmapItemType represents the type of roadmap item. [04-onboard §3.1]
type RoadmapItemType string

const (
	ItemTypeTask        RoadmapItemType = "task"
	ItemTypeResource    RoadmapItemType = "resource"
	ItemTypeFeatureTour RoadmapItemType = "feature_tour"
)

// AgeGroup represents a student age bracket for content filtering. [04-onboard §10.3]
type AgeGroup string

const (
	AgeGroupPreschool   AgeGroup = "3-5"
	AgeGroupEarlyElem   AgeGroup = "6-8"
	AgeGroupUpperElem   AgeGroup = "9-12"
	AgeGroupMiddle      AgeGroup = "13-15"
	AgeGroupHighSchool  AgeGroup = "16-18"
)

// AgeGroupFromBirthYear maps a student's birth year to an age group. [04-onboard §10.3]
// Uses current year to compute age. Returns empty string if birth year is nil.
func AgeGroupFromBirthYear(birthYear *int16, currentYear int) AgeGroup {
	if birthYear == nil {
		return ""
	}
	age := currentYear - int(*birthYear)
	switch {
	case age <= 5:
		return AgeGroupPreschool
	case age <= 8:
		return AgeGroupEarlyElem
	case age <= 12:
		return AgeGroupUpperElem
	case age <= 15:
		return AgeGroupMiddle
	default:
		return AgeGroupHighSchool
	}
}

// ─── Custom DB Types ─────────────────────────────────────────────────────────

// WizardStepArray is a custom type for PostgreSQL onb_wizard_step_enum[] columns.
// Follows the SlugArray pattern from iam.models. [04-onboard §3.2]
type WizardStepArray []WizardStep

// Value implements driver.Valuer. Serializes to PostgreSQL array literal.
func (a WizardStepArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	parts := make([]string, len(a))
	for i, s := range a {
		parts[i] = string(s)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}

// Scan implements sql.Scanner. Parses PostgreSQL array literal.
func (a *WizardStepArray) Scan(src any) error {
	if src == nil {
		*a = nil
		return nil
	}
	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("WizardStepArray.Scan: unsupported type %T", src)
	}
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*a = WizardStepArray{}
		return nil
	}
	parts := strings.Split(str, ",")
	result := make(WizardStepArray, len(parts))
	for i, p := range parts {
		result[i] = WizardStep(strings.TrimSpace(p))
	}
	*a = result
	return nil
}

// ─── GORM Models ─────────────────────────────────────────────────────────────

// WizardProgress is the GORM model for onb_wizard_progress. [04-onboard §3.2]
type WizardProgress struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID         uuid.UUID       `gorm:"type:uuid;not null;uniqueIndex"`
	Status           WizardStatus    `gorm:"type:onb_wizard_status_enum;not null;default:'in_progress'"`
	CurrentStep      WizardStep      `gorm:"type:onb_wizard_step_enum;not null;default:'family_profile'"`
	CompletedSteps   WizardStepArray `gorm:"type:onb_wizard_step_enum[]"`
	QuizShareID      *string         `gorm:""`
	MethodologyPath  *string         `gorm:""`
	CompletedAt      *time.Time      `gorm:""`
	CreatedAt        time.Time       `gorm:"not null;default:now()"`
	UpdatedAt        time.Time       `gorm:"not null;default:now()"`
}

func (WizardProgress) TableName() string { return "onb_wizard_progress" }

func (m *WizardProgress) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// RoadmapItem is the GORM model for onb_roadmap_items. [04-onboard §3.2]
type RoadmapItem struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID         uuid.UUID       `gorm:"type:uuid;not null"`
	WizardID         uuid.UUID       `gorm:"type:uuid;not null"`
	MethodologySlug  string          `gorm:"not null"`
	ItemType         RoadmapItemType `gorm:"type:onb_roadmap_item_type_enum;not null"`
	Title            string          `gorm:"not null"`
	Description      *string         `gorm:""`
	LinkURL          *string         `gorm:""`
	AgeGroup         *string         `gorm:""`
	SortOrder        int16           `gorm:"not null;default:0"`
	IsCompleted      bool            `gorm:"not null;default:false"`
	CreatedAt        time.Time       `gorm:"not null;default:now()"`
	UpdatedAt        time.Time       `gorm:"not null;default:now()"`
}

func (RoadmapItem) TableName() string { return "onb_roadmap_items" }

func (m *RoadmapItem) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// StarterRecommendation is the GORM model for onb_starter_recommendations. [04-onboard §3.2]
type StarterRecommendation struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID        uuid.UUID `gorm:"type:uuid;not null"`
	WizardID        uuid.UUID `gorm:"type:uuid;not null"`
	MethodologySlug string    `gorm:"not null"`
	Title           string    `gorm:"not null"`
	Description     *string   `gorm:""`
	LinkURL         *string   `gorm:""`
	PriceType       string    `gorm:"not null"`
	AgeGroup        *string   `gorm:""`
	SortOrder       int16     `gorm:"not null;default:0"`
	CreatedAt       time.Time `gorm:"not null;default:now()"`
}

func (StarterRecommendation) TableName() string { return "onb_starter_recommendations" }

func (m *StarterRecommendation) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// CommunitySuggestion is the GORM model for onb_community_suggestions. [04-onboard §3.2]
type CommunitySuggestion struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID        uuid.UUID  `gorm:"type:uuid;not null"`
	WizardID        uuid.UUID  `gorm:"type:uuid;not null"`
	MethodologySlug string     `gorm:"not null"`
	SuggestionType  string     `gorm:"not null"`
	Title           string     `gorm:"not null"`
	Description     *string    `gorm:""`
	ReferenceID     *uuid.UUID `gorm:"type:uuid"`
	SortOrder       int16      `gorm:"not null;default:0"`
	CreatedAt       time.Time  `gorm:"not null;default:now()"`
}

func (CommunitySuggestion) TableName() string { return "onb_community_suggestions" }

func (m *CommunitySuggestion) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ─── Internal JSON Types (methodology config parsing) ────────────────────────

// onboardingConfigInternal is the parsed shape of method_definitions.onboarding_config.
type onboardingConfigInternal struct {
	RoadmapSteps []roadmapStepConfig `json:"roadmap_steps"`
	StarterRecs  []starterRecConfig  `json:"starter_recs"`
}

type roadmapStepConfig struct {
	Type        string  `json:"type"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	LinkURL     string  `json:"link_url"`
	AgeGroup    string  `json:"age_group,omitempty"`
	SortOrder   int16   `json:"sort_order"`
}

type starterRecConfig struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	LinkURL     string `json:"link_url"`
	PriceType   string `json:"price_type"`
	AgeGroup    string `json:"age_group,omitempty"`
	SortOrder   int16  `json:"sort_order"`
}

// communityConfigInternal is the parsed shape of method_definitions.community_config.
type communityConfigInternal struct {
	DefaultGroups []communityGroupConfig `json:"default_groups"`
}

type communityGroupConfig struct {
	Type        string `json:"type"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

// ─── API Request Types ───────────────────────────────────────────────────────

// UpdateFamilyProfileCommand is the request body for PATCH /v1/onboarding/family-profile. [04-onboard §4]
type UpdateFamilyProfileCommand struct {
	DisplayName    string  `json:"display_name"               validate:"required,min=1,max=200"`
	StateCode      *string `json:"state_code,omitempty"       validate:"omitempty,len=2"`
	LocationRegion *string `json:"location_region,omitempty"  validate:"omitempty,max=200"`
}

// AddChildCommand is the request body for POST /v1/onboarding/children. [04-onboard §4]
type AddChildCommand struct {
	DisplayName string  `json:"display_name"       validate:"required,min=1,max=100"`
	BirthYear   *int16  `json:"birth_year,omitempty" validate:"omitempty,min=2000,max=2030"`
	GradeLevel  *string `json:"grade_level,omitempty" validate:"omitempty,max=20"`
}

// SelectMethodologyCommand is the request body for PATCH /v1/onboarding/methodology. [04-onboard §4]
type SelectMethodologyCommand struct {
	PrimaryMethodologySlug    string   `json:"primary_methodology_slug"    validate:"required"`
	SecondaryMethodologySlugs []string `json:"secondary_methodology_slugs" validate:"max=5"`
	MethodologyPath           string   `json:"methodology_path"            validate:"required,oneof=quiz_informed exploration skip"`
	ExplanationAcknowledged   bool     `json:"explanation_acknowledged"`
}

// ImportQuizCommand is the request body for POST /v1/onboarding/methodology/import-quiz. [04-onboard §4]
type ImportQuizCommand struct {
	ShareID string `json:"share_id" validate:"required"`
}

// ─── API Response Types ──────────────────────────────────────────────────────

// WizardProgressResponse is the response for GET /v1/onboarding/progress. [04-onboard §4]
type WizardProgressResponse struct {
	ID               uuid.UUID      `json:"id"`
	FamilyID         uuid.UUID      `json:"family_id"`
	Status           WizardStatus   `json:"status"`
	CurrentStep      WizardStep     `json:"current_step"`
	CompletedSteps   []WizardStep   `json:"completed_steps"`
	QuizShareID      *string        `json:"quiz_share_id,omitempty"`
	MethodologyPath  *string        `json:"methodology_path,omitempty"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
}

// RoadmapResponse is the response for GET /v1/onboarding/roadmap. [04-onboard §4]
type RoadmapResponse struct {
	Groups []RoadmapAgeGroup `json:"groups"`
}

// RoadmapAgeGroup groups roadmap items by age bracket. nil age_group = "all ages".
type RoadmapAgeGroup struct {
	AgeGroup *string              `json:"age_group"`
	Items    []RoadmapItemResponse `json:"items"`
}

// RoadmapItemResponse is a single roadmap item.
type RoadmapItemResponse struct {
	ID              uuid.UUID       `json:"id"`
	MethodologySlug string          `json:"methodology_slug"`
	ItemType        RoadmapItemType `json:"item_type"`
	Title           string          `json:"title"`
	Description     *string         `json:"description,omitempty"`
	LinkURL         *string         `json:"link_url,omitempty"`
	AgeGroup        *string         `json:"age_group,omitempty"`
	SortOrder       int16           `json:"sort_order"`
	IsCompleted     bool            `json:"is_completed"`
}

// RecommendationsResponse is the response for GET /v1/onboarding/recommendations. [04-onboard §4]
type RecommendationsResponse struct {
	Groups []RecommendationAgeGroup `json:"groups"`
}

// RecommendationAgeGroup groups recommendation items by age bracket. nil age_group = "all ages".
type RecommendationAgeGroup struct {
	AgeGroup *string                      `json:"age_group"`
	Items    []RecommendationItemResponse `json:"items"`
}

// RecommendationItemResponse is a single recommendation item.
type RecommendationItemResponse struct {
	ID              uuid.UUID `json:"id"`
	MethodologySlug string    `json:"methodology_slug"`
	Title           string    `json:"title"`
	Description     *string   `json:"description,omitempty"`
	LinkURL         *string   `json:"link_url,omitempty"`
	PriceType       string    `json:"price_type"`
	AgeGroup        *string   `json:"age_group,omitempty"`
	SortOrder       int16     `json:"sort_order"`
}

// CommunityResponse is the response for GET /v1/onboarding/community. [04-onboard §4]
type CommunityResponse struct {
	Items []CommunitySuggestionResponse `json:"items"`
}

// CommunitySuggestionResponse is a single community suggestion.
type CommunitySuggestionResponse struct {
	ID              uuid.UUID  `json:"id"`
	MethodologySlug string     `json:"methodology_slug"`
	SuggestionType  string     `json:"suggestion_type"`
	Title           string     `json:"title"`
	Description     *string    `json:"description,omitempty"`
	ReferenceID     *uuid.UUID `json:"reference_id,omitempty"`
	SortOrder       int16      `json:"sort_order"`
}

// QuizImportResponse is the response for POST /v1/onboarding/methodology/import-quiz. [04-onboard §4]
type QuizImportResponse struct {
	ShareID                     string                     `json:"share_id"`
	SuggestedPrimarySlug        string                     `json:"suggested_primary_slug"`
	MethodologyRecommendations  []OnboardQuizRecommendation `json:"methodology_recommendations"`
}

// ─── Cross-Domain Response Types ─────────────────────────────────────────────
// Slim types that onboard:: uses for cross-domain adapter results.

// OnboardStudentInfo is the minimal student data needed by onboard:: for age grouping.
type OnboardStudentInfo struct {
	ID        uuid.UUID
	BirthYear *int16
}

// OnboardMethodologyConfig holds parsed methodology config for materialization.
type OnboardMethodologyConfig struct {
	Slug             string
	DisplayName      string
	OnboardingConfig json.RawMessage
	CommunityConfig  json.RawMessage
}

// OnboardQuizResult holds quiz result data for import.
type OnboardQuizResult struct {
	ShareID         string
	Recommendations []OnboardQuizRecommendation
}

// OnboardQuizRecommendation is a single methodology recommendation from a quiz result.
type OnboardQuizRecommendation struct {
	MethodologySlug string `json:"methodology_slug"`
	MethodologyName string `json:"methodology_name"`
	ScorePercentage uint8  `json:"score_percentage"`
	Explanation     string `json:"explanation"`
}
