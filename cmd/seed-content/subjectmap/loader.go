// Package subjectmap maps external content-source subject strings to
// learn_subject_taxonomy slugs. The mapping is driven by subject-map.yaml
// (embedded at compile time), so no runtime file-system access is required.
//
// The YAML top-level keys are source namespaces (e.g. "gutenberg_subjects").
// All namespaces are merged into a single lookup at first use.
package subjectmap

import (
	_ "embed"
	"log/slog"
	"maps"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed subject-map.yaml
var rawYAML []byte

// sourceFile mirrors the YAML structure: source-namespace → subject → slugs.
type sourceFile map[string]map[string][]string

var (
	once    sync.Once
	unified map[string][]string
)

func loadOnce() {
	var file sourceFile
	if err := yaml.Unmarshal(rawYAML, &file); err != nil {
		slog.Error("subjectmap: cannot parse subject-map.yaml", "err", err)
		unified = map[string][]string{}
		return
	}
	unified = make(map[string][]string, 256)
	for _, sourceMap := range file {
		maps.Copy(unified, sourceMap)
	}
}

// MapSubjects maps a slice of external subject strings to learn_subject_taxonomy
// slugs. Unknown subjects are logged as warnings and skipped. The returned slice
// is deduplicated. Returns nil when no subjects match.
func MapSubjects(sourceSubjects []string) []string {
	once.Do(loadOnce)

	seen := make(map[string]struct{}, len(sourceSubjects))
	var result []string

	for _, subject := range sourceSubjects {
		slugs, ok := unified[subject]
		if !ok {
			slog.Warn("subjectmap: unknown subject", "subject", subject)
			continue
		}
		for _, slug := range slugs {
			if _, dup := seen[slug]; !dup {
				seen[slug] = struct{}{}
				result = append(result, slug)
			}
		}
	}

	return result
}
