package domain

import (
	"testing"

	"github.com/google/uuid"
)

func newTestReport(category string) *ModerationReport {
	return NewModerationReport(
		uuid.Must(uuid.NewV7()),
		uuid.Must(uuid.NewV7()),
		uuid.Must(uuid.NewV7()),
		"post",
		uuid.Must(uuid.NewV7()),
		nil,
		category,
		nil,
	)
}

// ─── A1: Priority Derivation ────────────────────────────────────────

func TestNewModerationReport_csam_derives_critical(t *testing.T) {
	r := newTestReport("csam_child_safety")
	if r.Priority() != ReportPriorityCritical {
		t.Errorf("priority = %s, want critical", r.Priority())
	}
}

func TestNewModerationReport_harassment_derives_high(t *testing.T) {
	r := newTestReport("harassment")
	if r.Priority() != ReportPriorityHigh {
		t.Errorf("priority = %s, want high", r.Priority())
	}
}

func TestNewModerationReport_spam_derives_normal(t *testing.T) {
	r := newTestReport("spam")
	if r.Priority() != ReportPriorityNormal {
		t.Errorf("priority = %s, want normal", r.Priority())
	}
}

func TestNewModerationReport_other_derives_normal(t *testing.T) {
	r := newTestReport("other")
	if r.Priority() != ReportPriorityNormal {
		t.Errorf("priority = %s, want normal", r.Priority())
	}
}

func TestNewModerationReport_defaults(t *testing.T) {
	r := newTestReport("spam")
	if r.Status() != ReportStatusPending {
		t.Errorf("status = %s, want pending", r.Status())
	}
	if r.AssignedAdminID() != nil {
		t.Error("expected nil assignedAdminID")
	}
}

// ─── A1: Assign ─────────────────────────────────────────────────────

func TestAssign_from_pending_succeeds(t *testing.T) {
	r := newTestReport("spam")
	adminID := uuid.Must(uuid.NewV7())
	if err := r.Assign(adminID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status() != ReportStatusInReview {
		t.Errorf("status = %s, want in_review", r.Status())
	}
	if *r.AssignedAdminID() != adminID {
		t.Error("assignedAdminID not set")
	}
}

func TestAssign_from_in_review_succeeds(t *testing.T) {
	r := newTestReport("spam")
	_ = r.Assign(uuid.Must(uuid.NewV7()))
	newAdmin := uuid.Must(uuid.NewV7())
	if err := r.Assign(newAdmin); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *r.AssignedAdminID() != newAdmin {
		t.Error("reassign did not update admin")
	}
}

func TestAssign_from_resolved_action_taken_fails(t *testing.T) {
	r := newTestReport("spam")
	_ = r.Assign(uuid.Must(uuid.NewV7()))
	_ = r.ResolveActionTaken()
	err := r.Assign(uuid.Must(uuid.NewV7()))
	if err != ErrInvalidReportTransition {
		t.Errorf("err = %v, want ErrInvalidReportTransition", err)
	}
}

func TestAssign_from_dismissed_fails(t *testing.T) {
	r := newTestReport("spam")
	_ = r.Dismiss()
	err := r.Assign(uuid.Must(uuid.NewV7()))
	if err != ErrInvalidReportTransition {
		t.Errorf("err = %v, want ErrInvalidReportTransition", err)
	}
}

// ─── A1: ResolveActionTaken ─────────────────────────────────────────

func TestResolveActionTaken_from_in_review_succeeds(t *testing.T) {
	r := newTestReport("spam")
	_ = r.Assign(uuid.Must(uuid.NewV7()))
	if err := r.ResolveActionTaken(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status() != ReportStatusResolvedActionTaken {
		t.Errorf("status = %s, want resolved_action_taken", r.Status())
	}
	if r.ResolvedAt() == nil {
		t.Error("expected resolvedAt to be set")
	}
}

func TestResolveActionTaken_from_pending_fails(t *testing.T) {
	r := newTestReport("spam")
	err := r.ResolveActionTaken()
	if err != ErrInvalidReportTransition {
		t.Errorf("err = %v, want ErrInvalidReportTransition", err)
	}
}

// ─── A1: ResolveNoAction ────────────────────────────────────────────

func TestResolveNoAction_from_in_review_succeeds(t *testing.T) {
	r := newTestReport("spam")
	_ = r.Assign(uuid.Must(uuid.NewV7()))
	if err := r.ResolveNoAction(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status() != ReportStatusResolvedNoAction {
		t.Errorf("status = %s, want resolved_no_action", r.Status())
	}
}

func TestResolveNoAction_from_pending_fails(t *testing.T) {
	r := newTestReport("spam")
	err := r.ResolveNoAction()
	if err != ErrInvalidReportTransition {
		t.Errorf("err = %v, want ErrInvalidReportTransition", err)
	}
}

// ─── A1: Dismiss ────────────────────────────────────────────────────

func TestDismiss_from_pending_succeeds(t *testing.T) {
	r := newTestReport("spam")
	if err := r.Dismiss(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status() != ReportStatusDismissed {
		t.Errorf("status = %s, want dismissed", r.Status())
	}
}

func TestDismiss_from_in_review_succeeds(t *testing.T) {
	r := newTestReport("spam")
	_ = r.Assign(uuid.Must(uuid.NewV7()))
	if err := r.Dismiss(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Status() != ReportStatusDismissed {
		t.Errorf("status = %s, want dismissed", r.Status())
	}
}

func TestDismiss_from_resolved_action_taken_fails(t *testing.T) {
	r := newTestReport("spam")
	_ = r.Assign(uuid.Must(uuid.NewV7()))
	_ = r.ResolveActionTaken()
	err := r.Dismiss()
	if err != ErrInvalidReportTransition {
		t.Errorf("err = %v, want ErrInvalidReportTransition", err)
	}
}

func TestDismiss_from_dismissed_fails(t *testing.T) {
	r := newTestReport("spam")
	_ = r.Dismiss()
	err := r.Dismiss()
	if err != ErrInvalidReportTransition {
		t.Errorf("err = %v, want ErrInvalidReportTransition", err)
	}
}
