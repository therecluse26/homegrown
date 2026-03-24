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
			wantMessage:    "File type is not allowed for this upload context",
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:           "FileTooLarge maps to 422",
			sentinel:       ErrFileTooLarge,
			wantCode:       "file_too_large",
			wantMessage:    "File exceeds the maximum allowed size",
			wantStatusCode: http.StatusUnprocessableEntity,
		},
		{
			name:           "UploadNotConfirmed maps to 409",
			sentinel:       ErrUploadNotConfirmed,
			wantCode:       "upload_not_confirmed",
			wantMessage:    "Upload must be confirmed before this operation",
			wantStatusCode: http.StatusConflict,
		},
		{
			name:           "UploadQuarantined maps to 403",
			sentinel:       ErrUploadQuarantined,
			wantCode:       "upload_quarantined",
			wantMessage:    "This upload has been restricted",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "UploadRejected maps to 403",
			sentinel:       ErrUploadRejected,
			wantCode:       "upload_rejected",
			wantMessage:    "This upload was not published because it violates our content guidelines",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "UploadFlagged maps to 403",
			sentinel:       ErrUploadFlagged,
			wantCode:       "upload_flagged",
			wantMessage:    "This upload is under review",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "UploadExpired maps to 410",
			sentinel:       ErrUploadExpired,
			wantCode:       "upload_expired",
			wantMessage:    "Upload link has expired — please request a new one",
			wantStatusCode: http.StatusGone,
		},
		{
			name:           "NotOwner maps to 403",
			sentinel:       ErrNotOwner,
			wantCode:       "not_owner",
			wantMessage:    "You do not have permission to access this upload",
			wantStatusCode: http.StatusForbidden,
		},
		{
			name:           "ObjectStorageError maps to 502",
			sentinel:       ErrObjectStorageError,
			wantCode:       "storage_error",
			wantMessage:    "File storage is temporarily unavailable",
			wantStatusCode: http.StatusBadGateway,
		},
		{
			name:           "ScanServiceUnavailable maps to 503",
			sentinel:       ErrScanServiceUnavailable,
			wantCode:       "scan_unavailable",
			wantMessage:    "Content scanning is temporarily unavailable",
			wantStatusCode: http.StatusServiceUnavailable,
		},
		{
			name:           "ScanServiceFailed maps to 502",
			sentinel:       ErrScanServiceFailed,
			wantCode:       "scan_failed",
			wantMessage:    "Content scanning encountered an error",
			wantStatusCode: http.StatusBadGateway,
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
