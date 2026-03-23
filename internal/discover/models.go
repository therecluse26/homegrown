package discover

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ─── Internal JSON Types (unexported — scoring engine only) ──────────────────

// quizQuestionInternal is the internal representation of a quiz question as stored
// in the disc_quiz_definitions.questions JSONB column.
// These types are NEVER serialized to API responses. [03-discover §3.1]
type quizQuestionInternal struct {
	ID       string               `json:"id"`
	Category string               `json:"category"`
	Text     string               `json:"text"`
	HelpText string               `json:"help_text,omitempty"`
	Answers  []quizAnswerInternal `json:"answers"`
}

// quizAnswerInternal is the internal representation of a quiz answer option.
// The Weights map is the scoring payload: keys are methodology slugs, values are
// contribution weights in [0.0, 1.0]. [03-discover §3.1]
type quizAnswerInternal struct {
	ID      string             `json:"id"`
	Text    string             `json:"text"`
	Weights map[string]float64 `json:"weights"`
}

// quizExplanationsInternal is the internal shape of disc_quiz_definitions.explanations.
// Keys are methodology slugs; values contain text shown when score >= 0.5 (match)
// or < 0.5 (mismatch).
type quizExplanationsInternal map[string]struct {
	MatchText    string `json:"match_text"`
	MismatchText string `json:"mismatch_text"`
}

// ─── GORM Models ──────────────────────────────────────────────────────────────

// QuizDefinition is the GORM model for disc_quiz_definitions. [03-discover §6.1]
type QuizDefinition struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Version      int             `gorm:"not null;default:1"`
	Title        string          `gorm:"not null"`
	Description  string          `gorm:"not null;default:''"`
	Status       string          `gorm:"not null;default:'draft'"`
	Questions    json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"    swaggertype:"array,object"`
	Explanations json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"   swaggertype:"object"`
	CreatedAt    time.Time       `gorm:"not null;default:now()"`
	UpdatedAt    time.Time       `gorm:"not null;default:now()"`
}

func (QuizDefinition) TableName() string { return "disc_quiz_definitions" }

func (m *QuizDefinition) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// QuizResult is the GORM model for disc_quiz_results. [03-discover §6.2]
type QuizResult struct {
	ID                uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	QuizDefinitionID  uuid.UUID       `gorm:"type:uuid;not null"`
	ShareID           string          `gorm:"not null;uniqueIndex"`
	SessionToken      *string         `gorm:""`
	Answers           json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" swaggertype:"object"`
	Scores            json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" swaggertype:"object"`
	Recommendations   json.RawMessage `gorm:"type:jsonb;not null;default:'[]'" swaggertype:"array,object"`
	FamilyID          *uuid.UUID      `gorm:"type:uuid"`
	QuizVersion       int16           `gorm:"not null;default:0"`
	CreatedAt         time.Time       `gorm:"not null;default:now()"`
}

func (QuizResult) TableName() string { return "disc_quiz_results" }

func (m *QuizResult) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// StateGuide is the GORM model for disc_state_guides. [03-discover §6.3]
type StateGuide struct {
	ID             uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	StateCode      string          `gorm:"type:char(2);not null;uniqueIndex"`
	StateName      string          `gorm:"not null"`
	Status         string          `gorm:"not null;default:'draft'"`
	Requirements   json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" swaggertype:"object"`
	LastReviewedAt *time.Time      `gorm:""`
	GuideContent   string          `gorm:"not null;default:''"`
	NextReviewDue  *time.Time      `gorm:""`
	ReviewedBy     *string         `gorm:""`
	LegalDisclaimer string         `gorm:"not null"`
	CreatedAt      time.Time       `gorm:"not null;default:now()"`
	UpdatedAt      time.Time       `gorm:"not null;default:now()"`
}

func (StateGuide) TableName() string { return "disc_state_guides" }

func (m *StateGuide) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// StateGuideSummary is the result struct for list queries (select subset of columns).
type StateGuideSummary struct {
	StateCode      string
	StateName      string
	Status         string
	LastReviewedAt *time.Time
}

// ContentPage is the GORM model for disc_content_pages. [03-discover §6.4]
// Phase 2 — defined here for completeness; repository deferred.
type ContentPage struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	Slug           string     `gorm:"not null;uniqueIndex"`
	Category       string     `gorm:"not null"`
	Title          string     `gorm:"not null"`
	Content        string     `gorm:"not null;default:''"`
	MetaTitle      *string    `gorm:""`
	MetaDescription *string   `gorm:""`
	DisplayOrder   int16      `gorm:"not null;default:0"`
	Status         string     `gorm:"not null;default:'draft'"`
	CreatedAt      time.Time  `gorm:"not null;default:now()"`
	UpdatedAt      time.Time  `gorm:"not null;default:now()"`
}

func (ContentPage) TableName() string { return "disc_content_pages" }

func (m *ContentPage) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ContentPageSummary is the result struct for list queries.
type ContentPageSummary struct {
	Slug     string
	Category string
	Title    string
	Status   string
}

// ─── Repository Input Types ───────────────────────────────────────────────────

// CreateQuizResult carries all data needed to persist a new quiz result.
type CreateQuizResult struct {
	QuizDefinitionID uuid.UUID
	ShareID          string
	SessionToken     *string
	Answers          json.RawMessage
	Scores           json.RawMessage
	Recommendations  json.RawMessage
	QuizVersion      int16
}

// ─── API Request Types ────────────────────────────────────────────────────────

// SubmitQuizCommand is the request body for POST /v1/discovery/quiz/results.
type SubmitQuizCommand struct {
	// Answers maps question_id → answer_id.
	// Missing questions contribute zero weight (partial submissions are valid).
	Answers      map[string]string `json:"answers"       validate:"required"`
	SessionToken *string           `json:"session_token" validate:"omitempty,max=128"`
}

// ─── API Response Types ───────────────────────────────────────────────────────

// QuizResponse is the response for GET /v1/discovery/quiz. [03-discover §8.1]
// answer weights are stripped — clients MUST NOT receive scoring data.
type QuizResponse struct {
	QuizID      uuid.UUID              `json:"quiz_id"`
	Version     int                    `json:"version"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Questions   []QuizQuestionResponse `json:"questions"`
}

// QuizQuestionResponse is a single question in the quiz API response.
type QuizQuestionResponse struct {
	ID       string               `json:"id"`
	Category string               `json:"category"`
	Text     string               `json:"text"`
	HelpText string               `json:"help_text,omitempty"`
	Answers  []QuizAnswerResponse `json:"answers"`
}

// QuizAnswerResponse is a single answer option in the quiz API response.
// The Weights field from the internal representation is intentionally absent. [03-discover §3.1]
type QuizAnswerResponse struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// QuizResultResponse is the response for POST /v1/discovery/quiz/results
// and GET /v1/discovery/quiz/results/:share_id. [03-discover §8.2]
type QuizResultResponse struct {
	ShareID         string                      `json:"share_id"`
	QuizVersion     int16                       `json:"quiz_version"`
	CreatedAt       time.Time                   `json:"created_at"`
	IsClaimed       bool                        `json:"is_claimed"`
	Recommendations []MethodologyRecommendation `json:"recommendations"`
}

// MethodologyRecommendation is one ranked methodology result in a quiz response.
// ScorePercentage is an integer 0-100 (never a float). [03-discover §15.8]
type MethodologyRecommendation struct {
	Rank            uint8  `json:"rank"`
	MethodologySlug string `json:"methodology_slug"`
	MethodologyName string `json:"methodology_name"`
	ScorePercentage uint8  `json:"score_percentage"`
	Explanation     string `json:"explanation"`
}

// StateGuideSummaryResponse is one entry in the GET /v1/discovery/state-guides list.
// IsAvailable indicates whether the guide has published content. [03-discover §8.3]
type StateGuideSummaryResponse struct {
	StateCode      string     `json:"state_code"`
	StateName      string     `json:"state_name"`
	IsAvailable    bool       `json:"is_available"`
	LastReviewedAt *time.Time `json:"last_reviewed_at,omitempty"`
}

// StateGuideResponse is the response for GET /v1/discovery/state-guides/:state_code.
// Only returned when status == 'published'. [03-discover §8.3]
type StateGuideResponse struct {
	StateCode      string                 `json:"state_code"`
	StateName      string                 `json:"state_name"`
	Requirements   StateGuideRequirements `json:"requirements"`
	GuideContent   string                 `json:"guide_content"`
	LegalDisclaimer string                `json:"legal_disclaimer"`
	LastReviewedAt *time.Time             `json:"last_reviewed_at,omitempty"`
}

// StateGuideRequirements is the structured legal requirements for a state. [03-discover §6.3]
// Shared with comply:: when that domain reads disc_state_guides via GetStateRequirements().
// All boolean flags enable comply:: to act programmatically on requirements.
type StateGuideRequirements struct {
	NotificationRequired    bool     `json:"notification_required"`
	NotificationDetails     *string  `json:"notification_details,omitempty"`
	RequiredSubjects        []string `json:"required_subjects"`
	AssessmentRequired      bool     `json:"assessment_required"`
	AssessmentDetails       *string  `json:"assessment_details,omitempty"`
	RecordKeepingRequired   bool     `json:"record_keeping_required"`
	RecordKeepingDetails    *string  `json:"record_keeping_details,omitempty"`
	AttendanceRequired      bool     `json:"attendance_required"`
	AttendanceDays          *uint16  `json:"attendance_days,omitempty"`
	AttendanceDetails       *string  `json:"attendance_details,omitempty"`
	UmbrellaSchoolAvailable bool     `json:"umbrella_school_available"`
	UmbrellaSchoolDetails   *string  `json:"umbrella_school_details,omitempty"`
	RegulationLevel         string   `json:"regulation_level"` // "low"|"moderate"|"high"
}

// ContentPageResponse is the response for GET /v1/discovery/content/:slug (Phase 2).
type ContentPageResponse struct {
	Slug           string  `json:"slug"`
	Category       string  `json:"category"`
	Title          string  `json:"title"`
	Content        string  `json:"content"`
	MetaTitle      *string `json:"meta_title,omitempty"`
	MetaDescription *string `json:"meta_description,omitempty"`
}

// ContentPageSummaryResponse is one entry in the content page list (Phase 2).
type ContentPageSummaryResponse struct {
	Slug     string `json:"slug"`
	Category string `json:"category"`
	Title    string `json:"title"`
}

// ─── Sentinel Errors ──────────────────────────────────────────────────────────

var (
	ErrNoActiveQuiz             = errors.New("no active quiz")
	ErrQuizResultNotFound       = errors.New("quiz result not found")
	ErrInvalidQuestionID        = errors.New("invalid question ID")
	ErrInvalidAnswerID          = errors.New("invalid answer ID")
	ErrStateGuideNotFound       = errors.New("state guide not found")
	ErrStateGuideNotPublished   = errors.New("state guide not published")
	ErrContentPageNotFound      = errors.New("content page not found")
	ErrQuizResultAlreadyClaimed = errors.New("quiz result already claimed by a different family")
	ErrInvalidStateCode         = errors.New("invalid state code")
)

// ─── Domain Error Wrapper ─────────────────────────────────────────────────────

// DiscoverError wraps a sentinel error with optional context fields.
// All repository and service errors use this type. [CODING §2.2]
type DiscoverError struct {
	Err        error
	QuestionID string
	AnswerID   string
	StateCode  string
	Slug       string
}

func (e *DiscoverError) Error() string {
	switch {
	case e.QuestionID != "":
		return fmt.Sprintf("%s: question_id=%s", e.Err.Error(), e.QuestionID)
	case e.AnswerID != "":
		return fmt.Sprintf("%s: answer_id=%s", e.Err.Error(), e.AnswerID)
	case e.StateCode != "":
		return fmt.Sprintf("%s: state_code=%s", e.Err.Error(), e.StateCode)
	case e.Slug != "":
		return fmt.Sprintf("%s: slug=%s", e.Err.Error(), e.Slug)
	default:
		return e.Err.Error()
	}
}

func (e *DiscoverError) Unwrap() error { return e.Err }

// toAppError maps a DiscoverError sentinel to an *shared.AppError with the correct
// HTTP status code. Called by mapDiscoverError in handler.go. [03-discover §12.1]
func (e *DiscoverError) toAppError() *shared.AppError {
	switch {
	case errors.Is(e.Err, ErrNoActiveQuiz):
		return &shared.AppError{Code: "no_active_quiz", Message: "No active quiz is available", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrQuizResultNotFound):
		return &shared.AppError{Code: "quiz_result_not_found", Message: "Quiz result not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrInvalidQuestionID):
		return &shared.AppError{Code: "invalid_question_id", Message: "One or more question IDs are invalid", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrInvalidAnswerID):
		return &shared.AppError{Code: "invalid_answer_id", Message: "One or more answer IDs are invalid", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, ErrStateGuideNotFound):
		return &shared.AppError{Code: "state_guide_not_found", Message: "State guide not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrStateGuideNotPublished):
		// Draft guides return 404, not 403 — existence is not secret. [03-discover §3.2, §15.16]
		return &shared.AppError{Code: "state_guide_not_found", Message: "State guide not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrContentPageNotFound):
		return &shared.AppError{Code: "content_page_not_found", Message: "Content page not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, ErrQuizResultAlreadyClaimed):
		return &shared.AppError{Code: "quiz_result_claimed", Message: "Quiz result is already claimed by another family", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, ErrInvalidStateCode):
		return &shared.AppError{Code: "invalid_state_code", Message: "Invalid state code", StatusCode: http.StatusUnprocessableEntity}
	default:
		return shared.ErrInternal(e)
	}
}
