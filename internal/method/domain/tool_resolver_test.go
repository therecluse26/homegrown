package domain

import (
	"encoding/json"
	"testing"
)

// Test fixtures — two methodology slugs.
var (
	primaryMethodSlug   = "primary-methodology"
	secondaryMethodSlug = "secondary-methodology"
)

func activation(methodSlug, toolSlug string, active bool, sortOrder int16, overrides string) ToolActivationWithTool {
	return ToolActivationWithTool{
		MethodologySlug: methodSlug,
		ToolSlug:        toolSlug,
		ToolDisplayName: toolSlug,
		ToolTier:        "free",
		ToolIsActive:    active,
		ConfigOverrides: json.RawMessage(overrides),
		SortOrder:       sortOrder,
	}
}

// TestToolResolver_SingleMethodology verifies that a single-methodology family
// gets exactly the tools activated for that methodology. [02-method §13 assertion 5]
func TestToolResolver_SingleMethodology(t *testing.T) {
	activations := []ToolActivationWithTool{
		activation(primaryMethodSlug, "activities", true, 1, `{"label": "Lessons"}`),
		activation(primaryMethodSlug, "reading-lists", true, 2, `{"label": "Books"}`),
	}

	resolver := NewToolResolver(activations, primaryMethodSlug)
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 2 {
		t.Fatalf("want 2 tools, got %d", len(resolved))
	}
	if resolved[0].Slug != "activities" {
		t.Errorf("want first tool slug 'activities', got %q", resolved[0].Slug)
	}
	if resolved[1].Slug != "reading-lists" {
		t.Errorf("want second tool slug 'reading-lists', got %q", resolved[1].Slug)
	}
}

// TestToolResolver_Union verifies that tools from multiple methodologies are combined
// (union). [02-method §13 assertion 6]
func TestToolResolver_Union(t *testing.T) {
	activations := []ToolActivationWithTool{
		activation(primaryMethodSlug, "activities", true, 1, `{"label": "Primary"}`),
		activation(secondaryMethodSlug, "reading-lists", true, 2, `{"label": "Secondary"}`),
	}

	resolver := NewToolResolver(activations, primaryMethodSlug)
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 2 {
		t.Fatalf("want 2 tools from union, got %d", len(resolved))
	}

	slugs := map[string]bool{}
	for _, r := range resolved {
		slugs[r.Slug] = true
	}
	if !slugs["activities"] || !slugs["reading-lists"] {
		t.Errorf("union should include both tools, got slugs: %v", slugs)
	}
}

// TestToolResolver_DedupPrimaryWins verifies that when a tool appears in both primary
// and secondary, the primary methodology's config overrides are used. [02-method §13 assertion 7]
func TestToolResolver_DedupPrimaryWins(t *testing.T) {
	activations := []ToolActivationWithTool{
		activation(primaryMethodSlug, "activities", true, 1, `{"label": "Primary Label"}`),
		activation(secondaryMethodSlug, "activities", true, 1, `{"label": "Secondary Label"}`),
	}

	resolver := NewToolResolver(activations, primaryMethodSlug)
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 1 {
		t.Fatalf("want 1 tool (deduped), got %d", len(resolved))
	}
	if resolved[0].SourceMethodologySlug != primaryMethodSlug {
		t.Errorf("want source from primary methodology, got %q", resolved[0].SourceMethodologySlug)
	}

	// Check config overrides come from primary
	var overrides map[string]string
	if err := json.Unmarshal(resolved[0].ConfigOverrides, &overrides); err != nil {
		t.Fatal(err)
	}
	if overrides["label"] != "Primary Label" {
		t.Errorf("want primary label 'Primary Label', got %q", overrides["label"])
	}
}

// TestToolResolver_SecondaryOnlyTool verifies that when a tool is activated by
// secondary but not primary, the secondary's config overrides are used. [02-method §13 assertion 8]
func TestToolResolver_SecondaryOnlyTool(t *testing.T) {
	activations := []ToolActivationWithTool{
		activation(primaryMethodSlug, "activities", true, 1, `{}`),
		activation(secondaryMethodSlug, "nature-journals", true, 2, `{"label": "Nature Journal"}`),
	}

	resolver := NewToolResolver(activations, primaryMethodSlug)
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 2 {
		t.Fatalf("want 2 tools, got %d", len(resolved))
	}

	// Find the nature-journals tool
	var found bool
	for _, r := range resolved {
		if r.Slug == "nature-journals" {
			found = true
			if r.SourceMethodologySlug != secondaryMethodSlug {
				t.Errorf("want source from secondary methodology, got %q", r.SourceMethodologySlug)
			}
		}
	}
	if !found {
		t.Error("nature-journals tool not found in resolved tools")
	}
}

// TestToolResolver_FiltersInactive verifies that inactive tools are excluded
// from the resolved set. [02-method §13 assertion 5, §10.1 step 1]
func TestToolResolver_FiltersInactive(t *testing.T) {
	activations := []ToolActivationWithTool{
		activation(primaryMethodSlug, "activities", true, 1, `{}`),
		activation(primaryMethodSlug, "deprecated-tool", false, 2, `{}`), // inactive
		activation(primaryMethodSlug, "reading-lists", true, 3, `{}`),
	}

	resolver := NewToolResolver(activations, primaryMethodSlug)
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 2 {
		t.Fatalf("want 2 active tools, got %d", len(resolved))
	}
	for _, r := range resolved {
		if r.Slug == "deprecated-tool" {
			t.Error("inactive tool should not appear in resolved set")
		}
	}
}

// TestToolResolver_SortOrder verifies that resolved tools are sorted by sort_order.
func TestToolResolver_SortOrder(t *testing.T) {
	activations := []ToolActivationWithTool{
		activation(primaryMethodSlug, "third", true, 30, `{}`),
		activation(primaryMethodSlug, "first", true, 10, `{}`),
		activation(primaryMethodSlug, "second", true, 20, `{}`),
	}

	resolver := NewToolResolver(activations, primaryMethodSlug)
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 3 {
		t.Fatalf("want 3 tools, got %d", len(resolved))
	}
	if resolved[0].Slug != "first" || resolved[1].Slug != "second" || resolved[2].Slug != "third" {
		t.Errorf("tools not sorted by sort_order: %s, %s, %s",
			resolved[0].Slug, resolved[1].Slug, resolved[2].Slug)
	}
}

// TestToolResolver_EmptyActivations verifies that an empty activation set returns
// an empty resolved set.
func TestToolResolver_EmptyActivations(t *testing.T) {
	resolver := NewToolResolver(nil, primaryMethodSlug)
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 0 {
		t.Fatalf("want 0 tools from empty activations, got %d", len(resolved))
	}
}

// TestToolResolver_MultipleSecondaries verifies union across 3+ methodologies.
func TestToolResolver_MultipleSecondaries(t *testing.T) {
	thirdMethodSlug := "third-methodology"
	activations := []ToolActivationWithTool{
		activation(primaryMethodSlug, "activities", true, 1, `{"label": "Primary"}`),
		activation(secondaryMethodSlug, "reading-lists", true, 2, `{}`),
		activation(thirdMethodSlug, "projects", true, 3, `{}`),
		// activities also in secondary — primary should win
		activation(secondaryMethodSlug, "activities", true, 1, `{"label": "Secondary"}`),
	}

	resolver := NewToolResolver(activations, primaryMethodSlug)
	resolved, err := resolver.Resolve()
	if err != nil {
		t.Fatal(err)
	}

	if len(resolved) != 3 {
		t.Fatalf("want 3 unique tools, got %d", len(resolved))
	}

	// Verify primary won for activities
	for _, r := range resolved {
		if r.Slug == "activities" && r.SourceMethodologySlug != primaryMethodSlug {
			t.Errorf("activities should come from primary, got source %q", r.SourceMethodologySlug)
		}
	}
}
