package iam

import (
	"bytes"
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
	familyID := uuid.Must(uuid.NewV7())
	parentID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeFor[FamilyCreated](), &captureHandler{capture: &got})

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
	familyID := uuid.Must(uuid.NewV7())
	studentID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeFor[StudentCreated](), &captureHandler{capture: &got})

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
	familyID := uuid.Must(uuid.NewV7())
	studentID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeFor[StudentDeleted](), &captureHandler{capture: &got})

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
	familyID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeFor[CoppaConsentGranted](), &captureHandler{capture: &got})

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

// ─── SlugArray Tests ──────────────────────────────────────────────────────────

func TestSlugArray(t *testing.T) {
	t.Run("scan two slugs from string", func(t *testing.T) {
		var arr SlugArray
		if err := arr.Scan("{charlotte-mason,classical}"); err != nil {
			t.Fatal(err)
		}
		if len(arr) != 2 {
			t.Fatalf("want 2 elements, got %d", len(arr))
		}
		if arr[0] != "charlotte-mason" || arr[1] != "classical" {
			t.Errorf("unexpected values: %v", arr)
		}
	})

	t.Run("scan empty array", func(t *testing.T) {
		var arr SlugArray
		if err := arr.Scan("{}"); err != nil {
			t.Fatal(err)
		}
		if len(arr) != 0 {
			t.Fatalf("want 0 elements, got %d", len(arr))
		}
	})

	t.Run("scan nil", func(t *testing.T) {
		var arr SlugArray
		if err := arr.Scan(nil); err != nil {
			t.Fatal(err)
		}
		if arr != nil {
			t.Error("expected nil SlugArray after scanning nil")
		}
	})

	t.Run("value for two slugs", func(t *testing.T) {
		arr := SlugArray{"charlotte-mason", "classical"}
		val, err := arr.Value()
		if err != nil {
			t.Fatal(err)
		}
		if val != "{charlotte-mason,classical}" {
			t.Errorf("got %v, want {charlotte-mason,classical}", val)
		}
	})

	t.Run("value for empty array", func(t *testing.T) {
		arr := SlugArray{}
		val, err := arr.Value()
		if err != nil {
			t.Fatal(err)
		}
		if val != "{}" {
			t.Errorf("got %v, want {}", val)
		}
	})
}

// ─── Phase 2 Guard Tests ──────────────────────────────────────────────────────
// These test early-exit guards that fire before any DB access, allowing unit
// testing without a database. Methods return immediately on guard failure.

// TestInviteCoParent_RequiresPrimary verifies that InviteCoParent rejects non-primary parents.
func TestInviteCoParent_RequiresPrimary(t *testing.T) {
	svc := &IamServiceImpl{eventBus: shared.NewEventBus()}
	auth := &shared.AuthContext{IsPrimaryParent: false}
	scope := shared.NewFamilyScopeFromID(uuid.Must(uuid.NewV7()))
	_, err := svc.InviteCoParent(context.Background(), &scope, auth, InviteCoParentCommand{Email: "x@example.com"})
	if !errors.Is(err, ErrNotPrimaryParent) {
		t.Errorf("want ErrNotPrimaryParent, got %v", err)
	}
}

// TestCancelInvite_WrongStatus verifies that CancelInvite with a nil DB returns early guard error
// only for the "not primary" guard path (DB access happens after the invite fetch).
// Full accept/cancel logic is tested in integration tests.

// TestTransferPrimaryParent_SelfTransfer verifies ErrCannotTransferToSelf is returned.
func TestTransferPrimaryParent_SelfTransfer(t *testing.T) {
	parentID := uuid.Must(uuid.NewV7())
	svc := &IamServiceImpl{eventBus: shared.NewEventBus()}
	auth := &shared.AuthContext{IsPrimaryParent: true, ParentID: parentID}
	scope := shared.NewFamilyScopeFromID(uuid.Must(uuid.NewV7()))
	err := svc.TransferPrimaryParent(context.Background(), &scope, auth, TransferPrimaryCommand{
		NewPrimaryParentID: parentID, // same as requester
	})
	if !errors.Is(err, ErrCannotTransferToSelf) {
		t.Errorf("want ErrCannotTransferToSelf, got %v", err)
	}
}

// TestWithdrawCoppaConsent_InvalidState verifies state machine guard.
func TestWithdrawCoppaConsent_InvalidState(t *testing.T) {
	svc := &IamServiceImpl{eventBus: shared.NewEventBus()}
	auth := &shared.AuthContext{IsPrimaryParent: true, CoppaConsentStatus: string(CoppaConsentNoticed)}
	scope := shared.NewFamilyScopeFromID(uuid.Must(uuid.NewV7()))
	err := svc.WithdrawCoppaConsent(context.Background(), &scope, auth)
	var consentErr *InvalidConsentTransitionError
	if !errors.As(err, &consentErr) {
		t.Errorf("want InvalidConsentTransitionError, got %v", err)
	}
}

// TestRemoveCoParent_RequiresPrimary verifies guard before DB access.
func TestRemoveCoParent_RequiresPrimary(t *testing.T) {
	svc := &IamServiceImpl{eventBus: shared.NewEventBus()}
	auth := &shared.AuthContext{IsPrimaryParent: false}
	scope := shared.NewFamilyScopeFromID(uuid.Must(uuid.NewV7()))
	err := svc.RemoveCoParent(context.Background(), &scope, auth, uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrNotPrimaryParent) {
		t.Errorf("want ErrNotPrimaryParent, got %v", err)
	}
}

// TestRequestFamilyDeletion_RequiresPrimary verifies guard.
func TestRequestFamilyDeletion_RequiresPrimary(t *testing.T) {
	svc := &IamServiceImpl{eventBus: shared.NewEventBus()}
	auth := &shared.AuthContext{IsPrimaryParent: false}
	scope := shared.NewFamilyScopeFromID(uuid.Must(uuid.NewV7()))
	err := svc.RequestFamilyDeletion(context.Background(), &scope, auth)
	if !errors.Is(err, ErrNotPrimaryParent) {
		t.Errorf("want ErrNotPrimaryParent, got %v", err)
	}
}

// ─── generateToken Tests ──────────────────────────────────────────────────────

// TestGenerateToken verifies that generateToken produces a 64-char hex plaintext
// and a valid bcrypt hash.
func TestGenerateToken(t *testing.T) {
	pt, hash, err := generateToken(bytes.NewReader(make([]byte, 32)).Read)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	// 32 random bytes → 64 hex chars.
	if len(pt) != 64 {
		t.Errorf("plaintext length = %d, want 64", len(pt))
	}
	// bcrypt hash starts with "$2a$" or "$2b$".
	if len(hash) < 4 || (hash[:3] != "$2a" && hash[:3] != "$2b") {
		t.Errorf("hash does not look like a bcrypt hash: %s", hash[:min(len(hash), 10)])
	}
}

// TestGenerateToken_ReaderError verifies that generateToken propagates read errors.
func TestGenerateToken_ReaderError(t *testing.T) {
	errReader := func(_ []byte) (int, error) { return 0, errors.New("entropy failed") }
	_, _, err := generateToken(errReader)
	if err == nil {
		t.Fatal("expected error from broken reader, got nil")
	}
}

// ─── Phase 2 Event Tests ──────────────────────────────────────────────────────

func TestCoParentAddedEvent(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.Must(uuid.NewV7())
	coParentID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeFor[CoParentAdded](), &captureHandler{capture: &got})

	if err := bus.Publish(ctx, CoParentAdded{
		FamilyID:      familyID,
		CoParentID:    coParentID,
		CoParentEmail: "test@example.com",
		CoParentName:  "Test User",
	}); err != nil {
		t.Fatal(err)
	}

	evt, ok := got.(CoParentAdded)
	if !ok {
		t.Fatalf("got %T, want CoParentAdded", got)
	}
	if evt.FamilyID != familyID || evt.CoParentID != coParentID {
		t.Errorf("unexpected event fields: %+v", evt)
	}
	if evt.CoParentName != "Test User" {
		t.Errorf("CoParentName = %q, want %q", evt.CoParentName, "Test User")
	}
}

func TestFamilyDeletionScheduledEvent(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeFor[FamilyDeletionScheduled](), &captureHandler{capture: &got})

	if err := bus.Publish(ctx, FamilyDeletionScheduled{FamilyID: familyID}); err != nil {
		t.Fatal(err)
	}

	evt, ok := got.(FamilyDeletionScheduled)
	if !ok {
		t.Fatalf("got %T, want FamilyDeletionScheduled", got)
	}
	if evt.FamilyID != familyID {
		t.Errorf("got family_id %v, want %v", evt.FamilyID, familyID)
	}
}

func TestPrimaryParentTransferredEvent(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.Must(uuid.NewV7())
	newPrimary := uuid.Must(uuid.NewV7())
	prevPrimary := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeFor[PrimaryParentTransferred](), &captureHandler{capture: &got})

	if err := bus.Publish(ctx, PrimaryParentTransferred{
		FamilyID:      familyID,
		NewPrimaryID:  newPrimary,
		PrevPrimaryID: prevPrimary,
	}); err != nil {
		t.Fatal(err)
	}

	evt, ok := got.(PrimaryParentTransferred)
	if !ok {
		t.Fatalf("got %T, want PrimaryParentTransferred", got)
	}
	if evt.NewPrimaryID != newPrimary || evt.PrevPrimaryID != prevPrimary {
		t.Errorf("unexpected event fields: %+v", evt)
	}
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

// ─── TransferPrimaryParent Guard Tests ───────────────────────────────────────

func TestTransferPrimaryParent_RequiresPrimary(t *testing.T) {
	svc := &IamServiceImpl{eventBus: shared.NewEventBus()}
	auth := &shared.AuthContext{IsPrimaryParent: false}
	scope := shared.NewFamilyScopeFromID(uuid.Must(uuid.NewV7()))
	err := svc.TransferPrimaryParent(context.Background(), &scope, auth, TransferPrimaryCommand{
		NewPrimaryParentID: uuid.Must(uuid.NewV7()),
	})
	if !errors.Is(err, ErrNotPrimaryParent) {
		t.Errorf("want ErrNotPrimaryParent, got %v", err)
	}
}

// ─── WithdrawCoppaConsent Guard Tests ────────────────────────────────────────

func TestWithdrawCoppaConsent_RequiresPrimary(t *testing.T) {
	svc := &IamServiceImpl{eventBus: shared.NewEventBus()}
	auth := &shared.AuthContext{IsPrimaryParent: false, CoppaConsentStatus: string(CoppaConsentConsented)}
	scope := shared.NewFamilyScopeFromID(uuid.Must(uuid.NewV7()))
	err := svc.WithdrawCoppaConsent(context.Background(), &scope, auth)
	if !errors.Is(err, ErrNotPrimaryParent) {
		t.Errorf("want ErrNotPrimaryParent, got %v", err)
	}
}

func TestWithdrawCoppaConsent_RegisteredState_InvalidTransition(t *testing.T) {
	svc := &IamServiceImpl{eventBus: shared.NewEventBus()}
	auth := &shared.AuthContext{IsPrimaryParent: true, CoppaConsentStatus: string(CoppaConsentRegistered)}
	scope := shared.NewFamilyScopeFromID(uuid.Must(uuid.NewV7()))
	err := svc.WithdrawCoppaConsent(context.Background(), &scope, auth)
	var consentErr *InvalidConsentTransitionError
	if !errors.As(err, &consentErr) {
		t.Fatalf("want InvalidConsentTransitionError, got %T: %v", err, err)
	}
	if consentErr.From != string(CoppaConsentRegistered) {
		t.Errorf("From = %q, want %q", consentErr.From, CoppaConsentRegistered)
	}
	if consentErr.To != string(CoppaConsentWithdrawn) {
		t.Errorf("To = %q, want %q", consentErr.To, CoppaConsentWithdrawn)
	}
}

// ─── InvalidConsentTransitionError Tests ─────────────────────────────────────

func TestInvalidConsentTransitionError_Message(t *testing.T) {
	err := &InvalidConsentTransitionError{From: "registered", To: "consented"}
	want := "invalid COPPA consent transition from registered to consented"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

// ─── SlugArray Edge Cases ────────────────────────────────────────────────────

func TestSlugArray_ScanBytes(t *testing.T) {
	var arr SlugArray
	if err := arr.Scan([]byte("{unschooling,montessori}")); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 2 {
		t.Fatalf("want 2 elements, got %d", len(arr))
	}
	if arr[0] != "unschooling" || arr[1] != "montessori" {
		t.Errorf("unexpected values: %v", arr)
	}
}

func TestSlugArray_ScanUnsupportedType(t *testing.T) {
	var arr SlugArray
	err := arr.Scan(42)
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
}

func TestSlugArray_SingleElement(t *testing.T) {
	var arr SlugArray
	if err := arr.Scan("{charlotte-mason}"); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 1 || arr[0] != "charlotte-mason" {
		t.Errorf("unexpected: %v", arr)
	}
	val, err := arr.Value()
	if err != nil {
		t.Fatal(err)
	}
	if val != "{charlotte-mason}" {
		t.Errorf("got %v, want {charlotte-mason}", val)
	}
}

// ─── Event EventName Tests ───────────────────────────────────────────────────

func TestEventNames(t *testing.T) {
	tests := []struct {
		event shared.DomainEvent
		name  string
	}{
		{FamilyCreated{}, "iam.FamilyCreated"},
		{StudentCreated{}, "iam.StudentCreated"},
		{StudentDeleted{}, "iam.StudentDeleted"},
		{CoppaConsentGranted{}, "iam.CoppaConsentGranted"},
		{FamilyDeletionScheduled{}, "iam.FamilyDeletionScheduled"},
		{InviteCreated{}, "iam.InviteCreated"},
		{CoParentAdded{}, "iam.CoParentAdded"},
		{CoParentRemoved{}, "iam.CoParentRemoved"},
		{PrimaryParentTransferred{}, "iam.PrimaryParentTransferred"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.event.EventName(); got != tc.name {
				t.Errorf("EventName() = %q, want %q", got, tc.name)
			}
		})
	}
}

// ─── Phase 2 Event Publishing Tests (additional events) ──────────────────────

func TestInviteCreatedEvent(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.Must(uuid.NewV7())
	inviteID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeFor[InviteCreated](), &captureHandler{capture: &got})

	if err := bus.Publish(ctx, InviteCreated{
		FamilyID: familyID,
		InviteID: inviteID,
		Email:    "invited@example.com",
		Token:    "tok123",
	}); err != nil {
		t.Fatal(err)
	}

	evt, ok := got.(InviteCreated)
	if !ok {
		t.Fatalf("got %T, want InviteCreated", got)
	}
	if evt.FamilyID != familyID || evt.InviteID != inviteID {
		t.Errorf("unexpected event fields: %+v", evt)
	}
	if evt.Email != "invited@example.com" {
		t.Errorf("Email = %q, want %q", evt.Email, "invited@example.com")
	}
}

func TestCoParentRemovedEvent(t *testing.T) {
	ctx := context.Background()
	familyID := uuid.Must(uuid.NewV7())
	coParentID := uuid.Must(uuid.NewV7())

	bus := shared.NewEventBus()
	var got shared.DomainEvent
	bus.Subscribe(reflect.TypeFor[CoParentRemoved](), &captureHandler{capture: &got})

	if err := bus.Publish(ctx, CoParentRemoved{
		FamilyID:   familyID,
		CoParentID: coParentID,
	}); err != nil {
		t.Fatal(err)
	}

	evt, ok := got.(CoParentRemoved)
	if !ok {
		t.Fatalf("got %T, want CoParentRemoved", got)
	}
	if evt.FamilyID != familyID || evt.CoParentID != coParentID {
		t.Errorf("unexpected event fields: %+v", evt)
	}
}
