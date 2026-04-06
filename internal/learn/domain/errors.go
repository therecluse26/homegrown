package domain

import "errors"

// Sentinel errors for the learning domain. [06-learn §17]
// Handlers convert these to AppError via mapLearningError(). [§17.1]

// ─── Student Errors ─────────────────────────────────────────────────────────

var (
	ErrStudentNotFound  = errors.New("student not found")
	ErrStudentNotInFamily = errors.New("student does not belong to this family")
)

// ─── Activity Errors ────────────────────────────────────────────────────────

var (
	ErrActivityNotFound    = errors.New("activity not found")
	ErrActivityDefNotFound = errors.New("activity definition not found")
)

// ─── Journal Errors ─────────────────────────────────────────────────────────

var ErrJournalNotFound = errors.New("journal entry not found")

// ErrInvalidEntryType indicates an invalid journal entry type.
type ErrInvalidEntryType struct {
	EntryType string
}

func (e *ErrInvalidEntryType) Error() string {
	return "invalid entry type: " + e.EntryType
}

// ─── Reading Errors ─────────────────────────────────────────────────────────

var (
	ErrReadingItemNotFound             = errors.New("reading item not found")
	ErrReadingListNotFound             = errors.New("reading list not found")
	ErrReadingProgressNotFound         = errors.New("reading progress not found")
	ErrDuplicateReadingProgress        = errors.New("already tracking this reading item")
	ErrInvalidReadingStatusTransition  = errors.New("invalid reading status transition")
)

// ─── Subject Taxonomy Errors ────────────────────────────────────────────────

var ErrTaxonomyNotFound = errors.New("taxonomy node not found")

// ErrInvalidSubjectTag indicates an invalid subject tag.
type ErrInvalidSubjectTag struct {
	Tag string
}

func (e *ErrInvalidSubjectTag) Error() string {
	return "invalid subject tag: " + e.Tag
}

var ErrDuplicateCustomSubject = errors.New("duplicate custom subject")

// ─── Progress Errors ────────────────────────────────────────────────────────

var ErrSnapshotNotFound = errors.New("progress snapshot not found")

// ─── Validation Errors ──────────────────────────────────────────────────────

var (
	ErrFutureDateNotAllowed = errors.New("activity date cannot be in the future")
	ErrNegativeDuration     = errors.New("duration cannot be negative")
)

// ─── Artifact Link Errors ───────────────────────────────────────────────────

var (
	ErrSourceNotFound = errors.New("source content not found")
	ErrTargetNotFound = errors.New("target content not found")
	ErrDuplicateLink  = errors.New("duplicate artifact link")
	ErrLinkNotFound   = errors.New("artifact link not found")
)

// ErrInvalidArtifactType indicates an invalid artifact type.
type ErrInvalidArtifactType struct {
	ArtifactType string
}

func (e *ErrInvalidArtifactType) Error() string {
	return "invalid artifact type: " + e.ArtifactType
}

// ─── Tool & Tier Errors ────────────────────────────────────────────────────

var (
	ErrToolNotActive  = errors.New("tool not active for this student")
	ErrPremiumRequired = errors.New("premium subscription required")
)

// ─── Export Errors ──────────────────────────────────────────────────────────

var (
	ErrExportAlreadyInProgress = errors.New("export already in progress")
	ErrExportNotReady          = errors.New("export not ready")
	ErrExportExpired           = errors.New("export has expired")
	ErrExportNotFound          = errors.New("export not found")
)

// ─── Publisher Errors ───────────────────────────────────────────────────────

var ErrNotPublisherMember = errors.New("not a member of this publisher")

// ─── Attachment Errors ──────────────────────────────────────────────────────

var (
	ErrAttachmentTooLarge   = errors.New("attachment too large")
	ErrInvalidAttachmentType = errors.New("invalid attachment type")
)

// ─── Assessment Engine Errors ───────────────────────────────────────────────

var (
	ErrQuestionNotFound           = errors.New("question not found")
	ErrQuizDefNotFound            = errors.New("quiz definition not found")
	ErrQuizSessionNotFound        = errors.New("quiz session not found")
	ErrQuizSessionNotSubmitted    = errors.New("quiz session is not in submitted state")
	ErrQuizSessionAlreadySubmitted = errors.New("quiz session is already submitted")
	ErrInvalidAnswerData          = errors.New("answer data does not match question type")
	ErrQuizNoQuestions            = errors.New("quiz must have at least one question")
)

// ─── Sequence Engine Errors ─────────────────────────────────────────────────

var (
	ErrSequenceDefNotFound      = errors.New("sequence definition not found")
	ErrSequenceProgressNotFound = errors.New("sequence progress not found")
	ErrSequenceItemLocked       = errors.New("sequence item is locked")
	ErrSequenceNoItems          = errors.New("sequence must have at least one item")
)

// ─── Assessment/Project/Grading Errors (Phase 2) ────────────────────────────

var (
	ErrAssessmentDefNotFound    = errors.New("assessment definition not found")
	ErrAssessmentResultNotFound = errors.New("assessment result not found")
	ErrProjectDefNotFound       = errors.New("project definition not found")
	ErrProjectProgressNotFound  = errors.New("project progress not found")
	ErrGradingScaleNotFound     = errors.New("grading scale not found")
	ErrInvalidProjectStatusTransition = errors.New("invalid project status transition")
)

// ─── Assignment Errors ──────────────────────────────────────────────────────

var (
	ErrAssignmentNotFound             = errors.New("assignment not found")
	ErrInvalidAssignmentStatusTransition = errors.New("invalid assignment status transition")
)

// ─── Video Errors ───────────────────────────────────────────────────────────

var (
	ErrVideoDefNotFound      = errors.New("video definition not found")
	ErrVideoProgressNotFound = errors.New("video progress not found")
)
