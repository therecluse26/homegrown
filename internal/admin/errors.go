package admin

import "errors"

// Sentinel errors for the admin domain. [16-admin §13]
var (
	ErrFlagNotFound            = errors.New("feature flag not found")
	ErrFlagAlreadyExists       = errors.New("feature flag key already exists")
	ErrInvalidFlagKey          = errors.New("invalid flag key format")
	ErrUserNotFound            = errors.New("user not found")
	ErrDeadLetterNotFound      = errors.New("dead-letter job not found")
	ErrModerationItemNotFound  = errors.New("moderation queue item not found")
	ErrMethodologyNotFound     = errors.New("methodology not found")
	ErrRecoveryRequestNotFound = errors.New("recovery request not found")
)
