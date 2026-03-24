package media

import (
	"net/http"
	"testing"
)

func TestMediaError_toAppError(t *testing.T) {
	tests := []struct {
		name           string
		sentinel       error
		wantCode       string
		wantMessage    string
		wantStatusCode int
	}{
		{
			name:           "UploadNotFound maps to 404",
			sentinel:       ErrUploadNotFound,
			wantCode:       "upload_not_found",
			wantMessage:    "Upload not found",
			wantStatusCode: http.StatusNotFound,
		},
		{
			name:           "InvalidFileType maps to 422",
			sentinel:       ErrInvalidFileType,
			wantCode:       "invalid_file_type",
			wantMessage:    "Invalid file type for this context",
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:           "FileTooLarge maps to 422",
			sentinel:       ErrFileTooLarge,
			wantCode:       "file_too_large",
			wantMessage:    "File exceeds maximum size for this context",
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:           "UploadNotPending maps to 409",
			sentinel:       ErrUploadNotPending,
			wantCode:       "upload_not_pending",
			wantMessage:    "Upload is not in pending status",
			wantStatusCode: http.StatusConflict,
		},
		{
			name:           "UploadExpired maps to 410",
			sentinel:       ErrUploadExpired,
			wantCode:       "upload_expired",
			wantMessage:    "Upload presigned URL has expired",
			wantStatusCode: http.StatusGone,
		},
		{
			name:           "ObjectNotInStorage maps to 502",
			sentinel:       ErrObjectNotInStorage,
			wantCode:       "object_not_in_storage",
			wantMessage:    "Object not found in storage",
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "StorageOperation maps to 502",
			sentinel:       ErrStorageOperation,
			wantCode:       "storage_error",
			wantMessage:    "Storage operation failed",
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "MagicByteMismatch maps to 422",
			sentinel:       ErrMagicByteMismatch,
			wantCode:       "magic_byte_mismatch",
			wantMessage:    "File content does not match declared type",
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:           "CSAMDetected maps to 403",
			sentinel:       ErrCSAMDetected,
			wantCode:       "content_violation",
			wantMessage:    "Content policy violation",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "ModerationViolation maps to 403",
			sentinel:       ErrModerationViolation,
			wantCode:       "moderation_violation",
			wantMessage:    "Content moderation violation",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "ProcessingFailed maps to 500",
			sentinel:       ErrProcessingFailed,
			wantCode:       "processing_failed",
			wantMessage:    "Upload processing failed",
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name:           "NotOwner maps to 403",
			sentinel:       ErrNotOwner,
			wantCode:       "not_owner",
			wantMessage:    "Not the owner of this upload",
			wantStatusCode: http.StatusForbidden,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mediaErr := &MediaError{Err: tc.sentinel}
			appErr := mediaErr.toAppError()

			if appErr.Code != tc.wantCode {
				t.Errorf("Code = %q, want %q", appErr.Code, tc.wantCode)
			}
			if appErr.Message != tc.wantMessage {
				t.Errorf("Message = %q, want %q", appErr.Message, tc.wantMessage)
			}
			if appErr.StatusCode != tc.wantStatusCode {
				t.Errorf("StatusCode = %d, want %d", appErr.StatusCode, tc.wantStatusCode)
			}
		})
	}
}

func TestMediaError_toAppError_unknown_falls_to_internal(t *testing.T) {
	unknownErr := &MediaError{Err: errForTest}
	appErr := unknownErr.toAppError()

	if appErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("StatusCode = %d, want %d", appErr.StatusCode, http.StatusInternalServerError)
	}
	if appErr.Code != "internal_error" {
		t.Errorf("Code = %q, want %q", appErr.Code, "internal_error")
	}
}

func TestMediaError_Error(t *testing.T) {
	mediaErr := &MediaError{Err: ErrUploadNotFound}
	if mediaErr.Error() != "upload not found" {
		t.Errorf("Error() = %q, want %q", mediaErr.Error(), "upload not found")
	}
}

func TestMediaError_Unwrap(t *testing.T) {
	mediaErr := &MediaError{Err: ErrUploadNotFound}
	if mediaErr.Unwrap() != ErrUploadNotFound {
		t.Errorf("Unwrap() did not return the sentinel error")
	}
}

// errForTest is a sentinel used by the unknown-error test case.
var errForTest = &testError{msg: "test error"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }
