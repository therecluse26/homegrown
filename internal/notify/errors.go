package notify

import (
	"errors"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Sentinel errors for the notify domain. [08-notify §16]
var (
	ErrNotificationNotFound       = errors.New("notification not found")
	ErrNotificationNotOwned       = errors.New("notification not owned by family")
	ErrCannotDisableSystemCritical = errors.New("cannot disable system-critical notification type")
	ErrInvalidNotificationType    = errors.New("invalid notification type")
	ErrInvalidChannel             = errors.New("invalid notification channel")
	ErrInvalidCategory            = errors.New("invalid notification category")
	ErrInvalidUnsubscribeToken    = errors.New("invalid or expired unsubscribe token")
	ErrDuplicateNotification      = errors.New("duplicate notification")
	ErrEmailDeliveryFailed        = errors.New("email delivery failed")
	ErrInvalidDigestFrequency     = errors.New("invalid digest frequency")
	ErrPreferenceNotFound         = errors.New("preference not found")
	ErrDigestNotReady             = errors.New("digest not ready")
)

// NotifyError wraps a sentinel error with optional context. [CODING §2.2]
type NotifyError struct {
	Err error
}

func (e *NotifyError) Error() string { return e.Err.Error() }
func (e *NotifyError) Unwrap() error { return e.Err }

// toAppError maps a NotifyError sentinel to an *shared.AppError with the correct
// HTTP status code. Called by mapNotifyError in handler.go. [08-notify §16]
func (e *NotifyError) toAppError() *shared.AppError {
	switch {
	case errors.Is(e.Err, ErrNotificationNotFound):
		return &shared.AppError{Code: "notification_not_found", Message: "Notification not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrNotificationNotOwned):
		// Return 404 for enumeration prevention — never reveal existence. [S§18]
		return &shared.AppError{Code: "notification_not_found", Message: "Notification not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrCannotDisableSystemCritical):
		return &shared.AppError{Code: "cannot_disable_system_critical", Message: "System-critical notifications cannot be disabled", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrInvalidNotificationType):
		return &shared.AppError{Code: "invalid_notification_type", Message: "Invalid notification type", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrInvalidChannel):
		return &shared.AppError{Code: "invalid_channel", Message: "Invalid notification channel", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrInvalidCategory):
		return &shared.AppError{Code: "invalid_category", Message: "Invalid notification category", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrInvalidUnsubscribeToken):
		return &shared.AppError{Code: "invalid_unsubscribe_token", Message: "Invalid or expired unsubscribe token", StatusCode: http.StatusBadRequest}
	case errors.Is(e.Err, ErrDuplicateNotification):
		return &shared.AppError{Code: "duplicate_notification", Message: "Notification already exists", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrInvalidDigestFrequency):
		return &shared.AppError{Code: "invalid_digest_frequency", Message: "Invalid digest frequency", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrEmailDeliveryFailed):
		return &shared.AppError{Code: "email_delivery_failed", Message: "Email delivery failed", StatusCode: http.StatusBadGateway}
	default:
		return shared.ErrInternal(e)
	}
}

// mapNotifyError maps any error to an Echo-compatible HTTP error.
// If err is a *NotifyError, maps it to an AppError via toAppError().
// Otherwise returns it as-is for Echo's default error handling.
func mapNotifyError(err error) error {
	var ne *NotifyError
	if errors.As(err, &ne) {
		return ne.toAppError()
	}
	return err
}
