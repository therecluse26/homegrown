package domain

import "github.com/google/uuid"

// Quiz session status constants. [06-learn §8.1.10]
const (
	QuizStatusNotStarted = "not_started"
	QuizStatusInProgress = "in_progress"
	QuizStatusSubmitted  = "submitted"
	QuizStatusScored     = "scored"
)

// validQuizSessionTransitions defines allowed status transitions.
// not_started → in_progress → submitted → scored
// in_progress can also jump straight to scored (all auto-scorable).
var validQuizSessionTransitions = map[string]map[string]bool{
	QuizStatusNotStarted: {QuizStatusInProgress: true},
	QuizStatusInProgress: {QuizStatusSubmitted: true, QuizStatusScored: true},
	QuizStatusSubmitted:  {QuizStatusScored: true},
}

// ValidateQuizSessionTransition checks if a quiz session status transition is valid.
func ValidateQuizSessionTransition(from, to string) error {
	allowed, ok := validQuizSessionTransitions[from]
	if !ok || !allowed[to] {
		return ErrQuizSessionAlreadySubmitted
	}
	return nil
}

// QuizQuestionInfo holds the minimal data needed for quiz scoring computation.
type QuizQuestionInfo struct {
	QuestionID     uuid.UUID
	Points         float64
	PointsOverride *float64
	AutoScorable   bool
}

// effectivePoints returns the points for a question, respecting overrides.
func effectivePoints(q QuizQuestionInfo) float64 {
	if q.PointsOverride != nil {
		return *q.PointsOverride
	}
	return q.Points
}

// AutoScoreQuiz computes the max score and whether all questions are auto-scorable.
// Used when a student submits a quiz to determine if auto-scoring can complete it.
func AutoScoreQuiz(questions []QuizQuestionInfo) (maxScore float64, allAutoScorable bool) {
	allAutoScorable = true
	for _, q := range questions {
		maxScore += effectivePoints(q)
		if !q.AutoScorable {
			allAutoScorable = false
		}
	}
	return maxScore, allAutoScorable
}

// ComputeParentScore computes the total and max score for a parent-scored quiz.
// Auto-scorable questions receive full points; non-auto-scorable questions use
// the parent-provided scores from the parentScores map.
func ComputeParentScore(questions []QuizQuestionInfo, parentScores map[uuid.UUID]float64) (totalScore, maxScore float64) {
	for _, q := range questions {
		pts := effectivePoints(q)
		maxScore += pts
		if q.AutoScorable {
			totalScore += pts
		} else if awarded, ok := parentScores[q.QuestionID]; ok {
			totalScore += awarded
		}
	}
	return totalScore, maxScore
}
