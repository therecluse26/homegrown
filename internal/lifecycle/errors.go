package lifecycle

import "errors"

// Sentinel error variables for the Data Lifecycle domain. [15-data-lifecycle §16]

var (
	// ErrExportNotFound is returned when an export request cannot be found for the family.
	ErrExportNotFound = errors.New("export request not found")

	// ErrExportExpired is returned when an export archive has expired and is no longer downloadable.
	ErrExportExpired = errors.New("export has expired")

	// ErrDeletionAlreadyPending is returned when a family already has an active deletion request.
	ErrDeletionAlreadyPending = errors.New("an active deletion request already exists")

	// ErrGracePeriodExpired is returned when attempting to cancel a deletion after the grace period.
	ErrGracePeriodExpired = errors.New("cannot cancel deletion — grace period has ended")

	// ErrNotPrimaryParent is returned when a non-primary parent requests family deletion.
	ErrNotPrimaryParent = errors.New("only the primary parent can request family deletion")

	// ErrRecoveryNotFound is returned when a recovery request cannot be found.
	ErrRecoveryNotFound = errors.New("recovery request not found or expired")

	// ErrRecoveryExpired is returned when a recovery request has expired.
	ErrRecoveryExpired = errors.New("recovery request has expired")

	// ErrCannotRevokeCurrent is returned when attempting to revoke the current session.
	ErrCannotRevokeCurrent = errors.New("cannot revoke current session via this endpoint")

	// ErrDeletionNotFound is returned when no active deletion request exists for the family.
	ErrDeletionNotFound = errors.New("no active deletion request found")
)
