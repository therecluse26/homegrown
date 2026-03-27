package billing

import (
	"errors"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Sentinel errors for the billing domain. [10-billing §15]
var (
	// ─── Subscription Errors ────────────────────────────────────────────
	ErrSubscriptionNotFound      = errors.New("subscription not found")
	ErrSubscriptionAlreadyExists = errors.New("subscription already exists for this family")
	ErrCannotReactivate          = errors.New("cannot reactivate subscription in current state")
	ErrSubscriptionNotActive     = errors.New("subscription is not active")
	ErrSubscriptionNotPaused     = errors.New("subscription is not paused")
	ErrInvalidBillingInterval    = errors.New("invalid billing interval")

	// ─── Payment Method Errors ──────────────────────────────────────────
	ErrPaymentMethodNotFound         = errors.New("payment method not found")
	ErrCannotRemoveLastPaymentMethod = errors.New("cannot remove last payment method with active subscription")

	// ─── Payment Errors ─────────────────────────────────────────────────
	ErrPaymentDeclined         = errors.New("payment was declined")
	ErrCoppaVerificationFailed = errors.New("COPPA verification failed")

	// ─── Adapter Errors ─────────────────────────────────────────────────
	ErrPaymentAdapterUnavailable = errors.New("payment adapter unavailable")
	ErrInvalidWebhookSignature   = errors.New("invalid webhook signature")

	// ─── Infrastructure ─────────────────────────────────────────────────
	ErrDatabaseError = errors.New("database error")
	ErrAdapterError  = errors.New("adapter error")
)

// BillingError wraps a sentinel error with optional context. [CODING §2.2]
type BillingError struct {
	Err error
}

func (e *BillingError) Error() string { return e.Err.Error() }
func (e *BillingError) Unwrap() error { return e.Err }

// toAppError maps a BillingError sentinel to an *shared.AppError with the correct
// HTTP status code. Called by mapBillingError in handler.go. [10-billing §15]
func (e *BillingError) toAppError() *shared.AppError {
	switch {
	case errors.Is(e.Err, ErrSubscriptionNotFound):
		return &shared.AppError{Code: "subscription_not_found", Message: "Subscription not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrSubscriptionAlreadyExists):
		return &shared.AppError{Code: "subscription_exists", Message: "A subscription already exists for this family", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrCannotReactivate):
		return &shared.AppError{Code: "cannot_reactivate", Message: "Subscription cannot be reactivated in its current state", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrSubscriptionNotActive):
		return &shared.AppError{Code: "subscription_not_active", Message: "Subscription is not currently active", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrSubscriptionNotPaused):
		return &shared.AppError{Code: "subscription_not_paused", Message: "Subscription is not currently paused", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrInvalidBillingInterval):
		return &shared.AppError{Code: "invalid_billing_interval", Message: "Invalid billing interval — must be 'monthly' or 'annual'", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrPaymentMethodNotFound):
		return &shared.AppError{Code: "payment_method_not_found", Message: "Payment method not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrCannotRemoveLastPaymentMethod):
		return &shared.AppError{Code: "cannot_remove_last_payment_method", Message: "Cannot remove the only payment method while a subscription is active", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrPaymentDeclined):
		return &shared.AppError{Code: "payment_declined", Message: "Payment was declined — please try a different payment method", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrCoppaVerificationFailed):
		return &shared.AppError{Code: "coppa_verification_failed", Message: "Parental verification failed — please try again", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrPaymentAdapterUnavailable):
		return &shared.AppError{Code: "payment_adapter_unavailable", Message: "Payment service is temporarily unavailable", StatusCode: http.StatusBadGateway}
	default:
		return shared.ErrInternal(e)
	}
}

// mapBillingError maps any error to an Echo-compatible HTTP error.
// If err is a *BillingError, maps it to an AppError via toAppError().
// Otherwise returns it as-is for Echo's default error handling.
func mapBillingError(err error) error {
	var be *BillingError
	if errors.As(err, &be) {
		return be.toAppError()
	}
	return err
}
