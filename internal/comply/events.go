package comply

import "github.com/google/uuid"

// Domain events published by comply::. [14-comply §15, CODING §8.4]
// All events implement shared.DomainEvent.
// Subscribers are registered in cmd/server/main.go via eventBus.Subscribe().

// PortfolioGenerated is published when a portfolio PDF has been generated and is ready for download.
// Consumed by notify:: (in-app notification + optional email).
type PortfolioGenerated struct {
	FamilyID       uuid.UUID `json:"family_id"`
	StudentID      uuid.UUID `json:"student_id"`
	PortfolioID    uuid.UUID `json:"portfolio_id"`
	PortfolioTitle string    `json:"portfolio_title"`
}

func (PortfolioGenerated) EventName() string { return "comply.portfolio_generated" }

// TranscriptGenerated is published when a transcript PDF has been generated (Phase 3).
// Consumed by notify:: (in-app notification + optional email).
type TranscriptGenerated struct {
	FamilyID     uuid.UUID `json:"family_id"`
	StudentID    uuid.UUID `json:"student_id"`
	TranscriptID uuid.UUID `json:"transcript_id"`
}

func (TranscriptGenerated) EventName() string { return "comply.transcript_generated" }

// AttendanceThresholdWarning is published when a student's attendance pace falls below requirements.
// Consumed by notify:: (in-app + email warning to parent).
type AttendanceThresholdWarning struct {
	FamilyID     uuid.UUID `json:"family_id"`
	StudentID    uuid.UUID `json:"student_id"`
	StudentName  string    `json:"student_name"`
	PaceStatus   string    `json:"pace_status"`
	ActualDays   int32     `json:"actual_days"`
	ExpectedDays int32     `json:"expected_days"`
	RequiredDays int16     `json:"required_days"`
}

func (AttendanceThresholdWarning) EventName() string { return "comply.attendance_threshold_warning" }
