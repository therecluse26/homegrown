package domain

import "errors"

// Domain-level sentinel errors for aggregate root invariant violations. [11-safety §15]

var (
	// Report state machine errors.
	ErrInvalidReportTransition = errors.New("invalid report status transition")

	// Account moderation errors.
	ErrAccountBanned       = errors.New("account is permanently banned")
	ErrAccountSuspended    = errors.New("account is suspended")
	ErrCsamBanNotAppealable = errors.New("CSAM bans are not appealable")
	ErrInvalidActionType   = errors.New("invalid action for current account state")
)
