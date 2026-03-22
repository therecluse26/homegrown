package iam

import "errors"

// Sentinel errors for the IAM domain. [§12]
// Handlers convert these to AppError via mapIamError(). [§12.1]
var (
	// ─── Family ───────────────────────────────────────────────────────────────
	ErrFamilyNotFound = errors.New("family not found")

	// ─── Parent ───────────────────────────────────────────────────────────────
	ErrParentNotFound = errors.New("parent not found")

	// ─── Student ──────────────────────────────────────────────────────────────
	ErrStudentNotFound = errors.New("student not found")

	// ─── Co-parent Invite (Phase 2) ───────────────────────────────────────────
	ErrInviteNotFound        = errors.New("invite not found")
	ErrInviteExpired         = errors.New("invite expired")
	ErrInviteAlreadyAccepted = errors.New("invite already accepted")

	// ─── COPPA ────────────────────────────────────────────────────────────────
	ErrCoppaConsentRequired      = errors.New("COPPA consent required")
	ErrConsentVerificationFailed = errors.New("consent verification failed")

	// ─── Authorization ────────────────────────────────────────────────────────
	ErrNotPrimaryParent          = errors.New("not the primary parent")
	ErrCannotRemovePrimaryParent = errors.New("cannot remove primary parent")
	ErrCannotTransferToSelf      = errors.New("cannot transfer primary to self")

	// ─── Conflict ─────────────────────────────────────────────────────────────
	ErrParentAlreadyInFamily    = errors.New("parent already exists in this family")
	ErrEmailAlreadyAssociated   = errors.New("email already associated with a family")
	ErrDeletionAlreadyRequested = errors.New("family deletion already requested")
	ErrNoPendingDeletion        = errors.New("no pending deletion request")

	// ─── Subscription ─────────────────────────────────────────────────────────
	ErrPremiumRequired = errors.New("premium subscription required")

	// ─── Infrastructure ───────────────────────────────────────────────────────
	ErrKratosError = errors.New("kratos communication error")
)

// InvalidConsentTransitionError is a structured error for invalid COPPA state machine transitions.
// Returned by SubmitCoppaConsent when the requested transition is not permitted. [§9.2]
type InvalidConsentTransitionError struct {
	From string
	To   string
}

func (e *InvalidConsentTransitionError) Error() string {
	return "invalid COPPA consent transition from " + e.From + " to " + e.To
}
