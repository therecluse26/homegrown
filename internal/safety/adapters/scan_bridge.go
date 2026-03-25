package adapters

import (
	"strings"

	"github.com/homegrown-academy/homegrown-academy/internal/safety"
)

// suggestiveLabels that trigger flagging at 80% confidence. [11-safety §11.2.2]
var suggestiveLabels = map[string]bool{
	"suggestive":                    true,
	"female swimwear or underwear":  true,
	"male swimwear or underwear":    true,
	"revealing clothes":             true,
}

// violenceLabels that trigger flagging at min confidence. [11-safety §11.2.2]
var violenceLabels = map[string]bool{
	"violence":                  true,
	"graphic violence or gore":  true,
	"self-injury":               true,
}

// ignoredCategories are Rekognition label categories that are always ignored. [11-safety §11.2.2]
var ignoredCategories = map[string]bool{
	"drugs":    true,
	"tobacco":  true,
	"alcohol":  true,
	"hate symbols": true,
	"weapons":  true,
}

// underageLabels indicate possible underage subjects. [11-safety §11.2.2]
var underageLabels = map[string]bool{
	"minor": true,
}

func isIgnoredCategory(name string) bool {
	lower := strings.ToLower(name)
	for category := range ignoredCategories {
		if strings.Contains(lower, category) {
			return true
		}
	}
	return false
}

func isNudityLabel(name string, configLabels []string) bool {
	lower := strings.ToLower(name)
	for _, label := range configLabels {
		if strings.ToLower(label) == lower {
			return true
		}
	}
	return false
}

func isSuggestive(name string) bool {
	return suggestiveLabels[strings.ToLower(name)]
}

func isViolence(name string) bool {
	return violenceLabels[strings.ToLower(name)]
}

func isUnderageLabel(name string) bool {
	return underageLabels[strings.ToLower(name)]
}

// ApplyLabelRouting applies the platform's label routing table to raw Rekognition labels.
// Returns a ModerationResult with auto-reject/flag decisions. [11-safety §11.2.2]
func ApplyLabelRouting(rawLabels []safety.ModerationLabel, config *safety.SafetyConfig) *safety.ModerationResult {
	autoReject := false
	hasViolations := false
	var priority *string
	var keptLabels []safety.ModerationLabel

	hasUnderageIndicator := false
	for _, l := range rawLabels {
		if isUnderageLabel(l.Name) && l.Confidence >= 50.0 {
			hasUnderageIndicator = true
			break
		}
	}

	for _, label := range rawLabels {
		if isIgnoredCategory(label.Name) {
			continue
		}

		if isNudityLabel(label.Name, config.NudityAutoRejectLabels) &&
			label.Confidence >= config.RekognitionMinConfidence {
			autoReject = true
			hasViolations = true
			keptLabels = append(keptLabels, label)
		} else if isSuggestive(label.Name) && label.Confidence >= 80.0 {
			hasViolations = true
			keptLabels = append(keptLabels, label)
			if hasUnderageIndicator {
				p := "critical"
				priority = &p
			} else if priority == nil {
				p := "normal"
				priority = &p
			}
		} else if isViolence(label.Name) && label.Confidence >= config.RekognitionMinConfidence {
			hasViolations = true
			keptLabels = append(keptLabels, label)
			if priority == nil {
				p := "normal"
				priority = &p
			}
		}
	}

	if hasUnderageIndicator && hasViolations && !autoReject {
		p := "critical"
		priority = &p
	}

	var resultPriority *string
	if !autoReject {
		resultPriority = priority
	}

	return &safety.ModerationResult{
		HasViolations: hasViolations,
		AutoReject:    autoReject,
		Labels:        keptLabels,
		Priority:      resultPriority,
	}
}
