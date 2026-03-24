package media

import (
	"errors"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// MediaError wraps a sentinel error with optional context. [CODING §2.2]
type MediaError struct {
	Err error
}

func (e *MediaError) Error() string { return e.Err.Error() }
func (e *MediaError) Unwrap() error { return e.Err }

// ─── Sentinel Errors ──────────────────────────────────────────────────────────
// All media domain sentinel errors per 09-media §15.

var (
	// ─── Upload lifecycle ───────────────────────────────────────────────
	ErrUploadNotFound     = errors.New("upload not found")
	ErrInvalidFileType    = errors.New("invalid file type for this context")
	ErrFileTooLarge       = errors.New("file exceeds maximum size for this context")
	ErrUploadNotConfirmed = errors.New("upload has not been confirmed")
	ErrUploadQuarantined  = errors.New("upload is quarantined")
	ErrUploadRejected     = errors.New("upload was rejected by content policy")
	ErrUploadFlagged      = errors.New("upload is flagged for review")
	ErrUploadExpired      = errors.New("upload has expired")
	ErrNotOwner           = errors.New("not the upload owner")

	// ─── External service errors ────────────────────────────────────────
	ErrObjectStorageError     = errors.New("object storage operation failed")
	ErrScanServiceUnavailable = errors.New("safety scan service unavailable")
	ErrScanServiceFailed      = errors.New("safety scan failed")

	// ─── Pipeline errors (internal only — not mapped to HTTP responses) ─
	ErrMagicByteMismatch = errors.New("file content does not match declared type")
	ErrProcessingFailed  = errors.New("upload processing failed")
)

// toAppError maps a MediaError sentinel to a *shared.AppError with the correct
// HTTP status code. Called by mapMediaError in handler.go. [09-media §15.1]
func (e *MediaError) toAppError() *shared.AppError {
	switch {
	case errors.Is(e.Err, ErrUploadNotFound):
		return &shared.AppError{Code: "upload_not_found", Message: "Upload not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrInvalidFileType):
		return &shared.AppError{Code: "invalid_file_type", Message: "File type is not allowed for this upload context", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrFileTooLarge):
		return &shared.AppError{Code: "file_too_large", Message: "File exceeds the maximum allowed size", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrUploadNotConfirmed):
		return &shared.AppError{Code: "upload_not_confirmed", Message: "Upload must be confirmed before this operation", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrUploadQuarantined):
		return &shared.AppError{Code: "upload_quarantined", Message: "This upload has been restricted", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, ErrUploadRejected):
		return &shared.AppError{Code: "upload_rejected", Message: "This upload was not published because it violates our content guidelines", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, ErrUploadFlagged):
		return &shared.AppError{Code: "upload_flagged", Message: "This upload is under review", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, ErrUploadExpired):
		return &shared.AppError{Code: "upload_expired", Message: "Upload link has expired — please request a new one", StatusCode: http.StatusGone}
	case errors.Is(e.Err, ErrNotOwner):
		return &shared.AppError{Code: "not_owner", Message: "You do not have permission to access this upload", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, ErrObjectStorageError):
		return &shared.AppError{Code: "storage_error", Message: "File storage is temporarily unavailable", StatusCode: http.StatusBadGateway}
	case errors.Is(e.Err, ErrScanServiceUnavailable):
		return &shared.AppError{Code: "scan_unavailable", Message: "Content scanning is temporarily unavailable", StatusCode: http.StatusServiceUnavailable}
	case errors.Is(e.Err, ErrScanServiceFailed):
		return &shared.AppError{Code: "scan_failed", Message: "Content scanning encountered an error", StatusCode: http.StatusBadGateway}
	default:
		return shared.ErrInternal(e)
	}
}
