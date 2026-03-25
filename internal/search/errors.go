package search

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Sentinel Errors ─────────────────────────────────────────────────────────

var (
	// ErrQueryTooShort indicates query text is too short.
	ErrQueryTooShort = errors.New("query too short")

	// ErrInvalidScope indicates an invalid search scope was provided.
	ErrInvalidScope = errors.New("invalid search scope")

	// ErrInvalidSortForScope indicates an invalid sort order for the given scope.
	ErrInvalidSortForScope = errors.New("sort order not supported for scope")

	// ErrInvalidFilter indicates an invalid filter value.
	ErrInvalidFilter = errors.New("invalid filter value")

	// ErrBackendUnavailable indicates search backend is temporarily unavailable (Phase 2).
	ErrBackendUnavailable = errors.New("search service temporarily unavailable")
)

// ─── SearchError ─────────────────────────────────────────────────────────────

// SearchError wraps a search-specific sentinel error with optional context. [12-search §16]
type SearchError struct {
	Err    error
	Field  string // Optional: which field caused the error
	Reason string // Optional: additional detail
}

func (e *SearchError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s: field=%s reason=%s", e.Err.Error(), e.Field, e.Reason)
	}
	return e.Err.Error()
}

func (e *SearchError) Unwrap() error { return e.Err }

// StatusCode maps SearchError to HTTP status code.
func (e *SearchError) StatusCode() int {
	switch {
	case errors.Is(e.Err, ErrQueryTooShort):
		return http.StatusUnprocessableEntity // 422
	case errors.Is(e.Err, ErrInvalidScope):
		return http.StatusBadRequest // 400
	case errors.Is(e.Err, ErrInvalidSortForScope):
		return http.StatusBadRequest // 400
	case errors.Is(e.Err, ErrInvalidFilter):
		return http.StatusBadRequest // 400
	case errors.Is(e.Err, ErrBackendUnavailable):
		return http.StatusServiceUnavailable // 503
	default:
		return http.StatusInternalServerError // 500
	}
}

// toAppError converts a SearchError to a shared.AppError for HTTP response. [CODING §2.2]
func (e *SearchError) toAppError() *shared.AppError {
	switch {
	case errors.Is(e.Err, ErrQueryTooShort):
		return &shared.AppError{Code: "query_too_short", Message: "Query too short", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrInvalidScope):
		return &shared.AppError{Code: "invalid_scope", Message: "Invalid search scope", StatusCode: http.StatusBadRequest}
	case errors.Is(e.Err, ErrInvalidSortForScope):
		return &shared.AppError{Code: "invalid_sort", Message: "Sort order not supported for this scope", StatusCode: http.StatusBadRequest}
	case errors.Is(e.Err, ErrInvalidFilter):
		return &shared.AppError{Code: "invalid_filter", Message: "Invalid filter value", StatusCode: http.StatusBadRequest}
	case errors.Is(e.Err, ErrBackendUnavailable):
		return &shared.AppError{Code: "service_unavailable", Message: "Search service temporarily unavailable", StatusCode: http.StatusServiceUnavailable}
	default:
		return shared.ErrInternal(e)
	}
}

