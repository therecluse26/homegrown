package learn

import (
	"errors"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/learn/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// LearningError wraps a sentinel error with optional context. [CODING §2.2]
type LearningError struct {
	Err error
}

func (e *LearningError) Error() string { return e.Err.Error() }
func (e *LearningError) Unwrap() error { return e.Err }

// toAppError maps a LearningError sentinel to an *shared.AppError with the correct
// HTTP status code. Called by mapLearningError in handlers.go. [06-learn §17.1]
func (e *LearningError) toAppError() *shared.AppError {
	switch {
	// ─── Student ─────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrStudentNotFound):
		return &shared.AppError{Code: "student_not_found", Message: "Student not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrStudentNotInFamily):
		return &shared.AppError{Code: "student_not_in_family", Message: "Student does not belong to this family", StatusCode: http.StatusForbidden}

	// ─── Activity ────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrActivityNotFound):
		return &shared.AppError{Code: "activity_not_found", Message: "Activity not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrActivityDefNotFound):
		return &shared.AppError{Code: "activity_def_not_found", Message: "Activity definition not found", StatusCode: http.StatusNotFound}

	// ─── Journal ─────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrJournalNotFound):
		return &shared.AppError{Code: "journal_not_found", Message: "Journal entry not found", StatusCode: http.StatusNotFound}

	// ─── Reading ─────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrReadingItemNotFound):
		return &shared.AppError{Code: "reading_item_not_found", Message: "Reading item not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrReadingListNotFound):
		return &shared.AppError{Code: "reading_list_not_found", Message: "Reading list not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrReadingProgressNotFound):
		return &shared.AppError{Code: "reading_progress_not_found", Message: "Reading progress not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrDuplicateReadingProgress):
		return &shared.AppError{Code: "duplicate_reading_progress", Message: "Already tracking this reading item", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrInvalidReadingStatusTransition):
		return &shared.AppError{Code: "invalid_status_transition", Message: "Invalid reading status transition", StatusCode: http.StatusUnprocessableEntity}

	// ─── Subject Taxonomy ────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrTaxonomyNotFound):
		return &shared.AppError{Code: "taxonomy_not_found", Message: "Taxonomy node not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrDuplicateCustomSubject):
		return &shared.AppError{Code: "duplicate_custom_subject", Message: "Custom subject already exists", StatusCode: http.StatusConflict}

	// ─── Validation ──────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrFutureDateNotAllowed):
		return &shared.AppError{Code: "future_date_not_allowed", Message: "Date cannot be in the future", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrNegativeDuration):
		return &shared.AppError{Code: "negative_duration", Message: "Duration cannot be negative", StatusCode: http.StatusUnprocessableEntity}

	// ─── Artifact Links ──────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrSourceNotFound):
		return &shared.AppError{Code: "source_not_found", Message: "Source content not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrTargetNotFound):
		return &shared.AppError{Code: "target_not_found", Message: "Target content not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrDuplicateLink):
		return &shared.AppError{Code: "duplicate_link", Message: "Duplicate artifact link", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrLinkNotFound):
		return &shared.AppError{Code: "link_not_found", Message: "Artifact link not found", StatusCode: http.StatusNotFound}

	// ─── Tools & Tier ────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrToolNotActive):
		return &shared.AppError{Code: "tool_not_active", Message: "Tool is not active for this student", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, domain.ErrPremiumRequired):
		return &shared.AppError{Code: "premium_required", Message: "Premium subscription required", StatusCode: http.StatusForbidden}

	// ─── Progress ───────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrSnapshotNotFound):
		return &shared.AppError{Code: "snapshot_not_found", Message: "Progress snapshot not found", StatusCode: http.StatusNotFound}

	// ─── Export ──────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrExportAlreadyInProgress):
		return &shared.AppError{Code: "export_already_in_progress", Message: "Export already in progress", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrExportNotReady):
		return &shared.AppError{Code: "export_not_ready", Message: "Export is not ready yet", StatusCode: http.StatusAccepted}
	case errors.Is(e.Err, domain.ErrExportExpired):
		return &shared.AppError{Code: "export_expired", Message: "Export has expired", StatusCode: http.StatusGone}
	case errors.Is(e.Err, domain.ErrExportNotFound):
		return &shared.AppError{Code: "export_not_found", Message: "Export not found", StatusCode: http.StatusNotFound}

	// ─── Publisher ───────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrNotPublisherMember):
		return &shared.AppError{Code: "not_publisher_member", Message: "Not a member of this publisher", StatusCode: http.StatusForbidden}

	// ─── Attachments ────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrAttachmentTooLarge):
		return &shared.AppError{Code: "attachment_too_large", Message: "Attachment is too large", StatusCode: http.StatusRequestEntityTooLarge}
	case errors.Is(e.Err, domain.ErrInvalidAttachmentType):
		return &shared.AppError{Code: "invalid_attachment_type", Message: "Invalid attachment type", StatusCode: http.StatusUnprocessableEntity}

	// ─── Assessment Engine ──────────────────────────────────────
	case errors.Is(e.Err, domain.ErrQuestionNotFound):
		return &shared.AppError{Code: "question_not_found", Message: "Question not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrQuizDefNotFound):
		return &shared.AppError{Code: "quiz_def_not_found", Message: "Quiz definition not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrQuizSessionNotFound):
		return &shared.AppError{Code: "quiz_session_not_found", Message: "Quiz session not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrQuizSessionNotSubmitted):
		return &shared.AppError{Code: "quiz_session_not_submitted", Message: "Quiz session is not in submitted state", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrQuizSessionAlreadySubmitted):
		return &shared.AppError{Code: "quiz_session_already_submitted", Message: "Quiz session is already submitted", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrInvalidAnswerData):
		return &shared.AppError{Code: "invalid_answer_data", Message: "Answer data does not match question type", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrQuizNoQuestions):
		return &shared.AppError{Code: "quiz_no_questions", Message: "Quiz must have at least one question", StatusCode: http.StatusUnprocessableEntity}

	// ─── Sequence Engine ────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrSequenceDefNotFound):
		return &shared.AppError{Code: "sequence_def_not_found", Message: "Sequence definition not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrSequenceProgressNotFound):
		return &shared.AppError{Code: "sequence_progress_not_found", Message: "Sequence progress not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrSequenceItemLocked):
		return &shared.AppError{Code: "sequence_item_locked", Message: "Sequence item is locked", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrSequenceNoItems):
		return &shared.AppError{Code: "sequence_no_items", Message: "Sequence must have at least one item", StatusCode: http.StatusUnprocessableEntity}

	// ─── Assignment ─────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrAssignmentNotFound):
		return &shared.AppError{Code: "assignment_not_found", Message: "Assignment not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrInvalidAssignmentStatusTransition):
		return &shared.AppError{Code: "invalid_assignment_status", Message: "Invalid assignment status transition", StatusCode: http.StatusUnprocessableEntity}

	// ─── Video ──────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrVideoDefNotFound):
		return &shared.AppError{Code: "video_def_not_found", Message: "Video definition not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrVideoProgressNotFound):
		return &shared.AppError{Code: "video_progress_not_found", Message: "Video progress not found", StatusCode: http.StatusNotFound}

	// ─── Phase 2: Assessment/Project/Grading ──────────────��─────
	case errors.Is(e.Err, domain.ErrAssessmentDefNotFound):
		return &shared.AppError{Code: "assessment_def_not_found", Message: "Assessment definition not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrAssessmentResultNotFound):
		return &shared.AppError{Code: "assessment_result_not_found", Message: "Assessment result not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrProjectDefNotFound):
		return &shared.AppError{Code: "project_def_not_found", Message: "Project definition not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrProjectProgressNotFound):
		return &shared.AppError{Code: "project_progress_not_found", Message: "Project progress not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrGradingScaleNotFound):
		return &shared.AppError{Code: "grading_scale_not_found", Message: "Grading scale not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrInvalidProjectStatusTransition):
		return &shared.AppError{Code: "invalid_project_status", Message: "Invalid project status transition", StatusCode: http.StatusUnprocessableEntity}

	default:
		// Check for structured error types
		var invalidEntryType *domain.ErrInvalidEntryType
		if errors.As(e.Err, &invalidEntryType) {
			return &shared.AppError{Code: "invalid_entry_type", Message: invalidEntryType.Error(), StatusCode: http.StatusUnprocessableEntity}
		}
		var invalidSubjectTag *domain.ErrInvalidSubjectTag
		if errors.As(e.Err, &invalidSubjectTag) {
			return &shared.AppError{Code: "invalid_subject_tag", Message: invalidSubjectTag.Error(), StatusCode: http.StatusUnprocessableEntity}
		}
		var invalidArtifactType *domain.ErrInvalidArtifactType
		if errors.As(e.Err, &invalidArtifactType) {
			return &shared.AppError{Code: "invalid_artifact_type", Message: invalidArtifactType.Error(), StatusCode: http.StatusUnprocessableEntity}
		}
		return shared.ErrInternal(e)
	}
}

// mapLearningError maps any error to an Echo-compatible HTTP error.
// If err is a *LearningError, maps it to an AppError via toAppError().
// Otherwise returns it as-is for Echo's default error handling.
func mapLearningError(err error) error {
	var le *LearningError
	if errors.As(err, &le) {
		return le.toAppError()
	}
	return err
}
