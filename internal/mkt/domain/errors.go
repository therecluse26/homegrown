package domain

import "fmt"

// MktDomainErrorKind enumerates domain-level error causes. [CODING §8.3]
type MktDomainErrorKind int

const (
	ErrInvalidStateTransition MktDomainErrorKind = iota
	ErrListingHasNoFiles
	ErrInvalidPrice
)

// MktDomainError represents a violation of a marketplace aggregate invariant.
// Converted to shared.AppError in service.go. [CODING §8.3]
type MktDomainError struct {
	Kind   MktDomainErrorKind
	From   string // for InvalidStateTransition
	Action string // for InvalidStateTransition
}

func (e *MktDomainError) Error() string {
	switch e.Kind {
	case ErrInvalidStateTransition:
		return fmt.Sprintf("invalid state transition from %s via %s", e.From, e.Action)
	case ErrListingHasNoFiles:
		return "listing has no files attached"
	case ErrInvalidPrice:
		return "invalid price: must be >= 0"
	default:
		return "unknown domain error"
	}
}
