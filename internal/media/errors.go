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
	ErrUploadNotFound      = errors.New("upload not found")
	ErrInvalidFileType     = errors.New("invalid file type for this context")
	ErrFileTooLarge        = errors.New("file exceeds maximum size for this context")
	ErrUploadNotPending    = errors.New("upload is not in pending status")
	ErrUploadExpired       = errors.New("upload presigned URL has expired")
	ErrObjectNotInStorage  = errors.New("object not found in storage")
	ErrStorageOperation    = errors.New("storage operation failed")
	ErrMagicByteMismatch   = errors.New("file content does not match declared type")
	ErrCSAMDetected        = errors.New("CSAM content detected")
	ErrModerationViolation = errors.New("content moderation violation")
	ErrProcessingFailed    = errors.New("upload processing failed")
	ErrNotOwner            = errors.New("not the owner of this upload")
)

// toAppError maps a MediaError sentinel to a *shared.AppError with the correct
// HTTP status code. Called by mapMediaError in handler.go. [09-media §15.1]
func (e *MediaError) toAppError() *shared.AppError {
	switch {
	case errors.Is(e.Err, ErrUploadNotFound):
		return &shared.AppError{Code: "upload_not_found", Message: "Upload not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrInvalidFileType):
		return &shared.AppError{Code: "invalid_file_type", Message: "Invalid file type for this context", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrFileTooLarge):
		return &shared.AppError{Code: "file_too_large", Message: "File exceeds maximum size for this context", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrUploadNotPending):
		return &shared.AppError{Code: "upload_not_pending", Message: "Upload is not in pending status", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrUploadExpired):
		return &shared.AppError{Code: "upload_expired", Message: "Upload presigned URL has expired", StatusCode: http.StatusGone}
	case errors.Is(e.Err, ErrObjectNotInStorage):
		return &shared.AppError{Code: "object_not_in_storage", Message: "Object not found in storage", StatusCode: http.StatusBadGateway}
	case errors.Is(e.Err, ErrStorageOperation):
		return &shared.AppError{Code: "storage_error", Message: "Storage operation failed", StatusCode: http.StatusBadGateway}
	case errors.Is(e.Err, ErrMagicByteMismatch):
		return &shared.AppError{Code: "magic_byte_mismatch", Message: "File content does not match declared type", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrCSAMDetected):
		return &shared.AppError{Code: "content_violation", Message: "Content policy violation", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, ErrModerationViolation):
		return &shared.AppError{Code: "moderation_violation", Message: "Content moderation violation", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, ErrProcessingFailed):
		return &shared.AppError{Code: "processing_failed", Message: "Upload processing failed", StatusCode: http.StatusInternalServerError}
	case errors.Is(e.Err, ErrNotOwner):
		return &shared.AppError{Code: "not_owner", Message: "Not the owner of this upload", StatusCode: http.StatusForbidden}
	default:
		return shared.ErrInternal(e)
	}
}
