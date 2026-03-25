package safety

import (
	"errors"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/safety/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// SafetyError wraps a sentinel error with optional context. [CODING §2.2]
type SafetyError struct {
	Err error
}

func (e *SafetyError) Error() string { return e.Err.Error() }
func (e *SafetyError) Unwrap() error { return e.Err }

// ─── Sentinel Errors ────────────────────────────────────────────────────────────
// All safety domain sentinel errors per 11-safety §15.

var (
	ErrReportNotFound   = errors.New("report not found")
	ErrFlagNotFound     = errors.New("content flag not found")
	ErrActionNotFound   = errors.New("moderation action not found")
	ErrAppealNotFound   = errors.New("appeal not found")
	ErrDuplicateReport  = errors.New("duplicate report within 24 hours")
	ErrAppealAlreadyExists = errors.New("appeal already exists for this action")
	ErrSameAdminAppeal  = errors.New("appeal must be reviewed by a different admin")
	ErrFlagAlreadyReviewed = errors.New("flag has already been reviewed")

	// Re-exported domain errors for convenience.
	ErrInvalidReportTransition = domain.ErrInvalidReportTransition
	ErrAccountBanned           = domain.ErrAccountBanned
	ErrAccountSuspended        = domain.ErrAccountSuspended
	ErrCsamBanNotAppealable    = domain.ErrCsamBanNotAppealable
	ErrInvalidActionType       = domain.ErrInvalidActionType
)

// toAppError maps a SafetyError sentinel to a *shared.AppError. [11-safety §15.1]
func (e *SafetyError) toAppError() *shared.AppError {
	switch {
	case errors.Is(e.Err, ErrReportNotFound):
		return &shared.AppError{Code: "report_not_found", Message: "Report not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrFlagNotFound):
		return &shared.AppError{Code: "flag_not_found", Message: "Content flag not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrActionNotFound):
		return &shared.AppError{Code: "action_not_found", Message: "Moderation action not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrAppealNotFound):
		return &shared.AppError{Code: "appeal_not_found", Message: "Appeal not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrDuplicateReport):
		return &shared.AppError{Code: "duplicate_report", Message: "You have already reported this content recently", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrAppealAlreadyExists):
		return &shared.AppError{Code: "appeal_exists", Message: "An appeal already exists for this action", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrCsamBanNotAppealable):
		return &shared.AppError{Code: "csam_ban_not_appealable", Message: "This type of ban cannot be appealed", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrSameAdminAppeal):
		return &shared.AppError{Code: "same_admin_appeal", Message: "Appeal must be reviewed by a different moderator", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrInvalidReportTransition):
		return &shared.AppError{Code: "invalid_report_transition", Message: "Invalid report status transition", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrFlagAlreadyReviewed):
		return &shared.AppError{Code: "flag_already_reviewed", Message: "This flag has already been reviewed", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrAccountSuspended):
		return &shared.AppError{Code: "account_suspended", Message: "Account is suspended", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, ErrAccountBanned):
		return &shared.AppError{Code: "account_banned", Message: "Account is banned", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, ErrInvalidActionType):
		return &shared.AppError{Code: "invalid_action_type", Message: "Invalid action for current state", StatusCode: http.StatusUnprocessableEntity}
	default:
		return shared.ErrInternal(e)
	}
}

// mapSafetyError converts domain errors to AppError for HTTP responses. [11-safety §15.1]
func mapSafetyError(err error) *shared.AppError {
	var safetyErr *SafetyError
	if errors.As(err, &safetyErr) {
		return safetyErr.toAppError()
	}
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return shared.ErrInternal(err)
}
