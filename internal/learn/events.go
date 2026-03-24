package learn

import (
	"time"

	"github.com/google/uuid"
)

// Domain events published by the learning domain. [CODING §8.4, 06-learn §18.3]
// All events implement shared.DomainEvent.
// Subscribers are registered in cmd/server/main.go via eventBus.Subscribe().

// ActivityLogged is published when an activity is logged for a student.
// Subscribers:
//   - comply:: attendance tracking (needs SubjectTags, DurationMinutes)
//   - recs:: recommendation signal
//   - notify:: streak check
type ActivityLogged struct {
	FamilyID        uuid.UUID `json:"family_id"`
	StudentID       uuid.UUID `json:"student_id"`
	ActivityID      uuid.UUID `json:"activity_id"`
	SubjectTags     []string  `json:"subject_tags"`
	DurationMinutes *int16    `json:"duration_minutes"`
	ActivityDate    time.Time `json:"activity_date"`
}

func (ActivityLogged) EventName() string { return "learn.activity_logged" }

// MilestoneAchieved is published when a student reaches a learning milestone.
// Subscribers:
//   - notify:: sends notification
//   - social:: optional milestone post
type MilestoneAchieved struct {
	FamilyID      uuid.UUID `json:"family_id"`
	StudentID     uuid.UUID `json:"student_id"`
	StudentName   string    `json:"student_name"`
	MilestoneType string    `json:"milestone_type"` // "books_completed", "activity_streak", "subject_hours"
	Description   string    `json:"description"`
}

func (MilestoneAchieved) EventName() string { return "learn.milestone_achieved" }

// BookCompleted is published when a student finishes a book.
// Subscribers:
//   - notify:: reading milestone, streak check
type BookCompleted struct {
	FamilyID         uuid.UUID `json:"family_id"`
	StudentID        uuid.UUID `json:"student_id"`
	ReadingItemID    uuid.UUID `json:"reading_item_id"`
	ReadingItemTitle string    `json:"reading_item_title"`
}

func (BookCompleted) EventName() string { return "learn.book_completed" }

// DataExportReady is published when a data export is ready for download.
// Subscribers:
//   - notify:: sends download notification
type DataExportReady struct {
	FamilyID  uuid.UUID `json:"family_id"`
	ExportID  uuid.UUID `json:"export_id"`
	FileURL   string    `json:"file_url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (DataExportReady) EventName() string { return "learn.data_export_ready" }

// QuizCompleted is published when a quiz is fully scored.
// Subscribers:
//   - notify:: notify parent of quiz score
//   - recs:: recommendation signal
type QuizCompleted struct {
	FamilyID      uuid.UUID `json:"family_id"`
	StudentID     uuid.UUID `json:"student_id"`
	QuizDefID     uuid.UUID `json:"quiz_def_id"`
	QuizSessionID uuid.UUID `json:"quiz_session_id"`
	Score         float64   `json:"score"`
	MaxScore      float64   `json:"max_score"`
	Passed        bool      `json:"passed"`
}

func (QuizCompleted) EventName() string { return "learn.quiz_completed" }

// SequenceAdvanced is published when a student completes a sequence item.
// Subscribers:
//   - recs:: recommendation signal for sequence engagement
type SequenceAdvanced struct {
	FamilyID        uuid.UUID `json:"family_id"`
	StudentID       uuid.UUID `json:"student_id"`
	SequenceDefID   uuid.UUID `json:"sequence_def_id"`
	ItemIndex       int16     `json:"item_index"`
	ItemContentType string    `json:"item_content_type"`
	ItemContentID   uuid.UUID `json:"item_content_id"`
}

func (SequenceAdvanced) EventName() string { return "learn.sequence_advanced" }

// SequenceCompleted is published when a student completes all required items in a sequence.
// Subscribers:
//   - notify:: notify parent of sequence completion
//   - recs:: recommendation signal
type SequenceCompleted struct {
	FamilyID      uuid.UUID `json:"family_id"`
	StudentID     uuid.UUID `json:"student_id"`
	SequenceDefID uuid.UUID `json:"sequence_def_id"`
}

func (SequenceCompleted) EventName() string { return "learn.sequence_completed" }

// AssignmentCompleted is published when a student completes an assignment.
// Subscribers:
//   - notify:: notify parent of assignment completion
type AssignmentCompleted struct {
	FamilyID     uuid.UUID `json:"family_id"`
	StudentID    uuid.UUID `json:"student_id"`
	AssignmentID uuid.UUID `json:"assignment_id"`
	ContentType  string    `json:"content_type"`
	ContentID    uuid.UUID `json:"content_id"`
}

func (AssignmentCompleted) EventName() string { return "learn.assignment_completed" }
