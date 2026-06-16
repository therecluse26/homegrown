package learner_profile

import (
	"errors"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// Domain error sentinels. [18-learner-profile §9]

// ErrProfileNotFound is returned when no learner profile exists for a student.
var ErrProfileNotFound = &shared.AppError{Code: "not_found", Message: "no learner profile found for this student", StatusCode: 404}

// ErrStudentNotInFamily is returned when a student ID does not belong to the
// authenticated family — prevents cross-family data access.
var ErrStudentNotInFamily = &shared.AppError{Code: "forbidden", Message: "student does not belong to this family", StatusCode: 403}

// IsProfileNotFound reports whether err wraps ErrProfileNotFound.
func IsProfileNotFound(err error) bool {
	var ae *shared.AppError
	return errors.As(err, &ae) && ae.Code == "not_found"
}
