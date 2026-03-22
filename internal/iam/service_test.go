package iam

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// captureHandler is a test DomainEventHandler that stores the last received event.
type captureHandler struct {
	capture *shared.DomainEvent
}

func (h *captureHandler) Handle(_ context.Context, event shared.DomainEvent) error {
	*h.capture = event
	return nil
}

// ─── COPPA State Machine Tests ────────────────────────────────────────────────

func TestResolveConsentTransition(t *testing.T) {
	tests := []struct {
		name       string
		current    CoppaConsentStatus
		cmd        CoppaConsentCommand
		wantStatus CoppaConsentStatus
		wantError  bool
	}{
		{
			name:    "registered + notice only → noticed",
			current: CoppaConsentRegistered,
			cmd:     CoppaConsentCommand{Method: "m", VerificationToken: "", CoppaNoticeAcknowledged: true},
			wantStatus: CoppaConsentNoticed,
		},
		{
			name:    "registered + full consent → consented",
			current: CoppaConsentRegistered,
			cmd:     CoppaConsentCommand{Method: "m", VerificationToken: "tok", CoppaNoticeAcknowledged: true},
			wantStatus: CoppaConsentConsented,
		},
		{
			name:    "noticed + full consent → consented",
			current: CoppaConsentNoticed,
			cmd:     CoppaConsentCommand{Method: "m", VerificationToken: "tok", CoppaNoticeAcknowledged: true},
			wantStatus: CoppaConsentConsented,
		},
		{
			name:    "consented + re-verify → re_verified",
			current: CoppaConsentConsented,
			cmd:     CoppaConsentCommand{Method: "m", VerificationToken: "tok", CoppaNoticeAcknowledged: true},
			wantStatus: CoppaConsentReVerified,
		},
		{
			name:    "re_verified + re-verify → re_verified",
			current: CoppaConsentReVerified,
			cmd:     CoppaConsentCommand{Method: "m", VerificationToken: "tok", CoppaNoticeAcknowledged: true},
			wantStatus: CoppaConsentReVerified,
		},
		// Invalid transitions
		{
			name:      "no acknowledgement → error",
			current:   CoppaConsentRegistered,
			cmd:       CoppaConsentCommand{Method: "m", VerificationToken: "tok", CoppaNoticeAcknowledged: false},
			wantError: true,
		},
		{
			name:      "withdrawn → consented is invalid",
			current:   CoppaConsentWithdrawn,
			cmd:       CoppaConsentCommand{Method: "m", VerificationToken: "tok", CoppaNoticeAcknowledged: true},
			wantError: true,
		},
		{
			name:      "noticed + no token is invalid",
			current:   CoppaConsentNoticed,
			cmd:       CoppaConsentCommand{Method: "m", VerificationToken: "", CoppaNoticeAcknowledged: true},
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _, err := resolveConsentTransition(tc.current, tc.cmd)
			if tc.wantError {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var consentErr *InvalidConsentTransitionError
				if !errors.As(err, &consentErr) {
					t.Fatalf("expected InvalidConsentTransitionError, got %T: %v", err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantStatus {
				t.Errorf("got status %q, want %q", got, tc.wantStatus)
			}
		})
	}
}

// ─── Event Publishing Tests ───────────────────────────────────────────────────

// TestFamilyCreatedEvent verifies the FamilyCreated event is correctly published
// and received by subscribers.
func TestFamilyCreatedEvent(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.New()
	parentID := uuid.New()

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeOf(FamilyCreated{}), &captureHandler{capture: &got})

	if err := bus.Publish(ctx, FamilyCreated{FamilyID: familyID, ParentID: parentID}); err != nil {
		t.Fatal(err)
	}

	if got == nil {
		t.Fatal("no event received")
	}
	evt, ok := got.(FamilyCreated)
	if !ok {
		t.Fatalf("got %T, want FamilyCreated", got)
	}
	if evt.FamilyID != familyID || evt.ParentID != parentID {
		t.Errorf("unexpected event fields: %+v", evt)
	}
}

func TestStudentCreatedEvent(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.New()
	studentID := uuid.New()

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeOf(StudentCreated{}), &captureHandler{capture: &got})

	if err := bus.Publish(ctx, StudentCreated{FamilyID: familyID, StudentID: studentID}); err != nil {
		t.Fatal(err)
	}

	evt, ok := got.(StudentCreated)
	if !ok {
		t.Fatalf("got %T, want StudentCreated", got)
	}
	if evt.StudentID != studentID || evt.FamilyID != familyID {
		t.Errorf("unexpected event: %+v", evt)
	}
}

func TestStudentDeletedEvent(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.New()
	studentID := uuid.New()

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeOf(StudentDeleted{}), &captureHandler{capture: &got})

	if err := bus.Publish(ctx, StudentDeleted{FamilyID: familyID, StudentID: studentID}); err != nil {
		t.Fatal(err)
	}

	evt, ok := got.(StudentDeleted)
	if !ok {
		t.Fatalf("got %T, want StudentDeleted", got)
	}
	if evt.StudentID != studentID {
		t.Errorf("got student_id %v, want %v", evt.StudentID, studentID)
	}
}

func TestCoppaConsentGrantedEvent(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.New()

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeOf(CoppaConsentGranted{}), &captureHandler{capture: &got})

	if err := bus.Publish(ctx, CoppaConsentGranted{FamilyID: familyID}); err != nil {
		t.Fatal(err)
	}

	evt, ok := got.(CoppaConsentGranted)
	if !ok {
		t.Fatalf("got %T, want CoppaConsentGranted", got)
	}
	if evt.FamilyID != familyID {
		t.Errorf("got family_id %v, want %v", evt.FamilyID, familyID)
	}
}

// ─── UUIDArray Tests ──────────────────────────────────────────────────────────

func TestUUIDArray(t *testing.T) {
	id1 := uuid.MustParse("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11")
	id2 := uuid.MustParse("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12")

	t.Run("scan two UUIDs from string", func(t *testing.T) {
		var arr UUIDArray
		err := arr.Scan("{a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11,b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12}")
		if err != nil {
			t.Fatal(err)
		}
		if len(arr) != 2 {
			t.Fatalf("want 2 elements, got %d", len(arr))
		}
		if arr[0] != id1 || arr[1] != id2 {
			t.Errorf("unexpected values: %v", arr)
		}
	})

	t.Run("scan empty array", func(t *testing.T) {
		var arr UUIDArray
		if err := arr.Scan("{}"); err != nil {
			t.Fatal(err)
		}
		if len(arr) != 0 {
			t.Fatalf("want 0 elements, got %d", len(arr))
		}
	})

	t.Run("scan nil", func(t *testing.T) {
		var arr UUIDArray
		if err := arr.Scan(nil); err != nil {
			t.Fatal(err)
		}
		if arr != nil {
			t.Error("expected nil UUIDArray after scanning nil")
		}
	})

	t.Run("value for two UUIDs", func(t *testing.T) {
		arr := UUIDArray{id1, id2}
		val, err := arr.Value()
		if err != nil {
			t.Fatal(err)
		}
		want := "{a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11,b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12}"
		if val != want {
			t.Errorf("got %v, want %v", val, want)
		}
	})

	t.Run("value for empty array", func(t *testing.T) {
		arr := UUIDArray{}
		val, err := arr.Value()
		if err != nil {
			t.Fatal(err)
		}
		if val != "{}" {
			t.Errorf("got %v, want {}", val)
		}
	})
}

// ─── CoppaConsentStatus Helper Tests ─────────────────────────────────────────

func TestCoppaConsentStatus_CanCreateStudents(t *testing.T) {
	tests := []struct {
		status CoppaConsentStatus
		can    bool
	}{
		{CoppaConsentRegistered, false},
		{CoppaConsentNoticed, false},
		{CoppaConsentConsented, true},
		{CoppaConsentReVerified, true},
		{CoppaConsentWithdrawn, false},
	}
	for _, tc := range tests {
		if got := tc.status.CanCreateStudents(); got != tc.can {
			t.Errorf("status %q: CanCreateStudents() = %v, want %v", tc.status, got, tc.can)
		}
	}
}
