package discover

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"

	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// shareIDAlphabet is the base62 character set used for quiz result share IDs. [03-discover §15.11]
// URL-safe; avoids ambiguous characters (no '+', '/', '=').
const shareIDAlphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

// shareIDLength is the length of a share ID in characters. [03-discover §15.11]
const shareIDLength = 12

// ─── Service Implementation ───────────────────────────────────────────────────

// discoveryServiceImpl implements DiscoveryService.
type discoveryServiceImpl struct {
	quizDefRepo QuizDefinitionRepository
	quizResRepo QuizResultRepository
	stateRepo   StateGuideRepository
	contentRepo ContentPageRepository
	methodology MethodologyServiceForDiscover
}

// NewDiscoveryService creates a new DiscoveryService.
// Constructor returns the interface type per [CODING §2.1].
func NewDiscoveryService(
	quizDefRepo QuizDefinitionRepository,
	quizResRepo QuizResultRepository,
	stateRepo StateGuideRepository,
	contentRepo ContentPageRepository,
	methodSvc MethodologyServiceForDiscover,
) DiscoveryService {
	return &discoveryServiceImpl{
		quizDefRepo: quizDefRepo,
		quizResRepo: quizResRepo,
		stateRepo:   stateRepo,
		contentRepo: contentRepo,
		methodology: methodSvc,
	}
}

// ─── GetActiveQuiz ────────────────────────────────────────────────────────────

// GetActiveQuiz returns the active quiz with answer weights stripped. [03-discover §8.1]
func (s *discoveryServiceImpl) GetActiveQuiz(ctx context.Context) (*QuizResponse, error) {
	def, err := s.quizDefRepo.FindActive(ctx)
	if err != nil {
		return nil, err
	}
	return stripWeights(def)
}

// stripWeights converts a QuizDefinition (with internal weights) into a
// QuizResponse (without weights). This is the critical privacy boundary that
// prevents scoring data from leaking to clients. [03-discover §3.1, §15.1]
func stripWeights(def *QuizDefinition) (*QuizResponse, error) {
	var questions []quizQuestionInternal
	if err := json.Unmarshal(def.Questions, &questions); err != nil {
		return nil, shared.ErrInternal(err)
	}

	publicQuestions := make([]QuizQuestionResponse, len(questions))
	for i, q := range questions {
		answers := make([]QuizAnswerResponse, len(q.Answers))
		for j, a := range q.Answers {
			// Only copy ID and Text — weights are intentionally omitted.
			answers[j] = QuizAnswerResponse{ID: a.ID, Text: a.Text}
		}
		publicQuestions[i] = QuizQuestionResponse{
			ID:       q.ID,
			Category: q.Category,
			Text:     q.Text,
			HelpText: q.HelpText,
			Answers:  answers,
		}
	}

	return &QuizResponse{
		QuizID:      def.ID,
		Version:     def.Version,
		Title:       def.Title,
		Description: def.Description,
		Questions:   publicQuestions,
	}, nil
}

// ─── SubmitQuiz ───────────────────────────────────────────────────────────────

// SubmitQuiz runs the scoring engine on the submitted answers, persists the result,
// and returns a QuizResultResponse with a share_id. [03-discover §8.2]
//
// Scoring algorithm:
//  1. Get the active quiz definition (with weights).
//  2. Validate each submitted answer (invalid question/answer ID → 422).
//  3. Accumulate raw scores per methodology slug.
//  4. Compute max-possible score per methodology (sum of max weight across all questions).
//  5. Normalize: normalized = raw / max_possible.
//  6. Convert to uint8 percentage via math.Round(normalized * 100).
//  7. Rank by percentage descending, then slug for determinism on ties. [§15.26]
//  8. Look up explanations JSONB: match_text if score >= 0.5, else mismatch_text.
//  9. Fetch display names from method:: service.
//  10. Generate share_id (12-char base62 nanoid).
//  11. Persist and return.
func (s *discoveryServiceImpl) SubmitQuiz(ctx context.Context, cmd SubmitQuizCommand) (*QuizResultResponse, error) {
	def, err := s.quizDefRepo.FindActive(ctx)
	if err != nil {
		return nil, err
	}

	var questions []quizQuestionInternal
	if err := json.Unmarshal(def.Questions, &questions); err != nil {
		return nil, shared.ErrInternal(err)
	}

	// Build lookup maps for fast validation.
	questionMap := make(map[string]*quizQuestionInternal, len(questions))
	for i := range questions {
		questionMap[questions[i].ID] = &questions[i]
	}

	// Validate submitted answers.
	for qID, aID := range cmd.Answers {
		q, ok := questionMap[qID]
		if !ok {
			return nil, &DiscoverError{Err: ErrInvalidQuestionID, QuestionID: qID}
		}
		found := false
		for _, a := range q.Answers {
			if a.ID == aID {
				found = true
				break
			}
		}
		if !found {
			return nil, &DiscoverError{Err: ErrInvalidAnswerID, AnswerID: aID}
		}
	}

	// Accumulate raw scores and compute max-possible per methodology.
	rawScores := make(map[string]float64)
	maxPossible := make(map[string]float64)

	for i := range questions {
		q := &questions[i]
		// Accumulate submitted answer's weights.
		if aID, submitted := cmd.Answers[q.ID]; submitted {
			for _, a := range q.Answers {
				if a.ID == aID {
					for slug, w := range a.Weights {
						rawScores[slug] += w
					}
					break
				}
			}
		}
		// Per question, find the max weight for each methodology.
		qMax := make(map[string]float64)
		for _, a := range q.Answers {
			for slug, w := range a.Weights {
				if w > qMax[slug] {
					qMax[slug] = w
				}
			}
		}
		for slug, m := range qMax {
			maxPossible[slug] += m
		}
	}

	// Normalize and convert to percentage.
	type scoredEntry struct {
		slug       string
		normalized float64
		pct        uint8
	}
	entries := make([]scoredEntry, 0, len(maxPossible))
	for slug, maxP := range maxPossible {
		if maxP == 0 {
			continue
		}
		norm := rawScores[slug] / maxP
		pct := uint8(math.Round(norm * 100))
		if pct > 100 {
			pct = 100 // guard against floating-point edge cases
		}
		entries = append(entries, scoredEntry{slug: slug, normalized: norm, pct: pct})
	}

	// Sort by percentage descending, then by slug for determinism on ties. [03-discover §15.26]
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].pct != entries[j].pct {
			return entries[i].pct > entries[j].pct
		}
		return entries[i].slug < entries[j].slug
	})

	// Parse explanations JSONB.
	var explanations quizExplanationsInternal
	if len(def.Explanations) > 0 {
		if err := json.Unmarshal(def.Explanations, &explanations); err != nil {
			return nil, shared.ErrInternal(err)
		}
	}

	// Build recommendations with display names from method:: service.
	recommendations := make([]MethodologyRecommendation, 0, len(entries))
	for i, entry := range entries {
		displayName, dnErr := s.methodology.GetMethodologyDisplayName(ctx, entry.slug)
		if dnErr != nil || displayName == "" {
			displayName = entry.slug // graceful fallback to slug [03-discover §15.27]
		}

		explanation := ""
		if exp, ok := explanations[entry.slug]; ok {
			if entry.normalized >= 0.5 {
				explanation = exp.MatchText
			} else {
				explanation = exp.MismatchText
			}
		}

		recommendations = append(recommendations, MethodologyRecommendation{
			Rank:            uint8(i + 1),
			MethodologySlug: entry.slug,
			MethodologyName: displayName,
			ScorePercentage: entry.pct,
			Explanation:     explanation,
		})
	}

	// Generate share ID (12-char base62 nanoid). [03-discover §15.11]
	shareID, err := gonanoid.Generate(shareIDAlphabet, shareIDLength)
	if err != nil {
		return nil, shared.ErrInternal(err)
	}

	// Marshal scores and recommendations for storage.
	scoreMap := make(map[string]uint8, len(entries))
	for _, entry := range entries {
		scoreMap[entry.slug] = entry.pct
	}
	scoresJSON, err := json.Marshal(scoreMap)
	if err != nil {
		return nil, shared.ErrInternal(err)
	}

	recsJSON, err := json.Marshal(recommendations)
	if err != nil {
		return nil, shared.ErrInternal(err)
	}

	answersJSON, err := json.Marshal(cmd.Answers)
	if err != nil {
		return nil, shared.ErrInternal(err)
	}

	stored, err := s.quizResRepo.Create(ctx, CreateQuizResult{
		QuizDefinitionID: def.ID,
		ShareID:          shareID,
		SessionToken:     cmd.SessionToken,
		Answers:          json.RawMessage(answersJSON),
		Scores:           json.RawMessage(scoresJSON),
		Recommendations:  json.RawMessage(recsJSON),
		QuizVersion:      int16(def.Version),
	})
	if err != nil {
		return nil, err
	}

	return &QuizResultResponse{
		ShareID:         shareID,
		QuizVersion:     int16(def.Version),
		CreatedAt:       stored.CreatedAt,
		IsClaimed:       false,
		Recommendations: recommendations,
	}, nil
}

// ─── GetQuizResult ────────────────────────────────────────────────────────────

// GetQuizResult returns a stored quiz result by share_id. [03-discover §8.2]
func (s *discoveryServiceImpl) GetQuizResult(ctx context.Context, shareID string) (*QuizResultResponse, error) {
	result, err := s.quizResRepo.FindByShareID(ctx, shareID)
	if err != nil {
		return nil, err
	}

	var recommendations []MethodologyRecommendation
	if err := json.Unmarshal(result.Recommendations, &recommendations); err != nil {
		return nil, shared.ErrInternal(err)
	}
	if recommendations == nil {
		recommendations = []MethodologyRecommendation{}
	}

	return &QuizResultResponse{
		ShareID:         result.ShareID,
		QuizVersion:     result.QuizVersion,
		CreatedAt:       result.CreatedAt,
		IsClaimed:       result.FamilyID != nil,
		Recommendations: recommendations,
	}, nil
}

// ─── ListStateGuides ──────────────────────────────────────────────────────────

// ListStateGuides returns a summary of all 51 state guides. [03-discover §8.3]
func (s *discoveryServiceImpl) ListStateGuides(ctx context.Context) ([]StateGuideSummaryResponse, error) {
	summaries, err := s.stateRepo.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	resp := make([]StateGuideSummaryResponse, len(summaries))
	for i, sg := range summaries {
		resp[i] = StateGuideSummaryResponse{
			StateCode:      sg.StateCode,
			StateName:      sg.StateName,
			IsAvailable:    sg.Status == "published",
			LastReviewedAt: sg.LastReviewedAt,
		}
	}
	return resp, nil
}

// ─── GetStateGuide ────────────────────────────────────────────────────────────

// GetStateGuide returns the full state guide for a state code.
// Returns 404 for unpublished or missing guides — existence is not secret. [03-discover §3.2, §15.16]
func (s *discoveryServiceImpl) GetStateGuide(ctx context.Context, stateCode string) (*StateGuideResponse, error) {
	if len(strings.TrimSpace(stateCode)) != 2 {
		return nil, &DiscoverError{Err: ErrInvalidStateCode, StateCode: stateCode}
	}

	guide, err := s.stateRepo.FindByStateCode(ctx, stateCode)
	if err != nil {
		return nil, err
	}

	var reqs StateGuideRequirements
	if len(guide.Requirements) > 0 && string(guide.Requirements) != "{}" {
		if err := json.Unmarshal(guide.Requirements, &reqs); err != nil {
			return nil, shared.ErrInternal(err)
		}
	}
	if reqs.RequiredSubjects == nil {
		reqs.RequiredSubjects = []string{}
	}

	return &StateGuideResponse{
		StateCode:       guide.StateCode,
		StateName:       guide.StateName,
		Requirements:    reqs,
		GuideContent:    guide.GuideContent,
		LegalDisclaimer: guide.LegalDisclaimer,
		LastReviewedAt:  guide.LastReviewedAt,
	}, nil
}

// ─── ClaimQuizResult ──────────────────────────────────────────────────────────

// ClaimQuizResult delegates to the quiz result repository. [04-onboard §9.4]
func (s *discoveryServiceImpl) ClaimQuizResult(ctx context.Context, shareID string, familyID any) error {
	return s.quizResRepo.ClaimForFamily(ctx, shareID, familyID)
}

// ─── GetContentBySlug ─────────────────────────────────────────────────────────

// GetContentBySlug returns a published content page by its slug. [03-discover §8.4]
func (s *discoveryServiceImpl) GetContentBySlug(ctx context.Context, slug string) (*ContentPage, error) {
	page, err := s.contentRepo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return page, nil
}

// ─── GetStateRequirements ─────────────────────────────────────────────────────

// GetStateRequirements returns structured requirements for a state for comply:: consumption.
// Returns requirements regardless of guide published status. [03-discover §5, §13.1]
func (s *discoveryServiceImpl) GetStateRequirements(ctx context.Context, stateCode string) (*StateGuideRequirements, error) {
	if len(strings.TrimSpace(stateCode)) != 2 {
		return nil, &DiscoverError{Err: ErrInvalidStateCode, StateCode: stateCode}
	}

	guide, err := s.stateRepo.FindRequirementsByStateCode(ctx, stateCode)
	if err != nil {
		return nil, err
	}

	var reqs StateGuideRequirements
	if len(guide.Requirements) > 0 && string(guide.Requirements) != "{}" {
		if err := json.Unmarshal(guide.Requirements, &reqs); err != nil {
			return nil, shared.ErrInternal(err)
		}
	}
	if reqs.RequiredSubjects == nil {
		reqs.RequiredSubjects = []string{}
	}

	return &reqs, nil
}

// ─── HandleFamilyDeletionScheduled ────────────────────────────────────────────

// HandleFamilyDeletionScheduled clears family association from quiz results.
// Quiz results are made anonymous again (family_id = NULL) rather than deleted,
// since they may be referenced by share_id from external sources. [15-data-lifecycle §7]
func (s *discoveryServiceImpl) HandleFamilyDeletionScheduled(ctx context.Context, familyID any) error {
	return s.quizResRepo.UnclaimByFamilyID(ctx, familyID)
}
