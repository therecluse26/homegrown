package shared

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// AppError represents an application-level error with HTTP status mapping.
// All domain errors MUST convert to AppError before reaching the handler return. [CODING §2.2]
type AppError struct {
	Code       string
	Message    string
	StatusCode int
	Err        error // wrapped internal error, never exposed to client
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// ─── Error Constructors ──────────────────────────────────────────────────────

func ErrNotFound() *AppError {
	return &AppError{Code: "not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
}

func ErrUnauthorized() *AppError {
	return &AppError{Code: "unauthorized", Message: "Authentication required", StatusCode: http.StatusUnauthorized}
}

func ErrForbidden() *AppError {
	return &AppError{Code: "forbidden", Message: "Access denied", StatusCode: http.StatusForbidden}
}

func ErrPremiumRequired() *AppError {
	return &AppError{Code: "premium_required", Message: "Premium subscription required", StatusCode: http.StatusPaymentRequired}
}

func ErrCoppaConsentRequired() *AppError {
	return &AppError{Code: "coppa_consent_required", Message: "COPPA parental consent required", StatusCode: http.StatusForbidden}
}

func ErrValidation(msg string) *AppError {
	return &AppError{Code: "validation_error", Message: msg, StatusCode: http.StatusUnprocessableEntity}
}

func ErrConflict(msg string) *AppError {
	return &AppError{Code: "conflict", Message: msg, StatusCode: http.StatusConflict}
}

func ErrRateLimited() *AppError {
	return &AppError{Code: "rate_limited", Message: "Rate limit exceeded", StatusCode: http.StatusTooManyRequests}
}

func ErrBadRequest(msg string) *AppError {
	return &AppError{Code: "bad_request", Message: msg, StatusCode: http.StatusBadRequest}
}

func ErrAccountSuspended() *AppError {
	return &AppError{Code: "account_suspended", Message: "Your account has been temporarily suspended", StatusCode: http.StatusForbidden}
}

func ErrAccountBanned() *AppError {
	return &AppError{Code: "account_banned", Message: "Your account has been permanently restricted", StatusCode: http.StatusForbidden}
}

func ErrBadGateway(msg string) *AppError {
	return &AppError{Code: "bad_gateway", Message: msg, StatusCode: http.StatusBadGateway}
}

func ErrInternal(err error) *AppError {
	return &AppError{Code: "internal_error", Message: "An internal error occurred", StatusCode: http.StatusInternalServerError, Err: err}
}

func ErrDatabase(err error) *AppError {
	return &AppError{Code: "internal_error", Message: "An internal error occurred", StatusCode: http.StatusInternalServerError, Err: err}
}

// ─── HTTP Error Handler ──────────────────────────────────────────────────────

// ErrorResponse is the JSON structure returned for all errors.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody contains the machine-readable code and human-readable message.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// HTTPErrorHandler is a custom Echo error handler that maps AppError to JSON responses.
// Internal error details (Err field) are logged but NEVER exposed to the client. [CODING §2.2]
func HTTPErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		if appErr.Err != nil {
			slog.Error("internal server error", "error", appErr.Err)
		}
		_ = c.JSON(appErr.StatusCode, ErrorResponse{
			Error: ErrorBody{
				Code:    appErr.Code,
				Message: appErr.Message,
			},
		})
		return
	}

	// Fallback for non-AppError errors (e.g. Echo's own HTTPError)
	var echoErr *echo.HTTPError
	if errors.As(err, &echoErr) {
		msg := fmt.Sprintf("%v", echoErr.Message)
		_ = c.JSON(echoErr.Code, ErrorResponse{
			Error: ErrorBody{
				Code:    "error",
				Message: msg,
			},
		})
		return
	}

	slog.Error("unhandled error", "error", err)
	_ = c.JSON(http.StatusInternalServerError, ErrorResponse{
		Error: ErrorBody{
			Code:    "internal_error",
			Message: "An internal error occurred",
		},
	})
}

// ─── Validator Integration ────────────────────────────────────────────────────

// ValidationError converts go-playground/validator errors to an AppError.
// Returns a 422 Unprocessable Entity with human-readable field errors.
func ValidationError(err error) *AppError {
	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		msgs := make([]string, 0, len(ve))
		for _, fe := range ve {
			msgs = append(msgs, fmt.Sprintf("field '%s' failed on '%s' validation", fe.Field(), fe.Tag()))
		}
		return ErrValidation(strings.Join(msgs, "; "))
	}
	return ErrValidation(err.Error())
}
