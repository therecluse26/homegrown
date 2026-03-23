package onboard

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Sentinel errors for the onboard domain. [04-onboard §12]
// Handlers convert these to AppError via mapOnboardError(). [§12.1]
var (
	ErrWizardNotFound           = errors.New("wizard not found")
	ErrWizardAlreadyCompleted   = errors.New("wizard already completed")
	ErrWizardAlreadySkipped     = errors.New("wizard already skipped")
	ErrWizardNotInProgress      = errors.New("wizard is not in progress")
	ErrRequiredStepsMissing     = errors.New("required steps not completed")
	ErrInvalidMethodologyPath   = errors.New("invalid methodology path")
	ErrSecondaryWithoutAck      = errors.New("secondary methodologies require explanation_acknowledged")
	ErrQuizResultNotFound       = errors.New("quiz result not found")
	ErrQuizAlreadyImported      = errors.New("quiz already imported")
	ErrNoRoadmapItems           = errors.New("no roadmap items available")
	ErrNoRecommendations        = errors.New("no recommendations available")
	ErrNoCommunity              = errors.New("no community suggestions available")
	ErrInvalidMethodologySlug   = errors.New("invalid methodology slug")
)

// OnboardError wraps a sentinel error with optional context fields. [CODING §2.2]
type OnboardError struct {
	Err  error
	Step string
	Slug string
}

func (e *OnboardError) Error() string {
	switch {
	case e.Step != "":
		return fmt.Sprintf("%s: step=%s", e.Err.Error(), e.Step)
	case e.Slug != "":
		return fmt.Sprintf("%s: slug=%s", e.Err.Error(), e.Slug)
	default:
		return e.Err.Error()
	}
}

func (e *OnboardError) Unwrap() error { return e.Err }

// toAppError maps an OnboardError sentinel to an *shared.AppError with the correct
// HTTP status code. Called by mapOnboardError in handler.go. [04-onboard §12.1]
func (e *OnboardError) toAppError() *shared.AppError {
	switch {
	case errors.Is(e.Err, ErrWizardNotFound):
		return &shared.AppError{Code: "wizard_not_found", Message: "Onboarding wizard not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrWizardAlreadyCompleted):
		return &shared.AppError{Code: "wizard_already_completed", Message: "Onboarding wizard is already completed", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrWizardAlreadySkipped):
		return &shared.AppError{Code: "wizard_already_skipped", Message: "Onboarding wizard was skipped", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrWizardNotInProgress):
		return &shared.AppError{Code: "wizard_not_in_progress", Message: "Onboarding wizard is not in progress", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrRequiredStepsMissing):
		return &shared.AppError{Code: "required_steps_missing", Message: "Required onboarding steps are not completed", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrInvalidMethodologyPath):
		return &shared.AppError{Code: "invalid_methodology_path", Message: "Invalid methodology selection path", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrSecondaryWithoutAck):
		return &shared.AppError{Code: "secondary_without_ack", Message: "Secondary methodologies require acknowledgment of multi-methodology model", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrQuizResultNotFound):
		return &shared.AppError{Code: "quiz_result_not_found", Message: "Quiz result not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrQuizAlreadyImported):
		return &shared.AppError{Code: "quiz_already_imported", Message: "A quiz has already been imported for this wizard", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrNoRoadmapItems):
		return &shared.AppError{Code: "no_roadmap_items", Message: "No roadmap items available yet — complete methodology selection first", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrNoRecommendations):
		return &shared.AppError{Code: "no_recommendations", Message: "No recommendations available yet — complete methodology selection first", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrNoCommunity):
		return &shared.AppError{Code: "no_community_suggestions", Message: "No community suggestions available yet — complete methodology selection first", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrInvalidMethodologySlug):
		return &shared.AppError{Code: "invalid_methodology_slug", Message: "One or more methodology slugs are invalid", StatusCode: http.StatusUnprocessableEntity}
	default:
		return shared.ErrInternal(e)
	}
}
