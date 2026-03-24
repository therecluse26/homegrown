package learn

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/learn/domain"
)

// ─── Activity Domain Invariant Tests ────────────────────────────────────────

func TestNewActivity_NegativeDuration(t *testing.T) {
	dur := int16(-5)
	_, err := domain.NewActivity(testStudentID, "Test", nil, time.Now(), &dur)
	if err == nil {
		t.Fatal("expected error for negative duration")
	}
	if !errors.Is(err, domain.ErrNegativeDuration) {
		t.Errorf("expected ErrNegativeDuration, got %v", err)
	}
}

func TestNewActivity_FutureDate(t *testing.T) {
	futureDate := time.Now().Add(48 * time.Hour)
	_, err := domain.NewActivity(testStudentID, "Test", nil, futureDate, nil)
	if err == nil {
		t.Fatal("expected error for future date")
	}
	if !errors.Is(err, domain.ErrFutureDateNotAllowed) {
		t.Errorf("expected ErrFutureDateNotAllowed, got %v", err)
	}
}

func TestNewActivity_ValidToday(t *testing.T) {
	dur := int16(30)
	a, err := domain.NewActivity(testStudentID, "Math worksheet", []string{"math"}, time.Now(), &dur)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.Title() != "Math worksheet" {
		t.Errorf("expected title 'Math worksheet', got %s", a.Title())
	}
	if *a.DurationMinutes() != 30 {
		t.Errorf("expected duration 30, got %d", *a.DurationMinutes())
	}
}

func TestNewActivity_ValidPastDate(t *testing.T) {
	yesterday := time.Now().Add(-24 * time.Hour)
	_, err := domain.NewActivity(testStudentID, "Past activity", nil, yesterday, nil)
	if err != nil {
		t.Fatalf("unexpected error for past date: %v", err)
	}
}

func TestNewActivity_ZeroDuration(t *testing.T) {
	dur := int16(0)
	_, err := domain.NewActivity(testStudentID, "Quick task", nil, time.Now(), &dur)
	if err != nil {
		t.Fatalf("unexpected error for zero duration: %v", err)
	}
}

func TestNewActivity_NilDuration(t *testing.T) {
	_, err := domain.NewActivity(testStudentID, "Untracked", nil, time.Now(), nil)
	if err != nil {
		t.Fatalf("unexpected error for nil duration: %v", err)
	}
}

// ─── Error Mapping Tests ────────────────────────────────────────────────────

func TestLearningError_ToAppError_StudentNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrStudentNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "student_not_found" {
		t.Errorf("expected student_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_FutureDate(t *testing.T) {
	le := &LearningError{Err: domain.ErrFutureDateNotAllowed}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
	if ae.Code != "future_date_not_allowed" {
		t.Errorf("expected future_date_not_allowed, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_NegativeDuration(t *testing.T) {
	le := &LearningError{Err: domain.ErrNegativeDuration}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
}

func TestLearningError_ToAppError_InvalidSubjectTag(t *testing.T) {
	le := &LearningError{Err: &domain.ErrInvalidSubjectTag{Tag: "bogus"}}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
	if ae.Code != "invalid_subject_tag" {
		t.Errorf("expected invalid_subject_tag, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_StudentNotInFamily(t *testing.T) {
	le := &LearningError{Err: domain.ErrStudentNotInFamily}
	ae := le.toAppError()
	if ae.StatusCode != 403 {
		t.Errorf("expected 403, got %d", ae.StatusCode)
	}
}

func TestMapLearningError_NonLearningError(t *testing.T) {
	original := errors.New("some other error")
	result := mapLearningError(original)
	if result != original {
		t.Error("expected non-LearningError to pass through")
	}
}

func TestMapLearningError_LearningError(t *testing.T) {
	le := &LearningError{Err: domain.ErrActivityNotFound}
	result := mapLearningError(le)
	if result == le {
		t.Error("expected LearningError to be converted")
	}
}

// ─── Reading Status Transition Tests ─────────────────────────────────────────

func TestReadingStatusTransition_ToReadToInProgress(t *testing.T) {
	err := domain.ValidateReadingStatusTransition(domain.ReadingStatusToRead, domain.ReadingStatusInProgress)
	if err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}
}

func TestReadingStatusTransition_InProgressToCompleted(t *testing.T) {
	err := domain.ValidateReadingStatusTransition(domain.ReadingStatusInProgress, domain.ReadingStatusCompleted)
	if err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}
}

func TestReadingStatusTransition_ToReadToCompleted_Invalid(t *testing.T) {
	err := domain.ValidateReadingStatusTransition(domain.ReadingStatusToRead, domain.ReadingStatusCompleted)
	if err == nil {
		t.Fatal("expected error for to_read → completed skip")
	}
	if !errors.Is(err, domain.ErrInvalidReadingStatusTransition) {
		t.Errorf("expected ErrInvalidReadingStatusTransition, got %v", err)
	}
}

func TestReadingStatusTransition_CompletedToInProgress_Invalid(t *testing.T) {
	err := domain.ValidateReadingStatusTransition(domain.ReadingStatusCompleted, domain.ReadingStatusInProgress)
	if err == nil {
		t.Fatal("expected error for completed → in_progress reverse")
	}
}

func TestReadingStatusTransition_SameStatus_Invalid(t *testing.T) {
	err := domain.ValidateReadingStatusTransition(domain.ReadingStatusInProgress, domain.ReadingStatusInProgress)
	if err == nil {
		t.Fatal("expected error for same-status transition")
	}
}

// ─── Journal Entry Type Validation Tests ─────────────────────────────────────

func TestValidateEntryType_Freeform(t *testing.T) {
	if err := domain.ValidateEntryType("freeform"); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateEntryType_Narration(t *testing.T) {
	if err := domain.ValidateEntryType("narration"); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateEntryType_Reflection(t *testing.T) {
	if err := domain.ValidateEntryType("reflection"); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestValidateEntryType_Invalid(t *testing.T) {
	err := domain.ValidateEntryType("essay")
	if err == nil {
		t.Fatal("expected error for invalid entry type")
	}
	var invalidType *domain.ErrInvalidEntryType
	if !errors.As(err, &invalidType) {
		t.Errorf("expected ErrInvalidEntryType, got %T", err)
	}
}

// ─── Error Mapping Tests (Batch 3-4 Errors) ──────────────────────────────────

func TestLearningError_ToAppError_ReadingItemNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrReadingItemNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "reading_item_not_found" {
		t.Errorf("expected reading_item_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_DuplicateReadingProgress(t *testing.T) {
	le := &LearningError{Err: domain.ErrDuplicateReadingProgress}
	ae := le.toAppError()
	if ae.StatusCode != 409 {
		t.Errorf("expected 409, got %d", ae.StatusCode)
	}
	if ae.Code != "duplicate_reading_progress" {
		t.Errorf("expected duplicate_reading_progress, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_InvalidStatusTransition(t *testing.T) {
	le := &LearningError{Err: domain.ErrInvalidReadingStatusTransition}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
}

func TestLearningError_ToAppError_InvalidEntryType(t *testing.T) {
	le := &LearningError{Err: &domain.ErrInvalidEntryType{EntryType: "essay"}}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
	if ae.Code != "invalid_entry_type" {
		t.Errorf("expected invalid_entry_type, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_DuplicateCustomSubject(t *testing.T) {
	le := &LearningError{Err: domain.ErrDuplicateCustomSubject}
	ae := le.toAppError()
	if ae.StatusCode != 409 {
		t.Errorf("expected 409, got %d", ae.StatusCode)
	}
	if ae.Code != "duplicate_custom_subject" {
		t.Errorf("expected duplicate_custom_subject, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_JournalNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrJournalNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "journal_not_found" {
		t.Errorf("expected journal_not_found, got %s", ae.Code)
	}
}

// ─── Error Mapping Tests (Batch 5-6 Errors) ──────────────────────────────────

func TestLearningError_ToAppError_ExportAlreadyInProgress(t *testing.T) {
	le := &LearningError{Err: domain.ErrExportAlreadyInProgress}
	ae := le.toAppError()
	if ae.StatusCode != 409 {
		t.Errorf("expected 409, got %d", ae.StatusCode)
	}
	if ae.Code != "export_already_in_progress" {
		t.Errorf("expected export_already_in_progress, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_ExportNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrExportNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "export_not_found" {
		t.Errorf("expected export_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_ExportNotReady(t *testing.T) {
	le := &LearningError{Err: domain.ErrExportNotReady}
	ae := le.toAppError()
	if ae.StatusCode != 202 {
		t.Errorf("expected 202, got %d", ae.StatusCode)
	}
}

func TestLearningError_ToAppError_ExportExpired(t *testing.T) {
	le := &LearningError{Err: domain.ErrExportExpired}
	ae := le.toAppError()
	if ae.StatusCode != 410 {
		t.Errorf("expected 410, got %d", ae.StatusCode)
	}
}

func TestLearningError_ToAppError_LinkNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrLinkNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "link_not_found" {
		t.Errorf("expected link_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_DuplicateLink(t *testing.T) {
	le := &LearningError{Err: domain.ErrDuplicateLink}
	ae := le.toAppError()
	if ae.StatusCode != 409 {
		t.Errorf("expected 409, got %d", ae.StatusCode)
	}
	if ae.Code != "duplicate_link" {
		t.Errorf("expected duplicate_link, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_QuestionNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrQuestionNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "question_not_found" {
		t.Errorf("expected question_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_QuizDefNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrQuizDefNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "quiz_def_not_found" {
		t.Errorf("expected quiz_def_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_QuizSessionNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrQuizSessionNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "quiz_session_not_found" {
		t.Errorf("expected quiz_session_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_QuizSessionNotSubmitted(t *testing.T) {
	le := &LearningError{Err: domain.ErrQuizSessionNotSubmitted}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
	if ae.Code != "quiz_session_not_submitted" {
		t.Errorf("expected quiz_session_not_submitted, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_QuizSessionAlreadySubmitted(t *testing.T) {
	le := &LearningError{Err: domain.ErrQuizSessionAlreadySubmitted}
	ae := le.toAppError()
	if ae.StatusCode != 409 {
		t.Errorf("expected 409, got %d", ae.StatusCode)
	}
	if ae.Code != "quiz_session_already_submitted" {
		t.Errorf("expected quiz_session_already_submitted, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_QuizNoQuestions(t *testing.T) {
	le := &LearningError{Err: domain.ErrQuizNoQuestions}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
	if ae.Code != "quiz_no_questions" {
		t.Errorf("expected quiz_no_questions, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_InvalidAnswerData(t *testing.T) {
	le := &LearningError{Err: domain.ErrInvalidAnswerData}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
	if ae.Code != "invalid_answer_data" {
		t.Errorf("expected invalid_answer_data, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_SourceNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrSourceNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "source_not_found" {
		t.Errorf("expected source_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_TargetNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrTargetNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "target_not_found" {
		t.Errorf("expected target_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_ToolNotActive(t *testing.T) {
	le := &LearningError{Err: domain.ErrToolNotActive}
	ae := le.toAppError()
	if ae.StatusCode != 403 {
		t.Errorf("expected 403, got %d", ae.StatusCode)
	}
	if ae.Code != "tool_not_active" {
		t.Errorf("expected tool_not_active, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_PremiumRequired(t *testing.T) {
	le := &LearningError{Err: domain.ErrPremiumRequired}
	ae := le.toAppError()
	if ae.StatusCode != 403 {
		t.Errorf("expected 403, got %d", ae.StatusCode)
	}
	if ae.Code != "premium_required" {
		t.Errorf("expected premium_required, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_InvalidArtifactType(t *testing.T) {
	le := &LearningError{Err: &domain.ErrInvalidArtifactType{ArtifactType: "bogus"}}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
	if ae.Code != "invalid_artifact_type" {
		t.Errorf("expected invalid_artifact_type, got %s", ae.Code)
	}
}

// ─── Assignment Status Transition Tests ──────────────────────────────────────

func TestAssignmentStatusTransition_AssignedToInProgress(t *testing.T) {
	err := domain.ValidateAssignmentStatusTransition("assigned", "in_progress")
	if err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}
}

func TestAssignmentStatusTransition_InProgressToCompleted(t *testing.T) {
	err := domain.ValidateAssignmentStatusTransition("in_progress", "completed")
	if err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}
}

func TestAssignmentStatusTransition_AssignedToSkipped(t *testing.T) {
	err := domain.ValidateAssignmentStatusTransition("assigned", "skipped")
	if err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}
}

func TestAssignmentStatusTransition_InProgressToSkipped(t *testing.T) {
	err := domain.ValidateAssignmentStatusTransition("in_progress", "skipped")
	if err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}
}

func TestAssignmentStatusTransition_CompletedToInProgress_Invalid(t *testing.T) {
	err := domain.ValidateAssignmentStatusTransition("completed", "in_progress")
	if err == nil {
		t.Fatal("expected error for completed → in_progress reverse")
	}
	if !errors.Is(err, domain.ErrInvalidAssignmentStatusTransition) {
		t.Errorf("expected ErrInvalidAssignmentStatusTransition, got %v", err)
	}
}

func TestAssignmentStatusTransition_SkippedToAssigned_Invalid(t *testing.T) {
	err := domain.ValidateAssignmentStatusTransition("skipped", "assigned")
	if err == nil {
		t.Fatal("expected error for skipped → assigned (terminal state)")
	}
}

func TestAssignmentStatusTransition_AssignedToCompleted_Invalid(t *testing.T) {
	err := domain.ValidateAssignmentStatusTransition("assigned", "completed")
	if err == nil {
		t.Fatal("expected error for assigned → completed skip")
	}
}

// ─── Error Mapping Tests (Batch 7-8 Errors) ──────────────────────────────────

func TestLearningError_ToAppError_SequenceDefNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrSequenceDefNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "sequence_def_not_found" {
		t.Errorf("expected sequence_def_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_SequenceProgressNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrSequenceProgressNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "sequence_progress_not_found" {
		t.Errorf("expected sequence_progress_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_SequenceItemLocked(t *testing.T) {
	le := &LearningError{Err: domain.ErrSequenceItemLocked}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
}

func TestLearningError_ToAppError_SequenceNoItems(t *testing.T) {
	le := &LearningError{Err: domain.ErrSequenceNoItems}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
}

func TestLearningError_ToAppError_AssignmentNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrAssignmentNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "assignment_not_found" {
		t.Errorf("expected assignment_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_InvalidAssignmentStatusTransition(t *testing.T) {
	le := &LearningError{Err: domain.ErrInvalidAssignmentStatusTransition}
	ae := le.toAppError()
	if ae.StatusCode != 422 {
		t.Errorf("expected 422, got %d", ae.StatusCode)
	}
	if ae.Code != "invalid_assignment_status" {
		t.Errorf("expected invalid_assignment_status, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_VideoDefNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrVideoDefNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "video_def_not_found" {
		t.Errorf("expected video_def_not_found, got %s", ae.Code)
	}
}

func TestLearningError_ToAppError_VideoProgressNotFound(t *testing.T) {
	le := &LearningError{Err: domain.ErrVideoProgressNotFound}
	ae := le.toAppError()
	if ae.StatusCode != 404 {
		t.Errorf("expected 404, got %d", ae.StatusCode)
	}
	if ae.Code != "video_progress_not_found" {
		t.Errorf("expected video_progress_not_found, got %s", ae.Code)
	}
}

// ─── Slugify Tests ───────────────────────────────────────────────────────────

func TestSlugify_Basic(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Math", "math"},
		{"Language Arts", "language-arts"},
		{"  Science & Nature  ", "science--nature"},
		{"History 101", "history-101"},
	}
	for _, tt := range tests {
		got := domain.Slugify(tt.input)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// ─── Quiz Session Domain Tests ──────────────────────────────────────────────

func TestValidateQuizSessionTransition_Valid(t *testing.T) {
	tests := []struct {
		from, to string
	}{
		{domain.QuizStatusNotStarted, domain.QuizStatusInProgress},
		{domain.QuizStatusInProgress, domain.QuizStatusSubmitted},
		{domain.QuizStatusInProgress, domain.QuizStatusScored},
		{domain.QuizStatusSubmitted, domain.QuizStatusScored},
	}
	for _, tt := range tests {
		if err := domain.ValidateQuizSessionTransition(tt.from, tt.to); err != nil {
			t.Errorf("expected valid transition %s→%s, got %v", tt.from, tt.to, err)
		}
	}
}

func TestValidateQuizSessionTransition_Invalid(t *testing.T) {
	tests := []struct {
		from, to string
	}{
		{domain.QuizStatusSubmitted, domain.QuizStatusInProgress},
		{domain.QuizStatusScored, domain.QuizStatusInProgress},
		{domain.QuizStatusScored, domain.QuizStatusSubmitted},
		{domain.QuizStatusNotStarted, domain.QuizStatusScored},
	}
	for _, tt := range tests {
		if err := domain.ValidateQuizSessionTransition(tt.from, tt.to); err == nil {
			t.Errorf("expected invalid transition %s→%s, got nil", tt.from, tt.to)
		}
	}
}

func TestAutoScoreQuiz_AllAutoScorable(t *testing.T) {
	questions := []domain.QuizQuestionInfo{
		{Points: 10, AutoScorable: true},
		{Points: 20, AutoScorable: true},
	}
	maxScore, allAuto := domain.AutoScoreQuiz(questions)
	if maxScore != 30 {
		t.Errorf("expected maxScore 30, got %f", maxScore)
	}
	if !allAuto {
		t.Error("expected allAutoScorable true")
	}
}

func TestAutoScoreQuiz_MixedScorable(t *testing.T) {
	override := 15.0
	questions := []domain.QuizQuestionInfo{
		{Points: 10, AutoScorable: true},
		{Points: 20, PointsOverride: &override, AutoScorable: false},
	}
	maxScore, allAuto := domain.AutoScoreQuiz(questions)
	if maxScore != 25 { // 10 + 15 (override)
		t.Errorf("expected maxScore 25, got %f", maxScore)
	}
	if allAuto {
		t.Error("expected allAutoScorable false")
	}
}

func TestComputeParentScore(t *testing.T) {
	q1 := uuid.Must(uuid.NewV7())
	q2 := uuid.Must(uuid.NewV7())
	questions := []domain.QuizQuestionInfo{
		{QuestionID: q1, Points: 10, AutoScorable: true},
		{QuestionID: q2, Points: 20, AutoScorable: false},
	}
	parentScores := map[uuid.UUID]float64{q2: 15}
	totalScore, maxScore := domain.ComputeParentScore(questions, parentScores)
	if maxScore != 30 {
		t.Errorf("expected maxScore 30, got %f", maxScore)
	}
	if totalScore != 25 { // 10 (auto) + 15 (parent)
		t.Errorf("expected totalScore 25, got %f", totalScore)
	}
}

// ─── DefaultDateRange Domain Tests ──────────────────────────────────────────

func TestDefaultDateRange_NilDefaults(t *testing.T) {
	from, to := domain.DefaultDateRange(nil, nil)
	now := time.Now()
	// from should be ~1 month ago
	expectedFrom := now.AddDate(0, -1, 0).Truncate(24 * time.Hour)
	if from != expectedFrom {
		t.Errorf("expected from %v, got %v", expectedFrom, from)
	}
	// to should be end of today
	if to.Before(now.Truncate(24 * time.Hour)) {
		t.Error("expected to be at least start of today")
	}
}

func TestDefaultDateRange_CustomValues(t *testing.T) {
	customFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	customTo := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	from, to := domain.DefaultDateRange(&customFrom, &customTo)
	if from != customFrom {
		t.Errorf("expected from %v, got %v", customFrom, from)
	}
	if to != customTo {
		t.Errorf("expected to %v, got %v", customTo, to)
	}
}
