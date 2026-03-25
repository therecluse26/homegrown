package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func newTestAccountState() *AccountModerationState {
	return NewAccountModerationState(uuid.Must(uuid.NewV7()))
}

// ─── A2: Defaults ───────────────────────────────────────────────────

func TestNewAccountState_defaults_active(t *testing.T) {
	s := newTestAccountState()
	if s.Status() != AccountStatusActive {
		t.Errorf("status = %s, want active", s.Status())
	}
	if s.SuspendedAt() != nil {
		t.Error("expected nil suspendedAt")
	}
	if s.BannedAt() != nil {
		t.Error("expected nil bannedAt")
	}
}

// ─── A2: Suspend ────────────────────────────────────────────────────

func TestSuspend_from_active_succeeds(t *testing.T) {
	s := newTestAccountState()
	adminID := uuid.Must(uuid.NewV7())
	evt, err := s.Suspend(adminID, "policy violation", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status() != AccountStatusSuspended {
		t.Errorf("status = %s, want suspended", s.Status())
	}
	if evt == nil {
		t.Fatal("expected AccountSuspendedEvent")
	}
	if evt.FamilyID != s.FamilyID() {
		t.Error("event familyID mismatch")
	}
}

func TestSuspend_sets_expiry(t *testing.T) {
	s := newTestAccountState()
	before := time.Now().UTC()
	_, _ = s.Suspend(uuid.Must(uuid.NewV7()), "test", 7)
	after := time.Now().UTC()

	if s.SuspensionExpiresAt() == nil {
		t.Fatal("expected non-nil expiresAt")
	}
	expires := *s.SuspensionExpiresAt()
	expected := before.AddDate(0, 0, 7)
	if expires.Before(expected.Add(-time.Second)) || expires.After(after.AddDate(0, 0, 7).Add(time.Second)) {
		t.Errorf("expiresAt = %v, expected ~%v", expires, expected)
	}
}

func TestSuspend_from_banned_fails(t *testing.T) {
	s := newTestAccountState()
	_ = s.Ban("bad")
	_, err := s.Suspend(uuid.Must(uuid.NewV7()), "test", 7)
	if !errors.Is(err, ErrAccountBanned) {
		t.Errorf("err = %v, want ErrAccountBanned", err)
	}
}

// ─── A2: Ban ────────────────────────────────────────────────────────

func TestBan_from_active_succeeds(t *testing.T) {
	s := newTestAccountState()
	if err := s.Ban("severe violation"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status() != AccountStatusBanned {
		t.Errorf("status = %s, want banned", s.Status())
	}
	if s.SuspensionExpiresAt() != nil {
		t.Error("expected nil suspensionExpiresAt after ban")
	}
}

func TestBan_from_suspended_clears_suspension(t *testing.T) {
	s := newTestAccountState()
	_, _ = s.Suspend(uuid.Must(uuid.NewV7()), "test", 7)
	if err := s.Ban("escalation"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status() != AccountStatusBanned {
		t.Errorf("status = %s, want banned", s.Status())
	}
	if s.SuspensionExpiresAt() != nil {
		t.Error("suspensionExpiresAt should be nil after ban")
	}
	if s.SuspendedAt() != nil {
		t.Error("suspendedAt should be nil after ban")
	}
}

func TestBan_from_banned_fails(t *testing.T) {
	s := newTestAccountState()
	_ = s.Ban("first")
	err := s.Ban("second")
	if !errors.Is(err, ErrAccountBanned) {
		t.Errorf("err = %v, want ErrAccountBanned", err)
	}
}

// ─── A2: LiftSuspension ────────────────────────────────────────────

func TestLiftSuspension_from_suspended_succeeds(t *testing.T) {
	s := newTestAccountState()
	_, _ = s.Suspend(uuid.Must(uuid.NewV7()), "test", 7)
	if err := s.LiftSuspension(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status() != AccountStatusActive {
		t.Errorf("status = %s, want active", s.Status())
	}
}

func TestLiftSuspension_from_active_fails(t *testing.T) {
	s := newTestAccountState()
	err := s.LiftSuspension()
	if !errors.Is(err, ErrInvalidActionType) {
		t.Errorf("err = %v, want ErrInvalidActionType", err)
	}
}

func TestLiftSuspension_from_banned_fails(t *testing.T) {
	s := newTestAccountState()
	_ = s.Ban("bad")
	err := s.LiftSuspension()
	if !errors.Is(err, ErrInvalidActionType) {
		t.Errorf("err = %v, want ErrInvalidActionType", err)
	}
}

// ─── A2: CheckExpiry ────────────────────────────────────────────────

func TestCheckExpiry_expired_suspension_returns_true(t *testing.T) {
	s := newTestAccountState()
	_, _ = s.Suspend(uuid.Must(uuid.NewV7()), "test", 7)
	// Manually set expiry to the past.
	past := time.Now().Add(-time.Hour)
	s.suspensionExpiresAt = &past

	if !s.CheckExpiry() {
		t.Error("expected true for expired suspension")
	}
	if s.Status() != AccountStatusActive {
		t.Errorf("status = %s, want active after expiry", s.Status())
	}
}

func TestCheckExpiry_non_expired_returns_false(t *testing.T) {
	s := newTestAccountState()
	_, _ = s.Suspend(uuid.Must(uuid.NewV7()), "test", 7)

	if s.CheckExpiry() {
		t.Error("expected false for non-expired suspension")
	}
	if s.Status() != AccountStatusSuspended {
		t.Errorf("status = %s, want suspended", s.Status())
	}
}

func TestCheckExpiry_active_returns_false(t *testing.T) {
	s := newTestAccountState()
	if s.CheckExpiry() {
		t.Error("expected false for active account")
	}
}

func TestCheckExpiry_banned_returns_false(t *testing.T) {
	s := newTestAccountState()
	_ = s.Ban("bad")
	if s.CheckExpiry() {
		t.Error("expected false for banned account")
	}
}

// ─── A2: Unban ──────────────────────────────────────────────────────

func TestUnban_from_banned_non_csam_succeeds(t *testing.T) {
	s := newTestAccountState()
	_ = s.Ban("policy violation")
	if err := s.Unban(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status() != AccountStatusActive {
		t.Errorf("status = %s, want active", s.Status())
	}
}

func TestUnban_csam_ban_fails(t *testing.T) {
	s := newTestAccountState()
	_ = s.Ban("csam_violation")
	err := s.Unban()
	if !errors.Is(err, ErrCsamBanNotAppealable) {
		t.Errorf("err = %v, want ErrCsamBanNotAppealable", err)
	}
}

func TestUnban_from_active_fails(t *testing.T) {
	s := newTestAccountState()
	err := s.Unban()
	if !errors.Is(err, ErrInvalidActionType) {
		t.Errorf("err = %v, want ErrInvalidActionType", err)
	}
}
