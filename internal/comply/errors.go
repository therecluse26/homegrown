package comply

import (
	"errors"
	"fmt"
)

// Service-layer error sentinels and custom error types. [14-comply §16]
// Domain-layer errors (used by domain/ pure functions) are in domain/errors.go.
// Service imports domain and propagates those errors via errors.Is().

// ─── Configuration Errors ──────────────────────────────────────────

var (
	ErrFamilyConfigNotFound   = errors.New("family config not found")
	ErrInvalidStateCode       = errors.New("invalid state code")
	ErrStateConfigNotFound    = errors.New("state config not found")
	ErrInvalidSchoolYearRange = errors.New("invalid school year date range")
)

// ─── Schedule Errors ───────────────────────────────────────────────

var (
	ErrScheduleNotFound      = errors.New("schedule not found")
	ErrScheduleInUse         = errors.New("schedule in use by family config")
	ErrInvalidSchoolDaysArray = errors.New("invalid school days array — must have 7 elements")
)

// ─── Attendance Errors ─────────────────────────────────────────────

var (
	ErrAttendanceNotFound          = errors.New("attendance record not found")
	ErrBulkAttendanceLimitExceeded = errors.New("bulk attendance exceeds maximum of 31 records")
)

// ─── Assessment Errors ─────────────────────────────────────────────

var (
	ErrAssessmentNotFound    = errors.New("assessment record not found")
	ErrInvalidAssessmentType = errors.New("invalid assessment type")
)

// ─── Test Score Errors ─────────────────────────────────────────────

var ErrTestScoreNotFound = errors.New("test score not found")

// ─── Portfolio Errors ──────────────────────────────────────────────

var (
	ErrPortfolioNotFound           = errors.New("portfolio not found")
	ErrPortfolioExpired            = errors.New("portfolio has expired")
	ErrPortfolioItemSourceNotFound = errors.New("portfolio item source not found in learning domain")
	ErrDuplicatePortfolioItem      = errors.New("duplicate item in portfolio")
)

// ─── Transcript Errors (Phase 3) ───────────────────────────────────

var (
	ErrTranscriptNotFound = errors.New("transcript not found")
	ErrCourseNotFound     = errors.New("course not found")
	ErrInvalidCourseLevel = errors.New("invalid course level")
)

// ─── Student Errors ────────────────────────────────────────────────

var ErrStudentNotInFamily = errors.New("student not found in family")

// ─── Infrastructure ────────────────────────────────────────────────

// DbError wraps a database error — internal, NOT exposed in API.
type DbError struct {
	Err error
}

func (e *DbError) Error() string { return fmt.Sprintf("database error: %v", e.Err) }
func (e *DbError) Unwrap() error { return e.Err }

// PdfGenerationError wraps a PDF generation failure — internal, NOT exposed in API.
type PdfGenerationError struct {
	Detail string
}

func (e *PdfGenerationError) Error() string {
	return fmt.Sprintf("PDF generation failed: %s", e.Detail)
}

// MediaServiceError wraps a media service error — internal, NOT exposed in API.
type MediaServiceError struct {
	Detail string
}

func (e *MediaServiceError) Error() string {
	return fmt.Sprintf("media service error: %s", e.Detail)
}
