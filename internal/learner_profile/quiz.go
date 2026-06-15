package learner_profile

import "strings"

// quiz.go: question definitions, dimension→vector computation, and fit scoring.
// [18-learner-profile §5, §6]

// ─── Question Definitions ─────────────────────────────────────────────────────

// QuizQuestionDef defines a single scored quiz question.
type QuizQuestionDef struct {
	ID        int
	Dimension string // which learner_profiles column this scores
	// ParentText and ChildText are displayed based on the respondent mode.
	ParentText string
	ChildText  string
}

// scoredQuestions lists the 12 scored questions in order.
// Each dimension has exactly 2 questions; per-dimension value = mean of the two.
var scoredQuestions = []QuizQuestionDef{
	// activity_format (0=text/listen, 1=hands-on/build/move)
	{ID: 1, Dimension: "activity_format",
		ParentText: "How does {name} prefer to engage with learning material?",
		ChildText:  "How do YOU like to learn something new?"},
	{ID: 2, Dimension: "activity_format",
		ParentText: "When {name} learns something new, they usually…",
		ChildText:  "When you learn something new, you usually…"},

	// session_length (0=short-bursts, 1=long-deep-dives)
	{ID: 3, Dimension: "session_length",
		ParentText: "How long can {name} focus on one subject before needing a break?",
		ChildText:  "How long do you like to work on one thing before taking a break?"},
	{ID: 4, Dimension: "session_length",
		ParentText: "When {name} starts an interesting project, they typically…",
		ChildText:  "When you start something exciting, you usually…"},

	// motivation (0=mastery, 1=discovery)
	{ID: 5, Dimension: "motivation",
		ParentText: "What feels more rewarding to {name}?",
		ChildText:  "What feels better to you?"},
	{ID: 6, Dimension: "motivation",
		ParentText: "When {name} struggles with something, they usually…",
		ChildText:  "When something is hard for you, you usually…"},

	// solo_collaborative (0=solo, 1=collaborative)
	{ID: 7, Dimension: "solo_collaborative",
		ParentText: "Does {name} prefer to learn alone or with others?",
		ChildText:  "Do you like learning alone or with friends?"},
	{ID: 8, Dimension: "solo_collaborative",
		ParentText: "When working on a project, {name} usually prefers…",
		ChildText:  "When you do a big project, you prefer…"},

	// structure (0=step-by-step, 1=open-ended)
	{ID: 9, Dimension: "structure",
		ParentText: "How does {name} prefer to approach a new topic?",
		ChildText:  "How do you like to start learning something new?"},
	{ID: 10, Dimension: "structure",
		ParentText: "When learning, {name} works best with…",
		ChildText:  "You work best when…"},

	// outdoor_kinesthetic (0=not-important, 1=think-better-moving)
	{ID: 11, Dimension: "outdoor_kinesthetic",
		ParentText: "How important is physical movement or being outdoors to {name}'s learning?",
		ChildText:  "How important is moving around or being outside when you learn?"},
	{ID: 12, Dimension: "outdoor_kinesthetic",
		ParentText: "Does {name} concentrate better sitting quietly indoors or while moving or outside?",
		ChildText:  "Do you focus better sitting still or when you can move around?"},
}

// DimensionOrder controls the display and computation order. Exported for frontend
// quiz orchestration via generated types.
var DimensionOrder = []string{
	"activity_format",
	"session_length",
	"motivation",
	"solo_collaborative",
	"structure",
	"outdoor_kinesthetic",
}

// InterestChip represents a single interest option in the multi-select question.
type InterestChip struct {
	ID         string // displayed chip ID (matches the stored interest value)
	Label      string // display label
	SubjectTag string // maps to mkt_listings.subject_tags vocabulary
}

// InterestChips is the ordered list of interest options for Q13/Q14.
var InterestChips = []InterestChip{
	{ID: "animals", Label: "Animals", SubjectTag: "nature_study"},
	{ID: "art", Label: "Art & Drawing", SubjectTag: "art"},
	{ID: "building", Label: "Building & Making", SubjectTag: "crafts"},
	{ID: "coding", Label: "Coding", SubjectTag: "indoor_science"},
	{ID: "cooking", Label: "Cooking", SubjectTag: "cooking"},
	{ID: "drama", Label: "Drama & Acting", SubjectTag: "art"},
	{ID: "history", Label: "History", SubjectTag: "history"},
	{ID: "language", Label: "Language & Writing", SubjectTag: "writing"},
	{ID: "math", Label: "Math", SubjectTag: "mathematics"},
	{ID: "music", Label: "Music", SubjectTag: "music"},
	{ID: "nature", Label: "Nature & Outdoors", SubjectTag: "ecology"},
	{ID: "reading", Label: "Reading", SubjectTag: "reading"},
	{ID: "science", Label: "Science", SubjectTag: "indoor_science"},
	{ID: "space", Label: "Space & Astronomy", SubjectTag: "astronomy"},
	{ID: "sport", Label: "Sports & Movement", SubjectTag: "physical_education"},
}

// interestToSubjectTag maps interest chip IDs to their primary subject_tag.
var interestToSubjectTag map[string]string

func init() {
	interestToSubjectTag = make(map[string]string, len(InterestChips))
	for _, c := range InterestChips {
		interestToSubjectTag[c.ID] = c.SubjectTag
	}
}

// ─── Dimension Vector Computation ────────────────────────────────────────────

// DimensionVector holds the computed per-dimension values from quiz answers.
type DimensionVector struct {
	ActivityFormat     *float64
	SessionLength      *float64
	Motivation         *float64
	SoloCollaborative  *float64
	Structure          *float64
	OutdoorKinesthetic *float64
	AnsweredCount      int16
	Confidence         float64
}

// ComputeVector computes the dimension vector from a set of quiz answers.
// Per-dimension value = mean of answered questions covering that dimension.
// Skipped questions (nil Value) are excluded from the mean.
func ComputeVector(answers []QuizAnswer) DimensionVector {
	// Map question_id → value
	answerMap := make(map[int]float64, len(answers))
	for _, a := range answers {
		if a.Value != nil {
			answerMap[a.QuestionID] = *a.Value
		}
	}

	// Accumulate per-dimension sums
	dimSum := make(map[string]float64)
	dimCount := make(map[string]int)
	for _, q := range scoredQuestions {
		if v, ok := answerMap[q.ID]; ok {
			dimSum[q.Dimension] += v
			dimCount[q.Dimension]++
		}
	}

	// Compute means and answered_count
	var answeredCount int16
	for _, q := range scoredQuestions {
		if _, ok := answerMap[q.ID]; ok {
			answeredCount++
		}
	}

	mean := func(dim string) *float64 {
		if dimCount[dim] == 0 {
			return nil
		}
		v := dimSum[dim] / float64(dimCount[dim])
		return &v
	}

	confidence := float64(answeredCount) / 12.0

	return DimensionVector{
		ActivityFormat:     mean("activity_format"),
		SessionLength:      mean("session_length"),
		Motivation:         mean("motivation"),
		SoloCollaborative:  mean("solo_collaborative"),
		Structure:          mean("structure"),
		OutdoorKinesthetic: mean("outdoor_kinesthetic"),
		AnsweredCount:      answeredCount,
		Confidence:         confidence,
	}
}

// ─── Fit Score Computation ───────────────────────────────────────────────────

// FitResult holds the computed fit score and why-text for a content item.
type FitResult struct {
	Score   float64
	WhyText string
}

// fitWhyTemplates maps dimension name → why-text template ({name} placeholder).
var fitWhyTemplates = map[string]string{
	"activity_format":     "{name} loves hands-on, build-it learning.",
	"session_length":      "{name} gets absorbed — long, deep-dive content is their sweet spot.",
	"motivation":          "{name} is driven by discovery over mastery drills.",
	"solo_collaborative":  "{name} learns well with others.",
	"structure":           "{name} thrives with step-by-step structure.",
	"outdoor_kinesthetic": "{name} thinks better when moving.",
}

// ComputeFitScore computes fit_score and selects why-text for one content item.
// preferenceTags: map of dimension → content tag value (from mkt_listings.preference_tags).
// interests: student declared interests (from learner_profiles.interests).
// contentSubjectTags: content item's subject_tags array.
// studentName: substituted into the why-text template.
//
// Returns (score, whyText, ok). ok=false means badge gate not met (< 0.60 score
// or insufficient dimensional overlap).
func ComputeFitScore(
	vec DimensionVector,
	preferenceTags map[string]float64,
	interests []string,
	contentSubjectTags []string,
	studentName string,
) (score float64, whyText string, ok bool) {
	if len(preferenceTags) == 0 {
		return 0, "", false
	}

	type dimScore struct {
		dim   string
		score float64
	}

	var dimScores []dimScore
	var total float64

	profileDims := map[string]*float64{
		"activity_format":     vec.ActivityFormat,
		"session_length":      vec.SessionLength,
		"motivation":          vec.Motivation,
		"solo_collaborative":  vec.SoloCollaborative,
		"structure":           vec.Structure,
		"outdoor_kinesthetic": vec.OutdoorKinesthetic,
	}

	for dim, pVal := range profileDims {
		if pVal == nil {
			continue
		}
		cVal, hasTag := preferenceTags[dim]
		if !hasTag {
			continue
		}
		s := 1.0 - abs((*pVal) - cVal)
		dimScores = append(dimScores, dimScore{dim: dim, score: s})
		total += s
	}

	if len(dimScores) == 0 {
		return 0, "", false
	}

	fitScore := total / float64(len(dimScores))

	// Interest boost: +0.10 if any content subject_tag matches a declared interest
	contentTagSet := make(map[string]struct{}, len(contentSubjectTags))
	for _, t := range contentSubjectTags {
		contentTagSet[t] = struct{}{}
	}
	for _, interest := range interests {
		subjectTag := interestToSubjectTag[interest]
		if subjectTag == "" {
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

	// Badge gate: both dimensions required
	if fitScore < 0.60 {
		return fitScore, "", false
	}

	// Find highest-contributing dimension for why-text
	bestDim := ""
	var bestScore float64
	for _, ds := range dimScores {
		if ds.score > bestScore {
			bestScore = ds.score
			bestDim = ds.dim
		}
	}

	tmpl := fitWhyTemplates[bestDim]
	if tmpl == "" {
		tmpl = "{name} is a great match for this content."
	}
	why := strings.ReplaceAll(tmpl, "{name}", studentName)

	return fitScore, why, true
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
