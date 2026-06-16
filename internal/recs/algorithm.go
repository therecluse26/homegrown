package recs

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"time"

	"github.com/google/uuid"
)

// algorithm.go contains the pure helper functions used by the scoring algorithm.
// Content neutrality: this file MUST NOT read or reference listing classification
// fields unrelated to methodology. See §10.8 and §13.6 for the full audit invariant.

// ─── Scoring Weights [13-recs §10.9] ─────────────────────────────────────────

const (
	weightMethodologyMatch float32 = 0.35
	weightPopularity       float32 = 0.25
	weightRelevance        float32 = 0.25
	weightFreshness        float32 = 0.10
	weightExploration      float32 = 0.05
)

// ─── Seasonal Mapping [13-recs §10.5] ────────────────────────────────────────

// Season represents a meteorological season.
type Season string

const (
	SeasonSpring Season = "spring"
	SeasonSummer Season = "summer"
	SeasonAutumn Season = "autumn"
	SeasonWinter Season = "winter"
)

// SeasonForMonth maps a calendar month to a meteorological season (Northern Hemisphere).
// [13-recs §10.5]
func SeasonForMonth(month time.Month) Season {
	switch month {
	case time.March, time.April, time.May:
		return SeasonSpring
	case time.June, time.July, time.August:
		return SeasonSummer
	case time.September, time.October, time.November:
		return SeasonAutumn
	default: // December, January, February
		return SeasonWinter
	}
}

// ─── Age-Band Coarsening [13-recs §14.1] ─────────────────────────────────────

// CoarsenAgeBand maps an age in years to a 3-year age band string for anonymization.
// Returns "" for ages outside the supported range (< 4 or >= 19). [13-recs §14.1]
func CoarsenAgeBand(age int) string {
	switch {
	case age < 4:
		return ""
	case age <= 6:
		return "4-6"
	case age <= 9:
		return "7-9"
	case age <= 12:
		return "10-12"
	case age <= 15:
		return "13-15"
	case age <= 18:
		return "16-18"
	default: // age >= 19
		return ""
	}
}

// ─── Duration Rounding [13-recs §14.1] ───────────────────────────────────────

// RoundDurationToNearest5 rounds a duration in minutes to the nearest 5 minutes.
// This coarsens activity durations before anonymization. [13-recs §14.1]
// Examples: 2→0, 3→5, 7→5, 8→10.
func RoundDurationToNearest5(minutes int) int {
	return ((minutes + 2) / 5) * 5
}

// ─── Composite Scoring [13-recs §10.9] ───────────────────────────────────────

// ScoringFactors holds the individual scoring inputs for a candidate recommendation.
type ScoringFactors struct {
	MethodologyMatch float32 // 1.0 = primary match, 0.7 = secondary, 0.0 = none
	Popularity       float32 // normalized 0.0–1.0 from per-methodology percentile
	Relevance        float32 // Jaccard similarity on subject tags vs recent signals
	Freshness        float32 // exponential decay on listing/group age, 0.0–1.0
	Exploration      float32 // 1.0 for exploration slots, 0.0 otherwise
}

// ComputeScore returns the composite recommendation score for a candidate. [13-recs §10.9]
func ComputeScore(f ScoringFactors) float32 {
	return f.MethodologyMatch*weightMethodologyMatch +
		f.Popularity*weightPopularity +
		f.Relevance*weightRelevance +
		f.Freshness*weightFreshness +
		f.Exploration*weightExploration
}

// ─── Seasonal Subjects Lookup [13-recs §10.5] ────────────────────────────────

// SeasonalSubjects maps each season to the subject tags that receive a small score boost.
// These represent subjects that naturally align with seasonal rhythms (e.g., gardening in
// spring, astronomy in winter). Content neutrality: no worldview or methodology references.
var SeasonalSubjects = map[Season]map[string]struct{}{
	SeasonSpring: {
		"gardening": {}, "nature_study": {}, "botany": {}, "ecology": {},
		"life_science": {}, "biology": {},
	},
	SeasonSummer: {
		"outdoor_education": {}, "geography": {}, "physical_education": {},
		"art": {}, "nature_study": {}, "field_trips": {},
	},
	SeasonAutumn: {
		"history": {}, "literature": {}, "writing": {},
		"harvest": {}, "cooking": {}, "home_economics": {},
	},
	SeasonWinter: {
		"astronomy": {}, "mathematics": {}, "music": {},
		"reading": {}, "crafts": {}, "indoor_science": {},
	},
}

// seasonalBoost is the additive score boost for seasonally aligned content. [13-recs §10.5]
const seasonalBoost float32 = 0.05

// HasSeasonalOverlap checks whether any of the given subject tags overlap with the
// current season's emphasized subjects. Returns true and the matched tag if found.
func HasSeasonalOverlap(tags []string, season Season) (bool, string) {
	subjects, ok := SeasonalSubjects[season]
	if !ok {
		return false, ""
	}
	for _, tag := range tags {
		if _, match := subjects[tag]; match {
			return true, tag
		}
	}
	return false, ""
}

// ─── Dominant Signal Detection [13-recs §13.1] ──────────────────────────────

// DominantSignalResult holds the determined source signal and human-readable label.
type DominantSignalResult struct {
	Signal SourceSignalType
	Label  string
}

// DetermineDominantSignal determines the primary source signal for a candidate based on
// which scoring factor contributed most. Used for recommendation transparency. [13-recs §13.1]
func DetermineDominantSignal(f ScoringFactors, primarySlug string, seasonalMatch bool, season Season) DominantSignalResult {
	if seasonalMatch {
		return DominantSignalResult{
			Signal: SourceSeasonal,
			Label:  "Great for " + string(season) + " learning",
		}
	}
	if f.Relevance > 0.7 {
		return DominantSignalResult{
			Signal: SourcePurchaseHistory,
			Label:  "Based on your recent activity",
		}
	}
	if f.Popularity > 0.7 {
		return DominantSignalResult{
			Signal: SourcePopularity,
			Label:  "Popular with " + primarySlug + " families",
		}
	}
	return DominantSignalResult{
		Signal: SourceMethodologyMatch,
		Label:  "Matches your " + primarySlug + " methodology",
	}
}

// ─── HMAC Anonymization [13-recs §14.3] ─────────────────────────────────────

// computeHMAC returns a deterministic, one-way HMAC-SHA256 hex string for a family ID.
// Used to anonymize family_id before storing in recs_anonymized_interactions.
// The same family_id + key always produces the same output, but the original
// family_id cannot be recovered. [13-recs §14.3]
func computeHMAC(familyID uuid.UUID, key []byte) string {
	mac := hmac.New(sha256.New, key)
	mac.Write(familyID[:])
	return hex.EncodeToString(mac.Sum(nil))
}

// ─── Learner Fit Scoring [18-learner-profile §6] ─────────────────────────────

// recsInterestToSubjectTag maps declared interest chip IDs to their primary subject_tag.
// Must stay in sync with learner_profile.InterestChips (quiz.go). [18-learner-profile §5]
var recsInterestToSubjectTag = map[string]string{
	"animals":  "nature_study",
	"art":      "art",
	"building": "crafts",
	"coding":   "indoor_science",
	"cooking":  "cooking",
	"drama":    "art",
	"history":  "history",
	"language": "writing",
	"math":     "mathematics",
	"music":    "music",
	"nature":   "ecology",
	"reading":  "reading",
	"science":  "indoor_science",
	"space":    "astronomy",
	"sport":    "physical_education",
}

// studentProfileForFit holds the fields needed from learner_profiles for fit scoring.
// Loaded via raw SQL in handleComputeRecommendationsTask to avoid importing learner_profile::.
type studentProfileForFit struct {
	ActivityFormat     *float64
	SessionLength      *float64
	Motivation         *float64
	SoloCollaborative  *float64
	Structure          *float64
	OutdoorKinesthetic *float64
	Interests          []string
	Confidence         float64
}

// computeFitScore computes a learner-profile fit score for one content item.
// preferenceTags: content's preference_tags JSONB decoded to map[string]float64.
// Returns (score, whyText, ok). ok=false means badge gate not met.
// Formula: mean(1 - |P_d - C_d|) over shared dimensions, +0.10 interest boost.
// Both fit_score ≥ 0.60 AND profile.confidence ≥ 0.60 required. [18-learner-profile §6.2]
func computeFitScore(profile studentProfileForFit, preferenceTags map[string]float64, contentSubjectTags []string) (float32, string, bool) {
	if len(preferenceTags) == 0 || profile.Confidence < 0.60 {
		return 0, "", false
	}

	dimValues := map[string]*float64{
		"activity_format":     profile.ActivityFormat,
		"session_length":      profile.SessionLength,
		"motivation":          profile.Motivation,
		"solo_collaborative":  profile.SoloCollaborative,
		"structure":           profile.Structure,
		"outdoor_kinesthetic": profile.OutdoorKinesthetic,
	}

	type scoredDim struct{ dim string; score float64 }
	var scored []scoredDim
	var total float64

	for dim, pVal := range dimValues {
		if pVal == nil {
			continue
		}
		cVal, hasTag := preferenceTags[dim]
		if !hasTag {
			continue
		}
		d := *pVal - cVal
		if d < 0 {
			d = -d
		}
		s := 1.0 - d
		scored = append(scored, scoredDim{dim, s})
		total += s
	}

	if len(scored) == 0 {
		return 0, "", false
	}

	fitScore := total / float64(len(scored))

	// Interest boost: +0.10 if any content subject_tag matches a declared interest.
	// Interests are chip IDs (e.g. "math") mapped to subject_tags (e.g. "mathematics")
	// via recsInterestToSubjectTag before comparison. [18-learner-profile §6.3]
	contentTagSet := make(map[string]struct{}, len(contentSubjectTags))
	for _, t := range contentSubjectTags {
		contentTagSet[t] = struct{}{}
	}
	for _, interest := range profile.Interests {
		subjectTag, ok := recsInterestToSubjectTag[interest]
		if !ok {
			subjectTag = interest // fall back to direct match
		}
		if _, found := contentTagSet[subjectTag]; found {
			fitScore += 0.10
			break
		}
	}
	if fitScore > 1.0 {
		fitScore = 1.0
	}

	if fitScore < 0.60 {
		return float32(fitScore), "", false
	}

	// Why-text from highest-contributing dimension.
	bestDim := ""
	var bestScore float64
	for _, sd := range scored {
		if sd.score > bestScore {
			bestScore = sd.score
			bestDim = sd.dim
		}
	}

	whyTemplates := map[string]string{
		"activity_format":     "This activity is hands-on — matches how they like to learn.",
		"session_length":      "This content suits their preference for deep, focused sessions.",
		"motivation":          "This resource rewards discovery — their favourite way to learn.",
		"solo_collaborative":  "This works well with others — just how they learn best.",
		"structure":           "This follows a clear structure they thrive with.",
		"outdoor_kinesthetic": "This gets them moving — they think better that way.",
	}
	why := whyTemplates[bestDim]
	if why == "" {
		why = "This content is a strong match for their learning style."
	}

	return float32(fitScore), why, true
}

// ─── Jaccard Similarity [13-recs §10.9] ─────────────────────────────────────

// computeJaccardSimilarity computes the Jaccard similarity coefficient between two
// string slices (treated as sets). Returns 0.0 for empty inputs. [13-recs §10.9]
func computeJaccardSimilarity(a, b []string) float32 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	setA := make(map[string]struct{}, len(a))
	for _, v := range a {
		setA[v] = struct{}{}
	}

	var intersection int
	setB := make(map[string]struct{}, len(b))
	for _, v := range b {
		setB[v] = struct{}{}
		if _, ok := setA[v]; ok {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 0.0
	}
	return float32(intersection) / float32(union)
}

// ─── Freshness Decay [13-recs §10.9] ────────────────────────────────────────

// computeFreshness returns an exponential decay score for content age.
// Returns 1.0 for brand-new content, decaying to ~0.0 over 90 days.
// Uses lambda=0.03 (half-life ~23 days). [13-recs §10.4]
func computeFreshness(createdAt time.Time, now time.Time) float32 {
	days := now.Sub(createdAt).Hours() / 24.0
	if days < 0 {
		days = 0
	}
	return float32(math.Exp(-0.03 * days))
}
