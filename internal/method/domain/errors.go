package domain

import (
	"errors"
	"fmt"
)

// Sentinel errors for the method domain. [02-method §10.3]
// Handlers convert these to AppError via mapMethodError(). [§10.4]
var (
	ErrMethodologyNotFound   = errors.New("methodology not found")
	ErrMethodologyNotActive  = errors.New("methodology is not active")
	ErrInvalidMethodologyIDs = errors.New("invalid methodology IDs in selection")
	ErrPrimaryInSecondary    = errors.New("primary methodology cannot also be a secondary")
	ErrDuplicateSecondary    = errors.New("duplicate secondary methodology IDs")
	ErrStudentNotFound       = errors.New("student not found")
	ErrToolNotFound          = errors.New("tool not found")
)

// MethodError wraps a method-specific error with additional context.
// Supports errors.Is/errors.As via Unwrap. [CODING §2.2]
type MethodError struct {
	Err   error
	Slug  string
	Slugs []string
}

func (e *MethodError) Error() string {
	if e.Slug != "" {
		return fmt.Sprintf("%s: %s", e.Err.Error(), e.Slug)
	}
	if len(e.Slugs) > 0 {
		return fmt.Sprintf("%s: %v", e.Err.Error(), e.Slugs)
	}
	return e.Err.Error()
}

func (e *MethodError) Unwrap() error {
	return e.Err
}
