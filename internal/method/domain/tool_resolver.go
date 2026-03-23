package domain

import (
	"encoding/json"
	"sort"
)

// ToolActivationWithTool is a tool activation joined with tool metadata.
// Defined here (not in models.go) because ToolResolver depends on it and the domain
// package must not import the parent method package. [02-method §8.3, §10.1]
type ToolActivationWithTool struct {
	MethodologySlug string          `json:"methodology_slug"`
	ToolSlug        string          `json:"tool_slug"`
	ToolDisplayName string          `json:"tool_display_name"`
	ToolDescription *string         `json:"tool_description,omitempty"`
	ToolTier        string          `json:"tool_tier"`
	ToolIsActive    bool            `json:"tool_is_active"`
	ConfigOverrides json.RawMessage `json:"config_overrides"`
	SortOrder       int16           `json:"sort_order"`
}

// ResolvedTool is a tool with its resolved configuration (after dedup and precedence).
// [02-method §10.1]
type ResolvedTool struct {
	Slug                  string          `json:"slug"`
	DisplayName           string          `json:"display_name"`
	Description           *string         `json:"description,omitempty"`
	Tier                  string          `json:"tier"`
	ConfigOverrides       json.RawMessage `json:"config_overrides"`
	SortOrder             int16           `json:"sort_order"`
	SourceMethodologySlug string          `json:"source_methodology_slug"`
}

// ToolResolver resolves the active tool set for a given set of methodology selections.
// Enforces: deduplication, config precedence, inactive tool filtering. [S§4.2]
type ToolResolver struct {
	activations            []ToolActivationWithTool
	primaryMethodologySlug string
}

// NewToolResolver creates a new ToolResolver with the given activations and primary methodology slug.
func NewToolResolver(activations []ToolActivationWithTool, primaryMethodologySlug string) *ToolResolver {
	return &ToolResolver{
		activations:            activations,
		primaryMethodologySlug: primaryMethodologySlug,
	}
}

// Resolve resolves the active tool set by applying the tool resolution algorithm:
//
// 1. Filter out inactive tools (tool.ToolIsActive == false)
// 2. Union all tools across selected methodologies
// 3. Deduplicate: if a tool appears in multiple methodologies, keep the
//    activation from the PRIMARY methodology. If the tool is not activated
//    by the primary, keep the first secondary activation encountered.
// 4. Sort by the winning activation's sort_order
func (r *ToolResolver) Resolve() ([]ResolvedTool, error) {
	seen := make(map[string]ResolvedTool)

	// First pass: insert all primary methodology activations
	for _, activation := range r.activations {
		if !activation.ToolIsActive {
			continue
		}
		if activation.MethodologySlug == r.primaryMethodologySlug {
			seen[activation.ToolSlug] = newResolvedTool(&activation)
		}
	}

	// Second pass: insert secondary activations only if tool not already present
	for _, activation := range r.activations {
		if !activation.ToolIsActive {
			continue
		}
		if activation.MethodologySlug != r.primaryMethodologySlug {
			if _, exists := seen[activation.ToolSlug]; !exists {
				seen[activation.ToolSlug] = newResolvedTool(&activation)
			}
		}
	}

	tools := make([]ResolvedTool, 0, len(seen))
	for _, tool := range seen {
		tools = append(tools, tool)
	}
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].SortOrder < tools[j].SortOrder
	})
	return tools, nil
}

func newResolvedTool(activation *ToolActivationWithTool) ResolvedTool {
	return ResolvedTool{
		Slug:                  activation.ToolSlug,
		DisplayName:           activation.ToolDisplayName,
		Description:           activation.ToolDescription,
		Tier:                  activation.ToolTier,
		ConfigOverrides:       activation.ConfigOverrides,
		SortOrder:             activation.SortOrder,
		SourceMethodologySlug: activation.MethodologySlug,
	}
}
