package method

import (
	"encoding/json"
	"time"
)

// ─── Typed String Identifiers ─────────────────────────────────────────────────

// MethodologyID is a typed string for methodology slugs.
// Using a distinct type prevents accidental mixing of slug types and enables
// compile-time references via constants. [ADR: natural-string-pks]
type MethodologyID string

// ToolSlug is a typed string for tool slugs.
type ToolSlug string

// Platform-defined methodology slug constants. [02-method §8.3]
// These match the slug values seeded in migrations/20260322000003_seed_method_data.sql.
const (
	MethodologyCharlotteMason MethodologyID = "charlotte-mason"
	MethodologyTraditional    MethodologyID = "traditional"
	MethodologyClassical      MethodologyID = "classical"
	MethodologyWaldorf        MethodologyID = "waldorf"
	MethodologyMontessori     MethodologyID = "montessori"
	MethodologyUnschooling    MethodologyID = "unschooling"
)

// ─── GORM Models ──────────────────────────────────────────────────────────────

// MethodologyDefinition is the GORM model for the method_definitions table.
// slug is the natural primary key — no UUID indirection needed for immutable seed data.
// [02-method §8.3, ADR: natural-string-pks]
type MethodologyDefinition struct {
	Slug             MethodologyID   `gorm:"primaryKey;not null"`
	DisplayName      string          `gorm:"not null"`
	ShortDesc        string          `gorm:"not null"`
	IconURL          *string
	Philosophy       json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" swaggertype:"object"`
	OnboardingConfig json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" swaggertype:"object"`
	CommunityConfig  json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" swaggertype:"object"`
	MasteryPaths     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" swaggertype:"object"`
	Terminology      json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" swaggertype:"object"`
	DisplayOrder     int16           `gorm:"not null;default:0"`
	IsActive         bool            `gorm:"not null;default:true"`
	CreatedAt        time.Time       `gorm:"not null;default:now()"`
	UpdatedAt        time.Time       `gorm:"not null;default:now()"`
}

func (MethodologyDefinition) TableName() string { return "method_definitions" }

// Tool is the GORM model for the method_tools table.
// slug is the natural primary key. [02-method §8.3, ADR: natural-string-pks]
type Tool struct {
	Slug         ToolSlug        `gorm:"primaryKey;not null"`
	DisplayName  string          `gorm:"not null"`
	Description  *string
	ConfigSchema json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" swaggertype:"object"`
	Tier         string          `gorm:"not null;default:'free'"`
	IsActive     bool            `gorm:"not null;default:true"`
	CreatedAt    time.Time       `gorm:"not null;default:now()"`
}

func (Tool) TableName() string { return "method_tools" }

// ─── API Request Types ────────────────────────────────────────────────────────

// UpdateMethodologyCommand is the request body for PATCH /v1/families/methodology. [S§4.3]
type UpdateMethodologyCommand struct {
	PrimaryMethodologySlug    MethodologyID   `json:"primary_methodology_slug"    validate:"required"`
	SecondaryMethodologySlugs []MethodologyID `json:"secondary_methodology_slugs" validate:"max=5"`
}

// UpdateStudentMethodologyCommand is the request body for
// PATCH /v1/families/students/:id/methodology (Phase 2). [S§4.6]
type UpdateStudentMethodologyCommand struct {
	MethodologyOverrideSlug *MethodologyID `json:"methodology_override_slug"`
}

// ─── API Response Types ───────────────────────────────────────────────────────

// MethodologySummaryResponse is the list item for GET /v1/methodologies. [02-method §8.2]
// slug is the canonical identifier — no separate ID field needed. [ADR: natural-string-pks]
type MethodologySummaryResponse struct {
	Slug        MethodologyID `json:"slug"`
	DisplayName string        `json:"display_name"`
	ShortDesc   string        `json:"short_desc"`
	IconURL     *string       `json:"icon_url,omitempty"`
}

// MethodologyDetailResponse is the full detail for GET /v1/methodologies/:slug. [02-method §8.2]
type MethodologyDetailResponse struct {
	Slug             MethodologyID   `json:"slug"`
	DisplayName      string          `json:"display_name"`
	ShortDesc        string          `json:"short_desc"`
	IconURL          *string         `json:"icon_url,omitempty"`
	Philosophy       json.RawMessage `json:"philosophy"         swaggertype:"object"`
	OnboardingConfig json.RawMessage `json:"onboarding_config"  swaggertype:"object"`
	CommunityConfig  json.RawMessage `json:"community_config"   swaggertype:"object"`
	MasteryPaths     json.RawMessage `json:"mastery_paths"      swaggertype:"object"`
	Terminology      json.RawMessage `json:"terminology"        swaggertype:"object"`
}

// ActiveToolResponse is a tool as resolved for a family or methodology. [02-method §8.2]
// slug is the canonical identifier — no separate tool_id needed. [ADR: natural-string-pks]
type ActiveToolResponse struct {
	Slug            ToolSlug        `json:"slug"`
	DisplayName     string          `json:"display_name"`
	Label           *string         `json:"label,omitempty"`
	Description     *string         `json:"description,omitempty"`
	Tier            string          `json:"tier"`
	Guidance        *string         `json:"guidance,omitempty"`
	ConfigOverrides json.RawMessage `json:"config_overrides" swaggertype:"object"`
	SortOrder       int16           `json:"sort_order"`
}

// MethodologySelectionResponse is the response for PATCH /v1/families/methodology. [02-method §8.2]
type MethodologySelectionResponse struct {
	Primary         MethodologySummaryResponse   `json:"primary"`
	Secondary       []MethodologySummaryResponse `json:"secondary"`
	ActiveToolCount int                          `json:"active_tool_count"`
}

// MethodologyContext is the response for GET /v1/families/methodology-context (Phase 2). [ARCH §7.2]
type MethodologyContext struct {
	Primary      MethodologySummaryResponse   `json:"primary"`
	Secondary    []MethodologySummaryResponse `json:"secondary"`
	Terminology  json.RawMessage              `json:"terminology" swaggertype:"object"`
	MasteryLevel *string                      `json:"mastery_level,omitempty"`
}

// ─── Response Builders ────────────────────────────────────────────────────────

func toSummaryResponse(m *MethodologyDefinition) MethodologySummaryResponse {
	return MethodologySummaryResponse{
		Slug:        m.Slug,
		DisplayName: m.DisplayName,
		ShortDesc:   m.ShortDesc,
		IconURL:     m.IconURL,
	}
}

func toDetailResponse(m *MethodologyDefinition) *MethodologyDetailResponse {
	return &MethodologyDetailResponse{
		Slug:             m.Slug,
		DisplayName:      m.DisplayName,
		ShortDesc:        m.ShortDesc,
		IconURL:          m.IconURL,
		Philosophy:       m.Philosophy,
		OnboardingConfig: m.OnboardingConfig,
		CommunityConfig:  m.CommunityConfig,
		MasteryPaths:     m.MasteryPaths,
		Terminology:      m.Terminology,
	}
}
