package safety

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

func TestSafetyError_toAppError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{"report_not_found", ErrReportNotFound, http.StatusNotFound},
		{"flag_not_found", ErrFlagNotFound, http.StatusNotFound},
		{"action_not_found", ErrActionNotFound, http.StatusNotFound},
		{"appeal_not_found", ErrAppealNotFound, http.StatusNotFound},
		{"duplicate_report", ErrDuplicateReport, http.StatusConflict},
		{"appeal_already_exists", ErrAppealAlreadyExists, http.StatusConflict},
		{"csam_ban_not_appealable", ErrCsamBanNotAppealable, http.StatusUnprocessableEntity},
		{"same_admin_appeal", ErrSameAdminAppeal, http.StatusUnprocessableEntity},
		{"invalid_report_transition", ErrInvalidReportTransition, http.StatusUnprocessableEntity},
		{"flag_already_reviewed", ErrFlagAlreadyReviewed, http.StatusUnprocessableEntity},
		{"account_suspended", ErrAccountSuspended, http.StatusForbidden},
		{"account_banned", ErrAccountBanned, http.StatusForbidden},
		{"invalid_action_type", ErrInvalidActionType, http.StatusUnprocessableEntity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			se := &SafetyError{Err: tt.err}
			appErr := se.toAppError()
			if appErr.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", appErr.StatusCode, tt.wantStatus)
			}
		})
	}
}

func TestSafetyError_unknown_maps_to_500(t *testing.T) {
	se := &SafetyError{Err: fmt.Errorf("unexpected internal error")}
	appErr := se.toAppError()
	if appErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusInternalServerError)
	}
}

func TestMapSafetyError_safety_error(t *testing.T) {
	err := &SafetyError{Err: ErrReportNotFound}
	appErr := mapSafetyError(err)
	if appErr.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusNotFound)
	}
}

func TestMapSafetyError_app_error(t *testing.T) {
	err := shared.ErrForbidden()
	appErr := mapSafetyError(err)
	if appErr.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusForbidden)
	}
}

func TestMapSafetyError_unknown(t *testing.T) {
	err := errors.New("generic error")
	appErr := mapSafetyError(err)
	if appErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", appErr.StatusCode, http.StatusInternalServerError)
	}
}
