package iam

// Tests for IAM HTTP handler behaviour that can be exercised without a real
// database by injecting a pre-built AuthContext onto the Echo context.

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/labstack/echo/v4"
)

// Ensure mockIamService satisfies the IamService interface at compile time.
var _ IamService = (*mockIamService)(nil)

// TestCreateStudent_COPPA_NotConsented_Returns403 verifies that the
// createStudent handler immediately rejects requests when the family's COPPA
// consent status is not "consented" or "re_verified". [§15.4, §9.2]
//
// This check is inline in the handler and fires before the service is called,
// so the IamService is nil — any accidental call would panic and fail the test.
func TestCreateStudent_COPPA_NotConsented_Returns403(t *testing.T) {
	cases := []struct {
		name   string
		status string
	}{
		{"registered", "registered"},
		{"noticed", "noticed"},
		{"withdrawn", "withdrawn"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := echo.New()
			// nil service: the COPPA guard fires before the service is reached.
			h := NewHandler(nil, "")

			req := httptest.NewRequest(http.MethodPost, "/v1/families/students", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			shared.SetAuthContext(c, &shared.AuthContext{
				CoppaConsentStatus: tc.status,
			})

			err := h.createStudent(c)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			var appErr *shared.AppError
			if !errors.As(err, &appErr) {
				t.Fatalf("want *shared.AppError, got %T: %v", err, err)
			}
			if appErr.StatusCode != http.StatusForbidden {
				t.Errorf("want HTTP 403 Forbidden, got %d", appErr.StatusCode)
			}
			if appErr.Code != "coppa_consent_required" {
				t.Errorf("want code 'coppa_consent_required', got %q", appErr.Code)
			}
		})
	}
}

// TestCreateStudent_COPPA_Consented_NoConsentError verifies that for active
// consent statuses the COPPA guard does not return a coppa_consent_required
// error — the request proceeds past the gate (any subsequent error is
// unrelated to COPPA). [§9.2]
func TestCreateStudent_COPPA_Consented_NoConsentError(t *testing.T) {
	consentedStatuses := []string{"consented", "re_verified"}

	for _, status := range consentedStatuses {
		t.Run(status, func(t *testing.T) {
			e := echo.New()
			h := NewHandler(&mockIamService{}, "")

			req := httptest.NewRequest(http.MethodPost, "/v1/families/students", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			shared.SetAuthContext(c, &shared.AuthContext{
				CoppaConsentStatus: status,
			})

			err := h.createStudent(c)

			// Any error returned must NOT be a COPPA consent gate error.
			// (Validation or binding errors may occur — they are not COPPA errors.)
			if err != nil {
				var appErr *shared.AppError
				if errors.As(err, &appErr) && appErr.Code == "coppa_consent_required" {
					t.Errorf("status %q: COPPA gate should pass but returned coppa_consent_required", status)
				}
			}
		})
	}
}

// mockIamService is a minimal IamService stub for handler tests.
// Unimplemented methods panic to surface unexpected calls in tests.
type mockIamService struct{}

func (m *mockIamService) GetCurrentUser(_ context.Context, _ *shared.AuthContext) (*CurrentUserResponse, error) {
	panic("not implemented")
}
func (m *mockIamService) GetFamilyProfile(_ context.Context, _ *shared.FamilyScope) (*FamilyProfileResponse, error) {
	panic("not implemented")
}
func (m *mockIamService) ListStudents(_ context.Context, _ *shared.FamilyScope) ([]StudentResponse, error) {
	panic("not implemented")
}
func (m *mockIamService) GetConsentStatus(_ context.Context, _ *shared.FamilyScope) (*ConsentStatusResponse, error) {
	panic("not implemented")
}
func (m *mockIamService) HandlePostRegistration(_ context.Context, _ KratosWebhookPayload) error {
	panic("not implemented")
}
func (m *mockIamService) HandlePostLogin(_ context.Context, _ KratosWebhookPayload) error {
	panic("not implemented")
}
func (m *mockIamService) UpdateFamilyProfile(_ context.Context, _ *shared.FamilyScope, _ UpdateFamilyCommand) (*FamilyProfileResponse, error) {
	panic("not implemented")
}
func (m *mockIamService) CreateStudent(_ context.Context, _ *shared.FamilyScope, _ CreateStudentCommand) (*StudentResponse, error) {
	return &StudentResponse{}, nil
}
func (m *mockIamService) UpdateStudent(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ UpdateStudentCommand) (*StudentResponse, error) {
	panic("not implemented")
}
func (m *mockIamService) DeleteStudent(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) error {
	panic("not implemented")
}
func (m *mockIamService) SubmitCoppaConsent(_ context.Context, _ *shared.FamilyScope, _ *shared.AuthContext, _ CoppaConsentCommand) (*ConsentStatusResponse, error) {
	panic("not implemented")
}
