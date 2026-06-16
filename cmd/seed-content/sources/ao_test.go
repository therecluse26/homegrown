package sources_test

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/homegrown-academy/homegrown-academy/cmd/seed-content/sources"
)

// ─── ParseAOManifests — real data ────────────────────────────────────────────

func TestParseAOManifests_RealData(t *testing.T) {
	fsys := os.DirFS("../data/ao")
	manifests, err := sources.ParseAOManifests(fsys)
	if err != nil {
		t.Fatalf("ParseAOManifests: %v", err)
	}
	if len(manifests) != 6 {
		t.Errorf("expected 6 year manifests, got %d", len(manifests))
	}

	totalEntries := 0
	for _, m := range manifests {
		if m.Year < 1 || m.Year > 6 {
			t.Errorf("manifest has out-of-range year: %d", m.Year)
		}
		if m.GradeMin <= 0 {
			t.Errorf("year %d: grade_min must be positive, got %d", m.Year, m.GradeMin)
		}
		if m.GradeMax < m.GradeMin {
			t.Errorf("year %d: grade_max (%d) < grade_min (%d)", m.Year, m.GradeMax, m.GradeMin)
		}
		if len(m.Entries) == 0 {
			t.Errorf("year %d: has no entries", m.Year)
		}
		totalEntries += len(m.Entries)
	}

	// HOM-43 specifies ~120 listings total.
	if totalEntries < 100 || totalEntries > 150 {
		t.Errorf("expected ~120 total entries, got %d", totalEntries)
	}
}

func TestParseAOManifests_AllEntriesValid(t *testing.T) {
	fsys := os.DirFS("../data/ao")
	manifests, err := sources.ParseAOManifests(fsys)
	if err != nil {
		t.Fatalf("ParseAOManifests: %v", err)
	}

	validContentTypes := map[string]bool{
		"book_list":     true,
		"lesson_plan":   true,
		"curriculum":    true,
		"reading_guide": true,
	}

	seenURLs := make(map[string]string) // url → "year N: title"

	for _, m := range manifests {
		for i, e := range m.Entries {
			loc := fmt.Sprintf("year %d entry %d (%q)", m.Year, i, e.Title)
			if e.Title == "" {
				t.Errorf("%s: empty title", loc)
			}
			if e.Description == "" {
				t.Errorf("%s: empty description", loc)
			}
			if !validContentTypes[e.ContentType] {
				t.Errorf("%s: unknown content_type %q", loc, e.ContentType)
			}
			if len(e.SubjectTags) == 0 {
				t.Errorf("%s: no subject_tags", loc)
			}
			if e.PreviewURL == "" {
				t.Errorf("%s: empty preview_url", loc)
			}
			if len(e.WorldviewTags) == 0 {
				t.Errorf("%s: no worldview_tags", loc)
			}

			// Cross-entry dedup check: each preview_url must be unique.
			key := fmt.Sprintf("year %d: %s", m.Year, e.Title)
			if prev, dup := seenURLs[e.PreviewURL]; dup {
				t.Errorf("duplicate preview_url %q: %s and %s", e.PreviewURL, prev, key)
			}
			seenURLs[e.PreviewURL] = key
		}
	}
}

// ─── AOEntry.BuildDescription ────────────────────────────────────────────────

func TestAOEntryBuildDescription_ContainsAttribution(t *testing.T) {
	e := sources.AOEntry{
		Title:         "Aesop's Fables",
		Description:   "Classic animal fables teaching moral lessons.",
		ContentType:   "book_list",
		PreviewURL:    "https://www.gutenberg.org/ebooks/11339",
		WorldviewTags: []string{"secular"},
	}
	desc := e.BuildDescription()
	if !strings.Contains(desc, "amblesideonline.org") {
		t.Errorf("description missing amblesideonline.org attribution: %q", desc)
	}
	if !strings.Contains(desc, "Public Domain") {
		t.Errorf("description missing 'Public Domain': %q", desc)
	}
}

func TestAOEntryBuildDescription_PreservesBody(t *testing.T) {
	e := sources.AOEntry{
		Title:         "The Wind in the Willows",
		Description:   "Kenneth Grahame's timeless tale of Mole and Rat.",
		ContentType:   "book_list",
		PreviewURL:    "https://www.gutenberg.org/ebooks/289",
		WorldviewTags: []string{"secular"},
	}
	desc := e.BuildDescription()
	if !strings.Contains(desc, "Kenneth Grahame") {
		t.Errorf("description did not preserve body: %q", desc)
	}
}

func TestAOEntryBuildDescription_EmptyBodyFallback(t *testing.T) {
	e := sources.AOEntry{
		ContentType:   "book_list",
		PreviewURL:    "https://example.com",
		WorldviewTags: []string{"secular"},
	}
	desc := e.BuildDescription()
	if !strings.Contains(desc, "amblesideonline.org") {
		t.Errorf("empty-body fallback should still contain attribution: %q", desc)
	}
}

// ─── ParseAOManifests — synthetic FS ────────────────────────────────────────

func TestParseAOManifests_SyntheticFS(t *testing.T) {
	const yamlContent = `
year: 1
grade_min: 1
grade_max: 2
entries:
  - title: "Test Book"
    description: "A test description."
    content_type: book_list
    subject_tags: [language-arts]
    preview_url: "https://www.gutenberg.org/ebooks/1"
    worldview_tags: [secular]
`
	fsys := fstest.MapFS{
		"year-1.yaml": &fstest.MapFile{Data: []byte(yamlContent)},
	}
	manifests, err := sources.ParseAOManifests(fsys)
	if err != nil {
		t.Fatalf("ParseAOManifests: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(manifests))
	}
	m := manifests[0]
	if m.Year != 1 {
		t.Errorf("year = %d, want 1", m.Year)
	}
	if m.GradeMin != 1 {
		t.Errorf("grade_min = %d, want 1", m.GradeMin)
	}
	if m.GradeMax != 2 {
		t.Errorf("grade_max = %d, want 2", m.GradeMax)
	}
	if len(m.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(m.Entries))
	}
	e := m.Entries[0]
	if e.Title != "Test Book" {
		t.Errorf("title = %q, want %q", e.Title, "Test Book")
	}
	if e.ContentType != "book_list" {
		t.Errorf("content_type = %q, want %q", e.ContentType, "book_list")
	}
	desc := e.BuildDescription()
	if !strings.Contains(desc, "A test description") {
		t.Errorf("BuildDescription missing body: %q", desc)
	}
	if !strings.Contains(desc, "amblesideonline.org") {
		t.Errorf("BuildDescription missing attribution: %q", desc)
	}
}

func TestParseAOManifests_EmptyFS(t *testing.T) {
	fsys := fstest.MapFS{}
	_, err := sources.ParseAOManifests(fsys)
	if err == nil {
		t.Fatal("expected error for empty FS, got nil")
	}
}

func TestParseAOManifests_InvalidYAML(t *testing.T) {
	fsys := fstest.MapFS{
		"year-1.yaml": &fstest.MapFile{Data: []byte("{not valid yaml: [")},
	}
	_, err := sources.ParseAOManifests(fsys)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParseAOManifests_MultipleFiles(t *testing.T) {
	const yr1 = `
year: 1
grade_min: 1
grade_max: 2
entries:
  - title: "Book A"
    description: "Desc A."
    content_type: book_list
    subject_tags: [language-arts]
    preview_url: "https://example.com/a"
    worldview_tags: [secular]
`
	const yr2 = `
year: 2
grade_min: 2
grade_max: 3
entries:
  - title: "Book B"
    description: "Desc B."
    content_type: lesson_plan
    subject_tags: [history]
    preview_url: "https://example.com/b"
    worldview_tags: [christian]
`
	fsys := fstest.MapFS{
		"year-1.yaml": &fstest.MapFile{Data: []byte(yr1)},
		"year-2.yaml": &fstest.MapFile{Data: []byte(yr2)},
	}
	manifests, err := sources.ParseAOManifests(fsys)
	if err != nil {
		t.Fatalf("ParseAOManifests: %v", err)
	}
	if len(manifests) != 2 {
		t.Errorf("expected 2 manifests, got %d", len(manifests))
	}
	total := 0
	for _, m := range manifests {
		total += len(m.Entries)
	}
	if total != 2 {
		t.Errorf("expected 2 total entries, got %d", total)
	}
}
